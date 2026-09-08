import type { ReactNode } from 'react'
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'

interface QueryStateProps<T> {
  query: {
    isPending: boolean
    isError: boolean
    error: Error | null
    data: T | undefined
  }
  /** Loading placeholder, typically a grid of Skeletons. */
  skeleton: ReactNode
  errorTitle: string
  /** When provided, array data with length 0 renders this empty state. */
  emptyTitle?: string
  emptyDescription?: string
  emptyContent?: ReactNode
  children: (data: T) => ReactNode
}

/** Uniform isPending → Skeleton / isError → Empty / empty → Empty tri-state branch. */
export function QueryState<T>({
  query,
  skeleton,
  errorTitle,
  emptyTitle,
  emptyDescription,
  emptyContent,
  children,
}: QueryStateProps<T>) {
  if (query.isPending) return <>{skeleton}</>
  if (query.isError) {
    return (
      <Empty className="m-auto max-w-md">
        <EmptyHeader>
          <EmptyTitle>{errorTitle}</EmptyTitle>
          {query.error && <EmptyDescription>{query.error.message}</EmptyDescription>}
        </EmptyHeader>
      </Empty>
    )
  }
  if (query.data === undefined) return null
  if (emptyTitle !== undefined && Array.isArray(query.data) && query.data.length === 0) {
    return (
      <Empty className="m-auto max-w-md">
        <EmptyHeader>
          <EmptyTitle>{emptyTitle}</EmptyTitle>
          {emptyDescription && <EmptyDescription>{emptyDescription}</EmptyDescription>}
        </EmptyHeader>
        {emptyContent && <EmptyContent>{emptyContent}</EmptyContent>}
      </Empty>
    )
  }
  return <>{children(query.data)}</>
}
