# Editable Profile Settings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Permitir que el propietario del negocio edite el nombre del negocio y su propio nombre desde la página de configuración del dashboard.

**Architecture:** Se agregan dos endpoints PATCH en el backend Go (`/api/v1/tenant/profile` y `/api/v1/me/profile`), sus métodos de repositorio correspondientes, y se convierten las páginas read-only de settings en formularios editables inline con React Hook Form. El email no se edita en esta versión porque está vinculado a Supabase Auth (requiere flujo de verificación separado) — se muestra read-only con nota de contacto a soporte.

**Tech Stack:** Go 1.25 + Fiber v2 · pgx/v5 · React Hook Form · Zod · Next.js 14 App Router · Tailwind CSS

---

## Archivos a crear o modificar

| Archivo | Acción | Qué cambia |
|---------|--------|-----------|
| `apps/api/internal/domain/types.go` | Modificar | Agregar `UpdateTenantProfileRequest`, `UpdateUserProfileRequest` |
| `apps/api/internal/domain/interfaces.go` | Modificar | Agregar `UpdateTenantProfile`, `UpdateUserProfile` a `AuthRepository` |
| `apps/api/internal/repository/auth.go` | Modificar | Implementar los dos métodos nuevos |
| `apps/api/internal/handler/settings.go` | Modificar | Agregar handlers `UpdateTenantProfile` y `UpdateMyProfile` |
| `apps/api/cmd/server/main.go` | Modificar | Registrar las dos rutas nuevas |
| `apps/web/lib/api.ts` | Modificar | Agregar `updateBusinessProfile` y `updateMyProfile` |
| `apps/web/lib/i18n/locales/es.ts` | Modificar | Strings para edición de perfil |
| `apps/web/lib/i18n/locales/en.ts` | Modificar | Strings para edición de perfil (inglés) |
| `apps/web/lib/i18n/locales/pt.ts` | Modificar | Strings para edición de perfil (portugués) |
| `apps/web/app/dashboard/settings/business/page.tsx` | Modificar | Formulario editable para nombre del negocio |
| `apps/web/app/dashboard/settings/account/page.tsx` | Modificar | Formulario editable para nombre del propietario |

---

## Task 1: Tipos de dominio (Go)

**Files:**
- Modify: `apps/api/internal/domain/types.go` — agregar después del bloque `RegisterRequest`

- [ ] **Step 1: Agregar los dos nuevos request types**

Agregar después de `RegisterResponse` (línea ~99) en `types.go`:

```go
// UpdateTenantProfileRequest actualiza el perfil público del negocio.
type UpdateTenantProfileRequest struct {
	Name     string `json:"name"    validate:"required,min=2,max=120"`
	Phone    string `json:"phone"   validate:"omitempty,max=30"`
	City     string `json:"city"    validate:"omitempty,max=100"`
	Country  string `json:"country" validate:"omitempty,len=2"`
	Timezone string `json:"timezone" validate:"omitempty,max=60"`
}

// UpdateUserProfileRequest actualiza el nombre del propietario.
// El email NO se puede cambiar aquí — está vinculado a Supabase Auth.
type UpdateUserProfileRequest struct {
	Name string `json:"name" validate:"required,min=2,max=120"`
}
```

- [ ] **Step 2: Verificar que el archivo compila**

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go build ./...
```
Esperado: sin errores.

---

## Task 2: Interfaces de repositorio (Go)

**Files:**
- Modify: `apps/api/internal/domain/interfaces.go` — agregar métodos a `AuthRepository`

- [ ] **Step 1: Agregar los métodos a la interfaz**

Dentro de `AuthRepository` (después de `UpdateTenantSettings`), agregar:

```go
// UpdateTenantProfile actualiza nombre, teléfono, ciudad, país y timezone del negocio.
UpdateTenantProfile(ctx context.Context, tenantID uuid.UUID, req *UpdateTenantProfileRequest) error
// UpdateUserProfile actualiza el nombre del usuario propietario.
UpdateUserProfile(ctx context.Context, userID, tenantID uuid.UUID, req *UpdateUserProfileRequest) error
```

- [ ] **Step 2: Verificar que compila**

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go build ./...
```
Esperado: error de compilación — `authRepository does not implement AuthRepository` (falta la implementación).
Esto confirma que la interfaz está correcta.

---

## Task 3: Implementación del repositorio (Go)

**Files:**
- Modify: `apps/api/internal/repository/auth.go` — agregar al final del archivo

- [ ] **Step 1: Agregar `UpdateTenantProfile`**

```go
// UpdateTenantProfile actualiza nombre, teléfono, ciudad, país y timezone del negocio.
func (r *authRepository) UpdateTenantProfile(ctx context.Context, tenantID uuid.UUID, req *domain.UpdateTenantProfileRequest) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE tenants
		    SET name     = $2,
		        phone    = $3,
		        city     = $4,
		        country  = CASE WHEN $5 = '' THEN country ELSE $5 END,
		        timezone = CASE WHEN $6 = '' THEN timezone ELSE $6 END,
		        updated_at = NOW()
		  WHERE id = $1`,
		tenantID, req.Name, req.Phone, req.City, req.Country, req.Timezone,
	)
	if err != nil {
		return fmt.Errorf("authRepository.UpdateTenantProfile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// UpdateUserProfile actualiza el nombre del propietario.
func (r *authRepository) UpdateUserProfile(ctx context.Context, userID, tenantID uuid.UUID, req *domain.UpdateUserProfileRequest) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE users SET name = $3, updated_at = NOW() WHERE id = $1 AND tenant_id = $2`,
		userID, tenantID, req.Name,
	)
	if err != nil {
		return fmt.Errorf("authRepository.UpdateUserProfile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
```

- [ ] **Step 2: Verificar que compila**

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go build ./...
```
Esperado: sin errores.

- [ ] **Step 3: Commit**

```bash
cd /Users/al3jandro/project/agendAI
git add apps/api/internal/domain/types.go \
        apps/api/internal/domain/interfaces.go \
        apps/api/internal/repository/auth.go
git commit -m "feat(api): add UpdateTenantProfile and UpdateUserProfile to domain and repository"
```

---

## Task 4: Handlers HTTP (Go)

**Files:**
- Modify: `apps/api/internal/handler/settings.go`

- [ ] **Step 1: Agregar el import de `middleware` si no está**

Verificar que los imports en `settings.go` incluyen:
```go
import (
    "net/http"

    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"

    "github.com/citaspot/api/internal/domain"
    "github.com/citaspot/api/internal/middleware"
)
```
Ya están presentes — no modificar.

- [ ] **Step 2: Agregar `UpdateTenantProfile` al final del archivo**

```go
// UpdateTenantProfile actualiza el perfil del negocio (nombre, teléfono, ciudad, país, timezone).
func (h *SettingsHandler) UpdateTenantProfile(c *fiber.Ctx) error {
	tenantID := middleware.TenantIDFromContext(c)
	if tenantID == uuid.Nil {
		return fiber.NewError(http.StatusForbidden, "tenant no identificado")
	}

	var req domain.UpdateTenantProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "formato de datos inválido")
	}

	if err := h.authRepo.UpdateTenantProfile(c.Context(), tenantID, &req); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}

// UpdateMyProfile actualiza el nombre del propietario autenticado.
func (h *SettingsHandler) UpdateMyProfile(c *fiber.Ctx) error {
	user := middleware.UserFromContext(c)
	if user == nil {
		return fiber.NewError(http.StatusForbidden, "usuario no identificado")
	}

	var req domain.UpdateUserProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "formato de datos inválido")
	}

	if err := h.authRepo.UpdateUserProfile(c.Context(), user.ID, user.TenantID, &req); err != nil {
		return handleServiceError(c, err)
	}
	return c.SendStatus(http.StatusNoContent)
}
```

- [ ] **Step 3: Verificar que compila**

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go build ./...
```
Esperado: sin errores.

---

## Task 5: Registrar rutas en el servidor (Go)

**Files:**
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: Buscar el bloque donde se registran las rutas de settings**

En `main.go`, buscar donde está:
```go
api.Get("/settings", settingsHandler.Get)
api.Patch("/settings", settingsHandler.Update)
```

- [ ] **Step 2: Agregar las dos nuevas rutas justo después**

```go
api.Patch("/tenant/profile", settingsHandler.UpdateTenantProfile)
api.Patch("/me/profile",     settingsHandler.UpdateMyProfile)
```

- [ ] **Step 3: Compilar y correr tests**

```bash
cd /Users/al3jandro/project/agendAI/apps/api && go build ./... && go test ./...
```
Esperado: sin errores.

- [ ] **Step 4: Commit**

```bash
cd /Users/al3jandro/project/agendAI
git add apps/api/internal/handler/settings.go \
        apps/api/cmd/server/main.go
git commit -m "feat(api): add PATCH /tenant/profile and PATCH /me/profile endpoints"
```

---

## Task 6: Cliente API frontend (TypeScript)

**Files:**
- Modify: `apps/web/lib/api.ts`

- [ ] **Step 1: Buscar donde están los métodos del namespace `auth`**

En `lib/api.ts`, localizar el objeto `export const auth = { ... }` o la sección de métodos de autenticación.

- [ ] **Step 2: Agregar los dos nuevos métodos al cliente**

Dentro del objeto o módulo `auth`, agregar:

```typescript
export interface UpdateBusinessProfileInput {
  name: string;
  phone?: string;
  city?: string;
  country?: string;
  timezone?: string;
}

export interface UpdateMyProfileInput {
  name: string;
}
```

Y los métodos en el cliente:

```typescript
updateBusinessProfile: (data: UpdateBusinessProfileInput) =>
  request<void>('/api/v1/tenant/profile', {
    method: 'PATCH',
    body: JSON.stringify(data),
  }),

updateMyProfile: (data: UpdateMyProfileInput) =>
  request<void>('/api/v1/me/profile', {
    method: 'PATCH',
    body: JSON.stringify(data),
  }),
```

- [ ] **Step 3: Verificar que el proyecto compila**

```bash
cd /Users/al3jandro/project/agendAI/apps/web && npx tsc --noEmit 2>&1 | head -30
```
Esperado: sin errores de tipo.

---

## Task 7: Strings de i18n

**Files:**
- Modify: `apps/web/lib/i18n/locales/es.ts`
- Modify: `apps/web/lib/i18n/locales/en.ts`
- Modify: `apps/web/lib/i18n/locales/pt.ts`

- [ ] **Step 1: Agregar strings en `es.ts`**

Dentro de `settings.business`, agregar:
```typescript
business: {
  title: 'Negocio',
  nameLabel: 'Nombre',
  typeLabel: 'Tipo de negocio',
  bookingUrlLabel: 'URL de reservas',
  viewBookingPage: 'Ver página de reservas',
  // NUEVO:
  editName: 'Editar nombre del negocio',
  namePlaceholder: 'Ej: Salón Bella Vista',
  saveSuccess: 'Cambios guardados',
  saveError: 'No se pudieron guardar los cambios',
},
```

Dentro de `settings.account`, agregar:
```typescript
account: {
  title: 'Mi cuenta',
  nameLabel: 'Nombre',
  emailLabel: 'Email',
  roleLabel: 'Rol',
  ownerRole: 'Propietario',
  languageLabel: 'Idioma',
  // NUEVO:
  editName: 'Editar nombre',
  namePlaceholder: 'Tu nombre completo',
  emailReadonlyNote: 'Para cambiar el email contactá soporte.',
  saveSuccess: 'Nombre actualizado',
  saveError: 'No se pudo actualizar el nombre',
},
```

- [ ] **Step 2: Agregar strings en `en.ts`**

```typescript
// en settings.business:
editName: 'Edit business name',
namePlaceholder: 'e.g. Bella Vista Salon',
saveSuccess: 'Changes saved',
saveError: 'Could not save changes',

// en settings.account:
editName: 'Edit name',
namePlaceholder: 'Your full name',
emailReadonlyNote: 'To change your email, contact support.',
saveSuccess: 'Name updated',
saveError: 'Could not update name',
```

- [ ] **Step 3: Agregar strings en `pt.ts`**

```typescript
// en settings.business:
editName: 'Editar nome do negócio',
namePlaceholder: 'Ex: Salão Bella Vista',
saveSuccess: 'Alterações salvas',
saveError: 'Não foi possível salvar as alterações',

// en settings.account:
editName: 'Editar nome',
namePlaceholder: 'Seu nome completo',
emailReadonlyNote: 'Para alterar o e-mail, entre em contato com o suporte.',
saveSuccess: 'Nome atualizado',
saveError: 'Não foi possível atualizar o nome',
```

- [ ] **Step 4: Verificar que TypeScript no detecta tipos faltantes**

```bash
cd /Users/al3jandro/project/agendAI/apps/web && npx tsc --noEmit 2>&1 | head -30
```
Esperado: sin errores.

- [ ] **Step 5: Commit**

```bash
cd /Users/al3jandro/project/agendAI
git add apps/web/lib/api.ts \
        apps/web/lib/i18n/locales/es.ts \
        apps/web/lib/i18n/locales/en.ts \
        apps/web/lib/i18n/locales/pt.ts
git commit -m "feat(web): add updateBusinessProfile and updateMyProfile API methods with i18n strings"
```

---

## Task 8: Página de configuración del negocio editable

**Files:**
- Modify: `apps/web/app/dashboard/settings/business/page.tsx`

- [ ] **Step 1: Reemplazar el contenido completo del archivo**

```tsx
'use client';

import { useState, useEffect } from 'react';
import { Building2, ExternalLink, Pencil, X, Check } from 'lucide-react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Spinner } from '@/components/ui/spinner';
import { auth, TenantDTO } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

function InfoRow({ label, value }: { label: string; value: string | undefined }) {
  return (
    <div className="flex items-center justify-between py-2 text-sm">
      <span className="text-neutral-500">{label}</span>
      <span className="font-medium text-neutral-900">{value ?? '\u2014'}</span>
    </div>
  );
}

export default function BusinessPage() {
  const t = useTranslations();
  const [tenant, setTenant] = useState<TenantDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [nameValue, setNameValue] = useState('');
  const [saving, setSaving] = useState(false);
  const [feedback, setFeedback] = useState<{ ok: boolean; msg: string } | null>(null);
  const businessTypes = t.settings.businessTypes as Record<string, string>;

  useEffect(() => {
    auth.me()
      .then(({ tenant: tn }) => {
        setTenant(tn);
        setNameValue(tn.name ?? '');
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  function startEdit() {
    setNameValue(tenant?.name ?? '');
    setFeedback(null);
    setEditing(true);
  }

  function cancelEdit() {
    setEditing(false);
    setFeedback(null);
  }

  async function handleSave() {
    if (!nameValue.trim() || nameValue.trim().length < 2) return;
    setSaving(true);
    setFeedback(null);
    try {
      await auth.updateBusinessProfile({ name: nameValue.trim() });
      setTenant(prev => prev ? { ...prev, name: nameValue.trim() } : prev);
      setEditing(false);
      setFeedback({ ok: true, msg: t.settings.business.saveSuccess });
    } catch {
      setFeedback({ ok: false, msg: t.settings.business.saveError });
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <div className="flex justify-center py-16"><Spinner size="lg" /></div>;

  return (
    <Card>
      <CardHeader>
        <Building2 className="h-4 w-4 text-neutral-400" />
        <CardTitle>{t.settings.business.title}</CardTitle>
      </CardHeader>
      <div className="divide-y divide-neutral-100">
        {/* Nombre del negocio — editable */}
        <div className="flex items-center justify-between py-2 text-sm">
          <span className="text-neutral-500">{t.settings.business.nameLabel}</span>
          {editing ? (
            <div className="flex items-center gap-2">
              <Input
                value={nameValue}
                onChange={e => setNameValue(e.target.value)}
                placeholder={t.settings.business.namePlaceholder}
                className="h-7 text-sm w-48"
                autoFocus
                onKeyDown={e => {
                  if (e.key === 'Enter') handleSave();
                  if (e.key === 'Escape') cancelEdit();
                }}
              />
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7"
                onClick={handleSave}
                disabled={saving || nameValue.trim().length < 2}
                aria-label="Guardar"
              >
                {saving ? <Spinner size="sm" /> : <Check className="h-3.5 w-3.5 text-green-600" />}
              </Button>
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7"
                onClick={cancelEdit}
                disabled={saving}
                aria-label="Cancelar"
              >
                <X className="h-3.5 w-3.5 text-neutral-400" />
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <span className="font-medium text-neutral-900">{tenant?.name ?? '\u2014'}</span>
              <Button
                size="icon"
                variant="ghost"
                className="h-6 w-6"
                onClick={startEdit}
                aria-label={t.settings.business.editName}
              >
                <Pencil className="h-3 w-3 text-neutral-400" />
              </Button>
            </div>
          )}
        </div>
        <InfoRow label={t.settings.business.typeLabel} value={businessTypes[tenant?.business_type ?? ''] ?? tenant?.business_type} />
        <InfoRow label={t.settings.business.bookingUrlLabel} value={`citaspot.com/book/${tenant?.slug ?? ''}`} />
      </div>

      {feedback && (
        <p className={`mt-2 text-xs ${feedback.ok ? 'text-green-600' : 'text-red-500'}`}>
          {feedback.msg}
        </p>
      )}

      {tenant?.slug && (
        <div className="mt-3 pt-3 border-t border-neutral-100">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => window.open(`/book/${tenant.slug}`, '_blank')}
          >
            <ExternalLink className="mr-1.5 h-3.5 w-3.5" />
            {t.settings.business.viewBookingPage}
          </Button>
        </div>
      )}
    </Card>
  );
}
```

- [ ] **Step 2: Verificar tipos**

```bash
cd /Users/al3jandro/project/agendAI/apps/web && npx tsc --noEmit 2>&1 | head -30
```
Esperado: sin errores.

---

## Task 9: Página de cuenta editable

**Files:**
- Modify: `apps/web/app/dashboard/settings/account/page.tsx`

- [ ] **Step 1: Reemplazar el contenido completo del archivo**

```tsx
'use client';

import { useState, useEffect } from 'react';
import { User, Globe, Pencil, X, Check } from 'lucide-react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Spinner } from '@/components/ui/spinner';
import { auth, UserDTO } from '@/lib/api';
import { useTranslations, useLanguage } from '@/lib/i18n';

function InfoRow({ label, value }: { label: string; value: string | undefined }) {
  return (
    <div className="flex items-center justify-between py-2 text-sm">
      <span className="text-neutral-500">{label}</span>
      <span className="font-medium text-neutral-900">{value ?? '\u2014'}</span>
    </div>
  );
}

export default function AccountPage() {
  const t = useTranslations();
  const { language, setLanguage } = useLanguage();
  const [user, setUser] = useState<UserDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState(false);
  const [nameValue, setNameValue] = useState('');
  const [saving, setSaving] = useState(false);
  const [feedback, setFeedback] = useState<{ ok: boolean; msg: string } | null>(null);

  useEffect(() => {
    auth.me()
      .then(({ user: u }) => {
        setUser(u);
        setNameValue(u.name ?? '');
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  function startEdit() {
    setNameValue(user?.name ?? '');
    setFeedback(null);
    setEditing(true);
  }

  function cancelEdit() {
    setEditing(false);
    setFeedback(null);
  }

  async function handleSave() {
    if (!nameValue.trim() || nameValue.trim().length < 2) return;
    setSaving(true);
    setFeedback(null);
    try {
      await auth.updateMyProfile({ name: nameValue.trim() });
      setUser(prev => prev ? { ...prev, name: nameValue.trim() } : prev);
      setEditing(false);
      setFeedback({ ok: true, msg: t.settings.account.saveSuccess });
    } catch {
      setFeedback({ ok: false, msg: t.settings.account.saveError });
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <div className="flex justify-center py-16"><Spinner size="lg" /></div>;

  return (
    <Card>
      <CardHeader>
        <User className="h-4 w-4 text-neutral-400" />
        <CardTitle>{t.settings.account.title}</CardTitle>
      </CardHeader>
      <div className="divide-y divide-neutral-100">
        {/* Nombre — editable */}
        <div className="flex items-center justify-between py-2 text-sm">
          <span className="text-neutral-500">{t.settings.account.nameLabel}</span>
          {editing ? (
            <div className="flex items-center gap-2">
              <Input
                value={nameValue}
                onChange={e => setNameValue(e.target.value)}
                placeholder={t.settings.account.namePlaceholder}
                className="h-7 text-sm w-48"
                autoFocus
                onKeyDown={e => {
                  if (e.key === 'Enter') handleSave();
                  if (e.key === 'Escape') cancelEdit();
                }}
              />
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7"
                onClick={handleSave}
                disabled={saving || nameValue.trim().length < 2}
                aria-label="Guardar"
              >
                {saving ? <Spinner size="sm" /> : <Check className="h-3.5 w-3.5 text-green-600" />}
              </Button>
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7"
                onClick={cancelEdit}
                disabled={saving}
                aria-label="Cancelar"
              >
                <X className="h-3.5 w-3.5 text-neutral-400" />
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <span className="font-medium text-neutral-900">{user?.name ?? '\u2014'}</span>
              <Button
                size="icon"
                variant="ghost"
                className="h-6 w-6"
                onClick={startEdit}
                aria-label={t.settings.account.editName}
              >
                <Pencil className="h-3 w-3 text-neutral-400" />
              </Button>
            </div>
          )}
        </div>

        {/* Email — read-only con nota */}
        <div className="py-2 text-sm">
          <div className="flex items-center justify-between">
            <span className="text-neutral-500">{t.settings.account.emailLabel}</span>
            <span className="font-medium text-neutral-900">{user?.email ?? '\u2014'}</span>
          </div>
          <p className="mt-0.5 text-xs text-neutral-400">{t.settings.account.emailReadonlyNote}</p>
        </div>

        <InfoRow
          label={t.settings.account.roleLabel}
          value={user?.role === 'owner' ? t.settings.account.ownerRole : user?.role}
        />
      </div>

      {feedback && (
        <p className={`mt-2 text-xs ${feedback.ok ? 'text-green-600' : 'text-red-500'}`}>
          {feedback.msg}
        </p>
      )}

      {/* Selector de idioma */}
      <div className="mt-4 pt-4 border-t border-neutral-100">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Globe className="h-4 w-4 text-neutral-400" />
            <span className="text-sm text-neutral-600">{t.settings.account.languageLabel}</span>
          </div>
          <div className="flex rounded-lg border border-neutral-200 overflow-hidden text-sm">
            {(['es', 'en', 'pt'] as const).map((lang) => (
              <button
                key={lang}
                onClick={() => setLanguage(lang)}
                className={`px-3 py-1.5 transition-colors ${
                  language === lang
                    ? 'bg-primary-600 text-white font-medium'
                    : 'text-neutral-600 hover:bg-neutral-50'
                }`}
              >
                {t.settings.language[lang]}
              </button>
            ))}
          </div>
        </div>
      </div>
    </Card>
  );
}
```

- [ ] **Step 2: Verificar tipos**

```bash
cd /Users/al3jandro/project/agendAI/apps/web && npx tsc --noEmit 2>&1 | head -30
```
Esperado: sin errores.

- [ ] **Step 3: Commit final**

```bash
cd /Users/al3jandro/project/agendAI
git add apps/web/app/dashboard/settings/business/page.tsx \
        apps/web/app/dashboard/settings/account/page.tsx
git commit -m "feat(web): make business name and owner name editable in settings"
```
