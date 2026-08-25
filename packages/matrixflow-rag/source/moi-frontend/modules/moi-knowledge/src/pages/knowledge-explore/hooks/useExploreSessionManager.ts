import { useCallback, useEffect, useMemo, useSyncExternalStore } from 'react';
import type { TFunction } from 'i18next';

import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import type { SessionRecord } from '../../../service/dialogSession';
import { getExploreSessionStore } from '../services/exploreSessionStore';
import { groupSessionsByMonth } from '../utils/session';

interface UseExploreSessionManagerOptions {
  http: AppHttpClient;
  locale: string;
  t: TFunction<'moi-knowledge'>;
  workspaceId?: string | null;
}

const exploreSessionStore = getExploreSessionStore();

export function useExploreSessionManager({ http, locale, t, workspaceId }: UseExploreSessionManagerOptions) {
  const sessionState = useSyncExternalStore(
    useCallback((listener: () => void) => exploreSessionStore.subscribe(listener), []),
    useCallback(() => exploreSessionStore.getState(), []),
    useCallback(() => exploreSessionStore.getState(), []),
  );

  useEffect(() => {
    if (!workspaceId) {
      return;
    }
    exploreSessionStore.bootstrap({ http, t }).catch((error) => {
      console.warn('[useExploreSessionManager] bootstrap store failed', { msg: (error as Error).message });
    });
    return () => {
      exploreSessionStore.cleanupTransientState();
    };
  }, [http, t, workspaceId]);

  // Refresh session list when the page becomes visible again (e.g. tab switch, window focus)
  // to avoid stale session lists after being away.
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        if (!workspaceId) {
          return;
        }
        exploreSessionStore.refresh({ http, t }).catch(() => {});
      }
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, [http, t, workspaceId]);

  const groupedSessions = useMemo(
    () => groupSessionsByMonth(sessionState.sessionList, locale, t),
    [locale, sessionState.sessionList, t],
  );

  const selectedSession = useMemo(() => {
    const allSessions: SessionRecord[] = [...sessionState.pinnedSessionList, ...sessionState.sessionList];
    return allSessions.find((session) => session.id === sessionState.selectedSessionId) || null;
  }, [sessionState.pinnedSessionList, sessionState.selectedSessionId, sessionState.sessionList]);

  const setSelectedSessionId = useCallback((sessionId: number | null) => {
    exploreSessionStore.setSelectedSessionId(sessionId);
  }, []);

  const releaseEmptySelection = useCallback(() => {
    exploreSessionStore.releaseEmptySelection();
  }, []);

  const handleSearch = useCallback(
    (keyword: string) => {
      exploreSessionStore.handleSearch(keyword, { http, t });
    },
    [http, t],
  );

  const loadMore = useCallback(async () => {
    await exploreSessionStore.loadMore({ http, t });
  }, [http, t]);

  const createNewSession = useCallback(
    async (defaultKnowledgeIds?: number[]) => {
      return await exploreSessionStore.createNewSession({ defaultKnowledgeIds }, { http, t });
    },
    [http, t],
  );

  const createFixedSession = useCallback(
    async (knowledgeId: number) => {
      return await exploreSessionStore.createFixedSession({ knowledgeId }, { http, t });
    },
    [http, t],
  );

  const renameSession = useCallback(
    async (sessionId: number, newTitle: string) => {
      return await exploreSessionStore.renameSession(sessionId, newTitle, { http, t });
    },
    [http, t],
  );

  const deleteSession = useCallback(
    async (sessionId: number) => {
      return await exploreSessionStore.deleteSession(sessionId, { http, t });
    },
    [http, t],
  );

  const syncSessionTitle = useCallback((sessionId: number, nextTitle: string) => {
    return exploreSessionStore.syncSessionTitle(sessionId, nextTitle);
  }, []);

  const togglePin = useCallback(
    async (session: SessionRecord) => {
      return await exploreSessionStore.togglePin(session, { http, t });
    },
    [http, t],
  );

  return {
    sessionList: sessionState.sessionList,
    pinnedSessionList: sessionState.pinnedSessionList,
    groupedSessions,
    selectedSessionId: sessionState.selectedSessionId,
    selectedSession,
    searchKeyword: sessionState.searchKeyword,
    isLoading: sessionState.isLoading,
    isPinnedLoading: sessionState.isPinnedLoading,
    isSearching: sessionState.isSearching,
    isActionLoading: sessionState.isActionLoading,
    isSessionCreating: sessionState.isSessionCreating,
    isLoadingMore: sessionState.isLoadingMore,
    hasMore: sessionState.hasMore,
    setSelectedSessionId,
    releaseEmptySelection,
    handleSearch,
    loadMore,
    createNewSession,
    createFixedSession,
    renameSession,
    deleteSession,
    syncSessionTitle,
    togglePin,
  };
}
