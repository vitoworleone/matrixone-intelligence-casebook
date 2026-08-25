package session

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/coreclient"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
)

var knowledgeBaseFileExtPattern = regexp.MustCompile(`^[a-z0-9]+$`)

func validateCreateSemanticModelSources(sources []CreateSemanticModelSourceRequest) error {
	// Source identity for catalog_file is file_id only. volume_id is a write-time
	// IAM/membership gate, not part of identity. One request must not claim the
	// same file twice (same or different volumes) or the insert path collides
	// on primary key / silently reuses the first row.
	seenCatalogFileVolumes := make(map[string]int64, len(sources))
	for i, source := range sources {
		if len(source.DeprecatedContentBase64) > 0 {
			return semanticModelSourceContentBase64UnsupportedError(i)
		}
		switch source.SourceType {
		case kbSourceTypeLocalFile:
			if source.FileName == "" {
				return semanticModelSourceFieldRequiredError(i, "file_name")
			}
			if source.FileID == "" {
				return semanticModelSourceFieldRequiredError(i, "file_id")
			}
			if source.UploadKind != "" && source.UploadKind != kbLocalUploadKindUnstructured && source.UploadKind != kbLocalUploadKindStructured {
				return semanticModelSourceFieldInvalidError(i, "upload_kind")
			}
			if source.UploadKind == kbLocalUploadKindStructured && source.TableConfig == "" {
				return semanticModelSourceTableConfigRequiredError(i)
			}
			if source.UploadKind != kbLocalUploadKindStructured && source.TableConfig != "" {
				return semanticModelSourceTableConfigUnsupportedError(i)
			}
		case kbSourceTypeCatalogFile:
			if source.FileID == "" {
				return semanticModelSourceFieldRequiredError(i, "file_id")
			}
			if source.VolumeID <= 0 {
				return semanticModelSourceFieldRequiredError(i, "volume_id")
			}
			if existingVolume, exists := seenCatalogFileVolumes[source.FileID]; exists {
				if existingVolume != source.VolumeID {
					return semanticModelCatalogFileVolumeConflictError()
				}
				return semanticModelCatalogFileDuplicateInRequestError()
			}
			seenCatalogFileVolumes[source.FileID] = source.VolumeID
		case kbSourceTypeCatalogTable:
			if source.TableID <= 0 {
				return semanticModelSourceFieldRequiredError(i, "table_id")
			}
		default:
			return semanticModelSourceFieldInvalidError(i, "source_type")
		}
	}
	return nil
}

// UploadLocalFile stores one knowledge-base local file after the HTTP layer has
// already authorized the owning semantic_model.create or semantic_model.update.
func (s *semanticModelService) UploadLocalFile(ctx context.Context, fileName string, reader io.Reader) (string, error) {
	if strings.TrimSpace(fileName) == "" {
		return "", fmt.Errorf("file name is required")
	}
	if reader == nil {
		return "", fmt.Errorf("file content is required")
	}
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return "", err
	}
	fileID := ""
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		resp, callErr := client.Files().Upload(callCtx, wsID, fileName, reader)
		if callErr != nil {
			return fmt.Errorf("upload local file %q: %w", fileName, callErr)
		}
		if resp == nil || strings.TrimSpace(resp.FileID) == "" {
			return fmt.Errorf("upload local file %q: empty file_id", fileName)
		}
		fileID = resp.FileID
		return nil
	})
	return fileID, err
}

func validateSemanticModelSourceSelections(selections []SemanticModelSourceSelectionRequest) error {
	for i, selection := range selections {
		switch selection.Kind {
		case kbSelectionKindDatabaseTables:
			if selection.DatabaseID <= 0 {
				return semanticModelSourceFieldRequiredError(i, "database_id")
			}
			if err := validateUniquePositiveInt64s(selection.SelectedTableIDs); err != nil {
				return semanticModelSourceFieldInvalidError(i, "selected_table_ids")
			}
			if err := validateUniquePositiveInt64s(selection.ExcludedTableIDs); err != nil {
				return semanticModelSourceFieldInvalidError(i, "excluded_table_ids")
			}
			if !selection.AllSelected && len(selection.SelectedTableIDs) == 0 {
				return semanticModelSourceFieldRequiredError(i, "selected_table_ids")
			}
		case kbSelectionKindVolumeFiles:
			if selection.VolumeID <= 0 {
				return semanticModelSourceFieldRequiredError(i, "volume_id")
			}
			if err := validateUniqueExactStrings(selection.SelectedFileIDs); err != nil {
				return semanticModelSourceFieldInvalidError(i, "selected_file_ids")
			}
			if err := validateUniqueExactStrings(selection.ExcludedFileIDs); err != nil {
				return semanticModelSourceFieldInvalidError(i, "excluded_file_ids")
			}
			if !selection.AllSelected && len(selection.SelectedFileIDs) == 0 {
				return semanticModelSourceFieldRequiredError(i, "selected_file_ids")
			}
			if err := validateUniqueExactStrings(selection.Filters.FileExt); err != nil {
				return semanticModelSourceFieldInvalidError(i, "filters.file_ext")
			}
			for _, ext := range selection.Filters.FileExt {
				if !knowledgeBaseFileExtPattern.MatchString(ext) {
					return semanticModelSourceFieldInvalidError(i, "filters.file_ext")
				}
			}
		default:
			return semanticModelSourceFieldInvalidError(i, "kind")
		}
	}
	return nil
}

func validateUniqueExactStrings(values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" || value != strings.TrimSpace(value) {
			return fmt.Errorf("value must be non-empty without surrounding whitespace")
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("duplicate value")
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateUniquePositiveInt64s(values []int64) error {
	seen := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			return fmt.Errorf("value must be positive")
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("duplicate value")
		}
		seen[value] = struct{}{}
	}
	return nil
}

func appendSourceRequests(base []CreateSemanticModelSourceRequest, more []CreateSemanticModelSourceRequest) []CreateSemanticModelSourceRequest {
	if len(more) == 0 {
		return base
	}
	out := make([]CreateSemanticModelSourceRequest, 0, len(base)+len(more))
	out = append(out, base...)
	out = append(out, more...)
	return out
}

func (s *semanticModelService) PreviewSourceSelectionCounts(ctx context.Context, params PreviewSemanticModelSourceSelectionsRequest) (*PreviewSemanticModelSourceSelectionsResponse, error) {
	if len(params.SourceSelections) == 0 {
		return nil, semanticModelSourcesRequiredError()
	}
	if err := validateSemanticModelSourceSelections(params.SourceSelections); err != nil {
		return nil, err
	}
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}
	var result *PreviewSemanticModelSourceSelectionsResponse
	err = coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		sources, callErr := s.expandSemanticModelSourceSelections(callCtx, client, wsID, int64(params.ModelID), params.SourceSelections, nil)
		if callErr != nil {
			return callErr
		}
		result = &PreviewSemanticModelSourceSelectionsResponse{}
		for _, source := range sources {
			switch source.SourceType {
			case kbSourceTypeCatalogFile:
				result.FileCount++
			case kbSourceTypeCatalogTable:
				result.TableCount++
			}
		}
		result.TotalCount = result.FileCount + result.TableCount
		return nil
	})
	return result, err
}

func (s *semanticModelService) expandSemanticModelSourceSelections(ctx context.Context, c *moi.Client, wsID string, modelID int64, selections []SemanticModelSourceSelectionRequest, explicit []CreateSemanticModelSourceRequest) ([]CreateSemanticModelSourceRequest, error) {
	if len(selections) == 0 {
		return nil, nil
	}
	// file_id -> volume_id already claimed in this request (explicit or earlier
	// selection). Same volume is idempotent skip; different volume fails closed.
	seenFiles := map[string]int64{}
	seenTables := map[int64]struct{}{}
	for _, source := range explicit {
		switch source.SourceType {
		case kbSourceTypeCatalogFile:
			if source.FileID != "" {
				seenFiles[source.FileID] = source.VolumeID
			}
		case kbSourceTypeCatalogTable:
			if source.TableID > 0 {
				seenTables[source.TableID] = struct{}{}
			}
		}
	}
	out := make([]CreateSemanticModelSourceRequest, 0)
	for i, selection := range selections {
		var (
			items []CreateSemanticModelSourceRequest
			err   error
		)
		switch selection.Kind {
		case kbSelectionKindDatabaseTables:
			items, err = s.expandDatabaseTableSelection(ctx, c, wsID, selection, seenTables)
		case kbSelectionKindVolumeFiles:
			items, err = s.expandVolumeFileSelection(ctx, selection, seenFiles)
		default:
			err = semanticModelSourceFieldInvalidError(i, "kind")
		}
		if err != nil {
			return nil, err
		}
		if modelID > 0 {
			items, err = s.excludeExistingKnowledgeBaseSelectionSources(ctx, modelID, items)
			if err != nil {
				return nil, err
			}
		}
		for _, item := range items {
			switch item.SourceType {
			case kbSourceTypeCatalogFile:
				seenFiles[item.FileID] = item.VolumeID
			case kbSourceTypeCatalogTable:
				seenTables[item.TableID] = struct{}{}
			}
			out = append(out, item)
		}
	}
	return out, nil
}

func (s *semanticModelService) excludeExistingKnowledgeBaseSelectionSources(ctx context.Context, modelID int64, sources []CreateSemanticModelSourceRequest) ([]CreateSemanticModelSourceRequest, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, fmt.Errorf("tenant db is required")
	}
	fileIDs := make([]string, 0, len(sources))
	tableIDs := make([]int64, 0, len(sources))
	for _, source := range sources {
		switch source.SourceType {
		case kbSourceTypeCatalogFile:
			fileIDs = append(fileIDs, source.FileID)
		case kbSourceTypeCatalogTable:
			tableIDs = append(tableIDs, source.TableID)
		}
	}
	// file_id -> recorded volume. Same volume: idempotent skip. Different volume: fail closed.
	existingFileVolumes := map[string]int64{}
	if len(fileIDs) > 0 {
		var err error
		existingFileVolumes, err = s.existingKnowledgeBaseCatalogFileVolumes(ctx, db, modelID, fileIDs)
		if err != nil {
			return nil, err
		}
	}
	existingTables := map[int64]struct{}{}
	if len(tableIDs) > 0 {
		var err error
		existingTables, err = s.existingKnowledgeBaseCatalogTableIDs(ctx, db, modelID, tableIDs)
		if err != nil {
			return nil, err
		}
	}
	out := make([]CreateSemanticModelSourceRequest, 0, len(sources))
	for _, source := range sources {
		if source.SourceType == kbSourceTypeCatalogFile {
			if existingVolume, exists := existingFileVolumes[source.FileID]; exists {
				if existingVolume > 0 && source.VolumeID > 0 && existingVolume != source.VolumeID {
					return nil, semanticModelCatalogFileVolumeConflictError()
				}
				continue
			}
		}
		if _, exists := existingTables[source.TableID]; source.SourceType == kbSourceTypeCatalogTable && exists {
			continue
		}
		out = append(out, source)
	}
	return out, nil
}

func (s *semanticModelService) expandDatabaseTableSelection(ctx context.Context, c *moi.Client, wsID string, selection SemanticModelSourceSelectionRequest, seen map[int64]struct{}) ([]CreateSemanticModelSourceRequest, error) {
	if selection.AllSelected {
		excluded := int64Set(selection.ExcludedTableIDs)
		pageToken := ""
		out := make([]CreateSemanticModelSourceRequest, 0)
		matched := false
		for {
			resp, err := s.dataDomainService.ListDatabaseTableLeaves(ctx, KnowledgeBaseTableLeafListParams{
				DatabaseID: selection.DatabaseID,
				PageSize:   kbSourceSelectionBatchSize,
				PageToken:  pageToken,
				Search:     selection.Filters.TableName,
			})
			if err != nil {
				return nil, err
			}
			if resp == nil || len(resp.Items) == 0 {
				break
			}
			for _, item := range resp.Items {
				matched = true
				if _, skip := excluded[item.TableID]; skip {
					continue
				}
				if _, exists := seen[item.TableID]; exists {
					continue
				}
				out = append(out, CreateSemanticModelSourceRequest{SourceType: kbSourceTypeCatalogTable, TableID: item.TableID})
			}
			if resp.NextPageToken == "" {
				break
			}
			pageToken = resp.NextPageToken
		}
		if !matched {
			return nil, fmt.Errorf("database table selection matched no tables")
		}
		return out, nil
	}

	tableIDs := nonZeroInt64s(selection.SelectedTableIDs)
	out := make([]CreateSemanticModelSourceRequest, 0, len(tableIDs))
	for _, tableID := range tableIDs {
		detail, err := c.Databases().GetTable(ctx, wsID, tableID)
		if err != nil {
			return nil, fmt.Errorf("get selected catalog table %d: %w", tableID, err)
		}
		if detail == nil || detail.Table == nil || detail.Table.DatabaseId != selection.DatabaseID {
			return nil, fmt.Errorf("selected catalog table %d does not belong to database %d", tableID, selection.DatabaseID)
		}
		if _, exists := seen[tableID]; exists {
			continue
		}
		out = append(out, CreateSemanticModelSourceRequest{SourceType: kbSourceTypeCatalogTable, TableID: tableID})
	}
	return out, nil
}

// expandVolumeFileSelection expands one volume_files selection into catalog_file
// source requests. seen maps file_id -> volume_id already claimed in this request.
// Same file + same volume is skipped (idempotent). Same file + different volume
// fails closed — identity is file_id only; volume_id is only a write-time gate.
func (s *semanticModelService) expandVolumeFileSelection(ctx context.Context, selection SemanticModelSourceSelectionRequest, seen map[string]int64) ([]CreateSemanticModelSourceRequest, error) {
	if s.fileService == nil {
		return nil, fmt.Errorf("catalog file service is not configured")
	}
	if seen == nil {
		seen = map[string]int64{}
	}
	if selection.AllSelected {
		excluded := stringSet(selection.ExcludedFileIDs)
		page := 1
		out := make([]CreateSemanticModelSourceRequest, 0)
		matched := false
		for {
			resp, err := s.fileService.ListFiles(ctx, KnowledgeBaseCatalogFileListParams{
				VolumeID: selection.VolumeID,
				Page:     page,
				PageSize: kbSourceSelectionBatchSize,
				FileName: selection.Filters.FileName,
				FileExt:  normalizeFileExtFilters(selection.Filters.FileExt),
			})
			if err != nil {
				return nil, err
			}
			if resp == nil || len(resp.Items) == 0 {
				break
			}
			for _, item := range resp.Items {
				matched = true
				if item.VolumeID != selection.VolumeID {
					return nil, fmt.Errorf("selected catalog file %s does not belong to volume %d", item.FileID, selection.VolumeID)
				}
				if err := validateKnowledgeBaseCatalogFileExtension(item.FileName); err != nil {
					return nil, err
				}
				if _, skip := excluded[item.FileID]; skip {
					continue
				}
				claimed, err := claimExpandedCatalogFile(seen, item.FileID, selection.VolumeID)
				if err != nil {
					return nil, err
				}
				if !claimed {
					continue
				}
				out = append(out, CreateSemanticModelSourceRequest{SourceType: kbSourceTypeCatalogFile, FileID: item.FileID, FileName: item.FileName, VolumeID: selection.VolumeID})
			}
			if page*kbSourceSelectionBatchSize >= resp.Total {
				break
			}
			page++
		}
		if !matched {
			return nil, fmt.Errorf("volume file selection matched no files")
		}
		return out, nil
	}

	fileIDs := compactUniqueStrings(selection.SelectedFileIDs)
	byID := make(map[string]KnowledgeBaseCatalogFileLeaf, len(fileIDs))
	for start := 0; start < len(fileIDs); start += kbSourceSelectionBatchSize {
		end := start + kbSourceSelectionBatchSize
		if end > len(fileIDs) {
			end = len(fileIDs)
		}
		resp, err := s.fileService.ListFiles(ctx, KnowledgeBaseCatalogFileListParams{
			VolumeID: selection.VolumeID,
			Page:     1,
			PageSize: end - start,
			FileIDs:  fileIDs[start:end],
		})
		if err != nil {
			return nil, err
		}
		if resp != nil {
			for _, item := range resp.Items {
				if item.FileID != "" {
					byID[item.FileID] = item
				}
			}
		}
	}
	out := make([]CreateSemanticModelSourceRequest, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		item, ok := byID[fileID]
		if !ok || item.VolumeID != selection.VolumeID {
			return nil, fmt.Errorf("selected catalog file %s does not belong to volume %d", fileID, selection.VolumeID)
		}
		if err := validateKnowledgeBaseCatalogFileExtension(item.FileName); err != nil {
			return nil, err
		}
		claimed, err := claimExpandedCatalogFile(seen, fileID, selection.VolumeID)
		if err != nil {
			return nil, err
		}
		if !claimed {
			continue
		}
		out = append(out, CreateSemanticModelSourceRequest{SourceType: kbSourceTypeCatalogFile, FileID: fileID, FileName: item.FileName, VolumeID: selection.VolumeID})
	}
	return out, nil
}

// claimExpandedCatalogFile records file_id at volumeID for this request.
// Returns true when the file is newly claimed and should be appended.
// Same volume re-claim returns false (idempotent skip). Different volume fails closed.
func claimExpandedCatalogFile(seen map[string]int64, fileID string, volumeID int64) (bool, error) {
	if fileID == "" || volumeID <= 0 {
		return false, nil
	}
	if existingVolume, exists := seen[fileID]; exists {
		if existingVolume > 0 && existingVolume != volumeID {
			return false, semanticModelCatalogFileVolumeConflictError()
		}
		return false, nil
	}
	seen[fileID] = volumeID
	return true, nil
}

func nonZeroInt64s(values []int64) []int64 {
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
	return out
}

func int64Set(values []int64) map[int64]struct{} {
	out := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value > 0 {
			out[value] = struct{}{}
		}
	}
	return out
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func compactUniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeFileExtFilters(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		value = strings.TrimPrefix(value, ".")
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
