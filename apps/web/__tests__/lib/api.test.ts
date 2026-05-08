import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// ── Mocks ─────────────────────────────────────────────────────────────────────

// Mock de Supabase browser client
vi.mock('@/lib/supabase/browser', () => ({
  createClient: vi.fn(() => ({
    auth: {
      getSession: vi.fn().mockResolvedValue({
        data: { session: { access_token: 'test-jwt-token' } },
      }),
      signOut: vi.fn().mockResolvedValue({}),
    },
  })),
}));

// Mock de window.location para evitar errores en jsdom al hacer redirect
const originalLocation = window.location;
beforeEach(() => {
  Object.defineProperty(window, 'location', {
    configurable: true,
    value: { href: '', pathname: '/dashboard' },
  });
});
afterEach(() => {
  Object.defineProperty(window, 'location', {
    configurable: true,
    value: originalLocation,
  });
  vi.restoreAllMocks();
});

// ── Importar API después de los mocks ─────────────────────────────────────────

import { appointments, professionals, services, APIError } from '@/lib/api';

// ── Helpers ───────────────────────────────────────────────────────────────────

function mockFetch(status: number, body: unknown) {
  return vi.spyOn(globalThis, 'fetch').mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  } as Response);
}

// ── appointments.list ─────────────────────────────────────────────────────────

describe('appointments.list', () => {
  it('llama al endpoint correcto con la fecha', async () => {
    const spy = mockFetch(200, { data: [] });
    await appointments.list('2026-03-07');
    expect(spy).toHaveBeenCalledOnce();
    const url = spy.mock.calls[0][0] as string;
    expect(url).toContain('/api/v1/appointments');
    expect(url).toContain('date=2026-03-07');
  });

  it('incluye el token de autorización en la cabecera', async () => {
    const spy = mockFetch(200, { data: [] });
    await appointments.list('2026-03-07');
    const init = spy.mock.calls[0][1] as RequestInit;
    expect((init.headers as Record<string, string>)['Authorization']).toBe(
      'Bearer test-jwt-token'
    );
  });

  it('retorna el array de citas en data', async () => {
    const mockAppts = [
      { id: '1', customer_name: 'Ana García', status: 'confirmed' },
    ];
    mockFetch(200, { data: mockAppts });
    const result = await appointments.list('2026-03-07');
    expect(result.data).toHaveLength(1);
    expect(result.data[0].customer_name).toBe('Ana García');
  });

  it('retorna data vacía cuando no hay citas', async () => {
    mockFetch(200, { data: [] });
    const result = await appointments.list('2026-03-07');
    expect(result.data).toEqual([]);
  });

  it('lanza APIError con status 500 en error del servidor', async () => {
    mockFetch(500, { error: 'Internal Server Error' });
    await expect(appointments.list('2026-03-07')).rejects.toThrow(APIError);
  });

  it('lanza APIError con status 404', async () => {
    mockFetch(404, { error: 'No encontrado' });
    await expect(appointments.list('2026-03-07')).rejects.toMatchObject({
      status: 404,
    });
  });

  it('lanza APIError "session_expired" en error 401', async () => {
    mockFetch(401, { error: 'Unauthorized' });
    await expect(appointments.list('2026-03-07')).rejects.toMatchObject({
      status: 401,
      code: 'session_expired',
    });
  });
});

// ── appointments.availability ─────────────────────────────────────────────────

describe('appointments.availability', () => {
  it('incluye professional_id, service_id y date en la URL', async () => {
    const spy = mockFetch(200, { data: [] });
    await appointments.availability('prof-1', 'svc-1', '2026-03-07');
    const url = spy.mock.calls[0][0] as string;
    expect(url).toContain('professional_id=prof-1');
    expect(url).toContain('service_id=svc-1');
    expect(url).toContain('date=2026-03-07');
  });

  it('retorna array de slots disponibles', async () => {
    const slots = [
      { starts_at: '2026-03-07T09:00:00', ends_at: '2026-03-07T09:30:00' },
      { starts_at: '2026-03-07T10:00:00', ends_at: '2026-03-07T10:30:00' },
    ];
    mockFetch(200, { data: slots });
    const result = await appointments.availability('prof-1', 'svc-1', '2026-03-07');
    expect(result.data).toHaveLength(2);
    expect(result.data[0].starts_at).toBe('2026-03-07T09:00:00');
  });
});

// ── professionals.list ────────────────────────────────────────────────────────

describe('professionals.list', () => {
  it('llama a /api/v1/professionals', async () => {
    const spy = mockFetch(200, { data: [] });
    await professionals.list();
    const url = spy.mock.calls[0][0] as string;
    expect(url).toContain('/api/v1/professionals');
  });

  it('retorna la lista de profesionales', async () => {
    const profs = [
      { id: '1', name: 'Dr. García', color: '#7c3aed', is_active: true },
    ];
    mockFetch(200, { data: profs });
    const result = await professionals.list();
    expect(result.data[0].name).toBe('Dr. García');
  });
});

// ── services.list ─────────────────────────────────────────────────────────────

describe('services.list', () => {
  it('llama a /api/v1/services', async () => {
    const spy = mockFetch(200, { data: [] });
    await services.list();
    const url = spy.mock.calls[0][0] as string;
    expect(url).toContain('/api/v1/services');
  });

  it('retorna la lista de servicios', async () => {
    const svcs = [
      { id: '1', name: 'Corte de cabello', duration_min: 30, currency: 'USD', is_active: true },
    ];
    mockFetch(200, { data: svcs });
    const result = await services.list();
    expect(result.data[0].name).toBe('Corte de cabello');
  });
});

// ── APIError ──────────────────────────────────────────────────────────────────

describe('APIError', () => {
  it('es instancia de Error', () => {
    const err = new APIError(422, 'Datos inválidos');
    expect(err).toBeInstanceOf(Error);
    expect(err).toBeInstanceOf(APIError);
  });

  it('expone status y message correctamente', () => {
    const err = new APIError(403, 'Acceso denegado');
    expect(err.status).toBe(403);
    expect(err.message).toBe('Acceso denegado');
    expect(err.name).toBe('APIError');
  });
});
