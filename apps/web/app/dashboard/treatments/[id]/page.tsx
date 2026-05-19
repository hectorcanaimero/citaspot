'use client';

import { useEffect, useState, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { useTranslations } from '@/lib/i18n';
import { treatments, treatmentSessions, Treatment, TreatmentSession } from '@/lib/api';
import { SessionModal } from '@/components/sessions/SessionModal';

const STATUS_COLORS: Record<string, string> = {
  pending:   'bg-violet-100 text-violet-700',
  completed: 'bg-green-100 text-green-700',
  cancelled: 'bg-red-100 text-red-600',
};

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('es', { day: 'numeric', month: 'short', year: 'numeric' });
}

export default function TreatmentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const t = useTranslations();
  const s = t.crm.sessions;

  const [treatment, setTreatment] = useState<Treatment | null>(null);
  const [sessions, setSessions] = useState<TreatmentSession[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [completing, setCompleting] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const [tr, sessionsRes] = await Promise.all([
        treatments.getById(id),
        treatmentSessions.list(id),
      ]);
      setTreatment(tr);
      setSessions(sessionsRes.data);
    } catch {
      router.push('/dashboard/treatments');
    } finally {
      setLoading(false);
    }
  }, [id, router]);

  useEffect(() => { load(); }, [load]);

  const handleComplete = async (session: TreatmentSession) => {
    setCompleting(session.id);
    try {
      await treatmentSessions.update(id, session.id, {
        status: 'completed',
        duration_minutes: session.duration_minutes ?? undefined,
        procedures_done: session.procedures_done,
        notes: session.notes,
      });
      load();
    } finally {
      setCompleting(null);
    }
  };

  const handleCancel = async (session: TreatmentSession) => {
    await treatmentSessions.update(id, session.id, {
      status: 'cancelled',
      duration_minutes: session.duration_minutes ?? undefined,
    });
    load();
  };

  const handleDelete = async (session: TreatmentSession) => {
    if (!confirm('¿Eliminar esta sesión?')) return;
    await treatmentSessions.remove(id, session.id);
    load();
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600" />
      </div>
    );
  }

  if (!treatment) return null;

  const total = treatment.total_sessions;
  const completed = treatment.completed_sessions;
  const sessionProgressPct = total ? Math.min(100, Math.round((completed / total) * 100)) : 0;
  const paidAmount = treatment.paid_amount ?? 0;
  const estimatedCost = treatment.estimated_cost;
  const paymentPct = estimatedCost && estimatedCost > 0
    ? Math.min(100, Math.round((paidAmount / estimatedCost) * 100))
    : 0;
  const remaining = estimatedCost != null ? Math.max(0, estimatedCost - paidAmount) : null;
  const canMarkCompleted = treatment.status !== 'completed' && treatment.status !== 'abandoned';

  const handleMarkTreatmentCompleted = async () => {
    if (!confirm(s.confirmMarkCompleted)) return;
    try {
      const updated = await treatments.updateStatus(id, 'completed');
      setTreatment(updated);
    } catch {
      // silencio: usuario reintenta
    }
  };

  return (
    <div className="max-w-3xl mx-auto px-4 py-8">
      {/* Header */}
      <div className="flex items-start justify-between mb-8">
        <div>
          <button
            onClick={() => router.back()}
            className="text-sm text-slate-500 hover:text-slate-700 mb-2 flex items-center gap-1"
          >
            ← Volver
          </button>
          <h1 className="text-2xl font-bold text-slate-900">{treatment.name}</h1>
          <p className="text-slate-500 text-sm mt-1">
            {treatment.customer_name && `${treatment.customer_name} · `}
            {completed}{total ? `/${total}` : ''} {s.title.toLowerCase()}
          </p>
        </div>
        <div className="flex items-center gap-2">
          {canMarkCompleted && (
            <button
              onClick={handleMarkTreatmentCompleted}
              className="border border-emerald-200 bg-emerald-50 text-emerald-700 px-3 py-2 rounded-lg text-sm font-medium hover:bg-emerald-100"
            >
              {s.markTreatmentCompleted}
            </button>
          )}
          <button
            onClick={() => setModalOpen(true)}
            className="bg-indigo-600 text-white px-4 py-2 rounded-lg text-sm font-semibold hover:bg-indigo-700 transition-colors"
          >
            {s.newSession}
          </button>
        </div>
      </div>

      {/* Resumen: Progreso + Pagos */}
      <div className="mb-8 grid gap-4 sm:grid-cols-2">
        {/* Progreso de sesiones */}
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <p className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-2">{s.progressTitle}</p>
          {total ? (
            <>
              <div className="flex items-baseline justify-between mb-2">
                <span className="text-2xl font-bold text-slate-900">{sessionProgressPct}%</span>
                <span className="text-sm text-slate-500">{completed} / {total} {s.title.toLowerCase()}</span>
              </div>
              <div className="h-2 w-full rounded-full bg-slate-100">
                <div className="h-full rounded-full bg-indigo-500 transition-all" style={{ width: `${sessionProgressPct}%` }} />
              </div>
            </>
          ) : (
            <p className="text-sm text-slate-400">{s.noTotalSessions}</p>
          )}
        </div>

        {/* Pagos */}
        <div className="rounded-xl border border-slate-200 bg-white p-5">
          <p className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-2">{s.paymentsTitle}</p>
          {estimatedCost && estimatedCost > 0 ? (
            <>
              <div className="flex items-baseline justify-between mb-2">
                <span className="text-2xl font-bold text-emerald-600">${paidAmount.toFixed(2)}</span>
                <span className="text-sm text-slate-500">/ ${estimatedCost.toFixed(2)} {treatment.currency}</span>
              </div>
              <div className="h-2 w-full rounded-full bg-slate-100">
                <div className="h-full rounded-full bg-emerald-500 transition-all" style={{ width: `${paymentPct}%` }} />
              </div>
              {remaining != null && remaining > 0 && (
                <p className="mt-2 text-xs text-slate-500">{s.remaining}: ${remaining.toFixed(2)} {treatment.currency}</p>
              )}
            </>
          ) : (
            <p className="text-sm text-slate-400">{s.noEstimatedCost}</p>
          )}
        </div>
      </div>

      {/* Sessions list */}
      <div>
        <p className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-3">
          {s.title} ({sessions.length})
        </p>

        {sessions.length === 0 ? (
          <div className="text-center py-16 text-slate-400">
            <p className="text-4xl mb-3">📋</p>
            <p>{s.noSessions}</p>
          </div>
        ) : (
          <div className="space-y-2">
            {sessions.map((session, index) => (
              <div
                key={session.id}
                className={`border rounded-xl p-4 flex items-center gap-4 ${
                  session.status === 'cancelled' ? 'opacity-60 bg-slate-50' : 'bg-white'
                }`}
              >
                {/* Número */}
                <div className={`w-9 h-9 rounded-full flex items-center justify-center text-xs font-bold shrink-0 ${
                  session.status === 'completed' ? 'bg-green-100 text-green-700' :
                  session.status === 'cancelled' ? 'bg-red-100 text-red-500' :
                  'bg-indigo-100 text-indigo-700'
                }`}>
                  {session.status === 'cancelled' ? '✕' : sessions.length - index}
                </div>

                {/* Info */}
                <div className="flex-1 min-w-0">
                  <p className={`text-sm font-medium text-slate-900 ${session.status === 'cancelled' ? 'line-through text-slate-400' : ''}`}>
                    {session.procedures_done || (session.status === 'pending' ? 'Sesión agendada' : 'Sesión completada')}
                  </p>
                  <p className="text-xs text-slate-400 mt-0.5">
                    {formatDate(session.scheduled_at)}
                    {session.duration_minutes && ` · ${session.duration_minutes} min`}
                    {session.paid_in_session && ` · $${session.paid_in_session} cobrados`}
                    {session.professional_name && ` · ${session.professional_name}`}
                  </p>
                </div>

                {/* Status + Actions */}
                <div className="flex items-center gap-2 shrink-0">
                  <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${STATUS_COLORS[session.status]}`}>
                    {(s.status as Record<string, string>)[session.status]}
                  </span>
                  {session.status === 'pending' && (
                    <>
                      <button
                        onClick={() => handleComplete(session)}
                        disabled={completing === session.id}
                        className="text-xs border border-slate-200 rounded-lg px-2 py-1 hover:bg-slate-50 disabled:opacity-50"
                      >
                        {completing === session.id ? '...' : s.actions.complete}
                      </button>
                      <button
                        onClick={() => handleCancel(session)}
                        className="text-xs text-slate-400 hover:text-slate-600"
                      >
                        {s.actions.cancel}
                      </button>
                    </>
                  )}
                  {session.status !== 'cancelled' && (
                    <button
                      onClick={() => handleDelete(session)}
                      className="text-xs text-red-400 hover:text-red-600 ml-1"
                    >
                      {s.actions.delete}
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <SessionModal
        isOpen={modalOpen}
        onClose={() => setModalOpen(false)}
        treatmentId={id}
        onSuccess={load}
      />
    </div>
  );
}
