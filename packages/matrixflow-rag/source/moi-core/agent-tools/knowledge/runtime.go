package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"
)

const runtimeRunStateKey = "matrixflow.agent_tools.knowledge.run_context"

type RuntimeRunState interface {
	GetOrCreate(key string, create func() any) any
}

type RuntimeRequestScope struct {
	WorkspaceID string
	UserID      string
}

type RuntimeManifest struct {
	ID          string
	WorkspaceID string
	Body        map[string]any
}

type RuntimeToolRequest struct {
	WorkspaceID  string
	Manifest     RuntimeManifest
	TurnMetadata map[string]any
	TurnParts    []map[string]any
	RequestScope RuntimeRequestScope
	RunState     RuntimeRunState
}

type RuntimeToolSnapshot struct {
	ID   string
	Kind string
}

type RuntimeToolExecutorFactory func(context.Context, RuntimeToolRequest, RuntimeToolSnapshot) (agentruntimev2.Tool, error)

type RuntimeScopeResolver interface {
	ResolveKnowledgeScope(context.Context, WorkspaceScope) (WorkspaceScope, error)
}

func RuntimeToolExecutors(registry *Registry, resolver RuntimeScopeResolver) map[string]RuntimeToolExecutorFactory {
	if registry == nil {
		return nil
	}
	return map[string]RuntimeToolExecutorFactory{
		ToolNameFindRAGFiles: func(ctx context.Context, req RuntimeToolRequest, _ RuntimeToolSnapshot) (agentruntimev2.Tool, error) {
			scope, err := RuntimeToolScope(ctx, req, resolver)
			if err != nil {
				return nil, err
			}
			return runtimeKnowledgeTool{Tool: NewFindRAGFilesTool(Options{Registry: registry, Scope: scope}), runState: req.RunState}, nil
		},
		ToolNameSearchRAGChunks: func(ctx context.Context, req RuntimeToolRequest, _ RuntimeToolSnapshot) (agentruntimev2.Tool, error) {
			scope, err := RuntimeToolScope(ctx, req, resolver)
			if err != nil {
				return nil, err
			}
			return runtimeKnowledgeTool{Tool: NewSearchRAGChunksTool(Options{Registry: registry, Scope: scope}), runState: req.RunState}, nil
		},
		ToolNameSearchVisualImage: func(ctx context.Context, req RuntimeToolRequest, _ RuntimeToolSnapshot) (agentruntimev2.Tool, error) {
			scope, err := RuntimeToolScope(ctx, req, resolver)
			if err != nil {
				return nil, err
			}
			return runtimeKnowledgeTool{Tool: NewSearchVisualImageTool(Options{
				Registry:     registry,
				Scope:        scope,
				QueryVisuals: QueryVisualsFromMessageParts(req.TurnParts),
			}), runState: req.RunState}, nil
		},
		ToolNameReadParsedMarkdown: func(ctx context.Context, req RuntimeToolRequest, _ RuntimeToolSnapshot) (agentruntimev2.Tool, error) {
			scope, err := RuntimeToolScope(ctx, req, resolver)
			if err != nil {
				return nil, err
			}
			return runtimeKnowledgeTool{Tool: NewReadParsedMarkdownTool(Options{Registry: registry, Scope: scope}), runState: req.RunState}, nil
		},
		ToolNameSearchParsedMarkdown: func(ctx context.Context, req RuntimeToolRequest, _ RuntimeToolSnapshot) (agentruntimev2.Tool, error) {
			scope, err := RuntimeToolScope(ctx, req, resolver)
			if err != nil {
				return nil, err
			}
			return runtimeKnowledgeTool{Tool: NewSearchParsedMarkdownTool(Options{Registry: registry, Scope: scope}), runState: req.RunState}, nil
		},
		ToolNameDescribeSchema: func(ctx context.Context, req RuntimeToolRequest, _ RuntimeToolSnapshot) (agentruntimev2.Tool, error) {
			scope, err := RuntimeToolScope(ctx, req, resolver)
			if err != nil {
				return nil, err
			}
			return runtimeKnowledgeTool{Tool: NewDescribeSchemaTool(Options{Registry: registry, Scope: scope}), runState: req.RunState}, nil
		},
		ToolNameQuerySQL: func(ctx context.Context, req RuntimeToolRequest, _ RuntimeToolSnapshot) (agentruntimev2.Tool, error) {
			scope, err := RuntimeToolScope(ctx, req, resolver)
			if err != nil {
				return nil, err
			}
			return runtimeKnowledgeTool{Tool: NewQuerySQLTool(Options{Registry: registry, Scope: scope}), runState: req.RunState}, nil
		},
		ToolNameUpsertKnowledgeTable: func(ctx context.Context, req RuntimeToolRequest, _ RuntimeToolSnapshot) (agentruntimev2.Tool, error) {
			scope, err := RuntimeToolScope(ctx, req, resolver)
			if err != nil {
				return nil, err
			}
			return runtimeKnowledgeTool{Tool: NewUpsertKnowledgeTableTool(Options{Registry: registry, Scope: scope}), runState: req.RunState}, nil
		},
		ToolNameSelectFinalSources: func(_ context.Context, req RuntimeToolRequest, _ RuntimeToolSnapshot) (agentruntimev2.Tool, error) {
			return runtimeKnowledgeTool{
				Tool:                    NewSelectFinalSourcesTool(),
				runState:                req.RunState,
				selectFinalSourcesTool:  true,
				citableEvidenceRequired: runtimeManifestHasCitableEvidenceTool(req.Manifest),
			}, nil
		},
		ToolNameSubmitFinalAnswer: func(_ context.Context, req RuntimeToolRequest, _ RuntimeToolSnapshot) (agentruntimev2.Tool, error) {
			return runtimeKnowledgeTool{Tool: NewSubmitFinalAnswerTool(), runState: req.RunState}, nil
		},
	}
}

type runtimeKnowledgeTool struct {
	agentruntimev2.Tool
	runState                RuntimeRunState
	selectFinalSourcesTool  bool
	citableEvidenceRequired bool
}

func (t runtimeKnowledgeTool) Execute(ctx context.Context, params json.RawMessage) (*agentruntimev2.ToolResult, error) {
	toolCtx, err := t.context(ctx)
	if err != nil {
		return nil, err
	}
	return t.Tool.Execute(toolCtx, params)
}

func (t runtimeKnowledgeTool) ExecuteInvocation(ctx context.Context, invocation agentruntimev2.ToolInvocation, params json.RawMessage) (*agentruntimev2.ToolResult, error) {
	toolCtx, err := t.context(ctx)
	if err != nil {
		return nil, err
	}
	if invocationTool, ok := t.Tool.(agentruntimev2.InvocationTool); ok {
		return invocationTool.ExecuteInvocation(toolCtx, invocation, params)
	}
	return t.Tool.Execute(toolCtx, params)
}

func (t runtimeKnowledgeTool) context(ctx context.Context) (context.Context, error) {
	toolCtx, err := ContextWithRuntimeRunContext(ctx, t.runState)
	if err != nil {
		return nil, err
	}
	if t.selectFinalSourcesTool {
		RunContextFrom(toolCtx).SetCitableEvidenceRequired(t.citableEvidenceRequired)
	}
	return toolCtx, nil
}

func runtimeManifestHasCitableEvidenceTool(manifest RuntimeManifest) bool {
	for _, tool := range runtimeSliceOfMaps(manifest.Body["tools"]) {
		switch strings.TrimSpace(runtimeStringFromMap(tool, "kind")) {
		case ToolNameSearchRAGChunks, ToolNameSearchVisualImage, ToolNameQuerySQL:
			return true
		}
	}
	return false
}

func ContextWithRuntimeRunContext(ctx context.Context, runState RuntimeRunState) (context.Context, error) {
	if RunContextFrom(ctx) != nil {
		return ctx, nil
	}
	if runState == nil {
		return nil, fmt.Errorf("knowledge runtime run context is not configured")
	}
	value := runState.GetOrCreate(runtimeRunStateKey, func() any {
		return NewRunContext()
	})
	runCtx, ok := value.(*RunContext)
	if !ok || runCtx == nil {
		return nil, fmt.Errorf("knowledge runtime run context has invalid type %T", value)
	}
	return ContextWithRunContext(ctx, runCtx), nil
}

func RuntimeToolScope(ctx context.Context, req RuntimeToolRequest, resolver RuntimeScopeResolver) (WorkspaceScope, error) {
	metadata := req.TurnMetadata
	scope := WorkspaceScope{
		WorkspaceID:             strings.TrimSpace(req.WorkspaceID),
		UserID:                  strings.TrimSpace(runtimeStringFromMap(metadata, "user_id")),
		SessionID:               strings.TrimSpace(runtimeStringFromMap(metadata, "session_id")),
		DBName:                  strings.TrimSpace(firstNonEmpty(runtimeStringFromMap(metadata, "database"), runtimeStringFromMap(metadata, "db_name"))),
		VolumeID:                strings.TrimSpace(runtimeStringFromMap(metadata, "volume_id")),
		VectorTable:             strings.TrimSpace(runtimeStringFromMap(metadata, "vector_table")),
		EmbeddingModel:          strings.TrimSpace(runtimeStringFromMap(metadata, "embedding_model")),
		ImageVectorTable:        strings.TrimSpace(runtimeStringFromMap(metadata, "image_vector_table")),
		ImageEmbeddingModel:     strings.TrimSpace(runtimeStringFromMap(metadata, "image_embedding_model")),
		ImageEmbeddingBackendID: strings.TrimSpace(runtimeStringFromMap(metadata, "image_embedding_backend_id")),
		ImageEmbeddingDimension: runtimeIntFromAny(metadata["image_embedding_dimension"]),
		ImagePreprocessVersion:  strings.TrimSpace(runtimeStringFromMap(metadata, "image_preprocess_version")),
		ImageDistanceMetric:     strings.TrimSpace(runtimeStringFromMap(metadata, "image_distance_metric")),
		Tables:                  runtimeStringsFromAny(metadata["tables"]),
		FileIDs:                 runtimeStringsFromAny(metadata["file_ids"]),
		SemanticModelIDs:        runtimeInt64sFromAny(metadata["semantic_model_ids"]),
	}
	if scope.WorkspaceID == "" {
		scope.WorkspaceID = strings.TrimSpace(req.Manifest.WorkspaceID)
	}
	if scope.WorkspaceID == "" {
		scope.WorkspaceID = strings.TrimSpace(req.RequestScope.WorkspaceID)
	}
	if strings.TrimSpace(req.RequestScope.UserID) != "" {
		scope.UserID = strings.TrimSpace(req.RequestScope.UserID)
	}
	if nested := runtimeMapFromMap(metadata, "scope"); len(nested) > 0 {
		scope = mergeRuntimeScopeFromMap(scope, nested)
	}
	if nested := runtimeMapFromMap(metadata, "scope_metadata"); len(nested) > 0 {
		scope = mergeRuntimeScopeFromMap(scope, nested)
	}
	scope = mergeRuntimeScopeFromManifest(scope, req.Manifest)
	if resolver != nil {
		return resolver.ResolveKnowledgeScope(ctx, scope)
	}
	return CompactScope(scope), nil
}

func mergeRuntimeScopeFromMap(scope WorkspaceScope, values map[string]any) WorkspaceScope {
	if scope.DBName == "" {
		scope.DBName = strings.TrimSpace(firstNonEmpty(runtimeStringFromMap(values, "database"), runtimeStringFromMap(values, "db_name")))
	}
	if scope.VolumeID == "" {
		scope.VolumeID = strings.TrimSpace(runtimeStringFromMap(values, "volume_id"))
	}
	if scope.VectorTable == "" {
		scope.VectorTable = strings.TrimSpace(runtimeStringFromMap(values, "vector_table"))
	}
	if scope.EmbeddingModel == "" {
		scope.EmbeddingModel = strings.TrimSpace(runtimeStringFromMap(values, "embedding_model"))
	}
	if scope.ImageVectorTable == "" {
		scope.ImageVectorTable = strings.TrimSpace(runtimeStringFromMap(values, "image_vector_table"))
	}
	if scope.ImageEmbeddingModel == "" {
		scope.ImageEmbeddingModel = strings.TrimSpace(runtimeStringFromMap(values, "image_embedding_model"))
	}
	if scope.ImageEmbeddingBackendID == "" {
		scope.ImageEmbeddingBackendID = strings.TrimSpace(runtimeStringFromMap(values, "image_embedding_backend_id"))
	}
	if scope.ImageEmbeddingDimension <= 0 {
		scope.ImageEmbeddingDimension = runtimeIntFromAny(values["image_embedding_dimension"])
	}
	if scope.ImagePreprocessVersion == "" {
		scope.ImagePreprocessVersion = strings.TrimSpace(runtimeStringFromMap(values, "image_preprocess_version"))
	}
	if scope.ImageDistanceMetric == "" {
		scope.ImageDistanceMetric = strings.TrimSpace(runtimeStringFromMap(values, "image_distance_metric"))
	}
	if len(scope.FileIDs) == 0 {
		scope.FileIDs = runtimeStringsFromAny(values["file_ids"])
	}
	if len(scope.Tables) == 0 {
		scope.Tables = runtimeStringsFromAny(values["tables"])
	}
	if len(scope.SemanticModelIDs) == 0 {
		scope.SemanticModelIDs = runtimeInt64sFromAny(values["semantic_model_ids"])
	}
	return scope
}

func mergeRuntimeScopeFromManifest(scope WorkspaceScope, manifest RuntimeManifest) WorkspaceScope {
	knowledgeSnapshots := runtimeKnowledgeBasesFromManifest(manifest)
	fileIDs := append([]string(nil), scope.FileIDs...)
	volumeIDs := make([]string, 0)
	modelIDs := append([]int64(nil), scope.SemanticModelIDs...)
	selectedModelIDs := runtimeInt64Set(scope.SemanticModelIDs)
	for _, kb := range knowledgeSnapshots {
		if len(selectedModelIDs) > 0 && !runtimeKnowledgeSnapshotMatchesSelectedModel(kb, selectedModelIDs) {
			continue
		}
		if len(selectedModelIDs) == 0 {
			if id, ok := runtimeSemanticModelIDFromSnapshot(kb); ok {
				modelIDs = append(modelIDs, id)
			}
		}
		for _, ref := range kb.CatalogAssetRefs {
			refType := strings.TrimSpace(runtimeStringFromMap(ref, "type"))
			refID := strings.TrimSpace(runtimeStringFromMap(ref, "id"))
			config := runtimeMapFromMap(ref, "config")
			switch refType {
			case "file":
				fileIDs = append(fileIDs, firstNonEmpty(runtimeStringFromMap(config, "file_id"), refID))
			case "volume":
				volumeIDs = append(volumeIDs, firstNonEmpty(runtimeStringFromMap(config, "volume_id"), refID))
			case "table":
				// Table identity is always database.table. Only explicit
				// config.db_name + config.table_name contribute SQL scope —
				// do not guess from config.name or ref id.
				dbName := strings.TrimSpace(runtimeStringFromMap(config, "db_name"))
				tableName := strings.TrimSpace(runtimeStringFromMap(config, "table_name"))
				if dbName == "" || tableName == "" {
					continue
				}
				if scope.DBName == "" {
					scope.DBName = dbName
				}
				scope.Tables = append(scope.Tables, dbName+"."+tableName)
			}
		}
	}
	scope.SemanticModelIDs = runtimeCompactInt64s(modelIDs)
	scope.Tables = runtimeCompactStrings(scope.Tables)
	scope.FileIDs = runtimeCompactStrings(fileIDs)
	volumeIDs = runtimeCompactStrings(volumeIDs)
	for _, volumeID := range volumeIDs {
		scope.RAGSources = append(scope.RAGSources, RAGSource{
			SemanticModelID: firstRuntimeScopeSemanticModelID(scope.SemanticModelIDs),
			VolumeID:        volumeID,
			VectorTable:     scope.VectorTable,
			EmbeddingModel:  scope.EmbeddingModel,
			Metadata:        map[string]string{"source": "manifest_knowledge"},
		})
	}
	if len(scope.FileIDs) > 0 || scope.VectorTable != "" || scope.EmbeddingModel != "" ||
		scope.ImageVectorTable != "" || scope.ImageEmbeddingModel != "" || scope.ImageEmbeddingBackendID != "" ||
		scope.ImageEmbeddingDimension > 0 || scope.ImagePreprocessVersion != "" || scope.ImageDistanceMetric != "" {
		scope.RAGSources = append(scope.RAGSources, RAGSource{
			SemanticModelID:         firstRuntimeScopeSemanticModelID(scope.SemanticModelIDs),
			FileIDs:                 append([]string(nil), scope.FileIDs...),
			VectorTable:             scope.VectorTable,
			EmbeddingModel:          scope.EmbeddingModel,
			ImageVectorTable:        scope.ImageVectorTable,
			ImageEmbeddingModel:     scope.ImageEmbeddingModel,
			ImageEmbeddingBackendID: scope.ImageEmbeddingBackendID,
			ImageEmbeddingDimension: scope.ImageEmbeddingDimension,
			ImagePreprocessVersion:  scope.ImagePreprocessVersion,
			ImageDistanceMetric:     scope.ImageDistanceMetric,
			Metadata:                map[string]string{"source": "manifest_knowledge"},
		})
	}
	return CompactScope(scope)
}

func CompactScope(scope WorkspaceScope) WorkspaceScope {
	scope.WorkspaceID = strings.TrimSpace(scope.WorkspaceID)
	scope.UserID = strings.TrimSpace(scope.UserID)
	scope.SessionID = strings.TrimSpace(scope.SessionID)
	scope.DBName = strings.TrimSpace(scope.DBName)
	scope.VectorTable = strings.TrimSpace(scope.VectorTable)
	scope.EmbeddingModel = strings.TrimSpace(scope.EmbeddingModel)
	scope.ImageVectorTable = strings.TrimSpace(scope.ImageVectorTable)
	scope.ImageEmbeddingModel = strings.TrimSpace(scope.ImageEmbeddingModel)
	scope.ImageEmbeddingBackendID = strings.TrimSpace(scope.ImageEmbeddingBackendID)
	scope.ImagePreprocessVersion = strings.TrimSpace(scope.ImagePreprocessVersion)
	scope.ImageDistanceMetric = strings.TrimSpace(scope.ImageDistanceMetric)
	scope.VolumeID = strings.TrimSpace(scope.VolumeID)
	scope.SemanticModelIDs = runtimeCompactInt64s(scope.SemanticModelIDs)
	scope.Tables = runtimeCompactStrings(scope.Tables)
	scope.FileIDs = runtimeCompactStrings(scope.FileIDs)
	scope.SourceRowIDByFileID = runtimeCompactStringMap(scope.SourceRowIDByFileID)
	scope.CurrentSegmentVersionByFileID = runtimeCompactStringMap(scope.CurrentSegmentVersionByFileID)
	scope.CurrentIndexVersionByFileID = runtimeCurrentIndexVersionByFileID(scope.CurrentIndexVersionByFileID)
	scope.IndexVersionConstraintByFileID = runtimeIndexVersionConstraintByFileID(scope.IndexVersionConstraintByFileID)
	scope.RAGSources = CompactRAGSources(scope.RAGSources)
	return scope
}

func CompactRAGSources(values []RAGSource) []RAGSource {
	out := make([]RAGSource, 0, len(values))
	seen := map[string]int{}
	for _, source := range values {
		source.DBName = strings.TrimSpace(source.DBName)
		source.VectorTable = strings.TrimSpace(source.VectorTable)
		source.EmbeddingModel = strings.TrimSpace(source.EmbeddingModel)
		source.ImageVectorTable = strings.TrimSpace(source.ImageVectorTable)
		source.ImageEmbeddingModel = strings.TrimSpace(source.ImageEmbeddingModel)
		source.ImageEmbeddingBackendID = strings.TrimSpace(source.ImageEmbeddingBackendID)
		source.ImagePreprocessVersion = strings.TrimSpace(source.ImagePreprocessVersion)
		source.ImageDistanceMetric = strings.TrimSpace(source.ImageDistanceMetric)
		source.VolumeID = strings.TrimSpace(source.VolumeID)
		source.SourceRowIDs = runtimeCompactStrings(source.SourceRowIDs)
		source.FileIDs = runtimeCompactStrings(source.FileIDs)
		source.SourceRowIDByFileID = runtimeCompactStringMap(source.SourceRowIDByFileID)
		source.CurrentSegmentVersionByFileID = runtimeCompactStringMap(source.CurrentSegmentVersionByFileID)
		source.CurrentIndexVersionByFileID = runtimeCurrentIndexVersionByFileID(source.CurrentIndexVersionByFileID)
		source.IndexVersionConstraintByFileID = runtimeIndexVersionConstraintByFileID(source.IndexVersionConstraintByFileID)
		source.SourceTagsByFileID = runtimeSourceTagsByFileID(source.SourceTagsByFileID)
		source.Metadata = runtimeCompactStringMap(source.Metadata)
		if !RAGSourceHasIndex(source) && source.VolumeID == "" && len(source.FileIDs) == 0 {
			continue
		}
		key := strings.Join([]string{
			strconv.FormatInt(source.SemanticModelID, 10),
			source.DBName,
			source.VectorTable,
			source.EmbeddingModel,
			source.ImageVectorTable,
			source.ImageEmbeddingModel,
			source.ImageEmbeddingBackendID,
			strconv.Itoa(source.ImageEmbeddingDimension),
			source.ImagePreprocessVersion,
			source.ImageDistanceMetric,
			source.VolumeID,
		}, "\x00")
		if idx, ok := seen[key]; ok {
			out[idx].FileIDs = runtimeCompactStrings(append(out[idx].FileIDs, source.FileIDs...))
			out[idx].SourceRowIDs = runtimeCompactStrings(append(out[idx].SourceRowIDs, source.SourceRowIDs...))
			out[idx].SourceRowIDByFileID = runtimeMergeStringMap(out[idx].SourceRowIDByFileID, source.SourceRowIDByFileID)
			out[idx].CurrentSegmentVersionByFileID = runtimeMergeStringMap(out[idx].CurrentSegmentVersionByFileID, source.CurrentSegmentVersionByFileID)
			out[idx].CurrentIndexVersionByFileID = runtimeMergeCurrentIndexVersionByFileID(out[idx].CurrentIndexVersionByFileID, source.CurrentIndexVersionByFileID)
			out[idx].IndexVersionConstraintByFileID = runtimeMergeIndexVersionConstraintByFileID(out[idx].IndexVersionConstraintByFileID, source.IndexVersionConstraintByFileID)
			out[idx].SourceTags = append(out[idx].SourceTags, source.SourceTags...)
			out[idx].SourceTagsByFileID = runtimeMergeSourceTagsByFileID(out[idx].SourceTagsByFileID, source.SourceTagsByFileID)
			out[idx].Metadata = runtimeMergeStringMap(out[idx].Metadata, source.Metadata)
			continue
		}
		seen[key] = len(out)
		out = append(out, source)
	}
	return out
}

func runtimeCompactStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func runtimeMergeStringMap(left, right map[string]string) map[string]string {
	if len(left) == 0 {
		return runtimeCompactStringMap(right)
	}
	out := runtimeCompactStringMap(left)
	if out == nil {
		out = map[string]string{}
	}
	for key, value := range runtimeCompactStringMap(right) {
		if _, exists := out[key]; !exists {
			out[key] = value
		}
	}
	return out
}

func runtimeCurrentIndexVersionByFileID(values map[string]int64) map[string]int64 {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]int64, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" || value <= 0 {
			continue
		}
		out[key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func runtimeMergeCurrentIndexVersionByFileID(left, right map[string]int64) map[string]int64 {
	if len(left) == 0 {
		return runtimeCurrentIndexVersionByFileID(right)
	}
	out := runtimeCurrentIndexVersionByFileID(left)
	if out == nil {
		out = map[string]int64{}
	}
	for key, value := range runtimeCurrentIndexVersionByFileID(right) {
		if _, exists := out[key]; !exists {
			out[key] = value
		}
	}
	return out
}

func runtimeIndexVersionConstraintByFileID(values map[string]RAGIndexVersionConstraint) map[string]RAGIndexVersionConstraint {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]RAGIndexVersionConstraint, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		switch value.Kind {
		case RAGIndexVersionConstraintValue:
			out[key] = RAGIndexVersionConstraint{Kind: RAGIndexVersionConstraintValue, Value: value.Value}
		case RAGIndexVersionConstraintNull:
			out[key] = RAGIndexVersionConstraint{Kind: RAGIndexVersionConstraintNull}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func runtimeMergeIndexVersionConstraintByFileID(left, right map[string]RAGIndexVersionConstraint) map[string]RAGIndexVersionConstraint {
	if len(left) == 0 {
		return runtimeIndexVersionConstraintByFileID(right)
	}
	out := runtimeIndexVersionConstraintByFileID(left)
	if out == nil {
		out = map[string]RAGIndexVersionConstraint{}
	}
	for key, value := range runtimeIndexVersionConstraintByFileID(right) {
		if _, exists := out[key]; !exists {
			out[key] = value
		}
	}
	return out
}

func runtimeSourceTagsByFileID(values map[string][]string) map[string][]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string][]string, len(values))
	for key, tags := range values {
		key = strings.TrimSpace(key)
		if key == "" || len(tags) == 0 {
			continue
		}
		out[key] = append([]string(nil), tags...)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func runtimeMergeSourceTagsByFileID(left, right map[string][]string) map[string][]string {
	if len(left) == 0 {
		return runtimeSourceTagsByFileID(right)
	}
	out := runtimeSourceTagsByFileID(left)
	if out == nil {
		out = map[string][]string{}
	}
	for key, values := range runtimeSourceTagsByFileID(right) {
		out[key] = append(out[key], values...)
	}
	return out
}

func RAGSourceHasIndex(source RAGSource) bool {
	return source.VectorTable != "" || source.EmbeddingModel != "" ||
		source.ImageVectorTable != "" || source.ImageEmbeddingModel != "" || source.ImageEmbeddingBackendID != "" ||
		source.ImageEmbeddingDimension > 0 || source.ImagePreprocessVersion != "" || source.ImageDistanceMetric != ""
}

func UniqueRAGSourceValue(sources []RAGSource, value func(RAGSource) string) string {
	out := ""
	for _, source := range sources {
		current := strings.TrimSpace(value(source))
		if current == "" {
			continue
		}
		if out != "" && out != current {
			return ""
		}
		out = current
	}
	return out
}

func UniqueRAGSourceIntValue(sources []RAGSource, value func(RAGSource) int) int {
	out := 0
	for _, source := range sources {
		current := value(source)
		if current <= 0 {
			continue
		}
		if out > 0 && out != current {
			return 0
		}
		out = current
	}
	return out
}

type runtimeKnowledgeManifestSnapshot struct {
	ID               string
	CatalogAssetRefs []map[string]any
	Metadata         map[string]any
}

func runtimeKnowledgeBasesFromManifest(manifest RuntimeManifest) []runtimeKnowledgeManifestSnapshot {
	items := runtimeSliceOfMaps(manifest.Body["knowledge_bases"])
	out := make([]runtimeKnowledgeManifestSnapshot, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(runtimeStringFromMap(item, "id"))
		if id == "" {
			continue
		}
		out = append(out, runtimeKnowledgeManifestSnapshot{
			ID:               id,
			CatalogAssetRefs: runtimeSliceOfMaps(item["catalog_asset_refs"]),
			Metadata:         runtimeMapFromMap(item, "metadata"),
		})
	}
	return out
}

func runtimeSemanticModelIDFromSnapshot(kb runtimeKnowledgeManifestSnapshot) (int64, bool) {
	if id, err := strconv.ParseInt(strings.TrimSpace(kb.ID), 10, 64); err == nil && id > 0 {
		return id, true
	}
	return runtimeInt64FromAny(kb.Metadata["semantic_model_id"])
}

func runtimeKnowledgeSnapshotMatchesSelectedModel(kb runtimeKnowledgeManifestSnapshot, selected map[int64]struct{}) bool {
	if len(selected) == 0 {
		return true
	}
	for _, id := range runtimeSemanticModelIDsFromSnapshot(kb) {
		if _, ok := selected[id]; ok {
			return true
		}
	}
	return false
}

func runtimeSemanticModelIDsFromSnapshot(kb runtimeKnowledgeManifestSnapshot) []int64 {
	out := make([]int64, 0, 2)
	if id, err := strconv.ParseInt(strings.TrimSpace(kb.ID), 10, 64); err == nil && id > 0 {
		out = append(out, id)
	}
	if id, ok := runtimeInt64FromAny(kb.Metadata["semantic_model_id"]); ok {
		out = append(out, id)
	}
	return runtimeCompactInt64s(out)
}

func firstRuntimeScopeSemanticModelID(ids []int64) int64 {
	if len(ids) == 0 {
		return 0
	}
	return ids[0]
}

func runtimeStringFromMap(values map[string]any, key string) string {
	if len(values) == 0 {
		return ""
	}
	return runtimeValueString(values[key])
}

func runtimeMapFromMap(values map[string]any, key string) map[string]any {
	if len(values) == 0 {
		return nil
	}
	return runtimeMapFromAny(values[key])
}

func runtimeMapFromAny(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return runtimeCloneMap(typed)
	case map[string]string:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = item
		}
		return out
	default:
		return nil
	}
}

func runtimeSliceOfMaps(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, runtimeCloneMap(item))
		}
		return out
	case []any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if mapped := runtimeMapFromAny(item); len(mapped) > 0 {
				out = append(out, mapped)
			}
		}
		return out
	default:
		return nil
	}
}

func runtimeStringsFromAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return runtimeCompactStrings(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text := runtimeValueString(item)
			if text != "" {
				out = append(out, text)
			}
		}
		return runtimeCompactStrings(out)
	case string:
		return runtimeCompactStrings([]string{typed})
	default:
		return nil
	}
}

func runtimeIntFromAny(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int8:
		return int(typed)
	case int16:
		return int(typed)
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case uint:
		return int(typed)
	case uint8:
		return int(typed)
	case uint16:
		return int(typed)
	case uint32:
		return int(typed)
	case uint64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		n, _ := strconv.Atoi(typed.String())
		return n
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(typed))
		return n
	default:
		return 0
	}
}

func runtimeInt64sFromAny(value any) []int64 {
	switch typed := value.(type) {
	case []int64:
		return runtimeCompactInt64s(typed)
	case []int:
		out := make([]int64, 0, len(typed))
		for _, item := range typed {
			out = append(out, int64(item))
		}
		return runtimeCompactInt64s(out)
	case []string:
		out := make([]int64, 0, len(typed))
		for _, item := range typed {
			if id, ok := runtimeInt64FromAny(item); ok {
				out = append(out, id)
			}
		}
		return runtimeCompactInt64s(out)
	case []any:
		out := make([]int64, 0, len(typed))
		for _, item := range typed {
			if id, ok := runtimeInt64FromAny(item); ok {
				out = append(out, id)
			}
		}
		return runtimeCompactInt64s(out)
	default:
		if id, ok := runtimeInt64FromAny(value); ok {
			return []int64{id}
		}
		return nil
	}
}

func runtimeInt64FromAny(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, typed > 0
	case int:
		return int64(typed), typed > 0
	case int32:
		return int64(typed), typed > 0
	case float64:
		id := int64(typed)
		return id, typed == float64(id) && id > 0
	case json.Number:
		id, err := typed.Int64()
		return id, err == nil && id > 0
	case string:
		id, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return id, err == nil && id > 0
	default:
		return 0, false
	}
}

func runtimeCompactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func runtimeCompactInt64s(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func runtimeInt64Set(values []int64) map[int64]struct{} {
	values = runtimeCompactInt64s(values)
	if len(values) == 0 {
		return nil
	}
	out := make(map[int64]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}

func runtimeValueString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return ""
	}
}

func runtimeCloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = runtimeCloneAny(value)
	}
	return out
}

func runtimeCloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return runtimeCloneMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, runtimeCloneAny(item))
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, runtimeCloneMap(item))
		}
		return out
	default:
		return typed
	}
}
