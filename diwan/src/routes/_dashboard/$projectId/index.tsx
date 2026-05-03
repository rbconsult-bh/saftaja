import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_dashboard/$projectId/')({
  component: RouteComponent,
})

function RouteComponent() {
  // TODO: call dashboard project overview RPC here
  return null
}
