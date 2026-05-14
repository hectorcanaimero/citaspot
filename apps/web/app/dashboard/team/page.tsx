'use client';

// Página de gestión de equipo: profesionales y sus horarios semanales.
import { useState, useEffect } from 'react';
import { Plus, ChevronDown, ChevronUp, Pencil, Check, Archive, ArchiveRestore, X } from 'lucide-react';
import { Button }  from '@/components/ui/button';
import { Input }   from '@/components/ui/input';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge }   from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { ToggleSwitch } from '@/components/ui/toggle-switch';
import { professionals as profsApi, services as servicesApi, Professional, Service, Schedule, ProfessionalInput, APIError } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

// Días de la semana — solo el dow, las etiquetas vienen de t.team.days
const DAYS_DOW = [1, 2, 3, 4, 5, 6, 0];

const COLORS = ['#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#3b82f6', '#ef4444', '#14b8a6'];

interface DayConfig { is_active: boolean; start_time: string; end_time: string; }
type DayMap = Record<number, DayConfig>;

function defaultDays(): DayMap {
  return Object.fromEntries(
    DAYS_DOW.map((dow) => [dow, { is_active: dow >= 1 && dow <= 5, start_time: '08:00', end_time: '18:00' }])
  );
}

function scheduleToDayMap(schedules: Schedule[]): DayMap {
  const map = defaultDays();
  DAYS_DOW.forEach((dow) => { map[dow].is_active = false; });
  schedules.forEach((s) => {
    map[s.day_of_week] = { is_active: s.is_active, start_time: s.start_time, end_time: s.end_time };
  });
  return map;
}

// ── Componente de horario inline ──────────────────────────────────────────────

function ScheduleEditor({ profId, initial }: { profId: string; initial: DayMap }) {
  const t = useTranslations();
  const dayLabels = t.team.days as Record<string, string>;

  const [days, setDays]     = useState<DayMap>(initial);
  const [saving, setSaving] = useState(false);
  const [saved,  setSaved]  = useState(false);
  const [error,  setError]  = useState('');

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
        start_time:  days[dow].start_time,
        end_time:    days[dow].end_time,
        is_active:   days[dow].is_active,
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
            <span className={`w-8 text-sm ${day.is_active ? 'text-neutral-900 font-medium' : 'text-neutral-400'}`}>
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

      <Button size="sm" variant={saved ? 'secondary' : 'primary'} onClick={save} loading={saving} className="self-end">
        {saved ? <><Check className="mr-1.5 h-3.5 w-3.5" />{t.team.saved}</> : t.team.saveSchedule}
      </Button>
    </div>
  );
}

// ── Componente de asignación de servicios ─────────────────────────────────────

function ServicesEditor({ profId }: { profId: string }) {
  const t = useTranslations();

  const [assigned,      setAssigned]      = useState<Service[]>([]);
  const [allServices,   setAllServices]   = useState<Service[]>([]);
  const [loadingInit,   setLoadingInit]   = useState(true);
  const [togglingId,    setTogglingId]    = useState<string | null>(null);
  const [error,         setError]         = useState('');

  useEffect(() => {
    Promise.all([
      profsApi.listServices(profId),
      servicesApi.list(),
    ])
      .then(([assignedRes, allRes]) => {
        setAssigned(assignedRes.data ?? []);
        setAllServices((allRes.data ?? []).filter((s) => s.is_active));
      })
      .catch(() => setError(t.team.servicesLoadError ?? 'Error cargando servicios'))
      .finally(() => setLoadingInit(false));
  }, [profId]);

  const assignedIds = new Set(assigned.map((s) => s.id));

  async function toggle(service: Service) {
    setTogglingId(service.id);
    setError('');
    try {
      if (assignedIds.has(service.id)) {
        await profsApi.removeService(profId, service.id);
        setAssigned((prev) => prev.filter((s) => s.id !== service.id));
      } else {
        await profsApi.assignService(profId, service.id);
        setAssigned((prev) => [...prev, service]);
      }
    } catch {
      setError(t.team.servicesToggleError ?? 'Error al actualizar servicios');
    } finally {
      setTogglingId(null);
    }
  }

  if (loadingInit) return <div className="flex justify-center py-4"><Spinner /></div>;

  return (
    <div className="flex flex-col gap-2">
      {allServices.length === 0 ? (
        <p className="text-xs text-neutral-400">{t.team.noServicesYet ?? 'No hay servicios creados aún'}</p>
      ) : (
        <div className="flex flex-wrap gap-2">
          {allServices.map((svc) => {
            const active = assignedIds.has(svc.id);
            const loading = togglingId === svc.id;
            return (
              <button
                key={svc.id}
                type="button"
                disabled={loading}
                onClick={() => toggle(svc)}
                className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-medium transition-colors ${
                  active
                    ? 'bg-primary-100 text-primary-700 ring-1 ring-primary-300 hover:bg-primary-200'
                    : 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200'
                } disabled:opacity-50`}
              >
                {active && <Check className="h-3 w-3" />}
                {svc.name}
                {active && !loading && <X className="h-3 w-3 opacity-50" />}
                {loading && <Spinner size="sm" />}
              </button>
            );
          })}
        </div>
      )}
      {error && <p className="rounded-md bg-red-50 px-3 py-1.5 text-xs text-red-700">{error}</p>}
    </div>
  );
}

// ── Componente de profesional ─────────────────────────────────────────────────

function ProfCard({ prof, onUpdated }: { prof: Professional; onUpdated: (p: Professional) => void; }) {
  const t = useTranslations();
  const [expanded,   setExpanded]  = useState(false);
  const [editing,    setEditing]   = useState(false);
  const [editName,   setEditName]  = useState(prof.name);
  const [editSpec,   setEditSpec]  = useState(prof.specialty ?? '');
  const [saving,     setSaving]    = useState(false);
  const [archiving,  setArchiving] = useState(false);
  const [schedules,  setSchedules] = useState<DayMap | null>(null);
  const [loadingSch, setLoadingSch] = useState(false);

  async function loadSchedule() {
    if (schedules) return;
    setLoadingSch(true);
    try {
      const res = await profsApi.getSchedule(prof.id);
      setSchedules(scheduleToDayMap(res.data));
    } catch {
      setSchedules(defaultDays());
    } finally {
      setLoadingSch(false);
    }
  }

  function toggleExpand() {
    if (!expanded) loadSchedule();
    setExpanded((v) => !v);
  }

  async function saveEdit() {
    setSaving(true);
    try {
      const updated = await profsApi.update(prof.id, { name: editName.trim(), specialty: editSpec.trim() });
      onUpdated(updated);
      setEditing(false);
    } catch { /* ignorar */ }
    finally { setSaving(false); }
  }

  async function toggleArchive() {
    if (!prof.is_archived && !window.confirm(t.team.confirmArchive)) return;
    setArchiving(true);
    try {
      const updated = await profsApi.update(prof.id, { is_archived: !prof.is_archived });
      onUpdated(updated);
    } catch { /* ignorar */ }
    finally { setArchiving(false); }
  }

  return (
    <Card className={prof.is_archived ? 'opacity-60' : ''}>
      <div className="flex items-center gap-3">
        {/* Avatar */}
        <div
          className={`flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full text-white text-sm font-semibold ${prof.is_archived ? 'grayscale' : ''}`}
          style={{ backgroundColor: prof.color || COLORS[0] }}
        >
          {prof.name[0]}
        </div>

        {/* Info o edición */}
        {editing ? (
          <div className="flex flex-1 flex-col gap-2">
            <Input
              label=""
              placeholder={t.team.namePlaceholder}
              value={editName}
              onChange={(e) => setEditName(e.target.value)}
              className="h-8 text-sm"
            />
            <Input
              label=""
              placeholder={t.team.specialtyPlaceholder}
              value={editSpec}
              onChange={(e) => setEditSpec(e.target.value)}
              className="h-8 text-sm"
            />
          </div>
        ) : (
          <div className="flex-1">
            <p className="text-sm font-medium text-neutral-900">{prof.name}</p>
            {prof.specialty && <p className="text-xs text-neutral-500">{prof.specialty}</p>}
          </div>
        )}

        {/* Acciones */}
        <div className="flex items-center gap-1.5">
          {prof.is_archived ? (
            <Badge variant="default" className="text-xs">{t.team.archived}</Badge>
          ) : (
            <Badge variant={prof.is_active ? 'success' : 'default'} className="text-xs">
              {prof.is_active ? t.common.active : t.common.inactive}
            </Badge>
          )}

          {editing ? (
            <>
              <Button size="sm" onClick={saveEdit} loading={saving}>{t.common.save}</Button>
              <Button size="sm" variant="ghost" onClick={() => setEditing(false)}>{t.common.cancel}</Button>
            </>
          ) : (
            <button
              type="button"
              onClick={() => setEditing(true)}
              className="rounded-md p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 transition-colors"
              title={t.common.edit}
            >
              <Pencil className="h-4 w-4" />
            </button>
          )}

          <button
            type="button"
            onClick={toggleArchive}
            disabled={archiving}
            className="rounded-md p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 transition-colors"
            title={prof.is_archived ? t.team.restore : t.team.archive}
          >
            {prof.is_archived ? <ArchiveRestore className="h-4 w-4" /> : <Archive className="h-4 w-4" />}
          </button>

          <button
            type="button"
            onClick={toggleExpand}
            className="rounded-md p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 transition-colors"
            title={t.team.viewSchedule}
          >
            {expanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
          </button>
        </div>
      </div>

      {/* Servicios + Horarios */}
      {expanded && (
        <div className="mt-4 border-t border-neutral-100 pt-4 flex flex-col gap-5">
          {/* Servicios asignados */}
          <div>
            <p className="mb-2 text-xs font-medium text-neutral-500 uppercase tracking-wide">
              {t.team.assignedServices ?? 'Servicios que ofrece'}
            </p>
            <ServicesEditor profId={prof.id} />
          </div>

          {/* Horario semanal */}
          <div>
            <p className="mb-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">{t.team.weeklySchedule}</p>
            {loadingSch ? (
              <div className="flex justify-center py-4"><Spinner /></div>
            ) : schedules ? (
              <ScheduleEditor profId={prof.id} initial={schedules} />
            ) : null}
          </div>
        </div>
      )}
    </Card>
  );
}

// ── Formulario de nuevo profesional ──────────────────────────────────────────

function NewProfForm({ onCreated }: { onCreated: (p: Professional) => void }) {
  const t = useTranslations();
  const [name,   setName]   = useState('');
  const [spec,   setSpec]   = useState('');
  const [color,  setColor]  = useState(COLORS[0]);
  const [saving, setSaving] = useState(false);
  const [error,  setError]  = useState('');

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setSaving(true);
    setError('');
    try {
      const input: ProfessionalInput = { name: name.trim(), specialty: spec.trim(), color, is_active: true };
      const prof = await profsApi.create(input);
      onCreated(prof);
      setName(''); setSpec(''); setColor(COLORS[0]);
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.team.addProfError);
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card>
      <p className="mb-3 text-sm font-medium text-neutral-900">{t.team.newProfessional}</p>
      <form onSubmit={handleSubmit} className="flex flex-col gap-3">
        <Input label={t.team.nameLabel} placeholder={t.team.namePlaceholder} value={name} onChange={(e) => setName(e.target.value)} required />
        <Input label={t.team.specialtyLabel} placeholder={t.team.specialtyPlaceholder} value={spec} onChange={(e) => setSpec(e.target.value)} />
        <div className="flex flex-col gap-1.5">
          <span className="text-sm font-medium text-neutral-700">{t.team.colorLabel}</span>
          <div className="flex gap-2">
            {COLORS.map((c) => (
              <button
                key={c}
                type="button"
                onClick={() => setColor(c)}
                className={`h-7 w-7 rounded-full transition-transform ${color === c ? 'ring-2 ring-offset-2 ring-primary-500 scale-110' : ''}`}
                style={{ backgroundColor: c }}
              />
            ))}
          </div>
        </div>
        {error && <p className="rounded-md bg-red-50 px-3 py-1.5 text-sm text-red-700">{error}</p>}
        <Button type="submit" size="sm" loading={saving}>{t.team.addProfessional}</Button>
      </form>
    </Card>
  );
}

// ── Página principal ──────────────────────────────────────────────────────────

export default function TeamPage() {
  const t = useTranslations();
  const [profs,        setProfs]        = useState<Professional[]>([]);
  const [loading,      setLoading]      = useState(true);
  const [showForm,     setShowForm]     = useState(false);
  const [showArchived, setShowArchived] = useState(false);

  useEffect(() => {
    setLoading(true);
    profsApi.list(showArchived)
      .then((res) => setProfs(res.data))
      .catch(() => { /* ignorar */ })
      .finally(() => setLoading(false));
  }, [showArchived]);

  function handleCreated(p: Professional) {
    setProfs((prev) => [p, ...prev]);
    setShowForm(false);
  }

  function handleUpdated(updated: Professional) {
    setProfs((prev) => prev.map((p) => (p.id === updated.id ? updated : p)));
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-neutral-900">{t.team.title}</h1>
          <p className="mt-0.5 text-sm text-neutral-500">{t.team.description}</p>
        </div>
        <div className="flex items-center gap-3">
          <label className="flex items-center gap-2 text-sm text-neutral-600 cursor-pointer select-none">
            <input
              type="checkbox"
              checked={showArchived}
              onChange={(e) => setShowArchived(e.target.checked)}
              className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
            />
            {t.team.showArchived}
          </label>
          <Button size="sm" onClick={() => setShowForm((v) => !v)}>
            <Plus className="mr-1.5 h-4 w-4" />
            {t.team.addProfessional}
          </Button>
        </div>
      </div>

      <div className="flex max-w-2xl flex-col gap-4">
        {showForm && <NewProfForm onCreated={handleCreated} />}

        {loading ? (
          <div className="flex justify-center py-16"><Spinner size="lg" /></div>
        ) : profs.length === 0 ? (
          <div className="flex flex-col items-center rounded-xl border border-dashed border-neutral-200 py-16 text-center">
            <p className="text-sm font-medium text-neutral-700 mb-1">{t.team.noTeam}</p>
            <p className="text-xs text-neutral-500 mb-4">{t.team.noTeamDesc}</p>
            <Button size="sm" onClick={() => setShowForm(true)}>
              <Plus className="mr-1.5 h-4 w-4" />
              {t.team.addProfessional}
            </Button>
          </div>
        ) : (
          profs.map((p) => (
            <ProfCard key={p.id} prof={p} onUpdated={handleUpdated} />
          ))
        )}
      </div>
    </div>
  );
}
