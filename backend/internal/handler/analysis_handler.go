package handler

import (
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

// AnalysisHandler handles analysis-related requests
type AnalysisHandler struct {
	analysisService *service.AnalysisService
}

// NewAnalysisHandler creates a new analysis handler
func NewAnalysisHandler(analysisService *service.AnalysisService) *AnalysisHandler {
	return &AnalysisHandler{
		analysisService: analysisService,
	}
}

// GetSessionAnalysis handles GET /sessions/:sessionId/analysis
func (h *AnalysisHandler) GetSessionAnalysis(w http.ResponseWriter, r *http.Request) {
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

	analysis, err := h.analysisService.GetSessionAnalysis(r.Context(), userID, sessionID)
	if err != nil {
		switch err.Error() {
		case "session not found or access denied":
			h.errorResponse(w, r, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found", nil)
		case "no messages found for analysis":
			h.errorResponse(w, r, http.StatusBadRequest, "NO_MESSAGES", "No messages found for analysis", nil)
		default:
			h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get analysis", err)
		}
		return
	}

	response := types.AnalysisResponse{
		Analysis: analysis,
	}

	render.JSON(w, r, response)
}

// TriggerSessionAnalysis handles POST /sessions/:sessionId/analysis
func (h *AnalysisHandler) TriggerSessionAnalysis(w http.ResponseWriter, r *http.Request) {
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

	analysis, err := h.analysisService.AnalyzeSession(r.Context(), userID, sessionID)
	if err != nil {
		switch err.Error() {
		case "session not found or access denied":
			h.errorResponse(w, r, http.StatusNotFound, "SESSION_NOT_FOUND", "Session not found", nil)
		case "no messages found for analysis":
			h.errorResponse(w, r, http.StatusBadRequest, "NO_MESSAGES", "No messages found for analysis", nil)
		default:
			h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to perform analysis", err)
		}
		return
	}

	response := types.AnalysisResponse{
		Analysis: analysis,
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, response)
}

// GetTensionScores handles GET /analysis/scores
func (h *AnalysisHandler) GetTensionScores(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		fmt.Printf("DEBUG: Failed to get user ID from context: %v\n", err)
		h.errorResponse(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err)
		return
	}

	// Parse query parameters
	days := 30 // default
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if parsedDays, err := strconv.Atoi(daysStr); err == nil && parsedDays > 0 && parsedDays <= 365 {
			days = parsedDays
		}
	}

	fmt.Printf("DEBUG: Getting tension scores for userID: %s, days: %d\n", userID, days)
	scores, err := h.analysisService.GetTensionScores(r.Context(), userID, days)
	if err != nil {
		fmt.Printf("DEBUG: Failed to get tension scores: %v\n", err)
		h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get tension scores", err)
		return
	}

	fmt.Printf("DEBUG: Successfully retrieved tension scores: %+v\n", scores)
	render.JSON(w, r, scores)
}

// GetCalendarData handles GET /calendar/:year/:month
func (h *AnalysisHandler) GetCalendarData(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		h.errorResponse(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err)
		return
	}

	yearStr := chi.URLParam(r, "year")
	monthStr := chi.URLParam(r, "month")

	if yearStr == "" || monthStr == "" {
		h.errorResponse(w, r, http.StatusBadRequest, "MISSING_PARAMETERS", "Year and month are required", nil)
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2020 || year > timeutil.NowJST().Year()+1 {
		h.errorResponse(w, r, http.StatusBadRequest, "INVALID_YEAR", "Invalid year parameter", nil)
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		h.errorResponse(w, r, http.StatusBadRequest, "INVALID_MONTH", "Invalid month parameter", nil)
		return
	}

	calendarData, err := h.analysisService.GetCalendarData(r.Context(), userID, year, month)
	if err != nil {
		h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get calendar data", err)
		return
	}

	render.JSON(w, r, calendarData)
}

// GetUserAnalyses handles GET /analysis/history
func (h *AnalysisHandler) GetUserAnalyses(w http.ResponseWriter, r *http.Request) {
	_, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		h.errorResponse(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err)
		return
	}

	// Parse query parameters
	limit := 20 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			limit = parsedLimit
		}
	}

	offset := 0 // default
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Note: This would require implementing GetAnalysesByUserID in analysis service
	// For now, we'll return a placeholder response
	response := map[string]interface{}{
		"message": "Analysis history endpoint - implementation pending",
		"limit":   limit,
		"offset":  offset,
	}

	render.JSON(w, r, response)
}

// GetAnalysisInsights handles GET /analysis/insights
func (h *AnalysisHandler) GetAnalysisInsights(w http.ResponseWriter, r *http.Request) {
	userID, err := middleware.GetUserIDFromContext(r.Context())
	if err != nil {
		h.errorResponse(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated", err)
		return
	}

	// Parse query parameters for timeframe
	days := 7 // default
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if parsedDays, err := strconv.Atoi(daysStr); err == nil && parsedDays > 0 && parsedDays <= 90 {
			days = parsedDays
		}
	}

	// Get tension scores for insights
	tensionData, err := h.analysisService.GetTensionScores(r.Context(), userID, days)
	if err != nil {
		h.errorResponse(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get insights", err)
		return
	}

	// Generate insights based on the data
	insights := h.generateInsights(tensionData, days)

	response := map[string]interface{}{
		"insights":   insights,
		"timeframe":  days,
		"statistics": tensionData.Statistics,
	}

	render.JSON(w, r, response)
}

// Helper methods

// generateInsights creates user-friendly insights from tension data
func (h *AnalysisHandler) generateInsights(data *types.TensionScoresResponse, days int) []map[string]interface{} {
	insights := []map[string]interface{}{}

	// Average score insight
	if data.Statistics.Average > 0 {
		var level string
		var message string

		if data.Statistics.Average >= 70 {
			level = "high"
			message = "過去の期間で高いテンションレベルを維持しています"
		} else if data.Statistics.Average >= 50 {
			level = "medium"
			message = "バランスの取れたテンションレベルです"
		} else {
			level = "low"
			message = "最近のテンションレベルが低めです。リラックスや休息を心がけましょう"
		}

		insights = append(insights, map[string]interface{}{
			"type":    "average_score",
			"level":   level,
			"message": message,
			"value":   data.Statistics.Average,
		})
	}

	// Trend insight
	switch data.Statistics.Trend {
	case "improving":
		insights = append(insights, map[string]interface{}{
			"type":    "trend",
			"level":   "positive",
			"message": "テンションスコアが改善傾向にあります！",
			"value":   "improving",
		})
	case "declining":
		insights = append(insights, map[string]interface{}{
			"type":    "trend",
			"level":   "attention",
			"message": "最近テンションが下がり気味です。セルフケアを心がけましょう",
			"value":   "declining",
		})
	default:
		insights = append(insights, map[string]interface{}{
			"type":    "trend",
			"level":   "neutral",
			"message": "安定したテンションレベルを保っています",
			"value":   "stable",
		})
	}

	// Consistency insight
	scoreRange := data.Statistics.Max - data.Statistics.Min
	if scoreRange <= 20 {
		insights = append(insights, map[string]interface{}{
			"type":    "consistency",
			"level":   "positive",
			"message": "安定した気分を保てています",
			"value":   "stable",
		})
	} else if scoreRange > 40 {
		insights = append(insights, map[string]interface{}{
			"type":    "consistency",
			"level":   "attention",
			"message": "気分の変動が大きいようです。リラックス方法を見つけましょう",
			"value":   "variable",
		})
	}

	return insights
}

// errorResponse sends an error response
func (h *AnalysisHandler) errorResponse(w http.ResponseWriter, r *http.Request, status int, code, message string, err error) {
	render.Status(r, status)
	render.JSON(w, r, types.ErrorResponse{
		Error: types.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
