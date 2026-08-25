package workitems

import "testing"

func TestEnsureKnowledgeIndexVersionSetsDocumentMetadata(t *testing.T) {
	input := map[string]interface{}{
		"documents": []interface{}{
			map[string]interface{}{
				"content":  "first",
				"metadata": map[string]interface{}{"file_id": "file-1"},
			},
			map[string]interface{}{
				"content": "second",
			},
		},
	}

	version := ensureKnowledgeIndexVersion(input)

	if version <= 0 {
		t.Fatalf("version=%d, want positive", version)
	}
	if got := toInt64(input["index_version"], 0); got != version {
		t.Fatalf("input index_version=%d, want %d", got, version)
	}
	for _, raw := range input["documents"].([]interface{}) {
		doc := raw.(map[string]interface{})
		meta := ensureMap(doc["metadata"])
		if got := toInt64(meta["index_version"], 0); got != version {
			t.Fatalf("document metadata index_version=%d, want %d", got, version)
		}
	}
}

func TestEnsureKnowledgeIndexVersionKeepsExplicitVersion(t *testing.T) {
	input := map[string]interface{}{
		"index_version": int64(123),
		"documents": []interface{}{
			map[string]interface{}{
				"content":  "first",
				"metadata": map[string]interface{}{"file_id": "file-1"},
			},
		},
	}

	version := ensureKnowledgeIndexVersion(input)

	if version != 123 {
		t.Fatalf("version=%d, want explicit version", version)
	}
	doc := input["documents"].([]interface{})[0].(map[string]interface{})
	meta := ensureMap(doc["metadata"])
	if got := toInt64(meta["index_version"], 0); got != 123 {
		t.Fatalf("document metadata index_version=%d, want 123", got)
	}
}
