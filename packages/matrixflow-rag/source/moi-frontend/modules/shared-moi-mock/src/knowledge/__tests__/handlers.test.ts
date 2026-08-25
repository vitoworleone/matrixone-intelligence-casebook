import { setupServer } from 'msw/node';
import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest';

import {
  createKnowledgeAppendSourcesResponseFixture,
  createKnowledgeCreateResponseFixture,
  createKnowledgeSourceFixture,
} from '../factories';
import { createKnowledgeHandlers } from '../handlers';

describe('knowledge mock append sources handlers', () => {
  const createResponse = createKnowledgeCreateResponseFixture({
    sources: [],
    jobs: [],
  });
  const appendResponse = createKnowledgeAppendSourcesResponseFixture({
    data_domain: { ...createResponse.data_domain, model_id: createResponse.model.id },
  });
  const server = setupServer(
    ...createKnowledgeHandlers(
      {
        createResponse,
        appendResponse,
      },
      'http://localhost',
    ),
  );

  beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
  afterEach(() => server.resetHandlers());
  afterAll(() => server.close());

  it('stores appended sources and jobs for later list calls', async () => {
    const append = await fetch(`http://localhost/semantic-models/${createResponse.model.id}/sources`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sources: [{ source_type: 'catalog_file', file_id: 'catalog-file-1', volume_id: 41 }] }),
    });

    expect(append.status).toBe(200);
    const appendBody = await append.json();
    expect(appendBody.data.sources).toHaveLength(1);
    expect(appendBody.data.jobs).toHaveLength(1);

    const sources = await fetch(`http://localhost/semantic-models/${createResponse.model.id}/sources`);
    const sourcesBody = await sources.json();
    expect(sourcesBody.data.items).toEqual(appendResponse.sources);

    const jobs = await fetch(`http://localhost/semantic-models/${createResponse.model.id}/source-jobs`);
    const jobsBody = await jobs.json();
    expect(jobsBody.data.items).toEqual(appendResponse.jobs);
  });

  it('returns source job reconcile success without mutating mock lists', async () => {
    const beforeSources = await fetch(`http://localhost/semantic-models/${createResponse.model.id}/sources`);
    const beforeSourcesBody = await beforeSources.json();
    const beforeJobs = await fetch(`http://localhost/semantic-models/${createResponse.model.id}/source-jobs`);
    const beforeJobsBody = await beforeJobs.json();

    const reconcile = await fetch(`http://localhost/semantic-models/${createResponse.model.id}/source-jobs/reconcile`, {
      method: 'POST',
    });

    expect(reconcile.status).toBe(200);
    const reconcileBody = await reconcile.json();
    expect(reconcileBody.data.updated).toBe(true);

    const afterSources = await fetch(`http://localhost/semantic-models/${createResponse.model.id}/sources`);
    const afterSourcesBody = await afterSources.json();
    const afterJobs = await fetch(`http://localhost/semantic-models/${createResponse.model.id}/source-jobs`);
    const afterJobsBody = await afterJobs.json();
    expect(afterSourcesBody.data).toEqual(beforeSourcesBody.data);
    expect(afterJobsBody.data).toEqual(beforeJobsBody.data);
  });

  it('returns legacy source backfill success', async () => {
    const backfill = await fetch(`http://localhost/semantic-models/${createResponse.model.id}/sources/backfill-legacy`, {
      method: 'POST',
    });

    expect(backfill.status).toBe(200);
    const backfillBody = await backfill.json();
    expect(backfillBody.data.updated).toBe(true);
  });

  it('echoes create-with-sources vector config into response model files', async () => {
    const create = await fetch('http://localhost/semantic-models/create-with-sources', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: 'KB',
        files: {
          file_ids: [],
          parents: [],
          vector_table: 'kb_text_idx',
          embedding_model: 'BAAI/bge-m3',
          image_vector_table: 'kb_text_idx_img',
          image_embedding_model: 'efficientnet-b3',
          image_embedding_backend_id: '-30010',
          image_embedding_dimension: 1536,
          image_preprocess_version: 'efficientnet-b3-v1-rgb-300-letterbox-imagenet',
          image_distance_metric: 'cosine',
        },
        sources: [],
      }),
    });

    expect(create.status).toBe(200);
    const createBody = await create.json();
    expect(createBody.data.model.files).toMatchObject({
      file_ids: createResponse.model.files?.file_ids,
      vector_table: 'kb_text_idx',
      embedding_model: 'BAAI/bge-m3',
      image_vector_table: 'kb_text_idx_img',
      image_embedding_model: 'efficientnet-b3',
      image_embedding_backend_id: '-30010',
      image_embedding_dimension: 1536,
      image_preprocess_version: 'efficientnet-b3-v1-rgb-300-letterbox-imagenet',
      image_distance_metric: 'cosine',
    });
  });

  it('creates an empty knowledge base without source or job response fields', async () => {
    const create = await fetch('http://localhost/semantic-models/create-empty', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: 'Data KB', description: 'data page', image_index_enabled: true }),
    });

    expect(create.status).toBe(200);
    const body = await create.json();
    expect(body.data).toMatchObject({
      model: { name: 'Data KB', description: 'data page', tables: [], source_counts: { files: 0, tables: 0, total: 0 } },
      data_domain: createResponse.data_domain,
    });
    expect(body.data).not.toHaveProperty('sources');
    expect(body.data).not.toHaveProperty('jobs');
  });
});

describe('knowledge mock legacy source handlers', () => {
  const modelId = 9201;
  const server = setupServer(
    ...createKnowledgeHandlers(
      {
        createResponse: createKnowledgeCreateResponseFixture({
          model: {
            id: modelId,
            name: 'KB',
            description: '',
            tables: [],
            files: { file_ids: [] },
            source_counts: { files: 0, tables: 0, total: 0 },
            table_set_hash: '',
            created_at: 0,
            updated_at: 0,
          },
          sources: [],
          jobs: [],
        }),
        legacySources: [
          createKnowledgeSourceFixture({
            row_id: 'candidate:9201:file:legacy-1',
            source_id: '',
            model_id: modelId,
            resource_id: 'legacy-1',
            display_name: 'legacy-1.pdf',
            governance_status: 'legacy_unbound',
            legacy_origin: 'semantic_model_explicit',
          }),
          createKnowledgeSourceFixture({
            row_id: 'candidate:9201:file:legacy-2',
            source_id: '',
            model_id: modelId,
            resource_id: 'legacy-2',
            display_name: 'legacy-2.pdf',
            governance_status: 'legacy_unbound',
            legacy_origin: 'lineage_register',
          }),
        ],
        legacyBackfillBatchSize: 1,
      },
      'http://localhost',
    ),
  );

  beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
  afterEach(() => server.resetHandlers());
  afterAll(() => server.close());

  it('backfills legacy candidates in bounded batches', async () => {
    const before = await fetch(`http://localhost/semantic-models/${modelId}/sources`);
    const beforeBody = await before.json();
    expect(beforeBody.data.legacy_backfill_required).toBe(true);
    expect(beforeBody.data.items.map((item: { governance_status?: string }) => item.governance_status)).toEqual([
      'legacy_unbound',
      'legacy_unbound',
    ]);

    await fetch(`http://localhost/semantic-models/${modelId}/sources/backfill-legacy`, { method: 'POST' });
    const afterFirst = await fetch(`http://localhost/semantic-models/${modelId}/sources`);
    const afterFirstBody = await afterFirst.json();
    expect(afterFirstBody.data.legacy_backfill_required).toBe(true);
    expect(afterFirstBody.data.items.map((item: { governance_status?: string }) => item.governance_status)).toEqual([
      'managed',
      'legacy_unbound',
    ]);

    await fetch(`http://localhost/semantic-models/${modelId}/sources/backfill-legacy`, { method: 'POST' });
    const afterSecond = await fetch(`http://localhost/semantic-models/${modelId}/sources`);
    const afterSecondBody = await afterSecond.json();
    expect(afterSecondBody.data.legacy_backfill_required).toBe(false);
    expect(afterSecondBody.data.items.map((item: { governance_status?: string }) => item.governance_status)).toEqual([
      'managed',
      'managed',
    ]);
  });
});

describe('knowledge mock source document handlers', () => {
  const documentSource = createKnowledgeSourceFixture({
    row_id: '9001:file:kb-file-1',
    model_id: 9001,
    resource_id: 'kb-file-1',
    display_name: 'manual.pdf',
    source_path: 'raw/manual.pdf',
    size_bytes: 2048,
    enabled: true,
    tags: ['product'],
    updated_at: 1782705000,
    index_version: 2,
    segment_version_id: 'segment-v1',
  });
  const createResponse = createKnowledgeCreateResponseFixture({
    model: {
      id: 9001,
      name: 'KB',
      description: '',
      tables: [],
      files: { file_ids: ['kb-file-1'] },
      source_counts: { files: 1, tables: 0, total: 1 },
      table_set_hash: '',
      created_at: 0,
      updated_at: 0,
    },
    sources: [documentSource],
    jobs: [],
  });
  const server = setupServer(...createKnowledgeHandlers({ createResponse }, 'http://localhost'));

  beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
  afterEach(() => server.resetHandlers());
  afterAll(() => server.close());

  it('returns preview and segment state for document detail', async () => {
    const detail = await fetch(
      `http://localhost/semantic-models/9001/sources/${encodeURIComponent(documentSource.row_id)}/document`,
    );

    expect(detail.status).toBe(200);
    const body = await detail.json();
    expect(body.data.source.row_id).toBe(documentSource.row_id);
    expect(body.data.preview.available).toBe(true);
    expect(body.data.segment_status.available).toBe(true);
    expect(body.data.segments).toHaveLength(1);
    expect(body.data.file_info.tags).toEqual(['product']);
  });

  it('persists segment edits as a new current version', async () => {
    const detail = await fetch(
      `http://localhost/semantic-models/9001/sources/${encodeURIComponent(documentSource.row_id)}/document`,
    );
    const detailBody = await detail.json();
    const segment = detailBody.data.segments[0];

    const edited = await fetch(
      `http://localhost/semantic-models/9001/sources/${encodeURIComponent(documentSource.row_id)}/segments/${segment.segment_id}`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          base_segment_version_id: detailBody.data.current_segment_version_id,
          base_index_version: detailBody.data.current_index_version,
          content: 'edited segment text',
        }),
      },
    );

    expect(edited.status).toBe(200);
    const editedBody = await edited.json();
    expect(editedBody.data.document.current_index_version).toBe(detailBody.data.current_index_version + 1);
    expect(editedBody.data.document.current_segment_version_id).not.toBe(detailBody.data.current_segment_version_id);
    expect(editedBody.data.document.segments[0].content).toBe('edited segment text');
    expect(editedBody.data.document.source.segment_version_id).toBe(editedBody.data.document.current_segment_version_id);
  });

  it('rejects segment delete when base version is stale', async () => {
    const detail = await fetch(
      `http://localhost/semantic-models/9001/sources/${encodeURIComponent(documentSource.row_id)}/document`,
    );
    const detailBody = await detail.json();
    const segment = detailBody.data.segments[0];

    const deleted = await fetch(
      `http://localhost/semantic-models/9001/sources/${encodeURIComponent(documentSource.row_id)}/segments/${segment.segment_id}`,
      {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          base_segment_version_id: 'stale-version',
          base_index_version: 0,
        }),
      },
    );

    expect(deleted.status).toBe(409);
    const deletedBody = await deleted.json();
    expect(deletedBody.code).toBe('ErrConflict');
  });

  it('persists segment delete as a new current version', async () => {
    const detail = await fetch(
      `http://localhost/semantic-models/9001/sources/${encodeURIComponent(documentSource.row_id)}/document`,
    );
    const detailBody = await detail.json();
    const segment = detailBody.data.segments[0];

    const deleted = await fetch(
      `http://localhost/semantic-models/9001/sources/${encodeURIComponent(documentSource.row_id)}/segments/${segment.segment_id}`,
      {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          base_segment_version_id: detailBody.data.current_segment_version_id,
          base_index_version: detailBody.data.current_index_version,
        }),
      },
    );

    expect(deleted.status).toBe(200);
    const deletedBody = await deleted.json();
    expect(deletedBody.data.document.current_index_version).toBe(detailBody.data.current_index_version + 1);
    expect(deletedBody.data.document.current_segment_version_id).not.toBe(detailBody.data.current_segment_version_id);
    expect(deletedBody.data.document.segment_versions[0].source).toBe('delete_chunk');
    expect(
      deletedBody.data.document.segments.some((item: { segment_id: string }) => item.segment_id === segment.segment_id),
    ).toBe(false);
  });

  it('persists governance patch for later detail and list calls', async () => {
    const patch = await fetch(
      `http://localhost/semantic-models/9001/sources/${encodeURIComponent(documentSource.row_id)}/governance`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          tags: ['product', 'phase4'],
          expires_at: 1,
          enabled: false,
          force_enabled_after_expiry: true,
        }),
      },
    );

    expect(patch.status).toBe(200);
    const patchBody = await patch.json();
    expect(patchBody.data.source.tags).toEqual(['product', 'phase4']);
    expect(patchBody.data.source.expired).toBe(true);
    expect(patchBody.data.source.effective_enabled).toBe(false);

    const sources = await fetch('http://localhost/semantic-models/9001/sources');
    const sourcesBody = await sources.json();
    expect(sourcesBody.data.items[0].tags).toEqual(['product', 'phase4']);
    expect(sourcesBody.data.items[0].source_path).toBe('raw/manual.pdf');
    expect(sourcesBody.data.items[0].size_bytes).toBe(2048);
    expect(sourcesBody.data.items[0].row_count).toBeNull();
    expect(sourcesBody.data.items[0].updated_at).toBe(1782705000);
  });

  it('keeps forced expired source effective only when source is enabled', async () => {
    const forcedEnabled = await fetch(
      `http://localhost/semantic-models/9001/sources/${encodeURIComponent(documentSource.row_id)}/governance`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          expires_at: 1,
          enabled: true,
          force_enabled_after_expiry: true,
        }),
      },
    );
    expect(forcedEnabled.status).toBe(200);
    const forcedEnabledBody = await forcedEnabled.json();
    expect(forcedEnabledBody.data.source.expired).toBe(true);
    expect(forcedEnabledBody.data.source.effective_enabled).toBe(true);

    const forcedDisabled = await fetch(
      `http://localhost/semantic-models/9001/sources/${encodeURIComponent(documentSource.row_id)}/governance`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          expires_at: 1,
          enabled: false,
          force_enabled_after_expiry: true,
        }),
      },
    );
    expect(forcedDisabled.status).toBe(200);
    const forcedDisabledBody = await forcedDisabled.json();
    expect(forcedDisabledBody.data.source.expired).toBe(true);
    expect(forcedDisabledBody.data.source.effective_enabled).toBe(false);
  });

  it('removes deleted source from later list and detail calls', async () => {
    const deleted = await fetch(`http://localhost/semantic-models/9001/sources/${encodeURIComponent(documentSource.row_id)}`, {
      method: 'DELETE',
    });
    expect(deleted.status).toBe(200);
    const deletedBody = await deleted.json();
    expect(deletedBody.data.deleted).toBe(true);

    const sources = await fetch('http://localhost/semantic-models/9001/sources');
    const sourcesBody = await sources.json();
    expect(sourcesBody.data.items).toEqual([]);

    const detail = await fetch(
      `http://localhost/semantic-models/9001/sources/${encodeURIComponent(documentSource.row_id)}/document`,
    );
    expect(detail.status).toBe(404);
  });
});

describe('knowledge mock table source governance handler', () => {
  const tableSource = createKnowledgeSourceFixture({
    row_id: '9002:table:2001',
    model_id: 9002,
    source_type: 'table',
    resource_id: '2001',
    source_resource_id: '1001',
    kb_resource_id: '2001',
    source_table_id: 1001,
    kb_table_id: 2001,
    display_name: 'orders',
    db_name: 'sales',
    table_name: 'orders',
    enabled: true,
    row_count: 32,
  });
  const createResponse = createKnowledgeCreateResponseFixture({
    model: {
      id: 9002,
      name: 'KB',
      description: '',
      tables: [{ db_name: 'sales', table_names: ['orders'] }],
      files: undefined,
      source_counts: { files: 0, tables: 1, total: 1 },
      table_set_hash: '',
      created_at: 0,
      updated_at: 0,
    },
    sources: [tableSource],
    jobs: [],
  });
  const server = setupServer(...createKnowledgeHandlers({ createResponse }, 'http://localhost'));

  beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
  afterEach(() => server.resetHandlers());
  afterAll(() => server.close());

  it('persists table source enabled state through governance patch', async () => {
    const patch = await fetch(
      `http://localhost/semantic-models/9002/sources/${encodeURIComponent(tableSource.row_id)}/governance`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: false }),
      },
    );

    expect(patch.status).toBe(200);
    const patchBody = await patch.json();
    expect(patchBody.data.source.enabled).toBe(false);
    expect(patchBody.data.source.effective_enabled).toBe(false);

    const sources = await fetch('http://localhost/semantic-models/9002/sources');
    const sourcesBody = await sources.json();
    expect(sourcesBody.data.items[0].enabled).toBe(false);
    expect(sourcesBody.data.items[0].row_count).toBe(32);
  });

  it('persists table source expiry through governance patch', async () => {
    const patch = await fetch(
      `http://localhost/semantic-models/9002/sources/${encodeURIComponent(tableSource.row_id)}/governance`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: true, expires_at: 1782705000 }),
      },
    );

    expect(patch.status).toBe(200);
    const patchBody = await patch.json();
    expect(patchBody.data.source.expires_at).toBe(1782705000);
    expect(patchBody.data.source.effective_enabled).toBe(true);

    const sources = await fetch('http://localhost/semantic-models/9002/sources');
    const sourcesBody = await sources.json();
    expect(sourcesBody.data.items[0].expires_at).toBe(1782705000);
  });

  it('rejects document governance fields for table source', async () => {
    const patch = await fetch(
      `http://localhost/semantic-models/9002/sources/${encodeURIComponent(tableSource.row_id)}/governance`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: true, tags: ['phase4'] }),
      },
    );

    expect(patch.status).toBe(400);
    const patchBody = await patch.json();
    expect(patchBody.msg).toBe('table source governance only supports enabled and expires_at');
  });
});

describe('knowledge mock append sources failure handler', () => {
  const createResponse = createKnowledgeCreateResponseFixture({
    sources: [],
    jobs: [],
  });
  const server = setupServer(
    ...createKnowledgeHandlers(
      {
        createResponse,
        appendFailure: {
          status: 403,
          code: 'Forbidden',
          msg: 'append denied',
        },
      },
      'http://localhost',
    ),
  );

  beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
  afterEach(() => server.resetHandlers());
  afterAll(() => server.close());

  it('returns append failure without mutating later list calls', async () => {
    const append = await fetch(`http://localhost/semantic-models/${createResponse.model.id}/sources`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sources: [{ source_type: 'catalog_file', file_id: 'catalog-file-1', volume_id: 41 }] }),
    });

    expect(append.status).toBe(403);
    const appendBody = await append.json();
    expect(appendBody.msg).toBe('append denied');

    const sources = await fetch(`http://localhost/semantic-models/${createResponse.model.id}/sources`);
    const sourcesBody = await sources.json();
    expect(sourcesBody.data.items).toEqual([]);

    const jobs = await fetch(`http://localhost/semantic-models/${createResponse.model.id}/source-jobs`);
    const jobsBody = await jobs.json();
    expect(jobsBody.data.items).toEqual([]);
  });
});
