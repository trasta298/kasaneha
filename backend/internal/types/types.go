package types

import (
	"encoding/json"
	"time"
)

// User represents a user in the system
type User struct {
	ID           string     `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Email        *string    `json:"email,omitempty" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	Timezone     string     `json:"timezone" db:"timezone"`
}

// ChatSession represents a daily chat session
type ChatSession struct {
	ID          string     `json:"id" db:"id"`
	UserID      string     `json:"user_id" db:"user_id"`
	SessionDate time.Time  `json:"session_date" db:"session_date"`
	Status      string     `json:"status" db:"status"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}

// Message represents a chat message
type Message struct {
	ID             string          `json:"id" db:"id"`
	SessionID      string          `json:"session_id" db:"session_id"`
	Sender         string          `json:"sender" db:"sender"`
	Content        string          `json:"content" db:"content"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	Metadata       json.RawMessage `json:"metadata,omitempty" db:"metadata"`
	SequenceNumber int             `json:"sequence_number" db:"sequence_number"`
}

// Analysis represents AI analysis of a chat session
type Analysis struct {
	ID                 string          `json:"id" db:"id"`
	SessionID          string          `json:"session_id" db:"session_id"`
	Summary            string          `json:"summary" db:"summary"`
	EmotionalState     json.RawMessage `json:"emotional_state" db:"emotional_state"`
	BehavioralInsights json.RawMessage `json:"behavioral_insights" db:"behavioral_insights"`
	TensionScore       int             `json:"tension_score" db:"tension_score"`
	RelativeScore      *int            `json:"relative_score,omitempty" db:"relative_score"`
	Keywords           json.RawMessage `json:"keywords" db:"keywords"`
	RawAnalysisData    json.RawMessage `json:"raw_analysis_data,omitempty" db:"raw_analysis_data"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
}

// UserStatistics represents cached user statistics
type UserStatistics struct {
	ID                  string          `json:"id" db:"id"`
	UserID              string          `json:"user_id" db:"user_id"`
	TotalSessions       int             `json:"total_sessions" db:"total_sessions"`
	AverageTensionScore *float64        `json:"average_tension_score,omitempty" db:"average_tension_score"`
	MinTensionScore     *int            `json:"min_tension_score,omitempty" db:"min_tension_score"`
	MaxTensionScore     *int            `json:"max_tension_score,omitempty" db:"max_tension_score"`
	MostCommonEmotions  json.RawMessage `json:"most_common_emotions" db:"most_common_emotions"`
	LastCalculatedAt    time.Time       `json:"last_calculated_at" db:"last_calculated_at"`
}

// EmotionalState represents the emotional analysis result
type EmotionalState struct {
	PrimaryEmotion string             `json:"primary_emotion"`
	Emotions       map[string]float64 `json:"emotions"`
	Confidence     float64            `json:"confidence"`
}

// TensionScoreData represents tension score information
type TensionScoreData struct {
	Date          string `json:"date"`
	TensionScore  int    `json:"tension_score"`
	RelativeScore int    `json:"relative_score"`
	SessionID     string `json:"session_id"`
}

// Constants for message senders
const (
	SenderUser = "user"
	SenderAI   = "ai"
)

// Constants for session status
const (
	SessionStatusActive    = "active"
	SessionStatusCompleted = "completed"
)

// API Request/Response types

// LoginRequest represents login request body
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3"`
	Password string `json:"password" validate:"required,min=6"`
}

// RegisterRequest represents registration request body
type RegisterRequest struct {
	Username string  `json:"username" validate:"required,min=3"`
	Password string  `json:"password" validate:"required,min=6"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// CreateSessionRequest represents session creation request
type CreateSessionRequest struct {
	Date string `json:"date" validate:"required"`
}

// CreateSessionResponse represents session creation response
type CreateSessionResponse struct {
	Session        ChatSession `json:"session"`
	InitialMessage Message     `json:"initial_message"`
}

// SendMessageRequest represents message sending request
type SendMessageRequest struct {
	Content string `json:"content" validate:"required,max=2000"`
}

// SendMessageResponse represents message sending response
type SendMessageResponse struct {
	UserMessage Message `json:"user_message"`
	AIResponse  Message `json:"ai_response"`
}

// SessionsResponse represents sessions list response
type SessionsResponse struct {
	Sessions   []SessionSummary `json:"sessions"`
	Pagination Pagination       `json:"pagination"`
}

// SessionSummary represents a summary of a chat session
type SessionSummary struct {
	ID           string    `json:"id"`
	Date         string    `json:"date"`
	Status       string    `json:"status"`
	MessageCount int       `json:"message_count"`
	HasAnalysis  bool      `json:"has_analysis"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SessionForBatch represents a session summary with user ID for batch processing
type SessionForBatch struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Date         string    `json:"date"`
	Status       string    `json:"status"`
	MessageCount int       `json:"message_count"`
	HasAnalysis  bool      `json:"has_analysis"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Pagination represents pagination information
type Pagination struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// AnalysisResponse represents analysis data response
type AnalysisResponse struct {
	Analysis *Analysis `json:"analysis"`
}

// TensionScoresResponse represents tension scores response
type TensionScoresResponse struct {
	Scores     []TensionScoreData `json:"scores"`
	Statistics TensionStatistics  `json:"statistics"`
}

// TensionStatistics represents tension score statistics
type TensionStatistics struct {
	Average float64 `json:"average"`
	Min     int     `json:"min"`
	Max     int     `json:"max"`
	Trend   string  `json:"trend"`
}

// CalendarResponse represents calendar data response
type CalendarResponse struct {
	MonthData CalendarMonthData `json:"month_data"`
}

// CalendarMonthData represents calendar month data
type CalendarMonthData struct {
	Year  int           `json:"year"`
	Month int           `json:"month"`
	Days  []CalendarDay `json:"days"`
}

// CalendarDay represents a day in the calendar
type CalendarDay struct {
	Date         string `json:"date"`
	HasSession   bool   `json:"has_session"`
	TensionScore *int   `json:"tension_score,omitempty"`
	Status       string `json:"status,omitempty"`
	MessageCount *int   `json:"message_count,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}
