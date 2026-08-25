package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"
)

type recordingSearchRAGChunks struct {
	searchRequest SearchRAGChunksRequest
	response      *SearchRAGChunksResponse
}

func TestRAGChunkHitKeepsRetrievalAnchorRankInternal(t *testing.T) {
	typeOfHit := reflect.TypeOf(RAGChunkHit{})
	field, ok := typeOfHit.FieldByName("RetrievalAnchorRank")
	if !ok {
		t.Fatal("RAGChunkHit must retain an internal RetrievalAnchorRank")
	}
	if got := field.Tag.Get("json"); got != "-" {
		t.Fatalf("RetrievalAnchorRank json tag = %q, want -", got)
	}

	hit := RAGChunkHit{ChunkID: "chunk_1", Content: "evidence"}
	value := reflect.ValueOf(&hit).Elem().FieldByIndex(field.Index)
	value.SetInt(7)
	encoded, err := json.Marshal(hit)
	if err != nil {
		t.Fatalf("marshal RAGChunkHit: %v", err)
	}
	if strings.Contains(string(encoded), "retrieval_anchor_rank") {
		t.Fatalf("public JSON leaked internal rank: %s", encoded)
	}
}

func (r *recordingSearchRAGChunks) Execute(_ context.Context, req SearchRAGChunksRequest) (*SearchRAGChunksResponse, error) {
	r.searchRequest = req
	if r.response != nil {
		return r.response, nil
	}
	return &SearchRAGChunksResponse{}, nil
}

type recordingQuerySQL struct {
	request  QuerySQLRequest
	response *QuerySQLResponse
}

func (r *recordingQuerySQL) Execute(_ context.Context, req QuerySQLRequest) (*QuerySQLResponse, error) {
	r.request = req
	if r.response != nil {
		return r.response, nil
	}
	return &QuerySQLResponse{}, nil
}

type recordingSearchVisualImage struct {
	request  SearchVisualImageRequest
	response *SearchVisualImageResponse
}

func (r *recordingSearchVisualImage) Execute(_ context.Context, req SearchVisualImageRequest) (*SearchVisualImageResponse, error) {
	r.request = req
	if r.response != nil {
		return r.response, nil
	}
	return &SearchVisualImageResponse{}, nil
}

func TestSearchRAGChunksAcceptsKeywords(t *testing.T) {
	search := &recordingSearchRAGChunks{}
	tool := NewSearchRAGChunksTool(Options{
		Registry: &Registry{SearchRAGChunks: search},
		Scope:    WorkspaceScope{WorkspaceID: "ws_1", UserID: "user_1"},
	})

	invocationTool, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("tool does not implement InvocationTool")
	}
	_, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{}, json.RawMessage(`{"keywords":[" 后端研发 ","后端研发","简历"],"max_hits":3}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if got, want := search.searchRequest.Keywards, []string{"后端研发", "简历"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("keywords forwarded as %v, want %v", got, want)
	}
	if search.searchRequest.MaxHits != 3 {
		t.Fatalf("MaxHits = %d, want 3", search.searchRequest.MaxHits)
	}
	if search.searchRequest.Scope.WorkspaceID != "ws_1" || search.searchRequest.Scope.UserID != "user_1" {
		t.Fatalf("scope = %+v, want workspace/user scope", search.searchRequest.Scope)
	}
}

func TestSearchRAGChunksReturnsCompactModelContent(t *testing.T) {
	fullContent := strings.Repeat("abcdefghij", 40) + "SECRET_TAIL"
	search := &recordingSearchRAGChunks{
		response: &SearchRAGChunksResponse{
			Keywards:       []string{"api", "hardware"},
			Routes:         []string{"fulltext", "vector"},
			RowCount:       1,
			ExpandedGroups: 1,
			EmbeddingModel: "embedder",
			Chunks: []RAGChunkHit{{
				Content:         fullContent,
				FileID:          "file_1",
				MarkdownFileID:  "md_1",
				FileName:        "manual.pdf",
				PageNumber:      7,
				VolumeID:        "secret-volume",
				IndexVersion:    "secret-index-version",
				ChunkID:         "chunk_1",
				Score:           12.5,
				Routes:          []string{"fulltext"},
				SourceTags:      []string{"tag_1"},
				EvidenceGroupID: "group_1",
				BBox:            []float64{1, 2, 3, 4},
			}},
		},
	}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("tool does not implement InvocationTool")
	}

	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "call_rag"}, json.RawMessage(`{"keywords":["api","hardware"],"max_hits":10}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.LLMContent != "" {
		t.Fatalf("LLMContent = %q, want runtime-rendered ModelView", result.LLMContent)
	}
	if result.ModelView == nil {
		t.Fatal("ModelView missing")
	}
	if !strings.Contains(result.Content, "SECRET_TAIL") || !strings.Contains(result.Content, "secret-index-version") {
		t.Fatalf("raw Content lost full result: %s", result.Content)
	}
	modelContent := agentruntimev2.ToolResultModelVisibleContent(*result)
	if strings.Contains(modelContent, "SECRET_TAIL") || strings.Contains(modelContent, "secret-index-version") ||
		strings.Contains(modelContent, "secret-volume") {
		t.Fatalf("model content leaked raw-only fields or full content: %s", modelContent)
	}

	var model map[string]any
	if err := json.Unmarshal([]byte(modelContent), &model); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if model["kind"] != "rag_chunks" || model["model_view"] != "compact" {
		t.Fatalf("model view identity = %v/%v", model["kind"], model["model_view"])
	}
	if model["artifact_id"] != "rag_chunks_call_rag" || model["full_result_artifact_id"] != "rag_chunks_call_rag" {
		t.Fatalf("artifact ids = %v/%v", model["artifact_id"], model["full_result_artifact_id"])
	}
	budget, ok := model["budget"].(map[string]any)
	if !ok ||
		int(budget["max_items"].(float64)) != ragChunksModelViewMaxItems ||
		int(budget["max_preview_chars"].(float64)) != ragChunksModelViewPreviewRunes ||
		int(budget["max_total_preview_chars"].(float64)) != ragChunksModelViewTotalPreviewRunes {
		t.Fatalf("budget = %#v", model["budget"])
	}
	summary, ok := model["summary"].(map[string]any)
	if !ok || int(summary["row_count"].(float64)) != 1 || int(summary["chunk_count"].(float64)) != 1 ||
		int(summary["expanded_groups"].(float64)) != 1 || summary["embedding_model"] != "embedder" {
		t.Fatalf("summary = %#v", model["summary"])
	}
	items, ok := model["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v", model["items"])
	}
	chunk, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("chunk = %#v", items[0])
	}
	if _, ok := chunk["content"]; ok {
		t.Fatalf("model chunk exposes full content: %#v", chunk)
	}
	refs, ok := chunk["refs"].(map[string]any)
	if !ok || refs["chunk_id"] != "chunk_1" || refs["file_id"] != "file_1" || refs["markdown_file_id"] != "md_1" {
		t.Fatalf("model chunk refs = %#v", chunk["refs"])
	}
	if chunk["item_id"] != "chunk_1" || chunk["source_id"] != "file_1" || chunk["source_name"] != "manual.pdf" ||
		chunk["group_id"] != "group_1" {
		t.Fatalf("model chunk identifiers = %#v", chunk)
	}
	if got := int(chunk["content_chars"].(float64)); got != len([]rune(fullContent)) {
		t.Fatalf("content_chars = %d, want %d", got, len([]rune(fullContent)))
	}
	if got := int(chunk["content_preview_chars"].(float64)); got != ragChunksModelViewPreviewRunes {
		t.Fatalf("content_preview_chars = %d, want %d", got, ragChunksModelViewPreviewRunes)
	}
	if chunk["content_preview_truncated"] != true {
		t.Fatalf("content_preview_truncated = %#v", chunk["content_preview_truncated"])
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].ArtifactID != "rag_chunks_call_rag" {
		t.Fatalf("artifacts = %+v", result.Artifacts)
	}
}

func TestSearchRAGChunksModelViewCentersLongTextPreviewOnLiteralKeyword(t *testing.T) {
	content := strings.Repeat("前置无关内容", 30) + "SOP划伤规范：超出面积二分之一应拒收" + strings.Repeat("后置无关内容", 30)
	result := &SearchRAGChunksResponse{
		Keywards: []string{"SOP", "划伤"},
		Chunks: []RAGChunkHit{{
			ChunkID: "chunk_1",
			FileID:  "file_1",
			Content: content,
		}},
	}

	modelContent := searchRAGChunksToolModelView(result).ModelContent()
	var payload struct {
		ContentChars            int  `json:"content_chars"`
		ContentPreviewTruncated bool `json:"content_preview_truncated"`
		Items                   []struct {
			ContentPreview          string `json:"content_preview"`
			ContentChars            int    `json:"content_chars"`
			ContentPreviewTruncated bool   `json:"content_preview_truncated"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("items = %+v", payload.Items)
	}
	preview := payload.Items[0].ContentPreview
	if !strings.Contains(preview, "SOP划伤规范") {
		t.Fatalf("preview does not contain keyword-centered evidence: %q", preview)
	}
	if !strings.HasPrefix(preview, "...") || !strings.HasSuffix(preview, "...") {
		t.Fatalf("preview omission markers missing: %q", preview)
	}
	if len([]rune(preview)) > ragChunksModelViewPreviewRunes {
		t.Fatalf("preview runes = %d, want <= %d", len([]rune(preview)), ragChunksModelViewPreviewRunes)
	}
	if payload.Items[0].ContentChars != len([]rune(content)) || !payload.Items[0].ContentPreviewTruncated {
		t.Fatalf("item preview metadata = %+v, want original content chars %d and truncated", payload.Items[0], len([]rune(content)))
	}
	if payload.ContentChars != len([]rune(content)) || !payload.ContentPreviewTruncated {
		t.Fatalf("view preview metadata = chars %d truncated %v", payload.ContentChars, payload.ContentPreviewTruncated)
	}
	if result.Chunks[0].Content != content {
		t.Fatal("query-aware preview changed the raw RAG chunk")
	}
}

func TestSearchRAGChunksModelViewKeepsPrefixWhenNoLiteralKeywordMatches(t *testing.T) {
	content := strings.Repeat("向量语义命中", 30) + "末尾答案"
	view := searchRAGChunksToolModelView(&SearchRAGChunksResponse{
		Keywards: []string{"paraphrase"},
		Chunks:   []RAGChunkHit{{ChunkID: "chunk_1", Content: content}},
	})
	modelContent := view.ModelContent()
	if strings.Contains(modelContent, "末尾答案") {
		t.Fatalf("no-literal-match preview should preserve prefix behavior: %s", modelContent)
	}
	if !strings.Contains(modelContent, strings.Repeat("向量语义命中", 10)) {
		t.Fatalf("no-literal-match preview lost content prefix: %s", modelContent)
	}
}

func TestSearchRAGChunksModelViewPrioritizesEvidenceGroupBreadth(t *testing.T) {
	chunks := []RAGChunkHit{
		{ChunkID: "a1", Content: "content a1", FileID: "file_a", FileName: "a.pdf", EvidenceGroupID: "group_a"},
		{ChunkID: "a2", Content: "content a2", FileID: "file_a", FileName: "a.pdf", EvidenceGroupID: "group_a"},
		{ChunkID: "a3", Content: "content a3", FileID: "file_a", FileName: "a.pdf", EvidenceGroupID: "group_a"},
		{ChunkID: "b1", Content: "content b1", FileID: "file_b", FileName: "b.pdf", EvidenceGroupID: "group_b"},
		{ChunkID: "b2", Content: "content b2", FileID: "file_b", FileName: "b.pdf", EvidenceGroupID: "group_b"},
		{ChunkID: "c1", Content: "content c1", FileID: "file_c", FileName: "c.pdf", EvidenceGroupID: "group_c"},
	}
	search := &recordingSearchRAGChunks{
		response: &SearchRAGChunksResponse{
			RowCount:       len(chunks),
			ExpandedGroups: 3,
			Chunks:         chunks,
		},
	}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("tool does not implement InvocationTool")
	}

	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "call_rag"}, json.RawMessage(`{"keywords":["实验"],"max_hits":10}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	modelContent := agentruntimev2.ToolResultModelVisibleContent(*result)
	var model map[string]any
	if err := json.Unmarshal([]byte(modelContent), &model); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	items := model["items"].([]any)
	gotIDs := make([]string, 0, len(items))
	for _, raw := range items {
		item := raw.(map[string]any)
		gotIDs = append(gotIDs, item["item_id"].(string))
	}
	wantIDs := []string{"a1", "b1", "c1", "a2", "a3", "b2"}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("model item ids = %v, want %v", gotIDs, wantIDs)
	}
	if !strings.Contains(result.Content, "content a3") || !strings.Contains(result.Content, "content b2") {
		t.Fatalf("raw Content lost expanded chunks: %s", result.Content)
	}
}

func TestSearchRAGChunksModelViewKeepsLargeExpansionInsideCompactBudget(t *testing.T) {
	chunks := make([]RAGChunkHit, 0, 45)
	for i := 0; i < 45; i++ {
		chunks = append(chunks, RAGChunkHit{
			ChunkID:         "chunk_" + strconv.Itoa(i),
			Content:         strings.Repeat("evidence "+strconv.Itoa(i)+" ", 30),
			FileID:          "file_" + strconv.Itoa(i%3),
			FileName:        "file.pdf",
			EvidenceGroupID: "group_" + strconv.Itoa(i/3),
		})
	}
	search := &recordingSearchRAGChunks{
		response: &SearchRAGChunksResponse{
			RowCount:       len(chunks),
			ExpandedGroups: 15,
			Chunks:         chunks,
		},
	}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("tool does not implement InvocationTool")
	}

	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "call_rag"}, json.RawMessage(`{"keywords":["论文","实验"],"max_hits":10}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	modelContent := agentruntimev2.ToolResultModelVisibleContent(*result)
	var model map[string]any
	if err := json.Unmarshal([]byte(modelContent), &model); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if emitted := int(model["emitted_item_count"].(float64)); emitted != ragChunksModelViewMaxItems {
		t.Fatalf("emitted_item_count = %d, want %d", emitted, ragChunksModelViewMaxItems)
	}
	if omitted := int(model["omitted_item_count"].(float64)); omitted != len(chunks)-ragChunksModelViewMaxItems {
		t.Fatalf("omitted_item_count = %d, want %d", omitted, len(chunks)-ragChunksModelViewMaxItems)
	}
	if previewChars := int(model["content_preview_chars"].(float64)); previewChars > ragChunksModelViewTotalPreviewRunes {
		t.Fatalf("content_preview_chars = %d, want <= %d", previewChars, ragChunksModelViewTotalPreviewRunes)
	}
	if len(modelContent) > 10_000 {
		t.Fatalf("model content len = %d, want under default tool output budget", len(modelContent))
	}
	if !strings.Contains(result.Content, "chunk_44") {
		t.Fatalf("raw Content lost final chunk: %s", result.Content)
	}
}

func TestQuerySQLReturnsRecordedSQLIdx(t *testing.T) {
	rc := NewRunContext()
	ctx := ContextWithRunContext(context.Background(), rc)
	query := &recordingQuerySQL{response: &QuerySQLResponse{
		DBName:     "retail",
		TableNames: []string{"orders"},
		Columns:    []string{"region"},
		Rows:       [][]any{{"华东"}},
		RowCount:   1,
		TotalCount: 1,
	}}
	tool := NewQuerySQLTool(Options{
		Registry: &Registry{QuerySQL: query},
		Scope:    WorkspaceScope{DBName: "retail"},
	})
	invocationTool, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("query tool does not implement InvocationTool")
	}

	out, err := invocationTool.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_sql"}, json.RawMessage(`{"sql":"select region from orders"}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	result := out.Data.(*QuerySQLResponse)
	if result.SQLIdx == nil || *result.SQLIdx != 0 {
		t.Fatalf("sql_idx = %v, want 0", result.SQLIdx)
	}
	if got := out.Metadata["sql_idx"]; got != 0 {
		t.Fatalf("metadata sql_idx = %#v, want 0", got)
	}
	artifactResult := out.Artifacts[0].Data.(*QuerySQLResponse)
	if artifactResult.SQLIdx == nil || *artifactResult.SQLIdx != 0 {
		t.Fatalf("artifact sql_idx = %v, want 0", artifactResult.SQLIdx)
	}
	if query.request.Scope.DBName != "retail" {
		t.Fatalf("query scope db_name = %q, want retail", query.request.Scope.DBName)
	}
}

func TestDescribeSchemaMetadataIncludesInjectedSemanticKeys(t *testing.T) {
	metadata := describeSchemaToolMetadata(&DescribeSchemaResponse{Tables: []TableDescription{{
		Name: "orders",
		SemanticEntries: []SemanticEntry{
			{ModelID: 11017, Kind: "metric", KeyName: "net_sales"},
			{ModelID: 11017, Kind: "named_filter", KeyName: "paid_orders"},
			{ModelID: 11017, Kind: "metric", KeyName: "11017:net_sales"},
		},
	}}})

	keys, ok := metadata["semantic_keys_injected"].([]string)
	if !ok {
		t.Fatalf("semantic_keys_injected missing from metadata: %#v", metadata)
	}
	if got, want := keys, []string{"metric:11017:net_sales", "named_filter:11017:paid_orders"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("semantic keys = %v, want %v", got, want)
	}
	display := metadata["display"].(map[string]any)
	params := display["params"].(map[string]any)
	if got, want := params["semantic_keys_injected"], []string{"metric:11017:net_sales", "named_filter:11017:paid_orders"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("display semantic keys = %v, want %v", got, want)
	}
}

func TestQuerySQLMetadataIncludesSemanticUsage(t *testing.T) {
	metadata := querySQLToolMetadata(&QuerySQLResponse{
		SemanticKeysUsed: []string{" metric:11017:net_sales ", "metric:11017:net_sales", "named_filter:11017:paid_orders"},
		AppliedConstraints: []SQLAppliedConstraint{{
			Kind: "named_filter",
			Key:  "named_filter:11017:paid_orders",
		}},
	})

	keys, ok := metadata["semantic_keys_used"].([]string)
	if !ok {
		t.Fatalf("semantic_keys_used missing from metadata: %#v", metadata)
	}
	if got, want := keys, []string{"metric:11017:net_sales", "named_filter:11017:paid_orders"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("semantic keys = %v, want %v", got, want)
	}
	constraints, ok := metadata["applied_constraints"].([]SQLAppliedConstraint)
	if !ok || len(constraints) != 1 || constraints[0].Key != "named_filter:11017:paid_orders" {
		t.Fatalf("applied constraints = %#v", metadata["applied_constraints"])
	}
}

func TestSubmitFinalAnswerResolvesRAGChunkSources(t *testing.T) {
	rc := NewRunContext()
	ctx := ContextWithRunContext(context.Background(), rc)
	search := &recordingSearchRAGChunks{
		response: &SearchRAGChunksResponse{
			Chunks: []RAGChunkHit{{
				ChunkID:        "chunk_1",
				FileID:         "file_1",
				FileName:       "report.md",
				VolumeID:       "vol_1",
				MarkdownFileID: "md_1",
				PageNumber:     3,
				Content:        "net sales evidence",
			}},
		},
	}
	searchTool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	searchInvocation, ok := searchTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("search tool does not implement InvocationTool")
	}
	searchResult, err := searchInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_rag"}, json.RawMessage(`{"keywords":["net sales"],"max_hits":1}`))
	if err != nil {
		t.Fatalf("search Execute returned error: %v", err)
	}
	if len(searchResult.Artifacts) != 1 || searchResult.Artifacts[0].ArtifactID != "rag_chunks_call_rag" {
		t.Fatalf("search artifacts = %+v", searchResult.Artifacts)
	}

	submitTool := NewSubmitFinalAnswerTool()
	submitInvocation, ok := submitTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("submit tool does not implement InvocationTool")
	}
	submitResult, err := submitInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_answer"}, json.RawMessage(`{
		"answer":"结论基于证据片段。",
		"sources":[{"type":"rag_chunk","chunk_id":"chunk_1"}]
	}`))
	if err != nil {
		t.Fatalf("submit Execute returned error: %v", err)
	}
	output := submitResult.Data.(map[string]any)
	sources := output["sources"].([]FinalAnswerSource)
	if len(sources) != 1 || sources[0].FileID != "file_1" || sources[0].Pages[0] != 3 {
		t.Fatalf("resolved sources = %+v", sources)
	}
	if len(submitResult.Artifacts) != 1 || submitResult.Artifacts[0].Type != "answer" {
		t.Fatalf("submit artifacts = %+v", submitResult.Artifacts)
	}
	metadataSources := submitResult.Artifacts[0].Metadata["source_refs"].([]FinalAnswerSource)
	if !reflect.DeepEqual(metadataSources, sources) {
		t.Fatalf("artifact sources = %+v, want %+v", metadataSources, sources)
	}
}

func TestSubmitFinalAnswerResolvesRAGChunkVisualRefs(t *testing.T) {
	rc := NewRunContext()
	ctx := ContextWithRunContext(context.Background(), rc)
	search := &recordingSearchRAGChunks{
		response: &SearchRAGChunksResponse{
			Chunks: []RAGChunkHit{{
				ChunkID:     "chunk_1",
				FileID:      "file_1",
				FileName:    "report.md",
				Content:     "<table><tr><td>[image:image_block_1]</td></tr></table>",
				ImageFileID: "table_image_1",
				VisualRefs: []RAGImageRef{{
					ChunkID:     "image_chunk_1",
					ObjectID:    "image_block_1",
					ImageFileID: "embedded_image_1",
				}},
			}},
		},
	}
	searchTool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	searchInvocation := searchTool.(agentruntimev2.InvocationTool)
	if _, err := searchInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_rag"}, json.RawMessage(`{"keywords":["table"],"max_hits":1}`)); err != nil {
		t.Fatalf("search Execute returned error: %v", err)
	}

	submitTool := NewSubmitFinalAnswerTool()
	submitInvocation := submitTool.(agentruntimev2.InvocationTool)
	submitResult, err := submitInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_answer"}, json.RawMessage(`{
		"answer":"| 项目 | 图片 |\n|---|---|\n| A | 见图 |",
		"sources":[{"type":"rag_chunk","chunk_id":"chunk_1"}]
	}`))
	if err != nil {
		t.Fatalf("submit Execute returned error: %v", err)
	}
	output := submitResult.Data.(map[string]any)
	sources := output["sources"].([]FinalAnswerSource)
	if len(sources) != 1 {
		t.Fatalf("sources = %+v", sources)
	}
	if len(sources[0].VisualRefs) != 1 {
		t.Fatalf("visual refs = %+v, want embedded image", sources[0].VisualRefs)
	}
	if sources[0].VisualRefs[0].ImageFileID != "embedded_image_1" {
		t.Fatalf("visual refs = %+v", sources[0].VisualRefs)
	}
}

func TestSubmitFinalAnswerResolvesSQLResultSources(t *testing.T) {
	rc := NewRunContext()
	ctx := ContextWithRunContext(context.Background(), rc)
	query := &recordingQuerySQL{response: &QuerySQLResponse{
		DBName:     "retail",
		SQL:        "select region, net_sales from orders",
		TableNames: []string{"orders"},
		Columns:    []string{"region", "net_sales"},
		Rows:       [][]any{{"华东", 100}},
		RowCount:   1,
		TotalCount: 1,
	}}
	queryTool := NewQuerySQLTool(Options{
		Registry: &Registry{QuerySQL: query},
		Scope:    WorkspaceScope{DBName: "retail"},
	})
	queryInvocation, ok := queryTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("query tool does not implement InvocationTool")
	}
	queryResult, err := queryInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_sql"}, json.RawMessage(`{"sql":"select region, net_sales from orders"}`))
	if err != nil {
		t.Fatalf("query Execute returned error: %v", err)
	}
	if len(queryResult.Artifacts) != 1 || queryResult.Artifacts[0].ArtifactID != "sql_result_call_sql" {
		t.Fatalf("query artifacts = %+v", queryResult.Artifacts)
	}

	submitTool := NewSubmitFinalAnswerTool()
	submitInvocation, ok := submitTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("submit tool does not implement InvocationTool")
	}
	submitResult, err := submitInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_answer"}, json.RawMessage(`{
		"answer":"华东净销售额为 100。",
		"sources":[{"type":"sql_result","artifact_id":"sql_result_call_sql"}]
	}`))
	if err != nil {
		t.Fatalf("submit Execute returned error: %v", err)
	}
	output := submitResult.Data.(map[string]any)
	sources := output["sources"].([]FinalAnswerSource)
	if len(sources) != 1 || sources[0].Type != "sql_table" || sources[0].Database != "retail" || sources[0].Table != "orders" || sources[0].SQL != "select region, net_sales from orders" {
		t.Fatalf("resolved sources = %+v", sources)
	}
}

func TestSubmitFinalAnswerPreservesDocumentSourceTags(t *testing.T) {
	startMS := int64(0)
	endMS := int64(1250)
	rc := NewRunContext()
	rc.RecordRAGChunksArtifact("rag_chunks_call_1", SearchRAGChunksResponse{Chunks: []RAGChunkHit{{
		ChunkID:    "chunk_1",
		FileID:     "file_1",
		FileName:   "policy.pdf",
		SourceTags: []string{" policy ", "policy", ""},
		StartMS:    &startMS,
		EndMS:      &endMS,
	}}})
	rc.RecordVisualSearchArtifact("visual_search_call_1", SearchVisualImageResponse{Results: []VisualSearchHit{{
		ObjectID:       "object_1",
		SourceFileID:   "file_1",
		SourceFileName: "policy.pdf",
		PageNumber:     1,
		ImageFileID:    "image_1",
		SourceTags:     []string{"visual", " visual "},
	}}})
	ctx := ContextWithRunContext(context.Background(), rc)
	submitTool := NewSubmitFinalAnswerTool()
	submitInvocation, ok := submitTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("submit tool does not implement InvocationTool")
	}

	submitResult, err := submitInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_answer"}, json.RawMessage(`{
		"answer":"结论基于文档证据。",
		"sources":[{"type":"rag_chunk","chunk_id":"chunk_1"},{"type":"visual_hit","image_file_id":"image_1"}]
	}`))
	if err != nil {
		t.Fatalf("submit Execute returned error: %v", err)
	}
	output := submitResult.Data.(map[string]any)
	sources := output["sources"].([]FinalAnswerSource)
	if len(sources) != 2 {
		t.Fatalf("sources = %+v, want two sources", sources)
	}
	if sources[0].StartMS == nil || *sources[0].StartMS != 0 || sources[0].EndMS == nil || *sources[0].EndMS != 1250 {
		t.Fatalf("rag source time range = %v..%v", sources[0].StartMS, sources[0].EndMS)
	}
	for _, source := range sources {
		switch source.Type {
		case "rag_chunk":
			want := []string{" policy ", "policy", ""}
			if !reflect.DeepEqual(source.SourceTags, want) {
				t.Fatalf("rag source tags = %#v, want %#v", source.SourceTags, want)
			}
		case "visual_hit":
			want := []string{"visual", " visual "}
			if !reflect.DeepEqual(source.SourceTags, want) {
				t.Fatalf("visual source tags = %#v, want %#v", source.SourceTags, want)
			}
		default:
			t.Fatalf("unexpected source type %q: %+v", source.Type, source)
		}
	}
}

func TestSubmitFinalAnswerPreservesAnswerText(t *testing.T) {
	rc := NewRunContext()
	rc.RecordSQLResultArtifact("sql_result_call_sql", QuerySQLResponse{
		DBName:     "retail",
		TableNames: []string{"orders"},
		Columns:    []string{"region", "net_sales"},
		Rows:       [][]any{{"华东", 100}},
		RowCount:   1,
		TotalCount: 1,
	})
	ctx := ContextWithRunContext(context.Background(), rc)
	submitTool := NewSubmitFinalAnswerTool()
	submitInvocation, ok := submitTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("submit tool does not implement InvocationTool")
	}

	const answer = "  华东净销售额为 100。\n"
	submitResult, err := submitInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_answer"}, json.RawMessage(`{
		"answer":"  华东净销售额为 100。\n",
		"sources":[{"type":"sql_result","artifact_id":"sql_result_call_sql"}]
	}`))
	if err != nil {
		t.Fatalf("submit Execute returned error: %v", err)
	}
	output := submitResult.Data.(map[string]any)
	if got := output["answer"]; got != answer {
		t.Fatalf("output answer = %q, want %q", got, answer)
	}
	if got := submitResult.Metadata["answer"]; got != answer {
		t.Fatalf("metadata answer = %q, want %q", got, answer)
	}
	if len(submitResult.Artifacts) != 1 {
		t.Fatalf("submit artifacts = %+v", submitResult.Artifacts)
	}
	artifact := submitResult.Artifacts[0]
	if artifact.DisplayText != answer || len(artifact.Parts) != 1 || artifact.Parts[0].Text != answer {
		t.Fatalf("artifact answer not preserved: %+v", artifact)
	}
}

func TestSearchVisualImageUsesQueryVisualsAndSubmitResolvesVisualHit(t *testing.T) {
	rc := NewRunContext()
	ctx := ContextWithRunContext(context.Background(), rc)
	visual := &recordingSearchVisualImage{
		response: &SearchVisualImageResponse{
			Results: []VisualSearchHit{{
				ObjectID:        "obj_1",
				ObjectKind:      "drawing_view",
				SourceFileID:    "source_1",
				SourceFileName:  "drawing.pdf",
				PageNumber:      1,
				BBox:            []float64{10, 20, 110, 220},
				ImageFileID:     "image_1",
				PageImageFileID: "page_image_1",
			}},
		},
	}
	visualTool := NewSearchVisualImageTool(Options{
		Registry: &Registry{SearchVisualImage: visual},
		Scope:    WorkspaceScope{WorkspaceID: "ws_1"},
		QueryVisuals: []QueryVisualRef{{
			FileID:   "query_visual_1",
			FileName: "clip.png",
			MimeType: "image/png",
		}},
	})
	visualInvocation, ok := visualTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("visual tool does not implement InvocationTool")
	}
	visualResult, err := visualInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_visual"}, json.RawMessage(`{"query_visual":1,"query_text":"泵站图","ranking_profile":"visual_object_first","top_k":3}`))
	if err != nil {
		t.Fatalf("visual Execute returned error: %v", err)
	}
	if visual.request.QueryVisualFileID != "query_visual_1" || visual.request.QueryText != "泵站图" || visual.request.RankingProfile != VisualSearchRankingProfileVisualObjectFirst || visual.request.TopK != 3 {
		t.Fatalf("visual request = %+v", visual.request)
	}
	if len(visualResult.Artifacts) != 1 || visualResult.Artifacts[0].ArtifactID != "visual_search_call_visual" {
		t.Fatalf("visual artifacts = %+v", visualResult.Artifacts)
	}

	submitTool := NewSubmitFinalAnswerTool()
	submitInvocation, ok := submitTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("submit tool does not implement InvocationTool")
	}
	submitResult, err := submitInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_answer"}, json.RawMessage(`{
		"answer":"图纸里包含目标泵站。",
		"sources":[{"type":"visual_hit","image_file_id":"image_1"}]
	}`))
	if err != nil {
		t.Fatalf("submit Execute returned error: %v", err)
	}
	output := submitResult.Data.(map[string]any)
	sources := output["sources"].([]FinalAnswerSource)
	if len(sources) != 1 || sources[0].Type != "visual_hit" || sources[0].FileID != "source_1" || sources[0].ImageFileID != "image_1" {
		t.Fatalf("resolved visual sources = %+v", sources)
	}
	if len(sources[0].VisualRefs) != 1 || sources[0].VisualRefs[0].ObjectID != "obj_1" {
		t.Fatalf("visual refs = %+v", sources[0].VisualRefs)
	}
}

func TestSearchVisualImageDefaultsCurrentImageForImageTextSearch(t *testing.T) {
	visual := &recordingSearchVisualImage{}
	tool := NewSearchVisualImageTool(Options{
		Registry: &Registry{SearchVisualImage: visual},
		Scope:    WorkspaceScope{WorkspaceID: "ws_1"},
		QueryVisuals: []QueryVisualRef{{
			FileID:   "query_visual_1",
			FileName: "clip.png",
			MimeType: "image/png",
		}},
	})
	invocationTool, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("tool does not implement InvocationTool")
	}

	_, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{}, json.RawMessage(`{"query_text":"60±3 shA"}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if visual.request.QueryVisualFileID != "query_visual_1" {
		t.Fatalf("QueryVisualFileID = %q, want current message image", visual.request.QueryVisualFileID)
	}
	if visual.request.QueryText != "60±3 shA" {
		t.Fatalf("QueryText = %q, want query text", visual.request.QueryText)
	}
	if visual.request.RankingProfile != VisualSearchRankingProfileVisualObjectFirst {
		t.Fatalf("RankingProfile = %q, want %q", visual.request.RankingProfile, VisualSearchRankingProfileVisualObjectFirst)
	}
}

func TestSearchVisualImageIgnoresUnavailableQueryVisualWhenTextProvided(t *testing.T) {
	visual := &recordingSearchVisualImage{}
	tool := NewSearchVisualImageTool(Options{
		Registry: &Registry{SearchVisualImage: visual},
		Scope:    WorkspaceScope{WorkspaceID: "ws_1"},
	})
	invocationTool, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("tool does not implement InvocationTool")
	}

	_, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{}, json.RawMessage(`{"query_visual":1,"query_text":"前稳定杆图纸"}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if visual.request.QueryVisualFileID != "" {
		t.Fatalf("QueryVisualFileID = %q, want empty visual input", visual.request.QueryVisualFileID)
	}
	if visual.request.QueryText != "前稳定杆图纸" {
		t.Fatalf("QueryText = %q, want text query", visual.request.QueryText)
	}
}

func TestSearchVisualImageStillRejectsImageOnlyWithoutCurrentImage(t *testing.T) {
	visual := &recordingSearchVisualImage{}
	tool := NewSearchVisualImageTool(Options{
		Registry: &Registry{SearchVisualImage: visual},
		Scope:    WorkspaceScope{WorkspaceID: "ws_1"},
	})
	invocationTool, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("tool does not implement InvocationTool")
	}

	_, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{}, json.RawMessage(`{"query_visual":1}`))
	if err == nil {
		t.Fatalf("Execute returned nil error, want missing query visual error")
	}
	if !strings.Contains(err.Error(), "query_visual was provided but no query visual is available") {
		t.Fatalf("error = %v, want missing query visual error", err)
	}
}

func TestSearchVisualImageAcceptsTextRegionRankingProfile(t *testing.T) {
	visual := &recordingSearchVisualImage{}
	tool := NewSearchVisualImageTool(Options{
		Registry: &Registry{SearchVisualImage: visual},
		Scope:    WorkspaceScope{WorkspaceID: "ws_1"},
		QueryVisuals: []QueryVisualRef{{
			FileID:   "query_visual_1",
			FileName: "table.png",
			MimeType: "image/png",
		}},
	})
	invocationTool, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("tool does not implement InvocationTool")
	}

	_, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{}, json.RawMessage(`{"query_visual":1,"query_text":"材料 PA66+GF35","ranking_profile":"visual_text_region_first"}`))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if visual.request.RankingProfile != VisualSearchRankingProfileTextRegionFirst {
		t.Fatalf("RankingProfile = %q, want %q", visual.request.RankingProfile, VisualSearchRankingProfileTextRegionFirst)
	}
}

func TestSubmitFinalAnswerAcceptsSingleSourceObject(t *testing.T) {
	rc := NewRunContext()
	rc.RecordVisualSearchArtifact("visual_search_call_1", SearchVisualImageResponse{Results: []VisualSearchHit{{
		ObjectID:        "obj_1",
		ObjectKind:      "drawing_view",
		SourceFileID:    "source_1",
		SourceFileName:  "20C114257.pdf",
		PageNumber:      1,
		ImageFileID:     "image_1",
		PageImageFileID: "page_image_1",
	}}})
	ctx := ContextWithRunContext(context.Background(), rc)
	submitTool := NewSubmitFinalAnswerTool()
	submitInvocation, ok := submitTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("submit tool does not implement InvocationTool")
	}
	submitResult, err := submitInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_answer"}, json.RawMessage(`{
		"answer":"命中 20C114257.pdf。",
		"sources":{"type":"visual_hit","image_file_id":"image_1"}
	}`))
	if err != nil {
		t.Fatalf("submit returned error: %v", err)
	}
	output := submitResult.Data.(map[string]any)
	sources := output["sources"].([]FinalAnswerSource)
	if len(sources) != 1 || sources[0].FileName != "20C114257.pdf" || sources[0].ImageFileID != "image_1" {
		t.Fatalf("resolved sources = %+v, want one visual source", sources)
	}
}

func TestSubmitFinalAnswerRejectsVisualFileOnlySource(t *testing.T) {
	rc := NewRunContext()
	rc.RecordVisualSearchArtifact("visual_search_call_1", SearchVisualImageResponse{Results: []VisualSearchHit{{
		ObjectID:        "obj_1",
		SourceFileID:    "source_1",
		PageNumber:      1,
		BBox:            []float64{10, 20, 110, 220},
		ImageFileID:     "image_1",
		PageImageFileID: "page_image_1",
	}}})
	ctx := ContextWithRunContext(context.Background(), rc)
	submitTool := NewSubmitFinalAnswerTool()
	submitInvocation, ok := submitTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("submit tool does not implement InvocationTool")
	}
	_, err := submitInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{}, json.RawMessage(`{
		"answer":"图纸里包含目标泵站。",
		"sources":[{"type":"visual_hit","file_id":"source_1"}]
	}`))
	if err == nil {
		t.Fatalf("submit returned nil err, want visual_hit key error")
	}
}

func TestSubmitFinalAnswerAcceptsNoEvidenceAnswerWithEmptySources(t *testing.T) {
	ctx := ContextWithRunContext(context.Background(), NewRunContext())
	submitTool := NewSubmitFinalAnswerTool()
	submitInvocation, ok := submitTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("submit tool does not implement InvocationTool")
	}
	submitResult, err := submitInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{}, json.RawMessage(`{
		"answer":"没有在当前知识范围内找到相关图纸。",
		"sources":[]
	}`))
	if err != nil {
		t.Fatalf("submit returned error: %v", err)
	}
	output := submitResult.Data.(map[string]any)
	if got := output["source_count"]; got != 0 {
		t.Fatalf("source_count = %v, want 0", got)
	}
}

func TestSubmitFinalAnswerRejectsNamedVisualDocumentMissingFromSources(t *testing.T) {
	rc := NewRunContext()
	rc.RecordVisualSearchArtifact("visual_search_call_1", SearchVisualImageResponse{Results: []VisualSearchHit{
		{
			ObjectID:        "obj_398",
			ObjectKind:      "drawing_view",
			SourceFileID:    "source_398",
			SourceFileName:  "20C114398.pdf",
			PageNumber:      1,
			ImageFileID:     "image_398",
			PageImageFileID: "page_image_398",
		},
		{
			ObjectID:        "obj_684",
			ObjectKind:      "drawing_view",
			SourceFileID:    "source_684",
			SourceFileName:  "20C114684.pdf",
			PageNumber:      1,
			ImageFileID:     "image_684",
			PageImageFileID: "page_image_684",
		},
	}})
	ctx := ContextWithRunContext(context.Background(), rc)
	submitTool := NewSubmitFinalAnswerTool()
	submitInvocation, ok := submitTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("submit tool does not implement InvocationTool")
	}
	_, err := submitInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_answer"}, json.RawMessage(`{
		"answer":"最匹配的是 20C114398.pdf。其他相关文件如 20C114684.pdf 也包含类似尺寸。",
		"sources":[{"type":"visual_hit","image_file_id":"image_684"}]
	}`))
	if err == nil {
		t.Fatalf("submit returned nil err, want missing named source error")
	}
	for _, want := range []string{"20C114398.pdf", "image_398"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err.Error(), want)
		}
	}
}

func TestSubmitFinalAnswerRejectsMissingNamedRAGVisualRefSource(t *testing.T) {
	rc := NewRunContext()
	rc.RecordRAGChunksArtifact("rag_chunks_call_1", SearchRAGChunksResponse{Chunks: []RAGChunkHit{
		{
			ChunkID:         "chunk_257",
			FileID:          "source_257",
			FileName:        "20C114257.pdf",
			PageNumber:      1,
			ObjectID:        "object_10",
			ObjectKind:      "page",
			ImageFileID:     "image_257",
			PageImageFileID: "page_image_257",
		},
	}})
	ctx := ContextWithRunContext(context.Background(), rc)
	submitTool := NewSubmitFinalAnswerTool()
	submitInvocation, ok := submitTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("submit tool does not implement InvocationTool")
	}
	_, err := submitInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_answer"}, json.RawMessage(`{
		"answer":"还检索到一张 20C114257.pdf（后稳定杆衬套），未列入上表。",
		"sources":[]
	}`))
	if err == nil {
		t.Fatalf("submit returned nil err, want missing visual_hit source error")
	}
	for _, want := range []string{"20C114257.pdf", "object_10"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error = %q, want %q", err.Error(), want)
		}
	}
}

func TestSubmitFinalAnswerAcceptsNamedFileCoveredByRAGChunkWhenRAGAlsoHasVisualRefs(t *testing.T) {
	rc := NewRunContext()
	rc.RecordRAGChunksArtifact("rag_chunks_call_1", SearchRAGChunksResponse{Chunks: []RAGChunkHit{
		{
			ChunkID:    "chunk_0",
			FileID:     "file_report",
			FileName:   "2112205248_方佳俊_检测简明报告.pdf",
			PageNumber: 1,
			BBox:       []float64{48, 68, 546, 726},
		},
		{
			ChunkID:    "chunk_3",
			FileID:     "file_paper",
			FileName:   "11845_2112205248_LW.pdf",
			PageNumber: 1,
			BBox:       []float64{125, 415, 497, 435},
		},
	}})
	ctx := ContextWithRunContext(context.Background(), rc)
	submitTool := NewSubmitFinalAnswerTool()
	submitInvocation, ok := submitTool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("submit tool does not implement InvocationTool")
	}
	submitResult, err := submitInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_answer"}, json.RawMessage(`{
		"answer":"该信息来自《2112205248_方佳俊_检测简明报告.pdf》和《11845_2112205248_LW.pdf》。",
		"sources":[
			{"type":"rag_chunk","chunk_id":"chunk_0"},
			{"type":"rag_chunk","chunk_id":"chunk_3"}
		]
	}`))
	if err != nil {
		t.Fatalf("submit Execute returned error: %v", err)
	}
	output := submitResult.Data.(map[string]any)
	sources := output["sources"].([]FinalAnswerSource)
	if len(sources) != 2 || sources[0].Type != "rag_chunk" || sources[1].Type != "rag_chunk" {
		t.Fatalf("sources = %+v, want two rag_chunk sources", sources)
	}
}

func TestSubmitFinalAnswerMissingNamedVisualDocumentErrorUsesRecordedOrder(t *testing.T) {
	for i := 0; i < 20; i++ {
		rc := NewRunContext()
		rc.RecordVisualSearchArtifact("visual_search_call_1", SearchVisualImageResponse{Results: []VisualSearchHit{
			{
				ObjectID:        "obj_b",
				ObjectKind:      "detail",
				SourceFileID:    "source_257",
				SourceFileName:  "20C114257.pdf",
				PageNumber:      1,
				ImageFileID:     "image_b",
				PageImageFileID: "page_image_b",
			},
			{
				ObjectID:        "obj_a",
				ObjectKind:      "detail",
				SourceFileID:    "source_257",
				SourceFileName:  "20C114257.pdf",
				PageNumber:      1,
				ImageFileID:     "image_a",
				PageImageFileID: "page_image_a",
			},
			{
				ObjectID:        "obj_z",
				ObjectKind:      "detail",
				SourceFileID:    "source_100",
				SourceFileName:  "10C114100.pdf",
				PageNumber:      1,
				ImageFileID:     "image_z",
				PageImageFileID: "page_image_z",
			},
		}})
		ctx := ContextWithRunContext(context.Background(), rc)
		submitTool := NewSubmitFinalAnswerTool()
		submitInvocation, ok := submitTool.(agentruntimev2.InvocationTool)
		if !ok {
			t.Fatalf("submit tool does not implement InvocationTool")
		}
		_, err := submitInvocation.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_answer"}, json.RawMessage(`{
			"answer":"命中 20C114257.pdf 和 10C114100.pdf。",
			"sources":[]
		}`))
		if err == nil {
			t.Fatalf("submit returned nil err, want missing source error")
		}
		msg := err.Error()
		for _, want := range []string{"20C114257.pdf", "obj_b", "image_b", "10C114100.pdf", "obj_z", "image_z"} {
			if !strings.Contains(msg, want) {
				t.Fatalf("iteration %d error = %q, want %q", i, msg, want)
			}
		}
		if strings.Contains(msg, "obj_a") || strings.Contains(msg, "image_a") {
			t.Fatalf("iteration %d error = %q, must not use lower lexical same-file candidate", i, msg)
		}
	}
}

func TestRunContextResolvesRAGChunkVisualRefsAsVisualHits(t *testing.T) {
	rc := NewRunContext()
	rc.RecordRAGChunksArtifact("rag_chunks_call_1", SearchRAGChunksResponse{Chunks: []RAGChunkHit{{
		ChunkID:         "chunk_1",
		FileID:          "source_1",
		FileName:        "1433113.pdf",
		MarkdownFileID:  "md_1",
		PageNumber:      1,
		Content:         "DETAIL Y evidence",
		ObjectID:        "object_5",
		ObjectKind:      "detail",
		ImageFileID:     "image_5",
		PageImageFileID: "page_image_1",
		BBox:            []float64{330, 1280, 1210, 1935},
	}}})

	got, err := rc.ResolveFinalAnswerSources([]FinalAnswerSource{
		{Type: "visual_hit", ObjectID: "object_5"},
	})
	if err != nil {
		t.Fatalf("ResolveFinalAnswerSources returned err: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("sources = %#v, want one visual source", got)
	}
	source := got[0]
	if source.Type != "visual_hit" || source.FileID != "source_1" || source.FileName != "1433113.pdf" || source.ObjectID != "object_5" || source.ImageFileID != "image_5" || source.PageImageFileID != "page_image_1" {
		t.Fatalf("visual source = %#v, want RAG visual backlinks", source)
	}
	if len(source.ChunkIDs) != 1 || source.ChunkIDs[0] != "chunk_1" || source.MarkdownFileID != "md_1" {
		t.Fatalf("visual source RAG linkage = %#v, want chunk and markdown ids", source)
	}
	if len(source.Pages) != 1 || source.Pages[0] != 1 || len(source.BBox) != 4 || source.BBox[0] != 330 || source.BBox[3] != 1935 {
		t.Fatalf("visual source geometry = %#v, want page/bbox", source)
	}
	if len(source.VisualRefs) != 1 || source.VisualRefs[0].ObjectID != "object_5" || source.VisualRefs[0].ChunkID != "chunk_1" {
		t.Fatalf("visual refs = %#v, want RAG visual ref", source.VisualRefs)
	}
}

func TestRunContextVisualSourceUsesTrustedSemanticModelIDOverModelSelection(t *testing.T) {
	source := resolveTrustedSemanticModelVisualSource(t)
	requirePublicSemanticModelID(t, "resolved visual source", source, 42)
}

func TestRunContextVisualRefKeepsTrustedSemanticModelID(t *testing.T) {
	source := resolveTrustedSemanticModelVisualSource(t)
	if len(source.VisualRefs) != 1 {
		t.Fatalf("resolved visual refs = %+v, want one trusted visual ref", source.VisualRefs)
	}
	requirePublicSemanticModelID(t, "resolved visual ref", source.VisualRefs[0], 42)
}

func TestProjectFinalAnswerSourceRefsKeepsTrustedSemanticModelID(t *testing.T) {
	source := resolveTrustedSemanticModelVisualSource(t)
	refs := ProjectFinalAnswerSourceRefs([]FinalAnswerSource{source})
	if len(refs) != 1 {
		t.Fatalf("projected refs = %+v, want one visual source ref", refs)
	}
	requirePublicSemanticModelID(t, "projected visual source ref", refs[0], 42)
	if len(refs[0].VisualRefs) != 1 {
		t.Fatalf("projected visual_refs = %+v, want one ID-level nested visual ref", refs[0].VisualRefs)
	}
	requirePublicSemanticModelID(t, "projected nested visual ref", refs[0].VisualRefs[0], 42)
}

func TestRunContextKeepsSameVisualEvidenceSeparateAcrossSemanticModels(t *testing.T) {
	var recorded SearchVisualImageResponse
	if err := json.Unmarshal([]byte(`{
		"results":[
				{
					"semantic_model_id":101,
					"source_row_id":"source_101",
				"object_id":"shared_object",
				"object_kind":"detail",
				"source_file_id":"shared_file",
				"source_file_name":"model-101.pdf",
				"page_number":1,
				"image_file_id":"shared_image",
				"page_image_file_id":"shared_page_image"
			},
				{
					"semantic_model_id":202,
					"source_row_id":"source_202",
				"object_id":"shared_object",
				"object_kind":"detail",
				"source_file_id":"shared_file",
				"source_file_name":"model-202.pdf",
				"page_number":1,
				"image_file_id":"shared_image",
				"page_image_file_id":"shared_page_image"
			}
		],
		"count":2
	}`), &recorded); err != nil {
		t.Fatalf("decode same-ID multi-model visual results: %v", err)
	}
	rc := NewRunContext()
	rc.RecordVisualSearchArtifact("visual_multi_model", recorded)

	var selected []FinalAnswerSource
	if err := json.Unmarshal([]byte(`[
		{
			"type":"visual_hit",
			"semantic_model_id":101,
			"image_file_id":"shared_image"
		},
		{
			"type":"visual_hit",
			"semantic_model_id":202,
			"image_file_id":"shared_image"
		}
	]`), &selected); err != nil {
		t.Fatalf("decode same-ID multi-model selection: %v", err)
	}
	resolved, err := rc.ResolveFinalAnswerSources(selected)
	if err != nil {
		t.Fatalf("ResolveFinalAnswerSources() error = %v", err)
	}
	if len(resolved) != 2 {
		t.Fatalf(
			"resolved sources = %+v, want two model-owned sources when file/object/image IDs collide",
			resolved,
		)
	}

	wantFileNameByModel := map[int64]string{
		101: "model-101.pdf",
		202: "model-202.pdf",
	}
	wantSourceRowByModel := map[int64]string{
		101: "source_101",
		202: "source_202",
	}
	for index, source := range resolved {
		modelID := publicSemanticModelID(t, fmt.Sprintf("resolved source %d", index), source)
		wantFileName, ok := wantFileNameByModel[modelID]
		if !ok {
			t.Fatalf("resolved source %d has unexpected semantic_model_id %d: %+v", index, modelID, source)
		}
		if source.FileName != wantFileName {
			t.Fatalf(
				"resolved source model %d file_name = %q, want %q; model ownership was crossed",
				modelID,
				source.FileName,
				wantFileName,
			)
		}
		if source.SourceRowID != wantSourceRowByModel[modelID] {
			t.Fatalf("resolved source model %d source_row_id = %q, want %q", modelID, source.SourceRowID, wantSourceRowByModel[modelID])
		}
		if len(source.VisualRefs) != 1 {
			t.Fatalf("resolved source model %d visual_refs = %+v, want one", modelID, source.VisualRefs)
		}
		requirePublicSemanticModelID(t, "resolved same-ID visual ref", source.VisualRefs[0], modelID)
		delete(wantFileNameByModel, modelID)
	}
	if len(wantFileNameByModel) != 0 {
		t.Fatalf("resolved sources lost semantic models: %+v", wantFileNameByModel)
	}
}

func TestRunContextRejectsAmbiguousSameIDVisualSelectionWithoutTrustedOwner(t *testing.T) {
	var recorded SearchVisualImageResponse
	if err := json.Unmarshal([]byte(`{
		"results":[
			{
				"semantic_model_id":101,
				"object_id":"shared_object",
				"source_file_id":"shared_file",
				"source_file_name":"model-101.pdf",
				"image_file_id":"shared_image",
				"page_image_file_id":"shared_page_image"
			},
			{
				"semantic_model_id":202,
				"object_id":"shared_object",
				"source_file_id":"shared_file",
				"source_file_name":"model-202.pdf",
				"image_file_id":"shared_image",
				"page_image_file_id":"shared_page_image"
			}
		],
		"count":2
	}`), &recorded); err != nil {
		t.Fatalf("decode same-ID multi-model visual results: %v", err)
	}

	for _, test := range []struct {
		name      string
		selection string
	}{
		{
			name:      "missing owner",
			selection: `[{"type":"visual_hit","image_file_id":"shared_image"}]`,
		},
		{
			name:      "unknown owner",
			selection: `[{"type":"visual_hit","semantic_model_id":999,"image_file_id":"shared_image"}]`,
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			rc := NewRunContext()
			rc.RecordVisualSearchArtifact("visual_multi_model", recorded)
			var selected []FinalAnswerSource
			if err := json.Unmarshal([]byte(test.selection), &selected); err != nil {
				t.Fatalf("decode visual selection: %v", err)
			}

			resolved, err := rc.ResolveFinalAnswerSources(selected)
			if err == nil {
				t.Errorf("ResolveFinalAnswerSources() error = nil, want explicit ambiguous/unknown semantic model owner error")
			}
			if len(resolved) != 0 {
				t.Errorf("ResolveFinalAnswerSources() = %+v, want zero results on ambiguous/unknown owner", resolved)
			}
		})
	}
}

func resolveTrustedSemanticModelVisualSource(t *testing.T) FinalAnswerSource {
	t.Helper()

	var recorded SearchVisualImageResponse
	if err := json.Unmarshal([]byte(`{
		"results":[{
			"semantic_model_id":42,
			"object_id":"object_5",
			"object_kind":"detail",
			"source_file_id":"source_1",
			"source_file_name":"1433113.pdf",
			"page_number":1,
			"image_file_id":"image_5",
			"page_image_file_id":"page_image_1",
			"bbox":[330,1280,1210,1935]
		}],
		"count":1
	}`), &recorded); err != nil {
		t.Fatalf("decode trusted visual search result: %v", err)
	}

	rc := NewRunContext()
	rc.RecordVisualSearchArtifact("visual_search_call_1", recorded)

	var modelSelection []FinalAnswerSource
	if err := json.Unmarshal([]byte(`[{
		"type":"visual_hit",
		"object_id":"object_5",
		"semantic_model_id":999,
		"visual_refs":[{
			"object_id":"object_5",
			"semantic_model_id":999
		}]
	}]`), &modelSelection); err != nil {
		t.Fatalf("decode model-selected visual source: %v", err)
	}

	resolved, err := rc.ResolveFinalAnswerSources(modelSelection)
	if err != nil {
		t.Fatalf("ResolveFinalAnswerSources returned err: %v", err)
	}
	if len(resolved) != 1 {
		t.Fatalf("resolved sources = %+v, want one visual source", resolved)
	}
	return resolved[0]
}

func requirePublicSemanticModelID(t *testing.T, label string, value any, want int64) {
	t.Helper()

	got := publicSemanticModelID(t, label, value)
	if got != want {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal %s after semantic_model_id mismatch: %v", label, err)
		}
		t.Fatalf(
			"%s semantic_model_id = %d, want trusted recorded value %d; contract=%s",
			label,
			got,
			want,
			encoded,
		)
	}
}

func publicSemanticModelID(t *testing.T, label string, value any) int64 {
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
	if !exists || !numberOK {
		t.Fatalf(
			"%s semantic_model_id = %v (present=%t), want a trusted numeric value; contract=%s",
			label,
			rawModelID,
			exists,
			encoded,
		)
	}
	return int64(gotModelID)
}

func TestSearchRAGChunksRejectsArgumentsOutsideKeywordsArrayContract(t *testing.T) {
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: &recordingSearchRAGChunks{}}})
	for _, raw := range []string{
		`{"keywards":["供应商"],"max_hits":2}`,
		`{"keywords":"R20±0.25","max_hits":2}`,
	} {
		if _, err := tool.Execute(context.Background(), json.RawMessage(raw)); err == nil {
			t.Fatalf("Execute(%s) = nil error, want invalid arguments", raw)
		}
	}
}

func TestSelectFinalSourcesAcceptsEmptySourcesAfterRetrievalWithCandidates(t *testing.T) {
	rc := NewRunContext()
	rc.RecordRAGChunksArtifact("rag_chunks_call_1", SearchRAGChunksResponse{Chunks: []RAGChunkHit{{
		ChunkID:  "chunk_1",
		FileID:   "file_1",
		FileName: "20C114257.pdf",
	}}})
	ctx := ContextWithRunContext(context.Background(), rc)
	tool := NewSelectFinalSourcesTool()
	inv, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("select tool does not implement InvocationTool")
	}
	res, err := inv.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_select"}, json.RawMessage(`{"sources":[]}`))
	if err != nil {
		t.Fatalf("select returned error: %v", err)
	}
	output := res.Data.(map[string]any)
	if output["source_count"] != 0 || output["candidate_count"] != 1 {
		t.Fatalf("selection counts = sources %v candidates %v, want 0 and 1", output["source_count"], output["candidate_count"])
	}
	if selected, ok := rc.SelectedFinalAnswerSources(); !ok || len(selected) != 0 {
		t.Fatalf("selected sources = (%v,%t), want empty selected", selected, ok)
	}
}

func TestSelectFinalSourcesAcceptsEmptySourcesWithoutCitableEvidence(t *testing.T) {
	rc := NewRunContext()
	rc.RecordRAGChunksArtifact("rag_chunks_empty", SearchRAGChunksResponse{})
	ctx := ContextWithRunContext(context.Background(), rc)
	tool := NewSelectFinalSourcesTool()
	inv, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("select tool does not implement InvocationTool")
	}
	res, err := inv.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_select"}, json.RawMessage(`{"sources":[]}`))
	if err != nil {
		t.Fatalf("select returned error: %v", err)
	}
	if output := res.Data.(map[string]any); output["source_count"] != 0 {
		t.Fatalf("source_count = %v, want 0", output["source_count"])
	}
	if selected, ok := rc.SelectedFinalAnswerSources(); !ok || len(selected) != 0 {
		t.Fatalf("selected sources = (%v,%t), want empty selected", selected, ok)
	}
}

func TestSelectFinalSourcesRejectsEmptySourcesBeforeCitableEvidenceRetrieval(t *testing.T) {
	ctx := ContextWithRunContext(context.Background(), NewRunContext())
	tool := NewSelectFinalSourcesTool()
	inv, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("select tool does not implement InvocationTool")
	}
	_, err := inv.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_select"}, json.RawMessage(`{"sources":[]}`))
	if err == nil || !strings.Contains(err.Error(), "sources cannot be empty before a citable evidence retrieval completes") {
		t.Fatalf("select error = %v, want missing retrieval rejection", err)
	}
}

func TestSelectFinalSourcesLocksResolvedSources(t *testing.T) {
	rc := NewRunContext()
	rc.RecordRAGChunksArtifact("rag_chunks_call_1", SearchRAGChunksResponse{Chunks: []RAGChunkHit{{
		ChunkID:  "chunk_1",
		FileID:   "file_1",
		FileName: "policy.md",
		Content:  "供应商必须通过资质审核。",
	}}})
	ctx := ContextWithRunContext(context.Background(), rc)
	tool := NewSelectFinalSourcesTool()
	inv, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatalf("select tool does not implement InvocationTool")
	}
	res, err := inv.ExecuteInvocation(ctx, agentruntimev2.ToolInvocation{CallID: "call_select"}, json.RawMessage(`{
		"sources":[{"type":"rag_chunk","chunk_id":"chunk_1"}]
	}`))
	if err != nil {
		t.Fatalf("select returned error: %v", err)
	}
	selected, ok := rc.SelectedFinalAnswerSources()
	if !ok || len(selected) != 1 {
		t.Fatalf("selected = (%+v,%t)", selected, ok)
	}
	if selected[0].FileName != "policy.md" {
		t.Fatalf("run context selected source = %+v, want full resolved source", selected[0])
	}
	output := res.Data.(map[string]any)
	if output["selected"] != true || output["source_count"] != 1 || output["ok"] != true || output["accepted"] != true {
		t.Fatalf("output = %+v", output)
	}
	if _, exists := output["source_coverage_candidates"]; exists {
		t.Fatalf("model-visible output must not include source_coverage_candidates: %+v", output)
	}
	if output["candidate_count"] != 1 {
		t.Fatalf("candidate_count = %v, want 1", output["candidate_count"])
	}
	sources, ok := output["sources"].([]FinalAnswerSource)
	if !ok || len(sources) != 1 {
		t.Fatalf("sources = %#v", output["sources"])
	}
	if sources[0].ChunkID != "chunk_1" || sources[0].Type != "rag_chunk" {
		t.Fatalf("projected source = %+v", sources[0])
	}
	if sources[0].MarkdownFileID != "" || sources[0].VolumeID != "" || len(sources[0].Pages) != 0 || len(sources[0].BBox) != 0 || len(sources[0].VisualRefs) != 0 {
		t.Fatalf("projected source leaked full evidence payload: %+v", sources[0])
	}
	if res.Metadata["selected_final_sources"] != true || res.Metadata["accepted"] != true || res.Metadata["ok"] != true {
		t.Fatalf("metadata = %+v", res.Metadata)
	}
	if _, exists := res.Metadata["source_coverage_candidates"]; exists {
		t.Fatalf("metadata must not include source_coverage_candidates: %+v", res.Metadata)
	}
}

func TestProjectFinalAnswerSourceRefsKeepsOnlyIDs(t *testing.T) {
	startMS := int64(0)
	endMS := int64(1250)
	refs := ProjectFinalAnswerSourceRefs([]FinalAnswerSource{{
		Type:           "rag_chunk",
		SourceRowID:    "source_1",
		ArtifactID:     "rag_chunks_call_1",
		ChunkIDs:       []string{"chunk_1", "chunk_2"},
		FileID:         "file_1",
		FileName:       "policy.md",
		StartMS:        &startMS,
		EndMS:          &endMS,
		VolumeID:       "vol_1",
		MarkdownFileID: "md_1",
		Pages:          []int{1, 2},
		BBox:           []float64{1, 2, 3, 4},
		VisualRefs:     []FinalAnswerVisualRef{{ObjectID: "obj_1", ImageFileID: "img_1"}},
	}, {
		Type:        "visual_hit",
		ArtifactID:  "visual_call_1",
		ObjectID:    "obj_2",
		ImageFileID: "img_2",
		FileName:    "drawing.pdf",
		BBox:        []float64{9, 8, 7, 6},
	}})
	if len(refs) != 2 {
		t.Fatalf("refs = %+v", refs)
	}
	if refs[0].ChunkID != "" || len(refs[0].ChunkIDs) != 2 || refs[0].VolumeID != "" || refs[0].MarkdownFileID != "" || len(refs[0].Pages) != 0 || len(refs[0].BBox) != 0 || len(refs[0].VisualRefs) != 1 {
		t.Fatalf("rag projection = %+v", refs[0])
	}
	if refs[0].SourceRowID != "source_1" {
		t.Fatalf("rag projected source row id = %q, want source_1", refs[0].SourceRowID)
	}
	if refs[0].VisualRefs[0].ObjectID != "obj_1" || refs[0].VisualRefs[0].ImageFileID != "img_1" || len(refs[0].VisualRefs[0].BBox) != 0 {
		t.Fatalf("rag projected visual ref = %+v, want identifiers without geometry", refs[0].VisualRefs[0])
	}
	if refs[0].StartMS == nil || *refs[0].StartMS != 0 || refs[0].EndMS == nil || *refs[0].EndMS != 1250 {
		t.Fatalf("rag projection time range = %v..%v", refs[0].StartMS, refs[0].EndMS)
	}
	if refs[1].ObjectID != "obj_2" || refs[1].ImageFileID != "img_2" || len(refs[1].BBox) != 0 {
		t.Fatalf("visual projection = %+v", refs[1])
	}
}

func TestProjectFinalAnswerSourceRefsKeepsNestedVisualRefSemanticModelID(t *testing.T) {
	var source FinalAnswerSource
	if err := json.Unmarshal([]byte(`{
		"type":"visual_hit",
		"semantic_model_id":42,
		"artifact_id":"visual_call_1",
		"file_id":"file_1",
		"object_id":"object_1",
		"image_file_id":"image_1",
		"visual_refs":[{
			"semantic_model_id":42,
			"object_id":"object_1",
			"image_file_id":"image_1",
			"page":7,
			"bbox":[1,2,3,4]
		}]
	}`), &source); err != nil {
		t.Fatalf("decode visual source contract: %v", err)
	}

	refs := ProjectFinalAnswerSourceRefs([]FinalAnswerSource{source})
	if len(refs) != 1 {
		t.Fatalf("projected refs = %+v, want one", refs)
	}
	t.Run("source_ref", func(t *testing.T) {
		requirePublicSemanticModelID(t, "projected source ref", refs[0], 42)
	})
	t.Run("nested_visual_ref", func(t *testing.T) {
		if len(refs[0].VisualRefs) != 1 {
			t.Fatalf("projected visual_refs = %+v, want one ID-level nested ref", refs[0].VisualRefs)
		}
		nested := refs[0].VisualRefs[0]
		requirePublicSemanticModelID(t, "projected nested visual ref", nested, 42)
		if nested.ObjectID != "object_1" || nested.ImageFileID != "image_1" || nested.Page != 7 {
			t.Fatalf("projected nested visual ref identifiers = %+v", nested)
		}
		if len(nested.BBox) != 0 {
			t.Fatalf("projected nested visual ref leaked geometry: %+v", nested)
		}
	})
}

func TestNormalizeFinalAnswerSourcesKeepsTimedChunksSeparate(t *testing.T) {
	firstStart, firstEnd := int64(0), int64(1250)
	secondStart, secondEnd := int64(1250), int64(2500)
	sources := normalizeFinalAnswerSources([]FinalAnswerSource{
		{Type: "rag_chunk", FileID: "file_1", ChunkID: "chunk_1", StartMS: &firstStart, EndMS: &firstEnd},
		{Type: "rag_chunk", FileID: "file_1", ChunkID: "chunk_2", StartMS: &secondStart, EndMS: &secondEnd},
	})
	if len(sources) != 2 {
		t.Fatalf("timed sources = %+v", sources)
	}
	if len(sources[0].ChunkIDs) != 1 || sources[0].ChunkIDs[0] != "chunk_1" || *sources[0].StartMS != firstStart || *sources[0].EndMS != firstEnd {
		t.Fatalf("first timed source = %+v", sources[0])
	}
	if len(sources[1].ChunkIDs) != 1 || sources[1].ChunkIDs[0] != "chunk_2" || *sources[1].StartMS != secondStart || *sources[1].EndMS != secondEnd {
		t.Fatalf("second timed source = %+v", sources[1])
	}
}

func TestSearchRAGChunksModelViewPrioritizesAnchorsWithoutMutatingServiceResponse(t *testing.T) {
	chunks := []RAGChunkHit{
		{ChunkID: "neighbor-a-first", Content: "neighbor a first", FileID: "file-a", EvidenceGroupID: "group-a"},
		{ChunkID: "anchor-rank-3", Content: "anchor rank 3", FileID: "file-z", EvidenceGroupID: "group-a", RetrievalAnchorRank: 3},
		{ChunkID: "neighbor-a-second", Content: "neighbor a second", FileID: "file-a", EvidenceGroupID: "group-a"},
		{ChunkID: "anchor-rank-5", Content: "anchor rank 5", FileID: "file-a", EvidenceGroupID: "group-b", RetrievalAnchorRank: 5},
		{ChunkID: "neighbor-b-first", Content: "neighbor b first", FileID: "file-b", EvidenceGroupID: "group-b"},
		{ChunkID: "ungrouped-neighbor", Content: "ungrouped neighbor", FileID: "file-c"},
	}
	originalChunks := append([]RAGChunkHit(nil), chunks...)
	search := &recordingSearchRAGChunks{response: &SearchRAGChunksResponse{
		Keywards: []string{"anchor"},
		Chunks:   chunks,
		RowCount: len(chunks),
	}}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool, ok := tool.(agentruntimev2.InvocationTool)
	if !ok {
		t.Fatal("tool does not implement InvocationTool")
	}

	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "anchor-priority"}, json.RawMessage(`{"keywords":["anchor"],"max_hits":10}`))
	if err != nil {
		t.Fatalf("ExecuteInvocation() error = %v", err)
	}
	if result.ModelView == nil {
		t.Fatal("ModelView missing")
	}
	modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
	if err != nil {
		t.Fatalf("ModelContentWithPolicy() error = %v", err)
	}
	var payload struct {
		Items []struct {
			ItemID string `json:"item_id"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	gotIDs := make([]string, 0, len(payload.Items))
	for _, item := range payload.Items {
		gotIDs = append(gotIDs, item.ItemID)
	}
	wantIDs := []string{
		"anchor-rank-3",
		"anchor-rank-5",
		"neighbor-a-first",
		"neighbor-b-first",
		"ungrouped-neighbor",
		"neighbor-a-second",
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("model item ids = %v, want %v", gotIDs, wantIDs)
	}
	if !reflect.DeepEqual(search.response.Chunks, originalChunks) {
		t.Fatalf("ModelView ordering modified service response chunks: got %+v, want %+v", search.response.Chunks, originalChunks)
	}
	if strings.Contains(result.Content, "retrieval_anchor_rank") {
		t.Fatalf("public tool JSON leaked internal rank: %s", result.Content)
	}
}

func TestSearchRAGChunksModelViewUsesStableKeyForEqualAnchorRanks(t *testing.T) {
	search := &recordingSearchRAGChunks{response: &SearchRAGChunksResponse{
		Chunks: []RAGChunkHit{
			{ChunkID: "anchor-z", Content: "z", FileID: "z-file", IndexVersion: "v1", RetrievalAnchorRank: 1},
			{ChunkID: "anchor-a", Content: "a", FileID: "a-file", IndexVersion: "v1", RetrievalAnchorRank: 1},
		},
		RowCount: 2,
	}}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool := tool.(agentruntimev2.InvocationTool)
	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "stable-anchor-ranks"}, json.RawMessage(`{"keywords":["anchor"],"max_hits":10}`))
	if err != nil {
		t.Fatalf("ExecuteInvocation() error = %v", err)
	}
	modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
	if err != nil {
		t.Fatalf("ModelContentWithPolicy() error = %v", err)
	}
	var payload struct {
		Items []struct {
			ItemID string `json:"item_id"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if len(payload.Items) != 2 || payload.Items[0].ItemID != "anchor-a" || payload.Items[1].ItemID != "anchor-z" {
		t.Fatalf("equal-rank anchors = %+v, want anchor-a then anchor-z", payload.Items)
	}
}

func TestSearchRAGChunksModelViewKeepsFocusedTableCriterionWithinDefaultPreview(t *testing.T) {
	const condition = "Reject if >50% of the surface"
	table := `<table><tr><td>SOP area</td><td>Scratch / Damage / Deform</td><td>` +
		strings.Repeat("long definition text ", 18) + `</td><td>` + condition + `</td></tr>` +
		`<tr><td>SOP area</td><td>Solder Extrusion</td><td>other definition</td><td>Reject if any bridge</td></tr></table>`
	if strings.Contains(string([]rune(table)[:ragChunksModelViewPreviewRunes]), condition) {
		t.Fatal("test fixture must place the criterion beyond the existing prefix preview")
	}
	search := &recordingSearchRAGChunks{response: &SearchRAGChunksResponse{
		Keywards: []string{"划伤", "scratch", "刮伤", "SOP", "通用规范"},
		Chunks: []RAGChunkHit{{
			ChunkID: "target-table", FileID: "file_1", Content: table, ChunkType: "table",
		}},
		RowCount: 1,
	}}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool := tool.(agentruntimev2.InvocationTool)
	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "table-preview"}, json.RawMessage(`{"keywords":["划伤","scratch","刮伤","SOP","通用规范"],"max_hits":30}`))
	if err != nil {
		t.Fatalf("ExecuteInvocation() error = %v", err)
	}
	if result.ModelView == nil {
		t.Fatal("ModelView missing")
	}
	modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
	if err != nil {
		t.Fatalf("ModelContentWithPolicy() error = %v", err)
	}
	var payload struct {
		Items []struct {
			ItemID         string `json:"item_id"`
			ContentPreview string `json:"content_preview"`
			Refs           struct {
				ChunkID string `json:"chunk_id"`
			} `json:"refs"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if len(payload.Items) != 1 || payload.Items[0].ItemID != "target-table" || payload.Items[0].Refs.ChunkID != "target-table" {
		t.Fatalf("serialized item is incomplete: %+v", payload.Items)
	}
	preview := payload.Items[0].ContentPreview
	wantPreview := "SOP area | Scratch / Damage / Deform | " + condition
	if preview != wantPreview {
		t.Fatalf("table preview = %q, want unchanged unique-winner preview %q", preview, wantPreview)
	}
}

func TestSearchRAGChunksModelViewKeepsAllHighestScoringTableRowsWithinDefaultPreview(t *testing.T) {
	table := `<table>` +
		`<tr><td>LGA pad</td><td>Scratch / 划伤</td><td>Reject if visible</td></tr>` +
		`<tr><td>SOP area</td><td>Scratch / Damage / Deform 划伤</td><td>Reject if >50% of the surface</td></tr>` +
		`</table>`
	search := &recordingSearchRAGChunks{response: &SearchRAGChunksResponse{
		Keywards: []string{"General spec", "Scratch", "划伤", "1.7"},
		Chunks: []RAGChunkHit{{
			ChunkID: "tied-table-rows", FileID: "file_1", Content: table, ChunkType: "table",
		}},
		RowCount: 1,
	}}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool := tool.(agentruntimev2.InvocationTool)
	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "tied-table-rows"}, json.RawMessage(`{"keywords":["General spec","Scratch","划伤","1.7"],"max_hits":30}`))
	if err != nil {
		t.Fatalf("ExecuteInvocation() error = %v", err)
	}
	modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
	if err != nil {
		t.Fatalf("ModelContentWithPolicy() error = %v", err)
	}
	var payload struct {
		Items []struct {
			ContentPreview string `json:"content_preview"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("table preview items = %+v, want 1", payload.Items)
	}
	preview := payload.Items[0].ContentPreview
	for _, required := range []string{"LGA pad", "SOP area", "Scratch / Damage / Deform", ">50%"} {
		if !strings.Contains(preview, required) {
			t.Fatalf("table preview = %q, want %q", preview, required)
		}
	}
	if strings.Index(preview, "LGA pad") > strings.Index(preview, "SOP area") {
		t.Fatalf("table preview changed source row order: %q", preview)
	}
	if got := len([]rune(preview)); got > ragChunksModelViewPreviewRunes {
		t.Fatalf("table preview runes = %d, want <= %d: %q", got, ragChunksModelViewPreviewRunes, preview)
	}
}

func TestSearchRAGChunksModelViewPreservesAtomsForThreeLongTiedTableRows(t *testing.T) {
	longMatch := func(label string) string {
		return "needle priority " + label + " " + strings.Repeat("matched evidence ", 8)
	}
	table := `<table>` +
		`<tr><td rowspan="3">SOP</td><td>Rule-A</td><td>` + longMatch("A") + `</td><td>Reject-A</td></tr>` +
		`<tr><td>Rule-B</td><td>` + longMatch("B") + `</td><td>Reject-B</td></tr>` +
		`<tr><td>Rule-C</td><td>` + longMatch("C") + `</td><td>Reject-C</td></tr>` +
		`<tr><td>Lower</td><td>needle only</td><td>Reject-Lower</td></tr>` +
		`</table>`
	search := &recordingSearchRAGChunks{response: &SearchRAGChunksResponse{
		Keywards: []string{"needle", "SOP", "Rule", "priority"},
		Chunks: []RAGChunkHit{{
			ChunkID: "three-tied-table-rows", FileID: "file_1", Content: table, ChunkType: "table",
		}},
		RowCount: 1,
	}}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool := tool.(agentruntimev2.InvocationTool)
	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "three-tied-table-rows"}, json.RawMessage(`{"keywords":["needle","SOP","Rule","priority"],"max_hits":30}`))
	if err != nil {
		t.Fatalf("ExecuteInvocation() error = %v", err)
	}
	modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
	if err != nil {
		t.Fatalf("ModelContentWithPolicy() error = %v", err)
	}
	var payload struct {
		Items []struct {
			ItemID         string `json:"item_id"`
			ContentPreview string `json:"content_preview"`
			Refs           struct {
				ChunkID string `json:"chunk_id"`
			} `json:"refs"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if len(payload.Items) != 1 || payload.Items[0].ItemID != "three-tied-table-rows" || payload.Items[0].Refs.ChunkID != "three-tied-table-rows" {
		t.Fatalf("serialized item is incomplete: %+v", payload.Items)
	}
	preview := payload.Items[0].ContentPreview
	lines := strings.Split(preview, "\n")
	if len(lines) != 3 {
		t.Fatalf("table preview lines = %d, want 3 highest-scoring rows: %q", len(lines), preview)
	}
	for index, label := range []string{"A", "B", "C"} {
		for _, required := range []string{"SOP", "Rule-" + label, "needle", "Reject-" + label} {
			if !strings.Contains(lines[index], required) {
				t.Fatalf("table preview line %d = %q, want %q", index, lines[index], required)
			}
		}
	}
	if strings.Contains(preview, "Lower") || strings.Contains(preview, "Reject-Lower") {
		t.Fatalf("lower-scoring row entered table preview: %q", preview)
	}
	if got := len([]rune(preview)); got > ragChunksModelViewPreviewRunes {
		t.Fatalf("table preview runes = %d, want <= %d: %q", got, ragChunksModelViewPreviewRunes, preview)
	}
}

func TestSearchRAGChunksModelViewKeepsEightTiedRowsWithinDefaultPreview(t *testing.T) {
	var table strings.Builder
	table.WriteString("<table>")
	for index := 0; index < 8; index++ {
		table.WriteString("<tr><td>R")
		table.WriteString(strconv.Itoa(index))
		table.WriteString("</td><td>nC")
		table.WriteString(strconv.Itoa(index))
		table.WriteString("</td></tr>")
	}
	table.WriteString("</table>")

	search := &recordingSearchRAGChunks{response: &SearchRAGChunksResponse{
		Keywards: []string{"n"},
		Chunks: []RAGChunkHit{{
			ChunkID: "eight-tied-table-rows", FileID: "file_1", Content: table.String(), ChunkType: "table",
		}},
		RowCount: 1,
	}}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool := tool.(agentruntimev2.InvocationTool)
	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "eight-tied-table-rows"}, json.RawMessage(`{"keywords":["n"],"max_hits":30}`))
	if err != nil {
		t.Fatalf("ExecuteInvocation() error = %v", err)
	}
	modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
	if err != nil {
		t.Fatalf("ModelContentWithPolicy() error = %v", err)
	}
	var payload struct {
		Items []struct {
			ContentPreview string `json:"content_preview"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("table preview items = %+v, want 1", payload.Items)
	}
	preview := payload.Items[0].ContentPreview
	lines := strings.Split(preview, "\n")
	if len(lines) != 8 {
		t.Fatalf("table preview lines = %d, want all 8 tied rows: %q", len(lines), preview)
	}
	for index, line := range lines {
		want := "R" + strconv.Itoa(index) + " | nC" + strconv.Itoa(index)
		if line != want {
			t.Fatalf("table preview line %d = %q, want %q", index, line, want)
		}
		if strings.Count(line, "nC") != 1 {
			t.Fatalf("matched condition cell was duplicated in line %d: %q", index, line)
		}
	}
	if got := len([]rune(preview)); got > ragChunksModelViewPreviewRunes {
		t.Fatalf("table preview runes = %d, want <= %d: %q", got, ragChunksModelViewPreviewRunes, preview)
	}
}

func TestSearchRAGChunksModelViewKeepsMatchExcerptForEightLongTiedRows(t *testing.T) {
	var table strings.Builder
	table.WriteString("<table>")
	for index := 0; index < 8; index++ {
		label := strconv.Itoa(index)
		table.WriteString("<tr><td>identity-" + label + strings.Repeat("-i", 10) + "</td>")
		table.WriteString("<td>context-" + label + strings.Repeat("-x", 10) + "</td>")
		table.WriteString("<td>matched-prefix-" + label + strings.Repeat("-m", 10) + "-needle-tail</td>")
		table.WriteString("<td>condition-" + label + strings.Repeat("-c", 10) + "</td></tr>")
	}
	table.WriteString("</table>")

	search := &recordingSearchRAGChunks{response: &SearchRAGChunksResponse{
		Keywards: []string{"needle"},
		Chunks: []RAGChunkHit{{
			ChunkID: "eight-long-tied-table-rows", FileID: "file_1", Content: table.String(), ChunkType: "table",
		}},
		RowCount: 1,
	}}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool := tool.(agentruntimev2.InvocationTool)
	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "eight-long-tied-table-rows"}, json.RawMessage(`{"keywords":["needle"],"max_hits":30}`))
	if err != nil {
		t.Fatalf("ExecuteInvocation() error = %v", err)
	}
	modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
	if err != nil {
		t.Fatalf("ModelContentWithPolicy() error = %v", err)
	}
	var payload struct {
		Items []struct {
			ContentPreview string `json:"content_preview"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("table preview items = %+v, want 1", payload.Items)
	}
	preview := payload.Items[0].ContentPreview
	lines := strings.Split(preview, "\n")
	if len(lines) != 8 {
		t.Fatalf("table preview lines = %d, want all 8 tied rows: %q", len(lines), preview)
	}
	for index, line := range lines {
		for _, required := range []string{"iden", "need", "cond"} {
			if !strings.Contains(line, required) {
				t.Fatalf("table preview line %d = %q, want %q", index, line, required)
			}
		}
	}
	if got := len([]rune(preview)); got > ragChunksModelViewPreviewRunes {
		t.Fatalf("table preview runes = %d, want <= %d: %q", got, ragChunksModelViewPreviewRunes, preview)
	}
}

func TestSearchRAGChunksModelViewKeepsMatchedTableCellWithinDefaultPreview(t *testing.T) {
	table := `<table><tr><td>row identity</td><td>ordinary description</td><td>needle matched evidence</td><td>supporting detail</td><td>final criterion</td></tr></table>`
	search := &recordingSearchRAGChunks{response: &SearchRAGChunksResponse{
		Keywards: []string{"needle"},
		Chunks: []RAGChunkHit{{
			ChunkID: "matched-cell-table", FileID: "file_1", Content: table, ChunkType: "table",
		}},
		RowCount: 1,
	}}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool := tool.(agentruntimev2.InvocationTool)
	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "matched-table-cell"}, json.RawMessage(`{"keywords":["needle"],"max_hits":30}`))
	if err != nil {
		t.Fatalf("ExecuteInvocation() error = %v", err)
	}
	modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
	if err != nil {
		t.Fatalf("ModelContentWithPolicy() error = %v", err)
	}
	var payload struct {
		Items []struct {
			ContentPreview string `json:"content_preview"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if len(payload.Items) != 1 || !strings.Contains(payload.Items[0].ContentPreview, "needle matched evidence") {
		t.Fatalf("table preview = %+v, want matched table cell", payload.Items)
	}
}

func TestSearchRAGChunksModelViewKeepsCrossCellPhraseEvidenceWithinDefaultPreview(t *testing.T) {
	table := `<table><tr><td>row identity</td><td>ordinary description</td><td>needle</td><td>matched evidence</td><td>final criterion</td></tr></table>`
	search := &recordingSearchRAGChunks{response: &SearchRAGChunksResponse{
		Keywards: []string{"needle matched"},
		Chunks: []RAGChunkHit{{
			ChunkID: "cross-cell-phrase-table", FileID: "file_1", Content: table, ChunkType: "table",
		}},
		RowCount: 1,
	}}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool := tool.(agentruntimev2.InvocationTool)
	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "cross-cell-phrase"}, json.RawMessage(`{"keywords":["needle matched"],"max_hits":30}`))
	if err != nil {
		t.Fatalf("ExecuteInvocation() error = %v", err)
	}
	modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
	if err != nil {
		t.Fatalf("ModelContentWithPolicy() error = %v", err)
	}
	var payload struct {
		Items []struct {
			ContentPreview string `json:"content_preview"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("table preview items = %+v, want 1", payload.Items)
	}
	for _, required := range []string{"row identity", "needle", "matched evidence", "final criterion"} {
		if !strings.Contains(payload.Items[0].ContentPreview, required) {
			t.Fatalf("table preview = %q, want %q", payload.Items[0].ContentPreview, required)
		}
	}
}

func TestSearchRAGChunksModelViewKeepsRowspanContextWithinDefaultPreview(t *testing.T) {
	table := `<table><tr><td rowspan="2">SOP area</td><td>Solder Extrusion</td><td>Reject if any bridge</td></tr><tr><td>Scratch / Damage / Deform</td><td>Reject if >50% of the surface</td></tr></table>`
	search := &recordingSearchRAGChunks{response: &SearchRAGChunksResponse{
		Keywards: []string{"scratch"},
		Chunks: []RAGChunkHit{{
			ChunkID: "rowspan-table", FileID: "file_1", Content: table, ChunkType: "table",
		}},
		RowCount: 1,
	}}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool := tool.(agentruntimev2.InvocationTool)
	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "rowspan-table"}, json.RawMessage(`{"keywords":["scratch"],"max_hits":30}`))
	if err != nil {
		t.Fatalf("ExecuteInvocation() error = %v", err)
	}
	modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
	if err != nil {
		t.Fatalf("ModelContentWithPolicy() error = %v", err)
	}
	var payload struct {
		Items []struct {
			ContentPreview string `json:"content_preview"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("table preview items = %+v, want 1", payload.Items)
	}
	preview := payload.Items[0].ContentPreview
	for _, required := range []string{"SOP area", "Scratch / Damage / Deform", "Reject if >50% of the surface"} {
		if !strings.Contains(preview, required) {
			t.Fatalf("table preview = %q, want %q", preview, required)
		}
	}
}

func TestSearchRAGChunksModelViewKeepsNonFirstRowspanInLogicalColumnOrder(t *testing.T) {
	for _, test := range []struct {
		name        string
		leadingCell string
	}{
		{name: "non-empty leading cell", leadingCell: "<td>other-area</td>"},
		{name: "empty leading cell", leadingCell: "<td></td>"},
	} {
		t.Run(test.name, func(t *testing.T) {
			table := `<table><tr>` + test.leadingCell +
				`<td rowspan="2">shared-rule</td><td>` + strings.Repeat("filler ", 40) + `</td></tr>` +
				`<tr><td>target-area</td><td>final-condition</td></tr></table>`
			search := &recordingSearchRAGChunks{response: &SearchRAGChunksResponse{
				Keywards: []string{"target-area shared-rule"},
				Chunks: []RAGChunkHit{{
					ChunkID: "non-first-rowspan-table", FileID: "file_1", Content: table, ChunkType: "table",
				}},
				RowCount: 1,
			}}
			tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
			invocationTool := tool.(agentruntimev2.InvocationTool)
			result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "non-first-rowspan-table"}, json.RawMessage(`{"keywords":["target-area shared-rule"],"max_hits":30}`))
			if err != nil {
				t.Fatalf("ExecuteInvocation() error = %v", err)
			}
			modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
			if err != nil {
				t.Fatalf("ModelContentWithPolicy() error = %v", err)
			}
			var payload struct {
				Items []struct {
					ContentPreview string `json:"content_preview"`
				} `json:"items"`
			}
			if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
				t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
			}
			if len(payload.Items) != 1 {
				t.Fatalf("table preview items = %+v, want 1", payload.Items)
			}
			preview := payload.Items[0].ContentPreview
			for _, required := range []string{"target-area", "shared-rule", "final-condition"} {
				if !strings.Contains(preview, required) {
					t.Fatalf("table preview = %q, want %q", preview, required)
				}
			}
			if strings.Index(preview, "target-area") > strings.Index(preview, "shared-rule") {
				t.Fatalf("table preview moved inherited second-column cell before the first column: %q", preview)
			}
		})
	}
}

func TestSearchRAGChunksModelViewKeepsRank14And15AnchorsWithinDefaultSerializedBudget(t *testing.T) {
	chunks := ragAnchorPriorityTraceChunks()
	search := &recordingSearchRAGChunks{response: &SearchRAGChunksResponse{
		Keywards:       []string{"通用规范", "划伤"},
		Routes:         []string{"fulltext", "vector_l2"},
		Chunks:         chunks,
		RowCount:       len(chunks),
		ExpandedGroups: len(chunks),
	}}
	tool := NewSearchRAGChunksTool(Options{Registry: &Registry{SearchRAGChunks: search}})
	invocationTool := tool.(agentruntimev2.InvocationTool)
	result, err := invocationTool.ExecuteInvocation(context.Background(), agentruntimev2.ToolInvocation{CallID: "trace-shaped-anchor-priority"}, json.RawMessage(`{"keywords":["通用规范","划伤"],"max_hits":10}`))
	if err != nil {
		t.Fatalf("ExecuteInvocation() error = %v", err)
	}
	if result.ModelView == nil {
		t.Fatal("ModelView missing")
	}
	modelContent, err := result.ModelView.ModelContentWithPolicy(agentruntimev2.DefaultToolOutputTruncationPolicy())
	if err != nil {
		t.Fatalf("ModelContentWithPolicy() error = %v", err)
	}
	if len(modelContent) <= 10_000 || len(modelContent) > 12_000 {
		t.Fatalf("serialized default-policy content = %d bytes, want 10000 < bytes <= 12000", len(modelContent))
	}
	var payload struct {
		ItemCount        int `json:"item_count"`
		EmittedItemCount int `json:"emitted_item_count"`
		OmittedItemCount int `json:"omitted_item_count"`
		Items            []struct {
			ItemID string `json:"item_id"`
			Refs   struct {
				ChunkID string `json:"chunk_id"`
			} `json:"refs"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(modelContent), &payload); err != nil {
		t.Fatalf("model content is not valid JSON: %v\n%s", err, modelContent)
	}
	if payload.EmittedItemCount != len(payload.Items) || payload.EmittedItemCount+payload.OmittedItemCount != payload.ItemCount {
		t.Fatalf("model item counts = emitted %d omitted %d total %d items %d", payload.EmittedItemCount, payload.OmittedItemCount, payload.ItemCount, len(payload.Items))
	}
	if payload.EmittedItemCount >= ragChunksModelViewMaxItems {
		t.Fatalf("emitted_item_count = %d, want serialized-byte admission below %d", payload.EmittedItemCount, ragChunksModelViewMaxItems)
	}
	seenTargets := map[string]bool{}
	for _, item := range payload.Items {
		if item.ItemID == "" || item.Refs.ChunkID == "" || item.ItemID != item.Refs.ChunkID {
			t.Fatalf("serialized item is incomplete: %+v", item)
		}
		if !strings.HasPrefix(item.ItemID, "anchor-") {
			t.Fatalf("neighbor %q entered ModelView before an omitted anchor", item.ItemID)
		}
		seenTargets[item.ItemID] = true
	}
	for _, target := range []string{"anchor-page-10-pdf", "anchor-page-10-doc"} {
		if !seenTargets[target] {
			t.Fatalf("target rank 14/15 anchor %q is missing from ModelView items: %+v", target, payload.Items)
		}
	}
	if strings.Contains(result.Content, "retrieval_anchor_rank") {
		t.Fatalf("public tool JSON leaked internal rank: %s", result.Content)
	}
	if !strings.Contains(result.Content, "anchor-page-10-pdf") || !strings.Contains(result.Content, "anchor-page-10-doc") {
		t.Fatalf("raw tool content lost target anchors: %s", result.Content)
	}
}

func ragAnchorPriorityTraceChunks() []RAGChunkHit {
	chunks := make([]RAGChunkHit, 46)
	for index := range chunks {
		id := "neighbor-" + strconv.Itoa(index)
		chunks[index] = RAGChunkHit{
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
		chunks[position] = RAGChunkHit{
			ChunkID:             id,
			Content:             strings.Repeat("通用规范划伤证据", 20) + id,
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
