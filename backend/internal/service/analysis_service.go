package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/trasta298/kasaneha/backend/internal/ai"
	"github.com/trasta298/kasaneha/backend/internal/repository"
	"github.com/trasta298/kasaneha/backend/internal/types"
	"github.com/trasta298/kasaneha/backend/pkg/timeutil"
)

// AnalysisService handles analysis-related business logic
type AnalysisService struct {
	analysisRepo *repository.AnalysisRepository
	sessionRepo  *repository.SessionRepository
	messageRepo  *repository.MessageRepository
	userRepo     *repository.UserRepository
	aiClient     *ai.Client
}

// NewAnalysisService creates a new analysis service
func NewAnalysisService(
	analysisRepo *repository.AnalysisRepository,
	sessionRepo *repository.SessionRepository,
	messageRepo *repository.MessageRepository,
	userRepo *repository.UserRepository,
	aiClient *ai.Client,
) *AnalysisService {
	return &AnalysisService{
		analysisRepo: analysisRepo,
		sessionRepo:  sessionRepo,
		messageRepo:  messageRepo,
		userRepo:     userRepo,
		aiClient:     aiClient,
	}
}

// AnalyzeSession performs comprehensive analysis of a chat session
func (s *AnalysisService) AnalyzeSession(ctx context.Context, userID, sessionID string) (*types.Analysis, error) {
	fmt.Printf("DEBUG: Starting analysis for sessionID: %s\n", sessionID)

	// Verify session ownership
	isOwner, err := s.sessionRepo.CheckSessionOwnership(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check session ownership: %w", err)
	}
	if !isOwner {
		return nil, fmt.Errorf("session not found or access denied")
	}

	// Check if analysis already exists
	existingAnalysis, err := s.analysisRepo.GetAnalysisBySessionID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing analysis: %w", err)
	}
	if existingAnalysis != nil {
		fmt.Printf("DEBUG: Returning existing analysis for sessionID: %s\n", sessionID)
		return existingAnalysis, nil // Return existing analysis
	}

	// Get session information
	_, err = s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Get conversation log
	conversationLog, err := s.messageRepo.GetConversationLog(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation log: %w", err)
	}

	if conversationLog == "" {
		return nil, fmt.Errorf("no messages found for analysis")
	}

	// Perform emotion analysis
	fmt.Printf("DEBUG: Starting emotion analysis\n")
	emotionAnalysis, err := s.aiClient.AnalyzeEmotion(ctx, conversationLog)
	if err != nil {
		fmt.Printf("DEBUG: Emotion analysis failed: %v\n", err)
		return nil, fmt.Errorf("failed to analyze emotion: %w", err)
	}

	// Get historical data for tension score calculation
	historicalData, err := s.getHistoricalDataForUser(ctx, userID, 30) // Last 30 days
	if err != nil {
		return nil, fmt.Errorf("failed to get historical data: %w", err)
	}

	// Calculate tension score
	fmt.Printf("DEBUG: Starting tension score calculation\n")
	tensionScoreAnalysis, err := s.aiClient.CalculateTensionScore(ctx, emotionAnalysis, historicalData)
	if err != nil {
		fmt.Printf("DEBUG: Tension score calculation failed: %v\n", err)
		return nil, fmt.Errorf("failed to calculate tension score: %w", err)
	}

	// Generate summary and insights
	summary, behavioralInsights, err := s.generateAnalysisSummary(ctx, emotionAnalysis, tensionScoreAnalysis, conversationLog)
	if err != nil {
		return nil, fmt.Errorf("failed to generate summary: %w", err)
	}

	// Prepare analysis data
	analysis := &types.Analysis{
		SessionID:     sessionID,
		Summary:       summary,
		TensionScore:  tensionScoreAnalysis.TensionScore,
		RelativeScore: &tensionScoreAnalysis.RelativeScore,
	}

	// Convert emotion data to JSON
	emotionalStateJSON, err := json.Marshal(map[string]interface{}{
		"primary_emotion": emotionAnalysis.PrimaryEmotion,
		"emotions":        emotionAnalysis.Emotions,
		"confidence":      emotionAnalysis.Confidence,
		"explanation":     emotionAnalysis.Explanation,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal emotional state: %w", err)
	}
	analysis.EmotionalState = emotionalStateJSON

	// Convert behavioral insights to JSON
	behavioralInsightsJSON, err := json.Marshal(behavioralInsights)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal behavioral insights: %w", err)
	}
	analysis.BehavioralInsights = behavioralInsightsJSON

	// Extract and convert keywords to JSON
	keywordsJSON, err := json.Marshal(tensionScoreAnalysis.KeyFactors)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal keywords: %w", err)
	}
	analysis.Keywords = keywordsJSON

	// Store raw analysis data
	rawData := map[string]interface{}{
		"emotion_analysis":       emotionAnalysis,
		"tension_score_analysis": tensionScoreAnalysis,
		"analysis_timestamp":     timeutil.NowJST(),
		"ai_model_version":       "gemini-1.5",
	}
	rawDataJSON, err := json.Marshal(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw data: %w", err)
	}
	analysis.RawAnalysisData = rawDataJSON

	// Save analysis to database
	savedAnalysis, err := s.analysisRepo.CreateAnalysis(ctx, analysis)
	if err != nil {
		fmt.Printf("DEBUG: Failed to save analysis: %v\n", err)
		return nil, fmt.Errorf("failed to save analysis: %w", err)
	}

	fmt.Printf("DEBUG: Analysis completed successfully for sessionID: %s, tensionScore: %d\n", sessionID, tensionScoreAnalysis.TensionScore)
	return savedAnalysis, nil
}

// GetSessionAnalysis retrieves analysis for a specific session
func (s *AnalysisService) GetSessionAnalysis(ctx context.Context, userID, sessionID string) (*types.Analysis, error) {
	// Verify session ownership
	isOwner, err := s.sessionRepo.CheckSessionOwnership(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check session ownership: %w", err)
	}
	if !isOwner {
		return nil, fmt.Errorf("session not found or access denied")
	}

	analysis, err := s.analysisRepo.GetAnalysisBySessionID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis: %w", err)
	}

	if analysis == nil {
		// Analysis doesn't exist yet, trigger analysis
		return s.AnalyzeSession(ctx, userID, sessionID)
	}

	return analysis, nil
}

// GetTensionScores retrieves tension scores for a user
func (s *AnalysisService) GetTensionScores(ctx context.Context, userID string, days int) (*types.TensionScoresResponse, error) {
	endDate := timeutil.NowJST()
	startDate := endDate.AddDate(0, 0, -days)

	// Get tension scores
	scores, err := s.analysisRepo.GetTensionScores(ctx, userID, startDate, endDate, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get tension scores: %w", err)
	}

	// Get statistics
	statistics, err := s.analysisRepo.GetTensionStatistics(ctx, userID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get tension statistics: %w", err)
	}

	return &types.TensionScoresResponse{
		Scores:     scores,
		Statistics: *statistics,
	}, nil
}

// GetCalendarData retrieves calendar data for a specific month
func (s *AnalysisService) GetCalendarData(ctx context.Context, userID string, year, month int) (*types.CalendarResponse, error) {
	calendarDays, err := s.sessionRepo.GetCalendarData(ctx, userID, year, month)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar data: %w", err)
	}

	// Fill in missing days for the month
	daysInMonth := s.getDaysInMonth(year, month)
	calendarMap := make(map[string]types.CalendarDay)

	// Map existing data
	for _, day := range calendarDays {
		calendarMap[day.Date] = day
	}

	// Fill missing days
	var fullCalendar []types.CalendarDay
	for day := 1; day <= daysInMonth; day++ {
		dateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
		if calendarDay, exists := calendarMap[dateStr]; exists {
			fullCalendar = append(fullCalendar, calendarDay)
		} else {
			fullCalendar = append(fullCalendar, types.CalendarDay{
				Date:       dateStr,
				HasSession: false,
			})
		}
	}

	return &types.CalendarResponse{
		MonthData: types.CalendarMonthData{
			Year:  year,
			Month: month,
			Days:  fullCalendar,
		},
	}, nil
}

// TriggerAnalysisForCompletedSession automatically analyzes a session when it's completed
func (s *AnalysisService) TriggerAnalysisForCompletedSession(ctx context.Context, userID, sessionID string) error {
	// This can be called asynchronously after session completion
	go func() {
		analysisCtx := context.Background()
		_, err := s.AnalyzeSession(analysisCtx, userID, sessionID)
		if err != nil {
			// Log error but don't fail the main flow
			// In production, this would go to proper logging system
			fmt.Printf("Failed to analyze session %s: %v\n", sessionID, err)
		}
	}()

	return nil
}

// BatchAnalyzeActiveSessions automatically analyzes active sessions with minimum message count
func (s *AnalysisService) BatchAnalyzeActiveSessions(ctx context.Context, minMessages int) error {
	fmt.Printf("Starting batch analysis for active sessions with at least %d messages\n", minMessages)

	// Get active sessions that need analysis
	sessions, err := s.sessionRepo.GetActiveSessionsWithMinMessages(ctx, minMessages)
	if err != nil {
		return fmt.Errorf("failed to get active sessions: %w", err)
	}

	if len(sessions) == 0 {
		fmt.Println("No active sessions found for batch analysis")
		return nil
	}

	fmt.Printf("Found %d active sessions to analyze\n", len(sessions))

	// Analyze each session
	successCount := 0
	errorCount := 0

	for _, session := range sessions {
		fmt.Printf("Analyzing session %s for user %s (messages: %d)\n",
			session.ID, session.UserID, session.MessageCount)

		analysis, err := s.AnalyzeSession(ctx, session.UserID, session.ID)
		if err != nil {
			fmt.Printf("Failed to analyze session %s: %v\n", session.ID, err)
			errorCount++
			continue
		}

		// Complete the session after successful analysis (or if analysis already existed)
		err = s.sessionRepo.CompleteSession(ctx, session.ID)
		if err != nil {
			fmt.Printf("Warning: Failed to complete session %s after analysis: %v\n", session.ID, err)
			// Continue processing - analysis was successful, status update is not critical
		} else {
			fmt.Printf("Session %s marked as completed\n", session.ID)
		}

		successCount++
		if analysis != nil {
			fmt.Printf("Successfully processed session %s (tension score: %d)\n", session.ID, analysis.TensionScore)
		} else {
			fmt.Printf("Successfully processed session %s\n", session.ID)
		}

		// Add a small delay between analyses to avoid overwhelming the AI service
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("Batch analysis completed: %d successful, %d errors\n", successCount, errorCount)

	if errorCount > 0 {
		return fmt.Errorf("batch analysis completed with %d errors out of %d sessions", errorCount, len(sessions))
	}

	return nil
}

// Helper methods

// getHistoricalDataForUser retrieves historical analysis data for tension score context
func (s *AnalysisService) getHistoricalDataForUser(ctx context.Context, userID string, days int) (string, error) {
	endDate := timeutil.NowJST()
	startDate := endDate.AddDate(0, 0, -days)

	scores, err := s.analysisRepo.GetTensionScores(ctx, userID, startDate, endDate, days)
	if err != nil {
		return "", err
	}

	if len(scores) == 0 {
		return "ユーザーの履歴データがありません。", nil
	}

	// Format historical data for AI analysis
	historicalData := fmt.Sprintf("過去%d日間のテンションスコア履歴:\n", days)
	for _, score := range scores {
		historicalData += fmt.Sprintf("日付: %s, スコア: %d\n", score.Date, score.TensionScore)
	}

	return historicalData, nil
}

// generateAnalysisSummary creates a human-readable summary of the analysis
func (s *AnalysisService) generateAnalysisSummary(ctx context.Context, emotionAnalysis *ai.EmotionAnalysis, tensionAnalysis *ai.TensionScoreAnalysis, conversationLog string) (string, map[string]interface{}, error) {
	// Generate a summary based on the analysis results
	summary := fmt.Sprintf(
		"今日の対話では%sの感情が主に見られ、テンションスコアは%d点でした。%s",
		emotionAnalysis.PrimaryEmotion,
		tensionAnalysis.TensionScore,
		tensionAnalysis.Reasoning,
	)

	// Extract behavioral insights
	insights := map[string]interface{}{
		"conversation_patterns": s.analyzeConversationPatterns(conversationLog),
		"emotional_progression": s.analyzeEmotionalProgression(emotionAnalysis),
		"recommendations":       s.generateRecommendations(tensionAnalysis.TensionScore, emotionAnalysis.PrimaryEmotion),
	}

	return summary, insights, nil
}

func (s *AnalysisService) analyzeConversationPatterns(conversationLog string) map[string]interface{} {
	// Simple pattern analysis (could be enhanced with more sophisticated NLP)
	wordCount := len(conversationLog)
	patterns := map[string]interface{}{
		"total_length":        wordCount,
		"estimated_depth":     s.estimateConversationDepth(conversationLog),
		"communication_style": s.analyzeCommunicationStyle(conversationLog),
	}
	return patterns
}

func (s *AnalysisService) analyzeEmotionalProgression(emotionAnalysis *ai.EmotionAnalysis) map[string]interface{} {
	return map[string]interface{}{
		"primary_emotion":   emotionAnalysis.PrimaryEmotion,
		"confidence_level":  emotionAnalysis.Confidence,
		"emotional_balance": s.calculateEmotionalBalance(emotionAnalysis.Emotions),
	}
}

func (s *AnalysisService) generateRecommendations(tensionScore int, primaryEmotion string) []string {
	var recommendations []string

	if tensionScore < 30 {
		recommendations = append(recommendations, "リラックスできる活動を取り入れてみましょう")
		recommendations = append(recommendations, "十分な睡眠を心がけましょう")
	} else if tensionScore > 70 {
		recommendations = append(recommendations, "ポジティブな気持ちを維持しましょう")
		recommendations = append(recommendations, "この調子で過ごしましょう")
	} else {
		recommendations = append(recommendations, "バランスの取れた生活を続けましょう")
	}

	switch primaryEmotion {
	case "sadness":
		recommendations = append(recommendations, "友人や家族との時間を大切にしましょう")
	case "anger":
		recommendations = append(recommendations, "深呼吸やストレッチで気持ちを落ち着けましょう")
	case "happiness":
		recommendations = append(recommendations, "今の気持ちを大切にして過ごしましょう")
	}

	return recommendations
}

func (s *AnalysisService) estimateConversationDepth(conversationLog string) string {
	if len(conversationLog) > 1000 {
		return "深い"
	} else if len(conversationLog) > 500 {
		return "普通"
	}
	return "浅い"
}

func (s *AnalysisService) analyzeCommunicationStyle(conversationLog string) string {
	// Simple heuristic - could be enhanced
	if len(conversationLog) > 800 {
		return "詳細"
	}
	return "簡潔"
}

func (s *AnalysisService) calculateEmotionalBalance(emotions map[string]float64) string {
	positiveEmotions := emotions["happiness"] + emotions["surprise"]
	negativeEmotions := emotions["sadness"] + emotions["anger"] + emotions["fear"]

	if positiveEmotions > negativeEmotions*1.5 {
		return "ポジティブ"
	} else if negativeEmotions > positiveEmotions*1.5 {
		return "ネガティブ"
	}
	return "バランス"
}

func (s *AnalysisService) getDaysInMonth(year, month int) int {
	// Return the number of days in the given month
	t := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, timeutil.JST)
	return t.Day()
}
