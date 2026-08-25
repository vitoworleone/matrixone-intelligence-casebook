package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/matrixflow/moi-core/agent-tools/knowledge"
)

type upsertKnowledgeTableService struct {
	reader   SchemaReader
	mutation knowledge.SQLMutationExecutor
}

const maxUpsertKnowledgeTableRecords = 100

func NewUpsertKnowledgeTable(deps Deps) knowledge.UpsertKnowledgeTable {
	return &upsertKnowledgeTableService{
		reader:   deps.SchemaReader,
		mutation: deps.SQLMutationExecutor,
	}
}

func (s *upsertKnowledgeTableService) Execute(ctx context.Context, req knowledge.UpsertKnowledgeTableRequest) (*knowledge.UpsertKnowledgeTableResponse, error) {
	if s == nil || s.reader == nil {
		return nil, fmt.Errorf("upsert_knowledge_table: schema reader is not configured")
	}
	if s.mutation == nil {
		return nil, fmt.Errorf("upsert_knowledge_table: mutation executor is not configured")
	}
	if strings.TrimSpace(req.Scope.WorkspaceID) == "" {
		return nil, fmt.Errorf("upsert_knowledge_table: workspace_id is required")
	}
	if strings.TrimSpace(req.Scope.DBName) == "" {
		return nil, fmt.Errorf("upsert_knowledge_table: db_name is required")
	}
	if len(req.Scope.Tables) == 0 {
		return nil, fmt.Errorf("upsert_knowledge_table: selected knowledge table scope is required")
	}

	tableName, err := scopedUpsertTable(req.TableName, req.Scope.Tables)
	if err != nil {
		return nil, err
	}
	columns, err := s.reader.ListColumns(ctx, req.Scope, []string{tableName})
	if err != nil {
		return nil, fmt.Errorf("upsert_knowledge_table: list columns: %w", err)
	}
	tableColumns := columnsByTableKey(columns)[tableKey(tableName)]
	if len(tableColumns) == 0 {
		return nil, fmt.Errorf("upsert_knowledge_table: table %q has no readable schema", tableName)
	}

	available := make(map[string]knowledge.ColumnInfo, len(tableColumns))
	primaryKeys := make(map[string]struct{})
	for _, column := range tableColumns {
		name := strings.TrimSpace(column.Name)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, exists := available[key]; exists {
			return nil, fmt.Errorf("upsert_knowledge_table: table %q has ambiguous column %q", tableName, name)
		}
		available[key] = column
		if column.PrimaryKey || column.IsPrimaryKey {
			primaryKeys[key] = struct{}{}
		}
	}
	if len(primaryKeys) == 0 {
		return nil, fmt.Errorf("upsert_knowledge_table: table %q must have a primary key", tableName)
	}

	records, err := upsertKnowledgeTableRecords(req)
	if err != nil {
		return nil, err
	}
	if len(records) > maxUpsertKnowledgeTableRecords {
		return nil, fmt.Errorf("upsert_knowledge_table: records exceeds the maximum of %d", maxUpsertKnowledgeTableRecords)
	}

	var keyNames []string
	var valueNames []string
	var firstKey map[string]any
	args := make([]any, 0, len(records)*4)
	rows := make([]string, 0, len(records))
	seenPrimaryKeys := make(map[string]struct{}, len(records))
	for index, record := range records {
		key, err := normalizeUpsertValues(fmt.Sprintf("records[%d].key", index), record.Key, available, false)
		if err != nil {
			return nil, err
		}
		values, err := normalizeUpsertValues(fmt.Sprintf("records[%d].values", index), record.Values, available, true)
		if err != nil {
			return nil, err
		}
		if err := validateUpsertRecord(key, values, primaryKeys, available); err != nil {
			return nil, err
		}

		recordKeyNames := sortedUpsertColumns(key, available)
		recordValueNames := sortedUpsertColumns(values, available)
		if index == 0 {
			keyNames = recordKeyNames
			valueNames = recordValueNames
			firstKey = upsertResponseValues(key, available)
		} else if !sameUpsertColumns(keyNames, recordKeyNames) || !sameUpsertColumns(valueNames, recordValueNames) {
			return nil, fmt.Errorf("upsert_knowledge_table: every batch record must use the same key and values column sets")
		}

		primaryKeyIdentity, err := upsertPrimaryKeyIdentity(key, primaryKeys, available)
		if err != nil {
			return nil, err
		}
		if _, exists := seenPrimaryKeys[primaryKeyIdentity]; exists {
			return nil, fmt.Errorf("upsert_knowledge_table: records contains duplicate primary key values")
		}
		seenPrimaryKeys[primaryKeyIdentity] = struct{}{}

		placeholders := make([]string, 0, len(keyNames)+len(valueNames))
		for _, name := range keyNames {
			args = append(args, key[strings.ToLower(name)])
			placeholders = append(placeholders, "?")
		}
		for _, name := range valueNames {
			args = append(args, values[strings.ToLower(name)])
			placeholders = append(placeholders, "?")
		}
		rows = append(rows, "("+strings.Join(placeholders, ", ")+")")
	}

	columnNames := append(append([]string(nil), keyNames...), valueNames...)

	quotedColumns := make([]string, 0, len(columnNames))
	updates := make([]string, 0, len(valueNames))
	for _, name := range columnNames {
		quotedColumns = append(quotedColumns, quoteIdentifier(name))
	}
	for _, name := range valueNames {
		quoted := quoteIdentifier(name)
		updates = append(updates, quoted+" = VALUES("+quoted+")")
	}
	statement := "INSERT INTO " + quoteIdentifier(tableName) + " (" + strings.Join(quotedColumns, ", ") + ") VALUES " + strings.Join(rows, ", ") + " ON DUPLICATE KEY UPDATE " + strings.Join(updates, ", ")

	ctx = knowledge.ContextWithScope(ctx, req.Scope)
	rowsAffected, err := s.mutation.ExecuteMutation(ctx, req.Scope.DBName, statement, args...)
	if err != nil {
		return nil, fmt.Errorf("upsert_knowledge_table: execute mutation: %w", err)
	}
	return &knowledge.UpsertKnowledgeTableResponse{
		TableName:    tableName,
		Key:          singleUpsertResponseKey(firstKey, len(records)),
		RecordCount:  len(records),
		RowsAffected: rowsAffected,
	}, nil
}

func upsertKnowledgeTableRecords(req knowledge.UpsertKnowledgeTableRequest) ([]knowledge.UpsertKnowledgeTableRecord, error) {
	if len(req.Records) > 0 {
		if len(req.Key) > 0 || len(req.Values) > 0 {
			return nil, fmt.Errorf("upsert_knowledge_table: records cannot be combined with key or values")
		}
		return req.Records, nil
	}
	if len(req.Key) == 0 || len(req.Values) == 0 {
		return nil, fmt.Errorf("upsert_knowledge_table: key and values are required when records is omitted")
	}
	return []knowledge.UpsertKnowledgeTableRecord{{Key: req.Key, Values: req.Values}}, nil
}

func validateUpsertRecord(key, values map[string]any, primaryKeys map[string]struct{}, columns map[string]knowledge.ColumnInfo) error {
	for primaryKey := range primaryKeys {
		if _, ok := key[primaryKey]; !ok {
			return fmt.Errorf("upsert_knowledge_table: key must include primary key column %q", columns[primaryKey].Name)
		}
	}
	for column := range values {
		if _, ok := key[column]; ok {
			return fmt.Errorf("upsert_knowledge_table: key and values must not both contain column %q", columns[column].Name)
		}
	}
	return nil
}

func sameUpsertColumns(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func upsertPrimaryKeyIdentity(key map[string]any, primaryKeys map[string]struct{}, columns map[string]knowledge.ColumnInfo) (string, error) {
	values := make(map[string]any, len(primaryKeys))
	for primaryKey := range primaryKeys {
		values[columns[primaryKey].Name] = key[primaryKey]
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("upsert_knowledge_table: encode primary key values: %w", err)
	}
	return string(encoded), nil
}

func singleUpsertResponseKey(key map[string]any, recordCount int) map[string]any {
	if recordCount != 1 {
		return nil
	}
	return key
}

func scopedUpsertTable(requested string, scopeTables []string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return "", fmt.Errorf("upsert_knowledge_table: table_name is required")
	}
	for _, table := range scopeTables {
		table = strings.TrimSpace(table)
		if table != "" && strings.EqualFold(table, requested) {
			return table, nil
		}
	}
	return "", fmt.Errorf("upsert_knowledge_table: table %q is outside selected knowledge table scope", requested)
}

func normalizeUpsertValues(field string, values map[string]any, columns map[string]knowledge.ColumnInfo, allowNull bool) (map[string]any, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("upsert_knowledge_table: %s is required", field)
	}
	out := make(map[string]any, len(values))
	for rawName, value := range values {
		name := strings.TrimSpace(rawName)
		if name == "" || name != rawName {
			return nil, fmt.Errorf("upsert_knowledge_table: %s contains an invalid column name", field)
		}
		key := strings.ToLower(name)
		column, ok := columns[key]
		if !ok {
			return nil, fmt.Errorf("upsert_knowledge_table: column %q is not in the selected table schema", name)
		}
		if _, exists := out[key]; exists {
			return nil, fmt.Errorf("upsert_knowledge_table: %s contains duplicate column %q", field, name)
		}
		if value == nil && !allowNull {
			return nil, fmt.Errorf("upsert_knowledge_table: key column %q cannot be null", column.Name)
		}
		normalized, err := normalizeUpsertValue(value, column)
		if err != nil {
			return nil, err
		}
		out[key] = normalized
	}
	return out, nil
}

func normalizeUpsertValue(value any, column knowledge.ColumnInfo) (any, error) {
	switch typed := value.(type) {
	case map[string]any, []any:
		if !strings.Contains(strings.ToLower(column.Type), "json") {
			return nil, fmt.Errorf("upsert_knowledge_table: column %q does not accept an object or array value", column.Name)
		}
		encoded, err := json.Marshal(typed)
		if err != nil {
			return nil, fmt.Errorf("upsert_knowledge_table: encode JSON column %q: %w", column.Name, err)
		}
		return encoded, nil
	default:
		return value, nil
	}
}

func sortedUpsertColumns(values map[string]any, columns map[string]knowledge.ColumnInfo) []string {
	names := make([]string, 0, len(values))
	for key := range values {
		names = append(names, columns[key].Name)
	}
	sort.Strings(names)
	return names
}

func upsertResponseValues(values map[string]any, columns map[string]knowledge.ColumnInfo) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[columns[key].Name] = value
	}
	return out
}
