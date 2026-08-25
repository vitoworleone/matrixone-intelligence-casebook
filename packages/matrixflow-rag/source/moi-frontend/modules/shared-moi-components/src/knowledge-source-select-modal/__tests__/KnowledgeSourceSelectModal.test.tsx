import React, { act, type ComponentProps } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { App } from 'antd';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { DataTypeSelector } from '@moi/shared-moi-components/data-type-selector';
import KnowledgeSourceSelectModal from '../KnowledgeSourceSelectModal';
import styles from '../KnowledgeSourceSelectModal.module.css';

type DataTypeSelectorProps = ComponentProps<typeof DataTypeSelector>;
const mockDataTypeSelector = vi.fn((_props: DataTypeSelectorProps) => <div data-testid="knowledge-source-data-type-selector" />);

vi.mock('@moi/shared-moi-components/data-type-selector', () => ({
  DataTypeSelector: (props: DataTypeSelectorProps) => mockDataTypeSelector(props),
  isPrimaryKeyType: (dataType: string) => !['TEXT', 'BLOB', 'VECF32', 'VECF64', 'JSON', 'DATALINK'].includes(dataType),
}));

async function flushStepFocus() {
  await act(async () => {
    await new Promise<void>((resolve) => window.requestAnimationFrame(() => resolve()));
  });
}

function createDirectoryDropEvent() {
  const event = new Event('drop', { bubbles: true, cancelable: true });
  Object.defineProperty(event, 'dataTransfer', {
    value: {
      items: [{ webkitGetAsEntry: () => ({ isDirectory: true }) }],
    },
  });
  return event;
}

describe('KnowledgeSourceSelectModal keyboard focus', () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    (globalThis as unknown as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT = true;
    mockDataTypeSelector.mockClear();
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
    document.body.innerHTML = '';
  });

  it('moves focus to the new step heading and restores the name input when going back', async () => {
    await act(async () => {
      root.render(
        <App>
          <KnowledgeSourceSelectModal
            open
            title="Create knowledge base"
            okText="Create"
            cancelText="Cancel"
            basicNextText="Next"
            showCreateSteps
            basicStepContent={<input aria-label="Knowledge base name" />}
            http={{ post: vi.fn() }}
            translate={(key) => key}
            testIdPrefix="focus-kb"
            onCancel={vi.fn()}
            onBasicNext={async () => true}
            onSubmit={vi.fn()}
          />
        </App>,
      );
    });
    await flushStepFocus();

    const nameInput = document.querySelector<HTMLInputElement>('input[aria-label="Knowledge base name"]');
    expect(document.activeElement).toBe(nameInput);

    await act(async () => {
      document.querySelector<HTMLButtonElement>('[data-testid="focus-kb-base-next-btn"]')?.click();
    });
    await flushStepFocus();
    expect(document.activeElement).toBe(document.querySelector('[data-testid="focus-kb-active-step-heading"]'));

    await act(async () => {
      document.querySelector<HTMLButtonElement>('[data-testid="focus-kb-source-back-btn"]')?.click();
    });
    await flushStepFocus();
    expect(document.activeElement).toBe(document.querySelector('input[aria-label="Knowledge base name"]'));
  });

  it('blocks directory drops and tells users to use the folder selector', async () => {
    const folderDropMessage = 'Dragging folders is not supported. Use Select Folder.';
    await act(async () => {
      root.render(
        <App>
          <KnowledgeSourceSelectModal
            open
            title="Create knowledge base"
            okText="Create"
            cancelText="Cancel"
            allowedSourceModes={['local']}
            http={{ post: vi.fn() }}
            translate={(key) => (key === 'knowledge.base.local-folder-drop-unsupported' ? folderDropMessage : key)}
            testIdPrefix="directory-drop-kb"
            onCancel={vi.fn()}
            onSubmit={vi.fn()}
          />
        </App>,
      );
    });

    const dropZone = document.querySelector<HTMLElement>('[data-testid="directory-drop-kb-local-files-drop-zone"]');
    expect(dropZone).not.toBeNull();
    const directoryDropEvent = createDirectoryDropEvent();

    await act(async () => {
      dropZone?.dispatchEvent(directoryDropEvent);
      await Promise.resolve();
    });

    expect(directoryDropEvent.defaultPrevented).toBe(true);
    expect(document.body.textContent).toContain(folderDropMessage);
  });

  it('blocks directory drops in structured mode', async () => {
    const folderDropMessage = 'Dragging folders is not supported. Use Select Folder.';
    await act(async () => {
      root.render(
        <App>
          <KnowledgeSourceSelectModal
            open
            title="Create knowledge base"
            okText="Create"
            cancelText="Cancel"
            allowedSourceModes={['local']}
            http={{ post: vi.fn() }}
            translate={(key) => (key === 'knowledge.base.local-folder-drop-unsupported' ? folderDropMessage : key)}
            testIdPrefix="structured-directory-drop-kb"
            onCancel={vi.fn()}
            onSubmit={vi.fn()}
          />
        </App>,
      );
    });

    await act(async () => {
      document.querySelector<HTMLButtonElement>('[data-testid="structured-directory-drop-kb-load-kind-structured"]')?.click();
    });

    const dropZone = document.querySelector<HTMLElement>('[data-testid="structured-directory-drop-kb-local-files-drop-zone"]');
    expect(dropZone).not.toBeNull();
    const directoryDropEvent = createDirectoryDropEvent();

    await act(async () => {
      dropZone?.dispatchEvent(directoryDropEvent);
      await Promise.resolve();
    });

    expect(directoryDropEvent.defaultPrevented).toBe(true);
    expect(document.body.textContent).toContain(folderDropMessage);
  });

  it('uses the shared data type selector for structured file columns', async () => {
    await act(async () => {
      root.render(
        <App>
          <KnowledgeSourceSelectModal
            open
            title="Create knowledge base"
            okText="Create"
            cancelText="Cancel"
            allowedSourceModes={['local']}
            http={{ post: vi.fn() }}
            translate={(key) => key}
            testIdPrefix="structured-kb"
            onCancel={vi.fn()}
            onSubmit={vi.fn()}
          />
        </App>,
      );
    });

    const structuredButton = document.querySelector<HTMLButtonElement>('[data-testid="structured-kb-load-kind-structured"]');
    if (!structuredButton) throw new Error('missing structured upload button');
    await act(async () => {
      structuredButton.click();
    });

    expect(document.querySelectorAll('[data-testid="structured-kb-structured-column-type-select"]')).toHaveLength(2);
    expect(document.querySelectorAll(`.${styles.structuredColumnMobileLabel}`)).toHaveLength(12);
    expect(mockDataTypeSelector).toHaveBeenCalledWith(
      expect.objectContaining({
        value: 'VARCHAR',
        precision: [255],
        categoryLabels: {
          INTEGER: 'knowledge.base.create-document-sources-structured-data-type-category-integer',
          FLOAT: 'knowledge.base.create-document-sources-structured-data-type-category-float',
          STRING: 'knowledge.base.create-document-sources-structured-data-type-category-string',
          DATETIME: 'knowledge.base.create-document-sources-structured-data-type-category-datetime',
          DECIMAL: 'knowledge.base.create-document-sources-structured-data-type-category-decimal',
          VECTOR: 'knowledge.base.create-document-sources-structured-data-type-category-vector',
          BOOLEAN: 'knowledge.base.create-document-sources-structured-data-type-category-boolean',
          OTHER: 'knowledge.base.create-document-sources-structured-data-type-category-other',
        },
        placeholder: 'knowledge.base.create-document-sources-structured-data-type-placeholder',
        disabled: false,
      }),
    );

    const selectorProps = mockDataTypeSelector.mock.calls[0]?.[0];
    if (!selectorProps) throw new Error('missing data type selector props');
    await act(async () => {
      selectorProps.onPrecisionChange?.([18, 2]);
      selectorProps.onChange?.('DECIMAL');
    });

    expect(mockDataTypeSelector).toHaveBeenCalledWith(
      expect.objectContaining({ value: 'DECIMAL', precision: [18, 2], disabled: false }),
    );

    const primaryKeyCheckbox = document.querySelector<HTMLInputElement>(
      'input[data-testid="structured-kb-structured-column-key-checkbox"], [data-testid="structured-kb-structured-column-key-checkbox"] input',
    );
    if (!primaryKeyCheckbox) throw new Error('missing primary key checkbox');
    expect(primaryKeyCheckbox.getAttribute('aria-label')).toBe(
      'knowledge.base.create-document-sources-structured-column-primary-key',
    );
    await act(async () => {
      primaryKeyCheckbox.click();
    });

    const decimalSelectorProps = mockDataTypeSelector.mock.calls
      .map(([props]) => props)
      .find((props) => props.value === 'DECIMAL');
    if (!decimalSelectorProps) throw new Error('missing decimal data type selector props');
    await act(async () => {
      decimalSelectorProps.onPrecisionChange?.(undefined);
      decimalSelectorProps.onChange?.('JSON');
    });

    expect(mockDataTypeSelector).toHaveBeenCalledWith(
      expect.objectContaining({ value: 'JSON', precision: undefined, disabled: false }),
    );
    expect(primaryKeyCheckbox.checked).toBe(false);
    expect(primaryKeyCheckbox.disabled).toBe(true);
  });
});
