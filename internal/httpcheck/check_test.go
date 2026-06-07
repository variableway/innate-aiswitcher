package httpcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/variableway/innate-aiswitcher/internal/store"
)

func TestCheckProviderOpenAICompatible(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("unexpected authorization header: %s", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["model"]; got != "gpt-test" {
			t.Fatalf("unexpected model: %v", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ok"}`))
	}))
	defer server.Close()

	provider := store.Provider{
		Slug:        "openai-compatible",
		BaseURL:     server.URL + "/v1",
		APIKey:      "sk-test",
		APIProtocol: "openai_chat",
	}
	result, err := CheckProvider(context.Background(), server.Client(), provider, "gpt-test")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.StatusCode != http.StatusOK {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCheckProviderAnthropic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "anthropic-key" {
			t.Fatalf("unexpected x-api-key header: %s", got)
		}
		if got := r.Header.Get("anthropic-version"); got == "" {
			t.Fatal("expected anthropic-version header")
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["model"]; got != "claude-test" {
			t.Fatalf("unexpected model: %v", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"type":"message"}`))
	}))
	defer server.Close()

	provider := store.Provider{
		Slug:        "anthropic-compatible",
		BaseURL:     server.URL,
		APIKey:      "anthropic-key",
		APIProtocol: "anthropic",
	}
	result, err := CheckProvider(context.Background(), server.Client(), provider, "claude-test")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.StatusCode != http.StatusOK {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCheckProviderUsesConfiguredEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/custom/chat" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"ok"}`))
	}))
	defer server.Close()

	provider := store.Provider{
		Slug:         "custom-endpoint",
		BaseURL:      server.URL + "/api",
		APIKey:       "sk-test",
		APIProtocol:  "openai_chat",
		DefaultModel: "gpt-test",
		Endpoints:    map[string]string{"chat_completions": "/custom/chat"},
	}
	result, err := CheckProvider(context.Background(), server.Client(), provider, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCheckProviderOpenAIResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if got := body["input"]; got == nil {
			t.Fatalf("expected responses input body: %+v", body)
		}
		_, _ = w.Write([]byte(`{"id":"ok"}`))
	}))
	defer server.Close()

	provider := store.Provider{
		Slug:         "responses",
		BaseURL:      server.URL + "/v1",
		APIKey:       "sk-test",
		APIProtocol:  "openai_responses",
		DefaultModel: "gpt-test",
	}
	result, err := CheckProvider(context.Background(), server.Client(), provider, "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("unexpected authorization header: %s", got)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-a"},{"id":"gpt-b"}]}`))
	}))
	defer server.Close()

	provider := store.Provider{
		Slug:        "models",
		BaseURL:     server.URL + "/v1",
		APIKey:      "sk-test",
		APIProtocol: "openai_chat",
	}
	result, err := ListModels(context.Background(), server.Client(), provider)
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || len(result.Models) != 2 || result.Models[0] != "gpt-a" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCheckProviderRequiresProviderDefaultModel(t *testing.T) {
	provider := store.Provider{
		Slug:        "missing-model",
		BaseURL:     "https://api.example.test/v1",
		APIKey:      "sk-test",
		APIProtocol: "openai_chat",
	}
	result, err := CheckProvider(context.Background(), nil, provider, "")
	if err == nil {
		t.Fatal("expected missing default model error")
	}
	if result.Endpoint != "" {
		t.Fatalf("request should not be built without a model: %+v", result)
	}
}
