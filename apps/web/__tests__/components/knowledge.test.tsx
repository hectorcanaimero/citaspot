import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/react';
import React from 'react';

// vi.hoisted() se ejecuta antes de que vi.mock haga el hoisting
const mockRedirect = vi.hoisted(() => vi.fn());

vi.mock('next/navigation', () => ({
  redirect: mockRedirect,
  usePathname: vi.fn(() => '/dashboard/knowledge'),
  useRouter: vi.fn(() => ({ push: vi.fn(), refresh: vi.fn() })),
}));

vi.mock('@/lib/supabase/browser', () => ({
  createClient: vi.fn(() => ({
    auth: { getSession: vi.fn(), signOut: vi.fn() },
  })),
}));

import KnowledgePage from '@/app/dashboard/knowledge/page';

describe('KnowledgePage', () => {
  it('redirects to /dashboard/chatbot', () => {
    // redirect() throws in Next.js to halt rendering, mock just records the call
    try {
      render(<KnowledgePage />);
    } catch {
      // redirect may throw — that's expected
    }
    expect(mockRedirect).toHaveBeenCalledWith('/dashboard/chatbot');
  });
});
