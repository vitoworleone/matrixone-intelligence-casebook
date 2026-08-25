import { describe, expect, it, vi } from 'vitest';

import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { answerSourceDisplayLabel, answerSourceIdentity, answerSourcePages, firstAnswerSourcePage } from './answerSourcePreview';
import type { ExploreA2AAnswerSourceRef } from './services/exploreA2AStreamParser';

type RoutedAnswerSourcePreview = {
  kind: 'source' | 'visual' | 'page_visual';
  fileId: string;
  fileName: string;
  volumeId: string;
  semanticModelId: number | null;
  sourceRowId: string;
  page: number | null;
  bbox: number[];
};

type AnswerSourcePreviewRoutingModule = typeof import('./answerSourcePreview') & {
  answerSourcePreviews: (source: ExploreA2AAnswerSourceRef) => RoutedAnswerSourcePreview[];
  answerInlineVisualPreviews: (sources: ExploreA2AAnswerSourceRef[]) => RoutedAnswerSourcePreview[];
  requestAnswerSourcePreviewBlob: (http: AppHttpClient, preview: RoutedAnswerSourcePreview) => Promise<Blob>;
};

function source(overrides: Partial<ExploreA2AAnswerSourceRef>): ExploreA2AAnswerSourceRef {
  return {
    type: 'rag_chunk',
    semanticModelId: null,
    sourceRowId: '',
    artifactId: '',
    chunkId: '',
    chunkIds: [],
    fileId: 'file_1',
    fileName: 'source.pdf',
    volumeId: '',
    markdownFileId: '',
    page: null,
    pages: [],
    database: '',
    table: '',
    sql: '',
    label: '',
    objectId: '',
    objectKind: '',
    imageFileId: '',
    pageImageFileId: '',
    bbox: [],
    visualRefs: [],
    ...overrides,
  };
}

async function previewRoutingModule(): Promise<AnswerSourcePreviewRoutingModule> {
  return (await import('./answerSourcePreview')) as unknown as AnswerSourcePreviewRoutingModule;
}

function mockHttp(blob = new Blob(['preview'])) {
  const get = vi.fn().mockResolvedValue({ data: blob, status: 200, headers: { 'content-type': 'image/png' } });
  const post = vi.fn().mockResolvedValue({ data: blob, status: 200, headers: { 'content-type': 'image/png' } });
  const http = {
    get,
    post,
    put: vi.fn(),
    delete: vi.fn(),
  } as unknown as AppHttpClient;
  return { http, get, post, blob };
}

describe('answer source preview page', () => {
  it('keeps pending sources with a shared file scoped to their knowledge model', () => {
    expect(answerSourceIdentity(101, 'source-1', 'shared.pdf')).not.toBe(answerSourceIdentity(202, 'source-2', 'shared.pdf'));
    expect(answerSourceIdentity(101, '', 'shared.pdf')).not.toBe(answerSourceIdentity(202, '', 'shared.pdf'));
  });

  it('keeps historical sources without a model on the file fallback', () => {
    expect(answerSourceIdentity(null, 'legacy-source-1', 'shared.pdf')).toBe(
      answerSourceIdentity(null, 'legacy-source-2', 'shared.pdf'),
    );
  });

  it('uses the first valid page from source pages', () => {
    const item = source({ pages: [8, 9], page: 3 });

    expect(answerSourcePages(item)).toEqual([8, 9]);
    expect(firstAnswerSourcePage(item)).toBe(8);
  });

  it('falls back to the single source page when pages are empty', () => {
    const item = source({ page: 5 });

    expect(answerSourcePages(item)).toEqual([5]);
    expect(firstAnswerSourcePage(item)).toBe(5);
  });

  it('does not invent a preview page from invalid values', () => {
    const item = source({ pages: [0, -1], page: null });

    expect(firstAnswerSourcePage(item)).toBeNull();
  });

  it('labels NL2SQL sources by table instead of sql result artifact id', () => {
    const item = source({
      type: 'sql_table',
      artifactId: 'sql_result_call_9fb9dec8f47e4c0c84b2a985',
      database: 'retail',
      table: 'top_products',
      fileId: '',
      fileName: '',
    });

    expect(answerSourceDisplayLabel(item, { rag: 'Document source', sql: 'Table source', image: 'Matched image' })).toBe(
      'retail.top_products',
    );
  });

  it('does not expose rag artifact ids as visible source labels', () => {
    const item = source({
      artifactId: 'rag_chunks_call_9fb9dec8f47e4c0c84b2a985',
      fileId: '',
      fileName: '',
      label: '',
    });

    expect(answerSourceDisplayLabel(item, { rag: 'Document source', sql: 'Table source', image: 'Matched image' })).toBe(
      'Document source',
    );
  });

  it('maps each visual ref to its own semantic model owner', async () => {
    const previewModule = await previewRoutingModule();
    expect(previewModule.answerSourcePreviews).toBeTypeOf('function');
    const item = source({
      fileId: 'mounted-source.pdf',
      volumeId: 'volume-1',
      visualRefs: [
        {
          chunkId: 'chunk-101',
          objectId: 'object-101',
          objectKind: 'drawing',
          imageFileId: 'shared-artifact',
          pageImageFileId: '',
          page: 1,
          bbox: [],
          semanticModelId: 101,
        },
        {
          chunkId: 'chunk-202',
          objectId: 'object-202',
          objectKind: 'drawing',
          imageFileId: '',
          pageImageFileId: 'shared-page-artifact',
          page: 2,
          bbox: [],
          semanticModelId: 202,
        },
      ],
    } as Partial<ExploreA2AAnswerSourceRef>);

    expect(previewModule.answerSourcePreviews(item)).toMatchObject([
      { kind: 'visual', fileId: 'shared-artifact', semanticModelId: 101 },
      { kind: 'page_visual', fileId: 'shared-page-artifact', semanticModelId: 202 },
      { kind: 'source', fileId: 'mounted-source.pdf', volumeId: 'volume-1', semanticModelId: null },
    ]);
  });

  it('uses the source owner only for a source-level visual fallback', async () => {
    const previewModule = await previewRoutingModule();
    expect(previewModule.answerSourcePreviews).toBeTypeOf('function');

    const previews = previewModule.answerSourcePreviews(
      source({
        semanticModelId: 101,
        imageFileId: 'source-level-artifact',
        visualRefs: [],
      } as Partial<ExploreA2AAnswerSourceRef>),
    );

    expect(previews.find((preview) => preview.kind === 'visual')).toMatchObject({
      fileId: 'source-level-artifact',
      semanticModelId: 101,
    });
  });

  it('does not borrow the parent owner for an explicit visual ref without ownership', async () => {
    const previewModule = await previewRoutingModule();
    expect(previewModule.answerSourcePreviews).toBeTypeOf('function');

    const previews = previewModule.answerSourcePreviews(
      source({
        semanticModelId: 101,
        imageFileId: '',
        visualRefs: [
          {
            chunkId: 'orphan-ref',
            objectId: 'orphan-object',
            objectKind: 'drawing',
            imageFileId: 'orphan-artifact',
            pageImageFileId: '',
            page: 1,
            bbox: [],
            semanticModelId: null,
          },
        ],
      } as Partial<ExploreA2AAnswerSourceRef>),
    );

    expect(previews.find((preview) => preview.kind === 'visual')).toMatchObject({
      fileId: 'orphan-artifact',
      semanticModelId: null,
    });
  });

  it('does not dedupe the same visual artifact id across semantic models', async () => {
    const previewModule = await previewRoutingModule();
    expect(previewModule.answerInlineVisualPreviews).toBeTypeOf('function');
    const shared = {
      fileId: '',
      imageFileId: 'shared-artifact',
      visualRefs: [
        {
          chunkId: 'shared-chunk',
          objectId: 'shared-object',
          objectKind: 'drawing',
          imageFileId: 'shared-artifact',
          pageImageFileId: '',
          page: 1,
          bbox: [],
        },
      ],
    };
    const first = source({
      ...shared,
      semanticModelId: 101,
      visualRefs: [{ ...shared.visualRefs[0], semanticModelId: 101 }],
    } as Partial<ExploreA2AAnswerSourceRef>);
    const second = source({
      ...shared,
      semanticModelId: 202,
      visualRefs: [{ ...shared.visualRefs[0], semanticModelId: 202 }],
    } as Partial<ExploreA2AAnswerSourceRef>);

    expect(previewModule.answerInlineVisualPreviews([first, second])).toMatchObject([
      { kind: 'visual', fileId: 'shared-artifact', semanticModelId: 101 },
      { kind: 'visual', fileId: 'shared-artifact', semanticModelId: 202 },
    ]);
  });

  it.each([
    { kind: 'visual' as const, semanticModelId: 101, fileId: 'shared-artifact' },
    { kind: 'visual' as const, semanticModelId: 202, fileId: 'shared-artifact' },
    { kind: 'page_visual' as const, semanticModelId: 303, fileId: 'shared-page-artifact' },
  ])('routes $kind artifacts through model $semanticModelId', async ({ kind, semanticModelId, fileId }) => {
    const previewModule = await previewRoutingModule();
    expect(previewModule.requestAnswerSourcePreviewBlob).toBeTypeOf('function');
    const { http, get, post, blob } = mockHttp();

    await expect(
      previewModule.requestAnswerSourcePreviewBlob(http, {
        kind,
        fileId,
        fileName: `${fileId}.png`,
        volumeId: '',
        semanticModelId,
        page: 1,
        bbox: [],
      }),
    ).resolves.toBe(blob);

    expect(get).toHaveBeenCalledWith(
      `/semantic-models/${semanticModelId}/artifacts/${fileId}/preview`,
      expect.objectContaining({ responseType: 'blob', responseContentType: 'blob' }),
    );
    expect(post).not.toHaveBeenCalled();
  });

  it('keeps mounted source previews on the Catalog preview route', async () => {
    const previewModule = await previewRoutingModule();
    expect(previewModule.requestAnswerSourcePreviewBlob).toBeTypeOf('function');
    const { http, get, post, blob } = mockHttp();

    await expect(
      previewModule.requestAnswerSourcePreviewBlob(http, {
        kind: 'source',
        fileId: 'mounted-source.pdf',
        fileName: 'mounted-source.pdf',
        volumeId: 'volume-1',
        semanticModelId: null,
        sourceRowId: '',
        page: 1,
        bbox: [],
      }),
    ).resolves.toBe(blob);

    expect(post).toHaveBeenCalledWith(
      '/catalog/file/preview_stream',
      { file_id: 'mounted-source.pdf', volume_id: 'volume-1' },
      expect.objectContaining({ responseType: 'blob', responseContentType: 'blob' }),
    );
    expect(get).not.toHaveBeenCalled();
  });

  it('routes knowledge-base source previews through the model-owned source file route', async () => {
    const previewModule = await previewRoutingModule();
    const { http, get, post, blob } = mockHttp();

    await expect(
      previewModule.requestAnswerSourcePreviewBlob(http, {
        kind: 'source',
        fileId: '20C114830.pdf',
        fileName: '20C114830.pdf',
        volumeId: 'volume-1',
        semanticModelId: 42,
        sourceRowId: 'source-row-1',
        page: 1,
        bbox: [],
      }),
    ).resolves.toBe(blob);

    expect(get).toHaveBeenCalledWith(
      '/semantic-models/42/sources/file/20C114830.pdf/preview',
      expect.objectContaining({ responseType: 'blob', responseContentType: 'blob' }),
    );
    expect(post).not.toHaveBeenCalled();
  });

  it('fails closed without issuing HTTP when a visual artifact has no model owner', async () => {
    const previewModule = await previewRoutingModule();
    expect(previewModule.requestAnswerSourcePreviewBlob).toBeTypeOf('function');
    const { http, get, post } = mockHttp();

    await expect(
      previewModule.requestAnswerSourcePreviewBlob(http, {
        kind: 'page_visual',
        fileId: 'orphan-artifact',
        fileName: 'orphan-artifact.png',
        volumeId: '',
        semanticModelId: null,
        page: 1,
        bbox: [],
      }),
    ).rejects.toThrow(/owner|semantic model/i);

    expect(get).not.toHaveBeenCalled();
    expect(post).not.toHaveBeenCalled();
  });

  it.each([
    { kind: 'visual' as const, semanticModelId: null },
    { kind: 'visual' as const, semanticModelId: 0 },
    { kind: 'visual' as const, semanticModelId: -1 },
    { kind: 'visual' as const, semanticModelId: 1.5 },
    { kind: 'visual' as const, semanticModelId: Number.NaN },
    { kind: 'page_visual' as const, semanticModelId: null },
    { kind: 'page_visual' as const, semanticModelId: 0 },
    { kind: 'page_visual' as const, semanticModelId: -1 },
    { kind: 'page_visual' as const, semanticModelId: 1.5 },
    { kind: 'page_visual' as const, semanticModelId: Number.NaN },
  ])('rejects $kind owner $semanticModelId before HTTP', async ({ kind, semanticModelId }) => {
    const previewModule = await previewRoutingModule();
    expect(previewModule.requestAnswerSourcePreviewBlob).toBeTypeOf('function');
    const { http, get, post } = mockHttp();

    await expect(
      previewModule.requestAnswerSourcePreviewBlob(http, {
        kind,
        fileId: 'invalid-owner-artifact',
        fileName: 'invalid-owner-artifact.png',
        volumeId: '',
        semanticModelId,
        page: 1,
        bbox: [],
      }),
    ).rejects.toThrow(/owner|semantic model/i);

    expect(get).not.toHaveBeenCalled();
    expect(post).not.toHaveBeenCalled();
  });
});
