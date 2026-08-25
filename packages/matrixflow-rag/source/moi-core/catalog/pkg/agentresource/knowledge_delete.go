package agentresource

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/tenant"
)

const (
	// AgentPackageDiagnosticCodeKnowledgeBaseDeleted is owned by semantic knowledge base delete.
	// Runner/provider readiness filters must preserve it.
	AgentPackageDiagnosticCodeKnowledgeBaseDeleted = "knowledge_base_deleted"
)

// SemanticKnowledgeBaseDeleteStats summarizes agent-side side effects of deleting one semantic knowledge base.
type SemanticKnowledgeBaseDeleteStats struct {
	UnboundAgents           int `json:"unbound_agents"`
	UnboundAgentBindings    int `json:"unbound_agent_bindings"`
	NeedsConfigurationCount int `json:"needs_configuration_count"`
}

// SemanticKnowledgeBaseBindingDeleteStore owns mutable binding cleanup for knowledge-base delete consistency.
type SemanticKnowledgeBaseBindingDeleteStore interface {
	ListAgentBindingsReferencingKnowledgeBaseForUpdate(ctx context.Context, workspaceID, knowledgeBaseID string) ([]AgentBindingRecord, error)
	ListAgentsReferencingKnowledgeBaseForUpdate(ctx context.Context, workspaceID, knowledgeBaseID string) ([]AgentMetadata, error)
	UpdateAgent(ctx context.Context, agent AgentMetadata) (*AgentMetadata, error)
	UpsertAgentBinding(ctx context.Context, binding AgentBindingRecord) (*AgentBindingRecord, error)
}

// SemanticKnowledgeBaseVersionDeleteStore owns package version invalidation for knowledge-base delete/rename consistency.
type SemanticKnowledgeBaseVersionDeleteStore interface {
	// ListNonDisabledAgentVersions returns non-disabled package versions without locking.
	// Callers must lock only the matched subset before mutating status/diagnostics.
	ListNonDisabledAgentVersions(ctx context.Context, workspaceID string) ([]AgentVersionRecord, error)
	UpdateAgentVersionStatusAndDiagnostics(ctx context.Context, workspaceID, agentID, version, status string, diagnostics []AgentPackageDiagnostic) (*AgentVersionRecord, error)
	GetAgentVersionForUpdate(ctx context.Context, workspaceID, agentID, version string) (*AgentVersionRecord, error)
}

// agentVersionStatusDiagnosticsApplier applies a status/diagnostics mutation under one store critical section.
// Memory store implements this so concurrent readiness/delete cannot interleave between read and write.
// apply returns write=false to leave the row unchanged.
type agentVersionStatusDiagnosticsApplier interface {
	ApplyAgentVersionStatusAndDiagnostics(
		ctx context.Context,
		workspaceID, agentID, version string,
		apply func(current AgentVersionRecord) (status string, diagnostics []AgentPackageDiagnostic, write bool, err error),
	) (*AgentVersionRecord, error)
}

// HandleSemanticKnowledgeBaseDeleted removes current-workspace references to a deleted semantic knowledge base
// from ordinary agent bindings and workspace overlays, and marks non-disabled package versions that reference
// the knowledge base by name as needs_configuration with a stable knowledge_base_deleted diagnostic.
//
// Callers must already hold a writable tenant transaction that has locked the target semantic model row.
func (s *AgentService) HandleSemanticKnowledgeBaseDeleted(
	ctx context.Context,
	workspaceID string,
	modelID int64,
	modelName string,
	userID string,
) (SemanticKnowledgeBaseDeleteStats, error) {
	var stats SemanticKnowledgeBaseDeleteStats
	if s == nil || s.store == nil {
		return stats, fmt.Errorf("agent resource store is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOpaqueID(workspaceID, "workspace_id", wrapInvalidAgentID); err != nil {
		return stats, err
	}
	if modelID <= 0 {
		return stats, fmt.Errorf("%w: model_id is required", ErrInvalidAgent)
	}
	knowledgeBaseID := strconv.FormatInt(modelID, 10)
	modelName = strings.TrimSpace(modelName)

	bindingDeleteStore, ok := s.store.(SemanticKnowledgeBaseBindingDeleteStore)
	if !ok {
		return stats, fmt.Errorf("agent resource store does not support semantic knowledge base binding cleanup")
	}
	now := s.currentTime()

	// Lock order after the caller's semantic-model lock:
	// 1) non-locking scan of non-disabled package versions
	// 2) lock only matched version rows (sorted) and re-check status
	// 3) lock/clean ordinary agents and overlays
	// This matches publication (model -> version -> root agent) while avoiding
	// locking unrelated versions for the whole workspace.
	if modelName != "" {
		versionDeleteStore, err := s.semanticKnowledgeBaseVersionDeleteStore()
		if err != nil {
			return stats, err
		}
		// DELETE-owned ID-based diagnostic so same-name recreation cannot clear it.
		diagnostic := knowledgeBaseDeletedDiagnostic(workspaceID, knowledgeBaseID)
		needsConfigurationCount, err := s.invalidatePackageVersionsReferencingKnowledgeBaseName(
			ctx, versionDeleteStore, workspaceID, modelName, diagnostic,
		)
		if err != nil {
			return stats, err
		}
		stats.NeedsConfigurationCount = needsConfigurationCount
	}

	agents, err := bindingDeleteStore.ListAgentsReferencingKnowledgeBaseForUpdate(ctx, workspaceID, knowledgeBaseID)
	if err != nil {
		return stats, err
	}
	for _, agent := range agents {
		next, changed := stripKnowledgeBaseFromBindingSummary(agent.Binding, workspaceID, knowledgeBaseID)
		if !changed {
			continue
		}
		agent.Binding = next
		agent.UpdatedBy = userID
		agent.UpdatedAt = now
		if _, err := bindingDeleteStore.UpdateAgent(ctx, agent); err != nil {
			return stats, fmt.Errorf("unbind knowledge base from agent %s: %w", agent.ID, err)
		}
		stats.UnboundAgents++
	}

	overlays, err := bindingDeleteStore.ListAgentBindingsReferencingKnowledgeBaseForUpdate(ctx, workspaceID, knowledgeBaseID)
	if err != nil {
		return stats, err
	}
	for _, overlay := range overlays {
		next, changed := stripKnowledgeBaseFromBindingSummary(overlay.Binding, workspaceID, knowledgeBaseID)
		if !changed {
			continue
		}
		overlay.Binding = next
		overlay.UpdatedBy = userID
		if overlay.CreatedBy == "" {
			overlay.CreatedBy = userID
		}
		overlay.UpdatedAt = now
		if overlay.CreatedAt.IsZero() {
			overlay.CreatedAt = now
		}
		if _, err := bindingDeleteStore.UpsertAgentBinding(ctx, overlay); err != nil {
			return stats, fmt.Errorf("unbind knowledge base from agent binding %s/%s: %w", overlay.AgentWorkspaceID, overlay.AgentID, err)
		}
		stats.UnboundAgentBindings++
	}
	return stats, nil
}

// HandleSemanticKnowledgeBaseRenamed invalidates non-disabled package versions that still reference the previous
// knowledge base name. Bindings are ID-based and do not need cleanup. Callers must already hold a writable tenant
// transaction that has locked the target semantic model row.
func (s *AgentService) HandleSemanticKnowledgeBaseRenamed(
	ctx context.Context,
	workspaceID string,
	modelID int64,
	oldName, newName string,
) (SemanticKnowledgeBaseDeleteStats, error) {
	var stats SemanticKnowledgeBaseDeleteStats
	if s == nil || s.store == nil {
		return stats, fmt.Errorf("agent resource store is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOpaqueID(workspaceID, "workspace_id", wrapInvalidAgentID); err != nil {
		return stats, err
	}
	if modelID <= 0 {
		return stats, fmt.Errorf("%w: model_id is required", ErrInvalidAgent)
	}
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" || oldName == newName {
		return stats, nil
	}
	versionDeleteStore, err := s.semanticKnowledgeBaseVersionDeleteStore()
	if err != nil {
		return stats, err
	}
	// Name-based diagnostic: the model row still exists under newName, but the immutable
	// package manifest still points at oldName. Readiness can clear this if the old name
	// is recreated, or after the package is rematerialized against the new name.
	diagnostic := AgentPackageDiagnostic{
		Severity: "error",
		Code:     AgentPackageDiagnosticCodeKnowledgeBaseDeleted,
		Message:  "bound knowledge base is missing by name: " + oldName,
		Ref:      knowledgeBaseDeletedDiagnosticRef(workspaceID, "name:"+oldName),
	}
	needsConfigurationCount, err := s.invalidatePackageVersionsReferencingKnowledgeBaseName(
		ctx, versionDeleteStore, workspaceID, oldName, diagnostic,
	)
	if err != nil {
		return stats, err
	}
	stats.NeedsConfigurationCount = needsConfigurationCount
	return stats, nil
}

func (s *AgentService) semanticKnowledgeBaseVersionDeleteStore() (SemanticKnowledgeBaseVersionDeleteStore, error) {
	if versionStore, ok := s.store.(SemanticKnowledgeBaseVersionDeleteStore); ok {
		return versionStore, nil
	}
	if s.versionService != nil {
		if versionStore, ok := s.versionService.store.(SemanticKnowledgeBaseVersionDeleteStore); ok {
			return versionStore, nil
		}
	}
	return nil, fmt.Errorf("agent version store does not support semantic knowledge base version cleanup")
}

func (s *AgentService) invalidatePackageVersionsReferencingKnowledgeBaseName(
	ctx context.Context,
	versionDeleteStore SemanticKnowledgeBaseVersionDeleteStore,
	workspaceID, modelName string,
	diagnostic AgentPackageDiagnostic,
) (int, error) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return 0, nil
	}
	if versionDeleteStore == nil {
		return 0, fmt.Errorf("agent version store does not support semantic knowledge base version cleanup")
	}
	candidateVersions, err := versionDeleteStore.ListNonDisabledAgentVersions(ctx, workspaceID)
	if err != nil {
		return 0, err
	}

	type versionKey struct {
		agentID string
		version string
	}
	matchedKeys := make([]versionKey, 0)
	for _, version := range candidateVersions {
		if !agentPackageManifestReferencesKnowledgeBase(version.Manifest, modelName) {
			continue
		}
		matchedKeys = append(matchedKeys, versionKey{agentID: version.AgentID, version: version.Version})
	}
	sort.Slice(matchedKeys, func(i, j int) bool {
		if matchedKeys[i].agentID != matchedKeys[j].agentID {
			return matchedKeys[i].agentID < matchedKeys[j].agentID
		}
		return matchedKeys[i].version < matchedKeys[j].version
	})

	applier, _ := versionDeleteStore.(agentVersionStatusDiagnosticsApplier)
	count := 0
	for _, key := range matchedKeys {
		if applier != nil {
			updated, err := applier.ApplyAgentVersionStatusAndDiagnostics(ctx, workspaceID, key.agentID, key.version, func(current AgentVersionRecord) (string, []AgentPackageDiagnostic, bool, error) {
				if current.Status == AgentVersionStatusDisabled {
					return "", nil, false, nil
				}
				if !agentPackageManifestReferencesKnowledgeBase(current.Manifest, modelName) {
					return "", nil, false, nil
				}
				nextDiagnostics, changed := appendKnowledgeBaseDeletedDiagnostic(current.Diagnostics, diagnostic)
				status := agentVersionStatusForDiagnostics(nextDiagnostics)
				if !changed && current.Status == status {
					return "", nil, false, nil
				}
				return status, nextDiagnostics, true, nil
			})
			if err != nil {
				if errors.Is(err, ErrAgentVersionNotFound) {
					continue
				}
				return count, fmt.Errorf("mark agent version %s/%s needs_configuration: %w", key.agentID, key.version, err)
			}
			if updated != nil {
				count++
			}
			continue
		}

		current, err := versionDeleteStore.GetAgentVersionForUpdate(ctx, workspaceID, key.agentID, key.version)
		if err != nil {
			if errors.Is(err, ErrAgentVersionNotFound) {
				continue
			}
			return count, err
		}
		if current.Status == AgentVersionStatusDisabled {
			continue
		}
		if !agentPackageManifestReferencesKnowledgeBase(current.Manifest, modelName) {
			continue
		}
		nextDiagnostics, changed := appendKnowledgeBaseDeletedDiagnostic(current.Diagnostics, diagnostic)
		status := agentVersionStatusForDiagnostics(nextDiagnostics)
		if !changed && current.Status == status {
			continue
		}
		if _, err := versionDeleteStore.UpdateAgentVersionStatusAndDiagnostics(ctx, current.WorkspaceID, current.AgentID, current.Version, status, nextDiagnostics); err != nil {
			return count, fmt.Errorf("mark agent version %s/%s needs_configuration: %w", current.AgentID, current.Version, err)
		}
		count++
	}
	return count, nil
}

func stripKnowledgeBaseFromBindingSummary(binding AgentBindingSummary, workspaceID, knowledgeBaseID string) (AgentBindingSummary, bool) {
	next := normalizeAgentBindingSummary(binding)
	changed := false

	if len(next.KnowledgeBaseRefs) > 0 {
		filtered := make([]AgentBindingResourceRef, 0, len(next.KnowledgeBaseRefs))
		for _, ref := range next.KnowledgeBaseRefs {
			refWorkspaceID := bindingResourceWorkspaceID(workspaceID, ref)
			if knowledgeBaseIDsEqual(ref.ID, knowledgeBaseID) && refWorkspaceID == workspaceID {
				changed = true
				continue
			}
			filtered = append(filtered, ref)
		}
		if len(filtered) == 0 {
			next.KnowledgeBaseRefs = nil
		} else {
			next.KnowledgeBaseRefs = filtered
		}
	}
	return next, changed
}

// knowledgeBaseIDsEqual matches resolver/lock numeric semantics: exact string or the same
// positive decimal value (so "000123" / "+123" match canonical "123").
func knowledgeBaseIDsEqual(left, right string) bool {
	if left == right {
		return true
	}
	leftID, leftErr := strconv.ParseInt(strings.TrimSpace(left), 10, 64)
	rightID, rightErr := strconv.ParseInt(strings.TrimSpace(right), 10, 64)
	if leftErr != nil || rightErr != nil {
		return false
	}
	return leftID > 0 && leftID == rightID
}

func agentPackageManifestReferencesKnowledgeBase(manifest AgentPackageManifest, modelName string) bool {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return false
	}
	for _, ref := range manifest.MOIResourceRefs.KnowledgeBases {
		if strings.TrimSpace(ref.Name) != modelName {
			continue
		}
		// Package load resolves knowledge bases only in the target workspace and ignores
		// ref.WorkspaceID (see resolveAgentPackageKnowledgeBaseRefIDs). Match that contract so
		// delete/rename invalidation covers every ref that would have bound this workspace KB.
		return true
	}
	return false
}

func knowledgeBaseDeletedDiagnostic(workspaceID, knowledgeBaseID string) AgentPackageDiagnostic {
	return AgentPackageDiagnostic{
		Severity: "error",
		Code:     AgentPackageDiagnosticCodeKnowledgeBaseDeleted,
		Message:  "bound knowledge base was deleted: " + knowledgeBaseID,
		Ref:      knowledgeBaseDeletedDiagnosticRef(workspaceID, knowledgeBaseID),
	}
}

func knowledgeBaseDeletedDiagnosticRef(workspaceID, knowledgeBaseID string) string {
	return strings.TrimSpace(workspaceID) + "/" + strings.TrimSpace(knowledgeBaseID)
}

func appendKnowledgeBaseDeletedDiagnostic(diagnostics []AgentPackageDiagnostic, diagnostic AgentPackageDiagnostic) ([]AgentPackageDiagnostic, bool) {
	out := append([]AgentPackageDiagnostic(nil), diagnostics...)
	for _, existing := range out {
		if existing.Code == diagnostic.Code && existing.Ref == diagnostic.Ref {
			return out, false
		}
	}
	out = append(out, diagnostic)
	return out, true
}

func mergeAgentPackageDiagnosticsByCodeRef(base []AgentPackageDiagnostic, owned []AgentPackageDiagnostic) []AgentPackageDiagnostic {
	out := append([]AgentPackageDiagnostic(nil), base...)
	for _, diagnostic := range owned {
		replaced := false
		for i := range out {
			if out[i].Code == diagnostic.Code && out[i].Ref == diagnostic.Ref {
				out[i] = diagnostic
				replaced = true
				break
			}
		}
		if !replaced {
			out = append(out, diagnostic)
		}
	}
	return out
}

// knowledgeBaseNumericBindingRef pairs a positive decimal identity used for semantic-model
// locking with every raw binding string that parsed to that identity. Legacy
// agent_resource_knowledge_bases IDs are opaque, so fallback resolve must use the raw form
// ("000123" / "+123"), not only strconv.FormatInt(id, 10).
type knowledgeBaseNumericBindingRef struct {
	id   int64
	raws []string
}

func knowledgeBaseNumericRefsFromBindingSummary(workspaceID string, binding AgentBindingSummary) []knowledgeBaseNumericBindingRef {
	byID := make(map[int64]*knowledgeBaseNumericBindingRef)
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return
		}
		entry, ok := byID[id]
		if !ok {
			entry = &knowledgeBaseNumericBindingRef{id: id}
			byID[id] = entry
		}
		for _, existing := range entry.raws {
			if existing == raw {
				return
			}
		}
		entry.raws = append(entry.raws, raw)
	}
	for _, ref := range binding.KnowledgeBaseRefs {
		if bindingResourceWorkspaceID(workspaceID, ref) != workspaceID {
			continue
		}
		add(ref.ID)
	}
	out := make([]knowledgeBaseNumericBindingRef, 0, len(byID))
	for _, entry := range byID {
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

func knowledgeBaseNamesFromManifest(manifest AgentPackageManifest) []string {
	if len(manifest.MOIResourceRefs.KnowledgeBases) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(manifest.MOIResourceRefs.KnowledgeBases))
	out := make([]string, 0, len(manifest.MOIResourceRefs.KnowledgeBases))
	for _, ref := range manifest.MOIResourceRefs.KnowledgeBases {
		name := strings.TrimSpace(ref.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// semanticModelLockStorage locks semantic models inside an ambient tenant transaction.
type semanticModelLockStorage interface {
	LockSemanticModelsForUpdate(ctx context.Context, modelIDs []int64) ([]*tenant.SemanticModelRecord, error)
	GetSemanticModelForUpdate(ctx context.Context, modelID int64) (*tenant.SemanticModelRecord, error)
	ListSemanticModels(ctx context.Context, opts ...tenant.ListOption) ([]*tenant.SemanticModelRecord, int64, error)
}

func lockBindingKnowledgeBases(
	ctx context.Context,
	workspaceID string,
	binding AgentBindingSummary,
	locker semanticModelLockStorage,
) error {
	if locker == nil {
		return nil
	}
	numericRefs := knowledgeBaseNumericRefsFromBindingSummary(workspaceID, binding)
	if len(numericRefs) == 0 {
		return nil
	}
	modelIDs := make([]int64, 0, len(numericRefs))
	for _, ref := range numericRefs {
		modelIDs = append(modelIDs, ref.id)
	}
	locked, err := locker.LockSemanticModelsForUpdate(ctx, modelIDs)
	if err != nil {
		return err
	}
	found := make(map[int64]struct{}, len(locked))
	for _, model := range locked {
		if model == nil {
			continue
		}
		found[model.ID] = struct{}{}
	}
	for _, ref := range numericRefs {
		if _, ok := found[ref.id]; ok {
			continue
		}
		display := strconv.FormatInt(ref.id, 10)
		if len(ref.raws) > 0 {
			display = ref.raws[0]
		}
		return wrapInvalidAgent("binding knowledge base does not exist: " + display)
	}
	return nil
}

func resolveAndLockManifestKnowledgeBases(
	ctx context.Context,
	workspaceID string,
	manifest AgentPackageManifest,
	locker semanticModelLockStorage,
) ([]AgentPackageDiagnostic, error) {
	if locker == nil {
		return nil, nil
	}
	names := knowledgeBaseNamesFromManifest(manifest)
	if len(names) == 0 {
		return nil, nil
	}
	type resolvedKB struct {
		name string
		id   int64
	}
	var diagnostics []AgentPackageDiagnostic
	resolved := make([]resolvedKB, 0, len(names))
	for _, name := range names {
		// Exact equality lookup (not fuzzy search) so unique name index is usable and
		// page-limited LIKE cannot false-miss an existing model.
		models, total, err := locker.ListSemanticModels(ctx, tenant.WithPageSize(2), tenant.WithFilter("name", []string{name}, false))
		if err != nil {
			return nil, fmt.Errorf("resolve knowledge base %q for version readiness: %w", name, err)
		}
		if total == 0 || len(models) == 0 {
			diagnostics = append(diagnostics, AgentPackageDiagnostic{
				Severity: "error",
				Code:     AgentPackageDiagnosticCodeKnowledgeBaseDeleted,
				Message:  "bound knowledge base is missing by name: " + name,
				Ref:      knowledgeBaseDeletedDiagnosticRef(workspaceID, "name:"+name),
			})
			continue
		}
		if total > 1 || len(models) > 1 {
			return nil, fmt.Errorf("%w: knowledge base %q is ambiguous in target workspace", ErrInvalidAgentVersion, name)
		}
		resolved = append(resolved, resolvedKB{name: name, id: models[0].ID})
	}
	if len(resolved) == 0 {
		return diagnostics, nil
	}
	modelIDs := make([]int64, 0, len(resolved))
	for _, item := range resolved {
		modelIDs = append(modelIDs, item.id)
	}
	locked, err := locker.LockSemanticModelsForUpdate(ctx, modelIDs)
	if err != nil {
		return nil, err
	}
	foundByID := make(map[int64]*tenant.SemanticModelRecord, len(locked))
	for _, model := range locked {
		if model == nil {
			continue
		}
		foundByID[model.ID] = model
	}
	// Re-check names after lock so a concurrent rename cannot leave readiness thinking the
	// pre-lock name still exists on the locked row.
	for _, item := range resolved {
		model, ok := foundByID[item.id]
		if !ok {
			diagnostics = append(diagnostics, knowledgeBaseDeletedDiagnostic(workspaceID, strconv.FormatInt(item.id, 10)))
			continue
		}
		if strings.TrimSpace(model.Name) != item.name {
			diagnostics = append(diagnostics, AgentPackageDiagnostic{
				Severity: "error",
				Code:     AgentPackageDiagnosticCodeKnowledgeBaseDeleted,
				Message:  "bound knowledge base is missing by name: " + item.name,
				Ref:      knowledgeBaseDeletedDiagnosticRef(workspaceID, "name:"+item.name),
			})
		}
	}
	return diagnostics, nil
}
