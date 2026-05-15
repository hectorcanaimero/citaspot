'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { Settings2, Kanban } from 'lucide-react';
import { Spinner } from '@/components/ui/spinner';
import {
  pipelineStages as stagesApi,
  customers as customersApi,
  PipelineStage,
  Customer,
} from '@/lib/api';
import { useTranslations } from '@/lib/i18n';
import { KanbanBoard } from '@/components/dashboard/pipeline/KanbanBoard';
import { StageSettingsModal } from '@/components/dashboard/pipeline/StageSettingsModal';

const UNASSIGNED_ID = 'unassigned';

function groupByStage(customers: Customer[], stages: PipelineStage[]): Record<string, Customer[]> {
  const map: Record<string, Customer[]> = { [UNASSIGNED_ID]: [] };
  for (const s of stages) map[s.id] = [];
  for (const c of customers) {
    const key = c.stage_id ?? UNASSIGNED_ID;
    if (!map[key]) map[key] = [];
    map[key].push(c);
  }
  return map;
}

export default function PipelinePage() {
  const t = useTranslations();

  const [stages, setStages] = useState<PipelineStage[]>([]);
  const [allCustomers, setAllCustomers] = useState<Customer[]>([]);
  const [loading, setLoading] = useState(true);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);
  const toastTimer = useRef<ReturnType<typeof setTimeout>>();

  const showToast = useCallback((message: string, type: 'success' | 'error') => {
    setToast({ message, type });
    if (toastTimer.current) clearTimeout(toastTimer.current);
    toastTimer.current = setTimeout(() => setToast(null), 3000);
  }, []);

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [stagesRes, customersRes] = await Promise.all([
        stagesApi.list(),
        customersApi.list('', 500, 0),
      ]);
      setStages((stagesRes.data ?? []).sort((a, b) => a.position - b.position));
      setAllCustomers(customersRes.data ?? []);
    } catch {
      setStages([]);
      setAllCustomers([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleMoveCustomer = useCallback(
    async (customerId: string, newStageId: string | null, stageName: string) => {
      // Optimistic update
      setAllCustomers(prev =>
        prev.map(c => (c.id === customerId ? { ...c, stage_id: newStageId } : c)),
      );

      try {
        await customersApi.updateStage(customerId, newStageId);
        const label = stageName === UNASSIGNED_ID ? t.crm.pipeline.unassigned : stageName;
        showToast(t.crm.pipeline.moveSuccess.replace('{stage}', label), 'success');
      } catch {
        // Revert on error
        loadData();
        showToast(t.crm.pipeline.moveError, 'error');
      }
    },
    [loadData, showToast, t],
  );

  const customersByStage = groupByStage(allCustomers, stages);

  if (loading) {
    return (
      <div className="flex h-[60vh] items-center justify-center">
        <Spinner size="lg" />
      </div>
    );
  }

  // Empty state — no stages yet
  if (stages.length === 0) {
    return (
      <div className="p-6">
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <Kanban className="mb-3 h-12 w-12 text-neutral-200" />
          <p className="text-sm font-medium text-neutral-500">{t.crm.pipeline.noStages}</p>
          <p className="mt-1 mb-5 text-xs text-neutral-400">{t.crm.pipeline.noStagesDesc}</p>
          <button
            onClick={() => setSettingsOpen(true)}
            className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-semibold text-white hover:bg-primary-500"
          >
            {t.crm.pipeline.stageSettings}
          </button>
        </div>
        <StageSettingsModal
          open={settingsOpen}
          onClose={() => setSettingsOpen(false)}
          onStagesChanged={loadData}
        />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col p-6">
      {/* Header */}
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-neutral-900">{t.crm.pipeline.boardTitle}</h1>
          <p className="mt-0.5 text-sm text-neutral-500">{t.crm.pipeline.boardDescription}</p>
        </div>
        <button
          onClick={() => setSettingsOpen(true)}
          className="flex items-center gap-1.5 rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-xs font-medium text-neutral-700 hover:bg-neutral-50"
        >
          <Settings2 className="h-3.5 w-3.5" />
          {t.crm.pipeline.stageSettings}
        </button>
      </div>

      {/* Board */}
      <div className="flex-1 overflow-hidden">
        <KanbanBoard
          stages={stages}
          customersByStage={customersByStage}
          unassignedTitle={t.crm.pipeline.unassigned}
          onMoveCustomer={handleMoveCustomer}
        />
      </div>

      {/* Toast */}
      {toast && (
        <div
          className={`fixed bottom-6 left-1/2 -translate-x-1/2 rounded-lg px-4 py-2.5 text-sm font-medium shadow-lg transition-all ${
            toast.type === 'success'
              ? 'bg-emerald-600 text-white'
              : 'bg-red-600 text-white'
          }`}
        >
          {toast.message}
        </div>
      )}

      {/* Settings modal */}
      <StageSettingsModal
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        onStagesChanged={loadData}
      />
    </div>
  );
}
