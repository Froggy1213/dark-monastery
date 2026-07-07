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
	mode := flag.String("mode", "terminal", "Режим запуска: terminal или web")
	port := flag.String("port", "8080", "Порт для веб-сервера")
	flag.Parse()

	if err := godotenv.Load(); err != nil {
		log.Println("Предупреждение: файл .env не найден, попытаемся взять ключ из системы")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("КРИТИЧЕСКАЯ ОШИБКА: Переменная GEMINI_API_KEY не найдена!")
	}

	// Инициализация AI компонентов
	geminiClient := ai.NewGeminiClient(apiKey)
	embedClient := ai.NewEmbeddingClient(apiKey)
	lore := game.DefaultLore()
	geminiClient.SetLore(lore)
	engine := game.NewEngine(geminiClient)

	// Файловое хранилище (для состояний и метаданных)
	homeDir, _ := os.UserHomeDir()
	saveDir := filepath.Join(homeDir, ".dark-monastery", "saves")
	fileStore, err := storage.NewFileStore(saveDir)
	if err != nil {
		log.Fatalf("Ошибка инициализации файлового хранилища: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Инициализация RAG (PostgreSQL + Async Extractor)
	var pgStore *storage.PgStore
	var extractor *memory.Extractor

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		pgStore, err = storage.NewPgStore(ctx, dbURL)
		if err != nil {
			log.Printf("ПРЕДУПРЕЖДЕНИЕ: Ошибка подключения к PostgreSQL, игра запустится в fallback-режиме (без RAG): %v", err)
		} else {
			if err := pgStore.Migrate(ctx); err != nil {
				log.Fatalf("Ошибка миграции БД: %v", err)
			}
			defer pgStore.Close()

			extractor = memory.NewExtractor(embedClient, pgStore, 2)
			extractor.Start(ctx)
			defer extractor.Stop()
		}
	} else {
		log.Println("ПРЕДУПРЕЖДЕНИЕ: DATABASE_URL не задан, игра запустится в fallback-режиме (без долгосрочной памяти)")
	}

	switch *mode {
	case "web":
		runWebServer(engine, fileStore, pgStore, embedClient, extractor, lore, *port)
	default:
		runTerminal(ctx, engine, fileStore, pgStore, embedClient, extractor)
	}

	fmt.Println("Покидаем монастырь...")
}
