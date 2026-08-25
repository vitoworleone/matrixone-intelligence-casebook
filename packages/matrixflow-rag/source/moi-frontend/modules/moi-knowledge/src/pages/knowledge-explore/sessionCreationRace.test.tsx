import React, { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { App } from 'antd';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { KnowledgeExplorePage } from './index';

const mocks = vi.hoisted(() => ({
  createFixedSession: vi.fn(),
  releaseEmptySelection: vi.fn(),
  sendSessionMessage: vi.fn(),
  setSelectedSessionId: vi.fn(),
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    i18n: { language: 'en-US' },
    t: (key: string) => key,
  }),
}));

vi.mock('@moi/shared-moi-app-protocol/app-context', () => ({
  useHttpClient: () => ({}),
  useSSEClient: () => ({}),
  useTimezone: () => 'UTC',
}));

vi.mock('@moi/shared-moi-app-protocol/business-context', () => ({
  useWorkspaceId: () => 'workspace-1',
}));

vi.mock('@moi/shared-moi-components/ai-chat-message/agent-chat-page', () => ({
  AgentChatPage: ({ composer }: { composer: React.ReactNode }) => <>{composer}</>,
}));

vi.mock('@moi/shared-moi-components/ai-chat-session-manager', () => ({
  AiChatSessionManager: () => null,
}));

vi.mock('./components/ChatComposer', () => ({
  ChatComposer: ({
    canCreateSessionOnSend,
    currentRuntime,
    onDraftChange,
    onModelChange,
    onSend,
  }: {
    canCreateSessionOnSend: boolean;
    currentRuntime: { selectedKnowledgeIds: number[] };
    onDraftChange: (value: string) => void;
    onModelChange: (value: string, backendId?: number | null) => void;
    onSend: () => void;
  }) => (
    <div>
      <span data-testid="pending-knowledge-ids">{currentRuntime.selectedKnowledgeIds.join(',')}</span>
      <span data-testid="can-create-session">{String(canCreateSessionOnSend)}</span>
      <button type="button" data-testid="set-model" onClick={() => onModelChange('model-1', 1)}>
        set model
      </button>
      <button type="button" data-testid="set-draft" onClick={() => onDraftChange('question')}>
        set draft
      </button>
      <button type="button" data-testid="send" onClick={onSend}>
        send
      </button>
    </div>
  ),
}));

vi.mock('./hooks/useExploreSessionManager', () => ({
  useExploreSessionManager: () => ({
    createFixedSession: mocks.createFixedSession,
    createNewSession: vi.fn(),
    deleteSession: vi.fn(),
    groupedSessions: [],
    handleSearch: vi.fn(),
    hasMore: false,
    isActionLoading: false,
    isLoading: false,
    isLoadingMore: false,
    isPinnedLoading: false,
    isSearching: false,
    isSessionCreating: false,
    loadMore: vi.fn(),
    pinnedSessionList: [],
    releaseEmptySelection: mocks.releaseEmptySelection,
    renameSession: vi.fn(),
    searchKeyword: '',
    selectedSession: null,
    selectedSessionId: null,
    setSelectedSessionId: mocks.setSelectedSessionId,
    syncSessionTitle: vi.fn(),
    togglePin: vi.fn(),
  }),
}));

vi.mock('./hooks/useExploreA2ARuntimeManager', () => ({
  useExploreA2ARuntimeManager: () => ({
    addCurrentQueryVisual: vi.fn(),
    currentRuntime: {
      draftInput: '',
      isSendPending: false,
      isStreaming: false,
      messages: [],
      projection: {},
      queryVisuals: [],
      selectedKnowledgeIds: [],
      selectedModel: '',
      selectedModelBackendId: null,
      selectedReasoningEffort: '',
      updatedAt: 0,
    },
    removeCurrentKnowledgeSelection: vi.fn(),
    removeCurrentQueryVisual: vi.fn(),
    runningSessionCount: 0,
    sendCurrentSessionMessage: vi.fn(),
    sendSessionMessage: mocks.sendSessionMessage,
    setCurrentDraftInput: vi.fn(),
    setCurrentKnowledgeSelection: vi.fn(),
    setCurrentModel: vi.fn(),
    stopCurrentSessionStream: vi.fn(),
    submitCurrentSessionInput: vi.fn(),
    syncSessionDefaultKnowledge: vi.fn(),
    syncSessionDefaultModel: vi.fn(),
    syncSessionDefaultReasoningEffort: vi.fn(),
    toggleCurrentKnowledgeSelection: vi.fn(),
  }),
}));

vi.mock('./hooks/useExploreKnowledgeOptions', () => ({
  useExploreKnowledgeOptions: () => ({ isLoading: false, knowledgeList: [] }),
}));

vi.mock('./hooks/useExploreModelOptions', () => ({
  decodeModelValue: (value: string) => ({ backendId: null, model: value }),
  encodeModelValue: (model: string) => model,
  formatExploreModelLabel: (model: { model: string }) => model.model,
  isSystemDefaultModel: () => false,
  useExploreModelOptions: () => ({ isLoading: false, modelList: [] }),
}));

vi.mock('./hooks/useSessionConfigPersistence', () => ({
  useSessionConfigPersistence: () => vi.fn(),
}));

describe('KnowledgeExplorePage lazy-session creation', () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    (globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
    vi.clearAllMocks();
    mocks.sendSessionMessage.mockResolvedValue(true);
  });

  afterEach(() => {
    act(() => {
      root.unmount();
    });
    container.remove();
  });

  it('keeps a newer virtual session after an older fixed-session creation completes', async () => {
    const creation = createDeferred<{ id: number }>();
    mocks.createFixedSession.mockImplementationOnce(() => creation.promise);

    await render({ knowledgeId: 11, requestId: 1 });
    await click('[data-testid="set-model"]');
    await click('[data-testid="set-draft"]');
    await click('[data-testid="send"]');
    expect(mocks.createFixedSession).toHaveBeenCalledWith(11);

    await render({ knowledgeId: 22, requestId: 2 });
    expect(text('[data-testid="pending-knowledge-ids"]')).toBe('22');

    await act(async () => {
      creation.resolve({ id: 100 });
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.sendSessionMessage).toHaveBeenCalledWith(
      100,
      expect.objectContaining({ selectedKnowledgeIds: [11] }),
      undefined,
    );
    expect(text('[data-testid="pending-knowledge-ids"]')).toBe('22');
    expect(text('[data-testid="can-create-session"]')).toBe('true');
  });

  async function render(createRequest: { knowledgeId: number; requestId: number }): Promise<void> {
    await act(async () => {
      root.render(
        <App>
          <KnowledgeExplorePage createRequest={createRequest} />
        </App>,
      );
      await Promise.resolve();
    });
  }

  async function click(selector: string): Promise<void> {
    const element = container.querySelector(selector);
    if (!(element instanceof HTMLElement)) {
      throw new Error(`Missing element: ${selector}`);
    }
    await act(async () => {
      element.click();
      await Promise.resolve();
    });
  }

  function text(selector: string): string | null {
    return container.querySelector(selector)?.textContent ?? null;
  }
});

function createDeferred<T>() {
  let resolvePromise: (value: T | PromiseLike<T>) => void = () => undefined;
  const promise = new Promise<T>((resolve) => {
    resolvePromise = resolve;
  });
  return {
    promise,
    resolve: resolvePromise,
  };
}
