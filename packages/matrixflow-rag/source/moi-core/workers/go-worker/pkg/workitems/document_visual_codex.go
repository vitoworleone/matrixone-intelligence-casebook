package workitems

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/data"
	corelog "github.com/matrixflow/moi-core/model/logging"
	"github.com/matrixflow/moi-core/model/mowl"
	"github.com/matrixflow/moi-core/workers/go-worker/pkg/runtime"
	"go.uber.org/zap"
)

const (
	documentVisualCodexParseMethod      = "document_visual_codex"
	documentVisualCodexExtractionSource = "codex_document_visual"
	defaultDocumentVisualCodexTimeout   = 30 * time.Minute
)

type DocumentVisualCodexConfig struct {
	Command         []string
	Timeout         time.Duration
	Model           string
	ReasoningEffort string
}

type documentVisualCodexOverrides struct {
	APIKey          string
	BaseURL         string
	Model           string
	ReasoningEffort string
}

type documentVisualCodexCommandResult struct {
	Stdout string
	Stderr string
}

type documentVisualCodexOutputDiagnostics struct {
	RootEntries        []string
	OutputEntries      []string
	MarkdownCandidates []string
	JSONCandidates     []string
	Errors             []string
}

type DocumentVisualParseCodex struct {
	Factory *runtime.ClientFactory
	Config  DocumentVisualCodexConfig
}

type documentVisualCodexSidecar struct {
	Pages    []documentVisualCodexPage   `json:"pages"`
	Objects  []documentVisualCodexObject `json:"objects"`
	Entities []documentVisualCodexEntity `json:"entities"`
}

type documentVisualCodexPage struct {
	PageNumber        int       `json:"page_number"`
	FullPageImagePath string    `json:"full_page_image_path"`
	Width             float64   `json:"width"`
	Height            float64   `json:"height"`
	BBox              []float64 `json:"bbox"`
	Summary           string    `json:"summary,omitempty"`
	Text              string    `json:"text,omitempty"`
}

type documentVisualCodexObject struct {
	ObjectID   string    `json:"object_id"`
	ObjectKind string    `json:"object_kind"`
	PageNumber int       `json:"page_number"`
	BBox       []float64 `json:"bbox"`
	ImagePath  string    `json:"image_path"`
	Caption    string    `json:"caption"`
	Text       string    `json:"text"`
	OCR        string    `json:"ocr,omitempty"`
	Context    string    `json:"context"`
}

type documentVisualCodexEntity struct {
	EntityID   string `json:"entity_id"`
	Type       string `json:"type"`
	Value      string `json:"value"`
	PageNumber int    `json:"page_number"`
	ObjectID   string `json:"object_id"`
	Evidence   string `json:"evidence"`
}

type documentVisualCodexUploadedArtifact struct {
	FileID      string
	ContentType string
	Width       int
	Height      int
	LocalPath   string
}

func (p *DocumentVisualParseCodex) Handle(ctx context.Context, wctx moi.WorkItemContext, msg *mowl.MowlMessage) (result *mowl.MowlMessage, retErr error) {
	execCtx := wctx.ExecutionContext()
	trace := newDocumentVisualCodexTrace(ctx, execCtx.WorkspaceId, msg)
	totalStart := time.Now()
	source := documentVisualSource{}
	profile := ""
	manifestFileID := ""
	markdownFileID := ""
	pageCount := 0
	objectCount := 0
	entityCount := 0
	documentCount := 0
	defer func() {
		fields := []zap.Field{
			zap.Int("page_count", pageCount),
			zap.Int("object_count", objectCount),
			zap.Int("entity_count", entityCount),
			zap.Int("document_count", documentCount),
			zap.String("manifest_file_id", manifestFileID),
			zap.String("markdown_file_id", markdownFileID),
		}
		if retErr != nil {
			trace.ErrorDuration("document visual codex parse failed", "document_visual_codex.parse", totalStart, retErr, fields...)
			return
		}
		trace.InfoDuration("document visual codex parse completed", "document_visual_codex.parse", totalStart, fields...)
	}()

	var in documentVisualParseInput
	if err := json.Unmarshal([]byte(msg.Data), &in); err != nil {
		return nil, fmt.Errorf("document_visual.parse.codex: parse input: %w", err)
	}
	if len(p.Config.Command) == 0 {
		return nil, fmt.Errorf("document_visual.parse.codex: document_visual_codex.command is required")
	}
	profile = strings.TrimSpace(in.Profile)
	if profile == "" {
		profile = documentVisualDefaultProfile
	}
	source = inferDocumentVisualSource(in, nil)
	if source.FileID == "" {
		return nil, fmt.Errorf("document_visual.parse.codex: sources/file_id/file_ids is required")
	}
	if source.FileName == "" {
		source.FileName = source.FileID
	}
	codexModel, codexBackendID, err := documentVisualCodexModelSelectionFromOptions(in.Options)
	if err != nil {
		return nil, err
	}
	codexReasoningEffort := documentVisualCodexReasoningEffortFromOptions(in.Options)
	trace.SetSourceProfile(source, profile)
	trace.Info("document visual codex parse started", "document_visual_codex.parse",
		zap.Duration("timeout", documentVisualCodexEffectiveTimeout(p.Config)),
		zap.String("codex_model", codexModel),
		zap.String("codex_reasoning_effort", documentVisualCodexEffectiveReasoningEffort(p.Config, codexReasoningEffort)),
		zap.Int64("codex_backend_id", codexBackendID),
	)

	client, err := p.Factory.Get(execCtx)
	if err != nil {
		return nil, fmt.Errorf("document_visual.parse.codex: create SDK client: %w", err)
	}
	if err := validateDocumentVisualCodexModelAccess(ctx, client, execCtx.WorkspaceId, codexModel, codexBackendID); err != nil {
		return nil, err
	}
	overrides, err := p.resolveDocumentVisualCodexOverrides(ctx, execCtx.WorkspaceId, codexModel, codexBackendID)
	if err != nil {
		return nil, err
	}
	overrides.ReasoningEffort = codexReasoningEffort
	tmpDir, err := os.MkdirTemp("", "moi-document-visual-codex-*")
	if err != nil {
		return nil, fmt.Errorf("document_visual.parse.codex: create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputDir := filepath.Join(tmpDir, "input")
	if err := os.MkdirAll(inputDir, 0700); err != nil {
		return nil, fmt.Errorf("document_visual.parse.codex: create input dir: %w", err)
	}
	sourcePath := filepath.Join(inputDir, filepath.Base(source.FileName))
	downloadStart := time.Now()
	if err := client.Files().DownloadToFile(ctx, execCtx.WorkspaceId, source.FileID, sourcePath); err != nil {
		return nil, fmt.Errorf("document_visual.parse.codex: download source file %s: %w", source.FileID, err)
	}
	trace.InfoDuration("document visual codex source downloaded", "document_visual_codex.download_source", downloadStart,
		zap.String("source_path", filepath.ToSlash(sourcePath)),
	)

	stem := strings.TrimSuffix(filepath.Base(source.FileName), filepath.Ext(source.FileName))
	if stem == "" {
		stem = source.FileID
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "outputs"), 0700); err != nil {
		return nil, fmt.Errorf("document_visual.parse.codex: create outputs dir: %w", err)
	}
	prompt := documentVisualCodexPrompt(filepath.ToSlash(filepath.Join("input", filepath.Base(source.FileName))), stem)
	commandStart := time.Now()
	runResult, err := runDocumentVisualCodexCommand(ctx, tmpDir, p.Config, overrides, prompt, trace)
	if err != nil {
		return nil, err
	}
	trace.InfoDuration("document visual codex command completed", "document_visual_codex.command", commandStart,
		zap.Duration("timeout", documentVisualCodexEffectiveTimeout(p.Config)),
		zap.Int("stdout_bytes", len(runResult.Stdout)),
		zap.Int("stderr_bytes", len(runResult.Stderr)),
	)

	mdPath := filepath.Join(tmpDir, "outputs", stem+"_extracted.md")
	sidecarPath := filepath.Join(tmpDir, "outputs", stem+"_manifest.json")
	loadOutputStart := time.Now()
	mdBytes, err := os.ReadFile(mdPath)
	if err != nil {
		return nil, documentVisualCodexMissingOutputError(trace, tmpDir, filepath.ToSlash(filepath.Join("outputs", stem+"_extracted.md")), err, runResult)
	}
	if _, err := os.Stat(sidecarPath); err != nil {
		return nil, documentVisualCodexMissingOutputError(trace, tmpDir, filepath.ToSlash(filepath.Join("outputs", stem+"_manifest.json")), err, runResult)
	}
	sidecar, err := loadDocumentVisualCodexSidecar(sidecarPath, tmpDir)
	if err != nil {
		return nil, err
	}
	pageCount = len(sidecar.Pages)
	objectCount = len(sidecar.Objects)
	entityCount = len(sidecar.Entities)
	trace.InfoDuration("document visual codex outputs loaded", "document_visual_codex.load_outputs", loadOutputStart,
		zap.Int("markdown_bytes", len(mdBytes)),
		zap.Int("page_count", pageCount),
		zap.Int("object_count", objectCount),
		zap.Int("entity_count", entityCount),
	)

	artifactUploadStart := time.Now()
	pageUploads, objectUploads, refFileIDs, err := p.uploadCodexArtifacts(ctx, client, execCtx.WorkspaceId, tmpDir, source, sidecar)
	if err != nil {
		return nil, err
	}
	trace.InfoDuration("document visual codex artifacts uploaded", "document_visual_codex.upload_artifacts", artifactUploadStart,
		zap.Int("page_upload_count", len(pageUploads)),
		zap.Int("object_upload_count", len(objectUploads)),
		zap.Int("markdown_ref_count", len(refFileIDs)),
	)
	mdBytes = rewriteDocumentVisualCodexMarkdownReferences(mdBytes, refFileIDs)
	mdName := stem + "_extracted.md"
	markdownUploadStart := time.Now()
	mdResp, err := client.Files().UploadBytes(ctx, execCtx.WorkspaceId, mdName, mdBytes)
	if err != nil {
		return nil, fmt.Errorf("document_visual.parse.codex: upload markdown %s: %w", mdName, err)
	}
	markdownFileID = mdResp.FileID
	trace.InfoDuration("document visual codex markdown uploaded", "document_visual_codex.upload_markdown", markdownUploadStart,
		zap.String("markdown_file_id", markdownFileID),
		zap.String("markdown_file_name", mdName),
		zap.Int("markdown_bytes", len(mdBytes)),
	)

	manifestBuildStart := time.Now()
	manifest, err := buildDocumentVisualCodexManifest(source, profile, sidecar, pageUploads, objectUploads)
	if err != nil {
		return nil, err
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("document_visual.parse.codex: marshal manifest: %w", err)
	}
	trace.InfoDuration("document visual codex manifest built", "document_visual_codex.build_manifest", manifestBuildStart,
		zap.Int("manifest_bytes", len(manifestBytes)),
		zap.Int("page_count", pageCount),
		zap.Int("object_count", objectCount),
		zap.Int("entity_count", entityCount),
	)
	if boolPtrDefault(in.WriteManifestFile, true) {
		fileName := strings.TrimSpace(in.ManifestFileName)
		if fileName == "" {
			fileName = documentVisualManifestFileName(source.FileName, source.FileID)
		}
		manifestUploadStart := time.Now()
		resp, err := client.Files().UploadBytes(ctx, execCtx.WorkspaceId, fileName, manifestBytes)
		if err != nil {
			return nil, fmt.Errorf("document_visual.parse.codex: upload manifest: %w", err)
		}
		manifestFileID = resp.FileID
		trace.InfoDuration("document visual codex manifest uploaded", "document_visual_codex.upload_manifest", manifestUploadStart,
			zap.String("manifest_file_id", manifestFileID),
			zap.String("manifest_file_name", fileName),
			zap.Int("manifest_bytes", len(manifestBytes)),
		)
	}
	documentBuildStart := time.Now()
	docs, err := buildDocumentVisualCodexDocuments(manifest, sidecar, mdResp.FileID, manifestFileID, string(mdBytes))
	if err != nil {
		return nil, err
	}
	documentCount = len(docs)
	trace.InfoDuration("document visual codex documents built", "document_visual_codex.build_documents", documentBuildStart,
		zap.Int("document_count", documentCount),
	)
	out := documentVisualParseOutput{
		SchemaVersion:          documentVisualSchemaVersion,
		Profile:                profile,
		Manifest:               manifest,
		ManifestFileID:         manifestFileID,
		DerivedFileIDsBySource: documentVisualDerivedFileIDsBySource([]documentVisualManifest{manifest}),
		Documents:              docs,
		Validation:             manifest.Validation,
	}
	payload, _ := json.Marshal(out)
	msg.Data = string(payload)
	return nil, nil
}

type documentVisualCodexTrace struct {
	ctx         context.Context
	logger      *zap.Logger
	workspaceID string
	msg         *mowl.MowlMessage
	source      documentVisualSource
	profile     string
}

func newDocumentVisualCodexTrace(ctx context.Context, workspaceID string, msg *mowl.MowlMessage) *documentVisualCodexTrace {
	return &documentVisualCodexTrace{
		ctx:         ctx,
		logger:      corelog.WithContext(ctx, nil),
		workspaceID: workspaceID,
		msg:         msg,
	}
}

func (t *documentVisualCodexTrace) SetSourceProfile(source documentVisualSource, profile string) {
	t.source = source
	t.profile = profile
}

func (t *documentVisualCodexTrace) Info(message, operation string, fields ...zap.Field) {
	t.logger.Info(message, t.fields(operation, fields...)...)
}

func (t *documentVisualCodexTrace) InfoDuration(message, operation string, start time.Time, fields ...zap.Field) {
	t.Info(message, operation, append(durationFields(time.Since(start)), fields...)...)
}

func (t *documentVisualCodexTrace) Warn(message, operation string, fields ...zap.Field) {
	t.logger.Warn(message, t.fields(operation, fields...)...)
}

func (t *documentVisualCodexTrace) ErrorDuration(message, operation string, start time.Time, err error, fields ...zap.Field) {
	fields = append(durationFields(time.Since(start)), fields...)
	fields = append(fields, zap.Error(err))
	t.logger.Error(message, t.fields(operation, fields...)...)
}

func (t *documentVisualCodexTrace) fields(operation string, extra ...zap.Field) []zap.Field {
	fields := []zap.Field{
		zap.String("component", "go-worker.document_visual_codex"),
		zap.String("operation", operation),
		zap.String("workspace_id", t.workspaceID),
		zap.String("case_id", t.msg.GetCaseId()),
		zap.String("workitem_id", t.msg.GetWorkItemId()),
		zap.String("source_file_id", t.source.FileID),
		zap.String("source_file_name", t.source.FileName),
		zap.String("profile", t.profile),
	}
	return append(fields, extra...)
}

func durationFields(duration time.Duration) []zap.Field {
	return []zap.Field{
		zap.Duration("duration", duration),
		zap.Int64("duration_ms", duration.Milliseconds()),
	}
}

func documentVisualCodexModelSelectionFromOptions(options map[string]interface{}) (string, int64, error) {
	model := toString(options["model"])
	backendID := toInt64(options["backend_id"], 0)
	if model == "" {
		return "", 0, fmt.Errorf("document_visual.parse.codex: options.model is required")
	}
	if backendID == 0 {
		return "", 0, fmt.Errorf("document_visual.parse.codex: options.backend_id is required")
	}
	return model, backendID, nil
}

func documentVisualCodexReasoningEffortFromOptions(options map[string]interface{}) string {
	return toString(options["model_reasoning_effort"])
}

func documentVisualCodexEffectiveReasoningEffort(cfg DocumentVisualCodexConfig, optionValue string) string {
	if optionValue != "" {
		return optionValue
	}
	return strings.TrimSpace(cfg.ReasoningEffort)
}

func validateDocumentVisualCodexModelAccess(ctx context.Context, client *moi.Client, workspaceID, model string, backendID int64) error {
	if client == nil {
		return fmt.Errorf("document_visual.parse.codex: SDK client is required for model access validation")
	}
	resp, err := client.LLM(workspaceID).ListModels(ctx, moi.WithWorkspaceModelType("chat"))
	if err != nil {
		return fmt.Errorf("document_visual.parse.codex: validate LLM model access for backend_id %d: %w", backendID, err)
	}
	if resp == nil {
		return fmt.Errorf("document_visual.parse.codex: validate LLM model access for backend_id %d returned empty response", backendID)
	}
	for _, item := range resp.Models {
		if item.Model == model && item.BackendID == backendID {
			return nil
		}
	}
	return fmt.Errorf("document_visual.parse.codex: selected LLM model %q with backend_id %d is not available to the execution user", model, backendID)
}

func (p *DocumentVisualParseCodex) resolveDocumentVisualCodexOverrides(ctx context.Context, workspaceID, model string, backendID int64) (documentVisualCodexOverrides, error) {
	client, err := p.Factory.GetSystem()
	if err != nil {
		return documentVisualCodexOverrides{}, fmt.Errorf("document_visual.parse.codex: create system SDK client for LLM route: %w", err)
	}
	route, err := client.LLM(workspaceID).ResolveRoute(ctx, model, backendID)
	if err != nil {
		return documentVisualCodexOverrides{}, fmt.Errorf("document_visual.parse.codex: resolve LLM route for backend_id %d: %w", backendID, err)
	}
	if route == nil {
		return documentVisualCodexOverrides{}, fmt.Errorf("document_visual.parse.codex: resolve LLM route for backend_id %d returned empty response", backendID)
	}
	if route.GetBackendId() != backendID {
		return documentVisualCodexOverrides{}, fmt.Errorf("document_visual.parse.codex: resolved backend_id %d does not match requested backend_id %d", route.GetBackendId(), backendID)
	}
	if route.GetEndpointAddress() == "" {
		return documentVisualCodexOverrides{}, fmt.Errorf("document_visual.parse.codex: resolved LLM endpoint for backend_id %d has empty address", backendID)
	}
	if route.GetApiKeyEncrypted() == "" {
		return documentVisualCodexOverrides{}, fmt.Errorf("document_visual.parse.codex: resolved LLM backend_id %d has empty API key", backendID)
	}
	resolvedModel := route.GetModel()
	if resolvedModel == "" {
		resolvedModel = model
	}
	return documentVisualCodexOverrides{
		APIKey:  route.GetApiKeyEncrypted(),
		BaseURL: route.GetEndpointAddress(),
		Model:   resolvedModel,
	}, nil
}

func (o documentVisualCodexOverrides) Enabled() bool {
	return o.APIKey != "" || o.BaseURL != "" || o.Model != "" || o.ReasoningEffort != ""
}

func documentVisualCodexPrompt(sourcePath, stem string) string {
	return fmt.Sprintf(`对目录下的 %s 进行内容识别：
1、基于零件的每一个加工视图做截取，并对每个视图中信息（尺寸、弧度）进行提取和对含义进行说明和解读，要求把裁切的图片也嵌入到 Markdown 文件中，并放在对应的位置，即裁切图片要和其被提取的内容在同一个文档区域；
2、请先将 PDF 渲染为整页 PNG，再按每个加工视图、剖面视图、局部视图、表格、NOTES、标题栏分别裁切；
3、裁切后必须检查每张图片是否包含完整尺寸线和文字；如果有边缘文字被裁掉，必须重裁；如果带入了其他视图中的片段，要严格区分归属，只能提取当前视图中的内容到对应文档区域；
4、表格、文本都要精准还原和提取；
5、需要严格还原原图的结构，不做主观整理；
6、每个视图、剖面视图、局部视图的尺寸都必须逐项提取，包括括号内参考尺寸、带星号关键尺寸、直径符号、数量前缀、角度、弧度、弧向尺寸、总高度、基准字母、T 编号和公差；括号内参考尺寸不能因为是参考尺寸而忽略，必须说明其参考含义和所在视图；
7、如果同一个尺寸靠近视图边界，必须根据完整尺寸线、箭头、标注文字和裁切区域判断归属；不能把剖面视图或局部视图的尺寸归到主视图，也不能把其他视图片段混入当前视图；
8、Markdown 中每个裁切对象必须包含：裁切图片、原图位置/视图名称、提取表格（提取项、原图数值或文本、含义说明）、边界归属说明；文末必须有裁切检查记录，逐张说明是否完整包含尺寸线和文字；
9、object_kind 为 table 的裁切对象必须在 Markdown 中用 Markdown 表格还原原表行列和表头；例如修订履历表要输出 REV、DESCRIPTION、LOC、BY、DATE、ECN NO 等列和每一行修订记录。不能只用一段自然语言描述代替原表格；自然语言只能作为表格后的含义说明；
10、sidecar manifest 的 objects.text、objects.context、entities 也必须写入上述尺寸、参考尺寸、表格字段和归属说明，不能只写在 Markdown；
11、最终必须生成 outputs/%s_extracted.md、outputs/%s_manifest.json 和 outputs/%s_assets/images/*.png。

outputs/%s_manifest.json 必须是机器可读 JSON，结构如下：
{
  "pages": [{"page_number": 1, "full_page_image_path": "outputs/%s_assets/images/page_1.png", "width": 0, "height": 0, "bbox": [0, 0, 0, 0]}],
  "objects": [{"object_id": "object_1", "object_kind": "view|section|detail|table|notes|title_block", "page_number": 1, "bbox": [0, 0, 0, 0], "image_path": "outputs/%s_assets/images/object_1.png", "caption": "", "text": "", "context": ""}],
  "entities": [{"entity_id": "entity_1", "type": "", "value": "", "page_number": 1, "object_id": "object_1", "evidence": ""}]
}

sidecar 中的图片路径必须是相对当前目录的路径，不要写绝对路径。bbox 必须是 4 个数字。业务结果只写入上述文件。`, sourcePath, stem, stem, stem, stem, stem, stem)
}

func runDocumentVisualCodexCommand(ctx context.Context, workDir string, cfg DocumentVisualCodexConfig, overrides documentVisualCodexOverrides, prompt string, trace *documentVisualCodexTrace) (documentVisualCodexCommandResult, error) {
	timeout := documentVisualCodexEffectiveTimeout(cfg)
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := documentVisualCodexCommandArgs(cfg, overrides)
	cmd := exec.CommandContext(runCtx, args[0], args[1:]...)
	cmd.Dir = workDir
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr bytes.Buffer
	cmd.Env = append(os.Environ(), "CODEX_LAUNCHER_QUIET=1")
	if overrides.Enabled() {
		codexHome, err := prepareDocumentVisualCodexOverrideHome(workDir, cmd.Env, overrides)
		if err != nil {
			return documentVisualCodexCommandResult{}, err
		}
		defer os.RemoveAll(codexHome)
		cmd.Env = append(cmd.Env, "CODEX_HOME="+codexHome)
	}
	logDocumentVisualCodexCommandConfig(trace, cmd.Env, args, workDir, prompt)
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return documentVisualCodexCommandResult{}, fmt.Errorf("document_visual.parse.codex: create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return documentVisualCodexCommandResult{}, fmt.Errorf("document_visual.parse.codex: create stderr pipe: %w", err)
	}
	start := time.Now()
	if err := cmd.Start(); err != nil {
		return documentVisualCodexCommandResult{}, fmt.Errorf("document_visual.parse.codex: start command: %w", err)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		captureDocumentVisualCodexStdout(trace, stdoutPipe, &stdout, start)
	}()
	go func() {
		defer wg.Done()
		captureDocumentVisualCodexStream(stderrPipe, &stderr)
	}()
	err = cmd.Wait()
	wg.Wait()
	result := documentVisualCodexCommandResult{
		Stdout: redactDocumentVisualCodexOutput(stdout.String(), overrides),
		Stderr: redactDocumentVisualCodexOutput(stderr.String(), overrides),
	}
	if err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return result, fmt.Errorf("document_visual.parse.codex: command timed out after %s", timeout)
		}
		return result, fmt.Errorf("document_visual.parse.codex: command failed: %w; stderr=%s; stdout=%s", err, strings.TrimSpace(result.Stderr), strings.TrimSpace(result.Stdout))
	}
	return result, nil
}

func prepareDocumentVisualCodexOverrideHome(workDir string, env []string, overrides documentVisualCodexOverrides) (string, error) {
	sourceHome := documentVisualCodexHome(env)
	targetHome, err := os.MkdirTemp("", "moi-document-visual-codex-home-*")
	if err != nil {
		return "", fmt.Errorf("document_visual.parse.codex: create temporary CODEX_HOME: %w", err)
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(targetHome)
		}
	}()

	sourceConfig := filepath.Join(sourceHome, "config.toml")
	targetConfig := filepath.Join(targetHome, "config.toml")
	if err := copyDocumentVisualCodexFile(sourceConfig, targetConfig, 0600); err != nil {
		return "", fmt.Errorf("document_visual.parse.codex: copy CODEX config.toml: %w", err)
	}
	if overrides.BaseURL != "" || overrides.Model != "" || overrides.ReasoningEffort != "" {
		raw, err := os.ReadFile(targetConfig)
		if err != nil {
			return "", fmt.Errorf("document_visual.parse.codex: read temporary CODEX config.toml: %w", err)
		}
		updated := string(raw)
		if overrides.BaseURL != "" {
			updated, err = updateDocumentVisualCodexConfigString(updated, "base_url", overrides.BaseURL, "[model_providers.OpenAI]")
			if err != nil {
				return "", err
			}
		}
		if overrides.Model != "" {
			updated, err = updateDocumentVisualCodexConfigString(updated, "model", overrides.Model, "")
			if err != nil {
				return "", err
			}
		}
		if overrides.ReasoningEffort != "" {
			updated, err = updateDocumentVisualCodexConfigString(updated, "model_reasoning_effort", overrides.ReasoningEffort, "")
			if err != nil {
				return "", err
			}
		}
		if err := os.WriteFile(targetConfig, []byte(updated), 0600); err != nil {
			return "", fmt.Errorf("document_visual.parse.codex: write temporary CODEX config.toml: %w", err)
		}
	}

	sourceAuth := filepath.Join(sourceHome, "auth.json")
	targetAuth := filepath.Join(targetHome, "auth.json")
	if err := copyDocumentVisualCodexFile(sourceAuth, targetAuth, 0600); err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("document_visual.parse.codex: copy CODEX auth.json: %w", err)
		}
	}
	if overrides.APIKey != "" {
		if err := writeDocumentVisualCodexAuthAPIKey(targetAuth, overrides.APIKey); err != nil {
			return "", err
		}
	}
	if rel, err := filepath.Rel(workDir, targetHome); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && rel != ".." {
		return "", fmt.Errorf("document_visual.parse.codex: temporary CODEX_HOME must not be inside workDir")
	}
	success = true
	return targetHome, nil
}

func documentVisualCodexHome(env []string) string {
	if codecHome := documentVisualCodexEnvValue(env, "CODEX_HOME"); codecHome != "" {
		return codecHome
	}
	if codecHome := os.Getenv("CODEX_HOME"); codecHome != "" {
		return codecHome
	}
	return filepath.Join(os.Getenv("HOME"), ".codex")
}

func copyDocumentVisualCodexFile(source, target string, perm fs.FileMode) error {
	raw, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(target, raw, perm)
}

func writeDocumentVisualCodexAuthAPIKey(authPath, apiKey string) error {
	raw, err := os.ReadFile(authPath)
	if err != nil {
		if os.IsNotExist(err) {
			raw, err := json.MarshalIndent(map[string]string{"OPENAI_API_KEY": apiKey}, "", "  ")
			if err != nil {
				return fmt.Errorf("document_visual.parse.codex: marshal temporary CODEX auth.json: %w", err)
			}
			raw = append(raw, '\n')
			if err := os.WriteFile(authPath, raw, 0600); err != nil {
				return fmt.Errorf("document_visual.parse.codex: write temporary CODEX auth.json: %w", err)
			}
			return nil
		}
		return fmt.Errorf("document_visual.parse.codex: read temporary CODEX auth.json: %w", err)
	}
	updated, err := updateDocumentVisualCodexAuthAPIKey(string(raw), apiKey)
	if err != nil {
		return err
	}
	if err := os.WriteFile(authPath, []byte(updated), 0600); err != nil {
		return fmt.Errorf("document_visual.parse.codex: write temporary CODEX auth.json: %w", err)
	}
	return nil
}

func updateDocumentVisualCodexAuthAPIKey(auth, apiKey string) (string, error) {
	payload := map[string]interface{}{}
	if err := json.Unmarshal([]byte(auth), &payload); err != nil {
		return "", fmt.Errorf("document_visual.parse.codex: parse temporary CODEX auth.json: %w", err)
	}
	payload["OPENAI_API_KEY"] = apiKey
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("document_visual.parse.codex: marshal temporary CODEX auth.json: %w", err)
	}
	return string(append(raw, '\n')), nil
}

func updateDocumentVisualCodexConfigString(config, key, value, section string) (string, error) {
	quoted, err := documentVisualCodexTOMLString(value)
	if err != nil {
		return "", err
	}
	lines := strings.SplitAfter(config, "\n")
	inTargetSection := section == ""
	foundTargetSection := section == ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			inTargetSection = trimmed == section
			if inTargetSection {
				foundTargetSection = true
			}
			continue
		}
		if !inTargetSection {
			continue
		}
		lineKey, ok := documentVisualCodexTOMLKey(trimmed)
		if !ok || lineKey != key {
			continue
		}
		lines[i] = documentVisualCodexReplaceTOMLStringValue(line, key, quoted)
		return strings.Join(lines, ""), nil
	}
	if !foundTargetSection {
		return "", fmt.Errorf("document_visual.parse.codex: CODEX config.toml missing %s for %s override", section, key)
	}
	if section == "" {
		return "", fmt.Errorf("document_visual.parse.codex: CODEX config.toml missing %s for override", key)
	}
	return "", fmt.Errorf("document_visual.parse.codex: CODEX config.toml missing %s in %s for override", key, section)
}

func documentVisualCodexTOMLKey(line string) (string, bool) {
	if line == "" || strings.HasPrefix(line, "#") {
		return "", false
	}
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", false
	}
	return strings.TrimSpace(parts[0]), true
}

func documentVisualCodexReplaceTOMLStringValue(line, key, quotedValue string) string {
	newline := ""
	body := line
	if strings.HasSuffix(body, "\n") {
		body = strings.TrimSuffix(body, "\n")
		newline = "\n"
	}
	if strings.HasSuffix(body, "\r") {
		body = strings.TrimSuffix(body, "\r")
		newline = "\r" + newline
	}
	indent := body[:len(body)-len(strings.TrimLeft(body, " \t"))]
	return indent + key + " = " + quotedValue + newline
}

func documentVisualCodexTOMLString(value string) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("document_visual.parse.codex: encode base_url for temporary CODEX config.toml: %w", err)
	}
	return string(raw), nil
}

func redactDocumentVisualCodexOutput(value string, overrides documentVisualCodexOverrides) string {
	if overrides.APIKey != "" {
		value = strings.ReplaceAll(value, overrides.APIKey, "[REDACTED]")
	}
	if overrides.BaseURL != "" {
		value = strings.ReplaceAll(value, overrides.BaseURL, "[REDACTED_BASE_URL]")
	}
	return value
}

func logDocumentVisualCodexCommandConfig(trace *documentVisualCodexTrace, env, args []string, workDir, prompt string) {
	codecHome := documentVisualCodexHome(env)
	configPath := filepath.Join(codecHome, "config.toml")
	authPath := filepath.Join(codecHome, "auth.json")
	configSize, configHash, configErr := documentVisualCodexFileDigest(configPath)
	authSize, authErr := documentVisualCodexFileSize(authPath)
	version, versionErr := documentVisualCodexCommandVersion(trace.ctx, env, args)

	fields := []zap.Field{
		zap.Strings("command_args", args),
		zap.String("work_dir", filepath.ToSlash(workDir)),
		zap.Int("prompt_bytes", len(prompt)),
		zap.String("codex_home", filepath.ToSlash(codecHome)),
		zap.String("codex_config_path", filepath.ToSlash(configPath)),
		zap.Int64("codex_config_bytes", configSize),
		zap.String("codex_config_sha256", configHash),
		zap.String("codex_auth_path", filepath.ToSlash(authPath)),
		zap.Int64("codex_auth_bytes", authSize),
		zap.String("codex_version", version),
	}
	if configErr != nil {
		fields = append(fields, zap.String("codex_config_error", configErr.Error()))
	}
	if authErr != nil {
		fields = append(fields, zap.String("codex_auth_error", authErr.Error()))
	}
	if versionErr != nil {
		fields = append(fields, zap.String("codex_version_error", versionErr.Error()))
	}
	trace.Info("document visual codex command config", "document_visual_codex.command_config", fields...)
}

func documentVisualCodexEnvValue(env []string, key string) string {
	prefix := key + "="
	for i := len(env) - 1; i >= 0; i-- {
		if strings.HasPrefix(env[i], prefix) {
			return strings.TrimPrefix(env[i], prefix)
		}
	}
	return ""
}

func documentVisualCodexFileDigest(filePath string) (int64, string, error) {
	raw, err := os.ReadFile(filePath)
	if err != nil {
		return 0, "", err
	}
	sum := sha256.Sum256(raw)
	return int64(len(raw)), fmt.Sprintf("%x", sum), nil
}

func documentVisualCodexFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func documentVisualCodexCommandVersion(ctx context.Context, env, args []string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("codex command is empty")
	}
	versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(versionCtx, args[0], "--version")
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if versionCtx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("codex version timed out")
	}
	if err != nil {
		return strings.TrimSpace(string(out)), err
	}
	return strings.TrimSpace(string(out)), nil
}

func captureDocumentVisualCodexStdout(trace *documentVisualCodexTrace, reader io.Reader, dst *bytes.Buffer, start time.Time) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	eventIndex := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		dst.Write(line)
		dst.WriteByte('\n')
		eventIndex++
		eventType := documentVisualCodexJSONEventType(line)
		if eventType == "" {
			eventType = "unknown"
		}
		trace.Info("document visual codex event", "document_visual_codex.command_event",
			zap.Int("event_index", eventIndex),
			zap.String("event_type", eventType),
			zap.Int64("elapsed_ms", time.Since(start).Milliseconds()),
			zap.Int("line_bytes", len(line)),
		)
	}
	if err := scanner.Err(); err != nil {
		trace.Warn("document visual codex stdout scan failed", "document_visual_codex.command_event",
			zap.Error(err),
		)
	}
}

func captureDocumentVisualCodexStream(reader io.Reader, dst *bytes.Buffer) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		dst.Write(scanner.Bytes())
		dst.WriteByte('\n')
	}
}

func documentVisualCodexJSONEventType(line []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(line, &payload); err != nil {
		return ""
	}
	if eventType, ok := payload["type"].(string); ok {
		return eventType
	}
	msg, ok := payload["msg"].(map[string]any)
	if !ok {
		return ""
	}
	if eventType, ok := msg["type"].(string); ok {
		return eventType
	}
	return ""
}

func documentVisualCodexEffectiveTimeout(cfg DocumentVisualCodexConfig) time.Duration {
	if cfg.Timeout <= 0 {
		return defaultDocumentVisualCodexTimeout
	}
	return cfg.Timeout
}

func documentVisualCodexMissingOutputError(trace *documentVisualCodexTrace, workDir, expectedPath string, readErr error, result documentVisualCodexCommandResult) error {
	diag := collectDocumentVisualCodexOutputDiagnostics(workDir)
	trace.Warn("document visual codex output missing", "document_visual_codex.read_output",
		zap.String("work_dir", workDir),
		zap.String("expected_path", expectedPath),
		zap.Strings("root_entries", diag.RootEntries),
		zap.Strings("output_entries", diag.OutputEntries),
		zap.Strings("markdown_candidates", diag.MarkdownCandidates),
		zap.Strings("json_candidates", diag.JSONCandidates),
		zap.Strings("diagnostic_errors", diag.Errors),
		zap.String("codex_stdout", strings.TrimSpace(result.Stdout)),
		zap.String("codex_stderr", strings.TrimSpace(result.Stderr)),
		zap.Error(readErr),
	)
	return fmt.Errorf("document_visual.parse.codex: read output %s: %w; root_entries=%v; output_entries=%v; markdown_candidates=%v; json_candidates=%v; diagnostic_errors=%v",
		expectedPath, readErr, diag.RootEntries, diag.OutputEntries, diag.MarkdownCandidates, diag.JSONCandidates, diag.Errors)
}

func collectDocumentVisualCodexOutputDiagnostics(workDir string) documentVisualCodexOutputDiagnostics {
	rootEntries, rootErr := documentVisualCodexDirEntries(workDir)
	outputEntries, outputErr := documentVisualCodexDirEntries(filepath.Join(workDir, "outputs"))
	diag := documentVisualCodexOutputDiagnostics{
		RootEntries:   rootEntries,
		OutputEntries: outputEntries,
	}
	if rootErr != "" {
		diag.Errors = append(diag.Errors, "read root: "+rootErr)
	}
	if outputErr != "" {
		diag.Errors = append(diag.Errors, "read outputs: "+outputErr)
	}
	if err := filepath.WalkDir(workDir, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			diag.Errors = append(diag.Errors, "walk "+filepath.ToSlash(filePath)+": "+err.Error())
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(workDir, filePath)
		if err != nil {
			diag.Errors = append(diag.Errors, "rel "+filepath.ToSlash(filePath)+": "+err.Error())
			return nil
		}
		rel = filepath.ToSlash(rel)
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".md":
			diag.MarkdownCandidates = append(diag.MarkdownCandidates, rel)
		case ".json":
			diag.JSONCandidates = append(diag.JSONCandidates, rel)
		}
		return nil
	}); err != nil {
		diag.Errors = append(diag.Errors, "walk root: "+err.Error())
	}
	return diag
}

func documentVisualCodexDirEntries(dir string) ([]string, string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err.Error()
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		out = append(out, name)
	}
	return out, ""
}

func documentVisualCodexCommandArgs(cfg DocumentVisualCodexConfig, overrides documentVisualCodexOverrides) []string {
	if overrides.Model != "" {
		cfg.Model = overrides.Model
	}
	if overrides.ReasoningEffort != "" {
		cfg.ReasoningEffort = overrides.ReasoningEffort
	}
	args := append([]string(nil), cfg.Command...)
	if strings.TrimSpace(cfg.Model) != "" && !stringSliceHas(args, "--model") && !stringSliceHas(args, "-m") {
		args = insertCodexArgBeforeStdin(args, "--model", strings.TrimSpace(cfg.Model))
	}
	reasoningEffort := strings.TrimSpace(cfg.ReasoningEffort)
	if reasoningEffort != "" {
		if overrides.ReasoningEffort != "" {
			var replaced bool
			args, replaced = replaceCodexConfigArg(args, "model_reasoning_effort", reasoningEffort)
			if !replaced {
				args = insertCodexArgBeforeStdin(args, "-c", fmt.Sprintf("model_reasoning_effort=%q", reasoningEffort))
			}
		} else if !codexConfigContains(args, "model_reasoning_effort") {
			args = insertCodexArgBeforeStdin(args, "-c", fmt.Sprintf("model_reasoning_effort=%q", reasoningEffort))
		}
	}
	return args
}

func insertCodexArgBeforeStdin(args []string, values ...string) []string {
	if len(args) == 0 {
		return args
	}
	pos := len(args)
	if args[len(args)-1] == "-" {
		pos = len(args) - 1
	}
	out := make([]string, 0, len(args)+len(values))
	out = append(out, args[:pos]...)
	out = append(out, values...)
	out = append(out, args[pos:]...)
	return out
}

func stringSliceHas(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func codexConfigContains(args []string, key string) bool {
	for i, value := range args {
		if value == "-c" && i+1 < len(args) && codexConfigArgHasKey(args[i+1], key) {
			return true
		}
		if strings.HasPrefix(value, "-c") && codexConfigArgHasKey(strings.TrimPrefix(value, "-c"), key) {
			return true
		}
		if codexConfigArgHasKey(value, key) {
			return true
		}
	}
	return false
}

func replaceCodexConfigArg(args []string, key, value string) ([]string, bool) {
	assignment := fmt.Sprintf("%s=%q", key, value)
	out := append([]string(nil), args...)
	for i, current := range out {
		if current == "-c" && i+1 < len(out) {
			if codexConfigArgHasKey(out[i+1], key) {
				out[i+1] = assignment
				return out, true
			}
			continue
		}
		if strings.HasPrefix(current, "-c") && codexConfigArgHasKey(strings.TrimPrefix(current, "-c"), key) {
			out[i] = "-c" + assignment
			return out, true
		}
		if codexConfigArgHasKey(current, key) {
			out[i] = assignment
			return out, true
		}
	}
	return args, false
}

func codexConfigArgHasKey(arg, key string) bool {
	argKey, ok := codexConfigArgKey(arg)
	return ok && argKey == key
}

func codexConfigArgKey(arg string) (string, bool) {
	trimmed := strings.TrimSpace(arg)
	if trimmed == "" {
		return "", false
	}
	index := strings.Index(trimmed, "=")
	if index <= 0 {
		return "", false
	}
	return strings.TrimSpace(trimmed[:index]), true
}

func loadDocumentVisualCodexSidecar(sidecarPath, root string) (documentVisualCodexSidecar, error) {
	raw, err := os.ReadFile(sidecarPath)
	if err != nil {
		return documentVisualCodexSidecar{}, fmt.Errorf("document_visual.parse.codex: read sidecar manifest: %w", err)
	}
	var sidecar documentVisualCodexSidecar
	if err := json.Unmarshal(raw, &sidecar); err != nil {
		return documentVisualCodexSidecar{}, fmt.Errorf("document_visual.parse.codex: parse sidecar manifest: %w", err)
	}
	if err := validateDocumentVisualCodexSidecar(sidecar, root); err != nil {
		return documentVisualCodexSidecar{}, err
	}
	return sidecar, nil
}

func validateDocumentVisualCodexSidecar(sidecar documentVisualCodexSidecar, root string) error {
	if len(sidecar.Pages) == 0 {
		return fmt.Errorf("document_visual.parse.codex: sidecar pages is required")
	}
	pageNumbers := map[int]struct{}{}
	for i, page := range sidecar.Pages {
		if page.PageNumber <= 0 {
			return fmt.Errorf("document_visual.parse.codex: pages[%d].page_number must be > 0", i)
		}
		if page.Width <= 0 || page.Height <= 0 {
			return fmt.Errorf("document_visual.parse.codex: pages[%d].width/height must be > 0", i)
		}
		if err := validateDocumentVisualCodexBBox(page.BBox, fmt.Sprintf("pages[%d].bbox", i)); err != nil {
			return err
		}
		if _, err := resolveDocumentVisualCodexArtifactPath(root, page.FullPageImagePath); err != nil {
			return fmt.Errorf("document_visual.parse.codex: pages[%d].full_page_image_path: %w", i, err)
		}
		pageNumbers[page.PageNumber] = struct{}{}
	}
	if len(sidecar.Objects) == 0 {
		return fmt.Errorf("document_visual.parse.codex: sidecar objects is required")
	}
	objectIDs := map[string]struct{}{}
	for i, obj := range sidecar.Objects {
		if strings.TrimSpace(obj.ObjectID) == "" {
			return fmt.Errorf("document_visual.parse.codex: objects[%d].object_id is required", i)
		}
		if strings.TrimSpace(obj.ObjectKind) == "" {
			return fmt.Errorf("document_visual.parse.codex: objects[%d].object_kind is required", i)
		}
		if obj.PageNumber <= 0 {
			return fmt.Errorf("document_visual.parse.codex: objects[%d].page_number must be > 0", i)
		}
		if _, ok := pageNumbers[obj.PageNumber]; !ok {
			return fmt.Errorf("document_visual.parse.codex: objects[%d].page_number %d not found in pages", i, obj.PageNumber)
		}
		if err := validateDocumentVisualCodexBBox(obj.BBox, fmt.Sprintf("objects[%d].bbox", i)); err != nil {
			return err
		}
		if strings.TrimSpace(obj.Context) == "" {
			return fmt.Errorf("document_visual.parse.codex: objects[%d].context is required", i)
		}
		if _, err := resolveDocumentVisualCodexArtifactPath(root, obj.ImagePath); err != nil {
			return fmt.Errorf("document_visual.parse.codex: objects[%d].image_path: %w", i, err)
		}
		if _, exists := objectIDs[obj.ObjectID]; exists {
			return fmt.Errorf("document_visual.parse.codex: duplicate object_id %q", obj.ObjectID)
		}
		objectIDs[obj.ObjectID] = struct{}{}
	}
	for i, entity := range sidecar.Entities {
		if strings.TrimSpace(entity.EntityID) == "" {
			return fmt.Errorf("document_visual.parse.codex: entities[%d].entity_id is required", i)
		}
		if strings.TrimSpace(entity.Type) == "" {
			return fmt.Errorf("document_visual.parse.codex: entities[%d].type is required", i)
		}
		if strings.TrimSpace(entity.Value) == "" {
			return fmt.Errorf("document_visual.parse.codex: entities[%d].value is required", i)
		}
		if entity.PageNumber <= 0 {
			return fmt.Errorf("document_visual.parse.codex: entities[%d].page_number must be > 0", i)
		}
		if strings.TrimSpace(entity.ObjectID) != "" {
			if _, ok := objectIDs[entity.ObjectID]; !ok {
				return fmt.Errorf("document_visual.parse.codex: entities[%d].object_id %q not found in objects", i, entity.ObjectID)
			}
		}
	}
	return nil
}

func validateDocumentVisualCodexBBox(bbox []float64, field string) error {
	if len(bbox) != 4 {
		return fmt.Errorf("document_visual.parse.codex: %s must contain 4 numbers", field)
	}
	return nil
}

func resolveDocumentVisualCodexArtifactPath(root, rawPath string) (string, error) {
	if strings.TrimSpace(rawPath) == "" {
		return "", fmt.Errorf("path is required")
	}
	if filepath.IsAbs(rawPath) {
		return "", fmt.Errorf("absolute path %q is not allowed", rawPath)
	}
	clean := filepath.Clean(filepath.FromSlash(rawPath))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes output directory", rawPath)
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	full, err := filepath.Abs(filepath.Join(rootAbs, clean))
	if err != nil {
		return "", err
	}
	if full != rootAbs && !strings.HasPrefix(full, rootAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes output directory", rawPath)
	}
	if _, err := os.Stat(full); err != nil {
		return "", fmt.Errorf("path %q not found: %w", rawPath, err)
	}
	return full, nil
}

func (p *DocumentVisualParseCodex) uploadCodexArtifacts(ctx context.Context, client *moi.Client, workspaceID, root string, source documentVisualSource, sidecar documentVisualCodexSidecar) (map[int]documentVisualCodexUploadedArtifact, map[string]documentVisualCodexUploadedArtifact, map[string]string, error) {
	pageUploads := map[int]documentVisualCodexUploadedArtifact{}
	objectUploads := map[string]documentVisualCodexUploadedArtifact{}
	refFileIDs := map[string]string{}
	for _, page := range sidecar.Pages {
		localPath, err := resolveDocumentVisualCodexArtifactPath(root, page.FullPageImagePath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("document_visual.parse.codex: resolve page image %d: %w", page.PageNumber, err)
		}
		artifact, err := uploadDocumentVisualCodexImage(ctx, client, workspaceID, localPath, documentVisualPageImageNameWithExt(source.FileName, page.PageNumber, documentVisualCodexArtifactExt(page.FullPageImagePath)))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("document_visual.parse.codex: upload page image %d: %w", page.PageNumber, err)
		}
		pageUploads[page.PageNumber] = artifact
		addDocumentVisualCodexImageAliases(refFileIDs, root, page.FullPageImagePath, localPath, artifact.FileID)
	}
	for _, obj := range sidecar.Objects {
		localPath, err := resolveDocumentVisualCodexArtifactPath(root, obj.ImagePath)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("document_visual.parse.codex: resolve object image %s: %w", obj.ObjectID, err)
		}
		artifact, err := uploadDocumentVisualCodexImage(ctx, client, workspaceID, localPath, documentVisualObjectImageNameWithExt(source.FileName, obj.ObjectID, documentVisualCodexArtifactExt(obj.ImagePath)))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("document_visual.parse.codex: upload object image %s: %w", obj.ObjectID, err)
		}
		objectUploads[obj.ObjectID] = artifact
		addDocumentVisualCodexImageAliases(refFileIDs, root, obj.ImagePath, localPath, artifact.FileID)
	}
	return pageUploads, objectUploads, refFileIDs, nil
}

func uploadDocumentVisualCodexImage(ctx context.Context, client *moi.Client, workspaceID, localPath, uploadName string) (documentVisualCodexUploadedArtifact, error) {
	imgBytes, err := os.ReadFile(localPath)
	if err != nil {
		return documentVisualCodexUploadedArtifact{}, err
	}
	resp, err := client.Files().UploadBytes(ctx, workspaceID, uploadName, imgBytes)
	if err != nil {
		return documentVisualCodexUploadedArtifact{}, err
	}
	width, height := imageSize(imgBytes)
	return documentVisualCodexUploadedArtifact{
		FileID:      resp.FileID,
		ContentType: http.DetectContentType(imgBytes),
		Width:       width,
		Height:      height,
		LocalPath:   localPath,
	}, nil
}

func documentVisualCodexArtifactExt(rawPath string) string {
	ext := strings.TrimPrefix(path.Ext(rawPath), ".")
	if ext == "" {
		return "png"
	}
	return ext
}

func documentVisualObjectImageNameWithExt(sourceFileName, objectID, ext string) string {
	if strings.TrimSpace(ext) == "" {
		ext = "png"
	}
	stem := strings.TrimSuffix(path.Base(sourceFileName), path.Ext(sourceFileName))
	if stem == "" {
		stem = "drawing"
	}
	return fmt.Sprintf("%s_object_%s_%s.%s", stem, objectID, hashShort(sourceFileName, objectID), ext)
}

func addDocumentVisualCodexImageAliases(refFileIDs map[string]string, root, rawPath, localPath, fileID string) {
	for _, alias := range []string{
		rawPath,
		filepath.ToSlash(filepath.Clean(filepath.FromSlash(rawPath))),
		path.Base(rawPath),
		filepath.Base(localPath),
	} {
		if strings.TrimSpace(alias) != "" {
			refFileIDs[markdownImageAliasKey(alias)] = fileID
		}
	}
	if rel, err := filepath.Rel(root, localPath); err == nil {
		refFileIDs[markdownImageAliasKey(filepath.ToSlash(rel))] = fileID
	}
}

func rewriteDocumentVisualCodexMarkdownReferences(mdBytes []byte, refFileIDs map[string]string) []byte {
	if len(refFileIDs) == 0 {
		return mdBytes
	}
	md := string(mdBytes)
	md = markdownImageRefPattern.ReplaceAllStringFunc(md, func(match string) string {
		parts := markdownImageRefPattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		if fileID := refFileIDs[markdownImageAliasKey(parts[2])]; fileID != "" {
			return fmt.Sprintf("![%s](%s%s)", parts[1], fileID, parts[3])
		}
		return match
	})
	md = htmlImageSrcPattern.ReplaceAllStringFunc(md, func(match string) string {
		parts := htmlImageSrcPattern.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		if fileID := refFileIDs[markdownImageAliasKey(parts[2])]; fileID != "" {
			return parts[1] + fileID + parts[3]
		}
		return match
	})
	return []byte(md)
}

func buildDocumentVisualCodexManifest(source documentVisualSource, profile string, sidecar documentVisualCodexSidecar, pageUploads map[int]documentVisualCodexUploadedArtifact, objectUploads map[string]documentVisualCodexUploadedArtifact) (documentVisualManifest, error) {
	pages := make([]documentVisualPage, 0, len(sidecar.Pages))
	pageIndex := map[int]int{}
	for _, page := range sidecar.Pages {
		upload := pageUploads[page.PageNumber]
		pages = append(pages, documentVisualPage{
			PageNumber:      page.PageNumber,
			PageImageFileID: upload.FileID,
			Width:           page.Width,
			Height:          page.Height,
			Summary:         page.Summary,
			Text:            page.Text,
			BBox:            append([]float64(nil), page.BBox...),
		})
		pageIndex[page.PageNumber] = len(pages) - 1
	}
	objects := make([]documentVisualObject, 0, len(sidecar.Objects))
	for _, obj := range sidecar.Objects {
		upload := objectUploads[obj.ObjectID]
		pageUpload := pageUploads[obj.PageNumber]
		visualObject := documentVisualObject{
			ObjectID:         obj.ObjectID,
			ObjectKind:       obj.ObjectKind,
			PageNumber:       obj.PageNumber,
			BBox:             append([]float64(nil), obj.BBox...),
			OCRBBox:          append([]float64(nil), obj.BBox...),
			ImageFileID:      upload.FileID,
			PageImageFileID:  pageUpload.FileID,
			Text:             obj.Text,
			OCR:              obj.OCR,
			Caption:          obj.Caption,
			Context:          obj.Context,
			ExtractionSource: documentVisualCodexExtractionSource,
		}
		objects = append(objects, visualObject)
		if idx, ok := pageIndex[obj.PageNumber]; ok {
			pages[idx].ObjectIDs = append(pages[idx].ObjectIDs, obj.ObjectID)
		}
	}
	entities := make([]documentVisualEntity, 0, len(sidecar.Entities))
	for _, entity := range sidecar.Entities {
		entities = append(entities, documentVisualEntity{
			EntityID:   entity.EntityID,
			Type:       entity.Type,
			Value:      entity.Value,
			PageNumber: entity.PageNumber,
			ObjectID:   entity.ObjectID,
			Evidence:   entity.Evidence,
			Metadata: map[string]interface{}{
				"extraction_source": documentVisualCodexExtractionSource,
			},
		})
	}
	source.PageCount = len(pages)
	manifest := documentVisualManifest{
		SchemaVersion: documentVisualSchemaVersion,
		Profile:       profile,
		Source:        source,
		Pages:         pages,
		Objects:       objects,
		Entities:      entities,
	}
	manifest.Validation = validateDocumentVisualManifest(manifest, documentVisualBuildOptions{
		RequirePageImages:    true,
		RequireObjectImages:  true,
		RequireVisualContext: true,
	})
	if !manifest.Validation.Valid {
		return manifest, fmt.Errorf("document_visual.parse.codex: manifest validation failed: %s", strings.Join(manifest.Validation.Errors, "; "))
	}
	return manifest, nil
}

func buildDocumentVisualCodexDocuments(manifest documentVisualManifest, sidecar documentVisualCodexSidecar, mdFileID, manifestFileID, markdown string) ([]*data.Document, error) {
	imageArtifacts := documentVisualCodexMarkdownImageArtifacts(manifest, sidecar)
	imageArtifactsJSON, err := json.Marshal(imageArtifacts)
	if err != nil {
		return nil, fmt.Errorf("document_visual.parse.codex: marshal image artifacts: %w", err)
	}
	mdMeta := map[string]interface{}{
		"file_id":            manifest.Source.FileID,
		"raw_file_id":        manifest.Source.FileID,
		"source_file_id":     manifest.Source.FileID,
		"file_name":          manifest.Source.FileName,
		"source_file_name":   manifest.Source.FileName,
		"source_type":        "pdf",
		"block_type":         "codex_markdown",
		"parse_method":       documentVisualCodexParseMethod,
		"profile":            manifest.Profile,
		"extraction_source":  documentVisualCodexExtractionSource,
		"md_file_id":         mdFileID,
		"md_file_url":        mdFileID,
		"manifest_file_id":   manifestFileID,
		"schema_version":     documentVisualSchemaVersion,
		"document_visual_v1": true,
		"image_artifacts":    string(imageArtifactsJSON),
	}
	mdDoc, err := documentToProto(Document{
		ID:       documentVisualIndexID(manifest.Source.FileID, "codex_markdown", mdFileID),
		Type:     "text",
		Content:  markdown,
		Metadata: mdMeta,
	})
	if err != nil {
		return nil, fmt.Errorf("document_visual.parse.codex: build markdown document: %w", err)
	}
	return []*data.Document{mdDoc}, nil
}

func documentVisualCodexMarkdownImageArtifacts(manifest documentVisualManifest, sidecar documentVisualCodexSidecar) []map[string]interface{} {
	artifacts := make([]map[string]interface{}, 0, len(manifest.Pages)+len(manifest.Objects))
	add := func(fileID, localPath string) {
		fileID = strings.TrimSpace(fileID)
		localPath = strings.TrimSpace(localPath)
		if fileID == "" || localPath == "" {
			return
		}
		artifacts = append(artifacts, map[string]interface{}{
			"file_id":    fileID,
			"local_path": localPath,
		})
	}
	pagePathByNumber := make(map[int]string, len(sidecar.Pages))
	for _, page := range sidecar.Pages {
		pagePathByNumber[page.PageNumber] = page.FullPageImagePath
	}
	objectPathByID := make(map[string]string, len(sidecar.Objects))
	for _, obj := range sidecar.Objects {
		objectPathByID[obj.ObjectID] = obj.ImagePath
	}
	for _, page := range manifest.Pages {
		add(page.PageImageFileID, pagePathByNumber[page.PageNumber])
	}
	for _, obj := range manifest.Objects {
		add(obj.ImageFileID, objectPathByID[obj.ObjectID])
	}
	return artifacts
}
