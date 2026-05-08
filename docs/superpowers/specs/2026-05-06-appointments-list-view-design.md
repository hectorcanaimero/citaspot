# Spec: Tab "Lista" en Agenda — DataTable de Citas

**Fecha:** 2026-05-06  
**Estado:** Draft  
**Autor:** Héctor Rodríguez + Claude

---

## Resumen

Agregar un cuarto tab "Lista" a la página de Agenda (`/agenda`) que muestra todas las citas en formato tabla con filtros, paginación y acciones rápidas por fila. Complementa las vistas de calendario existentes (Día/Semana/Mes) para búsqueda y gestión masiva.

---

## Motivación

La vista de calendario es ideal para ver la agenda del día/semana, pero no permite:
- Buscar citas por nombre de cliente
- Filtrar por estado (ej: "todas las canceladas del mes")
- Ver un rango amplio de fechas de forma compacta
- Actuar rápido sobre múltiples citas (confirmar, completar, marcar no-show)

---

## Diseño de UI

### Ubicación

Tab dentro de la página `/agenda` existente. Los tabs quedan:

```
[ Día ] [ Semana ] [ Mes ] [ Lista ]
```

Al seleccionar "Lista", se oculta el calendario y se muestra la tabla con filtros.

### Barra de filtros

Fila horizontal encima de la tabla con los siguientes controles:

| Control | Tipo | Default | Notas |
|---------|------|---------|-------|
| Fecha desde | input date | inicio de semana actual (lunes) | formato nativo del browser |
| Fecha hasta | input date | fin de semana actual (domingo) | - |
| Profesional | select dropdown | "Todos" | populated desde `professionals.list()` |
| Servicio | select dropdown | "Todos" | populated desde `services.list()` |
| Estado | select dropdown | "Todos" | opciones: Pendiente, Confirmada, Completada, Cancelada, No Show |
| Buscar cliente | input text | vacío | búsqueda por nombre, debounce 300ms |

Los filtros se aplican en conjunto (AND). Cambiar cualquier filtro resetea a página 1.

### Tabla

**Columnas:**

| # | Columna | Sortable | Formato |
|---|---------|----------|---------|
| 1 | Fecha | ✓ | `jue 15 mayo` (día semana + día + mes) |
| 2 | Hora | ✓ | `10:00 AM` (12h format) |
| 3 | Cliente | ✓ | Nombre completo |
| 4 | Servicio | ✗ | Nombre del servicio |
| 5 | Profesional | ✗ | Nombre del profesional |
| 6 | Estado | ✗ | Badge con color semántico |
| 7 | Acciones | ✗ | Botones contextuales según estado |

**Orden default:** Fecha DESC, Hora ASC (más recientes primero, dentro del día cronológico).

**Colores de estado (badges):**

| Estado | Background | Text |
|--------|-----------|------|
| Pendiente | amber-100 | amber-800 |
| Confirmada | green-100 | green-800 |
| Completada | blue-100 | blue-800 |
| Cancelada | red-100 | red-800 |
| No Show | neutral-200 | neutral-700 |

### Acciones por fila

Las acciones visibles dependen del estado actual de la cita:

| Estado actual | Acciones |
|---|---|
| Pendiente | ✓ Confirmar · ✕ Cancelar |
| Confirmada | ✓ Completar · ⊘ No Show · ✕ Cancelar |
| Completada | — (sin acciones) |
| Cancelada | — (sin acciones) |
| No Show | — (sin acciones) |

Cada acción muta el estado vía `appointments.updateStatus()` y actualiza la fila in-place sin recargar toda la tabla.

### Paginación

- 20 items por página
- Footer de tabla: "Mostrando X–Y de Z citas" a la izquierda
- Navegación: `← Anterior | Página N de M | Siguiente →` a la derecha
- Se persiste en state local (no en URL para evitar conflictos con los otros tabs)

### Estado vacío

Si no hay citas que coincidan con los filtros:
```
📋 No hay citas para los filtros seleccionados.
Intenta cambiar el rango de fechas o los filtros.
```

### Loading

Skeleton de 5 filas con animación pulse mientras carga.

---

## API Backend

### Endpoint necesario

```
GET /api/v1/appointments
```

**Query params:**

| Param | Tipo | Required | Descripción |
|-------|------|----------|-------------|
| date_from | string (YYYY-MM-DD) | ✓ | Fecha inicio del rango |
| date_to | string (YYYY-MM-DD) | ✓ | Fecha fin del rango |
| professional_id | UUID | ✗ | Filtrar por profesional |
| service_id | UUID | ✗ | Filtrar por servicio |
| status | string | ✗ | Filtrar por estado |
| search | string | ✗ | Búsqueda por nombre de cliente (ILIKE) |
| sort_by | string | ✗ | Campo de orden: `date`, `time`, `customer_name`. Default: `date` |
| sort_dir | string | ✗ | `asc` o `desc`. Default: `desc` |
| page | int | ✗ | Número de página (1-based). Default: 1 |
| per_page | int | ✗ | Items por página. Default: 20, max: 100 |

**Response:**

```json
{
  "data": [
    {
      "id": "uuid",
      "starts_at": "2026-05-15T14:00:00Z",
      "ends_at": "2026-05-15T14:30:00Z",
      "status": "confirmed",
      "customer_name": "María García",
      "customer_phone": "+18095551234",
      "service_name": "Limpieza Bucal",
      "service_duration_min": 30,
      "professional_name": "Maria Luna",
      "price": 25.00,
      "notes": "Primera visita",
      "source": "whatsapp",
      "created_at": "2026-05-10T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "per_page": 20,
    "total": 28,
    "total_pages": 2
  }
}
```

**Notas:**
- El endpoint respeta RLS — solo retorna citas del tenant autenticado
- `search` busca con `ILIKE '%term%'` en `customers.name`
- Los timestamps se retornan en UTC; el frontend convierte a timezone del tenant

### Endpoint existente reutilizado

```
PATCH /api/v1/appointments/:id  — updateStatus (ya existe)
```

---

## Implementación Frontend

### Archivos a crear/modificar

| Archivo | Acción | Descripción |
|---------|--------|-------------|
| `app/(dashboard)/agenda/page.tsx` | Modificar | Agregar tab "Lista" y renderizar `AppointmentsList` |
| `components/dashboard/appointments-list.tsx` | Crear | Componente completo: filtros + tabla + paginación |
| `lib/api.ts` | Modificar | Agregar `appointments.listFiltered()` con los query params |
| `lib/i18n/locales/es.ts` | Modificar | Agregar strings de la vista lista |
| `lib/i18n/locales/en.ts` | Modificar | Agregar strings de la vista lista |
| `lib/i18n/locales/pt.ts` | Modificar | Agregar strings de la vista lista |

### Archivos backend a crear/modificar

| Archivo | Acción | Descripción |
|---------|--------|-------------|
| `apps/api/internal/handler/appointments.go` | Modificar | Agregar handler `List` con filtros y paginación |
| `apps/api/internal/service/appointments.go` | Modificar | Agregar método `List` con lógica de filtrado |
| `apps/api/internal/repository/appointments.go` | Modificar | Agregar query con filtros dinámicos |
| `apps/api/internal/domain/types.go` | Modificar | Agregar `AppointmentListQuery` y `PaginatedResponse` |

### Librerías

No se agregan librerías nuevas. Se usa:
- HTML `<table>` semántico + Tailwind (consistente con página de clientes)
- `date-fns` para formateo de fechas
- API client existente (`lib/api.ts`)

---

## Fuera de scope

- Export a CSV/Excel
- Selección múltiple con acciones en batch
- Drag & drop para reagendar
- Vista responsive para móvil (la tabla scrollea horizontal)
- Filtros persistidos en URL (se mantienen en state local)

---

## Criterios de aceptación

1. El tab "Lista" aparece junto a Día/Semana/Mes y se activa al hacer click
2. La tabla muestra todas las citas del rango seleccionado con los datos correctos
3. Todos los filtros funcionan en conjunto (AND) y actualizan la tabla
4. La búsqueda por cliente tiene debounce y filtra correctamente
5. Las acciones mutan el estado de la cita y actualizan la fila sin reload
6. La paginación funciona y muestra el conteo correcto
7. Los horarios se muestran en la timezone del tenant
8. Las columnas de Fecha, Hora y Cliente son ordenables
9. El estado vacío muestra el mensaje apropiado
10. El loading muestra skeleton de 5 filas
