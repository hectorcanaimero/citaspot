'use client';

// Editor de asignación de servicios a un profesional.
// Extraído de app/dashboard/team/page.tsx para reutilizar en el detalle.

import { useEffect, useState } from 'react';
import { Check, X } from 'lucide-react';
import { Spinner } from '@/components/ui/spinner';
import { professionals as profsApi, services as servicesApi, Service } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

interface ServicesEditorProps {
  profId: string;
}

export function ServicesEditor({ profId }: ServicesEditorProps) {
  const t = useTranslations();

  const [assigned, setAssigned] = useState<Service[]>([]);
  const [allServices, setAllServices] = useState<Service[]>([]);
  const [loadingInit, setLoadingInit] = useState(true);
  const [togglingId, setTogglingId] = useState<string | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    Promise.all([profsApi.listServices(profId), servicesApi.list()])
      .then(([assignedRes, allRes]) => {
        setAssigned(assignedRes.data ?? []);
        setAllServices((allRes.data ?? []).filter((s) => s.is_active));
      })
      .catch(() => setError(t.team.servicesLoadError ?? 'Error cargando servicios'))
      .finally(() => setLoadingInit(false));
  }, [profId, t.team.servicesLoadError]);

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

  if (loadingInit) {
    return (
      <div className="flex justify-center py-4">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {allServices.length === 0 ? (
        <p className="text-xs text-neutral-400">
          {t.team.noServicesYet ?? 'No hay servicios creados aún'}
        </p>
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
