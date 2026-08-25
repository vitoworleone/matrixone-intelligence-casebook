package tests

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"
	knowledge "github.com/matrixflow/moi-core/agent-tools/knowledge"
	"github.com/matrixflow/moi-core/tests/framework"
	"github.com/stretchr/testify/require"
)

const issue13719TargetCriterion = "Reject if >50% of the surface"

func TestIssue13719RAGAnchorPriority(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	framework.RunMOITests(t, func(env *framework.TestEnv) {
		env.RequireSharedWorkspace(t)
		chunks := issue13719TraceChunks()
		var targetContent string
		for _, chunk := range chunks {
			if chunk.ChunkID == "anchor-page-10-pdf" {
				targetContent = chunk.Content
				break
			}
		}
		require.NotEmpty(t, targetContent)
		require.NotContains(t, string([]rune(targetContent)[:120]), issue13719TargetCriterion, "fixture must place the criterion beyond the legacy prefix preview")
		search := &issue13719FixedSearchRAGChunks{response: &knowledge.SearchRAGChunksResponse{
			Keywards:       []string{"通用规范", "划伤"},
			Routes:         []string{"fulltext", "vector_l2"},
			Chunks:         chunks,
			RowCount:       46,
			ExpandedGroups: 46,
		}}
		tool := knowledge.NewSearchRAGChunksTool(knowledge.Options{
			Registry: &knowledge.Registry{SearchRAGChunks: search},
		})
		invocationTool, ok := tool.(agentruntimev2.InvocationTool)
		require.True(t, ok, "search_rag_chunks must support runtime invocation")

		result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "issue-13719"}, json.RawMessage(`{"keywords":["通用规范","划伤"],"max_hits":10}`))
		require.NoError(t, err)
		require.NotNil(t, result.ModelView)

		modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
		require.NoError(t, err)
		require.Greater(t, len(modelContent), 10_000, "default policy must use serialized-byte admission")
		require.LessOrEqual(t, len(modelContent), 12_000)

		var payload struct {
			ItemCount        int `json:"item_count"`
			EmittedItemCount int `json:"emitted_item_count"`
			OmittedItemCount int `json:"omitted_item_count"`
			Items            []struct {
				ItemID         string `json:"item_id"`
				ContentPreview string `json:"content_preview"`
				Refs           struct {
					ChunkID string `json:"chunk_id"`
				} `json:"refs"`
			} `json:"items"`
		}
		require.NoError(t, json.Unmarshal([]byte(modelContent), &payload))
		require.Equal(t, payload.EmittedItemCount, len(payload.Items))
		require.Equal(t, payload.ItemCount, payload.EmittedItemCount+payload.OmittedItemCount)
		require.Less(t, payload.EmittedItemCount, 20)

		seenTargets := map[string]bool{}
		previews := map[string]string{}
		for _, item := range payload.Items {
			require.NotEmpty(t, item.ItemID)
			require.Equal(t, item.ItemID, item.Refs.ChunkID, "each emitted item must remain a complete JSON item")
			require.Truef(t, strings.HasPrefix(item.ItemID, "anchor-"), "neighbor %q entered before an omitted anchor", item.ItemID)
			seenTargets[item.ItemID] = true
			previews[item.ItemID] = item.ContentPreview
		}
		require.True(t, seenTargets["anchor-page-10-pdf"], "rank 14 target anchor must be model-visible")
		require.True(t, seenTargets["anchor-page-10-doc"], "rank 15 target anchor must be model-visible")
		require.Contains(t, previews["anchor-page-10-pdf"], "Scratch / Damage / Deform")
		require.Contains(t, previews["anchor-page-10-pdf"], issue13719TargetCriterion)
		require.NotContains(t, result.Content, "retrieval_anchor_rank")
		require.Contains(t, result.Content, "anchor-page-10-pdf")
		require.Contains(t, result.Content, "anchor-page-10-doc")
	})
}

func TestIssue13719RAGTablePreviewEvidenceIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	framework.RunMOITests(t, func(env *framework.TestEnv) {
		env.RequireSharedWorkspace(t)
		table := `<table><tr><td rowspan="2">SOP area</td><td>Solder Extrusion</td><td>Reject if any bridge</td></tr><tr><td>Scratch / Damage / Deform</td><td>ordinary description</td><td>needle</td><td>matched evidence</td><td>final criterion</td></tr></table>`
		search := &issue13719FixedSearchRAGChunks{response: &knowledge.SearchRAGChunksResponse{
			Keywards: []string{"needle matched"},
			Chunks: []knowledge.RAGChunkHit{{
				ChunkID: "rowspan-table", FileID: "file_1", Content: table, ChunkType: "table",
			}},
			RowCount: 1,
		}}
		tool := knowledge.NewSearchRAGChunksTool(knowledge.Options{
			Registry: &knowledge.Registry{SearchRAGChunks: search},
		})
		invocationTool, ok := tool.(agentruntimev2.InvocationTool)
		require.True(t, ok, "search_rag_chunks must support runtime invocation")

		result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "issue-13719-table-evidence"}, json.RawMessage(`{"keywords":["needle matched"],"max_hits":10}`))
		require.NoError(t, err)
		require.NotNil(t, result.ModelView)
		modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
		require.NoError(t, err)

		var payload struct {
			Items []struct {
				ContentPreview string `json:"content_preview"`
			} `json:"items"`
		}
		require.NoError(t, json.Unmarshal([]byte(modelContent), &payload))
		require.Len(t, payload.Items, 1)
		for _, required := range []string{"SOP area", "needle", "matched evidence", "final criterion"} {
			require.Contains(t, payload.Items[0].ContentPreview, required)
		}
	})
}

type issue13719FixedSearchRAGChunks struct {
	response *knowledge.SearchRAGChunksResponse
}

func (s *issue13719FixedSearchRAGChunks) Execute(_ context.Context, req knowledge.SearchRAGChunksRequest) (*knowledge.SearchRAGChunksResponse, error) {
	response := *s.response
	response.Keywards = append([]string(nil), req.Keywards...)
	response.Chunks = append([]knowledge.RAGChunkHit(nil), s.response.Chunks...)
	return &response, nil
}

func issue13719TraceChunks() []knowledge.RAGChunkHit {
	chunks := make([]knowledge.RAGChunkHit, 46)
	for index := range chunks {
		id := "neighbor-" + strconv.Itoa(index)
		chunks[index] = knowledge.RAGChunkHit{
			ChunkID:         id,
			Content:         strings.Repeat("通用规范划伤证据", 20) + id,
			FileID:          "file-" + strconv.Itoa(index),
			MarkdownFileID:  "markdown-" + strconv.Itoa(index),
			FileName:        "customer-spec.pdf",
			IndexVersion:    "v1",
			Level:           "chunk",
			PageNumber:      1,
			EvidenceGroupID: "trace-group-" + strconv.Itoa(index),
			Score:           1 - float64(index)/100,
			Routes:          []string{"fulltext", "vector_l2"},
			SourceTags:      []string{"pdf", "trace"},
		}
	}
	anchorPositions := map[int]int{14: 21, 15: 23}
	nextPosition := 24
	for rank := 1; rank <= 20; rank++ {
		position, ok := anchorPositions[rank]
		if !ok {
			position = nextPosition
			nextPosition++
		}
		id := "anchor-" + strconv.Itoa(rank)
		if rank == 14 {
			id = "anchor-page-10-pdf"
		}
		if rank == 15 {
			id = "anchor-page-10-doc"
		}
		content := strings.Repeat("通用规范划伤证据", 20) + id
		if rank == 14 {
			content = `<table><tr><td>通用规范</td><td>Scratch / Damage / Deform</td><td>` +
				strings.Repeat("long scratch definition ", 18) + `</td><td>` + issue13719TargetCriterion + `</td></tr></table>` +
				strings.Repeat("通用规范划伤证据", 20)
		}
		chunks[position] = knowledge.RAGChunkHit{
			ChunkID:             id,
			Content:             content,
			FileID:              "anchor-file-" + strconv.Itoa(rank),
			MarkdownFileID:      "anchor-markdown-" + strconv.Itoa(rank),
			FileName:            "target-page-10.pdf",
			IndexVersion:        "v1",
			Level:               "chunk",
			PageNumber:          10,
			EvidenceGroupID:     "trace-group-" + strconv.Itoa(position),
			Score:               1 - float64(rank)/100,
			Routes:              []string{"fulltext", "vector_l2"},
			SourceTags:          []string{"pdf", "trace"},
			RetrievalAnchorRank: rank,
		}
	}
	return chunks
}
