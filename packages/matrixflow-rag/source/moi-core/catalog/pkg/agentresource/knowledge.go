package agentresource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	agenttools "github.com/matrixflow/moi-core/agent-tools"
)

const (
	KnowledgeBaseStatusDraft    = "draft"
	KnowledgeBaseStatusActive   = "active"
	KnowledgeBaseStatusDisabled = "disabled"
	KnowledgeBaseStatusArchived = "archived"

	KnowledgeBaseSourceManual          = "manual"
	KnowledgeBaseSourceCatalogResource = "catalog_resource"
	KnowledgeBaseSourceFileCollection  = "file_collection"
	KnowledgeBaseSourceExternalDrive   = "external_drive"
	KnowledgeBaseSourceConnectorExport = "connector_export"

	KnowledgeBaseVisibilityPrivate   = "private"
	KnowledgeBaseVisibilityWorkspace = "workspace"

	KnowledgeBaseIndexStatusEmpty    = "empty"
	KnowledgeBaseIndexStatusIndexing = "indexing"
	KnowledgeBaseIndexStatusReady    = "ready"
	KnowledgeBaseIndexStatusFailed   = "failed"

	knowledgeBaseMaxNameLen        = 128
	knowledgeBaseMaxDescriptionLen = 4096
	knowledgeBaseMaxIndexErrorLen  = 4096

	KnowledgeBaseMetadataRuntimeToolRefsKey = "runtime_tool_refs"
)

var (
	ErrKnowledgeBaseNotFound      = errors.New("knowledge base not found")
	ErrKnowledgeBaseAlreadyExists = errors.New("knowledge base already exists")
	ErrInvalidKnowledgeBase       = errors.New("invalid knowledge base metadata")
)

type KnowledgeBase struct {
	ID                         string                     `json:"id"`
	WorkspaceID                string                     `json:"workspace_id"`
	Name                       string                     `json:"name"`
	Description                string                     `json:"description,omitempty"`
	Status                     string                     `json:"status"`
	SourceType                 string                     `json:"source_type"`
	CatalogAssetRefs           []KnowledgeCatalogAssetRef `json:"catalog_asset_refs,omitempty"`
	DefaultRetrievalProfileRef string                     `json:"default_retrieval_profile_ref,omitempty"`
	Tags                       []string                   `json:"tags,omitempty"`
	OwnerUserID                string                     `json:"owner_user_id,omitempty"`
	Visibility                 string                     `json:"visibility"`
	IndexStatus                string                     `json:"index_status"`
	LastIndexedAt              *time.Time                 `json:"last_indexed_at,omitempty"`
	LastIndexError             string                     `json:"last_index_error,omitempty"`
	Version                    int                        `json:"version"`
	Labels                     map[string]string          `json:"labels,omitempty"`
	Annotations                map[string]string          `json:"annotations,omitempty"`
	Metadata                   map[string]any             `json:"metadata,omitempty"`
	CreatedBy                  string                     `json:"created_by,omitempty"`
	UpdatedBy                  string                     `json:"updated_by,omitempty"`
	CreatedAt                  time.Time                  `json:"created_at"`
	UpdatedAt                  time.Time                  `json:"updated_at"`
}

type KnowledgeCatalogAssetRef struct {
	Type    string         `json:"type"`
	ID      string         `json:"id,omitempty"`
	URI     string         `json:"uri,omitempty"`
	Version string         `json:"version,omitempty"`
	Role    string         `json:"role,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

type KnowledgeBaseCreateInput struct {
	ID                         string                     `json:"id,omitempty"`
	WorkspaceID                string                     `json:"workspace_id,omitempty"`
	Name                       string                     `json:"name"`
	Description                string                     `json:"description,omitempty"`
	Status                     string                     `json:"status,omitempty"`
	SourceType                 string                     `json:"source_type,omitempty"`
	CatalogAssetRefs           []KnowledgeCatalogAssetRef `json:"catalog_asset_refs,omitempty"`
	DefaultRetrievalProfileRef string                     `json:"default_retrieval_profile_ref,omitempty"`
	Tags                       []string                   `json:"tags,omitempty"`
	OwnerUserID                string                     `json:"owner_user_id,omitempty"`
	Visibility                 string                     `json:"visibility,omitempty"`
	IndexStatus                string                     `json:"index_status,omitempty"`
	LastIndexedAt              *time.Time                 `json:"last_indexed_at,omitempty"`
	LastIndexError             string                     `json:"last_index_error,omitempty"`
	Labels                     map[string]string          `json:"labels,omitempty"`
	Annotations                map[string]string          `json:"annotations,omitempty"`
	Metadata                   map[string]any             `json:"metadata,omitempty"`
	UserID                     string                     `json:"user_id,omitempty"`
}

type KnowledgeBaseUpdateInput struct {
	Name                       *string                     `json:"name,omitempty"`
	Description                *string                     `json:"description,omitempty"`
	Status                     *string                     `json:"status,omitempty"`
	SourceType                 *string                     `json:"source_type,omitempty"`
	CatalogAssetRefs           *[]KnowledgeCatalogAssetRef `json:"catalog_asset_refs,omitempty"`
	DefaultRetrievalProfileRef *string                     `json:"default_retrieval_profile_ref,omitempty"`
	Tags                       *[]string                   `json:"tags,omitempty"`
	OwnerUserID                *string                     `json:"owner_user_id,omitempty"`
	Visibility                 *string                     `json:"visibility,omitempty"`
	IndexStatus                *string                     `json:"index_status,omitempty"`
	LastIndexedAt              *time.Time                  `json:"last_indexed_at,omitempty"`
	LastIndexError             *string                     `json:"last_index_error,omitempty"`
	Labels                     map[string]string           `json:"labels,omitempty"`
	Annotations                map[string]string           `json:"annotations,omitempty"`
	Metadata                   map[string]any              `json:"metadata,omitempty"`
	UserID                     string                      `json:"user_id,omitempty"`
}

type KnowledgeBaseListFilter struct {
	WorkspaceID string
	Name        string
	Status      string
	SourceType  string
	Visibility  string
	IndexStatus string
	OwnerUserID string
	Query       string
	Limit       int
	Offset      int
}

type KnowledgeBaseStore interface {
	CreateKnowledgeBase(ctx context.Context, kb KnowledgeBase) (*KnowledgeBase, error)
	GetKnowledgeBase(ctx context.Context, workspaceID, kbID string) (*KnowledgeBase, error)
	UpdateKnowledgeBase(ctx context.Context, kb KnowledgeBase) (*KnowledgeBase, error)
	DeleteKnowledgeBase(ctx context.Context, workspaceID, kbID string) error
	ListKnowledgeBases(ctx context.Context, filter KnowledgeBaseListFilter) ([]KnowledgeBase, int, error)
}

type KnowledgeBaseService struct {
	store                  KnowledgeBaseStore
	now                    func() time.Time
	newID                  func() string
	defaultRuntimeToolRefs []AgentBindingResourceRef
}

type KnowledgeBaseServiceOption func(*KnowledgeBaseService)

func WithKnowledgeBaseDefaultRuntimeToolRefs(refs []AgentBindingResourceRef) KnowledgeBaseServiceOption {
	return func(s *KnowledgeBaseService) {
		s.defaultRuntimeToolRefs = cloneAgentBindingResourceRefs(refs)
	}
}

func NewKnowledgeBaseService(store KnowledgeBaseStore, opts ...KnowledgeBaseServiceOption) *KnowledgeBaseService {
	service := &KnowledgeBaseService{
		store: store,
		now:   time.Now,
		newID: func() string {
			return uuid.NewString()
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(service)
		}
	}
	return service
}

func (s *KnowledgeBaseService) CreateKnowledgeBase(ctx context.Context, input KnowledgeBaseCreateInput) (*KnowledgeBase, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("knowledge base store is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	now := s.currentTime()
	id := input.ID
	if id == "" {
		id = s.newID()
	}
	metadata := cloneAnyMap(input.Metadata)
	if _, ok := metadata[KnowledgeBaseMetadataRuntimeToolRefsKey]; !ok && len(s.defaultRuntimeToolRefs) > 0 {
		if metadata == nil {
			metadata = map[string]any{}
		}
		metadata[KnowledgeBaseMetadataRuntimeToolRefsKey] = KnowledgeBaseRuntimeToolRefsMetadata(s.defaultRuntimeToolRefs)
	}
	kb, err := NormalizeKnowledgeBase(KnowledgeBase{
		ID:                         id,
		WorkspaceID:                input.WorkspaceID,
		Name:                       input.Name,
		Description:                input.Description,
		Status:                     input.Status,
		SourceType:                 input.SourceType,
		CatalogAssetRefs:           input.CatalogAssetRefs,
		DefaultRetrievalProfileRef: input.DefaultRetrievalProfileRef,
		Tags:                       input.Tags,
		OwnerUserID:                firstNonEmpty(input.OwnerUserID, input.UserID),
		Visibility:                 input.Visibility,
		IndexStatus:                input.IndexStatus,
		LastIndexedAt:              input.LastIndexedAt,
		LastIndexError:             input.LastIndexError,
		Version:                    1,
		Labels:                     input.Labels,
		Annotations:                input.Annotations,
		Metadata:                   metadata,
		CreatedBy:                  input.UserID,
		UpdatedBy:                  input.UserID,
		CreatedAt:                  now,
		UpdatedAt:                  now,
	}, now)
	if err != nil {
		return nil, err
	}
	return s.store.CreateKnowledgeBase(ctx, kb)
}

func (s *KnowledgeBaseService) GetKnowledgeBase(ctx context.Context, workspaceID, kbID string) (*KnowledgeBase, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("knowledge base store is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOpaqueID(workspaceID, "workspace_id", wrapInvalidKnowledgeBaseID); err != nil {
		return nil, err
	}
	if err := validateOpaqueID(kbID, "knowledge_base_id", wrapInvalidKnowledgeBaseID); err != nil {
		return nil, err
	}
	return s.store.GetKnowledgeBase(ctx, workspaceID, kbID)
}

func (s *KnowledgeBaseService) UpdateKnowledgeBase(ctx context.Context, workspaceID, kbID string, input KnowledgeBaseUpdateInput) (*KnowledgeBase, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("knowledge base store is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOpaqueID(workspaceID, "workspace_id", wrapInvalidKnowledgeBaseID); err != nil {
		return nil, err
	}
	if err := validateOpaqueID(kbID, "knowledge_base_id", wrapInvalidKnowledgeBaseID); err != nil {
		return nil, err
	}
	existing, err := s.store.GetKnowledgeBase(ctx, workspaceID, kbID)
	if err != nil {
		return nil, err
	}
	next := cloneKnowledgeBaseValue(*existing)
	if input.Name != nil {
		next.Name = *input.Name
	}
	if input.Description != nil {
		next.Description = *input.Description
	}
	if input.Status != nil {
		next.Status = *input.Status
	}
	if input.SourceType != nil {
		next.SourceType = *input.SourceType
	}
	if input.CatalogAssetRefs != nil {
		next.CatalogAssetRefs = cloneKnowledgeCatalogAssetRefs(*input.CatalogAssetRefs)
	}
	if input.DefaultRetrievalProfileRef != nil {
		next.DefaultRetrievalProfileRef = *input.DefaultRetrievalProfileRef
	}
	if input.Tags != nil {
		next.Tags = append([]string(nil), (*input.Tags)...)
	}
	if input.OwnerUserID != nil {
		next.OwnerUserID = *input.OwnerUserID
	}
	if input.Visibility != nil {
		next.Visibility = *input.Visibility
	}
	if input.IndexStatus != nil {
		next.IndexStatus = *input.IndexStatus
	}
	if input.LastIndexedAt != nil {
		lastIndexedAt := *input.LastIndexedAt
		next.LastIndexedAt = &lastIndexedAt
	}
	if input.LastIndexError != nil {
		next.LastIndexError = *input.LastIndexError
	}
	if input.Labels != nil {
		next.Labels = cloneStringMap(input.Labels)
	}
	if input.Annotations != nil {
		next.Annotations = cloneStringMap(input.Annotations)
	}
	if input.Metadata != nil {
		next.Metadata = cloneAnyMap(input.Metadata)
	}
	next.Version++
	next.UpdatedBy = strings.TrimSpace(input.UserID)
	next.UpdatedAt = s.currentTime()
	normalized, err := NormalizeKnowledgeBase(next, next.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s.store.UpdateKnowledgeBase(ctx, normalized)
}

func (s *KnowledgeBaseService) ListKnowledgeBases(ctx context.Context, filter KnowledgeBaseListFilter) ([]KnowledgeBase, int, error) {
	if s == nil || s.store == nil {
		return nil, 0, fmt.Errorf("knowledge base store is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOpaqueID(filter.WorkspaceID, "workspace_id", wrapInvalidKnowledgeBaseID); err != nil {
		return nil, 0, err
	}
	if err := validateOptionalOpaqueID(filter.OwnerUserID, "owner_user_id", wrapInvalidKnowledgeBaseID); err != nil {
		return nil, 0, err
	}
	return s.store.ListKnowledgeBases(ctx, filter)
}

func (s *KnowledgeBaseService) DeleteKnowledgeBase(ctx context.Context, workspaceID, kbID string) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("knowledge base store is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOpaqueID(workspaceID, "workspace_id", wrapInvalidKnowledgeBaseID); err != nil {
		return err
	}
	if err := validateOpaqueID(kbID, "knowledge_base_id", wrapInvalidKnowledgeBaseID); err != nil {
		return err
	}
	return s.store.DeleteKnowledgeBase(ctx, workspaceID, kbID)
}

func (s *KnowledgeBaseService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func NormalizeKnowledgeBase(input KnowledgeBase, now time.Time) (KnowledgeBase, error) {
	kb := cloneKnowledgeBaseValue(input)
	kb.Name = strings.TrimSpace(kb.Name)
	kb.Description = strings.TrimSpace(kb.Description)
	kb.Status = normalizeKnowledgeBaseStatus(kb.Status)
	kb.SourceType = normalizeKnowledgeBaseSourceType(kb.SourceType)
	kb.CatalogAssetRefs = cloneKnowledgeCatalogAssetRefs(kb.CatalogAssetRefs)
	kb.Tags = append([]string(nil), kb.Tags...)
	kb.OwnerUserID = strings.TrimSpace(kb.OwnerUserID)
	kb.Visibility = normalizeKnowledgeBaseVisibility(kb.Visibility)
	kb.IndexStatus = normalizeKnowledgeBaseIndexStatus(kb.IndexStatus)
	kb.LastIndexError = strings.TrimSpace(kb.LastIndexError)
	kb.Labels = cloneStringMap(kb.Labels)
	kb.Annotations = cloneStringMap(kb.Annotations)
	kb.Metadata = cloneAnyMap(kb.Metadata)
	kb.CreatedBy = strings.TrimSpace(kb.CreatedBy)
	kb.UpdatedBy = strings.TrimSpace(kb.UpdatedBy)
	if kb.LastIndexedAt != nil {
		lastIndexedAt := *kb.LastIndexedAt
		kb.LastIndexedAt = &lastIndexedAt
	}
	if kb.Version <= 0 {
		kb.Version = 1
	}
	if kb.CreatedAt.IsZero() {
		kb.CreatedAt = now
	}
	if kb.UpdatedAt.IsZero() {
		kb.UpdatedAt = now
	}
	return kb, ValidateKnowledgeBase(kb)
}

func ValidateKnowledgeBase(kb KnowledgeBase) error {
	if err := validateOpaqueID(kb.WorkspaceID, "workspace_id", wrapInvalidKnowledgeBaseID); err != nil {
		return err
	}
	if err := validateOpaqueID(kb.ID, "id", wrapInvalidKnowledgeBaseID); err != nil {
		return err
	}
	if kb.Name == "" {
		return wrapInvalidKnowledgeBase("name is required")
	}
	if len(kb.Name) > knowledgeBaseMaxNameLen {
		return wrapInvalidKnowledgeBase("name is too long")
	}
	if len(kb.Description) > knowledgeBaseMaxDescriptionLen {
		return wrapInvalidKnowledgeBase("description is too long")
	}
	if len(kb.LastIndexError) > knowledgeBaseMaxIndexErrorLen {
		return wrapInvalidKnowledgeBase("last_index_error is too long")
	}
	if !isKnowledgeBaseStatus(kb.Status) {
		return wrapInvalidKnowledgeBase("status is invalid")
	}
	if !isKnowledgeBaseSourceType(kb.SourceType) {
		return wrapInvalidKnowledgeBase("source_type is invalid")
	}
	if !isKnowledgeBaseVisibility(kb.Visibility) {
		return wrapInvalidKnowledgeBase("visibility is invalid")
	}
	if !isKnowledgeBaseIndexStatus(kb.IndexStatus) {
		return wrapInvalidKnowledgeBase("index_status is invalid")
	}
	if containsPathSeparator(kb.OwnerUserID) {
		return wrapInvalidKnowledgeBase("owner_user_id must not contain path separators")
	}
	if kb.DefaultRetrievalProfileRef != "" && kb.DefaultRetrievalProfileRef != strings.TrimSpace(kb.DefaultRetrievalProfileRef) {
		return wrapInvalidKnowledgeBase("default_retrieval_profile_ref must not contain leading or trailing whitespace")
	}
	if containsForbiddenMetadataKey(kb.Metadata) {
		return wrapInvalidKnowledgeBase("knowledge base metadata must not contain secrets or provider run/session refs")
	}
	if _, err := KnowledgeBaseRuntimeToolRefs(kb); err != nil {
		return err
	}
	for _, ref := range kb.CatalogAssetRefs {
		for field, value := range map[string]string{
			"catalog_asset_refs.type":    ref.Type,
			"catalog_asset_refs.id":      ref.ID,
			"catalog_asset_refs.uri":     ref.URI,
			"catalog_asset_refs.version": ref.Version,
			"catalog_asset_refs.role":    ref.Role,
		} {
			if value != "" && value != strings.TrimSpace(value) {
				return wrapInvalidKnowledgeBase(field + " must not contain leading or trailing whitespace")
			}
		}
		if ref.Type == "" {
			return wrapInvalidKnowledgeBase("catalog_asset_refs.type is required")
		}
		if ref.ID == "" && ref.URI == "" {
			return wrapInvalidKnowledgeBase("catalog_asset_refs.id or uri is required")
		}
		if containsPathSeparator(ref.ID) {
			return wrapInvalidKnowledgeBase("catalog_asset_refs.id must not contain path separators")
		}
		if ref.URI != "" && uriContainsSecret(ref.URI) {
			return wrapInvalidKnowledgeBase("catalog_asset_refs.uri must not contain credentials or secret query parameters")
		}
		if containsForbiddenMetadataKey(ref.Config) {
			return wrapInvalidKnowledgeBase("catalog_asset_refs.config must not contain secrets or provider run/session refs")
		}
	}
	return nil
}

func KnowledgeBaseRuntimeToolRefs(kb KnowledgeBase) ([]AgentBindingResourceRef, error) {
	if kb.Metadata == nil {
		return nil, nil
	}
	raw, ok := kb.Metadata[KnowledgeBaseMetadataRuntimeToolRefsKey]
	if !ok || raw == nil {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, wrapInvalidKnowledgeBase("metadata.runtime_tool_refs must be a JSON array")
	}
	var refs []AgentBindingResourceRef
	if err := json.Unmarshal(data, &refs); err != nil {
		return nil, wrapInvalidKnowledgeBase("metadata.runtime_tool_refs must be a JSON array of tool refs")
	}
	for _, ref := range refs {
		if ref.ID == "" {
			return nil, wrapInvalidKnowledgeBase("metadata.runtime_tool_refs id is required")
		}
		if ref.ID != strings.TrimSpace(ref.ID) {
			return nil, wrapInvalidKnowledgeBase("metadata.runtime_tool_refs id must not contain leading or trailing whitespace")
		}
		if containsPathSeparator(ref.ID) {
			return nil, wrapInvalidKnowledgeBase("metadata.runtime_tool_refs id must not contain path separators")
		}
		if ref.WorkspaceID == "" {
			return nil, wrapInvalidKnowledgeBase("metadata.runtime_tool_refs workspace_id is required")
		}
		if ref.WorkspaceID != strings.TrimSpace(ref.WorkspaceID) {
			return nil, wrapInvalidKnowledgeBase("metadata.runtime_tool_refs workspace_id must not contain leading or trailing whitespace")
		}
		if containsPathSeparator(ref.WorkspaceID) {
			return nil, wrapInvalidKnowledgeBase("metadata.runtime_tool_refs workspace_id must not contain path separators")
		}
		if ref.Kind != "" && ref.Kind != "tool" {
			return nil, wrapInvalidKnowledgeBase("metadata.runtime_tool_refs kind must be tool")
		}
		if ref.Status != "" {
			return nil, wrapInvalidKnowledgeBase("metadata.runtime_tool_refs status must be empty")
		}
		if containsForbiddenMetadataKey(ref.Config) {
			return nil, wrapInvalidKnowledgeBase("metadata.runtime_tool_refs config must not contain secrets or provider run/session refs")
		}
	}
	return cloneAgentBindingResourceRefs(refs), nil
}

func DefaultKnowledgeBaseRuntimeToolRefs(toolWorkspaceID string) []AgentBindingResourceRef {
	refs := make([]AgentBindingResourceRef, 0)
	for _, definition := range agenttools.BundledToolDefinitions() {
		if definition.Metadata[agenttools.MetadataBundleKey] != agenttools.ToolBundleKnowledge {
			continue
		}
		if !agenttools.IsRuntimeExecutableKind(runtimeExecutableKindFromDefinition(definition)) {
			continue
		}
		if !knowledgeBaseRuntimeToolRefAllowed(definition.ID) {
			continue
		}
		refs = append(refs, AgentBindingResourceRef{
			ID:          definition.ID,
			WorkspaceID: toolWorkspaceID,
			Kind:        "tool",
		})
	}
	for _, toolID := range []string{
		agenttools.ToolKindReadArtifact,
		agenttools.ToolKindReadArtifactPage,
		agenttools.ToolKindWriteFile,
		agenttools.ToolKindReadFile,
	} {
		refs = append(refs, AgentBindingResourceRef{
			ID:          toolID,
			WorkspaceID: toolWorkspaceID,
			Kind:        "tool",
		})
	}
	return dedupeKnowledgeBaseRuntimeToolRefs(refs)
}

func knowledgeBaseRuntimeToolRefAllowed(toolID string) bool {
	switch strings.TrimSpace(toolID) {
	case agenttools.ToolKindSubmitFinalAnswer, agenttools.ToolKindComputeResultTable:
		return false
	default:
		return true
	}
}

func runtimeExecutableKindFromDefinition(definition agenttools.ToolDefinition) string {
	for _, value := range []string{
		stringFromAny(definition.Metadata[agenttools.MetadataRuntimeToolKindKey]),
		stringFromAny(definition.Metadata[agenttools.MetadataRegistryIDKey]),
		definition.ID,
	} {
		if value == "" {
			continue
		}
		if value != strings.TrimSpace(value) {
			continue
		}
		if agenttools.IsRuntimeExecutableKind(value) {
			return value
		}
	}
	return ""
}

func KnowledgeBaseRuntimeToolRefsMetadata(refs []AgentBindingResourceRef) []map[string]any {
	if len(refs) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(refs))
	for _, ref := range refs {
		out = append(out, compactAnyMap(map[string]any{
			"id":           ref.ID,
			"workspace_id": ref.WorkspaceID,
			"kind":         ref.Kind,
			"config":       cloneAnyMap(ref.Config),
		}))
	}
	return out
}

func dedupeKnowledgeBaseRuntimeToolRefs(refs []AgentBindingResourceRef) []AgentBindingResourceRef {
	if len(refs) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]AgentBindingResourceRef, 0, len(refs))
	for _, ref := range refs {
		if ref.ID == "" || ref.WorkspaceID == "" {
			continue
		}
		key := ref.WorkspaceID + "\x00" + ref.ID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func normalizeKnowledgeBaseStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return KnowledgeBaseStatusDraft
	}
	return status
}

func isKnowledgeBaseStatus(status string) bool {
	switch status {
	case KnowledgeBaseStatusDraft, KnowledgeBaseStatusActive, KnowledgeBaseStatusDisabled, KnowledgeBaseStatusArchived:
		return true
	default:
		return false
	}
}

func normalizeKnowledgeBaseSourceType(sourceType string) string {
	sourceType = strings.ToLower(strings.TrimSpace(sourceType))
	if sourceType == "" {
		return KnowledgeBaseSourceManual
	}
	return sourceType
}

func isKnowledgeBaseSourceType(sourceType string) bool {
	switch sourceType {
	case KnowledgeBaseSourceManual, KnowledgeBaseSourceCatalogResource, KnowledgeBaseSourceFileCollection, KnowledgeBaseSourceExternalDrive, KnowledgeBaseSourceConnectorExport:
		return true
	default:
		return false
	}
}

func normalizeKnowledgeBaseVisibility(visibility string) string {
	visibility = strings.ToLower(strings.TrimSpace(visibility))
	if visibility == "" {
		return KnowledgeBaseVisibilityPrivate
	}
	return visibility
}

func isKnowledgeBaseVisibility(visibility string) bool {
	switch visibility {
	case KnowledgeBaseVisibilityPrivate, KnowledgeBaseVisibilityWorkspace:
		return true
	default:
		return false
	}
}

func normalizeKnowledgeBaseIndexStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return KnowledgeBaseIndexStatusEmpty
	}
	return status
}

func isKnowledgeBaseIndexStatus(status string) bool {
	switch status {
	case KnowledgeBaseIndexStatusEmpty, KnowledgeBaseIndexStatusIndexing, KnowledgeBaseIndexStatusReady, KnowledgeBaseIndexStatusFailed:
		return true
	default:
		return false
	}
}

func cloneKnowledgeBaseValue(kb KnowledgeBase) KnowledgeBase {
	kb.CatalogAssetRefs = cloneKnowledgeCatalogAssetRefs(kb.CatalogAssetRefs)
	kb.Tags = append([]string(nil), kb.Tags...)
	if kb.LastIndexedAt != nil {
		lastIndexedAt := *kb.LastIndexedAt
		kb.LastIndexedAt = &lastIndexedAt
	}
	kb.Labels = cloneStringMap(kb.Labels)
	kb.Annotations = cloneStringMap(kb.Annotations)
	kb.Metadata = cloneAnyMap(kb.Metadata)
	return kb
}

func cloneKnowledgeBasePtr(kb KnowledgeBase) *KnowledgeBase {
	cloned := cloneKnowledgeBaseValue(kb)
	return &cloned
}

func cloneKnowledgeCatalogAssetRefs(input []KnowledgeCatalogAssetRef) []KnowledgeCatalogAssetRef {
	if len(input) == 0 {
		return nil
	}
	out := make([]KnowledgeCatalogAssetRef, 0, len(input))
	for _, ref := range input {
		ref.Config = cloneAnyMap(ref.Config)
		out = append(out, ref)
	}
	return out
}

func wrapInvalidKnowledgeBase(message string) error {
	return errors.Join(ErrInvalidKnowledgeBase, errors.New(message))
}
