'use client';

import { useState, useEffect, useCallback } from 'react';
import { Search, Users, Phone, Mail, MessageCircle } from 'lucide-react';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Spinner } from '@/components/ui/spinner';
import { customers, Customer } from '@/lib/api';
import { format } from 'date-fns';
import { useTranslations, useDateLocale } from '@/lib/i18n';

export default function ClientsPage() {
  const t          = useTranslations();
  const dateLocale = useDateLocale();
  const [list, setList]       = useState<Customer[]>([]);
  const [search, setSearch]   = useState('');
  const [loading, setLoading] = useState(true);

  const load = useCallback(async (q: string) => {
    setLoading(true);
    try {
      const res = await customers.list(q);
      setList(res.data ?? []);
    } catch {
      setList([]);
    } finally {
      setLoading(false);
    }
  }, []);

  // Búsqueda con debounce simple
  useEffect(() => {
    const timer = setTimeout(() => load(search), 300);
    return () => clearTimeout(timer);
  }, [search, load]);

  return (
    <div className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-neutral-900">{t.clients.title}</h1>
          <p className="mt-0.5 text-sm text-neutral-500">{t.clients.description}</p>
        </div>
        <div className="flex items-center gap-2 text-sm text-neutral-500">
          <Users className="h-4 w-4" />
          <span>{t.clients.clientsCount.replace('{n}', String(list.length))}</span>
        </div>
      </div>

      {/* Buscador */}
      <div className="relative mb-4 max-w-sm">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-neutral-400" />
        <input
          type="text"
          placeholder={t.clients.searchPlaceholder}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full rounded-lg border border-neutral-200 bg-white py-2 pl-9 pr-3 text-sm text-neutral-900 placeholder:text-neutral-400 focus:border-primary-400 focus:outline-none focus:ring-2 focus:ring-primary-100"
        />
      </div>

      {loading ? (
        <div className="flex justify-center py-16"><Spinner size="lg" /></div>
      ) : list.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <Users className="mb-3 h-12 w-12 text-neutral-200" />
          <p className="text-sm font-medium text-neutral-500">
            {search ? t.clients.noClientsFound : t.clients.noClientsYet}
          </p>
          <p className="mt-1 text-xs text-neutral-400">
            {search ? t.clients.noClientsFoundDesc : t.clients.noClientsYetDesc}
          </p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-xl border border-neutral-200 bg-white">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-neutral-100 bg-neutral-50 text-left text-xs font-medium uppercase tracking-wide text-neutral-500">
                <th className="px-4 py-3">{t.clients.colName}</th>
                <th className="px-4 py-3">{t.clients.colContact}</th>
                <th className="px-4 py-3 text-center">{t.clients.colVisits}</th>
                <th className="px-4 py-3 text-center">{t.clients.colWhatsApp}</th>
                <th className="px-4 py-3">{t.clients.colRegistration}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-100">
              {list.map((c) => (
                <tr key={c.id} className="hover:bg-neutral-50 transition-colors">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-3">
                      <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-primary-100 text-xs font-semibold text-primary-700">
                        {c.name.charAt(0).toUpperCase()}
                      </div>
                      <span className="font-medium text-neutral-900">{c.name}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-neutral-600">
                    <div className="flex flex-col gap-0.5">
                      {c.phone && (
                        <span className="flex items-center gap-1.5">
                          <Phone className="h-3 w-3 text-neutral-400" />
                          {c.phone}
                        </span>
                      )}
                      {c.email && (
                        <span className="flex items-center gap-1.5">
                          <Mail className="h-3 w-3 text-neutral-400" />
                          {c.email}
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-center">
                    <Badge variant={c.total_visits > 0 ? 'primary' : 'default'}>
                      {c.total_visits}
                    </Badge>
                  </td>
                  <td className="px-4 py-3 text-center">
                    {c.wa_opt_in ? (
                      <MessageCircle className="mx-auto h-4 w-4 text-emerald-500" />
                    ) : (
                      <span className="text-neutral-300">—</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-neutral-500">
                    {format(new Date(c.created_at), 'd MMM yyyy', { locale: dateLocale })}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
