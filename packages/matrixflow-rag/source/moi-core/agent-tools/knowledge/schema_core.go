package knowledge

import (
	"context"
	"encoding/json"
)

// WorkspaceScope is the common identity carrier for knowledge tools that
// read tenant-scoped RAG and parsed-markdown resources.
type WorkspaceScope struct {
	WorkspaceID                    string                               `json:"workspace_id"`
	UserID                         string                               `json:"user_id,omitempty"`
	SessionID                      string                               `json:"session_id,omitempty"`
	CatalogID                      int64                                `json:"catalog_id,omitempty"`
	DBName                         string                               `json:"db_name,omitempty"`
	Tables                         []string                             `json:"tables,omitempty"`
	SemanticModelIDs               []int64                              `json:"semantic_model_ids,omitempty"`
	VectorTable                    string                               `json:"vector_table,omitempty"`
	EmbeddingModel                 string                               `json:"embedding_model,omitempty"`
	ImageVectorTable               string                               `json:"image_vector_table,omitempty"`
	ImageEmbeddingModel            string                               `json:"image_embedding_model,omitempty"`
	ImageEmbeddingBackendID        string                               `json:"image_embedding_backend_id,omitempty"`
	ImageEmbeddingDimension        int                                  `json:"image_embedding_dimension,omitempty"`
	ImagePreprocessVersion         string                               `json:"image_preprocess_version,omitempty"`
	ImageDistanceMetric            string                               `json:"image_distance_metric,omitempty"`
	VolumeID                       string                               `json:"volume_id,omitempty"`
	FileIDs                        []string                             `json:"file_ids,omitempty"`
	SourceRowIDByFileID            map[string]string                    `json:"source_row_id_by_file_id,omitempty"`
	CurrentSegmentVersionByFileID  map[string]string                    `json:"current_segment_version_by_file_id,omitempty"`
	CurrentIndexVersionByFileID    map[string]int64                     `json:"current_index_version_by_file_id,omitempty"`
	IndexVersionConstraintByFileID map[string]RAGIndexVersionConstraint `json:"index_version_constraint_by_file_id,omitempty"`
	RAGSources                     []RAGSource                          `json:"rag_sources,omitempty"`
}

type RAGSource struct {
	SemanticModelID                int64                                `json:"semantic_model_id,omitempty"`
	SemanticModelName              string                               `json:"semantic_model_name,omitempty"`
	SourceRowIDs                   []string                             `json:"source_row_ids,omitempty"`
	DBName                         string                               `json:"db_name,omitempty"`
	VectorTable                    string                               `json:"vector_table,omitempty"`
	EmbeddingModel                 string                               `json:"embedding_model,omitempty"`
	ImageVectorTable               string                               `json:"image_vector_table,omitempty"`
	ImageEmbeddingModel            string                               `json:"image_embedding_model,omitempty"`
	ImageEmbeddingBackendID        string                               `json:"image_embedding_backend_id,omitempty"`
	ImageEmbeddingDimension        int                                  `json:"image_embedding_dimension,omitempty"`
	ImagePreprocessVersion         string                               `json:"image_preprocess_version,omitempty"`
	ImageDistanceMetric            string                               `json:"image_distance_metric,omitempty"`
	VolumeID                       string                               `json:"volume_id,omitempty"`
	FileIDs                        []string                             `json:"file_ids,omitempty"`
	SourceRowIDByFileID            map[string]string                    `json:"source_row_id_by_file_id,omitempty"`
	CurrentSegmentVersionByFileID  map[string]string                    `json:"current_segment_version_by_file_id,omitempty"`
	CurrentIndexVersionByFileID    map[string]int64                     `json:"current_index_version_by_file_id,omitempty"`
	IndexVersionConstraintByFileID map[string]RAGIndexVersionConstraint `json:"index_version_constraint_by_file_id,omitempty"`
	SourceTags                     []string                             `json:"source_tags,omitempty"`
	SourceTagsByFileID             map[string][]string                  `json:"source_tags_by_file_id,omitempty"`
	Metadata                       map[string]string                    `json:"metadata,omitempty"`
}

type RAGIndexVersionConstraintKind string

const (
	RAGIndexVersionConstraintValue RAGIndexVersionConstraintKind = "value"
	RAGIndexVersionConstraintNull  RAGIndexVersionConstraintKind = "null"
)

type RAGIndexVersionConstraint struct {
	Kind  RAGIndexVersionConstraintKind `json:"kind"`
	Value int64                         `json:"value,omitempty"`
}

type SQLExecutor interface {
	ExecuteSQL(ctx context.Context, dbName string, sql string) (*SQLExecutionResult, error)
}

// SQLMutationExecutor executes a parameterized mutation in the current
// tenant/user database. It is separate from SQLExecutor so read-only tools
// cannot acquire write access accidentally.
type SQLMutationExecutor interface {
	ExecuteMutation(ctx context.Context, dbName string, sql string, args ...any) (int64, error)
}

type SQLExecutionResult struct {
	Columns    []string
	Rows       [][]any
	TotalCount int
}

type EmbeddingService interface {
	CreateEmbedding(ctx context.Context, workspaceID, model string, texts []string) ([][]float64, error)
}

type RetrieverConfig struct {
	EmbeddingModel string `json:"embedding_model,omitempty"`
}

type FindRAGFiles interface {
	Execute(ctx context.Context, req FindRAGFilesRequest) (*FindRAGFilesResponse, error)
}

type FindRAGFilesRequest struct {
	Scope      WorkspaceScope `json:"scope"`
	Query      string         `json:"query,omitempty"`
	Company    string         `json:"company,omitempty"`
	Year       string         `json:"year,omitempty"`
	ReportType string         `json:"report_type,omitempty"`
	VolumeID   string         `json:"volume_id,omitempty"`
	FileIDs    []string       `json:"file_ids,omitempty"`
	MaxFiles   int            `json:"max_files,omitempty"`
}

type FindRAGFilesResponse struct {
	Files    []RAGFileRef `json:"files"`
	RowCount int          `json:"row_count"`
}

type RAGFileRef struct {
	FileID       string   `json:"file_id"`
	IndexVersion string   `json:"index_version"`
	FileName     string   `json:"file_name,omitempty"`
	VolumeID     string   `json:"volume_id,omitempty"`
	SourceURI    string   `json:"source_uri,omitempty"`
	SourceTags   []string `json:"source_tags,omitempty"`
}

type SearchRAGChunks interface {
	Execute(ctx context.Context, req SearchRAGChunksRequest) (*SearchRAGChunksResponse, error)
}

type SearchRAGChunksRequest struct {
	Scope    WorkspaceScope `json:"scope"`
	Keywards []string       `json:"keywards"`
	VolumeID string         `json:"volume_id,omitempty"`
	FileIDs  []string       `json:"file_ids,omitempty"`
	MaxHits  int            `json:"max_hits,omitempty"`
	MaxRows  int            `json:"max_rows,omitempty"`
	Before   int            `json:"before,omitempty"`
	After    int            `json:"after,omitempty"`
}

type SearchRAGChunksResponse struct {
	ArtifactID     string        `json:"artifact_id,omitempty"`
	Keywards       []string      `json:"keywards"`
	Routes         []string      `json:"routes"`
	Chunks         []RAGChunkHit `json:"chunks"`
	RowCount       int           `json:"row_count"`
	ExpandedGroups int           `json:"expanded_groups,omitempty"`
	FulltextSQL    string        `json:"-"`
	VectorSQL      string        `json:"-"`
	ExpansionSQLs  []string      `json:"-"`
	FulltextRows   int           `json:"-"`
	VectorRows     int           `json:"-"`
	EmbeddingModel string        `json:"embedding_model,omitempty"`
}

type RAGChunkHit struct {
	SemanticModelID     int64         `json:"semantic_model_id,omitempty"`
	SourceRowID         string        `json:"source_row_id,omitempty"`
	Content             string        `json:"content"`
	StartMS             *int64        `json:"start_ms,omitempty"`
	EndMS               *int64        `json:"end_ms,omitempty"`
	FileID              string        `json:"file_id"`
	MarkdownFileID      string        `json:"markdown_file_id,omitempty"`
	FileName            string        `json:"file_name,omitempty"`
	PageNumber          int           `json:"page_number,omitempty"`
	VolumeID            string        `json:"volume_id,omitempty"`
	SourceURI           string        `json:"source_uri,omitempty"`
	IndexFileID         string        `json:"index_file_id,omitempty"`
	IndexVersion        string        `json:"index_version,omitempty"`
	ChunkID             string        `json:"chunk_id,omitempty"`
	Level               string        `json:"level,omitempty"`
	ChunkIndex          *int          `json:"chunk_index,omitempty"`
	ChunkIndexScope     string        `json:"chunk_index_scope,omitempty"`
	ParentIndex         *int          `json:"parent_index,omitempty"`
	ChunkStart          *int          `json:"chunk_start,omitempty"`
	ChunkEnd            *int          `json:"chunk_end,omitempty"`
	Score               float64       `json:"score,omitempty"`
	Routes              []string      `json:"routes,omitempty"`
	SourceTags          []string      `json:"source_tags,omitempty"`
	EvidenceGroupID     string        `json:"evidence_group_id,omitempty"`
	Scope               string        `json:"scope,omitempty"`
	ObjectID            string        `json:"object_id,omitempty"`
	ObjectKind          string        `json:"object_kind,omitempty"`
	ImageFileID         string        `json:"image_file_id,omitempty"`
	PageImageFileID     string        `json:"page_image_file_id,omitempty"`
	BBox                []float64     `json:"bbox,omitempty"`
	VisualRefs          []RAGImageRef `json:"visual_refs,omitempty"`
	RetrievalAnchorRank int           `json:"-"`
	BlockUUID           string        `json:"-"`
	ChunkType           string        `json:"-"`
	EmbeddedBlockUUIDs  []string      `json:"-"`
}

type RAGImageRef struct {
	SemanticModelID int64     `json:"semantic_model_id,omitempty"`
	ChunkID         string    `json:"chunk_id,omitempty"`
	Level           string    `json:"level,omitempty"`
	ChunkIndex      *int      `json:"chunk_index,omitempty"`
	ChunkIndexScope string    `json:"chunk_index_scope,omitempty"`
	ParentIndex     *int      `json:"parent_index,omitempty"`
	ChunkStart      *int      `json:"chunk_start,omitempty"`
	ChunkEnd        *int      `json:"chunk_end,omitempty"`
	Score           float64   `json:"score,omitempty"`
	Routes          []string  `json:"routes,omitempty"`
	SourceTags      []string  `json:"source_tags,omitempty"`
	EvidenceGroupID string    `json:"evidence_group_id,omitempty"`
	Scope           string    `json:"scope,omitempty"`
	ObjectID        string    `json:"object_id,omitempty"`
	ObjectKind      string    `json:"object_kind,omitempty"`
	ImageFileID     string    `json:"image_file_id,omitempty"`
	PageImageFileID string    `json:"page_image_file_id,omitempty"`
	Page            int       `json:"page,omitempty"`
	BBox            []float64 `json:"bbox,omitempty"`
}

type QueryVisualRef struct {
	FileID   string         `json:"file_id"`
	FileName string         `json:"file_name,omitempty"`
	MimeType string         `json:"mime_type,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type SearchVisualImage interface {
	Execute(ctx context.Context, req SearchVisualImageRequest) (*SearchVisualImageResponse, error)
}

const (
	VisualSearchRankingProfileVisualObjectFirst = "visual_object_first"
	VisualSearchRankingProfileTextRegionFirst   = "visual_text_region_first"
)

type SearchVisualImageRequest struct {
	Scope             WorkspaceScope `json:"scope"`
	QueryText         string         `json:"query_text,omitempty"`
	QueryVisualFileID string         `json:"query_visual_file_id,omitempty"`
	RankingProfile    string         `json:"ranking_profile,omitempty"`
	TopK              int            `json:"top_k,omitempty"`
}

type SearchVisualImageResponse struct {
	ArtifactID  string            `json:"artifact_id,omitempty"`
	Results     []VisualSearchHit `json:"results"`
	Count       int               `json:"count"`
	Mode        string            `json:"mode,omitempty"`
	Metadata    map[string]any    `json:"metadata,omitempty"`
	Diagnostics []map[string]any  `json:"diagnostics,omitempty"`
}

type VisualSearchHit struct {
	SemanticModelID int64          `json:"semantic_model_id,omitempty"`
	SourceRowID     string         `json:"source_row_id,omitempty"`
	ObjectID        string         `json:"object_id,omitempty"`
	ObjectKind      string         `json:"object_kind,omitempty"`
	Scope           string         `json:"scope,omitempty"`
	SourceFileID    string         `json:"source_file_id,omitempty"`
	SourceFileName  string         `json:"source_file_name,omitempty"`
	PageNumber      int            `json:"page_number,omitempty"`
	BBox            []float64      `json:"bbox,omitempty"`
	ImageFileID     string         `json:"image_file_id,omitempty"`
	PageImageFileID string         `json:"page_image_file_id,omitempty"`
	Score           float64        `json:"score,omitempty"`
	Reason          string         `json:"reason,omitempty"`
	ScoreParts      map[string]any `json:"score_parts,omitempty"`
	Content         string         `json:"content,omitempty"`
	SourceTags      []string       `json:"source_tags,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type ReadParsedMarkdown interface {
	Execute(ctx context.Context, req ReadParsedMarkdownRequest) (*ReadParsedMarkdownResponse, error)
}

type ReadParsedMarkdownRequest struct {
	Scope          WorkspaceScope `json:"scope"`
	MarkdownFileID string         `json:"markdown_file_id"`
	Cursor         int            `json:"cursor,omitempty"`
	LimitChars     int            `json:"limit_chars,omitempty"`
}

type ReadParsedMarkdownResponse struct {
	MarkdownFileID string         `json:"markdown_file_id"`
	FileName       string         `json:"file_name,omitempty"`
	TotalChars     int            `json:"total_chars"`
	Cursor         int            `json:"cursor"`
	ReturnedChars  int            `json:"returned_chars"`
	NextCursor     int            `json:"next_cursor,omitempty"`
	Content        string         `json:"content"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type SearchParsedMarkdown interface {
	Execute(ctx context.Context, req SearchParsedMarkdownRequest) (*SearchParsedMarkdownResponse, error)
}

type SearchParsedMarkdownRequest struct {
	Scope          WorkspaceScope `json:"scope"`
	MarkdownFileID string         `json:"markdown_file_id"`
	Query          string         `json:"query"`
	MaxMatches     int            `json:"max_matches,omitempty"`
	ContextChars   int            `json:"context_chars,omitempty"`
}

type MarkdownMatch struct {
	Start   int    `json:"start"`
	End     int    `json:"end"`
	Snippet string `json:"snippet"`
}

type SearchParsedMarkdownResponse struct {
	MarkdownFileID string          `json:"markdown_file_id"`
	FileName       string          `json:"file_name,omitempty"`
	Query          string          `json:"query"`
	TotalChars     int             `json:"total_chars"`
	Matches        []MarkdownMatch `json:"matches"`
	TotalMatches   int             `json:"total_matches"`
	Truncated      bool            `json:"truncated,omitempty"`
	Metadata       map[string]any  `json:"metadata,omitempty"`
}

type DescribeSchema interface {
	Execute(ctx context.Context, req DescribeSchemaRequest) (*DescribeSchemaResponse, error)
}

type DescribeSchemaRequest struct {
	Scope                   WorkspaceScope `json:"scope"`
	TableNames              []string       `json:"table_names,omitempty"`
	IncludeSamples          int            `json:"include_samples,omitempty"`
	MaxDDLChars             int            `json:"max_ddl_chars,omitempty"`
	SuppressSemanticEntries bool           `json:"-"`
}

type DescribeSchemaResponse struct {
	Tables []TableDescription `json:"tables"`
}

type TableDescription struct {
	Name            string            `json:"name"`
	TableID         int64             `json:"table_id,omitempty"`
	Queryable       *bool             `json:"queryable,omitempty"`
	Access          string            `json:"access,omitempty"`
	DDL             string            `json:"ddl,omitempty"`
	Columns         []ColumnInfo      `json:"columns,omitempty"`
	SampleRows      [][]any           `json:"sample_rows,omitempty"`
	Comments        map[string]string `json:"comments,omitempty"`
	SemanticEntries []SemanticEntry   `json:"semantic_entries,omitempty"`
	Metadata        map[string]any    `json:"metadata,omitempty"`
}

type ColumnInfo struct {
	Name            string  `json:"name"`
	Type            string  `json:"type,omitempty"`
	Comment         string  `json:"comment,omitempty"`
	Nullable        bool    `json:"nullable,omitempty"`
	PrimaryKey      bool    `json:"primary_key,omitempty"`
	IsPrimaryKey    bool    `json:"is_primary_key,omitempty"`
	DistinctRatio   float64 `json:"distinct_ratio,omitempty"`
	PopulationScore float64 `json:"population_score,omitempty"`
}

type SemanticEntry struct {
	ModelID int64           `json:"model_id"`
	Kind    string          `json:"kind"`
	KeyName string          `json:"key_name"`
	Tables  []string        `json:"tables,omitempty"`
	Spec    json.RawMessage `json:"spec,omitempty"`
}

type QuerySQL interface {
	Execute(ctx context.Context, req QuerySQLRequest) (*QuerySQLResponse, error)
}

type QuerySQLRequest struct {
	Scope               WorkspaceScope         `json:"scope"`
	SQL                 string                 `json:"sql"`
	MaxRows             int                    `json:"max_rows,omitempty"`
	SemanticClaims      []string               `json:"semantic_claims,omitempty"`
	AppliedConstraints  []SQLAppliedConstraint `json:"applied_constraints,omitempty"`
	RoundIndex          int                    `json:"round_index,omitempty"`
	RoundQuestion       string                 `json:"round_question,omitempty"`
	SchemaGateValidated bool                   `json:"-"`
}

type QuerySQLResponse struct {
	ArtifactID         string                 `json:"artifact_id,omitempty"`
	SQLIdx             *int                   `json:"sql_idx,omitempty"`
	SQL                string                 `json:"sql"`
	SourceSQLs         []string               `json:"source_sqls,omitempty"`
	Operation          string                 `json:"operation,omitempty"`
	DBName             string                 `json:"db_name,omitempty"`
	Columns            []string               `json:"columns,omitempty"`
	ColumnDescriptions []string               `json:"column_descriptions,omitempty"`
	ColumnFormats      []string               `json:"column_formats,omitempty"`
	Rows               [][]any                `json:"rows,omitempty"`
	RowCount           int                    `json:"row_count"`
	TotalCount         int                    `json:"total_count,omitempty"`
	MaxRows            int                    `json:"max_rows,omitempty"`
	Truncated          bool                   `json:"truncated,omitempty"`
	TableNames         []string               `json:"table_names,omitempty"`
	Tables             []TableRef             `json:"tables,omitempty"`
	SemanticKeysUsed   []string               `json:"semantic_keys_used,omitempty"`
	AppliedConstraints []SQLAppliedConstraint `json:"applied_constraints,omitempty"`
}

// UpsertKnowledgeTable creates or updates confirmed facts in a table from the
// runtime's resolved knowledge scope. The service validates the table and
// columns and builds one parameterized statement for each request.
type UpsertKnowledgeTable interface {
	Execute(ctx context.Context, req UpsertKnowledgeTableRequest) (*UpsertKnowledgeTableResponse, error)
}

type UpsertKnowledgeTableRequest struct {
	Scope     WorkspaceScope               `json:"scope"`
	TableName string                       `json:"table_name"`
	Key       map[string]any               `json:"key"`
	Values    map[string]any               `json:"values"`
	Records   []UpsertKnowledgeTableRecord `json:"records,omitempty"`
}

type UpsertKnowledgeTableRecord struct {
	Key    map[string]any `json:"key"`
	Values map[string]any `json:"values"`
}

type UpsertKnowledgeTableResponse struct {
	TableName    string         `json:"table_name"`
	Key          map[string]any `json:"key,omitempty"`
	RecordCount  int            `json:"record_count"`
	RowsAffected int64          `json:"rows_affected"`
}

type TableRef struct {
	CatalogID int64  `json:"catalog_id,omitempty"`
	TableID   int64  `json:"table_id,omitempty"`
	DBName    string `json:"db_name,omitempty"`
	Name      string `json:"name"`
}

type SQLAppliedConstraint struct {
	Kind      string   `json:"kind,omitempty"`
	Key       string   `json:"key,omitempty"`
	Table     string   `json:"table,omitempty"`
	Alias     string   `json:"alias,omitempty"`
	Column    string   `json:"column,omitempty"`
	Operator  string   `json:"operator,omitempty"`
	Values    []string `json:"values,omitempty"`
	Predicate string   `json:"predicate,omitempty"`
	Status    string   `json:"status,omitempty"`
}

type FinalAnswerSubmission struct {
	Answer  string              `json:"answer"`
	Sources []FinalAnswerSource `json:"sources"`
}

type FinalAnswerSource struct {
	Type            string                 `json:"type"`
	SemanticModelID int64                  `json:"semantic_model_id,omitempty"`
	SourceRowID     string                 `json:"source_row_id,omitempty"`
	StartMS         *int64                 `json:"start_ms,omitempty"`
	EndMS           *int64                 `json:"end_ms,omitempty"`
	ArtifactID      string                 `json:"artifact_id,omitempty"`
	ChunkID         string                 `json:"chunk_id,omitempty"`
	ChunkIDs        []string               `json:"chunk_ids,omitempty"`
	FileID          string                 `json:"file_id,omitempty"`
	FileName        string                 `json:"file_name,omitempty"`
	VolumeID        string                 `json:"volume_id,omitempty"`
	MarkdownFileID  string                 `json:"markdown_file_id,omitempty"`
	Page            int                    `json:"page,omitempty"`
	Pages           []int                  `json:"pages,omitempty"`
	Database        string                 `json:"database,omitempty"`
	Table           string                 `json:"table,omitempty"`
	SQL             string                 `json:"sql,omitempty"`
	Label           string                 `json:"label,omitempty"`
	ObjectID        string                 `json:"object_id,omitempty"`
	ObjectKind      string                 `json:"object_kind,omitempty"`
	ImageFileID     string                 `json:"image_file_id,omitempty"`
	PageImageFileID string                 `json:"page_image_file_id,omitempty"`
	BBox            []float64              `json:"bbox,omitempty"`
	VisualRefs      []FinalAnswerVisualRef `json:"visual_refs,omitempty"`
	SourceTags      []string               `json:"source_tags,omitempty"`
}

type FinalAnswerVisualRef struct {
	SemanticModelID int64     `json:"semantic_model_id,omitempty"`
	ChunkID         string    `json:"chunk_id,omitempty"`
	ObjectID        string    `json:"object_id,omitempty"`
	ObjectKind      string    `json:"object_kind,omitempty"`
	ImageFileID     string    `json:"image_file_id,omitempty"`
	PageImageFileID string    `json:"page_image_file_id,omitempty"`
	Page            int       `json:"page,omitempty"`
	BBox            []float64 `json:"bbox,omitempty"`
}
