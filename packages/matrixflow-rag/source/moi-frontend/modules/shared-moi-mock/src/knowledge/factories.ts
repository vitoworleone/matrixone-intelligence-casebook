import type {
  AppendSemanticModelSourcesResponse,
  CreateSemanticModelWithSourcesResponse,
  KnowledgeBaseDataDomain,
  KnowledgeBaseSourceJobRun,
  SemanticModel,
  SemanticModelDocumentSegment,
  SemanticModelSource,
  SemanticModelSourceDocument,
} from '@moi/shared-moi-api/knowledge';

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value));
}

export function createKnowledgeModelFixture(overrides: Partial<SemanticModel> = {}): SemanticModel {
  return {
    id: overrides.id ?? 9001,
    name: overrides.name ?? 'Mock Knowledge Base',
    description: overrides.description ?? 'Knowledge base created by mock route',
    tables: overrides.tables ?? [],
    files: overrides.files ?? { file_ids: ['kb-file-1'] },
    source_counts: overrides.source_counts ?? { files: 1, tables: 0, total: 1 },
    table_set_hash: overrides.table_set_hash ?? '',
    created_at: overrides.created_at ?? 1782705000,
    updated_at: overrides.updated_at ?? 0,
  };
}

export function createKnowledgeDataDomainFixture(overrides: Partial<KnowledgeBaseDataDomain> = {}): KnowledgeBaseDataDomain {
  return {
    model_id: overrides.model_id ?? 9001,
    catalog_id: overrides.catalog_id ?? 3,
    database_id: overrides.database_id ?? 9101,
    raw_volume_id: overrides.raw_volume_id ?? 9102,
    processed_volume_id: overrides.processed_volume_id ?? 9103,
    ensure_status: overrides.ensure_status ?? 'ready',
    last_ensure_error: overrides.last_ensure_error ?? null,
    last_checked_at: overrides.last_checked_at ?? 1782705000,
  };
}

type KnowledgeSourceJobFixtureOverrides = Partial<KnowledgeBaseSourceJobRun> & {
  model_id?: number;
  job_type?: string;
  idempotency_key?: string;
  operation_id?: string | null;
  retry_count?: number;
  next_retry_at?: number | null;
  workflow_execution_id?: string | null;
};

export function createKnowledgeSourceJobFixture(overrides: KnowledgeSourceJobFixtureOverrides = {}): KnowledgeBaseSourceJobRun {
  return {
    job_id: overrides.job_id ?? 'job-1',
    source_id: overrides.source_id ?? 'source-1',
    source_file_id: overrides.source_file_id ?? null,
    kb_file_id: overrides.kb_file_id ?? 'kb-file-1',
    source_table_id: overrides.source_table_id ?? null,
    kb_table_id: overrides.kb_table_id ?? null,
    job_status: overrides.job_status ?? 'queued',
    error: overrides.error ?? null,
    updated_at: overrides.updated_at ?? 1782705000,
    reconcile_required: overrides.reconcile_required ?? true,
  };
}

export function createKnowledgeSourceFixture(overrides: Partial<SemanticModelSource> = {}): SemanticModelSource {
  return {
    row_id: overrides.row_id ?? '9001:file:kb-file-1',
    source_id: overrides.source_id ?? overrides.row_id ?? '9001:file:kb-file-1',
    source_type: overrides.source_type ?? 'file',
    model_id: overrides.model_id ?? 9001,
    resource_id: overrides.resource_id ?? 'kb-file-1',
    source_resource_id: overrides.source_resource_id ?? null,
    kb_resource_id: overrides.kb_resource_id ?? overrides.resource_id ?? 'kb-file-1',
    display_name: overrides.display_name ?? 'mock.txt',
    path: overrides.path ?? [],
    source_path: overrides.source_path ?? null,
    db_name: overrides.db_name ?? null,
    table_name: overrides.table_name ?? null,
    size_bytes: overrides.size_bytes ?? null,
    row_count: overrides.row_count ?? null,
    ingest_status: overrides.ingest_status ?? 'pending',
    enabled: overrides.enabled ?? null,
    expires_at: overrides.expires_at ?? null,
    expired: overrides.expired ?? false,
    effective_enabled: overrides.effective_enabled ?? true,
    force_enabled_after_expiry: overrides.force_enabled_after_expiry ?? false,
    tags: overrides.tags ?? [],
    segment_version_id: overrides.segment_version_id ?? null,
    index_version: overrides.index_version ?? null,
    created_by: overrides.created_by ?? null,
    updated_by: overrides.updated_by ?? null,
    updated_at: overrides.updated_at ?? 1782705000,
    error: overrides.error ?? null,
    governance_status: overrides.governance_status ?? 'managed',
    legacy_origin: overrides.legacy_origin ?? null,
  };
}

export function createKnowledgeSourceDocumentFixture(
  overrides: Partial<SemanticModelSourceDocument> = {},
): SemanticModelSourceDocument {
  const source = createKnowledgeSourceFixture({
    segment_version_id: 'segment-v1',
    index_version: 1,
    ingest_status: 'ready',
    enabled: true,
    ...overrides.source,
  });
  const segments = clone(
    overrides.segments ?? [
      createKnowledgeSegmentFixture({
        metadata: { volume_id: source.processed_volume_id },
      }),
    ],
  );
  return {
    source,
    preview: overrides.preview ?? {
      available: true,
    },
    file_info: overrides.file_info ?? {
      tags: source.tags ?? [],
      expires_at: source.expires_at,
      enabled: source.enabled,
      expired: source.expired ?? false,
      effective_enabled: source.effective_enabled ?? true,
      force_enabled_after_expiry: source.force_enabled_after_expiry ?? false,
      index_version: source.index_version ?? null,
      segment_version_id: source.segment_version_id,
    },
    segment_status: overrides.segment_status ?? {
      available: true,
      total: segments.length,
    },
    current_segment_version_id: overrides.current_segment_version_id ?? source.segment_version_id,
    current_index_version: overrides.current_index_version ?? source.index_version,
    selected_segment_version_id: overrides.selected_segment_version_id ?? source.segment_version_id,
    selected_index_version: overrides.selected_index_version ?? source.index_version,
    segment_versions: overrides.segment_versions ?? [
      {
        version_id: source.segment_version_id ?? 'segment-v1',
        current: true,
        index_version: source.index_version ?? 1,
        status: 'committed',
        source: 'initial_import',
        chunk_count: segments.length,
        enabled_chunk_count: segments.filter((segment) => segment.enabled).length,
      },
    ],
    segments,
  };
}

export function createKnowledgeSegmentFixture(
  overrides: Partial<SemanticModelDocumentSegment> = {},
): SemanticModelDocumentSegment {
  return {
    segment_id: overrides.segment_id ?? 'segment-1',
    segment_type: overrides.segment_type ?? 'text',
    start_ms: overrides.start_ms,
    end_ms: overrides.end_ms,
    level: overrides.level ?? 'chunk',
    chunk_index: overrides.chunk_index ?? 0,
    chunk_id: overrides.chunk_id ?? null,
    content: overrides.content ?? 'Mock segment content',
    ocr_text: overrides.ocr_text ?? '',
    image_description: overrides.image_description ?? '',
    image_file_id: overrides.image_file_id ?? null,
    page_image_file_id: overrides.page_image_file_id ?? null,
    word_count: overrides.word_count ?? 20,
    recall_count: overrides.recall_count ?? 0,
    enabled: overrides.enabled ?? true,
    metadata: overrides.metadata,
  };
}

export function createKnowledgeCreateResponseFixture(
  overrides: Partial<CreateSemanticModelWithSourcesResponse> = {},
): CreateSemanticModelWithSourcesResponse {
  const model = overrides.model ?? createKnowledgeModelFixture();
  return {
    model,
    data_domain: overrides.data_domain ?? createKnowledgeDataDomainFixture({ model_id: model.id }),
    sources: clone(overrides.sources ?? [createKnowledgeSourceFixture({ model_id: model.id })]),
    jobs: clone(overrides.jobs ?? [createKnowledgeSourceJobFixture({ model_id: model.id })]),
  };
}

export function createKnowledgeAppendSourcesResponseFixture(
  overrides: Partial<AppendSemanticModelSourcesResponse> = {},
): AppendSemanticModelSourcesResponse {
  const modelId = overrides.data_domain?.model_id ?? overrides.sources?.[0]?.model_id ?? 9001;
  return {
    data_domain: overrides.data_domain ?? createKnowledgeDataDomainFixture({ model_id: modelId }),
    sources: clone(
      overrides.sources ?? [
        createKnowledgeSourceFixture({
          row_id: `${modelId}:file:kb-append-file-1`,
          model_id: modelId,
          resource_id: 'kb-append-file-1',
          display_name: 'append.txt',
        }),
      ],
    ),
    jobs: clone(
      overrides.jobs ?? [
        createKnowledgeSourceJobFixture({
          job_id: 'job-append-1',
          source_id: `${modelId}:file:kb-append-file-1`,
          kb_file_id: 'kb-append-file-1',
        }),
      ],
    ),
  };
}
