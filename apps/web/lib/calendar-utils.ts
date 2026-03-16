// Utilidades puras del calendario de citas.
// Extraídas para permitir tests unitarios sin dependencias de Next.js.

import { getHours, getMinutes } from 'date-fns';

// ── Constantes del grid de tiempo ─────────────────────────────────────────────

export const START_HOUR  = 6;
export const END_HOUR    = 23;
export const HOUR_PX     = 64;   // píxeles por hora
export const TOTAL_HOURS = END_HOUR - START_HOUR;
export const GRID_PX     = TOTAL_HOURS * HOUR_PX;

// ── Helpers de fecha ──────────────────────────────────────────────────────────

/** Formatea una fecha como string YYYY-MM-DD en UTC. */
export function toDateStr(d: Date): string {
  return d.toISOString().split('T')[0];
}

/** Devuelve el saludo apropiado según la hora actual. */
export function getGreeting(): string {
  const h = new Date().getHours();
  if (h < 12) return 'Buenos días';
  if (h < 19) return 'Buenas tardes';
  return 'Buenas noches';
}

/** Offset en píxeles desde el inicio del grid para una fecha ISO. */
export function topPx(isoStr: string): number {
  const d    = new Date(isoStr);
  const mins = (getHours(d) - START_HOUR) * 60 + getMinutes(d);
  return mins * (HOUR_PX / 60);
}

/** Altura en píxeles para una duración en minutos (mínimo 22px). */
export function heightPx(durationMin: number): number {
  return Math.max(durationMin * (HOUR_PX / 60), 22);
}

// ── Algoritmo de columnas para citas solapadas ────────────────────────────────

export interface WithTimeRange {
  starts_at: string;
  ends_at:   string;
}

export interface WithColumns {
  col:  number;
  span: number;
}

/**
 * Asigna col/span a cada cita para renderizar solapamientos lado a lado.
 * Agrupa en clusters de solapamiento y reparte columnas dentro de cada cluster.
 */
export function assignColumns<T extends WithTimeRange>(
  appts: T[]
): Array<T & WithColumns> {
  if (!appts.length) return [];

  const sorted = [...appts].sort(
    (a, b) => new Date(a.starts_at).getTime() - new Date(b.starts_at).getTime()
  );

  const clusters: T[][] = [];
  let cluster: T[]      = [sorted[0]];
  let clusterEnd        = new Date(sorted[0].ends_at);

  for (let i = 1; i < sorted.length; i++) {
    const appt  = sorted[i];
    const start = new Date(appt.starts_at);
    if (start < clusterEnd) {
      cluster.push(appt);
      const end = new Date(appt.ends_at);
      if (end > clusterEnd) clusterEnd = end;
    } else {
      clusters.push(cluster);
      cluster    = [appt];
      clusterEnd = new Date(appt.ends_at);
    }
  }
  clusters.push(cluster);

  const result: Array<T & WithColumns> = [];
  clusters.forEach(c =>
    c.forEach((appt, idx) => result.push({ ...appt, col: idx, span: c.length }))
  );
  return result;
}
