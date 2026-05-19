'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  CheckSquare, Plus, Check, XCircle, Clock, AlertTriangle, X, User,
} from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { tasks as tasksApi, Task, professionals, Professional } from '@/lib/api';
import { format, isToday, isBefore, isThisWeek, startOfDay } from 'date-fns';
import { useTranslations, useDateLocale } from '@/lib/i18n';

type StatusFilter = '' | 'pending' | 'in_progress' | 'completed' | 'dismissed';
type SourceFilter = '' | 'manual' | 'rule' | 'system';

// Clasificar tareas por urgencia temporal
function classifyTask(task: Task): 'overdue' | 'today' | 'thisWeek' | 'later' | 'noDueDate' {
  if (!task.due_at) return 'noDueDate';
  const due = new Date(task.due_at);
  const now = startOfDay(new Date());
  if (isBefore(due, now)) return 'overdue';
  if (isToday(due)) return 'today';
  if (isThisWeek(due, { weekStartsOn: 1 })) return 'thisWeek';
  return 'later';
}

const STATUS_COLORS: Record<string, { bg: string; text: string }> = {
  pending:     { bg: 'bg-amber-100',   text: 'text-amber-700'   },
  in_progress: { bg: 'bg-blue-100',    text: 'text-blue-700'    },
  completed:   { bg: 'bg-emerald-100', text: 'text-emerald-700' },
  dismissed:   { bg: 'bg-neutral-100', text: 'text-neutral-500' },
};

export default function TasksPage() {
  const t = useTranslations();
  const dateLocale = useDateLocale();

  const [list, setList]             = useState<Task[]>([]);
  const [profList, setProfList]     = useState<Professional[]>([]);
  const [loading, setLoading]       = useState(true);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('');
  const [assignedFilter, setAssignedFilter] = useState('');
  const [sourceFilter, setSourceFilter] = useState<SourceFilter>('');

  // Formulario de crear tarea
  const [showForm, setShowForm]     = useState(false);
  const [formTitle, setFormTitle]   = useState('');
  const [formDesc, setFormDesc]     = useState('');
  const [formAssigned, setFormAssigned] = useState('');
  const [formDueAt, setFormDueAt]   = useState('');
  const [saving, setSaving]         = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {};
      if (statusFilter) params.status = statusFilter;
      if (assignedFilter) params.assigned_to = assignedFilter;
      if (sourceFilter) params.source = sourceFilter;
      const res = await tasksApi.list(params);
      setList(res.data ?? []);
    } catch {
      setList([]);
    } finally {
      setLoading(false);
    }
  }, [statusFilter, assignedFilter, sourceFilter]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    professionals.list().then(r => setProfList(r.data ?? [])).catch(() => {});
  }, []);

  async function handleComplete(id: string) {
    try {
      await tasksApi.complete(id);
      await load();
    } catch { /* silencioso */ }
  }

  async function handleDismiss(id: string) {
    try {
      await tasksApi.dismiss(id);
      await load();
    } catch { /* silencioso */ }
  }

  async function handleCreate() {
    if (!formTitle.trim()) return;
    setSaving(true);
    try {
      await tasksApi.create({
        title: formTitle.trim(),
        description: formDesc.trim() || undefined,
        assigned_to: formAssigned || undefined,
        due_at: formDueAt ? new Date(`${formDueAt}T00:00:00`).toISOString() : undefined,
      });
      setShowForm(false);
      setFormTitle('');
      setFormDesc('');
      setFormAssigned('');
      setFormDueAt('');
      await load();
    } catch { /* silencioso */ } finally {
      setSaving(false);
    }
  }

  // Agrupar tareas por urgencia
  const grouped = list.reduce<Record<string, Task[]>>((acc, task) => {
    // No agrupar las completadas/descartadas
    if (task.status === 'completed' || task.status === 'dismissed') {
      const key = task.status;
      (acc[key] ??= []).push(task);
    } else {
      const key = classifyTask(task);
      (acc[key] ??= []).push(task);
    }
    return acc;
  }, {});

  const groupOrder = ['overdue', 'today', 'thisWeek', 'later', 'noDueDate', 'completed', 'dismissed'];
  const groupLabels: Record<string, string> = {
    overdue:   t.crm.tasks.overdue,
    today:     t.crm.tasks.today,
    thisWeek:  t.crm.tasks.thisWeek,
    later:     t.crm.tasks.later,
    noDueDate: t.crm.tasks.noDueDate,
    completed: t.crm.tasks.completed,
    dismissed: t.crm.tasks.dismissed,
  };
  const groupIcons: Record<string, typeof Clock> = {
    overdue:   AlertTriangle,
    today:     Clock,
    thisWeek:  Clock,
    later:     Clock,
    noDueDate: Clock,
    completed: Check,
    dismissed: XCircle,
  };

  const pendingCount = list.filter(t => t.status === 'pending' || t.status === 'in_progress').length;

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-neutral-900">{t.crm.tasks.title}</h1>
          <p className="mt-0.5 text-sm text-neutral-500">{t.crm.tasks.description}</p>
        </div>
        <div className="flex items-center gap-3">
          {pendingCount > 0 && (
            <Badge variant="primary">
              {t.crm.tasks.pendingCount.replace('{n}', String(pendingCount))}
            </Badge>
          )}
          <button
            onClick={() => setShowForm(true)}
            className="flex items-center gap-1.5 rounded-lg bg-primary-600 px-3 py-1.5 text-xs font-semibold text-white transition-colors hover:bg-primary-500"
          >
            <Plus className="h-3.5 w-3.5" />
            {t.crm.tasks.newTask}
          </button>
        </div>
      </div>

      {/* Formulario de crear tarea */}
      {showForm && (
        <Card className="mb-6 p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-neutral-800">{t.crm.tasks.newTask}</h3>
            <button onClick={() => setShowForm(false)} className="text-neutral-400 hover:text-neutral-600">
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-neutral-600 mb-1">
                {t.crm.tasks.titleLabel}
              </label>
              <input
                type="text"
                value={formTitle}
                onChange={e => setFormTitle(e.target.value)}
                placeholder={t.crm.tasks.titlePlaceholder}
                className="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
                onKeyDown={e => e.key === 'Enter' && handleCreate()}
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-neutral-600 mb-1">
                {t.crm.tasks.dueDateLabel}
              </label>
              <input
                type="date"
                value={formDueAt}
                onChange={e => setFormDueAt(e.target.value)}
                className="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
              />
            </div>
            <div className="md:col-span-2">
              <label className="block text-xs font-medium text-neutral-600 mb-1">
                {t.crm.tasks.descriptionLabel}
              </label>
              <textarea
                value={formDesc}
                onChange={e => setFormDesc(e.target.value)}
                placeholder={t.crm.tasks.descriptionPlaceholder}
                rows={2}
                className="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100 resize-none"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-neutral-600 mb-1">
                {t.crm.tasks.assignTo}
              </label>
              <select
                value={formAssigned}
                onChange={e => setFormAssigned(e.target.value)}
                className="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
              >
                <option value="">{t.crm.tasks.filterAll}</option>
                {profList.filter(p => p.is_active).map(p => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>
            <div className="flex items-end">
              <button
                onClick={handleCreate}
                disabled={saving || !formTitle.trim()}
                className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary-500 disabled:opacity-50"
              >
                {saving ? t.common.loading : t.common.create}
              </button>
            </div>
          </div>
        </Card>
      )}

      {/* Filtros */}
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <select
          value={statusFilter}
          onChange={e => setStatusFilter(e.target.value as StatusFilter)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:outline-none focus:ring-1 focus:ring-primary-400"
        >
          <option value="">{t.crm.tasks.filterStatus}: {t.crm.tasks.filterAll}</option>
          <option value="pending">{t.crm.tasks.pending}</option>
          <option value="in_progress">{t.crm.tasks.inProgress}</option>
          <option value="completed">{t.crm.tasks.completed}</option>
          <option value="dismissed">{t.crm.tasks.dismissed}</option>
        </select>

        <select
          value={assignedFilter}
          onChange={e => setAssignedFilter(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:outline-none focus:ring-1 focus:ring-primary-400"
        >
          <option value="">{t.crm.tasks.filterAssigned}: {t.crm.tasks.filterAll}</option>
          {profList.filter(p => p.is_active).map(p => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>

        <select
          value={sourceFilter}
          onChange={e => setSourceFilter(e.target.value as SourceFilter)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:outline-none focus:ring-1 focus:ring-primary-400"
        >
          <option value="">{t.crm.tasks.filterSource}: {t.crm.tasks.filterAll}</option>
          <option value="manual">{t.crm.tasks.manual}</option>
          <option value="rule">{t.crm.tasks.rule}</option>
          <option value="system">{t.crm.tasks.system}</option>
        </select>

        <span className="ml-auto text-xs text-neutral-400">
          {t.crm.tasks.tasksCount.replace('{n}', String(list.length))}
        </span>
      </div>

      {/* Contenido */}
      {loading ? (
        <div className="flex justify-center py-16"><Spinner size="lg" /></div>
      ) : list.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <CheckSquare className="mb-3 h-12 w-12 text-neutral-200" />
          <p className="text-sm font-medium text-neutral-500">{t.crm.tasks.noTasks}</p>
          <p className="mt-1 text-xs text-neutral-400">{t.crm.tasks.noTasksDesc}</p>
        </div>
      ) : (
        <div className="space-y-6">
          {groupOrder.map(group => {
            const items = grouped[group];
            if (!items || items.length === 0) return null;
            const GroupIcon = groupIcons[group] ?? Clock;
            const isOverdue = group === 'overdue';

            return (
              <div key={group}>
                <div className="flex items-center gap-2 mb-2">
                  <GroupIcon className={`h-4 w-4 ${isOverdue ? 'text-red-500' : 'text-neutral-400'}`} />
                  <h3 className={`text-xs font-semibold uppercase tracking-wide ${
                    isOverdue ? 'text-red-600' : 'text-neutral-500'
                  }`}>
                    {groupLabels[group]} ({items.length})
                  </h3>
                </div>
                <div className="space-y-1">
                  {items.map(task => {
                    const sc = STATUS_COLORS[task.status] ?? STATUS_COLORS.pending;
                    const isActionable = task.status === 'pending' || task.status === 'in_progress';

                    return (
                      <div
                        key={task.id}
                        className="flex items-center gap-3 rounded-lg border border-neutral-200 bg-white px-4 py-3 transition-colors hover:bg-neutral-50"
                      >
                        {/* Status badge */}
                        <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${sc.bg} ${sc.text}`}>
                          {task.status === 'pending' && t.crm.tasks.pending}
                          {task.status === 'in_progress' && t.crm.tasks.inProgress}
                          {task.status === 'completed' && t.crm.tasks.completed}
                          {task.status === 'dismissed' && t.crm.tasks.dismissed}
                        </span>

                        {/* Titulo y descripcion */}
                        <div className="flex-1 min-w-0">
                          <p className={`text-sm font-medium ${
                            task.status === 'completed' || task.status === 'dismissed'
                              ? 'text-neutral-400 line-through' : 'text-neutral-900'
                          }`}>
                            {task.title}
                          </p>
                          {task.customer_name && (
                            <p className="text-xs text-neutral-500 flex items-center gap-1 mt-0.5">
                              <User className="h-3 w-3" />
                              {task.customer_name}
                            </p>
                          )}
                        </div>

                        {/* Source badge */}
                        <span className="text-[10px] text-neutral-400 font-medium uppercase hidden sm:inline">
                          {task.source === 'manual' && t.crm.tasks.manual}
                          {task.source === 'rule' && t.crm.tasks.rule}
                          {task.source === 'system' && t.crm.tasks.system}
                        </span>

                        {/* Due date */}
                        {task.due_at && (
                          <span className={`text-xs hidden sm:inline ${isOverdue ? 'text-red-500 font-medium' : 'text-neutral-500'}`}>
                            {format(new Date(task.due_at), 'd MMM', { locale: dateLocale })}
                          </span>
                        )}

                        {/* Assigned */}
                        {task.assigned_to_name && (
                          <span className="text-xs text-neutral-400 max-w-[100px] truncate hidden md:inline">
                            {task.assigned_to_name}
                          </span>
                        )}

                        {/* Acciones rapidas */}
                        {isActionable && (
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleComplete(task.id)}
                              title={t.crm.tasks.complete}
                              className="rounded p-1 text-emerald-500 hover:bg-emerald-50 transition-colors"
                            >
                              <Check className="h-4 w-4" />
                            </button>
                            <button
                              onClick={() => handleDismiss(task.id)}
                              title={t.crm.tasks.dismiss}
                              className="rounded p-1 text-neutral-400 hover:bg-neutral-100 transition-colors"
                            >
                              <XCircle className="h-4 w-4" />
                            </button>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
