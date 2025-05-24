package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/trasta298/kasaneha/backend/internal/types"
)

// MessageRepository handles message data operations
type MessageRepository struct {
	db *Database
}

// NewMessageRepository creates a new message repository
func NewMessageRepository(db *Database) *MessageRepository {
	return &MessageRepository{db: db}
}

// CreateMessage creates a new message
func (r *MessageRepository) CreateMessage(ctx context.Context, sessionID, sender, content string, metadata map[string]interface{}) (*types.Message, error) {
	// Get next sequence number for this session
	sequenceQuery := `
		SELECT COALESCE(MAX(sequence_number), 0) + 1
		FROM messages
		WHERE session_id = $1
	`

	var sequenceNumber int
	err := r.db.Pool.QueryRow(ctx, sequenceQuery, sessionID).Scan(&sequenceNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get sequence number: %w", err)
	}

	// Convert metadata to JSON
	var metadataJSON []byte
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	// Insert message
	query := `
		INSERT INTO messages (session_id, sender, content, metadata, sequence_number)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, session_id, sender, content, created_at, metadata, sequence_number
	`

	var message types.Message
	row := r.db.Pool.QueryRow(ctx, query, sessionID, sender, content, metadataJSON, sequenceNumber)

	err = row.Scan(
		&message.ID,
		&message.SessionID,
		&message.Sender,
		&message.Content,
		&message.CreatedAt,
		&message.Metadata,
		&message.SequenceNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	return &message, nil
}

// GetSessionMessages retrieves all messages for a session
func (r *MessageRepository) GetSessionMessages(ctx context.Context, sessionID string) ([]types.Message, error) {
	query := `
		SELECT id, session_id, sender, content, created_at, metadata, sequence_number
		FROM messages
		WHERE session_id = $1
		ORDER BY sequence_number ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []types.Message
	for rows.Next() {
		var message types.Message
		err := rows.Scan(
			&message.ID,
			&message.SessionID,
			&message.Sender,
			&message.Content,
			&message.CreatedAt,
			&message.Metadata,
			&message.SequenceNumber,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, message)
	}

	return messages, nil
}

// GetSessionMessagesWithPagination retrieves messages for a session with pagination
func (r *MessageRepository) GetSessionMessagesWithPagination(ctx context.Context, sessionID string, limit, offset int) ([]types.Message, int, error) {
	// Count total messages
	countQuery := `
		SELECT COUNT(*)
		FROM messages
		WHERE session_id = $1
	`

	var total int
	err := r.db.Pool.QueryRow(ctx, countQuery, sessionID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	// Get messages with pagination
	query := `
		SELECT id, session_id, sender, content, created_at, metadata, sequence_number
		FROM messages
		WHERE session_id = $1
		ORDER BY sequence_number ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool.Query(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []types.Message
	for rows.Next() {
		var message types.Message
		err := rows.Scan(
			&message.ID,
			&message.SessionID,
			&message.Sender,
			&message.Content,
			&message.CreatedAt,
			&message.Metadata,
			&message.SequenceNumber,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, message)
	}

	return messages, total, nil
}

// GetLatestMessages retrieves the latest N messages for a session
func (r *MessageRepository) GetLatestMessages(ctx context.Context, sessionID string, limit int) ([]types.Message, error) {
	query := `
		SELECT id, session_id, sender, content, created_at, metadata, sequence_number
		FROM messages
		WHERE session_id = $1
		ORDER BY sequence_number DESC
		LIMIT $2
	`

	rows, err := r.db.Pool.Query(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest messages: %w", err)
	}
	defer rows.Close()

	var messages []types.Message
	for rows.Next() {
		var message types.Message
		err := rows.Scan(
			&message.ID,
			&message.SessionID,
			&message.Sender,
			&message.Content,
			&message.CreatedAt,
			&message.Metadata,
			&message.SequenceNumber,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, message)
	}

	// Reverse to get chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetMessageByID retrieves a specific message by ID
func (r *MessageRepository) GetMessageByID(ctx context.Context, messageID string) (*types.Message, error) {
	query := `
		SELECT id, session_id, sender, content, created_at, metadata, sequence_number
		FROM messages
		WHERE id = $1
	`

	var message types.Message
	row := r.db.Pool.QueryRow(ctx, query, messageID)

	err := row.Scan(
		&message.ID,
		&message.SessionID,
		&message.Sender,
		&message.Content,
		&message.CreatedAt,
		&message.Metadata,
		&message.SequenceNumber,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return &message, nil
}

// GetConversationLog retrieves all messages for a session as a formatted string for AI analysis
func (r *MessageRepository) GetConversationLog(ctx context.Context, sessionID string) (string, error) {
	messages, err := r.GetSessionMessages(ctx, sessionID)
	if err != nil {
		return "", err
	}

	var log string
	for _, msg := range messages {
		senderName := "ユーザー"
		if msg.Sender == types.SenderAI {
			senderName = "かさね"
		}
		log += fmt.Sprintf("%s: %s\n", senderName, msg.Content)
	}

	return log, nil
}

// UpdateMessage updates a message (for future editing functionality)
func (r *MessageRepository) UpdateMessage(ctx context.Context, messageID, content string, metadata map[string]interface{}) error {
	// Convert metadata to JSON
	var metadataJSON []byte
	var err error
	if metadata != nil {
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		UPDATE messages 
		SET content = $1, metadata = $2
		WHERE id = $3
	`

	_, err = r.db.Pool.Exec(ctx, query, content, metadataJSON, messageID)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	return nil
}

// DeleteMessage deletes a message (for future moderation functionality)
func (r *MessageRepository) DeleteMessage(ctx context.Context, messageID string) error {
	query := `
		DELETE FROM messages
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("message not found")
	}

	return nil
}

// GetMessageCount returns the total number of messages for a session
func (r *MessageRepository) GetMessageCount(ctx context.Context, sessionID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM messages
		WHERE session_id = $1
	`

	var count int
	err := r.db.Pool.QueryRow(ctx, query, sessionID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count messages: %w", err)
	}

	return count, nil
}
