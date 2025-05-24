package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/trasta298/kasaneha/backend/internal/ai"
	"github.com/trasta298/kasaneha/backend/internal/config"
	"github.com/trasta298/kasaneha/backend/internal/handler"
	customMiddleware "github.com/trasta298/kasaneha/backend/internal/middleware"
	"github.com/trasta298/kasaneha/backend/internal/repository"
	"github.com/trasta298/kasaneha/backend/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logger
	logger := customMiddleware.SetupLogger(cfg.IsDevelopment())
	logger.Info("Starting Kasaneha API server")

	// Initialize database
	db, err := repository.NewDatabase(cfg)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations if needed
	if err := runMigrations(db); err != nil {
		logger.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize AI client
	aiClient, err := ai.NewClient(cfg.AI.GeminiAPIKey, cfg.AI.Model)
	if err != nil {
		logger.Fatalf("Failed to initialize AI client: %v", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)

	// Initialize services
	chatService := service.NewChatService(sessionRepo, messageRepo, userRepo, aiClient)
	analysisService := service.NewAnalysisService(analysisRepo, sessionRepo, messageRepo, userRepo, aiClient)

	// Set circular dependency after initialization
	chatService.SetAnalysisService(analysisService)

	// Initialize middlewares
	authMiddleware := customMiddleware.NewAuthMiddleware(cfg.JWT.Secret)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(userRepo, authMiddleware)
	chatHandler := handler.NewChatHandler(chatService)
	analysisHandler := handler.NewAnalysisHandler(analysisService)

	// Setup router
	r := chi.NewRouter()

	// Global middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMiddleware.Logger(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:4321",
			"http://localhost:3000",
			"http://frontend:4321",
			"http://127.0.0.1:4321",
		}, // Astro dev server + container network
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
		})

		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.AuthenticateUser)

			// User routes
			r.Get("/auth/me", authHandler.Me)

			// Chat session routes
			r.Route("/sessions", func(r chi.Router) {
				r.Get("/today", chatHandler.GetTodaySession)
				r.Post("/", chatHandler.CreateSession)
				r.Get("/", chatHandler.GetUserSessions)

				r.Route("/{sessionId}", func(r chi.Router) {
					r.Get("/messages", chatHandler.GetSessionMessages)
					r.Post("/messages", chatHandler.SendMessage)
					r.Put("/complete", chatHandler.CompleteSession)
					r.Get("/stats", chatHandler.GetSessionStats)
					r.Get("/analysis", analysisHandler.GetSessionAnalysis)
					r.Post("/analysis", analysisHandler.TriggerSessionAnalysis)
				})
			})

			// Analysis routes
			r.Route("/analysis", func(r chi.Router) {
				r.Get("/scores", analysisHandler.GetTensionScores)
				r.Get("/insights", analysisHandler.GetAnalysisInsights)
				r.Get("/history", analysisHandler.GetUserAnalyses)
			})

			// Calendar routes
			r.Route("/calendar", func(r chi.Router) {
				r.Get("/{year}/{month}", analysisHandler.GetCalendarData)
			})
		})
	})

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

// runMigrations checks if tables exist and creates them if they don't
func runMigrations(db *repository.Database) error {
	ctx := context.Background()

	// Check if users table exists
	var exists bool
	err := db.Pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'users'
		)
	`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if users table exists: %w", err)
	}

	if !exists {
		// Read and execute migration file
		migrationSQL := `
-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    timezone VARCHAR(50) DEFAULT 'UTC',
    
    -- Constraints
    CONSTRAINT users_username_check CHECK (length(username) >= 3),
    CONSTRAINT users_email_check CHECK (email ~* '^[^@]+@[^@]+\.[^@]+$')
);

-- Chat sessions table
CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_date DATE NOT NULL,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'completed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- One session per user per day
    UNIQUE(user_id, session_date)
);

-- Messages table
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    sender VARCHAR(10) NOT NULL CHECK (sender IN ('user', 'ai')),
    content TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}'::jsonb,
    sequence_number INTEGER NOT NULL
);

-- Analyses table
CREATE TABLE analyses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    summary TEXT NOT NULL,
    emotional_state JSONB NOT NULL,
    behavioral_insights JSONB NOT NULL,
    tension_score INTEGER NOT NULL CHECK (tension_score >= 0 AND tension_score <= 100),
    relative_score INTEGER CHECK (relative_score >= -50 AND relative_score <= 50),
    keywords JSONB DEFAULT '[]'::jsonb,
    raw_analysis_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- One analysis per session
    UNIQUE(session_id)
);

-- User statistics table (cached data)
CREATE TABLE user_statistics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    total_sessions INTEGER DEFAULT 0,
    average_tension_score DECIMAL(5,2),
    min_tension_score INTEGER,
    max_tension_score INTEGER,
    most_common_emotions JSONB DEFAULT '[]'::jsonb,
    last_calculated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- One statistics record per user
    UNIQUE(user_id)
);

-- Indexes for performance

-- Users indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;

-- Chat sessions indexes
CREATE INDEX idx_chat_sessions_user_id ON chat_sessions(user_id);
CREATE INDEX idx_chat_sessions_date ON chat_sessions(session_date);
CREATE INDEX idx_chat_sessions_user_date ON chat_sessions(user_id, session_date);
CREATE INDEX idx_chat_sessions_status ON chat_sessions(status);

-- Messages indexes
CREATE INDEX idx_messages_session_id ON messages(session_id);
CREATE INDEX idx_messages_session_sequence ON messages(session_id, sequence_number);
CREATE INDEX idx_messages_created_at ON messages(created_at);
CREATE INDEX idx_messages_sender ON messages(sender);

-- Analyses indexes
CREATE INDEX idx_analyses_session_id ON analyses(session_id);
CREATE INDEX idx_analyses_tension_score ON analyses(tension_score);
CREATE INDEX idx_analyses_created_at ON analyses(created_at);

-- JSONB indexes for searching
CREATE INDEX idx_analyses_emotional_state ON analyses USING GIN (emotional_state);
CREATE INDEX idx_analyses_keywords ON analyses USING GIN (keywords);

-- User statistics indexes
CREATE INDEX idx_user_statistics_user_id ON user_statistics(user_id);

-- Composite indexes for common queries
CREATE INDEX idx_sessions_user_month ON chat_sessions(user_id, EXTRACT(YEAR FROM session_date), EXTRACT(MONTH FROM session_date));

-- Partial indexes for active sessions
CREATE INDEX idx_active_sessions ON chat_sessions(user_id, session_date) WHERE status = 'active';

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_chat_sessions_updated_at BEFORE UPDATE ON chat_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Sequence for message ordering
CREATE SEQUENCE message_sequence_seq;
		`

		_, err = db.Pool.Exec(ctx, migrationSQL)
		if err != nil {
			return fmt.Errorf("failed to execute migration: %w", err)
		}

		log.Println("Database migration completed successfully")
	}

	return nil
}
