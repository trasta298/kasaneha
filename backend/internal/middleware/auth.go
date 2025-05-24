package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/render"
	"github.com/golang-jwt/jwt/v5"
	"github.com/trasta298/kasaneha/backend/internal/types"
	"github.com/trasta298/kasaneha/backend/pkg/timeutil"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	jwtSecret []byte
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: []byte(jwtSecret),
	}
}

// UserClaims represents JWT claims for users
type UserClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// AuthenticateUser middleware that verifies JWT token
func (a *AuthMiddleware) AuthenticateUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			a.unauthorizedError(w, r, "missing authorization header")
			return
		}

		// Check if header starts with "Bearer "
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			a.unauthorizedError(w, r, "invalid authorization header format")
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Verify signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return a.jwtSecret, nil
		})

		if err != nil {
			a.unauthorizedError(w, r, fmt.Sprintf("invalid token: %v", err))
			return
		}

		// Extract claims
		claims, ok := token.Claims.(*UserClaims)
		if !ok || !token.Valid {
			a.unauthorizedError(w, r, "invalid token claims")
			return
		}

		// Add user info to request context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GenerateToken generates a JWT token for a user
func (a *AuthMiddleware) GenerateToken(userID, username string) (string, error) {
	claims := UserClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(timeutil.NowJST().Add(24 * time.Hour)), // 24 hours
			IssuedAt:  jwt.NewNumericDate(timeutil.NowJST()),
			Issuer:    "kasaneha",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// GetUserIDFromContext extracts user ID from request context
func GetUserIDFromContext(ctx context.Context) (string, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return "", fmt.Errorf("user_id not found in context")
	}
	return userID, nil
}

// GetUsernameFromContext extracts username from request context
func GetUsernameFromContext(ctx context.Context) (string, error) {
	username, ok := ctx.Value("username").(string)
	if !ok {
		return "", fmt.Errorf("username not found in context")
	}
	return username, nil
}

func (a *AuthMiddleware) unauthorizedError(w http.ResponseWriter, r *http.Request, message string) {
	w.WriteHeader(http.StatusUnauthorized)
	render.JSON(w, r, types.ErrorResponse{
		Error: types.ErrorDetail{
			Code:    "UNAUTHORIZED",
			Message: message,
		},
	})
}
