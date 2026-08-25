package gin

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/matrixflow/moi-core/catalog/pkg/api/backendauth"
	"github.com/matrixflow/moi-core/model/common"
	"github.com/matrixflow/moi-core/model/internalservice"
)

type routeTestHandler struct{}

func (h *routeTestHandler) ok(c *gin.Context) { c.Status(http.StatusNoContent) }

type recordingDataDashboardRouteHandler struct {
	routeTestHandler
	sqlDraftCalls int
}

func (h *recordingDataDashboardRouteHandler) RefreshPlan(c *gin.Context)         { h.ok(c) }
func (h *recordingDataDashboardRouteHandler) BeginDelete(c *gin.Context)         { h.ok(c) }
func (h *recordingDataDashboardRouteHandler) FinalizeDelete(c *gin.Context)      { h.ok(c) }
func (h *recordingDataDashboardRouteHandler) CreateChart(c *gin.Context)         { h.ok(c) }
func (h *recordingDataDashboardRouteHandler) UpdateChart(c *gin.Context)         { h.ok(c) }
func (h *recordingDataDashboardRouteHandler) ScheduleState(c *gin.Context)       { h.ok(c) }
func (h *recordingDataDashboardRouteHandler) BeginChartDelete(c *gin.Context)    { h.ok(c) }
func (h *recordingDataDashboardRouteHandler) FinalizeChartDelete(c *gin.Context) { h.ok(c) }
func (h *recordingDataDashboardRouteHandler) ExecutionSpec(c *gin.Context)       { h.ok(c) }
func (h *recordingDataDashboardRouteHandler) EvaluateAlert(c *gin.Context)       { h.ok(c) }
func (h *recordingDataDashboardRouteHandler) SQLDraft(c *gin.Context) {
	h.sqlDraftCalls++
	c.Status(http.StatusNoContent)
}

func (h *routeTestHandler) Health(c *gin.Context)                         { h.ok(c) }
func (h *routeTestHandler) Ready(c *gin.Context)                          { h.ok(c) }
func (h *routeTestHandler) Live(c *gin.Context)                           { h.ok(c) }
func (h *routeTestHandler) Create(c *gin.Context)                         { h.ok(c) }
func (h *routeTestHandler) List(c *gin.Context)                           { h.ok(c) }
func (h *routeTestHandler) ListTags(c *gin.Context)                       { h.ok(c) }
func (h *routeTestHandler) Tree(c *gin.Context)                           { h.ok(c) }
func (h *routeTestHandler) Delete(c *gin.Context)                         { h.ok(c) }
func (h *routeTestHandler) Get(c *gin.Context)                            { h.ok(c) }
func (h *routeTestHandler) Update(c *gin.Context)                         { h.ok(c) }
func (h *routeTestHandler) GetDBConnection(c *gin.Context)                { h.ok(c) }
func (h *routeTestHandler) GetOwnerDBConnection(c *gin.Context)           { h.ok(c) }
func (h *routeTestHandler) GetSystemRoles(c *gin.Context)                 { h.ok(c) }
func (h *routeTestHandler) GetUserDBConnection(c *gin.Context)            { h.ok(c) }
func (h *routeTestHandler) GetOwnerCredentialAPIKey(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) RevealOwnerCredentialAPIKey(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) RotateOwnerCredentialAPIKey(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) GetOwnerCredentialDBConnection(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) GetByEmail(c *gin.Context)                     { h.ok(c) }
func (h *routeTestHandler) GetByPhone(c *gin.Context)                     { h.ok(c) }
func (h *routeTestHandler) ListDatabases(c *gin.Context)                  { h.ok(c) }
func (h *routeTestHandler) ResolveMetadata(c *gin.Context)                { h.ok(c) }
func (h *routeTestHandler) ResolveDatabaseMetadata(c *gin.Context)        { h.ok(c) }
func (h *routeTestHandler) CompensateCreateIAM(c *gin.Context)            { h.ok(c) }
func (h *routeTestHandler) CompensateTableCreateIAM(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) GetStats(c *gin.Context)                       { h.ok(c) }
func (h *routeTestHandler) SyncMetadata(c *gin.Context)                   { h.ok(c) }
func (h *routeTestHandler) ResolveStructuredLoadTargetDatabase(c *gin.Context) {
	h.ok(c)
}
func (h *routeTestHandler) ResolveStructuredLoadTargetDatabaseRuntime(c *gin.Context) {
	h.ok(c)
}

func (h *routeTestHandler) ResolveDatabaseTables(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) ResolveStructuredLoadTargetTable(c *gin.Context) {
	h.ok(c)
}
func (h *routeTestHandler) ResolveTableMetadata(c *gin.Context)     { h.ok(c) }
func (h *routeTestHandler) ListByWorkspace(c *gin.Context)          { h.ok(c) }
func (h *routeTestHandler) ListTables(c *gin.Context)               { h.ok(c) }
func (h *routeTestHandler) GetTable(c *gin.Context)                 { h.ok(c) }
func (h *routeTestHandler) ListByDatabase(c *gin.Context)           { h.ok(c) }
func (h *routeTestHandler) ListSummaries(c *gin.Context)            { h.ok(c) }
func (h *routeTestHandler) ListChildren(c *gin.Context)             { h.ok(c) }
func (h *routeTestHandler) GetChildren(c *gin.Context)              { h.ok(c) }
func (h *routeTestHandler) GetPath(c *gin.Context)                  { h.ok(c) }
func (h *routeTestHandler) ResolveRoot(c *gin.Context)              { h.ok(c) }
func (h *routeTestHandler) ResolveFileRoots(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) TriggerGarbageCollection(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) Upload(c *gin.Context)                   { h.ok(c) }
func (h *routeTestHandler) UploadPrivateCatalogFile(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) Download(c *gin.Context)                 { h.ok(c) }
func (h *routeTestHandler) Preview(c *gin.Context)                  { h.ok(c) }
func (h *routeTestHandler) DownloadSemanticModelArtifact(c *gin.Context) {
	h.ok(c)
}

func (h *routeTestHandler) GetBuiltinFile(c *gin.Context)      { h.ok(c) }
func (h *routeTestHandler) PublishBuiltinFile(c *gin.Context)  { h.ok(c) }
func (h *routeTestHandler) AttachBuiltinFiles(c *gin.Context)  { h.ok(c) }
func (h *routeTestHandler) AddFiles(c *gin.Context)            { h.ok(c) }
func (h *routeTestHandler) TriggerFiles(c *gin.Context)        { h.ok(c) }
func (h *routeTestHandler) MoveFiles(c *gin.Context)           { h.ok(c) }
func (h *routeTestHandler) RemoveFiles(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) ListFiles(c *gin.Context)           { h.ok(c) }
func (h *routeTestHandler) ListFilesDetail(c *gin.Context)     { h.ok(c) }
func (h *routeTestHandler) ListContents(c *gin.Context)        { h.ok(c) }
func (h *routeTestHandler) CreateRole(c *gin.Context)          { h.ok(c) }
func (h *routeTestHandler) GetRole(c *gin.Context)             { h.ok(c) }
func (h *routeTestHandler) ListRoles(c *gin.Context)           { h.ok(c) }
func (h *routeTestHandler) UpdateRole(c *gin.Context)          { h.ok(c) }
func (h *routeTestHandler) DeleteRole(c *gin.Context)          { h.ok(c) }
func (h *routeTestHandler) GrantPermission(c *gin.Context)     { h.ok(c) }
func (h *routeTestHandler) RevokePermission(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) ListRolePermissions(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) AssignRole(c *gin.Context)          { h.ok(c) }
func (h *routeTestHandler) RevokeUserRole(c *gin.Context)      { h.ok(c) }
func (h *routeTestHandler) ListUserRoles(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) GetMyRoles(c *gin.Context)          { h.ok(c) }
func (h *routeTestHandler) GetMyPermissions(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) GetUserPermissions(c *gin.Context)  { h.ok(c) }
func (h *routeTestHandler) GrantRoleToRole(c *gin.Context)     { h.ok(c) }
func (h *routeTestHandler) RevokeRoleFromRole(c *gin.Context)  { h.ok(c) }
func (h *routeTestHandler) CreateEntry(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) ListEntries(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) UpdateEntry(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) DeleteEntry(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) Import(c *gin.Context)              { h.ok(c) }
func (h *routeTestHandler) Export(c *gin.Context)              { h.ok(c) }
func (h *routeTestHandler) Validate(c *gin.Context)            { h.ok(c) }
func (h *routeTestHandler) AgentCard(c *gin.Context)           { h.ok(c) }
func (h *routeTestHandler) JSONRPC(c *gin.Context)             { h.ok(c) }
func (h *routeTestHandler) PreviewQueryVisual(c *gin.Context)  { h.ok(c) }
func (h *routeTestHandler) QueryVisualContent(c *gin.Context)  { h.ok(c) }
func (h *routeTestHandler) MCPHTTP(c *gin.Context)             { h.ok(c) }
func (h *routeTestHandler) SkillHTTP(c *gin.Context)           { h.ok(c) }
func (h *routeTestHandler) RuntimeFileUpload(c *gin.Context)   { h.ok(c) }

func (h *routeTestHandler) RuntimeExecutorDispatchAuthorize(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) ModelResolve(c *gin.Context)                     { h.ok(c) }
func (h *routeTestHandler) ModelOpenAIChatCompletions(c *gin.Context) {
	h.ok(c)
}
func (h *routeTestHandler) GenericAgentCard(c *gin.Context)     { h.ok(c) }
func (h *routeTestHandler) GenericJSONRPC(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) ListDataParts(c *gin.Context)        { h.ok(c) }
func (h *routeTestHandler) ListTasks(c *gin.Context)            { h.ok(c) }
func (h *routeTestHandler) GetTask(c *gin.Context)              { h.ok(c) }
func (h *routeTestHandler) ListTaskEvents(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) GetManifest(c *gin.Context)          { h.ok(c) }
func (h *routeTestHandler) GetTurnSnapshot(c *gin.Context)      { h.ok(c) }
func (h *routeTestHandler) ListFeedbacks(c *gin.Context)        { h.ok(c) }
func (h *routeTestHandler) GetFeedbackStats(c *gin.Context)     { h.ok(c) }
func (h *routeTestHandler) ListRuntimeProviders(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) GetRuntimeProvider(c *gin.Context)   { h.ok(c) }
func (h *routeTestHandler) LoadPackage(c *gin.Context)          { h.ok(c) }
func (h *routeTestHandler) ExportPackage(c *gin.Context)        { h.ok(c) }
func (h *routeTestHandler) ListVersions(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) SetDefaultVersion(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) DisableVersion(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) DeleteVersion(c *gin.Context)        { h.ok(c) }
func (h *routeTestHandler) UpsertRuntimeBinding(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) ReconcileRuntimeBinding(c *gin.Context) {
	h.ok(c)
}
func (h *routeTestHandler) ListShares(c *gin.Context)          { h.ok(c) }
func (h *routeTestHandler) CreateShare(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) RevokeShare(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) GetBindings(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) UpdateBindings(c *gin.Context)      { h.ok(c) }
func (h *routeTestHandler) UpdateModelBinding(c *gin.Context)  { h.ok(c) }
func (h *routeTestHandler) GetSystemAgentSetup(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) ListSystemAgentGitHubProjects(c *gin.Context) {
	h.ok(c)
}
func (h *routeTestHandler) ListSystemAgentGitHubProjectsForCreate(c *gin.Context) {
	h.ok(c)
}
func (h *routeTestHandler) UpdateSystemAgentSetup(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) GetPolicies(c *gin.Context)            { h.ok(c) }
func (h *routeTestHandler) UpdatePolicies(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) InspectSkillPackage(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) ImportSkillPackage(c *gin.Context)     { h.ok(c) }
func (h *routeTestHandler) CreateSkill(c *gin.Context)            { h.ok(c) }
func (h *routeTestHandler) PolishSkillStream(c *gin.Context)      { h.ok(c) }
func (h *routeTestHandler) ListSkills(c *gin.Context)             { h.ok(c) }
func (h *routeTestHandler) ListSkillTags(c *gin.Context)          { h.ok(c) }
func (h *routeTestHandler) ListReferencingAgents(c *gin.Context)  { h.ok(c) }
func (h *routeTestHandler) GetSkill(c *gin.Context)               { h.ok(c) }
func (h *routeTestHandler) UpdateSkill(c *gin.Context)            { h.ok(c) }
func (h *routeTestHandler) ListSkillFiles(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) GetSkillFile(c *gin.Context)           { h.ok(c) }
func (h *routeTestHandler) ListSkillVersions(c *gin.Context)      { h.ok(c) }
func (h *routeTestHandler) SetCurrentVersion(c *gin.Context)      { h.ok(c) }
func (h *routeTestHandler) ExecuteSkill(c *gin.Context)           { h.ok(c) }
func (h *routeTestHandler) CreateTool(c *gin.Context)             { h.ok(c) }
func (h *routeTestHandler) ConnectGitHub(c *gin.Context)          { h.ok(c) }
func (h *routeTestHandler) GetGitHubConnection(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) DisconnectGitHub(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) GenerateWeComCallbackSecrets(c *gin.Context) {
	h.ok(c)
}
func (h *routeTestHandler) ConnectGrafana(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) GetGrafanaConnection(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) DisconnectGrafana(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) ConnectMail(c *gin.Context)          { h.ok(c) }
func (h *routeTestHandler) GetMailConnection(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) DisconnectMail(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) ListTools(c *gin.Context)            { h.ok(c) }
func (h *routeTestHandler) ListToolTags(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) GetTool(c *gin.Context)              { h.ok(c) }
func (h *routeTestHandler) UpdateTool(c *gin.Context)           { h.ok(c) }
func (h *routeTestHandler) CreateKnowledgeBase(c *gin.Context)  { h.ok(c) }
func (h *routeTestHandler) ListKnowledgeBases(c *gin.Context)   { h.ok(c) }
func (h *routeTestHandler) GetKnowledgeBase(c *gin.Context)     { h.ok(c) }
func (h *routeTestHandler) UpdateKnowledgeBase(c *gin.Context)  { h.ok(c) }
func (h *routeTestHandler) CreateModelConfig(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) ListModelConfigs(c *gin.Context)     { h.ok(c) }
func (h *routeTestHandler) GetModelConfig(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) UpdateModelConfig(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) CreateConnection(c *gin.Context)     { h.ok(c) }
func (h *routeTestHandler) ListConnections(c *gin.Context)      { h.ok(c) }
func (h *routeTestHandler) GetConnection(c *gin.Context)        { h.ok(c) }
func (h *routeTestHandler) UpdateConnection(c *gin.Context)     { h.ok(c) }
func (h *routeTestHandler) ProbeMCPConnection(c *gin.Context)   { h.ok(c) }
func (h *routeTestHandler) BatchCreateMCPTools(c *gin.Context)  { h.ok(c) }
func (h *routeTestHandler) CreateInstance(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) TestConfiguration(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) ListInstances(c *gin.Context)        { h.ok(c) }
func (h *routeTestHandler) GetInstance(c *gin.Context)          { h.ok(c) }
func (h *routeTestHandler) UpdateInstance(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) DeleteInstance(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) TestInstance(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) CreateRuntimePolicyProfile(c *gin.Context) {
	h.ok(c)
}
func (h *routeTestHandler) ListRuntimePolicyProfiles(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) GetRuntimePolicyProfile(c *gin.Context)   { h.ok(c) }
func (h *routeTestHandler) UpdateRuntimePolicyProfile(c *gin.Context) {
	h.ok(c)
}
func (h *routeTestHandler) ListOperations(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) GetOperation(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) CancelOperation(c *gin.Context)      { h.ok(c) }
func (h *routeTestHandler) CreateConversation(c *gin.Context)   { h.ok(c) }
func (h *routeTestHandler) ListConversations(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) GetConversation(c *gin.Context)      { h.ok(c) }
func (h *routeTestHandler) UpdateConversation(c *gin.Context)   { h.ok(c) }
func (h *routeTestHandler) ListMessages(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) CreateAutomationTask(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) ListAutomationTasks(c *gin.Context)  { h.ok(c) }
func (h *routeTestHandler) GetAutomationTask(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) UpdateAutomationTask(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) DeleteAutomationTask(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) ListAutomationRuns(c *gin.Context)   { h.ok(c) }
func (h *routeTestHandler) RunAutomationTaskNow(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) ListWorkspaceAutomationRuns(c *gin.Context) {
	h.ok(c)
}
func (h *routeTestHandler) GetAutomationRun(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) GetAutomationRunResult(c *gin.Context) {
	h.ok(c)
}
func (h *routeTestHandler) ListAutomationRunEvents(c *gin.Context) { h.ok(c) }
func (h *routeTestHandler) CreateTaskTemplate(c *gin.Context)      { h.ok(c) }
func (h *routeTestHandler) ListTaskTemplates(c *gin.Context)       { h.ok(c) }
func (h *routeTestHandler) GetTaskTemplate(c *gin.Context)         { h.ok(c) }
func (h *routeTestHandler) UpdateTaskTemplate(c *gin.Context)      { h.ok(c) }
func (h *routeTestHandler) CreateWorkflowBinding(c *gin.Context)   { h.ok(c) }
func (h *routeTestHandler) ListWorkflowBindings(c *gin.Context)    { h.ok(c) }
func (h *routeTestHandler) GetWorkflowBinding(c *gin.Context)      { h.ok(c) }
func (h *routeTestHandler) UpdateWorkflowBinding(c *gin.Context)   { h.ok(c) }

type recordingAgentRuntimeHandler struct {
	routeTestHandler
	cardAgentID          string
	cardAgentWorkspaceID string
	a2aBody              map[string]any
}

func (h *recordingAgentRuntimeHandler) GenericAgentCard(c *gin.Context) {
	h.cardAgentID = c.Query("agent_id")
	h.cardAgentWorkspaceID = c.Query("agent_workspace_id")
	c.Status(http.StatusNoContent)
}

func (h *recordingAgentRuntimeHandler) GenericJSONRPC(c *gin.Context) {
	body, _ := io.ReadAll(c.Request.Body)
	_ = json.Unmarshal(body, &h.a2aBody)
	c.Status(http.StatusNoContent)
}

func passthroughMiddleware(c *gin.Context) { c.Next() }

func newRouterForSemanticRouteTest(t *testing.T, semanticHandler SemanticModelHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:        h,
		APIKeyHandler:        h,
		WorkspaceHandler:     h,
		UserHandler:          h,
		CatalogHandler:       h,
		DatabaseHandler:      h,
		VolumeHandler:        h,
		GarbageHandler:       h,
		FileHandler:          h,
		VolumeFileHandler:    h,
		VolumeContentHandler: h,
		SemanticModelHandler: semanticHandler,
		HandlerWrapper:       wrapper,
		TraceIDMiddleware:    passthroughMiddleware,
		LoggingMiddleware:    passthroughMiddleware,
		MetricsMiddleware:    passthroughMiddleware,
		APIKeyMiddlewareFunc: func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:               zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func newRouterForDataDashboardSQLDraftRouteTest(t *testing.T, handler DataDashboardHandler, authorized bool) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)
	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:        h,
		APIKeyHandler:        h,
		WorkspaceHandler:     h,
		UserHandler:          h,
		CatalogHandler:       h,
		DatabaseHandler:      h,
		VolumeHandler:        h,
		GarbageHandler:       h,
		FileHandler:          h,
		VolumeFileHandler:    h,
		VolumeContentHandler: h,
		DataDashboardHandler: handler,
		HandlerWrapper:       wrapper,
		TraceIDMiddleware:    passthroughMiddleware,
		LoggingMiddleware:    passthroughMiddleware,
		MetricsMiddleware:    passthroughMiddleware,
		APIKeyMiddlewareFunc: func(_ []string) gin.HandlerFunc {
			return func(c *gin.Context) {
				if authorized {
					SetAuthenticatedBackendExecution(c, BackendExecutionContext{
						WorkspaceID:              c.Param("id"),
						VerifiedEffectiveRoleID:  "role-7",
						WorkspaceAccessVerified:  true,
						BusinessActionAuthorized: true,
					})
				}
				c.Next()
			}
		},
		Logger: zap.NewNop(),
	})
	require.NoError(t, err)
	router.SetupRoutes()
	return router
}

func TestDataDashboardSQLDraftRouteRequiresCurrentCoreAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &recordingDataDashboardRouteHandler{}
	path := "/api/v1/workspaces/workspace-1/data-dashboards/dashboard-1/sql-draft"

	unauthorized := newRouterForDataDashboardSQLDraftRouteTest(t, handler, false)
	response := httptest.NewRecorder()
	unauthorized.Engine().ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
	require.Equal(t, http.StatusForbidden, response.Code)
	require.Zero(t, handler.sqlDraftCalls, "route guard must reject before SQL draft handler invocation")

	authorized := newRouterForDataDashboardSQLDraftRouteTest(t, handler, true)
	response = httptest.NewRecorder()
	authorized.Engine().ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, 1, handler.sqlDraftCalls)
}

func newRouterForAgentResourceRouteTest(t *testing.T, agentResourceHandler AgentResourceHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:        h,
		APIKeyHandler:        h,
		WorkspaceHandler:     h,
		UserHandler:          h,
		CatalogHandler:       h,
		DatabaseHandler:      h,
		VolumeHandler:        h,
		GarbageHandler:       h,
		FileHandler:          h,
		VolumeFileHandler:    h,
		VolumeContentHandler: h,
		AgentResourceHandler: agentResourceHandler,
		HandlerWrapper:       wrapper,
		TraceIDMiddleware:    passthroughMiddleware,
		LoggingMiddleware:    passthroughMiddleware,
		MetricsMiddleware:    passthroughMiddleware,
		APIKeyMiddlewareFunc: func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:               zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func routeExists(routes []gin.RouteInfo, method, path string) bool {
	for _, route := range routes {
		if route.Method == method && route.Path == path {
			return true
		}
	}
	return false
}

func performRouteRequest(engine *gin.Engine, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	engine.ServeHTTP(rec, req)
	return rec
}

func TestSetupRoutes_WorkspaceOwnerCredentialRoutes(t *testing.T) {
	h := &routeTestHandler{}
	router := newRouterForSemanticRouteTest(t, h)
	routes := router.Engine().Routes()

	expected := []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/users/:user_id/owner-credential/api-key"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/users/:user_id/owner-credential/api-key"},
		{method: http.MethodPut, path: "/api/v1/workspaces/:id/users/:user_id/owner-credential/api-key"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/users/:user_id/owner-credential/db-connection"},
	}
	for _, route := range expected {
		assert.True(t, routeExists(routes, route.method, route.path), "%s %s should be registered", route.method, route.path)
	}
}

func TestAgentRuntimeProviderGatewayRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := newRouterForAgentRuntimeRouteTest(t, &routeTestHandler{})
	routes := router.Engine().Routes()
	expectedRoutes := []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/models/resolve"},
		{method: http.MethodPost, path: "/api/v1/models/openai/chat/completions"},
		{method: http.MethodPost, path: "/api/v1/mcp/http"},
		{method: http.MethodPost, path: "/api/v1/skills/http"},
	}

	for _, expected := range expectedRoutes {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
		rec := performRouteRequest(router.Engine(), expected.method, expected.path)
		assert.Equal(t, http.StatusNoContent, rec.Code, "%s %s should reach agent runtime handler", expected.method, expected.path)
	}
}

func expectedSemanticModelRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/semantic-models"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/semantic-models"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/semantic-models/tags"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/semantic-models/import"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/semantic-models/:model_id"},
		{method: http.MethodPut, path: "/api/v1/workspaces/:id/semantic-models/:model_id"},
		{method: http.MethodDelete, path: "/api/v1/workspaces/:id/semantic-models/:model_id"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/semantic-models/:model_id/export"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/semantic-models/:model_id/validate"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/semantic-models/:model_id/entries"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/semantic-models/:model_id/entries"},
		{method: http.MethodPut, path: "/api/v1/workspaces/:id/semantic-models/:model_id/entries/:entry_id"},
		{method: http.MethodDelete, path: "/api/v1/workspaces/:id/semantic-models/:model_id/entries/:entry_id"},
	}
}

func TestSetupRoutes_DoesNotRegisterDirectWorkspaceMembershipMutationRoutes(t *testing.T) {
	router := newRouterForSemanticRouteTest(t, &routeTestHandler{})
	routes := router.Engine().Routes()

	for _, legacyRoute := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/users"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/users"},
		{method: http.MethodDelete, path: "/api/v1/workspaces/:id/users/:user_id"},
	} {
		assert.False(t, routeExists(routes, legacyRoute.method, legacyRoute.path), "%s %s must not be registered", legacyRoute.method, legacyRoute.path)
	}
}

func TestSetupRoutes_RegistersSemanticModelRoutesWhenHandlerPresent(t *testing.T) {
	semanticHandler := &routeTestHandler{}
	router := newRouterForSemanticRouteTest(t, semanticHandler)

	routes := router.Engine().Routes()
	for _, expected := range expectedSemanticModelRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterSemanticModelRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForSemanticRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedSemanticModelRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_SemanticModelRoutesReachConfiguredHandler(t *testing.T) {
	router := newRouterForSemanticRouteTest(t, &routeTestHandler{})
	replacer := strings.NewReplacer(":id", "ws_1", ":model_id", "1", ":entry_id", "1")

	for _, route := range expectedSemanticModelRoutes() {
		rec := performRouteRequest(router.Engine(), route.method, replacer.Replace(route.path))
		assert.Equal(t, http.StatusNoContent, rec.Code, "%s %s should reach the configured handler", route.method, route.path)
	}
}

func expectedAgentResourceRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/agents"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agents"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agents/:agent_id"},
		{method: http.MethodPatch, path: "/api/v1/workspaces/:id/agents/:agent_id"},
		{method: http.MethodDelete, path: "/api/v1/workspaces/:id/agents/:agent_id"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agents/:agent_id/bindings"},
		{method: http.MethodPatch, path: "/api/v1/workspaces/:id/agents/:agent_id/bindings"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agents/:agent_id/policies"},
		{method: http.MethodPut, path: "/api/v1/workspaces/:id/agents/:agent_id/policies"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-runtime-providers"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-runtime-providers/:provider_id/profiles/:profile_id"},
	}
}

func TestSetupRoutes_RegistersAgentResourceRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentResourceRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentResourceRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentResourceRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentResourceRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentResourceRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func newRouterForAgentSkillRouteTest(t *testing.T, agentSkillHandler AgentSkillHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:        h,
		APIKeyHandler:        h,
		WorkspaceHandler:     h,
		UserHandler:          h,
		CatalogHandler:       h,
		DatabaseHandler:      h,
		VolumeHandler:        h,
		GarbageHandler:       h,
		FileHandler:          h,
		VolumeFileHandler:    h,
		VolumeContentHandler: h,
		AgentSkillHandler:    agentSkillHandler,
		HandlerWrapper:       wrapper,
		TraceIDMiddleware:    passthroughMiddleware,
		LoggingMiddleware:    passthroughMiddleware,
		MetricsMiddleware:    passthroughMiddleware,
		APIKeyMiddlewareFunc: func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:               zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func newRouterForAgentSkillRoutePermissionTest(t *testing.T, authorized bool) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:              h,
		APIKeyHandler:              h,
		WorkspaceHandler:           h,
		UserHandler:                h,
		CatalogHandler:             h,
		DatabaseHandler:            h,
		VolumeHandler:              h,
		GarbageHandler:             h,
		FileHandler:                h,
		VolumeFileHandler:          h,
		VolumeContentHandler:       h,
		AgentSkillHandler:          h,
		AgentToolHandler:           h,
		AgentAutomationTaskHandler: h,
		HandlerWrapper:             wrapper,
		TraceIDMiddleware:          passthroughMiddleware,
		LoggingMiddleware:          passthroughMiddleware,
		MetricsMiddleware:          passthroughMiddleware,
		APIKeyMiddlewareFunc: func(_ []string) gin.HandlerFunc {
			return func(c *gin.Context) {
				if authorized {
					SetAuthenticatedBackendExecution(c, BackendExecutionContext{
						WorkspaceID: c.Param("id"), VerifiedEffectiveRoleID: "7", WorkspaceAccessVerified: true,
					})
				}
				c.Next()
			}
		},
		Logger: zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentSkillRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/skills/import/inspect"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/skills/import"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/skills"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/skills"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/skills/tags"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/skills/:skill_id/referencing-agents"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/skills/:skill_id"},
		{method: http.MethodPatch, path: "/api/v1/workspaces/:id/skills/:skill_id"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/skills/:skill_id/files"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/skills/:skill_id/files/content"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/skills/:skill_id/versions"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/skills/:skill_id/versions/:version/current"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/skills/:skill_id/execute"},
	}
}

func TestSetupRoutes_RegistersAgentSkillRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentSkillRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentSkillRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentSkillRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentSkillRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentSkillRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_AgentSkillRoutesRequireCurrentVerifiedEffectiveRole(t *testing.T) {
	router := newRouterForAgentSkillRoutePermissionTest(t, false)

	rec := performRouteRequest(router.Engine(), http.MethodPost, "/api/v1/workspaces/ws_1/skills/sk_1/versions/1/current")
	assert.Equal(t, http.StatusForbidden, rec.Code)

	router = newRouterForAgentSkillRoutePermissionTest(t, true)
	rec = performRouteRequest(router.Engine(), http.MethodPost, "/api/v1/workspaces/ws_1/skills/sk_1/versions/1/current")
	assert.Equal(t, http.StatusNoContent, rec.Code)

	rec = performRouteRequest(router.Engine(), http.MethodPatch, "/api/v1/workspaces/ws_1/skills/sk_1")
	assert.Equal(t, http.StatusNoContent, rec.Code)

	rec = performRouteRequest(router.Engine(), http.MethodPost, "/api/v1/workspaces/ws_1/skills/sk_1/execute")
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestSetupRoutes_AgentMetadataReadRequiresVerifiedEffectiveRole(t *testing.T) {
	paths := []string{
		"/api/v1/workspaces/ws_1/skills/sk_1/referencing-agents",
		"/api/v1/workspaces/ws_1/tools/tool_1/referencing-agents",
		"/api/v1/workspaces/ws_1/agent-automation-tasks",
		"/api/v1/workspaces/ws_1/agent-automation-tasks/task_1",
	}

	router := newRouterForAgentSkillRoutePermissionTest(t, false)
	for _, path := range paths {
		rec := performRouteRequest(router.Engine(), http.MethodGet, path)
		assert.Equal(t, http.StatusForbidden, rec.Code, "GET %s should require a verified effective role", path)
	}

	router = newRouterForAgentSkillRoutePermissionTest(t, true)
	for _, path := range paths {
		rec := performRouteRequest(router.Engine(), http.MethodGet, path)
		assert.Equal(t, http.StatusNoContent, rec.Code, "GET %s should allow a verified effective role", path)
	}
}

func newRouterForAgentToolRouteTest(t *testing.T, agentToolHandler AgentToolHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:        h,
		APIKeyHandler:        h,
		WorkspaceHandler:     h,
		UserHandler:          h,
		CatalogHandler:       h,
		DatabaseHandler:      h,
		VolumeHandler:        h,
		GarbageHandler:       h,
		FileHandler:          h,
		VolumeFileHandler:    h,
		VolumeContentHandler: h,
		AgentToolHandler:     agentToolHandler,
		HandlerWrapper:       wrapper,
		TraceIDMiddleware:    passthroughMiddleware,
		LoggingMiddleware:    passthroughMiddleware,
		MetricsMiddleware:    passthroughMiddleware,
		APIKeyMiddlewareFunc: func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:               zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentToolRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/tools"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/tools/github/connect"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/tools/github/connect"},
		{method: http.MethodDelete, path: "/api/v1/workspaces/:id/tools/github/connect"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/tools/wecom/callback-secrets/generate"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/tools/grafana/connect"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/tools/grafana/connect"},
		{method: http.MethodDelete, path: "/api/v1/workspaces/:id/tools/grafana/connect"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/tools/mail/:provider/connect"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/tools/mail/:provider/connect"},
		{method: http.MethodDelete, path: "/api/v1/workspaces/:id/tools/mail/:provider/connect"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/tools"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/tools/tags"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/tools/:tool_id/referencing-agents"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/tools/:tool_id"},
		{method: http.MethodPatch, path: "/api/v1/workspaces/:id/tools/:tool_id"},
	}
}

func TestSetupRoutes_RegistersAgentToolRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentToolRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentToolRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentToolRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentToolRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentToolRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func newRouterForAgentKnowledgeBaseRouteTest(t *testing.T, agentKnowledgeBaseHandler AgentKnowledgeBaseHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:             h,
		APIKeyHandler:             h,
		WorkspaceHandler:          h,
		UserHandler:               h,
		CatalogHandler:            h,
		DatabaseHandler:           h,
		VolumeHandler:             h,
		GarbageHandler:            h,
		FileHandler:               h,
		VolumeFileHandler:         h,
		VolumeContentHandler:      h,
		AgentKnowledgeBaseHandler: agentKnowledgeBaseHandler,
		HandlerWrapper:            wrapper,
		TraceIDMiddleware:         passthroughMiddleware,
		LoggingMiddleware:         passthroughMiddleware,
		MetricsMiddleware:         passthroughMiddleware,
		APIKeyMiddlewareFunc:      func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:                    zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func newRouterForAgentKnowledgeBaseRoutePermissionTest(t *testing.T, authorized bool) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:             h,
		APIKeyHandler:             h,
		WorkspaceHandler:          h,
		UserHandler:               h,
		CatalogHandler:            h,
		DatabaseHandler:           h,
		VolumeHandler:             h,
		GarbageHandler:            h,
		FileHandler:               h,
		VolumeFileHandler:         h,
		VolumeContentHandler:      h,
		AgentKnowledgeBaseHandler: h,
		HandlerWrapper:            wrapper,
		TraceIDMiddleware:         passthroughMiddleware,
		LoggingMiddleware:         passthroughMiddleware,
		MetricsMiddleware:         passthroughMiddleware,
		APIKeyMiddlewareFunc: func(_ []string) gin.HandlerFunc {
			return func(c *gin.Context) {
				if authorized {
					SetAuthenticatedBackendExecution(c, BackendExecutionContext{
						WorkspaceID: c.Param("id"), VerifiedEffectiveRoleID: "7", WorkspaceAccessVerified: true,
						BusinessActionAuthorized: true,
					})
				}
				c.Next()
			}
		},
		Logger: zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentKnowledgeBaseRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/knowledge-bases"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/knowledge-bases"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/knowledge-bases/:knowledge_base_id"},
		{method: http.MethodPatch, path: "/api/v1/workspaces/:id/knowledge-bases/:knowledge_base_id"},
	}
}

func TestSetupRoutes_RegistersAgentKnowledgeBaseRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentKnowledgeBaseRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentKnowledgeBaseRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentKnowledgeBaseRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentKnowledgeBaseRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentKnowledgeBaseRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_AgentKnowledgeBaseRoutesRequireCurrentBackendAuthorization(t *testing.T) {
	router := newRouterForAgentKnowledgeBaseRoutePermissionTest(t, false)

	rec := performRouteRequest(router.Engine(), http.MethodGet, "/api/v1/workspaces/ws_1/knowledge-bases")
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = performRouteRequest(router.Engine(), http.MethodPost, "/api/v1/workspaces/ws_1/knowledge-bases")
	assert.Equal(t, http.StatusForbidden, rec.Code)

	rec = performRouteRequest(router.Engine(), http.MethodPatch, "/api/v1/workspaces/ws_1/knowledge-bases/kb_1")
	assert.Equal(t, http.StatusForbidden, rec.Code)

	router = newRouterForAgentKnowledgeBaseRoutePermissionTest(t, true)

	rec = performRouteRequest(router.Engine(), http.MethodGet, "/api/v1/workspaces/ws_1/knowledge-bases")
	assert.Equal(t, http.StatusNoContent, rec.Code)

	rec = performRouteRequest(router.Engine(), http.MethodPost, "/api/v1/workspaces/ws_1/knowledge-bases")
	assert.Equal(t, http.StatusNoContent, rec.Code)

	rec = performRouteRequest(router.Engine(), http.MethodPatch, "/api/v1/workspaces/ws_1/knowledge-bases/kb_1")
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func newRouterForAgentModelConfigRouteTest(t *testing.T, agentModelConfigHandler AgentModelConfigHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:           h,
		APIKeyHandler:           h,
		WorkspaceHandler:        h,
		UserHandler:             h,
		CatalogHandler:          h,
		DatabaseHandler:         h,
		VolumeHandler:           h,
		GarbageHandler:          h,
		FileHandler:             h,
		VolumeFileHandler:       h,
		VolumeContentHandler:    h,
		AgentModelConfigHandler: agentModelConfigHandler,
		HandlerWrapper:          wrapper,
		TraceIDMiddleware:       passthroughMiddleware,
		LoggingMiddleware:       passthroughMiddleware,
		MetricsMiddleware:       passthroughMiddleware,
		APIKeyMiddlewareFunc:    func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:                  zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentModelConfigRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/model-configs"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/model-configs"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/model-configs/:model_config_id"},
		{method: http.MethodPatch, path: "/api/v1/workspaces/:id/model-configs/:model_config_id"},
	}
}

func TestSetupRoutes_RegistersAgentModelConfigRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentModelConfigRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentModelConfigRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentModelConfigRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentModelConfigRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentModelConfigRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func newRouterForAgentConnectionRouteTest(t *testing.T, agentConnectionHandler AgentConnectionHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:          h,
		APIKeyHandler:          h,
		WorkspaceHandler:       h,
		UserHandler:            h,
		CatalogHandler:         h,
		DatabaseHandler:        h,
		VolumeHandler:          h,
		GarbageHandler:         h,
		FileHandler:            h,
		VolumeFileHandler:      h,
		VolumeContentHandler:   h,
		AgentConnectionHandler: agentConnectionHandler,
		HandlerWrapper:         wrapper,
		TraceIDMiddleware:      passthroughMiddleware,
		LoggingMiddleware:      passthroughMiddleware,
		MetricsMiddleware:      passthroughMiddleware,
		APIKeyMiddlewareFunc:   func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:                 zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentConnectionRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/connections"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/connections"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/connections/:connection_id"},
		{method: http.MethodPatch, path: "/api/v1/workspaces/:id/connections/:connection_id"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/connections/actions/probe-mcp"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/connections/actions/batch-create-mcp-tools"},
	}
}

func TestSetupRoutes_RegistersAgentConnectionRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentConnectionRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentConnectionRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentConnectionRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentConnectionRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentConnectionRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func newRouterForChannelInstanceRouteTest(t *testing.T, channelInstanceHandler ChannelInstanceHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:          h,
		APIKeyHandler:          h,
		WorkspaceHandler:       h,
		UserHandler:            h,
		CatalogHandler:         h,
		DatabaseHandler:        h,
		VolumeHandler:          h,
		GarbageHandler:         h,
		FileHandler:            h,
		VolumeFileHandler:      h,
		VolumeContentHandler:   h,
		ChannelInstanceHandler: channelInstanceHandler,
		HandlerWrapper:         wrapper,
		TraceIDMiddleware:      passthroughMiddleware,
		LoggingMiddleware:      passthroughMiddleware,
		MetricsMiddleware:      passthroughMiddleware,
		APIKeyMiddlewareFunc:   func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:                 zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedChannelInstanceRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/channels/:provider/instances"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/channels/:provider/instances/test"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/channels/:provider/instances"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/channels/:provider/instances/:instance_id"},
		{method: http.MethodPatch, path: "/api/v1/workspaces/:id/channels/:provider/instances/:instance_id"},
		{method: http.MethodDelete, path: "/api/v1/workspaces/:id/channels/:provider/instances/:instance_id"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/channels/:provider/instances/:instance_id/test"},
	}
}

func TestSetupRoutes_RegistersChannelInstanceRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForChannelInstanceRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedChannelInstanceRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_WebSearchCredentialListAcceptsRoleCandidateForCoreFilter(t *testing.T) {
	router := newRouterForChannelInstanceRouteTest(t, &routeTestHandler{})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws-1/channels/bocha/instances", nil)
	request.Header.Set(internalservice.HeaderRoleCandidateID, "7")
	response := httptest.NewRecorder()

	router.Engine().ServeHTTP(response, request)

	require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
}

func TestSetupRoutes_DoesNotRegisterChannelInstanceRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForChannelInstanceRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedChannelInstanceRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func newRouterForAgentRuntimePolicyRouteTest(t *testing.T, agentRuntimePolicyHandler AgentRuntimePolicyProfileHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:             h,
		APIKeyHandler:             h,
		WorkspaceHandler:          h,
		UserHandler:               h,
		CatalogHandler:            h,
		DatabaseHandler:           h,
		VolumeHandler:             h,
		GarbageHandler:            h,
		FileHandler:               h,
		VolumeFileHandler:         h,
		VolumeContentHandler:      h,
		AgentRuntimePolicyHandler: agentRuntimePolicyHandler,
		HandlerWrapper:            wrapper,
		TraceIDMiddleware:         passthroughMiddleware,
		LoggingMiddleware:         passthroughMiddleware,
		MetricsMiddleware:         passthroughMiddleware,
		APIKeyMiddlewareFunc:      func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:                    zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentRuntimePolicyRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/runtime-policy-profiles"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/runtime-policy-profiles"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/runtime-policy-profiles/:policy_id"},
		{method: http.MethodPatch, path: "/api/v1/workspaces/:id/runtime-policy-profiles/:policy_id"},
	}
}

func TestSetupRoutes_RegistersAgentRuntimePolicyRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentRuntimePolicyRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentRuntimePolicyRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentRuntimePolicyRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentRuntimePolicyRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentRuntimePolicyRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func newRouterForAgentOperationRouteTest(t *testing.T, agentOperationHandler AgentOperationHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:         h,
		APIKeyHandler:         h,
		WorkspaceHandler:      h,
		UserHandler:           h,
		CatalogHandler:        h,
		DatabaseHandler:       h,
		VolumeHandler:         h,
		GarbageHandler:        h,
		FileHandler:           h,
		VolumeFileHandler:     h,
		VolumeContentHandler:  h,
		AgentOperationHandler: agentOperationHandler,
		HandlerWrapper:        wrapper,
		TraceIDMiddleware:     passthroughMiddleware,
		LoggingMiddleware:     passthroughMiddleware,
		MetricsMiddleware:     passthroughMiddleware,
		APIKeyMiddlewareFunc:  func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:                zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentOperationRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/operations"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/operations/:operation_id"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/operations/:operation_id/cancel"},
	}
}

func TestSetupRoutes_RegistersAgentOperationRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentOperationRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentOperationRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentOperationRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentOperationRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentOperationRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func newRouterForAgentConversationRouteTest(t *testing.T, agentConversationHandler AgentConversationHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:            h,
		APIKeyHandler:            h,
		WorkspaceHandler:         h,
		UserHandler:              h,
		CatalogHandler:           h,
		DatabaseHandler:          h,
		VolumeHandler:            h,
		GarbageHandler:           h,
		FileHandler:              h,
		VolumeFileHandler:        h,
		VolumeContentHandler:     h,
		AgentConversationHandler: agentConversationHandler,
		HandlerWrapper:           wrapper,
		TraceIDMiddleware:        passthroughMiddleware,
		LoggingMiddleware:        passthroughMiddleware,
		MetricsMiddleware:        passthroughMiddleware,
		APIKeyMiddlewareFunc:     func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:                   zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentConversationRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/conversations"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/conversations"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/conversations/:conversation_id"},
		{method: http.MethodPatch, path: "/api/v1/workspaces/:id/conversations/:conversation_id"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/conversations/:conversation_id/messages"},
	}
}

func TestSetupRoutes_RegistersAgentConversationRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentConversationRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentConversationRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentConversationRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentConversationRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentConversationRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func newRouterForAgentAutomationTaskRouteTest(t *testing.T, handler AgentAutomationTaskHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:              h,
		APIKeyHandler:              h,
		WorkspaceHandler:           h,
		UserHandler:                h,
		CatalogHandler:             h,
		DatabaseHandler:            h,
		VolumeHandler:              h,
		GarbageHandler:             h,
		FileHandler:                h,
		VolumeFileHandler:          h,
		VolumeContentHandler:       h,
		AgentAutomationTaskHandler: handler,
		HandlerWrapper:             wrapper,
		TraceIDMiddleware:          passthroughMiddleware,
		LoggingMiddleware:          passthroughMiddleware,
		MetricsMiddleware:          passthroughMiddleware,
		APIKeyMiddlewareFunc:       func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:                     zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentAutomationTaskRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/agent-automation-tasks"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-automation-tasks"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-automation-tasks/:automation_task_id"},
		{method: http.MethodPatch, path: "/api/v1/workspaces/:id/agent-automation-tasks/:automation_task_id"},
		{method: http.MethodDelete, path: "/api/v1/workspaces/:id/agent-automation-tasks/:automation_task_id"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-automation-tasks/:automation_task_id/runs"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/agent-automation-tasks/:automation_task_id/runs"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-automation-runs"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-automation-runs/:run_id"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-automation-runs/:run_id/result"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-automation-runs/:run_id/events"},
	}
}

func TestSetupRoutes_RegistersAgentAutomationTaskRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentAutomationTaskRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentAutomationTaskRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
	assert.False(t,
		routeExists(routes, http.MethodPost, "/api/v1/workspaces/:id/agent-automation-tasks/:automation_task_id/invoke"),
		"agent automation tasks must not expose a dedicated invoke route; use dynamic-services/invoke",
	)
	assert.False(t,
		routeExists(routes, http.MethodPost, "/api/v1/workspaces/:id/agent-automation-runs/:run_id/retry"),
		"agent automation task center must not expose retry as a product API in this scope",
	)
	assert.False(t,
		routeExists(routes, http.MethodPost, "/api/v1/workspaces/:id/agent-automation-runs/:run_id/cancel"),
		"agent automation task center must not expose cancel as a product API in this scope",
	)
	assert.False(t,
		routeExists(routes, http.MethodPost, "/api/v1/workspaces/:id/agent-automation-runs/:run_id/approvals/:approval_id/resolve"),
		"agent automation task center must not expose approval resolution as a product API in this scope",
	)
}

func TestSetupRoutes_DoesNotRegisterAgentAutomationTaskRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentAutomationTaskRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentAutomationTaskRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func newRouterForAgentTaskTemplateRouteTest(t *testing.T, agentTaskTemplateHandler AgentTaskTemplateHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:            h,
		APIKeyHandler:            h,
		WorkspaceHandler:         h,
		UserHandler:              h,
		CatalogHandler:           h,
		DatabaseHandler:          h,
		VolumeHandler:            h,
		GarbageHandler:           h,
		FileHandler:              h,
		VolumeFileHandler:        h,
		VolumeContentHandler:     h,
		AgentTaskTemplateHandler: agentTaskTemplateHandler,
		HandlerWrapper:           wrapper,
		TraceIDMiddleware:        passthroughMiddleware,
		LoggingMiddleware:        passthroughMiddleware,
		MetricsMiddleware:        passthroughMiddleware,
		APIKeyMiddlewareFunc:     func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:                   zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentTaskTemplateRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/agent-task-templates"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-task-templates"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-task-templates/:template_id"},
		{method: http.MethodPatch, path: "/api/v1/workspaces/:id/agent-task-templates/:template_id"},
	}
}

func TestSetupRoutes_RegistersAgentTaskTemplateRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentTaskTemplateRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentTaskTemplateRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentTaskTemplateRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentTaskTemplateRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentTaskTemplateRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func newRouterForAgentWorkflowBindingRouteTest(t *testing.T, agentWorkflowBindingHandler AgentWorkflowBindingHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:               h,
		APIKeyHandler:               h,
		WorkspaceHandler:            h,
		UserHandler:                 h,
		CatalogHandler:              h,
		DatabaseHandler:             h,
		VolumeHandler:               h,
		GarbageHandler:              h,
		FileHandler:                 h,
		VolumeFileHandler:           h,
		VolumeContentHandler:        h,
		AgentWorkflowBindingHandler: agentWorkflowBindingHandler,
		HandlerWrapper:              wrapper,
		TraceIDMiddleware:           passthroughMiddleware,
		LoggingMiddleware:           passthroughMiddleware,
		MetricsMiddleware:           passthroughMiddleware,
		APIKeyMiddlewareFunc:        func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:                      zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentWorkflowBindingRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/agent-workflow-bindings"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-workflow-bindings"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-workflow-bindings/:binding_id"},
		{method: http.MethodPatch, path: "/api/v1/workspaces/:id/agent-workflow-bindings/:binding_id"},
	}
}

func TestSetupRoutes_RegistersAgentWorkflowBindingRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentWorkflowBindingRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentWorkflowBindingRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentWorkflowBindingRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentWorkflowBindingRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentWorkflowBindingRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func newRouterForAgentRuntimeReadRouteTest(t *testing.T, agentRuntimeReadHandler AgentRuntimeReadModelHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:                h,
		APIKeyHandler:                h,
		WorkspaceHandler:             h,
		UserHandler:                  h,
		CatalogHandler:               h,
		DatabaseHandler:              h,
		VolumeHandler:                h,
		GarbageHandler:               h,
		FileHandler:                  h,
		VolumeFileHandler:            h,
		VolumeContentHandler:         h,
		AgentRuntimeReadModelHandler: agentRuntimeReadHandler,
		HandlerWrapper:               wrapper,
		TraceIDMiddleware:            passthroughMiddleware,
		LoggingMiddleware:            passthroughMiddleware,
		MetricsMiddleware:            passthroughMiddleware,
		APIKeyMiddlewareFunc:         func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:                       zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentRuntimeReadRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-runtime/data-parts"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-runtime-tasks"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-runtime-tasks/:task_id"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-runtime-tasks/:task_id/events"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-runtime-manifests/:manifest_id"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agent-runtime-turn-snapshots/:snapshot_id"},
	}
}

func TestSetupRoutes_RegistersAgentRuntimeReadRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentRuntimeReadRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentRuntimeReadRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentRuntimeReadRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentRuntimeReadRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentRuntimeReadRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func newRouterForAgentFeedbackReviewRouteTest(t *testing.T, agentFeedbackReviewHandler AgentFeedbackReviewHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:              h,
		APIKeyHandler:              h,
		WorkspaceHandler:           h,
		UserHandler:                h,
		CatalogHandler:             h,
		DatabaseHandler:            h,
		VolumeHandler:              h,
		GarbageHandler:             h,
		FileHandler:                h,
		VolumeFileHandler:          h,
		VolumeContentHandler:       h,
		AgentFeedbackReviewHandler: agentFeedbackReviewHandler,
		HandlerWrapper:             wrapper,
		TraceIDMiddleware:          passthroughMiddleware,
		LoggingMiddleware:          passthroughMiddleware,
		MetricsMiddleware:          passthroughMiddleware,
		APIKeyMiddlewareFunc:       func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:                     zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentFeedbackReviewRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/feedback"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/feedback/stats"},
	}
}

func TestSetupRoutes_RegistersAgentFeedbackReviewRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentFeedbackReviewRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentFeedbackReviewRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentFeedbackReviewRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentFeedbackReviewRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentFeedbackReviewRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func newRouterForAgentRuntimeRouteTest(t *testing.T, agentRuntimeHandler AgentRuntimeHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:        h,
		APIKeyHandler:        h,
		WorkspaceHandler:     h,
		UserHandler:          h,
		CatalogHandler:       h,
		DatabaseHandler:      h,
		VolumeHandler:        h,
		GarbageHandler:       h,
		FileHandler:          h,
		VolumeFileHandler:    h,
		VolumeContentHandler: h,
		AgentRuntimeHandler:  agentRuntimeHandler,
		HandlerWrapper:       wrapper,
		TraceIDMiddleware:    passthroughMiddleware,
		LoggingMiddleware:    passthroughMiddleware,
		MetricsMiddleware:    passthroughMiddleware,
		APIKeyMiddlewareFunc: func(_ []string) gin.HandlerFunc { return passthroughMiddleware },
		Logger:               zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func newRouterForAgentRuntimeRoutePermissionHandlerTest(t *testing.T, agentRuntimeHandler AgentRuntimeHandler) *GinRouter {
	t.Helper()

	h := &routeTestHandler{}
	wrapper := backendauth.NewHandlerWrapper(
		func(_ *gin.Context) string { return "test-user" },
		func(c *gin.Context, statusCode int, _ common.ErrorCode, _ string) { c.AbortWithStatus(statusCode) },
		zap.NewNop(),
	)

	router, err := NewGinRouter(&GinRouterConfig{
		HealthHandler:        h,
		APIKeyHandler:        h,
		WorkspaceHandler:     h,
		UserHandler:          h,
		CatalogHandler:       h,
		DatabaseHandler:      h,
		VolumeHandler:        h,
		GarbageHandler:       h,
		FileHandler:          h,
		VolumeFileHandler:    h,
		VolumeContentHandler: h,
		AgentRuntimeHandler:  agentRuntimeHandler,
		HandlerWrapper:       wrapper,
		TraceIDMiddleware:    passthroughMiddleware,
		LoggingMiddleware:    passthroughMiddleware,
		MetricsMiddleware:    passthroughMiddleware,
		APIKeyMiddlewareFunc: func(_ []string) gin.HandlerFunc {
			return func(c *gin.Context) {
				workspaceID := c.Param("id")
				if workspaceID == "" {
					workspaceID = c.GetHeader("X-Workspace-ID")
				}
				SetAuthenticatedBackendExecution(c, BackendExecutionContext{
					WorkspaceID:             workspaceID,
					VerifiedEffectiveRoleID: "7",
					WorkspaceAccessVerified: true,
				})
				c.Next()
			}
		},
		Logger: zap.NewNop(),
	})
	require.NoError(t, err)

	router.SetupRoutes()
	return router
}

func expectedAgentRuntimeRoutes() []struct {
	method string
	path   string
} {
	return []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/models/resolve"},
		{method: http.MethodPost, path: "/api/v1/models/openai/chat/completions"},
		{method: http.MethodPost, path: "/api/v1/mcp/http"},
		{method: http.MethodPost, path: "/api/v1/skills/http"},
		{method: http.MethodGet, path: "/api/v1/query-visuals/:file_id/content"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agents/:agent_id/.well-known/agent-card.json"},
		{method: http.MethodGet, path: "/api/v1/workspaces/:id/agents/:agent_id/query-visuals/:file_id/preview"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/agents/:agent_id/a2a"},
		{method: http.MethodPost, path: "/api/v1/workspaces/:id/agents/:agent_id/mcp/http"},
	}
}

func TestSetupRoutes_RegistersAgentRuntimeRoutesWhenHandlerPresent(t *testing.T) {
	router := newRouterForAgentRuntimeRouteTest(t, &routeTestHandler{})

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentRuntimeRoutes() {
		assert.True(t, routeExists(routes, expected.method, expected.path), "%s %s should be registered", expected.method, expected.path)
	}
}

func TestSetupRoutes_DoesNotRegisterAgentRuntimeRoutesWhenHandlerAbsent(t *testing.T) {
	router := newRouterForAgentRuntimeRouteTest(t, nil)

	routes := router.Engine().Routes()
	for _, expected := range expectedAgentRuntimeRoutes() {
		assert.False(t, routeExists(routes, expected.method, expected.path), "%s %s should not be registered", expected.method, expected.path)
	}
}

func TestGenericAgentRoutesMapExploreAgentCodeToRuntime(t *testing.T) {
	runtimeHandler := &recordingAgentRuntimeHandler{}
	router := newRouterForAgentRuntimeRouteTest(t, runtimeHandler)

	card := performRouteRequest(router.Engine(), http.MethodGet, "/api/v1/agents/card?agent_code=explore")
	if card.Code != http.StatusNoContent {
		t.Fatalf("card status = %d", card.Code)
	}
	if runtimeHandler.cardAgentID != "explore" {
		t.Fatalf("runtime card agent_id = %q, want explore", runtimeHandler.cardAgentID)
	}
	if runtimeHandler.cardAgentWorkspaceID != "system" {
		t.Fatalf("runtime card agent_workspace_id = %q, want system", runtimeHandler.cardAgentWorkspaceID)
	}

	body := strings.NewReader(`{"agent_code":"explore","jsonrpc":"2.0","id":"req_1","method":"message/send","params":{"message":{"role":"user","parts":[{"kind":"text","text":"hello"}]}}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/a2a", body)
	req.Header.Set("X-Workspace-ID", "ws_1")
	router.Engine().ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("a2a status = %d body=%s", rec.Code, rec.Body.String())
	}
	if runtimeHandler.a2aBody["agent_id"] != "explore" {
		t.Fatalf("runtime a2a body = %+v", runtimeHandler.a2aBody)
	}
	if runtimeHandler.a2aBody["agent_workspace_id"] != "system" {
		t.Fatalf("runtime a2a body = %+v, want agent_workspace_id system", runtimeHandler.a2aBody)
	}
	if _, ok := runtimeHandler.a2aBody["agent_code"]; ok {
		t.Fatalf("runtime a2a body still has agent_code: %+v", runtimeHandler.a2aBody)
	}
}

func TestSetupRoutes_AgentRuntimeConcretePostValidatesAgentWorkspaceID(t *testing.T) {
	router := newRouterForAgentRuntimeRoutePermissionHandlerTest(t, &routeTestHandler{})

	rec := performRouteRequest(router.Engine(), http.MethodPost, "/api/v1/workspaces/ws_1/agents/ag_1/a2a")
	assert.Equal(t, http.StatusNoContent, rec.Code)

	rec = performRouteRequest(router.Engine(), http.MethodPost, "/api/v1/workspaces/ws_1/agents/ag_1/a2a?agent_workspace_id=system")
	assert.Equal(t, http.StatusNoContent, rec.Code)

	rec = performRouteRequest(router.Engine(), http.MethodPost, "/api/v1/workspaces/ws_1/agents/ag_1/mcp/http?agent_workspace_id=system")
	assert.Equal(t, http.StatusNoContent, rec.Code)

	rec = performRouteRequest(router.Engine(), http.MethodPost, "/api/v1/workspaces/ws_1/agents/ag_1/a2a?agent_workspace_id=other")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSetupRoutes_GenericAgentRuntimeA2APostPreservesAgentID(t *testing.T) {
	runtimeHandler := &recordingAgentRuntimeHandler{}
	router := newRouterForAgentRuntimeRoutePermissionHandlerTest(t, runtimeHandler)

	body := strings.NewReader(`{"agent_id":"ag_1","jsonrpc":"2.0","id":"req_1","method":"message/send","params":{}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/a2a", body)
	req.Header.Set("X-Workspace-ID", "ws_1")
	router.Engine().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	if assert.NotNil(t, runtimeHandler.a2aBody) {
		assert.Equal(t, "ag_1", runtimeHandler.a2aBody["agent_id"])
		assert.Nil(t, runtimeHandler.a2aBody["agent_workspace_id"])
	}
}
