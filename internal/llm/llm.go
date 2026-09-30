package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Polisher tightens a raw templated prompt using an LLM.
type Polisher interface {
	Polish(ctx context.Context, rawPrompt string) (string, error)
}

// OllamaPolisher polishes prompts using a local Ollama server's
// /api/generate endpoint.
type OllamaPolisher struct {
	Host   string // e.g. "http://localhost:11434"
	Model  string
	Client *http.Client
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

const polishInstruction = "Tighten and polish the following prompt for clarity and brevity. Return only the polished prompt text, with no commentary:\n\n"

// Polish sends rawPrompt to the configured Ollama server and returns the
// polished text.
func (p OllamaPolisher) Polish(ctx context.Context, rawPrompt string) (string, error) {
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}

	body, err := json.Marshal(ollamaRequest{
		Model:  p.Model,
		Prompt: polishInstruction + rawPrompt,
		Stream: false,
	})
	if err != nil {
		return "", fmt.Errorf("encoding ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Host+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("building ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling ollama at %s: %w", p.Host, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decoding ollama response: %w", err)
	}
	trimmed := strings.TrimSpace(parsed.Response)
	if trimmed == "" {
		return "", fmt.Errorf("ollama returned an empty response")
	}

	return trimmed, nil
}
