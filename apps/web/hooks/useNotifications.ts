'use client';

// Hook para gestionar el feed de notificaciones del sistema (Plane #41).
// Carga la primera página + el contador de no leídas al montar y expone acciones
// optimistas para `markRead`, `markAllRead` y `prepend` (usado por el stream SSE
// para inyectar `notification.created` sin refetch).

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  APIError,
  UserNotification,
  userNotificationsApi,
} from '@/lib/api';

interface UseNotificationsState {
  notifications: UserNotification[];
  unreadCount: number;
  loading: boolean;
  error: string | null;
  hasMore: boolean;
  /** true si el usuario carece de permisos (403) — el shell debe ocultar la campana. */
  forbidden: boolean;
}

export interface UseNotificationsResult extends UseNotificationsState {
  loadMore: () => Promise<void>;
  markRead: (id: string) => Promise<void>;
  markAllRead: () => Promise<void>;
  /** Inserta una notificación en la cabeza de la lista (no leídas). */
  prepend: (n: UserNotification) => void;
  /** Recarga la primera página (sin tocar el resto del estado mientras llega). */
  refresh: () => Promise<void>;
}

const DEFAULT_PAGE_SIZE = 20;

export function useNotifications(): UseNotificationsResult {
  const [notifications, setNotifications] = useState<UserNotification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [hasMore, setHasMore] = useState(false);
  const [forbidden, setForbidden] = useState(false);

  // El cursor para paginar es el `created_at` del último item devuelto.
  const nextCursorRef = useRef<string | null>(null);
  // Guardia anti-doble-fetch en loadMore.
  const loadingMoreRef = useRef(false);

  const loadInitial = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [page, count] = await Promise.all([
        userNotificationsApi.list({ limit: DEFAULT_PAGE_SIZE }),
        userNotificationsApi.unreadCount(),
      ]);
      setNotifications(page.items);
      nextCursorRef.current = page.nextCursor;
      setHasMore(page.nextCursor !== null);
      setUnreadCount(count);
      setForbidden(false);
    } catch (err) {
      if (err instanceof APIError && err.status === 403) {
        // Usuario sin permiso — modo "oculto" para el shell.
        setForbidden(true);
        setNotifications([]);
        setUnreadCount(0);
        setHasMore(false);
        nextCursorRef.current = null;
      } else {
        setError(err instanceof Error ? err.message : 'unknown_error');
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadInitial();
  }, [loadInitial]);

  const loadMore = useCallback(async () => {
    if (loadingMoreRef.current) return;
    if (!nextCursorRef.current) return;
    loadingMoreRef.current = true;
    try {
      const page = await userNotificationsApi.list({
        limit: DEFAULT_PAGE_SIZE,
        cursor: nextCursorRef.current,
      });
      setNotifications((prev) => {
        // Evitar duplicados si el cursor solapa (defensa en profundidad).
        const seen = new Set(prev.map((n) => n.id));
        const merged = [...prev];
        for (const n of page.items) {
          if (!seen.has(n.id)) merged.push(n);
        }
        return merged;
      });
      nextCursorRef.current = page.nextCursor;
      setHasMore(page.nextCursor !== null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'unknown_error');
    } finally {
      loadingMoreRef.current = false;
    }
  }, []);

  const markRead = useCallback(async (id: string) => {
    // Snapshot para rollback.
    const prevList = notifications;
    const prevCount = unreadCount;
    const target = prevList.find((n) => n.id === id);
    if (!target || target.read_at) return;

    const now = new Date().toISOString();
    setNotifications((prev) =>
      prev.map((n) => (n.id === id ? { ...n, read_at: now } : n)),
    );
    setUnreadCount((c) => Math.max(0, c - 1));

    try {
      await userNotificationsApi.markRead(id);
    } catch (err) {
      // Rollback si falla (excepto 404: backend lo trata como "ya leída").
      if (err instanceof APIError && err.status === 404) return;
      setNotifications(prevList);
      setUnreadCount(prevCount);
      throw err;
    }
  }, [notifications, unreadCount]);

  const markAllRead = useCallback(async () => {
    const prevList = notifications;
    const prevCount = unreadCount;
    const now = new Date().toISOString();
    setNotifications((prev) =>
      prev.map((n) => (n.read_at ? n : { ...n, read_at: now })),
    );
    setUnreadCount(0);
    try {
      await userNotificationsApi.markAllRead();
    } catch (err) {
      setNotifications(prevList);
      setUnreadCount(prevCount);
      throw err;
    }
  }, [notifications, unreadCount]);

  const prepend = useCallback((n: UserNotification) => {
    setNotifications((prev) => {
      // Dedupe por id (el SSE podría disparar dos veces en reconexiones).
      if (prev.some((x) => x.id === n.id)) return prev;
      return [n, ...prev];
    });
    if (n.read_at === null) {
      setUnreadCount((c) => c + 1);
    }
  }, []);

  return {
    notifications,
    unreadCount,
    loading,
    error,
    hasMore,
    forbidden,
    loadMore,
    markRead,
    markAllRead,
    prepend,
    refresh: loadInitial,
  };
}

export default useNotifications;
