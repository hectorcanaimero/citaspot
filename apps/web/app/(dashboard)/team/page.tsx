'use client';

// Página de gestión de equipo: profesionales y sus horarios semanales.
import { useState, useEffect } from 'react';
import { Plus, ChevronDown, ChevronUp, Pencil, Check } from 'lucide-react';
import { Button }  from '@/components/ui/button';
import { Input }   from '@/components/ui/input';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge }   from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { professionals as profsApi, Professional, Schedule, ProfessionalInput, APIError } from '@/lib/api';
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
      const schedules = DAYS_DOW.filter((dow) => days[dow].is_active).map((dow) => ({
        day_of_week: dow,
        start_time:  days[dow].start_time,
        end_time:    days[dow].end_time,
        is_active:   true,
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
            <button
              type="button"
              onClick={() => toggle(dow)}
              className={`relative h-5 w-9 flex-shrink-0 rounded-full transition-colors duration-200 ${
                day.is_active ? 'bg-primary-600' : 'bg-neutral-200'
              }`}
            >
              <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white shadow transition-transform duration-200 ${
                day.is_active ? 'translate-x-4' : 'translate-x-0.5'
              }`} />
            </button>
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

// ── Componente de profesional ─────────────────────────────────────────────────

function ProfCard({ prof, onUpdated }: { prof: Professional; onUpdated: (p: Professional) => void }) {
  const t = useTranslations();
  const [expanded,   setExpanded]  = useState(false);
  const [editing,    setEditing]   = useState(false);
  const [editName,   setEditName]  = useState(prof.name);
  const [editSpec,   setEditSpec]  = useState(prof.specialty ?? '');
  const [saving,     setSaving]    = useState(false);
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

  return (
    <Card>
      <div className="flex items-center gap-3">
        {/* Avatar */}
        <div
          className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full text-white text-sm font-semibold"
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
          <Badge variant={prof.is_active ? 'success' : 'default'} className="text-xs">
            {prof.is_active ? t.common.active : t.common.inactive}
          </Badge>

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
            onClick={toggleExpand}
            className="rounded-md p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 transition-colors"
            title={t.team.viewSchedule}
          >
            {expanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
          </button>
        </div>
      </div>

      {/* Horarios */}
      {expanded && (
        <div className="mt-4 border-t border-neutral-100 pt-4">
          <p className="mb-3 text-xs font-medium text-neutral-500 uppercase tracking-wide">{t.team.weeklySchedule}</p>
          {loadingSch ? (
            <div className="flex justify-center py-4"><Spinner /></div>
          ) : schedules ? (
            <ScheduleEditor profId={prof.id} initial={schedules} />
          ) : null}
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
  const [profs,    setProfs]    = useState<Professional[]>([]);
  const [loading,  setLoading]  = useState(true);
  const [showForm, setShowForm] = useState(false);

  useEffect(() => {
    profsApi.list()
      .then((res) => setProfs(res.data))
      .catch(() => { /* ignorar */ })
      .finally(() => setLoading(false));
  }, []);

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
        <Button size="sm" onClick={() => setShowForm((v) => !v)}>
          <Plus className="mr-1.5 h-4 w-4" />
          {t.team.addProfessional}
        </Button>
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
