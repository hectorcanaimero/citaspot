// Cliente HTTP centralizado para el Core API.
// NUNCA hacer fetch() directo en componentes — siempre a través de este módulo.
// El token lo gestiona Supabase SDK (@supabase/ssr) — no usamos localStorage manualmente.

import { createClient } from '@/lib/supabase/browser';
console.log('NEXT_PUBLIC_API_URL', process.env.NEXT_PUBLIC_API_URL);
const API_URL = process.env.NEXT_PUBLIC_API_URL ?? 'https://api.citaspot.com';

export class APIError extends Error {
  /** Código legible por máquina devuelto por el backend (e.g. "not_found").
   *  Usar con t.apiErrors[err.code] para obtener el mensaje traducido. */
  public code: string;

  constructor(
    public status: number,
    message: string,
    code?: string,
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
  // getSession() puede devolver null si el SDK aún no rehidrató desde cookies.
  // En ese caso, getUser() fuerza la rehidratación completa.
  const { data: { session } } = await supabase.auth.getSession();
  if (session?.access_token) return session.access_token;
  // Fallback: forzar rehidratación con getUser() que valida contra Supabase
  const { data: { user } } = await supabase.auth.getUser();
  if (!user) return null;
  // Después de getUser(), la sesión debería estar disponible
  const { data: { session: refreshed } } = await supabase.auth.getSession();
  return refreshed?.access_token ?? null;
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
    // No hacer signOut automático — el middleware protege las rutas.
    // signOut aquí destruiría la sesión ante un 401 transitorio (race condition al cargar).
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
  onboarding_done: boolean;
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

export interface AppointmentListParams {
  date_from: string;
  date_to: string;
  timezone?: string;
  professional_id?: string;
  service_id?: string;
  status?: string;
  search?: string;
  sort_by?: string;
  sort_dir?: string;
  page?: number;
  per_page?: number;
}

export interface PaginatedAppointments {
  data: Appointment[];
  pagination: {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
  };
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
    const result = (await res.json()) as {
      token: string;
      refresh_token: string;
      user: UserDTO;
      tenant: TenantDTO;
    };

    // 2. Inicializar la sesión de Supabase con los tokens del Go API
    if (typeof window !== 'undefined') {
      const supabase = createClient();
      await supabase.auth.setSession({
        access_token: result.token,
        refresh_token: result.refresh_token,
      });
    }
    return result;
  },

  async me() {
    return request<{ user: UserDTO; tenant: TenantDTO }>('/api/v1/me');
  },

  async completeOnboarding() {
    return request<{ ok: boolean }>('/api/v1/onboarding/complete', { method: 'POST' });
  },
};

// ── Appointments ───────────────────────────────────────────────────────────────

export const appointments = {
  async list(date: string, timezone?: string): Promise<{ data: Appointment[] }> {
    const tz = timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone;
    return request(`/api/v1/appointments?date=${date}&timezone=${encodeURIComponent(tz)}`);
  },

  async listFiltered(params: AppointmentListParams): Promise<PaginatedAppointments> {
    const tz = params.timezone ?? Intl.DateTimeFormat().resolvedOptions().timeZone;
    const searchParams = new URLSearchParams();
    searchParams.set('date_from', params.date_from);
    searchParams.set('date_to', params.date_to);
    searchParams.set('timezone', tz);
    if (params.professional_id) searchParams.set('professional_id', params.professional_id);
    if (params.service_id) searchParams.set('service_id', params.service_id);
    if (params.status) searchParams.set('status', params.status);
    if (params.search) searchParams.set('search', params.search);
    if (params.sort_by) searchParams.set('sort_by', params.sort_by);
    if (params.sort_dir) searchParams.set('sort_dir', params.sort_dir);
    if (params.page) searchParams.set('page', String(params.page));
    if (params.per_page) searchParams.set('per_page', String(params.per_page));
    return request(`/api/v1/appointments/search?${searchParams.toString()}`);
  },

  async updateStatus(id: string, update: { status?: string; internal_notes?: string; cancellation_reason?: string }) {
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
      `/api/v1/appointments/availability?professional_id=${professionalId}&service_id=${serviceId}&date=${date}&timezone=${encodeURIComponent(tz)}`,
    );
  },
};

// ── Professionals ──────────────────────────────────────────────────────────────

export interface Schedule {
  day_of_week: number; // 0=Dom, 1=Lun, ..., 6=Sáb
  start_time: string; // "HH:MM"
  end_time: string; // "HH:MM"
  is_active: boolean;
}

export interface ProfessionalInput {
  name: string;
  specialty?: string;
  bio?: string;
  color?: string;
  is_active?: boolean;
}

export const professionals = {
  async list(): Promise<{ data: Professional[] }> {
    return request('/api/v1/professionals');
  },

  async create(data: ProfessionalInput): Promise<Professional> {
    return request('/api/v1/professionals', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async update(id: string, data: Partial<ProfessionalInput>): Promise<Professional> {
    return request(`/api/v1/professionals/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async getSchedule(id: string): Promise<{ data: Schedule[] }> {
    return request(`/api/v1/professionals/${id}/schedule`);
  },

  async setSchedule(id: string, schedules: Omit<Schedule, never>[]): Promise<{ data: Schedule[] }> {
    return request(`/api/v1/professionals/${id}/schedule`, {
      method: 'PUT',
      body: JSON.stringify(schedules),
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

// ── CRM: Pipeline Stages ──────────────────────────────────────────────────────

export interface PipelineStage {
  id: string;
  tenant_id: string;
  name: string;
  position: number;
  color: string;
  is_default: boolean;
  auto_rules_enabled: boolean;
  created_at: string;
}

// ── CRM: Treatments ──────────────────────────────────────────────────────────

export interface Treatment {
  id: string;
  tenant_id: string;
  customer_id: string;
  professional_id: string;
  name: string;
  treatment_type: string;
  status: string; // 'proposed' | 'accepted' | 'in_progress' | 'completed' | 'abandoned'
  total_sessions: number | null;
  completed_sessions: number;
  estimated_cost: number | null;
  paid_amount: number;
  currency: string;
  tooth_numbers: number[];
  notes: string;
  started_at: string | null;
  completed_at: string | null;
  next_session_at: string | null;
  created_at: string;
  updated_at: string;
  // Campos expandidos del JOIN (opcionales, dependen del endpoint)
  customer_name?: string;
  professional_name?: string;
}

// ── CRM: Tasks ───────────────────────────────────────────────────────────────

export interface Task {
  id: string;
  tenant_id: string;
  assigned_to: string | null;
  customer_id: string | null;
  appointment_id: string | null;
  treatment_id: string | null;
  title: string;
  description: string;
  status: string; // 'pending' | 'in_progress' | 'completed' | 'dismissed'
  due_at: string | null;
  completed_at: string | null;
  source: string; // 'manual' | 'rule' | 'system'
  rule_id: string | null;
  created_at: string;
  // Campos expandidos (opcionales)
  assigned_to_name?: string;
  customer_name?: string;
}

// ── CRM: Automation Rules ────────────────────────────────────────────────────

export interface Rule {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  trigger_type: string; // 'event' | 'temporal'
  trigger_event: string;
  trigger_schedule: { interval_days: number; reference_field: string } | null;
  conditions: Array<{ field: string; op: string; value: unknown }>;
  actions: Array<{ type: string; template: string; params: Record<string, unknown> }>;
  is_active: boolean;
  is_template: boolean;
  template_key: string;
  cooldown_hours: number;
  priority: number;
  created_at: string;
  updated_at: string;
}

export interface RuleExecution {
  id: string;
  rule_id: string;
  customer_id: string | null;
  triggered_at: string;
  trigger_event: string;
  status: string; // 'success' | 'failed' | 'skipped'
  error_message: string;
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
  stage_id?: string | null;
  last_visit_at?: string | null;
  next_recall_at?: string | null;
  lifetime_value?: number;
  acquisition_source?: string | null;
  created_at: string;
}

export const customers = {
  async list(search = '', limit = 50, offset = 0): Promise<{ data: Customer[] }> {
    const q = new URLSearchParams({ search, limit: String(limit), offset: String(offset) });
    return request(`/api/v1/customers?${q}`);
  },

  async getById(id: string): Promise<Customer> {
    return request(`/api/v1/customers/${id}`);
  },

  async updateStage(id: string, stageId: string | null): Promise<Customer> {
    return request(`/api/v1/customers/${id}/stage`, {
      method: 'PATCH',
      body: JSON.stringify({ stage_id: stageId }),
    });
  },
};

// ── Pipeline Stages ──────────────────────────────────────────────────────────

export const pipelineStages = {
  async list(): Promise<{ data: PipelineStage[] }> {
    return request('/api/v1/pipeline-stages');
  },

  async create(data: { name: string; color: string; position?: number }): Promise<PipelineStage> {
    return request('/api/v1/pipeline-stages', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async update(id: string, data: Partial<{ name: string; color: string; position: number; auto_rules_enabled: boolean }>): Promise<PipelineStage> {
    return request(`/api/v1/pipeline-stages/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async remove(id: string): Promise<void> {
    return request(`/api/v1/pipeline-stages/${id}`, { method: 'DELETE' });
  },

  async reorder(orderedIds: string[]): Promise<void> {
    return request('/api/v1/pipeline-stages/reorder', {
      method: 'PUT',
      body: JSON.stringify({ ids: orderedIds }),
    });
  },

  async loadTemplate(): Promise<{ ok: boolean; stages: string[] }> {
    return request('/api/v1/pipeline-stages/load-template', { method: 'POST' });
  },
};

// ── Treatments ───────────────────────────────────────────────────────────────

export const treatments = {
  async list(params?: {
    customer_id?: string;
    professional_id?: string;
    status?: string;
    search?: string;
  }): Promise<{ data: Treatment[] }> {
    const qs = new URLSearchParams();
    if (params?.customer_id) qs.set('customer_id', params.customer_id);
    if (params?.professional_id) qs.set('professional_id', params.professional_id);
    if (params?.status) qs.set('status', params.status);
    if (params?.search) qs.set('search', params.search);
    const query = qs.toString();
    return request(`/api/v1/treatments${query ? `?${query}` : ''}`);
  },

  async getById(id: string): Promise<Treatment> {
    return request(`/api/v1/treatments/${id}`);
  },

  async create(data: {
    customer_id: string;
    professional_id: string;
    name: string;
    treatment_type: string;
    total_sessions?: number;
    estimated_cost?: number;
    currency?: string;
    tooth_numbers?: number[];
    notes?: string;
  }): Promise<Treatment> {
    return request('/api/v1/treatments', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async update(id: string, data: Partial<{
    name: string;
    treatment_type: string;
    total_sessions: number;
    estimated_cost: number;
    notes: string;
    tooth_numbers: number[];
  }>): Promise<Treatment> {
    return request(`/api/v1/treatments/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async updateStatus(id: string, status: string): Promise<Treatment> {
    return request(`/api/v1/treatments/${id}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    });
  },
};

// ── Tasks ────────────────────────────────────────────────────────────────────

export const tasks = {
  async list(params?: {
    status?: string;
    assigned_to?: string;
    customer_id?: string;
    source?: string;
  }): Promise<{ data: Task[] }> {
    const qs = new URLSearchParams();
    if (params?.status) qs.set('status', params.status);
    if (params?.assigned_to) qs.set('assigned_to', params.assigned_to);
    if (params?.customer_id) qs.set('customer_id', params.customer_id);
    if (params?.source) qs.set('source', params.source);
    const query = qs.toString();
    return request(`/api/v1/tasks${query ? `?${query}` : ''}`);
  },

  async getById(id: string): Promise<Task> {
    return request(`/api/v1/tasks/${id}`);
  },

  async create(data: {
    title: string;
    description?: string;
    assigned_to?: string;
    customer_id?: string;
    treatment_id?: string;
    due_at?: string;
  }): Promise<Task> {
    return request('/api/v1/tasks', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async update(id: string, data: Partial<{
    title: string;
    description: string;
    assigned_to: string;
    due_at: string;
    status: string;
  }>): Promise<Task> {
    return request(`/api/v1/tasks/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async complete(id: string): Promise<Task> {
    return request(`/api/v1/tasks/${id}/complete`, { method: 'POST' });
  },

  async dismiss(id: string): Promise<Task> {
    return request(`/api/v1/tasks/${id}/dismiss`, { method: 'POST' });
  },
};

// ── Automation Rules ─────────────────────────────────────────────────────────

export const rules = {
  async list(): Promise<{ data: Rule[] }> {
    return request('/api/v1/rules');
  },

  async getById(id: string): Promise<Rule> {
    return request(`/api/v1/rules/${id}`);
  },

  async create(data: {
    name: string;
    description?: string;
    trigger_type: string;
    trigger_event: string;
    trigger_schedule?: { interval_days: number; reference_field: string };
    conditions?: Array<{ field: string; op: string; value: unknown }>;
    actions: Array<{ type: string; template: string; params: Record<string, unknown> }>;
    cooldown_hours?: number;
    priority?: number;
  }): Promise<Rule> {
    return request('/api/v1/rules', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async update(id: string, data: Partial<{
    name: string;
    description: string;
    trigger_type: string;
    trigger_event: string;
    trigger_schedule: { interval_days: number; reference_field: string };
    conditions: Array<{ field: string; op: string; value: unknown }>;
    actions: Array<{ type: string; template: string; params: Record<string, unknown> }>;
    is_active: boolean;
    cooldown_hours: number;
    priority: number;
  }>): Promise<Rule> {
    return request(`/api/v1/rules/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },

  async remove(id: string): Promise<void> {
    return request(`/api/v1/rules/${id}`, { method: 'DELETE' });
  },

  async listExecutions(ruleId: string): Promise<{ data: RuleExecution[] }> {
    return request(`/api/v1/rules/${ruleId}/executions`);
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
      throw new APIError(401, 'Sesión expirada. Por favor inicia sesión de nuevo.', 'session_expired');
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
  booking_intro_text?: string;
  booking_success_text?: string;
  bot_name?: string;
  bot_greeting?: string;
  logo_url?: string;
  cover_url?: string;
  description?: string;
}

export interface TenantSettings {
  reminder_minutes: number[];
  booking_intro_text: string;
  booking_success_text: string;
  bot_name: string;
  bot_greeting: string;
  logo_url?: string;
  cover_url?: string;
  description?: string;
}

export type BrandingAssetKind = 'logo' | 'cover';

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

// ── Settings ──────────────────────────────────────────────────────────────────

export const settingsApi = {
  async get(): Promise<TenantSettings> {
    return request('/api/v1/settings');
  },

  async update(data: Partial<TenantSettings>): Promise<void> {
    return request('/api/v1/settings', {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  },
};

// ── Branding (logo / portada de la página pública) ───────────────────────────

async function uploadBrandingAsset(kind: BrandingAssetKind, file: File): Promise<{ url: string }> {
  const token = await getToken();
  const form = new FormData();
  form.append('file', file);

  const res = await fetch(`${API_URL}/api/v1/tenant/branding/${kind}`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: form,
  });

  if (res.status === 401) {
    throw new APIError(401, 'Sesión expirada. Por favor inicia sesión de nuevo.', 'session_expired');
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({ message: 'Error al subir el archivo' }));
    throw new APIError(res.status, body.message ?? body.error ?? 'Error al subir el archivo');
  }
  return res.json();
}

export const brandingApi = {
  uploadLogo: (file: File) => uploadBrandingAsset('logo', file),
  uploadCover: (file: File) => uploadBrandingAsset('cover', file),
  async remove(kind: BrandingAssetKind): Promise<void> {
    return request(`/api/v1/tenant/branding/${kind}`, { method: 'DELETE' });
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
      const {
        data: { session },
      } = await supabase.auth.getSession();
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
  amount: number; // centavos
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
      body: JSON.stringify({ plan, success_url: successUrl, cancel_url: cancelUrl }),
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

// ── CRM ──────────────────────────────────────────────────────────────────────

export interface StageCustomerCount {
  stage_id: string;
  stage_name: string;
  color: string;
  count: number;
}

export interface TopRuleMetric {
  rule_id: string;
  rule_name: string;
  executions: number;
}

export interface CRMMetrics {
  rules_fired_30d: number;
  rules_success_rate: number;
  active_treatments: number;
  pending_tasks: number;
  customers_per_stage: StageCustomerCount[];
  top_rules: TopRuleMetric[];
}

export const crm = {
  async getMetrics(): Promise<CRMMetrics> {
    return request('/api/v1/crm/metrics');
  },
};

// Compatibilidad con el export anterior
export const api = { health: () => request<{ status: string }>('/health') };
