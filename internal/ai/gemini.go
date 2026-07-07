package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"dark-monastery/internal/game"
)

const apiURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent"

// GeminiClient реализует game.AIClient через Google Gemini API.
type GeminiClient struct {
	apiKey        string
	httpClient    *http.Client
	lore          *game.LoreBook
	history       game.HistoryProvider // legacy
	memoryContext string               // RAG: контекст от MemoryManager
}

// NewGeminiClient создаёт новый экземпляр GeminiClient.
func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

// SetLore задаёт лор мира для инжекции в промпт.
func (g *GeminiClient) SetLore(lore *game.LoreBook) { g.lore = lore }

// SetHistory задаёт провайдер истории диалога для инжекции в промпт (legacy).
func (g *GeminiClient) SetHistory(h game.HistoryProvider) { g.history = h }

// SetMemoryContext задаёт контекст памяти от RAG-системы.
// Вызывается перед каждым GenerateNextTurn.
func (g *GeminiClient) SetMemoryContext(ctx string) { g.memoryContext = ctx }

// --- Внутренние типы для работы с Gemini API ---

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

// GenerateNextTurn отправляет состояние и действие игрока в Gemini,
// возвращает обновлённое состояние.
func (g *GeminiClient) GenerateNextTurn(currentState *game.GameState, playerAction string) (*game.GameState, error) {
	// Собираем системный промпт с лором, памятью и историей
	promptCtx := &game.PromptContext{
		Lore:          g.lore,
		History:       g.history,
		MemoryContext: g.memoryContext,
	}
	systemPrompt := game.BuildSystemPrompt(promptCtx)

	// Сбрасываем memoryContext после использования
	g.memoryContext = ""

	userPrompt := game.BuildUserPrompt(currentState, playerAction)

	reqBody := geminiRequest{
		SystemInstruction: &instruction{Parts: []part{{Text: systemPrompt}}},
		Contents:          []content{{Parts: []part{{Text: userPrompt}}}},
		GenerationConfig:  &config{ResponseMimeType: "application/json"},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("Ошибка сборки запроса: %w", err)
	}

	url := fmt.Sprintf("%s?key=%s", apiURL, g.apiKey)

	// Экспоненциальный backoff: до 3 повторов на 429 и 503
	var body []byte
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("Ошибка создания HTTP-запроса: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := g.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Ошибка сети: %w", err)
		}

		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			break
		}

		if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
			if attempt < maxRetries {
				sleepTime := time.Duration(attempt*2) * time.Second
				fmt.Printf("\n[Сервера Google перегружены. Авто-повтор %d/%d через %v...]\n", attempt, maxRetries, sleepTime)
				time.Sleep(sleepTime)
				continue
			}
		}

		return nil, fmt.Errorf("Ошибка API (статус %d): %s", resp.StatusCode, string(body))
	}

	var apiResp geminiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("Ошибка разбора ответа от API: %w", err)
	}

	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("Пустой ответ от API")
	}

	var newState game.GameState
	if err := json.Unmarshal([]byte(apiResp.Candidates[0].Content.Parts[0].Text), &newState); err != nil {
		return nil, fmt.Errorf("Ошибка разбора JSON от ИИ: %w", err)
	}

	return &newState, nil
}
