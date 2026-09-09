package models

import (
	"database/sql"
	"time"
)

// Class statuses. A class runs, then ends; cancelled covers one that never ran.
const (
	ClassStatusActive    = "ACTIVE"
	ClassStatusFinished  = "FINISHED"
	ClassStatusCancelled = "CANCELLED"
)

// Class is one cohort running a course: same material, its own teacher,
// timetable and roster. See V020__classes.sql for why courses alone were not
// enough.
type Class struct {
	ID        int64          `json:"id" db:"id"`
	CourseID  int64          `json:"course_id" db:"course_id"`
	Name      string         `json:"name" db:"name"`
	TeacherID sql.NullInt64  `json:"-" db:"teacher_id"`
	Schedule  sql.NullString `json:"-" db:"schedule"`
	Status    string         `json:"status" db:"status"`
	CreatedBy int64          `json:"created_by" db:"created_by"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

// ClassStudent is one learner's place on a roster.
type ClassStudent struct {
	ID        int64     `json:"id" db:"id"`
	ClassID   int64     `json:"class_id" db:"class_id"`
	StudentID int64     `json:"student_id" db:"student_id"`
	AddedBy   int64     `json:"added_by" db:"added_by"`
	JoinedAt  time.Time `json:"joined_at" db:"joined_at"`
}
