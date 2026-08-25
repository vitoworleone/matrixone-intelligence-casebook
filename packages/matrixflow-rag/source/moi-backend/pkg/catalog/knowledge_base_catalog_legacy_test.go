package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	corecatalog "github.com/matrixflow/moi-core/model/catalog"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
)

func TestDeleteCatalogRejectsKnowledgeBaseDataDomainReference(t *testing.T) {
	deleted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/catalogs":
			_ = json.NewEncoder(w).Encode(&corecatalog.ListCatalogsResponse{
				Items: []*corecatalog.Catalog{{Id: 7, Name: "ordinary"}},
				Total: 1,
			})
		case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/catalogs/7"):
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	sqlDB, mock, tenantDB := newMockGormDB(t)
	defer sqlDB.Close()
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `catalog_reserved_resource` WHERE resource_type = \\? AND resource_id = \\?").
		WithArgs(ReservedResourceTypeCatalog, "7").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `knowledge_base_data_domains` WHERE catalog_id = \\?").
		WithArgs(7).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	ctx := ctxutil.WithUID(context.Background(), "user-1")
	ctx = ctxutil.WithUserID(ctx, "user-1")
	ctx = ctxutil.WithWorkspaceID(ctx, "ws-1")
	ctx = ctxutil.WithTenantDB(ctx, tenantDB)
	ctx = verifiedCatalogAPIKeyContext(t, ctx, server.URL, "caller-key")

	svc := &dataCenterService{}
	err := svc.DeleteCatalog(ctx, &DeleteByIDRequest{ID: 7})
	if !errors.Is(err, ErrKnowledgeBaseCatalogInUse) {
		t.Fatalf("DeleteCatalog error = %v, want ErrKnowledgeBaseCatalogInUse", err)
	}
	if deleted {
		t.Fatal("DeleteCatalog sent remote delete for a referenced catalog")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet tenant SQL expectations: %v", err)
	}
}
