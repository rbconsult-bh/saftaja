import { useState } from 'react'
import { useMutation } from '@connectrpc/connect-query'
import { createFileRoute } from '@tanstack/react-router'
import { initiateAuth } from '../../gen/saftaja/dashboard/auth/v1/auth-AuthService_connectquery'

export const Route = createFileRoute('/_auth/auth')({
  component: RouteComponent,
})

function RouteComponent() {
  const [email, setEmail] = useState('')
  const initiateAuthMut = useMutation(initiateAuth)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!email) return

    initiateAuthMut.mutate({ email })
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-50 p-4 font-sans">
      <div className="w-full max-w-md space-y-8 rounded-2xl bg-white p-8 shadow-sm border border-zinc-100">
        <div className="text-center">
          <h2 className="text-2xl font-bold tracking-tight text-zinc-900">
            Sign in to Saftaja
          </h2>
          <p className="mt-2 text-sm text-zinc-500">
            Enter your email to continue to the dashboard
          </p>
        </div>

        <form className="mt-8 space-y-6" onSubmit={handleSubmit}>
          <div>
            <label htmlFor="email" className="block text-sm font-medium text-zinc-700">
              Email Address
            </label>
            <div className="mt-2">
              <input
                id="email"
                name="email"
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="block w-full rounded-lg border border-zinc-300 px-4 py-2.5 text-zinc-900 placeholder-zinc-400 focus:border-black focus:outline-none focus:ring-1 focus:ring-black sm:text-sm transition-colors"
                placeholder="you@example.com"
              />
            </div>
          </div>

          {/* ConnectRPC Error State */}
          {initiateAuthMut.isError && (
            <div className="rounded-md bg-red-50 p-4">
              <p className="text-sm text-red-700">
                {initiateAuthMut.error?.message || 'Failed to initiate authentication.'}
              </p>
            </div>
          )}

          {/* ConnectRPC Success State */}
          {initiateAuthMut.isSuccess && (
            <div className="rounded-md bg-green-50 p-4">
              <p className="text-sm text-green-700">
                Check your email for the next steps!
              </p>
            </div>
          )}

          <button
            type="submit"
            disabled={initiateAuthMut.isPending || !email}
            className="flex w-full justify-center rounded-lg bg-zinc-900 px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-zinc-800 focus:outline-none focus:ring-2 focus:ring-zinc-900 focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed transition-all"
          >
            {initiateAuthMut.isPending ? 'Sending...' : 'Continue'}
          </button>
        </form>
      </div>
    </div>
  )
}
