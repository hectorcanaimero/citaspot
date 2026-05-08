'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  Zap, Trash2, ChevronDown, ChevronUp, AlertCircle, Plus, Pencil,
} from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { rules as rulesApi, Rule, RuleExecution } from '@/lib/api';
import { format } from 'date-fns';
import { useTranslations, useDateLocale } from '@/lib/i18n';
import { useRouter } from 'next/navigation';

export default function AutomationsPage() {
  const t = useTranslations();
  const dateLocale = useDateLocale();
  const router = useRouter();

  const [list, setList]         = useState<Rule[]>([]);
  const [loading, setLoading]   = useState(true);

  // Expandir ejecuciones de una regla
  const [expandedId, setExpandedId]     = useState<string | null>(null);
  const [executions, setExecutions]     = useState<RuleExecution[]>([]);
  const [loadingExec, setLoadingExec]   = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await rulesApi.list();
      setList(res.data ?? []);
    } catch {
      setList([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function toggleActive(rule: Rule) {
    try {
      await rulesApi.update(rule.id, { is_active: !rule.is_active });
      await load();
    } catch { /* silencioso */ }
  }

  async function handleDelete(id: string) {
    if (!confirm(t.crm.automations.confirmDelete)) return;
    try {
      await rulesApi.remove(id);
      await load();
    } catch { /* silencioso */ }
  }

  async function toggleExpand(ruleId: string) {
    if (expandedId === ruleId) {
      setExpandedId(null);
      setExecutions([]);
      return;
    }
    setExpandedId(ruleId);
    setLoadingExec(true);
    try {
      const res = await rulesApi.listExecutions(ruleId);
      setExecutions(res.data ?? []);
    } catch {
      setExecutions([]);
    } finally {
      setLoadingExec(false);
    }
  }

  // Mapa de trigger events para labels
  const triggerLabel = (event: string): string => {
    const map = t.crm.automations.triggerEvents as unknown as Record<string, string>;
    return map[event] ?? event;
  };

  const activeCount = list.filter(r => r.is_active).length;

  const EXEC_STATUS_COLORS: Record<string, { bg: string; text: string }> = {
    success: { bg: 'bg-emerald-100', text: 'text-emerald-700' },
    failed:  { bg: 'bg-red-100',     text: 'text-red-600'     },
    skipped: { bg: 'bg-neutral-100', text: 'text-neutral-500' },
  };

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-neutral-900">{t.crm.automations.title}</h1>
          <p className="mt-0.5 text-sm text-neutral-500">{t.crm.automations.description}</p>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-xs text-neutral-500">
            {t.crm.automations.activeCount.replace('{n}', String(activeCount))}
            {' / '}
            {t.crm.automations.rulesCount.replace('{n}', String(list.length))}
          </span>
          <button
            type="button"
            onClick={() => router.push('/automations/new')}
            className="inline-flex items-center gap-1.5 rounded-lg bg-neutral-900 px-3 py-1.5 text-xs font-medium text-white hover:bg-neutral-700 transition-colors"
          >
            <Plus className="h-3.5 w-3.5" />
            {t.crm.automations.newRule}
          </button>
        </div>
      </div>

      {/* Contenido */}
      {loading ? (
        <div className="flex justify-center py-16"><Spinner size="lg" /></div>
      ) : list.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <Zap className="mb-3 h-12 w-12 text-neutral-200" />
          <p className="text-sm font-medium text-neutral-500">{t.crm.automations.noRules}</p>
          <p className="mt-1 text-xs text-neutral-400">{t.crm.automations.noRulesDesc}</p>
        </div>
      ) : (
        <div className="space-y-3">
          {list.map(rule => {
            const isExpanded = expandedId === rule.id;

            return (
              <div
                key={rule.id}
                className="rounded-xl border border-neutral-200 bg-white overflow-hidden transition-shadow hover:shadow-sm"
              >
                {/* Fila principal */}
                <div className="flex items-center gap-3 px-4 py-3">
                  {/* Toggle activo/inactivo */}
                  <button
                    type="button"
                    role="switch"
                    aria-checked={rule.is_active}
                    onClick={() => toggleActive(rule)}
                    className={`relative h-5 w-9 rounded-full transition-colors flex-shrink-0 focus:outline-none focus:ring-2 focus:ring-primary-400 focus:ring-offset-1 ${
                      rule.is_active ? 'bg-emerald-500' : 'bg-neutral-300'
                    }`}
                  >
                    <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white shadow transition-transform ${
                      rule.is_active ? 'translate-x-4' : 'translate-x-0.5'
                    }`} />
                  </button>

                  {/* Nombre y descripcion */}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-neutral-900">{rule.name}</p>
                    {rule.description && (
                      <p className="text-xs text-neutral-400 truncate">{rule.description}</p>
                    )}
                  </div>

                  {/* Trigger type badge */}
                  <Badge variant={rule.trigger_type === 'event' ? 'primary' : 'default'}>
                    {rule.trigger_type === 'event' ? t.crm.automations.event : t.crm.automations.temporal}
                  </Badge>

                  {/* Trigger event */}
                  <span className="text-xs text-neutral-500 max-w-[160px] truncate hidden md:inline">
                    {triggerLabel(rule.trigger_event)}
                  </span>

                  {/* Cooldown */}
                  {rule.cooldown_hours > 0 && (
                    <span className="text-[10px] text-neutral-400 hidden lg:inline">
                      {t.crm.automations.cooldownHours.replace('{n}', String(rule.cooldown_hours))}
                    </span>
                  )}

                  {/* Acciones */}
                  <div className="flex items-center gap-1 flex-shrink-0">
                    <button
                      onClick={() => toggleExpand(rule.id)}
                      className="rounded p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600"
                      title={t.crm.automations.executions}
                    >
                      {isExpanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                    </button>
                    <button
                      onClick={() => router.push(`/automations/${rule.id}`)}
                      className="rounded p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600"
                      title={t.crm.automations.editRule}
                    >
                      <Pencil className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => handleDelete(rule.id)}
                      className="rounded p-1 text-neutral-400 hover:bg-red-50 hover:text-red-500"
                      title={t.crm.automations.deleteRule}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>

                {/* Panel de ejecuciones expandido */}
                {isExpanded && (
                  <div className="border-t border-neutral-100 bg-neutral-50 px-4 py-3">
                    <h4 className="text-xs font-semibold text-neutral-600 mb-2">
                      {t.crm.automations.executions}
                    </h4>

                    {loadingExec ? (
                      <div className="flex justify-center py-4">
                        <Spinner size="sm" />
                      </div>
                    ) : executions.length === 0 ? (
                      <p className="text-xs text-neutral-400 py-2">{t.crm.automations.noExecutions}</p>
                    ) : (
                      <div className="space-y-1 max-h-[240px] overflow-y-auto">
                        {executions.slice(0, 20).map(exec => {
                          const esc = EXEC_STATUS_COLORS[exec.status] ?? EXEC_STATUS_COLORS.skipped;
                          return (
                            <div
                              key={exec.id}
                              className="flex items-center gap-3 rounded-md bg-white px-3 py-2 text-xs"
                            >
                              <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${esc.bg} ${esc.text}`}>
                                {exec.status === 'success' && t.crm.automations.executionSuccess}
                                {exec.status === 'failed' && t.crm.automations.executionFailed}
                                {exec.status === 'skipped' && t.crm.automations.executionSkipped}
                              </span>
                              <span className="text-neutral-500">
                                {format(new Date(exec.triggered_at), 'd MMM HH:mm', { locale: dateLocale })}
                              </span>
                              <span className="text-neutral-400 truncate flex-1">
                                {triggerLabel(exec.trigger_event)}
                              </span>
                              {exec.error_message && (
                                <span className="text-red-500 truncate max-w-[200px]" title={exec.error_message}>
                                  <AlertCircle className="h-3 w-3 inline mr-1" />
                                  {exec.error_message}
                                </span>
                              )}
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
