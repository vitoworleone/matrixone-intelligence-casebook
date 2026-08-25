import type { AppHttpRequestConfig } from '@moi/shared-moi-app-protocol/app-context';

// ─── Enums ────────────────────────────────────────────────────────────────────

/** Semantic Entry kind (replaces Nl2SqlKnowledgeType) */
export type SemanticKind =
  | 'dimension'
  | 'fact'
  | 'metric'
  | 'relationship'
  | 'column_preference'
  | 'named_filter'
  | 'verified_query'
  | 'glossary'
  | 'logic_text'
  | 'sql_resultset';

/** logic_text injection stages */
export type InjectionStage =
  | 'planner_policy'
  | 'sql_generation'
  | 'sql_followup'
  | 'sql_regenerate'
  | 'sql_decomposition'
  | 'executor_rule'
  | 'renderer_rule';

// ─── Semantic Model ───────────────────────────────────────────────────────────

export interface SemanticModelTable {
  db_name: string;
  table_names: string[];
  parents?: string[];
}

export interface SemanticModelFiles {
  file_ids: string[];
  tags?: string[];
  parents?: string[];
  volume_ids?: string[];
  vector_table?: string;
  embedding_model?: string;
  image_vector_table?: string;
  image_embedding_model?: string;
  image_embedding_dimension?: number;
  image_embedding_backend_id?: string;
  image_preprocess_version?: string;
  image_distance_metric?: string;
  image_index_configs?: SemanticModelImageIndexConfig[];
  active_image_index_config_id?: string;
  image_index_status?: string;
  image_index_file_statuses?: SemanticModelImageIndexFileStatus[];
  volumes?: Array<{
    volume_id: string;
    parents?: string[];
    path?: string[];
  }>;
}

export interface SemanticModelSourceCounts {
  files: number;
  tables: number;
  total: number;
}

export interface SemanticModelImageIndexConfig {
  id?: string;
  name?: string;
  image_vector_table?: string;
  image_embedding_model?: string;
  image_embedding_dimension?: number;
  image_embedding_backend_id?: string;
  image_preprocess_version?: string;
  image_distance_metric?: string;
  image_scope?: string;
  status?: string;
}

export interface SemanticModelImageIndexFileStatus {
  file_id?: string;
  config_id?: string;
  status?: string;
  error_code?: string;
  error_message?: string;
  indexed_images?: number;
}

export interface SemanticModel {
  id: number;
  name: string;
  description: string;
  tables: SemanticModelTable[];
  files?: SemanticModelFiles;
  source_counts: SemanticModelSourceCounts;
  table_set_hash: string;
  created_at: number;
  updated_at: number;
}

export type SemanticModelSourceType = 'file' | 'volume' | 'table';
export type SemanticModelSourceIngestStatus = string | null;
export type SemanticModelSourceGovernanceStatus = 'managed' | 'legacy_unbound';
export type SemanticModelSourceLegacyOrigin = 'semantic_model_explicit' | 'lineage_register';

export interface SemanticModelSource {
  row_id: string;
  source_id?: string;
  source_type: SemanticModelSourceType;
  model_id: number;
  resource_id: string;
  source_resource_id?: string | null;
  kb_resource_id?: string | null;
  source_file_id?: string | null;
  kb_file_id?: string | null;
  source_table_id?: number | null;
  kb_table_id?: number | null;
  display_name: string | null;
  path: string[];
  source_path?: string | null;
  db_name: string | null;
  table_name: string | null;
  size_bytes?: number | null;
  row_count?: number | null;
  ingest_status: SemanticModelSourceIngestStatus;
  enabled: boolean | null;
  expires_at: number | null;
  expired?: boolean;
  effective_enabled?: boolean;
  force_enabled_after_expiry?: boolean;
  tags?: string[];
  processed_volume_id?: number | null;
  segment_version_id: string | null;
  index_version?: number | null;
  created_by?: string | null;
  updated_by?: string | null;
  updated_at?: number | null;
  error: string | null;
  governance_status?: SemanticModelSourceGovernanceStatus;
  legacy_origin?: SemanticModelSourceLegacyOrigin | null;
}

export interface SemanticModelSourcePreview {
  available: boolean;
  reason?: string | null;
}

export interface SemanticModelSourceFileInfo {
  tags: string[];
  expires_at: number | null;
  enabled: boolean | null;
  expired: boolean;
  effective_enabled: boolean;
  force_enabled_after_expiry: boolean;
  index_version: number | null;
  segment_version_id: string | null;
}

export interface SemanticModelSegmentStatus {
  available: boolean;
  reason?: string | null;
  total: number;
}

export interface SemanticModelSegmentVersion {
  version_id: string;
  current: boolean;
  index_version?: number | null;
  base_version_id?: string | null;
  base_index_version?: number | null;
  status?: string;
  source?: string;
  chunk_count?: number;
  enabled_chunk_count?: number;
  created_by?: string | null;
  updated_by?: string | null;
  created_at?: number | null;
  updated_at?: number | null;
}

export interface SemanticModelSegment {
  segment_id: string;
  version_id: string;
  model_id: number;
  source_id: string;
  kb_file_id: string;
  index_version: number;
  level: string;
  chunk_index?: number | null;
  chunk_id?: string | null;
  content?: string | null;
  ocr_text?: string | null;
  image_description?: string | null;
  image_file_id?: string | null;
  page_image_file_id?: string | null;
  bbox?: unknown;
  word_count: number;
  recall_count: number;
  enabled: boolean;
  metadata?: unknown;
  created_at?: number | null;
  updated_at?: number | null;
}

export interface SemanticModelDocumentSegment {
  segment_id: string;
  segment_type?: 'text' | 'image' | 'table' | 'transcript' | string;
  start_ms?: number | null;
  end_ms?: number | null;
  level: string;
  chunk_index?: number | null;
  chunk_id?: string | null;
  content?: string | null;
  ocr_text?: string | null;
  image_description?: string | null;
  image_file_id?: string | null;
  page_image_file_id?: string | null;
  word_count: number;
  recall_count: number;
  enabled: boolean;
  metadata?: unknown;
}

export interface SemanticModelSourceDocument {
  source: SemanticModelSource;
  preview: SemanticModelSourcePreview;
  file_info: SemanticModelSourceFileInfo;
  segment_status: SemanticModelSegmentStatus;
  current_segment_version_id?: string | null;
  current_index_version?: number | null;
  selected_segment_version_id?: string | null;
  selected_index_version?: number | null;
  segment_versions: SemanticModelSegmentVersion[];
  segments: SemanticModelDocumentSegment[];
}

export interface GetSemanticModelSourceDocumentRequest {
  segment_version_id?: string;
}

export interface SemanticModelSegmentMutationBase {
  base_segment_version_id: string;
  base_index_version: number;
}

export type ImportInitialSemanticModelSegmentsRequest = SemanticModelSegmentMutationBase;

export interface UpdateSemanticModelSegmentRequest extends SemanticModelSegmentMutationBase {
  content?: string | null;
  ocr_text?: string | null;
  image_description?: string | null;
}

export interface CreateSemanticModelSegmentRequest extends SemanticModelSegmentMutationBase {
  level?: string;
  content?: string | null;
  ocr_text?: string | null;
  image_description?: string | null;
  bbox?: unknown;
  metadata?: unknown;
}

export interface UpdateSemanticModelSegmentEnabledRequest extends SemanticModelSegmentMutationBase {
  enabled: boolean;
}

export type DeleteSemanticModelSegmentRequest = SemanticModelSegmentMutationBase;

export type ReembedSemanticModelSegmentsRequest = SemanticModelSegmentMutationBase;

export type SetCurrentSemanticModelSegmentVersionRequest = SemanticModelSegmentMutationBase;

export interface SemanticModelSegmentMutationResult {
  document: SemanticModelSourceDocument;
}

export interface UpdateSemanticModelSourceGovernanceRequest {
  tags?: string[];
  expires_at?: number | null;
  enabled?: boolean;
  force_enabled_after_expiry?: boolean;
}

export interface UpdateSemanticModelSourceGovernanceResponse {
  source: SemanticModelSource;
}

export interface KnowledgeBaseDataDomain {
  model_id: number;
  catalog_id: number;
  database_id: number;
  raw_volume_id: number;
  processed_volume_id: number;
  ensure_status: string;
  last_ensure_error: string | null;
  last_checked_at: number;
}

export interface KnowledgeBaseSourceJob {
  id?: number;
  model_id: number;
  source_type: string;
  source_file_id: string | null;
  kb_file_id: string | null;
  display_name?: string;
  raw_volume_id: number;
  job_status: string;
  error: string | null;
  segment_version_id: string | null;
  index_version: number | null;
  workflow_execution_id: string | null;
}

export type SemanticModelCreateSourceType = 'local_file' | 'catalog_file' | 'catalog_table';

export interface SemanticModelCreateSource {
  source_type: SemanticModelCreateSourceType;
  file_name?: string;
  upload_kind?: 'unstructured' | 'structured';
  table_config?: string;
  file_id?: string;
  /** Required for catalog_file sources (authoritative volume location). */
  volume_id?: number;
  table_id?: number;
}

export interface SemanticModelSourceSelectionFilters {
  table_name?: string;
  file_name?: string;
  file_ext?: string[];
}

export interface SemanticModelDatabaseTableSourceSelection {
  kind: 'database_tables';
  database_id: number;
  all_selected: boolean;
  selected_table_ids?: number[];
  excluded_table_ids?: number[];
  filters?: Pick<SemanticModelSourceSelectionFilters, 'table_name'>;
}

export interface SemanticModelVolumeFileSourceSelection {
  kind: 'volume_files';
  volume_id: number;
  all_selected: boolean;
  selected_file_ids?: string[];
  excluded_file_ids?: string[];
  filters?: Pick<SemanticModelSourceSelectionFilters, 'file_name' | 'file_ext'>;
}

export type SemanticModelSourceSelection = SemanticModelDatabaseTableSourceSelection | SemanticModelVolumeFileSourceSelection;

export interface SemanticModelSourceSubmitPayload {
  sources: SemanticModelCreateSource[];
  source_selections?: SemanticModelSourceSelection[];
}

export interface CreateEmptySemanticModelRequest {
  name: string;
  description?: string;
  image_index_enabled?: boolean;
}

export interface CreateEmptySemanticModelResponse {
  model: SemanticModel;
  data_domain: KnowledgeBaseDataDomain;
}

export interface KnowledgeBaseSourceJobRun {
  job_id: string;
  source_id: string;
  job_status: string;
  source_file_id?: string | null;
  kb_file_id?: string | null;
  source_table_id?: number | null;
  kb_table_id?: number | null;
  error?: string | null;
  updated_at?: number;
  reconcile_required?: boolean;
}

// ─── Semantic Model CRUD types ────────────────────────────────────────────────

export interface SemanticModelUpsertRequest {
  name: string;
  description?: string;
  tables: SemanticModelTable[];
  files?: SemanticModelFiles;
}

export interface SemanticModelUpdateRequest {
  name: string;
  description?: string;
  tables?: SemanticModelTable[];
  files?: SemanticModelFiles;
}

export interface SemanticModelListRequest {
  page_size: number;
  page_token?: string;
  search?: string;
  tags?: string[];
}

export interface SemanticModelListResponse {
  items: SemanticModel[];
  total: number;
  next_page_token: string;
}

export interface SemanticModelTagStat {
  tag: string;
  count: number;
}

export interface SemanticModelTagListResponse {
  items: SemanticModelTagStat[];
}

export interface SemanticModelSourceListRequest {
  page?: number;
  page_size?: number;
}

export interface SemanticModelSourceListResponse {
  items: SemanticModelSource[];
  total: number;
  page?: number;
  page_size?: number;
  legacy_backfill_required?: boolean;
}

export interface CheckSemanticModelSourceExistenceRequest {
  file_ids?: string[];
  table_ids?: number[];
}

export interface CheckSemanticModelSourceExistenceResponse {
  file_ids: string[];
  table_ids: number[];
}

export interface CreateSemanticModelWithSourcesRequest {
  name: string;
  description?: string;
  files?: SemanticModelFiles;
  image_index_enabled?: boolean;
  sources: SemanticModelCreateSource[];
  source_selections?: SemanticModelSourceSelection[];
}

export interface CreateSemanticModelWithSourcesResponse {
  model: SemanticModel;
  data_domain: KnowledgeBaseDataDomain;
  sources: SemanticModelSource[];
  jobs: KnowledgeBaseSourceJobRun[];
}

export interface AppendSemanticModelSourcesRequest {
  sources: SemanticModelCreateSource[];
  source_selections?: SemanticModelSourceSelection[];
}

export interface AppendSemanticModelSourcesResponse {
  data_domain: KnowledgeBaseDataDomain;
  sources: SemanticModelSource[];
  jobs: KnowledgeBaseSourceJobRun[];
}

/** POST /semantic-models[/ :id]/local-files/upload */
export interface UploadSemanticModelLocalFileResponse {
  file_id: string;
}

export interface PreviewSemanticModelSourceSelectionsRequest {
  source_selections: SemanticModelSourceSelection[];
}

export interface PreviewSemanticModelSourceSelectionsResponse {
  file_count: number;
  table_count: number;
  total_count: number;
}

export interface SemanticModelSourceJobListResponse {
  items: KnowledgeBaseSourceJobRun[];
  total: number;
  reconcile_required: boolean;
}

export type ReconcileSemanticModelSourceJobsResponse = SemanticModelMutationResponse;

export type BackfillLegacySemanticModelSourcesResponse = SemanticModelMutationResponse;

export interface SemanticModelMutationResponse {
  updated?: boolean;
  deleted?: boolean;
}

export type SemanticModelCreateResponse = SemanticModel;

// ─── Spec structures (per kind) ──────────────────────────────────────────────

export interface DimensionSpec {
  column: string;
  data_type?: string;
  synonyms?: string[];
  is_enum?: boolean;
  sample_values?: string[];
  is_time?: boolean;
  deprecated?: boolean;
  description?: string;
}

export interface FactSpec {
  column: string;
  data_type?: string;
  description?: string;
  private?: boolean;
}

export interface MetricSpec {
  expr: string;
  synonyms?: string[];
  unit?: string;
  description?: string;
  requires_tables?: string[];
  requires_join?: string;
  semantic_pattern?: 'ratio' | 'semi_anti_join' | 'window';
}

export interface JoinColumnPair {
  left: string;
  right: string;
}

export interface RelationshipSpec {
  left_table: string;
  right_table: string;
  join_columns: JoinColumnPair[];
  description?: string;
  semantic_match?: boolean;
}

export interface ColumnPreferenceSpec {
  preferred: string;
  deprecated: string;
  reason?: string;
}

export interface NamedFilterSpec {
  expr: string;
  synonyms?: string[];
  description?: string;
  applies_to?: string[];
}

export interface VerifiedQuerySpec {
  question: string;
  sql: string;
  verified_by?: string;
  tags?: string[];
}

export interface GlossarySpec {
  term: string;
  definition: string;
  synonyms?: string[];
  related_metrics?: string[];
  formula_hint?: string;
}

export interface LogicTextSpec {
  content: string;
  injection_stages: InjectionStage[];
  priority?: number;
}

export interface SQLResultsetExpandSQLSpec {
  sql: string;
  params?: string[];
}

export type SQLResultsetResolveMode = 'semantic' | 'passthrough';

export interface SQLResultsetRetrievalSpec {
  enabled?: boolean;
  embedding_model?: string;
}

export interface SQLResultsetSpec {
  sql: string;
  description: string;
  resolve_mode?: SQLResultsetResolveMode;
  expand_sql?: SQLResultsetExpandSQLSpec;
  retrieval?: SQLResultsetRetrievalSpec;
  max_rows?: number;
  max_bytes?: number;
  timeout_seconds?: number;
}

export type SemanticEntrySpec =
  | DimensionSpec
  | FactSpec
  | MetricSpec
  | RelationshipSpec
  | ColumnPreferenceSpec
  | NamedFilterSpec
  | VerifiedQuerySpec
  | GlossarySpec
  | LogicTextSpec
  | SQLResultsetSpec;

// ─── Semantic Entry ───────────────────────────────────────────────────────────

export interface SemanticEntry {
  id: number;
  kind: SemanticKind;
  key: string;
  tables?: string[];
  spec: SemanticEntrySpec;
  created_at: number;
  updated_at: number;
}

// ─── Request types ────────────────────────────────────────────────────────────

export interface CreateSemanticEntryRequest {
  kind: SemanticKind;
  key: string;
  tables?: string[];
  spec: SemanticEntrySpec;
}

export type UpdateSemanticEntryRequest = CreateSemanticEntryRequest;

export interface ImportSemanticModelRequest {
  entries: CreateSemanticEntryRequest[];
}

export interface SemanticEntryListParams {
  kind?: SemanticKind;
  page_size?: number;
  page_token?: string;
}

// ─── Response types ───────────────────────────────────────────────────────────

export interface SemanticPagedResponse<T> {
  items: T[];
  total: number;
  next_page_token: string;
}

export type SemanticEntryListResponse = SemanticPagedResponse<SemanticEntry>;

export interface SemanticMutationResponse {
  updated?: boolean;
  deleted?: boolean;
}

export interface ImportSemanticModelResponse {
  imported: number;
  model_id: number;
}

export interface ExportSemanticModelResponse {
  model: Pick<SemanticModel, 'name' | 'tables'> & { files?: SemanticModelFiles | null };
  entries: Array<Omit<SemanticEntry, 'id' | 'created_at' | 'updated_at'>>;
}

export interface SemanticModelFilePreviewOptions extends AppHttpRequestConfig {
  requestId?: string | symbol;
  responseType?: 'blob';
  responseContentType?: 'blob';
}

export interface SemanticModelFilePreviewResponse {
  headers: Record<string, string>;
  data: Blob;
}

export type SemanticModelArtifactPreviewOptions = SemanticModelFilePreviewOptions;
export type SemanticModelArtifactPreviewResponse = SemanticModelFilePreviewResponse;

export interface ValidateSemanticModelResponse {
  valid: boolean;
  errors?: string[];
}
