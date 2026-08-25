package workitems

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	_ "image/png"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/data"
	"github.com/matrixflow/moi-core/model/mowl"
	"github.com/matrixflow/moi-core/workers/go-worker/pkg/runtime"
	"github.com/matrixflow/moi-core/workers/go-worker/pkg/workitems/parser"
	"go.uber.org/zap"
)

const (
	documentVisualSchemaVersion      = "document_visual_parse_v1"
	documentVisualDefaultProfile     = "industrial_drawing_v1"
	documentVisualStandardRAGProfile = "standard_rag_v1"
)

type DocumentVisualParse struct {
	Factory       *runtime.ClientFactory
	VersionRouter *parser.VersionRouter
	ParserQueues  *ParserAPIQueues
}

type DocumentVisualIndexText struct {
	Factory *runtime.ClientFactory
	Logger  *zap.Logger
}

type DocumentVisualIndexImage struct {
	Factory *runtime.ClientFactory
	Logger  *zap.Logger
}

func (p *DocumentVisualParse) Handle(ctx context.Context, wctx moi.WorkItemContext, msg *mowl.MowlMessage) (*mowl.MowlMessage, error) {
	execCtx := wctx.ExecutionContext()
	var in documentVisualParseInput
	if err := json.Unmarshal([]byte(msg.Data), &in); err != nil {
		return nil, fmt.Errorf("document_visual.parse: parse input: %w", err)
	}
	profile := strings.TrimSpace(in.Profile)
	if profile == "" {
		profile = documentVisualDefaultProfile
	}
	if in.Enabled != nil && !*in.Enabled {
		out := documentVisualParseOutput{
			SchemaVersion: documentVisualSchemaVersion,
			Profile:       profile,
			Manifest:      documentVisualManifest{},
			Manifests:     nil,
			Documents:     nil,
			Validation: documentVisualValidation{
				Valid: true,
			},
			Status: "disabled",
		}
		payload, _ := json.Marshal(out)
		msg.Data = string(payload)
		return nil, nil
	}
	engineeringOpts, err := documentVisualEngineeringOptionsFromInput(profile, in.Options)
	if err != nil {
		return nil, err
	}
	docs := in.Documents
	if len(docs) == 0 {
		payload, err := documentVisualParsePayload(in)
		if err != nil {
			return nil, err
		}
		parseMsg := &mowl.MowlMessage{Data: payload}
		if _, err := (&DocumentParse{Factory: p.Factory, VersionRouter: p.VersionRouter, ParserQueues: p.ParserQueues}).Handle(ctx, wctx, parseMsg); err != nil {
			return nil, fmt.Errorf("document_visual.parse: parse source documents: %w", err)
		}
		var parsed struct {
			Documents []*data.Document `json:"documents"`
		}
		if err := json.Unmarshal([]byte(parseMsg.Data), &parsed); err != nil {
			return nil, fmt.Errorf("document_visual.parse: decode parser output: %w", err)
		}
		docs = parsed.Documents
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("document_visual.parse: parsed documents is empty")
	}
	groups := documentVisualParseDocumentGroups(in, docs)
	for _, group := range groups {
		if _, err := documentVisualFigureCaptionRelationsFromProto(group.Documents); err != nil {
			return nil, fmt.Errorf("document_visual.parse: validate figure-caption relations for source %q: %w", group.FileID, err)
		}
	}
	if engineeringOpts.Enabled() && len(groups) != 1 {
		return nil, fmt.Errorf("document_visual.parse: engineering drawing enhancement supports one source per run, got %d", len(groups))
	}

	client, err := p.Factory.Get(execCtx)
	if err != nil {
		return nil, fmt.Errorf("document_visual.parse: create SDK client: %w", err)
	}
	layout, err := loadDocumentVisualLayout(ctx, client, execCtx.WorkspaceId, in.Layout, docs)
	if err != nil {
		return nil, err
	}
	opts := documentVisualBuildOptions{
		RequirePageImages:    boolPtrDefault(in.RequirePageImages, true),
		RequireObjectImages:  boolPtrDefault(in.RequireObjectImages, true),
		RequireVisualContext: boolPtrDefault(in.RequireVisualContext, true),
	}
	manifests := make([]documentVisualManifest, 0, len(groups))
	manifestFileIDs := make([]string, 0, len(groups))
	outDocs := append([]*data.Document(nil), docs...)
	for _, group := range groups {
		groupInput := in
		groupInput.Sources = group.Sources
		groupInput.FileID = group.FileID
		groupInput.FileIDs = nil
		pageAssets, source, err := p.preparePageAssets(ctx, client, execCtx.WorkspaceId, groupInput, group.Documents, layout)
		if err != nil {
			return nil, err
		}
		manifest, err := buildDocumentVisualManifest(ctx, client, execCtx.WorkspaceId, group.Documents, layout, pageAssets, source, profile, opts)
		if err != nil {
			return nil, err
		}
		if engineeringOpts.Enabled() {
			if strings.Contains(strings.ToLower(source.MimeType), "pdf") || strings.ToLower(path.Ext(source.FileName)) == ".pdf" {
				pageAssets, err = p.prepareEngineeringDrawingPageAssets(ctx, client, execCtx.WorkspaceId, source)
				if err != nil {
					return nil, err
				}
			}
			manifest, err = p.enhanceIndustrialDrawingManifest(ctx, *execCtx, client, execCtx.WorkspaceId, manifest, pageAssets, source, engineeringOpts, opts)
			if err != nil {
				return nil, err
			}
			outDocs, err = p.appendEngineeringDrawingDocuments(ctx, client, execCtx.WorkspaceId, manifest)
			if err != nil {
				return nil, err
			}
		}
		manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("document_visual.parse: marshal manifest: %w", err)
		}
		manifestFileID := ""
		if boolPtrDefault(in.WriteManifestFile, true) {
			fileName := strings.TrimSpace(in.ManifestFileName)
			if fileName == "" || len(groups) > 1 {
				fileName = documentVisualManifestFileName(source.FileName, source.FileID)
			}
			resp, err := client.Files().UploadBytes(ctx, execCtx.WorkspaceId, fileName, manifestBytes)
			if err != nil {
				return nil, fmt.Errorf("document_visual.parse: upload manifest: %w", err)
			}
			manifestFileID = resp.FileID
		}
		manifests = append(manifests, manifest)
		manifestFileIDs = append(manifestFileIDs, manifestFileID)
	}
	manifest := manifests[0]
	out := documentVisualParseOutput{
		SchemaVersion:          documentVisualSchemaVersion,
		Profile:                profile,
		Manifest:               manifest,
		Manifests:              manifests,
		ManifestFileID:         firstString(manifestFileIDs),
		ManifestFileIDs:        manifestFileIDs,
		DerivedFileIDsBySource: documentVisualDerivedFileIDsBySource(manifests),
		Documents:              outDocs,
		Validation:             manifest.Validation,
	}
	payload, _ := json.Marshal(out)
	msg.Data = string(payload)
	return nil, nil
}

func (w *DocumentVisualIndexText) Handle(ctx context.Context, wctx moi.WorkItemContext, msg *mowl.MowlMessage) (*mowl.MowlMessage, error) {
	execCtx := wctx.ExecutionContext()
	var in documentVisualIndexTextInput
	if err := json.Unmarshal([]byte(msg.Data), &in); err != nil {
		return nil, fmt.Errorf("document_visual.index.text: parse input: %w", err)
	}
	if in.Enabled != nil && !*in.Enabled {
		out := documentVisualIndexTextOutput{
			Written:         0,
			DocumentsCount:  0,
			TextVectorTable: "",
			EmbeddingModel:  "",
			Status:          "disabled",
		}
		payload, _ := json.Marshal(out)
		msg.Data = string(payload)
		return nil, nil
	}
	tableName := firstNonEmptyString(strings.TrimSpace(in.TextVectorTable), strings.TrimSpace(in.TableName))
	if tableName == "" {
		return nil, fmt.Errorf("document_visual.index.text: text_vector_table or table_name is required")
	}
	if strings.TrimSpace(in.EmbeddingModel) == "" {
		return nil, fmt.Errorf("document_visual.index.text: embedding_model is required")
	}
	client, err := w.Factory.Get(execCtx)
	if err != nil {
		return nil, fmt.Errorf("document_visual.index.text: create SDK client: %w", err)
	}
	manifest, err := loadDocumentVisualManifest(ctx, client, execCtx.WorkspaceId, in.Manifest, in.ManifestFileID)
	if err != nil {
		return nil, err
	}
	docs, err := buildDocumentVisualTextIndexDocuments(manifest)
	if err != nil {
		return nil, err
	}
	enableMultilevel := false
	if in.EnableMultilevel != nil {
		enableMultilevel = *in.EnableMultilevel
	}
	if enableMultilevel {
		docs, err = expandDocumentVisualTextIndexDocuments(ctx, wctx, docs, in.SectionSize)
		if err != nil {
			return nil, fmt.Errorf("document_visual.index.text: multilevel_index: %w", err)
		}
	}
	indexVersion := documentsIndexVersion(docs)
	if indexVersion <= 0 {
		indexVersion = time.Now().UnixMilli()
		setDocumentVisualIndexVersion(docs, indexVersion)
	}
	writeInput := map[string]interface{}{
		"documents":           docs,
		"embedding_model":     in.EmbeddingModel,
		"embedding_dimension": in.EmbeddingDimension,
		"table_name":          tableName,
		"policy":              firstNonEmptyString(in.Policy, "OVERWRITE"),
		"file_id":             firstNonEmptyString(in.FileID, manifest.Source.FileID),
		"volume_id":           in.VolumeID,
		"dataset_meta_table":  in.DatasetMetaTable,
	}
	writePayload, _ := json.Marshal(writeInput)
	writeMsg := &mowl.MowlMessage{Data: string(writePayload)}
	if _, err := (&DataVectorWrite{Factory: w.Factory, Logger: w.Logger}).Handle(ctx, wctx, writeMsg); err != nil {
		return nil, fmt.Errorf("document_visual.index.text: write vector index: %w", err)
	}
	var writeOut struct {
		Written      int   `json:"written"`
		IndexVersion int64 `json:"index_version"`
	}
	if err := json.Unmarshal([]byte(writeMsg.Data), &writeOut); err != nil {
		return nil, fmt.Errorf("document_visual.index.text: decode vector write output: %w", err)
	}
	if writeOut.IndexVersion > 0 {
		indexVersion = writeOut.IndexVersion
	}
	out := documentVisualIndexTextOutput{
		Written:         writeOut.Written,
		DocumentsCount:  len(docs),
		TextVectorTable: tableName,
		EmbeddingModel:  in.EmbeddingModel,
		IndexVersion:    indexVersion,
		ManifestFileID:  in.ManifestFileID,
		Documents:       docs,
	}
	payload, _ := json.Marshal(out)
	msg.Data = string(payload)
	return nil, nil
}

func setDocumentVisualIndexVersion(docs []map[string]interface{}, indexVersion int64) {
	for _, doc := range docs {
		meta := ensureMap(doc["metadata"])
		meta["index_version"] = indexVersion
		doc["metadata"] = meta
	}
}

func documentsIndexVersion(docs []map[string]interface{}) int64 {
	var version int64
	for _, doc := range docs {
		meta := ensureMap(doc["metadata"])
		v := toInt64(meta["index_version"], 0)
		if v <= 0 {
			continue
		}
		if version == 0 {
			version = v
			continue
		}
		if version != v {
			return 0
		}
	}
	return version
}

func expandDocumentVisualTextIndexDocuments(ctx context.Context, wctx moi.WorkItemContext, docs []map[string]interface{}, sectionSize int) ([]map[string]interface{}, error) {
	chunkDocs := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		meta := ensureMap(doc["metadata"])
		if toString(meta["level"]) == "chunk" {
			chunkDocs = append(chunkDocs, doc)
		}
	}
	if len(chunkDocs) == 0 {
		return nil, fmt.Errorf("no chunk-level documents")
	}
	mlInput := map[string]interface{}{
		"documents": chunkDocs,
		"enable":    true,
	}
	if sectionSize > 0 {
		mlInput["section_size"] = sectionSize
	}
	mlPayload, _ := json.Marshal(mlInput)
	mlMsg := &mowl.MowlMessage{Data: string(mlPayload)}
	if _, err := (&MultiLevelIndex{}).Handle(ctx, wctx, mlMsg); err != nil {
		return nil, err
	}
	var mlOut struct {
		Documents []map[string]interface{} `json:"documents"`
	}
	if err := json.Unmarshal([]byte(mlMsg.Data), &mlOut); err != nil {
		return nil, fmt.Errorf("decode multilevel output: %w", err)
	}
	return mlOut.Documents, nil
}

func (w *DocumentVisualIndexImage) Handle(ctx context.Context, wctx moi.WorkItemContext, msg *mowl.MowlMessage) (*mowl.MowlMessage, error) {
	execCtx := wctx.ExecutionContext()
	var in documentVisualIndexImageInput
	if err := json.Unmarshal([]byte(msg.Data), &in); err != nil {
		return nil, fmt.Errorf("document_visual.index.image: parse input: %w", err)
	}
	if in.Enabled != nil && !*in.Enabled {
		out := documentVisualIndexImageOutput{
			Written:              0,
			PageRows:             0,
			VisualObjectRows:     0,
			DocumentsCount:       0,
			ImageVectorTable:     "",
			EmbeddingModel:       "",
			EmbeddingDimension:   0,
			EmbeddingBackendID:   "",
			PreprocessVersion:    "",
			DistanceMetric:       "",
			EmbeddingSource:      firstNonEmptyString(strings.TrimSpace(in.EmbeddingSource), "real"),
			IndexVersion:         in.IndexVersion,
			AllSourceFileIDs:     []string{},
			IndexedSourceFileIDs: []string{},
			SourceFileIDs:        []string{},
			FileStatuses:         nil,
			Status:               "disabled",
		}
		payload, _ := json.Marshal(out)
		msg.Data = string(payload)
		return nil, nil
	}
	tableName := firstNonEmptyString(strings.TrimSpace(in.ImageVectorTable), strings.TrimSpace(in.TableName))
	if tableName == "" {
		return nil, fmt.Errorf("document_visual.index.image: image_vector_table or table_name is required")
	}
	model := firstNonEmptyString(strings.TrimSpace(in.ImageEmbeddingModel), strings.TrimSpace(in.EmbeddingModel))
	if model == "" {
		return nil, fmt.Errorf("document_visual.index.image: image_embedding_model is required")
	}
	dimension := firstPositiveInt(in.ImageEmbeddingDimension, in.EmbeddingDimension)
	if dimension <= 0 {
		return nil, fmt.Errorf("document_visual.index.image: image_embedding_dimension is required")
	}
	preprocessVersion := strings.TrimSpace(in.PreprocessVersion)
	if preprocessVersion == "" {
		return nil, fmt.Errorf("document_visual.index.image: preprocess_version is required")
	}
	distanceMetric := strings.TrimSpace(in.DistanceMetric)
	if distanceMetric == "" {
		return nil, fmt.Errorf("document_visual.index.image: distance_metric is required")
	}
	embeddingSource := firstNonEmptyString(strings.TrimSpace(in.EmbeddingSource), "real")
	if embeddingSource != "real" {
		return nil, fmt.Errorf("document_visual.index.image: embedding_source must be real for production indexing, got %q", embeddingSource)
	}
	if in.IndexVersion <= 0 {
		return nil, fmt.Errorf("document_visual.index.image: index_version is required")
	}
	client, err := w.Factory.Get(execCtx)
	if err != nil {
		return nil, fmt.Errorf("document_visual.index.image: create SDK client: %w", err)
	}
	manifests, manifestFileIDs, err := loadDocumentVisualIndexImageManifests(ctx, client, execCtx.WorkspaceId, in)
	if err != nil {
		return nil, err
	}
	docs := make([]VectorDoc, 0)
	counts := map[string]int{}
	sourceFileIDs := make([]string, 0, len(manifests))
	indexedSourceFileIDs := make([]string, 0, len(manifests))
	fileStatuses := make([]documentVisualImageIndexFileStatus, 0, len(manifests))
	for i, manifest := range manifests {
		manifestFileID := ""
		if i < len(manifestFileIDs) {
			manifestFileID = manifestFileIDs[i]
		}
		indexDocs, indexCounts, err := buildDocumentVisualImageIndexDocuments(ctx, client, execCtx.WorkspaceId, manifest, documentVisualImageIndexOptions{
			ManifestFileID:          manifestFileID,
			ImageEmbeddingModel:     model,
			ImageEmbeddingBackendID: firstNonEmptyString(strings.TrimSpace(in.ImageEmbeddingBackendID), strings.TrimSpace(in.EmbeddingBackendID)),
			EmbeddingDimension:      dimension,
			PreprocessVersion:       preprocessVersion,
			DistanceMetric:          distanceMetric,
			EmbeddingSource:         embeddingSource,
			IndexVersion:            in.IndexVersion,
			Scopes:                  in.Scopes,
			VolumeID:                in.VolumeID,
			AllowEmpty:              boolPtrDefault(in.AllowEmpty, false),
		})
		// allow_empty only treats "parsed content has no page/object images" as
		// a valid no-op. Image download, embedding, vector schema, and write
		// failures must still return errors to the workflow.
		if err != nil {
			return nil, err
		}
		docs = append(docs, indexDocs...)
		counts["page"] += indexCounts["page"]
		counts["visual_object"] += indexCounts["visual_object"]
		if manifest.Source.FileID != "" {
			sourceFileIDs = append(sourceFileIDs, manifest.Source.FileID)
			if len(indexDocs) > 0 {
				indexedSourceFileIDs = append(indexedSourceFileIDs, manifest.Source.FileID)
			}
			status := "no_indexable_images"
			if len(indexDocs) > 0 {
				status = "ready"
			}
			fileStatuses = append(fileStatuses, documentVisualImageIndexFileStatus{
				SourceFileID:  manifest.Source.FileID,
				Status:        status,
				IndexedImages: len(indexDocs),
			})
		}
	}

	db, table, err := openWorkspaceVectorDB(ctx, w.Factory, execCtx, tableName)
	if err != nil {
		return nil, fmt.Errorf("document_visual.index.image: open vector DB: %w", err)
	}
	defer db.Close()
	if err := ensureVectorTable(db, table, dimension); err != nil {
		return nil, fmt.Errorf("document_visual.index.image: ensure image vector table: %w", err)
	}

	policy := strings.ToUpper(strings.TrimSpace(in.Policy))
	if policy == "" {
		policy = "OVERWRITE"
	}
	if policy == "OVERWRITE" {
		for _, sourceFileID := range compactStrings(sourceFileIDs) {
			deleted, err := deleteDocumentVisualImageIndexRows(ctx, db, table, sourceFileID, model, preprocessVersion, distanceMetric, in.IndexVersion)
			if err != nil {
				return nil, err
			}
			if w.Logger != nil {
				w.Logger.Info("document visual image index overwrite scope cleared",
					zap.String("table", table),
					zap.String("source_file_id", sourceFileID),
					zap.String("embedding_model", model),
					zap.String("preprocess_version", preprocessVersion),
					zap.String("distance_metric", distanceMetric),
					zap.Int64("deleted", deleted),
				)
			}
		}
		policy = "FAIL"
	}
	if len(docs) == 0 {
		out := documentVisualIndexImageOutput{
			Written:              0,
			PageRows:             counts["page"],
			VisualObjectRows:     counts["visual_object"],
			DocumentsCount:       0,
			ImageVectorTable:     table,
			EmbeddingModel:       model,
			EmbeddingDimension:   dimension,
			EmbeddingBackendID:   firstNonEmptyString(strings.TrimSpace(in.ImageEmbeddingBackendID), strings.TrimSpace(in.EmbeddingBackendID)),
			PreprocessVersion:    preprocessVersion,
			DistanceMetric:       distanceMetric,
			EmbeddingSource:      embeddingSource,
			IndexVersion:         in.IndexVersion,
			ManifestFileID:       firstString(manifestFileIDs),
			SourceFileID:         "",
			AllSourceFileIDs:     compactStrings(sourceFileIDs),
			IndexedSourceFileIDs: []string{},
			SourceFileIDs:        []string{},
			FileStatuses:         fileStatuses,
			Status:               "no_indexable_images",
		}
		payload, _ := json.Marshal(out)
		msg.Data = string(payload)
		return nil, nil
	}
	written, err := upsertVectorRows(ctx, db, table, docs, policy)
	if err != nil {
		return nil, fmt.Errorf("document_visual.index.image: write image vector rows: %w", err)
	}
	out := documentVisualIndexImageOutput{
		Written:              written,
		PageRows:             counts["page"],
		VisualObjectRows:     counts["visual_object"],
		DocumentsCount:       len(docs),
		ImageVectorTable:     table,
		EmbeddingModel:       model,
		EmbeddingDimension:   dimension,
		EmbeddingBackendID:   firstNonEmptyString(strings.TrimSpace(in.ImageEmbeddingBackendID), strings.TrimSpace(in.EmbeddingBackendID)),
		PreprocessVersion:    preprocessVersion,
		DistanceMetric:       distanceMetric,
		EmbeddingSource:      embeddingSource,
		IndexVersion:         in.IndexVersion,
		ManifestFileID:       firstString(manifestFileIDs),
		SourceFileID:         firstString(indexedSourceFileIDs),
		AllSourceFileIDs:     compactStrings(sourceFileIDs),
		IndexedSourceFileIDs: compactStrings(indexedSourceFileIDs),
		SourceFileIDs:        compactStrings(indexedSourceFileIDs),
		FileStatuses:         fileStatuses,
		Status:               "ready",
	}
	payload, _ := json.Marshal(out)
	msg.Data = string(payload)
	return nil, nil
}

func documentVisualParsePayload(in documentVisualParseInput) (string, error) {
	payload := map[string]interface{}{}
	if len(in.Sources) > 0 {
		payload["sources"] = in.Sources
	}
	if in.FileID != "" {
		payload["file_id"] = in.FileID
	}
	if len(in.FileIDs) > 0 {
		payload["file_ids"] = in.FileIDs
	}
	options := map[string]interface{}{}
	for k, v := range in.Options {
		options[k] = v
	}
	if in.VLMModel != "" {
		options["vlm_ocr_model"] = in.VLMModel
		if _, ok := options["image_process_type"]; !ok {
			options["image_process_type"] = []string{"ocr", "caption"}
		}
	}
	if len(options) > 0 {
		payload["options"] = options
	}
	if len(payload) == 0 {
		return "", fmt.Errorf("document_visual.parse: sources, file_id, file_ids or documents is required")
	}
	data, _ := json.Marshal(payload)
	return string(data), nil
}

func documentVisualEngineeringOptionsFromInput(profile string, options map[string]interface{}) (documentVisualEngineeringOptions, error) {
	opts := documentVisualEngineeringOptions{
		EnablePagePlan:      getBool(options, "enable_engineering_page_plan"),
		EnableRegionExtract: getBool(options, "enable_engineering_region_extract"),
		Model:               toString(options["engineering_vlm_model"]),
		ReasoningEffort:     toString(options["engineering_vlm_reasoning_effort"]),
	}
	if opts.EnablePagePlan != opts.EnableRegionExtract {
		return opts, fmt.Errorf("document_visual.parse: enable_engineering_page_plan and enable_engineering_region_extract must be enabled together")
	}
	if (opts.EnablePagePlan || opts.EnableRegionExtract) && profile != documentVisualDefaultProfile {
		return opts, fmt.Errorf("document_visual.parse: engineering drawing enhancement requires profile=%s, got %q", documentVisualDefaultProfile, profile)
	}
	if opts.Enabled() && opts.Model == "" {
		return opts, fmt.Errorf("document_visual.parse: engineering_vlm_model is required when engineering drawing enhancement is enabled")
	}
	return opts, nil
}

func (o documentVisualEngineeringOptions) Enabled() bool {
	return o.EnablePagePlan && o.EnableRegionExtract
}

type documentVisualParseDocumentGroup struct {
	FileID    string
	Sources   []*data.Source
	Documents []*data.Document
}

func documentVisualParseDocumentGroups(in documentVisualParseInput, docs []*data.Document) []documentVisualParseDocumentGroup {
	if len(docs) == 0 {
		return nil
	}
	order := make([]string, 0)
	groups := map[string]*documentVisualParseDocumentGroup{}
	for _, doc := range docs {
		fileID := documentVisualDocumentSourceFileID(doc)
		if fileID == "" {
			fileID = firstNonEmptyString(in.FileID, firstString(in.FileIDs))
		}
		if fileID == "" && len(in.Sources) == 1 {
			fileID = in.Sources[0].GetFileId()
		}
		if _, ok := groups[fileID]; !ok {
			order = append(order, fileID)
			groups[fileID] = &documentVisualParseDocumentGroup{FileID: fileID, Sources: documentVisualSourcesForFileID(in.Sources, fileID)}
		}
		groups[fileID].Documents = append(groups[fileID].Documents, doc)
	}
	out := make([]documentVisualParseDocumentGroup, 0, len(order))
	for _, fileID := range order {
		out = append(out, *groups[fileID])
	}
	return out
}

func documentVisualDocumentSourceFileID(doc *data.Document) string {
	d := documentFromProto(doc)
	return canonicalDocumentSourceID(d.Metadata)
}

func documentVisualSourcesForFileID(sources []*data.Source, fileID string) []*data.Source {
	if len(sources) == 0 {
		return nil
	}
	fileID = strings.TrimSpace(fileID)
	if fileID == "" && len(sources) == 1 {
		return sources
	}
	for _, source := range sources {
		if source.GetFileId() == fileID {
			return []*data.Source{source}
		}
	}
	return nil
}

func (p *DocumentVisualParse) preparePageAssets(ctx context.Context, client *moi.Client, workspaceID string, in documentVisualParseInput, docs []*data.Document, layout documentVisualLayout) (map[int]documentVisualPageAsset, documentVisualSource, error) {
	source := inferDocumentVisualSource(in, docs)
	pageAssets := map[int]documentVisualPageAsset{}
	if source.FileID == "" {
		return pageAssets, source, nil
	}
	raw, err := client.Files().DownloadBytes(ctx, workspaceID, source.FileID)
	if err != nil {
		return nil, source, fmt.Errorf("document_visual.parse: download source file %s: %w", source.FileID, err)
	}
	ext := strings.ToLower(path.Ext(source.FileName))
	if strings.Contains(strings.ToLower(source.MimeType), "pdf") || ext == ".pdf" {
		blocks, err := (&PDFProcessor{Source: BlockSource{FileID: source.FileID, FileName: source.FileName}}).Process(ctx, raw, nil)
		if err != nil {
			return nil, source, fmt.Errorf("document_visual.parse: render PDF page images: %w", err)
		}
		for _, block := range blocks {
			pageNumber := block.Source.PageNumber
			imgBytes, err := base64.StdEncoding.DecodeString(block.Content)
			if err != nil {
				return nil, source, fmt.Errorf("document_visual.parse: decode rendered page %d: %w", pageNumber, err)
			}
			resp, err := client.Files().UploadBytes(ctx, workspaceID, documentVisualPageImageName(source.FileName, pageNumber), imgBytes)
			if err != nil {
				return nil, source, fmt.Errorf("document_visual.parse: upload page image %d: %w", pageNumber, err)
			}
			width, height := imageSize(imgBytes)
			pageAssets[pageNumber] = documentVisualPageAsset{FileID: resp.FileID, Bytes: imgBytes, Width: width, Height: height, MimeType: "image/jpeg"}
		}
		source.PageCount = len(pageAssets)
		return pageAssets, source, nil
	}
	if isDocumentVisualImageSource(source, raw) {
		respID := source.FileID
		width, height := imageSize(raw)
		pageAssets[1] = documentVisualPageAsset{FileID: respID, Bytes: raw, Width: width, Height: height}
		source.PageCount = 1
		return pageAssets, source, nil
	}
	if len(layout.Pages) > 0 {
		source.PageCount = len(layout.Pages)
	}
	return pageAssets, source, nil
}

func (p *DocumentVisualParse) prepareEngineeringDrawingPageAssets(ctx context.Context, client *moi.Client, workspaceID string, source documentVisualSource) (map[int]documentVisualPageAsset, error) {
	raw, err := client.Files().DownloadBytes(ctx, workspaceID, source.FileID)
	if err != nil {
		return nil, fmt.Errorf("document_visual.parse: download engineering drawing PDF %s: %w", source.FileID, err)
	}
	tmpDir, err := os.MkdirTemp("", "moi-engineering-pdf-*")
	if err != nil {
		return nil, fmt.Errorf("document_visual.parse: create engineering render temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	pdfPath := filepath.Join(tmpDir, "input.pdf")
	if err := os.WriteFile(pdfPath, raw, 0600); err != nil {
		return nil, fmt.Errorf("document_visual.parse: write engineering render PDF: %w", err)
	}
	outPrefix := filepath.Join(tmpDir, "page")
	if err := renderEngineeringDrawingPDFPages(ctx, pdfPath, outPrefix, nil); err != nil {
		return nil, fmt.Errorf("document_visual.parse: render engineering PDF page images: %w", err)
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return nil, fmt.Errorf("document_visual.parse: read engineering render temp dir: %w", err)
	}
	imgPaths := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") {
			imgPaths = append(imgPaths, filepath.Join(tmpDir, name))
		}
	}
	sortRenderedPDFImages(imgPaths)

	pageAssets := map[int]documentVisualPageAsset{}
	for i, imgPath := range imgPaths {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		imgBytes, err := os.ReadFile(imgPath)
		if err != nil {
			return nil, fmt.Errorf("document_visual.parse: read engineering page image %s: %w", filepath.Base(imgPath), err)
		}
		pageNumber := renderedPDFPageNumber(imgPath, i, nil)
		resp, err := client.Files().UploadBytes(ctx, workspaceID, documentVisualPageImageNameWithExt(source.FileName, pageNumber, "jpg"), imgBytes)
		if err != nil {
			return nil, fmt.Errorf("document_visual.parse: upload engineering page image %d: %w", pageNumber, err)
		}
		width, height := imageSize(imgBytes)
		pageAssets[pageNumber] = documentVisualPageAsset{FileID: resp.FileID, Bytes: imgBytes, Width: width, Height: height, MimeType: engineeringDrawingPDFImage}
	}
	if len(pageAssets) == 0 {
		return nil, fmt.Errorf("document_visual.parse: engineering PDF render produced no page images")
	}
	return pageAssets, nil
}

func buildDocumentVisualManifest(ctx context.Context, client *moi.Client, workspaceID string, docs []*data.Document, layout documentVisualLayout, pageAssets map[int]documentVisualPageAsset, source documentVisualSource, profile string, opts documentVisualBuildOptions) (documentVisualManifest, error) {
	documents := make([]Document, 0, len(docs))
	for _, doc := range docs {
		documents = append(documents, documentFromProto(doc))
	}
	figureCaptionRelations, err := documentVisualFigureCaptionRelations(documents)
	if err != nil {
		return documentVisualManifest{}, fmt.Errorf("document_visual.parse: validate figure-caption relations: %w", err)
	}
	if profile == "" {
		profile = documentVisualDefaultProfile
	}
	layoutByPage := map[int]documentVisualLayoutPage{}
	for _, page := range layout.Pages {
		pageNum := page.PageNumber
		if pageNum <= 0 {
			pageNum = page.PageIdx + 1
		}
		layoutByPage[pageNum] = page
	}
	linkedCaptionBlocks := documentVisualLinkedCaptionBlockIDs(documents, figureCaptionRelations)
	textByPage := map[int][]string{}
	for _, doc := range documents {
		if _, linked := linkedCaptionBlocks[getString(doc.Metadata, "block_uuid")]; linked {
			continue
		}
		pageNum := documentVisualDocPage(doc)
		if pageNum <= 0 {
			pageNum = 1
		}
		if text := documentVisualDocText(doc); text != "" {
			textByPage[pageNum] = append(textByPage[pageNum], text)
		}
	}
	pages := make([]documentVisualPage, 0)
	pageNumbers := map[int]struct{}{}
	for pageNum := range textByPage {
		pageNumbers[pageNum] = struct{}{}
	}
	for pageNum := range layoutByPage {
		pageNumbers[pageNum] = struct{}{}
	}
	for pageNum := range pageAssets {
		pageNumbers[pageNum] = struct{}{}
	}
	pageList := make([]int, 0, len(pageNumbers))
	for pageNum := range pageNumbers {
		pageList = append(pageList, pageNum)
	}
	sort.Ints(pageList)
	pageIndex := map[int]int{}
	for _, pageNum := range pageList {
		page := documentVisualPage{PageNumber: pageNum, Text: joinNonEmpty(textByPage[pageNum], "\n")}
		page.Summary = documentVisualPageSummary(page.Text)
		if asset, ok := pageAssets[pageNum]; ok {
			page.PageImageFileID = asset.FileID
			if asset.Width > 0 && asset.Height > 0 {
				page.Width = float64(asset.Width)
				page.Height = float64(asset.Height)
				page.BBox = []float64{0, 0, page.Width, page.Height}
			}
		}
		if layoutPage, ok := layoutByPage[pageNum]; ok && len(layoutPage.PageSize) >= 2 {
			page.Width = layoutPage.PageSize[0]
			page.Height = layoutPage.PageSize[1]
			page.BBox = []float64{0, 0, page.Width, page.Height}
		}
		pageIndex[pageNum] = len(pages)
		pages = append(pages, page)
	}

	objects := make([]documentVisualObject, 0)
	for i, doc := range documents {
		if !documentVisualDocIsVisual(doc) {
			continue
		}
		pageNum := documentVisualDocPage(doc)
		if pageNum <= 0 {
			pageNum = 1
		}
		layoutBlock := matchDocumentVisualLayoutBlock(doc, layoutByPage[pageNum])
		bbox := documentVisualDocBBox(doc)
		if len(bbox) == 0 {
			bbox = layoutBlock.BBox
		}
		semantics := extractImageSemantics(doc.Metadata)
		obj := documentVisualObject{
			ObjectID:         documentVisualObjectID(source.FileID, pageNum, i, doc),
			ObjectKind:       documentVisualObjectKind(doc),
			PageNumber:       pageNum,
			BBox:             bbox,
			ImageFileID:      normalizeParsedImageFileID(firstNonEmptyString(getString(doc.Metadata, "image_file_id"), getString(doc.Metadata, "image_url"), getString(doc.Metadata, "s3_image_url"), getString(doc.Metadata, "table_image_url"))),
			PageImageFileID:  documentVisualPageImageID(pages, pageIndex, pageNum),
			Text:             strings.TrimSpace(doc.Content),
			OCR:              semantics.OCR,
			FigureCaption:    semantics.FigureCaption,
			FigureNo:         getString(doc.Metadata, "figure_no"),
			Caption:          semantics.GeneratedCaption,
			CaptionBlockUUID: getString(doc.Metadata, "caption_block_uuid"),
			SourceBlockID:    firstNonEmptyString(getString(doc.Metadata, "block_uuid"), getString(doc.Metadata, "source_block_id")),
		}
		if profile == documentVisualStandardRAGProfile && !opts.RequireVisualContext {
			obj.Context = documentVisualObjectLocalContext(obj)
		} else {
			obj.Context = documentVisualObjectContext(obj, pages, pageIndex)
		}
		if obj.ImageFileID == "" && obj.PageImageFileID != "" && len(obj.BBox) == 4 {
			pageWidth, pageHeight := pageDimensions(pages, pageIndex, pageNum)
			croppedID, err := cropAndUploadDocumentVisualObject(ctx, client, workspaceID, source.FileName, obj, pageAssets[pageNum], pageWidth, pageHeight)
			if err != nil {
				return documentVisualManifest{}, err
			}
			obj.ImageFileID = croppedID
		}
		objects = append(objects, obj)
		if idx, ok := pageIndex[pageNum]; ok {
			pages[idx].ObjectIDs = append(pages[idx].ObjectIDs, obj.ObjectID)
		}
	}
	manifest := documentVisualManifest{
		SchemaVersion: documentVisualSchemaVersion,
		Profile:       profile,
		Source:        source,
		Pages:         pages,
		Objects:       objects,
		Entities:      collectDocumentVisualEntities(documents, source.FileID),
	}
	manifest.Validation = validateDocumentVisualManifest(manifest, opts)
	if !manifest.Validation.Valid {
		return manifest, fmt.Errorf("document_visual.parse: manifest validation failed: %s", strings.Join(manifest.Validation.Errors, "; "))
	}
	return manifest, nil
}

func documentVisualDerivedFileIDsBySource(manifests []documentVisualManifest) map[string][]string {
	bySource := make(map[string][]string, len(manifests))
	for _, manifest := range manifests {
		sourceFileID := strings.TrimSpace(manifest.Source.FileID)
		if sourceFileID == "" {
			continue
		}
		fileIDs := make([]string, 0, len(manifest.Pages)+len(manifest.Objects)*2)
		for _, page := range manifest.Pages {
			fileIDs = append(fileIDs, page.PageImageFileID)
		}
		for _, object := range manifest.Objects {
			fileIDs = append(fileIDs, object.ImageFileID, object.PageImageFileID)
		}
		fileIDs = compactStrings(fileIDs)
		derived := make([]string, 0, len(fileIDs))
		for _, fileID := range fileIDs {
			if fileID != sourceFileID {
				derived = append(derived, fileID)
			}
		}
		if len(derived) > 0 {
			bySource[sourceFileID] = derived
		}
	}
	return bySource
}

func stripDocumentVisualJSON(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	lines := strings.Split(s, "\n")
	var b strings.Builder
	for i, line := range lines {
		if i == 0 {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			break
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}

func loadDocumentVisualManifest(ctx context.Context, client *moi.Client, workspaceID string, manifest documentVisualManifest, fileID string) (documentVisualManifest, error) {
	if manifest.SchemaVersion != "" || manifest.Source.FileID != "" || len(manifest.Pages) > 0 || len(manifest.Objects) > 0 {
		return manifest, nil
	}
	if strings.TrimSpace(fileID) == "" {
		return documentVisualManifest{}, fmt.Errorf("document_visual: manifest or manifest_file_id is required")
	}
	raw, err := client.Files().DownloadBytes(ctx, workspaceID, fileID)
	if err != nil {
		return documentVisualManifest{}, fmt.Errorf("document_visual: download manifest %s: %w", fileID, err)
	}
	var loaded documentVisualManifest
	if err := json.Unmarshal(raw, &loaded); err != nil {
		return documentVisualManifest{}, fmt.Errorf("document_visual: parse manifest %s: %w", fileID, err)
	}
	return loaded, nil
}

func loadDocumentVisualIndexImageManifests(ctx context.Context, client *moi.Client, workspaceID string, in documentVisualIndexImageInput) ([]documentVisualManifest, []string, error) {
	if len(in.Manifests) > 0 {
		return in.Manifests, append([]string(nil), in.ManifestFileIDs...), nil
	}
	if len(in.ManifestFileIDs) > 0 {
		manifests := make([]documentVisualManifest, 0, len(in.ManifestFileIDs))
		for _, fileID := range in.ManifestFileIDs {
			manifest, err := loadDocumentVisualManifest(ctx, client, workspaceID, documentVisualManifest{}, fileID)
			if err != nil {
				return nil, nil, err
			}
			manifests = append(manifests, manifest)
		}
		return manifests, append([]string(nil), in.ManifestFileIDs...), nil
	}
	manifest, err := loadDocumentVisualManifest(ctx, client, workspaceID, in.Manifest, in.ManifestFileID)
	if err != nil {
		return nil, nil, err
	}
	return []documentVisualManifest{manifest}, []string{in.ManifestFileID}, nil
}

func loadDocumentVisualLayout(ctx context.Context, client *moi.Client, workspaceID string, inline map[string]interface{}, docs []*data.Document) (documentVisualLayout, error) {
	if len(inline) > 0 {
		raw, _ := json.Marshal(inline)
		var layout documentVisualLayout
		if err := json.Unmarshal(raw, &layout); err != nil {
			return documentVisualLayout{}, fmt.Errorf("document_visual.parse: parse inline layout: %w", err)
		}
		return layout, nil
	}
	layoutFileID := ""
	for _, doc := range docs {
		d := documentFromProto(doc)
		layoutFileID = firstNonEmptyString(layoutFileID, getString(d.Metadata, "layout_file_id"))
	}
	if layoutFileID == "" {
		return documentVisualLayout{}, nil
	}
	raw, err := client.Files().DownloadBytes(ctx, workspaceID, layoutFileID)
	if err != nil {
		return documentVisualLayout{}, fmt.Errorf("document_visual.parse: download layout %s: %w", layoutFileID, err)
	}
	var layout documentVisualLayout
	if err := json.Unmarshal(raw, &layout); err != nil {
		return documentVisualLayout{}, fmt.Errorf("document_visual.parse: parse layout %s: %w", layoutFileID, err)
	}
	return layout, nil
}

func floatSliceMetadataValue(values []float64) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func stringSliceMetadataValue(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func documentVisualIndexDocument(docType, id, content string, meta map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"id":       id,
		"type":     docType,
		"content":  strings.TrimSpace(content),
		"metadata": meta,
	}
}

func validateDocumentVisualManifest(manifest documentVisualManifest, opts documentVisualBuildOptions) documentVisualValidation {
	var errors []string
	if manifest.Source.FileID == "" {
		errors = append(errors, "source.file_id is required")
	}
	if len(manifest.Pages) == 0 {
		errors = append(errors, "pages is empty")
	}
	for _, page := range manifest.Pages {
		if opts.RequirePageImages && page.PageImageFileID == "" {
			errors = append(errors, fmt.Sprintf("page %d missing page_image_file_id", page.PageNumber))
		}
	}
	for _, obj := range manifest.Objects {
		if obj.ObjectID == "" {
			errors = append(errors, fmt.Sprintf("page %d visual object missing object_id", obj.PageNumber))
		}
		if (opts.RequireObjectImages || opts.RequirePageImages || opts.RequireVisualContext) && len(obj.BBox) != 4 {
			errors = append(errors, fmt.Sprintf("object %s missing bbox", obj.ObjectID))
		}
		if opts.RequireObjectImages && obj.ImageFileID == "" {
			errors = append(errors, fmt.Sprintf("object %s missing image_file_id", obj.ObjectID))
		}
		if opts.RequirePageImages && obj.PageImageFileID == "" {
			errors = append(errors, fmt.Sprintf("object %s missing page_image_file_id", obj.ObjectID))
		}
		if opts.RequireVisualContext && strings.TrimSpace(obj.Context) == "" {
			errors = append(errors, fmt.Sprintf("object %s missing context", obj.ObjectID))
		}
	}
	objectIDs := map[string]struct{}{}
	for _, obj := range manifest.Objects {
		if obj.ObjectID != "" {
			objectIDs[obj.ObjectID] = struct{}{}
		}
	}
	for _, page := range manifest.Pages {
		objectIDs[documentVisualPageObjectID(manifest.Source.FileID, page.PageNumber)] = struct{}{}
	}
	for _, entity := range manifest.Entities {
		if entity.ObjectID == "" {
			continue
		}
		if toString(entity.Metadata["extraction_source"]) != "engineering_region_extract" {
			continue
		}
		if _, ok := objectIDs[entity.ObjectID]; !ok {
			errors = append(errors, fmt.Sprintf("entity %s references unknown object_id %s", firstNonEmptyString(entity.EntityID, entity.Value), entity.ObjectID))
		}
	}
	return documentVisualValidation{Valid: len(errors) == 0, Errors: errors}
}

func inferDocumentVisualSource(in documentVisualParseInput, docs []*data.Document) documentVisualSource {
	source := documentVisualSource{}
	source.FileID = firstNonEmptyString(in.FileID, firstString(in.FileIDs))
	if len(in.Sources) > 0 {
		source.FileID = firstNonEmptyString(in.Sources[0].GetFileId(), source.FileID)
		source.FileName = firstNonEmptyString(in.Sources[0].GetName(), source.FileName)
		source.MimeType = firstNonEmptyString(in.Sources[0].GetMimeType(), source.MimeType)
	}
	for _, doc := range docs {
		d := documentFromProto(doc)
		source.FileID = firstNonEmptyString(source.FileID, canonicalDocumentSourceID(d.Metadata))
		source.FileName = firstNonEmptyString(source.FileName, getString(d.Metadata, "file_name"), getString(d.Metadata, "source_file_name"))
		source.MimeType = firstNonEmptyString(source.MimeType, getString(d.Metadata, "mime_type"))
	}
	if source.FileName == "" {
		source.FileName = source.FileID
	}
	return source
}

func documentVisualDocPage(doc Document) int {
	return firstPositiveInt(
		toInt(doc.Metadata["page_number"], 0),
		toInt(doc.Metadata["page_num"], 0),
		toInt(doc.Metadata["page"], 0),
	)
}

func documentVisualDocText(doc Document) string {
	if documentVisualDocIsImage(doc) {
		return extractImageSemantics(doc.Metadata).
			projectWithLocalText("\n", doc.Content, getString(doc.Metadata, "html")).Text
	}
	return joinNonEmpty([]string{
		doc.Content,
		getString(doc.Metadata, "ocr"),
		getString(doc.Metadata, "caption"),
		getString(doc.Metadata, "html"),
	}, "\n")
}

// documentVisualLinkedCaptionBlockIDs identifies only complete source-owned
// figure relations whose visible caption text is already present in the image
// projection. Suppressing that one reciprocal block prevents duplicate page
// text without guessing from equal strings or hiding dangling relations.
func documentVisualLinkedCaptionBlockIDs(documents []Document, relations []figureCaptionRelation) map[string]struct{} {
	linked := make(map[string]struct{})
	for _, relation := range relations {
		linked[getString(documents[relation.CaptionIndex].Metadata, "block_uuid")] = struct{}{}
	}
	return linked
}

func documentVisualFigureCaptionRelationsFromProto(documents []*data.Document) ([]figureCaptionRelation, error) {
	projected := make([]Document, 0, len(documents))
	for _, document := range documents {
		projected = append(projected, documentFromProto(document))
	}
	return documentVisualFigureCaptionRelations(projected)
}

func documentVisualFigureCaptionRelations(documents []Document) ([]figureCaptionRelation, error) {
	nodes := make([]figureCaptionNode, len(documents))
	for index := range documents {
		document := documents[index]
		nodes[index] = newFigureCaptionNode(
			document.Metadata,
			documentVisualDocKind(document),
			document.Content,
			canonicalDocumentSourceID(document.Metadata),
			"page_number", "page_num", "page",
		)
	}
	return validateFigureCaptionRelations(nodes)
}

func documentVisualDocIsImage(doc Document) bool {
	return documentVisualDocKind(doc) == "image"
}

func documentVisualDocIsVisual(doc Document) bool {
	switch documentVisualDocKind(doc) {
	case "image", "table":
		return true
	default:
		return false
	}
}

func documentVisualObjectKind(doc Document) string {
	if documentVisualDocKind(doc) == "table" {
		return "table"
	}
	return "drawing_view"
}

// documentVisualDocKind is the sole authority for interpreting the three type
// surfaces accepted by document_visual. A visual discriminator may refine a
// generic top-level type, while source priority remains Document.Type,
// metadata.block_type, then metadata.type.
func documentVisualDocKind(doc Document) string {
	candidates := []string{
		doc.Type,
		getString(doc.Metadata, "block_type"),
		getString(doc.Metadata, "type"),
	}
	for _, candidate := range candidates {
		switch kind := strings.ToLower(strings.TrimSpace(candidate)); kind {
		case "image", "table":
			return kind
		}
	}
	for _, candidate := range candidates {
		if kind := strings.ToLower(strings.TrimSpace(candidate)); kind != "" {
			return kind
		}
	}
	return ""
}

func documentVisualObjectID(sourceFileID string, pageNum, pos int, doc Document) string {
	if id := firstNonEmptyString(getString(doc.Metadata, "object_id"), getString(doc.Metadata, "block_uuid")); id != "" {
		return id
	}
	return hashShort("visual-object", sourceFileID, fmt.Sprint(pageNum), fmt.Sprint(pos), doc.Type, doc.Content)
}

func documentVisualDocBBox(doc Document) []float64 {
	for _, key := range []string{"bbox", "bounding_box"} {
		if bbox := interfaceToFloat64Slice(doc.Metadata[key]); len(bbox) == 4 {
			return bbox
		}
	}
	return nil
}

func matchDocumentVisualLayoutBlock(doc Document, page documentVisualLayoutPage) documentVisualLayoutBlock {
	if len(page.ParaBlocks) == 0 {
		return documentVisualLayoutBlock{}
	}
	inner := toInt(doc.Metadata["page_inner_number"], -1)
	if inner >= 0 && inner < len(page.ParaBlocks) {
		return page.ParaBlocks[inner]
	}
	target := documentVisualDocKind(doc)
	nth := toInt(doc.Metadata["index"], -1)
	count := 0
	for _, block := range page.ParaBlocks {
		if strings.EqualFold(block.Type, target) {
			if nth < 0 || count == nth {
				return block
			}
			count++
		}
	}
	return documentVisualLayoutBlock{}
}

func documentVisualObjectContext(obj documentVisualObject, pages []documentVisualPage, pageIndex map[int]int) string {
	pageText := ""
	if idx, ok := pageIndex[obj.PageNumber]; ok {
		pageText = pages[idx].Text
	}
	if strings.TrimSpace(pageText) != "" {
		return pageText
	}
	return documentVisualObjectLocalContext(obj)
}

func documentVisualObjectLocalContext(obj documentVisualObject) string {
	if obj.ObjectKind == "drawing_view" {
		return (imageSemantics{
			OCR:              obj.OCR,
			FigureCaption:    obj.FigureCaption,
			GeneratedCaption: obj.Caption,
		}).projectWithLocalText("\n", obj.Text).Text
	}
	return joinNonEmpty([]string{obj.Caption, obj.OCR, obj.Text}, "\n")
}

func collectDocumentVisualEntities(docs []Document, sourceFileID string) []documentVisualEntity {
	entities := make([]documentVisualEntity, 0)
	seen := map[string]struct{}{}
	for _, doc := range docs {
		for _, item := range documentVisualMetadataEntities(doc.Metadata) {
			key := strings.Join([]string{item.Type, item.Value, fmt.Sprint(item.PageNumber), item.ObjectID}, "\x00")
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			if item.EntityID == "" {
				item.EntityID = hashShort("entity", sourceFileID, key)
			}
			entities = append(entities, item)
		}
	}
	return entities
}

func documentVisualMetadataEntities(meta map[string]interface{}) []documentVisualEntity {
	raw, ok := meta["entities"]
	if !ok {
		raw = meta["extraction_result"]
	}
	if raw == nil {
		return nil
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	out := make([]documentVisualEntity, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		entity := documentVisualEntity{
			EntityID:   toString(m["entity_id"]),
			Type:       toString(m["type"]),
			Value:      toString(m["value"]),
			PageNumber: toInt(m["page_number"], 0),
			ObjectID:   toString(m["object_id"]),
			Evidence:   toString(m["evidence"]),
			Metadata:   ensureMap(m["metadata"]),
		}
		if entity.Type != "" || entity.Value != "" {
			out = append(out, entity)
		}
	}
	return out
}

func documentVisualDocumentContext(manifest documentVisualManifest) string {
	pageTexts := make([]string, 0, len(manifest.Pages))
	for _, page := range manifest.Pages {
		pageTexts = append(pageTexts, page.Text)
	}
	return joinNonEmpty([]string{
		manifest.Source.FileName,
		documentVisualEntityText(manifest.Entities),
		joinNonEmpty(pageTexts, "\n"),
	}, "\n")
}

func documentVisualEntityText(entities []documentVisualEntity) string {
	lines := make([]string, 0, len(entities))
	for _, entity := range entities {
		lines = append(lines, joinNonEmpty([]string{entity.Type, entity.Value, entity.Evidence}, " "))
	}
	return joinNonEmpty(lines, "\n")
}

func documentVisualPageSummary(text string) string {
	text = strings.TrimSpace(text)
	if len(text) <= 600 {
		return text
	}
	return truncateUTF8Bytes(text, 600)
}

func documentVisualPageImageID(pages []documentVisualPage, pageIndex map[int]int, pageNum int) string {
	if idx, ok := pageIndex[pageNum]; ok {
		return pages[idx].PageImageFileID
	}
	return ""
}

func pageDimensions(pages []documentVisualPage, pageIndex map[int]int, pageNum int) (float64, float64) {
	if idx, ok := pageIndex[pageNum]; ok {
		return pages[idx].Width, pages[idx].Height
	}
	return 0, 0
}

func documentVisualManifestFileName(sourceFileName, sourceFileID string) string {
	stem := strings.TrimSuffix(firstNonEmptyString(path.Base(sourceFileName), sourceFileID, "document"), path.Ext(sourceFileName))
	return fmt.Sprintf("%s.document-visual-manifest.json", stem)
}

func documentVisualPageImageName(sourceFileName string, pageNumber int) string {
	return documentVisualPageImageNameWithExt(sourceFileName, pageNumber, "jpg")
}

func documentVisualPageImageNameWithExt(sourceFileName string, pageNumber int, ext string) string {
	stem := strings.TrimSuffix(firstNonEmptyString(path.Base(sourceFileName), "document"), path.Ext(sourceFileName))
	return fmt.Sprintf("%s.page-%03d.%s", stem, pageNumber, ext)
}

func documentVisualObjectImageName(sourceFileName string, obj documentVisualObject) string {
	stem := strings.TrimSuffix(firstNonEmptyString(path.Base(sourceFileName), "document"), path.Ext(sourceFileName))
	return fmt.Sprintf("%s.page-%03d.%s.jpg", stem, obj.PageNumber, obj.ObjectID)
}

func documentVisualIndexID(parts ...string) string {
	return hashShort(append([]string{"document-visual-index"}, parts...)...)
}

func documentVisualPageObjectID(sourceFileID string, pageNumber int) string {
	return hashShort("visual-page", sourceFileID, fmt.Sprint(pageNumber))
}

func hashShort(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:24]
}

func boolPtrDefault(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func firstString(values []string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstPositiveInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func joinNonEmpty(values []string, sep string) string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return strings.Join(out, sep)
}

func interfaceToFloat64Slice(v interface{}) []float64 {
	switch x := v.(type) {
	case []float64:
		return append([]float64(nil), x...)
	case []interface{}:
		return toFloat64Slice(x)
	case []int:
		out := make([]float64, 0, len(x))
		for _, n := range x {
			out = append(out, float64(n))
		}
		return out
	default:
		return nil
	}
}

func clampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}
