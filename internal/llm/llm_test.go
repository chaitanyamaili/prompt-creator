package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaPolisherSendsPromptAndReturnsPolishedText(t *testing.T) {
	var gotBody ollamaRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Fatalf("expected path /api/generate, got %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		json.NewEncoder(w).Encode(ollamaResponse{Response: "Polished prompt"})
	}))
	defer server.Close()

	p := OllamaPolisher{Host: server.URL, Model: "llama3"}
	got, err := p.Polish(context.Background(), "Goal: write a poem")
	if err != nil {
		t.Fatalf("Polish returned error: %v", err)
	}
	if got != "Polished prompt" {
		t.Fatalf("got %q, want %q", got, "Polished prompt")
	}
	if gotBody.Model != "llama3" {
		t.Fatalf("expected model %q sent, got %q", "llama3", gotBody.Model)
	}
	if !strings.Contains(gotBody.Prompt, "Goal: write a poem") {
		t.Fatalf("expected raw prompt embedded in request, got %q", gotBody.Prompt)
	}
}

func TestOllamaPolisherReturnsErrorOnNonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	defer server.Close()

	p := OllamaPolisher{Host: server.URL, Model: "llama3"}
	_, err := p.Polish(context.Background(), "x")
	if err == nil {
		t.Fatalf("expected error for non-200 response")
	}
}

func TestOllamaPolisherReturnsErrorWhenUnreachable(t *testing.T) {
	p := OllamaPolisher{Host: "http://127.0.0.1:1", Model: "llama3"}
	_, err := p.Polish(context.Background(), "x")
	if err == nil {
		t.Fatalf("expected error when host is unreachable")
	}
}

func TestOllamaPolisherReturnsErrorOnMalformedHost(t *testing.T) {
	p := OllamaPolisher{Host: "not-a-url", Model: "llama3"}
	_, err := p.Polish(context.Background(), "x")
	if err == nil {
		t.Fatalf("expected error for malformed host")
	}
}

func TestOllamaPolisherTrimsWhitespaceFromResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ollamaResponse{Response: "\n\n  Polished prompt  \n\n"})
	}))
	defer server.Close()

	p := OllamaPolisher{Host: server.URL, Model: "llama3"}
	got, err := p.Polish(context.Background(), "x")
	if err != nil {
		t.Fatalf("Polish returned error: %v", err)
	}
	if got != "Polished prompt" {
		t.Fatalf("got %q, want %q", got, "Polished prompt")
	}
}

func TestOllamaPolisherReturnsErrorWhenResponseIsWhitespaceOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ollamaResponse{Response: "   \n  "})
	}))
	defer server.Close()

	p := OllamaPolisher{Host: server.URL, Model: "llama3"}
	_, err := p.Polish(context.Background(), "x")
	if err == nil {
		t.Fatalf("expected error for whitespace-only response")
	}
}
