import { act, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { AnswerFeedback } from './index';

const mocks = vi.hoisted(() => ({
  error: vi.fn(),
  success: vi.fn(),
}));

const translations: Record<string, string> = {
  'knowledge.explore.answer-copy': '复制',
  'knowledge.explore.answer-copy-aria': '复制回答',
  'knowledge.explore.answer-copy-success': '已复制',
  'knowledge.explore.answer-copy-failed': '复制失败',
  'knowledge.explore.answer-feedback-like': '回答有帮助',
  'knowledge.explore.answer-feedback-dislike': '回答没有帮助',
  'knowledge.explore.answer-feedback-submit-failed': '反馈提交失败，请稍后重试',
};

vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>();
  return {
    ...actual,
    App: {
      useApp: () => ({ message: mocks }),
    },
    Tooltip: ({ children, title }: { children: ReactNode; title: ReactNode }) => (
      <span data-tooltip-title={String(title)}>{children}</span>
    ),
  };
});

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => translations[key] ?? key,
  }),
}));

describe('AnswerFeedback', () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    (globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
    vi.clearAllMocks();
  });

  afterEach(() => {
    act(() => {
      root.unmount();
    });
    container.remove();
  });

  it('uses Chinese feedback labels and failure message', async () => {
    await act(async () => {
      root.render(<AnswerFeedback content="answer" onFeedback={() => Promise.reject(new Error('request failed'))} />);
      await Promise.resolve();
    });

    const likeButton = container.querySelector('[aria-label="回答有帮助"]');
    const dislikeButton = container.querySelector('[aria-label="回答没有帮助"]');
    expect(likeButton).not.toBeNull();
    expect(dislikeButton).not.toBeNull();
    expect(container.querySelector('[data-tooltip-title="回答有帮助"]')).not.toBeNull();
    expect(container.querySelector('[data-tooltip-title="回答没有帮助"]')).not.toBeNull();

    await act(async () => {
      likeButton?.dispatchEvent(new MouseEvent('click', { bubbles: true }));
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(mocks.error).toHaveBeenCalledWith('反馈提交失败，请稍后重试');
  });
});
