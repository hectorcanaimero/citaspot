'use client';

// Editor de horarios semanales de un profesional.
// Soporta múltiples bloques por día (ej: 08:00-12:00 + 14:00-18:00).

import { useState } from 'react';
import { Check, Plus, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { ToggleSwitch } from '@/components/ui/toggle-switch';
import { professionals as profsApi, Schedule, APIError } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

const DAYS_DOW = [1, 2, 3, 4, 5, 6, 0];

export interface TimeBlock {
  start_time: string;
  end_time: string;
}

export interface DayConfig {
  is_active: boolean;
  blocks: TimeBlock[];
}

export type DayMap = Record<number, DayConfig>;

function defaultBlocks(): TimeBlock[] {
  return [{ start_time: '08:00', end_time: '18:00' }];
}

export function defaultDays(): DayMap {
  return Object.fromEntries(
    DAYS_DOW.map((dow) => [
      dow,
      { is_active: dow >= 1 && dow <= 5, blocks: defaultBlocks() },
    ]),
  );
}

// Agrupa el array de Schedule por día. Un día queda activo si tiene al menos
// un bloque activo. Si no hay bloques para un día, se inicializa con default
// inactivo (preserva memoria visual).
export function scheduleToDayMap(schedules: Schedule[]): DayMap {
  const map: DayMap = Object.fromEntries(
    DAYS_DOW.map((dow) => [dow, { is_active: false, blocks: [] as TimeBlock[] }]),
  );
  schedules.forEach((s) => {
    map[s.day_of_week].blocks.push({ start_time: s.start_time, end_time: s.end_time });
    if (s.is_active) map[s.day_of_week].is_active = true;
  });
  DAYS_DOW.forEach((dow) => {
    if (map[dow].blocks.length === 0) {
      map[dow].blocks = defaultBlocks();
    } else {
      map[dow].blocks.sort((a, b) => a.start_time.localeCompare(b.start_time));
    }
  });
  return map;
}

interface ScheduleEditorProps {
  profId: string;
  initial: DayMap;
}

export function ScheduleEditor({ profId, initial }: ScheduleEditorProps) {
  const t = useTranslations();
  const dayLabels = t.team.days as Record<string, string>;

  const [days, setDays] = useState<DayMap>(initial);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState('');

  function toggleDay(dow: number) {
    setDays((d) => ({ ...d, [dow]: { ...d[dow], is_active: !d[dow].is_active } }));
    setSaved(false);
  }

  function updateBlock(dow: number, idx: number, field: keyof TimeBlock, value: string) {
    setDays((d) => {
      const blocks = d[dow].blocks.map((b, i) => (i === idx ? { ...b, [field]: value } : b));
      return { ...d, [dow]: { ...d[dow], blocks } };
    });
    setSaved(false);
  }

  function addBlock(dow: number) {
    setDays((d) => {
      const last = d[dow].blocks[d[dow].blocks.length - 1];
      const next: TimeBlock = last
        ? { start_time: last.end_time, end_time: '20:00' }
        : { start_time: '08:00', end_time: '12:00' };
      return { ...d, [dow]: { ...d[dow], blocks: [...d[dow].blocks, next] } };
    });
    setSaved(false);
  }

  function removeBlock(dow: number, idx: number) {
    setDays((d) => {
      const blocks = d[dow].blocks.filter((_, i) => i !== idx);
      // Si quedó vacío, deshabilitar el día (mantiene un bloque por defecto para la UI)
      if (blocks.length === 0) {
        return { ...d, [dow]: { is_active: false, blocks: defaultBlocks() } };
      }
      return { ...d, [dow]: { ...d[dow], blocks } };
    });
    setSaved(false);
  }

  function hasOverlap(dow: number): boolean {
    if (!days[dow].is_active) return false;
    const blocks = [...days[dow].blocks].sort((a, b) =>
      a.start_time.localeCompare(b.start_time),
    );
    for (let i = 0; i < blocks.length; i++) {
      if (blocks[i].start_time >= blocks[i].end_time) return true;
      if (i > 0 && blocks[i].start_time < blocks[i - 1].end_time) return true;
    }
    return false;
  }

  const anyOverlap = DAYS_DOW.some((dow) => hasOverlap(dow));

  async function save() {
    if (anyOverlap) {
      setError(t.team.scheduleOverlap);
      return;
    }
    setSaving(true);
    setError('');
    try {
      const schedules: Schedule[] = [];
      DAYS_DOW.forEach((dow) => {
        days[dow].blocks.forEach((b) => {
          schedules.push({
            day_of_week: dow,
            start_time: b.start_time,
            end_time: b.end_time,
            is_active: days[dow].is_active,
          });
        });
      });
      await profsApi.setSchedule(profId, schedules);
      setSaved(true);
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.team.saveError);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {DAYS_DOW.map((dow) => {
        const day = days[dow];
        const overlap = hasOverlap(dow);
        return (
          <div key={dow} className="flex flex-col gap-2 border-b border-neutral-100 pb-3 last:border-b-0 last:pb-0">
            <div className="flex items-center gap-3">
              <ToggleSwitch checked={day.is_active} onCheckedChange={() => toggleDay(dow)} />
              <span
                className={`w-20 text-sm ${day.is_active ? 'text-neutral-900 font-medium' : 'text-neutral-400'}`}
              >
                {dayLabels[String(dow)]}
              </span>
              {day.is_active && (
                <button
                  type="button"
                  onClick={() => addBlock(dow)}
                  className="ml-auto flex items-center gap-1 rounded-md border border-neutral-200 px-2 py-1 text-xs text-neutral-600 hover:bg-neutral-50 transition-colors"
                >
                  <Plus className="h-3 w-3" />
                  {t.team.addBlock}
                </button>
              )}
            </div>
            {day.is_active && (
              <div className="ml-[3.25rem] flex flex-col gap-1.5">
                {day.blocks.map((b, idx) => (
                  <div key={idx} className="flex items-center gap-2">
                    <input
                      type="time"
                      value={b.start_time}
                      onChange={(e) => updateBlock(dow, idx, 'start_time', e.target.value)}
                      className="h-7 rounded-md border border-neutral-200 px-2 text-xs text-neutral-900 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    />
                    <span className="text-xs text-neutral-400">—</span>
                    <input
                      type="time"
                      value={b.end_time}
                      onChange={(e) => updateBlock(dow, idx, 'end_time', e.target.value)}
                      className="h-7 rounded-md border border-neutral-200 px-2 text-xs text-neutral-900 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    />
                    {day.blocks.length > 1 && (
                      <button
                        type="button"
                        onClick={() => removeBlock(dow, idx)}
                        className="rounded-md p-1 text-neutral-400 hover:bg-red-50 hover:text-red-600 transition-colors"
                        title={t.team.removeBlock}
                      >
                        <X className="h-3.5 w-3.5" />
                      </button>
                    )}
                  </div>
                ))}
                {overlap && (
                  <p className="text-xs text-red-600">{t.team.scheduleOverlap}</p>
                )}
              </div>
            )}
          </div>
        );
      })}

      {error && <p className="rounded-md bg-red-50 px-3 py-1.5 text-xs text-red-700">{error}</p>}

      <Button
        size="sm"
        variant={saved ? 'secondary' : 'primary'}
        onClick={save}
        loading={saving}
        disabled={anyOverlap}
        className="self-end"
      >
        {saved ? (
          <>
            <Check className="mr-1.5 h-3.5 w-3.5" />
            {t.team.saved}
          </>
        ) : (
          t.team.saveSchedule
        )}
      </Button>
    </div>
  );
}
