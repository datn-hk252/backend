package repository

import (
	"context"
	"database/sql"
	"time"

	"example/hello/internal/dto"
)

// SkillAnalyticsRepository reads quiz results broken down by skill.
//
// It draws on three tables that already exist: quiz_student_answers holds the
// per-question outcome, question_skills links a question to a skill, and skills
// is the taxonomy. No new table is needed, and learning_events is deliberately
// not used — that table models self-paced study, whereas here the submitted
// attempt is the source of truth.
type SkillAnalyticsRepository struct {
	db *sql.DB
}

func NewSkillAnalyticsRepository(db *sql.DB) *SkillAnalyticsRepository {
	return &SkillAnalyticsRepository{db: db}
}

// Only submitted attempts count. An attempt still in progress says nothing
// about the student's ability.
const submittedStatuses = `('SUBMITTED','GRADED')`

// scanSkillScores reads grouped rows into a slice of SkillScore.
func scanSkillScores(rows *sql.Rows) ([]dto.SkillScore, error) {
	defer rows.Close()

	var out []dto.SkillScore
	for rows.Next() {
		var s dto.SkillScore
		var parent sql.NullString
		var earned, possible sql.NullFloat64
		if err := rows.Scan(&s.SkillID, &s.SkillName, &parent,
			&s.TotalAnswers, &s.CorrectAnswers, &earned, &possible); err != nil {
			return nil, err
		}
		s.ParentSkill = parent.String
		s.PointsEarned = earned.Float64
		s.PointsPossible = possible.Float64
		if s.PointsPossible > 0 {
			s.Percentage = s.PointsEarned / s.PointsPossible * 100
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetStudentQuizSkillScores returns one student's per-skill result on a quiz,
// using their most recent submitted attempt.
func (r *SkillAnalyticsRepository) GetStudentQuizSkillScores(
	ctx context.Context, quizID, studentID int64,
) ([]dto.SkillScore, error) {

	rows, err := r.db.QueryContext(ctx, `
		WITH latest AS (
			SELECT id FROM quiz_attempts
			WHERE quiz_id = $1 AND student_id = $2
			  AND status IN `+submittedStatuses+`
			ORDER BY attempt_number DESC
			LIMIT 1
		)
		SELECT s.id, s.name, p.name,
		       COUNT(*)::int,
		       COUNT(*) FILTER (WHERE a.is_correct)::int,
		       COALESCE(SUM(a.points_earned), 0),
		       COALESCE(SUM(qq.points), 0)
		FROM quiz_student_answers a
		JOIN latest              l  ON l.id = a.attempt_id
		JOIN quiz_questions      qq ON qq.id = a.question_id
		JOIN question_skills     qs ON qs.question_id = qq.id
		JOIN skills              s  ON s.id = qs.skill_id
		LEFT JOIN skills         p  ON p.id = s.parent_skill_id
		GROUP BY s.id, s.name, p.name
		ORDER BY COALESCE(p.name, s.name), s.name
	`, quizID, studentID)
	if err != nil {
		return nil, err
	}
	return scanSkillScores(rows)
}

// GetClassQuizSkillScores returns the class-wide per-skill result on a quiz.
// Each student contributes only their most recent submitted attempt.
func (r *SkillAnalyticsRepository) GetClassQuizSkillScores(
	ctx context.Context, quizID int64,
) ([]dto.SkillScore, int, error) {

	rows, err := r.db.QueryContext(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (student_id) id, student_id
			FROM quiz_attempts
			WHERE quiz_id = $1 AND status IN `+submittedStatuses+`
			ORDER BY student_id, attempt_number DESC
		)
		SELECT s.id, s.name, p.name,
		       COUNT(*)::int,
		       COUNT(*) FILTER (WHERE a.is_correct)::int,
		       COALESCE(SUM(a.points_earned), 0),
		       COALESCE(SUM(qq.points), 0)
		FROM quiz_student_answers a
		JOIN latest              l  ON l.id = a.attempt_id
		JOIN quiz_questions      qq ON qq.id = a.question_id
		JOIN question_skills     qs ON qs.question_id = qq.id
		JOIN skills              s  ON s.id = qs.skill_id
		LEFT JOIN skills         p  ON p.id = s.parent_skill_id
		GROUP BY s.id, s.name, p.name
		ORDER BY COALESCE(p.name, s.name), s.name
	`, quizID)
	if err != nil {
		return nil, 0, err
	}
	scores, err := scanSkillScores(rows)
	if err != nil {
		return nil, 0, err
	}

	var studentCount int
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT student_id)::int
		FROM quiz_attempts
		WHERE quiz_id = $1 AND status IN `+submittedStatuses+`
	`, quizID).Scan(&studentCount)
	if err != nil {
		return nil, 0, err
	}

	return scores, studentCount, nil
}

// CountUntaggedQuestions counts questions in the quiz that carry no skill tag.
// Surfacing this number tells the teacher how much of the quiz the breakdown
// leaves out.
func (r *SkillAnalyticsRepository) CountUntaggedQuestions(
	ctx context.Context, quizID int64,
) (int, error) {

	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)::int
		FROM quiz_questions qq
		WHERE qq.quiz_id = $1
		  AND NOT EXISTS (SELECT 1 FROM question_skills qs WHERE qs.question_id = qq.id)
	`, quizID).Scan(&n)
	return n, err
}

// GetStudentSkillTrend returns one student's per-skill results across every
// quiz in a course, ordered by submission time.
//
// A quiz reaches its course indirectly: quizzes -> section_content ->
// course_sections.
func (r *SkillAnalyticsRepository) GetStudentSkillTrend(
	ctx context.Context, studentID, courseID int64,
) ([]dto.SkillTrendSeries, error) {

	rows, err := r.db.QueryContext(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (att.quiz_id)
			       att.id, att.quiz_id, att.submitted_at
			FROM quiz_attempts att
			JOIN quizzes         qz ON qz.id = att.quiz_id
			JOIN section_content sc ON sc.id = qz.content_id
			JOIN course_sections cs ON cs.id = sc.section_id
			WHERE att.student_id = $1
			  AND cs.course_id = $2
			  AND att.status IN `+submittedStatuses+`
			ORDER BY att.quiz_id, att.attempt_number DESC
		)
		SELECT s.id, s.name, p.name,
		       l.quiz_id, qz.title, l.submitted_at,
		       COUNT(*)::int,
		       COUNT(*) FILTER (WHERE a.is_correct)::int,
		       COALESCE(SUM(a.points_earned), 0),
		       COALESCE(SUM(qq.points), 0)
		FROM quiz_student_answers a
		JOIN latest          l  ON l.id = a.attempt_id
		JOIN quizzes         qz ON qz.id = l.quiz_id
		JOIN quiz_questions  qq ON qq.id = a.question_id
		JOIN question_skills qs ON qs.question_id = qq.id
		JOIN skills          s  ON s.id = qs.skill_id
		LEFT JOIN skills     p  ON p.id = s.parent_skill_id
		GROUP BY s.id, s.name, p.name, l.quiz_id, qz.title, l.submitted_at
		ORDER BY COALESCE(p.name, s.name), s.name, l.submitted_at
	`, studentID, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Fold rows into one series per skill; the ORDER BY above already puts the
	// points of each skill in chronological order.
	var series []dto.SkillTrendSeries
	index := make(map[int64]int)

	for rows.Next() {
		var (
			skillID          int64
			skillName        string
			parent           sql.NullString
			quizID           int64
			quizTitle        string
			submittedAt      sql.NullTime
			total, correct   int
			earned, possible sql.NullFloat64
		)
		if err := rows.Scan(&skillID, &skillName, &parent,
			&quizID, &quizTitle, &submittedAt,
			&total, &correct, &earned, &possible); err != nil {
			return nil, err
		}

		point := dto.SkillTrendPoint{
			QuizID:         quizID,
			QuizTitle:      quizTitle,
			TotalAnswers:   total,
			CorrectAnswers: correct,
		}
		if submittedAt.Valid {
			t := submittedAt.Time
			point.SubmittedAt = &t
		}
		if possible.Float64 > 0 {
			point.Percentage = earned.Float64 / possible.Float64 * 100
		}

		if i, ok := index[skillID]; ok {
			series[i].Points = append(series[i].Points, point)
			continue
		}
		index[skillID] = len(series)
		series = append(series, dto.SkillTrendSeries{
			SkillID:     skillID,
			SkillName:   skillName,
			ParentSkill: parent.String,
			Points:      []dto.SkillTrendPoint{point},
		})
	}
	return series, rows.Err()
}

// GetQuizMeta returns the quiz title and, when studentID is given, that
// student's submission time for their most recent attempt.
func (r *SkillAnalyticsRepository) GetQuizMeta(
	ctx context.Context, quizID int64, studentID *int64,
) (title string, submittedAt *time.Time, err error) {

	err = r.db.QueryRowContext(ctx,
		`SELECT title FROM quizzes WHERE id = $1`, quizID).Scan(&title)
	if err != nil {
		return "", nil, err
	}
	if studentID == nil {
		return title, nil, nil
	}

	var at sql.NullTime
	err = r.db.QueryRowContext(ctx, `
		SELECT submitted_at FROM quiz_attempts
		WHERE quiz_id = $1 AND student_id = $2 AND status IN `+submittedStatuses+`
		ORDER BY attempt_number DESC LIMIT 1
	`, quizID, *studentID).Scan(&at)
	if err == sql.ErrNoRows {
		return title, nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	if at.Valid {
		t := at.Time
		submittedAt = &t
	}
	return title, submittedAt, nil
}
