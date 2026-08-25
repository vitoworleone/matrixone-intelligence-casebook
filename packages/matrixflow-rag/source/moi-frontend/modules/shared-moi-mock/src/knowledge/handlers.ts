import { http, HttpResponse, type HttpHandler } from 'msw';

import type {
  AppendSemanticModelSourcesResponse,
  CreateEmptySemanticModelRequest,
  CreateEmptySemanticModelResponse,
  CreateSemanticModelWithSourcesRequest,
  CreateSemanticModelWithSourcesResponse,
  SemanticModelDocumentSegment,
  SemanticModelListResponse,
  SemanticModelSegmentMutationBase,
  SemanticModelSource,
  SemanticModelSourceDocument,
  SemanticModelSourceListResponse,
  UpdateSemanticModelSourceGovernanceRequest,
} from '@moi/shared-moi-api/knowledge';
import { mockResponse } from '../utils/mockResponse';
import {
  createKnowledgeAppendSourcesResponseFixture,
  createKnowledgeCreateResponseFixture,
  createKnowledgeModelFixture,
  createKnowledgeSegmentFixture,
  createKnowledgeSourceDocumentFixture,
  createKnowledgeSourceFixture,
} from './factories';

export interface KnowledgeMockData {
  models: SemanticModelListResponse;
  createResponse: CreateSemanticModelWithSourcesResponse;
  appendResponse: AppendSemanticModelSourcesResponse;
  createFailure?: {
    status: number;
    code: string;
    msg: string;
  };
  appendFailure?: {
    status: number;
    code: string;
    msg: string;
  };
  legacySources?: SemanticModelSource[];
  legacyBackfillBatchSize?: number;
  legacyBackfillRequired?: boolean;
}

const defaultCreateResponse = createKnowledgeCreateResponseFixture();
const defaultAppendResponse = createKnowledgeAppendSourcesResponseFixture({
  data_domain: { ...defaultCreateResponse.data_domain },
  sources: [],
  jobs: [],
});

const defaultData: KnowledgeMockData = {
  models: {
    items: [createKnowledgeModelFixture({ id: 8001, name: 'Existing Knowledge Base' })],
    total: 1,
    next_page_token: '',
  },
  createResponse: defaultCreateResponse,
  appendResponse: defaultAppendResponse,
};

function segmentBaseConflict(document: SemanticModelSourceDocument, body: SemanticModelSegmentMutationBase): boolean {
  return (
    (document.current_segment_version_id ?? '') !== body.base_segment_version_id ||
    (document.current_index_version ?? 0) !== body.base_index_version
  );
}

function nextSegmentDocument(
  document: SemanticModelSourceDocument,
  segments: SemanticModelDocumentSegment[],
  source = 'edit_chunk',
): SemanticModelSourceDocument {
  const nextIndexVersion = (document.current_index_version ?? 0) + 1;
  const nextVersionID = `segment-v${nextIndexVersion}`;
  const nextSegments = segments.map((segment, index) => ({
    ...segment,
    segment_id: `segment-${nextIndexVersion}-${index}`,
    word_count: (segment.content ?? '').length + (segment.ocr_text ?? '').length + (segment.image_description ?? '').length,
  }));
  return createKnowledgeSourceDocumentFixture({
    source: {
      ...document.source,
      segment_version_id: nextVersionID,
      index_version: nextIndexVersion,
    },
    current_segment_version_id: nextVersionID,
    current_index_version: nextIndexVersion,
    selected_segment_version_id: nextVersionID,
    selected_index_version: nextIndexVersion,
    segments: nextSegments,
    segment_versions: [
      {
        version_id: nextVersionID,
        current: true,
        index_version: nextIndexVersion,
        status: 'committed',
        source,
        chunk_count: nextSegments.length,
        enabled_chunk_count: nextSegments.filter((segment) => segment.enabled).length,
      },
      ...document.segment_versions.map((version) => ({ ...version, current: false })),
    ],
  });
}

export function createKnowledgeHandlers(overrides: Partial<KnowledgeMockData> = {}, pathPrefix = ''): HttpHandler[] {
  const data: KnowledgeMockData = {
    ...defaultData,
    ...overrides,
  };
  const sourcesByModel = new Map<number, typeof data.createResponse.sources>();
  const jobsByModel = new Map<number, typeof data.createResponse.jobs>();
  const documentsBySource = new Map<string, SemanticModelSourceDocument>();
  const legacySourcesByModel = new Map<number, SemanticModelSource[]>();
  for (const source of data.createResponse.sources) {
    sourcesByModel.set(source.model_id, [...(sourcesByModel.get(source.model_id) ?? []), source]);
    if (source.source_type !== 'table') {
      documentsBySource.set(source.row_id, createKnowledgeSourceDocumentFixture({ source }));
    }
  }
  jobsByModel.set(data.createResponse.model.id, [...data.createResponse.jobs]);
  for (const source of data.legacySources ?? []) {
    legacySourcesByModel.set(source.model_id, [
      ...(legacySourcesByModel.get(source.model_id) ?? []),
      { ...source, governance_status: 'legacy_unbound' },
    ]);
  }

  return [
    http.get(`${pathPrefix}/semantic-models`, () => HttpResponse.json(mockResponse(data.models))),
    http.post(`${pathPrefix}/semantic-models/create-empty`, async ({ request }) => {
      if (data.createFailure) {
        return HttpResponse.json(
          { code: data.createFailure.code, msg: data.createFailure.msg, data: null },
          { status: data.createFailure.status },
        );
      }
      const body = (await request.json().catch(() => ({}))) as Partial<CreateEmptySemanticModelRequest>;
      const response: CreateEmptySemanticModelResponse = {
        model: {
          ...data.createResponse.model,
          name: body.name ?? data.createResponse.model.name,
          description: body.description ?? data.createResponse.model.description,
          tables: [],
          files: { ...data.createResponse.model.files, file_ids: [] },
          source_counts: { files: 0, tables: 0, total: 0 },
        },
        data_domain: data.createResponse.data_domain,
      };
      return HttpResponse.json(mockResponse(response));
    }),
    http.post(`${pathPrefix}/semantic-models/create-with-sources`, async ({ request }) => {
      if (data.createFailure) {
        return HttpResponse.json(
          { code: data.createFailure.code, msg: data.createFailure.msg, data: null },
          { status: data.createFailure.status },
        );
      }
      const body = (await request.json().catch(() => ({}))) as Partial<CreateSemanticModelWithSourcesRequest>;
      const responseFiles = data.createResponse.model.files;
      const modelFiles = body.files
        ? {
            ...body.files,
            file_ids: responseFiles?.file_ids ?? body.files.file_ids ?? [],
          }
        : responseFiles;
      return HttpResponse.json(
        mockResponse({
          ...data.createResponse,
          model: {
            ...data.createResponse.model,
            files: modelFiles,
          },
        }),
      );
    }),
    http.post(`${pathPrefix}/semantic-models/:modelId/sources`, ({ params }) => {
      if (data.appendFailure) {
        return HttpResponse.json(
          { code: data.appendFailure.code, msg: data.appendFailure.msg, data: null },
          { status: data.appendFailure.status },
        );
      }
      const modelId = Number(params.modelId);
      const appendResponse =
        data.appendResponse.data_domain.model_id === modelId
          ? data.appendResponse
          : createKnowledgeAppendSourcesResponseFixture({
              ...data.appendResponse,
              data_domain: { ...data.appendResponse.data_domain, model_id: modelId },
              sources: data.appendResponse.sources.map((source) => ({ ...source, model_id: modelId })),
              jobs: data.appendResponse.jobs,
            });
      sourcesByModel.set(modelId, [...(sourcesByModel.get(modelId) ?? []), ...appendResponse.sources]);
      for (const source of appendResponse.sources) {
        if (source.source_type !== 'table') {
          documentsBySource.set(source.row_id, createKnowledgeSourceDocumentFixture({ source }));
        }
      }
      jobsByModel.set(modelId, [...(jobsByModel.get(modelId) ?? []), ...appendResponse.jobs]);
      return HttpResponse.json(mockResponse(appendResponse));
    }),
    http.get(`${pathPrefix}/semantic-models/:modelId/sources`, ({ params }) => {
      const modelId = Number(params.modelId);
      const sources = sourcesByModel.get(modelId) ?? [];
      const legacySources = legacySourcesByModel.get(modelId) ?? [];
      const items = [...sources, ...legacySources];
      const response: SemanticModelSourceListResponse = {
        items,
        total: items.length,
        legacy_backfill_required: Boolean(data.legacyBackfillRequired || legacySources.length > 0),
      };
      return HttpResponse.json(mockResponse(response));
    }),
    http.get(`${pathPrefix}/semantic-models/:modelId/sources/:sourceRowId/document`, ({ params }) => {
      const modelId = Number(params.modelId);
      const sourceRowId = decodeURIComponent(String(params.sourceRowId));
      const source = (sourcesByModel.get(modelId) ?? []).find((item) => item.row_id === sourceRowId);
      if (!source || source.source_type === 'table') {
        return HttpResponse.json({ code: 'ErrNotFound', msg: 'source document not found', data: null }, { status: 404 });
      }
      return HttpResponse.json(
        mockResponse(documentsBySource.get(source.row_id) ?? createKnowledgeSourceDocumentFixture({ source })),
      );
    }),
    http.post(
      `${pathPrefix}/semantic-models/:modelId/sources/:sourceRowId/segments/import-initial`,
      async ({ params, request }) => {
        const modelId = Number(params.modelId);
        const sourceRowId = decodeURIComponent(String(params.sourceRowId));
        const source = (sourcesByModel.get(modelId) ?? []).find((item) => item.row_id === sourceRowId);
        if (!source || source.source_type === 'table') {
          return HttpResponse.json({ code: 'ErrNotFound', msg: 'source document not found', data: null }, { status: 404 });
        }
        const current = documentsBySource.get(sourceRowId) ?? createKnowledgeSourceDocumentFixture({ source });
        const body = (await request.json()) as SemanticModelSegmentMutationBase;
        if (segmentBaseConflict(current, body)) {
          return HttpResponse.json(
            { code: 'ErrConflict', msg: 'segment version was changed concurrently', data: null },
            { status: 409 },
          );
        }
        const next = nextSegmentDocument(current, [
          createKnowledgeSegmentFixture({
            content: 'Imported initial segment',
            metadata: { volume_id: source.processed_volume_id },
          }),
        ]);
        documentsBySource.set(sourceRowId, next);
        return HttpResponse.json(mockResponse({ document: next }));
      },
    ),
    http.post(
      `${pathPrefix}/semantic-models/:modelId/sources/:sourceRowId/segments/re-embedding`,
      async ({ params, request }) => {
        const sourceRowId = decodeURIComponent(String(params.sourceRowId));
        const current = documentsBySource.get(sourceRowId);
        if (!current) {
          return HttpResponse.json({ code: 'ErrNotFound', msg: 'source document not found', data: null }, { status: 404 });
        }
        const body = (await request.json()) as SemanticModelSegmentMutationBase;
        if (segmentBaseConflict(current, body)) {
          return HttpResponse.json(
            { code: 'ErrConflict', msg: 'segment version was changed concurrently', data: null },
            { status: 409 },
          );
        }
        const next = nextSegmentDocument(current, current.segments);
        documentsBySource.set(sourceRowId, next);
        return HttpResponse.json(mockResponse({ document: next }));
      },
    ),
    http.patch(
      `${pathPrefix}/semantic-models/:modelId/sources/:sourceRowId/segments/:segmentId/enabled`,
      async ({ params, request }) => {
        const sourceRowId = decodeURIComponent(String(params.sourceRowId));
        const segmentId = decodeURIComponent(String(params.segmentId));
        const current = documentsBySource.get(sourceRowId);
        if (!current) {
          return HttpResponse.json({ code: 'ErrNotFound', msg: 'source document not found', data: null }, { status: 404 });
        }
        const body = (await request.json()) as SemanticModelSegmentMutationBase & { enabled: boolean };
        if (segmentBaseConflict(current, body)) {
          return HttpResponse.json(
            { code: 'ErrConflict', msg: 'segment version was changed concurrently', data: null },
            { status: 409 },
          );
        }
        const next = nextSegmentDocument(
          current,
          current.segments.map((segment) => (segment.segment_id === segmentId ? { ...segment, enabled: body.enabled } : segment)),
        );
        documentsBySource.set(sourceRowId, next);
        return HttpResponse.json(mockResponse({ document: next }));
      },
    ),
    http.patch(`${pathPrefix}/semantic-models/:modelId/sources/:sourceRowId/segments/:segmentId`, async ({ params, request }) => {
      const sourceRowId = decodeURIComponent(String(params.sourceRowId));
      const segmentId = decodeURIComponent(String(params.segmentId));
      const current = documentsBySource.get(sourceRowId);
      if (!current) {
        return HttpResponse.json({ code: 'ErrNotFound', msg: 'source document not found', data: null }, { status: 404 });
      }
      const body = (await request.json()) as SemanticModelSegmentMutationBase &
        Partial<Pick<SemanticModelDocumentSegment, 'content' | 'ocr_text' | 'image_description'>>;
      if (segmentBaseConflict(current, body)) {
        return HttpResponse.json(
          { code: 'ErrConflict', msg: 'segment version was changed concurrently', data: null },
          { status: 409 },
        );
      }
      const next = nextSegmentDocument(
        current,
        current.segments.map((segment) =>
          segment.segment_id === segmentId
            ? {
                ...segment,
                content: body.content !== undefined ? body.content : segment.content,
                ocr_text: body.ocr_text !== undefined ? body.ocr_text : segment.ocr_text,
                image_description: body.image_description !== undefined ? body.image_description : segment.image_description,
              }
            : segment,
        ),
      );
      documentsBySource.set(sourceRowId, next);
      return HttpResponse.json(mockResponse({ document: next }));
    }),
    http.post(`${pathPrefix}/semantic-models/:modelId/sources/:sourceRowId/segments`, async ({ params, request }) => {
      const sourceRowId = decodeURIComponent(String(params.sourceRowId));
      const current = documentsBySource.get(sourceRowId);
      if (!current) {
        return HttpResponse.json({ code: 'ErrNotFound', msg: 'source document not found', data: null }, { status: 404 });
      }
      const body = (await request.json()) as SemanticModelSegmentMutationBase & Partial<SemanticModelDocumentSegment>;
      if (segmentBaseConflict(current, body)) {
        return HttpResponse.json(
          { code: 'ErrConflict', msg: 'segment version was changed concurrently', data: null },
          { status: 409 },
        );
      }
      const nextSegment = createKnowledgeSegmentFixture({
        chunk_index: current.segments.length,
        content: body.content ?? '',
        ocr_text: body.ocr_text ?? '',
        image_description: body.image_description ?? '',
      });
      const next = nextSegmentDocument(current, [...current.segments, nextSegment]);
      documentsBySource.set(sourceRowId, next);
      return HttpResponse.json(mockResponse({ document: next }));
    }),
    http.delete(
      `${pathPrefix}/semantic-models/:modelId/sources/:sourceRowId/segments/:segmentId`,
      async ({ params, request }) => {
        const sourceRowId = decodeURIComponent(String(params.sourceRowId));
        const segmentId = decodeURIComponent(String(params.segmentId));
        const current = documentsBySource.get(sourceRowId);
        if (!current) {
          return HttpResponse.json({ code: 'ErrNotFound', msg: 'source document not found', data: null }, { status: 404 });
        }
        const body = (await request.json()) as SemanticModelSegmentMutationBase;
        if (segmentBaseConflict(current, body)) {
          return HttpResponse.json(
            { code: 'ErrConflict', msg: 'segment version was changed concurrently', data: null },
            { status: 409 },
          );
        }
        const nextSegments = current.segments.filter((segment) => segment.segment_id !== segmentId);
        if (nextSegments.length === current.segments.length) {
          return HttpResponse.json({ code: 'ErrNotFound', msg: 'segment not found', data: null }, { status: 404 });
        }
        const next = nextSegmentDocument(current, nextSegments, 'delete_chunk');
        documentsBySource.set(sourceRowId, next);
        return HttpResponse.json(mockResponse({ document: next }));
      },
    ),
    http.patch(
      `${pathPrefix}/semantic-models/:modelId/sources/:sourceRowId/segment-versions/:versionId/current`,
      async ({ params, request }) => {
        const sourceRowId = decodeURIComponent(String(params.sourceRowId));
        const versionId = decodeURIComponent(String(params.versionId));
        const current = documentsBySource.get(sourceRowId);
        if (!current) {
          return HttpResponse.json({ code: 'ErrNotFound', msg: 'source document not found', data: null }, { status: 404 });
        }
        const body = (await request.json()) as SemanticModelSegmentMutationBase;
        if (segmentBaseConflict(current, body)) {
          return HttpResponse.json(
            { code: 'ErrConflict', msg: 'segment version was changed concurrently', data: null },
            { status: 409 },
          );
        }
        const version = current.segment_versions.find((item) => item.version_id === versionId);
        if (!version || version.status !== 'committed') {
          return HttpResponse.json(
            { code: 'ErrBadRequest', msg: 'only committed segment version can be set current', data: null },
            { status: 400 },
          );
        }
        const next: SemanticModelSourceDocument = {
          ...current,
          current_segment_version_id: version.version_id,
          current_index_version: version.index_version ?? null,
          segment_versions: current.segment_versions.map((item) => ({
            ...item,
            current: item.version_id === version.version_id,
          })),
          source: {
            ...current.source,
            segment_version_id: version.version_id,
            index_version: version.index_version ?? null,
          },
          file_info: {
            ...current.file_info,
            segment_version_id: version.version_id,
            index_version: version.index_version ?? null,
          },
        };
        documentsBySource.set(sourceRowId, next);
        return HttpResponse.json(mockResponse({ document: next }));
      },
    ),
    http.patch(`${pathPrefix}/semantic-models/:modelId/sources/:sourceRowId/governance`, async ({ params, request }) => {
      const modelId = Number(params.modelId);
      const sourceRowId = decodeURIComponent(String(params.sourceRowId));
      const sources = sourcesByModel.get(modelId) ?? [];
      const sourceIndex = sources.findIndex((item) => item.row_id === sourceRowId);
      if (sourceIndex < 0) {
        return HttpResponse.json({ code: 'ErrNotFound', msg: 'source not found', data: null }, { status: 404 });
      }

      const body = (await request.json()) as UpdateSemanticModelSourceGovernanceRequest;
      const current = sources[sourceIndex];
      if (current.source_type === 'table' && (body.tags !== undefined || body.force_enabled_after_expiry !== undefined)) {
        return HttpResponse.json(
          { code: 'ErrBadRequest', msg: 'table source governance only supports enabled and expires_at', data: null },
          { status: 400 },
        );
      }
      const currentEnabled = current.enabled ?? true;
      const currentExpired =
        current.expires_at !== null && current.expires_at !== undefined && current.expires_at <= Math.floor(Date.now() / 1000);
      const currentEffectiveEnabled = currentEnabled && (!currentExpired || Boolean(current.force_enabled_after_expiry));
      const updated: SemanticModelSource = {
        ...current,
        tags: current.source_type === 'table' ? current.tags : body.tags !== undefined ? body.tags : (current.tags ?? []),
        expires_at: body.expires_at !== undefined ? body.expires_at : current.expires_at,
        enabled: body.enabled !== undefined ? body.enabled : current.enabled,
        force_enabled_after_expiry:
          current.source_type !== 'table' && body.force_enabled_after_expiry !== undefined
            ? body.force_enabled_after_expiry
            : (current.force_enabled_after_expiry ?? false),
      };
      if (body.enabled !== undefined) {
        if (!body.enabled) {
          updated.force_enabled_after_expiry = false;
        } else if (!currentEffectiveEnabled) {
          const nextExpired =
            updated.expires_at !== null &&
            updated.expires_at !== undefined &&
            updated.expires_at <= Math.floor(Date.now() / 1000);
          if (nextExpired) {
            updated.force_enabled_after_expiry = true;
          }
        }
      }
      const enabled = updated.enabled ?? true;
      const expired =
        updated.expires_at !== null && updated.expires_at !== undefined && updated.expires_at <= Math.floor(Date.now() / 1000);
      updated.expired = expired;
      updated.effective_enabled = enabled && (!expired || Boolean(updated.force_enabled_after_expiry));
      sources[sourceIndex] = updated;
      sourcesByModel.set(modelId, sources);
      return HttpResponse.json(mockResponse({ source: updated }));
    }),
    http.delete(`${pathPrefix}/semantic-models/:modelId/sources/:sourceRowId`, ({ params }) => {
      const modelId = Number(params.modelId);
      const sourceRowId = decodeURIComponent(String(params.sourceRowId));
      const sources = sourcesByModel.get(modelId) ?? [];
      const nextSources = sources.filter((item) => item.row_id !== sourceRowId);
      if (nextSources.length === sources.length) {
        return HttpResponse.json({ code: 'ErrNotFound', msg: 'source not found', data: null }, { status: 404 });
      }
      sourcesByModel.set(modelId, nextSources);
      jobsByModel.set(
        modelId,
        (jobsByModel.get(modelId) ?? []).filter((job) => job.source_id !== sourceRowId),
      );
      return HttpResponse.json(mockResponse({ deleted: true }));
    }),
    http.post(`${pathPrefix}/semantic-models/:modelId/sources/backfill-legacy`, ({ params }) => {
      const modelId = Number(params.modelId);
      const legacySources = legacySourcesByModel.get(modelId) ?? [];
      const batchSize = data.legacyBackfillBatchSize ?? legacySources.length;
      const backfilled = legacySources.slice(0, batchSize).map((source) =>
        createKnowledgeSourceFixture({
          ...source,
          row_id: source.row_id.replace(/^candidate:/, ''),
          source_id: source.source_id || source.row_id.replace(/^candidate:/, ''),
          enabled: source.enabled ?? true,
          effective_enabled: source.effective_enabled ?? true,
          governance_status: 'managed',
          legacy_origin: null,
        }),
      );
      if (backfilled.length > 0) {
        sourcesByModel.set(modelId, [...(sourcesByModel.get(modelId) ?? []), ...backfilled]);
        legacySourcesByModel.set(modelId, legacySources.slice(batchSize));
      }
      return HttpResponse.json(mockResponse({ updated: true }));
    }),
    http.get(`${pathPrefix}/semantic-models/:modelId/source-jobs`, ({ params }) => {
      const modelId = Number(params.modelId);
      const jobs = jobsByModel.get(modelId) ?? [];
      return HttpResponse.json(mockResponse({ items: jobs, total: jobs.length }));
    }),
    http.post(`${pathPrefix}/semantic-models/:modelId/source-jobs/reconcile`, () =>
      HttpResponse.json(mockResponse({ updated: true })),
    ),
  ];
}

export const knowledgeHandlers = createKnowledgeHandlers();
