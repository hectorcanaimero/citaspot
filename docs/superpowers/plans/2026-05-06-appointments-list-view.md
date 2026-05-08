# Appointments List View — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a "Lista" tab to the Agenda page with a filterable, sortable, paginated data table of appointments.

**Architecture:** Extend the existing `GET /api/v1/appointments` handler to support date ranges, filters, search, sorting, and pagination. Add a new `ListFiltered` method to the repo/service layer. Create a new `AppointmentsList` React component rendered when the "Lista" tab is active.

**Tech Stack:** Go (Fiber + pgx) backend, Next.js 14 + Tailwind CSS frontend, date-fns for formatting, existing i18n system.

**Spec:** `docs/superpowers/specs/2026-05-06-appointments-list-view-design.md`

---

## File Structure

### Backend (apps/api)

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/domain/types.go` | Modify | Add `AppointmentListQuery` and `PaginatedAppointments` types |
| `internal/domain/interfaces.go` | Modify | Add `ListFiltered` to `AppointmentRepository` and `AppointmentSvc` |
| `internal/repository/appointments.go` | Modify | Implement `ListFiltered` with dynamic SQL, COUNT, LIMIT/OFFSET |
| `internal/service/appointments.go` | Modify | Implement `ListFiltered` pass-through |
| `internal/handler/appointments.go` | Modify | Add `ListFiltered` handler parsing query params |
| `cmd/server/main.go` | Modify | Register new route `GET /appointments/search` |

### Frontend (apps/web)

| File | Action | Responsibility |
|------|--------|----------------|
| `lib/api.ts` | Modify | Add `appointments.listFiltered()` method |
| `lib/i18n/locales/es.ts` | Modify | Add list view translation keys |
| `lib/i18n/locales/en.ts` | Modify | Add list view translation keys |
| `lib/i18n/locales/pt.ts` | Modify | Add list view translation keys |
| `components/dashboard/appointments-list.tsx` | Create | Full list component: filters + table + pagination + actions |
| `app/(dashboard)/agenda/page.tsx` | Modify | Add "Lista" tab to view switcher, render component |

---

## Task 1: Backend — Domain Types

**Files:**
- Modify: `apps/api/internal/domain/types.go` (after line 267, after `AvailabilityQuery`)

- [ ] **Step 1: Add query and response types**

Add after the `AvailabilityQuery` struct (line 267):

```go
// AppointmentListQuery filtros para listar citas con paginación.
type AppointmentListQuery struct {
	DateFrom       string     `json:"date_from"       validate:"required"`
	DateTo         string     `json:"date_to"         validate:"required"`
	Timezone       string     `json:"timezone"`
	ProfessionalID *uuid.UUID `json:"professional_id"`
	ServiceID      *uuid.UUID `json:"service_id"`
	Status         string     `json:"status"`
	Search         string     `json:"search"`
	SortBy         string     `json:"sort_by"`  // date, time, customer_name
	SortDir        string     `json:"sort_dir"` // asc, desc
	Page           int        `json:"page"`
	PerPage        int        `json:"per_page"`
}

// PaginatedAppointments resultado paginado de citas.
type PaginatedAppointments struct {
	Data       []*AppointmentWithDetails `json:"data"`
	Pagination Pagination                `json:"pagination"`
}

// Pagination metadatos de paginación.
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd apps/api && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/domain/types.go
git commit -m "feat(api): add AppointmentListQuery and PaginatedAppointments types"
```

---

## Task 2: Backend — Interfaces

**Files:**
- Modify: `apps/api/internal/domain/interfaces.go` (lines 110-125)

- [ ] **Step 1: Add ListFiltered to AppointmentRepository**

Add after line 113 (after `ListByDate`):

```go
ListFiltered(ctx context.Context, tenantID uuid.UUID, q *AppointmentListQuery) (*PaginatedAppointments, error)
```

The full interface block becomes:

```go
type AppointmentRepository interface {
	Create(ctx context.Context, a *Appointment) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*AppointmentWithDetails, error)
	ListByDate(ctx context.Context, tenantID uuid.UUID, date, timezone string) ([]*AppointmentWithDetails, error)
	ListFiltered(ctx context.Context, tenantID uuid.UUID, q *AppointmentListQuery) (*PaginatedAppointments, error)
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, req *UpdateAppointmentRequest) error
	CheckConflict(ctx context.Context, tenantID, professionalID uuid.UUID, startsAt, endsAt time.Time, excludeID *uuid.UUID) (bool, error)
}
```

- [ ] **Step 2: Add ListFiltered to AppointmentSvc**

Add after line 120 (after `List`):

```go
ListFiltered(ctx context.Context, tenantID uuid.UUID, q *AppointmentListQuery) (*PaginatedAppointments, error)
```

The full interface block becomes:

```go
type AppointmentSvc interface {
	List(ctx context.Context, tenantID uuid.UUID, date, timezone string) ([]*AppointmentWithDetails, error)
	ListFiltered(ctx context.Context, tenantID uuid.UUID, q *AppointmentListQuery) (*PaginatedAppointments, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*AppointmentWithDetails, error)
	Create(ctx context.Context, tenantID uuid.UUID, req *CreateAppointmentRequest) (*Appointment, error)
	Update(ctx context.Context, tenantID, id uuid.UUID, req *UpdateAppointmentRequest) error
	Cancel(ctx context.Context, tenantID, id uuid.UUID, reason string) error
}
```

- [ ] **Step 3: Verify it compiles (expect errors — implementations missing)**

Run: `cd apps/api && go build ./... 2>&1 | head -10`
Expected: compile errors about missing `ListFiltered` method on concrete types — this is correct, we implement in next tasks.

- [ ] **Step 4: Commit**

```bash
git add apps/api/internal/domain/interfaces.go
git commit -m "feat(api): add ListFiltered to appointment interfaces"
```

---

## Task 3: Backend — Repository Implementation

**Files:**
- Modify: `apps/api/internal/repository/appointments.go` (add after `ListByDate`, before `UpdateStatus`)

- [ ] **Step 1: Implement ListFiltered**

Add the following method after line 158 (after `ListByDate`):

```go
// ListFiltered retorna citas filtradas con paginación.
func (r *appointmentRepository) ListFiltered(ctx context.Context, tenantID uuid.UUID, q *domain.AppointmentListQuery) (*domain.PaginatedAppointments, error) {
	tz := q.Timezone
	if tz == "" {
		tz = "UTC"
	}

	// Defaults de paginación
	page := q.Page
	if page < 1 {
		page = 1
	}
	perPage := q.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	// Construir cláusula WHERE dinámica
	where := "WHERE ap.tenant_id = $1 AND DATE(ap.starts_at AT TIME ZONE $2) >= $3::DATE AND DATE(ap.starts_at AT TIME ZONE $2) <= $4::DATE"
	args := []any{tenantID, tz, q.DateFrom, q.DateTo}
	argN := 5

	if q.ProfessionalID != nil {
		where += fmt.Sprintf(" AND ap.professional_id = $%d", argN)
		args = append(args, *q.ProfessionalID)
		argN++
	}
	if q.ServiceID != nil {
		where += fmt.Sprintf(" AND ap.service_id = $%d", argN)
		args = append(args, *q.ServiceID)
		argN++
	}
	if q.Status != "" {
		where += fmt.Sprintf(" AND ap.status = $%d", argN)
		args = append(args, q.Status)
		argN++
	}
	if q.Search != "" {
		where += fmt.Sprintf(" AND c.name ILIKE $%d", argN)
		args = append(args, "%"+q.Search+"%")
		argN++
	}

	// Orden
	orderCol := "ap.starts_at"
	switch q.SortBy {
	case "customer_name":
		orderCol = "c.name"
	}
	orderDir := "DESC"
	if q.SortDir == "asc" {
		orderDir = "ASC"
	}
	orderClause := fmt.Sprintf("ORDER BY %s %s", orderCol, orderDir)

	joins := `
		JOIN customers     c ON c.id = ap.customer_id
		JOIN professionals p ON p.id = ap.professional_id
		JOIN services      s ON s.id = ap.service_id`

	result := &domain.PaginatedAppointments{}

	err := withTenant(ctx, r.db, tenantID, func(tx pgx.Tx) error {
		// COUNT total
		countSQL := fmt.Sprintf("SELECT COUNT(*) FROM appointments ap %s %s", joins, where)
		if err := tx.QueryRow(ctx, countSQL, args...).Scan(&result.Pagination.Total); err != nil {
			return fmt.Errorf("appointmentRepository.ListFiltered: count: %w", err)
		}

		// DATA con LIMIT/OFFSET
		dataSQL := fmt.Sprintf(`
			SELECT
				ap.id, ap.tenant_id, ap.customer_id, ap.professional_id, ap.service_id,
				ap.starts_at, ap.ends_at, ap.status, ap.source, ap.price, ap.notes,
				ap.internal_notes, ap.confirmed_at, ap.cancelled_at, ap.cancellation_reason,
				ap.created_at, ap.updated_at,
				c.name, c.phone,
				p.name,
				s.name, s.duration_min
			FROM appointments ap
			%s
			%s
			%s
			LIMIT %d OFFSET %d
		`, joins, where, orderClause, perPage, offset)

		rows, err := tx.Query(ctx, dataSQL, args...)
		if err != nil {
			return fmt.Errorf("appointmentRepository.ListFiltered: query: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			a := &domain.AppointmentWithDetails{}
			var notes, internalNotes, cancellationReason *string
			if err := rows.Scan(
				&a.ID, &a.TenantID, &a.CustomerID, &a.ProfessionalID, &a.ServiceID,
				&a.StartsAt, &a.EndsAt, &a.Status, &a.Source, &a.Price, &notes,
				&internalNotes, &a.ConfirmedAt, &a.CancelledAt, &cancellationReason,
				&a.CreatedAt, &a.UpdatedAt,
				&a.CustomerName, &a.CustomerPhone,
				&a.ProfessionalName,
				&a.ServiceName, &a.ServiceDuration,
			); err != nil {
				return fmt.Errorf("appointmentRepository.ListFiltered: scan: %w", err)
			}
			if notes != nil {
				a.Notes = *notes
			}
			if internalNotes != nil {
				a.InternalNotes = *internalNotes
			}
			if cancellationReason != nil {
				a.CancellationReason = *cancellationReason
			}
			result.Data = append(result.Data, a)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	result.Pagination.Page = page
	result.Pagination.PerPage = perPage
	total := result.Pagination.Total
	result.Pagination.TotalPages = (total + perPage - 1) / perPage
	if result.Data == nil {
		result.Data = []*domain.AppointmentWithDetails{}
	}

	return result, nil
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd apps/api && go build ./... 2>&1 | head -5`
Expected: Only errors about missing service method (not repository)

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/repository/appointments.go
git commit -m "feat(api): implement ListFiltered repository with dynamic filters and pagination"
```

---

## Task 4: Backend — Service Implementation

**Files:**
- Modify: `apps/api/internal/service/appointments.go` (add after `List` method)

- [ ] **Step 1: Implement ListFiltered in service**

Add after the existing `List` method (after line 35):

```go
// ListFiltered retorna citas filtradas con paginación.
func (s *appointmentService) ListFiltered(ctx context.Context, tenantID uuid.UUID, q *domain.AppointmentListQuery) (*domain.PaginatedAppointments, error) {
	return s.apptRepo.ListFiltered(ctx, tenantID, q)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd apps/api && go build ./...`
Expected: BUILD SUCCESS (all interfaces satisfied)

- [ ] **Step 3: Commit**

```bash
git add apps/api/internal/service/appointments.go
git commit -m "feat(api): implement ListFiltered service method"
```

---

## Task 5: Backend — Handler + Route

**Files:**
- Modify: `apps/api/internal/handler/appointments.go` (add after `List` handler)
- Modify: `apps/api/cmd/server/main.go` (register route)

- [ ] **Step 1: Add ListFiltered handler**

Add the following imports to the existing import block in `appointments.go`:

```go
"strconv"
```

Add the handler after the existing `List` handler (after line 49):

```go
// ListFiltered GET /appointments/search?date_from=&date_to=&professional_id=&service_id=&status=&search=&sort_by=&sort_dir=&page=&per_page=
func (h *AppointmentHandler) ListFiltered(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	tenant := middleware.TenantFromContext(c)

	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	if dateFrom == "" || dateTo == "" {
		return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "Los parámetros 'date_from' y 'date_to' son requeridos (YYYY-MM-DD)"})
	}

	timezone := c.Query("timezone")
	if timezone == "" && tenant != nil {
		timezone = tenant.Timezone
	}

	q := &domain.AppointmentListQuery{
		DateFrom: dateFrom,
		DateTo:   dateTo,
		Timezone: timezone,
		Status:   c.Query("status"),
		Search:   c.Query("search"),
		SortBy:   c.Query("sort_by"),
		SortDir:  c.Query("sort_dir"),
	}

	if profID := c.Query("professional_id"); profID != "" {
		id, err := uuid.Parse(profID)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "professional_id inválido"})
		}
		q.ProfessionalID = &id
	}
	if svcID := c.Query("service_id"); svcID != "" {
		id, err := uuid.Parse(svcID)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(errorResponse{Error: "service_id inválido"})
		}
		q.ServiceID = &id
	}
	if p := c.Query("page"); p != "" {
		q.Page, _ = strconv.Atoi(p)
	}
	if pp := c.Query("per_page"); pp != "" {
		q.PerPage, _ = strconv.Atoi(pp)
	}

	result, err := h.apptSvc.ListFiltered(c.Context(), tenantID, q)
	if err != nil {
		return handleServiceError(c, err)
	}
	return c.JSON(result)
}
```

- [ ] **Step 2: Register route**

In `apps/api/cmd/server/main.go`, find the appointments route group (around line 367-373). Add the new route BEFORE the `/:id` pattern:

```go
appts := protected.Group("/appointments")
appts.Get("/availability", apptHandler.Availability)
appts.Get("/search", apptHandler.ListFiltered)  // ← ADD THIS LINE
appts.Get("/", apptHandler.List)
appts.Post("/", apptHandler.Create)
appts.Get("/:id", apptHandler.GetByID)
appts.Patch("/:id", apptHandler.Update)
appts.Delete("/:id/cancel", apptHandler.Cancel)
```

- [ ] **Step 3: Verify it compiles**

Run: `cd apps/api && go build ./...`
Expected: BUILD SUCCESS

- [ ] **Step 4: Commit**

```bash
git add apps/api/internal/handler/appointments.go apps/api/cmd/server/main.go
git commit -m "feat(api): add ListFiltered handler at GET /appointments/search"
```

---

## Task 6: Frontend — API Client

**Files:**
- Modify: `apps/web/lib/api.ts` (inside the `appointments` object, after the `list` method)

- [ ] **Step 1: Add filter params type and listFiltered method**

Add the following interface before the `appointments` object (find a suitable location near other type definitions around line 130):

```typescript
export interface AppointmentListParams {
  date_from: string;
  date_to: string;
  timezone?: string;
  professional_id?: string;
  service_id?: string;
  status?: string;
  search?: string;
  sort_by?: string;
  sort_dir?: string;
  page?: number;
  per_page?: number;
}

export interface PaginatedAppointments {
  data: Appointment[];
  pagination: {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
  };
}
```

Add the following method inside the `appointments` object, after the `list` method:

```typescript
async listFiltered(params: AppointmentListParams): Promise<PaginatedAppointments> {
  const tz = params.timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone;
  const searchParams = new URLSearchParams();
  searchParams.set('date_from', params.date_from);
  searchParams.set('date_to', params.date_to);
  searchParams.set('timezone', tz);
  if (params.professional_id) searchParams.set('professional_id', params.professional_id);
  if (params.service_id) searchParams.set('service_id', params.service_id);
  if (params.status) searchParams.set('status', params.status);
  if (params.search) searchParams.set('search', params.search);
  if (params.sort_by) searchParams.set('sort_by', params.sort_by);
  if (params.sort_dir) searchParams.set('sort_dir', params.sort_dir);
  if (params.page) searchParams.set('page', String(params.page));
  if (params.per_page) searchParams.set('per_page', String(params.per_page));
  return request(`/api/v1/appointments/search?${searchParams.toString()}`);
},
```

- [ ] **Step 2: Commit**

```bash
git add apps/web/lib/api.ts
git commit -m "feat(web): add listFiltered API method with filter params"
```

---

## Task 7: Frontend — i18n Keys

**Files:**
- Modify: `apps/web/lib/i18n/locales/es.ts`
- Modify: `apps/web/lib/i18n/locales/en.ts`
- Modify: `apps/web/lib/i18n/locales/pt.ts`

- [ ] **Step 1: Add Spanish keys**

In `es.ts`, inside the `agenda` object, add:

```typescript
list: 'Lista',
filterDateFrom: 'Desde',
filterDateTo: 'Hasta',
filterProfessional: 'Profesional',
filterService: 'Servicio',
filterStatus: 'Estado',
filterSearch: 'Buscar cliente...',
filterAll: 'Todos',
colDate: 'Fecha',
colTime: 'Hora',
colClient: 'Cliente',
colService: 'Servicio',
colProfessional: 'Profesional',
colStatus: 'Estado',
colActions: 'Acciones',
actionConfirm: 'Confirmar',
actionComplete: 'Completar',
actionNoShow: 'No asistió',
actionCancel: 'Cancelar',
showingOf: 'Mostrando {from}–{to} de {total} citas',
prevPage: 'Anterior',
nextPage: 'Siguiente',
pageOf: 'Página {page} de {total}',
emptyTitle: 'No hay citas para los filtros seleccionados',
emptySubtitle: 'Intenta cambiar el rango de fechas o los filtros.',
```

- [ ] **Step 2: Add English keys**

In `en.ts`, inside the `agenda` object, add:

```typescript
list: 'List',
filterDateFrom: 'From',
filterDateTo: 'To',
filterProfessional: 'Professional',
filterService: 'Service',
filterStatus: 'Status',
filterSearch: 'Search client...',
filterAll: 'All',
colDate: 'Date',
colTime: 'Time',
colClient: 'Client',
colService: 'Service',
colProfessional: 'Professional',
colStatus: 'Status',
colActions: 'Actions',
actionConfirm: 'Confirm',
actionComplete: 'Complete',
actionNoShow: 'No show',
actionCancel: 'Cancel',
showingOf: 'Showing {from}–{to} of {total} appointments',
prevPage: 'Previous',
nextPage: 'Next',
pageOf: 'Page {page} of {total}',
emptyTitle: 'No appointments match the selected filters',
emptySubtitle: 'Try changing the date range or filters.',
```

- [ ] **Step 3: Add Portuguese keys**

In `pt.ts`, inside the `agenda` object, add:

```typescript
list: 'Lista',
filterDateFrom: 'De',
filterDateTo: 'Até',
filterProfessional: 'Profissional',
filterService: 'Serviço',
filterStatus: 'Status',
filterSearch: 'Buscar cliente...',
filterAll: 'Todos',
colDate: 'Data',
colTime: 'Hora',
colClient: 'Cliente',
colService: 'Serviço',
colProfessional: 'Profissional',
colStatus: 'Status',
colActions: 'Ações',
actionConfirm: 'Confirmar',
actionComplete: 'Completar',
actionNoShow: 'Não compareceu',
actionCancel: 'Cancelar',
showingOf: 'Mostrando {from}–{to} de {total} consultas',
prevPage: 'Anterior',
nextPage: 'Próxima',
pageOf: 'Página {page} de {total}',
emptyTitle: 'Nenhuma consulta para os filtros selecionados',
emptySubtitle: 'Tente mudar o intervalo de datas ou os filtros.',
```

- [ ] **Step 4: Commit**

```bash
git add apps/web/lib/i18n/locales/es.ts apps/web/lib/i18n/locales/en.ts apps/web/lib/i18n/locales/pt.ts
git commit -m "feat(web): add i18n keys for appointments list view"
```

---

## Task 8: Frontend — AppointmentsList Component

**Files:**
- Create: `apps/web/components/dashboard/appointments-list.tsx`

- [ ] **Step 1: Create the component**

This is the main component. It handles filters, data fetching, table rendering, actions, and pagination.

```tsx
'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Check,
  X,
  UserX,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  ChevronUp,
  ClipboardList,
  Search,
} from 'lucide-react';
import { format, startOfWeek, endOfWeek, parseISO } from 'date-fns';

import { useTranslations, useDateLocale } from '@/lib/i18n';
import {
  appointments as apptApi,
  professionals as profApi,
  services as svcApi,
} from '@/lib/api';
import type { Appointment, PaginatedAppointments, Professional, Service } from '@/lib/api';

// ── Status colors ────────────────────────────────────────────────────────────

const STATUS_COLORS: Record<
  Appointment['status'],
  { bg: string; text: string }
> = {
  pending:   { bg: 'bg-amber-50',   text: 'text-amber-800'   },
  confirmed: { bg: 'bg-primary-50', text: 'text-primary-800' },
  completed: { bg: 'bg-emerald-50', text: 'text-emerald-800' },
  cancelled: { bg: 'bg-red-50',     text: 'text-red-700'     },
  no_show:   { bg: 'bg-neutral-100',text: 'text-neutral-600' },
};

const STATUS_LABELS: Record<string, Record<Appointment['status'], string>> = {
  es: { pending: 'Pendiente', confirmed: 'Confirmada', completed: 'Completada', cancelled: 'Cancelada', no_show: 'No asistió' },
  en: { pending: 'Pending', confirmed: 'Confirmed', completed: 'Completed', cancelled: 'Cancelled', no_show: 'No show' },
  pt: { pending: 'Pendente', confirmed: 'Confirmada', completed: 'Completada', cancelled: 'Cancelada', no_show: 'Não compareceu' },
};

const PER_PAGE = 20;

// ── Helper: format date as "jue 15 mayo" ─────────────────────────────────────

function friendlyDate(isoStr: string, locale: Locale): string {
  const d = parseISO(isoStr);
  return format(d, 'EEE d MMM', { locale });
}

function friendlyTime(isoStr: string): string {
  return format(parseISO(isoStr), 'hh:mm a');
}

// ── Component ────────────────────────────────────────────────────────────────

export default function AppointmentsList() {
  const t = useTranslations();
  const dateLocale = useDateLocale();
  const lang = dateLocale.code?.slice(0, 2) ?? 'es';

  // Filters
  const now = new Date();
  const [dateFrom, setDateFrom] = useState(() => format(startOfWeek(now, { weekStartsOn: 1 }), 'yyyy-MM-dd'));
  const [dateTo, setDateTo] = useState(() => format(endOfWeek(now, { weekStartsOn: 1 }), 'yyyy-MM-dd'));
  const [profFilter, setProfFilter] = useState('');
  const [svcFilter, setSvcFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [search, setSearch] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');
  const [sortBy, setSortBy] = useState('date');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc');
  const [page, setPage] = useState(1);

  // Data
  const [result, setResult] = useState<PaginatedAppointments | null>(null);
  const [loading, setLoading] = useState(true);
  const [profs, setProfs] = useState<Professional[]>([]);
  const [svcs, setSvcs] = useState<Service[]>([]);

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 300);
    return () => clearTimeout(timer);
  }, [search]);

  // Load professionals and services for dropdowns
  useEffect(() => {
    profApi.list().then((r) => setProfs(r.data ?? [])).catch(() => {});
    svcApi.list().then((r) => setSvcs(r.data ?? [])).catch(() => {});
  }, []);

  // Fetch data
  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apptApi.listFiltered({
        date_from: dateFrom,
        date_to: dateTo,
        professional_id: profFilter || undefined,
        service_id: svcFilter || undefined,
        status: statusFilter || undefined,
        search: debouncedSearch || undefined,
        sort_by: sortBy,
        sort_dir: sortDir,
        page,
        per_page: PER_PAGE,
      });
      setResult(res);
    } catch {
      setResult(null);
    } finally {
      setLoading(false);
    }
  }, [dateFrom, dateTo, profFilter, svcFilter, statusFilter, debouncedSearch, sortBy, sortDir, page]);

  useEffect(() => { fetchData(); }, [fetchData]);

  // Reset page when filters change
  useEffect(() => { setPage(1); }, [dateFrom, dateTo, profFilter, svcFilter, statusFilter, debouncedSearch]);

  // Actions
  const updateStatus = async (id: string, status: string) => {
    try {
      await apptApi.updateStatus(id, { status });
      fetchData();
    } catch { /* ignore */ }
  };

  // Sort handler
  const toggleSort = (col: string) => {
    if (sortBy === col) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortBy(col);
      setSortDir('asc');
    }
  };

  const SortIcon = ({ col }: { col: string }) => {
    if (sortBy !== col) return <ChevronDown className="h-3 w-3 text-neutral-300" />;
    return sortDir === 'asc'
      ? <ChevronUp className="h-3 w-3 text-primary-600" />
      : <ChevronDown className="h-3 w-3 text-primary-600" />;
  };

  const data = result?.data ?? [];
  const pagination = result?.pagination;

  return (
    <div className="space-y-4">
      {/* ── Filtros ───────────────────────────────────────────────── */}
      <div className="flex flex-wrap items-center gap-2 rounded-xl border border-neutral-200 bg-neutral-50 p-3">
        <input
          type="date"
          value={dateFrom}
          onChange={(e) => setDateFrom(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-xs"
          title={t.agenda.filterDateFrom}
        />
        <span className="text-xs text-neutral-400">–</span>
        <input
          type="date"
          value={dateTo}
          onChange={(e) => setDateTo(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-xs"
          title={t.agenda.filterDateTo}
        />

        <select
          value={profFilter}
          onChange={(e) => setProfFilter(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-xs"
        >
          <option value="">{t.agenda.filterAll} — {t.agenda.filterProfessional}</option>
          {profs.map((p) => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>

        <select
          value={svcFilter}
          onChange={(e) => setSvcFilter(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-xs"
        >
          <option value="">{t.agenda.filterAll} — {t.agenda.filterService}</option>
          {svcs.map((s) => (
            <option key={s.id} value={s.id}>{s.name}</option>
          ))}
        </select>

        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="rounded-lg border border-neutral-200 bg-white px-3 py-1.5 text-xs"
        >
          <option value="">{t.agenda.filterAll} — {t.agenda.filterStatus}</option>
          {(['pending', 'confirmed', 'completed', 'cancelled', 'no_show'] as const).map((s) => (
            <option key={s} value={s}>{(STATUS_LABELS[lang] ?? STATUS_LABELS.es)[s]}</option>
          ))}
        </select>

        <div className="relative ml-auto">
          <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-neutral-400" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder={t.agenda.filterSearch}
            className="rounded-lg border border-neutral-200 bg-white py-1.5 pl-8 pr-3 text-xs w-48"
          />
        </div>
      </div>

      {/* ── Tabla ─────────────────────────────────────────────────── */}
      <div className="overflow-hidden rounded-xl border border-neutral-200 bg-white">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-neutral-100 bg-neutral-50 text-left text-xs font-medium uppercase tracking-wide text-neutral-500">
              <th className="px-4 py-3 cursor-pointer select-none" onClick={() => toggleSort('date')}>
                <span className="inline-flex items-center gap-1">{t.agenda.colDate} <SortIcon col="date" /></span>
              </th>
              <th className="px-4 py-3 cursor-pointer select-none" onClick={() => toggleSort('date')}>
                <span className="inline-flex items-center gap-1">{t.agenda.colTime} <SortIcon col="date" /></span>
              </th>
              <th className="px-4 py-3 cursor-pointer select-none" onClick={() => toggleSort('customer_name')}>
                <span className="inline-flex items-center gap-1">{t.agenda.colClient} <SortIcon col="customer_name" /></span>
              </th>
              <th className="px-4 py-3">{t.agenda.colService}</th>
              <th className="px-4 py-3">{t.agenda.colProfessional}</th>
              <th className="px-4 py-3">{t.agenda.colStatus}</th>
              <th className="px-4 py-3">{t.agenda.colActions}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-100">
            {loading ? (
              // Skeleton
              Array.from({ length: 5 }).map((_, i) => (
                <tr key={i}>
                  {Array.from({ length: 7 }).map((_, j) => (
                    <td key={j} className="px-4 py-3">
                      <div className="h-4 w-20 animate-pulse rounded bg-neutral-200" />
                    </td>
                  ))}
                </tr>
              ))
            ) : data.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-4 py-12 text-center">
                  <ClipboardList className="mx-auto mb-2 h-8 w-8 text-neutral-300" />
                  <p className="text-sm font-medium text-neutral-500">{t.agenda.emptyTitle}</p>
                  <p className="text-xs text-neutral-400">{t.agenda.emptySubtitle}</p>
                </td>
              </tr>
            ) : (
              data.map((appt) => {
                const statusColor = STATUS_COLORS[appt.status];
                const statusLabel = (STATUS_LABELS[lang] ?? STATUS_LABELS.es)[appt.status];
                return (
                  <tr key={appt.id} className="hover:bg-neutral-50 transition-colors">
                    <td className="px-4 py-3 text-neutral-700 capitalize">
                      {friendlyDate(appt.starts_at, dateLocale)}
                    </td>
                    <td className="px-4 py-3 font-medium text-neutral-900">
                      {friendlyTime(appt.starts_at)}
                    </td>
                    <td className="px-4 py-3 text-neutral-700">{appt.customer_name}</td>
                    <td className="px-4 py-3 text-neutral-500">{appt.service_name}</td>
                    <td className="px-4 py-3 text-neutral-500">{appt.professional_name}</td>
                    <td className="px-4 py-3">
                      <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium ${statusColor.bg} ${statusColor.text}`}>
                        {statusLabel}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-1">
                        {appt.status === 'pending' && (
                          <>
                            <button
                              onClick={() => updateStatus(appt.id, 'confirmed')}
                              className="rounded-md p-1 text-primary-600 hover:bg-primary-50 transition-colors"
                              title={t.agenda.actionConfirm}
                            >
                              <Check className="h-4 w-4" />
                            </button>
                            <button
                              onClick={() => updateStatus(appt.id, 'cancelled')}
                              className="rounded-md p-1 text-red-500 hover:bg-red-50 transition-colors"
                              title={t.agenda.actionCancel}
                            >
                              <X className="h-4 w-4" />
                            </button>
                          </>
                        )}
                        {appt.status === 'confirmed' && (
                          <>
                            <button
                              onClick={() => updateStatus(appt.id, 'completed')}
                              className="rounded-md p-1 text-emerald-600 hover:bg-emerald-50 transition-colors"
                              title={t.agenda.actionComplete}
                            >
                              <Check className="h-4 w-4" />
                            </button>
                            <button
                              onClick={() => updateStatus(appt.id, 'no_show')}
                              className="rounded-md p-1 text-neutral-500 hover:bg-neutral-100 transition-colors"
                              title={t.agenda.actionNoShow}
                            >
                              <UserX className="h-4 w-4" />
                            </button>
                            <button
                              onClick={() => updateStatus(appt.id, 'cancelled')}
                              className="rounded-md p-1 text-red-500 hover:bg-red-50 transition-colors"
                              title={t.agenda.actionCancel}
                            >
                              <X className="h-4 w-4" />
                            </button>
                          </>
                        )}
                        {['completed', 'cancelled', 'no_show'].includes(appt.status) && (
                          <span className="text-xs text-neutral-300">—</span>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>

        {/* ── Paginación ──────────────────────────────────────────── */}
        {pagination && pagination.total > 0 && (
          <div className="flex items-center justify-between border-t border-neutral-100 bg-neutral-50 px-4 py-2.5 text-xs text-neutral-500">
            <span>
              {t.agenda.showingOf
                .replace('{from}', String((pagination.page - 1) * pagination.per_page + 1))
                .replace('{to}', String(Math.min(pagination.page * pagination.per_page, pagination.total)))
                .replace('{total}', String(pagination.total))}
            </span>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={pagination.page <= 1}
                className="flex items-center gap-1 rounded-md px-2 py-1 hover:bg-neutral-200 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
              >
                <ChevronLeft className="h-3.5 w-3.5" />
                {t.agenda.prevPage}
              </button>
              <span className="text-neutral-700 font-medium">
                {t.agenda.pageOf
                  .replace('{page}', String(pagination.page))
                  .replace('{total}', String(pagination.total_pages))}
              </span>
              <button
                onClick={() => setPage((p) => Math.min(pagination.total_pages, p + 1))}
                disabled={pagination.page >= pagination.total_pages}
                className="flex items-center gap-1 rounded-md px-2 py-1 hover:bg-neutral-200 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
              >
                {t.agenda.nextPage}
                <ChevronRight className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add apps/web/components/dashboard/appointments-list.tsx
git commit -m "feat(web): create AppointmentsList component with filters, table, and pagination"
```

---

## Task 9: Frontend — Wire up the Lista tab in Agenda page

**Files:**
- Modify: `apps/web/app/(dashboard)/agenda/page.tsx`

- [ ] **Step 1: Import the component and List icon**

At the top of the file, add to the existing lucide-react import:

```typescript
import { List } from 'lucide-react';
```

Add the component import:

```typescript
import AppointmentsList from '@/components/dashboard/appointments-list';
```

- [ ] **Step 2: Update the CalView type**

Find line 29 (approximately):

```typescript
type CalView = 'day' | 'week' | 'month';
```

Change to:

```typescript
type CalView = 'day' | 'week' | 'month' | 'lista';
```

- [ ] **Step 3: Add Lista to the view switcher**

Find the view switcher array (around line 527). Add the lista entry:

```typescript
{([
  { key: 'day'   as const, label: t.agenda.day,   icon: AlignLeft    },
  { key: 'week'  as const, label: t.agenda.week,  icon: CalendarDays },
  { key: 'month' as const, label: t.agenda.month, icon: LayoutGrid   },
  { key: 'lista' as const, label: t.agenda.list,  icon: List         },
]).map(({ key, label, icon: Icon }) => (
```

- [ ] **Step 4: Render the list view**

Find the conditional rendering section (around line 615-634). Add after the month view and before the closing tags:

```tsx
{view === 'lista' && <AppointmentsList />}
```

- [ ] **Step 5: Hide calendar-specific controls when in lista view**

The date navigation and availability bar should be hidden when the "Lista" view is active, since the list has its own date filters. Find the header section and wrap the date navigation controls with:

```tsx
{view !== 'lista' && (
  // existing date nav: prev/today/next buttons, period label
)}
```

Also hide the availability toggle section (only visible on day view already, but verify).

- [ ] **Step 6: Verify it compiles**

Run: `cd apps/web && npx next build 2>&1 | tail -20`
Expected: BUILD SUCCESS (or at least no TypeScript errors)

- [ ] **Step 7: Commit**

```bash
git add apps/web/app/\(dashboard\)/agenda/page.tsx
git commit -m "feat(web): add Lista tab to agenda page with appointments list view"
```

---

## Summary

| Task | Layer | What |
|------|-------|------|
| 1 | Backend Domain | `AppointmentListQuery`, `PaginatedAppointments`, `Pagination` types |
| 2 | Backend Domain | `ListFiltered` on interfaces |
| 3 | Backend Repo | Dynamic SQL with filters, COUNT, LIMIT/OFFSET |
| 4 | Backend Service | Pass-through to repo |
| 5 | Backend Handler | Query param parsing + route registration |
| 6 | Frontend API | `listFiltered()` method + types |
| 7 | Frontend i18n | Translation keys (es/en/pt) |
| 8 | Frontend Component | `AppointmentsList` — filters, table, pagination, actions |
| 9 | Frontend Wiring | Add "Lista" tab to agenda page |
