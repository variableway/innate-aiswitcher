// Package market fetches the public lobehub model-market catalog over its
// tRPC HTTP endpoints (no auth required) and persists snapshots to a JSON
// file. The catalog is a reference data source: its models can be imported
// into vendor providers, where they share the provider's single API key.
package market

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// DefaultBaseURL is the public lobehub tRPC lambda endpoint.
const DefaultBaseURL = "https://app.lobehub.com/trpc/lambda"

// DefaultLocale is the catalog locale used for localized description fields.
const DefaultLocale = "zh-CN"

// Model mirrors one entry of the lobehub market model list. Identifiers are
// unique across the catalog; Providers lists every vendor/gateway serving
// the model (the first entries of a gateway-style catalog such as higress,
// aihubmix or openrouter).
type Model struct {
	ID                  string          `json:"id"`
	Identifier          string          `json:"identifier"`
	DisplayName         string          `json:"displayName"`
	Type                string          `json:"type,omitempty"`
	Category            string          `json:"category"`
	ProviderID          string          `json:"providerId"`
	Providers           []string        `json:"providers,omitempty"`
	ProviderCount       int             `json:"providerCount,omitempty"`
	ContextWindowTokens int             `json:"contextWindowTokens,omitempty"`
	Abilities           map[string]bool `json:"abilities,omitempty"`
	Pricing             json.RawMessage `json:"pricing,omitempty"`
	KnowledgeCutoff     string          `json:"knowledgeCutoff,omitempty"`
	Generation          string          `json:"generation,omitempty"`
	Family              string          `json:"family,omitempty"`
	Source              string          `json:"source,omitempty"`
	Enabled             *bool           `json:"enabled,omitempty"`
	ReleasedAt          string          `json:"releasedAt,omitempty"`
	Description         string          `json:"description,omitempty"`
}

// IsEnabled treats a missing enabled flag as enabled (the catalog defaults
// to true and only marks deprecated/offline entries explicitly).
func (m Model) IsEnabled() bool {
	return m.Enabled == nil || *m.Enabled
}

// Category is one vendor entry of the catalog's category list.
type Category struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// ModelsPage is one page of the paginated model list.
type ModelsPage struct {
	Items       []Model `json:"items"`
	CurrentPage int     `json:"currentPage"`
	PageSize    int     `json:"pageSize"`
	TotalCount  int     `json:"totalCount"`
	TotalPages  int     `json:"totalPages"`
}

// ListOptions controls one model-list query. Zero Page/PageSize mean the
// client defaults (page 1, 100 per page — the largest page the API accepts).
type ListOptions struct {
	Page     int
	PageSize int
	Category string
	Query    string
	Locale   string
}

// Client talks to a lobehub-compatible market API. The zero value is not
// usable; use NewClient and override fields as needed.
type Client struct {
	BaseURL string
	Locale  string
	HTTP    *http.Client
}

// NewClient returns a client pointed at the public lobehub catalog.
func NewClient() *Client {
	return &Client{
		BaseURL: DefaultBaseURL,
		Locale:  DefaultLocale,
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// batchInput is the tRPC batch envelope for a single-procedure GET query:
// {"0":{"json":{...},"meta":{"values":{"field":["undefined"]},"v":1}}}.
// Fields listed in meta.values carry a JSON null in the json object.
type batchInput struct {
	Zero struct {
		JSON map[string]any  `json:"json"`
		Meta *batchInputMeta `json:"meta,omitempty"`
	} `json:"0"`
}

type batchInputMeta struct {
	Values map[string][]string `json:"values"`
	V      int                 `json:"v"`
}

// batchEnvelope is the tRPC batch response wrapper; the procedure payload
// sits at [0].result.data.json (superjson-encoded).
type batchEnvelope []struct {
	Result struct {
		Data struct {
			JSON json.RawMessage `json:"json"`
		} `json:"data"`
	} `json:"result"`
}

// query runs one batched tRPC GET procedure and decodes its data.json into
// out. undefinedFields lists keys whose value is null.
func (c *Client) query(ctx context.Context, procedure string, params map[string]any, undefinedFields []string, out any) error {
	input := batchInput{}
	input.Zero.JSON = params
	if len(undefinedFields) > 0 {
		input.Zero.Meta = &batchInputMeta{Values: map[string][]string{}, V: 1}
		for _, field := range undefinedFields {
			input.Zero.Meta.Values[field] = []string{"undefined"}
		}
	}
	encoded, err := json.Marshal(input)
	if err != nil {
		return err
	}
	endpoint := c.BaseURL + "/" + procedure + "?batch=1&input=" + url.QueryEscape(string(encoded))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "innate-aiswitcher")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("market %s: unexpected status %d", procedure, resp.StatusCode)
	}

	var envelope batchEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("market %s: invalid response: %w", procedure, err)
	}
	if len(envelope) == 0 {
		return fmt.Errorf("market %s: empty batch response", procedure)
	}
	if len(envelope[0].Result.Data.JSON) == 0 {
		return fmt.Errorf("market %s: missing result data", procedure)
	}
	if err := json.Unmarshal(envelope[0].Result.Data.JSON, out); err != nil {
		return fmt.Errorf("market %s: decode result: %w", procedure, err)
	}
	return nil
}

// FetchCategories returns the vendor categories with their model counts.
func (c *Client) FetchCategories(ctx context.Context) ([]Category, error) {
	var categories []Category
	err := c.query(ctx, "market.getModelCategories", map[string]any{"q": nil}, []string{"q"}, &categories)
	return categories, err
}

// FetchModelsPage returns one page of the model list.
func (c *Client) FetchModelsPage(ctx context.Context, opts ListOptions) (*ModelsPage, error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.PageSize <= 0 {
		opts.PageSize = 100
	}
	locale := opts.Locale
	if locale == "" {
		locale = c.Locale
	}

	params := map[string]any{"page": opts.Page, "pageSize": opts.PageSize, "locale": locale}
	undefined := []string{}
	for field, value := range map[string]*string{
		"category": &opts.Category,
		"order":    nil,
		"q":        &opts.Query,
		"sort":     nil,
	} {
		if value == nil || *value == "" {
			params[field] = nil
			undefined = append(undefined, field)
		} else {
			params[field] = *value
		}
	}

	var page ModelsPage
	if err := c.query(ctx, "market.getModelList", params, undefined, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// FetchAllModels pages through the whole catalog (pageSize 100) and returns
// every model. Requests are spaced 200ms apart to stay gentle on the API.
func (c *Client) FetchAllModels(ctx context.Context) (*ModelsPage, []Model, error) {
	first, err := c.FetchModelsPage(ctx, ListOptions{Page: 1, PageSize: 100})
	if err != nil {
		return nil, nil, err
	}
	items := make([]Model, 0, first.TotalCount)
	items = append(items, first.Items...)
	for page := 2; page <= first.TotalPages; page++ {
		select {
		case <-ctx.Done():
			return first, items, ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
		result, err := c.FetchModelsPage(ctx, ListOptions{Page: page, PageSize: 100})
		if err != nil {
			return first, items, err
		}
		items = append(items, result.Items...)
	}
	return first, items, nil
}

// CategoriesFromModels derives the vendor category summary from a fetched
// model list (sorted by category).
func CategoriesFromModels(items []Model) []Category {
	counts := map[string]int{}
	for _, item := range items {
		if item.Category != "" {
			counts[item.Category]++
		}
	}
	categories := make([]Category, 0, len(counts))
	for name, count := range counts {
		categories = append(categories, Category{Category: name, Count: count})
	}
	for i := 1; i < len(categories); i++ {
		for j := i; j > 0 && categories[j-1].Category > categories[j].Category; j-- {
			categories[j-1], categories[j] = categories[j], categories[j-1]
		}
	}
	return categories
}
