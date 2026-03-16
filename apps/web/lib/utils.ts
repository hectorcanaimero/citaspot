import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';
import type { Translations } from '@/lib/i18n';
import type { APIError } from '@/lib/api';

// Combina clases Tailwind sin conflictos.
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

// Traduce un código de error de API al idioma activo.
// Usar cuando se muestra un APIError al usuario:
//   catch (err) { setError(translateApiError(err, t)); }
export function translateApiError(err: APIError | Error | unknown, t: Translations): string {
  if (err && typeof err === 'object' && 'code' in err) {
    const code = (err as APIError).code as string;
    const errors = t.apiErrors as Record<string, string>;
    if (errors[code]) return errors[code];
  }
  if (err instanceof Error) return err.message;
  return t.apiErrors.unknown;
}
