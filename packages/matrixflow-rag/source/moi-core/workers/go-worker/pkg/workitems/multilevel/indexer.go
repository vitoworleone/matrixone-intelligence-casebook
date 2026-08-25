package multilevel

import (
	"fmt"
	"strings"

)

const (
	// defaultDocSummaryMaxChars keeps document-level summaries bounded.
	defaultDocSummaryMaxChars = 2000
	// defaultSectionSize is the maximum number of chunks grouped into one section.
	defaultSectionSize = 3
	// defaultSectionContentMaxBytes keeps section-level embedding inputs below
	// the old 5*800 default chunk grouping with enough margin for embedding
	// providers that enforce a 4k input window.
	defaultSectionContentMaxBytes = 3000
)

// MultiLevelIndexer generates Document/Section/Chunk three-level index entries
// from a document's chunks.
type MultiLevelIndexer struct {
	MaxDocSummaryChars     int // default from defaultDocSummaryMaxChars (2000)
	SectionSize            int // maximum chunks per section, default 3
	MaxSectionContentBytes int // maximum bytes per section embedding input, default 3000
}

// NewMultiLevelIndexer creates a new indexer with default settings.
func NewMultiLevelIndexer() *MultiLevelIndexer {
	return &MultiLevelIndexer{
		MaxDocSummaryChars:     defaultDocSummaryMaxChars,
		SectionSize:            defaultSectionSize,
		MaxSectionContentBytes: defaultSectionContentMaxBytes,
	}
}

// GenerateEntries produces index entries at all three levels for a document.
// The returned entries share the same IndexVersion for consistency.
//
// Document level: concatenates first N chunks up to MaxDocSummaryChars,
//
//	truncated at the limit. Uses docID = "doc_<fileID>".
//
// Section level: groups consecutive chunks by SectionSize and content budget,
//
//	generates section_id = "sec_<fileID>_<sectionIdx>".
//	Section content = concatenation of chunk contents in that section.
//
// Chunk level: passes through existing chunks unchanged, adding level/docID/indexVersion.
func (m *MultiLevelIndexer) GenerateEntries(doc DocumentInput, indexVersion int64) []IndexEntry {
	if len(doc.Chunks) == 0 {
		return nil
	}

	docID := fmt.Sprintf("doc_%s", doc.FileID)
	sectionSize := m.SectionSize
	if sectionSize <= 0 {
		sectionSize = defaultSectionSize
	}
	maxChars := m.MaxDocSummaryChars
	if maxChars <= 0 {
		maxChars = defaultDocSummaryMaxChars
	}
	maxSectionBytes := m.MaxSectionContentBytes
	if maxSectionBytes <= 0 {
		maxSectionBytes = defaultSectionContentMaxBytes
	}

	var entries []IndexEntry

	// --- Document level ---
	docContent := m.buildDocSummary(doc.Chunks, maxChars)
	entries = append(entries, IndexEntry{
		Level:     IndexLevelDocument,
		DocID:     docID,
		SectionID: "",
		Content:   docContent,
		Metadata: map[string]any{
			"file_id":       doc.FileID,
			"file_name":     doc.FileName,
			"level":         string(IndexLevelDocument),
			"doc_id":        docID,
			"chunk_id":      docID,
			"section_id":    "",
			"index_version": indexVersion,
		},
		IndexVersion: indexVersion,
	})

	// --- Section level ---
	sectionGroups := m.buildSectionGroups(doc.Chunks, sectionSize, maxSectionBytes)
	for sectionIdx, group := range sectionGroups {
		sectionID := fmt.Sprintf("sec_%s_%d", doc.FileID, sectionIdx)

		sectionContent, truncated := m.buildSectionContent(doc.Chunks[group.Start:group.End], maxSectionBytes)
		meta := map[string]any{
			"file_id":           doc.FileID,
			"file_name":         doc.FileName,
			"section_idx":       sectionIdx,
			"chunk_start":       group.Start,
			"chunk_end":         group.End - 1,
			"chunk_index_scope": "file",
			"level":             string(IndexLevelSection),
			"doc_id":            docID,
			"chunk_id":          sectionID,
			"section_id":        sectionID,
			"index_version":     indexVersion,
		}
		if truncated {
			meta["section_content_truncated"] = true
			meta["section_content_max_bytes"] = maxSectionBytes
		}
		entries = append(entries, IndexEntry{
			Level:        IndexLevelSection,
			DocID:        docID,
			SectionID:    sectionID,
			Content:      sectionContent,
			Metadata:     meta,
			IndexVersion: indexVersion,
		})
	}

	// --- Chunk level ---
	for globalChunkIndex, chunk := range doc.Chunks {
		meta := make(map[string]any)
		for k, v := range chunk.Metadata {
			meta[k] = v
		}
		meta["file_id"] = chunk.FileID
		meta["chunk_index"] = globalChunkIndex
		meta["chunk_id"] = fmt.Sprintf("%s_chunk_%d", chunk.FileID, globalChunkIndex)
		meta["chunk_index_scope"] = "file"
		meta["level"] = string(IndexLevelChunk)
		meta["doc_id"] = docID
		meta["section_id"] = ""
		meta["index_version"] = indexVersion
		if doc.FileName != "" {
			meta["file_name"] = doc.FileName
		}

		entries = append(entries, IndexEntry{
			Level:        IndexLevelChunk,
			DocID:        docID,
			SectionID:    "",
			Content:      chunk.Content,
			Metadata:     meta,
			IndexVersion: indexVersion,
		})
	}

	return entries
}

// buildDocSummary concatenates chunk contents up to maxChars, truncating at the limit.
func (m *MultiLevelIndexer) buildDocSummary(chunks []ChunkInput, maxChars int) string {
	var b strings.Builder
	for _, chunk := range chunks {
		remaining := maxChars - b.Len()
		if remaining <= 0 {
			break
		}
		if b.Len() > 0 {
			// Account for the space separator.
			remaining--
			if remaining <= 0 {
				break
			}
			b.WriteByte(' ')
		}
		content := chunk.Content
		if len(content) > remaining {
			content = content[:remaining]
		}
		b.WriteString(content)
	}
	return b.String()
}

type sectionGroup struct {
	Start int
	End   int
}

func (m *MultiLevelIndexer) buildSectionGroups(chunks []ChunkInput, maxChunks, maxBytes int) []sectionGroup {
	if len(chunks) == 0 {
		return nil
	}
	if maxChunks <= 0 {
		maxChunks = defaultSectionSize
	}
	if maxBytes <= 0 {
		maxBytes = defaultSectionContentMaxBytes
	}
	groups := make([]sectionGroup, 0, (len(chunks)+maxChunks-1)/maxChunks)
	start := 0
	count := 0
	contentBytes := 0
	nonEmptyContentCount := 0

	for i, chunk := range chunks {
		nextBytes := len(chunk.Content)
		separatorBytes := 0
		if chunk.Content != "" && nonEmptyContentCount > 0 {
			separatorBytes = 1
		}
		exceedsChunkCount := count >= maxChunks
		exceedsContentBudget := count > 0 && maxBytes > 0 && contentBytes+separatorBytes+nextBytes > maxBytes
		if exceedsChunkCount || exceedsContentBudget {
			groups = append(groups, sectionGroup{Start: start, End: i})
			start = i
			count = 0
			contentBytes = 0
			nonEmptyContentCount = 0
			separatorBytes = 0
		}

		count++
		if chunk.Content != "" {
			contentBytes += separatorBytes + nextBytes
			nonEmptyContentCount++
		}
	}
	if count > 0 {
		groups = append(groups, sectionGroup{Start: start, End: len(chunks)})
	}
	return groups
}

// buildSectionContent concatenates chunk contents within a section and caps only
// the section-level aggregate. Chunk-level index entries are emitted unchanged.
func (m *MultiLevelIndexer) buildSectionContent(chunks []ChunkInput, maxBytes int) (string, bool) {
	var b strings.Builder
	truncated := false
	for _, chunk := range chunks {
		if chunk.Content == "" {
			continue
		}
		if b.Len() > 0 {
			if maxBytes > 0 && b.Len()+1 > maxBytes {
				truncated = true
				break
			}
			b.WriteByte(' ')
		}
		if maxBytes > 0 {
			remaining := maxBytes - b.Len()
			if remaining <= 0 {
				truncated = true
				break
			}
			if len(chunk.Content) > remaining {
				b.WriteString(truncateUTF8Bytes(chunk.Content, remaining))
				truncated = true
				break
			}
		}
		b.WriteString(chunk.Content)
	}
	return b.String(), truncated
}

func truncateUTF8Bytes(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	end := 0
	for i := range s {
		if i > maxBytes {
			break
		}
		end = i
	}
	return s[:end]
}
