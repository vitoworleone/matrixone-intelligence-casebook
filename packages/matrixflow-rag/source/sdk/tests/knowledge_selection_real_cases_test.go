package tests

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	sdk "github.com/matrixorigin/matrixflow/sdk/go-sdk"
	"github.com/matrixorigin/matrixflow/sdk/tests/framework"
)

// TestProductSDKKnowledgeSelectionRealCases verifies knowledge upload,
// selection preview, validation, and legacy reconciliation against the same
// fixture-owned Catalog file and semantic model.
func TestProductSDKKnowledgeSelectionRealCases(t *testing.T) {

	framework.RunProductSDKTests(t, func(env *framework.TestEnv) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		fixture := env.NewKnowledgeFixture(ctx, t, "knowledge-selection")
		selectedFile, err := fixture.Catalog.UploadBytes(ctx, "knowledge-selection.txt", []byte("Product SDK knowledge selection fixture"))
		if err != nil {
			t.Fatalf("upload knowledge selection file: %v", err)
		}
		selection := []sdk.SemanticModelSourceSelectionInput{{
			Kind:            "volume_files",
			VolumeID:        int64(fixture.Catalog.Volume.ID()),
			SelectedFileIDs: []string{selectedFile.ID()},
		}}

		exerciseGoKnowledgeSelection(t, ctx, fixture, selection)
		exercisePythonKnowledgeSelection(t, ctx, env, fixture.Catalog.Workspace.ID(), fixture.Model.ID(), fixture.Catalog.Volume.ID(), selectedFile.ID())

		for _, route := range []productRoute{
			{method: "POST", path: "/newmoi/semantic-models/local-files/upload"},
			{method: "POST", path: "/newmoi/semantic-models/source-selections/preview"},
			{method: "POST", path: "/newmoi/semantic-models/:model_id/source-selections/preview"},
			{method: "POST", path: "/newmoi/semantic-models/:model_id/sources/backfill-legacy"},
			{method: "POST", path: "/newmoi/semantic-models/:model_id/validate"},
		} {
			env.RequireRealResponse(t, route.method, route.path)
			env.RequirePythonSDKRealResponse(t, route.method, route.path)
		}
	})
}

func exerciseGoKnowledgeSelection(t *testing.T, ctx context.Context, fixture *framework.KnowledgeFixture, selections []sdk.SemanticModelSourceSelectionInput) {
	t.Helper()
	knowledge := fixture.Model.Service()

	uploaded, err := knowledge.UploadLocalFile(ctx, "go-knowledge-selection.txt", strings.NewReader("Go knowledge selection upload"))
	if err != nil || uploaded.GetFileId() == "" {
		t.Fatalf("upload Go global knowledge local file: result=%#v err=%v", uploaded, err)
	}
	globalPreview, err := knowledge.PreviewSourceSelectionCounts(ctx, selections)
	requireKnowledgeSelectionPreview(t, globalPreview, err, "Go global")
	modelPreview, err := knowledge.PreviewModelSourceSelectionCounts(ctx, fixture.Model.ID(), selections)
	requireKnowledgeSelectionPreview(t, modelPreview, err, "Go model")

	backfill, err := knowledge.BackfillLegacySources(ctx, fixture.Model.ID())
	if err != nil || !backfill.GetUpdated() {
		t.Fatalf("backfill Go legacy knowledge sources: result=%#v err=%v", backfill, err)
	}
	validated, err := knowledge.Validate(ctx, fixture.Model.ID())
	if err != nil || !validated.GetValid() || len(validated.GetErrors()) != 0 {
		t.Fatalf("validate Go knowledge model: result=%#v err=%v", validated, err)
	}
}

func requireKnowledgeSelectionPreview(t *testing.T, preview *sdk.SemanticModelSourceSelectionPreviewResult, err error, label string) {
	t.Helper()
	if err != nil || preview.GetFileCount() != 1 || preview.GetTableCount() != 0 || preview.GetTotalCount() != 1 {
		t.Fatalf("%s knowledge selection preview: result=%#v err=%v", label, preview, err)
	}
}

func exercisePythonKnowledgeSelection(t *testing.T, ctx context.Context, env *framework.TestEnv, workspaceID, modelID string, volumeID int, fileID string) {
	t.Helper()
	script := `
import sys
import moi_product_sdk as sdk

endpoint, personal_access_token, workspace_id, model_id, volume_id, file_id = sys.argv[1:]
client = sdk.new_with_personal_access_token(endpoint, personal_access_token)
knowledge = client.knowledge(workspace_id)
uploaded = knowledge.upload_local_file("python-knowledge-selection.txt", b"Python knowledge selection upload")
assert uploaded.file_id, uploaded
selection = sdk.SemanticModelSourceSelectionInput(
    "volume_files",
    volume_id=int(volume_id),
    selected_file_ids=[file_id],
)
global_preview = knowledge.preview_source_selection_counts([selection])
assert global_preview.file_count == 1 and global_preview.table_count == 0 and global_preview.total_count == 1, global_preview
model_preview = knowledge.preview_model_source_selection_counts(model_id, [selection])
assert model_preview.file_count == 1 and model_preview.table_count == 0 and model_preview.total_count == 1, model_preview
backfill = knowledge.backfill_legacy_sources(model_id)
assert backfill.updated, backfill
validated = knowledge.validate(model_id)
assert validated.valid and not validated.errors, validated
`
	out, err := env.RunPythonProductSDK(ctx, script, workspaceID, modelID, strconv.Itoa(volumeID), fileID)
	if err != nil && !errors.Is(err, framework.ErrPythonE2EDisabled) {
		t.Fatalf("run Python knowledge selection lifecycle: %v\n%s", err, string(out))
	}
}
