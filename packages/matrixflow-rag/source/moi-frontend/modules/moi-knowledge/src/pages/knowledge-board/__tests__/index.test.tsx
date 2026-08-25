// @vitest-environment happy-dom
import React, { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { KnowledgeBoardPage } from '..';

const mocks = vi.hoisted(() => ({
  exploreProps: [] as Array<{ enableDeveloperView?: boolean }>,
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock('../../knowledge-explore', () => ({
  KnowledgeExplorePage: (props: { enableDeveloperView?: boolean }) => {
    mocks.exploreProps.push(props);
    return <div data-testid="knowledge-explore-page-probe" data-developer-view={String(props.enableDeveloperView)} />;
  },
}));

vi.mock('../components/KnowledgeCardList', () => ({
  default: () => <div data-testid="knowledge-card-list-probe" />,
}));

describe('KnowledgeBoardPage', () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    (globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
    mocks.exploreProps.length = 0;
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => {
      root.unmount();
    });
    container.remove();
  });

  it('enables the developer view only for the knowledge-board conversation entry', async () => {
    await act(async () => {
      root.render(<KnowledgeBoardPage />);
    });

    const exploreTab = Array.from(container.querySelectorAll<HTMLElement>('[role="tab"]')).find(
      (tab) => tab.textContent === 'knowledge.base.board-tab-explore',
    );
    await act(async () => {
      exploreTab?.click();
    });

    expect(mocks.exploreProps).toContainEqual(expect.objectContaining({ enableDeveloperView: true }));
    expect(container.querySelector('[data-testid="knowledge-explore-page-probe"]')?.getAttribute('data-developer-view')).toBe(
      'true',
    );
  });
});
