'use client';

// Editor de horarios semanales de un profesional.
// Extraído de app/dashboard/team/page.tsx para reutilizar en el detalle.

import { useState } from 'react';
import { Check } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { ToggleSwitch } from '@/components/ui/toggle-switch';
import { professionals as profsApi, Schedule, APIError } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

// Días de la semana (Lun..Dom) — las etiquetas vienen de t.team.days
const DAYS_DOW = [1, 2, 3, 4, 5, 6, 0];

export interface DayConfig {
  is_active: boolean;
  start_time: string;
  end_time: string;
}

export type DayMap = Record<number, DayConfig>;

export function defaultDays(): DayMap {
  return Object.fromEntries(
    DAYS_DOW.map((dow) => [
      dow,
      { is_active: dow >= 1 && dow <= 5, start_time: '08:00', end_time: '18:00' },
    ]),
  );
}

export function scheduleToDayMap(schedules: Schedule[]): DayMap {
  const map = defaultDays();
  DAYS_DOW.forEach((dow) => {
    map[dow].is_active = false;
  });
  schedules.forEach((s) => {
    map[s.day_of_week] = {
      is_active: s.is_active,
      start_time: s.start_time,
      end_time: s.end_time,
    };
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

  function toggle(dow: number) {
    setDays((d) => ({ ...d, [dow]: { ...d[dow], is_active: !d[dow].is_active } }));
    setSaved(false);
  }

  function updateDay(dow: number, field: 'start_time' | 'end_time', value: string) {
    setDays((d) => ({ ...d, [dow]: { ...d[dow], [field]: value } }));
    setSaved(false);
  }

  async function save() {
    setSaving(true);
    setError('');
    try {
      // Enviar los 7 días con su estado real (activo/inactivo).
      // Filtrar solo activos causaba que los días desactivados no se persistieran en DB.
      const schedules = DAYS_DOW.map((dow) => ({
        day_of_week: dow,
        start_time: days[dow].start_time,
        end_time: days[dow].end_time,
        is_active: days[dow].is_active,
      }));
      await profsApi.setSchedule(profId, schedules);
      setSaved(true);
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.team.saveError);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="flex flex-col gap-3">
      {DAYS_DOW.map((dow) => {
        const day = days[dow];
        return (
          <div key={dow} className="flex items-center gap-3">
            <ToggleSwitch checked={day.is_active} onCheckedChange={() => toggle(dow)} />
            <span
              className={`w-8 text-sm ${day.is_active ? 'text-neutral-900 font-medium' : 'text-neutral-400'}`}
            >
              {dayLabels[String(dow)]}
            </span>
            {day.is_active && (
              <div className="flex flex-1 items-center gap-2">
                <input
                  type="time"
                  value={day.start_time}
                  onChange={(e) => updateDay(dow, 'start_time', e.target.value)}
                  className="h-7 rounded-md border border-neutral-200 px-2 text-xs text-neutral-900 focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
                <span className="text-xs text-neutral-400">—</span>
                <input
                  type="time"
                  value={day.end_time}
                  onChange={(e) => updateDay(dow, 'end_time', e.target.value)}
                  className="h-7 rounded-md border border-neutral-200 px-2 text-xs text-neutral-900 focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
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
