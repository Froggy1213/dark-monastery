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

// GeminiClient implements game.AIClient via the Google Gemini API.
type GeminiClient struct {
	apiKey        string
	httpClient    *http.Client
	lore          *game.LoreBook
	history       game.HistoryProvider // legacy
	memoryContext string               // RAG: context from MemoryManager
}

// NewGeminiClient creates a new GeminiClient instance.
func NewGeminiClient(apiKey string) *GeminiClient {
	return &GeminiClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

// SetLore sets the world lore for injection into the prompt.
func (g *GeminiClient) SetLore(lore *game.LoreBook) { g.lore = lore }

// SetHistory sets the dialogue history provider for injection into the prompt (legacy).
func (g *GeminiClient) SetHistory(h game.HistoryProvider) { g.history = h }

// SetMemoryContext sets the memory context from the RAG system.
// Called before each GenerateNextTurn.
func (g *GeminiClient) SetMemoryContext(ctx string) { g.memoryContext = ctx }

// --- Internal types for working with the Gemini API ---

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

// GenerateNextTurn sends the state and player action to Gemini,
// returns the updated state.
func (g *GeminiClient) GenerateNextTurn(currentState *game.GameState, playerAction string) (*game.GameState, error) {
	// Assemble the system prompt with lore, memory, and history
	promptCtx := &game.PromptContext{
		Lore:          g.lore,
		History:       g.history,
		MemoryContext: g.memoryContext,
	}
	systemPrompt := game.BuildSystemPrompt(promptCtx)

	// Reset memoryContext after use
	g.memoryContext = ""

	userPrompt := game.BuildUserPrompt(currentState, playerAction)

	reqBody := geminiRequest{
		SystemInstruction: &instruction{Parts: []part{{Text: systemPrompt}}},
		Contents:          []content{{Parts: []part{{Text: userPrompt}}}},
		GenerationConfig:  &config{ResponseMimeType: "application/json"},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("request build error: %w", err)
	}

	url := fmt.Sprintf("%s?key=%s", apiURL, g.apiKey)

	// Exponential backoff: up to 3 retries on 429 and 503
	var body []byte
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("HTTP request creation error: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := g.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("network error: %w", err)
		}

		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			break
		}

		if resp.StatusCode == http.StatusServiceUnavailable || resp.StatusCode == http.StatusTooManyRequests {
			if attempt < maxRetries {
				sleepTime := time.Duration(attempt*2) * time.Second
				fmt.Printf("\n[Google servers overloaded. Auto-retry %d/%d in %v...]\n", attempt, maxRetries, sleepTime)
				time.Sleep(sleepTime)
				continue
			}
		}

		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp geminiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("API response parse error: %w", err)
	}

	if len(apiResp.Candidates) == 0 || len(apiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("Empty API response")
	}

	var newState game.GameState
	if err := json.Unmarshal([]byte(apiResp.Candidates[0].Content.Parts[0].Text), &newState); err != nil {
		return nil, fmt.Errorf("AI JSON parse error: %w", err)
	}

	return &newState, nil
}
