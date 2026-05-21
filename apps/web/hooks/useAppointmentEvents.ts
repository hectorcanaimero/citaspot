'use client';

// Hook para consumir el stream SSE de eventos de citas en tiempo real.
// El backend expone GET /api/v1/realtime/appointments?token=<jwt> y emite
// eventos nombrados (`appointment.created`, `appointment.updated`,
// `appointment.cancelled`, `appointment.rescheduled`) en formato:
//   event: <type>
//   data: { event, data: Appointment, ts }
//
// Estrategia de reconexión:
//   - Backoff exponencial con jitter (1s → 30s).
//   - Si detectamos 3 errores consecutivos dentro de 2s probamos
//     refreshSession() una vez antes de seguir con el backoff (heurística
//     para 401 mid-stream, EventSource no expone status codes).
//   - Cleanup en unmount y en `pagehide` (BFCache safety).

import { useCallback, useEffect, useRef, useState } from 'react';
import { Appointment, getSupabaseAccessToken, realtimeAppointmentsUrl } from '@/lib/api';
import { createClient } from '@/lib/supabase/browser';

export type AppointmentEventType =
  | 'appointment.created'
  | 'appointment.updated'
  | 'appointment.cancelled'
  | 'appointment.rescheduled';

export interface AppointmentEventEnvelope {
  event: AppointmentEventType;
  data: Appointment;
  ts: string;
}

interface UseAppointmentEventsResult {
  connected: boolean;
}

const EVENT_TYPES: AppointmentEventType[] = [
  'appointment.created',
  'appointment.updated',
  'appointment.cancelled',
  'appointment.rescheduled',
];

const BACKOFF_BASE_MS = 1_000;
const BACKOFF_MAX_MS = 30_000;
const BACKOFF_FACTOR = 2;
const JITTER_RATIO = 0.2;
// Si recibimos `RAPID_ERROR_THRESHOLD` errores dentro de RAPID_ERROR_WINDOW_MS,
// asumimos token expirado y disparamos refreshSession().
const RAPID_ERROR_THRESHOLD = 3;
const RAPID_ERROR_WINDOW_MS = 2_000;

function computeBackoff(attempt: number): number {
  const exp = Math.min(BACKOFF_BASE_MS * BACKOFF_FACTOR ** attempt, BACKOFF_MAX_MS);
  const jitter = exp * JITTER_RATIO * (Math.random() * 2 - 1);
  return Math.max(BACKOFF_BASE_MS, Math.round(exp + jitter));
}

export function useAppointmentEvents(
  onEvent: (envelope: AppointmentEventEnvelope) => void,
): UseAppointmentEventsResult {
  const [connected, setConnected] = useState(false);

  // Mantenemos onEvent en una ref para evitar reabrir el EventSource cada vez
  // que el consumidor pasa una nueva función inline.
  const onEventRef = useRef(onEvent);
  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);

  // Refs internas — sobreviven a re-renders sin disparar effects.
  const esRef = useRef<EventSource | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const attemptRef = useRef(0);
  const recentErrorsRef = useRef<number[]>([]);
  const refreshTriedRef = useRef(false);
  const cancelledRef = useRef(false);

  const cleanup = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    }
    if (esRef.current) {
      esRef.current.close();
      esRef.current = null;
    }
  }, []);

  useEffect(() => {
    cancelledRef.current = false;

    async function open() {
      if (cancelledRef.current) return;

      const token = await getSupabaseAccessToken();
      if (!token) {
        setConnected(false);
        return;
      }
      if (cancelledRef.current) return;

      const url = realtimeAppointmentsUrl(token);
      const es = new EventSource(url);
      esRef.current = es;

      es.onopen = () => {
        setConnected(true);
        attemptRef.current = 0;
        recentErrorsRef.current = [];
        refreshTriedRef.current = false;
      };

      EVENT_TYPES.forEach((type) => {
        es.addEventListener(type, (raw: Event) => {
          const ev = raw as MessageEvent;
          try {
            const parsed = JSON.parse(ev.data) as {
              event?: AppointmentEventType;
              data: Appointment;
              ts: string;
            };
            onEventRef.current({
              event: parsed.event ?? type,
              data: parsed.data,
              ts: parsed.ts,
            });
          } catch (err) {
            // Mensaje malformado — ignorar pero loguear para diagnóstico.
            console.error('[useAppointmentEvents] failed to parse event', type, err);
          }
        });
      });

      es.onerror = async () => {
        setConnected(false);
        if (cancelledRef.current) return;
        // Cerrar el ES actual (siempre — el navegador podría intentar reconectar
        // automáticamente y queremos controlar la cadencia nosotros).
        es.close();
        esRef.current = null;

        // Trackear errores recientes para detectar token expirado.
        const now = Date.now();
        recentErrorsRef.current.push(now);
        recentErrorsRef.current = recentErrorsRef.current.filter(
          (t) => now - t <= RAPID_ERROR_WINDOW_MS,
        );

        const burstDetected =
          recentErrorsRef.current.length >= RAPID_ERROR_THRESHOLD && !refreshTriedRef.current;

        if (burstDetected) {
          refreshTriedRef.current = true;
          try {
            const supabase = createClient();
            const { data, error } = await supabase.auth.refreshSession();
            if (!error && data.session?.access_token) {
              // Reset backoff y reabrir inmediatamente con token fresco.
              attemptRef.current = 0;
              recentErrorsRef.current = [];
              if (!cancelledRef.current) void open();
              return;
            }
          } catch {
            // refreshSession falló — dejar que el flujo de backoff continúe.
          }
          // Si llegamos acá, no hay sesión recuperable — emitimos `auth_failed`
          // como log y dejamos de intentar.
          console.warn('[useAppointmentEvents] auth_failed: refreshSession did not recover');
          return;
        }

        // Backoff exponencial con jitter.
        const delay = computeBackoff(attemptRef.current);
        attemptRef.current += 1;
        reconnectTimerRef.current = setTimeout(() => {
          if (!cancelledRef.current) void open();
        }, delay);
      };
    }

    void open();

    // BFCache safety: cerrar el EventSource cuando la página se oculta.
    const handlePageHide = () => {
      cancelledRef.current = true;
      cleanup();
    };
    window.addEventListener('pagehide', handlePageHide);

    return () => {
      cancelledRef.current = true;
      window.removeEventListener('pagehide', handlePageHide);
      cleanup();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cleanup]);

  return { connected };
}

export default useAppointmentEvents;
