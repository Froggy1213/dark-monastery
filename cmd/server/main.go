package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"dark-monastery/internal/ai"
	"dark-monastery/internal/game"

	"github.com/joho/godotenv"
)

func main() {
	// 1. Инициализируем загрузку .env файла
	if err := godotenv.Load(); err != nil {
		log.Println("Предупреждение: файл .env не найден, попытаемся взять ключ из системы")
	}

	// 2. Безопасно достаем ключ
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("КРИТИЧЕСКАЯ ОШИБКА: Переменная GEMINI_API_KEY не найдена!")
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=================================================")
	fmt.Println("    ТЕМНЫЙ МОНАСТЫРЬ: ТЕСТОВЫЙ ТЕРМИНАЛ (ФАЗА 1)")
	fmt.Println("=================================================")

	currentState := game.NewPlayer()
	fmt.Printf("\n[ЛОКАЦИЯ]: %s\n", currentState.Location)
	fmt.Printf("[СОСТОЯНИЕ]: %s\n", currentState.Condition)
	fmt.Printf("[ИНВЕНТАРЬ]: %v\n", currentState.Inventory)
	fmt.Printf("\n%s\n", currentState.Message)

	for {
		fmt.Print("\n> ВАШЕ ДЕЙСТВИЕ: ")
		playerInput, _ := reader.ReadString('\n')
		playerInput = strings.TrimSpace(playerInput)

		if strings.ToLower(playerInput) == "выход" || strings.ToLower(playerInput) == "quit" {
			fmt.Println("Покидаем монастырь...")
			break
		}

		fmt.Println("\n[ИИ думает...]")

		// Передаем безопасно извлеченный ключ
		newState, err := ai.GenerateNextTurn(apiKey, currentState, playerInput)
		if err != nil {
			fmt.Printf("Ошибка нейросети: %v\n", err)
			continue
		}

		currentState = newState

		fmt.Println("=================================================")
		fmt.Printf("[ЛОКАЦИЯ]: %s\n", currentState.Location)
		fmt.Printf("[СОСТОЯНИЕ]: %s\n", currentState.Condition)
		fmt.Printf("[ИНВЕНТАРЬ]: %v\n", currentState.Inventory)
		fmt.Printf("\n%s\n", currentState.Message)
		fmt.Println("=================================================")
	}
}
