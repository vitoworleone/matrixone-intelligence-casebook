package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/matrixflow/moi-core/agent-tools/knowledge"
)

const defaultVisualTopK = 8

type VisualSearchBackend interface {
	ReadImageFile(ctx context.Context, scope knowledge.WorkspaceScope, fileID string) ([]byte, string, error)
	CreateImageEmbedding(ctx context.Context, workspaceID string, req VisualImageEmbeddingRequest) ([]float64, map[string]any, error)
	ResolveVisualScopeFileIDs(ctx context.Context, scope knowledge.WorkspaceScope) ([]string, bool, error)
}

type VisualImageEmbeddingRequest struct {
	Model             string
	BackendID         string
	PreprocessVersion string
	Raw               []byte
	MimeType          string
}

type visualSearchService struct {
	sql     knowledge.SQLExecutor
	backend VisualSearchBackend
}

func NewSearchVisualImage(deps Deps) knowledge.SearchVisualImage {
	return &visualSearchService{
		sql:     deps.SQLExecutor,
		backend: deps.VisualSearchBackend,
	}
}

func (s *visualSearchService) Execute(ctx context.Context, req knowledge.SearchVisualImageRequest) (*knowledge.SearchVisualImageResponse, error) {
	if s == nil || s.sql == nil || s.backend == nil {
		return nil, fmt.Errorf("visual search: searcher is not configured")
	}
	scope := mergeScopeWithContext(ctx, req.Scope)
	queryVisualFileID := strings.TrimSpace(req.QueryVisualFileID)
	queryText := strings.TrimSpace(req.QueryText)
	if queryVisualFileID == "" && queryText == "" {
		return nil, fmt.Errorf("visual search: query_text or query_visual_file_id is required")
	}
	if strings.TrimSpace(scope.WorkspaceID) == "" {
		return nil, fmt.Errorf("visual search: workspace_id is required")
	}
	rankingProfile := visualRankingProfile(req.RankingProfile)
	if !visualRankingProfileValid(rankingProfile) {
		return nil, fmt.Errorf("visual search: unsupported ranking_profile %q", req.RankingProfile)
	}
	visualQueries, err := visualImageQueries(scope)
	if err != nil {
		return nil, err
	}
	if len(visualQueries) == 0 {
		return &knowledge.SearchVisualImageResponse{
			Results: []knowledge.VisualSearchHit{},
			Count:   0,
			Mode:    "image",
			Metadata: map[string]any{
				"query_text": queryText,
			},
		}, nil
	}
	var rawMeta map[string]any
	var imageHits []knowledge.VisualSearchHit
	if queryVisualFileID != "" {
		raw, mimeType, err := s.backend.ReadImageFile(ctx, scope, queryVisualFileID)
		if err != nil {
			return nil, fmt.Errorf("visual search: read query visual file_id=%s: %w", queryVisualFileID, err)
		}
		for _, query := range visualQueries {
			queryEmbedding, meta, err := s.backend.CreateImageEmbedding(ctx, query.scope.WorkspaceID, VisualImageEmbeddingRequest{
				Model:             query.cfg.Model,
				BackendID:         query.cfg.BackendID,
				PreprocessVersion: query.cfg.PreprocessVersion,
				Raw:               raw,
				MimeType:          mimeType,
			})
			if err != nil {
				return nil, err
			}
			if len(queryEmbedding) != query.cfg.Dimension {
				return nil, fmt.Errorf("visual search: query visual embedding dimension mismatch: got %d want %d", len(queryEmbedding), query.cfg.Dimension)
			}
			rawMeta = meta
			hits, err := s.searchImageVector(ctx, query.scope, query.cfg, queryEmbedding, req.TopK, rankingProfile)
			if err != nil {
				return nil, err
			}
			stampVisualSearchHitsOwner(hits, query)
			imageHits = append(imageHits, hits...)
		}
	}
	var textHits []knowledge.VisualSearchHit
	if queryText != "" {
		for _, query := range visualQueries {
			hits, err := s.searchTextVisual(ctx, query.scope, query.cfg, queryText, req.TopK, rankingProfile)
			if err != nil {
				return nil, fmt.Errorf("visual search: text side search: %w", err)
			}
			stampVisualSearchHitsOwner(hits, query)
			textHits = append(textHits, hits...)
		}
	}
	mode := "image"
	results := imageHits
	if queryText != "" && queryVisualFileID != "" {
		results = fuseVisualResults(textHits, imageHits, normalizeVisualTopK(req.TopK), rankingProfile)
		mode = "hybrid"
	} else if queryText != "" {
		results = textHits
		mode = "text"
	}
	if rankingProfile == knowledge.VisualSearchRankingProfileVisualObjectFirst {
		results = rerankVisualObjectFirstResults(results, visualTextQueryFragments(queryText), normalizeVisualTopK(req.TopK))
	} else if rankingProfile == knowledge.VisualSearchRankingProfileTextRegionFirst {
		results = rerankVisualTextRegionFirstResults(results, visualTextQueryFragments(queryText), normalizeVisualTopK(req.TopK))
	} else if queryVisualFileID != "" && queryText == "" {
		sortVisualResultsByScoreDesc(results)
	}
	results = limitVisualResults(results, normalizeVisualTopK(req.TopK))
	metadata := visualSearchMetadata(visualQueries)
	if queryText != "" {
		metadata["query_text"] = queryText
	}
	if queryVisualFileID != "" {
		metadata["query_visual_file_id"] = queryVisualFileID
	}
	if rankingProfile != visualRankingProfileDocument {
		metadata["ranking_profile"] = rankingProfile
	}
	if rawMeta != nil {
		metadata["image_embedding_backend_metadata"] = rawMeta
	}
	return &knowledge.SearchVisualImageResponse{
		Results:  results,
		Count:    len(results),
		Mode:     mode,
		Metadata: metadata,
	}, nil
}

type visualImageQuery struct {
	scope           knowledge.WorkspaceScope
	cfg             visualImageIndexConfig
	semanticModelID int64
}

type visualImageIndexConfig struct {
	Table             string
	Model             string
	BackendID         string
	Dimension         int
	PreprocessVersion string
	DistanceMetric    string
}

func visualImageConfig(scope knowledge.WorkspaceScope) (visualImageIndexConfig, error) {
	cfg := visualImageIndexConfig{
		Table:             strings.TrimSpace(scope.ImageVectorTable),
		Model:             strings.TrimSpace(scope.ImageEmbeddingModel),
		BackendID:         strings.TrimSpace(scope.ImageEmbeddingBackendID),
		Dimension:         scope.ImageEmbeddingDimension,
		PreprocessVersion: strings.TrimSpace(scope.ImagePreprocessVersion),
		DistanceMetric:    strings.TrimSpace(scope.ImageDistanceMetric),
	}
	if cfg.Table == "" {
		cfg.Table = knowledge.UniqueRAGSourceValue(scope.RAGSources, func(source knowledge.RAGSource) string { return source.ImageVectorTable })
	}
	if cfg.Model == "" {
		cfg.Model = knowledge.UniqueRAGSourceValue(scope.RAGSources, func(source knowledge.RAGSource) string { return source.ImageEmbeddingModel })
	}
	if cfg.BackendID == "" {
		cfg.BackendID = knowledge.UniqueRAGSourceValue(scope.RAGSources, func(source knowledge.RAGSource) string { return source.ImageEmbeddingBackendID })
	}
	if cfg.Dimension <= 0 {
		cfg.Dimension = knowledge.UniqueRAGSourceIntValue(scope.RAGSources, func(source knowledge.RAGSource) int { return source.ImageEmbeddingDimension })
	}
	if cfg.PreprocessVersion == "" {
		cfg.PreprocessVersion = knowledge.UniqueRAGSourceValue(scope.RAGSources, func(source knowledge.RAGSource) string { return source.ImagePreprocessVersion })
	}
	if cfg.DistanceMetric == "" {
		cfg.DistanceMetric = knowledge.UniqueRAGSourceValue(scope.RAGSources, func(source knowledge.RAGSource) string { return source.ImageDistanceMetric })
	}
	missing := []string{}
	if cfg.Table == "" {
		missing = append(missing, "image_vector_table")
	}
	if cfg.Model == "" {
		missing = append(missing, "image_embedding_model")
	}
	if cfg.Dimension <= 0 {
		missing = append(missing, "image_embedding_dimension")
	}
	if cfg.PreprocessVersion == "" {
		missing = append(missing, "preprocess_version")
	}
	if cfg.DistanceMetric == "" {
		missing = append(missing, "distance_metric")
	}
	if len(missing) > 0 {
		return visualImageIndexConfig{}, fmt.Errorf("visual search: selected knowledge scope has no complete image index config; missing %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

func visualImageQueries(scope knowledge.WorkspaceScope) ([]visualImageQuery, error) {
	if len(scope.RAGSources) == 0 {
		cfg, err := visualImageConfig(scope)
		if err != nil {
			return nil, err
		}
		return []visualImageQuery{{
			scope:           visualImageQueryScope(scope, cfg),
			cfg:             cfg,
			semanticModelID: uniquePositiveSemanticModelID(scope.SemanticModelIDs),
		}}, nil
	}
	// Multi-KB visual scopes can have independent KB-owned image indexes. Query
	// each complete source with its own table/model instead of collapsing to a
	// single top-level image_vector_table.
	queries := make([]visualImageQuery, 0, len(scope.RAGSources))
	for _, source := range scope.RAGSources {
		if !visualRAGSourceHasCompleteImageConfig(source) {
			continue
		}
		nextScope := scope
		nextScope.DBName = strings.TrimSpace(source.DBName)
		nextScope.FileIDs = nil
		nextScope.VolumeID = ""
		nextScope.RAGSources = []knowledge.RAGSource{source}
		nextScope.ImageVectorTable = strings.TrimSpace(source.ImageVectorTable)
		nextScope.ImageEmbeddingModel = strings.TrimSpace(source.ImageEmbeddingModel)
		nextScope.ImageEmbeddingBackendID = strings.TrimSpace(source.ImageEmbeddingBackendID)
		nextScope.ImageEmbeddingDimension = source.ImageEmbeddingDimension
		nextScope.ImagePreprocessVersion = strings.TrimSpace(source.ImagePreprocessVersion)
		nextScope.ImageDistanceMetric = strings.TrimSpace(source.ImageDistanceMetric)
		cfg, err := visualImageConfig(nextScope)
		if err != nil {
			return nil, err
		}
		queries = append(queries, visualImageQuery{
			scope:           visualImageQueryScope(nextScope, cfg),
			cfg:             cfg,
			semanticModelID: source.SemanticModelID,
		})
	}
	if len(queries) > 0 {
		return queries, nil
	}
	cfg, err := visualImageConfig(scope)
	if err != nil {
		return nil, err
	}
	return []visualImageQuery{{
		scope:           visualImageQueryScope(scope, cfg),
		cfg:             cfg,
		semanticModelID: uniqueVisualRAGSourceSemanticModelID(scope.RAGSources),
	}}, nil
}

func uniqueVisualRAGSourceSemanticModelID(sources []knowledge.RAGSource) int64 {
	values := make([]int64, 0, len(sources))
	for _, source := range sources {
		values = append(values, source.SemanticModelID)
	}
	return uniquePositiveSemanticModelID(values)
}

func stampVisualSearchHitsOwner(hits []knowledge.VisualSearchHit, query visualImageQuery) {
	for index := range hits {
		if query.semanticModelID > 0 {
			hits[index].SemanticModelID = query.semanticModelID
		}
		hits[index].SourceRowID = sourceRowIDForScopeFile(query.scope, hits[index].SourceFileID)
	}
}

func visualRAGSourceHasCompleteImageConfig(source knowledge.RAGSource) bool {
	return strings.TrimSpace(source.ImageVectorTable) != "" &&
		strings.TrimSpace(source.ImageEmbeddingModel) != "" &&
		source.ImageEmbeddingDimension > 0 &&
		strings.TrimSpace(source.ImagePreprocessVersion) != "" &&
		strings.TrimSpace(source.ImageDistanceMetric) != ""
}

func visualImageQueryScope(scope knowledge.WorkspaceScope, cfg visualImageIndexConfig) knowledge.WorkspaceScope {
	if len(scope.RAGSources) == 0 {
		return scope
	}
	imageTable := strings.TrimSpace(cfg.Table)
	scope.DBName = knowledge.UniqueRAGSourceValue(scope.RAGSources, func(source knowledge.RAGSource) string {
		sourceTable := strings.TrimSpace(source.ImageVectorTable)
		if sourceTable == "" {
			return ""
		}
		if imageTable != "" && sourceTable != imageTable {
			return ""
		}
		return source.DBName
	})
	return scope
}

func limitVisualResults(hits []knowledge.VisualSearchHit, topK int) []knowledge.VisualSearchHit {
	topK = normalizeVisualTopK(topK)
	if len(hits) <= topK {
		return hits
	}
	return hits[:topK]
}

func sortVisualResultsByScoreDesc(hits []knowledge.VisualSearchHit) {
	sort.SliceStable(hits, func(i, j int) bool {
		return hits[i].Score > hits[j].Score
	})
}

func visualSearchMetadata(queries []visualImageQuery) map[string]any {
	metadata := map[string]any{}
	if len(queries) == 0 {
		return metadata
	}
	cfg := queries[0].cfg
	metadata["image_vector_table"] = cfg.Table
	metadata["image_embedding_model"] = cfg.Model
	metadata["image_embedding_backend_id"] = cfg.BackendID
	metadata["image_embedding_dimension"] = cfg.Dimension
	metadata["image_preprocess_version"] = cfg.PreprocessVersion
	metadata["image_distance_metric"] = cfg.DistanceMetric
	metadata["query_visual_embedding_model"] = cfg.Model
	if len(queries) > 1 {
		tables := make([]string, 0, len(queries))
		models := make([]string, 0, len(queries))
		for _, query := range queries {
			tables = append(tables, query.cfg.Table)
			models = append(models, query.cfg.Model)
		}
		metadata["image_vector_tables"] = compactVisualStrings(tables)
		metadata["image_embedding_models"] = compactVisualStrings(models)
	}
	return metadata
}

func (s *visualSearchService) searchImageVector(ctx context.Context, scope knowledge.WorkspaceScope, cfg visualImageIndexConfig, queryEmbedding []float64, topK int, rankingProfile string) ([]knowledge.VisualSearchHit, error) {
	rankingProfile = visualRankingProfile(rankingProfile)
	vectorLiteral, err := visualVectorLiteral(queryEmbedding)
	if err != nil {
		return nil, err
	}
	scopeFileIDs, scoped, err := s.backend.ResolveVisualScopeFileIDs(ctx, scope)
	if err != nil {
		return nil, err
	}
	if scoped && len(scopeFileIDs) == 0 {
		return nil, nil
	}
	scoreExpr := fmt.Sprintf("cosine_similarity(embedding, '%s')", ragEscapeSQLString(vectorLiteral))
	where := visualBaseWhere(cfg)
	if visualRankingProfileUsesObjectScope(rankingProfile) {
		where = append(where, visualObjectScopeWhere())
	}
	where = append(where, ragCurrentVersionWhere(scope, scopeFileIDs)...)
	limit := normalizeVisualTopK(topK) * 3
	sqlText := fmt.Sprintf(`
SELECT
  id,
  content,
  meta,
  file_id,
  page_number,
  %s AS score
FROM %s
WHERE %s
ORDER BY score DESC
LIMIT %d`, scoreExpr, quoteDescribeSchemaIdentifier(cfg.Table), strings.Join(where, "\n  AND "), limit)
	ctx = knowledge.ContextWithScope(ctx, scope)
	exec, err := s.sql.ExecuteSQL(ctx, strings.TrimSpace(scope.DBName), sqlText)
	if err != nil {
		return nil, fmt.Errorf("image vector query: %w", err)
	}
	if exec == nil {
		return nil, fmt.Errorf("image vector query returned nil result")
	}
	hits := make([]knowledge.VisualSearchHit, 0, normalizeVisualTopK(topK))
	for _, row := range exec.Rows {
		hit, ok := visualHitFromRow(exec.Columns, row, "image")
		if !ok {
			continue
		}
		hit.ScoreParts = map[string]any{"image": hit.Score}
		hit.SourceTags = visualSourceTagsForFile(scope, hit.SourceFileID)
		if len(hit.SourceTags) > 0 {
			if hit.Metadata == nil {
				hit.Metadata = map[string]any{}
			}
			hit.Metadata["source_tags"] = hit.SourceTags
		}
		if visualRankingProfileUsesObjectScope(rankingProfile) {
			hit.ScoreParts["ranking_profile"] = rankingProfile
			hit.ScoreParts["scope"] = hit.Scope
			hit.ScoreParts["fusion_key_type"] = "visual_object"
			hit.ScoreParts["fusion_key"] = visualObjectFusionKey(hit)
			if rankingProfile == knowledge.VisualSearchRankingProfileTextRegionFirst {
				hit.Reason = "object-level image candidate for visual text/table region"
			} else {
				hit.Reason = "object-level image match on visual object embedding"
			}
		} else {
			hit.Reason = "image match on visual page or object embedding"
		}
		hits = append(hits, hit)
		if len(hits) >= normalizeVisualTopK(topK) {
			break
		}
	}
	return hits, nil
}

func (s *visualSearchService) searchTextVisual(ctx context.Context, scope knowledge.WorkspaceScope, cfg visualImageIndexConfig, queryText string, topK int, rankingProfile string) ([]knowledge.VisualSearchHit, error) {
	rankingProfile = visualRankingProfile(rankingProfile)
	scopeFileIDs, scoped, err := s.backend.ResolveVisualScopeFileIDs(ctx, scope)
	if err != nil {
		return nil, err
	}
	if scoped && len(scopeFileIDs) == 0 {
		return nil, nil
	}
	where := visualBaseWhere(cfg)
	queryFragments := visualTextQueryFragments(queryText)
	if len(queryFragments) == 0 {
		return nil, nil
	}
	textWhere := make([]string, 0, len(queryFragments))
	for _, fragment := range queryFragments {
		textWhere = append(textWhere, fmt.Sprintf("content LIKE '%%%s%%'", escapeLike(fragment)))
	}
	scoreExpr := "1.0"
	orderExpr := "id ASC"
	if visualRankingProfileUsesObjectScope(rankingProfile) {
		scoreExpr = visualTextMatchScoreExpr(queryFragments)
		orderExpr = "score DESC, id ASC"
	}
	where = append(where, "("+strings.Join(textWhere, " OR ")+")")
	if visualRankingProfileUsesObjectScope(rankingProfile) {
		where = append(where, visualObjectScopeWhere())
	}
	where = append(where, ragCurrentVersionWhere(scope, scopeFileIDs)...)
	limit := normalizeVisualTopK(topK) * 3
	sqlText := fmt.Sprintf(`
SELECT
  id,
  content,
  meta,
  file_id,
  page_number,
  %s AS score
FROM %s
WHERE %s
ORDER BY %s
LIMIT %d`, scoreExpr, quoteDescribeSchemaIdentifier(cfg.Table), strings.Join(where, "\n  AND "), orderExpr, limit)
	ctx = knowledge.ContextWithScope(ctx, scope)
	exec, err := s.sql.ExecuteSQL(ctx, strings.TrimSpace(scope.DBName), sqlText)
	if err != nil {
		return nil, fmt.Errorf("image vector text query: %w", err)
	}
	if exec == nil {
		return nil, fmt.Errorf("image vector text query returned nil result")
	}
	hits := make([]knowledge.VisualSearchHit, 0, normalizeVisualTopK(topK))
	for _, row := range exec.Rows {
		hit, ok := visualHitFromRow(exec.Columns, row, "text")
		if !ok {
			continue
		}
		hit.ScoreParts = map[string]any{"text": hit.Score}
		hit.SourceTags = visualSourceTagsForFile(scope, hit.SourceFileID)
		if len(hit.SourceTags) > 0 {
			if hit.Metadata == nil {
				hit.Metadata = map[string]any{}
			}
			hit.Metadata["source_tags"] = hit.SourceTags
		}
		if visualRankingProfileUsesObjectScope(rankingProfile) {
			hit.ScoreParts["ranking_profile"] = rankingProfile
			hit.ScoreParts["scope"] = hit.Scope
			hit.ScoreParts["fusion_key_type"] = "visual_object"
			hit.ScoreParts["fusion_key"] = visualObjectFusionKey(hit)
			if rankingProfile == knowledge.VisualSearchRankingProfileTextRegionFirst {
				hit.Reason = "object-level text/table region match on visual object context"
			} else {
				hit.Reason = "object-level text match on visual object context"
			}
		} else {
			hit.Reason = "text match on visual page or object context"
		}
		hits = append(hits, hit)
		if len(hits) >= normalizeVisualTopK(topK) {
			break
		}
	}
	return hits, nil
}

func visualTextMatchScoreExpr(queryFragments []string) string {
	parts := make([]string, 0, len(queryFragments))
	for _, fragment := range queryFragments {
		parts = append(parts, fmt.Sprintf("IF(content LIKE '%%%s%%', 1, 0)", escapeLike(fragment)))
	}
	if len(parts) == 0 {
		return "0"
	}
	return strings.Join(parts, " + ")
}

func visualBaseWhere(cfg visualImageIndexConfig) []string {
	return []string{
		"embedding IS NOT NULL",
		"COALESCE(disabled, 0) = 0",
		"JSON_UNQUOTE(JSON_EXTRACT(meta, '$.modality')) = 'image'",
		"JSON_UNQUOTE(JSON_EXTRACT(meta, '$.asset_kind')) = 'document_visual'",
		fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(meta, '$.embedding_model')) = '%s'", ragEscapeSQLString(cfg.Model)),
		fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(meta, '$.preprocess_version')) = '%s'", ragEscapeSQLString(cfg.PreprocessVersion)),
		fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(meta, '$.distance_metric')) = '%s'", ragEscapeSQLString(cfg.DistanceMetric)),
		"JSON_UNQUOTE(JSON_EXTRACT(meta, '$.embedding_source')) = 'real'",
	}
}

func visualObjectScopeWhere() string {
	return `(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.scope')) = 'visual_object' OR (COALESCE(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.object_kind')), '') <> '' AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.object_kind')) <> 'page'))`
}

func visualTextQueryFragments(queryText string) []string {
	return strings.Fields(queryText)
}

func visualHitFromRow(columns []string, row []any, mode string) (knowledge.VisualSearchHit, bool) {
	record := executionRowRecord(columns, row)
	meta, _ := parseMetadataAny(record["meta"])
	hit := knowledge.VisualSearchHit{
		ObjectID:        stringFromMetadataAny(meta, "object_id"),
		ObjectKind:      stringFromMetadataAny(meta, "object_kind"),
		Scope:           stringFromMetadataAny(meta, "scope"),
		SourceFileID:    firstNonEmpty(stringFromMetadataAny(meta, "source_file_id"), stringFromMetadataAny(meta, "file_id"), stringFromRecordAny(record, "file_id")),
		SourceFileName:  stringFromMetadataAny(meta, "source_file_name"),
		PageNumber:      firstPositiveInt(intFromRecordAny(record, "page_number"), intFromMetadataAny(meta, "page_number")),
		BBox:            floatSliceFromMetadataAny(meta, "bbox"),
		ImageFileID:     stringFromMetadataAny(meta, "image_file_id"),
		PageImageFileID: stringFromMetadataAny(meta, "page_image_file_id"),
		Content:         stringFromRecordAny(record, "content"),
		Score:           floatFromRecordAny(record, "score"),
		Metadata:        meta,
		Reason:          mode + " match on visual page or object context",
	}
	if hit.ObjectID == "" && hit.ImageFileID == "" && hit.PageImageFileID == "" {
		return knowledge.VisualSearchHit{}, false
	}
	if hit.SourceFileID == "" || hit.PageNumber <= 0 {
		return knowledge.VisualSearchHit{}, false
	}
	return hit, true
}

func visualSourceTagsForFile(scope knowledge.WorkspaceScope, fileID string) []string {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return nil
	}
	out := make([]string, 0)
	for _, source := range scope.RAGSources {
		out = append(out, source.SourceTagsByFileID[fileID]...)
	}
	return out
}

func compactVisualStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func visualVectorLiteral(vector []float64) (string, error) {
	for _, value := range vector {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return "", fmt.Errorf("non-finite vector value %v", value)
		}
	}
	raw, err := json.Marshal(vector)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func normalizeVisualTopK(topK int) int {
	if topK <= 0 {
		return defaultVisualTopK
	}
	return topK
}

func escapeLike(value string) string {
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "%", "\\%")
	escaped = strings.ReplaceAll(escaped, "_", "\\_")
	return ragEscapeSQLString(escaped)
}

func executionRowRecord(columns []string, row []any) map[string]any {
	out := make(map[string]any, len(columns))
	for idx, col := range columns {
		if idx >= len(row) {
			break
		}
		out[normalizeColumnName(col)] = row[idx]
	}
	return out
}

func normalizeColumnName(col string) string {
	return strings.ToLower(strings.TrimSpace(col))
}

func parseMetadataAny(raw any) (map[string]any, error) {
	switch value := raw.(type) {
	case nil:
		return nil, nil
	case map[string]any:
		return value, nil
	case string:
		return parseMetadataText(value)
	case []byte:
		return parseMetadataText(string(value))
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		return parseMetadataText(string(data))
	}
}

func parseMetadataText(text string) (map[string]any, error) {
	text = strings.TrimSpace(text)
	if text == "" || text == "null" {
		return nil, nil
	}
	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()
	var meta map[string]any
	if err := decoder.Decode(&meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func stringFromRecordAny(record map[string]any, key string) string {
	return valueString(record[normalizeColumnName(key)])
}

func stringFromMetadataAny(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	return valueString(meta[key])
}

func intFromRecordAny(record map[string]any, key string) int {
	return intFromMeta(record, normalizeColumnName(key))
}

func intFromMetadataAny(meta map[string]any, key string) int {
	return intFromMeta(meta, key)
}

func optionalIntFromVisualMetadata(meta map[string]any, key string) *int {
	value, ok := intFromVisualMetadata(meta, key)
	if !ok {
		return nil
	}
	return &value
}

func int64FromVisualMetadata(meta map[string]any, key string) int64 {
	value, _ := int64FromVisualMetadataOK(meta, key)
	return value
}

func intFromVisualMetadata(meta map[string]any, key string) (int, bool) {
	value, ok := int64FromVisualMetadataOK(meta, key)
	if !ok {
		return 0, false
	}
	return int(value), true
}

func int64FromVisualMetadataOK(meta map[string]any, key string) (int64, bool) {
	if meta == nil {
		return 0, false
	}
	switch value := meta[key].(type) {
	case int:
		return int64(value), true
	case int8:
		return int64(value), true
	case int16:
		return int64(value), true
	case int32:
		return int64(value), true
	case int64:
		return value, true
	case uint:
		return int64(value), true
	case uint8:
		return int64(value), true
	case uint16:
		return int64(value), true
	case uint32:
		return int64(value), true
	case uint64:
		return int64(value), true
	case float64:
		return int64(value), true
	case json.Number:
		i, err := strconv.ParseInt(value.String(), 10, 64)
		return i, err == nil
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		return i, err == nil
	default:
		return 0, false
	}
}

func floatFromRecordAny(record map[string]any, key string) float64 {
	value, _ := floatFromAny(record[normalizeColumnName(key)])
	return value
}

func floatSliceFromMetadataAny(meta map[string]any, key string) []float64 {
	if meta == nil {
		return nil
	}
	switch values := meta[key].(type) {
	case []float64:
		return append([]float64(nil), values...)
	case []any:
		out := make([]float64, 0, len(values))
		for _, value := range values {
			n, ok := floatFromAny(value)
			if !ok {
				return nil
			}
			out = append(out, n)
		}
		return out
	default:
		return nil
	}
}

func floatFromAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case json.Number:
		n, err := strconv.ParseFloat(typed.String(), 64)
		return n, err == nil
	}
	return 0, false
}

func mergeScopeWithContext(ctx context.Context, scope knowledge.WorkspaceScope) knowledge.WorkspaceScope {
	ctxScope := knowledge.ScopeFromContext(ctx)
	if ctxScope.WorkspaceID != "" {
		scope.WorkspaceID = ctxScope.WorkspaceID
	}
	if ctxScope.UserID != "" {
		scope.UserID = ctxScope.UserID
	}
	if ctxScope.SessionID != "" {
		scope.SessionID = ctxScope.SessionID
	}
	if ctxScope.DBName != "" {
		scope.DBName = ctxScope.DBName
	}
	if len(ctxScope.Tables) > 0 {
		scope.Tables = append([]string(nil), ctxScope.Tables...)
	}
	if len(ctxScope.SemanticModelIDs) > 0 {
		scope.SemanticModelIDs = append([]int64(nil), ctxScope.SemanticModelIDs...)
	}
	if len(ctxScope.FileIDs) > 0 {
		scope.FileIDs = append([]string(nil), ctxScope.FileIDs...)
	}
	if ctxScope.VolumeID != "" {
		scope.VolumeID = ctxScope.VolumeID
	}
	if ctxScope.VectorTable != "" {
		scope.VectorTable = ctxScope.VectorTable
	}
	if ctxScope.EmbeddingModel != "" {
		scope.EmbeddingModel = ctxScope.EmbeddingModel
	}
	if ctxScope.ImageVectorTable != "" {
		scope.ImageVectorTable = ctxScope.ImageVectorTable
	}
	if ctxScope.ImageEmbeddingModel != "" {
		scope.ImageEmbeddingModel = ctxScope.ImageEmbeddingModel
	}
	if ctxScope.ImageEmbeddingBackendID != "" {
		scope.ImageEmbeddingBackendID = ctxScope.ImageEmbeddingBackendID
	}
	if ctxScope.ImageEmbeddingDimension > 0 {
		scope.ImageEmbeddingDimension = ctxScope.ImageEmbeddingDimension
	}
	if ctxScope.ImagePreprocessVersion != "" {
		scope.ImagePreprocessVersion = ctxScope.ImagePreprocessVersion
	}
	if ctxScope.ImageDistanceMetric != "" {
		scope.ImageDistanceMetric = ctxScope.ImageDistanceMetric
	}
	if len(ctxScope.RAGSources) > 0 {
		scope.RAGSources = append([]knowledge.RAGSource(nil), ctxScope.RAGSources...)
	}
	return scope
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func valueString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case []byte:
		return string(typed)
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func intFromMeta(meta map[string]any, key string) int {
	if meta == nil {
		return 0
	}
	switch value := meta[key].(type) {
	case int:
		return value
	case int8:
		return int(value)
	case int16:
		return int(value)
	case int32:
		return int(value)
	case int64:
		return int(value)
	case uint:
		return int(value)
	case uint8:
		return int(value)
	case uint16:
		return int(value)
	case uint32:
		return int(value)
	case uint64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		i, _ := strconv.Atoi(value.String())
		return i
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(value))
		return i
	default:
		return 0
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
