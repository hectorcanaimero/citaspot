import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  toDateStr,
  getGreeting,
  topPx,
  heightPx,
  assignColumns,
  START_HOUR,
  HOUR_PX,
  TOTAL_HOURS,
  GRID_PX,
} from '@/lib/calendar-utils';

// ── Constantes ────────────────────────────────────────────────────────────────

describe('Constantes del grid', () => {
  it('START_HOUR es 6', () => expect(START_HOUR).toBe(6));
  it('HOUR_PX es 64', () => expect(HOUR_PX).toBe(64));
  it('TOTAL_HOURS es 17 (06:00–23:00)', () => expect(TOTAL_HOURS).toBe(17));
  it('GRID_PX es 17 * 64 = 1088', () => expect(GRID_PX).toBe(1088));
});

// ── toDateStr ─────────────────────────────────────────────────────────────────

describe('toDateStr', () => {
  it('retorna formato YYYY-MM-DD', () => {
    const d = new Date('2026-03-07T15:30:00Z');
    expect(toDateStr(d)).toBe('2026-03-07');
  });

  it('no incluye la parte de tiempo', () => {
    const d = new Date('2026-12-31T23:59:59Z');
    expect(toDateStr(d)).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it('funciona con inicio del año', () => {
    const d = new Date('2026-01-01T00:00:00Z');
    expect(toDateStr(d)).toBe('2026-01-01');
  });
});

// ── getGreeting ───────────────────────────────────────────────────────────────

describe('getGreeting', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it('retorna "Buenos días" entre 00:00 y 11:59', () => {
    vi.setSystemTime(new Date('2026-03-07T09:00:00'));
    expect(getGreeting()).toBe('Buenos días');
  });

  it('retorna "Buenas tardes" entre 12:00 y 18:59', () => {
    vi.setSystemTime(new Date('2026-03-07T15:00:00'));
    expect(getGreeting()).toBe('Buenas tardes');
  });

  it('retorna "Buenas noches" a partir de las 19:00', () => {
    vi.setSystemTime(new Date('2026-03-07T21:00:00'));
    expect(getGreeting()).toBe('Buenas noches');
  });

  it('retorna "Buenos días" exactamente a medianoche (00:00)', () => {
    vi.setSystemTime(new Date('2026-03-07T00:00:00'));
    expect(getGreeting()).toBe('Buenos días');
  });

  it('retorna "Buenas tardes" exactamente al mediodía (12:00)', () => {
    vi.setSystemTime(new Date('2026-03-07T12:00:00'));
    expect(getGreeting()).toBe('Buenas tardes');
  });
});

// ── topPx ─────────────────────────────────────────────────────────────────────

describe('topPx', () => {
  // Tests con tz='UTC' e ISOs con 'Z' para evitar ambigüedad de zona local del runtime.
  const TZ = 'UTC';

  it('retorna 0 para las 06:00 UTC (inicio del grid)', () => {
    expect(topPx('2026-03-07T06:00:00Z', TZ)).toBe(0);
  });

  it('retorna HOUR_PX para una hora después del inicio', () => {
    expect(topPx('2026-03-07T07:00:00Z', TZ)).toBeCloseTo(HOUR_PX, 0);
  });

  it('calcula correctamente 30 minutos = HOUR_PX/2 px', () => {
    // 08:30 UTC → 2.5h desde START_HOUR (6) → 2.5 * HOUR_PX
    expect(topPx('2026-03-07T08:30:00Z', TZ)).toBeCloseTo(2.5 * HOUR_PX, 1);
  });

  it('el valor aumenta para horas más tardías', () => {
    expect(topPx('2026-03-07T11:00:00Z', TZ)).toBeGreaterThan(
      topPx('2026-03-07T09:00:00Z', TZ),
    );
  });

  it('respeta el timezone del tenant (Caracas UTC-4)', () => {
    // 12:00 UTC == 08:00 America/Caracas → 2h desde START_HOUR (6)
    expect(topPx('2026-03-07T12:00:00Z', 'America/Caracas')).toBeCloseTo(2 * HOUR_PX, 0);
  });
});

// ── heightPx ──────────────────────────────────────────────────────────────────

describe('heightPx', () => {
  it('60 minutos → exactamente HOUR_PX px', () => {
    expect(heightPx(60)).toBe(HOUR_PX);
  });

  it('30 minutos → HOUR_PX / 2 px', () => {
    expect(heightPx(30)).toBe(HOUR_PX / 2);
  });

  it('90 minutos → HOUR_PX * 1.5 px', () => {
    expect(heightPx(90)).toBe(HOUR_PX * 1.5);
  });

  it('duración muy corta devuelve mínimo 22px', () => {
    expect(heightPx(1)).toBe(22);
    expect(heightPx(5)).toBe(22);
  });

  it('exactamente en el límite: durationMin que produce 22px', () => {
    // 22px / (64/60) ≈ 20.6 min → heightPx(20) < 22, heightPx(21) ≈ 22.4
    const threshold = 22 / (HOUR_PX / 60); // ~20.625
    expect(heightPx(Math.floor(threshold))).toBe(22);
    expect(heightPx(Math.ceil(threshold) + 1)).toBeGreaterThan(22);
  });
});

// ── assignColumns ─────────────────────────────────────────────────────────────

/** Helper para construir citas mínimas */
function makeAppt(id: string, startsAt: string, endsAt: string) {
  return { id, starts_at: startsAt, ends_at: endsAt };
}

describe('assignColumns', () => {
  it('retorna [] con entrada vacía', () => {
    expect(assignColumns([])).toEqual([]);
  });

  it('asigna col=0, span=1 a una cita solitaria', () => {
    const result = assignColumns([
      makeAppt('a', '2026-03-07T09:00:00', '2026-03-07T10:00:00'),
    ]);
    expect(result).toHaveLength(1);
    expect(result[0].col).toBe(0);
    expect(result[0].span).toBe(1);
  });

  it('dos citas que no solapan → ambas con span=1', () => {
    const result = assignColumns([
      makeAppt('a', '2026-03-07T09:00:00', '2026-03-07T10:00:00'),
      makeAppt('b', '2026-03-07T11:00:00', '2026-03-07T12:00:00'),
    ]);
    expect(result).toHaveLength(2);
    expect(result.find(r => r.id === 'a')!.span).toBe(1);
    expect(result.find(r => r.id === 'b')!.span).toBe(1);
  });

  it('dos citas solapadas → span=2, cols 0 y 1', () => {
    const result = assignColumns([
      makeAppt('a', '2026-03-07T09:00:00', '2026-03-07T10:30:00'),
      makeAppt('b', '2026-03-07T09:30:00', '2026-03-07T11:00:00'),
    ]);
    expect(result).toHaveLength(2);
    const cols = result.map(r => r.col).sort();
    expect(cols).toEqual([0, 1]);
    expect(result.every(r => r.span === 2)).toBe(true);
  });

  it('tres citas solapadas → span=3, cols 0/1/2', () => {
    const result = assignColumns([
      makeAppt('a', '2026-03-07T10:00:00', '2026-03-07T12:00:00'),
      makeAppt('b', '2026-03-07T10:30:00', '2026-03-07T11:30:00'),
      makeAppt('c', '2026-03-07T11:00:00', '2026-03-07T13:00:00'),
    ]);
    expect(result).toHaveLength(3);
    const cols = result.map(r => r.col).sort((a, b) => a - b);
    expect(cols).toEqual([0, 1, 2]);
    expect(result.every(r => r.span === 3)).toBe(true);
  });

  it('dos clusters independientes → cada uno con su propio span', () => {
    const result = assignColumns([
      makeAppt('a', '2026-03-07T09:00:00', '2026-03-07T10:30:00'),
      makeAppt('b', '2026-03-07T09:30:00', '2026-03-07T11:00:00'),
      makeAppt('c', '2026-03-07T14:00:00', '2026-03-07T15:00:00'),
      makeAppt('d', '2026-03-07T14:30:00', '2026-03-07T16:00:00'),
    ]);
    expect(result).toHaveLength(4);
    const firstCluster  = result.filter(r => r.id === 'a' || r.id === 'b');
    const secondCluster = result.filter(r => r.id === 'c' || r.id === 'd');
    expect(firstCluster.every(r => r.span === 2)).toBe(true);
    expect(secondCluster.every(r => r.span === 2)).toBe(true);
  });

  it('ordena por starts_at antes de procesar (entrada desordenada)', () => {
    const result = assignColumns([
      makeAppt('b', '2026-03-07T10:30:00', '2026-03-07T11:30:00'),
      makeAppt('a', '2026-03-07T09:00:00', '2026-03-07T11:00:00'),
    ]);
    expect(result).toHaveLength(2);
    // 'a' empieza antes → debería ser col=0
    expect(result.find(r => r.id === 'a')!.col).toBe(0);
    expect(result.find(r => r.id === 'b')!.col).toBe(1);
  });

  it('preserva todos los campos del objeto original', () => {
    const appts = [{ id: 'x', starts_at: '2026-03-07T10:00:00', ends_at: '2026-03-07T11:00:00', name: 'Test' }];
    const result = assignColumns(appts);
    expect(result[0].name).toBe('Test');
    expect(result[0].id).toBe('x');
  });
});
