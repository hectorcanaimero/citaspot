import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';

import { SoundToggle } from '@/components/dashboard/SoundToggle';
import { useNotifications } from '@/store/notifications';

const STORAGE_KEY = 'citaspot:sound_enabled';

// ── Helpers ───────────────────────────────────────────────────────────────────

beforeEach(() => {
  // Reset store y localStorage entre tests para evitar contaminación.
  useNotifications.setState({ soundEnabled: true });
  localStorage.clear();
});

// ── Tests ─────────────────────────────────────────────────────────────────────

describe('SoundToggle', () => {
  it('renderiza el icono Bell cuando soundEnabled es true', () => {
    useNotifications.setState({ soundEnabled: true });
    const { container } = render(<SoundToggle />);
    // lucide-react inyecta `lucide-bell` (no `lucide-bell-off`) en el className
    const svg = container.querySelector('svg');
    expect(svg?.getAttribute('class')).toMatch(/lucide-bell(?!-off)/);
  });

  it('renderiza el icono BellOff cuando soundEnabled es false', () => {
    useNotifications.setState({ soundEnabled: false });
    const { container } = render(<SoundToggle />);
    const svg = container.querySelector('svg');
    expect(svg?.getAttribute('class')).toMatch(/lucide-bell-off/);
  });

  it('al hacer click invierte el estado en el store', () => {
    useNotifications.setState({ soundEnabled: true });
    render(<SoundToggle />);
    const btn = screen.getByRole('button');
    fireEvent.click(btn);
    expect(useNotifications.getState().soundEnabled).toBe(false);
    fireEvent.click(btn);
    expect(useNotifications.getState().soundEnabled).toBe(true);
  });

  it('expone aria-label y aria-pressed correctos cuando está activado', () => {
    useNotifications.setState({ soundEnabled: true });
    render(<SoundToggle />);
    const btn = screen.getByRole('button');
    expect(btn.getAttribute('aria-pressed')).toBe('true');
    // Default i18n es español
    expect(btn.getAttribute('aria-label')).toBe('Notificaciones activadas');
  });

  it('expone aria-label y aria-pressed correctos cuando está silenciado', () => {
    useNotifications.setState({ soundEnabled: false });
    render(<SoundToggle />);
    const btn = screen.getByRole('button');
    expect(btn.getAttribute('aria-pressed')).toBe('false');
    expect(btn.getAttribute('aria-label')).toBe('Notificaciones silenciadas');
  });

  it('persiste el estado en localStorage para sobrevivir un reload', () => {
    useNotifications.setState({ soundEnabled: true });
    render(<SoundToggle />);
    fireEvent.click(screen.getByRole('button'));
    // zustand/middleware/persist guarda en localStorage de forma síncrona
    const raw = localStorage.getItem(STORAGE_KEY);
    expect(raw).toBeTruthy();
    const parsed = JSON.parse(raw as string);
    expect(parsed.state.soundEnabled).toBe(false);
  });
});
