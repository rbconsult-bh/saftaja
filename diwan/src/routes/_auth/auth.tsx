import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/_auth/auth')({
  component: RouteComponent,
})

function RouteComponent() {
  return <div>Hello "/_auth/login"!</div>
}
