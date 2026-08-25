// Package tests contains integration tests for moi-core.
package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/tests/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// structuredTablesJSON builds the structured tables format required by the API.
// e.g. structuredTablesJSON("orders", "users") → [{"db_name":"","table_names":["orders","users"],"parents":[]}]
func structuredTablesJSON(names ...string) json.RawMessage {
	type entry struct {
		DBName     string   `json:"db_name"`
		TableNames []string `json:"table_names"`
		Parents    []string `json:"parents"`
	}
	data, _ := json.Marshal([]entry{{DBName: "", TableNames: names, Parents: []string{}}})
	return data
}

// TestSemanticModelSDK_CRUDAndValidation tests semantic model CRUD, entry CRUD, validate and export via Go SDK.
//
// Purpose: Verify new Go SDK semantic model interfaces work end-to-end against real APIs.
// Scenario: Create model, update/list/get, create/update/list entry, validate/export, then cleanup.
// Assertions: All operations succeed and returned data matches expected IDs/keys.
//
// **Validates: Requirements KB.V2 Semantic Model SDK Integration**
func TestSemanticModelSDK_CRUDAndValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	framework.RunMOITests(t, func(env *framework.TestEnv) {
		ctx := context.Background()
		env.RequireSharedWorkspace(t)

		client, err := env.GetSharedClient()
		require.NoError(t, err)

		wsID := env.SharedWorkspaceID
		svc := client.SemanticModels(wsID)

		modelName := fmt.Sprintf("test-semantic-model-%s", env.TestID)
		model, err := svc.Create(ctx, &moi.SemanticModelUpsertRequest{
			Name:        modelName,
			Description: "semantic model e2e",
			Tables: structuredTablesJSON("orders", "users"),
		})
		require.NoError(t, err)
		require.NotZero(t, model.ID)

		modelDeleted := false
		defer func() {
			if !modelDeleted {
				_ = svc.Delete(context.Background(), model.ID)
			}
		}()

		listResp, err := svc.List(ctx, moi.WithPageSize(100))
		require.NoError(t, err)
		assert.True(t, semanticModelExistsByID(listResp.Items, model.ID))

		got, err := svc.Get(ctx, model.ID)
		require.NoError(t, err)
		assert.Equal(t, modelName, got.Name)
		assert.JSONEq(t, string(structuredTablesJSON("orders", "users")), string(got.Tables))

		updateResp, err := svc.Update(ctx, model.ID, &moi.SemanticModelUpsertRequest{
			Name:        modelName + "-v2",
			Description: "semantic model e2e updated",
			Tables: structuredTablesJSON("orders", "users"),
		})
		require.NoError(t, err)
		assert.True(t, updateResp.Updated)

		entry, err := svc.CreateEntry(ctx, model.ID, &moi.SemanticEntryUpsertRequest{
			Kind:   "metric",
			Key:    "gmv",
			Tables: []string{"orders"},
			Spec:   json.RawMessage(`{"expr":"sum(price)"}`),
		})
		require.NoError(t, err)
		require.NotZero(t, entry.ID)

		entryDeleted := false
		defer func() {
			if !entryDeleted {
				_ = svc.DeleteEntry(context.Background(), model.ID, entry.ID)
			}
		}()

		entryList, err := svc.ListEntries(ctx, model.ID, "metric", moi.WithPageSize(100))
		require.NoError(t, err)
		assert.True(t, semanticEntryExistsByID(entryList.Items, entry.ID))

		entryUpdateResp, err := svc.UpdateEntry(ctx, model.ID, entry.ID, &moi.SemanticEntryUpsertRequest{
			Kind:   "metric",
			Key:    "gmv_total",
			Tables: []string{"orders"},
			Spec:   json.RawMessage(`{"expr":"sum(price)"}`),
		})
		require.NoError(t, err)
		assert.True(t, entryUpdateResp.Updated)

		validateResp, err := svc.Validate(ctx, model.ID)
		require.NoError(t, err)
		assert.True(t, validateResp.Valid)

		exportResp, err := svc.Export(ctx, model.ID)
		require.NoError(t, err)
		require.NotNil(t, exportResp.Model)
		assert.Equal(t, model.ID, exportResp.Model.ID)
		assert.True(t, semanticEntryKeyExists(exportResp.Entries, "gmv_total"))

		err = svc.DeleteEntry(ctx, model.ID, entry.ID)
		require.NoError(t, err)
		entryDeleted = true

		err = svc.Delete(ctx, model.ID)
		require.NoError(t, err)
		modelDeleted = true
	})
}

// TestSemanticModelSDK_ImportAndExport tests import and export flow via Go SDK.
//
// Purpose: Verify semantic model import API is correctly exposed by Go SDK.
// Scenario: Import a model with entries, then export and validate the imported content.
// Assertions: Imported model can be exported and contains expected entries.
//
// **Validates: Requirements KB.V2 Semantic Model Import/Export**
func TestSemanticModelSDK_ImportAndExport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	framework.RunMOITests(t, func(env *framework.TestEnv) {
		ctx := context.Background()
		env.RequireSharedWorkspace(t)

		client, err := env.GetSharedClient()
		require.NoError(t, err)

		wsID := env.SharedWorkspaceID
		svc := client.SemanticModels(wsID)

		modelName := fmt.Sprintf("test-semantic-import-%s", env.TestID)
		imported, err := svc.Import(ctx, &moi.SemanticModelImportRequest{
			Name:   modelName,
			Tables: structuredTablesJSON("orders"),
			Entries: []moi.SemanticEntryUpsertRequest{
				{
					Kind: "metric",
					Key:  "order_count",
					Spec: json.RawMessage(`{"expr":"count(*)"}`),
				},
			},
		})
		require.NoError(t, err)
		require.NotZero(t, imported.ID)

		defer func() {
			_ = svc.Delete(context.Background(), imported.ID)
		}()

		exported, err := svc.Export(ctx, imported.ID)
		require.NoError(t, err)
		require.NotNil(t, exported.Model)
		assert.Equal(t, imported.ID, exported.Model.ID)
		assert.Equal(t, modelName, exported.Model.Name)
		assert.True(t, semanticEntryKeyExists(exported.Entries, "order_count"))
	})
}

// TestSemanticModelSDK_ImportBatchAndExport tests batch import through the same /import API.
func TestSemanticModelSDK_ImportBatchAndExport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	framework.RunMOITests(t, func(env *framework.TestEnv) {
		ctx := context.Background()
		env.RequireSharedWorkspace(t)

		client, err := env.GetSharedClient()
		require.NoError(t, err)

		wsID := env.SharedWorkspaceID
		svc := client.SemanticModels(wsID)

		nameA := fmt.Sprintf("test-semantic-batch-a-%s", env.TestID)
		nameB := fmt.Sprintf("test-semantic-batch-b-%s", env.TestID)
		batchResp, err := svc.ImportBatch(ctx, []moi.SemanticModelImportRequest{
			{
				Name:   nameA,
				Tables: structuredTablesJSON("orders"),
				Entries: []moi.SemanticEntryUpsertRequest{
					{Kind: "metric", Key: "order_count_a", Spec: json.RawMessage(`{"expr":"count(*)"}`)},
				},
			},
			{
				Name:   nameB,
				Tables: structuredTablesJSON("users"),
				Entries: []moi.SemanticEntryUpsertRequest{
					{Kind: "dimension", Key: "user_id_b", Spec: json.RawMessage(`{"column":"users.id"}`)},
				},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, batchResp)
		require.Len(t, batchResp.Items, 2)
		assert.Equal(t, int64(2), batchResp.Total)

		createdIDs := make([]int64, 0, len(batchResp.Items))
		for _, item := range batchResp.Items {
			if item != nil && item.ID > 0 {
				createdIDs = append(createdIDs, item.ID)
			}
		}
		require.Len(t, createdIDs, 2)
		defer func() {
			for _, id := range createdIDs {
				_ = svc.Delete(context.Background(), id)
			}
		}()

		exportedA, err := svc.Export(ctx, createdIDs[0])
		require.NoError(t, err)
		require.NotNil(t, exportedA.Model)
		assert.True(t, semanticEntryKeyExists(exportedA.Entries, "order_count_a") || semanticEntryKeyExists(exportedA.Entries, "user_id_b"))

		exportedB, err := svc.Export(ctx, createdIDs[1])
		require.NoError(t, err)
		require.NotNil(t, exportedB.Model)
		assert.True(t, semanticEntryKeyExists(exportedB.Entries, "order_count_a") || semanticEntryKeyExists(exportedB.Entries, "user_id_b"))
	})
}

func semanticModelExistsByID(items []*moi.SemanticModel, id int64) bool {
	for _, item := range items {
		if item != nil && item.ID == id {
			return true
		}
	}
	return false
}

func semanticEntryExistsByID(items []*moi.SemanticEntry, id int64) bool {
	for _, item := range items {
		if item != nil && item.ID == id {
			return true
		}
	}
	return false
}

func semanticEntryKeyExists(items []*moi.SemanticEntry, key string) bool {
	for _, item := range items {
		if item != nil && item.Key == key {
			return true
		}
	}
	return false
}
