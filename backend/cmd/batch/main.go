package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/trasta298/kasaneha/backend/internal/ai"
	"github.com/trasta298/kasaneha/backend/internal/config"
	"github.com/trasta298/kasaneha/backend/internal/repository"
	"github.com/trasta298/kasaneha/backend/internal/service"
)

func main() {
	// Define command line flags
	minMessages := flag.Int("min-messages", 2, "Minimum number of messages required for analysis")
	dryRun := flag.Bool("dry-run", false, "Show sessions that would be analyzed without actually running analysis")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := repository.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repositories
	sessionRepo := repository.NewSessionRepository(db)
	messageRepo := repository.NewMessageRepository(db)
	analysisRepo := repository.NewAnalysisRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Initialize AI client
	aiClient, err := ai.NewClient(cfg.AI.GeminiAPIKey, cfg.AI.Model)
	if err != nil {
		log.Fatalf("Failed to initialize AI client: %v", err)
	}

	// Initialize analysis service
	analysisService := service.NewAnalysisService(
		analysisRepo,
		sessionRepo,
		messageRepo,
		userRepo,
		aiClient,
	)

	ctx := context.Background()

	if *dryRun {
		// Dry run: show sessions that would be analyzed
		fmt.Printf("DRY RUN: Finding active sessions with at least %d messages...\n", *minMessages)
		sessions, err := sessionRepo.GetActiveSessionsWithMinMessages(ctx, *minMessages)
		if err != nil {
			log.Fatalf("Failed to get active sessions: %v", err)
		}

		if len(sessions) == 0 {
			fmt.Println("No active sessions found that need analysis.")
			return
		}

		fmt.Printf("Found %d sessions that would be analyzed:\n", len(sessions))
		for _, session := range sessions {
			fmt.Printf("- Session %s (User: %s, Messages: %d, Date: %s)\n",
				session.ID, session.UserID, session.MessageCount, session.Date)
		}

		fmt.Printf("\nRun without --dry-run to actually perform the analysis.\n")
		return
	}

	// Actual batch analysis
	fmt.Printf("Starting batch analysis for active sessions with at least %d messages...\n", *minMessages)

	err = analysisService.BatchAnalyzeActiveSessions(ctx, *minMessages)
	if err != nil {
		log.Fatalf("Batch analysis failed: %v", err)
	}

	fmt.Println("Batch analysis completed successfully!")
}
