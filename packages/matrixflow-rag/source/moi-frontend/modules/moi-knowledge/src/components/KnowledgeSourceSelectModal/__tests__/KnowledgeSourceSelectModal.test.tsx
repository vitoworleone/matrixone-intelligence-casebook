import { act, createElement, type ReactNode } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

import KnowledgeSourceSelectModal from '../KnowledgeSourceSelectModal';

const mocks = vi.hoisted(() => ({
  catalogSelections: [] as Array<
    | {
        kind: 'volume_files';
        volume_id: number;
        all_selected: boolean;
        selected_file_ids?: string[];
      }
    | {
        kind: 'database_tables';
        database_id: number;
        all_selected: boolean;
        selected_table_ids?: number[];
      }
  >,
  http: {},
  message: {
    error: vi.fn(),
  },
  t: (key: string, params?: Record<string, unknown>) => {
    const translations: Record<string, string> = {
      'knowledge.base.create-document-sources-catalog-data-label': 'Select Data',
      'knowledge.base.create-document-sources-catalog-data-hint': 'Select Catalog files or tables',
      'knowledge.base.create-document-sources-catalog-data-required': 'Select Catalog files or tables',
      'knowledge.base.create-document-sources-catalog-data-unsupported': 'Only document files or data tables can be selected',
      'knowledge.base.create-document-sources-catalog-selected-summary': `${String(params?.files ?? 0)} files, ${String(
        params?.tables ?? 0,
      )} tables`,
      'knowledge.base.create-document-sources-local-file-required': 'Select a local file',
      'knowledge.base.create-document-sources-source-mode-change': 'Change',
      'knowledge.base.create-document-sources-source-mode-label': 'Data source',
      'knowledge.base.create-document-sources-source-mode-catalog': 'Select from Catalog',
      'knowledge.base.create-document-sources-source-mode-local': 'Local Upload',
    };
    return translations[key] ?? key;
  },
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: mocks.t,
  }),
}));

vi.mock('@moi/shared-moi-app-protocol/app-context', () => ({
  useHttpClient: () => mocks.http,
}));

vi.mock('antd', async () => {
  const React = await import('react');
  const App = Object.assign(({ children }: { children?: ReactNode }) => React.createElement('div', null, children), {
    useApp: () => ({
      message: mocks.message,
    }),
  });
  const Button = ({
    children,
    disabled,
    icon,
    loading,
    onClick,
    type: _type,
    ...props
  }: {
    children?: ReactNode;
    disabled?: boolean;
    icon?: ReactNode;
    loading?: boolean;
    onClick?: () => void;
    type?: string;
  }) =>
    React.createElement('button', { ...props, disabled: Boolean(disabled || loading), onClick, type: 'button' }, icon, children);
  const Checkbox = ({
    checked,
    disabled,
    onChange,
    ...props
  }: {
    checked?: boolean;
    disabled?: boolean;
    onChange?: (event: { target: { checked: boolean } }) => void;
  }) =>
    React.createElement('input', {
      ...props,
      checked,
      disabled,
      onChange: (event: React.ChangeEvent<HTMLInputElement>) => onChange?.({ target: { checked: event.target.checked } }),
      type: 'checkbox',
    });
  const Input = Object.assign(
    ({ allowClear: _allowClear, suffix: _suffix, ...props }: Record<string, unknown>) => React.createElement('input', props),
    {
      TextArea: ({ rows: _rows, ...props }: Record<string, unknown>) => React.createElement('textarea', props),
    },
  );
  const InputNumber = ({ min: _min, onChange, value, ...props }: Record<string, unknown>) =>
    React.createElement('input', {
      ...props,
      value,
      onChange: (event: React.ChangeEvent<HTMLInputElement>) =>
        (onChange as ((value: number) => void) | undefined)?.(Number(event.target.value)),
    });
  const Modal = ({
    cancelText: _cancelText,
    children,
    closable: _closable,
    confirmLoading: _confirmLoading,
    destroyOnHidden: _destroyOnHidden,
    footer,
    keyboard: _keyboard,
    maskClosable: _maskClosable,
    okText: _okText,
    onCancel: _onCancel,
    onOk: _onOk,
    open,
    title,
    width: _width,
    ...props
  }: {
    cancelText?: ReactNode;
    children?: ReactNode;
    closable?: boolean;
    confirmLoading?: boolean;
    destroyOnHidden?: boolean;
    footer?: ReactNode;
    keyboard?: boolean;
    maskClosable?: boolean;
    okText?: ReactNode;
    onCancel?: () => void;
    onOk?: () => void;
    open?: boolean;
    title?: ReactNode;
    width?: number;
  }) =>
    open
      ? React.createElement(
          'div',
          { ...props },
          React.createElement('h2', null, title),
          children,
          React.createElement('div', null, footer),
        )
      : null;
  const RadioButton = ({ children, value, ...props }: { children?: ReactNode; value?: string }) =>
    React.createElement('button', { ...props, type: 'button', value }, children);
  const RadioGroup = ({
    children,
    disabled,
    onChange,
    value,
    ...props
  }: {
    children?: ReactNode;
    disabled?: boolean;
    onChange?: (event: { target: { value: string } }) => void;
    value?: string;
  }) =>
    React.createElement(
      'div',
      props,
      React.Children.map(children, (child) => {
        if (!React.isValidElement<{ value?: string; onClick?: () => void; disabled?: boolean }>(child)) return child;
        return React.cloneElement(child, {
          disabled,
          onClick: () => child.props.value && onChange?.({ target: { value: child.props.value } }),
          'aria-pressed': child.props.value === value,
        });
      }),
    );
  const Radio = Object.assign(
    ({ children, value, ...props }: { children?: ReactNode; value?: string }) =>
      React.createElement('label', props, React.createElement('input', { type: 'radio', value }), children),
    {
      Button: RadioButton,
      Group: RadioGroup,
    },
  );
  const Select = ({ options, ...props }: { options?: Array<{ label: ReactNode; value: string }> }) =>
    React.createElement(
      'select',
      props,
      options?.map((option) => React.createElement('option', { key: option.value, value: option.value }, option.label)),
    );
  const Space = ({ children, ...props }: { children?: ReactNode }) => React.createElement('div', props, children);
  const Spin = ({ children }: { children?: ReactNode }) => React.createElement('div', null, children);
  const Switch = ({ checked, ...props }: { checked?: boolean }) =>
    React.createElement('input', { ...props, checked, type: 'checkbox' });
  const Typography = {
    Text: ({
      children,
      ellipsis: _ellipsis,
      strong: _strong,
      type: _textType,
      ...props
    }: {
      children?: ReactNode;
      ellipsis?: unknown;
      strong?: boolean;
      type?: string;
    }) => React.createElement('span', props, children),
  };
  const UploadBase = ({ children }: { children?: ReactNode }) => React.createElement('div', null, children);
  const Upload = Object.assign(UploadBase, {
    Dragger: UploadBase,
  });
  return {
    App,
    Button,
    Checkbox,
    Input,
    InputNumber,
    Modal,
    Radio,
    Select,
    Space,
    Spin,
    Switch,
    Typography,
    Upload,
  };
});

vi.mock('@moi/shared-moi-components/catalog-data-selector', async () => {
  const React = await import('react');
  return {
    CatalogDataSelector: ({
      onChange,
      onPreviewStatusChange,
    }: {
      onChange?: (items: typeof mocks.catalogSelections) => void;
      onPreviewStatusChange?: (status: 'ready') => void;
    }) =>
      React.createElement(
        'button',
        {
          'data-testid': 'catalog-data-selector',
          onClick: () => {
            onChange?.(mocks.catalogSelections);
            onPreviewStatusChange?.('ready');
          },
          type: 'button',
        },
        'Catalog data selector',
      ),
  };
});

describe('KnowledgeSourceSelectModal catalog-only append flow', () => {
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
    mocks.catalogSelections = [
      { kind: 'volume_files', volume_id: 3001, all_selected: false, selected_file_ids: ['catalog-file-1'] },
      { kind: 'database_tables', database_id: 1001, all_selected: false, selected_table_ids: [1001] },
    ];
    mocks.message.error.mockClear();
  });

  afterEach(() => {
    if (root) {
      act(() => {
        root?.unmount();
      });
    }
    container?.remove();
    root = null;
    container = null;
    vi.clearAllMocks();
  });

  it('submits catalog file and table payload without showing source mode switching', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);

    await act(async () => {
      root?.render(
        createElement(KnowledgeSourceSelectModal, {
          allowedSourceModes: ['catalog'],
          cancelText: 'Cancel',
          okText: 'Append',
          onCancel: vi.fn(),
          onSubmit,
          open: true,
          testIdPrefix: 'knowledge-source-select',
          title: 'Select Data',
        }),
      );
    });

    expect(container?.querySelector('[data-testid="knowledge-source-select-source-choice"]')).toBeNull();
    expect(container?.querySelector('[data-testid="knowledge-source-select-source-mode-change-btn"]')).toBeNull();
    expect(container?.querySelector('[data-testid="knowledge-source-select-local-files-upload-btn"]')).toBeNull();
    expect(container?.querySelector('[data-testid="knowledge-source-select-catalog-source-panel"]')).toBeTruthy();

    await act(async () => {
      requireElement<HTMLButtonElement>('[data-testid="catalog-data-selector"]').click();
    });
    await act(async () => {
      requireElement<HTMLButtonElement>('[data-testid="knowledge-source-select-submit-btn"]').click();
    });

    expect(onSubmit).toHaveBeenCalledWith({
      sources: [],
      source_selections: [
        { kind: 'volume_files', volume_id: 3001, all_selected: false, selected_file_ids: ['catalog-file-1'] },
        { kind: 'database_tables', database_id: 1001, all_selected: false, selected_table_ids: [1001] },
      ],
    });
    expect(mocks.message.error).not.toHaveBeenCalled();
  });

  it('blocks empty catalog selection before submit', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    mocks.catalogSelections = [];

    await act(async () => {
      root?.render(
        createElement(KnowledgeSourceSelectModal, {
          allowedSourceModes: ['catalog'],
          cancelText: 'Cancel',
          okText: 'Append',
          onCancel: vi.fn(),
          onSubmit,
          open: true,
          testIdPrefix: 'knowledge-source-select',
          title: 'Select Data',
        }),
      );
    });

    await act(async () => {
      requireElement<HTMLButtonElement>('[data-testid="knowledge-source-select-submit-btn"]').click();
    });

    expect(onSubmit).not.toHaveBeenCalled();
    expect(mocks.message.error).toHaveBeenCalledWith('Select Catalog files or tables');
  });

  function requireElement<T extends Element>(selector: string): T {
    const element = container?.querySelector<T>(selector);
    if (!element) {
      throw new Error(`missing element ${selector}`);
    }
    return element;
  }
});
