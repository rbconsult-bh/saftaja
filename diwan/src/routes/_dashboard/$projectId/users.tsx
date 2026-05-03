import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_dashboard/$projectId/users')({
  component: RouteComponent,
})

function RouteComponent() {
  // TODO: call dashboard users RPC here
  return null
}
