import {
  createSemanticEntryApi,
  deleteSemanticEntryApi,
  exportSemanticModelApi,
  getSemanticModelApi,
  importSemanticModelApi,
  listSemanticEntriesApi,
  updateSemanticEntryApi,
  validateSemanticModelApi,
  type CreateSemanticEntryRequest,
  type ExportSemanticModelResponse,
  type ImportSemanticModelRequest,
  type ImportSemanticModelResponse,
  type SemanticEntry,
  type SemanticEntryListParams,
  type SemanticEntryListResponse,
  type SemanticModel,
  type SemanticMutationResponse,
  type ValidateSemanticModelResponse,
} from '@moi/shared-moi-api/knowledge';
import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { unwrapApiResponse } from './http';

export type {
  CreateSemanticEntryRequest,
  ExportSemanticModelResponse,
  ImportSemanticModelRequest,
  ImportSemanticModelResponse,
  SemanticEntry,
  SemanticEntryListParams,
  SemanticEntryListResponse,
  SemanticModel,
  SemanticMutationResponse,
  ValidateSemanticModelResponse,
};

export async function getSemanticModel(http: AppHttpClient, modelId: number): Promise<SemanticModel> {
  if (!Number.isFinite(modelId) || modelId <= 0) {
    console.warn('[getSemanticModel] invalid modelId', { modelId });
    throw new Error('Invalid model id');
  }
  return unwrapApiResponse(await getSemanticModelApi(modelId, http), 'getSemanticModel');
}

export async function listSemanticEntries(
  http: AppHttpClient,
  modelId: number,
  params: SemanticEntryListParams = {},
): Promise<SemanticEntryListResponse> {
  if (!Number.isFinite(modelId) || modelId <= 0) {
    console.warn('[listSemanticEntries] invalid modelId', { modelId });
    throw new Error('Invalid model id');
  }
  return unwrapApiResponse(await listSemanticEntriesApi(modelId, params, http), 'listSemanticEntries');
}

export async function createSemanticEntry(
  http: AppHttpClient,
  modelId: number,
  body: CreateSemanticEntryRequest,
): Promise<SemanticEntry> {
  if (!Number.isFinite(modelId) || modelId <= 0) {
    console.warn('[createSemanticEntry] invalid modelId', { modelId });
    throw new Error('Invalid model id');
  }
  return unwrapApiResponse(await createSemanticEntryApi(modelId, body, http), 'createSemanticEntry');
}

export async function updateSemanticEntry(
  http: AppHttpClient,
  modelId: number,
  entryId: number,
  body: CreateSemanticEntryRequest,
): Promise<SemanticMutationResponse> {
  if (!Number.isFinite(modelId) || modelId <= 0) {
    console.warn('[updateSemanticEntry] invalid modelId', { modelId });
    throw new Error('Invalid model id');
  }
  if (!Number.isFinite(entryId) || entryId <= 0) {
    console.warn('[updateSemanticEntry] invalid entryId', { entryId });
    throw new Error('Invalid entry id');
  }
  return unwrapApiResponse(await updateSemanticEntryApi(modelId, entryId, body, http), 'updateSemanticEntry');
}

export async function deleteSemanticEntry(
  http: AppHttpClient,
  modelId: number,
  entryId: number,
): Promise<SemanticMutationResponse> {
  if (!Number.isFinite(modelId) || modelId <= 0) {
    console.warn('[deleteSemanticEntry] invalid modelId', { modelId });
    throw new Error('Invalid model id');
  }
  if (!Number.isFinite(entryId) || entryId <= 0) {
    console.warn('[deleteSemanticEntry] invalid entryId', { entryId });
    throw new Error('Invalid entry id');
  }
  return unwrapApiResponse(await deleteSemanticEntryApi(modelId, entryId, http), 'deleteSemanticEntry');
}

export async function importSemanticModel(
  http: AppHttpClient,
  modelId: number,
  body: ImportSemanticModelRequest,
): Promise<ImportSemanticModelResponse> {
  if (!Number.isFinite(modelId) || modelId <= 0) {
    console.warn('[importSemanticModel] invalid modelId', { modelId });
    throw new Error('Invalid model id');
  }
  return unwrapApiResponse(await importSemanticModelApi(modelId, body, http), 'importSemanticModel');
}

export async function exportSemanticModel(http: AppHttpClient, modelId: number): Promise<ExportSemanticModelResponse> {
  if (!Number.isFinite(modelId) || modelId <= 0) {
    console.warn('[exportSemanticModel] invalid modelId', { modelId });
    throw new Error('Invalid model id');
  }
  return unwrapApiResponse(await exportSemanticModelApi(modelId, http), 'exportSemanticModel');
}

export async function validateSemanticModel(http: AppHttpClient, modelId: number): Promise<ValidateSemanticModelResponse> {
  if (!Number.isFinite(modelId) || modelId <= 0) {
    console.warn('[validateSemanticModel] invalid modelId', { modelId });
    throw new Error('Invalid model id');
  }
  return unwrapApiResponse(await validateSemanticModelApi(modelId, http), 'validateSemanticModel');
}
