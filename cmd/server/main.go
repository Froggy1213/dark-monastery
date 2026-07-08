package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"dark-monastery/internal/ai"
	"dark-monastery/internal/game"
	"dark-monastery/internal/memory"
	"dark-monastery/internal/storage"

	"github.com/joho/godotenv"
)

func main() {
	mode := flag.String("mode", "terminal", "Run mode: terminal or web")
	port := flag.String("port", "8080", "Web server port")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, will try to get key from environment")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("CRITICAL ERROR: GEMINI_API_KEY environment variable not found!")
	}

	// Initialize AI components
	geminiClient := ai.NewGeminiClient(apiKey)
	embedClient := ai.NewEmbeddingClient(apiKey)
	lore := game.DefaultLore()
	geminiClient.SetLore(lore)
	engine := game.NewEngine(geminiClient)

	// File storage (for states and metadata)
	homeDir, _ := os.UserHomeDir()
	saveDir := filepath.Join(homeDir, ".dark-monastery", "saves")
	fileStore, err := storage.NewFileStore(saveDir)
	if err != nil {
		log.Fatalf("File storage initialization error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize RAG (PostgreSQL + Async Extractor)
	var pgStore *storage.PgStore
	var extractor *memory.Extractor

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		pgStore, err = storage.NewPgStore(ctx, dbURL)
		if err != nil {
			log.Printf("WARNING: PostgreSQL connection error, running in fallback mode (no RAG): %v", err)
		} else {
			if err := pgStore.Migrate(ctx); err != nil {
				log.Fatalf("DB migration error: %v", err)
			}
			defer pgStore.Close()

			extractor = memory.NewExtractor(embedClient, pgStore, 2)
			extractor.Start(ctx)
			defer extractor.Stop()
		}
	} else {
		log.Println("WARNING: DATABASE_URL not set, running in fallback mode (no long-term memory)")
	}

	switch *mode {
	case "web":
		runWebServer(engine, fileStore, pgStore, embedClient, extractor, lore, *port)
	default:
		runTerminal(ctx, engine, fileStore, pgStore, embedClient, extractor)
	}

	fmt.Println("Leaving the monastery...")
}
