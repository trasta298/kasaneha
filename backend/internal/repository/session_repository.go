package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/trasta298/kasaneha/backend/internal/types"
	"github.com/trasta298/kasaneha/backend/pkg/timeutil"
)

// SessionRepository handles chat session data operations
type SessionRepository struct {
	db *Database
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *Database) *SessionRepository {
	return &SessionRepository{db: db}
}

// GetTodaySession retrieves today's session for a user
func (r *SessionRepository) GetTodaySession(ctx context.Context, userID string) (*types.ChatSession, error) {
	today := timeutil.TodayJST()

	query := `
		SELECT id, user_id, session_date, status, created_at, updated_at, completed_at
		FROM chat_sessions
		WHERE user_id = $1 AND session_date = $2
	`

	var session types.ChatSession
	row := r.db.Pool.QueryRow(ctx, query, userID, today)

	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.SessionDate,
		&session.Status,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No session found for today
		}
		return nil, fmt.Errorf("failed to get today's session: %w", err)
	}

	return &session, nil
}

// CreateSession creates a new chat session
func (r *SessionRepository) CreateSession(ctx context.Context, userID, date string) (*types.ChatSession, error) {
	query := `
		INSERT INTO chat_sessions (user_id, session_date, status)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, session_date, status, created_at, updated_at, completed_at
	`

	var session types.ChatSession
	row := r.db.Pool.QueryRow(ctx, query, userID, date, types.SessionStatusActive)

	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.SessionDate,
		&session.Status,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &session, nil
}

// GetSessionByID retrieves a session by ID
func (r *SessionRepository) GetSessionByID(ctx context.Context, sessionID string) (*types.ChatSession, error) {
	query := `
		SELECT id, user_id, session_date, status, created_at, updated_at, completed_at
		FROM chat_sessions
		WHERE id = $1
	`

	var session types.ChatSession
	row := r.db.Pool.QueryRow(ctx, query, sessionID)

	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.SessionDate,
		&session.Status,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return &session, nil
}

// CompleteSession marks a session as completed
func (r *SessionRepository) CompleteSession(ctx context.Context, sessionID string) error {
	query := `
		UPDATE chat_sessions 
		SET status = $1, completed_at = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3
	`

	_, err := r.db.Pool.Exec(ctx, query, types.SessionStatusCompleted, timeutil.NowJST(), sessionID)
	if err != nil {
		return fmt.Errorf("failed to complete session: %w", err)
	}

	return nil
}

// GetUserSessions retrieves sessions for a user with pagination
func (r *SessionRepository) GetUserSessions(ctx context.Context, userID string, limit, offset int, year, month *int) ([]types.SessionSummary, int, error) {
	// Build base query
	whereClause := "WHERE user_id = $1"
	args := []interface{}{userID}
	argIndex := 2

	// Add date filters if provided
	if year != nil {
		whereClause += fmt.Sprintf(" AND EXTRACT(YEAR FROM session_date) = $%d", argIndex)
		args = append(args, *year)
		argIndex++
	}
	if month != nil {
		whereClause += fmt.Sprintf(" AND EXTRACT(MONTH FROM session_date) = $%d", argIndex)
		args = append(args, *month)
		argIndex++
	}

	// Count total sessions
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM chat_sessions
		%s
	`, whereClause)

	var total int
	err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	// Get sessions with pagination
	query := fmt.Sprintf(`
		SELECT 
			cs.id,
			cs.session_date::text,
			cs.status,
			cs.created_at,
			cs.updated_at,
			COUNT(m.id) as message_count,
			CASE WHEN a.id IS NOT NULL THEN true ELSE false END as has_analysis
		FROM chat_sessions cs
		LEFT JOIN messages m ON cs.id = m.session_id
		LEFT JOIN analyses a ON cs.id = a.session_id
		%s
		GROUP BY cs.id, cs.session_date, cs.status, cs.created_at, cs.updated_at, a.id
		ORDER BY cs.session_date DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get sessions: %w", err)
	}
	defer rows.Close()

	var sessions []types.SessionSummary
	for rows.Next() {
		var session types.SessionSummary
		err := rows.Scan(
			&session.ID,
			&session.Date,
			&session.Status,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.MessageCount,
			&session.HasAnalysis,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, total, nil
}

// GetCalendarData retrieves calendar data for a specific month
func (r *SessionRepository) GetCalendarData(ctx context.Context, userID string, year, month int) ([]types.CalendarDay, error) {
	query := `
		SELECT 
			cs.session_date,
			cs.status,
			a.tension_score,
			COALESCE(mc.message_count, 0) as message_count
		FROM chat_sessions cs
		LEFT JOIN analyses a ON cs.id = a.session_id
		LEFT JOIN (
			SELECT session_id, COUNT(*) as message_count
			FROM messages
			GROUP BY session_id
		) mc ON cs.id = mc.session_id
		WHERE cs.user_id = $1 
		  AND EXTRACT(YEAR FROM cs.session_date) = $2
		  AND EXTRACT(MONTH FROM cs.session_date) = $3
		ORDER BY cs.session_date
	`

	rows, err := r.db.Pool.Query(ctx, query, userID, year, month)
	if err != nil {
		return nil, fmt.Errorf("failed to get calendar data: %w", err)
	}
	defer rows.Close()

	var days []types.CalendarDay
	for rows.Next() {
		var sessionDate time.Time
		var status string
		var tensionScore *int
		var messageCount int

		err := rows.Scan(&sessionDate, &status, &tensionScore, &messageCount)
		if err != nil {
			return nil, fmt.Errorf("failed to scan calendar day: %w", err)
		}

		day := types.CalendarDay{
			Date:         timeutil.FormatJST(sessionDate, "2006-01-02"),
			HasSession:   true,
			Status:       status,
			TensionScore: tensionScore,
			MessageCount: &messageCount,
		}
		days = append(days, day)
	}

	return days, nil
}

// CheckSessionOwnership verifies if a session belongs to a user
func (r *SessionRepository) CheckSessionOwnership(ctx context.Context, sessionID, userID string) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM chat_sessions
		WHERE id = $1 AND user_id = $2
	`

	var count int
	err := r.db.Pool.QueryRow(ctx, query, sessionID, userID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check session ownership: %w", err)
	}

	return count > 0, nil
}
