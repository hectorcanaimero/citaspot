# Phone Mask with Country Code — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a phone input component with country code badge and validation to the booking form, auto-detecting the tenant's country.

**Architecture:** Frontend-only component change plus one backend field exposure. Create a PhoneInput component with country badge prefix, integrate it into the booking contact step, and expose tenant.country in the public API response.

**Tech Stack:** Next.js 14, React 18, TypeScript, Tailwind CSS, Go (Fiber)

---

## Pre-analysis: What already exists

The backend **already exposes** `country` in the public profile response:
- `apps/api/internal/domain/types.go:361` — `PublicProfile.Country` field: `Country string \`json:"country,omitempty"\``
- `apps/api/internal/service/public.go:65` — `Country: tenant.Country` is set in `GetProfile()`
- `apps/web/lib/api.ts:881` — `PublicProfile.country?: string` already typed in the frontend

The booking page at `apps/web/app/book/[slug]/page.tsx` already loads the profile with `publicApi.getProfile(slug)` and has access to `profile.country`.

**Therefore Tasks 1 and 2 from the spec are already complete.** The plan starts with the PhoneInput component.

---

## Task 1: Create PhoneInput component

> **File:** `apps/web/components/ui/PhoneInput.tsx` (NEW)

- [ ] **Step 1.1** — Create the phone country config and PhoneInput component

```typescript
// apps/web/components/ui/PhoneInput.tsx
'use client';

import { useState, useRef, useEffect, forwardRef } from 'react';
import { cn } from '@/lib/utils';
import { ChevronDown } from 'lucide-react';

// ── Country Config ────────────────────────────────────────────────────────────

export type CountryCode = 'VE' | 'BR' | 'DO';

interface CountryConfig {
  prefix: string;
  flag: string;
  name: string;
  digits: number[];
}

export const PHONE_COUNTRIES: Record<CountryCode, CountryConfig> = {
  VE: { prefix: '58', flag: '\u{1F1FB}\u{1F1EA}', name: 'Venezuela', digits: [10] },
  BR: { prefix: '55', flag: '\u{1F1E7}\u{1F1F7}', name: 'Brasil', digits: [10, 11] },
  DO: { prefix: '1',  flag: '\u{1F1E9}\u{1F1F4}', name: 'Rep. Dominicana', digits: [10] },
};

const COUNTRY_CODES = Object.keys(PHONE_COUNTRIES) as CountryCode[];

// ── Validation ────────────────────────────────────────────────────────────────

export function validatePhone(country: CountryCode, localNumber: string): boolean {
  const config = PHONE_COUNTRIES[country];
  if (!config) return false;
  const digitsOnly = localNumber.replace(/\D/g, '');
  return config.digits.includes(digitsOnly.length);
}

export function normalizePhone(country: CountryCode, localNumber: string): string {
  const config = PHONE_COUNTRIES[country];
  const digitsOnly = localNumber.replace(/\D/g, '');
  return `${config.prefix}${digitsOnly}`;
}

// ── Component ─────────────────────────────────────────────────────────────────

interface PhoneInputProps {
  /** ISO country code from tenant (e.g. "VE", "DO", "BR") */
  defaultCountry?: string;
  /** Full phone value: prefix + local (e.g. "584243222332") */
  value?: string;
  /** Called with full normalized phone: prefix + localDigits */
  onChange?: (fullPhone: string) => void;
  /** Field label */
  label?: string;
  /** Error message */
  error?: string;
  /** Hint text below the input */
  hint?: string;
  /** Required field */
  required?: boolean;
  /** HTML id override */
  id?: string;
}

export const PhoneInput = forwardRef<HTMLInputElement, PhoneInputProps>(
  ({ defaultCountry, value, onChange, label, error, hint, required, id }, ref) => {
    // Resolve initial country — fallback to DO if unknown
    const initialCountry: CountryCode =
      defaultCountry && defaultCountry in PHONE_COUNTRIES
        ? (defaultCountry as CountryCode)
        : 'DO';

    const [country, setCountry] = useState<CountryCode>(initialCountry);
    const [localNumber, setLocalNumber] = useState('');
    const [dropdownOpen, setDropdownOpen] = useState(false);
    const dropdownRef = useRef<HTMLDivElement>(null);

    const config = PHONE_COUNTRIES[country];
    const maxDigits = Math.max(...config.digits);
    const inputId = id ?? label?.toLowerCase().replace(/\s+/g, '-') ?? 'phone-input';

    // Parse incoming value prop into country + local on mount
    useEffect(() => {
      if (!value) return;
      // Try to detect country from prefix
      for (const code of COUNTRY_CODES) {
        const cfg = PHONE_COUNTRIES[code];
        if (value.startsWith(cfg.prefix)) {
          const local = value.slice(cfg.prefix.length);
          setCountry(code);
          setLocalNumber(local);
          return;
        }
      }
      // If no prefix matches, treat as raw local number
      setLocalNumber(value);
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []); // Only on mount

    // Close dropdown on outside click
    useEffect(() => {
      function handleClickOutside(e: MouseEvent) {
        if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
          setDropdownOpen(false);
        }
      }
      if (dropdownOpen) {
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
      }
    }, [dropdownOpen]);

    function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
      // Strip non-digits
      const digits = e.target.value.replace(/\D/g, '').slice(0, maxDigits);
      setLocalNumber(digits);
      onChange?.(normalizePhone(country, digits));
    }

    function handleCountrySelect(code: CountryCode) {
      setCountry(code);
      setDropdownOpen(false);
      // Re-emit with new prefix
      onChange?.(normalizePhone(code, localNumber));
    }

    return (
      <div className="flex flex-col gap-1.5">
        {label && (
          <label htmlFor={inputId} className="text-sm font-medium text-neutral-700">
            {label}
          </label>
        )}

        <div className="relative flex">
          {/* Country badge button */}
          <div ref={dropdownRef} className="relative">
            <button
              type="button"
              onClick={() => setDropdownOpen((o) => !o)}
              className={cn(
                'flex h-9 items-center gap-1 rounded-l-lg border border-r-0 bg-neutral-50 px-2.5 text-sm',
                'transition-colors hover:bg-neutral-100',
                error ? 'border-error' : 'border-neutral-200',
              )}
              aria-label="Select country code"
              aria-expanded={dropdownOpen}
            >
              <span className="text-base leading-none">{config.flag}</span>
              <span className="text-neutral-600">+{config.prefix}</span>
              <ChevronDown className="h-3 w-3 text-neutral-400" />
            </button>

            {/* Mini dropdown */}
            {dropdownOpen && (
              <div className="absolute left-0 top-full z-50 mt-1 w-52 rounded-lg border border-neutral-200 bg-white py-1 shadow-lg">
                {COUNTRY_CODES.map((code) => {
                  const cfg = PHONE_COUNTRIES[code];
                  const selected = code === country;
                  return (
                    <button
                      key={code}
                      type="button"
                      onClick={() => handleCountrySelect(code)}
                      className={cn(
                        'flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm transition-colors',
                        selected
                          ? 'bg-primary-50 text-primary-700'
                          : 'text-neutral-700 hover:bg-neutral-50',
                      )}
                    >
                      <span className="text-base leading-none">{cfg.flag}</span>
                      <span className="flex-1">{cfg.name}</span>
                      <span className="text-neutral-400">+{cfg.prefix}</span>
                    </button>
                  );
                })}
              </div>
            )}
          </div>

          {/* Number input */}
          <input
            ref={ref}
            id={inputId}
            type="tel"
            inputMode="numeric"
            pattern="[0-9]*"
            maxLength={maxDigits}
            value={localNumber}
            onChange={handleInputChange}
            required={required}
            placeholder={'0'.repeat(maxDigits)}
            className={cn(
              'h-9 w-full rounded-r-lg border bg-white px-3 text-sm text-neutral-900',
              'placeholder:text-neutral-400',
              'transition-colors duration-150',
              'focus:outline-none focus:ring-2 focus:ring-primary-600 focus:ring-offset-0 focus:border-primary-600',
              error
                ? 'border-error focus:ring-red-500'
                : 'border-neutral-200 hover:border-neutral-300',
            )}
            aria-invalid={!!error}
            aria-describedby={error ? `${inputId}-error` : hint ? `${inputId}-hint` : undefined}
          />
        </div>

        {error && (
          <p id={`${inputId}-error`} className="text-xs text-error">
            {error}
          </p>
        )}
        {!error && hint && (
          <p id={`${inputId}-hint`} className="text-xs text-neutral-500">
            {hint}
          </p>
        )}
      </div>
    );
  },
);
PhoneInput.displayName = 'PhoneInput';
```

- [ ] **Step 1.2** — Commit

```bash
git add apps/web/components/ui/PhoneInput.tsx
git commit -m "feat(booking): add PhoneInput component with country badge and validation"
```

---

## Task 2: Add phone-related translations

> **Files:** `apps/web/lib/i18n/locales/es.ts`, `en.ts`, `pt.ts`

- [ ] **Step 2.1** — Add phone translation keys to `es.ts`

In `apps/web/lib/i18n/locales/es.ts`, inside the `booking` object (after `phonePlaceholder` at line ~619), replace the existing `phonePlaceholder` value and add new keys:

```typescript
// Find and replace this line:
    phonePlaceholder: '+1 809 000 0000',
// With:
    phonePlaceholder: '4243222332',
    phoneInvalid: 'El numero de telefono no tiene la cantidad correcta de digitos',
    phoneCountryLabel: 'Codigo de pais',
```

- [ ] **Step 2.2** — Add phone translation keys to `en.ts`

In `apps/web/lib/i18n/locales/en.ts`, inside the `booking` object, replace the existing `phonePlaceholder` value and add new keys:

```typescript
// Find and replace this line:
    phonePlaceholder: '+1 555 000 0000',
// With:
    phonePlaceholder: '4243222332',
    phoneInvalid: 'Phone number does not have the correct number of digits',
    phoneCountryLabel: 'Country code',
```

- [ ] **Step 2.3** — Add phone translation keys to `pt.ts`

In `apps/web/lib/i18n/locales/pt.ts`, inside the `booking` object, replace the existing `phonePlaceholder` value and add new keys:

```typescript
// Find and replace this line:
    phonePlaceholder: '+55 11 9 0000 0000',
// With:
    phonePlaceholder: '41999999999',
    phoneInvalid: 'O numero de telefone nao tem a quantidade correta de digitos',
    phoneCountryLabel: 'Codigo do pais',
```

- [ ] **Step 2.4** — Commit

```bash
git add apps/web/lib/i18n/locales/es.ts apps/web/lib/i18n/locales/en.ts apps/web/lib/i18n/locales/pt.ts
git commit -m "feat(i18n): add phone input translation keys for booking form"
```

---

## Task 3: Integrate PhoneInput into booking form

> **File:** `apps/web/app/book/[slug]/page.tsx`

- [ ] **Step 3.1** — Add PhoneInput import

At the top of the file (line ~1), add to the imports:

```typescript
// After the existing Input import on line 8:
import { PhoneInput, validatePhone, CountryCode, PHONE_COUNTRIES } from '@/components/ui/PhoneInput';
```

- [ ] **Step 3.2** — Add phoneError state and phoneCountry state

After the existing state declarations (around line 35, after `const [phone, setPhone] = useState('');`), add:

```typescript
  const [phoneCountry, setPhoneCountry] = useState<CountryCode>('DO');
  const [phoneError, setPhoneError] = useState('');
```

- [ ] **Step 3.3** — Initialize phoneCountry from profile

After the profile is loaded (inside or after the existing `useEffect` that loads the profile around line 43–48), set the default country. Add a new useEffect:

```typescript
  // Sincronizar el pais del tenant con el componente de telefono
  useEffect(() => {
    if (profile?.country && profile.country in PHONE_COUNTRIES) {
      setPhoneCountry(profile.country as CountryCode);
    }
  }, [profile]);
```

- [ ] **Step 3.4** — Replace the phone `<Input>` with `<PhoneInput>` in the contact step

In the contact step (step === 'contact', around line 347–365), find and replace:

```typescript
// FIND this (line ~352-353):
              <Input label={t.booking.phoneLabel} type="tel" placeholder={t.booking.phonePlaceholder}
                value={phone} onChange={(e) => setPhone(e.target.value)} required />

// REPLACE with:
              <PhoneInput
                label={t.booking.phoneLabel}
                defaultCountry={profile?.country}
                value={phone}
                onChange={(fullPhone) => { setPhone(fullPhone); setPhoneError(''); }}
                error={phoneError}
                required
              />
```

- [ ] **Step 3.5** — Add phone validation before advancing to confirm step

Replace the "Review appointment" button's `disabled` and `onClick` to validate the phone. Find the button around line 360:

```typescript
// FIND this:
                <Button size="md" disabled={!name || !phone} onClick={() => setStep('confirm')} className="flex-1">
                  {t.booking.reviewAppointment}
                </Button>

// REPLACE with:
                <Button size="md" disabled={!name || !phone} onClick={() => {
                  // Detect current country from the phone prefix
                  let detectedCountry: CountryCode = phoneCountry;
                  for (const code of (Object.keys(PHONE_COUNTRIES) as CountryCode[])) {
                    if (phone.startsWith(PHONE_COUNTRIES[code].prefix)) {
                      detectedCountry = code;
                      break;
                    }
                  }
                  const localDigits = phone.slice(PHONE_COUNTRIES[detectedCountry].prefix.length);
                  if (!validatePhone(detectedCountry, localDigits)) {
                    setPhoneError(t.booking.phoneInvalid);
                    return;
                  }
                  setPhoneError('');
                  setStep('confirm');
                }} className="flex-1">
                  {t.booking.reviewAppointment}
                </Button>
```

- [ ] **Step 3.6** — Format the phone display in the confirmation step

In the confirm step (step === 'confirm'), the phone is displayed at line ~389-391. No change needed here since `phone` already holds the full normalized number like `584243222332` — this is what gets displayed and sent to the API.

Optionally, for nicer display in the confirmation, replace:

```typescript
// FIND (line ~390):
                  <span>{phone}</span>

// REPLACE with:
                  <span>+{phone}</span>
```

This adds a `+` prefix for display only (e.g. `+584243222332`).

- [ ] **Step 3.7** — Commit

```bash
git add apps/web/app/book/[slug]/page.tsx
git commit -m "feat(booking): integrate PhoneInput with country badge into contact step"
```

---

## Task 4: Verify end-to-end

- [ ] **Step 4.1** — Run TypeScript type check

```bash
cd apps/web && npx tsc --noEmit
```

- [ ] **Step 4.2** — Run linter

```bash
cd apps/web && npx next lint
```

- [ ] **Step 4.3** — Manual verification checklist

1. Navigate to `/book/{slug}` for a tenant with `country: "VE"` — badge should show flag VE + 58
2. Navigate for a tenant with `country: "DO"` — badge should show flag DO + 1
3. Navigate for a tenant with no country — should default to DO
4. Click the badge dropdown — 3 countries visible (VE, BR, DO)
5. Select a different country — prefix updates, maxLength updates
6. Type letters in the input — only digits accepted
7. Type too few digits and click "Review" — error message shown
8. Type correct digit count — advances to confirm step
9. Confirm step shows `+{prefix}{digits}` formatted phone
10. Submit booking — `customer_phone` sent as `584243222332` (no `+`)

---

## Summary of files changed

| File | Action | What |
|------|--------|------|
| `apps/web/components/ui/PhoneInput.tsx` | **CREATE** | PhoneInput component with country badge, dropdown, validation |
| `apps/web/lib/i18n/locales/es.ts` | MODIFY | Add `phoneInvalid`, `phoneCountryLabel`, update `phonePlaceholder` |
| `apps/web/lib/i18n/locales/en.ts` | MODIFY | Add `phoneInvalid`, `phoneCountryLabel`, update `phonePlaceholder` |
| `apps/web/lib/i18n/locales/pt.ts` | MODIFY | Add `phoneInvalid`, `phoneCountryLabel`, update `phonePlaceholder` |
| `apps/web/app/book/[slug]/page.tsx` | MODIFY | Import PhoneInput, add phone validation, replace `<Input type="tel">` |

**Backend:** No changes needed — `country` is already exposed in `PublicProfile` and the frontend already types it.
