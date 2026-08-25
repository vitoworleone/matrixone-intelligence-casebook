// Package moi provides the Go SDK for moi-core services.
package moi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	internalhttp "github.com/matrixflow/moi-core/go-sdk/internal/http"
)

// SemanticModelService provides semantic model APIs.
type SemanticModelService struct {
	http        *internalhttp.Client
	workspaceID string
}

func (s *SemanticModelService) base() string {
	return fmt.Sprintf("/api/v1/workspaces/%s/semantic-models", s.workspaceID)
}

// SemanticModel represents a semantic model.
type SemanticModel struct {
	ID          int64           `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Tables      json.RawMessage `json:"tables"`
	Files       json.RawMessage `json:"files,omitempty"`
	CreatedBy   string          `json:"created_by,omitempty"`
	UpdatedBy   string          `json:"updated_by,omitempty"`
	CreatedAt   int64           `json:"created_at"`
	UpdatedAt   int64           `json:"updated_at"`
}

// SemanticEntry represents a semantic model entry.
type SemanticEntry struct {
	ID        int64           `json:"id"`
	ModelID   int64           `json:"model_id"`
	Kind      string          `json:"kind"`
	Key       string          `json:"key"`
	Tables    []string        `json:"tables,omitempty"`
	Spec      json.RawMessage `json:"spec"`
	CreatedBy string          `json:"created_by,omitempty"`
	UpdatedBy string          `json:"updated_by,omitempty"`
	CreatedAt int64           `json:"created_at"`
	UpdatedAt int64           `json:"updated_at"`
}

// SemanticModelUpsertRequest is used by semantic model create/update.
type SemanticModelUpsertRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Tables      json.RawMessage `json:"tables,omitempty"`
	Files       json.RawMessage `json:"files,omitempty"`

	knowledgeBaseDatabaseDisplayName *knowledgeBaseDatabaseDisplayNameRequest
}

type knowledgeBaseDatabaseDisplayNameRequest struct {
	DatabaseID  int64  `json:"database_id"`
	DisplayName string `json:"display_name"`
}

// SetKnowledgeBaseDatabaseDisplayName attaches the backend-owned database display name update.
// Catalog accepts this field only from authenticated Backend executions.
func (r *SemanticModelUpsertRequest) SetKnowledgeBaseDatabaseDisplayName(databaseID int64, displayName string) {
	if r == nil {
		return
	}
	r.knowledgeBaseDatabaseDisplayName = &knowledgeBaseDatabaseDisplayNameRequest{
		DatabaseID:  databaseID,
		DisplayName: displayName,
	}
}

func (r SemanticModelUpsertRequest) MarshalJSON() ([]byte, error) {
	type semanticModelUpsertRequestJSON struct {
		Name                             string                                   `json:"name"`
		Description                      string                                   `json:"description,omitempty"`
		Tables                           json.RawMessage                          `json:"tables,omitempty"`
		Files                            json.RawMessage                          `json:"files,omitempty"`
		KnowledgeBaseDatabaseDisplayName *knowledgeBaseDatabaseDisplayNameRequest `json:"knowledge_base_database_display_name,omitempty"`
	}
	return json.Marshal(semanticModelUpsertRequestJSON{
		Name:                             r.Name,
		Description:                      r.Description,
		Tables:                           r.Tables,
		Files:                            r.Files,
		KnowledgeBaseDatabaseDisplayName: r.knowledgeBaseDatabaseDisplayName,
	})
}

// SemanticEntryUpsertRequest is used by semantic entry create/update.
type SemanticEntryUpsertRequest struct {
	Kind   string          `json:"kind"`
	Key    string          `json:"key"`
	Tables []string        `json:"tables,omitempty"`
	Spec   json.RawMessage `json:"spec"`
}

// SemanticModelImportRequest is used by semantic model import.
type SemanticModelImportRequest struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description,omitempty"`
	Tables      json.RawMessage              `json:"tables"`
	Files       json.RawMessage              `json:"files,omitempty"`
	Entries     []SemanticEntryUpsertRequest `json:"entries,omitempty"`
}

// SemanticModelImportBatchResponse is the batch import response.
type SemanticModelImportBatchResponse struct {
	Items []*SemanticModel `json:"items"`
	Total int64            `json:"total"`
}

// SemanticModelListResponse is the list semantic models response.
type SemanticModelListResponse struct {
	Items         []*SemanticModel `json:"items"`
	Total         int64            `json:"total"`
	NextPageToken string           `json:"next_page_token,omitempty"`
}

// SemanticModelTagStat is an aggregated semantic model tag count.
type SemanticModelTagStat struct {
	Tag   string `json:"tag"`
	Count int64  `json:"count"`
}

// SemanticModelTagListResponse is the list semantic model tags response.
type SemanticModelTagListResponse struct {
	Items []SemanticModelTagStat `json:"items"`
}

// SemanticEntryListResponse is the list semantic entries response.
type SemanticEntryListResponse struct {
	Items         []*SemanticEntry `json:"items"`
	Total         int64            `json:"total"`
	NextPageToken string           `json:"next_page_token,omitempty"`
}

// SemanticModelExportResponse is the export response.
type SemanticModelExportResponse struct {
	Model   *SemanticModel   `json:"model"`
	Entries []*SemanticEntry `json:"entries"`
}

// SemanticModelMutationResponse is the mutation response for update operations.
type SemanticModelMutationResponse struct {
	Updated bool `json:"updated,omitempty"`
	Deleted bool `json:"deleted,omitempty"`
}

// SemanticModelValidateResponse is the validate response.
type SemanticModelValidateResponse struct {
	Valid bool `json:"valid"`
}

// Create creates a semantic model.
func (s *SemanticModelService) Create(ctx context.Context, req *SemanticModelUpsertRequest) (*SemanticModel, error) {
	if err := validateSemanticModelUpsertRequest(req); err != nil {
		return nil, err
	}
	var out SemanticModel
	if err := s.http.Post(ctx, s.base(), req, &out); err != nil {
		if e := asMOIError(err); e != nil {
			return nil, e
		}
		return nil, err
	}
	return &out, nil
}

// List lists semantic models.
func (s *SemanticModelService) List(ctx context.Context, opts ...ListOption) (*SemanticModelListResponse, error) {
	options := &listOptions{}
	for _, opt := range opts {
		opt(options)
	}
	path := withListQuery(s.base(), options, "")
	var out SemanticModelListResponse
	if err := s.http.Get(ctx, path, &out); err != nil {
		if e := asMOIError(err); e != nil {
			return nil, e
		}
		return nil, err
	}
	return &out, nil
}

// ListTags lists aggregated semantic model tags.
func (s *SemanticModelService) ListTags(ctx context.Context, opts ...ListOption) (*SemanticModelTagListResponse, error) {
	options := &listOptions{}
	for _, opt := range opts {
		opt(options)
	}
	path := withListQuery(s.base()+"/tags", options, "")
	var out SemanticModelTagListResponse
	if err := s.http.Get(ctx, path, &out); err != nil {
		if e := asMOIError(err); e != nil {
			return nil, e
		}
		return nil, err
	}
	return &out, nil
}

// Get retrieves a semantic model by ID.
func (s *SemanticModelService) Get(ctx context.Context, modelID int64) (*SemanticModel, error) {
	if modelID <= 0 {
		return nil, fmt.Errorf("model_id must be greater than 0")
	}
	path := fmt.Sprintf("%s/%d", s.base(), modelID)
	var out SemanticModel
	if err := s.http.Get(ctx, path, &out); err != nil {
		if e := asMOIError(err); e != nil {
			return nil, e
		}
		return nil, err
	}
	return &out, nil
}

// Update updates a semantic model.
func (s *SemanticModelService) Update(ctx context.Context, modelID int64, req *SemanticModelUpsertRequest) (*SemanticModelMutationResponse, error) {
	if modelID <= 0 {
		return nil, fmt.Errorf("model_id must be greater than 0")
	}
	if err := validateSemanticModelUpsertRequest(req); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s/%d", s.base(), modelID)
	var out SemanticModelMutationResponse
	if err := s.http.Put(ctx, path, req, &out); err != nil {
		if e := asMOIError(err); e != nil {
			return nil, e
		}
		return nil, err
	}
	return &out, nil
}

// Delete deletes a semantic model.
func (s *SemanticModelService) Delete(ctx context.Context, modelID int64) error {
	if modelID <= 0 {
		return fmt.Errorf("model_id must be greater than 0")
	}
	path := fmt.Sprintf("%s/%d", s.base(), modelID)
	if err := s.http.Delete(ctx, path); err != nil {
		if e := asMOIError(err); e != nil {
			return e
		}
		return err
	}
	return nil
}

// Import imports a semantic model and its entries.
func (s *SemanticModelService) Import(ctx context.Context, req *SemanticModelImportRequest) (*SemanticModel, error) {
	if err := validateSemanticModelImportRequest(req); err != nil {
		return nil, err
	}
	path := s.base() + "/import"
	var out SemanticModel
	if err := s.http.Post(ctx, path, req, &out); err != nil {
		if e := asMOIError(err); e != nil {
			return nil, e
		}
		return nil, err
	}
	return &out, nil
}

// ImportBatch imports multiple semantic models in a single /import request.
func (s *SemanticModelService) ImportBatch(ctx context.Context, reqs []SemanticModelImportRequest) (*SemanticModelImportBatchResponse, error) {
	if len(reqs) == 0 {
		return nil, fmt.Errorf("requests must not be empty")
	}
	for i, req := range reqs {
		reqCopy := req
		if err := validateSemanticModelImportRequest(&reqCopy); err != nil {
			return nil, fmt.Errorf("request[%d]: %w", i, err)
		}
	}
	path := s.base() + "/import"
	var out SemanticModelImportBatchResponse
	if err := s.http.Post(ctx, path, reqs, &out); err != nil {
		if e := asMOIError(err); e != nil {
			return nil, e
		}
		return nil, err
	}
	return &out, nil
}

// Export exports a semantic model and all of its entries.
func (s *SemanticModelService) Export(ctx context.Context, modelID int64) (*SemanticModelExportResponse, error) {
	if modelID <= 0 {
		return nil, fmt.Errorf("model_id must be greater than 0")
	}
	path := fmt.Sprintf("%s/%d/export", s.base(), modelID)
	var out SemanticModelExportResponse
	if err := s.http.Get(ctx, path, &out); err != nil {
		if e := asMOIError(err); e != nil {
			return nil, e
		}
		return nil, err
	}
	return &out, nil
}

// Validate validates a semantic model.
func (s *SemanticModelService) Validate(ctx context.Context, modelID int64) (*SemanticModelValidateResponse, error) {
	if modelID <= 0 {
		return nil, fmt.Errorf("model_id must be greater than 0")
	}
	path := fmt.Sprintf("%s/%d/validate", s.base(), modelID)
	var out SemanticModelValidateResponse
	if err := s.http.Post(ctx, path, nil, &out); err != nil {
		if e := asMOIError(err); e != nil {
			return nil, e
		}
		return nil, err
	}
	return &out, nil
}

// CreateEntry creates a semantic entry.
func (s *SemanticModelService) CreateEntry(ctx context.Context, modelID int64, req *SemanticEntryUpsertRequest) (*SemanticEntry, error) {
	if modelID <= 0 {
		return nil, fmt.Errorf("model_id must be greater than 0")
	}
	if err := validateSemanticEntryUpsertRequest(req); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s/%d/entries", s.base(), modelID)
	var out SemanticEntry
	if err := s.http.Post(ctx, path, req, &out); err != nil {
		if e := asMOIError(err); e != nil {
			return nil, e
		}
		return nil, err
	}
	return &out, nil
}

// ListEntries lists semantic entries in a semantic model.
func (s *SemanticModelService) ListEntries(ctx context.Context, modelID int64, kind string, opts ...ListOption) (*SemanticEntryListResponse, error) {
	if modelID <= 0 {
		return nil, fmt.Errorf("model_id must be greater than 0")
	}
	options := &listOptions{}
	for _, opt := range opts {
		opt(options)
	}
	path := fmt.Sprintf("%s/%d/entries", s.base(), modelID)
	path = withListQuery(path, options, kind)
	var out SemanticEntryListResponse
	if err := s.http.Get(ctx, path, &out); err != nil {
		if e := asMOIError(err); e != nil {
			return nil, e
		}
		return nil, err
	}
	return &out, nil
}

// UpdateEntry updates a semantic entry.
func (s *SemanticModelService) UpdateEntry(ctx context.Context, modelID, entryID int64, req *SemanticEntryUpsertRequest) (*SemanticModelMutationResponse, error) {
	if modelID <= 0 {
		return nil, fmt.Errorf("model_id must be greater than 0")
	}
	if entryID <= 0 {
		return nil, fmt.Errorf("entry_id must be greater than 0")
	}
	if err := validateSemanticEntryUpsertRequest(req); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s/%d/entries/%d", s.base(), modelID, entryID)
	var out SemanticModelMutationResponse
	if err := s.http.Put(ctx, path, req, &out); err != nil {
		if e := asMOIError(err); e != nil {
			return nil, e
		}
		return nil, err
	}
	return &out, nil
}

// DeleteEntry deletes a semantic entry.
func (s *SemanticModelService) DeleteEntry(ctx context.Context, modelID, entryID int64) error {
	if modelID <= 0 {
		return fmt.Errorf("model_id must be greater than 0")
	}
	if entryID <= 0 {
		return fmt.Errorf("entry_id must be greater than 0")
	}
	path := fmt.Sprintf("%s/%d/entries/%d", s.base(), modelID, entryID)
	if err := s.http.Delete(ctx, path); err != nil {
		if e := asMOIError(err); e != nil {
			return e
		}
		return err
	}
	return nil
}

func withListQuery(path string, options *listOptions, kind string) string {
	params := url.Values{}
	if options != nil && options.pageSize > 0 {
		params.Set("page_size", strconv.Itoa(int(options.pageSize)))
	}
	if options != nil && options.pageToken != "" {
		params.Set("page_token", options.pageToken)
	}
	if options != nil && options.search != "" {
		params.Set("search", options.search)
	}
	if options != nil {
		for _, tag := range options.tags {
			params.Add("tags", tag)
		}
	}
	if strings.TrimSpace(kind) != "" {
		params.Set("kind", strings.TrimSpace(kind))
	}
	if encoded := params.Encode(); encoded != "" {
		return path + "?" + encoded
	}
	return path
}

func validateSemanticModelUpsertRequest(req *SemanticModelUpsertRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if err := validateStructuredTables(req.Tables); err != nil {
		return err
	}
	return nil
}

func validateSemanticModelImportRequest(req *SemanticModelImportRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if err := validateStructuredTables(req.Tables); err != nil {
		return err
	}
	return nil
}

// structuredTableEntry mirrors the expected tables format: [{db_name, table_names, parents}].
type structuredTableEntry struct {
	DBName     string   `json:"db_name"`
	TableNames []string `json:"table_names"`
	Parents    []string `json:"parents"`
}

// validateStructuredTables ensures tables is a structured array, rejecting legacy flat string arrays.
func validateStructuredTables(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil // tables is optional
	}
	var entries []structuredTableEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return fmt.Errorf("tables must be a structured array [{db_name, table_names, parents}]: %w", err)
	}
	if len(entries) == 0 {
		return nil // empty tables array is allowed
	}
	for i, entry := range entries {
		if len(entry.TableNames) == 0 {
			return fmt.Errorf("tables[%d].table_names is required and must not be empty", i)
		}
		for j, name := range entry.TableNames {
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("tables[%d].table_names[%d] must be a non-empty string", i, j)
			}
		}
	}
	return nil
}

func validateSemanticEntryUpsertRequest(req *SemanticEntryUpsertRequest) error {
	if req == nil {
		return fmt.Errorf("request is required")
	}
	if strings.TrimSpace(req.Kind) == "" {
		return fmt.Errorf("kind is required")
	}
	if strings.TrimSpace(req.Key) == "" {
		return fmt.Errorf("key is required")
	}
	if len(req.Spec) == 0 {
		return fmt.Errorf("spec is required")
	}
	return nil
}
