package models

import (
	"time"

	"github.com/google/uuid"
)

type RentalStatus string

const (
	RentalStatusPending   RentalStatus = "pending"
	RentalStatusActive    RentalStatus = "active"
	RentalStatusCompleted RentalStatus = "completed"
	RentalStatusCanceled  RentalStatus = "canceled"
)

type Rental struct {
	ID         uuid.UUID    `json:"id"`
	CarID      uuid.UUID    `json:"car_id"`
	UserID     uuid.UUID    `json:"user_id"`
	StartDate  time.Time    `json:"start_date"`
	EndDate    time.Time    `json:"end_date"`
	TotalPrice float64      `json:"total_price"`
	Status     RentalStatus `json:"status"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`
}

type RentalRequest struct {
	CarID     uuid.UUID `json:"car_id" binding:"required"`
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required,gtfield=StartDate"`
}

type RentalResponse struct {
	ID         uuid.UUID     `json:"id"`
	Car        *Car          `json:"car,omitempty"`
	User       *UserResponse `json:"user,omitempty"`
	StartDate  time.Time     `json:"start_date"`
	EndDate    time.Time     `json:"end_date"`
	TotalPrice float64       `json:"total_price"`
	Status     RentalStatus  `json:"status"`
	CreatedAt  time.Time     `json:"created_at"`
}

type ApproveRequest struct {
	Status RentalStatus `json:"status" binding:"required,oneof=active canceled"`
}

type ProfitAnalytics struct {
	TotalProfit      float64   `json:"total_profit"`
	CompletedRentals int64     `json:"completed_rentals"`
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
}
