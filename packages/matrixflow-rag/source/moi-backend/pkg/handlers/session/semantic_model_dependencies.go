package sessionh

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/iampep"
	session "github.com/matrixorigin/matrixflow/moi-backend/pkg/session"
)

const (
	semanticSourceLocalFile    = "local_file"
	semanticSourceCatalogFile  = "catalog_file"
	semanticSourceCatalogTable = "catalog_table"
	semanticSelectionDatabase  = "database_tables"
	semanticSelectionVolume    = "volume_files"
)

type SemanticModelDependencyResolver interface {
	ResolveSourceDependencies(context.Context, string, []session.CreateSemanticModelSourceRequest) ([]iampep.ResourceAuthorization, error)
	ResolveSelectionDependencies(context.Context, string, []session.SemanticModelSourceSelectionRequest) ([]iampep.ResourceAuthorization, error)
	ResolveLegacyDependencies(context.Context, string, []byte, []byte) ([]iampep.ResourceAuthorization, error)
	ResolveBackfillDependencies(context.Context, string, int64) ([]iampep.ResourceAuthorization, error)
}

func (r *CoreSemanticModelDependencyResolver) ResolveSelectionDependencies(ctx context.Context, workspaceID string, selections []session.SemanticModelSourceSelectionRequest) ([]iampep.ResourceAuthorization, error) {
	if r == nil || strings.TrimSpace(workspaceID) == "" || len(selections) == 0 {
		return nil, fmt.Errorf("semantic model selection dependency resolution is unavailable or invalid")
	}
	out := make([]iampep.ResourceAuthorization, 0, len(selections))
	seen := make(map[string]struct{}, len(selections))
	for _, selection := range selections {
		var target iampep.ResourceAuthorization
		switch selection.Kind {
		case semanticSelectionDatabase:
			if selection.DatabaseID <= 0 || selection.VolumeID != 0 {
				return nil, semanticModelDependencyRequestError{err: fmt.Errorf("semantic model database selection identity is invalid")}
			}
			target = iampep.ResourceAuthorization{
				ActionID: "database.read",
				Resource: iampep.ResourceDescriptor{ResourceType: iampep.ResourceTypeDatabase, ResourceID: strconv.FormatInt(selection.DatabaseID, 10)},
			}
		case semanticSelectionVolume:
			if selection.VolumeID <= 0 || selection.DatabaseID != 0 {
				return nil, semanticModelDependencyRequestError{err: fmt.Errorf("semantic model volume selection identity is invalid")}
			}
			if r.Volumes == nil {
				return nil, fmt.Errorf("semantic model volume selection resolver is unavailable")
			}
			rootID, err := r.Volumes.ResolveCanonicalRootVolume(ctx, workspaceID, selection.VolumeID)
			if err != nil {
				return nil, fmt.Errorf("resolve semantic model selection root volume: %w", err)
			}
			if rootID <= 0 {
				return nil, fmt.Errorf("semantic model selection has no canonical root volume")
			}
			target = iampep.ResourceAuthorization{
				ActionID: "volume.read",
				Resource: iampep.ResourceDescriptor{ResourceType: iampep.ResourceTypeVolume, ResourceID: strconv.FormatInt(rootID, 10)},
			}
		default:
			return nil, semanticModelDependencyRequestError{err: fmt.Errorf("semantic model source selection kind is invalid")}
		}
		key := target.ActionID + "\x00" + target.Resource.ResourceType + "\x00" + target.Resource.ResourceID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, target)
	}
	return out, nil
}

type semanticModelDependencyRequestError struct {
	err error
	key i18n.Key
}

func (e semanticModelDependencyRequestError) Error() string { return e.err.Error() }
func (semanticModelDependencyRequestError) IAMHTTPStatus() (int, string) {
	return 400, "ErrParamInvalid"
}
func (e semanticModelDependencyRequestError) IAMI18NKey() i18n.Key {
	if e.key.String() != "" {
		return e.key
	}
	return i18n.KeyParamInvalid
}

// CoreSemanticModelDependencyResolver maps only stable source identities to
// canonical IAM resources. Local uploads have no existing resource; Catalog
// files authorize the caller-supplied volume_id through Core root-volume
// resolution (file@volume membership is re-checked by the service); Table IDs
// remain the coarse product-level Table resource checked again by MatrixOne at
// use time. Multi-volume files must not go through file-root containment:
// ResolveCanonicalFileRoots fails closed when a file has more than one root.
type CoreSemanticModelDependencyResolver struct {
	Sources interface {
		ResolveLegacySourceIAMDependencies(context.Context, json.RawMessage, json.RawMessage) ([]session.CreateSemanticModelSourceRequest, error)
		ResolveBackfillSourceIAMDependencies(context.Context, int64) ([]session.CreateSemanticModelSourceRequest, error)
	}
	Volumes interface {
		ResolveCanonicalRootVolume(context.Context, string, int64) (int64, error)
	}
}

func (r *CoreSemanticModelDependencyResolver) ResolveBackfillDependencies(ctx context.Context, workspaceID string, modelID int64) ([]iampep.ResourceAuthorization, error) {
	if r == nil || r.Sources == nil || modelID <= 0 {
		return nil, fmt.Errorf("semantic model backfill dependency resolver is unavailable")
	}
	sources, err := r.Sources.ResolveBackfillSourceIAMDependencies(ctx, modelID)
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, nil
	}
	return r.ResolveSourceDependencies(ctx, workspaceID, sources)
}

func (r *CoreSemanticModelDependencyResolver) ResolveLegacyDependencies(ctx context.Context, workspaceID string, tables, files []byte) ([]iampep.ResourceAuthorization, error) {
	if r == nil || r.Sources == nil {
		return nil, fmt.Errorf("semantic model legacy dependency resolver is unavailable")
	}
	sources, err := r.Sources.ResolveLegacySourceIAMDependencies(ctx, json.RawMessage(tables), json.RawMessage(files))
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, nil
	}
	return r.ResolveSourceDependencies(ctx, workspaceID, sources)
}

func (r *CoreSemanticModelDependencyResolver) ResolveSourceDependencies(ctx context.Context, workspaceID string, sources []session.CreateSemanticModelSourceRequest) ([]iampep.ResourceAuthorization, error) {
	if r == nil || strings.TrimSpace(workspaceID) == "" || len(sources) == 0 {
		return nil, fmt.Errorf("semantic model dependency resolution is unavailable or invalid")
	}
	volumeIDs := make([]int64, 0, len(sources))
	tableIDs := make([]int64, 0, len(sources))
	for _, source := range sources {
		switch source.SourceType {
		case semanticSourceLocalFile:
			if strings.TrimSpace(source.FileName) == "" || source.FileID == "" || strings.TrimSpace(source.FileID) != source.FileID {
				return nil, semanticModelDependencyRequestError{
					err: fmt.Errorf("local source is incomplete"),
					key: i18n.KeySessionSemanticModelLocalFileIdentityRequired,
				}
			}
		case semanticSourceCatalogFile:
			if source.FileID == "" || strings.TrimSpace(source.FileID) != source.FileID {
				return nil, semanticModelDependencyRequestError{err: fmt.Errorf("catalog file id is invalid")}
			}
			// Direct catalog_file writes require an authoritative volume_id.
			// File-root containment cannot disambiguate multi-volume files and
			// would fail closed before the service can honor the selected volume.
			if source.VolumeID <= 0 {
				return nil, semanticModelDependencyRequestError{err: fmt.Errorf("catalog file %s requires volume_id", source.FileID)}
			}
			volumeIDs = append(volumeIDs, source.VolumeID)
		case semanticSourceCatalogTable:
			if source.TableID <= 0 {
				return nil, fmt.Errorf("catalog table id is invalid")
			}
			tableIDs = append(tableIDs, source.TableID)
		default:
			return nil, fmt.Errorf("semantic model source type is invalid")
		}
	}

	out := make([]iampep.ResourceAuthorization, 0, len(volumeIDs)+len(tableIDs))
	if len(volumeIDs) > 0 {
		if r.Volumes == nil {
			return nil, fmt.Errorf("semantic model volume resolver is unavailable")
		}
		seenRoots := make(map[int64]struct{}, len(volumeIDs))
		for _, volumeID := range uniqueInt64s(volumeIDs) {
			rootID, err := r.Volumes.ResolveCanonicalRootVolume(ctx, workspaceID, volumeID)
			if err != nil {
				return nil, fmt.Errorf("resolve semantic model catalog file root volume: %w", err)
			}
			if rootID <= 0 {
				return nil, fmt.Errorf("semantic model catalog file has no canonical root volume")
			}
			if _, duplicate := seenRoots[rootID]; duplicate {
				continue
			}
			seenRoots[rootID] = struct{}{}
			out = append(out, iampep.ResourceAuthorization{
				ActionID: "volume.read",
				Resource: iampep.ResourceDescriptor{ResourceType: iampep.ResourceTypeVolume, ResourceID: strconv.FormatInt(rootID, 10)},
			})
		}
	}
	for _, tableID := range uniqueInt64s(tableIDs) {
		out = append(out, iampep.ResourceAuthorization{ActionID: "table.read", Resource: iampep.ResourceDescriptor{ResourceType: iampep.ResourceTypeTable, ResourceID: strconv.FormatInt(tableID, 10)}})
	}
	return out, nil
}

func uniqueInt64s(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	return out
}
