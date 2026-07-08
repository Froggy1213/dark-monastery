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

// EmbeddingClient generates text embeddings via the Gemini Embedding API.
type EmbeddingClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewEmbeddingClient creates a client for generating embeddings.
func NewEmbeddingClient(apiKey string) *EmbeddingClient {
	return &EmbeddingClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// --- Request/response types ---

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

// Embed generates a vector for a single text.
// Returns a 768-dimensional float32 vector.
func (e *EmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text for embedding")
	}

	reqBody := embedRequest{
		Model:                "models/" + embeddingModel,
		Content:              embedContent{Parts: []embedPart{{Text: text}}},
		OutputDimensionality: embeddingDim,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("embedding request build error: %w", err)
	}

	url := fmt.Sprintf("%s?key=%s", embeddingURL, e.apiKey)

	// Exponential backoff: up to 3 retries on 429 and 503
	var body []byte
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return nil, fmt.Errorf("embedding HTTP request creation error: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := e.httpClient.Do(req)
		if err != nil {
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt*2) * time.Second)
				continue
			}
			return nil, fmt.Errorf("embedding network error: %w", err)
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

		return nil, fmt.Errorf("Embedding API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp embedResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("embedding response parse error: %w", err)
	}

	if apiResp.Embedding == nil || len(apiResp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("empty response from Embedding API")
	}

	return apiResp.Embedding.Values, nil
}

// EmbedBatch generates embeddings for a batch of texts.
// Calls Embed for each text sequentially (API does not support batch in embedContent).
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
			return nil, fmt.Errorf("batch embedding error [%d]: %w", i, err)
		}
		results[i] = vec
	}
	return results, nil
}
