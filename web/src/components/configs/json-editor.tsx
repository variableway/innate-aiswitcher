import { useMemo, useState } from 'react'
import CodeMirror from '@uiw/react-codemirror'
import { json } from '@codemirror/lang-json'
import { EditorView } from '@codemirror/view'
import { useTheme } from 'next-themes'
import { Eye, EyeOff } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useI18n } from '@/lib/i18n'

/** Mask API-key-ish values: JSON "…key…"/"token"/"secret" fields, TOML key = "…", and sk-* strings. */
export function maskSecrets(text: string): string {
  return text
    .replace(
      /("[^"]*(?:api[_-]?key|apikey|key|token|secret|password)[^"]*"\s*:\s*")([^"]*)(")/gi,
      '$1••••••••$3'
    )
    .replace(
      /(^\s*[\w.-]*(?:key|token|secret|password)[\w.-]*\s*=\s*")([^"]*)(")/gim,
      '$1••••••••$3'
    )
    .replace(/\b(sk-[A-Za-z0-9_-]{4})[A-Za-z0-9_-]+/g, '$1••••••••')
}

export function hasSecrets(text: string): boolean {
  return maskSecrets(text) !== text
}

const sizing = EditorView.theme({
  '&': { minHeight: '9rem', maxHeight: '24rem', fontSize: '12px' },
  '.cm-scroller': {
    overflow: 'auto',
    fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace',
  },
  '&.cm-focused': { outline: 'none' },
})

interface JsonEditorProps {
  value: string
  onChange?: (value: string) => void
  lang?: 'json' | 'toml' | string
  placeholder?: string
}

/**
 * CodeMirror-based config editor: JSON syntax highlighting, monospace,
 * content-sized height (min/max bounds). Sensitive values are masked by
 * default; revealing them (eye toggle) also enables editing.
 */
export function JsonEditor({ value, onChange, lang, placeholder }: JsonEditorProps) {
  const { t } = useI18n()
  const { resolvedTheme } = useTheme()
  const [revealed, setRevealed] = useState(false)
  const masked = useMemo(() => maskSecrets(value), [value])
  const secrets = masked !== value

  const extensions = useMemo(() => {
    const ext = [sizing, EditorView.lineWrapping]
    if (lang === 'json') ext.push(json())
    return ext
  }, [lang])

  const shown = secrets && !revealed ? masked : value

  return (
    <div className="focus-within:border-ring focus-within:ring-ring/50 relative overflow-hidden rounded-lg border focus-within:ring-2">
      <CodeMirror
        value={shown}
        onChange={(v) => onChange?.(v)}
        readOnly={!onChange || (secrets && !revealed)}
        placeholder={placeholder}
        theme={resolvedTheme === 'dark' ? 'dark' : 'light'}
        basicSetup={{ lineNumbers: false, foldGutter: false, highlightActiveLine: false }}
        extensions={extensions}
      />
      {secrets && (
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          className="text-muted-foreground absolute top-1.5 right-1.5"
          aria-label={revealed ? t('configs.hideSecrets') : t('configs.showSecrets')}
          title={revealed ? t('configs.hideSecrets') : t('configs.showSecrets')}
          onClick={() => setRevealed((v) => !v)}
        >
          {revealed ? <EyeOff /> : <Eye />}
        </Button>
      )}
    </div>
  )
}
