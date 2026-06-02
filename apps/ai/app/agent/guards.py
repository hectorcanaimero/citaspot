# Guardrails Layer 1: detectores regex que corren antes de intent.detect().
from __future__ import annotations

import logging
import re
from dataclasses import dataclass
from enum import Enum

log = logging.getLogger(__name__)


class GuardKind(str, Enum):
    PROMPT_INJECTION = "PROMPT_INJECTION"
    OFFENSIVE = "OFFENSIVE"
    OFF_TOPIC = "OFF_TOPIC"


@dataclass(frozen=True)
class GuardHit:
    kind: GuardKind
    pattern: str  # nombre del patrón que disparó (no el texto del usuario)


# ---------------------------------------------------------------------------
# Prompt injection: intentos de override del system prompt o jailbreak.
# Conservador: solo expresiones inequívocas que un cliente legítimo nunca usaría
# en el contexto de agendar una cita.
# ---------------------------------------------------------------------------
_INJECTION_PATTERNS: list[tuple[str, re.Pattern[str]]] = [
    ("ignore_instructions", re.compile(
        r"\bignor[ae]\s+(?:tus|las|estas|todas|las\s+anteriores)\s+"
        r"(?:instrucciones|reglas|órdenes|directivas)",
        re.IGNORECASE | re.UNICODE,
    )),
    ("forget_instructions", re.compile(
        r"\bolvid[ae]\s+(?:lo\s+anterior|todo|tus?|las?|tu\s+rol)",
        re.IGNORECASE | re.UNICODE,
    )),
    ("act_as", re.compile(
        r"\b(?:act[uú][ae]|comp[oó]rtate)\s+como\s+(?:un|una|si)",
        re.IGNORECASE | re.UNICODE,
    )),
    ("pretend_to_be", re.compile(
        r"\b(?:pret[eé]nd[ae]|finge|simula)\s+(?:ser|que\s+eres)",
        re.IGNORECASE | re.UNICODE,
    )),
    ("you_are_now", re.compile(
        r"\b(?:ahora\s+eres|eres\s+ahora|de\s+ahora\s+en\s+adelante\s+eres)\b",
        re.IGNORECASE | re.UNICODE,
    )),
    ("system_prompt_leak", re.compile(
        r"\b(?:system\s+prompt|prompt\s+del?\s+sistema|tu\s+prompt|"
        r"reveal\s+(?:your|the)\s+prompt|muéstrame?\s+tu\s+prompt)\b",
        re.IGNORECASE | re.UNICODE,
    )),
    ("jailbreak", re.compile(
        r"\b(?:jailbreak|DAN\s+mode|modo\s+DAN|developer\s+mode|"
        r"sin\s+restricciones|sin\s+filtros|sin\s+censura)\b",
        re.IGNORECASE | re.UNICODE,
    )),
    ("english_override", re.compile(
        r"\bignore\s+(?:all\s+)?(?:previous|prior|above)\s+(?:instructions|prompts|rules)\b",
        re.IGNORECASE | re.UNICODE,
    )),
]


def detect_prompt_injection(text: str) -> GuardHit | None:
    """Detecta intentos de override del system prompt. Retorna None si no hay match."""
    if not text:
        return None
    for name, pat in _INJECTION_PATTERNS:
        if pat.search(text):
            log.warning("guards.injection: hit pattern=%s", name)
            return GuardHit(kind=GuardKind.PROMPT_INJECTION, pattern=name)
    return None


# ---------------------------------------------------------------------------
# Contenido ofensivo: lista conservadora de insultos directos al asistente
# o al negocio. NO incluye palabras ambiguas que aparecen en contexto legítimo.
# Match con word boundaries y minúsculas.
# ---------------------------------------------------------------------------
_OFFENSIVE_TOKENS = [
    # Insultos directos al destinatario (es)
    "imbécil", "imbecil", "estúpido", "estupido", "estúpida", "estupida",
    "idiota", "pendejo", "pendeja", "gilipollas", "boludo", "boluda",
    "huevón", "huevon", "huevona", "pelotudo", "pelotuda",
    # Sexual/agresivo directo (es)
    "puto", "puta", "marica", "maricón", "maricon", "verga",
    # Inglés (común en spam internacional)
    "fuck you", "shut up", "asshole", "bitch",
]
_OFFENSIVE_PATTERN = re.compile(
    r"\b(?:" + "|".join(re.escape(t) for t in _OFFENSIVE_TOKENS) + r")\b",
    re.IGNORECASE | re.UNICODE,
)


def detect_offensive(text: str) -> GuardHit | None:
    """Detecta contenido ofensivo dirigido. Retorna None si no hay match."""
    if not text:
        return None
    m = _OFFENSIVE_PATTERN.search(text)
    if m:
        log.warning("guards.offensive: hit token=%s", m.group(0).lower())
        return GuardHit(kind=GuardKind.OFFENSIVE, pattern=m.group(0).lower())
    return None


# ---------------------------------------------------------------------------
# Off-topic: tópicos claramente fuera del scope de agenda/citas.
# Conservador: solo patrones inequívocos. El LLM se encarga del resto.
# ---------------------------------------------------------------------------
_OFF_TOPIC_PATTERNS: list[tuple[str, re.Pattern[str]]] = [
    ("jokes", re.compile(
        r"\b(?:cu[eé]ntame?\s+un\s+chiste|dime\s+un\s+chiste|"
        r"tell\s+me\s+a\s+joke|cont[áa]me?\s+un\s+chiste|"
        r"me\s+conoc[eé]s\s+un\s+chiste)\b",
        re.IGNORECASE | re.UNICODE,
    )),
    ("sports", re.compile(
        r"\b(?:qui[eé]n\s+gan[oó]\s+el\s+(?:mundial|partido|cl[aá]sico)|"
        r"resultado\s+del?\s+partido|"
        r"who\s+won\s+the\s+(?:world\s+cup|game))\b",
        re.IGNORECASE | re.UNICODE,
    )),
    ("weather", re.compile(
        r"\b(?:qu[eé]\s+(?:tiempo|clima)\s+(?:hace|va\s+a\s+hacer)|"
        r"va\s+a\s+llover|what's\s+the\s+weather)\b",
        re.IGNORECASE | re.UNICODE,
    )),
    ("politics", re.compile(
        r"\b(?:qui[eé]n\s+es\s+el\s+presidente|opini[oó]n\s+pol[ií]tica|"
        r"who\s+is\s+the\s+president)\b",
        re.IGNORECASE | re.UNICODE,
    )),
    ("recipes", re.compile(
        r"\b(?:receta\s+de|c[oó]mo\s+hacer\s+(?:una\s+)?(?:torta|pizza|"
        r"pasta|comida)|recipe\s+for)\b",
        re.IGNORECASE | re.UNICODE,
    )),
    ("general_chat", re.compile(
        r"\b(?:c[oó]mo\s+est[aá]s|how\s+are\s+you|qu[eé]\s+haces|"
        r"qu[eé]\s+opin[aá]s\s+de\s+la\s+vida|cu[eé]ntame\s+de\s+ti)\b",
        re.IGNORECASE | re.UNICODE,
    )),
    ("generic_facts", re.compile(
        r"\b(?:cu[aá]ntos?\s+(?:planetas|continentes|pa[ií]ses)|"
        r"capital\s+de\s+\w+|how\s+many\s+(?:planets|countries))\b",
        re.IGNORECASE | re.UNICODE,
    )),
]


def detect_off_topic(text: str) -> GuardHit | None:
    """Detecta mensajes claramente fuera del scope. Retorna None si no hay match."""
    if not text:
        return None
    for name, pat in _OFF_TOPIC_PATTERNS:
        if pat.search(text):
            log.info("guards.off_topic: hit pattern=%s", name)
            return GuardHit(kind=GuardKind.OFF_TOPIC, pattern=name)
    return None


def run_all_guards(text: str) -> GuardHit | None:
    """
    Corre todos los guards en orden de severidad y retorna el primer hit.
    Orden: injection > offensive > off_topic (injection es lo más grave).
    """
    hit = detect_prompt_injection(text)
    if hit is not None:
        return hit
    hit = detect_offensive(text)
    if hit is not None:
        return hit
    hit = detect_off_topic(text)
    if hit is not None:
        return hit
    return None
