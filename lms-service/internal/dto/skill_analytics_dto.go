package dto

import "time"

// Types for breaking a quiz result down by the skills it exercised.
//
// Results used to be stored only as one total score per attempt, which cannot
// answer the two questions the language centre actually asks: where is this
// student weak, and on what criteria do we group students. The types below feed
// three teacher-facing views: one student's profile on a quiz, the whole class
// on a quiz, and one student compared across quizzes over time.

// SkillScore is the aggregate result for a single skill.
type SkillScore struct {
	SkillID   int64  `json:"skill_id"`
	SkillName string `json:"skill_name"`
	// Name of the parent skill; empty when this is one of the four roots.
	ParentSkill string `json:"parent_skill,omitempty"`

	TotalAnswers   int     `json:"total_answers"`
	CorrectAnswers int     `json:"correct_answers"`
	PointsEarned   float64 `json:"points_earned"`
	PointsPossible float64 `json:"points_possible"`
	// Points earned over points available, 0 to 100.
	Percentage float64 `json:"percentage"`
}

// StudentQuizSkillBreakdown is one student's skill profile on one quiz.
type StudentQuizSkillBreakdown struct {
	StudentID   int64      `json:"student_id"`
	StudentName string     `json:"student_name"`
	QuizID      int64      `json:"quiz_id"`
	QuizTitle   string     `json:"quiz_title"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`

	// Questions in the quiz that carry no skill tag. Reported so the teacher
	// can see how much of the quiz the breakdown actually covers, rather than
	// assuming it covers all of it.
	UntaggedQuestions int `json:"untagged_questions"`

	Skills []SkillScore `json:"skills"`
}

// ClassQuizSkillBreakdown is the whole class's skill profile on one quiz, so a
// teacher can see which skill the class as a whole is weak at.
type ClassQuizSkillBreakdown struct {
	QuizID    int64  `json:"quiz_id"`
	QuizTitle string `json:"quiz_title"`
	// Number of students whose submitted attempt is counted here.
	StudentCount      int          `json:"student_count"`
	UntaggedQuestions int          `json:"untagged_questions"`
	Skills            []SkillScore `json:"skills"`
}

// SkillTrendPoint is one skill's result on one specific quiz.
type SkillTrendPoint struct {
	QuizID         int64      `json:"quiz_id"`
	QuizTitle      string     `json:"quiz_title"`
	SubmittedAt    *time.Time `json:"submitted_at,omitempty"`
	TotalAnswers   int        `json:"total_answers"`
	CorrectAnswers int        `json:"correct_answers"`
	Percentage     float64    `json:"percentage"`
}

// SkillTrendSeries groups the points of one skill in chronological order.
type SkillTrendSeries struct {
	SkillID     int64             `json:"skill_id"`
	SkillName   string            `json:"skill_name"`
	ParentSkill string            `json:"parent_skill,omitempty"`
	Points      []SkillTrendPoint `json:"points"`
}

// StudentSkillTrend compares one student's skill profile across every quiz in
// a single course.
type StudentSkillTrend struct {
	StudentID int64              `json:"student_id"`
	CourseID  int64              `json:"course_id"`
	Series    []SkillTrendSeries `json:"series"`
}

// SkillNode is one entry of the skill taxonomy. The parent's name is flattened
// onto the row so a client can group the list without a second lookup.
type SkillNode struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ParentID    *int64 `json:"parent_id,omitempty"`
	ParentName  string `json:"parent_name,omitempty"`
}
