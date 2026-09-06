package handler

import (
	"net/http"
	"strconv"

	"example/hello/internal/dto"
	"example/hello/internal/service"
	"example/hello/pkg/logger"

	"github.com/gin-gonic/gin"
)

// SkillAnalyticsHandler serves the three teacher views: one student's skill
// profile on a quiz, the whole class's, and one student compared over time.
//
// All three are teacher and admin only. There is no student-facing equivalent:
// the platform is built around supporting the teacher.
type SkillAnalyticsHandler struct {
	skillAnalyticsService *service.SkillAnalyticsService
}

func NewSkillAnalyticsHandler(s *service.SkillAnalyticsService) *SkillAnalyticsHandler {
	return &SkillAnalyticsHandler{skillAnalyticsService: s}
}

func getStudentIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("studentId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("invalid_student_id", "Invalid student ID"))
		return 0, false
	}
	return id, true
}

// GetStudentQuizSkillBreakdown godoc
// @Summary      One student's skill breakdown for a quiz
// @Tags         Analytics - Skills
// @Produce      json
// @Param        courseId  path int true "Course ID"
// @Param        quizId    path int true "Quiz ID"
// @Param        studentId path int true "Student ID"
// @Security     BearerAuth
// @Router       /courses/{courseId}/quizzes/{quizId}/skill-breakdown/students/{studentId} [get]
func (h *SkillAnalyticsHandler) GetStudentQuizSkillBreakdown(c *gin.Context) {
	courseID, ok := getCourseIDParam(c)
	if !ok {
		return
	}
	quizID, ok := getQuizIDParam(c)
	if !ok {
		return
	}
	studentID, ok := getStudentIDParam(c)
	if !ok {
		return
	}

	userID := c.MustGet("user_id").(int64)
	userRole := c.MustGet("user_role").(string)

	data, err := h.skillAnalyticsService.GetStudentQuizBreakdown(
		c.Request.Context(), courseID, quizID, studentID, userID, userRole)
	if err != nil {
		logger.Error("GetStudentQuizSkillBreakdown failed", err)
		c.JSON(http.StatusForbidden, dto.NewErrorResponse("forbidden", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewDataResponse(data))
}

// GetClassQuizSkillBreakdown godoc
// @Summary      Class-wide skill breakdown for a quiz
// @Tags         Analytics - Skills
// @Produce      json
// @Param        courseId path int true "Course ID"
// @Param        quizId   path int true "Quiz ID"
// @Security     BearerAuth
// @Router       /courses/{courseId}/quizzes/{quizId}/skill-breakdown [get]
func (h *SkillAnalyticsHandler) GetClassQuizSkillBreakdown(c *gin.Context) {
	courseID, ok := getCourseIDParam(c)
	if !ok {
		return
	}
	quizID, ok := getQuizIDParam(c)
	if !ok {
		return
	}

	userID := c.MustGet("user_id").(int64)
	userRole := c.MustGet("user_role").(string)

	data, err := h.skillAnalyticsService.GetClassQuizBreakdown(
		c.Request.Context(), courseID, quizID, userID, userRole)
	if err != nil {
		logger.Error("GetClassQuizSkillBreakdown failed", err)
		c.JSON(http.StatusForbidden, dto.NewErrorResponse("forbidden", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewDataResponse(data))
}

// GetStudentSkillTrend godoc
// @Summary      One student's skill scores compared across quizzes
// @Tags         Analytics - Skills
// @Produce      json
// @Param        courseId  path int true "Course ID"
// @Param        studentId path int true "Student ID"
// @Security     BearerAuth
// @Router       /courses/{courseId}/skill-trend/students/{studentId} [get]
func (h *SkillAnalyticsHandler) GetStudentSkillTrend(c *gin.Context) {
	courseID, ok := getCourseIDParam(c)
	if !ok {
		return
	}
	studentID, ok := getStudentIDParam(c)
	if !ok {
		return
	}

	userID := c.MustGet("user_id").(int64)
	userRole := c.MustGet("user_role").(string)

	data, err := h.skillAnalyticsService.GetStudentTrend(
		c.Request.Context(), courseID, studentID, userID, userRole)
	if err != nil {
		logger.Error("GetStudentSkillTrend failed", err)
		c.JSON(http.StatusForbidden, dto.NewErrorResponse("forbidden", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewDataResponse(data))
}

// ListSkills godoc
// @Summary      List the skill taxonomy
// @Tags         Analytics - Skills
// @Produce      json
// @Security     BearerAuth
// @Router       /skills [get]
func (h *SkillAnalyticsHandler) ListSkills(c *gin.Context) {
	data, err := h.skillAnalyticsService.ListSkills(c.Request.Context())
	if err != nil {
		logger.Error("ListSkills failed", err)
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("internal_error", "Failed to list skills"))
		return
	}
	c.JSON(http.StatusOK, dto.NewDataResponse(data))
}
