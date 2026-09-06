import { useState } from 'react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Spinner } from '@/components/ui/spinner'
import { useTestProvider } from '@/lib/queries'
import { useI18n } from '@/lib/i18n'

interface Props {
  slug: string
  defaultModel?: string
  models: string[]
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function TestDialog({ slug, defaultModel, models, open, onOpenChange }: Props) {
  const [model, setModel] = useState<string>(defaultModel ?? '')
  const testMutation = useTestProvider()
  const { t } = useI18n()

  const run = () => {
    testMutation.mutate(
      { slug, model: model || undefined },
      {
        onSuccess: (result) => {
          if (result.ok) toast.success(t('test.passed', { code: result.status_code }))
        },
      }
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t('test.title')}</DialogTitle>
          <DialogDescription>{t('test.desc')}</DialogDescription>
        </DialogHeader>

        <div className="flex items-center gap-2">
          <Select value={model} onValueChange={setModel}>
            <SelectTrigger className="w-full">
              <SelectValue placeholder={defaultModel ?? ''} />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {models.map((m) => (
                  <SelectItem key={m} value={m}>
                    {m}
                    {m === defaultModel ? t('test.default') : ''}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
          <Button onClick={run} disabled={testMutation.isPending}>
            {testMutation.isPending ? <Spinner data-icon="inline-start" /> : null}
            {testMutation.isPending ? t('test.running') : t('test.run')}
          </Button>
        </div>

        {testMutation.data && (
          <pre
            className={`max-h-56 overflow-auto rounded-lg border p-3 font-mono text-xs ${
              testMutation.data.ok ? '' : 'text-destructive'
            }`}
          >
            {JSON.stringify(testMutation.data, null, 2)}
          </pre>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('test.close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
