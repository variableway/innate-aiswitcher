package app

import (
	"log"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

func logServeStartup(dataDir, httpAddr string) {
	log.Printf("aisw: data dir %s", dataDir)
	log.Printf("aisw: bootstrapping database...")
}

func logServeReady(httpAddr string) {
	log.Printf("aisw: server listening on http://%s", httpAddr)
	log.Printf("aisw: web UI    http://%s/", httpAddr)
	log.Printf("aisw: REST API  http://%s/api/aisw/", httpAddr)
}

func registerAccessLog(e *core.ServeEvent, enabled bool) {
	if !enabled {
		return
	}
	e.Router.BindFunc(func(ev *core.RequestEvent) error {
		start := time.Now()
		err := ev.Next()
		status := ev.Status()
		if status == 0 {
			status = http.StatusOK
		}
		log.Printf(
			"aisw: %s %s %d %s",
			ev.Request.Method,
			ev.Request.URL.RequestURI(),
			status,
			time.Since(start).Round(time.Millisecond),
		)
		return err
	})
}
