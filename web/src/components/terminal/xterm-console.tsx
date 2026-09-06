import { useEffect, useRef } from 'react'
import { useI18n } from '@/lib/i18n'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

export interface TerminalSession {
  id: string
  title: string
}

interface Props {
  session: TerminalSession
  active: boolean
  initialCommand?: string
  onExit?: (id: string) => void
}

/**
 * A single browser terminal backed by a server-side PTY over WebSocket.
 *
 * Wire protocol (see internal/terminal):
 * - client → server text frames: raw keystrokes, except JSON control frames
 *   like {"type":"resize","cols":N,"rows":N}
 * - client → server binary frames: raw stdin bytes
 * - server → client binary frames: raw PTY output
 */
export function XtermConsole({ session, active, initialCommand, onExit }: Props) {
  const { t } = useI18n()
  const closedLabel = t('terminal.closed')
  const holderRef = useRef<HTMLDivElement>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const termRef = useRef<Terminal | null>(null)

  useEffect(() => {
    const holder = holderRef.current
    if (!holder) return

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
      scrollback: 5000,
      theme: {
        background: 'transparent',
      },
      allowProposedApi: true,
    })
    const fit = new FitAddon()
    term.loadAddon(fit)
    term.open(holder)
    termRef.current = term

    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(`${proto}://${location.host}/api/aisw/terminal`)
    ws.binaryType = 'arraybuffer'
    wsRef.current = ws

    ws.onopen = () => {
      const sendResize = () => {
        if (ws.readyState !== WebSocket.OPEN) return
        ws.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }))
      }
      sendResize()
      term.onData((data) => {
        if (ws.readyState === WebSocket.OPEN) ws.send(data)
      })
      term.onResize(sendResize)
      if (initialCommand) {
        ws.send(initialCommand.endsWith('\n') ? initialCommand : initialCommand + '\n')
      }
      term.focus()
    }
    ws.onmessage = (event) => {
      if (typeof event.data === 'string') {
        // JSON control frames from the server (e.g. exit notices)
        try {
          const msg = JSON.parse(event.data) as { type?: string }
          if (msg.type === 'exit') onExit?.(session.id)
        } catch {
          term.write(event.data)
        }
        return
      }
      term.write(new Uint8Array(event.data))
    }
    ws.onclose = () => {
      term.write(`\r\n\x1b[2m${closedLabel}\x1b[0m\r\n`)
    }

    const fitTimer = window.setInterval(() => {
      if (holder.clientWidth > 0) {
        try {
          fit.fit()
        } catch {
          // element may be mid-unmount
        }
      }
    }, 500)
    // initial fit once visible
    requestAnimationFrame(() => {
      try {
        fit.fit()
      } catch {
        /* noop */
      }
    })

    return () => {
      window.clearInterval(fitTimer)
      ws.close()
      term.dispose()
      wsRef.current = null
      termRef.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session.id, closedLabel])

  useEffect(() => {
    // Focus the active terminal so keystrokes land in the right session.
    if (active) {
      requestAnimationFrame(() => {
        try {
          termRef.current?.focus()
        } catch {
          /* noop */
        }
      })
    }
  }, [active])

  return (
    <div
      ref={holderRef}
      className="h-full w-full px-3 py-2"
      style={{ display: active ? 'block' : 'none' }}
    />
  )
}
