# Twenty-Inspired UI Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate CitaSpot dashboard from violet/shadow-based design to Indigo Deep/border-based design with a dark collapsible sidebar, inspired by Twenty CRM.

**Architecture:** Update Tailwind design tokens first (colors, radii, shadows), then rewrite UI components to use new tokens, then rebuild the sidebar as dark/collapsible, and finally update settings layout colors.

**Tech Stack:** Next.js 14, Tailwind CSS 3, TypeScript, lucide-react, localStorage for sidebar state

---

## File Structure

### Modified files
- `apps/web/tailwind.config.ts` — new color palette, border-radius, shadow overrides
- `apps/web/components/ui/card.tsx` — remove shadow, change radius
- `apps/web/components/ui/button.tsx` — update primary colors and radius
- `apps/web/components/ui/input.tsx` — update focus ring color and radius
- `apps/web/components/ui/badge.tsx` — update primary variant colors
- `apps/web/components/dashboard/sidebar.tsx` — full rewrite: dark, collapsible
- `apps/web/app/(dashboard)/layout.tsx` — support sidebar collapse state
- `apps/web/app/dashboard/layout.tsx` — mirror changes
- `apps/web/app/(dashboard)/settings/layout.tsx` — update active state colors

---

## Task 1: Tailwind Config — New Design Tokens

**Files:**
- Modify: `apps/web/tailwind.config.ts`

- [ ] **Step 1: Replace colors, borderRadius, and boxShadow**

Replace the entire `theme.extend` colors, borderRadius, and boxShadow sections:

```typescript
colors: {
  // Primary — Indigo Deep (Twenty-inspired)
  primary: {
    50:  '#eef2ff',
    100: '#e0e7ff',
    200: '#c7d2fe',
    300: '#a5b4fc',
    400: '#818cf8',
    500: '#6366f1',
    600: '#4f46e5',
    700: '#4338ca',
    800: '#3730a3',
    900: '#312e81',
    950: '#1e1b4b',
  },
  // Accent — rose (citas, urgencia, CTAs secundarios)
  accent: {
    50:  '#fff1f2',
    100: '#ffe4e6',
    500: '#f43f5e',
    600: '#e11d48',
    700: '#be123c',
  },
  // Semantic
  success: '#10b981',
  warning: '#f59e0b',
  error:   '#ef4444',
  // Neutrals — cool grays
  neutral: {
    50:  '#fafafa',
    100: '#f5f5f5',
    200: '#e5e5e5',
    300: '#d4d4d4',
    400: '#a3a3a3',
    500: '#737373',
    600: '#525252',
    700: '#404040',
    800: '#262626',
    900: '#171717',
    950: '#0a0a0a',
  },
  // Sidebar dark palette
  sidebar: {
    bg:     '#1c1c1e',
    border: '#2c2c2e',
    text:   '#8e8e93',
    active: '#2c2c2e',
    muted:  '#636366',
  },
},
```

Replace borderRadius:
```typescript
borderRadius: {
  sm:      '4px',
  DEFAULT: '8px',
  md:      '8px',
  lg:      '8px',
  xl:      '12px',
  '2xl':   '16px',
},
```

Replace boxShadow:
```typescript
boxShadow: {
  sm:      'none',
  DEFAULT: 'none',
  md:      '0 4px 6px -1px rgb(0 0 0 / 0.05), 0 2px 4px -2px rgb(0 0 0 / 0.05)',
  lg:      '0 10px 15px -3px rgb(0 0 0 / 0.08), 0 4px 6px -4px rgb(0 0 0 / 0.08)',
  xl:      '0 20px 25px -5px rgb(0 0 0 / 0.08), 0 8px 10px -6px rgb(0 0 0 / 0.08)',
},
```

- [ ] **Step 2: Update the comment at top of config**

Change the comment from:
```typescript
// Design tokens CitaSpot — 8px base unit, Inter, violeta-púrpura
```
To:
```typescript
// Design tokens CitaSpot — 8px base unit, Inter, Indigo Deep (Twenty-inspired)
```

- [ ] **Step 3: Verify build**

```bash
cd apps/web && npx next build --no-lint 2>&1 | tail -5
```

- [ ] **Step 4: Commit**

```bash
git add apps/web/tailwind.config.ts
git commit -m "style: migrate design tokens to Indigo Deep palette, flat borders, 8px radius"
```

---

## Task 2: UI Components — Card, Button, Input, Badge

**Files:**
- Modify: `apps/web/components/ui/card.tsx`
- Modify: `apps/web/components/ui/button.tsx`
- Modify: `apps/web/components/ui/input.tsx`
- Modify: `apps/web/components/ui/badge.tsx`

- [ ] **Step 1: Update Card — remove shadow, change radius**

In `card.tsx`, change the base classes in the `Card` component:

```
Old: 'rounded-xl border border-neutral-200 bg-white shadow-sm'
New: 'rounded-lg border border-neutral-200 bg-white'
```

- [ ] **Step 2: Update Button — primary variant colors and radius**

In `button.tsx`, update the `variants` object:

```typescript
const variants: Record<ButtonVariant, string> = {
  primary:
    'bg-primary-800 text-white hover:bg-primary-700 focus-visible:ring-primary-600',
  secondary:
    'bg-white text-neutral-700 border border-neutral-200 hover:bg-neutral-50 focus-visible:ring-primary-600',
  ghost:
    'text-neutral-600 hover:bg-neutral-100 focus-visible:ring-primary-600',
  danger:
    'bg-error text-white hover:bg-red-600 focus-visible:ring-red-500',
};
```

Update the `sizes` object to use consistent 8px radius:

```typescript
const sizes: Record<ButtonSize, string> = {
  sm: 'h-8  px-3 text-sm  rounded-lg',
  md: 'h-9  px-4 text-sm  rounded-lg',
  lg: 'h-11 px-6 text-base rounded-lg',
};
```

- [ ] **Step 3: Update Input — focus ring color and radius**

In `input.tsx`, update the className:

```
Old: 'h-9 w-full rounded-md border bg-white px-3 text-sm text-neutral-900'
New: 'h-9 w-full rounded-lg border bg-white px-3 text-sm text-neutral-900'
```

```
Old: 'focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-0 focus:border-primary-500'
New: 'focus:outline-none focus:ring-2 focus:ring-primary-600 focus:ring-offset-0 focus:border-primary-600'
```

- [ ] **Step 4: Update Badge — primary variant colors**

In `badge.tsx`, update the `primary` variant:

```
Old: primary: 'bg-primary-100 text-primary-700'
New: primary: 'bg-primary-100 text-primary-800'
```

- [ ] **Step 5: Commit**

```bash
git add apps/web/components/ui/
git commit -m "style: update Card, Button, Input, Badge to Indigo Deep tokens"
```

---

## Task 3: Sidebar — Dark Theme + Collapsible

**Files:**
- Modify: `apps/web/components/dashboard/sidebar.tsx` (full rewrite)

- [ ] **Step 1: Rewrite sidebar.tsx**

```tsx
'use client';

import { useState, useEffect } from 'react';
import Image from 'next/image';
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

  // Restore sidebar state from localStorage
  useEffect(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === 'collapsed') setCollapsed(true);
  }, []);

  function toggleCollapse() {
    const next = !collapsed;
    setCollapsed(next);
    localStorage.setItem(STORAGE_KEY, next ? 'collapsed' : 'expanded');
  }

  // Close mobile sidebar on route change
  useEffect(() => {
    setMobileOpen(false);
  }, [pathname]);

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
```

- [ ] **Step 2: Commit**

```bash
git add apps/web/components/dashboard/sidebar.tsx
git commit -m "style: rewrite sidebar with dark theme and collapsible support"
```

---

## Task 4: Dashboard Layout — Support Sidebar + Mobile

**Files:**
- Modify: `apps/web/app/(dashboard)/layout.tsx`
- Modify: `apps/web/app/dashboard/layout.tsx`

- [ ] **Step 1: Update (dashboard)/layout.tsx**

The current layout uses `bg-neutral-50` for the main background. Update the flex container:

Change:
```tsx
<div className="flex h-screen overflow-hidden bg-neutral-50">
  <Sidebar />
  <main className="flex-1 overflow-y-auto">
    {children}
  </main>
</div>
```

To:
```tsx
<div className="flex h-screen overflow-hidden bg-white">
  <Sidebar />
  <main className="flex-1 overflow-y-auto bg-neutral-50">
    <div className="md:hidden h-12" /> {/* Spacer for mobile hamburger */}
    {children}
  </main>
</div>
```

Note: `bg-white` on the outer container ensures no gap colors show during sidebar transitions. The spacer div prevents content from hiding behind the fixed hamburger button on mobile.

- [ ] **Step 2: Update dashboard/layout.tsx to mirror**

Read the file first. If it re-exports from `(dashboard)/layout.tsx`, no changes needed. If it has its own layout code, apply the same changes.

- [ ] **Step 3: Verify in browser**

Open `http://localhost:3000/dashboard` and verify:
- Sidebar is dark (#1c1c1e)
- Collapse/expand works with smooth animation
- State persists on refresh
- On mobile viewport: sidebar hidden, hamburger button visible

- [ ] **Step 4: Commit**

```bash
git add apps/web/app/\(dashboard\)/layout.tsx apps/web/app/dashboard/layout.tsx
git commit -m "style: update dashboard layout for dark sidebar and mobile support"
```

---

## Task 5: Settings Layout — Update Colors

**Files:**
- Modify: `apps/web/app/(dashboard)/settings/layout.tsx`

- [ ] **Step 1: Update active state colors and remove shadows**

In the desktop nav active state, change:
```
Old: 'bg-primary-50 text-primary-700'
New: 'bg-primary-50 text-primary-800'
```

In the mobile nav active state, change:
```
Old: 'bg-white text-neutral-900 shadow-sm'
New: 'bg-white text-neutral-900'
```

In the mobile nav container, change:
```
Old: 'rounded-xl bg-neutral-100'
New: 'rounded-lg bg-neutral-100'
```

- [ ] **Step 2: Commit**

```bash
git add apps/web/app/\(dashboard\)/settings/layout.tsx
git commit -m "style: update settings layout to Indigo Deep active states"
```

---

## Verification

1. **Design tokens propagation:** Open any dashboard page — cards should have no shadows, 8px radius, indigo buttons
2. **Sidebar:** Dark background, smooth collapse/expand animation (200ms), state persists in localStorage
3. **Mobile:** Hamburger button shows, sidebar opens as overlay with backdrop, closes on tap
4. **Settings:** Sub-navigation uses indigo-50/800 active state, no shadows on mobile pills
5. **Buttons:** Primary buttons are #3730a3 (Indigo Deep), hover lightens to #4338ca
6. **Inputs:** Focus ring is indigo-600, 8px radius
7. **Badges:** Primary variant uses indigo-100/800
8. **No regressions:** Spot check agenda, services, team pages — they should inherit new tokens without code changes
