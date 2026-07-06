package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"
	"github.com/spf13/cobra"
	"myapp/internal/store"
	"myapp/internal/webui"
)

type Options struct {
	DataDir   string
}

type PBGetter func() (*pocketbase.PocketBase, error)

// NewCLI creates the root cobra command with all subcommands.
func NewCLI() *cobra.Command {
	opts := Options{DataDir: "pb_data"}
	var pb *pocketbase.PocketBase

	getPB := func() (*pocketbase.PocketBase, error) {
		if pb != nil {
			return pb, nil
		}
		pb = NewWithOptions(opts)
		if err := pb.Bootstrap(); err != nil {
			return nil, err
		}
		return pb, nil
	}

	cmd := &cobra.Command{
		Use:          "myapp",
		Short:        "My Go application with PocketBase + CLI + REST API",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := getPB()
			if err != nil {
				return err
			}
			items, _ := store.New(pb).ListItems()
			for _, item := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", item.Slug, item.Name)
			}
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&opts.DataDir, "dir", opts.DataDir, "PocketBase data directory")
	cmd.AddCommand(
		itemCommand(getPB),
		serveCommand(getPB),
	)
	return cmd
}

// NewWithOptions creates a PocketBase app with routes and migrations.
func NewWithOptions(opts Options) *pocketbase.PocketBase {
	pb := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir:  opts.DataDir,
		HideStartBanner: true,
	})

	migratecmd.MustRegister(pb, pb.RootCmd, migratecmd.Config{Automigrate: true})

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

	registerRoutes(pb)
	return pb
}

func registerRoutes(pb *pocketbase.PocketBase) {
	pb.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			// Health check
			e.Router.GET("/api/myapp/health", func(ev *core.RequestEvent) error {
				return ev.JSON(http.StatusOK, map[string]any{"ok": true})
			})

			// Item CRUD
			e.Router.GET("/api/myapp/items", func(ev *core.RequestEvent) error {
				s := store.New(pb)
				items, err := s.ListItems()
				if err != nil {
					return ev.JSON(http.StatusInternalServerError, map[string]any{"ok": false, "message": err.Error()})
				}
				return ev.JSON(http.StatusOK, items)
			})

			e.Router.GET("/api/myapp/items/{slug}", func(ev *core.RequestEvent) error {
				s := store.New(pb)
				item, err := s.GetItem(ev.Request.PathValue("slug"))
				if err != nil || item == nil {
					return ev.JSON(http.StatusNotFound, map[string]any{"ok": false, "message": "item not found"})
				}
				return ev.JSON(http.StatusOK, item)
			})

			e.Router.POST("/api/myapp/items", func(ev *core.RequestEvent) error {
				var input store.Item
				if err := json.NewDecoder(ev.Request.Body).Decode(&input); err != nil {
					return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": "invalid JSON"})
				}
				s := store.New(pb)
				item, err := s.UpsertItem(input)
				if err != nil {
					return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
				}
				return ev.JSON(http.StatusCreated, item)
			})

			e.Router.PUT("/api/myapp/items/{slug}", func(ev *core.RequestEvent) error {
				var input store.Item
				json.NewDecoder(ev.Request.Body).Decode(&input)
				input.Slug = ev.Request.PathValue("slug")
				s := store.New(pb)
				item, err := s.UpsertItem(input)
				if err != nil {
					return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
				}
				return ev.JSON(http.StatusOK, item)
			})

			e.Router.DELETE("/api/myapp/items/{slug}", func(ev *core.RequestEvent) error {
				s := store.New(pb)
				if err := s.DeleteItem(ev.Request.PathValue("slug")); err != nil {
					return ev.JSON(http.StatusBadRequest, map[string]any{"ok": false, "message": err.Error()})
				}
				return ev.JSON(http.StatusOK, map[string]any{"ok": true})
			})

			// Serve web UI
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

// --- CLI Subcommands ---

func itemCommand(getPB PBGetter) *cobra.Command {
	var item store.Item

	addCmd := &cobra.Command{
		Use:   "add SLUG",
		Short: "Add an item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			item.Slug = args[0]
			s := store.New(pb)
			result, err := s.UpsertItem(item)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "saved item %s (%s)\n", result.Slug, result.Name)
			return nil
		},
	}
	addCmd.Flags().StringVar(&item.Name, "name", "", "display name")
	addCmd.Flags().StringVar(&item.Description, "desc", "", "description")
	_ = addCmd.MarkFlagRequired("name")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List items",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			items, _ := store.New(pb).ListItems()
			for _, item := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", item.Slug, item.Name)
			}
			return nil
		},
	}

	deleteCmd := &cobra.Command{
		Use:   "delete SLUG",
		Short: "Delete an item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pb, err := getPB()
			if err != nil {
				return err
			}
			return store.New(pb).DeleteItem(args[0])
		},
	}

	cmd := &cobra.Command{Use: "item", Short: "Manage items"}
	cmd.AddCommand(addCmd, listCmd, deleteCmd)
	return cmd
}

func serveCommand(getPB PBGetter) *cobra.Command {
	var httpAddr string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the REST API + Web UI server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if httpAddr == "" {
				httpAddr = "127.0.0.1:8090"
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Starting server on %s ...\n", httpAddr)
			fmt.Fprintf(cmd.OutOrStdout(), "  Web UI:    http://%s/\n", httpAddr)
			fmt.Fprintf(cmd.OutOrStdout(), "  REST API:  http://%s/api/myapp/\n", httpAddr)

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
	cmd.PersistentFlags().StringVar(&httpAddr, "http", "", "HTTP listen address (default 127.0.0.1:8090)")
	return cmd
}
