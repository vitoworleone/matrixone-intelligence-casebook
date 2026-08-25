package workitems

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/matrixflow/moi-core/model/data"
	"github.com/matrixflow/moi-core/model/mowl"
)

func TestMultiLevelIndexOrdersByParentThenLocalChunkIndex(t *testing.T) {
	docs := []*data.Document{
		mustDocumentProto(t, Document{
			Content: "second parent first chunk",
			Type:    "text",
			Metadata: map[string]interface{}{
				"file_id":      "file-1",
				"file_name":    "investor-record.pdf",
				"parent_index": 1,
				"chunk_index":  0,
			},
		}),
		mustDocumentProto(t, Document{
			Content: "first parent first chunk",
			Type:    "text",
			Metadata: map[string]interface{}{
				"file_id":      "file-1",
				"file_name":    "investor-record.pdf",
				"parent_index": 0,
				"chunk_index":  0,
			},
		}),
		mustDocumentProto(t, Document{
			Content: "first parent second chunk",
			Type:    "text",
			Metadata: map[string]interface{}{
				"file_id":      "file-1",
				"file_name":    "investor-record.pdf",
				"parent_index": 0,
				"chunk_index":  1,
			},
		}),
	}
	req := &data.SplitDocumentsLengthResponse{Documents: docs}
	raw, err := json.Marshal(map[string]interface{}{
		"documents":     req.Documents,
		"enable":        true,
		"section_size":  5,
		"index_version": 177,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	msg := &mowl.MowlMessage{Data: string(raw)}
	_, err = (&MultiLevelIndex{}).Handle(context.Background(), nil, msg)
	if err != nil {
		t.Fatalf("MultiLevelIndex.Handle: %v", err)
	}

	var out data.SplitDocumentsLengthResponse
	if err := json.Unmarshal([]byte(msg.Data), &out); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	var chunks []Document
	for _, p := range out.Documents {
		doc := documentFromProto(p)
		if toString(doc.Metadata["level"]) == "chunk" {
			chunks = append(chunks, doc)
		}
	}
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunk entries, got %d", len(chunks))
	}
	wantContent := []string{"first parent first chunk", "first parent second chunk", "second parent first chunk"}
	for i, want := range wantContent {
		if chunks[i].Content != want {
			t.Fatalf("chunk[%d] content=%q, want %q", i, chunks[i].Content, want)
		}
		if got := toInt(chunks[i].Metadata["chunk_index"], -1); got != i {
			t.Fatalf("chunk[%d] global chunk_index=%d, want %d", i, got, i)
		}
	}
}

func TestMultiLevelIndexRejectsParserDiagnosticDocument(t *testing.T) {
	docs := []*data.Document{
		mustDocumentProto(t, Document{
			Content: `parse error: invalid JSON response`,
			Type:    "text",
			Metadata: map[string]interface{}{
				"file_id":     "file-1",
				"file_name":   "broken.pdf",
				"chunk_index": 0,
			},
		}),
	}
	raw, err := json.Marshal(map[string]interface{}{
		"documents": docs,
		"enable":    true,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	_, err = (&MultiLevelIndex{}).Handle(context.Background(), nil, &mowl.MowlMessage{Data: string(raw)})
	if err == nil {
		t.Fatalf("expected parser diagnostic document to fail")
	}
	for _, want := range []string{"parser diagnostic output", "file_id=file-1", "parse error"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func mustDocumentProto(t *testing.T, doc Document) *data.Document {
	t.Helper()
	p, err := documentToProto(doc)
	if err != nil {
		t.Fatalf("documentToProto: %v", err)
	}
	return p
}
