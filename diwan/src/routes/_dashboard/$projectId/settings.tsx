import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_dashboard/$projectId/settings')({
  component: RouteComponent,
})

function RouteComponent() {
  // TODO: call dashboard project settings RPC here
  return null
}
