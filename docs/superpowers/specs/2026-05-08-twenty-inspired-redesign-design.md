# CitaSpot UI Redesign — Twenty-Inspired Design Spec

## Context

CitaSpot's dashboard currently uses a violet (#8b5cf6) primary color, warm grays, rounded-xl (16px) corners, and shadow-sm cards. The goal is to adopt Twenty CRM's design philosophy: monocromatic, clean, professional, content-over-decoration — while maintaining CitaSpot's identity as a health/beauty SaaS.

## Design Decisions (Validated)

| Decision | Value |
|----------|-------|
| Primary color | Indigo Deep `#3730a3` (Tailwind indigo-800) |
| Surfaces | Borders only, no box-shadows on cards |
| Border radius | 8px default (Tailwind `rounded-lg`) |
| Sidebar | Dark background (#1c1c1e), collapsible to icons (56px) |
| Scope | Design system tokens + sidebar + settings sub-pages |

## Scope

### In scope
1. **Tailwind config** — new color palette, border-radius defaults, remove shadows
2. **Sidebar** — dark theme, collapsible to icons, state persisted in localStorage
3. **UI components** — update Card, Button, Badge, Input to use new tokens
4. **Settings sub-pages** — apply new design to the 4 settings pages (business, account, subscription, customization)

### Out of scope (inherits tokens automatically)
- Dashboard overview page
- Agenda/calendar page
- Clients, services, team, knowledge, whatsapp, analytics pages
- Public booking page
- These pages will pick up new colors/radii/borders from the updated tokens without code changes

---

## 1. Color Palette

Replace the current violet-based palette with Indigo Deep.

### Primary (Indigo Deep)
```
primary-50:  #eef2ff
primary-100: #e0e7ff
primary-200: #c7d2fe
primary-300: #a5b4fc
primary-400: #818cf8
primary-500: #6366f1
primary-600: #4f46e5
primary-700: #4338ca
primary-800: #3730a3  ← main brand color
primary-900: #312e81
primary-950: #1e1b4b
```

### Neutrals (Cool grays — shift from warm to cool)
```
neutral-50:  #fafafa
neutral-100: #f5f5f5
neutral-200: #e5e5e5
neutral-300: #d4d4d4
neutral-400: #a3a3a3
neutral-500: #737373
neutral-600: #525252
neutral-700: #404040
neutral-800: #262626
neutral-900: #171717
neutral-950: #0a0a0a
```

### Sidebar Dark
```
sidebar-bg:      #1c1c1e
sidebar-border:  #2c2c2e
sidebar-text:    #8e8e93
sidebar-active:  #2c2c2e (bg) + #ffffff (text)
sidebar-muted:   #636366
```

### Semantic (unchanged)
```
success: #10b981
warning: #f59e0b / #d97706
error:   #ef4444
info:    #3b82f6
```

### Accent (keep rose for urgency/secondary)
```
accent-500: #f43f5e
accent-600: #e11d48
```

---

## 2. Sidebar Component

### Structure
- **Expanded width:** 220px
- **Collapsed width:** 56px
- **Background:** #1c1c1e (sidebar-bg)
- **Border:** right border 1px #2c2c2e (sidebar-border)
- **Collapse toggle:** button at top-right of sidebar header
- **State persistence:** `localStorage.setItem('citaspot_sidebar', 'collapsed' | 'expanded')`
- **Animation:** width transition 200ms ease

### Header
- Expanded: "CitaSpot" text (white, font-weight 700, 14px) + collapse button (‹‹)
- Collapsed: "C" letter (white, centered) + expand button (››)
- Bottom border: 1px #2c2c2e

### Navigation Items
- **Expanded:** Icon (lucide-react, 16px) + label (13px), gap 10px, padding 8px 12px
- **Collapsed:** Icon only, centered in 36x36px hit area
- **Active state:** background #2c2c2e, text white, font-weight 500
- **Inactive state:** text #8e8e93, hover: text #d1d1d6
- **Border radius:** 6px
- **Gap between items:** 1px

### Footer
- Logout button with top border separator (#2c2c2e)
- Same styling as nav items but muted (#636366)

### Mobile behavior
- On screens < 768px (md breakpoint): sidebar is hidden by default
- Hamburger button in top-left of content area opens sidebar as full-height overlay
- Backdrop overlay (#000 40% opacity) closes sidebar on tap
- No collapse mode on mobile — always full expanded or hidden

---

## 3. UI Component Updates

### Card
```
Before: rounded-xl border border-neutral-200 bg-white shadow-sm
After:  rounded-lg border border-neutral-200 bg-white
```
- Remove `shadow-sm` class
- Change `rounded-xl` → `rounded-lg` (16px → 8px)
- No other changes — Card remains a simple container

### Button
```
Before: bg-primary-600 (#7c3aed violet)
After:  bg-primary-800 (#3730a3 indigo deep)
```
- Primary variant: `bg-primary-800 hover:bg-primary-700`
- Secondary variant: unchanged (white bg, neutral border)
- Ghost variant: unchanged
- Border radius: `rounded-lg` (8px)

### Badge
- Border radius: keep `rounded-full` (pill shape) — this is a conscious contrast with the 8px cards
- Colors: update any primary-colored badges to use new indigo scale

### Input
- Border radius: `rounded-lg` (8px)
- Focus ring: `focus:ring-primary-800` (indigo deep)
- No other changes

### Tailwind Config Changes
```js
// tailwind.config.ts
borderRadius: {
  DEFAULT: '8px',    // was 6px
  sm: '4px',
  md: '8px',
  lg: '8px',         // was 12px — now same as default
  xl: '12px',        // was 16px
  '2xl': '16px',     // was 24px
  full: '9999px',
}

boxShadow: {
  sm: 'none',        // kill default shadow
  DEFAULT: 'none',
  md: '0 4px 6px -1px rgb(0 0 0 / 0.05)',  // only for dropdowns/modals
  lg: '0 10px 15px -3px rgb(0 0 0 / 0.08)', // only for modals
}
```

---

## 4. Settings Sub-Pages

The settings pages were recently restructured into sub-pages with a layout. Apply the new design:

### Settings Layout
- Desktop: vertical sidebar nav (left, 192px) + content area
- Mobile: horizontal scrollable nav at top
- Active nav item: `bg-primary-50 text-primary-800` (was primary-700)

### Individual pages
- Use updated Card component (no shadows, 8px radius)
- Buttons use `bg-primary-800`
- Inputs use `rounded-lg` and `focus:ring-primary-800`
- No content changes — only visual tokens

---

## 5. Files to Modify

| File | Change |
|------|--------|
| `apps/web/tailwind.config.ts` | New color palette, border-radius scale, shadow overrides |
| `apps/web/components/ui/card.tsx` | Remove shadow-sm, change rounded-xl → rounded-lg |
| `apps/web/components/ui/button.tsx` | Update primary variant colors |
| `apps/web/components/ui/input.tsx` | Update focus ring color |
| `apps/web/components/ui/badge.tsx` | Update any primary color references |
| `apps/web/components/dashboard/sidebar.tsx` | Full rewrite: dark theme, collapsible, localStorage |
| `apps/web/app/(dashboard)/layout.tsx` | Support sidebar collapse state, mobile hamburger |
| `apps/web/app/dashboard/layout.tsx` | Mirror changes from (dashboard)/layout.tsx |
| `apps/web/app/(dashboard)/settings/layout.tsx` | Update active state colors |

---

## 6. Verification

- All existing pages render correctly with new tokens (spot check dashboard, agenda, services)
- Sidebar collapses and expands with smooth animation
- Sidebar state persists across page navigations and refreshes
- Mobile: sidebar opens as overlay, closes on backdrop tap
- Settings sub-pages render with new design
- No visual regressions in badges, buttons, or form elements
- Primary color consistently Indigo Deep across all interactive elements
