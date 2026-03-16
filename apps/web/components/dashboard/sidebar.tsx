'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
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
} from 'lucide-react';
import { createClient } from '@/lib/supabase/browser';
import { useRouter } from 'next/navigation';
import { useTranslations } from '@/lib/i18n';

export function Sidebar() {
  const pathname = usePathname();
  const router = useRouter();
  const t = useTranslations();

  const NAV_ITEMS = [
    { href: '/dashboard',            icon: LayoutDashboard, label: t.nav.dashboard   },
    { href: '/dashboard/agenda',     icon: CalendarDays,    label: t.nav.agenda      },
    { href: '/dashboard/clients',    icon: Users,           label: t.nav.clients     },
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

  return (
    <aside className="flex h-screen w-60 flex-col border-r border-neutral-200 bg-white">
      {/* Logo */}
      <div className="flex h-16 items-center gap-3 border-b border-neutral-100 px-5">
        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary-600 text-white text-sm font-bold">
          A
        </div>
        <span className="font-semibold text-neutral-900">CitaSpot</span>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto px-3 py-4">
        <ul className="flex flex-col gap-0.5">
          {NAV_ITEMS.map(({ href, icon: Icon, label }) => {
            const active = pathname === href || (href !== '/dashboard' && pathname.startsWith(href));
            return (
              <li key={href}>
                <Link
                  href={href}
                  className={cn(
                    'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors duration-150',
                    active
                      ? 'bg-primary-50 text-primary-700'
                      : 'text-neutral-600 hover:bg-neutral-50 hover:text-neutral-900',
                  )}
                >
                  <Icon className="h-4 w-4 flex-shrink-0" />
                  {label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      {/* Logout */}
      <div className="border-t border-neutral-100 p-3">
        <button
          onClick={handleLogout}
          className="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium text-neutral-500 transition-colors hover:bg-neutral-50 hover:text-neutral-700"
        >
          <LogOut className="h-4 w-4" />
          {t.nav.logout}
        </button>
      </div>
    </aside>
  );
}
