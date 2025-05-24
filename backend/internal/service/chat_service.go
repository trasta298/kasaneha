package service

import (
	"context"
	"fmt"

	"github.com/trasta298/kasaneha/backend/internal/ai"
	"github.com/trasta298/kasaneha/backend/internal/repository"
	"github.com/trasta298/kasaneha/backend/internal/types"
	"github.com/trasta298/kasaneha/backend/pkg/timeutil"
)

// ChatService handles chat-related business logic
type ChatService struct {
	sessionRepo     *repository.SessionRepository
	messageRepo     *repository.MessageRepository
	userRepo        *repository.UserRepository
	aiClient        *ai.Client
	analysisService *AnalysisService
}

// NewChatService creates a new chat service
func NewChatService(
	sessionRepo *repository.SessionRepository,
	messageRepo *repository.MessageRepository,
	userRepo *repository.UserRepository,
	aiClient *ai.Client,
) *ChatService {
	return &ChatService{
		sessionRepo:     sessionRepo,
		messageRepo:     messageRepo,
		userRepo:        userRepo,
		aiClient:        aiClient,
		analysisService: nil, // Will be set after initialization
	}
}

// SetAnalysisService sets the analysis service (to avoid circular dependency)
func (s *ChatService) SetAnalysisService(analysisService *AnalysisService) {
	s.analysisService = analysisService
}

// GetTodaySession retrieves or creates today's session for a user
func (s *ChatService) GetTodaySession(ctx context.Context, userID string) (*types.ChatSession, *types.Message, error) {
	// Check if today's session already exists
	session, err := s.sessionRepo.GetTodaySession(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get today's session: %w", err)
	}

	if session != nil {
		// Session exists, return it without initial message
		return session, nil, nil
	}

	// No session exists, create a new one
	today := timeutil.TodayJST()
	session, err = s.sessionRepo.CreateSession(ctx, userID, today)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Generate initial AI message
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Determine time of day
	timeOfDay := s.getTimeOfDay()

	// Generate first message from AI
	aiResponse, err := s.aiClient.GenerateFirstMessage(ctx, user.Username, today, timeOfDay)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate first message: %w", err)
	}

	// Save the initial AI message
	initialMessage, err := s.messageRepo.CreateMessage(
		ctx,
		session.ID,
		types.SenderAI,
		aiResponse.Content,
		nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to save initial message: %w", err)
	}

	return session, initialMessage, nil
}

// SendMessage sends a user message and generates AI response
func (s *ChatService) SendMessage(ctx context.Context, userID, sessionID, content string) (*types.SendMessageResponse, error) {
	// Verify session ownership
	isOwner, err := s.sessionRepo.CheckSessionOwnership(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check session ownership: %w", err)
	}
	if !isOwner {
		return nil, fmt.Errorf("session not found or access denied")
	}

	// Get session to check status
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != types.SessionStatusActive {
		return nil, fmt.Errorf("session is not active")
	}

	// Save user message
	userMessage, err := s.messageRepo.CreateMessage(
		ctx,
		sessionID,
		types.SenderUser,
		content,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// Get recent conversation history for context
	recentMessages, err := s.messageRepo.GetLatestMessages(ctx, sessionID, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation history: %w", err)
	}

	// Convert to AI message format
	var conversationHistory []ai.Message
	for _, msg := range recentMessages {
		if msg.ID == userMessage.ID {
			continue // Skip the message we just added
		}
		conversationHistory = append(conversationHistory, ai.Message{
			Content: msg.Content,
			Sender:  msg.Sender,
		})
	}

	// Get user info for personalization
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Generate AI response
	aiRequest := ai.ConversationRequest{
		UserMessage:         content,
		ConversationHistory: conversationHistory,
		Date:                timeutil.FormatJST(session.SessionDate, "2006-01-02"),
		TimeOfDay:           s.getTimeOfDay(),
		UserName:            user.Username,
	}

	aiResponse, err := s.aiClient.GenerateResponse(ctx, aiRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to generate AI response: %w", err)
	}

	// Save AI response
	aiMessage, err := s.messageRepo.CreateMessage(
		ctx,
		sessionID,
		types.SenderAI,
		aiResponse.Content,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save AI message: %w", err)
	}

	return &types.SendMessageResponse{
		UserMessage: *userMessage,
		AIResponse:  *aiMessage,
	}, nil
}

// GetSessionMessages retrieves all messages for a session
func (s *ChatService) GetSessionMessages(ctx context.Context, userID, sessionID string) ([]types.Message, *types.ChatSession, error) {
	// Verify session ownership
	isOwner, err := s.sessionRepo.CheckSessionOwnership(ctx, sessionID, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check session ownership: %w", err)
	}
	if !isOwner {
		return nil, nil, fmt.Errorf("session not found or access denied")
	}

	// Get session info
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Get messages
	messages, err := s.messageRepo.GetSessionMessages(ctx, sessionID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get messages: %w", err)
	}

	return messages, session, nil
}

// CompleteSession marks a session as completed
func (s *ChatService) CompleteSession(ctx context.Context, userID, sessionID string) error {
	// Verify session ownership
	isOwner, err := s.sessionRepo.CheckSessionOwnership(ctx, sessionID, userID)
	if err != nil {
		return fmt.Errorf("failed to check session ownership: %w", err)
	}
	if !isOwner {
		return fmt.Errorf("session not found or access denied")
	}

	// Get session to check status
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if session.Status != types.SessionStatusActive {
		return fmt.Errorf("session is already completed")
	}

	// Complete the session
	err = s.sessionRepo.CompleteSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to complete session: %w", err)
	}

	// Trigger analysis in the background if analysis service is available
	if s.analysisService != nil {
		err = s.analysisService.TriggerAnalysisForCompletedSession(ctx, userID, sessionID)
		if err != nil {
			// Log error but don't fail the main flow
			// In production, this would go to proper logging system
			fmt.Printf("Failed to trigger analysis for session %s: %v\n", sessionID, err)
		}
	}

	return nil
}

// GetUserSessions retrieves session history for a user
func (s *ChatService) GetUserSessions(ctx context.Context, userID string, limit, offset int, year, month *int) (*types.SessionsResponse, error) {
	sessions, total, err := s.sessionRepo.GetUserSessions(ctx, userID, limit, offset, year, month)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	return &types.SessionsResponse{
		Sessions: sessions,
		Pagination: types.Pagination{
			Total:  total,
			Limit:  limit,
			Offset: offset,
		},
	}, nil
}

// CreateSessionForDate creates a session for a specific date (admin/testing purposes)
func (s *ChatService) CreateSessionForDate(ctx context.Context, userID, date string) (*types.ChatSession, error) {
	// Check if session already exists for this date
	session, err := s.sessionRepo.CreateSession(ctx, userID, date)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// getTimeOfDay determines the time of day for greeting personalization
func (s *ChatService) getTimeOfDay() string {
	hour := timeutil.NowJST().Hour()

	if hour >= 5 && hour < 12 {
		return "朝"
	} else if hour >= 12 && hour < 17 {
		return "昼"
	} else if hour >= 17 && hour < 21 {
		return "夕方"
	} else {
		return "夜"
	}
}

// GetSessionStats returns statistics for a session (message count, etc.)
func (s *ChatService) GetSessionStats(ctx context.Context, userID, sessionID string) (map[string]interface{}, error) {
	// Verify session ownership
	isOwner, err := s.sessionRepo.CheckSessionOwnership(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check session ownership: %w", err)
	}
	if !isOwner {
		return nil, fmt.Errorf("session not found or access denied")
	}

	// Get message count
	messageCount, err := s.messageRepo.GetMessageCount(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message count: %w", err)
	}

	// Get session info
	session, err := s.sessionRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	stats := map[string]interface{}{
		"message_count": messageCount,
		"status":        session.Status,
		"created_at":    session.CreatedAt,
		"updated_at":    session.UpdatedAt,
	}

	if session.CompletedAt != nil {
		stats["completed_at"] = session.CompletedAt
		stats["duration_minutes"] = int(session.CompletedAt.Sub(session.CreatedAt).Minutes())
	}

	return stats, nil
}
