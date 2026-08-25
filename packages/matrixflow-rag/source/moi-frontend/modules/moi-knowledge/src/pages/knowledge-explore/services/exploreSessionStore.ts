/**
 * Explore session store — owns session-list level state:
 * - CRUD / pin / search / pagination for sessions
 * - selected session orchestration for page UI
 *
 * Runtime stream state and history cache are delegated to dedicated stores.
 * Uses lazy getters instead of class-level field initializers to break
 * circular module initialization dependency.
 */

import { message } from 'antd';
import type { TFunction } from 'i18next';

import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import {
  createSession,
  deleteSession,
  getPinnedSessions,
  getSessionList,
  getUnpinnedSessions,
  isSessionPinned,
  pinSession,
  unpinSession,
  updateSession,
  type SessionListResponse,
  type SessionRecord,
} from '../../../service/dialogSession';
import { withPinnedTag } from '../utils/session';
import { getExploreA2ARuntimeStore } from './exploreA2ARuntimeStore';

const PAGE_SIZE = 10;

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface ExploreSessionStoreState {
  sessionList: SessionRecord[];
  pinnedSessionList: SessionRecord[];
  selectedSessionId: number | null;
  searchKeyword: string;
  isLoading: boolean;
  isPinnedLoading: boolean;
  isSearching: boolean;
  isActionLoading: boolean;
  isSessionCreating: boolean;
  isLoadingMore: boolean;
  currentPage: number;
  hasMore: boolean;
}

interface SessionStoreActionContext {
  http: AppHttpClient;
  t: TFunction<'moi-knowledge'>;
}

type StoreListener = () => void;

function createInitialState(): ExploreSessionStoreState {
  return {
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
}

function dedupeById(list: SessionRecord[]): SessionRecord[] {
  const map = new Map<number, SessionRecord>();
  list.forEach((item) => {
    map.set(item.id, item);
  });
  return Array.from(map.values());
}

function normalizePositiveIds(ids: unknown): number[] {
  if (!Array.isArray(ids)) {
    return [];
  }
  const seen = new Set<number>();
  const normalized: number[] = [];
  ids.forEach((id) => {
    if (Number.isInteger(id) && id > 0 && !seen.has(id)) {
      seen.add(id);
      normalized.push(id);
    }
  });
  return normalized;
}

// ---------------------------------------------------------------------------
// Store class
// ---------------------------------------------------------------------------

class ExploreSessionStore {
  private state: ExploreSessionStoreState = createInitialState();
  private listeners = new Set<StoreListener>();
  private isBootstrapped = false;
  private searchTimerRef: ReturnType<typeof setTimeout> | null = null;
  private isLoadingRef = false;
  private searchRequestSeq = 0;
  private selectionRevision = 0;
  private createdSessionRevision = 0;
  private keepEmptySelection = false;

  // Lazy getters to break circular module initialization.
  private get runtimeManager() {
    return getExploreA2ARuntimeStore();
  }

  subscribe(listener: StoreListener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  getState = (): ExploreSessionStoreState => {
    return this.state;
  };

  cleanupTransientState(): void {
    this.clearSearchTimer();
    this.searchRequestSeq += 1;
    if (this.state.isSearching) {
      this.setState((prev) => ({
        ...prev,
        isSearching: false,
      }));
    }
  }

  async bootstrap(context: SessionStoreActionContext): Promise<void> {
    if (this.isBootstrapped) {
      this.reconcileSelectedSession();
      return;
    }

    await this.refresh(context);
    this.isBootstrapped = true;
  }

  setSelectedSessionId(sessionId: number | null): void {
    this.selectSession(sessionId);
  }

  releaseEmptySelection(): void {
    if (!this.keepEmptySelection) {
      return;
    }
    this.selectSession(null, false);
  }

  async refresh(context: SessionStoreActionContext): Promise<void> {
    const selectionRevision = this.selectionRevision;
    await Promise.all([this.loadUnpinnedSessionList(context, { page: 1, reset: true }), this.loadPinnedSessionList(context)]);
    if (selectionRevision === this.selectionRevision && !this.state.searchKeyword.trim()) {
      this.reconcileSelectedSession();
    }
  }

  async loadMore(context: SessionStoreActionContext): Promise<void> {
    const { hasMore, isLoadingMore, searchKeyword, currentPage } = this.state;
    if (!hasMore || isLoadingMore || this.isLoadingRef || Boolean(searchKeyword.trim())) {
      return;
    }

    await this.loadUnpinnedSessionList(context, { page: currentPage + 1, reset: false });
  }

  handleSearch(keyword: string, context: SessionStoreActionContext): void {
    if (this.state.isSessionCreating) {
      return;
    }

    this.setState((prev) => ({
      ...prev,
      searchKeyword: keyword,
    }));

    this.clearSearchTimer();

    const trimmedKeyword = keyword.trim();
    if (!trimmedKeyword) {
      this.searchRequestSeq += 1;
      this.setState((prev) => ({
        ...prev,
        isSearching: false,
      }));
      this.refresh(context).catch((error) => {
        console.warn('[ExploreSessionStore] refresh after clearing search failed', { msg: (error as Error).message });
      });
      return;
    }

    const requestSeq = ++this.searchRequestSeq;
    this.searchTimerRef = setTimeout(() => {
      this.searchSessionList(trimmedKeyword, requestSeq, context).catch((error) => {
        console.warn('[ExploreSessionStore] debounced search failed', { msg: (error as Error).message });
      });
    }, 300);
  }

  async createNewSession(options: { defaultKnowledgeIds?: number[] }, context: SessionStoreActionContext): Promise<boolean> {
    if (this.state.isSessionCreating) {
      return false;
    }

    this.cancelSearchForSessionCreation(context);
    const previousSelectedSessionId = this.state.selectedSessionId;
    const shouldKeepEmptySelection = this.keepEmptySelection;
    this.selectSession(null, true);
    const creationSelectionRevision = this.selectionRevision;
    this.setState((prev) => ({
      ...prev,
      isActionLoading: true,
      isSessionCreating: true,
    }));

    try {
      const defaultKnowledgeIds = normalizePositiveIds(options.defaultKnowledgeIds);
      const createdSession = await createSession(context.http, {
        title: context.t('knowledge.explore.default-session-title'),
        config: JSON.stringify({
          type: 'free',
          ...(defaultKnowledgeIds.length > 0
            ? { semantic_models: defaultKnowledgeIds.map((id) => ({ semantic_model_id: id })) }
            : {}),
        }),
      });

      this.createdSessionRevision += 1;
      this.setState((prev) => ({
        ...prev,
        sessionList: [createdSession, ...prev.sessionList.filter((item) => item.id !== createdSession.id)],
      }));
      if (creationSelectionRevision === this.selectionRevision) {
        this.selectSession(createdSession.id);
      }
      if (defaultKnowledgeIds.length > 0) {
        this.runtimeManager.syncSessionDefaultKnowledge(createdSession.id, defaultKnowledgeIds);
      } else {
        this.runtimeManager.ensureSessionRuntime(createdSession.id);
      }
      message.success(context.t('knowledge.explore.create-success'));
      return true;
    } catch (error) {
      console.warn('[ExploreSessionStore] create session failed', { msg: (error as Error).message });
      if (creationSelectionRevision === this.selectionRevision) {
        this.selectSession(previousSelectedSessionId, shouldKeepEmptySelection);
      }
      message.error(context.t('knowledge.explore.create-failed'));
      return false;
    } finally {
      this.setState((prev) => ({
        ...prev,
        isActionLoading: false,
        isSessionCreating: false,
      }));
    }
  }

  async createFixedSession(
    options: { knowledgeId: number; title?: string },
    context: SessionStoreActionContext,
  ): Promise<SessionRecord | null> {
    if (!Number.isInteger(options.knowledgeId) || options.knowledgeId <= 0) {
      console.warn('[ExploreSessionStore.createFixedSession] invalid knowledge id', { knowledgeId: options.knowledgeId });
      return null;
    }

    if (this.state.isSessionCreating) {
      return null;
    }

    this.cancelSearchForSessionCreation(context);
    const previousSelectedSessionId = this.state.selectedSessionId;
    const shouldKeepEmptySelection = this.keepEmptySelection;
    this.selectSession(null, true);
    const creationSelectionRevision = this.selectionRevision;
    this.setState((prev) => ({
      ...prev,
      isActionLoading: true,
      isSessionCreating: true,
    }));

    try {
      const createdSession = await createSession(context.http, {
        title: options.title || context.t('knowledge.explore.default-session-title'),
        config: JSON.stringify({
          type: 'fixed',
          semantic_models: [{ semantic_model_id: options.knowledgeId }],
        }),
      });

      this.createdSessionRevision += 1;
      this.setState((prev) => ({
        ...prev,
        sessionList: [createdSession, ...prev.sessionList.filter((item) => item.id !== createdSession.id)],
      }));
      if (creationSelectionRevision === this.selectionRevision) {
        this.selectSession(createdSession.id);
      }
      this.runtimeManager.syncSessionDefaultKnowledge(createdSession.id, [options.knowledgeId]);

      message.success(context.t('knowledge.base.card-dialog-success'));
      return createdSession;
    } catch (error) {
      console.warn('[ExploreSessionStore] create fixed session failed', { msg: (error as Error).message });
      if (creationSelectionRevision === this.selectionRevision) {
        this.selectSession(previousSelectedSessionId, shouldKeepEmptySelection);
      }
      message.error(context.t('knowledge.base.card-dialog-failed'));
      return null;
    } finally {
      this.setState((prev) => ({
        ...prev,
        isActionLoading: false,
        isSessionCreating: false,
      }));
    }
  }

  async renameSession(sessionId: number, nextTitle: string, context: SessionStoreActionContext): Promise<boolean> {
    const trimmedTitle = nextTitle.trim();
    if (!trimmedTitle) {
      console.warn('[ExploreSessionStore.renameSession] empty title', { sessionId });
      return false;
    }

    this.setState((prev) => ({
      ...prev,
      isActionLoading: true,
    }));

    try {
      await updateSession(context.http, {
        session_id: sessionId,
        title: trimmedTitle,
      });

      this.setState((prev) => ({
        ...prev,
        sessionList: prev.sessionList.map((item) => (item.id === sessionId ? { ...item, title: trimmedTitle } : item)),
        pinnedSessionList: prev.pinnedSessionList.map((item) =>
          item.id === sessionId ? { ...item, title: trimmedTitle } : item,
        ),
      }));

      message.success(context.t('knowledge.explore.rename-success'));
      return true;
    } catch (error) {
      console.warn('[ExploreSessionStore] rename session failed', { msg: (error as Error).message });
      message.error(context.t('knowledge.explore.rename-failed'));
      return false;
    } finally {
      this.setState((prev) => ({
        ...prev,
        isActionLoading: false,
      }));
    }
  }

  syncSessionTitle(sessionId: number, nextTitle: string): boolean {
    const trimmedTitle = nextTitle.trim();
    if (!trimmedTitle) {
      console.warn('[ExploreSessionStore.syncSessionTitle] empty title', { sessionId });
      return false;
    }

    const currentSession = [...this.state.pinnedSessionList, ...this.state.sessionList].find((item) => item.id === sessionId);
    if (!currentSession) {
      console.warn('[ExploreSessionStore.syncSessionTitle] session not found', { sessionId });
      return false;
    }

    if (currentSession.title === trimmedTitle) {
      return true;
    }

    this.setState((prev) => ({
      ...prev,
      sessionList: prev.sessionList.map((item) => (item.id === sessionId ? { ...item, title: trimmedTitle } : item)),
      pinnedSessionList: prev.pinnedSessionList.map((item) => (item.id === sessionId ? { ...item, title: trimmedTitle } : item)),
    }));

    return true;
  }

  async deleteSession(sessionId: number, context: SessionStoreActionContext): Promise<boolean> {
    this.setState((prev) => ({
      ...prev,
      isActionLoading: true,
    }));

    try {
      await deleteSession(context.http, sessionId);

      this.setState((prev) => ({
        ...prev,
        sessionList: prev.sessionList.filter((item) => item.id !== sessionId),
        pinnedSessionList: prev.pinnedSessionList.filter((item) => item.id !== sessionId),
      }));
      this.runtimeManager.clearSessionRuntime(sessionId);
      this.reconcileSelectedSession();

      message.success(context.t('knowledge.explore.delete-success'));
      return true;
    } catch (error) {
      console.warn('[ExploreSessionStore] delete session failed', { msg: (error as Error).message });
      message.error(context.t('knowledge.explore.delete-failed'));
      return false;
    } finally {
      this.setState((prev) => ({
        ...prev,
        isActionLoading: false,
      }));
    }
  }

  async togglePin(session: SessionRecord, context: SessionStoreActionContext): Promise<boolean> {
    const currentlyPinned = isSessionPinned(session);

    this.setState((prev) => ({
      ...prev,
      isActionLoading: true,
    }));

    try {
      if (currentlyPinned) {
        await unpinSession(context.http, session.id);
      } else {
        await pinSession(context.http, session.id);
      }

      this.setState((prev) => {
        if (currentlyPinned) {
          return {
            ...prev,
            pinnedSessionList: prev.pinnedSessionList.filter((item) => item.id !== session.id),
            sessionList: [withPinnedTag(session, false), ...prev.sessionList.filter((item) => item.id !== session.id)],
          };
        }

        return {
          ...prev,
          sessionList: prev.sessionList.filter((item) => item.id !== session.id),
          pinnedSessionList: [withPinnedTag(session, true), ...prev.pinnedSessionList.filter((item) => item.id !== session.id)],
        };
      });
      this.reconcileSelectedSession();

      message.success(context.t(currentlyPinned ? 'knowledge.explore.unpin-success' : 'knowledge.explore.pin-success'));
      return true;
    } catch (error) {
      console.warn('[ExploreSessionStore] toggle pin failed', { msg: (error as Error).message });
      message.error(context.t(currentlyPinned ? 'knowledge.explore.unpin-failed' : 'knowledge.explore.pin-failed'));
      return false;
    } finally {
      this.setState((prev) => ({
        ...prev,
        isActionLoading: false,
      }));
    }
  }

  // ---- private ----

  private cancelSearchForSessionCreation(context: SessionStoreActionContext): void {
    const searchKeyword = this.state.searchKeyword;
    this.cleanupTransientState();
    if (!searchKeyword) {
      return;
    }

    this.setState((prev) => ({
      ...prev,
      searchKeyword: '',
    }));
    if (!searchKeyword.trim()) {
      return;
    }

    this.refresh(context).catch((error) => {
      console.warn('[ExploreSessionStore] refresh after cancelling search failed', { msg: (error as Error).message });
    });
  }

  private async loadUnpinnedSessionList(
    context: SessionStoreActionContext,
    options: { page: number; reset: boolean },
  ): Promise<void> {
    if (this.isLoadingRef && !options.reset) {
      return;
    }

    const selectionRevision = this.selectionRevision;
    const createdSessionRevision = this.createdSessionRevision;
    const sessionIdsAtRequest = new Set(this.state.sessionList.map((session) => session.id));
    this.isLoadingRef = true;

    this.setState((prev) => ({
      ...prev,
      isLoading: options.page === 1,
      isLoadingMore: options.page > 1,
    }));

    try {
      const response: SessionListResponse = await getUnpinnedSessions(context.http, {
        page: options.page,
        page_size: PAGE_SIZE,
      });
      if (this.state.searchKeyword.trim()) {
        return;
      }
      const incomingList = response.sessions || [];
      const responsePageSize = response.page_size || PAGE_SIZE;

      const isSelectionCurrent = selectionRevision === this.selectionRevision;
      const shouldRetainLocalSessions = createdSessionRevision !== this.createdSessionRevision;

      this.setState((prev) => ({
        ...prev,
        currentPage: options.page,
        hasMore: incomingList.length >= responsePageSize,
        sessionList: shouldRetainLocalSessions
          ? dedupeById([...incomingList.filter((session) => !sessionIdsAtRequest.has(session.id)), ...prev.sessionList])
          : options.reset
            ? incomingList
            : dedupeById([...prev.sessionList, ...incomingList]),
      }));
      if (isSelectionCurrent) {
        this.reconcileSelectedSession();
      }
    } catch (error) {
      if (this.state.searchKeyword.trim()) {
        return;
      }
      console.warn('[ExploreSessionStore] load unpinned sessions failed', { msg: (error as Error).message });
      message.error(context.t('knowledge.explore.load-failed'));
    } finally {
      this.isLoadingRef = false;
      this.setState((prev) => ({
        ...prev,
        isLoading: false,
        isLoadingMore: false,
      }));
    }
  }

  private async loadPinnedSessionList(context: SessionStoreActionContext): Promise<void> {
    const selectionRevision = this.selectionRevision;
    const createdSessionRevision = this.createdSessionRevision;
    const pinnedSessionIdsAtRequest = new Set(this.state.pinnedSessionList.map((session) => session.id));
    this.setState((prev) => ({
      ...prev,
      isPinnedLoading: true,
    }));

    try {
      const response = await getPinnedSessions(context.http, {
        page_size: 1000,
      });
      if (this.state.searchKeyword.trim()) {
        return;
      }
      const isSelectionCurrent = selectionRevision === this.selectionRevision;
      const shouldRetainLocalSessions = createdSessionRevision !== this.createdSessionRevision;
      const incomingList = response.sessions || [];
      this.setState((prev) => ({
        ...prev,
        pinnedSessionList: shouldRetainLocalSessions
          ? dedupeById([
              ...incomingList.filter((session) => !pinnedSessionIdsAtRequest.has(session.id)),
              ...prev.pinnedSessionList,
            ])
          : incomingList,
      }));
      if (isSelectionCurrent) {
        this.reconcileSelectedSession();
      }
    } catch (error) {
      if (this.state.searchKeyword.trim()) {
        return;
      }
      console.warn('[ExploreSessionStore] load pinned sessions failed', { msg: (error as Error).message });
      message.error(context.t('knowledge.explore.load-failed'));
    } finally {
      this.setState((prev) => ({
        ...prev,
        isPinnedLoading: false,
      }));
    }
  }

  private async searchSessionList(keyword: string, requestSeq: number, context: SessionStoreActionContext): Promise<void> {
    const selectionRevision = this.selectionRevision;
    const createdSessionRevision = this.createdSessionRevision;
    this.setState((prev) => ({
      ...prev,
      isSearching: true,
    }));

    try {
      const response = await getSessionList(context.http, {
        keyword,
        page: 1,
        page_size: PAGE_SIZE,
      });

      if (requestSeq !== this.searchRequestSeq) {
        return;
      }

      const isSelectionCurrent = selectionRevision === this.selectionRevision;
      if (createdSessionRevision !== this.createdSessionRevision) {
        return;
      }

      const sessions = response.sessions || [];
      const nextPinned = sessions.filter((session) => isSessionPinned(session));
      const nextUnpinned = sessions.filter((session) => !isSessionPinned(session));

      this.setState((prev) => ({
        ...prev,
        pinnedSessionList: nextPinned,
        sessionList: nextUnpinned,
        currentPage: 1,
        hasMore: false,
      }));
      if (isSelectionCurrent) {
        this.reconcileSelectedSession();
      }
    } catch (error) {
      if (requestSeq !== this.searchRequestSeq || selectionRevision !== this.selectionRevision) {
        return;
      }
      console.warn('[ExploreSessionStore] search sessions failed', { msg: (error as Error).message });
      message.error(context.t('knowledge.explore.search-failed'));
    } finally {
      if (requestSeq === this.searchRequestSeq) {
        this.setState((prev) => ({
          ...prev,
          isSearching: false,
        }));
      }
    }
  }

  private clearSearchTimer(): void {
    if (this.searchTimerRef) {
      clearTimeout(this.searchTimerRef);
      this.searchTimerRef = null;
    }
  }

  private reconcileSelectedSession(): void {
    if (this.keepEmptySelection) {
      return;
    }

    const allSessions = [...this.state.pinnedSessionList, ...this.state.sessionList];

    let nextSelectedSessionId = this.state.selectedSessionId;
    if (allSessions.length === 0) {
      nextSelectedSessionId = null;
    } else if (nextSelectedSessionId === null || !allSessions.some((session) => session.id === nextSelectedSessionId)) {
      nextSelectedSessionId = allSessions[0].id;
    }

    if (nextSelectedSessionId === this.state.selectedSessionId) {
      return;
    }

    this.selectSession(nextSelectedSessionId, false, false);
  }

  private selectSession(sessionId: number | null, keepEmptySelection = sessionId === null, invalidateListRequests = true): void {
    if (invalidateListRequests) {
      this.selectionRevision += 1;
    }
    this.keepEmptySelection = keepEmptySelection;
    this.setState((prev) => ({
      ...prev,
      selectedSessionId: sessionId,
    }));

    if (sessionId !== null) {
      this.runtimeManager.ensureSessionRuntime(sessionId);
    }
  }

  private setState(updater: (prev: ExploreSessionStoreState) => ExploreSessionStoreState): void {
    const nextState = updater(this.state);
    if (nextState === this.state) {
      return;
    }

    this.state = nextState;
    this.listeners.forEach((listener) => {
      listener();
    });
  }
}

// ---------------------------------------------------------------------------
// Lazy singleton
// ---------------------------------------------------------------------------

let exploreSessionStoreInstance: ExploreSessionStore | null = null;

export function getExploreSessionStore(): ExploreSessionStore {
  if (!exploreSessionStoreInstance) {
    exploreSessionStoreInstance = new ExploreSessionStore();
  }
  return exploreSessionStoreInstance;
}
