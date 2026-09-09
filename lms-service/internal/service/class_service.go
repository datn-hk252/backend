package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"example/hello/internal/dto"
	"example/hello/internal/models"
	"example/hello/internal/repository"
)

// ClassService enforces FR-CLS-01..04.
//
// Every write is the admin's: a centre places its learners rather than letting
// them place themselves, and FR-CLS-01 through 03 name the admin as the actor.
// Teachers read, and only their own classes (FR-CLS-04).
type ClassService struct {
	classRepo *repository.ClassRepository
}

func NewClassService(classRepo *repository.ClassRepository) *ClassService {
	return &ClassService{classRepo: classRepo}
}

var errClassNotFound = errors.New("class not found")

func (s *ClassService) Create(ctx context.Context, actorID int64, req *dto.CreateClassRequest) (*dto.ClassResponse, error) {
	exists, err := s.classRepo.CourseExists(ctx, req.CourseID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("course not found")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("class name is required")
	}
	taken, err := s.classRepo.NameTaken(ctx, req.CourseID, name, 0)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, fmt.Errorf("a class named %q already exists on this course", name)
	}

	class := &models.Class{
		CourseID:  req.CourseID,
		Name:      name,
		TeacherID: nullInt64(req.TeacherID),
		Schedule:  nullString(strings.TrimSpace(req.Schedule)),
		CreatedBy: actorID,
	}
	if err := s.classRepo.Create(ctx, class); err != nil {
		return nil, err
	}
	return s.classRepo.GetByID(ctx, class.ID)
}

func (s *ClassService) Update(ctx context.Context, classID int64, req *dto.UpdateClassRequest) (*dto.ClassResponse, error) {
	_, courseID, err := s.classRepo.TeacherOf(ctx, classID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errClassNotFound
		}
		return nil, err
	}

	var name *string
	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return nil, errors.New("class name is required")
		}
		taken, err := s.classRepo.NameTaken(ctx, courseID, trimmed, classID)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, fmt.Errorf("a class named %q already exists on this course", trimmed)
		}
		name = &trimmed
	}

	// FR-CLS-03. Sending teacher_id: 0 detaches the class rather than pointing
	// it at a user id nobody has.
	clearTeacher := req.TeacherID != nil && *req.TeacherID == 0
	teacherID := req.TeacherID
	if clearTeacher {
		teacherID = nil
	}

	if err := s.classRepo.Update(ctx, classID, name, teacherID, clearTeacher, req.Schedule, req.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errClassNotFound
		}
		return nil, err
	}
	return s.classRepo.GetByID(ctx, classID)
}

func (s *ClassService) Delete(ctx context.Context, classID int64) error {
	err := s.classRepo.Delete(ctx, classID)
	if errors.Is(err, sql.ErrNoRows) {
		return errClassNotFound
	}
	return err
}

// List gives an admin every class and a teacher only their own. FR-CLS-04.
func (s *ClassService) List(ctx context.Context, actorID int64, role string) ([]dto.ClassResponse, error) {
	if role == "ADMIN" {
		return s.classRepo.List(ctx, nil)
	}
	return s.classRepo.List(ctx, &actorID)
}

func (s *ClassService) Get(ctx context.Context, classID, actorID int64, role string) (*dto.ClassDetailResponse, error) {
	class, err := s.classRepo.GetByID(ctx, classID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errClassNotFound
		}
		return nil, err
	}
	if err := s.assertMayRead(class, actorID, role); err != nil {
		return nil, err
	}

	students, err := s.classRepo.ListStudents(ctx, classID)
	if err != nil {
		return nil, err
	}
	return &dto.ClassDetailResponse{ClassResponse: *class, Students: students}, nil
}

// AddStudent puts a learner on the roster. FR-CLS-02.
func (s *ClassService) AddStudent(ctx context.Context, classID, studentID, actorID int64) error {
	exists, err := s.classRepo.StudentExists(ctx, studentID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("student not found")
	}

	err = s.classRepo.AddStudent(ctx, classID, studentID, actorID)
	if errors.Is(err, sql.ErrNoRows) {
		return errClassNotFound
	}
	return err
}

// AddStudents fills a roster in one call. A rejected id is reported rather than
// aborting the batch: an admin pasting thirty student numbers should not lose
// twenty-nine of them to one typo.
func (s *ClassService) AddStudents(ctx context.Context, classID int64, studentIDs []int64, actorID int64) (*dto.AddClassStudentsResult, error) {
	if _, _, err := s.classRepo.TeacherOf(ctx, classID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errClassNotFound
		}
		return nil, err
	}

	result := &dto.AddClassStudentsResult{}
	for _, studentID := range studentIDs {
		exists, err := s.classRepo.StudentExists(ctx, studentID)
		if err != nil {
			return nil, err
		}
		if !exists {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("không tìm thấy học viên %d", studentID))
			continue
		}
		if err := s.classRepo.AddStudent(ctx, classID, studentID, actorID); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("học viên %d: %v", studentID, err))
			continue
		}
		result.Added++
	}
	return result, nil
}

// RemoveStudent takes a learner off the roster. FR-CLS-02.
func (s *ClassService) RemoveStudent(ctx context.Context, classID, studentID int64) error {
	err := s.classRepo.RemoveStudent(ctx, classID, studentID)
	if errors.Is(err, sql.ErrNoRows) {
		return errors.New("student is not in this class")
	}
	return err
}

// assertMayRead lets an admin see any class and a teacher see the ones they run.
func (s *ClassService) assertMayRead(class *dto.ClassResponse, actorID int64, role string) error {
	if role == "ADMIN" {
		return nil
	}
	if class.TeacherID != nil && *class.TeacherID == actorID {
		return nil
	}
	return errors.New("forbidden: this class belongs to another teacher")
}

func nullInt64(v *int64) sql.NullInt64 {
	if v == nil || *v == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func nullString(v string) sql.NullString {
	if v == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: v, Valid: true}
}
