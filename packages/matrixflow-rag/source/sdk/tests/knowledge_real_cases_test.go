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

// TestProductSDKKnowledgeModelLifecycleRealCases exercises model CRUD against
// a test-owned catalog table, the required business input for a knowledge base.
func TestProductSDKKnowledgeModelLifecycleRealCases(t *testing.T) {

	framework.RunProductSDKTests(t, func(env *framework.TestEnv) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		fixture := env.NewCatalogFixture(ctx, t, "knowledge-model")
		table, err := fixture.CreateTable(ctx, env.TestID+"_knowledge_source")
		if err != nil {
			t.Fatalf("create knowledge source table: %v", err)
		}
		tables := []map[string]any{{"db_name": fixture.Database.Name(), "table_names": []string{table.Name()}, "parents": []any{}}}
		knowledge := fixture.Workspace.Knowledge()

		goName := env.TestID + "-knowledge-go"
		created, err := knowledge.Create(ctx, goName, sdk.WithSemanticModelDescription("Go knowledge model"), sdk.WithSemanticModelTables(tables))
		if err != nil {
			t.Fatalf("create Go knowledge model: %v", err)
		}
		modelID := strconv.FormatInt(created.GetId(), 10)
		if created.GetId() == 0 || created.GetName() != goName || created.GetDescription() != "Go knowledge model" {
			t.Fatalf("created Go knowledge model is incomplete: %#v", created)
		}
		got, err := knowledge.Get(ctx, modelID)
		if err != nil || got.GetId() != created.GetId() || got.GetName() != goName {
			t.Fatalf("read Go knowledge model: got=%#v err=%v", got, err)
		}
		updatedName := goName
		updated, err := knowledge.Update(ctx, modelID, updatedName, sdk.WithSemanticModelDescription("updated by Go"), sdk.WithSemanticModelTables(tables))
		if err != nil || !updated.GetUpdated() {
			t.Fatalf("update Go knowledge model: result=%#v err=%v", updated, err)
		}
		listed, err := knowledge.List(ctx)
		if err != nil || !containsSemanticModel(listed.GetItems(), created.GetId(), updatedName) {
			t.Fatalf("list updated Go knowledge model: result=%#v err=%v", listed, err)
		}
		if deleted, err := knowledge.Delete(ctx, modelID); err != nil || !deleted.GetDeleted() {
			t.Fatalf("delete Go knowledge model: result=%#v err=%v", deleted, err)
		}

		script := `
import sys
import moi_product_sdk as sdk
endpoint, token, workspace_id, db_name, table_name, test_id = sys.argv[1:]
client = sdk.new_with_personal_access_token(endpoint, token)
knowledge = client.knowledge(workspace_id)
tables = [{"db_name": db_name, "table_names": [table_name], "parents": []}]
name = test_id + "-knowledge-python"
created = knowledge.create(name, sdk.with_semantic_model_description("Python knowledge model"), sdk.with_semantic_model_tables(tables))
assert created.id and created.name == name
try:
    got = knowledge.get(str(created.id))
    assert got.id == created.id and got.description == "Python knowledge model"
    updated_name = name
    updated = knowledge.update(str(created.id), updated_name, sdk.with_semantic_model_description("updated by Python"), sdk.with_semantic_model_tables(tables))
    assert updated.updated
    listed = knowledge.list()
    assert any(item.id == created.id and item.name == updated_name for item in listed.items)
finally:
    deleted = knowledge.delete(str(created.id))
    assert deleted.deleted
`
		out, err := env.RunPythonProductSDK(ctx, script, fixture.Workspace.ID(), fixture.Database.Name(), table.Name(), env.TestID)
		if err != nil && !errors.Is(err, framework.ErrPythonE2EDisabled) {
			t.Fatalf("run Python knowledge model lifecycle: %v\n%s", err, string(out))
		}
		for _, route := range []struct{ method, path string }{
			{"POST", "/newmoi/semantic-models"},
			{"GET", "/newmoi/semantic-models"},
			{"GET", "/newmoi/semantic-models/:model_id"},
			{"PUT", "/newmoi/semantic-models/:model_id"},
		} {
			env.RequireRealResponse(t, route.method, route.path)
			env.RequirePythonSDKRealResponse(t, route.method, route.path)
		}
		env.RequireRealResponse(t, "DELETE", "/newmoi/semantic-models/:model_id")
	})
}

func containsSemanticModel(models []*sdk.SemanticModelInfo, id int64, name string) bool {
	for _, model := range models {
		if model.GetId() == id && model.GetName() == name {
			return true
		}
	}
	return false
}
