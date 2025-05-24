package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/trasta298/kasaneha/backend/internal/middleware"
	"github.com/trasta298/kasaneha/backend/internal/service"
	"github.com/trasta298/kasaneha/backend/internal/types"
	"github.com/trasta298/kasaneha/backend/pkg/timeutil"
)

// ChatHandler handles chat-related requests
type ChatHandler struct {
	chatService *service.ChatService
}

// NewChatHandler creates a new chat handler
func NewChatHandler(chatService *service.ChatService) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
	}
}

// GetTodaySession handles GET /sessions/today
func (h *ChatHandler) GetTodaySession(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		h.errorResponse(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err)
		return
	}

	session, initialMessage, err := h.chatService.GetTodaySession(r.Context(), userID)
	if err != nil {
		h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get today's session", err)
		return
	}

	response := map[string]interface{}{
		"session": session,
	}

	if initialMessage != nil {
		response["initial_message"] = initialMessage
	}

	render.JSON(w, r, response)
}

// CreateSession handles POST /sessions
func (h *ChatHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		h.errorResponse(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err)
		return
	}

	var req types.CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err)
		return
	}

	// Validate date format
	if _, err := timeutil.ParseDateInJST("2006-01-02", req.Date); err != nil {
		h.errorResponse(w, r, http.StatusBadRequest, "INVALID_DATE", "Invalid date format (expected YYYY-MM-DD)", err)
		return
	}

	session, err := h.chatService.CreateSessionForDate(r.Context(), userID, req.Date)
	if err != nil {
		if err.Error() == "session already exists" {
			h.errorResponse(w, r, http.StatusConflict, "SESSION_EXISTS", "Session already exists for this date", nil)
			return
		}
		h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create session", err)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, map[string]interface{}{
		"session": session,
	})
}

// GetSessionMessages handles GET /sessions/:sessionId/messages
func (h *ChatHandler) GetSessionMessages(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		h.errorResponse(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err)
		return
	}

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.errorResponse(w, r, http.StatusBadRequest, "MISSING_SESSION_ID", "Session ID is required", nil)
		return
	}

	messages, session, err := h.chatService.GetSessionMessages(r.Context(), userID, sessionID)
	if err != nil {
		if err.Error() == "session not found or access denied" {
			h.errorResponse(w, r, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found", nil)
			return
		}
		h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get messages", err)
		return
	}

	response := map[string]interface{}{
		"messages": messages,
		"session":  session,
	}

	render.JSON(w, r, response)
}

// SendMessage handles POST /sessions/:sessionId/messages
func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {

	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		fmt.Printf("DEBUG: Failed to get user ID from context: %v\n", err)
		h.errorResponse(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err)
		return
	}

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		fmt.Println("DEBUG: Session ID is empty")
		h.errorResponse(w, r, http.StatusBadRequest, "MISSING_SESSION_ID", "Session ID is required", nil)
		return
	}

	var req types.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, r, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", err)
		return
	}

	// Validate message content
	if len(req.Content) == 0 {
		fmt.Println("DEBUG: Message content is empty")
		h.errorResponse(w, r, http.StatusBadRequest, "EMPTY_CONTENT", "Message content cannot be empty", nil)
		return
	}

	if len(req.Content) > 2000 {
		h.errorResponse(w, r, http.StatusBadRequest, "CONTENT_TOO_LONG", "Message content too long (max 2000 characters)", nil)
		return
	}

	fmt.Println("DEBUG: Calling chatService.SendMessage")
	response, err := h.chatService.SendMessage(r.Context(), userID, sessionID, req.Content)
	if err != nil {
		switch err.Error() {
		case "session not found or access denied":
			h.errorResponse(w, r, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found", nil)
		case "session is not active":
			h.errorResponse(w, r, http.StatusBadRequest, "SESSION_INACTIVE", "Session is not active", nil)
		default:
			h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to send message", err)
		}
		return
	}

	render.JSON(w, r, response)
}

// CompleteSession handles PUT /sessions/:sessionId/complete
func (h *ChatHandler) CompleteSession(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		h.errorResponse(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err)
		return
	}

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.errorResponse(w, r, http.StatusBadRequest, "MISSING_SESSION_ID", "Session ID is required", nil)
		return
	}

	err = h.chatService.CompleteSession(r.Context(), userID, sessionID)
	if err != nil {
		switch err.Error() {
		case "session not found or access denied":
			h.errorResponse(w, r, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found", nil)
		case "session is already completed":
			h.errorResponse(w, r, http.StatusBadRequest, "SESSION_ALREADY_COMPLETED", "Session is already completed", nil)
		default:
			h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to complete session", err)
		}
		return
	}

	response := map[string]interface{}{
		"message":      "Session completed successfully",
		"session_id":   sessionID,
		"completed_at": timeutil.NowJST(),
	}

	render.JSON(w, r, response)
}

// GetUserSessions handles GET /sessions
func (h *ChatHandler) GetUserSessions(w http.ResponseWriter, r *http.Request) {
	fmt.Println("DEBUG: GetUserSessions called")

	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		fmt.Printf("DEBUG: Failed to get userID from context: %v\n", err)
		h.errorResponse(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err)
		return
	}
	fmt.Printf("DEBUG: UserID: %s\n", userID)

	// Parse query parameters
	limit := 20 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}
	fmt.Printf("DEBUG: Limit: %d\n", limit)

	offset := 0 // default
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}
	fmt.Printf("DEBUG: Offset: %d\n", offset)

	var year, month *int
	if yearStr := r.URL.Query().Get("year"); yearStr != "" {
		if parsedYear, err := strconv.Atoi(yearStr); err == nil {
			year = &parsedYear
		}
	}
	if year != nil {
		fmt.Printf("DEBUG: Year: %d\n", *year)
	} else {
		fmt.Println("DEBUG: Year: nil")
	}

	if monthStr := r.URL.Query().Get("month"); monthStr != "" {
		if parsedMonth, err := strconv.Atoi(monthStr); err == nil && parsedMonth >= 1 && parsedMonth <= 12 {
			month = &parsedMonth
		}
	}
	if month != nil {
		fmt.Printf("DEBUG: Month: %d\n", *month)
	} else {
		fmt.Println("DEBUG: Month: nil")
	}

	fmt.Println("DEBUG: Calling chatService.GetUserSessions")
	response, err := h.chatService.GetUserSessions(r.Context(), userID, limit, offset, year, month)
	if err != nil {
		fmt.Printf("DEBUG: GetUserSessions failed: %v\n", err)
		h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get sessions", err)
		return
	}

	fmt.Printf("DEBUG: GetUserSessions successful, response: %+v\n", response)
	render.JSON(w, r, response)
}

// GetSessionStats handles GET /sessions/:sessionId/stats
func (h *ChatHandler) GetSessionStats(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		h.errorResponse(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err)
		return
	}

	sessionID := chi.URLParam(r, "sessionId")
	if sessionID == "" {
		h.errorResponse(w, r, http.StatusBadRequest, "MISSING_SESSION_ID", "Session ID is required", nil)
		return
	}

	stats, err := h.chatService.GetSessionStats(r.Context(), userID, sessionID)
	if err != nil {
		if err.Error() == "session not found or access denied" {
			h.errorResponse(w, r, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found", nil)
			return
		}
		h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get session stats", err)
		return
	}

	render.JSON(w, r, stats)
}

// errorResponse sends an error response
func (h *ChatHandler) errorResponse(w http.ResponseWriter, r *http.Request, status int, code, message string, err error) {
	render.Status(r, status)
	render.JSON(w, r, types.ErrorResponse{
		Error: types.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
