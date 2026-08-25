# Go SDK API Reference

> 本文档从 go-sdk 源码自动生成，请勿手动编辑。
> 修改源码后运行 `make doc-update` 更新。

## Client

Client is the moi-core SDK client.
It provides access to all moi-core services.

### APIKeys

```go
// APIKeys returns the API key management service.
func (c *Client) APIKeys() *APIKeyService
```

APIKeys returns the API key management service.

### AgentPackages

```go
// AgentPackages returns the agent package service scoped to the specified workspace.
func (c *Client) AgentPackages(workspaceID string) *AgentPackageService
```

AgentPackages returns the agent package service scoped to the specified workspace.

### AgentVersions

```go
// AgentVersions returns the agent version lifecycle service scoped to the specified workspace.
func (c *Client) AgentVersions(workspaceID string) *AgentVersionService
```

AgentVersions returns the agent version lifecycle service scoped to the specified workspace.

### Agents

```go
// Agents returns the generic A2A agent service.
func (c *Client) Agents() *AgentService
```

Agents returns the generic A2A agent service.

### CDH

```go
// CDH returns the CDH service scoped to the specified workspace.
// Provides methods for managing CDH configurations and syncing metadata.
func (c *Client) CDH(workspaceID string) *CDHService
```

CDH returns the CDH service scoped to the specified workspace.
Provides methods for managing CDH configurations and syncing metadata.

### Cases

```go
// Cases returns the workflow-case listing service scoped to the specified workspace.
// Used by the workflow execution-log frontend page (issue #9614).
func (c *Client) Cases(workspaceID string) *CaseService
```

Cases returns the workflow-case listing service scoped to the specified workspace.
Used by the workflow execution-log frontend page (issue #9614).

### CatalogTraces

```go
// CatalogTraces returns the Langfuse CatalogTrace management service.
// All operations require a service-account API key; moi-backend performs user-facing
// authorization before making CatalogTrace calls.
func (c *Client) CatalogTraces() *CatalogTraceService
```

CatalogTraces returns the Langfuse CatalogTrace management service.
All operations require a service-account API key; moi-backend performs user-facing
authorization before making CatalogTrace calls.

### Catalogs

```go
// Catalogs returns the catalog management service.
func (c *Client) Catalogs() *CatalogService
```

Catalogs returns the catalog management service.

### Close

```go
// Close closes the client and releases any resources.
// Currently, this is a no-op as the HTTP client doesn't require cleanup,
// but it's provided for future compatibility and to follow best practices.
func (c *Client) Close() error
```

Close closes the client and releases any resources.
Currently, this is a no-op as the HTTP client doesn't require cleanup,
but it's provided for future compatibility and to follow best practices.

### CustomOperators

```go
// CustomOperators returns the custom operator service scoped to the specified workspace.
func (c *Client) CustomOperators(workspaceID string) *CustomOperatorService
```

CustomOperators returns the custom operator service scoped to the specified workspace.

### DataAssets

```go
// DataAssets returns the data asset service.
func (c *Client) DataAssets(workspaceID string) *DataAssetService
```

DataAssets returns the data asset service.

### DataDashboards

```go
// DataDashboards returns the data dashboard service scoped to a workspace.
func (c *Client) DataDashboards(workspaceID string) *DataDashboardService
```

DataDashboards returns the data dashboard service scoped to a workspace.

### DataShare

```go
// DataShare returns the Data Publish and Data Subscription owner service.
func (c *Client) DataShare() *DataShareService
```

DataShare returns the Data Publish and Data Subscription owner service.

### Databases

```go
// Databases returns the database management service.
func (c *Client) Databases() *DatabaseService
```

Databases returns the database management service.

### Dataphin

```go
// Dataphin returns the Dataphin service scoped to the specified workspace.
// Provides methods for managing Dataphin configurations and syncing metadata.
func (c *Client) Dataphin(workspaceID string) *DataphinService
```

Dataphin returns the Dataphin service scoped to the specified workspace.
Provides methods for managing Dataphin configurations and syncing metadata.

### Embeddings

```go
// Embeddings returns the embedding service.
func (c *Client) Embeddings(workspaceID string) *EmbeddingService
```

Embeddings returns the embedding service.

### FileQuery

```go
// FileQuery returns a new FileQueryBuilder with the given context.
//
// Example:
//
//	files, err := client.FileQuery(ctx).
//	    Workspace(workspaceID).
//	    InVolume(volumeID).
//	    WithFileName("report").
//	    Get()
func (c *Client) FileQuery(ctx context.Context) *FileQueryBuilder
```

FileQuery returns a new FileQueryBuilder with the given context.

Example:

	files, err := client.FileQuery(ctx).
	    Workspace(workspaceID).
	    InVolume(volumeID).
	    WithFileName("report").
	    Get()

### Files

```go
// Files returns the file management service.
func (c *Client) Files() *FileService
```

Files returns the file management service.

### Garbage

```go
// Garbage returns the garbage collection service.
func (c *Client) Garbage() *GarbageService
```

Garbage returns the garbage collection service.

### LLM

```go
// LLM returns the LLM service scoped to the specified workspace.
// Covers sessions, messages, tags, backends, router-config, and chat completions.
func (c *Client) LLM(workspaceID string) *LLMService
```

LLM returns the LLM service scoped to the specified workspace.
Covers sessions, messages, tags, backends, router-config, and chat completions.

### MaxCompute

```go
// MaxCompute returns the MaxCompute service scoped to the specified workspace.
// Provides methods for managing MaxCompute configurations and syncing metadata.
func (c *Client) MaxCompute(workspaceID string) *MaxComputeService
```

MaxCompute returns the MaxCompute service scoped to the specified workspace.
Provides methods for managing MaxCompute configurations and syncing metadata.

### MowlLineage

```go
// MowlLineage returns the Phase 1 workflow-lineage atomic service scoped to the specified workspace.
func (c *Client) MowlLineage(workspaceID string) *MowlLineageService
```

MowlLineage returns the Phase 1 workflow-lineage atomic service scoped to the specified workspace.

### ParseResults

```go
// ParseResults returns the parse-result service scoped to the specified workspace.
func (c *Client) ParseResults(workspaceID string) *ParseResultService
```

ParseResults returns the parse-result service scoped to the specified workspace.

### Parsers

```go
// Parsers returns the parser backend management service scoped to the specified workspace.
func (c *Client) Parsers(workspaceID string) *ParserService
```

Parsers returns the parser backend management service scoped to the specified workspace.

### Query

```go
// Query returns a new QueryBuilder with the given context.
//
// Example:
//
//	tree, err := client.Query(ctx).
//	    Workspace(workspaceID).
//	    Catalogs().
//	    WithDatabases().
//	    Get()
func (c *Client) Query(ctx context.Context) *QueryBuilder
```

Query returns a new QueryBuilder with the given context.

Example:

	tree, err := client.Query(ctx).
	    Workspace(workspaceID).
	    Catalogs().
	    WithDatabases().
	    Get()

### Raw

```go
// Raw returns the SDK-owned raw HTTP transport. Authentication and Backend
// execution headers are applied by the SDK just like typed service calls.
func (c *Client) Raw() *RawService
```

Raw returns the SDK-owned raw HTTP transport. Authentication and Backend
execution headers are applied by the SDK just like typed service calls.

### SemanticModels

```go
// SemanticModels returns the semantic model service scoped to the workspace.
func (c *Client) SemanticModels(workspaceID string) *SemanticModelService
```

SemanticModels returns the semantic model service scoped to the workspace.

### ServiceAccounts

```go
// ServiceAccounts returns the server-side UC -> AI Studio catalog facade.
// It requires a client constructed with NewWithCatalogProvisionerBearerToken;
// a human PAT or service-account data-plane Bearer client is rejected before
// a catalog request is sent.
//
// Historical write methods remain source-compatible only. AI Studio no longer
// registers their routes: UC is the sole writer of service-account bindings.
func (c *Client) ServiceAccounts() *ServiceAccountService
```

ServiceAccounts returns the server-side UC -> AI Studio catalog facade.
It requires a client constructed with NewWithCatalogProvisionerBearerToken;
a human PAT or service-account data-plane Bearer client is rejected before
a catalog request is sent.

Historical write methods remain source-compatible only. AI Studio no longer
registers their routes: UC is the sole writer of service-account bindings.

### SetAPIKey

```go
// SetAPIKey updates the API key at runtime.
// This allows changing the API key without creating a new client.
func (c *Client) SetAPIKey(apiKey string)
```

SetAPIKey updates the API key at runtime.
This allows changing the API key without creating a new client.

### StructuredLoad

```go
// StructuredLoad returns the system structured-load runtime service.
func (c *Client) StructuredLoad() *StructuredLoadService
```

StructuredLoad returns the system structured-load runtime service.

### SystemDefaultAI

```go
// SystemDefaultAI returns the system default AI service configuration API.
// It requires a client initialized with the raw moi-core system API key.
func (c *Client) SystemDefaultAI() *SystemDefaultAIService
```

SystemDefaultAI returns the system default AI service configuration API.
It requires a client initialized with the raw moi-core system API key.

### SystemIAM

```go
// SystemIAM returns the system-key-only IAM service. Callers must construct
// the Client with the configured moi-core system API key.
func (c *Client) SystemIAM() *SystemIAMService
```

SystemIAM returns the system-key-only IAM service. Callers must construct
the Client with the configured moi-core system API key.

### SystemResourceDisplay

```go
// SystemResourceDisplay returns the system resource display management service.
func (c *Client) SystemResourceDisplay() *SystemResourceDisplayService
```

SystemResourceDisplay returns the system resource display management service.

### Tasks

```go
// Tasks returns the task management service scoped to the specified workspace.
// workspaceID is required and specifies the workspace context for all operations.
// Validates: Requirements 6.13-6.17
func (c *Client) Tasks(workspaceID string) *TaskService
```

Tasks returns the task management service scoped to the specified workspace.
workspaceID is required and specifies the workspace context for all operations.
Validates: Requirements 6.13-6.17

### Traces

```go
// Traces returns the trace service scoped to the specified workspace.
func (c *Client) Traces(workspaceID string) *TraceService
```

Traces returns the trace service scoped to the specified workspace.

### Upgrade

```go
// Upgrade returns the system auto-upgrade diagnostics API.
// It requires a client initialized with the raw moi-core system API key.
func (c *Client) Upgrade() *UpgradeService
```

Upgrade returns the system auto-upgrade diagnostics API.
It requires a client initialized with the raw moi-core system API key.

### Users

```go
// Users returns the user management service.
func (c *Client) Users() *UserService
```

Users returns the user management service.

### VolumeContent

```go
// VolumeContent returns a new VolumeContentBuilder with the given context.
//
// Example:
//
//	result, err := client.VolumeContent(ctx).
//	    Workspace(workspaceID).
//	    Volume(volumeID).
//	    WithAll().
//	    FilterByName("report").
//	    OrderBy("name").
//	    Get()
func (c *Client) VolumeContent(ctx context.Context) *VolumeContentBuilder
```

VolumeContent returns a new VolumeContentBuilder with the given context.

Example:

	result, err := client.VolumeContent(ctx).
	    Workspace(workspaceID).
	    Volume(volumeID).
	    WithAll().
	    FilterByName("report").
	    OrderBy("name").
	    Get()

### VolumeFiles

```go
// VolumeFiles returns the volume file management service.
func (c *Client) VolumeFiles() *VolumeFileService
```

VolumeFiles returns the volume file management service.

### Volumes

```go
// Volumes returns the volume management service.
func (c *Client) Volumes() *VolumeService
```

Volumes returns the volume management service.

### WorkItems

```go
// WorkItems returns the workitem management service scoped to the specified workspace.
// workspaceID is required and specifies the workspace context for all operations.
// Validates: Requirements 6.18
func (c *Client) WorkItems(workspaceID string) *WorkItemService
```

WorkItems returns the workitem management service scoped to the specified workspace.
workspaceID is required and specifies the workspace context for all operations.
Validates: Requirements 6.18

### Worker

```go
// Worker creates a new WorkerClient for connecting to the Mowl Engine
// via the Catalog Service gRPC proxy. The gRPC endpoint is derived from
// the Client's endpoint (same host:port).
// Requires WithWorkerID to be set on the Client.
// workspaceID is required and specifies the workspace context for all operations.
func (c *Client) Worker(workspaceID string, opts ...WorkerClientOption) *WorkerClient
```

Worker creates a new WorkerClient for connecting to the Mowl Engine
via the Catalog Service gRPC proxy. The gRPC endpoint is derived from
the Client's endpoint (same host:port).
Requires WithWorkerID to be set on the Client.
workspaceID is required and specifies the workspace context for all operations.

### WorkflowApps

```go
// WorkflowApps returns product-level workflow app management scoped to the workspace.
func (c *Client) WorkflowApps(workspaceID string) *WorkflowAppService
```

WorkflowApps returns product-level workflow app management scoped to the workspace.

### WorkflowDeployments

```go
// WorkflowDeployments returns the high-level transactional workflow deployment service.
func (c *Client) WorkflowDeployments(workspaceID string) *WorkflowDeploymentService
```

WorkflowDeployments returns the high-level transactional workflow deployment service.

### WorkflowVersions

```go
// WorkflowVersions returns the workflow version management service scoped to the specified workspace.
// workspaceID is required and specifies the workspace context for all operations.
// Validates: Requirements 6.7-6.12
func (c *Client) WorkflowVersions(workspaceID string) *WorkflowVersionService
```

WorkflowVersions returns the workflow version management service scoped to the specified workspace.
workspaceID is required and specifies the workspace context for all operations.
Validates: Requirements 6.7-6.12

### Workflows

```go
// Workflows returns the workflow management service scoped to the specified workspace.
// workspaceID is required and specifies the workspace context for all operations.
// Validates: Requirements 6.1-6.6
func (c *Client) Workflows(workspaceID string) *WorkflowService
```

Workflows returns the workflow management service scoped to the specified workspace.
workspaceID is required and specifies the workspace context for all operations.
Validates: Requirements 6.1-6.6

### Workspaces

```go
// Workspaces returns the workspace management service.
func (c *Client) Workspaces() *WorkspaceService
```

Workspaces returns the workspace management service.

### 构造函数与选项

### New

```go
// New creates a new moi-core SDK client using the legacy API-key-compatible
// constructor. It sends the provided credential only in X-API-Key.
//
// Callers using a UC personal access token should prefer
// NewWithPersonalAccessToken to make that intent explicit. New remains
// source-compatible for existing callers and has the same HTTP header
// behavior.
//
// Parameters:
//   - endpoint: The service endpoint URL (required), e.g., "https://api.example.com"
//   - apiKey: The API key for authentication (required)
//   - opts: Optional configuration options
//
// Returns an error if endpoint or apiKey is empty.
func New(endpoint, apiKey string, opts ...Option) (*Client, error)
```

New creates a new moi-core SDK client using the legacy API-key-compatible
constructor. It sends the provided credential only in X-API-Key.

Callers using a UC personal access token should prefer
NewWithPersonalAccessToken to make that intent explicit. New remains
source-compatible for existing callers and has the same HTTP header
behavior.

Parameters:
  - endpoint: The service endpoint URL (required), e.g., "https://api.example.com"
  - apiKey: The API key for authentication (required)
  - opts: Optional configuration options

Returns an error if endpoint or apiKey is empty.

### WithAPIKeyIdempotencyKey

```go
// WithAPIKeyIdempotencyKey makes system-managed API key creation safely retryable.
func WithAPIKeyIdempotencyKey(key string) CreateAPIKeyOption
```

WithAPIKeyIdempotencyKey makes system-managed API key creation safely retryable.

### WithAllWorkItemVersions

```go
// WithAllWorkItemVersions includes all versions in capability output.
func WithAllWorkItemVersions() WorkItemCapabilityBuildOption
```

WithAllWorkItemVersions includes all versions in capability output.

### WithAppendContent

```go
// WithAppendContent sets the content to append.
func WithAppendContent(content string) AppendModifiedResponseOption
```

WithAppendContent sets the content to append.

### WithBackendAPIKey

```go
// WithBackendAPIKey sets the encrypted API key.
func WithBackendAPIKey(key string) CreateBackendOption
```

WithBackendAPIKey sets the encrypted API key.

### WithBackendModels

```go
// WithBackendModels sets the supported model list.
func WithBackendModels(models []string) CreateBackendOption
```

WithBackendModels sets the supported model list.

### WithBackendName

```go
// WithBackendName sets the backend name.
func WithBackendName(name string) CreateBackendOption
```

WithBackendName sets the backend name.

### WithBackendReasoningControlProtocol

```go
// WithBackendReasoningControlProtocol sets how this backend encodes
// request-level reasoning/thinking controls.
func WithBackendReasoningControlProtocol(protocol catalog.ReasoningControlProtocol) CreateBackendOption
```

WithBackendReasoningControlProtocol sets how this backend encodes
request-level reasoning/thinking controls.

### WithBackendTimeout

```go
// WithBackendTimeout sets the timeout in seconds.
func WithBackendTimeout(seconds int32) CreateBackendOption
```

WithBackendTimeout sets the timeout in seconds.

### WithBackendType

```go
// WithBackendType sets the backend type.
func WithBackendType(t catalog.BackendType) CreateBackendOption
```

WithBackendType sets the backend type.

### WithBatchConcurrency

```go
// WithBatchConcurrency sets the concurrency level for batch operations.
func WithBatchConcurrency(n int) BatchOption
```

WithBatchConcurrency sets the concurrency level for batch operations.

### WithCDHConnectTimeout

```go
// WithCDHConnectTimeout sets the connection timeout (seconds) when creating a CDH config.
func WithCDHConnectTimeout(timeout int32) CreateCDHConfigOption
```

WithCDHConnectTimeout sets the connection timeout (seconds) when creating a CDH config.

### WithCDHHiveAddress

```go
// WithCDHHiveAddress sets the new hive address when updating a CDH config.
func WithCDHHiveAddress(addr string) UpdateCDHConfigOption
```

WithCDHHiveAddress sets the new hive address when updating a CDH config.

### WithCDHKerberosKeytab

```go
// WithCDHKerberosKeytab sets the Kerberos keytab path when creating a CDH config.
func WithCDHKerberosKeytab(keytab string) CreateCDHConfigOption
```

WithCDHKerberosKeytab sets the Kerberos keytab path when creating a CDH config.

### WithCDHKerberosPrincipal

```go
// WithCDHKerberosPrincipal sets the Kerberos principal when creating a CDH config.
func WithCDHKerberosPrincipal(principal string) CreateCDHConfigOption
```

WithCDHKerberosPrincipal sets the Kerberos principal when creating a CDH config.

### WithCDHMetastoreAddress

```go
// WithCDHMetastoreAddress sets the new metastore address when updating a CDH config.
func WithCDHMetastoreAddress(addr string) UpdateCDHConfigOption
```

WithCDHMetastoreAddress sets the new metastore address when updating a CDH config.

### WithCDHName

```go
// WithCDHName sets the new name when updating a CDH config.
func WithCDHName(name string) UpdateCDHConfigOption
```

WithCDHName sets the new name when updating a CDH config.

### WithCDHUpdatedConnectTimeout

```go
// WithCDHUpdatedConnectTimeout sets the new connection timeout when updating a CDH config.
func WithCDHUpdatedConnectTimeout(timeout int32) UpdateCDHConfigOption
```

WithCDHUpdatedConnectTimeout sets the new connection timeout when updating a CDH config.

### WithCDHUpdatedKerberosKeytab

```go
// WithCDHUpdatedKerberosKeytab sets the new Kerberos keytab path when updating a CDH config.
func WithCDHUpdatedKerberosKeytab(keytab string) UpdateCDHConfigOption
```

WithCDHUpdatedKerberosKeytab sets the new Kerberos keytab path when updating a CDH config.

### WithCDHUpdatedKerberosPrincipal

```go
// WithCDHUpdatedKerberosPrincipal sets the new Kerberos principal when updating a CDH config.
func WithCDHUpdatedKerberosPrincipal(principal string) UpdateCDHConfigOption
```

WithCDHUpdatedKerberosPrincipal sets the new Kerberos principal when updating a CDH config.

### WithCDHVersion

```go
// WithCDHVersion sets the new version when updating a CDH config.
func WithCDHVersion(version string) UpdateCDHConfigOption
```

WithCDHVersion sets the new version when updating a CDH config.

### WithCaseID

```go
// WithCaseID filters notifications to only the specified workflow case (run).
// Use together with WaitForWorkflowNotification to wait for a specific execution.
func WithCaseID(caseID string) NotificationOption
```

WithCaseID filters notifications to only the specified workflow case (run).
Use together with WaitForWorkflowNotification to wait for a specific execution.

### WithCaseLimit

```go
// WithCaseLimit sets the page size. Server caps at 200; zero/negative means server default (20).
func WithCaseLimit(limit int32) ListCasesOption
```

WithCaseLimit sets the page size. Server caps at 200; zero/negative means server default (20).

### WithCaseOffset

```go
// WithCaseOffset sets the pagination offset. Zero = start of results.
func WithCaseOffset(offset int32) ListCasesOption
```

WithCaseOffset sets the pagination offset. Zero = start of results.

### WithCaseWorkflowVersionIDs

```go
// WithCaseWorkflowVersionIDs filters cases by the given workflow_version_ids.
// An empty / missing option means "all versions".
func WithCaseWorkflowVersionIDs(ids ...string) ListCasesOption
```

WithCaseWorkflowVersionIDs filters cases by the given workflow_version_ids.
An empty / missing option means "all versions".

### WithChatExtra

```go
// WithChatExtra adds a custom top-level body field beyond ProxyExtension.
// Standard fields (model, messages, stream, temperature, max_tokens, and
// proxy) are reserved and make request construction fail; use their typed
// options instead.
func WithChatExtra(key string, value interface{}) ChatCompletionOption
```

WithChatExtra adds a custom top-level body field beyond ProxyExtension.
Standard fields (model, messages, stream, temperature, max_tokens, and
proxy) are reserved and make request construction fail; use their typed
options instead.

### WithChatExtras

```go
// WithChatExtras sets multiple custom top-level body fields. Standard fields
// are reserved as described by WithChatExtra.
func WithChatExtras(extras map[string]interface{}) ChatCompletionOption
```

WithChatExtras sets multiple custom top-level body fields. Standard fields
are reserved as described by WithChatExtra.

### WithChatMaxTokens

```go
// WithChatMaxTokens sets maximum tokens to generate.
func WithChatMaxTokens(n int) ChatCompletionOption
```

WithChatMaxTokens sets maximum tokens to generate.

### WithChatMessages

```go
// WithChatMessages overrides the default single user message (OpenAI messages array).
func WithChatMessages(messages []map[string]interface{}) ChatCompletionOption
```

WithChatMessages overrides the default single user message (OpenAI messages array).

### WithChatRequestBodyMaxBytes

```go
// WithChatRequestBodyMaxBytes rejects a serialized chat-completion request
// larger than maxBytes before any network I/O. It is a caller-supplied
// transport contract, useful when the selected upstream has a stricter body
// limit than Catalog itself.
func WithChatRequestBodyMaxBytes(maxBytes int) ChatCompletionOption
```

WithChatRequestBodyMaxBytes rejects a serialized chat-completion request
larger than maxBytes before any network I/O. It is a caller-supplied
transport contract, useful when the selected upstream has a stricter body
limit than Catalog itself.

### WithChatTemperature

```go
// WithChatTemperature sets sampling temperature (0..2).
func WithChatTemperature(t float64) ChatCompletionOption
```

WithChatTemperature sets sampling temperature (0..2).

### WithComment

```go
// WithComment sets the catalog comment when creating a catalog.
//
// Example:
//
//	catalog, err := client.Catalogs().Create(ctx, workspaceID, "my-catalog", moi.WithComment("This is my catalog"))
func WithComment(comment string) CreateCatalogOption
```

WithComment sets the catalog comment when creating a catalog.

Example:

	catalog, err := client.Catalogs().Create(ctx, workspaceID, "my-catalog", moi.WithComment("This is my catalog"))

### WithCompletedOnly

```go
// WithCompletedOnly restricts to completed messages only.
func WithCompletedOnly() LatestMessageOption
```

WithCompletedOnly restricts to completed messages only.

### WithContentType

```go
// WithContentType sets the file MIME type when uploading a file.
func WithContentType(contentType string) UploadOption
```

WithContentType sets the file MIME type when uploading a file.

### WithContinueOnError

```go
// WithContinueOnError sets whether to continue on error during batch operations.
func WithContinueOnError(v bool) BatchOption
```

WithContinueOnError sets whether to continue on error during batch operations.

### WithConvertInputFormat

```go
// WithConvertInputFormat sets the input file format explicitly for convert.
func WithConvertInputFormat(format string) ParserConvertOption
```

WithConvertInputFormat sets the input file format explicitly for convert.

### WithCreateTaskID

```go
// WithCreateTaskID supplies a caller-owned task ID. It is intended for durable
// internal deliveries that need retry-safe CreateTask semantics; normal
// interactive callers should leave the ID empty and let Core allocate it.
func WithCreateTaskID(taskID string) CreateTaskOption
```

WithCreateTaskID supplies a caller-owned task ID. It is intended for durable
internal deliveries that need retry-safe CreateTask semantics; normal
interactive callers should leave the ID empty and let Core allocate it.

### WithCreateUserNickname

```go
// WithCreateUserNickname sets the nickname when creating a user.
//
// Example:
//
//	user, err := client.Users().Create(ctx, "user@example.com", "username", "password",
//	    moi.WithCreateUserNickname("John Doe"))
func WithCreateUserNickname(nickname string) CreateUserOption
```

WithCreateUserNickname sets the nickname when creating a user.

Example:

	user, err := client.Users().Create(ctx, "user@example.com", "username", "password",
	    moi.WithCreateUserNickname("John Doe"))

### WithCreateUserPasswordless

```go
// WithCreateUserPasswordless provisions a user for an external identity
// provider without accepting a user-supplied password. Core stores only a
// randomly generated internal credential for the non-login-capable account.
func WithCreateUserPasswordless() CreateUserOption
```

WithCreateUserPasswordless provisions a user for an external identity
provider without accepting a user-supplied password. Core stores only a
randomly generated internal credential for the non-login-capable account.

### WithCreateUserPhone

```go
// WithCreateUserPhone sets the phone when creating a user.
//
// Example:
//
//	user, err := client.Users().Create(ctx, "user@example.com", "username", "password",
//	    moi.WithCreateUserPhone("+1234567890"))
func WithCreateUserPhone(phone string) CreateUserOption
```

WithCreateUserPhone sets the phone when creating a user.

Example:

	user, err := client.Users().Create(ctx, "user@example.com", "username", "password",
	    moi.WithCreateUserPhone("+1234567890"))

### WithCronExpression

```go
// WithCronExpression sets the cron expression for periodic task execution.
func WithCronExpression(cron string) WorkerTaskOption
```

WithCronExpression sets the cron expression for periodic task execution.

### WithCustomHeader

```go
// WithCustomHeader is a convenience function that adds a single custom header
// to all requests. It's a shorthand for WithRequestModifier.
//
// Example:
//
//	client, err := moi.New(endpoint, apiKey, moi.WithCustomHeader("X-Test-ID", "test-123"))
func WithCustomHeader(key, value string) Option
```

WithCustomHeader is a convenience function that adds a single custom header
to all requests. It's a shorthand for WithRequestModifier.

Example:

	client, err := moi.New(endpoint, apiKey, moi.WithCustomHeader("X-Test-ID", "test-123"))

### WithDBInterpolateParams

```go
// WithDBInterpolateParams enables client-side placeholder interpolation for
// SDK-opened DB connections. When enabled, parameterized Exec/Query are sent as
// plain SQL with literal values instead of server-side prepared statements.
func WithDBInterpolateParams(enabled bool) DBOpenOption
```

WithDBInterpolateParams enables client-side placeholder interpolation for
SDK-opened DB connections. When enabled, parameterized Exec/Query are sent as
plain SQL with literal values instead of server-side prepared statements.

### WithDBMultiStatements

```go
// WithDBMultiStatements enables multi-statement execution for SDK-opened DB connections.
func WithDBMultiStatements(enabled bool) DBOpenOption
```

WithDBMultiStatements enables multi-statement execution for SDK-opened DB connections.

### WithDBName

```go
// WithDBName overrides the database selected by DBConnection. Passing an empty
// string opens the account/user connection without selecting a default database.
func WithDBName(name string) DBOpenOption
```

WithDBName overrides the database selected by DBConnection. Passing an empty
string opens the account/user connection without selecting a default database.

### WithDBReadTimeout

```go
// WithDBReadTimeout sets the read timeout for SDK-opened DB connections.
func WithDBReadTimeout(timeout time.Duration) DBOpenOption
```

WithDBReadTimeout sets the read timeout for SDK-opened DB connections.

### WithDBTimeout

```go
// WithDBTimeout sets the TCP connection timeout for SDK-opened DB connections.
func WithDBTimeout(timeout time.Duration) DBOpenOption
```

WithDBTimeout sets the TCP connection timeout for SDK-opened DB connections.

### WithDBWriteTimeout

```go
// WithDBWriteTimeout sets the write timeout for SDK-opened DB connections.
func WithDBWriteTimeout(timeout time.Duration) DBOpenOption
```

WithDBWriteTimeout sets the write timeout for SDK-opened DB connections.

### WithDPAccessKeyID

```go
// WithDPAccessKeyID sets the new access key ID when updating a Dataphin config.
func WithDPAccessKeyID(accessKeyID string) UpdateDPConfigOption
```

WithDPAccessKeyID sets the new access key ID when updating a Dataphin config.

### WithDPAccessKeySecret

```go
// WithDPAccessKeySecret sets the new access key secret when updating a Dataphin config.
func WithDPAccessKeySecret(accessKeySecret string) UpdateDPConfigOption
```

WithDPAccessKeySecret sets the new access key secret when updating a Dataphin config.

### WithDPEndpoint

```go
// WithDPEndpoint sets the new endpoint when updating a Dataphin config.
func WithDPEndpoint(endpoint string) UpdateDPConfigOption
```

WithDPEndpoint sets the new endpoint when updating a Dataphin config.

### WithDPName

```go
// WithDPName sets the new name when updating a Dataphin config.
func WithDPName(name string) UpdateDPConfigOption
```

WithDPName sets the new name when updating a Dataphin config.

### WithDPProjectName

```go
// WithDPProjectName sets the project name when creating a Dataphin config.
func WithDPProjectName(projectName string) CreateDPConfigOption
```

WithDPProjectName sets the project name when creating a Dataphin config.

### WithDPRegion

```go
// WithDPRegion sets the region when creating a Dataphin config.
func WithDPRegion(region string) CreateDPConfigOption
```

WithDPRegion sets the region when creating a Dataphin config.

### WithDPUpdatedProjectName

```go
// WithDPUpdatedProjectName sets the new project name when updating a Dataphin config.
func WithDPUpdatedProjectName(projectName string) UpdateDPConfigOption
```

WithDPUpdatedProjectName sets the new project name when updating a Dataphin config.

### WithDPUpdatedRegion

```go
// WithDPUpdatedRegion sets the new region when updating a Dataphin config.
func WithDPUpdatedRegion(region string) UpdateDPConfigOption
```

WithDPUpdatedRegion sets the new region when updating a Dataphin config.

### WithData

```go
// WithData sets the initial data (JSON) for the task execution.
func WithData(data string) WorkerTaskOption
```

WithData sets the initial data (JSON) for the task execution.

### WithDataAssetID

```go
// WithDataAssetID sets asset_id.
func WithDataAssetID(id string) DataAssetOption
```

WithDataAssetID sets asset_id.

### WithDataAssetMeta

```go
// WithDataAssetMeta sets meta struct.
func WithDataAssetMeta(meta *structpb.Struct) DataAssetOption
```

WithDataAssetMeta sets meta struct.

### WithDataAssetMetaMap

```go
// WithDataAssetMetaMap sets meta from map.
func WithDataAssetMetaMap(meta map[string]interface{}) DataAssetOption
```

WithDataAssetMetaMap sets meta from map.

### WithDataAssetName

```go
// WithDataAssetName sets name.
func WithDataAssetName(name string) DataAssetOption
```

WithDataAssetName sets name.

### WithDataAssetReplaceMeta

```go
// WithDataAssetReplaceMeta updates an existing typed asset's meta when the
// same asset_type and asset_ref already exist.
func WithDataAssetReplaceMeta() DataAssetOption
```

WithDataAssetReplaceMeta updates an existing typed asset's meta when the
same asset_type and asset_ref already exist.

### WithDataAssetSource

```go
// WithDataAssetSource sets source.
func WithDataAssetSource(source string) DataAssetOption
```

WithDataAssetSource sets source.

### WithDataAssetType

```go
// WithDataAssetType sets asset_type.
func WithDataAssetType(assetType string) DataAssetOption
```

WithDataAssetType sets asset_type.

### WithDataAssetVolumeID

```go
// WithDataAssetVolumeID sets volume_id.
func WithDataAssetVolumeID(volumeID int64) DataAssetOption
```

WithDataAssetVolumeID sets volume_id.

### WithDataDerivationIdempotencyKey

```go
// WithDataDerivationIdempotencyKey sets an idempotency key for derivation upsert.
func WithDataDerivationIdempotencyKey(key string) DataDerivationOption
```

WithDataDerivationIdempotencyKey sets an idempotency key for derivation upsert.

### WithDataDerivationMeta

```go
// WithDataDerivationMeta sets meta struct.
func WithDataDerivationMeta(meta *structpb.Struct) DataDerivationOption
```

WithDataDerivationMeta sets meta struct.

### WithDataDerivationMetaMap

```go
// WithDataDerivationMetaMap sets meta from map.
func WithDataDerivationMetaMap(meta map[string]interface{}) DataDerivationOption
```

WithDataDerivationMetaMap sets meta from map.

### WithDataDerivationProducer

```go
// WithDataDerivationProducer sets the producing workitem and logical slot for a derivation edge.
func WithDataDerivationProducer(producerWorkitemID, logicalSlot string) DataDerivationOption
```

WithDataDerivationProducer sets the producing workitem and logical slot for a derivation edge.

### WithDataDerivationRootAssetID

```go
// WithDataDerivationRootAssetID sets the root asset id shared by a derivation chain.
func WithDataDerivationRootAssetID(rootAssetID string) DataDerivationOption
```

WithDataDerivationRootAssetID sets the root asset id shared by a derivation chain.

### WithDataDerivationRuntime

```go
// WithDataDerivationRuntime sets workflow runtime provenance for a derivation edge.
func WithDataDerivationRuntime(caseID, recordedByWorkitemID string, parallelIndex int32) DataDerivationOption
```

WithDataDerivationRuntime sets workflow runtime provenance for a derivation edge.

### WithDatabaseCreateIAM

```go
// WithDatabaseCreateIAM marks a user-initiated database creation sync. Core
// re-authorizes database.create on the parent Catalog, validates the new
// database name against the Catalog identifier contract, and atomically
// registers Database ownership. Background discovery must not use this option.
func WithDatabaseCreateIAM() SyncMetadataOption
```

WithDatabaseCreateIAM marks a user-initiated database creation sync. Core
re-authorizes database.create on the parent Catalog, validates the new
database name against the Catalog identifier contract, and atomically
registers Database ownership. Background discovery must not use this option.

### WithDatabaseDeleteAuthorizeOnly

```go
// WithDatabaseDeleteAuthorizeOnly re-authorizes database.delete without applying
// metadata changes. Used before irreversible MatrixOne DROP DATABASE.
func WithDatabaseDeleteAuthorizeOnly(databaseID int64) SyncMetadataOption
```

WithDatabaseDeleteAuthorizeOnly re-authorizes database.delete without applying
metadata changes. Used before irreversible MatrixOne DROP DATABASE.

### WithDatabaseDeleteIAM

```go
// WithDatabaseDeleteIAM marks a user-initiated database delete metadata sync.
// Core verifies database.delete against this trusted database id, then removes
// Catalog metadata after MatrixOne DROP (or when the database is already gone).
func WithDatabaseDeleteIAM(databaseID int64) SyncMetadataOption
```

WithDatabaseDeleteIAM marks a user-initiated database delete metadata sync.
Core verifies database.delete against this trusted database id, then removes
Catalog metadata after MatrixOne DROP (or when the database is already gone).

### WithDatabaseUpdateIAM

```go
// WithDatabaseUpdateIAM marks a user-initiated metadata update. Core verifies
// database.update against this trusted database id before applying the sync.
func WithDatabaseUpdateIAM(databaseID int64) SyncMetadataOption
```

WithDatabaseUpdateIAM marks a user-initiated metadata update. Core verifies
database.update against this trusted database id before applying the sync.

### WithDescription

```go
// WithDescription sets the description for the task.
// This is particularly useful for dynamic services to document their purpose.
func WithDescription(desc string) WorkerTaskOption
```

WithDescription sets the description for the task.
This is particularly useful for dynamic services to document their purpose.

### WithDynamicService

```go
// WithDynamicService marks the task as a dynamic service with the specified schemas.
// inputSchema and outputSchema are required for dynamic services.
// This option automatically sets the task type to "dynamic_service".
//
// Example:
//
//	inputSchema := moi.NewSchema().Property("url", moi.StringSchema()).Required("url")
//	outputSchema := moi.NewSchema().Property("result", moi.StringSchema())
//	task, err := worker.ExecuteByBuilder("my-service", builder,
//	    moi.WithDynamicService(inputSchema, outputSchema),
//	    moi.WithResultMode("oneshot"),
//	)
func WithDynamicService(inputSchema, outputSchema *SchemaBuilder) WorkerTaskOption
```

WithDynamicService marks the task as a dynamic service with the specified schemas.
inputSchema and outputSchema are required for dynamic services.
This option automatically sets the task type to "dynamic_service".

Example:

	inputSchema := moi.NewSchema().Property("url", moi.StringSchema()).Required("url")
	outputSchema := moi.NewSchema().Property("result", moi.StringSchema())
	task, err := worker.ExecuteByBuilder("my-service", builder,
	    moi.WithDynamicService(inputSchema, outputSchema),
	    moi.WithResultMode("oneshot"),
	)

### WithEmbeddingBackendID

```go
// WithEmbeddingBackendID pins the request to a specific backend.
func WithEmbeddingBackendID(id int64) EmbeddingOption
```

WithEmbeddingBackendID pins the request to a specific backend.

### WithEmbeddingEncodingFormat

```go
// WithEmbeddingEncodingFormat sets encoding_format (e.g. "float").
func WithEmbeddingEncodingFormat(format string) EmbeddingOption
```

WithEmbeddingEncodingFormat sets encoding_format (e.g. "float").

### WithEmbeddingExtra

```go
// WithEmbeddingExtra sets extra top-level body fields.
func WithEmbeddingExtra(key string, value interface{}) EmbeddingOption
```

WithEmbeddingExtra sets extra top-level body fields.

### WithEmbeddingProxyExtension

```go
// WithEmbeddingProxyExtension sets proxy extension.
func WithEmbeddingProxyExtension(ext *catalog.ProxyExtension) EmbeddingOption
```

WithEmbeddingProxyExtension sets proxy extension.

### WithEndpointAddress

```go
// WithEndpointAddress sets the endpoint address.
func WithEndpointAddress(address string) CreateEndpointOption
```

WithEndpointAddress sets the endpoint address.

### WithEndpointStatus

```go
// WithEndpointStatus sets the endpoint status.
func WithEndpointStatus(status catalog.EndpointStatus) SetEndpointStatusOption
```

WithEndpointStatus sets the endpoint status.

### WithExpectedSessionTitle

```go
// WithExpectedSessionTitle updates the session only while its title still
// matches expectedTitle. A mismatch returns the current unchanged session
// without an error.
func WithExpectedSessionTitle(expectedTitle string) UpdateSessionOption
```

WithExpectedSessionTitle updates the session only while its title still
matches expectedTitle. A mismatch returns the current unchanged session
without an error.

### WithExpiresInDays

```go
// WithExpiresInDays sets the expiration time in days when creating an API key.
// A value of 0 means the API key never expires.
//
// Example:
//
//	apiKey, err := client.APIKeys().Create(ctx, "my-api-key", moi.WithExpiresInDays(30))
func WithExpiresInDays(days int) CreateAPIKeyOption
```

WithExpiresInDays sets the expiration time in days when creating an API key.
A value of 0 means the API key never expires.

Example:

	apiKey, err := client.APIKeys().Create(ctx, "my-api-key", moi.WithExpiresInDays(30))

### WithFilesPageSize

```go
// WithFilesPageSize sets the page size for list files operations.
func WithFilesPageSize(size int32) ListFilesOption
```

WithFilesPageSize sets the page size for list files operations.

### WithFilesPageToken

```go
// WithFilesPageToken sets the page token for list files operations.
func WithFilesPageToken(token string) ListFilesOption
```

WithFilesPageToken sets the page token for list files operations.

### WithGarbageBatchSize

```go
// WithGarbageBatchSize sets the batch size for garbage collection.
//
// Example:
//
//	result, err := client.Garbage().TriggerGarbageCollection(ctx, workspaceID, moi.WithGarbageBatchSize(50))
func WithGarbageBatchSize(size int) TriggerGarbageCollectionOption
```

WithGarbageBatchSize sets the batch size for garbage collection.

Example:

	result, err := client.Garbage().TriggerGarbageCollection(ctx, workspaceID, moi.WithGarbageBatchSize(50))

### WithHTTPClient

```go
// WithHTTPClient sets a custom HTTP client for the SDK client.
// This allows customization of transport settings, TLS configuration, etc.
//
// Example:
//
//	httpClient := &http.Client{
//	    Transport: &http.Transport{
//	        MaxIdleConns: 100,
//	    },
//	}
//	client, err := moi.New(endpoint, apiKey, moi.WithHTTPClient(httpClient))
func WithHTTPClient(hc *http.Client) Option
```

WithHTTPClient sets a custom HTTP client for the SDK client.
This allows customization of transport settings, TLS configuration, etc.

Example:

	httpClient := &http.Client{
	    Transport: &http.Transport{
	        MaxIdleConns: 100,
	    },
	}
	client, err := moi.New(endpoint, apiKey, moi.WithHTTPClient(httpClient))

### WithHTTPRoundTripperDecorator

```go
// WithHTTPRoundTripperDecorator installs a transport wrapper for normal and
// streaming HTTP requests. It is useful for trusted request policy hooks.
func WithHTTPRoundTripperDecorator(decorator HTTPRoundTripperDecorator) Option
```

WithHTTPRoundTripperDecorator installs a transport wrapper for normal and
streaming HTTP requests. It is useful for trusted request policy hooks.

### WithI18nPacks

```go
// WithI18nPacks attaches i18n language packs to the workitem registration.
//
// Example:
//
//	worker.RegisterWorkItemWithOptions("moi:files.read_text", metadata, handler,
//	    WithI18nPacks(
//	        NewI18nPacks().
//	            Locale(common.Language_LANGUAGE_EN, `{"contract_version":"1.0","semantic":{"summary":"Read file"}}`).
//	            Locale(common.Language_LANGUAGE_ZH, `{"contract_version":"1.0","semantic":{"summary":"读取文件"}}`).
//	            Default(common.Language_LANGUAGE_EN),
//	    ))
func WithI18nPacks(builder *I18nPackBuilder) WorkItemOption
```

WithI18nPacks attaches i18n language packs to the workitem registration.

Example:

	worker.RegisterWorkItemWithOptions("moi:files.read_text", metadata, handler,
	    WithI18nPacks(
	        NewI18nPacks().
	            Locale(common.Language_LANGUAGE_EN, `{"contract_version":"1.0","semantic":{"summary":"Read file"}}`).
	            Locale(common.Language_LANGUAGE_ZH, `{"contract_version":"1.0","semantic":{"summary":"读取文件"}}`).
	            Default(common.Language_LANGUAGE_EN),
	    ))

### WithInitialRefCount

```go
// WithInitialRefCount sets the file metadata ref_count at upload time.
// The default is 0. Worker-produced artifacts use 1 so catalog file GC does
// not treat them as orphan files before downstream workflow nodes can read or
// index them.
func WithInitialRefCount(count int32) UploadFileOption
```

WithInitialRefCount sets the file metadata ref_count at upload time.
The default is 0. Worker-produced artifacts use 1 so catalog file GC does
not treat them as orphan files before downstream workflow nodes can read or
index them.

### WithInputSchema

```go
// WithInputSchema sets the input schema for a WorkItem.
// The schema must be a valid JSON Schema (draft-07 or higher).
//
// Example:
//
//	schema := `{"type": "object", "properties": {"url": {"type": "string"}}, "required": ["url"]}`
//	worker.RegisterWorkItemWithOptions("http_request", metadata, handler,
//	    WithInputSchema(schema))
func WithInputSchema(schema string) WorkItemOption
```

WithInputSchema sets the input schema for a WorkItem.
The schema must be a valid JSON Schema (draft-07 or higher).

Example:

	schema := `{"type": "object", "properties": {"url": {"type": "string"}}, "required": ["url"]}`
	worker.RegisterWorkItemWithOptions("http_request", metadata, handler,
	    WithInputSchema(schema))

### WithInputSchemaBuilder

```go
// WithInputSchemaBuilder sets the input schema using a SchemaBuilder.
// The builder's Build() method will be called automatically.
//
// Example:
//
//	builder := NewSchema().
//	    Property("url", StringSchema().MinLength(1)).
//	    Required("url")
//	worker.RegisterWorkItemWithOptions("http_request", metadata, handler,
//	    WithInputSchemaBuilder(builder))
func WithInputSchemaBuilder(builder *SchemaBuilder) WorkItemOption
```

WithInputSchemaBuilder sets the input schema using a SchemaBuilder.
The builder's Build() method will be called automatically.

Example:

	builder := NewSchema().
	    Property("url", StringSchema().MinLength(1)).
	    Required("url")
	worker.RegisterWorkItemWithOptions("http_request", metadata, handler,
	    WithInputSchemaBuilder(builder))

### WithInvokeTraceOptions

```go
// WithInvokeTraceOptions sets trace options for dynamic service invocation.
func WithInvokeTraceOptions(trace *mowl.TraceOptions) InvokeOption
```

WithInvokeTraceOptions sets trace options for dynamic service invocation.

### WithLabels

```go
// WithLabels sets scheduling labels on the worker (for example
// matrixorigin.io/compute-resource-id and mowl.runtime/worker-type). These labels
// are sent during registration and attach, and used by the scheduler for
// compute-resource-based isolation. A worker that registers WorkItems must declare
// mowl.runtime/worker-type; the SDK persists that exact value in each metadata
// contract and rejects an explicit mismatch.
func WithLabels(labels map[string]string) WorkerClientOption
```

WithLabels sets scheduling labels on the worker (for example
matrixorigin.io/compute-resource-id and mowl.runtime/worker-type). These labels
are sent during registration and attach, and used by the scheduler for
compute-resource-based isolation. A worker that registers WorkItems must declare
mowl.runtime/worker-type; the SDK persists that exact value in each metadata
contract and rejects an explicit mismatch.

### WithLogger

```go
// WithLogger sets a custom logger for the SDK client.
// The logger is used for debugging and tracing SDK operations.
//
// Example:
//
//	client, err := moi.New(endpoint, apiKey, moi.WithLogger(myLogger))
func WithLogger(l Logger) Option
```

WithLogger sets a custom logger for the SDK client.
The logger is used for debugging and tracing SDK operations.

Example:

	client, err := moi.New(endpoint, apiKey, moi.WithLogger(myLogger))

### WithMCAccessKeyID

```go
// WithMCAccessKeyID sets the new access key ID when updating a MaxCompute config.
func WithMCAccessKeyID(accessKeyID string) UpdateMCConfigOption
```

WithMCAccessKeyID sets the new access key ID when updating a MaxCompute config.

### WithMCAccessKeySecret

```go
// WithMCAccessKeySecret sets the new access key secret when updating a MaxCompute config.
func WithMCAccessKeySecret(accessKeySecret string) UpdateMCConfigOption
```

WithMCAccessKeySecret sets the new access key secret when updating a MaxCompute config.

### WithMCEndpoint

```go
// WithMCEndpoint sets the new endpoint when updating a MaxCompute config.
func WithMCEndpoint(endpoint string) UpdateMCConfigOption
```

WithMCEndpoint sets the new endpoint when updating a MaxCompute config.

### WithMCName

```go
// WithMCName sets the new name when updating a MaxCompute config.
func WithMCName(name string) UpdateMCConfigOption
```

WithMCName sets the new name when updating a MaxCompute config.

### WithMCProjectName

```go
// WithMCProjectName sets the new project name when updating a MaxCompute config.
func WithMCProjectName(projectName string) UpdateMCConfigOption
```

WithMCProjectName sets the new project name when updating a MaxCompute config.

### WithMCRegion

```go
// WithMCRegion sets the region when creating a MaxCompute config.
func WithMCRegion(region string) CreateMCConfigOption
```

WithMCRegion sets the region when creating a MaxCompute config.

### WithMCUpdatedRegion

```go
// WithMCUpdatedRegion sets the new region when updating a MaxCompute config.
func WithMCUpdatedRegion(region string) UpdateMCConfigOption
```

WithMCUpdatedRegion sets the new region when updating a MaxCompute config.

### WithMaxConcurrentExecuteWorkItems

```go
// WithMaxConcurrentExecuteWorkItems sets the maximum number of work items
// that can be executed concurrently by this worker client.
//
// Example:
//
//	client, _ := moi.New(endpoint, apiKey,
//	    moi.WithGRPCEndpoint("localhost:9090"),
//	    moi.WithWorkerID("worker-1"),
//	)
//	worker := client.Worker(workspaceID, moi.WithMaxConcurrentExecuteWorkItems(5))
func WithMaxConcurrentExecuteWorkItems(n int) WorkerClientOption
```

WithMaxConcurrentExecuteWorkItems sets the maximum number of work items
that can be executed concurrently by this worker client.

Example:

	client, _ := moi.New(endpoint, apiKey,
	    moi.WithGRPCEndpoint("localhost:9090"),
	    moi.WithWorkerID("worker-1"),
	)
	worker := client.Worker(workspaceID, moi.WithMaxConcurrentExecuteWorkItems(5))

### WithMessagesAfter

```go
// WithMessagesAfter lists messages after the given message ID.
func WithMessagesAfter(after int64) ListMessagesOption
```

WithMessagesAfter lists messages after the given message ID.

### WithMessagesLimit

```go
// WithMessagesLimit sets the maximum number of messages to return.
func WithMessagesLimit(limit int32) ListMessagesOption
```

WithMessagesLimit sets the maximum number of messages to return.

### WithMessagesRole

```go
// WithMessagesRole filters by message role.
func WithMessagesRole(role catalog.MessageRole) ListMessagesOption
```

WithMessagesRole filters by message role.

### WithMessagesStatus

```go
// WithMessagesStatus filters by message status.
func WithMessagesStatus(status string) ListMessagesOption
```

WithMessagesStatus filters by message status.

### WithMetadata

```go
// WithMetadata sets the file metadata when uploading a file.
func WithMetadata(metadata map[string]string) UploadOption
```

WithMetadata sets the file metadata when uploading a file.

### WithModifiedResponse

```go
// WithModifiedResponse sets the modified response content.
func WithModifiedResponse(content string) ModifyResponseOption
```

WithModifiedResponse sets the modified response content.

### WithName

```go
// WithName sets the new name when updating a catalog. New names must follow the
// Catalog resource naming contract documented in docs/guide/SDK_GUIDE.md.
//
// Example:
//
//	catalog, err := client.Catalogs().Update(ctx, workspaceID, catalogID, moi.WithName("new-name"))
func WithName(name string) UpdateCatalogOption
```

WithName sets the new name when updating a catalog. New names must follow the
Catalog resource naming contract documented in docs/guide/SDK_GUIDE.md.

Example:

	catalog, err := client.Catalogs().Update(ctx, workspaceID, catalogID, moi.WithName("new-name"))

### WithNotificationConfig

```go
// WithNotificationConfig sets the notification configuration for the task.
func WithNotificationConfig(notification *mowl.NotificationConfig) WorkerTaskOption
```

WithNotificationConfig sets the notification configuration for the task.

### WithNotifyNodeIDs

```go
// WithNotifyNodeIDs filters node notifications to only the specified node IDs.
//
// Example:
//
//	client.Worker(workspaceID).AddNodeNotifyHandler("my-handler", handler,
//	    moi.WithNotifyNodeIDs("node-1", "node-2"),
//	)
func WithNotifyNodeIDs(nodeIDs ...string) NotificationOption
```

WithNotifyNodeIDs filters node notifications to only the specified node IDs.

Example:

	client.Worker(workspaceID).AddNodeNotifyHandler("my-handler", handler,
	    moi.WithNotifyNodeIDs("node-1", "node-2"),
	)

### WithNotifyStates

```go
// WithNotifyStates filters notifications to only the specified states.
// Uses mowl.Status constants for type safety.
//
// Example:
//
//	client.Worker(workspaceID).AddWorkflowNotifyHandler("my-handler", handler,
//	    moi.WithNotifyStates(mowl.StatusCompleted, mowl.StatusFailed),
//	)
func WithNotifyStates(states ...mowl.Status) NotificationOption
```

WithNotifyStates filters notifications to only the specified states.
Uses mowl.Status constants for type safety.

Example:

	client.Worker(workspaceID).AddWorkflowNotifyHandler("my-handler", handler,
	    moi.WithNotifyStates(mowl.StatusCompleted, mowl.StatusFailed),
	)

### WithOrphanFileThreshold

```go
// WithOrphanFileThreshold sets the orphan file threshold in hours.
//
// Example:
//
//	result, err := client.Garbage().TriggerGarbageCollection(ctx, workspaceID, moi.WithOrphanFileThreshold(48))
func WithOrphanFileThreshold(hours int) TriggerGarbageCollectionOption
```

WithOrphanFileThreshold sets the orphan file threshold in hours.

Example:

	result, err := client.Garbage().TriggerGarbageCollection(ctx, workspaceID, moi.WithOrphanFileThreshold(48))

### WithOutputSchema

```go
// WithOutputSchema sets the output schema for a WorkItem.
// The schema must be a valid JSON Schema (draft-07 or higher).
//
// Example:
//
//	schema := `{"type": "object", "properties": {"status": {"type": "integer"}}, "required": ["status"]}`
//	worker.RegisterWorkItemWithOptions("http_request", metadata, handler,
//	    WithOutputSchema(schema))
func WithOutputSchema(schema string) WorkItemOption
```

WithOutputSchema sets the output schema for a WorkItem.
The schema must be a valid JSON Schema (draft-07 or higher).

Example:

	schema := `{"type": "object", "properties": {"status": {"type": "integer"}}, "required": ["status"]}`
	worker.RegisterWorkItemWithOptions("http_request", metadata, handler,
	    WithOutputSchema(schema))

### WithOutputSchemaBuilder

```go
// WithOutputSchemaBuilder sets the output schema using a SchemaBuilder.
// The builder's Build() method will be called automatically.
//
// Example:
//
//	builder := NewSchema().
//	    Property("status", IntegerSchema()).
//	    Required("status")
//	worker.RegisterWorkItemWithOptions("http_request", metadata, handler,
//	    WithOutputSchemaBuilder(builder))
func WithOutputSchemaBuilder(builder *SchemaBuilder) WorkItemOption
```

WithOutputSchemaBuilder sets the output schema using a SchemaBuilder.
The builder's Build() method will be called automatically.

Example:

	builder := NewSchema().
	    Property("status", IntegerSchema()).
	    Required("status")
	worker.RegisterWorkItemWithOptions("http_request", metadata, handler,
	    WithOutputSchemaBuilder(builder))

### WithPageSize

```go
// WithPageSize sets the page size for list operations.
//
// Example:
//
//	catalogs, err := client.Catalogs().List(ctx, moi.WithPageSize(10))
func WithPageSize(size int32) ListOption
```

WithPageSize sets the page size for list operations.

Example:

	catalogs, err := client.Catalogs().List(ctx, moi.WithPageSize(10))

### WithPageToken

```go
// WithPageToken sets the page token for list operations.
// The page token is used for pagination to retrieve the next page of results.
//
// Example:
//
//	catalogs, err := client.Catalogs().List(ctx, moi.WithPageToken("next-page-token"))
func WithPageToken(token string) ListOption
```

WithPageToken sets the page token for list operations.
The page token is used for pagination to retrieve the next page of results.

Example:

	catalogs, err := client.Catalogs().List(ctx, moi.WithPageToken("next-page-token"))

### WithParent

```go
// WithParent filters files by parent directory ID when listing files.
func WithParent(parentID string) ListFilesOption
```

WithParent filters files by parent directory ID when listing files.

### WithParentDirectory

```go
// WithParentDirectory sets the parent directory ID when uploading a file.
func WithParentDirectory(parentID string) UploadOption
```

WithParentDirectory sets the parent directory ID when uploading a file.

### WithParentVolume

```go
// WithParentVolume sets the parent volume ID when creating a child volume.
// This enables hierarchical volume structure.
// Requirements: 9.2
func WithParentVolume(parentID int64) CreateVolumeOption
```

WithParentVolume sets the parent volume ID when creating a child volume.
This enables hierarchical volume structure.
Requirements: 9.2

### WithParseOption

```go
// WithParseOption 添加额外的解析参数（如 layout_xml 等）
func WithParseOption(key, value string) ParseOption
```

WithParseOption 添加额外的解析参数（如 layout_xml 等）

### WithParsePageSelector

```go
// WithParsePageSelector limits parsing to the given 1-based page selector,
// for example "1-5" or "1,3,7-9".
func WithParsePageSelector(selector string) ParseOption
```

WithParsePageSelector limits parsing to the given 1-based page selector,
for example "1-5" or "1,3,7-9".

### WithParseResultBlockType

```go
// WithParseResultBlockType sets the block type of the parse result.
func WithParseResultBlockType(blockType string) ParseResultOption
```

WithParseResultBlockType sets the block type of the parse result.

### WithParseResultDisabled

```go
// WithParseResultDisabled sets whether the parse result is disabled.
func WithParseResultDisabled(disabled bool) ParseResultOption
```

WithParseResultDisabled sets whether the parse result is disabled.

### WithParseResultID

```go
// WithParseResultID sets the ID of the parse result.
func WithParseResultID(id string) ParseResultOption
```

WithParseResultID sets the ID of the parse result.

### WithParseResultIndex

```go
// WithParseResultIndex sets the index of the parse result.
func WithParseResultIndex(index int32) ParseResultOption
```

WithParseResultIndex sets the index of the parse result.

### WithParseResultLevel

```go
// WithParseResultLevel sets the heading level of the parse result.
func WithParseResultLevel(level string) ParseResultOption
```

WithParseResultLevel sets the heading level of the parse result.

### WithParseResultMeta

```go
// WithParseResultMeta sets the metadata of the parse result.
func WithParseResultMeta(meta map[string]interface{}) ParseResultOption
```

WithParseResultMeta sets the metadata of the parse result.

### WithParseResultSourceFiles

```go
// WithParseResultSourceFiles sets the source files of the parse result.
func WithParseResultSourceFiles(sourceFiles map[string]interface{}) ParseResultOption
```

WithParseResultSourceFiles sets the source files of the parse result.

### WithParseResultType

```go
// WithParseResultType sets the result type of the parse result.
func WithParseResultType(resultType string) ParseResultOption
```

WithParseResultType sets the result type of the parse result.

### WithParseResultUpstreamBlocks

```go
// WithParseResultUpstreamBlocks sets the upstream blocks of the parse result.
func WithParseResultUpstreamBlocks(upstreamBlocks map[string]interface{}) ParseResultOption
```

WithParseResultUpstreamBlocks sets the upstream blocks of the parse result.

### WithParseUnderlyingResultType

```go
// WithParseUnderlyingResultType sets the underlying result type of the parse result.
func WithParseUnderlyingResultType(underlyingResultType string) ParseResultOption
```

WithParseUnderlyingResultType sets the underlying result type of the parse result.

### WithParsedManifest

```go
// WithParsedManifest sets manifest struct.
func WithParsedManifest(manifest *structpb.Struct) ParsedManifestOption
```

WithParsedManifest sets manifest struct.

### WithParsedManifestMap

```go
// WithParsedManifestMap sets manifest from map.
func WithParsedManifestMap(manifest map[string]interface{}) ParsedManifestOption
```

WithParsedManifestMap sets manifest from map.

### WithParsedManifestParsedAssetID

```go
// WithParsedManifestParsedAssetID sets parsed_asset_id.
func WithParsedManifestParsedAssetID(parsedAssetID string) ParsedManifestOption
```

WithParsedManifestParsedAssetID sets parsed_asset_id.

### WithParserBackendAPIKey

```go
// WithParserBackendAPIKey sets the encrypted API key.
func WithParserBackendAPIKey(key string) CreateParserBackendOption
```

WithParserBackendAPIKey sets the encrypted API key.

### WithParserBackendAPIKeys

```go
// WithParserBackendAPIKeys sets the encrypted API key pool.
func WithParserBackendAPIKeys(keys []string) CreateParserBackendOption
```

WithParserBackendAPIKeys sets the encrypted API key pool.

### WithParserBackendMIMETypes

```go
// WithParserBackendMIMETypes sets the supported MIME types.
func WithParserBackendMIMETypes(mimeTypes []string) CreateParserBackendOption
```

WithParserBackendMIMETypes sets the supported MIME types.

### WithParserBackendName

```go
// WithParserBackendName sets the backend name.
func WithParserBackendName(name string) CreateParserBackendOption
```

WithParserBackendName sets the backend name.

### WithParserBackendTimeout

```go
// WithParserBackendTimeout sets the timeout in seconds.
func WithParserBackendTimeout(seconds int32) CreateParserBackendOption
```

WithParserBackendTimeout sets the timeout in seconds.

### WithParserBackendType

```go
// WithParserBackendType sets the backend type.
func WithParserBackendType(t catalog.ParserBackendType) CreateParserBackendOption
```

WithParserBackendType sets the backend type.

### WithPrefix

```go
// WithPrefix filters files by path prefix when listing files.
func WithPrefix(prefix string) ListFilesOption
```

WithPrefix filters files by path prefix when listing files.

### WithPreserveCatalogId

```go
// WithPreserveCatalogId instructs SyncMetadata not to overwrite the catalog_id
// of an existing database. Use this for background periodic/DDL discovery sync
// to avoid overriding user-assigned catalog membership.
// When set, the service will only assign catalog_id for newly discovered databases;
// existing databases retain their current catalog assignment.
func WithPreserveCatalogId() SyncMetadataOption
```

WithPreserveCatalogId instructs SyncMetadata not to overwrite the catalog_id
of an existing database. Use this for background periodic/DDL discovery sync
to avoid overriding user-assigned catalog membership.
When set, the service will only assign catalog_id for newly discovered databases;
existing databases retain their current catalog assignment.

### WithProxyExtension

```go
// WithProxyExtension sets the proxy extension sent in the request body under the "proxy" key.
// ProxyExtension 提供的字段（与 proto/catalog.ProxyExtension 一致）：
//   - RecordMessage: 是否记录消息
//   - SessionId: 会话 ID
//   - Source: 来源（如 "cli"）
//   - Role: 消息角色（MessageRole_USER / SYSTEM / ASSISTANT / AGENT_TOOL）
//   - OriginalContent: 原始内容
//   - Config: 配置 JSON
//   - MockResponse: 仅 dev-llm 生效，用于测试时直接返回该内容
//
// Example:
//
//	ch, err := client.LLM(ws).ChatCompletion(ctx, "hi", "gpt-4",
//	    moi.WithProxyExtension(&catalog.ProxyExtension{
//	        Source: "cli", SessionId: 123, RecordMessage: true,
//	    }),
//	)
func WithProxyExtension(ext *catalog.ProxyExtension) ChatCompletionOption
```

WithProxyExtension sets the proxy extension sent in the request body under the "proxy" key.
ProxyExtension 提供的字段（与 proto/catalog.ProxyExtension 一致）：
  - RecordMessage: 是否记录消息
  - SessionId: 会话 ID
  - Source: 来源（如 "cli"）
  - Role: 消息角色（MessageRole_USER / SYSTEM / ASSISTANT / AGENT_TOOL）
  - OriginalContent: 原始内容
  - Config: 配置 JSON
  - MockResponse: 仅 dev-llm 生效，用于测试时直接返回该内容

Example:

	ch, err := client.LLM(ws).ChatCompletion(ctx, "hi", "gpt-4",
	    moi.WithProxyExtension(&catalog.ProxyExtension{
	        Source: "cli", SessionId: 123, RecordMessage: true,
	    }),
	)

### WithRelationTagName

```go
// WithRelationTagName sets the tag name for a relation operation.
func WithRelationTagName(name string) TagRelationOption
```

WithRelationTagName sets the tag name for a relation operation.

### WithRelationTagSource

```go
// WithRelationTagSource sets the tag source for a relation operation.
func WithRelationTagSource(source string) TagRelationOption
```

WithRelationTagSource sets the tag source for a relation operation.

### WithRequestModifier

```go
// WithRequestModifier sets a request modifier function that will be called
// for each HTTP request before it's sent. This allows customization of requests
// such as adding custom headers or modifying the URL. Authentication headers
// and Cookie remain SDK-owned: human PAT clients send only X-API-Key and
// service-account Bearer clients send only Authorization: Bearer.
//
// Example:
//
//	modifier := func(req *http.Request) {
//	    req.Header.Set("X-Test-ID", "test-123")
//	    req.Header.Set("X-Trace-ID", "trace-456")
//	}
//	client, err := moi.New(endpoint, apiKey, moi.WithRequestModifier(modifier))
func WithRequestModifier(modifier func(*http.Request)) Option
```

WithRequestModifier sets a request modifier function that will be called
for each HTTP request before it's sent. This allows customization of requests
such as adding custom headers or modifying the URL. Authentication headers
and Cookie remain SDK-owned: human PAT clients send only X-API-Key and
service-account Bearer clients send only Authorization: Bearer.

Example:

	modifier := func(req *http.Request) {
	    req.Header.Set("X-Test-ID", "test-123")
	    req.Header.Set("X-Trace-ID", "trace-456")
	}
	client, err := moi.New(endpoint, apiKey, moi.WithRequestModifier(modifier))

### WithRequireUnlinked

```go
// WithRequireUnlinked rejects files already associated with another Volume.
// A file already associated with the target Volume remains an idempotent no-op.
// This option does not prove uploader identity or act as an upload capability.
func WithRequireUnlinked() AddFilesOption
```

WithRequireUnlinked rejects files already associated with another Volume.
A file already associated with the target Volume remains an idempotent no-op.
This option does not prove uploader identity or act as an upload capability.

### WithResolveAssetID

```go
// WithResolveAssetID sets asset_id.
func WithResolveAssetID(assetID string) DataAssetResolveOption
```

WithResolveAssetID sets asset_id.

### WithResolveAssetIdentity

```go
// WithResolveAssetIdentity sets asset_type and asset_ref.
func WithResolveAssetIdentity(assetType, assetRef string) DataAssetResolveOption
```

WithResolveAssetIdentity sets asset_type and asset_ref.

### WithResolveRawFileID

```go
// WithResolveRawFileID sets raw_file_id.
func WithResolveRawFileID(rawFileID string) DataAssetResolveOption
```

WithResolveRawFileID sets raw_file_id.

### WithResponseHeaderTimeout

```go
// WithResponseHeaderTimeout sets the timeout for waiting until the server sends
// response headers on regular HTTP requests. This is useful for long-running
// synchronous endpoints whose total request timeout must be much longer than
// the SDK default 60s time-to-first-byte defense.
//
// Streaming requests keep their dedicated stream client and are not affected.
func WithResponseHeaderTimeout(d time.Duration) Option
```

WithResponseHeaderTimeout sets the timeout for waiting until the server sends
response headers on regular HTTP requests. This is useful for long-running
synchronous endpoints whose total request timeout must be much longer than
the SDK default 60s time-to-first-byte defense.

Streaming requests keep their dedicated stream client and are not affected.

### WithResultMode

```go
// WithResultMode sets the result mode for dynamic services.
// Valid values: "oneshot" (default) or "stream".
// This option only applies when WithDynamicService is also used.
func WithResultMode(mode string) WorkerTaskOption
```

WithResultMode sets the result mode for dynamic services.
Valid values: "oneshot" (default) or "stream".
This option only applies when WithDynamicService is also used.

### WithRouterHealthCheckInterval

```go
// WithRouterHealthCheckInterval sets the health check interval in seconds.
func WithRouterHealthCheckInterval(seconds int32) PutRouterConfigOption
```

WithRouterHealthCheckInterval sets the health check interval in seconds.

### WithRouterMaxRetries

```go
// WithRouterMaxRetries sets the maximum number of retries.
func WithRouterMaxRetries(n int32) PutRouterConfigOption
```

WithRouterMaxRetries sets the maximum number of retries.

### WithRouterSessionAffinity

```go
// WithRouterSessionAffinity enables or disables session affinity.
func WithRouterSessionAffinity(enabled bool) PutRouterConfigOption
```

WithRouterSessionAffinity enables or disables session affinity.

### WithRouterStrategy

```go
// WithRouterStrategy sets the routing strategy.
func WithRouterStrategy(strategy catalog.RouterStrategy) PutRouterConfigOption
```

WithRouterStrategy sets the routing strategy.

### WithRuntimeConfigContractProviderV2

```go
// WithRuntimeConfigContractProviderV2 configures version-aware runtime config contract lookup.
func WithRuntimeConfigContractProviderV2(provider RuntimeConfigContractProviderV2) RuntimeConfigValidationOption
```

WithRuntimeConfigContractProviderV2 configures version-aware runtime config contract lookup.

### WithSchemas

```go
// WithSchemas sets both input and output schemas for a WorkItem.
// This is a convenience function equivalent to calling WithInputSchema and WithOutputSchema.
//
// Example:
//
//	inputSchema := `{"type": "object", "properties": {"url": {"type": "string"}}}`
//	outputSchema := `{"type": "object", "properties": {"status": {"type": "integer"}}}`
//	worker.RegisterWorkItemWithOptions("http_request", metadata, handler,
//	    WithSchemas(inputSchema, outputSchema))
func WithSchemas(inputSchema, outputSchema string) WorkItemOption
```

WithSchemas sets both input and output schemas for a WorkItem.
This is a convenience function equivalent to calling WithInputSchema and WithOutputSchema.

Example:

	inputSchema := `{"type": "object", "properties": {"url": {"type": "string"}}}`
	outputSchema := `{"type": "object", "properties": {"status": {"type": "integer"}}}`
	worker.RegisterWorkItemWithOptions("http_request", metadata, handler,
	    WithSchemas(inputSchema, outputSchema))

### WithScopes

```go
// WithScopes sets the permission scopes when creating an API key.
// Scopes define what operations the API key is allowed to perform.
//
// Example:
//
//	apiKey, err := client.APIKeys().Create(ctx, "my-api-key", moi.WithScopes("read", "write"))
func WithScopes(scopes ...string) CreateAPIKeyOption
```

WithScopes sets the permission scopes when creating an API key.
Scopes define what operations the API key is allowed to perform.

Example:

	apiKey, err := client.APIKeys().Create(ctx, "my-api-key", moi.WithScopes("read", "write"))

### WithSearch

```go
// WithSearch sets a fuzzy search keyword for list operations.
// The search keyword is matched against name and description fields.
//
// Example:
//
//	models, err := client.SemanticModels(wsID).List(ctx, moi.WithSearch("sales"))
func WithSearch(keyword string) ListOption
```

WithSearch sets a fuzzy search keyword for list operations.
The search keyword is matched against name and description fields.

Example:

	models, err := client.SemanticModels(wsID).List(ctx, moi.WithSearch("sales"))

### WithSessionConfig

```go
// WithSessionConfig sets the session config when creating a session.
func WithSessionConfig(config string) CreateSessionOption
```

WithSessionConfig sets the session config when creating a session.

### WithSessionConfigUpdate

```go
// WithSessionConfigUpdate sets the new config when updating a session.
func WithSessionConfigUpdate(config string) UpdateSessionOption
```

WithSessionConfigUpdate sets the new config when updating a session.

### WithSessionSource

```go
// WithSessionSource sets the session source when creating a session.
func WithSessionSource(source string) CreateSessionOption
```

WithSessionSource sets the session source when creating a session.

### WithSessionTitle

```go
// WithSessionTitle sets the new title when updating a session.
func WithSessionTitle(title string) UpdateSessionOption
```

WithSessionTitle sets the new title when updating a session.

### WithSessionsKeyword

```go
// WithSessionsKeyword filters sessions by keyword (searches session title).
//
// Example:
//
//	resp, err := client.LLM(workspaceID).ListSessions(ctx,
//	    moi.WithSessionsKeyword("finance"),
//	)
func WithSessionsKeyword(keyword string) ListSessionsOption
```

WithSessionsKeyword filters sessions by keyword (searches session title).

Example:

	resp, err := client.LLM(workspaceID).ListSessions(ctx,
	    moi.WithSessionsKeyword("finance"),
	)

### WithSessionsPage

```go
// WithSessionsPage sets page for list sessions.
func WithSessionsPage(page int32) ListSessionsOption
```

WithSessionsPage sets page for list sessions.

### WithSessionsPageSize

```go
// WithSessionsPageSize sets page size for list sessions.
func WithSessionsPageSize(pageSize int32) ListSessionsOption
```

WithSessionsPageSize sets page size for list sessions.

### WithSessionsSource

```go
// WithSessionsSource filters sessions by source.
func WithSessionsSource(source string) ListSessionsOption
```

WithSessionsSource filters sessions by source.

### WithSessionsTag

```go
// WithSessionsTag filters sessions by a single tag name.
//
// Example:
//
//	resp, err := client.LLM(workspaceID).ListSessions(ctx,
//	    moi.WithSessionsTag("important"),
//	)
func WithSessionsTag(tag string) ListSessionsOption
```

WithSessionsTag filters sessions by a single tag name.

Example:

	resp, err := client.LLM(workspaceID).ListSessions(ctx,
	    moi.WithSessionsTag("important"),
	)

### WithSyncComment

```go
// WithSyncComment sets the database comment during metadata sync.
// Pass a pointer to empty string to clear the comment.
// If not set, the comment is not modified (backward compatible).
//
// Example:
//
//	comment := "my database description"
//	resp, err := client.Databases().SyncMetadata(ctx, wsID, "my_db", catalogID, moi.WithSyncComment(&comment))
func WithSyncComment(comment *string) SyncMetadataOption
```

WithSyncComment sets the database comment during metadata sync.
Pass a pointer to empty string to clear the comment.
If not set, the comment is not modified (backward compatible).

Example:

	comment := "my database description"
	resp, err := client.Databases().SyncMetadata(ctx, wsID, "my_db", catalogID, moi.WithSyncComment(&comment))

### WithTableCreateIAM

```go
// WithTableCreateIAM marks a user-initiated table creation. Core validates the
// new table name against the Catalog identifier contract, verifies the coarse
// create action on the parent Database, and registers only the new Table's
// Direct Owner. MatrixOne remains the DDL/data permission authority.
func WithTableCreateIAM(databaseID int64, tableName string) SyncMetadataOption
```

WithTableCreateIAM marks a user-initiated table creation. Core validates the
new table name against the Catalog identifier contract, verifies the coarse
create action on the parent Database, and registers only the new Table's
Direct Owner. MatrixOne remains the DDL/data permission authority.

### WithTag

```go
// WithTag appends a tag filter for list operations that support tag filtering.
func WithTag(tag string) ListOption
```

WithTag appends a tag filter for list operations that support tag filtering.

### WithTagName

```go
// WithTagName sets the tag name.
func WithTagName(name string) CreateTagOption
```

WithTagName sets the tag name.

### WithTagSource

```go
// WithTagSource sets the tag source.
func WithTagSource(source string) CreateTagOption
```

WithTagSource sets the tag source.

### WithTagsKeyword

```go
// WithTagsKeyword filters tags by keyword.
func WithTagsKeyword(keyword string) ListTagsOption
```

WithTagsKeyword filters tags by keyword.

### WithTagsPage

```go
// WithTagsPage sets page for list tags.
func WithTagsPage(page int32) ListTagsOption
```

WithTagsPage sets page for list tags.

### WithTagsPageSize

```go
// WithTagsPageSize sets page size for list tags.
func WithTagsPageSize(pageSize int32) ListTagsOption
```

WithTagsPageSize sets page size for list tags.

### WithTagsSource

```go
// WithTagsSource filters tags by source.
func WithTagsSource(source string) ListTagsOption
```

WithTagsSource filters tags by source.

### WithTaskCronExpression

```go
// WithTaskCronExpression sets the cron expression for periodic task execution.
// If not set, the task will be executed once immediately.
//
// Example:
//
//	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
//	    moi.WithTaskWorkflowVersionID("version-uuid"),
//	    moi.WithTaskCronExpression("0 0 * * *"), // Run daily at midnight
//	)
func WithTaskCronExpression(cron string) CreateTaskOption
```

WithTaskCronExpression sets the cron expression for periodic task execution.
If not set, the task will be executed once immediately.

Example:

	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
	    moi.WithTaskWorkflowVersionID("version-uuid"),
	    moi.WithTaskCronExpression("0 0 * * *"), // Run daily at midnight
	)

### WithTaskData

```go
// WithTaskData sets the initial data (JSON) for the task execution.
//
// Example:
//
//	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
//	    moi.WithTaskWorkflowVersionID("version-uuid"),
//	    moi.WithTaskData(`{"input": "value"}`),
//	)
func WithTaskData(data string) CreateTaskOption
```

WithTaskData sets the initial data (JSON) for the task execution.

Example:

	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
	    moi.WithTaskWorkflowVersionID("version-uuid"),
	    moi.WithTaskData(`{"input": "value"}`),
	)

### WithTaskID

```go
// WithTaskID filters notifications to only the specified task.
func WithTaskID(taskID string) NotificationOption
```

WithTaskID filters notifications to only the specified task.

### WithTaskNotification

```go
// WithTaskNotification sets the notification configuration for the task.
//
// Example:
//
//	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
//	    moi.WithTaskWorkflowVersionID("version-uuid"),
//	    moi.WithTaskNotification(moi.NewHTTPNotification("https://callback.example.com")),
//	)
func WithTaskNotification(notification *mowl.NotificationConfig) CreateTaskOption
```

WithTaskNotification sets the notification configuration for the task.

Example:

	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
	    moi.WithTaskWorkflowVersionID("version-uuid"),
	    moi.WithTaskNotification(moi.NewHTTPNotification("https://callback.example.com")),
	)

### WithTaskPeriodicOnly

```go
// WithTaskPeriodicOnly filters to show only periodic tasks.
//
// Example:
//
//	tasks, err := client.Tasks(workspaceID).List(ctx, moi.WithTaskPeriodicOnly(true))
func WithTaskPeriodicOnly(periodicOnly bool) ListTasksOption
```

WithTaskPeriodicOnly filters to show only periodic tasks.

Example:

	tasks, err := client.Tasks(workspaceID).List(ctx, moi.WithTaskPeriodicOnly(true))

### WithTaskRuntimeSpecJson

```go
// WithTaskRuntimeSpecJson sets the runtime spec JSON for dynamic worker configuration.
// When set, the server will launch dynamic workers before executing the task.
//
// Example:
//
//	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
//	    moi.WithTaskWorkflowVersionID("version-uuid"),
//	    moi.WithTaskRuntimeSpecJson(`{"workers": [{"worker_id": "my-worker", "source": {"type": "image", "image": {"repository": "my-image:latest"}}}]}`),
//	)
func WithTaskRuntimeSpecJson(runtimeSpecJson string) CreateTaskOption
```

WithTaskRuntimeSpecJson sets the runtime spec JSON for dynamic worker configuration.
When set, the server will launch dynamic workers before executing the task.

Example:

	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
	    moi.WithTaskWorkflowVersionID("version-uuid"),
	    moi.WithTaskRuntimeSpecJson(`{"workers": [{"worker_id": "my-worker", "source": {"type": "image", "image": {"repository": "my-image:latest"}}}]}`),
	)

### WithTaskStatus

```go
// WithTaskStatus filters tasks by status.
// Status values: 0=active, 1=completed, 2=cancelled, -1=all
//
// Example:
//
//	tasks, err := client.Tasks(workspaceID).List(ctx, moi.WithTaskStatus(0)) // Active tasks only
func WithTaskStatus(status int32) ListTasksOption
```

WithTaskStatus filters tasks by status.
Status values: 0=active, 1=completed, 2=cancelled, -1=all

Example:

	tasks, err := client.Tasks(workspaceID).List(ctx, moi.WithTaskStatus(0)) // Active tasks only

### WithTaskTraceOptions

```go
// WithTaskTraceOptions sets trace options for task execution.
func WithTaskTraceOptions(trace *mowl.TraceOptions) CreateTaskOption
```

WithTaskTraceOptions sets trace options for task execution.

### WithTaskTransient

```go
// WithTaskTransient sets the transient flag for the task.
// Transient tasks don't persist their execution state.
//
// Example:
//
//	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
//	    moi.WithTaskWorkflowVersionID("version-uuid"),
//	    moi.WithTaskTransient(true),
//	)
func WithTaskTransient(transient bool) CreateTaskOption
```

WithTaskTransient sets the transient flag for the task.
Transient tasks don't persist their execution state.

Example:

	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
	    moi.WithTaskWorkflowVersionID("version-uuid"),
	    moi.WithTaskTransient(true),
	)

### WithTaskVars

```go
// WithTaskVars sets the initial variables (JSON) for the task execution.
//
// Example:
//
//	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
//	    moi.WithTaskWorkflowVersionID("version-uuid"),
//	    moi.WithTaskVars(`{"env": "production"}`),
//	)
func WithTaskVars(vars string) CreateTaskOption
```

WithTaskVars sets the initial variables (JSON) for the task execution.

Example:

	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
	    moi.WithTaskWorkflowVersionID("version-uuid"),
	    moi.WithTaskVars(`{"env": "production"}`),
	)

### WithTaskWorkflow

```go
// WithTaskWorkflow sets the embedded workflow definition when creating a task.
// Use this for ad-hoc workflow execution without storing the workflow definition.
//
// Example:
//
//	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
//	    moi.WithTaskWorkflow(workflowYAML),
//	)
func WithTaskWorkflow(workflow string) CreateTaskOption
```

WithTaskWorkflow sets the embedded workflow definition when creating a task.
Use this for ad-hoc workflow execution without storing the workflow definition.

Example:

	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
	    moi.WithTaskWorkflow(workflowYAML),
	)

### WithTaskWorkflowVersionID

```go
// WithTaskWorkflowVersionID sets the workflow version ID when creating a task.
// Use this to execute a stored workflow version.
//
// Example:
//
//	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
//	    moi.WithTaskWorkflowVersionID("version-uuid"),
//	)
func WithTaskWorkflowVersionID(versionID string) CreateTaskOption
```

WithTaskWorkflowVersionID sets the workflow version ID when creating a task.
Use this to execute a stored workflow version.

Example:

	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
	    moi.WithTaskWorkflowVersionID("version-uuid"),
	)

### WithTimeout

```go
// WithTimeout sets the request timeout for the client.
// The timeout applies to each individual HTTP request.
//
// Example:
//
//	client, err := moi.New(endpoint, apiKey, moi.WithTimeout(60*time.Second))
func WithTimeout(d time.Duration) Option
```

WithTimeout sets the request timeout for the client.
The timeout applies to each individual HTTP request.

Example:

	client, err := moi.New(endpoint, apiKey, moi.WithTimeout(60*time.Second))

### WithTraceOptions

```go
// WithTraceOptions sets trace options for task execution.
func WithTraceOptions(trace *mowl.TraceOptions) WorkerTaskOption
```

WithTraceOptions sets trace options for task execution.

### WithTransient

```go
// WithTransient sets the transient flag for the task.
func WithTransient(transient bool) WorkerTaskOption
```

WithTransient sets the transient flag for the task.

### WithTrustedWorkerCredentials

```go
// WithTrustedWorkerCredentials attaches trusted-worker metadata to worker
// registration and WorkerSession calls. The server validates the credentials
// before allowing reserved WorkItem namespaces.
func WithTrustedWorkerCredentials(profile, tokenID, token string) WorkerClientOption
```

WithTrustedWorkerCredentials attaches trusted-worker metadata to worker
registration and WorkerSession calls. The server validates the credentials
before allowing reserved WorkItem namespaces.

### WithUpdateBackendAPIKey

```go
// WithUpdateBackendAPIKey sets the new encrypted API key.
func WithUpdateBackendAPIKey(key string) UpdateBackendOption
```

WithUpdateBackendAPIKey sets the new encrypted API key.

### WithUpdateBackendModels

```go
// WithUpdateBackendModels sets the new supported model list.
func WithUpdateBackendModels(models []string) UpdateBackendOption
```

WithUpdateBackendModels sets the new supported model list.

### WithUpdateBackendName

```go
// WithUpdateBackendName sets the new backend name.
func WithUpdateBackendName(name string) UpdateBackendOption
```

WithUpdateBackendName sets the new backend name.

### WithUpdateBackendReasoningControlProtocol

```go
// WithUpdateBackendReasoningControlProtocol updates the backend reasoning
// control protocol.
func WithUpdateBackendReasoningControlProtocol(protocol catalog.ReasoningControlProtocol) UpdateBackendOption
```

WithUpdateBackendReasoningControlProtocol updates the backend reasoning
control protocol.

### WithUpdateBackendTimeout

```go
// WithUpdateBackendTimeout sets the new timeout in seconds.
func WithUpdateBackendTimeout(seconds int32) UpdateBackendOption
```

WithUpdateBackendTimeout sets the new timeout in seconds.

### WithUpdateParserBackendAPIKey

```go
// WithUpdateParserBackendAPIKey sets the new encrypted API key.
func WithUpdateParserBackendAPIKey(key string) UpdateParserBackendOption
```

WithUpdateParserBackendAPIKey sets the new encrypted API key.

### WithUpdateParserBackendAPIKeys

```go
// WithUpdateParserBackendAPIKeys sets the new encrypted API key pool.
func WithUpdateParserBackendAPIKeys(keys []string) UpdateParserBackendOption
```

WithUpdateParserBackendAPIKeys sets the new encrypted API key pool.

### WithUpdateParserBackendMIMETypes

```go
// WithUpdateParserBackendMIMETypes sets the new supported MIME types.
func WithUpdateParserBackendMIMETypes(mimeTypes []string) UpdateParserBackendOption
```

WithUpdateParserBackendMIMETypes sets the new supported MIME types.

### WithUpdateParserBackendName

```go
// WithUpdateParserBackendName sets the new backend name.
func WithUpdateParserBackendName(name string) UpdateParserBackendOption
```

WithUpdateParserBackendName sets the new backend name.

### WithUpdateParserBackendTimeout

```go
// WithUpdateParserBackendTimeout sets the new timeout in seconds.
func WithUpdateParserBackendTimeout(seconds int32) UpdateParserBackendOption
```

WithUpdateParserBackendTimeout sets the new timeout in seconds.

### WithUpdatedComment

```go
// WithUpdatedComment sets the new comment when updating a catalog.
// This is named differently from WithComment to avoid confusion with the create option.
//
// Example:
//
//	catalog, err := client.Catalogs().Update(ctx, workspaceID, catalogID, moi.WithUpdatedComment("Updated comment"))
func WithUpdatedComment(comment string) UpdateCatalogOption
```

WithUpdatedComment sets the new comment when updating a catalog.
This is named differently from WithComment to avoid confusion with the create option.

Example:

	catalog, err := client.Catalogs().Update(ctx, workspaceID, catalogID, moi.WithUpdatedComment("Updated comment"))

### WithUpdatedVolumeComment

```go
// WithUpdatedVolumeComment sets the new comment when updating a volume.
func WithUpdatedVolumeComment(comment string) UpdateVolumeOption
```

WithUpdatedVolumeComment sets the new comment when updating a volume.

### WithUpdatedWorkflowDefDescription

```go
// WithUpdatedWorkflowDefDescription sets the new description when updating a workflow definition.
//
// Example:
//
//	err := client.Workflows(workspaceID).Update(ctx, "workflow-uuid",
//	    moi.WithUpdatedWorkflowDefDescription("Updated description"),
//	)
func WithUpdatedWorkflowDefDescription(description string) UpdateWorkflowDefOption
```

WithUpdatedWorkflowDefDescription sets the new description when updating a workflow definition.

Example:

	err := client.Workflows(workspaceID).Update(ctx, "workflow-uuid",
	    moi.WithUpdatedWorkflowDefDescription("Updated description"),
	)

### WithUpdatedWorkflowDefName

```go
// WithUpdatedWorkflowDefName sets the new name when updating a workflow definition.
//
// Example:
//
//	err := client.Workflows(workspaceID).Update(ctx, "workflow-uuid",
//	    moi.WithUpdatedWorkflowDefName("new-workflow-name"),
//	)
func WithUpdatedWorkflowDefName(name string) UpdateWorkflowDefOption
```

WithUpdatedWorkflowDefName sets the new name when updating a workflow definition.

Example:

	err := client.Workflows(workspaceID).Update(ctx, "workflow-uuid",
	    moi.WithUpdatedWorkflowDefName("new-workflow-name"),
	)

### WithUpdatedWorkspaceDescription

```go
// WithUpdatedWorkspaceDescription sets the new description when updating a workspace.
// This is named differently from WithWorkspaceDescription to avoid confusion with the create option.
//
// Example:
//
//	ws, err := client.Workspaces().Update(ctx, workspaceID, moi.WithUpdatedWorkspaceDescription("Updated description"))
func WithUpdatedWorkspaceDescription(description string) UpdateWorkspaceOption
```

WithUpdatedWorkspaceDescription sets the new description when updating a workspace.
This is named differently from WithWorkspaceDescription to avoid confusion with the create option.

Example:

	ws, err := client.Workspaces().Update(ctx, workspaceID, moi.WithUpdatedWorkspaceDescription("Updated description"))

### WithUserID

```go
// WithUserID sets the user ID for whom the API key should be created.
// This option is only effective when the caller is the system user.
// Regular users cannot create API keys for other users.
//
// Example:
//
//	apiKey, err := systemClient.APIKeys().Create(ctx, "user-api-key", moi.WithUserID("user-123"))
func WithUserID(userID string) CreateAPIKeyOption
```

WithUserID sets the user ID for whom the API key should be created.
This option is only effective when the caller is the system user.
Regular users cannot create API keys for other users.

Example:

	apiKey, err := systemClient.APIKeys().Create(ctx, "user-api-key", moi.WithUserID("user-123"))

### WithUserNickname

```go
// WithUserNickname sets the new nickname when updating a user.
//
// Example:
//
//	user, err := client.Users().Update(ctx, userID, moi.WithUserNickname("New Name"))
func WithUserNickname(nickname string) UpdateUserOption
```

WithUserNickname sets the new nickname when updating a user.

Example:

	user, err := client.Users().Update(ctx, userID, moi.WithUserNickname("New Name"))

### WithUserPhone

```go
// WithUserPhone sets the new phone when updating a user.
//
// Example:
//
//	user, err := client.Users().Update(ctx, userID, moi.WithUserPhone("+1234567890"))
func WithUserPhone(phone string) UpdateUserOption
```

WithUserPhone sets the new phone when updating a user.

Example:

	user, err := client.Users().Update(ctx, userID, moi.WithUserPhone("+1234567890"))

### WithUserStatus

```go
// WithUserStatus sets the new status when updating a user.
//
// Example:
//
//	user, err := client.Users().Update(ctx, userID, moi.WithUserStatus(user.UserStatus_USER_STATUS_ACTIVE))
func WithUserStatus(status user.UserStatus) UpdateUserOption
```

WithUserStatus sets the new status when updating a user.

Example:

	user, err := client.Users().Update(ctx, userID, moi.WithUserStatus(user.UserStatus_USER_STATUS_ACTIVE))

### WithVars

```go
// WithVars sets the initial variables (JSON) for the task execution.
func WithVars(vars string) WorkerTaskOption
```

WithVars sets the initial variables (JSON) for the task execution.

### WithVersionDescription

```go
// WithVersionDescription sets the description for the workflow version.
func WithVersionDescription(desc string) CreateWorkflowVersionOption
```

WithVersionDescription sets the description for the workflow version.

### WithVersionDynamicService

```go
// WithVersionDynamicService marks the workflow version as a dynamic service with the specified schemas.
// inputSchema and outputSchema are required for dynamic services.
//
// Example:
//
//	inputSchema := moi.NewSchema().Property("url", moi.StringSchema()).Required("url")
//	outputSchema := moi.NewSchema().Property("result", moi.StringSchema())
//	version, err := client.WorkflowVersions(workspaceID).CreateByBuilder(ctx, "workflow-uuid", builder,
//	    moi.WithVersionDynamicService(inputSchema, outputSchema),
//	    moi.WithVersionResultMode(mowl.ResultMode_RESULT_MODE_ONESHOT),
//	)
func WithVersionDynamicService(inputSchema, outputSchema *SchemaBuilder) CreateWorkflowVersionOption
```

WithVersionDynamicService marks the workflow version as a dynamic service with the specified schemas.
inputSchema and outputSchema are required for dynamic services.

Example:

	inputSchema := moi.NewSchema().Property("url", moi.StringSchema()).Required("url")
	outputSchema := moi.NewSchema().Property("result", moi.StringSchema())
	version, err := client.WorkflowVersions(workspaceID).CreateByBuilder(ctx, "workflow-uuid", builder,
	    moi.WithVersionDynamicService(inputSchema, outputSchema),
	    moi.WithVersionResultMode(mowl.ResultMode_RESULT_MODE_ONESHOT),
	)

### WithVersionNumber

```go
// WithVersionNumber specifies the version number to invoke.
// If not specified or set to 0, the latest published version will be used.
//
// Example:
//
//	result, err := worker.InvokeDynamicServiceSync(ctx, "image-processor",
//	    `{"url": "https://..."}`,
//	    moi.WithVersionNumber(1),
//	)
func WithVersionNumber(version int32) InvokeOption
```

WithVersionNumber specifies the version number to invoke.
If not specified or set to 0, the latest published version will be used.

Example:

	result, err := worker.InvokeDynamicServiceSync(ctx, "image-processor",
	    `{"url": "https://..."}`,
	    moi.WithVersionNumber(1),
	)

### WithVersionResultMode

```go
// WithVersionResultMode sets the result mode for dynamic service versions.
// Valid values: ResultMode_RESULT_MODE_ONESHOT (default) or ResultMode_RESULT_MODE_STREAM.
func WithVersionResultMode(mode mowl.ResultMode) CreateWorkflowVersionOption
```

WithVersionResultMode sets the result mode for dynamic service versions.
Valid values: ResultMode_RESULT_MODE_ONESHOT (default) or ResultMode_RESULT_MODE_STREAM.

### WithVersionRuntimeSpecJson

```go
// WithVersionRuntimeSpecJson sets the runtime spec JSON for dynamic worker configuration.
// When set, publishing the version will launch dynamic workers as declared in the spec.
func WithVersionRuntimeSpecJson(json string) CreateWorkflowVersionOption
```

WithVersionRuntimeSpecJson sets the runtime spec JSON for dynamic worker configuration.
When set, publishing the version will launch dynamic workers as declared in the spec.

### WithVolumeComment

```go
// WithVolumeComment sets the volume comment when creating a volume.
func WithVolumeComment(comment string) CreateVolumeOption
```

WithVolumeComment sets the volume comment when creating a volume.

### WithVolumeFilesFilter

```go
// WithVolumeFilesFilter adds a filter condition for list volume files operations.
// Supported fields: file_id, file_name, file_path
func WithVolumeFilesFilter(field, value string) VolumeFileListOption
```

WithVolumeFilesFilter adds a filter condition for list volume files operations.
Supported fields: file_id, file_name, file_path

### WithVolumeFilesFuzzyFilter

```go
// WithVolumeFilesFuzzyFilter adds a fuzzy filter condition for list volume files operations.
// Supported fields: file_id, file_name, file_path
func WithVolumeFilesFuzzyFilter(field, value string) VolumeFileListOption
```

WithVolumeFilesFuzzyFilter adds a fuzzy filter condition for list volume files operations.
Supported fields: file_id, file_name, file_path

### WithVolumeFilesOrder

```go
// WithVolumeFilesOrder sets the order direction for list volume files operations ("asc" or "desc").
func WithVolumeFilesOrder(order string) VolumeFileListOption
```

WithVolumeFilesOrder sets the order direction for list volume files operations ("asc" or "desc").

### WithVolumeFilesOrderBy

```go
// WithVolumeFilesOrderBy sets the order by field for list volume files operations.
func WithVolumeFilesOrderBy(field string) VolumeFileListOption
```

WithVolumeFilesOrderBy sets the order by field for list volume files operations.

### WithVolumeFilesPageSize

```go
// WithVolumeFilesPageSize sets the page size for list volume files operations.
func WithVolumeFilesPageSize(size int32) VolumeFileListOption
```

WithVolumeFilesPageSize sets the page size for list volume files operations.

### WithVolumeFilesPageToken

```go
// WithVolumeFilesPageToken sets the page token for list volume files operations.
func WithVolumeFilesPageToken(token string) VolumeFileListOption
```

WithVolumeFilesPageToken sets the page token for list volume files operations.

### WithVolumeName

```go
// WithVolumeName sets the new name when updating a volume. New names must
// follow the Catalog resource naming contract documented in
// docs/guide/SDK_GUIDE.md.
func WithVolumeName(name string) UpdateVolumeOption
```

WithVolumeName sets the new name when updating a volume. New names must
follow the Catalog resource naming contract documented in
docs/guide/SDK_GUIDE.md.

### WithWorkItemConcurrencyLimits

```go
// WithWorkItemConcurrencyLimits sets per-workitem concurrency limits.
// When a workitem reaches its own limit, it waits BEFORE acquiring the global
// worker slot, so long-running I/O tasks do not occupy all global slots while queued.
//
// Example:
//
//	worker := client.Worker(workspaceID,
//	    moi.WithMaxConcurrentExecuteWorkItems(16),
//	    moi.WithWorkItemConcurrencyLimits(map[string]int{
//	        "moi:parser.convert.document.rich": 6,
//	        "moi:embedding.generate":           6,
//	    }),
//	)
func WithWorkItemConcurrencyLimits(limits map[string]int) WorkerClientOption
```

WithWorkItemConcurrencyLimits sets per-workitem concurrency limits.
When a workitem reaches its own limit, it waits BEFORE acquiring the global
worker slot, so long-running I/O tasks do not occupy all global slots while queued.

Example:

	worker := client.Worker(workspaceID,
	    moi.WithMaxConcurrentExecuteWorkItems(16),
	    moi.WithWorkItemConcurrencyLimits(map[string]int{
	        "moi:parser.convert.document.rich": 6,
	        "moi:embedding.generate":           6,
	    }),
	)

### WithWorkerID

```go
// WithWorkerID sets the worker ID for the Worker client.
// This ID is used to identify the worker when connecting to the Mowl Engine
// via the Catalog Service gRPC proxy.
//
// Example:
//
//	client, err := moi.New(endpoint, apiKey, moi.WithWorkerID("my-worker-1"))
//	worker := client.Worker(workspaceID)
func WithWorkerID(workerID string) Option
```

WithWorkerID sets the worker ID for the Worker client.
This ID is used to identify the worker when connecting to the Mowl Engine
via the Catalog Service gRPC proxy.

Example:

	client, err := moi.New(endpoint, apiKey, moi.WithWorkerID("my-worker-1"))
	worker := client.Worker(workspaceID)

### WithWorkflowDefDescription

```go
// WithWorkflowDefDescription sets the workflow description when creating a workflow definition.
//
// Example:
//
//	wf, err := client.Workflows(workspaceID).Create(ctx, "my-workflow",
//	    moi.WithWorkflowDefDescription("This workflow processes data"),
//	)
func WithWorkflowDefDescription(description string) CreateWorkflowDefOption
```

WithWorkflowDefDescription sets the workflow description when creating a workflow definition.

Example:

	wf, err := client.Workflows(workspaceID).Create(ctx, "my-workflow",
	    moi.WithWorkflowDefDescription("This workflow processes data"),
	)

### WithWorkflowNameFilter

```go
// WithWorkflowNameFilter filters workflows by name (partial match).
//
// Example:
//
//	workflows, err := client.Workflows(workspaceID).List(ctx, moi.WithWorkflowNameFilter("data-"))
func WithWorkflowNameFilter(nameFilter string) ListWorkflowsOption
```

WithWorkflowNameFilter filters workflows by name (partial match).

Example:

	workflows, err := client.Workflows(workspaceID).List(ctx, moi.WithWorkflowNameFilter("data-"))

### WithWorkspaceCreatedCallback

```go
// WithWorkspaceCreatedCallback sets a callback function that will be called
// whenever a workspace is successfully created. This is useful for test frameworks
// to automatically track created workspaces for cleanup.
//
// Example:
//
//	callback := func(ws *workspace.Workspace) {
//	    fmt.Printf("Workspace created: %s\n", ws.Id)
//	}
//	client, err := moi.New(endpoint, apiKey, moi.WithWorkspaceCreatedCallback(callback))
func WithWorkspaceCreatedCallback(callback WorkspaceCreatedCallback) Option
```

WithWorkspaceCreatedCallback sets a callback function that will be called
whenever a workspace is successfully created. This is useful for test frameworks
to automatically track created workspaces for cleanup.

Example:

	callback := func(ws *workspace.Workspace) {
	    fmt.Printf("Workspace created: %s\n", ws.Id)
	}
	client, err := moi.New(endpoint, apiKey, moi.WithWorkspaceCreatedCallback(callback))

### WithWorkspaceDescription

```go
// WithWorkspaceDescription sets the workspace description when creating a workspace.
//
// Example:
//
//	ws, err := client.Workspaces().Create(ctx, "my-workspace", moi.WithWorkspaceDescription("This is my workspace"))
func WithWorkspaceDescription(description string) CreateWorkspaceOption
```

WithWorkspaceDescription sets the workspace description when creating a workspace.

Example:

	ws, err := client.Workspaces().Create(ctx, "my-workspace", moi.WithWorkspaceDescription("This is my workspace"))

### WithWorkspaceIdempotencyKey

```go
// WithWorkspaceIdempotencyKey identifies one logical workspace creation across retries.
func WithWorkspaceIdempotencyKey(key string) CreateWorkspaceOption
```

WithWorkspaceIdempotencyKey identifies one logical workspace creation across retries.

### WithWorkspaceModelType

```go
// WithWorkspaceModelType filters workspace models by structured model type when
// the server can classify the model list.
func WithWorkspaceModelType(modelType string) ListWorkspaceModelOption
```

WithWorkspaceModelType filters workspace models by structured model type when
the server can classify the model list.

### WithWorkspaceName

```go
// WithWorkspaceName sets the new name when updating a workspace.
//
// Example:
//
//	ws, err := client.Workspaces().Update(ctx, workspaceID, moi.WithWorkspaceName("new-name"))
func WithWorkspaceName(name string) UpdateWorkspaceOption
```

WithWorkspaceName sets the new name when updating a workspace.

Example:

	ws, err := client.Workspaces().Update(ctx, workspaceID, moi.WithWorkspaceName("new-name"))

### WithWorkspaceOwnerID

```go
// WithWorkspaceOwnerID narrows WorkspaceService.ListPage to workspaces owned
// by exactly ownerID. Other list services ignore this option.
func WithWorkspaceOwnerID(ownerID string) ListOption
```

WithWorkspaceOwnerID narrows WorkspaceService.ListPage to workspaces owned
by exactly ownerID. Other list services ignore this option.

### WithWorkspaceShard

```go
// WithWorkspaceShard selects one stable workspace-ID shard for
// WorkspaceService.ListPage. shardCount and shardIndex are validated by
// Catalog; callers must persist the opaque next-page token per shard.
func WithWorkspaceShard(shardCount, shardIndex int) ListOption
```

WithWorkspaceShard selects one stable workspace-ID shard for
WorkspaceService.ListPage. shardCount and shardIndex are validated by
Catalog; callers must persist the opaque next-page token per shard.

## APIKeyService

APIKeyService provides API key management operations.

### Create

```go
// Create creates a new API key.
//
// Parameters:
//   - ctx: Context for the request
//   - name: The API key name (required)
//   - opts: Optional parameters (WithScopes, WithExpiresInDays, WithUserID)
//
// Returns the created API key with the secret key. The secret key is only
// returned once at creation time and cannot be retrieved later.
//
// Example:
//
//	apiKey, err := client.APIKeys().Create(ctx, "my-api-key",
//	    moi.WithScopes("read", "write"),
//	    moi.WithExpiresInDays(30),
//	)
//	if err != nil {
//	    return err
//	}
//	// Save the secret key - it won't be available again
//	fmt.Printf("API Key: %s\n", apiKey.Key)
//
// System users can create API keys for other users:
//
//	apiKey, err := systemClient.APIKeys().Create(ctx, "user-key", moi.WithUserID("user-123"))
func (s *APIKeyService) Create(ctx context.Context, name string, opts ...CreateAPIKeyOption) (*auth.APIKeyWithSecret, error)
```

Create creates a new API key.

Parameters:
  - ctx: Context for the request
  - name: The API key name (required)
  - opts: Optional parameters (WithScopes, WithExpiresInDays, WithUserID)

Returns the created API key with the secret key. The secret key is only
returned once at creation time and cannot be retrieved later.

Example:

	apiKey, err := client.APIKeys().Create(ctx, "my-api-key",
	    moi.WithScopes("read", "write"),
	    moi.WithExpiresInDays(30),
	)
	if err != nil {
	    return err
	}
	// Save the secret key - it won't be available again
	fmt.Printf("API Key: %s\n", apiKey.Key)

System users can create API keys for other users:

	apiKey, err := systemClient.APIKeys().Create(ctx, "user-key", moi.WithUserID("user-123"))

### Delete

```go
// Delete deletes an API key by ID.
//
// Parameters:
//   - ctx: Context for the request
//   - id: The API key ID (string)
//
// Returns an error if the deletion fails.
//
// Example:
//
//	err := client.APIKeys().Delete(ctx, "api-key-id")
func (s *APIKeyService) Delete(ctx context.Context, id string) error
```

Delete deletes an API key by ID.

Parameters:
  - ctx: Context for the request
  - id: The API key ID (string)

Returns an error if the deletion fails.

Example:

	err := client.APIKeys().Delete(ctx, "api-key-id")

### List

```go
// List retrieves all API keys with optional pagination.
//
// Parameters:
//   - ctx: Context for the request
//   - opts: Optional pagination parameters (WithPageSize, WithPageToken)
//
// Returns a list of API keys. Note that the secret key is not included
// in the response - only the key prefix is available for identification.
//
// Example:
//
//	apiKeys, err := client.APIKeys().List(ctx,
//	    moi.WithPageSize(10),
//	    moi.WithPageToken("next-page-token"),
//	)
func (s *APIKeyService) List(ctx context.Context, opts ...ListOption) ([]*auth.APIKey, error)
```

List retrieves all API keys with optional pagination.

Parameters:
  - ctx: Context for the request
  - opts: Optional pagination parameters (WithPageSize, WithPageToken)

Returns a list of API keys. Note that the secret key is not included
in the response - only the key prefix is available for identification.

Example:

	apiKeys, err := client.APIKeys().List(ctx,
	    moi.WithPageSize(10),
	    moi.WithPageToken("next-page-token"),
	)

## AgentPackageService

AgentPackageService provides agent package load/export operations scoped to a workspace.

### Export

```go
// Export downloads a persisted .moiagent archive for an agent version.
func (s *AgentPackageService) Export(ctx context.Context, agentID, version string) (*AgentPackageDownloadResponse, error)
```

Export downloads a persisted .moiagent archive for an agent version.

### Load

```go
// Load imports a .moiagent archive into the workspace.
func (s *AgentPackageService) Load(ctx context.Context, data []byte) (*AgentPackageLoadResponse, error)
```

Load imports a .moiagent archive into the workspace.

## AgentService

AgentService provides catalog-backed A2A agent operations.

### A2ASend

```go
// A2ASend sends a non-streaming A2A JSON-RPC request to the selected agent.
func (s *AgentService) A2ASend(ctx context.Context, req AgentA2ARequest) (json.RawMessage, error)
```

A2ASend sends a non-streaming A2A JSON-RPC request to the selected agent.

### A2AStream

```go
// A2AStream sends a streaming A2A JSON-RPC request to the selected agent.
func (s *AgentService) A2AStream(ctx context.Context, req AgentA2ARequest) (<-chan A2AStreamEvent, error)
```

A2AStream sends a streaming A2A JSON-RPC request to the selected agent.

### A2AStreamWithErrors

```go
// A2AStreamWithErrors sends a streaming A2A request and reports non-EOF read
// failures as results instead of collapsing them into a clean channel close.
func (s *AgentService) A2AStreamWithErrors(ctx context.Context, req AgentA2ARequest) (<-chan A2AStreamResult, error)
```

A2AStreamWithErrors sends a streaming A2A request and reports non-EOF read
failures as results instead of collapsing them into a clean channel close.

### Card

```go
// Card returns the A2A agent card for the selected agent.
func (s *AgentService) Card(ctx context.Context, selector AgentSelector) (json.RawMessage, error)
```

Card returns the A2A agent card for the selected agent.

## AgentVersionService

AgentVersionService provides Agent version lifecycle operations scoped to a workspace.

### Delete

```go
// Delete deletes a disabled non-default Agent version.
func (s *AgentVersionService) Delete(ctx context.Context, agentID, version string) (*AgentVersionDeleteResponse, error)
```

Delete deletes a disabled non-default Agent version.

### Disable

```go
// Disable disables a non-default Agent version for future runtime selection.
func (s *AgentVersionService) Disable(ctx context.Context, agentID, version string) (*AgentVersion, error)
```

Disable disables a non-default Agent version for future runtime selection.

### List

```go
// List returns all versions for one Agent lineage in the workspace.
func (s *AgentVersionService) List(ctx context.Context, agentID string) (*AgentVersionListResponse, error)
```

List returns all versions for one Agent lineage in the workspace.

### SetDefault

```go
// SetDefault sets the workspace default version for an Agent lineage.
func (s *AgentVersionService) SetDefault(ctx context.Context, agentID, version string) (*AgentLineage, error)
```

SetDefault sets the workspace default version for an Agent lineage.

## CDHService

CDHService provides CDH configuration and metadata management operations.
All operations are scoped to a specific workspace.

### CreateConfig

```go
// CreateConfig creates a new CDH configuration.
//
// Parameters:
//   - ctx: Context for the request
//   - name: The config name (required, unique within workspace)
//   - metastoreAddress: Hive Metastore address host:port (required)
//   - hiveAddress: HiveServer2 address host:port (required)
//   - version: The CDH version, e.g. "6.3.2" (required)
//   - opts: Optional parameters (WithCDHConnectTimeout, WithCDHKerberosPrincipal, WithCDHKerberosKeytab)
//
// Returns the created CDH config or an error.
func (s *CDHService) CreateConfig(ctx context.Context, name, metastoreAddress, hiveAddress, version string, opts ...CreateCDHConfigOption) (*catalog.CDHConfig, error)
```

CreateConfig creates a new CDH configuration.

Parameters:
  - ctx: Context for the request
  - name: The config name (required, unique within workspace)
  - metastoreAddress: Hive Metastore address host:port (required)
  - hiveAddress: HiveServer2 address host:port (required)
  - version: The CDH version, e.g. "6.3.2" (required)
  - opts: Optional parameters (WithCDHConnectTimeout, WithCDHKerberosPrincipal, WithCDHKerberosKeytab)

Returns the created CDH config or an error.

### DeleteConfig

```go
// DeleteConfig deletes a CDH configuration by ID.
func (s *CDHService) DeleteConfig(ctx context.Context, configID int64) error
```

DeleteConfig deletes a CDH configuration by ID.

### GetConfig

```go
// GetConfig retrieves a CDH configuration by ID.
func (s *CDHService) GetConfig(ctx context.Context, configID int64) (*catalog.CDHConfig, error)
```

GetConfig retrieves a CDH configuration by ID.

### GetDatabase

```go
// GetDatabase retrieves a CDH database by ID.
func (s *CDHService) GetDatabase(ctx context.Context, configID, databaseID int64) (*catalog.CDHDatabase, error)
```

GetDatabase retrieves a CDH database by ID.

### GetTable

```go
// GetTable retrieves a CDH table by ID, including its columns.
func (s *CDHService) GetTable(ctx context.Context, configID, databaseID, tableID int64) (*catalog.CDHTable, error)
```

GetTable retrieves a CDH table by ID, including its columns.

### HealthCheck

```go
// HealthCheck checks the health of a CDH connection.
func (s *CDHService) HealthCheck(ctx context.Context, configID int64) (*catalog.CDHHealthCheckResponse, error)
```

HealthCheck checks the health of a CDH connection.

### ListConfigs

```go
// ListConfigs lists all CDH configurations in the workspace.
func (s *CDHService) ListConfigs(ctx context.Context, opts ...ListOption) (*catalog.ListCDHConfigsResponse, error)
```

ListConfigs lists all CDH configurations in the workspace.

### ListDatabases

```go
// ListDatabases lists all databases synced from a CDH configuration.
func (s *CDHService) ListDatabases(ctx context.Context, configID int64, opts ...ListOption) (*catalog.ListCDHDatabasesResponse, error)
```

ListDatabases lists all databases synced from a CDH configuration.

### ListTables

```go
// ListTables lists all tables in a CDH database.
func (s *CDHService) ListTables(ctx context.Context, configID, databaseID int64, opts ...ListOption) (*catalog.ListCDHTablesResponse, error)
```

ListTables lists all tables in a CDH database.

### StopSync

```go
// StopSync cancels the periodic sync workflow for the specified config.
func (s *CDHService) StopSync(ctx context.Context, configID int64) error
```

StopSync cancels the periodic sync workflow for the specified config.

### SyncMetadata

```go
// SyncMetadata creates a periodic sync workflow for a CDH database.
func (s *CDHService) SyncMetadata(ctx context.Context, configID int64, databaseName, cronExpression string) (*catalog.SyncCDHMetadataResponse, error)
```

SyncMetadata creates a periodic sync workflow for a CDH database.

### UpdateConfig

```go
// UpdateConfig updates an existing CDH configuration.
func (s *CDHService) UpdateConfig(ctx context.Context, configID int64, opts ...UpdateCDHConfigOption) (*catalog.CDHConfig, error)
```

UpdateConfig updates an existing CDH configuration.

## CaseService

CaseService provides cross-task workflow-case (execution log) listing.
Always scoped to a workspace via client.Cases(workspaceID).

### List

```go
// List returns workflow execution cases visible to the authenticated user
// in this workspace, sorted by creation time desc.
//
// Example:
//
//	resp, err := client.Cases(workspaceID).List(ctx,
//	    moi.WithCaseWorkflowVersionIDs(verID1, verID2),
//	    moi.WithCaseLimit(20),
//	    moi.WithCaseOffset(0),
//	)
func (s *CaseService) List(ctx context.Context, opts ...ListCasesOption) (*mowl.ListCasesResponse, error)
```

List returns workflow execution cases visible to the authenticated user
in this workspace, sorted by creation time desc.

Example:

	resp, err := client.Cases(workspaceID).List(ctx,
	    moi.WithCaseWorkflowVersionIDs(verID1, verID2),
	    moi.WithCaseLimit(20),
	    moi.WithCaseOffset(0),
	)

### ListWorkitems

```go
// ListWorkitems returns every workitem of the given case, with the latest
// workitem-status row left-joined. Raw workitem data/result payloads are not
// returned by default; use ListWorkitemsWithPayload when a caller explicitly
// needs those large fields.
//
// workitem.Node is a raw JSON string — the SDK intentionally does not parse it
// because the schema depends on each workflow's DSL node contracts.
//
// Example:
//
//	resp, err := client.Cases(workspaceID).ListWorkitems(ctx, caseID)
func (s *CaseService) ListWorkitems(ctx context.Context, caseID string) (*mowl.ListCaseWorkitemsResponse, error)
```

ListWorkitems returns every workitem of the given case, with the latest
workitem-status row left-joined. Raw workitem data/result payloads are not
returned by default; use ListWorkitemsWithPayload when a caller explicitly
needs those large fields.

workitem.Node is a raw JSON string — the SDK intentionally does not parse it
because the schema depends on each workflow's DSL node contracts.

Example:

	resp, err := client.Cases(workspaceID).ListWorkitems(ctx, caseID)

### ListWorkitemsWithPayload

```go
// ListWorkitemsWithPayload returns the same rows as ListWorkitems and also
// includes raw workitem.Data / workitem.Result payloads. These fields can be
// very large for fan-out workflows, so callers should use this only for
// explicit detail views or export flows.
func (s *CaseService) ListWorkitemsWithPayload(ctx context.Context, caseID string) (*mowl.ListCaseWorkitemsResponse, error)
```

ListWorkitemsWithPayload returns the same rows as ListWorkitems and also
includes raw workitem.Data / workitem.Result payloads. These fields can be
very large for fan-out workflows, so callers should use this only for
explicit detail views or export flows.

## CatalogService

CatalogService provides catalog management operations.

### Create

```go
// Create creates a new catalog.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID (required)
//   - name: The catalog name (required); must follow the Catalog resource naming
//     contract documented in docs/guide/SDK_GUIDE.md
//   - opts: Optional parameters (WithComment)
//
// Returns the created catalog or an error.
//
// Example:
//
//	catalog, err := client.Catalogs().Create(ctx, workspaceID, "my-catalog",
//	    moi.WithComment("This is my catalog"),
//	)
func (s *CatalogService) Create(ctx context.Context, workspaceID string, name string, opts ...CreateCatalogOption) (*catalog.Catalog, error)
```

Create creates a new catalog.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID (required)
  - name: The catalog name (required); must follow the Catalog resource naming
    contract documented in docs/guide/SDK_GUIDE.md
  - opts: Optional parameters (WithComment)

Returns the created catalog or an error.

Example:

	catalog, err := client.Catalogs().Create(ctx, workspaceID, "my-catalog",
	    moi.WithComment("This is my catalog"),
	)

### Delete

```go
// Delete deletes a catalog by ID.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - catalogID: The catalog ID
//
// Returns an error if the deletion fails.
//
// Example:
//
//	err := client.Catalogs().Delete(ctx, workspaceID, catalogID)
func (s *CatalogService) Delete(ctx context.Context, workspaceID string, catalogID int64) error
```

Delete deletes a catalog by ID.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - catalogID: The catalog ID

Returns an error if the deletion fails.

Example:

	err := client.Catalogs().Delete(ctx, workspaceID, catalogID)

### DeleteMultiple

```go
// DeleteMultiple deletes multiple catalogs by IDs.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - catalogIDs: The catalog IDs to delete
//   - opts: Optional batch options (WithContinueOnError, WithBatchConcurrency)
//
// Returns a BatchResult with success/failure counts.
//
// Example:
//
//	result, err := client.Catalogs().DeleteMultiple(ctx, workspaceID, []int64{1, 2, 3},
//	    moi.WithContinueOnError(true),
//	)
func (s *CatalogService) DeleteMultiple(ctx context.Context, workspaceID string, catalogIDs []int64, opts ...BatchOption) (*BatchResult, error)
```

DeleteMultiple deletes multiple catalogs by IDs.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - catalogIDs: The catalog IDs to delete
  - opts: Optional batch options (WithContinueOnError, WithBatchConcurrency)

Returns a BatchResult with success/failure counts.

Example:

	result, err := client.Catalogs().DeleteMultiple(ctx, workspaceID, []int64{1, 2, 3},
	    moi.WithContinueOnError(true),
	)

### Get

```go
// Get retrieves a catalog by ID.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - catalogID: The catalog ID
//
// Returns the catalog or an error if not found.
//
// Example:
//
//	catalog, err := client.Catalogs().Get(ctx, workspaceID, catalogID)
func (s *CatalogService) Get(ctx context.Context, workspaceID string, catalogID int64) (*catalog.Catalog, error)
```

Get retrieves a catalog by ID.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - catalogID: The catalog ID

Returns the catalog or an error if not found.

Example:

	catalog, err := client.Catalogs().Get(ctx, workspaceID, catalogID)

### GetStats

```go
// GetStats retrieves statistics for a catalog by ID.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - catalogID: The catalog ID
//
// Returns the catalog statistics or an error if not found.
//
// Example:
//
//	stats, err := client.Catalogs().GetStats(ctx, workspaceID, catalogID)
func (s *CatalogService) GetStats(ctx context.Context, workspaceID string, catalogID int64) (*CatalogStatsResponse, error)
```

GetStats retrieves statistics for a catalog by ID.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - catalogID: The catalog ID

Returns the catalog statistics or an error if not found.

Example:

	stats, err := client.Catalogs().GetStats(ctx, workspaceID, catalogID)

### List

```go
// List retrieves all catalogs with optional pagination.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - opts: Optional pagination parameters (WithPageSize, WithPageToken)
//
// Returns a list response with catalogs or an error.
//
// Example:
//
//	resp, err := client.Catalogs().List(ctx, workspaceID,
//	    moi.WithPageSize(10),
//	    moi.WithPageToken("next-page-token"),
//	)
func (s *CatalogService) List(ctx context.Context, workspaceID string, opts ...ListOption) (*catalog.ListCatalogsResponse, error)
```

List retrieves all catalogs with optional pagination.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - opts: Optional pagination parameters (WithPageSize, WithPageToken)

Returns a list response with catalogs or an error.

Example:

	resp, err := client.Catalogs().List(ctx, workspaceID,
	    moi.WithPageSize(10),
	    moi.WithPageToken("next-page-token"),
	)

### ListDatabases

```go
// ListDatabases retrieves all databases under a catalog.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - catalogID: The catalog ID
//   - opts: Optional pagination parameters (WithPageSize, WithPageToken)
//
// Returns a list response with databases or an error.
//
// Example:
//
//	resp, err := client.Catalogs().ListDatabases(ctx, workspaceID, catalogID,
//	    moi.WithPageSize(10),
//	)
func (s *CatalogService) ListDatabases(ctx context.Context, workspaceID string, catalogID int64, opts ...ListOption) (*catalog.ListDatabasesResponse, error)
```

ListDatabases retrieves all databases under a catalog.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - catalogID: The catalog ID
  - opts: Optional pagination parameters (WithPageSize, WithPageToken)

Returns a list response with databases or an error.

Example:

	resp, err := client.Catalogs().ListDatabases(ctx, workspaceID, catalogID,
	    moi.WithPageSize(10),
	)

### ListDatabasesForSystem

```go
// ListDatabasesForSystem retrieves Catalog databases through the system-key-
// only trusted Backend read boundary.
func (s *CatalogService) ListDatabasesForSystem(ctx context.Context, workspaceID string, catalogID int64, databaseScope *catalog.CatalogReadScope, opts ...ListOption) (*catalog.ListDatabasesResponse, error)
```

ListDatabasesForSystem retrieves Catalog databases through the system-key-
only trusted Backend read boundary.

### ListIter

```go
// ListIter returns an iterator for listing catalogs.
func (s *CatalogService) ListIter(ctx context.Context, workspaceID string, opts ...ListOption) *CatalogIterator
```

ListIter returns an iterator for listing catalogs.

### ListSummaries

```go
// ListSummaries retrieves Catalog summaries in one Core request. Each
// item contains a Catalog and the number of visible Databases beneath it.
func (s *CatalogService) ListSummaries(ctx context.Context, workspaceID string, opts ...ListOption) (*catalog.ListCatalogSummariesResponse, error)
```

ListSummaries retrieves Catalog summaries in one Core request. Each
item contains a Catalog and the number of visible Databases beneath it.

### ListSummariesForSystem

```go
// ListSummariesForSystem retrieves summaries through the system-key-only
// trusted Backend read boundary. Both scopes must originate from a Core IAM
// projection for the authenticated Backend principal.
func (s *CatalogService) ListSummariesForSystem(ctx context.Context, workspaceID string, catalogScope, databaseScope *catalog.CatalogReadScope, opts ...ListOption) (*catalog.ListCatalogSummariesResponse, error)
```

ListSummariesForSystem retrieves summaries through the system-key-only
trusted Backend read boundary. Both scopes must originate from a Core IAM
projection for the authenticated Backend principal.

### ResolveMetadata

```go
// ResolveMetadata retrieves authoritative Catalog metadata through the
// system-key-only resolver. It does not grant caller access to the Catalog.
func (s *CatalogService) ResolveMetadata(ctx context.Context, workspaceID string, catalogID int64) (*catalog.Catalog, error)
```

ResolveMetadata retrieves authoritative Catalog metadata through the
system-key-only resolver. It does not grant caller access to the Catalog.

### Tree

```go
// Tree retrieves the complete permission-filtered Catalog hierarchy in one
// Core request.
func (s *CatalogService) Tree(ctx context.Context, workspaceID string, includeTableLeaves bool) (*catalog.GetCatalogTreeResponse, error)
```

Tree retrieves the complete permission-filtered Catalog hierarchy in one
Core request.

### Update

```go
// Update updates an existing catalog.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - catalogID: The catalog ID
//   - opts: Fields to update (WithName, WithUpdatedComment)
//
// Returns the updated catalog or an error.
//
// Example:
//
//	catalog, err := client.Catalogs().Update(ctx, workspaceID, catalogID,
//	    moi.WithName("new-name"),
//	    moi.WithUpdatedComment("Updated comment"),
//	)
func (s *CatalogService) Update(ctx context.Context, workspaceID string, catalogID int64, opts ...UpdateCatalogOption) (*catalog.Catalog, error)
```

Update updates an existing catalog.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - catalogID: The catalog ID
  - opts: Fields to update (WithName, WithUpdatedComment)

Returns the updated catalog or an error.

Example:

	catalog, err := client.Catalogs().Update(ctx, workspaceID, catalogID,
	    moi.WithName("new-name"),
	    moi.WithUpdatedComment("Updated comment"),
	)

## CatalogTraceService

CatalogTraceService provides Langfuse CatalogTrace management operations.

CatalogTrace records bind a Langfuse dataconn connector to a Catalog resource node,
tracking sync status and observation counts updated by the Mirror Worker.

Write routes (CreateCatalogTrace / UpdateCatalogTraceStats / SoftDeleteCatalogTrace)
are system-internal: the Client must be configured with a service-account API key.

### CreateCatalogTrace

```go
// CreateCatalogTrace creates a new CatalogTrace resource node for the given workspace.
// Called by moi-backend within the Langfuse connector creation path (C1).
//
// Requires a service-account (system) API key.
//
// Parameters:
//   - ctx: Context for the request.
//   - workspaceID: The workspace to create the trace in.
//   - req: Creation request (connector_id, catalog_id, langfuse_host, public_key, storage_ref).
//
// Returns the created CatalogTrace with generated ID and timestamps, or an error.
// Returns ErrAlreadyExists (HTTP 409) if a trace for the connector already exists.
func (s *CatalogTraceService) CreateCatalogTrace(
	ctx context.Context,
	workspaceID string,
	req *catalog.CreateCatalogTraceRequest,
) (*catalog.CatalogTrace, error)
```

CreateCatalogTrace creates a new CatalogTrace resource node for the given workspace.
Called by moi-backend within the Langfuse connector creation path (C1).

Requires a service-account (system) API key.

Parameters:
  - ctx: Context for the request.
  - workspaceID: The workspace to create the trace in.
  - req: Creation request (connector_id, catalog_id, langfuse_host, public_key, storage_ref).

Returns the created CatalogTrace with generated ID and timestamps, or an error.
Returns ErrAlreadyExists (HTTP 409) if a trace for the connector already exists.

### GetCatalogTrace

```go
// GetCatalogTrace retrieves the CatalogTrace node for the given connector.
// Used by moi-backend BFF after its user-facing DC2 permission check.
// Requires a service-account (system) API key.
//
// Parameters:
//   - ctx: Context for the request.
//   - workspaceID: The workspace the connector belongs to.
//   - connectorID: The Langfuse dataconn connector ID.
//
// Returns the CatalogTrace, or an error.
// Returns ErrNotFound (HTTP 404) if the connector has no associated trace or was deleted.
func (s *CatalogTraceService) GetCatalogTrace(
	ctx context.Context,
	workspaceID string,
	connectorID string,
) (*catalog.CatalogTrace, error)
```

GetCatalogTrace retrieves the CatalogTrace node for the given connector.
Used by moi-backend BFF after its user-facing DC2 permission check.
Requires a service-account (system) API key.

Parameters:
  - ctx: Context for the request.
  - workspaceID: The workspace the connector belongs to.
  - connectorID: The Langfuse dataconn connector ID.

Returns the CatalogTrace, or an error.
Returns ErrNotFound (HTTP 404) if the connector has no associated trace or was deleted.

### SoftDeleteCatalogTrace

```go
// SoftDeleteCatalogTrace marks the CatalogTrace for the given connector as deleted.
// Called by moi-backend when the Langfuse connector is deleted (C5 path).
//
// Requires a service-account (system) API key.
//
// Parameters:
//   - ctx: Context for the request.
//   - workspaceID: The workspace the connector belongs to.
//   - connectorID: The Langfuse dataconn connector ID.
//
// Returns an error if the deletion fails. The operation is idempotent — calling on an
// already-deleted trace returns nil.
func (s *CatalogTraceService) SoftDeleteCatalogTrace(
	ctx context.Context,
	workspaceID string,
	connectorID string,
) error
```

SoftDeleteCatalogTrace marks the CatalogTrace for the given connector as deleted.
Called by moi-backend when the Langfuse connector is deleted (C5 path).

Requires a service-account (system) API key.

Parameters:
  - ctx: Context for the request.
  - workspaceID: The workspace the connector belongs to.
  - connectorID: The Langfuse dataconn connector ID.

Returns an error if the deletion fails. The operation is idempotent — calling on an
already-deleted trace returns nil.

### UpdateCatalogTraceStats

```go
// UpdateCatalogTraceStats updates the sync statistics for the given connector's CatalogTrace.
// Called by the Mirror Worker after each sync round using a service-account API key.
//
// Requires a service-account (system) API key.
//
// Parameters:
//   - ctx: Context for the request.
//   - workspaceID: The workspace the connector belongs to.
//   - req: Statistics update (connector_id, sync_status, observation_count, last_synced_at, last_error).
//
// Returns the updated CatalogTrace, or an error.
// Returns ErrNotFound (HTTP 404) if no active trace exists for the connector.
func (s *CatalogTraceService) UpdateCatalogTraceStats(
	ctx context.Context,
	workspaceID string,
	req *catalog.UpdateCatalogTraceStatsRequest,
) (*catalog.CatalogTrace, error)
```

UpdateCatalogTraceStats updates the sync statistics for the given connector's CatalogTrace.
Called by the Mirror Worker after each sync round using a service-account API key.

Requires a service-account (system) API key.

Parameters:
  - ctx: Context for the request.
  - workspaceID: The workspace the connector belongs to.
  - req: Statistics update (connector_id, sync_status, observation_count, last_synced_at, last_error).

Returns the updated CatalogTrace, or an error.
Returns ErrNotFound (HTTP 404) if no active trace exists for the connector.

## CustomOperatorService

CustomOperatorService provides workspace custom operator operations.

### Delete

```go
// Delete removes one custom operator by ID.
func (s *CustomOperatorService) Delete(ctx context.Context, operatorID int64) error
```

Delete removes one custom operator by ID.

### Disable

```go
// Disable disables one custom operator by ID.
func (s *CustomOperatorService) Disable(ctx context.Context, operatorID int64) (*CustomOperator, error)
```

Disable disables one custom operator by ID.

### Enable

```go
// Enable enables one custom operator by ID.
func (s *CustomOperatorService) Enable(ctx context.Context, operatorID int64) (*CustomOperator, error)
```

Enable enables one custom operator by ID.

### Get

```go
// Get returns one custom operator by ID.
func (s *CustomOperatorService) Get(ctx context.Context, operatorID int64) (*CustomOperator, error)
```

Get returns one custom operator by ID.

### GetCode

```go
// GetCode downloads the custom operator source code.
func (s *CustomOperatorService) GetCode(ctx context.Context, operatorID int64) (string, error)
```

GetCode downloads the custom operator source code.

### List

```go
// List returns custom operators in the service workspace.
func (s *CustomOperatorService) List(ctx context.Context, req *ListCustomOperatorsRequest) (*ListCustomOperatorsResponse, error)
```

List returns custom operators in the service workspace.

### NewBuiltinBindingOperator

```go
// NewBuiltinBindingOperator starts building a custom operator that wraps an
// existing WorkItem with fixed inputs and input mappings.
func (s *CustomOperatorService) NewBuiltinBindingOperator(identifier, name, baseNodeID, baseNodeVersion string) *CustomOperatorBuilder
```

NewBuiltinBindingOperator starts building a custom operator that wraps an
existing WorkItem with fixed inputs and input mappings.

### NewPythonOperator

```go
// NewPythonOperator starts building a Python custom operator in the service workspace.
func (s *CustomOperatorService) NewPythonOperator(identifier, name string) *CustomOperatorBuilder
```

NewPythonOperator starts building a Python custom operator in the service workspace.

### ResolveIdentity

```go
// ResolveIdentity retrieves canonical Custom Operator metadata through the
// system-only Core boundary. The client must use the system API key.
func (s *CustomOperatorService) ResolveIdentity(ctx context.Context, nodeID, version string) (*CustomOperator, error)
```

ResolveIdentity retrieves canonical Custom Operator metadata through the
system-only Core boundary. The client must use the system API key.

### ResolveIdentityByID

```go
// ResolveIdentityByID retrieves canonical Custom Operator metadata by business
// ID through the system-only Core boundary. The client must use the system API
// key. The result is an ownership fact and never an authorization decision.
func (s *CustomOperatorService) ResolveIdentityByID(ctx context.Context, operatorID int64) (*CustomOperator, error)
```

ResolveIdentityByID retrieves canonical Custom Operator metadata by business
ID through the system-only Core boundary. The client must use the system API
key. The result is an ownership fact and never an authorization decision.

### Update

```go
// Update updates one custom operator by ID.
func (s *CustomOperatorService) Update(ctx context.Context, operatorID int64, req *UpdateCustomOperatorRequest) (*CustomOperator, error)
```

Update updates one custom operator by ID.

## DataAssetService

DataAssetService provides data asset APIs.

### BatchResolveCatalogFiles

```go
// BatchResolveCatalogFiles resolves multiple catalog file entries in one request.
func (s *DataAssetService) BatchResolveCatalogFiles(ctx context.Context, fileIDs []string) (*catalog.BatchResolveCatalogFilesResponse, error)
```

BatchResolveCatalogFiles resolves multiple catalog file entries in one request.

### CreateAsset

```go
// CreateAsset registers a typed data asset. assetRef is type-local; pass
// WithDataAssetType("file") for catalog file ids or "vector_index" for vector tables.
func (s *DataAssetService) CreateAsset(ctx context.Context, assetRef string, opts ...DataAssetOption) (*catalog.DataAsset, error)
```

CreateAsset registers a typed data asset. assetRef is type-local; pass
WithDataAssetType("file") for catalog file ids or "vector_index" for vector tables.

### CreateDerivation

```go
// CreateDerivation registers a typed derivation edge from sourceAssetID to targetAssetID.
func (s *DataAssetService) CreateDerivation(ctx context.Context, sourceAssetID, kind, targetAssetID string, opts ...DataDerivationOption) (*catalog.DataDerivation, error)
```

CreateDerivation registers a typed derivation edge from sourceAssetID to targetAssetID.

### GetCatalogFileAsset

```go
// GetCatalogFileAsset resolves a catalog file entry to its processed-artifact bridge.
func (s *DataAssetService) GetCatalogFileAsset(ctx context.Context, fileID string) (*catalog.CatalogFileAssetResolveItem, error)
```

GetCatalogFileAsset resolves a catalog file entry to its processed-artifact bridge.

### RegisterLineage

```go
// RegisterLineage atomically registers typed assets, derivations, and parsed manifest records.
// Existing typed assets are reused by asset_type and asset_ref when the request
// does not explicitly require a different asset_id.
func (s *DataAssetService) RegisterLineage(ctx context.Context, req *RegisterLineageRequest) (*RegisterLineageResponse, error)
```

RegisterLineage atomically registers typed assets, derivations, and parsed manifest records.
Existing typed assets are reused by asset_type and asset_ref when the request
does not explicitly require a different asset_id.

### ResolveAsset

```go
// ResolveAsset resolves asset by asset_id or typed identity.
func (s *DataAssetService) ResolveAsset(ctx context.Context, opts ...DataAssetResolveOption) (*catalog.DataAssetResolveResponse, error)
```

ResolveAsset resolves asset by asset_id or typed identity.

### UpsertParsedManifest

```go
// UpsertParsedManifest upserts parsed manifest.
func (s *DataAssetService) UpsertParsedManifest(ctx context.Context, assetID, rawFileID, parsedFileID string, opts ...ParsedManifestOption) (*catalog.ParsedManifest, error)
```

UpsertParsedManifest upserts parsed manifest.

## DataDashboardService

DataDashboardService accesses Core-owned data dashboard execution APIs.

### EvaluateAlert

```go
// EvaluateAlert applies a successful SQL result to the chart's persisted alert rule.
func (s *DataDashboardService) EvaluateAlert(ctx context.Context, dashboardID, chartID string, request *DataDashboardAlertEvaluationRequest) (*DataDashboardAlertEvaluationResult, error)
```

EvaluateAlert applies a successful SQL result to the chart's persisted alert rule.

### GenerateSQLDraft

```go
// GenerateSQLDraft asks Core to draft one read-only query for a persisted dashboard scope.
func (s *DataDashboardService) GenerateSQLDraft(ctx context.Context, dashboardID string, request *DataDashboardSQLDraftRequest) (*DataDashboardSQLDraftResult, error)
```

GenerateSQLDraft asks Core to draft one read-only query for a persisted dashboard scope.

### GetExecutionSpec

```go
// GetExecutionSpec returns the persisted chart SQL after Core authorization.
func (s *DataDashboardService) GetExecutionSpec(ctx context.Context, dashboardID, chartID string) (*DataDashboardExecutionSpec, error)
```

GetExecutionSpec returns the persisted chart SQL after Core authorization.

## DataShareService

DataShareService provides Data Publish and Data Subscription operations.

### AcceptSubscription

```go
// AcceptSubscription creates a subscription database and activates an invitation.
func (s *DataShareService) AcceptSubscription(ctx context.Context, workspaceID string, subscriptionID int64, request *catalogpb.AcceptDataShareSubscriptionRequest) (*catalogpb.DataShareSubscription, error)
```

AcceptSubscription creates a subscription database and activates an invitation.

### CheckPublicationName

```go
// CheckPublicationName reports whether a publication name is available.
func (s *DataShareService) CheckPublicationName(ctx context.Context, workspaceID, name string) (*catalogpb.DataShareNameAvailability, error)
```

CheckPublicationName reports whether a publication name is available.

### CreatePublication

```go
// CreatePublication creates a durable Data Publish operation.
func (s *DataShareService) CreatePublication(ctx context.Context, workspaceID string, request *catalogpb.CreateDataSharePublicationRequest) (*catalogpb.DataSharePublication, error)
```

CreatePublication creates a durable Data Publish operation.

### DeletePublication

```go
// DeletePublication begins a durable publication deletion.
func (s *DataShareService) DeletePublication(ctx context.Context, workspaceID string, publicationID int64) error
```

DeletePublication begins a durable publication deletion.

### DeleteSubscription

```go
// DeleteSubscription removes a subscription database and returns the relation to invitation state.
func (s *DataShareService) DeleteSubscription(ctx context.Context, workspaceID string, subscriptionID int64) (*catalogpb.DataShareSubscription, error)
```

DeleteSubscription removes a subscription database and returns the relation to invitation state.

### ListPublications

```go
// ListPublications lists publications visible to the current effective role.
func (s *DataShareService) ListPublications(ctx context.Context, workspaceID string, options DataShareListOptions) (*catalogpb.ListDataSharePublicationsResponse, error)
```

ListPublications lists publications visible to the current effective role.

### ListSourceTables

```go
// ListSourceTables lists Tables that may be selected for a publication. IDs
// are stable MatrixOne rel_id values accepted by table_scope.object_ids.
func (s *DataShareService) ListSourceTables(ctx context.Context, workspaceID, sourceDatabaseID string, options DataShareListOptions) (*catalogpb.ListDataShareSourceTablesResponse, error)
```

ListSourceTables lists Tables that may be selected for a publication. IDs
are stable MatrixOne rel_id values accepted by table_scope.object_ids.

### ListSubscriptions

```go
// ListSubscriptions lists invitations and subscriptions visible to the current effective role.
func (s *DataShareService) ListSubscriptions(ctx context.Context, workspaceID string, options DataShareListOptions) (*catalogpb.ListDataShareSubscriptionsResponse, error)
```

ListSubscriptions lists invitations and subscriptions visible to the current effective role.

### PublicationSummary

```go
// PublicationSummary returns publication and distinct target counts visible to the current effective role.
func (s *DataShareService) PublicationSummary(ctx context.Context, workspaceID string) (*catalogpb.DataSharePublicationSummary, error)
```

PublicationSummary returns publication and distinct target counts visible to the current effective role.

### UpdatePublication

```go
// UpdatePublication updates a publication through its durable operation.
func (s *DataShareService) UpdatePublication(ctx context.Context, workspaceID string, publicationID int64, request *catalogpb.UpdateDataSharePublicationRequest) (*catalogpb.DataSharePublication, error)
```

UpdatePublication updates a publication through its durable operation.

## DatabaseService

DatabaseService provides database metadata query operations.
Note: Database creation/deletion is done directly through MatrixOne DBConnection.
Use SyncMetadata to synchronize database and table metadata from MatrixOne to moi-core.

### CompensateCreateIAM

```go
// CompensateCreateIAM rolls back the Database ownership registered by the
// exact create request through the system API boundary.
func (s *DatabaseService) CompensateCreateIAM(ctx context.Context, workspaceID string, req CompensateDatabaseCreateIAMRequest) error
```

CompensateCreateIAM rolls back the Database ownership registered by the
exact create request through the system API boundary.

### CompensateTableCreateIAM

```go
// CompensateTableCreateIAM rolls back the Table ownership registered by the
// exact create request through the system API boundary.
func (s *DatabaseService) CompensateTableCreateIAM(ctx context.Context, workspaceID string, req CompensateTableCreateIAMRequest) error
```

CompensateTableCreateIAM rolls back the Table ownership registered by the
exact create request through the system API boundary.

### Get

```go
// Get retrieves a database metadata by its ID.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - databaseID: The database ID
//
// Returns the database metadata or an error if not found.
func (s *DatabaseService) Get(ctx context.Context, workspaceID string, databaseID int64) (*catalog.Database, error)
```

Get retrieves a database metadata by its ID.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - databaseID: The database ID

Returns the database metadata or an error if not found.

### GetStats

```go
// GetStats retrieves statistics for a database by its ID.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - databaseID: The database ID
//
// Returns the database statistics or an error if not found.
func (s *DatabaseService) GetStats(ctx context.Context, workspaceID string, databaseID int64) (*DatabaseStatsResponse, error)
```

GetStats retrieves statistics for a database by its ID.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - databaseID: The database ID

Returns the database statistics or an error if not found.

### GetTable

```go
// GetTable retrieves table metadata by table ID with its parent database and catalog.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - tableID: The table ID
//
// Returns the table metadata detail or an error if not found or unauthorized.
func (s *DatabaseService) GetTable(ctx context.Context, workspaceID string, tableID int64) (*catalog.GetTableResponse, error)
```

GetTable retrieves table metadata by table ID with its parent database and catalog.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - tableID: The table ID

Returns the table metadata detail or an error if not found or unauthorized.

### List

```go
// List retrieves all databases under a catalog.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - catalogID: The catalog ID
//   - opts: Optional pagination parameters (WithPageSize, WithPageToken)
//
// Returns a list response with databases or an error.
func (s *DatabaseService) List(ctx context.Context, workspaceID string, catalogID int64, opts ...ListOption) (*catalog.ListDatabasesResponse, error)
```

List retrieves all databases under a catalog.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - catalogID: The catalog ID
  - opts: Optional pagination parameters (WithPageSize, WithPageToken)

Returns a list response with databases or an error.

### ListChildren

```go
// ListChildren retrieves a Database's visible direct children in one Core
// request. It returns only the fields required to render the list.
func (s *DatabaseService) ListChildren(ctx context.Context, workspaceID string, databaseID int64) (*catalog.ListDatabaseChildrenResponse, error)
```

ListChildren retrieves a Database's visible direct children in one Core
request. It returns only the fields required to render the list.

### ListChildrenForSystem

```go
// ListChildrenForSystem retrieves Database direct children through the
// system-key-only trusted Backend read boundary.
func (s *DatabaseService) ListChildrenForSystem(ctx context.Context, workspaceID string, databaseID int64, tableScope, volumeScope *catalog.CatalogReadScope) (*catalog.ListDatabaseChildrenResponse, error)
```

ListChildrenForSystem retrieves Database direct children through the
system-key-only trusted Backend read boundary.

### ListIter

```go
// ListIter returns an iterator for listing databases under a catalog.
func (s *DatabaseService) ListIter(ctx context.Context, workspaceID string, catalogID int64, opts ...ListOption) *DatabaseIterator
```

ListIter returns an iterator for listing databases under a catalog.

### ListTables

```go
// ListTables retrieves all tables under a database.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - databaseID: The database ID
//   - opts: Optional pagination parameters (WithPageSize, WithPageToken)
//
// Returns a list response with tables or an error.
func (s *DatabaseService) ListTables(ctx context.Context, workspaceID string, databaseID int64, opts ...ListOption) (*catalog.ListTablesResponse, error)
```

ListTables retrieves all tables under a database.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - databaseID: The database ID
  - opts: Optional pagination parameters (WithPageSize, WithPageToken)

Returns a list response with tables or an error.

### ListVolumes

```go
// ListVolumes retrieves all volumes under a database.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - databaseID: The database ID
//   - opts: Optional pagination parameters (WithPageSize, WithPageToken)
//
// Returns a list response with volumes or an error.
func (s *DatabaseService) ListVolumes(ctx context.Context, workspaceID string, databaseID int64, opts ...ListOption) (*catalog.ListVolumesResponse, error)
```

ListVolumes retrieves all volumes under a database.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - databaseID: The database ID
  - opts: Optional pagination parameters (WithPageSize, WithPageToken)

Returns a list response with volumes or an error.

### ResolveDatabaseMetadata

```go
// ResolveDatabaseMetadata retrieves an authoritative Database fact through
// the generic system metadata boundary. It does not grant caller access.
func (s *DatabaseService) ResolveDatabaseMetadata(ctx context.Context, workspaceID string, databaseID int64) (*catalog.Database, error)
```

ResolveDatabaseMetadata retrieves an authoritative Database fact through
the generic system metadata boundary. It does not grant caller access.

### ResolveDatabaseTables

```go
// ResolveDatabaseTables returns authoritative workspace-scoped Table identities
// for a Database through the system boundary. It is intended for trusted
// authorization adapters resolving persisted names to stable IDs.
func (s *DatabaseService) ResolveDatabaseTables(ctx context.Context, workspaceID string, databaseID int64) (*catalog.ListTablesResponse, error)
```

ResolveDatabaseTables returns authoritative workspace-scoped Table identities
for a Database through the system boundary. It is intended for trusted
authorization adapters resolving persisted names to stable IDs.

### ResolveStructuredLoadTargetDatabase

```go
// ResolveStructuredLoadTargetDatabase retrieves database metadata through the
// Catalog system API. The caller must use a client initialized with the
// moi-core system API key.
func (s *DatabaseService) ResolveStructuredLoadTargetDatabase(ctx context.Context, workspaceID string, databaseID int64) (*catalog.Database, error)
```

ResolveStructuredLoadTargetDatabase retrieves database metadata through the
Catalog system API. The caller must use a client initialized with the
moi-core system API key.

### ResolveStructuredLoadTargetDatabaseRuntime

```go
// ResolveStructuredLoadTargetDatabaseRuntime retrieves database metadata and
// its structured-load runtime reference through the Catalog system API. The
// caller must use a client initialized with the moi-core system API key.
func (s *DatabaseService) ResolveStructuredLoadTargetDatabaseRuntime(ctx context.Context, workspaceID string, databaseID int64) (*dataconn.StructuredLoadTargetDatabaseResolution, error)
```

ResolveStructuredLoadTargetDatabaseRuntime retrieves database metadata and
its structured-load runtime reference through the Catalog system API. The
caller must use a client initialized with the moi-core system API key.

### ResolveStructuredLoadTargetTable

```go
// ResolveStructuredLoadTargetTable retrieves table metadata through the Catalog
// system API for structured-load target-runtime resolution. The caller must use
// a client initialized with the moi-core system API key.
func (s *DatabaseService) ResolveStructuredLoadTargetTable(ctx context.Context, workspaceID string, tableID int64) (*catalog.GetTableResponse, error)
```

ResolveStructuredLoadTargetTable retrieves table metadata through the Catalog
system API for structured-load target-runtime resolution. The caller must use
a client initialized with the moi-core system API key.

### ResolveTableMetadata

```go
// ResolveTableMetadata retrieves authoritative Table, Database, and Catalog
// metadata through the system boundary. The result is a fact for authorization
// adapters; it is not an allow decision.
func (s *DatabaseService) ResolveTableMetadata(ctx context.Context, workspaceID string, tableID int64) (*catalog.GetTableResponse, error)
```

ResolveTableMetadata retrieves authoritative Table, Database, and Catalog
metadata through the system boundary. The result is a fact for authorization
adapters; it is not an allow decision.

### SyncMetadata

```go
// SyncMetadata synchronizes database and table metadata from MatrixOne to moi-core.
// This operation queries MatrixOne's information_schema to get the database and its tables,
// then stores the metadata in moi-core's catalog_database and catalog_table tables.
// User-initiated create options validate the newly created Database or Table
// name. Ordinary background discovery preserves existing MatrixOne names.
//
// Only workspace administrators have permission to call this API.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - databaseName: The name of the database in MatrixOne to sync
//   - catalogID: The catalog ID to associate the database with
//   - opts: Optional parameters (e.g. WithSyncComment)
//
// Returns:
//   - *catalog.SyncMetadataResponse: Contains the synced database and tables info
//   - error: If the sync fails
//
// Example:
//
//	resp, err := client.Databases().SyncMetadata(ctx, workspaceID, "my_database", catalogID)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Synced database: %s, tables: %d\n", resp.Database.Name, resp.TablesSynced)
func (s *DatabaseService) SyncMetadata(ctx context.Context, workspaceID string, databaseName string, catalogID int64, opts ...SyncMetadataOption) (*catalog.SyncMetadataResponse, error)
```

SyncMetadata synchronizes database and table metadata from MatrixOne to moi-core.
This operation queries MatrixOne's information_schema to get the database and its tables,
then stores the metadata in moi-core's catalog_database and catalog_table tables.
User-initiated create options validate the newly created Database or Table
name. Ordinary background discovery preserves existing MatrixOne names.

Only workspace administrators have permission to call this API.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - databaseName: The name of the database in MatrixOne to sync
  - catalogID: The catalog ID to associate the database with
  - opts: Optional parameters (e.g. WithSyncComment)

Returns:
  - *catalog.SyncMetadataResponse: Contains the synced database and tables info
  - error: If the sync fails

Example:

	resp, err := client.Databases().SyncMetadata(ctx, workspaceID, "my_database", catalogID)
	if err != nil {
	    log.Fatal(err)
	}
	fmt.Printf("Synced database: %s, tables: %d\n", resp.Database.Name, resp.TablesSynced)

## DataphinService

DataphinService provides Dataphin configuration and metadata management operations.
All operations are scoped to a specific workspace.

### CreateConfig

```go
// CreateConfig creates a new Dataphin configuration.
//
// Parameters:
//   - ctx: Context for the request
//   - name: The config name (required, unique within workspace)
//   - accessKeyID: The Alibaba Cloud AccessKey ID (required)
//   - accessKeySecret: The Alibaba Cloud AccessKey Secret (required)
//   - endpoint: The Dataphin endpoint (required)
//   - opts: Optional parameters (WithDPRegion, WithDPProjectName)
//
// Returns the created Dataphin config or an error.
func (s *DataphinService) CreateConfig(ctx context.Context, name, accessKeyID, accessKeySecret, endpoint string, opts ...CreateDPConfigOption) (*catalog.DPConfig, error)
```

CreateConfig creates a new Dataphin configuration.

Parameters:
  - ctx: Context for the request
  - name: The config name (required, unique within workspace)
  - accessKeyID: The Alibaba Cloud AccessKey ID (required)
  - accessKeySecret: The Alibaba Cloud AccessKey Secret (required)
  - endpoint: The Dataphin endpoint (required)
  - opts: Optional parameters (WithDPRegion, WithDPProjectName)

Returns the created Dataphin config or an error.

### DeleteConfig

```go
// DeleteConfig deletes a Dataphin configuration by ID.
// This will also cascade delete all associated metadata (databases, tables, columns).
func (s *DataphinService) DeleteConfig(ctx context.Context, configID int64) error
```

DeleteConfig deletes a Dataphin configuration by ID.
This will also cascade delete all associated metadata (databases, tables, columns).

### GetConfig

```go
// GetConfig retrieves a Dataphin configuration by ID.
func (s *DataphinService) GetConfig(ctx context.Context, configID int64) (*catalog.DPConfig, error)
```

GetConfig retrieves a Dataphin configuration by ID.

### GetDatabase

```go
// GetDatabase retrieves a Dataphin database by ID.
func (s *DataphinService) GetDatabase(ctx context.Context, configID, databaseID int64) (*catalog.DPDatabase, error)
```

GetDatabase retrieves a Dataphin database by ID.

### GetTable

```go
// GetTable retrieves a Dataphin table by ID, including its columns.
func (s *DataphinService) GetTable(ctx context.Context, configID, databaseID, tableID int64) (*catalog.DPTable, error)
```

GetTable retrieves a Dataphin table by ID, including its columns.

### HealthCheck

```go
// HealthCheck checks the health of a Dataphin connection.
func (s *DataphinService) HealthCheck(ctx context.Context, configID int64) (*catalog.DPHealthCheckResponse, error)
```

HealthCheck checks the health of a Dataphin connection.

### ListConfigs

```go
// ListConfigs lists all Dataphin configurations in the workspace.
func (s *DataphinService) ListConfigs(ctx context.Context, opts ...ListOption) (*catalog.ListDPConfigsResponse, error)
```

ListConfigs lists all Dataphin configurations in the workspace.

### ListDatabases

```go
// ListDatabases lists all databases synced from a Dataphin configuration.
func (s *DataphinService) ListDatabases(ctx context.Context, configID int64, opts ...ListOption) (*catalog.ListDPDatabasesResponse, error)
```

ListDatabases lists all databases synced from a Dataphin configuration.

### ListTables

```go
// ListTables lists all tables in a Dataphin database.
func (s *DataphinService) ListTables(ctx context.Context, configID, databaseID int64, opts ...ListOption) (*catalog.ListDPTablesResponse, error)
```

ListTables lists all tables in a Dataphin database.

### StopSync

```go
// StopSync cancels the periodic sync workflow for the specified config.
//
// Parameters:
//   - ctx: Context for the request
//   - configID: The Dataphin config ID
//
// Returns an error if the operation fails.
func (s *DataphinService) StopSync(ctx context.Context, configID int64) error
```

StopSync cancels the periodic sync workflow for the specified config.

Parameters:
  - ctx: Context for the request
  - configID: The Dataphin config ID

Returns an error if the operation fails.

### SyncMetadata

```go
// SyncMetadata creates a periodic sync workflow for a Dataphin project.
// The workflow runs on the specified cron schedule. Use StopSync to cancel.
//
// Parameters:
//   - ctx: Context for the request
//   - configID: The Dataphin config ID
//   - projectName: The name of the Dataphin project to sync
//   - cronExpression: Cron expression for periodic sync schedule (e.g. "0 */6 * * *")
//
// Returns the sync result or an error.
func (s *DataphinService) SyncMetadata(ctx context.Context, configID int64, projectName, cronExpression string) (*catalog.SyncDPMetadataResponse, error)
```

SyncMetadata creates a periodic sync workflow for a Dataphin project.
The workflow runs on the specified cron schedule. Use StopSync to cancel.

Parameters:
  - ctx: Context for the request
  - configID: The Dataphin config ID
  - projectName: The name of the Dataphin project to sync
  - cronExpression: Cron expression for periodic sync schedule (e.g. "0 */6 * * *")

Returns the sync result or an error.

### UpdateConfig

```go
// UpdateConfig updates an existing Dataphin configuration.
func (s *DataphinService) UpdateConfig(ctx context.Context, configID int64, opts ...UpdateDPConfigOption) (*catalog.DPConfig, error)
```

UpdateConfig updates an existing Dataphin configuration.

## EmbeddingService

EmbeddingService provides embedding operations scoped to a workspace.

### CreateBackend

```go
// CreateBackend creates an embedding backend (requires PERM_MODEL_RESOURCE_CREATE or workspace admin).
func (s *EmbeddingService) CreateBackend(ctx context.Context, opts ...CreateBackendOption) (*catalog.Backend, error)
```

CreateBackend creates an embedding backend (requires PERM_MODEL_RESOURCE_CREATE or workspace admin).

### CreateEmbeddings

```go
// CreateEmbeddings calls OpenAI-compatible embedding endpoint and returns response.
func (s *EmbeddingService) CreateEmbeddings(ctx context.Context, model string, input []string, opts ...EmbeddingOption) (*catalog.EmbeddingResponse, error)
```

CreateEmbeddings calls OpenAI-compatible embedding endpoint and returns response.

### CreateEndpoint

```go
// CreateEndpoint adds an endpoint to an embedding backend (requires PERM_MODEL_RESOURCE_CREATE or PERM_MODEL_RESOURCE_UPDATE, or workspace admin).
func (s *EmbeddingService) CreateEndpoint(ctx context.Context, backendID int64, opts ...CreateEndpointOption) (*catalog.BackendEndpoint, error)
```

CreateEndpoint adds an endpoint to an embedding backend (requires PERM_MODEL_RESOURCE_CREATE or PERM_MODEL_RESOURCE_UPDATE, or workspace admin).

### CreateRaw

```go
// CreateRaw posts a caller-supplied embedding request body to the workspace
// embedding route and returns the raw response body. This is used by
// multimodal callers whose request/response contracts are not represented by
// the text-only EmbeddingRequest proto.
func (s *EmbeddingService) CreateRaw(ctx context.Context, payload []byte) ([]byte, error)
```

CreateRaw posts a caller-supplied embedding request body to the workspace
embedding route and returns the raw response body. This is used by
multimodal callers whose request/response contracts are not represented by
the text-only EmbeddingRequest proto.

### DeleteBackend

```go
// DeleteBackend deletes an embedding backend (requires PERM_MODEL_RESOURCE_DELETE or workspace admin).
func (s *EmbeddingService) DeleteBackend(ctx context.Context, backendID int64) error
```

DeleteBackend deletes an embedding backend (requires PERM_MODEL_RESOURCE_DELETE or workspace admin).

### GetBackend

```go
// GetBackend retrieves an embedding backend by ID with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).
func (s *EmbeddingService) GetBackend(ctx context.Context, backendID int64) (*catalog.Backend, error)
```

GetBackend retrieves an embedding backend by ID with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).

### GetRouterConfig

```go
// GetRouterConfig retrieves the embedding router config.
func (s *EmbeddingService) GetRouterConfig(ctx context.Context) (*catalog.GetRouterConfigResponse, error)
```

GetRouterConfig retrieves the embedding router config.

### ListBackends

```go
// ListBackends lists all embedding backends with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).
func (s *EmbeddingService) ListBackends(ctx context.Context) (*catalog.ListBackendsResponse, error)
```

ListBackends lists all embedding backends with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).

### ListEndpoints

```go
// ListEndpoints lists all endpoints for an embedding backend (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).
func (s *EmbeddingService) ListEndpoints(ctx context.Context, backendID int64) ([]*catalog.BackendEndpoint, error)
```

ListEndpoints lists all endpoints for an embedding backend (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).

### ListModels

```go
// ListModels returns a flat list of all embedding models available on the workspace's
// embedding backends. This is the EmbeddingService counterpart of LLMService.ListEmbeddingModels.
func (s *EmbeddingService) ListModels(ctx context.Context) (*ListEmbeddingModelsResponse, error)
```

ListModels returns a flat list of all embedding models available on the workspace's
embedding backends. This is the EmbeddingService counterpart of LLMService.ListEmbeddingModels.

### PutRouterConfig

```go
// PutRouterConfig updates the embedding router config.
func (s *EmbeddingService) PutRouterConfig(ctx context.Context, opts ...PutRouterConfigOption) (*catalog.GetRouterConfigResponse, error)
```

PutRouterConfig updates the embedding router config.

### SetEndpointStatus

```go
// SetEndpointStatus sets the status of an embedding endpoint (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).
func (s *EmbeddingService) SetEndpointStatus(ctx context.Context, backendID, endpointID int64, opts ...SetEndpointStatusOption) error
```

SetEndpointStatus sets the status of an embedding endpoint (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).

### UpdateBackend

```go
// UpdateBackend updates an embedding backend (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).
func (s *EmbeddingService) UpdateBackend(ctx context.Context, backendID int64, opts ...UpdateBackendOption) (*catalog.Backend, error)
```

UpdateBackend updates an embedding backend (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).

## FileService

FileService provides file management operations.
It handles file upload, download, delete, and metadata retrieval.
Requirements: 2.1, 3.1, 4.1, 5.1

### Delete

```go
// Delete deletes a file by ID.
// The file must have zero references (ref_count = 0) to be deleted.
// Requirements: 4.1
func (s *FileService) Delete(ctx context.Context, workspaceID string, fileID string) error
```

Delete deletes a file by ID.
The file must have zero references (ref_count = 0) to be deleted.
Requirements: 4.1

### Download

```go
// Download downloads a file by ID.
// Returns an io.ReadCloser that the caller must close after use.
// Requirements: 3.1
func (s *FileService) Download(ctx context.Context, workspaceID string, fileID string) (io.ReadCloser, error)
```

Download downloads a file by ID.
Returns an io.ReadCloser that the caller must close after use.
Requirements: 3.1

### DownloadBytes

```go
// DownloadBytes downloads a file and returns its content as a byte slice.
// This is a convenience method for small files that fit in memory.
//
// Example:
//
//	data, err := client.Files().DownloadBytes(ctx, wsID, fileID)
//	if err != nil {
//	    return err
//	}
//	fmt.Println(string(data))
func (s *FileService) DownloadBytes(ctx context.Context, workspaceID string, fileID string) ([]byte, error)
```

DownloadBytes downloads a file and returns its content as a byte slice.
This is a convenience method for small files that fit in memory.

Example:

	data, err := client.Files().DownloadBytes(ctx, wsID, fileID)
	if err != nil {
	    return err
	}
	fmt.Println(string(data))

### DownloadBytesLimited

```go
// DownloadBytesLimited downloads a file into memory but rejects a response
// that exceeds maxBytes. Callers that must materialize a file should use this
// rather than DownloadBytes so an untrusted Catalog object cannot allocate an
// unbounded byte slice in a worker process.
func (s *FileService) DownloadBytesLimited(ctx context.Context, workspaceID string, fileID string, maxBytes int64) ([]byte, error)
```

DownloadBytesLimited downloads a file into memory but rejects a response
that exceeds maxBytes. Callers that must materialize a file should use this
rather than DownloadBytes so an untrusted Catalog object cannot allocate an
unbounded byte slice in a worker process.

### DownloadBytesPrefix

```go
// DownloadBytesPrefix returns at most the first maxBytes of a file. Catalog
// applies the range before reading physical storage; this is for MIME sniffing,
// not a post-download slice.
func (s *FileService) DownloadBytesPrefix(ctx context.Context, workspaceID string, fileID string, maxBytes int64) ([]byte, error)
```

DownloadBytesPrefix returns at most the first maxBytes of a file. Catalog
applies the range before reading physical storage; this is for MIME sniffing,
not a post-download slice.

### DownloadSemanticModelArtifactWithMeta

```go
// DownloadSemanticModelArtifactWithMeta reads a semantic-model-owned artifact
// through the system-only workspace-scoped capability. Callers must establish
// the semantic-model association before invoking this method.
func (s *FileService) DownloadSemanticModelArtifactWithMeta(ctx context.Context, workspaceID string, fileID string) (*FileDownloadResponse, error)
```

DownloadSemanticModelArtifactWithMeta reads a semantic-model-owned artifact
through the system-only workspace-scoped capability. Callers must establish
the semantic-model association before invoking this method.

### DownloadToFile

```go
// DownloadToFile downloads a file and saves it to the local filesystem.
// The file will be created with 0644 permissions.
// If the file already exists, it will be overwritten.
//
// Example:
//
//	err := client.Files().DownloadToFile(ctx, wsID, fileID, "/path/to/save/document.pdf")
func (s *FileService) DownloadToFile(ctx context.Context, workspaceID string, fileID string, destPath string) error
```

DownloadToFile downloads a file and saves it to the local filesystem.
The file will be created with 0644 permissions.
If the file already exists, it will be overwritten.

Example:

	err := client.Files().DownloadToFile(ctx, wsID, fileID, "/path/to/save/document.pdf")

### DownloadToWriter

```go
// DownloadToWriter downloads a file and writes its content to the provided writer.
// This is useful for streaming content to HTTP responses, buffers, or other destinations.
//
// Example:
//
//	var buf bytes.Buffer
//	n, err := client.Files().DownloadToWriter(ctx, wsID, fileID, &buf)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Downloaded %d bytes\n", n)
func (s *FileService) DownloadToWriter(ctx context.Context, workspaceID string, fileID string, w io.Writer) (int64, error)
```

DownloadToWriter downloads a file and writes its content to the provided writer.
This is useful for streaming content to HTTP responses, buffers, or other destinations.

Example:

	var buf bytes.Buffer
	n, err := client.Files().DownloadToWriter(ctx, wsID, fileID, &buf)
	if err != nil {
	    return err
	}
	fmt.Printf("Downloaded %d bytes\n", n)

### DownloadWithMeta

```go
// DownloadWithMeta downloads a file and returns the body together with the
// filename and content-type from the response headers.
func (s *FileService) DownloadWithMeta(ctx context.Context, workspaceID string, fileID string) (*FileDownloadResponse, error)
```

DownloadWithMeta downloads a file and returns the body together with the
filename and content-type from the response headers.

### Get

```go
// Get retrieves file metadata by ID.
// Returns the complete file metadata including file ID, original name, MD5, size, ref_count, etc.
// Requirements: 5.1
func (s *FileService) Get(ctx context.Context, workspaceID string, fileID string) (*FileMetadata, error)
```

Get retrieves file metadata by ID.
Returns the complete file metadata including file ID, original name, MD5, size, ref_count, etc.
Requirements: 5.1

### GetBuiltin

```go
// GetBuiltin returns metadata for one system-owned shared object.
func (s *FileService) GetBuiltin(ctx context.Context, fileID string) (*BuiltinFileMetadata, error)
```

GetBuiltin returns metadata for one system-owned shared object.

### Preview

```go
// Preview returns a browser-friendly preview of a file.
// Returns an io.ReadCloser that the caller must close after use.
func (s *FileService) Preview(ctx context.Context, workspaceID string, fileID string) (io.ReadCloser, error)
```

Preview returns a browser-friendly preview of a file.
Returns an io.ReadCloser that the caller must close after use.

### PublishBuiltin

```go
// PublishBuiltin writes one immutable system-owned object. Repeating the call
// with the same ID and bytes is idempotent.
func (s *FileService) PublishBuiltin(ctx context.Context, fileID, filename string, reader io.Reader) (*BuiltinFileMetadata, error)
```

PublishBuiltin writes one immutable system-owned object. Repeating the call
with the same ID and bytes is idempotent.

### Upload

```go
// Upload uploads a file to the specified workspace.
// Returns the file metadata including the generated file ID.
// Requirements: 2.1
func (s *FileService) Upload(ctx context.Context, workspaceID string, filename string, reader io.Reader, opts ...UploadFileOption) (*UploadFileResponse, error)
```

Upload uploads a file to the specified workspace.
Returns the file metadata including the generated file ID.
Requirements: 2.1

### UploadBytes

```go
// UploadBytes uploads binary data as a file.
// This is a convenience method that wraps Upload with a bytes.Reader.
//
// Example:
//
//	data := []byte("Hello, World!")
//	resp, err := client.Files().UploadBytes(ctx, wsID, "hello.txt", data)
func (s *FileService) UploadBytes(ctx context.Context, workspaceID string, filename string, data []byte, opts ...UploadFileOption) (*UploadFileResponse, error)
```

UploadBytes uploads binary data as a file.
This is a convenience method that wraps Upload with a bytes.Reader.

Example:

	data := []byte("Hello, World!")
	resp, err := client.Files().UploadBytes(ctx, wsID, "hello.txt", data)

### UploadFile

```go
// UploadFile uploads a file from the local filesystem.
// The filename in the upload will be the base name of the file path.
//
// Example:
//
//	resp, err := client.Files().UploadFile(ctx, wsID, "/path/to/document.pdf")
func (s *FileService) UploadFile(ctx context.Context, workspaceID string, filePath string, opts ...UploadFileOption) (*UploadFileResponse, error)
```

UploadFile uploads a file from the local filesystem.
The filename in the upload will be the base name of the file path.

Example:

	resp, err := client.Files().UploadFile(ctx, wsID, "/path/to/document.pdf")

### UploadFileWithName

```go
// UploadFileWithName uploads a file from the local filesystem with a custom filename.
// Use this when you want to override the original filename.
//
// Example:
//
//	resp, err := client.Files().UploadFileWithName(ctx, wsID, "/path/to/doc.pdf", "renamed.pdf")
func (s *FileService) UploadFileWithName(ctx context.Context, workspaceID string, filePath string, filename string, opts ...UploadFileOption) (*UploadFileResponse, error)
```

UploadFileWithName uploads a file from the local filesystem with a custom filename.
Use this when you want to override the original filename.

Example:

	resp, err := client.Files().UploadFileWithName(ctx, wsID, "/path/to/doc.pdf", "renamed.pdf")

### UploadPrivateCatalogFile

```go
// UploadPrivateCatalogFile uploads a file into the caller's private Catalog volume.
// Unlike Upload, this always persists a durable Catalog identity (workspace/volume/file).
func (s *FileService) UploadPrivateCatalogFile(ctx context.Context, workspaceID string, filename string, reader io.Reader, opts ...UploadFileOption) (*UploadFileResponse, error)
```

UploadPrivateCatalogFile uploads a file into the caller's private Catalog volume.
Unlike Upload, this always persists a durable Catalog identity (workspace/volume/file).

### UploadPrivateCatalogFileFromPath

```go
// UploadPrivateCatalogFileFromPath uploads a local file into the caller's private Catalog volume.
func (s *FileService) UploadPrivateCatalogFileFromPath(ctx context.Context, workspaceID string, filePath string, opts ...UploadFileOption) (*UploadFileResponse, error)
```

UploadPrivateCatalogFileFromPath uploads a local file into the caller's private Catalog volume.

### UploadPrivateCatalogFileWithName

```go
// UploadPrivateCatalogFileWithName uploads a local file into the private Catalog volume with a custom filename.
func (s *FileService) UploadPrivateCatalogFileWithName(ctx context.Context, workspaceID string, filePath string, filename string, opts ...UploadFileOption) (*UploadFileResponse, error)
```

UploadPrivateCatalogFileWithName uploads a local file into the private Catalog volume with a custom filename.

## GarbageService

GarbageService provides garbage collection operations.

### TriggerGarbageCollection

```go
// TriggerGarbageCollection triggers garbage collection for a workspace.
// Only the workspace owner can trigger garbage collection.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - opts: Optional parameters (WithOrphanFileThreshold, WithGarbageBatchSize)
//
// Returns the garbage collection result or an error.
//
// Example:
//
//	result, err := client.Garbage().TriggerGarbageCollection(ctx, "workspace-id-123",
//	    moi.WithOrphanFileThreshold(48),
//	    moi.WithGarbageBatchSize(50),
//	)
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Cleaned %d orphan files and %d deleted volumes\n",
//	    result.OrphanFilesCleaned, result.DeletedVolumesCleaned)
func (s *GarbageService) TriggerGarbageCollection(ctx context.Context, workspaceID string, opts ...TriggerGarbageCollectionOption) (*GarbageCollectionResult, error)
```

TriggerGarbageCollection triggers garbage collection for a workspace.
Only the workspace owner can trigger garbage collection.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - opts: Optional parameters (WithOrphanFileThreshold, WithGarbageBatchSize)

Returns the garbage collection result or an error.

Example:

	result, err := client.Garbage().TriggerGarbageCollection(ctx, "workspace-id-123",
	    moi.WithOrphanFileThreshold(48),
	    moi.WithGarbageBatchSize(50),
	)
	if err != nil {
	    return err
	}
	fmt.Printf("Cleaned %d orphan files and %d deleted volumes\n",
	    result.OrphanFilesCleaned, result.DeletedVolumesCleaned)

## LLMService

LLMService provides LLM operations scoped to a workspace.
Obtain it via client.LLM(workspaceID). Covers sessions, messages, tags,
backends, router-config, and chat completions.

### AddMessageTagRelation

```go
// AddMessageTagRelation adds a tag to a message.
//
// Parameters:
//   - ctx: Context for the request
//   - messageID: The message ID
//   - opts: Options (WithRelationTagSource, WithRelationTagName)
//
// Returns an error if the operation fails.
func (s *LLMService) AddMessageTagRelation(ctx context.Context, messageID int64, opts ...TagRelationOption) error
```

AddMessageTagRelation adds a tag to a message.

Parameters:
  - ctx: Context for the request
  - messageID: The message ID
  - opts: Options (WithRelationTagSource, WithRelationTagName)

Returns an error if the operation fails.

### AddSessionTagRelation

```go
// AddSessionTagRelation adds a tag to a session.
//
// Parameters:
//   - ctx: Context for the request
//   - sessionID: The session ID
//   - opts: Options (WithRelationTagSource, WithRelationTagName)
//
// Returns an error if the operation fails.
func (s *LLMService) AddSessionTagRelation(ctx context.Context, sessionID int64, opts ...TagRelationOption) error
```

AddSessionTagRelation adds a tag to a session.

Parameters:
  - ctx: Context for the request
  - sessionID: The session ID
  - opts: Options (WithRelationTagSource, WithRelationTagName)

Returns an error if the operation fails.

### AppendModifiedResponse

```go
// AppendModifiedResponse appends to the modified response of a message.
//
// Parameters:
//   - ctx: Context for the request
//   - sessionID, messageID: Session and message IDs
//   - opts: Options (WithAppendContent)
//
// Returns an error if the operation fails.
func (s *LLMService) AppendModifiedResponse(ctx context.Context, sessionID, messageID int64, opts ...AppendModifiedResponseOption) error
```

AppendModifiedResponse appends to the modified response of a message.

Parameters:
  - ctx: Context for the request
  - sessionID, messageID: Session and message IDs
  - opts: Options (WithAppendContent)

Returns an error if the operation fails.

### BaseURL

```go
// BaseURL returns the absolute LLM base URL for OpenAI-compatible clients.
func (s *LLMService) BaseURL() string
```

BaseURL returns the absolute LLM base URL for OpenAI-compatible clients.

### ChatCompletion

```go
// ChatCompletion sends a streaming chat completion request (SSE) and returns a channel of content deltas.
// The server may send ": heartbeat" comment lines every ~10s; the client ignores them.
// The channel is closed when the stream ends or an error occurs. The caller must consume the channel
// until it is closed (or context is cancelled).
//
// Parameters:
//   - ctx: Context for the request; cancellation closes the stream
//   - question: User question (required)
//   - model: Model name (required)
//   - opts: Optional (WithChatTemperature, WithChatMaxTokens, WithChatExtra, etc.)
//
// Returns a receive-only channel of content delta strings, and an error if the request could not be started.
//
// Example:
//
//	ch, err := client.LLM(workspaceID).ChatCompletion(ctx, "Explain Go", "gpt-4")
//	if err != nil { return err }
//	for delta := range ch {
//	    fmt.Print(delta)
//	}
func (s *LLMService) ChatCompletion(ctx context.Context, question, model string, opts ...ChatCompletionOption) (<-chan string, error)
```

ChatCompletion sends a streaming chat completion request (SSE) and returns a channel of content deltas.
The server may send ": heartbeat" comment lines every ~10s; the client ignores them.
The channel is closed when the stream ends or an error occurs. The caller must consume the channel
until it is closed (or context is cancelled).

Parameters:
  - ctx: Context for the request; cancellation closes the stream
  - question: User question (required)
  - model: Model name (required)
  - opts: Optional (WithChatTemperature, WithChatMaxTokens, WithChatExtra, etc.)

Returns a receive-only channel of content delta strings, and an error if the request could not be started.

Example:

	ch, err := client.LLM(workspaceID).ChatCompletion(ctx, "Explain Go", "gpt-4")
	if err != nil { return err }
	for delta := range ch {
	    fmt.Print(delta)
	}

### ChatCompletionOnce

```go
// ChatCompletionOnce sends a non-streaming chat completion (stream:false) and returns the raw JSON body
// from the server (OpenAI-compatible). Use WithChatMessages for multimodal content (image_url + text),
// consistent with workflow_be vision calls (e.g. vLLM /v1/chat/completions).
func (s *LLMService) ChatCompletionOnce(ctx context.Context, model string, opts ...ChatCompletionOption) ([]byte, error)
```

ChatCompletionOnce sends a non-streaming chat completion (stream:false) and returns the raw JSON body
from the server (OpenAI-compatible). Use WithChatMessages for multimodal content (image_url + text),
consistent with workflow_be vision calls (e.g. vLLM /v1/chat/completions).

### ChatCompletionText

```go
// ChatCompletionText sends a streaming chat completion request and returns the
// aggregated assistant text. Unlike ChatCompletion's legacy channel API, stream
// read and provider SSE errors are returned to the caller.
func (s *LLMService) ChatCompletionText(ctx context.Context, question, model string, opts ...ChatCompletionOption) (string, error)
```

ChatCompletionText sends a streaming chat completion request and returns the
aggregated assistant text. Unlike ChatCompletion's legacy channel API, stream
read and provider SSE errors are returned to the caller.

### CreateBackend

```go
// CreateBackend creates a backend (requires PERM_MODEL_RESOURCE_CREATE or workspace admin).
//
// Parameters:
//   - ctx: Context for the request
//   - opts: Options (WithBackendName, WithBackendType, WithBackendAPIKey, WithBackendTimeout, WithBackendModels)
//
// Returns the created backend or an error.
func (s *LLMService) CreateBackend(ctx context.Context, opts ...CreateBackendOption) (*catalog.Backend, error)
```

CreateBackend creates a backend (requires PERM_MODEL_RESOURCE_CREATE or workspace admin).

Parameters:
  - ctx: Context for the request
  - opts: Options (WithBackendName, WithBackendType, WithBackendAPIKey, WithBackendTimeout, WithBackendModels)

Returns the created backend or an error.

### CreateEmbeddingBackend

```go
// CreateEmbeddingBackend creates an embedding backend (requires PERM_MODEL_RESOURCE_CREATE or workspace admin).
func (s *LLMService) CreateEmbeddingBackend(ctx context.Context, opts ...CreateBackendOption) (*catalog.Backend, error)
```

CreateEmbeddingBackend creates an embedding backend (requires PERM_MODEL_RESOURCE_CREATE or workspace admin).

### CreateEmbeddingEndpoint

```go
// CreateEmbeddingEndpoint adds an endpoint to an embedding backend (requires PERM_MODEL_RESOURCE_CREATE or PERM_MODEL_RESOURCE_UPDATE, or workspace admin).
func (s *LLMService) CreateEmbeddingEndpoint(ctx context.Context, backendID int64, opts ...CreateEndpointOption) (*catalog.BackendEndpoint, error)
```

CreateEmbeddingEndpoint adds an endpoint to an embedding backend (requires PERM_MODEL_RESOURCE_CREATE or PERM_MODEL_RESOURCE_UPDATE, or workspace admin).

### CreateEndpoint

```go
// CreateEndpoint adds an endpoint to a backend (requires PERM_MODEL_RESOURCE_CREATE or PERM_MODEL_RESOURCE_UPDATE, or workspace admin).
//
// Parameters:
//   - ctx: Context for the request
//   - backendID: The backend ID
//   - opts: Options (WithEndpointAddress)
//
// Returns the created endpoint or an error.
func (s *LLMService) CreateEndpoint(ctx context.Context, backendID int64, opts ...CreateEndpointOption) (*catalog.BackendEndpoint, error)
```

CreateEndpoint adds an endpoint to a backend (requires PERM_MODEL_RESOURCE_CREATE or PERM_MODEL_RESOURCE_UPDATE, or workspace admin).

Parameters:
  - ctx: Context for the request
  - backendID: The backend ID
  - opts: Options (WithEndpointAddress)

Returns the created endpoint or an error.

### CreateMessage

```go
// CreateMessage creates a message in a session.
//
// Parameters:
//   - ctx: Context for the request
//   - sessionID: The session ID
//   - msg: The message content (SessionId will be set automatically; must not be nil)
//
// Returns the created message or an error.
//
// Example:
//
//	msg, err := client.LLM(workspaceID).CreateMessage(ctx, 123, &catalog.ChatMessage{
//	    Role: catalog.MessageRole_USER, Content: "Hello",
//	})
func (s *LLMService) CreateMessage(ctx context.Context, sessionID int64, msg *catalog.ChatMessage) (*catalog.ChatMessage, error)
```

CreateMessage creates a message in a session.

Parameters:
  - ctx: Context for the request
  - sessionID: The session ID
  - msg: The message content (SessionId will be set automatically; must not be nil)

Returns the created message or an error.

Example:

	msg, err := client.LLM(workspaceID).CreateMessage(ctx, 123, &catalog.ChatMessage{
	    Role: catalog.MessageRole_USER, Content: "Hello",
	})

### CreateSession

```go
// CreateSession creates a session in the workspace.
//
// Parameters:
//   - ctx: Context for the request
//   - title: Session title (required)
//   - opts: Optional parameters (WithSessionSource, WithSessionConfig)
//
// Returns the created session or an error.
//
// Example:
//
//	session, err := client.LLM(workspaceID).CreateSession(ctx, "My Chat",
//	    moi.WithSessionSource("cli"),
//	)
func (s *LLMService) CreateSession(ctx context.Context, title string, opts ...CreateSessionOption) (*catalog.Session, error)
```

CreateSession creates a session in the workspace.

Parameters:
  - ctx: Context for the request
  - title: Session title (required)
  - opts: Optional parameters (WithSessionSource, WithSessionConfig)

Returns the created session or an error.

Example:

	session, err := client.LLM(workspaceID).CreateSession(ctx, "My Chat",
	    moi.WithSessionSource("cli"),
	)

### CreateTag

```go
// CreateTag creates a tag (or gets existing by source+name).
//
// Parameters:
//   - ctx: Context for the request
//   - opts: Options (WithTagSource, WithTagName)
//
// Returns the created or existing tag or an error.
//
// Example:
//
//	tag, err := client.LLM(workspaceID).CreateTag(ctx,
//	    moi.WithTagSource("cli"), moi.WithTagName("important"),
//	)
func (s *LLMService) CreateTag(ctx context.Context, opts ...CreateTagOption) (*catalog.Tag, error)
```

CreateTag creates a tag (or gets existing by source+name).

Parameters:
  - ctx: Context for the request
  - opts: Options (WithTagSource, WithTagName)

Returns the created or existing tag or an error.

Example:

	tag, err := client.LLM(workspaceID).CreateTag(ctx,
	    moi.WithTagSource("cli"), moi.WithTagName("important"),
	)

### DeleteBackend

```go
// DeleteBackend deletes a backend (requires PERM_MODEL_RESOURCE_DELETE or workspace admin).
//
// Parameters:
//   - ctx: Context for the request
//   - backendID: The backend ID
//
// Returns an error if the deletion fails.
func (s *LLMService) DeleteBackend(ctx context.Context, backendID int64) error
```

DeleteBackend deletes a backend (requires PERM_MODEL_RESOURCE_DELETE or workspace admin).

Parameters:
  - ctx: Context for the request
  - backendID: The backend ID

Returns an error if the deletion fails.

### DeleteEmbeddingBackend

```go
// DeleteEmbeddingBackend deletes an embedding backend (requires PERM_MODEL_RESOURCE_DELETE or workspace admin).
func (s *LLMService) DeleteEmbeddingBackend(ctx context.Context, backendID int64) error
```

DeleteEmbeddingBackend deletes an embedding backend (requires PERM_MODEL_RESOURCE_DELETE or workspace admin).

### DeleteSession

```go
// DeleteSession deletes a session by ID.
//
// Parameters:
//   - ctx: Context for the request
//   - sessionID: The session ID
//
// Returns an error if the deletion fails.
//
// Example:
//
//	err := client.LLM(workspaceID).DeleteSession(ctx, 123)
func (s *LLMService) DeleteSession(ctx context.Context, sessionID int64) error
```

DeleteSession deletes a session by ID.

Parameters:
  - ctx: Context for the request
  - sessionID: The session ID

Returns an error if the deletion fails.

Example:

	err := client.LLM(workspaceID).DeleteSession(ctx, 123)

### DeleteTag

```go
// DeleteTag deletes a tag by source and name.
//
// Parameters:
//   - ctx: Context for the request
//   - source, name: Tag source and name
//
// Returns an error if the deletion fails.
//
// Example:
//
//	err := client.LLM(workspaceID).DeleteTag(ctx, "cli", "old-tag")
func (s *LLMService) DeleteTag(ctx context.Context, source, name string) error
```

DeleteTag deletes a tag by source and name.

Parameters:
  - ctx: Context for the request
  - source, name: Tag source and name

Returns an error if the deletion fails.

Example:

	err := client.LLM(workspaceID).DeleteTag(ctx, "cli", "old-tag")

### GetBackend

```go
// GetBackend retrieves a backend by ID with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).
//
// Parameters:
//   - ctx: Context for the request
//   - backendID: The backend ID
//
// Returns the backend or an error.
func (s *LLMService) GetBackend(ctx context.Context, backendID int64) (*catalog.Backend, error)
```

GetBackend retrieves a backend by ID with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).

Parameters:
  - ctx: Context for the request
  - backendID: The backend ID

Returns the backend or an error.

### GetEmbeddingBackend

```go
// GetEmbeddingBackend retrieves an embedding backend by ID with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).
func (s *LLMService) GetEmbeddingBackend(ctx context.Context, backendID int64) (*catalog.Backend, error)
```

GetEmbeddingBackend retrieves an embedding backend by ID with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).

### GetEmbeddingRouterConfig

```go
// GetEmbeddingRouterConfig retrieves the embedding router config.
func (s *LLMService) GetEmbeddingRouterConfig(ctx context.Context) (*catalog.GetRouterConfigResponse, error)
```

GetEmbeddingRouterConfig retrieves the embedding router config.

### GetMessage

```go
// GetMessage retrieves a message by ID.
//
// Parameters:
//   - ctx: Context for the request
//   - messageID: The message ID
//
// Returns the message or an error.
//
// Example:
//
//	msg, err := client.LLM(workspaceID).GetMessage(ctx, 456)
func (s *LLMService) GetMessage(ctx context.Context, messageID int64) (*catalog.ChatMessage, error)
```

GetMessage retrieves a message by ID.

Parameters:
  - ctx: Context for the request
  - messageID: The message ID

Returns the message or an error.

Example:

	msg, err := client.LLM(workspaceID).GetMessage(ctx, 456)

### GetRouterConfig

```go
// GetRouterConfig retrieves the router config (requires PERM_MODEL_RESOURCE_READ or workspace admin).
//
// Parameters:
//   - ctx: Context for the request
//
// Returns the router config or an error.
//
// Example:
//
//	config, err := client.LLM(workspaceID).GetRouterConfig(ctx)
func (s *LLMService) GetRouterConfig(ctx context.Context) (*catalog.GetRouterConfigResponse, error)
```

GetRouterConfig retrieves the router config (requires PERM_MODEL_RESOURCE_READ or workspace admin).

Parameters:
  - ctx: Context for the request

Returns the router config or an error.

Example:

	config, err := client.LLM(workspaceID).GetRouterConfig(ctx)

### GetSession

```go
// GetSession retrieves a session by ID.
//
// Parameters:
//   - ctx: Context for the request
//   - sessionID: The session ID
//
// Returns the session or an error (e.g. ErrNotFound).
//
// Example:
//
//	session, err := client.LLM(workspaceID).GetSession(ctx, 123)
func (s *LLMService) GetSession(ctx context.Context, sessionID int64) (*catalog.Session, error)
```

GetSession retrieves a session by ID.

Parameters:
  - ctx: Context for the request
  - sessionID: The session ID

Returns the session or an error (e.g. ErrNotFound).

Example:

	session, err := client.LLM(workspaceID).GetSession(ctx, 123)

### LatestMessageID

```go
// LatestMessageID returns the latest message ID for a session.
//
// Parameters:
//   - ctx: Context for the request
//   - sessionID: The session ID
//   - opts: Optional parameters (WithCompletedOnly)
//
// Returns the latest message ID or an error.
//
// Example:
//
//	id, err := client.LLM(workspaceID).LatestMessageID(ctx, 123, moi.WithCompletedOnly())
func (s *LLMService) LatestMessageID(ctx context.Context, sessionID int64, opts ...LatestMessageOption) (int64, error)
```

LatestMessageID returns the latest message ID for a session.

Parameters:
  - ctx: Context for the request
  - sessionID: The session ID
  - opts: Optional parameters (WithCompletedOnly)

Returns the latest message ID or an error.

Example:

	id, err := client.LLM(workspaceID).LatestMessageID(ctx, 123, moi.WithCompletedOnly())

### ListBackends

```go
// ListBackends lists backends in the workspace with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).
//
// Parameters:
//   - ctx: Context for the request
//
// Returns the list response or an error.
//
// Example:
//
//	resp, err := client.LLM(workspaceID).ListBackends(ctx)
func (s *LLMService) ListBackends(ctx context.Context) (*catalog.ListBackendsResponse, error)
```

ListBackends lists backends in the workspace with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).

Parameters:
  - ctx: Context for the request

Returns the list response or an error.

Example:

	resp, err := client.LLM(workspaceID).ListBackends(ctx)

### ListEmbeddingBackends

```go
// ListEmbeddingBackends lists all embedding backends with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).
func (s *LLMService) ListEmbeddingBackends(ctx context.Context) (*catalog.ListBackendsResponse, error)
```

ListEmbeddingBackends lists all embedding backends with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).

### ListEmbeddingEndpoints

```go
// ListEmbeddingEndpoints lists all endpoints for an embedding backend (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).
func (s *LLMService) ListEmbeddingEndpoints(ctx context.Context, backendID int64) ([]*catalog.BackendEndpoint, error)
```

ListEmbeddingEndpoints lists all endpoints for an embedding backend (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).

### ListEmbeddingModels

```go
// ListEmbeddingModels returns the flat list of embedding models available in this
// workspace, one per (backend, model) pair.
func (s *LLMService) ListEmbeddingModels(ctx context.Context) (*ListEmbeddingModelsResponse, error)
```

ListEmbeddingModels returns the flat list of embedding models available in this
workspace, one per (backend, model) pair.

### ListEndpoints

```go
// ListEndpoints lists all endpoints for a backend (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).
//
// Parameters:
//   - ctx: Context for the request
//   - backendID: The backend ID
//
// Returns the endpoints or an error.
func (s *LLMService) ListEndpoints(ctx context.Context, backendID int64) ([]*catalog.BackendEndpoint, error)
```

ListEndpoints lists all endpoints for a backend (requires PERM_MODEL_RESOURCE_READ or legacy PERM_LLM_INVOKE, or workspace admin).

Parameters:
  - ctx: Context for the request
  - backendID: The backend ID

Returns the endpoints or an error.

### ListMessageTags

```go
// ListMessageTags lists tags for a message.
//
// Parameters:
//   - ctx: Context for the request
//   - messageID: The message ID
//
// Returns the list of tags or an error.
func (s *LLMService) ListMessageTags(ctx context.Context, messageID int64) ([]*catalog.Tag, error)
```

ListMessageTags lists tags for a message.

Parameters:
  - ctx: Context for the request
  - messageID: The message ID

Returns the list of tags or an error.

### ListMessages

```go
// ListMessages lists messages in a session.
//
// Parameters:
//   - ctx: Context for the request
//   - sessionID: The session ID
//   - opts: Optional parameters (WithMessagesAfter, WithMessagesLimit, WithMessagesRole, WithMessagesStatus)
//
// Returns the list response or an error.
//
// Example:
//
//	resp, err := client.LLM(workspaceID).ListMessages(ctx, 123,
//	    moi.WithMessagesLimit(50),
//	)
func (s *LLMService) ListMessages(ctx context.Context, sessionID int64, opts ...ListMessagesOption) (*catalog.ListMessagesResponse, error)
```

ListMessages lists messages in a session.

Parameters:
  - ctx: Context for the request
  - sessionID: The session ID
  - opts: Optional parameters (WithMessagesAfter, WithMessagesLimit, WithMessagesRole, WithMessagesStatus)

Returns the list response or an error.

Example:

	resp, err := client.LLM(workspaceID).ListMessages(ctx, 123,
	    moi.WithMessagesLimit(50),
	)

### ListModels

```go
// ListModels returns the flat list of models available in this workspace.
func (s *LLMService) ListModels(ctx context.Context, opts ...ListWorkspaceModelOption) (*ListModelsResponse, error)
```

ListModels returns the flat list of models available in this workspace.

### ListSessionTags

```go
// ListSessionTags lists tags for a session.
//
// Parameters:
//   - ctx: Context for the request
//   - sessionID: The session ID
//
// Returns the list of tags or an error.
//
// Example:
//
//	tags, err := client.LLM(workspaceID).ListSessionTags(ctx, 123)
func (s *LLMService) ListSessionTags(ctx context.Context, sessionID int64) ([]*catalog.Tag, error)
```

ListSessionTags lists tags for a session.

Parameters:
  - ctx: Context for the request
  - sessionID: The session ID

Returns the list of tags or an error.

Example:

	tags, err := client.LLM(workspaceID).ListSessionTags(ctx, 123)

### ListSessions

```go
// ListSessions lists sessions with optional filters and pagination.
//
// Parameters:
//   - ctx: Context for the request
//   - opts: Optional parameters (WithSessionsSource, WithSessionsPage, WithSessionsPageSize)
//
// Returns the list response or an error.
//
// Example:
//
//	resp, err := client.LLM(workspaceID).ListSessions(ctx,
//	    moi.WithSessionsPageSize(20),
//	)
func (s *LLMService) ListSessions(ctx context.Context, opts ...ListSessionsOption) (*catalog.ListSessionsResponse, error)
```

ListSessions lists sessions with optional filters and pagination.

Parameters:
  - ctx: Context for the request
  - opts: Optional parameters (WithSessionsSource, WithSessionsPage, WithSessionsPageSize)

Returns the list response or an error.

Example:

	resp, err := client.LLM(workspaceID).ListSessions(ctx,
	    moi.WithSessionsPageSize(20),
	)

### ListTags

```go
// ListTags lists tags with optional filters and pagination.
//
// Parameters:
//   - ctx: Context for the request
//   - opts: Optional parameters (WithTagsSource, WithTagsKeyword, WithTagsPage, WithTagsPageSize)
//
// Returns the list response or an error.
//
// Example:
//
//	resp, err := client.LLM(workspaceID).ListTags(ctx,
//	    moi.WithTagsPageSize(20),
//	)
func (s *LLMService) ListTags(ctx context.Context, opts ...ListTagsOption) (*catalog.ListTagsResponse, error)
```

ListTags lists tags with optional filters and pagination.

Parameters:
  - ctx: Context for the request
  - opts: Optional parameters (WithTagsSource, WithTagsKeyword, WithTagsPage, WithTagsPageSize)

Returns the list response or an error.

Example:

	resp, err := client.LLM(workspaceID).ListTags(ctx,
	    moi.WithTagsPageSize(20),
	)

### ModifyResponse

```go
// ModifyResponse updates the modified response of a message.
//
// Parameters:
//   - ctx: Context for the request
//   - sessionID, messageID: Session and message IDs
//   - opts: Options (WithModifiedResponse)
//
// Returns an error if the update fails.
func (s *LLMService) ModifyResponse(ctx context.Context, sessionID, messageID int64, opts ...ModifyResponseOption) error
```

ModifyResponse updates the modified response of a message.

Parameters:
  - ctx: Context for the request
  - sessionID, messageID: Session and message IDs
  - opts: Options (WithModifiedResponse)

Returns an error if the update fails.

### PutEmbeddingRouterConfig

```go
// PutEmbeddingRouterConfig updates the embedding router config.
func (s *LLMService) PutEmbeddingRouterConfig(ctx context.Context, opts ...PutRouterConfigOption) (*catalog.GetRouterConfigResponse, error)
```

PutEmbeddingRouterConfig updates the embedding router config.

### PutRouterConfig

```go
// PutRouterConfig updates the router config (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).
//
// Parameters:
//   - ctx: Context for the request
//   - opts: Options (WithRouterStrategy, WithRouterBackendIDs)
//
// Returns the updated config or an error.
func (s *LLMService) PutRouterConfig(ctx context.Context, opts ...PutRouterConfigOption) (*catalog.GetRouterConfigResponse, error)
```

PutRouterConfig updates the router config (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).

Parameters:
  - ctx: Context for the request
  - opts: Options (WithRouterStrategy, WithRouterBackendIDs)

Returns the updated config or an error.

### RemoveMessageTagRelation

```go
// RemoveMessageTagRelation removes a tag from a message.
//
// Parameters:
//   - ctx: Context for the request
//   - messageID: The message ID
//   - opts: Options (WithRelationTagSource, WithRelationTagName)
//
// Returns an error if the operation fails.
func (s *LLMService) RemoveMessageTagRelation(ctx context.Context, messageID int64, opts ...TagRelationOption) error
```

RemoveMessageTagRelation removes a tag from a message.

Parameters:
  - ctx: Context for the request
  - messageID: The message ID
  - opts: Options (WithRelationTagSource, WithRelationTagName)

Returns an error if the operation fails.

### RemoveSessionTagRelation

```go
// RemoveSessionTagRelation removes a tag from a session.
//
// Parameters:
//   - ctx: Context for the request
//   - sessionID: The session ID
//   - opts: Options (WithRelationTagSource, WithRelationTagName)
//
// Returns an error if the operation fails.
func (s *LLMService) RemoveSessionTagRelation(ctx context.Context, sessionID int64, opts ...TagRelationOption) error
```

RemoveSessionTagRelation removes a tag from a session.

Parameters:
  - ctx: Context for the request
  - sessionID: The session ID
  - opts: Options (WithRelationTagSource, WithRelationTagName)

Returns an error if the operation fails.

### ResolveRoute

```go
// ResolveRoute resolves the LLM backend and endpoint for a selected workspace model.
// This is an internal API restricted to system API keys.
func (s *LLMService) ResolveRoute(ctx context.Context, model string, backendID int64) (*catalog.ResolveLLMRouteResponse, error)
```

ResolveRoute resolves the LLM backend and endpoint for a selected workspace model.
This is an internal API restricted to system API keys.

### SetEmbeddingEndpointStatus

```go
// SetEmbeddingEndpointStatus sets the status of an embedding endpoint (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).
func (s *LLMService) SetEmbeddingEndpointStatus(ctx context.Context, backendID, endpointID int64, opts ...SetEndpointStatusOption) error
```

SetEmbeddingEndpointStatus sets the status of an embedding endpoint (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).

### SetEndpointStatus

```go
// SetEndpointStatus sets the status of an endpoint (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).
//
// Parameters:
//   - ctx: Context for the request
//   - backendID, endpointID: Backend and endpoint IDs
//   - opts: Options (WithEndpointStatus)
//
// Returns an error if the operation fails.
func (s *LLMService) SetEndpointStatus(ctx context.Context, backendID, endpointID int64, opts ...SetEndpointStatusOption) error
```

SetEndpointStatus sets the status of an endpoint (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).

Parameters:
  - ctx: Context for the request
  - backendID, endpointID: Backend and endpoint IDs
  - opts: Options (WithEndpointStatus)

Returns an error if the operation fails.

### UpdateBackend

```go
// UpdateBackend updates a backend (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).
//
// Parameters:
//   - ctx: Context for the request
//   - backendID: The backend ID
//   - opts: Options (WithUpdateBackendName, WithUpdateBackendAPIKey, WithUpdateBackendTimeout, WithUpdateBackendModels)
//
// Returns the updated backend or an error.
func (s *LLMService) UpdateBackend(ctx context.Context, backendID int64, opts ...UpdateBackendOption) (*catalog.Backend, error)
```

UpdateBackend updates a backend (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).

Parameters:
  - ctx: Context for the request
  - backendID: The backend ID
  - opts: Options (WithUpdateBackendName, WithUpdateBackendAPIKey, WithUpdateBackendTimeout, WithUpdateBackendModels)

Returns the updated backend or an error.

### UpdateEmbeddingBackend

```go
// UpdateEmbeddingBackend updates an embedding backend (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).
func (s *LLMService) UpdateEmbeddingBackend(ctx context.Context, backendID int64, opts ...UpdateBackendOption) (*catalog.Backend, error)
```

UpdateEmbeddingBackend updates an embedding backend (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).

### UpdateSession

```go
// UpdateSession updates a session.
//
// Parameters:
//   - ctx: Context for the request
//   - sessionID: The session ID
//   - opts: Fields to update (WithSessionTitle, WithSessionConfigUpdate,
//     WithExpectedSessionTitle)
//
// Returns the resulting session or an error. When WithExpectedSessionTitle is
// provided and the title does not match, the current unchanged session is
// returned without an error.
//
// Example:
//
//	session, err := client.LLM(workspaceID).UpdateSession(ctx, 123,
//	    moi.WithSessionTitle("New Title"),
//	)
func (s *LLMService) UpdateSession(ctx context.Context, sessionID int64, opts ...UpdateSessionOption) (*catalog.Session, error)
```

UpdateSession updates a session.

Parameters:
  - ctx: Context for the request
  - sessionID: The session ID
  - opts: Fields to update (WithSessionTitle, WithSessionConfigUpdate,
    WithExpectedSessionTitle)

Returns the resulting session or an error. When WithExpectedSessionTitle is
provided and the title does not match, the current unchanged session is
returned without an error.

Example:

	session, err := client.LLM(workspaceID).UpdateSession(ctx, 123,
	    moi.WithSessionTitle("New Title"),
	)

## MaxComputeService

MaxComputeService provides MaxCompute configuration and metadata management operations.
All operations are scoped to a specific workspace.

### CreateConfig

```go
// CreateConfig creates a new MaxCompute configuration.
//
// Parameters:
//   - ctx: Context for the request
//   - name: The config name (required, unique within workspace)
//   - accessKeyID: The Alibaba Cloud AccessKey ID (required)
//   - accessKeySecret: The Alibaba Cloud AccessKey Secret (required)
//   - endpoint: The MaxCompute endpoint (required)
//   - projectName: The MaxCompute project name (required)
//   - opts: Optional parameters (WithMCRegion)
//
// Returns the created MaxCompute config or an error.
func (s *MaxComputeService) CreateConfig(ctx context.Context, name, accessKeyID, accessKeySecret, endpoint, projectName string, opts ...CreateMCConfigOption) (*catalog.MCConfig, error)
```

CreateConfig creates a new MaxCompute configuration.

Parameters:
  - ctx: Context for the request
  - name: The config name (required, unique within workspace)
  - accessKeyID: The Alibaba Cloud AccessKey ID (required)
  - accessKeySecret: The Alibaba Cloud AccessKey Secret (required)
  - endpoint: The MaxCompute endpoint (required)
  - projectName: The MaxCompute project name (required)
  - opts: Optional parameters (WithMCRegion)

Returns the created MaxCompute config or an error.

### DeleteConfig

```go
// DeleteConfig deletes a MaxCompute configuration by ID.
// This will also cascade delete all associated metadata (databases, tables, columns).
func (s *MaxComputeService) DeleteConfig(ctx context.Context, configID int64) error
```

DeleteConfig deletes a MaxCompute configuration by ID.
This will also cascade delete all associated metadata (databases, tables, columns).

### GetConfig

```go
// GetConfig retrieves a MaxCompute configuration by ID.
func (s *MaxComputeService) GetConfig(ctx context.Context, configID int64) (*catalog.MCConfig, error)
```

GetConfig retrieves a MaxCompute configuration by ID.

### GetDatabase

```go
// GetDatabase retrieves a MaxCompute database by ID.
func (s *MaxComputeService) GetDatabase(ctx context.Context, configID, databaseID int64) (*catalog.MCDatabase, error)
```

GetDatabase retrieves a MaxCompute database by ID.

### GetTable

```go
// GetTable retrieves a MaxCompute table by ID, including its columns.
func (s *MaxComputeService) GetTable(ctx context.Context, configID, databaseID, tableID int64) (*catalog.MCTable, error)
```

GetTable retrieves a MaxCompute table by ID, including its columns.

### HealthCheck

```go
// HealthCheck checks the health of a MaxCompute connection.
func (s *MaxComputeService) HealthCheck(ctx context.Context, configID int64) (*catalog.MCHealthCheckResponse, error)
```

HealthCheck checks the health of a MaxCompute connection.

### ListConfigs

```go
// ListConfigs lists all MaxCompute configurations in the workspace.
func (s *MaxComputeService) ListConfigs(ctx context.Context, opts ...ListOption) (*catalog.ListMCConfigsResponse, error)
```

ListConfigs lists all MaxCompute configurations in the workspace.

### ListDatabases

```go
// ListDatabases lists all databases synced from a MaxCompute configuration.
func (s *MaxComputeService) ListDatabases(ctx context.Context, configID int64, opts ...ListOption) (*catalog.ListMCDatabasesResponse, error)
```

ListDatabases lists all databases synced from a MaxCompute configuration.

### ListTables

```go
// ListTables lists all tables in a MaxCompute database.
func (s *MaxComputeService) ListTables(ctx context.Context, configID, databaseID int64, opts ...ListOption) (*catalog.ListMCTablesResponse, error)
```

ListTables lists all tables in a MaxCompute database.

### StopSync

```go
// StopSync cancels the periodic sync workflow for the specified config.
//
// Parameters:
//   - ctx: Context for the request
//   - configID: The MaxCompute config ID
//
// Returns an error if the operation fails.
func (s *MaxComputeService) StopSync(ctx context.Context, configID int64) error
```

StopSync cancels the periodic sync workflow for the specified config.

Parameters:
  - ctx: Context for the request
  - configID: The MaxCompute config ID

Returns an error if the operation fails.

### SyncMetadata

```go
// SyncMetadata creates a periodic sync workflow for a MaxCompute project.
// The workflow runs on the specified cron schedule. Use StopSync to cancel.
//
// Parameters:
//   - ctx: Context for the request
//   - configID: The MaxCompute config ID
//   - projectName: The name of the MaxCompute project to sync
//   - cronExpression: Cron expression for periodic sync schedule (e.g. "0 */6 * * *")
//
// Returns the sync result or an error.
func (s *MaxComputeService) SyncMetadata(ctx context.Context, configID int64, projectName, cronExpression string) (*catalog.SyncMCMetadataResponse, error)
```

SyncMetadata creates a periodic sync workflow for a MaxCompute project.
The workflow runs on the specified cron schedule. Use StopSync to cancel.

Parameters:
  - ctx: Context for the request
  - configID: The MaxCompute config ID
  - projectName: The name of the MaxCompute project to sync
  - cronExpression: Cron expression for periodic sync schedule (e.g. "0 */6 * * *")

Returns the sync result or an error.

### UpdateConfig

```go
// UpdateConfig updates an existing MaxCompute configuration.
func (s *MaxComputeService) UpdateConfig(ctx context.Context, configID int64, opts ...UpdateMCConfigOption) (*catalog.MCConfig, error)
```

UpdateConfig updates an existing MaxCompute configuration.

## MowlLineageService

MowlLineageService provides atomic workflow-lineage APIs under one workspace.

### CancelRerun

```go
// CancelRerun cancels a catalog-created rerun branch.
func (s *MowlLineageService) CancelRerun(ctx context.Context, rerunID string) (*RerunCancelResponse, error)
```

CancelRerun cancels a catalog-created rerun branch.

### CreateBasicRerun

```go
// CreateBasicRerun previews the current immutable rerun contract and creates a
// basic rerun pinned to that hash.
func (s *MowlLineageService) CreateBasicRerun(ctx context.Context, caseID, nodeExecutionID string, req *mowlpb.RerunRequest) (*RerunCreateResponse, error)
```

CreateBasicRerun previews the current immutable rerun contract and creates a
basic rerun pinned to that hash.

### CreateBasicRerunWithContract

```go
// CreateBasicRerunWithContract creates a basic rerun only if the contract
// prepared by Core still matches expectedContractHash.
func (s *MowlLineageService) CreateBasicRerunWithContract(ctx context.Context, caseID, nodeExecutionID, expectedContractHash string, req *mowlpb.RerunRequest) (*RerunCreateResponse, error)
```

CreateBasicRerunWithContract creates a basic rerun only if the contract
prepared by Core still matches expectedContractHash.

### CreateFinalArtifactBlockRevision

```go
// CreateFinalArtifactBlockRevision creates a revision for one final artifact block.
func (s *MowlLineageService) CreateFinalArtifactBlockRevision(ctx context.Context, caseID, rootAssetID, outputArtifactID, blockID string, req *mowlpb.CreateRevisionRequest) (*mowlpb.OutputBlockRevision, error)
```

CreateFinalArtifactBlockRevision creates a revision for one final artifact block.

### CreateNodeOutputBlockRevision

```go
// CreateNodeOutputBlockRevision creates a revision for one node output block.
func (s *MowlLineageService) CreateNodeOutputBlockRevision(ctx context.Context, caseID, rootAssetID, nodeExecutionID, blockID string, req *mowlpb.CreateRevisionRequest) (*mowlpb.OutputBlockRevision, error)
```

CreateNodeOutputBlockRevision creates a revision for one node output block.

### CreateRevisionRerun

```go
// CreateRevisionRerun creates a downstream rerun branch from a revised node output emission.
func (s *MowlLineageService) CreateRevisionRerun(ctx context.Context, caseID, rootAssetID, nodeExecutionID, targetOutputEmissionID string, req *mowlpb.RevisionRerunRequest) (*RerunCreateResponse, error)
```

CreateRevisionRerun creates a downstream rerun branch from a revised node output emission.

### GetCaseArtifact

```go
// GetCaseArtifact returns artifact lineage detail for one asset case.
func (s *MowlLineageService) GetCaseArtifact(ctx context.Context, rootAssetID, caseID string) (*mowlpb.AssetArtifactDetailResponse, error)
```

GetCaseArtifact returns artifact lineage detail for one asset case.

### GetCaseInvocation

```go
// GetCaseInvocation returns case-level invocation input and vars snapshots.
func (s *MowlLineageService) GetCaseInvocation(ctx context.Context, caseID string) (*mowlpb.CaseInvocationResponse, error)
```

GetCaseInvocation returns case-level invocation input and vars snapshots.

### GetCaseWorkItem

```go
// GetCaseWorkItem returns one workitem lineage snapshot detail.
func (s *MowlLineageService) GetCaseWorkItem(ctx context.Context, caseID, workitemID string, parallelIndex int32) (*mowlpb.GetCaseWorkItemResponse, error)
```

GetCaseWorkItem returns one workitem lineage snapshot detail.

### GetRerun

```go
// GetRerun returns a single rerun plan.
func (s *MowlLineageService) GetRerun(ctx context.Context, rerunID string) (*mowlpb.RerunPlan, error)
```

GetRerun returns a single rerun plan.

### GetRerunByWorkflowExecution

```go
// GetRerunByWorkflowExecution returns the rerun plan that produced a workflow
// execution. It returns nil when the execution is not a rerun branch.
func (s *MowlLineageService) GetRerunByWorkflowExecution(ctx context.Context, workflowExecutionID string) (*mowlpb.RerunPlan, error)
```

GetRerunByWorkflowExecution returns the rerun plan that produced a workflow
execution. It returns nil when the execution is not a rerun branch.

### GetWorkItemArtifactScope

```go
// GetWorkItemArtifactScope returns one workitem narrowed to one root asset scope.
func (s *MowlLineageService) GetWorkItemArtifactScope(ctx context.Context, caseID, workitemID, rootAssetID string, parallelIndex int32, previewLimit int32) (*mowlpb.ArtifactScopeResponse, error)
```

GetWorkItemArtifactScope returns one workitem narrowed to one root asset scope.

### ListAssetCases

```go
// ListAssetCases lists lineage cases for the specified root asset.
func (s *MowlLineageService) ListAssetCases(ctx context.Context, rootAssetID string) (*mowlpb.ListAssetCasesResponse, error)
```

ListAssetCases lists lineage cases for the specified root asset.

### ListCaseWorkItems

```go
// ListCaseWorkItems lists workitem lineage snapshots for one case.
func (s *MowlLineageService) ListCaseWorkItems(ctx context.Context, caseID, rootAssetID string, includeRuntimeInputSnapshot bool) (*mowlpb.ListCaseWorkItemsResponse, error)
```

ListCaseWorkItems lists workitem lineage snapshots for one case.

### ListFinalArtifactBlockRevisions

```go
// ListFinalArtifactBlockRevisions lists revisions for one final artifact block.
func (s *MowlLineageService) ListFinalArtifactBlockRevisions(ctx context.Context, caseID, rootAssetID, outputArtifactID, blockID string) (*ListBlockRevisionsResponse, error)
```

ListFinalArtifactBlockRevisions lists revisions for one final artifact block.

### ListFinalArtifactBlocks

```go
// ListFinalArtifactBlocks lists output blocks for a final artifact target.
func (s *MowlLineageService) ListFinalArtifactBlocks(ctx context.Context, caseID, rootAssetID, outputArtifactID string) (*ListOutputBlocksResponse, error)
```

ListFinalArtifactBlocks lists output blocks for a final artifact target.

### ListFinalOutputArtifacts

```go
// ListFinalOutputArtifacts lists final output artifacts for a case-scoped root asset.
func (s *MowlLineageService) ListFinalOutputArtifacts(ctx context.Context, caseID, rootAssetID string) (*ListFinalOutputArtifactsResponse, error)
```

ListFinalOutputArtifacts lists final output artifacts for a case-scoped root asset.

### ListNodeOutputBlockRevisions

```go
// ListNodeOutputBlockRevisions lists revisions for one node output block.
func (s *MowlLineageService) ListNodeOutputBlockRevisions(ctx context.Context, caseID, rootAssetID, nodeExecutionID, blockID string) (*ListBlockRevisionsResponse, error)
```

ListNodeOutputBlockRevisions lists revisions for one node output block.

### ListNodeOutputBlocks

```go
// ListNodeOutputBlocks lists output blocks for a node output target.
func (s *MowlLineageService) ListNodeOutputBlocks(ctx context.Context, caseID, rootAssetID, nodeExecutionID string) (*ListOutputBlocksResponse, error)
```

ListNodeOutputBlocks lists output blocks for a node output target.

### ListRerunsBySourceExecution

```go
// ListRerunsBySourceExecution lists rerun branches created from a source
// workflow execution.
func (s *MowlLineageService) ListRerunsBySourceExecution(ctx context.Context, sourceWorkflowExecutionID string) (*mowlpb.ListRerunsResponse, error)
```

ListRerunsBySourceExecution lists rerun branches created from a source
workflow execution.

### PreviewBasicRerun

```go
// PreviewBasicRerun validates and fingerprints the contract used by create.
func (s *MowlLineageService) PreviewBasicRerun(ctx context.Context, caseID, nodeExecutionID string) (*RerunPreviewResponse, error)
```

PreviewBasicRerun validates and fingerprints the contract used by create.

### StartRerun

```go
// StartRerun marks a rerun branch ready and submits its bootstrap tokens to Mowl.
func (s *MowlLineageService) StartRerun(ctx context.Context, rerunID string) (*RerunStartResponse, error)
```

StartRerun marks a rerun branch ready and submits its bootstrap tokens to Mowl.

### SwitchFinalArtifactEffectiveRevisions

```go
// SwitchFinalArtifactEffectiveRevisions switches effective revisions for a final artifact target.
func (s *MowlLineageService) SwitchFinalArtifactEffectiveRevisions(ctx context.Context, caseID, rootAssetID, outputArtifactID string, req *mowlpb.SwitchEffectiveRevisionsRequest) (*SwitchEffectiveRevisionsResponse, error)
```

SwitchFinalArtifactEffectiveRevisions switches effective revisions for a final artifact target.

### SwitchNodeOutputEffectiveRevisions

```go
// SwitchNodeOutputEffectiveRevisions switches effective revisions for a node output target.
func (s *MowlLineageService) SwitchNodeOutputEffectiveRevisions(ctx context.Context, caseID, rootAssetID, nodeExecutionID string, req *mowlpb.SwitchEffectiveRevisionsRequest) (*SwitchEffectiveRevisionsResponse, error)
```

SwitchNodeOutputEffectiveRevisions switches effective revisions for a node output target.

### UpdateFinalArtifactBlockRevisionStatus

```go
// UpdateFinalArtifactBlockRevisionStatus updates a final artifact block revision lifecycle status.
func (s *MowlLineageService) UpdateFinalArtifactBlockRevisionStatus(ctx context.Context, caseID, rootAssetID, outputArtifactID, blockID, revisionID string, req *mowlpb.UpdateRevisionStatusRequest) (*UpdateRevisionStatusResponse, error)
```

UpdateFinalArtifactBlockRevisionStatus updates a final artifact block revision lifecycle status.

### UpdateNodeOutputBlockRevisionStatus

```go
// UpdateNodeOutputBlockRevisionStatus updates a node output block revision lifecycle status.
func (s *MowlLineageService) UpdateNodeOutputBlockRevisionStatus(ctx context.Context, caseID, rootAssetID, nodeExecutionID, blockID, revisionID string, req *mowlpb.UpdateRevisionStatusRequest) (*UpdateRevisionStatusResponse, error)
```

UpdateNodeOutputBlockRevisionStatus updates a node output block revision lifecycle status.

## ParseResultService

ParseResultService provides operations for viewing, modifying, and exporting parse results.

### Export

```go
// Export exports parse results to files through the specified parser.
func (s *ParseResultService) Export(
	ctx context.Context,
	parser catalogpb.ParseResultParser,
	fileName string,
	results []*catalogpb.ParseResult,
) ([]*catalogpb.ParseResultsExportFile, error)
```

Export exports parse results to files through the specified parser.

### Modify

```go
// Modify updates a parse result's content through the specified parser.
func (s *ParseResultService) Modify(
	ctx context.Context,
	parser catalogpb.ParseResultParser,
	result *catalogpb.ParseResult,
	content string,
) (*catalogpb.ParseResult, error)
```

Modify updates a parse result's content through the specified parser.

### View

```go
// View renders parse results through the specified parser for preview.
func (s *ParseResultService) View(
	ctx context.Context,
	parser catalogpb.ParseResultParser,
	results []*catalogpb.ParseResult,
) ([]*catalogpb.ParseResultView, error)
```

View renders parse results through the specified parser for preview.

## ParserService

ParserService provides parser backend management scoped to a workspace.
Obtain it via client.Parsers(workspaceID).

### Convert

```go
// Convert converts a document via multipart file upload.
// Calls POST /:id/parsers/convert with multipart/form-data.
func (s *ParserService) Convert(ctx context.Context, fileBytes []byte, outputFormat string, opts ...ParserConvertOption) (*ConvertResponse, error)
```

Convert converts a document via multipart file upload.
Calls POST /:id/parsers/convert with multipart/form-data.

### ConvertByFileID

```go
// ConvertByFileID converts a document referenced by file ID.
// Calls POST /:id/parsers/convert with JSON body.
func (s *ParserService) ConvertByFileID(ctx context.Context, fileID string, outputFormat string, opts ...ParserConvertOption) (*ConvertResponse, error)
```

ConvertByFileID converts a document referenced by file ID.
Calls POST /:id/parsers/convert with JSON body.

### CreateBackend

```go
// CreateBackend creates a parser backend (requires PERM_MODEL_RESOURCE_CREATE or workspace admin).
func (s *ParserService) CreateBackend(ctx context.Context, opts ...CreateParserBackendOption) (*catalog.ParserBackend, error)
```

CreateBackend creates a parser backend (requires PERM_MODEL_RESOURCE_CREATE or workspace admin).

### CreateEndpoint

```go
// CreateEndpoint adds an endpoint to a parser backend (requires PERM_MODEL_RESOURCE_CREATE or PERM_MODEL_RESOURCE_UPDATE, or workspace admin).
func (s *ParserService) CreateEndpoint(ctx context.Context, backendID int64, opts ...CreateEndpointOption) (*catalog.ParserBackendEndpoint, error)
```

CreateEndpoint adds an endpoint to a parser backend (requires PERM_MODEL_RESOURCE_CREATE or PERM_MODEL_RESOURCE_UPDATE, or workspace admin).

### DeleteBackend

```go
// DeleteBackend deletes a parser backend (requires PERM_MODEL_RESOURCE_DELETE or workspace admin).
func (s *ParserService) DeleteBackend(ctx context.Context, backendID int64) error
```

DeleteBackend deletes a parser backend (requires PERM_MODEL_RESOURCE_DELETE or workspace admin).

### GetBackend

```go
// GetBackend retrieves a parser backend by ID with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_PARSER_INVOKE, or workspace admin).
func (s *ParserService) GetBackend(ctx context.Context, backendID int64) (*catalog.ParserBackend, error)
```

GetBackend retrieves a parser backend by ID with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_PARSER_INVOKE, or workspace admin).

### GetRouterConfig

```go
// GetRouterConfig retrieves the parser router config (requires PERM_MODEL_RESOURCE_READ or workspace admin).
func (s *ParserService) GetRouterConfig(ctx context.Context) (*catalog.GetRouterConfigResponse, error)
```

GetRouterConfig retrieves the parser router config (requires PERM_MODEL_RESOURCE_READ or workspace admin).

### ListBackends

```go
// ListBackends lists all parser backends with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_PARSER_INVOKE, or workspace admin).
func (s *ParserService) ListBackends(ctx context.Context) (*catalog.ListParserBackendsResponse, error)
```

ListBackends lists all parser backends with api_key redacted (requires PERM_MODEL_RESOURCE_READ or legacy PERM_PARSER_INVOKE, or workspace admin).

### ListEndpoints

```go
// ListEndpoints lists all endpoints for a parser backend (requires PERM_MODEL_RESOURCE_READ or legacy PERM_PARSER_INVOKE, or workspace admin).
func (s *ParserService) ListEndpoints(ctx context.Context, backendID int64) ([]*catalog.ParserBackendEndpoint, error)
```

ListEndpoints lists all endpoints for a parser backend (requires PERM_MODEL_RESOURCE_READ or legacy PERM_PARSER_INVOKE, or workspace admin).

### Parse

```go
// Parse 调用 Catalog Parser API 解析文件。
// 通过 multipart/form-data 上传文件，Catalog 根据 MIME 类型路由到对应的解析器后端。
// 返回 proto 定义的 ParseFileResponse（text + metadata）。
func (s *ParserService) Parse(ctx context.Context, fileName, mimeType string, fileData []byte, opts ...ParseOption) (*catalog.ParseFileResponse, error)
```

Parse 调用 Catalog Parser API 解析文件。
通过 multipart/form-data 上传文件，Catalog 根据 MIME 类型路由到对应的解析器后端。
返回 proto 定义的 ParseFileResponse（text + metadata）。

### ParseByFileID

```go
// ParseByFileID 按文件 ID 调用 Catalog Parser API 解析文件。
// 通过 JSON body 发送 file_id，Catalog 从文件存储中读取文件并路由到对应的解析器后端。
func (s *ParserService) ParseByFileID(ctx context.Context, fileID string, opts ...ParseOption) (*catalog.ParseFileResponse, error)
```

ParseByFileID 按文件 ID 调用 Catalog Parser API 解析文件。
通过 JSON body 发送 file_id，Catalog 从文件存储中读取文件并路由到对应的解析器后端。

### PutRouterConfig

```go
// PutRouterConfig updates the parser router config (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).
func (s *ParserService) PutRouterConfig(ctx context.Context, opts ...PutRouterConfigOption) (*catalog.GetRouterConfigResponse, error)
```

PutRouterConfig updates the parser router config (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).

### ResolveConvertRoute

```go
// ResolveConvertRoute resolves the office-converter backend and endpoint
// Catalog would use for the given raw converter backend type (e.g. "WPS_HTTP").
// This is an internal API restricted to system API keys; Catalog answers 503
// when no ONLINE endpoint of that type is available.
func (s *ParserService) ResolveConvertRoute(ctx context.Context, backendType string) (*catalog.ResolveParserRouteResponse, error)
```

ResolveConvertRoute resolves the office-converter backend and endpoint
Catalog would use for the given raw converter backend type (e.g. "WPS_HTTP").
This is an internal API restricted to system API keys; Catalog answers 503
when no ONLINE endpoint of that type is available.

### ResolveRoute

```go
// ResolveRoute resolves the parser backend and endpoint Catalog would use for a MIME type.
// This is an internal API restricted to system API keys.
func (s *ParserService) ResolveRoute(ctx context.Context, mimeType string) (*catalog.ResolveParserRouteResponse, error)
```

ResolveRoute resolves the parser backend and endpoint Catalog would use for a MIME type.
This is an internal API restricted to system API keys.

### ResolveRouteExcluding

```go
// ResolveRouteExcluding is ResolveRoute with backends to skip (quota failover).
func (s *ParserService) ResolveRouteExcluding(ctx context.Context, mimeType string, excludeBackendIDs []int64) (*catalog.ResolveParserRouteResponse, error)
```

ResolveRouteExcluding is ResolveRoute with backends to skip (quota failover).

### SetEndpointStatus

```go
// SetEndpointStatus sets the status of a parser endpoint (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).
func (s *ParserService) SetEndpointStatus(ctx context.Context, backendID, endpointID int64, opts ...SetEndpointStatusOption) error
```

SetEndpointStatus sets the status of a parser endpoint (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).

### UpdateBackend

```go
// UpdateBackend updates a parser backend (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).
func (s *ParserService) UpdateBackend(ctx context.Context, backendID int64, opts ...UpdateParserBackendOption) (*catalog.ParserBackend, error)
```

UpdateBackend updates a parser backend (requires PERM_MODEL_RESOURCE_UPDATE or workspace admin).

## RawService

RawService supports the small set of Backend proxy routes whose response
contract is intentionally not decoded by the SDK.

### Do

```go
// Do issues an HTTP request against Catalog without decoding the body.
// Callers own the returned Body and must close it. Authentication and Backend
// execution headers follow the same rules as typed service methods.
func (s *RawService) Do(ctx context.Context, method, path string, body io.Reader, headers http.Header) (*RawResponse, error)
```

Do issues an HTTP request against Catalog without decoding the body.
Callers own the returned Body and must close it. Authentication and Backend
execution headers follow the same rules as typed service methods.

### DoStream

```go
// DoStream issues a streaming HTTP request against Catalog without decoding
// its response. The SDK timeout bounds only the wait for response headers;
// after headers arrive, the caller's context owns the stream lifetime.
func (s *RawService) DoStream(ctx context.Context, method, path string, body io.Reader, headers http.Header) (*RawResponse, error)
```

DoStream issues a streaming HTTP request against Catalog without decoding
its response. The SDK timeout bounds only the wait for response headers;
after headers arrive, the caller's context owns the stream lifetime.

## SemanticModelService

SemanticModelService provides semantic model APIs.

### Create

```go
// Create creates a semantic model.
func (s *SemanticModelService) Create(ctx context.Context, req *SemanticModelUpsertRequest) (*SemanticModel, error)
```

Create creates a semantic model.

### CreateEntry

```go
// CreateEntry creates a semantic entry.
func (s *SemanticModelService) CreateEntry(ctx context.Context, modelID int64, req *SemanticEntryUpsertRequest) (*SemanticEntry, error)
```

CreateEntry creates a semantic entry.

### Delete

```go
// Delete deletes a semantic model.
func (s *SemanticModelService) Delete(ctx context.Context, modelID int64) error
```

Delete deletes a semantic model.

### DeleteEntry

```go
// DeleteEntry deletes a semantic entry.
func (s *SemanticModelService) DeleteEntry(ctx context.Context, modelID, entryID int64) error
```

DeleteEntry deletes a semantic entry.

### Export

```go
// Export exports a semantic model and all of its entries.
func (s *SemanticModelService) Export(ctx context.Context, modelID int64) (*SemanticModelExportResponse, error)
```

Export exports a semantic model and all of its entries.

### Get

```go
// Get retrieves a semantic model by ID.
func (s *SemanticModelService) Get(ctx context.Context, modelID int64) (*SemanticModel, error)
```

Get retrieves a semantic model by ID.

### Import

```go
// Import imports a semantic model and its entries.
func (s *SemanticModelService) Import(ctx context.Context, req *SemanticModelImportRequest) (*SemanticModel, error)
```

Import imports a semantic model and its entries.

### ImportBatch

```go
// ImportBatch imports multiple semantic models in a single /import request.
func (s *SemanticModelService) ImportBatch(ctx context.Context, reqs []SemanticModelImportRequest) (*SemanticModelImportBatchResponse, error)
```

ImportBatch imports multiple semantic models in a single /import request.

### List

```go
// List lists semantic models.
func (s *SemanticModelService) List(ctx context.Context, opts ...ListOption) (*SemanticModelListResponse, error)
```

List lists semantic models.

### ListEntries

```go
// ListEntries lists semantic entries in a semantic model.
func (s *SemanticModelService) ListEntries(ctx context.Context, modelID int64, kind string, opts ...ListOption) (*SemanticEntryListResponse, error)
```

ListEntries lists semantic entries in a semantic model.

### ListTags

```go
// ListTags lists aggregated semantic model tags.
func (s *SemanticModelService) ListTags(ctx context.Context, opts ...ListOption) (*SemanticModelTagListResponse, error)
```

ListTags lists aggregated semantic model tags.

### Update

```go
// Update updates a semantic model.
func (s *SemanticModelService) Update(ctx context.Context, modelID int64, req *SemanticModelUpsertRequest) (*SemanticModelMutationResponse, error)
```

Update updates a semantic model.

### UpdateEntry

```go
// UpdateEntry updates a semantic entry.
func (s *SemanticModelService) UpdateEntry(ctx context.Context, modelID, entryID int64, req *SemanticEntryUpsertRequest) (*SemanticModelMutationResponse, error)
```

UpdateEntry updates a semantic entry.

### Validate

```go
// Validate validates a semantic model.
func (s *SemanticModelService) Validate(ctx context.Context, modelID int64) (*SemanticModelValidateResponse, error)
```

Validate validates a semantic model.

## ServiceAccountService

ServiceAccountService is the server-side UC -> AI Studio catalog facade.
It is not a service-account data-plane client and intentionally does not
expose human role APIs. The catalog routes are read-only; historical write
methods below fail locally and cannot issue an HTTP request.

### DeleteWorkspaceRole

```go
// DeleteWorkspaceRole always returns ErrServiceAccountWriteUnsupported without an HTTP request.
func (s *ServiceAccountService) DeleteWorkspaceRole(ctx context.Context, workspaceID, ucServiceAccountID string, opt ServiceAccountWriteOptions) error
```

DeleteWorkspaceRole always returns ErrServiceAccountWriteUnsupported without an HTTP request.

### ListAssignableRoles

```go
// ListAssignableRoles returns the assignable roles and remote version for a workspace.
func (s *ServiceAccountService) ListAssignableRoles(ctx context.Context, workspaceID string, opt ServiceAccountCatalogOptions) ([]ServiceAccountRole, string, error)
```

ListAssignableRoles returns the assignable roles and remote version for a workspace.

### ListManageableWorkspaces

```go
// ListManageableWorkspaces returns the workspaces available to the asserted human actor.
func (s *ServiceAccountService) ListManageableWorkspaces(ctx context.Context, opt ServiceAccountCatalogOptions) ([]ServiceAccountWorkspace, error)
```

ListManageableWorkspaces returns the workspaces available to the asserted human actor.

### UpsertPrincipal

```go
// UpsertPrincipal always returns ErrServiceAccountWriteUnsupported without an HTTP request.
func (s *ServiceAccountService) UpsertPrincipal(ctx context.Context, workspaceID string, spec ServiceAccountPrincipalSpec, opt ServiceAccountWriteOptions) (*ServiceAccountPrincipalResult, error)
```

UpsertPrincipal always returns ErrServiceAccountWriteUnsupported without an HTTP request.

## StructuredLoadService

StructuredLoadService calls the system structured-load runtime APIs.

### CancelRun

```go
// CancelRun releases a worker-owned run after the worker reaches a safe page boundary.
func (s *StructuredLoadService) CancelRun(ctx context.Context, req *structuredloadpb.CancelRunRequest) (*structuredloadpb.CancelRunResponse, error)
```

CancelRun releases a worker-owned run after the worker reaches a safe page boundary.

### CreateTask

```go
// CreateTask creates the core-owned structured-load task and snapshot.
func (s *StructuredLoadService) CreateTask(ctx context.Context, req *structuredloadpb.CreateStructuredLoadTaskRequest) (*structuredloadpb.CreateStructuredLoadTaskResponse, error)
```

CreateTask creates the core-owned structured-load task and snapshot.

### GetTask

```go
// GetTask reads the core-owned structured-load task and latest run projection.
func (s *StructuredLoadService) GetTask(ctx context.Context, req *structuredloadpb.GetStructuredLoadTaskRequest) (*structuredloadpb.GetStructuredLoadTaskResponse, error)
```

GetTask reads the core-owned structured-load task and latest run projection.

### ListRuns

```go
// ListRuns reads stable paged structured-load runs with committed row totals.
func (s *StructuredLoadService) ListRuns(ctx context.Context, req *structuredloadpb.ListStructuredLoadRunsRequest) (*structuredloadpb.ListStructuredLoadRunsResponse, error)
```

ListRuns reads stable paged structured-load runs with committed row totals.

### SetTaskStatus

```go
// SetTaskStatus persists the backend-owned lifecycle intent used by structured-load scheduling and recovery.
func (s *StructuredLoadService) SetTaskStatus(ctx context.Context, req *structuredloadpb.SetStructuredLoadTaskStatusRequest) (*structuredloadpb.SetStructuredLoadTaskStatusResponse, error)
```

SetTaskStatus persists the backend-owned lifecycle intent used by structured-load scheduling and recovery.

## SystemDefaultAIService

SystemDefaultAIService provides system-level default AI service configuration APIs.
The server accepts only the raw moi-core system API key for these operations.

### GetConfig

```go
// GetConfig returns the system default AI service configuration.
// serviceTypes may be empty to return all services, or contain "llm", "embedding", or "file_parser".
func (s *SystemDefaultAIService) GetConfig(ctx context.Context, serviceTypes ...string) (*SystemDefaultAIConfig, error)
```

GetConfig returns the system default AI service configuration.
serviceTypes may be empty to return all services, or contain "llm", "embedding", or "file_parser".

### PutConfig

```go
// PutConfig replaces the requested system default AI service configurations.
// Services not present in cfg are left unchanged by the server.
func (s *SystemDefaultAIService) PutConfig(ctx context.Context, cfg *SystemDefaultAIConfig) (*SystemDefaultAIConfig, error)
```

PutConfig replaces the requested system default AI service configurations.
Services not present in cfg are left unchanged by the server.

### ReplaceConfig

```go
// ReplaceConfig is an alias for PutConfig.
func (s *SystemDefaultAIService) ReplaceConfig(ctx context.Context, cfg *SystemDefaultAIConfig) (*SystemDefaultAIConfig, error)
```

ReplaceConfig is an alias for PutConfig.

## SystemIAMService

SystemIAMService calls the system-key-only IAM boundary. It deliberately
does not accept a user API key: the Backend has already authenticated the
browser actor and Core trusts the supplied principal only with a verified
system credential.

### CurrentPrincipalRoles

```go
// CurrentPrincipalRoles verifies the authenticated execution user's current
// workspace membership and resolves exactly one lifecycle-valid Effective Role.
func (s *SystemIAMService) CurrentPrincipalRoles(ctx context.Context, req *iampb.CurrentPrincipalRolesRequest) (*iampb.CurrentPrincipalRolesResponse, error)
```

CurrentPrincipalRoles verifies the authenticated execution user's current
workspace membership and resolves exactly one lifecycle-valid Effective Role.

### ProjectTrustedPrincipalAccess

```go
// ProjectTrustedPrincipalAccess resolves one Backend-authenticated user's IAM
// access projection without a second UC personal-access-token validation.
func (s *SystemIAMService) ProjectTrustedPrincipalAccess(ctx context.Context, req *iampb.CurrentPrincipalAccessProjectionRequest) (*iampb.CurrentPrincipalAccessProjectionResponse, error)
```

ProjectTrustedPrincipalAccess resolves one Backend-authenticated user's IAM
access projection without a second UC personal-access-token validation.

## SystemResourceDisplayService

SystemResourceDisplayService provides system API operations for producer-owned display mappings.

### EnsureResourceMappings

```go
// EnsureResourceMappings stores explicit display mappings for producer-created resources.
func (s *SystemResourceDisplayService) EnsureResourceMappings(ctx context.Context, workspaceID string, req *EnsureResourceDisplayMappingsRequest) error
```

EnsureResourceMappings stores explicit display mappings for producer-created resources.

## TaskService

TaskService provides task management operations.
TaskService is always scoped to a workspace via client.Tasks(workspaceID).

Validates: Requirements 6.13-6.17

### Cancel

```go
// Cancel cancels a task.
// Only the task creator can cancel the task.
//
// Parameters:
//   - ctx: Context for the request
//   - taskID: The task ID
//
// Returns an error if the cancellation fails.
//
// Example:
//
//	err := client.Tasks(workspaceID).Cancel(ctx, "task-uuid")
func (s *TaskService) Cancel(ctx context.Context, taskID string) error
```

Cancel cancels a task.
Only the task creator can cancel the task.

Parameters:
  - ctx: Context for the request
  - taskID: The task ID

Returns an error if the cancellation fails.

Example:

	err := client.Tasks(workspaceID).Cancel(ctx, "task-uuid")

### Create

```go
// Create creates a new task.
// Either WithTaskWorkflow or WithTaskWorkflowVersionID must be provided.
//
// Parameters:
//   - ctx: Context for the request
//   - name: The task name (required, unique per user)
//   - opts: Optional parameters (WithTaskWorkflow, WithTaskWorkflowVersionID, WithTaskCronExpression, etc.)
//
// Returns the created task or an error.
//
// Example:
//
//	// Create a one-time task with workflow version
//	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
//	    moi.WithTaskWorkflowVersionID("version-uuid"),
//	    moi.WithTaskData(`{"input": "value"}`),
//	)
//
//	// Create a periodic task with notification
//	task, err := client.Tasks(workspaceID).Create(ctx, "daily-task",
//	    moi.WithTaskWorkflowVersionID("version-uuid"),
//	    moi.WithTaskCronExpression("0 0 * * *"),
//	    moi.WithTaskNotification(moi.NewHTTPNotification("https://callback.example.com").Build()),
//	)
func (s *TaskService) Create(ctx context.Context, name string, opts ...CreateTaskOption) (*mowl.Task, error)
```

Create creates a new task.
Either WithTaskWorkflow or WithTaskWorkflowVersionID must be provided.

Parameters:
  - ctx: Context for the request
  - name: The task name (required, unique per user)
  - opts: Optional parameters (WithTaskWorkflow, WithTaskWorkflowVersionID, WithTaskCronExpression, etc.)

Returns the created task or an error.

Example:

	// Create a one-time task with workflow version
	task, err := client.Tasks(workspaceID).Create(ctx, "my-task",
	    moi.WithTaskWorkflowVersionID("version-uuid"),
	    moi.WithTaskData(`{"input": "value"}`),
	)

	// Create a periodic task with notification
	task, err := client.Tasks(workspaceID).Create(ctx, "daily-task",
	    moi.WithTaskWorkflowVersionID("version-uuid"),
	    moi.WithTaskCronExpression("0 0 * * *"),
	    moi.WithTaskNotification(moi.NewHTTPNotification("https://callback.example.com").Build()),
	)

### Get

```go
// Get retrieves a task by ID.
//
// Parameters:
//   - ctx: Context for the request
//   - taskID: The task ID
//
// Returns the task or an error if not found.
//
// Example:
//
//	task, err := client.Tasks(workspaceID).Get(ctx, "task-uuid")
func (s *TaskService) Get(ctx context.Context, taskID string) (*mowl.Task, error)
```

Get retrieves a task by ID.

Parameters:
  - ctx: Context for the request
  - taskID: The task ID

Returns the task or an error if not found.

Example:

	task, err := client.Tasks(workspaceID).Get(ctx, "task-uuid")

### GetCaseStatus

```go
// GetCaseStatus retrieves the status of a workflow case for a task.
func (s *TaskService) GetCaseStatus(ctx context.Context, taskID, caseID string) (*CaseStatusResponse, error)
```

GetCaseStatus retrieves the status of a workflow case for a task.

### GetCases

```go
// GetCases retrieves all case IDs created by a task.
// A case represents a single execution instance of the workflow.
//
// Parameters:
//   - ctx: Context for the request
//   - taskID: The task ID
//
// Returns a list of case IDs or an error.
//
// Example:
//
//	caseIDs, err := client.Tasks(workspaceID).GetCases(ctx, "task-uuid")
func (s *TaskService) GetCases(ctx context.Context, taskID string) ([]string, error)
```

GetCases retrieves all case IDs created by a task.
A case represents a single execution instance of the workflow.

Parameters:
  - ctx: Context for the request
  - taskID: The task ID

Returns a list of case IDs or an error.

Example:

	caseIDs, err := client.Tasks(workspaceID).GetCases(ctx, "task-uuid")

### List

```go
// List retrieves tasks with optional filters.
// Tasks are automatically filtered by the authenticated user's ID.
//
// Parameters:
//   - ctx: Context for the request
//   - opts: Optional filter parameters (WithTaskStatus, WithTaskPeriodicOnly)
//
// Returns a list of tasks or an error.
//
// Example:
//
//	// List all active tasks
//	tasks, err := client.Tasks(workspaceID).List(ctx, moi.WithTaskStatus(0))
//
//	// List periodic tasks only
//	tasks, err := client.Tasks(workspaceID).List(ctx, moi.WithTaskPeriodicOnly(true))
func (s *TaskService) List(ctx context.Context, opts ...ListTasksOption) ([]*mowl.Task, error)
```

List retrieves tasks with optional filters.
Tasks are automatically filtered by the authenticated user's ID.

Parameters:
  - ctx: Context for the request
  - opts: Optional filter parameters (WithTaskStatus, WithTaskPeriodicOnly)

Returns a list of tasks or an error.

Example:

	// List all active tasks
	tasks, err := client.Tasks(workspaceID).List(ctx, moi.WithTaskStatus(0))

	// List periodic tasks only
	tasks, err := client.Tasks(workspaceID).List(ctx, moi.WithTaskPeriodicOnly(true))

### Trigger

```go
// Trigger triggers an immediate execution for a task and returns the created case ID.
func (s *TaskService) Trigger(ctx context.Context, taskID string) (*TriggerTaskResponse, error)
```

Trigger triggers an immediate execution for a task and returns the created case ID.

## TraceService

TraceService provides access to workflow execution traces scoped to a workspace.

### Get

```go
// Get returns the trace of a workflow case.
func (s *TraceService) Get(ctx context.Context, caseID string) (*mowlpb.TraceResponse, error)
```

Get returns the trace of a workflow case.

## UpgradeService

UpgradeService exposes system auto-upgrade diagnostics and retry APIs.

### GetTenantTask

```go
// GetTenantTask returns one tenant upgrade task by ID.
func (s *UpgradeService) GetTenantTask(ctx context.Context, taskID uint64) (*catalogpb.UpgradeTenantTask, error)
```

GetTenantTask returns one tenant upgrade task by ID.

### ListTenantTaskEvents

```go
// ListTenantTaskEvents returns the event history for one tenant upgrade task.
func (s *UpgradeService) ListTenantTaskEvents(ctx context.Context, taskID uint64, limit, offset int) (*catalogpb.ListUpgradeTenantTaskEventsResponse, error)
```

ListTenantTaskEvents returns the event history for one tenant upgrade task.

### ListTenantTasks

```go
// ListTenantTasks lists tenant upgrade tasks for diagnostics.
func (s *UpgradeService) ListTenantTasks(ctx context.Context, opts ListUpgradeTenantTasksOptions) (*catalogpb.ListUpgradeTenantTasksResponse, error)
```

ListTenantTasks lists tenant upgrade tasks for diagnostics.

### RetryTenantTask

```go
// RetryTenantTask requests retry for a failed or blocked tenant upgrade task.
func (s *UpgradeService) RetryTenantTask(ctx context.Context, taskID uint64, operatorID string) (*catalogpb.UpgradeTenantTask, error)
```

RetryTenantTask requests retry for a failed or blocked tenant upgrade task.

### RetryTenantTaskWithOptions

```go
// RetryTenantTaskWithOptions requests a tenant task retry with explicit options.
func (s *UpgradeService) RetryTenantTaskWithOptions(ctx context.Context, taskID uint64, opts RetryTenantTaskOptions) (*catalogpb.UpgradeTenantTask, error)
```

RetryTenantTaskWithOptions requests a tenant task retry with explicit options.

### Status

```go
// Status returns global auto-upgrade status and per-step tenant counters.
func (s *UpgradeService) Status(ctx context.Context) (*catalogpb.UpgradeStatus, error)
```

Status returns global auto-upgrade status and per-step tenant counters.

## UserService

UserService provides user management operations.

### Create

```go
// Create creates a new user.
// This operation requires system-level API key permissions.
//
// Parameters:
//   - ctx: Context for the request
//   - email: The user's email address (must be unique)
//   - username: The user's username (must be unique, alphanumeric and underscores only)
//   - password: The user's password
//   - opts: Optional parameters (WithCreateUserNickname, WithCreateUserPhone)
//
// Returns the created user or an error.
//
// Example:
//
//	user, err := client.Users().Create(ctx, "user@example.com", "username", "password123",
//	    moi.WithCreateUserNickname("John Doe"))
func (s *UserService) Create(ctx context.Context, email, username, password string, opts ...CreateUserOption) (*user.User, error)
```

Create creates a new user.
This operation requires system-level API key permissions.

Parameters:
  - ctx: Context for the request
  - email: The user's email address (must be unique)
  - username: The user's username (must be unique, alphanumeric and underscores only)
  - password: The user's password
  - opts: Optional parameters (WithCreateUserNickname, WithCreateUserPhone)

Returns the created user or an error.

Example:

	user, err := client.Users().Create(ctx, "user@example.com", "username", "password123",
	    moi.WithCreateUserNickname("John Doe"))

### Delete

```go
// Delete deletes a user by ID.
// This operation also removes associated API keys and role bindings.
// Returns ErrNotFound if the user does not exist.
//
// Parameters:
//   - ctx: Context for the request
//   - id: The user ID
//
// Example:
//
//	err := client.Users().Delete(ctx, "user-id-123")
func (s *UserService) Delete(ctx context.Context, id string) error
```

Delete deletes a user by ID.
This operation also removes associated API keys and role bindings.
Returns ErrNotFound if the user does not exist.

Parameters:
  - ctx: Context for the request
  - id: The user ID

Example:

	err := client.Users().Delete(ctx, "user-id-123")

### Get

```go
// Get retrieves a user by ID.
//
// Parameters:
//   - ctx: Context for the request
//   - id: The user ID
//
// Returns the user or ErrNotFound if not found.
//
// Example:
//
//	user, err := client.Users().Get(ctx, "user-id-123")
func (s *UserService) Get(ctx context.Context, id string) (*user.User, error)
```

Get retrieves a user by ID.

Parameters:
  - ctx: Context for the request
  - id: The user ID

Returns the user or ErrNotFound if not found.

Example:

	user, err := client.Users().Get(ctx, "user-id-123")

### GetByEmail

```go
// GetByEmail retrieves a user by email address.
//
// Parameters:
//   - ctx: Context for the request
//   - email: The user's email address
//
// Returns the user or ErrNotFound if not found.
//
// Example:
//
//	user, err := client.Users().GetByEmail(ctx, "user@example.com")
func (s *UserService) GetByEmail(ctx context.Context, email string) (*user.User, error)
```

GetByEmail retrieves a user by email address.

Parameters:
  - ctx: Context for the request
  - email: The user's email address

Returns the user or ErrNotFound if not found.

Example:

	user, err := client.Users().GetByEmail(ctx, "user@example.com")

### GetByPhone

```go
// GetByPhone retrieves an existing user by exact phone number.
func (s *UserService) GetByPhone(ctx context.Context, phone string) (*user.User, error)
```

GetByPhone retrieves an existing user by exact phone number.

### List

```go
// List retrieves all users with optional pagination.
//
// Parameters:
//   - ctx: Context for the request
//   - opts: Optional pagination parameters (WithPageSize, WithPageToken)
//
// Returns a list of users or an error.
//
// Example:
//
//	users, err := client.Users().List(ctx,
//	    moi.WithPageSize(10),
//	    moi.WithPageToken("next-page-token"),
//	)
func (s *UserService) List(ctx context.Context, opts ...ListOption) ([]*user.User, error)
```

List retrieves all users with optional pagination.

Parameters:
  - ctx: Context for the request
  - opts: Optional pagination parameters (WithPageSize, WithPageToken)

Returns a list of users or an error.

Example:

	users, err := client.Users().List(ctx,
	    moi.WithPageSize(10),
	    moi.WithPageToken("next-page-token"),
	)

### Update

```go
// Update updates an existing user.
//
// Parameters:
//   - ctx: Context for the request
//   - id: The user ID
//   - opts: Fields to update (WithUserNickname, WithUserPhone, WithUserStatus)
//
// Returns the updated user or an error.
//
// Example:
//
//	user, err := client.Users().Update(ctx, "user-id-123",
//	    moi.WithUserNickname("New Name"),
//	    moi.WithUserPhone("+1234567890"),
//	)
func (s *UserService) Update(ctx context.Context, id string, opts ...UpdateUserOption) (*user.User, error)
```

Update updates an existing user.

Parameters:
  - ctx: Context for the request
  - id: The user ID
  - opts: Fields to update (WithUserNickname, WithUserPhone, WithUserStatus)

Returns the updated user or an error.

Example:

	user, err := client.Users().Update(ctx, "user-id-123",
	    moi.WithUserNickname("New Name"),
	    moi.WithUserPhone("+1234567890"),
	)

## VolumeFileService

VolumeFileService provides volume file management operations.
It handles adding, moving, removing, and listing files in volumes.
Requirements: 6.1, 7.1, 8.1

### AddFiles

```go
// AddFiles adds files to a volume.
// The files must already exist in the workspace.
// This operation increases the reference count of each file.
// Requirements: 6.1
func (s *VolumeFileService) AddFiles(ctx context.Context, workspaceID string, volumeID int64, fileIDs []string, opts ...AddFilesOption) error
```

AddFiles adds files to a volume.
The files must already exist in the workspace.
This operation increases the reference count of each file.
Requirements: 6.1

### AddFilesWithItems

```go
// AddFilesWithItems adds files to a volume with optional file_name and file_path overrides.
func (s *VolumeFileService) AddFilesWithItems(ctx context.Context, workspaceID string, volumeID int64, items []AddFileItem, opts ...AddFilesOption) error
```

AddFilesWithItems adds files to a volume with optional file_name and file_path overrides.

### AttachBuiltinFiles

```go
// AttachBuiltinFiles atomically creates tenant-local metadata and Volume
// references for immutable system-owned objects. It never copies object bytes.
func (s *VolumeFileService) AttachBuiltinFiles(ctx context.Context, workspaceID string, volumeID int64, items []BuiltinFileAttachment) ([]string, error)
```

AttachBuiltinFiles atomically creates tenant-local metadata and Volume
references for immutable system-owned objects. It never copies object bytes.

### ListFiles

```go
// ListFiles lists files in a volume with optional pagination and filtering.
// Requirements: 6.1
func (s *VolumeFileService) ListFiles(ctx context.Context, workspaceID string, volumeID int64, opts ...VolumeFileListOption) (*ListVolumeFilesResponse, error)
```

ListFiles lists files in a volume with optional pagination and filtering.
Requirements: 6.1

### ListFilesDetail

```go
// ListFilesDetail lists files in a volume with full file metadata (JOIN with file table).
// Returns volume file associations along with original_name, md5, size, and ref_count from the file table.
func (s *VolumeFileService) ListFilesDetail(ctx context.Context, workspaceID string, volumeID int64, opts ...VolumeFileListOption) (*ListVolumeFilesDetailResponse, error)
```

ListFilesDetail lists files in a volume with full file metadata (JOIN with file table).
Returns volume file associations along with original_name, md5, size, and ref_count from the file table.

### MoveFiles

```go
// MoveFiles moves files from one volume to another.
// This operation does not change the reference count of the files.
// Requirements: 7.1
func (s *VolumeFileService) MoveFiles(ctx context.Context, workspaceID string, sourceVolumeID int64, targetVolumeID int64, fileIDs []string) error
```

MoveFiles moves files from one volume to another.
This operation does not change the reference count of the files.
Requirements: 7.1

### RemoveFiles

```go
// RemoveFiles removes files from a volume.
// This operation decreases the reference count of each file.
// The actual file deletion is handled by the garbage collector when ref_count reaches 0.
// Requirements: 8.1
func (s *VolumeFileService) RemoveFiles(ctx context.Context, workspaceID string, volumeID int64, fileIDs []string) error
```

RemoveFiles removes files from a volume.
This operation decreases the reference count of each file.
The actual file deletion is handled by the garbage collector when ref_count reaches 0.
Requirements: 8.1

### ResolveDataAssetRoot

```go
// ResolveDataAssetRoot retrieves the canonical root Volume for one DataAsset.
// The system metadata result is an authorization input and never an allow.
func (s *VolumeFileService) ResolveDataAssetRoot(ctx context.Context, workspaceID, assetID string) (string, error)
```

ResolveDataAssetRoot retrieves the canonical root Volume for one DataAsset.
The system metadata result is an authorization input and never an allow.

### ResolveRoots

```go
// ResolveRoots retrieves trusted canonical root Volume IDs for Catalog files.
// The client must be initialized with the system API key; callers must perform
// their own IAM authorization using the returned resource IDs.
func (s *VolumeFileService) ResolveRoots(ctx context.Context, workspaceID string, fileIDs []string) ([]string, error)
```

ResolveRoots retrieves trusted canonical root Volume IDs for Catalog files.
The client must be initialized with the system API key; callers must perform
their own IAM authorization using the returned resource IDs.

### TriggerFiles

```go
// TriggerFiles creates volume trigger deliveries for existing files in a volume.
func (s *VolumeFileService) TriggerFiles(ctx context.Context, workspaceID string, volumeID int64, fileIDs []string) (*TriggerFilesResponse, error)
```

TriggerFiles creates volume trigger deliveries for existing files in a volume.

## VolumeService

VolumeService provides volume management operations.

### Create

```go
// Create creates a new volume under the specified database. The name must
// follow the Catalog resource naming contract documented in
// docs/guide/SDK_GUIDE.md. Child volumes use the same contract.
func (s *VolumeService) Create(ctx context.Context, workspaceID string, databaseID int64, name string, opts ...CreateVolumeOption) (*catalog.Volume, error)
```

Create creates a new volume under the specified database. The name must
follow the Catalog resource naming contract documented in
docs/guide/SDK_GUIDE.md. Child volumes use the same contract.

### Delete

```go
// Delete deletes a volume by ID.
func (s *VolumeService) Delete(ctx context.Context, workspaceID string, volumeID int64) error
```

Delete deletes a volume by ID.

### DeleteFile

```go
// DeleteFile deletes a file by ID.
func (s *VolumeService) DeleteFile(ctx context.Context, workspaceID string, fileID string) error
```

DeleteFile deletes a file by ID.

### DeleteMultiple

```go
// DeleteMultiple deletes multiple volumes by IDs.
func (s *VolumeService) DeleteMultiple(ctx context.Context, workspaceID string, volumeIDs []int64, opts ...BatchOption) (*BatchResult, error)
```

DeleteMultiple deletes multiple volumes by IDs.

### Download

```go
// Download downloads a file by ID.
func (s *VolumeService) Download(ctx context.Context, workspaceID string, fileID string) (io.ReadCloser, error)
```

Download downloads a file by ID.

### Get

```go
// Get retrieves a volume by ID.
func (s *VolumeService) Get(ctx context.Context, workspaceID string, volumeID int64) (*catalog.Volume, error)
```

Get retrieves a volume by ID.

### GetChildren

```go
// GetChildren retrieves the child volumes of a specified volume.
// Returns a paginated list of child volumes.
// Requirements: 9.3
func (s *VolumeService) GetChildren(ctx context.Context, workspaceID string, volumeID int64, opts ...ListOption) (*catalog.GetVolumeChildrenResponse, error)
```

GetChildren retrieves the child volumes of a specified volume.
Returns a paginated list of child volumes.
Requirements: 9.3

### GetPath

```go
// GetPath retrieves the complete path from root to the specified volume.
// Returns a list of volumes representing the path from root to the target volume.
// Requirements: 9.4
func (s *VolumeService) GetPath(ctx context.Context, workspaceID string, volumeID int64) (*catalog.GetVolumePathResponse, error)
```

GetPath retrieves the complete path from root to the specified volume.
Returns a list of volumes representing the path from root to the target volume.
Requirements: 9.4

### List

```go
// List retrieves all volumes under the specified database with optional pagination.
func (s *VolumeService) List(ctx context.Context, workspaceID string, databaseID int64, opts ...ListOption) (*catalog.ListVolumesResponse, error)
```

List retrieves all volumes under the specified database with optional pagination.

### ListFiles

```go
// ListFiles lists files in the specified volume with optional filtering and pagination.
func (s *VolumeService) ListFiles(ctx context.Context, workspaceID string, volumeID int64, opts ...ListFilesOption) ([]*catalog.File, error)
```

ListFiles lists files in the specified volume with optional filtering and pagination.

### ListIter

```go
// ListIter returns an iterator for listing volumes under a database.
func (s *VolumeService) ListIter(ctx context.Context, workspaceID string, databaseID int64, opts ...ListOption) *VolumeIterator
```

ListIter returns an iterator for listing volumes under a database.

### ResolveRoot

```go
// ResolveRoot retrieves the canonical root Volume through the Catalog system
// metadata boundary. The client must be initialized with the system API key.
func (s *VolumeService) ResolveRoot(ctx context.Context, workspaceID string, volumeID int64) (*catalog.Volume, error)
```

ResolveRoot retrieves the canonical root Volume through the Catalog system
metadata boundary. The client must be initialized with the system API key.

### Update

```go
// Update updates an existing volume.
func (s *VolumeService) Update(ctx context.Context, workspaceID string, volumeID int64, opts ...UpdateVolumeOption) (*catalog.Volume, error)
```

Update updates an existing volume.

### Upload

```go
// Upload uploads a file to the specified volume in two steps:
// 1. Multipart upload to /files to get file_id.
// 2. Add file to volume via POST /volumes/:id/files with {file_ids}.
func (s *VolumeService) Upload(ctx context.Context, workspaceID string, volumeID int64, filename string, reader io.Reader, opts ...UploadOption) (*catalog.File, error)
```

Upload uploads a file to the specified volume in two steps:
1. Multipart upload to /files to get file_id.
2. Add file to volume via POST /volumes/:id/files with {file_ids}.

## WorkItemService

WorkItemService provides workitem metadata query operations.
WorkItems represent executable nodes in a workflow.
WorkItemService is always scoped to a workspace via client.WorkItems(workspaceID).

Validates: Requirements 6.18

### GetCatalog

```go
// GetCatalog retrieves a single workitem catalog entry by node_id.
func (s *WorkItemService) GetCatalog(ctx context.Context, nodeID string) (map[string]interface{}, error)
```

GetCatalog retrieves a single workitem catalog entry by node_id.

### List

```go
// List retrieves all available workitem metadata.
// Returns workitems visible to the authenticated user based on isolation level:
//   - public: visible to all users
//   - shared: visible to specific users
//   - private: visible only to the owner
//
// Parameters:
//   - ctx: Context for the request
//
// Returns a map of node_id -> WorkItemMetadataList or an error.
//
// Example:
//
//	workitems, err := client.WorkItems(workspaceID).List(ctx)
func (s *WorkItemService) List(ctx context.Context) (map[string]*mowl.WorkItemMetadataList, error)
```

List retrieves all available workitem metadata.
Returns workitems visible to the authenticated user based on isolation level:
  - public: visible to all users
  - shared: visible to specific users
  - private: visible only to the owner

Parameters:
  - ctx: Context for the request

Returns a map of node_id -> WorkItemMetadataList or an error.

Example:

	workitems, err := client.WorkItems(workspaceID).List(ctx)

### ListCatalog

```go
// ListCatalog retrieves a frontend-friendly catalog of available workitems.
func (s *WorkItemService) ListCatalog(ctx context.Context) (*CatalogListResponse, error)
```

ListCatalog retrieves a frontend-friendly catalog of available workitems.

### ListUIMetadata

```go
// ListUIMetadata retrieves all visible workitems and returns a normalized UI metadata view.
// Compared with List, this method parses and validates:
//   - WorkItemMetadata
//   - input_ui_schema inside WorkItemMetadata
//   - output_ui_schema inside WorkItemMetadata
//
// It returns an error when any returned workitem has missing/invalid normalized UI metadata.
func (s *WorkItemService) ListUIMetadata(ctx context.Context) (map[string]*WorkItemUIMetadataList, error)
```

ListUIMetadata retrieves all visible workitems and returns a normalized UI metadata view.
Compared with List, this method parses and validates:
  - WorkItemMetadata
  - input_ui_schema inside WorkItemMetadata
  - output_ui_schema inside WorkItemMetadata

It returns an error when any returned workitem has missing/invalid normalized UI metadata.

### ListWithLocale

```go
// ListWithLocale retrieves all available workitem metadata with i18n text resolved
// for the specified language. If lang is LANGUAGE_UNSPECIFIED, behaves like List.
func (s *WorkItemService) ListWithLocale(ctx context.Context, lang commonpb.Language) (map[string]*mowl.WorkItemMetadataList, error)
```

ListWithLocale retrieves all available workitem metadata with i18n text resolved
for the specified language. If lang is LANGUAGE_UNSPECIFIED, behaves like List.

## WorkerClient

WorkerClient is the worker-side client that connects to the Mowl Engine
via the Catalog Service gRPC proxy. It manages work item registration,
WorkerSession bidirectional streaming, and notification handling.

### AddNodeNotifyHandler

```go
// AddNodeNotifyHandler registers a node notification handler with optional state and node ID filtering.
// If handler is nil, notifications will only be sent to the channel (for WaitForNodeNotification).
func (w *WorkerClient) AddNodeNotifyHandler(name string, handler NodeNotifyHandler, opts ...NotificationOption) error
```

AddNodeNotifyHandler registers a node notification handler with optional state and node ID filtering.
If handler is nil, notifications will only be sent to the channel (for WaitForNodeNotification).

### AddWorkflowNotifyHandler

```go
// AddWorkflowNotifyHandler registers a workflow notification handler with optional state filtering.
// If handler is nil, notifications will only be sent to the channel (for WaitForWorkflowNotification).
func (w *WorkerClient) AddWorkflowNotifyHandler(name string, handler WorkflowNotifyHandler, opts ...NotificationOption) error
```

AddWorkflowNotifyHandler registers a workflow notification handler with optional state filtering.
If handler is nil, notifications will only be sent to the channel (for WaitForWorkflowNotification).

### BufferNodeNotification

```go
// BufferNodeNotification stores node notifications for short-term catch-up.
func (w *WorkerClient) BufferNodeNotification(notif *mowl.NodeNotification)
```

BufferNodeNotification stores node notifications for short-term catch-up.

### BufferWorkflowNotification

```go
// BufferWorkflowNotification stores workflow notifications for short-term catch-up.
// This mitigates races where waiting starts just after callback arrival.
func (w *WorkerClient) BufferWorkflowNotification(notif *mowl.WorkflowNotification)
```

BufferWorkflowNotification stores workflow notifications for short-term catch-up.
This mitigates races where waiting starts just after callback arrival.

### CancelWorkflow

```go
// CancelWorkflow cancels a running workflow case.
func (w *WorkerClient) CancelWorkflow(caseID string) error
```

CancelWorkflow cancels a running workflow case.

### CheckWorkflow

```go
// CheckWorkflow queries the current execution status of a workflow case.
// Returns data, error text, status, and any error.
func (w *WorkerClient) CheckWorkflow(caseID string) (string, string, Status, error)
```

CheckWorkflow queries the current execution status of a workflow case.
Returns data, error text, status, and any error.

### Connect

```go
// Connect establishes a gRPC connection to the Catalog Service proxy,
// registers the worker and work items, and starts the WorkerSession
// bidirectional stream.
// Requires WithWorkerID to be set when creating the client; otherwise returns an error.
func (w *WorkerClient) Connect(ctx context.Context) error
```

Connect establishes a gRPC connection to the Catalog Service proxy,
registers the worker and work items, and starts the WorkerSession
bidirectional stream.
Requires WithWorkerID to be set when creating the client; otherwise returns an error.

### ConnectInvokeOnly

```go
// ConnectInvokeOnly establishes a gRPC connection for one-shot Mowl RPCs only.
// Unlike Connect, it does NOT require WithWorkerID, register the worker, start a
// WorkerSession, or heartbeat. This is useful for:
//   - CLI tools calling InvokeDynamicServiceSync/Stream
//   - system nodes such as moi:workflow.trigger that only create tasks
func (w *WorkerClient) ConnectInvokeOnly(_ context.Context) error
```

ConnectInvokeOnly establishes a gRPC connection for one-shot Mowl RPCs only.
Unlike Connect, it does NOT require WithWorkerID, register the worker, start a
WorkerSession, or heartbeat. This is useful for:
  - CLI tools calling InvokeDynamicServiceSync/Stream
  - system nodes such as moi:workflow.trigger that only create tasks

### ConnectWithRetry

```go
// ConnectWithRetry establishes the worker connection and retries transient
// startup registration failures until ctx is canceled.
func (w *WorkerClient) ConnectWithRetry(ctx context.Context) error
```

ConnectWithRetry establishes the worker connection and retries transient
startup registration failures until ctx is canceled.

### Disconnect

```go
// Disconnect gracefully shuts down the WorkerSession stream and gRPC connection.
func (w *WorkerClient) Disconnect() error
```

Disconnect gracefully shuts down the WorkerSession stream and gRPC connection.

### ExecuteByWorkflowName

```go
// ExecuteByWorkflowName executes a workflow using the latest published version of a named workflow.
// It resolves the workflow name to a definition, fetches the latest published version,
// then delegates to ExecuteByWorkflowVersion.
// Parameters:
//   - name: task name for this execution
//   - workflowName: workflow definition name (used to resolve the latest published version)
//
// Note: user is derived from API key on the server side.
// For timeout/cancellation control, use ExecuteByWorkflowNameWithContext.
func (w *WorkerClient) ExecuteByWorkflowName(name, workflowName string, opts ...WorkerTaskOption) (*mowl.Task, error)
```

ExecuteByWorkflowName executes a workflow using the latest published version of a named workflow.
It resolves the workflow name to a definition, fetches the latest published version,
then delegates to ExecuteByWorkflowVersion.
Parameters:
  - name: task name for this execution
  - workflowName: workflow definition name (used to resolve the latest published version)

Note: user is derived from API key on the server side.
For timeout/cancellation control, use ExecuteByWorkflowNameWithContext.

### ExecuteByWorkflowNameWithContext

```go
// ExecuteByWorkflowNameWithContext is like ExecuteByWorkflowName but accepts ctx for timeout and cancellation.
func (w *WorkerClient) ExecuteByWorkflowNameWithContext(ctx context.Context, name, workflowName string, opts ...WorkerTaskOption) (*mowl.Task, error)
```

ExecuteByWorkflowNameWithContext is like ExecuteByWorkflowName but accepts ctx for timeout and cancellation.

### ExecuteByWorkflowVersion

```go
// ExecuteByBuilder executes a workflow built from the DSL builder.
// The notification configuration is built from registered handlers unless set via WithNotificationConfig.
//
// Example:
//
//	builder := dsl.Workflow("my-net", "root").Chain(
//	    dsl.WorkItem("step1", "my-workitem"),
//	    dsl.JQ("cond", ".value > 0"),
//	)
//	result, err := worker.ExecuteByBuilder("task-name", builder, moi.WithData(`{"value": 1}`))
//
// ExecuteByWorkflowVersion executes a workflow referencing a specific workflow version.
// Parameters:
//   - name: task name for this execution (used to identify the task in listings and notifications)
//   - versionID: workflow version ID (the published or draft version to run)
//
// Note: user is derived from API key on the server side.
// For timeout/cancellation control, use ExecuteByWorkflowVersionWithContext.
func (w *WorkerClient) ExecuteByWorkflowVersion(name, versionID string, opts ...WorkerTaskOption) (*mowl.Task, error)
```

ExecuteByBuilder executes a workflow built from the DSL builder.
The notification configuration is built from registered handlers unless set via WithNotificationConfig.

Example:

	builder := dsl.Workflow("my-net", "root").Chain(
	    dsl.WorkItem("step1", "my-workitem"),
	    dsl.JQ("cond", ".value > 0"),
	)
	result, err := worker.ExecuteByBuilder("task-name", builder, moi.WithData(`{"value": 1}`))

ExecuteByWorkflowVersion executes a workflow referencing a specific workflow version.
Parameters:
  - name: task name for this execution (used to identify the task in listings and notifications)
  - versionID: workflow version ID (the published or draft version to run)

Note: user is derived from API key on the server side.
For timeout/cancellation control, use ExecuteByWorkflowVersionWithContext.

### ExecuteByWorkflowVersionWithContext

```go
// ExecuteByWorkflowVersionWithContext is like ExecuteByWorkflowVersion but accepts ctx for timeout and cancellation.
func (w *WorkerClient) ExecuteByWorkflowVersionWithContext(ctx context.Context, name, versionID string, opts ...WorkerTaskOption) (*mowl.Task, error)
```

ExecuteByWorkflowVersionWithContext is like ExecuteByWorkflowVersion but accepts ctx for timeout and cancellation.

### GetNodeHandlers

```go
// GetNodeHandlers implements notification.HandlerAccessor.
func (w *WorkerClient) GetNodeHandlers() map[string]*notification.NodeEntry
```

GetNodeHandlers implements notification.HandlerAccessor.

### GetWorkflowHandlers

```go
// GetWorkflowHandlers implements notification.HandlerAccessor.
func (w *WorkerClient) GetWorkflowHandlers() map[string]*notification.WorkflowEntry
```

GetWorkflowHandlers implements notification.HandlerAccessor.

### InvokeDynamicServiceStream

```go
// InvokeDynamicServiceStream 调用 stream 模式的动态服务，返回 StreamResult。
// 调用者通过 StreamResult.Recv() 逐条读取事件，无需额外 goroutine。
// Recv() 不会返回 io.EOF，流结束时通过 StreamEvent.Done == true 告知。
//
// Example:
//
//	sr, err := worker.InvokeDynamicServiceStream(ctx, "image-processor", `{"url": "https://example.com"}`)
//	for {
//	    event, err := sr.Recv()
//	    if err != nil { return err }
//	    if event.Done { break }
//	    // process event.Data
//	}
func (w *WorkerClient) InvokeDynamicServiceStream(ctx context.Context, workflowName string, input string, opts ...InvokeOption) (*StreamResult, error)
```

InvokeDynamicServiceStream 调用 stream 模式的动态服务，返回 StreamResult。
调用者通过 StreamResult.Recv() 逐条读取事件，无需额外 goroutine。
Recv() 不会返回 io.EOF，流结束时通过 StreamEvent.Done == true 告知。

Example:

	sr, err := worker.InvokeDynamicServiceStream(ctx, "image-processor", `{"url": "https://example.com"}`)
	for {
	    event, err := sr.Recv()
	    if err != nil { return err }
	    if event.Done { break }
	    // process event.Data
	}

### InvokeDynamicServiceSync

```go
// InvokeDynamicServiceSync 调用 oneshot 模式的动态服务，阻塞等待结果返回。
//
// Example:
//
//	result, err := worker.InvokeDynamicServiceSync(ctx, serviceID, `{"url": "https://example.com"}`)
func (w *WorkerClient) InvokeDynamicServiceSync(ctx context.Context, workflowName string, input string, opts ...InvokeOption) (*InvokeResult, error)
```

InvokeDynamicServiceSync 调用 oneshot 模式的动态服务，阻塞等待结果返回。

Example:

	result, err := worker.InvokeDynamicServiceSync(ctx, serviceID, `{"url": "https://example.com"}`)

### RegisterCallbackHandler

```go
// RegisterCallbackHandler registers a callback handler for a specific message type.
func (w *WorkerClient) RegisterCallbackHandler(message string, handler CallbackHandlerFunc) error
```

RegisterCallbackHandler registers a callback handler for a specific message type.

### RegisterDualModeWorkItem

```go
// RegisterDualModeWorkItem registers a work item that supports both one-shot and streaming modes.
// The engine will use the stream handler when the workitem metadata declares stream=true;
// otherwise the one-shot handler is used. This allows a single workitem to dynamically support
// both execution modes based on how it is invoked.
// Must be called before Connect.
func (w *WorkerClient) RegisterDualModeWorkItem(name string, md *mowl.WorkItemMetadata, fn ExternalWorkItemFunc, sfn StreamWorkItemFunc) error
```

RegisterDualModeWorkItem registers a work item that supports both one-shot and streaming modes.
The engine will use the stream handler when the workitem metadata declares stream=true;
otherwise the one-shot handler is used. This allows a single workitem to dynamically support
both execution modes based on how it is invoked.
Must be called before Connect.

### RegisterDualModeWorkItemWithOptions

```go
// RegisterDualModeWorkItemWithOptions registers a work item with both one-shot and stream handlers,
// while applying schema options to the shared metadata.
func (w *WorkerClient) RegisterDualModeWorkItemWithOptions(name string, md *mowl.WorkItemMetadata, fn ExternalWorkItemFunc, sfn StreamWorkItemFunc, opts ...WorkItemOption) error
```

RegisterDualModeWorkItemWithOptions registers a work item with both one-shot and stream handlers,
while applying schema options to the shared metadata.

### RegisterStreamWorkItem

```go
// RegisterStreamWorkItem registers a stream work item handler. The handler receives a StreamWriter
// and should call Emit(data, vars) multiple times then End(status). The stream capability is declared
// in workitem metadata (stream=true), so the workflow node does not need __stream_mode in VarMap.
// Optional protection: set md.StreamConfig (MaxDurationSeconds, IdleTimeoutSeconds, MaxEvents,
// EnableDownstreamFeedback) so the engine can enforce timeouts and limits (see design 4.7.8).
// Must be called before Connect.
func (w *WorkerClient) RegisterStreamWorkItem(name string, md *mowl.WorkItemMetadata, fn StreamWorkItemFunc) error
```

RegisterStreamWorkItem registers a stream work item handler. The handler receives a StreamWriter
and should call Emit(data, vars) multiple times then End(status). The stream capability is declared
in workitem metadata (stream=true), so the workflow node does not need __stream_mode in VarMap.
Optional protection: set md.StreamConfig (MaxDurationSeconds, IdleTimeoutSeconds, MaxEvents,
EnableDownstreamFeedback) so the engine can enforce timeouts and limits (see design 4.7.8).
Must be called before Connect.

### RegisterStreamWorkItemWithOptions

```go
// RegisterStreamWorkItemWithOptions registers a stream work item with options.
// This is an extended version of RegisterStreamWorkItem that supports schema configuration.
func (w *WorkerClient) RegisterStreamWorkItemWithOptions(name string, md *mowl.WorkItemMetadata, fn StreamWorkItemFunc, opts ...WorkItemOption) error
```

RegisterStreamWorkItemWithOptions registers a stream work item with options.
This is an extended version of RegisterStreamWorkItem that supports schema configuration.

### RegisterWorkItem

```go
// RegisterWorkItem registers a work item handler in the local registry.
// Must be called before Connect. The name is used as the node ID for dispatch.
func (w *WorkerClient) RegisterWorkItem(name string, md *mowl.WorkItemMetadata, fn ExternalWorkItemFunc) error
```

RegisterWorkItem registers a work item handler in the local registry.
Must be called before Connect. The name is used as the node ID for dispatch.

### RegisterWorkItemDispatcher

```go
// RegisterWorkItemDispatcher registers a WorkItem that handles a family of node IDs.
// The worker advertises name to Mowl (e.g. "moi:custom.operator"), while execution
// requests whose resolved node ID starts with dispatchPrefix are handled by the
// same local entry. Must be called before Connect. Matches Python
// register_work_item_dispatcher_with_options.
func (w *WorkerClient) RegisterWorkItemDispatcher(name, dispatchPrefix string, md *mowl.WorkItemMetadata, fn ExternalWorkItemFunc) error
```

RegisterWorkItemDispatcher registers a WorkItem that handles a family of node IDs.
The worker advertises name to Mowl (e.g. "moi:custom.operator"), while execution
requests whose resolved node ID starts with dispatchPrefix are handled by the
same local entry. Must be called before Connect. Matches Python
register_work_item_dispatcher_with_options.

### RegisterWorkItemWithOptions

```go
// RegisterWorkItemWithOptions registers a work item with options.
// This is an extended version of RegisterWorkItem that supports schema configuration.
//
// Example:
//
//	inputBuilder := NewSchema().
//	    Property("url", StringSchema().MinLength(1)).
//	    Required("url")
//	outputBuilder := NewSchema().
//	    Property("status", IntegerSchema()).
//	    Required("status")
//
//	err := worker.RegisterWorkItemWithOptions("http_request", metadata, handler,
//	    WithInputSchemaBuilder(inputBuilder),
//	    WithOutputSchemaBuilder(outputBuilder))
func (w *WorkerClient) RegisterWorkItemWithOptions(name string, md *mowl.WorkItemMetadata, fn ExternalWorkItemFunc, opts ...WorkItemOption) error
```

RegisterWorkItemWithOptions registers a work item with options.
This is an extended version of RegisterWorkItem that supports schema configuration.

Example:

	inputBuilder := NewSchema().
	    Property("url", StringSchema().MinLength(1)).
	    Required("url")
	outputBuilder := NewSchema().
	    Property("status", IntegerSchema()).
	    Required("status")

	err := worker.RegisterWorkItemWithOptions("http_request", metadata, handler,
	    WithInputSchemaBuilder(inputBuilder),
	    WithOutputSchemaBuilder(outputBuilder))

### RemoveNodeNotifyHandler

```go
// RemoveNodeNotifyHandler removes a node notification handler by name.
func (w *WorkerClient) RemoveNodeNotifyHandler(name string) error
```

RemoveNodeNotifyHandler removes a node notification handler by name.

### RemoveShared

```go
// RemoveShared removes a sharing relationship for a workitem.
func (w *WorkerClient) RemoveShared(nodeID, userID string) error
```

RemoveShared removes a sharing relationship for a workitem.

### RemoveWorkflowNotifyHandler

```go
// RemoveWorkflowNotifyHandler removes a workflow notification handler by name.
func (w *WorkerClient) RemoveWorkflowNotifyHandler(name string) error
```

RemoveWorkflowNotifyHandler removes a workflow notification handler by name.

### ShareTo

```go
// ShareTo shares a workitem with another user.
// The shared user will be able to see and execute the workitem.
func (w *WorkerClient) ShareTo(nodeID, userID string) error
```

ShareTo shares a workitem with another user.
The shared user will be able to see and execute the workitem.

### SharedList

```go
// SharedList returns the list of user IDs that a workitem is shared with.
func (w *WorkerClient) SharedList(nodeID string) ([]string, error)
```

SharedList returns the list of user IDs that a workitem is shared with.

### WaitForNodeNotification

```go
// WaitForNodeNotification blocks until a node notification matching the options is received or the context is cancelled.
func (w *WorkerClient) WaitForNodeNotification(ctx context.Context, opts ...NotificationOption) (*mowl.NodeNotification, error)
```

WaitForNodeNotification blocks until a node notification matching the options is received or the context is cancelled.

### WaitForWorkflowNotification

```go
// WaitForWorkflowNotification blocks until a workflow notification matching the options is received or the context is cancelled.
func (w *WorkerClient) WaitForWorkflowNotification(ctx context.Context, opts ...NotificationOption) (*mowl.WorkflowNotification, error)
```

WaitForWorkflowNotification blocks until a workflow notification matching the options is received or the context is cancelled.

### WaitWorkflow

```go
// WaitWorkflow blocks until a workflow case completes (or fails/cancels).
// Returns data, error text, status, and any error.
func (w *WorkerClient) WaitWorkflow(caseID string) (string, string, Status, error)
```

WaitWorkflow blocks until a workflow case completes (or fails/cancels).
Returns data, error text, status, and any error.

### WorkerBinding

```go
// WorkerBinding returns the exact MOWL worker identity and currently attached
// stream generation. A false result means no dispatchable generation exists.
func (w *WorkerClient) WorkerBinding() (workerID, generation string, ok bool)
```

WorkerBinding returns the exact MOWL worker identity and currently attached
stream generation. A false result means no dispatchable generation exists.

## WorkflowAppService

WorkflowAppService provides product-level workflow app and execution APIs.

### BatchRetryExecutions

```go
// BatchRetryExecutions creates one Volume dispatch job for selected or all
// failed file executions from a source dispatch job.
func (s *WorkflowAppService) BatchRetryExecutions(ctx context.Context, workflowID string, req *BatchRetryExecutionsRequest) (*BatchRetryExecutionsResponse, error)
```

BatchRetryExecutions creates one Volume dispatch job for selected or all
failed file executions from a source dispatch job.

### CancelExecution

```go
// CancelExecution cancels a running workflow execution.
func (s *WorkflowAppService) CancelExecution(ctx context.Context, workflowID, executionID string) (*WorkflowExecutionEnvelope, error)
```

CancelExecution cancels a running workflow execution.

### CreateExecution

```go
// CreateExecution creates and starts a workflow execution.
func (s *WorkflowAppService) CreateExecution(ctx context.Context, workflowID string, req *WorkflowExecutionCreateRequest) (*WorkflowExecutionEnvelope, error)
```

CreateExecution creates and starts a workflow execution.

### Delete

```go
// Delete deletes a workflow app only when it has no non-terminal executions.
func (s *WorkflowAppService) Delete(ctx context.Context, workflowID string) (*WorkflowAppDeleteResponse, error)
```

Delete deletes a workflow app only when it has no non-terminal executions.

### DeleteExecution

```go
// DeleteExecution deletes a workflow execution record.
func (s *WorkflowAppService) DeleteExecution(ctx context.Context, workflowID, executionID string) (*WorkflowAppDeleteResponse, error)
```

DeleteExecution deletes a workflow execution record.

### EnsureMemoryGovernanceSystemWorkflow

```go
// EnsureMemoryGovernanceSystemWorkflow provisions the hidden platform Memory
// Workflow only for an exact connector authorization already verified by the
// Backend PEP. Direct PAT callers cannot manufacture that signed action fact.
func (s *WorkflowAppService) EnsureMemoryGovernanceSystemWorkflow(ctx context.Context, connectorID string) (*SystemWorkflowRef, error)
```

EnsureMemoryGovernanceSystemWorkflow provisions the hidden platform Memory
Workflow only for an exact connector authorization already verified by the
Backend PEP. Direct PAT callers cannot manufacture that signed action fact.

### EnsureSystemWorkflow

```go
// EnsureSystemWorkflow provisions or refreshes a hidden system workflow app.
func (s *WorkflowAppService) EnsureSystemWorkflow(ctx context.Context, kind string) (*SystemWorkflowRef, error)
```

EnsureSystemWorkflow provisions or refreshes a hidden system workflow app.

### EnsureSystemWorkflowExecution

```go
// EnsureSystemWorkflow starts one execution for an idempotency key or returns
// the existing execution created by an earlier, possibly response-lost call.
func (s *WorkflowAppService) EnsureSystemWorkflowExecution(ctx context.Context, kind string, req *SystemWorkflowExecutionRequest) (*SystemWorkflowExecutionResponse, error)
```

EnsureSystemWorkflow starts one execution for an idempotency key or returns
the existing execution created by an earlier, possibly response-lost call.

### Get

```go
// Get returns one workflow app by ID.
func (s *WorkflowAppService) Get(ctx context.Context, workflowID string) (*WorkflowAppEnvelope, error)
```

Get returns one workflow app by ID.

### GetExecution

```go
// GetExecution returns one workflow execution by ID.
func (s *WorkflowAppService) GetExecution(ctx context.Context, workflowID, executionID string) (*WorkflowExecutionEnvelope, error)
```

GetExecution returns one workflow execution by ID.

### GetExecutionResult

```go
// GetExecutionResult refreshes runtime result state for one workflow execution.
func (s *WorkflowAppService) GetExecutionResult(ctx context.Context, workflowID, executionID string) (*WorkflowExecutionEnvelope, error)
```

GetExecutionResult refreshes runtime result state for one workflow execution.

### List

```go
// List returns workflow apps in the workspace.
func (s *WorkflowAppService) List(ctx context.Context, req *WorkflowAppListRequest) (*WorkflowAppListResponse, error)
```

List returns workflow apps in the workspace.

### ListExecutions

```go
// ListExecutions returns executions for one workflow app.
func (s *WorkflowAppService) ListExecutions(ctx context.Context, workflowID string, offset, limit int, status string) (*WorkflowExecutionListResponse, error)
```

ListExecutions returns executions for one workflow app.

### ListExecutionsWithQuery

```go
// ListExecutionsWithQuery returns workflow executions with optional filters.
func (s *WorkflowAppService) ListExecutionsWithQuery(ctx context.Context, req WorkflowExecutionListRequest) (*WorkflowExecutionListResponse, error)
```

ListExecutionsWithQuery returns workflow executions with optional filters.

### ListFileExecutions

```go
// ListFileExecutions returns executions associated with a specific file.
func (s *WorkflowAppService) ListFileExecutions(ctx context.Context, fileID string) (*FileExecutionsResponse, error)
```

ListFileExecutions returns executions associated with a specific file.

### ListSemanticModelFileExecutions

```go
// ListSemanticModelFileExecutions returns executions associated with a file and knowledge base.
func (s *WorkflowAppService) ListSemanticModelFileExecutions(ctx context.Context, fileID string, semanticModelID int64) (*FileExecutionsResponse, error)
```

ListSemanticModelFileExecutions returns executions associated with a file and knowledge base.

### Pause

```go
// Pause pauses a workflow app and its currently active executions.
func (s *WorkflowAppService) Pause(ctx context.Context, workflowID string) (*WorkflowAppEnvelope, error)
```

Pause pauses a workflow app and its currently active executions.

### PauseExecution

```go
// PauseExecution pauses a running workflow execution at the next scheduler boundary.
func (s *WorkflowAppService) PauseExecution(ctx context.Context, workflowID, executionID string) (*WorkflowExecutionEnvelope, error)
```

PauseExecution pauses a running workflow execution at the next scheduler boundary.

### Resume

```go
// Resume resumes a paused workflow app and executions paused by workflow-level pause.
func (s *WorkflowAppService) Resume(ctx context.Context, workflowID string) (*WorkflowAppEnvelope, error)
```

Resume resumes a paused workflow app and executions paused by workflow-level pause.

### ResumeExecution

```go
// ResumeExecution resumes a paused workflow execution.
func (s *WorkflowAppService) ResumeExecution(ctx context.Context, workflowID, executionID string) (*WorkflowExecutionEnvelope, error)
```

ResumeExecution resumes a paused workflow execution.

### RetryExecution

```go
// RetryExecution creates a new execution using the previous execution payload.
func (s *WorkflowAppService) RetryExecution(ctx context.Context, workflowID, executionID string) (*WorkflowExecutionEnvelope, error)
```

RetryExecution creates a new execution using the previous execution payload.

### RunMemoryGovernanceExecution

```go
// RunMemoryGovernanceExecution dispatches a synchronized session through the
// trusted Backend runtime boundary. Browser and PAT clients must not call it.
func (s *WorkflowAppService) RunMemoryGovernanceExecution(ctx context.Context, workflowID string, req *MemoryGovernanceExecutionRequest) (*MemoryGovernanceExecutionResponse, error)
```

RunMemoryGovernanceExecution dispatches a synchronized session through the
trusted Backend runtime boundary. Browser and PAT clients must not call it.

### RunSystemWorkflow

```go
// RunSystemWorkflow provisions the hidden system workflow app and starts an execution.
func (s *WorkflowAppService) RunSystemWorkflow(ctx context.Context, kind string, req *SystemWorkflowExecutionRequest) (*SystemWorkflowExecutionResponse, error)
```

RunSystemWorkflow provisions the hidden system workflow app and starts an execution.

### Update

```go
// Update changes workflow app metadata.
func (s *WorkflowAppService) Update(ctx context.Context, workflowID string, req *WorkflowAppUpdateRequest) (*WorkflowAppEnvelope, error)
```

Update changes workflow app metadata.

### ValidateDelete

```go
// ValidateDelete checks whether a workflow app can be deleted without deleting it.
func (s *WorkflowAppService) ValidateDelete(ctx context.Context, workflowID string) (*WorkflowAppValidateDeleteResponse, error)
```

ValidateDelete checks whether a workflow app can be deleted without deleting it.

## WorkflowDeploymentDynamicService

WorkflowDeploymentDynamicService configures a dynamic-service deployment.

## WorkflowDeploymentService

WorkflowDeploymentService provides transactional workflow deployment APIs.

### Deploy

```go
// Deploy publishes or updates a workflow in one catalog transaction.
func (s *WorkflowDeploymentService) Deploy(ctx context.Context, req *WorkflowDeploymentRequest) (*WorkflowDeploymentResponse, error)
```

Deploy publishes or updates a workflow in one catalog transaction.

## WorkflowService

WorkflowService provides workflow definition management operations.
WorkflowService is always scoped to a workspace via client.Workflows(workspaceID).

Validates: Requirements 6.1-6.6

### Create

```go
// Create creates a new workflow definition.
//
// Parameters:
//   - ctx: Context for the request
//   - name: The workflow name (required, unique per user)
//   - opts: Optional parameters (WithWorkflowDefDescription)
//
// Returns the created workflow definition or an error.
//
// Example:
//
//	wf, err := client.Workflows(workspaceID).Create(ctx, "my-workflow",
//	    moi.WithWorkflowDefDescription("This workflow processes data"),
//	)
func (s *WorkflowService) Create(ctx context.Context, name string, opts ...CreateWorkflowDefOption) (*mowl.WorkflowDefinition, error)
```

Create creates a new workflow definition.

Parameters:
  - ctx: Context for the request
  - name: The workflow name (required, unique per user)
  - opts: Optional parameters (WithWorkflowDefDescription)

Returns the created workflow definition or an error.

Example:

	wf, err := client.Workflows(workspaceID).Create(ctx, "my-workflow",
	    moi.WithWorkflowDefDescription("This workflow processes data"),
	)

### Delete

```go
// Delete deletes a workflow definition.
// Only the workflow creator can delete the workflow.
// All versions will be deleted as well.
//
// Parameters:
//   - ctx: Context for the request
//   - workflowID: The workflow definition ID
//
// Returns an error if the deletion fails.
//
// Example:
//
//	err := client.Workflows(workspaceID).Delete(ctx, "workflow-uuid")
func (s *WorkflowService) Delete(ctx context.Context, workflowID string) error
```

Delete deletes a workflow definition.
Only the workflow creator can delete the workflow.
All versions will be deleted as well.

Parameters:
  - ctx: Context for the request
  - workflowID: The workflow definition ID

Returns an error if the deletion fails.

Example:

	err := client.Workflows(workspaceID).Delete(ctx, "workflow-uuid")

### Get

```go
// Get retrieves a workflow definition by ID.
//
// Parameters:
//   - ctx: Context for the request
//   - workflowID: The workflow definition ID
//
// Returns the workflow definition or an error if not found.
//
// Example:
//
//	wf, err := client.Workflows(workspaceID).Get(ctx, "workflow-uuid")
func (s *WorkflowService) Get(ctx context.Context, workflowID string) (*mowl.WorkflowDefinition, error)
```

Get retrieves a workflow definition by ID.

Parameters:
  - ctx: Context for the request
  - workflowID: The workflow definition ID

Returns the workflow definition or an error if not found.

Example:

	wf, err := client.Workflows(workspaceID).Get(ctx, "workflow-uuid")

### GetByName

```go
// GetByName retrieves a workflow definition by name.
//
// Parameters:
//   - ctx: Context for the request
//   - name: The workflow name
//
// Returns the workflow definition or an error if not found.
//
// Example:
//
//	wf, err := client.Workflows(workspaceID).GetByName(ctx, "my-workflow")
func (s *WorkflowService) GetByName(ctx context.Context, name string) (*mowl.WorkflowDefinition, error)
```

GetByName retrieves a workflow definition by name.

Parameters:
  - ctx: Context for the request
  - name: The workflow name

Returns the workflow definition or an error if not found.

Example:

	wf, err := client.Workflows(workspaceID).GetByName(ctx, "my-workflow")

### List

```go
// List retrieves workflow definitions with optional filters.
//
// Parameters:
//   - ctx: Context for the request
//   - opts: Optional filter parameters (WithWorkflowNameFilter)
//
// Returns a list of workflow definitions or an error.
//
// Example:
//
//	// List all workflows
//	workflows, err := client.Workflows(workspaceID).List(ctx)
//
//	// List workflows with name filter
//	workflows, err := client.Workflows(workspaceID).List(ctx, moi.WithWorkflowNameFilter("data-"))
func (s *WorkflowService) List(ctx context.Context, opts ...ListWorkflowsOption) ([]*mowl.WorkflowDefinition, error)
```

List retrieves workflow definitions with optional filters.

Parameters:
  - ctx: Context for the request
  - opts: Optional filter parameters (WithWorkflowNameFilter)

Returns a list of workflow definitions or an error.

Example:

	// List all workflows
	workflows, err := client.Workflows(workspaceID).List(ctx)

	// List workflows with name filter
	workflows, err := client.Workflows(workspaceID).List(ctx, moi.WithWorkflowNameFilter("data-"))

### Update

```go
// Update updates an existing workflow definition.
// Only the workflow creator can update the workflow.
//
// Parameters:
//   - ctx: Context for the request
//   - workflowID: The workflow definition ID
//   - opts: Fields to update (WithUpdatedWorkflowDefName, WithUpdatedWorkflowDefDescription)
//
// Returns an error if the update fails.
//
// Example:
//
//	err := client.Workflows(workspaceID).Update(ctx, "workflow-uuid",
//	    moi.WithUpdatedWorkflowDefName("new-name"),
//	    moi.WithUpdatedWorkflowDefDescription("Updated description"),
//	)
func (s *WorkflowService) Update(ctx context.Context, workflowID string, opts ...UpdateWorkflowDefOption) error
```

Update updates an existing workflow definition.
Only the workflow creator can update the workflow.

Parameters:
  - ctx: Context for the request
  - workflowID: The workflow definition ID
  - opts: Fields to update (WithUpdatedWorkflowDefName, WithUpdatedWorkflowDefDescription)

Returns an error if the update fails.

Example:

	err := client.Workflows(workspaceID).Update(ctx, "workflow-uuid",
	    moi.WithUpdatedWorkflowDefName("new-name"),
	    moi.WithUpdatedWorkflowDefDescription("Updated description"),
	)

## WorkflowVersionService

WorkflowVersionService provides workflow version management operations.
WorkflowVersionService is always scoped to a workspace via client.WorkflowVersions(workspaceID).

Validates: Requirements 6.7-6.12

### CreateByBuilder

```go
// CreateByBuilder creates a new workflow version from a DSL builder.
// Supports both regular workflows and dynamic services via options.
//
// Example (regular workflow):
//
//	version, err := client.WorkflowVersions(workspaceID).CreateByBuilder(ctx, "workflow-uuid", workflowBuilder)
//
// Example (dynamic service):
//
//	inputSchema := moi.NewSchema().Property("url", moi.StringSchema()).Required("url")
//	outputSchema := moi.NewSchema().Property("result", moi.StringSchema())
//	version, err := client.WorkflowVersions(workspaceID).CreateByBuilder(ctx, "workflow-uuid", workflowBuilder,
//	    moi.WithVersionDynamicService(inputSchema, outputSchema),
//	    moi.WithVersionResultMode("oneshot"),
//	)
func (s *WorkflowVersionService) CreateByBuilder(ctx context.Context, workflowID string, builder *dsl.WorkflowBuilder, opts ...CreateWorkflowVersionOption) (*mowl.WorkflowVersion, error)
```

CreateByBuilder creates a new workflow version from a DSL builder.
Supports both regular workflows and dynamic services via options.

Example (regular workflow):

	version, err := client.WorkflowVersions(workspaceID).CreateByBuilder(ctx, "workflow-uuid", workflowBuilder)

Example (dynamic service):

	inputSchema := moi.NewSchema().Property("url", moi.StringSchema()).Required("url")
	outputSchema := moi.NewSchema().Property("result", moi.StringSchema())
	version, err := client.WorkflowVersions(workspaceID).CreateByBuilder(ctx, "workflow-uuid", workflowBuilder,
	    moi.WithVersionDynamicService(inputSchema, outputSchema),
	    moi.WithVersionResultMode("oneshot"),
	)

### CreateByDSLBytes

```go
// CreateByDSLBytes creates a new workflow version from YAML/DSL bytes.
// Supports both regular workflows and dynamic services via options.
//
// Example:
//
//	version, err := client.WorkflowVersions(workspaceID).CreateByDSLBytes(ctx, "workflow-uuid", yamlBytes)
func (s *WorkflowVersionService) CreateByDSLBytes(ctx context.Context, workflowID string, yamlBytes []byte, opts ...CreateWorkflowVersionOption) (*mowl.WorkflowVersion, error)
```

CreateByDSLBytes creates a new workflow version from YAML/DSL bytes.
Supports both regular workflows and dynamic services via options.

Example:

	version, err := client.WorkflowVersions(workspaceID).CreateByDSLBytes(ctx, "workflow-uuid", yamlBytes)

### CreateByDSLFile

```go
// CreateByDSLFile creates a new workflow version from a YAML/DSL file.
// Supports both regular workflows and dynamic services via options.
//
// Example:
//
//	version, err := client.WorkflowVersions(workspaceID).CreateByDSLFile(ctx, "workflow-uuid", "workflow.yaml")
func (s *WorkflowVersionService) CreateByDSLFile(ctx context.Context, workflowID string, yamlPath string, opts ...CreateWorkflowVersionOption) (*mowl.WorkflowVersion, error)
```

CreateByDSLFile creates a new workflow version from a YAML/DSL file.
Supports both regular workflows and dynamic services via options.

Example:

	version, err := client.WorkflowVersions(workspaceID).CreateByDSLFile(ctx, "workflow-uuid", "workflow.yaml")

### Delete

```go
// Delete deletes a workflow version.
// Only the workflow definition creator can delete versions.
//
// Parameters:
//   - ctx: Context for the request
//   - versionID: The workflow version ID
//
// Returns an error if the deletion fails.
//
// Example:
//
//	err := client.WorkflowVersions(workspaceID).Delete(ctx, "version-uuid")
func (s *WorkflowVersionService) Delete(ctx context.Context, versionID string) error
```

Delete deletes a workflow version.
Only the workflow definition creator can delete versions.

Parameters:
  - ctx: Context for the request
  - versionID: The workflow version ID

Returns an error if the deletion fails.

Example:

	err := client.WorkflowVersions(workspaceID).Delete(ctx, "version-uuid")

### Deprecate

```go
// Deprecate deprecates a workflow version.
// Only the workflow definition creator can deprecate versions.
// Deprecated versions cannot be used to create new tasks.
//
// Parameters:
//   - ctx: Context for the request
//   - versionID: The workflow version ID
//
// Returns an error if the deprecation fails.
//
// Example:
//
//	err := client.WorkflowVersions(workspaceID).Deprecate(ctx, "version-uuid")
func (s *WorkflowVersionService) Deprecate(ctx context.Context, versionID string) error
```

Deprecate deprecates a workflow version.
Only the workflow definition creator can deprecate versions.
Deprecated versions cannot be used to create new tasks.

Parameters:
  - ctx: Context for the request
  - versionID: The workflow version ID

Returns an error if the deprecation fails.

Example:

	err := client.WorkflowVersions(workspaceID).Deprecate(ctx, "version-uuid")

### Get

```go
// Get retrieves a workflow version by ID.
//
// Parameters:
//   - ctx: Context for the request
//   - versionID: The workflow version ID
//
// Returns the workflow version or an error if not found.
//
// Example:
//
//	version, err := client.WorkflowVersions(workspaceID).Get(ctx, "version-uuid")
func (s *WorkflowVersionService) Get(ctx context.Context, versionID string) (*mowl.WorkflowVersion, error)
```

Get retrieves a workflow version by ID.

Parameters:
  - ctx: Context for the request
  - versionID: The workflow version ID

Returns the workflow version or an error if not found.

Example:

	version, err := client.WorkflowVersions(workspaceID).Get(ctx, "version-uuid")

### GetByVersion

```go
// GetByVersion retrieves a workflow version by version number.
//
// Parameters:
//   - ctx: Context for the request
//   - workflowID: The workflow definition ID
//   - version: The version number (1, 2, 3, ...)
//
// Returns the workflow version or an error if not found.
//
// Example:
//
//	version, err := client.WorkflowVersions(workspaceID).GetByVersion(ctx, "workflow-uuid", 1)
func (s *WorkflowVersionService) GetByVersion(ctx context.Context, workflowID string, version int32) (*mowl.WorkflowVersion, error)
```

GetByVersion retrieves a workflow version by version number.

Parameters:
  - ctx: Context for the request
  - workflowID: The workflow definition ID
  - version: The version number (1, 2, 3, ...)

Returns the workflow version or an error if not found.

Example:

	version, err := client.WorkflowVersions(workspaceID).GetByVersion(ctx, "workflow-uuid", 1)

### GetLatestPublished

```go
// GetLatestPublished retrieves the latest published workflow version.
//
// Parameters:
//   - ctx: Context for the request
//   - workflowID: The workflow definition ID
//
// Returns the latest published version or an error if not found.
//
// Example:
//
//	version, err := client.WorkflowVersions(workspaceID).GetLatestPublished(ctx, "workflow-uuid")
func (s *WorkflowVersionService) GetLatestPublished(ctx context.Context, workflowID string) (*mowl.WorkflowVersion, error)
```

GetLatestPublished retrieves the latest published workflow version.

Parameters:
  - ctx: Context for the request
  - workflowID: The workflow definition ID

Returns the latest published version or an error if not found.

Example:

	version, err := client.WorkflowVersions(workspaceID).GetLatestPublished(ctx, "workflow-uuid")

### List

```go
// List retrieves all versions of a workflow.
//
// Parameters:
//   - ctx: Context for the request
//   - workflowID: The workflow definition ID
//
// Returns a list of workflow versions or an error.
//
// Example:
//
//	versions, err := client.WorkflowVersions(workspaceID).List(ctx, "workflow-uuid")
func (s *WorkflowVersionService) List(ctx context.Context, workflowID string) ([]*mowl.WorkflowVersion, error)
```

List retrieves all versions of a workflow.

Parameters:
  - ctx: Context for the request
  - workflowID: The workflow definition ID

Returns a list of workflow versions or an error.

Example:

	versions, err := client.WorkflowVersions(workspaceID).List(ctx, "workflow-uuid")

### Publish

```go
// Publish publishes a workflow version.
// Only the workflow definition creator can publish versions.
// Published versions can be used to create tasks.
//
// Parameters:
//   - ctx: Context for the request
//   - versionID: The workflow version ID
//
// Returns an error if the publish fails.
//
// Example:
//
//	err := client.WorkflowVersions(workspaceID).Publish(ctx, "version-uuid")
func (s *WorkflowVersionService) Publish(ctx context.Context, versionID string) error
```

Publish publishes a workflow version.
Only the workflow definition creator can publish versions.
Published versions can be used to create tasks.

Parameters:
  - ctx: Context for the request
  - versionID: The workflow version ID

Returns an error if the publish fails.

Example:

	err := client.WorkflowVersions(workspaceID).Publish(ctx, "version-uuid")

## WorkspaceService

WorkspaceService provides workspace management operations.

### AcceptInvitation

```go
// AcceptInvitation accepts a pending invitation targeted to the authenticated principal.
func (s *WorkspaceService) AcceptInvitation(ctx context.Context, invitationID string) (*WorkspaceInvitation, error)
```

AcceptInvitation accepts a pending invitation targeted to the authenticated principal.

### Create

```go
// Create creates a new workspace.
//
// Parameters:
//   - ctx: Context for the request
//   - name: The workspace name (required)
//   - opts: Optional parameters (WithWorkspaceDescription)
//
// Returns the created workspace or an error.
//
// Example:
//
//	ws, err := client.Workspaces().Create(ctx, "my-workspace",
//	    moi.WithWorkspaceDescription("This is my workspace"),
//	)
func (s *WorkspaceService) Create(ctx context.Context, name string, opts ...CreateWorkspaceOption) (*workspace.Workspace, error)
```

Create creates a new workspace.

Parameters:
  - ctx: Context for the request
  - name: The workspace name (required)
  - opts: Optional parameters (WithWorkspaceDescription)

Returns the created workspace or an error.

Example:

	ws, err := client.Workspaces().Create(ctx, "my-workspace",
	    moi.WithWorkspaceDescription("This is my workspace"),
	)

### CreateInvitation

```go
// CreateInvitation creates an invitation for an existing Core principal.
// CompleteImmediately runs the acceptance Saga in the same request.
func (s *WorkspaceService) CreateInvitation(ctx context.Context, workspaceID string, req *CreateWorkspaceInvitationRequest) (*WorkspaceInvitation, error)
```

CreateInvitation creates an invitation for an existing Core principal.
CompleteImmediately runs the acceptance Saga in the same request.

### Delete

```go
// Delete deletes a workspace by ID.
//
// Parameters:
//   - ctx: Context for the request
//   - id: The workspace ID
//
// Returns an error if the deletion fails.
//
// Example:
//
//	err := client.Workspaces().Delete(ctx, "workspace-id-123")
func (s *WorkspaceService) Delete(ctx context.Context, id string) error
```

Delete deletes a workspace by ID.

Parameters:
  - ctx: Context for the request
  - id: The workspace ID

Returns an error if the deletion fails.

Example:

	err := client.Workspaces().Delete(ctx, "workspace-id-123")

### Get

```go
// Get retrieves a workspace by ID.
//
// Parameters:
//   - ctx: Context for the request
//   - id: The workspace ID
//
// Returns the workspace or ErrNotFound if not found.
//
// Example:
//
//	ws, err := client.Workspaces().Get(ctx, "workspace-id-123")
func (s *WorkspaceService) Get(ctx context.Context, id string) (*workspace.Workspace, error)
```

Get retrieves a workspace by ID.

Parameters:
  - ctx: Context for the request
  - id: The workspace ID

Returns the workspace or ErrNotFound if not found.

Example:

	ws, err := client.Workspaces().Get(ctx, "workspace-id-123")

### GetDBConnection

```go
// GetDBConnection retrieves the database connection information for a workspace.
//
// Parameters:
//   - ctx: Context for the request
//   - id: The workspace ID
//
// Returns the database connection information or ErrNotFound if the workspace is not found.
//
// Example:
//
//	conn, err := client.Workspaces().GetDBConnection(ctx, "workspace-id-123")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Connect to %s:%d as %s\n", conn.Host, conn.Port, conn.Username)
func (s *WorkspaceService) GetDBConnection(ctx context.Context, id string) (*workspace.DBConnection, error)
```

GetDBConnection retrieves the database connection information for a workspace.

Parameters:
  - ctx: Context for the request
  - id: The workspace ID

Returns the database connection information or ErrNotFound if the workspace is not found.

Example:

	conn, err := client.Workspaces().GetDBConnection(ctx, "workspace-id-123")
	if err != nil {
	    return err
	}
	fmt.Printf("Connect to %s:%d as %s\n", conn.Host, conn.Port, conn.Username)

### GetDBConnectionForRole

```go
// GetDBConnectionForRole retrieves workspace database credentials bound to the
// explicitly selected Effective Role.
func (s *WorkspaceService) GetDBConnectionForRole(ctx context.Context, id, roleID string) (*workspace.DBConnection, error)
```

GetDBConnectionForRole retrieves workspace database credentials bound to the
explicitly selected Effective Role.

### GetOwnerCredentialAPIKey

```go
// GetOwnerCredentialAPIKey retrieves metadata for the workspace OWNER's independent API key.
// This internal API is restricted to system users.
func (s *WorkspaceService) GetOwnerCredentialAPIKey(ctx context.Context, workspaceID, userID string) (*auth.APIKey, error)
```

GetOwnerCredentialAPIKey retrieves metadata for the workspace OWNER's independent API key.
This internal API is restricted to system users.

### GetOwnerCredentialDBConnection

```go
// GetOwnerCredentialDBConnection retrieves the workspace OWNER's independent admin connection.
// This internal API is restricted to system users.
func (s *WorkspaceService) GetOwnerCredentialDBConnection(ctx context.Context, workspaceID, userID string) (*workspace.DBConnection, error)
```

GetOwnerCredentialDBConnection retrieves the workspace OWNER's independent admin connection.
This internal API is restricted to system users.

### GetOwnerDBConnection

```go
// GetOwnerDBConnection retrieves the workspace owner's database connection information.
// This is an internal API restricted to system users only. The caller must authenticate
// with a system API key. The returned connection has admin-level privileges (e.g., CREATE TABLE).
//
// Parameters:
//   - ctx: Context for the request
//   - id: The workspace ID
//
// Returns the owner's database connection information or an error.
//
// Example:
//
//	conn, err := client.Workspaces().GetOwnerDBConnection(ctx, "workspace-id-123")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Connect as owner %s:%d\n", conn.Host, conn.Port)
func (s *WorkspaceService) GetOwnerDBConnection(ctx context.Context, id string) (*workspace.DBConnection, error)
```

GetOwnerDBConnection retrieves the workspace owner's database connection information.
This is an internal API restricted to system users only. The caller must authenticate
with a system API key. The returned connection has admin-level privileges (e.g., CREATE TABLE).

Parameters:
  - ctx: Context for the request
  - id: The workspace ID

Returns the owner's database connection information or an error.

Example:

	conn, err := client.Workspaces().GetOwnerDBConnection(ctx, "workspace-id-123")
	if err != nil {
	    return err
	}
	fmt.Printf("Connect as owner %s:%d\n", conn.Host, conn.Port)

### GetSystemRoles

```go
// GetSystemRoles retrieves stable internal references for moi-core-owned workspace system roles.
// This is an internal API restricted to system users only. The caller must authenticate
// with a system API key.
//
// Parameters:
//   - ctx: Context for the request
//   - id: The workspace ID
//
// Returns the superadmin and admin system role references or an error.
//
// Example:
//
//	systemRoles, err := client.Workspaces().GetSystemRoles(ctx, "workspace-id-123")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Admin role ID: %d\n", systemRoles.AdminRole.Id)
func (s *WorkspaceService) GetSystemRoles(ctx context.Context, id string) (*workspace.WorkspaceSystemRoles, error)
```

GetSystemRoles retrieves stable internal references for moi-core-owned workspace system roles.
This is an internal API restricted to system users only. The caller must authenticate
with a system API key.

Parameters:
  - ctx: Context for the request
  - id: The workspace ID

Returns the superadmin and admin system role references or an error.

Example:

	systemRoles, err := client.Workspaces().GetSystemRoles(ctx, "workspace-id-123")
	if err != nil {
	    return err
	}
	fmt.Printf("Admin role ID: %d\n", systemRoles.AdminRole.Id)

### GetUserDBConnection

```go
// GetUserDBConnection retrieves the database connection information for a specified user in a workspace.
// This is an internal API restricted to system users only. The caller must authenticate
// with a system API key.
//
// Parameters:
//   - ctx: Context for the request
//   - workspaceID: The workspace ID
//   - userID: The target user ID
//
// Returns the user's database connection information or an error.
//
// Example:
//
//	conn, err := client.Workspaces().GetUserDBConnection(ctx, "workspace-id-123", "user-id-456")
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Connect as %s:%d\n", conn.Host, conn.Port)
func (s *WorkspaceService) GetUserDBConnection(ctx context.Context, workspaceID, userID string) (*workspace.DBConnection, error)
```

GetUserDBConnection retrieves the database connection information for a specified user in a workspace.
This is an internal API restricted to system users only. The caller must authenticate
with a system API key.

Parameters:
  - ctx: Context for the request
  - workspaceID: The workspace ID
  - userID: The target user ID

Returns the user's database connection information or an error.

Example:

	conn, err := client.Workspaces().GetUserDBConnection(ctx, "workspace-id-123", "user-id-456")
	if err != nil {
	    return err
	}
	fmt.Printf("Connect as %s:%d\n", conn.Host, conn.Port)

### GetUserDBConnectionForRole

```go
// GetUserDBConnectionForRole retrieves a target user's connection for an
// explicitly selected Effective Role. It is restricted to system callers.
func (s *WorkspaceService) GetUserDBConnectionForRole(ctx context.Context, workspaceID, userID, roleID string) (*workspace.DBConnection, error)
```

GetUserDBConnectionForRole retrieves a target user's connection for an
explicitly selected Effective Role. It is restricted to system callers.

### List

```go
// List retrieves one workspace page.
//
// This compatibility helper does not expose the next_page_token. Use ListPage
// when the caller needs to continue a scan, or ListAll when it intentionally
// needs to materialize every page.
//
// Parameters:
//   - ctx: Context for the request
//   - opts: Optional pagination parameters (WithPageSize, WithPageToken)
//
// Returns a list of workspaces or an error.
//
// Example:
//
//	workspaces, err := client.Workspaces().List(ctx,
//	    moi.WithPageSize(10),
//	    moi.WithPageToken("next-page-token"),
//	)
func (s *WorkspaceService) List(ctx context.Context, opts ...ListOption) ([]*workspace.Workspace, error)
```

List retrieves one workspace page.

This compatibility helper does not expose the next_page_token. Use ListPage
when the caller needs to continue a scan, or ListAll when it intentionally
needs to materialize every page.

Parameters:
  - ctx: Context for the request
  - opts: Optional pagination parameters (WithPageSize, WithPageToken)

Returns a list of workspaces or an error.

Example:

	workspaces, err := client.Workspaces().List(ctx,
	    moi.WithPageSize(10),
	    moi.WithPageToken("next-page-token"),
	)

### ListAll

```go
// ListAll retrieves and materializes every workspace page beginning at the
// optional page token. Use ListPage instead when a caller must retain its own
// cursor or bound the response size in memory.
func (s *WorkspaceService) ListAll(ctx context.Context, opts ...ListOption) ([]*workspace.Workspace, error)
```

ListAll retrieves and materializes every workspace page beginning at the
optional page token. Use ListPage instead when a caller must retain its own
cursor or bound the response size in memory.

### ListAllRunnableSystemWorkspaces

```go
// ListAllRunnableSystemWorkspaces retrieves every NORMAL workspace through the
// system-only control-plane endpoint. It is for trusted background services;
// it must not be used for a human caller's workspace list.
func (s *WorkspaceService) ListAllRunnableSystemWorkspaces(ctx context.Context) ([]*workspace.Workspace, error)
```

ListAllRunnableSystemWorkspaces retrieves every NORMAL workspace through the
system-only control-plane endpoint. It is for trusted background services;
it must not be used for a human caller's workspace list.

### ListPage

```go
// ListPage retrieves one keyset page and returns its opaque next_page_token.
// Use this method for control-plane scans that must retain progress without
// materializing the complete workspace inventory.
func (s *WorkspaceService) ListPage(ctx context.Context, opts ...ListOption) (*workspace.ListWorkspacesResponse, error)
```

ListPage retrieves one keyset page and returns its opaque next_page_token.
Use this method for control-plane scans that must retain progress without
materializing the complete workspace inventory.

### ListPendingInvitations

```go
// ListPendingInvitations lists pending invitations targeted to the authenticated principal.
func (s *WorkspaceService) ListPendingInvitations(ctx context.Context) ([]WorkspaceInvitation, error)
```

ListPendingInvitations lists pending invitations targeted to the authenticated principal.

### ListPendingInvitationsForWorkspace

```go
// ListPendingInvitationsForWorkspace lists pending invitations targeting one workspace via the system API.
func (s *WorkspaceService) ListPendingInvitationsForWorkspace(ctx context.Context, workspaceID string) ([]WorkspaceInvitation, error)
```

ListPendingInvitationsForWorkspace lists pending invitations targeting one workspace via the system API.

### OpenDB

```go
// OpenDB retrieves workspace database credentials and opens a MatrixOne
// connection that applies the moi-core-provided session initialization SQLs on
// every physical connection.
func (s *WorkspaceService) OpenDB(ctx context.Context, id string, opts ...DBOpenOption) (*sql.DB, *workspace.DBConnection, error)
```

OpenDB retrieves workspace database credentials and opens a MatrixOne
connection that applies the moi-core-provided session initialization SQLs on
every physical connection.

### OpenDBForRole

```go
// OpenDBForRole opens a MatrixOne connection whose physical sessions are
// initialized with the explicitly selected Effective Role.
func (s *WorkspaceService) OpenDBForRole(ctx context.Context, id, roleID string, opts ...DBOpenOption) (*sql.DB, *workspace.DBConnection, error)
```

OpenDBForRole opens a MatrixOne connection whose physical sessions are
initialized with the explicitly selected Effective Role.

### ResolveNames

```go
// ResolveNames resolves minimal id/name projections under the verified
// Effective Role of currentWorkspaceID. Returned projections are display-only
// and are not proof of membership in the projected workspaces.
func (s *WorkspaceService) ResolveNames(ctx context.Context, currentWorkspaceID string, ids []string) ([]WorkspaceNameProjection, error)
```

ResolveNames resolves minimal id/name projections under the verified
Effective Role of currentWorkspaceID. Returned projections are display-only
and are not proof of membership in the projected workspaces.

### RevealOwnerCredentialAPIKey

```go
// RevealOwnerCredentialAPIKey reveals the workspace OWNER's current independent API key.
// This internal API is restricted to system users.
func (s *WorkspaceService) RevealOwnerCredentialAPIKey(ctx context.Context, workspaceID, userID string) (*auth.APIKeyWithSecret, error)
```

RevealOwnerCredentialAPIKey reveals the workspace OWNER's current independent API key.
This internal API is restricted to system users.

### RotateOwnerCredentialAPIKey

```go
// RotateOwnerCredentialAPIKey rotates the workspace OWNER's independent API key.
// This internal API is restricted to system users.
func (s *WorkspaceService) RotateOwnerCredentialAPIKey(ctx context.Context, workspaceID, userID string) (*auth.APIKeyWithSecret, error)
```

RotateOwnerCredentialAPIKey rotates the workspace OWNER's independent API key.
This internal API is restricted to system users.

### Update

```go
// Update updates an existing workspace.
//
// Parameters:
//   - ctx: Context for the request
//   - id: The workspace ID
//   - opts: Fields to update (WithWorkspaceName, WithUpdatedWorkspaceDescription)
//
// Returns the updated workspace or an error.
//
// Example:
//
//	ws, err := client.Workspaces().Update(ctx, "workspace-id-123",
//	    moi.WithWorkspaceName("new-name"),
//	    moi.WithUpdatedWorkspaceDescription("Updated description"),
//	)
func (s *WorkspaceService) Update(ctx context.Context, id string, opts ...UpdateWorkspaceOption) (*workspace.Workspace, error)
```

Update updates an existing workspace.

Parameters:
  - ctx: Context for the request
  - id: The workspace ID
  - opts: Fields to update (WithWorkspaceName, WithUpdatedWorkspaceDescription)

Returns the updated workspace or an error.

Example:

	ws, err := client.Workspaces().Update(ctx, "workspace-id-123",
	    moi.WithWorkspaceName("new-name"),
	    moi.WithUpdatedWorkspaceDescription("Updated description"),
	)

## 工具函数

### AnySchema

```go
// AnySchema 创建一个任意类型的 schema（空对象）
func AnySchema() *SchemaBuilder
```

AnySchema 创建一个任意类型的 schema（空对象）

### ArraySchema

```go
// ArraySchema 创建一个数组类型的 schema
func ArraySchema() *SchemaBuilder
```

ArraySchema 创建一个数组类型的 schema

### AttachProvenance

```go
// AttachProvenance attaches a `_provenance` object to an item map.
func AttachProvenance(item map[string]interface{}, _ *RuntimeContext, rootAssetID, stageAssetID, rawFileID, kind string) map[string]interface{}
```

AttachProvenance attaches a `_provenance` object to an item map.

### BindToFromString

```go
// BindToFromString creates a json.RawMessage from a simple bind_to string.
func BindToFromString(s string) json.RawMessage
```

BindToFromString creates a json.RawMessage from a simple bind_to string.

### BooleanSchema

```go
// BooleanSchema 创建一个布尔类型的 schema
func BooleanSchema() *SchemaBuilder
```

BooleanSchema 创建一个布尔类型的 schema

### BuildFormLayout

```go
// BuildFormLayout computes a deterministic form layout by mapping each field's
// bind_to path(s) to the work_item(s) that reference the same scoped path in the DSL.
//
// Grouping rules:
//   - field matches exactly one workitem → grouped under that workitem
//   - exact bind_to matches take precedence over prefix matches
//   - field matches multiple layout owners → grouped under "shared"
//   - compound bind_to paths hitting different layout owners → grouped under "shared"
//   - document_visual text/image index workitems share one drawing knowledge-index owner
//   - standard RAG compound index fields that hit knowledge.index.build and its
//     optional image branch stay under the knowledge index owner
//   - field matches no workitem → grouped under "ungrouped"
//
// Group ordering: DSL workitem traversal order, then "shared", then "ungrouped".
func BuildFormLayout(dslYAML string, form *WorkflowAgentInputForm) (*FormLayout, error)
```

BuildFormLayout computes a deterministic form layout by mapping each field's
bind_to path(s) to the work_item(s) that reference the same scoped path in the DSL.

Grouping rules:
  - field matches exactly one workitem → grouped under that workitem
  - exact bind_to matches take precedence over prefix matches
  - field matches multiple layout owners → grouped under "shared"
  - compound bind_to paths hitting different layout owners → grouped under "shared"
  - document_visual text/image index workitems share one drawing knowledge-index owner
  - standard RAG compound index fields that hit knowledge.index.build and its
    optional image branch stay under the knowledge index owner
  - field matches no workitem → grouped under "ungrouped"

Group ordering: DSL workitem traversal order, then "shared", then "ungrouped".

### BuildPayloadProvenance

```go
// BuildPayloadProvenance builds the runtime payload anchor used by Phase 2.
func BuildPayloadProvenance(rootAssetID, stageAssetID, rawFileID, kind string) map[string]interface{}
```

BuildPayloadProvenance builds the runtime payload anchor used by Phase 2.

### BuildWorkItemCapabilities

```go
// BuildWorkItemCapabilities converts WorkItems.List() response into LLM-friendly capability context.
func BuildWorkItemCapabilities(workitems map[string]*mowl.WorkItemMetadataList, opts ...WorkItemCapabilityBuildOption) []WorkflowWorkItemCapability
```

BuildWorkItemCapabilities converts WorkItems.List() response into LLM-friendly capability context.

### BuildWorkItemUIMetadata

```go
// BuildWorkItemUIMetadata converts raw workitem metadata to parsed UI metadata.
// Returns an error if any item contains invalid/missing normalized UI contract fields.
func BuildWorkItemUIMetadata(workitems map[string]*mowl.WorkItemMetadataList) (map[string]*WorkItemUIMetadataList, error)
```

BuildWorkItemUIMetadata converts raw workitem metadata to parsed UI metadata.
Returns an error if any item contains invalid/missing normalized UI contract fields.

### CanonicalWorkflowPipelineJSON

```go
// CanonicalWorkflowPipelineJSON projects a published Mowl Workflow into the
// immutable node snapshot used by product runtimes. Node names are stable DSL
// identities; WorkItem IDs, pinned versions, dependencies and variables are
// preserved exactly, while protobuf map iteration order is removed.
func CanonicalWorkflowPipelineJSON(workflow *mowl.Workflow) (string, error)
```

CanonicalWorkflowPipelineJSON projects a published Mowl Workflow into the
immutable node snapshot used by product runtimes. Node names are stable DSL
identities; WorkItem IDs, pinned versions, dependencies and variables are
preserved exactly, while protobuf map iteration order is removed.

### ChatCompletionRequestBodySize

```go
// ChatCompletionRequestBodySize returns the exact serialized JSON size for the
// supplied request without applying its body-size limit or performing network
// I/O. Callers that embed base64 payloads can use it to reserve envelope and
// prompt overhead before applying WithChatRequestBodyMaxBytes as the final
// hard guard.
func ChatCompletionRequestBodySize(question, model string, stream bool, opts ...ChatCompletionOption) (int, error)
```

ChatCompletionRequestBodySize returns the exact serialized JSON size for the
supplied request without applying its body-size limit or performing network
I/O. Callers that embed base64 payloads can use it to reserve envelope and
prompt overhead before applying WithChatRequestBodyMaxBytes as the final
hard guard.

### CompareSemverLikeStable

```go
// CompareSemverLikeStable compares semantic-like versions.
// It first uses numeric semantic parts and then applies lexical tiebreaking
// when numeric parts are equal but raw version strings differ.
func CompareSemverLikeStable(a, b string) int
```

CompareSemverLikeStable compares semantic-like versions.
It first uses numeric semantic parts and then applies lexical tiebreaking
when numeric parts are equal but raw version strings differ.

### CompileWorkflowDSLWithDiagnostics

```go
// CompileWorkflowDSLWithDiagnostics compiles YAML DSL into workflow and emits structured diagnostics.
// It validates in 3 stages: parse -> schema -> build.
func CompileWorkflowDSLWithDiagnostics(yamlBytes []byte) (*WorkflowDSLCompileResult, error)
```

CompileWorkflowDSLWithDiagnostics compiles YAML DSL into workflow and emits structured diagnostics.
It validates in 3 stages: parse -> schema -> build.

### ComputeFieldNodeMappings

```go
// ComputeFieldNodeMappings determines which work_item nodes reference each form
// field by matching bind_to paths against template references in the DSL. Fields
// mapping to 2+ nodes would be placed in the "shared" group by BuildFormLayout.
func ComputeFieldNodeMappings(dslYAML string, form *WorkflowAgentInputForm) ([]FieldNodeMapping, error)
```

ComputeFieldNodeMappings determines which work_item nodes reference each form
field by matching bind_to paths against template references in the DSL. Fields
mapping to 2+ nodes would be placed in the "shared" group by BuildFormLayout.

### ContextReplaceHeaders

```go
// ContextReplaceHeaders replaces the full set of SDK transport headers on ctx.
// Parent headers are not merged. Empty values are still stored and will be sent.
func ContextReplaceHeaders(ctx context.Context, headers map[string]string) context.Context
```

ContextReplaceHeaders replaces the full set of SDK transport headers on ctx.
Parent headers are not merged. Empty values are still stored and will be sent.

### ContextWithHeader

```go
// ContextWithHeader returns a context carrying one extra HTTP header while
// preserving headers already attached to the context.
func ContextWithHeader(ctx context.Context, key, value string) context.Context
```

ContextWithHeader returns a context carrying one extra HTTP header while
preserving headers already attached to the context.

### ContextWithHeaders

```go
// ContextWithHeaders returns a context carrying extra HTTP headers that the SDK
// client will automatically include in every request (e.g. Accept-Language for i18n).
// Headers merge onto any already attached to ctx; use ContextReplaceHeaders to
// drop inherited keys when switching execution identity.
func ContextWithHeaders(ctx context.Context, headers map[string]string) context.Context
```

ContextWithHeaders returns a context carrying extra HTTP headers that the SDK
client will automatically include in every request (e.g. Accept-Language for i18n).
Headers merge onto any already attached to ctx; use ContextReplaceHeaders to
drop inherited keys when switching execution identity.

### DefaultRepairContextPolicy

```go
// DefaultRepairContextPolicy returns conservative defaults for LLM prompt safety.
func DefaultRepairContextPolicy() RepairContextPolicy
```

DefaultRepairContextPolicy returns conservative defaults for LLM prompt safety.

### DefaultWorkflowDSLDefinition

```go
// DefaultWorkflowDSLDefinition returns canonical machine-readable DSL definition metadata.
func DefaultWorkflowDSLDefinition() WorkflowDSLDefinition
```

DefaultWorkflowDSLDefinition returns canonical machine-readable DSL definition metadata.

### DeriveWorkItemCategoryAndTags

```go
// DeriveWorkItemCategoryAndTags exposes the standard node-id based category/tag fallback.
func DeriveWorkItemCategoryAndTags(nodeID string) (string, []string)
```

DeriveWorkItemCategoryAndTags exposes the standard node-id based category/tag fallback.

### DeriveWorkItemDisplayName

```go
// DeriveWorkItemDisplayName exposes the standard node-id based display name fallback.
func DeriveWorkItemDisplayName(nodeID string) string
```

DeriveWorkItemDisplayName exposes the standard node-id based display name fallback.

### ExtractBindToPaths

```go
// extractBindToPaths returns all scoped paths from a field's bind_to.
// ExtractBindToPaths returns all scoped paths from a field's bind_to.
// Simple: "vars.source_ref" → ["vars.source_ref"], "input.source_ref" → ["data.source_ref"]
// Compound: {"vector_table":"vars.vector_index.vector_table",...} → ["vars.vector_index.vector_table", ...]
func ExtractBindToPaths(raw json.RawMessage) []string
```

extractBindToPaths returns all scoped paths from a field's bind_to.
ExtractBindToPaths returns all scoped paths from a field's bind_to.
Simple: "vars.source_ref" → ["vars.source_ref"], "input.source_ref" → ["data.source_ref"]
Compound: {"vector_table":"vars.vector_index.vector_table",...} → ["vars.vector_index.vector_table", ...]

### HeadersFromContext

```go
// HeadersFromContext returns the HTTP headers carried for Core transport.
func HeadersFromContext(ctx context.Context) map[string]string
```

HeadersFromContext returns the HTTP headers carried for Core transport.

### IntegerSchema

```go
// IntegerSchema 创建一个整数类型的 schema
func IntegerSchema() *SchemaBuilder
```

IntegerSchema 创建一个整数类型的 schema

### IsCode

```go
// IsCode 判断错误是否为指定错误码。支持包装后的 *Error（通过 errors.As 解包）。
func IsCode(err error, code common.ErrorCode) bool
```

IsCode 判断错误是否为指定错误码。支持包装后的 *Error（通过 errors.As 解包）。

### IsCustomOperatorNodeID

```go
// IsCustomOperatorNodeID reports whether nodeID belongs to the Custom Operator
// namespace. The namespace is the canonical source classification boundary.
func IsCustomOperatorNodeID(nodeID string) bool
```

IsCustomOperatorNodeID reports whether nodeID belongs to the Custom Operator
namespace. The namespace is the canonical source classification boundary.

### IsNotFound

```go
// IsNotFound 判断错误是否为 NOT_FOUND（HTTP 404）。
func IsNotFound(err error) bool
```

IsNotFound 判断错误是否为 NOT_FOUND（HTTP 404）。

### IsSupportedWorkItemUIWidget

```go
// IsSupportedWorkItemUIWidget reports whether widget is accepted by WorkItem UI metadata validation.
func IsSupportedWorkItemUIWidget(widget string) bool
```

IsSupportedWorkItemUIWidget reports whether widget is accepted by WorkItem UI metadata validation.

### IsZeroWorkItemUISchema

```go
// IsZeroWorkItemUISchema reports whether a WorkItem UI schema has no meaningful fields.
func IsZeroWorkItemUISchema(schema *WorkItemUISchema) bool
```

IsZeroWorkItemUISchema reports whether a WorkItem UI schema has no meaningful fields.

### MergeLineageMeta

```go
// MergeLineageMeta merges DSL meta with runtime lineage fields.
func MergeLineageMeta(meta map[string]interface{}, runtime *RuntimeContext, rootAssetID, stageAssetID, producerWorkitemID string) map[string]interface{}
```

MergeLineageMeta merges DSL meta with runtime lineage fields.

### NewCloudNonUserConnector

```go
// NewCloudNonUserConnector marks untagged SQL executed through base as
// MatrixOne cloud_nonuser_sql while preserving explicit user SQL markers.
func NewCloudNonUserConnector(base driver.Connector) driver.Connector
```

NewCloudNonUserConnector marks untagged SQL executed through base as
MatrixOne cloud_nonuser_sql while preserving explicit user SQL markers.

### NewConnectorSecretClient

```go
// NewConnectorSecretClient creates a system API-key authenticated client for
// Catalog dataconn secret RPCs.
func NewConnectorSecretClient(endpoint, apiKey string) (*ConnectorSecretClient, error)
```

NewConnectorSecretClient creates a system API-key authenticated client for
Catalog dataconn secret RPCs.

### NewContext

```go
// NewContext creates a new WorkItemContext from a MowlMessage.
// It parses the workflow definition from the message's Node field (JSON-serialized)
// and locates the current node using the Consumer field as the node ID.
func NewContext(ctx context.Context, msg *mowl.MowlMessage) (WorkItemContext, error)
```

NewContext creates a new WorkItemContext from a MowlMessage.
It parses the workflow definition from the message's Node field (JSON-serialized)
and locates the current node using the Consumer field as the node ID.

### NewHTTPNotification

```go
// NewHTTPNotification creates a new HTTP notification configuration.
//
// Example:
//
//	notification := moi.NewHTTPNotification("https://callback.example.com").
//	    WithMethod("POST").
//	    WithTimeout(30).
//	    WithHeaders(map[string]string{"Authorization": "Bearer token"}).
//	    Build()
func NewHTTPNotification(url string) *NotificationBuilder
```

NewHTTPNotification creates a new HTTP notification configuration.

Example:

	notification := moi.NewHTTPNotification("https://callback.example.com").
	    WithMethod("POST").
	    WithTimeout(30).
	    WithHeaders(map[string]string{"Authorization": "Bearer token"}).
	    Build()

### NewI18nPacks

```go
// NewI18nPacks creates a new i18n pack builder.
func NewI18nPacks() *I18nPackBuilder
```

NewI18nPacks creates a new i18n pack builder.

### NewInternalInvokerClient

```go
// NewInternalInvokerClient creates a client for Mowl registered WorkItem RPCs.
// When apiKey is non-empty, it is sent as ordinary x-api-key metadata for Catalog gRPC.
func NewInternalInvokerClient(endpoint, apiKey string) (*InternalInvokerClient, error)
```

NewInternalInvokerClient creates a client for Mowl registered WorkItem RPCs.
When apiKey is non-empty, it is sent as ordinary x-api-key metadata for Catalog gRPC.

### NewParseResult

```go
// NewParseResult creates a new ParseResult with the given content and options.
func NewParseResult(content string, opts ...ParseResultOption) *catalogpb.ParseResult
```

NewParseResult creates a new ParseResult with the given content and options.

### NewSchema

```go
// NewSchema 创建一个新的 SchemaBuilder
func NewSchema() *SchemaBuilder
```

NewSchema 创建一个新的 SchemaBuilder

### NewTraceRenderer

```go
// NewTraceRenderer creates a renderer for the given response.
func NewTraceRenderer(resp *mowlpb.TraceResponse) *TraceRenderer
```

NewTraceRenderer creates a renderer for the given response.

### NewWithBearerToken

```go
// NewWithBearerToken is retained for source compatibility. New code that
// makes service-account requests must use NewWithServiceAccountBearerToken so
// the authentication category is explicit at the call site.
//
// Deprecated: use NewWithServiceAccountBearerToken.
func NewWithBearerToken(endpoint, accessToken string, opts ...Option) (*Client, error)
```

NewWithBearerToken is retained for source compatibility. New code that
makes service-account requests must use NewWithServiceAccountBearerToken so
the authentication category is explicit at the call site.

Deprecated: use NewWithServiceAccountBearerToken.

### NewWithCatalogProvisionerBearerToken

```go
// NewWithCatalogProvisionerBearerToken creates the server-to-server client
// for the UC -> AI Studio read-only catalog boundary. Its token is a dedicated
// provisioner credential, not a service-account data-plane token. Catalog
// calls additionally require a per-request UC actor assertion; this
// constructor must never be used by browser code.
//
// The transport still sends exactly one Authorization: Bearer <accessToken>
// header and no X-API-Key or Cookie. AI Studio distinguishes this credential
// from a service-account data token by verified azp/audience/scope claims and
// rejects principal_type=service_account on catalog routes.
func NewWithCatalogProvisionerBearerToken(endpoint, accessToken string, opts ...Option) (*Client, error)
```

NewWithCatalogProvisionerBearerToken creates the server-to-server client
for the UC -> AI Studio read-only catalog boundary. Its token is a dedicated
provisioner credential, not a service-account data-plane token. Catalog
calls additionally require a per-request UC actor assertion; this
constructor must never be used by browser code.

The transport still sends exactly one Authorization: Bearer <accessToken>
header and no X-API-Key or Cookie. AI Studio distinguishes this credential
from a service-account data token by verified azp/audience/scope claims and
rejects principal_type=service_account on catalog routes.

### NewWithPersonalAccessToken

```go
// NewWithPersonalAccessToken creates a moi-core SDK client for a UC personal
// access token (PAT). The token is opaque: this constructor never infers its
// credential class from a prefix. Every request sends exactly one
// X-API-Key header and no Authorization or Cookie header.
//
// personalAccessToken must be a non-empty raw X-API-Key header value. Its
// contents are opaque to the SDK; the authoritative verifier determines
// whether it is valid.
func NewWithPersonalAccessToken(endpoint, personalAccessToken string, opts ...Option) (*Client, error)
```

NewWithPersonalAccessToken creates a moi-core SDK client for a UC personal
access token (PAT). The token is opaque: this constructor never infers its
credential class from a prefix. Every request sends exactly one
X-API-Key header and no Authorization or Cookie header.

personalAccessToken must be a non-empty raw X-API-Key header value. Its
contents are opaque to the SDK; the authoritative verifier determines
whether it is valid.

### NewWithServiceAccountBearerToken

```go
// NewWithServiceAccountBearerToken creates a moi-core SDK client for a
// service-account access token. It authenticates every request with exactly
// one Authorization: Bearer <accessToken> header and never sends X-API-Key or
// Cookie. The API, not this SDK, validates the token's issuer, audience and
// service-account claims.
//
// accessToken must be the raw token, not an already formatted HTTP header.
// Tokens are intentionally treated as opaque: this SDK does not guess their
// class from a prefix and therefore cannot turn a legacy PAT into a valid
// service-account principal.
func NewWithServiceAccountBearerToken(endpoint, accessToken string, opts ...Option) (*Client, error)
```

NewWithServiceAccountBearerToken creates a moi-core SDK client for a
service-account access token. It authenticates every request with exactly
one Authorization: Bearer <accessToken> header and never sends X-API-Key or
Cookie. The API, not this SDK, validates the token's issuer, audience and
service-account claims.

accessToken must be the raw token, not an already formatted HTTP header.
Tokens are intentionally treated as opaque: this SDK does not guess their
class from a prefix and therefore cannot turn a legacy PAT into a valid
service-account principal.

### NewWorkerNotification

```go
// NewWorkerNotification creates a new worker notification configuration.
//
// Example:
//
//	notification := moi.NewWorkerNotification("worker-123").
//	    WithMessage("task_completed").
//	    Build()
func NewWorkerNotification(workerID string) *NotificationBuilder
```

NewWorkerNotification creates a new worker notification configuration.

Example:

	notification := moi.NewWorkerNotification("worker-123").
	    WithMessage("task_completed").
	    Build()

### NumberSchema

```go
// NumberSchema 创建一个数字类型的 schema
func NumberSchema() *SchemaBuilder
```

NumberSchema 创建一个数字类型的 schema

### ObjectSchema

```go
// ObjectSchema 创建一个对象类型的 schema（等同于 NewSchema）
func ObjectSchema() *SchemaBuilder
```

ObjectSchema 创建一个对象类型的 schema（等同于 NewSchema）

### OpenCloudNonUserDBFromConnection

```go
// OpenCloudNonUserDBFromConnection opens a MatrixOne database for platform
// SQL that must be recorded as cloud_nonuser_sql.
func OpenCloudNonUserDBFromConnection(conn *workspace.DBConnection, opts ...DBOpenOption) (*sql.DB, error)
```

OpenCloudNonUserDBFromConnection opens a MatrixOne database for platform
SQL that must be recorded as cloud_nonuser_sql.

### OpenDBFromConnection

```go
// OpenDBFromConnection opens a MatrixOne database using DBConnection and
// applies SessionInitSqls on every physical connection.
func OpenDBFromConnection(conn *workspace.DBConnection, opts ...DBOpenOption) (*sql.DB, error)
```

OpenDBFromConnection opens a MatrixOne database using DBConnection and
applies SessionInitSqls on every physical connection.

### ParseTemplateRefs

```go
// parseTemplateRefs extracts both .vars.xxx and .data.xxx paths from template strings.
// Returns scoped paths, e.g. "vars.source_ref", "data.source_ref".
// ParseTemplateRefs extracts scoped variable references (.vars.X, .data.Y) from a template string.
func ParseTemplateRefs(tmpl string) []string
```

parseTemplateRefs extracts both .vars.xxx and .data.xxx paths from template strings.
Returns scoped paths, e.g. "vars.source_ref", "data.source_ref".
ParseTemplateRefs extracts scoped variable references (.vars.X, .data.Y) from a template string.

### PrepareWorkflowDSLRepairContext

```go
// PrepareWorkflowDSLRepairContext normalizes size, fills missing constraint defaults,
// derives allowed_node_ids from workitems when absent, and validates the result.
func PrepareWorkflowDSLRepairContext(input *WorkflowDSLRepairContext, policy RepairContextPolicy) (*WorkflowDSLRepairContext, error)
```

PrepareWorkflowDSLRepairContext normalizes size, fills missing constraint defaults,
derives allowed_node_ids from workitems when absent, and validates the result.

### SetWorkflowValueAtPath

```go
// SetWorkflowValueAtPath sets a nested value by dot path. Supports "[]" for the first array
// element — missing intermediate maps/arrays are created on the fly. Examples:
//
//	file_id                              -> {"file_id": ...}
//	sources[].file_id                    -> {"sources":[{"file_id": ...}]}
//	execution_context.workspace_id       -> {"execution_context":{"workspace_id": ...}}
func SetWorkflowValueAtPath(root map[string]interface{}, path string, value interface{}) error
```

SetWorkflowValueAtPath sets a nested value by dot path. Supports "[]" for the first array
element — missing intermediate maps/arrays are created on the fly. Examples:

	file_id                              -> {"file_id": ...}
	sources[].file_id                    -> {"sources":[{"file_id": ...}]}
	execution_context.workspace_id       -> {"execution_context":{"workspace_id": ...}}

### ShadowedLineageMetaKeys

```go
// ShadowedLineageMetaKeys returns protected keys present in DSL meta that will
// be overwritten by runtime lineage fields during derivation creation.
func ShadowedLineageMetaKeys(meta map[string]interface{}, runtime *RuntimeContext, rootAssetID, stageAssetID, producerWorkitemID string) []string
```

ShadowedLineageMetaKeys returns protected keys present in DSL meta that will
be overwritten by runtime lineage fields during derivation creation.

### StringSchema

```go
// StringSchema 创建一个字符串类型的 schema
func StringSchema() *SchemaBuilder
```

StringSchema 创建一个字符串类型的 schema

### ValidateExplicitWorkItemMetadataContract

```go
// ValidateExplicitWorkItemMetadataContract validates complete user-facing metadata contract fields.
func ValidateExplicitWorkItemMetadataContract(name string, md *mowl.WorkItemMetadata) error
```

ValidateExplicitWorkItemMetadataContract validates complete user-facing metadata contract fields.

### ValidateRuntimeConfigContract

```go
// ValidateRuntimeConfigContract checks that every runtime parameter declared by workitems
// used in the DSL is handled according to its exposure:
//   - exposure=always returns error diagnostics when the DSL/form does not expose it
//   - exposure=ask returns warning diagnostics so callers can ask the user
//   - exposure=optional is ignored unless the DSL already includes it
//
// It returns diagnostics for missing fields. The caller (e.g. evaluateCandidate) decides
// whether to block the candidate or surface warnings.
func ValidateRuntimeConfigContract(
	dslYAML string,
	form *WorkflowAgentInputForm,
	provider RuntimeConfigContractProvider,
	opts ...RuntimeConfigValidationOption,
) ([]RuntimeConfigDiagnostic, error)
```

ValidateRuntimeConfigContract checks that every runtime parameter declared by workitems
used in the DSL is handled according to its exposure:
  - exposure=always returns error diagnostics when the DSL/form does not expose it
  - exposure=ask returns warning diagnostics so callers can ask the user
  - exposure=optional is ignored unless the DSL already includes it

It returns diagnostics for missing fields. The caller (e.g. evaluateCandidate) decides
whether to block the candidate or surface warnings.

### ValidateTemplateExpressions

```go
// ValidateTemplateExpressions validates {{ jq }} expressions in workflow DSL text
// against gojq parse/compile rules and the published template contract roots.
// This is the shared gate used by agent compile, catalog deploy, and backend save paths.
func ValidateTemplateExpressions(dslYAML string) []DSLCompileDiagnostic
```

ValidateTemplateExpressions validates {{ jq }} expressions in workflow DSL text
against gojq parse/compile rules and the published template contract roots.
This is the shared gate used by agent compile, catalog deploy, and backend save paths.

### ValidateTemplateExpressionsError

```go
// ValidateTemplateExpressionsError returns a single error when DSL contains invalid
// template expressions. Intended for deploy/save gates that only accept error values.
func ValidateTemplateExpressionsError(dslYAML string) error
```

ValidateTemplateExpressionsError returns a single error when DSL contains invalid
template expressions. Intended for deploy/save gates that only accept error values.

### ValidateWorkItemMetadataContract

```go
// ValidateWorkItemMetadataContract validates registration-safe metadata contract fields against input/output schemas.
func ValidateWorkItemMetadataContract(name string, md *mowl.WorkItemMetadata) error
```

ValidateWorkItemMetadataContract validates registration-safe metadata contract fields against input/output schemas.

### WorkItemMetadataAgentContext

```go
// WorkItemMetadataAgentContext extracts compact agent-facing facts from WorkItem metadata.
func WorkItemMetadataAgentContext(md *mowl.WorkItemMetadata) WorkItemAgentContext
```

WorkItemMetadataAgentContext extracts compact agent-facing facts from WorkItem metadata.

### WorkflowValueAtPath

```go
// WorkflowValueAtPath retrieves a value from a nested map by workflow path. Supports array
// notation like "sources[].file_id" — the "[]" segment dereferences the first element.
// Used by the server to expand submit-form values into candidate input/vars via bind_to.
func WorkflowValueAtPath(obj map[string]interface{}, path string) (interface{}, bool)
```

WorkflowValueAtPath retrieves a value from a nested map by workflow path. Supports array
notation like "sources[].file_id" — the "[]" segment dereferences the first element.
Used by the server to expand submit-form values into candidate input/vars via bind_to.

## 类型定义

### A2ARequest

A2ARequest is the JSON-RPC envelope for an A2A method call.

```go
// A2ARequest is the JSON-RPC envelope for an A2A method call.
type A2ARequest struct {
	JSONRPC	string		`json:"jsonrpc"`
	ID	any		`json:"id,omitempty"`
	Method	string		`json:"method"`
	Params	json.RawMessage	`json:"params,omitempty"`
}
```

### A2AStreamEvent

A2AStreamEvent is one server-sent event from a streaming A2A request.

```go
// A2AStreamEvent is one server-sent event from a streaming A2A request.
type A2AStreamEvent struct {
	ID	string		`json:"id,omitempty"`
	Event	string		`json:"event,omitempty"`
	Data	json.RawMessage	`json:"data"`
}
```

### A2AStreamResult

A2AStreamResult preserves the distinction between a clean stream EOF and a
transport read failure for server-side proxies that need terminal-aware
persistence. Event is populated when Err is nil.

```go
// A2AStreamResult preserves the distinction between a clean stream EOF and a
// transport read failure for server-side proxies that need terminal-aware
// persistence. Event is populated when Err is nil.
type A2AStreamResult struct {
	Event	A2AStreamEvent
	Err	error
}
```

### APIKey

APIKey represents an API key (without the secret).
Re-exported from model/auth for convenience.

```go
// APIKey represents an API key (without the secret).
// Re-exported from model/auth for convenience.
type APIKey = auth.APIKey
```

### APIKeyWithSecret

APIKeyWithSecret represents an API key with its secret.
Only returned when creating a new API key.
Re-exported from model/auth for convenience.

```go
// APIKeyWithSecret represents an API key with its secret.
// Only returned when creating a new API key.
// Re-exported from model/auth for convenience.
type APIKeyWithSecret = auth.APIKeyWithSecret
```

### AddFileItem

AddFileItem represents a file to add with optional metadata overrides.

```go
// AddFileItem represents a file to add with optional metadata overrides.
type AddFileItem struct {
	FileID		string	`json:"file_id"`
	FileName	string	`json:"file_name,omitempty"`
	FilePath	string	`json:"file_path,omitempty"`
}
```

### AddFilesOption

AddFilesOption is a function that configures add files options.

```go
// AddFilesOption is a function that configures add files options.
type AddFilesOption func(*addFilesOptions)
```

### AddFilesRequest

AddFilesRequest represents the request to add files to a volume.

```go
// AddFilesRequest represents the request to add files to a volume.
type AddFilesRequest struct {
	FileIDs	[]string	`json:"file_ids"`
	// RequireUnlinked is an association constraint, not uploader authorization.
	RequireUnlinked	bool	`json:"require_unlinked,omitempty"`
}
```

### AgentA2ARequest

AgentA2ARequest combines an agent selector with an A2A JSON-RPC request.

```go
// AgentA2ARequest combines an agent selector with an A2A JSON-RPC request.
type AgentA2ARequest struct {
	AgentSelector
	A2ARequest
}
```

### AgentLineage

AgentLineage describes the workspace lineage record shared by all versions of one Agent.

```go
// AgentLineage describes the workspace lineage record shared by all versions of one Agent.
type AgentLineage struct {
	WorkspaceID	string	`json:"workspace_id"`
	AgentID		string	`json:"agent_id"`
	DefaultVersion	string	`json:"default_version"`
	CreatedAt	string	`json:"created_at"`
	UpdatedAt	string	`json:"updated_at"`
}
```

### AgentPackageDownloadResponse

AgentPackageDownloadResponse holds an exported .moiagent archive stream.

```go
// AgentPackageDownloadResponse holds an exported .moiagent archive stream.
type AgentPackageDownloadResponse struct {
	Body		io.ReadCloser
	ContentType	string
}
```

### AgentPackageLoadDiagnostic

AgentPackageLoadDiagnostic is a validation or import diagnostic emitted while loading a package.

```go
// AgentPackageLoadDiagnostic is a validation or import diagnostic emitted while loading a package.
type AgentPackageLoadDiagnostic struct {
	Severity	string	`json:"severity"`
	Code		string	`json:"code"`
	Message		string	`json:"message"`
	Ref		string	`json:"ref,omitempty"`
}
```

### AgentPackageLoadPlan

AgentPackageLoadPlan mirrors the server-side pre-mutation load plan.

```go
// AgentPackageLoadPlan mirrors the server-side pre-mutation load plan.
type AgentPackageLoadPlan struct {
	WorkspaceID		string				`json:"workspace_id"`
	AgentID			string				`json:"agent_id"`
	AgentVersion		string				`json:"agent_version"`
	SourceDigest		string				`json:"source_digest"`
	RequiredPermissions	[]string			`json:"required_permissions,omitempty"`
	ResourceFiles		AgentPackageResourceFiles	`json:"resource_files"`
	Status			string				`json:"status"`
	Diagnostics		[]AgentPackageLoadDiagnostic	`json:"diagnostics,omitempty"`
}
```

### AgentPackageLoadResponse

AgentPackageLoadResponse is returned after a successful package load.

```go
// AgentPackageLoadResponse is returned after a successful package load.
type AgentPackageLoadResponse struct {
	Plan	AgentPackageLoadPlan	`json:"plan"`
	Version	AgentVersion		`json:"version"`
}
```

### AgentPackageResourceFiles

AgentPackageResourceFiles lists package resource file paths relevant to package load.

```go
// AgentPackageResourceFiles lists package resource file paths relevant to package load.
type AgentPackageResourceFiles struct {
	Instruction	string		`json:"instruction,omitempty"`
	CustomToolDirs	[]string	`json:"custom_tool_dirs,omitempty"`
	CustomSkillDirs	[]string	`json:"custom_skill_dirs,omitempty"`
}
```

### AgentSelector

AgentSelector identifies a registered agent by code or ID.

```go
// AgentSelector identifies a registered agent by code or ID.
type AgentSelector struct {
	AgentCode		string	`json:"agent_code,omitempty"`
	AgentID			string	`json:"agent_id,omitempty"`
	AgentWorkspaceID	string	`json:"agent_workspace_id,omitempty"`
	WorkspaceID		string	`json:"-"`
}
```

### AgentVersion

AgentVersion describes one loaded version of an Agent package.

```go
// AgentVersion describes one loaded version of an Agent package.
type AgentVersion struct {
	WorkspaceID	string				`json:"workspace_id"`
	AgentID		string				`json:"agent_id"`
	Version		string				`json:"version"`
	SourceDigest	string				`json:"source_digest"`
	MinMOIVersion	string				`json:"min_moi_version"`
	Status		string				`json:"status"`
	Diagnostics	[]AgentPackageLoadDiagnostic	`json:"diagnostics,omitempty"`
	LoadedBy	string				`json:"loaded_by"`
	LoadedAt	string				`json:"loaded_at"`
	DisabledBy	string				`json:"disabled_by,omitempty"`
	DisabledAt	string				`json:"disabled_at,omitempty"`
	Manifest	map[string]any			`json:"manifest,omitempty"`
}
```

### AgentVersionDeleteResponse

AgentVersionDeleteResponse reports whether a disabled Agent version was deleted.

```go
// AgentVersionDeleteResponse reports whether a disabled Agent version was deleted.
type AgentVersionDeleteResponse struct {
	Deleted bool `json:"deleted"`
}
```

### AgentVersionListResponse

AgentVersionListResponse is returned by AgentVersionService.List.

```go
// AgentVersionListResponse is returned by AgentVersionService.List.
type AgentVersionListResponse struct {
	Items	[]AgentVersion	`json:"items"`
	Total	int		`json:"total"`
	Lineage	*AgentLineage	`json:"lineage,omitempty"`
}
```

### AppendModifiedResponseOption

AppendModifiedResponseOption configures append modified response options.

```go
// AppendModifiedResponseOption configures append modified response options.
type AppendModifiedResponseOption func(*appendModifiedResponseOptions)
```

### AttachBuiltinFilesRequest

AttachBuiltinFilesRequest contains shared objects to attach to one Volume.

```go
// AttachBuiltinFilesRequest contains shared objects to attach to one Volume.
type AttachBuiltinFilesRequest struct {
	Items []BuiltinFileAttachment `json:"items"`
}
```

### AttachBuiltinFilesResponse

AttachBuiltinFilesResponse contains the attached stable file IDs.

```go
// AttachBuiltinFilesResponse contains the attached stable file IDs.
type AttachBuiltinFilesResponse struct {
	FileIDs []string `json:"file_ids"`
}
```

### BaseWorkItemContext

BaseWorkItemContext provides a default RuntimeContext implementation for
downstream custom contexts that already satisfy the rest of WorkItemContext.
Embed this struct to stay source-compatible when new runtime metadata helpers
are added to the interface.

```go
// BaseWorkItemContext provides a default RuntimeContext implementation for
// downstream custom contexts that already satisfy the rest of WorkItemContext.
// Embed this struct to stay source-compatible when new runtime metadata helpers
// are added to the interface.
type BaseWorkItemContext struct{}
```

### BatchFailure

BatchFailure represents a single failure in a batch operation.

```go
// BatchFailure represents a single failure in a batch operation.
type BatchFailure struct {
	ID	int64
	Error	error
}
```

### BatchOption

BatchOption is a function that configures batch options.

```go
// BatchOption is a function that configures batch options.
type BatchOption func(*batchOptions)
```

### BatchResult

BatchResult represents the result of a batch operation.

```go
// BatchResult represents the result of a batch operation.
type BatchResult struct {
	SuccessCount	int
	FailureCount	int
	Failures	[]BatchFailure
}
```

### BatchRetryExecutionsRequest

BatchRetryExecutionsRequest selects failed file executions from one source dispatch batch.

```go
// BatchRetryExecutionsRequest selects failed file executions from one source dispatch batch.
type BatchRetryExecutionsRequest struct {
	SourceDispatchJobID	string		`json:"source_dispatch_job_id"`
	ExecutionIDs		[]string	`json:"execution_ids,omitempty"`
	AllFailed		bool		`json:"all_failed,omitempty"`
	RequestID		string		`json:"request_id"`
}
```

### BatchRetryExecutionsResponse

BatchRetryExecutionsResponse describes the accepted retry dispatch batch.

```go
// BatchRetryExecutionsResponse describes the accepted retry dispatch batch.
type BatchRetryExecutionsResponse struct {
	SourceDispatchJobID	string	`json:"source_dispatch_job_id"`
	DispatchJobID		string	`json:"dispatch_job_id"`
	Status			string	`json:"status"`
	AcceptedCount		int	`json:"accepted_count"`
	Replayed		bool	`json:"replayed"`
}
```

### BuiltinFileAttachment

BuiltinFileAttachment describes the tenant-local reference created for a
system-owned shared object.

```go
// BuiltinFileAttachment describes the tenant-local reference created for a
// system-owned shared object.
type BuiltinFileAttachment struct {
	FileID		string	`json:"file_id"`
	OriginalName	string	`json:"original_name"`
	Size		int64	`json:"size"`
	MD5		string	`json:"md5"`
	ContentType	string	`json:"content_type"`
}
```

### BuiltinFileMetadata

BuiltinFileMetadata describes one immutable object shared by every workspace.

```go
// BuiltinFileMetadata describes one immutable object shared by every workspace.
type BuiltinFileMetadata struct {
	FileID	string	`json:"file_id"`
	Size	int64	`json:"size"`
	MD5	string	`json:"md5"`
}
```

### CDHColumn

CDHColumn represents a CDH column.
Re-exported from model/catalog for convenience.

```go
// CDHColumn represents a CDH column.
// Re-exported from model/catalog for convenience.
type CDHColumn = catalog.CDHColumn
```

### CDHConfig

CDHConfig represents a CDH connection configuration.
Re-exported from model/catalog for convenience.

```go
// CDHConfig represents a CDH connection configuration.
// Re-exported from model/catalog for convenience.
type CDHConfig = catalog.CDHConfig
```

### CDHDatabase

CDHDatabase represents a CDH database.
Re-exported from model/catalog for convenience.

```go
// CDHDatabase represents a CDH database.
// Re-exported from model/catalog for convenience.
type CDHDatabase = catalog.CDHDatabase
```

### CDHHealthCheckResponse

CDHHealthCheckResponse represents the response from CDH health check.
Re-exported from model/catalog for convenience.

```go
// CDHHealthCheckResponse represents the response from CDH health check.
// Re-exported from model/catalog for convenience.
type CDHHealthCheckResponse = catalog.CDHHealthCheckResponse
```

### CDHTable

CDHTable represents a CDH table.
Re-exported from model/catalog for convenience.

```go
// CDHTable represents a CDH table.
// Re-exported from model/catalog for convenience.
type CDHTable = catalog.CDHTable
```

### CallbackHandlerFunc

CallbackHandlerFunc is the handler function type for callback messages.
It receives the callback message and returns a response or an error.

```go
// CallbackHandlerFunc is the handler function type for callback messages.
// It receives the callback message and returns a response or an error.
type CallbackHandlerFunc func(ctx context.Context, msg *CallbackMessage) (*CallbackResponse, error)
```

### CallbackMessage

CallbackMessage represents an incoming callback message from the workflow engine.

```go
// CallbackMessage represents an incoming callback message from the workflow engine.
type CallbackMessage struct {
	// CaseID is the workflow case ID.
	CaseID	string
	// Message is the callback message type/name.
	Message	string
	// Data is the callback payload data.
	Data	string
	// Vars is the workflow variables at the time of callback.
	Vars	string
}
```

### CallbackResponse

CallbackResponse represents the response to a callback message.

```go
// CallbackResponse represents the response to a callback message.
type CallbackResponse struct {
	// Data is the response payload data.
	Data	string
	// Error is an optional error message.
	Error	string
}
```

### CaseStatusResponse

CaseStatusResponse represents the status of a workflow case.

```go
// CaseStatusResponse represents the status of a workflow case.
type CaseStatusResponse struct {
	CaseID	string	`json:"case_id"`
	Status	string	`json:"status"`
	Result	string	`json:"result,omitempty"`
	Error	string	`json:"error,omitempty"`
}
```

### Catalog

Catalog represents a catalog in the system.
Re-exported from model/catalog for convenience.

```go
// Catalog represents a catalog in the system.
// Re-exported from model/catalog for convenience.
type Catalog = catalog.Catalog
```

### CatalogFile

CatalogFile is the durable private Catalog identity returned by private uploads.

```go
// CatalogFile is the durable private Catalog identity returned by private uploads.
type CatalogFile struct {
	WorkspaceID	string	`json:"workspace_id"`
	VolumeID	int64	`json:"volume_id"`
	FileID		string	`json:"file_id"`
	Path		string	`json:"path"`
	Name		string	`json:"name"`
	Size		int64	`json:"size"`
	MD5		string	`json:"md5"`
}
```

### CatalogIterator

CatalogIterator iterates over catalogs with automatic pagination.

```go
// CatalogIterator iterates over catalogs with automatic pagination.
type CatalogIterator struct {
	service		*CatalogService
	ctx		context.Context
	workspaceID	string
	buffer		[]*catalog.Catalog
	index		int
	pageToken	string
	pageSize	int32
	done		bool
	stopped		bool
	err		error
}
```

### CatalogListItem

CatalogListItem is a frontend-friendly workitem summary.

```go
// CatalogListItem is a frontend-friendly workitem summary.
type CatalogListItem struct {
	NodeID			string		`json:"node_id"`
	DisplayName		string		`json:"display_name"`
	ShortDescription	string		`json:"short_description,omitempty"`
	Category		string		`json:"category,omitempty"`
	Tags			[]string	`json:"tags,omitempty"`
	Source			string		`json:"source,omitempty"`
	Provider		string		`json:"provider,omitempty"`
	PreferredVersion	string		`json:"preferred_version,omitempty"`
	Visibility		string		`json:"visibility,omitempty"`
	Stream			bool		`json:"stream"`
}
```

### CatalogListResponse

CatalogListResponse is the response from the catalog list endpoint.

```go
// CatalogListResponse is the response from the catalog list endpoint.
type CatalogListResponse struct {
	Items	[]CatalogListItem	`json:"items"`
	Total	int			`json:"total"`
}
```

### CatalogNode

CatalogNode represents a catalog in the tree.

```go
// CatalogNode represents a catalog in the tree.
type CatalogNode struct {
	catalog		*catalog.Catalog
	parent		TreeNode
	children	[]TreeNode
}
```

### CatalogResolveRequest

CatalogResolveRequest represents a catalog resolve workitem input.
Re-exported from model/data for convenience.

```go
// CatalogResolveRequest represents a catalog resolve workitem input.
// Re-exported from model/data for convenience.
type CatalogResolveRequest = data.CatalogResolveRequest
```

### CatalogResolveResponse

CatalogResolveResponse represents a catalog resolve workitem output.
Re-exported from model/data for convenience.

```go
// CatalogResolveResponse represents a catalog resolve workitem output.
// Re-exported from model/data for convenience.
type CatalogResolveResponse = data.CatalogResolveResponse
```

### CatalogStatsResponse

CatalogStatsResponse holds the statistics for a catalog.

```go
// CatalogStatsResponse holds the statistics for a catalog.
type CatalogStatsResponse struct {
	DatabaseCount	int64	`json:"database_count"`
	TableCount	int64	`json:"table_count"`
	VolumeCount	int64	`json:"volume_count"`
	FileCount	int64	`json:"file_count"`
}
```

### CatalogSummary

CatalogSummary is a lightweight Catalog directory projection.

```go
// CatalogSummary is a lightweight Catalog directory projection.
type CatalogSummary = catalog.CatalogSummary
```

### ChatCompletionOption

ChatCompletionOption configures chat completion request.

```go
// ChatCompletionOption configures chat completion request.
type ChatCompletionOption func(*chatCompletionOptions)
```

### ChatCompletionRequestBodyTooLargeError

ChatCompletionRequestBodyTooLargeError reports a local preflight rejection.
The request has not been sent when this error is returned.

```go
// ChatCompletionRequestBodyTooLargeError reports a local preflight rejection.
// The request has not been sent when this error is returned.
type ChatCompletionRequestBodyTooLargeError struct {
	Bytes		int
	MaxBytes	int
}
```

### CleanTextRequest

CleanTextRequest represents a clean text workitem input.
Re-exported from model/data for convenience.

```go
// CleanTextRequest represents a clean text workitem input.
// Re-exported from model/data for convenience.
type CleanTextRequest = data.CleanTextRequest
```

### CleanTextResponse

CleanTextResponse represents a clean text workitem output.
Re-exported from model/data for convenience.

```go
// CleanTextResponse represents a clean text workitem output.
// Re-exported from model/data for convenience.
type CleanTextResponse = data.CleanTextResponse
```

### CompensateDatabaseCreateIAMRequest

CompensateDatabaseCreateIAMRequest identifies the exact Database create IAM
ownership mutation that a trusted system caller needs to roll back.

```go
// CompensateDatabaseCreateIAMRequest identifies the exact Database create IAM
// ownership mutation that a trusted system caller needs to roll back.
type CompensateDatabaseCreateIAMRequest struct {
	DatabaseID	int64	`json:"database_id"`
	CatalogID	int64	`json:"catalog_id"`
	PrincipalID	string	`json:"principal_id"`
	// OriginalCreateRequestID is the wire request ID used by the create request.
	// Core derives the resource-scoped lifecycle request ID internally.
	OriginalCreateRequestID	string	`json:"original_create_request_id"`
	RequestID		string	`json:"request_id"`
	TraceID			string	`json:"trace_id"`
}
```

### CompensateTableCreateIAMRequest

CompensateTableCreateIAMRequest identifies the exact Table create IAM
ownership mutation that a trusted system caller needs to roll back.

```go
// CompensateTableCreateIAMRequest identifies the exact Table create IAM
// ownership mutation that a trusted system caller needs to roll back.
type CompensateTableCreateIAMRequest struct {
	TableID		int64	`json:"table_id"`
	DatabaseID	int64	`json:"database_id"`
	PrincipalID	string	`json:"principal_id"`
	// OriginalCreateRequestID is the wire request ID used by the create request.
	// Core derives the resource-scoped lifecycle request ID internally.
	OriginalCreateRequestID	string	`json:"original_create_request_id"`
	RequestID		string	`json:"request_id"`
	TraceID			string	`json:"trace_id"`
}
```

### ComputeResourceBinding

ComputeResourceBinding describes a workflow-level or node-level compute resource reference.

```go
// ComputeResourceBinding describes a workflow-level or node-level compute resource reference.
type ComputeResourceBinding struct {
	ID		string		`json:"id"`
	Name		string		`json:"name,omitempty"`
	WorkflowLevel	bool		`json:"workflow_level,omitempty"`
	NodeNames	[]string	`json:"node_names,omitempty"`
}
```

### ConnectorSecretClient

ConnectorSecretClient is an internal service-to-service client for
Catalog-owned dataconn secret references.

```go
// ConnectorSecretClient is an internal service-to-service client for
// Catalog-owned dataconn secret references.
type ConnectorSecretClient struct {
	endpoint	string
	apiKey		string
	conn		*grpc.ClientConn
	client		dataconn.ConnectorSecretServiceClient
}
```

### ConvertPlainRequest

ConvertPlainRequest represents a convert plain workitem input.
Re-exported from model/data for convenience.

```go
// ConvertPlainRequest represents a convert plain workitem input.
// Re-exported from model/data for convenience.
type ConvertPlainRequest = data.ConvertPlainRequest
```

### ConvertPlainResponse

ConvertPlainResponse represents a convert plain workitem output.
Re-exported from model/data for convenience.

```go
// ConvertPlainResponse represents a convert plain workitem output.
// Re-exported from model/data for convenience.
type ConvertPlainResponse = data.ConvertPlainResponse
```

### ConvertResponse

ConvertResponse represents the response from document conversion.

```go
// ConvertResponse represents the response from document conversion.
type ConvertResponse struct {
	FileBytes	[]byte			`json:"file_bytes"`
	OutputFormat	string			`json:"output_format"`
	FileSize	int64			`json:"file_size"`
	Metadata	map[string]interface{}	`json:"metadata"`
}
```

### CreateAPIKeyOption

CreateAPIKeyOption is a function that configures create API key options.

```go
// CreateAPIKeyOption is a function that configures create API key options.
type CreateAPIKeyOption func(*createAPIKeyOptions)
```

### CreateBackendOption

CreateBackendOption configures create backend options.

```go
// CreateBackendOption configures create backend options.
type CreateBackendOption func(*createBackendOptions)
```

### CreateCDHConfigOption

CreateCDHConfigOption is a function that configures create CDH config options.

```go
// CreateCDHConfigOption is a function that configures create CDH config options.
type CreateCDHConfigOption func(*createCDHConfigOptions)
```

### CreateCatalogOption

CreateCatalogOption is a function that configures create catalog options.

```go
// CreateCatalogOption is a function that configures create catalog options.
type CreateCatalogOption func(*createCatalogOptions)
```

### CreateDPConfigOption

CreateDPConfigOption is a function that configures create Dataphin config options.

```go
// CreateDPConfigOption is a function that configures create Dataphin config options.
type CreateDPConfigOption func(*createDPConfigOptions)
```

### CreateEndpointOption

CreateEndpointOption configures create endpoint options.

```go
// CreateEndpointOption configures create endpoint options.
type CreateEndpointOption func(*createEndpointOptions)
```

### CreateMCConfigOption

CreateMCConfigOption is a function that configures create MaxCompute config options.

```go
// CreateMCConfigOption is a function that configures create MaxCompute config options.
type CreateMCConfigOption func(*createMCConfigOptions)
```

### CreateParserBackendOption

CreateParserBackendOption configures create parser backend options.

```go
// CreateParserBackendOption configures create parser backend options.
type CreateParserBackendOption func(*createParserBackendOptions)
```

### CreateSessionOption

CreateSessionOption configures create session options.

```go
// CreateSessionOption configures create session options.
type CreateSessionOption func(*createSessionOptions)
```

### CreateTagOption

CreateTagOption configures create tag options.

```go
// CreateTagOption configures create tag options.
type CreateTagOption func(*createTagOptions)
```

### CreateTaskOption

CreateTaskOption is a function that configures create task options.

```go
// CreateTaskOption is a function that configures create task options.
type CreateTaskOption func(*createTaskOptions)
```

### CreateUserOption

CreateUserOption is a function that configures create user options.

```go
// CreateUserOption is a function that configures create user options.
type CreateUserOption func(*createUserOptions)
```

### CreateVolumeOption

CreateVolumeOption is a function that configures create volume options.

```go
// CreateVolumeOption is a function that configures create volume options.
type CreateVolumeOption func(*createVolumeOptions)
```

### CreateWorkflowDefOption

CreateWorkflowDefOption is a function that configures create workflow definition options.

```go
// CreateWorkflowDefOption is a function that configures create workflow definition options.
type CreateWorkflowDefOption func(*createWorkflowDefOptions)
```

### CreateWorkflowVersionOption

CreateWorkflowVersionOption is a function that configures create workflow version options.

```go
// CreateWorkflowVersionOption is a function that configures create workflow version options.
type CreateWorkflowVersionOption func(*createWorkflowVersionOptions)
```

### CreateWorkspaceInvitationRequest

CreateWorkspaceInvitationRequest describes a workspace invitation created through the Backend BFF boundary.

```go
// CreateWorkspaceInvitationRequest describes a workspace invitation created through the Backend BFF boundary.
type CreateWorkspaceInvitationRequest struct {
	TargetUserID		string					`json:"target_user_id"`	// Canonical Core users.id resolved by the BFF.
	InvitedByUserID		string					`json:"invited_by_user_id"`
	InitialRoleIDs		[]string				`json:"initial_role_ids"`
	DefaultRoleID		string					`json:"default_role_id"`
	MemberAlias		string					`json:"member_alias"`
	MemberDescription	string					`json:"member_description"`
	SubjectAttributes	[]WorkspaceInvitationSubjectAttribute	`json:"subject_attributes,omitempty"`
	RequestID		string					`json:"request_id"`
	TraceID			string					`json:"trace_id"`
	EffectiveRoleID		string					`json:"effective_role_id"`
	CompleteImmediately	bool					`json:"complete_immediately,omitempty"`
}
```

### CreateWorkspaceOption

CreateWorkspaceOption is a function that configures create workspace options.

```go
// CreateWorkspaceOption is a function that configures create workspace options.
type CreateWorkspaceOption func(*createWorkspaceOptions)
```

### CustomOperator

CustomOperator is a workspace custom workflow operator.

```go
// CustomOperator is a workspace custom workflow operator.
type CustomOperator = catalog.CustomOperator
```

### CustomOperatorBuilder

CustomOperatorBuilder builds and creates a workspace custom operator.

```go
// CustomOperatorBuilder builds and creates a workspace custom operator.
type CustomOperatorBuilder struct {
	service	*CustomOperatorService
	req	customOperatorCreateRequest
	err	error
}
```

### DBConnection

DBConnection represents database connection information.
Re-exported from model/workspace for convenience.

```go
// DBConnection represents database connection information.
// Re-exported from model/workspace for convenience.
type DBConnection = workspace.DBConnection
```

### DBOpenOption

DBOpenOption configures a database connection opened from DBConnection.

```go
// DBOpenOption configures a database connection opened from DBConnection.
type DBOpenOption func(*dbOpenConfig)
```

### DSLCompileDiagnostic

DSLCompileDiagnostic captures compile/parse validation failures for workflow DSL.

```go
// DSLCompileDiagnostic captures compile/parse validation failures for workflow DSL.
type DSLCompileDiagnostic struct {
	Code		string		`json:"code,omitempty"`
	Stage		string		`json:"stage,omitempty"`
	Path		string		`json:"path,omitempty"`
	Line		int		`json:"line,omitempty"`
	Column		int		`json:"column,omitempty"`
	Node		string		`json:"node,omitempty"`
	Message		string		`json:"message"`
	Suggestion	string		`json:"suggestion,omitempty"`
	Raw		string		`json:"raw,omitempty"`
	Evidence	map[string]any	`json:"evidence,omitempty"`
	Metadata	map[string]any	`json:"metadata,omitempty"`
}
```

### DataAssetLinkRequest

DataAssetLinkRequest represents a data asset link workitem input.
Re-exported from model/data for convenience.

```go
// DataAssetLinkRequest represents a data asset link workitem input.
// Re-exported from model/data for convenience.
type DataAssetLinkRequest = data.DataAssetLinkRequest
```

### DataAssetOption

DataAssetOption configures CreateAsset.

```go
// DataAssetOption configures CreateAsset.
type DataAssetOption func(*dataAssetOptions)
```

### DataAssetRegisterRequest

DataAssetRegisterRequest represents a data asset register workitem input.
Re-exported from model/data for convenience.

```go
// DataAssetRegisterRequest represents a data asset register workitem input.
// Re-exported from model/data for convenience.
type DataAssetRegisterRequest = data.DataAssetRegisterRequest
```

### DataAssetResolveOption

DataAssetResolveOption configures ResolveAsset.

```go
// DataAssetResolveOption configures ResolveAsset.
type DataAssetResolveOption func(*dataAssetResolveOptions)
```

### DataBundleResolveRequest

DataBundleResolveRequest represents a data bundle resolve workitem input.
Re-exported from model/data for convenience.

```go
// DataBundleResolveRequest represents a data bundle resolve workitem input.
// Re-exported from model/data for convenience.
type DataBundleResolveRequest = data.DataBundleResolveRequest
```

### DataDashboardAlertEvaluationRequest

DataDashboardAlertEvaluationRequest carries the successful SQL result rows.

```go
// DataDashboardAlertEvaluationRequest carries the successful SQL result rows.
type DataDashboardAlertEvaluationRequest struct {
	Rows []map[string]any `json:"rows"`
}
```

### DataDashboardAlertEvaluationResult

DataDashboardAlertEvaluationResult describes the persisted alert state transition.

```go
// DataDashboardAlertEvaluationResult describes the persisted alert state transition.
type DataDashboardAlertEvaluationResult struct {
	Triggered	bool	`json:"triggered"`
	Changed		bool	`json:"changed"`
	NotificationDue	bool	`json:"notification_due"`
	Status		string	`json:"status"`
	RecordID	string	`json:"record_id"`
	XValue		string	`json:"x_value"`
	Value		float64	`json:"value"`
}
```

### DataDashboardExecutionSpec

DataDashboardExecutionSpec is the authorized SQL execution scope for a chart.

```go
// DataDashboardExecutionSpec is the authorized SQL execution scope for a chart.
type DataDashboardExecutionSpec struct {
	DashboardID		string		`json:"dashboard_id"`
	ChartID			string		`json:"chart_id"`
	SQLText			string		`json:"sql_text"`
	Database		string		`json:"database"`
	Tables			[]string	`json:"tables"`
	SchedulerTaskID		string		`json:"scheduler_task_id,omitempty"`
	ScheduleStatus		string		`json:"schedule_status,omitempty"`
	ScheduleError		string		`json:"schedule_error,omitempty"`
	ScheduleIdentityMatches	bool		`json:"schedule_identity_matches"`
}
```

### DataDashboardSQLDraftRequest

DataDashboardSQLDraftRequest requests one scoped dashboard SQL candidate.

```go
// DataDashboardSQLDraftRequest requests one scoped dashboard SQL candidate.
type DataDashboardSQLDraftRequest struct {
	RequestID	string				`json:"request_id"`
	Question	string				`json:"question"`
	Title		string				`json:"title,omitempty"`
	Schema		[]DataDashboardTableSchema	`json:"schema"`
	PreviousSQL	string				`json:"previous_sql,omitempty"`
	ValidationError	string				`json:"validation_error,omitempty"`
}
```

### DataDashboardSQLDraftResult

DataDashboardSQLDraftResult is the scoped SQL draft returned by Core.

```go
// DataDashboardSQLDraftResult is the scoped SQL draft returned by Core.
type DataDashboardSQLDraftResult struct {
	DashboardID	string	`json:"dashboard_id"`
	SQLText		string	`json:"sql_text"`
	ChartType	string	`json:"chart_type"`
}
```

### DataDashboardTableSchema

DataDashboardTableSchema is role-scoped schema supplied by Backend.

```go
// DataDashboardTableSchema is role-scoped schema supplied by Backend.
type DataDashboardTableSchema struct {
	Name	string					`json:"name"`
	DDL	string					`json:"ddl,omitempty"`
	Columns	[]DataDashboardTableSchemaColumn	`json:"columns,omitempty"`
}
```

### DataDashboardTableSchemaColumn

DataDashboardTableSchemaColumn describes one role-visible table column.

```go
// DataDashboardTableSchemaColumn describes one role-visible table column.
type DataDashboardTableSchemaColumn struct {
	Name		string	`json:"name"`
	Type		string	`json:"type,omitempty"`
	Comment		string	`json:"comment,omitempty"`
	Nullable	bool	`json:"nullable,omitempty"`
	PrimaryKey	bool	`json:"primary_key,omitempty"`
	IsPrimaryKey	bool	`json:"is_primary_key,omitempty"`
	DistinctRatio	float64	`json:"distinct_ratio,omitempty"`
	PopulationScore	float64	`json:"population_score,omitempty"`
}
```

### DataDerivationOption

DataDerivationOption configures CreateDerivation.

```go
// DataDerivationOption configures CreateDerivation.
type DataDerivationOption func(*dataDerivationOptions)
```

### DataDocMapMetadataRequest

DataDocMapMetadataRequest represents a data doc map workitem input.
Re-exported from model/data for convenience.

```go
// DataDocMapMetadataRequest represents a data doc map workitem input.
// Re-exported from model/data for convenience.
type DataDocMapMetadataRequest = data.DataDocMapMetadataRequest
```

### DataShareListOptions

DataShareListOptions controls Data Share filtering and pagination.

```go
// DataShareListOptions controls Data Share filtering and pagination.
type DataShareListOptions struct {
	Keyword		string
	Status		string
	Page		int
	PageSize	int
}
```

### Database

Database represents a database within a catalog.
Re-exported from model/catalog for convenience.

```go
// Database represents a database within a catalog.
// Re-exported from model/catalog for convenience.
type Database = catalog.Database
```

### DatabaseIterator

DatabaseIterator iterates over databases with automatic pagination.

```go
// DatabaseIterator iterates over databases with automatic pagination.
type DatabaseIterator struct {
	service		*DatabaseService
	ctx		context.Context
	workspaceID	string
	catalogID	int64
	buffer		[]*catalog.Database
	index		int
	pageToken	string
	pageSize	int32
	done		bool
	stopped		bool
	err		error
}
```

### DatabaseNode

DatabaseNode represents a database in the tree.

```go
// DatabaseNode represents a database in the tree.
type DatabaseNode struct {
	database	*catalog.Database
	parent		TreeNode
	children	[]TreeNode
}
```

### DatabaseResolveRequest

DatabaseResolveRequest represents a database resolve workitem input.
Re-exported from model/data for convenience.

```go
// DatabaseResolveRequest represents a database resolve workitem input.
// Re-exported from model/data for convenience.
type DatabaseResolveRequest = data.DatabaseResolveRequest
```

### DatabaseResolveResponse

DatabaseResolveResponse represents a database resolve workitem output.
Re-exported from model/data for convenience.

```go
// DatabaseResolveResponse represents a database resolve workitem output.
// Re-exported from model/data for convenience.
type DatabaseResolveResponse = data.DatabaseResolveResponse
```

### DatabaseStatsResponse

DatabaseStatsResponse contains statistics for a database.

```go
// DatabaseStatsResponse contains statistics for a database.
type DatabaseStatsResponse struct {
	TableCount	int64	`json:"table_count"`
	VolumeCount	int64	`json:"volume_count"`
	FileCount	int64	`json:"file_count"`
}
```

### Document

Document represents a generic document payload used by workitems.
Re-exported from model/data for convenience.

```go
// Document represents a generic document payload used by workitems.
// Re-exported from model/data for convenience.
type Document = data.Document
```

### EmbeddingGenerateRequest

EmbeddingGenerateRequest represents an embedding generate workitem input.
Re-exported from model/data for convenience.

```go
// EmbeddingGenerateRequest represents an embedding generate workitem input.
// Re-exported from model/data for convenience.
type EmbeddingGenerateRequest = data.EmbeddingGenerateRequest
```

### EmbeddingGenerateResponse

EmbeddingGenerateResponse represents an embedding generate workitem output.
Re-exported from model/data for convenience.

```go
// EmbeddingGenerateResponse represents an embedding generate workitem output.
// Re-exported from model/data for convenience.
type EmbeddingGenerateResponse = data.EmbeddingGenerateResponse
```

### EmbeddingModelInfo

EmbeddingModelInfo is one available embedding model (flattened across backends).
This is the public read view, so it does not include API keys or endpoints.

```go
// EmbeddingModelInfo is one available embedding model (flattened across backends).
// This is the public read view, so it does not include API keys or endpoints.
type EmbeddingModelInfo struct {
	Model		string	`json:"model"`
	BackendID	int64	`json:"backend_id"`
	BackendName	string	`json:"backend_name"`
	// Type is the optional TaaS Embedding subtype (embedding_text / embedding_multimodal).
	Type	string	`json:"type,omitempty"`
	Dim	int32	`json:"dim,omitempty"`
}
```

### EmbeddingOption

EmbeddingOption configures embedding request.

```go
// EmbeddingOption configures embedding request.
type EmbeddingOption func(*embeddingOptions)
```

### EnsureResourceDisplayMappingsRequest

EnsureResourceDisplayMappingsRequest contains explicit display bindings.

```go
// EnsureResourceDisplayMappingsRequest contains explicit display bindings.
type EnsureResourceDisplayMappingsRequest struct {
	Bindings []ResourceDisplayBinding `json:"bindings"`
}
```

### EnsureResourceDisplayMappingsResponse

EnsureResourceDisplayMappingsResponse is returned after display mappings are ensured.

```go
// EnsureResourceDisplayMappingsResponse is returned after display mappings are ensured.
type EnsureResourceDisplayMappingsResponse struct {
	OK bool `json:"ok"`
}
```

### Error

Error SDK 错误，包装 Proto 定义的错误码

```go
// Error SDK 错误，包装 Proto 定义的错误码
type Error struct {
	Code		common.ErrorCode	`json:"code"`
	Message		string			`json:"message"`
	RequestID	string			`json:"request_id,omitempty"`
	Details		map[string]string	`json:"details,omitempty"`
	Reason		string			`json:"reason,omitempty"`
	Domain		string			`json:"domain,omitempty"`
	Metadata	map[string]string	`json:"metadata,omitempty"`
}
```

### ErrorCode

ErrorCode represents an error code from the API.
Re-exported from model/common for convenience.

```go
// ErrorCode represents an error code from the API.
// Re-exported from model/common for convenience.
type ErrorCode = common.ErrorCode
```

### ExecutionContext

ExecutionContext represents workitem execution context.
Re-exported from model/data for convenience.

```go
// ExecutionContext represents workitem execution context.
// Re-exported from model/data for convenience.
type ExecutionContext = data.ExecutionContext
```

### ExternalWorkItemFunc

ExternalWorkItemFunc is the handler function type for external work items.
It receives a Context for workflow data access and a MowlMessage with execution details.
The handler should process the work item and return the result message or an error.

```go
// ExternalWorkItemFunc is the handler function type for external work items.
// It receives a Context for workflow data access and a MowlMessage with execution details.
// The handler should process the work item and return the result message or an error.
type ExternalWorkItemFunc func(ctx context.Context, wctx WorkItemContext, msg *mowl.MowlMessage) (*mowl.MowlMessage, error)
```

### FieldNodeMapping

FieldNodeMapping describes which work_item nodes reference a given form field.

```go
// FieldNodeMapping describes which work_item nodes reference a given form field.
type FieldNodeMapping struct {
	FieldID	string		`json:"field_id"`
	Nodes	[]FieldNodeRef	`json:"nodes"`
}
```

### FieldNodeRef

FieldNodeRef identifies a work_item node that references a form field.

```go
// FieldNodeRef identifies a work_item node that references a form field.
type FieldNodeRef struct {
	NodeName	string	`json:"node_name"`
	NodeID		string	`json:"node_id"`
}
```

### File

File represents a file within a volume.
Re-exported from model/catalog for convenience.

```go
// File represents a file within a volume.
// Re-exported from model/catalog for convenience.
type File = catalog.File
```

### FileDownloadResponse

FileDownloadResponse holds the response body and metadata from a download.

```go
// FileDownloadResponse holds the response body and metadata from a download.
type FileDownloadResponse struct {
	Body		io.ReadCloser
	Filename	string	// parsed from Content-Disposition, empty if not present
	ContentType	string	// value of Content-Type header
}
```

### FileExecutionSummary

FileExecutionSummary describes a workflow execution associated with a file.

```go
// FileExecutionSummary describes a workflow execution associated with a file.
type FileExecutionSummary struct {
	ExecutionID		string	`json:"execution_id"`
	WorkflowID		string	`json:"workflow_id,omitempty"`
	Status			string	`json:"status"`
	ExecutionMode		string	`json:"execution_mode,omitempty"`
	StartedAt		string	`json:"started_at,omitempty"`
	EndedAt			string	`json:"ended_at,omitempty"`
	CreatedAt		string	`json:"created_at,omitempty"`
	UpdatedAt		string	`json:"updated_at,omitempty"`
	CaseID			string	`json:"case_id,omitempty"`
	SchedulerVisible	bool	`json:"scheduler_visible"`
	CaseStartState		string	`json:"case_start_state,omitempty"`
	Error			string	`json:"error,omitempty"`
}
```

### FileExecutionsResponse

FileExecutionsResponse contains workflow executions associated with a file.

```go
// FileExecutionsResponse contains workflow executions associated with a file.
type FileExecutionsResponse struct {
	Executions	[]FileExecutionSummary	`json:"executions"`
	Total		int			`json:"total"`
}
```

### FileMetadata

FileMetadata represents the file metadata returned by the API.
This is a type alias for the protobuf FileMetadata.

```go
// FileMetadata represents the file metadata returned by the API.
// This is a type alias for the protobuf FileMetadata.
type FileMetadata = catalog.FileMetadata
```

### FileMetadataGetRequest

FileMetadataGetRequest represents a file metadata get workitem input.
Re-exported from model/data for convenience.

```go
// FileMetadataGetRequest represents a file metadata get workitem input.
// Re-exported from model/data for convenience.
type FileMetadataGetRequest = data.FileMetadataGetRequest
```

### FileQueryBuilder

FileQueryBuilder builds file queries for workspace resources.
It provides a fluent interface for constructing file search queries.
Requirements: 5.1, 6.1

```go
// FileQueryBuilder builds file queries for workspace resources.
// It provides a fluent interface for constructing file search queries.
// Requirements: 5.1, 6.1
type FileQueryBuilder struct {
	client		*Client
	ctx		context.Context
	workspaceID	string
	volumeID	*int64
	// File metadata filters
	fileName	*string
	md5		*string
	minRefCount	*int32
	maxRefCount	*int32
	// Pagination
	pageSize	int32
	pageToken	string
}
```

### FileQueryResult

FileQueryResult represents the result of a file query.

```go
// FileQueryResult represents the result of a file query.
type FileQueryResult struct {
	// Files contains the file metadata when querying workspace files
	Files	[]*FileMetadata	`json:"files,omitempty"`
	// VolumeFiles contains the volume file associations when querying volume files
	VolumeFiles	[]*VolumeFile	`json:"volume_files,omitempty"`
	// Total is the total count of matching files
	Total	int64	`json:"total"`
	// NextPageToken is the token for the next page, empty if no more pages
	NextPageToken	string	`json:"next_page_token,omitempty"`
}
```

### FileReadTextRequest

FileReadTextRequest represents input for the disabled file text reader workitem.
Re-exported from model/data for convenience.

```go
// FileReadTextRequest represents input for the disabled file text reader workitem.
// Re-exported from model/data for convenience.
type FileReadTextRequest = data.FileReadTextRequest
```

### FileReadTextResponse

FileReadTextResponse is retained for compatibility with the disabled file text reader workitem.
Re-exported from model/data for convenience.

```go
// FileReadTextResponse is retained for compatibility with the disabled file text reader workitem.
// Re-exported from model/data for convenience.
type FileReadTextResponse = data.FileReadTextResponse
```

### FilesReadDocumentsRequest

FilesReadDocumentsRequest represents input for the disabled documents file reader workitem.
Re-exported from model/data for convenience.

```go
// FilesReadDocumentsRequest represents input for the disabled documents file reader workitem.
// Re-exported from model/data for convenience.
type FilesReadDocumentsRequest = data.FilesReadDocumentsRequest
```

### FilesReadDocumentsResponse

FilesReadDocumentsResponse is retained for compatibility with the disabled documents file reader workitem.
Re-exported from model/data for convenience.

```go
// FilesReadDocumentsResponse is retained for compatibility with the disabled documents file reader workitem.
// Re-exported from model/data for convenience.
type FilesReadDocumentsResponse = data.FilesReadDocumentsResponse
```

### FilesWriteDocumentsRequest

FilesWriteDocumentsRequest represents a files write documents workitem input.
Re-exported from model/data for convenience.

```go
// FilesWriteDocumentsRequest represents a files write documents workitem input.
// Re-exported from model/data for convenience.
type FilesWriteDocumentsRequest = data.FilesWriteDocumentsRequest
```

### FilesWriteDocumentsResponse

FilesWriteDocumentsResponse represents a files write documents workitem output.
Re-exported from model/data for convenience.

```go
// FilesWriteDocumentsResponse represents a files write documents workitem output.
// Re-exported from model/data for convenience.
type FilesWriteDocumentsResponse = data.FilesWriteDocumentsResponse
```

### FormFieldVisibleWhen

FormFieldVisibleWhen controls conditional field visibility.

```go
// FormFieldVisibleWhen controls conditional field visibility.
type FormFieldVisibleWhen struct {
	FieldID	string		`json:"field_id"`
	Value	interface{}	`json:"value"`
}
```

### FormFieldWidget

FormFieldWidget is the UI control a frontend should render for one form field.
The set is deliberately small and closed — any new widget must be added here AND
taught to the agent's system prompt AND implemented on every frontend that renders
forms. Unknown widget strings are invalid form contracts and should be surfaced as
explicit errors instead of being silently rendered as another control.

```go
// FormFieldWidget is the UI control a frontend should render for one form field.
// The set is deliberately small and closed — any new widget must be added here AND
// taught to the agent's system prompt AND implemented on every frontend that renders
// forms. Unknown widget strings are invalid form contracts and should be surfaced as
// explicit errors instead of being silently rendered as another control.
type FormFieldWidget = string
```

### FormLayout

FormLayout describes how to group runtime_fields into tabs.

```go
// FormLayout describes how to group runtime_fields into tabs.
type FormLayout struct {
	Version	string			`json:"version"`
	Groups	[]FormLayoutGroup	`json:"groups"`
}
```

### FormLayoutGroup

FormLayoutGroup is one tab/section in the form layout.

```go
// FormLayoutGroup is one tab/section in the form layout.
type FormLayoutGroup struct {
	GroupID		string		`json:"group_id"`
	Title		string		`json:"title,omitempty"`
	NodeName	string		`json:"node_name,omitempty"`
	NodeID		string		`json:"node_id,omitempty"`
	FieldIDs	[]string	`json:"field_ids"`
}
```

### FulltextDeleteRequest

FulltextDeleteRequest represents a fulltext delete workitem input.
Re-exported from model/data for convenience.

```go
// FulltextDeleteRequest represents a fulltext delete workitem input.
// Re-exported from model/data for convenience.
type FulltextDeleteRequest = data.FulltextDeleteRequest
```

### FulltextDeleteResponse

FulltextDeleteResponse represents a fulltext delete workitem output.
Re-exported from model/data for convenience.

```go
// FulltextDeleteResponse represents a fulltext delete workitem output.
// Re-exported from model/data for convenience.
type FulltextDeleteResponse = data.FulltextDeleteResponse
```

### FulltextDoc

FulltextDoc represents a fulltext document payload.
Re-exported from model/data for convenience.

```go
// FulltextDoc represents a fulltext document payload.
// Re-exported from model/data for convenience.
type FulltextDoc = data.FulltextDoc
```

### FulltextSearchRequest

FulltextSearchRequest represents a fulltext search workitem input.
Re-exported from model/data for convenience.

```go
// FulltextSearchRequest represents a fulltext search workitem input.
// Re-exported from model/data for convenience.
type FulltextSearchRequest = data.FulltextSearchRequest
```

### FulltextSearchResponse

FulltextSearchResponse represents a fulltext search workitem output.
Re-exported from model/data for convenience.

```go
// FulltextSearchResponse represents a fulltext search workitem output.
// Re-exported from model/data for convenience.
type FulltextSearchResponse = data.FulltextSearchResponse
```

### FulltextUpsertRequest

FulltextUpsertRequest represents a fulltext upsert workitem input.
Re-exported from model/data for convenience.

```go
// FulltextUpsertRequest represents a fulltext upsert workitem input.
// Re-exported from model/data for convenience.
type FulltextUpsertRequest = data.FulltextUpsertRequest
```

### FulltextUpsertResponse

FulltextUpsertResponse represents a fulltext upsert workitem output.
Re-exported from model/data for convenience.

```go
// FulltextUpsertResponse represents a fulltext upsert workitem output.
// Re-exported from model/data for convenience.
type FulltextUpsertResponse = data.FulltextUpsertResponse
```

### GarbageCollectionResult

GarbageCollectionResult represents the result of a garbage collection operation.

```go
// GarbageCollectionResult represents the result of a garbage collection operation.
type GarbageCollectionResult struct {
	// OrphanFilesCleaned is the number of orphan files cleaned.
	OrphanFilesCleaned	int	`json:"orphan_files_cleaned"`
	// DeletedVolumesCleaned is the number of deleted volumes cleaned.
	DeletedVolumesCleaned	int	`json:"deleted_volumes_cleaned"`
	// Message is a human-readable message about the operation.
	Message	string	`json:"message"`
}
```

### GetCatalogTreeResponse

GetCatalogTreeResponse represents the permission-filtered Catalog hierarchy.

```go
// GetCatalogTreeResponse represents the permission-filtered Catalog hierarchy.
type GetCatalogTreeResponse = catalog.GetCatalogTreeResponse
```

### GetTableResponse

GetTableResponse represents the response from getting one table.
Re-exported from model/catalog for convenience.

```go
// GetTableResponse represents the response from getting one table.
// Re-exported from model/catalog for convenience.
type GetTableResponse = catalog.GetTableResponse
```

### GetVolumeChildrenResponse

GetVolumeChildrenResponse represents the response from getting volume children.
Re-exported from model/catalog for convenience.

```go
// GetVolumeChildrenResponse represents the response from getting volume children.
// Re-exported from model/catalog for convenience.
type GetVolumeChildrenResponse = catalog.GetVolumeChildrenResponse
```

### GetVolumePathResponse

GetVolumePathResponse represents the response from getting volume path.
Re-exported from model/catalog for convenience.

```go
// GetVolumePathResponse represents the response from getting volume path.
// Re-exported from model/catalog for convenience.
type GetVolumePathResponse = catalog.GetVolumePathResponse
```

### HTTPRoundTripperDecorator

HTTPRoundTripperDecorator wraps the SDK HTTP transport. It receives the
already configured transport and must not inspect or persist request bodies
or credential headers unless explicitly authorized by its caller.

```go
// HTTPRoundTripperDecorator wraps the SDK HTTP transport. It receives the
// already configured transport and must not inspect or persist request bodies
// or credential headers unless explicitly authorized by its caller.
type HTTPRoundTripperDecorator func(http.RoundTripper) http.RoundTripper
```

### I18nPackBuilder

I18nPackBuilder builds i18n language packs for workitem registration.

```go
// I18nPackBuilder builds i18n language packs for workitem registration.
type I18nPackBuilder struct {
	packs		map[int32]string
	defaultLocale	commonpb.Language
}
```

### ImageIndexFileStatus

ImageIndexFileStatus records the image indexing result for one source file.

```go
// ImageIndexFileStatus records the image indexing result for one source file.
type ImageIndexFileStatus struct {
	SourceFileID	string	`json:"source_file_id"`
	Status		string	`json:"status"`
	IndexedImages	int	`json:"indexed_images"`
}
```

### InternalInvokerClient

InternalInvokerClient is a service-to-service client for Mowl registered WorkItem RPCs.

```go
// InternalInvokerClient is a service-to-service client for Mowl registered WorkItem RPCs.
type InternalInvokerClient struct {
	endpoint	string
	apiKey		string
	conn		*grpc.ClientConn
	mowlClient	mowl.MowlServiceClient
}
```

### InvokeOption

InvokeOption configures dynamic service invocation.

```go
// InvokeOption configures dynamic service invocation.
type InvokeOption func(*invokeOptions)
```

### InvokeResult

InvokeResult represents the final result of a dynamic service invocation (oneshot mode).

```go
// InvokeResult represents the final result of a dynamic service invocation (oneshot mode).
type InvokeResult struct {
	CaseID	string
	Status	string
	Result	string
	Error	string
}
```

### JSONRepairRequest

JSONRepairRequest represents a JSON repair workitem input.
Re-exported from model/data for convenience.

```go
// JSONRepairRequest represents a JSON repair workitem input.
// Re-exported from model/data for convenience.
type JSONRepairRequest = data.JSONRepairRequest
```

### JSONRepairResponse

JSONRepairResponse represents a JSON repair workitem output.
Re-exported from model/data for convenience.

```go
// JSONRepairResponse represents a JSON repair workitem output.
// Re-exported from model/data for convenience.
type JSONRepairResponse = data.JSONRepairResponse
```

### KeywordRetrievalRequest

KeywordRetrievalRequest represents a keyword retrieval workitem input.
Re-exported from model/data for convenience.

```go
// KeywordRetrievalRequest represents a keyword retrieval workitem input.
// Re-exported from model/data for convenience.
type KeywordRetrievalRequest = data.KeywordRetrievalRequest
```

### KeywordRetrievalResponse

KeywordRetrievalResponse represents a keyword retrieval workitem output.
Re-exported from model/data for convenience.

```go
// KeywordRetrievalResponse represents a keyword retrieval workitem output.
// Re-exported from model/data for convenience.
type KeywordRetrievalResponse = data.KeywordRetrievalResponse
```

### LatestMessageOption

LatestMessageOption configures latest message ID options.

```go
// LatestMessageOption configures latest message ID options.
type LatestMessageOption func(*latestMessageOptions)
```

### ListBlockRevisionsResponse

ListBlockRevisionsResponse contains output block revisions for one block.

```go
// ListBlockRevisionsResponse contains output block revisions for one block.
type ListBlockRevisionsResponse struct {
	Revisions []*mowlpb.OutputBlockRevision `json:"revisions"`
}
```

### ListCDHColumnsResponse

ListCDHColumnsResponse represents the response from listing CDH columns.
Re-exported from model/catalog for convenience.

```go
// ListCDHColumnsResponse represents the response from listing CDH columns.
// Re-exported from model/catalog for convenience.
type ListCDHColumnsResponse = catalog.ListCDHColumnsResponse
```

### ListCDHConfigsResponse

ListCDHConfigsResponse represents the response from listing CDH configs.
Re-exported from model/catalog for convenience.

```go
// ListCDHConfigsResponse represents the response from listing CDH configs.
// Re-exported from model/catalog for convenience.
type ListCDHConfigsResponse = catalog.ListCDHConfigsResponse
```

### ListCDHDatabasesResponse

ListCDHDatabasesResponse represents the response from listing CDH databases.
Re-exported from model/catalog for convenience.

```go
// ListCDHDatabasesResponse represents the response from listing CDH databases.
// Re-exported from model/catalog for convenience.
type ListCDHDatabasesResponse = catalog.ListCDHDatabasesResponse
```

### ListCDHTablesResponse

ListCDHTablesResponse represents the response from listing CDH tables.
Re-exported from model/catalog for convenience.

```go
// ListCDHTablesResponse represents the response from listing CDH tables.
// Re-exported from model/catalog for convenience.
type ListCDHTablesResponse = catalog.ListCDHTablesResponse
```

### ListCasesOption

ListCasesOption is a function that configures list cases options.

```go
// ListCasesOption is a function that configures list cases options.
type ListCasesOption func(*listCasesOptions)
```

### ListCatalogSummariesResponse

ListCatalogSummariesResponse contains paginated Catalog homepage summaries.

```go
// ListCatalogSummariesResponse contains paginated Catalog homepage summaries.
type ListCatalogSummariesResponse = catalog.ListCatalogSummariesResponse
```

### ListCatalogsResponse

ListCatalogsResponse represents the response from listing catalogs.
Re-exported from model/catalog for convenience.

```go
// ListCatalogsResponse represents the response from listing catalogs.
// Re-exported from model/catalog for convenience.
type ListCatalogsResponse = catalog.ListCatalogsResponse
```

### ListCustomOperatorsRequest

ListCustomOperatorsRequest contains filters for listing custom operators.

```go
// ListCustomOperatorsRequest contains filters for listing custom operators.
type ListCustomOperatorsRequest struct {
	PageSize	int32
	PageToken	string
	Enabled		*bool
	Language	string
	Kind		string
	NodeID		string
	Identifier	string
	Version		string
	BaseNodeID	string
}
```

### ListCustomOperatorsResponse

ListCustomOperatorsResponse is the response returned by CustomOperatorService.List.

```go
// ListCustomOperatorsResponse is the response returned by CustomOperatorService.List.
type ListCustomOperatorsResponse = catalog.ListCustomOperatorsResponse
```

### ListDatabaseChildrenResponse

ListDatabaseChildrenResponse contains lightweight direct Database children.

```go
// ListDatabaseChildrenResponse contains lightweight direct Database children.
type ListDatabaseChildrenResponse = catalog.ListDatabaseChildrenResponse
```

### ListDatabasesResponse

ListDatabasesResponse represents the response from listing databases.
Re-exported from model/catalog for convenience.

```go
// ListDatabasesResponse represents the response from listing databases.
// Re-exported from model/catalog for convenience.
type ListDatabasesResponse = catalog.ListDatabasesResponse
```

### ListEmbeddingModelsResponse

ListEmbeddingModelsResponse is the flat model list returned by ListEmbeddingModels.

```go
// ListEmbeddingModelsResponse is the flat model list returned by ListEmbeddingModels.
type ListEmbeddingModelsResponse struct {
	Models []EmbeddingModelInfo `json:"models"`
}
```

### ListFilesOption

ListFilesOption is a function that configures list files options.

```go
// ListFilesOption is a function that configures list files options.
type ListFilesOption func(*listFilesOptions)
```

### ListFinalOutputArtifactsResponse

ListFinalOutputArtifactsResponse lists final output artifacts for one
case-scoped root asset.

```go
// ListFinalOutputArtifactsResponse lists final output artifacts for one
// case-scoped root asset.
type ListFinalOutputArtifactsResponse struct {
	OutputArtifacts []*mowlpb.FinalOutputArtifact `json:"output_artifacts"`
}
```

### ListMessagesOption

ListMessagesOption configures list messages options.

```go
// ListMessagesOption configures list messages options.
type ListMessagesOption func(*listMessagesOptions)
```

### ListModelsResponse

ListModelsResponse is the flat workspace model list returned by ListModels.

```go
// ListModelsResponse is the flat workspace model list returned by ListModels.
type ListModelsResponse struct {
	Models []ModelInfo `json:"models"`
}
```

### ListOption

ListOption is a function that configures list options.

```go
// ListOption is a function that configures list options.
type ListOption func(*listOptions)
```

### ListOutputBlocksResponse

ListOutputBlocksResponse contains block refs for a final artifact or node output target.

```go
// ListOutputBlocksResponse contains block refs for a final artifact or node output target.
type ListOutputBlocksResponse struct {
	Blocks []*mowlpb.BlockRef `json:"blocks"`
}
```

### ListSessionsOption

ListSessionsOption configures list sessions options.

```go
// ListSessionsOption configures list sessions options.
type ListSessionsOption func(*listSessionsOptions)
```

### ListTablesResponse

ListTablesResponse represents the response from listing tables.
Re-exported from model/catalog for convenience.

```go
// ListTablesResponse represents the response from listing tables.
// Re-exported from model/catalog for convenience.
type ListTablesResponse = catalog.ListTablesResponse
```

### ListTagsOption

ListTagsOption configures list tags options.

```go
// ListTagsOption configures list tags options.
type ListTagsOption func(*listTagsOptions)
```

### ListTasksOption

ListTasksOption is a function that configures list tasks options.

```go
// ListTasksOption is a function that configures list tasks options.
type ListTasksOption func(*listTasksOptions)
```

### ListUpgradeTenantTasksOptions

ListUpgradeTenantTasksOptions controls filtering and pagination for tenant upgrade tasks.

```go
// ListUpgradeTenantTasksOptions controls filtering and pagination for tenant upgrade tasks.
type ListUpgradeTenantTasksOptions struct {
	State		[]string
	WorkspaceID	string
	UpgradeID	uint64
	Limit		int
	Offset		int
}
```

### ListVolumeFilesDetailResponse

ListVolumeFilesDetailResponse represents the response for listing volume files with full file metadata.

```go
// ListVolumeFilesDetailResponse represents the response for listing volume files with full file metadata.
type ListVolumeFilesDetailResponse struct {
	Items		[]*VolumeFileDetail	`json:"items"`
	Total		int64			`json:"total"`
	NextPageToken	string			`json:"next_page_token,omitempty"`
}
```

### ListVolumeFilesResponse

ListVolumeFilesResponse represents the response for listing volume files.

```go
// ListVolumeFilesResponse represents the response for listing volume files.
type ListVolumeFilesResponse struct {
	Items		[]*VolumeFile	`json:"items"`
	Total		int64		`json:"total"`
	NextPageToken	string		`json:"next_page_token,omitempty"`
}
```

### ListVolumesResponse

ListVolumesResponse represents the response from listing volumes.
Re-exported from model/catalog for convenience.

```go
// ListVolumesResponse represents the response from listing volumes.
// Re-exported from model/catalog for convenience.
type ListVolumesResponse = catalog.ListVolumesResponse
```

### ListWorkflowsOption

ListWorkflowsOption is a function that configures list workflows options.

```go
// ListWorkflowsOption is a function that configures list workflows options.
type ListWorkflowsOption func(*listWorkflowsOptions)
```

### ListWorkspaceModelOption

ListWorkspaceModelOption configures workspace model listing.

```go
// ListWorkspaceModelOption configures workspace model listing.
type ListWorkspaceModelOption func(*listWorkspaceModelOptions)
```

### ListWorkspacesResponse

ListWorkspacesResponse represents the response from listing workspaces.
Re-exported from model/workspace for convenience.

```go
// ListWorkspacesResponse represents the response from listing workspaces.
// Re-exported from model/workspace for convenience.
type ListWorkspacesResponse = workspace.ListWorkspacesResponse
```

### Logger

Logger defines the interface for logging.

```go
// Logger defines the interface for logging.
type Logger interface {
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}
```

### MemoryGovernanceExecutionRequest

MemoryGovernanceExecutionRequest is the trusted Backend runtime payload for
one session-triggered memory Workflow Run.

```go
// MemoryGovernanceExecutionRequest is the trusted Backend runtime payload for
// one session-triggered memory Workflow Run.
type MemoryGovernanceExecutionRequest struct {
	RunID			string		`json:"run_id"`
	WorkflowVersionID	string		`json:"workflow_version_id"`
	ConnectorID		string		`json:"connector_id,omitempty"`
	PlatformManaged		bool		`json:"platform_managed,omitempty"`
	Data			map[string]any	`json:"data"`
}
```

### MemoryGovernanceExecutionResponse

MemoryGovernanceExecutionResponse identifies the standard Workflow
Execution and its Mowl Task/Case created for one immutable Run.

```go
// MemoryGovernanceExecutionResponse identifies the standard Workflow
// Execution and its Mowl Task/Case created for one immutable Run.
type MemoryGovernanceExecutionResponse struct {
	ExecutionID	string	`json:"execution_id"`
	TaskID		string	`json:"task_id"`
	CaseID		string	`json:"case_id"`
}
```

### MimeRouterRequest

MimeRouterRequest represents a MIME router workitem input.
Re-exported from model/data for convenience.

```go
// MimeRouterRequest represents a MIME router workitem input.
// Re-exported from model/data for convenience.
type MimeRouterRequest = data.MimeRouterRequest
```

### MimeRouterResponse

MimeRouterResponse represents a MIME router workitem output.
Re-exported from model/data for convenience.

```go
// MimeRouterResponse represents a MIME router workitem output.
// Re-exported from model/data for convenience.
type MimeRouterResponse = data.MimeRouterResponse
```

### ModelInfo

ModelInfo is one available LLM model (flattened across backends).
This is the public read view for workflow-form dropdowns.

```go
// ModelInfo is one available LLM model (flattened across backends).
// This is the public read view for workflow-form dropdowns.
type ModelInfo struct {
	Model		string	`json:"model"`
	BackendID	int64	`json:"backend_id"`
	BackendName	string	`json:"backend_name"`
	ModelType	string	`json:"model_type,omitempty"`
}
```

### ModifyResponseOption

ModifyResponseOption configures modify response options.

```go
// ModifyResponseOption configures modify response options.
type ModifyResponseOption func(*modifyResponseOptions)
```

### MoveFilesRequest

MoveFilesRequest represents the request to move files between volumes.

```go
// MoveFilesRequest represents the request to move files between volumes.
type MoveFilesRequest struct {
	TargetVolumeID	int64		`json:"target_volume_id"`
	FileIDs		[]string	`json:"file_ids"`
}
```

### NodeNotifyHandler

NodeNotifyHandler is the handler function type for node-level notifications.

```go
// NodeNotifyHandler is the handler function type for node-level notifications.
type NodeNotifyHandler func(ctx context.Context, notification *mowl.NodeNotification)
```

### NodeType

NodeType represents the type of a tree node.

```go
// NodeType represents the type of a tree node.
type NodeType int
```

### NotificationBuilder

NotificationBuilder helps build NotificationConfig with a fluent API.

```go
// NotificationBuilder helps build NotificationConfig with a fluent API.
type NotificationBuilder struct {
	config *mowl.NotificationConfig
}
```

### NotificationOption

NotificationOption configures notification handler filtering.

```go
// NotificationOption configures notification handler filtering.
type NotificationOption func(*notificationOptions)
```

### Option

Option is a function that configures the client.

```go
// Option is a function that configures the client.
type Option func(*config)
```

### ParseOption

ParseOption 配置解析请求的可选参数

```go
// ParseOption 配置解析请求的可选参数
type ParseOption func(*parseOptions)
```

### ParseResultOption

ParseResultOption configures a protobuf ParseResult in a friendly way.

```go
// ParseResultOption configures a protobuf ParseResult in a friendly way.
type ParseResultOption func(*catalogpb.ParseResult)
```

### ParseResultParser

ParseResultParser aliases protobuf enum for developer-friendly usage.

```go
// ParseResultParser aliases protobuf enum for developer-friendly usage.
type ParseResultParser = catalogpb.ParseResultParser
```

### ParsedManifestOption

ParsedManifestOption configures UpsertParsedManifest.

```go
// ParsedManifestOption configures UpsertParsedManifest.
type ParsedManifestOption func(*parsedManifestOptions)
```

### ParserConvertOption

ParserConvertOption configures convert options on ParserService.

```go
// ParserConvertOption configures convert options on ParserService.
type ParserConvertOption func(*parserConvertOptions)
```

### PutRouterConfigOption

PutRouterConfigOption configures put router config options.

```go
// PutRouterConfigOption configures put router config options.
type PutRouterConfigOption func(*putRouterConfigOptions)
```

### QueryBuilder

QueryBuilder builds tree queries for workspace resources.

```go
// QueryBuilder builds tree queries for workspace resources.
type QueryBuilder struct {
	client			*Client
	ctx			context.Context
	workspaceID		string
	catalogID		*int64
	databaseID		*int64
	withCatalogs		bool
	withDatabases		bool
	withVolumes		bool
	withVolumeChildren	bool	// whether to recursively fetch child volumes
	concurrency		int
}
```

### RawResponse

RawResponse is an unmodified HTTP response from Catalog. The caller owns
Body and must close it.

```go
// RawResponse is an unmodified HTTP response from Catalog. The caller owns
// Body and must close it.
type RawResponse struct {
	StatusCode	int
	Header		http.Header
	Body		io.ReadCloser
}
```

### RegisterLineageEdgeProvenance

RegisterLineageEdgeProvenance identifies the producer slot for a lineage edge kind.

```go
// RegisterLineageEdgeProvenance identifies the producer slot for a lineage edge kind.
type RegisterLineageEdgeProvenance struct {
	ProducerWorkitemID	string	`json:"producer_workitem_id,omitempty"`
	LogicalSlot		string	`json:"logical_slot,omitempty"`
}
```

### RegisterLineageRequest

RegisterLineageRequest registers a complete source-to-artifact lineage chain.

```go
// RegisterLineageRequest registers a complete source-to-artifact lineage chain.
type RegisterLineageRequest struct {
	SourceFileID			string						`json:"source_file_id,omitempty"`
	SourceFileName			string						`json:"source_file_name,omitempty"`
	ParsedFileID			string						`json:"parsed_file_id,omitempty"`
	DerivedFileIDs			[]string					`json:"derived_file_ids,omitempty"`
	VectorTable			string						`json:"vector_table,omitempty"`
	EmbeddingModel			string						`json:"embedding_model,omitempty"`
	SemanticModelRefVectorTable	string						`json:"semantic_model_ref_vector_table,omitempty"`
	ImageVectorTable		string						`json:"image_vector_table,omitempty"`
	ImageEmbeddingModel		string						`json:"image_embedding_model,omitempty"`
	ImageEmbeddingBackendID		string						`json:"image_embedding_backend_id,omitempty"`
	ImageEmbeddingDimension		int						`json:"image_embedding_dimension,omitempty"`
	ImagePreprocessVersion		string						`json:"image_preprocess_version,omitempty"`
	ImageDistanceMetric		string						`json:"image_distance_metric,omitempty"`
	ImageIndexFileStatus		*ImageIndexFileStatus				`json:"image_index_file_status,omitempty"`
	OutputFileID			string						`json:"output_file_id,omitempty"`
	VolumeID			int64						`json:"volume_id,omitempty"`
	Source				string						`json:"source,omitempty"`
	Runtime				RegisterLineageRuntime				`json:"runtime,omitempty"`
	EdgeProvenance			map[string]RegisterLineageEdgeProvenance	`json:"edge_provenance,omitempty"`
}
```

### RegisterLineageResponse

RegisterLineageResponse contains the assets and derivations created for a lineage chain.

```go
// RegisterLineageResponse contains the assets and derivations created for a lineage chain.
type RegisterLineageResponse struct {
	SourceAsset		*catalog.DataAsset		`json:"source_asset,omitempty"`
	ParsedAsset		*catalog.DataAsset		`json:"parsed_asset,omitempty"`
	VectorAsset		*catalog.DataAsset		`json:"vector_asset,omitempty"`
	ImageVectorAsset	*catalog.DataAsset		`json:"image_vector_asset,omitempty"`
	OutputAsset		*catalog.DataAsset		`json:"output_asset,omitempty"`
	Derivations		[]*catalog.DataDerivation	`json:"derivations,omitempty"`
	Manifest		*catalog.ParsedManifest		`json:"manifest,omitempty"`
}
```

### RegisterLineageRuntime

RegisterLineageRuntime identifies the workflow execution that produced a lineage edge.

```go
// RegisterLineageRuntime identifies the workflow execution that produced a lineage edge.
type RegisterLineageRuntime struct {
	CaseID			string	`json:"case_id,omitempty"`
	RecordedByWorkitemID	string	`json:"recorded_by_workitem_id,omitempty"`
	ParallelIndex		int32	`json:"parallel_index,omitempty"`
}
```

### RemoveFilesRequest

RemoveFilesRequest represents the request to remove files from a volume.

```go
// RemoveFilesRequest represents the request to remove files from a volume.
type RemoveFilesRequest struct {
	FileIDs []string `json:"file_ids"`
}
```

### RepairContextPolicy

RepairContextPolicy controls list limits before sending context to LLM.

```go
// RepairContextPolicy controls list limits before sending context to LLM.
type RepairContextPolicy struct {
	MaxGoalChars		int
	MaxDSLChars		int
	MaxWorkItems		int
	MaxSchemaChars		int
	MaxDescriptionChars	int
	MaxUserIntakeItems	int
	MaxUserIntakeChars	int
	MaxRunSchemaChars	int
	MaxRunTemplateChars	int
	MaxRunTestCases		int
	MaxRunAssertionChars	int
	MaxCompileDiagnostics	int
	MaxDiagnosticChars	int
	MaxRuntimeFailures	int
	MaxRuntimeMessageChars	int
	MaxTestResults		int
	MaxTestPayloadChars	int
	MaxAttemptHistory	int
}
```

### RepairContextValidationError

RepairContextValidationError collects all contract validation failures.

```go
// RepairContextValidationError collects all contract validation failures.
type RepairContextValidationError struct {
	Issues []string
}
```

### RerunCancelResponse

RerunCancelResponse acknowledges cancellation of a rerun branch.

```go
// RerunCancelResponse acknowledges cancellation of a rerun branch.
type RerunCancelResponse struct {
	OK bool `json:"ok"`
}
```

### RerunCreateResponse

RerunCreateResponse identifies a newly created rerun branch.

```go
// RerunCreateResponse identifies a newly created rerun branch.
type RerunCreateResponse struct {
	RerunID			string	`json:"rerun_id"`
	BranchID		string	`json:"branch_id"`
	CaseID			string	`json:"case_id"`
	NewBranchID		string	`json:"new_branch_id"`
	NewCaseID		string	`json:"new_case_id"`
	NewWorkflowExecutionID	string	`json:"new_workflow_execution_id"`
	Status			string	`json:"status"`
}
```

### RerunPreviewResponse

RerunPreviewResponse identifies the immutable contract prepared for create.

```go
// RerunPreviewResponse identifies the immutable contract prepared for create.
type RerunPreviewResponse struct {
	OK			bool	`json:"ok"`
	RerunContractHash	string	`json:"rerun_contract_hash"`
}
```

### RerunStartResponse

RerunStartResponse identifies the rerun branch that was submitted to Mowl.

```go
// RerunStartResponse identifies the rerun branch that was submitted to Mowl.
type RerunStartResponse struct {
	OK			bool	`json:"ok"`
	RerunID			string	`json:"rerun_id"`
	BranchID		string	`json:"branch_id"`
	CaseID			string	`json:"case_id"`
	NewBranchID		string	`json:"new_branch_id"`
	NewCaseID		string	`json:"new_case_id"`
	NewWorkflowExecutionID	string	`json:"new_workflow_execution_id"`
	Status			string	`json:"status"`
}
```

### ResolveVolumeFileRootsRequest

ResolveVolumeFileRootsRequest identifies Catalog files whose trusted
containment roots are required for a subsequent authorization check.

```go
// ResolveVolumeFileRootsRequest identifies Catalog files whose trusted
// containment roots are required for a subsequent authorization check.
type ResolveVolumeFileRootsRequest struct {
	FileIDs []string `json:"file_ids"`
}
```

### ResolveVolumeFileRootsResponse

ResolveVolumeFileRootsResponse contains distinct canonical root Volume IDs.
It is metadata only and does not imply that the caller may access a root.

```go
// ResolveVolumeFileRootsResponse contains distinct canonical root Volume IDs.
// It is metadata only and does not imply that the caller may access a root.
type ResolveVolumeFileRootsResponse struct {
	RootVolumeIDs []string `json:"root_volume_ids"`
}
```

### ResourceDisplayBinding

ResourceDisplayBinding binds a resource field to producer-owned display metadata.

```go
// ResourceDisplayBinding binds a resource field to producer-owned display metadata.
type ResourceDisplayBinding struct {
	ResourceType	string	`json:"resource_type"`
	ResourceID	string	`json:"resource_id"`
	Field		string	`json:"field"`
	DisplayOwner	string	`json:"display_owner"`
	DisplayKey	string	`json:"display_key"`
	DefaultText	string	`json:"default_text"`
}
```

### RetryTenantTaskOptions

RetryTenantTaskOptions controls an explicit tenant task retry.

```go
// RetryTenantTaskOptions controls an explicit tenant task retry.
type RetryTenantTaskOptions struct {
	OperatorID	string
	// Force replays a terminal task. The Catalog API requires OperatorID when
	// Force is true, rejects Running/RetryRequested tasks, and fails closed
	// while a detached operation retains the task claim.
	Force	bool
}
```

### RichConvertRequest

RichConvertRequest represents a rich convert workitem input.
Re-exported from model/data for convenience.

```go
// RichConvertRequest represents a rich convert workitem input.
// Re-exported from model/data for convenience.
type RichConvertRequest = data.RichConvertRequest
```

### RichConvertResponse

RichConvertResponse represents a rich convert workitem output.
Re-exported from model/data for convenience.

```go
// RichConvertResponse represents a rich convert workitem output.
// Re-exported from model/data for convenience.
type RichConvertResponse = data.RichConvertResponse
```

### RunSQLRequest

RunSQLRequest represents a runsql workitem request.
Re-exported from model/data for convenience.

```go
// RunSQLRequest represents a runsql workitem request.
// Re-exported from model/data for convenience.
type RunSQLRequest = data.RunSQLRequest
```

### RunSQLResponse

RunSQLResponse represents a runsql workitem response.
Re-exported from model/data for convenience.

```go
// RunSQLResponse represents a runsql workitem response.
// Re-exported from model/data for convenience.
type RunSQLResponse = data.RunSQLResponse
```

### RuntimeConfigContractLookup

RuntimeConfigContractLookup describes a version-aware runtime config contract lookup.

```go
// RuntimeConfigContractLookup describes a version-aware runtime config contract lookup.
type RuntimeConfigContractLookup struct {
	NodeID	string
	Version	string
}
```

### RuntimeConfigContractProvider

RuntimeConfigContractProvider returns the runtime_config_contract for a given node_id.

```go
// RuntimeConfigContractProvider returns the runtime_config_contract for a given node_id.
type RuntimeConfigContractProvider func(nodeID string) []*RuntimeConfigParam
```

### RuntimeConfigContractProviderV2

RuntimeConfigContractProviderV2 returns the runtime_config_contract for a given node_id and version.

```go
// RuntimeConfigContractProviderV2 returns the runtime_config_contract for a given node_id and version.
type RuntimeConfigContractProviderV2 func(RuntimeConfigContractLookup) []*RuntimeConfigParam
```

### RuntimeConfigDependency

RuntimeConfigDependency declares when a runtime config parameter depends on another parameter.

```go
// RuntimeConfigDependency declares when a runtime config parameter depends on another parameter.
type RuntimeConfigDependency = mowl.RuntimeConfigDependency
```

### RuntimeConfigDiagnostic

RuntimeConfigDiagnostic is a validation issue found by ValidateRuntimeConfigContract.

```go
// RuntimeConfigDiagnostic is a validation issue found by ValidateRuntimeConfigContract.
type RuntimeConfigDiagnostic struct {
	// Severity: "error" for exposure=always violations, "warning" for exposure=ask.
	Severity	string	`json:"severity"`
	// NodeID is the workitem that declares the parameter.
	NodeID	string	`json:"node_id"`
	// NodeName is the workitem instance name in the DSL.
	NodeName	string	`json:"node_name"`
	// ParamPath is the parameter's path (e.g. "options.enable_parser_pipeline").
	ParamPath	string	`json:"param_path"`
	// Message describes what's wrong.
	Message	string	`json:"message"`
}
```

### RuntimeConfigExposure

RuntimeConfigExposure controls when a runtime config parameter appears in generated forms.

```go
// RuntimeConfigExposure controls when a runtime config parameter appears in generated forms.
type RuntimeConfigExposure = string
```

### RuntimeConfigParam

RuntimeConfigParam describes a workflow runtime parameter surfaced from WorkItem metadata.

```go
// RuntimeConfigParam describes a workflow runtime parameter surfaced from WorkItem metadata.
type RuntimeConfigParam = mowl.RuntimeConfigParam
```

### RuntimeConfigValidationOption

RuntimeConfigValidationOption customizes runtime config validation.

```go
// RuntimeConfigValidationOption customizes runtime config validation.
type RuntimeConfigValidationOption func(*runtimeConfigValidationOptions)
```

### RuntimeContext

RuntimeContext exposes server-injected runtime identifiers for the current
WorkItem request. CaseId, WorkitemId, and ParallelIndex remain as compatibility
aliases for the canonical ExecutionContext fields.

```go
// RuntimeContext exposes server-injected runtime identifiers for the current
// WorkItem request. CaseId, WorkitemId, and ParallelIndex remain as compatibility
// aliases for the canonical ExecutionContext fields.
type RuntimeContext struct {
	WorkspaceID		string	`json:"workspace_id,omitempty"`
	UserID			string	`json:"user_id,omitempty"`
	EffectiveRoleID		string	`json:"effective_role_id,omitempty"`
	CaseId			string	`json:"case_id,omitempty"`
	CaseID			string	`json:"-"`
	TaskID			string	`json:"task_id,omitempty"`
	BranchID		string	`json:"branch_id,omitempty"`
	WorkflowExecutionID	string	`json:"workflow_execution_id,omitempty"`
	WorkflowVersionID	string	`json:"workflow_version_id,omitempty"`
	RerunID			string	`json:"rerun_id,omitempty"`
	NodeExecutionID		string	`json:"node_execution_id,omitempty"`
	RuntimeWorkitemTaskID	string	`json:"runtime_workitem_task_id,omitempty"`
	WorkitemId		string	`json:"workitem_id,omitempty"`
	WorkflowNodeInstanceID	string	`json:"workflow_node_instance_id,omitempty"`
	WorkitemTypeID		string	`json:"workitem_type_id,omitempty"`
	ActivationID		string	`json:"activation_id,omitempty"`
	ActivationSequence	int32	`json:"activation_sequence,omitempty"`
	ParallelIndex		int32	`json:"parallel_index,omitempty"`
	ParallelTotal		int32	`json:"parallel_total,omitempty"`
	IdempotencyKey		string	`json:"idempotency_key,omitempty"`
}
```

### SchemaBuilder

SchemaBuilder 用于构建 JSON Schema

```go
// SchemaBuilder 用于构建 JSON Schema
type SchemaBuilder struct {
	schema map[string]interface{}
}
```

### SemanticEntry

SemanticEntry represents a semantic model entry.

```go
// SemanticEntry represents a semantic model entry.
type SemanticEntry struct {
	ID		int64		`json:"id"`
	ModelID		int64		`json:"model_id"`
	Kind		string		`json:"kind"`
	Key		string		`json:"key"`
	Tables		[]string	`json:"tables,omitempty"`
	Spec		json.RawMessage	`json:"spec"`
	CreatedBy	string		`json:"created_by,omitempty"`
	UpdatedBy	string		`json:"updated_by,omitempty"`
	CreatedAt	int64		`json:"created_at"`
	UpdatedAt	int64		`json:"updated_at"`
}
```

### SemanticEntryListResponse

SemanticEntryListResponse is the list semantic entries response.

```go
// SemanticEntryListResponse is the list semantic entries response.
type SemanticEntryListResponse struct {
	Items		[]*SemanticEntry	`json:"items"`
	Total		int64			`json:"total"`
	NextPageToken	string			`json:"next_page_token,omitempty"`
}
```

### SemanticEntryUpsertRequest

SemanticEntryUpsertRequest is used by semantic entry create/update.

```go
// SemanticEntryUpsertRequest is used by semantic entry create/update.
type SemanticEntryUpsertRequest struct {
	Kind	string		`json:"kind"`
	Key	string		`json:"key"`
	Tables	[]string	`json:"tables,omitempty"`
	Spec	json.RawMessage	`json:"spec"`
}
```

### SemanticModel

SemanticModel represents a semantic model.

```go
// SemanticModel represents a semantic model.
type SemanticModel struct {
	ID		int64		`json:"id"`
	WorkspaceID	string		`json:"workspace_id"`
	Name		string		`json:"name"`
	Description	string		`json:"description,omitempty"`
	Tables		json.RawMessage	`json:"tables"`
	Files		json.RawMessage	`json:"files,omitempty"`
	CreatedBy	string		`json:"created_by,omitempty"`
	UpdatedBy	string		`json:"updated_by,omitempty"`
	CreatedAt	int64		`json:"created_at"`
	UpdatedAt	int64		`json:"updated_at"`
}
```

### SemanticModelExportResponse

SemanticModelExportResponse is the export response.

```go
// SemanticModelExportResponse is the export response.
type SemanticModelExportResponse struct {
	Model	*SemanticModel		`json:"model"`
	Entries	[]*SemanticEntry	`json:"entries"`
}
```

### SemanticModelImportBatchResponse

SemanticModelImportBatchResponse is the batch import response.

```go
// SemanticModelImportBatchResponse is the batch import response.
type SemanticModelImportBatchResponse struct {
	Items	[]*SemanticModel	`json:"items"`
	Total	int64			`json:"total"`
}
```

### SemanticModelImportRequest

SemanticModelImportRequest is used by semantic model import.

```go
// SemanticModelImportRequest is used by semantic model import.
type SemanticModelImportRequest struct {
	Name		string				`json:"name"`
	Description	string				`json:"description,omitempty"`
	Tables		json.RawMessage			`json:"tables"`
	Files		json.RawMessage			`json:"files,omitempty"`
	Entries		[]SemanticEntryUpsertRequest	`json:"entries,omitempty"`
}
```

### SemanticModelListResponse

SemanticModelListResponse is the list semantic models response.

```go
// SemanticModelListResponse is the list semantic models response.
type SemanticModelListResponse struct {
	Items		[]*SemanticModel	`json:"items"`
	Total		int64			`json:"total"`
	NextPageToken	string			`json:"next_page_token,omitempty"`
}
```

### SemanticModelMutationResponse

SemanticModelMutationResponse is the mutation response for update operations.

```go
// SemanticModelMutationResponse is the mutation response for update operations.
type SemanticModelMutationResponse struct {
	Updated	bool	`json:"updated,omitempty"`
	Deleted	bool	`json:"deleted,omitempty"`
}
```

### SemanticModelTagListResponse

SemanticModelTagListResponse is the list semantic model tags response.

```go
// SemanticModelTagListResponse is the list semantic model tags response.
type SemanticModelTagListResponse struct {
	Items []SemanticModelTagStat `json:"items"`
}
```

### SemanticModelTagStat

SemanticModelTagStat is an aggregated semantic model tag count.

```go
// SemanticModelTagStat is an aggregated semantic model tag count.
type SemanticModelTagStat struct {
	Tag	string	`json:"tag"`
	Count	int64	`json:"count"`
}
```

### SemanticModelUpsertRequest

SemanticModelUpsertRequest is used by semantic model create/update.

```go
// SemanticModelUpsertRequest is used by semantic model create/update.
type SemanticModelUpsertRequest struct {
	Name		string		`json:"name"`
	Description	string		`json:"description,omitempty"`
	Tables		json.RawMessage	`json:"tables,omitempty"`
	Files		json.RawMessage	`json:"files,omitempty"`

	knowledgeBaseDatabaseDisplayName	*knowledgeBaseDatabaseDisplayNameRequest
}
```

### SemanticModelValidateResponse

SemanticModelValidateResponse is the validate response.

```go
// SemanticModelValidateResponse is the validate response.
type SemanticModelValidateResponse struct {
	Valid bool `json:"valid"`
}
```

### ServiceAccountCatalogOptions

ServiceAccountCatalogOptions carries the UC-issued actor assertion for one
server-to-server catalog request. It is never browser input and is paired
with a catalog-provisioner Bearer token, not a service-account data token.

It is intentionally a method argument for the future service-account
facade, rather than client configuration, so the assertion is never shared
by later requests.

```go
// ServiceAccountCatalogOptions carries the UC-issued actor assertion for one
// server-to-server catalog request. It is never browser input and is paired
// with a catalog-provisioner Bearer token, not a service-account data token.
//
// It is intentionally a method argument for the future service-account
// facade, rather than client configuration, so the assertion is never shared
// by later requests.
type ServiceAccountCatalogOptions struct {
	ActorAssertion string
}
```

### ServiceAccountError

ServiceAccountError is a typed v1 error. Code is the stable service-account
code; callers must not branch on the safe display message.

```go
// ServiceAccountError is a typed v1 error. Code is the stable service-account
// code; callers must not branch on the safe display message.
type ServiceAccountError struct {
	StatusCode	int
	Code		string
	Retryable	bool
	RequestID	string
}
```

### ServiceAccountPrincipalResult

ServiceAccountPrincipalResult describes the result of a historical binding request.

```go
// ServiceAccountPrincipalResult describes the result of a historical binding request.
type ServiceAccountPrincipalResult struct {
	UCServiceAccountID	string	`json:"uc_service_account_id"`
	WorkspaceID		string	`json:"workspace_id"`
	RoleID			int64	`json:"role_id"`
	RemoteRevision		string	`json:"remote_revision"`
	SyncState		string	`json:"sync_state"`
}
```

### ServiceAccountPrincipalSpec

ServiceAccountPrincipalSpec describes a historical service-account binding request.

```go
// ServiceAccountPrincipalSpec describes a historical service-account binding request.
type ServiceAccountPrincipalSpec struct {
	UCServiceAccountID	string	`json:"uc_service_account_id"`
	KeycloakSubject		string	`json:"keycloak_subject"`
	ServiceGeneration	int64	`json:"service_generation"`
	DisplayName		string	`json:"display_name"`
	RoleID			int64	`json:"role_id"`
}
```

### ServiceAccountRole

ServiceAccountRole is a role the asserted human actor may assign in a workspace.

```go
// ServiceAccountRole is a role the asserted human actor may assign in a workspace.
type ServiceAccountRole struct {
	RoleID		int64	`json:"role_id"`
	DisplayName	string	`json:"display_name"`
}
```

### ServiceAccountWorkspace

ServiceAccountWorkspace is a workspace the asserted human actor may manage.

```go
// ServiceAccountWorkspace is a workspace the asserted human actor may manage.
type ServiceAccountWorkspace struct {
	WorkspaceID	string	`json:"workspace_id"`
	DisplayName	string	`json:"display_name"`
	RemoteVersion	string	`json:"remote_version"`
}
```

### ServiceAccountWriteOptions

ServiceAccountWriteOptions is retained only for source compatibility with
the retired product write facade. AI Studio catalog-only mode rejects those
writes locally; UC is the sole binding writer.

It is intentionally a method argument for the future service-account
facade, rather than client configuration, so the assertion and operation
fields are never shared by later requests.

```go
// ServiceAccountWriteOptions is retained only for source compatibility with
// the retired product write facade. AI Studio catalog-only mode rejects those
// writes locally; UC is the sole binding writer.
//
// It is intentionally a method argument for the future service-account
// facade, rather than client configuration, so the assertion and operation
// fields are never shared by later requests.
type ServiceAccountWriteOptions struct {
	ActorAssertion	string
	OperationID	string
	OperationSeq	int64
	RequestHash	string
}
```

### SetEndpointStatusOption

SetEndpointStatusOption configures set endpoint status options.

```go
// SetEndpointStatusOption configures set endpoint status options.
type SetEndpointStatusOption func(*setEndpointStatusOptions)
```

### Source

Source represents an input source for parser workitems.
Re-exported from model/data for convenience.

```go
// Source represents an input source for parser workitems.
// Re-exported from model/data for convenience.
type Source = data.Source
```

### SplitDocumentsLengthRequest

SplitDocumentsLengthRequest represents a split documents length workitem input.
Re-exported from model/data for convenience.

```go
// SplitDocumentsLengthRequest represents a split documents length workitem input.
// Re-exported from model/data for convenience.
type SplitDocumentsLengthRequest = data.SplitDocumentsLengthRequest
```

### SplitDocumentsLengthResponse

SplitDocumentsLengthResponse represents a split documents length workitem output.
Re-exported from model/data for convenience.

```go
// SplitDocumentsLengthResponse represents a split documents length workitem output.
// Re-exported from model/data for convenience.
type SplitDocumentsLengthResponse = data.SplitDocumentsLengthResponse
```

### SplitLengthRequest

SplitLengthRequest represents a split length workitem input.
Re-exported from model/data for convenience.

```go
// SplitLengthRequest represents a split length workitem input.
// Re-exported from model/data for convenience.
type SplitLengthRequest = data.SplitLengthRequest
```

### SplitLengthResponse

SplitLengthResponse represents a split length workitem output.
Re-exported from model/data for convenience.

```go
// SplitLengthResponse represents a split length workitem output.
// Re-exported from model/data for convenience.
type SplitLengthResponse = data.SplitLengthResponse
```

### SplitLevelRequest

SplitLevelRequest represents a split level workitem input.
Re-exported from model/data for convenience.

```go
// SplitLevelRequest represents a split level workitem input.
// Re-exported from model/data for convenience.
type SplitLevelRequest = data.SplitLevelRequest
```

### SplitLevelResponse

SplitLevelResponse represents a split level workitem output.
Re-exported from model/data for convenience.

```go
// SplitLevelResponse represents a split level workitem output.
// Re-exported from model/data for convenience.
type SplitLevelResponse = data.SplitLevelResponse
```

### Status

Status is an alias to mowl.Status for backward compatibility.
Use mowl.Status directly for new code.

```go
// Status is an alias to mowl.Status for backward compatibility.
// Use mowl.Status directly for new code.
type Status = mowl.Status
```

### StreamEvent

StreamEvent represents a streaming event from a dynamic service (stream mode).

```go
// StreamEvent represents a streaming event from a dynamic service (stream mode).
type StreamEvent struct {
	CaseID	string
	Data	string
	Done	bool
	Error	string
}
```

### StreamExportRequest

StreamExportRequest represents a stream export workitem input.
Re-exported from model/data for convenience.

```go
// StreamExportRequest represents a stream export workitem input.
// Re-exported from model/data for convenience.
type StreamExportRequest = data.StreamExportRequest
```

### StreamResult

StreamResult wraps a gRPC server stream for dynamic service invocation.
调用者通过 Recv() 逐条读取事件，无需额外 goroutine。

```go
// StreamResult wraps a gRPC server stream for dynamic service invocation.
// 调用者通过 Recv() 逐条读取事件，无需额外 goroutine。
type StreamResult struct {
	stream	grpc.ServerStreamingClient[mowl.StreamEventResponse]
	done	bool
}
```

### StreamWorkItemFunc

StreamWorkItemFunc is the handler for stream work items. The SDK creates a StreamWriter and passes it
to the handler; the handler should first return (so the SDK can send WAITING to the engine), then use
the StreamWriter from its own goroutine to call Emit repeatedly and finally End.
Register with RegisterStreamWorkItem. The stream capability is declared in workitem metadata (stream=true).

```go
// StreamWorkItemFunc is the handler for stream work items. The SDK creates a StreamWriter and passes it
// to the handler; the handler should first return (so the SDK can send WAITING to the engine), then use
// the StreamWriter from its own goroutine to call Emit repeatedly and finally End.
// Register with RegisterStreamWorkItem. The stream capability is declared in workitem metadata (stream=true).
type StreamWorkItemFunc func(ctx context.Context, wctx WorkItemContext, msg *mowl.MowlMessage, sw StreamWriter) error
```

### StreamWriter

StreamWriter is used by stream work item handlers to send multiple events and then end the stream.
The engine will deliver each Emit as a token to downstream nodes; End finalizes the stream session.

Thread safety: StreamWriter is NOT safe for concurrent use. It should only be called from a single goroutine.

Lifecycle: Call Emit zero or more times, then call End exactly once. Calling Emit after End returns an error.
Calling End more than once returns an error.

Backpressure: Emit blocks if the internal send buffer is full, providing natural backpressure.
If the worker session is cancelled, Emit and End return a context error.

```go
// StreamWriter is used by stream work item handlers to send multiple events and then end the stream.
// The engine will deliver each Emit as a token to downstream nodes; End finalizes the stream session.
//
// Thread safety: StreamWriter is NOT safe for concurrent use. It should only be called from a single goroutine.
//
// Lifecycle: Call Emit zero or more times, then call End exactly once. Calling Emit after End returns an error.
// Calling End more than once returns an error.
//
// Backpressure: Emit blocks if the internal send buffer is full, providing natural backpressure.
// If the worker session is cancelled, Emit and End return a context error.
type StreamWriter interface {
	// Emit sends a streaming event (data/vars) to the engine. Can be called multiple times.
	// Blocks if the send buffer is full. Returns an error if End has already been called
	// or the session is closed.
	Emit(data, vars string) error
	// End ends the stream with the given status. Valid values: "COMPLETED", "FAILED".
	// Must be called exactly once. Returns an error if already called or the session is closed.
	End(status string) error
	// EndWithResult ends the stream with the given status and explicit final data/vars.
	// Use this when the stream workitem needs to hand off a final business result to downstream
	// workflow nodes while still emitting intermediate progress events.
	EndWithResult(status, data, vars string) error
}
```

### StructuredExtractAdvancedRequest

StructuredExtractAdvancedRequest represents a structured extract advanced workitem input.
Re-exported from model/data for convenience.

```go
// StructuredExtractAdvancedRequest represents a structured extract advanced workitem input.
// Re-exported from model/data for convenience.
type StructuredExtractAdvancedRequest = data.StructuredExtractAdvancedRequest
```

### StructuredExtractAdvancedResponse

StructuredExtractAdvancedResponse represents a structured extract advanced workitem output.
Re-exported from model/data for convenience.

```go
// StructuredExtractAdvancedResponse represents a structured extract advanced workitem output.
// Re-exported from model/data for convenience.
type StructuredExtractAdvancedResponse = data.StructuredExtractAdvancedResponse
```

### StructuredExtractRequest

StructuredExtractRequest represents a structured extract workitem input.
Re-exported from model/data for convenience.

```go
// StructuredExtractRequest represents a structured extract workitem input.
// Re-exported from model/data for convenience.
type StructuredExtractRequest = data.StructuredExtractRequest
```

### StructuredExtractResponse

StructuredExtractResponse represents a structured extract workitem output.
Re-exported from model/data for convenience.

```go
// StructuredExtractResponse represents a structured extract workitem output.
// Re-exported from model/data for convenience.
type StructuredExtractResponse = data.StructuredExtractResponse
```

### SwitchEffectiveRevisionsResponse

SwitchEffectiveRevisionsResponse returns the new target effective-set version.

```go
// SwitchEffectiveRevisionsResponse returns the new target effective-set version.
type SwitchEffectiveRevisionsResponse struct {
	EffectiveSetVersion int64 `json:"effective_set_version"`
}
```

### SyncCDHMetadataResponse

SyncCDHMetadataResponse represents the response from syncing CDH metadata.
Re-exported from model/catalog for convenience.

```go
// SyncCDHMetadataResponse represents the response from syncing CDH metadata.
// Re-exported from model/catalog for convenience.
type SyncCDHMetadataResponse = catalog.SyncCDHMetadataResponse
```

### SyncMetadataOption

SyncMetadataOption configures optional parameters for SyncMetadata.

```go
// SyncMetadataOption configures optional parameters for SyncMetadata.
type SyncMetadataOption func(*syncMetadataOptions)
```

### SystemDefaultAIBackendConfig

SystemDefaultAIBackendConfig describes one default backend.

```go
// SystemDefaultAIBackendConfig describes one default backend.
type SystemDefaultAIBackendConfig struct {
	ID			int64				`json:"id,omitempty"`
	Name			string				`json:"name"`
	Type			string				`json:"type"`
	APIKeyEncrypted		string				`json:"api_key_encrypted,omitempty"`
	APIKeysEncrypted	[]string			`json:"api_keys_encrypted,omitempty"`
	TimeoutSeconds		int32				`json:"timeout_seconds,omitempty"`
	Models			[]string			`json:"models,omitempty"`
	SupportedMimeTypes	[]string			`json:"supported_mime_types,omitempty"`
	Status			string				`json:"status,omitempty"`
	Priority		int32				`json:"priority,omitempty"`
	CreatedAt		int64				`json:"created_at,omitempty"`
	UpdatedAt		int64				`json:"updated_at,omitempty"`
	Endpoints		[]SystemDefaultAIEndpointConfig	`json:"endpoints"`
}
```

### SystemDefaultAIConfig

SystemDefaultAIConfig is the complete system default AI service config payload.

```go
// SystemDefaultAIConfig is the complete system default AI service config payload.
type SystemDefaultAIConfig struct {
	Services []SystemDefaultAIServiceConfig `json:"services"`
}
```

### SystemDefaultAIEndpointConfig

SystemDefaultAIEndpointConfig describes one backend endpoint.

```go
// SystemDefaultAIEndpointConfig describes one backend endpoint.
type SystemDefaultAIEndpointConfig struct {
	ID		int64	`json:"id,omitempty"`
	BackendID	int64	`json:"backend_id,omitempty"`
	Address		string	`json:"address"`
	Status		string	`json:"status,omitempty"`
	CreatedAt	int64	`json:"created_at,omitempty"`
	UpdatedAt	int64	`json:"updated_at,omitempty"`
}
```

### SystemDefaultAIRouterConfig

SystemDefaultAIRouterConfig controls backend routing for a default service.

```go
// SystemDefaultAIRouterConfig controls backend routing for a default service.
type SystemDefaultAIRouterConfig struct {
	Strategy			string	`json:"strategy,omitempty"`
	HealthCheckIntervalSeconds	int32	`json:"health_check_interval_seconds,omitempty"`
	MaxRetries			int32	`json:"max_retries,omitempty"`
	EnableSessionAffinity		bool	`json:"enable_session_affinity,omitempty"`
}
```

### SystemDefaultAIServiceConfig

SystemDefaultAIServiceConfig describes one service type's default backends and router config.

```go
// SystemDefaultAIServiceConfig describes one service type's default backends and router config.
type SystemDefaultAIServiceConfig struct {
	ServiceType	string				`json:"service_type"`
	Version		int64				`json:"version,omitempty"`
	RouterConfig	*SystemDefaultAIRouterConfig	`json:"router_config,omitempty"`
	Backends	[]SystemDefaultAIBackendConfig	`json:"backends"`
}
```

### SystemWorkflowExecutionRequest

SystemWorkflowExecutionRequest starts a hidden system workflow through the workflow-app runtime.
Set ComputeResourceID to SystemWorkflowComputeResourceShared to explicitly
select the shared worker pool instead of the workflow app's CR binding.

```go
// SystemWorkflowExecutionRequest starts a hidden system workflow through the workflow-app runtime.
// Set ComputeResourceID to SystemWorkflowComputeResourceShared to explicitly
// select the shared worker pool instead of the workflow app's CR binding.
type SystemWorkflowExecutionRequest struct {
	IdempotencyKey		string	`json:"idempotency_key,omitempty"`
	ExecutionMode		string	`json:"execution_mode,omitempty"`
	CronExpression		string	`json:"cron_expression,omitempty"`
	TriggerNow		bool	`json:"trigger_now,omitempty"`
	DataJSON		string	`json:"data_json,omitempty"`
	VarsJSON		string	`json:"vars_json,omitempty"`
	TaskID			string	`json:"task_id,omitempty"`
	TaskName		string	`json:"task_name,omitempty"`
	Transient		bool	`json:"transient,omitempty"`
	ComputeResourceID	string	`json:"compute_resource_id,omitempty"`
}
```

### SystemWorkflowExecutionResponse

SystemWorkflowExecutionResponse reports the workflow-app execution and linked Mowl task.

```go
// SystemWorkflowExecutionResponse reports the workflow-app execution and linked Mowl task.
type SystemWorkflowExecutionResponse struct {
	WorkflowAppID		string	`json:"workflow_app_id"`
	WorkflowDefID		string	`json:"mowl_workflow_def_id"`
	WorkflowVersionID	string	`json:"mowl_workflow_version_id"`
	ExecutionID		string	`json:"workflow_execution_id"`
	MoiTaskID		string	`json:"moi_task_id"`
	MoiCaseID		string	`json:"moi_case_id,omitempty"`
	Status			string	`json:"status"`
	Error			string	`json:"error,omitempty"`
}
```

### SystemWorkflowRef

SystemWorkflowRef identifies a provisioned hidden system workflow app.

```go
// SystemWorkflowRef identifies a provisioned hidden system workflow app.
type SystemWorkflowRef struct {
	WorkflowAppID		string	`json:"workflow_app_id"`
	WorkflowDefID		string	`json:"mowl_workflow_def_id"`
	WorkflowVersionID	string	`json:"mowl_workflow_version_id"`
	PipelineGraphJSON	string	`json:"pipeline_graph_json,omitempty"`
}
```

### TagRelationOption

TagRelationOption configures tag relation operations.

```go
// TagRelationOption configures tag relation operations.
type TagRelationOption func(*tagRelationOptions)
```

### TraceContext

TraceContext provides tracing hooks within a work item handler.
The default implementation is a no-op.

```go
// TraceContext provides tracing hooks within a work item handler.
// The default implementation is a no-op.
type TraceContext interface {
	StartSpan(name string, attrs map[string]string) (TraceSpan, error)
}
```

### TraceRenderer

TraceRenderer renders different views from a TraceResponse.

```go
// TraceRenderer renders different views from a TraceResponse.
type TraceRenderer struct {
	resp *mowlpb.TraceResponse
}
```

### TraceSpan

TraceSpan represents a span in a trace context.

```go
// TraceSpan represents a span in a trace context.
type TraceSpan interface {
	End(status, errMsg string)
}
```

### TraceTreeNode

TraceTreeNode represents a span node with children for tree rendering.

```go
// TraceTreeNode represents a span node with children for tree rendering.
type TraceTreeNode struct {
	Span		*mowlpb.TraceSpan
	Children	[]*TraceTreeNode
}
```

### Tree

Tree represents a hierarchical tree of workspace resources.

```go
// Tree represents a hierarchical tree of workspace resources.
type Tree struct {
	Root		TreeNode
	Depth		int
	NodeCount	int
	Errors		[]TreeError
}
```

### TreeError

TreeError represents an error that occurred during tree construction.

```go
// TreeError represents an error that occurred during tree construction.
type TreeError struct {
	NodeID		string	`json:"node_id"`
	NodeType	string	`json:"node_type"`
	Message		string	`json:"message"`
}
```

### TreeNode

TreeNode is the interface for all tree nodes.

```go
// TreeNode is the interface for all tree nodes.
type TreeNode interface {
	ID() int64
	Name() string
	Type() NodeType
	Children() []TreeNode
	Parent() TreeNode
	setParent(TreeNode)
	addChild(TreeNode)
}
```

### TreeStats

TreeStats contains statistics about the tree.

```go
// TreeStats contains statistics about the tree.
type TreeStats struct {
	CatalogCount	int	`json:"catalog_count"`
	DatabaseCount	int	`json:"database_count"`
	VolumeCount	int	`json:"volume_count"`
	Depth		int	`json:"depth"`
	TotalNodes	int	`json:"total_nodes"`
}
```

### TriggerFilesRequest

TriggerFilesRequest represents the request to trigger workflows for existing volume files.

```go
// TriggerFilesRequest represents the request to trigger workflows for existing volume files.
type TriggerFilesRequest struct {
	FileIDs []string `json:"file_ids"`
}
```

### TriggerFilesResponse

TriggerFilesResponse contains created delivery count.

```go
// TriggerFilesResponse contains created delivery count.
type TriggerFilesResponse struct {
	Triggered int `json:"triggered"`
}
```

### TriggerGarbageCollectionOption

TriggerGarbageCollectionOption is a function that configures trigger garbage collection options.

```go
// TriggerGarbageCollectionOption is a function that configures trigger garbage collection options.
type TriggerGarbageCollectionOption func(*TriggerGarbageCollectionOptions)
```

### TriggerGarbageCollectionOptions

TriggerGarbageCollectionOptions holds the optional parameters for triggering garbage collection.

```go
// TriggerGarbageCollectionOptions holds the optional parameters for triggering garbage collection.
type TriggerGarbageCollectionOptions struct {
	// OrphanFileThresholdHours is the minimum age of orphan files before they can be cleaned (in hours).
	// Default: 24 hours
	OrphanFileThresholdHours	int
	// BatchSize is the maximum number of items to process.
	// Default: 100
	BatchSize	int
}
```

### TriggerTaskResponse

TriggerTaskResponse represents the response from triggering a task execution.

```go
// TriggerTaskResponse represents the response from triggering a task execution.
type TriggerTaskResponse struct {
	TaskID	string	`json:"task_id"`
	CaseID	string	`json:"case_id"`
}
```

### UpdateBackendOption

UpdateBackendOption configures update backend options.

```go
// UpdateBackendOption configures update backend options.
type UpdateBackendOption func(*updateBackendOptions)
```

### UpdateCDHConfigOption

UpdateCDHConfigOption is a function that configures update CDH config options.

```go
// UpdateCDHConfigOption is a function that configures update CDH config options.
type UpdateCDHConfigOption func(*updateCDHConfigOptions)
```

### UpdateCatalogOption

UpdateCatalogOption is a function that configures update catalog options.

```go
// UpdateCatalogOption is a function that configures update catalog options.
type UpdateCatalogOption func(*updateCatalogOptions)
```

### UpdateCustomOperatorRequest

UpdateCustomOperatorRequest is the request body for updating a custom operator.

```go
// UpdateCustomOperatorRequest is the request body for updating a custom operator.
type UpdateCustomOperatorRequest struct {
	Name		*string	`json:"name,omitempty"`
	Description	*string	`json:"description,omitempty"`
	Kind		*string	`json:"kind,omitempty"`
	Language	*string	`json:"language,omitempty"`
	Handler		*string	`json:"handler,omitempty"`
	Version		*string	`json:"version,omitempty"`
	IsolationLevel	*string	`json:"isolation_level,omitempty"`
	InputSchema	any	`json:"input_schema,omitempty"`
	OutputSchema	any	`json:"output_schema,omitempty"`
	Code		*string	`json:"code,omitempty"`
	SourceFileID	*string	`json:"source_file_id,omitempty"`
	BaseNodeID	*string	`json:"base_node_id,omitempty"`
	BaseNodeVersion	*string	`json:"base_node_version,omitempty"`
	BindingConfig	any	`json:"binding_config,omitempty"`
	CatalogID	*int64	`json:"catalog_id,omitempty"`
	DatabaseID	*int64	`json:"database_id,omitempty"`
}
```

### UpdateDPConfigOption

UpdateDPConfigOption is a function that configures update Dataphin config options.

```go
// UpdateDPConfigOption is a function that configures update Dataphin config options.
type UpdateDPConfigOption func(*updateDPConfigOptions)
```

### UpdateMCConfigOption

UpdateMCConfigOption is a function that configures update MaxCompute config options.

```go
// UpdateMCConfigOption is a function that configures update MaxCompute config options.
type UpdateMCConfigOption func(*updateMCConfigOptions)
```

### UpdateParserBackendOption

UpdateParserBackendOption configures update parser backend options.

```go
// UpdateParserBackendOption configures update parser backend options.
type UpdateParserBackendOption func(*updateParserBackendOptions)
```

### UpdateRevisionStatusResponse

UpdateRevisionStatusResponse returns the updated revision and target effective-set version.

```go
// UpdateRevisionStatusResponse returns the updated revision and target effective-set version.
type UpdateRevisionStatusResponse struct {
	Revision		*mowlpb.OutputBlockRevision	`json:"revision"`
	EffectiveSetVersion	int64				`json:"effective_set_version"`
}
```

### UpdateSessionOption

UpdateSessionOption configures update session options.

```go
// UpdateSessionOption configures update session options.
type UpdateSessionOption func(*updateSessionOptions)
```

### UpdateUserOption

UpdateUserOption is a function that configures update user options.

```go
// UpdateUserOption is a function that configures update user options.
type UpdateUserOption func(*updateUserOptions)
```

### UpdateVolumeOption

UpdateVolumeOption is a function that configures update volume options.

```go
// UpdateVolumeOption is a function that configures update volume options.
type UpdateVolumeOption func(*updateVolumeOptions)
```

### UpdateWorkflowDefOption

UpdateWorkflowDefOption is a function that configures update workflow definition options.

```go
// UpdateWorkflowDefOption is a function that configures update workflow definition options.
type UpdateWorkflowDefOption func(*updateWorkflowDefOptions)
```

### UpdateWorkspaceOption

UpdateWorkspaceOption is a function that configures update workspace options.

```go
// UpdateWorkspaceOption is a function that configures update workspace options.
type UpdateWorkspaceOption func(*updateWorkspaceOptions)
```

### UploadFileOption

UploadFileOption is a function that configures upload file options.

```go
// UploadFileOption is a function that configures upload file options.
type UploadFileOption func(*uploadFileOptions)
```

### UploadFileResponse

UploadFileResponse represents the response from file upload.

```go
// UploadFileResponse represents the response from file upload.
type UploadFileResponse struct {
	FileID		string		`json:"file_id"`
	OriginalName	string		`json:"original_name"`
	Size		int64		`json:"size"`
	MD5		string		`json:"md5"`
	CatalogFile	*CatalogFile	`json:"catalog_file,omitempty"`
}
```

### UploadOption

UploadOption is a function that configures upload options.

```go
// UploadOption is a function that configures upload options.
type UploadOption func(*uploadOptions)
```

### User

User represents a user in the system.
Re-exported from model/user for convenience.

```go
// User represents a user in the system.
// Re-exported from model/user for convenience.
type User = user.User
```

### UserStatus

UserStatus represents the status of a user.
Re-exported from model/user for convenience.

```go
// UserStatus represents the status of a user.
// Re-exported from model/user for convenience.
type UserStatus = user.UserStatus
```

### VectorDeleteRequest

VectorDeleteRequest represents a vector delete workitem input.
Re-exported from model/data for convenience.

```go
// VectorDeleteRequest represents a vector delete workitem input.
// Re-exported from model/data for convenience.
type VectorDeleteRequest = data.VectorDeleteRequest
```

### VectorDeleteResponse

VectorDeleteResponse represents a vector delete workitem output.
Re-exported from model/data for convenience.

```go
// VectorDeleteResponse represents a vector delete workitem output.
// Re-exported from model/data for convenience.
type VectorDeleteResponse = data.VectorDeleteResponse
```

### VectorDoc

VectorDoc represents a vector document payload.
Re-exported from model/data for convenience.

```go
// VectorDoc represents a vector document payload.
// Re-exported from model/data for convenience.
type VectorDoc = data.VectorDoc
```

### VectorSearchRequest

VectorSearchRequest represents a vector search workitem input.
Re-exported from model/data for convenience.

```go
// VectorSearchRequest represents a vector search workitem input.
// Re-exported from model/data for convenience.
type VectorSearchRequest = data.VectorSearchRequest
```

### VectorSearchResponse

VectorSearchResponse represents a vector search workitem output.
Re-exported from model/data for convenience.

```go
// VectorSearchResponse represents a vector search workitem output.
// Re-exported from model/data for convenience.
type VectorSearchResponse = data.VectorSearchResponse
```

### VectorUpsertRequest

VectorUpsertRequest represents a vector upsert workitem input.
Re-exported from model/data for convenience.

```go
// VectorUpsertRequest represents a vector upsert workitem input.
// Re-exported from model/data for convenience.
type VectorUpsertRequest = data.VectorUpsertRequest
```

### VectorUpsertResponse

VectorUpsertResponse represents a vector upsert workitem output.
Re-exported from model/data for convenience.

```go
// VectorUpsertResponse represents a vector upsert workitem output.
// Re-exported from model/data for convenience.
type VectorUpsertResponse = data.VectorUpsertResponse
```

### Volume

Volume represents a volume within a database.
Re-exported from model/catalog for convenience.

```go
// Volume represents a volume within a database.
// Re-exported from model/catalog for convenience.
type Volume = catalog.Volume
```

### VolumeContentBuilder

VolumeContentBuilder builds queries for volume contents (sub-volumes and files).
It provides a fluent interface for constructing volume content queries.

```go
// VolumeContentBuilder builds queries for volume contents (sub-volumes and files).
// It provides a fluent interface for constructing volume content queries.
type VolumeContentBuilder struct {
	client		*Client
	ctx		context.Context
	workspaceID	string
	volumeID	int64
	// Content type filters
	includeVolumes	bool
	includeFiles	bool
	// Filters
	nameFilter	string
	typeFilter	string	// "volume" or "file"
	// Sorting
	orderBy		string
	orderDesc	bool
	// Pagination
	pageSize	int32
	pageToken	string
}
```

### VolumeContentItem

VolumeContentItem represents a single item in volume contents.

```go
// VolumeContentItem represents a single item in volume contents.
type VolumeContentItem struct {
	Type		string			`json:"type"`	// "volume" or "file"
	ID		string			`json:"id"`
	Name		string			`json:"name"`
	CreatedAt	int64			`json:"created_at"`
	UpdatedAt	int64			`json:"updated_at"`
	CreatedBy	string			`json:"created_by"`
	Volume		*catalog.Volume		`json:"volume,omitempty"`
	File		*catalog.VolumeFile	`json:"file,omitempty"`
}
```

### VolumeContentResult

VolumeContentResult represents the result of a volume content query.

```go
// VolumeContentResult represents the result of a volume content query.
type VolumeContentResult struct {
	Volume		*catalog.Volume		`json:"volume"`
	Items		[]*VolumeContentItem	`json:"items"`
	Total		int64			`json:"total"`
	NextPageToken	string			`json:"next_page_token,omitempty"`
	Stats		VolumeContentStats	`json:"stats"`
}
```

### VolumeContentStats

VolumeContentStats represents statistics about volume contents.

```go
// VolumeContentStats represents statistics about volume contents.
type VolumeContentStats struct {
	VolumeCount	int64	`json:"volume_count"`
	FileCount	int64	`json:"file_count"`
	TotalCount	int64	`json:"total_count"`
}
```

### VolumeEnsureRequest

VolumeEnsureRequest represents a volume ensure workitem input.
Re-exported from model/data for convenience.

```go
// VolumeEnsureRequest represents a volume ensure workitem input.
// Re-exported from model/data for convenience.
type VolumeEnsureRequest = data.VolumeEnsureRequest
```

### VolumeEnsureResponse

VolumeEnsureResponse represents a volume ensure workitem output.
Re-exported from model/data for convenience.

```go
// VolumeEnsureResponse represents a volume ensure workitem output.
// Re-exported from model/data for convenience.
type VolumeEnsureResponse = data.VolumeEnsureResponse
```

### VolumeFile

VolumeFile represents a file associated with a volume.
This is a type alias for the protobuf VolumeFile.

```go
// VolumeFile represents a file associated with a volume.
// This is a type alias for the protobuf VolumeFile.
type VolumeFile = catalog.VolumeFile
```

### VolumeFileDetail

VolumeFileDetail is a type alias for the protobuf VolumeFileDetail.

```go
// VolumeFileDetail is a type alias for the protobuf VolumeFileDetail.
type VolumeFileDetail = catalog.VolumeFileDetail
```

### VolumeFileItem

VolumeFileItem represents a volume file item payload.
Re-exported from model/data for convenience.

```go
// VolumeFileItem represents a volume file item payload.
// Re-exported from model/data for convenience.
type VolumeFileItem = data.VolumeFileItem
```

### VolumeFileListOption

ListFilesOption is a function that configures list files options.

```go
// ListFilesOption is a function that configures list files options.
type VolumeFileListOption func(*volumeFileListOptions)
```

### VolumeFilesAddRequest

VolumeFilesAddRequest represents a volume files add workitem input.
Re-exported from model/data for convenience.

```go
// VolumeFilesAddRequest represents a volume files add workitem input.
// Re-exported from model/data for convenience.
type VolumeFilesAddRequest = data.VolumeFilesAddRequest
```

### VolumeFilesAddResponse

VolumeFilesAddResponse represents a volume files add workitem output.
Re-exported from model/data for convenience.

```go
// VolumeFilesAddResponse represents a volume files add workitem output.
// Re-exported from model/data for convenience.
type VolumeFilesAddResponse = data.VolumeFilesAddResponse
```

### VolumeFilesListRequest

VolumeFilesListRequest represents a volume files list workitem input.
Re-exported from model/data for convenience.

```go
// VolumeFilesListRequest represents a volume files list workitem input.
// Re-exported from model/data for convenience.
type VolumeFilesListRequest = data.VolumeFilesListRequest
```

### VolumeFilesListResponse

VolumeFilesListResponse represents a volume files list workitem output.
Re-exported from model/data for convenience.

```go
// VolumeFilesListResponse represents a volume files list workitem output.
// Re-exported from model/data for convenience.
type VolumeFilesListResponse = data.VolumeFilesListResponse
```

### VolumeFilesMoveRequest

VolumeFilesMoveRequest represents a volume files move workitem input.
Re-exported from model/data for convenience.

```go
// VolumeFilesMoveRequest represents a volume files move workitem input.
// Re-exported from model/data for convenience.
type VolumeFilesMoveRequest = data.VolumeFilesMoveRequest
```

### VolumeFilesMoveResponse

VolumeFilesMoveResponse represents a volume files move workitem output.
Re-exported from model/data for convenience.

```go
// VolumeFilesMoveResponse represents a volume files move workitem output.
// Re-exported from model/data for convenience.
type VolumeFilesMoveResponse = data.VolumeFilesMoveResponse
```

### VolumeFilesRemoveRequest

VolumeFilesRemoveRequest represents a volume files remove workitem input.
Re-exported from model/data for convenience.

```go
// VolumeFilesRemoveRequest represents a volume files remove workitem input.
// Re-exported from model/data for convenience.
type VolumeFilesRemoveRequest = data.VolumeFilesRemoveRequest
```

### VolumeFilesRemoveResponse

VolumeFilesRemoveResponse represents a volume files remove workitem output.
Re-exported from model/data for convenience.

```go
// VolumeFilesRemoveResponse represents a volume files remove workitem output.
// Re-exported from model/data for convenience.
type VolumeFilesRemoveResponse = data.VolumeFilesRemoveResponse
```

### VolumeIterator

VolumeIterator iterates over volumes with automatic pagination.

```go
// VolumeIterator iterates over volumes with automatic pagination.
type VolumeIterator struct {
	service		*VolumeService
	ctx		context.Context
	workspaceID	string
	databaseID	int64
	buffer		[]*catalog.Volume
	index		int
	pageToken	string
	pageSize	int32
	done		bool
	stopped		bool
	err		error
}
```

### VolumeNode

VolumeNode represents a volume in the tree.
Supports hierarchical structure with child volumes.

```go
// VolumeNode represents a volume in the tree.
// Supports hierarchical structure with child volumes.
type VolumeNode struct {
	volume		*catalog.Volume
	parent		TreeNode
	children	[]TreeNode
}
```

### VolumeResolveRequest

VolumeResolveRequest represents a volume resolve workitem input.
Re-exported from model/data for convenience.

```go
// VolumeResolveRequest represents a volume resolve workitem input.
// Re-exported from model/data for convenience.
type VolumeResolveRequest = data.VolumeResolveRequest
```

### VolumeResolveResponse

VolumeResolveResponse represents a volume resolve workitem output.
Re-exported from model/data for convenience.

```go
// VolumeResolveResponse represents a volume resolve workitem output.
// Re-exported from model/data for convenience.
type VolumeResolveResponse = data.VolumeResolveResponse
```

### WaterfallItem

WaterfallItem represents a span with relative timing for waterfall charts.

```go
// WaterfallItem represents a span with relative timing for waterfall charts.
type WaterfallItem struct {
	Span		*mowlpb.TraceSpan
	StartMs		int64
	DurationMs	int64
}
```

### WorkItemAgentContext

WorkItemAgentContext is the compact metadata subset shown to planning agents.

```go
// WorkItemAgentContext is the compact metadata subset shown to planning agents.
type WorkItemAgentContext struct {
	Provider	string
	DisplayName	string
	Category	string
	Tags		[]string
	Summary		string
	SideEffectClass	string
	Idempotence	string
	RequiredFields	[]string
}
```

### WorkItemCapabilityBuildOption

WorkItemCapabilityBuildOption customizes capability shaping for BuildWorkItemCapabilities.

```go
// WorkItemCapabilityBuildOption customizes capability shaping for BuildWorkItemCapabilities.
type WorkItemCapabilityBuildOption func(*workItemCapabilityBuildOptions)
```

### WorkItemContext

WorkItemContext provides access to workflow execution context within a work item handler.

```go
// WorkItemContext provides access to workflow execution context within a work item handler.
type WorkItemContext interface {
	// GetContext returns the underlying context.
	GetContext() context.Context

	// GetWorkflow returns the workflow definition for the current execution.
	GetWorkflow() *mowl.Workflow

	// GetNode returns the current node definition.
	GetNode() *mowl.Node

	// GetInput returns the input data for the current work item.
	GetInput() string

	// GetVars returns the workflow variables.
	GetVars() string

	// ExecutionContext returns the server-injected execution context
	// (workspace_id, user_id, user_api_key). It is not part of workflow vars.
	ExecutionContext() *data.ExecutionContext

	// RuntimeContext returns runtime identifiers from the current MowlMessage.
	RuntimeContext() *RuntimeContext

	// Trace returns a TraceContext for custom span annotation.
	Trace() TraceContext

	// SetOutput sets the output data for the current work item.
	SetOutput(data string)

	// SetVars sets the workflow variables.
	SetVars(vars string)

	// WriteStreamResult writes a streaming result to the dynamic service caller (stream mode only).
	// Each call sends one streaming event to the caller's stream channel.
	// Only effective in dynamic service stream mode; no-op in other modes (oneshot or regular workflow).
	WriteStreamResult(data string) error
}
```

### WorkItemDataFlowContract

WorkItemDataFlowContract groups WorkItem input and output data-flow ports.

```go
// WorkItemDataFlowContract groups WorkItem input and output data-flow ports.
type WorkItemDataFlowContract = mowl.WorkItemDataFlowContract
```

### WorkItemDataFlowPort

WorkItemDataFlowPort describes one deterministic input or output port for DSL binding.

```go
// WorkItemDataFlowPort describes one deterministic input or output port for DSL binding.
type WorkItemDataFlowPort = mowl.WorkItemDataFlowPort
```

### WorkItemOption

WorkItemOption is a function that modifies WorkItemMetadata during registration.

```go
// WorkItemOption is a function that modifies WorkItemMetadata during registration.
type WorkItemOption func(*mowl.WorkItemMetadata) error
```

### WorkItemUIField

WorkItemUIField describes one field in a WorkItem dynamic form.

```go
// WorkItemUIField describes one field in a WorkItem dynamic form.
type WorkItemUIField = mowl.WorkItemUIField
```

### WorkItemUIMetadataItem

WorkItemUIMetadataItem is the parsed UI-focused metadata for one workitem version.

```go
// WorkItemUIMetadataItem is the parsed UI-focused metadata for one workitem version.
type WorkItemUIMetadataItem struct {
	NodeID		string
	Version		string
	Description	string
	Stream		bool
	InputUISchema	*WorkItemUISchema
	OutputUISchema	*WorkItemUISchema
}
```

### WorkItemUIMetadataList

WorkItemUIMetadataList groups parsed UI metadata items by node_id.

```go
// WorkItemUIMetadataList groups parsed UI metadata items by node_id.
type WorkItemUIMetadataList struct {
	Items []*WorkItemUIMetadataItem
}
```

### WorkItemUIOption

WorkItemUIOption is one selectable option for a WorkItem UI field.

```go
// WorkItemUIOption is one selectable option for a WorkItem UI field.
type WorkItemUIOption = mowl.WorkItemUIOption
```

### WorkItemUISchema

WorkItemUISchema describes the dynamic input or output form for a WorkItem.

```go
// WorkItemUISchema describes the dynamic input or output form for a WorkItem.
type WorkItemUISchema = mowl.WorkItemUISchema
```

### WorkItemUISelector

WorkItemUISelector describes how a UI field resolves selectable resources.

```go
// WorkItemUISelector describes how a UI field resolves selectable resources.
type WorkItemUISelector = mowl.WorkItemUISelector
```

### WorkerClientOption

WorkerClientOption configures the worker client.

```go
// WorkerClientOption configures the worker client.
type WorkerClientOption func(*workerClientOptions)
```

### WorkerTaskOption

WorkerTaskOption configures task creation from the worker client.

```go
// WorkerTaskOption configures task creation from the worker client.
type WorkerTaskOption func(*workerTaskOptions)
```

### WorkflowAgentFormField

WorkflowAgentFormField is one renderable field in the input form. The frontend reads
`widget` to decide which component to render; everything else is rendering metadata.

The `bind_to` field is for server-internal routing only — clients should treat it as
opaque. Do not let users edit bind_to via the UI.

```go
// WorkflowAgentFormField is one renderable field in the input form. The frontend reads
// `widget` to decide which component to render; everything else is rendering metadata.
//
// The `bind_to` field is for server-internal routing only — clients should treat it as
// opaque. Do not let users edit bind_to via the UI.
type WorkflowAgentFormField struct {
	// FieldID is the unique identifier used as the key in submitted values map.
	FieldID	string	`json:"field_id"`
	// Label is the short human-readable title shown above the control (e.g. "数据源").
	Label	string	`json:"label"`
	// Description is a one-sentence explanation shown under the label.
	Description	string	`json:"description,omitempty"`
	// Widget names the UI component type; see FormFieldWidget* constants.
	Widget	FormFieldWidget	`json:"widget"`
	// ModelType narrows model picker widgets to a structured model category
	// such as "chat", "embedding", "vision", "ocr", "reasoning", or "rerank".
	ModelType	string	`json:"model_type,omitempty"`
	// Required marks whether the user must provide a value before submitting.
	Required	bool	`json:"required,omitempty"`
	// Default is the pre-filled value. Type matches the widget's expected input shape.
	Default	interface{}	`json:"default,omitempty"`
	// Placeholder is the hint text inside empty text/number inputs.
	Placeholder	string	`json:"placeholder,omitempty"`
	// Pattern is an optional regular expression (RE2 / JS-compatible) the value
	// must match. Frontends should validate client-side and block submission on
	// mismatch. Only meaningful for text-shaped widgets (text, textarea, number).
	Pattern	string	`json:"pattern,omitempty"`
	// RuntimeConfigPath is the canonical WorkItem runtime-config path that authored
	// this field. Unlike FieldID, it does not depend on the DSL variable name.
	RuntimeConfigPath	string	`json:"runtime_config_path,omitempty"`
	// Section and Group preserve the product presentation hierarchy declared by
	// the WorkItem input UI schema.
	Section	string	`json:"section,omitempty"`
	Group	string	`json:"group,omitempty"`
	// OnValue and OffValue let switch controls submit enum values instead of bools.
	OnValue		string	`json:"on_value,omitempty"`
	OffValue	string	`json:"off_value,omitempty"`
	// Hidden keeps internal contract values out of the editable product surface.
	Hidden	bool	`json:"hidden,omitempty"`
	// ReadOnly marks a contract-owned field. If it declares a default, UI callers
	// must not let copied or persisted values override that default.
	ReadOnly	bool	`json:"read_only,omitempty"`

	// Options populates select / radio / multi_select / checkbox_group.
	Options	[]WorkflowAgentFormFieldOption	`json:"options,omitempty"`
	// Min / Max apply to the `number` widget. Zero means unbounded.
	Min	*float64	`json:"min,omitempty"`
	Max	*float64	`json:"max,omitempty"`
	// Accept lists allowed file extensions for file_picker / file_picker_multi, e.g.
	// ["pdf", "docx"]. Empty slice means any type accepted.
	Accept	[]string	`json:"accept,omitempty"`
	// ResourceType narrows volume_picker / catalog_picker to a specific resource kind
	// (e.g. "dataset_catalog"). Empty means any resource of the picker's type is allowed.
	ResourceType	string	`json:"resource_type,omitempty"`

	// VisibleWhen controls conditional visibility. When set, the field is only shown
	// if the referenced field's current value equals the specified value.
	// Example: {"field_id":"enable_parser_pipeline","value":true}
	VisibleWhen	*FormFieldVisibleWhen	`json:"visible_when,omitempty"`

	// BindTo routes submitted values to DSL input/vars paths.
	//
	// Simple form (string): "vars.table_name" — writes the submitted value directly.
	//
	// Compound form (map): {"vector_table":"vars.table_name","embedding_model":"vars.emb_model"}
	// — the submitted value must be an object; each map entry extracts a key from the object
	// and writes it to the corresponding DSL path. Used by widgets that bundle multiple
	// related values (e.g. vector_index_select returns {vector_table, embedding_model}).
	//
	// Frontend MUST NOT modify this field — it's produced by the agent and consumed by the server.
	BindTo	json.RawMessage	`json:"bind_to,omitempty"`

	// OwnerNodeName/OwnerNodeID identify the WorkItem that authored this form field.
	// They are set only for backend-derived WorkItem-owned fields. External business
	// fields leave them empty and are grouped by their DSL references.
	OwnerNodeName	string	`json:"owner_node_name,omitempty"`
	OwnerNodeID	string	`json:"owner_node_id,omitempty"`
}
```

### WorkflowAgentFormFieldOption

WorkflowAgentFormFieldOption is one option for select / radio / multi_select / checkbox_group widgets.

```go
// WorkflowAgentFormFieldOption is one option for select / radio / multi_select / checkbox_group widgets.
type WorkflowAgentFormFieldOption struct {
	// Label is the human-readable text shown in the UI.
	Label	string	`json:"label"`
	// Value is what gets submitted when this option is selected. Must be JSON-serializable.
	Value	interface{}	`json:"value"`
	// Description is optional helper text shown next to the option.
	Description	string	`json:"description,omitempty"`
}
```

### WorkflowAgentInputForm

WorkflowAgentInputForm is the whole form the frontend renders before saving or running an
accepted workflow candidate. It is produced by the workflow A2A agent and surfaced on the
workflow.candidate artifact.

```go
// WorkflowAgentInputForm is the whole form the frontend renders before saving or running an
// accepted workflow candidate. It is produced by the workflow A2A agent and surfaced on the
// workflow.candidate artifact.
type WorkflowAgentInputForm struct {
	// Title is the form's overall heading, e.g. "配置 RAG 导入流程".
	Title	string	`json:"title,omitempty"`
	// Description is one short paragraph explaining what the workflow will do once submitted.
	Description	string	`json:"description,omitempty"`
	// Fields are rendered in the order given. Frontends should not reorder them.
	Fields	[]WorkflowAgentFormField	`json:"fields,omitempty"`
}
```

### WorkflowAppDeleteResponse

WorkflowAppDeleteResponse reports deletion or cancellation side effects.

```go
// WorkflowAppDeleteResponse reports deletion or cancellation side effects.
type WorkflowAppDeleteResponse struct {
	Deleted			bool		`json:"deleted"`
	DisabledCronTaskIDs	[]string	`json:"disabled_cron_task_ids,omitempty"`
	Warnings		[]string	`json:"warnings,omitempty"`
}
```

### WorkflowAppDetail

WorkflowAppDetail contains the full product-level workflow app state.

```go
// WorkflowAppDetail contains the full product-level workflow app state.
type WorkflowAppDetail struct {
	WorkflowAppSummary
	Goal		string	`json:"goal,omitempty"`
	PlannerModel	string	`json:"planner_model,omitempty"`
	// SessionID is the workflow Copilot A2A conversation/context id.
	SessionID		string		`json:"session_id,omitempty"`
	DSLYAML			string		`json:"dsl_yaml"`
	InputFormJSON		string		`json:"runtime_fields_json,omitempty"`
	FormLayoutJSON		string		`json:"runtime_layout_json,omitempty"`
	DefaultValuesJSON	string		`json:"default_values_json,omitempty"`
	DesignGraphJSON		string		`json:"design_graph_json,omitempty"`
	RunContextJSON		string		`json:"run_context_json,omitempty"`
	DeploymentConfigJSON	string		`json:"deployment_config_json,omitempty"`
	DefaultValues		map[string]any	`json:"default_values,omitempty"`
}
```

### WorkflowAppEnvelope

WorkflowAppEnvelope wraps a workflow app detail response.

```go
// WorkflowAppEnvelope wraps a workflow app detail response.
type WorkflowAppEnvelope struct {
	Workflow WorkflowAppDetail `json:"workflow"`
}
```

### WorkflowAppExecutionSummary

WorkflowAppExecutionSummary summarizes recent workflow app execution state.

```go
// WorkflowAppExecutionSummary summarizes recent workflow app execution state.
type WorkflowAppExecutionSummary struct {
	TotalExecutions		int	`json:"total_executions"`
	ActiveExecutions	int	`json:"active_executions"`
	// ActiveExecutionID is the newest non-terminal execution (if any). Prefer this over
	// LatestExecutionID when pausing an in-flight run from the list view.
	ActiveExecutionID	string	`json:"active_execution_id,omitempty"`
	ActiveExecutionStatus	string	`json:"active_execution_status,omitempty"`
	LatestExecutionID	string	`json:"latest_execution_id,omitempty"`
	LatestExecutionStatus	string	`json:"latest_execution_status,omitempty"`
	LatestExecutionAt	string	`json:"latest_execution_at,omitempty"`
}
```

### WorkflowAppListRequest

WorkflowAppListRequest filters workflow app list results.

```go
// WorkflowAppListRequest filters workflow app list results.
type WorkflowAppListRequest struct {
	Offset			int
	Limit			int
	SourceType		string
	Status			string
	ExecutionMode		string
	NameSearch		string
	WorkflowDefID		string
	IncludeDynamicService	bool
}
```

### WorkflowAppListResponse

WorkflowAppListResponse contains workflow app summaries.

```go
// WorkflowAppListResponse contains workflow app summaries.
type WorkflowAppListResponse struct {
	Total		int			`json:"total"`
	Workflows	[]WorkflowAppSummary	`json:"workflows"`
}
```

### WorkflowAppSummary

WorkflowAppSummary is a compact workflow app projection.

```go
// WorkflowAppSummary is a compact workflow app projection.
type WorkflowAppSummary struct {
	ID			string				`json:"id"`
	Name			string				`json:"name"`
	Description		string				`json:"description,omitempty"`
	SourceType		string				`json:"source_type"`
	Status			string				`json:"status"`
	ExecutionMode		string				`json:"execution_mode"`
	CronExpression		string				`json:"cron_expression,omitempty"`
	DraftID			string				`json:"draft_id,omitempty"`
	CandidateID		string				`json:"candidate_id,omitempty"`
	MoiWorkflowDefID	string				`json:"moi_workflow_def_id,omitempty"`
	LatestVersionID		string				`json:"latest_workflow_version_id,omitempty"`
	ComputeResourceID	string				`json:"compute_resource_id,omitempty"`
	ComputeResourceBindings	[]ComputeResourceBinding	`json:"compute_resource_bindings,omitempty"`
	CreatedAt		string				`json:"created_at,omitempty"`
	UpdatedAt		string				`json:"updated_at,omitempty"`

	LatestVersionNumber	int				`json:"latest_version,omitempty"`
	LatestVersionStatus	string				`json:"latest_version_status,omitempty"`
	ParameterSummary	WorkflowParameterSummary	`json:"parameter_summary"`
	ExecutionSummary	WorkflowAppExecutionSummary	`json:"execution_summary"`
	TriggerSummary		WorkflowTriggerSummary		`json:"trigger_summary"`
}
```

### WorkflowAppUpdateRequest

WorkflowAppUpdateRequest updates mutable workflow app metadata.

```go
// WorkflowAppUpdateRequest updates mutable workflow app metadata.
type WorkflowAppUpdateRequest struct {
	Name			*string	`json:"name,omitempty"`
	Description		*string	`json:"description,omitempty"`
	Status			*string	`json:"status,omitempty"`
	DefaultValuesJSON	string	`json:"default_values_json,omitempty"`
	ComputeResourceID	*string	`json:"compute_resource_id,omitempty"`
	// SessionID binds the workflow Copilot conversation (A2A context id).
	SessionID	*string	`json:"session_id,omitempty"`
}
```

### WorkflowAppValidateDeleteResponse

WorkflowAppValidateDeleteResponse reports whether a workflow app can be deleted.

```go
// WorkflowAppValidateDeleteResponse reports whether a workflow app can be deleted.
type WorkflowAppValidateDeleteResponse struct {
	Deletable bool `json:"deletable"`
}
```

### WorkflowDSLCompileError

WorkflowDSLCompileError wraps structured diagnostics as one compile error.

```go
// WorkflowDSLCompileError wraps structured diagnostics as one compile error.
type WorkflowDSLCompileError struct {
	Diagnostics []DSLCompileDiagnostic
}
```

### WorkflowDSLCompileResult

WorkflowDSLCompileResult contains compile output and structured diagnostics.

```go
// WorkflowDSLCompileResult contains compile output and structured diagnostics.
type WorkflowDSLCompileResult struct {
	Workflow	*mowl.Workflow		`json:"-"`
	Diagnostics	[]DSLCompileDiagnostic	`json:"diagnostics,omitempty"`
}
```

### WorkflowDSLDefinition

WorkflowDSLDefinition is the machine-readable DSL metadata contract for LLM/agent generation and repair.

```go
// WorkflowDSLDefinition is the machine-readable DSL metadata contract for LLM/agent generation and repair.
type WorkflowDSLDefinition struct {
	SyntaxVersion		string				`json:"syntax_version"`
	Format			string				`json:"format"`
	Source			string				`json:"source,omitempty"`
	TopLevel		[]WorkflowDSLFieldSpec		`json:"top_level"`
	FlowKinds		[]WorkflowDSLKindSpec		`json:"flow_kinds"`
	NodeKinds		[]WorkflowDSLKindSpec		`json:"node_kinds"`
	SemanticRules		[]WorkflowDSLSemanticRule	`json:"semantic_rules,omitempty"`
	TemplateContract	WorkflowDSLTemplateContract	`json:"template_contract,omitempty"`
	JSONSchema		map[string]interface{}		`json:"json_schema"`
}
```

### WorkflowDSLDefinitionValidationError

WorkflowDSLDefinitionValidationError collects DSL definition contract issues.

```go
// WorkflowDSLDefinitionValidationError collects DSL definition contract issues.
type WorkflowDSLDefinitionValidationError struct {
	Issues []string
}
```

### WorkflowDSLFieldSpec

WorkflowDSLFieldSpec describes one structured field in DSL metadata.

```go
// WorkflowDSLFieldSpec describes one structured field in DSL metadata.
type WorkflowDSLFieldSpec struct {
	Name		string			`json:"name"`
	Type		string			`json:"type"`
	Required	bool			`json:"required,omitempty"`
	Description	string			`json:"description,omitempty"`
	AllowedValues	[]string		`json:"allowed_values,omitempty"`
	Fields		[]WorkflowDSLFieldSpec	`json:"fields,omitempty"`
}
```

### WorkflowDSLKindSpec

WorkflowDSLKindSpec describes one DSL flow kind or node kind.

```go
// WorkflowDSLKindSpec describes one DSL flow kind or node kind.
type WorkflowDSLKindSpec struct {
	Name		string		`json:"name"`
	Description	string		`json:"description,omitempty"`
	RequiredFields	[]string	`json:"required_fields,omitempty"`
	OptionalFields	[]string	`json:"optional_fields,omitempty"`
	ExampleYAML	string		`json:"example_yaml,omitempty"`
}
```

### WorkflowDSLRepairConstraints

WorkflowDSLRepairConstraints defines hard constraints that the repair agent must obey.

```go
// WorkflowDSLRepairConstraints defines hard constraints that the repair agent must obey.
type WorkflowDSLRepairConstraints struct {
	MaxOutputDSLBytes	int		`json:"max_output_dsl_bytes"`
	AllowedNodeIDs		[]string	`json:"allowed_node_ids,omitempty"`
	ForbiddenNodeIDs	[]string	`json:"forbidden_node_ids,omitempty"`
	PreserveNodeNames	[]string	`json:"preserve_node_names,omitempty"`
	PreserveWorkflowName	bool		`json:"preserve_workflow_name,omitempty"`
	PreserveRootNet		bool		`json:"preserve_root_net,omitempty"`
	MustKeepSemantics	[]string	`json:"must_keep_semantics,omitempty"`
}
```

### WorkflowDSLRepairContext

WorkflowDSLRepairContext is the canonical repair-phase context contract.

```go
// WorkflowDSLRepairContext is the canonical repair-phase context contract.
type WorkflowDSLRepairContext struct {
	WorkspaceID		string				`json:"workspace_id"`
	Goal			string				`json:"goal,omitempty"`
	UserIntake		WorkflowUserIntakeContext	`json:"user_intake,omitempty"`
	RunContext		WorkflowRunContext		`json:"run_context"`
	CurrentDSLYAML		string				`json:"current_dsl_yaml"`
	DSLDefinition		WorkflowDSLDefinition		`json:"dsl_definition"`
	WorkItems		[]WorkflowWorkItemCapability	`json:"workitems"`
	CompileDiagnostics	[]DSLCompileDiagnostic		`json:"compile_diagnostics,omitempty"`
	RuntimeFailures		[]WorkflowRuntimeFailure	`json:"runtime_failures,omitempty"`
	TestResults		[]WorkflowDSLTestResult		`json:"test_results,omitempty"`
	PreviousAttempts	[]WorkflowRepairAttempt		`json:"previous_attempts,omitempty"`
	Constraints		WorkflowDSLRepairConstraints	`json:"constraints"`
}
```

### WorkflowDSLSemanticRule

WorkflowDSLSemanticRule captures semantic constraints that are hard to express in plain schema.

```go
// WorkflowDSLSemanticRule captures semantic constraints that are hard to express in plain schema.
type WorkflowDSLSemanticRule struct {
	Code		string	`json:"code"`
	Description	string	`json:"description"`
	Scope		string	`json:"scope,omitempty"`
	Suggestion	string	`json:"suggestion,omitempty"`
}
```

### WorkflowDSLTemplateContextRoot

WorkflowDSLTemplateContextRoot describes one allowed root object in template interpolation context.
Agents frequently misuse these roots because the differences between data / vars / state are
non-obvious; the extra fields on this struct let the prompt spell out the exact semantics so
the agent picks the right root on the first try.

```go
// WorkflowDSLTemplateContextRoot describes one allowed root object in template interpolation context.
// Agents frequently misuse these roots because the differences between data / vars / state are
// non-obvious; the extra fields on this struct let the prompt spell out the exact semantics so
// the agent picks the right root on the first try.
type WorkflowDSLTemplateContextRoot struct {
	Name		string	`json:"name"`
	Description	string	`json:"description,omitempty"`
	ExamplePath	string	`json:"example_path,omitempty"`
	// Lifetime names how long values under this root live and which nodes can see them.
	// Example: "per-node (resets to immediate upstream's output each step)" for data.
	Lifetime	string	`json:"lifetime,omitempty"`
	// PopulatedBy documents the mechanism that writes values into this root so the agent
	// knows whether it has to do anything to make the value appear. Example: "engine
	// auto-flow: upstream.output → downstream.data" for data.
	PopulatedBy	string	`json:"populated_by,omitempty"`
	// WhenToUse lists the canonical scenarios for this root.
	WhenToUse	[]string	`json:"when_to_use,omitempty"`
	// WhenNotToUse lists anti-patterns the agent should NOT produce under this root.
	WhenNotToUse	[]string	`json:"when_not_to_use,omitempty"`
	// Gotchas are short warnings about common misuses. Keep each one under 120 chars.
	Gotchas	[]string	`json:"gotchas,omitempty"`
}
```

### WorkflowDSLTemplateContract

WorkflowDSLTemplateContract defines runtime interpolation semantics from mowl engine.

```go
// WorkflowDSLTemplateContract defines runtime interpolation semantics from mowl engine.
type WorkflowDSLTemplateContract struct {
	Syntax			string					`json:"syntax,omitempty"`
	ExpressionLanguage	string					`json:"expression_language,omitempty"`
	DelimiterOpen		string					`json:"delimiter_open,omitempty"`
	DelimiterClose		string					`json:"delimiter_close,omitempty"`
	InputContextRoots	[]WorkflowDSLTemplateContextRoot	`json:"input_context_roots,omitempty"`
	// VarMapFields documents the per-node DSL keys (input / output / vars / save / stream).
	// These are written in the DSL itself — in contrast to InputContextRoots which only
	// appear inside `{{...}}` templates.
	VarMapFields		[]WorkflowDSLVarMapField	`json:"var_map_fields,omitempty"`
	ExcludedVarPrefixes	[]string			`json:"excluded_var_prefixes,omitempty"`
	WholeExprReturnType	string				`json:"whole_expr_return_type,omitempty"`
	MixedTextReturnType	string				`json:"mixed_text_return_type,omitempty"`
	RuntimeErrorModes	[]string			`json:"runtime_error_modes,omitempty"`
	ValidExamples		[]string			`json:"valid_examples,omitempty"`
	InvalidExamples		[]string			`json:"invalid_examples,omitempty"`
	RuntimeSourceFiles	[]string			`json:"runtime_source_files,omitempty"`
	RuntimeSourceEntrypoint	string				`json:"runtime_source_entrypoint,omitempty"`
}
```

### WorkflowDSLTestResult

WorkflowDSLTestResult captures one test case result for repair feedback.

```go
// WorkflowDSLTestResult captures one test case result for repair feedback.
type WorkflowDSLTestResult struct {
	Name		string	`json:"name"`
	Passed		bool	`json:"passed"`
	TaskID		string	`json:"task_id,omitempty"`
	CaseID		string	`json:"case_id,omitempty"`
	InputJSON	string	`json:"input_json,omitempty"`
	ExpectedJSON	string	`json:"expected_json,omitempty"`
	ActualJSON	string	`json:"actual_json,omitempty"`
	Assertion	string	`json:"assertion,omitempty"`
	ErrorMessage	string	`json:"error_message,omitempty"`
}
```

### WorkflowDSLVarMapField

WorkflowDSLVarMapField describes one field inside a node's VarMap (input / output / vars /
save / stream). These are distinct from the top-level template context roots because they
are written INTO the DSL — the agent needs to know which ones to use and how they compose
with data/vars/state at runtime.

```go
// WorkflowDSLVarMapField describes one field inside a node's VarMap (input / output / vars /
// save / stream). These are distinct from the top-level template context roots because they
// are written INTO the DSL — the agent needs to know which ones to use and how they compose
// with data/vars/state at runtime.
type WorkflowDSLVarMapField struct {
	Name		string	`json:"name"`
	Description	string	`json:"description,omitempty"`
	// ExampleYAML is a short one- or two-line YAML snippet showing the field in context.
	ExampleYAML	string	`json:"example_yaml,omitempty"`
	// WhenToUse gives concrete scenarios where the agent should add this field.
	WhenToUse	[]string	`json:"when_to_use,omitempty"`
	// Gotchas calls out the most common misuses in ≤ 120 chars each.
	Gotchas	[]string	`json:"gotchas,omitempty"`
}
```

### WorkflowDeploymentCron

WorkflowDeploymentCron configures a cron deployment.

```go
// WorkflowDeploymentCron configures a cron deployment.
type WorkflowDeploymentCron struct {
	CronExpression	string	`json:"cron_expression"`
	TaskName	string	`json:"task_name,omitempty"`
	DataJSON	string	`json:"data_json,omitempty"`
	VarsJSON	string	`json:"vars_json,omitempty"`
}
```

### WorkflowDeploymentDynamicInfo

WorkflowDeploymentDynamicInfo describes a published dynamic service.

```go
// WorkflowDeploymentDynamicInfo describes a published dynamic service.
type WorkflowDeploymentDynamicInfo struct {
	ServiceName	string				`json:"service_name"`
	Invoke		*WorkflowDeploymentInvokeInfo	`json:"invoke,omitempty"`
	Version		*mowl.WorkflowVersion		`json:"version,omitempty"`
}
```

### WorkflowDeploymentInvokeInfo

WorkflowDeploymentInvokeInfo describes how to invoke a dynamic service.

```go
// WorkflowDeploymentInvokeInfo describes how to invoke a dynamic service.
type WorkflowDeploymentInvokeInfo struct {
	Method		string		`json:"method"`
	Path		string		`json:"path"`
	ResultMode	string		`json:"result_mode"`
	BodyExample	map[string]any	`json:"body_example,omitempty"`
}
```

### WorkflowDeploymentRequest

WorkflowDeploymentRequest publishes or updates a workflow and its execution resource.

```go
// WorkflowDeploymentRequest publishes or updates a workflow and its execution resource.
type WorkflowDeploymentRequest struct {
	Name		string	`json:"name"`
	Description	string	`json:"description,omitempty"`
	DSLYAML		string	`json:"dsl_yaml"`
	ExecutionMode	string	`json:"execution_mode"`
	DataJSON	string	`json:"data_json,omitempty"`
	VarsJSON	string	`json:"vars_json,omitempty"`

	WorkflowID		string	`json:"workflow_id,omitempty"`
	SourceType		string	`json:"source_type,omitempty"`
	Status			string	`json:"status,omitempty"`
	Goal			string	`json:"goal,omitempty"`
	PlannerModel		string	`json:"planner_model,omitempty"`
	DraftID			string	`json:"draft_id,omitempty"`
	SessionID		string	`json:"session_id,omitempty"`
	CandidateID		string	`json:"candidate_id,omitempty"`
	InputFormJSON		string	`json:"runtime_fields_json,omitempty"`
	FormLayoutJSON		string	`json:"runtime_layout_json,omitempty"`
	DefaultValuesJSON	string	`json:"default_values_json,omitempty"`
	DesignGraphJSON		string	`json:"design_graph_json,omitempty"`
	RunContextJSON		string	`json:"run_context_json,omitempty"`
	DeploymentConfigJSON	string	`json:"deployment_config_json,omitempty"`
	ComputeResourceID	string	`json:"compute_resource_id,omitempty"`

	VolumeTrigger	*WorkflowDeploymentVolumeTrigger	`json:"volume_trigger,omitempty"`
	Cron		*WorkflowDeploymentCron			`json:"cron,omitempty"`
	DynamicService	*WorkflowDeploymentDynamicService	`json:"dynamic_service,omitempty"`
}
```

### WorkflowDeploymentResponse

WorkflowDeploymentResponse wraps the deployment result and post-commit warnings.

```go
// WorkflowDeploymentResponse wraps the deployment result and post-commit warnings.
type WorkflowDeploymentResponse struct {
	Deployment	*WorkflowDeploymentResult	`json:"deployment"`
	Warnings	[]string			`json:"warnings,omitempty"`
}
```

### WorkflowDeploymentResult

WorkflowDeploymentResult describes the published workflow version and mode resource.

```go
// WorkflowDeploymentResult describes the published workflow version and mode resource.
type WorkflowDeploymentResult struct {
	WorkflowAppID		string				`json:"workflow_app_id,omitempty"`
	WorkflowDefinitionID	string				`json:"workflow_def_id"`
	WorkflowVersionID	string				`json:"workflow_version_id"`
	WorkflowName		string				`json:"workflow_name"`
	Version			int32				`json:"version"`
	ExecutionMode		string				`json:"execution_mode"`
	PreviousVersionID	string				`json:"previous_workflow_version_id,omitempty"`
	DisabledCronTaskIDs	[]string			`json:"disabled_cron_task_ids,omitempty"`
	VolumeTrigger		*catalog.VolumeWorkflowTrigger	`json:"volume_trigger,omitempty"`
	CronTask		*mowl.Task			`json:"cron_task,omitempty"`
	DynamicService		*WorkflowDeploymentDynamicInfo	`json:"dynamic_service,omitempty"`
}
```

### WorkflowDeploymentVolumeTrigger

WorkflowDeploymentVolumeTrigger configures a volume-trigger deployment.

```go
// WorkflowDeploymentVolumeTrigger configures a volume-trigger deployment.
type WorkflowDeploymentVolumeTrigger struct {
	VolumeID		int64	`json:"volume_id"`
	Enabled			*bool	`json:"enabled,omitempty"`
	AutoDispatchEnabled	*bool	`json:"auto_dispatch_enabled,omitempty"`
	VarsJSON		string	`json:"vars_json,omitempty"`
	// MaxConcurrency limits concurrently active Mowl runtime cases for this trigger.
	// A slot is released when the case reaches a terminal mowl_case_status.
	MaxConcurrency	int32	`json:"max_concurrency,omitempty"`
}
```

### WorkflowDraftDetail

WorkflowDraftDetail contains the persisted workflow draft snapshot.

```go
// WorkflowDraftDetail contains the persisted workflow draft snapshot.
type WorkflowDraftDetail struct {
	WorkflowDraftSummary
	InitialDSLYAML		string	`json:"initial_dsl_yaml,omitempty"`
	LatestDSLYAML		string	`json:"latest_dsl_yaml,omitempty"`
	RunContextJSON		string	`json:"run_context_json,omitempty"`
	InputFormJSON		string	`json:"runtime_fields_json,omitempty"`
	SubmittedValuesJSON	string	`json:"submitted_values_json,omitempty"`
	CompileDiagnosticsJSON	string	`json:"compile_diagnostics_json,omitempty"`
	TestRunJSON		string	`json:"test_run_json,omitempty"`
	CandidatesJSON		string	`json:"candidates_json,omitempty"`
	ErrorMessage		string	`json:"error_message,omitempty"`
}
```

### WorkflowDraftSummary

WorkflowDraftSummary is a compact workflow draft projection.

```go
// WorkflowDraftSummary is a compact workflow draft projection.
type WorkflowDraftSummary struct {
	DraftID			string	`json:"draft_id"`
	SourceType		string	`json:"source_type"`
	Status			string	`json:"status"`
	Finalized		bool	`json:"finalized"`
	WorkflowID		string	`json:"workflow_id,omitempty"`
	Goal			string	`json:"goal,omitempty"`
	PlannerModel		string	`json:"planner_model,omitempty"`
	SessionStatus		string	`json:"session_status,omitempty"`
	LatestCandidateID	string	`json:"candidate_id,omitempty"`
	CreatedAt		string	`json:"created_at,omitempty"`
	UpdatedAt		string	`json:"updated_at,omitempty"`
}
```

### WorkflowExecutionCreateRequest

WorkflowExecutionCreateRequest creates a workflow execution.

```go
// WorkflowExecutionCreateRequest creates a workflow execution.
type WorkflowExecutionCreateRequest struct {
	InputPayloadJSON	string	`json:"input_payload_json,omitempty"`
	VarsPayloadJSON		string	`json:"vars_payload_json,omitempty"`
	TriggerNow		*bool	`json:"trigger_now,omitempty"`
	RunOnce			bool	`json:"run_once,omitempty"`
	ComputeResourceID	string	`json:"compute_resource_id,omitempty"`
}
```

### WorkflowExecutionDetail

WorkflowExecutionDetail describes a product-level execution and linked mowl task/case.

```go
// WorkflowExecutionDetail describes a product-level execution and linked mowl task/case.
type WorkflowExecutionDetail struct {
	ExecutionID		string		`json:"execution_id"`
	DispatchJobID		string		`json:"dispatch_job_id,omitempty"`
	WorkflowID		string		`json:"workflow_id"`
	WorkflowName		string		`json:"workflow_name,omitempty"`
	Status			string		`json:"status"`
	ExecutionMode		string		`json:"execution_mode"`
	CronExpression		string		`json:"cron_expression,omitempty"`
	InputPayloadJSON	string		`json:"input_payload_json,omitempty"`
	VarsPayloadJSON		string		`json:"vars_payload_json,omitempty"`
	MoiTaskID		string		`json:"moi_task_id,omitempty"`
	MoiCaseID		string		`json:"moi_case_id,omitempty"`
	MoiWorkflowDefID	string		`json:"moi_workflow_def_id,omitempty"`
	MoiWorkflowVersion	string		`json:"moi_workflow_version_id,omitempty"`
	PauseScope		string		`json:"pause_scope,omitempty"`
	Error			string		`json:"error,omitempty"`
	DataName		string		`json:"data_name,omitempty"`
	CaseResult		string		`json:"case_result,omitempty"`
	CaseError		string		`json:"case_error,omitempty"`
	StartedAt		string		`json:"started_at,omitempty"`
	EndedAt			string		`json:"ended_at,omitempty"`
	CreatedAt		string		`json:"created_at,omitempty"`
	UpdatedAt		string		`json:"updated_at,omitempty"`
	InputPayload		map[string]any	`json:"input_payload,omitempty"`
	VarsPayload		map[string]any	`json:"vars_payload,omitempty"`
}
```

### WorkflowExecutionEnvelope

WorkflowExecutionEnvelope wraps a workflow execution detail response.

```go
// WorkflowExecutionEnvelope wraps a workflow execution detail response.
type WorkflowExecutionEnvelope struct {
	Execution WorkflowExecutionDetail `json:"execution"`
}
```

### WorkflowExecutionListRequest

WorkflowExecutionListRequest filters workflow executions.

```go
// WorkflowExecutionListRequest filters workflow executions.
type WorkflowExecutionListRequest struct {
	WorkflowID	string
	Offset		int
	Limit		int
	Status		string
	ExecutionMode	string
	WorkflowName	string
}
```

### WorkflowExecutionListResponse

WorkflowExecutionListResponse contains workflow execution details.

```go
// WorkflowExecutionListResponse contains workflow execution details.
type WorkflowExecutionListResponse struct {
	Total		int				`json:"total"`
	Executions	[]WorkflowExecutionDetail	`json:"executions"`
}
```

### WorkflowNotifyHandler

WorkflowNotifyHandler is the handler function type for workflow-level notifications.

```go
// WorkflowNotifyHandler is the handler function type for workflow-level notifications.
type WorkflowNotifyHandler func(ctx context.Context, notification *mowl.WorkflowNotification)
```

### WorkflowParameterSummary

WorkflowParameterSummary summarizes workflow app input parameter completeness.

```go
// WorkflowParameterSummary summarizes workflow app input parameter completeness.
type WorkflowParameterSummary struct {
	Status				string		`json:"status"`
	TotalFields			int		`json:"total_fields"`
	RequiredFields			int		`json:"required_fields"`
	FilledRequiredFields		int		`json:"filled_required_fields"`
	MissingRequiredFields		int		`json:"missing_required_fields"`
	MissingRequiredFieldIDs		[]string	`json:"missing_required_field_ids,omitempty"`
	MissingRequiredFieldLabels	[]string	`json:"missing_required_field_labels,omitempty"`
}
```

### WorkflowRepairAttempt

WorkflowRepairAttempt records previous repair iteration outcomes for anti-looping hints.

```go
// WorkflowRepairAttempt records previous repair iteration outcomes for anti-looping hints.
type WorkflowRepairAttempt struct {
	Iteration	int	`json:"iteration"`
	ChangeSummary	string	`json:"change_summary,omitempty"`
	Outcome		string	`json:"outcome,omitempty"`
	ErrorMessage	string	`json:"error_message,omitempty"`
}
```

### WorkflowRunContext

WorkflowRunContext is the execution-input contract used after DSL generation.
It tells the caller how to prepare WithTaskData/WithTaskVars and smoke test data.

```go
// WorkflowRunContext is the execution-input contract used after DSL generation.
// It tells the caller how to prepare WithTaskData/WithTaskVars and smoke test data.
type WorkflowRunContext struct {
	ExecutionMode	string			`json:"execution_mode"`
	TaskDataSchema	string			`json:"task_data_schema"`
	TaskVarsSchema	string			`json:"task_vars_schema"`
	InputTemplate	string			`json:"input_template"`
	VarsTemplate	string			`json:"vars_template"`
	TestCases	[]WorkflowRunTestCase	`json:"test_cases,omitempty"`
	RuntimeOptions	WorkflowRunOptions	`json:"runtime_options,omitempty"`
}
```

### WorkflowRunContextValidationError

WorkflowRunContextValidationError captures run-context contract issues.

```go
// WorkflowRunContextValidationError captures run-context contract issues.
type WorkflowRunContextValidationError struct {
	Issues []string
}
```

### WorkflowRunOptions

WorkflowRunOptions describes execution-level knobs for runtime invocation.

```go
// WorkflowRunOptions describes execution-level knobs for runtime invocation.
type WorkflowRunOptions struct {
	Transient	bool	`json:"transient,omitempty"`
	TraceLevel	string	`json:"trace_level,omitempty"`
	ResultMode	string	`json:"result_mode,omitempty"`
	TimeoutSeconds	int	`json:"timeout_seconds,omitempty"`
}
```

### WorkflowRunTestCase

WorkflowRunTestCase captures one runnable input case for workflow execution.

```go
// WorkflowRunTestCase captures one runnable input case for workflow execution.
type WorkflowRunTestCase struct {
	Name		string		`json:"name"`
	InputJSON	string		`json:"input_json"`
	VarsJSON	string		`json:"vars_json,omitempty"`
	ExpectedStatus	string		`json:"expected_status,omitempty"`
	Assertions	[]string	`json:"assertions,omitempty"`
}
```

### WorkflowRuntimeFailure

WorkflowRuntimeFailure captures runtime failure evidence from task/case execution.

```go
// WorkflowRuntimeFailure captures runtime failure evidence from task/case execution.
type WorkflowRuntimeFailure struct {
	TaskID		string	`json:"task_id,omitempty"`
	CaseID		string	`json:"case_id,omitempty"`
	Node		string	`json:"node,omitempty"`
	WorkItemID	string	`json:"workitem_id,omitempty"`
	Status		string	`json:"status,omitempty"`
	ErrorMessage	string	`json:"error_message,omitempty"`
	TraceSummary	string	`json:"trace_summary,omitempty"`
}
```

### WorkflowTriggerSummary

WorkflowTriggerSummary summarizes workflow app trigger configuration.

```go
// WorkflowTriggerSummary summarizes workflow app trigger configuration.
type WorkflowTriggerSummary struct {
	Mode		string	`json:"mode"`
	Configured	bool	`json:"configured"`
	Enabled		bool	`json:"enabled"`
	VolumeID	int64	`json:"volume_id,omitempty"`
	CronExpression	string	`json:"cron_expression,omitempty"`
	ServiceName	string	`json:"service_name,omitempty"`
}
```

### WorkflowUserIntakeContext

WorkflowUserIntakeContext captures user-collected requirement information
that should be carried through generate/compile/test/repair loops.

```go
// WorkflowUserIntakeContext captures user-collected requirement information
// that should be carried through generate/compile/test/repair loops.
type WorkflowUserIntakeContext struct {
	OriginalRequest		string		`json:"original_request,omitempty"`
	ConfirmedRequirements	[]string	`json:"confirmed_requirements,omitempty"`
	ProvidedConstraints	[]string	`json:"provided_constraints,omitempty"`
	AcceptanceCriteria	[]string	`json:"acceptance_criteria,omitempty"`
	OpenQuestions		[]string	`json:"open_questions,omitempty"`
	Assumptions		[]string	`json:"assumptions,omitempty"`
}
```

### WorkflowWorkItemCapability

WorkflowWorkItemCapability is the compact workitem context sent to the repair agent.

```go
// WorkflowWorkItemCapability is the compact workitem context sent to the repair agent.
type WorkflowWorkItemCapability struct {
	NodeID		string		`json:"node_id"`
	Version		string		`json:"version,omitempty"`
	Visibility	string		`json:"visibility,omitempty"`
	DisplayName	string		`json:"display_name,omitempty"`
	DisplayGroup	string		`json:"display_group,omitempty"`
	Description	string		`json:"description,omitempty"`
	SemanticSummary	string		`json:"semantic_summary,omitempty"`
	SideEffectClass	string		`json:"side_effect_class,omitempty"`
	Idempotence	string		`json:"idempotence,omitempty"`
	RequiredFields	[]string	`json:"required_fields,omitempty"`
	Stream		bool		`json:"stream"`
	InputSchema	string		`json:"input_schema,omitempty"`
	OutputSchema	string		`json:"output_schema,omitempty"`
	// InputUISchema is the normalized UI contract authored by the workitem. Workflow-agent
	// tools use it as the deterministic source for user-facing fields such as catalog_picker
	// resource_type. It is intentionally separate from InputSchema so planner logic never
	// guesses UI controls from raw JSON field names.
	InputUISchema	*WorkItemUISchema	`json:"input_ui_schema,omitempty"`
	// RuntimeConfigContract declares workitem input paths that must be exposed or considered
	// at run time. The planner and validators consume this generically; it is not UI metadata.
	RuntimeConfigContract	[]*RuntimeConfigParam	`json:"runtime_config_contract,omitempty"`
	NodeRole		string			`json:"node_role,omitempty"`
	FinalOutput		bool			`json:"final_output,omitempty"`
	Tags			[]string		`json:"tags,omitempty"`
}
```

### Workspace

Workspace represents a workspace in the system.
Re-exported from model/workspace for convenience.

```go
// Workspace represents a workspace in the system.
// Re-exported from model/workspace for convenience.
type Workspace = workspace.Workspace
```

### WorkspaceCreatedCallback

WorkspaceCreatedCallback is called when a workspace is successfully created.
This allows external code to track created workspaces for cleanup purposes.

```go
// WorkspaceCreatedCallback is called when a workspace is successfully created.
// This allows external code to track created workspaces for cleanup purposes.
type WorkspaceCreatedCallback func(ws *workspace.Workspace)
```

### WorkspaceInvitation

WorkspaceInvitation describes a Core-owned invitation and its target workspace projection.

```go
// WorkspaceInvitation describes a Core-owned invitation and its target workspace projection.
type WorkspaceInvitation struct {
	ID			string		`json:"id"`
	WorkspaceID		string		`json:"workspace_id"`
	WorkspaceName		string		`json:"workspace_name"`
	WorkspaceStatus		int32		`json:"workspace_status"`
	OwnerID			string		`json:"owner_id"`
	TargetUserID		string		`json:"target_user_id"`
	InvitedByUserID		string		`json:"invited_by_user_id"`
	Status			string		`json:"status"`
	InitialRoleIDs		[]string	`json:"initial_role_ids"`
	DefaultRoleID		string		`json:"default_role_id"`
	MemberAlias		string		`json:"member_alias"`
	MemberDescription	string		`json:"member_description"`
	CreatedAt		int64		`json:"created_at"`
}
```

### WorkspaceInvitationSubjectAttribute

WorkspaceInvitationSubjectAttribute assigns one Core-owned subject attribute value when an invitation is accepted.

```go
// WorkspaceInvitationSubjectAttribute assigns one Core-owned subject attribute value when an invitation is accepted.
type WorkspaceInvitationSubjectAttribute struct {
	AttributeID	int64	`json:"attribute_id"`
	Value		string	`json:"value"`
}
```

### WorkspaceNameProjection

WorkspaceNameProjection is a display-only workspace identity. It does not
establish membership or authorize access to the projected workspace.

```go
// WorkspaceNameProjection is a display-only workspace identity. It does not
// establish membership or authorize access to the projected workspace.
type WorkspaceNameProjection struct {
	ID	string	`json:"id"`
	Name	string	`json:"name"`
}
```

### WorkspaceNode

WorkspaceNode represents a workspace in the tree.

```go
// WorkspaceNode represents a workspace in the tree.
type WorkspaceNode struct {
	id		string
	name		string
	children	[]TreeNode
}
```

### WorkspaceStatus

WorkspaceStatus represents the status of a workspace.
Re-exported from model/workspace for convenience.

```go
// WorkspaceStatus represents the status of a workspace.
// Re-exported from model/workspace for convenience.
type WorkspaceStatus = workspace.WorkspaceStatus
```

### WorkspaceUser

WorkspaceUser represents a user's access to a workspace.
Re-exported from model/workspace for convenience.

```go
// WorkspaceUser represents a user's access to a workspace.
// Re-exported from model/workspace for convenience.
type WorkspaceUser = workspace.WorkspaceUser
```
