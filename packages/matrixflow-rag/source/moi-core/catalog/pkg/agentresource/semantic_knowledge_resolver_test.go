package agentresource

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	agenttools "github.com/matrixflow/moi-core/agent-tools"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/tenant"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/transaction"
)

type semanticResolverTestPool struct {
	err                       error
	governanceRows            *sqlmock.Rows
	skipGovernanceExpectation bool
	transactionManagerCalls   *int
}

func (p semanticResolverTestPool) GetConnection(context.Context, string) (*sql.DB, error) {
	return nil, p.err
}

func (p semanticResolverTestPool) GetDBExecutor(context.Context, string) (tenant.DBExecutor, error) {
	return nil, p.err
}

func (p semanticResolverTestPool) GetTransactionManager(context.Context, string) (*transaction.Manager, error) {
	if p.err != nil {
		return nil, p.err
	}
	if p.transactionManagerCalls != nil {
		(*p.transactionManagerCalls)++
	}
	db, mock, err := sqlmock.New()
	if err != nil {
		return nil, err
	}
	mock.ExpectBegin()
	if p.skipGovernanceExpectation {
		mock.ExpectCommit()
		return transaction.NewManager(db), nil
	}
	if p.governanceRows != nil {
		mock.ExpectQuery("(?s)SELECT\\s+kbs.model_id,\\s+kbs.source_id,\\s+COALESCE\\(kbs.kb_file_id, ''\\),.*kbs.status <> 'removed'").
			WillReturnRows(p.governanceRows)
	} else {
		mock.ExpectQuery("(?s)SELECT\\s+kbs.model_id,\\s+kbs.source_id,\\s+COALESCE\\(kbs.kb_file_id, ''\\),.*kbs.status <> 'removed'").
			WillReturnRows(sqlmock.NewRows([]string{"model_id", "source_id", "kb_file_id", "db_name", "table_name", "source_table_id", "kb_table_id", "enabled", "expires_at", "force_enabled_after_expiry", "tags"}))
	}
	mock.ExpectCommit()
	return transaction.NewManager(db), nil
}

func (p semanticResolverTestPool) GetTx(context.Context, string) (*sql.Tx, error) {
	return nil, p.err
}

func (p semanticResolverTestPool) Close() error {
	return nil
}

type semanticResolverTestStorage struct {
	model     *tenant.SemanticModelRecord
	models    []*tenant.SemanticModelRecord
	err       error
	getCalls  int
	listCalls int
}

func (s *semanticResolverTestStorage) CreateSemanticModel(context.Context, *tenant.SemanticModelRecord) (*tenant.SemanticModelRecord, error) {
	return nil, errors.New("not implemented")
}

func (s *semanticResolverTestStorage) GetSemanticModel(context.Context, int64) (*tenant.SemanticModelRecord, error) {
	s.getCalls++
	return s.model, s.err
}

func (s *semanticResolverTestStorage) GetSemanticModelForUpdate(ctx context.Context, modelID int64) (*tenant.SemanticModelRecord, error) {
	return s.GetSemanticModel(ctx, modelID)
}

func (s *semanticResolverTestStorage) LockSemanticModelsForUpdate(ctx context.Context, modelIDs []int64) ([]*tenant.SemanticModelRecord, error) {
	out := make([]*tenant.SemanticModelRecord, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		record, err := s.GetSemanticModelForUpdate(ctx, modelID)
		if err != nil {
			if errors.Is(err, tenant.ErrSemanticModelNotFound) {
				continue
			}
			return nil, err
		}
		if record != nil {
			out = append(out, record)
		}
	}
	return out, nil
}

func (s *semanticResolverTestStorage) ListSemanticModels(context.Context, ...tenant.ListOption) ([]*tenant.SemanticModelRecord, int64, error) {
	s.listCalls++
	if s.err != nil {
		return nil, 0, s.err
	}
	if s.models != nil {
		return s.models, int64(len(s.models)), nil
	}
	if s.model != nil {
		return []*tenant.SemanticModelRecord{s.model}, 1, nil
	}
	return nil, 0, nil
}

func (s *semanticResolverTestStorage) ListSemanticModelTags(context.Context, ...tenant.ListOption) ([]tenant.SemanticModelTagStat, error) {
	return nil, nil
}

func (s *semanticResolverTestStorage) UpdateSemanticModel(context.Context, *tenant.SemanticModelRecord) error {
	return errors.New("not implemented")
}

func (s *semanticResolverTestStorage) DeleteSemanticModel(context.Context, int64) error {
	return errors.New("not implemented")
}

func (s *semanticResolverTestStorage) CreateSemanticEntry(context.Context, *tenant.SemanticEntryRecord) (*tenant.SemanticEntryRecord, error) {
	return nil, errors.New("not implemented")
}

func (s *semanticResolverTestStorage) GetSemanticEntry(context.Context, int64, int64) (*tenant.SemanticEntryRecord, error) {
	return nil, errors.New("not implemented")
}

func (s *semanticResolverTestStorage) ListSemanticEntries(context.Context, int64, string, ...tenant.ListOption) ([]*tenant.SemanticEntryRecord, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (s *semanticResolverTestStorage) UpdateSemanticEntry(context.Context, *tenant.SemanticEntryRecord) error {
	return errors.New("not implemented")
}

func (s *semanticResolverTestStorage) DeleteSemanticEntry(context.Context, int64, int64) error {
	return errors.New("not implemented")
}

func TestSemanticKnowledgeBaseResolverProjectsSemanticModel(t *testing.T) {
	tables, _ := json.Marshal([]map[string]any{{
		"db_name":     "sales",
		"table_names": []string{"orders"},
		"parents":     []string{"catalog_1"},
	}})
	files, _ := json.Marshal(map[string]any{
		"file_ids": []string{"file_1"},
		"volumes": []map[string]any{{
			"volume_id": "vol_1",
			"path":      []string{"docs", "policy.md"},
		}},
	})
	resolver := NewSemanticKnowledgeBaseResolver(semanticResolverTestPool{}, &semanticResolverTestStorage{
		model: &tenant.SemanticModelRecord{
			ID:           42,
			Name:         "Sales Knowledge",
			Description:  "Sales semantic model",
			Tables:       tables,
			Files:        files,
			TableSetHash: "hash_1",
			CreatedBy:    "user_1",
			UpdatedBy:    "user_2",
			CreatedAt:    100,
			UpdatedAt:    200,
		},
	})

	kb, err := resolver.ResolveKnowledgeBase(context.Background(), "ws_1", "42")
	if err != nil {
		t.Fatalf("ResolveKnowledgeBase() error = %v", err)
	}
	if kb.ID != "42" || kb.Name != "Sales Knowledge" || kb.Status != KnowledgeBaseStatusActive {
		t.Fatalf("knowledge base = %+v", kb)
	}
	if kb.Metadata["semantic_model_id"] != int64(42) || kb.Metadata["resource_kind"] != "semantic_model" {
		t.Fatalf("metadata = %+v", kb.Metadata)
	}
	if _, ok := kb.Metadata[KnowledgeBaseMetadataRuntimeToolRefsKey]; ok {
		t.Fatalf("metadata[%q] = %+v, want absent", KnowledgeBaseMetadataRuntimeToolRefsKey, kb.Metadata[KnowledgeBaseMetadataRuntimeToolRefsKey])
	}
	if len(kb.CatalogAssetRefs) != 3 {
		t.Fatalf("asset refs = %+v", kb.CatalogAssetRefs)
	}
	if kb.CatalogAssetRefs[0].Type != "table" || kb.CatalogAssetRefs[0].ID != "sales.orders" {
		t.Fatalf("table ref = %+v", kb.CatalogAssetRefs[0])
	}
	if kb.LastIndexedAt == nil || !kb.LastIndexedAt.Equal(time.Unix(200, 0).UTC()) {
		t.Fatalf("last indexed at = %v", kb.LastIndexedAt)
	}
}

func TestSemanticKnowledgeBaseResolverResolvesModelsInOneBatch(t *testing.T) {
	transactionManagerCalls := 0
	storage := &semanticResolverTestStorage{
		models: []*tenant.SemanticModelRecord{
			{ID: 41, Name: "Knowledge 41", CreatedAt: 100, UpdatedAt: 200},
			{ID: 42, Name: "Knowledge 42", CreatedAt: 100, UpdatedAt: 200},
		},
	}
	resolver := NewSemanticKnowledgeBaseResolver(
		semanticResolverTestPool{transactionManagerCalls: &transactionManagerCalls},
		storage,
	)

	items, err := resolver.ResolveKnowledgeBases(context.Background(), "ws_1", []string{"41", "42", "missing"})
	if err != nil {
		t.Fatalf("ResolveKnowledgeBases() error = %v", err)
	}
	if len(items) != 2 || items["41"].Name != "Knowledge 41" || items["42"].Name != "Knowledge 42" {
		t.Fatalf("items = %+v", items)
	}
	if transactionManagerCalls != 1 || storage.listCalls != 1 || storage.getCalls != 0 {
		t.Fatalf("transaction/list/get calls = %d/%d/%d, want 1/1/0", transactionManagerCalls, storage.listCalls, storage.getCalls)
	}
}

func TestSemanticKnowledgeBaseResolverInjectsConfiguredRuntimeToolRefs(t *testing.T) {
	resolver := NewSemanticKnowledgeBaseResolver(
		semanticResolverTestPool{},
		&semanticResolverTestStorage{
			model: &tenant.SemanticModelRecord{
				ID:        42,
				Name:      "Sales Knowledge",
				CreatedAt: 100,
				UpdatedAt: 200,
			},
		},
		WithSemanticKnowledgeBaseRuntimeToolRefs(DefaultKnowledgeBaseRuntimeToolRefs(systemResourceWorkspaceID)),
	)

	kb, err := resolver.ResolveKnowledgeBase(context.Background(), "ws_1", "42")
	if err != nil {
		t.Fatalf("ResolveKnowledgeBase() error = %v", err)
	}
	refs, err := KnowledgeBaseRuntimeToolRefs(*kb)
	if err != nil {
		t.Fatalf("KnowledgeBaseRuntimeToolRefs() error = %v", err)
	}
	if !agentBindingRefsContainID(refs, agenttools.ToolKindSearchRAGChunks) ||
		!agentBindingRefsContainID(refs, agenttools.ToolKindQuerySQL) {
		t.Fatalf("runtime tool refs = %+v", refs)
	}
	if agentBindingRefsContainID(refs, agenttools.ToolKindComputeResultTable) {
		t.Fatalf("runtime tool refs still expose compute_result_table: %+v", refs)
	}
	for _, ref := range refs {
		if ref.WorkspaceID != systemResourceWorkspaceID {
			t.Fatalf("runtime tool ref workspace = %+v", ref)
		}
	}
}

func TestSemanticKnowledgeBaseResolverAppliesDocumentGovernance(t *testing.T) {
	now := time.Now().Unix()
	files, _ := json.Marshal(map[string]any{
		"file_ids": []string{"file_enabled", "file_disabled", "file_forced", "file_expired", "file_disabled_forced", "file_unmanaged"},
	})
	rows := sqlmock.NewRows([]string{"model_id", "source_id", "kb_file_id", "db_name", "table_name", "source_table_id", "kb_table_id", "enabled", "expires_at", "force_enabled_after_expiry", "tags"}).
		AddRow(42, "source_enabled", "file_enabled", "", "", "", "", true, nil, false, `["policy","finance"]`).
		AddRow(42, "source_disabled", "file_disabled", "", "", "", "", false, nil, false, `["disabled"]`).
		AddRow(42, "source_forced", "file_forced", "", "", "", "", true, now-60, true, `["forced"]`).
		AddRow(42, "source_expired", "file_expired", "", "", "", "", true, now-60, false, `["expired"]`).
		AddRow(42, "source_disabled_forced", "file_disabled_forced", "", "", "", "", false, now-60, true, `["disabled-forced"]`)
	resolver := NewSemanticKnowledgeBaseResolver(semanticResolverTestPool{governanceRows: rows}, &semanticResolverTestStorage{
		model: &tenant.SemanticModelRecord{
			ID:        42,
			Name:      "Governed Knowledge",
			Files:     files,
			CreatedAt: 100,
			UpdatedAt: 200,
		},
	})

	kb, err := resolver.ResolveKnowledgeBase(context.Background(), "ws_1", "42")
	if err != nil {
		t.Fatalf("ResolveKnowledgeBase() error = %v", err)
	}
	got := map[string]KnowledgeCatalogAssetRef{}
	for _, ref := range kb.CatalogAssetRefs {
		got[ref.ID] = ref
	}
	for _, id := range []string{"file_disabled", "file_expired", "file_disabled_forced"} {
		if _, ok := got[id]; ok {
			t.Fatalf("governed inactive file %s was projected: %+v", id, kb.CatalogAssetRefs)
		}
	}
	for _, id := range []string{"file_enabled", "file_forced", "file_unmanaged"} {
		if _, ok := got[id]; !ok {
			t.Fatalf("file %s missing from projected refs: %+v", id, kb.CatalogAssetRefs)
		}
	}
	if got["file_enabled"].Config["source_row_id"] != "source_enabled" {
		t.Fatalf("enabled file config = %+v", got["file_enabled"].Config)
	}
	tags, ok := got["file_enabled"].Config["source_tags"].([]string)
	if !ok || len(tags) != 2 || tags[0] != "policy" || tags[1] != "finance" {
		t.Fatalf("enabled file tags = %#v", got["file_enabled"].Config["source_tags"])
	}
}

func TestSemanticKnowledgeBaseResolverAppliesTableGovernance(t *testing.T) {
	now := time.Now().Unix()
	tables, _ := json.Marshal([]map[string]any{
		{
			"db_name":     "sales",
			"table_names": []string{"orders", "customers", "expired_orders", "forced_orders"},
		},
		{
			"db_name":     "support",
			"table_names": []string{"orders"},
		},
	})
	rows := sqlmock.NewRows([]string{"model_id", "source_id", "kb_file_id", "db_name", "table_name", "source_table_id", "kb_table_id", "enabled", "expires_at", "force_enabled_after_expiry", "tags"}).
		AddRow(42, "source_sales_orders", "", "sales", "orders", "src_orders", "kb_orders", false, nil, true, `["ignored"]`).
		AddRow(42, "source_sales_customers", "", "sales", "customers", "src_customers", "kb_customers", true, nil, false, nil).
		AddRow(42, "source_sales_expired", "", "sales", "expired_orders", "src_expired", "kb_expired", true, now-60, false, nil).
		AddRow(42, "source_sales_forced", "", "sales", "forced_orders", "src_forced", "kb_forced", true, now-60, true, nil)
	resolver := NewSemanticKnowledgeBaseResolver(semanticResolverTestPool{governanceRows: rows}, &semanticResolverTestStorage{
		model: &tenant.SemanticModelRecord{
			ID:        42,
			Name:      "Governed Tables",
			Tables:    tables,
			CreatedAt: 100,
			UpdatedAt: 200,
		},
	})

	kb, err := resolver.ResolveKnowledgeBase(context.Background(), "ws_1", "42")
	if err != nil {
		t.Fatalf("ResolveKnowledgeBase() error = %v", err)
	}
	got := map[string]KnowledgeCatalogAssetRef{}
	for _, ref := range kb.CatalogAssetRefs {
		got[ref.ID] = ref
	}
	if _, ok := got["sales.orders"]; ok {
		t.Fatalf("disabled table was projected: %+v", kb.CatalogAssetRefs)
	}
	if _, ok := got["sales.expired_orders"]; ok {
		t.Fatalf("expired table was projected: %+v", kb.CatalogAssetRefs)
	}
	for _, id := range []string{"sales.customers", "sales.forced_orders", "support.orders"} {
		if _, ok := got[id]; !ok {
			t.Fatalf("table %s missing from projected refs: %+v", id, kb.CatalogAssetRefs)
		}
	}
}

func TestSemanticKnowledgeBaseResolverMapsMissingModel(t *testing.T) {
	resolver := NewSemanticKnowledgeBaseResolver(semanticResolverTestPool{}, &semanticResolverTestStorage{err: tenant.ErrSemanticModelNotFound})

	_, err := resolver.ResolveKnowledgeBase(context.Background(), "ws_1", "42")
	if !errors.Is(err, ErrKnowledgeBaseNotFound) {
		t.Fatalf("error = %v, want ErrKnowledgeBaseNotFound", err)
	}
}

func TestSemanticKnowledgeBaseResolverListsSemanticModelsByExactName(t *testing.T) {
	tables, _ := json.Marshal([]map[string]any{{
		"db_name":     "tpcc",
		"table_names": []string{"warehouse"},
	}})
	resolver := NewSemanticKnowledgeBaseResolver(semanticResolverTestPool{}, &semanticResolverTestStorage{
		models: []*tenant.SemanticModelRecord{
			{
				ID:           41,
				Name:         "TPCC Warehouse Archive",
				Description:  "Different semantic model",
				Tables:       tables,
				TableSetHash: "hash_archive",
				CreatedAt:    100,
				UpdatedAt:    200,
			},
			{
				ID:           42,
				Name:         "TPCC 10 Warehouse",
				Description:  "TPCC semantic model",
				Tables:       tables,
				TableSetHash: "hash_tpcc",
				CreatedAt:    100,
				UpdatedAt:    200,
			},
		},
	})

	items, total, err := resolver.ListKnowledgeBases(context.Background(), KnowledgeBaseListFilter{
		WorkspaceID: "ws_1",
		Name:        "TPCC 10 Warehouse",
		Limit:       2,
	})
	if err != nil {
		t.Fatalf("ListKnowledgeBases() error = %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("items=%+v total=%d, want one exact match", items, total)
	}
	if items[0].ID != "42" || items[0].Name != "TPCC 10 Warehouse" || items[0].Status != KnowledgeBaseStatusActive {
		t.Fatalf("knowledge base = %+v", items[0])
	}
	if len(items[0].CatalogAssetRefs) != 1 || items[0].CatalogAssetRefs[0].ID != "tpcc.warehouse" {
		t.Fatalf("asset refs = %+v", items[0].CatalogAssetRefs)
	}
}

func TestSemanticKnowledgeBaseResolverListsModelsWithOneGovernanceQuery(t *testing.T) {
	rows := sqlmock.NewRows([]string{"model_id", "source_id", "kb_file_id", "db_name", "table_name", "source_table_id", "kb_table_id", "enabled", "expires_at", "force_enabled_after_expiry", "tags"}).
		AddRow(41, "source_41", "file_41", "", "", "", "", true, nil, false, nil).
		AddRow(42, "source_42", "file_42", "", "", "", "", true, nil, false, nil)
	resolver := NewSemanticKnowledgeBaseResolver(semanticResolverTestPool{governanceRows: rows}, &semanticResolverTestStorage{
		models: []*tenant.SemanticModelRecord{
			{ID: 41, Name: "Knowledge 41", Files: json.RawMessage(`{"file_ids":["file_41"]}`), CreatedAt: 100, UpdatedAt: 200},
			{ID: 42, Name: "Knowledge 42", Files: json.RawMessage(`{"file_ids":["file_42"]}`), CreatedAt: 100, UpdatedAt: 200},
		},
	})

	items, total, err := resolver.ListKnowledgeBases(context.Background(), KnowledgeBaseListFilter{WorkspaceID: "ws_1", Limit: 2})
	if err != nil {
		t.Fatalf("ListKnowledgeBases() error = %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("items=%+v total=%d, want two models", items, total)
	}
	for _, item := range items {
		if len(item.CatalogAssetRefs) != 1 || item.CatalogAssetRefs[0].ID != "file_"+item.ID {
			t.Fatalf("knowledge base = %+v", item)
		}
	}
}

func TestAgentPackageKnowledgeBaseRefsResolveSemanticKnowledgeBase(t *testing.T) {
	resolver := NewSemanticKnowledgeBaseResolver(
		semanticResolverTestPool{},
		&semanticResolverTestStorage{
			model: &tenant.SemanticModelRecord{
				ID:        42,
				Name:      "xdcvf",
				CreatedAt: 100,
				UpdatedAt: 200,
			},
		},
	)

	ids, err := resolveAgentPackageKnowledgeBaseRefIDs(context.Background(), "ws_1", []AgentPackageNamedResourceRef{{Name: "xdcvf"}}, resolver)
	if err != nil {
		t.Fatalf("resolveAgentPackageKnowledgeBaseRefIDs() error = %v", err)
	}
	if len(ids) != 1 || ids[0] != "42" {
		t.Fatalf("ids = %+v, want [42]", ids)
	}
}

func TestAgentInstanceResolverResolvesAstraPackageSemanticKnowledgeBaseRef(t *testing.T) {
	ctx := context.Background()
	tables, _ := json.Marshal([]map[string]any{{
		"db_name":     "sales",
		"table_names": []string{"orders"},
	}})
	store := newTestSystemOverlayStore(t)
	service := NewAgentService(store).WithKnowledgeBaseResolver(NewSemanticKnowledgeBaseResolver(
		semanticResolverTestPool{},
		&semanticResolverTestStorage{
			model: &tenant.SemanticModelRecord{
				ID:           42,
				Name:         "Sales Knowledge",
				Description:  "Sales semantic model",
				Tables:       tables,
				TableSetHash: "hash_1",
				CreatedAt:    100,
				UpdatedAt:    200,
			},
		},
	))
	if _, err := service.CreateAgent(ctx, AgentCreateInput{
		ID:          "ag_pkg",
		WorkspaceID: "ws_1",
		Name:        "Sales Package Agent",
		Status:      AgentStatusActive,
		Runtime: AgentRuntimeTarget{
			Provider: RuntimeProviderAstra,
			Profile:  RuntimeProviderProfileDefault,
		},
		Model:  AgentModelConfig{DefaultModel: "qwen3.6-plus"},
		UserID: "user_1",
	}); err != nil {
		t.Fatalf("CreateAgent() error = %v", err)
	}

	versionStore := NewInMemoryAgentVersionStore()
	versionService := NewAgentVersionService(versionStore)
	if _, err := versionService.CreateVersion(ctx, AgentVersionCreateInput{
		WorkspaceID:  "ws_1",
		AgentID:      "ag_pkg",
		Version:      "1.0.0",
		SourceDigest: testAgentVersionSourceDigest(),
		Status:       AgentVersionStatusRunnable,
		Manifest: AgentPackageManifest{
			SchemaVersion: 1,
			ID:            "ag_pkg",
			Version:       "1.0.0",
			Name:          "Sales Package Agent",
			Model:         "qwen3.6-plus",
			Instruction:   AgentPackagePathRef{Path: "prompts/role.md"},
			MOIResourceRefs: AgentPackageMOIResourceRefs{
				KnowledgeBases: []AgentPackageNamedResourceRef{{Name: "Sales Knowledge"}},
			},
		},
		SourceFiles: map[string][]byte{
			"prompts/role.md": []byte("Use sales knowledge."),
		},
	}); err != nil {
		t.Fatalf("CreateVersion() error = %v", err)
	}
	if _, err := versionService.SetDefaultVersion(ctx, "ws_1", "ag_pkg", "1.0.0"); err != nil {
		t.Fatalf("SetDefaultVersion() error = %v", err)
	}

	descriptor, err := NewAgentInstanceResolver(service, WithAgentInstanceAgentVersionStore(versionStore)).DescribeAgent(ctx, "ws_1", "user_1", "ag_pkg")
	if err != nil {
		t.Fatalf("DescribeAgent() error = %v", err)
	}
	if len(descriptor.KnowledgeBases) != 1 {
		t.Fatalf("knowledge snapshots = %+v", descriptor.KnowledgeBases)
	}
	snapshot := descriptor.KnowledgeBases[0]
	if snapshot.ID != "42" || snapshot.Name != "Sales Knowledge" {
		t.Fatalf("knowledge snapshot = %+v", snapshot)
	}
	if snapshot.Metadata["resource_kind"] != "semantic_model" || snapshot.Metadata["semantic_model_id"] != int64(42) {
		t.Fatalf("knowledge metadata = %+v", snapshot.Metadata)
	}
	if len(snapshot.CatalogAssetRefs) != 1 || snapshot.CatalogAssetRefs[0]["id"] != "sales.orders" {
		t.Fatalf("knowledge asset refs = %+v", snapshot.CatalogAssetRefs)
	}
}
