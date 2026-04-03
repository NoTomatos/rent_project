package services

import (
	"database/sql"
	"fmt"
	"time"

	"rent_project/internal/models"
)

type AnalyticsService struct {
	db *sql.DB
}

func NewAnalyticsService(db *sql.DB) *AnalyticsService {
	return &AnalyticsService{
		db: db,
	}
}

func (s *AnalyticsService) GetProfitAnalytics(startDate, endDate time.Time) (*models.ProfitAnalytics, error) {
	// Валидация дат
	if startDate.After(endDate) {
		return nil, fmt.Errorf("start date must be before end date")
	}

	query := `
        SELECT 
            COALESCE(SUM(total_price), 0) as total_profit,
            COUNT(*) as completed_rentals
        FROM rentals
        WHERE status = 'completed'
            AND updated_at BETWEEN $1 AND $2
    `

	var analytics models.ProfitAnalytics
	err := s.db.QueryRow(query, startDate, endDate).Scan(
		&analytics.TotalProfit,
		&analytics.CompletedRentals,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get profit analytics: %w", err)
	}

	analytics.StartDate = startDate
	analytics.EndDate = endDate

	return &analytics, nil
}

func (s *AnalyticsService) GetPopularBrands(limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	query := `
        SELECT 
            c.brand,
            COUNT(r.id) as rental_count,
            SUM(r.total_price) as total_revenue
        FROM rentals r
        JOIN cars c ON r.car_id = c.id
        WHERE r.status IN ('active', 'completed')
        GROUP BY c.brand
        ORDER BY rental_count DESC
        LIMIT $1
    `

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular brands: %w", err)
	}
	defer rows.Close()

	var brands []map[string]interface{}
	for rows.Next() {
		var brand string
		var rentalCount int64
		var totalRevenue float64

		err := rows.Scan(&brand, &rentalCount, &totalRevenue)
		if err != nil {
			return nil, fmt.Errorf("failed to scan brand stats: %w", err)
		}

		brands = append(brands, map[string]interface{}{
			"brand":         brand,
			"rental_count":  rentalCount,
			"total_revenue": totalRevenue,
		})
	}

	return brands, nil
}

func (s *AnalyticsService) GetDailyProfit(startDate, endDate time.Time) ([]map[string]interface{}, error) {
	if startDate.After(endDate) {
		return nil, fmt.Errorf("start date must be before end date")
	}

	query := `
        SELECT 
            DATE(updated_at) as date,
            COALESCE(SUM(total_price), 0) as daily_profit,
            COUNT(*) as rentals_count
        FROM rentals
        WHERE status = 'completed'
            AND updated_at BETWEEN $1 AND $2
        GROUP BY DATE(updated_at)
        ORDER BY date ASC
    `

	rows, err := s.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily profit: %w", err)
	}
	defer rows.Close()

	var dailyStats []map[string]interface{}
	for rows.Next() {
		var date time.Time
		var dailyProfit float64
		var rentalsCount int64

		err := rows.Scan(&date, &dailyProfit, &rentalsCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily stats: %w", err)
		}

		dailyStats = append(dailyStats, map[string]interface{}{
			"date":          date.Format("2006-01-02"),
			"daily_profit":  dailyProfit,
			"rentals_count": rentalsCount,
		})
	}

	return dailyStats, nil
}

func (s *AnalyticsService) GetUserStatistics() (map[string]interface{}, error) {
	query := `
        SELECT 
            COUNT(DISTINCT id) as total_users,
            COUNT(DISTINCT CASE WHEN role = 'admin' THEN id END) as admins,
            COUNT(DISTINCT CASE WHEN role = 'manager' THEN id END) as managers,
            COUNT(DISTINCT CASE WHEN role = 'client' THEN id END) as clients,
            COUNT(DISTINCT r.id) as total_rentals,
            COUNT(DISTINCT CASE WHEN r.status = 'active' THEN r.id END) as active_rentals,
            COUNT(DISTINCT CASE WHEN r.status = 'completed' THEN r.id END) as completed_rentals
        FROM users u
        LEFT JOIN rentals r ON u.id = r.user_id
    `

	var stats map[string]interface{}
	var totalUsers, admins, managers, clients int64
	var totalRentals, activeRentals, completedRentals int64

	err := s.db.QueryRow(query).Scan(
		&totalUsers, &admins, &managers, &clients,
		&totalRentals, &activeRentals, &completedRentals,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user statistics: %w", err)
	}

	stats = map[string]interface{}{
		"users": map[string]interface{}{
			"total":    totalUsers,
			"admins":   admins,
			"managers": managers,
			"clients":  clients,
		},
		"rentals": map[string]interface{}{
			"total":     totalRentals,
			"active":    activeRentals,
			"completed": completedRentals,
		},
	}

	return stats, nil
}

func (s *AnalyticsService) GetCarUtilization() ([]map[string]interface{}, error) {
	query := `
        SELECT 
            c.id,
            c.brand,
            c.model,
            c.price_per_day,
            COUNT(r.id) as total_rentals,
            COALESCE(SUM(r.total_price), 0) as total_revenue,
            AVG(EXTRACT(DAY FROM (r.end_date - r.start_date))) as avg_rental_days
        FROM cars c
        LEFT JOIN rentals r ON c.id = r.car_id AND r.status IN ('active', 'completed')
        GROUP BY c.id, c.brand, c.model, c.price_per_day
        ORDER BY total_rentals DESC
    `

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get car utilization: %w", err)
	}
	defer rows.Close()

	var utilization []map[string]interface{}
	for rows.Next() {
		var id string
		var brand, model string
		var pricePerDay float64
		var totalRentals int64
		var totalRevenue float64
		var avgRentalDays sql.NullFloat64

		err := rows.Scan(&id, &brand, &model, &pricePerDay, &totalRentals, &totalRevenue, &avgRentalDays)
		if err != nil {
			return nil, fmt.Errorf("failed to scan car utilization: %w", err)
		}

		avgDays := 0.0
		if avgRentalDays.Valid {
			avgDays = avgRentalDays.Float64
		}

		utilization = append(utilization, map[string]interface{}{
			"car_id":          id,
			"brand":           brand,
			"model":           model,
			"price_per_day":   pricePerDay,
			"total_rentals":   totalRentals,
			"total_revenue":   totalRevenue,
			"avg_rental_days": avgDays,
		})
	}

	return utilization, nil
}
