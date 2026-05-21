import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';

// ── Mock de EventSource ───────────────────────────────────────────────────────
// jsdom no incluye EventSource. Implementamos un mock controlable con
// `addEventListener`, `close`, `onopen`, `onerror` y un helper `dispatch` para
// emitir eventos nombrados desde el test.

class MockEventSource {
  static instances: MockEventSource[] = [];
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSED = 2;

  url: string;
  readyState = MockEventSource.CONNECTING;
  onopen: ((ev: Event) => void) | null = null;
  onerror: ((ev: Event) => void) | null = null;
  onmessage: ((ev: MessageEvent) => void) | null = null;
  private listeners = new Map<string, Set<(ev: Event) => void>>();

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  addEventListener(type: string, handler: (ev: Event) => void) {
    if (!this.listeners.has(type)) this.listeners.set(type, new Set());
    this.listeners.get(type)!.add(handler);
  }

  removeEventListener(type: string, handler: (ev: Event) => void) {
    this.listeners.get(type)?.delete(handler);
  }

  close() {
    this.readyState = MockEventSource.CLOSED;
  }

  // Helpers que usa el test (no son parte de la API real).
  __open() {
    this.readyState = MockEventSource.OPEN;
    this.onopen?.(new Event('open'));
  }

  __error() {
    this.onerror?.(new Event('error'));
  }

  __dispatch(type: string, payload: unknown) {
    const ev = new MessageEvent(type, { data: JSON.stringify(payload) });
    this.listeners.get(type)?.forEach((h) => h(ev));
  }

  static reset() {
    MockEventSource.instances = [];
  }

  static last(): MockEventSource | undefined {
    return MockEventSource.instances[MockEventSource.instances.length - 1];
  }
}

// Inyectar antes de importar el hook
(globalThis as unknown as { EventSource: typeof MockEventSource }).EventSource =
  MockEventSource;

// ── Mocks de dependencias ─────────────────────────────────────────────────────

const mockRefreshSession = vi.fn();

vi.mock('@/lib/supabase/browser', () => ({
  createClient: vi.fn(() => ({
    auth: { refreshSession: mockRefreshSession },
  })),
}));

vi.mock('@/lib/api', () => ({
  getSupabaseAccessToken: vi.fn(async () => 'test-token'),
  realtimeAppointmentsUrl: (token: string) =>
    `https://api.example.com/api/v1/realtime/appointments?token=${token}`,
}));

// ── Importar el hook después de los mocks ─────────────────────────────────────

import { useAppointmentEvents } from '@/hooks/useAppointmentEvents';
import { getSupabaseAccessToken } from '@/lib/api';

// ── Setup / teardown ──────────────────────────────────────────────────────────

beforeEach(() => {
  MockEventSource.reset();
  vi.mocked(getSupabaseAccessToken).mockResolvedValue('test-token');
  mockRefreshSession.mockReset();
});

afterEach(() => {
  vi.useRealTimers();
});

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('useAppointmentEvents', () => {
  it('abre EventSource en mount con la URL correcta', async () => {
    const onEvent = vi.fn();
    renderHook(() => useAppointmentEvents(onEvent));

    await waitFor(() => {
      expect(MockEventSource.instances.length).toBe(1);
    });
    const es = MockEventSource.last()!;
    expect(es.url).toContain('/api/v1/realtime/appointments');
    expect(es.url).toContain('token=test-token');
  });

  it('marca connected=true al recibir onopen', async () => {
    const onEvent = vi.fn();
    const { result } = renderHook(() => useAppointmentEvents(onEvent));

    await waitFor(() => expect(MockEventSource.last()).toBeTruthy());
    act(() => {
      MockEventSource.last()!.__open();
    });
    await waitFor(() => expect(result.current.connected).toBe(true));
  });

  it('invoca onEvent con el envelope parseado en appointment.created', async () => {
    const onEvent = vi.fn();
    renderHook(() => useAppointmentEvents(onEvent));

    await waitFor(() => expect(MockEventSource.last()).toBeTruthy());

    const payload = {
      event: 'appointment.created',
      data: { id: 'appt-1', customer_name: 'Ana', status: 'pending' },
      ts: '2026-05-21T10:00:00Z',
    };
    act(() => {
      MockEventSource.last()!.__open();
      MockEventSource.last()!.__dispatch('appointment.created', payload);
    });

    await waitFor(() => expect(onEvent).toHaveBeenCalledTimes(1));
    const env = onEvent.mock.calls[0][0];
    expect(env.event).toBe('appointment.created');
    expect(env.data.id).toBe('appt-1');
    expect(env.ts).toBe('2026-05-21T10:00:00Z');
  });

  it('cierra el EventSource en unmount', async () => {
    const onEvent = vi.fn();
    const { unmount } = renderHook(() => useAppointmentEvents(onEvent));

    await waitFor(() => expect(MockEventSource.last()).toBeTruthy());
    const es = MockEventSource.last()!;
    expect(es.readyState).not.toBe(MockEventSource.CLOSED);

    unmount();
    expect(es.readyState).toBe(MockEventSource.CLOSED);
  });

  it('no abre EventSource si no hay token', async () => {
    vi.mocked(getSupabaseAccessToken).mockResolvedValueOnce(null);
    const onEvent = vi.fn();
    const { result } = renderHook(() => useAppointmentEvents(onEvent));

    // dejamos correr el microtask del async open()
    await new Promise((r) => setTimeout(r, 10));
    expect(MockEventSource.instances.length).toBe(0);
    expect(result.current.connected).toBe(false);
  });

  it('reintenta tras onerror con backoff (fake timers)', async () => {
    vi.useFakeTimers();
    const onEvent = vi.fn();
    renderHook(() => useAppointmentEvents(onEvent));

    // Esperar a que se cree el primer EventSource
    await vi.waitFor(() => {
      expect(MockEventSource.instances.length).toBe(1);
    });

    act(() => {
      MockEventSource.last()!.__error();
    });
    expect(MockEventSource.last()!.readyState).toBe(MockEventSource.CLOSED);

    // El backoff inicial es ~1s con jitter ±20% → avanzar 2s cubre cualquier valor
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2_000);
    });

    await vi.waitFor(() => {
      expect(MockEventSource.instances.length).toBe(2);
    });
  });

  it('intenta refreshSession tras 3 errores rápidos (heurística 401)', async () => {
    vi.useFakeTimers();
    mockRefreshSession.mockResolvedValue({
      data: { session: { access_token: 'fresh-token' } },
      error: null,
    });

    const onEvent = vi.fn();
    renderHook(() => useAppointmentEvents(onEvent));

    await vi.waitFor(() => expect(MockEventSource.instances.length).toBe(1));

    // 3 errores consecutivos dentro de la ventana de 2s
    for (let i = 0; i < 3; i++) {
      act(() => {
        MockEventSource.last()!.__error();
      });
      // Avanzar lo suficiente para que la reconexión cree un nuevo EventSource
      await act(async () => {
        await vi.advanceTimersByTimeAsync(50);
      });
    }

    await vi.waitFor(() => {
      expect(mockRefreshSession).toHaveBeenCalled();
    });
  });
});
