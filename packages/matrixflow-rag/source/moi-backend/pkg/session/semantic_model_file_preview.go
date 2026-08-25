package session

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
)

var semanticModelArtifactImageRefPattern = regexp.MustCompile(`(?i)^([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})(\.(png|jpe?g|gif|webp|bmp|tiff?))?$`)

const (
	semanticModelArtifactInvalidMessage       = "semantic model artifact identity is invalid"
	semanticModelArtifactNotFoundMessage      = "semantic model artifact not found"
	semanticModelArtifactUnavailableMessage   = "semantic model artifact preview is unavailable"
	semanticModelSourceFileInvalidMessage     = "semantic model source file identity is invalid"
	semanticModelSourceFileNotFoundMessage    = "semantic model source file not found"
	semanticModelSourceFileUnavailableMessage = "semantic model source file preview is unavailable"
)

type semanticModelWorkflowFileKind string

const (
	semanticModelWorkflowSourceFile               semanticModelWorkflowFileKind = "source"
	semanticModelWorkflowArtifactFile             semanticModelWorkflowFileKind = "artifact"
	semanticModelWorkflowArtifactMetadataMatchSQL                               = `
		JSON_UNQUOTE(JSON_EXTRACT(meta, '$.image_file_id')) = ?
		OR JSON_UNQUOTE(JSON_EXTRACT(meta, '$.page_image_file_id')) = ?
		OR JSON_UNQUOTE(JSON_EXTRACT(meta, '$.image_url')) = ?
		OR JSON_UNQUOTE(JSON_EXTRACT(meta, '$.s3_image_url')) = ?
		OR JSON_UNQUOTE(JSON_EXTRACT(meta, '$.table_image_url')) = ?
		OR JSON_UNQUOTE(JSON_EXTRACT(meta, '$.md_file_id')) = ?
		OR JSON_UNQUOTE(JSON_EXTRACT(meta, '$.layout_file_id')) = ?
		OR (
			? <> ''
			AND (
				LOWER(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.image_url'))) REGEXP ?
				OR LOWER(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.s3_image_url'))) REGEXP ?
				OR LOWER(JSON_UNQUOTE(JSON_EXTRACT(meta, '$.table_image_url'))) REGEXP ?
			)
		)`
)

type semanticModelWorkflowVectorTableRef struct {
	TableName string `gorm:"column:table_name"`
}

func (s *semanticModelService) PreviewArtifact(ctx context.Context, modelID int, fileID string) (*SemanticModelFilePreview, error) {
	if modelID <= 0 || fileID == "" || strings.TrimSpace(fileID) != fileID {
		return nil, &ServiceError{Code: ErrCodeBadRequest, Msg: semanticModelArtifactInvalidMessage}
	}

	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, &ServiceError{
			Code: ErrCodeInternal,
			Msg:  semanticModelArtifactUnavailableMessage,
			Err:  errors.New("tenant database is unavailable"),
		}
	}

	if artifactImageID := semanticModelArtifactImageID(fileID); artifactImageID != "" {
		fileID = artifactImageID
	}
	associated, err := s.hasSemanticModelWorkflowFileAssociation(ctx, modelID, fileID, semanticModelWorkflowArtifactFile)
	if err != nil {
		return nil, &ServiceError{
			Code: ErrCodeInternal,
			Msg:  semanticModelArtifactUnavailableMessage,
			Err:  fmt.Errorf("resolve semantic model artifact workflow association: %w", err),
		}
	}
	if !associated {
		return nil, &ServiceError{Code: ErrCodeNotFound, Msg: semanticModelArtifactNotFoundMessage}
	}
	return s.previewSemanticModelFile(ctx, fileID, semanticModelArtifactUnavailableMessage, "artifact")
}

func semanticModelArtifactImageID(ref string) string {
	match := semanticModelArtifactImageRefPattern.FindStringSubmatch(ref)
	if len(match) < 2 {
		return ""
	}
	return strings.ToLower(match[1])
}

func (s *semanticModelService) PreviewSourceFile(ctx context.Context, modelID int, fileID string) (*SemanticModelFilePreview, error) {
	if modelID <= 0 || fileID == "" || strings.TrimSpace(fileID) != fileID {
		return nil, &ServiceError{Code: ErrCodeBadRequest, Msg: semanticModelSourceFileInvalidMessage}
	}
	if ctxutil.TenantDBFrom(ctx) == nil {
		return nil, &ServiceError{
			Code: ErrCodeInternal,
			Msg:  semanticModelSourceFileUnavailableMessage,
			Err:  errors.New("tenant database is unavailable"),
		}
	}

	associated, err := s.hasSemanticModelWorkflowFileAssociation(ctx, modelID, fileID, semanticModelWorkflowSourceFile)
	if err != nil {
		return nil, &ServiceError{
			Code: ErrCodeInternal,
			Msg:  semanticModelSourceFileUnavailableMessage,
			Err:  fmt.Errorf("resolve semantic model source file workflow association: %w", err),
		}
	}
	if !associated {
		return nil, &ServiceError{Code: ErrCodeNotFound, Msg: semanticModelSourceFileNotFoundMessage}
	}
	return s.previewSemanticModelFile(ctx, fileID, semanticModelSourceFileUnavailableMessage, "source file")
}

// hasSemanticModelWorkflowFileAssociation verifies the lineage produced by a
// successful knowledge-base workflow. Source previews accept only the root
// document; artifact previews accept only files derived from that document.
func (s *semanticModelService) hasSemanticModelWorkflowFileAssociation(
	ctx context.Context,
	modelID int,
	fileID string,
	kind semanticModelWorkflowFileKind,
) (bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, errors.New("tenant database is unavailable")
	}

	match := `(root.asset_ref = ? OR pm.source_file_id = ?)`
	args := []any{int64(modelID), fileID, fileID}
	if kind == semanticModelWorkflowArtifactFile {
		match = `(pm.parsed_file_id = ? OR EXISTS (
			SELECT 1
			FROM data_derivation artifact_derivation
			INNER JOIN data_asset artifact
				ON artifact.asset_id = artifact_derivation.target_asset_id
				AND artifact.asset_type = 'file'
			WHERE artifact_derivation.root_asset_id = root.asset_id
				AND artifact_derivation.kind IN ('derived_file_from', 'transformed_from')
				AND artifact.asset_ref = ?
		))`
		args = []any{int64(modelID), fileID, fileID}
	}

	var associations int64
	query := `SELECT COUNT(*)
		FROM semantic_models sm
		INNER JOIN data_asset vector_asset
			ON vector_asset.asset_type = 'vector_index'
			AND (
				vector_asset.asset_ref = JSON_UNQUOTE(JSON_EXTRACT(sm.files, '$.vector_table'))
				OR vector_asset.asset_ref = JSON_UNQUOTE(JSON_EXTRACT(sm.files, '$.image_vector_table'))
			)
		INNER JOIN data_derivation indexed_derivation
			ON indexed_derivation.target_asset_id = vector_asset.asset_id
			AND indexed_derivation.kind = 'indexed_from'
		INNER JOIN data_asset root
			ON root.asset_id = indexed_derivation.root_asset_id
			AND root.asset_type = 'file'
		LEFT JOIN parsed_manifest pm
			ON pm.root_asset_id = root.asset_id
		WHERE sm.id = ?
			AND ` + match
	if err := db.WithContext(ctx).Raw(query, args...).Scan(&associations).Error; err != nil {
		return false, err
	}
	if associations > 0 {
		return true, nil
	}
	return s.hasSemanticModelWorkflowVectorFileAssociation(ctx, modelID, fileID, kind)
}

// hasSemanticModelWorkflowVectorFileAssociation covers workflow results that
// exist in the model's vector rows when Catalog lineage metadata is absent.
func (s *semanticModelService) hasSemanticModelWorkflowVectorFileAssociation(ctx context.Context, modelID int, fileID string, kind semanticModelWorkflowFileKind) (bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, errors.New("tenant database is unavailable")
	}

	where := "file_id = ?"
	args := []any{fileID}
	if kind == semanticModelWorkflowArtifactFile {
		metadataImageID := semanticModelArtifactImageID(fileID)
		metadataImagePattern := "^$"
		if metadataImageID != "" {
			metadataImagePattern = "^" + metadataImageID + `(\.(png|jpe?g|gif|webp|bmp|tiff?))?$`
		}
		where = `file_id IS NOT NULL
			AND file_id <> ''
			AND (` + semanticModelWorkflowArtifactMetadataMatchSQL + `)`
		args = []any{
			fileID,
			fileID,
			fileID,
			fileID,
			fileID,
			fileID,
			fileID,
			metadataImageID,
			metadataImagePattern,
			metadataImagePattern,
			metadataImagePattern,
		}
	} else if kind != semanticModelWorkflowSourceFile {
		return false, fmt.Errorf("unsupported semantic model workflow file kind %q", kind)
	}

	var tables []semanticModelWorkflowVectorTableRef
	if err := db.WithContext(ctx).Raw(`SELECT JSON_UNQUOTE(JSON_EXTRACT(files, '$.vector_table')) AS table_name
		FROM semantic_models
		WHERE id = ?
		UNION
		SELECT JSON_UNQUOTE(JSON_EXTRACT(files, '$.image_vector_table')) AS table_name
		FROM semantic_models
		WHERE id = ?`, int64(modelID), int64(modelID)).Scan(&tables).Error; err != nil {
		return false, err
	}

	seen := make(map[string]struct{}, len(tables))
	for _, table := range tables {
		tableName := strings.TrimSpace(table.TableName)
		if tableName == "" {
			continue
		}
		if _, ok := seen[tableName]; ok {
			continue
		}
		seen[tableName] = struct{}{}

		quotedTable, err := quotedOptionalVectorTable(tableName)
		if err != nil {
			return false, err
		}
		var associations int64
		query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s`, quotedTable, where)
		if err := db.WithContext(ctx).Raw(query, args...).Scan(&associations).Error; err != nil {
			return false, err
		}
		if associations > 0 {
			return true, nil
		}
	}
	return false, nil
}

func (s *semanticModelService) previewSemanticModelFile(
	ctx context.Context,
	fileID string,
	unavailableMessage string,
	fileKind string,
) (*SemanticModelFilePreview, error) {
	if s.fileService == nil {
		return nil, &ServiceError{
			Code: ErrCodeInternal,
			Msg:  unavailableMessage,
			Err:  errors.New("catalog file service is unavailable"),
		}
	}
	result, err := s.fileService.PreviewFile(ctx, fileID)
	if err != nil {
		return nil, &ServiceError{
			Code: ErrCodeInternal,
			Msg:  unavailableMessage,
			Err:  fmt.Errorf("preview semantic model %s: %w", fileKind, err),
		}
	}
	if result == nil || result.Body == nil {
		return nil, &ServiceError{
			Code: ErrCodeInternal,
			Msg:  unavailableMessage,
			Err:  errors.New("catalog file preview returned empty response"),
		}
	}
	return result, nil
}
