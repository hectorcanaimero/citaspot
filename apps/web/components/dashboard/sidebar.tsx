'use client';

import { useState, useEffect } from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import { cn } from '@/lib/utils';
import {
  CalendarDays,
  Users,
  Scissors,
  BookOpen,
  MessageCircle,
  BarChart3,
  Settings,
  LogOut,
  UserCog,
  LayoutDashboard,
  HelpCircle,
  ChevronsLeft,
  ChevronsRight,
  Menu,
  X,
  Kanban,
  Stethoscope,
  CheckSquare,
  Zap,
} from 'lucide-react';
import { createClient } from '@/lib/supabase/browser';
import { useTranslations } from '@/lib/i18n';

const STORAGE_KEY = 'citaspot_sidebar';

export function Sidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const t = useTranslations();

  const [collapsed, setCollapsed] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === 'collapsed') setCollapsed(true);
  }, []);

  function toggleCollapse() {
    const next = !collapsed;
    setCollapsed(next);
    localStorage.setItem(STORAGE_KEY, next ? 'collapsed' : 'expanded');
  }

  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

  const NAV_ITEMS = [
    { href: '/dashboard',            icon: LayoutDashboard, label: t.nav.dashboard   },
    { href: '/dashboard/agenda',     icon: CalendarDays,    label: t.nav.agenda      },
    { href: '/dashboard/clients',    icon: Users,           label: t.nav.clients     },
    { href: '/dashboard/pipeline',     icon: Kanban,       label: t.nav.pipeline     },
    { href: '/dashboard/treatments',   icon: Stethoscope,  label: t.nav.treatments   },
    { href: '/dashboard/tasks',        icon: CheckSquare,  label: t.nav.tasks        },
    { href: '/dashboard/automations',  icon: Zap,          label: t.nav.automations  },
    { href: '/dashboard/services',   icon: Scissors,        label: t.nav.services    },
    { href: '/dashboard/team',       icon: UserCog,         label: t.nav.team        },
    { href: '/dashboard/knowledge',  icon: BookOpen,        label: t.nav.knowledge   },
    { href: '/dashboard/whatsapp',   icon: MessageCircle,   label: t.nav.whatsapp    },
    { href: '/dashboard/analytics',  icon: BarChart3,       label: t.nav.analytics   },
    { href: '/dashboard/settings',   icon: Settings,        label: t.nav.settings    },
    { href: '/dashboard/faq',        icon: HelpCircle,      label: t.nav.help        },
  ];

  async function handleLogout() {
    const supabase = createClient();
    await supabase.auth.signOut();
    router.push('/login');
    router.refresh();
  }

  const sidebarContent = (
    <>
      {/* Header */}
      <div className={cn(
        'flex items-center border-b border-sidebar-border h-14',
        collapsed ? 'justify-center px-0' : 'justify-between px-4',
      )}>
        {collapsed ? (
          <span className="text-white font-bold text-lg">C</span>
        ) : (
          <span className="text-white font-bold text-sm tracking-tight">CitaSpot</span>
        )}
        <button
          onClick={toggleCollapse}
          className="hidden md:flex items-center justify-center text-sidebar-muted hover:text-white transition-colors"
          title={collapsed ? 'Expandir' : 'Colapsar'}
        >
          {collapsed ? <ChevronsRight className="h-4 w-4" /> : <ChevronsLeft className="h-4 w-4" />}
        </button>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto p-1.5">
        <ul className="flex flex-col gap-px">
          {NAV_ITEMS.map(({ href, icon: Icon, label }) => {
            const active = pathname === href || (href !== '/dashboard' && pathname.startsWith(href));
            return (
              <li key={href}>
                <Link
                  href={href}
                  title={collapsed ? label : undefined}
                  className={cn(
                    'flex items-center rounded-md text-[13px] font-medium transition-colors duration-100',
                    collapsed ? 'justify-center h-9 w-9 mx-auto' : 'gap-2.5 px-3 py-2',
                    active
                      ? 'bg-sidebar-active text-white'
                      : 'text-sidebar-text hover:text-neutral-200 hover:bg-sidebar-active/50',
                  )}
                >
                  <Icon className="h-4 w-4 flex-shrink-0" />
                  {!collapsed && label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      {/* Logout */}
      <div className="border-t border-sidebar-border p-1.5">
        <button
          onClick={handleLogout}
          title={collapsed ? t.nav.logout : undefined}
          className={cn(
            'flex w-full items-center rounded-md text-[13px] font-medium text-sidebar-muted transition-colors hover:text-neutral-300 hover:bg-sidebar-active/50',
            collapsed ? 'justify-center h-9 w-9 mx-auto' : 'gap-2.5 px-3 py-2',
          )}
        >
          <LogOut className="h-4 w-4" />
          {!collapsed && t.nav.logout}
        </button>
      </div>
    </>
  );

  return (
    <>
      {/* Mobile hamburger */}
      <button
        onClick={() => setMobileOpen(true)}
        className="fixed top-3 left-3 z-40 flex md:hidden items-center justify-center h-9 w-9 rounded-lg bg-sidebar-bg text-white"
      >
        <Menu className="h-5 w-5" />
      </button>

      {/* Mobile overlay */}
      {mobileOpen && (
        <div className="fixed inset-0 z-50 md:hidden">
          <div
            className="absolute inset-0 bg-black/40"
            onClick={() => setMobileOpen(false)}
          />
          <aside className="relative flex h-screen w-60 flex-col bg-sidebar-bg">
            <button
              onClick={() => setMobileOpen(false)}
              className="absolute top-3 right-3 text-sidebar-muted hover:text-white"
            >
              <X className="h-5 w-5" />
            </button>
            {sidebarContent}
          </aside>
        </div>
      )}

      {/* Desktop sidebar */}
      <aside
        className={cn(
          'hidden md:flex h-screen flex-col bg-sidebar-bg transition-all duration-200 ease-in-out flex-shrink-0',
          collapsed ? 'w-14' : 'w-[220px]',
        )}
      >
        {sidebarContent}
      </aside>
    </>
  );
}
