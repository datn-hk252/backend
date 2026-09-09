package dto

import "time"

// ── Requests ────────────────────────────────────────────────────────────────

// CreateClassRequest opens a new cohort on an existing course. FR-CLS-01.
type CreateClassRequest struct {
	CourseID  int64  `json:"course_id" binding:"required"`
	Name      string `json:"name" binding:"required,max=255"`
	TeacherID *int64 `json:"teacher_id" binding:"omitempty"`
	Schedule  string `json:"schedule" binding:"omitempty,max=255"`
}

// UpdateClassRequest carries only what the admin actually changed; every field
// is a pointer so "clear the teacher" is distinguishable from "leave it alone".
// Reassigning the teacher is FR-CLS-03.
type UpdateClassRequest struct {
	Name      *string `json:"name" binding:"omitempty,max=255"`
	TeacherID *int64  `json:"teacher_id" binding:"omitempty"`
	Schedule  *string `json:"schedule" binding:"omitempty,max=255"`
	Status    *string `json:"status" binding:"omitempty,oneof=ACTIVE FINISHED CANCELLED"`
}

// AddClassStudentRequest puts one learner on the roster. FR-CLS-02.
type AddClassStudentRequest struct {
	StudentID int64 `json:"student_id" binding:"required"`
}

// AddClassStudentsRequest fills a roster in one go, which is how a new class
// actually starts: the admin has a list, not one name at a time.
type AddClassStudentsRequest struct {
	StudentIDs []int64 `json:"student_ids" binding:"required,min=1,max=500"`
}

// AddClassStudentsResult reports what happened per learner, so a single bad id
// in a long list does not discard the rest.
type AddClassStudentsResult struct {
	Added   int      `json:"added"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors,omitempty"`
}

// ── Responses ───────────────────────────────────────────────────────────────

// ClassResponse is a class as the class list shows it. Course and teacher names
// are joined in so the list does not need a second round of lookups.
type ClassResponse struct {
	ID           int64     `json:"id"`
	CourseID     int64     `json:"course_id"`
	CourseTitle  string    `json:"course_title"`
	Name         string    `json:"name"`
	TeacherID    *int64    `json:"teacher_id,omitempty"`
	TeacherName  string    `json:"teacher_name,omitempty"`
	Schedule     string    `json:"schedule,omitempty"`
	Status       string    `json:"status"`
	StudentCount int       `json:"student_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ClassStudentResponse is one row of a roster. FR-CLS-04.
type ClassStudentResponse struct {
	StudentID int64     `json:"student_id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	JoinedAt  time.Time `json:"joined_at"`
}

// ClassDetailResponse is a class together with its roster.
type ClassDetailResponse struct {
	ClassResponse
	Students []ClassStudentResponse `json:"students"`
}
