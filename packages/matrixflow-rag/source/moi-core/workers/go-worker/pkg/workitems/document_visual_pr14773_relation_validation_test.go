package workitems

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/matrixflow/moi-core/model/data"
	"github.com/matrixflow/moi-core/model/mowl"
	workruntime "github.com/matrixflow/moi-core/workers/go-worker/pkg/runtime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixPR14773BuildDocumentVisualManifestKeepsValidAndLegacyRelationControls(t *testing.T) {
	const sourceID = "source-1"
	validTextPair := fixPR14773ValidDocumentVisualRelationPair(t, sourceID, "image-valid-text", "caption-valid-text", "Figure 1. Valid text caption", 1)
	validListPair := []*data.Document{
		fixPR14773DocumentVisualDocument(t, sourceID, "image-valid-list", "image", "", 1, map[string]interface{}{
			"figure_caption":     "Figure 2. Valid list caption",
			"caption_block_uuid": "caption-valid-list",
		}),
		fixPR14773DocumentVisualDocument(t, sourceID, "caption-valid-list", "list", "Figure 2. Valid list caption", 1, map[string]interface{}{
			"role":        "caption",
			"caption_for": "image-valid-list",
		}),
	}
	tests := []struct {
		name      string
		documents []*data.Document
	}{
		{
			name: "no relation marker",
			documents: []*data.Document{
				fixPR14773DocumentVisualDocument(t, sourceID, "image-generated-caption-only", "image", "", 1, map[string]interface{}{
					"ocr":     "OCR text",
					"caption": "Generated caption",
				}),
			},
		},
		{
			name: "legacy caption role without v3 edge",
			documents: []*data.Document{
				fixPR14773DocumentVisualDocument(t, sourceID, "legacy-caption", "text", "Legacy contained caption", 1, map[string]interface{}{
					"role":         "caption",
					"contained_in": "legacy-image",
				}),
			},
		},
		{name: "valid reciprocal text caption", documents: validTextPair},
		{name: "valid reciprocal list caption", documents: validListPair},
		{
			name: "valid reciprocal image keeps unrelated role metadata",
			documents: func() []*data.Document {
				documents := fixPR14773ValidDocumentVisualRelationPair(
					t, sourceID, "image-with-role", "caption-for-image-with-role", "Figure 3. Open image metadata", 1,
				)
				imageDocument := documentFromProto(documents[0])
				imageDocument.Metadata["role"] = "decorative"
				documents[0] = mustDocumentProto(t, imageDocument)
				return documents
			}(),
		},
		{
			name: "duplicate IDs outside relation graph",
			documents: []*data.Document{
				fixPR14773DocumentVisualDocument(t, sourceID, "unrelated-duplicate", "text", "ordinary text one", 1, nil),
				fixPR14773DocumentVisualDocument(t, sourceID, "unrelated-duplicate", "text", "ordinary text two", 1, nil),
			},
		},
		{
			name:      "multiple valid reciprocal pairs",
			documents: append(validTextPair, validListPair...),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manifest, err := buildDocumentVisualManifest(
				context.Background(),
				nil,
				"ws-test",
				tc.documents,
				documentVisualLayout{},
				nil,
				documentVisualSource{FileID: sourceID, FileName: sourceID + ".txt", MimeType: "text/plain"},
				documentVisualStandardRAGProfile,
				documentVisualBuildOptions{},
			)
			require.NoError(t, err)
			require.True(t, manifest.Validation.Valid)
		})
	}
}

func TestFixPR14773BuildDocumentVisualManifestRejectsMalformedFigureCaptionRelations(t *testing.T) {
	for _, tc := range fixPR14773MalformedDocumentVisualRelationCases(t, "source-1") {
		t.Run(tc.name, func(t *testing.T) {
			_, err := buildDocumentVisualManifest(
				context.Background(),
				nil,
				"ws-test",
				tc.documents,
				documentVisualLayout{},
				nil,
				documentVisualSource{FileID: "source-1", FileName: "source-1.txt", MimeType: "text/plain"},
				documentVisualStandardRAGProfile,
				documentVisualBuildOptions{},
			)
			require.Error(t, err, "malformed figure-caption relation must fail the whole manifest build")
			require.ErrorContains(t, err, "figure-caption relation")
		})
	}
}

func TestFixPR14773BuildDocumentVisualManifestRejectsCrossSourceFigureCaptionRelation(t *testing.T) {
	documents := fixPR14773ValidDocumentVisualRelationPair(
		t, "source-image", "image-cross-source", "caption-cross-source", "Figure 8. Cross-source caption", 1,
	)
	caption := documentFromProto(documents[1])
	caption.Metadata["raw_file_id"] = "source-caption"
	documents[1] = mustDocumentProto(t, caption)

	_, err := buildDocumentVisualManifest(
		context.Background(),
		nil,
		"ws-test",
		documents,
		documentVisualLayout{},
		nil,
		documentVisualSource{FileID: "source-image", FileName: "source-image.txt", MimeType: "text/plain"},
		documentVisualStandardRAGProfile,
		documentVisualBuildOptions{},
	)
	require.EqualError(t, err,
		`document_visual.parse: validate figure-caption relations: figure-caption relation node 0 (image-cross-source): source "source-image" must match caption "caption-cross-source" source "source-caption"`)
}

func TestFixPR14773DocumentVisualParseRejectsMalformedRelationsBeforeAnyFileRequest(t *testing.T) {
	for _, tc := range fixPR14773MalformedDocumentVisualRelationCases(t, "source-1") {
		t.Run(tc.name, func(t *testing.T) {
			var requestCount atomic.Int32
			var uploadCount atomic.Int32
			server := newFixPR14773DocumentVisualFileServer(t, &requestCount, &uploadCount)
			defer server.Close()

			msg := fixPR14773DocumentVisualParseMessage(t, []*data.Source{{
				FileId:   "source-1",
				Name:     "source-1.txt",
				MimeType: "text/plain",
			}}, tc.documents)
			item := &DocumentVisualParse{
				Factory: workruntime.NewClientFactory(server.URL, "worker-key", 0),
			}

			_, err := item.Handle(t.Context(), &mockWctx{}, msg)
			assert.Error(t, err, "malformed figure-caption relation must fail the Handle boundary")
			assert.ErrorContains(t, err, "figure-caption relation")
			assert.Zero(t, requestCount.Load(), "relation validation must run before source download or any upload")
			assert.Zero(t, uploadCount.Load(), "invalid input must not upload a page image or manifest")
		})
	}
}

func TestFixPR14773DocumentVisualParsePrevalidatesAllSourcesBeforeUploadingFirstManifest(t *testing.T) {
	validDocuments := fixPR14773ValidDocumentVisualRelationPair(t, "source-valid", "image-valid", "caption-valid", "Figure 1. Valid relation", 1)
	invalidDocuments := []*data.Document{
		fixPR14773DocumentVisualDocument(t, "source-invalid", "image-invalid-a", "image", "", 1, map[string]interface{}{
			"figure_caption":     "Figure 2. Invalid relation",
			"caption_block_uuid": "caption-invalid",
		}),
		fixPR14773DocumentVisualDocument(t, "source-invalid", "image-invalid-b", "image", "", 1, map[string]interface{}{
			"figure_caption":     "Figure 2. Invalid relation",
			"caption_block_uuid": "caption-invalid",
		}),
		fixPR14773DocumentVisualDocument(t, "source-invalid", "caption-invalid", "text", "Figure 2. Invalid relation", 1, map[string]interface{}{
			"role":        "caption",
			"caption_for": "image-invalid-a",
		}),
	}
	documents := append(validDocuments, invalidDocuments...)

	var requestCount atomic.Int32
	var uploadCount atomic.Int32
	server := newFixPR14773DocumentVisualFileServer(t, &requestCount, &uploadCount)
	defer server.Close()

	msg := fixPR14773DocumentVisualParseMessage(t, []*data.Source{
		{FileId: "source-valid", Name: "source-valid.txt", MimeType: "text/plain"},
		{FileId: "source-invalid", Name: "source-invalid.txt", MimeType: "text/plain"},
	}, documents)
	item := &DocumentVisualParse{
		Factory: workruntime.NewClientFactory(server.URL, "worker-key", 0),
	}

	_, err := item.Handle(t.Context(), &mockWctx{}, msg)
	assert.EqualError(t, err, `document_visual.parse: validate figure-caption relations for source "source-invalid": figure-caption relation node 0 (image-invalid-a): caption_block_uuid "caption-invalid" must have exactly one IMAGE owner, found 2`)
	assert.Zero(t, requestCount.Load(), "all source groups must be validated before the first source download")
	assert.Zero(t, uploadCount.Load(), "a later invalid source must not leave the first source manifest uploaded")
}

func TestFixPR14773DocumentVisualParseAllowsSourceLocalFigureCaptionBlockIDs(t *testing.T) {
	first := fixPR14773ValidDocumentVisualRelationPair(t, "source-a", "local-image", "local-caption", "Figure 1. Source-local relation", 1)
	second := fixPR14773ValidDocumentVisualRelationPair(t, "source-b", "local-image", "local-caption", "Figure 1. Source-local relation", 1)
	documents := append(first, second...)

	var requestCount atomic.Int32
	var uploadCount atomic.Int32
	server := newFixPR14773DocumentVisualFileServer(t, &requestCount, &uploadCount)
	defer server.Close()

	msg := fixPR14773DocumentVisualParseMessage(t, []*data.Source{
		{FileId: "source-a", Name: "source-a.txt", MimeType: "text/plain"},
		{FileId: "source-b", Name: "source-b.txt", MimeType: "text/plain"},
	}, documents)
	item := &DocumentVisualParse{
		Factory: workruntime.NewClientFactory(server.URL, "worker-key", 0),
	}

	_, err := item.Handle(t.Context(), &mockWctx{}, msg)
	require.NoError(t, err)
	require.Equal(t, int32(2), uploadCount.Load(), "each valid source should upload one manifest")
}

func TestFixPR14773DocumentVisualParseRejectsInvalidPDFRelationBeforePageImageUpload(t *testing.T) {
	documents := []*data.Document{
		fixPR14773DocumentVisualDocument(t, "source-pdf", "image-dangling-pdf", "image", "", 1, map[string]interface{}{
			"file_name":          "source-pdf.pdf",
			"mime_type":          "application/pdf",
			"figure_caption":     "Figure 3. Invalid PDF relation",
			"caption_block_uuid": "missing-caption-pdf",
		}),
	}

	var requestCount atomic.Int32
	var uploadCount atomic.Int32
	server := newFixPR14773DocumentVisualFileServerWithSource(
		t,
		&requestCount,
		&uploadCount,
		createMinimalPDFForRenderTest(t, 1),
		"application/pdf",
	)
	defer server.Close()

	msg := fixPR14773DocumentVisualParseMessage(t, []*data.Source{{
		FileId:   "source-pdf",
		Name:     "source-pdf.pdf",
		MimeType: "application/pdf",
	}}, documents)
	item := &DocumentVisualParse{
		Factory: workruntime.NewClientFactory(server.URL, "worker-key", 0),
	}

	_, err := item.Handle(t.Context(), &mockWctx{}, msg)
	assert.Error(t, err, "malformed PDF relation must fail at the relation boundary")
	assert.ErrorContains(t, err, "figure-caption relation")
	assert.Zero(t, requestCount.Load(), "relation validation must precede PDF download and rendering")
	assert.Zero(t, uploadCount.Load(), "invalid PDF input must not upload a rendered page image or manifest")
}

type fixPR14773DocumentVisualRelationCase struct {
	name      string
	documents []*data.Document
}

func fixPR14773MalformedDocumentVisualRelationCases(t *testing.T, sourceID string) []fixPR14773DocumentVisualRelationCase {
	t.Helper()
	const caption = "Figure 7. Canonical source caption"

	validPair := func(imageID, captionID string) []*data.Document {
		return fixPR14773ValidDocumentVisualRelationPair(t, sourceID, imageID, captionID, caption, 1)
	}
	image := func(imageID, captionID string, extra map[string]interface{}) *data.Document {
		metadata := map[string]interface{}{
			"figure_caption":     caption,
			"caption_block_uuid": captionID,
		}
		for key, value := range extra {
			metadata[key] = value
		}
		return fixPR14773DocumentVisualDocument(t, sourceID, imageID, "image", "", 1, metadata)
	}
	captionBlock := func(captionID, imageID, text, role string) *data.Document {
		return fixPR14773DocumentVisualDocument(t, sourceID, captionID, "text", text, 1, map[string]interface{}{
			"role":        role,
			"caption_for": imageID,
		})
	}

	validAndInvalid := validPair("image-valid", "caption-valid")
	validAndInvalid = append(validAndInvalid, image("image-dangling-after-valid", "missing-caption-after-valid", nil))

	return []fixPR14773DocumentVisualRelationCase{
		{
			name: "image carries caption_for",
			documents: []*data.Document{
				fixPR14773DocumentVisualDocument(t, sourceID, "image-with-caption-for", "image", "", 1, map[string]interface{}{
					"figure_caption":     caption,
					"caption_block_uuid": "caption-for-image-with-caption-for",
					"caption_for":        "some-image",
				}),
			},
		},
		{
			name: "text carries figure_caption",
			documents: []*data.Document{
				fixPR14773DocumentVisualDocument(t, sourceID, "text-with-figure-caption", "text", caption, 1, map[string]interface{}{
					"role":           "caption",
					"caption_for":    "some-image",
					"figure_caption": caption,
				}),
			},
		},
		{
			name: "list carries caption_block_uuid",
			documents: []*data.Document{
				fixPR14773DocumentVisualDocument(t, sourceID, "list-with-caption-block-uuid", "list", caption, 1, map[string]interface{}{
					"role":               "caption",
					"caption_for":        "some-image",
					"caption_block_uuid": "some-caption",
				}),
			},
		},
		{
			name: "standalone figure caption",
			documents: []*data.Document{
				fixPR14773DocumentVisualDocument(t, sourceID, "image-standalone", "image", "", 1, map[string]interface{}{
					"figure_caption": caption,
				}),
			},
		},
		{
			name: "image to caption one way",
			documents: []*data.Document{
				image("image-one-way", "caption-one-way", nil),
				fixPR14773DocumentVisualDocument(t, sourceID, "caption-one-way", "text", caption, 1, map[string]interface{}{
					"role": "caption",
				}),
			},
		},
		{
			name: "caption to image one way",
			documents: []*data.Document{
				fixPR14773DocumentVisualDocument(t, sourceID, "image-caption-one-way", "image", "", 1, nil),
				captionBlock("caption-to-image-one-way", "image-caption-one-way", caption, "caption"),
			},
		},
		{
			name: "dangling caption target",
			documents: []*data.Document{
				image("image-dangling", "missing-caption", nil),
			},
		},
		{
			name: "wrong caption role",
			documents: []*data.Document{
				image("image-wrong-role", "caption-wrong-role", nil),
				captionBlock("caption-wrong-role", "image-wrong-role", caption, "body"),
			},
		},
		{
			name: "caption text mismatch",
			documents: []*data.Document{
				image("image-text-mismatch", "caption-text-mismatch", nil),
				captionBlock("caption-text-mismatch", "image-text-mismatch", caption+" changed", "caption"),
			},
		},
		{
			name: "caption text case mismatch",
			documents: []*data.Document{
				image("image-text-case-mismatch", "caption-text-case-mismatch", nil),
				captionBlock("caption-text-case-mismatch", "image-text-case-mismatch", "figure 7. Canonical source caption", "caption"),
			},
		},
		{
			name: "wrong caption endpoint kind",
			documents: []*data.Document{
				image("image-wrong-kind", "caption-wrong-kind", nil),
				fixPR14773DocumentVisualDocument(t, sourceID, "caption-wrong-kind", "table", caption, 1, map[string]interface{}{
					"role":        "caption",
					"caption_for": "image-wrong-kind",
				}),
			},
		},
		{
			name: "duplicate relation endpoint",
			documents: []*data.Document{
				image("image-duplicate", "caption-duplicate", nil),
				captionBlock("caption-duplicate", "image-duplicate", caption, "caption"),
				captionBlock("caption-duplicate", "image-duplicate", caption, "caption"),
			},
		},
		{
			name: "many images to one caption",
			documents: []*data.Document{
				image("image-many-to-one-a", "caption-many-to-one", nil),
				image("image-many-to-one-b", "caption-many-to-one", nil),
				captionBlock("caption-many-to-one", "image-many-to-one-a", caption, "caption"),
			},
		},
		{
			name: "one image to many captions",
			documents: []*data.Document{
				image("image-one-to-many", "caption-one-to-many-a", nil),
				captionBlock("caption-one-to-many-a", "image-one-to-many", caption, "caption"),
				captionBlock("caption-one-to-many-b", "image-one-to-many", caption, "caption"),
			},
		},
		{
			name:      "valid plus invalid is atomic",
			documents: validAndInvalid,
		},
		{
			name: "cross page relation",
			documents: []*data.Document{
				image("image-cross-page", "caption-cross-page", nil),
				fixPR14773DocumentVisualDocument(t, sourceID, "caption-cross-page", "text", caption, 2, map[string]interface{}{
					"role":        "caption",
					"caption_for": "image-cross-page",
				}),
			},
		},
		{
			name: "one-sided source identity",
			documents: func() []*data.Document {
				documents := validPair("image-source-one-sided", "caption-source-one-sided")
				caption := documentFromProto(documents[1])
				delete(caption.Metadata, "raw_file_id")
				documents[1] = mustDocumentProto(t, caption)
				return documents
			}(),
		},
		{
			name: "missing relation page",
			documents: func() []*data.Document {
				documents := validPair("image-page-missing", "caption-page-missing")
				caption := documentFromProto(documents[1])
				delete(caption.Metadata, "page_number")
				documents[1] = mustDocumentProto(t, caption)
				return documents
			}(),
		},
		{
			name: "nil relation page",
			documents: func() []*data.Document {
				documents := validPair("image-page-nil", "caption-page-nil")
				image := documentFromProto(documents[0])
				image.Metadata["page_number"] = nil
				documents[0] = mustDocumentProto(t, image)
				return documents
			}(),
		},
		{
			name: "non-integer relation page",
			documents: func() []*data.Document {
				documents := validPair("image-page-type", "caption-page-type")
				caption := documentFromProto(documents[1])
				caption.Metadata["page_number"] = "1"
				documents[1] = mustDocumentProto(t, caption)
				return documents
			}(),
		},
		{
			name: "figure number mismatch",
			documents: []*data.Document{
				image("image-figure-number-mismatch", "caption-figure-number-mismatch", map[string]interface{}{
					"figure_no": "7",
				}),
				fixPR14773DocumentVisualDocument(t, sourceID, "caption-figure-number-mismatch", "text", caption, 1, map[string]interface{}{
					"role":        "caption",
					"caption_for": "image-figure-number-mismatch",
					"figure_no":   "8",
				}),
			},
		},
		{
			name: "blank present relation field",
			documents: []*data.Document{
				fixPR14773DocumentVisualDocument(t, sourceID, "image-blank-figure-caption", "image", "", 1, map[string]interface{}{
					"figure_caption":     "  ",
					"caption_block_uuid": "caption-blank-figure-caption",
				}),
				captionBlock("caption-blank-figure-caption", "image-blank-figure-caption", caption, "caption"),
			},
		},
		{
			name: "blank relation endpoint id",
			documents: []*data.Document{
				fixPR14773DocumentVisualDocument(t, sourceID, "image-blank-id", "image", "", 1, map[string]interface{}{
					"block_uuid":         " ",
					"figure_caption":     caption,
					"caption_block_uuid": "caption-for-blank-image-id",
				}),
				captionBlock("caption-for-blank-image-id", "image-blank-id", caption, "caption"),
			},
		},
		{
			name: "non string relation field",
			documents: []*data.Document{
				fixPR14773DocumentVisualDocument(t, sourceID, "image-non-string-caption-id", "image", "", 1, map[string]interface{}{
					"figure_caption":     caption,
					"caption_block_uuid": 17,
				}),
			},
		},
	}
}

func fixPR14773ValidDocumentVisualRelationPair(t *testing.T, sourceID, imageID, captionID, caption string, pageNumber int) []*data.Document {
	t.Helper()
	return []*data.Document{
		fixPR14773DocumentVisualDocument(t, sourceID, imageID, "image", "local image text", pageNumber, map[string]interface{}{
			"figure_caption":     caption,
			"caption_block_uuid": captionID,
		}),
		fixPR14773DocumentVisualDocument(t, sourceID, captionID, "text", caption, pageNumber, map[string]interface{}{
			"role":        "caption",
			"caption_for": imageID,
		}),
	}
}

func fixPR14773DocumentVisualDocument(t *testing.T, sourceID, blockID, blockType, content string, pageNumber int, extra map[string]interface{}) *data.Document {
	t.Helper()
	metadata := map[string]interface{}{
		"page_number": pageNumber,
		"block_uuid":  blockID,
		"raw_file_id": sourceID,
		"file_name":   sourceID + ".txt",
		"mime_type":   "text/plain",
	}
	for key, value := range extra {
		metadata[key] = value
	}
	return mustDocumentProto(t, Document{
		ID:       blockID,
		Type:     blockType,
		Content:  content,
		Metadata: metadata,
	})
}

func fixPR14773DocumentVisualParseMessage(t *testing.T, sources []*data.Source, documents []*data.Document) *mowl.MowlMessage {
	t.Helper()
	disabled := false
	enabled := true
	payload, err := json.Marshal(documentVisualParseInput{
		Sources:              sources,
		Documents:            documents,
		Profile:              documentVisualStandardRAGProfile,
		RequirePageImages:    &disabled,
		RequireObjectImages:  &disabled,
		RequireVisualContext: &disabled,
		WriteManifestFile:    &enabled,
	})
	require.NoError(t, err)
	return &mowl.MowlMessage{Data: string(payload)}
}

func newFixPR14773DocumentVisualFileServer(t *testing.T, requestCount, uploadCount *atomic.Int32) *httptest.Server {
	t.Helper()
	return newFixPR14773DocumentVisualFileServerWithSource(
		t,
		requestCount,
		uploadCount,
		[]byte("source placeholder"),
		"text/plain",
	)
}

func newFixPR14773DocumentVisualFileServerWithSource(t *testing.T, requestCount, uploadCount *atomic.Int32, source []byte, contentType string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		switch {
		case r.Method == http.MethodGet:
			w.Header().Set("Content-Type", contentType)
			_, _ = w.Write(source)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/workspaces/ws-test/files":
			uploadID := uploadCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"file_id":       "uploaded-manifest-" + toString(uploadID),
				"original_name": "manifest.json",
				"size":          1,
			})
		default:
			http.NotFound(w, r)
		}
	}))
}
