package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	embeddingModel = "gemini-embedding-001"
	embeddingURL   = "https://generativelanguage.googleapis.com/v1beta/models/" + embeddingModel + ":embedContent"
	embeddingDim   = 768
)

// EmbeddingClient генерирует эмбеддинги текста через Gemini Embedding API.
type EmbeddingClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewEmbeddingClient создаёт клиент для генерации эмбеддингов.
func NewEmbeddingClient(apiKey string) *EmbeddingClient {
	return &EmbeddingClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// --- Типы запроса/ответа ---

type embedRequest struct {
	Model                string       `json:"model"`
	Content              embedContent `json:"content"`
	OutputDimensionality int          `json:"outputDimensionality,omitempty"`
}

type embedContent struct {
	Parts []embedPart `json:"parts"`
}

type embedPart struct {
	Text string `json:"text"`
}

type embedResponse struct {
	Embedding *embedResult `json:"embedding"`
}

type embedResult struct {
	Values []float32 `json:"values"`
}

// Embed генерирует вектор для одного текста.
// Возвращает float32 вектор размерностью 768.
func (e *EmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("пустой текст для эмбеддинга")
	}

	reqBody := embedRequest{
		Model:                "models/" + embeddingModel,
		Content:              embedContent{Parts: []embedPart{{Text: text}}},
		OutputDimensionality: embeddingDim,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка сборки запроса embedding: %w", err)
	}

	url := fmt.Sprintf("%s?key=%s", embeddingURL, e.apiKey)

	// Экспоненциальный backoff: до 3 повторов на 429 и 503
	var body []byte
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("ошибка создания HTTP-запроса embedding: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := e.httpClient.Do(req)
		if err != nil {
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt*2) * time.Second)
				continue
			}
			return nil, fmt.Errorf("ошибка сети embedding: %w", err)
		}

		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			break
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			if attempt < maxRetries {
				sleepTime := time.Duration(attempt*2) * time.Second
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(sleepTime):
				}
				continue
			}
		}

		return nil, fmt.Errorf("ошибка Embedding API (статус %d): %s", resp.StatusCode, string(body))
	}

	var apiResp embedResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("ошибка разбора ответа embedding: %w", err)
	}

	if apiResp.Embedding == nil || len(apiResp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("пустой ответ от Embedding API")
	}

	return apiResp.Embedding.Values, nil
}

// EmbedBatch генерирует эмбеддинги для пакета текстов.
// Вызывает Embed для каждого текста последовательно (API не поддерживает batch в embedContent).
func (e *EmbeddingClient) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		vec, err := e.Embed(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("ошибка batch embedding [%d]: %w", i, err)
		}
		results[i] = vec
	}
	return results, nil
}
