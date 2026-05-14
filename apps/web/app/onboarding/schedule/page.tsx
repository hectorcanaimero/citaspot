'use client';

// Paso 2 del onboarding: crear el primer profesional y definir sus horarios.
import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button } from '@/components/ui/button';
import { Input }  from '@/components/ui/input';
import { Card }   from '@/components/ui/card';
import { ToggleSwitch } from '@/components/ui/toggle-switch';
import { professionals as profsApi, APIError } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

const DAYS_DOW = [1, 2, 3, 4, 5, 6, 0];

interface DayConfig {
  is_active: boolean;
  start_time: string;
  end_time: string;
}

const DEFAULT_DAYS: Record<number, DayConfig> = Object.fromEntries(
  DAYS_DOW.map((dow) => [dow, { is_active: dow >= 1 && dow <= 5, start_time: '08:00', end_time: '18:00' }])
);

export default function OnboardingSchedulePage() {
  const t      = useTranslations();
  const router = useRouter();
  const [name,   setName]   = useState('');
  const [saving, setSaving] = useState(false);
  const [error,  setError]  = useState('');
  const [days,   setDays]   = useState<Record<number, DayConfig>>(DEFAULT_DAYS);

  const dayLabels = t.team.days as Record<string, string>;

  function toggleDay(dow: number) {
    setDays((d) => ({ ...d, [dow]: { ...d[dow], is_active: !d[dow].is_active } }));
  }

  function updateDay(dow: number, field: 'start_time' | 'end_time', value: string) {
    setDays((d) => ({ ...d, [dow]: { ...d[dow], [field]: value } }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) { setError(t.onboarding.professionalRequired); return; }

    const activeDays = Object.entries(days).filter(([, d]) => d.is_active);
    if (activeDays.length === 0) { setError(t.onboarding.atLeastOneDay); return; }

    setSaving(true);
    setError('');
    try {
      const prof = await profsApi.create({ name: name.trim(), is_active: true });

      const schedules = activeDays.map(([dow, d]) => ({
        day_of_week: parseInt(dow, 10),
        start_time:  d.start_time,
        end_time:    d.end_time,
        is_active:   true,
      }));
      await profsApi.setSchedule(prof.id, schedules);

      router.push('/onboarding/whatsapp');
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.onboarding.scheduleError);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="mx-auto max-w-lg px-4 py-10">
      {/* Progreso */}
      <div className="mb-2 flex gap-2">
        {[1, 2, 3, 4].map((s) => (
          <div
            key={s}
            className={`h-1.5 flex-1 rounded-full transition-colors duration-300 ${
              s <= 2 ? 'bg-primary-600' : 'bg-neutral-200'
            }`}
          />
        ))}
      </div>
      <p className="mb-6 text-xs text-neutral-400">
        {t.onboarding.stepOf
          .replace('{step}', '2')
          .replace('{total}', '4')
          .replace('{label}', t.onboarding.scheduleLabel)}
      </p>

      <h1 className="mb-1 text-xl font-bold text-neutral-900">{t.onboarding.scheduleTitle}</h1>
      <p className="mb-6 text-sm text-neutral-500">{t.onboarding.scheduleDesc}</p>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <Card>
          <Input
            label={t.onboarding.professionalNameLabel}
            placeholder={t.onboarding.professionalNamePlaceholder}
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
        </Card>

        <Card>
          <p className="mb-3 text-sm font-medium text-neutral-700">{t.onboarding.workingHours}</p>
          <div className="flex flex-col gap-3">
            {DAYS_DOW.map((dow) => {
              const day = days[dow];
              return (
                <div key={dow} className="flex items-center gap-3">
                  {/* Toggle */}
                  <ToggleSwitch checked={day.is_active} onCheckedChange={() => toggleDay(dow)} />

                  {/* Día */}
                  <span className={`w-20 text-sm ${day.is_active ? 'text-neutral-900 font-medium' : 'text-neutral-400'}`}>
                    {dayLabels[String(dow)]}
                  </span>

                  {/* Horas */}
                  {day.is_active && (
                    <div className="flex flex-1 items-center gap-2">
                      <input
                        type="time"
                        value={day.start_time}
                        onChange={(e) => updateDay(dow, 'start_time', e.target.value)}
                        className="h-8 rounded-md border border-neutral-200 px-2 text-sm text-neutral-900 focus:outline-none focus:ring-2 focus:ring-primary-500"
                      />
                      <span className="text-xs text-neutral-400">—</span>
                      <input
                        type="time"
                        value={day.end_time}
                        onChange={(e) => updateDay(dow, 'end_time', e.target.value)}
                        className="h-8 rounded-md border border-neutral-200 px-2 text-sm text-neutral-900 focus:outline-none focus:ring-2 focus:ring-primary-500"
                      />
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </Card>

        {error && (
          <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">{error}</p>
        )}

        <div className="flex gap-3">
          <Button type="button" variant="secondary" onClick={() => router.push('/onboarding/services')}>
            {t.onboarding.backButton}
          </Button>
          <Button type="submit" size="lg" loading={saving} className="flex-1">
            {t.onboarding.continueButton}
          </Button>
        </div>
      </form>
    </div>
  );
}
