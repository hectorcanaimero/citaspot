// Middleware de Next.js — protección de rutas con Supabase Auth.
// Usa @supabase/ssr para verificar la sesión y manejar refresh automático.
import { createServerClient } from '@supabase/ssr';
import { type NextRequest, NextResponse } from 'next/server';

const PUBLIC_PATHS = ['/login', '/register', '/book'];

function isPublic(pathname: string): boolean {
  return PUBLIC_PATHS.some((p) => pathname.startsWith(p));
}

export async function middleware(request: NextRequest) {
  // Rutas públicas: siempre accesibles sin verificar sesión
  if (isPublic(request.nextUrl.pathname)) {
    return NextResponse.next();
  }

  let supabaseResponse = NextResponse.next({ request });

  // Crear cliente Supabase SSR — lee/escribe cookies en el response
  const supabase = createServerClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
    {
      cookies: {
        getAll() {
          return request.cookies.getAll();
        },
        setAll(cookiesToSet: { name: string; value: string; options?: Record<string, unknown> }[]) {
          // Propagar las cookies actualizadas (refresh de sesión) al response
          cookiesToSet.forEach(({ name, value }) =>
            request.cookies.set(name, value),
          );
          supabaseResponse = NextResponse.next({ request });
          cookiesToSet.forEach(({ name, value, options }) =>
            supabaseResponse.cookies.set(name, value, options as Parameters<typeof supabaseResponse.cookies.set>[2]),
          );
        },
      },
    },
  );

  // getUser() verifica el token con Supabase y renueva si es necesario
  const {
    data: { user },
    error: authError,
  } = await supabase.auth.getUser();

  console.log('[middleware] path:', request.nextUrl.pathname, '| user:', user?.id ?? 'null', '| error:', authError?.message ?? 'none');
  console.log('[middleware] supabaseUrl:', process.env.NEXT_PUBLIC_SUPABASE_URL?.substring(0, 40));
  console.log('[middleware] cookies:', request.cookies.getAll().map(c => c.name).join(', ') || 'ninguna');

  if (!user) {
    const loginUrl = request.nextUrl.clone();
    loginUrl.pathname = '/login';
    loginUrl.searchParams.set('from', request.nextUrl.pathname);
    return NextResponse.redirect(loginUrl);
  }

  return supabaseResponse;
}

export const config = {
  matcher: ['/dashboard/:path*', '/onboarding/:path*', '/settings/:path*'],
};
