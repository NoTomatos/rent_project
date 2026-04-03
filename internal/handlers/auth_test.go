package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"rent_project/internal/config"
	"rent_project/internal/database"
	"rent_project/internal/models"
	"rent_project/internal/services"
	"rent_project/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() (*gin.Engine, *services.AuthService) {
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

	jwtService := utils.NewJWTService(cfg.JWTSecret, 24)
	authService := services.NewAuthService(db.DB, jwtService)
	authHandler := NewAuthHandler(authService)

	router := gin.Default()
	router.POST("/auth/register", authHandler.Register)
	router.POST("/auth/login", authHandler.Login)

	return router, authService
}

func TestRegister(t *testing.T) {
	router, _ := setupTestRouter()

	// Тестовые данные
	reqBody := models.RegisterRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "password123",
	}

	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response models.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, "Test User", response.User.Name)
	assert.Equal(t, "test@example.com", response.User.Email)
}

func TestRegisterDuplicateEmail(t *testing.T) {
	router, _ := setupTestRouter()

	reqBody := models.RegisterRequest{
		Name:     "Test User",
		Email:    "duplicate@example.com",
		Password: "password123",
	}

	jsonBody, _ := json.Marshal(reqBody)

	req1, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	req2, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestLogin(t *testing.T) {
	router, _ := setupTestRouter()

	// Сначала регистрируем пользователя
	registerReq := models.RegisterRequest{
		Name:     "Login User",
		Email:    "login@example.com",
		Password: "password123",
	}

	jsonBody, _ := json.Marshal(registerReq)
	req, _ := http.NewRequest("POST", "/auth/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Теперь логинимся
	loginReq := models.LoginRequest{
		Email:    "login@example.com",
		Password: "password123",
	}

	jsonLoginBody, _ := json.Marshal(loginReq)
	loginReqHTTP, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonLoginBody))
	loginReqHTTP.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginReqHTTP)

	assert.Equal(t, http.StatusOK, loginW.Code)

	var response models.LoginResponse
	err := json.Unmarshal(loginW.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, "Login User", response.User.Name)
}

func TestLoginInvalidCredentials(t *testing.T) {
	router, _ := setupTestRouter()

	loginReq := models.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "wrongpassword",
	}

	jsonBody, _ := json.Marshal(loginReq)
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
