package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rent_project/internal/config"
	"rent_project/internal/database"
	"rent_project/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupCarTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "gorrent_user",
		DBPassword: "gorrent_password",
		DBName:     "gorrent_test",
		DBSSLMode:  "disable",
		JWTSecret:  "test-secret-key",
	}

	db, _ := database.Connect(cfg)
	carService := services.NewCarService(db.DB)
	carHandler := NewCarHandler(carService)

	router := gin.Default()
	router.GET("/cars", carHandler.GetCars)
	router.GET("/cars/:id", carHandler.GetCarByID)

	return router
}

func TestGetCars(t *testing.T) {
	router := setupCarTestRouter()

	req, _ := http.NewRequest("GET", "/cars?available=true&limit=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "data")
	assert.Contains(t, response, "meta")
}

func TestGetCarByIDNotFound(t *testing.T) {
	router := setupCarTestRouter()

	nonExistentID := uuid.New().String()
	req, _ := http.NewRequest("GET", "/cars/"+nonExistentID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCarByIDInvalidUUID(t *testing.T) {
	router := setupCarTestRouter()

	req, _ := http.NewRequest("GET", "/cars/invalid-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
