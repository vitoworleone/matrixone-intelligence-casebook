package iamcore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/tenant"
	authzcore "github.com/matrixorigin/matrixflow/shared/authz/pkg/core"
)

type SemanticModelResourceRegisterRequest struct {
	WorkspaceID, SemanticModelID, PrincipalID, RoleID string
	OperationID, RequestID, TraceID                   string
}

type SemanticModelResourceDeleteRequest struct {
	WorkspaceID, SemanticModelID, PrincipalID, RoleID string
	OperationID, RequestID, TraceID                   string
}

// SemanticModelResourceRegistrar is the trusted ownership lifecycle boundary
// for Core-owned semantic models. IAM lifecycle remains in the canonical IAM
// tables; semantic_models stores business state only.
type SemanticModelResourceRegistrar struct {
	Authorizer *IAMCoreAuthorizer
	Writer     WorkflowResourceOwnershipWriter
}

func (r *SemanticModelResourceRegistrar) Register(ctx context.Context, req SemanticModelResourceRegisterRequest) (*tenant.IAMResourceOwnershipResult, error) {
	if err := r.validate(req.WorkspaceID, req.SemanticModelID, req.PrincipalID, req.OperationID, req.RequestID, req.TraceID); err != nil {
		return nil, err
	}
	payloadHash := semanticModelLifecyclePayloadHash(req.WorkspaceID, req.SemanticModelID, req.PrincipalID, req.RoleID)
	if replay, ok, err := r.Writer.ReplayIAMResourceLifecycleOperation(ctx, tenant.IAMResourceLifecycleReplayRequest{
		WorkspaceID: req.WorkspaceID, OperationType: "create", RequestID: req.RequestID, PayloadHash: payloadHash,
		ResourceType: IAMResourceSemanticModel, ResourceID: req.SemanticModelID,
	}); err != nil {
		return nil, err
	} else if ok {
		return replay, nil
	}
	role, err := r.authorize(ctx, req.WorkspaceID, req.PrincipalID, req.RoleID, IAMActionSemanticModelCreate, IAMResourceWorkspace, req.WorkspaceID, req.RequestID, req.TraceID)
	if err != nil {
		return nil, err
	}
	return r.RegisterAuthorized(ctx, req, role.RoleID)
}

func (r *SemanticModelResourceRegistrar) RegisterAuthorized(ctx context.Context, req SemanticModelResourceRegisterRequest, verifiedEffectiveRoleID string) (*tenant.IAMResourceOwnershipResult, error) {
	if err := r.validate(req.WorkspaceID, req.SemanticModelID, req.PrincipalID, req.OperationID, req.RequestID, req.TraceID); err != nil {
		return nil, err
	}
	if err := requireCleanString(ErrIAMAdapterInvalidInput, "verified_effective_role_id", verifiedEffectiveRoleID); err != nil {
		return nil, err
	}
	payloadHash := semanticModelLifecyclePayloadHash(req.WorkspaceID, req.SemanticModelID, req.PrincipalID, verifiedEffectiveRoleID)
	if replay, ok, err := r.Writer.ReplayIAMResourceLifecycleOperation(ctx, tenant.IAMResourceLifecycleReplayRequest{
		WorkspaceID: req.WorkspaceID, OperationType: "create", RequestID: req.RequestID, PayloadHash: payloadHash,
		ResourceType: IAMResourceSemanticModel, ResourceID: req.SemanticModelID,
	}); err != nil {
		return nil, err
	} else if ok {
		return replay, nil
	}
	summary, err := json.Marshal(map[string]string{"event_type": "semantic_model_owner_registered", "resource_type": IAMResourceSemanticModel, "resource_id": req.SemanticModelID, "owner_role_id": verifiedEffectiveRoleID})
	if err != nil {
		return nil, fmt.Errorf("%w: encode semantic model ownership audit summary", ErrIAMAdapterUnavailable)
	}
	audit := sha256.Sum256([]byte("semantic_model_owner\x00" + req.RequestID + "\x00" + req.TraceID))
	return r.Writer.RegisterIAMResourceOwnership(ctx, tenant.IAMResourceOwnershipRegister{
		WorkspaceID: req.WorkspaceID, ResourceType: IAMResourceSemanticModel, ResourceID: req.SemanticModelID, OwnerRoleID: verifiedEffectiveRoleID,
		OperationID: req.OperationID, RequestID: req.RequestID, PayloadHash: payloadHash,
		ActorPrincipalType: tenant.IAMPrincipalTypeUser, ActorPrincipalID: req.PrincipalID, ActorActiveRoleID: verifiedEffectiveRoleID,
		AuditEventID: "aud_" + hex.EncodeToString(audit[:16]), TraceID: req.TraceID, AuditSummaryJSON: string(summary),
	})
}

func (r *SemanticModelResourceRegistrar) BeginDelete(ctx context.Context, req SemanticModelResourceDeleteRequest) (*tenant.IAMResourceOwnershipResult, error) {
	if err := r.validate(req.WorkspaceID, req.SemanticModelID, req.PrincipalID, req.OperationID, req.RequestID, req.TraceID); err != nil {
		return nil, err
	}
	payloadHash := semanticModelLifecyclePayloadHash(req.WorkspaceID, req.SemanticModelID, req.PrincipalID, req.RoleID)
	if replay, ok, err := r.Writer.ReplayIAMResourceLifecycleOperation(ctx, tenant.IAMResourceLifecycleReplayRequest{
		WorkspaceID: req.WorkspaceID, OperationType: "delete", RequestID: req.RequestID, PayloadHash: payloadHash,
		ResourceType: IAMResourceSemanticModel, ResourceID: req.SemanticModelID,
	}); err != nil {
		return nil, err
	} else if ok {
		return replay, nil
	}
	role, err := r.authorize(ctx, req.WorkspaceID, req.PrincipalID, req.RoleID, IAMActionSemanticModelDelete, IAMResourceSemanticModel, req.SemanticModelID, req.RequestID, req.TraceID)
	if err != nil {
		return nil, err
	}
	owner, err := getActiveOwnershipOrBootstrapSuperAdmin(ctx, r.Authorizer, r.Writer, superAdminLegacyOwnershipRequest{
		WorkspaceID: req.WorkspaceID, ResourceType: IAMResourceSemanticModel, ResourceID: req.SemanticModelID,
		PrincipalID: req.PrincipalID, EffectiveRoleID: role.RoleID, OperationID: req.OperationID, RequestID: req.RequestID, TraceID: req.TraceID,
	})
	if err != nil {
		return nil, err
	}
	summary, _ := json.Marshal(map[string]string{"event_type": "semantic_model_owner_deleting", "resource_type": IAMResourceSemanticModel, "resource_id": req.SemanticModelID})
	audit := sha256.Sum256([]byte("semantic_model_delete\x00" + req.RequestID + "\x00" + req.TraceID))
	return r.Writer.MarkIAMResourceOwnershipDeleting(ctx, tenant.IAMResourceOwnershipDelete{
		WorkspaceID: req.WorkspaceID, ResourceType: IAMResourceSemanticModel, ResourceID: req.SemanticModelID, ExpectedOwnershipVersion: owner.OwnershipVersion,
		OperationID: req.OperationID, RequestID: req.RequestID, PayloadHash: payloadHash,
		ActorPrincipalType: tenant.IAMPrincipalTypeUser, ActorPrincipalID: req.PrincipalID, ActorActiveRoleID: role.RoleID,
		AuditEventID: "aud_" + hex.EncodeToString(audit[:16]), TraceID: req.TraceID, AuditSummaryJSON: string(summary),
	})
}

func (r *SemanticModelResourceRegistrar) FinalizeDelete(ctx context.Context, req SemanticModelResourceDeleteRequest, ownershipVersion int64, authorizedRoleID string) error {
	authorizedRoleID = strings.TrimSpace(authorizedRoleID)
	if authorizedRoleID == "" {
		return fmt.Errorf("%w: authorized role snapshot is required", ErrIAMAdapterInvalidInput)
	}
	if err := cleanupSpecificResourceGrants(ctx, r.Authorizer, r.Writer, specificGrantCleanupRequest{WorkspaceID: req.WorkspaceID, ResourceType: IAMResourceSemanticModel, ResourceID: req.SemanticModelID, PrincipalID: req.PrincipalID, EffectiveRoleID: authorizedRoleID, OperationID: req.OperationID, RequestID: req.RequestID, TraceID: req.TraceID}); err != nil {
		return err
	}
	audit := sha256.Sum256([]byte("semantic_model_delete_final\x00" + req.RequestID + "\x00" + req.TraceID))
	summary, _ := json.Marshal(map[string]string{"event_type": "semantic_model_owner_deleted", "resource_type": IAMResourceSemanticModel, "resource_id": req.SemanticModelID})
	return r.Writer.FinalizeIAMResourceOwnershipDelete(ctx, tenant.IAMResourceOwnershipFinalizeDelete{WorkspaceID: req.WorkspaceID, ResourceType: IAMResourceSemanticModel, ResourceID: req.SemanticModelID, ExpectedOwnershipVersion: ownershipVersion, ActorPrincipalType: tenant.IAMPrincipalTypeUser, ActorPrincipalID: req.PrincipalID, ActorActiveRoleID: authorizedRoleID, RequestID: req.RequestID + ".finalize", TraceID: req.TraceID, AuditEventID: "aud_" + hex.EncodeToString(audit[:16]), AuditSummaryJSON: string(summary)})
}

func (r *SemanticModelResourceRegistrar) validate(values ...string) error {
	if r == nil || r.Authorizer == nil || interfaceIsNil(r.Writer) {
		return ErrIAMAdapterUnavailable
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" || strings.TrimSpace(value) != value {
			return fmt.Errorf("%w: semantic model lifecycle input is required and must be canonical", ErrIAMAdapterInvalidInput)
		}
	}
	return nil
}

func (r *SemanticModelResourceRegistrar) authorize(ctx context.Context, workspaceID, principalID, roleID, actionID, resourceType, resourceID, requestID, traceID string) (VerifiedEffectiveRole, error) {
	decision, err := r.Authorizer.Authorize(ctx, IAMAuthorizeRequest{PrincipalID: principalID, WorkspaceID: workspaceID, RoleID: roleID, Action: actionID, Resources: map[string]authzcore.ResourceRef{"resource": {TenantID: workspaceID, Kind: resourceType, ID: resourceID}}, Context: map[string]string{"request_id": deriveIAMSubrequestID(requestID, "semantic-model-lifecycle-authorize"), "trace_id": traceID, "entrypoint": "web"}})
	if err != nil {
		return VerifiedEffectiveRole{}, err
	}
	if !decision.Allow {
		return VerifiedEffectiveRole{}, authzcore.ErrDenied
	}
	return verifiedEffectiveRoleFromDecision(decision, workspaceID, principalID)
}

func semanticModelLifecyclePayloadHash(workspaceID, semanticModelID, principalID, roleID string) string {
	payload := sha256.Sum256([]byte(workspaceID + "\x00" + semanticModelID + "\x00" + principalID + "\x00" + roleID))
	return hex.EncodeToString(payload[:])
}
