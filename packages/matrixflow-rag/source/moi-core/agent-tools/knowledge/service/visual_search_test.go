package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/matrixflow/moi-core/agent-tools/knowledge"
)

type stubVisualBackend struct{}

func (stubVisualBackend) ReadImageFile(context.Context, knowledge.WorkspaceScope, string) ([]byte, string, error) {
	return nil, "", nil
}

func (stubVisualBackend) CreateImageEmbedding(context.Context, string, VisualImageEmbeddingRequest) ([]float64, map[string]any, error) {
	return nil, nil, nil
}

func (stubVisualBackend) ResolveVisualScopeFileIDs(context.Context, knowledge.WorkspaceScope) ([]string, bool, error) {
	return nil, false, nil
}

type recordingVisualBackend struct {
	embeddings [][]float64
	requests   []VisualImageEmbeddingRequest
	workspaces []string
}

func (b *recordingVisualBackend) ReadImageFile(context.Context, knowledge.WorkspaceScope, string) ([]byte, string, error) {
	return []byte("query-image"), "image/png", nil
}

func (b *recordingVisualBackend) CreateImageEmbedding(_ context.Context, workspaceID string, req VisualImageEmbeddingRequest) ([]float64, map[string]any, error) {
	b.workspaces = append(b.workspaces, workspaceID)
	b.requests = append(b.requests, req)
	if len(b.embeddings) == 0 {
		return []float64{0.1, 0.2}, nil, nil
	}
	embedding := b.embeddings[0]
	b.embeddings = b.embeddings[1:]
	return embedding, nil, nil
}

func (b *recordingVisualBackend) ResolveVisualScopeFileIDs(context.Context, knowledge.WorkspaceScope) ([]string, bool, error) {
	return nil, false, nil
}

func TestVisualHitFromRowAllowsPageOnlyEvidence(t *testing.T) {
	meta := `{"page_image_file_id":"page_img_1","source_file_id":"file_1","page_number":2}`
	hit, ok := visualHitFromRow(
		[]string{"content", "meta", "file_id", "page_number", "score"},
		[]any{"page match", meta, "file_1", 2, 0.9},
		"text",
	)
	if !ok {
		t.Fatalf("visualHitFromRow rejected page-only evidence")
	}
	if hit.PageImageFileID != "page_img_1" {
		t.Fatalf("PageImageFileID = %q, want page_img_1", hit.PageImageFileID)
	}
	if hit.SourceFileID != "file_1" || hit.PageNumber != 2 {
		t.Fatalf("source/page = %q/%d, want file_1/2", hit.SourceFileID, hit.PageNumber)
	}
}

func TestVisualSearchQuotesQualifiedImageVectorTable(t *testing.T) {
	exec := &stubQuerySQLExecutor{results: []*knowledge.SQLExecutionResult{{
		Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"},
	}}}
	svc := NewSearchVisualImage(Deps{
		SQLExecutor:         exec,
		VisualSearchBackend: stubVisualBackend{},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchVisualImageRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:             "ws_1",
			DBName:                  "db",
			ImageVectorTable:        "idx_db.image_vec",
			ImageEmbeddingModel:     "efficientnet-b3",
			ImageEmbeddingDimension: 3,
			ImagePreprocessVersion:  "v1",
			ImageDistanceMetric:     "cosine",
		},
		QueryText: "pump",
		TopK:      1,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(exec.sqls) != 1 {
		t.Fatalf("SQL calls = %d, want 1", len(exec.sqls))
	}
	if !strings.Contains(exec.sqls[0], "FROM `idx_db`.`image_vec`") {
		t.Fatalf("qualified image table was not quoted component-wise: %s", exec.sqls[0])
	}
	if strings.Contains(exec.sqls[0], "FROM `idx_db.image_vec`") {
		t.Fatalf("qualified image table was quoted as one identifier: %s", exec.sqls[0])
	}
}

func TestVisualSearchRAGSourceDoesNotInheritStructuredDBName(t *testing.T) {
	exec := &stubQuerySQLExecutor{results: []*knowledge.SQLExecutionResult{{
		Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"},
	}}}
	svc := NewSearchVisualImage(Deps{
		SQLExecutor:         exec,
		VisualSearchBackend: stubVisualBackend{},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchVisualImageRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "ws_1",
			DBName:      "structured_db",
			RAGSources: []knowledge.RAGSource{{
				ImageVectorTable:        "image_vec",
				ImageEmbeddingModel:     "efficientnet-b3",
				ImageEmbeddingDimension: 3,
				ImagePreprocessVersion:  "v1",
				ImageDistanceMetric:     "cosine",
				FileIDs:                 []string{"file_1"},
			}},
		},
		QueryText: "drawing",
		TopK:      1,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(exec.dbNames) != 1 || exec.dbNames[0] != "" {
		t.Fatalf("db calls = %#v, want tenant connection default database", exec.dbNames)
	}
	if len(exec.sqls) != 1 || !strings.Contains(exec.sqls[0], "FROM `image_vec`") {
		t.Fatalf("visual SQL = %#v, want unqualified image vector table", exec.sqls)
	}
}

func TestVisualSearchUsesRAGSourceDBNameInsteadOfStructuredScopeDB(t *testing.T) {
	exec := &stubQuerySQLExecutor{results: []*knowledge.SQLExecutionResult{{
		Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"},
	}}}
	svc := NewSearchVisualImage(Deps{
		SQLExecutor:         exec,
		VisualSearchBackend: stubVisualBackend{},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchVisualImageRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "ws_1",
			DBName:      "structured_db",
			RAGSources: []knowledge.RAGSource{{
				DBName:                  "rag_db",
				ImageVectorTable:        "image_vec",
				ImageEmbeddingModel:     "efficientnet-b3",
				ImageEmbeddingDimension: 3,
				ImagePreprocessVersion:  "v1",
				ImageDistanceMetric:     "cosine",
				FileIDs:                 []string{"file_1"},
			}},
		},
		QueryText: "drawing",
		TopK:      1,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(exec.dbNames) != 1 || exec.dbNames[0] != "rag_db" {
		t.Fatalf("db calls = %#v, want rag_db", exec.dbNames)
	}
}

func TestVisualSearchQueriesEachRAGSourceImageIndex(t *testing.T) {
	exec := &stubQuerySQLExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"}},
		{Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"}},
	}}
	svc := NewSearchVisualImage(Deps{
		SQLExecutor:         exec,
		VisualSearchBackend: stubVisualBackend{},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchVisualImageRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "ws_1",
			DBName:      "structured_db",
			RAGSources: []knowledge.RAGSource{
				{
					DBName:                  "kb_a",
					ImageVectorTable:        "image_vec_a",
					ImageEmbeddingModel:     "efficientnet-b3",
					ImageEmbeddingDimension: 3,
					ImagePreprocessVersion:  "v1",
					ImageDistanceMetric:     "cosine",
					FileIDs:                 []string{"file_a"},
				},
				{
					DBName:                  "kb_b",
					ImageVectorTable:        "image_vec_b",
					ImageEmbeddingModel:     "clip-vit-large",
					ImageEmbeddingDimension: 3,
					ImagePreprocessVersion:  "v2",
					ImageDistanceMetric:     "cosine",
					FileIDs:                 []string{"file_b"},
				},
			},
		},
		QueryText: "drawing",
		TopK:      1,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(exec.sqls) != 2 {
		t.Fatalf("SQL calls = %d, want one query per image-capable RAG source", len(exec.sqls))
	}
	if len(exec.dbNames) != 2 || exec.dbNames[0] != "kb_a" || exec.dbNames[1] != "kb_b" {
		t.Fatalf("db calls = %#v, want kb_a then kb_b", exec.dbNames)
	}
	if !strings.Contains(exec.sqls[0], "FROM `image_vec_a`") || !strings.Contains(exec.sqls[1], "FROM `image_vec_b`") {
		t.Fatalf("visual SQLs = %#v, want each source image vector table", exec.sqls)
	}
}

func TestVisualSearchImageQueryUsesEachSourceModelAndGlobalScore(t *testing.T) {
	exec := &stubQuerySQLExecutor{results: []*knowledge.SQLExecutionResult{
		{
			Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"},
			Rows: [][]any{{
				"low",
				"low score visual match",
				`{"image_file_id":"img_low","source_file_id":"file_a","page_number":1}`,
				"file_a",
				1,
				0.1,
			}},
		},
		{
			Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"},
			Rows: [][]any{{
				"high",
				"high score visual match",
				`{"image_file_id":"img_high","source_file_id":"file_b","page_number":2}`,
				"file_b",
				2,
				0.9,
			}},
		},
	}}
	backend := &recordingVisualBackend{
		embeddings: [][]float64{{0.1, 0.2}, {0.3, 0.4}},
	}
	svc := NewSearchVisualImage(Deps{
		SQLExecutor:         exec,
		VisualSearchBackend: backend,
	})

	resp, err := svc.Execute(context.Background(), knowledge.SearchVisualImageRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "ws_1",
			RAGSources: []knowledge.RAGSource{
				{
					DBName:                  "kb_a",
					ImageVectorTable:        "image_vec_a",
					ImageEmbeddingModel:     "efficientnet-b3",
					ImageEmbeddingBackendID: "11",
					ImageEmbeddingDimension: 2,
					ImagePreprocessVersion:  "efficientnet-v1",
					ImageDistanceMetric:     "cosine",
					FileIDs:                 []string{"file_a"},
				},
				{
					DBName:                  "kb_b",
					ImageVectorTable:        "image_vec_b",
					ImageEmbeddingModel:     "clip-vit-large",
					ImageEmbeddingBackendID: "22",
					ImageEmbeddingDimension: 2,
					ImagePreprocessVersion:  "clip-v2",
					ImageDistanceMetric:     "cosine",
					FileIDs:                 []string{"file_b"},
				},
			},
		},
		QueryVisualFileID: "query_image",
		TopK:              1,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(backend.requests) != 2 {
		t.Fatalf("embedding requests = %d, want 2", len(backend.requests))
	}
	if backend.requests[0].Model != "efficientnet-b3" || backend.requests[0].BackendID != "11" || backend.requests[0].PreprocessVersion != "efficientnet-v1" {
		t.Fatalf("first embedding request = %+v, want efficientnet-b3/backend 11/efficientnet-v1", backend.requests[0])
	}
	if backend.requests[1].Model != "clip-vit-large" || backend.requests[1].BackendID != "22" || backend.requests[1].PreprocessVersion != "clip-v2" {
		t.Fatalf("second embedding request = %+v, want clip-vit-large/backend 22/clip-v2", backend.requests[1])
	}
	if len(exec.sqls) != 2 || !strings.Contains(exec.sqls[0], "FROM `image_vec_a`") || !strings.Contains(exec.sqls[1], "FROM `image_vec_b`") {
		t.Fatalf("visual SQLs = %#v, want both source image vector tables", exec.sqls)
	}
	if len(exec.dbNames) != 2 || exec.dbNames[0] != "kb_a" || exec.dbNames[1] != "kb_b" {
		t.Fatalf("db calls = %#v, want kb_a then kb_b", exec.dbNames)
	}
	if resp.Count != 1 || len(resp.Results) != 1 {
		t.Fatalf("result count = %d/%d, want one top hit", resp.Count, len(resp.Results))
	}
	if resp.Results[0].SourceFileID != "file_b" || resp.Results[0].Score != 0.9 {
		t.Fatalf("top result = %+v, want globally highest score from file_b", resp.Results[0])
	}
}

func TestVisualSearchMultiModelHitsKeepTrustedSemanticModelID(t *testing.T) {
	exec := &stubQuerySQLExecutor{results: []*knowledge.SQLExecutionResult{
		{
			Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"},
			Rows: [][]any{{
				"hit_a",
				"visual match from model a",
				`{"image_file_id":"img_a","source_file_id":"file_a","page_number":1}`,
				"file_a",
				1,
				0.8,
			}},
		},
		{
			Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"},
			Rows: [][]any{{
				"hit_b",
				"visual match from model b",
				`{"image_file_id":"img_b","source_file_id":"file_b","page_number":2}`,
				"file_b",
				2,
				0.9,
			}},
		},
	}}
	backend := &recordingVisualBackend{
		embeddings: [][]float64{{0.1, 0.2}, {0.3, 0.4}},
	}
	svc := NewSearchVisualImage(Deps{
		SQLExecutor:         exec,
		VisualSearchBackend: backend,
	})

	resp, err := svc.Execute(context.Background(), knowledge.SearchVisualImageRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "ws_1",
			RAGSources: []knowledge.RAGSource{
				{
					SemanticModelID:         101,
					DBName:                  "kb_a",
					ImageVectorTable:        "image_vec_a",
					ImageEmbeddingModel:     "efficientnet-b3",
					ImageEmbeddingDimension: 2,
					ImagePreprocessVersion:  "efficientnet-v1",
					ImageDistanceMetric:     "cosine",
					FileIDs:                 []string{"file_a"},
					SourceRowIDByFileID:     map[string]string{"file_a": "source_a"},
				},
				{
					SemanticModelID:         202,
					DBName:                  "kb_b",
					ImageVectorTable:        "image_vec_b",
					ImageEmbeddingModel:     "clip-vit-large",
					ImageEmbeddingDimension: 2,
					ImagePreprocessVersion:  "clip-v2",
					ImageDistanceMetric:     "cosine",
					FileIDs:                 []string{"file_b"},
					SourceRowIDByFileID:     map[string]string{"file_b": "source_b"},
				},
			},
		},
		QueryVisualFileID: "query_image",
		TopK:              2,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("results = %+v, want one hit from each semantic model", resp.Results)
	}

	wantModelByFile := map[string]int64{
		"file_a": 101,
		"file_b": 202,
	}
	wantSourceByFile := map[string]string{
		"file_a": "source_a",
		"file_b": "source_b",
	}
	for _, hit := range resp.Results {
		wantModelID, ok := wantModelByFile[hit.SourceFileID]
		if !ok {
			t.Errorf("unexpected result source file %q: %+v", hit.SourceFileID, hit)
			continue
		}
		t.Run(hit.SourceFileID, func(t *testing.T) {
			requireServiceSemanticModelID(t, "visual search hit", hit, wantModelID)
			if hit.SourceRowID != wantSourceByFile[hit.SourceFileID] {
				t.Fatalf("visual search hit source_row_id = %q, want %q", hit.SourceRowID, wantSourceByFile[hit.SourceFileID])
			}
		})
	}
}

func requireServiceSemanticModelID(t *testing.T, label string, value any, want int64) {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal %s: %v", label, err)
	}
	var contract map[string]any
	if err := json.Unmarshal(encoded, &contract); err != nil {
		t.Fatalf("decode %s public contract: %v", label, err)
	}
	rawModelID, exists := contract["semantic_model_id"]
	gotModelID, numberOK := rawModelID.(float64)
	if !exists || !numberOK || int64(gotModelID) != want {
		t.Fatalf(
			"%s semantic_model_id = %v (present=%t), want trusted value %d; contract=%s",
			label,
			rawModelID,
			exists,
			want,
			encoded,
		)
	}
}

func TestVisualObjectFirstRerankPrefersSameObjectConstraintMatches(t *testing.T) {
	hits := []knowledge.VisualSearchHit{
		{
			ObjectID:       "shape_only",
			ObjectKind:     "view",
			SourceFileID:   "file_shape",
			SourceFileName: "20C114582.pdf",
			PageNumber:     1,
			ImageFileID:    "image_shape",
			Score:          0.95,
			Content:        "view outline only",
			ScoreParts:     map[string]any{"image": 0.95},
		},
		{
			ObjectID:       "correct",
			ObjectKind:     "view",
			SourceFileID:   "file_correct",
			SourceFileName: "20C114257.pdf",
			PageNumber:     1,
			ImageFileID:    "image_correct",
			Score:          0.7,
			Content:        "R20±0.25 Ø17±0.2 34±0.35 A-A",
			ScoreParts:     map[string]any{"image": 0.7},
		},
		{
			ObjectID:       "title",
			ObjectKind:     "title_block",
			SourceFileID:   "file_title",
			SourceFileName: "20C114709.pdf",
			PageNumber:     1,
			ImageFileID:    "image_title",
			Score:          0.9,
			Content:        "R20±0.25",
			ScoreParts:     map[string]any{"text": 1.0},
		},
	}

	got := rerankVisualObjectFirstResults(hits, []string{"R20±0.25", "Ø17±0.2", "34±0.35", "A-A"}, 3)
	if len(got) != 3 {
		t.Fatalf("rerank count = %d, want 3", len(got))
	}
	if got[0].SourceFileName != "20C114257.pdf" {
		t.Fatalf("top hit = %+v, want 20C114257.pdf", got[0])
	}
	if got[2].ObjectKind != "title_block" {
		t.Fatalf("title_block rank = %+v, want last", got)
	}
}

func TestVisualObjectFusionKeyIncludesSourceFileID(t *testing.T) {
	textHits := []knowledge.VisualSearchHit{{
		ObjectID:       "object_1",
		SourceFileID:   "file_a",
		SourceFileName: "file-a.pdf",
		ImageFileID:    "image_a",
		Score:          1,
		Content:        "R21.7±0.3 Ø20.7",
	}}
	imageHits := []knowledge.VisualSearchHit{{
		ObjectID:       "object_1",
		SourceFileID:   "file_b",
		SourceFileName: "file-b.pdf",
		ImageFileID:    "image_b",
		Score:          0.99,
		Content:        "visual match",
	}}

	got := fuseVisualResults(textHits, imageHits, 10, knowledge.VisualSearchRankingProfileVisualObjectFirst)
	if len(got) != 2 {
		t.Fatalf("fused hit count = %d, want 2: %+v", len(got), got)
	}
	keys := map[string]bool{}
	for _, hit := range got {
		key, _ := hit.ScoreParts["fusion_key"].(string)
		key = strings.TrimSpace(key)
		keys[key] = true
		if key == "object_1" {
			t.Fatalf("fusion key used local object id without source file: %+v", hit.ScoreParts)
		}
	}
	if !keys["file_a:object_1"] || !keys["file_b:object_1"] {
		t.Fatalf("fusion keys = %+v, want file-scoped object keys", keys)
	}
}

func TestVisualFusionKeepsSameEvidenceFromDifferentSemanticModels(t *testing.T) {
	var textHits []knowledge.VisualSearchHit
	if err := json.Unmarshal([]byte(`[
		{
			"semantic_model_id":101,
			"object_id":"shared_object",
			"source_file_id":"shared_file",
			"source_file_name":"model-101.pdf",
			"image_file_id":"shared_image",
			"score":1,
			"content":"model 101 text match"
		},
		{
			"semantic_model_id":202,
			"object_id":"shared_object",
			"source_file_id":"shared_file",
			"source_file_name":"model-202.pdf",
			"image_file_id":"shared_image",
			"score":0.9,
			"content":"model 202 text match"
		}
	]`), &textHits); err != nil {
		t.Fatalf("decode text hits: %v", err)
	}
	var imageHits []knowledge.VisualSearchHit
	if err := json.Unmarshal([]byte(`[
		{
			"semantic_model_id":101,
			"object_id":"shared_object",
			"source_file_id":"shared_file",
			"source_file_name":"model-101.pdf",
			"image_file_id":"shared_image",
			"score":0.8,
			"content":"model 101 image match"
		},
		{
			"semantic_model_id":202,
			"object_id":"shared_object",
			"source_file_id":"shared_file",
			"source_file_name":"model-202.pdf",
			"image_file_id":"shared_image",
			"score":0.7,
			"content":"model 202 image match"
		}
	]`), &imageHits); err != nil {
		t.Fatalf("decode image hits: %v", err)
	}

	got := fuseVisualResults(textHits, imageHits, 10, knowledge.VisualSearchRankingProfileVisualObjectFirst)
	if len(got) != 2 {
		t.Fatalf(
			"fused hit count = %d, want two model-owned hits even when file/object/image IDs collide: %+v",
			len(got),
			got,
		)
	}
	wantModels := map[int64]bool{101: false, 202: false}
	for index, hit := range got {
		encoded, err := json.Marshal(hit)
		if err != nil {
			t.Fatalf("marshal fused hit %d: %v", index, err)
		}
		var contract map[string]any
		if err := json.Unmarshal(encoded, &contract); err != nil {
			t.Fatalf("decode fused hit %d: %v", index, err)
		}
		rawModelID, exists := contract["semantic_model_id"]
		modelID, numberOK := rawModelID.(float64)
		if !exists || !numberOK {
			t.Fatalf("fused hit %d lacks semantic_model_id: %s", index, encoded)
		}
		id := int64(modelID)
		if _, ok := wantModels[id]; !ok {
			t.Fatalf("fused hit %d semantic_model_id = %d, want 101 or 202: %s", index, id, encoded)
		}
		wantModels[id] = true
	}
	for modelID, seen := range wantModels {
		if !seen {
			t.Fatalf("fused hits lost semantic model %d: %+v", modelID, got)
		}
	}
}

func TestVisualTextRegionFirstUsesObjectScopeSQL(t *testing.T) {
	exec := &stubQuerySQLExecutor{results: []*knowledge.SQLExecutionResult{{
		Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"},
	}}}
	svc := NewSearchVisualImage(Deps{
		SQLExecutor:         exec,
		VisualSearchBackend: stubVisualBackend{},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchVisualImageRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:             "ws_1",
			DBName:                  "db",
			ImageVectorTable:        "image_vec",
			ImageEmbeddingModel:     "efficientnet-b3",
			ImageEmbeddingDimension: 3,
			ImagePreprocessVersion:  "v1",
			ImageDistanceMetric:     "cosine",
		},
		QueryText:      "GB/T 3672.1-2002",
		RankingProfile: knowledge.VisualSearchRankingProfileTextRegionFirst,
		TopK:           1,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(exec.sqls) != 1 {
		t.Fatalf("SQL calls = %d, want 1", len(exec.sqls))
	}
	if !strings.Contains(exec.sqls[0], "JSON_UNQUOTE(JSON_EXTRACT(meta, '$.scope')) = 'visual_object'") ||
		!strings.Contains(exec.sqls[0], "JSON_UNQUOTE(JSON_EXTRACT(meta, '$.object_kind')) <> 'page'") {
		t.Fatalf("visual_text_region_first should filter visual object rows, got %s", exec.sqls[0])
	}
}

func TestVisualTextSearchOrdersCandidatesByConstraintCoverageBeforeLimit(t *testing.T) {
	exec := &stubQuerySQLExecutor{results: []*knowledge.SQLExecutionResult{{
		Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"},
	}}}
	svc := NewSearchVisualImage(Deps{
		SQLExecutor:         exec,
		VisualSearchBackend: stubVisualBackend{},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchVisualImageRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:             "ws_1",
			DBName:                  "db",
			ImageVectorTable:        "image_vec",
			ImageEmbeddingModel:     "efficientnet-b3",
			ImageEmbeddingDimension: 3,
			ImagePreprocessVersion:  "v1",
			ImageDistanceMetric:     "cosine",
		},
		QueryText:      "前稳定杆 前稳定杆总成 stabilizer bar front anti-roll bar 工程图纸 零件图",
		RankingProfile: knowledge.VisualSearchRankingProfileVisualObjectFirst,
		TopK:           30,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(exec.sqls) != 1 {
		t.Fatalf("SQL calls = %d, want 1", len(exec.sqls))
	}
	sql := exec.sqls[0]
	for _, fragment := range []string{"前稳定杆", "前稳定杆总成", "stabilizer", "bar", "front", "anti-roll", "工程图纸", "零件图"} {
		if !strings.Contains(sql, "IF(content LIKE '%"+fragment+"%', 1, 0)") {
			t.Fatalf("text search SQL missing match score fragment %q: %s", fragment, sql)
		}
	}
	if strings.Contains(sql, "LIKE '%前稳定杆 前稳定杆总成 stabilizer bar front anti-roll bar 工程图纸 零件图%'") {
		t.Fatalf("text search SQL should split whitespace-separated query fragments: %s", sql)
	}
	if !strings.Contains(sql, "ORDER BY score DESC, id ASC") {
		t.Fatalf("text search SQL should order by constraint coverage before id: %s", sql)
	}
}

func TestVisualTextSearchFiltersCurrentIndexVersion(t *testing.T) {
	exec := &stubQuerySQLExecutor{results: []*knowledge.SQLExecutionResult{{
		Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"},
		Rows: [][]any{{
			"row-1",
			"pump region",
			`{"page_image_file_id":"page-img","source_file_id":"file_a","page_number":2,"index_version":7,"level":"chunk","chunk_index":0}`,
			"file_a",
			2,
			1.0,
		}},
	}}}
	svc := NewSearchVisualImage(Deps{
		SQLExecutor:         exec,
		VisualSearchBackend: stubVisualBackend{},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchVisualImageRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:             "ws_1",
			DBName:                  "db",
			ImageVectorTable:        "image_vec",
			ImageEmbeddingModel:     "efficientnet-b3",
			ImageEmbeddingDimension: 3,
			ImagePreprocessVersion:  "v1",
			ImageDistanceMetric:     "cosine",
			RAGSources: []knowledge.RAGSource{{
				SemanticModelID:     42,
				FileIDs:             []string{"file_a"},
				SourceRowIDByFileID: map[string]string{"file_a": "source_a"},
				CurrentIndexVersionByFileID: map[string]int64{
					"file_a": 7,
				},
			}},
		},
		QueryText: "pump",
		TopK:      1,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(exec.sqls) != 1 {
		t.Fatalf("SQL calls = %d, want query only", len(exec.sqls))
	}
	if !strings.Contains(exec.sqls[0], "(file_id = 'file_a' AND index_version = '7')") {
		t.Fatalf("visual text SQL missing current version filter: %s", exec.sqls[0])
	}
	for _, sql := range exec.sqls {
		if strings.Contains(sql, "knowledge_base_chunk_recall_stats") {
			t.Fatalf("read visual search should not write recall stats: %s", sql)
		}
	}
}

func TestVisualTextSearchSplitsWhitespaceSeparatedQueryText(t *testing.T) {
	exec := &stubQuerySQLExecutor{results: []*knowledge.SQLExecutionResult{{
		Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"},
	}}}
	svc := NewSearchVisualImage(Deps{
		SQLExecutor:         exec,
		VisualSearchBackend: stubVisualBackend{},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchVisualImageRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:             "ws_1",
			DBName:                  "db",
			ImageVectorTable:        "image_vec",
			ImageEmbeddingModel:     "efficientnet-b3",
			ImageEmbeddingDimension: 3,
			ImagePreprocessVersion:  "v1",
			ImageDistanceMetric:     "cosine",
		},
		QueryText:      "前稳定杆 stabilizer bar anti-roll",
		RankingProfile: knowledge.VisualSearchRankingProfileVisualObjectFirst,
		TopK:           10,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(exec.sqls) != 1 {
		t.Fatalf("SQL calls = %d, want 1", len(exec.sqls))
	}
	sql := exec.sqls[0]
	if strings.Contains(sql, "content LIKE '%前稳定杆 stabilizer bar anti-roll%'") {
		t.Fatalf("text search SQL should not use the whole whitespace-joined query as one LIKE fragment: %s", sql)
	}
	for _, fragment := range []string{"前稳定杆", "stabilizer", "bar", "anti-roll"} {
		if !strings.Contains(sql, "content LIKE '%"+fragment+"%'") {
			t.Fatalf("text search SQL missing OR LIKE fragment %q: %s", fragment, sql)
		}
		if !strings.Contains(sql, "IF(content LIKE '%"+fragment+"%', 1, 0)") {
			t.Fatalf("text search SQL missing match score fragment %q: %s", fragment, sql)
		}
	}
}

func TestVisualTextSearchDefaultProfileKeepsDocumentOrder(t *testing.T) {
	exec := &stubQuerySQLExecutor{results: []*knowledge.SQLExecutionResult{{
		Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"},
	}}}
	svc := NewSearchVisualImage(Deps{
		SQLExecutor:         exec,
		VisualSearchBackend: stubVisualBackend{},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchVisualImageRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:             "ws_1",
			DBName:                  "db",
			ImageVectorTable:        "image_vec",
			ImageEmbeddingModel:     "efficientnet-b3",
			ImageEmbeddingDimension: 3,
			ImagePreprocessVersion:  "v1",
			ImageDistanceMetric:     "cosine",
		},
		QueryText: "A-A\nR20±0.25",
		TopK:      30,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(exec.sqls) != 1 {
		t.Fatalf("SQL calls = %d, want 1", len(exec.sqls))
	}
	sql := exec.sqls[0]
	if !strings.Contains(sql, "1.0 AS score") {
		t.Fatalf("default visual text search should keep constant score: %s", sql)
	}
	if strings.Contains(sql, "IF(content LIKE") {
		t.Fatalf("default visual text search should not use constraint coverage score: %s", sql)
	}
	if !strings.Contains(sql, "ORDER BY id ASC") {
		t.Fatalf("default visual text search should keep document ordering: %s", sql)
	}
}

func TestVisualTextRegionFirstRerankPrefersConstraintCoverageWithoutMetadataPenalty(t *testing.T) {
	hits := []knowledge.VisualSearchHit{
		{
			ObjectID:       "shape_like",
			ObjectKind:     "view",
			SourceFileID:   "file_shape",
			SourceFileName: "20C114257.pdf",
			PageNumber:     1,
			ImageFileID:    "image_shape",
			Score:          0.98,
			Content:        "outline similar",
			ScoreParts:     map[string]any{"image": 0.98},
		},
		{
			ObjectID:       "notes_match",
			ObjectKind:     "notes",
			SourceFileID:   "file_notes",
			SourceFileName: "20C114684.pdf",
			PageNumber:     1,
			ImageFileID:    "image_notes",
			Score:          0.4,
			Content:        "GB/T 3672.1-2002 DIN ISO 3302-1 60±3 ShA 150000 cycles",
			ScoreParts:     map[string]any{"text": 1.0},
		},
		{
			ObjectID:       "title_match",
			ObjectKind:     "title_block",
			SourceFileID:   "file_title",
			SourceFileName: "20C114820.pdf",
			PageNumber:     1,
			ImageFileID:    "image_title",
			Score:          0.3,
			Content:        "GB/T 3672.1-2002",
			ScoreParts:     map[string]any{"text": 1.0},
		},
	}

	got := rerankVisualTextRegionFirstResults(hits, []string{"GB/T 3672.1-2002", "DIN ISO 3302-1", "60±3 ShA", "150000 cycles"}, 3)
	if len(got) != 3 {
		t.Fatalf("rerank count = %d, want 3", len(got))
	}
	if got[0].SourceFileName != "20C114684.pdf" {
		t.Fatalf("top hit = %+v, want 20C114684.pdf", got[0])
	}
	if got[0].ObjectKind != "notes" {
		t.Fatalf("notes hit should not be penalized in text-region profile: %+v", got[0])
	}
}

func TestVisualTextRegionFirstRerankPrefersTableKeyValueCoverage(t *testing.T) {
	hits := []knowledge.VisualSearchHit{
		{
			ObjectID:       "layout",
			ObjectKind:     "table",
			SourceFileID:   "file_layout",
			SourceFileName: "20C114398.pdf",
			PageNumber:     1,
			ImageFileID:    "image_layout",
			Score:          0.9,
			Content:        "材料 硬度",
			ScoreParts:     map[string]any{"image": 0.9},
		},
		{
			ObjectID:       "table",
			ObjectKind:     "table",
			SourceFileID:   "file_table",
			SourceFileName: "20C114820.pdf",
			PageNumber:     1,
			ImageFileID:    "image_table",
			Score:          0.4,
			Content:        "材料 PA66+GF35 硬度 60±3 ShA 版本 A",
			ScoreParts:     map[string]any{"text": 1.0},
		},
	}

	got := rerankVisualTextRegionFirstResults(hits, []string{"材料", "硬度", "材料 PA66+GF35", "PA66+GF35", "60±3 ShA"}, 2)
	if got[0].SourceFileName != "20C114820.pdf" {
		t.Fatalf("top hit = %+v, want 20C114820.pdf", got[0])
	}
}

func TestVisualTextRegionFirstRerankUsesImageRankForNearTextTie(t *testing.T) {
	fragments := make([]string, 0, 34)
	content683 := strings.Builder{}
	content684 := strings.Builder{}
	for i := 0; i < 33; i++ {
		fragment := fmt.Sprintf("constraint-%02d", i)
		fragments = append(fragments, fragment)
		content683.WriteString(fragment)
		content683.WriteByte(' ')
		content684.WriteString(fragment)
		content684.WriteByte(' ')
	}
	fragments = append(fragments, "generic-extra")
	content683.WriteString("generic-extra")

	hits := []knowledge.VisualSearchHit{
		{
			ObjectID:       "object_6",
			ObjectKind:     "notes",
			SourceFileID:   "file_683",
			SourceFileName: "20C114683.pdf",
			Score:          0.032,
			Content:        content683.String(),
			ScoreParts: map[string]any{
				"image_rank": 2,
				"text_rank":  1,
			},
		},
		{
			ObjectID:       "object_5",
			ObjectKind:     "notes",
			SourceFileID:   "file_684",
			SourceFileName: "20C114684.pdf",
			Score:          0.032,
			Content:        content684.String(),
			ScoreParts: map[string]any{
				"image_rank": 1,
				"text_rank":  2,
			},
		},
	}

	got := rerankVisualTextRegionFirstResults(hits, fragments, 2)
	if got[0].SourceFileName != "20C114684.pdf" {
		t.Fatalf("top hit = %+v, want image-rank winner 20C114684.pdf", got[0])
	}
	if got[0].ScoreParts["text_region_match_count"] != 33 {
		t.Fatalf("top text_region_match_count = %v, want 33", got[0].ScoreParts["text_region_match_count"])
	}
	if got[0].ScoreParts["text_region_visual_tiebreak_applied"] != true {
		t.Fatalf("top score parts missing visual tiebreak: %+v", got[0].ScoreParts)
	}
	if got[0].ScoreParts["text_region_image_rank"] != 1 {
		t.Fatalf("top text_region_image_rank = %v, want 1", got[0].ScoreParts["text_region_image_rank"])
	}
}

func TestVisualTextRegionFirstRerankDoesNotLetImageRankOverrideClearTextGap(t *testing.T) {
	fragments := []string{"a", "b", "c", "d", "e"}
	hits := []knowledge.VisualSearchHit{
		{
			ObjectID:       "text",
			ObjectKind:     "notes",
			SourceFileID:   "file_text",
			SourceFileName: "20C114683.pdf",
			Score:          0.01,
			Content:        "a b c d e",
			ScoreParts: map[string]any{
				"image_rank": 5,
				"text_rank":  1,
			},
		},
		{
			ObjectID:       "image",
			ObjectKind:     "notes",
			SourceFileID:   "file_image",
			SourceFileName: "20C114684.pdf",
			Score:          0.99,
			Content:        "a b c",
			ScoreParts: map[string]any{
				"image_rank": 1,
				"text_rank":  2,
			},
		},
	}

	got := rerankVisualTextRegionFirstResults(hits, fragments, 2)
	if got[0].SourceFileName != "20C114683.pdf" {
		t.Fatalf("top hit = %+v, want clear text winner 20C114683.pdf", got[0])
	}
	if got[0].ScoreParts["text_region_visual_tiebreak_applied"] != false {
		t.Fatalf("clear text winner should not use visual tiebreak: %+v", got[0].ScoreParts)
	}
}
