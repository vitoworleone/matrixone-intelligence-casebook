import type { TFunction } from 'i18next';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import type { SessionListResponse, SessionRecord } from '../../../service/dialogSession';

const mocks = vi.hoisted(() => ({
  createSession: vi.fn(),
  clearSessionRuntime: vi.fn(),
  deleteSession: vi.fn(),
  ensureSessionRuntime: vi.fn(),
  getPinnedSessions: vi.fn(),
  getSessionList: vi.fn(),
  getUnpinnedSessions: vi.fn(),
  messageError: vi.fn(),
  messageSuccess: vi.fn(),
  pinSession: vi.fn(),
  syncSessionDefaultKnowledge: vi.fn(),
}));

vi.mock('antd', () => ({
  message: {
    error: mocks.messageError,
    success: mocks.messageSuccess,
  },
}));

vi.mock('../../../service/dialogSession', () => ({
  createSession: mocks.createSession,
  deleteSession: mocks.deleteSession,
  getPinnedSessions: mocks.getPinnedSessions,
  getSessionList: mocks.getSessionList,
  getUnpinnedSessions: mocks.getUnpinnedSessions,
  isSessionPinned: () => false,
  pinSession: mocks.pinSession,
  unpinSession: vi.fn(),
  updateSession: vi.fn(),
}));

vi.mock('./exploreA2ARuntimeStore', () => ({
  getExploreA2ARuntimeStore: () => ({
    clearSessionRuntime: mocks.clearSessionRuntime,
    ensureSessionRuntime: mocks.ensureSessionRuntime,
    syncSessionDefaultKnowledge: mocks.syncSessionDefaultKnowledge,
  }),
}));

describe('ExploreSessionStore', () => {
  const http = {} as AppHttpClient;
  const t = ((key: string) => key) as TFunction<'moi-knowledge'>;
  const context = { http, t };

  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    vi.spyOn(console, 'warn').mockImplementation(() => {});
    mocks.getPinnedSessions.mockResolvedValue({ sessions: [] });
    mocks.getSessionList.mockResolvedValue(sessionList([]));
    mocks.getUnpinnedSessions.mockResolvedValue(sessionList([]));
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('keeps the knowledge-card virtual session selected when an old list response arrives late', async () => {
    const store = await createStore();
    const oldSession = session(1, 'old session');
    const unpinnedRequest = createDeferred<SessionListResponse>();
    const pinnedRequest = createDeferred<{ sessions: SessionRecord[] }>();
    mocks.getUnpinnedSessions.mockImplementationOnce(() => unpinnedRequest.promise);
    mocks.getPinnedSessions.mockImplementationOnce(() => pinnedRequest.promise);

    const refreshPromise = store.refresh(context);
    store.setSelectedSessionId(null);

    unpinnedRequest.resolve(sessionList([oldSession]));
    pinnedRequest.resolve({ sessions: [] });
    await refreshPromise;

    expect(store.getState().selectedSessionId).toBeNull();
    expect(store.getState().sessionList).toContainEqual(oldSession);
  });

  it('clears the old session during a slow creation, rejects duplicates, and retains a late history response', async () => {
    const store = await createStore();
    const oldSession = session(1, 'old session');
    const newSession = session(2, 'new session');
    mocks.getUnpinnedSessions.mockResolvedValueOnce(sessionList([oldSession]));
    await store.refresh(context);

    const unpinnedRequest = createDeferred<SessionListResponse>();
    const pinnedRequest = createDeferred<{ sessions: SessionRecord[] }>();
    mocks.getUnpinnedSessions.mockImplementationOnce(() => unpinnedRequest.promise);
    mocks.getPinnedSessions.mockImplementationOnce(() => pinnedRequest.promise);
    const creation = createDeferred<SessionRecord>();
    mocks.createSession.mockImplementationOnce(() => creation.promise);
    const refreshPromise = store.refresh(context);
    const firstCreation = store.createNewSession({}, context);

    expect(store.getState()).toMatchObject({
      isActionLoading: true,
      isSessionCreating: true,
      selectedSessionId: null,
    });
    await expect(store.createNewSession({}, context)).resolves.toBe(false);
    expect(mocks.createSession).toHaveBeenCalledTimes(1);

    creation.resolve(newSession);
    await expect(firstCreation).resolves.toBe(true);

    unpinnedRequest.resolve(sessionList([oldSession]));
    pinnedRequest.resolve({ sessions: [] });
    await refreshPromise;

    expect(store.getState()).toMatchObject({
      isActionLoading: false,
      isSessionCreating: false,
      selectedSessionId: newSession.id,
    });
    expect(store.getState().sessionList).toContainEqual(newSession);
    expect(store.getState().sessionList).toContainEqual(oldSession);
  });

  it('cancels a queued search when a new session starts', async () => {
    const store = await createStore();
    const oldSession = session(1, 'old session');
    const newSession = session(2, 'new session');
    mocks.getUnpinnedSessions.mockResolvedValueOnce(sessionList([oldSession]));
    mocks.createSession.mockResolvedValueOnce(newSession);
    await store.refresh(context);

    vi.useFakeTimers();
    try {
      store.handleSearch('old', context);
      await expect(store.createNewSession({}, context)).resolves.toBe(true);
      await vi.advanceTimersByTimeAsync(300);

      expect(mocks.getSessionList).not.toHaveBeenCalled();
      expect(store.getState()).toMatchObject({
        searchKeyword: '',
        selectedSessionId: newSession.id,
      });
      expect(mocks.getUnpinnedSessions).toHaveBeenCalledTimes(2);
      expect(store.getState().sessionList).toContainEqual(newSession);
      expect(store.getState().hasMore).toBe(false);
    } finally {
      vi.useRealTimers();
    }
  });

  it('clears an applied search before fixed session creation and reloads the full list', async () => {
    const store = await createStore();
    const allSession = session(1, 'all session');
    const matchingSession = session(2, 'matching session');
    const fixedSession = session(3, 'fixed session');
    mocks.getUnpinnedSessions.mockResolvedValueOnce(sessionList([allSession, matchingSession]));
    mocks.getUnpinnedSessions.mockResolvedValueOnce(sessionList([allSession, matchingSession]));
    mocks.getSessionList.mockResolvedValueOnce(sessionList([matchingSession]));
    mocks.createSession.mockResolvedValueOnce(fixedSession);
    await store.refresh(context);

    vi.useFakeTimers();
    try {
      store.handleSearch('matching', context);
      await vi.advanceTimersByTimeAsync(300);
      expect(store.getState().sessionList).toEqual([matchingSession]);

      await expect(store.createFixedSession({ knowledgeId: 7 }, context)).resolves.toEqual(fixedSession);
      await Promise.resolve();

      expect(store.getState()).toMatchObject({
        searchKeyword: '',
        hasMore: false,
      });
      expect(store.getState().sessionList).toEqual(expect.arrayContaining([allSession, matchingSession, fixedSession]));
    } finally {
      vi.useRealTimers();
    }
  });

  it('does not let the refresh after search cancellation overwrite a later search', async () => {
    const store = await createStore();
    const oldSession = session(1, 'old session');
    const newSession = session(2, 'new session');
    const matchingSession = session(3, 'matching session');
    const refreshRequest = createDeferred<SessionListResponse>();
    mocks.getUnpinnedSessions.mockResolvedValueOnce(sessionList([oldSession]));
    mocks.getUnpinnedSessions.mockImplementationOnce(() => refreshRequest.promise);
    mocks.getSessionList.mockResolvedValueOnce(sessionList([matchingSession]));
    mocks.createSession.mockResolvedValueOnce(newSession);
    await store.refresh(context);

    vi.useFakeTimers();
    try {
      store.handleSearch('old', context);
      await expect(store.createNewSession({}, context)).resolves.toBe(true);

      store.handleSearch('matching', context);
      await vi.advanceTimersByTimeAsync(300);
      expect(store.getState().sessionList).toEqual([matchingSession]);

      refreshRequest.resolve(sessionList([oldSession, newSession]));
      await Promise.resolve();
      await Promise.resolve();

      expect(store.getState()).toMatchObject({
        searchKeyword: 'matching',
        sessionList: [matchingSession],
      });
    } finally {
      vi.useRealTimers();
    }
  });

  it('does not start a new search while a session is being created', async () => {
    const store = await createStore();
    const newSession = session(2, 'new session');
    const creation = createDeferred<SessionRecord>();
    mocks.createSession.mockImplementationOnce(() => creation.promise);

    vi.useFakeTimers();
    try {
      const createPromise = store.createNewSession({}, context);
      store.handleSearch('old', context);
      await vi.advanceTimersByTimeAsync(300);

      expect(mocks.getSessionList).not.toHaveBeenCalled();
      expect(store.getState().searchKeyword).toBe('');

      creation.resolve(newSession);
      await expect(createPromise).resolves.toBe(true);
      expect(store.getState().selectedSessionId).toBe(newSession.id);
    } finally {
      vi.useRealTimers();
    }
  });

  it('ignores an in-flight search when session creation fails', async () => {
    const store = await createStore();
    const previousSession = session(1, 'previous session');
    const matchingSession = session(2, 'matching session');
    const searchRequest = createDeferred<SessionListResponse>();
    const creation = createDeferred<SessionRecord>();
    mocks.getUnpinnedSessions.mockResolvedValueOnce(sessionList([previousSession]));
    mocks.getUnpinnedSessions.mockResolvedValueOnce(sessionList([previousSession]));
    mocks.getSessionList.mockImplementationOnce(() => searchRequest.promise);
    mocks.createSession.mockImplementationOnce(() => creation.promise);
    await store.refresh(context);

    vi.useFakeTimers();
    try {
      store.handleSearch('matching', context);
      await vi.advanceTimersByTimeAsync(300);
      expect(mocks.getSessionList).toHaveBeenCalledTimes(1);

      const createPromise = store.createNewSession({}, context);
      expect(store.getState().isSearching).toBe(false);
      searchRequest.resolve(sessionList([matchingSession]));
      await Promise.resolve();
      await Promise.resolve();
      creation.reject(new Error('create failed'));

      await expect(createPromise).resolves.toBe(false);
      expect(store.getState().selectedSessionId).toBe(previousSession.id);
      expect(store.getState().sessionList).toContainEqual(previousSession);
    } finally {
      vi.useRealTimers();
    }
  });

  it('does not restore a deleted session from a pre-creation list response', async () => {
    const store = await createStore();
    const oldSession = session(1, 'old session');
    const newSession = session(2, 'new session');
    mocks.getUnpinnedSessions.mockResolvedValueOnce(sessionList([oldSession]));
    await store.refresh(context);

    const unpinnedRequest = createDeferred<SessionListResponse>();
    const pinnedRequest = createDeferred<{ sessions: SessionRecord[] }>();
    const creation = createDeferred<SessionRecord>();
    mocks.getUnpinnedSessions.mockImplementationOnce(() => unpinnedRequest.promise);
    mocks.getPinnedSessions.mockImplementationOnce(() => pinnedRequest.promise);
    mocks.createSession.mockImplementationOnce(() => creation.promise);
    mocks.deleteSession.mockResolvedValueOnce(undefined);

    const refreshPromise = store.refresh(context);
    const createPromise = store.createNewSession({}, context);
    creation.resolve(newSession);
    await expect(createPromise).resolves.toBe(true);
    await expect(store.deleteSession(oldSession.id, context)).resolves.toBe(true);

    unpinnedRequest.resolve(sessionList([oldSession]));
    pinnedRequest.resolve({ sessions: [] });
    await refreshPromise;

    expect(store.getState().sessionList).toEqual([newSession]);
  });

  it('keeps a locally pinned session out of a stale unpinned response after creation', async () => {
    const store = await createStore();
    const oldSession = session(1, 'old session');
    const newSession = session(2, 'new session');
    mocks.getUnpinnedSessions.mockResolvedValueOnce(sessionList([oldSession]));
    await store.refresh(context);

    const unpinnedRequest = createDeferred<SessionListResponse>();
    const pinnedRequest = createDeferred<{ sessions: SessionRecord[] }>();
    const creation = createDeferred<SessionRecord>();
    mocks.getUnpinnedSessions.mockImplementationOnce(() => unpinnedRequest.promise);
    mocks.getPinnedSessions.mockImplementationOnce(() => pinnedRequest.promise);
    mocks.createSession.mockImplementationOnce(() => creation.promise);
    mocks.pinSession.mockResolvedValueOnce(undefined);

    const refreshPromise = store.refresh(context);
    const createPromise = store.createNewSession({}, context);
    creation.resolve(newSession);
    await expect(createPromise).resolves.toBe(true);
    await expect(store.togglePin(oldSession, context)).resolves.toBe(true);

    unpinnedRequest.resolve(sessionList([oldSession]));
    pinnedRequest.resolve({ sessions: [] });
    await refreshPromise;

    expect(store.getState().sessionList).toEqual([newSession]);
    expect(store.getState().pinnedSessionList).toContainEqual(expect.objectContaining({ id: oldSession.id }));
  });

  it('keeps a newer virtual session selected when a normal creation completes', async () => {
    const store = await createStore();
    const oldSession = session(1, 'old session');
    const newSession = session(2, 'new session');
    const creation = createDeferred<SessionRecord>();
    mocks.getUnpinnedSessions.mockResolvedValue(sessionList([oldSession]));
    mocks.createSession.mockImplementationOnce(() => creation.promise);
    await store.refresh(context);

    const createPromise = store.createNewSession({}, context);
    store.setSelectedSessionId(null);
    creation.resolve(newSession);

    await expect(createPromise).resolves.toBe(true);

    expect(store.getState().selectedSessionId).toBeNull();
    expect(store.getState().sessionList).toContainEqual(newSession);
  });

  it('keeps a newer virtual session selected when a normal creation fails', async () => {
    const store = await createStore();
    const oldSession = session(1, 'old session');
    const creation = createDeferred<SessionRecord>();
    mocks.getUnpinnedSessions.mockResolvedValue(sessionList([oldSession]));
    mocks.createSession.mockImplementationOnce(() => creation.promise);
    await store.refresh(context);

    const createPromise = store.createNewSession({}, context);
    store.setSelectedSessionId(null);
    creation.reject(new Error('create failed'));

    await expect(createPromise).resolves.toBe(false);

    expect(store.getState().selectedSessionId).toBeNull();
  });

  it('restores the previous session after creation fails', async () => {
    const store = await createStore();
    const oldSession = session(1, 'old session');
    mocks.getUnpinnedSessions.mockResolvedValue(sessionList([oldSession]));
    mocks.createSession.mockRejectedValue(new Error('create failed'));
    await store.refresh(context);

    await expect(store.createNewSession({}, context)).resolves.toBe(false);

    expect(store.getState()).toMatchObject({
      isActionLoading: false,
      isSessionCreating: false,
      selectedSessionId: oldSession.id,
    });
    expect(store.getState().sessionList).toContainEqual(oldSession);
  });

  it('updates a late pinned list after the user selects another session', async () => {
    const store = await createStore();
    const firstSession = session(1, 'first session');
    const selectedSession = session(2, 'selected session');
    const pinnedSession = session(3, 'pinned session');
    const unpinnedRequest = createDeferred<SessionListResponse>();
    const pinnedRequest = createDeferred<{ sessions: SessionRecord[] }>();
    mocks.getUnpinnedSessions.mockImplementationOnce(() => unpinnedRequest.promise);
    mocks.getPinnedSessions.mockImplementationOnce(() => pinnedRequest.promise);

    const refreshPromise = store.refresh(context);
    unpinnedRequest.resolve(sessionList([firstSession, selectedSession]));
    await Promise.resolve();
    store.setSelectedSessionId(selectedSession.id);
    pinnedRequest.resolve({ sessions: [pinnedSession] });
    await refreshPromise;

    expect(store.getState().selectedSessionId).toBe(selectedSession.id);
    expect(store.getState().pinnedSessionList).toContainEqual(pinnedSession);
  });

  it('keeps a newer virtual session selected after fixed creation and a late list response', async () => {
    const store = await createStore();
    const oldSession = session(1, 'old session');
    const fixedSession = session(2, 'fixed session');
    const unpinnedRequest = createDeferred<SessionListResponse>();
    const pinnedRequest = createDeferred<{ sessions: SessionRecord[] }>();
    const creation = createDeferred<SessionRecord>();
    mocks.getUnpinnedSessions.mockImplementationOnce(() => unpinnedRequest.promise);
    mocks.getPinnedSessions.mockImplementationOnce(() => pinnedRequest.promise);
    mocks.createSession.mockImplementationOnce(() => creation.promise);
    store.setSelectedSessionId(null);

    const refreshPromise = store.refresh(context);
    const createPromise = store.createFixedSession({ knowledgeId: 7 }, context);
    creation.resolve(fixedSession);

    await expect(createPromise).resolves.toEqual(fixedSession);
    expect(store.getState().selectedSessionId).toBe(fixedSession.id);

    store.setSelectedSessionId(null);
    unpinnedRequest.resolve(sessionList([oldSession]));
    pinnedRequest.resolve({ sessions: [] });
    await refreshPromise;

    expect(store.getState().selectedSessionId).toBeNull();
    expect(store.getState().sessionList).toContainEqual(fixedSession);
    expect(store.getState().sessionList).toContainEqual(oldSession);
  });

  it('releases only the virtual empty selection when its page exits', async () => {
    const store = await createStore();
    const oldSession = session(1, 'old session');
    const manuallySelectedSession = session(2, 'manually selected session');
    mocks.getUnpinnedSessions.mockResolvedValue(sessionList([oldSession, manuallySelectedSession]));
    await store.refresh(context);

    store.setSelectedSessionId(null);
    store.releaseEmptySelection();
    await store.bootstrap(context);

    expect(store.getState().selectedSessionId).toBe(oldSession.id);

    store.setSelectedSessionId(null);
    store.setSelectedSessionId(manuallySelectedSession.id);
    store.releaseEmptySelection();
    await store.bootstrap(context);

    expect(store.getState().selectedSessionId).toBe(manuallySelectedSession.id);
  });
});

async function createStore() {
  const { getExploreSessionStore } = await import('./exploreSessionStore');
  return getExploreSessionStore();
}

function session(id: number, title: string): SessionRecord {
  return {
    id,
    title,
    source: 'moi',
    config: '{}',
  };
}

function sessionList(sessions: SessionRecord[]): SessionListResponse {
  return {
    sessions,
    total: sessions.length,
    page: 1,
    page_size: 10,
  };
}

function createDeferred<T>() {
  let resolvePromise: (value: T | PromiseLike<T>) => void = () => undefined;
  let rejectPromise: (reason?: unknown) => void = () => undefined;
  const promise = new Promise<T>((resolve, reject) => {
    resolvePromise = resolve;
    rejectPromise = reject;
  });
  return {
    promise,
    resolve: resolvePromise,
    reject: rejectPromise,
  };
}
