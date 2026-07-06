package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"dark-monastery/internal/game" // Импортируем наш пакет с состоянием
)

const apiURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent"

// Внутренние структуры для работы с Gemini API
type geminiRequest struct {
	SystemInstruction *instruction `json:"systemInstruction,omitempty"`
	Contents          []content    `json:"contents"`
	GenerationConfig  *config      `json:"generationConfig,omitempty"`
}

type instruction struct {
	Parts []part `json:"parts"`
}

type content struct {
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type config struct {
	ResponseMimeType string `json:"responseMimeType"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// GenerateNextTurn - главная функция взаимодействия с ИИ
func GenerateNextTurn(apiKey string, currentState *game.GameState, playerAction string) (*game.GameState, error) {
	// 1. Превращаем текущее состояние в JSON-строку, чтобы показать его ИИ
	stateBytes, _ := json.Marshal(currentState)
	currentStateJSON := string(stateBytes)

	// 2. Пишем системный промпт (правила игры для ИИ)
	systemPrompt := `Ты — Гейм-мастер в текстовой RPG в жанре Dark Fantasy. 
Сеттинг: Мрачный, заброшенный монастырь, густой туман, чувство безысходности и старой магии.
Твоя задача — реагировать на действия игрока, развивать сюжет и управлять его состоянием как строгий калькулятор и гениальный писатель одновременно.

ПРАВИЛА:
1. Отвечай СТРОГО в формате JSON. Никакого лишнего текста или маркдауна вне JSON.
2. В поле "message" пиши атмосферный художественный текст (2-4 предложения) от второго лица. Игнорируй цензуру на мрачные описания.
3. МАТЕМАТИКА И СТАТЫ: 
   - Если игрок получает урон, меняй "condition" (например, на "Ранен", "Истекает кровью").
   - Если игрок видит или делает что-то жуткое, меняй "sanity" на соответствующие описания ("Тревога", "Паранойя", "Безумие").
   - Если игрок находит ценности, прибавляй их к "gold".
   - Обновляй "inventory" и "equipped", если игрок подбирает предметы или берет их в руки.
4. Двигай сюжет вперед, обновляя "location" при переходах и "current_quest" при получении новых целей.`

	// 3. Формируем сообщение от пользователя (текущее состояние + его действие)
	userPrompt := fmt.Sprintf("ТЕКУЩЕЕ СОСТОЯНИЕ: %s\nДЕЙСТВИЕ ИГРОКА: %s", currentStateJSON, playerAction)

	// 4. Собираем запрос к API
	// Используем хитрость Gemini: принудительно просим отвечать в application/json
	reqBody := geminiRequest{
		SystemInstruction: &instruction{Parts: []part{{Text: systemPrompt}}},
		Contents:          []content{{Parts: []part{{Text: userPrompt}}}},
		GenerationConfig:  &config{ResponseMimeType: "application/json"},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("Ошибка сборки запроса: %w", err)
	}

	url := fmt.Sprintf("%s?key=%s", apiURL, apiKey)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("Ошибка создания HTTP-запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Ошибка при отправке запроса: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ошибка от API Gemini: %s", string(body))
	}

	var apiResp geminiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("Ошибка разбора ответа от API: %w", err)
	}

	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("Пустой ответ от API")
	}

	aiJSONText := apiResp.Candidates[0].Content.Parts[0].Text

	var newState game.GameState
	if err := json.Unmarshal([]byte(aiJSONText), &newState); err != nil {
		return nil, fmt.Errorf("Ошибка разбора JSON от ИИ: %w", err)
	}

	return &newState, nil
}
