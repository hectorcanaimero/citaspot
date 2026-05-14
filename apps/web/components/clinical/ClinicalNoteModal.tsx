'use client';

import { useState, useRef } from 'react';
import { X, Upload, FileText, Trash2 } from 'lucide-react';
import { Spinner } from '@/components/ui/spinner';
import { clinicalNotes, clinicalFiles } from '@/lib/api';
import type { ClinicalNoteWithDetails } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

const FILE_CATEGORIES = ['xray', 'lab_result', 'photo', 'report', 'prescription', 'other'] as const;

interface PendingFile {
  file: File;
  category: typeof FILE_CATEGORIES[number];
  description: string;
}

interface ClinicalNoteModalProps {
  appointmentId?: string;
  noteId?: string;
  customerId: string;
  professionalId: string;
  existingNote?: ClinicalNoteWithDetails;
  onSaved: () => void;
  onClose: () => void;
}

export function ClinicalNoteModal({
  appointmentId,
  noteId,
  customerId,
  professionalId,
  existingNote,
  onSaved,
  onClose,
}: ClinicalNoteModalProps) {
  const t = useTranslations();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const isEdit = !!noteId && !!existingNote;

  const [subjective, setSubjective] = useState(existingNote?.subjective ?? '');
  const [objective, setObjective]   = useState(existingNote?.objective ?? '');
  const [assessment, setAssessment] = useState(existingNote?.assessment ?? '');
  const [plan, setPlan]             = useState(existingNote?.plan ?? '');
  const [pendingFiles, setPendingFiles] = useState<PendingFile[]>([]);
  const [saving, setSaving]         = useState(false);
  const [error, setError]           = useState('');

  // File staging state
  const [fileCategory, setFileCategory] = useState<typeof FILE_CATEGORIES[number]>('other');
  const [fileDescription, setFileDescription] = useState('');

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files) return;
    const newFiles: PendingFile[] = Array.from(files).map((f) => ({
      file: f,
      category: fileCategory,
      description: fileDescription,
    }));
    setPendingFiles((prev) => [...prev, ...newFiles]);
    setFileDescription('');
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  const removePendingFile = (idx: number) => {
    setPendingFiles((prev) => prev.filter((_, i) => i !== idx));
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    const files = e.dataTransfer.files;
    if (!files.length) return;
    const newFiles: PendingFile[] = Array.from(files).map((f) => ({
      file: f,
      category: fileCategory,
      description: '',
    }));
    setPendingFiles((prev) => [...prev, ...newFiles]);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError('');

    try {
      let savedNoteId = noteId;

      if (isEdit && noteId) {
        await clinicalNotes.update(noteId, {
          subjective: subjective.trim() || undefined,
          objective: objective.trim() || undefined,
          assessment: assessment.trim() || undefined,
          plan: plan.trim() || undefined,
        });
        savedNoteId = noteId;
      } else if (appointmentId) {
        const created = await clinicalNotes.create(appointmentId, {
          professional_id: professionalId,
          subjective: subjective.trim() || undefined,
          objective: objective.trim() || undefined,
          assessment: assessment.trim() || undefined,
          plan: plan.trim() || undefined,
        });
        savedNoteId = created.id;
      }

      // Upload pending files
      if (savedNoteId && pendingFiles.length > 0) {
        for (const pf of pendingFiles) {
          await clinicalFiles.upload(savedNoteId, pf.file, professionalId, pf.category, pf.description);
        }
      }

      onSaved();
    } catch {
      setError(t.clients.errorSaving);
    } finally {
      setSaving(false);
    }
  };

  const inputCls = 'w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100';
  const labelCls = 'mb-1 block text-xs font-medium text-neutral-600';

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-lg max-h-[90vh] overflow-hidden rounded-xl border border-neutral-200 bg-white shadow-xl flex flex-col">
        <div className="flex items-center justify-between border-b border-neutral-100 px-5 py-4">
          <h3 className="text-sm font-semibold text-neutral-900">
            {isEdit ? t.clients.editClinicalNote : t.clients.newClinicalNote}
          </h3>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 transition-colors"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto px-5 py-4 space-y-3">
          {/* SOAP Fields */}
          <div>
            <label className={labelCls}>{t.clients.subjective}</label>
            <textarea
              className={inputCls}
              rows={2}
              placeholder={t.clients.subjectivePlaceholder}
              value={subjective}
              onChange={(e) => setSubjective(e.target.value)}
            />
          </div>

          <div>
            <label className={labelCls}>{t.clients.objective}</label>
            <textarea
              className={inputCls}
              rows={2}
              placeholder={t.clients.objectivePlaceholder}
              value={objective}
              onChange={(e) => setObjective(e.target.value)}
            />
          </div>

          <div>
            <label className={labelCls}>{t.clients.assessment}</label>
            <textarea
              className={inputCls}
              rows={2}
              placeholder={t.clients.assessmentPlaceholder}
              value={assessment}
              onChange={(e) => setAssessment(e.target.value)}
            />
          </div>

          <div>
            <label className={labelCls}>{t.clients.planLabel}</label>
            <textarea
              className={inputCls}
              rows={2}
              placeholder={t.clients.planPlaceholder}
              value={plan}
              onChange={(e) => setPlan(e.target.value)}
            />
          </div>

          {/* File upload area */}
          <div className="border-t border-neutral-100 pt-3">
            <label className={labelCls}>{t.clients.attachFiles}</label>

            <div className="grid grid-cols-2 gap-2 mb-2">
              <div>
                <label className={labelCls}>{t.clients.fileCategory}</label>
                <select
                  className={inputCls}
                  value={fileCategory}
                  onChange={(e) => setFileCategory(e.target.value as typeof FILE_CATEGORIES[number])}
                >
                  {FILE_CATEGORIES.map((cat) => (
                    <option key={cat} value={cat}>
                      {(t.clients.fileCategories as Record<string, string>)[cat] ?? cat}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className={labelCls}>{t.clients.fileDescription}</label>
                <input
                  className={inputCls}
                  value={fileDescription}
                  onChange={(e) => setFileDescription(e.target.value)}
                  placeholder={t.common.optional}
                />
              </div>
            </div>

            <div
              onDrop={handleDrop}
              onDragOver={handleDragOver}
              onClick={() => fileInputRef.current?.click()}
              className="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed border-neutral-200 bg-neutral-50 px-4 py-5 text-center hover:border-primary-300 hover:bg-primary-50/30 transition-colors"
            >
              <Upload className="mb-1.5 h-5 w-5 text-neutral-400" />
              <p className="text-xs text-neutral-500">{t.clients.uploadFile}</p>
              <p className="mt-0.5 text-[10px] text-neutral-400">{t.clients.maxFileSize}</p>
              <input
                ref={fileInputRef}
                type="file"
                multiple
                className="hidden"
                onChange={handleFileSelect}
                accept="image/*,.pdf,.doc,.docx"
              />
            </div>

            {/* Pending files list */}
            {pendingFiles.length > 0 && (
              <div className="mt-2 space-y-1">
                {pendingFiles.map((pf, idx) => (
                  <div key={idx} className="flex items-center justify-between rounded-lg bg-neutral-50 px-3 py-2">
                    <div className="flex items-center gap-2 min-w-0">
                      <FileText className="h-3.5 w-3.5 flex-shrink-0 text-neutral-400" />
                      <span className="truncate text-xs text-neutral-700">{pf.file.name}</span>
                      <span className="flex-shrink-0 rounded bg-neutral-200 px-1.5 py-0.5 text-[10px] text-neutral-600">
                        {(t.clients.fileCategories as Record<string, string>)[pf.category] ?? pf.category}
                      </span>
                    </div>
                    <button
                      type="button"
                      onClick={() => removePendingFile(idx)}
                      className="ml-2 rounded p-0.5 text-neutral-400 hover:text-red-500 transition-colors"
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          {error && <p className="text-xs text-red-500">{error}</p>}

          <div className="flex justify-end gap-2 pt-1">
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg border border-neutral-200 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-50 transition-colors"
            >
              {t.common.cancel}
            </button>
            <button
              type="submit"
              disabled={saving || (!isEdit && !appointmentId)}
              className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 disabled:opacity-50 transition-colors"
            >
              {saving ? <Spinner size="sm" /> : t.common.save}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
