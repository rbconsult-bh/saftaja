import { createFileRoute, Link, useNavigate } from '@tanstack/react-router'
import { useMutation } from '@connectrpc/connect-query'
import { completeAuth } from '../../gen/saftaja/dashboard/auth/v1/auth-AuthService_connectquery'
import { useSessionStore } from '../../core/session/store'

type MagicLinkSearch = {
  token?: string
}

export const Route = createFileRoute('/_auth/magic-link')({
  validateSearch: (search: Record<string, unknown>): MagicLinkSearch => {
    return {
      token: search.token as string | undefined,
    }
  },
  component: RouteComponent,
})

function RouteComponent() {
  const { token } = Route.useSearch()
  const login = useSessionStore((state) => state.actions.login)
  const navigate = useNavigate();

  const completeAuthMut = useMutation(completeAuth)

  const handleLoginClick = () => {
    if (!token) return

    completeAuthMut.mutate(
      { token },
      {
        onSuccess: (data) => {
          login(data.accessToken, data.refreshToken)
          navigate({ to: '/project' });
        },
      }
    )
  }

  if (!token) {
    return (
      <AuthLayout title="Invalid Link">
        <p className="text-sm text-zinc-500">No authentication token was found in the URL.</p>
        <Link to="/auth" className="mt-4 block text-sm font-medium text-zinc-900 hover:underline">
          Return to login
        </Link>
      </AuthLayout>
    )
  }

  if (completeAuthMut.isError) {
    return (
      <AuthLayout title="Authentication Failed">
        <div className="rounded-md bg-red-50 p-4 mb-4 text-left">
          <p className="text-sm text-red-700">
            {completeAuthMut.error?.message || 'This link has expired or already been used.'}
          </p>
        </div>
        <Link
          to="/auth"
          className="flex w-full justify-center rounded-lg bg-zinc-900 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-zinc-800 transition-all"
        >
          Request new link
        </Link>
      </AuthLayout>
    )
  }

  return (
    <AuthLayout title="Ready to sign in?">
      <div className="flex flex-col items-center justify-center space-y-6 mt-4">
        <p className="text-sm text-zinc-500">
          Your magic link is verified. Click below to securely enter your Saftaja dashboard.
        </p>

        <button
          onClick={handleLoginClick}
          disabled={completeAuthMut.isPending || completeAuthMut.isSuccess}
          className="flex w-full justify-center rounded-lg bg-zinc-900 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-zinc-800 focus:outline-none focus:ring-2 focus:ring-zinc-900 focus:ring-offset-2 disabled:opacity-50 transition-all"
        >
          {completeAuthMut.isPending ? 'Authenticating...' : completeAuthMut.isSuccess ? 'Redirecting...' : 'Enter Dashboard'}
        </button>
      </div>
    </AuthLayout>
  )
}

function AuthLayout({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-50 p-4 font-sans">
      <div className="w-full max-w-md rounded-2xl bg-white p-8 shadow-sm border border-zinc-100 text-center">
        <h2 className="text-2xl font-bold tracking-tight text-zinc-900">
          {title}
        </h2>
        {children}
      </div>
    </div>
  )
}
