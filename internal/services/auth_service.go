package services

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"rent_project/internal/models"
	"rent_project/internal/utils"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db         *sql.DB
	jwtService *utils.JWTService
}

func NewAuthService(db *sql.DB, jwtService *utils.JWTService) *AuthService {
	return &AuthService{
		db:         db,
		jwtService: jwtService,
	}
}

func (s *AuthService) Register(req *models.RegisterRequest) (*models.UserResponse, string, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)"
	err := s.db.QueryRow(query, req.Email).Scan(&exists)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return nil, "", errors.New("user with this email already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash password: %w", err)
	}

	userID := uuid.New()
	query = `
        INSERT INTO users (id, name, email, password_hash, role)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, name, email, role, created_at, updated_at
    `

	var user models.User
	err = s.db.QueryRow(
		query,
		userID,
		req.Name,
		req.Email,
		string(hashedPassword),
		models.RoleClient,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, "", fmt.Errorf("failed to create user: %w", err)
	}

	token, err := s.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return user.ToResponse(), token, nil
}

func (s *AuthService) Login(req *models.LoginRequest) (*models.UserResponse, string, error) {
	var user models.User

	query := `
        SELECT id, name, email, password_hash, role, created_at, updated_at
        FROM users
        WHERE email = $1
    `
	err := s.db.QueryRow(query, req.Email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", errors.New("invalid credentials")
		}
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, "", errors.New("invalid credentials")
	}

	token, err := s.jwtService.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return user.ToResponse(), token, nil
}

func (s *AuthService) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	query := `
        SELECT id, name, email, password_hash, role, created_at, updated_at
        FROM users
        WHERE id = $1
    `
	err := s.db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (s *AuthService) UpdateUserRole(userID uuid.UUID, role models.UserRole) error {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)"
	err := s.db.QueryRow(query, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if !exists {
		return errors.New("user not found")
	}

	query = "UPDATE users SET role = $1, updated_at = $2 WHERE id = $3"
	_, err = s.db.Exec(query, role, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	return nil
}
