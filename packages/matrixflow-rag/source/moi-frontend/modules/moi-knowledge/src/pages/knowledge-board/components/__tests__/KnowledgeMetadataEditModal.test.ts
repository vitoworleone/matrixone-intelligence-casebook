import { act, createElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

import KnowledgeMetadataEditModal, { KNOWLEDGE_NAME_MAX_LENGTH } from '../KnowledgeMetadataEditModal';

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

describe('KnowledgeMetadataEditModal', () => {
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
    root = null;
    container = null;
    vi.clearAllMocks();
  });

  it('renders the knowledge name input with the backend schema limit for issue 11383', async () => {
    await act(async () => {
      root?.render(
        createElement(KnowledgeMetadataEditModal, {
          open: true,
          initialValues: {
            name: '',
            description: '',
          },
          onCancel: vi.fn(),
          onSubmit: vi.fn(),
        }),
      );
    });

    const nameInput = document.body.querySelector<HTMLInputElement>('input[placeholder="knowledge.base.form-name-placeholder"]');
    expect(nameInput?.maxLength).toBe(KNOWLEDGE_NAME_MAX_LENGTH);
    expect(nameInput?.disabled).toBe(true);
  });
});
