import { useState } from 'react'
import { Pencil, Plus, Trash2, Zap } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { PageHeader } from '@/components/page-header'
import { QueryState } from '@/components/query-state'
import { ProfileFormDialog } from '@/components/profiles/profile-form'
import { useDeleteProfile, useProfiles } from '@/lib/queries'
import { useI18n } from '@/lib/i18n'
import { CARD_GRID } from '@/lib/layout'
import type { Profile } from '@/lib/types'

export function ProfilesPage() {
  const query = useProfiles()
  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<Profile | null>(null)
  const { t } = useI18n()

  const openAdd = () => {
    setEditing(null)
    setFormOpen(true)
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <PageHeader
        icon={Zap}
        title={t('profiles.title')}
        subtitle={t('profiles.subtitle')}
        actions={
          <Button onClick={openAdd}>
            <Plus data-icon="inline-start" />
            {t('profiles.add')}
          </Button>
        }
      />

      <div className="flex-1 overflow-auto p-6">
        <QueryState
          query={query}
          skeleton={
            <div className={CARD_GRID}>
              {Array.from({ length: 2 }).map((_, i) => (
                <Skeleton key={i} className="h-40 w-full" />
              ))}
            </div>
          }
          errorTitle={t('profiles.loadFailed')}
          emptyTitle={t('profiles.empty.title')}
          emptyDescription={t('profiles.empty.desc')}
          emptyContent={
            <Button onClick={openAdd}>
              <Plus data-icon="inline-start" />
              {t('profiles.add')}
            </Button>
          }
        >
          {(profiles) => (
            <div className={CARD_GRID}>
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
        </QueryState>
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
  const [deleteOpen, setDeleteOpen] = useState(false)
  const { t } = useI18n()
  return (
    <Card className="flex h-full flex-col">
      <CardHeader>
        <div className="flex items-start justify-between gap-2">
          <div className="flex min-w-0 flex-col gap-1">
            <CardTitle className="truncate" title={profile.name}>
              {profile.name}
            </CardTitle>
            <CardDescription className="truncate font-mono" title={profile.slug}>
              {profile.slug}
            </CardDescription>
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
      </CardContent>
      <CardFooter className="mt-auto justify-end gap-2">
        <Button variant="outline" size="sm" onClick={onEdit}>
          <Pencil data-icon="inline-start" />
          {t('profiles.edit')}
        </Button>
        <Button
          variant="destructive"
          size="sm"
          disabled={deleteMutation.isPending}
          onClick={() => setDeleteOpen(true)}
        >
          <Trash2 data-icon="inline-start" />
          {t('profiles.delete')}
        </Button>
      </CardFooter>
      <ConfirmDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        title={t('profiles.delete')}
        description={t('profiles.deleteConfirm', { name: profile.name })}
        confirmLabel={t('profiles.delete')}
        pending={deleteMutation.isPending}
        onConfirm={() =>
          deleteMutation.mutate(profile.slug, { onSuccess: () => setDeleteOpen(false) })
        }
      />
    </Card>
  )
}
