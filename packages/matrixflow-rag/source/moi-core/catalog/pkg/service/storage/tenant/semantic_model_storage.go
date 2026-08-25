package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrSemanticModelNotFound     = errors.New("semantic model not found")
	ErrSemanticModelAlreadyExist = errors.New("semantic model already exists")
	ErrSemanticEntryNotFound     = errors.New("semantic entry not found")
	ErrSemanticEntryAlreadyExist = errors.New("semantic entry already exists")
)

// SemanticModelRecord is the tenant storage record for semantic_models.
type SemanticModelRecord struct {
	ID           int64
	Name         string
	Description  string
	Tables       json.RawMessage // structured JSON: [{db_name, table_names, parents}]
	Files        json.RawMessage // structured JSON: {file_ids, parents}
	TableSetHash string
	CreatedBy    string
	UpdatedBy    string
	CreatedAt    int64
	UpdatedAt    int64
}

// SemanticModelTagStat is an aggregated KB-level tag count for semantic models.
type SemanticModelTagStat struct {
	Tag   string
	Count int64
}

// SemanticEntryRecord is the tenant storage record for semantic_entries.
type SemanticEntryRecord struct {
	ID        int64
	ModelID   int64
	Kind      string
	KeyName   string
	Tables    []string
	Spec      json.RawMessage
	CreatedBy string
	UpdatedBy string
	CreatedAt int64
	UpdatedAt int64
}

// SemanticModelStorage defines tenant data access for semantic model CRUD.
type SemanticModelStorage interface {
	CreateSemanticModel(ctx context.Context, model *SemanticModelRecord) (*SemanticModelRecord, error)
	GetSemanticModel(ctx context.Context, modelID int64) (*SemanticModelRecord, error)
	// GetSemanticModelForUpdate locks the target semantic_models row with SELECT ... FOR UPDATE.
	// Callers must already be inside a writable tenant transaction.
	GetSemanticModelForUpdate(ctx context.Context, modelID int64) (*SemanticModelRecord, error)
	// LockSemanticModelsForUpdate locks the given model IDs in ascending order and returns the found rows.
	// Missing IDs are omitted from the result; callers decide whether absence is an error.
	LockSemanticModelsForUpdate(ctx context.Context, modelIDs []int64) ([]*SemanticModelRecord, error)
	ListSemanticModels(ctx context.Context, opts ...ListOption) ([]*SemanticModelRecord, int64, error)
	ListSemanticModelTags(ctx context.Context, opts ...ListOption) ([]SemanticModelTagStat, error)
	UpdateSemanticModel(ctx context.Context, model *SemanticModelRecord) error
	DeleteSemanticModel(ctx context.Context, modelID int64) error

	CreateSemanticEntry(ctx context.Context, entry *SemanticEntryRecord) (*SemanticEntryRecord, error)
	GetSemanticEntry(ctx context.Context, modelID, entryID int64) (*SemanticEntryRecord, error)
	ListSemanticEntries(ctx context.Context, modelID int64, kind string, opts ...ListOption) ([]*SemanticEntryRecord, int64, error)
	UpdateSemanticEntry(ctx context.Context, entry *SemanticEntryRecord) error
	DeleteSemanticEntry(ctx context.Context, modelID, entryID int64) error
}

var _ SemanticModelStorage = (*TenantStorageImpl)(nil)

const semanticEntryInsertBatchSize = 100

func (s *TenantStorageImpl) CreateSemanticModel(ctx context.Context, model *SemanticModelRecord) (*SemanticModelRecord, error) {
	if model == nil {
		return nil, fmt.Errorf("semantic model is required")
	}
	if strings.TrimSpace(model.Name) == "" {
		return nil, fmt.Errorf("name is required")
	}

	executor, err := s.getExecutor(ctx)
	if err != nil {
		return nil, err
	}

	var tablesArg interface{}
	if len(model.Tables) > 0 {
		tablesArg = string(model.Tables)
	}

	var filesArg interface{}
	if len(model.Files) > 0 {
		filesArg = string(model.Files)
	}

	res, err := executor.ExecContext(
		ctx,
		fmt.Sprintf(`INSERT INTO %s (name, description, tables, files, table_set_hash, created_by, updated_by) VALUES (?, ?, ?, ?, ?, ?, ?)`, s.tableName("semantic_models")),
		model.Name,
		model.Description,
		tablesArg,
		filesArg,
		model.TableSetHash,
		model.CreatedBy,
		model.UpdatedBy,
	)
	if err != nil {
		if IsDuplicateKeyError(err) {
			return nil, ErrSemanticModelAlreadyExist
		}
		return nil, fmt.Errorf("create semantic model: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get semantic model last insert id: %w", err)
	}
	return s.GetSemanticModel(ctx, id)
}

func (s *TenantStorageImpl) GetSemanticModel(ctx context.Context, modelID int64) (*SemanticModelRecord, error) {
	return s.getSemanticModel(ctx, modelID, false)
}

func (s *TenantStorageImpl) GetSemanticModelForUpdate(ctx context.Context, modelID int64) (*SemanticModelRecord, error) {
	return s.getSemanticModel(ctx, modelID, true)
}

func (s *TenantStorageImpl) LockSemanticModelsForUpdate(ctx context.Context, modelIDs []int64) ([]*SemanticModelRecord, error) {
	if len(modelIDs) == 0 {
		return nil, nil
	}
	ordered := uniqueSortedPositiveInt64(modelIDs)
	if len(ordered) == 0 {
		return nil, nil
	}
	out := make([]*SemanticModelRecord, 0, len(ordered))
	for _, modelID := range ordered {
		record, err := s.GetSemanticModelForUpdate(ctx, modelID)
		if err != nil {
			if errors.Is(err, ErrSemanticModelNotFound) {
				continue
			}
			return nil, err
		}
		out = append(out, record)
	}
	return out, nil
}

func (s *TenantStorageImpl) getSemanticModel(ctx context.Context, modelID int64, forUpdate bool) (*SemanticModelRecord, error) {
	if modelID <= 0 {
		return nil, fmt.Errorf("model_id is required")
	}

	executor, err := s.getExecutor(ctx)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT id, name, description, tables, files, table_set_hash, created_by, updated_by, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at) FROM %s WHERE id = ? LIMIT 1`, s.tableName("semantic_models"))
	if forUpdate {
		query += " FOR UPDATE"
	}
	var (
		record                               SemanticModelRecord
		description, tablesRaw, createdByRaw sql.NullString
		filesRaw                             sql.NullString
		updatedByRaw                         sql.NullString
	)
	if err := executor.QueryRowContext(ctx, query, modelID).Scan(
		&record.ID,
		&record.Name,
		&description,
		&tablesRaw,
		&filesRaw,
		&record.TableSetHash,
		&createdByRaw,
		&updatedByRaw,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSemanticModelNotFound
		}
		if forUpdate {
			return nil, fmt.Errorf("get semantic model for update: %w", err)
		}
		return nil, fmt.Errorf("get semantic model: %w", err)
	}

	record.Description = description.String
	record.Tables = parseJSONRaw(tablesRaw)
	record.Files = parseJSONRaw(filesRaw)
	record.CreatedBy = createdByRaw.String
	record.UpdatedBy = updatedByRaw.String
	return &record, nil
}

func uniqueSortedPositiveInt64(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (s *TenantStorageImpl) ListSemanticModels(ctx context.Context, opts ...ListOption) ([]*SemanticModelRecord, int64, error) {
	executor, err := s.getExecutor(ctx)
	if err != nil {
		return nil, 0, err
	}

	options := applyListOptions(opts)
	offset := GetCurrentOffset(options.PageToken)

	filterSQL, filterArgs, err := semanticModelFilterSQL(options.Filters)
	if err != nil {
		return nil, 0, err
	}

	var total int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s%s`, s.tableName("semantic_models"), filterSQL)
	if err := executor.QueryRowContext(ctx, countQuery, filterArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count semantic models: %w", err)
	}

	query := fmt.Sprintf(`SELECT id, name, description, tables, files, table_set_hash, created_by, updated_by, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at) FROM %s%s ORDER BY id DESC LIMIT ? OFFSET ?`, s.tableName("semantic_models"), filterSQL)
	queryArgs := append(filterArgs, options.PageSize, offset)
	rows, err := executor.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list semantic models: %w", err)
	}
	defer rows.Close()

	list := make([]*SemanticModelRecord, 0)
	for rows.Next() {
		var (
			record                               SemanticModelRecord
			description, tablesRaw, createdByRaw sql.NullString
			filesRaw                             sql.NullString
			updatedByRaw                         sql.NullString
		)
		if err := rows.Scan(
			&record.ID,
			&record.Name,
			&description,
			&tablesRaw,
			&filesRaw,
			&record.TableSetHash,
			&createdByRaw,
			&updatedByRaw,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan semantic model: %w", err)
		}
		record.Description = description.String
		record.Tables = parseJSONRaw(tablesRaw)
		record.Files = parseJSONRaw(filesRaw)
		record.CreatedBy = createdByRaw.String
		record.UpdatedBy = updatedByRaw.String
		list = append(list, &record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate semantic models: %w", err)
	}
	return list, total, nil
}

func (s *TenantStorageImpl) ListSemanticModelTags(ctx context.Context, opts ...ListOption) ([]SemanticModelTagStat, error) {
	executor, err := s.getExecutor(ctx)
	if err != nil {
		return nil, err
	}

	options := applyListOptions(opts)
	filterSQL, filterArgs, err := semanticModelFilterSQL(options.Filters)
	if err != nil {
		return nil, err
	}
	rows, err := executor.QueryContext(ctx, fmt.Sprintf(`SELECT files FROM %s%s`, s.tableName("semantic_models"), filterSQL), filterArgs...)
	if err != nil {
		return nil, fmt.Errorf("list semantic model tags: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var filesRaw sql.NullString
		if err := rows.Scan(&filesRaw); err != nil {
			return nil, fmt.Errorf("scan semantic model tags: %w", err)
		}
		for _, tag := range semanticModelTagsFromFiles(filesRaw.String) {
			counts[tag]++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate semantic model tags: %w", err)
	}

	stats := make([]SemanticModelTagStat, 0, len(counts))
	for tag, count := range counts {
		stats = append(stats, SemanticModelTagStat{Tag: tag, Count: count})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Count != stats[j].Count {
			return stats[i].Count > stats[j].Count
		}
		return stats[i].Tag < stats[j].Tag
	})
	return stats, nil
}

func semanticModelFilterSQL(filters []ListFilter) (string, []any, error) {
	var queryParts []string
	var args []any
	for _, f := range filters {
		switch f.Name {
		case "name":
			if len(f.Values) == 0 || strings.TrimSpace(f.Values[0]) == "" {
				continue
			}
			// Exact-name lookup used by readiness/package gates. Prefer equality so the
			// unique name index can be used and avoid false misses from page-limited LIKE.
			if f.Fuzzy {
				keyword := "%" + escapeSQLLike(f.Values[0]) + "%"
				queryParts = append(queryParts, "name LIKE ?")
				args = append(args, keyword)
			} else {
				queryParts = append(queryParts, "name = ?")
				args = append(args, f.Values[0])
			}
		case "search":
			if len(f.Values) == 0 || strings.TrimSpace(f.Values[0]) == "" {
				continue
			}
			// Escape SQL LIKE wildcards in user input to prevent unintended pattern matching.
			keyword := "%" + escapeSQLLike(f.Values[0]) + "%"
			queryParts = append(queryParts, "(name LIKE ? OR description LIKE ?)")
			args = append(args, keyword, keyword)
		case "tags":
			var tagParts []string
			for _, tag := range f.Values {
				tag = strings.TrimSpace(tag)
				if tag == "" {
					continue
				}
				encoded, err := json.Marshal(tag)
				if err != nil {
					continue
				}
				tagParts = append(tagParts, "JSON_CONTAINS(JSON_EXTRACT(files, '$.tags'), ?)")
				args = append(args, string(encoded))
			}
			if len(tagParts) > 0 {
				queryParts = append(queryParts, "("+strings.Join(tagParts, " OR ")+")")
			}
		case "ids":
			// IAM resource ids are applied before count/pagination.
			if len(f.Values) == 0 {
				queryParts = append(queryParts, "1 = 0")
				continue
			}
			placeholders := make([]string, 0, len(f.Values))
			for _, value := range f.Values {
				id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
				if err != nil || id <= 0 {
					return "", nil, fmt.Errorf("invalid semantic model IAM resource id %q", value)
				}
				placeholders = append(placeholders, "?")
				args = append(args, id)
			}
			queryParts = append(queryParts, "id IN ("+strings.Join(placeholders, ",")+")")
		}
	}
	if len(queryParts) == 0 {
		return "", nil, nil
	}
	return " WHERE " + strings.Join(queryParts, " AND "), args, nil
}

func escapeSQLLike(value string) string {
	escaped := strings.ReplaceAll(value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "%", "\\%")
	escaped = strings.ReplaceAll(escaped, "_", "\\_")
	return escaped
}

func semanticModelTagsFromFiles(files string) []string {
	if strings.TrimSpace(files) == "" {
		return nil
	}
	var payload struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal([]byte(files), &payload); err != nil {
		return nil
	}
	seen := make(map[string]struct{})
	tags := make([]string, 0, len(payload.Tags))
	for _, tag := range payload.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}
	return tags
}

func (s *TenantStorageImpl) UpdateSemanticModel(ctx context.Context, model *SemanticModelRecord) error {
	if model == nil {
		return fmt.Errorf("semantic model is required")
	}
	if model.ID <= 0 {
		return fmt.Errorf("model_id is required")
	}
	if strings.TrimSpace(model.Name) == "" {
		return fmt.Errorf("name is required")
	}

	executor, err := s.getExecutor(ctx)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`UPDATE %s SET name = ?, description = ?`, s.tableName("semantic_models"))
	args := []interface{}{
		model.Name,
		model.Description,
	}
	if model.Tables != nil {
		var tablesArg interface{}
		if len(model.Tables) > 0 {
			tablesArg = string(model.Tables)
		}
		query += `, tables = ?, table_set_hash = ?`
		args = append(args, tablesArg, model.TableSetHash)
	}
	if model.Files != nil {
		query += `, files = ?`
		args = append(args, string(model.Files))
	}
	query += `, updated_by = ? WHERE id = ?`
	args = append(args,
		model.UpdatedBy,
		model.ID,
	)

	res, err := executor.ExecContext(ctx, query, args...)
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrSemanticModelAlreadyExist
		}
		return fmt.Errorf("update semantic model: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("semantic model update rows_affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrSemanticModelNotFound
	}
	return nil
}

func (s *TenantStorageImpl) DeleteSemanticModel(ctx context.Context, modelID int64) error {
	if modelID <= 0 {
		return fmt.Errorf("model_id is required")
	}

	executor, err := s.getExecutor(ctx)
	if err != nil {
		return err
	}

	if _, err := executor.ExecContext(
		ctx,
		fmt.Sprintf(`DELETE FROM %s WHERE model_id = ?`, s.tableName("semantic_entries")),
		modelID,
	); err != nil {
		return fmt.Errorf("delete semantic entries by model_id: %w", err)
	}

	res, err := executor.ExecContext(
		ctx,
		fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, s.tableName("semantic_models")),
		modelID,
	)
	if err != nil {
		return fmt.Errorf("delete semantic model: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("semantic model delete rows_affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrSemanticModelNotFound
	}
	return nil
}

func (s *TenantStorageImpl) CreateSemanticEntry(ctx context.Context, entry *SemanticEntryRecord) (*SemanticEntryRecord, error) {
	if err := validateSemanticEntryRecord(entry); err != nil {
		return nil, err
	}

	executor, err := s.getExecutor(ctx)
	if err != nil {
		return nil, err
	}

	tablesJSON, err := json.Marshal(entry.Tables)
	if err != nil {
		return nil, fmt.Errorf("marshal entry tables: %w", err)
	}

	res, err := executor.ExecContext(
		ctx,
		fmt.Sprintf(`INSERT INTO %s (model_id, kind, key_name, tables, spec, created_by, updated_by) VALUES (?, ?, ?, ?, ?, ?, ?)`, s.tableName("semantic_entries")),
		entry.ModelID,
		entry.Kind,
		entry.KeyName,
		string(tablesJSON),
		string(entry.Spec),
		entry.CreatedBy,
		entry.UpdatedBy,
	)
	if err != nil {
		if IsDuplicateKeyError(err) {
			return nil, ErrSemanticEntryAlreadyExist
		}
		return nil, fmt.Errorf("create semantic entry: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get semantic entry last insert id: %w", err)
	}
	return s.GetSemanticEntry(ctx, entry.ModelID, id)
}

// CreateSemanticEntriesBatch inserts multiple semantic entries with batch SQL.
// It is intended for import paths to reduce round trips.
func (s *TenantStorageImpl) CreateSemanticEntriesBatch(ctx context.Context, entries []*SemanticEntryRecord) error {
	if len(entries) == 0 {
		return nil
	}

	executor, err := s.getExecutor(ctx)
	if err != nil {
		return err
	}

	for start := 0; start < len(entries); start += semanticEntryInsertBatchSize {
		end := start + semanticEntryInsertBatchSize
		if end > len(entries) {
			end = len(entries)
		}
		batch := entries[start:end]

		values := make([]string, 0, len(batch))
		args := make([]any, 0, len(batch)*7)
		for _, entry := range batch {
			if err := validateSemanticEntryRecord(entry); err != nil {
				return err
			}
			tablesJSON, err := json.Marshal(entry.Tables)
			if err != nil {
				return fmt.Errorf("marshal entry tables: %w", err)
			}
			values = append(values, "(?, ?, ?, ?, ?, ?, ?)")
			args = append(args,
				entry.ModelID,
				entry.Kind,
				entry.KeyName,
				string(tablesJSON),
				string(entry.Spec),
				entry.CreatedBy,
				entry.UpdatedBy,
			)
		}

		query := fmt.Sprintf(
			`INSERT INTO %s (model_id, kind, key_name, tables, spec, created_by, updated_by) VALUES %s`,
			s.tableName("semantic_entries"),
			strings.Join(values, ","),
		)
		if _, err := executor.ExecContext(ctx, query, args...); err != nil {
			if IsDuplicateKeyError(err) {
				return ErrSemanticEntryAlreadyExist
			}
			return fmt.Errorf("create semantic entries batch: %w", err)
		}
	}

	return nil
}

func validateSemanticEntryRecord(entry *SemanticEntryRecord) error {
	if entry == nil {
		return fmt.Errorf("semantic entry is required")
	}
	if entry.ModelID <= 0 {
		return fmt.Errorf("model_id is required")
	}
	if strings.TrimSpace(entry.Kind) == "" {
		return fmt.Errorf("kind is required")
	}
	if strings.TrimSpace(entry.KeyName) == "" {
		return fmt.Errorf("key_name is required")
	}
	if len(entry.Spec) == 0 {
		return fmt.Errorf("spec is required")
	}
	return nil
}

func (s *TenantStorageImpl) GetSemanticEntry(ctx context.Context, modelID, entryID int64) (*SemanticEntryRecord, error) {
	if modelID <= 0 {
		return nil, fmt.Errorf("model_id is required")
	}
	if entryID <= 0 {
		return nil, fmt.Errorf("entry_id is required")
	}

	executor, err := s.getExecutor(ctx)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT id, model_id, kind, key_name, tables, spec, created_by, updated_by, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at) FROM %s WHERE model_id = ? AND id = ? LIMIT 1`, s.tableName("semantic_entries"))
	var (
		record                           SemanticEntryRecord
		kindRaw, keyNameRaw              string
		tablesRaw, specRaw, createdByRaw sql.NullString
		updatedByRaw                     sql.NullString
	)
	if err := executor.QueryRowContext(ctx, query, modelID, entryID).Scan(
		&record.ID,
		&record.ModelID,
		&kindRaw,
		&keyNameRaw,
		&tablesRaw,
		&specRaw,
		&createdByRaw,
		&updatedByRaw,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSemanticEntryNotFound
		}
		return nil, fmt.Errorf("get semantic entry: %w", err)
	}

	record.Kind = kindRaw
	record.KeyName = keyNameRaw
	record.Tables = parseJSONStringArray(tablesRaw)
	record.Spec = parseJSONRaw(specRaw)
	record.CreatedBy = createdByRaw.String
	record.UpdatedBy = updatedByRaw.String
	return &record, nil
}

func (s *TenantStorageImpl) ListSemanticEntries(ctx context.Context, modelID int64, kind string, opts ...ListOption) ([]*SemanticEntryRecord, int64, error) {
	if modelID <= 0 {
		return nil, 0, fmt.Errorf("model_id is required")
	}

	executor, err := s.getExecutor(ctx)
	if err != nil {
		return nil, 0, err
	}

	options := applyListOptions(opts)
	offset := GetCurrentOffset(options.PageToken)

	whereSQL := "WHERE model_id = ?"
	args := make([]interface{}, 0, 4)
	args = append(args, modelID)
	if strings.TrimSpace(kind) != "" {
		whereSQL += " AND kind = ?"
		args = append(args, kind)
	}

	var total int64
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM %s %s`, s.tableName("semantic_entries"), whereSQL)
	if err := executor.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count semantic entries: %w", err)
	}

	query := fmt.Sprintf(`SELECT id, model_id, kind, key_name, tables, spec, created_by, updated_by, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at) FROM %s %s ORDER BY id DESC LIMIT ? OFFSET ?`, s.tableName("semantic_entries"), whereSQL)
	args = append(args, options.PageSize, offset)
	rows, err := executor.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list semantic entries: %w", err)
	}
	defer rows.Close()

	list := make([]*SemanticEntryRecord, 0)
	for rows.Next() {
		var (
			record                           SemanticEntryRecord
			kindRaw, keyNameRaw              string
			tablesRaw, specRaw, createdByRaw sql.NullString
			updatedByRaw                     sql.NullString
		)
		if err := rows.Scan(
			&record.ID,
			&record.ModelID,
			&kindRaw,
			&keyNameRaw,
			&tablesRaw,
			&specRaw,
			&createdByRaw,
			&updatedByRaw,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan semantic entry: %w", err)
		}
		record.Kind = kindRaw
		record.KeyName = keyNameRaw
		record.Tables = parseJSONStringArray(tablesRaw)
		record.Spec = parseJSONRaw(specRaw)
		record.CreatedBy = createdByRaw.String
		record.UpdatedBy = updatedByRaw.String
		list = append(list, &record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate semantic entries: %w", err)
	}

	return list, total, nil
}

func (s *TenantStorageImpl) UpdateSemanticEntry(ctx context.Context, entry *SemanticEntryRecord) error {
	if entry == nil {
		return fmt.Errorf("semantic entry is required")
	}
	if entry.ID <= 0 {
		return fmt.Errorf("entry_id is required")
	}
	if entry.ModelID <= 0 {
		return fmt.Errorf("model_id is required")
	}
	if strings.TrimSpace(entry.Kind) == "" {
		return fmt.Errorf("kind is required")
	}
	if strings.TrimSpace(entry.KeyName) == "" {
		return fmt.Errorf("key_name is required")
	}
	if len(entry.Spec) == 0 {
		return fmt.Errorf("spec is required")
	}

	executor, err := s.getExecutor(ctx)
	if err != nil {
		return err
	}

	tablesJSON, err := json.Marshal(entry.Tables)
	if err != nil {
		return fmt.Errorf("marshal entry tables: %w", err)
	}

	res, err := executor.ExecContext(
		ctx,
		fmt.Sprintf(`UPDATE %s SET kind = ?, key_name = ?, tables = ?, spec = ?, updated_by = ? WHERE model_id = ? AND id = ?`, s.tableName("semantic_entries")),
		entry.Kind,
		entry.KeyName,
		string(tablesJSON),
		string(entry.Spec),
		entry.UpdatedBy,
		entry.ModelID,
		entry.ID,
	)
	if err != nil {
		if IsDuplicateKeyError(err) {
			return ErrSemanticEntryAlreadyExist
		}
		return fmt.Errorf("update semantic entry: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("semantic entry update rows_affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrSemanticEntryNotFound
	}
	return nil
}

func (s *TenantStorageImpl) DeleteSemanticEntry(ctx context.Context, modelID, entryID int64) error {
	if modelID <= 0 {
		return fmt.Errorf("model_id is required")
	}
	if entryID <= 0 {
		return fmt.Errorf("entry_id is required")
	}

	executor, err := s.getExecutor(ctx)
	if err != nil {
		return err
	}

	res, err := executor.ExecContext(
		ctx,
		fmt.Sprintf(`DELETE FROM %s WHERE model_id = ? AND id = ?`, s.tableName("semantic_entries")),
		modelID,
		entryID,
	)
	if err != nil {
		return fmt.Errorf("delete semantic entry: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("semantic entry delete rows_affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrSemanticEntryNotFound
	}
	return nil
}

func parseJSONStringArray(raw sql.NullString) []string {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw.String), &out); err != nil {
		return nil
	}
	return out
}

func parseJSONRaw(raw sql.NullString) json.RawMessage {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	return json.RawMessage(raw.String)
}
