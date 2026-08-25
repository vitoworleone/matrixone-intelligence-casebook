package workitems

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestDeleteDocumentVisualImageIndexRowsExecsParameterizedDelete(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	query := regexp.QuoteMeta(`DELETE FROM image_index
WHERE JSON_UNQUOTE(JSON_EXTRACT(meta, '$.source_file_id')) = ?
  AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.embedding_model')) = ?
  AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.preprocess_version')) = ?
  AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.distance_metric')) = ?
  AND index_version = ?`)
	mock.ExpectExec(query).
		WithArgs("source-file", "model", "v1", "cosine", int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 3))

	deleted, err := deleteDocumentVisualImageIndexRows(
		context.Background(), db, "image_index", "source-file", "model", "v1", "cosine", 42,
	)
	if err != nil {
		t.Fatalf("deleteDocumentVisualImageIndexRows: %v", err)
	}
	if deleted != 3 {
		t.Fatalf("deleted=%d, want 3", deleted)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
