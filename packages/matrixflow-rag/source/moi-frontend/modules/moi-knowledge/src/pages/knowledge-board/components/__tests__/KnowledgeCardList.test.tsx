import { act, createElement, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

import { createEmptyKnowledge, queryKnowledgeList } from '../../../../service/knowledge';
import KnowledgeCardList from '../KnowledgeCardList';

const mocks = vi.hoisted(() => ({
  formValues: {} as { name?: string; description?: string; image_index_enabled?: boolean },
  http: { get: vi.fn(), post: vi.fn() },
  message: { error: vi.fn(), success: vi.fn() },
  goToPage: vi.fn(),
  resetFields: vi.fn(),
  setFields: vi.fn(),
}));

vi.mock('antd', async () => {
  const React = await import('react');

  const Form = Object.assign(({ children }: { children?: ReactNode }) => React.createElement('form', null, children), {
    useForm: () => [
      {
        validateFields: vi.fn(async () => mocks.formValues),
        resetFields: mocks.resetFields,
        setFields: mocks.setFields,
      },
    ],
    Item: ({ children, label }: { children?: ReactNode; label?: ReactNode }) =>
      React.createElement('label', null, label, children),
  });
  const Input = Object.assign(
    ({ allowClear: _allowClear, showCount: _showCount, ...props }: Record<string, unknown>) =>
      React.createElement('input', props),
    {
      TextArea: ({ allowClear: _allowClear, showCount: _showCount, ...props }: Record<string, unknown>) =>
        React.createElement('textarea', props),
    },
  );
  const Modal = ({
    open,
    title,
    basicStepContent,
    onOk,
    onCancel,
    okText,
    cancelText,
    confirmLoading,
    'data-testid': testID,
  }: {
    open?: boolean;
    title?: ReactNode;
    basicStepContent?: ReactNode;
    onOk?: () => void;
    onCancel?: () => void;
    okText?: ReactNode;
    cancelText?: ReactNode;
    confirmLoading?: boolean;
    'data-testid'?: string;
  }) =>
    open
      ? React.createElement(
          'section',
          { 'data-testid': testID },
          React.createElement('h2', null, title),
          basicStepContent,
          React.createElement(
            'button',
            { type: 'button', 'data-testid': 'knowledge-base-create-cancel-btn', disabled: confirmLoading, onClick: onCancel },
            cancelText,
          ),
          React.createElement(
            'button',
            { type: 'button', 'data-testid': 'knowledge-base-create-submit-btn', disabled: confirmLoading, onClick: onOk },
            okText,
          ),
        )
      : null;

  return {
    App: Object.assign(({ children }: { children?: ReactNode }) => React.createElement('div', null, children), {
      useApp: () => ({ message: mocks.message }),
    }),
    Button: ({ children, onClick, ...props }: { children?: ReactNode; onClick?: (event: MouseEvent) => void }) =>
      React.createElement('button', { ...props, type: 'button', onClick }, children),
    Card: ({
      children,
      onClick,
      hoverable: _hoverable,
      classNames: _classNames,
      ...props
    }: {
      children?: ReactNode;
      onClick?: () => void;
    }) => React.createElement('div', { ...props, onClick }, children),
    Col: ({ children }: { children?: ReactNode }) => React.createElement('div', null, children),
    Collapse: ({ items }: { items?: Array<{ key: string; label: ReactNode; children: ReactNode }> }) =>
      React.createElement(
        'div',
        null,
        items?.map((item) => React.createElement('section', { key: item.key }, item.label, item.children)),
      ),
    Form,
    Input,
    Modal,
    Row: ({ children }: { children?: ReactNode }) => React.createElement('div', null, children),
    Space: ({ children }: { children?: ReactNode }) => React.createElement('div', null, children),
    Spin: ({ children }: { children?: ReactNode }) => React.createElement('div', null, children),
    Switch: (props: Record<string, unknown>) => React.createElement('input', { ...props, type: 'checkbox' }),
    Typography: {
      Text: ({ children, ...props }: { children?: ReactNode }) => React.createElement('span', props, children),
      Paragraph: ({ children, ...props }: { children?: ReactNode }) => React.createElement('p', props, children),
    },
  };
});

vi.mock('@ant-design/icons', async () => {
  const actual = await vi.importActual<typeof import('@ant-design/icons')>('@ant-design/icons');
  return {
    ...actual,
    LoadingOutlined: () => null,
    PlusOutlined: () => null,
    SearchOutlined: () => null,
  };
});

vi.mock('@moi/shared-moi-app-protocol/app-context', () => ({ useHttpClient: () => mocks.http }));
vi.mock('@moi/shared-moi-app-protocol/module-context', () => ({
  useModuleNavigator: () => ({ goToPage: mocks.goToPage }),
}));
vi.mock('@moi/shared-moi-utils/catalog/identifier', () => ({ isValidCatalogIdentifier: () => true }));
vi.mock('react-i18next', () => ({ useTranslation: () => ({ t: (key: string) => key }) }));
vi.mock('../../../../service/knowledge', () => ({
  createEmptyKnowledge: vi.fn(),
  deleteKnowledgeById: vi.fn(),
  getKnowledgeDetail: vi.fn(),
  queryKnowledgeList: vi.fn(),
  updateKnowledge: vi.fn(),
}));
vi.mock('../../../../shared/knowledge/error-message', () => ({
  resolveKnowledgeErrorMessage: (error: { response?: { data?: { msg?: string } } }, fallback: string) =>
    error?.response?.data?.msg ?? fallback,
}));
vi.mock('../../../../components/KnowledgeSourceSelectModal/KnowledgeSourceSelectModal', () => ({
  default: ({
    open,
    title,
    basicStepContent,
    onCancel,
    onBasicNext,
    basicNextText,
    basicNextButtonTestId,
    cancelButtonTestId,
    testIdPrefix,
  }: {
    open?: boolean;
    title?: ReactNode;
    basicStepContent?: ReactNode;
    onCancel?: () => void;
    onBasicNext?: () => Promise<boolean>;
    basicNextText?: ReactNode;
    basicNextButtonTestId?: string;
    cancelButtonTestId?: string;
    testIdPrefix?: string;
  }) =>
    open
      ? createElement(
          'section',
          { 'data-testid': `${testIdPrefix}-modal` },
          createElement('h2', null, title),
          createElement('div', { 'data-testid': `${testIdPrefix}-steps` }, '1'),
          basicStepContent,
          createElement('button', { type: 'button', 'data-testid': cancelButtonTestId, onClick: onCancel }, 'cancel'),
          createElement(
            'button',
            {
              type: 'button',
              'data-testid': basicNextButtonTestId,
              onClick: () => onBasicNext?.(),
            },
            basicNextText,
          ),
        )
      : null,
}));
vi.mock('@moi/shared-moi-components/list-action', () => ({
  ListActionButton: ({ label, onClick, ...props }: { label?: string; onClick?: (event: MouseEvent) => void }) =>
    createElement('button', { ...props, 'aria-label': label, type: 'button', onClick }, label),
}));
vi.mock('./KnowledgeMetadataEditModal', () => ({
  default: ({ open }: { open?: boolean }) =>
    open ? createElement('div', { 'data-testid': 'knowledge-metadata-edit-modal' }) : null,
}));

describe('KnowledgeCardList data-side empty knowledge base creation', () => {
  let root: Root;
  let container: HTMLDivElement;
  const reactActEnvironment = globalThis as typeof globalThis & { IS_REACT_ACT_ENVIRONMENT?: boolean };
  let previousActEnvironment: boolean | undefined;

  beforeAll(() => {
    previousActEnvironment = reactActEnvironment.IS_REACT_ACT_ENVIRONMENT;
    reactActEnvironment.IS_REACT_ACT_ENVIRONMENT = true;
    globalThis.IntersectionObserver = class {
      observe() {}
      disconnect() {}
      unobserve() {}
      takeRecords() {
        return [];
      }
      root = null;
      rootMargin = '';
      thresholds = [];
    } as typeof IntersectionObserver;
  });

  afterAll(() => {
    reactActEnvironment.IS_REACT_ACT_ENVIRONMENT = previousActEnvironment;
  });

  beforeEach(() => {
    mocks.formValues = { name: 'data_kb', description: 'created from data', image_index_enabled: true };
    vi.mocked(queryKnowledgeList).mockResolvedValue({ items: [], total: 0, next_page_token: '' });
    vi.mocked(createEmptyKnowledge).mockResolvedValue({
      model: {
        id: 77,
        name: 'data_kb',
        description: 'created from data',
        tables: [],
        files: { file_ids: [], vector_table: 'kb_77_text_index', embedding_model: 'bge-m3' },
        source_counts: { files: 0, tables: 0, total: 0 },
        table_set_hash: '',
        created_at: 0,
        updated_at: 0,
      },
      data_domain: {
        model_id: 77,
        catalog_id: 3,
        database_id: 11,
        raw_volume_id: 12,
        processed_volume_id: 13,
        ensure_status: 'ready',
        last_ensure_error: null,
        last_checked_at: 0,
      },
    });
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    vi.clearAllMocks();
  });

  async function renderAndOpenCreateModal() {
    await act(async () => {
      root.render(createElement(KnowledgeCardList));
      await Promise.resolve();
    });
    await act(async () => {
      (container.querySelector('[data-testid="knowledge-base-create-btn"]') as HTMLButtonElement).click();
      await Promise.resolve();
    });
  }

  async function submitCreate() {
    await act(async () => {
      (container.querySelector('[data-testid="knowledge-base-create-submit-btn"]') as HTMLButtonElement).click();
      await Promise.resolve();
    });
  }

  it('loads the next page when the list sentinel becomes visible', async () => {
    const originalIntersectionObserver = globalThis.IntersectionObserver;
    globalThis.IntersectionObserver = class {
      constructor(private readonly callback: IntersectionObserverCallback) {}

      observe(target: Element) {
        this.callback([{ isIntersecting: true, target } as IntersectionObserverEntry], this);
      }

      disconnect() {}
      unobserve() {}
      takeRecords() {
        return [];
      }
      root = null;
      rootMargin = '';
      thresholds = [];
    } as typeof IntersectionObserver;
    vi.mocked(queryKnowledgeList)
      .mockResolvedValueOnce({
        items: [
          {
            id: 1,
            name: 'first page',
            description: '',
            files: { file_ids: [], volume_ids: [] },
            tables: [],
            source_counts: { files: 1, tables: 0, total: 1 },
            table_set_hash: '',
            created_at: 0,
            updated_at: 0,
          },
        ],
        total: 2,
        next_page_token: 'next-page',
      })
      .mockResolvedValueOnce({
        items: [
          {
            id: 2,
            name: 'second page',
            description: '',
            files: { file_ids: [], volume_ids: [] },
            tables: [],
            source_counts: { files: 0, tables: 1, total: 1 },
            table_set_hash: '',
            created_at: 0,
            updated_at: 0,
          },
        ],
        total: 2,
        next_page_token: '',
      });

    try {
      await act(async () => {
        root.render(createElement(KnowledgeCardList));
        await Promise.resolve();
        await Promise.resolve();
      });

      expect(queryKnowledgeList).toHaveBeenLastCalledWith(mocks.http, expect.objectContaining({ page_token: 'next-page' }));
      expect(container.textContent).toContain('first page');
      expect(container.textContent).toContain('second page');
    } finally {
      globalThis.IntersectionObserver = originalIntersectionObserver;
    }
  });

  it('shows knowledge base actions without a frontend permission context', async () => {
    vi.mocked(queryKnowledgeList).mockResolvedValueOnce({
      items: [
        {
          id: 7,
          name: 'Operations Knowledge',
          description: 'Existing semantic model',
          files: { file_ids: ['file-1'], volume_ids: ['volume-1'] },
          tables: [],
          source_counts: { files: 2, tables: 1, total: 3 },
          table_set_hash: '',
          created_at: 0,
          updated_at: 0,
        },
      ],
      total: 1,
      next_page_token: '',
    });

    await act(async () => {
      root.render(createElement(KnowledgeCardList));
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(container.textContent).toContain('Operations Knowledge');
    expect(container.querySelector('[aria-label="knowledge.base.card-edit-metadata"]')).toBeTruthy();
    expect(container.querySelector('[aria-label="knowledge.base.card-edit-advanced"]')).toBeTruthy();
    expect(container.querySelector('[aria-label="knowledge.base.card-delete"]')).toBeTruthy();
    expect(container.querySelector('[data-testid="knowledge-base-card-metadata-edit-7"]')).toBeTruthy();
    expect(container.querySelector('[data-testid="knowledge-base-card-advanced-config-7"]')).toBeTruthy();
    expect(container.querySelector('[data-testid="knowledge-base-card-delete-7"]')).toBeTruthy();
  });

  it('shows only basic information and advanced options, without source selection steps', async () => {
    await renderAndOpenCreateModal();

    expect(container.querySelector('[data-testid="knowledge-base-create-modal"]')).toBeTruthy();
    expect(container.querySelector('[data-testid="knowledge-base-create-base-step"]')).toBeTruthy();
    expect(container.querySelector('[data-testid="knowledge-base-create-name-input"]')).toBeTruthy();
    expect(container.querySelector('[data-testid="knowledge-base-create-description-input"]')).toBeTruthy();
    expect(container.querySelector('[data-testid="knowledge-base-create-advanced-options"]')).toBeTruthy();
    expect(container.querySelector('[data-testid="knowledge-base-create-image-index-switch"]')).toBeTruthy();
    expect(container.querySelector('[data-testid="knowledge-base-create-source-choice"]')).toBeNull();
    expect(container.querySelector('[data-testid="knowledge-base-create-base-next-btn"]')).toBeNull();
    expect(container.querySelector('[data-testid="knowledge-base-create-steps"]')?.textContent).toBe('1');
  });

  it('shows a user-facing create success message without internal status or job count', async () => {
    await renderAndOpenCreateModal();

    await submitCreate();

    expect(createEmptyKnowledge).toHaveBeenCalledWith(mocks.http, {
      name: 'data_kb',
      description: 'created from data',
      image_index_enabled: true,
    });
    const request = vi.mocked(createEmptyKnowledge).mock.calls[0]?.[1];
    expect(request).not.toHaveProperty('sources');
    expect(request).not.toHaveProperty('source_selections');
    expect(mocks.message.success).toHaveBeenCalledWith({ content: 'knowledge.base.create-success', duration: 1 });
  });

  it('keeps name validation in the basic information form', async () => {
    mocks.formValues = { name: '', description: 'created from data', image_index_enabled: false };
    await renderAndOpenCreateModal();

    await submitCreate();

    expect(createEmptyKnowledge).not.toHaveBeenCalled();
    expect(mocks.message.error).toHaveBeenCalledWith('knowledge.base.form-name-required');
  });

  it('defaults image indexing to disabled when the form omits it', async () => {
    mocks.formValues = { name: 'data_kb', description: 'created from data' };
    await renderAndOpenCreateModal();

    await submitCreate();

    expect(createEmptyKnowledge).toHaveBeenCalledWith(mocks.http, expect.objectContaining({ image_index_enabled: false }));
  });

  it('shows a field error when the empty knowledge-base name conflicts', async () => {
    const conflictMessage = 'Knowledge base name conflicts with Catalog database "Team Catalog/data_kb"';
    vi.mocked(createEmptyKnowledge).mockRejectedValueOnce({ response: { status: 409, data: { msg: conflictMessage } } });
    await renderAndOpenCreateModal();

    await submitCreate();

    expect(mocks.setFields).toHaveBeenCalledWith([{ name: 'name', errors: [conflictMessage] }]);
    expect(mocks.message.error).toHaveBeenCalledWith(conflictMessage);
  });

  it('shows the generic create error for a non-conflict failure', async () => {
    vi.mocked(createEmptyKnowledge).mockRejectedValueOnce({});
    await renderAndOpenCreateModal();

    await submitCreate();

    expect(mocks.setFields).not.toHaveBeenCalled();
    expect(mocks.message.error).toHaveBeenCalledWith('knowledge.base.create-failed');
  });

  it('cancels the pending redirect when creation is reopened', async () => {
    vi.useFakeTimers();
    try {
      await renderAndOpenCreateModal();
      await submitCreate();

      await act(async () => {
        (container.querySelector('[data-testid="knowledge-base-create-btn"]') as HTMLButtonElement).click();
        await vi.advanceTimersByTimeAsync(1000);
      });

      expect(mocks.goToPage).not.toHaveBeenCalled();
      expect(container.querySelector('[data-testid="knowledge-base-create-modal"]')).toBeTruthy();
    } finally {
      vi.useRealTimers();
    }
  });
});
