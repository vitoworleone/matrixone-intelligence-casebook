package tests

import (
	"context"
	"strings"
	"testing"

	"github.com/matrixflow/moi-core/agent-tools/knowledge"
	knowledgeservice "github.com/matrixflow/moi-core/agent-tools/knowledge/service"
	"github.com/stretchr/testify/require"
)

func TestKnowledgeLegacyIndexConstraintScopesRAGAndVisualSearch(t *testing.T) {
	scope := knowledge.WorkspaceScope{
		WorkspaceID: "ws_legacy_scope",
		DBName:      "tenant_db",
		RAGSources: []knowledge.RAGSource{{
			DBName:                  "tenant_db",
			VectorTable:             "kb_text_vec",
			EmbeddingModel:          "bge-m3",
			ImageVectorTable:        "kb_image_vec",
			ImageEmbeddingModel:     "clip",
			ImageEmbeddingDimension: 3,
			ImagePreprocessVersion:  "v1",
			ImageDistanceMetric:     "cosine",
			FileIDs:                 []string{"file_current", "file_zero", "file_null"},
			CurrentIndexVersionByFileID: map[string]int64{
				"file_current": 7,
			},
			IndexVersionConstraintByFileID: map[string]knowledge.RAGIndexVersionConstraint{
				"file_zero": {Kind: knowledge.RAGIndexVersionConstraintValue, Value: 0},
				"file_null": {Kind: knowledge.RAGIndexVersionConstraintNull},
			},
		}},
	}

	ragSQL := &knowledgeLegacyIndexScopeSQLExecutor{}
	rag := knowledgeservice.NewSearchRAGChunks(knowledgeservice.Deps{
		SQLExecutor: ragSQL,
		Embedder:    knowledgeLegacyIndexScopeEmbedder{},
	})
	ragResp, err := rag.Execute(context.Background(), knowledge.SearchRAGChunksRequest{
		Scope:    scope,
		Keywards: []string{"legacy scoped recall"},
		MaxHits:  12,
	})
	require.NoError(t, err)
	require.Contains(t, ragResp.FulltextSQL, "(file_id = 'file_current' AND index_version = '7')")
	require.Contains(t, ragResp.FulltextSQL, "(file_id = 'file_zero' AND index_version = '0')")
	require.Contains(t, ragResp.FulltextSQL, "(file_id = 'file_null' AND index_version IS NULL)")
	require.NotContains(t, ragResp.FulltextSQL, "file_id IN ('file_current','file_zero','file_null')")
	require.NotContains(t, ragResp.FulltextSQL, "file_id IN ('file_current', 'file_zero', 'file_null')")
	require.Contains(t, ragResp.VectorSQL, "(file_id = 'file_current' AND index_version = '7')")
	require.Contains(t, ragResp.VectorSQL, "(file_id = 'file_zero' AND index_version = '0')")
	require.Contains(t, ragResp.VectorSQL, "(file_id = 'file_null' AND index_version IS NULL)")

	visualSQL := &knowledgeLegacyIndexScopeSQLExecutor{}
	visual := knowledgeservice.NewSearchVisualImage(knowledgeservice.Deps{
		SQLExecutor:         visualSQL,
		VisualSearchBackend: knowledgeLegacyIndexScopeVisualBackend{},
	})
	_, err = visual.Execute(context.Background(), knowledge.SearchVisualImageRequest{
		Scope:     scope,
		QueryText: "legacy scoped visual",
		TopK:      3,
	})
	require.NoError(t, err)
	require.Len(t, visualSQL.sqls, 1)
	require.Contains(t, visualSQL.sqls[0], "(file_id = 'file_current' AND index_version = '7')")
	require.Contains(t, visualSQL.sqls[0], "(file_id = 'file_zero' AND index_version = '0')")
	require.Contains(t, visualSQL.sqls[0], "(file_id = 'file_null' AND index_version IS NULL)")
	require.NotContains(t, visualSQL.sqls[0], "file_id IN ('file_current','file_zero','file_null')")
	require.NotContains(t, visualSQL.sqls[0], "file_id IN ('file_current', 'file_zero', 'file_null')")
}

type knowledgeLegacyIndexScopeSQLExecutor struct {
	sqls []string
}

func (e *knowledgeLegacyIndexScopeSQLExecutor) ExecuteSQL(_ context.Context, _ string, sqlText string) (*knowledge.SQLExecutionResult, error) {
	e.sqls = append(e.sqls, sqlText)
	switch {
	case strings.HasPrefix(sqlText, "SHOW COLUMNS"):
		return &knowledge.SQLExecutionResult{
			Columns: []string{"Field"},
			Rows: [][]any{
				{"file_id"},
				{"index_version"},
				{"level"},
				{"content"},
				{"meta"},
				{"embedding"},
				{"chunk_index"},
				{"disabled"},
			},
		}, nil
	case strings.Contains(sqlText, "rag_fulltext_candidates"), strings.Contains(sqlText, "rag_vector_candidates"):
		return &knowledge.SQLExecutionResult{
			Columns: []string{"route", "level", "content", "meta", "file_id", "index_version", "chunk_index", "score"},
		}, nil
	default:
		return &knowledge.SQLExecutionResult{
			Columns: []string{"id", "content", "meta", "file_id", "page_number", "score"},
		}, nil
	}
}

type knowledgeLegacyIndexScopeEmbedder struct{}

func (knowledgeLegacyIndexScopeEmbedder) CreateEmbedding(_ context.Context, _, _ string, texts []string) ([][]float64, error) {
	out := make([][]float64, 0, len(texts))
	for range texts {
		out = append(out, []float64{0.1, 0.2, 0.3})
	}
	return out, nil
}

type knowledgeLegacyIndexScopeVisualBackend struct{}

func (knowledgeLegacyIndexScopeVisualBackend) ReadImageFile(context.Context, knowledge.WorkspaceScope, string) ([]byte, string, error) {
	return nil, "", nil
}

func (knowledgeLegacyIndexScopeVisualBackend) CreateImageEmbedding(context.Context, string, knowledgeservice.VisualImageEmbeddingRequest) ([]float64, map[string]any, error) {
	return nil, nil, nil
}

func (knowledgeLegacyIndexScopeVisualBackend) ResolveVisualScopeFileIDs(context.Context, knowledge.WorkspaceScope) ([]string, bool, error) {
	return []string{"file_current", "file_zero", "file_null"}, true, nil
}
