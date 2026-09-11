package repository

import (
	"context"
	"database/sql"
	"fmt"

	"example/hello/internal/dto"
	"example/hello/internal/models"
)

type ClassRepository struct {
	db *sql.DB
}

func NewClassRepository(db *sql.DB) *ClassRepository {
	return &ClassRepository{db: db}
}

// classSelect is shared by the list and the single-row read so the two can
// never drift into reporting different things about the same class.
const classSelect = `
	SELECT c.id, c.course_id, co.title, c.name,
	       c.teacher_id, COALESCE(t.full_name, ''),
	       COALESCE(c.schedule, ''), c.status,
	       (SELECT COUNT(*) FROM class_students cs
	         WHERE cs.class_id = c.id AND cs.status = 'ACTIVE'),
	       c.created_at, c.updated_at
	FROM classes c
	JOIN courses co ON co.id = c.course_id
	LEFT JOIN users t ON t.id = c.teacher_id
`

func scanClass(row interface{ Scan(...any) error }) (*dto.ClassResponse, error) {
	var out dto.ClassResponse
	var teacherID sql.NullInt64
	err := row.Scan(
		&out.ID, &out.CourseID, &out.CourseTitle, &out.Name,
		&teacherID, &out.TeacherName,
		&out.Schedule, &out.Status,
		&out.StudentCount,
		&out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if teacherID.Valid {
		out.TeacherID = &teacherID.Int64
	}
	return &out, nil
}

func (r *ClassRepository) Create(ctx context.Context, class *models.Class) error {
	query := `
		INSERT INTO classes (course_id, name, teacher_id, schedule, created_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		class.CourseID, class.Name, class.TeacherID, class.Schedule, class.CreatedBy,
	).Scan(&class.ID, &class.Status, &class.CreatedAt, &class.UpdatedAt)
}

func (r *ClassRepository) GetByID(ctx context.Context, id int64) (*dto.ClassResponse, error) {
	return scanClass(r.db.QueryRowContext(ctx, classSelect+" WHERE c.id = $1", id))
}

// List returns every class, or only those a given teacher runs. FR-CLS-04:
// admins see all of them, a teacher sees the ones they are responsible for.
func (r *ClassRepository) List(ctx context.Context, teacherID *int64) ([]dto.ClassResponse, error) {
	query := classSelect
	args := []any{}
	if teacherID != nil {
		query += " WHERE c.teacher_id = $1"
		args = append(args, *teacherID)
	}
	query += " ORDER BY c.status = 'ACTIVE' DESC, c.created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	classes := []dto.ClassResponse{}
	for rows.Next() {
		item, err := scanClass(rows)
		if err != nil {
			return nil, err
		}
		classes = append(classes, *item)
	}
	return classes, rows.Err()
}

// Update writes only the fields the caller supplied. Passing a nil TeacherID
// with clearTeacher set detaches the class from its teacher.
func (r *ClassRepository) Update(ctx context.Context, id int64, name *string,
	teacherID *int64, clearTeacher bool, schedule *string, status *string) error {

	query := `
		UPDATE classes SET
			name       = COALESCE($2, name),
			teacher_id = CASE WHEN $3::boolean THEN NULL ELSE COALESCE($4, teacher_id) END,
			schedule   = COALESCE($5, schedule),
			status     = COALESCE($6, status)
		WHERE id = $1
	`
	result, err := r.db.ExecContext(ctx, query, id, name, clearTeacher, teacherID, schedule, status)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *ClassRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM classes WHERE id = $1", id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// TeacherOf reports who runs a class, so the service can decide whether the
// caller is allowed to look at it.
func (r *ClassRepository) TeacherOf(ctx context.Context, id int64) (sql.NullInt64, int64, error) {
	var teacherID sql.NullInt64
	var courseID int64
	err := r.db.QueryRowContext(ctx,
		"SELECT teacher_id, course_id FROM classes WHERE id = $1", id,
	).Scan(&teacherID, &courseID)
	return teacherID, courseID, err
}

func (r *ClassRepository) ListStudents(ctx context.Context, classID int64) ([]dto.ClassStudentResponse, error) {
	query := `
		SELECT cs.student_id, COALESCE(u.full_name, ''), COALESCE(u.email, ''),
		       cs.joined_at, cs.status, cs.left_at
		FROM class_students cs
		JOIN users u ON u.id = cs.student_id
		WHERE cs.class_id = $1
		ORDER BY cs.status = 'ACTIVE' DESC, u.full_name
	`
	rows, err := r.db.QueryContext(ctx, query, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	students := []dto.ClassStudentResponse{}
	for rows.Next() {
		var s dto.ClassStudentResponse
		var leftAt sql.NullTime
		if err := rows.Scan(&s.StudentID, &s.FullName, &s.Email,
			&s.JoinedAt, &s.Status, &leftAt); err != nil {
			return nil, err
		}
		if leftAt.Valid {
			s.LeftAt = &leftAt.Time
		}
		students = append(students, s)
	}
	return students, rows.Err()
}

// AddStudent puts a learner on the roster and gives them the course in one
// transaction.
//
// enrollments is the projection every access check, progress figure and
// analytics query already reads. Writing the roster without it would put a
// student in a class that shows them nothing; writing it outside a transaction
// would let a crash leave exactly that state on disk.
func (r *ClassRepository) AddStudent(ctx context.Context, classID, studentID, addedBy int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var courseID int64
	if err := tx.QueryRowContext(ctx,
		"SELECT course_id FROM classes WHERE id = $1", classID,
	).Scan(&courseID); err != nil {
		return err
	}

	// DO NOTHING would silently ignore someone being put back after dropping
	// out. joined_at moves with them: the roster shows current membership, so
	// the date on it should be when that membership began.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO class_students (class_id, student_id, added_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (class_id, student_id) DO UPDATE
		SET status = 'ACTIVE', left_at = NULL, joined_at = NOW()
	`, classID, studentID, addedBy); err != nil {
		return err
	}

	if err := syncCourseAccess(ctx, tx, studentID, courseID); err != nil {
		return err
	}

	return tx.Commit()
}

// RemoveStudent takes a learner off the roster, and takes the course away with
// it unless another class of the same course still holds them.
func (r *ClassRepository) RemoveStudent(ctx context.Context, classID, studentID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var courseID int64
	if err := tx.QueryRowContext(ctx,
		"SELECT course_id FROM classes WHERE id = $1", classID,
	).Scan(&courseID); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx,
		"DELETE FROM class_students WHERE class_id = $1 AND student_id = $2", classID, studentID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	if err := syncCourseAccess(ctx, tx, studentID, courseID); err != nil {
		return err
	}

	return tx.Commit()
}

// SetStudentStatus marks a roster entry active or dropped.
//
// Dropping is not removing. The row stays, so the centre can still answer which
// class a learner attended and what they scored there; only course access
// follows the active roster. Removing a row means the opposite - that the
// person was never in this class, and an admin is correcting a mis-assignment.
func (r *ClassRepository) SetStudentStatus(
	ctx context.Context, classID, studentID int64, status string,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var courseID int64
	if err := tx.QueryRowContext(ctx,
		"SELECT course_id FROM classes WHERE id = $1", classID,
	).Scan(&courseID); err != nil {
		return err
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE class_students
		SET status  = $3,
		    left_at = CASE WHEN $3 = 'DROPPED' THEN NOW() ELSE NULL END
		WHERE class_id = $1 AND student_id = $2
	`, classID, studentID, status)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}

	if err := syncCourseAccess(ctx, tx, studentID, courseID); err != nil {
		return err
	}

	return tx.Commit()
}

// syncCourseAccess makes the enrolment row agree with the roster: a learner
// holds a course exactly while some class of that course still lists them as
// active.
//
// Add, remove and drop all finish here, so the three can never drift into
// disagreeing about who can open the material. Counting rows rather than
// active rows is the subtle way to get this wrong - a learner who dropped
// every class of a course would keep access because the rows are still there.
func syncCourseAccess(ctx context.Context, tx *sql.Tx, studentID, courseID int64) error {
	var active int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM class_students cs
		JOIN classes c ON c.id = cs.class_id
		WHERE cs.student_id = $1 AND c.course_id = $2 AND cs.status = 'ACTIVE'
	`, studentID, courseID).Scan(&active); err != nil {
		return err
	}

	if active > 0 {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO enrollments (course_id, student_id, status)
			VALUES ($1, $2, 'ACCEPTED')
			ON CONFLICT (course_id, student_id) DO NOTHING
		`, courseID, studentID)
		return err
	}

	_, err := tx.ExecContext(ctx,
		"DELETE FROM enrollments WHERE course_id = $1 AND student_id = $2",
		courseID, studentID)
	return err
}

// StudentExists guards against putting an id on a roster that belongs to
// nobody, which the foreign key would catch but only with an opaque message -
// and against someone who has left the centre, which the foreign key would not
// catch at all, since soft delete leaves their row in place.
func (r *ClassRepository) StudentExists(ctx context.Context, studentID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL)",
		studentID).Scan(&exists)
	return exists, err
}

// CourseExists is the same guard for the course a class is opened on.
func (r *ClassRepository) CourseExists(ctx context.Context, courseID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM courses WHERE id = $1)", courseID).Scan(&exists)
	return exists, err
}

// NameTaken reports whether this course already has a class by that name, so
// the service can say so plainly instead of surfacing a constraint violation.
func (r *ClassRepository) NameTaken(ctx context.Context, courseID int64, name string, excludeID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM classes
			WHERE course_id = $1 AND lower(name) = lower($2) AND id <> $3
		)
	`, courseID, name, excludeID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking class name: %w", err)
	}
	return exists, nil
}
