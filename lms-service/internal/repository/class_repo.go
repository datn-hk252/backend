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
	       (SELECT COUNT(*) FROM class_students cs WHERE cs.class_id = c.id),
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
		SELECT cs.student_id, COALESCE(u.full_name, ''), COALESCE(u.email, ''), cs.joined_at
		FROM class_students cs
		JOIN users u ON u.id = cs.student_id
		WHERE cs.class_id = $1
		ORDER BY u.full_name
	`
	rows, err := r.db.QueryContext(ctx, query, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	students := []dto.ClassStudentResponse{}
	for rows.Next() {
		var s dto.ClassStudentResponse
		if err := rows.Scan(&s.StudentID, &s.FullName, &s.Email, &s.JoinedAt); err != nil {
			return nil, err
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

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO class_students (class_id, student_id, added_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (class_id, student_id) DO NOTHING
	`, classID, studentID, addedBy); err != nil {
		return err
	}

	// The learner may already hold this course through another class of it.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO enrollments (course_id, student_id, status)
		VALUES ($1, $2, 'ACCEPTED')
		ON CONFLICT (course_id, student_id) DO NOTHING
	`, courseID, studentID); err != nil {
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

	// Revoking the course from someone still enrolled through a sibling class
	// would hide material they are supposed to be studying.
	var remaining int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM class_students cs
		JOIN classes c ON c.id = cs.class_id
		WHERE cs.student_id = $1 AND c.course_id = $2
	`, studentID, courseID).Scan(&remaining); err != nil {
		return err
	}

	if remaining == 0 {
		if _, err := tx.ExecContext(ctx,
			"DELETE FROM enrollments WHERE course_id = $1 AND student_id = $2",
			courseID, studentID); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// StudentExists guards against putting an id on a roster that belongs to
// nobody, which the foreign key would catch but only with an opaque message.
func (r *ClassRepository) StudentExists(ctx context.Context, studentID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		"SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)", studentID).Scan(&exists)
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
