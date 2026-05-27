'use client';

// Campana de notificaciones del sistema (Plane #41).
// - Muestra badge con el contador de no leídas (clampa a "9+").
// - Click abre dropdown con la lista paginada.
// - Click en una fila marca como leída (optimista) y navega a la sección
//   relevante (agenda para citas, WhatsApp para mensajes entrantes).
//
// El componente NO se suscribe al SSE — eso lo hace el dashboard, que llama a
// `prepend()` del mismo hook cuando llega `notification.created`. Por eso este
// componente recibe el resultado de `useNotifications()` como prop, garantizando
// que comparten la misma instancia de estado.

import { useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  Bell,
  CalendarPlus,
  CalendarX,
  CalendarClock,
  MessageCircle,
} from 'lucide-react';
import { useTranslations } from '@/lib/i18n';
import type { UserNotification, UserNotificationType } from '@/lib/api';
import type { UseNotificationsResult } from '@/hooks/useNotifications';
import { cn } from '@/lib/utils';

// ── Helpers ───────────────────────────────────────────────────────────────────

function formatRelative(
  iso: string,
  t: ReturnType<typeof useTranslations>,
): string {
  const created = new Date(iso).getTime();
  if (Number.isNaN(created)) return '';
  const diffMs = Date.now() - created;
  const minutes = Math.floor(diffMs / 60_000);
  if (minutes < 1) return t.notifications.justNow;
  if (minutes < 60) return t.notifications.minutesAgo.replace('{n}', String(minutes));
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return t.notifications.hoursAgo.replace('{n}', String(hours));
  const days = Math.floor(hours / 24);
  return t.notifications.daysAgo.replace('{n}', String(days));
}

function IconForType({ type }: { type: UserNotificationType }) {
  const base = 'h-4 w-4';
  switch (type) {
    case 'appointment.created':
      return <CalendarPlus className={cn(base, 'text-emerald-600')} />;
    case 'appointment.cancelled':
      return <CalendarX className={cn(base, 'text-red-600')} />;
    case 'appointment.rescheduled':
      return <CalendarClock className={cn(base, 'text-amber-600')} />;
    case 'whatsapp.inbound':
      return <MessageCircle className={cn(base, 'text-primary-600')} />;
    default:
      return <Bell className={cn(base, 'text-neutral-500')} />;
  }
}

/**
 * Retorna el href para deep-link en el dashboard.
 *
 * Rutas disponibles verificadas:
 * - /dashboard/agenda          → para citas (no existe /dashboard/appointments/[id])
 * - /dashboard/whatsapp        → para mensajes entrantes (no existe /dashboard/conversations/[id])
 *
 * Si en el futuro existieran rutas más específicas (e.g. detalle de cita),
 * solo hay que ampliar este switch.
 */
function deepLinkFor(n: UserNotification): string | null {
  switch (n.type) {
    case 'appointment.created':
    case 'appointment.cancelled':
    case 'appointment.rescheduled':
      return '/dashboard/agenda';
    case 'whatsapp.inbound':
      return '/dashboard/whatsapp';
    default:
      return null;
  }
}

// ── Componente ────────────────────────────────────────────────────────────────

export interface NotificationBellProps {
  notifications: UseNotificationsResult;
}

export function NotificationBell({ notifications }: NotificationBellProps) {
  const t = useTranslations();
  const router = useRouter();
  const containerRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);

  const {
    notifications: items,
    unreadCount,
    loading,
    hasMore,
    forbidden,
    loadMore,
    markRead,
    markAllRead,
  } = notifications;

  // Cerrar al clickear fuera o presionar Escape.
  useEffect(() => {
    if (!open) return;
    const onClickOutside = (e: MouseEvent) => {
      if (!containerRef.current?.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('mousedown', onClickOutside);
    document.addEventListener('keydown', onKey);
    return () => {
      document.removeEventListener('mousedown', onClickOutside);
      document.removeEventListener('keydown', onKey);
    };
  }, [open]);

  // Modo "oculto" — usuario sin permisos (staff).
  if (forbidden) return null;

  const handleRowClick = async (n: UserNotification) => {
    setOpen(false);
    try {
      if (n.read_at === null) await markRead(n.id);
    } catch {
      // markRead ya hace rollback — silenciamos para no interrumpir la navegación.
    }
    const href = deepLinkFor(n);
    if (href) router.push(href);
  };

  const handleMarkAll = async () => {
    try {
      await markAllRead();
    } catch {
      // El optimismo del hook ya hizo rollback si falló.
    }
  };

  const badgeText = unreadCount > 9 ? '9+' : String(unreadCount);

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-label={t.notifications.title}
        aria-expanded={open}
        aria-haspopup="true"
        className="relative flex h-9 w-9 items-center justify-center rounded-lg bg-white/10 text-white transition-colors hover:bg-white/20"
      >
        <Bell className="h-4 w-4" />
        {unreadCount > 0 && (
          <span
            aria-label={t.notifications.unreadBadge.replace('{count}', String(unreadCount))}
            className="absolute -right-1 -top-1 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-accent-500 px-1 text-[10px] font-semibold leading-none text-white shadow-sm"
          >
            {badgeText}
          </span>
        )}
      </button>

      {open && (
        <div
          role="dialog"
          aria-label={t.notifications.title}
          className="animate-fade-in absolute right-0 top-full z-50 mt-2 w-[min(92vw,360px)] overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-lg"
        >
          {/* Header */}
          <div className="flex items-center justify-between border-b border-neutral-200 px-4 py-3">
            <h3 className="text-sm font-semibold text-neutral-900">
              {t.notifications.title}
            </h3>
            {unreadCount > 0 && (
              <button
                type="button"
                onClick={handleMarkAll}
                className="text-xs font-medium text-primary-600 transition-colors hover:text-primary-700"
              >
                {t.notifications.markAllRead}
              </button>
            )}
          </div>

          {/* Body */}
          <div className="max-h-[60vh] overflow-y-auto">
            {loading && items.length === 0 ? (
              <div className="space-y-2 p-3">
                {[0, 1, 2].map((i) => (
                  <div
                    key={i}
                    className="h-14 animate-pulse rounded-lg bg-neutral-100"
                  />
                ))}
              </div>
            ) : items.length === 0 ? (
              <div className="px-4 py-8 text-center text-sm text-neutral-500">
                {t.notifications.empty}
              </div>
            ) : (
              <ul className="divide-y divide-neutral-100">
                {items.map((n) => {
                  const isUnread = n.read_at === null;
                  return (
                    <li key={n.id}>
                      <button
                        type="button"
                        onClick={() => handleRowClick(n)}
                        className={cn(
                          'flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-neutral-50',
                          isUnread && 'bg-primary-50/40',
                        )}
                      >
                        <span className="mt-0.5 flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-full bg-neutral-100">
                          <IconForType type={n.type} />
                        </span>
                        <div className="min-w-0 flex-1">
                          <div className="flex items-start justify-between gap-2">
                            <p
                              className={cn(
                                'truncate text-sm leading-tight',
                                isUnread
                                  ? 'font-semibold text-neutral-900'
                                  : 'font-medium text-neutral-700',
                              )}
                            >
                              {n.title}
                            </p>
                            {isUnread && (
                              <span
                                aria-hidden
                                className="mt-1 h-2 w-2 flex-shrink-0 rounded-full bg-primary-500"
                              />
                            )}
                          </div>
                          {n.body && (
                            <p className="mt-0.5 line-clamp-2 text-xs text-neutral-500">
                              {n.body}
                            </p>
                          )}
                          <p className="mt-1 text-[11px] text-neutral-400">
                            {formatRelative(n.created_at, t)}
                          </p>
                        </div>
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>

          {/* Footer */}
          {hasMore && items.length > 0 && (
            <div className="border-t border-neutral-200 px-4 py-2">
              <button
                type="button"
                onClick={() => void loadMore()}
                className="w-full rounded-lg py-2 text-center text-xs font-medium text-primary-600 transition-colors hover:bg-primary-50"
              >
                {t.notifications.loadMore}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default NotificationBell;
