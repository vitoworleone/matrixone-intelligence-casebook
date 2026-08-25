package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

type scopeContextKey struct{}
type runContextKey struct{}

func ContextWithScope(ctx context.Context, scope WorkspaceScope) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, scopeContextKey{}, scope)
}

func ScopeFromContext(ctx context.Context) WorkspaceScope {
	if ctx == nil {
		return WorkspaceScope{}
	}
	scope, _ := ctx.Value(scopeContextKey{}).(WorkspaceScope)
	return scope
}

type RunContext struct {
	mu                       sync.Mutex
	sqlResults               []QuerySQLResponse
	ragChunks                map[string][]FinalAnswerSource
	visualHits               map[string][]FinalAnswerSource
	visualSources            []FinalAnswerSource
	ragSources               []FinalAnswerSource
	sqlArtifacts             map[string][]FinalAnswerSource
	citableEvidenceRetrieved bool
	citableEvidenceRequired  bool
	selectedFinalSources     []FinalAnswerSource
	hasSelectedFinalSources  bool
}

func NewRunContext() *RunContext {
	return &RunContext{citableEvidenceRequired: true}
}

func ContextWithRunContext(ctx context.Context, rc *RunContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, runContextKey{}, rc)
}

func RunContextFrom(ctx context.Context) *RunContext {
	if ctx == nil {
		return nil
	}
	rc, _ := ctx.Value(runContextKey{}).(*RunContext)
	return rc
}

func (rc *RunContext) RecordSQLResult(result QuerySQLResponse) int {
	if rc == nil {
		return -1
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.sqlResults = append(rc.sqlResults, cloneQuerySQLResponse(result))
	return len(rc.sqlResults) - 1
}

func (rc *RunContext) RecordSQLResultArtifact(artifactID string, result QuerySQLResponse) int {
	if rc == nil {
		return -1
	}
	artifactID = strings.TrimSpace(artifactID)
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.citableEvidenceRetrieved = true
	rc.sqlResults = append(rc.sqlResults, cloneQuerySQLResponse(result))
	idx := len(rc.sqlResults) - 1
	if artifactID != "" {
		rc.ensureMapsLocked()
		rc.sqlArtifacts[artifactID] = tableSourcesFromSQLResult(artifactID, result)
	}
	return idx
}

func (rc *RunContext) SQLResultSnapshot(idx int) (QuerySQLResponse, bool) {
	if rc == nil || idx < 0 {
		return QuerySQLResponse{}, false
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if idx >= len(rc.sqlResults) {
		return QuerySQLResponse{}, false
	}
	return cloneQuerySQLResponse(rc.sqlResults[idx]), true
}

func (rc *RunContext) RecordRAGChunksArtifact(artifactID string, result SearchRAGChunksResponse) {
	if rc == nil {
		return
	}
	artifactID = strings.TrimSpace(artifactID)
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.citableEvidenceRetrieved = true
	rc.ensureMapsLocked()
	for _, chunk := range result.Chunks {
		chunkID := strings.TrimSpace(chunk.ChunkID)
		if chunkID == "" {
			continue
		}
		source := FinalAnswerSource{
			Type:            "rag_chunk",
			SemanticModelID: chunk.SemanticModelID,
			SourceRowID:     strings.TrimSpace(chunk.SourceRowID),
			StartMS:         cloneFinalAnswerInt64Ptr(chunk.StartMS),
			EndMS:           cloneFinalAnswerInt64Ptr(chunk.EndMS),
			ArtifactID:      artifactID,
			ChunkIDs:        []string{chunkID},
			FileID:          strings.TrimSpace(chunk.FileID),
			FileName:        strings.TrimSpace(chunk.FileName),
			VolumeID:        strings.TrimSpace(chunk.VolumeID),
			MarkdownFileID:  strings.TrimSpace(chunk.MarkdownFileID),
			Pages:           compactFinalAnswerPages([]int{chunk.PageNumber}),
			ObjectID:        strings.TrimSpace(chunk.ObjectID),
			ObjectKind:      strings.TrimSpace(chunk.ObjectKind),
			ImageFileID:     strings.TrimSpace(chunk.ImageFileID),
			PageImageFileID: strings.TrimSpace(chunk.PageImageFileID),
			BBox:            append([]float64(nil), chunk.BBox...),
			SourceTags:      append([]string(nil), chunk.SourceTags...),
		}
		source.VisualRefs = finalAnswerVisualRefsFromRAGImageRefs(chunk.VisualRefs)
		for index := range source.VisualRefs {
			if source.VisualRefs[index].SemanticModelID <= 0 {
				source.VisualRefs[index].SemanticModelID = chunk.SemanticModelID
			}
		}
		if len(source.VisualRefs) == 0 {
			if ref, ok := finalAnswerVisualRefFromRAGChunk(chunk); ok {
				source.VisualRefs = []FinalAnswerVisualRef{ref}
			}
		}
		source.VisualRefs = compactFinalAnswerVisualRefs(source.VisualRefs)
		for _, ref := range source.VisualRefs {
			visualSource := source
			visualSource.Type = "visual_hit"
			visualSource.VisualRefs = []FinalAnswerVisualRef{ref}
			rc.visualSources = append(rc.visualSources, visualSource)
			for _, key := range finalAnswerVisualRefEvidenceKeys(ref) {
				rc.visualHits[key] = appendFinalAnswerSourceCandidate(rc.visualHits[key], visualSource)
			}
		}
		rc.ragChunks[chunkID] = appendFinalAnswerSourceCandidate(rc.ragChunks[chunkID], source)
		rc.ragSources = append(rc.ragSources, source)
	}
}

func (rc *RunContext) RecordVisualSearchArtifact(artifactID string, result SearchVisualImageResponse) {
	if rc == nil {
		return
	}
	artifactID = strings.TrimSpace(artifactID)
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.citableEvidenceRetrieved = true
	rc.ensureMapsLocked()
	for _, hit := range result.Results {
		ref := FinalAnswerVisualRef{
			SemanticModelID: hit.SemanticModelID,
			ObjectID:        strings.TrimSpace(hit.ObjectID),
			ObjectKind:      strings.TrimSpace(hit.ObjectKind),
			ImageFileID:     strings.TrimSpace(hit.ImageFileID),
			PageImageFileID: strings.TrimSpace(hit.PageImageFileID),
			Page:            hit.PageNumber,
			BBox:            append([]float64(nil), hit.BBox...),
		}
		if ref.ObjectID == "" && ref.ImageFileID == "" && ref.PageImageFileID == "" {
			continue
		}
		source := FinalAnswerSource{
			Type:            "visual_hit",
			SemanticModelID: hit.SemanticModelID,
			SourceRowID:     strings.TrimSpace(hit.SourceRowID),
			ArtifactID:      artifactID,
			FileID:          strings.TrimSpace(hit.SourceFileID),
			FileName:        strings.TrimSpace(hit.SourceFileName),
			Pages:           compactFinalAnswerPages([]int{hit.PageNumber}),
			ObjectID:        ref.ObjectID,
			ObjectKind:      ref.ObjectKind,
			ImageFileID:     ref.ImageFileID,
			PageImageFileID: ref.PageImageFileID,
			BBox:            append([]float64(nil), ref.BBox...),
			VisualRefs:      []FinalAnswerVisualRef{ref},
			SourceTags:      append([]string(nil), hit.SourceTags...),
		}
		for _, key := range visualHitEvidenceKeys(hit) {
			rc.visualHits[key] = appendFinalAnswerSourceCandidate(rc.visualHits[key], source)
		}
		rc.visualSources = append(rc.visualSources, source)
	}
}

func (rc *RunContext) ResolveFinalAnswerSources(raw []FinalAnswerSource) ([]FinalAnswerSource, error) {
	if rc == nil {
		return nil, fmt.Errorf("final answer evidence run context is nil")
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	resolved := make([]FinalAnswerSource, 0, len(raw))
	for _, source := range raw {
		source.Type = strings.TrimSpace(source.Type)
		source.ArtifactID = strings.TrimSpace(source.ArtifactID)
		source.ChunkID = strings.TrimSpace(source.ChunkID)
		source.ChunkIDs = compactFinalAnswerStrings(append(source.ChunkIDs, source.ChunkID))
		switch source.Type {
		case "rag_chunk":
			if len(source.ChunkIDs) == 0 {
				return nil, fmt.Errorf("rag_chunk source requires chunk_id or chunk_ids")
			}
			for _, chunkID := range source.ChunkIDs {
				candidates := rc.ragChunks[chunkID]
				if len(candidates) == 0 {
					return nil, fmt.Errorf("unknown rag chunk source %q", chunkID)
				}
				chunkSource, err := resolveFinalAnswerSourceCandidate(candidates, source.SemanticModelID, "rag chunk", chunkID)
				if err != nil {
					return nil, err
				}
				if source.ArtifactID != "" {
					chunkSource.ArtifactID = source.ArtifactID
				}
				resolved = append(resolved, chunkSource)
			}
		case "sql_result":
			if source.ArtifactID == "" {
				return nil, fmt.Errorf("sql_result source requires artifact_id")
			}
			tableSources := rc.sqlArtifacts[source.ArtifactID]
			if len(tableSources) == 0 {
				return nil, fmt.Errorf("unknown sql result source %q", source.ArtifactID)
			}
			resolved = append(resolved, tableSources...)
		case "visual_hit":
			keys := compactFinalAnswerStrings([]string{
				source.ObjectID,
				source.ImageFileID,
				source.PageImageFileID,
			})
			if len(keys) == 0 {
				return nil, fmt.Errorf("visual_hit source requires object_id, image_file_id, or page_image_file_id")
			}
			for _, key := range keys {
				candidates := rc.visualHits[key]
				if len(candidates) == 0 {
					return nil, fmt.Errorf("unknown visual hit source %q", key)
				}
				visualSource, err := resolveFinalAnswerSourceCandidate(candidates, source.SemanticModelID, "visual hit", key)
				if err != nil {
					return nil, err
				}
				if source.ArtifactID != "" {
					visualSource.ArtifactID = source.ArtifactID
				}
				resolved = append(resolved, visualSource)
			}
		default:
			return nil, fmt.Errorf("unsupported final answer source type %q", source.Type)
		}
	}
	return normalizeFinalAnswerSources(resolved), nil
}

func appendFinalAnswerSourceCandidate(candidates []FinalAnswerSource, source FinalAnswerSource) []FinalAnswerSource {
	for index := range candidates {
		if candidates[index].SemanticModelID == source.SemanticModelID {
			candidates[index] = source
			return candidates
		}
	}
	return append(candidates, source)
}

func resolveFinalAnswerSourceCandidate(candidates []FinalAnswerSource, requestedSemanticModelID int64, kind, evidenceID string) (FinalAnswerSource, error) {
	if len(candidates) == 0 {
		return FinalAnswerSource{}, fmt.Errorf("unknown %s source %q", kind, evidenceID)
	}
	if len(candidates) == 1 {
		// A unique recorded candidate is authoritative. Ignore an untrusted
		// model-generated owner mismatch for backward compatibility.
		return candidates[0], nil
	}
	if requestedSemanticModelID <= 0 {
		return FinalAnswerSource{}, fmt.Errorf(
			"ambiguous %s source %q across semantic models; semantic_model_id is required",
			kind,
			evidenceID,
		)
	}
	var matched *FinalAnswerSource
	for index := range candidates {
		if candidates[index].SemanticModelID != requestedSemanticModelID {
			continue
		}
		if matched != nil {
			return FinalAnswerSource{}, fmt.Errorf(
				"ambiguous %s source %q for semantic_model_id %d",
				kind,
				evidenceID,
				requestedSemanticModelID,
			)
		}
		candidate := candidates[index]
		matched = &candidate
	}
	if matched == nil {
		return FinalAnswerSource{}, fmt.Errorf(
			"unknown %s source %q for semantic_model_id %d",
			kind,
			evidenceID,
			requestedSemanticModelID,
		)
	}
	return *matched, nil
}

func (rc *RunContext) ValidateAnswerSourceCoverage(answer string, sources []FinalAnswerSource) error {
	if rc == nil {
		return fmt.Errorf("final answer evidence run context is nil")
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return ValidateAnswerSourceCoverageCandidates(answer, sources, rc.answerSourceCoverageCandidatesLocked())
}

func (rc *RunContext) FinalAnswerSourceSelectionCandidates() ([]FinalAnswerSource, bool, bool) {
	if rc == nil {
		return nil, false, true
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.answerSourceCoverageCandidatesLocked(), rc.citableEvidenceRetrieved, rc.citableEvidenceRequired
}

func (rc *RunContext) SetCitableEvidenceRequired(required bool) {
	if rc == nil {
		return
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.citableEvidenceRequired = required
}

func (rc *RunContext) SetSelectedFinalAnswerSources(sources []FinalAnswerSource) {
	if rc == nil {
		return
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.selectedFinalSources = normalizeFinalAnswerSources(sources)
	rc.hasSelectedFinalSources = true
}

func (rc *RunContext) SelectedFinalAnswerSources() ([]FinalAnswerSource, bool) {
	if rc == nil {
		return nil, false
	}
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if !rc.hasSelectedFinalSources {
		return nil, false
	}
	out := make([]FinalAnswerSource, len(rc.selectedFinalSources))
	copy(out, rc.selectedFinalSources)
	return out, true
}

func (rc *RunContext) answerSourceCoverageCandidatesLocked() []FinalAnswerSource {
	candidates := make([]FinalAnswerSource, 0, len(rc.visualSources)+len(rc.ragSources)+len(rc.sqlArtifacts))
	candidates = append(candidates, rc.visualSources...)
	candidates = append(candidates, rc.ragSources...)
	artifactIDs := make([]string, 0, len(rc.sqlArtifacts))
	for artifactID, sources := range rc.sqlArtifacts {
		if strings.TrimSpace(artifactID) == "" || len(sources) == 0 {
			continue
		}
		artifactIDs = append(artifactIDs, artifactID)
	}
	sort.Strings(artifactIDs)
	for _, artifactID := range artifactIDs {
		candidates = append(candidates, FinalAnswerSource{Type: "sql_result", ArtifactID: artifactID})
	}
	return normalizeFinalAnswerSources(candidates)
}

func ValidateAnswerSourceCoverageCandidates(answer string, sources []FinalAnswerSource, candidates []FinalAnswerSource) error {
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return nil
	}
	covered := finalAnswerSourceCoverage(sources)
	candidatesByFile := answerFileCandidates(answer, candidates)
	if len(candidatesByFile) == 0 {
		return nil
	}
	fileNames := make([]string, 0, len(candidatesByFile))
	for fileName := range candidatesByFile {
		fileNames = append(fileNames, fileName)
	}
	sort.Strings(fileNames)
	missing := make([]string, 0, len(fileNames))
	for _, fileName := range fileNames {
		coveredSources := covered[fileName]
		candidates := candidatesByFile[fileName]
		if finalAnswerFileCoveredByAnyCandidate(coveredSources, candidates) {
			continue
		}
		missing = append(missing, fileName+" via "+finalAnswerSourceRetryExample(preferFinalAnswerSourceCandidateForRetry(candidates)))
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("answer names evidence files that are not included in sources: %s", strings.Join(missing, "; "))
}

func answerFileCandidates(answer string, sourceCandidates []FinalAnswerSource) map[string][]FinalAnswerSource {
	candidateByFile := map[string][]FinalAnswerSource{}
	for _, source := range sourceCandidates {
		candidateByFile = appendAnswerFileCandidate(answer, candidateByFile, source)
	}
	return candidateByFile
}

func appendAnswerFileCandidate(answer string, candidateByFile map[string][]FinalAnswerSource, source FinalAnswerSource) map[string][]FinalAnswerSource {
	fileName := strings.TrimSpace(source.FileName)
	if fileName == "" || !strings.Contains(answer, fileName) {
		return candidateByFile
	}
	candidateByFile[fileName] = append(candidateByFile[fileName], source)
	return candidateByFile
}

func finalAnswerSourceCoverage(sources []FinalAnswerSource) map[string][]FinalAnswerSource {
	covered := make(map[string][]FinalAnswerSource, len(sources))
	for _, source := range sources {
		fileName := strings.TrimSpace(source.FileName)
		if fileName == "" {
			continue
		}
		covered[fileName] = append(covered[fileName], source)
	}
	return covered
}

func finalAnswerFileCoveredByAnyCandidate(coveredSources []FinalAnswerSource, candidates []FinalAnswerSource) bool {
	if len(coveredSources) == 0 {
		return false
	}
	for _, candidate := range candidates {
		for _, covered := range coveredSources {
			if strings.TrimSpace(covered.Type) != strings.TrimSpace(candidate.Type) {
				continue
			}
			if covered.SemanticModelID > 0 || candidate.SemanticModelID > 0 {
				if covered.SemanticModelID != candidate.SemanticModelID {
					continue
				}
			}
			return true
		}
	}
	return false
}

func preferFinalAnswerSourceCandidateForRetry(candidates []FinalAnswerSource) FinalAnswerSource {
	var preferred FinalAnswerSource
	for _, candidate := range candidates {
		preferred = preferFinalAnswerSourceCandidate(preferred, candidate)
	}
	return preferred
}

func preferFinalAnswerSourceCandidate(existing FinalAnswerSource, next FinalAnswerSource) FinalAnswerSource {
	if existing.Type == "" {
		return next
	}
	if existing.Type != "visual_hit" && next.Type == "visual_hit" {
		return next
	}
	if existing.ImageFileID == "" && next.ImageFileID != "" {
		return next
	}
	if existing.ObjectID == "" && next.ObjectID != "" {
		return next
	}
	return existing
}

func finalAnswerSourceRetryExample(source FinalAnswerSource) string {
	retry := FinalAnswerSource{
		Type:            source.Type,
		SemanticModelID: source.SemanticModelID,
	}
	switch source.Type {
	case "visual_hit":
		retry.ObjectID = source.ObjectID
		retry.ImageFileID = source.ImageFileID
		retry.PageImageFileID = source.PageImageFileID
	case "rag_chunk":
		retry.ChunkIDs = append([]string(nil), source.ChunkIDs...)
	default:
		retry.ArtifactID = source.ArtifactID
	}
	raw, err := json.Marshal(retry)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func (rc *RunContext) ensureMapsLocked() {
	if rc.ragChunks == nil {
		rc.ragChunks = map[string][]FinalAnswerSource{}
	}
	if rc.visualHits == nil {
		rc.visualHits = map[string][]FinalAnswerSource{}
	}
	if rc.sqlArtifacts == nil {
		rc.sqlArtifacts = map[string][]FinalAnswerSource{}
	}
}

func visualHitEvidenceKeys(hit VisualSearchHit) []string {
	return compactFinalAnswerStrings([]string{
		hit.ObjectID,
		hit.ImageFileID,
		hit.PageImageFileID,
	})
}

func finalAnswerVisualRefEvidenceKeys(ref FinalAnswerVisualRef) []string {
	return compactFinalAnswerStrings([]string{
		ref.ObjectID,
		ref.ImageFileID,
		ref.PageImageFileID,
	})
}

func finalAnswerVisualRefFromRAGChunk(chunk RAGChunkHit) (FinalAnswerVisualRef, bool) {
	ref := FinalAnswerVisualRef{
		SemanticModelID: chunk.SemanticModelID,
		ChunkID:         strings.TrimSpace(chunk.ChunkID),
		ObjectID:        strings.TrimSpace(chunk.ObjectID),
		ObjectKind:      strings.TrimSpace(chunk.ObjectKind),
		ImageFileID:     strings.TrimSpace(chunk.ImageFileID),
		PageImageFileID: strings.TrimSpace(chunk.PageImageFileID),
		Page:            chunk.PageNumber,
		BBox:            append([]float64(nil), chunk.BBox...),
	}
	if ref.ObjectID == "" && ref.ObjectKind == "" && ref.ImageFileID == "" && ref.PageImageFileID == "" && len(ref.BBox) == 0 {
		return FinalAnswerVisualRef{}, false
	}
	return ref, true
}

func finalAnswerVisualRefsFromRAGImageRefs(values []RAGImageRef) []FinalAnswerVisualRef {
	if len(values) == 0 {
		return nil
	}
	out := make([]FinalAnswerVisualRef, 0, len(values))
	for _, value := range values {
		out = append(out, FinalAnswerVisualRef{
			SemanticModelID: value.SemanticModelID,
			ChunkID:         strings.TrimSpace(value.ChunkID),
			ObjectID:        strings.TrimSpace(value.ObjectID),
			ObjectKind:      strings.TrimSpace(value.ObjectKind),
			ImageFileID:     strings.TrimSpace(value.ImageFileID),
			PageImageFileID: strings.TrimSpace(value.PageImageFileID),
			Page:            value.Page,
			BBox:            append([]float64(nil), value.BBox...),
		})
	}
	return compactFinalAnswerVisualRefs(out)
}

func tableSourcesFromSQLResult(artifactID string, result QuerySQLResponse) []FinalAnswerSource {
	out := make([]FinalAnswerSource, 0)
	sourceSQL := finalAnswerSourceSQL(result)
	if len(result.Tables) > 0 {
		for _, table := range result.Tables {
			database := strings.TrimSpace(table.DBName)
			if database == "" {
				database = strings.TrimSpace(result.DBName)
			}
			name := strings.TrimSpace(table.Name)
			if database == "" && name == "" {
				continue
			}
			out = append(out, FinalAnswerSource{
				Type:       "sql_table",
				ArtifactID: artifactID,
				Database:   database,
				Table:      name,
				SQL:        sourceSQL,
			})
		}
	}
	if len(out) == 0 && len(result.TableNames) > 0 {
		for _, tableName := range result.TableNames {
			tableName = strings.TrimSpace(tableName)
			if tableName == "" {
				continue
			}
			out = append(out, FinalAnswerSource{
				Type:       "sql_table",
				ArtifactID: artifactID,
				Database:   strings.TrimSpace(result.DBName),
				Table:      tableName,
				SQL:        sourceSQL,
			})
		}
	}
	if len(out) == 0 && strings.TrimSpace(result.DBName) != "" {
		out = append(out, FinalAnswerSource{
			Type:       "sql_table",
			ArtifactID: artifactID,
			Database:   strings.TrimSpace(result.DBName),
			SQL:        sourceSQL,
		})
	}
	return normalizeFinalAnswerSources(out)
}

func finalAnswerSourceSQL(result QuerySQLResponse) string {
	sqlText := strings.TrimSpace(result.SQL)
	if sqlText != "" {
		return sqlText
	}
	return strings.Join(compactFinalAnswerStrings(result.SourceSQLs), "\n\n")
}

// ProjectFinalAnswerSourceRefs returns the ID-level projection that may safely
// travel through model-visible tool output and runtime metadata. Full resolved
// evidence objects stay in RunContext and are rehydrated by ID when needed.
func ProjectFinalAnswerSourceRefs(sources []FinalAnswerSource) []FinalAnswerSource {
	out := make([]FinalAnswerSource, 0, len(sources))
	for _, source := range sources {
		chunkIDs := append([]string(nil), source.ChunkIDs...)
		chunkIDs = append(chunkIDs, source.ChunkID)
		projected := FinalAnswerSource{
			Type:            strings.TrimSpace(source.Type),
			SemanticModelID: source.SemanticModelID,
			SourceRowID:     strings.TrimSpace(source.SourceRowID),
			StartMS:         cloneFinalAnswerInt64Ptr(source.StartMS),
			EndMS:           cloneFinalAnswerInt64Ptr(source.EndMS),
			ArtifactID:      strings.TrimSpace(source.ArtifactID),
			ChunkID:         strings.TrimSpace(source.ChunkID),
			ChunkIDs:        compactFinalAnswerStrings(chunkIDs),
			FileID:          strings.TrimSpace(source.FileID),
			FileName:        strings.TrimSpace(source.FileName),
			ObjectID:        strings.TrimSpace(source.ObjectID),
			ImageFileID:     strings.TrimSpace(source.ImageFileID),
			PageImageFileID: strings.TrimSpace(source.PageImageFileID),
			Database:        strings.TrimSpace(source.Database),
			Table:           strings.TrimSpace(source.Table),
			Label:           strings.TrimSpace(source.Label),
			VisualRefs:      projectFinalAnswerVisualRefs(source.VisualRefs),
		}
		if projected.Type == "" {
			continue
		}
		// Keep a single primary identifier shape for rag_chunk sources.
		if projected.Type == "rag_chunk" {
			if len(projected.ChunkIDs) == 1 {
				projected.ChunkID = projected.ChunkIDs[0]
				projected.ChunkIDs = nil
			} else {
				projected.ChunkID = ""
			}
		}
		out = append(out, projected)
	}
	return out
}

func projectFinalAnswerVisualRefs(values []FinalAnswerVisualRef) []FinalAnswerVisualRef {
	if len(values) == 0 {
		return nil
	}
	projected := make([]FinalAnswerVisualRef, 0, len(values))
	for _, value := range values {
		ref := FinalAnswerVisualRef{
			SemanticModelID: value.SemanticModelID,
			ChunkID:         strings.TrimSpace(value.ChunkID),
			ObjectID:        strings.TrimSpace(value.ObjectID),
			ImageFileID:     strings.TrimSpace(value.ImageFileID),
			PageImageFileID: strings.TrimSpace(value.PageImageFileID),
			Page:            value.Page,
		}
		if ref.ChunkID == "" && ref.ObjectID == "" && ref.ImageFileID == "" && ref.PageImageFileID == "" {
			continue
		}
		projected = append(projected, ref)
	}
	return compactFinalAnswerVisualRefs(projected)
}

func cloneFinalAnswerInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func normalizeFinalAnswerSources(sources []FinalAnswerSource) []FinalAnswerSource {
	groups := make(map[string]FinalAnswerSource, len(sources))
	order := make([]string, 0, len(sources))
	for _, source := range sources {
		source.Type = strings.TrimSpace(source.Type)
		source.ArtifactID = strings.TrimSpace(source.ArtifactID)
		source.ChunkID = strings.TrimSpace(source.ChunkID)
		source.ChunkIDs = compactFinalAnswerStrings(append(source.ChunkIDs, source.ChunkID))
		source.FileID = strings.TrimSpace(source.FileID)
		source.SourceRowID = strings.TrimSpace(source.SourceRowID)
		source.FileName = strings.TrimSpace(source.FileName)
		source.VolumeID = strings.TrimSpace(source.VolumeID)
		source.MarkdownFileID = strings.TrimSpace(source.MarkdownFileID)
		source.Database = strings.TrimSpace(source.Database)
		source.Table = strings.TrimSpace(source.Table)
		source.SQL = strings.TrimSpace(source.SQL)
		source.Label = strings.TrimSpace(source.Label)
		source.ObjectID = strings.TrimSpace(source.ObjectID)
		source.ObjectKind = strings.TrimSpace(source.ObjectKind)
		source.ImageFileID = strings.TrimSpace(source.ImageFileID)
		source.PageImageFileID = strings.TrimSpace(source.PageImageFileID)
		source.VisualRefs = compactFinalAnswerVisualRefs(source.VisualRefs)
		source.Pages = compactFinalAnswerPages(append(source.Pages, source.Page))
		source.Page = 0
		source.ChunkID = ""
		if source.Type == "" {
			continue
		}
		key := finalAnswerSourceGroupKey(source, len(order))
		existing, ok := groups[key]
		if !ok {
			order = append(order, key)
			groups[key] = source
			continue
		}
		existing.ArtifactID = firstNonEmpty(existing.ArtifactID, source.ArtifactID)
		existing.FileID = firstNonEmpty(existing.FileID, source.FileID)
		existing.FileName = firstNonEmpty(existing.FileName, source.FileName)
		existing.VolumeID = firstNonEmpty(existing.VolumeID, source.VolumeID)
		existing.MarkdownFileID = firstNonEmpty(existing.MarkdownFileID, source.MarkdownFileID)
		existing.Database = firstNonEmpty(existing.Database, source.Database)
		existing.Table = firstNonEmpty(existing.Table, source.Table)
		existing.SQL = firstNonEmpty(existing.SQL, source.SQL)
		existing.Label = firstNonEmpty(existing.Label, source.Label)
		existing.ObjectID = firstNonEmpty(existing.ObjectID, source.ObjectID)
		existing.ObjectKind = firstNonEmpty(existing.ObjectKind, source.ObjectKind)
		existing.ImageFileID = firstNonEmpty(existing.ImageFileID, source.ImageFileID)
		existing.PageImageFileID = firstNonEmpty(existing.PageImageFileID, source.PageImageFileID)
		if len(existing.BBox) == 0 && len(source.BBox) > 0 {
			existing.BBox = append([]float64(nil), source.BBox...)
		}
		existing.VisualRefs = compactFinalAnswerVisualRefs(append(existing.VisualRefs, source.VisualRefs...))
		existing.ChunkIDs = compactFinalAnswerStrings(append(existing.ChunkIDs, source.ChunkIDs...))
		existing.Pages = compactFinalAnswerPages(append(existing.Pages, source.Pages...))
		existing.SourceTags = append(existing.SourceTags, source.SourceTags...)
		groups[key] = existing
	}
	out := make([]FinalAnswerSource, 0, len(order))
	for _, key := range order {
		out = append(out, groups[key])
	}
	return out
}

func compactFinalAnswerVisualRefs(values []FinalAnswerVisualRef) []FinalAnswerVisualRef {
	out := make([]FinalAnswerVisualRef, 0, len(values))
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
			fmt.Sprintf("%d", value.SemanticModelID),
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

func finalAnswerSourceGroupKey(source FinalAnswerSource, index int) string {
	ownerKey := ""
	if source.SemanticModelID > 0 {
		ownerKey = fmt.Sprintf("semantic_model:%d\x00", source.SemanticModelID)
	}
	switch source.Type {
	case "rag_chunk":
		if source.StartMS != nil && source.EndMS != nil && *source.StartMS >= 0 && *source.StartMS < *source.EndMS {
			return ownerKey + source.Type + "\x00" + firstNonEmpty(append(source.ChunkIDs, fmt.Sprintf("source_%d", index))...)
		}
		return ownerKey + source.Type + "\x00" + firstNonEmpty(source.FileID, source.FileName, source.Label, source.ArtifactID, fmt.Sprintf("source_%d", index))
	case "visual_hit":
		return ownerKey + source.Type + "\x00" + firstNonEmpty(source.FileID, source.FileName, source.ObjectID, source.ImageFileID, source.PageImageFileID, source.Label, source.ArtifactID, fmt.Sprintf("source_%d", index))
	case "sql_table":
		return source.Type + "\x00" + source.Database + "\x00" + firstNonEmpty(source.Table, source.Label, fmt.Sprintf("source_%d", index)) + "\x00" + source.SQL
	default:
		return ownerKey + source.Type + "\x00" + firstNonEmpty(source.ArtifactID, source.Label, source.FileID, source.Database+"."+source.Table, fmt.Sprintf("source_%d", index))
	}
}

func compactFinalAnswerStrings(values []string) []string {
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

func compactFinalAnswerPages(values []int) []int {
	out := make([]int, 0, len(values))
	seen := make(map[int]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Ints(out)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
