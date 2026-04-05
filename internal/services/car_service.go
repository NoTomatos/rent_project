package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"rent_project/internal/models"

	"github.com/google/uuid"
)

type CarService struct {
	db *sql.DB
}

func NewCarService(db *sql.DB) *CarService {
	return &CarService{
		db: db,
	}
}

func (s *CarService) GetAllCars(req *models.CarListRequest) ([]*models.Car, int, error) {
	query := `
        SELECT id, model, brand, year, price_per_day, is_available, created_at, updated_at
        FROM cars
        WHERE 1=1
    `
	countQuery := "SELECT COUNT(*) FROM cars WHERE 1=1"

	var args []interface{}
	var countArgs []interface{}
	argIndex := 1

	if req.Available != nil {
		query += fmt.Sprintf(" AND is_available = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND is_available = $%d", argIndex)
		args = append(args, *req.Available)
		countArgs = append(countArgs, *req.Available)
		argIndex++
	}

	if req.Brand != "" {
		query += fmt.Sprintf(" AND LOWER(brand) = $%d", argIndex)
		countQuery += fmt.Sprintf(" AND LOWER(brand) = $%d", argIndex)
		args = append(args, strings.ToLower(req.Brand))
		countArgs = append(countArgs, strings.ToLower(req.Brand))
		argIndex++
	}

	if req.Model != "" {
		query += fmt.Sprintf(" AND LOWER(model) LIKE $%d", argIndex)
		countQuery += fmt.Sprintf(" AND LOWER(model) LIKE $%d", argIndex)
		args = append(args, "%"+strings.ToLower(req.Model)+"%")
		countArgs = append(countArgs, "%"+strings.ToLower(req.Model)+"%")
		argIndex++
	}

	if req.MinPrice > 0 {
		query += fmt.Sprintf(" AND price_per_day >= $%d", argIndex)
		countQuery += fmt.Sprintf(" AND price_per_day >= $%d", argIndex)
		args = append(args, req.MinPrice)
		countArgs = append(countArgs, req.MinPrice)
		argIndex++
	}

	if req.MaxPrice > 0 {
		query += fmt.Sprintf(" AND price_per_day <= $%d", argIndex)
		countQuery += fmt.Sprintf(" AND price_per_day <= $%d", argIndex)
		args = append(args, req.MaxPrice)
		countArgs = append(countArgs, req.MaxPrice)
		argIndex++
	}

	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	var total int
	err := s.db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count cars: %w", err)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query cars: %w", err)
	}
	defer rows.Close()

	var cars []*models.Car
	for rows.Next() {
		var car models.Car
		err := rows.Scan(
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
			return nil, 0, fmt.Errorf("failed to scan car: %w", err)
		}
		cars = append(cars, &car)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows iteration error: %w", err)
	}

	return cars, total, nil
}

func (s *CarService) GetCarByID(id uuid.UUID) (*models.Car, error) {
	var car models.Car
	query := `
        SELECT id, model, brand, year, price_per_day, is_available, created_at, updated_at
        FROM cars
        WHERE id = $1
    `

	err := s.db.QueryRow(query, id).Scan(
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
		return nil, fmt.Errorf("failed to get car: %w", err)
	}

	return &car, nil
}

func (s *CarService) CreateCar(req *models.CarRequest) (*models.Car, error) {
	if req.Year < 1900 || req.Year > 2030 {
		return nil, errors.New("invalid year: must be between 1900 and 2030")
	}

	if req.PricePerDay <= 0 {
		return nil, errors.New("price per day must be greater than 0")
	}

	carID := uuid.New()
	query := `
        INSERT INTO cars (id, model, brand, year, price_per_day, is_available)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, model, brand, year, price_per_day, is_available, created_at, updated_at
    `

	var car models.Car
	err := s.db.QueryRow(
		query,
		carID,
		req.Model,
		req.Brand,
		req.Year,
		req.PricePerDay,
		true,
	).Scan(
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
		return nil, fmt.Errorf("failed to create car: %w", err)
	}

	return &car, nil
}

func (s *CarService) UpdateCar(id uuid.UUID, req *models.CarRequest) (*models.Car, error) {
	exists, err := s.carExists(id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("car not found")
	}

	if req.Year < 1900 || req.Year > 2030 {
		return nil, errors.New("invalid year: must be between 1900 and 2030")
	}
	if req.PricePerDay <= 0 {
		return nil, errors.New("price per day must be greater than 0")
	}

	query := `
        UPDATE cars
        SET model = $1, brand = $2, year = $3, price_per_day = $4
        WHERE id = $5
        RETURNING id, model, brand, year, price_per_day, is_available, created_at, updated_at
    `

	var car models.Car
	err = s.db.QueryRow(
		query,
		req.Model,
		req.Brand,
		req.Year,
		req.PricePerDay,
		id,
	).Scan(
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
		return nil, fmt.Errorf("failed to update car: %w", err)
	}

	return &car, nil
}

func (s *CarService) DeleteCar(id uuid.UUID) error {
	exists, err := s.carExists(id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("car not found")
	}

	var activeRentals int
	query := `
        SELECT COUNT(*)
        FROM rentals
        WHERE car_id = $1 AND status IN ('pending', 'active')
    `
	err = s.db.QueryRow(query, id).Scan(&activeRentals)
	if err != nil {
		return fmt.Errorf("failed to check active rentals: %w", err)
	}

	if activeRentals > 0 {
		return errors.New("cannot delete car with active or pending rentals")
	}

	_, err = s.db.Exec("DELETE FROM cars WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete car: %w", err)
	}

	return nil
}

func (s *CarService) UpdateCarAvailability(id uuid.UUID, isAvailable bool) error {
	exists, err := s.carExists(id)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("car not found")
	}

	query := "UPDATE cars SET is_available = $1 WHERE id = $2"
	_, err = s.db.Exec(query, isAvailable, id)
	if err != nil {
		return fmt.Errorf("failed to update car availability: %w", err)
	}

	return nil
}

func (s *CarService) carExists(id uuid.UUID) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM cars WHERE id = $1)"
	err := s.db.QueryRow(query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check car existence: %w", err)
	}
	return exists, nil
}
