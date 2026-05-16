# Mensajes del asistente de WhatsApp por idioma.
# Configurar con la variable de entorno WHATSAPP_LANGUAGE (default: 'es').
# Soporta: es (español), en (inglés), pt (portugués brasileño).

MESSAGES: dict[str, dict[str, str]] = {
    "es": {
        "slow_response": "Un momento, estoy consultando eso... ⏳",
        "handoff": "Te conecto con nuestro equipo ahora mismo. ¡Un agente te atenderá pronto! 🙋",
        "booking_cancelled": "Entendido, he cancelado la reserva. ¿Puedo ayudarte con algo más?",
        "cancel_cta": "Entendido. Si necesitas cancelar una cita específica, dime el día y hora de tu cita.",
        "no_services": "Lo siento, no hay servicios disponibles en este momento. Intenta más tarde.",
        "choose_service_header": "¡Claro! ¿Qué servicio te gustaría agendar? 😊",
        "choose_service_footer": "Responde con el número de tu elección.",
        "invalid_option": "Por favor elige una opción válida:",
        "choose_professional": "¿Con qué profesional prefieres tu {service}?",
        "no_professionals": "Lo siento, no hay profesionales disponibles en este momento.",
        "choose_date_with_prof": (
            "¿Para qué fecha quieres tu {service} con {professional}? "
            "Dime el día y mes (ej: 15-01 o 15/01)."
        ),
        "choose_professional_prompt": "Por favor elige un profesional:",
        "choose_date_short": "¿Para qué fecha quieres tu cita con {professional}? Dime el día y mes (ej: 15-01 o 15/01).",
        "invalid_date_format": "No entendí la fecha. Dime el día y mes, por ejemplo: 15-01 o 15/01.",
        "no_slots": "Lo siento, no hay horarios disponibles el {date}. ¿Quieres probar con otra fecha?",
        "available_slots_header": "Horarios disponibles para el {date}:",
        "available_slots_footer": "Elige el número de tu horario preferido.",
        "invalid_slot": "Por favor elige un número del 1 al {max}.",
        "confirm_prompt": (
            "¿Confirmas tu cita de *{service}* con {professional} el {datetime}? "
            "Responde *Sí* para confirmar o *No* para cancelar."
        ),
        "ask_name": "¿Cuál es tu nombre completo para la reserva? 😊",
        "booking_failed": (
            "Lo siento, no pude completar la reserva. "
            "Por favor intenta de nuevo o escríbenos para ayudarte."
        ),
        "booking_confirmed": (
            "¡Tu cita está confirmada! ✅\n"
            "*{service}* — {datetime}\n"
            "Recibirás un recordatorio 24h antes. ¡Nos vemos! 🙌"
        ),
        "invalid_name": "Por favor dime tu nombre completo.",
        # Partes del system prompt
        "system_intro": (
            "Eres {bot_name} de {business_name}, un negocio de salud y belleza.\n"
            "Tu objetivo es ayudar a los clientes a agendar citas, responder preguntas sobre "
            "servicios y dar información útil.\n\n"
            "Pautas de comunicación:\n"
            "- Tono amable, profesional y conciso\n"
            "- Máximo 3 oraciones por respuesta\n"
            "- Si el cliente quiere agendar: guíalo paso a paso (servicio → profesional → fecha → confirmar)\n"
            "- Si no puedes resolver algo: ofrece escalar con un humano"
        ),
        "system_services_header": "Servicios disponibles:",
        "system_professionals_header": "Equipo profesional:",
        "system_mapping_header": "Profesionales por servicio:",
        "system_rag_header": "Información adicional del negocio:",
        "system_warning": "IMPORTANTE: Nunca inventes información. Si no sabes algo, di que vas a verificarlo.",
        # Consulta de citas
        "no_appointments": "No encontré citas agendadas a tu nombre. ¿Te gustaría agendar una? 😊",
        "my_appointments_header": "Tus próximas citas:",
        "appointment_line": "{idx}. *{service}* — {datetime} con {professional}",
        # Flujo de cancelación
        "cancel_which": "¿Cuál cita querés cancelar?",
        "cancel_confirm": "¿Confirmas que querés cancelar tu cita de *{service}* el {datetime}?",
        "cancel_success": "Tu cita ha sido cancelada. ¿Puedo ayudarte con algo más?",
        "cancel_failed": "No pude cancelar la cita. Por favor intenta de nuevo o escríbenos.",
        "no_appointments_cancel": "No encontré citas pendientes para cancelar.",
        # Flujo de reagendamiento
        "reschedule_which": "¿Cuál cita querés reagendar?",
        "reschedule_date": "¿Para qué fecha querés mover tu *{service}*? Dime día y mes (ej: 23-05).",
        "reschedule_confirm": "¿Confirmas mover tu *{service}* al {datetime}?",
        "reschedule_success": "¡Listo! Tu cita fue movida al {datetime}. ¡Nos vemos! 🙌",
        "reschedule_failed": "No pude reagendar la cita. Por favor intenta de nuevo.",
        "no_appointments_reschedule": "No encontré citas pendientes para reagendar.",
        # Smart booking link
        "booking_link": "Tenemos varias opciones disponibles. Te comparto este link para que elijas cómodamente:\n{url}",
    },
    "en": {
        "slow_response": "One moment, I'm looking that up... ⏳",
        "handoff": "I'm connecting you with our team right now. An agent will assist you shortly! 🙋",
        "booking_cancelled": "Understood, I've cancelled the reservation. Can I help you with anything else?",
        "cancel_cta": "Got it. If you need to cancel a specific appointment, let me know the day and time.",
        "no_services": "Sorry, there are no services available at the moment. Please try again later.",
        "choose_service_header": "Sure! What service would you like to schedule? 😊",
        "choose_service_footer": "Reply with the number of your choice.",
        "invalid_option": "Please choose a valid option:",
        "choose_professional": "Which professional would you prefer for your {service}?",
        "no_professionals": "Sorry, there are no professionals available at the moment.",
        "choose_date_with_prof": (
            "What date would you like your {service} with {professional}? "
            "Tell me the day and month (e.g., 15-01 or 15/01)."
        ),
        "choose_professional_prompt": "Please choose a professional:",
        "choose_date_short": "What date would you like your appointment with {professional}? (e.g., 15-01 or 15/01).",
        "invalid_date_format": "I didn't get that date. Tell me the day and month, for example: 15-01 or 15/01.",
        "no_slots": "Sorry, there are no available slots on {date}. Would you like to try another date?",
        "available_slots_header": "Available slots for {date}:",
        "available_slots_footer": "Choose the number of your preferred time slot.",
        "invalid_slot": "Please choose a number from 1 to {max}.",
        "confirm_prompt": (
            "Do you confirm your *{service}* appointment with {professional} on {datetime}? "
            "Reply *Yes* to confirm or *No* to cancel."
        ),
        "ask_name": "What is your full name for the booking? 😊",
        "booking_failed": (
            "Sorry, I couldn't complete the booking. "
            "Please try again or contact us for help."
        ),
        "booking_confirmed": (
            "Your appointment is confirmed! ✅\n"
            "*{service}* — {datetime}\n"
            "You'll receive a reminder 24h before. See you then! 🙌"
        ),
        "invalid_name": "Please provide your full name.",
        # System prompt parts
        "system_intro": (
            "You are {bot_name} at {business_name}, a health and beauty business.\n"
            "Your goal is to help clients schedule appointments, answer questions about services, "
            "and provide useful information.\n\n"
            "Communication guidelines:\n"
            "- Friendly, professional, and concise tone\n"
            "- Maximum 3 sentences per response\n"
            "- If the client wants to schedule: guide them step by step (service → professional → date → confirm)\n"
            "- If you can't resolve something: offer to escalate to a human"
        ),
        "system_services_header": "Available services:",
        "system_professionals_header": "Professional team:",
        "system_mapping_header": "Professionals by service:",
        "system_rag_header": "Additional business information:",
        "system_warning": "IMPORTANT: Never make up information. If you don't know something, say you'll verify it.",
        # Appointment queries
        "no_appointments": "I didn't find any upcoming appointments for you. Would you like to book one? 😊",
        "my_appointments_header": "Your upcoming appointments:",
        "appointment_line": "{idx}. *{service}* — {datetime} with {professional}",
        # Cancel flow
        "cancel_which": "Which appointment would you like to cancel?",
        "cancel_confirm": "Are you sure you want to cancel your *{service}* appointment on {datetime}?",
        "cancel_success": "Your appointment has been cancelled. Can I help you with anything else?",
        "cancel_failed": "I couldn't cancel the appointment. Please try again or contact us.",
        "no_appointments_cancel": "I didn't find any pending appointments to cancel.",
        # Reschedule flow
        "reschedule_which": "Which appointment would you like to reschedule?",
        "reschedule_date": "What date would you like to move your *{service}* to? Tell me the day and month (e.g., 23-05).",
        "reschedule_confirm": "Confirm moving your *{service}* to {datetime}?",
        "reschedule_success": "Done! Your appointment has been moved to {datetime}. See you then! 🙌",
        "reschedule_failed": "I couldn't reschedule the appointment. Please try again.",
        "no_appointments_reschedule": "I didn't find any pending appointments to reschedule.",
        # Smart booking link
        "booking_link": "We have several options available. Here's a link so you can choose at your convenience:\n{url}",
    },
    "pt": {
        "slow_response": "Um momento, estou consultando isso... ⏳",
        "handoff": "Estou te conectando com nossa equipe agora. Um agente vai te atender em breve! 🙋",
        "booking_cancelled": "Entendido, cancelei a reserva. Posso te ajudar com mais alguma coisa?",
        "cancel_cta": "Entendido. Se precisar cancelar uma consulta específica, me diga o dia e horário.",
        "no_services": "Desculpe, não há serviços disponíveis no momento. Tente novamente mais tarde.",
        "choose_service_header": "Claro! Qual serviço você gostaria de agendar? 😊",
        "choose_service_footer": "Responda com o número da sua escolha.",
        "invalid_option": "Por favor escolha uma opção válida:",
        "choose_professional": "Com qual profissional você prefere seu {service}?",
        "no_professionals": "Desculpe, não há profissionais disponíveis no momento.",
        "choose_date_with_prof": (
            "Para qual data você quer seu {service} com {professional}? "
            "Me diga o dia e mês (ex: 15-01 ou 15/01)."
        ),
        "choose_professional_prompt": "Por favor escolha um profissional:",
        "choose_date_short": "Para qual data você quer sua consulta com {professional}? (ex: 15-01 ou 15/01).",
        "invalid_date_format": "Não entendi a data. Me diga o dia e mês, por exemplo: 15-01 ou 15/01.",
        "no_slots": "Desculpe, não há horários disponíveis em {date}. Gostaria de tentar outra data?",
        "available_slots_header": "Horários disponíveis para {date}:",
        "available_slots_footer": "Escolha o número do horário preferido.",
        "invalid_slot": "Por favor escolha um número de 1 a {max}.",
        "confirm_prompt": (
            "Você confirma sua consulta de *{service}* com {professional} em {datetime}? "
            "Responda *Sim* para confirmar ou *Não* para cancelar."
        ),
        "ask_name": "Qual é o seu nome completo para a reserva? 😊",
        "booking_failed": (
            "Desculpe, não consegui completar a reserva. "
            "Por favor tente novamente ou entre em contato conosco."
        ),
        "booking_confirmed": (
            "Sua consulta está confirmada! ✅\n"
            "*{service}* — {datetime}\n"
            "Você receberá um lembrete 24h antes. Até lá! 🙌"
        ),
        "invalid_name": "Por favor informe seu nome completo.",
        # Partes do system prompt
        "system_intro": (
            "Você é {bot_name} de {business_name}, um negócio de saúde e beleza.\n"
            "Seu objetivo é ajudar os clientes a agendar consultas, responder perguntas sobre "
            "serviços e fornecer informações úteis.\n\n"
            "Diretrizes de comunicação:\n"
            "- Tom amigável, profissional e conciso\n"
            "- Máximo 3 frases por resposta\n"
            "- Se o cliente quiser agendar: guie-o passo a passo (serviço → profissional → data → confirmar)\n"
            "- Se não conseguir resolver algo: ofereça escalar para um humano"
        ),
        "system_services_header": "Serviços disponíveis:",
        "system_professionals_header": "Equipe profissional:",
        "system_mapping_header": "Profissionais por serviço:",
        "system_rag_header": "Informações adicionais do negócio:",
        "system_warning": "IMPORTANTE: Nunca invente informações. Se não souber algo, diga que vai verificar.",
        # Consulta de consultas
        "no_appointments": "Não encontrei consultas agendadas no seu nome. Gostaria de agendar uma? 😊",
        "my_appointments_header": "Suas próximas consultas:",
        "appointment_line": "{idx}. *{service}* — {datetime} com {professional}",
        # Fluxo de cancelamento
        "cancel_which": "Qual consulta você quer cancelar?",
        "cancel_confirm": "Confirma que quer cancelar sua consulta de *{service}* em {datetime}?",
        "cancel_success": "Sua consulta foi cancelada. Posso ajudar com mais alguma coisa?",
        "cancel_failed": "Não consegui cancelar a consulta. Tente novamente ou entre em contato.",
        "no_appointments_cancel": "Não encontrei consultas pendentes para cancelar.",
        # Fluxo de reagendamento
        "reschedule_which": "Qual consulta você quer reagendar?",
        "reschedule_date": "Para que data quer mover sua *{service}*? Me diga o dia e mês (ex: 23-05).",
        "reschedule_confirm": "Confirma mover sua *{service}* para {datetime}?",
        "reschedule_success": "Pronto! Sua consulta foi movida para {datetime}. Até lá! 🙌",
        "reschedule_failed": "Não consegui reagendar a consulta. Tente novamente.",
        "no_appointments_reschedule": "Não encontrei consultas pendentes para reagendar.",
        # Smart booking link
        "booking_link": "Temos várias opções disponíveis. Compartilho este link para que escolha com calma:\n{url}",
    },
}


def get_messages(lang: str) -> dict[str, str]:
    """Retorna el diccionario de mensajes para el idioma dado. Fallback a 'es'."""
    return MESSAGES.get(lang, MESSAGES["es"])
