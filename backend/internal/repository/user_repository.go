package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/trasta298/kasaneha/backend/internal/types"
	"github.com/trasta298/kasaneha/backend/pkg/timeutil"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository handles user data operations
type UserRepository struct {
	db *Database
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *Database) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser creates a new user
func (r *UserRepository) CreateUser(ctx context.Context, req *types.RegisterRequest) (*types.User, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	query := `
		INSERT INTO users (username, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, username, email, created_at, updated_at, last_login_at, is_active, timezone
	`

	var user types.User
	row := r.db.Pool.QueryRow(ctx, query, req.Username, req.Email, string(hashedPassword))

	err = row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.IsActive,
		&user.Timezone,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*types.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at, updated_at, 
		       last_login_at, is_active, timezone
		FROM users
		WHERE username = $1 AND is_active = true
	`

	var user types.User
	row := r.db.Pool.QueryRow(ctx, query, username)

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.IsActive,
		&user.Timezone,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *UserRepository) GetUserByID(ctx context.Context, userID string) (*types.User, error) {
	query := `
		SELECT id, username, email, created_at, updated_at, 
		       last_login_at, is_active, timezone
		FROM users
		WHERE id = $1 AND is_active = true
	`

	var user types.User
	row := r.db.Pool.QueryRow(ctx, query, userID)

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
		&user.IsActive,
		&user.Timezone,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	query := `
		UPDATE users 
		SET last_login_at = $1
		WHERE id = $2
	`

	_, err := r.db.Pool.Exec(ctx, query, timeutil.NowJST(), userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// ValidatePassword validates a user's password
func (r *UserRepository) ValidatePassword(ctx context.Context, username, password string) (*types.User, error) {
	user, err := r.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	// Compare password with hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

// UpdateUser updates user information
func (r *UserRepository) UpdateUser(ctx context.Context, userID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic query
	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)
	argIndex := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	query := fmt.Sprintf(`
		UPDATE users 
		SET %s, updated_at = CURRENT_TIMESTAMP
		WHERE id = $%d
	`, fmt.Sprintf("%s", setParts), argIndex)

	args = append(args, userID)

	_, err := r.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// DeactivateUser soft deletes a user
func (r *UserRepository) DeactivateUser(ctx context.Context, userID string) error {
	query := `
		UPDATE users 
		SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	return nil
}
