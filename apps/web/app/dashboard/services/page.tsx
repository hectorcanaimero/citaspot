'use client';

import { useState, useEffect } from 'react';
import { Plus, Pencil, Clock, DollarSign, Trash2 } from 'lucide-react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { Input } from '@/components/ui/input';
import { AlertDialog } from '@/components/ui/alert-dialog';
import { services, Service, ServiceInput } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

const EMPTY: ServiceInput = {
  name: '',
  description: '',
  duration_min: 30,
  price: undefined,
  currency: 'USD',
  buffer_min: 0,
  is_active: true,
};

export default function ServicesPage() {
  const t = useTranslations();
  const [list, setList]           = useState<Service[]>([]);
  const [loading, setLoading]     = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing]     = useState<Service | null>(null);
  const [form, setForm]           = useState<ServiceInput>(EMPTY);
  const [saving, setSaving]       = useState(false);
  const [error, setError]         = useState('');
  const [deleting, setDeleting]   = useState<string | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Service | null>(null);

  async function load() {
    setLoading(true);
    try {
      const res = await services.list();
      setList(res.data ?? []);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, []);

  function openCreate() {
    setEditing(null);
    setForm(EMPTY);
    setError('');
    setShowModal(true);
  }

  function openEdit(svc: Service) {
    setEditing(svc);
    setForm({
      name:         svc.name,
      description:  svc.description ?? '',
      duration_min: svc.duration_min,
      price:        svc.price ?? undefined,
      currency:     svc.currency ?? 'USD',
      buffer_min:   0,
      is_active:    svc.is_active,
    });
    setError('');
    setShowModal(true);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError('');
    try {
      const payload: ServiceInput = {
        ...form,
        price:        form.price ? Number(form.price) : undefined,
        duration_min: Number(form.duration_min),
        buffer_min:   Number(form.buffer_min ?? 0),
      };
      if (editing) {
        await services.update(editing.id, payload);
      } else {
        await services.create(payload);
      }
      setShowModal(false);
      load();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t.services.saveError);
    } finally {
      setSaving(false);
    }
  }

  async function toggleActive(svc: Service) {
    try {
      await services.update(svc.id, { is_active: !svc.is_active });
      load();
    } catch { /* ignorar */ }
  }

  async function executeDelete(svc: Service) {
    setDeleting(svc.id);
    try {
      await services.delete(svc.id);
      load();
    } catch {
      alert(t.services.deleteError);
    } finally {
      setDeleting(null);
    }
  }

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-neutral-900">{t.services.title}</h1>
          <p className="mt-0.5 text-sm text-neutral-500">{t.services.description}</p>
        </div>
        <Button onClick={openCreate}>
          <Plus className="mr-1.5 h-4 w-4" />
          {t.services.newService}
        </Button>
      </div>

      {loading ? (
        <div className="flex justify-center py-16"><Spinner size="lg" /></div>
      ) : list.length === 0 ? (
        <Card className="flex flex-col items-center justify-center py-16 text-center">
          <p className="text-sm font-medium text-neutral-500">{t.services.noServices}</p>
          <p className="mt-1 text-xs text-neutral-400">{t.services.noServicesDesc}</p>
          <Button className="mt-4" onClick={openCreate}>
            <Plus className="mr-1.5 h-4 w-4" />{t.services.newService}
          </Button>
        </Card>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {list.map((svc) => (
            <Card key={svc.id} className="flex flex-col gap-3">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <p className="truncate font-medium text-neutral-900">{svc.name}</p>
                  {svc.description && (
                    <p className="mt-0.5 line-clamp-2 text-xs text-neutral-500">{svc.description}</p>
                  )}
                </div>
                <Badge variant={svc.is_active ? 'success' : 'default'}>
                  {svc.is_active ? t.common.active : t.common.inactive}
                </Badge>
              </div>

              <div className="flex items-center gap-4 text-sm text-neutral-600">
                <span className="flex items-center gap-1.5">
                  <Clock className="h-3.5 w-3.5 text-neutral-400" />
                  {svc.duration_min} {t.common.min}
                </span>
                {svc.price != null && (
                  <span className="flex items-center gap-1.5">
                    <DollarSign className="h-3.5 w-3.5 text-neutral-400" />
                    {svc.price.toFixed(0)} {svc.currency}
                  </span>
                )}
              </div>

              <div className="flex items-center gap-2 border-t border-neutral-100 pt-3">
                <Button variant="ghost" size="sm" onClick={() => openEdit(svc)}>
                  <Pencil className="mr-1 h-3.5 w-3.5" />{t.common.edit}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setDeleteTarget(svc)}
                  disabled={deleting === svc.id}
                  className="text-red-500 hover:text-red-700 hover:bg-red-50"
                >
                  <Trash2 className="mr-1 h-3.5 w-3.5" />
                  {t.common.delete}
                </Button>
                <button
                  onClick={() => toggleActive(svc)}
                  className="ml-auto text-xs text-neutral-400 hover:text-neutral-600 transition-colors"
                >
                  {svc.is_active ? t.common.deactivate : t.common.activate}
                </button>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Dialog confirmar eliminacion */}
      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => { if (!open) setDeleteTarget(null); }}
        title={t.common.delete}
        description={t.services.confirmDelete}
        cancelLabel={t.common.cancel}
        confirmLabel={t.common.delete}
        onConfirm={() => {
          if (deleteTarget) executeDelete(deleteTarget);
        }}
      />

      {/* Modal crear/editar */}
      {showModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-md rounded-2xl bg-white shadow-xl">
            <div className="border-b border-neutral-100 px-6 py-4">
              <h2 className="text-base font-semibold text-neutral-900">
                {editing ? t.services.editService : t.services.newService}
              </h2>
            </div>
            <form onSubmit={handleSubmit} className="flex flex-col gap-4 p-6">
              <Input
                label={t.services.nameLabel}
                required
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder={t.services.namePlaceholder}
              />
              <Input
                label={t.services.descLabel}
                value={form.description ?? ''}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
                placeholder={t.services.descPlaceholder}
              />
              <div className="grid grid-cols-2 gap-3">
                <Input
                  label={t.services.durationLabel}
                  type="number"
                  required
                  min={5}
                  max={480}
                  value={form.duration_min}
                  onChange={(e) => setForm({ ...form, duration_min: Number(e.target.value) })}
                />
                <Input
                  label={t.services.priceLabel}
                  type="number"
                  min={0}
                  step="0.01"
                  value={form.price ?? ''}
                  onChange={(e) => setForm({ ...form, price: e.target.value ? Number(e.target.value) : undefined })}
                  placeholder="0.00"
                />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <Input
                  label={t.services.currencyLabel}
                  value={form.currency ?? 'USD'}
                  onChange={(e) => setForm({ ...form, currency: e.target.value })}
                  placeholder="USD"
                />
                <Input
                  label={t.services.bufferLabel}
                  type="number"
                  min={0}
                  max={120}
                  value={form.buffer_min ?? 0}
                  onChange={(e) => setForm({ ...form, buffer_min: Number(e.target.value) })}
                />
              </div>
              <label className="flex cursor-pointer items-center gap-2 text-sm text-neutral-700">
                <input
                  type="checkbox"
                  checked={form.is_active ?? true}
                  onChange={(e) => setForm({ ...form, is_active: e.target.checked })}
                  className="rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                />
                {t.services.activeLabel}
              </label>

              {error && (
                <p className="rounded-md bg-red-50 px-3 py-2 text-sm text-red-600">{error}</p>
              )}

              <div className="flex justify-end gap-3 border-t border-neutral-100 pt-2">
                <Button type="button" variant="ghost" onClick={() => setShowModal(false)} disabled={saving}>
                  {t.common.cancel}
                </Button>
                <Button type="submit" loading={saving}>
                  {editing ? t.common.saveChanges : t.services.createService}
                </Button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
