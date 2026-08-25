import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import type { TFunction } from 'i18next';
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import type { ExploreSessionStoreState } from '../services/exploreSessionStore';
import { useExploreSessionManager } from './useExploreSessionManager';

const mockSessionStore = vi.hoisted(() => {
  const state: ExploreSessionStoreState = {
    sessionList: [],
    pinnedSessionList: [],
    selectedSessionId: null,
    searchKeyword: '',
    isLoading: false,
    isPinnedLoading: false,
    isSearching: false,
    isActionLoading: false,
    isSessionCreating: false,
    isLoadingMore: false,
    currentPage: 1,
    hasMore: true,
  };
  const listeners = new Set<() => void>();
  const store = {
    bootstrap: vi.fn(async () => {}),
    cleanupTransientState: vi.fn(),
    createFixedSession: vi.fn(async () => null),
    createNewSession: vi.fn(async () => true),
    deleteSession: vi.fn(async () => true),
    getState: vi.fn(() => state),
    handleSearch: vi.fn(),
    loadMore: vi.fn(async () => {}),
    refresh: vi.fn(async () => {}),
    renameSession: vi.fn(async () => true),
    setSelectedSessionId: vi.fn(),
    subscribe: vi.fn((listener: () => void) => {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    }),
    syncSessionTitle: vi.fn(() => true),
    togglePin: vi.fn(async () => true),
  };

  return { listeners, state, store };
});

vi.mock('../services/exploreSessionStore', () => ({
  getExploreSessionStore: () => mockSessionStore.store,
}));

describe('useExploreSessionManager', () => {
  const http = {} as AppHttpClient;
  const t = ((key: string) => key) as TFunction<'moi-knowledge'>;
  const reactActGlobal = globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: unknown };
  let root: Root | null = null;
  let container: HTMLDivElement | null = null;
  let previousActEnvironment: unknown;

  beforeAll(() => {
    previousActEnvironment = reactActGlobal.IS_REACT_ACT_ENVIRONMENT;
    reactActGlobal.IS_REACT_ACT_ENVIRONMENT = true;
  });

  afterAll(() => {
    reactActGlobal.IS_REACT_ACT_ENVIRONMENT = previousActEnvironment;
  });

  beforeEach(() => {
    vi.clearAllMocks();
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => {
      root?.unmount();
    });
    container?.remove();
    root = null;
    container = null;
  });

  it('does not load session lists before workspace id is available', async () => {
    await renderSessionManager(null);

    expect(mockSessionStore.store.bootstrap).not.toHaveBeenCalled();
    expect(mockSessionStore.store.refresh).not.toHaveBeenCalled();
  });

  it('loads session lists after workspace id becomes available', async () => {
    await renderSessionManager(null);

    expect(mockSessionStore.store.bootstrap).not.toHaveBeenCalled();

    await renderSessionManager('ws-1');

    expect(mockSessionStore.store.bootstrap).toHaveBeenCalledTimes(1);
    expect(mockSessionStore.store.bootstrap).toHaveBeenCalledWith({ http, t });

    act(() => {
      root?.unmount();
    });

    expect(mockSessionStore.store.cleanupTransientState).toHaveBeenCalledTimes(1);
  });

  it('does not refresh on visibility change without workspace id', async () => {
    await renderSessionManager(null);

    await act(async () => {
      document.dispatchEvent(new Event('visibilitychange'));
    });

    expect(mockSessionStore.store.refresh).not.toHaveBeenCalled();
  });

  async function renderSessionManager(workspaceId: string | null): Promise<void> {
    await act(async () => {
      root?.render(createElement(SessionManagerProbe, { workspaceId }));
    });
  }

  function SessionManagerProbe({ workspaceId }: { workspaceId: string | null }) {
    useExploreSessionManager({ http, locale: 'zh-CN', t, workspaceId });
    return null;
  }
});
