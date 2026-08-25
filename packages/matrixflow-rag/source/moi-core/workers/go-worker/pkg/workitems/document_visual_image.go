package workitems

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"
	moi "github.com/matrixflow/moi-core/go-sdk"
)

type documentVisualImageIndexOptions struct {
	ManifestFileID          string
	ImageEmbeddingModel     string
	ImageEmbeddingBackendID string
	EmbeddingDimension      int
	PreprocessVersion       string
	DistanceMetric          string
	EmbeddingSource         string
	IndexVersion            int64
	Scopes                  []string
	VolumeID                int64
	AllowEmpty              bool
}

type documentVisualImageIndexTarget struct {
	Scope            string
	ObjectID         string
	ObjectKind       string
	PageNumber       int
	ImageFileID      string
	PageImageFileID  string
	BBox             []float64
	Semantics        imageSemantics
	FigureNo         string
	CaptionBlockUUID string
	SourceBlockID    string
	NearbyText       string
}

type documentVisualImageEmbeddingResponse struct {
	Object string `json:"object,omitempty"`
	Model  string `json:"model,omitempty"`
	Data   []struct {
		Object    string    `json:"object,omitempty"`
		Index     int       `json:"index,omitempty"`
		Embedding []float64 `json:"embedding,omitempty"`
	} `json:"data"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func buildDocumentVisualImageIndexDocuments(ctx context.Context, client *moi.Client, workspaceID string, manifest documentVisualManifest, opts documentVisualImageIndexOptions) ([]VectorDoc, map[string]int, error) {
	if manifest.Source.FileID == "" {
		return nil, nil, fmt.Errorf("document_visual.index.image: manifest.source.file_id is required")
	}
	scopeSet := documentVisualImageScopeSet(opts.Scopes)
	docs := make([]VectorDoc, 0, len(manifest.Pages)+len(manifest.Objects))
	counts := map[string]int{}

	if scopeSet["page"] {
		for _, page := range manifest.Pages {
			if strings.TrimSpace(page.PageImageFileID) == "" {
				if opts.AllowEmpty {
					continue
				}
				return nil, nil, fmt.Errorf("document_visual.index.image: page %d missing page_image_file_id", page.PageNumber)
			}
			bbox := page.BBox
			if len(bbox) == 0 && page.Width > 0 && page.Height > 0 {
				bbox = []float64{0, 0, page.Width, page.Height}
			}
			target := documentVisualImageIndexTarget{
				Scope:           "page",
				ObjectID:        documentVisualPageObjectID(manifest.Source.FileID, page.PageNumber),
				ObjectKind:      "page",
				PageNumber:      page.PageNumber,
				ImageFileID:     page.PageImageFileID,
				PageImageFileID: page.PageImageFileID,
				BBox:            bbox,
				Semantics:       imageSemantics{OCR: page.Text},
				NearbyText:      documentVisualPageIndexText(manifest, page),
			}
			meta := documentVisualImageIndexMeta(manifest, opts, target)
			doc, err := documentVisualImageIndexDoc(ctx, client, workspaceID, opts, page.PageImageFileID, target.NearbyText, meta)
			if err != nil {
				return nil, nil, fmt.Errorf("document_visual.index.image: page %d image %s: %w", page.PageNumber, page.PageImageFileID, err)
			}
			docs = append(docs, doc)
			counts["page"]++
		}
	}
	if scopeSet["visual_object"] {
		for _, obj := range manifest.Objects {
			if strings.TrimSpace(obj.ImageFileID) == "" {
				if opts.AllowEmpty {
					continue
				}
				return nil, nil, fmt.Errorf("document_visual.index.image: object %s missing image_file_id", obj.ObjectID)
			}
			if missing := documentVisualImageObjectMissingBacklink(manifest, obj); missing != "" {
				if opts.AllowEmpty {
					continue
				}
				return nil, nil, fmt.Errorf("document_visual.index.image: object %s missing %s", obj.ObjectID, missing)
			}
			target := documentVisualImageIndexTarget{
				Scope:           "visual_object",
				ObjectID:        obj.ObjectID,
				ObjectKind:      firstNonEmptyString(obj.ObjectKind, "drawing_view"),
				PageNumber:      obj.PageNumber,
				ImageFileID:     obj.ImageFileID,
				PageImageFileID: obj.PageImageFileID,
				BBox:            obj.BBox,
				Semantics: imageSemantics{
					OCR:              obj.OCR,
					FigureCaption:    obj.FigureCaption,
					GeneratedCaption: obj.Caption,
				},
				FigureNo:         obj.FigureNo,
				CaptionBlockUUID: obj.CaptionBlockUUID,
				SourceBlockID:    obj.SourceBlockID,
				NearbyText:       documentVisualImageObjectNearbyText(manifest, obj),
			}
			meta := documentVisualImageIndexMeta(manifest, opts, target)
			doc, err := documentVisualImageIndexDoc(ctx, client, workspaceID, opts, obj.ImageFileID, target.NearbyText, meta)
			if err != nil {
				return nil, nil, fmt.Errorf("document_visual.index.image: object %s image %s: %w", obj.ObjectID, obj.ImageFileID, err)
			}
			docs = append(docs, doc)
			counts["visual_object"]++
		}
	}
	if len(docs) == 0 {
		if opts.AllowEmpty {
			return docs, counts, nil
		}
		return nil, nil, fmt.Errorf("document_visual.index.image: no page or visual_object images selected for indexing")
	}
	return docs, counts, nil
}

func documentVisualImageObjectNearbyText(manifest documentVisualManifest, obj documentVisualObject) string {
	if manifest.Profile == documentVisualStandardRAGProfile {
		return documentVisualObjectLocalContext(obj)
	}
	if context := strings.TrimSpace(obj.Context); context != "" {
		return context
	}
	return documentVisualObjectLocalContext(obj)
}

func documentVisualImageObjectMissingBacklink(manifest documentVisualManifest, obj documentVisualObject) string {
	if manifest.Profile == documentVisualStandardRAGProfile {
		return ""
	}
	if strings.TrimSpace(obj.PageImageFileID) == "" {
		return "page_image_file_id"
	}
	if len(obj.BBox) != 4 {
		return "bbox"
	}
	return ""
}

func documentVisualImageIndexDoc(ctx context.Context, client *moi.Client, workspaceID string, opts documentVisualImageIndexOptions, imageFileID, content string, meta map[string]interface{}) (VectorDoc, error) {
	embedding, rawMeta, err := documentVisualImageEmbeddingForFile(ctx, client, workspaceID, imageFileID, opts.ImageEmbeddingModel, opts.ImageEmbeddingBackendID, opts.PreprocessVersion)
	if err != nil {
		return VectorDoc{}, err
	}
	if len(embedding) == 0 {
		return VectorDoc{}, fmt.Errorf("image embedding is empty")
	}
	if opts.EmbeddingDimension > 0 && len(embedding) != opts.EmbeddingDimension {
		return VectorDoc{}, fmt.Errorf("image embedding dimension mismatch: got %d want %d", len(embedding), opts.EmbeddingDimension)
	}
	if rawMeta != nil {
		if got := strings.TrimSpace(toString(rawMeta["preprocess_version"])); got != "" && got != opts.PreprocessVersion {
			return VectorDoc{}, fmt.Errorf("image embedding preprocess_version mismatch: got %s want %s", got, opts.PreprocessVersion)
		}
		if got := strings.TrimSpace(toString(rawMeta["distance_metric"])); got != "" && got != opts.DistanceMetric {
			return VectorDoc{}, fmt.Errorf("image embedding distance_metric mismatch: got %s want %s", got, opts.DistanceMetric)
		}
		meta["embedding_backend_metadata"] = rawMeta
	}
	id := documentVisualImageIndexID(
		toString(meta["source_file_id"]),
		fmt.Sprint(opts.IndexVersion),
		toString(meta["level"]),
		toString(meta["chunk_id"]),
		imageFileID,
		opts.ImageEmbeddingModel,
		fmt.Sprint(len(embedding)),
		opts.PreprocessVersion,
		opts.DistanceMetric,
	)
	meta["embedding_dimension"] = len(embedding)
	pageNum := toInt(meta["page_number"], 0)
	fileID := toString(meta["source_file_id"])
	volumeID := ""
	if opts.VolumeID > 0 {
		volumeID = fmt.Sprint(opts.VolumeID)
		meta["volume_id"] = volumeID
	}
	return VectorDoc{
		ID:           id,
		Content:      strings.TrimSpace(content),
		Embedding:    embedding,
		Metadata:     meta,
		FileID:       fileID,
		VolumeID:     volumeID,
		PageNumber:   &pageNum,
		Level:        "chunk",
		DocID:        "doc_" + fileID,
		SectionID:    toString(meta["object_id"]),
		IndexVersion: opts.IndexVersion,
	}, nil
}

func documentVisualImageIndexMeta(manifest documentVisualManifest, opts documentVisualImageIndexOptions, target documentVisualImageIndexTarget) map[string]interface{} {
	chunkID := documentVisualImageChunkID(target.Scope, target.ObjectID)
	meta := map[string]interface{}{
		"modality":                   "image",
		"segment_type":               "image",
		"asset_kind":                 "document_visual",
		"schema_version":             documentVisualSchemaVersion,
		"profile":                    manifest.Profile,
		"scope":                      target.Scope,
		"level":                      "chunk",
		"chunk_id":                   chunkID,
		"object_id":                  target.ObjectID,
		"object_kind":                target.ObjectKind,
		"source_file_id":             manifest.Source.FileID,
		"source_file_name":           manifest.Source.FileName,
		"file_id":                    manifest.Source.FileID,
		"manifest_file_id":           opts.ManifestFileID,
		"page_number":                target.PageNumber,
		"image_file_id":              target.ImageFileID,
		"page_image_file_id":         target.PageImageFileID,
		"bbox":                       target.BBox,
		"caption":                    target.Semantics.GeneratedCaption,
		"ocr_text":                   target.Semantics.OCR,
		"nearby_text":                target.NearbyText,
		"embedding_model":            opts.ImageEmbeddingModel,
		"image_embedding_model":      opts.ImageEmbeddingModel,
		"embedding_backend_id":       opts.ImageEmbeddingBackendID,
		"image_embedding_backend_id": opts.ImageEmbeddingBackendID,
		"embedding_dimension":        opts.EmbeddingDimension,
		"image_embedding_dimension":  opts.EmbeddingDimension,
		"embedding_source":           opts.EmbeddingSource,
		"image_embedding_source":     opts.EmbeddingSource,
		"preprocess_version":         opts.PreprocessVersion,
		"image_preprocess_version":   opts.PreprocessVersion,
		"distance_metric":            opts.DistanceMetric,
		"image_distance_metric":      opts.DistanceMetric,
		"index_version":              opts.IndexVersion,
	}
	addDocumentVisualSemanticMetadata(meta, target.Semantics, target.FigureNo, target.CaptionBlockUUID, target.SourceBlockID)
	return meta
}

func documentVisualImageEmbeddingForFile(ctx context.Context, client *moi.Client, workspaceID, fileID, model, imageEmbeddingBackendID, preprocessVersion string) ([]float64, map[string]interface{}, error) {
	download, err := client.Files().DownloadWithMeta(ctx, workspaceID, fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("download image file: %w", err)
	}
	defer download.Body.Close()
	raw, err := io.ReadAll(download.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read image file: %w", err)
	}
	mimeType := strings.TrimSpace(strings.Split(download.ContentType, ";")[0])
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = http.DetectContentType(raw)
	}
	return documentVisualImageEmbedding(ctx, client, workspaceID, model, imageEmbeddingBackendID, preprocessVersion, raw, mimeType)
}

func documentVisualImageEmbedding(ctx context.Context, client *moi.Client, workspaceID, model, imageEmbeddingBackendID, preprocessVersion string, raw []byte, mimeType string) ([]float64, map[string]interface{}, error) {
	if len(raw) == 0 {
		return nil, nil, fmt.Errorf("image bytes are empty")
	}
	mimeType = strings.TrimSpace(strings.Split(mimeType, ";")[0])
	if !strings.HasPrefix(mimeType, "image/") {
		return nil, nil, fmt.Errorf("image content type must be image/*, got %q", mimeType)
	}
	imageURL, err := agentruntimev2.LosslessImageDataURL(raw, mimeType)
	if err != nil {
		return nil, nil, fmt.Errorf("encode lossless image data URL: %w", err)
	}
	body := map[string]interface{}{
		"model": model,
		"type":  "embedding_multimodal",
		"input": []map[string]interface{}{
			{
				"content": []map[string]interface{}{
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": imageURL,
						},
					},
				},
			},
		},
		"embedding_mode":     "fusion",
		"output_cardinality": "one_per_input",
		"encoding_format":    "float",
	}
	if backendID := documentVisualNumericImageEmbeddingBackendID(imageEmbeddingBackendID); backendID != 0 {
		body["backend_id"] = backendID
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal image embedding request: %w", err)
	}
	respBody, err := client.Embeddings(workspaceID).CreateRaw(ctx, payload)
	if err != nil {
		return nil, nil, fmt.Errorf("create image embedding model=%s: %w", model, err)
	}
	var resp documentVisualImageEmbeddingResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, nil, fmt.Errorf("decode image embedding response: %w", err)
	}
	if len(resp.Data) != 1 {
		return nil, nil, fmt.Errorf("image embedding response count mismatch: got %d want 1", len(resp.Data))
	}
	embedding := append([]float64(nil), resp.Data[0].Embedding...)
	if len(embedding) == 0 {
		return nil, nil, fmt.Errorf("image embedding is empty")
	}
	return embedding, resp.Metadata, nil
}

func documentVisualNumericImageEmbeddingBackendID(raw string) int64 {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}
	backendID, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return backendID
}

func deleteDocumentVisualImageIndexRows(ctx context.Context, db *sql.DB, tableName, sourceFileID, model, preprocessVersion, distanceMetric string, indexVersion int64) (int64, error) {
	if strings.TrimSpace(sourceFileID) == "" {
		return 0, fmt.Errorf("document_visual.index.image: source_file_id is required for OVERWRITE")
	}
	if indexVersion <= 0 {
		return 0, fmt.Errorf("document_visual.index.image: index_version is required for OVERWRITE")
	}
	q := fmt.Sprintf(`DELETE FROM %s
WHERE JSON_UNQUOTE(JSON_EXTRACT(meta, '$.source_file_id')) = ?
  AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.embedding_model')) = ?
  AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.preprocess_version')) = ?
  AND JSON_UNQUOTE(JSON_EXTRACT(meta, '$.distance_metric')) = ?
  AND index_version = ?`, tableName)
	res, err := db.ExecContext(ctx, q, sourceFileID, model, preprocessVersion, distanceMetric, indexVersion)
	if err != nil {
		return 0, fmt.Errorf("document_visual.index.image: delete overwrite scope rows: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

func documentVisualImageScopeSet(scopes []string) map[string]bool {
	if len(scopes) == 0 {
		return map[string]bool{"page": true, "visual_object": true}
	}
	out := map[string]bool{}
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "page" || scope == "visual_object" {
			out[scope] = true
		}
	}
	return out
}

func documentVisualImageIndexID(parts ...string) string {
	h := sha256.New()
	h.Write([]byte("document-visual-image-index"))
	h.Write([]byte{0})
	for _, part := range parts {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func documentVisualImageChunkID(scope, objectID string) string {
	return "document_visual_image:" + scope + ":" + objectID
}
