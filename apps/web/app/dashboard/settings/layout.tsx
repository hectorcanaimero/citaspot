'use client';

import { usePathname } from 'next/navigation';
import Link from 'next/link';
import { Building2, User, CreditCard, Palette } from 'lucide-react';
import { useTranslations } from '@/lib/i18n';

const NAV_ITEMS = [
  { href: '/dashboard/settings/business', icon: Building2, labelKey: 'business' },
  { href: '/dashboard/settings/account', icon: User, labelKey: 'account' },
  { href: '/dashboard/settings/subscription', icon: CreditCard, labelKey: 'billing' },
  { href: '/dashboard/settings/customization', icon: Palette, labelKey: 'customization' },
];

export default function SettingsLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const t = useTranslations();

  const labels: Record<string, string> = {
    business: t.settings?.tabs?.business ?? 'Negocio',
    account: t.settings?.tabs?.account ?? 'Cuenta',
    billing: t.settings?.tabs?.billing ?? 'Suscripci\u00f3n',
    customization: 'Personalizaci\u00f3n',
  };

  return (
    <div className="p-6">
      <div className="mb-6">
        <h1 className="text-xl font-semibold text-neutral-900">{t.settings.title}</h1>
        <p className="mt-0.5 text-sm text-neutral-500">{t.settings.description}</p>
      </div>

      {/* Mobile: horizontal scroll nav */}
      <nav className="mb-6 flex gap-1 overflow-x-auto rounded-lg bg-neutral-100 p-1 md:hidden">
        {NAV_ITEMS.map(({ href, icon: Icon, labelKey }) => {
          const active = pathname === href;
          return (
            <Link
              key={href}
              href={href}
              className={`flex items-center gap-1.5 whitespace-nowrap rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                active
                  ? 'bg-white text-neutral-900'
                  : 'text-neutral-500 hover:text-neutral-700'
              }`}
            >
              <Icon className="h-4 w-4" />
              {labels[labelKey]}
            </Link>
          );
        })}
      </nav>

      <div className="flex gap-6">
        {/* Desktop: vertical sidebar nav */}
        <nav className="hidden md:flex md:w-48 md:flex-shrink-0 md:flex-col md:gap-1">
          {NAV_ITEMS.map(({ href, icon: Icon, labelKey }) => {
            const active = pathname === href;
            return (
              <Link
                key={href}
                href={href}
                className={`flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-colors ${
                  active
                    ? 'bg-primary-50 text-primary-800'
                    : 'text-neutral-500 hover:text-neutral-700 hover:bg-neutral-100'
                }`}
              >
                <Icon className="h-4 w-4" />
                {labels[labelKey]}
              </Link>
            );
          })}
        </nav>

        {/* Content */}
        <div className="flex-1 max-w-2xl">
          {children}
        </div>
      </div>
    </div>
  );
}
