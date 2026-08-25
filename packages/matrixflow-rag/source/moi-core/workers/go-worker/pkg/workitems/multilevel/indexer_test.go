package multilevel

import (
	"strings"
	"testing"
)

func TestGenerateEntries_EmptyDocument(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	entries := indexer.GenerateEntries(DocumentInput{
		FileID:   "f1",
		FileName: "test.pdf",
		Chunks:   nil,
	}, 42)

	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty document, got %d", len(entries))
	}
}

func TestGenerateEntries_SingleChunk(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	doc := DocumentInput{
		FileID:   "f1",
		FileName: "test.pdf",
		Chunks: []ChunkInput{
			{Content: "Hello world", ChunkIndex: 0, FileID: "f1", Metadata: map[string]any{"key": "val"}},
		},
	}
	entries := indexer.GenerateEntries(doc, 100)

	// Expect: 1 doc + 1 section + 1 chunk = 3 entries
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Document level
	docEntry := entries[0]
	if docEntry.Level != IndexLevelDocument {
		t.Errorf("expected doc level, got %s", docEntry.Level)
	}
	if docEntry.DocID != "doc_f1" {
		t.Errorf("expected DocID doc_f1, got %s", docEntry.DocID)
	}
	if docEntry.SectionID != "" {
		t.Errorf("expected empty SectionID for doc level, got %s", docEntry.SectionID)
	}
	if docEntry.Content != "Hello world" {
		t.Errorf("expected doc content 'Hello world', got %q", docEntry.Content)
	}
	if docEntry.IndexVersion != 100 {
		t.Errorf("expected IndexVersion 100, got %d", docEntry.IndexVersion)
	}

	// Section level
	secEntry := entries[1]
	if secEntry.Level != IndexLevelSection {
		t.Errorf("expected section level, got %s", secEntry.Level)
	}
	if secEntry.SectionID != "sec_f1_0" {
		t.Errorf("expected SectionID sec_f1_0, got %s", secEntry.SectionID)
	}
	if secEntry.DocID != "doc_f1" {
		t.Errorf("expected DocID doc_f1, got %s", secEntry.DocID)
	}

	// Chunk level
	chunkEntry := entries[2]
	if chunkEntry.Level != IndexLevelChunk {
		t.Errorf("expected chunk level, got %s", chunkEntry.Level)
	}
	if chunkEntry.DocID != "doc_f1" {
		t.Errorf("expected DocID doc_f1, got %s", chunkEntry.DocID)
	}
	if chunkEntry.Content != "Hello world" {
		t.Errorf("expected chunk content 'Hello world', got %q", chunkEntry.Content)
	}
	if chunkEntry.IndexVersion != 100 {
		t.Errorf("expected IndexVersion 100, got %d", chunkEntry.IndexVersion)
	}
	// Verify original metadata is preserved
	if v, ok := chunkEntry.Metadata["key"]; !ok || v != "val" {
		t.Errorf("expected chunk metadata key=val, got %v", chunkEntry.Metadata)
	}
	// Verify chunk_id
	if chunkEntry.Metadata["chunk_id"] != "f1_chunk_0" {
		t.Errorf("expected chunk_id=f1_chunk_0, got %v", chunkEntry.Metadata["chunk_id"])
	}
}

func TestGenerateEntries_MultiChunk_SectionGrouping(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	chunks := make([]ChunkInput, 12)
	for i := range chunks {
		chunks[i] = ChunkInput{
			Content:    strings.Repeat("x", 100),
			ChunkIndex: i,
			FileID:     "f2",
		}
	}
	doc := DocumentInput{FileID: "f2", FileName: "big.pdf", Chunks: chunks}
	entries := indexer.GenerateEntries(doc, 7)

	// 12 chunks → 4 sections (3+3+3+3) → 1 doc + 4 sections + 12 chunks = 17
	docCount, secCount, chunkCount := countLevels(entries)
	if docCount != 1 {
		t.Errorf("expected 1 doc entry, got %d", docCount)
	}
	if secCount != 4 {
		t.Errorf("expected 4 section entries, got %d", secCount)
	}
	if chunkCount != 12 {
		t.Errorf("expected 12 chunk entries, got %d", chunkCount)
	}

	// Verify section IDs
	sectionIDs := make(map[string]bool)
	for _, e := range entries {
		if e.Level == IndexLevelSection {
			sectionIDs[e.SectionID] = true
		}
	}
	for _, expected := range []string{"sec_f2_0", "sec_f2_1", "sec_f2_2", "sec_f2_3"} {
		if !sectionIDs[expected] {
			t.Errorf("missing expected section ID %s", expected)
		}
	}
}

func TestGenerateEntries_SectionGroupingRespectsContentBudget(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	indexer.SectionSize = 5
	indexer.MaxSectionContentBytes = 2500
	chunks := make([]ChunkInput, 3)
	for i := range chunks {
		chunks[i] = ChunkInput{
			Content:    strings.Repeat("a", 1200),
			ChunkIndex: i,
			FileID:     "f-budget",
		}
	}
	doc := DocumentInput{FileID: "f-budget", FileName: "budget.pdf", Chunks: chunks}
	entries := indexer.GenerateEntries(doc, 9)

	sections := filterByLevel(entries, IndexLevelSection)
	if len(sections) != 2 {
		t.Fatalf("expected 2 budgeted sections, got %d", len(sections))
	}
	if len(sections[0].Content) != 2401 {
		t.Fatalf("expected first section to contain two chunks plus separator, got len=%d", len(sections[0].Content))
	}
	if len(sections[1].Content) != 1200 {
		t.Fatalf("expected second section to contain one chunk, got len=%d", len(sections[1].Content))
	}
	if sections[0].Metadata["chunk_start"] != 0 || sections[0].Metadata["chunk_end"] != 1 {
		t.Fatalf("unexpected first section chunk range: %+v", sections[0].Metadata)
	}
	if sections[1].Metadata["chunk_start"] != 2 || sections[1].Metadata["chunk_end"] != 2 {
		t.Fatalf("unexpected second section chunk range: %+v", sections[1].Metadata)
	}

	_, _, chunkCount := countLevels(entries)
	if chunkCount != 3 {
		t.Fatalf("expected chunk-level entries to remain unchanged, got %d", chunkCount)
	}
}

func TestGenerateEntries_DefaultSectionGroupingKeepsHeadroom(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	chunks := make([]ChunkInput, 5)
	for i := range chunks {
		chunks[i] = ChunkInput{
			Content:    strings.Repeat("a", 800),
			ChunkIndex: i,
			FileID:     "f-default-budget",
		}
	}
	doc := DocumentInput{FileID: "f-default-budget", FileName: "default.pdf", Chunks: chunks}
	entries := indexer.GenerateEntries(doc, 11)

	sections := filterByLevel(entries, IndexLevelSection)
	if len(sections) != 2 {
		t.Fatalf("expected default budget to split 5x800 chunks into 2 sections, got %d", len(sections))
	}
	if len(sections[0].Content) >= 4000 {
		t.Fatalf("first section is still near old 5x800 grouping, len=%d", len(sections[0].Content))
	}
	if len(sections[0].Content) > defaultSectionContentMaxBytes {
		t.Fatalf("first section len=%d exceeds default budget=%d", len(sections[0].Content), defaultSectionContentMaxBytes)
	}
	if sections[0].Metadata["chunk_start"] != 0 || sections[0].Metadata["chunk_end"] != 2 {
		t.Fatalf("unexpected first section chunk range: %+v", sections[0].Metadata)
	}
	if sections[1].Metadata["chunk_start"] != 3 || sections[1].Metadata["chunk_end"] != 4 {
		t.Fatalf("unexpected second section chunk range: %+v", sections[1].Metadata)
	}
}

func TestGenerateEntries_SectionContentCapDoesNotRewriteChunkEntry(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	indexer.MaxSectionContentBytes = 10
	doc := DocumentInput{
		FileID:   "f-long-chunk",
		FileName: "long.pdf",
		Chunks: []ChunkInput{
			{Content: strings.Repeat("中", 10), ChunkIndex: 0, FileID: "f-long-chunk"},
		},
	}
	entries := indexer.GenerateEntries(doc, 3)

	sections := filterByLevel(entries, IndexLevelSection)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if len(sections[0].Content) > indexer.MaxSectionContentBytes {
		t.Fatalf("section content len=%d exceeds max=%d", len(sections[0].Content), indexer.MaxSectionContentBytes)
	}
	if sections[0].Metadata["section_content_truncated"] != true {
		t.Fatalf("expected truncated metadata, got %+v", sections[0].Metadata)
	}

	chunks := filterByLevel(entries, IndexLevelChunk)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk entry, got %d", len(chunks))
	}
	if chunks[0].Content != strings.Repeat("中", 10) {
		t.Fatalf("chunk content was rewritten: %q", chunks[0].Content)
	}
}

func TestGenerateEntries_DocSummaryTruncation(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	// Create chunks whose total content exceeds 2000 chars
	chunks := make([]ChunkInput, 5)
	for i := range chunks {
		chunks[i] = ChunkInput{
			Content:    strings.Repeat("a", 600),
			ChunkIndex: i,
			FileID:     "f3",
		}
	}
	doc := DocumentInput{FileID: "f3", FileName: "long.pdf", Chunks: chunks}
	entries := indexer.GenerateEntries(doc, 1)

	docEntry := entries[0]
	if docEntry.Level != IndexLevelDocument {
		t.Fatalf("expected doc level first")
	}
	if len(docEntry.Content) > defaultDocSummaryMaxChars {
		t.Errorf("doc summary length %d exceeds max %d", len(docEntry.Content), defaultDocSummaryMaxChars)
	}
	if len(docEntry.Content) == 0 {
		t.Error("doc summary should not be empty")
	}
}

func TestGenerateEntries_AllShareSameIndexVersion(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	doc := DocumentInput{
		FileID:   "f4",
		FileName: "versioned.pdf",
		Chunks: []ChunkInput{
			{Content: "chunk1", ChunkIndex: 0, FileID: "f4"},
			{Content: "chunk2", ChunkIndex: 1, FileID: "f4"},
			{Content: "chunk3", ChunkIndex: 2, FileID: "f4"},
		},
	}
	version := int64(999)
	entries := indexer.GenerateEntries(doc, version)

	for i, e := range entries {
		if e.IndexVersion != version {
			t.Errorf("entry %d (level=%s): expected IndexVersion %d, got %d", i, e.Level, version, e.IndexVersion)
		}
	}
}

func TestGenerateEntries_DocIDAndSectionIDFormat(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	doc := DocumentInput{
		FileID:   "abc-123",
		FileName: "doc.pdf",
		Chunks: []ChunkInput{
			{Content: "c1", ChunkIndex: 0, FileID: "abc-123"},
			{Content: "c2", ChunkIndex: 1, FileID: "abc-123"},
			{Content: "c3", ChunkIndex: 2, FileID: "abc-123"},
			{Content: "c4", ChunkIndex: 3, FileID: "abc-123"},
			{Content: "c5", ChunkIndex: 4, FileID: "abc-123"},
			{Content: "c6", ChunkIndex: 5, FileID: "abc-123"},
		},
	}
	entries := indexer.GenerateEntries(doc, 1)

	for _, e := range entries {
		if e.DocID != "doc_abc-123" {
			t.Errorf("expected DocID doc_abc-123, got %s", e.DocID)
		}
	}

	// 6 chunks → 2 sections (5+1)
	sections := filterByLevel(entries, IndexLevelSection)
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if sections[0].SectionID != "sec_abc-123_0" {
		t.Errorf("expected sec_abc-123_0, got %s", sections[0].SectionID)
	}
	if sections[1].SectionID != "sec_abc-123_1" {
		t.Errorf("expected sec_abc-123_1, got %s", sections[1].SectionID)
	}
}

func TestGenerateEntries_SectionContentIsConcatenation(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	doc := DocumentInput{
		FileID:   "f5",
		FileName: "concat.pdf",
		Chunks: []ChunkInput{
			{Content: "alpha", ChunkIndex: 0, FileID: "f5"},
			{Content: "beta", ChunkIndex: 1, FileID: "f5"},
			{Content: "gamma", ChunkIndex: 2, FileID: "f5"},
		},
	}
	entries := indexer.GenerateEntries(doc, 1)

	sections := filterByLevel(entries, IndexLevelSection)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	expected := "alpha beta gamma"
	if sections[0].Content != expected {
		t.Errorf("expected section content %q, got %q", expected, sections[0].Content)
	}
}

func TestGenerateEntries_ChunkMetadataPreserved(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	doc := DocumentInput{
		FileID:   "f6",
		FileName: "meta.pdf",
		Chunks: []ChunkInput{
			{
				Content:    "data",
				ChunkIndex: 0,
				FileID:     "f6",
				Metadata:   map[string]any{"custom_key": "custom_val", "num": 42},
			},
		},
	}
	entries := indexer.GenerateEntries(doc, 1)

	chunks := filterByLevel(entries, IndexLevelChunk)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	meta := chunks[0].Metadata
	if meta["custom_key"] != "custom_val" {
		t.Errorf("expected custom_key=custom_val, got %v", meta["custom_key"])
	}
	if meta["num"] != 42 {
		t.Errorf("expected num=42, got %v", meta["num"])
	}
	// Also verify injected metadata
	if meta["file_id"] != "f6" {
		t.Errorf("expected file_id=f6, got %v", meta["file_id"])
	}
	if meta["chunk_index"] != 0 {
		t.Errorf("expected chunk_index=0, got %v", meta["chunk_index"])
	}
	// Verify chunk_id
	if meta["chunk_id"] != "f6_chunk_0" {
		t.Errorf("expected chunk_id=f6_chunk_0, got %v", meta["chunk_id"])
	}
}

func TestGenerateEntries_ChunkIDGeneration(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	doc := DocumentInput{
		FileID:   "cid1",
		FileName: "resume.pdf",
		Chunks: []ChunkInput{
			{Content: "chunk0", ChunkIndex: 0, FileID: "cid1"},
			{Content: "chunk1", ChunkIndex: 1, FileID: "cid1"},
		},
	}
	entries := indexer.GenerateEntries(doc, 1)

	// Document level: chunk_id == docID
	docEntries := filterByLevel(entries, IndexLevelDocument)
	if len(docEntries) != 1 {
		t.Fatalf("expected 1 doc entry, got %d", len(docEntries))
	}
	if docEntries[0].Metadata["chunk_id"] != "doc_cid1" {
		t.Errorf("doc chunk_id: expected doc_cid1, got %v", docEntries[0].Metadata["chunk_id"])
	}

	// Section level: chunk_id == sectionID
	secEntries := filterByLevel(entries, IndexLevelSection)
	if len(secEntries) != 1 {
		t.Fatalf("expected 1 section entry, got %d", len(secEntries))
	}
	if secEntries[0].Metadata["chunk_id"] != "sec_cid1_0" {
		t.Errorf("section chunk_id: expected sec_cid1_0, got %v", secEntries[0].Metadata["chunk_id"])
	}

	// Chunk level: chunk_id == "<fileID>_chunk_<idx>"
	chunkEntries := filterByLevel(entries, IndexLevelChunk)
	if len(chunkEntries) != 2 {
		t.Fatalf("expected 2 chunk entries, got %d", len(chunkEntries))
	}
	if chunkEntries[0].Metadata["chunk_id"] != "cid1_chunk_0" {
		t.Errorf("chunk[0] chunk_id: expected cid1_chunk_0, got %v", chunkEntries[0].Metadata["chunk_id"])
	}
	if chunkEntries[1].Metadata["chunk_id"] != "cid1_chunk_1" {
		t.Errorf("chunk[1] chunk_id: expected cid1_chunk_1, got %v", chunkEntries[1].Metadata["chunk_id"])
	}
}

func TestGenerateEntries_ChunkIDGenerationUsesFileGlobalOrder(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	doc := DocumentInput{
		FileID:   "cid2",
		FileName: "investor-record.pdf",
		Chunks: []ChunkInput{
			{
				Content:    "first parser document first chunk",
				ChunkIndex: 0,
				FileID:     "cid2",
				Metadata:   map[string]any{"parent_index": 0, "chunk_index": 0},
			},
			{
				Content:    "second parser document first chunk",
				ChunkIndex: 0,
				FileID:     "cid2",
				Metadata:   map[string]any{"parent_index": 1, "chunk_index": 0},
			},
		},
	}
	entries := indexer.GenerateEntries(doc, 1)

	chunks := filterByLevel(entries, IndexLevelChunk)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunk entries, got %d", len(chunks))
	}
	if chunks[0].Metadata["chunk_index"] != 0 {
		t.Errorf("chunk[0] chunk_index: expected 0, got %v", chunks[0].Metadata["chunk_index"])
	}
	if chunks[0].Metadata["chunk_id"] != "cid2_chunk_0" {
		t.Errorf("chunk[0] chunk_id: expected cid2_chunk_0, got %v", chunks[0].Metadata["chunk_id"])
	}
	if chunks[0].Metadata["chunk_index_scope"] != "file" {
		t.Errorf("chunk[0] chunk_index_scope: expected file, got %v", chunks[0].Metadata["chunk_index_scope"])
	}
	if chunks[1].Metadata["chunk_index"] != 1 {
		t.Errorf("chunk[1] chunk_index: expected 1, got %v", chunks[1].Metadata["chunk_index"])
	}
	if chunks[1].Metadata["chunk_id"] != "cid2_chunk_1" {
		t.Errorf("chunk[1] chunk_id: expected cid2_chunk_1, got %v", chunks[1].Metadata["chunk_id"])
	}
	if chunks[1].Metadata["parent_index"] != 1 {
		t.Errorf("chunk[1] parent_index should be preserved, got %v", chunks[1].Metadata["parent_index"])
	}
}

func TestGenerateEntries_SectionUsesFileScopedChunkRange(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	indexer.SectionSize = 5
	doc := DocumentInput{
		FileID:   "cid3",
		FileName: "table.pdf",
		Chunks: []ChunkInput{
			{Content: "parent 0 chunk 0", ChunkIndex: 0, FileID: "cid3", Metadata: map[string]any{"parent_index": 0}},
			{Content: "parent 0 chunk 1", ChunkIndex: 1, FileID: "cid3", Metadata: map[string]any{"parent_index": 0}},
			{Content: "parent 1 chunk 0", ChunkIndex: 0, FileID: "cid3", Metadata: map[string]any{"parent_index": 1}},
			{Content: "parent 1 chunk 1", ChunkIndex: 1, FileID: "cid3", Metadata: map[string]any{"parent_index": 1}},
		},
	}
	entries := indexer.GenerateEntries(doc, 1)

	sections := filterByLevel(entries, IndexLevelSection)
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	meta := sections[0].Metadata
	if meta["chunk_start"] != 0 || meta["chunk_end"] != 3 {
		t.Fatalf("unexpected section chunk range: %+v", meta)
	}
	if meta["chunk_index_scope"] != "file" {
		t.Fatalf("unexpected section chunk_index_scope: %+v", meta)
	}
}

func TestNewMultiLevelIndexer_Defaults(t *testing.T) {
	indexer := NewMultiLevelIndexer()
	if indexer.MaxDocSummaryChars != defaultDocSummaryMaxChars {
		t.Errorf("expected MaxDocSummaryChars=%d, got %d", defaultDocSummaryMaxChars, indexer.MaxDocSummaryChars)
	}
	if indexer.SectionSize != defaultSectionSize {
		t.Errorf("expected SectionSize=%d, got %d", defaultSectionSize, indexer.SectionSize)
	}
	if indexer.MaxSectionContentBytes != defaultSectionContentMaxBytes {
		t.Errorf("expected MaxSectionContentBytes=%d, got %d", defaultSectionContentMaxBytes, indexer.MaxSectionContentBytes)
	}
}

// --- helpers ---

func countLevels(entries []IndexEntry) (doc, section, chunk int) {
	for _, e := range entries {
		switch e.Level {
		case IndexLevelDocument:
			doc++
		case IndexLevelSection:
			section++
		case IndexLevelChunk:
			chunk++
		}
	}
	return
}

func filterByLevel(entries []IndexEntry, level IndexLevel) []IndexEntry {
	var out []IndexEntry
	for _, e := range entries {
		if e.Level == level {
			out = append(out, e)
		}
	}
	return out
}
