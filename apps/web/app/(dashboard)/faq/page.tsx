'use client';

import { useState } from 'react';
import {
  ChevronDown,
  Rocket,
  CalendarDays,
  MessageCircle,
  BookOpen,
  Users,
  CreditCard,
  Wrench,
  Search,
} from 'lucide-react';
import { useTranslations } from '@/lib/i18n';

// Iconos por sección (en el mismo orden que t.faq.sections)
const SECTION_ICONS = [Rocket, CalendarDays, MessageCircle, BookOpen, Users, CreditCard, Wrench];
const SECTION_COLORS = [
  'text-violet-600 bg-violet-50',
  'text-blue-600 bg-blue-50',
  'text-emerald-600 bg-emerald-50',
  'text-orange-600 bg-orange-50',
  'text-pink-600 bg-pink-50',
  'text-indigo-600 bg-indigo-50',
  'text-neutral-600 bg-neutral-100',
];

interface FAQItem { q: string; a: string; }

function FAQAccordion({ items }: { items: FAQItem[] }) {
  const [open, setOpen] = useState<number | null>(null);
  return (
    <div className="divide-y divide-neutral-100">
      {items.map((item, i) => (
        <div key={i}>
          <button
            onClick={() => setOpen(open === i ? null : i)}
            className="flex w-full items-start justify-between gap-4 py-4 text-left"
          >
            <span className="text-sm font-medium text-neutral-800 leading-snug">{item.q}</span>
            <ChevronDown
              className={`mt-0.5 h-4 w-4 flex-shrink-0 text-neutral-400 transition-transform duration-200 ${open === i ? 'rotate-180' : ''}`}
            />
          </button>
          {open === i && (
            <div className="pb-4 text-sm text-neutral-600 leading-relaxed">
              <p>{item.a}</p>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

export default function FAQPage() {
  const t = useTranslations();
  const [search, setSearch]             = useState('');
  const [activeSection, setActiveSection] = useState<string | null>(null);

  const query = search.toLowerCase().trim();

  // Combinar secciones de traducción con iconos y colores
  const sections = t.faq.sections.map((s, i) => ({
    ...s,
    icon:  SECTION_ICONS[i] ?? Wrench,
    color: SECTION_COLORS[i] ?? 'text-neutral-600 bg-neutral-100',
  }));

  const filtered = sections.map((section) => ({
    ...section,
    items: section.items.filter(
      (item) =>
        item.q.toLowerCase().includes(query) ||
        item.a.toLowerCase().includes(query)
    ),
  })).filter((s) => s.items.length > 0);

  const displaySections = query
    ? filtered
    : activeSection
    ? sections.filter((s) => s.label === activeSection)
    : sections;

  return (
    <div className="mx-auto max-w-3xl px-4 py-8">
      {/* Header */}
      <div className="mb-8 text-center">
        <h1 className="text-2xl font-bold text-neutral-900">{t.faq.title}</h1>
        <p className="mt-2 text-sm text-neutral-500">{t.faq.subtitle}</p>
      </div>

      {/* Buscador */}
      <div className="relative mb-6">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
        <input
          type="text"
          placeholder={t.faq.searchPlaceholder}
          value={search}
          onChange={(e) => { setSearch(e.target.value); setActiveSection(null); }}
          className="w-full rounded-xl border border-neutral-200 bg-white py-3 pl-10 pr-4 text-sm text-neutral-800 placeholder-neutral-400 shadow-sm focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
        />
      </div>

      {/* Filtros por sección */}
      {!query && (
        <div className="mb-6 flex flex-wrap gap-2">
          <button
            onClick={() => setActiveSection(null)}
            className={`rounded-full px-3 py-1.5 text-xs font-medium transition-colors ${
              activeSection === null
                ? 'bg-primary-600 text-white'
                : 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200'
            }`}
          >
            {t.faq.all}
          </button>
          {sections.map((s) => {
            const Icon = s.icon;
            return (
              <button
                key={s.label}
                onClick={() => setActiveSection(activeSection === s.label ? null : s.label)}
                className={`flex items-center gap-1.5 rounded-full px-3 py-1.5 text-xs font-medium transition-colors ${
                  activeSection === s.label
                    ? 'bg-primary-600 text-white'
                    : 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200'
                }`}
              >
                <Icon className="h-3 w-3" />
                {s.label}
              </button>
            );
          })}
        </div>
      )}

      {/* Sin resultados */}
      {query && filtered.length === 0 && (
        <div className="rounded-xl border border-neutral-100 bg-neutral-50 px-6 py-12 text-center">
          <p className="text-sm text-neutral-500">
            {t.faq.noResultsTitle.replace('{query}', search)}
          </p>
          <p className="mt-1 text-xs text-neutral-400">{t.faq.noResultsDesc}</p>
        </div>
      )}

      {/* Secciones */}
      <div className="space-y-4">
        {displaySections.map((section) => {
          const Icon = section.icon;
          return (
            <div key={section.label} className="overflow-hidden rounded-xl border border-neutral-100 bg-white shadow-sm">
              <div className="flex items-center gap-3 border-b border-neutral-50 px-5 py-4">
                <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${section.color}`}>
                  <Icon className="h-4 w-4" />
                </div>
                <h2 className="text-sm font-semibold text-neutral-800">{section.label}</h2>
                <span className="ml-auto rounded-full bg-neutral-100 px-2 py-0.5 text-xs text-neutral-500">
                  {t.faq.questionsCount.replace('{n}', String(section.items.length))}
                </span>
              </div>
              <div className="px-5">
                <FAQAccordion items={section.items as FAQItem[]} />
              </div>
            </div>
          );
        })}
      </div>

      {/* Footer */}
      <div className="mt-8 rounded-xl border border-primary-100 bg-primary-50 px-6 py-5 text-center">
        <p className="text-sm font-medium text-primary-800">{t.faq.contactTitle}</p>
        <p className="mt-1 text-xs text-primary-600">{t.faq.contactDesc}</p>
      </div>
    </div>
  );
}
