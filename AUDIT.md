# CitaSpot — Auditoría de bugs y propuestas de mejora

> **Sistema:** CitaSpot — Gestión de citas con IA  
> **Negocio:** Diente Feliz Venezolano  
> **URL local:** http://localhost:3000  
> **Fecha de auditoría:** 2026-05-15  
> **Auditado por:** Revisión completa de UI/UX y funcionalidad (15 secciones)

---

## Contexto del stack

El sistema es una aplicación Next.js corriendo en `localhost:3000`. Las rutas del sidebar usan inglés internamente (`/dashboard/clients`, `/dashboard/treatments`, etc.) con labels en español. Integra WhatsApp Business API, chatbot con IA y base de conocimiento vectorial.

---

## BUGS CRÍTICOS (prioridad 1 — corregir antes de usar en producción)

---

### BUG-01 — Inconsistencia de tema dark/light en Analíticas y Configuración

**Severidad:** Crítica  
**Páginas afectadas:** `/dashboard/analytics`, `/dashboard/settings`, `/dashboard/settings/*`  
**Síntoma:** El sidebar y fondo de estas dos secciones renderizan en modo oscuro, mientras el 100% del resto del sistema usa tema claro. La transición al navegar a estas secciones es visualmente abrupta.

**Causa probable:**  
Los componentes de layout de estas rutas no heredan el mismo `Layout` wrapper que el resto. Posiblemente tienen su propio layout definido en `app/dashboard/analytics/layout.tsx` y `app/dashboard/settings/layout.tsx` con clases de Tailwind hardcodeadas (`bg-gray-900`, `text-white`, etc.) en lugar de usar las variables CSS del sistema de diseño.

**Propuesta de fix:**

1. Verificar si `app/dashboard/analytics/layout.tsx` y `app/dashboard/settings/layout.tsx` existen como layouts separados.
2. Si existen, eliminarlos y dejar que hereden el layout padre `app/dashboard/layout.tsx`.
3. Si no existen layouts separados, buscar en los componentes de sidebar/nav de estas páginas clases como `bg-gray-900`, `bg-zinc-900`, `dark:` prefixes o `className` con colores hardcodeados y reemplazarlos con las variables del sistema (`bg-background`, `text-foreground` o equivalente).
4. Verificar que el componente `<Sidebar />` usado en analytics/settings sea el mismo que el del resto del dashboard.

```bash
# Buscar archivos que puedan tener el layout separado
find . -path "*/dashboard/analytics/layout*" -o -path "*/dashboard/settings/layout*"

# Buscar clases de color oscuro hardcodeadas en estas rutas
grep -r "bg-gray-9\|bg-zinc-9\|bg-slate-9\|bg-neutral-9" app/dashboard/analytics app/dashboard/settings
```

---

### BUG-02 — Contador "Visitas" bloqueado en 0 para todos los clientes

**Severidad:** Crítica  
**Página afectada:** `/dashboard/clients`  
**Síntoma:** La columna "VISITAS" muestra `0` para todos los clientes aunque existen citas con estado "Completada" en la agenda.

**Causa probable:**  
El campo `visits` (o equivalente) en el modelo de `Client`/`Patient` no se actualiza cuando una cita pasa a estado "Completada". El contador probablemente debería incrementarse en el handler que procesa la transición de estado de la cita, o calcularse dinámicamente como un `count` de citas completadas relacionadas.

**Propuesta de fix — opción A (campo calculado en query):**

```typescript
// En la query que obtiene la lista de clientes, hacer un join/count
// Ejemplo con Prisma:
const clients = await prisma.client.findMany({
  include: {
    _count: {
      select: {
        appointments: {
          where: { status: 'completed' },
        },
      },
    },
  },
});
// Mapear: client.visits = client._count.appointments
```

**Propuesta de fix — opción B (actualizar al completar cita):**

```typescript
// En el handler de actualización de estado de cita:
if (newStatus === 'completed' && oldStatus !== 'completed') {
  await prisma.client.update({
    where: { id: appointment.clientId },
    data: { visits: { increment: 1 } },
  });
}
```

**Recomendación:** Opción A es más robusta (no pierde sincronía). Opción B es más performante para listas grandes.

---

### BUG-03 — Tratamiento creado sin cliente ni profesional asignado

**Severidad:** Crítica  
**Página afectada:** `/dashboard/treatments`  
**Síntoma:** El único tratamiento registrado ("Inplante") tiene cliente `—` y profesional `—`. No hay validación que exija estos campos al crear un tratamiento.

**Propuesta de fix:**

1. En el formulario de creación de tratamiento, marcar `clientId` y `professionalId` como campos requeridos con validación frontend (zod/yup schema).
2. En la API route de creación, agregar validación server-side:

```typescript
// schema de validación (zod):
const createTreatmentSchema = z.object({
  name: z.string().min(1),
  clientId: z.string().uuid('Cliente requerido'), // <-- agregar
  professionalId: z.string().uuid('Profesional requerido'), // <-- agregar
  estimatedCost: z.number().optional(),
  totalSessions: z.number().min(1),
});
```

3. En la UI, mostrar el campo cliente y profesional como `required` con asterisco y mensaje de error inline.

---

### BUG-04 — Servicio "Limpieza Dental" sin precio en página pública de reservas

**Severidad:** Crítica  
**Página afectada:** `/book/diente-feliz-3` (página pública)  
**Síntoma:** Los servicios "Extracción" ($20 USD) y "Limpieza Bucal" ($10 USD) muestran su precio como badge, pero "Limpieza Dental" aparece sin precio.

**Causa probable:**  
El servicio "Limpieza Dental" tiene el campo `price` como `null`, `0`, o `undefined` en la base de datos, y el componente de la tarjeta de servicio solo renderiza el badge cuando `price > 0`.

**Propuesta de fix:**

1. En el admin de servicios (`/dashboard/services`), editar "Limpieza Dental" y asignar precio.
2. En el componente de la tarjeta de servicio de la página de booking, manejar el caso `price = 0` o `price = null` explícitamente:

```tsx
// Antes (probablemente):
{
  service.price && <Badge>${service.price} USD</Badge>;
}

// Después:
{
  service.price != null && service.price > 0 ? (
    <Badge>${service.price} USD</Badge>
  ) : (
    <Badge variant="outline">Precio a consultar</Badge>
  );
}
```

---

### BUG-05 — Base de conocimiento del chatbot con nombre de clínica incorrecto

**Severidad:** Crítica  
**Página afectada:** `/dashboard/chatbot` → tab "Conocimiento"  
**Síntoma:** El documento "Equipo" contiene el texto "Clínica Dental Sonrisa es una clínica familiar especializada en..." — nombre de otra clínica. El chatbot responde a los pacientes con este nombre incorrecto.

**Propuesta de fix (datos):**
Editar el documento "Equipo" en la base de conocimiento y reemplazar todas las menciones de "Clínica Dental Sonrisa" por "Diente Feliz Venezolano".

**Propuesta de fix (prevención técnica):**

```typescript
// Al crear documentos de conocimiento, hacer un lint/warning si el texto
// contiene un nombre de negocio diferente al registrado en settings:
function validateKnowledgeDoc(content: string, businessName: string) {
  // Lista de nombres comunes de clínicas de demostración a detectar
  const demoNames = ['Clínica Dental Sonrisa', 'Dental Example', 'Demo Clinic'];
  const hasDemoName = demoNames.some((name) => content.includes(name));
  if (hasDemoName) {
    return {
      warning:
        'El documento parece contener el nombre de una clínica de ejemplo. Verifica que el contenido sea el correcto.',
    };
  }
}
```

---

### BUG-06 — Dirección ficticia en base de conocimiento (Ubicación)

**Severidad:** Crítica  
**Página afectada:** `/dashboard/chatbot` → tab "Conocimiento" → documento "Ubicación"  
**Síntoma:** La dirección guardada es "Av. Ficticia 123, Santo Domingo, República Dominicana" — dirección de ejemplo, no real. El chatbot la comparte a pacientes que preguntan cómo llegar.

**Propuesta de fix (datos):**
Editar el documento "Ubicación" con la dirección real del negocio.

**Propuesta de fix (prevención técnica):**

```typescript
// Detectar dirección placeholder al guardar:
const PLACEHOLDER_KEYWORDS = ['ficticia', 'example', 'test', 'lorem', 'placeholder', '123 main'];

function isPlaceholderAddress(address: string): boolean {
  return PLACEHOLDER_KEYWORDS.some((kw) => address.toLowerCase().includes(kw));
}
// Mostrar warning al guardar si se detecta dirección placeholder
```

---

## BUGS MEDIOS (prioridad 2 — corregir en el próximo sprint)

---

### BUG-07 — Etiqueta de evento de automatización en inglés

**Severidad:** Media  
**Página afectada:** `/dashboard/automations`  
**Síntoma:** La regla "Cita reagendada" muestra el evento como `appointment.rescheduled` (raw event key) mientras las otras 3 reglas muestran labels en español legibles ("Cita cancelada", "Cita confirmada", "Cita creada").

**Causa probable:**  
El mapa de traducción de eventos no incluye la key `appointment.rescheduled`.

**Propuesta de fix:**

```typescript
// En el archivo de constantes/traducciones de eventos:
export const EVENT_LABELS: Record<string, string> = {
  'appointment.cancelled': 'Cita cancelada',
  'appointment.confirmed': 'Cita confirmada',
  'appointment.created': 'Cita creada',
  'appointment.rescheduled': 'Cita reagendada', // <-- agregar esta línea
  // agregar cualquier otro evento que falte
};

// En el componente de la lista de automatizaciones:
const label = EVENT_LABELS[automation.event] ?? automation.event;
```

---

### BUG-08 — Clientes duplicados "miguel" / "Miguel"

**Severidad:** Media  
**Página afectada:** `/dashboard/clients`, `/dashboard/pipeline`  
**Síntoma:** Existen dos registros con el mismo número de teléfono venezolano: "miguel" (4243148415) y "Miguel" (+584243148415). Son el mismo paciente.

**Propuesta de fix:**

1. Migración de datos: fusionar los dos registros, conservando el que tenga más datos completos.
2. Prevención: normalizar el número de teléfono antes de guardar (quitar espacios, agregar código de país si falta) y hacer upsert por número normalizado:

```typescript
function normalizePhone(phone: string, defaultCountryCode = '58'): string {
  // Quitar todo lo que no sea dígito
  const digits = phone.replace(/\D/g, '');
  // Si no empieza con código de país, agregar el default (Venezuela = 58)
  if (digits.length === 10 && !digits.startsWith(defaultCountryCode)) {
    return `+${defaultCountryCode}${digits}`;
  }
  return `+${digits}`;
}

// Al crear cliente, normalizar teléfono y verificar duplicado:
const normalizedPhone = normalizePhone(input.phone);
const existing = await prisma.client.findUnique({ where: { phone: normalizedPhone } });
if (existing) {
  // Retornar error o sugerir merge
}
```

3. En el formulario de creación de cliente, capitalizar automáticamente el nombre (`toProperCase`).

---

### BUG-09 — Datos de prueba en clientes activos

**Severidad:** Media  
**Registros afectados:** "tretry" (tel: 4356734435), "wefdsdfs" (tel: 324534565435)  
**Síntoma:** Clientes con nombres claramente de prueba aparecen en búsquedas, pipeline y analíticas, distorsionando las métricas.

**Propuesta de fix (datos):**
Eliminar o archivar los registros "tretry" y "wefdsdfs" desde `/dashboard/clients`.

**Propuesta de fix (prevención):**

```typescript
// Validación básica de nombre de cliente:
const clientNameSchema = z
  .string()
  .min(2, 'El nombre debe tener al menos 2 caracteres')
  .regex(/^[a-záéíóúüñA-ZÁÉÍÓÚÜÑ\s'-]+$/, 'El nombre solo puede contener letras');
// Esto rechazaría "tretry" si quisieras ser estricto,
// aunque es mejor solo requerir mínimo 2 chars y formato básico
```

---

### BUG-10 — Pipeline sin asignación automática al confirmar cita

**Severidad:** Media  
**Página afectada:** `/dashboard/pipeline`  
**Síntoma:** 6 de 8 clientes están en etapa "Sin asignar". El pipeline existe pero no tiene automatizaciones que muevan clientes entre etapas.

**Propuesta de fix:**
Crear automatización que, al confirmarse la primera cita de un cliente, lo mueva a "Nuevo contacto" o "Primera consulta agendada":

```typescript
// Nueva regla de automatización sugerida:
{
  name: 'Mover cliente al pipeline al confirmar cita',
  event: 'appointment.confirmed',
  conditions: [
    { field: 'client.pipelineStage', operator: 'equals', value: null } // solo si no tiene etapa
  ],
  action: {
    type: 'update_pipeline_stage',
    value: 'primera-consulta-agendada'
  }
}
```

También se puede hacer directamente en el handler de confirmación de cita:

```typescript
// En appointment confirmation handler:
if (appointment.client.pipelineStage === null) {
  await updateClientPipelineStage(appointment.clientId, 'primera-consulta-agendada');
}
```

---

### BUG-11 — Chatbot incompleto: faltan paso 4 y 5 de configuración

**Severidad:** Media  
**Página afectada:** `/dashboard/chatbot` → tab "Validación"  
**Síntoma:** El wizard de configuración tiene 5 pasos, solo 3 completados. Falta: política de cancelación y prueba del chat.

**Propuesta de fix:**

1. Agregar documento "Política de cancelación" a la base de conocimiento con el texto real de la política de la clínica.
2. Completar la prueba del chatbot desde el panel derecho de pruebas.
3. Considerar marcar estos pasos como opcionales si el chatbot ya está activo en producción, o mostrar un warning más visible en el dashboard.

---

### BUG-12 — Sin visibilidad de logs de fallo en automatizaciones

**Severidad:** Media  
**Página afectada:** `/dashboard/automations`  
**Síntoma:** El dashboard reporta 67% de tasa de éxito en automatizaciones pero no hay forma de ver cuál regla falla ni por qué desde la UI.

**Propuesta de fix:**

```typescript
// Agregar a cada regla de automatización un componente de estado:
interface AutomationRule {
  id: string;
  name: string;
  // ... campos existentes
  stats?: {
    // <-- agregar
    totalRuns: number;
    successCount: number;
    lastRunAt: Date;
    lastError?: string;
  };
}

// En la UI de la lista de automatizaciones, mostrar por cada regla:
// - Última ejecución: "hace 2 horas"
// - Tasa de éxito: "8/9 (89%)"
// - Botón "Ver logs" que abre un drawer con el historial
```

---

### BUG-13 — Sin confirmación modal al eliminar registros

**Severidad:** Media  
**Páginas afectadas:** `/dashboard/team`, `/dashboard/services`  
**Síntoma:** El botón "Eliminar" en Equipo y Servicios puede ejecutar la acción sin un paso de confirmación visible que prevenga eliminaciones accidentales.

**Propuesta de fix:**

```tsx
// Reemplazar el botón de eliminar directo por uno con confirmación:
<AlertDialog>
  <AlertDialogTrigger asChild>
    <Button variant="destructive" size="sm">
      <Trash2 className="h-4 w-4" />
      Eliminar
    </Button>
  </AlertDialogTrigger>
  <AlertDialogContent>
    <AlertDialogHeader>
      <AlertDialogTitle>¿Eliminar {item.name}?</AlertDialogTitle>
      <AlertDialogDescription>Esta acción no se puede deshacer.</AlertDialogDescription>
    </AlertDialogHeader>
    <AlertDialogFooter>
      <AlertDialogCancel>Cancelar</AlertDialogCancel>
      <AlertDialogAction onClick={() => handleDelete(item.id)}>Eliminar</AlertDialogAction>
    </AlertDialogFooter>
  </AlertDialogContent>
</AlertDialog>
```

---

### BUG-14 — Logo de clínica no configurado en página de reservas

**Severidad:** Media  
**Página afectada:** `/book/diente-feliz-3`  
**Síntoma:** La página pública de reservas muestra un ícono genérico "JS" (favicon/default) en lugar del logo real de la clínica.

**Propuesta de fix:**

1. Agregar campo `logoUrl` al modelo de negocio en Configuración → Negocio.
2. En la página de booking, usar `businessSettings.logoUrl` con fallback al avatar de iniciales:

```tsx
// En el componente de la página de booking:
{
  business.logoUrl ? (
    <Image src={business.logoUrl} alt={business.name} width={80} height={80} className="rounded-full object-cover" />
  ) : (
    <div className="w-20 h-20 rounded-full bg-primary flex items-center justify-center text-white text-2xl font-medium">
      {business.name.charAt(0)}
    </div>
  );
}
```

---

### BUG-15 — Miembro del equipo "Dr. Test" activo en producción

**Severidad:** Media  
**Página afectada:** `/dashboard/team`  
**Síntoma:** "Dr. Test" tiene estado Activo y aparece como opción en filtros de agenda y asignación de citas.

**Propuesta de fix (datos):**
Archivar o eliminar el registro "Dr. Test" desde `/dashboard/team`.

---

## BUGS BAJOS (prioridad 3 — pulido)

---

### BUG-16 — Typo: "Inplante" en lugar de "Implante"

**Severidad:** Baja  
**Página afectada:** `/dashboard/treatments`  
**Fix:** Editar el tratamiento y corregir nombre a "Implante" y slug a `implante`.

---

### BUG-17 — Acentos faltantes en textos del sistema

**Severidad:** Baja  
**Páginas afectadas:**

- `/dashboard/tasks` → empty state: "automatizacion" → "automatización"
- `/dashboard/automations` → subtítulo: "acciones automaticas" → "acciones automáticas"

**Fix:** Buscar y reemplazar en los archivos de copia/i18n correspondientes:

```bash
grep -r "automatizacion\b\|automaticas\b" --include="*.tsx" --include="*.ts" src/
```

---

### BUG-18 — Ejemplo de instrucciones del chatbot no corresponde al rubro

**Severidad:** Baja  
**Página afectada:** `/dashboard/chatbot` → tab "Personalidad" → campo "Instrucciones personalizadas"  
**Síntoma:** El placeholder dice `"Siempre ofrecer combo corte+barba"` — ejemplo de barbería.

**Fix:** Cambiar el placeholder a uno relevante para clínica dental:

```tsx
// En el componente del campo de instrucciones:
<Textarea placeholder='Ej: "Siempre mencionar que ofrecemos financiamiento en 3 cuotas sin interés para tratamientos de ortodoncia"' />
```

---

## MEJORAS DE UX PROPUESTAS (no son bugs, son oportunidades)

---

### UX-01 — Capitalización automática de nombres de clientes

**Problema:** Nombres en minúscula ("miguel") por falta de normalización al ingresar datos.  
**Propuesta:**

```typescript
// Utility function:
function toProperCase(name: string): string {
  return name.toLowerCase().replace(/(?:^|\s)\S/g, (char) => char.toUpperCase());
}

// Aplicar al guardar cliente y al mostrar en UI
```

---

### UX-02 — Indicador de notificaciones en el header

**Problema:** No hay campana de notificaciones visible. Las 4 citas "pendientes de aprobación" no tienen alerta visual dentro del sistema.  
**Propuesta:** Agregar un ícono de campana en el header del dashboard con un badge numérico que muestre las acciones pendientes:

- Citas pendientes de aprobación
- Automatizaciones con errores recientes
- Tareas vencidas

---

### UX-03 — Analíticas con rango de fechas personalizable

**Problema:** Las analíticas solo muestran los últimos 30 días sin opción de cambiar el rango.  
**Propuesta:** Agregar un selector de período: Última semana / Últimos 30 días / Últimos 3 meses / Personalizado. El dato de "ingresos estimados: $10" con solo 30 días no es representativo del negocio.

---

### UX-04 — Vista de historial de conversaciones WhatsApp

**Problema:** La sección WhatsApp solo muestra el estado de conexión. No hay visibilidad de las conversaciones activas o recientes con pacientes.  
**Propuesta:** Agregar un log de conversaciones recientes (últimas 24h) con link a cada hilo, visible desde el dashboard o desde la sección WhatsApp.

---

### UX-05 — Deduplicación de clientes al importar/crear

**Problema:** El sistema permite crear dos registros del mismo paciente con el mismo número de teléfono.  
**Propuesta:** Al crear un cliente, hacer lookup por teléfono normalizado y mostrar un aviso:

> "Ya existe un cliente con este número: Miguel (+584243148415). ¿Deseas actualizar ese registro en lugar de crear uno nuevo?"

---

### UX-06 — Precio y descripción de servicios en la agenda

**Problema:** Al ver una cita en la agenda, no se muestra el precio ni la descripción del servicio directamente.  
**Propuesta:** En el tooltip/detalle de la cita al hacer click, incluir: servicio + duración + precio + profesional + estado de pago.

---

## CHECKLIST DE ACCIONES PARA CLAUDE CODE

### Acciones inmediatas (datos — sin código):

- [ ] Editar documento "Equipo" en `/dashboard/chatbot`: reemplazar "Clínica Dental Sonrisa" por "Diente Feliz Venezolano"
- [ ] Editar documento "Ubicación": reemplazar "Av. Ficticia 123..." con dirección real
- [ ] Agregar precio a "Limpieza Dental" en `/dashboard/services`
- [ ] Eliminar clientes "tretry" y "wefdsdfs" en `/dashboard/clients`
- [ ] Fusionar clientes "miguel" / "Miguel" (mismo número +584243148415)
- [ ] Archivar "Dr. Test" en `/dashboard/team`
- [ ] Corregir nombre de tratamiento "Inplante" → "Implante"

### Fixes de código (ordenados por impacto):

1. **[BUG-01]** Unificar tema visual en `/dashboard/analytics` y `/dashboard/settings` — eliminar layouts separados o clases de color hardcodeadas
2. **[BUG-02]** Corregir contador de visitas en clientes — calcular dinámicamente desde citas completadas
3. **[BUG-03]** Agregar validación de `clientId` y `professionalId` requeridos al crear tratamiento
4. **[BUG-04]** Manejar `price = null/0` en componente de servicio de página de booking
5. **[BUG-07]** Agregar `'appointment.rescheduled': 'Cita reagendada'` al mapa de labels de eventos
6. **[BUG-08]** Normalizar teléfonos al guardar clientes y verificar duplicados
7. **[BUG-12]** Agregar stats de ejecución por regla en la vista de automatizaciones
8. **[BUG-13]** Agregar `AlertDialog` de confirmación antes de eliminar en Equipo y Servicios
9. **[BUG-14]** Agregar campo `logoUrl` en configuración de negocio y usarlo en página de booking
10. **[BUG-17]** Corregir acentos faltantes en textos de empty states
11. **[BUG-18]** Actualizar placeholder de instrucciones del chatbot a ejemplo dental

---

## ARCHIVOS CLAVE A INVESTIGAR

```
# Layout y tema
app/dashboard/layout.tsx
app/dashboard/analytics/layout.tsx  (verificar si existe)
app/dashboard/settings/layout.tsx   (verificar si existe)

# Clientes y visitas
app/dashboard/clients/page.tsx
app/api/clients/route.ts
lib/db/queries/clients.ts

# Tratamientos (validación)
app/dashboard/treatments/page.tsx
app/api/treatments/route.ts
lib/validations/treatment.ts

# Automatizaciones (labels)
lib/constants/events.ts  (o similar)
components/automations/AutomationList.tsx

# Página pública de reservas
app/book/[slug]/page.tsx
components/booking/ServiceCard.tsx

# Chatbot / base de conocimiento
app/dashboard/chatbot/page.tsx
components/chatbot/KnowledgeDoc.tsx
```

---

_Documento generado automáticamente a partir de auditoría visual completa del sistema CitaSpot._  
_Total: 18 bugs (6 críticos, 9 medios, 3 bajos) + 6 propuestas de mejora UX._
