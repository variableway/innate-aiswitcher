# Go + PocketBase + CLI + REST API — Patterns Reference

## 1. Complete Migration Example

```go
package migrations

import (
    "github.com/pocketbase/pocketbase/core"
    "github.com/pocketbase/pocketbase/migrations"
)

func init() {
    migrations.Register(
        // UP
        func(txApp core.App) error {
            // --- Collection: providers ---
            providers := core.NewBaseCollection("providers")
            providers.ListRule = strPtr("")
            providers.ViewRule = strPtr("")
            providers.Fields.Add(
                &core.TextField{Name: "slug", Required: true, Presentable: true},
                &core.TextField{Name: "name", Required: true},
                &core.URLField{Name: "base_url", Required: true},
                &core.TextField{Name: "api_key", Hidden: true},
                &core.SelectField{
                    Name: "api_protocol", Required: true,
                    Values: []string{"protocol_a", "protocol_b", "generic"},
                },
                &core.TextField{Name: "default_model"},
                &core.JSONField{Name: "headers"},
                &core.JSONField{Name: "endpoints"},
                &core.BoolField{Name: "active"},
            )
            providers.AddIndex("idx_providers_slug", true, "slug", "")
            if err := txApp.Save(providers); err != nil {
                return err
            }

            // --- Collection: profiles (with relations) ---
            profiles := core.NewBaseCollection("profiles")
            profiles.ListRule = strPtr("")
            profiles.ViewRule = strPtr("")
            profiles.Fields.Add(
                &core.TextField{Name: "slug", Required: true, Presentable: true},
                &core.TextField{Name: "name", Required: true},
                &core.RelationField{
                    Name: "provider", Required: true,
                    CollectionId: providers.Id, CascadeDelete: true, MaxSelect: 1,
                },
                &core.TextField{Name: "model"},
                &core.JSONField{Name: "config_overrides"},
                &core.BoolField{Name: "is_default"},
            )
            profiles.AddIndex("idx_profiles_slug", true, "slug", "")
            if err := txApp.Save(profiles); err != nil {
                return err
            }

            return nil
        },
        // DOWN
        func(txApp core.App) error {
            for _, name := range []string{"profiles", "providers"} {
                coll, err := txApp.FindCollectionByNameOrId(name)
                if err == nil {
                    txApp.Delete(coll)
                }
            }
            return nil
        },
    )
}

func strPtr(s string) *string { return &s }
```

## 2. Full Store Implementation

```go
package store

import (
    "fmt"
    "strings"

    "github.com/pocketbase/dbx"
    "github.com/pocketbase/pocketbase/core"
)

type Store struct { app core.App }

func New(app core.App) *Store { return &Store{app: app} }

// --- Upsert ---
func (s *Store) UpsertProvider(input Provider) (*Provider, error) {
    input.Slug = normalizeSlug(input.Slug)
    if input.Slug == "" {
        return nil, fmt.Errorf("provider slug is required")
    }
    if input.Name == "" {
        input.Name = input.Slug
    }

    record, err := s.findRecordBySlug("providers", input.Slug)
    if err != nil {
        return nil, err
    }
    if record == nil {
        collection, err := s.app.FindCollectionByNameOrId("providers")
        if err != nil {
            return nil, err
        }
        record = core.NewRecord(collection)
    }

    record.Set("slug", input.Slug)
    record.Set("name", input.Name)
    record.Set("base_url", strings.TrimRight(input.BaseURL, "/"))
    if input.APIKey != "" {
        record.Set("api_key", input.APIKey)
    }
    record.Set("api_protocol", input.APIProtocol)
    record.Set("default_model", input.DefaultModel)
    record.Set("headers", nonNilMap(input.Headers))
    record.Set("endpoints", nonNilMap(input.Endpoints))
    record.Set("active", input.Active)

    if err := s.app.Save(record); err != nil {
        return nil, err
    }
    return recordToProvider(record), nil
}

// --- List ---
func (s *Store) ListProviders() ([]Provider, error) {
    records, err := s.app.FindRecordsByFilter("providers", "", "slug", 0, 0)
    if err != nil {
        return nil, err
    }
    providers := make([]Provider, 0, len(records))
    for _, record := range records {
        providers = append(providers, *recordToProvider(record))
    }
    return providers, nil
}

// --- Get ---
func (s *Store) GetProvider(slug string) (*Provider, error) {
    record, err := s.findRecordBySlug("providers", slug)
    if err != nil || record == nil {
        return nil, err
    }
    return recordToProvider(record), nil
}

// --- Delete ---
func (s *Store) DeleteProvider(slug string) error {
    record, err := s.findRecordBySlug("providers", slug)
    if err != nil || record == nil {
        return err
    }
    return s.app.Delete(record)
}

// --- Transaction ---
func (s *Store) RunInTransaction(fn func(*Store) error) error {
    return s.app.RunInTransaction(func(txApp core.App) error {
        return fn(New(txApp))
    })
}

// --- Helpers ---
func (s *Store) findRecordBySlug(collection, slug string) (*core.Record, error) {
    slug = normalizeSlug(slug)
    if slug == "" {
        return nil, nil
    }
    record, err := s.app.FindFirstRecordByFilter(
        collection, "slug={:slug}", dbx.Params{"slug": slug},
    )
    if err != nil {
        return nil, nil // not found is not an error
    }
    return record, nil
}

func normalizeSlug(value string) string {
    value = strings.TrimSpace(strings.ToLower(value))
    value = strings.ReplaceAll(value, "_", "-")
    return value
}

func nonNilMap(value map[string]string) map[string]string {
    if value == nil {
        return map[string]string{}
    }
    return value
}

// --- Record-to-Model ---
func recordToProvider(record *core.Record) *Provider {
    return &Provider{
        ID:           record.Id,
        Slug:         record.GetString("slug"),
        Name:         record.GetString("name"),
        BaseURL:      record.GetString("base_url"),
        APIKey:       record.GetString("api_key"),
        APIProtocol:  record.GetString("api_protocol"),
        DefaultModel: record.GetString("default_model"),
        Headers:      decodeJSONMap[map[string]string](record.Get("headers")),
        Endpoints:    decodeJSONMap[map[string]string](record.Get("endpoints")),
        Active:       record.GetBool("active"),
    }
}

func decodeJSONMap[T any](value any) T {
    var out T
    if value == nil {
        return out
    }
    bytes, err := json.Marshal(value)
    if err != nil {
        return out
    }
    _ = json.Unmarshal(bytes, &out)
    return out
}
```

## 3. Complete Route Registration

```go
func registerRoutes(pb *pocketbase.PocketBase, opts Options) {
    pb.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
        Func: func(e *core.ServeEvent) error {
            // Health check
            e.Router.GET("/api/myapp/health", func(ev *core.RequestEvent) error {
                return ev.JSON(http.StatusOK, map[string]any{
                    "ok": true, "service": "myapp",
                })
            })

            // Provider CRUD
            e.Router.GET("/api/myapp/providers", func(ev *core.RequestEvent) error {
                s := store.New(pb)
                providers, err := s.ListProviders()
                if err != nil {
                    return ev.JSON(http.StatusInternalServerError,
                        map[string]any{"ok": false, "message": err.Error()})
                }
                // mask secrets
                for i := range providers {
                    providers[i].APIKey = maskAPIKey(providers[i].APIKey)
                }
                return ev.JSON(http.StatusOK, providers)
            })

            e.Router.GET("/api/myapp/providers/{slug}", func(ev *core.RequestEvent) error {
                s := store.New(pb)
                p, err := s.GetProvider(ev.Request.PathValue("slug"))
                if err != nil || p == nil {
                    return ev.JSON(http.StatusNotFound,
                        map[string]any{"ok": false, "message": "provider not found"})
                }
                p.APIKey = maskAPIKey(p.APIKey)
                return ev.JSON(http.StatusOK, p)
            })

            e.Router.POST("/api/myapp/providers", func(ev *core.RequestEvent) error {
                var input store.Provider
                if err := json.NewDecoder(ev.Request.Body).Decode(&input); err != nil {
                    return ev.JSON(http.StatusBadRequest,
                        map[string]any{"ok": false, "message": "invalid JSON: " + err.Error()})
                }
                s := store.New(pb)
                provider, err := s.UpsertProvider(input)
                if err != nil {
                    return ev.JSON(http.StatusBadRequest,
                        map[string]any{"ok": false, "message": err.Error()})
                }
                provider.APIKey = maskAPIKey(provider.APIKey)
                return ev.JSON(http.StatusCreated, provider)
            })

            e.Router.PUT("/api/myapp/providers/{slug}", func(ev *core.RequestEvent) error {
                var input store.Provider
                json.NewDecoder(ev.Request.Body).Decode(&input)
                input.Slug = ev.Request.PathValue("slug")
                s := store.New(pb)
                // preserve existing secret if empty in update
                if input.APIKey == "" {
                    existing, _ := s.GetProvider(input.Slug)
                    if existing != nil {
                        input.APIKey = existing.APIKey
                    }
                }
                provider, err := s.UpsertProvider(input)
                if err != nil {
                    return ev.JSON(http.StatusBadRequest,
                        map[string]any{"ok": false, "message": err.Error()})
                }
                provider.APIKey = maskAPIKey(provider.APIKey)
                return ev.JSON(http.StatusOK, provider)
            })

            e.Router.DELETE("/api/myapp/providers/{slug}", func(ev *core.RequestEvent) error {
                s := store.New(pb)
                if err := s.DeleteProvider(ev.Request.PathValue("slug")); err != nil {
                    return ev.JSON(http.StatusBadRequest,
                        map[string]any{"ok": false, "message": err.Error()})
                }
                return ev.JSON(http.StatusOK, map[string]any{"ok": true})
            })

            // Serve embedded web UI
            webHandler, err := webui.Handler()
            if err == nil {
                e.Router.GET("/{$}", func(ev *core.RequestEvent) error {
                    webHandler.(http.Handler).ServeHTTP(ev.Response, ev.Request)
                    return nil
                })
                e.Router.GET("/style.css", func(ev *core.RequestEvent) error {
                    webHandler.(http.Handler).ServeHTTP(ev.Response, ev.Request)
                    return nil
                })
                e.Router.GET("/app.js", func(ev *core.RequestEvent) error {
                    webHandler.(http.Handler).ServeHTTP(ev.Response, ev.Request)
                    return nil
                })
            }

            return e.Next()
        },
    })
}

func maskAPIKey(key string) string {
    if key == "" {
        return ""
    }
    if len(key) <= 8 {
        return strings.Repeat("*", len(key))
    }
    return key[:4] + strings.Repeat("*", len(key)-8) + key[len(key)-4:]
}
```

## 4. Full Cobra CLI with Serve Command

```go
import (
    "errors"
    "fmt"
    "net/http"
    "os"

    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/apis"
    "github.com/spf13/cobra"
)

func NewCLI() *cobra.Command {
    opts := Options{DataDir: "pb_data"}
    var pb *pocketbase.PocketBase

    getPB := func() (*pocketbase.PocketBase, error) {
        if pb != nil {
            return pb, nil
        }
        pb = pocketbase.NewWithConfig(pocketbase.Config{
            DefaultDataDir: opts.DataDir,
        })
        migratecmd.MustRegister(pb, pb.RootCmd,
            migratecmd.Config{Automigrate: true})
        if err := pb.Bootstrap(); err != nil {
            return nil, err
        }
        return pb, nil
    }

    cmd := &cobra.Command{
        Use:   "myapp",
        Short: "My Go CLI application",
        RunE: func(cmd *cobra.Command, args []string) error {
            pb, err := getPB()
            if err != nil {
                return err
            }
            // default: run TUI or print help
            return nil
        },
    }

    cmd.PersistentFlags().StringVar(&opts.DataDir, "dir",
        opts.DataDir, "PocketBase data directory")

    cmd.AddCommand(serveCommand(getPB))
    return cmd
}

func serveCommand(getPB func() (*pocketbase.PocketBase, error)) *cobra.Command {
    var httpAddr string
    cmd := &cobra.Command{
        Use:   "serve",
        Short: "Start the REST API server",
        RunE: func(cmd *cobra.Command, args []string) error {
            if httpAddr == "" {
                httpAddr = "127.0.0.1:8090"
            }

            fmt.Fprintf(cmd.OutOrStdout(),
                "Starting server on %s ...\n", httpAddr)
            fmt.Fprintf(cmd.OutOrStdout(),
                "  Web UI:    http://%s/\n", httpAddr)
            fmt.Fprintf(cmd.OutOrStdout(),
                "  REST API:  http://%s/api/myapp/\n", httpAddr)

            pb, err := getPB()
            if err != nil {
                return err
            }

            err = apis.Serve(pb, apis.ServeConfig{
                HttpAddr:        httpAddr,
                ShowStartBanner: false,
                AllowedOrigins:  []string{"*"},
            })
            if errors.Is(err, http.ErrServerClosed) {
                return nil
            }
            return err
        },
    }
    cmd.PersistentFlags().StringVar(&httpAddr, "http",
        "", "HTTP listen address (default 127.0.0.1:8090)")
    return cmd
}
```

## 5. Embedded Web UI Pattern

```go
// internal/webui/webui.go
package webui

import (
    "embed"
    "io/fs"
    "net/http"
)

//go:embed static/*
var staticFiles embed.FS

func Handler() (http.Handler, error) {
    sub, err := fs.Sub(staticFiles, "static")
    if err != nil {
        return nil, err
    }
    return http.FileServer(http.FS(sub)), nil
}
```

The `static/` directory contains:
- `index.html` — Single-page app entry point
- `style.css` — CSS stylesheet
- `app.js` — Vanilla JavaScript calling the REST API

## 6. Configuration Export/Import (TOML/JSON)

### Models with Dual Tags

```go
type Provider struct {
    ID       string `json:"id" toml:"-"`
    Slug     string `json:"slug" toml:"slug"`
    Name     string `json:"name" toml:"name"`
    APIKey   string `json:"api_key,omitempty" toml:"api_key,omitempty"`
}
```

### Export

```go
func Export(s *Store, path string, format Format, includeSecrets bool) error {
    providers, _ := s.ListProviders()
    if !includeSecrets {
        for i := range providers {
            providers[i].APIKey = ""
        }
    }

    data := struct {
        Providers []Provider `toml:"providers" json:"providers"`
    }{Providers: providers}

    var buf bytes.Buffer
    switch format {
    case FormatJSON:
        enc := json.NewEncoder(&buf)
        enc.SetIndent("", "  ")
        enc.Encode(data)
    case FormatTOML:
        toml.NewEncoder(&buf).Encode(data)
    }

    return os.WriteFile(path, buf.Bytes(), 0644)
}
```

### Import

```go
func Import(s *Store, path string, format Format) error {
    data, _ := os.ReadFile(path)

    var input struct {
        Providers []Provider `toml:"providers" json:"providers"`
    }

    if format == "" {
        // auto-detect
        ext := filepath.Ext(path)
        switch ext {
        case ".json":
            format = FormatJSON
        default:
            format = FormatTOML
        }
    }

    switch format {
    case FormatJSON:
        json.Unmarshal(data, &input)
    case FormatTOML:
        toml.Unmarshal(data, &input)
    }

    return s.RunInTransaction(func(tx *Store) error {
        for _, p := range input.Providers {
            if _, err := tx.UpsertProvider(p); err != nil {
                return err
            }
        }
        return nil
    })
}
```

## 7. Key Dependencies (go.mod)

```
require (
    github.com/pocketbase/pocketbase v0.39.1
    github.com/pocketbase/dbx v1.12.0
    github.com/spf13/cobra v1.10.2
    github.com/BurntSushi/toml v1.6.0
)
```
