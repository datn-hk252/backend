package handler

import (
	"net/http"
	"strconv"
	"strings"

	"example/hello/internal/dto"
	"example/hello/internal/service"
	"example/hello/pkg/logger"

	"github.com/gin-gonic/gin"
)

type ClassHandler struct {
	classService *service.ClassService
}

func NewClassHandler(classService *service.ClassService) *ClassHandler {
	return &ClassHandler{classService: classService}
}

// respondClassError maps the service's errors onto status codes. Kept in one
// place so every route in this file answers the same way.
func respondClassError(c *gin.Context, err error, action string) {
	message := err.Error()
	switch {
	case strings.Contains(message, "not found"):
		c.JSON(http.StatusNotFound, dto.NewErrorResponse("not_found", message))
	case strings.Contains(message, "forbidden"):
		c.JSON(http.StatusForbidden, dto.NewErrorResponse("forbidden", message))
	case strings.Contains(message, "already exists"),
		strings.Contains(message, "is required"),
		strings.Contains(message, "not in this class"):
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("validation_error", message))
	default:
		logger.Error(action, err)
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("internal_error", message))
	}
}

func classIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("classId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("invalid_class_id", "Invalid class ID"))
		return 0, false
	}
	return id, true
}

// CreateClass opens a new cohort on a course.
// @Summary Create a class
// @Description Create a class with a name and an assigned teacher (admin only)
// @Tags classes
// @Accept json
// @Produce json
// @Param request body dto.CreateClassRequest true "Class to create"
// @Security BearerAuth
// @Success 201 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /classes [post]
func (h *ClassHandler) CreateClass(c *gin.Context) {
	var req dto.CreateClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("validation_error", err.Error()))
		return
	}

	class, err := h.classService.Create(c.Request.Context(), c.GetInt64("user_id"), &req)
	if err != nil {
		respondClassError(c, err, "Failed to create class")
		return
	}
	c.JSON(http.StatusCreated, dto.NewDataResponse(class))
}

// ListClasses returns every class for an admin, or a teacher's own.
// @Summary List classes
// @Description Admins see every class; teachers see the classes they run
// @Tags classes
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.SuccessResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /classes [get]
func (h *ClassHandler) ListClasses(c *gin.Context) {
	classes, err := h.classService.List(c.Request.Context(), c.GetInt64("user_id"), getRoleFromContext(c))
	if err != nil {
		respondClassError(c, err, "Failed to list classes")
		return
	}
	c.JSON(http.StatusOK, dto.NewDataResponse(classes))
}

// GetClass returns one class together with its roster.
// @Summary Get a class and its students
// @Tags classes
// @Produce json
// @Param classId path int true "Class ID"
// @Security BearerAuth
// @Success 200 {object} dto.SuccessResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /classes/{classId} [get]
func (h *ClassHandler) GetClass(c *gin.Context) {
	classID, ok := classIDParam(c)
	if !ok {
		return
	}

	class, err := h.classService.Get(c.Request.Context(), classID, c.GetInt64("user_id"), getRoleFromContext(c))
	if err != nil {
		respondClassError(c, err, "Failed to load class")
		return
	}
	c.JSON(http.StatusOK, dto.NewDataResponse(class))
}

// UpdateClass renames a class, reassigns its teacher or closes it.
// @Summary Update a class
// @Description Rename, reassign the teacher (FR-CLS-03) or change the status
// @Tags classes
// @Accept json
// @Produce json
// @Param classId path int true "Class ID"
// @Param request body dto.UpdateClassRequest true "Fields to change"
// @Security BearerAuth
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /classes/{classId} [put]
func (h *ClassHandler) UpdateClass(c *gin.Context) {
	classID, ok := classIDParam(c)
	if !ok {
		return
	}

	var req dto.UpdateClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("validation_error", err.Error()))
		return
	}

	class, err := h.classService.Update(c.Request.Context(), classID, &req)
	if err != nil {
		respondClassError(c, err, "Failed to update class")
		return
	}
	c.JSON(http.StatusOK, dto.NewDataResponse(class))
}

// DeleteClass removes a class and its roster.
// @Summary Delete a class
// @Tags classes
// @Produce json
// @Param classId path int true "Class ID"
// @Security BearerAuth
// @Success 200 {object} dto.SuccessResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /classes/{classId} [delete]
func (h *ClassHandler) DeleteClass(c *gin.Context) {
	classID, ok := classIDParam(c)
	if !ok {
		return
	}

	if err := h.classService.Delete(c.Request.Context(), classID); err != nil {
		respondClassError(c, err, "Failed to delete class")
		return
	}
	c.JSON(http.StatusOK, dto.NewMessageResponse("Đã xóa lớp"))
}

// AddStudent puts a learner on the roster and gives them the course.
// @Summary Add a student to a class
// @Tags classes
// @Accept json
// @Produce json
// @Param classId path int true "Class ID"
// @Param request body dto.AddClassStudentRequest true "Student to add"
// @Security BearerAuth
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /classes/{classId}/students [post]
func (h *ClassHandler) AddStudent(c *gin.Context) {
	classID, ok := classIDParam(c)
	if !ok {
		return
	}

	var req dto.AddClassStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("validation_error", err.Error()))
		return
	}

	err := h.classService.AddStudent(c.Request.Context(), classID, req.StudentID, c.GetInt64("user_id"))
	if err != nil {
		respondClassError(c, err, "Failed to add student to class")
		return
	}
	c.JSON(http.StatusOK, dto.NewMessageResponse("Đã thêm học viên vào lớp"))
}

// AddStudents fills a roster from a list.
// @Summary Add several students to a class
// @Tags classes
// @Accept json
// @Produce json
// @Param classId path int true "Class ID"
// @Param request body dto.AddClassStudentsRequest true "Students to add"
// @Security BearerAuth
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /classes/{classId}/students/bulk [post]
func (h *ClassHandler) AddStudents(c *gin.Context) {
	classID, ok := classIDParam(c)
	if !ok {
		return
	}

	var req dto.AddClassStudentsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("validation_error", err.Error()))
		return
	}

	result, err := h.classService.AddStudents(c.Request.Context(), classID, req.StudentIDs, c.GetInt64("user_id"))
	if err != nil {
		respondClassError(c, err, "Failed to add students to class")
		return
	}
	c.JSON(http.StatusOK, dto.NewDataResponse(result))
}

// RemoveStudent takes a learner off the roster.
// @Summary Remove a student from a class
// @Tags classes
// @Produce json
// @Param classId path int true "Class ID"
// @Param studentId path int true "Student ID"
// @Security BearerAuth
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Router /classes/{classId}/students/{studentId} [delete]
func (h *ClassHandler) RemoveStudent(c *gin.Context) {
	classID, ok := classIDParam(c)
	if !ok {
		return
	}
	studentID, err := strconv.ParseInt(c.Param("studentId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("invalid_student_id", "Invalid student ID"))
		return
	}

	if err := h.classService.RemoveStudent(c.Request.Context(), classID, studentID); err != nil {
		respondClassError(c, err, "Failed to remove student from class")
		return
	}
	c.JSON(http.StatusOK, dto.NewMessageResponse("Đã gỡ học viên khỏi lớp"))
}

// SetStudentStatus godoc
// @Summary Mark a learner as still attending this class, or as having left it
// @Tags classes
// @Param classId path int true "Class ID"
// @Param studentId path int true "Student ID"
// @Param request body dto.SetStudentStatusRequest true "New status"
// @Success 200 {object} dto.MessageResponse
// @Router /classes/{classId}/students/{studentId}/status [put]
func (h *ClassHandler) SetStudentStatus(c *gin.Context) {
	classID, ok := classIDParam(c)
	if !ok {
		return
	}
	studentID, err := strconv.ParseInt(c.Param("studentId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("invalid_student_id", "Invalid student ID"))
		return
	}

	var req dto.SetStudentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("validation_error", err.Error()))
		return
	}

	if err := h.classService.SetStudentStatus(c.Request.Context(), classID, studentID, req.Status); err != nil {
		respondClassError(c, err, "Failed to update student status")
		return
	}

	if req.Status == "DROPPED" {
		c.JSON(http.StatusOK, dto.NewMessageResponse("Đã đánh dấu học viên nghỉ lớp"))
		return
	}
	c.JSON(http.StatusOK, dto.NewMessageResponse("Đã đánh dấu học viên học lại"))
}
