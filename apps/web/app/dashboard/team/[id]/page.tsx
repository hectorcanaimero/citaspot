'use client';

// Página de detalle de un profesional del equipo.
// Layout 2 columnas (perfil + tabs) — mismo patrón que /dashboard/clients/[id].

import { useState, useEffect, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import {
  ArrowLeft,
  Phone,
  Mail,
  AlertCircle,
  Pencil,
  X,
  Archive,
  ArchiveRestore,
  Users,
  MessageCircle,
} from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import {
  professionals as profsApi,
  auth,
  APIError,
  Professional,
  Customer,
  Schedule,
} from '@/lib/api';
import {
  PhoneInput,
  validatePhone,
  CountryCode,
  PHONE_COUNTRIES,
} from '@/components/ui/PhoneInput';
import { useTranslations, useDateLocale } from '@/lib/i18n';
import { format } from 'date-fns';
import {
  ScheduleEditor,
  scheduleToDayMap,
  defaultDays,
  DayMap,
} from '@/components/team/ScheduleEditor';
import { ServicesEditor } from '@/components/team/ServicesEditor';
import { BlocksEditor } from '@/components/team/BlocksEditor';

const COLORS = ['#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#3b82f6', '#ef4444', '#14b8a6'];

// ── Modal genérico ────────────────────────────────────────────────────────────

interface ModalProps {
  title: string;
  onClose: () => void;
  children: React.ReactNode;
}

function Modal({ title, onClose, children }: ModalProps) {
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

// ── Form de edición de perfil ─────────────────────────────────────────────────

interface EditProfileFormProps {
  prof: Professional;
  tenantCountry?: string;
  onSaved: (updated: Professional) => void;
  onClose: () => void;
}

function EditProfileForm({ prof, tenantCountry, onSaved, onClose }: EditProfileFormProps) {
  const t = useTranslations();
  const [name, setName] = useState(prof.name);
  const [specialty, setSpecialty] = useState(prof.specialty ?? '');
  const [email, setEmail] = useState(prof.email ?? '');
  const [phone, setPhone] = useState(prof.phone ?? '');
  const [color, setColor] = useState(prof.color || COLORS[0]);
  const [phoneError, setPhoneError] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;

    if (phone) {
      let detected: CountryCode = 'DO';
      for (const code of Object.keys(PHONE_COUNTRIES) as CountryCode[]) {
        if (phone.startsWith(PHONE_COUNTRIES[code].prefix)) {
          detected = code;
          break;
        }
      }
      const local = phone.slice(PHONE_COUNTRIES[detected].prefix.length);
      if (!validatePhone(detected, local)) {
        setPhoneError(t.booking.phoneInvalid);
        return;
      }
    }
    setPhoneError('');

    setSaving(true);
    setError('');
    try {
      const updated = await profsApi.update(prof.id, {
        name: name.trim(),
        specialty: specialty.trim() || undefined,
        email: email.trim() || undefined,
        phone: phone || undefined,
        color,
      });
      onSaved(updated);
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.team.saveError);
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-3">
      <Input
        label={t.team.nameLabel}
        placeholder={t.team.namePlaceholder}
        value={name}
        onChange={(e) => setName(e.target.value)}
        required
      />
      <Input
        label={t.team.specialtyLabel}
        placeholder={t.team.specialtyPlaceholder}
        value={specialty}
        onChange={(e) => setSpecialty(e.target.value)}
      />
      <Input
        label={t.team.email}
        type="email"
        placeholder={t.team.emailPlaceholder}
        value={email}
        onChange={(e) => setEmail(e.target.value)}
      />
      <PhoneInput
        label={t.team.phoneLabel}
        defaultCountry={tenantCountry}
        value={phone}
        onChange={(full) => {
          setPhone(full);
          setPhoneError('');
        }}
        error={phoneError}
      />
      <div className="flex flex-col gap-1.5">
        <span className="text-sm font-medium text-neutral-700">{t.team.colorLabel}</span>
        <div className="flex gap-2">
          {COLORS.map((c) => (
            <button
              key={c}
              type="button"
              onClick={() => setColor(c)}
              className={`h-7 w-7 rounded-full transition-transform ${
                color === c ? 'ring-2 ring-offset-2 ring-primary-500 scale-110' : ''
              }`}
              style={{ backgroundColor: c }}
            />
          ))}
        </div>
      </div>
      {error && <p className="rounded-md bg-red-50 px-3 py-1.5 text-sm text-red-700">{error}</p>}
      <div className="flex justify-end gap-2 pt-1">
        <button
          type="button"
          onClick={onClose}
          className="rounded-lg border border-neutral-200 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-50 transition-colors"
        >
          {t.common.cancel}
        </button>
        <Button type="submit" size="sm" loading={saving} disabled={!name.trim()}>
          {t.common.save}
        </Button>
      </div>
    </form>
  );
}

// ── Form de edición de bio ────────────────────────────────────────────────────

interface EditBioFormProps {
  profId: string;
  initialBio: string;
  onSaved: (updated: Professional) => void;
  onClose: () => void;
}

function EditBioForm({ profId, initialBio, onSaved, onClose }: EditBioFormProps) {
  const t = useTranslations();
  const [bio, setBio] = useState(initialBio);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError('');
    try {
      const updated = await profsApi.update(profId, { bio: bio.trim() });
      onSaved(updated);
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.team.saveError);
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-3">
      <textarea
        rows={6}
        placeholder={t.team.bioPlaceholder}
        value={bio}
        onChange={(e) => setBio(e.target.value)}
        className="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
      />
      {error && <p className="rounded-md bg-red-50 px-3 py-1.5 text-sm text-red-700">{error}</p>}
      <div className="flex justify-end gap-2 pt-1">
        <button
          type="button"
          onClick={onClose}
          className="rounded-lg border border-neutral-200 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-50 transition-colors"
        >
          {t.common.cancel}
        </button>
        <Button type="submit" size="sm" loading={saving}>
          {t.common.save}
        </Button>
      </div>
    </form>
  );
}

// ── Tab: Clientes atendidos ───────────────────────────────────────────────────

interface CustomersTabProps {
  profId: string;
}

function CustomersTab({ profId }: CustomersTabProps) {
  const t = useTranslations();
  const dateLocale = useDateLocale();
  const router = useRouter();
  const [list, setList] = useState<Customer[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    profsApi
      .listCustomers(profId)
      .then((res) => {
        if (!cancelled) setList(res.data ?? []);
      })
      .catch(() => {
        if (!cancelled) setList([]);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [profId]);

  if (loading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    );
  }

  if (list.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <Users className="mb-3 h-10 w-10 text-neutral-200" />
        <p className="text-sm font-medium text-neutral-500">{t.team.noAttendedCustomers}</p>
        <p className="mt-1 text-xs text-neutral-400">{t.team.noAttendedCustomersDesc}</p>
      </div>
    );
  }

  return (
    <table className="w-full text-sm">
      <thead>
        <tr className="border-b border-neutral-100 bg-neutral-50 text-left text-xs font-medium uppercase tracking-wide text-neutral-500">
          <th className="px-4 py-3">{t.clients.colName}</th>
          <th className="px-4 py-3">{t.clients.colContact}</th>
          <th className="px-4 py-3 text-center">{t.clients.colVisits}</th>
          <th className="px-4 py-3 text-center">{t.clients.colWhatsApp}</th>
          <th className="px-4 py-3">{t.clients.colRegistration}</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-neutral-100">
        {list.map((c) => (
          <tr
            key={c.id}
            className="cursor-pointer transition-colors hover:bg-neutral-50"
            onClick={() => router.push(`/dashboard/clients/${c.id}`)}
          >
            <td className="px-4 py-3">
              <div className="flex items-center gap-3">
                <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 text-xs font-semibold text-primary-700">
                  {c.name.charAt(0).toUpperCase()}
                </div>
                <span className="font-medium text-neutral-900">{c.name}</span>
              </div>
            </td>
            <td className="px-4 py-3 text-neutral-600">
              <div className="flex flex-col gap-0.5">
                {c.phone && (
                  <span className="flex items-center gap-1.5">
                    <Phone className="h-3 w-3 text-neutral-400" />
                    {c.phone}
                  </span>
                )}
                {c.email && (
                  <span className="flex items-center gap-1.5">
                    <Mail className="h-3 w-3 text-neutral-400" />
                    {c.email}
                  </span>
                )}
              </div>
            </td>
            <td className="px-4 py-3 text-center">
              <Badge variant={c.total_visits > 0 ? 'primary' : 'default'}>{c.total_visits}</Badge>
            </td>
            <td className="px-4 py-3 text-center">
              {c.wa_opt_in ? (
                <MessageCircle className="mx-auto h-4 w-4 text-emerald-500" />
              ) : (
                <span className="text-neutral-300">—</span>
              )}
            </td>
            <td className="px-4 py-3 text-neutral-500">
              {format(new Date(c.created_at), 'd MMM yyyy', { locale: dateLocale })}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

// ── Tab: Horarios ─────────────────────────────────────────────────────────────

function ScheduleTab({ profId }: { profId: string }) {
  const [scheduleMap, setScheduleMap] = useState<DayMap | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    profsApi
      .getSchedule(profId)
      .then((res: { data: Schedule[] }) => {
        if (!cancelled) setScheduleMap(scheduleToDayMap(res.data ?? []));
      })
      .catch(() => {
        if (!cancelled) setScheduleMap(defaultDays());
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [profId]);

  if (loading || !scheduleMap) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="p-5">
      <ScheduleEditor profId={profId} initial={scheduleMap} />
    </div>
  );
}

// ── Página principal ──────────────────────────────────────────────────────────

type TabKey = 'customers' | 'schedule' | 'blocks' | 'services';

export default function ProfessionalProfilePage() {
  const t = useTranslations();
  const router = useRouter();
  const { id } = useParams<{ id: string }>();

  const [prof, setProf] = useState<Professional | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tenantCountry, setTenantCountry] = useState<string | undefined>(undefined);
  const [showEditProfile, setShowEditProfile] = useState(false);
  const [showEditBio, setShowEditBio] = useState(false);
  const [archiving, setArchiving] = useState(false);
  const [activeTab, setActiveTab] = useState<TabKey>('customers');

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // El endpoint /me ya trae datos del tenant; lo pedimos en paralelo.
      // No hay un GET /professionals/:id en este momento — derivamos el prof
      // desde la lista completa (incluye archivados para soportar acceso directo
      // a un profesional archivado).
      const [res, meRes] = await Promise.all([
        profsApi.list(true),
        auth.me().catch(() => null),
      ]);
      const found = (res.data ?? []).find((p) => p.id === id) ?? null;
      if (!found) {
        setError(t.team.notFound);
      } else {
        setProf(found);
      }
      if (meRes?.tenant?.country) setTenantCountry(meRes.tenant.country);
    } catch {
      setError(t.team.notFound);
    } finally {
      setLoading(false);
    }
  }, [id, t.team.notFound]);

  useEffect(() => {
    load();
  }, [load]);

  async function toggleArchive() {
    if (!prof) return;
    if (!prof.is_archived && !window.confirm(t.team.confirmArchive)) return;
    setArchiving(true);
    try {
      const updated = await profsApi.update(prof.id, { is_archived: !prof.is_archived });
      setProf(updated);
    } catch {
      /* ignorar */
    } finally {
      setArchiving(false);
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <Spinner size="lg" />
          <p className="text-sm text-neutral-500">{t.common.loading}</p>
        </div>
      </div>
    );
  }

  if (error || !prof) {
    return (
      <div className="flex min-h-[60vh] flex-col items-center justify-center gap-4 p-6">
        <AlertCircle className="h-12 w-12 text-neutral-300" />
        <p className="text-sm font-medium text-neutral-600">{error ?? t.team.notFound}</p>
        <button
          type="button"
          onClick={() => router.push('/dashboard/team')}
          className="rounded-lg border border-neutral-200 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-50 transition-colors"
        >
          {t.team.back}
        </button>
      </div>
    );
  }

  const tabsConfig: { key: TabKey; label: string }[] = [
    { key: 'customers', label: t.team.tabCustomers },
    { key: 'schedule', label: t.team.tabSchedule },
    { key: 'blocks', label: t.team.tabBlocks },
    { key: 'services', label: t.team.tabServices },
  ];

  return (
    <div className="mx-auto max-w-7xl p-6">
      {/* Top bar */}
      <button
        type="button"
        onClick={() => router.push('/dashboard/team')}
        className="mb-6 flex items-center gap-1.5 text-sm text-neutral-500 hover:text-neutral-800 transition-colors"
      >
        <ArrowLeft className="h-4 w-4" />
        {t.team.back}
      </button>

      <div className="lg:grid lg:grid-cols-[340px_1fr] lg:gap-6">
        {/* ── Columna izquierda ────────────────────────────────────────────── */}
        <div className="mb-6 space-y-4 lg:sticky lg:top-6 lg:mb-0 lg:self-start">
          {/* Perfil */}
          <div className="overflow-hidden rounded-xl border border-neutral-200 bg-white">
            <div className="flex flex-col items-center p-6">
              <div
                className={`flex h-20 w-20 flex-shrink-0 items-center justify-center rounded-full text-3xl font-bold text-white ${
                  prof.is_archived ? 'grayscale' : ''
                }`}
                style={{ backgroundColor: prof.color || COLORS[0] }}
              >
                {prof.name[0]?.toUpperCase() ?? '?'}
              </div>
              <div className="mt-4 text-center">
                <h1 className="text-lg font-semibold text-neutral-900">{prof.name}</h1>
                {prof.specialty && (
                  <Badge variant="default" className="mt-1">
                    {prof.specialty}
                  </Badge>
                )}
                <div className="mt-2 flex items-center justify-center gap-2">
                  {prof.is_archived ? (
                    <Badge variant="default">{t.team.archived}</Badge>
                  ) : (
                    <Badge variant={prof.is_active ? 'success' : 'default'}>
                      {prof.is_active ? t.common.active : t.common.inactive}
                    </Badge>
                  )}
                </div>
              </div>
              <div className="mt-4 flex w-full flex-col gap-2">
                {prof.phone && (
                  <span className="flex items-center gap-2 text-sm text-neutral-600">
                    <Phone className="h-4 w-4 flex-shrink-0 text-neutral-400" />+{prof.phone}
                  </span>
                )}
                {prof.email && (
                  <span className="flex items-center gap-2 text-sm text-neutral-600 break-all">
                    <Mail className="h-4 w-4 flex-shrink-0 text-neutral-400" />
                    {prof.email}
                  </span>
                )}
              </div>
              <div className="mt-4 flex w-full flex-col gap-2">
                <button
                  type="button"
                  onClick={() => setShowEditProfile(true)}
                  className="flex w-full items-center justify-center gap-1.5 rounded-lg border border-neutral-200 px-3 py-2 text-sm text-neutral-700 hover:bg-neutral-50 transition-colors"
                >
                  <Pencil className="h-3.5 w-3.5" />
                  {t.team.editProfile}
                </button>
                <button
                  type="button"
                  onClick={toggleArchive}
                  disabled={archiving}
                  className="flex w-full items-center justify-center gap-1.5 rounded-lg border border-neutral-200 px-3 py-2 text-sm text-neutral-700 hover:bg-neutral-50 transition-colors disabled:opacity-50"
                >
                  {prof.is_archived ? (
                    <>
                      <ArchiveRestore className="h-3.5 w-3.5" />
                      {t.team.restore}
                    </>
                  ) : (
                    <>
                      <Archive className="h-3.5 w-3.5" />
                      {t.team.archive}
                    </>
                  )}
                </button>
              </div>
            </div>
          </div>

          {/* BIO */}
          <div className="overflow-hidden rounded-xl border border-neutral-200 bg-white">
            <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-3">
              <h2 className="text-sm font-semibold text-neutral-700">{t.team.bio}</h2>
              <button
                type="button"
                onClick={() => setShowEditBio(true)}
                className="rounded-md p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 transition-colors"
                title={t.team.editBio}
              >
                <Pencil className="h-3.5 w-3.5" />
              </button>
            </div>
            <div className="px-4 py-4">
              {prof.bio ? (
                <p className="whitespace-pre-wrap text-sm text-neutral-700">{prof.bio}</p>
              ) : (
                <p className="text-sm text-neutral-400">{t.team.bioEmpty}</p>
              )}
            </div>
          </div>
        </div>

        {/* ── Columna derecha ───────────────────────────────────────────────── */}
        <div>
          {/* Tab bar */}
          <div className="sticky top-0 z-10 rounded-t-xl border border-b-0 border-neutral-200 bg-white">
            <div className="flex border-b border-neutral-200">
              {tabsConfig.map((tab) => (
                <button
                  key={tab.key}
                  type="button"
                  onClick={() => setActiveTab(tab.key)}
                  className={`relative px-4 py-3 text-sm font-medium transition-colors ${
                    activeTab === tab.key
                      ? 'border-b-2 border-primary-600 font-semibold text-primary-600'
                      : 'text-neutral-500 hover:text-neutral-700'
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>

          {/* Tab content */}
          <div className="overflow-hidden rounded-b-xl border border-t-0 border-neutral-200 bg-white">
            {activeTab === 'customers' && <CustomersTab profId={prof.id} />}
            {activeTab === 'schedule' && <ScheduleTab profId={prof.id} />}
            {activeTab === 'blocks' && <BlocksEditor profId={prof.id} />}
            {activeTab === 'services' && (
              <div className="p-5">
                <ServicesEditor profId={prof.id} />
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Modales */}
      {showEditProfile && (
        <Modal title={t.team.editProfile} onClose={() => setShowEditProfile(false)}>
          <EditProfileForm
            prof={prof}
            tenantCountry={tenantCountry}
            onSaved={(updated) => {
              setProf(updated);
              setShowEditProfile(false);
            }}
            onClose={() => setShowEditProfile(false)}
          />
        </Modal>
      )}

      {showEditBio && (
        <Modal title={t.team.editBio} onClose={() => setShowEditBio(false)}>
          <EditBioForm
            profId={prof.id}
            initialBio={prof.bio ?? ''}
            onSaved={(updated) => {
              setProf(updated);
              setShowEditBio(false);
            }}
            onClose={() => setShowEditBio(false)}
          />
        </Modal>
      )}
    </div>
  );
}
