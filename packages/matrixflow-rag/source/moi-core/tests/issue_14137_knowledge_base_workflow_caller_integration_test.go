package tests

import (
	"context"
	"crypto/sha1"
	"fmt"
	"testing"
	"time"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/tests/framework"
	"github.com/stretchr/testify/require"
)

func TestIssue14137KnowledgeBaseWorkflowUsesCallerEffectiveRole(t *testing.T) {
	if testing.Short() {
		t.Skip("requires MatrixOne and Mowl")
	}

	framework.RunMOITests(t, func(env *framework.TestEnv) {
		env.RequireSharedWorkspace(t)
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		effectiveRoleID := env.SharedEffectiveRoleID
		require.NotEmpty(t, effectiveRoleID)
		client, err := env.GetSharedBFFClient()
		require.NoError(t, err)
		t.Cleanup(func() { client.Close() })

		modelID := time.Now().UnixNano()
		digest := sha1.Sum([]byte(fmt.Sprintf("%s\x00%d\x00", env.SharedWorkspaceID, modelID)))
		workflowID := "kb-rag-workflow-" + fmt.Sprintf("%x", digest)[:24]
		workflowName := fmt.Sprintf("issue-14137-kb-%d", modelID)
		const dslYAML = "workflow:\n  name: issue-14137-kb\n  root: root\nroot:\n  chain: []\n"

		deployed, err := client.WorkflowDeployments(env.SharedWorkspaceID).Deploy(ctx, &moi.WorkflowDeploymentRequest{
			Name:              workflowName,
			Description:       "caller-owned knowledge base workflow regression",
			DSLYAML:           dslYAML,
			ExecutionMode:     "one_shot",
			DataJSON:          `{}`,
			VarsJSON:          `{}`,
			WorkflowID:        workflowID,
			SourceType:        "manual_dsl",
			Status:            "ready",
			DefaultValuesJSON: fmt.Sprintf(`{"semantic_model_id":%d}`, modelID),
		})
		require.NoError(t, err)
		require.NotNil(t, deployed.Deployment)
		require.Equal(t, workflowID, deployed.Deployment.WorkflowAppID)
		require.NotEmpty(t, deployed.Deployment.WorkflowDefinitionID)
		require.NotEmpty(t, deployed.Deployment.WorkflowVersionID)

		var executionID string
		t.Cleanup(func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cleanupCancel()
			if executionID != "" {
				_, _ = client.WorkflowApps(env.SharedWorkspaceID).CancelExecution(cleanupCtx, workflowID, executionID)
				_, _ = client.WorkflowApps(env.SharedWorkspaceID).DeleteExecution(cleanupCtx, workflowID, executionID)
			}
			_, _ = client.WorkflowApps(env.SharedWorkspaceID).Delete(cleanupCtx, workflowID)
		})

		app, err := client.WorkflowApps(env.SharedWorkspaceID).Get(ctx, workflowID)
		require.NoError(t, err)
		require.Equal(t, workflowID, app.Workflow.ID)
		require.Equal(t, "manual_dsl", app.Workflow.SourceType)
		require.Equal(t, "one_shot", app.Workflow.ExecutionMode)
		require.Equal(t, deployed.Deployment.WorkflowDefinitionID, app.Workflow.MoiWorkflowDefID)
		require.Equal(t, deployed.Deployment.WorkflowVersionID, app.Workflow.LatestVersionID)
		require.JSONEq(t, fmt.Sprintf(`{"semantic_model_id":%d}`, modelID), app.Workflow.DefaultValuesJSON)

		version, err := client.WorkflowVersions(env.SharedWorkspaceID).Get(ctx, deployed.Deployment.WorkflowVersionID)
		require.NoError(t, err)
		require.Equal(t, deployed.Deployment.WorkflowVersionID, version.GetId())
		require.Equal(t, env.SharedWorkspaceID, version.GetWorkspaceId())
		require.Equal(t, deployed.Deployment.WorkflowDefinitionID, version.GetWorkflowId())
		require.Equal(t, env.SharedUserID, version.GetCreatedBy())
		require.Equal(t, "published", version.GetStatus())

		execution, err := client.WorkflowApps(env.SharedWorkspaceID).CreateExecution(ctx, workflowID, &moi.WorkflowExecutionCreateRequest{
			InputPayloadJSON: `{"value":"issue-14137"}`,
			VarsPayloadJSON:  `{}`,
		})
		require.NoError(t, err)
		require.NotNil(t, execution)
		executionID = execution.Execution.ExecutionID
		require.NotEmpty(t, executionID)
		require.Equal(t, workflowID, execution.Execution.WorkflowID)
		require.Equal(t, deployed.Deployment.WorkflowVersionID, execution.Execution.MoiWorkflowVersion)
		require.NotEmpty(t, execution.Execution.MoiTaskID)

		systemDB, err := env.DB()
		require.NoError(t, err)
		var appUserID, appDefinitionID, appVersionID string
		err = systemDB.QueryRowContext(ctx, `
SELECT user_id, mowl_workflow_def_id, latest_workflow_version_id
FROM workflow_app
WHERE workspace_id = ? AND id = ?`, env.SharedWorkspaceID, workflowID).
			Scan(&appUserID, &appDefinitionID, &appVersionID)
		require.NoError(t, err)
		require.Equal(t, deployed.Deployment.WorkflowDefinitionID, appDefinitionID)
		require.Equal(t, deployed.Deployment.WorkflowVersionID, appVersionID)

		var definitionUserID string
		err = systemDB.QueryRowContext(ctx, `
SELECT user_id
FROM mowl_workflow_definition
WHERE workspace_id = ? AND id = ?`, env.SharedWorkspaceID, appDefinitionID).
			Scan(&definitionUserID)
		require.NoError(t, err)

		var versionUserID, versionCreatedBy string
		err = systemDB.QueryRowContext(ctx, `
SELECT user_id, created_by
FROM mowl_workflow_version
WHERE workspace_id = ? AND id = ? AND workflow_id = ?`, env.SharedWorkspaceID, appVersionID, appDefinitionID).
			Scan(&versionUserID, &versionCreatedBy)
		require.NoError(t, err)

		var executionUserID, executionTaskID, executionDefinitionID, executionVersionID string
		err = systemDB.QueryRowContext(ctx, `
SELECT user_id, mowl_task_id, mowl_workflow_def_id, mowl_workflow_version_id
FROM workflow_execution
WHERE workspace_id = ? AND workflow_id = ? AND id = ?`, env.SharedWorkspaceID, workflowID, executionID).
			Scan(&executionUserID, &executionTaskID, &executionDefinitionID, &executionVersionID)
		require.NoError(t, err)
		require.Equal(t, execution.Execution.MoiTaskID, executionTaskID)
		require.Equal(t, appDefinitionID, executionDefinitionID)
		require.Equal(t, appVersionID, executionVersionID)

		var taskUserID, taskVersionID string
		err = systemDB.QueryRowContext(ctx, `
SELECT user_id, workflow_version_id
FROM mowl_task
WHERE workspace_id = ? AND id = ?`, env.SharedWorkspaceID, executionTaskID).
			Scan(&taskUserID, &taskVersionID)
		require.NoError(t, err)
		require.Equal(t, appVersionID, taskVersionID)
		for _, ownerUserID := range []string{appUserID, definitionUserID, versionUserID, versionCreatedBy, executionUserID, taskUserID} {
			require.Equal(t, env.SharedUserID, ownerUserID)
		}

		tenantDB, err := env.OpenWorkspaceAdminDB(ctx, env.SharedWorkspaceID)
		require.NoError(t, err)
		defer tenantDB.Close()
		var ownerRoleID, ownershipStatus, createdByPrincipalID, createdByRoleID string
		err = tenantDB.QueryRowContext(ctx, `
SELECT owner_role_id, status, created_by_principal_id, created_by_role_id
FROM iam_resource_ownership
WHERE workspace_id = ? AND resource_type = 'workflow' AND resource_id = ?`,
			env.SharedWorkspaceID, workflowID,
		).Scan(&ownerRoleID, &ownershipStatus, &createdByPrincipalID, &createdByRoleID)
		require.NoError(t, err)
		require.Equal(t, "active", ownershipStatus)
		require.Equal(t, effectiveRoleID, ownerRoleID)
		require.Equal(t, env.SharedUserID, createdByPrincipalID)
		require.Equal(t, effectiveRoleID, createdByRoleID)
	})
}
