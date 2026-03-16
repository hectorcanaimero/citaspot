// Cliente Supabase para uso en el navegador (Client Components).
// Usar SOLO en archivos con 'use client' o en funciones que corren en el browser.
import { createBrowserClient } from '@supabase/ssr';

export function createClient() {
  return createBrowserClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
  );
}
