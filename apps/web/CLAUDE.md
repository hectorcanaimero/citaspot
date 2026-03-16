# CLAUDE.md — apps/web (Next.js Dashboard + Booking)
> Reglas específicas para el frontend. Lee también el CLAUDE.md raíz.

---

## 🎯 RESPONSABILIDADES

- Dashboard del negocio (`/dashboard/*`) — panel de administración
- Página pública de reservas (`/book/[slug]`) — para clientes finales
- Flujo de onboarding (`/onboarding/*`) — setup inicial del negocio
- Autenticación (`/login`, `/register`)

---

## 📁 ESTRUCTURA OBLIGATORIA

```
apps/web/
├── CLAUDE.md
├── app/
│   ├── (auth)/                  ← Grupo sin layout de dashboard
│   │   ├── login/page.tsx
│   │   └── register/page.tsx
│   │
│   ├── (dashboard)/             ← Rutas protegidas — requieren auth
│   │   ├── layout.tsx           ← Sidebar + Header + auth guard
│   │   ├── page.tsx             ← Overview / home del dashboard
│   │   ├── appointments/
│   │   │   ├── page.tsx         ← Vista de calendario de citas
│   │   │   └── [id]/page.tsx   ← Detalle de una cita
│   │   ├── clients/
│   │   │   ├── page.tsx         ← Lista de clientes
│   │   │   └── [id]/page.tsx   ← Perfil del cliente
│   │   ├── services/
│   │   │   └── page.tsx         ← Gestión de servicios y precios
│   │   ├── team/
│   │   │   └── page.tsx         ← Gestión de profesionales y horarios
│   │   ├── knowledge/
│   │   │   └── page.tsx         ← Editor de base de conocimiento IA
│   │   ├── whatsapp/
│   │   │   └── page.tsx         ← Conectar/ver estado del número WA
│   │   ├── analytics/
│   │   │   └── page.tsx         ← KPIs: ocupación, no-shows, etc.
│   │   └── settings/
│   │       └── page.tsx         ← Configuración general del negocio
│   │
│   ├── (booking)/               ← Rutas públicas — sin auth
│   │   └── book/
│   │       └── [slug]/
│   │           ├── page.tsx     ← Página de reserva del negocio
│   │           └── confirmation/page.tsx
│   │
│   ├── (onboarding)/            ← Flujo de setup inicial
│   │   └── onboarding/
│   │       ├── layout.tsx       ← Progress bar + pasos
│   │       └── steps/
│   │           ├── profile/page.tsx
│   │           ├── services/page.tsx
│   │           ├── schedule/page.tsx
│   │           ├── whatsapp/page.tsx
│   │           ├── knowledge/page.tsx
│   │           └── done/page.tsx
│   │
│   └── api/                     ← API routes de Next.js
│       ├── auth/[...supabase]/route.ts
│       └── webhooks/stripe/route.ts
│
├── components/
│   ├── ui/                      ← shadcn/ui components (no modificar)
│   ├── dashboard/               ← Componentes específicos del dashboard
│   │   ├── AppointmentCalendar.tsx
│   │   ├── AppointmentCard.tsx
│   │   ├── StatsCard.tsx
│   │   ├── Sidebar.tsx
│   │   └── Header.tsx
│   ├── booking/                 ← Componentes de la página pública
│   │   ├── ServiceSelector.tsx
│   │   ├── ProfessionalSelector.tsx
│   │   ├── SlotPicker.tsx
│   │   └── BookingForm.tsx
│   └── shared/                  ← Componentes reutilizables
│       ├── LoadingSpinner.tsx
│       ├── ErrorBoundary.tsx
│       └── EmptyState.tsx
│
├── lib/
│   ├── api.ts                   ← Cliente HTTP para Core API (React Query)
│   ├── supabase/
│   │   ├── client.ts            ← Supabase browser client
│   │   └── server.ts            ← Supabase server client
│   └── utils.ts                 ← Utilidades generales
│
├── hooks/                       ← Custom React hooks
│   ├── useAppointments.ts       ← React Query hooks
│   ├── useProfessionals.ts
│   └── useWhatsAppStatus.ts
│
├── store/                       ← Zustand stores (estado global)
│   └── onboarding.ts            ← Estado del flujo de onboarding
│
├── types/                       ← TypeScript types (importar de shared-types)
│   └── index.ts
│
├── middleware.ts                 ← Protección de rutas
├── next.config.ts
├── tailwind.config.ts
└── package.json
```

---

## 🔧 STACK Y CONVENCIONES

### Framework y herramientas
```
Next.js 14      → App Router (NUNCA usar Pages Router)
TypeScript      → strict mode, NUNCA usar 'any'
Tailwind CSS    → utilidades core solamente
shadcn/ui       → componentes UI base (no instalar otras libs sin ok)
React Query v5  → data fetching (NUNCA useEffect para fetch)
Zustand         → estado global (NUNCA Redux, NUNCA Context para estado complejo)
React Hook Form → formularios
Zod             → validación de schemas
```

### TypeScript estricto
```typescript
// ✅ Correcto
interface Appointment {
  id: string;
  startsAt: Date;
  status: 'pending' | 'confirmed' | 'cancelled' | 'completed' | 'no_show';
}

// ❌ Incorrecto — any destruye el type safety
const appointment: any = fetchData();
```

### Naming conventions
```
Archivos de páginas:    page.tsx, layout.tsx (Next.js convention)
Componentes:            PascalCase.tsx → AppointmentCard.tsx
Hooks:                  camelCase.ts → useAppointments.ts
Utilidades:             camelCase.ts → formatDate.ts
```

---

## ⚡ DATA FETCHING — React Query

```typescript
// ✅ CORRECTO — usar React Query
// hooks/useAppointments.ts
export function useAppointments(date: Date) {
  return useQuery({
    queryKey: ['appointments', format(date, 'yyyy-MM-dd')],
    queryFn: () => api.getAppointments({ date: format(date, 'yyyy-MM-dd') }),
    staleTime: 30_000,  // 30 segundos
  });
}

// Uso en componente
function AppointmentCalendar() {
  const { data, isLoading, error } = useAppointments(selectedDate);
  if (isLoading) return <LoadingSpinner />;
  if (error) return <ErrorMessage error={error} />;
  return <Calendar appointments={data} />;
}

// ❌ INCORRECTO — useEffect para fetch
function AppointmentCalendar() {
  const [data, setData] = useState([]);
  useEffect(() => {
    fetch('/api/appointments').then(r => r.json()).then(setData);
  }, []);
}
```

---

## 🔐 AUTENTICACIÓN (Supabase Auth)

```typescript
// middleware.ts — proteger rutas del dashboard
import { createServerClient } from '@supabase/ssr';
import { NextResponse } from 'next/server';

export async function middleware(request: NextRequest) {
  // Verificar sesión en rutas del dashboard
  if (request.nextUrl.pathname.startsWith('/dashboard')) {
    const { session } = await getSession(request);
    if (!session) {
      return NextResponse.redirect(new URL('/login', request.url));
    }
  }
  return NextResponse.next();
}

export const config = {
  matcher: ['/dashboard/:path*', '/onboarding/:path*'],
};
```

---

## 📱 DISEÑO — Mobile First

```
El dueño del negocio usa el dashboard desde su celular.
SIEMPRE diseñar mobile-first, luego adaptar a desktop.

Breakpoints:
sm: 640px   → Teléfonos grandes
md: 768px   → Tablets
lg: 1024px  → Desktop

Reglas de diseño:
✅ Touch targets mínimo 44x44px
✅ Texto mínimo 16px para evitar zoom en iOS
✅ Formularios con labels visibles (no solo placeholders)
✅ Loading states en todas las acciones asíncronas
✅ Empty states con mensajes y CTA claros
✅ Manejo de errores visible (no solo console.error)
```

---

## 🌐 PÁGINA PÚBLICA DE RESERVAS (`/book/[slug]`)

Esta página es para los clientes finales del negocio.
Requisitos de rendimiento:

```
✅ Server-side rendering (SSR) — el SEO importa
✅ Optimistic UI en selección de slot
✅ Sin autenticación requerida
✅ Accesible (WCAG AA mínimo)
✅ Carga < 3 segundos en 3G lento
```

---

## 🎯 ONBOARDING — REGLAS CRÍTICAS

```
El onboarding es el flujo más importante para la retención.
Meta: cualquier usuario completa el setup en < 15 minutos.

Reglas:
✅ Estado del progreso guardado en Zustand (persistido en localStorage)
✅ Si cierra el browser, puede retomar desde donde quedó
✅ Validación en tiempo real (no solo al enviar)
✅ Feedback positivo en cada paso completado
✅ El paso de WhatsApp muestra QR code con polling de estado cada 5 segundos
✅ Skip permitido en pasos opcionales (knowledge base)
```

---

## 🔄 CLIENTE HTTP (lib/api.ts)

```typescript
// lib/api.ts — todas las llamadas al Core API van aquí

const apiClient = {
  async getAppointments(params: GetAppointmentsParams): Promise<Appointment[]> {
    const res = await fetch(`${API_URL}/api/v1/appointments?${new URLSearchParams(params)}`, {
      headers: await getAuthHeaders(),
    });
    if (!res.ok) throw new APIError(res.status, await res.json());
    return res.json();
  },
  
  // ... un método por endpoint
};

// NUNCA hacer fetch() directamente en componentes o páginas
// SIEMPRE ir a través de este cliente
```

---

## 🌐 INTERNACIONALIZACIÓN

Sistema i18n propio en `lib/i18n/` — sin librerías externas.

```
Idiomas soportados: es (default), en, pt
Persistencia: localStorage key 'citaspot_language'
Detección automática: localStorage → navigator.language → 'es'

Hooks disponibles:
  useTranslations()  → objeto de traducciones del idioma activo
  useLanguage()      → { language, setLanguage }
  useDateLocale()    → locale de date-fns (es / enUS / ptBR)

Reglas:
✅ SIEMPRE usar useTranslations() — nunca hardcodear strings en español
✅ Strings parametrizadas con .replace('{key}', value) en el punto de uso
✅ Fechas: usar useDateLocale() y pasarlo como { locale } a date-fns
✅ Selector de idioma en Settings → pestaña Cuenta
❌ NUNCA importar { es } from 'date-fns/locale' directamente

Formato de fechas: DD/MM/YYYY (no MM/DD/YYYY)
Formato de hora: 12h con AM/PM
Moneda: USD (mostrar como "$15.00 USD")
```

---

## 📦 DEPENDENCIAS APROBADAS

```json
{
  "dependencies": {
    "next": "14.x",
    "@supabase/ssr": "latest",
    "@supabase/supabase-js": "latest",
    "@tanstack/react-query": "5.x",
    "zustand": "4.x",
    "react-hook-form": "7.x",
    "zod": "3.x",
    "date-fns": "3.x",
    "clsx": "latest",
    "tailwind-merge": "latest",
    "lucide-react": "latest"
  }
}
```

> ⚠️ Para agregar una nueva dependencia, justificarla en el PR.
> No agregar librerías de UI adicionales sin aprobación.
