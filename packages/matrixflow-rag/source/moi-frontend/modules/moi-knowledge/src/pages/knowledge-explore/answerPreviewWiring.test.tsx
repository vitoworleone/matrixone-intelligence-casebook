import { act, type ReactElement } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { AnswerInlineVisualRefs, AnswerSourceRefs } from './index';
import {
  createExploreA2AProjection,
  reduceExploreA2AResponse,
  type ExploreA2AAnswerSourceRef,
} from './services/exploreA2AStreamParser';

const httpMocks = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
}));

vi.mock('@moi/shared-moi-app-protocol/app-context', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@moi/shared-moi-app-protocol/app-context')>();
  return {
    ...actual,
    useHttpClient: () => httpMocks as unknown as AppHttpClient,
  };
});

vi.mock('@moi/shared-moi-app-protocol/business-context', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@moi/shared-moi-app-protocol/business-context')>();
  return {
    ...actual,
    useWorkspaceId: () => 'workspace-1',
  };
});

vi.mock('@moi/shared-moi-components/file-preview', () => ({
  FilePreviewModal: ({ fetchBlob, initialPage }: { fetchBlob: () => Promise<Blob>; initialPage?: number }) => (
    <button
      type="button"
      data-testid="answer-source-modal-fetch"
      data-page={initialPage}
      onClick={() => {
        fetchBlob().catch(() => {});
      }}
    >
      fetch modal
    </button>
  ),
}));

vi.mock('./previewBlobCache', () => ({
  getCachedPreviewBlob: (_key: unknown, load: () => Promise<Blob>) => load(),
}));

function visualSource(input: {
  semanticModelId: number | null;
  imageFileId?: string;
  pageImageFileId?: string;
}): ExploreA2AAnswerSourceRef {
  return {
    type: 'visual_hit',
    sourceRowId: '',
    artifactId: 'visual-search-1',
    chunkId: 'chunk-1',
    chunkIds: ['chunk-1'],
    fileId: '',
    fileName: 'drawing.pdf',
    volumeId: '',
    markdownFileId: '',
    page: 1,
    pages: [1],
    database: '',
    table: '',
    sql: '',
    label: 'drawing',
    objectId: 'object-1',
    objectKind: 'drawing',
    imageFileId: input.imageFileId ?? '',
    pageImageFileId: input.pageImageFileId ?? '',
    bbox: [1, 2, 3, 4],
    semanticModelId: input.semanticModelId,
    visualRefs: [
      {
        chunkId: 'chunk-1',
        objectId: 'object-1',
        objectKind: 'drawing',
        imageFileId: input.imageFileId ?? '',
        pageImageFileId: input.pageImageFileId ?? '',
        page: 1,
        bbox: [1, 2, 3, 4],
        semanticModelId: input.semanticModelId,
      },
    ],
  } as ExploreA2AAnswerSourceRef;
}

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

function projectedVisualSources(input: {
  semanticModelId: number;
  imageFileId?: string;
  pageImageFileId?: string;
  sourceRowId?: string;
  sourceFileId?: string;
}): ExploreA2AAnswerSourceRef[] {
  const projection = reduceExploreA2AResponse(
    createExploreA2AProjection(),
    {
      jsonrpc: '2.0',
      id: 'request-1',
      result: {
        kind: 'artifact-update',
        taskId: 'task-1',
        artifact: {
          artifactId: 'answer-task-1',
          name: 'answer',
          parts: [{ kind: 'text', text: '匹配结果' }],
          metadata: {
            matrixflow_type: 'knowledge.answer',
            source_refs: [
              {
                type: 'visual_hit',
                semantic_model_id: input.semanticModelId,
                source_row_id: input.sourceRowId ?? '',
                file_id: input.sourceFileId ?? '',
                file_name: input.sourceFileId ?? '',
                object_id: 'object-1',
                image_file_id: input.imageFileId ?? '',
                page_image_file_id: input.pageImageFileId ?? '',
                visual_refs: [
                  {
                    semantic_model_id: input.semanticModelId,
                    object_id: 'object-1',
                    image_file_id: input.imageFileId ?? '',
                    page_image_file_id: input.pageImageFileId ?? '',
                    page: 1,
                  },
                ],
              },
            ],
          },
        },
      },
    } as never,
    1,
  );
  return projection.answerSources;
}

beforeEach(() => {
  container = document.createElement('div');
  document.body.appendChild(container);
  root = createRoot(container);
  httpMocks.get.mockReset().mockResolvedValue({
    data: new Blob(['preview']),
    status: 200,
    headers: { 'content-type': 'image/png' },
  });
  httpMocks.post.mockReset().mockResolvedValue({
    data: new Blob(['preview']),
    status: 200,
    headers: { 'content-type': 'image/png' },
  });
  httpMocks.put.mockReset();
  httpMocks.delete.mockReset();
  Object.defineProperty(URL, 'createObjectURL', {
    configurable: true,
    value: vi.fn(() => 'blob:answer-preview'),
  });
  Object.defineProperty(URL, 'revokeObjectURL', {
    configurable: true,
    value: vi.fn(),
  });
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

async function renderIntoDocument(element: ReactElement): Promise<HTMLDivElement> {
  await act(async () => {
    root?.render(element);
    await Promise.resolve();
  });
  if (!container) {
    throw new Error('test container is not initialized');
  }
  return container;
}

async function click(element: Element | null): Promise<void> {
  if (!(element instanceof HTMLElement)) {
    throw new Error('expected a clickable element');
  }
  await act(async () => {
    element.click();
    await Promise.resolve();
  });
}

async function waitForAssertion(assertion: () => void): Promise<void> {
  await vi.waitFor(assertion, { timeout: 1_000, interval: 10 });
}

describe('knowledge explore answer artifact preview wiring', () => {
  it('renders inline visual hits through the model-scoped endpoint', async () => {
    await renderIntoDocument(
      <AnswerInlineVisualRefs
        sources={projectedVisualSources({ semanticModelId: 101, imageFileId: 'image-1' })}
        t={(key) => key}
      />,
    );

    await waitForAssertion(() => {
      expect(httpMocks.get).toHaveBeenCalledWith(
        '/semantic-models/101/artifacts/image-1/preview',
        expect.objectContaining({ responseType: 'blob', responseContentType: 'blob' }),
      );
    });
    expect(httpMocks.post).not.toHaveBeenCalled();
  });

  it('uses the model-scoped endpoint for both page-visual tiles and their modal', async () => {
    const rendered = await renderIntoDocument(
      <AnswerSourceRefs
        sources={projectedVisualSources({ semanticModelId: 202, pageImageFileId: 'page-image-1' })}
        t={(key) => key}
      />,
    );

    await waitForAssertion(() => {
      expect(httpMocks.get).toHaveBeenCalledWith(
        '/semantic-models/202/artifacts/page-image-1/preview',
        expect.objectContaining({ responseType: 'blob', responseContentType: 'blob' }),
      );
    });
    const tileRequestCount = httpMocks.get.mock.calls.length;
    await click(rendered.querySelector('button'));
    await waitForAssertion(() => {
      expect(document.body.querySelector('[data-testid="answer-source-modal-fetch"]')).not.toBeNull();
    });
    await click(document.body.querySelector('[data-testid="answer-source-modal-fetch"]'));
    await waitForAssertion(() => expect(httpMocks.get.mock.calls.length).toBeGreaterThan(tileRequestCount));

    expect(httpMocks.get.mock.calls).toEqual(
      expect.arrayContaining([
        [
          '/semantic-models/202/artifacts/page-image-1/preview',
          expect.objectContaining({ responseType: 'blob', responseContentType: 'blob' }),
        ],
      ]),
    );
    expect(httpMocks.post).not.toHaveBeenCalled();
  });

  it('keeps grouped same-id visual tiles isolated by semantic model', async () => {
    const first = {
      ...visualSource({ semanticModelId: 101, imageFileId: 'shared-artifact' }),
      fileId: 'shared-source',
      volumeId: 'volume-1',
    };
    const second = {
      ...visualSource({ semanticModelId: 202, imageFileId: 'shared-artifact' }),
      fileId: 'shared-source',
      volumeId: 'volume-1',
    };

    await renderIntoDocument(<AnswerSourceRefs sources={[first, second]} t={(key) => key} />);

    await waitForAssertion(() => {
      expect(httpMocks.get).toHaveBeenCalledWith(
        '/semantic-models/101/artifacts/shared-artifact/preview',
        expect.objectContaining({ responseType: 'blob', responseContentType: 'blob' }),
      );
      expect(httpMocks.get).toHaveBeenCalledWith(
        '/semantic-models/202/artifacts/shared-artifact/preview',
        expect.objectContaining({ responseType: 'blob', responseContentType: 'blob' }),
      );
    });
    expect(httpMocks.get).toHaveBeenCalledTimes(2);
    expect(httpMocks.post).not.toHaveBeenCalled();
  });

  it('routes a knowledge-base source modal through the model-owned source endpoint', async () => {
    const projection = reduceExploreA2AResponse(
      createExploreA2AProjection(),
      {
        jsonrpc: '2.0',
        id: 'request-1',
        result: {
          kind: 'artifact-update',
          taskId: 'task-1',
          artifact: {
            artifactId: 'answer-task-1',
            name: 'answer',
            parts: [{ kind: 'text', text: '匹配结果' }],
            metadata: {
              matrixflow_type: 'knowledge.answer',
              source_refs: [
                {
                  type: 'rag_chunk',
                  semantic_model_id: 42,
                  source_row_id: 'source-row-9',
                  file_id: 'mounted-source.pdf',
                  file_name: 'mounted-source.pdf',
                  volume_id: 'volume-9',
                },
              ],
            },
          },
        },
      } as never,
      1,
    );

    const rendered = await renderIntoDocument(<AnswerSourceRefs sources={projection.answerSources} t={(key) => key} />);
    await click(rendered.querySelector('button'));
    await waitForAssertion(() => {
      expect(document.body.querySelector('[data-testid="answer-source-modal-fetch"]')).not.toBeNull();
    });
    await click(document.body.querySelector('[data-testid="answer-source-modal-fetch"]'));

    await waitForAssertion(() => {
      expect(httpMocks.get).toHaveBeenCalledWith(
        '/semantic-models/42/sources/file/mounted-source.pdf/preview',
        expect.objectContaining({ responseType: 'blob', responseContentType: 'blob' }),
      );
    });
    expect(httpMocks.post).not.toHaveBeenCalled();
  });

  it('keeps same-file knowledge-base sources isolated by model', async () => {
    const projection = reduceExploreA2AResponse(
      createExploreA2AProjection(),
      {
        jsonrpc: '2.0',
        id: 'request-1',
        result: {
          kind: 'artifact-update',
          taskId: 'task-1',
          artifact: {
            artifactId: 'answer-task-1',
            name: 'answer',
            parts: [{ kind: 'text', text: '匹配结果' }],
            metadata: {
              matrixflow_type: 'knowledge.answer',
              source_refs: [
                {
                  type: 'rag_chunk',
                  semantic_model_id: 101,
                  source_row_id: 'source-row-101',
                  file_id: 'shared.pdf',
                  file_name: 'model-101.pdf',
                  pages: [1],
                },
                {
                  type: 'rag_chunk',
                  semantic_model_id: 202,
                  source_row_id: 'source-row-202',
                  file_id: 'shared.pdf',
                  file_name: 'model-202.pdf',
                  pages: [2],
                },
              ],
            },
          },
        },
      } as never,
      1,
    );

    const rendered = await renderIntoDocument(<AnswerSourceRefs sources={projection.answerSources} t={(key) => key} />);
    const sourceButtons = Array.from(rendered.querySelectorAll('button')).filter((button) =>
      button.textContent?.includes('model-'),
    );
    expect(sourceButtons).toHaveLength(2);

    await click(sourceButtons[0]);
    const modalFetch = document.body.querySelector('[data-testid="answer-source-modal-fetch"]');
    expect(modalFetch?.getAttribute('data-page')).toBe('1');
    await click(modalFetch);
    await click(sourceButtons[1]);
    const secondModalFetch = document.body.querySelector('[data-testid="answer-source-modal-fetch"]');
    expect(secondModalFetch?.getAttribute('data-page')).toBe('2');
    await click(secondModalFetch);

    await waitForAssertion(() => {
      expect(httpMocks.get).toHaveBeenCalledWith(
        '/semantic-models/101/sources/file/shared.pdf/preview',
        expect.objectContaining({ responseType: 'blob', responseContentType: 'blob' }),
      );
      expect(httpMocks.get).toHaveBeenCalledWith(
        '/semantic-models/202/sources/file/shared.pdf/preview',
        expect.objectContaining({ responseType: 'blob', responseContentType: 'blob' }),
      );
    });
    expect(httpMocks.post).not.toHaveBeenCalled();
  });

  it('routes a visual-hit source document through its model-owned source endpoint', async () => {
    const rendered = await renderIntoDocument(
      <AnswerSourceRefs
        sources={projectedVisualSources({
          semanticModelId: 303,
          imageFileId: 'visual-image-1',
          sourceRowId: 'visual-source-row-1',
          sourceFileId: 'visual-source.pdf',
        })}
        t={(key) => key}
      />,
    );

    const sourceButton = Array.from(rendered.querySelectorAll('button')).find((button) =>
      button.textContent?.includes('visual-source.pdf'),
    );
    await click(sourceButton ?? null);
    await click(document.body.querySelector('[data-testid="answer-source-modal-fetch"]'));

    await waitForAssertion(() => {
      expect(httpMocks.get).toHaveBeenCalledWith(
        '/semantic-models/303/sources/file/visual-source.pdf/preview',
        expect.objectContaining({ responseType: 'blob', responseContentType: 'blob' }),
      );
    });
    expect(httpMocks.post).not.toHaveBeenCalled();
  });

  it('renders an unavailable inline visual without any HTTP when owner is missing', async () => {
    const rendered = await renderIntoDocument(
      <AnswerInlineVisualRefs
        sources={[visualSource({ semanticModelId: null, imageFileId: 'orphan-image' })]}
        t={(key) => key}
      />,
    );

    await waitForAssertion(() => {
      expect(rendered.textContent).toContain('knowledge.explore.source-preview-unsupported');
    });
    expect(httpMocks.get).not.toHaveBeenCalled();
    expect(httpMocks.post).not.toHaveBeenCalled();
  });
});
