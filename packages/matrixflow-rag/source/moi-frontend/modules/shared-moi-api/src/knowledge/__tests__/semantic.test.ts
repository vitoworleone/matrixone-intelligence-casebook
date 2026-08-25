import { describe, expect, it, vi } from 'vitest';

import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import type { ApiResponse } from '../../types';
import {
  appendSemanticModelSourcesApi,
  backfillLegacySemanticModelSourcesApi,
  checkSemanticModelSourceExistenceApi,
  createEmptySemanticModelApi,
  createSemanticModelSegmentApi,
  createSemanticModelWithSourcesApi,
  deleteSemanticModelSegmentApi,
  deleteSemanticModelSourceApi,
  getSemanticModelSourceDocumentApi,
  importInitialSemanticModelSegmentsApi,
  listSemanticModelSourceJobsApi,
  listSemanticModelSourcesApi,
  previewSemanticModelArtifactApi,
  previewSemanticModelSourceFileApi,
  previewSemanticModelSourceSelectionsApi,
  reconcileSemanticModelSourceJobsApi,
  reembedSemanticModelSegmentsApi,
  setCurrentSemanticModelSegmentVersionApi,
  updateSemanticModelSegmentApi,
  updateSemanticModelSegmentEnabledApi,
  updateSemanticModelSourceGovernanceApi,
} from '../index';
import type {
  AppendSemanticModelSourcesResponse,
  CheckSemanticModelSourceExistenceResponse,
  CreateEmptySemanticModelResponse,
  CreateSemanticModelWithSourcesResponse,
  SemanticModelMutationResponse,
  SemanticModelSourceDocument,
  SemanticModelSourceJobListResponse,
  SemanticModelSourceListResponse,
  UpdateSemanticModelSourceGovernanceResponse,
} from '../semantic.types';

describe('semantic model artifact preview API', () => {
  it('uses the model-owned artifact route and requests a blob', async () => {
    const blob = new Blob(['image-bytes'], { type: 'image/png' });
    const get = vi.fn().mockResolvedValue({
      headers: { 'content-type': 'image/png' },
      data: blob,
    });
    const http = { get } as unknown as AppHttpClient;

    const result = await previewSemanticModelArtifactApi(42, 'page/image-9', http, { requestId: 'preview-request' });

    expect(get).toHaveBeenCalledWith('/semantic-models/42/artifacts/page%2Fimage-9/preview', {
      requestId: 'preview-request',
      responseContentType: 'blob',
      responseType: 'blob',
    });
    expect(result).toEqual({
      headers: { 'content-type': 'image/png' },
      data: blob,
    });
  });
});

describe('semantic model source-file preview API', () => {
  it('uses the model-scoped workflow source file route and requests a blob', async () => {
    const blob = new Blob(['document-bytes'], { type: 'application/pdf' });
    const get = vi.fn().mockResolvedValue({
      headers: { 'content-type': 'application/pdf' },
      data: blob,
    });
    const http = { get } as unknown as AppHttpClient;

    const result = await previewSemanticModelSourceFileApi(42, 'source/file-9', http, { requestId: 'source-preview-request' });

    expect(get).toHaveBeenCalledWith('/semantic-models/42/sources/file/source%2Ffile-9/preview', {
      requestId: 'source-preview-request',
      responseContentType: 'blob',
      responseType: 'blob',
    });
    expect(result).toEqual({
      headers: { 'content-type': 'application/pdf' },
      data: blob,
    });
  });
});

describe('semantic model source-list API', () => {
  it('calls the backend source-list route and returns the response envelope', async () => {
    const response: ApiResponse<SemanticModelSourceListResponse> = {
      code: 'OK',
      msg: 'OK',
      data: {
        total: 1,
        items: [
          {
            row_id: '7:table:sales_db:orders',
            source_type: 'table',
            model_id: 7,
            resource_id: 'sales_db.orders',
            display_name: 'orders',
            path: ['sales_db', 'orders'],
            source_path: 'sales_db/orders',
            db_name: 'sales_db',
            table_name: 'orders',
            size_bytes: null,
            row_count: null,
            ingest_status: 'unsupported',
            enabled: null,
            expires_at: null,
            expired: false,
            effective_enabled: true,
            force_enabled_after_expiry: false,
            tags: ['finance'],
            segment_version_id: null,
            index_version: null,
            updated_at: 1782705000,
            error: null,
            governance_status: 'managed',
            legacy_origin: null,
          },
          {
            row_id: 'candidate:7:file:legacy-file-1',
            source_type: 'file',
            model_id: 7,
            resource_id: 'legacy-file-1',
            display_name: 'legacy.pdf',
            path: ['catalog', 'legacy'],
            source_path: 'catalog/legacy',
            db_name: null,
            table_name: null,
            size_bytes: 1024,
            row_count: null,
            ingest_status: 'succeeded',
            enabled: null,
            expires_at: null,
            expired: false,
            effective_enabled: false,
            force_enabled_after_expiry: false,
            tags: [],
            segment_version_id: null,
            index_version: null,
            updated_at: 1782705100,
            error: null,
            governance_status: 'legacy_unbound',
            legacy_origin: 'lineage_register',
          },
        ],
        legacy_backfill_required: true,
      },
    };
    const http = { get: vi.fn().mockResolvedValue({ data: response }) } as unknown as AppHttpClient;

    const result = await listSemanticModelSourcesApi(7, http);

    expect(http.get).toHaveBeenCalledWith('/semantic-models/7/sources');
    expect(result).toBe(response);
    expect(result.data?.items[1]?.governance_status).toBe('legacy_unbound');
    expect(result.data?.legacy_backfill_required).toBe(true);
  });

  it('passes source-list page and page size query params', async () => {
    const response: ApiResponse<SemanticModelSourceListResponse> = {
      code: 'OK',
      msg: 'OK',
      data: {
        total: 21,
        page: 2,
        page_size: 20,
        items: [],
      },
    };
    const http = { get: vi.fn().mockResolvedValue({ data: response }) } as unknown as AppHttpClient;

    const result = await listSemanticModelSourcesApi(7, http, { page: 2, page_size: 20 });

    expect(http.get).toHaveBeenCalledWith('/semantic-models/7/sources?page=2&page_size=20');
    expect(result).toBe(response);
  });
});

describe('semantic model source existence API', () => {
  it('posts current-page source ids to the backend membership route', async () => {
    const response: ApiResponse<CheckSemanticModelSourceExistenceResponse> = {
      code: 'OK',
      msg: 'OK',
      data: {
        file_ids: ['file-1'],
        table_ids: [1002],
      },
    };
    const http = { post: vi.fn().mockResolvedValue({ data: response }) } as unknown as AppHttpClient;

    const result = await checkSemanticModelSourceExistenceApi(
      7,
      { file_ids: ['file-1', 'file-2'], table_ids: [1001, 1002] },
      http,
    );

    expect(http.post).toHaveBeenCalledWith('/semantic-models/7/sources/existence', {
      file_ids: ['file-1', 'file-2'],
      table_ids: [1001, 1002],
    });
    expect(result).toBe(response);
  });
});

describe('semantic model legacy source backfill API', () => {
  it('calls the backend legacy backfill route without request body', async () => {
    const response: ApiResponse<SemanticModelMutationResponse> = {
      code: 'OK',
      msg: 'OK',
      data: { updated: true },
    };
    const http = { post: vi.fn().mockResolvedValue({ data: response }) } as unknown as AppHttpClient;

    const result = await backfillLegacySemanticModelSourcesApi(7, http);

    expect(http.post).toHaveBeenCalledWith('/semantic-models/7/sources/backfill-legacy');
    expect(result).toBe(response);
  });
});

describe('semantic model source delete API', () => {
  it('calls the backend source delete route with encoded source row id', async () => {
    const response: ApiResponse<SemanticModelMutationResponse> = {
      code: 'OK',
      msg: 'OK',
      data: { deleted: true },
    };
    const http = { delete: vi.fn().mockResolvedValue({ data: response }) } as unknown as AppHttpClient;

    const result = await deleteSemanticModelSourceApi(7, '7:file:kb-file-1', http);

    expect(http.delete).toHaveBeenCalledWith('/semantic-models/7/sources/7%3Afile%3Akb-file-1');
    expect(result).toBe(response);
  });
});

describe('semantic model source document API', () => {
  it('calls the backend document route and returns the response envelope', async () => {
    const response: ApiResponse<SemanticModelSourceDocument> = {
      code: 'OK',
      msg: 'OK',
      data: {
        source: {
          row_id: '7:file:file-1',
          source_id: 'source-1',
          source_type: 'file',
          model_id: 7,
          resource_id: 'file-1',
          source_resource_id: 'catalog-file-1',
          kb_resource_id: 'file-1',
          display_name: 'manual.pdf',
          path: [],
          source_path: null,
          db_name: null,
          table_name: null,
          size_bytes: 2048,
          row_count: null,
          ingest_status: 'ready',
          enabled: true,
          expires_at: null,
          expired: false,
          effective_enabled: true,
          force_enabled_after_expiry: false,
          tags: ['product'],
          segment_version_id: 'segment-v1',
          index_version: 3,
          updated_at: 1782705000,
          error: null,
        },
        preview: {
          available: false,
          reason: 'parsed content is unavailable',
        },
        file_info: {
          tags: ['product'],
          expires_at: null,
          enabled: true,
          expired: false,
          effective_enabled: true,
          force_enabled_after_expiry: false,
          index_version: 3,
          segment_version_id: 'segment-v1',
        },
        segment_status: {
          available: true,
          total: 1,
        },
        current_segment_version_id: 'segment-v1',
        current_index_version: 3,
        selected_segment_version_id: 'segment-v1',
        selected_index_version: 3,
        segment_versions: [{ version_id: 'segment-v1', current: true, index_version: 3, status: 'committed' }],
        segments: [
          {
            segment_id: 'seg-1',
            level: 'chunk',
            chunk_index: 0,
            content: 'chunk text',
            word_count: 10,
            recall_count: 2,
            enabled: true,
          },
        ],
      },
    };
    const http = { get: vi.fn().mockResolvedValue({ data: response }) } as unknown as AppHttpClient;

    const result = await getSemanticModelSourceDocumentApi(7, '7:file:file-1', http, {
      segment_version_id: 'segment-v1',
    });

    expect(http.get).toHaveBeenCalledWith('/semantic-models/7/sources/7%3Afile%3Afile-1/document?segment_version_id=segment-v1');
    expect(result).toBe(response);
  });

  it('calls segment mutation routes with base version payloads', async () => {
    const document = {
      source: {
        row_id: '7:file:file-1',
        source_type: 'file' as const,
        model_id: 7,
        resource_id: 'file-1',
        display_name: 'manual.pdf',
        path: [],
        source_path: null,
        db_name: null,
        table_name: null,
        size_bytes: 2048,
        row_count: null,
        ingest_status: 'ready',
        enabled: true,
        expires_at: null,
        expired: false,
        effective_enabled: true,
        force_enabled_after_expiry: false,
        tags: [],
        segment_version_id: 'segment-v2',
        index_version: 4,
        error: null,
      },
      preview: { available: true, content: 'preview' },
      file_info: {
        tags: [],
        expires_at: null,
        enabled: true,
        expired: false,
        effective_enabled: true,
        force_enabled_after_expiry: false,
        index_version: 4,
        segment_version_id: 'segment-v2',
      },
      segment_status: { available: true, total: 1 },
      current_segment_version_id: 'segment-v2',
      current_index_version: 4,
      selected_segment_version_id: 'segment-v2',
      selected_index_version: 4,
      segment_versions: [{ version_id: 'segment-v2', current: true, index_version: 4 }],
      segments: [],
    };
    const response: ApiResponse<{ document: typeof document }> = {
      code: 'OK',
      msg: 'OK',
      data: { document },
    };
    const body = { base_segment_version_id: 'segment-v1', base_index_version: 3 };
    const http = {
      post: vi.fn().mockResolvedValue({ data: response }),
      patch: vi.fn().mockResolvedValue({ data: response }),
      delete: vi.fn().mockResolvedValue({ data: response }),
    } as unknown as AppHttpClient;

    await importInitialSemanticModelSegmentsApi(7, '7:file:file-1', body, http);
    await updateSemanticModelSegmentApi(7, '7:file:file-1', 'seg-1', { ...body, content: 'new text' }, http);
    await createSemanticModelSegmentApi(7, '7:file:file-1', { ...body, level: 'chunk', content: 'created' }, http);
    await updateSemanticModelSegmentEnabledApi(7, '7:file:file-1', 'seg-1', { ...body, enabled: false }, http);
    await deleteSemanticModelSegmentApi(7, '7:file:file-1', 'seg-1', body, http);
    await reembedSemanticModelSegmentsApi(7, '7:file:file-1', body, http);
    await setCurrentSemanticModelSegmentVersionApi(7, '7:file:file-1', 'segment-v2', body, http);

    expect(http.post).toHaveBeenNthCalledWith(1, '/semantic-models/7/sources/7%3Afile%3Afile-1/segments/import-initial', body);
    expect(http.patch).toHaveBeenNthCalledWith(1, '/semantic-models/7/sources/7%3Afile%3Afile-1/segments/seg-1', {
      ...body,
      content: 'new text',
    });
    expect(http.post).toHaveBeenNthCalledWith(2, '/semantic-models/7/sources/7%3Afile%3Afile-1/segments', {
      ...body,
      level: 'chunk',
      content: 'created',
    });
    expect(http.patch).toHaveBeenNthCalledWith(2, '/semantic-models/7/sources/7%3Afile%3Afile-1/segments/seg-1/enabled', {
      ...body,
      enabled: false,
    });
    expect(http.delete).toHaveBeenNthCalledWith(1, '/semantic-models/7/sources/7%3Afile%3Afile-1/segments/seg-1', {
      data: body,
    });
    expect(http.post).toHaveBeenNthCalledWith(3, '/semantic-models/7/sources/7%3Afile%3Afile-1/segments/re-embedding', body);
    expect(http.patch).toHaveBeenNthCalledWith(
      3,
      '/semantic-models/7/sources/7%3Afile%3Afile-1/segment-versions/segment-v2/current',
      body,
    );
  });

  it('calls the governance route with partial fields and returns the response envelope', async () => {
    const request = {
      tags: ['product', 'expired-review'],
      expires_at: 1782705000,
      enabled: false,
      force_enabled_after_expiry: true,
    };
    const response: ApiResponse<UpdateSemanticModelSourceGovernanceResponse> = {
      code: 'OK',
      msg: 'OK',
      data: {
        source: {
          row_id: '7:file:file-1',
          source_type: 'file',
          model_id: 7,
          resource_id: 'file-1',
          display_name: 'manual.pdf',
          path: [],
          source_path: null,
          db_name: null,
          table_name: null,
          size_bytes: 2048,
          row_count: null,
          ingest_status: 'ready',
          enabled: false,
          expires_at: 1782705000,
          expired: true,
          effective_enabled: true,
          force_enabled_after_expiry: true,
          tags: ['product', 'expired-review'],
          segment_version_id: null,
          index_version: 2,
          updated_at: 1782705000,
          error: null,
        },
      },
    };
    const http = { patch: vi.fn().mockResolvedValue({ data: response }) } as unknown as AppHttpClient;

    const result = await updateSemanticModelSourceGovernanceApi(7, '7:file:file-1', request, http);

    expect(http.patch).toHaveBeenCalledWith('/semantic-models/7/sources/7%3Afile%3Afile-1/governance', request);
    expect(result).toBe(response);
  });
});

describe('semantic model source-jobs API', () => {
  it('calls the reconcile route without request body', async () => {
    const response: ApiResponse<SemanticModelMutationResponse> = {
      code: 'OK',
      msg: 'OK',
      data: { updated: true },
    };
    const http = { post: vi.fn().mockResolvedValue({ data: response }) } as unknown as AppHttpClient;

    const result = await reconcileSemanticModelSourceJobsApi(7, http);

    expect(http.post).toHaveBeenCalledWith('/semantic-models/7/source-jobs/reconcile');
    expect(result).toBe(response);
  });
});

describe('semantic model source-jobs API', () => {
  it('calls the backend source-jobs route and returns the response envelope', async () => {
    const response: ApiResponse<SemanticModelSourceJobListResponse> = {
      code: 'OK',
      msg: 'OK',
      data: {
        total: 1,
        reconcile_required: true,
        items: [
          {
            job_id: 'job-1',
            source_id: 'source-1',
            job_status: 'queued',
            source_file_id: null,
            kb_file_id: 'kb-file-1',
            source_table_id: null,
            kb_table_id: null,
            error: null,
          },
        ],
      },
    };
    const http = { get: vi.fn().mockResolvedValue({ data: response }) } as unknown as AppHttpClient;

    const result = await listSemanticModelSourceJobsApi(7, http);

    expect(http.get).toHaveBeenCalledWith('/semantic-models/7/source-jobs');
    expect(result).toBe(response);
  });
});

describe('semantic model source-selection preview API', () => {
  it('uses create and append preview routes with the complete selection payload', async () => {
    const request = {
      source_selections: [
        {
          kind: 'volume_files' as const,
          volume_id: 42,
          all_selected: true,
          excluded_file_ids: [],
          filters: { file_name: 'report' },
        },
      ],
    };
    const response = {
      code: 'OK' as const,
      msg: 'OK',
      data: { file_count: 100, table_count: 0, total_count: 100 },
    };
    const http = {
      post: vi.fn().mockResolvedValue({ data: response }),
    } as unknown as AppHttpClient;

    await expect(previewSemanticModelSourceSelectionsApi(request, http)).resolves.toBe(response);
    const controller = new AbortController();
    await expect(previewSemanticModelSourceSelectionsApi(request, http, 7, controller.signal)).resolves.toBe(response);

    expect(http.post).toHaveBeenNthCalledWith(1, '/semantic-models/source-selections/preview', request);
    expect(http.post).toHaveBeenNthCalledWith(2, '/semantic-models/7/source-selections/preview', request, {
      signal: controller.signal,
    });
  });
});

describe('semantic model create-with-sources API', () => {
  it('calls the backend create route with source payload and returns the response envelope', async () => {
    const request = {
      name: 'phase2 kb',
      description: 'phase2 docs',
      files: {
        file_ids: [],
        vector_table: 'kb_phase2_text_idx',
        embedding_model: 'BAAI/bge-m3',
        image_vector_table: 'kb_phase2_text_idx_img',
        image_embedding_model: 'efficientnet-b3',
        image_embedding_dimension: 1536,
        image_embedding_backend_id: '-30010',
        image_preprocess_version: 'efficientnet-b3-v1-rgb-300-letterbox-imagenet',
        image_distance_metric: 'cosine',
      },
      sources: [
        {
          source_type: 'local_file' as const,
          file_name: 'local.txt',
          file_id: 'local-file-1',
          upload_kind: 'structured' as const,
          table_config: '{"new_table":true,"create_table":{"name":"structured_table"}}',
        },
        { source_type: 'catalog_file' as const, file_id: 'catalog-file-1', volume_id: 41 },
        { source_type: 'catalog_table' as const, table_id: 1001 },
      ],
    };
    const response: ApiResponse<CreateSemanticModelWithSourcesResponse> = {
      code: 'OK',
      msg: 'OK',
      data: {
        model: {
          id: 77,
          name: 'phase2 kb',
          description: 'phase2 docs',
          tables: [],
          files: {
            file_ids: ['kb-local-file', 'kb-catalog-copy'],
            vector_table: 'kb_phase2_text_idx',
            embedding_model: 'BAAI/bge-m3',
            image_vector_table: 'kb_phase2_text_idx_img',
            image_embedding_model: 'efficientnet-b3',
            image_embedding_dimension: 1536,
            image_embedding_backend_id: '-30010',
            image_preprocess_version: 'efficientnet-b3-v1-rgb-300-letterbox-imagenet',
            image_distance_metric: 'cosine',
          },
          source_counts: { files: 2, tables: 0, total: 2 },
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
          last_checked_at: 1782705000,
        },
        sources: [
          {
            row_id: '77:file:kb-local-file',
            source_type: 'file',
            model_id: 77,
            resource_id: 'kb-local-file',
            display_name: 'local.txt',
            path: [],
            db_name: null,
            table_name: null,
            ingest_status: 'pending',
            enabled: null,
            expires_at: null,
            expired: false,
            effective_enabled: true,
            force_enabled_after_expiry: false,
            tags: [],
            segment_version_id: null,
            index_version: null,
            error: null,
          },
        ],
        jobs: [
          {
            job_id: 'job-1',
            source_id: 'source-1',
            source_file_id: null,
            kb_file_id: 'kb-local-file',
            source_table_id: null,
            kb_table_id: null,
            job_status: 'queued',
            error: null,
          },
        ],
      },
    };
    const http = { post: vi.fn().mockResolvedValue({ data: response }) } as unknown as AppHttpClient;

    const result = await createSemanticModelWithSourcesApi(request, http);

    expect(http.post).toHaveBeenCalledWith('/semantic-models/create-with-sources', request);
    expect(result).toBe(response);
  });
});

describe('semantic model create-empty API', () => {
  it('creates a data-side knowledge base without source fields', async () => {
    const request = {
      name: 'data kb',
      description: 'created from the data knowledge page',
      image_index_enabled: true,
    };
    const response: ApiResponse<CreateEmptySemanticModelResponse> = {
      code: 'OK',
      msg: 'OK',
      data: {
        model: {
          id: 77,
          name: 'data kb',
          description: 'created from the data knowledge page',
          tables: [],
          files: {
            file_ids: [],
            vector_table: 'kb_77_text_index',
            embedding_model: 'bge-m3',
          },
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
          last_checked_at: 1782705000,
        },
      },
    };
    const post = vi.fn().mockResolvedValue({ data: response });
    const http = { post } as unknown as AppHttpClient;

    const result = await createEmptySemanticModelApi(request, http);

    expect(post).toHaveBeenCalledWith('/semantic-models/create-empty', request);
    expect(post.mock.calls[0]?.[1]).not.toHaveProperty('sources');
    expect(post.mock.calls[0]?.[1]).not.toHaveProperty('source_selections');
    expect(result).toBe(response);
  });
});

describe('semantic model append sources API', () => {
  it('calls the backend append route with source payload and returns the response envelope', async () => {
    const request = {
      sources: [
        {
          source_type: 'local_file' as const,
          file_name: 'append.txt',
          file_id: 'append-file-1',
          upload_kind: 'unstructured' as const,
        },
        { source_type: 'catalog_file' as const, file_id: 'catalog-file-append', volume_id: 41 },
        { source_type: 'catalog_table' as const, table_id: 1002 },
      ],
    };
    const response: ApiResponse<AppendSemanticModelSourcesResponse> = {
      code: 'OK',
      msg: 'OK',
      data: {
        data_domain: {
          model_id: 77,
          catalog_id: 3,
          database_id: 11,
          raw_volume_id: 12,
          processed_volume_id: 13,
          ensure_status: 'ready',
          last_ensure_error: null,
          last_checked_at: 1782705000,
        },
        sources: [
          {
            row_id: '77:file:kb-append-file',
            source_type: 'file',
            model_id: 77,
            resource_id: 'kb-append-file',
            display_name: 'append.txt',
            path: [],
            db_name: null,
            table_name: null,
            ingest_status: 'pending',
            enabled: null,
            expires_at: null,
            expired: false,
            effective_enabled: true,
            force_enabled_after_expiry: false,
            tags: [],
            segment_version_id: null,
            index_version: null,
            error: null,
          },
        ],
        jobs: [
          {
            job_id: 'job-append-1',
            source_id: '77:file:kb-append-file',
            source_file_id: null,
            kb_file_id: 'kb-append-file',
            source_table_id: null,
            kb_table_id: null,
            job_status: 'queued',
            error: null,
          },
        ],
      },
    };
    const http = { post: vi.fn().mockResolvedValue({ data: response }) } as unknown as AppHttpClient;

    const result = await appendSemanticModelSourcesApi(77, request, http);

    expect(http.post).toHaveBeenCalledWith('/semantic-models/77/sources', request);
    expect(result).toBe(response);
  });
});
