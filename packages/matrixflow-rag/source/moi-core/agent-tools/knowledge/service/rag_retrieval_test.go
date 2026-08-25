package service

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"
	"github.com/matrixflow/moi-core/agent-tools/knowledge"
)

type stubRAGExecutor struct {
	results     []*knowledge.SQLExecutionResult
	resultBySQL func(string) *knowledge.SQLExecutionResult
	gotDBNames  []string
	gotSQLs     []string
}

func (s *stubRAGExecutor) ExecuteSQL(_ context.Context, dbName string, sql string) (*knowledge.SQLExecutionResult, error) {
	s.gotDBNames = append(s.gotDBNames, dbName)
	s.gotSQLs = append(s.gotSQLs, sql)
	if s.resultBySQL != nil {
		return s.resultBySQL(sql), nil
	}
	if len(s.results) == 0 {
		return nil, nil
	}
	result := s.results[0]
	s.results = s.results[1:]
	return result, nil
}

type stubRAGEmbedder struct {
	embeddings [][]float64
}

func (s *stubRAGEmbedder) CreateEmbedding(_ context.Context, _, _ string, inputs []string) ([][]float64, error) {
	if len(s.embeddings) == 1 && len(inputs) > 1 {
		out := make([][]float64, 0, len(inputs))
		for range inputs {
			out = append(out, append([]float64(nil), s.embeddings[0]...))
		}
		return out, nil
	}
	return s.embeddings, nil
}

func TestFindRAGFilesTreatsBlankFileIDsAsNotProvided(t *testing.T) {
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}}},
		{Columns: []string{"file_id", "index_version", "file_name", "volume_id", "source_uri"}, Rows: nil},
	}}
	svc := NewFindRAGFiles(Deps{SQLExecutor: exec})

	_, err := svc.Execute(context.Background(), knowledge.FindRAGFilesRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "w",
			DBName:      "d",
			VectorTable: "vector_store",
		},
		FileIDs: []string{" "},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.gotSQLs) < 2 {
		t.Fatalf("expected inspect and search SQL, got %d calls", len(exec.gotSQLs))
	}
	if strings.Contains(exec.gotSQLs[1], "file_id IN") {
		t.Fatalf("blank file_ids should not add a file filter, got %s", exec.gotSQLs[1])
	}
}

func TestSearchRAGChunksTreatsBlankFileIDsAsNotProvided(t *testing.T) {
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:    "w",
			DBName:         "d",
			VectorTable:    "vector_store",
			EmbeddingModel: "bge-m3",
		},
		Keywards: []string{"营业收入"},
		FileIDs:  []string{" "},
		MaxHits:  12,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.gotSQLs) < 3 {
		t.Fatalf("expected inspect, fulltext, and vector SQL, got %d calls", len(exec.gotSQLs))
	}
	if strings.Contains(exec.gotSQLs[1], "file_id IN") {
		t.Fatalf("blank file_ids should not add a fulltext file filter, got %s", exec.gotSQLs[1])
	}
	if strings.Contains(exec.gotSQLs[2], "file_id IN") {
		t.Fatalf("blank file_ids should not add a vector file filter, got %s", exec.gotSQLs[2])
	}
}

func TestSearchRAGChunksIntersectsRequestedFileIDsWithScopedSources(t *testing.T) {
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "w",
			DBName:      "d",
			RAGSources: []knowledge.RAGSource{{
				DBName:         "d",
				VectorTable:    "vector_store",
				EmbeddingModel: "bge-m3",
				FileIDs:        []string{"file_allowed"},
			}},
		},
		Keywards: []string{"营业收入"},
		FileIDs:  []string{"file_allowed", "file_blocked"},
		MaxHits:  12,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.gotSQLs) < 3 {
		t.Fatalf("expected inspect, fulltext, and vector SQL, got %d calls", len(exec.gotSQLs))
	}
	if !strings.Contains(exec.gotSQLs[1], "file_id IN ('file_allowed')") || strings.Contains(exec.gotSQLs[1], "file_blocked") {
		t.Fatalf("fulltext SQL did not use requested/scope intersection: %s", exec.gotSQLs[1])
	}
	if !strings.Contains(exec.gotSQLs[2], "file_id IN ('file_allowed')") || strings.Contains(exec.gotSQLs[2], "file_blocked") {
		t.Fatalf("vector SQL did not use requested/scope intersection: %s", exec.gotSQLs[2])
	}
}

func TestSearchRAGChunksSkipsSemanticImageOnlySource(t *testing.T) {
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:    "w",
			VectorTable:    "text_vec",
			EmbeddingModel: "bge-m3",
			RAGSources: []knowledge.RAGSource{
				{
					SemanticModelID: 1,
					DBName:          "kb_text",
					VectorTable:     "text_vec",
					EmbeddingModel:  "bge-m3",
					FileIDs:         []string{"file_text"},
				},
				{
					SemanticModelID:         2,
					DBName:                  "kb_image",
					ImageVectorTable:        "image_vec",
					ImageEmbeddingModel:     "clip-vit-large",
					ImageEmbeddingDimension: 2,
					ImagePreprocessVersion:  "clip-v1",
					ImageDistanceMetric:     "cosine",
					FileIDs:                 []string{"file_image"},
				},
			},
		},
		Keywards: []string{"revenue"},
		MaxHits:  12,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.gotSQLs) != 3 {
		t.Fatalf("SQL calls = %d, want only the text-capable semantic source", len(exec.gotSQLs))
	}
	for idx, dbName := range exec.gotDBNames {
		if dbName != "kb_text" {
			t.Fatalf("db call %d = %q, want kb_text only: %#v", idx, dbName, exec.gotDBNames)
		}
	}
	for _, sql := range exec.gotSQLs {
		if strings.Contains(sql, "file_image") || strings.Contains(sql, "kb_image") {
			t.Fatalf("image-only semantic source leaked into text RAG SQL: %s", sql)
		}
	}
}

func TestSearchRAGChunksRAGSourceDoesNotInheritStructuredDBName(t *testing.T) {
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "w",
			DBName:      "structured_db",
			RAGSources: []knowledge.RAGSource{{
				VectorTable:    "rag_db.vector_store",
				EmbeddingModel: "bge-m3",
			}},
		},
		Keywards: []string{"education"},
		MaxHits:  12,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.gotDBNames) < 3 {
		t.Fatalf("expected inspect, fulltext, and vector calls, got %d", len(exec.gotDBNames))
	}
	for i, dbName := range exec.gotDBNames {
		if dbName != "" {
			t.Fatalf("RAG call %d inherited structured DBName %q", i, dbName)
		}
	}
}

func TestSearchRAGChunksReturnsEmptyWhenRequestedFileIDsOutsideScope(t *testing.T) {
	exec := &stubRAGExecutor{}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	resp, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "w",
			DBName:      "d",
			RAGSources: []knowledge.RAGSource{{
				DBName:         "d",
				VectorTable:    "vector_store",
				EmbeddingModel: "bge-m3",
				FileIDs:        []string{"file_allowed"},
			}},
		},
		Keywards: []string{"营业收入"},
		FileIDs:  []string{"file_blocked"},
		MaxHits:  12,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || len(resp.Chunks) != 0 || resp.RowCount != 0 {
		t.Fatalf("response = %+v, want empty result", resp)
	}
	if len(exec.gotSQLs) != 0 {
		t.Fatalf("executor should not be called when requested files are outside scope, got %d SQLs", len(exec.gotSQLs))
	}
}

func TestSearchRAGChunksExpandsFileScopedTableByChunkIndexSection(t *testing.T) {
	tableMeta0 := `{"file_name":"report.pdf","chunk_type":"table","parent_index":66,"chunk_index":20,"chunk_index_scope":"file","chunk_start":0,"chunk_end":512,"chunk_id":"f_chunk_20"}`
	tableMeta1 := `{"file_name":"report.pdf","chunk_type":"text","parent_index":67,"chunk_index":21,"chunk_index_scope":"file","chunk_start":448,"chunk_end":960,"chunk_id":"f_chunk_21"}`
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}, {"chunk_index"}}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "chunk_index_scope", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: [][]any{
			{"fulltext", "chunk", "<table><tr><td>营业收入</td></tr>", tableMeta0, "f", "100", 20, "file", 66, 0, 512, 1},
		}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "chunk_index_scope", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: nil},
		{Columns: []string{"section_id", "chunk_start", "chunk_end"}, Rows: [][]any{
			{"sec_f_4", 20, 21},
		}},
		{Columns: []string{"level", "content", "meta", "file_id", "index_version", "chunk_index", "chunk_index_scope", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: [][]any{
			{"chunk", "<table><tr><td>营业收入</td></tr>", tableMeta0, "f", "100", 20, "file", 66, 0, 512, 1},
			{"chunk", "table continuation", tableMeta1, "f", "100", 21, "file", 67, 448, 960, 1},
		}},
		{Columns: []string{"level", "content", "meta", "file_id", "markdown_file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: nil},
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	resp, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:    "w",
			DBName:         "d",
			VectorTable:    "vector_store",
			EmbeddingModel: "bge-m3",
		},
		Keywards: []string{"营业收入"},
		MaxHits:  12,
		MaxRows:  1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Chunks) != 2 {
		t.Fatalf("expected full file-scoped table section, got %d: %+v", len(resp.Chunks), resp.Chunks)
	}
	if resp.Chunks[0].Content != "<table><tr><td>营业收入</td></tr>" || resp.Chunks[1].Content != "table continuation" {
		t.Fatalf("unexpected expanded chunks: %+v", resp.Chunks)
	}
	if len(exec.gotSQLs) < 5 {
		t.Fatalf("expected inspect, candidate routes, section lookup, and expansion SQLs, got %d calls", len(exec.gotSQLs))
	}
	sectionSQL := exec.gotSQLs[3]
	if !strings.Contains(sectionSQL, "level = 'section'") ||
		!strings.Contains(sectionSQL, "JSON_UNQUOTE(JSON_EXTRACT(meta, '$.chunk_index_scope')) = 'file'") ||
		!strings.Contains(sectionSQL, "CAST(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.chunk_start')) AS SIGNED) <= 20") {
		t.Fatalf("section lookup should use file-scoped chunk range, got %s", sectionSQL)
	}
	expandSQL := exec.gotSQLs[4]
	if !strings.Contains(expandSQL, "BETWEEN 20 AND 21") ||
		strings.Contains(expandSQL, "BETWEEN 66 AND 66") {
		t.Fatalf("file-scoped table expansion should use chunk_index range, got %s", expandSQL)
	}
}

func TestSearchRAGChunksAddsTableEmbeddedImageRefs(t *testing.T) {
	const tableImageFileID = "00000000-0000-0000-0000-000000000001"
	const embeddedImageFileID = "00000000-0000-0000-0000-000000000002"
	tableMeta := `{"file_name":"report.pdf","source_block_type":"TABLE","chunk_type":"table","block_uuid":"table-block","parent_index":7,"chunk_index":3,"chunk_start":0,"chunk_end":120,"chunk_id":"table_chunk","image_url":"` + tableImageFileID + `","embedded_block_uuids":["image-block","other-image-block"]}`
	imageMeta := `{"file_name":"report.pdf","source_block_type":"IMAGE","block_uuid":"image-block","chunk_id":"image_chunk","image_url":"` + embeddedImageFileID + `","page_image_file_id":"page-image","object_kind":"table_embedded_image"}`
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}, {"chunk_index"}}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: [][]any{
			{"fulltext", "chunk", "<table><tr><td>[image:image-block]</td><td>Reject</td></tr></table>", tableMeta, "f", "100", 3, 7, 0, 120, 1},
		}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: nil},
		{Columns: []string{"level", "content", "meta", "file_id", "markdown_file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: [][]any{
			{"chunk", "<table><tr><td>[image:image-block]</td><td>Reject</td></tr></table>", tableMeta, "f", "", "100", 3, 7, 0, 120, 1},
		}},
		{Columns: []string{"level", "content", "meta", "file_id", "markdown_file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: [][]any{
			{"chunk", "embedded image ocr", imageMeta, "f", "", "100", 4, 7, 121, 160, 1},
		}},
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	resp, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:    "w",
			DBName:         "d",
			VectorTable:    "vector_store",
			EmbeddingModel: "bge-m3",
		},
		Keywards: []string{"Reject"},
		MaxHits:  12,
		MaxRows:  1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Chunks) != 1 {
		t.Fatalf("chunks = %+v", resp.Chunks)
	}
	chunk := resp.Chunks[0]
	if chunk.ImageFileID != tableImageFileID {
		t.Fatalf("table image id = %q, want %q", chunk.ImageFileID, tableImageFileID)
	}
	if len(chunk.VisualRefs) != 2 {
		t.Fatalf("visual refs = %+v", chunk.VisualRefs)
	}
	if chunk.VisualRefs[0].ObjectID != "table-block" || chunk.VisualRefs[0].ImageFileID != tableImageFileID {
		t.Fatalf("visual refs = %+v", chunk.VisualRefs)
	}
	if chunk.VisualRefs[1].ObjectID != "image-block" || chunk.VisualRefs[1].ImageFileID != embeddedImageFileID || chunk.VisualRefs[1].PageImageFileID != "page-image" {
		t.Fatalf("visual refs = %+v", chunk.VisualRefs)
	}
	if len(exec.gotSQLs) < 5 || !strings.Contains(exec.gotSQLs[4], "source_block_type')) = 'IMAGE'") || !strings.Contains(exec.gotSQLs[4], "'image-block'") || strings.Contains(exec.gotSQLs[4], "other-image-block") {
		t.Fatalf("embedded image lookup SQL missing, sqls=%+v", exec.gotSQLs)
	}
}

func TestSearchRAGChunksAddsTableImageVisualRef(t *testing.T) {
	const tableImageFileID = "00000000-0000-0000-0000-000000000003"
	tableMeta := `{"file_name":"report.pdf","source_block_type":"TABLE","chunk_type":"table","block_uuid":"table-block","parent_index":7,"chunk_index":3,"chunk_start":0,"chunk_end":120,"chunk_id":"table_chunk","image_url":"` + tableImageFileID + `"}`
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}, {"chunk_index"}}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: [][]any{
			{"fulltext", "chunk", "<table><tr><td>Reject</td></tr></table>", tableMeta, "f", "100", 3, 7, 0, 120, 1},
		}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: nil},
		{Columns: []string{"level", "content", "meta", "file_id", "markdown_file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: [][]any{
			{"chunk", "<table><tr><td>Reject</td></tr></table>", tableMeta, "f", "", "100", 3, 7, 0, 120, 1},
		}},
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	resp, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:    "w",
			DBName:         "d",
			VectorTable:    "vector_store",
			EmbeddingModel: "bge-m3",
		},
		Keywards: []string{"Reject"},
		MaxHits:  12,
		MaxRows:  1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Chunks) != 1 {
		t.Fatalf("chunks = %+v", resp.Chunks)
	}
	refs := resp.Chunks[0].VisualRefs
	if len(refs) != 1 {
		t.Fatalf("visual refs = %+v", refs)
	}
	if refs[0].ObjectID != "table-block" || refs[0].ImageFileID != tableImageFileID {
		t.Fatalf("visual refs = %+v", refs)
	}
}

func TestSearchRAGChunksAddsAdjacentImageVisualRef(t *testing.T) {
	const adjacentImageFileID = "00000000-0000-0000-0000-000000000004"
	textMeta := `{"file_name":"report.pdf","source_block_type":"TEXT","block_uuid":"text-block","parent_index":400,"chunk_index":10,"chunk_start":0,"chunk_end":80,"chunk_id":"text_chunk"}`
	imageMeta := `{"file_name":"report.pdf","source_block_type":"IMAGE","block_uuid":"image-block","parent_index":399,"chunk_index":9,"chunk_start":0,"chunk_end":20,"chunk_id":"image_chunk","image_url":"` + adjacentImageFileID + `"}`
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}, {"chunk_index"}}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: [][]any{
			{"fulltext", "chunk", "Thickness measure points (4points in total): 测量点 （一共四点）", textMeta, "f", "100", 10, 400, 0, 80, 1},
		}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: nil},
		{Columns: []string{"level", "content", "meta", "file_id", "markdown_file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: [][]any{
			{"chunk", "Thickness measure points (4points in total): 测量点 （一共四点）", textMeta, "f", "", "100", 10, 400, 0, 80, 1},
		}},
		{Columns: []string{"level", "content", "meta", "file_id", "markdown_file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}, Rows: [][]any{
			{"chunk", "1 2 3 4", imageMeta, "f", "", "100", 9, 399, 0, 20, 1},
		}},
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	resp, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:    "w",
			DBName:         "d",
			VectorTable:    "vector_store",
			EmbeddingModel: "bge-m3",
		},
		Keywards: []string{"CSP", "thickness", "point"},
		MaxHits:  12,
		MaxRows:  1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Chunks) != 1 {
		t.Fatalf("chunks = %+v", resp.Chunks)
	}
	refs := resp.Chunks[0].VisualRefs
	if len(refs) != 1 {
		t.Fatalf("visual refs = %+v", refs)
	}
	if refs[0].ObjectID != "image-block" || refs[0].ImageFileID != adjacentImageFileID {
		t.Fatalf("visual refs = %+v", refs)
	}
	if len(exec.gotSQLs) < 5 || !strings.Contains(exec.gotSQLs[4], "source_block_type')) = 'IMAGE'") || !strings.Contains(exec.gotSQLs[4], "398, 399, 401, 402") {
		t.Fatalf("adjacent image lookup SQL missing, sqls=%+v", exec.gotSQLs)
	}
}

func TestFocusRAGTableEvidenceRowsKeepsRowspanContext(t *testing.T) {
	noiseRows := strings.Repeat(`<tr><td>Other area</td><td>Noise row</td><td>not relevant</td></tr>`, 20)
	table := `<table>` + noiseRows + `
<tr><td rowspan="5">SOP area</td><td>Solder Extrusion / Bridge</td><td>Reject if any bridging</td></tr>
<tr><td>Missing Solder</td><td>Not allowed</td></tr>
<tr><td>Insufficient Solder</td><td>Not allowed</td></tr>
<tr><td>Scratch / Damage / Deform刮伤/损伤/变形</td><td>超出面积1/2以上或导致高度不足或平面尺寸不足,拒收</td></tr>
<tr><td>No Coin</td><td>Not allowed</td></tr>
</table>`
	group := []knowledge.RAGChunkHit{{
		Content:   table,
		FileID:    "f",
		ChunkType: "table",
	}}

	focused := focusRAGTableEvidenceRows(group, []string{"SOP", "划伤"})
	if len(focused) != 1 {
		t.Fatalf("focused chunks = %+v", focused)
	}
	content := focused[0].Content
	if !strings.Contains(content, "SOP area") {
		t.Fatalf("focused table should keep rowspan context, got %s", content)
	}
	if !strings.Contains(content, "Scratch / Damage / Deform") || !strings.Contains(content, "超出面积1/2以上或导致高度不足或平面尺寸不足") {
		t.Fatalf("focused table should keep the target SOP damage row, got %s", content)
	}
	if strings.Contains(content, "Noise row") {
		t.Fatalf("focused table should drop unrelated long table rows, got %s", content)
	}
}

func TestFocusRAGTableEvidenceRowsRendersHighestScoringRowsFirst(t *testing.T) {
	noiseRows := strings.Repeat(`<tr><td>Other area</td><td>Noise row</td><td>not relevant</td></tr>`, 20)
	table := `<table>` + noiseRows + `
<tr><td>LGA area</td><td>Scratch</td><td>unrelated scratch rule</td></tr>
<tr><td>SOP area</td><td>Scratch / Damage / Deform划伤/损伤/变形</td><td>通用规范：超出面积1/2以上,拒收</td></tr>
</table>`
	group := []knowledge.RAGChunkHit{{Content: table, FileID: "f", ChunkType: "table"}}

	focused := focusRAGTableEvidenceRows(group, []string{"SOP", "划伤", "Scratch", "通用规范"})
	if len(focused) != 1 {
		t.Fatalf("focused chunks = %d, want unchanged folded count 1", len(focused))
	}
	content := focused[0].Content
	target := strings.Index(content, "SOP area")
	lga := strings.Index(content, "LGA area")
	if target < 0 || lga < 0 || target > lga {
		t.Fatalf("highest-scoring target row should render first: %s", content)
	}
}

type focusedRAGModelViewSearchStub struct {
	chunks []knowledge.RAGChunkHit
}

func (s focusedRAGModelViewSearchStub) Execute(_ context.Context, req knowledge.SearchRAGChunksRequest) (*knowledge.SearchRAGChunksResponse, error) {
	return &knowledge.SearchRAGChunksResponse{
		Keywards: append([]string(nil), req.Keywards...),
		Chunks:   append([]knowledge.RAGChunkHit(nil), s.chunks...),
		RowCount: len(s.chunks),
	}, nil
}

func TestFocusedRAGTableHighestScoringRowRemainsFirstInModelView(t *testing.T) {
	noiseRows := strings.Repeat(`<tr><td>Other area</td><td>Noise row</td><td>not relevant</td></tr>`, 20)
	table := `<table>` + noiseRows + `
<tr><td>LGA area</td><td>Scratch</td><td>` + strings.Repeat("low score filler ", 20) + `</td></tr>
<tr><td>SOP area</td><td>Damage / Deform划伤/损伤/变形</td><td>通用规范：超出面积1/2以上,拒收</td></tr>
</table>`
	keywords := []string{"Scratch", "SOP", "划伤", "通用规范"}
	focused := focusRAGTableEvidenceRows([]knowledge.RAGChunkHit{{
		Content:   table,
		FileID:    "file_1",
		ChunkID:   "chunk_1",
		ChunkType: "table",
	}}, keywords)
	if len(focused) != 1 || !strings.HasPrefix(focused[0].Content, "<table><tr><td>SOP area") {
		t.Fatalf("focused table does not start with highest-scoring row: %+v", focused)
	}

	tool := knowledge.NewSearchRAGChunksTool(knowledge.Options{
		Registry: &knowledge.Registry{SearchRAGChunks: focusedRAGModelViewSearchStub{chunks: focused}},
	})
	invocationTool := tool.(agentruntimev2.InvocationTool)
	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "call_1"}, json.RawMessage(`{"keywords":["Scratch","SOP","划伤","通用规范"],"max_hits":1}`))
	if err != nil {
		t.Fatalf("ExecuteInvocation() error = %v", err)
	}
	var payload struct {
		Items []struct {
			ContentPreview string `json:"content_preview"`
		} `json:"items"`
	}
	modelContent := agentruntimev2.ToolResultModelVisibleContent(*result)
	if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if len(payload.Items) != 1 || !strings.Contains(payload.Items[0].ContentPreview, "SOP area") {
		t.Fatalf("model preview lost highest-scoring table row: %+v", payload.Items)
	}
}

func TestSearchRAGChunksRequiresEmbeddingModelFromScopeOrConfig(t *testing.T) {
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}, {"chunk_index"}}},
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "w",
			DBName:      "d",
			VectorTable: "vector_store",
		},
		Keywards: []string{"营业收入"},
		MaxHits:  12,
	})
	if err == nil || !strings.Contains(err.Error(), "embedding_model required") {
		t.Fatalf("want embedding_model required error, got %v", err)
	}
}

func TestSearchRAGChunksFiltersCurrentIndexVersion(t *testing.T) {
	exec := &stubRAGExecutor{}
	exec.resultBySQL = func(sql string) *knowledge.SQLExecutionResult {
		switch {
		case strings.HasPrefix(sql, "SHOW COLUMNS"):
			return &knowledge.SQLExecutionResult{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}, {"chunk_index"}, {"disabled"}}}
		case strings.Contains(sql, "rag_fulltext_candidates"):
			return &knowledge.SQLExecutionResult{
				Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "score"},
				Rows: [][]any{{
					"fulltext",
					"chunk",
					"content hit",
					`{"chunk_id":"chunk-3"}`,
					"file_a",
					"7",
					3,
					1,
				}},
			}
		case strings.Contains(sql, "rag_vector_candidates"):
			return &knowledge.SQLExecutionResult{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "score"}, Rows: nil}
		default:
			return &knowledge.SQLExecutionResult{}
		}
	}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	resp, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "w",
			DBName:      "d",
			RAGSources: []knowledge.RAGSource{{
				DBName:              "d",
				SemanticModelID:     42,
				VectorTable:         "vector_store",
				EmbeddingModel:      "bge-m3",
				FileIDs:             []string{"file_a"},
				SourceRowIDs:        []string{"source_a", "source_other"},
				SourceRowIDByFileID: map[string]string{"file_a": "source_a"},
				CurrentIndexVersionByFileID: map[string]int64{
					"file_a": 7,
				},
			}},
		},
		Keywards: []string{"营业收入"},
		MaxHits:  12,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Chunks) != 1 {
		t.Fatalf("chunks = %+v, want one hit", resp.Chunks)
	}
	if resp.Chunks[0].SemanticModelID != 42 || resp.Chunks[0].SourceRowID != "source_a" {
		t.Fatalf("chunk owner = model %d source %q, want model 42 source_a", resp.Chunks[0].SemanticModelID, resp.Chunks[0].SourceRowID)
	}
	if len(exec.gotSQLs) < 3 {
		t.Fatalf("expected inspect, fulltext, vector SQLs, got %d", len(exec.gotSQLs))
	}
	fulltextSQL := exec.gotSQLs[1]
	if !strings.Contains(fulltextSQL, "(file_id = 'file_a' AND index_version = '7')") {
		t.Fatalf("fulltext SQL missing current version filter: %s", fulltextSQL)
	}
	for _, sql := range exec.gotSQLs {
		if strings.Contains(sql, "knowledge_base_chunk_recall_stats") {
			t.Fatalf("read search should not write recall stats: %s", sql)
		}
	}
}

func TestSearchRAGChunksFiltersIndexVersionConstraintZeroAndNull(t *testing.T) {
	exec := &stubRAGExecutor{}
	exec.resultBySQL = func(sql string) *knowledge.SQLExecutionResult {
		switch {
		case strings.HasPrefix(sql, "SHOW COLUMNS"):
			return &knowledge.SQLExecutionResult{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}, {"chunk_index"}, {"disabled"}}}
		case strings.Contains(sql, "rag_fulltext_candidates"):
			return &knowledge.SQLExecutionResult{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "score"}, Rows: nil}
		case strings.Contains(sql, "rag_vector_candidates"):
			return &knowledge.SQLExecutionResult{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "score"}, Rows: nil}
		default:
			return &knowledge.SQLExecutionResult{}
		}
	}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "w",
			DBName:      "d",
			RAGSources: []knowledge.RAGSource{{
				DBName:         "d",
				VectorTable:    "vector_store",
				EmbeddingModel: "bge-m3",
				FileIDs:        []string{"file_zero", "file_null"},
				IndexVersionConstraintByFileID: map[string]knowledge.RAGIndexVersionConstraint{
					"file_zero": {Kind: knowledge.RAGIndexVersionConstraintValue, Value: 0},
					"file_null": {Kind: knowledge.RAGIndexVersionConstraintNull},
				},
			}},
		},
		Keywards: []string{"历史文档"},
		MaxHits:  12,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.gotSQLs) < 3 {
		t.Fatalf("expected inspect, fulltext, vector SQLs, got %d", len(exec.gotSQLs))
	}
	fulltextSQL := exec.gotSQLs[1]
	if !strings.Contains(fulltextSQL, "(file_id = 'file_null' AND index_version IS NULL)") {
		t.Fatalf("fulltext SQL missing NULL index version filter: %s", fulltextSQL)
	}
	if !strings.Contains(fulltextSQL, "(file_id = 'file_zero' AND index_version = '0')") {
		t.Fatalf("fulltext SQL missing zero index version filter: %s", fulltextSQL)
	}
	if strings.Contains(fulltextSQL, "file_id IN ('file_null','file_zero')") || strings.Contains(fulltextSQL, "file_id IN ('file_zero','file_null')") {
		t.Fatalf("fulltext SQL should not fall back to bare file_id for constrained files: %s", fulltextSQL)
	}
}

func TestSearchRAGChunksSupportsVectorTableWithoutIndexVersionColumn(t *testing.T) {
	exec := &stubRAGExecutor{}
	exec.resultBySQL = func(sql string) *knowledge.SQLExecutionResult {
		switch {
		case strings.HasPrefix(sql, "SHOW COLUMNS"):
			return &knowledge.SQLExecutionResult{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"level"}, {"content"}, {"meta"}, {"embedding"}, {"chunk_index"}, {"disabled"}}}
		case strings.Contains(sql, "rag_fulltext_candidates"):
			return &knowledge.SQLExecutionResult{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "score"}, Rows: nil}
		case strings.Contains(sql, "rag_vector_candidates"):
			return &knowledge.SQLExecutionResult{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "score"}, Rows: nil}
		default:
			return &knowledge.SQLExecutionResult{}
		}
	}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "w",
			DBName:      "d",
			RAGSources: []knowledge.RAGSource{{
				DBName:         "d",
				VectorTable:    "vector_store",
				EmbeddingModel: "bge-m3",
				FileIDs:        []string{"file_a"},
				IndexVersionConstraintByFileID: map[string]knowledge.RAGIndexVersionConstraint{
					"file_a": {Kind: knowledge.RAGIndexVersionConstraintValue, Value: 7},
				},
			}},
		},
		Keywards: []string{"历史文档"},
		MaxHits:  12,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.gotSQLs) < 3 {
		t.Fatalf("expected inspect, fulltext, vector SQLs, got %d", len(exec.gotSQLs))
	}
	fulltextSQL := exec.gotSQLs[1]
	if !strings.Contains(fulltextSQL, "NULL AS index_version") {
		t.Fatalf("fulltext SQL should project NULL index_version for unversioned vector table: %s", fulltextSQL)
	}
	if strings.Contains(fulltextSQL, "index_version = '7'") {
		t.Fatalf("fulltext SQL should not filter missing index_version column: %s", fulltextSQL)
	}
	if !strings.Contains(fulltextSQL, "file_id IN ('file_a')") {
		t.Fatalf("fulltext SQL should keep file scope for unversioned vector table: %s", fulltextSQL)
	}
}

func TestSearchRAGChunksKeepsAssociatedFilesWithoutCurrentIndexVersion(t *testing.T) {
	exec := &stubRAGExecutor{}
	exec.resultBySQL = func(sql string) *knowledge.SQLExecutionResult {
		switch {
		case strings.HasPrefix(sql, "SHOW COLUMNS"):
			return &knowledge.SQLExecutionResult{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}, {"chunk_index"}, {"disabled"}}}
		case strings.Contains(sql, "rag_fulltext_candidates"):
			return &knowledge.SQLExecutionResult{
				Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "score"},
				Rows: [][]any{{
					"fulltext",
					"chunk",
					"legacy workflow content",
					`{"chunk_id":"chunk-legacy"}`,
					"file_legacy",
					"3",
					3,
					1,
				}},
			}
		case strings.Contains(sql, "rag_vector_candidates"):
			return &knowledge.SQLExecutionResult{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "score"}, Rows: nil}
		default:
			return &knowledge.SQLExecutionResult{}
		}
	}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	resp, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "w",
			DBName:      "d",
			RAGSources: []knowledge.RAGSource{{
				DBName:          "d",
				SemanticModelID: 42,
				VectorTable:     "vector_store",
				EmbeddingModel:  "bge-m3",
				FileIDs:         []string{"file_a", "file_legacy"},
				CurrentIndexVersionByFileID: map[string]int64{
					"file_a": 7,
				},
			}},
		},
		Keywards: []string{"历史文档"},
		MaxHits:  12,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Chunks) != 1 || resp.Chunks[0].FileID != "file_legacy" {
		t.Fatalf("chunks = %+v, want legacy associated file hit", resp.Chunks)
	}
	if len(exec.gotSQLs) < 3 {
		t.Fatalf("expected inspect, fulltext, vector SQLs, got %d", len(exec.gotSQLs))
	}
	fulltextSQL := exec.gotSQLs[1]
	if !strings.Contains(fulltextSQL, "(file_id = 'file_a' AND index_version = '7')") {
		t.Fatalf("fulltext SQL missing current version filter for versioned file: %s", fulltextSQL)
	}
	if !strings.Contains(fulltextSQL, "file_id IN ('file_legacy')") {
		t.Fatalf("fulltext SQL missing associated unversioned file: %s", fulltextSQL)
	}
}

func TestSearchRAGChunksUsesRAGSourceDBNameInsteadOfStructuredScopeDB(t *testing.T) {
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "w",
			DBName:      "structured_db",
			RAGSources: []knowledge.RAGSource{{
				DBName:         "rag_db",
				VectorTable:    "vector_store",
				EmbeddingModel: "bge-m3",
				FileIDs:        []string{"file_1"},
			}},
		},
		Keywards: []string{"客户版本号"},
		MaxHits:  12,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.gotDBNames) != 3 {
		t.Fatalf("db calls = %#v, want inspect/fulltext/vector", exec.gotDBNames)
	}
	for _, dbName := range exec.gotDBNames {
		if dbName != "rag_db" {
			t.Fatalf("db calls = %#v, want all rag_db", exec.gotDBNames)
		}
	}
}

func TestSearchRAGChunksAllowsQualifiedVectorTableWithoutRAGSourceDBName(t *testing.T) {
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "w",
			DBName:      "structured_db",
			RAGSources: []knowledge.RAGSource{{
				VectorTable:    "moi.vector_store",
				EmbeddingModel: "bge-m3",
				FileIDs:        []string{"file_1"},
			}},
		},
		Keywards: []string{"客户版本号"},
		MaxHits:  12,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, dbName := range exec.gotDBNames {
		if dbName != "" {
			t.Fatalf("db calls = %#v, want qualified table to run without selected db", exec.gotDBNames)
		}
	}
	if len(exec.gotSQLs) == 0 || !strings.Contains(exec.gotSQLs[0], "`moi`.`vector_store`") {
		t.Fatalf("inspect SQL = %#v, want qualified vector table", exec.gotSQLs)
	}
}

func TestSearchRAGChunksAllowsUnqualifiedRAGSourceWithoutDBName(t *testing.T) {
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		{Columns: []string{"Field"}, Rows: [][]any{{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"}}},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
		{Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "score"}, Rows: nil},
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	_, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "w",
			DBName:      "structured_db",
			RAGSources: []knowledge.RAGSource{{
				VectorTable:    "vector_store",
				EmbeddingModel: "bge-m3",
				FileIDs:        []string{"file_1"},
			}},
		},
		Keywards: []string{"客户版本号"},
		MaxHits:  12,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exec.gotDBNames) != 3 {
		t.Fatalf("db calls = %#v, want inspect/fulltext/vector", exec.gotDBNames)
	}
	for _, dbName := range exec.gotDBNames {
		if dbName != "" {
			t.Fatalf("db calls = %#v, want tenant connection default database", exec.gotDBNames)
		}
	}
	if len(exec.gotSQLs) == 0 || !strings.Contains(exec.gotSQLs[0], "SHOW COLUMNS FROM `vector_store`") {
		t.Fatalf("inspect SQL = %#v, want unqualified vector table", exec.gotSQLs)
	}
}

func TestIssue11520RAGChunkHitPreservesTranscriptTimeRange(t *testing.T) {
	hit, err := ragChunkHitFromRecord(map[string]any{
		"content": "opening",
		"file_id": "audio-1",
		"meta":    `{"chunk_id":"chunk-1","start_ms":0,"end_ms":1250}`,
	})
	if err != nil {
		t.Fatalf("ragChunkHitFromRecord: %v", err)
	}
	if hit.StartMS == nil || *hit.StartMS != 0 || hit.EndMS == nil || *hit.EndMS != 1250 {
		t.Fatalf("time range = %v..%v", hit.StartMS, hit.EndMS)
	}
}

func TestRAGChunkHitUsesCanonicalImageFileID(t *testing.T) {
	const fileID = "b7e3cccb-4255-4d3d-827f-e4027390dc5f"
	tests := []struct {
		name string
		meta string
		want string
	}{
		{name: "canonical field", meta: `{"image_file_id":"canonical-image","image_url":"` + fileID + `.jpg"}`, want: "canonical-image"},
		{name: "bare UUID image ref", meta: `{"image_url":"` + fileID + `"}`, want: fileID},
		{name: "bare UUID S3 ref", meta: `{"s3_image_url":"` + fileID + `"}`, want: fileID},
		{name: "bare UUID table ref", meta: `{"table_image_url":"` + fileID + `"}`, want: fileID},
		{name: "local display ref", meta: `{"image_url":"` + fileID + `.jpg"}`, want: fileID},
		{name: "uppercase extension", meta: `{"table_image_url":"` + fileID + `.PNG"}`, want: fileID},
		{name: "bare relative filename", meta: `{"image_url":"logo.png"}`, want: ""},
		{name: "extensionless value", meta: `{"image_url":"legacy-image"}`, want: ""},
		{name: "unsupported extension", meta: `{"image_url":"` + fileID + `.svg"}`, want: ""},
		{name: "external URL", meta: `{"image_url":"https://example.com/image.jpg"}`, want: ""},
		{name: "data URL", meta: `{"image_url":"data:image/png;base64,AAAA"}`, want: ""},
		{name: "cid URL", meta: `{"image_url":"cid:image.jpg"}`, want: ""},
		{name: "blob URL", meta: `{"image_url":"blob:https://example.com/image-id"}`, want: ""},
		{name: "relative path", meta: `{"image_url":"images/` + fileID + `.jpg"}`, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit, err := ragChunkHitFromRecord(map[string]any{"content": "image", "file_id": "source-file", "meta": tt.meta})
			if err != nil {
				t.Fatalf("ragChunkHitFromRecord: %v", err)
			}
			if hit.ImageFileID != tt.want {
				t.Fatalf("image file id = %q, want %q", hit.ImageFileID, tt.want)
			}
		})
	}
}

type ragAnchorPriorityTestChunk struct {
	fileID      string
	fileName    string
	chunkID     string
	content     string
	parentIndex *int
	chunkIndex  *int
}

func ragAnchorPriorityTestColumns() *knowledge.SQLExecutionResult {
	return &knowledge.SQLExecutionResult{
		Columns: []string{"Field"},
		Rows: [][]any{
			{"file_id"}, {"index_version"}, {"level"}, {"content"}, {"meta"}, {"embedding"},
			{"chunk_index"}, {"parent_index"}, {"chunk_start"}, {"chunk_end"}, {"disabled"},
		},
	}
}

func ragAnchorPriorityTestRows(route string, chunks ...ragAnchorPriorityTestChunk) *knowledge.SQLExecutionResult {
	rows := make([][]any, 0, len(chunks))
	for index, chunk := range chunks {
		if chunk.fileID == "" {
			chunk.fileID = "file_" + chunk.chunkID
		}
		if chunk.fileName == "" {
			chunk.fileName = chunk.fileID + ".pdf"
		}
		if chunk.chunkIndex == nil {
			if chunk.parentIndex != nil {
				chunk.chunkIndex = ragAnchorPriorityInt(*chunk.parentIndex)
			} else {
				chunk.chunkIndex = ragAnchorPriorityInt(index)
			}
		}
		var parentIndex any
		var chunkStart any
		var chunkEnd any
		if chunk.parentIndex != nil {
			parentIndex = *chunk.parentIndex
			chunkStart = *chunk.parentIndex
			chunkEnd = *chunk.parentIndex
		}
		rows = append(rows, []any{
			route,
			"chunk",
			chunk.content,
			fmt.Sprintf(`{"chunk_id":%q,"source_file_name":%q}`, chunk.chunkID, chunk.fileName),
			chunk.fileID,
			"v1",
			*chunk.chunkIndex,
			parentIndex,
			chunkStart,
			chunkEnd,
			1,
		})
	}
	return &knowledge.SQLExecutionResult{
		Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"},
		Rows:    rows,
	}
}

func ragAnchorPriorityInt(value int) *int {
	return &value
}

func TestSearchRAGChunksAssignsRetrievalAnchorRanksAfterCandidateSort(t *testing.T) {
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		ragAnchorPriorityTestColumns(),
		ragAnchorPriorityTestRows("fulltext",
			ragAnchorPriorityTestChunk{chunkID: "low", content: "alpha"},
			ragAnchorPriorityTestChunk{chunkID: "high", fileName: "alpha.pdf", content: "alpha beta"},
			ragAnchorPriorityTestChunk{chunkID: "middle", content: "alpha beta"},
		),
		ragAnchorPriorityTestRows("vector_l2"),
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	resp, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:    "workspace",
			DBName:         "db",
			VectorTable:    "rag_chunks",
			EmbeddingModel: "bge-m3",
		},
		Keywards: []string{"alpha", "beta"},
		MaxHits:  10,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(resp.Chunks) != 3 {
		t.Fatalf("chunk count = %d, want 3", len(resp.Chunks))
	}
	for index, wantID := range []string{"high", "middle", "low"} {
		if got := resp.Chunks[index].ChunkID; got != wantID {
			t.Fatalf("chunk %d = %q, want %q", index, got, wantID)
		}
		if got, want := resp.Chunks[index].RetrievalAnchorRank, index+1; got != want {
			t.Fatalf("chunk %q rank = %d, want %d", wantID, got, want)
		}
	}
}

func TestPlanRAGExpansionsPrioritizesMergedMinimumAnchorRankWithoutReplacingRepresentative(t *testing.T) {
	firstParent := 10
	secondParent := 11
	otherParent := 30
	first := ragCandidate{hit: knowledge.RAGChunkHit{
		FileID: "same-file", IndexVersion: "v1", ChunkID: "representative", ParentIndex: &firstParent,
		Routes: []string{"fulltext"}, Score: 11, RetrievalAnchorRank: 8,
	}, rank: 1}
	second := ragCandidate{hit: knowledge.RAGChunkHit{
		FileID: "same-file", IndexVersion: "v1", ChunkID: "merged-anchor", ParentIndex: &secondParent,
		Routes: []string{"vector_l2"}, Score: 22, RetrievalAnchorRank: 5,
	}, rank: 3}
	other := ragCandidate{hit: knowledge.RAGChunkHit{
		FileID: "other-file", IndexVersion: "v1", ChunkID: "higher-priority", ParentIndex: &otherParent,
		Routes: []string{"vector_l2"}, Score: 33, RetrievalAnchorRank: 2,
	}, rank: 2}

	plans, err := (&searchRAGChunksService{}).planRAGExpansions(context.Background(), []ragCandidate{first, second, other}, 1, 1)
	if err != nil {
		t.Fatalf("planRAGExpansions() error = %v", err)
	}
	if len(plans) != 2 {
		t.Fatalf("plan count = %d, want 2", len(plans))
	}
	if got := plans[0].candidate.hit.ChunkID; got != "higher-priority" {
		t.Fatalf("first plan representative = %q, want higher-priority", got)
	}
	if got := plans[1].candidate.hit.ChunkID; got != "representative" {
		t.Fatalf("merged plan representative = %q, want representative", got)
	}
	if got, want := plans[1].candidate.hit.Routes, []string{"fulltext"}; len(got) != 1 || got[0] != want[0] {
		t.Fatalf("merged plan routes = %v, want %v", got, want)
	}
	if got := plans[1].candidate.hit.Score; got != 11 {
		t.Fatalf("merged plan score = %v, want representative score 11", got)
	}
}

func TestPlanRAGExpansionsUsesStableKeyForEqualAnchorRanks(t *testing.T) {
	plans, err := (&searchRAGChunksService{}).planRAGExpansions(context.Background(), []ragCandidate{
		{hit: knowledge.RAGChunkHit{FileID: "z-file", IndexVersion: "v1", ChunkID: "z", RetrievalAnchorRank: 2}, rank: 1},
		{hit: knowledge.RAGChunkHit{FileID: "a-file", IndexVersion: "v1", ChunkID: "a", RetrievalAnchorRank: 2}, rank: 2},
	}, 0, 0)
	if err != nil {
		t.Fatalf("planRAGExpansions() error = %v", err)
	}
	if got, want := plans[0].candidate.hit.ChunkID, "a"; got != want {
		t.Fatalf("first equal-rank plan = %q, want stable key %q", got, want)
	}
}

func TestSearchRAGChunksUsesCompleteAnchorMapAcrossChainOverlapAndMaxRows(t *testing.T) {
	parentA := 0
	parentC := 3
	parentB := 2
	parentNeighbor := 1
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		ragAnchorPriorityTestColumns(),
		ragAnchorPriorityTestRows("fulltext",
			ragAnchorPriorityTestChunk{fileID: "file", chunkID: "a", content: "one two three", parentIndex: &parentA},
			ragAnchorPriorityTestChunk{fileID: "file", chunkID: "c", content: "one two", parentIndex: &parentC},
			ragAnchorPriorityTestChunk{fileID: "file", chunkID: "b", content: "one", parentIndex: &parentB},
		),
		ragAnchorPriorityTestRows("vector_l2"),
		ragAnchorPriorityTestRows("",
			ragAnchorPriorityTestChunk{fileID: "file", chunkID: "a", content: "one two three", parentIndex: &parentA},
			ragAnchorPriorityTestChunk{fileID: "file", chunkID: "neighbor", content: "context", parentIndex: &parentNeighbor},
			ragAnchorPriorityTestChunk{fileID: "file", chunkID: "b", content: "one", parentIndex: &parentB},
			ragAnchorPriorityTestChunk{fileID: "file", chunkID: "c", content: "one two", parentIndex: &parentC},
		),
		ragAnchorPriorityTestRows(""),
	}}
	svc := NewSearchRAGChunks(Deps{SQLExecutor: exec, Embedder: &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}}})
	resp, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope:    knowledge.WorkspaceScope{WorkspaceID: "workspace", DBName: "db", VectorTable: "rag_chunks", EmbeddingModel: "bge-m3"},
		Keywards: []string{"one", "two", "three"},
		MaxHits:  10,
		MaxRows:  1,
		Before:   1,
		After:    1,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	ranks := map[string]int{}
	for _, chunk := range resp.Chunks {
		ranks[chunk.ChunkID] = chunk.RetrievalAnchorRank
	}
	if got := ranks["a"]; got != 1 {
		t.Fatalf("anchor a rank = %d, want 1", got)
	}
	if got := ranks["c"]; got != 2 {
		t.Fatalf("unvisited-plan anchor c rank = %d, want 2", got)
	}
	if got := ranks["b"]; got != 3 {
		t.Fatalf("anchor b rank = %d, want 3", got)
	}
	if got := ranks["neighbor"]; got != 0 {
		t.Fatalf("expansion-only neighbor rank = %d, want 0", got)
	}
}

func TestSearchRAGChunksKeepsEveryRankedTableAnchorInMergedExpansionGroup(t *testing.T) {
	rankOneContent := `<table><tr><td>SOP A</td><td>needle</td><td>rank-one evidence</td></tr>` +
		strings.Repeat(`<tr><td>noise</td><td>unrelated first table context</td></tr>`, 12) +
		`</table>`
	rankTwoContent := `<table><tr><td>SOP B</td><td>needle</td><td>rank-two evidence</td></tr>` +
		strings.Repeat(`<tr><td>noise</td><td>unrelated second table context</td></tr>`, 12) +
		`</table>`
	if utf8RuneCount(rankOneContent)+utf8RuneCount(rankTwoContent) < ragFocusedTableMinRunes {
		t.Fatal("fixture must enter the table focus path")
	}
	rankOneMeta := `{"chunk_id":"rank-one","chunk_type":"table","block_uuid":"shared-table","page_num":10,"image_file_id":"rank-one-image","page_image_file_id":"rank-one-page-image","object_id":"rank-one-object","object_kind":"table","bbox":[1,2,3,4]}`
	rankTwoMeta := `{"chunk_id":"rank-two","chunk_type":"table","block_uuid":"shared-table","page_num":11,"image_file_id":"rank-two-image","page_image_file_id":"rank-two-page-image","object_id":"rank-two-object","object_kind":"table","bbox":[5,6,7,8]}`
	candidateColumns := []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}
	expansionColumns := []string{"level", "content", "meta", "file_id", "markdown_file_id", "index_version", "chunk_index", "parent_index", "chunk_start", "chunk_end", "score"}
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		ragAnchorPriorityTestColumns(),
		{
			Columns: candidateColumns,
			Rows: [][]any{
				{"fulltext", "chunk", rankOneContent, rankOneMeta, "file", "v1", 10, 10, 10, 10, 1},
				{"fulltext", "chunk", rankTwoContent, rankTwoMeta, "file", "v1", 11, 11, 11, 11, 1},
			},
		},
		{Columns: candidateColumns},
		{
			Columns: expansionColumns,
			Rows: [][]any{
				{"chunk", rankOneContent, rankOneMeta, "file", "", "v1", 10, 10, 10, 10, 1},
				{"chunk", rankTwoContent, rankTwoMeta, "file", "", "v1", 11, 11, 11, 11, 1},
			},
		},
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	resp, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID:    "workspace",
			DBName:         "db",
			VectorTable:    "rag_chunks",
			EmbeddingModel: "bge-m3",
		},
		Keywards: []string{"needle"},
		MaxHits:  10,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got := len(resp.ExpansionSQLs); got != 1 {
		t.Fatalf("expansion SQL count = %d, want one merged plan", got)
	}
	if got := resp.ExpandedGroups; got != 1 {
		t.Fatalf("expanded group count = %d, want one merged group", got)
	}
	if len(resp.Chunks) != 2 {
		ids := make([]string, 0, len(resp.Chunks))
		for _, chunk := range resp.Chunks {
			ids = append(ids, chunk.ChunkID)
		}
		t.Fatalf("focused ranked table anchors = %d, want 2; got IDs=%v", len(resp.Chunks), ids)
	}

	byID := make(map[string]knowledge.RAGChunkHit, len(resp.Chunks))
	for _, chunk := range resp.Chunks {
		byID[chunk.ChunkID] = chunk
	}
	tests := []struct {
		id              string
		rank            int
		page            int
		imageFileID     string
		pageImageFileID string
		objectID        string
		evidence        string
		bbox            []float64
	}{
		{id: "rank-one", rank: 1, page: 10, imageFileID: "rank-one-image", pageImageFileID: "rank-one-page-image", objectID: "rank-one-object", evidence: "rank-one evidence", bbox: []float64{1, 2, 3, 4}},
		{id: "rank-two", rank: 2, page: 11, imageFileID: "rank-two-image", pageImageFileID: "rank-two-page-image", objectID: "rank-two-object", evidence: "rank-two evidence", bbox: []float64{5, 6, 7, 8}},
	}
	for _, tt := range tests {
		chunk, ok := byID[tt.id]
		if !ok {
			t.Fatalf("missing ranked table anchor %q in %+v", tt.id, resp.Chunks)
		}
		if chunk.RetrievalAnchorRank != tt.rank || chunk.PageNumber != tt.page ||
			chunk.ImageFileID != tt.imageFileID || chunk.PageImageFileID != tt.pageImageFileID ||
			chunk.ObjectID != tt.objectID || !reflect.DeepEqual(chunk.BBox, tt.bbox) {
			t.Fatalf("ranked table anchor %q provenance = %+v, want rank=%d page=%d image=%q page_image=%q object=%q bbox=%v",
				tt.id, chunk, tt.rank, tt.page, tt.imageFileID, tt.pageImageFileID, tt.objectID, tt.bbox)
		}
		if !strings.Contains(chunk.Content, tt.evidence) || strings.Contains(chunk.Content, "unrelated") {
			t.Fatalf("ranked table anchor %q focused content = %q, want its matching evidence only", tt.id, chunk.Content)
		}
		if len(chunk.VisualRefs) != 1 || chunk.VisualRefs[0].ChunkID != tt.id ||
			chunk.VisualRefs[0].ImageFileID != tt.imageFileID ||
			chunk.VisualRefs[0].PageImageFileID != tt.pageImageFileID ||
			!reflect.DeepEqual(chunk.VisualRefs[0].BBox, tt.bbox) {
			t.Fatalf("ranked table anchor %q visual refs = %+v, want own provenance", tt.id, chunk.VisualRefs)
		}
	}
}

func TestFocusRAGTableEvidenceRowsKeepsAnchorCarrierProvenance(t *testing.T) {
	noiseRows := strings.Repeat(`<tr><td>noise</td><td>unrelated context</td></tr>`, 24)
	group := []knowledge.RAGChunkHit{
		{
			ChunkID:         "neighbor-table",
			PageNumber:      9,
			ChunkType:       "table",
			Content:         noiseRows,
			ImageFileID:     "neighbor-image",
			PageImageFileID: "neighbor-page-image",
			ObjectID:        "neighbor-object",
			BBox:            []float64{1, 2, 3, 4},
			VisualRefs: []knowledge.RAGImageRef{{
				ChunkID: "neighbor-visual", ImageFileID: "neighbor-visual-image",
			}},
		},
		{
			ChunkID:             "anchor-table",
			PageNumber:          10,
			ChunkType:           "table",
			Content:             `<tr><td>SOP</td><td>target evidence</td></tr>`,
			RetrievalAnchorRank: 7,
			ImageFileID:         "anchor-image",
			PageImageFileID:     "anchor-page-image",
			ObjectID:            "anchor-object",
			BBox:                []float64{5, 6, 7, 8},
			VisualRefs: []knowledge.RAGImageRef{{
				ChunkID: "anchor-visual", ImageFileID: "anchor-visual-image",
			}},
		},
	}

	focused := focusRAGTableEvidenceRows(group, []string{"SOP"})
	if len(focused) != 1 {
		t.Fatalf("focused chunk count = %d, want 1", len(focused))
	}
	got := focused[0]
	if got.ChunkID != "anchor-table" || got.PageNumber != 10 || got.ImageFileID != "anchor-image" || got.PageImageFileID != "anchor-page-image" || got.ObjectID != "anchor-object" {
		t.Fatalf("surviving table carrier metadata = %+v, want anchor carrier", got)
	}
	if len(got.BBox) != 4 || got.BBox[0] != 5 || got.BBox[3] != 8 {
		t.Fatalf("surviving table bbox = %v, want anchor bbox", got.BBox)
	}
	if len(got.VisualRefs) != 1 || got.VisualRefs[0].ChunkID != "anchor-visual" || got.VisualRefs[0].ImageFileID != "anchor-visual-image" {
		t.Fatalf("surviving table visual refs = %+v, want anchor refs", got.VisualRefs)
	}
	if got.RetrievalAnchorRank != 7 {
		t.Fatalf("surviving table anchor rank = %d, want 7", got.RetrievalAnchorRank)
	}
	if !strings.Contains(got.Content, "target evidence") || strings.Contains(got.Content, "unrelated context") {
		t.Fatalf("focused content = %q, want anchor evidence without neighbor evidence", got.Content)
	}
}

func TestFocusRAGTableEvidenceRowsKeepsCarrierContentWhenNeighborAlsoMatches(t *testing.T) {
	neighborRows := strings.Repeat(`<tr><td>SOP</td><td>page nine neighbor evidence</td></tr>`, 20)
	group := []knowledge.RAGChunkHit{
		{
			ChunkID:     "page-nine-neighbor",
			PageNumber:  9,
			ChunkType:   "table",
			Content:     neighborRows,
			ImageFileID: "page-nine-image",
		},
		{
			ChunkID:             "page-ten-anchor",
			PageNumber:          10,
			ChunkType:           "table",
			Content:             `<tr><td>SOP</td><td>page ten anchor rule</td></tr>`,
			RetrievalAnchorRank: 7,
			ImageFileID:         "page-ten-image",
		},
	}

	focused := focusRAGTableEvidenceRows(group, []string{"SOP"})
	if len(focused) != 1 {
		t.Fatalf("focused chunk count = %d, want 1", len(focused))
	}
	got := focused[0]
	if got.ChunkID != "page-ten-anchor" || got.PageNumber != 10 || got.ImageFileID != "page-ten-image" || got.RetrievalAnchorRank != 7 {
		t.Fatalf("surviving carrier = %+v, want page ten anchor", got)
	}
	if strings.Contains(got.Content, "page nine neighbor evidence") || !strings.Contains(got.Content, "page ten anchor rule") {
		t.Fatalf("carrier content must originate from the surviving carrier, got %q", got.Content)
	}
}

func TestFocusRAGTableEvidenceRowsKeepsOriginalRankedTableWhenNoRowMatches(t *testing.T) {
	rankOneContent := strings.Repeat(`<tr><td>needle</td><td>rank one evidence</td></tr>`, 10)
	rankTwoContent := strings.Repeat(`<tr><td>vector-only evidence</td><td>rank two rule</td></tr>`, 10)
	if utf8RuneCount(rankOneContent)+utf8RuneCount(rankTwoContent) < ragFocusedTableMinRunes {
		t.Fatal("fixture must enter the table focus path")
	}
	group := []knowledge.RAGChunkHit{
		{ChunkID: "rank-one", ChunkType: "table", Content: rankOneContent, RetrievalAnchorRank: 1},
		{ChunkID: "rank-two", ChunkType: "table", Content: rankTwoContent, RetrievalAnchorRank: 2},
	}

	got := focusRAGTableEvidenceRows(group, []string{"needle"})
	if len(got) != 2 {
		t.Fatalf("focused ranked table anchors = %d, want 2: %+v", len(got), got)
	}
	if got[1].ChunkID != "rank-two" || got[1].RetrievalAnchorRank != 2 || got[1].Content != rankTwoContent {
		t.Fatalf("unfocusable ranked table = %+v, want its original content and identity", got[1])
	}
}

func TestFocusRAGTableEvidenceRowsKeepsUnrankedTableNeighborsIntact(t *testing.T) {
	firstRows := strings.Repeat(`<tr><td>SOP</td><td>generic matching context</td></tr>`, 20)
	secondRows := strings.Repeat(`<tr><td>SOP</td><td>needle matched criterion</td></tr>`, 20)
	if utf8RuneCount(firstRows)+utf8RuneCount(secondRows) < ragFocusedTableMinRunes {
		t.Fatal("fixture must enter the table focus path")
	}
	group := []knowledge.RAGChunkHit{
		{
			ChunkID:             "direct-anchor",
			ChunkType:           "text",
			Content:             "direct anchor evidence",
			RetrievalAnchorRank: 1,
			PageNumber:          8,
			ImageFileID:         "direct-image",
		},
		{
			ChunkID:         "page-nine-generic-table",
			ChunkType:       "table",
			Content:         firstRows,
			PageNumber:      9,
			ImageFileID:     "page-nine-image",
			PageImageFileID: "page-nine-page-image",
			ObjectID:        "page-nine-object",
			BBox:            []float64{1, 2, 3, 4},
			VisualRefs: []knowledge.RAGImageRef{{
				ChunkID: "page-nine-visual", ImageFileID: "page-nine-visual-image",
			}},
		},
		{
			ChunkID:         "page-ten-criterion-table",
			ChunkType:       "table",
			Content:         secondRows,
			PageNumber:      10,
			ImageFileID:     "page-ten-image",
			PageImageFileID: "page-ten-page-image",
			ObjectID:        "page-ten-object",
			BBox:            []float64{5, 6, 7, 8},
			VisualRefs: []knowledge.RAGImageRef{{
				ChunkID: "page-ten-visual", ImageFileID: "page-ten-visual-image",
			}},
		},
	}

	got := focusRAGTableEvidenceRows(group, []string{"SOP", "needle", "criterion"})
	if !reflect.DeepEqual(got, group) {
		t.Fatalf("all-rank-zero table neighbors must keep their own content and provenance:\n got %#v\nwant %#v", got, group)
	}
}

func TestFocusRAGTableEvidenceRowsKeepsNonFirstRowspanInLogicalColumnOrder(t *testing.T) {
	table := `<table>` +
		`<tr><td>other-area</td><td rowspan="2">shared-rule</td><td>` + strings.Repeat("filler ", 160) + `</td></tr>` +
		`<tr><td>target-area</td><td>final-condition</td></tr>` +
		`</table>`
	group := []knowledge.RAGChunkHit{{
		ChunkID: "non-first-rowspan-table",
		Content: table,
	}}

	got := focusRAGTableEvidenceRows(group, []string{"target-area shared-rule"})
	if len(got) != 1 {
		t.Fatalf("focused group = %+v, want one carrier", got)
	}
	focused := got[0].Content
	for _, required := range []string{"target-area", "shared-rule", "final-condition"} {
		if !strings.Contains(focused, required) {
			t.Fatalf("focused table = %q, want %q", focused, required)
		}
	}
	if strings.Contains(focused, "filler") {
		t.Fatalf("focused table retained the non-matching row: %q", focused)
	}
	targetIndex := strings.Index(focused, "target-area")
	sharedIndex := strings.Index(focused, "shared-rule")
	conditionIndex := strings.Index(focused, "final-condition")
	if !(targetIndex < sharedIndex && sharedIndex < conditionIndex) {
		t.Fatalf("focused table logical order = %q, want target-area then shared-rule then final-condition", focused)
	}
}

func TestSearchRAGChunksMakesMultiSourceRanksGloballyComparableAndKeepsBestDedupedRank(t *testing.T) {
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		ragAnchorPriorityTestColumns(),
		ragAnchorPriorityTestRows("fulltext",
			ragAnchorPriorityTestChunk{fileID: "source-one", chunkID: "source-one-first", content: "term first"},
			ragAnchorPriorityTestChunk{fileID: "shared", chunkID: "shared", content: "term shared", chunkIndex: ragAnchorPriorityInt(7)},
		),
		ragAnchorPriorityTestRows("vector_l2"),
		ragAnchorPriorityTestColumns(),
		ragAnchorPriorityTestRows("fulltext",
			ragAnchorPriorityTestChunk{fileID: "shared", chunkID: "shared", content: "term shared", chunkIndex: ragAnchorPriorityInt(7)},
			ragAnchorPriorityTestChunk{fileID: "source-two", chunkID: "source-two-second", content: "term second"},
		),
		ragAnchorPriorityTestRows("vector_l2"),
	}}
	svc := NewSearchRAGChunks(Deps{SQLExecutor: exec, Embedder: &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}}})
	resp, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "workspace",
			RAGSources: []knowledge.RAGSource{
				{DBName: "source_one", VectorTable: "rag_one", EmbeddingModel: "bge-m3"},
				{DBName: "source_two", VectorTable: "rag_two", EmbeddingModel: "bge-m3"},
			},
		},
		Keywards: []string{"term"},
		MaxHits:  10,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(resp.Chunks) != 3 {
		t.Fatalf("deduped chunk count = %d, want 3", len(resp.Chunks))
	}
	ranks := map[string]int{}
	for _, chunk := range resp.Chunks {
		ranks[chunk.ChunkID] = chunk.RetrievalAnchorRank
	}
	if got := ranks["source-one-first"]; got != 1 {
		t.Fatalf("source one first rank = %d, want 1", got)
	}
	if got := ranks["shared"]; got != 2 {
		t.Fatalf("deduped shared rank = %d, want source-one global rank 2", got)
	}
	if got := ranks["source-two-second"]; got != 4 {
		t.Fatalf("source two second rank = %d, want 4", got)
	}
	if got, want := exec.gotDBNames, []string{"source_one", "source_one", "source_one", "source_two", "source_two", "source_two"}; len(got) != len(want) {
		t.Fatalf("source query count = %d, want %d: %v", len(got), len(want), got)
	} else {
		for index := range want {
			if got[index] != want[index] {
				t.Fatalf("source query %d db = %q, want %q", index, got[index], want[index])
			}
		}
	}
}

func TestSearchRAGChunksMultiModelSemanticModelIDReachesProjectedVisualRefs(t *testing.T) {
	exec := &stubRAGExecutor{results: []*knowledge.SQLExecutionResult{
		ragAnchorPriorityTestColumns(),
		ragSharedSemanticModelVisualRows("fulltext"),
		ragAnchorPriorityTestRows("vector_l2"),
		ragAnchorPriorityTestColumns(),
		ragSharedSemanticModelVisualRows("fulltext"),
		ragAnchorPriorityTestRows("vector_l2"),
	}}
	svc := NewSearchRAGChunks(Deps{
		SQLExecutor: exec,
		Embedder:    &stubRAGEmbedder{embeddings: [][]float64{{0.1, 0.2}}},
	})

	resp, err := svc.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "workspace",
			RAGSources: []knowledge.RAGSource{
				{
					SemanticModelID: 101,
					DBName:          "model_101",
					VectorTable:     "rag_101",
					EmbeddingModel:  "bge-m3",
				},
				{
					SemanticModelID: 202,
					DBName:          "model_202",
					VectorTable:     "rag_202",
					EmbeddingModel:  "bge-m3",
				},
			},
		},
		Keywards: []string{"model"},
		MaxHits:  10,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	wantModelIDs := []int64{101, 202}
	t.Run("search_keeps_same_ID_evidence_per_owner", func(t *testing.T) {
		if len(resp.Chunks) != len(wantModelIDs) {
			t.Errorf(
				"chunks = %+v, want two owner-distinct hits despite identical chunk/file/object/image/page-image IDs",
				resp.Chunks,
			)
		}
		for index, wantModelID := range wantModelIDs {
			index, wantModelID := index, wantModelID
			t.Run(fmt.Sprintf("hit_owner_%d", wantModelID), func(t *testing.T) {
				if index >= len(resp.Chunks) {
					t.Fatalf("missing RAG search hit for semantic model %d", wantModelID)
				}
				chunk := resp.Chunks[index]
				if chunk.ChunkID != "shared_chunk" ||
					chunk.FileID != "shared_file" ||
					chunk.ObjectID != "shared_object" ||
					chunk.ImageFileID != "shared_image" ||
					chunk.PageImageFileID != "shared_page_image" {
					t.Errorf("RAG search hit %d identifiers = %+v, want the shared collision fixture", index, chunk)
				}
				requireServiceSemanticModelID(t, fmt.Sprintf("RAG search hit %d", index), chunk, wantModelID)
			})
			t.Run(fmt.Sprintf("nested_owner_%d", wantModelID), func(t *testing.T) {
				if index >= len(resp.Chunks) {
					t.Fatalf("missing RAG search hit for semantic model %d", wantModelID)
				}
				chunk := resp.Chunks[index]
				if len(chunk.VisualRefs) != 1 {
					t.Fatalf("RAG search hit %d visual_refs = %+v, want one nested RAG image ref", index, chunk.VisualRefs)
				}
				ref := chunk.VisualRefs[0]
				if ref.ChunkID != "shared_chunk" ||
					ref.ObjectID != "shared_object" ||
					ref.ImageFileID != "shared_image" ||
					ref.PageImageFileID != "shared_page_image" {
					t.Errorf("nested RAG image ref %d identifiers = %+v, want the shared collision fixture", index, ref)
				}
				requireServiceSemanticModelID(t, fmt.Sprintf("nested RAG image ref %d", index), ref, wantModelID)
			})
		}
	})

	rc := knowledge.NewRunContext()
	rc.RecordRAGChunksArtifact("rag_multi_model", *resp)
	var selected []knowledge.FinalAnswerSource
	if err := json.Unmarshal([]byte(`[
		{"type":"rag_chunk","chunk_id":"shared_chunk","semantic_model_id":101},
		{"type":"rag_chunk","chunk_id":"shared_chunk","semantic_model_id":202}
	]`), &selected); err != nil {
		t.Fatalf("decode owner-qualified RAG selection: %v", err)
	}
	resolved, err := rc.ResolveFinalAnswerSources(selected)
	if err != nil {
		t.Errorf("ResolveFinalAnswerSources() error = %v", err)
	}
	assertOwnedRAGFinalSources(t, "RunContext", resolved, wantModelIDs)

	projected := knowledge.ProjectFinalAnswerSourceRefs(resolved)
	assertOwnedRAGFinalSources(t, "projected", projected, wantModelIDs)
}

func assertOwnedRAGFinalSources(t *testing.T, label string, sources []knowledge.FinalAnswerSource, wantModelIDs []int64) {
	t.Helper()

	if len(sources) != len(wantModelIDs) {
		t.Errorf(
			"%s sources = %+v, want two owner-distinct sources despite identical evidence IDs",
			label,
			sources,
		)
	}
	for index, wantModelID := range wantModelIDs {
		index, wantModelID := index, wantModelID
		t.Run(fmt.Sprintf("%s_source_owner_%d", label, wantModelID), func(t *testing.T) {
			if index >= len(sources) {
				t.Fatalf("%s sources missing semantic model %d", label, wantModelID)
			}
			source := sources[index]
			if source.FileID != "shared_file" ||
				source.ObjectID != "shared_object" ||
				source.ImageFileID != "shared_image" ||
				source.PageImageFileID != "shared_page_image" {
				t.Errorf("%s source %d identifiers = %+v, want the shared collision fixture", label, index, source)
			}
			requireServiceSemanticModelID(t, fmt.Sprintf("%s RAG source %d", label, index), source, wantModelID)
		})
		t.Run(fmt.Sprintf("%s_nested_owner_%d", label, wantModelID), func(t *testing.T) {
			if index >= len(sources) {
				t.Fatalf("%s sources missing semantic model %d", label, wantModelID)
			}
			source := sources[index]
			if len(source.VisualRefs) != 1 {
				t.Fatalf("%s source %d visual_refs = %+v, want one nested ID-level ref", label, index, source.VisualRefs)
			}
			ref := source.VisualRefs[0]
			if ref.ChunkID != "shared_chunk" ||
				ref.ObjectID != "shared_object" ||
				ref.ImageFileID != "shared_image" ||
				ref.PageImageFileID != "shared_page_image" {
				t.Errorf("%s nested visual ref %d identifiers = %+v, want the shared collision fixture", label, index, ref)
			}
			requireServiceSemanticModelID(t, fmt.Sprintf("%s nested visual ref %d", label, index), ref, wantModelID)
		})
	}
}

func ragSharedSemanticModelVisualRows(route string) *knowledge.SQLExecutionResult {
	meta := `{"chunk_id":"shared_chunk","source_file_name":"shared.pdf","image_file_id":"shared_image","page_image_file_id":"shared_page_image","object_id":"shared_object","object_kind":"figure","page_number":1,"bbox":[1,2,3,4]}`
	return &knowledge.SQLExecutionResult{
		Columns: []string{
			"route",
			"level",
			"content",
			"meta",
			"file_id",
			"index_version",
			"chunk_index",
			"parent_index",
			"chunk_start",
			"chunk_end",
			"score",
		},
		Rows: [][]any{{
			route,
			"chunk",
			"shared visual evidence",
			meta,
			"shared_file",
			"v1",
			1,
			nil,
			nil,
			nil,
			1.0,
		}},
	}
}
