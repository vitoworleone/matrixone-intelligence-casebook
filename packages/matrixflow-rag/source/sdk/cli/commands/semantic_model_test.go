package commands

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/matrixorigin/matrixflow/sdk/go-sdk"
)

func TestCreateWithSourcesImageIndexEnabledSendsRequestBody(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Errorf("decode body: %v raw=%s", err, string(raw))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "OK",
			"msg":  "OK",
			"data": map[string]any{
				"model":   map[string]any{"id": 101, "name": "kb-image"},
				"sources": []any{},
				"jobs":    []any{},
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := sdk.New(server.URL+"/newmoi", sdk.WithAPIKey("user-key"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext(client, "ws-1", false, true, true)

	if err := runKnowledgeServiceCreateWithSources(ctx, []string{
		"--name", "kb-image",
		"--image-index-enabled",
		"--json", `[]`,
	}); err != nil {
		t.Fatalf("create-with-sources --image-index-enabled: %v", err)
	}

	if gotPath != "/newmoi/semantic-models/create-with-sources" {
		t.Fatalf("path = %q, want /newmoi/semantic-models/create-with-sources", gotPath)
	}
	if got := gotBody["image_index_enabled"]; got != true {
		t.Fatalf("image_index_enabled = %#v, want true; body=%#v", got, gotBody)
	}
	if got := gotBody["name"]; got != "kb-image" {
		t.Fatalf("name = %#v, want kb-image", got)
	}
}

func TestCreateWithSourcesOmitsImageIndexEnabledByDefault(t *testing.T) {
	t.Parallel()

	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Errorf("decode body: %v raw=%s", err, string(raw))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "OK",
			"msg":  "OK",
			"data": map[string]any{
				"model":   map[string]any{"id": 101, "name": "kb-text"},
				"sources": []any{},
				"jobs":    []any{},
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := sdk.New(server.URL+"/newmoi", sdk.WithAPIKey("user-key"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext(client, "ws-1", false, true, true)

	if err := runKnowledgeServiceCreateWithSources(ctx, []string{
		"--name", "kb-text",
		"--json", `[]`,
	}); err != nil {
		t.Fatalf("create-with-sources default: %v", err)
	}

	if _, ok := gotBody["image_index_enabled"]; ok {
		t.Fatalf("image_index_enabled must be omitted by default; body=%#v", gotBody)
	}
}

func TestCreateEmptySendsDataKnowledgeBaseOptions(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := json.Unmarshal(raw, &gotBody); err != nil {
			t.Errorf("decode body: %v raw=%s", err, string(raw))
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "OK",
			"msg":  "OK",
			"data": map[string]any{
				"model":       map[string]any{"id": 101, "name": "kb-empty"},
				"data_domain": map[string]any{"model_id": 101},
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := sdk.New(server.URL+"/newmoi", sdk.WithAPIKey("user-key"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := NewContext(client, "ws-1", false, true, true)

	if err := ExecuteSemanticModelCommand(ctx, []string{
		"create-empty",
		"--name", "kb-empty",
		"--description", "data knowledge base",
		"--image-index-enabled",
	}); err != nil {
		t.Fatalf("create-empty: %v", err)
	}

	if gotPath != "/newmoi/semantic-models/create-empty" {
		t.Fatalf("path = %q, want /newmoi/semantic-models/create-empty", gotPath)
	}
	if got := gotBody["name"]; got != "kb-empty" {
		t.Fatalf("name = %#v, want kb-empty", got)
	}
	if got := gotBody["description"]; got != "data knowledge base" {
		t.Fatalf("description = %#v, want data knowledge base", got)
	}
	if got := gotBody["image_index_enabled"]; got != true {
		t.Fatalf("image_index_enabled = %#v, want true", got)
	}
}
