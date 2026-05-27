'use client';

// Lista de profesionales del equipo en formato tabla.
// Click en fila → /dashboard/team/{id} con el detalle completo.

import { useState, useEffect, useCallback, useMemo } from 'react';
import { useRouter } from 'next/navigation';
import { Plus, Search, Users, Phone, Mail } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { professionals as profsApi, auth, Professional } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';
import { NewProfModal } from '@/components/team/NewProfModal';

const COLORS = ['#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#3b82f6', '#ef4444', '#14b8a6'];

export default function TeamPage() {
  const t = useTranslations();
  const router = useRouter();

  const [profs, setProfs] = useState<Professional[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [showArchived, setShowArchived] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [tenantCountry, setTenantCountry] = useState<string | undefined>(undefined);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await profsApi.list(showArchived);
      setProfs(res.data ?? []);
    } catch {
      setProfs([]);
    } finally {
      setLoading(false);
    }
  }, [showArchived]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    auth
      .me()
      .then(({ tenant }) => setTenantCountry(tenant.country))
      .catch(() => {});
  }, []);

  // Debounce búsqueda para suavizar el filtro en cliente.
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 300);
    return () => clearTimeout(timer);
  }, [search]);

  // Filtrado en cliente por nombre / especialidad — el endpoint no acepta search.
  const filtered = useMemo(() => {
    const q = debouncedSearch.trim().toLowerCase();
    if (!q) return profs;
    return profs.filter((p) => {
      const name = p.name.toLowerCase();
      const spec = (p.specialty ?? '').toLowerCase();
      return name.includes(q) || spec.includes(q);
    });
  }, [profs, debouncedSearch]);

  function handleCreated(p: Professional) {
    setProfs((prev) => [p, ...prev]);
    setShowForm(false);
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-neutral-900">{t.team.title}</h1>
          <p className="mt-0.5 text-sm text-neutral-500">{t.team.description}</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 text-sm text-neutral-500">
            <Users className="h-4 w-4" />
            <span>{t.team.profsCount.replace('{n}', String(profs.length))}</span>
          </div>
          <Button size="sm" onClick={() => setShowForm(true)}>
            <Plus className="mr-1.5 h-4 w-4" />
            {t.team.addProfessional}
          </Button>
        </div>
      </div>

      {/* Búsqueda + filtros */}
      <div className="mb-4 flex items-center justify-between gap-3">
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
          <input
            type="text"
            placeholder={t.team.searchPlaceholder}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-neutral-200 bg-white py-2 pl-9 pr-3 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
          />
        </div>
        <label className="flex cursor-pointer select-none items-center gap-2 text-sm text-neutral-600">
          <input
            type="checkbox"
            checked={showArchived}
            onChange={(e) => setShowArchived(e.target.checked)}
            className="h-4 w-4 rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
          />
          {t.team.showArchived}
        </label>
      </div>

      {/* Contenido */}
      {loading ? (
        <div className="flex justify-center py-16">
          <Spinner size="lg" />
        </div>
      ) : filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-neutral-200 py-16 text-center">
          <Users className="mb-3 h-12 w-12 text-neutral-200" />
          <p className="text-sm font-medium text-neutral-700 mb-1">
            {debouncedSearch ? t.team.noTeamSearch : t.team.noTeam}
          </p>
          <p className="mb-4 text-xs text-neutral-500">
            {debouncedSearch ? t.team.noTeamSearchDesc : t.team.noTeamDesc}
          </p>
          {!debouncedSearch && (
            <Button size="sm" onClick={() => setShowForm(true)}>
              <Plus className="mr-1.5 h-4 w-4" />
              {t.team.addProfessional}
            </Button>
          )}
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-neutral-200 bg-white">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-neutral-100 bg-neutral-50 text-left text-xs font-medium uppercase tracking-wide text-neutral-500">
                <th className="px-4 py-3">{t.team.colName}</th>
                <th className="px-4 py-3">{t.team.colSpecialty}</th>
                <th className="px-4 py-3">{t.team.colContact}</th>
                <th className="px-4 py-3">{t.team.colStatus}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-100">
              {filtered.map((p) => (
                <tr
                  key={p.id}
                  className={`cursor-pointer transition-colors hover:bg-neutral-50 ${
                    p.is_archived ? 'opacity-60' : ''
                  }`}
                  onClick={() => router.push(`/dashboard/team/${p.id}`)}
                >
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-3">
                      <div
                        className={`flex h-9 w-9 flex-shrink-0 items-center justify-center rounded-full text-sm font-semibold text-white ${
                          p.is_archived ? 'grayscale' : ''
                        }`}
                        style={{ backgroundColor: p.color || COLORS[0] }}
                      >
                        {p.name[0]?.toUpperCase() ?? '?'}
                      </div>
                      <span className="font-medium text-neutral-900">{p.name}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-neutral-600">
                    {p.specialty ?? <span className="text-neutral-300">—</span>}
                  </td>
                  <td className="px-4 py-3 text-neutral-600">
                    <div className="flex flex-col gap-0.5">
                      {p.phone && (
                        <span className="flex items-center gap-1.5">
                          <Phone className="h-3 w-3 text-neutral-400" />+{p.phone}
                        </span>
                      )}
                      {p.email && (
                        <span className="flex items-center gap-1.5">
                          <Mail className="h-3 w-3 text-neutral-400" />
                          {p.email}
                        </span>
                      )}
                      {!p.phone && !p.email && (
                        <span className="text-neutral-300">—</span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    {p.is_archived ? (
                      <Badge variant="default">{t.team.archived}</Badge>
                    ) : (
                      <Badge variant={p.is_active ? 'success' : 'default'}>
                        {p.is_active ? t.common.active : t.common.inactive}
                      </Badge>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Modal de nuevo profesional */}
      {showForm && (
        <NewProfModal
          onCreated={handleCreated}
          onClose={() => setShowForm(false)}
          tenantCountry={tenantCountry}
        />
      )}
    </div>
  );
}
