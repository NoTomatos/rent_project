package handlers

import (
	"net/http"
	"strconv"

	"rent_project/internal/models"
	"rent_project/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CarHandler struct {
	carService *services.CarService
}

func NewCarHandler(carService *services.CarService) *CarHandler {
	return &CarHandler{
		carService: carService,
	}
}

// @Summary Получить список автомобилей
// @Description Возвращает список доступных автомобилей с возможностью фильтрации
// @Tags cars
// @Accept json
// @Produce json
// @Param available query bool false "Только доступные"
// @Param brand query string false "Фильтр по бренду"
// @Param model query string false "Фильтр по модели"
// @Param min_price query number false "Минимальная цена"
// @Param max_price query number false "Максимальная цена"
// @Param limit query int false "Лимит записей" default(20)
// @Param offset query int false "Смещение" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /cars [get]
func (h *CarHandler) GetCars(c *gin.Context) {
	var req models.CarListRequest

	if availableStr := c.Query("available"); availableStr != "" {
		available, err := strconv.ParseBool(availableStr)
		if err == nil {
			req.Available = &available
		}
	}

	req.Brand = c.Query("brand")

	req.Model = c.Query("model")

	if minPriceStr := c.Query("min_price"); minPriceStr != "" {
		minPrice, err := strconv.ParseFloat(minPriceStr, 64)
		if err == nil {
			req.MinPrice = minPrice
		}
	}

	if maxPriceStr := c.Query("max_price"); maxPriceStr != "" {
		maxPrice, err := strconv.ParseFloat(maxPriceStr, 64)
		if err == nil {
			req.MaxPrice = maxPrice
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err == nil {
			req.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err == nil {
			req.Offset = offset
		}
	}

	cars, total, err := h.carService.GetAllCars(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": cars,
		"meta": gin.H{
			"total":  total,
			"limit":  req.Limit,
			"offset": req.Offset,
		},
	})
}

// @Summary Получить автомобиль по ID
// @Description Возвращает информацию об автомобиле
// @Tags cars
// @Accept json
// @Produce json
// @Param id path string true "ID автомобиля"
// @Success 200 {object} models.Car
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /cars/{id} [get]
func (h *CarHandler) GetCarByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car ID format"})
		return
	}

	car, err := h.carService.GetCarByID(id)
	if err != nil {
		if err.Error() == "car not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, car)
}

// @Summary Создать автомобиль
// @Description Добавляет новый автомобиль в автопарк (доступно manager и admin)
// @Tags cars
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CarRequest true "Данные автомобиля"
// @Success 201 {object} models.Car
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /cars [post]
func (h *CarHandler) CreateCar(c *gin.Context) {
	var req models.CarRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	car, err := h.carService.CreateCar(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, car)
}

// @Summary Обновить автомобиль
// @Description Обновляет данные существующего автомобиля (доступно manager и admin)
// @Tags cars
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID автомобиля"
// @Param request body models.CarRequest true "Данные автомобиля"
// @Success 200 {object} models.Car
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /cars/{id} [put]
func (h *CarHandler) UpdateCar(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car ID format"})
		return
	}

	var req models.CarRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	car, err := h.carService.UpdateCar(id, &req)
	if err != nil {
		if err.Error() == "car not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, car)
}

// @Summary Удалить автомобиль
// @Description Удаляет автомобиль из автопарка (доступно manager и admin)
// @Tags cars
// @Security BearerAuth
// @Param id path string true "ID автомобиля"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /cars/{id} [delete]
func (h *CarHandler) DeleteCar(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car ID format"})
		return
	}

	err = h.carService.DeleteCar(id)
	if err != nil {
		if err.Error() == "car not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "cannot delete car with active or pending rentals" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// @Summary Обновить статус доступности
// @Description Изменяет статус доступности автомобиля (доступно manager и admin)
// @Tags cars
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID автомобиля"
// @Param request body object true "Статус доступности"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /cars/{id}/availability [put]
func (h *CarHandler) UpdateCarAvailability(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid car ID format"})
		return
	}

	var req struct {
		IsAvailable bool `json:"is_available"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.carService.UpdateCarAvailability(id, req.IsAvailable)
	if err != nil {
		if err.Error() == "car not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "car availability updated successfully"})
}
