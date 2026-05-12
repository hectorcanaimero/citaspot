import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import React from 'react';

// ── Mocks de Next.js ──────────────────────────────────────────────────────────

const mockPush    = vi.fn();
const mockRefresh = vi.fn();

vi.mock('next/navigation', () => ({
  usePathname: vi.fn(() => '/dashboard'),
  useRouter:   vi.fn(() => ({ push: mockPush, refresh: mockRefresh })),
}));

vi.mock('next/link', () => ({
  default: ({
    href,
    children,
    className,
  }: {
    href: string;
    children: React.ReactNode;
    className?: string;
  }) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
}));

// ── Mock de Supabase ──────────────────────────────────────────────────────────

const mockSignOut = vi.fn().mockResolvedValue({});

vi.mock('@/lib/supabase/browser', () => ({
  createClient: vi.fn(() => ({
    auth: { signOut: mockSignOut },
  })),
}));

// ── Importar componente después de mocks ──────────────────────────────────────

import { Sidebar } from '@/components/dashboard/sidebar';
import { usePathname } from 'next/navigation';

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('Sidebar', () => {
  beforeEach(() => {
    vi.mocked(usePathname).mockReturnValue('/dashboard');
    mockPush.mockClear();
    mockRefresh.mockClear();
    mockSignOut.mockClear();
  });

  // ── Renderizado básico ──────────────────────────────────────────────────────

  it('muestra el logo CitaSpot', () => {
    render(<Sidebar />);
    expect(screen.getByText('CitaSpot')).toBeInTheDocument();
  });

  it('muestra todos los ítems de navegación', () => {
    render(<Sidebar />);
    expect(screen.getByText('Dashboard')).toBeInTheDocument();
    expect(screen.getByText('Agenda')).toBeInTheDocument();
    expect(screen.getByText('Clientes')).toBeInTheDocument();
    expect(screen.getByText('Servicios')).toBeInTheDocument();
    expect(screen.getByText('Equipo')).toBeInTheDocument();
    expect(screen.getByText('Conocimiento')).toBeInTheDocument();
    expect(screen.getByText('WhatsApp')).toBeInTheDocument();
    expect(screen.getByText('Analíticas')).toBeInTheDocument();
    expect(screen.getByText('Configuración')).toBeInTheDocument();
  });

  it('muestra el botón de cerrar sesión', () => {
    render(<Sidebar />);
    expect(screen.getByText('Cerrar sesión')).toBeInTheDocument();
  });

  // ── Links de navegación ──────────────────────────────────────────────────────

  it('el link Dashboard apunta a /dashboard', () => {
    render(<Sidebar />);
    const link = screen.getByText('Dashboard').closest('a');
    expect(link).toHaveAttribute('href', '/dashboard');
  });

  it('el link Agenda apunta a /dashboard/agenda', () => {
    render(<Sidebar />);
    const link = screen.getByText('Agenda').closest('a');
    expect(link).toHaveAttribute('href', '/dashboard/agenda');
  });

  it('el link Clientes apunta a /dashboard/clients', () => {
    render(<Sidebar />);
    const link = screen.getByText('Clientes').closest('a');
    expect(link).toHaveAttribute('href', '/dashboard/clients');
  });

  // ── Estado activo ────────────────────────────────────────────────────────────

  it('Dashboard está activo en /dashboard (coincidencia exacta)', () => {
    vi.mocked(usePathname).mockReturnValue('/dashboard');
    render(<Sidebar />);
    const link = screen.getByText('Dashboard').closest('a');
    expect(link?.className).toContain('bg-sidebar-active');
    expect(link?.className).toContain('text-white');
  });

  it('Dashboard NO está activo en /dashboard/agenda', () => {
    vi.mocked(usePathname).mockReturnValue('/dashboard/agenda');
    render(<Sidebar />);
    const link = screen.getByText('Dashboard').closest('a');
    expect(link?.className).not.toMatch(/\bbg-sidebar-active\b(?!\/)/)
  });

  it('Agenda está activa en /dashboard/agenda', () => {
    vi.mocked(usePathname).mockReturnValue('/dashboard/agenda');
    render(<Sidebar />);
    const link = screen.getByText('Agenda').closest('a');
    expect(link?.className).toContain('bg-sidebar-active');
    expect(link?.className).toContain('text-white');
  });

  it('Clientes está activo en subrutas de /dashboard/clients', () => {
    vi.mocked(usePathname).mockReturnValue('/dashboard/clients/123');
    render(<Sidebar />);
    const link = screen.getByText('Clientes').closest('a');
    expect(link?.className).toContain('bg-sidebar-active');
  });

  it('ítems no activos tienen clase de texto sidebar', () => {
    vi.mocked(usePathname).mockReturnValue('/dashboard');
    render(<Sidebar />);
    const link = screen.getByText('Agenda').closest('a');
    expect(link?.className).toContain('text-sidebar-text');
  });

  // ── Logout ───────────────────────────────────────────────────────────────────

  it('al hacer click en Cerrar sesión llama a supabase.auth.signOut', async () => {
    render(<Sidebar />);
    const logoutBtn = screen.getByText('Cerrar sesión');
    fireEvent.click(logoutBtn);
    await waitFor(() => expect(mockSignOut).toHaveBeenCalledOnce());
  });

  it('al hacer click en Cerrar sesión redirige a /login', async () => {
    render(<Sidebar />);
    fireEvent.click(screen.getByText('Cerrar sesión'));
    await waitFor(() => expect(mockPush).toHaveBeenCalledWith('/login'));
  });

  // ── Estructura ──────────────────────────────────────────────────────────────

  it('la barra lateral tiene ancho fijo en escritorio', () => {
    const { container } = render(<Sidebar />);
    const aside = container.querySelector('aside');
    expect(aside?.className).toContain('w-[220px]');
  });

  it('la nav contiene los 14 ítems del dashboard', () => {
    render(<Sidebar />);
    const links = document.querySelectorAll('nav a');
    expect(links).toHaveLength(14);
  });
});
