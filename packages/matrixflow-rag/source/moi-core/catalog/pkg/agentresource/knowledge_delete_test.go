package agentresource

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/tenant"
)

func TestUpdateVersionReadinessReplacesDiagnosticsAndClearsRecovered(t *testing.T) {
	versionStore := NewInMemoryAgentVersionStore()
	service := NewAgentVersionService(versionStore)
	ctx := context.Background()
	now := time.Unix(100, 0)

	if _, err := versionStore.CreateAgentVersion(ctx, AgentVersionRecord{
		WorkspaceID: "ws_1",
		AgentID:     "agent_1",
		Version:     "1.0.0",
		Status:      AgentVersionStatusNeedsConfiguration,
		Diagnostics: []AgentPackageDiagnostic{{
			Severity: "error",
			Code:     "custom_tool_worker_install_failed",
			Message:  "install failed",
			Ref:      "cap_1",
		}},
		LoadedAt: now,
	}); err != nil {
		t.Fatalf("CreateAgentVersion: %v", err)
	}

	// Full replace with nil must clear the recovered non-KB condition and restore runnable.
	updated, err := service.UpdateVersionReadiness(ctx, "ws_1", "agent_1", "1.0.0", nil)
	if err != nil {
		t.Fatalf("UpdateVersionReadiness(nil): %v", err)
	}
	if updated.Status != AgentVersionStatusRunnable {
		t.Fatalf("status = %s, want runnable", updated.Status)
	}
	if len(updated.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %+v, want empty after clear", updated.Diagnostics)
	}

	// DELETE-owned ID-based diagnostics must survive a full replace.
	if _, err := versionStore.UpdateAgentVersionStatusAndDiagnostics(ctx, "ws_1", "agent_1", "1.0.0", AgentVersionStatusNeedsConfiguration, []AgentPackageDiagnostic{
		knowledgeBaseDeletedDiagnostic("ws_1", "20001"),
		{
			Severity: "error",
			Code:     "custom_tool_worker_install_failed",
			Message:  "install failed again",
			Ref:      "cap_1",
		},
	}); err != nil {
		t.Fatalf("seed mixed diagnostics: %v", err)
	}
	updated, err = service.UpdateVersionReadiness(ctx, "ws_1", "agent_1", "1.0.0", nil)
	if err != nil {
		t.Fatalf("UpdateVersionReadiness(nil) with delete-owned: %v", err)
	}
	if updated.Status != AgentVersionStatusNeedsConfiguration {
		t.Fatalf("status = %s, want needs_configuration while delete-owned remains", updated.Status)
	}
	if len(updated.Diagnostics) != 1 || updated.Diagnostics[0].Code != AgentPackageDiagnosticCodeKnowledgeBaseDeleted || updated.Diagnostics[0].Ref != "ws_1/20001" {
		t.Fatalf("diagnostics = %+v, want only ID-based knowledge_base_deleted", updated.Diagnostics)
	}
}

func TestCommitVersionReadinessPreservesKnowledgeBaseDeleted(t *testing.T) {
	versionStore := NewInMemoryAgentVersionStore()
	service := NewAgentVersionService(versionStore)
	ctx := context.Background()
	now := time.Unix(100, 0)

	if _, err := versionStore.CreateAgentVersion(ctx, AgentVersionRecord{
		WorkspaceID: "ws_1",
		AgentID:     "agent_1",
		Version:     "1.0.0",
		Status:      AgentVersionStatusNeedsConfiguration,
		Diagnostics: []AgentPackageDiagnostic{
			knowledgeBaseDeletedDiagnostic("ws_1", "20001"),
			{
				Severity: "error",
				Code:     "custom_tool_worker_install_failed",
				Message:  "install failed",
				Ref:      "cap_1",
			},
		},
		Manifest: AgentPackageManifest{
			MOIResourceRefs: AgentPackageMOIResourceRefs{
				KnowledgeBases: []AgentPackageNamedResourceRef{{Name: "kb_sales"}},
			},
		},
		LoadedAt: now,
	}); err != nil {
		t.Fatalf("CreateAgentVersion: %v", err)
	}

	updated, err := service.CommitVersionReadiness(ctx, "ws_1", "agent_1", "1.0.0", nonRunnerInstallDiagnostics, nil)
	if err != nil {
		t.Fatalf("CommitVersionReadiness: %v", err)
	}
	if updated.Status != AgentVersionStatusNeedsConfiguration {
		t.Fatalf("status = %s, want needs_configuration", updated.Status)
	}
	foundDeleted := false
	for _, diagnostic := range updated.Diagnostics {
		if diagnostic.Code == AgentPackageDiagnosticCodeKnowledgeBaseDeleted {
			foundDeleted = true
		}
		if diagnostic.Code == "custom_tool_worker_install_failed" {
			t.Fatalf("runner owned diagnostic should be replaced/removed: %+v", updated.Diagnostics)
		}
	}
	if !foundDeleted {
		t.Fatalf("knowledge_base_deleted was lost: %+v", updated.Diagnostics)
	}
}

func TestDeleteVsReadinessCommitOrders(t *testing.T) {
	run := func(t *testing.T, deleteFirst bool) {
		t.Helper()
		store := NewInMemoryAgentStore()
		versionStore := NewInMemoryAgentVersionStore()
		agentService := NewAgentService(store).WithAgentVersionService(NewAgentVersionService(versionStore))
		versionService := NewAgentVersionService(versionStore)
		ctx := context.Background()
		now := time.Unix(100, 0)

		if _, err := versionStore.CreateAgentVersion(ctx, AgentVersionRecord{
			WorkspaceID: "ws_1",
			AgentID:     "agent_1",
			Version:     "1.0.0",
			Status:      AgentVersionStatusRunnable,
			Manifest: AgentPackageManifest{
				MOIResourceRefs: AgentPackageMOIResourceRefs{
					KnowledgeBases: []AgentPackageNamedResourceRef{{Name: "kb_sales"}},
				},
			},
			LoadedAt: now,
		}); err != nil {
			t.Fatalf("CreateAgentVersion: %v", err)
		}

		deleteFn := func() {
			if _, err := agentService.HandleSemanticKnowledgeBaseDeleted(ctx, "ws_1", 20001, "kb_sales", "user_1"); err != nil {
				t.Fatalf("delete: %v", err)
			}
		}
		readyFn := func() {
			if _, err := versionService.CommitVersionReadiness(ctx, "ws_1", "agent_1", "1.0.0", nil, nil); err != nil {
				t.Fatalf("readiness: %v", err)
			}
		}

		if deleteFirst {
			deleteFn()
			readyFn()
		} else {
			readyFn()
			deleteFn()
		}

		got, err := versionStore.GetAgentVersion(ctx, "ws_1", "agent_1", "1.0.0")
		if err != nil {
			t.Fatalf("GetAgentVersion: %v", err)
		}
		if got.Status != AgentVersionStatusNeedsConfiguration {
			t.Fatalf("status = %s, want needs_configuration (deleteFirst=%v)", got.Status, deleteFirst)
		}
		found := false
		for _, diagnostic := range got.Diagnostics {
			if diagnostic.Code == AgentPackageDiagnosticCodeKnowledgeBaseDeleted {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing knowledge_base_deleted (deleteFirst=%v): %+v", deleteFirst, got.Diagnostics)
		}
	}

	t.Run("delete_then_readiness", func(t *testing.T) { run(t, true) })
	t.Run("readiness_then_delete", func(t *testing.T) { run(t, false) })
}

type recordingVersionDeleteStore struct {
	*InMemoryAgentVersionStore
	locked []string
}

func (s *recordingVersionDeleteStore) GetAgentVersionForUpdate(ctx context.Context, workspaceID, agentID, version string) (*AgentVersionRecord, error) {
	s.locked = append(s.locked, agentID+"/"+version)
	return s.InMemoryAgentVersionStore.GetAgentVersionForUpdate(ctx, workspaceID, agentID, version)
}

func (s *recordingVersionDeleteStore) ApplyAgentVersionStatusAndDiagnostics(
	ctx context.Context,
	workspaceID, agentID, version string,
	apply func(current AgentVersionRecord) (status string, diagnostics []AgentPackageDiagnostic, write bool, err error),
) (*AgentVersionRecord, error) {
	s.locked = append(s.locked, agentID+"/"+version)
	return s.InMemoryAgentVersionStore.ApplyAgentVersionStatusAndDiagnostics(ctx, workspaceID, agentID, version, apply)
}

func TestDeleteLocksOnlyMatchedPackageVersions(t *testing.T) {
	store := NewInMemoryAgentStore()
	versionStore := &recordingVersionDeleteStore{InMemoryAgentVersionStore: NewInMemoryAgentVersionStore()}
	service := NewAgentService(store).WithAgentVersionService(NewAgentVersionService(versionStore))
	ctx := context.Background()
	now := time.Unix(100, 0)

	// matched version
	if _, err := versionStore.CreateAgentVersion(ctx, AgentVersionRecord{
		WorkspaceID: "ws_1", AgentID: "agent_match", Version: "1.0.0", Status: AgentVersionStatusRunnable,
		Manifest: AgentPackageManifest{MOIResourceRefs: AgentPackageMOIResourceRefs{KnowledgeBases: []AgentPackageNamedResourceRef{{Name: "kb_sales"}}}},
		LoadedAt: now,
	}); err != nil {
		t.Fatalf("CreateAgentVersion match: %v", err)
	}
	// unrelated version
	if _, err := versionStore.CreateAgentVersion(ctx, AgentVersionRecord{
		WorkspaceID: "ws_1", AgentID: "agent_other", Version: "1.0.0", Status: AgentVersionStatusRunnable,
		Manifest: AgentPackageManifest{MOIResourceRefs: AgentPackageMOIResourceRefs{KnowledgeBases: []AgentPackageNamedResourceRef{{Name: "kb_other"}}}},
		LoadedAt: now,
	}); err != nil {
		t.Fatalf("CreateAgentVersion other: %v", err)
	}

	if _, err := service.HandleSemanticKnowledgeBaseDeleted(ctx, "ws_1", 20001, "kb_sales", "user_1"); err != nil {
		t.Fatalf("HandleSemanticKnowledgeBaseDeleted: %v", err)
	}
	if len(versionStore.locked) != 1 || versionStore.locked[0] != "agent_match/1.0.0" {
		t.Fatalf("locked versions = %+v, want only agent_match/1.0.0", versionStore.locked)
	}
}

type concurrentDeleteOnLockVersionStore struct {
	*InMemoryAgentVersionStore
	atomicCalls   int
	lineageWrites int
}

func (s *concurrentDeleteOnLockVersionStore) SetDefaultAgentVersionIfRunnable(ctx context.Context, workspaceID, agentID, version string) (*AgentLineageRecord, error) {
	s.atomicCalls++
	// Concurrent DELETE wins the version critical section and flips status before SetDefault continues.
	if _, err := s.InMemoryAgentVersionStore.UpdateAgentVersionStatusAndDiagnostics(ctx, workspaceID, agentID, version, AgentVersionStatusNeedsConfiguration, []AgentPackageDiagnostic{
		knowledgeBaseDeletedDiagnostic(workspaceID, "20001"),
	}); err != nil {
		return nil, err
	}
	return s.InMemoryAgentVersionStore.SetDefaultAgentVersionIfRunnable(ctx, workspaceID, agentID, version)
}

func (s *concurrentDeleteOnLockVersionStore) SetDefaultAgentVersionIfRunnableAndCurrent(ctx context.Context, workspaceID, agentID, expectedVersion, version string) (*AgentLineageRecord, error) {
	s.atomicCalls++
	if _, err := s.InMemoryAgentVersionStore.UpdateAgentVersionStatusAndDiagnostics(ctx, workspaceID, agentID, version, AgentVersionStatusNeedsConfiguration, []AgentPackageDiagnostic{
		knowledgeBaseDeletedDiagnostic(workspaceID, "20001"),
	}); err != nil {
		return nil, err
	}
	return s.InMemoryAgentVersionStore.SetDefaultAgentVersionIfRunnableAndCurrent(ctx, workspaceID, agentID, expectedVersion, version)
}

func (s *concurrentDeleteOnLockVersionStore) SetDefaultAgentVersion(ctx context.Context, workspaceID, agentID, version string) (*AgentLineageRecord, error) {
	s.lineageWrites++
	return s.InMemoryAgentVersionStore.SetDefaultAgentVersion(ctx, workspaceID, agentID, version)
}

func (s *concurrentDeleteOnLockVersionStore) SetDefaultAgentVersionIfCurrent(ctx context.Context, workspaceID, agentID, expectedVersion, version string) (*AgentLineageRecord, error) {
	s.lineageWrites++
	return s.InMemoryAgentVersionStore.SetDefaultAgentVersionIfCurrent(ctx, workspaceID, agentID, expectedVersion, version)
}

func TestSetDefaultVersionRejectsAfterConcurrentDeleteOnLock(t *testing.T) {
	versionStore := &concurrentDeleteOnLockVersionStore{InMemoryAgentVersionStore: NewInMemoryAgentVersionStore()}
	service := NewAgentVersionService(versionStore)
	ctx := context.Background()
	now := time.Unix(100, 0)
	if _, err := versionStore.CreateAgentVersion(ctx, AgentVersionRecord{
		WorkspaceID: "ws_1", AgentID: "agent_1", Version: "1.0.0", Status: AgentVersionStatusRunnable, LoadedAt: now,
	}); err != nil {
		t.Fatalf("CreateAgentVersion: %v", err)
	}

	// Atomic runnable check must observe the concurrent delete and refuse the default write.
	if _, err := service.SetDefaultVersion(ctx, "ws_1", "agent_1", "1.0.0"); err == nil {
		t.Fatalf("expected SetDefaultVersion to reject after concurrent delete on lock")
	}
	if versionStore.atomicCalls != 1 {
		t.Fatalf("SetDefaultAgentVersionIfRunnable calls = %d, want 1", versionStore.atomicCalls)
	}
	if versionStore.lineageWrites != 0 {
		t.Fatalf("lineage writes = %d, want 0", versionStore.lineageWrites)
	}

	// Same gate for compare-and-set entrypoint.
	versionStore.atomicCalls = 0
	versionStore.lineageWrites = 0
	if _, err := versionStore.UpdateAgentVersionStatusAndDiagnostics(ctx, "ws_1", "agent_1", "1.0.0", AgentVersionStatusRunnable, nil); err != nil {
		t.Fatalf("reset runnable: %v", err)
	}
	if _, err := service.SetDefaultVersionIfCurrent(ctx, "ws_1", "agent_1", "", "1.0.0"); err == nil {
		t.Fatalf("expected SetDefaultVersionIfCurrent to reject after concurrent delete on lock")
	}
	if versionStore.atomicCalls != 1 {
		t.Fatalf("IfCurrent atomic calls = %d, want 1", versionStore.atomicCalls)
	}
	if versionStore.lineageWrites != 0 {
		t.Fatalf("IfCurrent lineage writes = %d, want 0", versionStore.lineageWrites)
	}
}

type sameNameLocker struct{}

func (sameNameLocker) LockSemanticModelsForUpdate(_ context.Context, modelIDs []int64) ([]*tenant.SemanticModelRecord, error) {
	out := make([]*tenant.SemanticModelRecord, 0, len(modelIDs))
	for _, id := range modelIDs {
		out = append(out, &tenant.SemanticModelRecord{ID: id, Name: "kb_sales"})
	}
	return out, nil
}
func (sameNameLocker) GetSemanticModelForUpdate(ctx context.Context, modelID int64) (*tenant.SemanticModelRecord, error) {
	rows, err := (sameNameLocker{}).LockSemanticModelsForUpdate(ctx, []int64{modelID})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, tenant.ErrSemanticModelNotFound
	}
	return rows[0], nil
}
func (sameNameLocker) ListSemanticModels(_ context.Context, opts ...tenant.ListOption) ([]*tenant.SemanticModelRecord, int64, error) {
	// Same-name recreation: name resolves to a new model id.
	return []*tenant.SemanticModelRecord{{ID: 30001, Name: "kb_sales"}}, 1, nil
}

func TestHandleSemanticKnowledgeBaseRenamedInvalidatesOldNameVersions(t *testing.T) {
	store := NewInMemoryAgentStore()
	versionStore := NewInMemoryAgentVersionStore()
	service := NewAgentService(store).WithAgentVersionService(NewAgentVersionService(versionStore))
	ctx := context.Background()
	now := time.Unix(100, 0)

	if _, err := versionStore.CreateAgentVersion(ctx, AgentVersionRecord{
		WorkspaceID: "ws_1",
		AgentID:     "agent_pkg",
		Version:     "1.0.0",
		Status:      AgentVersionStatusRunnable,
		Manifest: AgentPackageManifest{
			MOIResourceRefs: AgentPackageMOIResourceRefs{
				KnowledgeBases: []AgentPackageNamedResourceRef{{Name: "kb_old"}},
			},
		},
		LoadedAt: now,
	}); err != nil {
		t.Fatalf("CreateAgentVersion: %v", err)
	}

	stats, err := service.HandleSemanticKnowledgeBaseRenamed(ctx, "ws_1", 20001, "kb_old", "kb_new")
	if err != nil {
		t.Fatalf("HandleSemanticKnowledgeBaseRenamed: %v", err)
	}
	if stats.NeedsConfigurationCount != 1 {
		t.Fatalf("stats = %+v", stats)
	}

	got, err := versionStore.GetAgentVersion(ctx, "ws_1", "agent_pkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetAgentVersion: %v", err)
	}
	if got.Status != AgentVersionStatusNeedsConfiguration {
		t.Fatalf("status = %s, want needs_configuration", got.Status)
	}
	found := false
	for _, diagnostic := range got.Diagnostics {
		if diagnostic.Code == AgentPackageDiagnosticCodeKnowledgeBaseDeleted && diagnostic.Ref == "ws_1/name:kb_old" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected name-based knowledge_base_deleted for kb_old, got %+v", got.Diagnostics)
	}

	// Subsequent delete by new name must not leave the version runnable either
	// (bindings clean by ID; version already invalidated by rename).
	if _, err := service.HandleSemanticKnowledgeBaseDeleted(ctx, "ws_1", 20001, "kb_new", "user_1"); err != nil {
		t.Fatalf("HandleSemanticKnowledgeBaseDeleted after rename: %v", err)
	}
	got, err = versionStore.GetAgentVersion(ctx, "ws_1", "agent_pkg", "1.0.0")
	if err != nil {
		t.Fatalf("GetAgentVersion after delete: %v", err)
	}
	if got.Status != AgentVersionStatusNeedsConfiguration {
		t.Fatalf("status after delete = %s, want needs_configuration", got.Status)
	}
}

func TestAgentPackageManifestReferencesKnowledgeBaseIgnoresWorkspaceID(t *testing.T) {
	manifest := AgentPackageManifest{
		MOIResourceRefs: AgentPackageMOIResourceRefs{
			KnowledgeBases: []AgentPackageNamedResourceRef{
				{Name: "kb_sales", WorkspaceID: "other_ws"},
			},
		},
	}
	// Load path ignores ref.WorkspaceID and resolves in target workspace; delete must match that.
	if !agentPackageManifestReferencesKnowledgeBase(manifest, "kb_sales") {
		t.Fatalf("expected foreign workspace_id ref to still match target-workspace load contract")
	}
	if agentPackageManifestReferencesKnowledgeBase(manifest, "kb_other") {
		t.Fatalf("unexpected match for unrelated name")
	}

	store := NewInMemoryAgentStore()
	versionStore := NewInMemoryAgentVersionStore()
	service := NewAgentService(store).WithAgentVersionService(NewAgentVersionService(versionStore))
	ctx := context.Background()
	now := time.Unix(100, 0)
	if _, err := versionStore.CreateAgentVersion(ctx, AgentVersionRecord{
		WorkspaceID: "ws_1",
		AgentID:     "agent_1",
		Version:     "1.0.0",
		Status:      AgentVersionStatusRunnable,
		Manifest:    manifest,
		LoadedAt:    now,
	}); err != nil {
		t.Fatalf("CreateAgentVersion: %v", err)
	}
	stats, err := service.HandleSemanticKnowledgeBaseDeleted(ctx, "ws_1", 20001, "kb_sales", "user_1")
	if err != nil {
		t.Fatalf("HandleSemanticKnowledgeBaseDeleted: %v", err)
	}
	if stats.NeedsConfigurationCount != 1 {
		t.Fatalf("stats = %+v, want needs_configuration_count=1", stats)
	}
}

func TestSetDefaultVersionAtomicWithConcurrentDelete(t *testing.T) {
	// Barrier store: SetDefault enters the critical section, then waits while delete flips
	// status under the same mutex-free window is impossible — instead delete runs between
	// two sequential atomic ops and the second rejects. The real invariant is enforced by
	// SetDefaultAgentVersionIfRunnable holding mu across check+write.
	versionStore := NewInMemoryAgentVersionStore()
	service := NewAgentVersionService(versionStore)
	ctx := context.Background()
	now := time.Unix(100, 0)
	if _, err := versionStore.CreateAgentVersion(ctx, AgentVersionRecord{
		WorkspaceID: "ws_1",
		AgentID:     "agent_1",
		Version:     "1.0.0",
		Status:      AgentVersionStatusRunnable,
		Manifest: AgentPackageManifest{
			MOIResourceRefs: AgentPackageMOIResourceRefs{
				KnowledgeBases: []AgentPackageNamedResourceRef{{Name: "kb_sales"}},
			},
		},
		LoadedAt: now,
	}); err != nil {
		t.Fatalf("CreateAgentVersion: %v", err)
	}

	start := make(chan struct{})
	setDone := make(chan error, 1)
	deleteDone := make(chan error, 1)
	agentService := NewAgentService(NewInMemoryAgentStore()).WithAgentVersionService(service)

	go func() {
		<-start
		_, err := service.SetDefaultVersion(ctx, "ws_1", "agent_1", "1.0.0")
		setDone <- err
	}()
	go func() {
		<-start
		_, err := agentService.HandleSemanticKnowledgeBaseDeleted(ctx, "ws_1", 20001, "kb_sales", "user_1")
		deleteDone <- err
	}()
	close(start)
	setErr := <-setDone
	deleteErr := <-deleteDone
	if deleteErr != nil {
		t.Fatalf("delete path: %v", deleteErr)
	}

	got, err := versionStore.GetAgentVersion(ctx, "ws_1", "agent_1", "1.0.0")
	if err != nil {
		t.Fatalf("GetAgentVersion: %v", err)
	}
	// Delete always leaves needs_configuration. SetDefault either observed runnable first
	// (and may have written default) or lost and returned ErrAgentVersionNotRunnable.
	// The forbidden state (write default after observing non-runnable) is covered by
	// TestSetDefaultVersionRejectsAfterConcurrentDeleteOnLock.
	if got.Status != AgentVersionStatusNeedsConfiguration {
		t.Fatalf("status = %s, want needs_configuration", got.Status)
	}
	if setErr != nil && !errors.Is(setErr, ErrAgentVersionNotRunnable) {
		t.Fatalf("SetDefaultVersion error = %v", setErr)
	}
}

func TestCommitVersionReadinessPreservesConcurrentDeleteDiagnostic(t *testing.T) {
	// Memory store serializes ApplyAgentVersionStatusAndDiagnostics under one mutex, so
	// concurrent readiness+delete cannot drop the delete diagnostic.
	base := NewInMemoryAgentVersionStore()
	service := NewAgentVersionService(base)
	agentService := NewAgentService(NewInMemoryAgentStore()).WithAgentVersionService(service)
	ctx := context.Background()
	now := time.Unix(100, 0)
	if _, err := base.CreateAgentVersion(ctx, AgentVersionRecord{
		WorkspaceID: "ws_1",
		AgentID:     "agent_1",
		Version:     "1.0.0",
		Status:      AgentVersionStatusRunnable,
		Manifest: AgentPackageManifest{
			MOIResourceRefs: AgentPackageMOIResourceRefs{
				KnowledgeBases: []AgentPackageNamedResourceRef{{Name: "kb_sales"}},
			},
		},
		LoadedAt: now,
	}); err != nil {
		t.Fatalf("CreateAgentVersion: %v", err)
	}

	start := make(chan struct{})
	errCh := make(chan error, 2)
	go func() {
		<-start
		_, err := service.CommitVersionReadiness(ctx, "ws_1", "agent_1", "1.0.0", nil, nil)
		errCh <- err
	}()
	go func() {
		<-start
		_, err := agentService.HandleSemanticKnowledgeBaseDeleted(ctx, "ws_1", 20001, "kb_sales", "user_1")
		errCh <- err
	}()
	close(start)
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("concurrent op: %v", err)
		}
	}
	got, err := base.GetAgentVersion(ctx, "ws_1", "agent_1", "1.0.0")
	if err != nil {
		t.Fatalf("GetAgentVersion: %v", err)
	}
	if got.Status != AgentVersionStatusNeedsConfiguration {
		t.Fatalf("status = %s, want needs_configuration after concurrent readiness+delete", got.Status)
	}
	found := false
	for _, diagnostic := range got.Diagnostics {
		if diagnostic.Code == AgentPackageDiagnosticCodeKnowledgeBaseDeleted {
			found = true
		}
	}
	if !found {
		t.Fatalf("knowledge_base_deleted lost under concurrency: %+v", got.Diagnostics)
	}
}

func TestCommitVersionReadinessPreservesIDBasedDeleteDiagnosticAfterSameNameRecreate(t *testing.T) {
	versionStore := NewInMemoryAgentVersionStore()
	service := NewAgentVersionService(versionStore).WithSemanticModelLocker(sameNameLocker{})
	ctx := context.Background()
	now := time.Unix(100, 0)
	if _, err := versionStore.CreateAgentVersion(ctx, AgentVersionRecord{
		WorkspaceID: "ws_1",
		AgentID:     "agent_1",
		Version:     "1.0.0",
		Status:      AgentVersionStatusNeedsConfiguration,
		Diagnostics: []AgentPackageDiagnostic{
			knowledgeBaseDeletedDiagnostic("ws_1", "20001"), // DELETE-owned ID-based
			{
				Severity: "error",
				Code:     AgentPackageDiagnosticCodeKnowledgeBaseDeleted,
				Message:  "bound knowledge base is missing by name: kb_sales",
				Ref:      knowledgeBaseDeletedDiagnosticRef("ws_1", "name:kb_sales"), // readiness-owned name-based
			},
		},
		Manifest: AgentPackageManifest{
			MOIResourceRefs: AgentPackageMOIResourceRefs{
				KnowledgeBases: []AgentPackageNamedResourceRef{{Name: "kb_sales"}},
			},
		},
		LoadedAt: now,
	}); err != nil {
		t.Fatalf("CreateAgentVersion: %v", err)
	}

	updated, err := service.CommitVersionReadiness(ctx, "ws_1", "agent_1", "1.0.0", nil, nil)
	if err != nil {
		t.Fatalf("CommitVersionReadiness: %v", err)
	}
	if updated.Status != AgentVersionStatusNeedsConfiguration {
		t.Fatalf("status = %s, want needs_configuration", updated.Status)
	}
	foundID := false
	foundName := false
	for _, diagnostic := range updated.Diagnostics {
		if diagnostic.Code != AgentPackageDiagnosticCodeKnowledgeBaseDeleted {
			continue
		}
		switch diagnostic.Ref {
		case "ws_1/20001":
			foundID = true
		case "ws_1/name:kb_sales":
			foundName = true
		}
	}
	if !foundID {
		t.Fatalf("ID-based knowledge_base_deleted was cleared after same-name recreation: %+v", updated.Diagnostics)
	}
	if foundName {
		t.Fatalf("name-based knowledge_base_deleted should be cleared once same-name KB exists: %+v", updated.Diagnostics)
	}
}
