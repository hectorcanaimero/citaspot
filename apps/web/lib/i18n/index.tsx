'use client';

// Sistema de internacionalización para CitaSpot.
// Soporta: es (español), en (inglés), pt (portugués brasileño).
// Prioridad: admin-set (localStorage con override) > tenant.country (post-login) > 'es'

import { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import { es as dateFnsEs, enUS as dateFnsEn, ptBR as dateFnsPt } from 'date-fns/locale';
import { es as esTranslations } from './locales/es';
import { en as enTranslations } from './locales/en';
import { pt as ptTranslations } from './locales/pt';

export type Language = 'es' | 'en' | 'pt';

// Convierte todos los valores hoja a string para que los distintos idiomas sean asignables.
type StringValues<T> = {
  readonly [K in keyof T]: T[K] extends object ? StringValues<T[K]> : string;
};
export type Translations = StringValues<typeof esTranslations>;

const STORAGE_KEY = 'citaspot_language';
const STORAGE_KEY_OVERRIDDEN = 'citaspot_language_overridden';

const TRANSLATIONS: Record<Language, Translations> = {
  es: esTranslations,
  en: enTranslations,
  pt: ptTranslations,
};

const DATE_LOCALES = {
  es: dateFnsEs,
  en: dateFnsEn,
  pt: dateFnsPt,
};

function detectLanguage(): Language {
  if (typeof window === 'undefined') return 'es';
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === 'en' || stored === 'es' || stored === 'pt') return stored;
  return 'es';
}

export function countryToLanguage(country?: string | null): Language {
  if (!country) return 'es';
  const c = country.toUpperCase();
  if (c === 'BR') return 'pt';
  if (c === 'US' || c === 'GB' || c === 'CA') return 'en';
  return 'es';
}

interface LanguageContextValue {
  language: Language;
  setLanguage: (lang: Language) => void;
  setLanguageFromCountry: (country?: string | null) => void;
  t: Translations;
}

const LanguageContext = createContext<LanguageContextValue>({
  language: 'es',
  setLanguage: () => {},
  setLanguageFromCountry: () => {},
  t: esTranslations,
});

export function LanguageProvider({ children }: { children: ReactNode }) {
  const [language, setLanguageState] = useState<Language>('es');

  // Detectar idioma en el cliente (evita hydration mismatch)
  useEffect(() => {
    const detected = detectLanguage();
    setLanguageState(detected);
  }, []);

  // Migración one-shot: usuarios pre-existentes con idioma guardado se marcan como overridden
  // para no pisar su elección al hidratar tenant.country.
  useEffect(() => {
    if (typeof window === 'undefined') return;
    if (localStorage.getItem(STORAGE_KEY) && !localStorage.getItem(STORAGE_KEY_OVERRIDDEN)) {
      localStorage.setItem(STORAGE_KEY_OVERRIDDEN, 'true');
    }
  }, []);

  // Actualizar el atributo lang del documento cuando cambia el idioma
  useEffect(() => {
    document.documentElement.lang = language;
  }, [language]);

  function setLanguage(lang: Language) {
    localStorage.setItem(STORAGE_KEY, lang);
    localStorage.setItem(STORAGE_KEY_OVERRIDDEN, 'true');
    setLanguageState(lang);
  }

  function setLanguageFromCountry(country?: string | null) {
    if (typeof window === 'undefined') return;
    if (localStorage.getItem(STORAGE_KEY_OVERRIDDEN) === 'true') return;
    const lang = countryToLanguage(country);
    localStorage.setItem(STORAGE_KEY, lang);
    setLanguageState(lang);
  }

  return (
    <LanguageContext.Provider
      value={{ language, setLanguage, setLanguageFromCountry, t: TRANSLATIONS[language] }}
    >
      {children}
    </LanguageContext.Provider>
  );
}

/** Retorna el objeto de traducciones para el idioma activo. */
export function useTranslations(): Translations {
  return useContext(LanguageContext).t;
}

/** Retorna el idioma activo y la función para cambiarlo. */
export function useLanguage() {
  const { language, setLanguage, setLanguageFromCountry } = useContext(LanguageContext);
  return { language, setLanguage, setLanguageFromCountry };
}

/** Retorna el locale de date-fns para el idioma activo. */
export function useDateLocale() {
  const { language } = useContext(LanguageContext);
  return DATE_LOCALES[language];
}
