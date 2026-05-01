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
    if (email) initiateAuthMut.mutate({ email })
  }

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h2 className="text-2xl font-bold tracking-tight">Sign in to Saftaja</h2>
        <p className="mt-2 text-sm text-zinc-500">Enter your email to continue to the dashboard</p>
      </div>

      <form className="space-y-6" onSubmit={handleSubmit}>
        <div>
          <label htmlFor="email" className="block text-sm font-medium text-zinc-700">
            Email Address
          </label>
          <input
            id="email"
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="mt-2 block w-full rounded-lg border border-zinc-300 px-4 py-2.5 placeholder-zinc-400 focus:border-black focus:outline-none focus:ring-1 focus:ring-black sm:text-sm transition-colors"
            placeholder="you@example.com"
            dir="ltr"
          />
        </div>

        {initiateAuthMut.isError && (
          <div className="rounded-md bg-red-50 p-4 text-sm text-red-700">
            {initiateAuthMut.error?.message || 'Failed to initiate authentication.'}
          </div>
        )}

        {initiateAuthMut.isSuccess && (
          <div className="rounded-md bg-green-50 p-4 text-sm text-green-700">
            Check your email for the next steps!
          </div>
        )}

        <button
          type="submit"
          disabled={initiateAuthMut.isPending || !email}
          className="flex w-full justify-center rounded-lg bg-zinc-900 px-4 py-2.5 text-sm font-semibold text-white hover:bg-zinc-800 disabled:opacity-50 transition-all"
        >
          {initiateAuthMut.isPending ? 'Sending...' : 'Continue'}
        </button>
      </form>
    </div>
  )
}
