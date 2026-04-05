package services

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"rent_project/internal/models"

	"github.com/google/uuid"
)

type RentalService struct {
	db         *sql.DB
	carService *CarService
}

func NewRentalService(db *sql.DB, carService *CarService) *RentalService {
	return &RentalService{
		db:         db,
		carService: carService,
	}
}

func (s *RentalService) CreateRental(userID uuid.UUID, req *models.RentalRequest) (*models.RentalResponse, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var car models.Car
	lockQuery := `
        SELECT id, model, brand, year, price_per_day, is_available, created_at, updated_at
        FROM cars
        WHERE id = $1
        FOR UPDATE
    `
	err = tx.QueryRow(lockQuery, req.CarID).Scan(
		&car.ID,
		&car.Model,
		&car.Brand,
		&car.Year,
		&car.PricePerDay,
		&car.IsAvailable,
		&car.CreatedAt,
		&car.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("car not found")
		}
		return nil, fmt.Errorf("failed to lock car: %w", err)
	}

	if !car.IsAvailable {
		return nil, errors.New("car is not available for rent")
	}

	isAvailable, err := s.isCarAvailableForDates(tx, req.CarID, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}
	if !isAvailable {
		return nil, errors.New("car is already booked for the selected dates")
	}

	days := req.EndDate.Sub(req.StartDate).Hours() / 24
	if days <= 0 {
		return nil, errors.New("end date must be after start date")
	}
	totalPrice := car.PricePerDay * days

	rentalID := uuid.New()
	insertQuery := `
        INSERT INTO rentals (id, car_id, user_id, start_date, end_date, total_price, status)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, car_id, user_id, start_date, end_date, total_price, status, created_at, updated_at
    `

	var rental models.Rental
	err = tx.QueryRow(
		insertQuery,
		rentalID,
		req.CarID,
		userID,
		req.StartDate,
		req.EndDate,
		totalPrice,
		models.RentalStatusPending,
	).Scan(
		&rental.ID,
		&rental.CarID,
		&rental.UserID,
		&rental.StartDate,
		&rental.EndDate,
		&rental.TotalPrice,
		&rental.Status,
		&rental.CreatedAt,
		&rental.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create rental: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s.getRentalWithDetails(rental.ID)
}

func (s *RentalService) isCarAvailableForDates(tx *sql.Tx, carID uuid.UUID, startDate, endDate time.Time) (bool, error) {
	query := `
        SELECT COUNT(*)
        FROM rentals
        WHERE car_id = $1
            AND status IN ('pending', 'active')
            AND daterange(start_date, end_date, '[]') && daterange($2, $3, '[]')
		FOR UPDATE
    `

	var count int
	err := tx.QueryRow(query, carID, startDate, endDate).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check availability: %w", err)
	}

	return count == 0, nil
}

func (s *RentalService) CancelRental(rentalID uuid.UUID, userID uuid.UUID) (*models.RentalResponse, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var rental models.Rental
	lockQuery := `
        SELECT id, car_id, user_id, start_date, end_date, total_price, status, created_at, updated_at
        FROM rentals
        WHERE id = $1
        FOR UPDATE
    `
	err = tx.QueryRow(lockQuery, rentalID).Scan(
		&rental.ID,
		&rental.CarID,
		&rental.UserID,
		&rental.StartDate,
		&rental.EndDate,
		&rental.TotalPrice,
		&rental.Status,
		&rental.CreatedAt,
		&rental.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("rental not found")
		}
		return nil, fmt.Errorf("failed to lock rental: %w", err)
	}

	if rental.UserID != userID {
		return nil, errors.New("you can only cancel your own rentals")
	}

	if rental.Status == models.RentalStatusCanceled {
		return nil, errors.New("rental is already canceled")
	}

	if rental.Status == models.RentalStatusCompleted {
		return nil, errors.New("cannot cancel completed rental")
	}

	updateQuery := `
        UPDATE rentals
        SET status = $1, updated_at = $2
        WHERE id = $3
        RETURNING id, car_id, user_id, start_date, end_date, total_price, status, created_at, updated_at
    `

	err = tx.QueryRow(
		updateQuery,
		models.RentalStatusCanceled,
		time.Now(),
		rentalID,
	).Scan(
		&rental.ID,
		&rental.CarID,
		&rental.UserID,
		&rental.StartDate,
		&rental.EndDate,
		&rental.TotalPrice,
		&rental.Status,
		&rental.CreatedAt,
		&rental.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to cancel rental: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s.getRentalWithDetails(rental.ID)
}

func (s *RentalService) ApproveRental(rentalID uuid.UUID, status models.RentalStatus) (*models.RentalResponse, error) {
	if status != models.RentalStatusActive && status != models.RentalStatusCanceled {
		return nil, errors.New("status must be 'active' or 'canceled'")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var rental models.Rental
	lockQuery := `
        SELECT id, car_id, user_id, start_date, end_date, total_price, status, created_at, updated_at
        FROM rentals
        WHERE id = $1
        FOR UPDATE
    `
	err = tx.QueryRow(lockQuery, rentalID).Scan(
		&rental.ID,
		&rental.CarID,
		&rental.UserID,
		&rental.StartDate,
		&rental.EndDate,
		&rental.TotalPrice,
		&rental.Status,
		&rental.CreatedAt,
		&rental.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("rental not found")
		}
		return nil, fmt.Errorf("failed to lock rental: %w", err)
	}

	if rental.Status != models.RentalStatusPending {
		return nil, fmt.Errorf("cannot approve rental with status: %s", rental.Status)
	}

	if status == models.RentalStatusActive {
		var car models.Car
		carLockQuery := `
            SELECT id, is_available
            FROM cars
            WHERE id = $1
            FOR UPDATE
        `
		err = tx.QueryRow(carLockQuery, rental.CarID).Scan(&car.ID, &car.IsAvailable)
		if err != nil {
			return nil, fmt.Errorf("failed to lock car: %w", err)
		}

		if !car.IsAvailable {
			return nil, errors.New("car is no longer available")
		}

		isAvailable, err := s.isCarAvailableForDates(tx, rental.CarID, rental.StartDate, rental.EndDate)
		if err != nil {
			return nil, err
		}
		if !isAvailable {
			return nil, errors.New("car is already booked for these dates")
		}
	}

	updateQuery := `
        UPDATE rentals
        SET status = $1, updated_at = $2
        WHERE id = $3
        RETURNING id, car_id, user_id, start_date, end_date, total_price, status, created_at, updated_at
    `

	err = tx.QueryRow(
		updateQuery,
		status,
		time.Now(),
		rentalID,
	).Scan(
		&rental.ID,
		&rental.CarID,
		&rental.UserID,
		&rental.StartDate,
		&rental.EndDate,
		&rental.TotalPrice,
		&rental.Status,
		&rental.CreatedAt,
		&rental.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update rental status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return s.getRentalWithDetails(rental.ID)
}

func (s *RentalService) GetUserRentals(userID uuid.UUID, limit, offset int) ([]*models.RentalResponse, int, error) {
	var total int
	countQuery := "SELECT COUNT(*) FROM rentals WHERE user_id = $1"
	err := s.db.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count rentals: %w", err)
	}

	query := `
        SELECT id, car_id, user_id, start_date, end_date, total_price, status, created_at, updated_at
        FROM rentals
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3
    `

	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query rentals: %w", err)
	}
	defer rows.Close()

	var rentals []*models.RentalResponse
	for rows.Next() {
		var rental models.Rental
		err := rows.Scan(
			&rental.ID,
			&rental.CarID,
			&rental.UserID,
			&rental.StartDate,
			&rental.EndDate,
			&rental.TotalPrice,
			&rental.Status,
			&rental.CreatedAt,
			&rental.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan rental: %w", err)
		}

		rentalResp, err := s.getRentalWithDetails(rental.ID)
		if err != nil {
			return nil, 0, err
		}
		rentals = append(rentals, rentalResp)
	}

	return rentals, total, nil
}

func (s *RentalService) GetAllRentals(status string, limit, offset int) ([]*models.RentalResponse, int, error) {
	countQuery := "SELECT COUNT(*) FROM rentals"
	query := `
        SELECT id, car_id, user_id, start_date, end_date, total_price, status, created_at, updated_at
        FROM rentals
    `

	args := []interface{}{}
	argIndex := 1

	if status != "" {
		countQuery += " WHERE status = $" + fmt.Sprintf("%d", argIndex)
		query += " WHERE status = $" + fmt.Sprintf("%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	var total int
	err := s.db.QueryRow(countQuery, args[:len(args)-2]...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count rentals: %w", err)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query rentals: %w", err)
	}
	defer rows.Close()

	var rentals []*models.RentalResponse
	for rows.Next() {
		var rental models.Rental
		err := rows.Scan(
			&rental.ID,
			&rental.CarID,
			&rental.UserID,
			&rental.StartDate,
			&rental.EndDate,
			&rental.TotalPrice,
			&rental.Status,
			&rental.CreatedAt,
			&rental.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan rental: %w", err)
		}

		rentalResp, err := s.getRentalWithDetails(rental.ID)
		if err != nil {
			return nil, 0, err
		}
		rentals = append(rentals, rentalResp)
	}

	return rentals, total, nil
}

func (s *RentalService) getRentalWithDetails(rentalID uuid.UUID) (*models.RentalResponse, error) {
	query := `
        SELECT 
            r.id, r.car_id, r.user_id, r.start_date, r.end_date, r.total_price, r.status, r.created_at, r.updated_at,
            c.id, c.model, c.brand, c.year, c.price_per_day, c.is_available,
            u.id, u.name, u.email, u.role
        FROM rentals r
        JOIN cars c ON r.car_id = c.id
        JOIN users u ON r.user_id = u.id
        WHERE r.id = $1
    `

	var rental models.Rental
	var car models.Car
	var user models.User

	err := s.db.QueryRow(query, rentalID).Scan(
		&rental.ID, &rental.CarID, &rental.UserID, &rental.StartDate, &rental.EndDate,
		&rental.TotalPrice, &rental.Status, &rental.CreatedAt, &rental.UpdatedAt,
		&car.ID, &car.Model, &car.Brand, &car.Year, &car.PricePerDay, &car.IsAvailable,
		&user.ID, &user.Name, &user.Email, &user.Role,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("rental not found")
		}
		return nil, fmt.Errorf("failed to get rental details: %w", err)
	}

	return &models.RentalResponse{
		ID:         rental.ID,
		Car:        &car,
		User:       user.ToResponse(),
		StartDate:  rental.StartDate,
		EndDate:    rental.EndDate,
		TotalPrice: rental.TotalPrice,
		Status:     rental.Status,
		CreatedAt:  rental.CreatedAt,
	}, nil
}
