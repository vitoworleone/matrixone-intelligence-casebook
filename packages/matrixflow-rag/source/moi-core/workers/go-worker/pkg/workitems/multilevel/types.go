package multilevel

// IndexLevel represents the granularity level of an index entry.
type IndexLevel string

const (
	IndexLevelDocument IndexLevel = "doc"
	IndexLevelSection  IndexLevel = "section"
	IndexLevelChunk    IndexLevel = "chunk"
)

// IndexEntry represents a single entry to be indexed at any granularity level.
type IndexEntry struct {
	Level        IndexLevel     // doc, section, chunk
	DocID        string         // document-level ID (shared across all levels for same doc)
	SectionID    string         // section-level ID (empty for doc level)
	Content      string         // text content for embedding
	Metadata     map[string]any // additional metadata
	IndexVersion int64          // version number for consistency
}

// ChunkInput represents a single chunk from the existing ingest pipeline.
type ChunkInput struct {
	Content    string
	ChunkIndex int
	FileID     string
	Metadata   map[string]any
}

// DocumentInput represents a full document for multi-level indexing.
type DocumentInput struct {
	FileID   string
	FileName string
	Chunks   []ChunkInput
}
