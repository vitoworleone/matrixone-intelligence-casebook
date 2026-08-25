package iamcore

import (
	"context"
	"errors"
	"testing"

	"github.com/matrixflow/moi-core/catalog/pkg/service/storage/tenant"
	"github.com/stretchr/testify/require"
)

type semanticModelOwnershipWriterStub struct {
	replayRequest tenant.IAMResourceLifecycleReplayRequest
	replay        *tenant.IAMResourceOwnershipResult
	replayFound   bool
	replayErr     error
}

func (s *semanticModelOwnershipWriterStub) ReplayIAMResourceLifecycleOperation(_ context.Context, req tenant.IAMResourceLifecycleReplayRequest) (*tenant.IAMResourceOwnershipResult, bool, error) {
	s.replayRequest = req
	return s.replay, s.replayFound, s.replayErr
}
func (*semanticModelOwnershipWriterStub) RegisterIAMResourceOwnership(context.Context, tenant.IAMResourceOwnershipRegister) (*tenant.IAMResourceOwnershipResult, error) {
	return nil, errors.New("unexpected register")
}
func (*semanticModelOwnershipWriterStub) MarkIAMResourceOwnershipDeleting(context.Context, tenant.IAMResourceOwnershipDelete) (*tenant.IAMResourceOwnershipResult, error) {
	return nil, errors.New("unexpected begin delete")
}
func (*semanticModelOwnershipWriterStub) FinalizeIAMResourceOwnershipDelete(context.Context, tenant.IAMResourceOwnershipFinalizeDelete) error {
	return errors.New("unexpected finalize")
}
func (*semanticModelOwnershipWriterStub) GetActiveIAMResourceOwnership(context.Context, string, string, string) (*tenant.IAMResourceOwnershipRecord, error) {
	return nil, errors.New("unexpected owner lookup")
}
func (*semanticModelOwnershipWriterStub) ListActiveIAMPolicyDocuments(context.Context, string) ([]tenant.IAMPolicyDocument, error) {
	return nil, errors.New("unexpected policy list")
}
func (*semanticModelOwnershipWriterStub) SaveValidatedIAMApplicationPolicy(context.Context, tenant.IAMValidatedPolicySave) (*tenant.IAMPolicySaveResult, error) {
	return nil, errors.New("unexpected policy save")
}

func TestSemanticModelRegisterUsesCanonicalLifecycleReplay(t *testing.T) {
	want := &tenant.IAMResourceOwnershipResult{WorkspaceID: "ws-1", ResourceType: IAMResourceSemanticModel, ResourceID: "42", OwnerRoleID: "7", OwnershipVersion: 1}
	registrar := &SemanticModelResourceRegistrar{Authorizer: &IAMCoreAuthorizer{}, Writer: &semanticModelOwnershipWriterStub{replay: want, replayFound: true}}
	got, err := registrar.Register(context.Background(), validSemanticModelRegisterRequest())
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestSemanticModelRegisterFailsClosedOnLifecycleReplayError(t *testing.T) {
	registrar := &SemanticModelResourceRegistrar{Authorizer: &IAMCoreAuthorizer{}, Writer: &semanticModelOwnershipWriterStub{replayErr: errors.New("store unavailable")}}
	_, err := registrar.Register(context.Background(), validSemanticModelRegisterRequest())
	require.ErrorContains(t, err, "store unavailable")
}

func TestSemanticModelLifecycleRejectsNonCanonicalIdentity(t *testing.T) {
	registrar := &SemanticModelResourceRegistrar{Authorizer: &IAMCoreAuthorizer{}, Writer: &semanticModelOwnershipWriterStub{}}
	req := validSemanticModelRegisterRequest()
	req.SemanticModelID = " 42"
	_, err := registrar.Register(context.Background(), req)
	require.ErrorIs(t, err, ErrIAMAdapterInvalidInput)
}

func validSemanticModelRegisterRequest() SemanticModelResourceRegisterRequest {
	return SemanticModelResourceRegisterRequest{WorkspaceID: "ws-1", SemanticModelID: "42", PrincipalID: "user-1", RoleID: "7", OperationID: "op-1", RequestID: "req-1", TraceID: "trace-1"}
}
