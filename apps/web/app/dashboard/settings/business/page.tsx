'use client';

import { useState, useEffect } from 'react';
import { Building2, ExternalLink } from 'lucide-react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Spinner } from '@/components/ui/spinner';
import { auth, TenantDTO } from '@/lib/api';
import { useTranslations } from '@/lib/i18n';

function InfoRow({ label, value }: { label: string; value: string | undefined }) {
  return (
    <div className="flex items-center justify-between py-2 text-sm">
      <span className="text-neutral-500">{label}</span>
      <span className="font-medium text-neutral-900">{value ?? '\u2014'}</span>
    </div>
  );
}

export default function BusinessPage() {
  const t = useTranslations();
  const [tenant, setTenant] = useState<TenantDTO | null>(null);
  const [loading, setLoading] = useState(true);
  const businessTypes = t.settings.businessTypes as Record<string, string>;

  useEffect(() => {
    auth.me()
      .then(({ tenant: tn }) => setTenant(tn))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="flex justify-center py-16"><Spinner size="lg" /></div>;

  return (
    <Card>
      <CardHeader>
        <Building2 className="h-4 w-4 text-neutral-400" />
        <CardTitle>{t.settings.business.title}</CardTitle>
      </CardHeader>
      <div className="divide-y divide-neutral-100">
        <InfoRow label={t.settings.business.nameLabel}       value={tenant?.name} />
        <InfoRow label={t.settings.business.typeLabel}       value={businessTypes[tenant?.business_type ?? ''] ?? tenant?.business_type} />
        <InfoRow label={t.settings.business.bookingUrlLabel} value={`citaspot.com/book/${tenant?.slug ?? ''}`} />
      </div>
      {tenant?.slug && (
        <div className="mt-3 pt-3 border-t border-neutral-100">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => window.open(`/book/${tenant.slug}`, '_blank')}
          >
            <ExternalLink className="mr-1.5 h-3.5 w-3.5" />
            {t.settings.business.viewBookingPage}
          </Button>
        </div>
      )}
    </Card>
  );
}
