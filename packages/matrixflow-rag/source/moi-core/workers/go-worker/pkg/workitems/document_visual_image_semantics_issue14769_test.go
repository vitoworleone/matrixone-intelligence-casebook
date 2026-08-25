package workitems

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matrixflow/moi-core/model/data"
	"github.com/matrixflow/moi-core/model/mowl"
	workruntime "github.com/matrixflow/moi-core/workers/go-worker/pkg/runtime"
	"github.com/stretchr/testify/require"
)

func TestFixPR14773DocumentVisualParseProjectsValidImageSemanticsOnceInStableOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet ||
			r.URL.Path != "/api/v1/workspaces/ws-test/files/source-1/download" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("source placeholder"))
	}))
	defer server.Close()

	distinct := mustDocumentProto(t, Document{
		ID:      "image-distinct",
		Type:    "image",
		Content: "serial 123",
		Metadata: map[string]interface{}{
			"page_number": 1,
			"block_uuid":  "image-distinct",
			"ocr":         "serial 123 and status label",
			"caption":     "Serial 123 and status label",
		},
	})
	exactDuplicate := mustDocumentProto(t, Document{
		ID:      "image-exact-duplicate",
		Type:    "image",
		Content: "Shared semantic text",
		Metadata: map[string]interface{}{
			"page_number": 1,
			"block_uuid":  "image-exact-duplicate",
			"ocr":         "Shared semantic text",
			"caption":     "  Shared semantic text  ",
		},
	})
	partialDuplicate := mustDocumentProto(t, Document{
		ID:      "image-partial-duplicate",
		Type:    "image",
		Content: "Repeated OCR and local text",
		Metadata: map[string]interface{}{
			"page_number":        1,
			"block_uuid":         "image-partial-duplicate",
			"ocr":                "Repeated OCR and local text",
			"figure_caption":     "  Repeated OCR and local text  ",
			"caption":            "Distinct caption",
			"caption_block_uuid": "source-caption-partial-duplicate",
		},
	})
	partialDuplicateCaptionBlock := mustDocumentProto(t, Document{
		ID:      "source-caption-partial-duplicate",
		Type:    "text",
		Content: "Repeated OCR and local text",
		Metadata: map[string]interface{}{
			"page_number": 1,
			"block_uuid":  "source-caption-partial-duplicate",
			"role":        "caption",
			"caption_for": "image-partial-duplicate",
		},
	})
	figureLinked := mustDocumentProto(t, Document{
		ID:      "image-figure-linked",
		Type:    "image",
		Content: "local diagram label",
		Metadata: map[string]interface{}{
			"page_number":        1,
			"block_uuid":         "image-figure-linked",
			"ocr":                "RESET_N",
			"figure_caption":     "Figure 9-2. Power domain overview",
			"figure_no":          "9-2",
			"caption":            "Generated diagram description",
			"caption_block_uuid": "source-caption-9-2",
		},
	})
	figureCaptionBlock := mustDocumentProto(t, Document{
		ID:      "source-caption-9-2",
		Type:    "text",
		Content: "Figure 9-2. Power domain overview",
		Metadata: map[string]interface{}{
			"page_number": 1,
			"block_uuid":  "source-caption-9-2",
			"role":        "caption",
			"caption_for": "image-figure-linked",
			"figure_no":   "9-2",
		},
	})
	disabled := false
	payload, err := json.Marshal(documentVisualParseInput{
		Sources: []*data.Source{{
			FileId:   "source-1",
			Name:     "fixture.txt",
			MimeType: "text/plain",
		}},
		Documents:            []*data.Document{distinct, exactDuplicate, partialDuplicate, partialDuplicateCaptionBlock, figureLinked, figureCaptionBlock},
		Profile:              documentVisualStandardRAGProfile,
		RequirePageImages:    &disabled,
		RequireObjectImages:  &disabled,
		RequireVisualContext: &disabled,
		WriteManifestFile:    &disabled,
	})
	require.NoError(t, err)

	msg := &mowl.MowlMessage{Data: string(payload)}
	item := &DocumentVisualParse{
		Factory: workruntime.NewClientFactory(server.URL, "worker-key", 0),
	}
	_, err = item.Handle(t.Context(), &mockWctx{}, msg)
	require.NoError(t, err)

	var out documentVisualParseOutput
	require.NoError(t, json.Unmarshal([]byte(msg.Data), &out))
	require.True(t, out.Validation.Valid)
	require.Len(t, out.Manifest.Pages, 1)
	require.Len(t, out.Manifest.Objects, 4)

	objects := make(map[string]documentVisualObject, len(out.Manifest.Objects))
	for _, object := range out.Manifest.Objects {
		objects[object.ObjectID] = object
	}

	gotDistinct := objects["image-distinct"]
	require.Equal(t, "serial 123", gotDistinct.Text)
	require.Equal(t, "serial 123 and status label", gotDistinct.OCR)
	require.Equal(t, "Serial 123 and status label", gotDistinct.Caption)
	require.Equal(t, []string{
		"serial 123 and status label",
		"Serial 123 and status label",
		"serial 123",
	}, strings.Split(gotDistinct.Context, "\n"))

	gotDuplicate := objects["image-exact-duplicate"]
	require.Equal(t, "Shared semantic text", gotDuplicate.Text)
	require.Equal(t, "Shared semantic text", gotDuplicate.OCR)
	require.Equal(t, "  Shared semantic text  ", gotDuplicate.Caption)
	require.Equal(t, "Shared semantic text", gotDuplicate.Context)

	gotPartialDuplicate := objects["image-partial-duplicate"]
	require.Equal(t, "Repeated OCR and local text", gotPartialDuplicate.Text)
	require.Equal(t, "Repeated OCR and local text", gotPartialDuplicate.OCR)
	require.Equal(t, "Distinct caption", gotPartialDuplicate.Caption)
	require.Equal(t, "Repeated OCR and local text\nDistinct caption", gotPartialDuplicate.Context)

	gotFigureLinked := objects["image-figure-linked"]
	require.Equal(t,
		"RESET_N\nFigure 9-2. Power domain overview\nGenerated diagram description\nlocal diagram label",
		gotFigureLinked.Context,
	)
	// Assert the public JSON wire rather than private Go fields so this remains a
	// contract test even while the manifest schema gains structured figure data.
	var wire struct {
		Manifest struct {
			Objects []struct {
				ObjectID         string `json:"object_id"`
				FigureCaption    string `json:"figure_caption"`
				FigureNo         string `json:"figure_no"`
				CaptionBlockUUID string `json:"caption_block_uuid"`
			} `json:"objects"`
		} `json:"manifest"`
	}
	require.NoError(t, json.Unmarshal([]byte(msg.Data), &wire))
	var linkedWire *struct {
		ObjectID         string `json:"object_id"`
		FigureCaption    string `json:"figure_caption"`
		FigureNo         string `json:"figure_no"`
		CaptionBlockUUID string `json:"caption_block_uuid"`
	}
	var duplicateWire *struct {
		ObjectID         string `json:"object_id"`
		FigureCaption    string `json:"figure_caption"`
		FigureNo         string `json:"figure_no"`
		CaptionBlockUUID string `json:"caption_block_uuid"`
	}
	for index := range wire.Manifest.Objects {
		switch wire.Manifest.Objects[index].ObjectID {
		case "image-figure-linked":
			linkedWire = &wire.Manifest.Objects[index]
		case "image-partial-duplicate":
			duplicateWire = &wire.Manifest.Objects[index]
		}
	}
	require.NotNil(t, linkedWire)
	require.Equal(t, "Figure 9-2. Power domain overview", linkedWire.FigureCaption)
	require.Equal(t, "9-2", linkedWire.FigureNo)
	require.Equal(t, "source-caption-9-2", linkedWire.CaptionBlockUUID)
	require.NotNil(t, duplicateWire)
	require.Equal(t, "  Repeated OCR and local text  ", duplicateWire.FigureCaption)

	require.Equal(t, strings.Join([]string{
		"serial 123 and status label",
		"Serial 123 and status label",
		"serial 123",
		"Shared semantic text",
		"Repeated OCR and local text",
		"Distinct caption",
		"RESET_N",
		"Figure 9-2. Power domain overview",
		"Generated diagram description",
		"local diagram label",
	}, "\n"), out.Manifest.Pages[0].Text)
	require.Equal(t, 1, strings.Count(out.Manifest.Pages[0].Text, "Figure 9-2. Power domain overview"))
}

func TestIssue14769DocumentVisualTextIndexDoesNotReappendStructuredImageFields(t *testing.T) {
	const canonicalContext = "OCR semantic\nCaption semantic\nlocal text"
	manifest := documentVisualManifest{
		Source:  documentVisualSource{FileID: "source-1"},
		Profile: documentVisualStandardRAGProfile,
		Objects: []documentVisualObject{{
			ObjectID:   "image-1",
			ObjectKind: "drawing_view",
			PageNumber: 1,
			Context:    canonicalContext,
			OCR:        "OCR semantic",
			Caption:    "Caption semantic",
			Text:       "local text",
		}, {
			ObjectID:   "image-context-fallback",
			ObjectKind: "drawing_view",
			PageNumber: 1,
			Context:    " \n ",
			OCR:        "fallback shared semantic",
			Caption:    "fallback caption",
			Text:       "  fallback shared semantic  ",
		}},
	}

	documents, err := buildDocumentVisualTextIndexDocuments(manifest)
	require.NoError(t, err)

	objectDocuments := make(map[string]map[string]interface{})
	for _, document := range documents {
		metadata := ensureMap(document["metadata"])
		if metadata["scope"] == "visual_object" {
			objectDocuments[toString(metadata["object_id"])] = document
		}
	}
	require.Equal(t, canonicalContext, objectDocuments["image-1"]["content"])
	require.Equal(t,
		"fallback shared semantic\nfallback caption",
		objectDocuments["image-context-fallback"]["content"],
	)
}

func TestIssue14769DocumentVisualImageNearbyTextUsesCanonicalContextOnce(t *testing.T) {
	const canonicalContext = "OCR semantic\nCaption semantic\nlocal text\npage context"
	manifest := documentVisualManifest{Profile: documentVisualDefaultProfile}
	object := documentVisualObject{
		ObjectKind: "drawing_view",
		Context:    canonicalContext,
		OCR:        "OCR semantic",
		Caption:    "Caption semantic",
		Text:       "local text",
	}

	require.Equal(t, canonicalContext, documentVisualImageObjectNearbyText(manifest, object))

	fallback := documentVisualObject{
		ObjectKind: "drawing_view",
		Context:    " \n ",
		OCR:        "fallback shared semantic",
		Caption:    "fallback caption",
		Text:       "  fallback shared semantic  ",
	}
	require.Equal(t,
		"fallback shared semantic\nfallback caption",
		documentVisualImageObjectNearbyText(manifest, fallback),
	)
}
