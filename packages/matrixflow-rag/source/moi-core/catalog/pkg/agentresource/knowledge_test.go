package agentresource

import (
	"context"
	"errors"
	"testing"
	"time"

	agenttools "github.com/matrixflow/moi-core/agent-tools"
)

func TestKnowledgeBaseServiceCreateListAndUpdate(t *testing.T) {
	store := NewInMemoryAgentStore()
	service := NewKnowledgeBaseService(
		store,
		WithKnowledgeBaseDefaultRuntimeToolRefs(DefaultKnowledgeBaseRuntimeToolRefs(systemResourceWorkspaceID)),
	)
	service.now = func() time.Time { return time.Unix(100, 0) }
	service.newID = func() string { return "kb_1" }

	kb, err := service.CreateKnowledgeBase(context.Background(), KnowledgeBaseCreateInput{
		WorkspaceID: "ws_1",
		Name:        "Supplier catalog",
		Description: "Approved supplier documents",
		Status:      KnowledgeBaseStatusActive,
		SourceType:  KnowledgeBaseSourceCatalogResource,
		CatalogAssetRefs: []KnowledgeCatalogAssetRef{
			{Type: "catalog", ID: "cat_1", Role: "source"},
			{Type: "catalog", ID: "cat_1", Role: "source"},
		},
		Tags:        []string{"supplier", "supplier", "bid"},
		Visibility:  KnowledgeBaseVisibilityWorkspace,
		IndexStatus: KnowledgeBaseIndexStatusReady,
		UserID:      "user_1",
	})
	if err != nil {
		t.Fatalf("CreateKnowledgeBase() error = %v", err)
	}
	if kb.ID != "kb_1" || kb.Version != 1 || kb.OwnerUserID != "user_1" || len(kb.Tags) != 3 || len(kb.CatalogAssetRefs) != 2 {
		t.Fatalf("kb = %+v", kb)
	}
	runtimeToolRefs, err := KnowledgeBaseRuntimeToolRefs(*kb)
	if err != nil {
		t.Fatalf("KnowledgeBaseRuntimeToolRefs() error = %v", err)
	}
	if !agentBindingRefsContainID(runtimeToolRefs, agenttools.ToolKindSearchRAGChunks) ||
		!agentBindingRefsContainID(runtimeToolRefs, agenttools.ToolKindQuerySQL) ||
		!agentBindingRefsContainID(runtimeToolRefs, agenttools.ToolKindReadFile) ||
		!agentBindingRefsContainID(runtimeToolRefs, agenttools.ToolKindSelectFinalSources) {
		t.Fatalf("runtime tool refs = %+v", runtimeToolRefs)
	}
	if agentBindingRefsContainID(runtimeToolRefs, agenttools.ToolKindComputeResultTable) {
		t.Fatalf("runtime tool refs still expose compute_result_table: %+v", runtimeToolRefs)
	}
	if agentBindingRefsContainID(runtimeToolRefs, agenttools.ToolKindSubmitFinalAnswer) {
		t.Fatalf("runtime tool refs exposed legacy submit final answer tool: %+v", runtimeToolRefs)
	}

	items, total, err := service.ListKnowledgeBases(context.Background(), KnowledgeBaseListFilter{
		WorkspaceID: "ws_1",
		Status:      KnowledgeBaseStatusActive,
		SourceType:  KnowledgeBaseSourceCatalogResource,
		Visibility:  KnowledgeBaseVisibilityWorkspace,
		IndexStatus: KnowledgeBaseIndexStatusReady,
		OwnerUserID: "user_1",
		Query:       "supplier",
	})
	if err != nil {
		t.Fatalf("ListKnowledgeBases() error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != "kb_1" {
		t.Fatalf("total=%d items=%+v", total, items)
	}

	disabled := KnowledgeBaseStatusDisabled
	updated, err := service.UpdateKnowledgeBase(context.Background(), "ws_1", "kb_1", KnowledgeBaseUpdateInput{
		Status: &disabled,
		UserID: "user_2",
	})
	if err != nil {
		t.Fatalf("UpdateKnowledgeBase() error = %v", err)
	}
	if updated.Status != KnowledgeBaseStatusDisabled || updated.Version != 2 || updated.UpdatedBy != "user_2" {
		t.Fatalf("updated = %+v", updated)
	}
	updatedRuntimeToolRefs, err := KnowledgeBaseRuntimeToolRefs(*updated)
	if err != nil {
		t.Fatalf("updated KnowledgeBaseRuntimeToolRefs() error = %v", err)
	}
	if !agentBindingRefsContainID(updatedRuntimeToolRefs, agenttools.ToolKindSearchRAGChunks) {
		t.Fatalf("updated runtime tool refs = %+v", updatedRuntimeToolRefs)
	}

	if err := service.DeleteKnowledgeBase(context.Background(), "ws_1", "kb_1"); err != nil {
		t.Fatalf("DeleteKnowledgeBase() error = %v", err)
	}
	if _, err := service.GetKnowledgeBase(context.Background(), "ws_1", "kb_1"); !errors.Is(err, ErrKnowledgeBaseNotFound) {
		t.Fatalf("GetKnowledgeBase() error = %v, want ErrKnowledgeBaseNotFound", err)
	}
}

func TestKnowledgeBaseServiceDoesNotInjectRuntimeToolRefsWithoutDefault(t *testing.T) {
	store := NewInMemoryAgentStore()
	service := NewKnowledgeBaseService(store)
	service.now = func() time.Time { return time.Unix(100, 0) }
	service.newID = func() string { return "kb_1" }

	kb, err := service.CreateKnowledgeBase(context.Background(), KnowledgeBaseCreateInput{
		WorkspaceID: "ws_1",
		Name:        "Supplier catalog",
		Status:      KnowledgeBaseStatusActive,
		SourceType:  KnowledgeBaseSourceCatalogResource,
		UserID:      "user_1",
	})
	if err != nil {
		t.Fatalf("CreateKnowledgeBase() error = %v", err)
	}
	if _, ok := kb.Metadata[KnowledgeBaseMetadataRuntimeToolRefsKey]; ok {
		t.Fatalf("metadata[%q] = %+v, want absent", KnowledgeBaseMetadataRuntimeToolRefsKey, kb.Metadata[KnowledgeBaseMetadataRuntimeToolRefsKey])
	}
	runtimeToolRefs, err := KnowledgeBaseRuntimeToolRefs(*kb)
	if err != nil {
		t.Fatalf("KnowledgeBaseRuntimeToolRefs() error = %v", err)
	}
	if len(runtimeToolRefs) != 0 {
		t.Fatalf("runtime tool refs = %+v, want none", runtimeToolRefs)
	}
}

func agentBindingRefsContainID(refs []AgentBindingResourceRef, id string) bool {
	for _, ref := range refs {
		if ref.ID == id {
			return true
		}
	}
	return false
}

func TestKnowledgeBaseValidationRejectsSecretMetadata(t *testing.T) {
	_, err := NormalizeKnowledgeBase(KnowledgeBase{
		ID:          "kb_1",
		WorkspaceID: "ws_1",
		Name:        "Bad KB",
		Status:      KnowledgeBaseStatusActive,
		SourceType:  KnowledgeBaseSourceCatalogResource,
		CatalogAssetRefs: []KnowledgeCatalogAssetRef{
			{Type: "catalog", ID: "cat_1", Config: map[string]any{"api_token": "secret"}},
		},
	}, time.Unix(100, 0))
	if !errors.Is(err, ErrInvalidKnowledgeBase) {
		t.Fatalf("NormalizeKnowledgeBase() error = %v, want ErrInvalidKnowledgeBase", err)
	}
}

func TestKnowledgeBaseValidationRejectsSecretAssetURI(t *testing.T) {
	_, err := NormalizeKnowledgeBase(KnowledgeBase{
		ID:          "kb_1",
		WorkspaceID: "ws_1",
		Name:        "Bad KB",
		Status:      KnowledgeBaseStatusActive,
		SourceType:  KnowledgeBaseSourceCatalogResource,
		CatalogAssetRefs: []KnowledgeCatalogAssetRef{
			{Type: "file", URI: "https://files.example.test/report.pdf?access_token=secret"},
		},
	}, time.Unix(100, 0))
	if !errors.Is(err, ErrInvalidKnowledgeBase) {
		t.Fatalf("NormalizeKnowledgeBase() error = %v, want ErrInvalidKnowledgeBase", err)
	}
}

func TestKnowledgeBaseValidationRejectsInvalidAssetRef(t *testing.T) {
	_, err := NormalizeKnowledgeBase(KnowledgeBase{
		ID:          "kb_1",
		WorkspaceID: "ws_1",
		Name:        "Bad KB",
		Status:      KnowledgeBaseStatusActive,
		SourceType:  KnowledgeBaseSourceCatalogResource,
		CatalogAssetRefs: []KnowledgeCatalogAssetRef{
			{Type: "catalog"},
		},
	}, time.Unix(100, 0))
	if !errors.Is(err, ErrInvalidKnowledgeBase) {
		t.Fatalf("NormalizeKnowledgeBase() error = %v, want ErrInvalidKnowledgeBase", err)
	}
}

func TestKnowledgeBaseValidationRejectsPaddedRuntimeRefs(t *testing.T) {
	for name, mutate := range map[string]func(*KnowledgeBase){
		"default_retrieval_profile_ref": func(kb *KnowledgeBase) {
			kb.DefaultRetrievalProfileRef = " retrieval_1 "
		},
		"asset_type": func(kb *KnowledgeBase) {
			kb.CatalogAssetRefs = []KnowledgeCatalogAssetRef{{Type: " catalog ", ID: "cat_1"}}
		},
		"asset_id": func(kb *KnowledgeBase) {
			kb.CatalogAssetRefs = []KnowledgeCatalogAssetRef{{Type: "catalog", ID: " cat_1 "}}
		},
		"asset_uri": func(kb *KnowledgeBase) {
			kb.CatalogAssetRefs = []KnowledgeCatalogAssetRef{{Type: "file", URI: " https://files.example.test/report.pdf "}}
		},
		"asset_version": func(kb *KnowledgeBase) {
			kb.CatalogAssetRefs = []KnowledgeCatalogAssetRef{{Type: "catalog", ID: "cat_1", Version: " v1 "}}
		},
		"asset_role": func(kb *KnowledgeBase) {
			kb.CatalogAssetRefs = []KnowledgeCatalogAssetRef{{Type: "catalog", ID: "cat_1", Role: " source "}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			kb := KnowledgeBase{
				ID:          "kb_1",
				WorkspaceID: "ws_1",
				Name:        "Bad KB",
				Status:      KnowledgeBaseStatusActive,
				SourceType:  KnowledgeBaseSourceCatalogResource,
			}
			mutate(&kb)
			_, err := NormalizeKnowledgeBase(kb, time.Unix(100, 0))
			if !errors.Is(err, ErrInvalidKnowledgeBase) {
				t.Fatalf("NormalizeKnowledgeBase() error = %v, want ErrInvalidKnowledgeBase", err)
			}
		})
	}
}
