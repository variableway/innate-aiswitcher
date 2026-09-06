import { useState } from 'react'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import { Skeleton } from '@/components/ui/skeleton'
import { ProfileFormDialog } from '@/components/profiles/profile-form'
import { useDeleteProfile, useProfiles } from '@/lib/queries'
import { useI18n } from '@/lib/i18n'
import type { Profile } from '@/lib/types'

export function ProfilesPage() {
  const { data: profiles, isPending, isError, error } = useProfiles()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<Profile | null>(null)
  const { t } = useI18n()

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <header className="border-b flex items-center justify-between px-6 py-4">
        <div className="flex flex-col gap-0.5">
          <h1 className="text-lg font-semibold">{t('profiles.title')}</h1>
          <p className="text-muted-foreground text-sm">{t('profiles.subtitle')}</p>
        </div>
        <Button
          onClick={() => {
            setEditing(null)
            setFormOpen(true)
          }}
        >
          <Plus data-icon="inline-start" />
          {t('profiles.add')}
        </Button>
      </header>

      <div className="flex-1 overflow-auto p-6">
        {isPending ? (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 2 }).map((_, i) => (
              <Skeleton key={i} className="h-40 w-full" />
            ))}
          </div>
        ) : isError ? (
          <Empty className="m-auto max-w-md">
            <EmptyHeader>
              <EmptyTitle>{t('profiles.loadFailed')}</EmptyTitle>
              <EmptyDescription>{error.message}</EmptyDescription>
            </EmptyHeader>
          </Empty>
        ) : profiles.length === 0 ? (
          <Empty className="m-auto max-w-md">
            <EmptyHeader>
              <EmptyTitle>{t('profiles.empty.title')}</EmptyTitle>
              <EmptyDescription>{t('profiles.empty.desc')}</EmptyDescription>
            </EmptyHeader>
            <EmptyContent>
              <Button
                onClick={() => {
                  setEditing(null)
                  setFormOpen(true)
                }}
              >
                <Plus data-icon="inline-start" />
                {t('profiles.add')}
              </Button>
            </EmptyContent>
          </Empty>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
            {profiles.map((profile) => (
              <ProfileCard
                key={profile.slug}
                profile={profile}
                onEdit={() => {
                  setEditing(profile)
                  setFormOpen(true)
                }}
              />
            ))}
          </div>
        )}
      </div>

      <ProfileFormDialog
        open={formOpen}
        onOpenChange={(open) => {
          setFormOpen(open)
          if (!open) setEditing(null)
        }}
        profile={editing}
      />
    </div>
  )
}

function ProfileCard({ profile, onEdit }: { profile: Profile; onEdit: () => void }) {
  const deleteMutation = useDeleteProfile()
  const { t } = useI18n()
  return (
    <Card className="flex h-full flex-col">
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div className="flex min-w-0 flex-col gap-1">
            <CardTitle className="truncate">{profile.name}</CardTitle>
            <CardDescription className="font-mono">{profile.slug}</CardDescription>
          </div>
          {profile.is_default && <Badge>{t('profiles.default')}</Badge>}
        </div>
      </CardHeader>
      <CardContent className="flex flex-1 flex-col gap-3">
        <div className="text-muted-foreground flex flex-col gap-1 text-sm">
          <span>
            {t('profiles.agent')}: <span className="text-foreground font-mono">{profile.agent}</span>{' '}
            → {t('profiles.provider')}:{' '}
            <span className="text-foreground font-mono">{profile.provider}</span>
          </span>
          {profile.model && (
            <span>
              {t('profiles.model')}: <span className="text-foreground font-mono">{profile.model}</span>
            </span>
          )}
          {profile.default_args && <span className="font-mono text-xs">{profile.default_args}</span>}
        </div>
        <div className="mt-auto flex justify-end gap-2 pt-2">
          <Button variant="outline" size="sm" onClick={onEdit}>
            <Pencil data-icon="inline-start" />
            {t('profiles.edit')}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            disabled={deleteMutation.isPending}
            onClick={() => {
              if (window.confirm(t('profiles.deleteConfirm', { name: profile.name }))) {
                deleteMutation.mutate(profile.slug)
              }
            }}
          >
            <Trash2 data-icon="inline-start" />
            {t('profiles.delete')}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
