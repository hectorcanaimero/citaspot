# Análisis de Costos — CitaSpot por Tenant

> Estimado basado en el stack real del proyecto: Gemini 2.5 Flash + Evolution API (self-hosted) + Hostinger VPS.
> Precios referenciados: marzo 2026.

---

## Variables base

### Modelos de IA

| Servicio | Precio |
|----------|--------|
| Gemini 2.5 Flash — Input | $0.075 / 1M tokens |
| Gemini 2.5 Flash — Output | $0.30 / 1M tokens |
| OpenAI text-embedding-3-small | $0.02 / 1M tokens |

### Consumo LLM por mensaje WhatsApp

Cada mensaje entrante genera:
- **Intent detection** (siempre): ~350 tokens (280 input + 70 output)
- **RAG response** (solo consultas, ~20% del tráfico): ~1,600 tokens adicionales (1,200 input + 400 output)
- **Booking flow** (80% del tráfico): solo intent detection — el flujo es determinístico, sin LLM extra

---

## Volumen por tipo de negocio

| Tipo | Citas/mes | Msgs/cita | Consultas/mes | Msgs/consulta | **Total msgs/mes** |
|------|-----------|-----------|---------------|---------------|-------------------|
| Clínica dental | 50 | 14 | 30 | 5 | **850** |
| Barbería | 100 | 8 | 20 | 3 | **860** |
| Peluquería | 80 | 11 | 40 | 5 | **1,080** |
| **Promedio usado** | **77** | **11** | **30** | **4** | **~950** |

### Cálculo LLM por tenant/mes (~950 msgs)

```
Booking msgs (760, 80%):
  Intent: 760 × (280 input + 70 output) = 213K input + 53K output

Consultas (190, 20%):
  Intent:    190 × (280 + 70)   =  53K input + 13K output
  Respuesta: 190 × (1,200 + 400) = 228K input + 76K output

TOTAL: 494K input + 142K output
COSTO: (494K × $0.075 + 142K × $0.30) / 1M = $0.037 + $0.043 = ~$0.08/tenant/mes
```

> **Conclusión: el LLM cuesta menos de $0.10 por tenant al mes. El costo dominante es el servidor.**

---

## Infraestructura — Hostinger VPS (KVM, contrato anual)

| Plan | vCPU | RAM | Precio/mes |
|------|------|-----|-----------|
| KVM 1 | 1 | 4 GB | ~$5 |
| KVM 2 | 2 | 8 GB | ~$7 |
| KVM 4 | 4 | 16 GB | ~$14 |
| KVM 8 | 8 | 32 GB | ~$26 |

> **Limitante real de RAM:** Evolution API consume ~100–200 MB por sesión WhatsApp activa.
> Con 50 tenants → ~7.5 GB solo para Evolution API. Dimensionar en consecuencia.

---

## Costos por escala

### 1 Tenant

| Componente | Costo/mes |
|------------|-----------|
| Hostinger KVM 1 (1 vCPU / 4 GB RAM) | $5.00 |
| Gemini 2.5 Flash | $0.08 |
| Embeddings OpenAI | $0.001 |
| Supabase (free tier — solo auth) | $0.00 |
| WhatsApp — Evolution API (self-hosted) | $0.00 |
| Stripe (1 sub × $15) | $0.74 |
| **TOTAL** | **~$5.82/mes** |

| Precio cliente | Ganancia/mes | Margen |
|---------------|-------------|--------|
| $15 | $9.18 | 61% |
| $25 | $19.18 | 77% |

---

### 10 Tenants

| Componente | Costo/mes |
|------------|-----------|
| Hostinger KVM 1 (10 sesiones WA ≈ 1.5 GB) | $5.00 |
| Gemini 2.5 Flash (×10) | $0.80 |
| Embeddings | $0.01 |
| Supabase free | $0.00 |
| Stripe (10 × $15) | $7.35 |
| **TOTAL** | **~$13/mes** |

| Precio cliente | MRR | Ganancia/mes | Costo/tenant | Margen |
|---------------|-----|-------------|-------------|--------|
| $15 | $150 | $137 | $1.32 | 91% |
| $25 | $250 | $237 | $1.32 | 95% |

---

### 50 Tenants

> 50 sesiones WA × ~150 MB = **~7.5 GB** → requiere al menos 16 GB de RAM

| Componente | Costo/mes |
|------------|-----------|
| Hostinger KVM 4 (4 vCPU / 16 GB RAM) | $14.00 |
| Gemini 2.5 Flash (×50) | $4.00 |
| Embeddings | $0.05 |
| Supabase Pro (recomendado a este punto) | $25.00 |
| Stripe (50 × $15) | $21.75 |
| **TOTAL** | **~$65/mes** |

| Precio cliente | MRR | Ganancia/mes | Costo/tenant | Margen |
|---------------|-----|-------------|-------------|--------|
| $15 | $750 | $685 | $1.30 | 91% |
| $25 | $1,250 | $1,185 | $1.30 | 95% |

---

### 100 Tenants

> 100 sesiones WA × ~150 MB = **~15 GB** → requiere 32 GB de RAM

| Componente | Costo/mes |
|------------|-----------|
| Hostinger KVM 8 (8 vCPU / 32 GB RAM) | $26.00 |
| Gemini 2.5 Flash (×100) | $8.30 |
| Embeddings | $0.10 |
| Supabase Pro | $25.00 |
| Stripe (100 × $15) | $43.50 |
| Backups adicionales | $5.00 |
| **TOTAL** | **~$108/mes** |

| Precio cliente | MRR | Ganancia/mes | Costo/tenant | Margen |
|---------------|-----|-------------|-------------|--------|
| $15 | $1,500 | $1,392 | $1.08 | 93% |
| $25 | $2,500 | $2,392 | $1.08 | 96% |

---

## Resumen ejecutivo

| Escala | Plan Hostinger | Costo total/mes | Costo/tenant | Margen a $15 | Margen a $25 |
|--------|---------------|----------------|-------------|-------------|-------------|
| **1 tenant** | KVM 1 ($5) | $5.82 | $5.82 | 61% | 77% |
| **10 tenants** | KVM 1 ($5) | $13 | $1.32 | 91% | 95% |
| **50 tenants** | KVM 4 ($14) | $65 | $1.30 | 91% | 95% |
| **100 tenants** | KVM 8 ($26) | $108 | $1.08 | 93% | 96% |

---

## Conclusiones y recomendaciones

### 1. El negocio es rentable desde el primer cliente
Break-even real: **1 tenant**. Desde el primer cliente a $15/mes ya cubres el servidor con margen.

### 2. RAM es el limitante, no CPU ni disco
Evolution API es el servicio más hambrienta de memoria (~150 MB/sesión). Monitorea el consumo por sesión WhatsApp activa antes de agregar tenants.

### 3. El costo de IA es marginal
$8.30/mes para 100 tenants. Aunque el uso se triplique, el LLM seguirá siendo < $25/mes. Gemini 2.5 Flash es extremadamente barato para este tipo de carga.

### 4. A 30+ tenants, considera separar Evolution API
Mover Evolution API a un segundo VPS de $5–7/mes en Hostinger alivia el server principal y facilita escalar las sesiones WA de forma independiente.

### 5. Supabase free aguanta hasta ~50 tenants activos
El plan Pro ($25/mes) se justifica cuando superás los 50 tenants o si querés métricas y soporte prioritario.

### 6. Stripe es el segundo costo más relevante a escala
A 100 tenants, Stripe ($43/mes) supera al LLM ($8/mes). A escala mayor conviene evaluar opciones como Lemon Squeezy o MercadoPago (2.9% + sin fee fijo en algunos planes).

---

## Proyección de rentabilidad

```
Plan $15/tenant:
  10 tenants  →  MRR $150   →  Ganancia neta ~$137/mes
  50 tenants  →  MRR $750   →  Ganancia neta ~$685/mes
  100 tenants →  MRR $1,500 →  Ganancia neta ~$1,392/mes

Plan $25/tenant:
  10 tenants  →  MRR $250   →  Ganancia neta ~$237/mes
  50 tenants  →  MRR $1,250 →  Ganancia neta ~$1,185/mes
  100 tenants →  MRR $2,500 →  Ganancia neta ~$2,392/mes
```

---

*Fuentes: [Hostinger VPS Pricing](https://www.hostinger.com/pricing/vps-hosting) · [Google AI Pricing](https://ai.google.dev/pricing) · [OpenAI Pricing](https://openai.com/pricing) · Estimados basados en el stack real de CitaSpot.*
