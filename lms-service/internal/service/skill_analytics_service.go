package service

import (
	"context"
	"fmt"

	"example/hello/internal/dto"
	"example/hello/internal/repository"
)

// SkillAnalyticsService backs the three teacher-facing skill breakdown views.
// Every method verifies course ownership before reading any result data.
type SkillAnalyticsService struct {
	skillRepo  *repository.SkillAnalyticsRepository
	courseRepo *repository.CourseRepository
	quizRepo   *repository.QuizRepository
}

func NewSkillAnalyticsService(
	skillRepo *repository.SkillAnalyticsRepository,
	courseRepo *repository.CourseRepository,
	quizRepo *repository.QuizRepository,
) *SkillAnalyticsService {
	return &SkillAnalyticsService{
		skillRepo:  skillRepo,
		courseRepo: courseRepo,
		quizRepo:   quizRepo,
	}
}

// verifyCourseAccess lets admins see everything, while a teacher may only see
// courses they created or co-teach.
func (s *SkillAnalyticsService) verifyCourseAccess(
	ctx context.Context, courseID, userID int64, userRole string,
) error {
	if userRole == "ADMIN" {
		return nil
	}
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return fmt.Errorf("course not found")
	}
	if course.CreatedBy == userID {
		return nil
	}
	canTeachCourse, err := s.courseRepo.IsCourseTeacher(ctx, courseID, userID)
	if err != nil || !canTeachCourse {
		return fmt.Errorf("permission denied: you don't own this course")
	}
	return nil
}

// GetStudentQuizBreakdown returns one student's skill profile on one quiz.
func (s *SkillAnalyticsService) GetStudentQuizBreakdown(
	ctx context.Context, courseID, quizID, studentID, userID int64, userRole string,
) (*dto.StudentQuizSkillBreakdown, error) {

	if err := s.verifyCourseAccess(ctx, courseID, userID, userRole); err != nil {
		return nil, err
	}

	scores, err := s.skillRepo.GetStudentQuizSkillScores(ctx, quizID, studentID)
	if err != nil {
		return nil, err
	}
	untagged, err := s.skillRepo.CountUntaggedQuestions(ctx, quizID)
	if err != nil {
		return nil, err
	}
	title, submittedAt, err := s.skillRepo.GetQuizMeta(ctx, quizID, &studentID)
	if err != nil {
		return nil, err
	}

	return &dto.StudentQuizSkillBreakdown{
		StudentID:         studentID,
		QuizID:            quizID,
		QuizTitle:         title,
		SubmittedAt:       submittedAt,
		UntaggedQuestions: untagged,
		Skills:            scores,
	}, nil
}

// GetClassQuizBreakdown returns the whole class's skill profile on one quiz.
func (s *SkillAnalyticsService) GetClassQuizBreakdown(
	ctx context.Context, courseID, quizID, userID int64, userRole string,
) (*dto.ClassQuizSkillBreakdown, error) {

	if err := s.verifyCourseAccess(ctx, courseID, userID, userRole); err != nil {
		return nil, err
	}

	scores, studentCount, err := s.skillRepo.GetClassQuizSkillScores(ctx, quizID)
	if err != nil {
		return nil, err
	}
	untagged, err := s.skillRepo.CountUntaggedQuestions(ctx, quizID)
	if err != nil {
		return nil, err
	}
	title, _, err := s.skillRepo.GetQuizMeta(ctx, quizID, nil)
	if err != nil {
		return nil, err
	}

	return &dto.ClassQuizSkillBreakdown{
		QuizID:            quizID,
		QuizTitle:         title,
		StudentCount:      studentCount,
		UntaggedQuestions: untagged,
		Skills:            scores,
	}, nil
}

// GetStudentTrend compares one student's skill profile across the quizzes of
// a course.
func (s *SkillAnalyticsService) GetStudentTrend(
	ctx context.Context, courseID, studentID, userID int64, userRole string,
) (*dto.StudentSkillTrend, error) {

	if err := s.verifyCourseAccess(ctx, courseID, userID, userRole); err != nil {
		return nil, err
	}

	series, err := s.skillRepo.GetStudentSkillTrend(ctx, studentID, courseID)
	if err != nil {
		return nil, err
	}

	return &dto.StudentSkillTrend{
		StudentID: studentID,
		CourseID:  courseID,
		Series:    series,
	}, nil
}

// ListSkills returns the skill taxonomy. It is shared reference data, so any
// authenticated caller may read it; there is nothing course-specific to guard.
func (s *SkillAnalyticsService) ListSkills(ctx context.Context) ([]dto.SkillNode, error) {
	return s.skillRepo.ListSkills(ctx)
}
