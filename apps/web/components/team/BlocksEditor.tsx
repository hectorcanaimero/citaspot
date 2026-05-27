'use client';

// Editor de bloqueos de horario de un profesional.
// Soporta bloqueos puntuales (vacaciones, congresos) y recurrentes (ej: almuerzo cada día).

import { useState, useEffect, useCallback } from 'react';
import { Plus, Trash2, Calendar, Repeat, X } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Spinner } from '@/components/ui/spinner';
import { Badge } from '@/components/ui/badge';
import { scheduleBlocks as blocksApi, ScheduleBlock, APIError } from '@/lib/api';
import { useTranslations, useDateLocale } from '@/lib/i18n';
import { format } from 'date-fns';

const DAYS_DOW = [1, 2, 3, 4, 5, 6, 0];

interface BlocksEditorProps {
  profId: string;
}

export function BlocksEditor({ profId }: BlocksEditorProps) {
  const t = useTranslations();
  const dateLocale = useDateLocale();
  const dayLabels = t.team.days as Record<string, string>;

  const [blocks, setBlocks] = useState<ScheduleBlock[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await blocksApi.list(profId);
      setBlocks(res.data ?? []);
    } catch {
      setBlocks([]);
    } finally {
      setLoading(false);
    }
  }, [profId]);

  useEffect(() => {
    load();
  }, [load]);

  async function handleDelete(id: string) {
    if (!window.confirm(t.team.confirmDeleteBlock)) return;
    setDeleting(id);
    try {
      await blocksApi.delete(id);
      setBlocks((bs) => bs.filter((b) => b.id !== id));
    } catch {
      /* silencioso — UI se mantiene */
    } finally {
      setDeleting(null);
    }
  }

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="p-5">
      <div className="mb-4 flex items-center justify-between">
        <p className="text-xs text-neutral-500">{t.team.blocksDesc}</p>
        <Button size="sm" variant="primary" onClick={() => setShowModal(true)}>
          <Plus className="mr-1.5 h-3.5 w-3.5" />
          {t.team.newBlock}
        </Button>
      </div>

      {blocks.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <Calendar className="mb-3 h-10 w-10 text-neutral-200" />
          <p className="text-sm font-medium text-neutral-500">{t.team.noBlocks}</p>
          <p className="mt-1 text-xs text-neutral-400">{t.team.noBlocksDesc}</p>
        </div>
      ) : (
        <ul className="divide-y divide-neutral-100">
          {blocks.map((b) => (
            <li key={b.id} className="flex items-start gap-3 py-3">
              <div className="mt-0.5 flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-full bg-neutral-100 text-neutral-500">
                {b.is_recurring ? (
                  <Repeat className="h-3.5 w-3.5" />
                ) : (
                  <Calendar className="h-3.5 w-3.5" />
                )}
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-neutral-900">
                    {b.reason || t.team.blockNoReason}
                  </span>
                  <Badge variant={b.is_recurring ? 'primary' : 'default'}>
                    {b.is_recurring ? t.team.recurring : t.team.oneTime}
                  </Badge>
                </div>
                <p className="mt-0.5 text-xs text-neutral-500">
                  {b.is_recurring
                    ? formatRecurring(b, dayLabels)
                    : `${format(new Date(b.starts_at), "d MMM yyyy 'a las' HH:mm", { locale: dateLocale })} — ${format(
                        new Date(b.ends_at),
                        'HH:mm',
                        { locale: dateLocale },
                      )}`}
                </p>
              </div>
              <button
                type="button"
                onClick={() => handleDelete(b.id)}
                disabled={deleting === b.id}
                className="rounded-md p-1.5 text-neutral-400 hover:bg-red-50 hover:text-red-600 transition-colors disabled:opacity-50"
                title={t.common.delete}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </li>
          ))}
        </ul>
      )}

      {showModal && (
        <NewBlockModal
          profId={profId}
          onClose={() => setShowModal(false)}
          onCreated={(b) => {
            setBlocks((bs) => [...bs, b]);
            setShowModal(false);
          }}
        />
      )}
    </div>
  );
}

function formatRecurring(b: ScheduleBlock, dayLabels: Record<string, string>): string {
  const days = (b.recurrence_days ?? []).map((d) => dayLabels[String(d)]).join(', ');
  const start = b.starts_at.slice(11, 16);
  const end = b.ends_at.slice(11, 16);
  return `${days} · ${start}–${end}`;
}

// ── Modal de creación ─────────────────────────────────────────────────────────

interface NewBlockModalProps {
  profId: string;
  onClose: () => void;
  onCreated: (b: ScheduleBlock) => void;
}

function NewBlockModal({ profId, onClose, onCreated }: NewBlockModalProps) {
  const t = useTranslations();
  const dayLabels = t.team.days as Record<string, string>;

  const [mode, setMode] = useState<'one_time' | 'recurring'>('one_time');
  const [startsAt, setStartsAt] = useState('');
  const [endsAt, setEndsAt] = useState('');
  const [startTime, setStartTime] = useState('12:00');
  const [endTime, setEndTime] = useState('13:00');
  const [days, setDays] = useState<Set<number>>(new Set());
  const [reason, setReason] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  function toggleDay(d: number) {
    setDays((s) => {
      const next = new Set(s);
      if (next.has(d)) next.delete(d);
      else next.add(d);
      return next;
    });
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError('');

    if (mode === 'one_time') {
      if (!startsAt || !endsAt) {
        setError(t.team.blockMissingDates);
        return;
      }
      if (new Date(endsAt) <= new Date(startsAt)) {
        setError(t.team.blockEndBeforeStart);
        return;
      }
    } else {
      if (days.size === 0) {
        setError(t.team.blockMissingDays);
        return;
      }
      if (startTime >= endTime) {
        setError(t.team.blockEndBeforeStart);
        return;
      }
    }

    setSaving(true);
    try {
      const payload =
        mode === 'one_time'
          ? {
              professional_id: profId,
              starts_at: new Date(startsAt).toISOString(),
              ends_at: new Date(endsAt).toISOString(),
              reason: reason.trim(),
              is_recurring: false,
            }
          : {
              professional_id: profId,
              // Para recurrentes el backend usa solo la parte TIME — fecha es irrelevante.
              starts_at: new Date(`1970-01-01T${startTime}:00Z`).toISOString(),
              ends_at: new Date(`1970-01-01T${endTime}:00Z`).toISOString(),
              reason: reason.trim(),
              is_recurring: true,
              recurrence_days: Array.from(days).sort(),
            };
      const created = await blocksApi.create(payload);
      onCreated(created);
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.team.saveError);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-md overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-xl">
        <div className="flex items-center justify-between border-b border-neutral-100 px-5 py-4">
          <h3 className="text-sm font-semibold text-neutral-900">{t.team.newBlock}</h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <form onSubmit={submit} className="flex flex-col gap-3 px-5 py-4">
          {/* Mode toggle */}
          <div className="grid grid-cols-2 gap-1.5 rounded-lg bg-neutral-100 p-1">
            <button
              type="button"
              onClick={() => setMode('one_time')}
              className={`rounded-md py-2 text-xs font-medium transition-colors ${
                mode === 'one_time' ? 'bg-white text-neutral-900 shadow-sm' : 'text-neutral-500'
              }`}
            >
              {t.team.oneTime}
            </button>
            <button
              type="button"
              onClick={() => setMode('recurring')}
              className={`rounded-md py-2 text-xs font-medium transition-colors ${
                mode === 'recurring' ? 'bg-white text-neutral-900 shadow-sm' : 'text-neutral-500'
              }`}
            >
              {t.team.recurring}
            </button>
          </div>

          {mode === 'one_time' ? (
            <>
              <div className="flex flex-col gap-1">
                <label className="text-xs font-medium text-neutral-700">{t.team.blockStartsAt}</label>
                <input
                  type="datetime-local"
                  value={startsAt}
                  onChange={(e) => setStartsAt(e.target.value)}
                  className="h-9 rounded-md border border-neutral-200 px-2 text-sm text-neutral-900 focus:outline-none focus:ring-2 focus:ring-primary-500"
                  required
                />
              </div>
              <div className="flex flex-col gap-1">
                <label className="text-xs font-medium text-neutral-700">{t.team.blockEndsAt}</label>
                <input
                  type="datetime-local"
                  value={endsAt}
                  onChange={(e) => setEndsAt(e.target.value)}
                  className="h-9 rounded-md border border-neutral-200 px-2 text-sm text-neutral-900 focus:outline-none focus:ring-2 focus:ring-primary-500"
                  required
                />
              </div>
            </>
          ) : (
            <>
              <div className="flex flex-col gap-1">
                <label className="text-xs font-medium text-neutral-700">{t.team.blockDays}</label>
                <div className="grid grid-cols-7 gap-1">
                  {DAYS_DOW.map((d) => (
                    <button
                      key={d}
                      type="button"
                      onClick={() => toggleDay(d)}
                      className={`rounded-md border px-1.5 py-1.5 text-xs transition-colors ${
                        days.has(d)
                          ? 'border-primary-500 bg-primary-50 text-primary-700 font-medium'
                          : 'border-neutral-200 text-neutral-600 hover:bg-neutral-50'
                      }`}
                    >
                      {dayLabels[String(d)]}
                    </button>
                  ))}
                </div>
              </div>
              <div className="flex gap-2">
                <div className="flex flex-1 flex-col gap-1">
                  <label className="text-xs font-medium text-neutral-700">{t.team.blockTimeFrom}</label>
                  <input
                    type="time"
                    value={startTime}
                    onChange={(e) => setStartTime(e.target.value)}
                    className="h-9 rounded-md border border-neutral-200 px-2 text-sm text-neutral-900 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    required
                  />
                </div>
                <div className="flex flex-1 flex-col gap-1">
                  <label className="text-xs font-medium text-neutral-700">{t.team.blockTimeTo}</label>
                  <input
                    type="time"
                    value={endTime}
                    onChange={(e) => setEndTime(e.target.value)}
                    className="h-9 rounded-md border border-neutral-200 px-2 text-sm text-neutral-900 focus:outline-none focus:ring-2 focus:ring-primary-500"
                    required
                  />
                </div>
              </div>
            </>
          )}

          <Input
            label={t.team.blockReason}
            placeholder={t.team.blockReasonPlaceholder}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />

          {error && <p className="rounded-md bg-red-50 px-3 py-1.5 text-xs text-red-700">{error}</p>}

          <div className="flex justify-end gap-2 pt-1">
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg border border-neutral-200 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-50 transition-colors"
            >
              {t.common.cancel}
            </button>
            <Button type="submit" size="sm" loading={saving}>
              {t.team.createBlock}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
