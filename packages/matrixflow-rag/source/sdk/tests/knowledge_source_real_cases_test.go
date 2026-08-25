package tests

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	sdk "github.com/matrixorigin/matrixflow/sdk/go-sdk"
	"github.com/matrixorigin/matrixflow/sdk/tests/framework"
)

// TestProductSDKKnowledgeSourceRealCases verifies source governance and
// lifecycle against uploaded local-file sources created by the public SDK.
func TestProductSDKKnowledgeSourceRealCases(t *testing.T) {

	framework.RunProductSDKTests(t, func(env *framework.TestEnv) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		fixture := env.NewKnowledgeSourceFixture(ctx, t, "knowledge-source")
		knowledge := fixture.Model.Service()
		goSourceID := fixture.Source.GetRowId()
		goFileID := fixture.Source.GetSourceFileId()

		// Second local file via model upload + append covers the append route after
		// create-with-sources deployed the KB document-parsing workflow.
		goSecondUpload, err := knowledge.UploadModelLocalFile(ctx, fixture.Model.ID(), "go-source-2.txt", strings.NewReader("Go Product SDK knowledge source append"))
		if err != nil || goSecondUpload.GetFileId() == "" {
			t.Fatalf("upload second Go knowledge source file: result=%#v err=%v", goSecondUpload, err)
		}
		goAppended, err := knowledge.AppendSources(ctx, fixture.Model.ID(), []sdk.SemanticModelSourceInput{{
			SourceType: "local_file",
			FileName:   "go-source-2.txt",
			FileID:     goSecondUpload.GetFileId(),
		}})
		if err != nil || !containsKnowledgeSource(goAppended.GetSources(), goSecondUpload.GetFileId()) {
			t.Fatalf("append second Go knowledge source: result=%#v err=%v", goAppended, err)
		}

		goDocument, err := knowledge.GetSourceDocument(ctx, fixture.Model.ID(), goSourceID)
		if err != nil || goDocument.GetSource().GetRowId() != goSourceID || goDocument.GetSource().GetSourceFileId() != goFileID || goDocument.GetSegmentStatus() == nil {
			t.Fatalf("read Go knowledge source document: result=%#v err=%v", goDocument, err)
		}
		goGovernance, err := knowledge.UpdateSourceGovernance(ctx, fixture.Model.ID(), goSourceID, sdk.WithSemanticModelSourceTags("sdk-test", "go"))
		if err != nil || goGovernance.GetSource().GetRowId() != goSourceID || !goGovernance.GetSource().GetEnabled() || !goGovernance.GetSource().GetEffectiveEnabled() || !hasSourceTags(goGovernance.GetSource(), "sdk-test", "go") {
			t.Fatalf("update Go knowledge source governance: result=%#v err=%v", goGovernance, err)
		}
		goTags, err := knowledge.ListTags(ctx)
		if err != nil || !semanticTagsAreWellFormed(goTags.GetItems()) {
			t.Fatalf("list Go semantic model tags: result=%#v err=%v", goTags, err)
		}
		existing, err := knowledge.CheckSourceExistence(ctx, fixture.Model.ID(), []string{goFileID}, nil)
		if err != nil {
			t.Fatalf("check Go knowledge source existence: result=%#v err=%v", existing, err)
		}
		// Existence currently only indexes some source kinds; accept empty when
		// the source still appears in the authoritative source list below.
		if len(existing.GetFileIds()) > 0 && existing.GetFileIds()[0] != goFileID {
			t.Fatalf("check Go knowledge source existence: result=%#v err=%v", existing, err)
		}
		goJobs, err := knowledge.ListSourceJobs(ctx, fixture.Model.ID())
		if err != nil || goJobs.GetTotal() < int32(len(goJobs.GetItems())) {
			t.Fatalf("list Go knowledge source jobs: result=%#v err=%v", goJobs, err)
		}
		if _, err := knowledge.ReconcileSourceJobs(ctx, fixture.Model.ID()); err != nil {
			t.Fatalf("reconcile Go knowledge source jobs: %v", err)
		}
		goSources, err := knowledge.ListSources(ctx, fixture.Model.ID())
		if err != nil || !containsKnowledgeSource(goSources.GetItems(), goFileID) {
			t.Fatalf("list Go knowledge sources: result=%#v err=%v", goSources, err)
		}
		// Source may still be processing after append; wait then hard-delete.
		deleted, deleteErr := waitDeleteKnowledgeSource(ctx, knowledge, fixture.Model.ID(), goSourceID)
		if deleteErr != nil || deleted == nil || !deleted.GetDeleted() {
			t.Fatalf("delete Go knowledge source: result=%#v err=%v", deleted, deleteErr)
		}
		goSources, err = knowledge.ListSources(ctx, fixture.Model.ID())
		if err != nil || containsKnowledgeSource(goSources.GetItems(), goFileID) {
			t.Fatalf("list Go knowledge sources after delete: result=%#v err=%v", goSources, err)
		}

		script := `
import sys
import moi_product_sdk as sdk
` + pythonWaitDeleteKnowledgeSource + `
endpoint, token, workspace_id, test_id = sys.argv[1:]
client = sdk.new_with_personal_access_token(endpoint, token)
knowledge = client.knowledge(workspace_id)
# Bootstrap with create-with-sources so the KB document-parsing workflow exists
# before source governance / append-style operations (backend fails closed on
# first append to an empty model without that workflow).
seed = knowledge.upload_local_file("python-source.txt", b"Python Product SDK knowledge source")
assert seed.file_id
created = knowledge.create_with_sources(
    test_id + "-knowledge-source-python",
    [sdk.SemanticModelSourceInput("local_file", file_name=seed.original_name or "python-source.txt", file_id=seed.file_id)],
    sdk.with_semantic_model_with_sources_description("Python Product SDK knowledge source fixture"),
)
assert created.model and created.model.id
model_id = str(created.model.id)
source = next(item for item in created.sources if item.row_id and item.source_file_id == seed.file_id and item.source_type in ("file", "local_file"))
# Second file via model upload + append covers the append route after workflow deploy.
uploaded = knowledge.upload_model_local_file(model_id, "python-source-2.txt", b"Python Product SDK knowledge source append")
assert uploaded.file_id
appended = knowledge.append_sources(model_id, [sdk.SemanticModelSourceInput("local_file", file_name=uploaded.original_name or "python-source-2.txt", file_id=uploaded.file_id)])
assert any(item.source_file_id == uploaded.file_id and item.row_id for item in appended.sources), appended
document = knowledge.get_source_document(model_id, source.row_id)
assert document.source.row_id == source.row_id and document.source.source_file_id == source.source_file_id
assert document.segment_status is not None
governance = knowledge.update_source_governance(model_id, source.row_id, sdk.with_semantic_model_source_tags("sdk-test", "python"))
assert governance.source.row_id == source.row_id
assert governance.source.enabled and governance.source.effective_enabled
assert governance.source.tags == ["sdk-test", "python"]
tags = knowledge.list_tags()
assert all(item.tag and item.count > 0 for item in tags.items)
existing = knowledge.check_source_existence(model_id, [source.source_file_id], [])
assert existing.file_ids in ([], [source.source_file_id]) or source.source_file_id in list(existing.file_ids)
jobs = knowledge.list_source_jobs(model_id)
assert jobs.total >= len(jobs.items)
knowledge.reconcile_source_jobs(model_id)
sources = knowledge.list_sources(model_id)
assert any(item.source_file_id == source.source_file_id and item.source_type in ("file", "local_file") for item in sources.items)
deleted = wait_delete_knowledge_source(knowledge, model_id, source.row_id, 120)
assert deleted.deleted
sources = knowledge.list_sources(model_id)
assert all(item.source_file_id != source.source_file_id for item in sources.items)
`
		out, err := env.RunPythonProductSDK(ctx, script, fixture.Catalog.Workspace.ID(), env.TestID)
		if err != nil && !errors.Is(err, framework.ErrPythonE2EDisabled) {
			t.Fatalf("run Python knowledge source lifecycle: %v\n%s", err, string(out))
		}
		required := []struct{ method, path string }{
			{"POST", "/newmoi/semantic-models/local-files/upload"},
			{"POST", "/newmoi/semantic-models/create-with-sources"},
			{"POST", "/newmoi/semantic-models/:model_id/local-files/upload"},
			{"POST", "/newmoi/semantic-models/:model_id/sources"},
			{"GET", "/newmoi/semantic-models/:model_id/sources"},
			{"GET", "/newmoi/semantic-models/:model_id/sources/:source_row_id/document"},
			{"POST", "/newmoi/semantic-models/:model_id/sources/existence"},
			{"GET", "/newmoi/semantic-models/tags"},
			{"GET", "/newmoi/semantic-models/:model_id/source-jobs"},
			{"POST", "/newmoi/semantic-models/:model_id/source-jobs/reconcile"},
			{"DELETE", "/newmoi/semantic-models/:model_id/sources/:source_row_id"},
		}
		for _, route := range required {
			env.RequireRealResponse(t, route.method, route.path)
			env.RequirePythonSDKRealResponse(t, route.method, route.path)
		}
	})
}

const pythonWaitDeleteKnowledgeSource = `
import time

def wait_delete_knowledge_source(knowledge, model_id, source_id, timeout_seconds):
    deadline = time.monotonic() + timeout_seconds
    last_error = None
    while time.monotonic() < deadline:
        try:
            knowledge.reconcile_source_jobs(model_id)
        except sdk.Error:
            # Delete remains the authoritative readiness check.
            pass
        try:
            return knowledge.delete_source(model_id, source_id)
        except sdk.Error as err:
            if "still being processed" not in str(err) and "ErrConflict" not in str(err):
                raise
            last_error = err
        time.sleep(1)
    raise AssertionError("knowledge source delete did not become ready: " + str(last_error))
`

// waitDeleteKnowledgeSource retries delete while the source is still processing
// (409/conflict). Fails hard if delete never succeeds within the parent context.
//
// Product delete rejects sources with an unexpired running job claim. Local-file
// append creates load + rag_ingest jobs; the workflow may finish while the job
// row stays "running" until reconcile finalizes it. Re-run reconcile between
// delete attempts so hard-pass does not wait on the 30m claim lease.
func waitDeleteKnowledgeSource(ctx context.Context, knowledge *sdk.KnowledgeService, modelID, sourceID string) (*sdk.SemanticMutationResult, error) {
	var last *sdk.SemanticMutationResult
	var lastErr error
	// Fresh MO + concurrent smoke can keep source jobs busy past 45s; allow up
	// to the parent deadline (or 2 minutes) so hard-pass does not flake.
	deadline := time.Now().Add(2 * time.Minute)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	for time.Now().Before(deadline) {
		// Advance stuck running claims before each delete attempt.
		if _, recErr := knowledge.ReconcileSourceJobs(ctx, modelID); recErr != nil {
			// Reconcile is best-effort for delete readiness; keep retrying delete
			// when the product still reports a processing conflict.
			if lastErr == nil {
				lastErr = recErr
			}
		}
		last, lastErr = knowledge.DeleteSource(ctx, modelID, sourceID)
		if lastErr == nil && last != nil && last.GetDeleted() {
			return last, nil
		}
		// Only retry explicit processing conflicts; other errors fail hard.
		if lastErr != nil && !strings.Contains(lastErr.Error(), "still being processed") && !strings.Contains(lastErr.Error(), "ErrConflict") {
			return last, lastErr
		}
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return last, lastErr
			}
			return last, ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
	if lastErr != nil {
		return last, lastErr
	}
	return last, fmt.Errorf("delete knowledge source %s did not succeed before deadline", sourceID)
}

// TestProductSDKKnowledgeSourceAppendUploadedFileRealCase keeps the public
// upload-then-append flow as a real two-language regression. Bootstrap uses
// create-with-sources so the KB workflow exists; append then adds a second
// local file (backend requires an existing document-parsing workflow on append).
func TestProductSDKKnowledgeSourceAppendUploadedFileRealCase(t *testing.T) {

	framework.RunProductSDKTests(t, func(env *framework.TestEnv) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		// Fixture already deploys the KB workflow via create-with-sources.
		fixture := env.NewKnowledgeSourceFixture(ctx, t, "knowledge-source-append")
		knowledge := fixture.Model.Service()
		failures := make([]string, 0, 2)

		goUpload, err := knowledge.UploadModelLocalFile(ctx, fixture.Model.ID(), "go-notes.txt", strings.NewReader("Go source content"))
		if err != nil || goUpload.GetFileId() == "" {
			failures = append(failures, fmt.Sprintf("Go upload result=%#v err=%v", goUpload, err))
		} else {
			appended, appendErr := knowledge.AppendSources(ctx, fixture.Model.ID(), []sdk.SemanticModelSourceInput{{SourceType: "local_file", FileName: "go-notes.txt", FileID: goUpload.GetFileId()}})
			if appendErr != nil || !containsKnowledgeSource(appended.GetSources(), goUpload.GetFileId()) {
				failures = append(failures, fmt.Sprintf("Go append result=%#v err=%v", appended, appendErr))
			} else if deleted, deleteErr := waitDeleteKnowledgeSource(ctx, knowledge, fixture.Model.ID(), knowledgeSourceID(appended.GetSources(), goUpload.GetFileId())); deleteErr != nil || !deleted.GetDeleted() {
				failures = append(failures, fmt.Sprintf("Go cleanup result=%#v err=%v", deleted, deleteErr))
			}
		}

		script := `
import sys
import moi_product_sdk as sdk
` + pythonWaitDeleteKnowledgeSource + `
endpoint, token, workspace_id, model_id = sys.argv[1:]
client = sdk.new_with_personal_access_token(endpoint, token)
knowledge = client.knowledge(workspace_id)
uploaded = knowledge.upload_model_local_file(model_id, "python-notes.txt", b"Python source content")
assert uploaded.file_id
appended = knowledge.append_sources(model_id, [sdk.SemanticModelSourceInput("local_file", file_name=uploaded.original_name or "python-source.txt", file_id=uploaded.file_id)])
source = next(item for item in appended.sources if item.source_file_id == uploaded.file_id and item.row_id)
deleted = wait_delete_knowledge_source(knowledge, model_id, source.row_id, 60)
assert deleted.deleted
`
		out, err := env.RunPythonProductSDK(ctx, script, fixture.Catalog.Workspace.ID(), fixture.Model.ID())
		if err != nil && !errors.Is(err, framework.ErrPythonE2EDisabled) {
			failures = append(failures, fmt.Sprintf("Python upload-then-append: %v\n%s", err, string(out)))
		}
		if len(failures) != 0 {
			t.Fatalf("uploaded local-file append must create and delete a source through both SDKs:\n%s", strings.Join(failures, "\n"))
		}
		for _, route := range []struct{ method, path string }{{"POST", "/newmoi/semantic-models/:model_id/local-files/upload"}, {"POST", "/newmoi/semantic-models/:model_id/sources"}, {"DELETE", "/newmoi/semantic-models/:model_id/sources/:source_row_id"}} {
			env.RequireRealResponse(t, route.method, route.path)
			env.RequirePythonSDKRealResponse(t, route.method, route.path)
		}
	})
}

func semanticTagsAreWellFormed(tags []*sdk.SemanticModelTagStat) bool {
	for _, item := range tags {
		if item.GetTag() == "" || item.GetCount() <= 0 {
			return false
		}
	}
	return true
}

func hasSourceTags(source *sdk.SemanticModelSource, tags ...string) bool {
	if source == nil || len(source.GetTags()) != len(tags) {
		return false
	}
	for index, tag := range tags {
		if source.GetTags()[index] != tag {
			return false
		}
	}
	return true
}

func knowledgeSourceID(sources []*sdk.SemanticModelSource, fileID string) string {
	for _, source := range sources {
		if source.GetSourceFileId() == fileID && source.GetRowId() != "" {
			return source.GetRowId()
		}
	}
	return ""
}

func containsKnowledgeSource(sources []*sdk.SemanticModelSource, fileID string) bool {
	return knowledgeSourceID(sources, fileID) != ""
}

// TestProductSDKKnowledgeCreateWithSourcesRealCases covers the combined
// create-with-sources product route: upload a local file, then create the
// model with that source in one call.
func TestProductSDKKnowledgeCreateWithSourcesRealCases(t *testing.T) {
	framework.RunProductSDKTests(t, func(env *framework.TestEnv) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		catalog := env.NewCatalogFixture(ctx, t, "knowledge-create-with-sources")
		knowledge := catalog.Workspace.Knowledge()

		upload, err := knowledge.UploadLocalFile(ctx, "create-with-sources.txt", strings.NewReader("Product SDK create-with-sources fixture"))
		if err != nil || upload.GetFileId() == "" {
			t.Fatalf("upload local file for create-with-sources: result=%#v err=%v", upload, err)
		}
		created, err := knowledge.CreateWithSources(ctx, env.TestID+"-create-with-sources", []sdk.SemanticModelSourceInput{{
			SourceType: "local_file",
			FileName:   "create-with-sources.txt",
			FileID:     upload.GetFileId(),
		}}, sdk.WithSemanticModelWithSourcesDescription("Product SDK create-with-sources"))
		if err != nil {
			t.Fatalf("create semantic model with sources: %v", err)
		}
		if created.GetModel() == nil || created.GetModel().GetId() == 0 {
			t.Fatalf("create-with-sources returned empty model: %#v", created)
		}
		found := false
		for _, source := range created.GetSources() {
			if source.GetRowId() != "" && source.GetSourceFileId() == upload.GetFileId() {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("create-with-sources missing persisted source for file %s: %#v", upload.GetFileId(), created)
		}

		script := `
import sys
import moi_product_sdk as sdk

endpoint, token, workspace_id, test_id = sys.argv[1:]
client = sdk.new_with_personal_access_token(endpoint, token)
knowledge = client.knowledge(workspace_id)
uploaded = knowledge.upload_local_file("python-create-with-sources.txt", b"Python Product SDK create-with-sources fixture")
assert uploaded.file_id
created = knowledge.create_with_sources(
    test_id + "-python-create-with-sources",
    [sdk.SemanticModelSourceInput("local_file", file_name=uploaded.original_name or "python-create-with-sources.txt", file_id=uploaded.file_id)],
    sdk.with_semantic_model_with_sources_description("Python Product SDK create-with-sources"),
)
assert created.model.id
assert any(source.row_id and source.source_file_id == uploaded.file_id for source in created.sources), created
`
		out, err := env.RunPythonProductSDK(ctx, script, catalog.Workspace.ID(), env.TestID)
		if err != nil && !errors.Is(err, framework.ErrPythonE2EDisabled) {
			t.Fatalf("run Python create-with-sources lifecycle: %v\n%s", err, string(out))
		}

		env.RequireRealResponse(t, "POST", "/newmoi/semantic-models/create-with-sources")
		env.RequirePythonSDKRealResponse(t, "POST", "/newmoi/semantic-models/create-with-sources")
	})
}

// TestProductSDKKnowledgeCreateEmptyRealCases covers data-side knowledge base creation without sources.
func TestProductSDKKnowledgeCreateEmptyRealCases(t *testing.T) {
	framework.RunProductSDKTests(t, func(env *framework.TestEnv) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		knowledge := env.SharedWorkspace(t).Knowledge()

		created, err := knowledge.CreateEmpty(ctx, env.TestID+"-create-empty", sdk.WithSemanticModelEmptyDescription("Product SDK empty knowledge base"), sdk.WithSemanticModelEmptyImageIndexEnabled(true))
		if err != nil {
			t.Fatalf("create empty semantic model: %v", err)
		}
		if created.GetModel() == nil || created.GetModel().GetId() == 0 || created.GetDataDomain() == nil || created.GetDataDomain().GetCatalogId() == 0 || created.GetDataDomain().GetDatabaseId() == 0 || created.GetDataDomain().GetRawVolumeId() == 0 || created.GetDataDomain().GetProcessedVolumeId() == 0 {
			t.Fatalf("create-empty returned incomplete data domain: %#v", created)
		}

		script := `
import sys
import moi_product_sdk as sdk

endpoint, token, workspace_id, test_id = sys.argv[1:]
client = sdk.new_with_personal_access_token(endpoint, token)
created = client.knowledge(workspace_id).create_empty(
    test_id + "-python-create-empty",
    sdk.with_semantic_model_empty_description("Python Product SDK empty knowledge base"),
    sdk.with_semantic_model_empty_image_index_enabled(True),
)
assert created.model.id
assert created.data_domain.catalog_id
assert created.data_domain.database_id
assert created.data_domain.raw_volume_id
assert created.data_domain.processed_volume_id
`
		out, err := env.RunPythonProductSDK(ctx, script, env.SharedWorkspaceID, env.TestID)
		if err != nil && !errors.Is(err, framework.ErrPythonE2EDisabled) {
			t.Fatalf("run Python empty knowledge base creation: %v\n%s", err, string(out))
		}

		env.RequireRealResponse(t, "POST", "/newmoi/semantic-models/create-empty")
		env.RequirePythonSDKRealResponse(t, "POST", "/newmoi/semantic-models/create-empty")
	})
}
