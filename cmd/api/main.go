// main.go

// @title           GoRent API
// @version         1.0
// @description     API для сервиса аренды автомобилей с ролевой моделью, JWT-аутентификацией и финансовой аналитикой.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@gorrent.com

// @license.name   MIT
// @license.url    https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите JWT токен в формате: Bearer {token}

package main

import (
	"log"

	"rent_project/internal/config"
	"rent_project/internal/database"
	"rent_project/internal/handlers"
	"rent_project/internal/middleware"
	"rent_project/internal/services"
	"rent_project/internal/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if cfg.ServerMode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	jwtService := utils.NewJWTService(cfg.JWTSecret, cfg.JWTExpirationHours)

	authService := services.NewAuthService(db.DB, jwtService)
	carService := services.NewCarService(db.DB)
	rentalService := services.NewRentalService(db.DB, carService)
	analyticsService := services.NewAnalyticsService(db.DB)

	authHandler := handlers.NewAuthHandler(authService)
	carHandler := handlers.NewCarHandler(carService)
	rentalHandler := handlers.NewRentalHandler(rentalService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)

	authGroup := router.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
	}

	publicCarGroup := router.Group("/cars")
	{
		publicCarGroup.GET("", carHandler.GetCars)
		publicCarGroup.GET("/:id", carHandler.GetCarByID)
	}

	protectedGroup := router.Group("/")
	protectedGroup.Use(middleware.AuthMiddleware(jwtService))
	{
		protectedGroup.GET("/users/me", authHandler.GetMe)

		protectedGroup.POST("/rentals", rentalHandler.CreateRental)
		protectedGroup.PUT("/rentals/:id/cancel", rentalHandler.CancelRental)
		protectedGroup.GET("/rentals/my", rentalHandler.GetMyRentals)
	}

	managerGroup := router.Group("/")
	managerGroup.Use(middleware.AuthMiddleware(jwtService))
	managerGroup.Use(middleware.RoleMiddleware("manager", "admin"))
	{
		managerGroup.POST("/cars", carHandler.CreateCar)
		managerGroup.PUT("/cars/:id", carHandler.UpdateCar)
		managerGroup.DELETE("/cars/:id", carHandler.DeleteCar)
		managerGroup.PUT("/cars/:id/availability", carHandler.UpdateCarAvailability)

		managerGroup.PUT("/rentals/:id/approve", rentalHandler.ApproveRental)
	}

	adminGroup := router.Group("/admin")
	adminGroup.Use(middleware.AuthMiddleware(jwtService))
	adminGroup.Use(middleware.RoleMiddleware("admin"))
	{
		adminGroup.PUT("/users/:id/role", authHandler.UpdateUserRole)

		adminGroup.GET("/rentals", rentalHandler.GetAllRentals)

		adminGroup.GET("/analytics/profit", analyticsHandler.GetProfit)
		adminGroup.GET("/analytics/popular-brands", analyticsHandler.GetPopularBrands)
		adminGroup.GET("/analytics/daily-profit", analyticsHandler.GetDailyProfit)
		adminGroup.GET("/analytics/statistics", analyticsHandler.GetStatistics)
		adminGroup.GET("/analytics/car-utilization", analyticsHandler.GetCarUtilization)
	}

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := router.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
