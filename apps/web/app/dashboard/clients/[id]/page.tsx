'use client';

import { useState, useEffect, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import {
  ArrowLeft,
  Phone,
  Mail,
  Calendar,
  DollarSign,
  ChevronDown,
  ChevronUp,
  AlertCircle,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import type { BadgeVariant } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { customers, treatments, tasks, pipelineStages } from '@/lib/api';
import type { Customer, Treatment, Task, PipelineStage } from '@/lib/api';
import { useTranslations, useDateLocale } from '@/lib/i18n';
import { format } from 'date-fns';

// ── Helpers ───────────────────────────────────────────────────────────────────

function treatmentStatusVariant(status: string): BadgeVariant {
  switch (status) {
    case 'accepted':    return 'primary';
    case 'in_progress': return 'warning';
    case 'completed':   return 'success';
    case 'abandoned':   return 'error';
    default:            return 'default'; // proposed
  }
}

function taskStatusVariant(status: string): BadgeVariant {
  switch (status) {
    case 'in_progress': return 'primary';
    case 'completed':   return 'success';
    case 'dismissed':   return 'error';
    default:            return 'warning'; // pending
  }
}

function taskSourceVariant(source: string): BadgeVariant {
  return source === 'manual' ? 'default' : 'primary';
}

// ── Sección de Tratamientos ───────────────────────────────────────────────────

interface TreatmentsProps {
  items: Treatment[];
  t: ReturnType<typeof useTranslations>;
  dateLocale: ReturnType<typeof useDateLocale>;
}

function TreatmentsSection({ items, t, dateLocale }: TreatmentsProps) {
  if (items.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-10 text-center">
        <AlertCircle className="mb-3 h-10 w-10 text-neutral-200" />
        <p className="text-sm font-medium text-neutral-500">{t.clients.noTreatments}</p>
        <p className="mt-1 text-xs text-neutral-400">{t.clients.noTreatmentsDesc}</p>
      </div>
    );
  }

  return (
    <div className="divide-y divide-neutral-100">
      {items.map((tr) => {
        const done  = tr.completed_sessions ?? 0;
        const total = tr.total_sessions ?? 0;
        const pct   = total > 0 ? Math.round((done / total) * 100) : 0;
        const statusLabel = (t.clients.treatmentStatus as Record<string, string>)[tr.status] ?? tr.status;

        return (
          <div key={tr.id} className="px-4 py-4">
            <div className="flex items-start justify-between gap-3">
              <div className="flex-1 min-w-0">
                <p className="truncate font-medium text-neutral-900">{tr.name}</p>
                <p className="mt-0.5 text-xs text-neutral-500">{tr.treatment_type}</p>
              </div>
              <Badge variant={treatmentStatusVariant(tr.status)}>{statusLabel}</Badge>
            </div>

            {total > 0 && (
              <div className="mt-3">
                <div className="mb-1 flex items-center justify-between text-xs text-neutral-500">
                  <span>
                    {t.clients.sessions
                      .replace('{done}', String(done))
                      .replace('{total}', String(total))}
                  </span>
                  <span>{pct}%</span>
                </div>
                <div className="h-1.5 w-full overflow-hidden rounded-full bg-neutral-100">
                  <div
                    className="h-full rounded-full bg-primary-500 transition-all"
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>
            )}

            {tr.estimated_cost != null && tr.estimated_cost > 0 && (
              <p className="mt-2 flex items-center gap-1 text-xs text-neutral-500">
                <DollarSign className="h-3 w-3" />
                {tr.estimated_cost.toFixed(2)} {tr.currency ?? 'USD'}
              </p>
            )}
          </div>
        );
      })}
    </div>
  );
}

// ── Sección de Tareas ─────────────────────────────────────────────────────────

interface TasksProps {
  items: Task[];
  t: ReturnType<typeof useTranslations>;
  dateLocale: ReturnType<typeof useDateLocale>;
}

function TasksSection({ items, t, dateLocale }: TasksProps) {
  const [showCompleted, setShowCompleted] = useState(false);

  const active    = items.filter((tk) => tk.status === 'pending' || tk.status === 'in_progress');
  const completed = items.filter((tk) => tk.status === 'completed' || tk.status === 'dismissed');

  if (items.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-10 text-center">
        <AlertCircle className="mb-3 h-10 w-10 text-neutral-200" />
        <p className="text-sm font-medium text-neutral-500">{t.clients.noTasks}</p>
        <p className="mt-1 text-xs text-neutral-400">{t.clients.noTasksDesc}</p>
      </div>
    );
  }

  const renderTask = (tk: Task) => {
    const statusLabel = (t.clients.taskStatus as Record<string, string>)[tk.status] ?? tk.status;
    const sourceLabel = (t.clients.taskSource as Record<string, string>)[tk.source] ?? tk.source;

    return (
      <div key={tk.id} className="flex items-start gap-3 px-4 py-3">
        <div className="flex-1 min-w-0">
          <p className="truncate text-sm font-medium text-neutral-900">{tk.title}</p>
          {tk.due_at && (
            <p className="mt-0.5 text-xs text-neutral-500">
              {t.clients.due.replace('{date}', format(new Date(tk.due_at), 'd MMM yyyy', { locale: dateLocale }))}
            </p>
          )}
        </div>
        <div className="flex flex-shrink-0 items-center gap-1.5">
          <Badge variant={taskSourceVariant(tk.source)}>{sourceLabel}</Badge>
          <Badge variant={taskStatusVariant(tk.status)}>{statusLabel}</Badge>
        </div>
      </div>
    );
  };

  return (
    <div>
      <div className="divide-y divide-neutral-100">
        {active.map(renderTask)}
      </div>

      {completed.length > 0 && (
        <div className="border-t border-neutral-100">
          <button
            type="button"
            onClick={() => setShowCompleted((v) => !v)}
            className="flex w-full items-center justify-between px-4 py-3 text-xs font-medium text-neutral-500 hover:bg-neutral-50 transition-colors"
          >
            <span>
              {showCompleted
                ? t.clients.hideCompleted
                : t.clients.showCompleted.replace('{n}', String(completed.length))}
            </span>
            {showCompleted ? (
              <ChevronUp className="h-3.5 w-3.5" />
            ) : (
              <ChevronDown className="h-3.5 w-3.5" />
            )}
          </button>
          {showCompleted && (
            <div className="divide-y divide-neutral-100 bg-neutral-50">
              {completed.map(renderTask)}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// ── Página principal ──────────────────────────────────────────────────────────

export default function CustomerProfilePage() {
  const t          = useTranslations();
  const dateLocale = useDateLocale();
  const router     = useRouter();
  const { id }     = useParams<{ id: string }>();

  const [customer,   setCustomer]   = useState<Customer | null>(null);
  const [treatList,  setTreatList]  = useState<Treatment[]>([]);
  const [taskList,   setTaskList]   = useState<Task[]>([]);
  const [stages,     setStages]     = useState<PipelineStage[]>([]);
  const [loading,    setLoading]    = useState(true);
  const [error,      setError]      = useState<string | null>(null);
  const [stageLoading, setStageLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [cust, trRes, tkRes, stRes] = await Promise.all([
        customers.getById(id),
        treatments.list({ customer_id: id }),
        tasks.list({ customer_id: id }),
        pipelineStages.list(),
      ]);
      setCustomer(cust);
      setTreatList(trRes.data ?? []);
      setTaskList(tkRes.data ?? []);
      setStages(stRes.data ?? []);
    } catch {
      setError(t.clients.notFound);
    } finally {
      setLoading(false);
    }
  }, [id, t.clients.notFound]);

  useEffect(() => { load(); }, [load]);

  const handleStageChange = useCallback(
    async (stageId: string) => {
      if (!customer) return;
      setStageLoading(true);
      try {
        const updated = await customers.updateStage(id, stageId === '' ? null : stageId);
        setCustomer(updated);
      } catch {
        // Silently revert — no toasts en esta versión
      } finally {
        setStageLoading(false);
      }
    },
    [customer, id],
  );

  if (loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <Spinner size="lg" />
          <p className="text-sm text-neutral-500">{t.clients.loading}</p>
        </div>
      </div>
    );
  }

  if (error || !customer) {
    return (
      <div className="flex min-h-[60vh] flex-col items-center justify-center gap-4 p-6">
        <AlertCircle className="h-12 w-12 text-neutral-300" />
        <p className="text-sm font-medium text-neutral-600">{error ?? t.clients.notFound}</p>
        <button
          type="button"
          onClick={() => router.push('/clients')}
          className="rounded-lg border border-neutral-200 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-50 transition-colors"
        >
          {t.clients.back}
        </button>
      </div>
    );
  }

  const sourceLabel = customer.acquisition_source
    ? ((t.clients.source as Record<string, string>)[customer.acquisition_source] ?? customer.acquisition_source)
    : null;

  const currentStage = stages.find((s) => s.id === customer.stage_id);

  return (
    <div className="mx-auto max-w-3xl p-6">
      {/* Botón volver */}
      <button
        type="button"
        onClick={() => router.push('/clients')}
        className="mb-6 flex items-center gap-1.5 text-sm text-neutral-500 hover:text-neutral-800 transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        {t.clients.back}
      </button>

      {/* ── Header ──────────────────────────────────────────────────────────── */}
      <div className="mb-6 overflow-hidden rounded-xl border border-neutral-200 bg-white">
        <div className="flex flex-col gap-4 p-6 sm:flex-row sm:items-start">
          {/* Avatar */}
          <div className="flex h-16 w-16 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 text-2xl font-bold text-primary-700">
            {customer.name.charAt(0).toUpperCase()}
          </div>

          {/* Info principal */}
          <div className="flex-1 min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-lg font-semibold text-neutral-900">{customer.name}</h1>
              {sourceLabel && (
                <Badge variant="default">{sourceLabel}</Badge>
              )}
            </div>

            <div className="mt-2 flex flex-col gap-1">
              {customer.phone && (
                <span className="flex items-center gap-1.5 text-sm text-neutral-600">
                  <Phone className="h-3.5 w-3.5 text-neutral-400" />
                  {customer.phone}
                </span>
              )}
              {customer.email && (
                <span className="flex items-center gap-1.5 text-sm text-neutral-600">
                  <Mail className="h-3.5 w-3.5 text-neutral-400" />
                  {customer.email}
                </span>
              )}
            </div>
          </div>
        </div>

        {/* KPIs */}
        <div className="flex flex-wrap divide-x divide-neutral-100 border-t border-neutral-100">
          <div className="flex flex-1 flex-col items-center px-4 py-3">
            <span className="text-xs text-neutral-500">{t.clients.totalVisits}</span>
            <span className="mt-0.5 text-xl font-semibold text-neutral-900">{customer.total_visits}</span>
          </div>

          {customer.last_visit_at && (
            <div className="flex flex-1 flex-col items-center px-4 py-3">
              <span className="text-xs text-neutral-500">{t.clients.lastVisit}</span>
              <span className="mt-0.5 flex items-center gap-1 text-sm font-medium text-neutral-700">
                <Calendar className="h-3.5 w-3.5 text-neutral-400" />
                {format(new Date(customer.last_visit_at), 'd MMM yyyy', { locale: dateLocale })}
              </span>
            </div>
          )}

          {customer.lifetime_value != null && customer.lifetime_value > 0 && (
            <div className="flex flex-1 flex-col items-center px-4 py-3">
              <span className="text-xs text-neutral-500">{t.clients.lifetimeValue}</span>
              <span className="mt-0.5 flex items-center gap-0.5 text-sm font-medium text-neutral-700">
                <DollarSign className="h-3.5 w-3.5 text-neutral-400" />
                {customer.lifetime_value.toFixed(2)}
              </span>
            </div>
          )}
        </div>
      </div>

      {/* ── Pipeline Stage ───────────────────────────────────────────────────── */}
      {stages.length > 0 && (
        <div className="mb-4 overflow-hidden rounded-xl border border-neutral-200 bg-white">
          <div className="border-b border-neutral-100 px-4 py-3">
            <h2 className="text-sm font-semibold text-neutral-700">{t.clients.pipelineStage}</h2>
          </div>
          <div className="flex items-center gap-3 px-4 py-3">
            {currentStage && (
              <span
                className="h-3 w-3 rounded-full flex-shrink-0"
                style={{ backgroundColor: currentStage.color }}
              />
            )}
            <select
              value={customer.stage_id ?? ''}
              onChange={(e) => handleStageChange(e.target.value)}
              disabled={stageLoading}
              className="flex-1 rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100 disabled:opacity-50"
            >
              <option value="">{t.clients.noStage}</option>
              {stages.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
            {stageLoading && <Spinner size="sm" />}
          </div>
        </div>
      )}

      {/* ── Tratamientos ─────────────────────────────────────────────────────── */}
      <div className="mb-4 overflow-hidden rounded-xl border border-neutral-200 bg-white">
        <div className="border-b border-neutral-100 px-4 py-3">
          <h2 className="text-sm font-semibold text-neutral-700">{t.clients.treatments}</h2>
        </div>
        <TreatmentsSection items={treatList} t={t} dateLocale={dateLocale} />
      </div>

      {/* ── Tareas ───────────────────────────────────────────────────────────── */}
      <div className="mb-4 overflow-hidden rounded-xl border border-neutral-200 bg-white">
        <div className="border-b border-neutral-100 px-4 py-3">
          <h2 className="text-sm font-semibold text-neutral-700">{t.clients.tasks}</h2>
        </div>
        <TasksSection items={taskList} t={t} dateLocale={dateLocale} />
      </div>

      {/* ── Notas ────────────────────────────────────────────────────────────── */}
      <div className="overflow-hidden rounded-xl border border-neutral-200 bg-white">
        <div className="border-b border-neutral-100 px-4 py-3">
          <h2 className="text-sm font-semibold text-neutral-700">{t.clients.notes}</h2>
        </div>
        <div className="px-4 py-4">
          {customer.notes ? (
            <p className="whitespace-pre-wrap text-sm text-neutral-700">{customer.notes}</p>
          ) : (
            <p className="text-sm text-neutral-400">{t.clients.noNotes}</p>
          )}
        </div>
      </div>
    </div>
  );
}
