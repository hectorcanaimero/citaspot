# CITAS-22 — Filtro multi-profesional con colores en el calendario

## Contexto

El calendario de agenda actualmente permite filtrar por un solo profesional a la vez (single-select pills). Los bloques de citas muestran únicamente un borde izquierdo de 2px con el color del profesional.

CITAS-22 pide: tabs/pills para filtrar por especialista con selección múltiple y etiquetas de color más prominentes en el calendario.

## Decisiones de diseño

| Aspecto | Decisión |
|---------|----------|
| Control de filtro | Pills multi-selección horizontal (toggle individual) |
| "Todos" | Activo cuando ningún profesional está seleccionado |
| Color en bloques | Fondo ~8% opacidad + borde izquierdo 4px + nombre del profesional en su color |
| Vista con varios seleccionados | Grilla única, citas intercaladas (comportamiento actual, ya maneja solapamientos con `assignColumns`) |

## Alcance

**Un solo archivo:** `apps/web/app/dashboard/agenda/page.tsx`

Sin cambios de backend. Sin nuevas APIs. Sin nuevos componentes.

## Cambios específicos

### 1. Estado

```ts
// Antes
const [selectedProfId, setSelectedProfId] = useState<string | null>(null)

// Después
const [selectedProfIds, setSelectedProfIds] = useState<Set<string>>(new Set())
```

### 2. Toggle function

```ts
function toggleProfessional(id: string) {
  setSelectedProfIds(prev => {
    const next = new Set(prev)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    return next
  })
}
```

### 3. Filtrado de citas (aplicar en todos los views)

```ts
const visibleAppointments = selectedProfIds.size === 0
  ? appointments
  : appointments.filter(a => selectedProfIds.has(a.professional_id))
```

### 4. Estilo de bloques de citas

```tsx
// backgroundColor: color del profesional con ~8% opacidad (hex + "14")
// borderLeftWidth: 4px
// nombre del profesional: <span style={{ color: profColor }}>
```

### 5. Pills UI

- Seleccionada: fondo sólido del color + texto blanco
- No seleccionada: fondo `${color}20` + borde del color + texto del color
- "Todos": activo (fondo sólido primario) cuando `selectedProfIds.size === 0`

## Criterios de aceptación

- [ ] Puedo seleccionar múltiples profesionales simultáneamente
- [ ] Puedo deseleccionar un profesional clickeando su pill nuevamente
- [ ] "Todos" limpia la selección y muestra todas las citas
- [ ] Las citas tienen fondo suave del color del profesional en todos los views
- [ ] El borde izquierdo de las citas es de 4px
- [ ] El nombre del profesional aparece en su color dentro del bloque
- [ ] El filtro funciona en vista día, semana, mes y lista
