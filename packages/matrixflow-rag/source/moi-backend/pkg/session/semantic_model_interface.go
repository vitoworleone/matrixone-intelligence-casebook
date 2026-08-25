package session

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/model"
)

// ========== Semantic Model Types ==========

// SemanticKind enumerates the types of semantic entries.
type SemanticKind = string

const (
	KindDimension         SemanticKind = "dimension"
	KindFact              SemanticKind = "fact"
	KindMetric            SemanticKind = "metric"
	KindRelationship      SemanticKind = "relationship"
	KindColumnPreference  SemanticKind = "column_preference"
	KindNamedFilter       SemanticKind = "named_filter"
	KindDefaultConstraint SemanticKind = "default_constraint"
	KindVerifiedQuery     SemanticKind = "verified_query"
	KindGlossary          SemanticKind = "glossary"
	KindLogicText         SemanticKind = "logic_text"
	KindSQLResultset      SemanticKind = "sql_resultset"
)

// ValidSemanticKinds is the set of all valid SemanticKind values.
var ValidSemanticKinds = map[string]bool{
	KindDimension:         true,
	KindFact:              true,
	KindMetric:            true,
	KindRelationship:      true,
	KindColumnPreference:  true,
	KindNamedFilter:       true,
	KindDefaultConstraint: true,
	KindVerifiedQuery:     true,
	KindGlossary:          true,
	KindLogicText:         true,
	KindSQLResultset:      true,
}

// InjectionStage enumerates the stages where logic_text can be injected.
type InjectionStage = string

const (
	StagePlannerPolicy    InjectionStage = "planner_policy"
	StageSQLGeneration    InjectionStage = "sql_generation"
	StageSQLFollowup      InjectionStage = "sql_followup"
	StageSQLRegenerate    InjectionStage = "sql_regenerate"
	StageSQLDecomposition InjectionStage = "sql_decomposition"
	StageExecutorRule     InjectionStage = "executor_rule"
	StageRendererRule     InjectionStage = "renderer_rule"
)

// ValidInjectionStages is the set of all valid InjectionStage values.
var ValidInjectionStages = map[string]bool{
	StagePlannerPolicy:    true,
	StageSQLGeneration:    true,
	StageSQLFollowup:      true,
	StageSQLRegenerate:    true,
	StageSQLDecomposition: true,
	StageExecutorRule:     true,
	StageRendererRule:     true,
}

// ========== Semantic Model ==========

type SemanticModelInfo struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Tables      json.RawMessage `json:"tables"`
	Files       json.RawMessage `json:"files,omitempty"`
	// SourceCounts is enriched by list/detail paths. Create responses return the
	// model snapshot plus explicit Sources/Jobs; callers that need authoritative
	// counts after creation should reload list/detail.
	SourceCounts SemanticModelSourceCounts `json:"source_counts"`
	CreatedAt    int64                     `json:"created_at"`
	UpdatedAt    int64                     `json:"updated_at"`
}

type SemanticModelSourceCounts struct {
	Files  int64 `json:"files"`
	Tables int64 `json:"tables"`
	Total  int64 `json:"total"`
}

type SemanticModelSourceType string

const (
	SemanticModelSourceTypeFile   SemanticModelSourceType = "file"
	SemanticModelSourceTypeVolume SemanticModelSourceType = "volume"
	SemanticModelSourceTypeTable  SemanticModelSourceType = "table"
)

const (
	SemanticModelSourceGovernanceManaged       = "managed"
	SemanticModelSourceGovernanceLegacyUnbound = "legacy_unbound"
	SemanticModelSourceLegacyOriginExplicit    = "semantic_model_explicit"
	SemanticModelSourceLegacyOriginLineage     = "lineage_register"
)

type SemanticModelSource struct {
	RowID            string                  `json:"row_id"`
	SourceID         string                  `json:"source_id,omitempty"`
	SourceType       SemanticModelSourceType `json:"source_type"`
	ModelID          int64                   `json:"model_id"`
	ResourceID       string                  `json:"resource_id"`
	SourceResourceID *string                 `json:"source_resource_id,omitempty"`
	KBResourceID     *string                 `json:"kb_resource_id,omitempty"`
	SourceFileID     *string                 `json:"source_file_id,omitempty"`
	KBFileID         *string                 `json:"kb_file_id,omitempty"`
	SourceTableID    *int64                  `json:"source_table_id,omitempty"`
	KBTableID        *int64                  `json:"kb_table_id,omitempty"`
	DisplayName      *string                 `json:"display_name"`
	Path             []string                `json:"path"`
	SourcePath       *string                 `json:"source_path,omitempty"`
	DBName           *string                 `json:"db_name"`
	TableName        *string                 `json:"table_name"`
	SizeBytes        *int64                  `json:"size_bytes"`
	RowCount         *int64                  `json:"row_count"`
	IngestStatus     *string                 `json:"ingest_status"`
	Enabled          *bool                   `json:"enabled"`
	ExpiresAt        *int64                  `json:"expires_at"`
	Expired          bool                    `json:"expired"`
	EffectiveEnabled bool                    `json:"effective_enabled"`
	ForceEnabled     bool                    `json:"force_enabled_after_expiry"`
	Tags             []string                `json:"tags,omitempty"`
	SegmentVersionID *string                 `json:"segment_version_id"`
	IndexVersion     *int64                  `json:"index_version"`
	CreatedBy        *string                 `json:"created_by,omitempty"`
	UpdatedBy        *string                 `json:"updated_by,omitempty"`
	UpdatedAt        *int64                  `json:"updated_at"`
	Error            *string                 `json:"error"`
	GovernanceStatus string                  `json:"governance_status,omitempty"`
	LegacyOrigin     *string                 `json:"legacy_origin,omitempty"`
}

type KnowledgeBaseSourceRecord struct {
	SourceID          string  `json:"source_id"`
	ModelID           int64   `json:"model_id"`
	CatalogID         int64   `json:"catalog_id"`
	DatabaseID        int64   `json:"database_id"`
	RawVolumeID       int64   `json:"raw_volume_id"`
	ProcessedVolumeID int64   `json:"processed_volume_id"`
	SourceType        string  `json:"source_type"`
	SourceFileID      *string `json:"source_file_id"`
	SourceTableID     *int64  `json:"source_table_id"`
	KBFileID          *string `json:"kb_file_id"`
	KBTableID         *int64  `json:"kb_table_id"`
	DisplayName       *string `json:"display_name"`
	SourcePath        *string `json:"source_path"`
	DBName            *string `json:"db_name"`
	TableName         *string `json:"table_name"`
	Status            string  `json:"status"`
	Error             *string `json:"error"`
	Enabled           *bool   `json:"enabled"`
	ExpiresAt         *int64  `json:"expires_at"`
	Tags              *string `json:"tags"`
	ForceEnabled      bool    `json:"force_enabled_after_expiry"`
	SegmentVersionID  *string `json:"segment_version_id"`
	IndexVersion      *int64  `json:"index_version"`
	SizeBytes         *int64  `json:"size_bytes"`
	RowCount          *int64  `json:"row_count"`
	CreatedBy         *string `json:"created_by"`
	UpdatedBy         *string `json:"updated_by"`
	UpdatedAt         *int64  `json:"updated_at"`
}

type KnowledgeBaseDataDomain struct {
	ModelID           int64   `json:"model_id"`
	CatalogID         int64   `json:"catalog_id"`
	DatabaseID        int64   `json:"database_id"`
	RawVolumeID       int64   `json:"raw_volume_id"`
	ProcessedVolumeID int64   `json:"processed_volume_id"`
	EnsureStatus      string  `json:"ensure_status"`
	LastEnsureError   *string `json:"last_ensure_error"`
	LastCheckedAt     int64   `json:"last_checked_at"`
}

type KnowledgeBaseSourceJob struct {
	ID                  int64   `json:"id,omitempty"`
	ModelID             int64   `json:"model_id"`
	SourceType          string  `json:"source_type"`
	SourceFileID        *string `json:"source_file_id"`
	KBFileID            *string `json:"kb_file_id"`
	DisplayName         string  `json:"display_name,omitempty"`
	RawVolumeID         int64   `json:"raw_volume_id"`
	JobStatus           string  `json:"job_status"`
	Error               *string `json:"error"`
	SegmentVersionID    *string `json:"segment_version_id"`
	IndexVersion        *int64  `json:"index_version"`
	WorkflowExecutionID *string `json:"workflow_execution_id"`
}

type KnowledgeBaseSourceJobRun struct {
	JobID                   string  `json:"job_id"`
	SourceID                string  `json:"source_id"`
	ModelID                 int64   `json:"model_id"`
	JobType                 string  `json:"job_type"`
	JobStatus               string  `json:"job_status"`
	IdempotencyKey          string  `json:"idempotency_key"`
	OperationID             *string `json:"operation_id"`
	WorkflowExecutionID     *string `json:"workflow_execution_id"`
	RuntimeActorMOIUserID   *string `json:"-"`
	RuntimeEffectiveRoleID  *string `json:"-"`
	RuntimeIsWorkspaceOwner bool    `json:"-"`
	SourceFileID            *string `json:"source_file_id"`
	KBFileID                *string `json:"kb_file_id"`
	SourceTableID           *int64  `json:"source_table_id"`
	KBTableID               *int64  `json:"kb_table_id"`
	RetryCount              int64   `json:"retry_count"`
	NextRetryAt             *int64  `json:"next_retry_at"`
	Error                   *string `json:"error"`
	CreatedAt               int64   `json:"created_at,omitempty"`
	UpdatedAt               int64   `json:"updated_at,omitempty"`
}

type KnowledgeBaseSourceJobView struct {
	JobID             string  `json:"job_id"`
	SourceID          string  `json:"source_id"`
	JobStatus         string  `json:"job_status"`
	SourceFileID      *string `json:"source_file_id,omitempty"`
	KBFileID          *string `json:"kb_file_id,omitempty"`
	SourceTableID     *int64  `json:"source_table_id,omitempty"`
	KBTableID         *int64  `json:"kb_table_id,omitempty"`
	Error             *string `json:"error,omitempty"`
	UpdatedAt         *int64  `json:"updated_at,omitempty"`
	ReconcileRequired bool    `json:"reconcile_required,omitempty"`
}

type KnowledgeBaseWorkflowDeployRequest struct {
	WorkflowID          string
	Name                string
	Description         string
	DSLYAML             string
	InputFormJSON       string
	DefaultValues       map[string]any
	RawVolumeID         int64
	TriggerEnabled      bool
	AutoDispatchEnabled bool
	ExecutionMode       string
}

type KnowledgeBaseWorkflowRunResult struct {
	ExecutionID string
}

// ========== Semantic Entry ==========

type SemanticEntry struct {
	ID        int64           `json:"id"`
	Kind      string          `json:"kind"`
	Key       string          `json:"key"`
	Tables    []string        `json:"tables,omitempty"`
	Spec      json.RawMessage `json:"spec"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
}

// ========== Request / Response Types ==========

type CreateSemanticModelRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Tables      json.RawMessage `json:"tables"`
	Files       json.RawMessage `json:"files,omitempty"`
}

type CreateSemanticModelLocalFileSource struct {
	FileName string `json:"file_name"`
	FileID   string `json:"file_id"`
}

type CreateSemanticModelCatalogFileSource struct {
	FileID string `json:"file_id"`
}

type CreateSemanticModelCatalogTableSource struct {
	TableID int64 `json:"table_id"`
}

type CreateSemanticModelSourceRequest struct {
	SourceType  string `json:"source_type"`
	FileName    string `json:"file_name,omitempty"`
	UploadKind  string `json:"upload_kind,omitempty"`
	TableConfig string `json:"table_config,omitempty"`
	FileID      string `json:"file_id,omitempty"`
	// VolumeID is required for catalog_file write paths: write-time permission
	// gate and location pointer (stored as source raw_volume_id). Not part of
	// business identity (source_id is file-scoped). Selections carry the verified
	// volume; missing volume_id fails instead of guessing from volume_files.
	VolumeID int64 `json:"volume_id,omitempty"`
	TableID  int64 `json:"table_id,omitempty"`

	// DeprecatedContentBase64 is decoded only so legacy requests can be rejected.
	DeprecatedContentBase64 json.RawMessage `json:"content_base64,omitempty" swaggerignore:"true"`
}

type SemanticModelSourceSelectionFilters struct {
	TableName string   `json:"table_name,omitempty"`
	FileName  string   `json:"file_name,omitempty"`
	FileExt   []string `json:"file_ext,omitempty"`
}

type SemanticModelSourceSelectionRequest struct {
	Kind             string                              `json:"kind"`
	DatabaseID       int64                               `json:"database_id,omitempty"`
	VolumeID         int64                               `json:"volume_id,omitempty"`
	AllSelected      bool                                `json:"all_selected"`
	SelectedTableIDs []int64                             `json:"selected_table_ids,omitempty"`
	ExcludedTableIDs []int64                             `json:"excluded_table_ids,omitempty"`
	SelectedFileIDs  []string                            `json:"selected_file_ids,omitempty"`
	ExcludedFileIDs  []string                            `json:"excluded_file_ids,omitempty"`
	Filters          SemanticModelSourceSelectionFilters `json:"filters,omitempty"`
}

type CreateSemanticModelWithSourcesRequest struct {
	Name              string                                `json:"name"`
	Description       string                                `json:"description,omitempty"`
	Files             json.RawMessage                       `json:"files,omitempty"`
	ImageIndexEnabled bool                                  `json:"image_index_enabled"`
	Sources           []CreateSemanticModelSourceRequest    `json:"sources"`
	SourceSelections  []SemanticModelSourceSelectionRequest `json:"source_selections,omitempty"`
	SelectionSources  []CreateSemanticModelSourceRequest    `json:"-"`
}

type CreateSemanticModelWithSourcesResponse struct {
	// Model is the created semantic model snapshot. Source counts are not the
	// authoritative source list in this response; use Sources/Jobs or reload
	// list/detail when counts are needed.
	Model      *SemanticModelInfo          `json:"model"`
	DataDomain *KnowledgeBaseDataDomain    `json:"data_domain"`
	Sources    []SemanticModelSource       `json:"sources"`
	Jobs       []KnowledgeBaseSourceJobRun `json:"jobs"`
}

// CreateEmptySemanticModelRequest initializes a data-side knowledge base without sources.
type CreateEmptySemanticModelRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description,omitempty"`
	ImageIndexEnabled bool   `json:"image_index_enabled"`
}

type CreateEmptySemanticModelResponse struct {
	Model      *SemanticModelInfo       `json:"model"`
	DataDomain *KnowledgeBaseDataDomain `json:"data_domain"`
}

type AppendSemanticModelSourcesRequest struct {
	ModelID          int                                   `json:"-"`
	Sources          []CreateSemanticModelSourceRequest    `json:"sources"`
	SourceSelections []SemanticModelSourceSelectionRequest `json:"source_selections,omitempty"`
	SelectionSources []CreateSemanticModelSourceRequest    `json:"-"`
}

type AppendSemanticModelSourcesResponse struct {
	DataDomain *KnowledgeBaseDataDomain    `json:"data_domain"`
	Sources    []SemanticModelSource       `json:"sources"`
	Jobs       []KnowledgeBaseSourceJobRun `json:"jobs"`
}

type PreviewSemanticModelSourceSelectionsRequest struct {
	ModelID          int                                   `json:"-"`
	SourceSelections []SemanticModelSourceSelectionRequest `json:"source_selections" binding:"required"`
}

type PreviewSemanticModelSourceSelectionsResponse struct {
	FileCount  int `json:"file_count"`
	TableCount int `json:"table_count"`
	TotalCount int `json:"total_count"`
}

type UpdateSemanticModelRequest struct {
	ModelID     int             `json:"-"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Tables      json.RawMessage `json:"tables"`
	Files       json.RawMessage `json:"files,omitempty"`
}

type ListSemanticModelsRequest struct {
	PageSize  int      `json:"page_size"`
	PageToken string   `json:"page_token"`
	Search    string   `json:"search"`
	Tags      []string `json:"tags,omitempty"`
}

type ListSemanticModelsResponse struct {
	Items         []*SemanticModelInfo `json:"items"`
	Total         int64                `json:"total"`
	NextPageToken string               `json:"next_page_token,omitempty"`
}

type SemanticModelTagStat struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

type ListSemanticModelTagsResponse struct {
	Items []SemanticModelTagStat `json:"items"`
}

type GetSemanticModelRequest struct {
	ModelID int `json:"-"`
}

type ListSemanticModelSourcesParams struct {
	ModelID  int `json:"-"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type ListSemanticModelSourcesResult struct {
	Items                  []SemanticModelSource `json:"items"`
	Total                  int                   `json:"total"`
	Page                   int                   `json:"page"`
	PageSize               int                   `json:"page_size"`
	LegacyBackfillRequired bool                  `json:"legacy_backfill_required,omitempty"`
}

type CheckSemanticModelSourceExistenceParams struct {
	ModelID  int      `json:"-"`
	FileIDs  []string `json:"file_ids"`
	TableIDs []int64  `json:"table_ids"`
}

type CheckSemanticModelSourceExistenceResult struct {
	FileIDs  []string `json:"file_ids"`
	TableIDs []int64  `json:"table_ids"`
}

type ListSemanticModelSourceJobsParams struct {
	ModelID int `json:"-"`
}

type ListSemanticModelSourceJobsResult struct {
	Items             []KnowledgeBaseSourceJobView `json:"items"`
	Total             int                          `json:"total"`
	ReconcileRequired bool                         `json:"reconcile_required"`
}

type OptionalInt64 struct {
	Set   bool
	Value *int64
}

func (v *OptionalInt64) UnmarshalJSON(data []byte) error {
	v.Set = true
	if string(data) == "null" {
		v.Value = nil
		return nil
	}
	var value int64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	v.Value = &value
	return nil
}

type GetSemanticModelSourceDocumentParams struct {
	ModelID          int    `json:"-"`
	SourceID         string `json:"-"`
	SegmentVersionID string `json:"-"`
}

type SemanticModelSourceDocument struct {
	Source                   SemanticModelSource            `json:"source"`
	Preview                  SemanticModelSourcePreview     `json:"preview"`
	FileInfo                 SemanticModelSourceFileInfo    `json:"file_info"`
	SegmentStatus            SemanticModelSegmentStatus     `json:"segment_status"`
	CurrentSegmentVersionID  *string                        `json:"current_segment_version_id,omitempty"`
	CurrentIndexVersion      *int64                         `json:"current_index_version,omitempty"`
	SelectedSegmentVersionID *string                        `json:"selected_segment_version_id,omitempty"`
	SelectedIndexVersion     *int64                         `json:"selected_index_version,omitempty"`
	SegmentVersions          []SemanticModelSegmentVersion  `json:"segment_versions"`
	Segments                 []SemanticModelDocumentSegment `json:"segments"`
}

type SemanticModelSourcePreview struct {
	Available bool    `json:"available"`
	Content   *string `json:"content,omitempty"`
	Reason    *string `json:"reason,omitempty"`
}

type SemanticModelSourceFileInfo struct {
	Tags             []string `json:"tags"`
	ExpiresAt        *int64   `json:"expires_at"`
	Enabled          *bool    `json:"enabled"`
	Expired          bool     `json:"expired"`
	EffectiveEnabled bool     `json:"effective_enabled"`
	ForceEnabled     bool     `json:"force_enabled_after_expiry"`
	IndexVersion     *int64   `json:"index_version"`
	SegmentVersionID *string  `json:"segment_version_id"`
}

type SemanticModelSegmentStatus struct {
	Available bool    `json:"available"`
	Reason    *string `json:"reason,omitempty"`
	Total     int     `json:"total"`
}

type SemanticModelSegmentVersion struct {
	VersionID         string  `json:"version_id"`
	Current           bool    `json:"current"`
	IndexVersion      *int64  `json:"index_version,omitempty"`
	BaseVersionID     *string `json:"base_version_id,omitempty"`
	BaseIndexVersion  *int64  `json:"base_index_version,omitempty"`
	Status            string  `json:"status,omitempty"`
	Source            string  `json:"source,omitempty"`
	ChunkCount        int64   `json:"chunk_count,omitempty"`
	EnabledChunkCount int64   `json:"enabled_chunk_count,omitempty"`
	CreatedBy         *string `json:"created_by,omitempty"`
	UpdatedBy         *string `json:"updated_by,omitempty"`
	CreatedAt         *int64  `json:"created_at,omitempty"`
	UpdatedAt         *int64  `json:"updated_at,omitempty"`
}

type SemanticModelDocumentSegment struct {
	SegmentID        string          `json:"segment_id"`
	SegmentType      string          `json:"segment_type"`
	StartMS          *int64          `json:"start_ms,omitempty"`
	EndMS            *int64          `json:"end_ms,omitempty"`
	Level            string          `json:"level"`
	ChunkIndex       *int64          `json:"chunk_index,omitempty"`
	ChunkID          *string         `json:"chunk_id,omitempty"`
	Content          *string         `json:"content,omitempty"`
	OCRText          *string         `json:"ocr_text,omitempty"`
	ImageDescription *string         `json:"image_description,omitempty"`
	ImageFileID      *string         `json:"image_file_id,omitempty"`
	PageImageFileID  *string         `json:"page_image_file_id,omitempty"`
	WordCount        int64           `json:"word_count"`
	RecallCount      int64           `json:"recall_count"`
	Enabled          bool            `json:"enabled"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
}

type SemanticModelSegment struct {
	SegmentID        string          `json:"segment_id"`
	VersionID        string          `json:"version_id"`
	ModelID          int64           `json:"model_id"`
	SourceID         string          `json:"source_id"`
	KBFileID         string          `json:"kb_file_id"`
	IndexVersion     int64           `json:"index_version"`
	Level            string          `json:"level"`
	ChunkIndex       *int64          `json:"chunk_index,omitempty"`
	ChunkID          *string         `json:"chunk_id,omitempty"`
	Content          *string         `json:"content,omitempty"`
	OCRText          *string         `json:"ocr_text,omitempty"`
	ImageDescription *string         `json:"image_description,omitempty"`
	ImageFileID      *string         `json:"image_file_id,omitempty"`
	PageImageFileID  *string         `json:"page_image_file_id,omitempty"`
	BBox             json.RawMessage `json:"bbox,omitempty"`
	WordCount        int64           `json:"word_count"`
	RecallCount      int64           `json:"recall_count"`
	Enabled          bool            `json:"enabled"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
	CreatedAt        *int64          `json:"created_at,omitempty"`
	UpdatedAt        *int64          `json:"updated_at,omitempty"`
}

type SemanticModelSegmentMutationBase struct {
	BaseSegmentVersionID *string `json:"base_segment_version_id"`
	BaseIndexVersion     *int64  `json:"base_index_version"`
}

type ImportInitialSemanticModelSegmentsParams struct {
	ModelID  int    `json:"-"`
	SourceID string `json:"-"`
	SemanticModelSegmentMutationBase
}

type UpdateSemanticModelSegmentParams struct {
	ModelID   int    `json:"-"`
	SourceID  string `json:"-"`
	SegmentID string `json:"-"`
	SemanticModelSegmentMutationBase
	Content          *string `json:"content,omitempty"`
	OCRText          *string `json:"ocr_text,omitempty"`
	ImageDescription *string `json:"image_description,omitempty"`
}

type CreateSemanticModelSegmentParams struct {
	ModelID  int    `json:"-"`
	SourceID string `json:"-"`
	SemanticModelSegmentMutationBase
	Level            string          `json:"level,omitempty"`
	Content          *string         `json:"content,omitempty"`
	OCRText          *string         `json:"ocr_text,omitempty"`
	ImageDescription *string         `json:"image_description,omitempty"`
	ImageFileID      *string         `json:"image_file_id,omitempty"`
	PageImageFileID  *string         `json:"page_image_file_id,omitempty"`
	BBox             json.RawMessage `json:"bbox,omitempty"`
	Metadata         json.RawMessage `json:"metadata,omitempty"`
}

type UpdateSemanticModelSegmentEnabledParams struct {
	ModelID   int    `json:"-"`
	SourceID  string `json:"-"`
	SegmentID string `json:"-"`
	SemanticModelSegmentMutationBase
	Enabled *bool `json:"enabled"`
}

type DeleteSemanticModelSegmentParams struct {
	ModelID   int    `json:"-"`
	SourceID  string `json:"-"`
	SegmentID string `json:"-"`
	SemanticModelSegmentMutationBase
}

type ReembedSemanticModelSegmentsParams struct {
	ModelID  int    `json:"-"`
	SourceID string `json:"-"`
	SemanticModelSegmentMutationBase
}

type SetCurrentSemanticModelSegmentVersionParams struct {
	ModelID   int    `json:"-"`
	SourceID  string `json:"-"`
	VersionID string `json:"-"`
	SemanticModelSegmentMutationBase
}

type SemanticModelSegmentMutationResult struct {
	Document SemanticModelSourceDocument `json:"document"`
}

type UpdateSemanticModelSourceGovernanceParams struct {
	ModelID                 int           `json:"-"`
	SourceID                string        `json:"-"`
	Tags                    *[]string     `json:"tags,omitempty"`
	ExpiresAt               OptionalInt64 `json:"expires_at,omitempty"`
	Enabled                 *bool         `json:"enabled,omitempty"`
	ForceEnabledAfterExpiry *bool         `json:"force_enabled_after_expiry,omitempty"`
}

type UpdateSemanticModelSourceGovernanceResult struct {
	Source SemanticModelSource `json:"source"`
}

type DeleteSemanticModelSourceParams struct {
	ModelID  int    `json:"-"`
	SourceID string `json:"-"`
}

type RunPendingKnowledgeBaseSourceJobsParams struct {
	ModelID int64
}

type ReconcileKnowledgeBaseSourceJobsParams struct {
	ModelID int64
}

type BackfillLegacyKnowledgeBaseSourcesParams struct {
	ModelID int64
}

type ListSemanticEntriesRequest struct {
	ModelID   int    `json:"-"`
	Kind      string `json:"kind"`
	PageSize  int    `json:"page_size"`
	PageToken string `json:"page_token"`
}

type ListSemanticEntriesResponse struct {
	Items         []SemanticEntry `json:"items"`
	Total         int             `json:"total"`
	NextPageToken string          `json:"next_page_token"`
}

type CreateSemanticEntryRequest struct {
	ModelID int             `json:"-"`
	Kind    string          `json:"kind"`
	Key     string          `json:"key"`
	Tables  []string        `json:"tables,omitempty"`
	Spec    json.RawMessage `json:"spec"`
}

type UpdateSemanticEntryRequest struct {
	ModelID int             `json:"-"`
	EntryID int             `json:"-"`
	Kind    string          `json:"kind"`
	Key     string          `json:"key"`
	Tables  []string        `json:"tables,omitempty"`
	Spec    json.RawMessage `json:"spec"`
}

type DeleteSemanticEntryRequest struct {
	ModelID int `json:"-"`
	EntryID int `json:"-"`
}

type ImportSemanticModelRequest struct {
	ModelID int                          `json:"-"`
	Entries []CreateSemanticEntryRequest `json:"entries"`
}

type ImportSemanticModelResponse struct {
	Imported int   `json:"imported"`
	ModelID  int64 `json:"model_id"`
}

type ExportSemanticModelResponse struct {
	Model   SemanticModelInfo `json:"model"`
	Entries []SemanticEntry   `json:"entries"`
}

type ValidateSemanticModelResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

type MutationResponse struct {
	Updated bool `json:"updated,omitempty"`
	Deleted bool `json:"deleted,omitempty"`
}

type SemanticModelFilePreview struct {
	Filename    string
	ContentType string
	Body        io.ReadCloser
}

// SemanticModelArtifactPreview remains the artifact-facing name for callers
// that predate source-file preview support.
type SemanticModelArtifactPreview = SemanticModelFilePreview

// ========== Service Interface ==========

// SemanticModelService manages semantic models and entries.
type SemanticModelService interface {
	// CreateModel creates a semantic-model-backed knowledge base and its base Catalog resources.
	CreateModel(ctx context.Context, params CreateSemanticModelRequest) (*SemanticModelInfo, error)

	// CreateEmptyModel initializes a data-side knowledge base without adding sources.
	CreateEmptyModel(ctx context.Context, params CreateEmptySemanticModelRequest) (*CreateEmptySemanticModelResponse, error)

	// CreateModelWithSources creates a semantic-model-backed knowledge base with initial sources.
	CreateModelWithSources(ctx context.Context, params CreateSemanticModelWithSourcesRequest) (*CreateSemanticModelWithSourcesResponse, error)

	// UploadLocalFile stores one knowledge-base local file and returns the catalog file_id.
	// Callers bind the returned file_id through create-with-sources or append sources.
	UploadLocalFile(ctx context.Context, fileName string, reader io.Reader) (string, error)

	// AppendModelSources appends sources to an existing semantic-model-backed knowledge base.
	AppendModelSources(ctx context.Context, params AppendSemanticModelSourcesRequest) (*AppendSemanticModelSourcesResponse, error)
	// PreviewSourceSelectionCounts expands and counts source selections without persisting them.
	PreviewSourceSelectionCounts(ctx context.Context, params PreviewSemanticModelSourceSelectionsRequest) (*PreviewSemanticModelSourceSelectionsResponse, error)

	// ListModels lists semantic models.
	ListModels(ctx context.Context, params ListSemanticModelsRequest) (*ListSemanticModelsResponse, error)

	// ListModelTags lists aggregated semantic model tags.
	ListModelTags(ctx context.Context, params ListSemanticModelsRequest) (*ListSemanticModelTagsResponse, error)

	// ListModelsByIDs lists semantic models by explicit authorized model IDs.
	ListModelsByIDs(ctx context.Context, ids []int64, params ListSemanticModelsRequest) (*ListSemanticModelsResponse, error)

	// GetModel returns a semantic model by ID.
	GetModel(ctx context.Context, modelID int) (*SemanticModelInfo, error)

	// PreviewArtifact streams a workflow-derived file associated with a semantic model.
	PreviewArtifact(ctx context.Context, modelID int, fileID string) (*SemanticModelFilePreview, error)
	// PreviewSourceFile streams a workflow source document associated with a semantic model.
	PreviewSourceFile(ctx context.Context, modelID int, fileID string) (*SemanticModelFilePreview, error)

	// ListSources returns read-only file, volume, and table source rows for a semantic model.
	ListSources(ctx context.Context, params ListSemanticModelSourcesParams) (*ListSemanticModelSourcesResult, error)
	// CheckSourceExistence returns which current-page Catalog files or tables already exist in a knowledge base.
	CheckSourceExistence(ctx context.Context, params CheckSemanticModelSourceExistenceParams) (*CheckSemanticModelSourceExistenceResult, error)
	// GetSourceDocument returns governance and read-only detail state for one document source.
	GetSourceDocument(ctx context.Context, params GetSemanticModelSourceDocumentParams) (*SemanticModelSourceDocument, error)
	ResolveLegacySourceIAMDependencies(ctx context.Context, tables, files json.RawMessage) ([]CreateSemanticModelSourceRequest, error)
	ResolveBackfillSourceIAMDependencies(ctx context.Context, modelID int64) ([]CreateSemanticModelSourceRequest, error)
	// ImportInitialSegments creates the initial committed chunk version from existing indexed rows.
	ImportInitialSegments(ctx context.Context, params ImportInitialSemanticModelSegmentsParams) (*SemanticModelSegmentMutationResult, error)
	// UpdateSegment edits one chunk and commits a new materialized version.
	UpdateSegment(ctx context.Context, params UpdateSemanticModelSegmentParams) (*SemanticModelSegmentMutationResult, error)
	// CreateSegment appends one chunk and commits a new materialized version.
	CreateSegment(ctx context.Context, params CreateSemanticModelSegmentParams) (*SemanticModelSegmentMutationResult, error)
	// UpdateSegmentEnabled toggles one chunk and commits a new materialized version.
	UpdateSegmentEnabled(ctx context.Context, params UpdateSemanticModelSegmentEnabledParams) (*SemanticModelSegmentMutationResult, error)
	// DeleteSegment removes one chunk from the current version and commits a new materialized version.
	DeleteSegment(ctx context.Context, params DeleteSemanticModelSegmentParams) (*SemanticModelSegmentMutationResult, error)
	// ReembedSegments rematerializes the current committed chunk version with a new index version.
	ReembedSegments(ctx context.Context, params ReembedSemanticModelSegmentsParams) (*SemanticModelSegmentMutationResult, error)
	// SetCurrentSegmentVersion points production retrieval at an existing committed version.
	SetCurrentSegmentVersion(ctx context.Context, params SetCurrentSemanticModelSegmentVersionParams) (*SemanticModelSegmentMutationResult, error)
	// UpdateSourceGovernance updates source-level governance fields.
	UpdateSourceGovernance(ctx context.Context, params UpdateSemanticModelSourceGovernanceParams) (*UpdateSemanticModelSourceGovernanceResult, error)
	// DeleteSource removes one knowledge base source relation. File sources keep their Catalog files and vectors.
	DeleteSource(ctx context.Context, params DeleteSemanticModelSourceParams) error
	// ListSourceJobs returns persisted backend job runs for a semantic model.
	ListSourceJobs(ctx context.Context, params ListSemanticModelSourceJobsParams) (*ListSemanticModelSourceJobsResult, error)
	// BackfillLegacySources fills missing source/job-run rows for workflow-linked legacy knowledge-base files.
	BackfillLegacySources(ctx context.Context, params BackfillLegacyKnowledgeBaseSourcesParams) error
	// ReconcileKnowledgeBaseSourceJobs refreshes persisted source jobs and publishes completed parse results.
	// When its context has a MOI user ID, only NULL/empty runtime actors on retryable
	// RAG ingest jobs are backfilled with that caller; existing actors are preserved.
	ReconcileKnowledgeBaseSourceJobs(ctx context.Context, params ReconcileKnowledgeBaseSourceJobsParams) error
	// RunPendingKnowledgeBaseSourceJobs reconciles persisted source jobs for a semantic model.
	RunPendingKnowledgeBaseSourceJobs(ctx context.Context, params RunPendingKnowledgeBaseSourceJobsParams) error

	// UpdateModel updates a semantic model.
	UpdateModel(ctx context.Context, params UpdateSemanticModelRequest) error

	// DeleteModel deletes a semantic model and all its entries.
	DeleteModel(ctx context.Context, modelID int) error

	// ListEntries lists semantic entries for a knowledge base.
	ListEntries(ctx context.Context, params ListSemanticEntriesRequest) (*ListSemanticEntriesResponse, error)

	// CreateEntry creates a semantic entry under a knowledge base's semantic model.
	// If the semantic model does not exist yet, it is auto-created.
	CreateEntry(ctx context.Context, params CreateSemanticEntryRequest) (*SemanticEntry, error)

	// UpdateEntry updates a semantic entry.
	UpdateEntry(ctx context.Context, params UpdateSemanticEntryRequest) error

	// DeleteEntry deletes a semantic entry.
	DeleteEntry(ctx context.Context, params DeleteSemanticEntryRequest) error

	// Import replaces all entries of a knowledge base's semantic model.
	Import(ctx context.Context, params ImportSemanticModelRequest) (*ImportSemanticModelResponse, error)

	// Export exports the semantic model and all entries for a knowledge base.
	Export(ctx context.Context, kbID int) (*ExportSemanticModelResponse, error)

	// Validate validates the semantic model for a knowledge base.
	Validate(ctx context.Context, kbID int) (*ValidateSemanticModelResponse, error)
}

type SemanticModelCatalogDataDomainService interface {
	ResolveDefaultCatalogID(ctx context.Context) (int64, error)
	ResolveCatalogIDByDatabaseID(ctx context.Context, databaseID int64) (int64, error)
	ResolveDatabaseByName(ctx context.Context, catalogID int64, name string) (int64, string, bool, error)
	CreateDatabase(ctx context.Context, catalogID int64, name, description, displayName string) (int64, error)
	CreateVolume(ctx context.Context, databaseID int64, name, description string) (int64, error)
	ResolveVolumeIDByName(ctx context.Context, databaseID int64, name string) (int64, bool, error)
	ListDatabaseTableLeaves(ctx context.Context, params KnowledgeBaseTableLeafListParams) (*KnowledgeBaseTableLeafListResult, error)
	CloneTableForKnowledgeBase(ctx context.Context, sourceTableID, targetDatabaseID int64, idempotencyKey string) (*KnowledgeBaseTableCloneResult, error)
	DeleteVolume(ctx context.Context, volumeID int64) error
	DeleteDatabase(ctx context.Context, databaseID int64) error
}

type SemanticModelCatalogFileService interface {
	ListFiles(ctx context.Context, params KnowledgeBaseCatalogFileListParams) (*KnowledgeBaseCatalogFileListResult, error)
	PreviewFile(ctx context.Context, fileID string) (*SemanticModelFilePreview, error)
	DeleteFileFromVolume(ctx context.Context, volumeID int64, fileID string) error
}

type KnowledgeBaseTableLeafListParams struct {
	DatabaseID int64
	PageSize   int
	PageToken  string
	Search     string
}

type KnowledgeBaseTableLeaf struct {
	TableID    int64
	TableName  string
	DatabaseID int64
}

type KnowledgeBaseTableLeafListResult struct {
	Items         []KnowledgeBaseTableLeaf
	Total         int
	NextPageToken string
}

type KnowledgeBaseCatalogFileListParams struct {
	VolumeID int64
	Page     int
	PageSize int
	FileName string
	FileExt  []string
	FileIDs  []string
}

type KnowledgeBaseCatalogFileLeaf struct {
	FileID   string
	FileName string
	VolumeID int64
}

type KnowledgeBaseCatalogFileListResult struct {
	Items []KnowledgeBaseCatalogFileLeaf
	Total int
}

type SemanticModelLocalFileImportService interface {
	UploadToVolume(ctx context.Context, params KnowledgeBaseLocalFileImportParams) (*KnowledgeBaseLocalFileImportResult, error)
}

type SemanticModelWorkflowTemplateService interface {
	GetByTemplateKey(ctx context.Context, templateKey string) (*model.WorkflowTemplate, error)
}

type SemanticModelWorkflowService interface {
	DeployKnowledgeBaseWorkflow(ctx context.Context, params KnowledgeBaseWorkflowDeployRequest) error
	RequireKnowledgeBaseWorkflow(ctx context.Context, workflowID string) error
	RunKnowledgeBaseWorkflow(ctx context.Context, workflowID string, values map[string]any) (*KnowledgeBaseWorkflowRunResult, error)
	ValidateWorkflowDelete(ctx context.Context, workflowID string) error
	DeleteWorkflow(ctx context.Context, workflowID string) error
	ListFileExecutions(ctx context.Context, fileID string, semanticModelID int64) (*moi.FileExecutionsResponse, error)
}

type KnowledgeBaseLocalFileImportParams struct {
	VolumeID    int64
	FileName    string
	FileID      string
	Reader      io.Reader
	UploadKind  string
	TableConfig string
}

type KnowledgeBaseLocalFileImportResult struct {
	TaskID              string
	FileIDs             []string
	WorkflowExecutionID *string
}

type KnowledgeBaseImportTaskState struct {
	TaskID              string
	Status              string
	Error               *string
	WorkflowExecutionID *string
}

type KnowledgeBaseTableCloneResult struct {
	OperationID string
	Status      string
	SourceDB    string
	SourceTable string
	TargetDB    string
	TargetTable string
	TargetID    *int64
	Error       *string
}

// ========== Structured Service Errors ==========

// ServiceErrorCode identifies the category of a service error.
type ServiceErrorCode string

const (
	ErrCodeNotFound   ServiceErrorCode = "not_found"
	ErrCodeConflict   ServiceErrorCode = "conflict"
	ErrCodeBadRequest ServiceErrorCode = "bad_request"
	ErrCodeInternal   ServiceErrorCode = "internal"
)

// ServiceError is a structured error returned by SemanticModelService.
type ServiceError struct {
	Code ServiceErrorCode
	Msg  string
	Err  error
}

func (e *ServiceError) Error() string {
	if e == nil {
		return ""
	}
	if e.Msg != "" {
		return e.Msg
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return string(e.Code)
}

func (e *ServiceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// IsServiceError checks if err is a *ServiceError with the given code.
func IsServiceError(err error, code ServiceErrorCode) bool {
	var se *ServiceError
	return errors.As(err, &se) && se.Code == code
}
