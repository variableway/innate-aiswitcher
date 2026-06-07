// Package httpcheck provides connectivity and model-listing checks
// against LLM provider endpoints. It supports OpenAI-compatible,
// OpenAI Responses, and Anthropic protocol request shapes.
package httpcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/variableway/innate-aiswitcher/internal/store"
)

type Result struct {
	OK         bool   `json:"ok"`
	StatusCode int    `json:"status_code"`
	Endpoint   string `json:"endpoint"`
	Message    string `json:"message"`
}

type ModelsResult struct {
	OK         bool     `json:"ok"`
	StatusCode int      `json:"status_code"`
	Endpoint   string   `json:"endpoint"`
	Models     []string `json:"models"`
	Message    string   `json:"message"`
}

func CheckProvider(ctx context.Context, client *http.Client, provider store.Provider, model string) (Result, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if model == "" {
		model = provider.DefaultModel
	}
	if model == "" {
		return Result{}, fmt.Errorf("provider %s has no default model; set provider.default_model or pass --model", provider.Slug)
	}

	endpoint, body, headers := requestFor(provider, model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return Result{}, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	for key, value := range provider.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return Result{Endpoint: endpoint, Message: err.Error()}, err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	message := strings.TrimSpace(string(bodyBytes))
	if len(message) > 500 {
		message = message[:500]
	}

	return Result{
		OK:         resp.StatusCode >= 200 && resp.StatusCode < 300,
		StatusCode: resp.StatusCode,
		Endpoint:   endpoint,
		Message:    message,
	}, nil
}

func ListModels(ctx context.Context, client *http.Client, provider store.Provider) (ModelsResult, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	endpoint := endpointURL(provider, "models", modelsFallback(provider))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ModelsResult{}, err
	}
	for key, value := range authHeaders(provider) {
		req.Header.Set(key, value)
	}
	for key, value := range provider.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return ModelsResult{Endpoint: endpoint, Message: err.Error()}, err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	message := strings.TrimSpace(string(bodyBytes))
	if len(message) > 500 {
		message = message[:500]
	}

	return ModelsResult{
		OK:         resp.StatusCode >= 200 && resp.StatusCode < 300,
		StatusCode: resp.StatusCode,
		Endpoint:   endpoint,
		Models:     extractModelIDs(bodyBytes),
		Message:    message,
	}, nil
}

func requestFor(provider store.Provider, model string) (string, []byte, map[string]string) {
	headers := map[string]string{"Content-Type": "application/json"}
	for key, value := range authHeaders(provider) {
		headers[key] = value
	}

	switch provider.APIProtocol {
	case "anthropic":
		endpoint := endpointURL(provider, "messages", "/v1/messages")
		body, _ := json.Marshal(map[string]interface{}{
			"model":      model,
			"max_tokens": 8,
			"messages": []map[string]string{{
				"role":    "user",
				"content": "Reply with ok.",
			}},
		})
		return endpoint, body, headers
	case "openai_responses":
		endpoint := endpointURL(provider, "responses", "/responses")
		body, _ := json.Marshal(map[string]interface{}{
			"model":             model,
			"input":             "Reply with ok.",
			"max_output_tokens": 8,
		})
		return endpoint, body, headers
	default:
		endpoint := endpointURL(provider, "chat_completions", "/chat/completions")
		body, _ := json.Marshal(map[string]interface{}{
			"model": model,
			"messages": []map[string]string{{
				"role":    "user",
				"content": "Reply with ok.",
			}},
			"max_tokens": 8,
		})
		return endpoint, body, headers
	}
}

func authHeaders(provider store.Provider) map[string]string {
	headers := map[string]string{"Authorization": "Bearer " + provider.APIKey}
	if provider.APIProtocol == "anthropic" {
		headers["x-api-key"] = provider.APIKey
		headers["anthropic-version"] = "2023-06-01"
	}
	return headers
}

func modelsFallback(provider store.Provider) string {
	if provider.APIProtocol == "anthropic" {
		return "/v1/models"
	}
	return "/models"
}

func endpointURL(provider store.Provider, key string, fallbackSuffix string) string {
	base := strings.TrimRight(provider.BaseURL, "/")
	endpoint := strings.TrimSpace(provider.Endpoints[key])
	if endpoint == "" {
		if strings.HasSuffix(base, fallbackSuffix) {
			return base
		}
		endpoint = fallbackSuffix
	}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}
	if strings.HasPrefix(endpoint, "/") {
		return base + endpoint
	}
	return base + "/" + endpoint
}

func Format(result Result) string {
	status := "failed"
	if result.OK {
		status = "ok"
	}
	return fmt.Sprintf("%s status=%d endpoint=%s\n%s", status, result.StatusCode, result.Endpoint, result.Message)
}

func FormatModels(result ModelsResult) string {
	status := "failed"
	if result.OK {
		status = "ok"
	}
	if len(result.Models) == 0 {
		return fmt.Sprintf("%s status=%d endpoint=%s\n%s", status, result.StatusCode, result.Endpoint, result.Message)
	}
	return fmt.Sprintf("%s status=%d endpoint=%s\n%s", status, result.StatusCode, result.Endpoint, strings.Join(result.Models, "\n"))
}

func extractModelIDs(body []byte) []string {
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Models []struct {
			ID string `json:"id"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	models := make([]string, 0, len(payload.Data)+len(payload.Models))
	seen := map[string]bool{}
	for _, item := range payload.Data {
		if item.ID != "" && !seen[item.ID] {
			models = append(models, item.ID)
			seen[item.ID] = true
		}
	}
	for _, item := range payload.Models {
		if item.ID != "" && !seen[item.ID] {
			models = append(models, item.ID)
			seen[item.ID] = true
		}
	}
	return models
}
