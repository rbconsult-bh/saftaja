import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_dashboard/$projectId/invoices')({
  component: RouteComponent,
})

function RouteComponent() {
  // TODO: call dashboard invoices RPC here
  return null
}
