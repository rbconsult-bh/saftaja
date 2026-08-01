import { useState } from 'react'
import { useMutation } from '@connectrpc/connect-query'
import { createFileRoute } from '@tanstack/react-router'
import { initiateAuth } from '../../gen/saftaja/dashboard/auth/v1/auth-AuthService_connectquery'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Loader2, Mail } from "lucide-react"
import { m } from "@/paraglide/messages"

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
      <div className="text-center space-y-2">
        <h2 className="text-2xl font-bold tracking-tight">{m.auth_login_title()}</h2>
        <p className="text-sm text-muted-foreground">{m.auth_login_subtitle()}</p>
      </div>

      <form className="space-y-4" onSubmit={handleSubmit}>
        <div className="grid gap-2">
          <Label htmlFor="email" className="rtl:text-right">{m.auth_login_email_label()}</Label>
          <div className="relative">
            <Mail className="absolute left-3 top-3 h-4 w-4 text-muted-foreground rtl:right-3 rtl:left-auto" />
            <Input
              id="email"
              type="email"
              placeholder={m.auth_login_email_placeholder()}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="pl-10 rtl:pr-10 rtl:pl-3"
              required
            />
          </div>
        </div>

        {initiateAuthMut.isError && (
          <Alert variant="destructive">
            <AlertDescription>
              {initiateAuthMut.error?.message || m.common_error()}
            </AlertDescription>
          </Alert>
        )}

        {initiateAuthMut.isSuccess && (
          <Alert className="border-green-500 bg-green-50 dark:bg-green-950/20">
            <AlertDescription className="text-green-600 dark:text-green-400">
              {m.auth_login_success()}
            </AlertDescription>
          </Alert>
        )}

        <Button
          type="submit"
          className="w-full"
          disabled={initiateAuthMut.isPending || !email}
        >
          {initiateAuthMut.isPending ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin rtl:ml-2 rtl:mr-0" />
              {m.auth_login_sending()}
            </>
          ) : m.auth_login_button()}
        </Button>
      </form>
    </div>
  )
}
