package handlers

import (
	"net/http"
	"strconv"

	"rent_project/internal/middleware"
	"rent_project/internal/models"
	"rent_project/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RentalHandler struct {
	rentalService *services.RentalService
}

func NewRentalHandler(rentalService *services.RentalService) *RentalHandler {
	return &RentalHandler{
		rentalService: rentalService,
	}
}

// @Summary Создать аренду
// @Description Бронирует автомобиль на указанные даты
// @Tags rentals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.RentalRequest true "Данные аренды"
// @Success 201 {object} models.RentalResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /rentals [post]
func (h *RentalHandler) CreateRental(c *gin.Context) {
	userIDStr, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID format"})
		return
	}

	var req models.RentalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rental, err := h.rentalService.CreateRental(userID, &req)
	if err != nil {
		if err.Error() == "car not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "car is not available for rent" ||
			err.Error() == "car is already booked for the selected dates" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, rental)
}

// @Summary Отменить аренду
// @Description Отменяет существующее бронирование
// @Tags rentals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID аренды"
// @Success 200 {object} models.RentalResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /rentals/{id}/cancel [put]
func (h *RentalHandler) CancelRental(c *gin.Context) {
	userIDStr, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID format"})
		return
	}

	rentalIDStr := c.Param("id")
	rentalID, err := uuid.Parse(rentalIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rental ID format"})
		return
	}

	rental, err := h.rentalService.CancelRental(rentalID, userID)
	if err != nil {
		if err.Error() == "rental not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "you can only cancel your own rentals" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "rental is already canceled" ||
			err.Error() == "cannot cancel completed rental" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rental)
}

// @Summary Подтвердить аренду
// @Description Подтверждает или отклоняет бронирование менеджером
// @Tags rentals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID аренды"
// @Param request body models.ApproveRequest true "Статус аренды (active/canceled)"
// @Success 200 {object} models.RentalResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /rentals/{id}/approve [put]
func (h *RentalHandler) ApproveRental(c *gin.Context) {
	rentalIDStr := c.Param("id")
	rentalID, err := uuid.Parse(rentalIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid rental ID format"})
		return
	}

	var req models.ApproveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rental, err := h.rentalService.ApproveRental(rentalID, req.Status)
	if err != nil {
		if err.Error() == "rental not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "car is no longer available" ||
			err.Error() == "car is already booked for these dates" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rental)
}

// @Summary Мои аренды
// @Description Возвращает список всех аренд текущего пользователя
// @Tags rentals
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Лимит записей" default(20)
// @Param offset query int false "Смещение" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /rentals/my [get]
func (h *RentalHandler) GetMyRentals(c *gin.Context) {
	userIDStr, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID format"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	rentals, total, err := h.rentalService.GetUserRentals(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": rentals,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// @Summary Все аренды
// @Description Возвращает список всех аренд (доступно manager и admin)
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Фильтр по статусу (pending, active, completed, canceled)"
// @Param limit query int false "Лимит записей" default(20)
// @Param offset query int false "Смещение" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/rentals [get]
func (h *RentalHandler) GetAllRentals(c *gin.Context) {
	// Парсим параметры
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	rentals, total, err := h.rentalService.GetAllRentals(status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": rentals,
		"meta": gin.H{
			"total":  total,
			"limit":  limit,
			"offset": offset,
			"status": status,
		},
	})
}
