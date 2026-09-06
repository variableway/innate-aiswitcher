// Package terminal bridges browser WebSocket sessions to local PTY shells,
// letting the Web UI run interactive terminals (and through them, agent
// sessions started via `aisw start`).
//
// Wire protocol:
//   - client → server text frames: raw keystrokes, except JSON control frames
//     of the form {"type":"resize","cols":N,"rows":N}
//   - client → server binary frames: raw stdin bytes
//   - server → client binary frames: raw PTY output
//   - server → client text frame {"type":"exit"}: the shell exited
package terminal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

// Options configures the terminal handler.
type Options struct {
	// Command is the program to run inside the PTY. Empty means the user's
	// login shell ($SHELL, falling back to /bin/bash or /bin/sh).
	Command []string
	// InitialEnv is appended to os.Environ() for the PTY process.
	InitialEnv []string
}

// Handler returns an http.HandlerFunc that upgrades the request to a
// WebSocket and attaches it to a fresh PTY session.
func Handler(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ServeWS(w, r, opts)
	}
}

var upgrader = websocket.Upgrader{
	// The switcher is a local tool served from the same origin; PocketBase's
	// CORS config already governs API exposure.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWS upgrades an http request and pipes it to a PTY until either side
// closes.
func ServeWS(w http.ResponseWriter, r *http.Request, opts Options) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	command := opts.Command
	if len(command) == 0 {
		command = []string{loginShell(), "-l"}
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	cmd.Env = append(cmd.Env, opts.InitialEnv...)
	cmd.Env = append(cmd.Env, "COLUMNS=80", "LINES=24")

	ptty, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		writeJSON(conn, map[string]string{"type": "error", "message": err.Error()})
		return
	}
	defer ptty.Close()

	// PTY output → WebSocket (binary frames). When the shell exits, notify
	// the client with an exit control frame and close the connection, which
	// also unblocks the read loop below.
	done := make(chan struct{})
	var once sync.Once
	go func() {
		defer once.Do(func() { close(done) })
		buf := make([]byte, 32*1024)
		for {
			n, err := ptty.Read(buf)
			if n > 0 {
				if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		writeJSON(conn, map[string]string{"type": "exit"})
		conn.Close()
	}()

	// WebSocket → PTY stdin; text frames double as the control channel.
	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}
		if messageType == websocket.TextMessage {
			if control, ok := parseControl(payload); ok {
				if control.Type == "resize" {
					_ = pty.Setsize(ptty, &pty.Winsize{Rows: control.Rows, Cols: control.Cols})
				}
				continue
			}
		}
		if _, err := ptty.Write(payload); err != nil {
			break
		}
	}

	// Client went away: terminate the shell so the PTY goroutine stops.
	if cmd.Process != nil {
		_ = cmd.Process.Signal(os.Interrupt)
	}
	<-done
	_ = cmd.Wait()
}

type controlFrame struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// parseControl reports whether a text frame is a JSON control message.
// Keystrokes are plain text and almost never parse as an object with a
// "type" field, so this heuristic is safe in practice.
func parseControl(payload []byte) (controlFrame, bool) {
	var frame controlFrame
	if len(payload) == 0 || payload[0] != '{' {
		return frame, false
	}
	if err := json.Unmarshal(payload, &frame); err != nil {
		return frame, false
	}
	return frame, frame.Type != ""
}

func writeJSON(conn *websocket.Conn, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		return
	}
	_ = conn.WriteMessage(websocket.TextMessage, payload)
}

func loginShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	if runtime.GOOS == "windows" {
		return "cmd.exe"
	}
	for _, candidate := range []string{"/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "/bin/sh"
}

// LoopbackWarning returns a warning to log when the terminal endpoint would
// be exposed beyond localhost.
func LoopbackWarning(addr string) string {
	return fmt.Sprintf(
		"WARNING: terminal sessions are enabled and the server listens on %s — "+
			"anyone who can reach this address gets a local shell. Keep the binding on 127.0.0.1 unless you trust the network.",
		addr)
}
