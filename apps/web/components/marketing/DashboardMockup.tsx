'use client';

import { useTranslations } from '@/lib/i18n';
import {
  Calendar,
  MessageCircle,
  BarChart3,
  Users,
  Settings,
  Search,
  Bell,
} from 'lucide-react';

export function DashboardMockup() {
  const t = useTranslations();
  const l = t.landing;

  const appointments = [
    { time: '09:00', name: 'Carla Méndez', service: 'Limpieza facial', status: 'confirmed' },
    { time: '10:30', name: 'Andrés Pérez', service: 'Corte + barba', status: 'confirmed' },
    { time: '12:00', name: 'Lucía Vega', service: 'Manicure', status: 'pending' },
    { time: '14:00', name: 'Pedro Torres', service: 'Masaje 60min', status: 'confirmed' },
    { time: '16:00', name: 'María Soto', service: 'Tinte + corte', status: 'confirmed' },
  ] as const;

  return (
    <div className="overflow-hidden rounded-3xl border border-lp-border bg-white shadow-2xl shadow-primary-600/10">
      {/* Browser bar */}
      <div className="flex items-center gap-2 border-b border-lp-border bg-lp-bg-subtle px-5 py-3">
        <div className="flex gap-1.5">
          <span className="h-2.5 w-2.5 rounded-full bg-rose-400/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-amber-400/70" />
          <span className="h-2.5 w-2.5 rounded-full bg-emerald-400/70" />
        </div>
        <div className="ml-3 hidden flex-1 items-center gap-2 rounded-md border border-lp-border bg-white px-3 py-1 text-xs text-lp-ink-muted sm:flex">
          <span className="text-emerald-500">●</span> app.citaspot.com/dashboard
        </div>
      </div>

      <div className="grid grid-cols-[60px_1fr] md:grid-cols-[200px_1fr]">
        {/* Sidebar */}
        <aside className="border-r border-lp-border bg-neutral-50 p-3 md:p-4">
          <div className="mb-5 hidden font-serif text-lg font-semibold text-lp-ink md:block">CitaSpot</div>
          <nav className="flex flex-col gap-1">
            {[
              { icon: BarChart3, label: 'Inicio', active: false },
              { icon: Calendar, label: 'Agenda', active: true },
              { icon: MessageCircle, label: 'WhatsApp', active: false },
              { icon: Users, label: 'Clientes', active: false },
              { icon: Settings, label: 'Ajustes', active: false },
            ].map(({ icon: Icon, label, active }) => (
              <div
                key={label}
                className={[
                  'flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm',
                  active
                    ? 'bg-primary-50 font-semibold text-primary-700'
                    : 'text-lp-ink-soft',
                ].join(' ')}
              >
                <Icon size={16} />
                <span className="hidden md:inline">{label}</span>
              </div>
            ))}
          </nav>
        </aside>

        {/* Main */}
        <main className="bg-white p-4 md:p-6">
          {/* Header */}
          <div className="mb-5 flex items-center justify-between">
            <div>
              <div className="text-[11px] font-semibold uppercase tracking-wider text-lp-ink-muted">
                {l.dashboardTodayLabel}
              </div>
              <div className="font-serif text-xl font-semibold text-lp-ink md:text-2xl">
                Martes, 12 de marzo
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button className="hidden h-8 w-8 items-center justify-center rounded-full text-lp-ink-soft sm:flex">
                <Search size={16} />
              </button>
              <button className="relative flex h-8 w-8 items-center justify-center rounded-full text-lp-ink-soft">
                <Bell size={16} />
                <span className="absolute -right-0.5 -top-0.5 h-2 w-2 rounded-full bg-rose-500" />
              </button>
              <div className="h-8 w-8 rounded-full bg-gradient-to-br from-primary-400 to-primary-700" />
            </div>
          </div>

          {/* Stats */}
          <div className="mb-5 grid grid-cols-3 gap-2 md:gap-3">
            <StatCard label={l.dashboardAppointmentsLabel} value="12" trend="+3" />
            <StatCard label={l.dashboardOccupancyLabel} value="86%" trend="+8%" />
            <StatCard label={l.dashboardConfirmedLabel} value="10/12" trend="" />
          </div>

          {/* Appointments + chat */}
          <div className="grid gap-3 md:grid-cols-[1.4fr_1fr]">
            {/* Appointments list */}
            <div className="rounded-xl border border-lp-border">
              <div className="border-b border-lp-border px-4 py-2.5 text-sm font-semibold text-lp-ink">
                Próximas citas
              </div>
              <ul className="divide-y divide-lp-border">
                {appointments.map((a) => (
                  <li key={a.time} className="flex items-center gap-3 px-4 py-2.5 text-sm">
                    <div className="w-12 font-semibold text-lp-ink">{a.time}</div>
                    <div className="min-w-0 flex-1">
                      <div className="truncate font-medium text-lp-ink">{a.name}</div>
                      <div className="truncate text-xs text-lp-ink-muted">{a.service}</div>
                    </div>
                    <span
                      className={[
                        'rounded-full px-2 py-0.5 text-[10px] font-semibold',
                        a.status === 'confirmed'
                          ? 'bg-emerald-50 text-emerald-700'
                          : 'bg-amber-50 text-amber-700',
                      ].join(' ')}
                    >
                      {a.status === 'confirmed' ? l.dashboardConfirmedLabel : l.dashboardPendingLabel}
                    </span>
                  </li>
                ))}
              </ul>
            </div>

            {/* WhatsApp panel */}
            <div className="hidden rounded-xl border border-lp-border md:block">
              <div className="border-b border-lp-border px-4 py-2.5 text-sm font-semibold text-lp-ink">
                WhatsApp · en vivo
              </div>
              <div className="space-y-2 p-3">
                <div className="rounded-xl rounded-bl-md bg-lp-bg-subtle px-3 py-2 text-xs text-lp-ink">
                  Hola, ¿tienen turno hoy?
                </div>
                <div className="rounded-xl rounded-br-md bg-primary-50 px-3 py-2 text-xs text-primary-900">
                  ¡Hola! Tengo 14:00 o 16:00. ¿Cuál prefieres?
                </div>
                <div className="rounded-xl rounded-bl-md bg-lp-bg-subtle px-3 py-2 text-xs text-lp-ink">
                  14:00 está bien
                </div>
                <div className="flex items-center gap-1.5 px-1 pt-1 text-[10px] text-lp-ink-muted">
                  <span className="h-1.5 w-1.5 rounded-full bg-success animate-pulse" />
                  Sarai está escribiendo…
                </div>
              </div>
            </div>
          </div>
        </main>
      </div>
    </div>
  );
}

function StatCard({ label, value, trend }: { label: string; value: string; trend: string }) {
  return (
    <div className="rounded-xl border border-lp-border bg-lp-bg-subtle p-3 md:p-4">
      <div className="text-[10px] font-semibold uppercase tracking-wider text-lp-ink-muted md:text-[11px]">
        {label}
      </div>
      <div className="mt-1.5 font-serif text-2xl font-semibold text-lp-ink md:text-3xl">
        {value}
      </div>
      {trend && (
        <div className="mt-1 text-[10px] font-semibold text-emerald-600 md:text-xs">{trend}</div>
      )}
    </div>
  );
}
