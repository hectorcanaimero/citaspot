'use client';

import { useState, useEffect } from 'react';
import { useTranslations } from '@/lib/i18n';
import { treatmentSessions, TreatmentSessionInput, professionals } from '@/lib/api';

interface Professional {
  id: string;
  name: string;
}

interface SessionModalProps {
  isOpen: boolean;
  onClose: () => void;
  treatmentId: string;
  onSuccess: () => void;
}

type Mode = 'schedule' | 'log';

const DURATION_OPTIONS = [15, 30, 45, 60, 90] as const;

export function SessionModal({ isOpen, onClose, treatmentId, onSuccess }: SessionModalProps) {
  const t = useTranslations();
  const s = t.crm.sessions;
  const [mode, setMode] = useState<Mode>('schedule');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [profList, setProfList] = useState<Professional[]>([]);

  useEffect(() => {
    if (!isOpen) return;
    professionals.list().then(res => setProfList(res.data.map(p => ({ id: p.id, name: p.name }))));
  }, [isOpen]);

  const [form, setForm] = useState({
    professional_id: '',
    scheduled_at: '',
    time: '10:00',
    duration_minutes: 45,
    notes: '',
    procedures_done: '',
    paid_in_session: '',
    currency: 'USD',
    next_session_at: '',
  });

  if (!isOpen) return null;

  const set = (key: string, value: string | number) =>
    setForm(prev => ({ ...prev, [key]: value }));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const scheduledAt = mode === 'schedule'
        ? new Date(`${form.scheduled_at}T${form.time}:00`).toISOString()
        : new Date(form.scheduled_at).toISOString();

      const input: TreatmentSessionInput = {
        professional_id: form.professional_id,
        status: mode === 'schedule' ? 'pending' : 'completed',
        scheduled_at: scheduledAt,
        duration_minutes: form.duration_minutes || undefined,
        notes: form.notes || undefined,
        procedures_done: mode === 'log' ? form.procedures_done || undefined : undefined,
        paid_in_session: mode === 'log' && form.paid_in_session ? Number(form.paid_in_session) : undefined,
        currency: mode === 'log' && form.paid_in_session ? form.currency : undefined,
        next_session_at: form.next_session_at ? new Date(form.next_session_at).toISOString() : undefined,
      };

      await treatmentSessions.create(treatmentId, input);
      onSuccess();
      onClose();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Error al guardar la sesión');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto">
        <div className="p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-slate-900">
              {mode === 'schedule' ? s.scheduleSession : s.logCompletedSession}
            </h2>
            <button onClick={onClose} className="text-slate-400 hover:text-slate-600">✕</button>
          </div>

          {/* Toggle */}
          <div className="flex gap-1 bg-slate-100 rounded-lg p-1 mb-5">
            <button
              type="button"
              onClick={() => setMode('schedule')}
              className={`flex-1 py-2 px-3 rounded-md text-sm font-medium transition-all ${
                mode === 'schedule'
                  ? 'bg-white text-indigo-600 shadow-sm'
                  : 'text-slate-500 hover:text-slate-700'
              }`}
            >
              📅 {s.schedule}
            </button>
            <button
              type="button"
              onClick={() => setMode('log')}
              className={`flex-1 py-2 px-3 rounded-md text-sm font-medium transition-all ${
                mode === 'log'
                  ? 'bg-white text-indigo-600 shadow-sm'
                  : 'text-slate-500 hover:text-slate-700'
              }`}
            >
              ✓ {s.logCompleted}
            </button>
          </div>

          <form onSubmit={handleSubmit} className="space-y-4">
            {/* Profesional */}
            <div>
              <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                {s.professional}
              </label>
              <select
                value={form.professional_id}
                onChange={e => set('professional_id', e.target.value)}
                required
                className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                <option value="">—</option>
                {profList.map(p => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>

            {/* Fecha + Hora */}
            <div className={`grid gap-3 ${mode === 'schedule' ? 'grid-cols-2' : 'grid-cols-1'}`}>
              <div>
                <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                  {s.date}
                </label>
                <input
                  type="date"
                  value={form.scheduled_at}
                  onChange={e => set('scheduled_at', e.target.value)}
                  required
                  className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                />
              </div>
              {mode === 'schedule' && (
                <div>
                  <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                    {s.time}
                  </label>
                  <input
                    type="time"
                    value={form.time}
                    onChange={e => set('time', e.target.value)}
                    required
                    className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  />
                </div>
              )}
            </div>

            {/* Duración */}
            <div>
              <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                {mode === 'schedule' ? s.estimatedDuration : s.actualDuration}
              </label>
              <select
                value={form.duration_minutes}
                onChange={e => set('duration_minutes', Number(e.target.value))}
                className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              >
                {DURATION_OPTIONS.map(min => (
                  <option key={min} value={min}>{(s.duration as Record<string, string>)[`min${min}`]}</option>
                ))}
              </select>
            </div>

            {/* Procedimientos (solo log) */}
            {mode === 'log' && (
              <div>
                <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                  {s.proceduresDone}
                </label>
                <textarea
                  value={form.procedures_done}
                  onChange={e => set('procedures_done', e.target.value)}
                  rows={3}
                  className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none"
                  placeholder="Ej: Ajuste de brackets superiores, radiografía..."
                />
              </div>
            )}

            {/* Notas */}
            <div>
              <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                {mode === 'log' ? s.clinicalNotes : s.priorNotes}
              </label>
              <textarea
                value={form.notes}
                onChange={e => set('notes', e.target.value)}
                rows={2}
                className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 resize-none"
              />
            </div>

            {/* Cobrado (solo log) */}
            {mode === 'log' && (
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                    {s.paidInSession}
                  </label>
                  <input
                    type="number"
                    min="0"
                    step="0.01"
                    value={form.paid_in_session}
                    onChange={e => set('paid_in_session', e.target.value)}
                    className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                    placeholder="0.00"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                    Moneda
                  </label>
                  <select
                    value={form.currency}
                    onChange={e => set('currency', e.target.value)}
                    className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  >
                    <option value="USD">USD</option>
                    <option value="DOP">DOP</option>
                    <option value="VES">VES</option>
                    <option value="EUR">EUR</option>
                  </select>
                </div>
              </div>
            )}

            {/* Próxima sesión */}
            <div>
              <label className="block text-xs font-semibold text-slate-500 uppercase tracking-wide mb-1">
                {s.nextSession}
              </label>
              <input
                type="date"
                value={form.next_session_at}
                onChange={e => set('next_session_at', e.target.value)}
                className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
              />
            </div>

            {error && (
              <p className="text-sm text-red-600 bg-red-50 rounded-lg px-3 py-2">{error}</p>
            )}

            <button
              type="submit"
              disabled={loading}
              className={`w-full py-2.5 px-4 rounded-lg text-sm font-semibold text-white transition-colors ${
                mode === 'schedule'
                  ? 'bg-indigo-600 hover:bg-indigo-700 disabled:bg-indigo-300'
                  : 'bg-green-600 hover:bg-green-700 disabled:bg-green-300'
              }`}
            >
              {loading ? '...' : mode === 'schedule' ? s.scheduleBtn : s.logBtn}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
