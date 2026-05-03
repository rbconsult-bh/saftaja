import {
  LayoutDashboard,
  Settings,
  ReceiptText,
  CreditCard,
  Users,
  LogOut,
  ChevronsUpDown,
  Plus,
  type LucideIcon
} from "lucide-react"
import { Link, useNavigate, useParams } from "@tanstack/react-router"
import { m } from "@/paraglide/messages"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarGroupContent
} from "@/components/ui/sidebar"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger
} from "@/components/ui/dropdown-menu"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { useLanguage } from "./LanguageProvider"
import { useSessionStore } from "@/core/session/store"
import { useMutation } from "@connectrpc/connect-query"
import { logout as logoutRpc } from "@/gen/saftaja/dashboard/auth/v1/auth-AuthService_connectquery"

export function AppSidebar() {
  const { dir } = useLanguage()
  const params = useParams({ strict: false }) as { projectId?: string }
  const pid = params.projectId || "default-project"

  return (
    <Sidebar collapsible="icon" side={dir === "rtl" ? "right" : "left"}>
      <SidebarHeader className="h-16 border-b border-sidebar-border justify-center">
        <WorkspaceSwitcher activeProjectId={pid} />
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>{m.dashboard_nav_group()}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <NavMenuItem
                to="/$projectId"
                projectId={pid}
                icon={LayoutDashboard}
                label={m.dashboard_nav_projects()}
              />
              <NavMenuItem
                to="/$projectId/invoices"
                projectId={pid}
                icon={ReceiptText}
                label={m.dashboard_nav_invoices()}
              />
              <NavMenuItem
                to="/$projectId/gateways"
                projectId={pid}
                icon={CreditCard}
                label={m.dashboard_nav_gateways()}
              />
              <NavMenuItem
                to="/$projectId/users"
                projectId={pid}
                icon={Users}
                label={m.dashboard_nav_users()}
              />
              <NavMenuItem
                to="/$projectId/settings"
                projectId={pid}
                icon={Settings}
                label={m.dashboard_nav_settings()}
              />
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter className="border-t border-sidebar-border p-2">
        <UserMenu projectId={pid} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}

type ProjectRoute = '/$projectId' | '/$projectId/invoices' | '/$projectId/gateways' | '/$projectId/users' | '/$projectId/settings'

function NavMenuItem({ to, projectId, icon: Icon, label }: { to: ProjectRoute, projectId: string, icon: LucideIcon, label: string }) {
  return (
    <SidebarMenuItem>
      <SidebarMenuButton asChild tooltip={label}>
        <Link
          to={to}
          params={{ projectId }}
          activeProps={{ className: "bg-sidebar-accent text-sidebar-accent-foreground font-medium" }}
        >
          <Icon className="size-4" />
          <span>{label}</span>
        </Link>
      </SidebarMenuButton>
    </SidebarMenuItem>
  )
}

function WorkspaceSwitcher({ activeProjectId }: { activeProjectId: string }) {
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton size="lg" className="data-[state=open]:bg-sidebar-accent">
              <div className="flex aspect-square size-8 items-center justify-center rounded-lg bg-zinc-900 text-white font-bold">
                S
              </div>
              <div className="grid flex-1 text-left text-sm leading-tight group-data-[collapsible=icon]:hidden">
                <span className="truncate font-semibold">{m.dashboard_workspace_name_placeholder()}</span>
                <span className="truncate text-xs text-muted-foreground">{activeProjectId}</span>
              </div>
              <ChevronsUpDown className="ml-auto size-4 group-data-[collapsible=icon]:hidden" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="w-[--radix-dropdown-menu-trigger-width] min-w-56" align="start" sideOffset={4}>
            <DropdownMenuLabel className="text-xs text-muted-foreground">{m.dashboard_workspace_group_label()}</DropdownMenuLabel>
            {/* TODO: call dashboard workspace/project RPC here */}
            <DropdownMenuItem className="gap-2 p-2">
              <div className="flex size-6 items-center justify-center rounded-sm border">S</div>
              {m.dashboard_workspace_name_placeholder()}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="gap-2 p-2">
              <Plus className="size-4" />
              <span className="font-medium text-muted-foreground text-xs">{m.dashboard_workspace_create_project()}</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}

function UserMenu({ projectId }: { projectId: string }) {
  const navigate = useNavigate()
  const storeLogout = useSessionStore((state) => state.actions.logout)
  const logoutMut = useMutation(logoutRpc)

  const handleLogout = async () => {
    logoutMut.mutate({}, {
      onSettled: () => {
        storeLogout()
        navigate({ to: '/auth' })
      }
    })
  }

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton size="lg" className="hover:bg-sidebar-accent transition-colors">
              <Avatar className="h-8 w-8 rounded-lg">
                <AvatarFallback className="rounded-lg bg-zinc-100 text-xs font-bold text-zinc-900">
                  ME
                </AvatarFallback>
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight group-data-[collapsible=icon]:hidden">
                <span className="truncate font-semibold text-zinc-900">{m.dashboard_user_name_placeholder()}</span>
                <span className="truncate text-xs text-muted-foreground">{m.dashboard_user_email_placeholder()}</span>
              </div>
              <ChevronsUpDown className="ml-auto size-4 group-data-[collapsible=icon]:hidden" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent className="w-[--radix-dropdown-menu-trigger-width] min-w-56" align="end" side="top" sideOffset={8}>
            <DropdownMenuLabel className="p-0 font-normal">
              <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                <Avatar className="h-8 w-8 rounded-lg">
                  <AvatarFallback className="rounded-lg font-bold">ME</AvatarFallback>
                </Avatar>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-semibold text-zinc-900">{m.dashboard_user_name_placeholder()}</span>
                  <span className="truncate text-xs text-muted-foreground">{m.dashboard_user_email_placeholder()}</span>
                </div>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <Link to="/$projectId/settings" params={{ projectId }} className="w-full flex items-center cursor-pointer">
                <Settings className="mr-2 size-4" /> {m.dashboard_global_settings()}
              </Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={handleLogout}
              disabled={logoutMut.isPending}
              className="text-destructive focus:text-destructive cursor-pointer"
            >
              <LogOut className="mr-2 size-4" />
              {m.dashboard_logout()}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
