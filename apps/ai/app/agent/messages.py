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
        "no_professionals_for_service": "Lo siento, no hay profesionales que ofrezcan {service} en este momento. ¿Querés elegir otro servicio?",
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
            "Eres {bot_name}, asistente conversacional de {business_name}.\n\n"
            "Esto es una CONVERSACIÓN por WhatsApp. {greeting_instruction}\n\n"
            "Cómo hablar:\n"
            "- Como una persona real, NO como un menú ni un formulario.\n"
            "- Respuestas cortas (1–3 oraciones). Sin párrafos largos.\n"
            "- Sin repetir el nombre del negocio en cada respuesta.\n"
            "- Sin saludos genéricos tipo \"¡Hola! ¿En qué puedo ayudarte?\" en mitad de la conversación.\n"
            "- Si el cliente quiere agendar, guialo paso a paso (servicio → profesional → fecha → confirmar).\n"
            "- Si no podés resolver algo, ofrecé escalar a un humano."
        ),
        "greeting_first_turn": "Es el PRIMER mensaje del cliente — podés saludar brevemente una sola vez.",
        "greeting_continuation": "Ya estás en medio de la conversación — NO saludes, continuá directo al punto.",
        "system_services_header": "Servicios disponibles:",
        "system_professionals_header": "Equipo profesional:",
        "system_mapping_header": "Profesionales por servicio:",
        "system_rag_header": (
            "Información oficial del negocio (USAR ESTO PRIMERO antes de responder cualquier "
            "pregunta del cliente — si la respuesta está acá, citala directamente):"
        ),
        "system_warning": "IMPORTANTE: Nunca inventes información. Si no sabes algo, di que vas a verificarlo.",
        "system_no_catalog": (
            "ATENCIÓN: Este negocio NO tiene servicios ni catálogo cargado en el sistema. "
            "Si el cliente pregunta por servicios, precios, horarios o cualquier oferta concreta, "
            "NO inventes nada bajo ningún concepto. Decile amablemente que el equipo aún está "
            "terminando de configurar la información del negocio y que un humano lo va a contactar. "
            "NUNCA respondas con servicios genéricos del rubro."
        ),
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
        # Saludo determinista de primer turno
        "first_turn_greeting": (
            "¡Hola! Soy el asistente de *{business_name}*. 👋\n\n"
            "Estos son nuestros servicios:\n{services_list}\n\n"
            "Decime el servicio que querés agendar, o si preferís te mando el link para reservar online."
        ),
        # Oferta suave del link (cuando el flujo se estanca o fallan las tools)
        "booking_link_soft": "Si te resulta más cómodo, también podés agendar desde la web:\n{url}",
        # Respuesta a pedido explícito del link
        "booking_link_explicit": "¡Claro! Acá tenés el link para reservar:\n{url}",
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
        "no_professionals_for_service": "Sorry, no professionals offer {service} at the moment. Would you like to choose another service?",
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
            "You are {bot_name}, the conversational assistant for {business_name}.\n\n"
            "This is a CONVERSATION over WhatsApp. {greeting_instruction}\n\n"
            "How to talk:\n"
            "- Like a real person, NOT like a menu or a form.\n"
            "- Short replies (1–3 sentences). No long paragraphs.\n"
            "- Don't repeat the business name in every response.\n"
            "- No generic greetings like \"Hello! How can I help you?\" mid-conversation.\n"
            "- If the client wants to book, guide them step by step (service → professional → date → confirm).\n"
            "- If you can't resolve something, offer to escalate to a human."
        ),
        "greeting_first_turn": "This is the FIRST message from the client — you may greet them briefly, just once.",
        "greeting_continuation": "You are already mid-conversation — DO NOT greet, get straight to the point.",
        "system_services_header": "Available services:",
        "system_professionals_header": "Professional team:",
        "system_mapping_header": "Professionals by service:",
        "system_rag_header": (
            "Official business information (USE THIS FIRST before answering any client question — "
            "if the answer is here, quote it directly):"
        ),
        "system_warning": "IMPORTANT: Never make up information. If you don't know something, say you'll verify it.",
        "system_no_catalog": (
            "ATTENTION: This business has NO services or catalog loaded in the system. "
            "If the client asks about services, prices, hours or any concrete offering, "
            "DO NOT invent anything under any circumstance. Kindly tell them that the team "
            "is still finishing the business setup and that a human will contact them. "
            "NEVER respond with generic industry services."
        ),
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
        # Deterministic first-turn greeting
        "first_turn_greeting": (
            "Hi! I'm the assistant for *{business_name}*. 👋\n\n"
            "Here are our services:\n{services_list}\n\n"
            "Tell me which service you'd like to book, or if you prefer I can send you the link to book online."
        ),
        # Soft link offer (when flow stalls or tools fail)
        "booking_link_soft": "If it's easier for you, you can also book from the web:\n{url}",
        # Reply to explicit link request
        "booking_link_explicit": "Of course! Here's the link to book:\n{url}",
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
        "no_professionals_for_service": "Desculpe, não há profissionais que ofereçam {service} no momento. Quer escolher outro serviço?",
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
            "Você é {bot_name}, assistente conversacional de {business_name}.\n\n"
            "Isto é uma CONVERSA pelo WhatsApp. {greeting_instruction}\n\n"
            "Como falar:\n"
            "- Como uma pessoa real, NÃO como um menu ou formulário.\n"
            "- Respostas curtas (1–3 frases). Sem parágrafos longos.\n"
            "- Sem repetir o nome do negócio em cada resposta.\n"
            "- Sem saudações genéricas tipo \"Olá! Como posso ajudar?\" no meio da conversa.\n"
            "- Se o cliente quiser agendar, guie passo a passo (serviço → profissional → data → confirmar).\n"
            "- Se não conseguir resolver algo, ofereça escalar para um humano."
        ),
        "greeting_first_turn": "É a PRIMEIRA mensagem do cliente — você pode saudar brevemente uma única vez.",
        "greeting_continuation": "Você já está no meio da conversa — NÃO cumprimente, vá direto ao ponto.",
        "system_services_header": "Serviços disponíveis:",
        "system_professionals_header": "Equipe profissional:",
        "system_mapping_header": "Profissionais por serviço:",
        "system_rag_header": (
            "Informações oficiais do negócio (USE ISSO PRIMEIRO antes de responder qualquer "
            "pergunta do cliente — se a resposta está aqui, cite-a diretamente):"
        ),
        "system_warning": "IMPORTANTE: Nunca invente informações. Se não souber algo, diga que vai verificar.",
        "system_no_catalog": (
            "ATENÇÃO: Este negócio NÃO tem serviços nem catálogo carregado no sistema. "
            "Se o cliente perguntar por serviços, preços, horários ou qualquer oferta concreta, "
            "NÃO invente nada em hipótese alguma. Diga gentilmente que a equipe ainda está "
            "terminando de configurar as informações e que um humano vai entrar em contato. "
            "NUNCA responda com serviços genéricos do setor."
        ),
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
        # Saudação determinística de primeiro turno
        "first_turn_greeting": (
            "Olá! Sou o assistente de *{business_name}*. 👋\n\n"
            "Estes são nossos serviços:\n{services_list}\n\n"
            "Me diga qual serviço você quer agendar, ou se preferir eu te mando o link para reservar online."
        ),
        # Oferta suave do link (quando o fluxo trava ou as tools falham)
        "booking_link_soft": "Se for mais cômodo, você também pode agendar pela web:\n{url}",
        # Resposta a pedido explícito do link
        "booking_link_explicit": "Claro! Aqui está o link para reservar:\n{url}",
    },
}


def get_messages(lang: str) -> dict[str, str]:
    """Retorna el diccionario de mensajes para el idioma dado. Fallback a 'es'."""
    return MESSAGES.get(lang, MESSAGES["es"])
