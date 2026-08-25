package agentresource

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (s *SQLAgentStore) CreateKnowledgeBase(ctx context.Context, kb KnowledgeBase) (*KnowledgeBase, error) {
	if s == nil || s.tm == nil {
		return nil, fmt.Errorf("knowledge base store is not configured")
	}
	if err := s.insertKnowledgeBase(ctx, s.tm.GetExecutor(ctx), kb); err != nil {
		return nil, err
	}
	return cloneKnowledgeBasePtr(kb), nil
}

func (s *SQLAgentStore) GetKnowledgeBase(ctx context.Context, workspaceID, kbID string) (*KnowledgeBase, error) {
	if s == nil || s.tm == nil {
		return nil, fmt.Errorf("knowledge base store is not configured")
	}
	return s.scanKnowledgeBase(s.tm.GetExecutor(ctx).QueryRowContext(ctx, fmt.Sprintf(`
SELECT id, workspace_id, name, description, status, source_type, catalog_asset_refs_json,
       default_retrieval_profile_ref, tags_json, owner_user_id, visibility, index_status,
       last_indexed_at, last_index_error, version, labels_json, annotations_json, metadata_json,
       created_by, updated_by, created_at, updated_at
FROM %s
	WHERE workspace_id = ? AND id = ?`, s.table("agent_resource_knowledge_bases")), workspaceID, kbID))
}

func (s *SQLAgentStore) GetKnowledgeBasesByIDs(ctx context.Context, workspaceID string, kbIDs []string) (map[string]KnowledgeBase, error) {
	if s == nil || s.tm == nil {
		return nil, fmt.Errorf("knowledge base store is not configured")
	}
	if len(kbIDs) == 0 {
		return map[string]KnowledgeBase{}, nil
	}
	placeholders := make([]string, 0, len(kbIDs))
	args := make([]any, 0, len(kbIDs)+1)
	args = append(args, workspaceID)
	for _, kbID := range kbIDs {
		placeholders = append(placeholders, "?")
		args = append(args, kbID)
	}
	rows, err := s.tm.GetExecutor(ctx).QueryContext(ctx, fmt.Sprintf(`
SELECT id, workspace_id, name, description, status, source_type, catalog_asset_refs_json,
       default_retrieval_profile_ref, tags_json, owner_user_id, visibility, index_status,
       last_indexed_at, last_index_error, version, labels_json, annotations_json, metadata_json,
       created_by, updated_by, created_at, updated_at
FROM %s
WHERE workspace_id = ? AND id IN (%s)`, s.table("agent_resource_knowledge_bases"), strings.Join(placeholders, ",")), args...)
	if err != nil {
		return nil, fmt.Errorf("get knowledge bases by ids: %w", err)
	}
	defer rows.Close()

	knowledgeBases := make(map[string]KnowledgeBase, len(kbIDs))
	for rows.Next() {
		kb, err := s.scanKnowledgeBase(rows)
		if err != nil {
			return nil, err
		}
		knowledgeBases[kb.ID] = *kb
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate knowledge bases by ids: %w", err)
	}
	return knowledgeBases, nil
}

func (s *SQLAgentStore) UpdateKnowledgeBase(ctx context.Context, kb KnowledgeBase) (*KnowledgeBase, error) {
	if s == nil || s.tm == nil {
		return nil, fmt.Errorf("knowledge base store is not configured")
	}
	if err := s.updateKnowledgeBase(ctx, s.tm.GetExecutor(ctx), kb); err != nil {
		return nil, err
	}
	return cloneKnowledgeBasePtr(kb), nil
}

func (s *SQLAgentStore) DeleteKnowledgeBase(ctx context.Context, workspaceID, kbID string) error {
	if s == nil || s.tm == nil {
		return fmt.Errorf("knowledge base store is not configured")
	}
	result, err := s.tm.GetExecutor(ctx).ExecContext(ctx, fmt.Sprintf(`
DELETE FROM %s
WHERE workspace_id = ? AND id = ?`, s.table("agent_resource_knowledge_bases")), workspaceID, kbID)
	if err != nil {
		return fmt.Errorf("delete knowledge base: %w", err)
	}
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return ErrKnowledgeBaseNotFound
	}
	return nil
}

func (s *SQLAgentStore) ListKnowledgeBases(ctx context.Context, filter KnowledgeBaseListFilter) ([]KnowledgeBase, int, error) {
	if s == nil || s.tm == nil {
		return nil, 0, fmt.Errorf("knowledge base store is not configured")
	}
	if err := validateOpaqueID(filter.WorkspaceID, "workspace_id", wrapInvalidKnowledgeBaseID); err != nil {
		return nil, 0, err
	}
	if err := validateOptionalOpaqueID(filter.OwnerUserID, "owner_user_id", wrapInvalidKnowledgeBaseID); err != nil {
		return nil, 0, err
	}
	limit, offset := normalizeResourceListPagination(filter.Limit, filter.Offset)
	queryParts := []string{"workspace_id = ?"}
	args := []any{filter.WorkspaceID}
	if filter.Name != "" {
		queryParts = append(queryParts, "name = ?")
		args = append(args, filter.Name)
	}
	if strings.TrimSpace(filter.Status) != "" {
		queryParts = append(queryParts, "status = ?")
		args = append(args, normalizeKnowledgeBaseStatus(filter.Status))
	}
	if strings.TrimSpace(filter.SourceType) != "" {
		queryParts = append(queryParts, "source_type = ?")
		args = append(args, normalizeKnowledgeBaseSourceType(filter.SourceType))
	}
	if strings.TrimSpace(filter.Visibility) != "" {
		queryParts = append(queryParts, "visibility = ?")
		args = append(args, normalizeKnowledgeBaseVisibility(filter.Visibility))
	}
	if strings.TrimSpace(filter.IndexStatus) != "" {
		queryParts = append(queryParts, "index_status = ?")
		args = append(args, normalizeKnowledgeBaseIndexStatus(filter.IndexStatus))
	}
	if filter.OwnerUserID != "" {
		queryParts = append(queryParts, "owner_user_id = ?")
		args = append(args, filter.OwnerUserID)
	}
	if strings.TrimSpace(filter.Query) != "" {
		queryParts = append(queryParts, "(name LIKE ? OR description LIKE ?)")
		needle := "%" + strings.TrimSpace(filter.Query) + "%"
		args = append(args, needle, needle)
	}
	where := strings.Join(queryParts, " AND ")
	var total int
	if err := s.tm.GetExecutor(ctx).QueryRowContext(ctx, fmt.Sprintf(`
SELECT COUNT(*) FROM %s WHERE %s`, s.table("agent_resource_knowledge_bases"), where), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count knowledge bases: %w", err)
	}
	rows, err := s.tm.GetExecutor(ctx).QueryContext(ctx, fmt.Sprintf(`
SELECT id, workspace_id, name, description, status, source_type, catalog_asset_refs_json,
       default_retrieval_profile_ref, tags_json, owner_user_id, visibility, index_status,
       last_indexed_at, last_index_error, version, labels_json, annotations_json, metadata_json,
       created_by, updated_by, created_at, updated_at
FROM %s
WHERE %s
ORDER BY updated_at DESC, id
LIMIT ? OFFSET ?`, s.table("agent_resource_knowledge_bases"), where), append(args, limit, offset)...)
	if err != nil {
		return nil, 0, fmt.Errorf("list knowledge bases: %w", err)
	}
	defer rows.Close()

	var items []KnowledgeBase
	for rows.Next() {
		kb, err := s.scanKnowledgeBase(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *kb)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate knowledge bases: %w", err)
	}
	return items, total, nil
}

func (s *SQLAgentStore) insertKnowledgeBase(ctx context.Context, exec interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, kb KnowledgeBase) error {
	encoded, err := encodeKnowledgeBaseColumns(kb)
	if err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, fmt.Sprintf(`
INSERT INTO %s (
    id, workspace_id, name, description, status, source_type, catalog_asset_refs_json,
    default_retrieval_profile_ref, tags_json, owner_user_id, visibility, index_status,
    last_indexed_at, last_index_error, version, labels_json, annotations_json, metadata_json,
    created_by, updated_by, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, s.table("agent_resource_knowledge_bases")),
		kb.ID, kb.WorkspaceID, kb.Name, kb.Description, kb.Status, kb.SourceType, encoded.catalogAssetRefs,
		kb.DefaultRetrievalProfileRef, encoded.tags, kb.OwnerUserID, kb.Visibility, kb.IndexStatus,
		nullableTime(kb.LastIndexedAt), kb.LastIndexError, kb.Version, encoded.labels, encoded.annotations, encoded.metadata,
		kb.CreatedBy, kb.UpdatedBy, kb.CreatedAt, kb.UpdatedAt,
	); err != nil {
		return mapSQLInsertError(err, ErrKnowledgeBaseAlreadyExists, "knowledge base")
	}
	return nil
}

func (s *SQLAgentStore) updateKnowledgeBase(ctx context.Context, exec interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, kb KnowledgeBase) error {
	encoded, err := encodeKnowledgeBaseColumns(kb)
	if err != nil {
		return err
	}
	result, err := exec.ExecContext(ctx, fmt.Sprintf(`
UPDATE %s SET
    name = ?, description = ?, status = ?, source_type = ?, catalog_asset_refs_json = ?,
    default_retrieval_profile_ref = ?, tags_json = ?, owner_user_id = ?, visibility = ?, index_status = ?,
    last_indexed_at = ?, last_index_error = ?, version = ?, labels_json = ?, annotations_json = ?, metadata_json = ?,
    updated_by = ?, updated_at = ?
WHERE workspace_id = ? AND id = ?`, s.table("agent_resource_knowledge_bases")),
		kb.Name, kb.Description, kb.Status, kb.SourceType, encoded.catalogAssetRefs,
		kb.DefaultRetrievalProfileRef, encoded.tags, kb.OwnerUserID, kb.Visibility, kb.IndexStatus,
		nullableTime(kb.LastIndexedAt), kb.LastIndexError, kb.Version, encoded.labels, encoded.annotations, encoded.metadata,
		kb.UpdatedBy, kb.UpdatedAt, kb.WorkspaceID, kb.ID,
	)
	if err != nil {
		return fmt.Errorf("update knowledge base: %w", err)
	}
	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return ErrKnowledgeBaseNotFound
	}
	return nil
}

func (s *SQLAgentStore) scanKnowledgeBase(row interface {
	Scan(dest ...any) error
}) (*KnowledgeBase, error) {
	var kb KnowledgeBase
	var catalogAssetRefs, tags, labels, annotations, metadata sql.NullString
	var lastIndexedAt sql.NullTime
	if err := row.Scan(
		&kb.ID, &kb.WorkspaceID, &kb.Name, &kb.Description, &kb.Status, &kb.SourceType, &catalogAssetRefs,
		&kb.DefaultRetrievalProfileRef, &tags, &kb.OwnerUserID, &kb.Visibility, &kb.IndexStatus,
		&lastIndexedAt, &kb.LastIndexError, &kb.Version, &labels, &annotations, &metadata,
		&kb.CreatedBy, &kb.UpdatedBy, &kb.CreatedAt, &kb.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrKnowledgeBaseNotFound
		}
		return nil, fmt.Errorf("scan knowledge base: %w", err)
	}
	if lastIndexedAt.Valid {
		value := lastIndexedAt.Time
		kb.LastIndexedAt = &value
	}
	if err := decodeJSON(catalogAssetRefs, &kb.CatalogAssetRefs); err != nil {
		return nil, err
	}
	if err := decodeJSON(tags, &kb.Tags); err != nil {
		return nil, err
	}
	if err := decodeJSON(labels, &kb.Labels); err != nil {
		return nil, err
	}
	if err := decodeJSON(annotations, &kb.Annotations); err != nil {
		return nil, err
	}
	if err := decodeJSON(metadata, &kb.Metadata); err != nil {
		return nil, err
	}
	return cloneKnowledgeBasePtr(kb), nil
}

type encodedKnowledgeBaseColumns struct {
	catalogAssetRefs string
	tags             string
	labels           string
	annotations      string
	metadata         string
}

func encodeKnowledgeBaseColumns(kb KnowledgeBase) (encodedKnowledgeBaseColumns, error) {
	var out encodedKnowledgeBaseColumns
	var err error
	if out.catalogAssetRefs, err = encodeJSON(kb.CatalogAssetRefs); err != nil {
		return out, err
	}
	if out.tags, err = encodeJSON(kb.Tags); err != nil {
		return out, err
	}
	if out.labels, err = encodeJSON(kb.Labels); err != nil {
		return out, err
	}
	if out.annotations, err = encodeJSON(kb.Annotations); err != nil {
		return out, err
	}
	if out.metadata, err = encodeJSON(kb.Metadata); err != nil {
		return out, err
	}
	return out, nil
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}
