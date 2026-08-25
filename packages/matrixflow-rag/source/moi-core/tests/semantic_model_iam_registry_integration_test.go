package tests

import (
	"context"
	"testing"

	"github.com/matrixflow/moi-core/catalog/pkg/iamcore"
	authzcore "github.com/matrixorigin/matrixflow/shared/authz/pkg/core"
	"github.com/stretchr/testify/require"
)

func TestSemanticModelIAMRegistryAndRouteContract(t *testing.T) {
	registry := iamcore.NewCanonicalIAMRegistry()
	ctx := context.Background()

	resource, err := registry.GetIAMResource(ctx, iamcore.IAMResourceSemanticModel)
	require.NoError(t, err)
	require.Equal(t, iamcore.IAMResourceSemanticModel, resource.ResourceType)
	require.ElementsMatch(t, []authzcore.ResourceScopeKind{
		authzcore.ResourceScopeExact,
		authzcore.ResourceScopeKindWide,
	}, resource.ScopeKinds)

	cases := []struct {
		actionID     string
		resourceType string
		mode         authzcore.RequirementMode
		directOnly   bool
	}{
		{iamcore.IAMActionSemanticModelCreate, iamcore.IAMResourceWorkspace, authzcore.RequirementModeWrite, true},
		{iamcore.IAMActionSemanticModelRead, iamcore.IAMResourceSemanticModel, authzcore.RequirementModeRead, false},
		{iamcore.IAMActionSemanticModelUpdate, iamcore.IAMResourceSemanticModel, authzcore.RequirementModeWrite, false},
		{iamcore.IAMActionSemanticModelDelete, iamcore.IAMResourceSemanticModel, authzcore.RequirementModeWrite, false},
		{iamcore.IAMActionSemanticModelUse, iamcore.IAMResourceSemanticModel, authzcore.RequirementModeRead, false},
	}
	for _, tc := range cases {
		action, err := registry.GetIAMAction(ctx, tc.resourceType, tc.actionID)
		require.NoError(t, err, tc.actionID)
		require.Equal(t, tc.resourceType, action.ResourceType)

		operation, err := registry.GetIAMOperation(ctx, tc.actionID)
		require.NoError(t, err, tc.actionID)
		require.Len(t, operation.Requirements, 1)
		require.Equal(t, tc.resourceType, operation.Requirements[0].ResourceType)
		require.Equal(t, tc.mode, operation.Requirements[0].Mode)
		require.Equal(t, tc.directOnly, operation.Requirements[0].DirectOnly)
		require.Equal(t, tc.directOnly, operation.Requirements[0].RequiresApplicationPolicy)
	}

	bindings := iamcore.CanonicalM4SemanticModelRouteActionBindings()
	require.NotEmpty(t, bindings)
	require.Contains(t, bindings, iamcore.IAMRouteActionBinding{
		RouteID:       "moi-backend.semantic_model.create",
		Method:        "POST",
		Path:          "/newmoi/semantic-models",
		Entrypoint:    iamcore.IAMRouteEntrypointWeb,
		ResolverKind:  iamcore.IAMRouteRoleCandidateExplicitOverride,
		ExtractorKind: iamcore.IAMRouteExtractorWorkspace,
		ResourceType:  iamcore.IAMResourceWorkspace,
		ActionID:      iamcore.IAMActionSemanticModelCreate,
		DirectOnly:    true,
	})

	seenUseAndUpdate := map[string]bool{}
	for _, binding := range bindings {
		require.NotEqual(t, "semantic_model.list", binding.ActionID)
		if binding.Path == "/newmoi/semantic-models/:model_id/sources" && binding.Method == "POST" {
			seenUseAndUpdate[binding.ActionID] = true
		}
	}
	require.True(t, seenUseAndUpdate[iamcore.IAMActionSemanticModelUse])
	require.True(t, seenUseAndUpdate[iamcore.IAMActionSemanticModelUpdate])
}
