'use client';

// Toggle de notificaciones sonoras para el dashboard.
// Estado persistido en zustand (`citaspot:sound_enabled`).
// Sin estado local: lee y muta directamente el store.

import { Bell, BellOff } from 'lucide-react';
import { useNotifications } from '@/store/notifications';
import { useTranslations } from '@/lib/i18n';

export function SoundToggle() {
  const t = useTranslations();
  const soundEnabled = useNotifications((s) => s.soundEnabled);
  const toggleSound = useNotifications((s) => s.toggleSound);

  const label = soundEnabled ? t.dashboard.soundOn : t.dashboard.soundOff;
  const Icon = soundEnabled ? Bell : BellOff;

  return (
    <button
      type="button"
      onClick={toggleSound}
      aria-label={label}
      aria-pressed={soundEnabled}
      title={label}
      className="flex h-9 w-9 items-center justify-center rounded-lg bg-white/10 text-white transition-colors hover:bg-white/20"
    >
      <Icon className="h-4 w-4" />
    </button>
  );
}

export default SoundToggle;
