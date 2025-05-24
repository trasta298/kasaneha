package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/trasta298/kasaneha/backend/internal/types"
	"github.com/trasta298/kasaneha/backend/pkg/timeutil"
)

// AnalysisRepository handles analysis data operations
type AnalysisRepository struct {
	db *Database
}

// NewAnalysisRepository creates a new analysis repository
func NewAnalysisRepository(db *Database) *AnalysisRepository {
	return &AnalysisRepository{db: db}
}

// CreateAnalysis creates a new analysis record
func (r *AnalysisRepository) CreateAnalysis(ctx context.Context, analysis *types.Analysis) (*types.Analysis, error) {
	// Convert JSON fields to bytes
	emotionalStateJSON, err := json.Marshal(analysis.EmotionalState)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal emotional state: %w", err)
	}

	behavioralInsightsJSON, err := json.Marshal(analysis.BehavioralInsights)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal behavioral insights: %w", err)
	}

	keywordsJSON, err := json.Marshal(analysis.Keywords)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal keywords: %w", err)
	}

	var rawAnalysisDataJSON []byte
	if analysis.RawAnalysisData != nil {
		rawAnalysisDataJSON, err = json.Marshal(analysis.RawAnalysisData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal raw analysis data: %w", err)
		}
	}

	query := `
		INSERT INTO analyses (
			session_id, summary, emotional_state, behavioral_insights, 
			tension_score, relative_score, keywords, raw_analysis_data
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, session_id, summary, emotional_state, behavioral_insights, 
		          tension_score, relative_score, keywords, raw_analysis_data, created_at
	`

	var result types.Analysis
	row := r.db.Pool.QueryRow(
		ctx, query,
		analysis.SessionID,
		analysis.Summary,
		emotionalStateJSON,
		behavioralInsightsJSON,
		analysis.TensionScore,
		analysis.RelativeScore,
		keywordsJSON,
		rawAnalysisDataJSON,
	)

	err = row.Scan(
		&result.ID,
		&result.SessionID,
		&result.Summary,
		&result.EmotionalState,
		&result.BehavioralInsights,
		&result.TensionScore,
		&result.RelativeScore,
		&result.Keywords,
		&result.RawAnalysisData,
		&result.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create analysis: %w", err)
	}

	return &result, nil
}

// GetAnalysisBySessionID retrieves analysis for a specific session
func (r *AnalysisRepository) GetAnalysisBySessionID(ctx context.Context, sessionID string) (*types.Analysis, error) {
	query := `
		SELECT id, session_id, summary, emotional_state, behavioral_insights, 
		       tension_score, relative_score, keywords, raw_analysis_data, created_at
		FROM analyses
		WHERE session_id = $1
	`

	var analysis types.Analysis
	row := r.db.Pool.QueryRow(ctx, query, sessionID)

	err := row.Scan(
		&analysis.ID,
		&analysis.SessionID,
		&analysis.Summary,
		&analysis.EmotionalState,
		&analysis.BehavioralInsights,
		&analysis.TensionScore,
		&analysis.RelativeScore,
		&analysis.Keywords,
		&analysis.RawAnalysisData,
		&analysis.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No analysis found for this session
		}
		return nil, fmt.Errorf("failed to get analysis: %w", err)
	}

	return &analysis, nil
}

// GetTensionScores retrieves tension scores for a user within a date range
func (r *AnalysisRepository) GetTensionScores(ctx context.Context, userID string, startDate, endDate time.Time, limit int) ([]types.TensionScoreData, error) {
	query := `
		SELECT 
			cs.session_date::text as date,
			a.tension_score,
			a.relative_score,
			a.session_id
		FROM analyses a
		JOIN chat_sessions cs ON a.session_id = cs.id
		WHERE cs.user_id = $1 
		  AND cs.session_date >= $2 
		  AND cs.session_date <= $3
		ORDER BY cs.session_date DESC
		LIMIT $4
	`

	rows, err := r.db.Pool.Query(ctx, query, userID, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get tension scores: %w", err)
	}
	defer rows.Close()

	var scores []types.TensionScoreData
	for rows.Next() {
		var score types.TensionScoreData
		var relativeScore *int

		err := rows.Scan(&score.Date, &score.TensionScore, &relativeScore, &score.SessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tension score: %w", err)
		}

		if relativeScore != nil {
			score.RelativeScore = *relativeScore
		}

		scores = append(scores, score)
	}

	return scores, nil
}

// GetTensionStatistics calculates tension score statistics for a user
func (r *AnalysisRepository) GetTensionStatistics(ctx context.Context, userID string, days int) (*types.TensionStatistics, error) {
	endDate := timeutil.NowJST()
	startDate := endDate.AddDate(0, 0, -days)

	query := `
		SELECT 
			AVG(a.tension_score) as average,
			MIN(a.tension_score) as min_score,
			MAX(a.tension_score) as max_score,
			COUNT(*) as count
		FROM analyses a
		JOIN chat_sessions cs ON a.session_id = cs.id
		WHERE cs.user_id = $1 
		  AND cs.session_date >= $2 
		  AND cs.session_date <= $3
	`

	var avg *float64
	var minScore, maxScore *int
	var count int

	row := r.db.Pool.QueryRow(ctx, query, userID, startDate, endDate)
	err := row.Scan(&avg, &minScore, &maxScore, &count)
	if err != nil {
		return nil, fmt.Errorf("failed to get tension statistics: %w", err)
	}

	if count == 0 {
		return &types.TensionStatistics{
			Average: 0,
			Min:     0,
			Max:     0,
			Trend:   "stable",
		}, nil
	}

	// Calculate trend by comparing recent vs older scores
	trend := "stable"
	if count >= 7 { // Need at least a week of data
		trendQuery := `
			WITH recent_scores AS (
				SELECT AVG(a.tension_score) as avg_score
				FROM analyses a
				JOIN chat_sessions cs ON a.session_id = cs.id
				WHERE cs.user_id = $1 
				  AND cs.session_date >= $2
			),
			older_scores AS (
				SELECT AVG(a.tension_score) as avg_score
				FROM analyses a
				JOIN chat_sessions cs ON a.session_id = cs.id
				WHERE cs.user_id = $1 
				  AND cs.session_date >= $3
				  AND cs.session_date < $2
			)
			SELECT 
				recent_scores.avg_score - COALESCE(older_scores.avg_score, recent_scores.avg_score) as trend_diff
			FROM recent_scores
			LEFT JOIN older_scores ON true
		`

		recentDate := endDate.AddDate(0, 0, -3)     // Last 3 days
		olderStartDate := endDate.AddDate(0, 0, -7) // 4-7 days ago

		var trendDiff float64
		err = r.db.Pool.QueryRow(ctx, trendQuery, userID, recentDate, olderStartDate).Scan(&trendDiff)
		if err == nil {
			if trendDiff > 5 {
				trend = "improving"
			} else if trendDiff < -5 {
				trend = "declining"
			}
		}
	}

	stats := &types.TensionStatistics{
		Average: *avg,
		Min:     *minScore,
		Max:     *maxScore,
		Trend:   trend,
	}

	return stats, nil
}

// UpdateAnalysis updates an existing analysis record
func (r *AnalysisRepository) UpdateAnalysis(ctx context.Context, analysisID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic query
	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)
	argIndex := 1

	for field, value := range updates {
		// Handle JSON fields
		if field == "emotional_state" || field == "behavioral_insights" ||
			field == "keywords" || field == "raw_analysis_data" {
			jsonValue, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("failed to marshal %s: %w", field, err)
			}
			setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
			args = append(args, jsonValue)
		} else {
			setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
			args = append(args, value)
		}
		argIndex++
	}

	query := fmt.Sprintf(`
		UPDATE analyses 
		SET %s
		WHERE id = $%d
	`, fmt.Sprintf("%s", setParts), argIndex)

	args = append(args, analysisID)

	_, err := r.db.Pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update analysis: %w", err)
	}

	return nil
}

// DeleteAnalysis deletes an analysis record
func (r *AnalysisRepository) DeleteAnalysis(ctx context.Context, analysisID string) error {
	query := `
		DELETE FROM analyses
		WHERE id = $1
	`

	result, err := r.db.Pool.Exec(ctx, query, analysisID)
	if err != nil {
		return fmt.Errorf("failed to delete analysis: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("analysis not found")
	}

	return nil
}

// GetAnalysesByUserID retrieves all analyses for a user with pagination
func (r *AnalysisRepository) GetAnalysesByUserID(ctx context.Context, userID string, limit, offset int) ([]types.Analysis, int, error) {
	// Count total analyses
	countQuery := `
		SELECT COUNT(*)
		FROM analyses a
		JOIN chat_sessions cs ON a.session_id = cs.id
		WHERE cs.user_id = $1
	`

	var total int
	err := r.db.Pool.QueryRow(ctx, countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count analyses: %w", err)
	}

	// Get analyses with pagination
	query := `
		SELECT 
			a.id, a.session_id, a.summary, a.emotional_state, a.behavioral_insights,
			a.tension_score, a.relative_score, a.keywords, a.raw_analysis_data, a.created_at
		FROM analyses a
		JOIN chat_sessions cs ON a.session_id = cs.id
		WHERE cs.user_id = $1
		ORDER BY a.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get analyses: %w", err)
	}
	defer rows.Close()

	var analyses []types.Analysis
	for rows.Next() {
		var analysis types.Analysis
		err := rows.Scan(
			&analysis.ID,
			&analysis.SessionID,
			&analysis.Summary,
			&analysis.EmotionalState,
			&analysis.BehavioralInsights,
			&analysis.TensionScore,
			&analysis.RelativeScore,
			&analysis.Keywords,
			&analysis.RawAnalysisData,
			&analysis.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan analysis: %w", err)
		}
		analyses = append(analyses, analysis)
	}

	return analyses, total, nil
}
