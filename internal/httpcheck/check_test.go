package httpcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestFormat(t *testing.T) {
	r := Result{OK: true, StatusCode: 200, Endpoint: "/test", Message: "ok"}
	if !strings.Contains(Format(r), "ok status=200") {
		t.Fatalf("unexpected format: %s", Format(r))
	}

	r = Result{OK: false, StatusCode: 500, Endpoint: "/fail", Message: "error"}
	if !strings.Contains(Format(r), "failed status=500") {
		t.Fatalf("unexpected format: %s", Format(r))
	}
}

func TestFormatModels(t *testing.T) {
	r := ModelsResult{OK: true, StatusCode: 200, Endpoint: "/models", Models: []string{"a", "b"}, Message: "ok"}
	formatted := FormatModels(r)
	if !strings.Contains(formatted, "ok status=200") {
		t.Fatalf("unexpected format: %s", formatted)
	}
	if !strings.Contains(formatted, "a") || !strings.Contains(formatted, "b") {
		t.Fatalf("expected model list in output: %s", formatted)
	}

	r = ModelsResult{OK: false, StatusCode: 404, Endpoint: "/models", Models: nil, Message: "not found"}
	formatted = FormatModels(r)
	if !strings.Contains(formatted, "failed status=404") {
		t.Fatalf("unexpected format: %s", formatted)
	}
}

func TestEndpointURL(t *testing.T) {
	tests := []struct {
		name     string
		provider store.Provider
		key      string
		fallback string
		want     string
	}{
		{
			name:     "empty endpoint uses fallback",
			provider: store.Provider{BaseURL: "https://api.example.com/v1"},
			key:      "chat_completions",
			fallback: "/chat/completions",
			want:     "https://api.example.com/v1/chat/completions",
		},
		{
			name:     "configured relative endpoint",
			provider: store.Provider{BaseURL: "https://api.example.com", Endpoints: map[string]string{"chat_completions": "/custom/chat"}},
			key:      "chat_completions",
			fallback: "/chat/completions",
			want:     "https://api.example.com/custom/chat",
		},
		{
			name:     "configured absolute endpoint",
			provider: store.Provider{BaseURL: "https://api.example.com", Endpoints: map[string]string{"chat_completions": "https://other.example.com/chat"}},
			key:      "chat_completions",
			fallback: "/chat/completions",
			want:     "https://other.example.com/chat",
		},
		{
			name:     "base already ends with fallback",
			provider: store.Provider{BaseURL: "https://api.example.com/v1/chat/completions"},
			key:      "chat_completions",
			fallback: "/chat/completions",
			want:     "https://api.example.com/v1/chat/completions",
		},
		{
			name:     "endpoint without leading slash",
			provider: store.Provider{BaseURL: "https://api.example.com/v1", Endpoints: map[string]string{"chat_completions": "custom/chat"}},
			key:      "chat_completions",
			fallback: "/chat/completions",
			want:     "https://api.example.com/v1/custom/chat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := endpointURL(tt.provider, tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("endpointURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAuthHeaders(t *testing.T) {
	openai := store.Provider{APIKey: "sk-test", APIProtocol: "openai_chat"}
	h := authHeaders(openai)
	if h["Authorization"] != "Bearer sk-test" {
		t.Fatalf("unexpected authorization: %s", h["Authorization"])
	}
	if h["x-api-key"] != "" {
		t.Fatal("openai provider should not set x-api-key")
	}

	anthropic := store.Provider{APIKey: "anthropic-key", APIProtocol: "anthropic"}
	h = authHeaders(anthropic)
	if h["x-api-key"] != "anthropic-key" {
		t.Fatalf("unexpected x-api-key: %s", h["x-api-key"])
	}
	if h["anthropic-version"] != "2023-06-01" {
		t.Fatalf("unexpected anthropic-version: %s", h["anthropic-version"])
	}
}

func TestModelsFallback(t *testing.T) {
	if got := modelsFallback(store.Provider{APIProtocol: "anthropic"}); got != "/v1/models" {
		t.Fatalf("unexpected anthropic fallback: %s", got)
	}
	if got := modelsFallback(store.Provider{APIProtocol: "openai_chat"}); got != "/models" {
		t.Fatalf("unexpected openai fallback: %s", got)
	}
}

func TestExtractModelIDs(t *testing.T) {
	body := []byte(`{"data":[{"id":"gpt-4"},{"id":"gpt-3.5"}],"models":[{"id":"custom-model"}]}`)
	ids := extractModelIDs(body)
	if len(ids) != 3 {
		t.Fatalf("expected 3 models, got %d: %v", len(ids), ids)
	}
	if ids[0] != "gpt-4" || ids[1] != "gpt-3.5" || ids[2] != "custom-model" {
		t.Fatalf("unexpected ids: %v", ids)
	}

	// Duplicates should be deduplicated.
	body = []byte(`{"data":[{"id":"gpt-4"},{"id":"gpt-4"}]}`)
	ids = extractModelIDs(body)
	if len(ids) != 1 {
		t.Fatalf("expected 1 unique model, got %d: %v", len(ids), ids)
	}

	// Invalid JSON returns nil.
	ids = extractModelIDs([]byte("not json"))
	if len(ids) != 0 {
		t.Fatalf("expected empty for invalid json, got %v", ids)
	}

	// Empty data returns empty.
	ids = extractModelIDs([]byte(`{"data":[]}`))
	if len(ids) != 0 {
		t.Fatalf("expected empty for empty data, got %v", ids)
	}
}
