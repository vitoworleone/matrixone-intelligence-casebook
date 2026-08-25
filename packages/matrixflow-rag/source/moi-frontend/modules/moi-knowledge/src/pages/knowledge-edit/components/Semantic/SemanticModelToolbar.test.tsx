// @vitest-environment happy-dom
import React, { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { SemanticModelToolbar } from './SemanticModelToolbar';

const mocks = vi.hoisted(() => ({
  importSemanticModel: vi.fn(),
  onImportSuccess: vi.fn(),
  message: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

vi.mock('antd', () => ({
  Alert: ({
    title,
    description,
    showIcon: _showIcon,
    ...props
  }: {
    title?: React.ReactNode;
    description?: React.ReactNode;
    showIcon?: boolean;
  }) => (
    <div {...props}>
      {title}
      {description}
    </div>
  ),
  App: {
    useApp: () => ({ message: mocks.message }),
  },
  Button: ({
    icon: _icon,
    loading: _loading,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement> & { icon?: React.ReactNode; loading?: boolean }) => <button {...props} />,
  Input: {
    TextArea: (props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) => <textarea {...props} />,
  },
  Modal: ({
    open,
    children,
    onOk,
    onCancel,
  }: {
    open?: boolean;
    children?: React.ReactNode;
    onOk?: () => void;
    onCancel?: () => void;
  }) =>
    open ? (
      <div data-testid="semantic-config-import-modal">
        {children}
        <button onClick={onOk}>confirm</button>
        <button onClick={onCancel}>cancel</button>
      </div>
    ) : null,
  Space: ({ children }: { children?: React.ReactNode }) => <div>{children}</div>,
  Typography: {
    Text: ({ children }: { children?: React.ReactNode }) => <span>{children}</span>,
  },
}));

vi.mock('@ant-design/icons', () => ({
  DownloadOutlined: () => null,
  SafetyCertificateOutlined: () => null,
  UploadOutlined: () => null,
}));

vi.mock('../../../../service/semanticModel', () => ({
  exportSemanticModel: vi.fn(),
  importSemanticModel: mocks.importSemanticModel,
  validateSemanticModel: vi.fn(),
}));

function t(key: string, options?: Record<string, string | number>): string {
  const messages: Record<string, string> = {
    'knowledge.entry.import-failed': 'Import failed',
    'knowledge.entry.import-invalid-json': 'Invalid JSON: {{path}}',
    'knowledge.entry.import-invalid-shape': 'Invalid JSON object: {{path}}',
    'knowledge.entry.import-invalid-entries': 'Invalid entries: {{path}}',
    'knowledge.entry.import-permission-denied': 'Permission denied',
    'knowledge.entry.import-request-id': 'Request ID: {{id}}',
    'knowledge.entry.import-service-unavailable': 'Service unavailable',
    'knowledge.entry.import-success': 'Import succeeded',
    'knowledge.entry.toolbar-import': 'Import',
  };
  return (messages[key] || key).replace(/{{(\w+)}}/g, (_, name: string) => String(options?.[name] ?? ''));
}

describe('SemanticModelToolbar import diagnostics', () => {
  let container: HTMLDivElement;
  let root: Root;

  const openModal = async () => {
    await act(async () => {
      container.querySelector<HTMLButtonElement>('[data-testid="semantic-config-import-btn"]')?.click();
    });
  };

  const setImportText = async (value: string) => {
    const textarea = container.querySelector<HTMLTextAreaElement>('[data-testid="semantic-config-import-input"]');
    await act(async () => {
      if (!textarea) return;
      const setter = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')?.set;
      setter?.call(textarea, value);
      textarea.dispatchEvent(new Event('input', { bubbles: true }));
      textarea.dispatchEvent(new Event('change', { bubbles: true }));
      await Promise.resolve();
    });
  };

  const confirmImport = async () => {
    await act(async () => {
      Array.from(container.querySelectorAll('button'))
        .find((button) => button.textContent === 'confirm')
        ?.click();
      await Promise.resolve();
      await Promise.resolve();
    });
  };

  beforeEach(async () => {
    (globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
    mocks.importSemanticModel.mockReset();
    mocks.onImportSuccess.mockReset();
    mocks.message.error.mockReset();
    mocks.message.success.mockReset();
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
    await act(async () => {
      root.render(<SemanticModelToolbar http={{} as never} modelId={1} t={t as never} onImportSuccess={mocks.onImportSuccess} />);
    });
    await openModal();
  });

  afterEach(() => {
    act(() => {
      root.unmount();
    });
    container.remove();
  });

  it('shows a local JSON syntax error without calling the API', async () => {
    await setImportText('{');
    await confirmImport();
    expect(container.textContent).toContain('Invalid JSON: $');
    expect(mocks.importSemanticModel).not.toHaveBeenCalled();
  });

  it.each([
    ['null', 'Invalid JSON object: $'],
    ['1', 'Invalid JSON object: $'],
    ['{"entries":null}', 'Invalid entries: entries'],
    ['{"entries":[]}', 'Invalid entries: entries'],
    ['{"entries":[null]}', 'Invalid JSON object: entries[0]'],
    ['{"entries":[1]}', 'Invalid JSON object: entries[0]'],
    ['{"entries":[{"spec":null}]}', 'Invalid JSON object: entries[0].spec'],
    ['{"entries":[{"spec":1}]}', 'Invalid JSON object: entries[0].spec'],
    ['{"entries":[{"spec":[]}]}', 'Invalid JSON object: entries[0].spec'],
  ])('shows the local shape error for %s', async (input, expectedMessage) => {
    await setImportText(input);
    await confirmImport();

    expect(container.textContent).toContain(expectedMessage);
    expect(mocks.importSemanticModel).not.toHaveBeenCalled();
  });

  it('shows the API 400 message and a copyable request ID', async () => {
    mocks.importSemanticModel.mockRejectedValueOnce({
      response: {
        status: 400,
        headers: { 'x-request-id': 'req-400' },
        data: { msg: 'Entry metrics_1 references a missing table' },
      },
    });

    await setImportText('{"entries":[{"spec":{}}]}');
    await confirmImport();

    expect(container.textContent).toContain('Entry metrics_1 references a missing table');
    expect(container.textContent).toContain('Request ID: req-400');
  });

  it.each([401, 403])('maps HTTP %s to the permission error', async (status) => {
    mocks.importSemanticModel.mockRejectedValueOnce({ response: { status, data: { msg: 'raw permission failure' } } });

    await setImportText('{"entries":[{"spec":{}}]}');
    await confirmImport();

    expect(container.textContent).toContain('Permission denied');
    expect(container.textContent).not.toContain('raw permission failure');
  });

  it.each([{ response: { status: 500, data: { msg: 'raw server failure' } } }, new Error('network down')])(
    'maps unavailable failures to a generic message',
    async (error) => {
      mocks.importSemanticModel.mockRejectedValueOnce(error);

      await setImportText('{"entries":[{"spec":{}}]}');
      await confirmImport();

      expect(container.textContent).toContain('Service unavailable');
      expect(container.textContent).not.toContain('raw server failure');
      expect(container.textContent).not.toContain('network down');
    },
  );

  it('clears a previous error after a successful import', async () => {
    mocks.importSemanticModel.mockRejectedValueOnce({ response: { status: 500 } });
    await setImportText('{"entries":[{"spec":{}}]}');
    await confirmImport();
    expect(container.textContent).toContain('Service unavailable');

    mocks.importSemanticModel.mockResolvedValueOnce({ imported: 1 });
    await setImportText('{"entries":[{"spec":{}}]}');
    await confirmImport();

    expect(container.querySelector('[data-testid="semantic-config-import-error"]')).toBeNull();
    expect(container.querySelector('[data-testid="semantic-config-import-modal"]')).toBeNull();
    expect(mocks.onImportSuccess).toHaveBeenCalledOnce();
  });
});
