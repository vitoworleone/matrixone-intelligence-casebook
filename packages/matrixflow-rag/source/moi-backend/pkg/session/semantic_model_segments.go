package session

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/coreclient"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
	"gorm.io/gorm"
)

const (
	kbSegmentStatusCommitted     = "committed"
	kbSegmentStatusMaterialized  = "materializing"
	kbSegmentSourceInitial       = "initial_import"
	kbSegmentSourceEdit          = "edit_chunk"
	kbSegmentSourceCreate        = "create_chunk"
	kbSegmentSourceDisable       = "disable_chunk"
	kbSegmentSourceDelete        = "delete_chunk"
	kbSegmentSourceReembed       = "reembed"
	kbSegmentSourceExternal      = "external_workflow"
	kbSegmentLevelChunk          = "chunk"
	kbSegmentImageIdentityPrefix = "image:"
)

type duplicateSegmentVersionInsertError struct {
	versionID string
	err       error
}

var errExternalWorkflowVectorBindingMismatch = errors.New("external workflow vector rows do not match knowledge base vector binding")

func (e *duplicateSegmentVersionInsertError) Error() string {
	return fmt.Sprintf("insert segment version: %v", e.err)
}

func (e *duplicateSegmentVersionInsertError) Unwrap() error {
	return e.err
}

type kbSegmentVersionRecord struct {
	VersionID           string
	ModelID             int64
	SourceID            string
	KBFileID            string
	IndexVersion        int64
	BaseVersionID       *string
	BaseIndexVersion    *int64
	Status              string
	Source              string
	ChunkCount          int64
	EnabledChunkCount   int64
	VectorTable         string
	EmbeddingModel      string
	ImageVectorTable    *string
	ImageEmbeddingModel *string
	CreatedAt           *int64
	UpdatedAt           *int64
}

type kbSegmentRecord struct {
	SegmentID        string
	VersionID        string
	ModelID          int64
	SourceID         string
	KBFileID         string
	IndexVersion     int64
	Level            string
	ChunkIndex       *int64
	ChunkID          *string
	IdentityKey      string
	Content          *string
	OCRText          *string
	ImageDescription *string
	ImageFileID      *string
	PageImageFileID  *string
	BBox             json.RawMessage
	WordCount        int64
	RecallCount      int64
	Enabled          bool
	Metadata         json.RawMessage
	CreatedAt        *int64
	UpdatedAt        *int64
	ReuseFrom        *kbSegmentRecord
}

type kbVectorBinding struct {
	VectorTable             string
	EmbeddingModel          string
	ImageVectorTable        string
	ImageEmbeddingModel     string
	ImageEmbeddingBackendID string
	ImageEmbeddingDimension int
	ImagePreprocessVersion  string
	ImageDistanceMetric     string
}

type kbVectorTableSchema struct {
	EmbeddingDimension int
	Columns            map[string]struct{}
}

func (s kbVectorTableSchema) HasColumn(name string) bool {
	if s.Columns == nil {
		return false
	}
	_, ok := s.Columns[strings.ToLower(name)]
	return ok
}

type kbVectorInsert struct {
	Segment    kbSegmentRecord
	RowID      string
	Content    string
	Metadata   string
	Embedding  string
	PageNumber *int
}

type kbSegmentImageEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding,omitempty"`
	} `json:"data"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type kbSegmentMaterialization struct {
	TextRows  []kbVectorInsert
	ImageRows []kbVectorInsert
}

type kbReusableVectorRow struct {
	Segment    kbSegmentRecord
	Embedding  string
	PageNumber *int
}

type kbSegmentMaterializationReusePlan struct {
	TextRows  map[string]kbVectorInsert
	ImageRows map[string]kbVectorInsert
}

var errReusableVectorRowNotFound = errors.New("reusable segment vector row not found")

func (s *semanticModelService) buildSourceDocument(ctx context.Context, modelFiles json.RawMessage, record KnowledgeBaseSourceRecord, selectedVersionID string) (*SemanticModelSourceDocument, error) {
	source := sourceRecordToSemanticModelSource(record)
	doc := &SemanticModelSourceDocument{
		Source: source,
		Preview: SemanticModelSourcePreview{
			Available: false,
		},
		FileInfo: SemanticModelSourceFileInfo{
			Tags:             source.Tags,
			ExpiresAt:        source.ExpiresAt,
			Enabled:          source.Enabled,
			Expired:          source.Expired,
			EffectiveEnabled: source.EffectiveEnabled,
			ForceEnabled:     source.ForceEnabled,
			IndexVersion:     source.IndexVersion,
			SegmentVersionID: source.SegmentVersionID,
		},
		CurrentSegmentVersionID: source.SegmentVersionID,
		CurrentIndexVersion:     source.IndexVersion,
		SegmentVersions:         []SemanticModelSegmentVersion{},
		Segments:                []SemanticModelDocumentSegment{},
	}

	versions, err := s.listKnowledgeBaseSegmentVersions(ctx, record.ModelID, record.SourceID)
	if err != nil {
		return nil, err
	}
	for i := range versions {
		if record.SegmentVersionID != nil && versions[i].VersionID == *record.SegmentVersionID {
			versions[i].Current = true
		}
	}
	doc.SegmentVersions = versions

	if selectedVersionID == "" && record.SegmentVersionID != nil {
		selectedVersionID = *record.SegmentVersionID
	}
	if selectedVersionID == "" {
		doc.SegmentStatus = SemanticModelSegmentStatus{Available: false, Total: 0}
		return doc, nil
	}
	selected, ok := findSegmentVersion(versions, selectedVersionID)
	if !ok {
		return nil, segmentVersionNotFoundError()
	}
	doc.SelectedSegmentVersionID = &selected.VersionID
	doc.SelectedIndexVersion = selected.IndexVersion

	segments, err := s.listKnowledgeBaseSegments(ctx, record.ModelID, record.SourceID, selected.VersionID)
	if err != nil {
		return nil, err
	}
	doc.Segments = semanticModelDocumentSegments(segments)
	doc.SegmentStatus = SemanticModelSegmentStatus{Available: true, Total: len(segments)}
	if len(segments) > 0 {
		doc.Preview = SemanticModelSourcePreview{Available: true}
	}
	_ = modelFiles
	return doc, nil
}

func semanticModelDocumentSegments(segments []SemanticModelSegment) []SemanticModelDocumentSegment {
	out := make([]SemanticModelDocumentSegment, 0, len(segments))
	for _, segment := range segments {
		segmentType, startMS, endMS := documentSegmentCanonicalMetadata(segment.Metadata)
		segmentType = canonicalSegmentType(segmentType, segment.ImageFileID, segment.PageImageFileID)
		out = append(out, SemanticModelDocumentSegment{
			SegmentID:        segment.SegmentID,
			SegmentType:      segmentType,
			StartMS:          startMS,
			EndMS:            endMS,
			Level:            segment.Level,
			ChunkIndex:       segment.ChunkIndex,
			ChunkID:          segment.ChunkID,
			Content:          segment.Content,
			OCRText:          segment.OCRText,
			ImageDescription: segment.ImageDescription,
			ImageFileID:      segment.ImageFileID,
			PageImageFileID:  segment.PageImageFileID,
			WordCount:        segment.WordCount,
			RecallCount:      segment.RecallCount,
			Enabled:          segment.Enabled,
			Metadata:         documentSegmentMetadata(segment.Metadata),
		})
	}
	return out
}

func documentSegmentCanonicalMetadata(raw json.RawMessage) (string, *int64, *int64) {
	if len(raw) == 0 {
		return "", nil, nil
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	var metadata map[string]any
	if err := decoder.Decode(&metadata); err != nil {
		return "", nil, nil
	}
	segmentType, _ := metadata["segment_type"].(string)
	return segmentType, documentSegmentMetadataInt64(metadata["start_ms"]), documentSegmentMetadataInt64(metadata["end_ms"])
}

func canonicalSegmentType(segmentType string, imageFileID, pageImageFileID *string) string {
	if segmentType == "image" && strings.TrimSpace(ptrValue(imageFileID)) == "" && strings.TrimSpace(ptrValue(pageImageFileID)) == "" {
		return "text"
	}
	return segmentType
}

func documentSegmentMetadataInt64(value any) *int64 {
	number, ok := value.(json.Number)
	if !ok {
		return nil
	}
	parsed, err := number.Int64()
	if err != nil {
		return nil
	}
	return &parsed
}

func documentSegmentMetadata(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil
	}
	value, ok := metadata["volume_id"]
	if !ok {
		return nil
	}
	trimmed, err := json.Marshal(map[string]any{"volume_id": value})
	if err != nil {
		return nil
	}
	return trimmed
}

func (s *semanticModelService) listKnowledgeBaseSegmentVersions(ctx context.Context, modelID int64, sourceID string) ([]SemanticModelSegmentVersion, error) {
	db := ctxutil.TenantDBFrom(ctx)
	rows, err := db.WithContext(ctx).Raw(`SELECT version_id, index_version, base_version_id, base_index_version, status, source, chunk_count, enabled_chunk_count, created_by, updated_by, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at)
		FROM knowledge_base_segment_versions
		WHERE model_id = ? AND source_id = ?
		ORDER BY index_version DESC, created_at DESC`, modelID, sourceID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SemanticModelSegmentVersion
	for rows.Next() {
		var item SemanticModelSegmentVersion
		var indexVersion sql.NullInt64
		var baseVersion sql.NullString
		var baseIndex sql.NullInt64
		var createdBy sql.NullString
		var updatedBy sql.NullString
		var createdAt sql.NullInt64
		var updatedAt sql.NullInt64
		if err := rows.Scan(&item.VersionID, &indexVersion, &baseVersion, &baseIndex, &item.Status, &item.Source, &item.ChunkCount, &item.EnabledChunkCount, &createdBy, &updatedBy, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if indexVersion.Valid {
			item.IndexVersion = &indexVersion.Int64
		}
		if baseVersion.Valid {
			item.BaseVersionID = &baseVersion.String
		}
		if baseIndex.Valid {
			item.BaseIndexVersion = &baseIndex.Int64
		}
		if createdBy.Valid {
			item.CreatedBy = &createdBy.String
		}
		if updatedBy.Valid {
			item.UpdatedBy = &updatedBy.String
		}
		if createdAt.Valid {
			item.CreatedAt = &createdAt.Int64
		}
		if updatedAt.Valid {
			item.UpdatedAt = &updatedAt.Int64
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *semanticModelService) listKnowledgeBaseSegments(ctx context.Context, modelID int64, sourceID, versionID string) ([]SemanticModelSegment, error) {
	db := ctxutil.TenantDBFrom(ctx)
	rows, err := db.WithContext(ctx).Raw(`SELECT s.segment_id, s.version_id, s.model_id, s.source_id, s.kb_file_id, s.index_version, s.level, s.chunk_index, s.chunk_id,
			s.content, s.ocr_text, s.image_description, s.image_file_id, s.page_image_file_id, s.bbox, s.word_count, COALESCE(st.recall_count, 0), s.enabled, s.metadata, UNIX_TIMESTAMP(s.created_at), UNIX_TIMESTAMP(s.updated_at)
		FROM knowledge_base_segments s
		LEFT JOIN knowledge_base_chunk_recall_stats st
			ON st.model_id = s.model_id
			AND st.source_id = s.source_id
			AND st.kb_file_id = s.kb_file_id
			AND st.index_version = s.index_version
			AND st.level = s.level
			AND st.identity_key = s.identity_key
		WHERE s.model_id = ? AND s.source_id = ? AND s.version_id = ? AND s.level = '`+kbSegmentLevelChunk+`'
		ORDER BY s.level, COALESCE(s.chunk_index, 9223372036854775807), s.segment_id`, modelID, sourceID, versionID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []SemanticModelSegment{}
	for rows.Next() {
		seg, err := scanSemanticModelSegment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, seg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanSemanticModelSegment(rows *sql.Rows) (SemanticModelSegment, error) {
	var seg SemanticModelSegment
	var chunkIndex sql.NullInt64
	var chunkID, content, ocr, desc, imageFileID, pageImageFileID sql.NullString
	var bbox, metadata sql.NullString
	var createdAt, updatedAt sql.NullInt64
	if err := rows.Scan(&seg.SegmentID, &seg.VersionID, &seg.ModelID, &seg.SourceID, &seg.KBFileID, &seg.IndexVersion, &seg.Level, &chunkIndex, &chunkID,
		&content, &ocr, &desc, &imageFileID, &pageImageFileID, &bbox, &seg.WordCount, &seg.RecallCount, &seg.Enabled, &metadata, &createdAt, &updatedAt); err != nil {
		return SemanticModelSegment{}, err
	}
	if chunkIndex.Valid {
		seg.ChunkIndex = &chunkIndex.Int64
	}
	if chunkID.Valid {
		seg.ChunkID = &chunkID.String
	}
	if content.Valid {
		seg.Content = &content.String
	}
	if ocr.Valid {
		seg.OCRText = &ocr.String
	}
	if desc.Valid {
		seg.ImageDescription = &desc.String
	}
	if imageFileID.Valid {
		seg.ImageFileID = &imageFileID.String
	}
	if pageImageFileID.Valid {
		seg.PageImageFileID = &pageImageFileID.String
	}
	if bbox.Valid && json.Valid([]byte(bbox.String)) {
		seg.BBox = json.RawMessage(bbox.String)
	}
	if metadata.Valid && json.Valid([]byte(metadata.String)) {
		seg.Metadata = json.RawMessage(metadata.String)
	}
	if createdAt.Valid {
		seg.CreatedAt = &createdAt.Int64
	}
	if updatedAt.Valid {
		seg.UpdatedAt = &updatedAt.Int64
	}
	return seg, nil
}

func findSegmentVersion(versions []SemanticModelSegmentVersion, versionID string) (SemanticModelSegmentVersion, bool) {
	for _, version := range versions {
		if version.VersionID == versionID {
			return version, true
		}
	}
	return SemanticModelSegmentVersion{}, false
}

func (s *semanticModelService) ImportInitialSegments(ctx context.Context, params ImportInitialSemanticModelSegmentsParams) (*SemanticModelSegmentMutationResult, error) {
	return s.mutateSegments(ctx, params.ModelID, params.SourceID, params.SemanticModelSegmentMutationBase, kbSegmentSourceInitial, func(ctx context.Context, c *moi.Client, wsID string, model *moi.SemanticModel, record KnowledgeBaseSourceRecord, binding kbVectorBinding, current []kbSegmentRecord) ([]kbSegmentRecord, int64, bool, error) {
		if len(current) > 0 {
			return nil, 0, false, initialSegmentVersionExistsError()
		}
		segments, indexVersion, err := s.importInitialSegmentsFromVectorRows(ctx, c, model.Files, record, binding)
		return segments, indexVersion, false, err
	})
}

func (s *semanticModelService) UpdateSegment(ctx context.Context, params UpdateSemanticModelSegmentParams) (*SemanticModelSegmentMutationResult, error) {
	return s.mutateSegments(ctx, params.ModelID, params.SourceID, params.SemanticModelSegmentMutationBase, kbSegmentSourceEdit, func(_ context.Context, _ *moi.Client, _ string, _ *moi.SemanticModel, _ KnowledgeBaseSourceRecord, _ kbVectorBinding, current []kbSegmentRecord) ([]kbSegmentRecord, int64, bool, error) {
		next := cloneSegmentRecords(current)
		matched := mutateClonedSegmentByCurrentID(current, next, params.SegmentID, func(seg *kbSegmentRecord) {
			if params.Content != nil {
				seg.Content = params.Content
			}
			if params.OCRText != nil {
				seg.OCRText = params.OCRText
			}
			if params.ImageDescription != nil {
				seg.ImageDescription = params.ImageDescription
			}
			seg.WordCount = segmentWordCount(*seg)
		})
		if !matched {
			return nil, 0, false, segmentNotFoundError()
		}
		return next, 0, true, nil
	})
}

func (s *semanticModelService) CreateSegment(ctx context.Context, params CreateSemanticModelSegmentParams) (*SemanticModelSegmentMutationResult, error) {
	if params.ImageFileID != nil || params.PageImageFileID != nil {
		return nil, segmentArtifactIdentityReadOnlyError()
	}
	return s.mutateSegments(ctx, params.ModelID, params.SourceID, params.SemanticModelSegmentMutationBase, kbSegmentSourceCreate, func(_ context.Context, _ *moi.Client, _ string, _ *moi.SemanticModel, _ KnowledgeBaseSourceRecord, _ kbVectorBinding, current []kbSegmentRecord) ([]kbSegmentRecord, int64, bool, error) {
		next := cloneSegmentRecords(current)
		level := params.Level
		if level == "" {
			level = kbSegmentLevelChunk
		}
		chunkIndex := int64(0)
		seg := kbSegmentRecord{
			Level:            level,
			ChunkIndex:       &chunkIndex,
			Content:          params.Content,
			OCRText:          params.OCRText,
			ImageDescription: params.ImageDescription,
			ImageFileID:      params.ImageFileID,
			PageImageFileID:  params.PageImageFileID,
			BBox:             params.BBox,
			Metadata:         params.Metadata,
			Enabled:          true,
		}
		seg.IdentityKey = segmentIdentityKey(seg)
		seg.WordCount = segmentWordCount(seg)
		return prependSegmentRecord(next, seg), 0, true, nil
	})
}

func (s *semanticModelService) UpdateSegmentEnabled(ctx context.Context, params UpdateSemanticModelSegmentEnabledParams) (*SemanticModelSegmentMutationResult, error) {
	if params.Enabled == nil {
		return nil, segmentEnabledRequiredError()
	}
	return s.mutateSegments(ctx, params.ModelID, params.SourceID, params.SemanticModelSegmentMutationBase, kbSegmentSourceDisable, func(_ context.Context, _ *moi.Client, _ string, _ *moi.SemanticModel, _ KnowledgeBaseSourceRecord, _ kbVectorBinding, current []kbSegmentRecord) ([]kbSegmentRecord, int64, bool, error) {
		next := cloneSegmentRecords(current)
		matched := mutateClonedSegmentByCurrentID(current, next, params.SegmentID, func(seg *kbSegmentRecord) {
			seg.Enabled = *params.Enabled
		})
		if !matched {
			return nil, 0, false, segmentNotFoundError()
		}
		return next, 0, true, nil
	})
}

func (s *semanticModelService) DeleteSegment(ctx context.Context, params DeleteSemanticModelSegmentParams) (*SemanticModelSegmentMutationResult, error) {
	return s.mutateSegments(ctx, params.ModelID, params.SourceID, params.SemanticModelSegmentMutationBase, kbSegmentSourceDelete, func(_ context.Context, _ *moi.Client, _ string, _ *moi.SemanticModel, _ KnowledgeBaseSourceRecord, _ kbVectorBinding, current []kbSegmentRecord) ([]kbSegmentRecord, int64, bool, error) {
		next, matched := removeClonedSegmentByCurrentID(current, params.SegmentID)
		if !matched {
			return nil, 0, false, segmentNotFoundError()
		}
		return next, 0, true, nil
	})
}

func (s *semanticModelService) ReembedSegments(ctx context.Context, params ReembedSemanticModelSegmentsParams) (*SemanticModelSegmentMutationResult, error) {
	return s.mutateSegments(ctx, params.ModelID, params.SourceID, params.SemanticModelSegmentMutationBase, kbSegmentSourceReembed, func(_ context.Context, _ *moi.Client, _ string, _ *moi.SemanticModel, _ KnowledgeBaseSourceRecord, _ kbVectorBinding, current []kbSegmentRecord) ([]kbSegmentRecord, int64, bool, error) {
		if len(current) == 0 {
			return nil, 0, false, currentSegmentVersionEmptyError()
		}
		return cloneSegmentRecords(current), 0, true, nil
	})
}

func (s *semanticModelService) SetCurrentSegmentVersion(ctx context.Context, params SetCurrentSemanticModelSegmentVersionParams) (*SemanticModelSegmentMutationResult, error) {
	if params.ModelID == 0 || params.SourceID == "" || params.VersionID == "" {
		return nil, segmentVersionRequiredError()
	}
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *SemanticModelSegmentMutationResult
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.setCurrentSegmentVersion(callCtx, client, wsID, params)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) setCurrentSegmentVersion(ctx context.Context, client *moi.Client, wsID string, params SetCurrentSemanticModelSegmentVersionParams) (*SemanticModelSegmentMutationResult, error) {
	model, record, err := s.segmentMutationContext(ctx, client, wsID, int64(params.ModelID), params.SourceID)
	if err != nil {
		return nil, err
	}
	if err := assertSegmentBase(record, params.SemanticModelSegmentMutationBase); err != nil {
		return nil, err
	}
	version, err := s.getSegmentVersionRecord(ctx, record.ModelID, record.SourceID, params.VersionID)
	if err != nil {
		return nil, err
	}
	if version.Status != kbSegmentStatusCommitted {
		return nil, committedSegmentVersionRequiredError()
	}
	db := ctxutil.TenantDBFrom(ctx)
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(`UPDATE knowledge_base_sources
			SET segment_version_id = ?, index_version = ?, updated_by = ?
			WHERE model_id = ? AND source_id = ? AND ((segment_version_id = ?) OR (segment_version_id IS NULL AND ? = '')) AND COALESCE(index_version, 0) = ?`,
			version.VersionID, version.IndexVersion, ctxutil.UIDFrom(ctx),
			record.ModelID, record.SourceID, baseVersionValue(params.BaseSegmentVersionID), baseVersionValue(params.BaseSegmentVersionID), baseIndexValue(params.BaseIndexVersion))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return segmentVersionConflictError()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	record.SegmentVersionID = &version.VersionID
	record.IndexVersion = &version.IndexVersion
	doc, err := s.buildSourceDocument(ctx, model.Files, record, version.VersionID)
	if err != nil {
		return nil, err
	}
	return &SemanticModelSegmentMutationResult{Document: *doc}, nil
}

type segmentMutationFunc func(context.Context, *moi.Client, string, *moi.SemanticModel, KnowledgeBaseSourceRecord, kbVectorBinding, []kbSegmentRecord) ([]kbSegmentRecord, int64, bool, error)

func (s *semanticModelService) mutateSegments(ctx context.Context, modelID int, sourceID string, base SemanticModelSegmentMutationBase, source string, mutate segmentMutationFunc) (*SemanticModelSegmentMutationResult, error) {
	if modelID == 0 || sourceID == "" {
		return nil, semanticModelSourceRequiredError()
	}
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var response *SemanticModelSegmentMutationResult
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		var callErr error
		response, callErr = s.mutateSegmentsWithClient(callCtx, client, wsID, modelID, sourceID, base, source, mutate)
		return callErr
	})
	return response, err
}

func (s *semanticModelService) mutateSegmentsWithClient(ctx context.Context, c *moi.Client, wsID string, modelID int, sourceID string, base SemanticModelSegmentMutationBase, source string, mutate segmentMutationFunc) (*SemanticModelSegmentMutationResult, error) {
	model, record, err := s.segmentMutationContext(ctx, c, wsID, int64(modelID), sourceID)
	if err != nil {
		return nil, err
	}
	if err := assertSegmentBase(record, base); err != nil {
		return nil, err
	}
	binding, err := parseKBVectorBinding(model.Files)
	if err != nil {
		return nil, err
	}
	current, err := s.currentSegmentRecords(ctx, record)
	if err != nil {
		return nil, err
	}
	next, explicitIndexVersion, shouldMaterialize, err := mutate(ctx, c, wsID, model, record, binding, current)
	if err != nil {
		return nil, err
	}
	if len(next) == 0 {
		return nil, segmentVersionEmptyError()
	}
	indexVersion := explicitIndexVersion
	if indexVersion == 0 {
		indexVersion, err = s.nextSegmentIndexVersion(ctx, record)
		if err != nil {
			return nil, err
		}
	}
	versionID := stableID("kb-segver", record.SourceID, indexVersion, source)
	if err := prepareNextSegmentVersion(record, versionID, indexVersion, next); err != nil {
		return nil, err
	}
	materialized := kbSegmentMaterialization{}
	if shouldMaterialize {
		materialized, err = s.materializeSegmentsForMutation(ctx, c, wsID, binding, next, source)
		if err != nil {
			return nil, err
		}
	}
	if err := s.commitSegmentVersion(ctx, record, binding, source, versionID, indexVersion, next, materialized, base); err != nil {
		return nil, err
	}
	record.SegmentVersionID = &versionID
	record.IndexVersion = &indexVersion
	doc, err := s.buildSourceDocument(ctx, model.Files, record, versionID)
	if err != nil {
		return nil, err
	}
	return &SemanticModelSegmentMutationResult{Document: *doc}, nil
}

func (s *semanticModelService) segmentMutationContext(ctx context.Context, c *moi.Client, wsID string, modelID int64, sourceID string) (*moi.SemanticModel, KnowledgeBaseSourceRecord, error) {
	model, err := c.SemanticModels(wsID).Get(ctx, modelID)
	if err != nil {
		return nil, KnowledgeBaseSourceRecord{}, err
	}
	record, err := s.getKnowledgeBaseSource(ctx, modelID, sourceID)
	if err != nil {
		return nil, KnowledgeBaseSourceRecord{}, err
	}
	if record.SourceType == kbSourceTypeCatalogTable {
		return nil, KnowledgeBaseSourceRecord{}, tableSourceSegmentsUnsupportedError()
	}
	if record.KBFileID == nil || *record.KBFileID == "" {
		return nil, KnowledgeBaseSourceRecord{}, knowledgeBaseFileIDRequiredError()
	}
	return model, record, nil
}

func assertSegmentBase(record KnowledgeBaseSourceRecord, base SemanticModelSegmentMutationBase) error {
	if base.BaseSegmentVersionID == nil || base.BaseIndexVersion == nil {
		return segmentBaseRequiredError()
	}
	if baseVersionValue(record.SegmentVersionID) != baseVersionValue(base.BaseSegmentVersionID) || baseIndexValue(record.IndexVersion) != baseIndexValue(base.BaseIndexVersion) {
		return segmentVersionConflictError()
	}
	return nil
}

func parseKBVectorBinding(files json.RawMessage) (kbVectorBinding, error) {
	var payload semanticModelFilesPayload
	if len(files) == 0 {
		return kbVectorBinding{}, semanticModelFilesRequiredError()
	}
	if err := json.Unmarshal(files, &payload); err != nil {
		return kbVectorBinding{}, semanticModelFilesInvalidError()
	}
	if payload.VectorTable == "" {
		return kbVectorBinding{}, segmentVectorTableRequiredError()
	}
	if payload.EmbeddingModel == "" {
		return kbVectorBinding{}, segmentEmbeddingModelRequiredError()
	}
	return kbVectorBinding{
		VectorTable:             payload.VectorTable,
		EmbeddingModel:          payload.EmbeddingModel,
		ImageVectorTable:        payload.ImageVectorTable,
		ImageEmbeddingModel:     payload.ImageEmbeddingModel,
		ImageEmbeddingBackendID: payload.ImageEmbeddingBackendID,
		ImageEmbeddingDimension: payload.ImageEmbeddingDimension,
		ImagePreprocessVersion:  payload.ImagePreprocessVersion,
		ImageDistanceMetric:     payload.ImageDistanceMetric,
	}, nil
}

func (s *semanticModelService) currentSegmentRecords(ctx context.Context, record KnowledgeBaseSourceRecord) ([]kbSegmentRecord, error) {
	if record.SegmentVersionID == nil || *record.SegmentVersionID == "" {
		return nil, nil
	}
	return s.segmentRecords(ctx, record.ModelID, record.SourceID, *record.SegmentVersionID)
}

func (s *semanticModelService) segmentRecords(ctx context.Context, modelID int64, sourceID, versionID string) ([]kbSegmentRecord, error) {
	db := ctxutil.TenantDBFrom(ctx)
	rows, err := db.WithContext(ctx).Raw(`SELECT segment_id, version_id, model_id, source_id, kb_file_id, index_version, level, chunk_index, chunk_id, identity_key,
			content, ocr_text, image_description, image_file_id, page_image_file_id, bbox, word_count, enabled, metadata, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at)
		FROM knowledge_base_segments
		WHERE model_id = ? AND source_id = ? AND version_id = ?
		ORDER BY level, COALESCE(chunk_index, 9223372036854775807), segment_id`, modelID, sourceID, versionID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []kbSegmentRecord{}
	for rows.Next() {
		var seg kbSegmentRecord
		var chunkIndex sql.NullInt64
		var chunkID, content, ocr, desc, imageFileID, pageImageFileID sql.NullString
		var bbox, metadata sql.NullString
		var createdAt, updatedAt sql.NullInt64
		if err := rows.Scan(&seg.SegmentID, &seg.VersionID, &seg.ModelID, &seg.SourceID, &seg.KBFileID, &seg.IndexVersion, &seg.Level, &chunkIndex, &chunkID, &seg.IdentityKey,
			&content, &ocr, &desc, &imageFileID, &pageImageFileID, &bbox, &seg.WordCount, &seg.Enabled, &metadata, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		if chunkIndex.Valid {
			seg.ChunkIndex = &chunkIndex.Int64
		}
		if chunkID.Valid {
			seg.ChunkID = &chunkID.String
		}
		if content.Valid {
			seg.Content = &content.String
		}
		if ocr.Valid {
			seg.OCRText = &ocr.String
		}
		if desc.Valid {
			seg.ImageDescription = &desc.String
		}
		if imageFileID.Valid {
			seg.ImageFileID = &imageFileID.String
		}
		if pageImageFileID.Valid {
			seg.PageImageFileID = &pageImageFileID.String
		}
		if bbox.Valid && json.Valid([]byte(bbox.String)) {
			seg.BBox = json.RawMessage(bbox.String)
		}
		if metadata.Valid && json.Valid([]byte(metadata.String)) {
			seg.Metadata = json.RawMessage(metadata.String)
		}
		if createdAt.Valid {
			seg.CreatedAt = &createdAt.Int64
		}
		if updatedAt.Valid {
			seg.UpdatedAt = &updatedAt.Int64
		}
		out = append(out, seg)
	}
	return out, rows.Err()
}

func cloneSegmentRecords(in []kbSegmentRecord) []kbSegmentRecord {
	out := make([]kbSegmentRecord, len(in))
	for i := range in {
		out[i] = in[i]
		out[i].ReuseFrom = &in[i]
		out[i].SegmentID = ""
		out[i].VersionID = ""
		out[i].IndexVersion = 0
	}
	return out
}

func mutateClonedSegmentByCurrentID(current, cloned []kbSegmentRecord, segmentID string, mutate func(*kbSegmentRecord)) bool {
	for i := range cloned {
		if i >= len(current) || current[i].SegmentID != segmentID {
			continue
		}
		mutate(&cloned[i])
		return true
	}
	return false
}

func removeClonedSegmentByCurrentID(current []kbSegmentRecord, segmentID string) ([]kbSegmentRecord, bool) {
	next := make([]kbSegmentRecord, 0, len(current))
	matched := false
	for i := range current {
		if current[i].SegmentID == segmentID {
			matched = true
			continue
		}
		clone := current[i]
		clone.ReuseFrom = &current[i]
		clone.SegmentID = ""
		clone.VersionID = ""
		clone.IndexVersion = 0
		next = append(next, clone)
	}
	return next, matched
}

func prependSegmentRecord(segments []kbSegmentRecord, seg kbSegmentRecord) []kbSegmentRecord {
	next := make([]kbSegmentRecord, 0, len(segments)+1)
	next = append(next, seg)
	for i := range segments {
		if segments[i].Level == seg.Level && segments[i].ChunkIndex != nil {
			shifted := *segments[i].ChunkIndex + 1
			segments[i].ChunkIndex = &shifted
		}
		next = append(next, segments[i])
	}
	return next
}

func prepareNextSegmentVersion(record KnowledgeBaseSourceRecord, versionID string, indexVersion int64, segments []kbSegmentRecord) error {
	for i := range segments {
		if segments[i].Level == "" {
			segments[i].Level = kbSegmentLevelChunk
		}
		if err := canonicalizeSegmentTypeMetadata(&segments[i]); err != nil {
			return err
		}
		identityKey := segmentIdentityKey(segments[i])
		if identityKey == "" {
			return segmentIdentityRequiredError()
		}
		segments[i].IdentityKey = identityKey
		segments[i].VersionID = versionID
		segments[i].ModelID = record.ModelID
		segments[i].SourceID = record.SourceID
		segments[i].KBFileID = ptrValue(record.KBFileID)
		segments[i].IndexVersion = indexVersion
		segments[i].SegmentID = stableID("kb-segment", versionID, segments[i].Level, identityKey)
		segments[i].WordCount = segmentWordCount(segments[i])
	}
	return nil
}

func canonicalizeSegmentTypeMetadata(seg *kbSegmentRecord) error {
	segmentType, _, _ := documentSegmentCanonicalMetadata(seg.Metadata)
	canonicalType := canonicalSegmentType(segmentType, seg.ImageFileID, seg.PageImageFileID)
	if canonicalType == segmentType {
		return nil
	}
	metadata := map[string]any{}
	if err := json.Unmarshal(seg.Metadata, &metadata); err != nil {
		return fmt.Errorf("decode segment metadata: %w", err)
	}
	metadata["segment_type"] = canonicalType
	raw, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode segment metadata: %w", err)
	}
	seg.Metadata = raw
	return nil
}

func (s *semanticModelService) nextSegmentIndexVersion(ctx context.Context, record KnowledgeBaseSourceRecord) (int64, error) {
	db := ctxutil.TenantDBFrom(ctx)
	var maxVersion sql.NullInt64
	if err := db.WithContext(ctx).Raw(`SELECT MAX(index_version) FROM knowledge_base_segment_versions WHERE model_id = ? AND source_id = ?`, record.ModelID, record.SourceID).Scan(&maxVersion).Error; err != nil {
		return 0, err
	}
	next := int64(1)
	if maxVersion.Valid && maxVersion.Int64 >= next {
		next = maxVersion.Int64 + 1
	}
	if record.IndexVersion != nil && *record.IndexVersion >= next {
		next = *record.IndexVersion + 1
	}
	return next, nil
}

func (s *semanticModelService) importInitialSegmentsFromVectorRows(ctx context.Context, client *moi.Client, modelFiles json.RawMessage, record KnowledgeBaseSourceRecord, binding kbVectorBinding) ([]kbSegmentRecord, int64, error) {
	ready, err := s.sourceReadyForInitialSegmentImport(ctx, client, record)
	if err != nil {
		return nil, 0, err
	}
	if !ready {
		return nil, 0, sourceParsingIncompleteError()
	}
	db := ctxutil.TenantDBFrom(ctx)
	quoted, err := quoteQualifiedSQLIdentifier(binding.VectorTable)
	if err != nil {
		return nil, 0, err
	}
	if _, err := ensureInitialImportVectorTableSchema(ctx, db, binding.VectorTable, quoted, ptrValue(record.KBFileID), initialImportLegacyIndexVersion(record)); err != nil {
		return nil, 0, err
	}
	quotedImageTable, err := quotedOptionalVectorTable(binding.ImageVectorTable)
	if err != nil {
		return nil, 0, err
	}
	indexVersion, err := s.resolveInitialIndexVersion(ctx, db, quoted, record)
	if err != nil {
		return nil, 0, err
	}
	segments, err := readSegmentVectorRows(ctx, db, quoted, record, indexVersion, segmentVectorRowKindText)
	if err != nil {
		return nil, 0, err
	}
	if quotedImageTable != "" {
		imageSegments, err := s.importInitialImageSegmentsFromVectorRows(ctx, db, binding.ImageVectorTable, quotedImageTable, record, indexVersion)
		if err != nil {
			return nil, 0, err
		}
		segments = append(segments, imageSegments...)
	}
	if len(segments) == 0 {
		return nil, 0, segmentRowsUnavailableError()
	}
	return segments, indexVersion, nil
}

func (s *semanticModelService) sourceReadyForInitialSegmentImport(ctx context.Context, client *moi.Client, record KnowledgeBaseSourceRecord) (bool, error) {
	if record.Status == kbSourceStatusSucceeded {
		return true, nil
	}
	jobs, err := s.listKnowledgeBaseSourceJobRuns(ctx, record.ModelID)
	if err != nil {
		return false, err
	}
	if wsID := ctxutil.WorkspaceIDFrom(ctx); wsID != "" {
		jobs, err = s.enrichKnowledgeBaseSourceJobRunsFromLinkedJobs(ctx, client, wsID, jobs)
		if err != nil {
			return false, err
		}
	}
	sourceJobs := make([]KnowledgeBaseSourceJobRun, 0, len(jobs))
	for _, job := range jobs {
		if job.SourceID == record.SourceID {
			sourceJobs = append(sourceJobs, job)
		}
	}
	if len(sourceJobs) == 0 {
		return false, nil
	}
	status, _, _ := deriveKnowledgeBaseSourceStatus(SemanticModelSourceType(record.SourceType), sourceJobs)
	return status == kbSourceStatusSucceeded, nil
}

func (s *semanticModelService) importExternalWorkflowSegmentsFromVectorRows(ctx context.Context, record KnowledgeBaseSourceRecord, binding kbVectorBinding) ([]kbSegmentRecord, int64, error) {
	db := ctxutil.TenantDBFrom(ctx)
	quoted, err := quoteQualifiedSQLIdentifier(binding.VectorTable)
	if err != nil {
		return nil, 0, err
	}
	if _, err := validateVectorTableSchema(ctx, db, binding.VectorTable, 0); err != nil {
		return nil, 0, err
	}
	quotedImageTable, err := quotedOptionalVectorTable(binding.ImageVectorTable)
	if err != nil {
		return nil, 0, err
	}
	vectorRecord := record
	vectorRecord.KBFileID = stringPtr(firstNonEmptySegmentString(ptrValue(record.SourceFileID), ptrValue(record.KBFileID)))
	indexVersion, err := s.resolveLatestVectorIndexVersion(ctx, db, quoted, vectorRecord)
	if err != nil {
		return nil, 0, err
	}
	if record.IndexVersion != nil && indexVersion <= *record.IndexVersion {
		return nil, indexVersion, workflowIndexVersionNotNewerError()
	}
	segments, err := readSegmentVectorRows(ctx, db, quoted, vectorRecord, indexVersion, segmentVectorRowKindText)
	if err != nil {
		return nil, 0, err
	}
	if err := validateExternalWorkflowSegmentMetadata(segments, binding, segmentVectorRowKindText); err != nil {
		return nil, 0, err
	}
	if quotedImageTable != "" {
		imageSegments, err := readOptionalImageSegmentsFromVectorRows(ctx, db, binding.ImageVectorTable, quotedImageTable, vectorRecord, indexVersion)
		if err != nil {
			return nil, 0, err
		}
		if err := validateExternalWorkflowSegmentMetadata(imageSegments, binding, segmentVectorRowKindImage); err != nil {
			return nil, 0, err
		}
		segments = append(segments, imageSegments...)
	}
	if len(segments) == 0 {
		return nil, 0, segmentRowsUnavailableError()
	}
	return segments, indexVersion, nil
}

func validateExternalWorkflowSegmentMetadata(segments []kbSegmentRecord, binding kbVectorBinding, rowKind segmentVectorRowKind) error {
	for _, seg := range segments {
		meta := map[string]any{}
		if len(seg.Metadata) > 0 {
			if err := json.Unmarshal(seg.Metadata, &meta); err != nil {
				return fmt.Errorf("decode external workflow vector metadata %s: %w", seg.IdentityKey, err)
			}
		}
		switch rowKind {
		case segmentVectorRowKindText:
			if !textVectorMetadataMatchesBinding(meta, binding) {
				return fmt.Errorf("%w: text chunk %s", errExternalWorkflowVectorBindingMismatch, seg.IdentityKey)
			}
		case segmentVectorRowKindImage:
			if !imageVectorMetadataMatchesBinding(meta, binding) {
				return fmt.Errorf("%w: image chunk %s", errExternalWorkflowVectorBindingMismatch, seg.IdentityKey)
			}
		}
	}
	return nil
}

func quotedOptionalVectorTable(tableName string) (string, error) {
	if tableName == "" {
		return "", nil
	}
	return quoteQualifiedSQLIdentifier(tableName)
}

func (s *semanticModelService) importInitialImageSegmentsFromVectorRows(ctx context.Context, db *gorm.DB, tableName, quotedTable string, record KnowledgeBaseSourceRecord, indexVersion int64) ([]kbSegmentRecord, error) {
	segments, err := readOptionalImageSegmentsFromVectorRows(ctx, db, tableName, quotedTable, record, indexVersion)
	if err != nil || len(segments) > 0 {
		return segments, err
	}
	repaired, err := repairLegacyImageRowsForInitialImport(ctx, db, tableName, quotedTable, record, indexVersion)
	if err != nil {
		return nil, err
	}
	if !repaired {
		return nil, nil
	}
	return readOptionalImageSegmentsFromVectorRows(ctx, db, tableName, quotedTable, record, indexVersion)
}

func readOptionalImageSegmentsFromVectorRows(ctx context.Context, db *gorm.DB, tableName, quotedTable string, record KnowledgeBaseSourceRecord, indexVersion int64) ([]kbSegmentRecord, error) {
	if tableName == "" || quotedTable == "" {
		return nil, nil
	}
	if _, err := validateVectorTableSchema(ctx, db, tableName, 0); err != nil {
		return nil, err
	}
	// Image segment presence is row-driven: a backend-owned image_vector_table
	// binding means we look for kb_file_id + index_version image chunks. Zero
	// matching rows is valid for documents without page images or visual objects;
	// schema/query errors still fail instead of becoming text-only success.
	segments, err := readSegmentVectorRows(ctx, db, quotedTable, record, indexVersion, segmentVectorRowKindImage)
	if err != nil {
		return nil, err
	}
	return segments, nil
}

func (s *semanticModelService) importSegmentsFromVectorRows(ctx context.Context, db *gorm.DB, quotedTable, quotedImageTable string, record KnowledgeBaseSourceRecord, indexVersion int64) ([]kbSegmentRecord, error) {
	segments, err := readSegmentVectorRows(ctx, db, quotedTable, record, indexVersion, segmentVectorRowKindText)
	if err != nil {
		return nil, err
	}
	if quotedImageTable != "" {
		imageSegments, err := readSegmentVectorRows(ctx, db, quotedImageTable, record, indexVersion, segmentVectorRowKindImage)
		if err != nil {
			return nil, err
		}
		segments = append(segments, imageSegments...)
	}
	if len(segments) == 0 {
		return nil, segmentRowsUnavailableError()
	}
	return segments, nil
}

type segmentVectorRowKind string

const (
	segmentVectorRowKindText  segmentVectorRowKind = "text"
	segmentVectorRowKindImage segmentVectorRowKind = "image"
)

func readSegmentVectorRows(ctx context.Context, db *gorm.DB, quotedTable string, record KnowledgeBaseSourceRecord, indexVersion int64, rowKind segmentVectorRowKind) ([]kbSegmentRecord, error) {
	rows, err := db.WithContext(ctx).Raw(fmt.Sprintf(`SELECT id, content, meta, level, chunk_index, index_version
		FROM %s
		WHERE file_id = ? AND COALESCE(disabled, 0) = 0 AND index_version = ? AND level = '`+kbSegmentLevelChunk+`'
		ORDER BY level, COALESCE(chunk_index, 9223372036854775807), id`, quotedTable), ptrValue(record.KBFileID), indexVersion).Rows()
	if err != nil {
		return nil, fmt.Errorf("read vector rows: %w", err)
	}
	defer rows.Close()

	segments := []kbSegmentRecord{}
	for rows.Next() {
		var id string
		var content, meta, level sql.NullString
		var chunkIndex, rowIndexVersion sql.NullInt64
		if err := rows.Scan(&id, &content, &meta, &level, &chunkIndex, &rowIndexVersion); err != nil {
			return nil, err
		}
		seg, err := segmentFromVectorRow(id, content, meta, level, chunkIndex, rowIndexVersion, indexVersion, rowKind)
		if err != nil {
			return nil, err
		}
		segments = append(segments, seg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return segments, nil
}

func readReusableVectorRows(ctx context.Context, db *gorm.DB, quotedTable, fileID string, indexVersion int64, rowKind segmentVectorRowKind) ([]kbReusableVectorRow, error) {
	selectPageNumber := "NULL"
	if rowKind == segmentVectorRowKindImage {
		selectPageNumber = "page_number"
	}
	rows, err := db.WithContext(ctx).Raw(fmt.Sprintf(`SELECT id, embedding, content, meta, level, chunk_index, index_version, %s
		FROM %s
		WHERE file_id = ? AND COALESCE(disabled, 0) = 0 AND index_version = ?
		ORDER BY level, COALESCE(chunk_index, 9223372036854775807), id`, selectPageNumber, quotedTable), fileID, indexVersion).Rows()
	if err != nil {
		return nil, fmt.Errorf("read reusable vector rows: %w", err)
	}
	defer rows.Close()

	out := []kbReusableVectorRow{}
	for rows.Next() {
		var id string
		var embedding, content, meta, level sql.NullString
		var chunkIndex, rowIndexVersion, pageNumber sql.NullInt64
		if err := rows.Scan(&id, &embedding, &content, &meta, &level, &chunkIndex, &rowIndexVersion, &pageNumber); err != nil {
			return nil, err
		}
		if !embedding.Valid || strings.TrimSpace(embedding.String) == "" {
			continue
		}
		seg, err := segmentFromVectorRow(id, content, meta, level, chunkIndex, rowIndexVersion, indexVersion, rowKind)
		if err != nil {
			return nil, err
		}
		row := kbReusableVectorRow{Segment: seg, Embedding: embedding.String}
		if pageNumber.Valid {
			n := int(pageNumber.Int64)
			row.PageNumber = &n
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *semanticModelService) publishCatalogFileVectorReuse(ctx context.Context, record KnowledgeBaseSourceRecord, binding kbVectorBinding, textCandidate catalogFileVectorReuseCandidate, imageCandidate catalogFileVectorReuseCandidate, actor string) (bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, fmt.Errorf("tenant db is required")
	}
	kbFileID := ptrValue(record.KBFileID)
	if kbFileID == "" {
		return false, nil
	}
	reuseFileID := strings.TrimSpace(textCandidate.ReuseFileID)
	if reuseFileID == "" {
		return false, nil
	}
	textReady, err := ensureVectorReuseTargetTable(ctx, db, textCandidate.VectorTable, binding.VectorTable, 0)
	if err != nil || !textReady {
		return false, err
	}
	sourceTextQuoted, err := quoteQualifiedSQLIdentifier(textCandidate.VectorTable)
	if err != nil {
		return false, err
	}
	maxVersion, err := maxVectorIndexVersion(ctx, db, sourceTextQuoted, reuseFileID, true)
	if err != nil {
		return false, fmt.Errorf("read reusable vector index version: %w", err)
	}
	if !maxVersion.Valid || maxVersion.Int64 <= 0 {
		return false, nil
	}
	indexVersion := maxVersion.Int64
	textRows, err := readReusableVectorRows(ctx, db, sourceTextQuoted, reuseFileID, indexVersion, segmentVectorRowKindText)
	if err != nil || len(textRows) == 0 {
		return false, err
	}

	imageRows := []kbReusableVectorRow{}
	if binding.ImageVectorTable != "" {
		if err := validateImageVectorBinding(binding); err != nil {
			return false, err
		}
		imageReady, err := ensureVectorReuseTargetTable(ctx, db, imageCandidate.VectorTable, binding.ImageVectorTable, binding.ImageEmbeddingDimension)
		if err != nil || !imageReady {
			return false, err
		}
		sourceImageQuoted, err := quoteQualifiedSQLIdentifier(imageCandidate.VectorTable)
		if err != nil {
			return false, err
		}
		imageRows, err = readReusableVectorRows(ctx, db, sourceImageQuoted, reuseFileID, indexVersion, segmentVectorRowKindImage)
		if err != nil {
			return false, err
		}
	}

	segments := make([]kbSegmentRecord, 0, len(textRows)+len(imageRows))
	for _, row := range textRows {
		segments = append(segments, row.Segment)
	}
	for _, row := range imageRows {
		segments = append(segments, row.Segment)
	}
	versionID := stableID("kb-segver", record.SourceID, int64(1), kbSegmentSourceExternal)
	if err := prepareNextSegmentVersion(record, versionID, 1, segments); err != nil {
		return false, err
	}

	copyTextRows := textCandidate.VectorTable != binding.VectorTable || reuseFileID != kbFileID || indexVersion != 1
	copyImageRows := imageCandidate.VectorTable != "" && (imageCandidate.VectorTable != binding.ImageVectorTable || reuseFileID != kbFileID || indexVersion != 1)
	quotedTargetTextTable := ""
	if copyTextRows {
		quotedTargetTextTable, err = quoteQualifiedSQLIdentifier(binding.VectorTable)
		if err != nil {
			return false, err
		}
	}
	quotedTargetImageTable := ""
	if copyImageRows {
		quotedTargetImageTable, err = quoteQualifiedSQLIdentifier(binding.ImageVectorTable)
		if err != nil {
			return false, err
		}
	}
	materialized, err := buildReusableVectorMaterialization(ctx, db, quotedTargetTextTable, quotedTargetImageTable, segments, textRows, imageRows, binding, copyTextRows, copyImageRows)
	if err != nil {
		return false, err
	}
	base := SemanticModelSegmentMutationBase{}
	err = s.commitSegmentVersionWithTxHook(ctx, record, binding, kbSegmentSourceExternal, versionID, 1, segments, materialized, base, nil)
	if err != nil {
		var duplicateVersionErr *duplicateSegmentVersionInsertError
		if errors.As(err, &duplicateVersionErr) && duplicateVersionErr.versionID == versionID {
			alreadyCommitted, checkErr := externalWorkflowSegmentVersionAlreadyCommitted(ctx, record, kbFileID, versionID, 1)
			if checkErr != nil {
				return false, checkErr
			}
			if alreadyCommitted {
				return true, nil
			}
		}
		return false, err
	}
	return true, nil
}

func ensureVectorReuseTargetTable(ctx context.Context, db *gorm.DB, sourceTable, targetTable string, expectedDimension int) (bool, error) {
	if strings.TrimSpace(sourceTable) == "" || strings.TrimSpace(targetTable) == "" {
		return false, nil
	}
	if _, err := validateVectorTableSchema(ctx, db, sourceTable, 0); err != nil {
		return false, fmt.Errorf("validate source vector table %s: %w", sourceTable, err)
	}
	if sourceTable == targetTable {
		if _, err := validateVectorTableSchema(ctx, db, targetTable, expectedDimension); err != nil {
			return false, fmt.Errorf("validate target vector table %s: %w", targetTable, err)
		}
		return true, nil
	}
	if _, err := validateVectorTableSchema(ctx, db, targetTable, expectedDimension); err == nil {
		return true, nil
	} else if !isVectorTableUnavailableError(err) {
		return false, fmt.Errorf("validate target vector table %s: %w", targetTable, err)
	}
	quotedSource, err := quoteQualifiedSQLIdentifier(sourceTable)
	if err != nil {
		return false, err
	}
	quotedTarget, err := quoteQualifiedSQLIdentifier(targetTable)
	if err != nil {
		return false, err
	}
	if err := db.WithContext(ctx).Exec(fmt.Sprintf(`CREATE TABLE %s LIKE %s`, quotedTarget, quotedSource)).Error; err != nil {
		return false, fmt.Errorf("create vector reuse target table %s like %s: %w", targetTable, sourceTable, err)
	}
	if _, err := validateVectorTableSchema(ctx, db, targetTable, expectedDimension); err != nil {
		return false, fmt.Errorf("validate created vector reuse target table %s: %w", targetTable, err)
	}
	return true, nil
}

func isVectorTableUnavailableError(err error) bool {
	if err == nil {
		return false
	}
	var svcErr *ServiceError
	return errors.As(err, &svcErr) && strings.Contains(svcErr.Err.Error(), i18n.KeySessionVectorTableUnavailable.String())
}

func buildReusableVectorMaterialization(ctx context.Context, db *gorm.DB, quotedTextTable, quotedImageTable string, segments []kbSegmentRecord, textRows, imageRows []kbReusableVectorRow, binding kbVectorBinding, copyTextRows, copyImageRows bool) (kbSegmentMaterialization, error) {
	materialized := kbSegmentMaterialization{}
	if copyTextRows {
		for i, row := range textRows {
			seg := segments[i]
			meta, err := segmentTextVectorMetadata(seg, binding)
			if err != nil {
				return kbSegmentMaterialization{}, err
			}
			insert := kbVectorInsert{
				Segment:   seg,
				RowID:     segmentVectorRowID(seg),
				Content:   segmentEmbeddingInput(seg),
				Metadata:  meta,
				Embedding: row.Embedding,
			}
			exists, err := reusableTargetVectorRowExists(ctx, db, quotedTextTable, insert, binding, false)
			if err != nil {
				return kbSegmentMaterialization{}, err
			}
			if exists {
				continue
			}
			materialized.TextRows = append(materialized.TextRows, insert)
		}
	}
	if copyImageRows {
		offset := len(textRows)
		for i, row := range imageRows {
			seg := segments[offset+i]
			imageFileID := segmentImageFileID(seg)
			if imageFileID == "" {
				continue
			}
			pageNumber := 0
			if row.PageNumber != nil {
				pageNumber = *row.PageNumber
			}
			metaMap := map[string]any{}
			if len(row.Segment.Metadata) > 0 && json.Valid(row.Segment.Metadata) {
				_ = json.Unmarshal(row.Segment.Metadata, &metaMap)
			}
			meta, err := segmentImageVectorMetadata(seg, binding, imageFileID, pageNumber, metaMap)
			if err != nil {
				return kbSegmentMaterialization{}, err
			}
			insert := kbVectorInsert{
				Segment:    seg,
				RowID:      segmentImageVectorRowID(seg, imageFileID),
				Content:    segmentEmbeddingInput(seg),
				Metadata:   meta,
				Embedding:  row.Embedding,
				PageNumber: row.PageNumber,
			}
			exists, err := reusableTargetVectorRowExists(ctx, db, quotedImageTable, insert, binding, true)
			if err != nil {
				return kbSegmentMaterialization{}, err
			}
			if exists {
				continue
			}
			materialized.ImageRows = append(materialized.ImageRows, insert)
		}
	}
	return materialized, nil
}

type legacyImageVectorRow struct {
	ID              string
	Embedding       string
	Content         string
	PageNumber      *int64
	Level           string
	ChunkID         string
	ImageFileID     string
	PageImageFileID string
	Metadata        map[string]any
}

func repairLegacyImageRowsForInitialImport(ctx context.Context, db *gorm.DB, tableName, quotedTable string, record KnowledgeBaseSourceRecord, indexVersion int64) (bool, error) {
	if indexVersion <= 0 {
		return false, initialImportIndexVersionPositiveError()
	}
	rows, err := readLegacyImageVectorRowsForRepair(ctx, db, quotedTable, record)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	var existing int64
	if err := db.WithContext(ctx).Raw(fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE file_id = ? AND COALESCE(disabled, 0) = 0 AND index_version = ?`, quotedTable), ptrValue(record.KBFileID), indexVersion).Scan(&existing).Error; err != nil {
		return false, fmt.Errorf("check versioned image vector rows: %w", err)
	}
	if existing > 0 {
		return false, nil
	}
	if _, err := validateVectorTableSchema(ctx, db, tableName, 0); err != nil {
		return false, err
	}
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			meta := row.Metadata
			meta["index_version"] = indexVersion
			meta["chunk_id"] = row.ChunkID
			delete(meta, "chunk_index")
			metaText, err := json.Marshal(meta)
			if err != nil {
				return err
			}
			rowID := stableID("kbimgrow", ptrValue(record.KBFileID), indexVersion, row.Level, row.ChunkID, firstNonEmptySegmentString(row.ImageFileID, row.PageImageFileID))
			if err := tx.Exec(fmt.Sprintf(`INSERT INTO %s
				(id, embedding, content, meta, file_id, page_number, level, doc_id, section_id, chunk_index, index_version, disabled)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, quotedTable),
				rowID, row.Embedding, row.Content, string(metaText), ptrValue(record.KBFileID), row.PageNumber, row.Level, nil, nil, nil, indexVersion, false).Error; err != nil {
				return fmt.Errorf("insert repaired legacy image vector row: %w", err)
			}
		}
		return nil
	}); err != nil {
		return false, err
	}
	return true, nil
}

func readLegacyImageVectorRowsForRepair(ctx context.Context, db *gorm.DB, quotedTable string, record KnowledgeBaseSourceRecord) ([]legacyImageVectorRow, error) {
	rows, err := db.WithContext(ctx).Raw(fmt.Sprintf(`SELECT id, embedding, content, meta, page_number, level
		FROM %s
		WHERE file_id = ? AND COALESCE(disabled, 0) = 0 AND COALESCE(index_version, 0) = 0
		ORDER BY COALESCE(page_number, 0), id`, quotedTable), ptrValue(record.KBFileID)).Rows()
	if err != nil {
		return nil, fmt.Errorf("read legacy image vector rows: %w", err)
	}
	defer rows.Close()

	out := []legacyImageVectorRow{}
	for rows.Next() {
		var row legacyImageVectorRow
		var meta sql.NullString
		var pageNumber sql.NullInt64
		var level sql.NullString
		if err := rows.Scan(&row.ID, &row.Embedding, &row.Content, &meta, &pageNumber, &level); err != nil {
			return nil, err
		}
		if level.Valid && level.String != "" {
			row.Level = level.String
		} else {
			row.Level = kbSegmentLevelChunk
		}
		if pageNumber.Valid {
			v := pageNumber.Int64
			row.PageNumber = &v
		}
		metaMap := map[string]any{}
		if !meta.Valid || !json.Valid([]byte(meta.String)) {
			return nil, legacyImageVectorMetadataInvalidError()
		}
		if err := json.Unmarshal([]byte(meta.String), &metaMap); err != nil {
			return nil, err
		}
		chunkID, imageFileID, pageImageFileID, err := legacyDocumentVisualImageIdentity(metaMap)
		if err != nil {
			return nil, err
		}
		row.ChunkID = chunkID
		row.ImageFileID = imageFileID
		row.PageImageFileID = pageImageFileID
		row.Metadata = metaMap
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func legacyDocumentVisualImageIdentity(metaMap map[string]any) (string, string, string, error) {
	if stringFromMap(metaMap, "asset_kind") != "document_visual" {
		return "", "", "", legacyImageVectorKindInvalidError()
	}
	scope := stringFromMap(metaMap, "scope")
	objectID := stringFromMap(metaMap, "object_id")
	imageFileID := stringFromMap(metaMap, "image_file_id")
	pageImageFileID := stringFromMap(metaMap, "page_image_file_id")
	if scope == "" || objectID == "" || firstNonEmptySegmentString(imageFileID, pageImageFileID) == "" {
		return "", "", "", legacyImageVectorIdentityMissingError()
	}
	return "document_visual_image:" + scope + ":" + objectID, imageFileID, pageImageFileID, nil
}

func firstNonEmptySegmentString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func segmentFromVectorRow(id string, content, meta, level sql.NullString, chunkIndex, rowIndexVersion sql.NullInt64, indexVersion int64, rowKind segmentVectorRowKind) (kbSegmentRecord, error) {
	metaMap := map[string]any{}
	if meta.Valid && json.Valid([]byte(meta.String)) {
		_ = json.Unmarshal([]byte(meta.String), &metaMap)
	}
	chunkID := stringFromMap(metaMap, "chunk_id")
	if !chunkIndex.Valid && chunkID == "" {
		return kbSegmentRecord{}, segmentIdentityRequiredError()
	}
	seg := kbSegmentRecord{
		Level:        kbSegmentLevelChunk,
		Content:      nil,
		Enabled:      true,
		Metadata:     validRawMessage(meta),
		IndexVersion: indexVersion,
	}
	if rowIndexVersion.Valid && rowIndexVersion.Int64 > 0 {
		seg.IndexVersion = rowIndexVersion.Int64
	}
	if level.Valid && level.String != "" {
		seg.Level = level.String
	}
	if content.Valid && rowKind == segmentVectorRowKindText {
		seg.Content = &content.String
	}
	if chunkIndex.Valid {
		seg.ChunkIndex = &chunkIndex.Int64
	} else {
		seg.ChunkID = &chunkID
	}
	if rowKind == segmentVectorRowKindImage {
		promoteImageSegmentMetadata(&seg, metaMap)
	}
	seg.IdentityKey = chunkIdentityKey(seg.ChunkIndex, seg.ChunkID)
	if rowKind == segmentVectorRowKindImage {
		seg.IdentityKey = imageSegmentIdentityKey(seg.IdentityKey)
	}
	seg.WordCount = segmentWordCount(seg)
	return seg, nil
}

func validRawMessage(value sql.NullString) json.RawMessage {
	if !value.Valid || !json.Valid([]byte(value.String)) {
		return nil
	}
	return json.RawMessage(value.String)
}

func promoteImageSegmentMetadata(seg *kbSegmentRecord, metaMap map[string]any) {
	if v := stringFromMap(metaMap, "ocr_text"); v != "" {
		seg.OCRText = &v
	}
	if v := segmentImageDescriptionFromMetadata(metaMap); v != "" {
		seg.ImageDescription = &v
	}
	if v := stringFromMap(metaMap, "image_file_id"); v != "" {
		seg.ImageFileID = &v
	}
	if v := stringFromMap(metaMap, "page_image_file_id"); v != "" {
		seg.PageImageFileID = &v
	}
	if raw, ok := metaMap["bbox"]; ok {
		if b, marshalErr := json.Marshal(raw); marshalErr == nil && json.Valid(b) {
			seg.BBox = json.RawMessage(b)
		}
	}
}

func segmentImageDescriptionFromMetadata(metaMap map[string]any) string {
	if v := stringFromMap(metaMap, "image_description"); v != "" {
		return v
	}
	if v := stringFromMap(metaMap, "caption"); v != "" {
		return v
	}
	if v := stringFromMap(metaMap, "description"); v != "" {
		return v
	}
	return ""
}

func ensureInitialImportVectorTableSchema(ctx context.Context, db *gorm.DB, tableName, quotedTable, fileID string, legacyIndexVersion int64) (kbVectorTableSchema, error) {
	if legacyIndexVersion <= 0 {
		legacyIndexVersion = 1
	}
	schema, err := inspectVectorTableSchema(ctx, db, tableName)
	if err != nil {
		return kbVectorTableSchema{}, err
	}
	for _, required := range []string{"id", "embedding", "content", "meta", "file_id", "level"} {
		if _, ok := schema.Columns[required]; !ok {
			return kbVectorTableSchema{}, vectorTableColumnMissingError(required)
		}
	}
	if schema.EmbeddingDimension <= 0 {
		return kbVectorTableSchema{}, vectorTableEmbeddingColumnInvalidError()
	}
	for _, column := range []struct {
		name string
		def  string
	}{
		{name: "chunk_index", def: "INT NULL"},
		{name: "index_version", def: "BIGINT DEFAULT 0"},
		{name: "disabled", def: "TINYINT(1) DEFAULT 0"},
	} {
		if _, ok := schema.Columns[column.name]; ok {
			continue
		}
		if err := db.WithContext(ctx).Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", quotedTable, column.name, column.def)).Error; err != nil {
			return kbVectorTableSchema{}, fmt.Errorf("add legacy vector column %s: %w", column.name, err)
		}
		schema.Columns[column.name] = struct{}{}
	}
	if err := db.WithContext(ctx).Exec(fmt.Sprintf(`UPDATE %s SET index_version = ? WHERE file_id = ? AND (index_version IS NULL OR index_version = 0)`, quotedTable), legacyIndexVersion, fileID).Error; err != nil {
		return kbVectorTableSchema{}, fmt.Errorf("backfill legacy vector index_version: %w", err)
	}
	if err := db.WithContext(ctx).Exec(fmt.Sprintf(`UPDATE %s SET disabled = ? WHERE file_id = ? AND disabled IS NULL`, quotedTable), false, fileID).Error; err != nil {
		return kbVectorTableSchema{}, fmt.Errorf("backfill legacy vector disabled: %w", err)
	}
	if err := backfillLegacyVectorChunkIndex(ctx, db, quotedTable, fileID); err != nil {
		return kbVectorTableSchema{}, err
	}
	return schema, nil
}

func initialImportLegacyIndexVersion(record KnowledgeBaseSourceRecord) int64 {
	if record.IndexVersion != nil && *record.IndexVersion > 0 {
		return *record.IndexVersion
	}
	return 1
}

func backfillLegacyVectorChunkIndex(ctx context.Context, db *gorm.DB, quotedTable, fileID string) error {
	rows, err := db.WithContext(ctx).Raw(fmt.Sprintf(`SELECT id FROM %s WHERE file_id = ? AND chunk_index IS NULL ORDER BY id`, quotedTable), fileID).Rows()
	if err != nil {
		return fmt.Errorf("read legacy vector rows for chunk_index: %w", err)
	}
	defer rows.Close()
	rowIDs := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		if id == "" {
			return fmt.Errorf("legacy vector row for file %s has empty id", fileID)
		}
		rowIDs = append(rowIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for idx, id := range rowIDs {
		if err := db.WithContext(ctx).Exec(fmt.Sprintf(`UPDATE %s SET chunk_index = ? WHERE id = ?`, quotedTable), int64(idx), id).Error; err != nil {
			return fmt.Errorf("backfill legacy vector chunk_index for row %s: %w", id, err)
		}
	}
	return nil
}

func (s *semanticModelService) resolveInitialIndexVersion(ctx context.Context, db *gorm.DB, quotedTable string, record KnowledgeBaseSourceRecord) (int64, error) {
	if record.IndexVersion != nil && *record.IndexVersion > 0 {
		return *record.IndexVersion, nil
	}
	maxVersion, err := maxVectorIndexVersion(ctx, db, quotedTable, ptrValue(record.KBFileID), false)
	if err != nil {
		return 0, fmt.Errorf("read current vector index_version: %w", err)
	}
	if !maxVersion.Valid {
		return 0, initialSegmentRowsUnavailableError()
	}
	return maxVersion.Int64, nil
}

func (s *semanticModelService) resolveLatestVectorIndexVersion(ctx context.Context, db *gorm.DB, quotedTable string, record KnowledgeBaseSourceRecord) (int64, error) {
	maxVersion, err := maxVectorIndexVersion(ctx, db, quotedTable, ptrValue(record.KBFileID), false)
	if err != nil {
		return 0, fmt.Errorf("read workflow vector index_version: %w", err)
	}
	if !maxVersion.Valid || maxVersion.Int64 <= 0 {
		return 0, workflowSegmentRowsUnavailableError()
	}
	return maxVersion.Int64, nil
}

func maxVectorIndexVersion(ctx context.Context, db *gorm.DB, quotedTable, fileID string, positiveOnly bool) (sql.NullInt64, error) {
	var maxVersion sql.NullInt64
	query := fmt.Sprintf(`SELECT MAX(index_version) FROM %s WHERE file_id = ?`, quotedTable)
	if positiveOnly {
		query += ` AND index_version > 0`
	}
	if err := db.WithContext(ctx).Raw(query, fileID).Scan(&maxVersion).Error; err != nil {
		return sql.NullInt64{}, err
	}
	return maxVersion, nil
}

func (s *semanticModelService) materializeSegments(ctx context.Context, c *moi.Client, wsID string, binding kbVectorBinding, segments []kbSegmentRecord) (kbSegmentMaterialization, error) {
	return s.materializeSegmentsWithReuse(ctx, c, wsID, binding, segments, kbSegmentMaterializationReusePlan{})
}

func (s *semanticModelService) materializeSegmentsForMutation(ctx context.Context, c *moi.Client, wsID string, binding kbVectorBinding, segments []kbSegmentRecord, source string) (kbSegmentMaterialization, error) {
	if source == kbSegmentSourceReembed {
		return s.materializeSegments(ctx, c, wsID, binding, segments)
	}
	reusePlan, err := s.planReusableSegmentRows(ctx, binding, segments)
	if err != nil {
		return kbSegmentMaterialization{}, err
	}
	return s.materializeSegmentsWithReuse(ctx, c, wsID, binding, segments, reusePlan)
}

func (s *semanticModelService) materializeSegmentsWithReuse(ctx context.Context, c *moi.Client, wsID string, binding kbVectorBinding, segments []kbSegmentRecord, reusePlan kbSegmentMaterializationReusePlan) (kbSegmentMaterialization, error) {
	textRows, err := s.materializeTextSegments(ctx, c, wsID, binding, segments, reusePlan)
	if err != nil {
		return kbSegmentMaterialization{}, err
	}
	imageRows, err := s.materializeImageSegments(ctx, c, wsID, binding, segments, reusePlan)
	if err != nil {
		return kbSegmentMaterialization{}, err
	}
	return kbSegmentMaterialization{TextRows: textRows, ImageRows: imageRows}, nil
}

func (s *semanticModelService) materializeTextSegments(ctx context.Context, c *moi.Client, wsID string, binding kbVectorBinding, segments []kbSegmentRecord, reusePlan kbSegmentMaterializationReusePlan) ([]kbVectorInsert, error) {
	db := ctxutil.TenantDBFrom(ctx)
	enabled := make([]kbSegmentRecord, 0, len(segments))
	inputs := make([]string, 0, len(segments))
	rows := make([]kbVectorInsert, 0, len(segments))
	for _, seg := range segments {
		if !seg.Enabled {
			continue
		}
		if row, ok := reusePlan.TextRows[segmentVectorRowID(seg)]; ok {
			rows = append(rows, row)
			continue
		}
		text := segmentEmbeddingInput(seg)
		if text == "" {
			if segmentImageFileID(seg) != "" && binding.ImageVectorTable != "" {
				continue
			}
			return nil, segmentEmbeddingInputEmptyError()
		}
		enabled = append(enabled, seg)
		inputs = append(inputs, text)
	}
	if len(enabled) == 0 {
		_, err := validateVectorTableSchema(ctx, db, binding.VectorTable, 0)
		return rows, err
	}
	resp, err := c.Embeddings(wsID).CreateEmbeddings(ctx, binding.EmbeddingModel, inputs)
	if err != nil {
		return nil, fmt.Errorf("create segment embeddings: %w", err)
	}
	if len(resp.GetData()) != len(enabled) {
		return nil, segmentEmbeddingResponseInvalidError()
	}
	firstEmbedding := resp.GetData()[0].GetEmbedding()
	schema, err := validateVectorTableSchema(ctx, db, binding.VectorTable, len(firstEmbedding))
	if err != nil {
		return nil, err
	}
	if schema.EmbeddingDimension > 0 && schema.EmbeddingDimension != len(firstEmbedding) {
		return nil, segmentEmbeddingDimensionMismatchError()
	}
	for i, seg := range enabled {
		embedding := resp.GetData()[i].GetEmbedding()
		if len(embedding) != len(firstEmbedding) {
			return nil, segmentEmbeddingDimensionMismatchError()
		}
		meta, err := segmentTextVectorMetadata(seg, binding)
		if err != nil {
			return nil, err
		}
		embeddingJSON, err := json.Marshal(embedding)
		if err != nil {
			return nil, err
		}
		rows = append(rows, kbVectorInsert{
			Segment:   seg,
			RowID:     segmentVectorRowID(seg),
			Content:   segmentEmbeddingInput(seg),
			Metadata:  meta,
			Embedding: string(embeddingJSON),
		})
	}
	return rows, nil
}

func (s *semanticModelService) materializeImageSegments(ctx context.Context, c *moi.Client, wsID string, binding kbVectorBinding, segments []kbSegmentRecord, reusePlan kbSegmentMaterializationReusePlan) ([]kbVectorInsert, error) {
	if binding.ImageVectorTable == "" {
		for _, seg := range segments {
			if seg.Enabled && segmentEmbeddingInput(seg) == "" && segmentImageFileID(seg) != "" {
				return nil, imageVectorTableRequiredError()
			}
		}
		return nil, nil
	}
	if err := validateImageVectorBinding(binding); err != nil {
		return nil, err
	}
	db := ctxutil.TenantDBFrom(ctx)
	schema, err := validateVectorTableSchema(ctx, db, binding.ImageVectorTable, binding.ImageEmbeddingDimension)
	if err != nil {
		return nil, err
	}
	if !schema.HasColumn("page_number") {
		return nil, imageVectorTableColumnMissingError("page_number")
	}

	rows := make([]kbVectorInsert, 0, len(segments))
	for _, seg := range segments {
		if !seg.Enabled {
			continue
		}
		imageFileID := segmentImageFileID(seg)
		if imageFileID == "" {
			continue
		}
		if row, ok := reusePlan.ImageRows[segmentImageVectorRowID(seg, imageFileID)]; ok {
			rows = append(rows, row)
			continue
		}
		row, err := s.materializeImageSegment(ctx, c, wsID, binding, seg, imageFileID)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func (s *semanticModelService) planReusableSegmentRows(ctx context.Context, binding kbVectorBinding, segments []kbSegmentRecord) (kbSegmentMaterializationReusePlan, error) {
	plan := kbSegmentMaterializationReusePlan{
		TextRows:  map[string]kbVectorInsert{},
		ImageRows: map[string]kbVectorInsert{},
	}
	db := ctxutil.TenantDBFrom(ctx)
	quotedTextTable := ""
	quotedImageTable := ""
	for _, seg := range segments {
		if canReuseTextSegment(seg) {
			if quotedTextTable == "" {
				quoted, err := quoteQualifiedSQLIdentifier(binding.VectorTable)
				if err != nil {
					return kbSegmentMaterializationReusePlan{}, err
				}
				quotedTextTable = quoted
			}
			row, err := reusedTextVectorRow(ctx, db, quotedTextTable, binding, seg)
			if err != nil {
				if !errors.Is(err, errReusableVectorRowNotFound) {
					return kbSegmentMaterializationReusePlan{}, err
				}
			} else {
				plan.TextRows[row.RowID] = row
			}
		}
		if canReuseImageSegmentCandidate(seg, binding) {
			if quotedImageTable == "" {
				quoted, err := quoteQualifiedSQLIdentifier(binding.ImageVectorTable)
				if err != nil {
					return kbSegmentMaterializationReusePlan{}, err
				}
				quotedImageTable = quoted
			}
			row, reusable, err := reusedImageVectorRow(ctx, db, quotedImageTable, binding, seg)
			if err != nil {
				return kbSegmentMaterializationReusePlan{}, err
			}
			if reusable {
				plan.ImageRows[row.RowID] = row
			}
		}
	}
	return plan, nil
}

func canReuseTextSegment(seg kbSegmentRecord) bool {
	if !seg.Enabled || seg.ReuseFrom == nil || !seg.ReuseFrom.Enabled {
		return false
	}
	input := segmentEmbeddingInput(seg)
	return input != "" && input == segmentEmbeddingInput(*seg.ReuseFrom)
}

func canReuseImageSegmentCandidate(seg kbSegmentRecord, binding kbVectorBinding) bool {
	if binding.ImageVectorTable == "" || !seg.Enabled || seg.ReuseFrom == nil || !seg.ReuseFrom.Enabled {
		return false
	}
	imageFileID := segmentImageFileID(seg)
	return imageFileID != "" && imageFileID == segmentImageFileID(*seg.ReuseFrom)
}

func reusedTextVectorRow(ctx context.Context, db *gorm.DB, quotedTable string, binding kbVectorBinding, seg kbSegmentRecord) (kbVectorInsert, error) {
	old := *seg.ReuseFrom
	embedding, oldMetadata, err := readReusableVectorRow(ctx, db, quotedTable, segmentVectorRowID(old), old, "")
	if err != nil {
		return kbVectorInsert{}, err
	}
	if !textVectorMetadataMatchesBinding(oldMetadata, binding) {
		return kbVectorInsert{}, errReusableVectorRowNotFound
	}
	meta, err := segmentTextVectorMetadata(seg, binding)
	if err != nil {
		return kbVectorInsert{}, err
	}
	return kbVectorInsert{
		Segment:   seg,
		RowID:     segmentVectorRowID(seg),
		Content:   segmentEmbeddingInput(seg),
		Metadata:  meta,
		Embedding: embedding,
	}, nil
}

func reusedImageVectorRow(ctx context.Context, db *gorm.DB, quotedTable string, binding kbVectorBinding, seg kbSegmentRecord) (kbVectorInsert, bool, error) {
	old := *seg.ReuseFrom
	imageFileID := segmentImageFileID(seg)
	pageNumber, ok := segmentPageNumber(seg)
	if !ok {
		return kbVectorInsert{}, false, imageSegmentPageNumberRequiredError()
	}
	embedding, oldMetadata, err := readReusableVectorRow(ctx, db, quotedTable, segmentImageVectorRowID(old, imageFileID), old, imageFileID)
	if err != nil {
		return kbVectorInsert{}, false, err
	}
	if !imageVectorMetadataMatchesBinding(oldMetadata, binding) {
		return kbVectorInsert{}, false, nil
	}
	rawMeta := embeddingBackendMetadata(oldMetadata)
	meta, err := segmentImageVectorMetadata(seg, binding, imageFileID, pageNumber, rawMeta)
	if err != nil {
		return kbVectorInsert{}, false, err
	}
	return kbVectorInsert{
		Segment:    seg,
		RowID:      segmentImageVectorRowID(seg, imageFileID),
		Content:    segmentEmbeddingInput(seg),
		Metadata:   meta,
		Embedding:  embedding,
		PageNumber: &pageNumber,
	}, true, nil
}

func readReusableVectorRow(ctx context.Context, db *gorm.DB, quotedTable, rowID string, old kbSegmentRecord, imageFileID string) (string, map[string]any, error) {
	embedding, meta, err := readReusableVectorRowByID(ctx, db, quotedTable, rowID, old)
	if err == nil {
		return embedding, meta, nil
	}
	if !errors.Is(err, errReusableVectorRowNotFound) {
		return "", nil, err
	}
	// External workflow rows keep workflow-generated ids. Segment snapshots are
	// derived from the same row identity, so copy-forward can locate them by
	// file/version/level/chunk identity without matching on content.
	embedding, meta, identityErr := readReusableVectorRowByIdentity(ctx, db, quotedTable, old, imageFileID)
	if identityErr == nil {
		return embedding, meta, nil
	}
	if errors.Is(identityErr, errReusableVectorRowNotFound) {
		return "", nil, fmt.Errorf("reuse previous segment vector row %s: %w", rowID, errReusableVectorRowNotFound)
	}
	return "", nil, identityErr
}

func readReusableVectorRowByID(ctx context.Context, db *gorm.DB, quotedTable, rowID string, old kbSegmentRecord) (string, map[string]any, error) {
	var embedding string
	var metadata sql.NullString
	err := db.WithContext(ctx).Raw(fmt.Sprintf(`SELECT embedding, meta FROM %s WHERE id = ? AND file_id = ? AND index_version = ? AND disabled = ? LIMIT 1`, quotedTable),
		rowID, old.KBFileID, old.IndexVersion, false).Row().Scan(&embedding, &metadata)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil, errReusableVectorRowNotFound
		}
		return "", nil, fmt.Errorf("reuse previous segment vector row %s: %w", rowID, err)
	}
	meta, err := decodeReusableVectorRowMetadata(rowID, metadata)
	if err != nil {
		return "", nil, err
	}
	return embedding, meta, nil
}

func reusableTargetVectorRowExists(ctx context.Context, db *gorm.DB, quotedTable string, row kbVectorInsert, binding kbVectorBinding, image bool) (bool, error) {
	if quotedTable == "" {
		return false, nil
	}
	_, meta, err := readReusableVectorRowByID(ctx, db, quotedTable, row.RowID, row.Segment)
	if err != nil {
		if errors.Is(err, errReusableVectorRowNotFound) {
			return false, nil
		}
		return false, err
	}
	if image {
		if imageVectorMetadataMatchesBinding(meta, binding) {
			return true, nil
		}
		return false, fmt.Errorf("target image vector row %s in %s has mismatched binding metadata", row.RowID, quotedTable)
	}
	if textVectorMetadataMatchesBinding(meta, binding) {
		return true, nil
	}
	return false, fmt.Errorf("target text vector row %s in %s has mismatched binding metadata", row.RowID, quotedTable)
}

func readReusableVectorRowByIdentity(ctx context.Context, db *gorm.DB, quotedTable string, old kbSegmentRecord, imageFileID string) (string, map[string]any, error) {
	where := []string{"file_id = ?", "index_version = ?", "disabled = ?", "level = ?"}
	args := []any{old.KBFileID, old.IndexVersion, false, old.Level}
	identity := old.IdentityKey
	if old.ChunkIndex != nil {
		where = append(where, "chunk_index = ?")
		args = append(args, *old.ChunkIndex)
	} else if old.ChunkID != nil && *old.ChunkID != "" {
		where = append(where, "JSON_EXTRACT(meta, '$.chunk_id') = ?")
		args = append(args, *old.ChunkID)
	} else {
		return "", nil, errReusableVectorRowNotFound
	}
	if imageFileID != "" {
		where = append(where, "(JSON_EXTRACT(meta, '$.image_file_id') = ? OR JSON_EXTRACT(meta, '$.page_image_file_id') = ?)")
		args = append(args, imageFileID, imageFileID)
		identity += ":" + imageFileID
	}
	rows, err := db.WithContext(ctx).Raw(fmt.Sprintf(`SELECT embedding, meta FROM %s WHERE %s LIMIT 2`, quotedTable, strings.Join(where, " AND ")), args...).Rows()
	if err != nil {
		return "", nil, fmt.Errorf("reuse previous segment vector row identity %s: %w", identity, err)
	}
	defer rows.Close()

	var embedding string
	var metadata sql.NullString
	matched := 0
	for rows.Next() {
		matched++
		if matched > 1 {
			return "", nil, fmt.Errorf("reuse previous segment vector row identity %s: matched multiple rows", identity)
		}
		if err := rows.Scan(&embedding, &metadata); err != nil {
			return "", nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return "", nil, err
	}
	if matched == 0 {
		return "", nil, errReusableVectorRowNotFound
	}
	meta, err := decodeReusableVectorRowMetadata(identity, metadata)
	if err != nil {
		return "", nil, err
	}
	return embedding, meta, nil
}

func decodeReusableVectorRowMetadata(rowID string, metadata sql.NullString) (map[string]any, error) {
	meta := map[string]any{}
	if !metadata.Valid {
		return meta, nil
	}
	raw := strings.TrimSpace(metadata.String)
	if raw == "" {
		return meta, nil
	}
	if !json.Valid([]byte(raw)) {
		return nil, fmt.Errorf("decode previous segment vector row %s metadata: invalid JSON", rowID)
	}
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, fmt.Errorf("decode previous segment vector row %s metadata: %w", rowID, err)
	}
	return meta, nil
}

func imageVectorMetadataMatchesBinding(meta map[string]any, binding kbVectorBinding) bool {
	return firstNonEmptySegmentString(stringFromMap(meta, "image_embedding_model"), stringFromMap(meta, "embedding_model")) == binding.ImageEmbeddingModel &&
		intFromMapWithFallback(meta, "image_embedding_dimension", "embedding_dimension") == binding.ImageEmbeddingDimension &&
		firstNonEmptySegmentString(stringFromMap(meta, "image_preprocess_version"), stringFromMap(meta, "preprocess_version")) == binding.ImagePreprocessVersion &&
		firstNonEmptySegmentString(stringFromMap(meta, "image_distance_metric"), stringFromMap(meta, "distance_metric")) == binding.ImageDistanceMetric
}

func textVectorMetadataMatchesBinding(meta map[string]any, binding kbVectorBinding) bool {
	embeddingModel, hasEmbeddingModel := stringFromMapPresence(meta, "embedding_model")
	if hasEmbeddingModel {
		return embeddingModel == binding.EmbeddingModel
	}
	vectorTable, hasVectorTable := stringFromMapPresence(meta, "vector_table")
	if hasVectorTable {
		return vectorTable == binding.VectorTable
	}
	return true
}

func intFromMapWithFallback(values map[string]any, keys ...string) int {
	for _, key := range keys {
		if value := intFromMap(values, key); value != 0 {
			return value
		}
	}
	return 0
}

func embeddingBackendMetadata(meta map[string]any) map[string]any {
	raw, ok := meta["embedding_backend_metadata"].(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]any, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func validateImageVectorBinding(binding kbVectorBinding) error {
	missing := []string{}
	if binding.ImageEmbeddingModel == "" {
		missing = append(missing, "image_embedding_model")
	}
	if binding.ImageEmbeddingBackendID == "" {
		missing = append(missing, "image_embedding_backend_id")
	}
	if binding.ImageEmbeddingDimension <= 0 {
		missing = append(missing, "image_embedding_dimension")
	}
	if binding.ImagePreprocessVersion == "" {
		missing = append(missing, "image_preprocess_version")
	}
	if binding.ImageDistanceMetric == "" {
		missing = append(missing, "image_distance_metric")
	}
	if len(missing) > 0 {
		return imageVectorConfigMissingError(strings.Join(missing, ", "))
	}
	return nil
}

func (s *semanticModelService) materializeImageSegment(ctx context.Context, c *moi.Client, wsID string, binding kbVectorBinding, seg kbSegmentRecord, imageFileID string) (kbVectorInsert, error) {
	pageNumber, ok := segmentPageNumber(seg)
	if !ok {
		return kbVectorInsert{}, imageSegmentPageNumberRequiredError()
	}
	raw, err := c.Files().DownloadBytes(ctx, wsID, imageFileID)
	if err != nil {
		return kbVectorInsert{}, fmt.Errorf("download segment image file %s: %w", imageFileID, err)
	}
	embedding, rawMeta, err := createSegmentImageEmbedding(ctx, c, wsID, binding, raw)
	if err != nil {
		return kbVectorInsert{}, err
	}
	if len(embedding) != binding.ImageEmbeddingDimension {
		return kbVectorInsert{}, imageEmbeddingDimensionMismatchError()
	}
	if rawMeta != nil {
		if got := strings.TrimSpace(stringFromMap(rawMeta, "preprocess_version")); got != "" && got != binding.ImagePreprocessVersion {
			return kbVectorInsert{}, imageEmbeddingMetadataMismatchError("preprocess_version")
		}
		if got := strings.TrimSpace(stringFromMap(rawMeta, "distance_metric")); got != "" && got != binding.ImageDistanceMetric {
			return kbVectorInsert{}, imageEmbeddingMetadataMismatchError("distance_metric")
		}
	}
	meta, err := segmentImageVectorMetadata(seg, binding, imageFileID, pageNumber, rawMeta)
	if err != nil {
		return kbVectorInsert{}, err
	}
	embeddingJSON, err := json.Marshal(embedding)
	if err != nil {
		return kbVectorInsert{}, err
	}
	return kbVectorInsert{
		Segment:    seg,
		RowID:      segmentImageVectorRowID(seg, imageFileID),
		Content:    segmentEmbeddingInput(seg),
		Metadata:   meta,
		Embedding:  string(embeddingJSON),
		PageNumber: &pageNumber,
	}, nil
}

func createSegmentImageEmbedding(ctx context.Context, c *moi.Client, wsID string, binding kbVectorBinding, raw []byte) ([]float64, map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil, imageBytesEmptyError()
	}
	body := map[string]any{
		"model":              binding.ImageEmbeddingModel,
		"encoding_format":    "float",
		"preprocess_version": binding.ImagePreprocessVersion,
		"images": []map[string]any{{
			"base64":    base64.StdEncoding.EncodeToString(raw),
			"mime_type": http.DetectContentType(raw),
		}},
	}
	if backendID := numericImageEmbeddingBackendID(binding.ImageEmbeddingBackendID); backendID != 0 {
		body["backend_id"] = backendID
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}
	respBody, err := c.Embeddings(wsID).CreateRaw(ctx, payload)
	if err != nil {
		return nil, nil, fmt.Errorf("create segment image embedding: %w", err)
	}
	var resp kbSegmentImageEmbeddingResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, nil, fmt.Errorf("decode segment image embedding response: %w", err)
	}
	if len(resp.Data) != 1 {
		return nil, nil, imageEmbeddingResponseInvalidError()
	}
	if len(resp.Data[0].Embedding) == 0 {
		return nil, nil, imageEmbeddingResponseInvalidError()
	}
	return append([]float64(nil), resp.Data[0].Embedding...), resp.Metadata, nil
}

func numericImageEmbeddingBackendID(raw string) int64 {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}
	backendID, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return backendID
}

func validateVectorTableSchema(ctx context.Context, db *gorm.DB, tableName string, expectedDimension int) (kbVectorTableSchema, error) {
	schema, err := inspectVectorTableSchema(ctx, db, tableName)
	if err != nil {
		return kbVectorTableSchema{}, err
	}
	for _, required := range []string{"id", "embedding", "content", "meta", "file_id", "level", "chunk_index", "index_version", "disabled"} {
		if _, ok := schema.Columns[required]; !ok {
			return kbVectorTableSchema{}, vectorTableColumnMissingError(required)
		}
	}
	if schema.EmbeddingDimension <= 0 {
		return kbVectorTableSchema{}, vectorTableEmbeddingColumnInvalidError()
	}
	if expectedDimension > 0 && schema.EmbeddingDimension > 0 && schema.EmbeddingDimension != expectedDimension {
		return kbVectorTableSchema{}, segmentEmbeddingDimensionMismatchError()
	}
	return schema, nil
}

func inspectVectorTableSchema(ctx context.Context, db *gorm.DB, tableName string) (kbVectorTableSchema, error) {
	dbName, simpleName, err := splitSQLTableName(tableName)
	if err != nil {
		return kbVectorTableSchema{}, err
	}
	query := `SELECT COLUMN_NAME, COLUMN_TYPE FROM information_schema.COLUMNS WHERE TABLE_NAME = ?`
	args := []any{simpleName}
	if dbName == "" {
		query += ` AND TABLE_SCHEMA = DATABASE()`
	} else {
		query += ` AND TABLE_SCHEMA = ?`
		args = append(args, dbName)
	}
	rows, err := db.WithContext(ctx).Raw(query, args...).Rows()
	if err != nil {
		return kbVectorTableSchema{}, fmt.Errorf("inspect vector table schema: %w", err)
	}
	defer rows.Close()

	columns := map[string]string{}
	for rows.Next() {
		var name, typ string
		if err := rows.Scan(&name, &typ); err != nil {
			return kbVectorTableSchema{}, err
		}
		columns[strings.ToLower(name)] = strings.ToLower(typ)
	}
	if err := rows.Err(); err != nil {
		return kbVectorTableSchema{}, err
	}
	if len(columns) == 0 {
		return kbVectorTableSchema{}, vectorTableUnavailableError()
	}
	dim := parseVectorColumnDimension(columns["embedding"])
	return kbVectorTableSchema{EmbeddingDimension: dim, Columns: columnSet(columns)}, nil
}

func columnSet(columns map[string]string) map[string]struct{} {
	out := make(map[string]struct{}, len(columns))
	for name := range columns {
		out[strings.ToLower(name)] = struct{}{}
	}
	return out
}

var vectorColumnDimensionPattern = regexp.MustCompile(`(?i)vecf(32|64)\((\d+)\)`)

func parseVectorColumnDimension(columnType string) int {
	matches := vectorColumnDimensionPattern.FindStringSubmatch(columnType)
	if len(matches) != 3 {
		return 0
	}
	dim, _ := strconv.Atoi(matches[2])
	return dim
}

func splitSQLTableName(name string) (string, string, error) {
	parts := strings.Split(name, ".")
	if len(parts) == 1 {
		if !isValidSQLIdentifier(parts[0]) {
			return "", "", fmt.Errorf("invalid vector table name: %q", name)
		}
		return "", parts[0], nil
	}
	if len(parts) == 2 {
		if !isValidSQLIdentifier(parts[0]) || !isValidSQLIdentifier(parts[1]) {
			return "", "", fmt.Errorf("invalid vector table name: %q", name)
		}
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("invalid vector table name: %q", name)
}

const kbSegmentInsertBatchSize = 100

func (s *semanticModelService) commitSegmentVersion(ctx context.Context, record KnowledgeBaseSourceRecord, binding kbVectorBinding, source, versionID string, indexVersion int64, segments []kbSegmentRecord, materialized kbSegmentMaterialization, base SemanticModelSegmentMutationBase) error {
	return s.commitSegmentVersionWithTxHook(ctx, record, binding, source, versionID, indexVersion, segments, materialized, base, nil)
}

func (s *semanticModelService) commitSegmentVersionWithTxHook(ctx context.Context, record KnowledgeBaseSourceRecord, binding kbVectorBinding, source, versionID string, indexVersion int64, segments []kbSegmentRecord, materialized kbSegmentMaterialization, base SemanticModelSegmentMutationBase, txHook func(*gorm.DB) error) error {
	db := ctxutil.TenantDBFrom(ctx)
	quotedVectorTable, err := quoteQualifiedSQLIdentifier(binding.VectorTable)
	if err != nil {
		return err
	}
	quotedImageVectorTable := ""
	if binding.ImageVectorTable != "" {
		quotedImageVectorTable, err = quoteQualifiedSQLIdentifier(binding.ImageVectorTable)
		if err != nil {
			return err
		}
	}
	actor := ctxutil.UIDFrom(ctx)
	enabledCount := int64(0)
	segmentRows := make([]map[string]any, 0, len(segments))
	statRows := make([]map[string]any, 0, len(segments))
	kbFileID := ptrValue(record.KBFileID)
	for _, seg := range segments {
		if seg.Enabled {
			enabledCount++
		}
		segmentRows = append(segmentRows, map[string]any{
			"segment_id":         seg.SegmentID,
			"version_id":         versionID,
			"model_id":           record.ModelID,
			"source_id":          record.SourceID,
			"kb_file_id":         kbFileID,
			"index_version":      indexVersion,
			"level":              seg.Level,
			"chunk_index":        seg.ChunkIndex,
			"chunk_id":           seg.ChunkID,
			"identity_key":       seg.IdentityKey,
			"content":            seg.Content,
			"ocr_text":           seg.OCRText,
			"image_description":  seg.ImageDescription,
			"image_file_id":      seg.ImageFileID,
			"page_image_file_id": seg.PageImageFileID,
			"bbox":               nullableJSON(seg.BBox),
			"word_count":         seg.WordCount,
			"enabled":            seg.Enabled,
			"metadata":           nullableJSON(seg.Metadata),
			"created_by":         actor,
			"updated_by":         actor,
		})
		statRows = append(statRows, map[string]any{
			"model_id":      record.ModelID,
			"source_id":     record.SourceID,
			"kb_file_id":    kbFileID,
			"index_version": indexVersion,
			"level":         seg.Level,
			"chunk_index":   seg.ChunkIndex,
			"chunk_id":      seg.ChunkID,
			"identity_key":  seg.IdentityKey,
			"recall_count":  int64(0),
		})
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// The unique version row is the concurrency gate for this stable vector set.
		if err := tx.Exec(`INSERT INTO knowledge_base_segment_versions
			(version_id, model_id, source_id, kb_file_id, index_version, base_version_id, base_index_version, status, source, chunk_count, enabled_chunk_count, vector_table, embedding_model, image_vector_table, image_embedding_model, created_by, updated_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			versionID, record.ModelID, record.SourceID, kbFileID, indexVersion, emptyStringNil(base.BaseSegmentVersionID), baseIndexNil(base.BaseIndexVersion),
			kbSegmentStatusCommitted, source, int64(len(segments)), enabledCount, binding.VectorTable, binding.EmbeddingModel, emptyStringNil(&binding.ImageVectorTable), emptyStringNil(&binding.ImageEmbeddingModel), actor, actor).Error; err != nil {
			if isDuplicateEntryError(err) {
				return &duplicateSegmentVersionInsertError{versionID: versionID, err: err}
			}
			return fmt.Errorf("insert segment version: %w", err)
		}
		if err := insertSegmentVectorRows(tx, quotedVectorTable, materialized.TextRows, false); err != nil {
			return err
		}
		if quotedImageVectorTable != "" {
			if err := insertSegmentVectorRows(tx, quotedImageVectorTable, materialized.ImageRows, true); err != nil {
				return err
			}
		}
		if len(segmentRows) > 0 {
			// Keep the outer db.Transaction as the only transaction owner. CreateInBatches
			// would otherwise open a SAVEPOINT when len(rows) > batch size.
			batchTx := tx.Session(&gorm.Session{SkipDefaultTransaction: true})
			if err := batchTx.Table("knowledge_base_segments").CreateInBatches(segmentRows, kbSegmentInsertBatchSize).Error; err != nil {
				// Keep existing HTTP/error-code mapping; do not add a new service-layer message.
				return err
			}
			if err := batchTx.Table("knowledge_base_chunk_recall_stats").CreateInBatches(statRows, kbSegmentInsertBatchSize).Error; err != nil {
				return fmt.Errorf("insert segment recall stats: %w", err)
			}
		}
		res := tx.Exec(`UPDATE knowledge_base_sources
			SET kb_file_id = ?, segment_version_id = ?, index_version = ?, status = ?, error = NULL, updated_by = ?
			WHERE model_id = ? AND source_id = ? AND ((segment_version_id = ?) OR (segment_version_id IS NULL AND ? = '')) AND COALESCE(index_version, 0) = ?`,
			kbFileID, versionID, indexVersion, kbSourceStatusSucceeded, actor,
			record.ModelID, record.SourceID, baseVersionValue(base.BaseSegmentVersionID), baseVersionValue(base.BaseSegmentVersionID), baseIndexValue(base.BaseIndexVersion))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return segmentVersionConflictError()
		}
		if txHook != nil {
			if err := txHook(tx); err != nil {
				return err
			}
		}
		return nil
	})
}

func insertSegmentVectorRows(tx *gorm.DB, quotedTable string, rows []kbVectorInsert, includePageNumber bool) error {
	for _, row := range rows {
		seg := row.Segment
		if !includePageNumber {
			if err := tx.Exec(fmt.Sprintf(`INSERT INTO %s
				(id, embedding, content, meta, file_id, level, doc_id, section_id, chunk_index, index_version, disabled)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, quotedTable),
				row.RowID, row.Embedding, row.Content, row.Metadata, seg.KBFileID, seg.Level, nil, nil, seg.ChunkIndex, seg.IndexVersion, false).Error; err != nil {
				return fmt.Errorf("insert vector row %s: %w", row.RowID, err)
			}
			continue
		}
		if err := tx.Exec(fmt.Sprintf(`INSERT INTO %s
			(id, embedding, content, meta, file_id, page_number, level, doc_id, section_id, chunk_index, index_version, disabled)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, quotedTable),
			row.RowID, row.Embedding, row.Content, row.Metadata, seg.KBFileID, row.PageNumber, seg.Level, nil, nil, seg.ChunkIndex, seg.IndexVersion, false).Error; err != nil {
			return fmt.Errorf("insert vector row %s: %w", row.RowID, err)
		}
	}
	return nil
}

func (s *semanticModelService) getSegmentVersionRecord(ctx context.Context, modelID int64, sourceID, versionID string) (kbSegmentVersionRecord, error) {
	db := ctxutil.TenantDBFrom(ctx)
	row := db.WithContext(ctx).Raw(`SELECT version_id, model_id, source_id, kb_file_id, index_version, base_version_id, base_index_version, status, source, chunk_count, enabled_chunk_count, vector_table, embedding_model, image_vector_table, image_embedding_model, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at)
		FROM knowledge_base_segment_versions
		WHERE model_id = ? AND source_id = ? AND version_id = ?
		LIMIT 1`, modelID, sourceID, versionID).Row()
	var rec kbSegmentVersionRecord
	var baseVersion sql.NullString
	var baseIndex sql.NullInt64
	var imageTable, imageModel sql.NullString
	var createdAt, updatedAt sql.NullInt64
	if err := row.Scan(&rec.VersionID, &rec.ModelID, &rec.SourceID, &rec.KBFileID, &rec.IndexVersion, &baseVersion, &baseIndex, &rec.Status, &rec.Source, &rec.ChunkCount, &rec.EnabledChunkCount, &rec.VectorTable, &rec.EmbeddingModel, &imageTable, &imageModel, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return kbSegmentVersionRecord{}, segmentVersionNotFoundError()
		}
		return kbSegmentVersionRecord{}, err
	}
	if baseVersion.Valid {
		rec.BaseVersionID = &baseVersion.String
	}
	if baseIndex.Valid {
		rec.BaseIndexVersion = &baseIndex.Int64
	}
	if imageTable.Valid {
		rec.ImageVectorTable = &imageTable.String
	}
	if imageModel.Valid {
		rec.ImageEmbeddingModel = &imageModel.String
	}
	if createdAt.Valid {
		rec.CreatedAt = &createdAt.Int64
	}
	if updatedAt.Valid {
		rec.UpdatedAt = &updatedAt.Int64
	}
	return rec, nil
}

func chunkIdentityKey(chunkIndex *int64, chunkID *string) string {
	if chunkIndex != nil {
		return "idx:" + strconv.FormatInt(*chunkIndex, 10)
	}
	if chunkID != nil && *chunkID != "" {
		return "id:" + *chunkID
	}
	return ""
}

func segmentIdentityKey(seg kbSegmentRecord) string {
	identityKey := chunkIdentityKey(seg.ChunkIndex, seg.ChunkID)
	if identityKey == "" {
		return ""
	}
	if strings.HasPrefix(seg.IdentityKey, kbSegmentImageIdentityPrefix) {
		return imageSegmentIdentityKey(identityKey)
	}
	return identityKey
}

func imageSegmentIdentityKey(identityKey string) string {
	if identityKey == "" || strings.HasPrefix(identityKey, kbSegmentImageIdentityPrefix) {
		return identityKey
	}
	return kbSegmentImageIdentityPrefix + identityKey
}

func segmentWordCount(seg kbSegmentRecord) int64 {
	return int64(utf8.RuneCountInString(segmentEmbeddingInput(seg)))
}

func segmentEmbeddingInput(seg kbSegmentRecord) string {
	parts := make([]string, 0, 3)
	if seg.Content != nil && *seg.Content != "" {
		parts = append(parts, *seg.Content)
	}
	if seg.OCRText != nil && *seg.OCRText != "" {
		parts = append(parts, *seg.OCRText)
	}
	if seg.ImageDescription != nil && *seg.ImageDescription != "" {
		parts = append(parts, *seg.ImageDescription)
	}
	return strings.Join(parts, "\n")
}

func segmentVectorRowID(seg kbSegmentRecord) string {
	return stableID("kbsegrow", seg.KBFileID, seg.IndexVersion, seg.Level, seg.IdentityKey)
}

func segmentImageVectorRowID(seg kbSegmentRecord, imageFileID string) string {
	return stableID("kbimgsegrow", seg.KBFileID, seg.IndexVersion, seg.Level, seg.IdentityKey, imageFileID)
}

func segmentVectorMetadata(seg kbSegmentRecord) (string, error) {
	meta := map[string]any{
		"source_id":          seg.SourceID,
		"segment_version_id": seg.VersionID,
		"index_version":      seg.IndexVersion,
		"level":              seg.Level,
		"chunk_index":        seg.ChunkIndex,
		"chunk_id":           seg.ChunkID,
		"identity_key":       seg.IdentityKey,
		"enabled":            seg.Enabled,
		"ocr_text":           seg.OCRText,
		"image_description":  seg.ImageDescription,
		"image_file_id":      seg.ImageFileID,
		"page_image_file_id": seg.PageImageFileID,
	}
	if len(seg.BBox) > 0 && json.Valid(seg.BBox) {
		var bbox any
		if err := json.Unmarshal(seg.BBox, &bbox); err != nil {
			return "", err
		}
		meta["bbox"] = bbox
	}
	if len(seg.Metadata) > 0 && json.Valid(seg.Metadata) {
		var existing map[string]any
		if err := json.Unmarshal(seg.Metadata, &existing); err == nil {
			for k, v := range existing {
				if _, exists := meta[k]; !exists {
					meta[k] = v
				}
			}
		}
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func segmentTextVectorMetadata(seg kbSegmentRecord, binding kbVectorBinding) (string, error) {
	metaText, err := segmentVectorMetadata(seg)
	if err != nil {
		return "", err
	}
	meta := map[string]any{}
	if err := json.Unmarshal([]byte(metaText), &meta); err != nil {
		return "", err
	}
	meta["vector_table"] = binding.VectorTable
	meta["embedding_model"] = binding.EmbeddingModel
	b, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func segmentImageVectorMetadata(seg kbSegmentRecord, binding kbVectorBinding, imageFileID string, pageNumber int, rawMeta map[string]any) (string, error) {
	metaText, err := segmentVectorMetadata(seg)
	if err != nil {
		return "", err
	}
	meta := map[string]any{}
	if err := json.Unmarshal([]byte(metaText), &meta); err != nil {
		return "", err
	}
	meta["modality"] = "image"
	meta["asset_kind"] = "document_visual"
	if stringFromMap(meta, "scope") == "" {
		meta["scope"] = "page"
	}
	meta["level"] = seg.Level
	meta["source_file_id"] = seg.KBFileID
	meta["file_id"] = seg.KBFileID
	meta["page_number"] = pageNumber
	meta["image_file_id"] = seg.ImageFileID
	meta["page_image_file_id"] = seg.PageImageFileID
	meta["embedding_model"] = binding.ImageEmbeddingModel
	meta["image_embedding_model"] = binding.ImageEmbeddingModel
	meta["embedding_backend_id"] = binding.ImageEmbeddingBackendID
	meta["image_embedding_backend_id"] = binding.ImageEmbeddingBackendID
	meta["embedding_dimension"] = binding.ImageEmbeddingDimension
	meta["image_embedding_dimension"] = binding.ImageEmbeddingDimension
	meta["embedding_source"] = "real"
	meta["image_embedding_source"] = "real"
	meta["preprocess_version"] = binding.ImagePreprocessVersion
	meta["image_preprocess_version"] = binding.ImagePreprocessVersion
	meta["distance_metric"] = binding.ImageDistanceMetric
	meta["image_distance_metric"] = binding.ImageDistanceMetric
	if rawMeta != nil {
		meta["embedding_backend_metadata"] = rawMeta
	}
	b, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func segmentImageFileID(seg kbSegmentRecord) string {
	if seg.ImageFileID != nil && *seg.ImageFileID != "" {
		return *seg.ImageFileID
	}
	if seg.PageImageFileID != nil && *seg.PageImageFileID != "" {
		return *seg.PageImageFileID
	}
	return ""
}

func segmentPageNumber(seg kbSegmentRecord) (int, bool) {
	meta := map[string]any{}
	if len(seg.Metadata) > 0 && json.Valid(seg.Metadata) {
		_ = json.Unmarshal(seg.Metadata, &meta)
	}
	if page := intFromMap(meta, "page_number"); page > 0 {
		return page, true
	}
	if page := intFromMap(meta, "page_num"); page > 0 {
		return page, true
	}
	return 0, false
}

func nullableJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}

func emptyStringNil(value *string) any {
	if value == nil || *value == "" {
		return nil
	}
	return *value
}

func baseIndexNil(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func baseVersionValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func baseIndexValue(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func stringFromMap(values map[string]any, key string) string {
	value, _ := stringFromMapPresence(values, key)
	return value
}

func stringFromMapPresence(values map[string]any, key string) (string, bool) {
	switch v := values[key].(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	default:
		return "", false
	}
}

func intFromMap(values map[string]any, key string) int {
	switch v := values[key].(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := strconv.Atoi(v.String())
		return i
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(v))
		return i
	default:
		return 0
	}
}
