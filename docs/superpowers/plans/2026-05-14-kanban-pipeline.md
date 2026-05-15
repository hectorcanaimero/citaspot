# Kanban Pipeline Board — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the pipeline stages config list with a full Kanban board where each column is a pipeline stage and each card is a customer, with drag-and-drop to move customers between stages.

**Architecture:** Frontend-only change. The backend already supports `customers.updateStage(id, stageId)` and `pipelineStages.list()`. We install `@hello-pangea/dnd` for drag-and-drop, rewrite the pipeline page as a Kanban board, and move the stages CRUD into a settings modal accessible from a gear icon.

**Tech Stack:** Next.js 14, React 18, `@hello-pangea/dnd`, Tailwind CSS, lucide-react icons, existing `lib/api.ts` client.

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `apps/web/app/dashboard/pipeline/page.tsx` | Rewrite | Kanban board page — loads stages + customers, renders board |
| `apps/web/components/dashboard/pipeline/KanbanBoard.tsx` | Create | Board layout — columns with drag-and-drop context |
| `apps/web/components/dashboard/pipeline/KanbanColumn.tsx` | Create | Single column — stage header + droppable area + customer cards |
| `apps/web/components/dashboard/pipeline/KanbanCard.tsx` | Create | Customer card — name, phone, visits, value, tags |
| `apps/web/components/dashboard/pipeline/StageSettingsModal.tsx` | Create | Modal for CRUD of pipeline stages (extracted from current page) |
| `apps/web/lib/i18n/locales/es.ts` | Modify | Add kanban-specific translation keys under `crm.pipeline` |
| `apps/web/lib/i18n/locales/en.ts` | Modify | Add kanban-specific translation keys under `crm.pipeline` |
| `apps/web/lib/i18n/locales/pt.ts` | Modify | Add kanban-specific translation keys under `crm.pipeline` |

---

### Task 1: Install @hello-pangea/dnd

**Files:**
- Modify: `apps/web/package.json`

- [ ] **Step 1: Install the dependency**

```bash
cd apps/web && npm install @hello-pangea/dnd
```

- [ ] **Step 2: Verify installation**

```bash
cd apps/web && node -e "require('@hello-pangea/dnd'); console.log('OK')"
```

Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add apps/web/package.json apps/web/package-lock.json
git commit -m "chore(web): install @hello-pangea/dnd for kanban board"
```

---

### Task 2: Add i18n keys for the Kanban board

**Files:**
- Modify: `apps/web/lib/i18n/locales/es.ts`
- Modify: `apps/web/lib/i18n/locales/en.ts`
- Modify: `apps/web/lib/i18n/locales/pt.ts`

Add these keys inside the existing `crm.pipeline` object in each locale file (after the existing keys, before the closing `}`):

- [ ] **Step 1: Add keys to es.ts**

Add after `stagesCount: '{n} etapas',` inside `crm.pipeline`:

```typescript
    // Kanban board
    boardTitle: 'Pipeline de clientes',
    boardDescription: 'Arrastrá clientes entre etapas para actualizar su estado.',
    unassigned: 'Sin asignar',
    unassignedDesc: 'Clientes sin etapa asignada',
    customerCount: '{n} clientes',
    noCustomers: 'Sin clientes en esta etapa',
    moveSuccess: 'Cliente movido a {stage}',
    moveError: 'No se pudo mover el cliente. Reintentá.',
    stageSettings: 'Configurar etapas',
    visits: '{n} visitas',
    settingsTitle: 'Configurar etapas del pipeline',
    settingsDescription: 'Agregá, editá o reordená las etapas de tu pipeline.',
    closeSettings: 'Cerrar',
```

- [ ] **Step 2: Add keys to en.ts**

Add after `stagesCount: '{n} stages',` inside `crm.pipeline`:

```typescript
    // Kanban board
    boardTitle: 'Client Pipeline',
    boardDescription: 'Drag clients between stages to update their status.',
    unassigned: 'Unassigned',
    unassignedDesc: 'Clients without a stage',
    customerCount: '{n} clients',
    noCustomers: 'No clients in this stage',
    moveSuccess: 'Client moved to {stage}',
    moveError: 'Could not move client. Try again.',
    stageSettings: 'Configure stages',
    visits: '{n} visits',
    settingsTitle: 'Configure pipeline stages',
    settingsDescription: 'Add, edit, or reorder your pipeline stages.',
    closeSettings: 'Close',
```

- [ ] **Step 3: Add keys to pt.ts**

Add after `stagesCount: '{n} etapas',` inside `crm.pipeline`:

```typescript
    // Kanban board
    boardTitle: 'Pipeline de clientes',
    boardDescription: 'Arraste clientes entre etapas para atualizar o status.',
    unassigned: 'Sem etapa',
    unassignedDesc: 'Clientes sem etapa atribuída',
    customerCount: '{n} clientes',
    noCustomers: 'Nenhum cliente nesta etapa',
    moveSuccess: 'Cliente movido para {stage}',
    moveError: 'Não foi possível mover o cliente. Tente novamente.',
    stageSettings: 'Configurar etapas',
    visits: '{n} visitas',
    settingsTitle: 'Configurar etapas do pipeline',
    settingsDescription: 'Adicione, edite ou reordene as etapas do pipeline.',
    closeSettings: 'Fechar',
```

- [ ] **Step 4: Commit**

```bash
git add apps/web/lib/i18n/locales/es.ts apps/web/lib/i18n/locales/en.ts apps/web/lib/i18n/locales/pt.ts
git commit -m "feat(web): add i18n keys for kanban pipeline board"
```

---

### Task 3: Create KanbanCard component

**Files:**
- Create: `apps/web/components/dashboard/pipeline/KanbanCard.tsx`

- [ ] **Step 1: Create the component**

```tsx
'use client';

import { Draggable } from '@hello-pangea/dnd';
import { Phone, Mail } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { Customer } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

interface KanbanCardProps {
  customer: Customer;
  index: number;
}

export function KanbanCard({ customer, index }: KanbanCardProps) {
  const t = useTranslations();
  const router = useRouter();

  return (
    <Draggable draggableId={customer.id} index={index}>
      {(provided, snapshot) => (
        <div
          ref={provided.innerRef}
          {...provided.draggableProps}
          {...provided.dragHandleProps}
          onClick={() => router.push(`/dashboard/clients/${customer.id}`)}
          className={`rounded-lg border bg-white p-3 shadow-sm transition-shadow cursor-pointer hover:shadow-md ${
            snapshot.isDragging ? 'shadow-lg ring-2 ring-primary-300' : 'border-neutral-200'
          }`}
        >
          {/* Name */}
          <p className="text-sm font-medium text-neutral-900 truncate">{customer.name}</p>

          {/* Contact */}
          <div className="mt-1.5 space-y-0.5">
            {customer.phone && (
              <div className="flex items-center gap-1.5 text-xs text-neutral-500">
                <Phone className="h-3 w-3 flex-shrink-0" />
                <span className="truncate">{customer.phone}</span>
              </div>
            )}
            {customer.email && (
              <div className="flex items-center gap-1.5 text-xs text-neutral-500">
                <Mail className="h-3 w-3 flex-shrink-0" />
                <span className="truncate">{customer.email}</span>
              </div>
            )}
          </div>

          {/* Stats row */}
          <div className="mt-2 flex items-center gap-2 text-xs text-neutral-400">
            {customer.total_visits > 0 && (
              <span>{t.crm.pipeline.visits.replace('{n}', String(customer.total_visits))}</span>
            )}
            {(customer.lifetime_value ?? 0) > 0 && (
              <span className="font-medium text-emerald-600">${customer.lifetime_value?.toFixed(0)}</span>
            )}
          </div>

          {/* Tags */}
          {customer.tags && customer.tags.length > 0 && (
            <div className="mt-2 flex flex-wrap gap-1">
              {customer.tags.slice(0, 3).map(tag => (
                <span
                  key={tag}
                  className="inline-block rounded-full bg-neutral-100 px-2 py-0.5 text-[10px] font-medium text-neutral-600"
                >
                  {tag}
                </span>
              ))}
              {customer.tags.length > 3 && (
                <span className="text-[10px] text-neutral-400">+{customer.tags.length - 3}</span>
              )}
            </div>
          )}
        </div>
      )}
    </Draggable>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/web/components/dashboard/pipeline/KanbanCard.tsx
git commit -m "feat(web): add KanbanCard component for pipeline board"
```

---

### Task 4: Create KanbanColumn component

**Files:**
- Create: `apps/web/components/dashboard/pipeline/KanbanColumn.tsx`

- [ ] **Step 1: Create the component**

```tsx
'use client';

import { Droppable } from '@hello-pangea/dnd';
import { Customer } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';
import { KanbanCard } from './KanbanCard';

interface KanbanColumnProps {
  stageId: string;       // 'unassigned' for customers without stage
  title: string;
  color: string;
  customers: Customer[];
}

export function KanbanColumn({ stageId, title, color, customers }: KanbanColumnProps) {
  const t = useTranslations();

  return (
    <div className="flex w-72 flex-shrink-0 flex-col rounded-xl bg-neutral-50 border border-neutral-200">
      {/* Column header */}
      <div className="flex items-center gap-2 px-3 py-2.5 border-b border-neutral-200">
        <div className="h-3 w-3 rounded-full flex-shrink-0" style={{ backgroundColor: color }} />
        <h3 className="text-sm font-semibold text-neutral-800 truncate">{title}</h3>
        <span className="ml-auto rounded-full bg-neutral-200 px-2 py-0.5 text-[11px] font-medium text-neutral-600">
          {customers.length}
        </span>
      </div>

      {/* Droppable area */}
      <Droppable droppableId={stageId}>
        {(provided, snapshot) => (
          <div
            ref={provided.innerRef}
            {...provided.droppableProps}
            className={`flex-1 space-y-2 overflow-y-auto p-2 transition-colors ${
              snapshot.isDraggingOver ? 'bg-primary-50/50' : ''
            }`}
            style={{ minHeight: 80, maxHeight: 'calc(100vh - 220px)' }}
          >
            {customers.length === 0 && !snapshot.isDraggingOver && (
              <p className="py-6 text-center text-xs text-neutral-400">{t.crm.pipeline.noCustomers}</p>
            )}
            {customers.map((customer, index) => (
              <KanbanCard key={customer.id} customer={customer} index={index} />
            ))}
            {provided.placeholder}
          </div>
        )}
      </Droppable>
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/web/components/dashboard/pipeline/KanbanColumn.tsx
git commit -m "feat(web): add KanbanColumn component with droppable area"
```

---

### Task 5: Create KanbanBoard component

**Files:**
- Create: `apps/web/components/dashboard/pipeline/KanbanBoard.tsx`

- [ ] **Step 1: Create the component**

```tsx
'use client';

import { useCallback } from 'react';
import { DragDropContext, DropResult } from '@hello-pangea/dnd';
import { PipelineStage, Customer, customers as customersApi } from '@/lib/api';
import { KanbanColumn } from './KanbanColumn';

interface KanbanBoardProps {
  stages: PipelineStage[];
  customersByStage: Record<string, Customer[]>;
  onMoveCustomer: (customerId: string, newStageId: string | null, stageName: string) => void;
}

const UNASSIGNED_ID = 'unassigned';

export function KanbanBoard({ stages, customersByStage, onMoveCustomer }: KanbanBoardProps) {
  const handleDragEnd = useCallback(
    (result: DropResult) => {
      const { draggableId, destination, source } = result;
      if (!destination) return;
      if (destination.droppableId === source.droppableId) return;

      const newStageId = destination.droppableId === UNASSIGNED_ID ? null : destination.droppableId;
      const stageName =
        destination.droppableId === UNASSIGNED_ID
          ? UNASSIGNED_ID
          : stages.find(s => s.id === destination.droppableId)?.name ?? '';

      onMoveCustomer(draggableId, newStageId, stageName);
    },
    [stages, onMoveCustomer],
  );

  return (
    <DragDropContext onDragEnd={handleDragEnd}>
      <div className="flex gap-3 overflow-x-auto pb-4">
        {/* Unassigned column */}
        <KanbanColumn
          stageId={UNASSIGNED_ID}
          title={customersByStage[UNASSIGNED_ID]?.[0] ? 'Sin asignar' : 'Sin asignar'}
          color="#94a3b8"
          customers={customersByStage[UNASSIGNED_ID] ?? []}
        />

        {/* Stage columns */}
        {stages.map(stage => (
          <KanbanColumn
            key={stage.id}
            stageId={stage.id}
            title={stage.name}
            color={stage.color}
            customers={customersByStage[stage.id] ?? []}
          />
        ))}
      </div>
    </DragDropContext>
  );
}
```

Note: The "Sin asignar" title will be replaced with the i18n key in the page component that passes the title. Let me fix that — the title should come from the page. Actually, looking at the design, the `KanbanColumn` receives `title` as a prop, and the page will pass the translated string. But the `KanbanBoard` itself hardcodes "Sin asignar". We need to accept the unassigned title as a prop.

**Corrected version** — replace the component with:

```tsx
'use client';

import { useCallback } from 'react';
import { DragDropContext, DropResult } from '@hello-pangea/dnd';
import { PipelineStage, Customer } from '@/lib/api';
import { KanbanColumn } from './KanbanColumn';

interface KanbanBoardProps {
  stages: PipelineStage[];
  customersByStage: Record<string, Customer[]>;
  unassignedTitle: string;
  onMoveCustomer: (customerId: string, newStageId: string | null, stageName: string) => void;
}

const UNASSIGNED_ID = 'unassigned';

export function KanbanBoard({ stages, customersByStage, unassignedTitle, onMoveCustomer }: KanbanBoardProps) {
  const handleDragEnd = useCallback(
    (result: DropResult) => {
      const { draggableId, destination, source } = result;
      if (!destination) return;
      if (destination.droppableId === source.droppableId) return;

      const newStageId = destination.droppableId === UNASSIGNED_ID ? null : destination.droppableId;
      const stageName =
        destination.droppableId === UNASSIGNED_ID
          ? unassignedTitle
          : stages.find(s => s.id === destination.droppableId)?.name ?? '';

      onMoveCustomer(draggableId, newStageId, stageName);
    },
    [stages, unassignedTitle, onMoveCustomer],
  );

  return (
    <DragDropContext onDragEnd={handleDragEnd}>
      <div className="flex gap-3 overflow-x-auto pb-4">
        {/* Unassigned column */}
        <KanbanColumn
          stageId={UNASSIGNED_ID}
          title={unassignedTitle}
          color="#94a3b8"
          customers={customersByStage[UNASSIGNED_ID] ?? []}
        />

        {/* Stage columns */}
        {stages.map(stage => (
          <KanbanColumn
            key={stage.id}
            stageId={stage.id}
            title={stage.name}
            color={stage.color}
            customers={customersByStage[stage.id] ?? []}
          />
        ))}
      </div>
    </DragDropContext>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/web/components/dashboard/pipeline/KanbanBoard.tsx
git commit -m "feat(web): add KanbanBoard component with drag-and-drop context"
```

---

### Task 6: Create StageSettingsModal component

Extract the existing stages CRUD from the current pipeline page into a modal.

**Files:**
- Create: `apps/web/components/dashboard/pipeline/StageSettingsModal.tsx`

- [ ] **Step 1: Create the modal component**

This component contains the full CRUD for stages — create, edit, delete, reorder — extracted from the current `pipeline/page.tsx`. It renders as a full-screen overlay modal.

```tsx
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
```

- [ ] **Step 2: Commit**

```bash
git add apps/web/components/dashboard/pipeline/StageSettingsModal.tsx
git commit -m "feat(web): add StageSettingsModal for pipeline stage CRUD"
```

---

### Task 7: Rewrite the pipeline page as Kanban board

**Files:**
- Rewrite: `apps/web/app/dashboard/pipeline/page.tsx`

This is the main integration task. The page loads stages and customers, groups customers by `stage_id`, renders the `KanbanBoard`, and handles the `onMoveCustomer` callback with optimistic UI.

- [ ] **Step 1: Rewrite the page**

```tsx
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
```

- [ ] **Step 2: Commit**

```bash
git add apps/web/app/dashboard/pipeline/page.tsx
git commit -m "feat(web): rewrite pipeline page as kanban board with drag-and-drop"
```

---

### Task 8: Manual smoke test

- [ ] **Step 1: Build to verify no TypeScript errors**

```bash
cd apps/web && npx next build 2>&1 | head -50
```

Expected: Build succeeds without type errors.

- [ ] **Step 2: Verify the page loads**

Start dev server and navigate to `http://localhost:3000/dashboard/pipeline`. Verify:
- Columns render for each pipeline stage
- Customer cards appear in their assigned column
- Unassigned column shows customers without a stage
- Dragging a card to another column updates the stage
- Settings gear icon opens the stages CRUD modal
- Toast appears after moving a customer

- [ ] **Step 3: Final commit if any fixes needed**

```bash
git add -A
git commit -m "fix(web): polish kanban pipeline board"
```
