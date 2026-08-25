package catalog

import (
	"context"
	"errors"
	"fmt"

	moi "github.com/matrixflow/moi-core/go-sdk"
	catalogpb "github.com/matrixflow/moi-core/model/catalog"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/util"
)

const knowledgeBaseCatalogOwnerMarker = "moi-backend:knowledge-base-catalog:v1"

// ErrKnowledgeBaseCatalogInUse prevents deleting a legacy Catalog that still
// carries knowledge-base ownership or data-domain references.
var ErrKnowledgeBaseCatalogInUse = errors.New("knowledge base catalog is in use")

func hasKnowledgeBaseCatalogDisplayMapping(item *catalogpb.Catalog) bool {
	if item == nil {
		return false
	}
	for _, binding := range item.GetDisplayBindings() {
		if binding != nil && binding.GetField() == displayFieldName &&
			binding.GetDisplayOwner() == DisplayOwnerBackend &&
			binding.GetDisplayKey() == i18n.KeyKnowledgeBaseCatalogName.String() {
			return true
		}
	}
	return false
}

func listWorkspaceCatalogs(ctx context.Context, client *moi.Client, workspaceID string) ([]*catalogpb.Catalog, error) {
	return util.FetchAllPages(func(opts ...moi.ListOption) (*util.PagedResult[*catalogpb.Catalog], error) {
		resp, err := client.Catalogs().List(ctx, workspaceID, opts...)
		if err != nil || resp == nil {
			return nil, err
		}
		return &util.PagedResult[*catalogpb.Catalog]{Items: resp.Items, NextPageToken: resp.NextPageToken}, nil
	})
}

func knowledgeBaseCatalogInUseError(catalogID int) error {
	return fmt.Errorf("catalog %d: %w", catalogID, ErrKnowledgeBaseCatalogInUse)
}
