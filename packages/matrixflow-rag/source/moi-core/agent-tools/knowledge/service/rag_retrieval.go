package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	tools "github.com/matrixflow/moi-core/agent-tools/knowledge"
	corelog "github.com/matrixflow/moi-core/model/logging"
	"go.uber.org/zap"
	xhtml "golang.org/x/net/html"
)

const (
	defaultRAGFileLimit       = 20
	maxRAGFileLimit           = 50
	ragVectorDistanceFunction = "l2_distance"
	ragFulltextRoute          = "fulltext"
	ragVectorRoute            = "vector_l2"
	ragHeadingMaxRunes        = 80
	ragHeadingForwardWindow   = 3
	ragFocusedTableMinRunes   = 800
	ragFocusedTableMaxRows    = 8
	ragAdjacentImageWindow    = 2
	ragMaxAdjacentImageRefs   = 1
)

var ragCatalogImageRefPattern = regexp.MustCompile(`(?i)^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})(\.(png|jpe?g|gif|webp|bmp|tiff?))?$`)

type findRAGFilesService struct {
	executor tools.SQLExecutor
}

func NewFindRAGFiles(deps Deps) tools.FindRAGFiles {
	return &findRAGFilesService{executor: deps.SQLExecutor}
}

func (s *findRAGFilesService) Execute(ctx context.Context, req tools.FindRAGFilesRequest) (*tools.FindRAGFilesResponse, error) {
	if s.executor == nil {
		return nil, errors.New("find_rag_files: executor not configured")
	}
	maxFiles := clampInt(req.MaxFiles, defaultRAGFileLimit, maxRAGFileLimit)
	queries, err := ragToolQueries(req.Scope, req.VolumeID, req.FileIDs)
	if err != nil {
		return nil, err
	}
	if len(queries) == 0 {
		return &tools.FindRAGFilesResponse{
			Files:    []tools.RAGFileRef{},
			RowCount: 0,
		}, nil
	}
	out := &tools.FindRAGFilesResponse{
		Files: make([]tools.RAGFileRef, 0, maxFiles),
	}
	seen := map[string]struct{}{}
	rowCount := 0
	for _, query := range queries {
		resp, err := s.executeOne(ctx, req, query, maxFiles)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, errors.New("find_rag_files: executor returned nil response")
		}
		rowCount += resp.RowCount
		for _, file := range resp.Files {
			file.SourceTags = ragSourceTagsForFile(query.scope, file.FileID)
			key := strings.TrimSpace(file.FileID) + "\x00" + strings.TrimSpace(file.IndexVersion) + "\x00" + strings.TrimSpace(file.SourceURI)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out.Files = append(out.Files, file)
			if len(out.Files) >= maxFiles {
				break
			}
		}
		if len(out.Files) >= maxFiles {
			break
		}
	}
	out.RowCount = rowCount
	return out, nil
}

func (s *findRAGFilesService) executeOne(ctx context.Context, req tools.FindRAGFilesRequest, query ragToolQuery, maxFiles int) (*tools.FindRAGFilesResponse, error) {
	table := strings.TrimSpace(query.scope.VectorTable)
	if table == "" {
		return nil, errors.New("find_rag_files: vector_table required")
	}
	columns, err := s.loadRAGColumns(ctx, query.scope, table)
	if err != nil {
		return nil, err
	}
	where := []string{"COALESCE(disabled, 0) = 0", "level = 'doc'"}
	where = append(where, ragCurrentVersionWhere(query.scope, query.fileIDs, columns)...)
	if volumeID := strings.TrimSpace(query.volumeID); volumeID != "" {
		where = append(where, ragVolumeExpr(columns)+" = '"+ragEscapeSQLString(volumeID)+"'")
	}
	for _, term := range []string{req.Company, req.Year, req.ReportType, req.Query} {
		if v := strings.TrimSpace(term); v != "" {
			where = append(where, ragFileNameExpr()+" LIKE '%"+ragEscapeLike(v)+"%'")
		}
	}
	sqlText := fmt.Sprintf(`
SELECT DISTINCT
  file_id,
  %s,
  %s AS file_name,
  %s AS volume_id,
  %s AS source_uri
FROM %s
WHERE %s
ORDER BY file_name
LIMIT %d`,
		ragIndexVersionSelectExpr(columns),
		ragFileNameExpr(),
		ragVolumeExpr(columns),
		ragSourceURIExpr(),
		ragQuoteIdentifier(table),
		strings.Join(where, "\n  AND "),
		maxFiles+1,
	)
	exec, err := s.executor.ExecuteSQL(ctx, strings.TrimSpace(query.scope.DBName), sqlText)
	if err != nil {
		return nil, fmt.Errorf("find_rag_files: %w", err)
	}
	if exec == nil {
		return nil, errors.New("find_rag_files: executor returned nil result")
	}
	rows := exec.Rows
	if len(rows) > maxFiles {
		rows = rows[:maxFiles]
	}
	out := make([]tools.RAGFileRef, 0, len(rows))
	for _, row := range rows {
		record := ragRowRecord(exec.Columns, row)
		fileID := ragRowString(record, "file_id")
		if fileID == "" {
			continue
		}
		out = append(out, tools.RAGFileRef{
			FileID:       fileID,
			IndexVersion: ragRowString(record, "index_version"),
			FileName:     ragRowString(record, "file_name"),
			VolumeID:     ragRowString(record, "volume_id"),
			SourceURI:    ragRowString(record, "source_uri"),
		})
	}
	return &tools.FindRAGFilesResponse{Files: out, RowCount: len(out)}, nil
}

type searchRAGChunksService struct {
	executor              tools.SQLExecutor
	embedder              tools.EmbeddingService
	defaultEmbeddingModel string
}

func NewSearchRAGChunks(deps Deps) tools.SearchRAGChunks {
	return &searchRAGChunksService{
		executor:              deps.SQLExecutor,
		embedder:              deps.Embedder,
		defaultEmbeddingModel: strings.TrimSpace(deps.DefaultRetrieverConfig.EmbeddingModel),
	}
}

func (s *searchRAGChunksService) Execute(ctx context.Context, req tools.SearchRAGChunksRequest) (*tools.SearchRAGChunksResponse, error) {
	queries, err := ragToolQueries(req.Scope, req.VolumeID, req.FileIDs)
	if err != nil {
		return nil, err
	}
	if len(queries) == 0 {
		return &tools.SearchRAGChunksResponse{
			Keywards: append([]string(nil), ragSearchKeywards(req.Keywards)...),
			Routes:   []string{},
			Chunks:   []tools.RAGChunkHit{},
		}, nil
	}
	if len(queries) == 1 {
		next := req
		next.Scope = queries[0].scope
		next.VolumeID = queries[0].volumeID
		next.FileIDs = queries[0].fileIDs
		resp, err := s.executeOne(ctx, next)
		if err != nil {
			return nil, err
		}
		stampRAGResponseOwner(resp, queries[0])
		return resp, nil
	}
	out := &tools.SearchRAGChunksResponse{
		Keywards:       append([]string(nil), ragSearchKeywards(req.Keywards)...),
		Routes:         []string{},
		Chunks:         []tools.RAGChunkHit{},
		ExpansionSQLs:  []string{},
		EmbeddingModel: "",
	}
	seenChunks := map[string]int{}
	seenRoutes := map[string]struct{}{}
	seenGroups := map[string]struct{}{}
	anchorRankOffset := 0
	for _, query := range queries {
		next := req
		next.Scope = query.scope
		next.VolumeID = query.volumeID
		next.FileIDs = query.fileIDs
		resp, err := s.executeOne(ctx, next)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, errors.New("search_rag_chunks: executor returned nil response")
		}
		stampRAGResponseOwner(resp, query)
		out.FulltextRows += resp.FulltextRows
		out.VectorRows += resp.VectorRows
		out.FulltextSQL = appendRAGDebugSQL(out.FulltextSQL, resp.FulltextSQL)
		out.VectorSQL = appendRAGDebugSQL(out.VectorSQL, resp.VectorSQL)
		out.ExpansionSQLs = append(out.ExpansionSQLs, resp.ExpansionSQLs...)
		out.EmbeddingModel = firstRAGNonEmpty(out.EmbeddingModel, resp.EmbeddingModel)
		for _, route := range resp.Routes {
			route = strings.TrimSpace(route)
			if route == "" {
				continue
			}
			if _, ok := seenRoutes[route]; ok {
				continue
			}
			seenRoutes[route] = struct{}{}
			out.Routes = append(out.Routes, route)
		}
		sourceMaxAnchorRank := 0
		for _, hit := range resp.Chunks {
			if hit.RetrievalAnchorRank > sourceMaxAnchorRank {
				sourceMaxAnchorRank = hit.RetrievalAnchorRank
			}
			if hit.RetrievalAnchorRank > 0 {
				hit.RetrievalAnchorRank += anchorRankOffset
			}
			hit.SourceTags = ragSourceTagsForFile(query.scope, hit.FileID)
			key := ragChunkHitKey(hit)
			if outputIndex, ok := seenChunks[key]; ok {
				out.Chunks[outputIndex].RetrievalAnchorRank = ragBestRetrievalAnchorRank(
					out.Chunks[outputIndex].RetrievalAnchorRank,
					hit.RetrievalAnchorRank,
				)
				continue
			}
			seenChunks[key] = len(out.Chunks)
			out.Chunks = append(out.Chunks, hit)
			if groupID := strings.TrimSpace(hit.EvidenceGroupID); groupID != "" {
				seenGroups[ragSemanticModelEvidenceKey(hit.SemanticModelID, groupID)] = struct{}{}
			}
		}
		anchorRankOffset += sourceMaxAnchorRank
	}
	out.RowCount = len(out.Chunks)
	out.ExpandedGroups = len(seenGroups)
	return out, nil
}

func (s *searchRAGChunksService) executeOne(ctx context.Context, req tools.SearchRAGChunksRequest) (*tools.SearchRAGChunksResponse, error) {
	totalStart := time.Now()
	log := ragRetrievalLogger(ctx)
	if s.executor == nil {
		return nil, errors.New("search_rag_chunks: executor not configured")
	}
	if s.embedder == nil {
		return nil, errors.New("search_rag_chunks: embedder not configured")
	}
	keywards := ragSearchKeywards(req.Keywards)
	if len(keywards) == 0 {
		return nil, errors.New("search_rag_chunks: keywards required")
	}
	table := strings.TrimSpace(req.Scope.VectorTable)
	if table == "" {
		return nil, errors.New("search_rag_chunks: vector_table required")
	}
	maxHits := req.MaxHits
	if maxHits <= 0 {
		return nil, errors.New("search_rag_chunks: max_hits required")
	}
	maxRows := req.MaxRows
	before := req.Before
	after := req.After
	if before < 0 {
		return nil, errors.New("search_rag_chunks: before must be >= 0")
	}
	if after < 0 {
		return nil, errors.New("search_rag_chunks: after must be >= 0")
	}
	log.Info("search_rag_chunks request received",
		zap.String("operation", "request_received"),
		zap.String("workspace_id", strings.TrimSpace(req.Scope.WorkspaceID)),
		zap.String("db_name", strings.TrimSpace(req.Scope.DBName)),
		zap.String("vector_table", table),
		zap.Int("keyward_count", len(keywards)),
		zap.Strings("keywards", keywards),
		zap.Int("file_id_count", len(req.FileIDs)),
		zap.Strings("file_ids", req.FileIDs),
		zap.Bool("has_volume_id", strings.TrimSpace(req.VolumeID) != ""),
		zap.Int("requested_max_hits", req.MaxHits),
		zap.Int("requested_max_rows", req.MaxRows),
		zap.Int("requested_before", req.Before),
		zap.Int("requested_after", req.After),
	)
	loadColumnsStart := time.Now()
	columns, err := s.loadRAGColumns(ctx, req.Scope, table)
	if err != nil {
		log.Error("search_rag_chunks load columns failed",
			zap.String("operation", "load_columns_failed"),
			zap.Int64("stage_duration_ms", time.Since(loadColumnsStart).Milliseconds()),
			zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
			zap.Error(err),
		)
		return nil, err
	}
	log.Info("search_rag_chunks load columns completed",
		zap.String("operation", "load_columns_completed"),
		zap.Int("column_count", len(columns)),
		zap.Int64("stage_duration_ms", time.Since(loadColumnsStart).Milliseconds()),
		zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
	)
	embeddingModel := strings.TrimSpace(req.Scope.EmbeddingModel)
	if embeddingModel == "" {
		embeddingModel = s.defaultEmbeddingModel
	}
	if embeddingModel == "" {
		return nil, errors.New("search_rag_chunks: embedding_model required")
	}
	log.Info("search_rag_chunks normalized options",
		zap.String("operation", "options_normalized"),
		zap.Int("max_hits", maxHits),
		zap.Int("max_rows", maxRows),
		zap.Int("before", before),
		zap.Int("after", after),
		zap.String("embedding_model", embeddingModel),
	)

	fulltextSQL := ragFulltextCandidateSQL(req.Scope, table, columns, keywards, req.FileIDs, req.VolumeID, maxHits)
	fulltextStart := time.Now()
	fulltext, err := s.executor.ExecuteSQL(ctx, strings.TrimSpace(req.Scope.DBName), fulltextSQL)
	if err != nil {
		log.Error("search_rag_chunks fulltext route failed",
			zap.String("operation", "fulltext_route_failed"),
			zap.Int64("stage_duration_ms", time.Since(fulltextStart).Milliseconds()),
			zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("search_rag_chunks fulltext route: %w", err)
	}
	if fulltext == nil {
		return nil, errors.New("search_rag_chunks fulltext route: executor returned nil result")
	}
	log.Info("search_rag_chunks fulltext route completed",
		zap.String("operation", "fulltext_route_completed"),
		zap.Int("row_count", len(fulltext.Rows)),
		zap.Int("keyward_count", len(keywards)),
		zap.Int64("stage_duration_ms", time.Since(fulltextStart).Milliseconds()),
		zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
	)

	embeddingStart := time.Now()
	embeddings, err := s.embedder.CreateEmbedding(ctx, req.Scope.WorkspaceID, embeddingModel, keywards)
	if err != nil {
		log.Error("search_rag_chunks embedding failed",
			zap.String("operation", "embedding_failed"),
			zap.String("embedding_model", embeddingModel),
			zap.Int64("stage_duration_ms", time.Since(embeddingStart).Milliseconds()),
			zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("search_rag_chunks vector route embedding: %w", err)
	}
	if len(embeddings) != len(keywards) {
		return nil, fmt.Errorf("search_rag_chunks vector route embedding: got %d embeddings for %d keywards", len(embeddings), len(keywards))
	}
	for idx, embedding := range embeddings {
		if len(embedding) == 0 {
			return nil, fmt.Errorf("search_rag_chunks vector route embedding: empty vector for keyward %d", idx)
		}
	}
	log.Info("search_rag_chunks embedding completed",
		zap.String("operation", "embedding_completed"),
		zap.String("embedding_model", embeddingModel),
		zap.Int("embedding_count", len(embeddings)),
		zap.Int("embedding_dim", len(embeddings[0])),
		zap.Int64("stage_duration_ms", time.Since(embeddingStart).Milliseconds()),
		zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
	)
	vectorSQL, err := ragVectorCandidateSQL(req.Scope, table, columns, embeddings, req.FileIDs, req.VolumeID, maxHits)
	if err != nil {
		return nil, fmt.Errorf("search_rag_chunks vector route embedding: %w", err)
	}
	vectorStart := time.Now()
	vector, err := s.executor.ExecuteSQL(ctx, strings.TrimSpace(req.Scope.DBName), vectorSQL)
	if err != nil {
		log.Error("search_rag_chunks vector route failed",
			zap.String("operation", "vector_route_failed"),
			zap.Int64("stage_duration_ms", time.Since(vectorStart).Milliseconds()),
			zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("search_rag_chunks vector route: %w", err)
	}
	if vector == nil {
		return nil, errors.New("search_rag_chunks vector route: executor returned nil result")
	}
	log.Info("search_rag_chunks vector route completed",
		zap.String("operation", "vector_route_completed"),
		zap.Int("row_count", len(vector.Rows)),
		zap.Int64("stage_duration_ms", time.Since(vectorStart).Milliseconds()),
		zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
	)

	mergeStart := time.Now()
	candidates, err := mergeRAGCandidates(fulltext, vector)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		leftLiteral := ragCandidateLiteralMatchScore(candidates[i].hit, keywards)
		rightLiteral := ragCandidateLiteralMatchScore(candidates[j].hit, keywards)
		if leftLiteral != rightLiteral {
			return leftLiteral > rightLiteral
		}
		if candidates[i].rank != candidates[j].rank {
			return candidates[i].rank < candidates[j].rank
		}
		return ragChunkHitKey(candidates[i].hit) < ragChunkHitKey(candidates[j].hit)
	})
	anchorRankByKey := make(map[string]int, len(candidates))
	for i := range candidates {
		candidates[i].hit.RetrievalAnchorRank = i + 1
		anchorRankByKey[ragChunkHitKey(candidates[i].hit)] = candidates[i].hit.RetrievalAnchorRank
	}
	log.Info("search_rag_chunks candidates merged",
		zap.String("operation", "candidates_merged"),
		zap.Int("candidate_count", len(candidates)),
		zap.Int64("stage_duration_ms", time.Since(mergeStart).Milliseconds()),
		zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
	)

	resp := &tools.SearchRAGChunksResponse{
		Keywards:       append([]string(nil), keywards...),
		Routes:         []string{ragFulltextRoute, ragVectorRoute},
		FulltextSQL:    fulltextSQL,
		VectorSQL:      vectorSQL,
		FulltextRows:   len(fulltext.Rows),
		VectorRows:     len(vector.Rows),
		EmbeddingModel: embeddingModel,
	}
	planStart := time.Now()
	plans, err := s.planRAGExpansions(ctx, candidates, before, after)
	if err != nil {
		return nil, err
	}
	log.Info("search_rag_chunks expansion plans prepared",
		zap.String("operation", "expansion_plans_prepared"),
		zap.Int("candidate_count", len(candidates)),
		zap.Int("plan_count", len(plans)),
		zap.Int64("stage_duration_ms", time.Since(planStart).Milliseconds()),
		zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
	)

	seenChunks := make(map[string]int)
	seenGroups := make(map[string]struct{})
	plansVisited := 0
	for planIndex, plan := range plans {
		if maxRows > 0 && plansVisited >= maxRows {
			break
		}
		plansVisited++
		group, sqls, err := s.expandRAGPlan(ctx, req.Scope, table, columns, plan, anchorRankByKey)
		if err != nil {
			return nil, err
		}
		resp.ExpansionSQLs = append(resp.ExpansionSQLs, sqls...)
		if len(group) == 0 {
			group = []tools.RAGChunkHit{plan.candidate.hit}
		}
		group = focusRAGTableEvidenceRows(group, keywards)
		for _, hit := range group {
			if strings.TrimSpace(hit.Content) == "" {
				continue
			}
			hit.SourceTags = ragSourceTagsForFile(req.Scope, hit.FileID)
			if len(hit.Routes) == 0 {
				hit.Routes = append([]string(nil), plan.candidate.hit.Routes...)
			}
			if hit.Score == 0 {
				hit.Score = plan.candidate.hit.Score
			}
			key := ragChunkHitKey(hit)
			if outputIndex, exists := seenChunks[key]; exists {
				resp.Chunks[outputIndex].RetrievalAnchorRank = ragBestRetrievalAnchorRank(
					resp.Chunks[outputIndex].RetrievalAnchorRank,
					hit.RetrievalAnchorRank,
				)
				continue
			}
			seenChunks[key] = len(resp.Chunks)
			if strings.TrimSpace(hit.EvidenceGroupID) != "" {
				if _, exists := seenGroups[hit.EvidenceGroupID]; !exists {
					seenGroups[hit.EvidenceGroupID] = struct{}{}
					resp.ExpandedGroups++
				}
			}
			resp.Chunks = append(resp.Chunks, hit)
		}
		remainingMaxRows := 0
		if maxRows > 0 {
			remainingMaxRows = maxRows - plansVisited
		}
		log.Info("search_rag_chunks expansion plan processed",
			zap.String("operation", "expansion_plan_processed"),
			zap.Int("plan_index", planIndex),
			zap.String("plan_file_id", plan.candidate.hit.FileID),
			zap.String("plan_index_version", plan.candidate.hit.IndexVersion),
			zap.String("plan_level", plan.candidate.hit.Level),
			zap.Strings("plan_routes", plan.candidate.hit.Routes),
			zap.String("plan_chunk_id", plan.candidate.hit.ChunkID),
			zap.String("plan_parent_index", ragPtrString(plan.candidate.hit.ParentIndex)),
			zap.String("plan_chunk_index", ragPtrString(plan.candidate.hit.ChunkIndex)),
			zap.Bool("file_scoped_table", plan.fileScopedTable),
			zap.Int("range_start", plan.start),
			zap.Int("range_end", plan.end),
			zap.Int("min_parent", plan.minParent),
			zap.Int("max_parent", plan.maxParent),
			zap.Int("group_row_count", len(group)),
			zap.Int("group_content_chars", ragChunkHitsContentChars(group)),
			zap.Int("response_chunk_count", len(resp.Chunks)),
			zap.Int("response_content_chars", ragChunkHitsContentChars(resp.Chunks)),
			zap.Int("remaining_max_rows", remainingMaxRows),
			zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
		)
	}
	imageRefStart := time.Now()
	enrichedChunks, imageRefSQLs, err := s.attachRAGEmbeddedImageRefs(ctx, req.Scope, table, columns, resp.Chunks)
	if err != nil {
		log.Error("search_rag_chunks image ref enrichment failed",
			zap.String("operation", "image_ref_enrichment_failed"),
			zap.Int64("stage_duration_ms", time.Since(imageRefStart).Milliseconds()),
			zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
			zap.Error(err),
		)
		return nil, err
	}
	resp.Chunks = enrichedChunks
	resp.Chunks = ensureRAGChunkVisualRefs(resp.Chunks)
	enrichedChunks, adjacentImageRefSQLs, err := s.attachRAGAdjacentImageRefs(ctx, req.Scope, table, columns, resp.Chunks)
	if err != nil {
		log.Error("search_rag_chunks adjacent image ref enrichment failed",
			zap.String("operation", "adjacent_image_ref_enrichment_failed"),
			zap.Int64("stage_duration_ms", time.Since(imageRefStart).Milliseconds()),
			zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
			zap.Error(err),
		)
		return nil, err
	}
	resp.Chunks = enrichedChunks
	resp.ExpansionSQLs = append(resp.ExpansionSQLs, imageRefSQLs...)
	resp.ExpansionSQLs = append(resp.ExpansionSQLs, adjacentImageRefSQLs...)
	resp.RowCount = len(resp.Chunks)
	log.Info("search_rag_chunks completed",
		zap.String("operation", "completed"),
		zap.Int("fulltext_rows", len(fulltext.Rows)),
		zap.Int("vector_rows", len(vector.Rows)),
		zap.Int("candidate_count", len(candidates)),
		zap.Int("plans_visited", plansVisited),
		zap.Int("expanded_group_count", resp.ExpandedGroups),
		zap.Int("expansion_sql_count", len(resp.ExpansionSQLs)),
		zap.Int("chunk_count", len(resp.Chunks)),
		zap.Int("content_chars", ragChunkHitsContentChars(resp.Chunks)),
		zap.Bool("hit_max_rows", maxRows > 0 && plansVisited >= maxRows),
		zap.Int64("total_duration_ms", time.Since(totalStart).Milliseconds()),
	)
	return resp, nil
}

func (s *searchRAGChunksService) attachRAGEmbeddedImageRefs(
	ctx context.Context,
	scope tools.WorkspaceScope,
	table string,
	columns ragColumnSet,
	chunks []tools.RAGChunkHit,
) ([]tools.RAGChunkHit, []string, error) {
	type groupKey struct {
		fileID       string
		indexVersion string
	}
	uuidIndexes := make(map[groupKey]map[string][]int)
	for idx, chunk := range chunks {
		fileID := strings.TrimSpace(chunk.FileID)
		indexVersion := strings.TrimSpace(chunk.IndexVersion)
		if fileID == "" || indexVersion == "" {
			continue
		}
		uuids := ragEmbeddedImageBlockUUIDs(chunk)
		if len(uuids) == 0 {
			continue
		}
		key := groupKey{fileID: fileID, indexVersion: indexVersion}
		if uuidIndexes[key] == nil {
			uuidIndexes[key] = map[string][]int{}
		}
		for _, uuid := range uuids {
			uuidIndexes[key][uuid] = append(uuidIndexes[key][uuid], idx)
		}
	}
	if len(uuidIndexes) == 0 {
		return chunks, nil, nil
	}
	out := append([]tools.RAGChunkHit(nil), chunks...)
	sqls := make([]string, 0, len(uuidIndexes))
	markdownFileIDExpr := ragMarkdownFileIDExpr(columns)
	parentExpr := ragTopLevelOrMetaStringExpr("parent_index", columns)
	chunkStartExpr := ragTopLevelOrMetaStringExpr("chunk_start", columns)
	chunkEndExpr := ragTopLevelOrMetaStringExpr("chunk_end", columns)
	for key, byUUID := range uuidIndexes {
		uuids := make([]string, 0, len(byUUID))
		for uuid := range byUUID {
			uuids = append(uuids, uuid)
		}
		sort.Strings(uuids)
		sqlText := fmt.Sprintf(`
SELECT
  level,
  content,
  meta,
  file_id,
  %s AS markdown_file_id,
  index_version,
  chunk_index,
  %s AS parent_index,
  %s AS chunk_start,
  %s AS chunk_end,
  1 AS score
FROM %s
WHERE COALESCE(disabled, 0) = 0
  AND file_id = '%s'
  AND index_version = '%s'
  AND level = 'chunk'
  AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.source_block_type')) = 'IMAGE'
  AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.block_uuid')) IN (%s)
ORDER BY
  chunk_index`,
			markdownFileIDExpr,
			parentExpr,
			chunkStartExpr,
			chunkEndExpr,
			ragQuoteIdentifier(table),
			ragEscapeSQLString(key.fileID),
			ragEscapeSQLString(key.indexVersion),
			ragQuotedList(uuids),
		)
		sqls = append(sqls, sqlText)
		exec, err := s.executor.ExecuteSQL(ctx, strings.TrimSpace(scope.DBName), sqlText)
		if err != nil {
			return nil, nil, fmt.Errorf("search_rag_chunks embedded image refs file_id=%s index_version=%s: %w", key.fileID, key.indexVersion, err)
		}
		if exec == nil {
			return nil, nil, errors.New("search_rag_chunks embedded image refs: executor returned nil result")
		}
		for rowIdx, row := range exec.Rows {
			record := ragRowRecord(exec.Columns, row)
			imageHit, err := ragChunkHitFromRecord(record)
			if err != nil {
				return nil, nil, fmt.Errorf("search_rag_chunks embedded image refs row=%d: %w", rowIdx, err)
			}
			blockUUID := strings.TrimSpace(imageHit.BlockUUID)
			indexes := byUUID[blockUUID]
			if len(indexes) == 0 {
				continue
			}
			ref, ok := ragImageRefFromChunk(imageHit)
			if !ok {
				continue
			}
			for _, idx := range indexes {
				ref.ChunkID = firstRAGNonEmpty(ref.ChunkID, out[idx].ChunkID)
				out[idx].VisualRefs = compactRAGImageRefs(append(out[idx].VisualRefs, ref))
			}
		}
	}
	return out, sqls, nil
}

func (s *searchRAGChunksService) attachRAGAdjacentImageRefs(
	ctx context.Context,
	scope tools.WorkspaceScope,
	table string,
	columns ragColumnSet,
	chunks []tools.RAGChunkHit,
) ([]tools.RAGChunkHit, []string, error) {
	type groupKey struct {
		fileID       string
		indexVersion string
	}
	type targetRef struct {
		parent int
		index  int
	}
	targets := make(map[groupKey][]targetRef)
	neighborIndexes := make(map[groupKey]map[int]struct{})
	for idx, chunk := range chunks {
		if len(chunk.VisualRefs) > 0 || strings.TrimSpace(chunk.ImageFileID) != "" || strings.TrimSpace(chunk.PageImageFileID) != "" {
			continue
		}
		fileID := strings.TrimSpace(chunk.FileID)
		indexVersion := strings.TrimSpace(chunk.IndexVersion)
		if fileID == "" || indexVersion == "" || chunk.ParentIndex == nil {
			continue
		}
		key := groupKey{fileID: fileID, indexVersion: indexVersion}
		parent := *chunk.ParentIndex
		targets[key] = append(targets[key], targetRef{parent: parent, index: idx})
		if neighborIndexes[key] == nil {
			neighborIndexes[key] = map[int]struct{}{}
		}
		for next := parent - ragAdjacentImageWindow; next <= parent+ragAdjacentImageWindow; next++ {
			if next < 0 || next == parent {
				continue
			}
			neighborIndexes[key][next] = struct{}{}
		}
	}
	if len(targets) == 0 {
		return chunks, nil, nil
	}
	out := append([]tools.RAGChunkHit(nil), chunks...)
	sqls := make([]string, 0, len(targets))
	markdownFileIDExpr := ragMarkdownFileIDExpr(columns)
	parentExpr := ragTopLevelOrMetaStringExpr("parent_index", columns)
	chunkStartExpr := ragTopLevelOrMetaStringExpr("chunk_start", columns)
	chunkEndExpr := ragTopLevelOrMetaStringExpr("chunk_end", columns)
	for key, refs := range targets {
		indexes := make([]int, 0, len(neighborIndexes[key]))
		for value := range neighborIndexes[key] {
			indexes = append(indexes, value)
		}
		if len(indexes) == 0 {
			continue
		}
		sort.Ints(indexes)
		sqlText := fmt.Sprintf(`
SELECT
  level,
  content,
  meta,
  file_id,
  %s AS markdown_file_id,
  index_version,
  chunk_index,
  %s AS parent_index,
  %s AS chunk_start,
  %s AS chunk_end,
  1 AS score
FROM %s
WHERE COALESCE(disabled, 0) = 0
  AND file_id = '%s'
  AND index_version = '%s'
  AND level = 'chunk'
  AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.source_block_type')) = 'IMAGE'
  AND CAST(%s AS SIGNED) IN (%s)
ORDER BY
  CAST(%s AS SIGNED),
  chunk_index`,
			markdownFileIDExpr,
			parentExpr,
			chunkStartExpr,
			chunkEndExpr,
			ragQuoteIdentifier(table),
			ragEscapeSQLString(key.fileID),
			ragEscapeSQLString(key.indexVersion),
			parentExpr,
			ragIntList(indexes),
			parentExpr,
		)
		sqls = append(sqls, sqlText)
		exec, err := s.executor.ExecuteSQL(ctx, strings.TrimSpace(scope.DBName), sqlText)
		if err != nil {
			return nil, nil, fmt.Errorf("search_rag_chunks adjacent image refs file_id=%s index_version=%s: %w", key.fileID, key.indexVersion, err)
		}
		if exec == nil {
			return nil, nil, errors.New("search_rag_chunks adjacent image refs: executor returned nil result")
		}
		images := make([]tools.RAGChunkHit, 0, len(exec.Rows))
		for rowIdx, row := range exec.Rows {
			record := ragRowRecord(exec.Columns, row)
			imageHit, err := ragChunkHitFromRecord(record)
			if err != nil {
				return nil, nil, fmt.Errorf("search_rag_chunks adjacent image refs row=%d: %w", rowIdx, err)
			}
			if imageHit.ParentIndex == nil {
				continue
			}
			if _, ok := ragImageRefFromChunk(imageHit); !ok {
				continue
			}
			images = append(images, imageHit)
		}
		if len(images) == 0 {
			continue
		}
		for _, target := range refs {
			matches := nearestRAGAdjacentImageRefs(target.parent, images)
			for _, ref := range matches {
				ref.ChunkID = firstRAGNonEmpty(ref.ChunkID, out[target.index].ChunkID)
				out[target.index].VisualRefs = compactRAGImageRefs(append(out[target.index].VisualRefs, ref))
			}
		}
	}
	return out, sqls, nil
}

func nearestRAGAdjacentImageRefs(parent int, images []tools.RAGChunkHit) []tools.RAGImageRef {
	if len(images) == 0 {
		return nil
	}
	type scoredRef struct {
		distance int
		parent   int
		ref      tools.RAGImageRef
	}
	scored := make([]scoredRef, 0, len(images))
	for _, image := range images {
		if image.ParentIndex == nil {
			continue
		}
		distance := parent - *image.ParentIndex
		if distance < 0 {
			distance = -distance
		}
		if distance == 0 || distance > ragAdjacentImageWindow {
			continue
		}
		ref, ok := ragImageRefFromChunk(image)
		if !ok {
			continue
		}
		scored = append(scored, scoredRef{distance: distance, parent: *image.ParentIndex, ref: ref})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].distance != scored[j].distance {
			return scored[i].distance < scored[j].distance
		}
		return scored[i].parent < scored[j].parent
	})
	if len(scored) > ragMaxAdjacentImageRefs {
		scored = scored[:ragMaxAdjacentImageRefs]
	}
	out := make([]tools.RAGImageRef, 0, len(scored))
	for _, item := range scored {
		out = append(out, item.ref)
	}
	return out
}

func ensureRAGChunkVisualRefs(chunks []tools.RAGChunkHit) []tools.RAGChunkHit {
	if len(chunks) == 0 {
		return chunks
	}
	out := append([]tools.RAGChunkHit(nil), chunks...)
	for idx := range out {
		ref, ok := ragImageRefFromChunk(out[idx])
		if !ok || (ref.ImageFileID == "" && ref.PageImageFileID == "") {
			continue
		}
		out[idx].VisualRefs = compactRAGImageRefs(append([]tools.RAGImageRef{ref}, out[idx].VisualRefs...))
	}
	return out
}

func appendRAGDebugSQL(existing, next string) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return existing
	}
	if strings.TrimSpace(existing) == "" {
		return next
	}
	return existing + "\n\n" + next
}

type ragToolQuery struct {
	scope           tools.WorkspaceScope
	semanticModelID int64
	volumeID        string
	fileIDs         []string
}

func ragToolQueries(scope tools.WorkspaceScope, reqVolumeID string, requestedFileIDs []string) ([]ragToolQuery, error) {
	type sourceFilter struct {
		scope           tools.WorkspaceScope
		semanticModelID int64
		volumeID        string
		fileIDs         []string
	}
	filters := make([]sourceFilter, 0, len(scope.RAGSources)+1)
	if len(scope.RAGSources) > 0 {
		for _, source := range scope.RAGSources {
			nextScope := scope
			nextScope.DBName = strings.TrimSpace(source.DBName)
			if ragSourceIsSemanticModel(source) {
				// Semantic KB sources must use the text index declared by that
				// KB. Do not inherit another KB's top-level vector_table fallback.
				if strings.TrimSpace(source.VectorTable) == "" || strings.TrimSpace(source.EmbeddingModel) == "" {
					continue
				}
				nextScope.VectorTable = strings.TrimSpace(source.VectorTable)
				nextScope.EmbeddingModel = strings.TrimSpace(source.EmbeddingModel)
			} else {
				nextScope.VectorTable = firstRAGNonEmpty(source.VectorTable, scope.VectorTable)
				nextScope.EmbeddingModel = firstRAGNonEmpty(source.EmbeddingModel, scope.EmbeddingModel)
			}
			nextScope.RAGSources = []tools.RAGSource{source}
			filters = append(filters, sourceFilter{
				scope:           nextScope,
				semanticModelID: source.SemanticModelID,
				volumeID:        firstRAGNonEmpty(reqVolumeID, source.VolumeID),
				fileIDs:         source.FileIDs,
			})
		}
	} else {
		nextScope := scope
		nextScope.RAGSources = nil
		filters = append(filters, sourceFilter{
			scope:           nextScope,
			semanticModelID: uniquePositiveSemanticModelID(scope.SemanticModelIDs),
			volumeID:        firstRAGNonEmpty(reqVolumeID, scope.VolumeID),
			fileIDs:         scope.FileIDs,
		})
	}
	requested := ragCompactStrings(requestedFileIDs)
	out := make([]ragToolQuery, 0, len(filters))
	seen := map[string]struct{}{}
	for _, filter := range filters {
		filter.scope.WorkspaceID = scope.WorkspaceID
		filter.scope.DBName = strings.TrimSpace(filter.scope.DBName)
		filter.scope.VectorTable = strings.TrimSpace(filter.scope.VectorTable)
		if filter.scope.VectorTable == "" {
			return nil, fmt.Errorf("rag search: vector_table is required")
		}
		filter.scope.EmbeddingModel = strings.TrimSpace(filter.scope.EmbeddingModel)
		fileIDs := ragCompactStrings(filter.fileIDs)
		if len(requested) > 0 {
			if len(fileIDs) > 0 {
				fileIDs = ragIntersectStrings(fileIDs, requested)
			} else {
				fileIDs = requested
			}
			if len(fileIDs) == 0 {
				continue
			}
		}
		key := strings.TrimSpace(filter.scope.DBName) + "\x00" +
			filter.scope.VectorTable + "\x00" +
			filter.scope.EmbeddingModel + "\x00" +
			strconv.FormatInt(filter.semanticModelID, 10) + "\x00" +
			strings.TrimSpace(filter.volumeID) + "\x00" +
			strings.Join(fileIDs, "\x00") + "\x00" +
			ragCurrentVersionKey(filter.scope)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ragToolQuery{
			scope:           filter.scope,
			semanticModelID: filter.semanticModelID,
			volumeID:        strings.TrimSpace(filter.volumeID),
			fileIDs:         fileIDs,
		})
	}
	return out, nil
}

func uniquePositiveSemanticModelID(values []int64) int64 {
	var unique int64
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if unique != 0 && unique != value {
			return 0
		}
		unique = value
	}
	return unique
}

func stampRAGResponseOwner(resp *tools.SearchRAGChunksResponse, query ragToolQuery) {
	if resp == nil {
		return
	}
	for chunkIndex := range resp.Chunks {
		chunk := &resp.Chunks[chunkIndex]
		if query.semanticModelID > 0 {
			chunk.SemanticModelID = query.semanticModelID
			for refIndex := range chunk.VisualRefs {
				chunk.VisualRefs[refIndex].SemanticModelID = query.semanticModelID
			}
		}
		chunk.SourceRowID = sourceRowIDForScopeFile(query.scope, chunk.IndexFileID)
		if chunk.SourceRowID == "" {
			chunk.SourceRowID = sourceRowIDForScopeFile(query.scope, chunk.FileID)
		}
	}
}

func sourceRowIDForScopeFile(scope tools.WorkspaceScope, fileID string) string {
	sourceRowIDByFileID := scope.SourceRowIDByFileID
	if len(scope.RAGSources) == 1 {
		sourceRowIDByFileID = scope.RAGSources[0].SourceRowIDByFileID
	}
	return strings.TrimSpace(sourceRowIDByFileID[strings.TrimSpace(fileID)])
}

func ragSemanticModelEvidenceKey(semanticModelID int64, evidenceKey string) string {
	return strconv.FormatInt(semanticModelID, 10) + "\x00" + strings.TrimSpace(evidenceKey)
}

func ragSourceIsSemanticModel(source tools.RAGSource) bool {
	if source.SemanticModelID != 0 {
		return true
	}
	return strings.TrimSpace(source.Metadata["source"]) == "semantic_model"
}

func ragSourceTagsForFile(scope tools.WorkspaceScope, fileID string) []string {
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

func ragCurrentVersionWhere(scope tools.WorkspaceScope, requestedFileIDs []string, columnSets ...ragColumnSet) []string {
	hasIndexVersion := true
	if len(columnSets) > 0 && !columnSets[0].has("index_version") {
		hasIndexVersion = false
	}
	current := ragIndexVersionConstraintByFileID(scope)
	requested := ragCompactStrings(requestedFileIDs)
	if !hasIndexVersion {
		if len(requested) > 0 {
			return []string{"file_id IN (" + ragQuotedList(requested) + ")"}
		}
		if len(current) == 0 {
			return nil
		}
		files := make([]string, 0, len(current))
		for fileID := range current {
			files = append(files, fileID)
		}
		sort.Strings(files)
		return []string{"file_id IN (" + ragQuotedList(files) + ")"}
	}
	if len(current) == 0 {
		if len(requested) == 0 {
			return nil
		}
		return []string{"file_id IN (" + ragQuotedList(requested) + ")"}
	}
	versioned := make([]string, 0, len(current))
	unversioned := make([]string, 0, len(requested))
	if len(requested) > 0 {
		for _, fileID := range requested {
			if _, ok := current[fileID]; ok {
				versioned = append(versioned, fileID)
			} else {
				unversioned = append(unversioned, fileID)
			}
		}
	} else {
		for fileID := range current {
			versioned = append(versioned, fileID)
		}
	}
	sort.Strings(versioned)
	sort.Strings(unversioned)
	if len(versioned) == 0 && len(unversioned) == 0 {
		return []string{"1 = 0"}
	}
	parts := make([]string, 0, len(versioned)+1)
	for _, fileID := range versioned {
		constraint := current[fileID]
		switch constraint.Kind {
		case tools.RAGIndexVersionConstraintValue:
			parts = append(parts, fmt.Sprintf("(file_id = '%s' AND index_version = '%d')", ragEscapeSQLString(fileID), constraint.Value))
		case tools.RAGIndexVersionConstraintNull:
			parts = append(parts, fmt.Sprintf("(file_id = '%s' AND index_version IS NULL)", ragEscapeSQLString(fileID)))
		}
	}
	if len(unversioned) > 0 {
		parts = append(parts, "file_id IN ("+ragQuotedList(unversioned)+")")
	}
	if len(parts) == 0 {
		return []string{"1 = 0"}
	}
	return []string{"(" + strings.Join(parts, " OR ") + ")"}
}

func ragIndexVersionConstraintByFileID(scope tools.WorkspaceScope) map[string]tools.RAGIndexVersionConstraint {
	out := make(map[string]tools.RAGIndexVersionConstraint)
	for fileID, version := range scope.CurrentIndexVersionByFileID {
		fileID = strings.TrimSpace(fileID)
		if fileID != "" && version > 0 {
			out[fileID] = tools.RAGIndexVersionConstraint{Kind: tools.RAGIndexVersionConstraintValue, Value: version}
		}
	}
	for _, source := range scope.RAGSources {
		for fileID, version := range source.CurrentIndexVersionByFileID {
			fileID = strings.TrimSpace(fileID)
			if fileID != "" && version > 0 {
				out[fileID] = tools.RAGIndexVersionConstraint{Kind: tools.RAGIndexVersionConstraintValue, Value: version}
			}
		}
	}
	ragMergeIndexVersionConstraints(out, scope.IndexVersionConstraintByFileID)
	for _, source := range scope.RAGSources {
		ragMergeIndexVersionConstraints(out, source.IndexVersionConstraintByFileID)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ragMergeIndexVersionConstraints(out map[string]tools.RAGIndexVersionConstraint, values map[string]tools.RAGIndexVersionConstraint) {
	for fileID, constraint := range values {
		fileID = strings.TrimSpace(fileID)
		if fileID == "" {
			continue
		}
		switch constraint.Kind {
		case tools.RAGIndexVersionConstraintValue:
			out[fileID] = tools.RAGIndexVersionConstraint{Kind: tools.RAGIndexVersionConstraintValue, Value: constraint.Value}
		case tools.RAGIndexVersionConstraintNull:
			out[fileID] = tools.RAGIndexVersionConstraint{Kind: tools.RAGIndexVersionConstraintNull}
		}
	}
}

func ragCurrentIndexVersionByFileID(scope tools.WorkspaceScope) map[string]int64 {
	out := make(map[string]int64)
	for fileID, version := range scope.CurrentIndexVersionByFileID {
		fileID = strings.TrimSpace(fileID)
		if fileID != "" && version > 0 {
			out[fileID] = version
		}
	}
	for _, source := range scope.RAGSources {
		for fileID, version := range source.CurrentIndexVersionByFileID {
			fileID = strings.TrimSpace(fileID)
			if fileID != "" && version > 0 {
				out[fileID] = version
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func ragCurrentVersionKey(scope tools.WorkspaceScope) string {
	current := ragIndexVersionConstraintByFileID(scope)
	if len(current) == 0 {
		return ""
	}
	files := make([]string, 0, len(current))
	for fileID := range current {
		files = append(files, fileID)
	}
	sort.Strings(files)
	parts := make([]string, 0, len(files))
	for _, fileID := range files {
		constraint := current[fileID]
		switch constraint.Kind {
		case tools.RAGIndexVersionConstraintValue:
			parts = append(parts, fileID+"="+strconv.FormatInt(constraint.Value, 10))
		case tools.RAGIndexVersionConstraintNull:
			parts = append(parts, fileID+"=NULL")
		}
	}
	return strings.Join(parts, ",")
}

func ragIntersectStrings(scopeIDs, requestedIDs []string) []string {
	scopeIDs = ragCompactStrings(scopeIDs)
	requestedIDs = ragCompactStrings(requestedIDs)
	if len(scopeIDs) == 0 || len(requestedIDs) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(scopeIDs))
	for _, value := range scopeIDs {
		allowed[value] = struct{}{}
	}
	out := make([]string, 0, len(requestedIDs))
	for _, value := range requestedIDs {
		if _, ok := allowed[value]; ok {
			out = append(out, value)
		}
	}
	return ragCompactStrings(out)
}

type ragCandidate struct {
	hit  tools.RAGChunkHit
	rank float64
}

type ragExpansionPlan struct {
	candidate       ragCandidate
	minAnchorRank   int
	hasRange        bool
	blockUUID       string
	sameParentOnly  bool
	fileScopedTable bool
	start           int
	end             int
	minParent       int
	maxParent       int
}

func (s *findRAGFilesService) loadRAGColumns(ctx context.Context, scope tools.WorkspaceScope, table string) (ragColumnSet, error) {
	return loadRAGColumns(ctx, s.executor, scope, table, "find_rag_files")
}

func (s *searchRAGChunksService) loadRAGColumns(ctx context.Context, scope tools.WorkspaceScope, table string) (ragColumnSet, error) {
	return loadRAGColumns(ctx, s.executor, scope, table, "search_rag_chunks")
}

func loadRAGColumns(ctx context.Context, executor tools.SQLExecutor, scope tools.WorkspaceScope, table, toolName string) (ragColumnSet, error) {
	sqlText := fmt.Sprintf("SHOW COLUMNS FROM %s", ragQuoteIdentifier(table))
	exec, err := executor.ExecuteSQL(ctx, strings.TrimSpace(scope.DBName), sqlText)
	if err != nil {
		return nil, fmt.Errorf("%s inspect index columns: %w", toolName, err)
	}
	if exec == nil {
		return nil, fmt.Errorf("%s inspect index columns: executor returned nil result", toolName)
	}
	columns := make(ragColumnSet)
	for _, row := range exec.Rows {
		record := ragRowRecord(exec.Columns, row)
		name := firstRAGNonEmpty(ragRowString(record, "field"), ragRowString(record, "column_name"), ragRowString(record, "name"))
		if name == "" && len(row) > 0 {
			name = ragAnyString(row[0])
		}
		if name = normalizeRAGColumnName(name); name != "" {
			columns[name] = struct{}{}
		}
	}
	return columns, nil
}

type ragColumnSet map[string]struct{}

func (c ragColumnSet) has(name string) bool {
	if c == nil {
		return true
	}
	_, ok := c[normalizeRAGColumnName(name)]
	return ok
}

func ragFulltextCandidateSQL(scope tools.WorkspaceScope, table string, columns ragColumnSet, phrases []string, fileIDs []string, volumeID string, maxHits int) string {
	parentExpr := ragTopLevelOrMetaStringExpr("parent_index", columns)
	chunkStartExpr := ragTopLevelOrMetaStringExpr("chunk_start", columns)
	chunkEndExpr := ragTopLevelOrMetaStringExpr("chunk_end", columns)
	chunkIndexScopeExpr := ragMetaStringExpr("chunk_index_scope")
	markdownFileIDExpr := ragMarkdownFileIDExpr(columns)
	indexVersionExpr := ragIndexVersionSelectExpr(columns)
	baseWhere := []string{"COALESCE(disabled, 0) = 0", "level = 'chunk'"}
	baseWhere = append(baseWhere, ragCurrentVersionWhere(scope, fileIDs, columns)...)
	if volumeID = strings.TrimSpace(volumeID); volumeID != "" {
		baseWhere = append(baseWhere, ragVolumeExpr(columns)+" = '"+ragEscapeSQLString(volumeID)+"'")
	}
	parts := make([]string, 0, len(phrases))
	for _, phrase := range phrases {
		matchExpr := ragFulltextMatchExpr(phrase)
		where := append(append([]string(nil), baseWhere...), matchExpr)
		parts = append(parts, fmt.Sprintf(`(
SELECT
  '%s' AS route,
  level,
  content,
  meta,
  file_id,
  %s AS markdown_file_id,
  %s,
  chunk_index,
  %s AS chunk_index_scope,
  %s AS parent_index,
  %s AS chunk_start,
  %s AS chunk_end,
  %s AS score
FROM %s
WHERE %s
ORDER BY score DESC
LIMIT %d)`,
			ragFulltextRoute,
			markdownFileIDExpr,
			indexVersionExpr,
			chunkIndexScopeExpr,
			parentExpr,
			chunkStartExpr,
			chunkEndExpr,
			matchExpr,
			ragQuoteIdentifier(table),
			strings.Join(where, "\n  AND "),
			maxHits,
		))
	}
	return fmt.Sprintf(`
SELECT *
FROM (
%s
) AS rag_fulltext_candidates
ORDER BY score DESC
LIMIT %d`,
		strings.Join(parts, "\nUNION ALL\n"),
		maxHits*len(phrases),
	)
}

func ragVectorCandidateSQL(scope tools.WorkspaceScope, table string, columns ragColumnSet, embeddings [][]float64, fileIDs []string, volumeID string, maxHits int) (string, error) {
	parentExpr := ragTopLevelOrMetaStringExpr("parent_index", columns)
	chunkStartExpr := ragTopLevelOrMetaStringExpr("chunk_start", columns)
	chunkEndExpr := ragTopLevelOrMetaStringExpr("chunk_end", columns)
	chunkIndexScopeExpr := ragMetaStringExpr("chunk_index_scope")
	markdownFileIDExpr := ragMarkdownFileIDExpr(columns)
	indexVersionExpr := ragIndexVersionSelectExpr(columns)
	baseWhere := []string{"COALESCE(disabled, 0) = 0", "level = 'chunk'", "embedding IS NOT NULL"}
	baseWhere = append(baseWhere, ragCurrentVersionWhere(scope, fileIDs, columns)...)
	if volumeID = strings.TrimSpace(volumeID); volumeID != "" {
		baseWhere = append(baseWhere, ragVolumeExpr(columns)+" = '"+ragEscapeSQLString(volumeID)+"'")
	}
	parts := make([]string, 0, len(embeddings))
	for _, embedding := range embeddings {
		vectorLiteral, err := ragVectorLiteral(embedding)
		if err != nil {
			return "", err
		}
		scoreExpr := fmt.Sprintf("%s(embedding, '%s')", ragVectorDistanceFunction, ragEscapeSQLString(vectorLiteral))
		parts = append(parts, fmt.Sprintf(`(
SELECT
  '%s' AS route,
  level,
  content,
  meta,
  file_id,
  %s AS markdown_file_id,
  %s,
  chunk_index,
  %s AS chunk_index_scope,
  %s AS parent_index,
  %s AS chunk_start,
  %s AS chunk_end,
  %s AS score
FROM %s
WHERE %s
ORDER BY score ASC
LIMIT %d)`,
			ragVectorRoute,
			markdownFileIDExpr,
			indexVersionExpr,
			chunkIndexScopeExpr,
			parentExpr,
			chunkStartExpr,
			chunkEndExpr,
			scoreExpr,
			ragQuoteIdentifier(table),
			strings.Join(baseWhere, "\n  AND "),
			maxHits,
		))
	}
	return fmt.Sprintf(`
SELECT *
FROM (
%s
) AS rag_vector_candidates
ORDER BY score ASC
LIMIT %d`,
		strings.Join(parts, "\nUNION ALL\n"),
		maxHits*len(embeddings),
	), nil
}

func ragFulltextMatchExpr(query string) string {
	return fmt.Sprintf("MATCH(content) AGAINST('%s' IN NATURAL LANGUAGE MODE)", ragEscapeSQLString(strings.TrimSpace(query)))
}

func ragSearchKeywards(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func ragCompactStrings(values []string) []string {
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

func mergeRAGCandidates(fulltext, vector *tools.SQLExecutionResult) ([]ragCandidate, error) {
	merged := make(map[string]*ragCandidate)
	appendRows := func(result *tools.SQLExecutionResult, route string, routeWeight float64) error {
		for idx, row := range result.Rows {
			record := ragRowRecord(result.Columns, row)
			hit, err := ragChunkHitFromRecord(record)
			if err != nil {
				return fmt.Errorf("search_rag_chunks %s row=%d: %w", route, idx, err)
			}
			if hit.FileID == "" {
				continue
			}
			if len(hit.Routes) == 0 {
				hit.Routes = []string{route}
			}
			key := ragChunkHitKey(hit)
			if existing, ok := merged[key]; ok {
				existing.hit.Routes = mergeRouteNames(existing.hit.Routes, hit.Routes)
				if hit.Score != 0 && (existing.hit.Score == 0 || route == ragFulltextRoute) {
					existing.hit.Score = hit.Score
				}
				existing.rank -= routeWeight
				continue
			}
			rank := float64(idx+1) - routeWeight
			merged[key] = &ragCandidate{hit: hit, rank: rank}
		}
		return nil
	}
	if err := appendRows(fulltext, ragFulltextRoute, 0.35); err != nil {
		return nil, err
	}
	if err := appendRows(vector, ragVectorRoute, 0.15); err != nil {
		return nil, err
	}
	out := make([]ragCandidate, 0, len(merged))
	for _, item := range merged {
		out = append(out, *item)
	}
	return out, nil
}

func ragCandidateLiteralMatchScore(hit tools.RAGChunkHit, keywards []string) int {
	score := 0
	fileName := strings.ToLower(hit.FileName)
	content := strings.ToLower(hit.Content)
	objectID := strings.ToLower(hit.ObjectID)
	for _, keyward := range keywards {
		keyward = strings.TrimSpace(keyward)
		if keyward == "" {
			continue
		}
		literal := strings.ToLower(keyward)
		if strings.Contains(fileName, literal) {
			score += 4
		}
		if strings.Contains(content, literal) {
			score += 2
		}
		if strings.Contains(objectID, literal) {
			score++
		}
	}
	if ragRoutesContain(hit.Routes, ragFulltextRoute) && score > 0 {
		score += 2
	}
	return score
}

func (s *searchRAGChunksService) planRAGExpansions(
	ctx context.Context,
	candidates []ragCandidate,
	before int,
	after int,
) ([]ragExpansionPlan, error) {
	log := ragRetrievalLogger(ctx)
	fileLatestVersion := make(map[string]string)
	for _, candidate := range candidates {
		fileID := strings.TrimSpace(candidate.hit.FileID)
		indexVersion := strings.TrimSpace(candidate.hit.IndexVersion)
		if fileID == "" || indexVersion == "" {
			continue
		}
		if current := fileLatestVersion[fileID]; current == "" || compareRAGIndexVersion(indexVersion, current) > 0 {
			fileLatestVersion[fileID] = indexVersion
		}
	}

	plans := make([]ragExpansionPlan, 0, len(candidates))
	for candidateIndex, candidate := range candidates {
		hit := candidate.hit
		fileID := strings.TrimSpace(hit.FileID)
		indexVersion := strings.TrimSpace(hit.IndexVersion)
		if latest := fileLatestVersion[fileID]; latest != "" && indexVersion != "" && indexVersion != latest {
			log.Info("search_rag_chunks expansion candidate skipped by stale index version",
				zap.String("operation", "expansion_candidate_skipped_stale_index"),
				zap.Int("candidate_index", candidateIndex),
				zap.String("file_id", fileID),
				zap.String("index_version", indexVersion),
				zap.String("latest_index_version", latest),
			)
			continue
		}
		start := hit.ParentIndex
		end := hit.ParentIndex
		blockUUID := strings.TrimSpace(hit.BlockUUID)
		tableLike := ragLooksLikeTableChunk(hit)
		if start == nil || end == nil {
			plans = append(plans, ragExpansionPlan{
				candidate:     candidate,
				minAnchorRank: candidate.hit.RetrievalAnchorRank,
			})
			continue
		}
		minParent := *start
		maxParent := *end
		sameParentOnly := blockUUID == "" && tableLike
		fileScopedTable := blockUUID == "" && tableLike &&
			strings.EqualFold(strings.TrimSpace(hit.ChunkIndexScope), "file") &&
			hit.ChunkIndex != nil
		if blockUUID == "" && !tableLike {
			minParent = *start - before
			if minParent < 0 {
				minParent = 0
			}
			forwardWindow := after
			if ragLooksLikeHeadingCandidate(hit) && forwardWindow < ragHeadingForwardWindow {
				forwardWindow = ragHeadingForwardWindow
			}
			maxParent = *end + forwardWindow
		}
		plans = append(plans, ragExpansionPlan{
			candidate:       candidate,
			minAnchorRank:   candidate.hit.RetrievalAnchorRank,
			hasRange:        true,
			blockUUID:       blockUUID,
			sameParentOnly:  sameParentOnly,
			fileScopedTable: fileScopedTable,
			start:           *start,
			end:             *end,
			minParent:       minParent,
			maxParent:       maxParent,
		})
	}
	if len(plans) <= 1 {
		return plans, nil
	}

	merged := make([]ragExpansionPlan, 0, len(plans))
	for _, plan := range plans {
		if strings.TrimSpace(plan.candidate.hit.IndexVersion) == "" || !plan.hasRange {
			merged = append(merged, plan)
			continue
		}
		mergedIntoExisting := false
		for i := range merged {
			existing := &merged[i]
			if !ragExpansionPlansCanMerge(*existing, plan) {
				continue
			}
			ragMergeExpansionPlan(existing, plan)
			mergedIntoExisting = true
			break
		}
		if !mergedIntoExisting {
			merged = append(merged, plan)
		}
	}
	sort.SliceStable(merged, func(i, j int) bool {
		leftRank := merged[i].minAnchorRank
		rightRank := merged[j].minAnchorRank
		if leftRank > 0 || rightRank > 0 {
			if leftRank > 0 && rightRank > 0 {
				if leftRank != rightRank {
					return leftRank < rightRank
				}
				return ragChunkHitKey(merged[i].candidate.hit) < ragChunkHitKey(merged[j].candidate.hit)
			}
			return leftRank > 0
		}
		if merged[i].candidate.rank != merged[j].candidate.rank {
			return merged[i].candidate.rank < merged[j].candidate.rank
		}
		return ragChunkHitKey(merged[i].candidate.hit) < ragChunkHitKey(merged[j].candidate.hit)
	})
	log.Info("search_rag_chunks expansion plans merged",
		zap.String("operation", "expansion_plans_merged"),
		zap.Int("input_plan_count", len(plans)),
		zap.Int("merged_plan_count", len(merged)),
	)
	return merged, nil
}

func (s *searchRAGChunksService) expandRAGPlan(
	ctx context.Context,
	scope tools.WorkspaceScope,
	table string,
	columns ragColumnSet,
	plan ragExpansionPlan,
	anchorRankByKey map[string]int,
) ([]tools.RAGChunkHit, []string, error) {
	stageStart := time.Now()
	log := ragRetrievalLogger(ctx)
	hit := plan.candidate.hit
	indexVersion := strings.TrimSpace(hit.IndexVersion)
	if indexVersion == "" || !plan.hasRange {
		hit.EvidenceGroupID = ragEvidenceGroupID(hit.FileID, indexVersion, hit.ParentIndex, hit.ParentIndex)
		log.Info("search_rag_chunks expansion skipped without range",
			zap.String("operation", "expand_no_range"),
			zap.String("file_id", hit.FileID),
			zap.String("index_version", indexVersion),
			zap.String("level", hit.Level),
			zap.String("parent_index", ragPtrString(hit.ParentIndex)),
			zap.String("chunk_index", ragPtrString(hit.ChunkIndex)),
			zap.Int64("stage_duration_ms", time.Since(stageStart).Milliseconds()),
		)
		return []tools.RAGChunkHit{hit}, nil, nil
	}
	if plan.fileScopedTable {
		return s.expandRAGFileScopedTablePlan(ctx, scope, table, columns, plan, anchorRankByKey, stageStart)
	}
	rangeWidth := plan.end - plan.start + 1
	log.Info("search_rag_chunks expansion range resolved",
		zap.String("operation", "expand_range_resolved"),
		zap.String("file_id", hit.FileID),
		zap.String("index_version", indexVersion),
		zap.String("level", hit.Level),
		zap.String("block_uuid", plan.blockUUID),
		zap.String("chunk_type", hit.ChunkType),
		zap.Bool("same_parent_only", plan.sameParentOnly),
		zap.Int("range_start", plan.start),
		zap.Int("range_end", plan.end),
		zap.Int("range_width", rangeWidth),
		zap.Int("min_parent", plan.minParent),
		zap.Int("max_parent", plan.maxParent),
	)
	parentExpr := ragTopLevelOrMetaStringExpr("parent_index", columns)
	chunkStartExpr := ragTopLevelOrMetaStringExpr("chunk_start", columns)
	chunkEndExpr := ragTopLevelOrMetaStringExpr("chunk_end", columns)
	chunkIndexExpr := ragTopLevelOrMetaStringExpr("chunk_index", columns)
	markdownFileIDExpr := ragMarkdownFileIDExpr(columns)
	expansionWhere := fmt.Sprintf(`
  AND CAST(%s AS SIGNED) BETWEEN %d AND %d`, parentExpr, plan.minParent, plan.maxParent)
	groupStart := plan.start
	groupEnd := plan.end
	if plan.blockUUID != "" && hit.Level == "chunk" {
		expansionWhere = fmt.Sprintf(`
  AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.block_uuid')) = '%s'`, ragEscapeSQLString(plan.blockUUID))
		if hit.ParentIndex != nil {
			groupStart = *hit.ParentIndex
			groupEnd = *hit.ParentIndex
		}
	}
	groupID := ragEvidenceGroupID(hit.FileID, indexVersion, &groupStart, &groupEnd)
	expansionSQL := fmt.Sprintf(`
SELECT
  level,
  content,
  meta,
  file_id,
  %s AS markdown_file_id,
  index_version,
  chunk_index,
  %s AS parent_index,
  %s AS chunk_start,
  %s AS chunk_end,
  1 AS score
FROM %s
WHERE COALESCE(disabled, 0) = 0
  AND file_id = '%s'
  AND index_version = '%s'
  AND level = 'chunk'
%s
ORDER BY
  CAST(%s AS SIGNED),
  CAST(%s AS SIGNED)`,
		markdownFileIDExpr,
		parentExpr,
		chunkStartExpr,
		chunkEndExpr,
		ragQuoteIdentifier(table),
		ragEscapeSQLString(hit.FileID),
		ragEscapeSQLString(indexVersion),
		expansionWhere,
		parentExpr,
		chunkIndexExpr,
	)
	sqls := []string{expansionSQL}
	queryStart := time.Now()
	exec, err := s.executor.ExecuteSQL(ctx, strings.TrimSpace(scope.DBName), expansionSQL)
	if err != nil {
		log.Error("search_rag_chunks expansion query failed",
			zap.String("operation", "expand_query_failed"),
			zap.String("file_id", hit.FileID),
			zap.String("index_version", indexVersion),
			zap.Int("range_start", plan.start),
			zap.Int("range_end", plan.end),
			zap.Int("range_width", rangeWidth),
			zap.Int("min_parent", plan.minParent),
			zap.Int("max_parent", plan.maxParent),
			zap.Int64("query_duration_ms", time.Since(queryStart).Milliseconds()),
			zap.Int64("stage_duration_ms", time.Since(stageStart).Milliseconds()),
			zap.Error(err),
		)
		return nil, nil, fmt.Errorf("search_rag_chunks expand file_id=%s index_version=%s range=%d-%d: %w", hit.FileID, indexVersion, plan.start, plan.end, err)
	}
	if exec == nil {
		return nil, nil, errors.New("search_rag_chunks expand: executor returned nil result")
	}
	out := make([]tools.RAGChunkHit, 0, len(exec.Rows))
	for idx, row := range exec.Rows {
		record := ragRowRecord(exec.Columns, row)
		item, err := ragChunkHitFromRecord(record)
		if err != nil {
			return nil, nil, fmt.Errorf("search_rag_chunks expand row=%d: %w", idx, err)
		}
		item.Routes = append([]string(nil), hit.Routes...)
		item.Score = hit.Score
		item.EvidenceGroupID = groupID
		item.RetrievalAnchorRank = anchorRankByKey[ragChunkHitKey(item)]
		if plan.blockUUID != "" && strings.TrimSpace(item.BlockUUID) != plan.blockUUID {
			continue
		}
		if plan.blockUUID == "" && item.ParentIndex != nil && (*item.ParentIndex < plan.minParent || *item.ParentIndex > plan.maxParent) {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		hit.EvidenceGroupID = groupID
		out = append(out, hit)
	}
	log.Info("search_rag_chunks expansion completed",
		zap.String("operation", "expand_completed"),
		zap.String("file_id", hit.FileID),
		zap.String("index_version", indexVersion),
		zap.String("level", hit.Level),
		zap.String("block_uuid", plan.blockUUID),
		zap.String("chunk_type", hit.ChunkType),
		zap.Bool("same_parent_only", plan.sameParentOnly),
		zap.Int("range_start", plan.start),
		zap.Int("range_end", plan.end),
		zap.Int("range_width", rangeWidth),
		zap.Int("min_parent", plan.minParent),
		zap.Int("max_parent", plan.maxParent),
		zap.Int("sql_row_count", len(exec.Rows)),
		zap.Int("returned_row_count", len(out)),
		zap.Int("returned_content_chars", ragChunkHitsContentChars(out)),
		zap.Int64("query_duration_ms", time.Since(queryStart).Milliseconds()),
		zap.Int64("stage_duration_ms", time.Since(stageStart).Milliseconds()),
	)
	return out, sqls, nil
}

func (s *searchRAGChunksService) expandRAGFileScopedTablePlan(
	ctx context.Context,
	scope tools.WorkspaceScope,
	table string,
	columns ragColumnSet,
	plan ragExpansionPlan,
	anchorRankByKey map[string]int,
	stageStart time.Time,
) ([]tools.RAGChunkHit, []string, error) {
	log := ragRetrievalLogger(ctx)
	hit := plan.candidate.hit
	indexVersion := strings.TrimSpace(hit.IndexVersion)
	chunkIndexExpr := ragQuoteIdentifier("chunk_index")
	chunkStartExpr := ragMetaStringExpr("chunk_start")
	chunkEndExpr := ragMetaStringExpr("chunk_end")
	chunkIndexScopeExpr := ragMetaStringExpr("chunk_index_scope")
	sectionIDExpr := ragMetaStringExpr("section_id")

	anchor := hit.ChunkIndex
	if anchor == nil {
		return nil, nil, fmt.Errorf("search_rag_chunks expand file-scoped table missing chunk_index file_id=%s index_version=%s", hit.FileID, indexVersion)
	}
	sectionSQL := fmt.Sprintf(`
SELECT
  %s AS section_id,
  %s AS chunk_start,
  %s AS chunk_end
FROM %s
WHERE COALESCE(disabled, 0) = 0
  AND file_id = '%s'
  AND index_version = '%s'
  AND level = 'section'
  AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.chunk_index_scope')) = 'file'
  AND CAST(%s AS SIGNED) <= %d
  AND CAST(%s AS SIGNED) >= %d
ORDER BY
  CAST(%s AS SIGNED) - CAST(%s AS SIGNED) ASC
LIMIT 1`,
		sectionIDExpr,
		chunkStartExpr,
		chunkEndExpr,
		ragQuoteIdentifier(table),
		ragEscapeSQLString(hit.FileID),
		ragEscapeSQLString(indexVersion),
		chunkStartExpr,
		*anchor,
		chunkEndExpr,
		*anchor,
		chunkEndExpr,
		chunkStartExpr,
	)
	queryStart := time.Now()
	sectionExec, err := s.executor.ExecuteSQL(ctx, strings.TrimSpace(scope.DBName), sectionSQL)
	if err != nil {
		return nil, nil, fmt.Errorf("search_rag_chunks expand file-scoped section file_id=%s index_version=%s chunk_index=%d: %w", hit.FileID, indexVersion, *anchor, err)
	}
	if sectionExec == nil {
		return nil, nil, errors.New("search_rag_chunks expand file-scoped section: executor returned nil result")
	}
	if len(sectionExec.Rows) == 0 {
		return nil, nil, fmt.Errorf("search_rag_chunks expand file-scoped section not found file_id=%s index_version=%s chunk_index=%d", hit.FileID, indexVersion, *anchor)
	}
	sectionRecord := ragRowRecord(sectionExec.Columns, sectionExec.Rows[0])
	start := ragRowIntPtr(sectionRecord, "chunk_start")
	end := ragRowIntPtr(sectionRecord, "chunk_end")
	if start == nil || end == nil {
		return nil, nil, fmt.Errorf("search_rag_chunks expand file-scoped section missing chunk_start/chunk_end")
	}
	groupID := ragEvidenceGroupID(hit.FileID, indexVersion, start, end)
	markdownFileIDExpr := ragMarkdownFileIDExpr(columns)
	expansionSQL := fmt.Sprintf(`
SELECT
  level,
  content,
  meta,
  file_id,
  %s AS markdown_file_id,
  index_version,
  chunk_index,
  %s AS chunk_index_scope,
  %s AS parent_index,
  %s AS chunk_start,
  %s AS chunk_end,
  1 AS score
FROM %s
WHERE COALESCE(disabled, 0) = 0
  AND file_id = '%s'
  AND index_version = '%s'
  AND level = 'chunk'
  AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.chunk_index_scope')) = 'file'
  AND CAST(%s AS SIGNED) BETWEEN %d AND %d
ORDER BY
  CAST(%s AS SIGNED)`,
		markdownFileIDExpr,
		chunkIndexScopeExpr,
		ragTopLevelOrMetaStringExpr("parent_index", columns),
		chunkStartExpr,
		chunkEndExpr,
		ragQuoteIdentifier(table),
		ragEscapeSQLString(hit.FileID),
		ragEscapeSQLString(indexVersion),
		chunkIndexExpr,
		*start,
		*end,
		chunkIndexExpr,
	)
	exec, err := s.executor.ExecuteSQL(ctx, strings.TrimSpace(scope.DBName), expansionSQL)
	if err != nil {
		return nil, nil, fmt.Errorf("search_rag_chunks expand file-scoped table file_id=%s index_version=%s range=%d-%d: %w", hit.FileID, indexVersion, *start, *end, err)
	}
	if exec == nil {
		return nil, nil, errors.New("search_rag_chunks expand file-scoped table: executor returned nil result")
	}
	out := make([]tools.RAGChunkHit, 0, len(exec.Rows))
	for idx, row := range exec.Rows {
		record := ragRowRecord(exec.Columns, row)
		item, err := ragChunkHitFromRecord(record)
		if err != nil {
			return nil, nil, fmt.Errorf("search_rag_chunks expand file-scoped table row=%d: %w", idx, err)
		}
		item.Routes = append([]string(nil), hit.Routes...)
		item.Score = hit.Score
		item.EvidenceGroupID = groupID
		item.RetrievalAnchorRank = anchorRankByKey[ragChunkHitKey(item)]
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil, nil, fmt.Errorf("search_rag_chunks expand file-scoped table returned no rows file_id=%s index_version=%s range=%d-%d", hit.FileID, indexVersion, *start, *end)
	}
	log.Info("search_rag_chunks file-scoped table expansion completed",
		zap.String("operation", "expand_file_scoped_table_completed"),
		zap.String("file_id", hit.FileID),
		zap.String("index_version", indexVersion),
		zap.Int("range_start", *start),
		zap.Int("range_end", *end),
		zap.Int("sql_row_count", len(exec.Rows)),
		zap.Int("returned_row_count", len(out)),
		zap.Int64("query_duration_ms", time.Since(queryStart).Milliseconds()),
		zap.Int64("stage_duration_ms", time.Since(stageStart).Milliseconds()),
	)
	return out, []string{sectionSQL, expansionSQL}, nil
}

func ragChunkHitFromRecord(record map[string]any) (tools.RAGChunkHit, error) {
	meta, err := ragRowMeta(record["meta"])
	if err != nil {
		return tools.RAGChunkHit{}, err
	}
	hit := tools.RAGChunkHit{
		Content: strings.TrimSpace(ragAnyString(record["content"])),
		StartMS: ragIntPtrToInt64(ragMetaIntPtr(meta, "start_ms")),
		EndMS:   ragIntPtrToInt64(ragMetaIntPtr(meta, "end_ms")),
		FileID:  firstRAGNonEmpty(ragRowString(record, "file_id"), ragMetaString(meta, "source_file_id"), ragMetaString(meta, "file_id")),
		MarkdownFileID: firstRAGNonEmpty(
			ragRowString(record, "markdown_file_id"),
			ragRowString(record, "md_file_id"),
			ragMetaString(meta, "markdown_file_id"),
			ragMetaString(meta, "md_file_id"),
		),
		FileName:     firstRAGNonEmpty(ragRowString(record, "file_name"), ragMetaString(meta, "source_file_name"), ragMetaString(meta, "file_name")),
		PageNumber:   firstPositiveIntLocal(ragRowIntValue(record, "page_num"), ragRowIntValue(record, "page_number"), ragMetaIntValue(meta, "page_num"), ragMetaIntValue(meta, "page_number")),
		VolumeID:     firstRAGNonEmpty(ragRowString(record, "volume_id"), ragMetaString(meta, "source_volume_id"), ragMetaString(meta, "volume_id")),
		SourceURI:    firstRAGNonEmpty(ragRowString(record, "source_uri"), ragMetaString(meta, "source_uri"), ragMetaString(meta, "file_url"), ragMetaString(meta, "presign_url")),
		IndexFileID:  firstRAGNonEmpty(ragRowString(record, "file_id"), ragMetaString(meta, "file_id")),
		IndexVersion: firstRAGNonEmpty(ragRowString(record, "index_version"), ragMetaString(meta, "index_version")),
		ChunkID:      firstRAGNonEmpty(ragRowString(record, "chunk_id"), ragRowString(record, "id"), ragMetaString(meta, "chunk_id"), ragMetaString(meta, "section_id")),
		Level:        strings.ToLower(strings.TrimSpace(ragRowString(record, "level"))),
		ChunkIndex:   firstRAGIntPtr(ragRowIntPtr(record, "chunk_index"), ragMetaIntPtr(meta, "chunk_index")),
		ChunkIndexScope: firstRAGNonEmpty(
			ragRowString(record, "chunk_index_scope"),
			ragMetaString(meta, "chunk_index_scope"),
		),
		ParentIndex:        firstRAGIntPtr(ragRowIntPtr(record, "parent_index"), ragMetaIntPtr(meta, "parent_index")),
		ChunkStart:         firstRAGIntPtr(ragRowIntPtr(record, "chunk_start"), ragMetaIntPtr(meta, "chunk_start"), ragMetaIntPtr(meta, "split_idx_start")),
		ChunkEnd:           firstRAGIntPtr(ragRowIntPtr(record, "chunk_end"), ragMetaIntPtr(meta, "chunk_end")),
		Score:              firstPositiveFloatLocal(ragRowFloatValue(record, "score"), ragRowFloatValue(record, "distance")),
		BlockUUID:          ragMetaString(meta, "block_uuid"),
		ChunkType:          ragMetaString(meta, "chunk_type"),
		Scope:              ragMetaString(meta, "scope"),
		ObjectID:           ragMetaString(meta, "object_id"),
		ObjectKind:         ragMetaString(meta, "object_kind"),
		ImageFileID:        ragImageFileID(meta),
		PageImageFileID:    ragMetaString(meta, "page_image_file_id"),
		BBox:               ragMetaFloat64Slice(meta, "bbox"),
		EmbeddedBlockUUIDs: ragMetaStringSlice(meta, "embedded_block_uuids"),
	}
	route := strings.TrimSpace(ragRowString(record, "route"))
	if route != "" {
		if route == "vector" {
			route = "vector_l2"
		}
		hit.Routes = []string{route}
	}
	if hit.ChunkID == "" {
		hit.ChunkID = ragSynthesizedChunkID(hit)
	}
	return hit, nil
}

func ragImageFileID(meta map[string]any) string {
	if fileID := firstRAGNonEmpty(ragMetaString(meta, "image_file_id"), ragMetaString(meta, "visual_image_file_id")); fileID != "" {
		return fileID
	}
	for _, key := range []string{"image_url", "s3_image_url", "table_image_url"} {
		if fileID := ragLocalImageFileID(ragMetaString(meta, key)); fileID != "" {
			return fileID
		}
	}
	return ""
}

func ragLocalImageFileID(ref string) string {
	match := ragCatalogImageRefPattern.FindStringSubmatch(ref)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func ragIntPtrToInt64(value *int) *int64 {
	if value == nil {
		return nil
	}
	converted := int64(*value)
	return &converted
}

func ragEmbeddedImageBlockUUIDs(hit tools.RAGChunkHit) []string {
	contentTokens := ragImageTokensFromContent(hit.Content)
	if len(contentTokens) > 0 {
		return ragCompactStrings(contentTokens)
	}
	return ragCompactStrings(hit.EmbeddedBlockUUIDs)
}

func ragImageTokensFromContent(content string) []string {
	const prefix = "[image:"
	values := make([]string, 0)
	rest := content
	for {
		start := strings.Index(rest, prefix)
		if start < 0 {
			return values
		}
		rest = rest[start+len(prefix):]
		end := strings.Index(rest, "]")
		if end < 0 {
			return values
		}
		token := strings.TrimSpace(rest[:end])
		if token != "" {
			values = append(values, token)
		}
		rest = rest[end+1:]
	}
}

func ragImageRefFromChunk(hit tools.RAGChunkHit) (tools.RAGImageRef, bool) {
	ref := tools.RAGImageRef{
		ChunkID:         strings.TrimSpace(hit.ChunkID),
		ObjectID:        strings.TrimSpace(firstRAGNonEmpty(hit.ObjectID, hit.BlockUUID)),
		ObjectKind:      strings.TrimSpace(hit.ObjectKind),
		ImageFileID:     strings.TrimSpace(hit.ImageFileID),
		PageImageFileID: strings.TrimSpace(hit.PageImageFileID),
		Page:            hit.PageNumber,
		BBox:            append([]float64(nil), hit.BBox...),
	}
	if ref.ObjectID == "" && ref.ObjectKind == "" && ref.ImageFileID == "" && ref.PageImageFileID == "" && len(ref.BBox) == 0 {
		return tools.RAGImageRef{}, false
	}
	return ref, true
}

func compactRAGImageRefs(values []tools.RAGImageRef) []tools.RAGImageRef {
	out := make([]tools.RAGImageRef, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value.ChunkID = strings.TrimSpace(value.ChunkID)
		value.ObjectID = strings.TrimSpace(value.ObjectID)
		value.ObjectKind = strings.TrimSpace(value.ObjectKind)
		value.ImageFileID = strings.TrimSpace(value.ImageFileID)
		value.PageImageFileID = strings.TrimSpace(value.PageImageFileID)
		if value.ChunkID == "" && value.ObjectID == "" && value.ImageFileID == "" && value.PageImageFileID == "" && len(value.BBox) == 0 {
			continue
		}
		if len(value.BBox) > 0 {
			value.BBox = append([]float64(nil), value.BBox...)
		}
		key := strings.Join([]string{
			strconv.FormatInt(value.SemanticModelID, 10),
			value.ChunkID,
			value.ObjectID,
			value.ObjectKind,
			value.ImageFileID,
			value.PageImageFileID,
			fmt.Sprint(value.Page),
			fmt.Sprint(value.BBox),
		}, "\x00")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func ragFileNameExpr() string {
	return "COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.source_file_name')), ''), JSON_UNQUOTE(JSON_EXTRACT(meta, '$.file_name')))"
}

func ragVolumeExpr(columns ragColumnSet) string {
	parts := []string{
		"NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.source_volume_id')), '')",
		"JSON_UNQUOTE(JSON_EXTRACT(meta, '$.volume_id'))",
	}
	if columns.has("volume_id") {
		parts = append(parts, "CAST(volume_id AS CHAR)")
	}
	return "COALESCE(" + strings.Join(parts, ", ") + ")"
}

func ragSourceURIExpr() string {
	return "COALESCE(NULLIF(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.source_uri')), ''), JSON_UNQUOTE(JSON_EXTRACT(meta, '$.file_url')), JSON_UNQUOTE(JSON_EXTRACT(meta, '$.presign_url')))"
}

func ragMarkdownFileIDExpr(columns ragColumnSet) string {
	if columns.has("markdown_file_id") {
		return "markdown_file_id"
	}
	if columns.has("md_file_id") {
		return "md_file_id"
	}
	return "''"
}

func ragIndexVersionSelectExpr(columns ragColumnSet) string {
	if columns.has("index_version") {
		return "index_version"
	}
	return "NULL AS index_version"
}

func ragTopLevelOrMetaStringExpr(name string, columns ragColumnSet) string {
	meta := fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(meta, '$.%s'))", ragEscapeJSONPathKey(name))
	if !columns.has(name) {
		return meta
	}
	return fmt.Sprintf("COALESCE(NULLIF(CAST(%s AS CHAR), ''), %s)", ragQuoteIdentifier(name), meta)
}

func ragMetaStringExpr(name string) string {
	return fmt.Sprintf("JSON_UNQUOTE(JSON_EXTRACT(meta, '$.%s'))", ragEscapeJSONPathKey(name))
}

func ragVectorLiteral(vector []float64) (string, error) {
	for _, v := range vector {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return "", fmt.Errorf("non-finite vector value %v", v)
		}
	}
	raw, err := json.Marshal(vector)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func ragRowRecord(columns []string, row []any) map[string]any {
	record := make(map[string]any, len(columns))
	for i, col := range columns {
		if i >= len(row) {
			break
		}
		record[normalizeRAGColumnName(col)] = row[i]
	}
	return record
}

func normalizeRAGColumnName(col string) string {
	col = strings.TrimSpace(col)
	col = strings.Trim(col, "`\" ")
	if dot := strings.LastIndexByte(col, '.'); dot >= 0 && dot+1 < len(col) {
		col = col[dot+1:]
	}
	return strings.ToLower(strings.Trim(col, "`\" "))
}

func ragRowMeta(raw any) (map[string]any, error) {
	if raw == nil {
		return nil, nil
	}
	switch v := raw.(type) {
	case map[string]any:
		return v, nil
	case []byte:
		return parseRAGMeta(string(v))
	case string:
		return parseRAGMeta(v)
	default:
		text := strings.TrimSpace(fmt.Sprint(v))
		if text == "" || text == "<nil>" {
			return nil, nil
		}
		return parseRAGMeta(text)
	}
}

func parseRAGMeta(text string) (map[string]any, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(text), &meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func ragRowString(record map[string]any, key string) string {
	return strings.TrimSpace(ragAnyString(record[normalizeRAGColumnName(key)]))
}

func ragAnyString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	case json.Number:
		return t.String()
	case time.Time:
		return t.Format(time.RFC3339Nano)
	default:
		return fmt.Sprint(t)
	}
}

func ragMetaString(meta map[string]any, key string) string {
	if meta == nil {
		return ""
	}
	return strings.TrimSpace(ragAnyString(meta[key]))
}

func ragMetaStringSlice(meta map[string]any, key string) []string {
	if meta == nil {
		return nil
	}
	raw, ok := meta[key]
	if !ok {
		return nil
	}
	switch values := raw.(type) {
	case []any:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s := strings.TrimSpace(ragAnyString(value)); s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		out := make([]string, 0, len(values))
		for _, value := range values {
			if s := strings.TrimSpace(value); s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		text := strings.TrimSpace(values)
		if text == "" {
			return nil
		}
		var arrayValues []string
		if err := json.Unmarshal([]byte(text), &arrayValues); err == nil {
			return ragCompactStrings(arrayValues)
		}
		var anyValues []any
		if err := json.Unmarshal([]byte(text), &anyValues); err == nil {
			out := make([]string, 0, len(anyValues))
			for _, value := range anyValues {
				if s := strings.TrimSpace(ragAnyString(value)); s != "" {
					out = append(out, s)
				}
			}
			return out
		}
		return []string{text}
	default:
		return nil
	}
}

func ragRowIntPtr(record map[string]any, key string) *int {
	return ragIntPtr(record[normalizeRAGColumnName(key)])
}

func ragMetaIntPtr(meta map[string]any, key string) *int {
	if meta == nil {
		return nil
	}
	return ragIntPtr(meta[key])
}

func ragRowIntValue(record map[string]any, key string) int {
	if p := ragRowIntPtr(record, key); p != nil {
		return *p
	}
	return 0
}

func ragMetaIntValue(meta map[string]any, key string) int {
	if p := ragMetaIntPtr(meta, key); p != nil {
		return *p
	}
	return 0
}

func ragMetaFloat64Slice(meta map[string]any, key string) []float64 {
	if meta == nil {
		return nil
	}
	values, ok := meta[key].([]any)
	if !ok {
		return nil
	}
	out := make([]float64, 0, len(values))
	for _, value := range values {
		n, ok := ragAnyFloat64(value)
		if !ok {
			return nil
		}
		out = append(out, n)
	}
	return out
}

func ragAnyFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case json.Number:
		f, err := strconv.ParseFloat(v.String(), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func ragIntPtr(v any) *int {
	switch t := v.(type) {
	case nil:
		return nil
	case int:
		return &t
	case int8:
		n := int(t)
		return &n
	case int16:
		n := int(t)
		return &n
	case int32:
		n := int(t)
		return &n
	case int64:
		n := int(t)
		return &n
	case uint:
		n := int(t)
		return &n
	case uint8:
		n := int(t)
		return &n
	case uint16:
		n := int(t)
		return &n
	case uint32:
		n := int(t)
		return &n
	case uint64:
		n := int(t)
		return &n
	case float32:
		n := int(t)
		if float32(n) == t {
			return &n
		}
	case float64:
		n := int(t)
		if float64(n) == t {
			return &n
		}
	case json.Number:
		if n, err := strconv.Atoi(t.String()); err == nil {
			return &n
		}
		if f, err := strconv.ParseFloat(t.String(), 64); err == nil {
			n := int(f)
			if float64(n) == f {
				return &n
			}
		}
	case []byte:
		return ragIntPtr(strings.TrimSpace(string(t)))
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return nil
		}
		if n, err := strconv.Atoi(s); err == nil {
			return &n
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			n := int(f)
			if float64(n) == f {
				return &n
			}
		}
	}
	return nil
}

func ragRowFloatValue(record map[string]any, key string) float64 {
	return ragFloatValue(record[normalizeRAGColumnName(key)])
}

func ragFloatValue(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case json.Number:
		f, _ := strconv.ParseFloat(t.String(), 64)
		return f
	case string:
		f, _ := strconv.ParseFloat(strings.TrimSpace(strings.ReplaceAll(t, ",", "")), 64)
		return f
	case []byte:
		f, _ := strconv.ParseFloat(strings.TrimSpace(strings.ReplaceAll(string(t), ",", "")), 64)
		return f
	default:
		return 0
	}
}

func firstRAGNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func firstRAGIntPtr(values ...*int) *int {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

func firstPositiveIntLocal(values ...int) int {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}

func firstPositiveFloatLocal(values ...float64) float64 {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}

func mergeRouteNames(left, right []string) []string {
	seen := make(map[string]struct{}, len(left)+len(right))
	out := make([]string, 0, len(left)+len(right))
	for _, route := range append(append([]string(nil), left...), right...) {
		route = strings.TrimSpace(route)
		if route == "" {
			continue
		}
		if _, ok := seen[route]; ok {
			continue
		}
		seen[route] = struct{}{}
		out = append(out, route)
	}
	return out
}

func ragChunkHitKey(hit tools.RAGChunkHit) string {
	parts := []string{
		strconv.FormatInt(hit.SemanticModelID, 10),
		hit.FileID,
		hit.IndexVersion,
		hit.Level,
		ragPtrString(hit.ParentIndex),
		ragPtrString(hit.ChunkIndex),
		hit.ChunkID,
	}
	if strings.Join(parts, "") == "" {
		parts = append(parts, truncateRunSQLCell(hit.Content, 80))
	}
	return strings.Join(parts, "|")
}

func truncateRunSQLCell(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + fmt.Sprintf("...<truncated chars=%d>", len(runes)-maxRunes)
}

func ragLooksLikeTableChunk(hit tools.RAGChunkHit) bool {
	if strings.EqualFold(strings.TrimSpace(hit.ChunkType), "table") {
		return true
	}
	content := strings.ToLower(hit.Content)
	return strings.Contains(content, "<table") ||
		strings.Contains(content, "<tr") ||
		strings.Contains(content, "<td") ||
		strings.Contains(content, "</td>")
}

type ragTableEvidenceCell struct {
	html string
	text string
}

type ragTableEvidenceRow struct {
	cells []ragTableEvidenceCell
	text  string
	index int
}

type ragActiveTableCell struct {
	cell      ragTableEvidenceCell
	column    int
	remaining int
}

func focusRAGTableEvidenceRows(group []tools.RAGChunkHit, keywards []string) []tools.RAGChunkHit {
	if len(group) == 0 {
		return group
	}
	tableIndexes := make([]int, 0, len(group))
	tableRunes := 0
	for idx, hit := range group {
		if !ragLooksLikeTableChunk(hit) {
			continue
		}
		tableIndexes = append(tableIndexes, idx)
		tableRunes += utf8RuneCount(hit.Content)
	}
	if len(tableIndexes) == 0 || tableRunes < ragFocusedTableMinRunes {
		return group
	}
	rankedTableCount := 0
	for _, idx := range tableIndexes {
		if group[idx].RetrievalAnchorRank > 0 {
			rankedTableCount++
		}
	}
	if rankedTableCount == 0 {
		if len(tableIndexes) > 1 {
			return group
		}
		tableIndex := tableIndexes[0]
		focused := focusedRAGTableRowsHTML(group[tableIndex].Content, keywards)
		if focused == "" {
			return group
		}
		out := append([]tools.RAGChunkHit(nil), group...)
		out[tableIndex].Content = focused
		return out
	}

	tableIndexSet := make(map[int]struct{}, len(tableIndexes))
	for _, idx := range tableIndexes {
		tableIndexSet[idx] = struct{}{}
	}
	out := make([]tools.RAGChunkHit, 0, len(group)-len(tableIndexes)+rankedTableCount)
	for idx, hit := range group {
		if _, isTable := tableIndexSet[idx]; !isTable {
			out = append(out, hit)
			continue
		}
		if hit.RetrievalAnchorRank > 0 {
			if focused := focusedRAGTableRowsHTML(hit.Content, keywards); focused != "" {
				hit.Content = focused
			}
			out = append(out, hit)
		}
	}
	return out
}

func focusedRAGTableRowsHTML(content string, keywards []string) string {
	rows := extractRAGTableRows(content)
	if len(rows) == 0 {
		return ""
	}
	type scoredRow struct {
		row   ragTableEvidenceRow
		score int
	}
	scored := make([]scoredRow, 0, len(rows))
	for _, row := range rows {
		score := ragTableRowMatchScore(row, keywards)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredRow{row: row, score: score})
	}
	if len(scored) == 0 {
		return ""
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].row.index < scored[j].row.index
	})
	if len(scored) > ragFocusedTableMaxRows {
		scored = scored[:ragFocusedTableMaxRows]
	}
	selected := make([]ragTableEvidenceRow, 0, len(scored))
	for _, item := range scored {
		selected = append(selected, item.row)
	}
	return renderRAGTableRows(selected)
}

func extractRAGTableRows(content string) []ragTableEvidenceRow {
	root, err := xhtml.Parse(strings.NewReader("<table>" + content + "</table>"))
	if err != nil {
		return nil
	}
	trNodes := make([]*xhtml.Node, 0)
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n == nil {
			return
		}
		if n.Type == xhtml.ElementNode && strings.EqualFold(n.Data, "tr") {
			trNodes = append(trNodes, n)
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)

	rows := make([]ragTableEvidenceRow, 0, len(trNodes))
	active := make([]ragActiveTableCell, 0)
	for _, tr := range trNodes {
		cellsByColumn := make(map[int]ragTableEvidenceCell)
		nextActive := make([]ragActiveTableCell, 0, len(active))
		for _, item := range active {
			cellsByColumn[item.column] = item.cell
			item.remaining--
			if item.remaining > 0 {
				nextActive = append(nextActive, item)
			}
		}
		active = nextActive

		column := 0
		for child := tr.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != xhtml.ElementNode || (!strings.EqualFold(child.Data, "td") && !strings.EqualFold(child.Data, "th")) {
				continue
			}
			for {
				if _, occupied := cellsByColumn[column]; !occupied {
					break
				}
				column++
			}
			cell := ragTableEvidenceCell{
				html: renderRAGHTMLChildren(child),
				text: textFromRAGHTMLNode(child),
			}
			cellsByColumn[column] = cell
			if rowspan := ragHTMLAttrPositiveInt(child, "rowspan"); rowspan > 1 {
				active = append(active, ragActiveTableCell{cell: cell, column: column, remaining: rowspan - 1})
			}
			column++
		}
		if len(cellsByColumn) == 0 {
			continue
		}
		columns := make([]int, 0, len(cellsByColumn))
		for column := range cellsByColumn {
			columns = append(columns, column)
		}
		sort.Ints(columns)
		cells := make([]ragTableEvidenceCell, 0, len(columns))
		for _, column := range columns {
			cells = append(cells, cellsByColumn[column])
		}
		textParts := make([]string, 0, len(cells))
		for _, cell := range cells {
			if text := strings.TrimSpace(cell.text); text != "" {
				textParts = append(textParts, text)
			}
		}
		rows = append(rows, ragTableEvidenceRow{
			cells: cells,
			text:  strings.Join(textParts, " "),
			index: len(rows),
		})
	}
	return rows
}

func ragTableRowMatchScore(row ragTableEvidenceRow, keywards []string) int {
	text := strings.ToLower(row.text)
	score := 0
	for _, keyward := range keywards {
		keyward = strings.TrimSpace(keyward)
		if keyward == "" {
			continue
		}
		if strings.Contains(text, strings.ToLower(keyward)) {
			score += 2
		}
	}
	return score
}

func renderRAGTableRows(rows []ragTableEvidenceRow) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<table>")
	for _, row := range rows {
		b.WriteString("<tr>")
		for _, cell := range row.cells {
			b.WriteString("<td>")
			b.WriteString(cell.html)
			b.WriteString("</td>")
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</table>")
	return b.String()
}

func renderRAGHTMLChildren(n *xhtml.Node) string {
	var b bytes.Buffer
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if err := xhtml.Render(&b, child); err != nil {
			return strings.TrimSpace(textFromRAGHTMLNode(n))
		}
	}
	return strings.TrimSpace(b.String())
}

func textFromRAGHTMLNode(n *xhtml.Node) string {
	var parts []string
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node == nil {
			return
		}
		if node.Type == xhtml.TextNode {
			if text := strings.TrimSpace(node.Data); text != "" {
				parts = append(parts, text)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return strings.Join(parts, " ")
}

func ragHTMLAttrPositiveInt(n *xhtml.Node, name string) int {
	for _, attr := range n.Attr {
		if !strings.EqualFold(attr.Key, name) {
			continue
		}
		value, err := strconv.Atoi(strings.TrimSpace(attr.Val))
		if err != nil || value <= 0 {
			return 0
		}
		return value
	}
	return 0
}

func ragLooksLikeHeadingCandidate(hit tools.RAGChunkHit) bool {
	if !ragRoutesContain(hit.Routes, ragFulltextRoute) {
		return false
	}
	if ragLooksLikeTableChunk(hit) {
		return false
	}
	chunkType := strings.ToLower(strings.TrimSpace(hit.ChunkType))
	if chunkType != "" && chunkType != "text" && chunkType != "list" {
		return false
	}
	content := strings.TrimSpace(hit.Content)
	if content == "" || utf8RuneCount(content) > ragHeadingMaxRunes {
		return false
	}
	return true
}

func ragRoutesContain(routes []string, route string) bool {
	route = strings.TrimSpace(route)
	for _, item := range routes {
		if strings.TrimSpace(item) == route {
			return true
		}
	}
	return false
}

func utf8RuneCount(text string) int {
	count := 0
	for range text {
		count++
	}
	return count
}

func ragSynthesizedChunkID(hit tools.RAGChunkHit) string {
	parts := []string{hit.FileID, hit.IndexVersion, hit.Level, ragPtrString(hit.ParentIndex), ragPtrString(hit.ChunkIndex)}
	return strings.Trim(strings.Join(parts, ":"), ":")
}

func ragEvidenceGroupID(fileID, indexVersion string, start, end *int) string {
	return strings.Join([]string{"search_rag_chunks", strings.TrimSpace(fileID), strings.TrimSpace(indexVersion), ragPtrString(start), ragPtrString(end)}, ":")
}

func compareRAGIndexVersion(left, right string) int {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == right {
		return 0
	}
	leftInt, leftErr := strconv.ParseInt(left, 10, 64)
	rightInt, rightErr := strconv.ParseInt(right, 10, 64)
	if leftErr == nil && rightErr == nil {
		switch {
		case leftInt > rightInt:
			return 1
		case leftInt < rightInt:
			return -1
		default:
			return 0
		}
	}
	return strings.Compare(left, right)
}

func ragExpansionPlansCanMerge(left, right ragExpansionPlan) bool {
	if strings.TrimSpace(left.candidate.hit.FileID) != strings.TrimSpace(right.candidate.hit.FileID) {
		return false
	}
	if strings.TrimSpace(left.candidate.hit.IndexVersion) != strings.TrimSpace(right.candidate.hit.IndexVersion) {
		return false
	}
	if strings.TrimSpace(left.blockUUID) != "" || strings.TrimSpace(right.blockUUID) != "" {
		return strings.TrimSpace(left.blockUUID) != "" && strings.TrimSpace(left.blockUUID) == strings.TrimSpace(right.blockUUID)
	}
	if left.sameParentOnly || right.sameParentOnly {
		return left.sameParentOnly &&
			right.sameParentOnly &&
			left.fileScopedTable == right.fileScopedTable &&
			left.minParent == right.minParent &&
			left.maxParent == right.maxParent
	}
	if left.fileScopedTable != right.fileScopedTable {
		return false
	}
	return left.minParent <= right.maxParent && right.minParent <= left.maxParent
}

func ragMergeExpansionPlan(left *ragExpansionPlan, right ragExpansionPlan) {
	if left == nil {
		return
	}
	left.minAnchorRank = ragBestRetrievalAnchorRank(left.minAnchorRank, right.minAnchorRank)
	if right.candidate.rank < left.candidate.rank {
		left.candidate = right.candidate
	}
	if right.start < left.start {
		left.start = right.start
	}
	if right.end > left.end {
		left.end = right.end
	}
	if right.minParent < left.minParent {
		left.minParent = right.minParent
	}
	if right.maxParent > left.maxParent {
		left.maxParent = right.maxParent
	}
	left.sameParentOnly = left.sameParentOnly && right.sameParentOnly
	left.fileScopedTable = left.fileScopedTable && right.fileScopedTable
}

func ragRetrievalLogger(ctx context.Context) *zap.Logger {
	return corelog.WithContext(ctx, nil).With(zap.String("component", "agent_tools.knowledge.rag_retrieval"))
}

func ragChunkHitsContentChars(hits []tools.RAGChunkHit) int {
	total := 0
	for _, hit := range hits {
		total += len(hit.Content)
	}
	return total
}

func ragPtrString(v *int) string {
	if v == nil {
		return ""
	}
	return strconv.Itoa(*v)
}

func ragBestRetrievalAnchorRank(current, candidate int) int {
	if candidate <= 0 {
		return current
	}
	if current <= 0 || candidate < current {
		return candidate
	}
	return current
}

func ragQuotedList(values []string) string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, "'"+ragEscapeSQLString(value)+"'")
	}
	if len(out) == 0 {
		return "''"
	}
	return strings.Join(out, ", ")
}

func ragIntList(values []int) string {
	if len(values) == 0 {
		return "0"
	}
	out := make([]string, 0, len(values))
	previous := 0
	hasPrevious := false
	for _, value := range values {
		if hasPrevious && value == previous {
			continue
		}
		out = append(out, strconv.Itoa(value))
		previous = value
		hasPrevious = true
	}
	if len(out) == 0 {
		return "0"
	}
	return strings.Join(out, ", ")
}

func ragQuoteIdentifier(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, "`\"")
	if name == "" {
		return ""
	}
	parts := strings.Split(name, ".")
	quoted := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.Trim(part, "`\""))
		if part == "" {
			continue
		}
		quoted = append(quoted, "`"+strings.ReplaceAll(part, "`", "``")+"`")
	}
	return strings.Join(quoted, ".")
}

func ragEscapeSQLString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func ragEscapeLike(value string) string {
	value = ragEscapeSQLString(value)
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "%", "\\%")
	value = strings.ReplaceAll(value, "_", "\\_")
	return value
}

func ragEscapeJSONPathKey(key string) string {
	return strings.ReplaceAll(strings.TrimSpace(key), "'", "''")
}

func clampInt(raw, fallback, maxValue int) int {
	if raw <= 0 {
		raw = fallback
	}
	if maxValue > 0 && raw > maxValue {
		return maxValue
	}
	return raw
}
