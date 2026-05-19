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
  Plus,
  X,
  Check,
  Ban,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import type { BadgeVariant } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { customers, treatments, treatmentSessions, tasks, pipelineStages, professionals, clinicalNotes } from '@/lib/api';
import type { Customer, Treatment, TreatmentSession, Task, PipelineStage, Professional, ClinicalNoteWithDetails } from '@/lib/api';
import { SessionModal } from '@/components/sessions/SessionModal';
import { ClinicalHistorySection } from '@/components/clinical/ClinicalHistorySection';
import { ClinicalNoteModal } from '@/components/clinical/ClinicalNoteModal';
import { useTranslations, useDateLocale } from '@/lib/i18n';
import { useTenantStore } from '@/store/tenant';
import { format } from 'date-fns';

// ── Helpers ───────────────────────────────────────────────────────────────────

const TREATMENT_TYPES = [
  'ortodoncia', 'endodoncia', 'implante', 'protesis',
  'cirugia', 'periodoncia', 'estetica', 'general',
] as const;

const TREATMENT_STATUSES = [
  'proposed', 'accepted', 'in_progress', 'completed', 'abandoned',
] as const;

function treatmentStatusVariant(status: string): BadgeVariant {
  switch (status) {
    case 'accepted':    return 'primary';
    case 'in_progress': return 'warning';
    case 'completed':   return 'success';
    case 'abandoned':   return 'error';
    default:            return 'default';
  }
}

function taskStatusVariant(status: string): BadgeVariant {
  switch (status) {
    case 'in_progress': return 'primary';
    case 'completed':   return 'success';
    case 'dismissed':   return 'error';
    default:            return 'warning';
  }
}

function taskSourceVariant(source: string): BadgeVariant {
  return source === 'manual' ? 'default' : 'primary';
}

// ── Modal simple ──────────────────────────────────────────────────────────────

function Modal({ title, onClose, children }: {
  title: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-md overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-xl">
        <div className="flex items-center justify-between border-b border-neutral-100 px-5 py-4">
          <h3 className="text-sm font-semibold text-neutral-900">{title}</h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="px-5 py-4">{children}</div>
      </div>
    </div>
  );
}

// ── Formulario: Nuevo tratamiento ─────────────────────────────────────────────

interface NewTreatmentFormProps {
  customerId: string;
  profList: Professional[];
  t: ReturnType<typeof useTranslations>;
  onSaved: (tr: Treatment) => void;
  onClose: () => void;
}

function NewTreatmentForm({ customerId, profList, t, onSaved, onClose }: NewTreatmentFormProps) {
  const [name, setName]           = useState('');
  const [type, setType]           = useState('');
  const [profId, setProfId]       = useState('');
  const [sessions, setSessions]   = useState('');
  const [cost, setCost]           = useState('');
  const [notes, setNotes]         = useState('');
  const [saving, setSaving]       = useState(false);
  const [error, setError]         = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !type || !profId) return;
    setSaving(true);
    setError('');
    try {
      const tr = await treatments.create({
        customer_id:     customerId,
        professional_id: profId,
        name:            name.trim(),
        treatment_type:  type,
        total_sessions:  sessions ? Number(sessions) : undefined,
        estimated_cost:  cost ? Number(cost) : undefined,
        notes:           notes.trim() || undefined,
      });
      onSaved(tr);
    } catch {
      setError(t.clients.errorSaving);
    } finally {
      setSaving(false);
    }
  };

  const inputCls = 'w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100';
  const labelCls = 'mb-1 block text-xs font-medium text-neutral-600';

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div>
        <label className={labelCls}>{t.clients.treatmentNameLabel} *</label>
        <input className={inputCls} placeholder={t.clients.treatmentNamePlaceholder} value={name} onChange={(e) => setName(e.target.value)} required />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className={labelCls}>{t.clients.treatmentTypesLabel} *</label>
          <select className={inputCls} value={type} onChange={(e) => setType(e.target.value)} required>
            <option value="">{t.clients.selectType}</option>
            {TREATMENT_TYPES.map((tt) => (
              <option key={tt} value={tt}>
                {(t.clients.treatmentTypes as Record<string, string>)[tt] ?? tt}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className={labelCls}>{t.clients.professionalLabel} *</label>
          <select className={inputCls} value={profId} onChange={(e) => setProfId(e.target.value)} required>
            <option value="">{t.clients.selectProfessional}</option>
            {profList.map((p) => (
              <option key={p.id} value={p.id}>{p.name}</option>
            ))}
          </select>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className={labelCls}>{t.clients.totalSessionsLabel}</label>
          <input className={inputCls} type="number" min="1" placeholder="—" value={sessions} onChange={(e) => setSessions(e.target.value)} />
        </div>
        <div>
          <label className={labelCls}>{t.clients.estimatedCostLabel}</label>
          <input className={inputCls} type="number" min="0" step="0.01" placeholder="0.00" value={cost} onChange={(e) => setCost(e.target.value)} />
        </div>
      </div>

      <div>
        <label className={labelCls}>{t.clients.treatmentNotesLabel}</label>
        <textarea className={inputCls} rows={2} placeholder={t.clients.taskDescPlaceholder} value={notes} onChange={(e) => setNotes(e.target.value)} />
      </div>

      {error && <p className="text-xs text-red-500">{error}</p>}

      <div className="flex justify-end gap-2 pt-1">
        <button type="button" onClick={onClose} className="rounded-lg border border-neutral-200 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-50 transition-colors">
          {t.common.cancel}
        </button>
        <button type="submit" disabled={saving || !name.trim() || !type || !profId} className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 disabled:opacity-50 transition-colors">
          {saving ? t.clients.saving : t.common.save}
        </button>
      </div>
    </form>
  );
}

// ── Formulario: Nueva tarea ───────────────────────────────────────────────────

interface NewTaskFormProps {
  customerId: string;
  profList: Professional[];
  t: ReturnType<typeof useTranslations>;
  onSaved: (tk: Task) => void;
  onClose: () => void;
}

function NewTaskForm({ customerId, profList, t, onSaved, onClose }: NewTaskFormProps) {
  const [title, setTitle]     = useState('');
  const [desc, setDesc]       = useState('');
  const [dueAt, setDueAt]     = useState('');
  const [assignTo, setAssignTo] = useState('');
  const [saving, setSaving]   = useState(false);
  const [error, setError]     = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) return;
    setSaving(true);
    setError('');
    try {
      const tk = await tasks.create({
        title:       title.trim(),
        description: desc.trim() || undefined,
        customer_id: customerId,
        assigned_to: assignTo || undefined,
        due_at:      dueAt ? new Date(dueAt).toISOString() : undefined,
      });
      onSaved(tk);
    } catch {
      setError(t.clients.errorSaving);
    } finally {
      setSaving(false);
    }
  };

  const inputCls = 'w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100';
  const labelCls = 'mb-1 block text-xs font-medium text-neutral-600';

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div>
        <label className={labelCls}>{t.clients.taskTitleLabel} *</label>
        <input className={inputCls} placeholder={t.clients.taskTitlePlaceholder} value={title} onChange={(e) => setTitle(e.target.value)} required />
      </div>

      <div>
        <label className={labelCls}>{t.clients.taskDescLabel}</label>
        <textarea className={inputCls} rows={2} placeholder={t.clients.taskDescPlaceholder} value={desc} onChange={(e) => setDesc(e.target.value)} />
      </div>

      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className={labelCls}>{t.clients.taskDueLabel}</label>
          <input className={inputCls} type="date" value={dueAt} onChange={(e) => setDueAt(e.target.value)} />
        </div>
        <div>
          <label className={labelCls}>{t.clients.assignToLabel}</label>
          <select className={inputCls} value={assignTo} onChange={(e) => setAssignTo(e.target.value)}>
            <option value="">—</option>
            {profList.map((p) => (
              <option key={p.id} value={p.id}>{p.name}</option>
            ))}
          </select>
        </div>
      </div>

      {error && <p className="text-xs text-red-500">{error}</p>}

      <div className="flex justify-end gap-2 pt-1">
        <button type="button" onClick={onClose} className="rounded-lg border border-neutral-200 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-50 transition-colors">
          {t.common.cancel}
        </button>
        <button type="submit" disabled={saving || !title.trim()} className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 disabled:opacity-50 transition-colors">
          {saving ? t.clients.saving : t.common.save}
        </button>
      </div>
    </form>
  );
}

// ── Sección de Tratamientos ───────────────────────────────────────────────────

interface TreatmentsProps {
  items: Treatment[];
  t: ReturnType<typeof useTranslations>;
  dateLocale: ReturnType<typeof useDateLocale>;
  onStatusChange: (id: string, status: string) => Promise<void>;
  sessionsByTreatment: Record<string, TreatmentSession[]>;
  onNewSession: (treatmentId: string) => void;
}

function TreatmentsSection({ items, t, dateLocale, onStatusChange, sessionsByTreatment, onNewSession }: TreatmentsProps) {
  const [changing, setChanging] = useState<string | null>(null);

  const handleStatus = async (id: string, status: string) => {
    setChanging(id);
    try { await onStatusChange(id, status); } finally { setChanging(null); }
  };

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
                <p className="mt-0.5 text-xs text-neutral-500">
                  {(t.clients.treatmentTypes as Record<string, string>)[tr.treatment_type] ?? tr.treatment_type}
                </p>
              </div>

              <div className="flex flex-shrink-0 items-center gap-2">
                <button
                  type="button"
                  onClick={() => onNewSession(tr.id)}
                  className="rounded-lg border border-indigo-200 bg-indigo-50 px-2 py-1 text-xs font-medium text-indigo-600 hover:bg-indigo-100 transition-colors"
                >
                  + Sesión
                </button>
                <Badge variant={treatmentStatusVariant(tr.status)}>{statusLabel}</Badge>
                {/* Cambiar estado */}
                <select
                  value={tr.status}
                  disabled={changing === tr.id}
                  onChange={(e) => handleStatus(tr.id, e.target.value)}
                  className="rounded-md border border-neutral-200 bg-white px-2 py-1 text-xs text-neutral-600 focus:border-primary-400 focus:outline-none disabled:opacity-50"
                >
                  {TREATMENT_STATUSES.map((s) => (
                    <option key={s} value={s}>
                      {(t.clients.treatmentStatus as Record<string, string>)[s] ?? s}
                    </option>
                  ))}
                </select>
                {changing === tr.id && <Spinner size="sm" />}
              </div>
            </div>

            {total > 0 && (
              <div className="mt-3">
                <div className="mb-1 flex items-center justify-between text-xs text-neutral-500">
                  <span>{t.clients.sessions.replace('{done}', String(done)).replace('{total}', String(total))}</span>
                  <span>{pct}%</span>
                </div>
                <div className="h-1.5 w-full overflow-hidden rounded-full bg-neutral-100">
                  <div className="h-full rounded-full bg-primary-500 transition-all" style={{ width: `${pct}%` }} />
                </div>
              </div>
            )}

            {tr.estimated_cost != null && tr.estimated_cost > 0 && (
              <p className="mt-2 flex items-center gap-1 text-xs text-neutral-500">
                <DollarSign className="h-3 w-3" />
                {tr.estimated_cost.toFixed(2)} {tr.currency ?? 'USD'}
              </p>
            )}

            {/* Últimas sesiones */}
            {(() => {
              const trSessions = (sessionsByTreatment[tr.id] ?? []).slice(0, 3);
              if (trSessions.length === 0) return null;
              return (
                <div className="border-t border-slate-100 pt-3 mt-3">
                  <p className="text-xs font-semibold text-slate-400 uppercase tracking-wide mb-2">
                    Últimas sesiones
                  </p>
                  <div className="space-y-1">
                    {trSessions.map(sess => (
                      <div key={sess.id} className="flex items-center justify-between text-xs">
                        <div className="flex items-center gap-2">
                          <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${
                            sess.status === 'completed' ? 'bg-green-500' :
                            sess.status === 'cancelled' ? 'bg-red-400' : 'bg-violet-400'
                          }`} />
                          <span className="text-neutral-700 truncate max-w-[180px]">
                            {sess.procedures_done || (sess.status === 'pending' ? 'Sesión agendada' : 'Sesión completada')}
                          </span>
                        </div>
                        <span className="text-neutral-400 shrink-0 ml-2">
                          {new Date(sess.scheduled_at).toLocaleDateString('es', { day: 'numeric', month: 'short' })}
                        </span>
                      </div>
                    ))}
                  </div>
                  <a
                    href={`/dashboard/treatments/${tr.id}`}
                    className="text-xs text-indigo-600 hover:text-indigo-700 mt-2 inline-block"
                  >
                    Ver todas las sesiones →
                  </a>
                </div>
              );
            })()}
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
  onComplete: (id: string) => Promise<void>;
  onDismiss: (id: string) => Promise<void>;
}

function TasksSection({ items, t, dateLocale, onComplete, onDismiss }: TasksProps) {
  const [showCompleted, setShowCompleted] = useState(false);
  const [acting, setActing] = useState<string | null>(null);

  const active    = items.filter((tk) => tk.status === 'pending' || tk.status === 'in_progress');
  const completed = items.filter((tk) => tk.status === 'completed' || tk.status === 'dismissed');

  const handleComplete = async (id: string) => {
    setActing(id);
    try { await onComplete(id); } finally { setActing(null); }
  };

  const handleDismiss = async (id: string) => {
    setActing(id);
    try { await onDismiss(id); } finally { setActing(null); }
  };

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
    const isActive    = tk.status === 'pending' || tk.status === 'in_progress';
    const isActing    = acting === tk.id;

    return (
      <div key={tk.id} className="flex items-start gap-3 px-4 py-3">
        <div className="flex-1 min-w-0">
          <p className="truncate text-sm font-medium text-neutral-900">{tk.title}</p>
          {tk.due_at && (
            <p className="mt-0.5 text-xs text-neutral-500">
              {t.clients.due.replace('{date}', format(new Date(tk.due_at), 'd MMM yyyy', { locale: dateLocale }))}
            </p>
          )}
          {tk.description && (
            <p className="mt-0.5 text-xs text-neutral-400 truncate">{tk.description}</p>
          )}
        </div>
        <div className="flex flex-shrink-0 items-center gap-1.5">
          <Badge variant={taskSourceVariant(tk.source)}>{sourceLabel}</Badge>
          <Badge variant={taskStatusVariant(tk.status)}>{statusLabel}</Badge>

          {isActive && (
            <>
              <button
                type="button"
                onClick={() => handleComplete(tk.id)}
                disabled={isActing}
                title={t.clients.complete}
                className="rounded-md p-1 text-emerald-600 hover:bg-emerald-50 disabled:opacity-50 transition-colors"
              >
                {isActing ? <Spinner size="sm" /> : <Check className="h-3.5 w-3.5" />}
              </button>
              <button
                type="button"
                onClick={() => handleDismiss(tk.id)}
                disabled={isActing}
                title={t.clients.dismiss}
                className="rounded-md p-1 text-neutral-400 hover:bg-neutral-100 disabled:opacity-50 transition-colors"
              >
                <Ban className="h-3.5 w-3.5" />
              </button>
            </>
          )}
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
            {showCompleted ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
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
  const hasDental  = useTenantStore((s) => s.tenant?.enabled_modules?.includes('dental') ?? false);

  const [customer,             setCustomer]             = useState<Customer | null>(null);
  const [treatList,            setTreatList]            = useState<Treatment[]>([]);
  const [taskList,             setTaskList]             = useState<Task[]>([]);
  const [stages,               setStages]               = useState<PipelineStage[]>([]);
  const [profList,             setProfList]             = useState<Professional[]>([]);
  const [loading,              setLoading]              = useState(true);
  const [error,                setError]                = useState<string | null>(null);
  const [stageLoading,         setStageLoading]         = useState(false);
  const [showTreatForm,        setShowTreatForm]        = useState(false);
  const [showTaskForm,         setShowTaskForm]         = useState(false);
  const [sessionsByTreatment,  setSessionsByTreatment]  = useState<Record<string, TreatmentSession[]>>({});
  const [sessionModalTreatment, setSessionModalTreatment] = useState<string | null>(null);
  const [clinicalNoteList,       setClinicalNoteList]       = useState<ClinicalNoteWithDetails[]>([]);
  const [showClinicalNoteForm,   setShowClinicalNoteForm]   = useState(false);
  const [activeTab,              setActiveTab]              = useState<'treatments' | 'tasks' | 'clinical'>(hasDental ? 'treatments' : 'tasks');

  useEffect(() => {
    if (!hasDental && (activeTab === 'treatments' || activeTab === 'clinical')) {
      setActiveTab('tasks');
    }
  }, [hasDental, activeTab]);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [cust, trRes, tkRes, stRes, prRes, cnRes] = await Promise.all([
        customers.getById(id),
        hasDental ? treatments.list({ customer_id: id }) : Promise.resolve({ data: [], total: 0 }),
        tasks.list({ customer_id: id }),
        pipelineStages.list(),
        professionals.list(),
        hasDental ? clinicalNotes.listByCustomer(id).catch(() => ({ data: [], total: 0 })) : Promise.resolve({ data: [], total: 0 }),
      ]);
      setCustomer(cust);
      setTreatList(trRes.data ?? []);
      setTaskList(tkRes.data ?? []);
      setStages(stRes.data ?? []);
      setProfList(prRes.data ?? []);
      setClinicalNoteList(cnRes.data ?? []);
    } catch {
      setError(t.clients.notFound);
    } finally {
      setLoading(false);
    }
  }, [id, hasDental, t.clients.notFound]);

  useEffect(() => { load(); }, [load]);

  // Cargar sesiones para todos los tratamientos cuando cambia la lista
  useEffect(() => {
    if (treatList.length === 0) return;
    Promise.all(
      treatList.map(tr =>
        treatmentSessions.list(tr.id)
          .then(res => ({ id: tr.id, data: res.data }))
          .catch(() => ({ id: tr.id, data: [] }))
      )
    ).then(results => {
      const map: Record<string, TreatmentSession[]> = {};
      for (const r of results) {
        map[r.id] = r.data;
      }
      setSessionsByTreatment(map);
    });
  }, [treatList]);

  const handleStageChange = useCallback(async (stageId: string) => {
    if (!customer) return;
    setStageLoading(true);
    try {
      const updated = await customers.updateStage(id, stageId === '' ? null : stageId);
      setCustomer(updated);
    } catch {
      // silently revert
    } finally {
      setStageLoading(false);
    }
  }, [customer, id]);

  const handleTreatmentStatusChange = useCallback(async (treatId: string, status: string) => {
    const updated = await treatments.updateStatus(treatId, status);
    setTreatList((prev) => prev.map((tr) => tr.id === treatId ? updated : tr));
  }, []);

  const handleTaskComplete = useCallback(async (taskId: string) => {
    const updated = await tasks.complete(taskId);
    setTaskList((prev) => prev.map((tk) => tk.id === taskId ? updated : tk));
  }, []);

  const handleTaskDismiss = useCallback(async (taskId: string) => {
    const updated = await tasks.dismiss(taskId);
    setTaskList((prev) => prev.map((tk) => tk.id === taskId ? updated : tk));
  }, []);

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
          onClick={() => router.push('/dashboard/clients')}
          className="rounded-lg border border-neutral-200 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-50 transition-colors"
        >
          {t.clients.back}
        </button>
      </div>
    );
  }

  const sourceLabel    = customer.acquisition_source
    ? ((t.clients.source as Record<string, string>)[customer.acquisition_source] ?? customer.acquisition_source)
    : null;
  const currentStage   = stages.find((s) => s.id === customer.stage_id);

  const tabsConfig = [
    ...(hasDental
      ? [{ key: 'treatments' as const, label: t.clients.treatments, count: treatList.length }]
      : []),
    { key: 'tasks' as const, label: t.clients.tasks, count: taskList.filter(tk => tk.status === 'pending' || tk.status === 'in_progress').length },
    ...(hasDental
      ? [{ key: 'clinical' as const, label: t.clients.clinicalHistory, count: clinicalNoteList.length }]
      : []),
  ];

  return (
    <div className="mx-auto max-w-7xl p-6">
      {/* ── Top bar: Botón volver ──────────────────────────────────────────── */}
      <button
        type="button"
        onClick={() => router.push('/dashboard/clients')}
        className="mb-6 flex items-center gap-1.5 text-sm text-neutral-500 hover:text-neutral-800 transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        {t.clients.back}
      </button>

      {/* ── Two-column CRM layout ──────────────────────────────────────────── */}
      <div className="lg:grid lg:grid-cols-[340px_1fr] lg:gap-6">

        {/* ── LEFT COLUMN ────────────────────────────────────────────────────── */}
        <div className="lg:sticky lg:top-6 lg:self-start space-y-4 mb-6 lg:mb-0">

          {/* Profile card */}
          <div className="overflow-hidden rounded-xl border border-neutral-200 bg-white">
            <div className="flex flex-col items-center p-6">
              <div className="flex h-20 w-20 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 text-3xl font-bold text-primary-700">
                {customer.name.charAt(0).toUpperCase()}
              </div>
              <div className="mt-4 text-center">
                <h1 className="text-lg font-semibold text-neutral-900">{customer.name}</h1>
                {sourceLabel && <Badge variant="default" className="mt-1">{sourceLabel}</Badge>}
              </div>
              <div className="mt-4 flex flex-col gap-2 w-full">
                {customer.phone && (
                  <span className="flex items-center gap-2 text-sm text-neutral-600">
                    <Phone className="h-4 w-4 text-neutral-400 flex-shrink-0" />
                    {customer.phone}
                  </span>
                )}
                {customer.email && (
                  <span className="flex items-center gap-2 text-sm text-neutral-600">
                    <Mail className="h-4 w-4 text-neutral-400 flex-shrink-0" />
                    {customer.email}
                  </span>
                )}
              </div>
            </div>
          </div>

          {/* KPIs — vertical list on desktop, horizontal on mobile */}
          <div className="overflow-hidden rounded-xl border border-neutral-200 bg-white">
            {/* Mobile: horizontal row */}
            <div className="flex flex-wrap divide-x divide-neutral-100 lg:hidden">
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
            {/* Desktop: vertical list with icons */}
            <div className="hidden lg:block divide-y divide-neutral-100">
              <div className="flex items-center gap-3 px-4 py-3">
                <Calendar className="h-4 w-4 text-neutral-400 flex-shrink-0" />
                <div className="flex-1 min-w-0">
                  <span className="text-xs text-neutral-500">{t.clients.totalVisits}</span>
                </div>
                <span className="text-sm font-semibold text-neutral-900">{customer.total_visits}</span>
              </div>
              {customer.last_visit_at && (
                <div className="flex items-center gap-3 px-4 py-3">
                  <Calendar className="h-4 w-4 text-neutral-400 flex-shrink-0" />
                  <div className="flex-1 min-w-0">
                    <span className="text-xs text-neutral-500">{t.clients.lastVisit}</span>
                  </div>
                  <span className="text-sm font-medium text-neutral-700">
                    {format(new Date(customer.last_visit_at), 'd MMM yyyy', { locale: dateLocale })}
                  </span>
                </div>
              )}
              {customer.lifetime_value != null && customer.lifetime_value > 0 && (
                <div className="flex items-center gap-3 px-4 py-3">
                  <DollarSign className="h-4 w-4 text-neutral-400 flex-shrink-0" />
                  <div className="flex-1 min-w-0">
                    <span className="text-xs text-neutral-500">{t.clients.lifetimeValue}</span>
                  </div>
                  <span className="text-sm font-medium text-neutral-700">
                    ${customer.lifetime_value.toFixed(2)}
                  </span>
                </div>
              )}
            </div>
          </div>

          {/* Pipeline Stage */}
          {stages.length > 0 && (
            <div className="overflow-hidden rounded-xl border border-neutral-200 bg-white">
              <div className="border-b border-neutral-100 px-4 py-3">
                <h2 className="text-sm font-semibold text-neutral-700">{t.clients.pipelineStage}</h2>
              </div>
              <div className="flex items-center gap-3 px-4 py-3">
                {currentStage && (
                  <span className="h-3 w-3 rounded-full flex-shrink-0" style={{ backgroundColor: currentStage.color }} />
                )}
                <select
                  value={customer.stage_id ?? ''}
                  onChange={(e) => handleStageChange(e.target.value)}
                  disabled={stageLoading}
                  className="flex-1 rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100 disabled:opacity-50"
                >
                  <option value="">{t.clients.noStage}</option>
                  {stages.map((s) => (
                    <option key={s.id} value={s.id}>{s.name}</option>
                  ))}
                </select>
                {stageLoading && <Spinner size="sm" />}
              </div>
            </div>
          )}

          {/* Notas */}
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

        {/* ── RIGHT COLUMN ───────────────────────────────────────────────────── */}
        <div>
          {/* Tab bar */}
          <div className="sticky top-0 z-10 bg-white rounded-t-xl border border-neutral-200 border-b-0">
            <div className="flex items-center justify-between">
              <div className="flex border-b border-neutral-200 flex-1">
                {tabsConfig.map((tab) => (
                  <button
                    key={tab.key}
                    type="button"
                    onClick={() => setActiveTab(tab.key)}
                    className={`relative px-4 py-3 text-sm font-medium transition-colors ${
                      activeTab === tab.key
                        ? 'border-b-2 border-primary-600 text-primary-600 font-semibold'
                        : 'text-neutral-500 hover:text-neutral-700'
                    }`}
                  >
                    {tab.label}
                    {tab.count > 0 && (
                      <span className={`ml-1.5 inline-flex items-center justify-center rounded-full px-1.5 py-0.5 text-xs font-medium ${
                        activeTab === tab.key
                          ? 'bg-primary-100 text-primary-700'
                          : 'bg-neutral-100 text-neutral-500'
                      }`}>
                        {tab.count}
                      </span>
                    )}
                  </button>
                ))}
              </div>
              {/* "+ New" button for the active tab */}
              <div className="px-3 flex-shrink-0">
                {activeTab === 'treatments' && hasDental && (
                  <button
                    type="button"
                    onClick={() => setShowTreatForm(true)}
                    className="flex items-center gap-1 rounded-lg border border-neutral-200 px-2.5 py-1.5 text-xs font-medium text-neutral-700 hover:bg-neutral-50 transition-colors"
                  >
                    <Plus className="h-3.5 w-3.5" />
                    {t.clients.newTreatment}
                  </button>
                )}
                {activeTab === 'tasks' && (
                  <button
                    type="button"
                    onClick={() => setShowTaskForm(true)}
                    className="flex items-center gap-1 rounded-lg border border-neutral-200 px-2.5 py-1.5 text-xs font-medium text-neutral-700 hover:bg-neutral-50 transition-colors"
                  >
                    <Plus className="h-3.5 w-3.5" />
                    {t.clients.newTask}
                  </button>
                )}
                {activeTab === 'clinical' && hasDental && (
                  <button
                    type="button"
                    onClick={() => setShowClinicalNoteForm(true)}
                    className="flex items-center gap-1 rounded-lg border border-neutral-200 px-2.5 py-1.5 text-xs font-medium text-neutral-700 hover:bg-neutral-50 transition-colors"
                  >
                    <Plus className="h-3.5 w-3.5" />
                    {t.clients.newClinicalNote}
                  </button>
                )}
              </div>
            </div>
          </div>

          {/* Tab content */}
          <div className="overflow-hidden rounded-b-xl border border-neutral-200 border-t-0 bg-white">
            {activeTab === 'treatments' && hasDental && (
              <TreatmentsSection
                items={treatList}
                t={t}
                dateLocale={dateLocale}
                onStatusChange={handleTreatmentStatusChange}
                sessionsByTreatment={sessionsByTreatment}
                onNewSession={(treatmentId) => setSessionModalTreatment(treatmentId)}
              />
            )}

            {activeTab === 'tasks' && (
              <TasksSection
                items={taskList}
                t={t}
                dateLocale={dateLocale}
                onComplete={handleTaskComplete}
                onDismiss={handleTaskDismiss}
              />
            )}

            {activeTab === 'clinical' && hasDental && (
              <ClinicalHistorySection
                customerId={id}
                professionalId={profList[0]?.id ?? ''}
                profList={profList}
                notes={clinicalNoteList}
                onNotesChange={setClinicalNoteList}
              />
            )}
          </div>
        </div>
      </div>

      {/* ── Modales ──────────────────────────────────────────────────────────── */}
      {showTreatForm && (
        <Modal title={t.clients.newTreatment} onClose={() => setShowTreatForm(false)}>
          <NewTreatmentForm
            customerId={id}
            profList={profList}
            t={t}
            onSaved={(tr) => {
              setTreatList((prev) => [tr, ...prev]);
              setShowTreatForm(false);
            }}
            onClose={() => setShowTreatForm(false)}
          />
        </Modal>
      )}

      {showTaskForm && (
        <Modal title={t.clients.newTask} onClose={() => setShowTaskForm(false)}>
          <NewTaskForm
            customerId={id}
            profList={profList}
            t={t}
            onSaved={(tk) => {
              setTaskList((prev) => [tk, ...prev]);
              setShowTaskForm(false);
            }}
            onClose={() => setShowTaskForm(false)}
          />
        </Modal>
      )}

      {showClinicalNoteForm && (
        <ClinicalNoteModal
          customerId={id}
          professionalId={profList[0]?.id ?? ''}
          onSaved={async () => {
            setShowClinicalNoteForm(false);
            try {
              const res = await clinicalNotes.listByCustomer(id);
              setClinicalNoteList(res.data ?? []);
            } catch { /* noop */ }
          }}
          onClose={() => setShowClinicalNoteForm(false)}
        />
      )}

      {sessionModalTreatment && (
        <SessionModal
          isOpen={true}
          onClose={() => setSessionModalTreatment(null)}
          treatmentId={sessionModalTreatment}
          onSuccess={() => {
            // Recargar sesiones del tratamiento afectado
            treatmentSessions.list(sessionModalTreatment)
              .then(res => setSessionsByTreatment(prev => ({
                ...prev,
                [sessionModalTreatment]: res.data,
              })))
              .catch(() => {});
          }}
        />
      )}
    </div>
  );
}
