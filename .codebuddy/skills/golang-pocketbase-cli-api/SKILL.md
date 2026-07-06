---
name: golang-pocketbase-cli-api
description: >
  Creates Go applications combining PocketBase (SQLite-backed DB + auto CRUD),
  cobra CLI (subcommands, flags, args), and REST API endpoints. Use when the user
  asks to build a Go CLI tool with local database storage, a REST server with custom
  routes, or a PocketBase-backed application with web admin UI. Covers repository/store
  patterns, embedded static files (Go embed), migration writing, and TOML/JSON
  config import/export.
---

# Go + PocketBase + CLI + REST API Skill

Build Go applications that integrate **PocketBase** (embedded SQLite database with
auto-migration), **cobra** (rich CLI with subcommands and persistent flags), and a
custom **REST API** layer served alongside the PocketBase HTTP server.

## When to Use

- Building a Go CLI tool that needs a local SQLite database with schema migrations.
- Creating a local server exposing CRUD REST endpoints backed by PocketBase collections.
- Adding a web-based configuration UI served from embedded static files (`embed.FS`).
- Needing auto-run app migrations, presets/templates, and TOML/JSON config import/export.
- Any project where PocketBase is the embedded data layer and cobra is the CLI framework.

## Architecture Overview

The canonical project structure (shown in `assets/project-template/`):

```
project/
├── main.go                  # Entry point: app.NewCLI().Execute()
├── cmd/
│   └── project/
│       └── main.go          # Package main; imports internal/app
├── internal/
│   ├── app/
│   │   └── app.go           # PocketBase init, cobra CLI tree, route registration
│   ├── store/
│   │   ├── models.go        # Typed Go structs with JSON+TOML tags
│   │   └── store.go         # CRUD wrapper around core.App + core.Record
│   ├── webui/
│   │   ├── webui.go         # embed static/ → http.FileServer
│   │   └── static/          # HTML/CSS/JS for web config UI
│   └── templates/
│       ├── templates.go     # Parse/generate presets from TOML constants
│       └── files/
│           └── presets.toml # Provider/entity presets used by CLI and API
├── migrations/
│   └── *.go                 # Register with migrations.Register(func, func)
├── go.mod
└── Taskfile.yml
```

## PocketBase Integration

### Initializing PocketBase

```go
import (
    "github.com/pocketbase/pocketbase"
    "github.com/pocketbase/pocketbase/plugins/migratecmd"
    "github.com/pocketbase/pocketbase/tools/hook"
    "github.com/pocketbase/pocketbase/core"
)

pb := pocketbase.NewWithConfig(pocketbase.Config{
    DefaultDataDir:  "pb_data",   // SQLite file location
    HideStartBanner: true,        // suppress PocketBase startup banner
})

// Register auto-migration for all migrations/ files
migratecmd.MustRegister(pb, pb.RootCmd, migratecmd.Config{Automigrate: true})

// App migrations run after PocketBase collections are ready
pb.OnBootstrap().Bind(&hook.Handler[*core.BootstrapEvent]{
    Func: func(e *core.BootstrapEvent) error {
        if err := e.Next(); err != nil {
            return err
        }
        if err := e.App.RunAppMigrations(); err != nil {
            return err
        }
        return e.App.ReloadCachedCollections()
    },
})
```

### Registering Custom Routes

Routes are added through `pb.OnServe()` hook, which fires before the HTTP server starts:

```go
pb.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
    Func: func(e *core.ServeEvent) error {
        // e.Router is a standard http.ServeMux-compatible router with path params
        e.Router.GET("/api/my/health", func(ev *core.RequestEvent) error {
            return ev.JSON(http.StatusOK, map[string]any{"ok": true})
        })

        e.Router.GET("/api/my/items/{slug}", func(ev *core.RequestEvent) error {
            slug := ev.Request.PathValue("slug")
            // ... fetch item ...
            return ev.JSON(http.StatusOK, item)
        })

        e.Router.POST("/api/my/items", func(ev *core.RequestEvent) error {
            var input MyModel
            json.NewDecoder(ev.Request.Body).Decode(&input)
            // ... save ...
            return ev.JSON(http.StatusCreated, result)
        })

        return e.Next()
    },
})
```

**Key point**: `e.JSON(status, data)` is the standard response method. Use `*core.RequestEvent` for route handlers, not `http.ResponseWriter/Request` directly.

### Serving Embedded Static Files

```go
// webui.go
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

Then register in the serve hook:

```go
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
```

### Starting the Server

```go
import (
    "errors"
    "net/http"
    "github.com/pocketbase/pocketbase/apis"
)

err = apis.Serve(pb, apis.ServeConfig{
    HttpAddr:           "127.0.0.1:8090",
    HttpsAddr:          "",              // empty = no TLS
    ShowStartBanner:    true,
    AllowedOrigins:     []string{"*"},   // CORS
    CertificateDomains: []string{},      // Let's Encrypt domains
})
if errors.Is(err, http.ErrServerClosed) {
    return nil  // graceful shutdown
}
```

## CLI Structure with Cobra

### Root Command

```go
func NewCLI() *cobra.Command {
    var dataDir string
    cmd := &cobra.Command{
        Use:          "myapp",
        Short:        "A Go CLI with PocketBase",
        SilenceUsage: true,
        RunE: func(cmd *cobra.Command, args []string) error {
            // Default action when no subcommand given
            return nil
        },
    }
    cmd.PersistentFlags().StringVar(&dataDir, "dir", "pb_data", "data directory")
    cmd.AddCommand(
        itemCommand(),
        serveCommand(),
    )
    return cmd
}
```

### Subcommands with Required Flags

```go
func itemCommand() *cobra.Command {
    var item MyModel
    var myFlag string

    addCmd := &cobra.Command{
        Use:   "add SLUG",
        Short: "Add an item",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            item.Slug = args[0]
            // ... validate and save ...
            fmt.Fprintf(cmd.OutOrStdout(), "saved %s\n", item.Slug)
            return nil
        },
    }
    addCmd.Flags().StringVar(&item.Name, "name", "", "display name")
    addCmd.Flags().StringVar(&myFlag, "extra", "", "extra config")
    _ = addCmd.MarkFlagRequired("name")

    listCmd := &cobra.Command{
        Use:   "list",
        Short: "List items",
        Args:  cobra.NoArgs,
        RunE: func(cmd *cobra.Command, args []string) error {
            items, _ := store.ListItems()
            for _, item := range items {
                fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", item.Slug, item.Name)
            }
            return nil
        },
    }

    cmd := &cobra.Command{Use: "item", Short: "Manage items"}
    cmd.AddCommand(addCmd, listCmd)
    return cmd
}
```

### Lazy PB Initialization with Func Getter

Use a function variable to lazily create PocketBase once:

```go
type PBGetter func() (*pocketbase.PocketBase, error)

// In CLI setup:
var pb *pocketbase.PocketBase
getPB := func() (*pocketbase.PocketBase, error) {
    if pb != nil {
        return pb, nil
    }
    pb = pocketbase.NewWithConfig(...)
    if err := pb.Bootstrap(); err != nil {
        return nil, err
    }
    return pb, nil
}
// Pass getPB to every subcommand
cmd.AddCommand(itemCommand(getPB), serveCommand(getPB))
```

## Store / Repository Pattern

### Models

Define typed structs with JSON and TOML tags for dual serialization:

```go
type Item struct {
    ID      string `json:"id" toml:"-"`
    Slug    string `json:"slug" toml:"slug"`
    Name    string `json:"name" toml:"name"`
    Active  bool   `json:"active" toml:"active"`
}
```

- `toml:"-"` excludes the field from TOML export.
- `json:",omitempty"` omits empty values from JSON output.
- `Hidden` fields (like `api_key`) should have `json:"...,omitempty"`.

### Store CRUD

```go
type Store struct {
    app core.App
}

func New(app core.App) *Store {
    return &Store{app: app}
}

func (s *Store) UpsertItem(input Item) (*Item, error) {
    input.Slug = strings.TrimSpace(strings.ToLower(input.Slug))
    if input.Slug == "" {
        return nil, fmt.Errorf("slug is required")
    }

    // Find existing or create new record
    record, _ := s.app.FindFirstRecordByFilter("items", "slug={:slug}", dbx.Params{"slug": input.Slug})
    if record == nil {
        collection, _ := s.app.FindCollectionByNameOrId("items")
        record = core.NewRecord(collection)
    }

    record.Set("slug", input.Slug)
    record.Set("name", input.Name)
    record.Set("active", input.Active)

    if err := s.app.Save(record); err != nil {
        return nil, err
    }
    return recordToItem(record), nil
}

func (s *Store) ListItems() ([]Item, error) {
    records, _ := s.app.FindRecordsByFilter("items", "", "slug", 0, 0)
    items := make([]Item, 0, len(records))
    for _, r := range records {
        items = append(items, *recordToItem(r))
    }
    return items, nil
}

func (s *Store) DeleteItem(slug string) error {
    record, _ := s.app.FindFirstRecordByFilter("items", "slug={:slug}", dbx.Params{"slug": slug})
    if record == nil {
        return fmt.Errorf("not found: %s", slug)
    }
    return s.app.Delete(record)
}

// record → struct conversion
func recordToItem(record *core.Record) *Item {
    return &Item{
        ID:     record.Id,
        Slug:   record.GetString("slug"),
        Name:   record.GetString("name"),
        Active: record.GetBool("active"),
    }
}
```

**Key patterns**:
- `FindFirstRecordByFilter` with `dbx.Params` for single lookup.
- `FindRecordsByFilter` with sort field, offset, limit for listing.
- `core.NewRecord(collection)` for inserts; existing record for updates.
- JSON fields use `record.Get("field")` and need `json.Marshal/Unmarshal` to decode into Go maps.

## Database Migrations

Each migration is a file in the `migrations/` package registered via `init()`:

```go
package migrations

import (
    "github.com/pocketbase/pocketbase/core"
    "github.com/pocketbase/pocketbase/migrations"
)

func init() {
    migrations.Register(
        func(txApp core.App) error {
            // UP: create collection
            items := core.NewBaseCollection("items")
            items.Fields.Add(
                &core.TextField{Name: "slug", Required: true, Presentable: true},
                &core.TextField{Name: "name", Required: true},
                &core.BoolField{Name: "active"},
            )
            items.AddIndex("idx_items_slug", true, "slug", "")
            return txApp.Save(items)
        },
        func(txApp core.App) error {
            // DOWN: delete collection
            collection, _ := txApp.FindCollectionByNameOrId("items")
            return txApp.Delete(collection)
        },
    )
}
```

**Field types**: `core.TextField`, `core.URLField`, `core.BoolField`, `core.NumberField`, `core.SelectField` (with `Values`), `core.JSONField`, `core.RelationField` (with `CollectionId`, `CascadeDelete`, `MaxSelect`), `core.EmailField`, `core.FileField`.

**Relation fields** link collections:
```go
&core.RelationField{
    Name:           "provider",
    Required:       true,
    CollectionId:   providers.Id,
    CascadeDelete:  true,
    MaxSelect:      1,
}
```

**Seed data** is added in the same UP function by creating `core.NewRecord(collection)` and calling `txApp.Save(record)`.

## REST API Design Patterns

### CRUD Route Conventions

| Method   | Path                          | Purpose                |
|----------|-------------------------------|------------------------|
| GET      | `/api/ns/items`               | List all               |
| GET      | `/api/ns/items/{slug}`        | Get one by slug        |
| POST     | `/api/ns/items`               | Create                 |
| PUT      | `/api/ns/items/{slug}`        | Update                 |
| DELETE   | `/api/ns/items/{slug}`        | Delete                 |
| POST     | `/api/ns/items/{slug}/action` | Sub-resource action    |

### Error Response Pattern

Always return `{"ok": false, "message": "..."}` on errors:

```go
return ev.JSON(http.StatusNotFound, map[string]any{
    "ok": false, "message": "item not found",
})
```

On success, return the entity directly (list returns `[]Item`, single returns `Item`):

```go
return ev.JSON(http.StatusOK, item)        // single object
return ev.JSON(http.StatusOK, items)       // array
return ev.JSON(http.StatusCreated, result) // after create
return ev.JSON(http.StatusOK, map[string]any{"ok": true}) // after delete
```

### Sensitive Field Handling

When returning data, mask secrets and exclude internal IDs:

```go
for i := range items {
    items[i].APIKey = maskAPIKey(items[i].APIKey)
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

### Update with Partial Secrets

When updating, if the API key is empty in the request, preserve the existing one:

```go
e.Router.PUT("/api/ns/items/{slug}", func(ev *core.RequestEvent) error {
    var input Item
    json.NewDecoder(ev.Request.Body).Decode(&input)
    input.Slug = ev.Request.PathValue("slug")

    s := store.New(pb)
    if input.SecretField == "" {
        existing, _ := s.GetItem(input.Slug)
        if existing != nil {
            input.SecretField = existing.SecretField
        }
    }
    result, err := s.UpsertItem(input)
    // ...
})
```

## Presets and Templates Pattern

### Embedding TOML Presets

Store provider presets in embedded TOML, parse them into Go structs:

```go
//go:embed files/presets.toml
var presetsTOML embed.FS

type Preset struct {
    Slug    string      `toml:"slug"`
    Name    string      `toml:"name"`
    Options []URLOption `toml:"url_options"`
}

func ProviderPresets() ([]Preset, error) {
    data, _ := fs.ReadFile(presetsTOML, "files/presets.toml")
    var presets []Preset
    _ = toml.Unmarshal(data, &presets)
    return presets, nil
}

// Convert preset + user API key into a full Provider
func ProviderFromPreset(preset Preset, option URLOption, apiKey string) Provider {
    return Provider{
        Slug:        preset.Slug + "-" + option.Slug,
        Name:        fmt.Sprintf("%s (%s)", preset.Name, option.Label),
        BaseURL:     option.BaseURL,
        APIProtocol: option.Protocol,
        DefaultModel: option.DefaultModel,
        APIKey:      apiKey,
        Active:      true,
    }
}
```

## CLI to Web UI Integration

To integrate both CLI and Web UI operating on the same database:

1. **serve command** starts PocketBase with custom routes (`OnServe` hook) and optional embedded web UI.
2. **CLI subcommands** use `pb.Bootstrap()` (without `Start()`) and call store methods directly.
3. Both share the same `store.Store` abstraction, so data written by CLI is instantly visible via API and vice versa.

## Common Patterns

### Disable Admin UI by Default

```go
ui.DistDirFS = nil  // hides PocketBase's built-in admin UI
```

### CORS Configuration

Allow all origins in serve command with `--origins` flag:
```go
e.Router.OPTIONS("/*", func(ev *core.RequestEvent) error {
    return ev.NoContent(http.StatusNoContent)
})
```

Or set via `apis.ServeConfig.AllowedOrigins`.

### Startup Logging

Print explicit startup messages for user clarity:
```go
fmt.Fprintf(cmd.OutOrStdout(), "Starting server on %s ...\n", addr)
fmt.Fprintf(cmd.OutOrStdout(), "  Web UI:    http://%s/\n", addr)
fmt.Fprintf(cmd.OutOrStdout(), "  REST API:  http://%s/api/\n", addr)
```

## Resources

### references/patterns.md
Detailed code examples extracted from the aiswitcher reference project, showing complete endpoint implementations, migration files, store methods, and the webui embed pattern.

### assets/project-template/
A minimal Go project scaffold with:
- `main.go` entry point
- `internal/app/app.go` with PocketBase init + cobra CLI
- `internal/store/` with models + CRUD
- `migrations/` sample migration
- `internal/webui/` embedded static file serving
- `go.mod` with required dependencies
- `Taskfile.yml` for build/serve tasks

### scripts/init-project.ps1
PowerShell script to initialize a new Go project from the template.
