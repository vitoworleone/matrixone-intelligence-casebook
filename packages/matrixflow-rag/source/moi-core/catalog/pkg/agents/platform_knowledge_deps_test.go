package agents

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	knowledgeservice "github.com/matrixflow/moi-core/agent-tools/knowledge/service"
	embeddingpkg "github.com/matrixflow/moi-core/catalog/pkg/embedding"
	embeddingadapter "github.com/matrixflow/moi-core/catalog/pkg/embedding/adapter"
	catalogpb "github.com/matrixflow/moi-core/model/catalog"
)

type recordingImageEmbeddingAdapter struct {
	lastBody      []byte
	lastBackendID int64
}

func (a *recordingImageEmbeddingAdapter) Embeddings(_ context.Context, backend *catalogpb.Backend, _ *catalogpb.BackendEndpoint, req *embeddingadapter.EmbeddingRequest) (*embeddingadapter.EmbeddingResponse, error) {
	if backend != nil {
		a.lastBackendID = backend.Id
	}
	if req != nil {
		a.lastBody = append([]byte(nil), req.Body...)
	}
	return &embeddingadapter.EmbeddingResponse{
		Body: []byte(`{"data":[{"embedding":[0.1,0.2,0.3]}],"metadata":{"provider":"test"}}`),
	}, nil
}

type singleAdapterRegistry struct {
	adapter embeddingadapter.Adapter
}

func (r singleAdapterRegistry) Get(catalogpb.BackendType) (embeddingadapter.Adapter, bool) {
	if r.adapter == nil {
		return nil, false
	}
	return r.adapter, true
}

func TestCatalogKnowledgeVisualBackendCreateImageEmbeddingMultimodalBody(t *testing.T) {
	raw, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR42mP4z8AAAAMBAQD3A0FDAAAAAElFTkSuQmCC")
	if err != nil {
		t.Fatalf("decode png fixture: %v", err)
	}
	adapter := &recordingImageEmbeddingAdapter{}
	cache := embeddingpkg.NewConfigCache(nil)
	cache.Update("ws_1", 1, &embeddingpkg.WorkspaceConfig{
		WorkspaceIDValue: "ws_1",
		Backends: []*catalogpb.Backend{
			{Id: 1, Name: "image-emb", Type: catalogpb.BackendType_OPENAI, Models: []string{"efficientnet-b3"}},
			{Id: 2, Name: "image-emb-bound", Type: catalogpb.BackendType_OPENAI, Models: []string{"efficientnet-b3"}},
		},
		Endpoints: []*catalogpb.BackendEndpoint{
			{Id: 10, BackendId: 1, Status: catalogpb.EndpointStatus_ONLINE},
			{Id: 20, BackendId: 2, Status: catalogpb.EndpointStatus_ONLINE},
		},
	})
	backend := &catalogKnowledgeVisualBackend{
		configCache:     cache,
		router:          embeddingpkg.NewRouter(),
		adapterRegistry: singleAdapterRegistry{adapter: adapter},
	}

	embedding, meta, err := backend.CreateImageEmbedding(context.Background(), "ws_1", knowledgeservice.VisualImageEmbeddingRequest{
		Model:     "efficientnet-b3",
		BackendID: "2",
		Raw:       raw,
		MimeType:  "image/png",
	})
	if err != nil {
		t.Fatalf("CreateImageEmbedding() error = %v", err)
	}
	if len(embedding) != 3 {
		t.Fatalf("embedding = %#v", embedding)
	}
	if meta["provider"] != "test" {
		t.Fatalf("metadata = %#v", meta)
	}
	if adapter.lastBackendID != 2 {
		t.Fatalf("selected backend_id = %d, want 2", adapter.lastBackendID)
	}

	var body map[string]any
	if err := json.Unmarshal(adapter.lastBody, &body); err != nil {
		t.Fatalf("decode embedding body: %v body=%s", err, adapter.lastBody)
	}
	if body["type"] != "embedding_multimodal" {
		t.Fatalf("type = %#v, want embedding_multimodal", body["type"])
	}
	if body["embedding_mode"] != "fusion" {
		t.Fatalf("embedding_mode = %#v", body["embedding_mode"])
	}
	if body["output_cardinality"] != "one_per_input" {
		t.Fatalf("output_cardinality = %#v", body["output_cardinality"])
	}
	if body["encoding_format"] != "float" {
		t.Fatalf("encoding_format = %#v", body["encoding_format"])
	}
	if body["backend_id"] != float64(2) {
		t.Fatalf("body backend_id = %#v, want 2", body["backend_id"])
	}
	if _, ok := body["images"]; ok {
		t.Fatalf("legacy images[] must not be present: %#v", body)
	}
	inputs, ok := body["input"].([]any)
	if !ok || len(inputs) != 1 {
		t.Fatalf("input = %#v", body["input"])
	}
	input0, _ := inputs[0].(map[string]any)
	contents, _ := input0["content"].([]any)
	if len(contents) != 1 {
		t.Fatalf("content = %#v", input0["content"])
	}
	content0, _ := contents[0].(map[string]any)
	if content0["type"] != "image_url" {
		t.Fatalf("content type = %#v", content0["type"])
	}
	imageURL, _ := content0["image_url"].(map[string]any)
	url, _ := imageURL["url"].(string)
	if !strings.HasPrefix(url, "data:image/png;base64,") {
		t.Fatalf("image_url.url = %q", url)
	}
}

func TestCatalogKnowledgeVisualBackendCreateImageEmbeddingOmitsBackendIDWhenEmpty(t *testing.T) {
	raw, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR42mP4z8AAAAMBAQD3A0FDAAAAAElFTkSuQmCC")
	if err != nil {
		t.Fatalf("decode png fixture: %v", err)
	}
	adapter := &recordingImageEmbeddingAdapter{}
	cache := embeddingpkg.NewConfigCache(nil)
	cache.Update("ws_1", 1, &embeddingpkg.WorkspaceConfig{
		WorkspaceIDValue: "ws_1",
		Backends: []*catalogpb.Backend{
			{Id: 1, Name: "image-emb", Type: catalogpb.BackendType_OPENAI, Models: []string{"efficientnet-b3"}},
		},
		Endpoints: []*catalogpb.BackendEndpoint{
			{Id: 10, BackendId: 1, Status: catalogpb.EndpointStatus_ONLINE},
		},
	})
	backend := &catalogKnowledgeVisualBackend{
		configCache:     cache,
		router:          embeddingpkg.NewRouter(),
		adapterRegistry: singleAdapterRegistry{adapter: adapter},
	}

	if _, _, err := backend.CreateImageEmbedding(context.Background(), "ws_1", knowledgeservice.VisualImageEmbeddingRequest{
		Model:    "efficientnet-b3",
		Raw:      raw,
		MimeType: "image/png",
	}); err != nil {
		t.Fatalf("CreateImageEmbedding() error = %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(adapter.lastBody, &body); err != nil {
		t.Fatalf("decode embedding body: %v", err)
	}
	if _, ok := body["backend_id"]; ok {
		t.Fatalf("backend_id must be omitted when empty: %#v", body)
	}
	if adapter.lastBackendID != 1 {
		t.Fatalf("selected backend_id = %d, want model-only fallback 1", adapter.lastBackendID)
	}
}

func TestCatalogKnowledgeVisualBackendCreateImageEmbeddingRejectsNonNumericBackendID(t *testing.T) {
	raw, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAIAAACQd1PeAAAADElEQVR42mP4z8AAAAMBAQD3A0FDAAAAAElFTkSuQmCC")
	if err != nil {
		t.Fatalf("decode png fixture: %v", err)
	}
	adapter := &recordingImageEmbeddingAdapter{}
	cache := embeddingpkg.NewConfigCache(nil)
	cache.Update("ws_1", 1, &embeddingpkg.WorkspaceConfig{
		WorkspaceIDValue: "ws_1",
		Backends: []*catalogpb.Backend{
			{Id: 1, Name: "image-emb", Type: catalogpb.BackendType_OPENAI, Models: []string{"efficientnet-b3"}},
		},
		Endpoints: []*catalogpb.BackendEndpoint{
			{Id: 10, BackendId: 1, Status: catalogpb.EndpointStatus_ONLINE},
		},
	})
	backend := &catalogKnowledgeVisualBackend{
		configCache:     cache,
		router:          embeddingpkg.NewRouter(),
		adapterRegistry: singleAdapterRegistry{adapter: adapter},
	}

	_, _, err = backend.CreateImageEmbedding(context.Background(), "ws_1", knowledgeservice.VisualImageEmbeddingRequest{
		Model:     "efficientnet-b3",
		BackendID: "local-efficientnet-b3",
		Raw:       raw,
		MimeType:  "image/png",
	})
	if err == nil || !strings.Contains(err.Error(), "must be a numeric catalog backend id") {
		t.Fatalf("error = %v, want non-numeric backend_id rejection", err)
	}
	if len(adapter.lastBody) != 0 {
		t.Fatalf("adapter must not be called for invalid backend_id, body=%s", adapter.lastBody)
	}
}

func TestCatalogKnowledgeVisualBackendCreateImageEmbeddingRejectsInvalidInput(t *testing.T) {
	adapter := &recordingImageEmbeddingAdapter{}
	cache := embeddingpkg.NewConfigCache(nil)
	cache.Update("ws_1", 1, &embeddingpkg.WorkspaceConfig{
		WorkspaceIDValue: "ws_1",
		Backends: []*catalogpb.Backend{
			{Id: 1, Name: "image-emb", Type: catalogpb.BackendType_OPENAI, Models: []string{"efficientnet-b3"}},
		},
		Endpoints: []*catalogpb.BackendEndpoint{
			{Id: 10, BackendId: 1, Status: catalogpb.EndpointStatus_ONLINE},
		},
	})
	backend := &catalogKnowledgeVisualBackend{
		configCache:     cache,
		router:          embeddingpkg.NewRouter(),
		adapterRegistry: singleAdapterRegistry{adapter: adapter},
	}

	if _, _, err := backend.CreateImageEmbedding(context.Background(), "ws_1", knowledgeservice.VisualImageEmbeddingRequest{
		Model: "efficientnet-b3",
	}); err == nil || !strings.Contains(err.Error(), "image bytes are empty") {
		t.Fatalf("empty bytes error = %v", err)
	}
	if _, _, err := backend.CreateImageEmbedding(context.Background(), "ws_1", knowledgeservice.VisualImageEmbeddingRequest{
		Model:    "efficientnet-b3",
		Raw:      []byte("plain text not image"),
		MimeType: "text/plain",
	}); err == nil || !strings.Contains(err.Error(), "image content type must be image/*") {
		t.Fatalf("non-image mime error = %v", err)
	}
	if len(adapter.lastBody) != 0 {
		t.Fatalf("adapter should not be called on validation failure, body=%s", adapter.lastBody)
	}
}


func TestIssue11017KnowledgeImageIndexFromMetaReadsImageEmbeddingKeys(t *testing.T) {
	index, ok, err := knowledgeImageIndexFromMeta("kb_image_index", sql.NullString{
		Valid: true,
		String: `{
			"index_modality": "image",
			"image_embedding_model": "efficientnet-b3",
			"image_embedding_backend_id": "local-efficientnet-b3",
			"image_embedding_dimension": 1536,
			"preprocess_version": "v1",
			"distance_metric": "cosine"
		}`,
	})
	if err != nil {
		t.Fatalf("knowledgeImageIndexFromMeta returned error: %v", err)
	}
	if !ok {
		t.Fatalf("knowledgeImageIndexFromMeta did not identify image index")
	}
	if index.imageEmbeddingModel != "efficientnet-b3" {
		t.Fatalf("imageEmbeddingModel = %q, want efficientnet-b3", index.imageEmbeddingModel)
	}
	if index.imageEmbeddingBackendID != "local-efficientnet-b3" {
		t.Fatalf("imageEmbeddingBackendID = %q, want local-efficientnet-b3", index.imageEmbeddingBackendID)
	}
	if index.imageEmbeddingDimension != 1536 {
		t.Fatalf("imageEmbeddingDimension = %d, want 1536", index.imageEmbeddingDimension)
	}
	if _, err := uniqueKnowledgeResolvedImageIndex([]resolvedKnowledgeImageIndex{index}); err != nil {
		t.Fatalf("uniqueKnowledgeResolvedImageIndex returned error: %v", err)
	}
}

func TestFilterKnowledgeResolvedVectorIndexesKeepsRequestedSemanticIndex(t *testing.T) {
	indexes := []resolvedKnowledgeVectorIndex{
		{vectorTable: "other_kb_text_vec", embeddingModel: "bge-m3"},
		{vectorTable: "current_kb_text_vec", embeddingModel: "bge-m3"},
	}
	filtered := filterKnowledgeResolvedVectorIndexes(indexes, "current_kb_text_vec", "bge-m3")
	vectorTable, embeddingModel, err := uniqueKnowledgeResolvedVectorIndex(filtered)
	if err != nil {
		t.Fatalf("uniqueKnowledgeResolvedVectorIndex returned error: %v", err)
	}
	if vectorTable != "current_kb_text_vec" || embeddingModel != "bge-m3" {
		t.Fatalf("filtered vector index = %q/%q, want current_kb_text_vec/bge-m3", vectorTable, embeddingModel)
	}
}

func TestFilterKnowledgeResolvedImageIndexesKeepsRequestedSemanticIndex(t *testing.T) {
	indexes := []resolvedKnowledgeImageIndex{
		{
			imageVectorTable:        "other_kb_img",
			imageEmbeddingModel:     "efficientnet-b3",
			imageEmbeddingDimension: 1536,
			imagePreprocessVersion:  "efficientnet-b3-v1",
			imageDistanceMetric:     "cosine",
		},
		{
			imageVectorTable:        "current_kb_img",
			imageEmbeddingModel:     "efficientnet-b3",
			imageEmbeddingDimension: 1536,
			imagePreprocessVersion:  "efficientnet-b3-v1",
			imageDistanceMetric:     "cosine",
		},
	}
	filtered := filterKnowledgeResolvedImageIndexes(indexes, "current_kb_img", "efficientnet-b3")
	index, err := uniqueKnowledgeResolvedImageIndex(filtered)
	if err != nil {
		t.Fatalf("uniqueKnowledgeResolvedImageIndex returned error: %v", err)
	}
	if index.imageVectorTable != "current_kb_img" || index.imageEmbeddingModel != "efficientnet-b3" {
		t.Fatalf("filtered image index = %#v, want current_kb_img/efficientnet-b3", index)
	}
}

// Multi-db Resolve leaves DBName empty and emits database.table labels. Agents
// commonly describe a single selected table; ListColumns must still decode the
// prefix using full scope.Tables for knownDB (not only the selected subset).
func TestPlatformKnowledgeGroupTablesByDatabaseMultiDBSubsetUsesFullScope(t *testing.T) {
	scopeTables := []string{"sales.orders", "support.tickets"}
	// Agent subset: one qualified label only (selected lefts==1).
	byDB, order, err := platformKnowledgeGroupTablesByDatabase(scopeTables, []string{"sales.orders"}, "")
	if err != nil {
		t.Fatalf("group subset qualified: %v", err)
	}
	if len(order) != 1 || order[0] != "sales" {
		t.Fatalf("order = %#v, want [sales]", order)
	}
	if got := byDB["sales"]; len(got) != 1 || got[0] != "sales.orders" {
		t.Fatalf("byDB[sales] = %#v, want [sales.orders]", got)
	}

	// Selected-only knownDB (old bug): empty defaultDB + single left fails.
	_, _, err = platformKnowledgeGroupTablesByDatabase(nil, []string{"sales.orders"}, "")
	if err == nil {
		t.Fatal("selected-only multi-db label without scope must fail qualification")
	}
	if !strings.Contains(err.Error(), "missing database qualification") {
		t.Fatalf("error = %v, want missing database qualification", err)
	}
}

func TestPlatformKnowledgeGroupTablesByDatabaseMultiDBFullSelection(t *testing.T) {
	scopeTables := []string{"sales.orders", "support.tickets"}
	byDB, order, err := platformKnowledgeGroupTablesByDatabase(scopeTables, scopeTables, "")
	if err != nil {
		t.Fatalf("group full selection: %v", err)
	}
	if len(order) != 2 {
		t.Fatalf("order = %#v, want two databases", order)
	}
	if len(byDB["sales"]) != 1 || byDB["sales"][0] != "sales.orders" {
		t.Fatalf("sales group = %#v", byDB["sales"])
	}
	if len(byDB["support"]) != 1 || byDB["support"][0] != "support.tickets" {
		t.Fatalf("support group = %#v", byDB["support"])
	}
}

func TestPlatformKnowledgeGroupTablesByDatabaseDefaultDBBareName(t *testing.T) {
	byDB, order, err := platformKnowledgeGroupTablesByDatabase(
		[]string{"ffff_15.test.csv"},
		[]string{"test.csv"},
		"ffff_15",
	)
	if err != nil {
		t.Fatalf("group bare with defaultDB: %v", err)
	}
	if len(order) != 1 || order[0] != "ffff_15" {
		t.Fatalf("order = %#v, want [ffff_15]", order)
	}
	if got := byDB["ffff_15"]; len(got) != 1 || got[0] != "test.csv" {
		t.Fatalf("byDB = %#v, want bare label preserved for response matching", got)
	}
}
