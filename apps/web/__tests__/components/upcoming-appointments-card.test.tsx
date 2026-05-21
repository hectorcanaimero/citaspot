import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import React from 'react';

// ── Mocks ─────────────────────────────────────────────────────────────────────

const mockUpcoming = vi.fn();

vi.mock('@/lib/api', () => {
  // Re-exportamos APIError como una clase simple para no romper imports.
  class APIError extends Error {
    constructor(public status: number, message: string) {
      super(message);
    }
  }
  return {
    appointments: {
      upcoming: (limit: number) => mockUpcoming(limit),
    },
    APIError,
  };
});

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

// El store de tenant — devolvemos una zona horaria fija (UTC) para tests deterministas.
vi.mock('@/store/tenant', () => ({
  useTenantTimezone: () => 'UTC',
  useTenantStore: () => null,
}));

import { UpcomingAppointmentsCard } from '@/components/dashboard/UpcomingAppointmentsCard';

// ── Helpers ───────────────────────────────────────────────────────────────────

interface ApptOverrides {
  id?: string;
  customer_name?: string;
  professional_name?: string;
  service_name?: string;
  starts_at?: string;
  status?: 'pending' | 'confirmed';
}

function mkAppt(overrides: ApptOverrides = {}) {
  return {
    id: overrides.id ?? 'a-1',
    tenant_id: 't-1',
    customer_id: 'c-1',
    professional_id: 'p-1',
    service_id: 's-1',
    starts_at: overrides.starts_at ?? '2026-05-21T15:00:00Z',
    ends_at: '2026-05-21T15:30:00Z',
    status: overrides.status ?? 'confirmed',
    source: 'dashboard',
    customer_name: overrides.customer_name ?? 'Ana García',
    customer_phone: '+1',
    professional_name: overrides.professional_name ?? 'Dr. López',
    service_name: overrides.service_name ?? 'Limpieza dental',
    service_duration_min: 30,
    created_at: '',
    updated_at: '',
  };
}

beforeEach(() => {
  mockUpcoming.mockReset();
  // Hoy es 2026-05-21 (de currentDate). Fijamos el reloj para "hoy"/"mañana" deterministas.
  vi.setSystemTime(new Date('2026-05-21T10:00:00Z'));
});

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('UpcomingAppointmentsCard', () => {
  it('renderiza la lista en el orden recibido', async () => {
    mockUpcoming.mockResolvedValue({
      data: [
        mkAppt({ id: 'a-1', customer_name: 'Ana', starts_at: '2026-05-21T15:00:00Z' }),
        mkAppt({ id: 'a-2', customer_name: 'Beto', starts_at: '2026-05-22T15:00:00Z' }),
        mkAppt({ id: 'a-3', customer_name: 'Carla', starts_at: '2026-05-24T15:00:00Z' }),
      ],
    });

    render(<UpcomingAppointmentsCard />);

    await waitFor(() => expect(screen.getByText('Ana')).toBeInTheDocument());
    const items = screen.getAllByRole('listitem');
    expect(items).toHaveLength(3);
    expect(items[0].textContent).toContain('Ana');
    expect(items[1].textContent).toContain('Beto');
    expect(items[2].textContent).toContain('Carla');
  });

  it('muestra empty state con CTA a /dashboard/agenda', async () => {
    mockUpcoming.mockResolvedValue({ data: [] });

    render(<UpcomingAppointmentsCard />);

    await waitFor(() =>
      expect(screen.getByText('No tenés citas próximas.')).toBeInTheDocument(),
    );
    const cta = screen.getByText('Ir a la agenda').closest('a');
    expect(cta).toHaveAttribute('href', '/dashboard/agenda');
  });

  it('badge muestra "Hoy" para citas del día actual', async () => {
    mockUpcoming.mockResolvedValue({
      data: [mkAppt({ starts_at: '2026-05-21T18:00:00Z' })],
    });
    render(<UpcomingAppointmentsCard />);
    await waitFor(() => expect(screen.getByText('Hoy')).toBeInTheDocument());
  });

  it('badge muestra "Mañana" para citas del día siguiente', async () => {
    mockUpcoming.mockResolvedValue({
      data: [mkAppt({ starts_at: '2026-05-22T18:00:00Z' })],
    });
    render(<UpcomingAppointmentsCard />);
    await waitFor(() => expect(screen.getByText('Mañana')).toBeInTheDocument());
  });

  it('badge muestra fecha formateada para citas más allá de mañana', async () => {
    mockUpcoming.mockResolvedValue({
      data: [mkAppt({ starts_at: '2026-05-25T10:00:00Z' })],
    });
    render(<UpcomingAppointmentsCard />);
    // date-fns con locale es formatea "d MMM" → "25 may"
    await waitFor(() =>
      expect(screen.getByText(/25 may/i)).toBeInTheDocument(),
    );
  });

  it('refetcha cuando cambia refreshKey', async () => {
    mockUpcoming
      .mockResolvedValueOnce({ data: [mkAppt({ id: 'a-1', customer_name: 'Ana' })] })
      .mockResolvedValueOnce({ data: [mkAppt({ id: 'a-2', customer_name: 'Beto' })] });

    const { rerender } = render(<UpcomingAppointmentsCard refreshKey={1} />);
    await waitFor(() => expect(screen.getByText('Ana')).toBeInTheDocument());

    rerender(<UpcomingAppointmentsCard refreshKey={2} />);
    await waitFor(() => expect(screen.getByText('Beto')).toBeInTheDocument());
    expect(mockUpcoming).toHaveBeenCalledTimes(2);
  });
});
