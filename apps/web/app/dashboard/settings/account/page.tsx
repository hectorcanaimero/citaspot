'use client';

import { useState, useEffect } from 'react';
import { User, Globe } from 'lucide-react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Spinner } from '@/components/ui/spinner';
import { auth, UserDTO } from '@/lib/api';
import { useTranslations, useLanguage } from '@/lib/i18n';

function InfoRow({ label, value }: { label: string; value: string | undefined }) {
  return (
    <div className="flex items-center justify-between py-2 text-sm">
      <span className="text-neutral-500">{label}</span>
      <span className="font-medium text-neutral-900">{value ?? '\u2014'}</span>
    </div>
  );
}

export default function AccountPage() {
  const t = useTranslations();
  const { language, setLanguage } = useLanguage();
  const [user, setUser] = useState<UserDTO | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    auth.me()
      .then(({ user: u }) => setUser(u))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="flex justify-center py-16"><Spinner size="lg" /></div>;

  return (
    <Card>
      <CardHeader>
        <User className="h-4 w-4 text-neutral-400" />
        <CardTitle>{t.settings.account.title}</CardTitle>
      </CardHeader>
      <div className="divide-y divide-neutral-100">
        <InfoRow label={t.settings.account.nameLabel}  value={user?.name} />
        <InfoRow label={t.settings.account.emailLabel} value={user?.email} />
        <InfoRow label={t.settings.account.roleLabel}  value={user?.role === 'owner' ? t.settings.account.ownerRole : user?.role} />
      </div>
      {/* Selector de idioma */}
      <div className="mt-4 pt-4 border-t border-neutral-100">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Globe className="h-4 w-4 text-neutral-400" />
            <span className="text-sm text-neutral-600">{t.settings.account.languageLabel}</span>
          </div>
          <div className="flex rounded-lg border border-neutral-200 overflow-hidden text-sm">
            {(['es', 'en', 'pt'] as const).map((lang) => (
              <button
                key={lang}
                onClick={() => setLanguage(lang)}
                className={`px-3 py-1.5 transition-colors ${
                  language === lang
                    ? 'bg-primary-600 text-white font-medium'
                    : 'text-neutral-600 hover:bg-neutral-50'
                }`}
              >
                {t.settings.language[lang]}
              </button>
            ))}
          </div>
        </div>
      </div>
    </Card>
  );
}
