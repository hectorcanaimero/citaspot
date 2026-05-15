'use client';

import { useState, useRef } from 'react';
import { Plus, Pencil, Trash2, BookOpen, FileText, Upload, Loader2, AlertCircle } from 'lucide-react';
import { Button }  from '@/components/ui/button';
import { Input }   from '@/components/ui/input';
import { Card }    from '@/components/ui/card';
import { Badge }   from '@/components/ui/badge';
import { knowledge, KnowledgeDocument, APIError } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

type SourceTab = 'text' | 'file';

interface TextFormData {
  category: string;
  title: string;
  content: string;
  is_active: boolean;
}

function defaultTextForm(): TextFormData {
  return { category: 'faq', title: '', content: '', is_active: true };
}

interface Props {
  docs: KnowledgeDocument[];
  onRefresh: () => void;
}

export function KnowledgeTab({ docs, onRefresh }: Props) {
  const t = useTranslations();
  const categories = t.knowledge.categories as Record<string, string>;
  const CATEGORIES = Object.entries(categories);

  const [error,    setError]    = useState('');
  const [editing,  setEditing]  = useState<KnowledgeDocument | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [tab,      setTab]      = useState<SourceTab>('text');
  const [form,     setForm]     = useState<TextFormData>(defaultTextForm());
  const [file,     setFile]     = useState<File | null>(null);
  const [saving,   setSaving]   = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  function openCreate() {
    setEditing(null);
    setForm(defaultTextForm());
    setFile(null);
    setTab('text');
    setError('');
    setShowForm(true);
  }

  function openEdit(doc: KnowledgeDocument) {
    setEditing(doc);
    setForm({ category: doc.category, title: doc.title, content: doc.content, is_active: doc.is_active });
    setFile(null);
    setTab('text');
    setError('');
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setError('');
    setFile(null);
  }

  async function handleDelete(id: string) {
    if (!confirm(t.knowledge.confirmDelete)) return;
    try {
      await knowledge.remove(id);
      onRefresh();
    } catch {
      setError(t.knowledge.deleteError);
    }
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError('');
    try {
      if (editing) {
        await knowledge.update(editing.id, form);
        closeForm();
        onRefresh();
        return;
      }
      if (tab === 'file') {
        if (!file) { setError(t.knowledge.selectFile); return; }
        await knowledge.upload(file, form.title, form.category, form.is_active);
      } else {
        await knowledge.create(form);
      }
      closeForm();
      onRefresh();
    } catch (err) {
      setError(err instanceof APIError ? err.message : t.knowledge.saveError);
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-neutral-900">{t.knowledge.title}</h2>
          <p className="mt-0.5 text-sm text-neutral-500">{t.knowledge.description}</p>
        </div>
        <Button onClick={openCreate} size="md">
          <Plus className="h-4 w-4" />
          {t.knowledge.newDocument}
        </Button>
      </div>

      {error && !showForm && (
        <div className="mb-4 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>
      )}

      {/* Modal */}
      {showForm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <Card className="w-full max-w-lg animate-slide-up">
            <h2 className="mb-4 text-base font-semibold text-neutral-900">
              {editing ? t.knowledge.editDocument : t.knowledge.newDocument}
            </h2>

            {!editing && (
              <div className="mb-4 flex gap-1 rounded-lg bg-neutral-100 p-1">
                <button
                  type="button"
                  onClick={() => { setTab('text'); setFile(null); setError(''); }}
                  className={`flex flex-1 items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                    tab === 'text'
                      ? 'bg-white text-neutral-900 shadow-sm'
                      : 'text-neutral-500 hover:text-neutral-700'
                  }`}
                >
                  <FileText className="h-3.5 w-3.5" />
                  {t.knowledge.textTab}
                </button>
                <button
                  type="button"
                  onClick={() => { setTab('file'); setError(''); }}
                  className={`flex flex-1 items-center justify-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                    tab === 'file'
                      ? 'bg-white text-neutral-900 shadow-sm'
                      : 'text-neutral-500 hover:text-neutral-700'
                  }`}
                >
                  <Upload className="h-3.5 w-3.5" />
                  {t.knowledge.fileTab}
                </button>
              </div>
            )}

            <form onSubmit={handleSave} className="flex flex-col gap-4">
              <div className="flex flex-col gap-1.5">
                <label className="text-sm font-medium text-neutral-700">{t.knowledge.categoryLabel}</label>
                <select
                  value={form.category}
                  onChange={(e) => setForm((f) => ({ ...f, category: e.target.value }))}
                  className="h-9 w-full rounded-md border border-neutral-200 bg-white px-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
                >
                  {CATEGORIES.map(([v, l]) => (
                    <option key={v} value={v}>{l}</option>
                  ))}
                </select>
              </div>

              <Input
                label={t.knowledge.titleLabel}
                placeholder={t.knowledge.titlePlaceholder}
                value={form.title}
                onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
                required
              />

              {tab === 'text' ? (
                <div className="flex flex-col gap-1.5">
                  <label className="text-sm font-medium text-neutral-700">{t.knowledge.contentLabel}</label>
                  <textarea
                    rows={6}
                    placeholder={t.knowledge.contentPlaceholder}
                    value={form.content}
                    onChange={(e) => setForm((f) => ({ ...f, content: e.target.value }))}
                    required
                    className="w-full rounded-md border border-neutral-200 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 resize-none"
                  />
                </div>
              ) : (
                <div className="flex flex-col gap-1.5">
                  <label className="text-sm font-medium text-neutral-700">{t.knowledge.fileLabel}</label>
                  <div
                    onClick={() => fileInputRef.current?.click()}
                    className={`flex cursor-pointer flex-col items-center gap-2 rounded-lg border-2 border-dashed px-4 py-6 transition-colors ${
                      file
                        ? 'border-primary-300 bg-primary-50'
                        : 'border-neutral-200 hover:border-primary-300 hover:bg-neutral-50'
                    }`}
                  >
                    <Upload className={`h-6 w-6 ${file ? 'text-primary-500' : 'text-neutral-400'}`} />
                    {file ? (
                      <div className="text-center">
                        <p className="text-sm font-medium text-primary-700">{file.name}</p>
                        <p className="text-xs text-neutral-500">{(file.size / 1024).toFixed(0)} KB · {t.knowledge.fileClickToChange}</p>
                      </div>
                    ) : (
                      <div className="text-center">
                        <p className="text-sm text-neutral-600">{t.knowledge.fileClickToSelect}</p>
                        <p className="text-xs text-neutral-400">{t.knowledge.fileFormats}</p>
                      </div>
                    )}
                  </div>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".pdf,.docx,.doc,.xlsx,.xls,.csv"
                    className="hidden"
                    onChange={(e) => {
                      const f = e.target.files?.[0] ?? null;
                      setFile(f);
                      setError('');
                    }}
                  />
                  <p className="text-xs text-neutral-400">{t.knowledge.fileAutoExtract}</p>
                </div>
              )}

              <label className="flex items-center gap-2 text-sm text-neutral-700">
                <input
                  type="checkbox"
                  checked={form.is_active}
                  onChange={(e) => setForm((f) => ({ ...f, is_active: e.target.checked }))}
                  className="rounded border-neutral-300 text-primary-600"
                />
                {t.knowledge.activeDoc}
              </label>

              {error && <p className="text-sm text-red-600">{error}</p>}

              <div className="flex gap-3">
                <Button type="button" variant="secondary" onClick={closeForm} className="flex-1">
                  {t.common.cancel}
                </Button>
                <Button type="submit" loading={saving} className="flex-1">
                  {editing ? t.common.saveChanges : tab === 'file' ? t.knowledge.uploadDocument : t.knowledge.createDocument}
                </Button>
              </div>
            </form>
          </Card>
        </div>
      )}

      {/* Lista */}
      {docs.length === 0 ? (
        <div className="flex flex-col items-center gap-3 py-16 text-center">
          <BookOpen className="h-10 w-10 text-neutral-300" />
          <p className="text-sm text-neutral-500">{t.knowledge.noDocuments}</p>
          <Button onClick={openCreate} variant="secondary">{t.knowledge.createFirst}</Button>
        </div>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {docs.map((doc) => (
            <Card key={doc.id} className="flex flex-col gap-3">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <Badge variant={doc.is_active ? 'success' : 'default'} className="mb-1.5">
                    {categories[doc.category] ?? doc.category}
                  </Badge>
                  <p className="truncate text-sm font-medium text-neutral-900">{doc.title}</p>
                  {doc.source_type === 'file' && doc.file_name && (
                    <p className="mt-0.5 truncate text-xs text-neutral-400">
                      <FileText className="mr-0.5 inline h-3 w-3" />
                      {doc.file_name}
                    </p>
                  )}
                </div>
                <div className="flex flex-shrink-0 gap-1">
                  <button
                    aria-label={t.common.edit}
                    onClick={() => openEdit(doc)}
                    disabled={doc.status === 'processing'}
                    className="rounded p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-700 disabled:cursor-not-allowed disabled:opacity-40"
                  >
                    <Pencil className="h-3.5 w-3.5" />
                  </button>
                  <button
                    aria-label={t.common.delete}
                    onClick={() => handleDelete(doc.id)}
                    className="rounded p-1.5 text-neutral-400 hover:bg-red-50 hover:text-red-600"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>

              {doc.status === 'ready' && (
                <p className="line-clamp-3 text-xs text-neutral-500">{doc.content}</p>
              )}

              <div className="flex flex-wrap gap-1.5">
                {!doc.is_active && (
                  <Badge variant="warning" className="self-start">{t.common.inactive}</Badge>
                )}
                {doc.status === 'processing' && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-600">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    {t.knowledge.processing}
                  </span>
                )}
                {doc.status === 'error' && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-red-50 px-2 py-0.5 text-xs font-medium text-red-600">
                    <AlertCircle className="h-3 w-3" />
                    {t.knowledge.processingError}
                  </span>
                )}
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
