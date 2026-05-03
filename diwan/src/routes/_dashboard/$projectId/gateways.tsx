import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_dashboard/$projectId/gateways')({
  component: RouteComponent,
})

function RouteComponent() {
  // TODO: call dashboard gateways RPC here
  return null
}
