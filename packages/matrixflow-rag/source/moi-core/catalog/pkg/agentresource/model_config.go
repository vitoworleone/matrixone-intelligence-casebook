package agentresource

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ModelConfigStatusDraft    = "draft"
	ModelConfigStatusActive   = "active"
	ModelConfigStatusDisabled = "disabled"
	ModelConfigStatusArchived = "archived"

	ModelConfigSourceSystem    = "system"
	ModelConfigSourceWorkspace = "workspace"
	ModelConfigSourceProvider  = "provider"
	ModelConfigSourceImported  = "imported"

	ModelCategoryChat       = "chat"
	ModelCategoryEmbedding  = "embedding"
	ModelCategoryRerank     = "rerank"
	ModelCategoryMultimodal = "multimodal"
	ModelCategoryParser     = "parser"

	modelConfigMaxNameLen        = 128
	modelConfigMaxDescriptionLen = 4096
)

var (
	ErrModelConfigNotFound      = errors.New("model config not found")
	ErrModelConfigAlreadyExists = errors.New("model config already exists")
	ErrInvalidModelConfig       = errors.New("invalid model config metadata")
)

type ModelConfig struct {
	ID              string            `json:"id"`
	WorkspaceID     string            `json:"workspace_id"`
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	Status          string            `json:"status"`
	SourceKind      string            `json:"source_kind"`
	ModelCategory   string            `json:"model_category"`
	ProviderRef     string            `json:"provider_ref,omitempty"`
	ModelName       string            `json:"model_name"`
	DisplayName     string            `json:"display_name,omitempty"`
	ConnectionRef   string            `json:"connection_ref,omitempty"`
	CredentialRef   string            `json:"credential_ref,omitempty"`
	Parameters      map[string]any    `json:"parameters,omitempty"`
	ParameterSchema map[string]any    `json:"parameter_schema,omitempty"`
	Capabilities    []string          `json:"capabilities,omitempty"`
	Limits          map[string]any    `json:"limits,omitempty"`
	PricingRef      string            `json:"pricing_ref,omitempty"`
	Version         int               `json:"version"`
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	Metadata        map[string]any    `json:"metadata,omitempty"`
	CreatedBy       string            `json:"created_by,omitempty"`
	UpdatedBy       string            `json:"updated_by,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type ModelConfigCreateInput struct {
	ID              string            `json:"id,omitempty"`
	WorkspaceID     string            `json:"workspace_id,omitempty"`
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	Status          string            `json:"status,omitempty"`
	SourceKind      string            `json:"source_kind,omitempty"`
	ModelCategory   string            `json:"model_category,omitempty"`
	ProviderRef     string            `json:"provider_ref,omitempty"`
	ModelName       string            `json:"model_name"`
	DisplayName     string            `json:"display_name,omitempty"`
	ConnectionRef   string            `json:"connection_ref,omitempty"`
	CredentialRef   string            `json:"credential_ref,omitempty"`
	Parameters      map[string]any    `json:"parameters,omitempty"`
	ParameterSchema map[string]any    `json:"parameter_schema,omitempty"`
	Capabilities    []string          `json:"capabilities,omitempty"`
	Limits          map[string]any    `json:"limits,omitempty"`
	PricingRef      string            `json:"pricing_ref,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	Metadata        map[string]any    `json:"metadata,omitempty"`
	UserID          string            `json:"user_id,omitempty"`
}

type ModelConfigUpdateInput struct {
	Name            *string           `json:"name,omitempty"`
	Description     *string           `json:"description,omitempty"`
	Status          *string           `json:"status,omitempty"`
	SourceKind      *string           `json:"source_kind,omitempty"`
	ModelCategory   *string           `json:"model_category,omitempty"`
	ProviderRef     *string           `json:"provider_ref,omitempty"`
	ModelName       *string           `json:"model_name,omitempty"`
	DisplayName     *string           `json:"display_name,omitempty"`
	ConnectionRef   *string           `json:"connection_ref,omitempty"`
	CredentialRef   *string           `json:"credential_ref,omitempty"`
	Parameters      map[string]any    `json:"parameters,omitempty"`
	ParameterSchema map[string]any    `json:"parameter_schema,omitempty"`
	Capabilities    *[]string         `json:"capabilities,omitempty"`
	Limits          map[string]any    `json:"limits,omitempty"`
	PricingRef      *string           `json:"pricing_ref,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
	Metadata        map[string]any    `json:"metadata,omitempty"`
	UserID          string            `json:"user_id,omitempty"`
}

type ModelConfigListFilter struct {
	WorkspaceID   string
	Status        string
	SourceKind    string
	ModelCategory string
	ProviderRef   string
	Query         string
	Limit         int
	Offset        int
}

type ModelConfigStore interface {
	CreateModelConfig(ctx context.Context, config ModelConfig) (*ModelConfig, error)
	GetModelConfig(ctx context.Context, workspaceID, configID string) (*ModelConfig, error)
	UpdateModelConfig(ctx context.Context, config ModelConfig) (*ModelConfig, error)
	ListModelConfigs(ctx context.Context, filter ModelConfigListFilter) ([]ModelConfig, int, error)
}

type ModelConfigConnectionRefStore interface {
	GetConnection(ctx context.Context, workspaceID, connectionID string) (*Connection, error)
}

type ModelConfigService struct {
	store           ModelConfigStore
	connectionStore ModelConfigConnectionRefStore
	now             func() time.Time
	newID           func() string
}

func NewModelConfigService(store ModelConfigStore) *ModelConfigService {
	service := &ModelConfigService{
		store: store,
		now:   time.Now,
		newID: func() string {
			return uuid.NewString()
		},
	}
	if connectionStore, ok := store.(ModelConfigConnectionRefStore); ok {
		service.connectionStore = connectionStore
	}
	return service
}

func (s *ModelConfigService) CreateModelConfig(ctx context.Context, input ModelConfigCreateInput) (*ModelConfig, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("model config store is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	now := s.currentTime()
	id := input.ID
	if id == "" {
		id = s.newID()
	}
	config, err := NormalizeModelConfig(ModelConfig{
		ID:              id,
		WorkspaceID:     input.WorkspaceID,
		Name:            input.Name,
		Description:     input.Description,
		Status:          input.Status,
		SourceKind:      input.SourceKind,
		ModelCategory:   input.ModelCategory,
		ProviderRef:     input.ProviderRef,
		ModelName:       input.ModelName,
		DisplayName:     input.DisplayName,
		ConnectionRef:   input.ConnectionRef,
		CredentialRef:   input.CredentialRef,
		Parameters:      input.Parameters,
		ParameterSchema: input.ParameterSchema,
		Capabilities:    input.Capabilities,
		Limits:          input.Limits,
		PricingRef:      input.PricingRef,
		Version:         1,
		Labels:          input.Labels,
		Annotations:     input.Annotations,
		Metadata:        input.Metadata,
		CreatedBy:       input.UserID,
		UpdatedBy:       input.UserID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, now)
	if err != nil {
		return nil, err
	}
	if err := s.validateModelConfigRefs(ctx, config); err != nil {
		return nil, err
	}
	return s.store.CreateModelConfig(ctx, config)
}

func (s *ModelConfigService) GetModelConfig(ctx context.Context, workspaceID, configID string) (*ModelConfig, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("model config store is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOpaqueID(workspaceID, "workspace_id", wrapInvalidModelConfigID); err != nil {
		return nil, err
	}
	if err := validateOpaqueID(configID, "model_config_id", wrapInvalidModelConfigID); err != nil {
		return nil, err
	}
	return s.store.GetModelConfig(ctx, workspaceID, configID)
}

func (s *ModelConfigService) UpdateModelConfig(ctx context.Context, workspaceID, configID string, input ModelConfigUpdateInput) (*ModelConfig, error) {
	if s == nil || s.store == nil {
		return nil, fmt.Errorf("model config store is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOpaqueID(workspaceID, "workspace_id", wrapInvalidModelConfigID); err != nil {
		return nil, err
	}
	if err := validateOpaqueID(configID, "model_config_id", wrapInvalidModelConfigID); err != nil {
		return nil, err
	}
	existing, err := s.store.GetModelConfig(ctx, workspaceID, configID)
	if err != nil {
		return nil, err
	}
	next := cloneModelConfigValue(*existing)
	if input.Name != nil {
		next.Name = *input.Name
	}
	if input.Description != nil {
		next.Description = *input.Description
	}
	if input.Status != nil {
		next.Status = *input.Status
	}
	if input.SourceKind != nil {
		next.SourceKind = *input.SourceKind
	}
	if input.ModelCategory != nil {
		next.ModelCategory = *input.ModelCategory
	}
	if input.ProviderRef != nil {
		next.ProviderRef = *input.ProviderRef
	}
	if input.ModelName != nil {
		next.ModelName = *input.ModelName
	}
	if input.DisplayName != nil {
		next.DisplayName = *input.DisplayName
	}
	if input.ConnectionRef != nil {
		next.ConnectionRef = *input.ConnectionRef
	}
	if input.CredentialRef != nil {
		next.CredentialRef = *input.CredentialRef
	}
	if input.Parameters != nil {
		next.Parameters = cloneAnyMap(input.Parameters)
	}
	if input.ParameterSchema != nil {
		next.ParameterSchema = cloneAnyMap(input.ParameterSchema)
	}
	if input.Capabilities != nil {
		next.Capabilities = append([]string(nil), (*input.Capabilities)...)
	}
	if input.Limits != nil {
		next.Limits = cloneAnyMap(input.Limits)
	}
	if input.PricingRef != nil {
		next.PricingRef = *input.PricingRef
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
	normalized, err := NormalizeModelConfig(next, next.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err := s.validateModelConfigRefs(ctx, normalized); err != nil {
		return nil, err
	}
	return s.store.UpdateModelConfig(ctx, normalized)
}

func (s *ModelConfigService) ListModelConfigs(ctx context.Context, filter ModelConfigListFilter) ([]ModelConfig, int, error) {
	if s == nil || s.store == nil {
		return nil, 0, fmt.Errorf("model config store is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOpaqueID(filter.WorkspaceID, "workspace_id", wrapInvalidModelConfigID); err != nil {
		return nil, 0, err
	}
	return s.store.ListModelConfigs(ctx, filter)
}

func (s *ModelConfigService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *ModelConfigService) validateModelConfigRefs(ctx context.Context, config ModelConfig) error {
	if containsPathSeparator(config.CredentialRef) {
		return wrapInvalidModelConfig("credential_ref must not contain path separators")
	}
	if containsPathSeparator(config.ConnectionRef) {
		return wrapInvalidModelConfig("connection_ref must not contain path separators")
	}
	if config.ConnectionRef == "" {
		return nil
	}
	if s == nil || s.connectionStore == nil {
		return wrapInvalidModelConfig("connection store is not configured")
	}
	if _, err := s.connectionStore.GetConnection(ctx, config.WorkspaceID, config.ConnectionRef); err != nil {
		if errors.Is(err, ErrConnectionNotFound) {
			return errors.Join(ErrInvalidModelConfig, ErrConnectionNotFound, errors.New("connection_ref not found"))
		}
		return err
	}
	return nil
}

func NormalizeModelConfig(input ModelConfig, now time.Time) (ModelConfig, error) {
	config := cloneModelConfigValue(input)
	config.Name = strings.TrimSpace(config.Name)
	config.Description = strings.TrimSpace(config.Description)
	config.Status = normalizeModelConfigStatus(config.Status)
	config.SourceKind = normalizeModelConfigSourceKind(config.SourceKind)
	config.ModelCategory = normalizeModelCategory(config.ModelCategory)
	config.DisplayName = strings.TrimSpace(config.DisplayName)
	config.Parameters = cloneAnyMap(config.Parameters)
	config.ParameterSchema = cloneAnyMap(config.ParameterSchema)
	config.Capabilities = append([]string(nil), config.Capabilities...)
	config.Limits = cloneAnyMap(config.Limits)
	config.Labels = cloneStringMap(config.Labels)
	config.Annotations = cloneStringMap(config.Annotations)
	config.Metadata = cloneAnyMap(config.Metadata)
	config.CreatedBy = strings.TrimSpace(config.CreatedBy)
	config.UpdatedBy = strings.TrimSpace(config.UpdatedBy)
	if config.Version <= 0 {
		config.Version = 1
	}
	if config.CreatedAt.IsZero() {
		config.CreatedAt = now
	}
	if config.UpdatedAt.IsZero() {
		config.UpdatedAt = now
	}
	return config, ValidateModelConfig(config)
}

func ValidateModelConfig(config ModelConfig) error {
	if err := validateOpaqueID(config.WorkspaceID, "workspace_id", wrapInvalidModelConfigID); err != nil {
		return err
	}
	if err := validateOpaqueID(config.ID, "id", wrapInvalidModelConfigID); err != nil {
		return err
	}
	if config.Name == "" {
		return wrapInvalidModelConfig("name is required")
	}
	if len(config.Name) > modelConfigMaxNameLen {
		return wrapInvalidModelConfig("name is too long")
	}
	if len(config.Description) > modelConfigMaxDescriptionLen {
		return wrapInvalidModelConfig("description is too long")
	}
	if config.ModelName == "" {
		return wrapInvalidModelConfig("model_name is required")
	}
	for field, value := range map[string]string{
		"provider_ref":   config.ProviderRef,
		"model_name":     config.ModelName,
		"connection_ref": config.ConnectionRef,
		"credential_ref": config.CredentialRef,
		"pricing_ref":    config.PricingRef,
	} {
		if value != "" && value != strings.TrimSpace(value) {
			return wrapInvalidModelConfig(field + " must not contain leading or trailing whitespace")
		}
	}
	if !isModelConfigStatus(config.Status) {
		return wrapInvalidModelConfig("status is invalid")
	}
	if !isModelConfigSourceKind(config.SourceKind) {
		return wrapInvalidModelConfig("source_kind is invalid")
	}
	if !isModelCategory(config.ModelCategory) {
		return wrapInvalidModelConfig("model_category is invalid")
	}
	for _, capability := range config.Capabilities {
		if capability == "" {
			return wrapInvalidModelConfig("capabilities must not contain empty values")
		}
		if capability != strings.TrimSpace(capability) {
			return wrapInvalidModelConfig("capabilities must not contain leading or trailing whitespace")
		}
	}
	if containsPathSeparator(config.ConnectionRef) || containsPathSeparator(config.CredentialRef) {
		return wrapInvalidModelConfig("connection_ref and credential_ref must not contain path separators")
	}
	if containsForbiddenMetadataKey(config.Parameters) ||
		containsForbiddenMetadataKey(config.ParameterSchema) ||
		containsForbiddenMetadataKey(config.Limits) ||
		containsForbiddenMetadataKey(config.Metadata) {
		return wrapInvalidModelConfig("model config metadata must not contain secrets or provider run/session refs")
	}
	return nil
}

func normalizeModelConfigStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return ModelConfigStatusDraft
	}
	return status
}

func isModelConfigStatus(status string) bool {
	switch status {
	case ModelConfigStatusDraft, ModelConfigStatusActive, ModelConfigStatusDisabled, ModelConfigStatusArchived:
		return true
	default:
		return false
	}
}

func normalizeModelConfigSourceKind(sourceKind string) string {
	sourceKind = strings.ToLower(strings.TrimSpace(sourceKind))
	if sourceKind == "" {
		return ModelConfigSourceWorkspace
	}
	return sourceKind
}

func isModelConfigSourceKind(sourceKind string) bool {
	switch sourceKind {
	case ModelConfigSourceSystem, ModelConfigSourceWorkspace, ModelConfigSourceProvider, ModelConfigSourceImported:
		return true
	default:
		return false
	}
}

func normalizeModelCategory(category string) string {
	category = strings.ToLower(strings.TrimSpace(category))
	if category == "" {
		return ModelCategoryChat
	}
	return category
}

func isModelCategory(category string) bool {
	switch category {
	case ModelCategoryChat, ModelCategoryEmbedding, ModelCategoryRerank, ModelCategoryMultimodal, ModelCategoryParser:
		return true
	default:
		return false
	}
}

func cloneModelConfigValue(config ModelConfig) ModelConfig {
	config.Parameters = cloneAnyMap(config.Parameters)
	config.ParameterSchema = cloneAnyMap(config.ParameterSchema)
	config.Capabilities = append([]string(nil), config.Capabilities...)
	config.Limits = cloneAnyMap(config.Limits)
	config.Labels = cloneStringMap(config.Labels)
	config.Annotations = cloneStringMap(config.Annotations)
	config.Metadata = cloneAnyMap(config.Metadata)
	return config
}

func cloneModelConfigPtr(config ModelConfig) *ModelConfig {
	cloned := cloneModelConfigValue(config)
	return &cloned
}

func wrapInvalidModelConfig(message string) error {
	return errors.Join(ErrInvalidModelConfig, errors.New(message))
}
