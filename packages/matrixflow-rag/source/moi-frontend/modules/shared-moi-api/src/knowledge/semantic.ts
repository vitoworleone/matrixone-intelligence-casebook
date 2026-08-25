import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import type { ApiResponse } from '../types';
import type {
  AppendSemanticModelSourcesRequest,
  AppendSemanticModelSourcesResponse,
  BackfillLegacySemanticModelSourcesResponse,
  CheckSemanticModelSourceExistenceRequest,
  CheckSemanticModelSourceExistenceResponse,
  CreateEmptySemanticModelRequest,
  CreateEmptySemanticModelResponse,
  CreateSemanticEntryRequest,
  CreateSemanticModelSegmentRequest,
  CreateSemanticModelWithSourcesRequest,
  CreateSemanticModelWithSourcesResponse,
  DeleteSemanticModelSegmentRequest,
  ExportSemanticModelResponse,
  GetSemanticModelSourceDocumentRequest,
  ImportInitialSemanticModelSegmentsRequest,
  ImportSemanticModelRequest,
  ImportSemanticModelResponse,
  PreviewSemanticModelSourceSelectionsRequest,
  PreviewSemanticModelSourceSelectionsResponse,
  ReconcileSemanticModelSourceJobsResponse,
  ReembedSemanticModelSegmentsRequest,
  SemanticEntry,
  SemanticEntryListParams,
  SemanticEntryListResponse,
  SemanticModel,
  SemanticModelArtifactPreviewOptions,
  SemanticModelArtifactPreviewResponse,
  SemanticModelCreateResponse,
  SemanticModelFilePreviewOptions,
  SemanticModelFilePreviewResponse,
  SemanticModelListRequest,
  SemanticModelListResponse,
  SemanticModelMutationResponse,
  SemanticModelSegmentMutationResult,
  SemanticModelSourceDocument,
  SemanticModelSourceJobListResponse,
  SemanticModelSourceListRequest,
  SemanticModelSourceListResponse,
  SemanticModelTagListResponse,
  SemanticModelUpdateRequest,
  SemanticModelUpsertRequest,
  SemanticMutationResponse,
  SetCurrentSemanticModelSegmentVersionRequest,
  UpdateSemanticEntryRequest,
  UpdateSemanticModelSegmentEnabledRequest,
  UpdateSemanticModelSegmentRequest,
  UpdateSemanticModelSourceGovernanceRequest,
  UpdateSemanticModelSourceGovernanceResponse,
  UploadSemanticModelLocalFileResponse,
  ValidateSemanticModelResponse,
} from './semantic.types';

const BASE = '/semantic-models';

interface SemanticModelPostHttpClient {
  post<T = unknown>(url: string, data?: unknown): Promise<{ data: T }>;
}

function sourcePath(modelId: number, sourceRowId: string): string {
  return `${BASE}/${modelId}/sources/${encodeURIComponent(sourceRowId)}`;
}

/** GET /semantic-models — list semantic models (cursor pagination) */
export async function listSemanticModelsApi(
  params: SemanticModelListRequest,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelListResponse>> {
  const query = new URLSearchParams();
  query.set('page_size', String(params.page_size));
  if (params.page_token) query.set('page_token', params.page_token);
  if (params.search) query.set('search', params.search);
  params.tags?.forEach((tag) => query.append('tags', tag));
  const qs = query.toString();
  const res = await http.get<ApiResponse<SemanticModelListResponse>>(`${BASE}${qs ? `?${qs}` : ''}`);
  return res.data;
}

/** GET /semantic-models/tags — aggregate semantic model tags */
export async function listSemanticModelTagsApi(
  params: Pick<SemanticModelListRequest, 'search'>,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelTagListResponse>> {
  const query = new URLSearchParams();
  if (params.search) query.set('search', params.search);
  const qs = query.toString();
  const res = await http.get<ApiResponse<SemanticModelTagListResponse>>(`${BASE}/tags${qs ? `?${qs}` : ''}`);
  return res.data;
}

/** GET /semantic-models/:model_id/artifacts/:file_id/preview — preview a model-owned parsing artifact */
export async function previewSemanticModelArtifactApi(
  modelId: number,
  fileId: string,
  http: AppHttpClient,
  options?: SemanticModelArtifactPreviewOptions,
): Promise<SemanticModelArtifactPreviewResponse> {
  const config: SemanticModelArtifactPreviewOptions = {
    responseType: 'blob',
    responseContentType: 'blob',
    ...options,
  };
  const res = await http.get<Blob>(`${BASE}/${modelId}/artifacts/${encodeURIComponent(fileId)}/preview`, config);
  return {
    headers: res.headers,
    data: res.data,
  };
}

/** GET /semantic-models/:model_id/sources/file/:file_id/preview — preview a model-owned source file */
export async function previewSemanticModelSourceFileApi(
  modelId: number,
  fileId: string,
  http: AppHttpClient,
  options?: SemanticModelFilePreviewOptions,
): Promise<SemanticModelFilePreviewResponse> {
  const config: SemanticModelFilePreviewOptions = {
    responseType: 'blob',
    responseContentType: 'blob',
    ...options,
  };
  const res = await http.get<Blob>(`${BASE}/${modelId}/sources/file/${encodeURIComponent(fileId)}/preview`, config);
  return {
    headers: res.headers,
    data: res.data,
  };
}

/** POST /semantic-models — create a semantic model */
/** POST /semantic-models — create a semantic model */
export async function createSemanticModelApi(
  req: SemanticModelUpsertRequest,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelCreateResponse>> {
  const res = await http.post<ApiResponse<SemanticModelCreateResponse>>(`${BASE}`, req);
  return res.data;
}

/** POST /semantic-models/create-with-sources — create a KB with initial sources */
export async function createSemanticModelWithSourcesApi(
  req: CreateSemanticModelWithSourcesRequest,
  http: AppHttpClient,
): Promise<ApiResponse<CreateSemanticModelWithSourcesResponse>> {
  const res = await http.post<ApiResponse<CreateSemanticModelWithSourcesResponse>>(`${BASE}/create-with-sources`, req);
  return res.data;
}

/** POST /semantic-models/create-empty — create a data-side KB without sources */
export async function createEmptySemanticModelApi(
  req: CreateEmptySemanticModelRequest,
  http: SemanticModelPostHttpClient,
): Promise<ApiResponse<CreateEmptySemanticModelResponse>> {
  const res = await http.post<ApiResponse<CreateEmptySemanticModelResponse>>(`${BASE}/create-empty`, req);
  return res.data;
}

/** POST /semantic-models/:id/sources — append sources to an existing KB */
export async function appendSemanticModelSourcesApi(
  modelId: number,
  req: AppendSemanticModelSourcesRequest,
  http: AppHttpClient,
): Promise<ApiResponse<AppendSemanticModelSourcesResponse>> {
  const res = await http.post<ApiResponse<AppendSemanticModelSourcesResponse>>(`${BASE}/${modelId}/sources`, req);
  return res.data;
}

/**
 * POST /semantic-models/local-files/upload
 * POST /semantic-models/:id/local-files/upload
 * Upload a knowledge-base local file before create-with-sources (K1) or append sources (object K3).
 */
export async function uploadSemanticModelLocalFileApi(
  formData: FormData,
  http: AppHttpClient,
  modelId?: number,
): Promise<ApiResponse<UploadSemanticModelLocalFileResponse>> {
  const path =
    modelId !== undefined && Number.isFinite(modelId) && modelId > 0
      ? `${BASE}/${modelId}/local-files/upload`
      : `${BASE}/local-files/upload`;
  const res = await http.post<ApiResponse<UploadSemanticModelLocalFileResponse>>(path, formData);
  return res.data;
}

/** POST /semantic-models[/ :id]/source-selections/preview — preview deduplicated source selection counts */
export async function previewSemanticModelSourceSelectionsApi(
  req: PreviewSemanticModelSourceSelectionsRequest,
  http: AppHttpClient,
  modelId?: number,
  signal?: AbortSignal,
): Promise<ApiResponse<PreviewSemanticModelSourceSelectionsResponse>> {
  const path = modelId ? `${BASE}/${modelId}/source-selections/preview` : `${BASE}/source-selections/preview`;
  const res = signal
    ? await http.post<ApiResponse<PreviewSemanticModelSourceSelectionsResponse>>(path, req, { signal })
    : await http.post<ApiResponse<PreviewSemanticModelSourceSelectionsResponse>>(path, req);
  return res.data;
}

/** PUT /semantic-model/:id — update a semantic model */
export async function updateSemanticModelApi(
  modelId: number,
  req: SemanticModelUpdateRequest,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelMutationResponse>> {
  const res = await http.put<ApiResponse<SemanticModelMutationResponse>>(`${BASE}/${modelId}`, req);
  return res.data;
}

/** DELETE /semantic-model/:id — delete a semantic model */
export async function deleteSemanticModelApi(
  modelId: number,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelMutationResponse>> {
  const res = await http.delete<ApiResponse<SemanticModelMutationResponse>>(`${BASE}/${modelId}`);
  return res.data;
}

/** DELETE /semantic-model/:id/sources/:sourceRowId — delete one KB source and its Catalog source */
export async function deleteSemanticModelSourceApi(
  modelId: number,
  sourceRowId: string,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelMutationResponse>> {
  const res = await http.delete<ApiResponse<SemanticModelMutationResponse>>(sourcePath(modelId, sourceRowId));
  return res.data;
}

/** GET /semantic-model/:id */
export async function getSemanticModelApi(modelId: number, http: AppHttpClient): Promise<ApiResponse<SemanticModel>> {
  const res = await http.get<ApiResponse<SemanticModel>>(`${BASE}/${modelId}`);
  return res.data;
}

/** GET /semantic-model/:id/sources */
export async function listSemanticModelSourcesApi(
  modelId: number,
  http: AppHttpClient,
  params: SemanticModelSourceListRequest = {},
): Promise<ApiResponse<SemanticModelSourceListResponse>> {
  const query = new URLSearchParams();
  if (params.page) query.set('page', String(params.page));
  if (params.page_size) query.set('page_size', String(params.page_size));
  const qs = query.toString();
  const res = await http.get<ApiResponse<SemanticModelSourceListResponse>>(`${BASE}/${modelId}/sources${qs ? `?${qs}` : ''}`);
  return res.data;
}

/** POST /semantic-model/:id/sources/existence */
export async function checkSemanticModelSourceExistenceApi(
  modelId: number,
  req: CheckSemanticModelSourceExistenceRequest,
  http: SemanticModelPostHttpClient,
): Promise<ApiResponse<CheckSemanticModelSourceExistenceResponse>> {
  const res = await http.post<ApiResponse<CheckSemanticModelSourceExistenceResponse>>(
    `${BASE}/${modelId}/sources/existence`,
    req,
  );
  return res.data;
}

/** POST /semantic-model/:id/sources/backfill-legacy */
export async function backfillLegacySemanticModelSourcesApi(
  modelId: number,
  http: AppHttpClient,
): Promise<ApiResponse<BackfillLegacySemanticModelSourcesResponse>> {
  const res = await http.post<ApiResponse<BackfillLegacySemanticModelSourcesResponse>>(
    `${BASE}/${modelId}/sources/backfill-legacy`,
  );
  return res.data;
}

/** GET /semantic-model/:id/sources/:sourceRowId/document */
export async function getSemanticModelSourceDocumentApi(
  modelId: number,
  sourceRowId: string,
  http: AppHttpClient,
  params: GetSemanticModelSourceDocumentRequest = {},
): Promise<ApiResponse<SemanticModelSourceDocument>> {
  const query = new URLSearchParams();
  if (params.segment_version_id) query.set('segment_version_id', params.segment_version_id);
  const qs = query.toString();
  const res = await http.get<ApiResponse<SemanticModelSourceDocument>>(
    `${sourcePath(modelId, sourceRowId)}/document${qs ? `?${qs}` : ''}`,
  );
  return res.data;
}

/** POST /semantic-model/:id/sources/:sourceRowId/segments/import-initial */
export async function importInitialSemanticModelSegmentsApi(
  modelId: number,
  sourceRowId: string,
  req: ImportInitialSemanticModelSegmentsRequest,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelSegmentMutationResult>> {
  const res = await http.post<ApiResponse<SemanticModelSegmentMutationResult>>(
    `${sourcePath(modelId, sourceRowId)}/segments/import-initial`,
    req,
  );
  return res.data;
}

/** PATCH /semantic-model/:id/sources/:sourceRowId/segments/:segmentId */
export async function updateSemanticModelSegmentApi(
  modelId: number,
  sourceRowId: string,
  segmentId: string,
  req: UpdateSemanticModelSegmentRequest,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelSegmentMutationResult>> {
  if (typeof http.patch !== 'function') {
    throw new Error('AppHttpClient.patch is required for semantic model segment updates');
  }
  const res = await http.patch<ApiResponse<SemanticModelSegmentMutationResult>>(
    `${sourcePath(modelId, sourceRowId)}/segments/${encodeURIComponent(segmentId)}`,
    req,
  );
  return res.data;
}

/** POST /semantic-model/:id/sources/:sourceRowId/segments */
export async function createSemanticModelSegmentApi(
  modelId: number,
  sourceRowId: string,
  req: CreateSemanticModelSegmentRequest,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelSegmentMutationResult>> {
  const res = await http.post<ApiResponse<SemanticModelSegmentMutationResult>>(
    `${sourcePath(modelId, sourceRowId)}/segments`,
    req,
  );
  return res.data;
}

/** PATCH /semantic-model/:id/sources/:sourceRowId/segments/:segmentId/enabled */
export async function updateSemanticModelSegmentEnabledApi(
  modelId: number,
  sourceRowId: string,
  segmentId: string,
  req: UpdateSemanticModelSegmentEnabledRequest,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelSegmentMutationResult>> {
  if (typeof http.patch !== 'function') {
    throw new Error('AppHttpClient.patch is required for semantic model segment enabled updates');
  }
  const res = await http.patch<ApiResponse<SemanticModelSegmentMutationResult>>(
    `${sourcePath(modelId, sourceRowId)}/segments/${encodeURIComponent(segmentId)}/enabled`,
    req,
  );
  return res.data;
}

/** DELETE /semantic-models/:id/sources/:sourceRowId/segments/:segmentId */
export async function deleteSemanticModelSegmentApi(
  modelId: number,
  sourceRowId: string,
  segmentId: string,
  req: DeleteSemanticModelSegmentRequest,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelSegmentMutationResult>> {
  const res = await http.delete<ApiResponse<SemanticModelSegmentMutationResult>>(
    `${sourcePath(modelId, sourceRowId)}/segments/${encodeURIComponent(segmentId)}`,
    { data: req },
  );
  return res.data;
}

/** POST /semantic-model/:id/sources/:sourceRowId/segments/re-embedding */
export async function reembedSemanticModelSegmentsApi(
  modelId: number,
  sourceRowId: string,
  req: ReembedSemanticModelSegmentsRequest,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelSegmentMutationResult>> {
  const res = await http.post<ApiResponse<SemanticModelSegmentMutationResult>>(
    `${sourcePath(modelId, sourceRowId)}/segments/re-embedding`,
    req,
  );
  return res.data;
}

/** PATCH /semantic-model/:id/sources/:sourceRowId/segment-versions/:versionId/current */
export async function setCurrentSemanticModelSegmentVersionApi(
  modelId: number,
  sourceRowId: string,
  versionId: string,
  req: SetCurrentSemanticModelSegmentVersionRequest,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelSegmentMutationResult>> {
  if (typeof http.patch !== 'function') {
    throw new Error('AppHttpClient.patch is required for semantic model segment version updates');
  }
  const res = await http.patch<ApiResponse<SemanticModelSegmentMutationResult>>(
    `${sourcePath(modelId, sourceRowId)}/segment-versions/${encodeURIComponent(versionId)}/current`,
    req,
  );
  return res.data;
}

/** PATCH /semantic-model/:id/sources/:sourceRowId/governance */
export async function updateSemanticModelSourceGovernanceApi(
  modelId: number,
  sourceRowId: string,
  req: UpdateSemanticModelSourceGovernanceRequest,
  http: AppHttpClient,
): Promise<ApiResponse<UpdateSemanticModelSourceGovernanceResponse>> {
  if (typeof http.patch !== 'function') {
    throw new Error('AppHttpClient.patch is required for semantic model source governance updates');
  }
  const res = await http.patch<ApiResponse<UpdateSemanticModelSourceGovernanceResponse>>(
    `${sourcePath(modelId, sourceRowId)}/governance`,
    req,
  );
  return res.data;
}

/** GET /semantic-model/:id/source-jobs */
export async function listSemanticModelSourceJobsApi(
  modelId: number,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticModelSourceJobListResponse>> {
  const res = await http.get<ApiResponse<SemanticModelSourceJobListResponse>>(`${BASE}/${modelId}/source-jobs`);
  return res.data;
}

/** POST /semantic-model/:id/source-jobs/reconcile */
export async function reconcileSemanticModelSourceJobsApi(
  modelId: number,
  http: AppHttpClient,
): Promise<ApiResponse<ReconcileSemanticModelSourceJobsResponse>> {
  const res = await http.post<ApiResponse<ReconcileSemanticModelSourceJobsResponse>>(`${BASE}/${modelId}/source-jobs/reconcile`);
  return res.data;
}

/** GET /semantic-model/:id/entries */
export async function listSemanticEntriesApi(
  modelId: number,
  params: SemanticEntryListParams,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticEntryListResponse>> {
  const query = new URLSearchParams();
  if (params.kind) query.set('kind', params.kind);
  if (params.page_size !== undefined && params.page_size !== null) query.set('page_size', String(params.page_size));
  if (params.page_token) query.set('page_token', params.page_token);
  const qs = query.toString();
  const url = `${BASE}/${modelId}/entries${qs ? `?${qs}` : ''}`;
  const res = await http.get<ApiResponse<SemanticEntryListResponse>>(url);
  return res.data;
}

/** POST /semantic-model/:id/entries */
export async function createSemanticEntryApi(
  modelId: number,
  req: CreateSemanticEntryRequest,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticEntry>> {
  const res = await http.post<ApiResponse<SemanticEntry>>(`${BASE}/${modelId}/entries`, req);
  return res.data;
}

/** PUT /semantic-model/:id/entries/:entryId */
export async function updateSemanticEntryApi(
  modelId: number,
  entryId: number,
  req: UpdateSemanticEntryRequest,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticMutationResponse>> {
  const res = await http.put<ApiResponse<SemanticMutationResponse>>(`${BASE}/${modelId}/entries/${entryId}`, req);
  return res.data;
}

/** DELETE /semantic-model/:id/entries/:entryId */
export async function deleteSemanticEntryApi(
  modelId: number,
  entryId: number,
  http: AppHttpClient,
): Promise<ApiResponse<SemanticMutationResponse>> {
  const res = await http.delete<ApiResponse<SemanticMutationResponse>>(`${BASE}/${modelId}/entries/${entryId}`);
  return res.data;
}

/** POST /semantic-model/:id/import */
export async function importSemanticModelApi(
  modelId: number,
  req: ImportSemanticModelRequest,
  http: AppHttpClient,
): Promise<ApiResponse<ImportSemanticModelResponse>> {
  const res = await http.post<ApiResponse<ImportSemanticModelResponse>>(`${BASE}/${modelId}/import`, req);
  return res.data;
}

/** GET /semantic-model/:id/export */
export async function exportSemanticModelApi(
  modelId: number,
  http: AppHttpClient,
): Promise<ApiResponse<ExportSemanticModelResponse>> {
  const res = await http.get<ApiResponse<ExportSemanticModelResponse>>(`${BASE}/${modelId}/export`);
  return res.data;
}

/** POST /semantic-model/:id/validate */
export async function validateSemanticModelApi(
  modelId: number,
  http: AppHttpClient,
): Promise<ApiResponse<ValidateSemanticModelResponse>> {
  const res = await http.post<ApiResponse<ValidateSemanticModelResponse>>(`${BASE}/${modelId}/validate`, {});
  return res.data;
}
