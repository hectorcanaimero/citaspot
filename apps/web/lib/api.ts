// Cliente HTTP centralizado para el Core API.
// NUNCA hacer fetch() directo en componentes — siempre a través de este módulo.
// El token lo gestiona Supabase SDK (@supabase/ssr) — no usamos localStorage manualmente.

import { createClient } from '@/lib/supabase/browser';

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:3001';

export class APIError extends Error {
  /** Código legible por máquina devuelto por el backend (e.g. "not_found").
   *  Usar con t.apiErrors[err.code] para obtener el mensaje traducido. */
  public code: string;

  constructor(
    public status: number,
    message: string,
    code?: string
  ) {
    super(message);
    this.name = 'APIError';
    this.code = code ?? message;
  }
}

// Obtiene el access_token desde la sesión de Supabase.
// El SDK renueva automáticamente si el token está por expirar.
async function getToken(): Promise<string | null> {
  if (typeof window === 'undefined') return null;
  const supabase = createClient();
  const { data: { session } } = await supabase.auth.getSession();
  return session?.access_token ?? null;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = await getToken();
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init?.headers,
    },
  });

  if (res.status === 401) {
    // Sesión inválida — cerrar sesión y redirigir al login
    if (typeof window !== 'undefined') {
      const supabase = createClient();
      await supabase.auth.signOut();
      window.location.href = '/login';
    }
    throw new APIError(401, 'Sesión expirada. Por favor inicia sesión de nuevo.', 'session_expired');
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ code: 'unknown', message: 'Error desconocido' }));
    // El backend devuelve { code, message, error } — usar code para i18n, message como fallback
    const code = body.code as string | undefined;
    const message = (body.message ?? body.error ?? 'Error del servidor') as string;
    throw new APIError(res.status, message, code);
  }

  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

// ── Tipos ──────────────────────────────────────────────────────────────────────

export interface UserDTO {
  id: string;
  tenant_id: string;
  email: string;
  name: string;
  role: string;
}

export interface TenantDTO {
  id: string;
  slug: string;
  name: string;
  business_type: string;
  plan: string;
  plan_status: string;
  trial_ends_at?: string;
}

export interface Appointment {
  id: string;
  tenant_id: string;
  customer_id: string;
  professional_id: string;
  service_id: string;
  starts_at: string;
  ends_at: string;
  status: 'pending' | 'confirmed' | 'cancelled' | 'completed' | 'no_show';
  source: string;
  price?: number;
  notes?: string;
  internal_notes?: string;
  customer_name: string;
  customer_phone: string;
  professional_name: string;
  service_name: string;
  service_duration_min: number;
  created_at: string;
  updated_at: string;
}

export interface Professional {
  id: string;
  name: string;
  specialty?: string;
  color: string;
  is_active: boolean;
}

export interface Service {
  id: string;
  name: string;
  description?: string;
  duration_min: number;
  price?: number;
  currency: string;
  is_active: boolean;
}

export interface TimeSlot {
  starts_at: string;
  ends_at: string;
}

// ── Auth ───────────────────────────────────────────────────────────────────────

export const auth = {
  // register: el Go API crea el tenant en nuestra DB + el usuario en Supabase Auth.
  // Luego inicializamos la sesión de Supabase con los tokens retornados.
  async register(data: {
    email: string;
    password: string;
    name: string;
    business_name: string;
    business_type: string;
    city?: string;
    country?: string;
    timezone?: string;
  }) {
    // 1. Llamar al Go API (crea tenant + user en DB + usuario en Supabase)
    const res = await fetch(`${API_URL}/api/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: 'Error al registrarse' }));
      throw new APIError(res.status, body.error ?? 'Error al registrarse');
    }
    const result = await res.json() as {
      token: string;
      refresh_token: string;
      user: UserDTO;
      tenant: TenantDTO;
    };

    // 2. Inicializar la sesión de Supabase con los tokens del Go API
    if (typeof window !== 'undefined') {
      const supabase = createClient();
      await supabase.auth.setSession({
        access_token:  result.token,
        refresh_token: result.refresh_token,
      });
    }
    return result;
  },

  async me() {
    return request<{ user: UserDTO; tenant: TenantDTO }>('/api/v1/me');
  },
};

// ── Appointments ───────────────────────────────────────────────────────────────

export const appointments = {
  async list(date: string, timezone?: string): Promise<{ data: Appointment[] }> {
    const tz = timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone;
    return request(`/api/v1/appointments?date=${date}&timezone=${encodeURIComponent(tz)}`);
  },

  async updateStatus(
    id: string,
    update: { status?: string; internal_notes?: string; cancellation_reason?: string }
  ) {
    return request(`/api/v1/appointments/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(update),
    });
  },

  async cancel(id: string, reason?: string) {
    return request(`/api/v1/appointments/${id}/cancel`, {
      method: 'DELETE',
      body: JSON.stringify({ reason }),
    });
  },

  async availability(professionalId: string, serviceId: string, date: string): Promise<{ data: TimeSlot[] }> {
    const tz = Intl.DateTimeFormat().resolvedOptions().timeZone;
    return request(
      `/api/v1/appointments/availability?professional_id=${professionalId}&service_id=${serviceId}&date=${date}&timezone=${encodeURIComponent(tz)}`
    );
  },
};

// ── Professionals ──────────────────────────────────────────────────────────────

export interface Schedule {
  day_of_week: number; // 0=Dom, 1=Lun, ..., 6=Sáb
  start_time:  string; // "HH:MM"
  end_time:    string; // "HH:MM"
  is_active:   boolean;
}

export interface ProfessionalInput {
  name:       string;
  specialty?: string;
  bio?:       string;
  color?:     string;
  is_active?: boolean;
}

export const professionals = {
  async list(): Promise<{ data: Professional[] }> {
    return request('/api/v1/professionals');
  },

  async create(data: ProfessionalInput): Promise<Professional> {
    return request('/api/v1/professionals', {
      method: 'POST',
      body:   JSON.stringify(data),
    });
  },

  async update(id: string, data: Partial<ProfessionalInput>): Promise<Professional> {
    return request(`/api/v1/professionals/${id}`, {
      method: 'PATCH',
      body:   JSON.stringify(data),
    });
  },

  async getSchedule(id: string): Promise<{ data: Schedule[] }> {
    return request(`/api/v1/professionals/${id}/schedule`);
  },

  async setSchedule(id: string, schedules: Omit<Schedule, never>[]): Promise<{ data: Schedule[] }> {
    return request(`/api/v1/professionals/${id}/schedule`, {
      method: 'PUT',
      body:   JSON.stringify(schedules),
    });
  },
};

// ── Services ───────────────────────────────────────────────────────────────────

export const services = {
  async list(): Promise<{ data: Service[] }> {
    return request('/api/v1/services');
  },
  async create(data: Omit<ServiceInput, 'id'>) {
    return request<Service>('/api/v1/services', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },
  async update(id: string, data: Partial<ServiceInput>) {
    return request<Service>(`/api/v1/services/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },
};

export interface ServiceInput {
  name: string;
  description?: string;
  duration_min: number;
  price?: number;
  currency?: string;
  buffer_min?: number;
  sort_order?: number;
  is_active?: boolean;
}

// ── Customers ──────────────────────────────────────────────────────────────────

export interface Customer {
  id: string;
  tenant_id: string;
  name: string;
  phone: string;
  email?: string;
  notes?: string;
  tags: string[];
  wa_opt_in: boolean;
  total_visits: number;
  created_at: string;
}

export const customers = {
  async list(search = '', limit = 50, offset = 0): Promise<{ data: Customer[] }> {
    const q = new URLSearchParams({ search, limit: String(limit), offset: String(offset) });
    return request(`/api/v1/customers?${q}`);
  },
};

// ── Knowledge Base ─────────────────────────────────────────────────────────────

export interface KnowledgeDocument {
  id: string;
  category: string;
  title: string;
  content: string;
  is_active: boolean;
  source_type: 'text' | 'file';
  file_name?: string;
  status: 'ready' | 'processing' | 'error';
  created_at: string;
  updated_at: string;
}

export const knowledge = {
  async list(): Promise<{ data: KnowledgeDocument[] }> {
    return request('/api/v1/knowledge');
  },
  async create(data: { category: string; title: string; content: string; is_active?: boolean }) {
    return request<KnowledgeDocument>('/api/v1/knowledge', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },
  async update(id: string, data: { category: string; title: string; content: string; is_active?: boolean }) {
    return request<KnowledgeDocument>(`/api/v1/knowledge/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },
  async upload(file: File, title: string, category: string, isActive = true): Promise<KnowledgeDocument> {
    const token = await getToken();
    const form = new FormData();
    form.append('file', file);
    form.append('title', title);
    form.append('category', category);
    form.append('is_active', String(isActive));

    const res = await fetch(`${API_URL}/api/v1/knowledge/upload`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: form,
    });

    if (res.status === 401) {
      if (typeof window !== 'undefined') {
        const supabase = createClient();
        await supabase.auth.signOut();
        window.location.href = '/login';
      }
      throw new APIError(401, 'Sesión expirada. Por favor inicia sesión de nuevo.');
    }
    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: 'Error al subir el archivo' }));
      throw new APIError(res.status, body.error ?? 'Error al subir el archivo');
    }
    return res.json();
  },
  async remove(id: string) {
    return request(`/api/v1/knowledge/${id}`, { method: 'DELETE' });
  },
};

// ── Public (sin auth) ──────────────────────────────────────────────────────────

export interface PublicProfile {
  slug: string;
  name: string;
  business_type: string;
  city?: string;
  country?: string;
  services: Service[];
  professionals: Professional[];
}

export const publicApi = {
  async getProfile(slug: string): Promise<PublicProfile> {
    return fetch(`${API_URL}/api/v1/public/${slug}`).then(async (r) => {
      if (!r.ok) throw new APIError(r.status, 'Negocio no encontrado');
      return r.json();
    });
  },

  async getAvailability(
    slug: string,
    professionalId: string,
    serviceId: string,
    date: string,
    timezone?: string,
  ): Promise<{ data: TimeSlot[] }> {
    const tz = timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone;
    const url = `${API_URL}/api/v1/public/${slug}/availability?professional_id=${professionalId}&service_id=${serviceId}&date=${date}&timezone=${encodeURIComponent(tz)}`;
    return fetch(url).then(async (r) => {
      if (!r.ok) throw new APIError(r.status, 'Error consultando disponibilidad');
      return r.json();
    });
  },

  async book(
    slug: string,
    data: {
      professional_id: string;
      service_id: string;
      starts_at: string;
      customer_name: string;
      customer_phone: string;
      customer_email?: string;
      notes?: string;
    },
  ) {
    return fetch(`${API_URL}/api/v1/public/${slug}/book`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    }).then(async (r) => {
      if (!r.ok) {
        const body = await r.json().catch(() => ({ error: 'Error al reservar' }));
        throw new APIError(r.status, body.error ?? 'Error al reservar');
      }
      return r.json();
    });
  },
};

// ── WhatsApp ───────────────────────────────────────────────────────────────────

export const whatsapp = {
  async getStatus(): Promise<{ status: 'CONNECTED' | 'DISCONNECTED' | 'CONNECTING'; instance: string }> {
    return request('/api/v1/whatsapp/status');
  },

  // Inicia la conexión. El QR llega vía webhook en segundos — usar getQR() para obtenerlo.
  async connect(): Promise<{ status: string; instance: string }> {
    return request('/api/v1/whatsapp/connect', { method: 'POST' });
  },

  // Retorna el QR base64 si ya fue recibido por el backend vía webhook.
  // Retorna null si el QR aún no está disponible (204 No Content).
  async getQR(): Promise<{ qr: string } | null> {
    const token = await (async () => {
      if (typeof window === 'undefined') return null;
      const { createClient } = await import('@/lib/supabase/browser');
      const supabase = createClient();
      const { data: { session } } = await supabase.auth.getSession();
      return session?.access_token ?? null;
    })();
    const API_URL = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:3001';
    const res = await fetch(`${API_URL}/api/v1/whatsapp/qr`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });
    if (res.status === 204) return null; // QR no disponible aún
    if (!res.ok) return null;
    return res.json();
  },

  // Cierra la sesión de WhatsApp del tenant.
  async disconnect(): Promise<void> {
    return request('/api/v1/whatsapp/disconnect', { method: 'DELETE' });
  },
};

// ── Billing ────────────────────────────────────────────────────────────────────

export interface StripeSubscription {
  id: string;
  status: string;
  current_period_end: string;
  cancel_at_period_end: boolean;
}

export interface StripeInvoice {
  id: string;
  number: string;
  amount: number;     // centavos
  currency: string;
  status: string;
  created_at: string;
  pdf_url: string;
  hosted_url: string;
}

export const billing = {
  async checkout(plan: 'starter' | 'professional', successUrl: string, cancelUrl: string): Promise<{ url: string }> {
    return request('/api/v1/billing/checkout', {
      method: 'POST',
      body:   JSON.stringify({ plan, success_url: successUrl, cancel_url: cancelUrl }),
    });
  },

  async subscription(): Promise<{ subscription: StripeSubscription | null }> {
    return request('/api/v1/billing/subscription');
  },

  async invoices(): Promise<{ data: StripeInvoice[] }> {
    return request('/api/v1/billing/invoices');
  },

  async cancel(): Promise<{ cancel_at_period_end: boolean; current_period_end: string }> {
    return request('/api/v1/billing/cancel', { method: 'POST' });
  },
};

// Compatibilidad con el export anterior
export const api = { health: () => request<{ status: string }>('/health') };
