package workitems

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/data"
	"github.com/matrixflow/moi-core/model/mowl"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestDocumentVisualTextIndexDocumentsKeepVisualBacklinks(t *testing.T) {
	manifest := sampleDocumentVisualManifest()

	docs, err := buildDocumentVisualTextIndexDocuments(manifest)
	if err != nil {
		t.Fatalf("build index documents: %v", err)
	}

	var pageDoc, objectDoc map[string]interface{}
	for _, doc := range docs {
		meta := doc["metadata"].(map[string]interface{})
		switch meta["scope"] {
		case "page":
			pageDoc = doc
		case "visual_object":
			objectDoc = doc
		}
	}
	if pageDoc == nil {
		t.Fatal("page index row not found")
	}
	if objectDoc == nil {
		t.Fatal("visual_object index row not found")
	}

	for name, doc := range map[string]map[string]interface{}{"page": pageDoc, "object": objectDoc} {
		meta := doc["metadata"].(map[string]interface{})
		for _, key := range []string{"object_id", "image_file_id", "page_image_file_id", "source_file_id", "page_number", "bbox"} {
			if meta[key] == nil || toString(meta[key]) == "" {
				t.Fatalf("%s row missing %s: %#v", name, key, meta)
			}
		}
	}
	pageContent := toString(pageDoc["content"])
	for _, want := range []string{"20C114220.pdf", "Q235", "图纸号 20C114220"} {
		if !strings.Contains(pageContent, want) {
			t.Fatalf("page content missing %q: %s", want, pageContent)
		}
	}
	if strings.Contains(pageContent, "主视图") {
		t.Fatalf("page content leaked object-only context: %s", pageContent)
	}
	objectContent := toString(objectDoc["content"])
	for _, want := range []string{"Q235", "主视图"} {
		if !strings.Contains(objectContent, want) {
			t.Fatalf("object content missing %q: %s", want, objectContent)
		}
	}
	if strings.Contains(objectContent, "20C114220.pdf") {
		t.Fatalf("object content leaked document-only context: %s", objectContent)
	}
}

func TestDocumentVisualTextIndexDocumentsCanExpandMultilevel(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	docs, err := buildDocumentVisualTextIndexDocuments(manifest)
	if err != nil {
		t.Fatalf("build text index documents: %v", err)
	}

	expanded, err := expandDocumentVisualTextIndexDocuments(context.Background(), nil, docs, 2)
	if err != nil {
		t.Fatalf("expand multilevel documents: %v", err)
	}

	var levels []string
	for _, raw := range expanded {
		meta := ensureMap(raw["metadata"])
		levels = append(levels, toString(meta["level"]))
		if toString(meta["file_id"]) != manifest.Source.FileID {
			t.Fatalf("expanded doc file_id=%v, want %s", meta["file_id"], manifest.Source.FileID)
		}
		if toString(meta["file_name"]) != manifest.Source.FileName {
			t.Fatalf("expanded doc file_name=%v, want %s", meta["file_name"], manifest.Source.FileName)
		}
	}

	for _, want := range []string{"doc", "section", "chunk"} {
		if !stringSliceContains(levels, want) {
			t.Fatalf("expanded levels %v missing %q", levels, want)
		}
	}
}

func TestDocumentVisualTextIndexSetsVersionOnPlainDocuments(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	docs, err := buildDocumentVisualTextIndexDocuments(manifest)
	if err != nil {
		t.Fatalf("build text index documents: %v", err)
	}
	if got := documentsIndexVersion(docs); got != 0 {
		t.Fatalf("initial documents index_version=%d, want 0 before assignment", got)
	}

	setDocumentVisualIndexVersion(docs, 1782999000000)

	if got := documentsIndexVersion(docs); got != 1782999000000 {
		t.Fatalf("documents index_version=%d, want assigned version", got)
	}
	for _, doc := range docs {
		meta := ensureMap(doc["metadata"])
		if got := toInt64(meta["index_version"], 0); got != 1782999000000 {
			t.Fatalf("doc metadata index_version=%d", got)
		}
	}
}

func TestDocumentVisualParseDisabledSkipsManifestBuild(t *testing.T) {
	enabled := false
	payload, err := json.Marshal(documentVisualParseInput{Enabled: &enabled, Profile: "standard_rag_v1"})
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	msg := &mowl.MowlMessage{Data: string(payload)}

	if _, err := (&DocumentVisualParse{}).Handle(context.Background(), &mockWctx{}, msg); err != nil {
		t.Fatalf("disabled parse should not fail: %v", err)
	}

	var out documentVisualParseOutput
	if err := json.Unmarshal([]byte(msg.Data), &out); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if out.Status != "disabled" {
		t.Fatalf("status=%q, want disabled", out.Status)
	}
	if !out.Validation.Valid {
		t.Fatalf("validation should be valid for disabled output: %#v", out.Validation)
	}
	if len(out.Manifests) != 0 || len(out.Documents) != 0 {
		t.Fatalf("disabled output should not build manifests/documents: manifests=%d documents=%d", len(out.Manifests), len(out.Documents))
	}
}

func TestDocumentVisualManifestUsesRenderedPageSizeWhenLayoutMissing(t *testing.T) {
	manifest, err := buildDocumentVisualManifest(
		context.Background(),
		nil,
		"workspace-1",
		[]*data.Document{
			testDocument("图纸号 C114220，材料 Q235", map[string]interface{}{"page_number": 1}),
		},
		documentVisualLayout{},
		map[int]documentVisualPageAsset{
			1: {FileID: "page-img-1", Width: 640, Height: 480},
		},
		documentVisualSource{FileID: "source-pdf-1", FileName: "20C114220.pdf"},
		documentVisualDefaultProfile,
		documentVisualBuildOptions{},
	)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if len(manifest.Pages) != 1 {
		t.Fatalf("pages=%d, want 1", len(manifest.Pages))
	}

	page := manifest.Pages[0]
	if page.PageImageFileID != "page-img-1" {
		t.Fatalf("page image file id=%q, want page-img-1", page.PageImageFileID)
	}
	if page.Width != 640 || page.Height != 480 {
		t.Fatalf("page size=%vx%v, want 640x480", page.Width, page.Height)
	}
	if len(page.BBox) != 4 || page.BBox[0] != 0 || page.BBox[1] != 0 || page.BBox[2] != 640 || page.BBox[3] != 480 {
		t.Fatalf("page bbox=%#v, want rendered page bbox", page.BBox)
	}

	indexDocs, err := buildDocumentVisualTextIndexDocuments(manifest)
	if err != nil {
		t.Fatalf("build text index docs: %v", err)
	}
	var pageMeta map[string]interface{}
	for _, doc := range indexDocs {
		meta := doc["metadata"].(map[string]interface{})
		if meta["scope"] == "page" {
			pageMeta = meta
			break
		}
	}
	if pageMeta == nil {
		t.Fatal("page text index row not found")
	}
	if got := pageMeta["page_image_file_id"]; got != "page-img-1" {
		t.Fatalf("page_image_file_id=%#v, want page-img-1", got)
	}
	bbox, ok := pageMeta["bbox"].([]float64)
	if !ok || len(bbox) != 4 || bbox[2] != 640 || bbox[3] != 480 {
		t.Fatalf("page index bbox=%#v, want rendered page bbox", pageMeta["bbox"])
	}
}

func TestDocumentVisualImageIndexMetaBindsEmbeddingIdentity(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	opts := documentVisualImageIndexOptions{
		ImageEmbeddingModel:     "efficientnet-b3",
		ImageEmbeddingBackendID: "local-efficientnet-b3",
		EmbeddingDimension:      1536,
		PreprocessVersion:       "efficientnet-b3-v1-rgb-300-letterbox-imagenet",
		DistanceMetric:          "cosine",
		EmbeddingSource:         "real",
		IndexVersion:            1782873668656,
	}

	meta := documentVisualImageIndexMeta(
		manifest,
		opts,
		documentVisualImageIndexTarget{
			Scope:           "visual_object",
			ObjectID:        "obj-1",
			ObjectKind:      "drawing_view",
			PageNumber:      1,
			ImageFileID:     "obj-img-1",
			PageImageFileID: "page-img-1",
			BBox:            []float64{10, 20, 300, 400},
			Semantics: imageSemantics{
				GeneratedCaption: "工业零件主视图",
				OCR:              "尺寸 10x20",
			},
			NearbyText: "图纸号 20C114220 材质 Q235",
		},
	)

	for key, want := range map[string]interface{}{
		"scope":                      "visual_object",
		"object_id":                  "obj-1",
		"image_file_id":              "obj-img-1",
		"page_image_file_id":         "page-img-1",
		"source_file_id":             "file-1",
		"image_embedding_model":      "efficientnet-b3",
		"image_embedding_dimension":  1536,
		"image_embedding_backend_id": "local-efficientnet-b3",
		"image_preprocess_version":   "efficientnet-b3-v1-rgb-300-letterbox-imagenet",
		"image_distance_metric":      "cosine",
		"image_embedding_source":     "real",
		"embedding_model":            "efficientnet-b3",
		"preprocess_version":         "efficientnet-b3-v1-rgb-300-letterbox-imagenet",
		"distance_metric":            "cosine",
		"embedding_source":           "real",
		"schema_version":             documentVisualSchemaVersion,
		"asset_kind":                 "document_visual",
		"modality":                   "image",
		"segment_type":               "image",
		"index_version":              int64(1782873668656),
		"chunk_id":                   "document_visual_image:visual_object:obj-1",
		"ocr_text":                   "尺寸 10x20",
	} {
		if got := meta[key]; got != want {
			t.Fatalf("meta[%s]=%#v, want %#v", key, got, want)
		}
	}
	if got := meta["bbox"].([]float64); len(got) != 4 || got[0] != 10 || got[3] != 400 {
		t.Fatalf("bbox=%#v, want image crop bbox", got)
	}
	if _, ok := meta["chunk_index"]; ok {
		t.Fatalf("chunk_index should not be set when image chunk uses chunk_id: %#v", meta["chunk_index"])
	}
}

func TestDocumentVisualImageIndexAllowEmptySkipsMissingImages(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Pages[0].PageImageFileID = ""
	manifest.Objects[0].ImageFileID = ""
	manifest.Objects[0].PageImageFileID = ""

	docs, counts, err := buildDocumentVisualImageIndexDocuments(context.Background(), nil, "ws-1", manifest, documentVisualImageIndexOptions{
		ImageEmbeddingModel: "clip-vit-large",
		EmbeddingDimension:  768,
		PreprocessVersion:   "clip-v2-rgb-224",
		DistanceMetric:      "cosine",
		EmbeddingSource:     "real",
		AllowEmpty:          true,
	})
	if err != nil {
		t.Fatalf("build image index documents: %v", err)
	}
	if len(docs) != 0 {
		t.Fatalf("docs = %#v, want empty", docs)
	}
	if counts["page"] != 0 || counts["visual_object"] != 0 {
		t.Fatalf("counts = %#v, want no indexed images", counts)
	}
}

func TestDocumentVisualManifestValidationAllowsStandardRAGObjectWithoutBBox(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects[0].BBox = nil

	validation := validateDocumentVisualManifest(manifest, documentVisualBuildOptions{
		RequirePageImages:    false,
		RequireObjectImages:  false,
		RequireVisualContext: false,
	})
	if !validation.Valid {
		t.Fatalf("standard RAG visual manifest validation = %#v, want valid without object bbox", validation.Errors)
	}

	validation = validateDocumentVisualManifest(manifest, documentVisualBuildOptions{
		RequirePageImages:    true,
		RequireObjectImages:  true,
		RequireVisualContext: true,
	})
	if validation.Valid || !strings.Contains(strings.Join(validation.Errors, "; "), "missing bbox") {
		t.Fatalf("strict visual manifest validation = %#v, want missing bbox error", validation.Errors)
	}
}

func TestDocumentVisualStandardRAGObjectContextUsesOnlyObjectLocalText(t *testing.T) {
	docs := []*data.Document{
		testDocument("whole page text should stay on page", map[string]interface{}{
			"source_file_id": "source-file-1",
			"file_name":      "source.pdf",
			"page_number":    1,
			"block_type":     "text",
		}),
		testDocument("object local text", map[string]interface{}{
			"source_file_id": "source-file-1",
			"file_name":      "source.pdf",
			"page_number":    1,
			"block_type":     "image",
			"block_uuid":     "obj-1",
			"image_file_id":  "obj-img-1",
			"caption":        "object caption",
			"ocr":            "object ocr",
			"bbox":           floatSliceMetadataValue([]float64{1, 2, 3, 4}),
		}),
	}
	manifest, err := buildDocumentVisualManifest(
		context.Background(),
		nil,
		"ws-1",
		docs,
		documentVisualLayout{},
		nil,
		documentVisualSource{FileID: "source-file-1", FileName: "source.pdf", PageCount: 1},
		documentVisualStandardRAGProfile,
		documentVisualBuildOptions{
			RequirePageImages:    false,
			RequireObjectImages:  false,
			RequireVisualContext: false,
		},
	)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if len(manifest.Objects) != 1 {
		t.Fatalf("objects=%d, want 1", len(manifest.Objects))
	}
	got := manifest.Objects[0].Context
	if strings.Contains(got, "whole page text") {
		t.Fatalf("object context should not include page text: %q", got)
	}
	for _, want := range []string{"object caption", "object ocr", "object local text"} {
		if !strings.Contains(got, want) {
			t.Fatalf("object context=%q, want %q", got, want)
		}
	}
}

func TestDocumentVisualImageIndexObjectBacklinksAreProfileScoped(t *testing.T) {
	obj := sampleDocumentVisualManifest().Objects[0]
	obj.PageImageFileID = ""
	obj.BBox = nil

	standard := sampleDocumentVisualManifest()
	standard.Profile = documentVisualStandardRAGProfile
	if missing := documentVisualImageObjectMissingBacklink(standard, obj); missing != "" {
		t.Fatalf("standard RAG object missing backlink = %q, want no hard backlink requirement", missing)
	}

	drawing := sampleDocumentVisualManifest()
	drawing.Profile = documentVisualDefaultProfile
	if missing := documentVisualImageObjectMissingBacklink(drawing, obj); missing != "page_image_file_id" {
		t.Fatalf("drawing object missing backlink = %q, want page_image_file_id", missing)
	}

	obj.PageImageFileID = "page-img-1"
	if missing := documentVisualImageObjectMissingBacklink(drawing, obj); missing != "bbox" {
		t.Fatalf("drawing object missing backlink = %q, want bbox", missing)
	}
}

func TestDocumentVisualImageIndexObjectContentUsesLocalTextAndObjectImage(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Profile = documentVisualStandardRAGProfile
	manifest.Objects[0].Context = "polluted context with 图纸号 20C114220 材质 Q235"

	var downloadedPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/obj-img-1/download":
			downloadedPaths = append(downloadedPaths, r.URL.Path)
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(minimalPNGBytes())
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/embeddings":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{{
					"embedding": []float64{0.1, 0.2, 0.3, 0.4},
				}},
				"metadata": map[string]interface{}{
					"preprocess_version": "efficientnet-b3-v1",
					"distance_metric":    "cosine",
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("create SDK client: %v", err)
	}
	defer client.Close()

	docs, counts, err := buildDocumentVisualImageIndexDocuments(context.Background(), client, "ws-1", manifest, documentVisualImageIndexOptions{
		ImageEmbeddingModel:     "efficientnet-b3",
		ImageEmbeddingBackendID: "1",
		EmbeddingDimension:      4,
		PreprocessVersion:       "efficientnet-b3-v1",
		DistanceMetric:          "cosine",
		EmbeddingSource:         "real",
		IndexVersion:            1783060753649,
		Scopes:                  []string{"visual_object"},
	})
	if err != nil {
		t.Fatalf("build image index documents: %v", err)
	}
	if counts["visual_object"] != 1 || len(docs) != 1 {
		t.Fatalf("counts=%#v docs=%d, want one visual object", counts, len(docs))
	}
	if len(downloadedPaths) != 1 || downloadedPaths[0] != "/api/v1/workspaces/ws-1/files/obj-img-1/download" {
		t.Fatalf("downloaded paths = %#v, want object image download", downloadedPaths)
	}
	doc := docs[0]
	for _, polluted := range []string{"图纸号 20C114220", "材质 Q235", "polluted context"} {
		if strings.Contains(doc.Content, polluted) {
			t.Fatalf("image index content should not include %q: %q", polluted, doc.Content)
		}
		if nearby := toString(doc.Metadata["nearby_text"]); strings.Contains(nearby, polluted) {
			t.Fatalf("nearby_text should not include %q: %q", polluted, nearby)
		}
	}
	for _, want := range []string{"工业零件主视图", "尺寸 10x20", "主视图"} {
		if !strings.Contains(doc.Content, want) {
			t.Fatalf("image index content=%q, want %q", doc.Content, want)
		}
	}
	if doc.Metadata["image_file_id"] != "obj-img-1" || doc.Metadata["page_image_file_id"] != "page-img-1" || doc.Metadata["ocr_text"] != "尺寸 10x20" || doc.Metadata["caption"] != "工业零件主视图" {
		t.Fatalf("image metadata = %#v, want visual backlinks and local fields", doc.Metadata)
	}
}

func TestDocumentVisualImageIndexStrictProfileSkipsObjectMissingBacklinks(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Pages = nil
	manifest.Objects[0].PageImageFileID = ""
	manifest.Objects[0].BBox = nil

	docs, counts, err := buildDocumentVisualImageIndexDocuments(context.Background(), nil, "ws-1", manifest, documentVisualImageIndexOptions{
		ImageEmbeddingModel: "clip-vit-large",
		EmbeddingDimension:  768,
		PreprocessVersion:   "clip-v2-rgb-224",
		DistanceMetric:      "cosine",
		EmbeddingSource:     "real",
		Scopes:              []string{"visual_object"},
		AllowEmpty:          true,
	})
	if err != nil {
		t.Fatalf("build image index documents: %v", err)
	}
	if len(docs) != 0 || counts["visual_object"] != 0 {
		t.Fatalf("docs=%d counts=%#v, want strict profile object without backlinks skipped", len(docs), counts)
	}
}

func TestDocumentVisualImageIndexStrictProfileRejectsObjectMissingBacklinks(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Pages = nil
	manifest.Objects[0].PageImageFileID = ""
	manifest.Objects[0].BBox = nil

	_, _, err := buildDocumentVisualImageIndexDocuments(context.Background(), nil, "ws-1", manifest, documentVisualImageIndexOptions{
		ImageEmbeddingModel: "clip-vit-large",
		EmbeddingDimension:  768,
		PreprocessVersion:   "clip-v2-rgb-224",
		DistanceMetric:      "cosine",
		EmbeddingSource:     "real",
		Scopes:              []string{"visual_object"},
	})
	if err == nil || !strings.Contains(err.Error(), "missing page_image_file_id") {
		t.Fatalf("build image index documents error = %v, want missing page image backlink", err)
	}
}

func TestDocumentVisualImageIndexDisabledSkipsConfigValidation(t *testing.T) {
	enabled := false
	payload, err := json.Marshal(documentVisualIndexImageInput{Enabled: &enabled})
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	msg := &mowl.MowlMessage{Data: string(payload)}
	if _, err := (&DocumentVisualIndexImage{}).Handle(context.Background(), &mockWctx{}, msg); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	var out documentVisualIndexImageOutput
	if err := json.Unmarshal([]byte(msg.Data), &out); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if out.Status != "disabled" || out.Written != 0 || len(out.SourceFileIDs) != 0 || len(out.AllSourceFileIDs) != 0 || len(out.IndexedSourceFileIDs) != 0 {
		t.Fatalf("output = %#v, want disabled empty image index output", out)
	}
	for _, field := range []string{`"all_source_file_ids":[]`, `"indexed_source_file_ids":[]`, `"source_file_ids":[]`} {
		if !strings.Contains(msg.Data, field) {
			t.Fatalf("disabled output should include %s: %s", field, msg.Data)
		}
	}
	if !strings.Contains(msg.Data, `"embedding_backend_id":""`) {
		t.Fatalf("disabled output should include empty embedding_backend_id: %s", msg.Data)
	}
}

func TestDocumentVisualTextIndexDisabledSkipsManifestRequirement(t *testing.T) {
	enabled := false
	payload, err := json.Marshal(documentVisualIndexTextInput{
		Enabled:            &enabled,
		EmbeddingModel:     "bge-m3",
		EmbeddingDimension: 1024,
		TextVectorTable:    "visual_text_index_disabled",
	})
	if err != nil {
		t.Fatalf("marshal input: %v", err)
	}
	msg := &mowl.MowlMessage{Data: string(payload)}
	if _, err := (&DocumentVisualIndexText{}).Handle(context.Background(), &mockWctx{}, msg); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	var out documentVisualIndexTextOutput
	if err := json.Unmarshal([]byte(msg.Data), &out); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if out.Status != "disabled" || out.Written != 0 || out.DocumentsCount != 0 {
		t.Fatalf("output = %#v, want disabled empty text index output", out)
	}
	if out.TextVectorTable != "" || out.EmbeddingModel != "" {
		t.Fatalf("disabled output should clear table/model fields: %#v", out)
	}
	if !strings.Contains(msg.Data, `"status":"disabled"`) {
		t.Fatalf("disabled output should include status=disabled: %s", msg.Data)
	}
}

func TestDocumentVisualImageEmbeddingForwardsNumericBackendID(t *testing.T) {
	var embeddingRequest map[string]interface{}
	sourceImage := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			sourceImage.SetRGBA(x, y, color.RGBA{24, 96, 180, 255})
		}
	}
	var sourcePNG bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.NoCompression}).Encode(&sourcePNG, sourceImage); err != nil {
		t.Fatalf("encode source png: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/workspaces/ws-1/files/image-file/download":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(sourcePNG.Bytes())
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-1/embeddings":
			if err := json.NewDecoder(r.Body).Decode(&embeddingRequest); err != nil {
				t.Fatalf("decode embedding request: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"object": "list",
				"model":  "efficientnet-b3",
				"data": []map[string]interface{}{{
					"object":    "embedding",
					"index":     0,
					"embedding": []float64{0.1, 0.2, 0.3, 0.4},
				}},
				"metadata": map[string]interface{}{
					"preprocess_version": "efficientnet-b3-v1",
					"distance_metric":    "cosine",
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("create SDK client: %v", err)
	}
	defer client.Close()

	embedding, rawMeta, err := documentVisualImageEmbeddingForFile(context.Background(), client, "ws-1", "image-file", "efficientnet-b3", "-30010", "efficientnet-b3-v1")
	if err != nil {
		t.Fatalf("documentVisualImageEmbeddingForFile: %v", err)
	}
	if len(embedding) != 4 {
		t.Fatalf("embedding len = %d, want 4", len(embedding))
	}
	if rawMeta["preprocess_version"] != "efficientnet-b3-v1" {
		t.Fatalf("raw metadata = %+v", rawMeta)
	}
	if embeddingRequest["model"] != "efficientnet-b3" || embeddingRequest["type"] != "embedding_multimodal" || embeddingRequest["backend_id"] != float64(-30010) {
		t.Fatalf("embedding request = %+v", embeddingRequest)
	}
	if embeddingRequest["embedding_mode"] != "fusion" || embeddingRequest["output_cardinality"] != "one_per_input" || embeddingRequest["encoding_format"] != "float" {
		t.Fatalf("embedding request options = %+v", embeddingRequest)
	}
	if _, ok := embeddingRequest["images"]; ok {
		t.Fatalf("embedding request should not contain legacy images: %+v", embeddingRequest)
	}
	inputs, ok := embeddingRequest["input"].([]interface{})
	if !ok || len(inputs) != 1 {
		t.Fatalf("embedding request input = %+v", embeddingRequest["input"])
	}
	input := inputs[0].(map[string]interface{})
	content := input["content"].([]interface{})
	imageContent := content[0].(map[string]interface{})
	imageURL := imageContent["image_url"].(map[string]interface{})["url"].(string)
	if imageContent["type"] != "image_url" || !strings.HasPrefix(imageURL, "data:image/png;base64,") {
		t.Fatalf("embedding request image content = %+v", imageContent)
	}
	encoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(imageURL, "data:image/png;base64,"))
	if err != nil {
		t.Fatalf("decode embedding image data URL: %v", err)
	}
	if len(encoded) >= sourcePNG.Len() {
		t.Fatalf("embedding image payload should use smaller lossless PNG: got %d bytes from %d", len(encoded), sourcePNG.Len())
	}
	decoded, err := png.Decode(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("decode embedding image payload: %v", err)
	}
	if decoded.Bounds() != sourceImage.Bounds() || decoded.At(128, 128) != sourceImage.At(128, 128) {
		t.Fatalf("embedding image payload changed image: bounds=%v pixel=%v", decoded.Bounds(), decoded.At(128, 128))
	}
}

func TestDocumentVisualImageEmbeddingForwardsSVGBytes(t *testing.T) {
	source := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"><rect width="16" height="16" fill="blue"/></svg>`)
	var imageURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/workspaces/ws-1/embeddings" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		var request struct {
			Input []struct {
				Content []struct {
					ImageURL struct {
						URL string `json:"url"`
					} `json:"image_url"`
				} `json:"content"`
			} `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode embedding request: %v", err)
		}
		if len(request.Input) != 1 || len(request.Input[0].Content) != 1 {
			t.Fatalf("embedding request = %+v", request)
		}
		imageURL = request.Input[0].Content[0].ImageURL.URL
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{{"embedding": []float64{0.1}}},
		})
	}))
	defer server.Close()

	client, err := moi.New(server.URL, "caller-key")
	if err != nil {
		t.Fatalf("create SDK client: %v", err)
	}
	defer client.Close()

	if _, _, err := documentVisualImageEmbedding(context.Background(), client, "ws-1", "efficientnet-b3", "", "efficientnet-b3-v1", source, "image/svg+xml"); err != nil {
		t.Fatalf("documentVisualImageEmbedding: %v", err)
	}
	wantURL := "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString(source)
	if imageURL != wantURL {
		t.Fatalf("embedding image URL = %q, want %q", imageURL, wantURL)
	}
}

func TestDocumentVisualImageEmbeddingRejectsNonImageContentType(t *testing.T) {
	_, _, err := documentVisualImageEmbedding(context.Background(), nil, "ws-1", "efficientnet-b3", "", "efficientnet-b3-v1", []byte("not an image"), "text/plain")
	if err == nil || !strings.Contains(err.Error(), "image content type must be image/*") {
		t.Fatalf("documentVisualImageEmbedding() error = %v, want image content type error", err)
	}
}

func TestDocumentVisualImageEmbeddingRejectsUndecodableImageData(t *testing.T) {
	_, _, err := documentVisualImageEmbedding(context.Background(), nil, "ws-1", "efficientnet-b3", "", "efficientnet-b3-v1", []byte("not an image"), "image/png")
	if err == nil || !strings.Contains(err.Error(), "decode image config") {
		t.Fatalf("documentVisualImageEmbedding() error = %v, want image decode error", err)
	}
}

func TestDocumentVisualImageIndexRejectsMissingImagesByDefault(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Pages[0].PageImageFileID = ""

	_, _, err := buildDocumentVisualImageIndexDocuments(context.Background(), nil, "ws-1", manifest, documentVisualImageIndexOptions{
		ImageEmbeddingModel: "clip-vit-large",
		EmbeddingDimension:  768,
		PreprocessVersion:   "clip-v2-rgb-224",
		DistanceMetric:      "cosine",
		EmbeddingSource:     "real",
	})
	if err == nil || !strings.Contains(err.Error(), "missing page_image_file_id") {
		t.Fatalf("build image index documents error = %v, want missing page image error", err)
	}
}

func TestDocumentVisualImageChunkIDIsStable(t *testing.T) {
	got := documentVisualImageChunkID("visual_object", "obj-1")
	if got != "document_visual_image:visual_object:obj-1" {
		t.Fatalf("chunk_id=%q, want stable document visual image identity", got)
	}
}

func TestDocumentVisualImageScopeSetDefaultsToPageAndObject(t *testing.T) {
	got := documentVisualImageScopeSet(nil)

	if !got["page"] || !got["visual_object"] || len(got) != 2 {
		t.Fatalf("default image scopes=%#v, want page and visual_object", got)
	}

	got = documentVisualImageScopeSet([]string{"entity", "visual_object"})
	if got["page"] || !got["visual_object"] || len(got) != 1 {
		t.Fatalf("filtered image scopes=%#v, want only visual_object", got)
	}
}

func TestDocumentVisualParseDocumentGroupsBySourceFileID(t *testing.T) {
	docs := []*data.Document{
		testDocument("file one page", map[string]interface{}{
			"raw_file_id": "file-1",
			"file_name":   "one.pdf",
		}),
		testDocument("file two page", map[string]interface{}{
			"source_file_id": "file-2",
			"file_name":      "two.pdf",
		}),
		testDocument("file one page two", map[string]interface{}{
			"raw_file_id": "file-1",
			"file_name":   "one.pdf",
		}),
	}

	groups := documentVisualParseDocumentGroups(documentVisualParseInput{}, docs)

	if len(groups) != 2 {
		t.Fatalf("groups = %#v, want two source groups", groups)
	}
	if groups[0].FileID != "file-1" || len(groups[0].Documents) != 2 {
		t.Fatalf("group[0] = %#v, want file-1 with two docs", groups[0])
	}
	if groups[1].FileID != "file-2" || len(groups[1].Documents) != 1 {
		t.Fatalf("group[1] = %#v, want file-2 with one doc", groups[1])
	}
}

func TestDocumentVisualDerivedFileIDsBySourceKeepsSourcesSeparated(t *testing.T) {
	manifests := []documentVisualManifest{
		{
			Source: documentVisualSource{FileID: "source-a"},
			Pages: []documentVisualPage{
				{PageImageFileID: "page-a"},
				{PageImageFileID: "page-a"},
			},
			Objects: []documentVisualObject{
				{ImageFileID: "object-a", PageImageFileID: "page-a"},
				{ImageFileID: "source-a"},
			},
		},
		{
			Source:  documentVisualSource{FileID: "source-b"},
			Pages:   []documentVisualPage{{PageImageFileID: "page-b"}},
			Objects: []documentVisualObject{{ImageFileID: "object-b"}},
		},
	}

	got := documentVisualDerivedFileIDsBySource(manifests)
	want := map[string][]string{
		"source-a": {"page-a", "object-a"},
		"source-b": {"page-b", "object-b"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("derived file IDs by source = %#v, want %#v", got, want)
	}
}

func TestDocumentVisualSourcePrefersRawSourceFileIDOverBlockImageFileID(t *testing.T) {
	doc := testDocument("caption", map[string]interface{}{
		"file_id":     "object-image-file",
		"raw_file_id": "source-pdf-file",
		"file_name":   "20C114220.pdf",
	})

	source := inferDocumentVisualSource(documentVisualParseInput{}, []*data.Document{doc})

	if source.FileID != "source-pdf-file" {
		t.Fatalf("source file id=%q, want raw source file", source.FileID)
	}
}

func TestDocumentVisualCropRectRotatesWhenRenderedPageOrientationDiffers(t *testing.T) {
	x0, y0, x1, y1 := documentVisualCropRect([]float64{100, 200, 300, 400}, 1000, 500, 500, 1000)

	if x0 != 200 || y0 != 100 || x1 != 400 || y1 != 300 {
		t.Fatalf("crop rect=(%d,%d,%d,%d), want rotated mapping (200,100,400,300)", x0, y0, x1, y1)
	}
}

func TestDocumentVisualEngineeringOptionsRequireExplicitValidConfig(t *testing.T) {
	_, err := documentVisualEngineeringOptionsFromInput("generic_document_v1", map[string]interface{}{
		"enable_engineering_page_plan":      true,
		"enable_engineering_region_extract": true,
		"engineering_vlm_model":             "gpt-5.5",
	})
	if err == nil || !strings.Contains(err.Error(), "profile=industrial_drawing_v1") {
		t.Fatalf("profile mismatch error=%v", err)
	}

	_, err = documentVisualEngineeringOptionsFromInput(documentVisualDefaultProfile, map[string]interface{}{
		"enable_engineering_page_plan": true,
		"engineering_vlm_model":        "gpt-5.5",
	})
	if err == nil || !strings.Contains(err.Error(), "must be enabled together") {
		t.Fatalf("partial switch error=%v", err)
	}

	_, err = documentVisualEngineeringOptionsFromInput(documentVisualDefaultProfile, map[string]interface{}{
		"enable_engineering_page_plan":      true,
		"enable_engineering_region_extract": true,
	})
	if err == nil || !strings.Contains(err.Error(), "engineering_vlm_model is required") {
		t.Fatalf("missing model error=%v", err)
	}

	opts, err := documentVisualEngineeringOptionsFromInput(documentVisualDefaultProfile, map[string]interface{}{
		"enable_engineering_page_plan":      true,
		"enable_engineering_region_extract": true,
		"engineering_vlm_model":             "gpt-5.5",
		"engineering_vlm_reasoning_effort":  "custom-effort",
	})
	if err != nil {
		t.Fatalf("valid engineering options: %v", err)
	}
	if !opts.Enabled() || opts.Model != "gpt-5.5" || opts.ReasoningEffort != "custom-effort" {
		t.Fatalf("unexpected options=%#v", opts)
	}
}

func TestDocumentVisualImageBBoxToPageBBoxCoversOrientationAndValidation(t *testing.T) {
	got, err := documentVisualImageBBoxToPageBBox([]float64{100, 200, 300, 400}, 1000, 500, 1000, 500)
	if err != nil {
		t.Fatalf("convert bbox: %v", err)
	}
	want := []float64{100, 200, 300, 400}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("bbox=%v, want %v", got, want)
		}
	}

	got, err = documentVisualImageBBoxToPageBBox([]float64{200, 100, 400, 300}, 1000, 500, 500, 1000)
	if err != nil {
		t.Fatalf("convert rotated bbox: %v", err)
	}
	want = []float64{100, 200, 300, 400}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("rotated bbox=%v, want %v", got, want)
		}
	}

	if _, err := documentVisualImageBBoxToPageBBox([]float64{-1, 0, 10, 10}, 1000, 500, 1000, 500); err == nil {
		t.Fatalf("negative image bbox should fail")
	}
	if _, err := documentVisualImageBBoxToPageBBox([]float64{10, 10, 10, 20}, 1000, 500, 1000, 500); err == nil {
		t.Fatalf("empty image bbox should fail")
	}
}

func TestDocumentVisualEngineeringPlanBBoxConvertsNormalizedCoordinates(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects = nil
	manifest.Pages[0].ObjectIDs = nil
	page := engineeringImageCoordinatePage(manifest.Pages[0], documentVisualPageAsset{Width: 2000, Height: 1000})
	asset := documentVisualPageAsset{Width: 2000, Height: 1000}
	plan := engineeringDrawingPagePlan{
		Regions: []engineeringDrawingRegion{
			{
				RegionID: "revision_history_table",
				Kind:     "table",
				BBox:     []float64{100, 200, 600, 700},
			},
		},
	}

	candidates := engineeringRegionCandidates(manifest, page, asset, plan)
	if len(candidates) != 1 {
		t.Fatalf("candidates=%d, want 1", len(candidates))
	}
	wantBBox := []float64{200, 200, 1200, 700}
	for i, want := range wantBBox {
		if candidates[0].BBox[i] != want {
			t.Fatalf("candidate bbox=%v, want %v", candidates[0].BBox, wantBBox)
		}
	}
	if candidates[0].PlanBBox[2] != 600 {
		t.Fatalf("plan bbox=%v, want normalized original bbox", candidates[0].PlanBBox)
	}

	entities := engineeringEntitiesFromExtract("file-1", documentVisualObject{
		ObjectID:           candidates[0].ObjectID,
		PageNumber:         candidates[0].PageNumber,
		BBox:               candidates[0].BBox,
		OCRBBox:            candidates[0].BBox,
		PageImageFileID:    page.PageImageFileID,
		ImageFileID:        "region-img-1",
		ExtractionWarnings: []string{"bbox_clipped_to_page_image"},
	}, candidates[0], engineeringDrawingRegionExtract{
		RegionID: "revision_history_table",
		Items: []engineeringDrawingExtractItem{
			{Text: "27.57±0.15", Kind: "dimension"},
		},
	})
	if len(entities) != 1 {
		t.Fatalf("entities=%d, want 1", len(entities))
	}
	metadata := entities[0].Metadata
	if got := interfaceToFloat64Slice(metadata["vlm_plan_bbox"]); got[2] != 600 {
		t.Fatalf("vlm_plan_bbox=%v, want normalized original bbox", got)
	}
	if metadata["vlm_plan_bbox_coordinate"] != engineeringPagePlanBBoxCoordinate {
		t.Fatalf("vlm_plan_bbox_coordinate=%#v", metadata["vlm_plan_bbox_coordinate"])
	}
	if metadata["item_bbox_coordinate"] != "region_crop" {
		t.Fatalf("item_bbox_coordinate=%#v, want region_crop", metadata["item_bbox_coordinate"])
	}

	unclippedCandidates := engineeringRegionCandidates(manifest, page, asset, engineeringDrawingPagePlan{
		Regions: []engineeringDrawingRegion{
			{
				RegionID: "in_bounds_view",
				Kind:     "part",
				BBox:     []float64{50, 40, 150, 275},
			},
		},
	})
	unclippedEntities := engineeringEntitiesFromExtract("file-1", documentVisualObject{
		ObjectID:        unclippedCandidates[0].ObjectID,
		PageNumber:      unclippedCandidates[0].PageNumber,
		BBox:            unclippedCandidates[0].BBox,
		OCRBBox:         unclippedCandidates[0].BBox,
		PageImageFileID: page.PageImageFileID,
		ImageFileID:     "region-img-2",
	}, unclippedCandidates[0], engineeringDrawingRegionExtract{
		RegionID: "in_bounds_view",
		Items: []engineeringDrawingExtractItem{
			{Text: "(307.55)", Kind: "dimension"},
		},
	})
	unclippedMetadata := unclippedEntities[0].Metadata
	if _, ok := unclippedMetadata["vlm_plan_bbox_clipped"]; ok {
		t.Fatalf("vlm_plan_bbox_clipped should be absent for normalized page plan bbox: %#v", unclippedMetadata)
	}
}

func TestEngineeringRegionCandidatesReportsSkippedRegions(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	page := engineeringImageCoordinatePage(manifest.Pages[0], documentVisualPageAsset{Width: 2000, Height: 1000})
	asset := documentVisualPageAsset{Width: 2000, Height: 1000}
	plan := engineeringDrawingPagePlan{
		Regions: []engineeringDrawingRegion{
			{RegionID: "bad-kind", Kind: "legend", BBox: []float64{10, 20, 100, 120}},
			{RegionID: "bad-bbox", Kind: "table", BBox: []float64{1001, 20, 1100, 120}},
			{RegionID: "ok-table", Kind: "table", BBox: []float64{100, 200, 600, 700}},
		},
	}

	candidates, warnings := engineeringRegionCandidatesWithSkippedReasons(manifest, page, asset, plan)

	if len(candidates) != 1 || candidates[0].RegionID != "ok-table" {
		t.Fatalf("candidates=%#v, want only ok-table", candidates)
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings=%d, want 2: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "bad-kind") || !strings.Contains(warnings[0], "invalid kind") {
		t.Fatalf("first warning=%q, want invalid kind with region id", warnings[0])
	}
	if !strings.Contains(warnings[1], "bad-bbox") || !strings.Contains(warnings[1], "out of normalized bounds") {
		t.Fatalf("second warning=%q, want bbox error with region id", warnings[1])
	}
}

func TestEngineeringPagePlanBBoxToImageBBoxRejectsInvalidNormalizedCoordinates(t *testing.T) {
	got, err := engineeringPagePlanBBoxToImageBBox([]float64{100, 200, 600, 700}, 2000, 1000)
	if err != nil {
		t.Fatalf("convert normalized bbox: %v", err)
	}
	want := []float64{200, 200, 1200, 700}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("image bbox=%v, want %v", got, want)
		}
	}

	_, err = engineeringPagePlanBBoxToImageBBox([]float64{1001, 40, 1100, 275}, 2000, 1000)
	if err == nil || !strings.Contains(err.Error(), "out of normalized bounds") {
		t.Fatalf("out-of-bounds bbox error=%v", err)
	}
	_, err = engineeringPagePlanBBoxToImageBBox([]float64{math.NaN(), 40, 300, 275}, 2000, 1000)
	if err == nil || !strings.Contains(err.Error(), "not finite") {
		t.Fatalf("nan bbox error=%v", err)
	}
	_, err = engineeringPagePlanBBoxToImageBBox([]float64{300, 40, 200, 275}, 2000, 1000)
	if err == nil || !strings.Contains(err.Error(), "is invalid") {
		t.Fatalf("negative-size bbox error=%v", err)
	}
}

func TestDocumentVisualEngineeringPlanUsesVLMRegionsAsFinalObjects(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	page := manifest.Pages[0]
	asset := documentVisualPageAsset{Width: 640, Height: 480}
	plan := engineeringDrawingPagePlan{
		Regions: []engineeringDrawingRegion{
			{
				RegionID: "duplicate_view",
				Kind:     "part",
				BBox:     []float64{10, 20, 300, 400},
			},
			{
				RegionID: "new_view",
				Kind:     "part",
				BBox:     []float64{400, 20, 620, 200},
			},
		},
	}

	candidates := engineeringRegionCandidates(manifest, page, asset, plan)
	if len(candidates) != 2 {
		t.Fatalf("candidates=%d, want both VLM regions as final candidates", len(candidates))
	}
	if candidates[0].RegionID != "duplicate_view" || candidates[1].RegionID != "new_view" {
		t.Fatalf("candidates=%s,%s; want duplicate_view,new_view", candidates[0].RegionID, candidates[1].RegionID)
	}
	if candidates[0].ObjectID == manifest.Objects[0].ObjectID {
		t.Fatalf("VLM region should create final engineering object, not reuse base object id")
	}
}

func TestDocumentVisualEngineeringPlanAcceptsOriginalRegionKinds(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	page := manifest.Pages[0]
	asset := documentVisualPageAsset{Width: 640, Height: 480}
	plan := engineeringDrawingPagePlan{
		Regions: []engineeringDrawingRegion{
			{RegionID: "view", Kind: "drawing_view", BBox: []float64{10, 20, 100, 120}},
			{RegionID: "notes", Kind: "notes", BBox: []float64{110, 20, 200, 120}},
			{RegionID: "dims", Kind: "dimension_cluster", BBox: []float64{210, 20, 300, 120}},
		},
	}

	candidates := engineeringRegionCandidates(manifest, page, asset, plan)
	if len(candidates) != 3 {
		t.Fatalf("candidates=%d, want original region kinds accepted", len(candidates))
	}
}

func TestEngineeringRegionIsTableLikeUsesContainsTables(t *testing.T) {
	cases := []struct {
		name      string
		candidate engineeringRegionCandidate
		want      bool
	}{
		{name: "table kind", candidate: engineeringRegionCandidate{Kind: "table"}, want: true},
		{name: "title block kind", candidate: engineeringRegionCandidate{Kind: "title_block"}, want: true},
		{name: "part with table", candidate: engineeringRegionCandidate{Kind: "part", ContainsTables: true}, want: true},
		{name: "part without table", candidate: engineeringRegionCandidate{Kind: "part"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := engineeringRegionIsTableLike(tc.candidate); got != tc.want {
				t.Fatalf("engineeringRegionIsTableLike(%#v)=%v, want %v", tc.candidate, got, tc.want)
			}
		})
	}
}

func TestDocumentVisualEngineeringPlanKeepsHighOverlapVLMRegions(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects = nil
	manifest.Pages[0].ObjectIDs = nil
	page := manifest.Pages[0]
	asset := documentVisualPageAsset{Width: 640, Height: 480}
	plan := engineeringDrawingPagePlan{
		Regions: []engineeringDrawingRegion{
			{
				RegionID: "view_a",
				Kind:     "part",
				BBox:     []float64{10, 20, 300, 400},
			},
			{
				RegionID: "view_a_duplicate",
				Kind:     "part",
				BBox:     []float64{12, 22, 298, 398},
			},
			{
				RegionID: "view_b",
				Kind:     "part",
				BBox:     []float64{400, 20, 620, 200},
			},
		},
	}

	candidates := engineeringRegionCandidates(manifest, page, asset, plan)
	if len(candidates) != 3 {
		t.Fatalf("candidates=%d, want all VLM regions", len(candidates))
	}
	if candidates[0].RegionID != "view_a" || candidates[1].RegionID != "view_a_duplicate" || candidates[2].RegionID != "view_b" {
		t.Fatalf("candidates=%s,%s,%s; want view_a,view_a_duplicate,view_b", candidates[0].RegionID, candidates[1].RegionID, candidates[2].RegionID)
	}
}

func TestEngineeringDrawingAttachmentMarkdownUsesRegionExtractEntities(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects[0].OCR = "DETAIL Z\n\n(300.55)"
	manifest.Objects[0].Text = manifest.Objects[0].OCR
	manifest.Objects[0].ImageFileID = "obj-image-1"
	manifest.Entities = append(manifest.Entities,
		documentVisualEntity{
			Type:       "dimension",
			Value:      "(307.55)",
			PageNumber: 1,
			ObjectID:   "obj-1",
			Metadata: map[string]interface{}{
				"extraction_source": "engineering_region_extract",
				"kind":              "dimension",
				"view_name":         "DETAIL Z",
				"source":            "region_crop",
			},
		},
		documentVisualEntity{
			Type:       "dimension",
			Value:      "27.57±0.15",
			PageNumber: 1,
			ObjectID:   "obj-1",
			Metadata: map[string]interface{}{
				"extraction_source": "engineering_region_extract",
				"kind":              "dimension",
				"view_name":         "SECTION B-B",
				"source":            "region_crop",
			},
		},
		documentVisualEntity{
			Type:  "dimension",
			Value: "(300.55)",
		},
	)

	attachment := engineeringDrawingAttachmentMarkdown(manifest)

	for _, want := range []string{"### OCR", "(307.55)", "27.57±0.15", "engineering_region_extract"} {
		if !strings.Contains(attachment, want) {
			t.Fatalf("attachment missing %q:\n%s", want, attachment)
		}
	}
	if strings.Contains(attachment, "(300.55)") {
		t.Fatalf("attachment should not include non-engineering entity:\n%s", attachment)
	}
}

func TestEngineeringDrawingManifestMarkdownUsesOnlyFinalEngineeringObjects(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects[0].ImageFileID = "obj-image-1"
	manifest.Objects = append(manifest.Objects, documentVisualObject{
		ObjectID:         "eng-1",
		ObjectKind:       "drawing_view",
		PageNumber:       1,
		BBox:             []float64{20, 30, 290, 390},
		ImageFileID:      "engineering-region-1",
		PageImageFileID:  "page-img-1",
		ExtractionSource: "engineering_page_plan",
	})
	manifest.Entities = append(manifest.Entities, documentVisualEntity{
		Type:       "dimension",
		Value:      "PLASTIC MARKINGS ZONE",
		PageNumber: 1,
		ObjectID:   "eng-1",
		Metadata: map[string]interface{}{
			"extraction_source": "engineering_region_extract",
			"kind":              "marking",
			"source":            "region_crop",
		},
	})

	md := string(engineeringDrawingManifestMarkdown(manifest))

	if !strings.Contains(md, "![](engineering-region-1)\n\n### OCR\n\n- PLASTIC MARKINGS ZONE") {
		t.Fatalf("engineering region should have its own image and OCR block:\n%s", md)
	}
	if strings.Contains(md, "obj-image-1") {
		t.Fatalf("base parser image should not appear in final engineering markdown:\n%s", md)
	}
	if strings.Count(md, "PLASTIC MARKINGS ZONE") != 1 {
		t.Fatalf("engineering value should not be duplicated:\n%s", md)
	}
	if !strings.Contains(md, "object_id=eng-1") {
		t.Fatalf("engineering line should preserve source object_id:\n%s", md)
	}
}

func TestEngineeringDrawingManifestMarkdownIncludesObjectsWithoutOCR(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects[0].ImageFileID = "base-parser-image"
	manifest.Objects = append(manifest.Objects, documentVisualObject{
		ObjectID:         "eng-empty",
		ObjectKind:       "drawing_view",
		PageNumber:       1,
		ImageFileID:      "engineering-region-empty",
		PageImageFileID:  "page-img-1",
		Caption:          "Pure geometry view",
		ExtractionSource: "engineering_page_plan",
	})
	manifest.Entities = nil

	md := string(engineeringDrawingManifestMarkdown(manifest))

	captionIndex := strings.Index(md, "### Caption\n\nPure geometry view")
	imageIndex := strings.Index(md, "![](engineering-region-empty)")
	if captionIndex < 0 || imageIndex < 0 || captionIndex > imageIndex {
		t.Fatalf("caption should appear before image:\n%s", md)
	}
	if !strings.Contains(md, "![](engineering-region-empty)\n\n### OCR") {
		t.Fatalf("object without OCR should still have image/OCR block:\n%s", md)
	}
	if strings.Contains(md, "base-parser-image") {
		t.Fatalf("base parser image should not appear in final engineering markdown:\n%s", md)
	}
}

func TestEngineeringEntitiesFromExtractIncludesVisualMarkerMetadata(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	candidate := engineeringRegionCandidate{
		RegionID:        "view-a",
		Label:           "A-A剖视图",
		PageNumber:      1,
		BBox:            []float64{10, 20, 300, 400},
		PageImageFileID: "page-img-1",
		ObjectID:        "obj-marker",
	}
	obj := documentVisualObject{
		ObjectID:        candidate.ObjectID,
		PageNumber:      candidate.PageNumber,
		BBox:            candidate.BBox,
		OCRBBox:         candidate.BBox,
		PageImageFileID: candidate.PageImageFileID,
		ImageFileID:     "region-img-1",
	}

	entities := engineeringEntitiesFromExtract(manifest.Source.FileID, obj, candidate, engineeringDrawingRegionExtract{
		RegionID: "view-a",
		Items: []engineeringDrawingExtractItem{
			{
				Text: "34±0.35",
				Kind: "dimension",
				BBox: []float64{40, 50, 120, 80},
				VisualMarker: &engineeringDrawingVisualMarker{
					Shape: "rounded_rectangle",
					BBox:  []float64{35, 45, 125, 85},
				},
			},
		},
	})
	if len(entities) != 1 {
		t.Fatalf("entities=%d, want 1", len(entities))
	}
	meta := entities[0].Metadata
	if getString(meta, "visual_marker_shape") != "rounded_rectangle" {
		t.Fatalf("visual_marker_shape=%#v", meta["visual_marker_shape"])
	}
	if getString(meta, "visual_marker_id") == "" {
		t.Fatalf("visual_marker_id missing: %#v", meta)
	}
	if got := interfaceToFloat64Slice(meta["visual_marker_bbox"]); len(got) != 4 || got[0] != 35 || got[3] != 85 {
		t.Fatalf("visual_marker_bbox=%#v", meta["visual_marker_bbox"])
	}
}

func TestEngineeringDrawingHasMarkdownObjectAllowsEmptyOCRDocuments(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects[0].ImageFileID = "base-parser-image"
	manifest.Objects = append(manifest.Objects, documentVisualObject{
		ObjectID:         "eng-empty",
		ObjectKind:       "drawing_view",
		PageNumber:       1,
		ImageFileID:      "engineering-region-empty",
		ExtractionSource: "engineering_page_plan",
	})
	manifest.Entities = nil

	if attachment := engineeringDrawingAttachmentMarkdown(manifest); attachment != "" {
		t.Fatalf("attachment=%q, want empty without OCR entities", attachment)
	}
	if !engineeringDrawingHasMarkdownObject(manifest) {
		t.Fatal("engineering markdown object should allow document generation even without OCR")
	}
}

func TestEngineeringDrawingFinalPagesDropBaseParserText(t *testing.T) {
	manifest := sampleDocumentVisualManifest()

	pages := engineeringDrawingFinalPages(manifest.Pages, map[int]documentVisualPageAsset{
		1: {FileID: "engineering-page-img-1", Width: 640, Height: 480},
	})

	if len(pages) != 1 {
		t.Fatalf("pages=%d, want 1", len(pages))
	}
	if pages[0].Text != "" {
		t.Fatalf("page text=%q, want empty", pages[0].Text)
	}
	if pages[0].Summary != "" {
		t.Fatalf("page summary=%q, want empty", pages[0].Summary)
	}
	if len(pages[0].ObjectIDs) != 0 {
		t.Fatalf("page object ids=%v, want empty", pages[0].ObjectIDs)
	}
	if pages[0].PageImageFileID != "engineering-page-img-1" {
		t.Fatalf("page image file id=%q, want engineering-page-img-1", pages[0].PageImageFileID)
	}
	if pages[0].Width != 640 || pages[0].Height != 480 {
		t.Fatalf("page size=%vx%v, want rendered image size", pages[0].Width, pages[0].Height)
	}
}

func TestEngineeringDrawingObjectContextDoesNotUseBaseParserText(t *testing.T) {
	got := engineeringDrawingObjectContext(engineeringRegionCandidate{
		Label:  "DETAIL Z",
		Reason: "contains dimensions",
	})

	if strings.Contains(got, "图纸号") {
		t.Fatalf("context should not include base parser page text: %q", got)
	}
	if !strings.Contains(got, "DETAIL Z") || !strings.Contains(got, "contains dimensions") {
		t.Fatalf("context=%q, want engineering candidate context", got)
	}
}

func TestValidateEngineeringDrawingObjectEntitiesAllowsMissingOCR(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects = []documentVisualObject{
		{
			ObjectID:        "eng-1",
			PageNumber:      1,
			ImageFileID:     "engineering-region-1",
			PageImageFileID: "page-img-1",
		},
	}
	manifest.Entities = nil

	err := validateEngineeringDrawingObjectEntities(manifest)
	if err != nil {
		t.Fatalf("object without OCR should be allowed: %v", err)
	}
}

func TestValidateEngineeringDrawingObjectEntitiesRejectsEmptyPagePlan(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects = nil
	manifest.Entities = nil
	manifest.Pages[0].ObjectIDs = nil

	err := validateEngineeringDrawingObjectEntities(manifest)
	if err == nil || !strings.Contains(err.Error(), "produced no engineering objects") {
		t.Fatalf("empty engineering objects error=%v", err)
	}
}

func TestValidateEngineeringDrawingObjectEntitiesAllowsPartialOCR(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects = []documentVisualObject{
		{
			ObjectID:        "eng-1",
			PageNumber:      1,
			ImageFileID:     "engineering-region-1",
			PageImageFileID: "page-img-1",
		},
		{
			ObjectID:        "eng-2",
			PageNumber:      1,
			ImageFileID:     "engineering-region-2",
			PageImageFileID: "page-img-1",
		},
	}
	manifest.Entities = []documentVisualEntity{
		{
			Type:     "dimension",
			Value:    "27.57±0.15",
			ObjectID: "eng-1",
			Metadata: map[string]interface{}{
				"extraction_source": "engineering_region_extract",
			},
		},
	}

	err := validateEngineeringDrawingObjectEntities(manifest)
	if err != nil {
		t.Fatalf("partial OCR entities should be allowed: %v", err)
	}
}

func TestAppendEngineeringDrawingImageDocumentsAddsVLMRegionImages(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects = nil
	manifest.Pages[0].ObjectIDs = nil
	manifest.Objects = append(manifest.Objects, documentVisualObject{
		ObjectID:         "eng-1",
		ObjectKind:       "drawing_view",
		PageNumber:       1,
		BBox:             []float64{20, 30, 290, 390},
		OCRBBox:          []float64{10, 20, 300, 400},
		ImageFileID:      "engineering-region-1",
		PageImageFileID:  "page-img-1",
		Caption:          "DETAIL Z",
		ExtractionSource: "engineering_page_plan",
	})

	docs, err := appendEngineeringDrawingImageDocuments(nil, manifest, "md-new")
	if err != nil {
		t.Fatalf("append engineering image documents: %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("docs=%d, want 1", len(docs))
	}
	doc := documentFromProto(docs[0])
	if doc.Type != "image" {
		t.Fatalf("doc type=%q, want image", doc.Type)
	}
	if got := getString(doc.Metadata, "image_url"); got != "engineering-region-1" {
		t.Fatalf("image_url=%q, want engineering-region-1", got)
	}
	if got := getString(doc.Metadata, "s3_image_url"); got != "engineering-region-1" {
		t.Fatalf("s3_image_url=%q, want engineering-region-1", got)
	}
	if got := getString(doc.Metadata, "md_file_id"); got != "md-new" {
		t.Fatalf("md_file_id=%q, want md-new", got)
	}
}

func TestAppendEngineeringDrawingImageDocumentsAcceptsWarningMetadata(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects = []documentVisualObject{
		{
			ObjectID:           "eng-warning",
			ObjectKind:         "part",
			PageNumber:         1,
			BBox:               []float64{20, 30, 290, 390},
			OCRBBox:            []float64{20, 30, 290, 390},
			ImageFileID:        "engineering-region-warning",
			PageImageFileID:    "page-img-1",
			Caption:            "DETAIL Z",
			ExtractionSource:   "engineering_page_plan",
			ExtractionWarnings: []string{"bbox_clipped_to_page_image"},
		},
	}

	docs, err := appendEngineeringDrawingImageDocuments(nil, manifest, "md-new")
	if err != nil {
		t.Fatalf("append engineering image documents with warning metadata: %v", err)
	}
	doc := documentFromProto(docs[0])
	warnings, ok := doc.Metadata["extraction_warnings"].([]interface{})
	if !ok || len(warnings) != 1 || warnings[0] != "bbox_clipped_to_page_image" {
		t.Fatalf("extraction_warnings=%#v, want proto-compatible list", doc.Metadata["extraction_warnings"])
	}
}

func TestAppendEngineeringDrawingImageDocumentsAddsTableDocuments(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects = []documentVisualObject{
		{
			ObjectID:         "table-ok",
			ObjectKind:       "table",
			PageNumber:       1,
			BBox:             []float64{20, 30, 290, 390},
			OCRBBox:          []float64{20, 30, 290, 390},
			ImageFileID:      "table-ok-img",
			PageImageFileID:  "page-img-1",
			Caption:          "明细表",
			Text:             "<table><tr><td>A</td></tr></table>",
			ExtractionSource: "engineering_page_plan",
			Table: &documentVisualTableMetadata{
				TableStatus:            engineeringTableStatusExtracted,
				TableSource:            engineeringTableSourceVLMExtract,
				HTMLValidationStatus:   engineeringTableValidationPassed,
				CandidateSource:        engineeringTableCandidateSourcePagePlan,
				PagePlanMatch:          true,
				PagePlanRegionID:       "table-region",
				PagePlanRegionKind:     "table",
				PagePlanBBox:           []float64{20, 30, 290, 390},
				PagePlanBBoxCoordinate: engineeringPagePlanBBoxCoordinate,
			},
		},
		{
			ObjectID:         "table-bad",
			ObjectKind:       "title_block",
			PageNumber:       1,
			BBox:             []float64{300, 30, 500, 130},
			OCRBBox:          []float64{300, 30, 500, 130},
			ImageFileID:      "table-bad-img",
			PageImageFileID:  "page-img-1",
			Caption:          "标题栏",
			ExtractionSource: "engineering_page_plan",
			Table: &documentVisualTableMetadata{
				TableStatus:            engineeringTableStatusUnavailable,
				TableSource:            engineeringTableSourceUnavailable,
				HTMLValidationStatus:   engineeringTableValidationFailed,
				CandidateSource:        engineeringTableCandidateSourcePagePlan,
				PagePlanMatch:          true,
				PagePlanRegionID:       "title-region",
				PagePlanRegionKind:     "title_block",
				PagePlanBBox:           []float64{300, 30, 500, 130},
				PagePlanBBoxCoordinate: engineeringPagePlanBBoxCoordinate,
				FailureStage:           engineeringTableFailureStageExtract,
				TableUnavailableReason: "vlm returned no tables",
			},
		},
	}

	docs, err := appendEngineeringDrawingImageDocuments(nil, manifest, "md-new")
	if err != nil {
		t.Fatalf("append engineering image documents: %v", err)
	}
	if len(docs) != 4 {
		t.Fatalf("docs=%d, want two image docs and two table docs", len(docs))
	}
	tableDocs := map[string]Document{}
	for _, protoDoc := range docs {
		doc := documentFromProto(protoDoc)
		if doc.Type == "table" {
			tableDocs[doc.ID] = doc
		}
	}
	if got := tableDocs["table-ok-table"].Content; got != "<table><tr><td>A</td></tr></table>" {
		t.Fatalf("successful table content=%q", got)
	}
	if got := tableDocs["table-bad-table"].Content; got != "" {
		t.Fatalf("unavailable table content=%q, want empty", got)
	}
	if getString(tableDocs["table-bad-table"].Metadata, "table_status") != engineeringTableStatusUnavailable {
		t.Fatalf("unavailable table metadata=%#v", tableDocs["table-bad-table"].Metadata)
	}
	if got := getString(tableDocs["table-ok-table"].Metadata, "page_plan_bbox_coordinate"); got != engineeringPagePlanBBoxCoordinate {
		t.Fatalf("page_plan_bbox_coordinate=%q", got)
	}
}

func TestEngineeringTableMetadataUsesNormalizedPagePlanBBox(t *testing.T) {
	candidate := engineeringRegionCandidate{
		RegionID: "table-region",
		Kind:     "table",
		BBox:     []float64{200, 200, 1200, 700},
		PlanBBox: []float64{100, 200, 600, 700},
	}

	meta := engineeringTableMetadata(candidate, engineeringTableStatusExtracted, engineeringTableSourceVLMExtract, engineeringTableValidationPassed, "")

	if len(meta.PagePlanBBox) != 4 || meta.PagePlanBBox[2] != 600 {
		t.Fatalf("page plan bbox=%v, want normalized plan bbox", meta.PagePlanBBox)
	}
	if meta.PagePlanBBoxCoordinate != engineeringPagePlanBBoxCoordinate {
		t.Fatalf("page plan bbox coordinate=%q", meta.PagePlanBBoxCoordinate)
	}
}

func TestFirstValidEngineeringTableHTMLSkipsInvalidCandidates(t *testing.T) {
	table, err := firstValidEngineeringTableHTML([]engineeringTableExtractTable{
		{HTML: ""},
		{HTML: "<div>not a table</div>"},
		{HTML: "<table><tr><td>A</td></tr></table>", Uncertain: true, Truncated: true},
	})
	if err != nil {
		t.Fatalf("first valid table: %v", err)
	}
	if table.HTML != "<table><tr><td>A</td></tr></table>" {
		t.Fatalf("table html=%q", table.HTML)
	}
}

func TestDocumentVisualTextIndexSkipsUnavailableTableObject(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects = []documentVisualObject{
		{
			ObjectID:        "table-ok",
			ObjectKind:      "table",
			PageNumber:      1,
			BBox:            []float64{20, 30, 290, 390},
			ImageFileID:     "table-ok-img",
			PageImageFileID: "page-img-1",
			Text:            "<table><tr><td>A</td></tr></table>",
			Table: &documentVisualTableMetadata{
				TableStatus:          engineeringTableStatusExtracted,
				TableSource:          engineeringTableSourceVLMExtract,
				HTMLValidationStatus: engineeringTableValidationPassed,
			},
		},
		{
			ObjectID:        "table-bad",
			ObjectKind:      "table",
			PageNumber:      1,
			BBox:            []float64{300, 30, 500, 130},
			ImageFileID:     "table-bad-img",
			PageImageFileID: "page-img-1",
			Table: &documentVisualTableMetadata{
				TableStatus:          engineeringTableStatusUnavailable,
				TableSource:          engineeringTableSourceUnavailable,
				HTMLValidationStatus: engineeringTableValidationFailed,
				FailureStage:         engineeringTableFailureStageExtract,
			},
		},
	}

	docs, err := buildDocumentVisualTextIndexDocuments(manifest)
	if err != nil {
		t.Fatalf("build text index documents: %v", err)
	}
	for _, doc := range docs {
		meta := doc["metadata"].(map[string]interface{})
		if meta["object_id"] == "table-bad" {
			t.Fatalf("unavailable table should not enter text index: %#v", doc)
		}
	}
}

func TestDocumentVisualTextIndexSkipsFailedOCRObject(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Objects = []documentVisualObject{
		{
			ObjectID:                "ocr-failed",
			ObjectKind:              "drawing_view",
			PageNumber:              1,
			BBox:                    []float64{20, 30, 290, 390},
			ImageFileID:             "ocr-failed-img",
			PageImageFileID:         "page-img-1",
			Caption:                 "DETAIL Z",
			Context:                 "page plan caption only",
			ExtractionWarnings:      []string{engineeringRegionExtractFailedWarning},
			ExtractionFailureReason: "vlm timeout",
		},
		{
			ObjectID:        "ocr-ok",
			ObjectKind:      "drawing_view",
			PageNumber:      1,
			BBox:            []float64{300, 30, 500, 130},
			ImageFileID:     "ocr-ok-img",
			PageImageFileID: "page-img-1",
			OCR:             "34±0.35",
			Text:            "34±0.35",
		},
	}

	docs, err := buildDocumentVisualTextIndexDocuments(manifest)
	if err != nil {
		t.Fatalf("build text index documents: %v", err)
	}
	for _, doc := range docs {
		meta := doc["metadata"].(map[string]interface{})
		if meta["object_id"] == "ocr-failed" {
			t.Fatalf("failed OCR object should not enter text index: %#v", doc)
		}
		content := toString(doc["content"])
		if strings.Contains(content, "page plan caption only") {
			t.Fatalf("failed OCR object context should not enter text index content: %#v", doc)
		}
	}
}

func TestDocumentVisualValidationRejectsDanglingEntityObjectID(t *testing.T) {
	manifest := sampleDocumentVisualManifest()
	manifest.Entities = append(manifest.Entities, documentVisualEntity{
		EntityID:   "ent-dangling",
		Type:       "dimension",
		Value:      "27.57±0.15",
		PageNumber: 1,
		ObjectID:   "missing-object",
		Metadata:   map[string]interface{}{"extraction_source": "engineering_region_extract"},
	})

	validation := validateDocumentVisualManifest(manifest, documentVisualBuildOptions{})

	if validation.Valid {
		t.Fatalf("dangling entity object_id should fail validation")
	}
	if len(validation.Errors) == 0 || !strings.Contains(validation.Errors[0], "unknown object_id") {
		t.Fatalf("validation errors=%v", validation.Errors)
	}
}

func sampleDocumentVisualManifest() documentVisualManifest {
	return documentVisualManifest{
		SchemaVersion: documentVisualSchemaVersion,
		Profile:       documentVisualDefaultProfile,
		Source: documentVisualSource{
			FileID:    "file-1",
			FileName:  "20C114220.pdf",
			PageCount: 1,
		},
		Pages: []documentVisualPage{
			{
				PageNumber:      1,
				PageImageFileID: "page-img-1",
				Width:           1000,
				Height:          500,
				BBox:            []float64{0, 0, 1000, 500},
				Text:            "图纸号 20C114220 材质 Q235",
				Summary:         "图纸号 20C114220",
				ObjectIDs:       []string{"obj-1"},
			},
		},
		Objects: []documentVisualObject{
			{
				ObjectID:        "obj-1",
				ObjectKind:      "drawing_view",
				PageNumber:      1,
				BBox:            []float64{10, 20, 300, 400},
				ImageFileID:     "obj-img-1",
				PageImageFileID: "page-img-1",
				Text:            "主视图",
				OCR:             "尺寸 10x20",
				Caption:         "工业零件主视图",
				Context:         "主视图 图纸号 20C114220 材质 Q235",
			},
		},
		Entities: []documentVisualEntity{
			{EntityID: "ent-1", Type: "material", Value: "Q235", Evidence: "材质 Q235"},
		},
		Validation: documentVisualValidation{Valid: true},
	}
}

func testDocument(content string, meta map[string]interface{}) *data.Document {
	s, err := structpb.NewStruct(meta)
	if err != nil {
		panic(err)
	}
	return &data.Document{Content: content, Metadata: s}
}
