'use client';

import { useState, useEffect, useCallback } from 'react';
import { AlertCircle, ChevronDown, ChevronUp, FileText, Paperclip } from 'lucide-react';
import { Spinner } from '@/components/ui/spinner';
import { clinicalNotes } from '@/lib/api';
import type { ClinicalNoteWithDetails, Professional } from '@/lib/api';
import { ClinicalNoteModal } from './ClinicalNoteModal';
import { useTranslations, useDateLocale } from '@/lib/i18n';
import { format } from 'date-fns';

interface ClinicalHistorySectionProps {
  customerId: string;
  professionalId: string;
  profList: Professional[];
  notes: ClinicalNoteWithDetails[];
  onNotesChange: (notes: ClinicalNoteWithDetails[]) => void;
}

export function ClinicalHistorySection({
  customerId,
  professionalId,
  profList,
  notes,
  onNotesChange,
}: ClinicalHistorySectionProps) {
  const t = useTranslations();
  const dateLocale = useDateLocale();

  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [showModal, setShowModal] = useState(false);
  const [editNote, setEditNote] = useState<ClinicalNoteWithDetails | null>(null);

  const toggleExpand = (id: string) => {
    setExpandedId((prev) => (prev === id ? null : id));
  };

  const handleSaved = async () => {
    setShowModal(false);
    setEditNote(null);
    // Refresh notes
    try {
      const res = await clinicalNotes.listByCustomer(customerId);
      onNotesChange(res.data ?? []);
    } catch {
      // silently fail — user will see the stale list
    }
  };

  if (notes.length === 0) {
    return (
      <>
        <div className="flex flex-col items-center justify-center py-10 text-center">
          <AlertCircle className="mb-3 h-10 w-10 text-neutral-200" />
          <p className="text-sm font-medium text-neutral-500">{t.clients.noClinicalNotes}</p>
          <p className="mt-1 text-xs text-neutral-400">{t.clients.noClinicalNotesDesc}</p>
        </div>

        {showModal && (
          <ClinicalNoteModal
            customerId={customerId}
            professionalId={professionalId}
            onSaved={handleSaved}
            onClose={() => setShowModal(false)}
          />
        )}
      </>
    );
  }

  return (
    <>
      <div className="divide-y divide-neutral-100">
        {notes.map((note) => {
          const isExpanded = expandedId === note.id;
          const filesCount = note.files?.length ?? 0;

          return (
            <div key={note.id} className="px-4 py-4">
              {/* Summary row */}
              <button
                type="button"
                onClick={() => toggleExpand(note.id)}
                className="flex w-full items-start justify-between gap-3 text-left"
              >
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-neutral-900">
                      {format(new Date(note.appointment_date), 'd MMM yyyy', { locale: dateLocale })}
                    </span>
                    {filesCount > 0 && (
                      <span className="flex items-center gap-0.5 rounded bg-neutral-100 px-1.5 py-0.5 text-[10px] text-neutral-500">
                        <Paperclip className="h-2.5 w-2.5" />
                        {filesCount}
                      </span>
                    )}
                  </div>
                  <p className="mt-0.5 text-xs text-neutral-500">
                    {note.professional_name} &middot; {note.service_name}
                  </p>
                  {!isExpanded && note.assessment && (
                    <p className="mt-1 truncate text-xs text-neutral-400">{note.assessment}</p>
                  )}
                </div>
                <div className="flex-shrink-0 pt-0.5">
                  {isExpanded
                    ? <ChevronUp className="h-4 w-4 text-neutral-400" />
                    : <ChevronDown className="h-4 w-4 text-neutral-400" />
                  }
                </div>
              </button>

              {/* Expanded detail */}
              {isExpanded && (
                <div className="mt-3 space-y-3">
                  {note.subjective && (
                    <div>
                      <p className="text-[10px] font-semibold uppercase tracking-wide text-neutral-400">{t.clients.subjective}</p>
                      <p className="mt-0.5 whitespace-pre-wrap text-xs text-neutral-700">{note.subjective}</p>
                    </div>
                  )}
                  {note.objective && (
                    <div>
                      <p className="text-[10px] font-semibold uppercase tracking-wide text-neutral-400">{t.clients.objective}</p>
                      <p className="mt-0.5 whitespace-pre-wrap text-xs text-neutral-700">{note.objective}</p>
                    </div>
                  )}
                  {note.assessment && (
                    <div>
                      <p className="text-[10px] font-semibold uppercase tracking-wide text-neutral-400">{t.clients.assessment}</p>
                      <p className="mt-0.5 whitespace-pre-wrap text-xs text-neutral-700">{note.assessment}</p>
                    </div>
                  )}
                  {note.plan && (
                    <div>
                      <p className="text-[10px] font-semibold uppercase tracking-wide text-neutral-400">{t.clients.planLabel}</p>
                      <p className="mt-0.5 whitespace-pre-wrap text-xs text-neutral-700">{note.plan}</p>
                    </div>
                  )}

                  {/* File thumbnails */}
                  {filesCount > 0 && (
                    <div>
                      <p className="text-[10px] font-semibold uppercase tracking-wide text-neutral-400 mb-1.5">
                        {t.clients.attachFiles}
                      </p>
                      <div className="flex flex-wrap gap-2">
                        {note.files.map((f) => {
                          const isImage = f.content_type.startsWith('image/');
                          return (
                            <a
                              key={f.id}
                              href={f.file_url}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="group relative flex h-16 w-16 items-center justify-center overflow-hidden rounded-lg border border-neutral-200 bg-neutral-50 hover:border-primary-300 transition-colors"
                              title={f.description || f.file_name}
                            >
                              {isImage ? (
                                <img
                                  src={f.file_url}
                                  alt={f.description || f.file_name}
                                  className="h-full w-full object-cover"
                                />
                              ) : (
                                <FileText className="h-5 w-5 text-neutral-400 group-hover:text-primary-500" />
                              )}
                              <span className="absolute bottom-0 left-0 right-0 truncate bg-black/50 px-1 py-0.5 text-[8px] text-white">
                                {(t.clients.fileCategories as Record<string, string>)[f.category] ?? f.category}
                              </span>
                            </a>
                          );
                        })}
                      </div>
                    </div>
                  )}

                  {/* Edit button */}
                  <button
                    type="button"
                    onClick={() => {
                      setEditNote(note);
                      setShowModal(true);
                    }}
                    className="text-xs font-medium text-primary-600 hover:text-primary-700 transition-colors"
                  >
                    {t.clients.editClinicalNote}
                  </button>
                </div>
              )}
            </div>
          );
        })}
      </div>

      {showModal && (
        <ClinicalNoteModal
          customerId={customerId}
          professionalId={professionalId}
          appointmentId={editNote?.appointment_id}
          noteId={editNote?.id}
          existingNote={editNote ?? undefined}
          onSaved={handleSaved}
          onClose={() => {
            setShowModal(false);
            setEditNote(null);
          }}
        />
      )}
    </>
  );
}
