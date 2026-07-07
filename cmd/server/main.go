package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"dark-monastery/internal/ai"
	"dark-monastery/internal/game"
	"dark-monastery/internal/handlers"
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
		runTerminal(ctx, engine, fileStore, pgStore, embedClient, extractor, *port)
	}
}

func runTerminal(
	ctx context.Context,
	engine *game.Engine,
	fileStore *storage.FileStore,
	pgStore *storage.PgStore,
	embedClient *ai.EmbeddingClient,
	extractor *memory.Extractor,
	port string,
) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=================================================")
	fmt.Println("      ТЁМНЫЙ МОНАСТЫРЬ (ТЕРМИНАЛ)")
	fmt.Println("=================================================")
	fmt.Println("Команды: /save, /load, /new, /saves, выход/quit")
	fmt.Println("=================================================")

	currentState := game.NewPlayer()
	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	turnCount := 0

	var memManager *memory.MemoryManager
	if pgStore != nil && embedClient != nil {
		memManager = memory.NewMemoryManager(pgStore, embedClient, extractor, sessionID)
	}

	if saves, _ := fileStore.List(); len(saves) > 0 {
		fmt.Printf("\nНайдено сохранений: %d\n", len(saves))
		for _, m := range saves {
			fmt.Printf("  [%s] %s — %s (ход %d)\n", m.SessionID, m.PlayerName, m.Location, m.TurnCount)
		}
		fmt.Print("\nЗагрузить последнее? (y/n): ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "y" || answer == "да" {
			last := saves[len(saves)-1]
			if err := loadGame(fileStore, last.SessionID, currentState, memManager); err != nil {
				fmt.Printf("Ошибка загрузки: %v\n", err)
			} else {
				sessionID = last.SessionID
				turnCount = last.TurnCount
				fmt.Println("Сохранение загружено!")
			}
		}
	}

	printState(currentState, turnCount)

	for {
		select {
		case <-sigCh:
			fmt.Println("\n\nАвтосохранение...")
			saveGame(fileStore, sessionID, "Игрок", currentState, turnCount, memManager)
			fmt.Println("Покидаем монастырь...")
			return
		default:
		}

		fmt.Print("\n> ВАШЕ ДЕЙСТВИЕ: ")
		playerInput, _ := reader.ReadString('\n')
		playerInput = strings.TrimSpace(playerInput)

		if playerInput == "" {
			continue
		}

		switch strings.ToLower(playerInput) {
		case "выход", "quit":
			fmt.Println("Сохраняем игру...")
			saveGame(fileStore, sessionID, "Игрок", currentState, turnCount, memManager)
			fmt.Println("Покидаем монастырь...")
			return

		case "/save":
			saveGame(fileStore, sessionID, "Игрок", currentState, turnCount, memManager)
			continue

		case "/load":
			fmt.Print("ID сохранения: ")
			id, _ := reader.ReadString('\n')
			id = strings.TrimSpace(id)
			if memManager != nil {
				memManager.Clear()
				memManager.SetSession(id)
			}
			if err := loadGame(fileStore, id, currentState, memManager); err != nil {
				fmt.Printf("Ошибка загрузки: %v\n", err)
			} else {
				sessionID = id
				fmt.Println("Сохранение загружено!")
				printState(currentState, turnCount)
			}
			continue

		case "/new":
			fmt.Print("Начать новую игру? (y/n): ")
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer == "y" || answer == "да" {
				currentState = game.NewPlayer()
				sessionID = fmt.Sprintf("session-%d", time.Now().Unix())
				turnCount = 0
				if memManager != nil {
					memManager.Clear()
					memManager.SetSession(sessionID)
				}
				fmt.Println("Новая игра начата!")
				printState(currentState, turnCount)
			}
			continue

		case "/saves":
			saves, err := fileStore.List()
			if err != nil {
				fmt.Printf("Ошибка: %v\n", err)
			} else if len(saves) == 0 {
				fmt.Println("Сохранений нет.")
			} else {
				fmt.Println("\n=== СОХРАНЕНИЯ ===")
				for _, m := range saves {
					fmt.Printf("[%s] %s — %s (ход %d, %s)\n",
						m.SessionID, m.PlayerName, m.Location,
						m.TurnCount, m.UpdatedAt.Format("02.01.2006 15:04"))
				}
			}
			continue
		}

		fmt.Println("\n[ИИ думает...]")

		// Строим RAG контекст
		if memManager != nil {
			memCtx, err := memManager.BuildContext(ctx, playerInput, currentState.Location)
			if err != nil {
				log.Printf("Ошибка получения памяти: %v", err)
			}
			if gemini, ok := engine.AIClient().(*ai.GeminiClient); ok {
				gemini.SetMemoryContext(memCtx)
			}
		}

		newState, err := engine.ProcessTurn(currentState, playerInput)
		if err != nil {
			fmt.Printf("Ошибка нейросети: %v\n", err)
			continue
		}

		turnCount++

		if memManager != nil {
			memManager.Add(playerInput, newState.Message, newState.Location)
		}

		currentState = newState

		printState(currentState, turnCount)
		saveGame(fileStore, sessionID, "Игрок", currentState, turnCount, memManager)
	}
}

func runWebServer(
	engine *game.Engine,
	fileStore *storage.FileStore,
	pgStore *storage.PgStore,
	embedClient *ai.EmbeddingClient,
	extractor *memory.Extractor,
	lore *game.LoreBook,
	port string,
) {
	srv := handlers.NewServer(engine, fileStore, lore, pgStore, embedClient, extractor)

	mux := http.NewServeMux()

	// API и WebSocket
	srv.RegisterRoutes(mux)

	// Статика
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Главная страница
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "./static/index.html")
			return
		}
		http.NotFound(w, r)
	})

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nЗавершение работы сервера...")
		os.Exit(0)
	}()

	fmt.Println("=================================================")
	fmt.Printf("  ТЁМНЫЙ МОНАСТЫРЬ — ВЕБ-СЕРВЕР\n")
	fmt.Printf("  Откройте http://localhost:%s\n", port)
	fmt.Println("=================================================")

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Ошибка сервера: %v", err)
	}
}

func printState(state *game.GameState, turn int) {
	fmt.Println("=================================================")
	fmt.Printf("Ход: %d\n", turn)
	fmt.Printf("[ЛОКАЦИЯ]: %s\n", state.Location)
	fmt.Printf("[СОСТОЯНИЕ]: %s | HP: %d/%d\n", state.Condition, state.HP, state.MaxHP)
	fmt.Printf("[РАССУДОК]: %s | Мана: %d\n", state.Sanity, state.Mana)
	fmt.Printf("[ЗОЛОТО]: %d\n", state.Gold)
	fmt.Printf("[ЭКИПИРОВАНО]: %s\n", state.Equipped)
	fmt.Printf("[ИНВЕНТАРЬ]: %v\n", state.Inventory)
	if len(state.ActiveQuests) > 0 {
		fmt.Printf("[КВЕСТЫ]: %v\n", state.ActiveQuests)
	}
	if len(state.Skills) > 0 {
		fmt.Printf("[НАВЫКИ]: %v\n", state.Skills)
	}
	fmt.Printf("\n%s\n", state.Message)
	fmt.Println("=================================================")
}

func saveGame(store *storage.FileStore, sessionID, playerName string, state *game.GameState, turnCount int, memManager *memory.MemoryManager) {
	meta := &storage.SaveMeta{
		SessionID:  sessionID,
		PlayerName: playerName,
		Location:   state.Location,
		TurnCount:  turnCount,
	}

	var records []storage.TurnRecord
	if memManager != nil {
		h := memManager.ShortTermHistory()
		recent := h.Recent(h.Len())
		records = make([]storage.TurnRecord, len(recent))
		for i, r := range recent {
			records[i] = storage.TurnRecord{
				PlayerAction: r.PlayerAction,
				AIResponse:   r.AIResponse,
			}
		}
	}

	if err := store.SaveWithHistory(sessionID, state, meta, records); err != nil {
		fmt.Printf("[Ошибка сохранения: %v]\n", err)
	} else {
		fmt.Printf("[Игра сохранена: %s]\n", sessionID)
	}
}

func loadGame(store *storage.FileStore, sessionID string, state *game.GameState, memManager *memory.MemoryManager) error {
	meta, records, err := store.LoadWithHistory(sessionID, state)
	if err != nil {
		return err
	}

	if memManager != nil {
		memManager.Clear()
		memManager.SetSession(sessionID)
		if meta != nil {
			memManager.SetTurnCounter(meta.TurnCount)
		}
		for _, r := range records {
			memManager.ShortTermHistory().Add(r.PlayerAction, r.AIResponse)
		}
	}
	return nil
}
