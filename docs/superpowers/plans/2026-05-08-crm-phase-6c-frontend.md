# CRM Phase 6C: Frontend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans.

**Goal:** Add 4 CRM dashboard pages (pipeline, treatments, tasks, automations) with API client, i18n, and sidebar navigation.

**Architecture:** Next.js 14 App Router pages following existing dashboard patterns. Manual state management (useState + useCallback + useEffect). Tailwind CSS for styling.

**Tech Stack:** Next.js 14, React 18, TypeScript, Tailwind CSS, date-fns, lucide-react

---

## Task 1: API Client Extensions (`apps/web/lib/api.ts`)

**Files to modify:**
- `apps/web/lib/api.ts`

### 1A. Add TypeScript types after the `Customer` interface (after line ~352)

Append these types before the `customers` export:

```typescript
// ── CRM: Pipeline Stages ──────────────────────────────────────────────────────

export interface PipelineStage {
  id: string;
  tenant_id: string;
  name: string;
  position: number;
  color: string;
  is_default: boolean;
  auto_rules_enabled: boolean;
  created_at: string;
}

// ── CRM: Treatments ──────────────────────────────────────────────────────────

export interface Treatment {
  id: string;
  tenant_id: string;
  customer_id: string;
  professional_id: string;
  name: string;
  treatment_type: string;
  status: string; // 'proposed' | 'accepted' | 'in_progress' | 'completed' | 'abandoned'
  total_sessions: number | null;
  completed_sessions: number;
  estimated_cost: number | null;
  paid_amount: number;
  currency: string;
  tooth_numbers: number[];
  notes: string;
  started_at: string | null;
  completed_at: string | null;
  next_session_at: string | null;
  created_at: string;
  updated_at: string;
  // Campos expandidos del JOIN (opcionales, dependen del endpoint)
  customer_name?: string;
  professional_name?: string;
}

// ── CRM: Tasks ───────────────────────────────────────────────────────────────

export interface Task {
  id: string;
  tenant_id: string;
  assigned_to: string | null;
  customer_id: string | null;
  appointment_id: string | null;
  treatment_id: string | null;
  title: string;
  description: string;
  status: string; // 'pending' | 'in_progress' | 'completed' | 'dismissed'
  due_at: string | null;
  completed_at: string | null;
  source: string; // 'manual' | 'rule' | 'system'
  rule_id: string | null;
  created_at: string;
  // Campos expandidos (opcionales)
  assigned_to_name?: string;
  customer_name?: string;
}

// ── CRM: Automation Rules ────────────────────────────────────────────────────

export interface Rule {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  trigger_type: string; // 'event' | 'temporal'
  trigger_event: string;
  trigger_schedule: { interval_days: number; reference_field: string } | null;
  conditions: Array<{ field: string; op: string; value: unknown }>;
  actions: Array<{ type: string; template: string; params: Record<string, unknown> }>;
  is_active: boolean;
  is_template: boolean;
  template_key: string;
  cooldown_hours: number;
  priority: number;
  created_at: string;
  updated_at: string;
}

export interface RuleExecution {
  id: string;
  rule_id: string;
  customer_id: string | null;
  triggered_at: string;
  trigger_event: string;
  status: string; // 'success' | 'failed' | 'skipped'
  error_message: string;
}
```

### 1B. Add `pipelineStages` API module after the `customers` export (~line 359)

```typescript
// ── Pipeline Stages ──────────────────────────────────────────────────────────

export const pipelineStages = {
  async list(): Promise<{ data: PipelineStage[] }> {
    return request('/api/v1/pipeline-stages');
  },

  async create(data: { name: string; color: string; position?: number }): Promise<PipelineStage> {
    return request('/api/v1/pipeline-stages', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async update(id: string, data: Partial<{ name: string; color: string; position: number; auto_rules_enabled: boolean }>): Promise<PipelineStage> {
    return request(`/api/v1/pipeline-stages/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async remove(id: string): Promise<void> {
    return request(`/api/v1/pipeline-stages/${id}`, { method: 'DELETE' });
  },

  async reorder(orderedIds: string[]): Promise<void> {
    return request('/api/v1/pipeline-stages/reorder', {
      method: 'PUT',
      body: JSON.stringify({ ids: orderedIds }),
    });
  },
};
```

### 1C. Add `treatments` API module

```typescript
// ── Treatments ───────────────────────────────────────────────────────────────

export const treatments = {
  async list(params?: {
    customer_id?: string;
    professional_id?: string;
    status?: string;
    search?: string;
  }): Promise<{ data: Treatment[] }> {
    const qs = new URLSearchParams();
    if (params?.customer_id) qs.set('customer_id', params.customer_id);
    if (params?.professional_id) qs.set('professional_id', params.professional_id);
    if (params?.status) qs.set('status', params.status);
    if (params?.search) qs.set('search', params.search);
    const query = qs.toString();
    return request(`/api/v1/treatments${query ? `?${query}` : ''}`);
  },

  async getById(id: string): Promise<Treatment> {
    return request(`/api/v1/treatments/${id}`);
  },

  async create(data: {
    customer_id: string;
    professional_id: string;
    name: string;
    treatment_type: string;
    total_sessions?: number;
    estimated_cost?: number;
    currency?: string;
    tooth_numbers?: number[];
    notes?: string;
  }): Promise<Treatment> {
    return request('/api/v1/treatments', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async update(id: string, data: Partial<{
    name: string;
    treatment_type: string;
    total_sessions: number;
    estimated_cost: number;
    notes: string;
    tooth_numbers: number[];
  }>): Promise<Treatment> {
    return request(`/api/v1/treatments/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async updateStatus(id: string, status: string): Promise<Treatment> {
    return request(`/api/v1/treatments/${id}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    });
  },
};
```

### 1D. Add `tasks` API module

```typescript
// ── Tasks ────────────────────────────────────────────────────────────────────

export const tasks = {
  async list(params?: {
    status?: string;
    assigned_to?: string;
    customer_id?: string;
    source?: string;
  }): Promise<{ data: Task[] }> {
    const qs = new URLSearchParams();
    if (params?.status) qs.set('status', params.status);
    if (params?.assigned_to) qs.set('assigned_to', params.assigned_to);
    if (params?.customer_id) qs.set('customer_id', params.customer_id);
    if (params?.source) qs.set('source', params.source);
    const query = qs.toString();
    return request(`/api/v1/tasks${query ? `?${query}` : ''}`);
  },

  async getById(id: string): Promise<Task> {
    return request(`/api/v1/tasks/${id}`);
  },

  async create(data: {
    title: string;
    description?: string;
    assigned_to?: string;
    customer_id?: string;
    treatment_id?: string;
    due_at?: string;
  }): Promise<Task> {
    return request('/api/v1/tasks', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async update(id: string, data: Partial<{
    title: string;
    description: string;
    assigned_to: string;
    due_at: string;
    status: string;
  }>): Promise<Task> {
    return request(`/api/v1/tasks/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async complete(id: string): Promise<Task> {
    return request(`/api/v1/tasks/${id}/complete`, { method: 'POST' });
  },

  async dismiss(id: string): Promise<Task> {
    return request(`/api/v1/tasks/${id}/dismiss`, { method: 'POST' });
  },
};
```

### 1E. Add `rules` API module

```typescript
// ── Automation Rules ─────────────────────────────────────────────────────────

export const rules = {
  async list(): Promise<{ data: Rule[] }> {
    return request('/api/v1/rules');
  },

  async getById(id: string): Promise<Rule> {
    return request(`/api/v1/rules/${id}`);
  },

  async create(data: {
    name: string;
    description?: string;
    trigger_type: string;
    trigger_event: string;
    trigger_schedule?: { interval_days: number; reference_field: string };
    conditions?: Array<{ field: string; op: string; value: unknown }>;
    actions: Array<{ type: string; template: string; params: Record<string, unknown> }>;
    cooldown_hours?: number;
    priority?: number;
  }): Promise<Rule> {
    return request('/api/v1/rules', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async update(id: string, data: Partial<{
    name: string;
    description: string;
    trigger_type: string;
    trigger_event: string;
    trigger_schedule: { interval_days: number; reference_field: string };
    conditions: Array<{ field: string; op: string; value: unknown }>;
    actions: Array<{ type: string; template: string; params: Record<string, unknown> }>;
    is_active: boolean;
    cooldown_hours: number;
    priority: number;
  }>): Promise<Rule> {
    return request(`/api/v1/rules/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async remove(id: string): Promise<void> {
    return request(`/api/v1/rules/${id}`, { method: 'DELETE' });
  },

  async listExecutions(ruleId: string): Promise<{ data: RuleExecution[] }> {
    return request(`/api/v1/rules/${ruleId}/executions`);
  },
};
```

---

## Task 2: i18n Keys

**Files to modify:**
- `apps/web/lib/i18n/locales/es.ts`
- `apps/web/lib/i18n/locales/en.ts`
- `apps/web/lib/i18n/locales/pt.ts` (keep in sync)

### 2A. Add `crm` section to `es.ts`

Add this block after the `faq` section (after line ~649, before `landing`):

```typescript
  crm: {
    pipeline: {
      title: 'Pipeline',
      description: 'Gestiona las etapas del pipeline de tus clientes.',
      stages: 'Etapas',
      addStage: 'Agregar etapa',
      editStage: 'Editar etapa',
      deleteStage: 'Eliminar etapa',
      noStages: 'No hay etapas configuradas aún.',
      noStagesDesc: 'Crea la primera etapa para organizar a tus clientes en el pipeline.',
      dragToReorder: 'Usa las flechas para reordenar',
      nameLabel: 'Nombre de la etapa',
      namePlaceholder: 'Ej. Primer contacto',
      colorLabel: 'Color',
      confirmDelete: 'Esta etapa se eliminara. Los clientes en esta etapa no seran afectados.',
      position: 'Posicion',
      autoRules: 'Reglas automaticas',
      defaultStage: 'Etapa por defecto',
      stagesCount: '{n} etapas',
    },
    treatments: {
      title: 'Tratamientos',
      description: 'Seguimiento de tratamientos y planes de tus clientes.',
      noTreatments: 'No hay tratamientos registrados.',
      noTreatmentsDesc: 'Los tratamientos aparecen aqui cuando los creas para tus clientes.',
      newTreatment: 'Nuevo tratamiento',
      type: 'Tipo',
      status: 'Estado',
      sessions: 'Sesiones',
      sessionsProgress: '{completed} de {total}',
      cost: 'Costo estimado',
      paid: 'Pagado',
      toothNumbers: 'Piezas dentales',
      notes: 'Notas',
      customer: 'Cliente',
      professional: 'Profesional',
      statusProposed: 'Propuesto',
      statusAccepted: 'Aceptado',
      statusInProgress: 'En progreso',
      statusCompleted: 'Completado',
      statusAbandoned: 'Abandonado',
      filterStatus: 'Estado',
      filterCustomer: 'Cliente',
      filterProfessional: 'Profesional',
      filterAll: 'Todos',
      searchPlaceholder: 'Buscar tratamiento...',
      treatmentsCount: '{n} tratamientos',
      nameLabel: 'Nombre del tratamiento',
      namePlaceholder: 'Ej. Ortodoncia completa',
      typeLabel: 'Tipo de tratamiento',
      typePlaceholder: 'Ej. ortodoncia',
      totalSessionsLabel: 'Total de sesiones',
      estimatedCostLabel: 'Costo estimado',
      currencyLabel: 'Moneda',
    },
    tasks: {
      title: 'Tareas',
      description: 'Bandeja de tareas pendientes y completadas.',
      noTasks: 'No hay tareas pendientes.',
      noTasksDesc: 'Las tareas se crean manualmente o son generadas por las reglas de automatizacion.',
      newTask: 'Nueva tarea',
      assignTo: 'Asignar a',
      dueAt: 'Vence',
      complete: 'Completar',
      dismiss: 'Descartar',
      pending: 'Pendiente',
      inProgress: 'En progreso',
      completed: 'Completada',
      dismissed: 'Descartada',
      overdue: 'Vencidas',
      today: 'Hoy',
      thisWeek: 'Esta semana',
      later: 'Mas adelante',
      noDueDate: 'Sin fecha',
      manual: 'Manual',
      rule: 'Regla',
      system: 'Sistema',
      filterStatus: 'Estado',
      filterAssigned: 'Asignado a',
      filterSource: 'Origen',
      filterAll: 'Todos',
      tasksCount: '{n} tareas',
      pendingCount: '{n} pendientes',
      titleLabel: 'Titulo de la tarea',
      titlePlaceholder: 'Ej. Llamar al paciente',
      descriptionLabel: 'Descripcion',
      descriptionPlaceholder: 'Detalles adicionales...',
      dueDateLabel: 'Fecha de vencimiento',
      customerLabel: 'Cliente asociado',
    },
    automations: {
      title: 'Automatizaciones',
      description: 'Reglas que ejecutan acciones automaticas cuando ocurren eventos.',
      noRules: 'No hay reglas configuradas.',
      noRulesDesc: 'Crea reglas para automatizar tareas, enviar mensajes y mas.',
      newRule: 'Nueva regla',
      editRule: 'Editar regla',
      deleteRule: 'Eliminar regla',
      active: 'Activa',
      inactive: 'Inactiva',
      trigger: 'Disparador',
      conditions: 'Condiciones',
      actions: 'Acciones',
      executions: 'Ejecuciones',
      cooldown: 'Cooldown',
      cooldownHours: '{n}h de cooldown',
      template: 'Plantilla',
      event: 'Evento',
      temporal: 'Temporal',
      lastExecution: 'Ultima ejecucion',
      noExecutions: 'Sin ejecuciones aun',
      executionSuccess: 'Exito',
      executionFailed: 'Error',
      executionSkipped: 'Omitida',
      confirmDelete: 'Se eliminara esta regla y su historial de ejecuciones.',
      nameLabel: 'Nombre de la regla',
      namePlaceholder: 'Ej. Recordatorio post-cita',
      descriptionLabel: 'Descripcion',
      descriptionPlaceholder: 'Que hace esta regla...',
      triggerTypeLabel: 'Tipo de disparador',
      triggerEventLabel: 'Evento',
      conditionsLabel: 'Condiciones (opcional)',
      actionsLabel: 'Acciones a ejecutar',
      cooldownLabel: 'Cooldown (horas)',
      priorityLabel: 'Prioridad',
      rulesCount: '{n} reglas',
      activeCount: '{n} activas',
      triggerEvents: {
        'appointment.completed': 'Cita completada',
        'appointment.cancelled': 'Cita cancelada',
        'appointment.no_show': 'Cliente no asistio',
        'customer.created': 'Cliente nuevo',
        'treatment.completed': 'Tratamiento completado',
        'treatment.abandoned': 'Tratamiento abandonado',
      },
    },
  },
```

### 2B. Add `crm` section to `en.ts`

Same structure, English translations:

```typescript
  crm: {
    pipeline: {
      title: 'Pipeline',
      description: 'Manage your customer pipeline stages.',
      stages: 'Stages',
      addStage: 'Add stage',
      editStage: 'Edit stage',
      deleteStage: 'Delete stage',
      noStages: 'No stages configured yet.',
      noStagesDesc: 'Create the first stage to organize your customers in the pipeline.',
      dragToReorder: 'Use arrows to reorder',
      nameLabel: 'Stage name',
      namePlaceholder: 'E.g. First contact',
      colorLabel: 'Color',
      confirmDelete: 'This stage will be deleted. Customers in this stage will not be affected.',
      position: 'Position',
      autoRules: 'Auto rules',
      defaultStage: 'Default stage',
      stagesCount: '{n} stages',
    },
    treatments: {
      title: 'Treatments',
      description: 'Track treatments and plans for your customers.',
      noTreatments: 'No treatments registered.',
      noTreatmentsDesc: 'Treatments will appear here when you create them for your customers.',
      newTreatment: 'New treatment',
      type: 'Type',
      status: 'Status',
      sessions: 'Sessions',
      sessionsProgress: '{completed} of {total}',
      cost: 'Estimated cost',
      paid: 'Paid',
      toothNumbers: 'Tooth numbers',
      notes: 'Notes',
      customer: 'Customer',
      professional: 'Professional',
      statusProposed: 'Proposed',
      statusAccepted: 'Accepted',
      statusInProgress: 'In progress',
      statusCompleted: 'Completed',
      statusAbandoned: 'Abandoned',
      filterStatus: 'Status',
      filterCustomer: 'Customer',
      filterProfessional: 'Professional',
      filterAll: 'All',
      searchPlaceholder: 'Search treatment...',
      treatmentsCount: '{n} treatments',
      nameLabel: 'Treatment name',
      namePlaceholder: 'E.g. Full orthodontics',
      typeLabel: 'Treatment type',
      typePlaceholder: 'E.g. orthodontics',
      totalSessionsLabel: 'Total sessions',
      estimatedCostLabel: 'Estimated cost',
      currencyLabel: 'Currency',
    },
    tasks: {
      title: 'Tasks',
      description: 'Inbox for pending and completed tasks.',
      noTasks: 'No pending tasks.',
      noTasksDesc: 'Tasks are created manually or generated by automation rules.',
      newTask: 'New task',
      assignTo: 'Assign to',
      dueAt: 'Due',
      complete: 'Complete',
      dismiss: 'Dismiss',
      pending: 'Pending',
      inProgress: 'In progress',
      completed: 'Completed',
      dismissed: 'Dismissed',
      overdue: 'Overdue',
      today: 'Today',
      thisWeek: 'This week',
      later: 'Later',
      noDueDate: 'No due date',
      manual: 'Manual',
      rule: 'Rule',
      system: 'System',
      filterStatus: 'Status',
      filterAssigned: 'Assigned to',
      filterSource: 'Source',
      filterAll: 'All',
      tasksCount: '{n} tasks',
      pendingCount: '{n} pending',
      titleLabel: 'Task title',
      titlePlaceholder: 'E.g. Call the patient',
      descriptionLabel: 'Description',
      descriptionPlaceholder: 'Additional details...',
      dueDateLabel: 'Due date',
      customerLabel: 'Associated customer',
    },
    automations: {
      title: 'Automations',
      description: 'Rules that execute automatic actions when events occur.',
      noRules: 'No rules configured.',
      noRulesDesc: 'Create rules to automate tasks, send messages and more.',
      newRule: 'New rule',
      editRule: 'Edit rule',
      deleteRule: 'Delete rule',
      active: 'Active',
      inactive: 'Inactive',
      trigger: 'Trigger',
      conditions: 'Conditions',
      actions: 'Actions',
      executions: 'Executions',
      cooldown: 'Cooldown',
      cooldownHours: '{n}h cooldown',
      template: 'Template',
      event: 'Event',
      temporal: 'Temporal',
      lastExecution: 'Last execution',
      noExecutions: 'No executions yet',
      executionSuccess: 'Success',
      executionFailed: 'Failed',
      executionSkipped: 'Skipped',
      confirmDelete: 'This rule and its execution history will be deleted.',
      nameLabel: 'Rule name',
      namePlaceholder: 'E.g. Post-appointment reminder',
      descriptionLabel: 'Description',
      descriptionPlaceholder: 'What this rule does...',
      triggerTypeLabel: 'Trigger type',
      triggerEventLabel: 'Event',
      conditionsLabel: 'Conditions (optional)',
      actionsLabel: 'Actions to execute',
      cooldownLabel: 'Cooldown (hours)',
      priorityLabel: 'Priority',
      rulesCount: '{n} rules',
      activeCount: '{n} active',
      triggerEvents: {
        'appointment.completed': 'Appointment completed',
        'appointment.cancelled': 'Appointment cancelled',
        'appointment.no_show': 'Customer no-show',
        'customer.created': 'New customer',
        'treatment.completed': 'Treatment completed',
        'treatment.abandoned': 'Treatment abandoned',
      },
    },
  },
```

### 2C. Add nav keys to both locale files

In **es.ts** `nav` section, add after `help`:

```typescript
    pipeline: 'Pipeline',
    treatments: 'Tratamientos',
    tasks: 'Tareas',
    automations: 'Automatizaciones',
```

In **en.ts** `nav` section, add after `help`:

```typescript
    pipeline: 'Pipeline',
    treatments: 'Treatments',
    tasks: 'Tasks',
    automations: 'Automations',
```

### 2D. Update `pt.ts` with the same structure (Portuguese translations, mirroring es.ts keys)

Use the same key structure. Portuguese translations should follow the existing pt.ts pattern.

---

## Task 3: Sidebar Navigation (`apps/web/components/dashboard/sidebar.tsx`)

**Files to modify:**
- `apps/web/components/dashboard/sidebar.tsx`

### 3A. Add lucide-react icon imports

Add to the import statement at line 8:

```typescript
import {
  CalendarDays,
  Users,
  Scissors,
  BookOpen,
  MessageCircle,
  BarChart3,
  Settings,
  LogOut,
  UserCog,
  LayoutDashboard,
  HelpCircle,
  ChevronsLeft,
  ChevronsRight,
  Menu,
  X,
  Kanban,
  Stethoscope,
  CheckSquare,
  Zap,
} from 'lucide-react';
```

### 3B. Add 4 CRM items to `NAV_ITEMS` array

Insert these 4 items **after the `clients` entry** (line 55) and **before the `services` entry** (line 56):

```typescript
    { href: '/dashboard/pipeline',     icon: Kanban,       label: t.nav.pipeline     },
    { href: '/dashboard/treatments',   icon: Stethoscope,  label: t.nav.treatments   },
    { href: '/dashboard/tasks',        icon: CheckSquare,  label: t.nav.tasks        },
    { href: '/dashboard/automations',  icon: Zap,          label: t.nav.automations  },
```

The final nav order will be:
1. Dashboard
2. Agenda
3. Clients
4. **Pipeline** (new)
5. **Treatments** (new)
6. **Tasks** (new)
7. **Automations** (new)
8. Services
9. Team
10. Knowledge
11. WhatsApp
12. Analytics
13. Settings
14. Help

---

## Task 4: Pipeline Page (`apps/web/app/(dashboard)/pipeline/page.tsx`)

**Files to create:**
- `apps/web/app/(dashboard)/pipeline/page.tsx`

### Full implementation

```tsx
'use client';

import { useState, useEffect, useCallback } from 'react';
import { Kanban, Plus, Trash2, Pencil, ChevronUp, ChevronDown, X } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { pipelineStages, PipelineStage } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

// Colores predefinidos para las etapas del pipeline
const STAGE_COLORS = [
  '#8b5cf6', // violet
  '#3b82f6', // blue
  '#06b6d4', // cyan
  '#10b981', // emerald
  '#f59e0b', // amber
  '#ef4444', // red
  '#ec4899', // pink
  '#6366f1', // indigo
];

export default function PipelinePage() {
  const t = useTranslations();

  const [stages, setStages] = useState<PipelineStage[]>([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  // Formulario de crear/editar
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
    load();
  }, [load]);

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
    } catch {
      // Error silencioso por ahora
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string) {
    if (!confirm(t.crm.pipeline.confirmDelete)) return;
    try {
      await pipelineStages.remove(id);
      await load();
    } catch {
      // Error silencioso
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
    } catch {
      await load(); // Revertir al estado del servidor en caso de error
    }
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-neutral-900">{t.crm.pipeline.title}</h1>
          <p className="mt-0.5 text-sm text-neutral-500">{t.crm.pipeline.description}</p>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-neutral-500">
            {t.crm.pipeline.stagesCount.replace('{n}', String(stages.length))}
          </span>
          <button
            onClick={openCreateForm}
            className="flex items-center gap-1.5 rounded-lg bg-primary-600 px-3 py-1.5 text-xs font-semibold text-white transition-colors hover:bg-primary-500"
          >
            <Plus className="h-3.5 w-3.5" />
            {t.crm.pipeline.addStage}
          </button>
        </div>
      </div>

      {/* Formulario inline de crear/editar */}
      {showForm && (
        <Card className="mb-6 p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-neutral-800">
              {editingId ? t.crm.pipeline.editStage : t.crm.pipeline.addStage}
            </h3>
            <button onClick={closeForm} className="text-neutral-400 hover:text-neutral-600">
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="flex items-end gap-3">
            <div className="flex-1">
              <label className="block text-xs font-medium text-neutral-600 mb-1">
                {t.crm.pipeline.nameLabel}
              </label>
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
              <label className="block text-xs font-medium text-neutral-600 mb-1">
                {t.crm.pipeline.colorLabel}
              </label>
              <div className="flex gap-1.5">
                {STAGE_COLORS.map(color => (
                  <button
                    key={color}
                    onClick={() => setFormColor(color)}
                    className={`h-8 w-8 rounded-full border-2 transition-all ${
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
              className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary-500 disabled:opacity-50"
            >
              {saving ? t.common.loading : t.common.save}
            </button>
          </div>
        </Card>
      )}

      {/* Lista de etapas */}
      {loading ? (
        <div className="flex justify-center py-16"><Spinner size="lg" /></div>
      ) : stages.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <Kanban className="mb-3 h-12 w-12 text-neutral-200" />
          <p className="text-sm font-medium text-neutral-500">{t.crm.pipeline.noStages}</p>
          <p className="mt-1 text-xs text-neutral-400">{t.crm.pipeline.noStagesDesc}</p>
        </div>
      ) : (
        <div className="space-y-2">
          <p className="text-xs text-neutral-400 mb-2">{t.crm.pipeline.dragToReorder}</p>
          {stages.map((stage, idx) => (
            <div
              key={stage.id}
              className="flex items-center gap-3 rounded-xl border border-neutral-200 bg-white px-4 py-3 transition-colors hover:bg-neutral-50"
            >
              {/* Color badge */}
              <div
                className="h-4 w-4 rounded-full flex-shrink-0"
                style={{ backgroundColor: stage.color }}
              />

              {/* Posicion */}
              <span className="text-xs font-mono text-neutral-400 w-6 text-center">
                {idx + 1}
              </span>

              {/* Nombre */}
              <span className="flex-1 text-sm font-medium text-neutral-900">{stage.name}</span>

              {/* Badges */}
              {stage.is_default && (
                <Badge variant="default">{t.crm.pipeline.defaultStage}</Badge>
              )}
              {stage.auto_rules_enabled && (
                <Badge variant="primary">{t.crm.pipeline.autoRules}</Badge>
              )}

              {/* Acciones */}
              <div className="flex items-center gap-1">
                <button
                  onClick={() => handleMove(idx, 'up')}
                  disabled={idx === 0}
                  className="rounded p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 disabled:opacity-30"
                >
                  <ChevronUp className="h-4 w-4" />
                </button>
                <button
                  onClick={() => handleMove(idx, 'down')}
                  disabled={idx === stages.length - 1}
                  className="rounded p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 disabled:opacity-30"
                >
                  <ChevronDown className="h-4 w-4" />
                </button>
                <button
                  onClick={() => openEditForm(stage)}
                  className="rounded p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600"
                >
                  <Pencil className="h-4 w-4" />
                </button>
                <button
                  onClick={() => handleDelete(stage.id)}
                  className="rounded p-1 text-neutral-400 hover:bg-red-50 hover:text-red-500"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
```

---

## Task 5: Tasks Page (`apps/web/app/(dashboard)/tasks/page.tsx`)

**Files to create:**
- `apps/web/app/(dashboard)/tasks/page.tsx`

### Full implementation

```tsx
'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  CheckSquare, Plus, Check, XCircle, Clock, AlertTriangle, X, User,
} from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { tasks as tasksApi, Task, professionals, Professional } from '@/lib/api';
import { format, isToday, isBefore, isThisWeek, startOfDay } from 'date-fns';
import { useTranslations, useDateLocale } from '@/lib/i18n';

type StatusFilter = '' | 'pending' | 'in_progress' | 'completed' | 'dismissed';
type SourceFilter = '' | 'manual' | 'rule' | 'system';

// Clasificar tareas por urgencia temporal
function classifyTask(task: Task): 'overdue' | 'today' | 'thisWeek' | 'later' | 'noDueDate' {
  if (!task.due_at) return 'noDueDate';
  const due = new Date(task.due_at);
  const now = startOfDay(new Date());
  if (isBefore(due, now)) return 'overdue';
  if (isToday(due)) return 'today';
  if (isThisWeek(due, { weekStartsOn: 1 })) return 'thisWeek';
  return 'later';
}

const STATUS_COLORS: Record<string, { bg: string; text: string }> = {
  pending:     { bg: 'bg-amber-100',   text: 'text-amber-700'   },
  in_progress: { bg: 'bg-blue-100',    text: 'text-blue-700'    },
  completed:   { bg: 'bg-emerald-100', text: 'text-emerald-700' },
  dismissed:   { bg: 'bg-neutral-100', text: 'text-neutral-500' },
};

export default function TasksPage() {
  const t = useTranslations();
  const dateLocale = useDateLocale();

  const [list, setList]             = useState<Task[]>([]);
  const [profList, setProfList]     = useState<Professional[]>([]);
  const [loading, setLoading]       = useState(true);
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('');
  const [assignedFilter, setAssignedFilter] = useState('');
  const [sourceFilter, setSourceFilter] = useState<SourceFilter>('');

  // Formulario de crear tarea
  const [showForm, setShowForm]     = useState(false);
  const [formTitle, setFormTitle]   = useState('');
  const [formDesc, setFormDesc]     = useState('');
  const [formAssigned, setFormAssigned] = useState('');
  const [formDueAt, setFormDueAt]   = useState('');
  const [saving, setSaving]         = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const params: Record<string, string> = {};
      if (statusFilter) params.status = statusFilter;
      if (assignedFilter) params.assigned_to = assignedFilter;
      if (sourceFilter) params.source = sourceFilter;
      const res = await tasksApi.list(params);
      setList(res.data ?? []);
    } catch {
      setList([]);
    } finally {
      setLoading(false);
    }
  }, [statusFilter, assignedFilter, sourceFilter]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    professionals.list().then(r => setProfList(r.data ?? [])).catch(() => {});
  }, []);

  async function handleComplete(id: string) {
    try {
      await tasksApi.complete(id);
      await load();
    } catch { /* silencioso */ }
  }

  async function handleDismiss(id: string) {
    try {
      await tasksApi.dismiss(id);
      await load();
    } catch { /* silencioso */ }
  }

  async function handleCreate() {
    if (!formTitle.trim()) return;
    setSaving(true);
    try {
      await tasksApi.create({
        title: formTitle.trim(),
        description: formDesc.trim() || undefined,
        assigned_to: formAssigned || undefined,
        due_at: formDueAt || undefined,
      });
      setShowForm(false);
      setFormTitle('');
      setFormDesc('');
      setFormAssigned('');
      setFormDueAt('');
      await load();
    } catch { /* silencioso */ } finally {
      setSaving(false);
    }
  }

  // Agrupar tareas por urgencia
  const grouped = list.reduce<Record<string, Task[]>>((acc, task) => {
    // No agrupar las completadas/descartadas
    if (task.status === 'completed' || task.status === 'dismissed') {
      const key = task.status;
      (acc[key] ??= []).push(task);
    } else {
      const key = classifyTask(task);
      (acc[key] ??= []).push(task);
    }
    return acc;
  }, {});

  const groupOrder = ['overdue', 'today', 'thisWeek', 'later', 'noDueDate', 'completed', 'dismissed'];
  const groupLabels: Record<string, string> = {
    overdue:   t.crm.tasks.overdue,
    today:     t.crm.tasks.today,
    thisWeek:  t.crm.tasks.thisWeek,
    later:     t.crm.tasks.later,
    noDueDate: t.crm.tasks.noDueDate,
    completed: t.crm.tasks.completed,
    dismissed: t.crm.tasks.dismissed,
  };
  const groupIcons: Record<string, typeof Clock> = {
    overdue:   AlertTriangle,
    today:     Clock,
    thisWeek:  Clock,
    later:     Clock,
    noDueDate: Clock,
    completed: Check,
    dismissed: XCircle,
  };

  const pendingCount = list.filter(t => t.status === 'pending' || t.status === 'in_progress').length;

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-neutral-900">{t.crm.tasks.title}</h1>
          <p className="mt-0.5 text-sm text-neutral-500">{t.crm.tasks.description}</p>
        </div>
        <div className="flex items-center gap-3">
          {pendingCount > 0 && (
            <Badge variant="primary">
              {t.crm.tasks.pendingCount.replace('{n}', String(pendingCount))}
            </Badge>
          )}
          <button
            onClick={() => setShowForm(true)}
            className="flex items-center gap-1.5 rounded-lg bg-primary-600 px-3 py-1.5 text-xs font-semibold text-white transition-colors hover:bg-primary-500"
          >
            <Plus className="h-3.5 w-3.5" />
            {t.crm.tasks.newTask}
          </button>
        </div>
      </div>

      {/* Formulario de crear tarea */}
      {showForm && (
        <Card className="mb-6 p-4">
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold text-neutral-800">{t.crm.tasks.newTask}</h3>
            <button onClick={() => setShowForm(false)} className="text-neutral-400 hover:text-neutral-600">
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-neutral-600 mb-1">
                {t.crm.tasks.titleLabel}
              </label>
              <input
                type="text"
                value={formTitle}
                onChange={e => setFormTitle(e.target.value)}
                placeholder={t.crm.tasks.titlePlaceholder}
                className="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
                onKeyDown={e => e.key === 'Enter' && handleCreate()}
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-neutral-600 mb-1">
                {t.crm.tasks.dueDateLabel}
              </label>
              <input
                type="date"
                value={formDueAt}
                onChange={e => setFormDueAt(e.target.value)}
                className="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
              />
            </div>
            <div className="md:col-span-2">
              <label className="block text-xs font-medium text-neutral-600 mb-1">
                {t.crm.tasks.descriptionLabel}
              </label>
              <textarea
                value={formDesc}
                onChange={e => setFormDesc(e.target.value)}
                placeholder={t.crm.tasks.descriptionPlaceholder}
                rows={2}
                className="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100 resize-none"
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-neutral-600 mb-1">
                {t.crm.tasks.assignTo}
              </label>
              <select
                value={formAssigned}
                onChange={e => setFormAssigned(e.target.value)}
                className="w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm text-neutral-900 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
              >
                <option value="">{t.crm.tasks.filterAll}</option>
                {profList.filter(p => p.is_active).map(p => (
                  <option key={p.id} value={p.id}>{p.name}</option>
                ))}
              </select>
            </div>
            <div className="flex items-end">
              <button
                onClick={handleCreate}
                disabled={saving || !formTitle.trim()}
                className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-primary-500 disabled:opacity-50"
              >
                {saving ? t.common.loading : t.common.create}
              </button>
            </div>
          </div>
        </Card>
      )}

      {/* Filtros */}
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <select
          value={statusFilter}
          onChange={e => setStatusFilter(e.target.value as StatusFilter)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:outline-none focus:ring-1 focus:ring-primary-400"
        >
          <option value="">{t.crm.tasks.filterStatus}: {t.crm.tasks.filterAll}</option>
          <option value="pending">{t.crm.tasks.pending}</option>
          <option value="in_progress">{t.crm.tasks.inProgress}</option>
          <option value="completed">{t.crm.tasks.completed}</option>
          <option value="dismissed">{t.crm.tasks.dismissed}</option>
        </select>

        <select
          value={assignedFilter}
          onChange={e => setAssignedFilter(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:outline-none focus:ring-1 focus:ring-primary-400"
        >
          <option value="">{t.crm.tasks.filterAssigned}: {t.crm.tasks.filterAll}</option>
          {profList.filter(p => p.is_active).map(p => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>

        <select
          value={sourceFilter}
          onChange={e => setSourceFilter(e.target.value as SourceFilter)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:outline-none focus:ring-1 focus:ring-primary-400"
        >
          <option value="">{t.crm.tasks.filterSource}: {t.crm.tasks.filterAll}</option>
          <option value="manual">{t.crm.tasks.manual}</option>
          <option value="rule">{t.crm.tasks.rule}</option>
          <option value="system">{t.crm.tasks.system}</option>
        </select>

        <span className="ml-auto text-xs text-neutral-400">
          {t.crm.tasks.tasksCount.replace('{n}', String(list.length))}
        </span>
      </div>

      {/* Contenido */}
      {loading ? (
        <div className="flex justify-center py-16"><Spinner size="lg" /></div>
      ) : list.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <CheckSquare className="mb-3 h-12 w-12 text-neutral-200" />
          <p className="text-sm font-medium text-neutral-500">{t.crm.tasks.noTasks}</p>
          <p className="mt-1 text-xs text-neutral-400">{t.crm.tasks.noTasksDesc}</p>
        </div>
      ) : (
        <div className="space-y-6">
          {groupOrder.map(group => {
            const items = grouped[group];
            if (!items || items.length === 0) return null;
            const GroupIcon = groupIcons[group] ?? Clock;
            const isOverdue = group === 'overdue';

            return (
              <div key={group}>
                <div className="flex items-center gap-2 mb-2">
                  <GroupIcon className={`h-4 w-4 ${isOverdue ? 'text-red-500' : 'text-neutral-400'}`} />
                  <h3 className={`text-xs font-semibold uppercase tracking-wide ${
                    isOverdue ? 'text-red-600' : 'text-neutral-500'
                  }`}>
                    {groupLabels[group]} ({items.length})
                  </h3>
                </div>
                <div className="space-y-1">
                  {items.map(task => {
                    const sc = STATUS_COLORS[task.status] ?? STATUS_COLORS.pending;
                    const isActionable = task.status === 'pending' || task.status === 'in_progress';

                    return (
                      <div
                        key={task.id}
                        className="flex items-center gap-3 rounded-lg border border-neutral-200 bg-white px-4 py-3 transition-colors hover:bg-neutral-50"
                      >
                        {/* Status badge */}
                        <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${sc.bg} ${sc.text}`}>
                          {task.status === 'pending' && t.crm.tasks.pending}
                          {task.status === 'in_progress' && t.crm.tasks.inProgress}
                          {task.status === 'completed' && t.crm.tasks.completed}
                          {task.status === 'dismissed' && t.crm.tasks.dismissed}
                        </span>

                        {/* Titulo y descripcion */}
                        <div className="flex-1 min-w-0">
                          <p className={`text-sm font-medium ${
                            task.status === 'completed' || task.status === 'dismissed'
                              ? 'text-neutral-400 line-through' : 'text-neutral-900'
                          }`}>
                            {task.title}
                          </p>
                          {task.customer_name && (
                            <p className="text-xs text-neutral-500 flex items-center gap-1 mt-0.5">
                              <User className="h-3 w-3" />
                              {task.customer_name}
                            </p>
                          )}
                        </div>

                        {/* Source badge */}
                        <span className="text-[10px] text-neutral-400 font-medium uppercase">
                          {task.source === 'manual' && t.crm.tasks.manual}
                          {task.source === 'rule' && t.crm.tasks.rule}
                          {task.source === 'system' && t.crm.tasks.system}
                        </span>

                        {/* Due date */}
                        {task.due_at && (
                          <span className={`text-xs ${isOverdue ? 'text-red-500 font-medium' : 'text-neutral-500'}`}>
                            {format(new Date(task.due_at), 'd MMM', { locale: dateLocale })}
                          </span>
                        )}

                        {/* Assigned */}
                        {task.assigned_to_name && (
                          <span className="text-xs text-neutral-400 max-w-[100px] truncate">
                            {task.assigned_to_name}
                          </span>
                        )}

                        {/* Acciones rapidas */}
                        {isActionable && (
                          <div className="flex items-center gap-1">
                            <button
                              onClick={() => handleComplete(task.id)}
                              title={t.crm.tasks.complete}
                              className="rounded p-1 text-emerald-500 hover:bg-emerald-50 transition-colors"
                            >
                              <Check className="h-4 w-4" />
                            </button>
                            <button
                              onClick={() => handleDismiss(task.id)}
                              title={t.crm.tasks.dismiss}
                              className="rounded p-1 text-neutral-400 hover:bg-neutral-100 transition-colors"
                            >
                              <XCircle className="h-4 w-4" />
                            </button>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
```

---

## Task 6: Treatments Page (`apps/web/app/(dashboard)/treatments/page.tsx`)

**Files to create:**
- `apps/web/app/(dashboard)/treatments/page.tsx`

### Full implementation

```tsx
'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  Stethoscope, Search, Plus, X,
} from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import {
  treatments as treatmentsApi, Treatment,
  professionals, Professional,
  customers, Customer,
} from '@/lib/api';
import { format } from 'date-fns';
import { useTranslations, useDateLocale } from '@/lib/i18n';

type StatusFilter = '' | 'proposed' | 'accepted' | 'in_progress' | 'completed' | 'abandoned';

const STATUS_CFG: Record<string, { bg: string; text: string }> = {
  proposed:    { bg: 'bg-neutral-100',  text: 'text-neutral-600'  },
  accepted:    { bg: 'bg-blue-100',     text: 'text-blue-700'     },
  in_progress: { bg: 'bg-violet-100',   text: 'text-violet-700'   },
  completed:   { bg: 'bg-emerald-100',  text: 'text-emerald-700'  },
  abandoned:   { bg: 'bg-red-100',      text: 'text-red-600'      },
};

export default function TreatmentsPage() {
  const t = useTranslations();
  const dateLocale = useDateLocale();

  const [list, setList]             = useState<Treatment[]>([]);
  const [profList, setProfList]     = useState<Professional[]>([]);
  const [loading, setLoading]       = useState(true);
  const [search, setSearch]         = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('');
  const [profFilter, setProfFilter] = useState('');

  // Mapa de status -> label
  const statusLabel: Record<string, string> = {
    proposed:    t.crm.treatments.statusProposed,
    accepted:    t.crm.treatments.statusAccepted,
    in_progress: t.crm.treatments.statusInProgress,
    completed:   t.crm.treatments.statusCompleted,
    abandoned:   t.crm.treatments.statusAbandoned,
  };

  const load = useCallback(async (q: string) => {
    setLoading(true);
    try {
      const params: Record<string, string> = {};
      if (statusFilter) params.status = statusFilter;
      if (profFilter) params.professional_id = profFilter;
      if (q) params.search = q;
      const res = await treatmentsApi.list(params);
      setList(res.data ?? []);
    } catch {
      setList([]);
    } finally {
      setLoading(false);
    }
  }, [statusFilter, profFilter]);

  // Busqueda con debounce
  useEffect(() => {
    const timer = setTimeout(() => load(search), 300);
    return () => clearTimeout(timer);
  }, [search, load]);

  useEffect(() => {
    professionals.list().then(r => setProfList(r.data ?? [])).catch(() => {});
  }, []);

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-neutral-900">{t.crm.treatments.title}</h1>
          <p className="mt-0.5 text-sm text-neutral-500">{t.crm.treatments.description}</p>
        </div>
        <div className="flex items-center gap-2 text-sm text-neutral-500">
          <Stethoscope className="h-4 w-4" />
          <span>{t.crm.treatments.treatmentsCount.replace('{n}', String(list.length))}</span>
        </div>
      </div>

      {/* Buscador + filtros */}
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
          <input
            type="text"
            placeholder={t.crm.treatments.searchPlaceholder}
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="w-full rounded-lg border border-neutral-200 bg-white py-2 pl-9 pr-3 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
          />
        </div>

        <select
          value={statusFilter}
          onChange={e => setStatusFilter(e.target.value as StatusFilter)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:outline-none focus:ring-1 focus:ring-primary-400"
        >
          <option value="">{t.crm.treatments.filterStatus}: {t.crm.treatments.filterAll}</option>
          <option value="proposed">{t.crm.treatments.statusProposed}</option>
          <option value="accepted">{t.crm.treatments.statusAccepted}</option>
          <option value="in_progress">{t.crm.treatments.statusInProgress}</option>
          <option value="completed">{t.crm.treatments.statusCompleted}</option>
          <option value="abandoned">{t.crm.treatments.statusAbandoned}</option>
        </select>

        <select
          value={profFilter}
          onChange={e => setProfFilter(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-2 py-1.5 text-xs text-neutral-700 focus:outline-none focus:ring-1 focus:ring-primary-400"
        >
          <option value="">{t.crm.treatments.filterProfessional}: {t.crm.treatments.filterAll}</option>
          {profList.filter(p => p.is_active).map(p => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>
      </div>

      {/* Contenido */}
      {loading ? (
        <div className="flex justify-center py-16"><Spinner size="lg" /></div>
      ) : list.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <Stethoscope className="mb-3 h-12 w-12 text-neutral-200" />
          <p className="text-sm font-medium text-neutral-500">
            {search ? t.crm.treatments.noTreatments : t.crm.treatments.noTreatments}
          </p>
          <p className="mt-1 text-xs text-neutral-400">{t.crm.treatments.noTreatmentsDesc}</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-neutral-200 bg-white">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-neutral-100 bg-neutral-50 text-left text-xs font-medium uppercase tracking-wide text-neutral-500">
                <th className="px-4 py-3">{t.crm.treatments.nameLabel}</th>
                <th className="px-4 py-3">{t.crm.treatments.customer}</th>
                <th className="px-4 py-3">{t.crm.treatments.professional}</th>
                <th className="px-4 py-3 text-center">{t.crm.treatments.status}</th>
                <th className="px-4 py-3 text-center">{t.crm.treatments.sessions}</th>
                <th className="px-4 py-3 text-right">{t.crm.treatments.cost}</th>
                <th className="px-4 py-3 text-right">{t.crm.treatments.paid}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-100">
              {list.map(tr => {
                const sc = STATUS_CFG[tr.status] ?? STATUS_CFG.proposed;
                const progressPct = tr.total_sessions
                  ? Math.min(100, Math.round((tr.completed_sessions / tr.total_sessions) * 100))
                  : 0;

                return (
                  <tr key={tr.id} className="hover:bg-neutral-50 transition-colors">
                    {/* Nombre + tipo */}
                    <td className="px-4 py-3">
                      <p className="font-medium text-neutral-900">{tr.name}</p>
                      <p className="text-xs text-neutral-400">{tr.treatment_type}</p>
                    </td>

                    {/* Cliente */}
                    <td className="px-4 py-3 text-neutral-600">
                      {tr.customer_name ?? '—'}
                    </td>

                    {/* Profesional */}
                    <td className="px-4 py-3 text-neutral-600">
                      {tr.professional_name ?? '—'}
                    </td>

                    {/* Status */}
                    <td className="px-4 py-3 text-center">
                      <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${sc.bg} ${sc.text}`}>
                        {statusLabel[tr.status] ?? tr.status}
                      </span>
                    </td>

                    {/* Sesiones con barra de progreso */}
                    <td className="px-4 py-3">
                      {tr.total_sessions ? (
                        <div className="flex flex-col items-center gap-1">
                          <span className="text-xs text-neutral-600">
                            {t.crm.treatments.sessionsProgress
                              .replace('{completed}', String(tr.completed_sessions))
                              .replace('{total}', String(tr.total_sessions))}
                          </span>
                          <div className="h-1.5 w-full max-w-[80px] rounded-full bg-neutral-100">
                            <div
                              className="h-full rounded-full bg-primary-500 transition-all"
                              style={{ width: `${progressPct}%` }}
                            />
                          </div>
                        </div>
                      ) : (
                        <span className="text-xs text-neutral-300 block text-center">—</span>
                      )}
                    </td>

                    {/* Costo estimado */}
                    <td className="px-4 py-3 text-right text-neutral-600">
                      {tr.estimated_cost != null
                        ? `$${tr.estimated_cost.toFixed(2)} ${tr.currency}`
                        : '—'}
                    </td>

                    {/* Pagado */}
                    <td className="px-4 py-3 text-right">
                      <span className={tr.paid_amount > 0 ? 'text-emerald-600 font-medium' : 'text-neutral-300'}>
                        {tr.paid_amount > 0
                          ? `$${tr.paid_amount.toFixed(2)} ${tr.currency}`
                          : '—'}
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
```

---

## Task 7: Automations Page (`apps/web/app/(dashboard)/automations/page.tsx`)

**Files to create:**
- `apps/web/app/(dashboard)/automations/page.tsx`

### Full implementation

```tsx
'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  Zap, Plus, Trash2, Pencil, Play, ChevronDown, ChevronUp, X, AlertCircle,
} from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { rules as rulesApi, Rule, RuleExecution } from '@/lib/api';
import { format } from 'date-fns';
import { useTranslations, useDateLocale } from '@/lib/i18n';

export default function AutomationsPage() {
  const t = useTranslations();
  const dateLocale = useDateLocale();

  const [list, setList]         = useState<Rule[]>([]);
  const [loading, setLoading]   = useState(true);

  // Expandir ejecuciones de una regla
  const [expandedId, setExpandedId]     = useState<string | null>(null);
  const [executions, setExecutions]     = useState<RuleExecution[]>([]);
  const [loadingExec, setLoadingExec]   = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await rulesApi.list();
      setList(res.data ?? []);
    } catch {
      setList([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function toggleActive(rule: Rule) {
    try {
      await rulesApi.update(rule.id, { is_active: !rule.is_active });
      await load();
    } catch { /* silencioso */ }
  }

  async function handleDelete(id: string) {
    if (!confirm(t.crm.automations.confirmDelete)) return;
    try {
      await rulesApi.remove(id);
      await load();
    } catch { /* silencioso */ }
  }

  async function toggleExpand(ruleId: string) {
    if (expandedId === ruleId) {
      setExpandedId(null);
      setExecutions([]);
      return;
    }
    setExpandedId(ruleId);
    setLoadingExec(true);
    try {
      const res = await rulesApi.listExecutions(ruleId);
      setExecutions(res.data ?? []);
    } catch {
      setExecutions([]);
    } finally {
      setLoadingExec(false);
    }
  }

  // Mapa de trigger events para labels
  const triggerLabel = (event: string): string => {
    const map = t.crm.automations.triggerEvents as Record<string, string>;
    return map[event] ?? event;
  };

  const activeCount = list.filter(r => r.is_active).length;

  const EXEC_STATUS_COLORS: Record<string, { bg: string; text: string }> = {
    success: { bg: 'bg-emerald-100', text: 'text-emerald-700' },
    failed:  { bg: 'bg-red-100',     text: 'text-red-600'     },
    skipped: { bg: 'bg-neutral-100', text: 'text-neutral-500' },
  };

  return (
    <div className="p-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-neutral-900">{t.crm.automations.title}</h1>
          <p className="mt-0.5 text-sm text-neutral-500">{t.crm.automations.description}</p>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-xs text-neutral-500">
            {t.crm.automations.activeCount.replace('{n}', String(activeCount))}
            {' / '}
            {t.crm.automations.rulesCount.replace('{n}', String(list.length))}
          </span>
          {/* NOTE: El formulario de crear regla es complejo — futuro Task 6D (visual builder).
              Por ahora solo link al endpoint. */}
        </div>
      </div>

      {/* Contenido */}
      {loading ? (
        <div className="flex justify-center py-16"><Spinner size="lg" /></div>
      ) : list.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <Zap className="mb-3 h-12 w-12 text-neutral-200" />
          <p className="text-sm font-medium text-neutral-500">{t.crm.automations.noRules}</p>
          <p className="mt-1 text-xs text-neutral-400">{t.crm.automations.noRulesDesc}</p>
        </div>
      ) : (
        <div className="space-y-3">
          {list.map(rule => {
            const isExpanded = expandedId === rule.id;

            return (
              <div
                key={rule.id}
                className="rounded-xl border border-neutral-200 bg-white overflow-hidden transition-shadow hover:shadow-sm"
              >
                {/* Fila principal */}
                <div className="flex items-center gap-3 px-4 py-3">
                  {/* Toggle activo/inactivo */}
                  <button
                    type="button"
                    role="switch"
                    aria-checked={rule.is_active}
                    onClick={() => toggleActive(rule)}
                    className={`relative h-5 w-9 rounded-full transition-colors flex-shrink-0 focus:outline-none focus:ring-2 focus:ring-primary-400 focus:ring-offset-1 ${
                      rule.is_active ? 'bg-emerald-500' : 'bg-neutral-300'
                    }`}
                  >
                    <span className={`absolute top-0.5 h-4 w-4 rounded-full bg-white shadow transition-transform ${
                      rule.is_active ? 'translate-x-4' : 'translate-x-0.5'
                    }`} />
                  </button>

                  {/* Nombre y descripcion */}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-neutral-900">{rule.name}</p>
                    {rule.description && (
                      <p className="text-xs text-neutral-400 truncate">{rule.description}</p>
                    )}
                  </div>

                  {/* Trigger type badge */}
                  <Badge variant={rule.trigger_type === 'event' ? 'primary' : 'default'}>
                    {rule.trigger_type === 'event' ? t.crm.automations.event : t.crm.automations.temporal}
                  </Badge>

                  {/* Trigger event */}
                  <span className="text-xs text-neutral-500 max-w-[160px] truncate hidden md:inline">
                    {triggerLabel(rule.trigger_event)}
                  </span>

                  {/* Cooldown */}
                  {rule.cooldown_hours > 0 && (
                    <span className="text-[10px] text-neutral-400 hidden lg:inline">
                      {t.crm.automations.cooldownHours.replace('{n}', String(rule.cooldown_hours))}
                    </span>
                  )}

                  {/* Acciones */}
                  <div className="flex items-center gap-1 flex-shrink-0">
                    <button
                      onClick={() => toggleExpand(rule.id)}
                      className="rounded p-1 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600"
                      title={t.crm.automations.executions}
                    >
                      {isExpanded ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                    </button>
                    <button
                      onClick={() => handleDelete(rule.id)}
                      className="rounded p-1 text-neutral-400 hover:bg-red-50 hover:text-red-500"
                      title={t.crm.automations.deleteRule}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>

                {/* Panel de ejecuciones expandido */}
                {isExpanded && (
                  <div className="border-t border-neutral-100 bg-neutral-50 px-4 py-3">
                    <h4 className="text-xs font-semibold text-neutral-600 mb-2">
                      {t.crm.automations.executions}
                    </h4>

                    {loadingExec ? (
                      <div className="flex justify-center py-4">
                        <Spinner size="sm" />
                      </div>
                    ) : executions.length === 0 ? (
                      <p className="text-xs text-neutral-400 py-2">{t.crm.automations.noExecutions}</p>
                    ) : (
                      <div className="space-y-1 max-h-[240px] overflow-y-auto">
                        {executions.slice(0, 20).map(exec => {
                          const esc = EXEC_STATUS_COLORS[exec.status] ?? EXEC_STATUS_COLORS.skipped;
                          return (
                            <div
                              key={exec.id}
                              className="flex items-center gap-3 rounded-md bg-white px-3 py-2 text-xs"
                            >
                              <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium ${esc.bg} ${esc.text}`}>
                                {exec.status === 'success' && t.crm.automations.executionSuccess}
                                {exec.status === 'failed' && t.crm.automations.executionFailed}
                                {exec.status === 'skipped' && t.crm.automations.executionSkipped}
                              </span>
                              <span className="text-neutral-500">
                                {format(new Date(exec.triggered_at), 'd MMM HH:mm', { locale: dateLocale })}
                              </span>
                              <span className="text-neutral-400 truncate flex-1">
                                {triggerLabel(exec.trigger_event)}
                              </span>
                              {exec.error_message && (
                                <span className="text-red-500 truncate max-w-[200px]" title={exec.error_message}>
                                  <AlertCircle className="h-3 w-3 inline mr-1" />
                                  {exec.error_message}
                                </span>
                              )}
                            </div>
                          );
                        })}
                      </div>
                    )}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
```

---

## Verification Checklist

After implementing all tasks, verify:

- [ ] `apps/web/lib/api.ts` compiles without TypeScript errors (all new types + modules)
- [ ] `apps/web/lib/i18n/locales/es.ts` and `en.ts` have identical key structures in the `crm` section
- [ ] `apps/web/lib/i18n/locales/pt.ts` has the same `crm` key structure
- [ ] Sidebar shows 4 new nav items between Clients and Services
- [ ] All 4 pages render: `/dashboard/pipeline`, `/dashboard/treatments`, `/dashboard/tasks`, `/dashboard/automations`
- [ ] Each page shows: loading state -> empty state -> data (conditional rendering)
- [ ] Each page uses `useTranslations()` for all user-visible strings (no hardcoded Spanish)
- [ ] No `any` types in new code
- [ ] Mobile-responsive layout on all pages (test at 375px width)
- [ ] No direct `fetch()` calls — all API calls go through `lib/api.ts`
