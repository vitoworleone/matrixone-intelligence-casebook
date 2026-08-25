// Package config provides configuration management for the moi-core catalog service.
// It supports loading configuration from TOML files with environment variable substitution.
package config

import (
	"fmt"
	"strings"
	"time"

	billingconfig "github.com/matrixflow/moi-core/catalog/pkg/billing/config"
)

// Config is the main configuration structure for the catalog service.
type Config struct {
	Server              ServerConfig              `toml:"server"`
	Database            DatabaseConfig            `toml:"database"`
	Provider            ProviderConfig            `toml:"provider"`
	Storage             StorageConfig             `toml:"storage"`
	Logging             LoggingConfig             `toml:"logging"`
	Metrics             MetricsConfig             `toml:"metrics"`
	Pprof               PprofConfig               `toml:"pprof"`
	Mowl                MowlConfig                `toml:"mowl"`
	Runtime             RuntimeConfig             `toml:"runtime"`
	Parser              ParserConfig              `toml:"parser"`
	Compute             ComputeConfig             `toml:"compute_resource"`
	AgentRuntime        AgentRuntimeConfig        `toml:"agent_runtime"`
	HTTPWorkItem        HTTPWorkItemConfig        `toml:"http_workitem"`
	Worker              WorkerConfig              `toml:"worker"`
	Explore             ExploreConfig             `toml:"explore"`
	TaaS                TaaSConfig                `toml:"taas"`
	PersonalAccessToken PersonalAccessTokenConfig `toml:"personal_access_token"`
	TenantDBPool        TenantDBPoolConfig        `toml:"tenant_db_pool"`
	Reconcile           ReconcileConfig           `toml:"reconcile"`
	Upgrade             UpgradeConfig             `toml:"upgrade"`
	Garbage             GarbageConfig             `toml:"garbage"`
	AccountDeletion     AccountDeletionConfig     `toml:"workspace_account_deletion_recovery"`
	DataconnSecret      DataconnSecretConfig      `toml:"dataconn_secret"`
	WebSearch           WebSearchConfig           `toml:"web_search"`
	WebOpen             WebOpenConfig             `toml:"web_open"`
	Billing             billingconfig.Config      `toml:"billing"`
}

// ParserConfig controls Catalog-owned parser behavior.
type ParserConfig struct {
	MinerU MinerUParserConfig `toml:"mineru"`
}

// MinerUParserConfig controls MinerU-specific parser behavior.
type MinerUParserConfig struct {
	PDFSplit MinerUPDFSplitConfig `toml:"pdf_split"`
}

// MinerUPDFSplitConfig controls PDF splitting before direct MinerU requests.
type MinerUPDFSplitConfig struct {
	ThresholdPages int32 `toml:"threshold_pages"`
	PagesPerChunk  int32 `toml:"pages_per_chunk"`
}

// WebSearchConfig controls the moi_web_search platform tool.
type WebSearchConfig struct {
	Enabled                       bool                  `toml:"enabled"`
	Backend                       string                `toml:"backend"` // default "auto"; also bocha, tavily, brave, mcp
	CredentialEncryptionKeyID     string                `toml:"credential_encryption_key_id"`
	CredentialEncryptionKeyBase64 string                `toml:"credential_encryption_key_base64"`
	MCP                           WebSearchMCPConfig    `toml:"mcp"`
	Bocha                         WebSearchBochaConfig  `toml:"bocha"`
	Tavily                        WebSearchTavilyConfig `toml:"tavily"`
	Brave                         WebSearchBraveConfig  `toml:"brave"`
}

// WebSearchMCPConfig configures the MCP web provider backend.
type WebSearchMCPConfig struct {
	Adapter        string `toml:"adapter"`         // default "exa"
	Endpoint       string `toml:"endpoint"`        // default Exa hosted MCP
	SearchTool     string `toml:"search_tool"`     // default web_search_advanced_exa
	TimeoutSeconds int    `toml:"timeout_seconds"` // 0 → 30
}

// WebOpenConfig controls the moi_web_open platform tool independently from
// web search so deployments can combine different providers.
type WebOpenConfig struct {
	Enabled bool             `toml:"enabled"`
	Backend string           `toml:"backend"` // default "mcp"
	MCP     WebOpenMCPConfig `toml:"mcp"`
}

// WebOpenMCPConfig configures the MCP page-read backend.
type WebOpenMCPConfig struct {
	Adapter        string `toml:"adapter"`         // default "exa"
	Endpoint       string `toml:"endpoint"`        // default Exa hosted MCP
	OpenTool       string `toml:"open_tool"`       // default web_fetch_exa
	TimeoutSeconds int    `toml:"timeout_seconds"` // 0 → 30
}

// WebSearchBochaConfig configures the Bocha Web Search API backend. The API
// key is deployment-owned and must be injected through the environment; it
// never enters Agent/Workspace state, tool output, logs or traces.
type WebSearchBochaConfig struct {
	Endpoint       string `toml:"endpoint"`        // default https://api.bochaai.com/v1/web-search
	APIKey         string `toml:"api_key"`         // required when backend = "bocha"; supports ${VAR}
	TimeoutSeconds int    `toml:"timeout_seconds"` // 0 → 30
}

// WebSearchTavilyConfig configures the Tavily Search API backend. The API
// key is deployment-owned and must be injected through the environment; it
// never enters Agent/Workspace state, tool output, logs or traces.
type WebSearchTavilyConfig struct {
	Endpoint       string `toml:"endpoint"`        // default https://api.tavily.com/search
	APIKey         string `toml:"api_key"`         // required when backend = "tavily"; supports ${VAR}
	TimeoutSeconds int    `toml:"timeout_seconds"` // 0 → 30
}

// WebSearchBraveConfig configures the Brave Web Search API backend. The API
// key is deployment-owned and must be injected through the environment; it
// never enters Agent/Workspace state, tool output, logs or traces.
type WebSearchBraveConfig struct {
	Endpoint       string `toml:"endpoint"`        // default https://api.search.brave.com/res/v1/web/search
	APIKey         string `toml:"api_key"`         // required when backend = "brave"; supports ${VAR}
	TimeoutSeconds int    `toml:"timeout_seconds"` // 0 → 30
}

// DataconnSecretConfig controls the Catalog-owned connector secret store.
type DataconnSecretConfig struct {
	Enabled              bool   `toml:"enabled"`
	EncryptionKeyID      string `toml:"encryption_key_id"`
	EncryptionKeyBase64  string `toml:"encryption_key_base64"`
	AttestationKeyID     string `toml:"attestation_key_id"`
	AttestationKeyBase64 string `toml:"attestation_key_base64"`
}

// UpgradeConfig controls catalog auto-upgrade behavior.
type UpgradeConfig struct {
	CompatibilityCheckEnabled bool `toml:"compatibility_check_enabled"`
	BootstrapEnabled          bool `toml:"bootstrap_enabled"`
	TenantGuardEnabled        bool `toml:"tenant_guard_enabled"`
}

// GarbageConfig controls catalog file garbage collection.
type GarbageConfig struct {
	Enabled bool `toml:"enabled"`
}

// AccountDeletionConfig controls retry of MatrixOne account deletion after the
// corresponding workspace metadata has been removed.
type AccountDeletionConfig struct {
	IntervalSeconds int `toml:"interval_seconds"`
}

// ReconcileConfig controls the workflow_execution status reconcile worker.
// The worker periodically scans non-terminal executions, asks mowl for the
// engine truth via SignalWorkflow(CHECK), and writes terminal states back
// (with monotonic guard). See pkg/workflowapp/reconciler.
type ReconcileConfig struct {
	Enabled         bool          `toml:"enabled"`           // default true; set to false to disable (e.g. read-only replica or incident triage)
	Interval        time.Duration `toml:"interval"`          // scan tick interval, default 30s
	Cooldown        time.Duration `toml:"cooldown"`          // per-row cooldown after a reconcile, default 60s
	Grace           time.Duration `toml:"grace"`             // skip rows newer than this, default 60s
	BatchSize       int           `toml:"batch_size"`        // rows per scan, default 100
	PerCheckTimeout time.Duration `toml:"per_check_timeout"` // single SignalWorkflow(CHECK) timeout, default 5s
	StartupJitter   time.Duration `toml:"startup_jitter"`    // randomized initial delay to desynchronize replicas, default 10s
}

// DefaultReconcileConfig returns the recommended defaults.
//
// Enabled defaults to true: this keeps workflow_execution.status converged
// with the engine truth when the catalog callback workitem event is missed.
// Operators who need to disable it (e.g. during incident triage or on a
// replica that shouldn't write) can set [reconcile].enabled = false.
func DefaultReconcileConfig() ReconcileConfig {
	return ReconcileConfig{
		Enabled:         true,
		Interval:        30 * time.Second,
		Cooldown:        60 * time.Second,
		Grace:           60 * time.Second,
		BatchSize:       100,
		PerCheckTimeout: 5 * time.Second,
		StartupJitter:   10 * time.Second,
	}
}

// TenantDBPoolConfig 控制访问租户业务库（非 catalog 元数据库）的连接池超时。
// 跨公网/慢链路场景下应调高，否则使用 30s 默认值。
type TenantDBPoolConfig struct {
	ConnectionTimeoutSeconds int `toml:"connection_timeout_seconds"` // 0 → 30
	ReadTimeoutSeconds       int `toml:"read_timeout_seconds"`       // 0 → 30
	WriteTimeoutSeconds      int `toml:"write_timeout_seconds"`      // 0 → 30
}

// WorkerConfig contains catalog built-in worker settings.
type WorkerConfig struct {
	// MaxConcurrency is the maximum number of concurrent work item executions.
	// Defaults to runtime.NumCPU() * 4 when 0 or not set.
	MaxConcurrency int `toml:"max_concurrency"`

	// WorkerID overrides the catalog built-in worker identity (e.g. "${MOI_WORKER_ID}").
	// When empty, defaults to "catalog-builtin".
	WorkerID string `toml:"worker_id"`
}

// HTTPWorkItemConfig is the global HTTP client configuration for the HTTP WorkItem.
type HTTPWorkItemConfig struct {
	// Connection pool settings
	MaxIdleConns        int `toml:"max_idle_conns"`            // default 100
	MaxIdleConnsPerHost int `toml:"max_idle_conns_per_host"`   // default 10
	MaxConnsPerHost     int `toml:"max_conns_per_host"`        // default 0 (unlimited)
	IdleConnTimeout     int `toml:"idle_conn_timeout_seconds"` // default 90

	// Timeout settings
	ReadTimeout  int `toml:"read_timeout_seconds"`  // default 30
	WriteTimeout int `toml:"write_timeout_seconds"` // default 30
}

// ServerConfig contains HTTP and gRPC server settings.
type ServerConfig struct {
	Host         string `toml:"host"`          // Listen address, default "0.0.0.0"
	Port         int    `toml:"port"`          // HTTP port, default 8081
	GRPCPort     int    `toml:"grpc_port"`     // gRPC port, default 8082
	ReadTimeout  int    `toml:"read_timeout"`  // Read timeout in seconds, default 30
	WriteTimeout int    `toml:"write_timeout"` // Write timeout in seconds, default 30
	// AgentAutomationTaskDescriptionMaxRunes limits task descriptions by Unicode rune count.
	AgentAutomationTaskDescriptionMaxRunes int `toml:"agent_automation_task_description_max_runes"` // 1-16383, default 8192
}

// DatabaseConfig contains database connection settings.
type DatabaseConfig struct {
	Host            string `toml:"host"`
	Port            int    `toml:"port"`
	User            string `toml:"user"`
	Password        string `toml:"password"` // Supports env var ${DB_PASSWORD}
	Database        string `toml:"database"`
	ConnectTimeout  int    `toml:"connect_timeout"` // In seconds
	ReadTimeout     int    `toml:"read_timeout"`    // In seconds
	WriteTimeout    int    `toml:"write_timeout"`   // In seconds
	MaxOpenConns    int    `toml:"max_open_conns"`
	MaxIdleConns    int    `toml:"max_idle_conns"`
	ConnMaxLifetime int    `toml:"conn_max_lifetime"` // In seconds
}

// ProviderConfig contains database provider settings.
type ProviderConfig struct {
	Type  string              `toml:"type"`  // Provider type, e.g., "local"
	Local LocalProviderConfig `toml:"local"` // Local provider configuration
}

// LocalProviderConfig contains local database provider settings.
type LocalProviderConfig struct {
	Host string `toml:"host"` // Database host for local provider
	Port int    `toml:"port"` // Database port for local provider
}

// StorageConfig contains object storage settings.
type StorageConfig struct {
	Type               string `toml:"type"` // "s3" or "local"
	Endpoint           string `toml:"endpoint"`
	Bucket             string `toml:"bucket"`
	AccessKey          string `toml:"access_key"` // Supports env var
	SecretKey          string `toml:"secret_key"` // Supports env var
	Region             string `toml:"region"`
	UseSSL             bool   `toml:"use_ssl"`
	ForcePathStyle     bool   `toml:"force_path_style"`     // Use path-style URLs (required for MinIO and some S3-compatible services)
	IsMinio            bool   `toml:"is_minio"`             // Enable MinIO-specific authentication and optimizations
	InsecureSkipVerify bool   `toml:"insecure_skip_verify"` // Skip TLS certificate verification (for self-signed certs or IP-based endpoints)
	NoBucketValidation bool   `toml:"no_bucket_validation"` // Skip bucket existence check on startup (for S3 services with limited permissions)
}

// LoggingConfig contains logging settings.
type LoggingConfig struct {
	Level      string `toml:"level"`       // debug, info, warn, error
	Format     string `toml:"format"`      // json, text
	Output     string `toml:"output"`      // stdout, file
	FilePath   string `toml:"file_path"`   // Log file path
	MaxSize    int    `toml:"max_size"`    // Max file size in MB
	MaxBackups int    `toml:"max_backups"` // Number of backup files to keep
	MaxAge     int    `toml:"max_age"`     // Days to keep old files
}

// MetricsConfig contains Prometheus metrics settings.
type MetricsConfig struct {
	Enabled bool   `toml:"enabled"`
	Path    string `toml:"path"` // Default "/metrics"
}

// PprofConfig contains net/http/pprof server settings.
type PprofConfig struct {
	Enabled bool   `toml:"enabled"`
	Addr    string `toml:"addr"` // Default ":6061"
}

// MowlConfig contains Mowl Engine deployment settings.
type MowlConfig struct {
	// Embedded controls whether Mowl Engine runs inside the Catalog process.
	// When true, Mowl shares the HTTP port via cmux for gRPC multiplexing.
	// When false, ProxyEndpoint must be set to forward gRPC to an external Mowl process.
	Embedded bool `toml:"embedded"`

	// ProxyEndpoint is the gRPC address of the external Mowl Engine (e.g., "localhost:50051").
	// Only used when Embedded is false.
	ProxyEndpoint string `toml:"proxy_endpoint"`

	// RuntimeLease controls the embedded Mowl singleton runtime lease timing.
	// Only used when Embedded is true.
	RuntimeLease MowlRuntimeLeaseConfig `toml:"runtime_lease"`

	// TrustedWorkerTokens marks worker registrations that may own reserved
	// WorkItem namespaces. Empty means no worker can be trusted.
	TrustedWorkerTokens []MowlTrustedWorkerToken `toml:"trusted_worker_tokens"`
}

type MowlRuntimeLeaseConfig struct {
	TTLSeconds           int `toml:"ttl_seconds"`
	RenewIntervalSeconds int `toml:"renew_interval_seconds"`
}

type MowlTrustedWorkerToken struct {
	Profile string `toml:"profile"`
	TokenID string `toml:"token_id"`
	Token   string `toml:"token"`
}

// WorkerImageConfig represents one configured worker image.
//
// WorkerType is the stable scheduling key consumed by WorkItem.worker_type.
// ImageKey identifies a selectable image variant within that worker type.
// Kind and Type are the documented legacy aliases respectively.
type WorkerImageConfig struct {
	WorkerType         string `toml:"worker_type"`
	Kind               string `toml:"kind"`
	ImageKey           string `toml:"image_key"`
	Type               string `toml:"type"`
	Tag                string `toml:"tag"`
	Repository         string `toml:"repository"`
	Platform           string `toml:"platform"`
	Description        string `toml:"description"`
	Default            bool   `toml:"default"`
	DeploymentTemplate string `toml:"deployment_template"`
}

// NormalizeWorkerImageConfig resolves documented legacy aliases at the config
// boundary and rejects conflicting dual writes. A single legacy type entry is
// the historical one-variant form, so its image key is also its worker type.
func NormalizeWorkerImageConfig(img *WorkerImageConfig) error {
	if img == nil {
		return nil
	}
	workerType := strings.TrimSpace(img.WorkerType)
	kindAlias := strings.TrimSpace(img.Kind)
	if workerType != "" && kindAlias != "" && workerType != kindAlias {
		return fmt.Errorf("conflicting worker_type and kind: %q != %q", workerType, kindAlias)
	}
	if workerType == "" {
		workerType = kindAlias
	}

	imageKey := strings.TrimSpace(img.ImageKey)
	typeAlias := strings.TrimSpace(img.Type)
	if imageKey != "" && typeAlias != "" && imageKey != typeAlias {
		return fmt.Errorf("conflicting image_key and type: %q != %q", imageKey, typeAlias)
	}
	if imageKey == "" {
		imageKey = typeAlias
	}

	// A legacy type-only image represented both concepts. Keep that exact
	// compatibility mapping without inferring a prefix-based worker type.
	if workerType == "" {
		workerType = imageKey
	}
	if imageKey == "" {
		imageKey = workerType
	}
	if workerType == "" || imageKey == "" {
		return fmt.Errorf("worker image requires worker_type/image_key or legacy kind/type")
	}

	img.WorkerType = workerType
	img.Kind = workerType
	img.ImageKey = imageKey
	img.Type = imageKey
	return nil
}

// NormalizeWorkerImageConfigs normalizes the configured catalog image list in
// place before it is used by the runtime or synchronized into Catalog.
func NormalizeWorkerImageConfigs(images []WorkerImageConfig) error {
	for i := range images {
		if err := NormalizeWorkerImageConfig(&images[i]); err != nil {
			return fmt.Errorf("runtime.worker_images[%d]: %w", i, err)
		}
	}
	return nil
}

// ComputeResourceSpecAdminConfig controls who can manage compute resource specs.
type ComputeResourceSpecAdminConfig struct {
	Accounts []string `toml:"accounts"`
}

// ComputeResourceSpecConfig represents an initial compute resource spec entry.
type ComputeResourceSpecConfig struct {
	ID                string  `toml:"id"`
	Kind              string  `toml:"kind"`
	Family            string  `toml:"family"`
	FamilyName        string  `toml:"family_name"`
	FamilyNameEn      string  `toml:"family_name_en"`
	CPUMilli          int     `toml:"cpu_milli"`
	MemoryMiB         int     `toml:"memory_mib"`
	GPUCount          int     `toml:"gpu_count"`
	GPUMemoryMiB      int     `toml:"gpu_memory_mib"`
	GPUCores          int     `toml:"gpu_cores"`
	CreditPerHour     float64 `toml:"credit_per_hour"`
	Enabled           bool    `toml:"enabled"`
	Description       string  `toml:"description"`
	DescriptionEn     string  `toml:"description_en"`
	NodePlacementJSON string  `toml:"node_placement_json"`
}

// ComputeConfig contains ComputeResource catalog-managed settings.
type ComputeConfig struct {
	SpecAdmin ComputeResourceSpecAdminConfig `toml:"spec_admin"`
	Specs     []ComputeResourceSpecConfig    `toml:"specs"`
}

// AgentRuntimeConfig controls the agent-runtime service. It is independent from
// RuntimeConfig, which only manages workflow worker runtime providers.
type AgentRuntimeConfig struct {
	DevelopmentStubEnabled   bool   `toml:"development_stub_enabled"`
	RuntimeGrantKey          string `toml:"runtime_grant_key"`
	RuntimeGrantTTLSeconds   int    `toml:"runtime_grant_ttl_seconds"`
	CapabilityGatewayBaseURL string `toml:"capability_gateway_base_url"`
	// RuntimeFileTransferBaseURL is the base URL embedded in managed Edge
	// artifact-publish descriptors. It may differ from CapabilityGatewayBaseURL
	// when Astra and the Edge executor run in different network namespaces. Empty
	// disables managed Edge artifact publishing without preventing the rest of
	// the Astra runtime from starting; no other URL is used as fallback.
	RuntimeFileTransferBaseURL string             `toml:"runtime_file_transfer_base_url"`
	ProviderHMACKey            string             `toml:"provider_hmac_key"`
	Sandbox                    AgentSandboxConfig `toml:"sandbox"`
	AstraBinding               AstraBindingConfig `toml:"astra_binding"`
	AstraRuntime               AstraRuntimeConfig `toml:"astra_runtime"`
}

// AgentSandboxConfig controls the temporary Bash/Python sandbox capability.
// Production enablement is paired with moi-gitops charts/moi-core/values.yaml:
// global.agentSandbox configures the trusted Worker identity and staging quotas;
// sandbox-worker must run privileged with /dev/fuse and cgroup v2. Catalog,
// MOWL, and Worker must receive the same system/trust key. Missing or mismatched
// dependencies fail startup or capability attachment rather than degrading.
type AgentSandboxConfig struct {
	Enabled      bool  `toml:"enabled"`
	MaxFileBytes int64 `toml:"max_file_bytes"`
}

// AstraBindingConfig controls production Astra Agent Binding registration.
type AstraBindingConfig struct {
	Enabled        bool   `toml:"enabled"`
	Endpoint       string `toml:"endpoint"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
}

// AstraRuntimeConfig controls production Astra Agent Binding runtime turns.
// TimeoutSeconds is the maximum interval without response data on the Astra SSE stream.
type AstraRuntimeConfig struct {
	Enabled        bool   `toml:"enabled"`
	Endpoint       string `toml:"endpoint"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
	// JWTSecret is the shared HMAC key used to sign and verify per-user tokens
	// for sandbox chat requests. When set, chat requests carry a short-lived
	// per-user token instead of the system auth_token, so Astra can identify
	// the caller through the external authorize_request callback.
	JWTSecret string `toml:"jwt_secret"`
	// EphemeralSandboxBackendURL / EphemeralSandboxServiceKey wire the
	// moi-backend internal API for auto-provisioning conversation-bound
	// ephemeral Sandboxes (design: docs/sandbox/ephemeral-sandbox-lifecycle-design.md).
	// Both must be set to enable the feature; the per-workspace switch lives
	// in moi-backend (workspace_sandbox_quotas.auto_ephemeral_enabled).
	EphemeralSandboxBackendURL string `toml:"ephemeral_sandbox_backend_url"`
	EphemeralSandboxServiceKey string `toml:"ephemeral_sandbox_service_key"`
	// EphemeralSandboxWaitSeconds bounds how long a turn waits for a freshly
	// created Sandbox's edge agent to connect (default 45s).
	EphemeralSandboxWaitSeconds int `toml:"ephemeral_sandbox_wait_seconds"`
	// ExternalCatalogServiceKey authenticates astra's model-catalog scope
	// callbacks (POST /api/v1/astra/external-catalog: list_catalog_by_scope /
	// issue_runtime_context_by_scope). Empty disables the surface. Design:
	// docs/sandbox/runner-cli-default-model-design.md.
	ExternalCatalogServiceKey string `toml:"external_catalog_service_key"`
	// ExternalCatalogDefaultContextWindow is advertised as each workspace chat
	// model's limits.context_window to astra-cli. Workspace model config has no
	// per-model window; astra-cli rejects non-positive values. Default 128000.
	ExternalCatalogDefaultContextWindow int `toml:"external_catalog_default_context_window"`
}

// RuntimeConfig contains RuntimeProvider settings for dynamic worker management.
type RuntimeConfig struct {
	Enabled                 bool                `toml:"enabled"`
	Local                   LocalRuntimeConfig  `toml:"local"`
	Cloud                   CloudRuntimeConfig  `toml:"cloud"`
	GC                      GCConfig            `toml:"gc"`
	WorkerImages            []WorkerImageConfig `toml:"worker_images"`
	SyncWorkerImagesMaxWait time.Duration       `toml:"sync_worker_images_max_wait"`
}

// LocalRuntimeConfig contains settings for the local RuntimeProvider (Docker + binary).
type LocalRuntimeConfig struct {
	Enabled           bool     `toml:"enabled"`
	DockerSocket      string   `toml:"docker_socket"`
	BinarySearchPaths []string `toml:"binary_search_paths"`
	PythonWorkerCmd   string   `toml:"python_worker_cmd"`  // e.g. "python3"
	PythonWorkerArgs  []string `toml:"python_worker_args"` // e.g. ["workers/python-worker/cmd/python_worker.py"]
}

// CloudRuntimeConfig contains settings for the cloud RuntimeProvider (Kubernetes).
type CloudRuntimeConfig struct {
	Enabled         bool   `toml:"enabled"`
	Kubeconfig      string `toml:"kubeconfig"`
	Namespace       string `toml:"namespace"`
	ImagePullSecret string `toml:"image_pull_secret"`
	// WorkerEndpoint is the Mowl gRPC endpoint visible from launched Worker Pods.
	// When empty, Catalog uses its existing in-process endpoint selection.
	WorkerEndpoint       string `toml:"worker_endpoint"`
	GoWorkerTemplate     string `toml:"go_worker_template"`
	PythonWorkerTemplate string `toml:"python_worker_template"`
	CPUNodePlacementJSON string `toml:"cpu_node_placement_json"`
	GPUNodePlacementJSON string `toml:"gpu_node_placement_json"`
	// StartupTimeout is the maximum seconds to wait for a worker to register. Default 120.
	StartupTimeout int `toml:"startup_timeout"`
	// KeepFailedPodsDuration is how long (seconds) to keep Failed pods for debugging. Default 600.
	KeepFailedPodsDuration int `toml:"keep_failed_pods_duration"`
	// RestartPolicy for worker Pods. Valid: "Always", "OnFailure", "Never". Default "Always".
	RestartPolicy string `toml:"restart_policy"`
}

// GCConfig contains settings for the orphan worker garbage collection mechanism.
type GCConfig struct {
	Enabled     bool `toml:"enabled"`
	Interval    int  `toml:"interval"`     // Scan interval in seconds, default 120
	GracePeriod int  `toml:"grace_period"` // Grace period in seconds, default 300
}

// ExploreConfig contains Explore engine settings loaded from TOML.
// Zero values mean "use explore package defaults"; only non-zero fields override.
type ExploreConfig struct {
	DefaultLLM ExploreLLMConfig       `toml:"default_llm"`
	Retriever  ExploreRetrieverConfig `toml:"retriever"`
	Context    ExploreContextConfig   `toml:"context"`
	Direct     ExploreDirectConfig    `toml:"direct"`
	Budget     ExploreBudgetConfig    `toml:"budget"`
	Memory     ExploreMemoryConfig    `toml:"memory"`

	// SchemaCacheTTLSeconds overrides the schema cache TTL. Default: 300.
	SchemaCacheTTLSeconds int `toml:"schema_cache_ttl_seconds"`
	// SessionTitleTimeoutSeconds overrides the session title generation timeout. Default: 10.
	SessionTitleTimeoutSeconds int `toml:"session_title_timeout_seconds"`
}

type ExploreLLMConfig struct {
	Model                      string  `toml:"model"`
	Temperature                float64 `toml:"temperature"`
	MaxTokens                  int     `toml:"max_tokens"`
	ModelContextWindow         int     `toml:"model_context_window"`
	ReasoningMode              string  `toml:"reasoning_mode"`
	ReasoningEffortWhenEnabled string  `toml:"reasoning_effort_when_enabled"`
}

// ExploreRetrieverConfig contains retriever tuning parameters.
type ExploreRetrieverConfig struct {
	MaxRepairAttempts              int    `toml:"max_repair_attempts"`               // Default: 2
	SQLGenerationMaxTokens         int    `toml:"sql_generation_max_tokens"`         // Default: 4096
	SQLRepairMaxTokens             int    `toml:"sql_repair_max_tokens"`             // Default: 4096
	SQLResultsetResolverMaxTokens  int    `toml:"sql_resultset_resolver_max_tokens"` // Default: 16384
	RAGTopK                        int    `toml:"rag_top_k"`                         // Default: 5
	ConcurrencyLimit               int    `toml:"concurrency_limit"`                 // Default: 3
	RerankEndpoint                 string `toml:"rerank_endpoint"`
	RerankAPIKey                   string `toml:"rerank_api_key"`
	RerankModel                    string `toml:"rerank_model"`
	SubQuestionTimeoutSeconds      int    `toml:"sub_question_timeout_seconds"`       // Default: 60
	LookupResolverMaxSourceRows    int    `toml:"lookup_resolver_max_source_rows"`    // Default: 2000
	LookupResolverMaxReturnedCodes int    `toml:"lookup_resolver_max_returned_codes"` // Default: 24
	LookupIndexEmbeddingBatchSize  int    `toml:"lookup_index_embedding_batch_size"`  // Default: 128
	EnableSQLValidation            *bool  `toml:"enable_sql_validation"`              // Default: true
	EnableSQLDecomposition         *bool  `toml:"enable_sql_decomposition"`           // Default: true
}

// ExploreContextConfig contains context compression parameters.
type ExploreContextConfig struct {
	SchemaCacheTTLSeconds int `toml:"schema_cache_ttl_seconds"` // Default: 300
}

// ExploreDirectConfig contains Direct Retriever parameters.
type ExploreDirectConfig struct {
	MaxContentChars int `toml:"max_content_chars"` // Default: 32000
	MaxFileCount    int `toml:"max_file_count"`    // Default: 3
	MaxTotalBytes   int `toml:"max_total_bytes"`   // Default: 512000
}

// ExploreBudgetConfig contains token and time budget parameters.
type ExploreBudgetConfig struct {
	GuaranteedMinTokenRatio float64 `toml:"guaranteed_min_token_ratio"` // Default: 0.4
	GuaranteedMinTimeRatio  float64 `toml:"guaranteed_min_time_ratio"`  // Default: 0.5
	TokenEstimationStrategy string  `toml:"token_estimation_strategy"`  // Default: "chars_div_4"
}

// ExploreMemoryConfig contains session history parameters.
type ExploreMemoryConfig struct {
	MaxHistoryMessages  int  `toml:"max_history_messages"`  // Default: 10
	RecentWindow        int  `toml:"recent_window"`         // Default: 5
	HistoryTokenBudget  int  `toml:"history_token_budget"`  // Default: 2000
	EnableResultContext bool `toml:"enable_result_context"` // Default: true
	EnableSummary       bool `toml:"enable_summary"`        // Default: true
}

// TaaSConfig configures an independently deployed OpenAI-compatible TaaS model service.
type TaaSConfig struct {
	BaseURL        string `toml:"base_url"`
	APIKey         string `toml:"api_key"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
}

// PersonalAccessTokenConfig configures UC-backed user authentication for
// Catalog. The deployment credentials are only used for protected
// moi-backend endpoints; they are never a user PAT or Catalog API key.
type PersonalAccessTokenConfig struct {
	Enabled                   bool   `toml:"enabled"`
	UCIntrospectionURL        string `toml:"uc_introspection_url"`
	SubjectResolverURL        string `toml:"subject_resolver_url"`
	SubjectResolverCredential string `toml:"subject_resolver_credential"`
	TimeoutSeconds            int    `toml:"timeout_seconds"`
}

// DefaultTaaSConfig returns the default TaaS model-service configuration.
func DefaultTaaSConfig() TaaSConfig {
	return TaaSConfig{
		TimeoutSeconds: 120,
	}
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:                                   "0.0.0.0",
			Port:                                   8081,
			GRPCPort:                               8082,
			ReadTimeout:                            30,
			WriteTimeout:                           30,
			AgentAutomationTaskDescriptionMaxRunes: 8192,
		},
		Database: DatabaseConfig{
			Host:            "localhost",
			Port:            6001,
			User:            "root",
			Database:        "moi",
			ConnectTimeout:  10,
			ReadTimeout:     30,
			WriteTimeout:    30,
			MaxOpenConns:    100,
			MaxIdleConns:    10,
			ConnMaxLifetime: 3600,
		},
		Provider: ProviderConfig{
			Type: "local",
			Local: LocalProviderConfig{
				Host: "localhost",
				Port: 6001,
			},
		},
		Storage: StorageConfig{
			Type:   "s3",
			Region: "us-east-1",
			UseSSL: false,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
		},
		Pprof: PprofConfig{
			Enabled: true,
			Addr:    ":6061",
		},
		Mowl: MowlConfig{
			Embedded: true,
			RuntimeLease: MowlRuntimeLeaseConfig{
				TTLSeconds:           30,
				RenewIntervalSeconds: 3,
			},
		},
		Runtime: RuntimeConfig{
			Enabled: false,
			Local: LocalRuntimeConfig{
				Enabled:          false,
				DockerSocket:     "/var/run/docker.sock",
				PythonWorkerCmd:  "python3",
				PythonWorkerArgs: []string{"workers/python-worker/cmd/python_worker.py"},
			},
			Cloud: CloudRuntimeConfig{
				Enabled:                false,
				Namespace:              "mowl-workers",
				GoWorkerTemplate:       "moi-go-worker",
				PythonWorkerTemplate:   "moi-python-worker",
				StartupTimeout:         120,
				KeepFailedPodsDuration: 600,
			},
			GC: GCConfig{
				Enabled:     true,
				Interval:    120,
				GracePeriod: 300,
			},
		},
		AccountDeletion: AccountDeletionConfig{
			IntervalSeconds: 600,
		},
		Parser: ParserConfig{
			MinerU: MinerUParserConfig{
				PDFSplit: MinerUPDFSplitConfig{
					ThresholdPages: 180,
					PagesPerChunk:  50,
				},
			},
		},
		AgentRuntime: AgentRuntimeConfig{
			DevelopmentStubEnabled: false,
			RuntimeGrantTTLSeconds: 600,
			Sandbox: AgentSandboxConfig{
				MaxFileBytes: 512 << 20,
			},
			AstraBinding: AstraBindingConfig{
				TimeoutSeconds: 10,
			},
			AstraRuntime: AstraRuntimeConfig{
				TimeoutSeconds: 180,
			},
		},
		HTTPWorkItem: HTTPWorkItemConfig{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			MaxConnsPerHost:     0,
			IdleConnTimeout:     90,
			ReadTimeout:         30,
			WriteTimeout:        30,
		},
		Explore: ExploreConfig{
			Retriever: ExploreRetrieverConfig{
				MaxRepairAttempts:              2,
				SQLGenerationMaxTokens:         4096,
				SQLRepairMaxTokens:             4096,
				SQLResultsetResolverMaxTokens:  16384,
				RAGTopK:                        5,
				ConcurrencyLimit:               3,
				SubQuestionTimeoutSeconds:      60,
				LookupResolverMaxSourceRows:    2000,
				LookupResolverMaxReturnedCodes: 24,
				LookupIndexEmbeddingBatchSize:  128,
			},
			Context: ExploreContextConfig{
				SchemaCacheTTLSeconds: 300,
			},
			Direct: ExploreDirectConfig{
				MaxContentChars: 32000,
				MaxFileCount:    3,
				MaxTotalBytes:   512000,
			},
			Budget: ExploreBudgetConfig{
				GuaranteedMinTokenRatio: 0.4,
				GuaranteedMinTimeRatio:  0.5,
				TokenEstimationStrategy: "chars_div_4",
			},
			Memory: ExploreMemoryConfig{
				MaxHistoryMessages:  10,
				RecentWindow:        5,
				HistoryTokenBudget:  2000,
				EnableResultContext: true,
				EnableSummary:       true,
			},
			SchemaCacheTTLSeconds:      300,
			SessionTitleTimeoutSeconds: 10,
		},
		TaaS: DefaultTaaSConfig(),
		PersonalAccessToken: PersonalAccessTokenConfig{
			Enabled:        false,
			TimeoutSeconds: 5,
		},
		TenantDBPool: TenantDBPoolConfig{
			ConnectionTimeoutSeconds: 30,
			ReadTimeoutSeconds:       120,
			WriteTimeoutSeconds:      30,
		},
		Upgrade: UpgradeConfig{
			CompatibilityCheckEnabled: true,
			BootstrapEnabled:          true,
			TenantGuardEnabled:        true,
		},
		Garbage: GarbageConfig{
			Enabled: true,
		},
		Reconcile: DefaultReconcileConfig(),
		DataconnSecret: DataconnSecretConfig{
			Enabled: false,
		},
		WebSearch: WebSearchConfig{
			Enabled: true,
			Backend: "auto",
		},
		WebOpen: WebOpenConfig{
			Enabled: true,
			Backend: "mcp",
		},
		Billing: billingconfig.DefaultConfig(),
	}
}
