package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	ginctx "github.com/matrixflow/moi-core/catalog/pkg/api"
	"github.com/matrixflow/moi-core/catalog/pkg/iamcore"
	"github.com/matrixflow/moi-core/model/common"
	"github.com/matrixflow/moi-core/model/internalservice"
	"github.com/matrixorigin/matrixflow/shared/authz/pkg/core"
)

// knowledgeBaseProvisioningDecision consumes the exact semantic_model.create
// decision established by Backend for this workspace. It is deliberately
// separate from normal Catalog IAM: the default Catalog is a backend-owned
// system resource, while the caller's product permission is semantic_model.create.
func knowledgeBaseProvisioningDecision(c *gin.Context, workspaceID string) (core.Decision, int64, bool) {
	if c == nil || c.GetHeader(internalservice.HeaderKnowledgeBaseProvisioning) != "true" {
		return core.Decision{}, 0, false
	}
	defaultCatalogID, err := strconv.ParseInt(c.GetHeader(internalservice.HeaderKnowledgeBaseProvisioningCatalogID), 10, 64)
	if err != nil || defaultCatalogID <= 0 {
		ginctx.WriteError(c, http.StatusForbidden, common.ErrorCode_PERMISSION_DENIED, "")
		return core.Decision{}, 0, false
	}
	execution, ok := ginctx.GetAuthenticatedBackendExecution(c)
	if !ok || execution.WorkspaceID != workspaceID || !execution.WorkspaceAccessVerified ||
		strings.TrimSpace(execution.VerifiedEffectiveRoleID) == "" ||
		!execution.HasAuthorizedAction(iamcore.IAMActionSemanticModelCreate, iamcore.IAMResourceWorkspace, workspaceID) {
		ginctx.WriteError(c, http.StatusForbidden, common.ErrorCode_PERMISSION_DENIED, "")
		return core.Decision{}, 0, false
	}
	return core.Decision{Allow: true, VerifiedEffectiveRoleID: execution.VerifiedEffectiveRoleID}, defaultCatalogID, true
}
