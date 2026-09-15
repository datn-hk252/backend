package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"example/hello/internal/dto"
	"example/hello/internal/repository"
	"example/hello/pkg/ai"
	"example/hello/pkg/cache"
	"example/hello/pkg/logger"

)

type AnalyticsService struct {
	analyticsRepo  *repository.AnalyticsRepository
	courseRepo     *repository.CourseRepository
	enrollmentRepo *repository.EnrollmentRepository
	aiClient       *ai.Client
	redisCache     *cache.RedisCache
}

func NewAnalyticsService(
	analyticsRepo *repository.AnalyticsRepository,
	courseRepo *repository.CourseRepository,
	enrollmentRepo *repository.EnrollmentRepository,
	aiClient *ai.Client,
	redisCache *cache.RedisCache,
) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepo:  analyticsRepo,
		courseRepo:     courseRepo,
		enrollmentRepo: enrollmentRepo,
		aiClient:       aiClient,
		redisCache:     redisCache,
	}
}

// ─── Teacher methods ──────────────────────────────────────────────────────────

func (s *AnalyticsService) GetCourseQuizAnalytics(ctx context.Context, courseID int64) ([]dto.QuizPerformanceSummary, error) {
	rows, err := s.analyticsRepo.GetCourseQuizAnalytics(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("GetCourseQuizAnalytics: %w", err)
	}

	result := make([]dto.QuizPerformanceSummary, 0, len(rows))
	for _, r := range rows {
		item := dto.QuizPerformanceSummary{
			QuizID:         r.QuizID,
			QuizTitle:      r.QuizTitle,
			ContentID:      r.ContentID,
			TotalAttempts:  r.TotalAttempts,
			UniqueStudents: r.UniqueStudents,
			AvgScore:       nfv(r.AvgScore),
			AvgPercentage:  nfv(r.AvgPercentage),
			PassRate:       nfv(r.PassRate),
		}
		if r.PassingScore.Valid {
			v := r.PassingScore.Float64
			item.PassingScore = &v
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *AnalyticsService) GetQuizAllAttempts(ctx context.Context, quizID int64) ([]dto.StudentAttemptOverview, error) {
	rows, err := s.analyticsRepo.GetQuizAllAttempts(ctx, quizID)
	if err != nil {
		return nil, fmt.Errorf("GetQuizAllAttempts: %w", err)
	}

	result := make([]dto.StudentAttemptOverview, 0, len(rows))
	for _, r := range rows {
		item := dto.StudentAttemptOverview{
			StudentID:     r.StudentID,
			StudentName:   r.StudentName,
			StudentEmail:  r.StudentEmail,
			QuizID:        r.QuizID,
			QuizTitle:     r.QuizTitle,
			AttemptNumber: r.AttemptNumber,
			TotalPoints:   r.TotalPoints,
			Status:        r.Status,
		}
		if r.EarnedPoints.Valid {
			v := r.EarnedPoints.Float64
			item.EarnedPoints = &v
		}
		if r.Percentage.Valid {
			v := r.Percentage.Float64
			item.Percentage = &v
		}
		if r.IsPassed.Valid {
			v := r.IsPassed.Bool
			item.IsPassed = &v
		}
		if r.SubmittedAt.Valid {
			v := r.SubmittedAt.Time
			item.SubmittedAt = &v
		}
		result = append(result, item)
	}
	return result, nil
}

func (s *AnalyticsService) GetQuizWrongAnswerStats(ctx context.Context, quizID int64) ([]dto.WrongAnswerStat, error) {
	rows, err := s.analyticsRepo.GetQuizWrongAnswerStats(ctx, quizID)
	if err != nil {
		return nil, fmt.Errorf("GetQuizWrongAnswerStats: %w", err)
	}

	result := make([]dto.WrongAnswerStat, 0, len(rows))
	for _, r := range rows {
		result = append(result, dto.WrongAnswerStat{
			QuestionID:   r.QuestionID,
			QuestionText: r.QuestionText,
			QuestionType: r.QuestionType,
			TotalAnswers: r.TotalAnswers,
			WrongCount:   r.WrongCount,
			WrongRate:    r.WrongRate,
		})
	}
	return result, nil
}

func (s *AnalyticsService) GetCourseStudentProgressOverview(ctx context.Context, courseID int64) ([]dto.CourseStudentProgress, error) {
	rows, err := s.analyticsRepo.GetCourseStudentProgressOverview(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("GetCourseStudentProgressOverview: %w", err)
	}

	result := make([]dto.CourseStudentProgress, 0, len(rows))
	for _, r := range rows {
		item := dto.CourseStudentProgress{
			StudentID:        r.StudentID,
			StudentName:      r.StudentName,
			StudentEmail:     r.StudentEmail,
			StudentAvatarURL: r.StudentAvatarURL,
			TotalMandatory:   r.TotalMandatory,
			CompletedContent: r.CompletedContent,
			ProgressPercent:  r.ProgressPercent,
		}
		if r.QuizAvgScore.Valid {
			v := r.QuizAvgScore.Float64
			item.QuizAvgScore = &v
		}
		if r.LastActivity.Valid {
			v := r.LastActivity.Time
			item.LastActivity = &v
		}
		result = append(result, item)
	}
	return result, nil
}

// ─── Student method ───────────────────────────────────────────────────────────

func (s *AnalyticsService) GetMyQuizScores(ctx context.Context, courseID, studentID int64) ([]dto.StudentQuizScore, error) {
	rows, err := s.analyticsRepo.GetStudentQuizScores(ctx, courseID, studentID)
	if err != nil {
		return nil, fmt.Errorf("GetMyQuizScores: %w", err)
	}

	result := make([]dto.StudentQuizScore, 0, len(rows))
	for _, r := range rows {
		item := dto.StudentQuizScore{
			QuizID:        r.QuizID,
			QuizTitle:     r.QuizTitle,
			TotalPoints:   r.TotalPoints,
			AttemptsCount: r.AttemptsCount,
			Status:        r.Status,
		}
		if r.BestPct.Valid {
			v := r.BestPct.Float64
			item.BestPercentage = &v
		}
		if r.BestPoints.Valid {
			v := r.BestPoints.Float64
			item.BestPoints = &v
		}
		if r.IsPassed.Valid {
			v := r.IsPassed.Bool
			item.IsPassed = &v
		}
		if r.PassingScore.Valid {
			v := r.PassingScore.Float64
			item.PassingScore = &v
		}
		if r.LastAttemptAt.Valid {
			v := r.LastAttemptAt.Time
			item.LastAttemptAt = &v
		}
		result = append(result, item)
	}
	return result, nil
}

// ─── Permission helpers ───────────────────────────────────────────────────────

// VerifyCourseOwnership checks the caller may act on the course as one of its
// teachers - author, co-teacher, or teacher of a class running it - or is admin.
func (s *AnalyticsService) VerifyCourseOwnership(ctx context.Context, courseID, userID int64, userRole string) error {
	if userRole == "ADMIN" {
		return nil
	}
	course, err := s.courseRepo.GetByID(ctx, courseID)
	if err != nil {
		return fmt.Errorf("course not found")
	}
	if course.CreatedBy != userID {
		canTeachCourse, err := s.courseRepo.IsCourseTeacher(ctx, courseID, userID)
		if err != nil || !canTeachCourse {
			return fmt.Errorf("permission denied: you don't own this course")
		}
	}
	return nil
}

// VerifyQuizCourseOwnership checks the caller owns the course that contains the quiz.
func (s *AnalyticsService) VerifyQuizCourseOwnership(ctx context.Context, quizID, userID int64, userRole string) error {
	if userRole == "ADMIN" {
		return nil
	}
	courseID, err := s.analyticsRepo.GetQuizCourseID(ctx, quizID)
	if err != nil || courseID == 0 {
		return fmt.Errorf("quiz not found")
	}
	return s.VerifyCourseOwnership(ctx, courseID, userID, userRole)
}

func (s *AnalyticsService) GetTeacherDashboardSummary(ctx context.Context, teacherID int64) (*dto.TeacherDashboardSummaryResponse, error) {
	cacheKey := fmt.Sprintf("analytics:teacher:%d:dashboard", teacherID)

	// Check cache
	if s.redisCache != nil {
		if cachedVal, err := s.redisCache.Get(ctx, cacheKey); err == nil && cachedVal != "" {
			var cachedResp dto.TeacherDashboardSummaryResponse
			if err := json.Unmarshal([]byte(cachedVal), &cachedResp); err == nil {
				logger.Info(fmt.Sprintf("Cache HIT: teacher dashboard summary for teacher %d", teacherID))
				return &cachedResp, nil
			}
		}
	}
	logger.Info(fmt.Sprintf("Cache MISS: teacher dashboard summary for teacher %d", teacherID))

	repoSummary, err := s.analyticsRepo.GetTeacherDashboardSummary(ctx, teacherID)
	if err != nil {
		return nil, fmt.Errorf("failed to get teacher dashboard summary: %w", err)
	}

	resp := &dto.TeacherDashboardSummaryResponse{
		TotalCoursesCount:     repoSummary.TotalCoursesCount,
		PublishedCoursesCount: repoSummary.PublishedCoursesCount,
		DraftCoursesCount:     repoSummary.DraftCoursesCount,
		TotalUniqueStudents:   repoSummary.TotalUniqueStudents,
		RegistrationTimeline:  make([]dto.RegistrationTimeline, 0, len(repoSummary.RegistrationTimeline)),
		CourseStats:          make([]dto.TeacherCourseStats, 0, len(repoSummary.CourseStats)),
	}

	for _, item := range repoSummary.RegistrationTimeline {
		resp.RegistrationTimeline = append(resp.RegistrationTimeline, dto.RegistrationTimeline{
			Date:  item.EnrollDate.Format("02/01"),
			Count: item.NewLearners,
		})
	}

	for _, item := range repoSummary.CourseStats {
		var avgQuiz *float64
		if item.AvgQuiz.Valid {
			val := item.AvgQuiz.Float64
			avgQuiz = &val
		}

		thumbnailURL := ""
		if item.ThumbnailURL.Valid {
			thumbnailURL = item.ThumbnailURL.String
		}

		resp.CourseStats = append(resp.CourseStats, dto.TeacherCourseStats{
			ID:           item.CourseID,
			Title:        item.Title,
			ThumbnailURL: thumbnailURL,
			StudentCount: item.StudentCount,
			AvgProgress:  item.AvgProgress,
			AvgQuiz:      avgQuiz,
		})
	}

	// Cache the result for 5 minutes (300 seconds)
	if s.redisCache != nil {
		if data, err := json.Marshal(resp); err == nil {
			_ = s.redisCache.Set(ctx, cacheKey, data, 5*time.Minute)
		}
	}

	return resp, nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// nfv (null float value) returns 0 for invalid NullFloat64.
func nfv(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0
}
