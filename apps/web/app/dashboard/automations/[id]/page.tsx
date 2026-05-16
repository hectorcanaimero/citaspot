'use client';

import { useState, useEffect, useCallback } from 'react';
import { useRouter, useParams } from 'next/navigation';
import { ArrowLeft, Plus, X } from 'lucide-react';
import { Spinner } from '@/components/ui/spinner';
import { ToggleSwitch } from '@/components/ui/toggle-switch';
import { rules as rulesApi, Rule } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

// ── Tipos locales ─────────────────────────────────────────────────────────────

interface Condition {
  field: string;
  op: string;
  value: string;
}

interface Action {
  type: string;
  template: string;
  params: Record<string, unknown>;
}

const TRIGGER_EVENTS = [
  'appointment.created',
  'appointment.confirmed',
  'appointment.completed',
  'appointment.cancelled',
  'appointment.rescheduled',
  'appointment.no_show',
  'customer.created',
  'customer.stage_changed',
  'treatment.proposed',
  'treatment.accepted',
  'treatment.completed',
  'treatment.abandoned',
] as const;

const REFERENCE_FIELDS = ['last_visit_at', 'next_recall_at', 'created_at'] as const;

const OPS = ['eq', 'neq', 'gt', 'lt', 'gte', 'lte', 'contains', 'in'] as const;

const ACTION_TYPES = ['send_whatsapp', 'create_task', 'move_stage', 'update_field'] as const;

// ── Helpers de UI ─────────────────────────────────────────────────────────────

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="mb-3 text-xs font-semibold uppercase tracking-wide text-neutral-500">
      {children}
    </h3>
  );
}

function FieldLabel({ children, required }: { children: React.ReactNode; required?: boolean }) {
  return (
    <label className="mb-1 block text-sm font-medium text-neutral-700">
      {children}
      {required && <span className="ml-0.5 text-red-500">*</span>}
    </label>
  );
}

const inputCls =
  'w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-neutral-400 focus:outline-none focus:ring-1 focus:ring-neutral-300';

const selectCls =
  'rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-neutral-400 focus:outline-none focus:ring-1 focus:ring-neutral-300';

// ── Componente principal ──────────────────────────────────────────────────────

export default function EditRulePage() {
  const t = useTranslations();
  const router = useRouter();
  const params = useParams<{ id: string }>();
  const id = params.id;
  const ta = t.crm.automations;

  // Carga inicial
  const [loadingRule, setLoadingRule] = useState(true);
  const [loadError, setLoadError]     = useState<string | null>(null);

  // Estado del form
  const [name, setName]                   = useState('');
  const [description, setDescription]     = useState('');
  const [triggerType, setTriggerType]     = useState<'event' | 'temporal'>('event');
  const [triggerEvent, setTriggerEvent]   = useState<string>(TRIGGER_EVENTS[0]);
  const [intervalDays, setIntervalDays]   = useState(7);
  const [referenceField, setReferenceField] = useState<string>(REFERENCE_FIELDS[0]);
  const [conditions, setConditions]       = useState<Condition[]>([]);
  const [actions, setActions]             = useState<Action[]>([
    { type: 'send_whatsapp', template: '', params: {} },
  ]);
  const [cooldownHours, setCooldownHours] = useState(24);
  const [isActive, setIsActive]           = useState(true);
  const [saving, setSaving]               = useState(false);
  const [error, setError]                 = useState<string | null>(null);

  // Cargar regla al montar
  const loadRule = useCallback(async () => {
    setLoadingRule(true);
    setLoadError(null);
    try {
      const rule: Rule = await rulesApi.getById(id);
      setName(rule.name);
      setDescription(rule.description ?? '');
      setTriggerType(rule.trigger_type === 'temporal' ? 'temporal' : 'event');
      setTriggerEvent(rule.trigger_event || TRIGGER_EVENTS[0]);
      if (rule.trigger_schedule) {
        setIntervalDays(rule.trigger_schedule.interval_days);
        setReferenceField(rule.trigger_schedule.reference_field || REFERENCE_FIELDS[0]);
      }
      setConditions(
        (rule.conditions ?? []).map(c => ({
          field: String(c.field),
          op: String(c.op),
          value: String(c.value),
        })),
      );
      setActions(
        (rule.actions ?? []).map(a => ({
          type: a.type,
          template: a.template ?? '',
          params: a.params ?? {},
        })),
      );
      setCooldownHours(rule.cooldown_hours ?? 24);
      setIsActive(rule.is_active ?? true);
    } catch (err) {
      setLoadError(err instanceof Error ? err.message : t.common.error);
    } finally {
      setLoadingRule(false);
    }
  }, [id, t.common.error]);

  useEffect(() => {
    loadRule();
  }, [loadRule]);

  // Conditions helpers
  const addCondition = useCallback(() => {
    setConditions(prev => [...prev, { field: '', op: 'eq', value: '' }]);
  }, []);

  const removeCondition = useCallback((idx: number) => {
    setConditions(prev => prev.filter((_, i) => i !== idx));
  }, []);

  const updateCondition = useCallback((idx: number, patch: Partial<Condition>) => {
    setConditions(prev => prev.map((c, i) => (i === idx ? { ...c, ...patch } : c)));
  }, []);

  // Actions helpers
  const addAction = useCallback(() => {
    setActions(prev => [...prev, { type: 'send_whatsapp', template: '', params: {} }]);
  }, []);

  const removeAction = useCallback((idx: number) => {
    setActions(prev => prev.filter((_, i) => i !== idx));
  }, []);

  const updateAction = useCallback((idx: number, patch: Partial<Action>) => {
    setActions(prev => prev.map((a, i) => (i === idx ? { ...a, ...patch } : a)));
  }, []);

  const updateActionParams = useCallback((idx: number, raw: string) => {
    try {
      const parsed = JSON.parse(raw) as Record<string, unknown>;
      setActions(prev => prev.map((a, i) => (i === idx ? { ...a, params: parsed } : a)));
    } catch {
      // JSON invalido — no actualizar params todavia
    }
  }, []);

  // Submit
  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      setError(null);

      if (actions.length === 0) {
        setError(ta.atLeastOneAction);
        return;
      }

      setSaving(true);
      try {
        await rulesApi.update(id, {
          name: name.trim(),
          description: description.trim(),
          trigger_type: triggerType,
          trigger_event: triggerType === 'event' ? triggerEvent : '',
          trigger_schedule:
            triggerType === 'temporal'
              ? { interval_days: intervalDays, reference_field: referenceField }
              : undefined,
          conditions: conditions.map(c => ({ ...c, value: c.value })),
          actions,
          cooldown_hours: cooldownHours,
          is_active: isActive,
        });
        router.push('/dashboard/automations');
      } catch (err) {
        setError(err instanceof Error ? err.message : t.common.error);
      } finally {
        setSaving(false);
      }
    },
    [
      id, name, description, triggerType, triggerEvent, intervalDays, referenceField,
      conditions, actions, cooldownHours, isActive, router, ta, t.common.error,
    ],
  );

  // ── Render ──────────────────────────────────────────────────────────────────

  if (loadingRule) {
    return (
      <div className="flex justify-center py-24">
        <Spinner size="lg" />
      </div>
    );
  }

  if (loadError) {
    return (
      <div className="mx-auto max-w-2xl p-6">
        <p className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-600">
          {loadError}
        </p>
        <button
          type="button"
          onClick={() => router.push('/dashboard/automations')}
          className="mt-4 text-sm text-neutral-500 underline"
        >
          {t.common.back}
        </button>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl p-6">
      {/* Header */}
      <div className="mb-6 flex items-center gap-3">
        <button
          type="button"
          onClick={() => router.push('/dashboard/automations')}
          className="rounded-lg p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600"
        >
          <ArrowLeft className="h-4 w-4" />
        </button>
        <div>
          <h1 className="text-xl font-semibold text-neutral-900">{ta.editTitle}</h1>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Nombre y descripcion */}
        <div className="rounded-xl border border-neutral-200 bg-white p-5">
          <SectionTitle>{ta.nameLabel}</SectionTitle>
          <div className="space-y-4">
            <div>
              <FieldLabel required>{ta.nameLabel}</FieldLabel>
              <input
                type="text"
                required
                value={name}
                onChange={e => setName(e.target.value)}
                placeholder={ta.namePlaceholder}
                className={inputCls}
              />
            </div>
            <div>
              <FieldLabel>{ta.descriptionLabel}</FieldLabel>
              <input
                type="text"
                value={description}
                onChange={e => setDescription(e.target.value)}
                placeholder={ta.descriptionPlaceholder}
                className={inputCls}
              />
            </div>
          </div>
        </div>

        {/* Disparador */}
        <div className="rounded-xl border border-neutral-200 bg-white p-5">
          <SectionTitle>{ta.triggerTypeLabel}</SectionTitle>
          <div className="space-y-4">
            {/* Tipo */}
            <div className="flex gap-4">
              {(['event', 'temporal'] as const).map(type => (
                <label key={type} className="flex cursor-pointer items-center gap-2">
                  <input
                    type="radio"
                    name="triggerType"
                    value={type}
                    checked={triggerType === type}
                    onChange={() => setTriggerType(type)}
                    className="accent-neutral-900"
                  />
                  <span className="text-sm font-medium text-neutral-700">
                    {type === 'event' ? ta.event : ta.temporal}
                  </span>
                </label>
              ))}
            </div>

            {/* Campos segun tipo */}
            {triggerType === 'event' && (
              <div>
                <FieldLabel required>{ta.triggerEventLabel}</FieldLabel>
                <select
                  value={triggerEvent}
                  onChange={e => setTriggerEvent(e.target.value)}
                  className={`${selectCls} w-full`}
                >
                  {TRIGGER_EVENTS.map(ev => (
                    <option key={ev} value={ev}>
                      {(ta.triggerEvents as unknown as Record<string, string>)[ev] ?? ev}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {triggerType === 'temporal' && (
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <FieldLabel required>{ta.intervalDaysLabel}</FieldLabel>
                  <input
                    type="number"
                    min={1}
                    value={intervalDays}
                    onChange={e => setIntervalDays(Number(e.target.value))}
                    className={inputCls}
                  />
                </div>
                <div>
                  <FieldLabel required>{ta.referenceFieldLabel}</FieldLabel>
                  <select
                    value={referenceField}
                    onChange={e => setReferenceField(e.target.value)}
                    className={`${selectCls} w-full`}
                  >
                    {REFERENCE_FIELDS.map(rf => (
                      <option key={rf} value={rf}>
                        {(ta.referenceFields as unknown as Record<string, string>)[rf] ?? rf}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Condiciones */}
        <div className="rounded-xl border border-neutral-200 bg-white p-5">
          <div className="mb-3 flex items-center justify-between">
            <SectionTitle>{ta.conditionsLabel}</SectionTitle>
            <button
              type="button"
              onClick={addCondition}
              className="inline-flex items-center gap-1 rounded-lg border border-neutral-200 px-2.5 py-1 text-xs font-medium text-neutral-600 hover:bg-neutral-50"
            >
              <Plus className="h-3 w-3" />
              {ta.addCondition}
            </button>
          </div>

          {conditions.length === 0 ? (
            <p className="text-xs text-neutral-400">
              Sin condiciones — la regla se ejecuta para todos los eventos del disparador.
            </p>
          ) : (
            <div className="space-y-2">
              {conditions.map((cond, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <input
                    type="text"
                    value={cond.field}
                    onChange={e => updateCondition(idx, { field: e.target.value })}
                    placeholder={ta.fieldLabel}
                    className={`${inputCls} flex-1`}
                  />
                  <select
                    value={cond.op}
                    onChange={e => updateCondition(idx, { op: e.target.value })}
                    className={selectCls}
                  >
                    {OPS.map(op => (
                      <option key={op} value={op}>
                        {(ta.ops as unknown as Record<string, string>)[op] ?? op}
                      </option>
                    ))}
                  </select>
                  <input
                    type="text"
                    value={cond.value}
                    onChange={e => updateCondition(idx, { value: e.target.value })}
                    placeholder={ta.valueLabel}
                    className={`${inputCls} flex-1`}
                  />
                  <button
                    type="button"
                    onClick={() => removeCondition(idx)}
                    className="rounded p-1 text-neutral-400 hover:bg-red-50 hover:text-red-500"
                    title={ta.removeCondition}
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Acciones */}
        <div className="rounded-xl border border-neutral-200 bg-white p-5">
          <div className="mb-3 flex items-center justify-between">
            <SectionTitle>{ta.actionsLabel}</SectionTitle>
            <button
              type="button"
              onClick={addAction}
              className="inline-flex items-center gap-1 rounded-lg border border-neutral-200 px-2.5 py-1 text-xs font-medium text-neutral-600 hover:bg-neutral-50"
            >
              <Plus className="h-3 w-3" />
              {ta.addAction}
            </button>
          </div>

          <div className="space-y-4">
            {actions.map((action, idx) => (
              <div key={idx} className="rounded-lg border border-neutral-100 bg-neutral-50 p-3">
                <div className="mb-2 flex items-center justify-between">
                  <select
                    value={action.type}
                    onChange={e => updateAction(idx, { type: e.target.value })}
                    className={selectCls}
                  >
                    {ACTION_TYPES.map(at => (
                      <option key={at} value={at}>
                        {(ta.actionTypes as unknown as Record<string, string>)[at] ?? at}
                      </option>
                    ))}
                  </select>
                  <button
                    type="button"
                    onClick={() => removeAction(idx)}
                    className="rounded p-1 text-neutral-400 hover:bg-red-50 hover:text-red-500"
                    title={ta.removeAction}
                  >
                    <X className="h-4 w-4" />
                  </button>
                </div>

                {/* Template para whatsapp y create_task */}
                {(action.type === 'send_whatsapp' || action.type === 'create_task') && (
                  <div>
                    <FieldLabel>{ta.templateLabel}</FieldLabel>
                    <textarea
                      value={action.template}
                      onChange={e => updateAction(idx, { template: e.target.value })}
                      placeholder={ta.templatePlaceholder}
                      rows={action.type === 'send_whatsapp' ? 3 : 1}
                      className={`${inputCls} resize-none`}
                    />
                    {action.type === 'send_whatsapp' && (
                      <p className="mt-1 text-[11px] text-neutral-400">
                        Variables: {'{{customer_name}}'}, {'{{service_name}}'}, {'{{tenant_name}}'}
                      </p>
                    )}
                  </div>
                )}

                {/* Params para move_stage y update_field */}
                {(action.type === 'move_stage' || action.type === 'update_field') && (
                  <div>
                    <FieldLabel>{ta.paramsLabel}</FieldLabel>
                    <textarea
                      defaultValue={JSON.stringify(action.params, null, 2)}
                      onChange={e => updateActionParams(idx, e.target.value)}
                      rows={3}
                      placeholder='{}'
                      className={`${inputCls} resize-none font-mono text-xs`}
                    />
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Configuracion adicional */}
        <div className="rounded-xl border border-neutral-200 bg-white p-5">
          <SectionTitle>Configuracion</SectionTitle>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <FieldLabel>{ta.cooldownLabel}</FieldLabel>
              <input
                type="number"
                min={0}
                value={cooldownHours}
                onChange={e => setCooldownHours(Number(e.target.value))}
                className={inputCls}
              />
            </div>
            <div className="flex items-end pb-2">
              <label className="flex cursor-pointer items-center gap-2">
                <ToggleSwitch checked={isActive} onCheckedChange={() => setIsActive(v => !v)} activeColor="bg-emerald-500" />
                <span className="text-sm font-medium text-neutral-700">{ta.isActiveLabel}</span>
              </label>
            </div>
          </div>
        </div>

        {/* Error */}
        {error && (
          <p className="rounded-lg border border-red-200 bg-red-50 px-4 py-2.5 text-sm text-red-600">
            {error}
          </p>
        )}

        {/* Botones */}
        <div className="flex justify-end gap-3">
          <button
            type="button"
            onClick={() => router.push('/dashboard/automations')}
            className="rounded-lg border border-neutral-200 px-4 py-2 text-sm font-medium text-neutral-600 hover:bg-neutral-50"
          >
            {t.common.cancel}
          </button>
          <button
            type="submit"
            disabled={saving}
            className="inline-flex items-center gap-2 rounded-lg bg-neutral-900 px-5 py-2 text-sm font-medium text-white hover:bg-neutral-700 disabled:opacity-60"
          >
            {saving && <Spinner size="sm" />}
            {saving ? ta.saving : t.common.saveChanges}
          </button>
        </div>
      </form>
    </div>
  );
}
