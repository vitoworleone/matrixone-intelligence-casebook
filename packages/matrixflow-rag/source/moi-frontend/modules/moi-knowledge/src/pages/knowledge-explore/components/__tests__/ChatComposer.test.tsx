import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import type { TFunction } from 'i18next';
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

import { ChatComposer } from '../ChatComposer';

vi.mock('@ant-design/x', () => ({
  Sender: ({
    value,
    onChange,
    onSubmit,
    disabled,
    header,
    footer,
    'data-testid': testId,
  }: {
    value: string;
    onChange: (value: string) => void;
    onSubmit: () => void;
    disabled?: boolean;
    header?: JSX.Element;
    footer?: (
      actions: unknown,
      context: {
        components: {
          SendButton: (props: { disabled?: boolean; 'data-testid'?: string }) => JSX.Element;
        };
      },
    ) => JSX.Element;
    'data-testid'?: string;
  }) => (
    <div data-testid={testId}>
      <textarea
        data-testid={`${testId}-input`}
        disabled={disabled}
        value={value}
        onChange={(event) => onChange(event.currentTarget.value)}
      />
      <button data-testid={`${testId}-submit`} disabled={disabled} onClick={onSubmit}>
        submit
      </button>
      {header ? <div data-testid={`${testId}-header`}>{header}</div> : null}
      <div data-testid={`${testId}-footer`}>
        {footer?.(null, {
          components: {
            SendButton: (props) => (
              <button data-testid={props['data-testid']} disabled={props.disabled} onClick={onSubmit}>
                send
              </button>
            ),
          },
        })}
      </div>
    </div>
  ),
}));

describe('ChatComposer', () => {
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
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => {
      root?.unmount();
    });
    container?.remove();
    document.querySelectorAll('.ant-dropdown').forEach((element) => element.remove());
    root = null;
    container = null;
    vi.clearAllMocks();
  });

  it('keeps composer disabled when no session can be created on send', async () => {
    await renderComposer({
      selectedSessionId: null,
      canCreateSessionOnSend: false,
      onSend: vi.fn(),
    });

    expect(input().disabled).toBe(true);
    expect(sendButton().disabled).toBe(true);
  });

  it('allows first message submit for a pending lazy session', async () => {
    const onSend = vi.fn();
    await renderComposer({
      selectedSessionId: null,
      canCreateSessionOnSend: true,
      onSend,
    });

    expect(input().disabled).toBe(false);
    expect(sendButton().disabled).toBe(false);

    await act(async () => {
      submitButton().click();
    });

    expect(onSend).toHaveBeenCalledTimes(1);
  });

  it('disables a pending lazy session while its backend session is being created', async () => {
    await renderComposer({
      selectedSessionId: null,
      canCreateSessionOnSend: true,
      isSessionCreating: true,
    });

    expect(input().disabled).toBe(true);
    expect(submitButton().disabled).toBe(true);
    expect(sendButton().disabled).toBe(true);
    expect(container?.querySelector('[data-testid="knowledge-explore-composer-stop-btn"]')).toBeNull();
  });

  it('does not render reasoning effort in the composer surface', async () => {
    await renderComposer({
      selectedReasoningEffort: 'medium',
    });

    expect(container?.querySelector('[data-testid="knowledge-explore-reasoning-effort-input"]')).toBeNull();
  });

  it('exposes image upload through the plus menu without submitting', async () => {
    const onSend = vi.fn();
    const onUploadImage = vi.fn(async () => {});
    await renderComposer({
      selectedSessionId: 12,
      onSend,
      onUploadImage,
    });

    expect(container?.querySelector('[data-testid="knowledge-explore-query-visual-upload-btn"]')).toBeNull();

    await act(async () => {
      uploadMenuButton().click();
      await nextTick();
    });

    expect(document.body.querySelector('[data-testid="knowledge-explore-query-visual-upload-btn"]')).not.toBeNull();

    const uploadInput = requireBodyElement<HTMLInputElement>('input[type="file"]');
    const file = new File(['image'], 'chart.png', { type: 'image/png' });

    await act(async () => {
      Object.defineProperty(uploadInput, 'files', { value: [file], configurable: true });
      uploadInput.dispatchEvent(new Event('change', { bubbles: true }));
      await nextTick();
    });

    expect(onUploadImage).toHaveBeenCalledWith(file);
    expect(onSend).not.toHaveBeenCalled();
  });

  async function renderComposer(props: {
    selectedSessionId?: number | null;
    canCreateSessionOnSend?: boolean;
    isSessionCreating?: boolean;
    onSend?: () => void;
    onUploadImage?: (file: File) => Promise<void>;
    selectedReasoningEffort?: string;
  }): Promise<void> {
    await act(async () => {
      root?.render(
        createElement(ChatComposer, {
          currentRuntime: {
            draftInput: 'hello',
            selectedModel: 'model-a',
            selectedReasoningEffort: props.selectedReasoningEffort ?? '',
            selectedKnowledgeIds: [7],
            isSendPending: false,
            isStreaming: false,
            queryVisuals: [],
          },
          selectedSessionId: props.selectedSessionId === undefined ? 1 : props.selectedSessionId,
          canCreateSessionOnSend: props.canCreateSessionOnSend ?? false,
          isSessionCreating: props.isSessionCreating ?? false,
          selectedModelValue: 'model-a',
          t,
          knowledgeList: [{ id: 7, name: 'Knowledge 7' }],
          isKnowledgeLoading: false,
          modelOptions: [{ value: 'model-a', label: 'Model A', backendId: 0 }],
          isModelLoading: false,
          onDraftChange: vi.fn(),
          onModelChange: vi.fn(),
          onToggleKnowledge: vi.fn(),
          onRemoveKnowledge: vi.fn(),
          onUploadImage: props.onUploadImage ?? vi.fn(async () => {}),
          onRemoveQueryVisual: vi.fn(),
          onSend: props.onSend ?? vi.fn(),
          onStop: vi.fn(),
        }),
      );
    });
  }

  function input(): HTMLTextAreaElement {
    return requireElement<HTMLTextAreaElement>('[data-testid="knowledge-explore-composer-sender-input"]');
  }

  function submitButton(): HTMLButtonElement {
    return requireElement<HTMLButtonElement>('[data-testid="knowledge-explore-composer-sender-submit"]');
  }

  function sendButton(): HTMLButtonElement {
    return requireElement<HTMLButtonElement>('[data-testid="knowledge-explore-composer-send-btn"]');
  }

  function uploadMenuButton(): HTMLButtonElement {
    return requireElement<HTMLButtonElement>('[data-testid="knowledge-explore-upload-menu-btn"]');
  }

  function requireElement<T extends HTMLElement>(selector: string): T {
    const element = container?.querySelector<T>(selector);
    if (!element) {
      throw new Error(`missing element: ${selector}`);
    }
    return element;
  }
});

function requireBodyElement<T extends HTMLElement>(selector: string): T {
  const element = document.body.querySelector<T>(selector);
  if (!element) {
    throw new Error(`missing body element: ${selector}`);
  }
  return element;
}

function nextTick(): Promise<void> {
  return new Promise((resolve) => {
    setTimeout(resolve, 0);
  });
}
