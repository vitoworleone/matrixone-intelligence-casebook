package workitems

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/matrixflow/moi-core/model/logging/log"
	"sort"
	"time"

	"github.com/matrixflow/moi-core/workers/go-worker/pkg/workitems/multilevel"
	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/data"
	"github.com/matrixflow/moi-core/model/mowl"
)

// MultiLevelIndex expands chunk-level documents into doc/section/chunk index entries.
// It preserves chunk metadata and adds level/doc_id/section_id/index_version metadata
// so downstream retrieval can use multi-granularity indexing.
type MultiLevelIndex struct{}

func (w *MultiLevelIndex) Handle(_ context.Context, _ moi.WorkItemContext, msg *mowl.MowlMessage) (*mowl.MowlMessage, error) {
	var in struct {
		Documents          []map[string]interface{} `json:"documents"`
		Enable             *bool                    `json:"enable"`
		IndexVersion       int64                    `json:"index_version"`
		SectionSize        int                      `json:"section_size"`
		MaxDocSummaryChars int                      `json:"max_doc_summary_chars"`
	}
	if err := json.Unmarshal([]byte(msg.Data), &in); err != nil {
		return nil, fmt.Errorf("parse input: %w", err)
	}
	if len(in.Documents) == 0 {
		return nil, fmt.Errorf("documents is empty")
	}
	log.Printf("[multilevel_index] input docs=%d enable=%v", len(in.Documents), in.Enable)

	enable := true
	if in.Enable != nil {
		enable = *in.Enable
	}
	if !enable {
		out, _ := json.Marshal(map[string]interface{}{"documents": in.Documents})
		msg.Data = string(out)
		return nil, nil
	}

	docs := make([]Document, 0, len(in.Documents))
	for _, raw := range in.Documents {
		doc := Document{
			ID:       toString(raw["id"]),
			Content:  toString(raw["content"]),
			Type:     toString(raw["type"]),
			Metadata: ensureMap(raw["metadata"]),
		}
		if emb, ok := raw["embedding"].([]interface{}); ok {
			doc.Embedding = toFloat64Slice(emb)
		}
		docs = append(docs, doc)
	}
	if err := validateIndexableDocuments(docs); err != nil {
		return nil, err
	}

	indexVersion := in.IndexVersion
	if indexVersion <= 0 {
		indexVersion = time.Now().UnixMilli()
	}

	indexer := multilevel.NewMultiLevelIndexer()
	if in.SectionSize > 0 {
		indexer.SectionSize = in.SectionSize
	}
	if in.MaxDocSummaryChars > 0 {
		indexer.MaxDocSummaryChars = in.MaxDocSummaryChars
	}

	type fileGroup struct {
		fileID   string
		fileName string
		chunks   []Document
	}
	groups := map[string]*fileGroup{}
	orphan := make([]Document, 0)

	for _, doc := range docs {
		meta := doc.Metadata
		fileID := toString(meta["file_id"])
		if fileID == "" {
			fileID = toString(meta["raw_file_id"])
		}
		if fileID == "" {
			orphan = append(orphan, doc)
			continue
		}
		meta["file_id"] = fileID
		fileName := toString(meta["file_name"])
		group := groups[fileID]
		if group == nil {
			group = &fileGroup{fileID: fileID, fileName: fileName}
			groups[fileID] = group
		}
		if group.fileName == "" && fileName != "" {
			group.fileName = fileName
		}
		group.chunks = append(group.chunks, doc)
	}

	outDocs := make([]Document, 0, len(docs))

	for _, group := range groups {
		if group.fileName == "" {
			return nil, fmt.Errorf("file_name is missing for file_id=%s; ensure the upstream parser sets file_name in document metadata", group.fileID)
		}
		sort.SliceStable(group.chunks, func(i, j int) bool {
			iParent := toInt(group.chunks[i].Metadata["parent_index"], i)
			jParent := toInt(group.chunks[j].Metadata["parent_index"], j)
			if iParent != jParent {
				return iParent < jParent
			}
			iIdx := toInt(group.chunks[i].Metadata["chunk_index"], i)
			jIdx := toInt(group.chunks[j].Metadata["chunk_index"], j)
			return iIdx < jIdx
		})
		chunks := make([]multilevel.ChunkInput, 0, len(group.chunks))
		for idx, doc := range group.chunks {
			chunkIdx := toInt(doc.Metadata["chunk_index"], idx)
			chunks = append(chunks, multilevel.ChunkInput{
				Content:    doc.Content,
				ChunkIndex: chunkIdx,
				FileID:     group.fileID,
				Metadata:   doc.Metadata,
			})
		}
		entries := indexer.GenerateEntries(multilevel.DocumentInput{
			FileID:   group.fileID,
			FileName: group.fileName,
			Chunks:   chunks,
		}, indexVersion)

		for _, e := range entries {
			meta := cloneMap(e.Metadata)
			if meta == nil {
				meta = map[string]interface{}{}
			}
			if meta["file_id"] == nil {
				meta["file_id"] = group.fileID
			}
			if meta["file_name"] == nil && group.fileName != "" {
				meta["file_name"] = group.fileName
			}
			if meta["level"] == nil {
				meta["level"] = string(e.Level)
			}
			if meta["doc_id"] == nil && e.DocID != "" {
				meta["doc_id"] = e.DocID
			}
			if meta["section_id"] == nil && e.SectionID != "" {
				meta["section_id"] = e.SectionID
			}
			if meta["index_version"] == nil && e.IndexVersion != 0 {
				meta["index_version"] = e.IndexVersion
			}
			outDocs = append(outDocs, Document{
				Content:  e.Content,
				Type:     "text",
				Metadata: meta,
			})
		}
	}

	for _, doc := range orphan {
		if doc.Metadata == nil {
			doc.Metadata = map[string]interface{}{}
		}
		if _, ok := doc.Metadata["level"]; !ok {
			doc.Metadata["level"] = "chunk"
		}
		outDocs = append(outDocs, doc)
	}

	protoDocs := make([]*data.Document, 0, len(outDocs))
	for _, d := range outDocs {
		p, err := documentToProto(d)
		if err != nil {
			return nil, err
		}
		protoDocs = append(protoDocs, p)
	}
	resp := &data.SplitDocumentsLengthResponse{Documents: protoDocs}
	out, _ := json.Marshal(resp)
	msg.Data = string(out)
	return nil, nil
}
