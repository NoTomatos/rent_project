package models

import (
	"time"

	"github.com/google/uuid"
)

type Car struct {
	ID          uuid.UUID `json:"id"`
	Model       string    `json:"model"`
	Brand       string    `json:"brand"`
	Year        int       `json:"year"`
	PricePerDay float64   `json:"price_per_day"`
	IsAvailable bool      `json:"is_available"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CarRequest struct {
	Model       string  `json:"model" binding:"required,min=1,max=100"`
	Brand       string  `json:"brand" binding:"required,min=1,max=100"`
	Year        int     `json:"year" binding:"required,min=1900,max=2030"`
	PricePerDay float64 `json:"price_per_day" binding:"required,gt=0"`
}

type CarListRequest struct {
	Available *bool   `form:"available"`
	Brand     string  `form:"brand"`
	Model     string  `form:"model"`
	MinPrice  float64 `form:"min_price"`
	MaxPrice  float64 `form:"max_price"`
	Limit     int     `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset    int     `form:"offset" binding:"omitempty,min=0"`
}
