package knowledge

import (
	"encoding/json"
	"fmt"
	"strings"

	agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"
)

func decodeParams[T any](name string, raw json.RawMessage) (T, error) {
	var params T
	if err := agentruntimev2.StrictDecode(raw, &params); err != nil {
		return params, fmt.Errorf("%s: invalid arguments: %w", name, err)
	}
	return params, nil
}

func marshalToolResult(value any) (*agentruntimev2.ToolResult, error) {
	return marshalToolResultWithMetadata(value, nil)
}

func marshalToolResultWithMetadata(value any, metadata map[string]any) (*agentruntimev2.ToolResult, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal tool result: %w", err)
	}
	return &agentruntimev2.ToolResult{
		Content:  string(data),
		Data:     value,
		Metadata: metadata,
	}, nil
}

func mergeToolMetadata(base map[string]any, additions map[string]any) map[string]any {
	if len(base) == 0 && len(additions) == 0 {
		return nil
	}
	out := make(map[string]any, len(base)+len(additions))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range additions {
		out[key] = value
	}
	return out
}

func invocationArtifactID(invocation agentruntimev2.ToolInvocation, prefix string) string {
	callID := strings.TrimSpace(invocation.CallID)
	if callID == "" {
		return prefix
	}
	return prefix + "_" + callID
}

func normalizeTopK(value int) int {
	if value <= 0 {
		return 8
	}
	return value
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func cloneQuerySQLResponse(in QuerySQLResponse) QuerySQLResponse {
	out := in
	out.SourceSQLs = append([]string(nil), in.SourceSQLs...)
	out.Columns = append([]string(nil), in.Columns...)
	out.ColumnDescriptions = append([]string(nil), in.ColumnDescriptions...)
	out.ColumnFormats = append([]string(nil), in.ColumnFormats...)
	out.Rows = make([][]any, len(in.Rows))
	for i, row := range in.Rows {
		out.Rows[i] = append([]any(nil), row...)
	}
	out.TableNames = append([]string(nil), in.TableNames...)
	out.Tables = append([]TableRef(nil), in.Tables...)
	out.SemanticKeysUsed = append([]string(nil), in.SemanticKeysUsed...)
	out.AppliedConstraints = cloneSQLAppliedConstraints(in.AppliedConstraints)
	return out
}

func cloneSQLAppliedConstraints(in []SQLAppliedConstraint) []SQLAppliedConstraint {
	if len(in) == 0 {
		return nil
	}
	out := make([]SQLAppliedConstraint, len(in))
	for i, constraint := range in {
		out[i] = constraint
		out[i].Values = append([]string(nil), constraint.Values...)
	}
	return out
}

func QueryVisualsFromMessageParts(parts []map[string]any) []QueryVisualRef {
	if len(parts) == 0 {
		return nil
	}
	out := make([]QueryVisualRef, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		ref, ok := queryVisualFromMessagePart(part)
		if !ok {
			continue
		}
		if _, exists := seen[ref.FileID]; exists {
			continue
		}
		seen[ref.FileID] = struct{}{}
		out = append(out, ref)
	}
	return out
}

func queryVisualFromMessagePart(part map[string]any) (QueryVisualRef, bool) {
	kind := strings.ToLower(strings.TrimSpace(knowledgeStringFromMap(part, "kind")))
	if kind != "image" && kind != "file" && kind != "data" {
		return QueryVisualRef{}, false
	}
	record := knowledgeMapFromAny(part["data"])
	metadata := knowledgeMapFromAny(part["metadata"])
	file := knowledgeMapFromAny(part["file"])
	fileID := firstNonEmpty(
		knowledgeStringFromMap(record, "file_id"),
		knowledgeStringFromMap(record, "fileId"),
		knowledgeStringFromMap(record, "id"),
		knowledgeStringFromMap(metadata, "file_id"),
		knowledgeStringFromMap(metadata, "fileId"),
		knowledgeStringFromMap(metadata, "id"),
		knowledgeStringFromMap(file, "file_id"),
		knowledgeStringFromMap(file, "fileId"),
		knowledgeStringFromMap(file, "id"),
	)
	if fileID == "" {
		return QueryVisualRef{}, false
	}
	mimeType := firstNonEmpty(
		knowledgeStringFromMap(record, "mime_type"),
		knowledgeStringFromMap(record, "mimeType"),
		knowledgeStringFromMap(metadata, "mime_type"),
		knowledgeStringFromMap(metadata, "mimeType"),
		knowledgeStringFromMap(file, "mime_type"),
		knowledgeStringFromMap(file, "mimeType"),
	)
	if kind == "file" && mimeType != "" && !strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		return QueryVisualRef{}, false
	}
	return QueryVisualRef{
		FileID: fileID,
		FileName: firstNonEmpty(
			knowledgeStringFromMap(record, "file_name"),
			knowledgeStringFromMap(record, "fileName"),
			knowledgeStringFromMap(record, "name"),
			knowledgeStringFromMap(metadata, "file_name"),
			knowledgeStringFromMap(metadata, "fileName"),
			knowledgeStringFromMap(metadata, "name"),
			knowledgeStringFromMap(file, "file_name"),
			knowledgeStringFromMap(file, "fileName"),
			knowledgeStringFromMap(file, "name"),
		),
		MimeType: mimeType,
		Metadata: map[string]any{
			"part_kind": kind,
		},
	}, true
}

func knowledgeMapFromAny(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneKnowledgeMap(typed)
	case map[string]string:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = item
		}
		return out
	default:
		return nil
	}
}

func knowledgeStringFromMap(values map[string]any, key string) string {
	if len(values) == 0 {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func cloneKnowledgeMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = cloneKnowledgeAny(value)
	}
	return out
}

func cloneKnowledgeAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneKnowledgeMap(typed)
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, cloneKnowledgeAny(item))
		}
		return out
	default:
		return typed
	}
}
