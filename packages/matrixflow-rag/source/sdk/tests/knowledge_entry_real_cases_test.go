package tests

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	sdk "github.com/matrixorigin/matrixflow/sdk/go-sdk"
	"github.com/matrixorigin/matrixflow/sdk/tests/framework"
)

// TestProductSDKKnowledgeEntryLifecycleRealCases verifies semantic entries
// against a model bound to a real table.
func TestProductSDKKnowledgeEntryLifecycleRealCases(t *testing.T) {

	framework.RunProductSDKTests(t, func(env *framework.TestEnv) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		fixture := env.NewKnowledgeFixture(ctx, t, "knowledge-entry")
		entry := sdk.SemanticEntryInput{Kind: "metric", Key: "total_rows", Tables: []string{fixture.Table.Name()}, Spec: map[string]any{"expr": "COUNT(*)", "unit": "rows"}}
		goEntry, created, err := fixture.Model.CreateEntry(ctx, entry)
		if err != nil {
			t.Fatalf("create Go semantic entry: %v", err)
		}
		if created.GetId() == 0 || created.GetKind() != entry.Kind || created.GetKey() != entry.Key {
			t.Fatalf("created Go semantic entry is incomplete: %#v", created)
		}
		updatedEntry := entry
		updatedEntry.Spec = map[string]any{"expr": "COUNT(id)", "unit": "rows"}
		updated, err := goEntry.Update(ctx, updatedEntry)
		if err != nil || !updated.GetUpdated() {
			t.Fatalf("update Go semantic entry: result=%#v err=%v", updated, err)
		}
		listed, err := fixture.Model.Entries(ctx)
		if err != nil || !containsSemanticEntry(listed.GetItems(), created.GetId(), entry.Key) {
			t.Fatalf("list Go semantic entries: result=%#v err=%v", listed, err)
		}
		deleted, err := goEntry.Delete(ctx)
		if err != nil || !deleted.GetDeleted() {
			t.Fatalf("delete Go semantic entry: result=%#v err=%v", deleted, err)
		}
		imported, err := fixture.Model.Import(ctx, []sdk.SemanticEntryInput{
			{Kind: "glossary", Key: env.TestID + "_row_count", Spec: map[string]any{"term": "row count", "definition": "number of rows"}},
			{Kind: "glossary", Key: env.TestID + "_row_name", Spec: map[string]any{"term": "row name", "definition": "display name for a row"}},
		})
		if err != nil || imported.GetImported() < 2 {
			t.Fatalf("import Go semantic entry: result=%#v err=%v", imported, err)
		}
		exported, exportErr := fixture.Model.Export(ctx)
		if exportErr != nil || exported.GetModel().GetId() == 0 || len(exported.GetEntries()) < 2 {
			t.Fatalf("export Go semantic entries: result=%#v err=%v", exported, exportErr)
		}
		for _, importedEntry := range exported.GetEntries() {
			entryHandle, err := fixture.Model.Entry(strconv.FormatInt(importedEntry.GetId(), 10))
			if err != nil {
				t.Fatalf("open imported Go semantic entry %d: %v", importedEntry.GetId(), err)
			}
			deleted, err := entryHandle.Delete(ctx)
			if err != nil || !deleted.GetDeleted() {
				t.Fatalf("delete imported Go semantic entry %d: result=%#v err=%v", importedEntry.GetId(), deleted, err)
			}
		}
		remaining, err := fixture.Model.Entries(ctx)
		if err != nil || len(remaining.GetItems()) != 0 {
			t.Fatalf("prepare empty semantic model for Python import: result=%#v err=%v", remaining, err)
		}

		script := `
import sys
import moi_product_sdk as sdk
endpoint, token, workspace_id, model_id, table_name = sys.argv[1:]
client = sdk.new_with_personal_access_token(endpoint, token)
knowledge = client.knowledge(workspace_id)
entry = sdk.SemanticEntryInput("metric", "python_total_rows", {"expr": "COUNT(*)", "unit": "rows"}, [table_name])
created = knowledge.create_entry(model_id, entry)
assert created.id and created.kind == "metric" and created.key == "python_total_rows"
updated = knowledge.update_entry(model_id, str(created.id), sdk.SemanticEntryInput("metric", "python_total_rows", {"expr": "COUNT(id)", "unit": "rows"}, [table_name]))
assert updated.updated
listed = knowledge.list_entries(model_id)
assert any(item.id == created.id and item.key == "python_total_rows" for item in listed.items)
deleted = knowledge.delete_entry(model_id, str(created.id))
assert deleted.deleted
imported = knowledge.import_entries(model_id, [
    sdk.SemanticEntryInput("glossary", "python_row_count", {"term": "python row count", "definition": "number of rows"}),
    sdk.SemanticEntryInput("glossary", "python_row_name", {"term": "python row name", "definition": "display name for a row"}),
])
assert imported.imported >= 2
exported = knowledge.export(model_id)
assert exported.model.id == int(model_id) and len(exported.entries) >= 2
`
		out, err := env.RunPythonProductSDK(ctx, script, fixture.Catalog.Workspace.ID(), fixture.Model.ID(), fixture.Table.Name())
		if err != nil && !errors.Is(err, framework.ErrPythonE2EDisabled) {
			t.Fatalf("run Python semantic entry lifecycle: %v\n%s", err, string(out))
		}
		for _, route := range []struct{ method, path string }{
			{"GET", "/newmoi/semantic-models/:model_id/entries"},
			{"POST", "/newmoi/semantic-models/:model_id/entries"},
			{"PUT", "/newmoi/semantic-models/:model_id/entries/:entry_id"},
			{"DELETE", "/newmoi/semantic-models/:model_id/entries/:entry_id"},
		} {
			env.RequireRealResponse(t, route.method, route.path)
			env.RequirePythonSDKRealResponse(t, route.method, route.path)
		}
		env.RequireRealResponse(t, "POST", "/newmoi/semantic-models/:model_id/import")
		env.RequireRealResponse(t, "GET", "/newmoi/semantic-models/:model_id/export")
	})
}

func containsSemanticEntry(entries []*sdk.SemanticEntry, id int64, key string) bool {
	for _, entry := range entries {
		if entry.GetId() == id && entry.GetKey() == key {
			return true
		}
	}
	return false
}
