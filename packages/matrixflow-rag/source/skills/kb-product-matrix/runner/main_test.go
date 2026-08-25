package main

import (
	"testing"

	sdk "github.com/matrixorigin/matrixflow/sdk/go-sdk"
	product "github.com/matrixorigin/matrixflow/sdk/product-model/product"
)

func TestResolveDefaultCatalogID(t *testing.T) {
	tests := []struct {
		name     string
		catalogs []*product.CatalogDTO
		wantID   int64
		wantErr  bool
	}{
		{
			name: "finds default catalog",
			catalogs: []*product.CatalogDTO{
				{Id: 2, Name: "user-catalog"},
				{Id: 1, Name: defaultCatalogName},
			},
			wantID: 1,
		},
		{
			name: "rejects missing default catalog",
			catalogs: []*product.CatalogDTO{
				{Id: 2, Name: "user-catalog"},
			},
			wantErr: true,
		},
		{
			name: "rejects duplicate default catalogs",
			catalogs: []*product.CatalogDTO{
				{Id: 1, Name: defaultCatalogName},
				{Id: 2, Name: defaultCatalogName},
			},
			wantErr: true,
		},
		{
			name: "rejects invalid default catalog id",
			catalogs: []*product.CatalogDTO{
				{Id: 0, Name: defaultCatalogName},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveDefaultCatalogID(&sdk.CatalogListResult{List: tt.catalogs})
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveDefaultCatalogID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.wantID {
				t.Fatalf("resolveDefaultCatalogID() = %d, want %d", got, tt.wantID)
			}
		})
	}
}
