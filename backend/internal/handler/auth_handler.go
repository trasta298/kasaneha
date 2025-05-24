package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/render"
	"github.com/trasta298/kasaneha/backend/internal/middleware"
	"github.com/trasta298/kasaneha/backend/internal/repository"
	"github.com/trasta298/kasaneha/backend/internal/types"
)

// AuthHandler handles authentication related requests
type AuthHandler struct {
	userRepo *repository.UserRepository
	auth     *middleware.AuthMiddleware
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userRepo *repository.UserRepository, auth *middleware.AuthMiddleware) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
		auth:     auth,
	}
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req types.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err)
		return
	}

	// Validate request
	if err := h.validateRegisterRequest(&req); err != nil {
		h.errorResponse(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}

	// Create user
	user, err := h.userRepo.CreateUser(r.Context(), &req)
	if err != nil {
		// Check if it's a duplicate username/email error
		if isUniqueConstraintError(err) {
			h.errorResponse(w, r, http.StatusConflict, "USER_EXISTS", "Username or email already exists", nil)
			return
		}
		h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create user", err)
		return
	}

	// Generate JWT token
	token, err := h.auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token", err)
		return
	}

	// Return response
	response := types.LoginResponse{
		Token: token,
		User:  *user,
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, response)
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req types.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err)
		return
	}

	// Validate password
	user, err := h.userRepo.ValidatePassword(r.Context(), req.Username, req.Password)
	if err != nil {
		h.errorResponse(w, r, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password", nil)
		return
	}

	// Update last login
	if err := h.userRepo.UpdateLastLogin(r.Context(), user.ID); err != nil {
		// Log error but don't fail the login
		// In a real app, you might want to use a proper logger here
	}

	// Generate JWT token
	token, err := h.auth.GenerateToken(user.ID, user.Username)
	if err != nil {
		h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate token", err)
		return
	}

	// Return response
	response := types.LoginResponse{
		Token: token,
		User:  *user,
	}

	render.JSON(w, r, response)
}

// Me returns current user information
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		h.errorResponse(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err)
		return
	}

	user, err := h.userRepo.GetUserByID(r.Context(), userID)
	if err != nil {
		h.errorResponse(w, r, http.StatusNotFound, "USER_NOT_FOUND", "User not found", err)
		return
	}

	render.JSON(w, r, user)
}

// validateRegisterRequest validates registration request
func (h *AuthHandler) validateRegisterRequest(req *types.RegisterRequest) error {
	if len(req.Username) < 3 {
		return fmt.Errorf("username must be at least 3 characters long")
	}
	if len(req.Password) < 6 {
		return fmt.Errorf("password must be at least 6 characters long")
	}
	if req.Email != nil && *req.Email != "" {
		// Simple email validation
		if !strings.Contains(*req.Email, "@") {
			return fmt.Errorf("invalid email format")
		}
	}
	return nil
}

// errorResponse sends an error response
func (h *AuthHandler) errorResponse(w http.ResponseWriter, r *http.Request, status int, code, message string, err error) {
	render.Status(r, status)
	render.JSON(w, r, types.ErrorResponse{
		Error: types.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// isUniqueConstraintError checks if the error is a unique constraint violation
func isUniqueConstraintError(err error) bool {
	// This is a simplified check. In a real application, you'd want to
	// check for specific PostgreSQL error codes (23505 for unique_violation)
	return strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate")
}
