import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { useMutation } from '@connectrpc/connect-query'
import { completeAuth } from '../../gen/saftaja/dashboard/auth/v1/auth-AuthService_connectquery'
import { useSessionStore } from '../../core/session/store'

type MagicLinkSearch = { token?: string }

export const Route = createFileRoute('/_auth/magic-link')({
  validateSearch: (search: Record<string, unknown>): MagicLinkSearch => ({
    token: search.token as string | undefined,
  }),
  component: RouteComponent,
})

function RouteComponent() {
  const { token } = Route.useSearch()
  const login = useSessionStore((state) => state.actions.login)
  const navigate = useNavigate()
  const completeAuthMut = useMutation(completeAuth)

  const handleLoginClick = () => {
    if (token) {
      completeAuthMut.mutate(
        { token },
        {
          onSuccess: (data) => {
            login(data.accessToken, data.refreshToken)
            navigate({ to: '/project' })
          },
        }
      )
    }
  }

  if (!token) {
    return (
      <div className="text-center space-y-4">
        <h2 className="text-2xl font-bold">Invalid Link</h2>
        <p className="text-sm text-zinc-500">No authentication token was found.</p>
        <Link to="/auth" className="block text-sm font-medium hover:underline">Return to login</Link>
      </div>
    )
  }

  if (completeAuthMut.isError) {
    return (
      <div className="text-center space-y-4">
        <h2 className="text-2xl font-bold">Authentication Failed</h2>
        <div className="rounded-md bg-red-50 p-4 text-sm text-red-700 text-left">
          {completeAuthMut.error?.message || 'This link has expired or already been used.'}
        </div>
        <Link to="/auth" className="block w-full rounded-lg bg-zinc-900 px-4 py-2.5 text-sm font-semibold text-white hover:bg-zinc-800">
          Request new link
        </Link>
      </div>
    )
  }

  return (
    <div className="text-center space-y-6">
      <h2 className="text-2xl font-bold">Ready to sign in?</h2>
      <p className="text-sm text-zinc-500">Click below to securely enter your dashboard.</p>
      <button
        onClick={handleLoginClick}
        disabled={completeAuthMut.isPending || completeAuthMut.isSuccess}
        className="w-full rounded-lg bg-zinc-900 px-4 py-2.5 text-sm font-semibold text-white hover:bg-zinc-800 disabled:opacity-50"
      >
        {completeAuthMut.isPending ? 'Authenticating...' : 'Enter Dashboard'}
      </button>
    </div>
  )
}
