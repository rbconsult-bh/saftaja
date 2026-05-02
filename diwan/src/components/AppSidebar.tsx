import { LayoutDashboard, Settings } from "lucide-react"
import { Link } from "@tanstack/react-router"
import { m } from "@/paraglide/messages"
import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "@/components/ui/sidebar"
import { useLanguage } from "./LanguageProvider";

export function AppSidebar() {
  const { dir } = useLanguage();

  const items = [
    {
      title: m.dashboard_nav_projects(),
      url: "/project",
      icon: LayoutDashboard,
    },
    {
      title: m.dashboard_nav_settings(),
      url: "/settings",
      icon: Settings,
    },
  ]

  return (
    <Sidebar collapsible="icon" side={dir == "rtl" ? "right" : "left"}>
      <SidebarHeader className="h-16 flex items-center px-6 border-b">
        <span className="font-bold text-xl tracking-tight">{m.saftaja()}</span>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>{m.dashboard_nav_group()}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {items.map((item) => (
                <SidebarMenuItem key={item.url}>
                  <SidebarMenuButton asChild tooltip={item.title}>
                    <Link
                      to={item.url}
                      activeProps={{ className: "bg-sidebar-accent text-sidebar-accent-foreground" }}
                    >
                      <item.icon />
                      <span>{item.title}</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  )
}

