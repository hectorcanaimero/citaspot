// Utilidades puras del calendario de citas.
// Extraídas para permitir tests unitarios sin dependencias de Next.js.

import { getHours, getMinutes } from 'date-fns';
import { fromZonedTime, toZonedTime } from 'date-fns-tz';

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

/**
 * Offset en píxeles desde el inicio del grid para una fecha ISO,
 * calculado en el timezone del tenant (NO en el del browser).
 */
export function topPx(isoStr: string, tz: string): number {
  const d    = toZonedTime(isoStr, tz);
  const mins = (getHours(d) - START_HOUR) * 60 + getMinutes(d);
  return mins * (HOUR_PX / 60);
}

/** Hora local del tenant en píxeles desde START_HOUR (para la línea "ahora"). */
export function nowPx(now: Date, tz: string): number {
  const d    = toZonedTime(now, tz);
  const mins = (getHours(d) - START_HOUR) * 60 + getMinutes(d);
  return mins * (HOUR_PX / 60);
}

/** Altura en píxeles para una duración en minutos (mínimo 22px). */
export function heightPx(durationMin: number): number {
  return Math.max(durationMin * (HOUR_PX / 60), 22);
}

// ── Inverso de topPx: convierte coordenada Y del grid + fecha local → ISO UTC ─

/** Snap de minutos al múltiplo más cercano. */
export function snapMinutes(mins: number, step = 15): number {
  return Math.round(mins / step) * step;
}

/**
 * Dado un Y en píxeles dentro del grid (relativo a START_HOUR) y una fecha del día
 * (en hora local del navegador, sin importar la hora — solo cuenta y/m/d), devuelve
 * el ISO UTC que corresponde a esa Y interpretada en el timezone del tenant.
 *
 * snap = paso de redondeo en minutos (default 15).
 * minMinutes / maxMinutes = clamp dentro del grid (0..TOTAL_HOURS*60).
 */
export function gridYToIsoUTC(
  date: Date,
  y: number,
  tz: string,
  snap = 15,
): string {
  const totalGridMinutes = TOTAL_HOURS * 60;
  const rawMinutes       = (y / HOUR_PX) * 60;
  let mins               = snapMinutes(rawMinutes, snap);
  if (mins < 0) mins = 0;
  if (mins > totalGridMinutes - snap) mins = totalGridMinutes - snap;

  const totalFromStart = START_HOUR * 60 + mins;
  const hh = Math.floor(totalFromStart / 60);
  const mm = totalFromStart % 60;

  // Componemos el string local "YYYY-MM-DDTHH:MM:00" en el tz del tenant.
  // Usamos los componentes locales del Date para evitar saltos por DST.
  const y4 = date.getFullYear();
  const m2 = String(date.getMonth() + 1).padStart(2, '0');
  const d2 = String(date.getDate()).padStart(2, '0');
  const hh2 = String(hh).padStart(2, '0');
  const mm2 = String(mm).padStart(2, '0');
  const localStr = `${y4}-${m2}-${d2}T${hh2}:${mm2}:00`;

  // fromZonedTime interpreta el string como hora local en `tz` y devuelve el Date UTC equivalente.
  return fromZonedTime(localStr, tz).toISOString();
}

/** Suma minutos a un ISO y devuelve el nuevo ISO (UTC, conserva el tz origen). */
export function addMinutesIso(iso: string, mins: number): string {
  return new Date(new Date(iso).getTime() + mins * 60_000).toISOString();
}

/** Diferencia en minutos entre dos ISOs (ends_at - starts_at). */
export function diffMinutesIso(startIso: string, endIso: string): number {
  return Math.round((new Date(endIso).getTime() - new Date(startIso).getTime()) / 60_000);
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
