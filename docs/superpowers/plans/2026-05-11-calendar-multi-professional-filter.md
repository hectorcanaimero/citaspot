# Calendar Multi-Professional Filter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the agenda calendar to support multi-select professional filtering with color-coded appointment blocks (subtle background + 4px left border + professional name in their color).

**Architecture:** All changes are isolated to `apps/web/app/dashboard/agenda/page.tsx`. The state type changes from a single `filterProfId: string` to `filterProfIds: Set<string>`. The `filteredDayMap` memo and filter pills UI are updated accordingly. The `ApptBlock` component gets enhanced color styling.

**Tech Stack:** Next.js 14, React 18, TypeScript strict, Tailwind CSS, date-fns

---

## Files

| Action | File | What changes |
|--------|------|-------------|
| Modify | `apps/web/app/dashboard/agenda/page.tsx:418` | `filterProfId: string` → `filterProfIds: Set<string>` |
| Modify | `apps/web/app/dashboard/agenda/page.tsx:428-439` | `filteredDayMap` memo uses `Set.has()` instead of `===` |
| Modify | `apps/web/app/dashboard/agenda/page.tsx:57-89` | `ApptBlock` adds `backgroundColor` + thicker border + colored professional name |
| Modify | `apps/web/app/dashboard/agenda/page.tsx:598-629` | Filter pills become multi-select toggle |
| Modify | `apps/web/app/dashboard/agenda/page.tsx:371-388` | Month view appointment rows get professional color dot |

---

### Task 1: Change filter state from single to multi-select

**Files:**
- Modify: `apps/web/app/dashboard/agenda/page.tsx:418`
- Modify: `apps/web/app/dashboard/agenda/page.tsx:428-439`

- [ ] **Step 1: Replace `filterProfId` state with `filterProfIds` Set**

In `apps/web/app/dashboard/agenda/page.tsx`, replace line 418:

```ts
// BEFORE (line 418)
const [filterProfId, setFilterProfId] = useState<string>('');

// AFTER
const [filterProfIds, setFilterProfIds] = useState<Set<string>>(new Set());
```

- [ ] **Step 2: Replace `filteredDayMap` memo**

Replace lines 428–439 with:

```ts
// dayMap filtrado por profesionales seleccionados (vacío = todos)
const filteredDayMap = useMemo(() => {
  if (filterProfIds.size === 0) return dayMap;
  const filtered: Record<string, DayState> = {};
  for (const [key, val] of Object.entries(dayMap)) {
    if (Array.isArray(val)) {
      filtered[key] = val.filter(a => filterProfIds.has(a.professional_id));
    } else {
      filtered[key] = val;
    }
  }
  return filtered;
}, [dayMap, filterProfIds]);
```

- [ ] **Step 3: Verify TypeScript compiles without errors**

```bash
cd /Users/al3jandro/project/agendAI/apps/web && npx tsc --noEmit 2>&1 | head -30
```

Expected: no errors related to `filterProfId` (there may be errors in other files, that's ok — focus on `agenda/page.tsx`)

- [ ] **Step 4: Commit**

```bash
cd /Users/al3jandro/project/agendAI
git add apps/web/app/dashboard/agenda/page.tsx
git commit -m "refactor(agenda): change filter state from single to multi-select Set"
```

---

### Task 2: Update ApptBlock with professional color styling

**Files:**
- Modify: `apps/web/app/dashboard/agenda/page.tsx:57-89`

- [ ] **Step 1: Replace the `ApptBlock` component**

Replace lines 57–89 with:

```tsx
function ApptBlock({ appt, profColor }: { appt: ApptWithCol; profColor?: string }) {
  const top    = topPx(appt.starts_at);
  const height = heightPx(appt.service_duration_min);
  const pct    = 100 / appt.span;
  const color  = profColor ?? '#6b7280';

  return (
    <div
      className="absolute z-10 overflow-hidden rounded border border-neutral-200 px-1.5 py-0.5 text-xs cursor-pointer transition-all hover:z-20 hover:shadow-md"
      style={{
        top,
        height,
        width:           `calc(${pct}% - 4px)`,
        left:            `calc(${(appt.col / appt.span) * 100}% + ${appt.col > 0 ? 2 : 0}px)`,
        minWidth:        0,
        borderLeftWidth: '4px',
        borderLeftColor: color,
        backgroundColor: `${color}14`,
      }}
      title={`${appt.customer_name} · ${appt.service_name} · ${appt.professional_name}`}
    >
      <p className="font-semibold leading-tight truncate text-neutral-800">
        {format(new Date(appt.starts_at), 'HH:mm')} {appt.customer_name}
      </p>
      {height >= 38 && (
        <p className="truncate leading-tight text-neutral-500">{appt.service_name}</p>
      )}
      {height >= 54 && (
        <p className="truncate leading-tight font-medium" style={{ color }}>
          {appt.professional_name}
        </p>
      )}
    </div>
  );
}
```

Note: we drop `STATUS_CFG` classes on the block itself (they caused conflicting backgrounds). The status is still visible via the `AppointmentsList` in list view. If you want to keep status indication, add a small status dot instead — but the spec doesn't require it.

- [ ] **Step 2: Verify TypeScript compiles without errors**

```bash
cd /Users/al3jandro/project/agendAI/apps/web && npx tsc --noEmit 2>&1 | head -30
```

- [ ] **Step 3: Commit**

```bash
cd /Users/al3jandro/project/agendAI
git add apps/web/app/dashboard/agenda/page.tsx
git commit -m "feat(agenda): add professional color background to appointment blocks"
```

---

### Task 3: Update filter pills to multi-select toggle

**Files:**
- Modify: `apps/web/app/dashboard/agenda/page.tsx:598-629`

- [ ] **Step 1: Replace the filter pills section**

Replace lines 598–629 with:

```tsx
{/* ── Filtro por profesional (multi-selección) ─────────────────────── */}
{profList.length > 1 && (
  <div className="flex flex-shrink-0 items-center gap-2 border-b border-neutral-100 bg-white px-5 py-2 overflow-x-auto">
    <button
      onClick={() => setFilterProfIds(new Set())}
      className={`flex-shrink-0 rounded-full px-3 py-1 text-xs font-medium transition-all ${
        filterProfIds.size === 0
          ? 'bg-neutral-900 text-white'
          : 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200'
      }`}
    >
      {t.agenda.allProfessionals}
    </button>
    {profList.filter(p => p.is_active && !p.is_archived).map(p => {
      const isSelected = filterProfIds.has(p.id);
      return (
        <button
          key={p.id}
          onClick={() => {
            setFilterProfIds(prev => {
              const next = new Set(prev);
              if (next.has(p.id)) next.delete(p.id);
              else next.add(p.id);
              return next;
            });
          }}
          className="flex-shrink-0 rounded-full px-3 py-1 text-xs font-medium transition-all hover:opacity-80"
          style={{
            backgroundColor: isSelected ? p.color : `${p.color}20`,
            color:           isSelected ? 'white'  : p.color,
            border:          isSelected ? 'none'   : `1.5px solid ${p.color}`,
          }}
        >
          {p.name}
        </button>
      );
    })}
  </div>
)}
```

- [ ] **Step 2: Verify TypeScript compiles without errors**

```bash
cd /Users/al3jandro/project/agendAI/apps/web && npx tsc --noEmit 2>&1 | head -30
```

- [ ] **Step 3: Commit**

```bash
cd /Users/al3jandro/project/agendAI
git add apps/web/app/dashboard/agenda/page.tsx
git commit -m "feat(agenda): multi-select professional filter pills"
```

---

### Task 4: Add professional color dots to MonthView appointment rows

**Files:**
- Modify: `apps/web/app/dashboard/agenda/page.tsx:282-394` (MonthView component)

The `MonthView` component receives `dayMap` but not `profColorMap`. We need to thread it through.

- [ ] **Step 1: Add `profColorMap` prop to MonthView**

Replace the MonthView function signature (line ~282):

```tsx
function MonthView({
  month,
  dayMap,
  onDayClick,
  profColorMap = {},
}: {
  month: Date;
  dayMap: Record<string, DayState>;
  onDayClick: (d: Date) => void;
  profColorMap?: Record<string, string>;
}) {
```

- [ ] **Step 2: Update the appointment row inside MonthView to use professional color**

Replace the appointment row rendering (lines ~372-387, inside the `inMonth && appts.length > 0` block):

```tsx
{inMonth && appts.length > 0 && (
  <div className="mt-1 space-y-0.5">
    {appts.slice(0, 2).map(a => {
      const color = profColorMap[a.professional_id];
      return (
        <p
          key={a.id}
          className="truncate rounded px-1 py-0.5 text-[10px] text-neutral-700"
          style={{
            backgroundColor: color ? `${color}14` : '#f3f4f6',
            borderLeft:      color ? `2px solid ${color}` : '2px solid #d1d5db',
          }}
        >
          {format(new Date(a.starts_at), 'HH:mm')} {a.customer_name}
        </p>
      );
    })}
    {appts.length > 2 && (
      <p className="text-[10px] text-neutral-400">
        {t.common.more.replace('{n}', String(appts.length - 2))}
      </p>
    )}
  </div>
)}
```

- [ ] **Step 3: Pass `profColorMap` to MonthView in the render (line ~707)**

Replace:

```tsx
{view === 'month' && (
  <MonthView
    month={currentDate}
    dayMap={filteredDayMap}
    onDayClick={d => { setCurrentDate(d); setView('day'); }}
  />
)}
```

With:

```tsx
{view === 'month' && (
  <MonthView
    month={currentDate}
    dayMap={filteredDayMap}
    onDayClick={d => { setCurrentDate(d); setView('day'); }}
    profColorMap={profColorMap}
  />
)}
```

- [ ] **Step 4: Verify TypeScript compiles without errors**

```bash
cd /Users/al3jandro/project/agendAI/apps/web && npx tsc --noEmit 2>&1 | head -30
```

- [ ] **Step 5: Commit**

```bash
cd /Users/al3jandro/project/agendAI
git add apps/web/app/dashboard/agenda/page.tsx
git commit -m "feat(agenda): add professional color to month view appointment rows"
```

---

### Task 5: Update Plane issue + final verification

- [ ] **Step 1: Move CITAS-22 to In Progress in Plane**

```bash
source /Users/al3jandro/project/agendAI/.env && \
STATE_ID=$(curl -s -H "X-API-Key: $PLANE_GURIA_KEY" \
  "https://plane.guria.lat/api/v1/workspaces/pideai/projects/a4e7f2ca-dc77-41b1-acb6-df32e23e7460/states/" \
  | python3 -c "import json,sys; states=json.load(sys.stdin)['results']; print(next(s['id'] for s in states if s['group']=='started'))") && \
curl -s -X PATCH -H "X-API-Key: $PLANE_GURIA_KEY" -H "Content-Type: application/json" \
  -d "{\"state\":\"$STATE_ID\"}" \
  "https://plane.guria.lat/api/v1/workspaces/pideai/projects/a4e7f2ca-dc77-41b1-acb6-df32e23e7460/issues/cdf31738-a8c4-470a-a8d1-bbd64e10f1b2/" | python3 -c "import json,sys; r=json.load(sys.stdin); print('OK' if 'id' in r else r)"
```

- [ ] **Step 2: Add analysis comment to CITAS-22**

```bash
source /Users/al3jandro/project/agendAI/.env && \
curl -s -X POST -H "X-API-Key: $PLANE_GURIA_KEY" -H "Content-Type: application/json" \
  -d '{"comment_html":"<p><strong>Análisis CITAS-22 — Filtro multi-profesional con colores</strong></p><p><strong>Plan de ejecución:</strong></p><ul><li>Cambiar estado de filtro de string único a Set&lt;string&gt; para multi-selección</li><li>Pills con toggle individual + \"Todos\" limpia la selección</li><li>ApptBlock: fondo suave (8% opacidad del color del profesional) + borde 4px + nombre en color</li><li>MonthView: filas de citas con color del profesional</li></ul><p><strong>Scope:</strong> Solo apps/web/app/dashboard/agenda/page.tsx. Sin cambios de backend. El campo color ya existe en el modelo Professional.</p>"}' \
  "https://plane.guria.lat/api/v1/workspaces/pideai/projects/a4e7f2ca-dc77-41b1-acb6-df32e23e7460/issues/cdf31738-a8c4-470a-a8d1-bbd64e10f1b2/comments/" | python3 -c "import json,sys; r=json.load(sys.stdin); print('Comment OK' if 'id' in r else r)"
```

- [ ] **Step 3: Manual smoke test**

Verify in browser (run `make dev` if not running):
- [ ] Pills de profesionales muestran borde coloreado cuando no están seleccionadas
- [ ] Click en una pill la selecciona (fondo sólido, texto blanco)
- [ ] Click en otra pill agrega a la selección (multi-select)
- [ ] Click en pill seleccionada la deselecciona
- [ ] "Todos" limpia toda la selección
- [ ] Los bloques de citas muestran fondo suave del color del profesional
- [ ] El borde izquierdo es 4px
- [ ] El nombre del profesional aparece en su color
- [ ] En vista mes, las filas de citas tienen el color del profesional

- [ ] **Step 4: Add done comment + move to Done in Plane**

```bash
source /Users/al3jandro/project/agendAI/.env && \
DONE_STATE_ID=$(curl -s -H "X-API-Key: $PLANE_GURIA_KEY" \
  "https://plane.guria.lat/api/v1/workspaces/pideai/projects/a4e7f2ca-dc77-41b1-acb6-df32e23e7460/states/" \
  | python3 -c "import json,sys; states=json.load(sys.stdin)['results']; print(next(s['id'] for s in states if s['group']=='completed'))") && \
curl -s -X POST -H "X-API-Key: $PLANE_GURIA_KEY" -H "Content-Type: application/json" \
  -d '{"comment_html":"<p><strong>✅ CITAS-22 completado</strong></p><p>Implementado en <code>apps/web/app/dashboard/agenda/page.tsx</code>:</p><ul><li>Estado de filtro cambiado a <code>Set&lt;string&gt;</code> para multi-selección</li><li>Pills con toggle individual — borde coloreado cuando inactiva, fondo sólido cuando activa</li><li>\"Todos\" limpia la selección</li><li>ApptBlock: fondo <code>${color}14</code> + borde 4px + nombre del profesional en su color</li><li>MonthView: filas de citas con color del profesional</li></ul>"}' \
  "https://plane.guria.lat/api/v1/workspaces/pideai/projects/a4e7f2ca-dc77-41b1-acb6-df32e23e7460/issues/cdf31738-a8c4-470a-a8d1-bbd64e10f1b2/comments/" && \
curl -s -X PATCH -H "X-API-Key: $PLANE_GURIA_KEY" -H "Content-Type: application/json" \
  -d "{\"state\":\"$DONE_STATE_ID\"}" \
  "https://plane.guria.lat/api/v1/workspaces/pideai/projects/a4e7f2ca-dc77-41b1-acb6-df32e23e7460/issues/cdf31738-a8c4-470a-a8d1-bbd64e10f1b2/" | python3 -c "import json,sys; r=json.load(sys.stdin); print('Moved to Done' if 'id' in r else r)"
```
