package terminal_test

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/variableway/innate-aiswitcher/internal/terminal"
)

func TestTerminalSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Terminal Suite")
}

var _ = Describe("browser terminal sessions over a local PTY", func() {
	var server *httptest.Server
	var conn *websocket.Conn

	BeforeEach(func() {
		server = httptest.NewServer(terminal.Handler(terminal.Options{
			Command: []string{"/bin/sh", "-i"},
		}))
		DeferCleanup(func() { server.Close() })

		var err error
		conn, _, err = websocket.DefaultDialer.Dial("ws"+server.URL[len("http"):], nil)
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { conn.Close() })
	})

	readUntil := func(marker string, timeout time.Duration) string {
		deadline := time.Now().Add(timeout)
		accumulated := ""
		for time.Now().Before(deadline) {
			_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			_, payload, err := conn.ReadMessage()
			if err == nil {
				accumulated += string(payload)
				if strings.Contains(accumulated, marker) {
					return accumulated
				}
			}
		}
		Fail(fmt.Sprintf("timed out waiting for %q in terminal output:\n%s", marker, accumulated))
		return accumulated
	}

	It("echoes a shell command's output back to the browser", func() {
		Expect(conn.WriteMessage(websocket.TextMessage, []byte("echo aisw-terminal-ok\n"))).To(Succeed())
		Expect(readUntil("aisw-terminal-ok", 10*time.Second)).NotTo(BeEmpty())
	})

	It("accepts resize control frames without disturbing the session", func() {
		Expect(conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"resize","cols":120,"rows":40}`))).To(Succeed())
		Expect(conn.WriteMessage(websocket.TextMessage, []byte("echo aisw-after-resize\n"))).To(Succeed())
		readUntil("aisw-after-resize", 10*time.Second)
	})

	It("forwards binary frames as raw stdin", func() {
		Expect(conn.WriteMessage(websocket.BinaryMessage, []byte("echo aisw-binary-stdin\n"))).To(Succeed())
		readUntil("aisw-binary-stdin", 10*time.Second)
	})

	It("sends an exit control frame when the shell quits", func() {
		Expect(conn.WriteMessage(websocket.TextMessage, []byte("exit\n"))).To(Succeed())
		deadline := time.Now().Add(10 * time.Second)
		failures := 0
		for time.Now().Before(deadline) {
			_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			messageType, payload, err := conn.ReadMessage()
			if err != nil {
				// Server closed the socket before the exit frame arrived.
				failures++
				if failures > 20 {
					break
				}
				time.Sleep(50 * time.Millisecond)
				continue
			}
			if messageType == websocket.TextMessage && strings.Contains(string(payload), `"type":"exit"`) {
				return
			}
		}
		Fail("expected an exit control frame before the socket closed")
	})
})
