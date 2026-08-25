package workitems

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matrixflow/moi-core/workers/go-worker/pkg/runtime"
)

func TestDocumentVisualCodexSidecarValidation(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(tmpDir, "outputs", "drawing_assets", "images", "page_1.png"), minimalPNGBytes())
	mustWriteTestFile(t, filepath.Join(tmpDir, "outputs", "drawing_assets", "images", "object_1.png"), minimalPNGBytes())

	sidecar := documentVisualCodexSidecar{
		Pages: []documentVisualCodexPage{
			{
				PageNumber:        1,
				FullPageImagePath: "outputs/drawing_assets/images/page_1.png",
				Width:             100,
				Height:            200,
				BBox:              []float64{0, 0, 100, 200},
			},
		},
		Objects: []documentVisualCodexObject{
			{
				ObjectID:   "object_1",
				ObjectKind: "view",
				PageNumber: 1,
				BBox:       []float64{10, 10, 50, 50},
				ImagePath:  "outputs/drawing_assets/images/object_1.png",
				Caption:    "front view",
				Text:       "10 mm",
				Context:    "front view dimension",
			},
		},
		Entities: []documentVisualCodexEntity{
			{
				EntityID:   "entity_1",
				Type:       "dimension",
				Value:      "10 mm",
				PageNumber: 1,
				ObjectID:   "object_1",
				Evidence:   "10 mm",
			},
		},
	}
	if err := validateDocumentVisualCodexSidecar(sidecar, tmpDir); err != nil {
		t.Fatalf("validate sidecar: %v", err)
	}

	manifest, err := buildDocumentVisualCodexManifest(
		documentVisualSource{FileID: "source-file", FileName: "drawing.pdf", MimeType: "application/pdf"},
		documentVisualDefaultProfile,
		sidecar,
		map[int]documentVisualCodexUploadedArtifact{1: {FileID: "page-file"}},
		map[string]documentVisualCodexUploadedArtifact{"object_1": {FileID: "object-file"}},
	)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if _, err := buildDocumentVisualTextIndexDocuments(manifest); err != nil {
		t.Fatalf("manifest should feed text index: %v", err)
	}
	if manifest.Pages[0].PageImageFileID == "" {
		t.Fatalf("page image file id is required by image index")
	}
	if manifest.Objects[0].ImageFileID == "" || manifest.Objects[0].PageImageFileID == "" || len(manifest.Objects[0].BBox) != 4 {
		t.Fatalf("object image metadata is incomplete for image index: %+v", manifest.Objects[0])
	}
}

func TestDocumentVisualCodexSidecarRejectsInvalidArtifacts(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(tmpDir, "outputs", "page.png"), minimalPNGBytes())
	mustWriteTestFile(t, filepath.Join(tmpDir, "outputs", "object.png"), minimalPNGBytes())

	base := documentVisualCodexSidecar{
		Pages: []documentVisualCodexPage{
			{PageNumber: 1, FullPageImagePath: "outputs/page.png", Width: 100, Height: 100, BBox: []float64{0, 0, 100, 100}},
		},
		Objects: []documentVisualCodexObject{
			{ObjectID: "object_1", ObjectKind: "view", PageNumber: 1, BBox: []float64{0, 0, 10, 10}, ImagePath: "outputs/object.png", Context: "ctx"},
		},
	}
	tests := []struct {
		name    string
		mutate  func(documentVisualCodexSidecar) documentVisualCodexSidecar
		wantErr string
	}{
		{
			name: "missing image",
			mutate: func(s documentVisualCodexSidecar) documentVisualCodexSidecar {
				s.Objects[0].ImagePath = "outputs/missing.png"
				return s
			},
			wantErr: "not found",
		},
		{
			name: "path escape",
			mutate: func(s documentVisualCodexSidecar) documentVisualCodexSidecar {
				s.Objects[0].ImagePath = "../object.png"
				return s
			},
			wantErr: "escapes",
		},
		{
			name: "bad bbox",
			mutate: func(s documentVisualCodexSidecar) documentVisualCodexSidecar {
				s.Objects[0].BBox = []float64{1, 2, 3}
				return s
			},
			wantErr: "4 numbers",
		},
		{
			name: "missing context",
			mutate: func(s documentVisualCodexSidecar) documentVisualCodexSidecar {
				s.Objects[0].Context = ""
				return s
			},
			wantErr: "context is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDocumentVisualCodexSidecar(tt.mutate(cloneDocumentVisualCodexSidecarForTest(base)), tmpDir)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want contains %q", err, tt.wantErr)
			}
		})
	}
}

func TestRewriteDocumentVisualCodexMarkdownReferences(t *testing.T) {
	md := []byte("![view](outputs/drawing_assets/images/object_1.png)\n<img src=\"page_1.png\">")
	out := rewriteDocumentVisualCodexMarkdownReferences(md, map[string]string{
		markdownImageAliasKey("outputs/drawing_assets/images/object_1.png"): "object-file",
		markdownImageAliasKey("page_1.png"):                                 "page-file",
	})
	got := string(out)
	if !strings.Contains(got, "![view](object-file)") {
		t.Fatalf("markdown image not rewritten: %s", got)
	}
	if !strings.Contains(got, "<img src=\"page-file\">") {
		t.Fatalf("html image not rewritten: %s", got)
	}
}

func TestDocumentVisualCodexDocumentsUseRewrittenMarkdown(t *testing.T) {
	manifest := documentVisualManifest{
		SchemaVersion: documentVisualSchemaVersion,
		Profile:       documentVisualDefaultProfile,
		Source: documentVisualSource{
			FileID:   "source-file",
			FileName: "drawing.pdf",
		},
		Pages: []documentVisualPage{
			{PageNumber: 1, Width: 100, Height: 100, PageImageFileID: "page-file"},
		},
		Objects: []documentVisualObject{
			{
				ObjectID:        "object_1",
				ObjectKind:      "view",
				PageNumber:      1,
				BBox:            []float64{0, 0, 50, 50},
				OCRBBox:         []float64{0, 0, 50, 50},
				ImageFileID:     "object-file",
				PageImageFileID: "page-file",
				Caption:         "front view",
				Context:         "front view context",
			},
		},
	}
	sidecar := documentVisualCodexSidecar{
		Pages: []documentVisualCodexPage{
			{PageNumber: 1, FullPageImagePath: "drawing_assets/images/page_1.png"},
		},
		Objects: []documentVisualCodexObject{
			{ObjectID: "object_1", ImagePath: "drawing_assets/images/object_1.png"},
		},
	}

	mdBytes := rewriteDocumentVisualCodexMarkdownReferences(
		[]byte("![view](drawing_assets/images/object_1.png)"),
		map[string]string{markdownImageAliasKey("drawing_assets/images/object_1.png"): "object-file"},
	)
	docs, err := buildDocumentVisualCodexDocuments(manifest, sidecar, "md-file", "manifest-file", string(mdBytes))
	if err != nil {
		t.Fatalf("build documents: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("documents count = %d, want 1", len(docs))
	}
	if got := docs[0].GetContent(); !strings.Contains(got, "![view](object-file)") {
		t.Fatalf("document content was not rewritten: %s", got)
	}
	if strings.Contains(docs[0].GetContent(), "drawing_assets/images/object_1.png") {
		t.Fatalf("document content still contains local image path: %s", docs[0].GetContent())
	}
}

func TestBuildDocumentVisualCodexDocumentsOutputsAggregatedMarkdownOnly(t *testing.T) {
	manifest := documentVisualManifest{
		SchemaVersion: documentVisualSchemaVersion,
		Profile:       documentVisualDefaultProfile,
		Source: documentVisualSource{
			FileID:   "source-file",
			FileName: "drawing.pdf",
		},
		Pages: []documentVisualPage{
			{
				PageNumber:      1,
				Width:           100,
				Height:          100,
				PageImageFileID: "page-file",
			},
		},
		Objects: []documentVisualObject{
			{
				ObjectID:        "object_1",
				ObjectKind:      "view",
				PageNumber:      1,
				BBox:            []float64{0, 0, 50, 50},
				OCRBBox:         []float64{0, 0, 50, 50},
				ImageFileID:     "object-file",
				PageImageFileID: "page-file",
				Caption:         "front view",
				Context:         "front view context",
			},
		},
	}
	sidecar := documentVisualCodexSidecar{
		Pages: []documentVisualCodexPage{
			{PageNumber: 1, FullPageImagePath: "1433113_assets/images/page_1.png"},
		},
		Objects: []documentVisualCodexObject{
			{ObjectID: "object_1", ImagePath: "1433113_assets/images/object_1.png"},
		},
	}

	docs, err := buildDocumentVisualCodexDocuments(manifest, sidecar, "md-file", "manifest-file", "# drawing")
	if err != nil {
		t.Fatalf("build documents: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("documents count = %d, want 1 aggregated markdown document", len(docs))
	}
	doc := docs[0]
	if doc.GetType() != "text" {
		t.Fatalf("document type = %q, want text", doc.GetType())
	}
	if doc.GetContent() != "# drawing" {
		t.Fatalf("document content = %q", doc.GetContent())
	}
	meta := doc.GetMetadata().AsMap()
	if got := meta["block_type"]; got != "codex_markdown" {
		t.Fatalf("block_type = %v, want codex_markdown", got)
	}
	if got := meta["md_file_id"]; got != "md-file" {
		t.Fatalf("md_file_id = %v, want md-file", got)
	}
	if got := meta["manifest_file_id"]; got != "manifest-file" {
		t.Fatalf("manifest_file_id = %v, want manifest-file", got)
	}
	rawArtifacts, ok := meta["image_artifacts"].(string)
	if !ok {
		t.Fatalf("image_artifacts = %#v, want page and object image artifacts", meta["image_artifacts"])
	}
	var artifacts []map[string]interface{}
	if err := json.Unmarshal([]byte(rawArtifacts), &artifacts); err != nil {
		t.Fatalf("parse image_artifacts: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("image_artifacts count = %d, want 2", len(artifacts))
	}
	if got := artifacts[1]["local_path"]; got != "1433113_assets/images/object_1.png" {
		t.Fatalf("object local_path = %v, want 1433113_assets/images/object_1.png", got)
	}
}

func TestDocumentVisualCodexCommandArgs(t *testing.T) {
	args := documentVisualCodexCommandArgs(DocumentVisualCodexConfig{
		Command:         []string{"/usr/local/bin/codex", "exec", "--json", "-"},
		Model:           "gpt-5.5",
		ReasoningEffort: "high",
	}, documentVisualCodexOverrides{})
	got := strings.Join(args, " ")
	if !strings.Contains(got, "--model gpt-5.5") {
		t.Fatalf("model arg missing: %v", args)
	}
	if !strings.Contains(got, "-c model_reasoning_effort=\"high\"") {
		t.Fatalf("reasoning effort arg missing: %v", args)
	}
	if args[len(args)-1] != "-" {
		t.Fatalf("stdin marker should remain last: %v", args)
	}
}

func TestDocumentVisualCodexCommandArgsUsesReasoningEffortOverride(t *testing.T) {
	args := documentVisualCodexCommandArgs(DocumentVisualCodexConfig{
		Command:         []string{"/usr/local/bin/codex", "exec", "--json", "-"},
		ReasoningEffort: "low",
	}, documentVisualCodexOverrides{
		ReasoningEffort: "high",
	})
	got := strings.Join(args, " ")
	if strings.Contains(got, `model_reasoning_effort="low"`) {
		t.Fatalf("reasoning effort override should replace worker config: %v", args)
	}
	if !strings.Contains(got, `-c model_reasoning_effort="high"`) {
		t.Fatalf("reasoning effort override missing: %v", args)
	}
}

func TestDocumentVisualCodexCommandArgsReplacesExistingReasoningEffortConfig(t *testing.T) {
	args := documentVisualCodexCommandArgs(DocumentVisualCodexConfig{
		Command: []string{"/usr/local/bin/codex", "exec", "-c", `model_reasoning_effort="low"`, "--json", "-"},
	}, documentVisualCodexOverrides{
		ReasoningEffort: "high",
	})
	got := strings.Join(args, " ")
	if strings.Contains(got, `model_reasoning_effort="low"`) {
		t.Fatalf("reasoning effort command config should be replaced: %v", args)
	}
	if !strings.Contains(got, `-c model_reasoning_effort="high"`) {
		t.Fatalf("replacement reasoning effort missing: %v", args)
	}
}

func TestPrepareDocumentVisualCodexOverrideHomeRejectsMissingBaseURLConfig(t *testing.T) {
	sourceHome := t.TempDir()
	workDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(sourceHome, "config.toml"), []byte(`[model_providers.OpenAI]
model = "gpt-5.5"
`))
	mustWriteTestFile(t, filepath.Join(sourceHome, "auth.json"), []byte(`{"OPENAI_API_KEY":"default-secret"}`))

	_, err := prepareDocumentVisualCodexOverrideHome(workDir, []string{"CODEX_HOME=" + sourceHome}, documentVisualCodexOverrides{
		BaseURL: "https://example.test/v1",
	})
	if err == nil || !strings.Contains(err.Error(), "missing base_url in [model_providers.OpenAI]") {
		t.Fatalf("error = %v, want missing base_url config", err)
	}
}

func TestPrepareDocumentVisualCodexOverrideHomeRejectsMissingReasoningEffortConfig(t *testing.T) {
	sourceHome := t.TempDir()
	workDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(sourceHome, "config.toml"), []byte(`model = "gpt-5.5"

[model_providers.OpenAI]
name = "OpenAI"
base_url = "https://default.test/v1"
`))
	mustWriteTestFile(t, filepath.Join(sourceHome, "auth.json"), []byte(`{"OPENAI_API_KEY":"default-secret"}`))

	_, err := prepareDocumentVisualCodexOverrideHome(workDir, []string{"CODEX_HOME=" + sourceHome}, documentVisualCodexOverrides{
		ReasoningEffort: "high",
	})
	if err == nil || !strings.Contains(err.Error(), "missing model_reasoning_effort for override") {
		t.Fatalf("error = %v, want missing model_reasoning_effort config", err)
	}
}

func TestDocumentVisualCodexModelSelectionRequiresBackendID(t *testing.T) {
	_, _, err := documentVisualCodexModelSelectionFromOptions(map[string]interface{}{
		"model": "gpt-5.5",
	})
	if err == nil || !strings.Contains(err.Error(), "options.backend_id is required") {
		t.Fatalf("error = %v, want backend_id required", err)
	}
}

func TestDocumentVisualCodexModelSelectionRequiresModel(t *testing.T) {
	_, _, err := documentVisualCodexModelSelectionFromOptions(map[string]interface{}{
		"backend_id": float64(1),
	})
	if err == nil || !strings.Contains(err.Error(), "options.model is required") {
		t.Fatalf("error = %v, want model required", err)
	}
}

func TestDocumentVisualCodexModelSelectionUsesBackendID(t *testing.T) {
	model, backendID, err := documentVisualCodexModelSelectionFromOptions(map[string]interface{}{
		"model":      "gpt-5.5",
		"backend_id": float64(1),
	})
	if err != nil {
		t.Fatalf("selection: %v", err)
	}
	if model != "gpt-5.5" || backendID != 1 {
		t.Fatalf("model=%q backendID=%d", model, backendID)
	}
}

func TestDocumentVisualCodexReasoningEffortFromOptions(t *testing.T) {
	got := documentVisualCodexReasoningEffortFromOptions(map[string]interface{}{
		"model_reasoning_effort": "high",
	})
	if got != "high" {
		t.Fatalf("reasoning effort = %q, want high", got)
	}
}

func TestValidateDocumentVisualCodexModelAccessUsesUserModelList(t *testing.T) {
	var gotPath string
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		if r.URL.Path != "/api/v1/workspaces/ws-1/llm/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("type") != "chat" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"models":[{"model":"gpt-5.5","backend_id":1,"backend_name":"gpt-5.5","model_type":"chat"}]}}`))
	}))
	defer server.Close()
	factory := runtime.NewClientFactory(server.URL, "worker-key", time.Minute)
	client, err := factory.Get(&runtime.ExecutionContext{WorkspaceId: "ws-test", UserId: "user-1", UserApiKey: "user-key", EffectiveRoleId: "42"})
	if err != nil {
		t.Fatalf("create user client: %v", err)
	}

	if err := validateDocumentVisualCodexModelAccess(context.Background(), client, "ws-1", "gpt-5.5", 1); err != nil {
		t.Fatalf("validate access: %v", err)
	}
	if gotPath != "/api/v1/workspaces/ws-1/llm/models" || gotQuery != "type=chat" {
		t.Fatalf("unexpected request: path=%s query=%s", gotPath, gotQuery)
	}
}

func TestValidateDocumentVisualCodexModelAccessRejectsUnavailableSelection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"models":[{"model":"gpt-5.5","backend_id":2,"backend_name":"other","model_type":"chat"}]}}`))
	}))
	defer server.Close()
	factory := runtime.NewClientFactory(server.URL, "worker-key", time.Minute)
	client, err := factory.Get(&runtime.ExecutionContext{WorkspaceId: "ws-test", UserId: "user-1", UserApiKey: "user-key", EffectiveRoleId: "42"})
	if err != nil {
		t.Fatalf("create user client: %v", err)
	}

	err = validateDocumentVisualCodexModelAccess(context.Background(), client, "ws-1", "gpt-5.5", 1)
	if err == nil || !strings.Contains(err.Error(), "is not available to the execution user") {
		t.Fatalf("error = %v, want unavailable selection", err)
	}
}

func TestDocumentVisualCodexResolveOverridesUsesBackendID(t *testing.T) {
	var gotPath string
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		if r.URL.Path != "/api/v1/system/workspaces/ws-1/llm/route" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("model") != "gpt-5.5" || r.URL.Query().Get("backend_id") != "1" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"data":{"backend_id":1,"endpoint_id":11,"endpoint_address":"https://llm.example/v1","api_key_encrypted":"secret-key","model":"gpt-5.5"}}`))
	}))
	defer server.Close()
	item := &DocumentVisualParseCodex{Factory: runtime.NewClientFactory(server.URL, "worker-key", time.Minute)}

	overrides, err := item.resolveDocumentVisualCodexOverrides(context.Background(), "ws-1", "gpt-5.5", 1)
	if err != nil {
		t.Fatalf("resolve overrides: %v", err)
	}
	if overrides.Model != "gpt-5.5" || overrides.BaseURL != "https://llm.example/v1" || overrides.APIKey != "secret-key" {
		t.Fatalf("overrides = %+v", overrides)
	}
	if !strings.Contains(gotPath+"?"+gotQuery, "backend_id=1") {
		t.Fatalf("resolver did not request backend_id: path=%s query=%s", gotPath, gotQuery)
	}
}

func TestPrepareDocumentVisualCodexOverrideHomeUpdatesConcreteTemporaryCodexHome(t *testing.T) {
	sourceHome := t.TempDir()
	workDir := t.TempDir()
	mustWriteTestFile(t, filepath.Join(sourceHome, "config.toml"), []byte(`model = "default-model"
model_reasoning_effort = "low"

[model_providers.OpenAI]
name = "OpenAI"
base_url = "https://default.test/v1"
`))
	mustWriteTestFile(t, filepath.Join(sourceHome, "auth.json"), []byte(`{
  "OPENAI_API_KEY": "default-secret",
  "other": "kept"
}
`))

	targetHome, err := prepareDocumentVisualCodexOverrideHome(workDir, []string{"CODEX_HOME=" + sourceHome}, documentVisualCodexOverrides{
		APIKey:          "sk-test-secret",
		BaseURL:         "https://example.test/v1",
		Model:           "gpt-5.5",
		ReasoningEffort: "high",
	})
	if err != nil {
		t.Fatalf("prepare override home: %v", err)
	}
	defer os.RemoveAll(targetHome)
	if rel, err := filepath.Rel(workDir, targetHome); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != ".." {
		t.Fatalf("temporary CODEX_HOME must not be inside workDir: workDir=%s targetHome=%s rel=%s", workDir, targetHome, rel)
	}
	if _, err := os.Stat(filepath.Join(workDir, ".codex-runtime")); !os.IsNotExist(err) {
		t.Fatalf("workDir must not contain temporary CODEX_HOME, stat err=%v", err)
	}
	sourceConfig, err := os.ReadFile(filepath.Join(sourceHome, "config.toml"))
	if err != nil {
		t.Fatalf("read source config: %v", err)
	}
	if !strings.Contains(string(sourceConfig), `base_url = "https://default.test/v1"`) {
		t.Fatalf("source config was modified: %s", sourceConfig)
	}
	if !strings.Contains(string(sourceConfig), `model = "default-model"`) {
		t.Fatalf("source config was modified: %s", sourceConfig)
	}
	if !strings.Contains(string(sourceConfig), `model_reasoning_effort = "low"`) {
		t.Fatalf("source config was modified: %s", sourceConfig)
	}
	targetConfig, err := os.ReadFile(filepath.Join(targetHome, "config.toml"))
	if err != nil {
		t.Fatalf("read target config: %v", err)
	}
	if !strings.Contains(string(targetConfig), `base_url = "https://example.test/v1"`) {
		t.Fatalf("target config missing base_url override: %s", targetConfig)
	}
	if !strings.Contains(string(targetConfig), `model = "gpt-5.5"`) {
		t.Fatalf("target config missing model override: %s", targetConfig)
	}
	if !strings.Contains(string(targetConfig), `model_reasoning_effort = "high"`) {
		t.Fatalf("target config missing model_reasoning_effort override: %s", targetConfig)
	}

	targetAuth, err := os.ReadFile(filepath.Join(targetHome, "auth.json"))
	if err != nil {
		t.Fatalf("read target auth: %v", err)
	}
	if strings.Contains(string(targetAuth), "default-secret") {
		t.Fatalf("target auth still contains default secret: %s", targetAuth)
	}
	if !strings.Contains(string(targetAuth), `"OPENAI_API_KEY": "sk-test-secret"`) {
		t.Fatalf("target auth missing api_key override: %s", targetAuth)
	}
	if !strings.Contains(string(targetAuth), `"other": "kept"`) {
		t.Fatalf("target auth should preserve other fields: %s", targetAuth)
	}
}

func TestDocumentVisualCodexRedactsOverrideValues(t *testing.T) {
	out := redactDocumentVisualCodexOutput("stderr sk-test-secret https://example.test/v1", documentVisualCodexOverrides{
		APIKey:  "sk-test-secret",
		BaseURL: "https://example.test/v1",
	})
	if strings.Contains(out, "sk-test-secret") || strings.Contains(out, "https://example.test/v1") {
		t.Fatalf("redacted output leaked override values: %s", out)
	}
	if !strings.Contains(out, "[REDACTED]") || !strings.Contains(out, "[REDACTED_BASE_URL]") {
		t.Fatalf("redacted output missing markers: %s", out)
	}
}

func TestDocumentVisualCodexJSONEventType(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "top level type",
			line: `{"type":"session.started"}`,
			want: "session.started",
		},
		{
			name: "nested message type",
			line: `{"msg":{"type":"agent_reasoning_delta"}}`,
			want: "agent_reasoning_delta",
		},
		{
			name: "invalid json",
			line: `not-json`,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := documentVisualCodexJSONEventType([]byte(tt.line)); got != tt.want {
				t.Fatalf("event type = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDocumentVisualCodexPromptRequiresDrawingDimensionAttribution(t *testing.T) {
	prompt := documentVisualCodexPrompt("input/1433092.pdf", "1433092")
	for _, required := range []string{
		"括号内参考尺寸",
		"弧向尺寸",
		"总高度",
		"不能把剖面视图或局部视图的尺寸归到主视图",
		"边界归属说明",
		"裁切检查记录",
		"object_kind 为 table",
		"Markdown 表格还原原表行列和表头",
		"不能只用一段自然语言描述代替原表格",
		"objects.context",
		"entities",
	} {
		if !strings.Contains(prompt, required) {
			t.Fatalf("prompt does not contain %q:\n%s", required, prompt)
		}
	}
}

func mustWriteTestFile(t *testing.T, filename string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func cloneDocumentVisualCodexSidecarForTest(in documentVisualCodexSidecar) documentVisualCodexSidecar {
	out := in
	out.Pages = append([]documentVisualCodexPage(nil), in.Pages...)
	for i := range out.Pages {
		out.Pages[i].BBox = append([]float64(nil), out.Pages[i].BBox...)
	}
	out.Objects = append([]documentVisualCodexObject(nil), in.Objects...)
	for i := range out.Objects {
		out.Objects[i].BBox = append([]float64(nil), out.Objects[i].BBox...)
	}
	out.Entities = append([]documentVisualCodexEntity(nil), in.Entities...)
	return out
}

func minimalPNGBytes() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
		0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
		0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x03, 0x01, 0x01, 0x00, 0x18, 0xdd, 0x8d,
		0xb0, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
		0x44, 0xae, 0x42, 0x60, 0x82,
	}
}
