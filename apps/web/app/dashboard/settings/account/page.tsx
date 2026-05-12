'use client';

import { useState, useEffect } from 'react';
import { User, Globe, Pencil, X, Check } from 'lucide-react';
import { Card, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
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
  const [editing, setEditing] = useState(false);
  const [nameValue, setNameValue] = useState('');
  const [saving, setSaving] = useState(false);
  const [feedback, setFeedback] = useState<{ ok: boolean; msg: string } | null>(null);

  useEffect(() => {
    auth.me()
      .then(({ user: u }) => {
        setUser(u);
        setNameValue(u.name ?? '');
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  function startEdit() {
    setNameValue(user?.name ?? '');
    setFeedback(null);
    setEditing(true);
  }

  function cancelEdit() {
    setEditing(false);
    setFeedback(null);
  }

  async function handleSave() {
    if (!nameValue.trim() || nameValue.trim().length < 2) return;
    setSaving(true);
    setFeedback(null);
    try {
      await auth.updateMyProfile({ name: nameValue.trim() });
      setUser(prev => prev ? { ...prev, name: nameValue.trim() } : prev);
      setEditing(false);
      setFeedback({ ok: true, msg: t.settings.account.saveSuccess });
    } catch {
      setFeedback({ ok: false, msg: t.settings.account.saveError });
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <div className="flex justify-center py-16"><Spinner size="lg" /></div>;

  return (
    <Card>
      <CardHeader>
        <User className="h-4 w-4 text-neutral-400" />
        <CardTitle>{t.settings.account.title}</CardTitle>
      </CardHeader>
      <div className="divide-y divide-neutral-100">
        {/* Nombre — editable */}
        <div className="flex items-center justify-between py-2 text-sm">
          <span className="text-neutral-500">{t.settings.account.nameLabel}</span>
          {editing ? (
            <div className="flex items-center gap-2">
              <Input
                value={nameValue}
                onChange={e => setNameValue(e.target.value)}
                placeholder={t.settings.account.namePlaceholder}
                className="h-7 text-sm w-48"
                autoFocus
                onKeyDown={e => {
                  if (e.key === 'Enter') handleSave();
                  if (e.key === 'Escape') cancelEdit();
                }}
              />
              <Button
                size="sm"
                variant="ghost"
                className="p-0 h-7 w-7"
                onClick={handleSave}
                disabled={saving || nameValue.trim().length < 2}
                aria-label="Guardar"
              >
                {saving ? <Spinner size="sm" /> : <Check className="h-3.5 w-3.5 text-green-600" />}
              </Button>
              <Button
                size="sm"
                variant="ghost"
                className="p-0 h-7 w-7"
                onClick={cancelEdit}
                disabled={saving}
                aria-label="Cancelar"
              >
                <X className="h-3.5 w-3.5 text-neutral-400" />
              </Button>
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <span className="font-medium text-neutral-900">{user?.name ?? '\u2014'}</span>
              <Button
                size="sm"
                variant="ghost"
                className="p-0 h-6 w-6"
                onClick={startEdit}
                aria-label={t.settings.account.editName}
              >
                <Pencil className="h-3 w-3 text-neutral-400" />
              </Button>
            </div>
          )}
        </div>

        {/* Email — read-only con nota */}
        <div className="py-2 text-sm">
          <div className="flex items-center justify-between">
            <span className="text-neutral-500">{t.settings.account.emailLabel}</span>
            <span className="font-medium text-neutral-900">{user?.email ?? '\u2014'}</span>
          </div>
          <p className="mt-0.5 text-xs text-neutral-400">{t.settings.account.emailReadonlyNote}</p>
        </div>

        <InfoRow
          label={t.settings.account.roleLabel}
          value={user?.role === 'owner' ? t.settings.account.ownerRole : user?.role}
        />
      </div>

      {feedback && (
        <p className={`mt-2 text-xs ${feedback.ok ? 'text-green-600' : 'text-red-500'}`}>
          {feedback.msg}
        </p>
      )}

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
