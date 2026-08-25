package session

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
)

func serviceError(code ServiceErrorCode, key i18n.Key, data map[string]any) *ServiceError {
	return &ServiceError{Code: code, Err: i18n.NewError(key, data)}
}

func localizedError(key i18n.Key, data map[string]any) error {
	return i18n.NewError(key, data)
}

func wrapLocalizedError(key i18n.Key, cause error, data map[string]any) error {
	return i18n.WrapError(key, cause, data)
}

func invalidSemanticKindError(kind string) error {
	return localizedError(i18n.KeySessionInvalidKind, map[string]any{"Kind": kind})
}

func invalidSemanticSpecError(kind string, cause error) error {
	field := "spec"
	var typeErr *json.UnmarshalTypeError
	if errors.As(cause, &typeErr) && typeErr.Field != "" {
		field += "." + typeErr.Field
	}
	return wrapLocalizedError(i18n.KeySessionInvalidSpec, cause, map[string]any{
		"Kind":  kind,
		"Field": field,
	})
}

func semanticModelNotFoundError() *ServiceError {
	return serviceError(ErrCodeNotFound, i18n.KeySessionSemanticModelNotFound, nil)
}

func semanticModelKBNotFoundError() *ServiceError {
	return serviceError(ErrCodeNotFound, i18n.KeySessionSemanticModelKBNotFound, nil)
}

func semanticModelNameExistsError() *ServiceError {
	return serviceError(ErrCodeConflict, i18n.KeySessionSemanticModelNameExists, nil)
}

func knowledgeBaseDatabaseNameExistsError(path string) *ServiceError {
	return serviceError(ErrCodeConflict, i18n.KeySessionKnowledgeBaseDatabaseNameExists, map[string]any{"Path": path})
}

func knowledgeBaseNameImmutableError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelNameImmutable, nil)
}

func knowledgeBaseWorkflowDeleteConflictError(cause error) *ServiceError {
	return &ServiceError{
		Code: ErrCodeConflict,
		Err:  i18n.WrapError(i18n.KeySessionKnowledgeBaseWorkflowDeleteConflict, cause, nil),
	}
}

func knowledgeBaseSourceDeleteConflictError() *ServiceError {
	return serviceError(ErrCodeConflict, i18n.KeySessionKnowledgeBaseSourceDeleteConflict, nil)
}

func semanticEntryNotFoundError(entryID int) *ServiceError {
	return serviceError(ErrCodeNotFound, i18n.KeySessionEntryNotFound, map[string]any{"ID": entryID})
}

func semanticKindCannotBeChangedError(existingKind string, requestedKind string) *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionKindCannotBeChanged, map[string]any{
		"Detail": fmt.Sprintf(" (current: %s, requested: %s)", existingKind, requestedKind),
	})
}

func semanticModelEntriesImportBlockedError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionModelEntriesImportBlocked, nil)
}

func invalidPageTokenError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionInvalidPageToken, nil)
}

func sourceParsingIncompleteError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSourceParsingIncomplete, nil)
}

func semanticModelSourcesRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelSourcesRequired, nil)
}

func semanticModelSourceRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelSourceRequired, nil)
}

func semanticModelSourceDocumentRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelSourceDocumentRequired, nil)
}

func knowledgeBaseSourceNotFoundError() *ServiceError {
	return serviceError(ErrCodeNotFound, i18n.KeySessionSemanticModelSourceNotFound, nil)
}

func knowledgeBaseDataDomainNotFoundError() *ServiceError {
	return serviceError(ErrCodeNotFound, i18n.KeySessionKnowledgeBaseDataDomainNotFound, nil)
}

func knowledgeBaseEmbeddingModelNotAvailableError(model string) *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionKnowledgeBaseEmbeddingModelNotAvailable, map[string]any{"Model": model})
}

func knowledgeBaseEmbeddingCapabilityUnavailableError(cause error) *ServiceError {
	return &ServiceError{
		Code: ErrCodeInternal,
		Err:  i18n.WrapError(i18n.KeySessionKnowledgeBaseEmbeddingCapabilityUnavailable, cause, nil),
	}
}

func knowledgeBaseDataDomainCatalogRepairFailedError(cause error) *ServiceError {
	return &ServiceError{
		Code: ErrCodeConflict,
		Err:  i18n.WrapError(i18n.KeySessionKBCatalogRepairFailed, cause, nil),
	}
}

func semanticModelSourceFieldRequiredError(index int, field string) *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelSourceFieldRequired, map[string]any{"Index": index, "Field": field})
}

func semanticModelSourceFieldInvalidError(index int, field string) *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelSourceFieldInvalid, map[string]any{"Index": index, "Field": field})
}

// Same catalog file listed more than once in one create/append request
// (including different volumes). Source identity is file-scoped.
func semanticModelCatalogFileDuplicateInRequestError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelCatalogFileDuplicateInRequest, nil)
}

// Same file_id claimed under a different volume_id (request expand, request
// body, or active source reuse). Identity is file_id only; volume is a gate.
func semanticModelCatalogFileVolumeConflictError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelCatalogFileVolumeConflict, nil)
}

func semanticModelSourceContentBase64UnsupportedError(index int) *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelContentBase64Unsupported, map[string]any{"Index": index})
}

func semanticModelSourceTypeUnsupportedError(sourceType string) *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelSourceTypeUnsupported, map[string]any{"Type": sourceType})
}

func semanticModelSourceTableConfigRequiredError(index int) *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelSourceTableConfigRequired, map[string]any{"Index": index})
}

func semanticModelSourceTableConfigUnsupportedError(index int) *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelSourceTableConfigUnsupported, map[string]any{"Index": index})
}

func tableSourceDocumentUnsupportedError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionTableSourceDocumentUnsupported, nil)
}

func tableSourceGovernanceUnsupportedError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionTableSourceGovernanceUnsupported, nil)
}

func tableSourceSegmentsUnsupportedError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionTableSourceSegmentsUnsupported, nil)
}

func knowledgeBaseFileIDRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionKnowledgeBaseFileIDRequired, nil)
}

func segmentVersionNotFoundError() *ServiceError {
	return serviceError(ErrCodeNotFound, i18n.KeySessionSegmentVersionNotFound, nil)
}

func initialSegmentVersionExistsError() *ServiceError {
	return serviceError(ErrCodeConflict, i18n.KeySessionInitialSegmentVersionExists, nil)
}

func segmentNotFoundError() *ServiceError {
	return serviceError(ErrCodeNotFound, i18n.KeySessionSegmentNotFound, nil)
}

func segmentEnabledRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSegmentEnabledRequired, nil)
}

func currentSegmentVersionEmptyError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionCurrentSegmentVersionEmpty, nil)
}

func segmentVersionRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSegmentVersionRequired, nil)
}

func committedSegmentVersionRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionCommittedSegmentVersionRequired, nil)
}

func segmentVersionConflictError() *ServiceError {
	return serviceError(ErrCodeConflict, i18n.KeySessionSegmentVersionConflict, nil)
}

func segmentBaseRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSegmentBaseRequired, nil)
}

func segmentVersionEmptyError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSegmentVersionEmpty, nil)
}

func segmentArtifactIdentityReadOnlyError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSegmentArtifactIdentityReadOnly, nil)
}

func semanticModelFilesRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelFilesRequired, nil)
}

func semanticModelFilesInvalidError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelFilesInvalid, nil)
}

func semanticModelTablesInvalidError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSemanticModelTablesInvalid, nil)
}

func segmentVectorTableRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSegmentVectorTableRequired, nil)
}

func segmentEmbeddingModelRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSegmentEmbeddingModelRequired, nil)
}

func segmentIdentityRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSegmentIdentityRequired, nil)
}

func segmentRowsUnavailableError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSegmentRowsUnavailable, nil)
}

func initialSegmentRowsUnavailableError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionInitialSegmentRowsUnavailable, nil)
}

func workflowSegmentRowsUnavailableError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionWorkflowSegmentRowsUnavailable, nil)
}

func workflowIndexVersionNotNewerError() *ServiceError {
	return serviceError(ErrCodeConflict, i18n.KeySessionWorkflowIndexVersionNotNewer, nil)
}

func initialImportIndexVersionPositiveError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionInitialImportIndexVersionPositive, nil)
}

func legacyImageVectorMetadataInvalidError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionLegacyImageVectorMetadataInvalid, nil)
}

func legacyImageVectorKindInvalidError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionLegacyImageVectorKindInvalid, nil)
}

func legacyImageVectorIdentityMissingError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionLegacyImageVectorIdentityMissing, nil)
}

func vectorTableColumnMissingError(column string) *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionVectorTableColumnMissing, map[string]any{"Column": column})
}

func vectorTableEmbeddingColumnInvalidError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionVectorTableEmbeddingColumnInvalid, nil)
}

func segmentEmbeddingInputEmptyError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSegmentEmbeddingInputEmpty, nil)
}

func segmentEmbeddingDimensionMismatchError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionSegmentEmbeddingDimensionMismatch, nil)
}

func imageVectorTableRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionImageVectorTableRequired, nil)
}

func imageVectorTableColumnMissingError(column string) *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionImageVectorTableColumnMissing, map[string]any{"Column": column})
}

func imageVectorConfigMissingError(fields string) *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionImageVectorConfigMissing, map[string]any{"Fields": fields})
}

func imageSegmentPageNumberRequiredError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionImageSegmentPageNumberRequired, nil)
}

func imageEmbeddingDimensionMismatchError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionImageEmbeddingDimensionMismatch, nil)
}

func imageEmbeddingMetadataMismatchError(field string) *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionImageEmbeddingMetadataMismatch, map[string]any{"Field": field})
}

func imageBytesEmptyError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionImageBytesEmpty, nil)
}

func vectorTableUnavailableError() *ServiceError {
	return serviceError(ErrCodeBadRequest, i18n.KeySessionVectorTableUnavailable, nil)
}

func vectorTableUnavailableWithCauseError(cause error) *ServiceError {
	return &ServiceError{
		Code: ErrCodeBadRequest,
		Err:  i18n.WrapError(i18n.KeySessionVectorTableUnavailable, cause, nil),
	}
}

func segmentEmbeddingResponseInvalidError() *ServiceError {
	return serviceError(ErrCodeInternal, i18n.KeySessionSegmentEmbeddingResponseInvalid, nil)
}

func imageEmbeddingResponseInvalidError() *ServiceError {
	return serviceError(ErrCodeInternal, i18n.KeySessionImageEmbeddingResponseInvalid, nil)
}
