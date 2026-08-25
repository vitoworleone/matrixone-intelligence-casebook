package workitems

import "github.com/matrixflow/moi-core/model/data"

type documentVisualParseInput struct {
	Enabled              *bool                  `json:"enabled,omitempty"`
	Sources              []*data.Source         `json:"sources,omitempty"`
	FileID               string                 `json:"file_id,omitempty"`
	FileIDs              []string               `json:"file_ids,omitempty"`
	Documents            []*data.Document       `json:"documents,omitempty"`
	Layout               map[string]interface{} `json:"layout,omitempty"`
	Profile              string                 `json:"profile,omitempty"`
	VLMModel             string                 `json:"vlm_model,omitempty"`
	Options              map[string]interface{} `json:"options,omitempty"`
	ManifestFileName     string                 `json:"manifest_file_name,omitempty"`
	RequirePageImages    *bool                  `json:"require_page_images,omitempty"`
	RequireObjectImages  *bool                  `json:"require_object_images,omitempty"`
	RequireVisualContext *bool                  `json:"require_visual_context,omitempty"`
	WriteManifestFile    *bool                  `json:"write_manifest_file,omitempty"`
}

type documentVisualEngineeringOptions struct {
	EnablePagePlan      bool
	EnableRegionExtract bool
	Model               string
	ReasoningEffort     string
}

type documentVisualParseOutput struct {
	SchemaVersion          string                   `json:"schema_version"`
	Profile                string                   `json:"profile"`
	Manifest               documentVisualManifest   `json:"manifest"`
	Manifests              []documentVisualManifest `json:"manifests,omitempty"`
	ManifestFileID         string                   `json:"manifest_file_id,omitempty"`
	ManifestFileIDs        []string                 `json:"manifest_file_ids,omitempty"`
	DerivedFileIDsBySource map[string][]string      `json:"derived_file_ids_by_source,omitempty"`
	Documents              []*data.Document         `json:"documents,omitempty"`
	Validation             documentVisualValidation `json:"validation"`
	Status                 string                   `json:"status,omitempty"`
}

type documentVisualIndexTextInput struct {
	Enabled            *bool                  `json:"enabled,omitempty"`
	Manifest           documentVisualManifest `json:"manifest,omitempty"`
	ManifestFileID     string                 `json:"manifest_file_id,omitempty"`
	TableName          string                 `json:"table_name,omitempty"`
	TextVectorTable    string                 `json:"text_vector_table,omitempty"`
	EmbeddingModel     string                 `json:"embedding_model,omitempty"`
	EmbeddingDimension int                    `json:"embedding_dimension,omitempty"`
	Policy             string                 `json:"policy,omitempty"`
	FileID             string                 `json:"file_id,omitempty"`
	VolumeID           int64                  `json:"volume_id,omitempty"`
	DatasetMetaTable   string                 `json:"dataset_meta_table,omitempty"`
	EnableMultilevel   *bool                  `json:"enable_multilevel_index,omitempty"`
	SectionSize        int                    `json:"section_size,omitempty"`
}

type documentVisualIndexTextOutput struct {
	Written         int                      `json:"written"`
	DocumentsCount  int                      `json:"documents_count"`
	TextVectorTable string                   `json:"text_vector_table"`
	EmbeddingModel  string                   `json:"embedding_model"`
	IndexVersion    int64                    `json:"index_version,omitempty"`
	ManifestFileID  string                   `json:"manifest_file_id,omitempty"`
	Documents       []map[string]interface{} `json:"documents,omitempty"`
	Status          string                   `json:"status,omitempty"`
}

type documentVisualIndexImageInput struct {
	Enabled                 *bool                    `json:"enabled,omitempty"`
	Manifest                documentVisualManifest   `json:"manifest,omitempty"`
	Manifests               []documentVisualManifest `json:"manifests,omitempty"`
	ManifestFileID          string                   `json:"manifest_file_id,omitempty"`
	ManifestFileIDs         []string                 `json:"manifest_file_ids,omitempty"`
	TableName               string                   `json:"table_name,omitempty"`
	ImageVectorTable        string                   `json:"image_vector_table,omitempty"`
	ImageEmbeddingModel     string                   `json:"image_embedding_model,omitempty"`
	EmbeddingModel          string                   `json:"embedding_model,omitempty"`
	ImageEmbeddingBackendID string                   `json:"image_embedding_backend_id,omitempty"`
	EmbeddingBackendID      string                   `json:"embedding_backend_id,omitempty"`
	ImageEmbeddingDimension int                      `json:"image_embedding_dimension,omitempty"`
	EmbeddingDimension      int                      `json:"embedding_dimension,omitempty"`
	PreprocessVersion       string                   `json:"preprocess_version,omitempty"`
	DistanceMetric          string                   `json:"distance_metric,omitempty"`
	EmbeddingSource         string                   `json:"embedding_source,omitempty"`
	IndexVersion            int64                    `json:"index_version,omitempty"`
	Scopes                  []string                 `json:"scopes,omitempty"`
	Policy                  string                   `json:"policy,omitempty"`
	VolumeID                int64                    `json:"volume_id,omitempty"`
	AllowEmpty              *bool                    `json:"allow_empty,omitempty"`
}

type documentVisualIndexImageOutput struct {
	Written              int                                  `json:"written"`
	PageRows             int                                  `json:"page_rows"`
	VisualObjectRows     int                                  `json:"visual_object_rows"`
	DocumentsCount       int                                  `json:"documents_count"`
	ImageVectorTable     string                               `json:"image_vector_table"`
	EmbeddingModel       string                               `json:"embedding_model"`
	EmbeddingDimension   int                                  `json:"embedding_dimension"`
	EmbeddingBackendID   string                               `json:"embedding_backend_id"`
	PreprocessVersion    string                               `json:"preprocess_version"`
	DistanceMetric       string                               `json:"distance_metric"`
	EmbeddingSource      string                               `json:"embedding_source"`
	IndexVersion         int64                                `json:"index_version,omitempty"`
	ManifestFileID       string                               `json:"manifest_file_id,omitempty"`
	SourceFileID         string                               `json:"source_file_id,omitempty"`
	AllSourceFileIDs     []string                             `json:"all_source_file_ids"`
	IndexedSourceFileIDs []string                             `json:"indexed_source_file_ids"`
	SourceFileIDs        []string                             `json:"source_file_ids"`
	FileStatuses         []documentVisualImageIndexFileStatus `json:"file_statuses,omitempty"`
	Status               string                               `json:"status,omitempty"`
}

type documentVisualImageIndexFileStatus struct {
	SourceFileID  string `json:"source_file_id"`
	Status        string `json:"status"`
	IndexedImages int    `json:"indexed_images"`
}

type documentVisualManifest struct {
	SchemaVersion string                     `json:"schema_version,omitempty"`
	Profile       string                     `json:"profile,omitempty"`
	Source        documentVisualSource       `json:"source,omitempty"`
	Pages         []documentVisualPage       `json:"pages,omitempty"`
	Objects       []documentVisualObject     `json:"objects,omitempty"`
	Entities      []documentVisualEntity     `json:"entities,omitempty"`
	TableSummary  documentVisualTableSummary `json:"table_summary,omitempty"`
	Validation    documentVisualValidation   `json:"validation,omitempty"`
}

type documentVisualSource struct {
	FileID    string `json:"file_id,omitempty"`
	FileName  string `json:"file_name,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	PageCount int    `json:"page_count,omitempty"`
}

type documentVisualPage struct {
	PageNumber      int       `json:"page_number"`
	PageImageFileID string    `json:"page_image_file_id,omitempty"`
	Width           float64   `json:"width,omitempty"`
	Height          float64   `json:"height,omitempty"`
	Summary         string    `json:"summary,omitempty"`
	Text            string    `json:"text,omitempty"`
	ObjectIDs       []string  `json:"object_ids,omitempty"`
	BBox            []float64 `json:"bbox,omitempty"`
}

type documentVisualObject struct {
	ObjectID                      string                       `json:"object_id"`
	ObjectKind                    string                       `json:"object_kind,omitempty"`
	PageNumber                    int                          `json:"page_number"`
	BBox                          []float64                    `json:"bbox,omitempty"`
	OCRBBox                       []float64                    `json:"ocr_bbox,omitempty"`
	OCRBBoxClipped                bool                         `json:"ocr_bbox_clipped,omitempty"`
	ImageFileID                   string                       `json:"image_file_id,omitempty"`
	PageImageFileID               string                       `json:"page_image_file_id,omitempty"`
	Text                          string                       `json:"text,omitempty"`
	OCR                           string                       `json:"ocr,omitempty"`
	FigureCaption                 string                       `json:"figure_caption,omitempty"`
	FigureNo                      string                       `json:"figure_no,omitempty"`
	Caption                       string                       `json:"caption,omitempty"`
	CaptionBlockUUID              string                       `json:"caption_block_uuid,omitempty"`
	Context                       string                       `json:"context,omitempty"`
	SourceBlockID                 string                       `json:"source_block_id,omitempty"`
	ExtractionSource              string                       `json:"extraction_source,omitempty"`
	ExtractionWarnings            []string                     `json:"extraction_warnings,omitempty"`
	ExtractionFailureReason       string                       `json:"extraction_failure_reason,omitempty"`
	EngineeringPagePlanRetryCount int                          `json:"engineering_page_plan_retry_count,omitempty"`
	Table                         *documentVisualTableMetadata `json:"table,omitempty"`
}

type documentVisualEntity struct {
	EntityID   string                 `json:"entity_id,omitempty"`
	Type       string                 `json:"type,omitempty"`
	Value      string                 `json:"value,omitempty"`
	PageNumber int                    `json:"page_number,omitempty"`
	ObjectID   string                 `json:"object_id,omitempty"`
	Evidence   string                 `json:"evidence,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type documentVisualValidation struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type documentVisualBuildOptions struct {
	RequirePageImages    bool
	RequireObjectImages  bool
	RequireVisualContext bool
}

type documentVisualPageAsset struct {
	FileID   string
	Bytes    []byte
	Width    int
	Height   int
	MimeType string
}

type documentVisualLayout struct {
	Pages []documentVisualLayoutPage `json:"pages"`
}

type documentVisualLayoutPage struct {
	PageIdx    int                         `json:"page_idx"`
	PageNumber int                         `json:"page_number"`
	PageSize   []float64                   `json:"page_size"`
	ParaBlocks []documentVisualLayoutBlock `json:"para_blocks"`
}

type documentVisualLayoutBlock struct {
	Type string    `json:"type"`
	BBox []float64 `json:"bbox"`
}
