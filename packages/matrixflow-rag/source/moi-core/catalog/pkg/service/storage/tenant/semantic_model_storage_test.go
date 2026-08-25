package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantStorageImpl_CreateSemanticModel(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		storage := NewTenantStorage("")
		tm := transaction.NewManager(db)
		ctx := WithTransactionManager(context.Background(), tm)

		tablesJSON := json.RawMessage(`[{"db_name":"sales","table_names":["orders","customers"],"parents":[]}]`)
		filesJSON := json.RawMessage(`{"file_ids":["f1"],"parents":[]}`)

		model := &SemanticModelRecord{
			Name:         "sales_model",
			Description:  "sales analytics",
			Tables:       tablesJSON,
			Files:        filesJSON,
			TableSetHash: "abc123",
			CreatedBy:    "u1",
			UpdatedBy:    "u1",
		}

		mock.ExpectExec("INSERT INTO semantic_models").
			WithArgs("sales_model", "sales analytics",
				string(tablesJSON), string(filesJSON), "abc123", "u1", "u1").
			WillReturnResult(sqlmock.NewResult(11, 1))
		mock.ExpectQuery("SELECT id, name, description, tables, files, table_set_hash").
			WithArgs(int64(11)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "name", "description", "tables", "files",
				"table_set_hash", "created_by", "updated_by", "created_at", "updated_at",
			}).AddRow(11, "sales_model", "sales analytics",
				string(tablesJSON), string(filesJSON), "abc123", "u1", "u1", int64(100), int64(101)))

		created, err := storage.CreateSemanticModel(ctx, model)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, int64(11), created.ID)
		assert.JSONEq(t, string(tablesJSON), string(created.Tables))
		assert.JSONEq(t, string(filesJSON), string(created.Files))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success_without_files", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		storage := NewTenantStorage("")
		tm := transaction.NewManager(db)
		ctx := WithTransactionManager(context.Background(), tm)

		tablesJSON := json.RawMessage(`[{"db_name":"db1","table_names":["orders"],"parents":[]}]`)

		model := &SemanticModelRecord{
			Name:         "no_files_model",
			Tables:       tablesJSON,
			TableSetHash: "hash1",
			CreatedBy:    "u1",
			UpdatedBy:    "u1",
		}

		mock.ExpectExec("INSERT INTO semantic_models").
			WithArgs("no_files_model", "",
				string(tablesJSON), nil, "hash1", "u1", "u1").
			WillReturnResult(sqlmock.NewResult(12, 1))
		mock.ExpectQuery("SELECT id, name, description, tables, files, table_set_hash").
			WithArgs(int64(12)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "name", "description", "tables", "files",
				"table_set_hash", "created_by", "updated_by", "created_at", "updated_at",
			}).AddRow(12, "no_files_model", nil,
				string(tablesJSON), nil, "hash1", "u1", "u1", int64(100), int64(101)))

		created, err := storage.CreateSemanticModel(ctx, model)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Nil(t, created.Files)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("success_empty_tables", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		storage := NewTenantStorage("")
		tm := transaction.NewManager(db)
		ctx := WithTransactionManager(context.Background(), tm)

		model := &SemanticModelRecord{
			Name:      "files_only_model",
			Files:     json.RawMessage(`{"file_ids":["f1"],"parents":[]}`),
			CreatedBy: "u1",
			UpdatedBy: "u1",
		}

		mock.ExpectExec("INSERT INTO semantic_models").
			WithArgs("files_only_model", "", nil, `{"file_ids":["f1"],"parents":[]}`, "", "u1", "u1").
			WillReturnResult(sqlmock.NewResult(20, 1))
		mock.ExpectQuery("SELECT id, name, description, tables, files, table_set_hash").
			WithArgs(int64(20)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "name", "description", "tables", "files",
				"table_set_hash", "created_by", "updated_by", "created_at", "updated_at",
			}).AddRow(20, "files_only_model", nil, nil, `{"file_ids":["f1"],"parents":[]}`, "", "u1", "u1", int64(100), int64(101)))

		created, err := storage.CreateSemanticModel(ctx, model)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, int64(20), created.ID)
		assert.Nil(t, created.Tables)
		assert.NotNil(t, created.Files)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		storage := NewTenantStorage("")
		tm := transaction.NewManager(db)
		ctx := WithTransactionManager(context.Background(), tm)

		mock.ExpectExec("INSERT INTO semantic_models").
			WillReturnError(errors.New("Duplicate entry 'sales_model' for key 'uk_semantic_models_name'"))

		_, err = storage.CreateSemanticModel(ctx, &SemanticModelRecord{
			Name:         "sales_model",
			Tables:       json.RawMessage(`[{"db_name":"db","table_names":["orders"],"parents":[]}]`),
			TableSetHash: "hash",
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrSemanticModelAlreadyExist))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantStorageImpl_GetAndListSemanticModels(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewTenantStorage("")
	tm := transaction.NewManager(db)
	ctx := WithTransactionManager(context.Background(), tm)

	tablesJSON := `[{"db_name":"db1","table_names":["orders"],"parents":[]}]`

	mock.ExpectQuery("SELECT id, name, description, tables, files, table_set_hash").
		WithArgs(int64(9)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "description", "tables", "files",
			"table_set_hash", "created_by", "updated_by", "created_at", "updated_at",
		}).AddRow(9, "m1", "desc", tablesJSON, nil, "h1", "u1", "u1", int64(100), int64(101)))

	model, err := storage.GetSemanticModel(ctx, 9)
	require.NoError(t, err)
	require.NotNil(t, model)
	assert.JSONEq(t, tablesJSON, string(model.Tables))
	assert.Nil(t, model.Files)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM semantic_models`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(2)))
	mock.ExpectQuery("SELECT id, name, description, tables, files, table_set_hash").
		WithArgs(int32(DefaultPageSize), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "description", "tables", "files",
			"table_set_hash", "created_by", "updated_by", "created_at", "updated_at",
		}).
			AddRow(10, "m2", "", `[{"db_name":"db2","table_names":["customers"],"parents":[]}]`, nil, "h2", "u1", "u1", int64(200), int64(201)).
			AddRow(9, "m1", "desc", tablesJSON, nil, "h1", "u1", "u1", int64(100), int64(101)))

	list, total, err := storage.ListSemanticModels(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, list, 2)
	assert.Equal(t, int64(10), list[0].ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSemanticModelFilterSQLTagsUseFilesTagsPath(t *testing.T) {
	filterSQL, args, err := semanticModelFilterSQL([]ListFilter{
		{Name: "tags", Values: []string{"finance", "risk%_"}},
	})
	require.NoError(t, err)

	assert.Contains(t, filterSQL, "JSON_CONTAINS(JSON_EXTRACT(files, '$.tags'), ?)")
	assert.NotContains(t, filterSQL, "files LIKE ?")
	assert.Equal(t, []any{`"finance"`, `"risk%_"`}, args)
}

func TestTenantStorageImpl_UpdateSemanticModelPreservesFilesWhenOmitted(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewTenantStorage("")
	ctx := WithTransactionManager(context.Background(), transaction.NewManager(db))
	tablesJSON := json.RawMessage(`[{"db_name":"sales","table_names":["orders"],"parents":[]}]`)

	mock.ExpectExec(`UPDATE semantic_models SET name = \?, description = \?, tables = \?, table_set_hash = \?, updated_by = \? WHERE id = \?`).
		WithArgs("sales_model_renamed", "renamed", string(tablesJSON), "hash2", "u2", int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = storage.UpdateSemanticModel(ctx, &SemanticModelRecord{
		ID:           9,
		Name:         "sales_model_renamed",
		Description:  "renamed",
		Tables:       tablesJSON,
		TableSetHash: "hash2",
		UpdatedBy:    "u2",
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantStorageImpl_UpdateSemanticModelPreservesTablesAndFilesWhenOmitted(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewTenantStorage("")
	ctx := WithTransactionManager(context.Background(), transaction.NewManager(db))

	mock.ExpectExec(`UPDATE semantic_models SET name = \?, description = \?, updated_by = \? WHERE id = \?`).
		WithArgs("sales_model", "metadata only", "u2", int64(9)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = storage.UpdateSemanticModel(ctx, &SemanticModelRecord{
		ID:          9,
		Name:        "sales_model",
		Description: "metadata only",
		UpdatedBy:   "u2",
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantStorageImpl_UpdateSemanticModelReplacesFilesWhenProvided(t *testing.T) {
	tests := []struct {
		name  string
		files json.RawMessage
	}{
		{
			name:  "complete files metadata",
			files: json.RawMessage(`{"file_ids":["f1"],"vector_table":"vec","image_vector_table":"img"}`),
		},
		{
			name:  "explicit empty file ids",
			files: json.RawMessage(`{"file_ids":[]}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			storage := NewTenantStorage("")
			ctx := WithTransactionManager(context.Background(), transaction.NewManager(db))
			tablesJSON := json.RawMessage(`[{"db_name":"sales","table_names":["orders"],"parents":[]}]`)

			mock.ExpectExec(`UPDATE semantic_models SET name = \?, description = \?, tables = \?, table_set_hash = \?, files = \?, updated_by = \? WHERE id = \?`).
				WithArgs("sales_model", "updated", string(tablesJSON), "hash2", string(tt.files), "u2", int64(9)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err = storage.UpdateSemanticModel(ctx, &SemanticModelRecord{
				ID:           9,
				Name:         "sales_model",
				Description:  "updated",
				Tables:       tablesJSON,
				Files:        tt.files,
				TableSetHash: "hash2",
				UpdatedBy:    "u2",
			})
			require.NoError(t, err)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTenantStorageImpl_ListSemanticModelsAppliesIAMIDsBeforePagination(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	storage := NewTenantStorage("")
	ctx := WithTransactionManager(context.Background(), transaction.NewManager(db))

	dbMock.ExpectQuery(`SELECT COUNT\(\*\) FROM semantic_models WHERE id IN \(\?,\?\) AND \(name LIKE \? OR description LIKE \?\)`).
		WithArgs(int64(9), int64(11), "%sales%", "%sales%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	dbMock.ExpectQuery("SELECT id, name, description, tables, files, table_set_hash").
		WithArgs(int64(9), int64(11), "%sales%", "%sales%", int32(5), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "tables", "files", "table_set_hash", "created_by", "updated_by", "created_at", "updated_at"}).AddRow(11, "sales", "", nil, nil, "h", "u", "u", int64(1), int64(1)))

	items, total, err := storage.ListSemanticModels(ctx, WithFilter("ids", []string{"9", "11"}, false), WithFilter("search", []string{"sales"}, true), WithPageSize(5))
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	require.Equal(t, int64(11), items[0].ID)
	require.NoError(t, dbMock.ExpectationsWereMet())
}

func TestTenantStorageImpl_ListSemanticModelsEmptyIAMIDsReturnsEmpty(t *testing.T) {
	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	storage := NewTenantStorage("")
	ctx := WithTransactionManager(context.Background(), transaction.NewManager(db))
	dbMock.ExpectQuery(`SELECT COUNT\(\*\) FROM semantic_models WHERE 1 = 0`).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(0)))
	dbMock.ExpectQuery("SELECT id, name, description, tables, files, table_set_hash").WithArgs(int32(DefaultPageSize), int64(0)).WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "tables", "files", "table_set_hash", "created_by", "updated_by", "created_at", "updated_at"}))
	items, total, err := storage.ListSemanticModels(ctx, WithFilter("ids", nil, false))
	require.NoError(t, err)
	require.Zero(t, total)
	require.Empty(t, items)
	require.NoError(t, dbMock.ExpectationsWereMet())
}

func TestTenantStorageImpl_ListSemanticModelsRejectsInvalidIAMID(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	storage := NewTenantStorage("")
	ctx := WithTransactionManager(context.Background(), transaction.NewManager(db))
	_, _, err = storage.ListSemanticModels(ctx, WithFilter("ids", []string{"not-an-id"}, false))
	require.ErrorContains(t, err, "invalid semantic model IAM resource id")
}

func TestTenantStorageImpl_SemanticEntryCRUD(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewTenantStorage("")
	tm := transaction.NewManager(db)
	ctx := WithTransactionManager(context.Background(), tm)

	mock.ExpectExec("INSERT INTO semantic_entries").
		WithArgs(int64(3), "named_filter", "completed_orders", `["orders"]`, `{"expr":"status='PAID'"}`, "u1", "u1").
		WillReturnResult(sqlmock.NewResult(7, 1))
	mock.ExpectQuery("SELECT id, model_id, kind, key_name, tables, spec").
		WithArgs(int64(3), int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "model_id", "kind", "key_name", "tables", "spec", "created_by", "updated_by", "created_at", "updated_at",
		}).AddRow(7, 3, "named_filter", "completed_orders", `["orders"]`, `{"expr":"status='PAID'"}`, "u1", "u1", int64(11), int64(12)))

	entry, err := storage.CreateSemanticEntry(ctx, &SemanticEntryRecord{
		ModelID:   3,
		Kind:      "named_filter",
		KeyName:   "completed_orders",
		Tables:    []string{"orders"},
		Spec:      []byte(`{"expr":"status='PAID'"}`),
		CreatedBy: "u1",
		UpdatedBy: "u1",
	})
	require.NoError(t, err)
	require.NotNil(t, entry)
	assert.Equal(t, int64(7), entry.ID)

	mock.ExpectExec("UPDATE semantic_entries SET kind = \\?, key_name = \\?, tables = \\?, spec = \\?, updated_by = \\? WHERE model_id = \\? AND id = \\?").
		WithArgs("named_filter", "completed_orders", `["orders"]`, `{"expr":"status='COMPLETE'"}`, "u2", int64(3), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = storage.UpdateSemanticEntry(ctx, &SemanticEntryRecord{
		ID:        7,
		ModelID:   3,
		Kind:      "named_filter",
		KeyName:   "completed_orders",
		Tables:    []string{"orders"},
		Spec:      []byte(`{"expr":"status='COMPLETE'"}`),
		UpdatedBy: "u2",
	})
	require.NoError(t, err)

	mock.ExpectExec("DELETE FROM semantic_entries WHERE model_id = \\? AND id = \\?").
		WithArgs(int64(3), int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = storage.DeleteSemanticEntry(ctx, 3, 7)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantStorageImpl_CreateSemanticEntriesBatch(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		storage := NewTenantStorage("")
		tm := transaction.NewManager(db)
		ctx := WithTransactionManager(context.Background(), tm)

		mock.ExpectExec("INSERT INTO semantic_entries").
			WithArgs(
				int64(3), "dimension", "order_id", `["orders"]`, `{"column":"orders.id"}`, "u1", "u1",
				int64(3), "metric", "gmv", `["orders"]`, `{"expr":"sum(amount)"}`, "u1", "u1",
			).
			WillReturnResult(sqlmock.NewResult(0, 2))

		err = storage.CreateSemanticEntriesBatch(ctx, []*SemanticEntryRecord{
			{
				ModelID:   3,
				Kind:      "dimension",
				KeyName:   "order_id",
				Tables:    []string{"orders"},
				Spec:      []byte(`{"column":"orders.id"}`),
				CreatedBy: "u1",
				UpdatedBy: "u1",
			},
			{
				ModelID:   3,
				Kind:      "metric",
				KeyName:   "gmv",
				Tables:    []string{"orders"},
				Spec:      []byte(`{"expr":"sum(amount)"}`),
				CreatedBy: "u1",
				UpdatedBy: "u1",
			},
		})
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("duplicate", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		storage := NewTenantStorage("")
		tm := transaction.NewManager(db)
		ctx := WithTransactionManager(context.Background(), tm)

		mock.ExpectExec("INSERT INTO semantic_entries").
			WillReturnError(errors.New("Duplicate entry '3-metric-gmv' for key 'uk_semantic_entries_model_kind_key'"))

		err = storage.CreateSemanticEntriesBatch(ctx, []*SemanticEntryRecord{
			{
				ModelID: 3,
				Kind:    "metric",
				KeyName: "gmv",
				Tables:  []string{"orders"},
				Spec:    []byte(`{"expr":"sum(amount)"}`),
			},
		})
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrSemanticEntryAlreadyExist))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantStorageImpl_SemanticModelNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewTenantStorage("")
	tm := transaction.NewManager(db)
	ctx := WithTransactionManager(context.Background(), tm)

	mock.ExpectQuery("SELECT id, name, description, tables, files, table_set_hash").
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	got, err := storage.GetSemanticModel(ctx, 999)
	require.Error(t, err)
	assert.Nil(t, got)
	assert.True(t, errors.Is(err, ErrSemanticModelNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantStorageImpl_ListSemanticModels_ExactName(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := NewTenantStorage("")
	tm := transaction.NewManager(db)
	ctx := WithTransactionManager(context.Background(), tm)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM semantic_models WHERE name = \?`).
		WithArgs("sales_model").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT id, name, description, tables, files, table_set_hash, created_by, updated_by, UNIX_TIMESTAMP\(created_at\), UNIX_TIMESTAMP\(updated_at\) FROM semantic_models WHERE name = \?`).
		WithArgs("sales_model", int32(2), int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "name", "description", "tables", "files",
			"table_set_hash", "created_by", "updated_by", "created_at", "updated_at",
		}).AddRow(9, "sales_model", "desc", `[]`, nil, "hash", "u1", "u1", int64(1), int64(2)))

	items, total, err := storage.ListSemanticModels(ctx, WithPageSize(2), WithFilter("name", []string{"sales_model"}, false))
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, items, 1)
	assert.Equal(t, "sales_model", items[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantStorageImpl_GetSemanticModelForUpdate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		storage := NewTenantStorage("")
		tm := transaction.NewManager(db)
		ctx := WithTransactionManager(context.Background(), tm)

		mock.ExpectQuery("SELECT id, name, description, tables, files, table_set_hash[\\s\\S]*FOR UPDATE").
			WithArgs(int64(9)).
			WillReturnRows(sqlmock.NewRows([]string{
				"id", "name", "description", "tables", "files",
				"table_set_hash", "created_by", "updated_by", "created_at", "updated_at",
			}).AddRow(9, "sales_model", "desc", `[]`, nil, "hash", "u1", "u1", int64(1), int64(2)))

		model, err := storage.GetSemanticModelForUpdate(ctx, 9)
		require.NoError(t, err)
		require.NotNil(t, model)
		assert.Equal(t, int64(9), model.ID)
		assert.Equal(t, "sales_model", model.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not_found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		storage := NewTenantStorage("")
		tm := transaction.NewManager(db)
		ctx := WithTransactionManager(context.Background(), tm)

		mock.ExpectQuery("SELECT id, name, description, tables, files, table_set_hash").
			WithArgs(int64(99)).
			WillReturnError(sql.ErrNoRows)

		_, err = storage.GetSemanticModelForUpdate(ctx, 99)
		require.ErrorIs(t, err, ErrSemanticModelNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTenantStorageImpl_DeleteSemanticModel(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		storage := NewTenantStorage("")
		tm := transaction.NewManager(db)
		ctx := WithTransactionManager(context.Background(), tm)

		mock.ExpectExec("DELETE FROM semantic_entries WHERE model_id = \\?").
			WithArgs(int64(9)).
			WillReturnResult(sqlmock.NewResult(0, 3))
		mock.ExpectExec("DELETE FROM semantic_models WHERE id = \\?").
			WithArgs(int64(9)).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err = storage.DeleteSemanticModel(ctx, 9)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not_found", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		storage := NewTenantStorage("")
		tm := transaction.NewManager(db)
		ctx := WithTransactionManager(context.Background(), tm)

		mock.ExpectExec("DELETE FROM semantic_entries WHERE model_id = \\?").
			WithArgs(int64(99)).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("DELETE FROM semantic_models WHERE id = \\?").
			WithArgs(int64(99)).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err = storage.DeleteSemanticModel(ctx, 99)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrSemanticModelNotFound))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
