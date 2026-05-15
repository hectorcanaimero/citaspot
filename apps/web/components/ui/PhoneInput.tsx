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
      for (const code of COUNTRY_CODES) {
        const cfg = PHONE_COUNTRIES[code];
        if (value.startsWith(cfg.prefix)) {
          const local = value.slice(cfg.prefix.length);
          setCountry(code);
          setLocalNumber(local);
          return;
        }
      }
      setLocalNumber(value);
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

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
      const digits = e.target.value.replace(/\D/g, '').slice(0, maxDigits);
      setLocalNumber(digits);
      onChange?.(normalizePhone(country, digits));
    }

    function handleCountrySelect(code: CountryCode) {
      setCountry(code);
      setDropdownOpen(false);
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
