// Datos estáticos de templates de chatbot — NO requieren API.
export interface ChatbotTemplate {
  id: string;
  name: string;
  icon: string;
  defaults: {
    bot_name: string;
    bot_greeting: string;
    tone: 'friendly' | 'professional' | 'premium' | 'casual';
  };
  suggested_docs: Array<{
    category: string;
    title: string;
    content: string;
  }>;
}

export const CHATBOT_TEMPLATES: ChatbotTemplate[] = [
  {
    id: 'salon',
    name: 'Peluquería',
    icon: '💇',
    defaults: {
      bot_name: 'Luna',
      bot_greeting: '¡Hola! Soy Luna, asistente virtual de {business_name}. ¿En qué puedo ayudarte hoy? 😊',
      tone: 'friendly',
    },
    suggested_docs: [
      {
        category: 'faq',
        title: 'Preguntas frecuentes',
        content: '## Preguntas frecuentes\n\n**¿Cómo puedo cancelar mi cita?**\nPodés cancelar tu cita hasta 4 horas antes del horario programado sin costo. Cancelaciones tardías pueden tener un cargo.\n\n**¿Qué métodos de pago aceptan?**\nAceptamos efectivo, tarjetas de débito y crédito.\n\n**¿Necesito reservar con anticipación?**\nRecomendamos reservar al menos 24 horas antes para garantizar tu horario preferido.',
      },
      {
        category: 'policies',
        title: 'Políticas del salón',
        content: '## Políticas\n\n**Cancelación:** Cancelar con al menos 4 horas de anticipación. Cancelaciones tardías: cargo del 50% del servicio.\n\n**Puntualidad:** Si llegás más de 15 minutos tarde, la cita podría ser reprogramada.\n\n**Menores de edad:** Los menores de 16 años deben venir acompañados de un adulto.',
      },
    ],
  },
  {
    id: 'dental',
    name: 'Dental',
    icon: '🦷',
    defaults: {
      bot_name: 'Dra. Asistente',
      bot_greeting: '¡Bienvenido/a a {business_name}! Soy la asistente virtual de la clínica. ¿En qué puedo ayudarle?',
      tone: 'professional',
    },
    suggested_docs: [
      {
        category: 'faq',
        title: 'Preguntas frecuentes — primera cita',
        content: '## Preguntas frecuentes\n\n**¿Qué debo llevar a mi primera cita?**\nDocumento de identidad, historial médico y lista de medicamentos actuales.\n\n**¿Aceptan seguros dentales?**\nTrabajamos con las principales aseguradoras. Consulte si su plan está incluido.\n\n**¿Cuánto dura una consulta general?**\nLa primera consulta dura aproximadamente 45 minutos incluyendo evaluación y radiografías.',
      },
      {
        category: 'policies',
        title: 'Políticas de la clínica',
        content: '## Políticas\n\n**Cancelación:** Cancelar con 24 horas de anticipación. Cancelaciones sin aviso: cargo de consulta completo.\n\n**Preparación:** Cepillarse los dientes antes de la consulta. No consumir café o alimentos pigmentados 2 horas antes de blanqueamiento.',
      },
    ],
  },
  {
    id: 'spa',
    name: 'Spa / Estética',
    icon: '🧖',
    defaults: {
      bot_name: 'Serenity',
      bot_greeting: 'Bienvenido/a a {business_name}. Soy Serenity, tu asistente personal. ¿En qué puedo ayudarte para tu próxima experiencia de bienestar?',
      tone: 'premium',
    },
    suggested_docs: [
      {
        category: 'faq',
        title: 'Preguntas frecuentes',
        content: '## Preguntas frecuentes\n\n**¿Hay contraindicaciones para los tratamientos?**\nAlgunos tratamientos no son recomendados durante el embarazo o con ciertas condiciones médicas. Consultá con nuestro equipo.\n\n**¿Ofrecen paquetes?**\nSí, tenemos paquetes especiales de relajación y belleza con descuentos. Preguntá por nuestras promociones vigentes.',
      },
      {
        category: 'policies',
        title: 'Cuidados post-tratamiento',
        content: '## Cuidados post-tratamiento\n\n**Faciales:** Evitar maquillaje por 24 horas. Usar protector solar SPF 50+.\n\n**Masajes:** Hidratarse bien. Evitar ejercicio intenso por 12 horas.\n\n**Depilación:** No exponerse al sol por 48 horas. Hidratar la zona tratada.',
      },
    ],
  },
  {
    id: 'barbershop',
    name: 'Barbería',
    icon: '💈',
    defaults: {
      bot_name: 'Barber Bot',
      bot_greeting: '¡Qué onda! Soy el asistente de {business_name}. ¿Querés agendar un corte o tenés alguna pregunta?',
      tone: 'casual',
    },
    suggested_docs: [
      {
        category: 'faq',
        title: 'Preguntas frecuentes',
        content: '## Preguntas frecuentes\n\n**¿Necesito turno previo?**\nRecomendamos reservar, pero también atendemos sin turno según disponibilidad.\n\n**¿Cuánto demora un corte?**\nUn corte clásico toma unos 30 minutos. Corte + barba unos 45 minutos.',
      },
      {
        category: 'services',
        title: 'Servicios y precios',
        content: '## Servicios\n\n- Corte clásico\n- Corte + barba\n- Afeitado tradicional\n- Tratamiento capilar\n- Cejas\n\nConsultá precios actualizados con nuestro equipo.',
      },
    ],
  },
  {
    id: 'generic',
    name: 'Genérico',
    icon: '🤖',
    defaults: {
      bot_name: 'Asistente',
      bot_greeting: '¡Hola! Soy el asistente virtual de {business_name}. ¿En qué puedo ayudarte?',
      tone: 'professional',
    },
    suggested_docs: [
      {
        category: 'faq',
        title: 'Preguntas frecuentes',
        content: '## Preguntas frecuentes\n\n**¿Cómo puedo agendar una cita?**\nPodés agendar directamente por WhatsApp o desde nuestra página web.\n\n**¿Cuáles son los horarios de atención?**\nLunes a viernes de 9:00 a 18:00. Sábados de 9:00 a 13:00.',
      },
      {
        category: 'policies',
        title: 'Políticas generales',
        content: '## Políticas\n\n**Cancelación:** Cancelar con al menos 4 horas de anticipación.\n\n**Reagendamiento:** Podés reagendar tu cita sin costo hasta 2 horas antes.',
      },
    ],
  },
];
