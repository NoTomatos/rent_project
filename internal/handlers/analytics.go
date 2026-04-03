package handlers

import (
	"net/http"
	"strconv"
	"time"

	"rent_project/internal/services"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
}

func NewAnalyticsHandler(analyticsService *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// @Summary Финансовая аналитика
// @Description Возвращает общую прибыль и количество аренд за указанный период
// @Tags analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param from query string true "Дата начала (YYYY-MM-DD)"
// @Param to query string true "Дата окончания (YYYY-MM-DD)"
// @Success 200 {object} models.ProfitAnalytics
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/analytics/profit [get]
func (h *AnalyticsHandler) GetProfit(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")

	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to parameters are required"})
		return
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from date format, use YYYY-MM-DD"})
		return
	}

	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to date format, use YYYY-MM-DD"})
		return
	}

	to = to.Add(24*time.Hour - time.Second)

	analytics, err := h.analyticsService.GetProfitAnalytics(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

// @Summary Популярные бренды
// @Description Возвращает список самых популярных брендов по количеству аренд
// @Tags analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Количество брендов" default(5)
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/analytics/popular-brands [get]
func (h *AnalyticsHandler) GetPopularBrands(c *gin.Context) {
	limit := 5
	if limitStr := c.Query("limit"); limitStr != "" {
		l, err := strconv.Atoi(limitStr)
		if err == nil && l > 0 {
			limit = l
		}
	}

	brands, err := h.analyticsService.GetPopularBrands(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  brands,
		"limit": limit,
	})
}

// @Summary Ежедневная прибыль
// @Description Возвращает ежедневную разбивку прибыли за период
// @Tags analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param from query string true "Дата начала (YYYY-MM-DD)"
// @Param to query string true "Дата окончания (YYYY-MM-DD)"
// @Success 200 {array} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/analytics/daily-profit [get]
func (h *AnalyticsHandler) GetDailyProfit(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")

	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to parameters are required"})
		return
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from date format, use YYYY-MM-DD"})
		return
	}

	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to date format, use YYYY-MM-DD"})
		return
	}

	to = to.Add(24*time.Hour - time.Second)

	dailyStats, err := h.analyticsService.GetDailyProfit(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": dailyStats,
		"from": from.Format("2006-01-02"),
		"to":   toStr,
	})
}

// @Summary Общая статистика
// @Description Возвращает статистику по пользователям и арендам
// @Tags analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/analytics/statistics [get]
func (h *AnalyticsHandler) GetStatistics(c *gin.Context) {
	stats, err := h.analyticsService.GetUserStatistics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// @Summary Использование автомобилей
// @Description Возвращает статистику по использованию каждого автомобиля
// @Tags analytics
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/analytics/car-utilization [get]
func (h *AnalyticsHandler) GetCarUtilization(c *gin.Context) {
	utilization, err := h.analyticsService.GetCarUtilization()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       utilization,
		"total_cars": len(utilization),
	})
}
