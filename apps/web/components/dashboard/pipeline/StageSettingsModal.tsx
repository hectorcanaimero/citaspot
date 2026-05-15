'use client';

import { useState, useEffect, useCallback } from 'react';
import { Plus, Trash2, Pencil, ChevronUp, ChevronDown, X } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { pipelineStages, PipelineStage } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

const STAGE_COLORS = [
  '#8b5cf6', '#3b82f6', '#06b6d4', '#10b981',
  '#f59e0b', '#ef4444', '#ec4899', '#6366f1',
];

interface StageSettingsModalProps {
  open: boolean;
  onClose: () => void;
  onStagesChanged: () => void;
}

export function StageSettingsModal({ open, onClose, onStagesChanged }: StageSettingsModalProps) {
  const t = useTranslations();

  const [stages, setStages] = useState<PipelineStage[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formName, setFormName] = useState('');
  const [formColor, setFormColor] = useState(STAGE_COLORS[0]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await pipelineStages.list();
      setStages((res.data ?? []).sort((a, b) => a.position - b.position));
    } catch {
      setStages([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open) load();
  }, [open, load]);

  function openCreateForm() {
    setEditingId(null);
    setFormName('');
    setFormColor(STAGE_COLORS[stages.length % STAGE_COLORS.length]);
    setShowForm(true);
  }

  function openEditForm(stage: PipelineStage) {
    setEditingId(stage.id);
    setFormName(stage.name);
    setFormColor(stage.color);
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setEditingId(null);
    setFormName('');
  }

  async function handleSave() {
    if (!formName.trim()) return;
    setSaving(true);
    try {
      if (editingId) {
        await pipelineStages.update(editingId, { name: formName.trim(), color: formColor });
      } else {
        await pipelineStages.create({ name: formName.trim(), color: formColor });
      }
      closeForm();
      await load();
      onStagesChanged();
    } catch {
      // silent
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm(t.crm.pipeline.confirmDelete)) return;
    try {
      await pipelineStages.remove(id);
      await load();
      onStagesChanged();
    } catch {
      // silent
    }
  }

  async function handleMove(index: number, direction: 'up' | 'down') {
    const newStages = [...stages];
    const swapIdx = direction === 'up' ? index - 1 : index + 1;
    if (swapIdx < 0 || swapIdx >= newStages.length) return;
    [newStages[index], newStages[swapIdx]] = [newStages[swapIdx], newStages[index]];
    setStages(newStages);
    try {
      await pipelineStages.reorder(newStages.map(s => s.id));
      onStagesChanged();
    } catch {
      await load();
    }
  }

  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="relative mx-4 max-h-[85vh] w-full max-w-lg overflow-y-auto rounded-2xl bg-white p-6 shadow-xl">
        {/* Header */}
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-neutral-900">{t.crm.pipeline.settingsTitle}</h2>
            <p className="text-sm text-neutral-500">{t.crm.pipeline.settingsDescription}</p>
          </div>
          <button onClick={onClose} className="rounded-lg p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600">
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Add button */}
        <button
          onClick={openCreateForm}
          className="mb-4 flex items-center gap-1.5 rounded-lg bg-primary-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-primary-500"
        >
          <Plus className="h-3.5 w-3.5" />
          {t.crm.pipeline.addStage}
        </button>

        {/* Inline form */}
        {showForm && (
          <Card className="mb-4 p-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-neutral-800">
                {editingId ? t.crm.pipeline.editStage : t.crm.pipeline.addStage}
              </h3>
              <button onClick={closeForm} className="text-neutral-400 hover:text-neutral-600">
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-neutral-600 mb-1">{t.crm.pipeline.nameLabel}</label>
                <input
                  type="text"
                  value={formName}
                  onChange={e => setFormName(e.target.value)}
                  placeholder={t.crm.pipeline.namePlaceholder}
                  className="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
                  onKeyDown={e => e.key === 'Enter' && handleSave()}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-neutral-600 mb-1">{t.crm.pipeline.colorLabel}</label>
                <div className="flex gap-1.5">
                  {STAGE_COLORS.map(color => (
                    <button
                      key={color}
                      onClick={() => setFormColor(color)}
                      className={`h-7 w-7 rounded-full border-2 transition-all ${
                        formColor === color ? 'border-neutral-800 scale-110' : 'border-transparent'
                      }`}
                      style={{ backgroundColor: color }}
                    />
                  ))}
                </div>
              </div>
              <button
                onClick={handleSave}
                disabled={saving || !formName.trim()}
                className="w-full rounded-lg bg-primary-600 px-4 py-2 text-sm font-semibold text-white hover:bg-primary-500 disabled:opacity-50"
              >
                {saving ? t.common.loading : t.common.save}
              </button>
            </div>
          </Card>
        )}

        {/* Stages list */}
        {loading ? (
          <div className="flex justify-center py-8"><Spinner size="lg" /></div>
        ) : stages.length === 0 ? (
          <p className="py-8 text-center text-sm text-neutral-500">{t.crm.pipeline.noStages}</p>
        ) : (
          <div className="space-y-2">
            {stages.map((stage, idx) => (
              <div
                key={stage.id}
                className="flex items-center gap-3 rounded-xl border border-neutral-200 bg-white px-3 py-2.5 hover:bg-neutral-50"
              >
                <div className="h-3.5 w-3.5 rounded-full flex-shrink-0" style={{ backgroundColor: stage.color }} />
                <span className="text-xs font-mono text-neutral-400 w-5 text-center">{idx + 1}</span>
                <span className="flex-1 text-sm font-medium text-neutral-900">{stage.name}</span>
                {stage.is_default && <Badge variant="default">{t.crm.pipeline.defaultStage}</Badge>}
                <div className="flex items-center gap-0.5">
                  <button onClick={() => handleMove(idx, 'up')} disabled={idx === 0} className="rounded p-1 text-neutral-400 hover:bg-neutral-100 disabled:opacity-30">
                    <ChevronUp className="h-4 w-4" />
                  </button>
                  <button onClick={() => handleMove(idx, 'down')} disabled={idx === stages.length - 1} className="rounded p-1 text-neutral-400 hover:bg-neutral-100 disabled:opacity-30">
                    <ChevronDown className="h-4 w-4" />
                  </button>
                  <button onClick={() => openEditForm(stage)} className="rounded p-1 text-neutral-400 hover:bg-neutral-100">
                    <Pencil className="h-4 w-4" />
                  </button>
                  <button onClick={() => handleDelete(stage.id)} className="rounded p-1 text-neutral-400 hover:bg-red-50 hover:text-red-500">
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
