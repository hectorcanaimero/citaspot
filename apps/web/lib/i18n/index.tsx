'use client';

// Sistema de internacionalización para CitaSpot.
// Soporta: es (español), en (inglés), pt (portugués brasileño).
// Prioridad: admin-set (localStorage) > idioma del navegador > 'es'

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
  const lang = navigator.language.split('-')[0].toLowerCase();
  if (lang === 'en') return 'en';
  if (lang === 'pt') return 'pt';
  return 'es';
}

interface LanguageContextValue {
  language: Language;
  setLanguage: (lang: Language) => void;
  t: Translations;
}

const LanguageContext = createContext<LanguageContextValue>({
  language: 'es',
  setLanguage: () => {},
  t: esTranslations,
});

export function LanguageProvider({ children }: { children: ReactNode }) {
  const [language, setLanguageState] = useState<Language>('es');

  // Detectar idioma en el cliente (evita hydration mismatch)
  useEffect(() => {
    const detected = detectLanguage();
    setLanguageState(detected);
  }, []);

  // Actualizar el atributo lang del documento cuando cambia el idioma
  useEffect(() => {
    document.documentElement.lang = language;
  }, [language]);

  function setLanguage(lang: Language) {
    localStorage.setItem(STORAGE_KEY, lang);
    setLanguageState(lang);
  }

  return (
    <LanguageContext.Provider
      value={{ language, setLanguage, t: TRANSLATIONS[language] }}
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
  const { language, setLanguage } = useContext(LanguageContext);
  return { language, setLanguage };
}

/** Retorna el locale de date-fns para el idioma activo. */
export function useDateLocale() {
  const { language } = useContext(LanguageContext);
  return DATE_LOCALES[language];
}
