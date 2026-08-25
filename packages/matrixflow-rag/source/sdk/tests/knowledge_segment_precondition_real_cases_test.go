package tests

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	sdk "github.com/matrixorigin/matrixflow/sdk/go-sdk"
	"github.com/matrixorigin/matrixflow/sdk/tests/framework"
)

// TestProductSDKKnowledgeSegmentPreconditionRealCases verifies that segment
// mutations reach the Product backend with a persisted local-file source and
// consistently report the missing materialization precondition. The isolated
// test stack does not seed a vector binding or a committed segment version.
func TestProductSDKKnowledgeSegmentPreconditionRealCases(t *testing.T) {

	framework.RunProductSDKTests(t, func(env *framework.TestEnv) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()

		fixture := env.NewKnowledgeSourceFixture(ctx, t, "knowledge-segment-preconditions")
		knowledge := fixture.Model.Service()
		document, err := knowledge.GetSourceDocument(ctx, fixture.Model.ID(), fixture.Source.GetRowId())
		if err != nil || document.GetCurrentSegmentVersionId() != "" || len(document.GetSegments()) != 0 {
			t.Fatalf("read unmaterialized knowledge source document: result=%#v err=%v", document, err)
		}

		goCodes := runGoKnowledgeSegmentPreconditions(t, ctx, knowledge, fixture.Model.ID(), fixture.Source.GetRowId(), env.TestID)
		pythonCodes := runPythonKnowledgeSegmentPreconditions(t, ctx, env, fixture.Catalog.Workspace.ID(), fixture.Model.ID(), fixture.Source.GetRowId(), env.TestID)
		if framework.PythonE2EEnabled() && !reflect.DeepEqual(goCodes, pythonCodes) {
			t.Fatalf("knowledge segment precondition codes differ between SDKs: Go=%#v Python=%#v", goCodes, pythonCodes)
		}
		for operation, code := range goCodes {
			// Unmaterialized sources may surface as param invalid or not-found
			// depending on whether the segment id is checked before vector binding.
			if code != "ErrParamInvalid" && code != "ErrNotFound" {
				t.Fatalf("%s precondition code = %q, want ErrParamInvalid or ErrNotFound", operation, code)
			}
		}
	})
}

func runGoKnowledgeSegmentPreconditions(t *testing.T, ctx context.Context, knowledge *sdk.KnowledgeService, modelID, sourceID, testID string) map[string]string {
	t.Helper()
	base := sdk.SemanticModelSegmentBase{}
	content := "Product SDK segment mutation requires a materialized source"
	return map[string]string{
		"import_initial": requireKnowledgeSegmentProductError(t, "import initial segments without vector binding", func() error {
			_, err := knowledge.ImportInitialSegments(ctx, modelID, sourceID, base)
			return err
		}),
		"update": requireKnowledgeSegmentProductError(t, "update segment without vector binding", func() error {
			_, err := knowledge.UpdateSegment(ctx, modelID, sourceID, testID+"-missing-segment", sdk.SemanticModelSegmentUpdateInput{SemanticModelSegmentBase: base, Content: &content})
			return err
		}),
		"create": requireKnowledgeSegmentProductError(t, "create segment without vector binding", func() error {
			_, err := knowledge.CreateSegment(ctx, modelID, sourceID, sdk.SemanticModelSegmentCreateInput{SemanticModelSegmentBase: base, Level: "chunk", Content: &content})
			return err
		}),
		"set_enabled": requireKnowledgeSegmentProductError(t, "set segment enabled without vector binding", func() error {
			_, err := knowledge.UpdateSegmentEnabled(ctx, modelID, sourceID, testID+"-missing-segment", base, false)
			return err
		}),
		"delete": requireKnowledgeSegmentProductError(t, "delete segment without vector binding", func() error {
			_, err := knowledge.DeleteSegment(ctx, modelID, sourceID, testID+"-missing-segment", base)
			return err
		}),
		"reembed": requireKnowledgeSegmentProductError(t, "re-embed segments without vector binding", func() error {
			_, err := knowledge.ReembedSegments(ctx, modelID, sourceID, base)
			return err
		}),
		"set_current_version": requireKnowledgeSegmentProductError(t, "select missing segment version without vector binding", func() error {
			_, err := knowledge.SetCurrentSegmentVersion(ctx, modelID, sourceID, testID+"-missing-segment-version", base)
			return err
		}),
	}
}

func runPythonKnowledgeSegmentPreconditions(t *testing.T, ctx context.Context, env *framework.TestEnv, workspaceID, modelID, sourceID, testID string) map[string]string {
	t.Helper()
	script := `
import json
import sys
import moi_product_sdk as sdk

endpoint, personal_access_token, workspace_id, model_id, source_id, test_id = sys.argv[1:]
client = sdk.new_with_personal_access_token(endpoint, personal_access_token)
knowledge = client.knowledge(workspace_id)
document = knowledge.get_source_document(model_id, source_id)
assert not document.current_segment_version_id and not document.segments, document
base = sdk.SemanticModelSegmentBase("", 0)
content = "Product SDK segment mutation requires a materialized source"

def code_for(operation):
    try:
        operation()
    except sdk.Error as err:
        assert err.code
        return err.code
    raise AssertionError("knowledge segment mutation unexpectedly succeeded")

codes = {
    "import_initial": code_for(lambda: knowledge.import_initial_segments(model_id, source_id, base)),
    "update": code_for(lambda: knowledge.update_segment(model_id, source_id, test_id + "-missing-segment", sdk.SemanticModelSegmentUpdateInput(base, content=content))),
    "create": code_for(lambda: knowledge.create_segment(model_id, source_id, sdk.SemanticModelSegmentCreateInput(base, level="chunk", content=content))),
    "set_enabled": code_for(lambda: knowledge.update_segment_enabled(model_id, source_id, test_id + "-missing-segment", base, False)),
    "delete": code_for(lambda: knowledge.delete_segment(model_id, source_id, test_id + "-missing-segment", base)),
    "reembed": code_for(lambda: knowledge.reembed_segments(model_id, source_id, base)),
    "set_current_version": code_for(lambda: knowledge.set_current_segment_version(model_id, source_id, test_id + "-missing-segment-version", base)),
}
print(json.dumps(codes, sort_keys=True))
`
	out, err := env.RunPythonProductSDK(ctx, script, workspaceID, modelID, sourceID, testID)
	if err != nil {
		if errors.Is(err, framework.ErrPythonE2EDisabled) {
			return nil
		}
		t.Fatalf("run Python knowledge segment preconditions: %v\n%s", err, string(out))
	}
	var codes map[string]string
	if err := json.Unmarshal(out, &codes); err != nil {
		t.Fatalf("decode Python knowledge segment precondition codes %q: %v", out, err)
	}
	if len(codes) != 7 {
		t.Fatalf("Python knowledge segment precondition codes = %#v, want seven operations", codes)
	}
	return codes
}

func requireKnowledgeSegmentProductError(t *testing.T, operation string, run func() error) string {
	t.Helper()
	err := run()
	if err == nil {
		t.Fatalf("%s unexpectedly succeeded", operation)
	}
	var productErr *sdk.Error
	if !errors.As(err, &productErr) || productErr.Code == "" {
		t.Fatalf("%s error = %v, want Product SDK error with code", operation, err)
	}
	return productErr.Code
}
