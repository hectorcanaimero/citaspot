'use client';

// Widget de próximas citas del dashboard.
// Muestra hasta 10 citas futuras (pending/confirmed) ordenadas por starts_at ASC.
// Se refetcha cuando `refreshKey` cambia — el padre lo incrementa al recibir
// eventos SSE de creación/actualización/reagendamiento.

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { CalendarClock, ArrowUpRight, User } from 'lucide-react';
import { formatInTimeZone } from 'date-fns-tz';
import { appointments, Appointment, APIError } from '@/lib/api';
import { useTranslations, useDateLocale } from '@/lib/i18n';
import { useTenantTimezone } from '@/store/tenant';

interface UpcomingAppointmentsCardProps {
  /** Cambiar este valor fuerza un refetch (incrementar al recibir un evento). */
  refreshKey?: number;
}

const STATUS_DOT: Partial<Record<Appointment['status'], string>> = {
  pending: 'bg-amber-400',
  confirmed: 'bg-primary-500',
};

function RowSkeleton() {
  return <div className="h-[64px] animate-pulse rounded-xl border border-neutral-200 bg-white" />;
}

/** Devuelve YYYY-MM-DD para `date` en la zona horaria indicada (sin formateo de idioma). */
function dateKeyInTz(date: Date, timezone: string): string {
  return formatInTimeZone(date, timezone, 'yyyy-MM-dd');
}

export function UpcomingAppointmentsCard({ refreshKey = 0 }: UpcomingAppointmentsCardProps) {
  const t = useTranslations();
  const dateLocale = useDateLocale();
  const tz = useTenantTimezone();

  const [items, setItems] = useState<Appointment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    appointments
      .upcoming(10)
      .then((res) => {
        if (!cancelled) setItems(res.data ?? []);
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err instanceof APIError ? err.message : t.dashboard.errorLoadingAppts);
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [refreshKey, t]);

  // Etiqueta de fecha: Hoy / Mañana / dd MMM (en zona del tenant)
  function dateBadge(startsAt: string): string {
    const now = new Date();
    const apptDate = new Date(startsAt);
    const todayKey = dateKeyInTz(now, tz);
    const tomorrowKey = dateKeyInTz(new Date(now.getTime() + 24 * 60 * 60 * 1000), tz);
    const apptKey = dateKeyInTz(apptDate, tz);

    if (apptKey === todayKey) return t.common.today;
    if (apptKey === tomorrowKey) return t.common.tomorrow;
    return formatInTimeZone(apptDate, tz, 'd MMM', { locale: dateLocale });
  }

  return (
    <div
      aria-live="polite"
      className="rounded-xl border border-neutral-200 bg-white p-4 shadow-sm"
    >
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-sm font-semibold text-neutral-900">
          {t.dashboard.upcomingAppointments}
        </h3>
        <CalendarClock className="h-4 w-4 text-neutral-400" />
      </div>

      {loading ? (
        <div className="space-y-2">
          {[...Array(3)].map((_, i) => (
            <RowSkeleton key={i} />
          ))}
        </div>
      ) : error ? (
        <p className="py-4 text-center text-xs text-red-600">{error}</p>
      ) : items.length === 0 ? (
        <div className="flex flex-col items-center rounded-xl bg-gradient-to-br from-white to-primary-50/30 px-4 py-8 text-center">
          <div className="mb-2 flex h-10 w-10 items-center justify-center rounded-xl bg-primary-50">
            <CalendarClock className="h-5 w-5 text-primary-500" />
          </div>
          <p className="text-xs text-neutral-500">{t.dashboard.upcomingEmpty}</p>
          <Link
            href="/dashboard/agenda"
            className="mt-3 inline-flex items-center gap-1.5 rounded-lg bg-primary-600 px-3 py-1.5 text-xs font-semibold text-white transition-colors hover:bg-primary-500"
          >
            {t.dashboard.upcomingEmptyCTA}
            <ArrowUpRight className="h-3 w-3" />
          </Link>
        </div>
      ) : (
        <ul role="list" className="space-y-2">
          {items.map((appt) => {
            const dot = STATUS_DOT[appt.status] ?? 'bg-neutral-300';
            return (
              <li
                role="listitem"
                key={appt.id}
                className="flex items-center gap-3 rounded-xl border border-neutral-100 bg-white px-3 py-2.5 transition-colors hover:border-primary-200"
              >
                <div className="w-14 flex-shrink-0 text-center">
                  <p className="text-xs font-semibold text-primary-700">
                    {dateBadge(appt.starts_at)}
                  </p>
                  <p className="text-xs text-neutral-500">
                    {formatInTimeZone(appt.starts_at, tz, 'HH:mm')}
                  </p>
                </div>
                <div className="h-9 w-1 flex-shrink-0 rounded-full bg-primary-300/70" />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-semibold text-neutral-900">
                    {appt.customer_name}
                  </p>
                  <div className="mt-0.5 flex items-center gap-2 text-xs text-neutral-500">
                    <span className="flex items-center gap-1 truncate">
                      <User className="h-3 w-3" />
                      {appt.professional_name}
                    </span>
                    <span>·</span>
                    <span className="truncate">{appt.service_name}</span>
                  </div>
                </div>
                <span
                  className={`h-2 w-2 flex-shrink-0 rounded-full ${dot}`}
                  aria-label={appt.status}
                />
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

export default UpcomingAppointmentsCard;
