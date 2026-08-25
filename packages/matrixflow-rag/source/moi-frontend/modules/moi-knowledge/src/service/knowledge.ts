import {
  appendSemanticModelSourcesApi,
  createEmptySemanticModelApi,
  createSemanticModelApi,
  createSemanticModelSegmentApi,
  createSemanticModelWithSourcesApi,
  deleteSemanticModelApi,
  deleteSemanticModelSegmentApi,
  deleteSemanticModelSourceApi,
  getSemanticModelApi,
  getSemanticModelSourceDocumentApi,
  importInitialSemanticModelSegmentsApi,
  listSemanticModelsApi,
  listSemanticModelSourceJobsApi,
  listSemanticModelSourcesApi,
  reconcileSemanticModelSourceJobsApi,
  reembedSemanticModelSegmentsApi,
  setCurrentSemanticModelSegmentVersionApi,
  updateSemanticModelApi,
  updateSemanticModelSegmentApi,
  updateSemanticModelSegmentEnabledApi,
  updateSemanticModelSourceGovernanceApi,
  type AppendSemanticModelSourcesRequest,
  type AppendSemanticModelSourcesResponse,
  type CreateEmptySemanticModelRequest,
  type CreateEmptySemanticModelResponse,
  type CreateSemanticModelSegmentRequest,
  type CreateSemanticModelWithSourcesRequest,
  type CreateSemanticModelWithSourcesResponse,
  type DeleteSemanticModelSegmentRequest,
  type ImportInitialSemanticModelSegmentsRequest,
  type ReembedSemanticModelSegmentsRequest,
  type SemanticModel,
  type SemanticModelCreateResponse,
  type SemanticModelFiles,
  type SemanticModelListResponse,
  type SemanticModelMutationResponse,
  type SemanticModelSegmentMutationResult,
  type SemanticModelSourceDocument,
  type SemanticModelSourceJobListResponse,
  type SemanticModelSourceListRequest,
  type SemanticModelSourceListResponse,
  type SemanticModelTable,
  type SemanticModelUpsertRequest,
  type SetCurrentSemanticModelSegmentVersionRequest,
  type UpdateSemanticModelSegmentEnabledRequest,
  type UpdateSemanticModelSegmentRequest,
  type UpdateSemanticModelSourceGovernanceRequest,
  type UpdateSemanticModelSourceGovernanceResponse,
} from '@moi/shared-moi-api/knowledge';
import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { unwrapApiResponse } from './http';

export type { SemanticModelFiles, SemanticModelTable };

// Type aliases for backward compatibility with page-layer imports
export type KnowledgeListItem = SemanticModel;
export type GetKnowledgeDetailResponse = SemanticModel;
export type CreateKnowledgeResponse = SemanticModelCreateResponse;
export type CommonSuccessResponse = SemanticModelMutationResponse;
export type QueryKnowledgeListResponse = SemanticModelListResponse;
export type QueryKnowledgeSourcesResponse = SemanticModelSourceListResponse;
export type GetKnowledgeSourceDocumentResponse = SemanticModelSourceDocument;
export type CreateKnowledgeRequest = SemanticModelUpsertRequest;
export type CreateEmptyKnowledgeRequest = CreateEmptySemanticModelRequest;
export type CreateEmptyKnowledgeResponse = CreateEmptySemanticModelResponse;
export type CreateKnowledgeWithSourcesRequest = CreateSemanticModelWithSourcesRequest;
export type CreateKnowledgeWithSourcesResponse = CreateSemanticModelWithSourcesResponse;
export type AppendKnowledgeSourcesResponse = AppendSemanticModelSourcesResponse;
export type QueryKnowledgeSourceJobsResponse = SemanticModelSourceJobListResponse;
export type UpdateKnowledgeSourceGovernanceResponse = UpdateSemanticModelSourceGovernanceResponse;
export interface UpdateKnowledgeRequest {
  id: number;
  name: string;
  description?: string;
}

export interface QueryKnowledgeListRequest {
  page_size?: number;
  page_token?: string;
  search?: string;
}

export interface DeleteKnowledgeRequest {
  id: number;
}

export interface GetKnowledgeDetailRequest {
  id: number;
}

export type QueryKnowledgeSourcesRequest = GetKnowledgeDetailRequest & SemanticModelSourceListRequest;

export type AppendKnowledgeSourcesRequest = AppendSemanticModelSourcesRequest & { id: number };

export interface GetKnowledgeSourceDocumentRequest {
  id: number;
  sourceRowId: string;
  segmentVersionId?: string;
}

export interface DeleteKnowledgeSourceRequest {
  id: number;
  sourceRowId: string;
}

export type UpdateKnowledgeSourceGovernanceRequest = UpdateSemanticModelSourceGovernanceRequest & {
  id: number;
  sourceRowId: string;
};

export type ImportInitialKnowledgeSegmentsRequest = ImportInitialSemanticModelSegmentsRequest & {
  id: number;
  sourceRowId: string;
};

export type UpdateKnowledgeSegmentRequest = UpdateSemanticModelSegmentRequest & {
  id: number;
  sourceRowId: string;
  segmentId: string;
};

export type CreateKnowledgeSegmentRequest = CreateSemanticModelSegmentRequest & {
  id: number;
  sourceRowId: string;
};

export type UpdateKnowledgeSegmentEnabledRequest = UpdateSemanticModelSegmentEnabledRequest & {
  id: number;
  sourceRowId: string;
  segmentId: string;
};

export type DeleteKnowledgeSegmentRequest = DeleteSemanticModelSegmentRequest & {
  id: number;
  sourceRowId: string;
  segmentId: string;
};

export type ReembedKnowledgeSegmentsRequest = ReembedSemanticModelSegmentsRequest & {
  id: number;
  sourceRowId: string;
};

export type SetCurrentKnowledgeSegmentVersionRequest = SetCurrentSemanticModelSegmentVersionRequest & {
  id: number;
  sourceRowId: string;
  versionId: string;
};

export type MutateKnowledgeSegmentsResponse = SemanticModelSegmentMutationResult;

/**
 * List semantic models with cursor pagination.
 */
export async function queryKnowledgeList(
  http: AppHttpClient,
  params: QueryKnowledgeListRequest = { page_size: 10 },
): Promise<QueryKnowledgeListResponse> {
  return unwrapApiResponse(
    await listSemanticModelsApi(
      { page_size: params.page_size ?? 10, page_token: params.page_token, search: params.search },
      http,
    ),
    'queryKnowledgeList',
  );
}

export async function createKnowledge(http: AppHttpClient, body: CreateKnowledgeRequest): Promise<CreateKnowledgeResponse> {
  return unwrapApiResponse(await createSemanticModelApi(body, http), 'createKnowledge');
}

export async function createEmptyKnowledge(
  http: AppHttpClient,
  body: CreateEmptyKnowledgeRequest,
): Promise<CreateEmptyKnowledgeResponse> {
  return unwrapApiResponse(await createEmptySemanticModelApi(body, http), 'createEmptyKnowledge');
}

export async function createKnowledgeWithSources(
  http: AppHttpClient,
  body: CreateKnowledgeWithSourcesRequest,
): Promise<CreateKnowledgeWithSourcesResponse> {
  return unwrapApiResponse(await createSemanticModelWithSourcesApi(body, http), 'createKnowledgeWithSources');
}

export async function appendKnowledgeSources(
  http: AppHttpClient,
  body: AppendKnowledgeSourcesRequest,
): Promise<AppendKnowledgeSourcesResponse> {
  const { id, ...payload } = body;
  return unwrapApiResponse(await appendSemanticModelSourcesApi(id, payload, http), 'appendKnowledgeSources');
}

export async function deleteKnowledgeById(http: AppHttpClient, body: DeleteKnowledgeRequest): Promise<CommonSuccessResponse> {
  return unwrapApiResponse(await deleteSemanticModelApi(body.id, http), 'deleteKnowledgeById');
}

export async function deleteKnowledgeSource(
  http: AppHttpClient,
  body: DeleteKnowledgeSourceRequest,
): Promise<CommonSuccessResponse> {
  return unwrapApiResponse(await deleteSemanticModelSourceApi(body.id, body.sourceRowId, http), 'deleteKnowledgeSource');
}

export async function getKnowledgeDetail(
  http: AppHttpClient,
  params: GetKnowledgeDetailRequest,
): Promise<GetKnowledgeDetailResponse> {
  return unwrapApiResponse(await getSemanticModelApi(params.id, http), 'getKnowledgeDetail');
}

export async function listKnowledgeSources(
  http: AppHttpClient,
  params: QueryKnowledgeSourcesRequest,
): Promise<QueryKnowledgeSourcesResponse> {
  return unwrapApiResponse(
    await listSemanticModelSourcesApi(params.id, http, { page: params.page, page_size: params.page_size }),
    'listKnowledgeSources',
  );
}

export async function listKnowledgeSourceJobs(
  http: AppHttpClient,
  params: GetKnowledgeDetailRequest,
): Promise<QueryKnowledgeSourceJobsResponse> {
  return unwrapApiResponse(await listSemanticModelSourceJobsApi(params.id, http), 'listKnowledgeSourceJobs');
}

export async function reconcileKnowledgeSourceJobs(
  http: AppHttpClient,
  params: GetKnowledgeDetailRequest,
): Promise<CommonSuccessResponse> {
  return unwrapApiResponse(await reconcileSemanticModelSourceJobsApi(params.id, http), 'reconcileKnowledgeSourceJobs');
}

export async function getKnowledgeSourceDocument(
  http: AppHttpClient,
  params: GetKnowledgeSourceDocumentRequest,
): Promise<GetKnowledgeSourceDocumentResponse> {
  return unwrapApiResponse(
    await getSemanticModelSourceDocumentApi(
      params.id,
      params.sourceRowId,
      http,
      params.segmentVersionId ? { segment_version_id: params.segmentVersionId } : {},
    ),
    'getKnowledgeSourceDocument',
  );
}

export async function importInitialKnowledgeSegments(
  http: AppHttpClient,
  body: ImportInitialKnowledgeSegmentsRequest,
): Promise<MutateKnowledgeSegmentsResponse> {
  const { id, sourceRowId, ...payload } = body;
  return unwrapApiResponse(
    await importInitialSemanticModelSegmentsApi(id, sourceRowId, payload, http),
    'importInitialKnowledgeSegments',
  );
}

export async function updateKnowledgeSegment(
  http: AppHttpClient,
  body: UpdateKnowledgeSegmentRequest,
): Promise<MutateKnowledgeSegmentsResponse> {
  const { id, sourceRowId, segmentId, ...payload } = body;
  return unwrapApiResponse(
    await updateSemanticModelSegmentApi(id, sourceRowId, segmentId, payload, http),
    'updateKnowledgeSegment',
  );
}

export async function createKnowledgeSegment(
  http: AppHttpClient,
  body: CreateKnowledgeSegmentRequest,
): Promise<MutateKnowledgeSegmentsResponse> {
  const { id, sourceRowId, ...payload } = body;
  return unwrapApiResponse(await createSemanticModelSegmentApi(id, sourceRowId, payload, http), 'createKnowledgeSegment');
}

export async function updateKnowledgeSegmentEnabled(
  http: AppHttpClient,
  body: UpdateKnowledgeSegmentEnabledRequest,
): Promise<MutateKnowledgeSegmentsResponse> {
  const { id, sourceRowId, segmentId, ...payload } = body;
  return unwrapApiResponse(
    await updateSemanticModelSegmentEnabledApi(id, sourceRowId, segmentId, payload, http),
    'updateKnowledgeSegmentEnabled',
  );
}

export async function deleteKnowledgeSegment(
  http: AppHttpClient,
  body: DeleteKnowledgeSegmentRequest,
): Promise<MutateKnowledgeSegmentsResponse> {
  const { id, sourceRowId, segmentId, ...payload } = body;
  return unwrapApiResponse(
    await deleteSemanticModelSegmentApi(id, sourceRowId, segmentId, payload, http),
    'deleteKnowledgeSegment',
  );
}

export async function reembedKnowledgeSegments(
  http: AppHttpClient,
  body: ReembedKnowledgeSegmentsRequest,
): Promise<MutateKnowledgeSegmentsResponse> {
  const { id, sourceRowId, ...payload } = body;
  return unwrapApiResponse(await reembedSemanticModelSegmentsApi(id, sourceRowId, payload, http), 'reembedKnowledgeSegments');
}

export async function setCurrentKnowledgeSegmentVersion(
  http: AppHttpClient,
  body: SetCurrentKnowledgeSegmentVersionRequest,
): Promise<MutateKnowledgeSegmentsResponse> {
  const { id, sourceRowId, versionId, ...payload } = body;
  return unwrapApiResponse(
    await setCurrentSemanticModelSegmentVersionApi(id, sourceRowId, versionId, payload, http),
    'setCurrentKnowledgeSegmentVersion',
  );
}

export async function updateKnowledgeSourceGovernance(
  http: AppHttpClient,
  body: UpdateKnowledgeSourceGovernanceRequest,
): Promise<UpdateKnowledgeSourceGovernanceResponse> {
  const { id, sourceRowId, ...payload } = body;
  return unwrapApiResponse(
    await updateSemanticModelSourceGovernanceApi(id, sourceRowId, payload, http),
    'updateKnowledgeSourceGovernance',
  );
}

export async function updateKnowledge(http: AppHttpClient, body: UpdateKnowledgeRequest): Promise<CommonSuccessResponse> {
  const { id, ...payload } = body;
  return unwrapApiResponse(await updateSemanticModelApi(id, payload, http), 'updateKnowledge');
}
