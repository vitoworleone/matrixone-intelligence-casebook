import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

import type { AppHttpClient, AppSSEClient } from '@moi/shared-moi-app-protocol/app-context';
import { getExploreA2ARuntimeStore } from '../services/exploreA2ARuntimeStore';
import { useExploreA2ARuntimeManager } from './useExploreA2ARuntimeManager';

const streamAgentA2AApiMock = vi.hoisted(() => vi.fn());
const getSessionMessagesMock = vi.hoisted(() => vi.fn());
const http = {} as AppHttpClient;
const sseClient = {} as AppSSEClient;
const knowledgeList = [{ id: 1, name: 'knowledge' }] as never;
const t = ((key: string) => key) as never;

vi.mock('@moi/shared-moi-api/agent', async () => {
  const actual = await vi.importActual<typeof import('@moi/shared-moi-api/agent')>('@moi/shared-moi-api/agent');
  return {
    ...actual,
    streamAgentA2AApi: streamAgentA2AApiMock.mockImplementation(() => ({ abort: vi.fn() })),
  };
});

vi.mock('../../../service/dialogSession', () => ({
  getSessionMessages: getSessionMessagesMock,
}));

describe('useExploreA2ARuntimeManager', () => {
  const sessionId = 1001;
  const runtimeStore = getExploreA2ARuntimeStore();
  const reactActGlobal = globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: unknown };
  let root: Root | null = null;
  let container: HTMLDivElement | null = null;
  let sendCurrentSessionMessage: ((question?: string) => Promise<boolean>) | null = null;
  let previousActEnvironment: unknown;

  beforeAll(() => {
    previousActEnvironment = reactActGlobal.IS_REACT_ACT_ENVIRONMENT;
    reactActGlobal.IS_REACT_ACT_ENVIRONMENT = true;
  });

  afterAll(() => {
    reactActGlobal.IS_REACT_ACT_ENVIRONMENT = previousActEnvironment;
  });

  beforeEach(() => {
    runtimeStore.clearSessionRuntime(sessionId);
    runtimeStore.setSelectedModel(sessionId, 'model-a');
    runtimeStore.setKnowledgeSelection(sessionId, [1]);
    streamAgentA2AApiMock.mockClear();
    getSessionMessagesMock.mockReset().mockResolvedValue([]);
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => {
      root?.unmount();
    });
    runtimeStore.clearSessionRuntime(sessionId);
    container?.remove();
    root = null;
    container = null;
    sendCurrentSessionMessage = null;
  });

  it('sends the original question as title metadata and the transformed question to the agent', async () => {
    const onBeforeSend = vi.fn(async () => {});

    await act(async () => {
      root?.render(
        createElement(Probe, {
          onBeforeSend,
        }),
      );
      await Promise.resolve();
    });

    await act(async () => {
      await expect(sendCurrentSessionMessage?.('candidate fit')).resolves.toBe(true);
    });

    expect(onBeforeSend).toHaveBeenCalledTimes(1);
    expect(streamAgentA2AApiMock).toHaveBeenCalledWith(
      expect.objectContaining({
        params: expect.objectContaining({
          message: expect.objectContaining({
            parts: [{ kind: 'text', text: 'Job Description:\nSenior engineer\n\nUser Question:\ncandidate fit' }],
          }),
        }),
      }),
      expect.anything(),
      expect.anything(),
    );
    const request = streamAgentA2AApiMock.mock.calls[0]?.[0] as { params?: Record<string, unknown> } | undefined;
    const params = request?.params as
      | { metadata?: Record<string, unknown>; message?: { metadata?: Record<string, unknown> } }
      | undefined;
    expect(params?.metadata).not.toHaveProperty('session_title_question');
    expect(params?.message?.metadata).toMatchObject({ session_title_question: 'candidate fit' });
  });

  function Probe({ onBeforeSend }: { onBeforeSend: () => Promise<void> }) {
    const manager = useExploreA2ARuntimeManager({
      http,
      sseClient,
      selectedSessionId: sessionId,
      workspaceId: 'ws_1',
      t,
      knowledgeList,
      transformQuestion: (question) => `Job Description:\nSenior engineer\n\nUser Question:\n${question}`,
      onBeforeSend,
    });
    sendCurrentSessionMessage = manager.sendCurrentSessionMessage;
    return null;
  }
});
