package session

import (
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
)

func TestSegmentFromImageVectorRowDoesNotUseVectorContentAsSegmentContent(t *testing.T) {
	content := sql.NullString{
		String: "20C114830.pdf\n<table><tr><td>whole page text</td></tr></table>",
		Valid:  true,
	}
	meta := sql.NullString{
		String: `{"chunk_id":"document_visual_image:visual_object:5121ca5b-b04a-45a8-bc23-99ca6637fa70","image_file_id":"image-file-1","page_image_file_id":"page-image-1","ocr_text":"local ocr","caption":"local caption","bbox":[1,2,3,4]}`,
		Valid:  true,
	}
	level := sql.NullString{String: kbSegmentLevelChunk, Valid: true}

	seg, err := segmentFromVectorRow("row-1", content, meta, level, sql.NullInt64{}, sql.NullInt64{Int64: 9, Valid: true}, 9, segmentVectorRowKindImage)
	if err != nil {
		t.Fatalf("segmentFromVectorRow: %v", err)
	}
	if seg.Content != nil {
		t.Fatalf("image segment content = %q, want nil", *seg.Content)
	}
	if seg.OCRText == nil || *seg.OCRText != "local ocr" {
		t.Fatalf("image segment ocr_text = %+v", seg.OCRText)
	}
	if seg.ImageDescription == nil || *seg.ImageDescription != "local caption" {
		t.Fatalf("image segment image_description = %+v", seg.ImageDescription)
	}
	if seg.ImageFileID == nil || *seg.ImageFileID != "image-file-1" {
		t.Fatalf("image segment image_file_id = %+v", seg.ImageFileID)
	}
	if seg.PageImageFileID == nil || *seg.PageImageFileID != "page-image-1" {
		t.Fatalf("image segment page_image_file_id = %+v", seg.PageImageFileID)
	}
	if seg.ChunkID == nil || *seg.ChunkID != "document_visual_image:visual_object:5121ca5b-b04a-45a8-bc23-99ca6637fa70" {
		t.Fatalf("image segment chunk_id = %+v", seg.ChunkID)
	}
}

func TestSegmentImageVectorMetadataPreservesVisualObjectScope(t *testing.T) {
	imageFileID := "image-file-1"
	pageImageFileID := "page-image-1"
	meta := json.RawMessage(`{"scope":"visual_object","object_id":"obj-1","object_kind":"drawing_view","page_number":1}`)
	seg := kbSegmentRecord{
		VersionID:       "seg-v1",
		ModelID:         7,
		SourceID:        "source-file-1",
		KBFileID:        "kb-file",
		IndexVersion:    1,
		Level:           kbSegmentLevelChunk,
		ChunkID:         stringPtr("document_visual_image:visual_object:obj-1"),
		IdentityKey:     imageSegmentIdentityKey("document_visual_image:visual_object:obj-1"),
		ImageFileID:     &imageFileID,
		PageImageFileID: &pageImageFileID,
		Metadata:        meta,
		Enabled:         true,
	}

	metadata, err := segmentImageVectorMetadata(seg, kbVectorBinding{
		ImageVectorTable:        "kb_image_idx",
		ImageEmbeddingModel:     "image-embed",
		ImageEmbeddingBackendID: "5",
		ImageEmbeddingDimension: 4,
		ImagePreprocessVersion:  "image-v1",
		ImageDistanceMetric:     "cosine",
	}, imageFileID, 1, nil)
	if err != nil {
		t.Fatalf("segmentImageVectorMetadata: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(metadata), &got); err != nil {
		t.Fatalf("unmarshal metadata: %v", err)
	}
	if got["scope"] != "visual_object" || got["object_id"] != "obj-1" || got["object_kind"] != "drawing_view" {
		t.Fatalf("visual identity metadata was not preserved: %v", got)
	}
}

func TestPrepareNextSegmentVersionSeparatesTextAndImageVectorIdentities(t *testing.T) {
	chunkIndex := int64(0)
	level := sql.NullString{String: kbSegmentLevelChunk, Valid: true}
	textSeg, err := segmentFromVectorRow(
		"text-row-0",
		sql.NullString{String: "text content", Valid: true},
		sql.NullString{String: `{"chunk_id":"text-chunk-0"}`, Valid: true},
		level,
		sql.NullInt64{Int64: chunkIndex, Valid: true},
		sql.NullInt64{Int64: 1, Valid: true},
		1,
		segmentVectorRowKindText,
	)
	if err != nil {
		t.Fatalf("text segmentFromVectorRow: %v", err)
	}
	imageSeg, err := segmentFromVectorRow(
		"image-row-0",
		sql.NullString{String: "page text", Valid: true},
		sql.NullString{String: `{"segment_type":"image","chunk_id":"image-chunk-0","image_file_id":"image-file-0","page_image_file_id":"page-image-0","image_description":"image desc"}`, Valid: true},
		level,
		sql.NullInt64{Int64: chunkIndex, Valid: true},
		sql.NullInt64{Int64: 1, Valid: true},
		1,
		segmentVectorRowKindImage,
	)
	if err != nil {
		t.Fatalf("image segmentFromVectorRow: %v", err)
	}
	segments := []kbSegmentRecord{textSeg, imageSeg}
	record := KnowledgeBaseSourceRecord{
		ModelID:  77,
		SourceID: "source-file-1",
		KBFileID: stringPtr("kb-file-1"),
	}
	versionID := stableID("kb-segver", record.SourceID, int64(1), kbSegmentSourceExternal)
	if err := prepareNextSegmentVersion(record, versionID, 1, segments); err != nil {
		t.Fatalf("prepareNextSegmentVersion: %v", err)
	}
	if segments[0].IdentityKey != "idx:0" {
		t.Fatalf("text identity = %q, want idx:0", segments[0].IdentityKey)
	}
	if segments[1].IdentityKey != "image:idx:0" {
		t.Fatalf("image identity = %q, want image:idx:0", segments[1].IdentityKey)
	}
	if segments[0].SegmentID == segments[1].SegmentID {
		t.Fatalf("text and image segment ids should differ: %q", segments[0].SegmentID)
	}
	segmentType, _, _ := documentSegmentCanonicalMetadata(segments[1].Metadata)
	if segmentType != "image" {
		t.Fatalf("image vector segment type = %q, want image", segmentType)
	}
}

func TestPrepareNextSegmentVersionCanonicalizesImageTypeWithoutImageFile(t *testing.T) {
	content := "ocr text"
	chunkIndex := int64(2)
	current := []kbSegmentRecord{{
		SegmentID:  "segment-current",
		Level:      kbSegmentLevelChunk,
		ChunkIndex: &chunkIndex,
		Content:    &content,
		Metadata:   json.RawMessage(`{"segment_type":"image","source_block_type":"IMAGE","volume_id":"2"}`),
		Enabled:    true,
	}}
	next := cloneSegmentRecords(current)
	record := KnowledgeBaseSourceRecord{
		ModelID:  3,
		SourceID: "source-file-1",
		KBFileID: stringPtr("kb-file-1"),
	}

	if err := prepareNextSegmentVersion(record, "version-next", 2, next); err != nil {
		t.Fatalf("prepareNextSegmentVersion: %v", err)
	}
	segmentType, _, _ := documentSegmentCanonicalMetadata(next[0].Metadata)
	if segmentType != "text" {
		t.Fatalf("segment_type = %q, want text; metadata=%s", segmentType, next[0].Metadata)
	}
}

func TestSemanticModelDocumentSegmentsCanonicalizesImageTypeByFileIdentity(t *testing.T) {
	imageFileID := "image-file-1"
	segments := semanticModelDocumentSegments([]SemanticModelSegment{
		{SegmentID: "text-image-origin", Metadata: json.RawMessage(`{"segment_type":"image"}`)},
		{SegmentID: "image-vector", ImageFileID: &imageFileID, Metadata: json.RawMessage(`{"segment_type":"image"}`)},
		{SegmentID: "table", Metadata: json.RawMessage(`{"segment_type":"table"}`)},
	})

	if segments[0].SegmentType != "text" {
		t.Fatalf("image-derived text segment type = %q, want text", segments[0].SegmentType)
	}
	if segments[1].SegmentType != "image" {
		t.Fatalf("image vector segment type = %q, want image", segments[1].SegmentType)
	}
	if segments[2].SegmentType != "table" {
		t.Fatalf("table segment type = %q, want table", segments[2].SegmentType)
	}
}

func TestMutateClonedSegmentByCurrentIDMatchesCurrentRecordIDs(t *testing.T) {
	originalContent := "before"
	nextContent := "after"
	current := []kbSegmentRecord{{
		SegmentID: "kb-segment-current",
		Content:   &originalContent,
	}}
	cloned := cloneSegmentRecords(current)
	if cloned[0].SegmentID != "" {
		t.Fatalf("cloned segment id = %q, want empty before next version is prepared", cloned[0].SegmentID)
	}

	matched := mutateClonedSegmentByCurrentID(current, cloned, "kb-segment-current", func(seg *kbSegmentRecord) {
		seg.Content = &nextContent
	})

	if !matched {
		t.Fatalf("mutateClonedSegmentByCurrentID did not match current segment id")
	}
	if cloned[0].Content == nil || *cloned[0].Content != "after" {
		t.Fatalf("cloned content = %+v", cloned[0].Content)
	}
}

func TestSegmentVectorRowIDsStayWithinVectorTableIDLimit(t *testing.T) {
	chunkID := "document_visual_image:visual_object:5121ca5b-b04a-45a8-bc23-99ca6637fa70"
	imageFileID := "7403558f-e631-47db-8068-7eb06c275e89"
	seg := kbSegmentRecord{
		KBFileID:     "c1de9776-8cff-4e88-b791-29b73e9729b4",
		IndexVersion: 1783054297193,
		Level:        kbSegmentLevelChunk,
		ChunkID:      &chunkID,
		IdentityKey:  chunkIdentityKey(nil, &chunkID),
	}

	textRowID := segmentVectorRowID(seg)
	imageRowID := segmentImageVectorRowID(seg, imageFileID)
	for name, id := range map[string]string{"text": textRowID, "image": imageRowID} {
		if len(id) > 128 {
			t.Fatalf("%s row id length = %d, want <= 128: %s", name, len(id), id)
		}
		if strings.Contains(id, chunkID) || strings.Contains(id, imageFileID) {
			t.Fatalf("%s row id leaked long identity: %s", name, id)
		}
	}
	if !strings.HasPrefix(textRowID, "kbsegrow-") {
		t.Fatalf("text row id = %q, want kbsegrow prefix", textRowID)
	}
	if !strings.HasPrefix(imageRowID, "kbimgsegrow-") {
		t.Fatalf("image row id = %q, want kbimgsegrow prefix", imageRowID)
	}
	if textRowID == imageRowID {
		t.Fatalf("text and image row ids should differ: %q", textRowID)
	}
}
