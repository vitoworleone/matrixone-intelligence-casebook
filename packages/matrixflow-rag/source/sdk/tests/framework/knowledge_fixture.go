package framework

import (
	"context"
	"strconv"
	"strings"

	sdk "github.com/matrixorigin/matrixflow/sdk/go-sdk"
)

// KnowledgeFixture owns the catalog table and semantic model shared by
// knowledge entry, source, segment, and validation scenarios.
type KnowledgeFixture struct {
	Catalog *CatalogFixture
	Table   *sdk.TableHandle
	Model   *sdk.SemanticModelHandle
}

// KnowledgeSourceFixture owns a semantic model with one persisted inline
// local-file source. Source and segment scenarios can use its model, source
// row, and source file without reproducing the source bootstrap flow.
type KnowledgeSourceFixture struct {
	Catalog *CatalogFixture
	Model   *sdk.SemanticModelHandle
	Source  *sdk.SemanticModelSource
}

// NewKnowledgeFixture creates a model bound to a real table. The parent
// CatalogFixture owns its workspace and removes the complete resource tree,
// including this model, in one product operation.
func (e *TestEnv) NewKnowledgeFixture(ctx context.Context, t interface {
	Helper()
	Cleanup(func())
	Fatalf(format string, args ...any)
}, label string) *KnowledgeFixture {
	t.Helper()
	catalog := e.NewCatalogFixture(ctx, t, label)
	// Shared-workspace smoke reuses one database; table names must be unique per case.
	table, err := catalog.CreateTable(ctx, uniqueSQLName(e.TestID, label, "knowledge_source"))
	if err != nil {
		t.Fatalf("create knowledge source table: %v", err)
	}
	tables := []map[string]any{{"db_name": catalog.Database.Name(), "table_names": []string{table.Name()}, "parents": []any{}}}
	model, created, err := catalog.Workspace.CreateSemanticModel(ctx, e.TestID+"-"+label+"-model", sdk.WithSemanticModelDescription("Product SDK knowledge fixture"), sdk.WithSemanticModelTables(tables))
	if err != nil {
		t.Fatalf("create knowledge fixture model: %v", err)
	}
	if model.ID() == "" || created.GetId() == 0 {
		t.Fatalf("knowledge fixture model has empty identity: %#v", created)
	}
	return &KnowledgeFixture{Catalog: catalog, Table: table, Model: model}
}

// NewKnowledgeSourceFixture creates a knowledge model with one local-file source
// via create-with-sources. That path deploys the KB document-parsing workflow;
// empty Create + first Append no longer does (backend keeps existing workflow
// definitions on append and fails closed when none exist). Workspace cleanup
// removes the full knowledge-base resource tree.
func (e *TestEnv) NewKnowledgeSourceFixture(ctx context.Context, t interface {
	Helper()
	Cleanup(func())
	Fatalf(format string, args ...any)
}, label string) *KnowledgeSourceFixture {
	t.Helper()
	catalog := e.NewCatalogFixture(ctx, t, label)
	knowledge := catalog.Workspace.Knowledge()
	fileName := label + ".txt"
	upload, err := knowledge.UploadLocalFile(ctx, fileName, strings.NewReader("Product SDK knowledge source fixture"))
	if err != nil {
		t.Fatalf("upload knowledge source fixture file: %v", err)
	}
	if upload.GetFileId() == "" {
		t.Fatalf("knowledge source fixture upload returned no file ID: %#v", upload)
	}
	created, err := knowledge.CreateWithSources(
		ctx,
		e.TestID+"-"+label+"-model",
		[]sdk.SemanticModelSourceInput{{
			SourceType: "local_file",
			FileName:   fileName,
			FileID:     upload.GetFileId(),
		}},
		sdk.WithSemanticModelWithSourcesDescription("Product SDK knowledge source fixture"),
	)
	if err != nil {
		t.Fatalf("create knowledge source fixture model with sources: %v", err)
	}
	if created.GetModel() == nil || created.GetModel().GetId() == 0 {
		t.Fatalf("knowledge source fixture model has empty identity: %#v", created)
	}
	model, err := catalog.Workspace.SemanticModel(strconv.FormatInt(created.GetModel().GetId(), 10))
	if err != nil {
		t.Fatalf("bind knowledge source fixture model handle: %v", err)
	}
	for _, source := range created.GetSources() {
		if source.GetRowId() != "" && source.GetSourceFileId() != "" && (source.GetSourceType() == "local_file" || source.GetSourceType() == "file") {
			return &KnowledgeSourceFixture{Catalog: catalog, Model: model, Source: source}
		}
	}
	t.Fatalf("knowledge source fixture has no persisted local-file source: %#v", created)
	return nil
}
