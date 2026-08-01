import { createFileRoute } from '@tanstack/react-router'
import { Skeleton } from '@/components/ui/skeleton'

export const Route = createFileRoute('/_dashboard/$projectId/settings')({
  component: RouteComponent,
})

function RouteComponent() {
  return (
    <div className="p-6 space-y-4">
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-4 w-96" />
      <Skeleton className="h-64 w-full rounded-lg" />
    </div>
  )
}
