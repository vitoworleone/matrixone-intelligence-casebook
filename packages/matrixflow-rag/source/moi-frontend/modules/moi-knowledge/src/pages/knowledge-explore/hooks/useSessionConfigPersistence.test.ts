import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { useSessionConfigPersistence } from './useSessionConfigPersistence';

const dialogSessionMocks = vi.hoisted(() => ({
  getSession: vi.fn(),
  updateSession: vi.fn(),
}));

vi.mock('../../../service/dialogSession', () => ({
  getSession: dialogSessionMocks.getSession,
  updateSession: dialogSessionMocks.updateSession,
}));

describe('useSessionConfigPersistence', () => {
  const http = {} as AppHttpClient;
  const reactActGlobal = globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: unknown };
  let root: Root | null = null;
  let container: HTMLDivElement | null = null;
  let previousActEnvironment: unknown;
  let persistSessionConfig: ReturnType<typeof useSessionConfigPersistence> | null = null;

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
    persistSessionConfig = null;
  });

  afterEach(() => {
    act(() => {
      root?.unmount();
    });
    container?.remove();
    root = null;
    container = null;
    persistSessionConfig = null;
  });

  it('removes legacy reasoning effort fields when selected reasoning effort is cleared', async () => {
    dialogSessionMocks.getSession.mockResolvedValue({
      config: JSON.stringify({
        semantic_models: [{ semantic_model_id: 1 }],
        llm: {
          model: 'model-a',
          backend_id: 2,
          reasoning_effort: 'high',
        },
        model: 'model-a',
        llm_reasoning_effort: 'high',
        reasoning_effort: 'high',
      }),
    });
    dialogSessionMocks.updateSession.mockResolvedValue({ session_id: 10 });

    await renderProbe();

    await act(async () => {
      await persistSessionConfig?.({
        sessionId: 10,
        selectedKnowledgeIds: [1],
        selectedModel: 'model-a',
        selectedModelBackendId: 2,
        selectedReasoningEffort: '',
      });
    });

    expect(dialogSessionMocks.updateSession).toHaveBeenCalledTimes(1);
    const body = dialogSessionMocks.updateSession.mock.calls[0]?.[1];
    const savedConfig = JSON.parse(body.config) as Record<string, unknown>;
    const llmConfig = savedConfig.llm as Record<string, unknown>;

    expect(llmConfig.reasoning_effort).toBeUndefined();
    expect(savedConfig.llm_reasoning_effort).toBeUndefined();
    expect(savedConfig.reasoning_effort).toBeUndefined();
  });

  async function renderProbe(): Promise<void> {
    await act(async () => {
      root?.render(createElement(SessionConfigPersistenceProbe));
    });
  }

  function SessionConfigPersistenceProbe() {
    persistSessionConfig = useSessionConfigPersistence(http);
    return null;
  }
});
