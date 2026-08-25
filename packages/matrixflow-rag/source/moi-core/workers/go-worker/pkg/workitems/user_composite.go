package workitems

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/mowl"
	"github.com/matrixflow/moi-core/workers/go-worker/pkg/runtime"
	"go.uber.org/zap"
)

type KnowledgeIndexBuild struct {
	Factory *runtime.ClientFactory
	Logger  *zap.Logger
}

type DataTransform struct{}

func (b *KnowledgeIndexBuild) Handle(ctx context.Context, wctx moi.WorkItemContext, msg *mowl.MowlMessage) (*mowl.MowlMessage, error) {
	input, err := parseJSONMap(msg.Data)
	if err != nil {
		return nil, err
	}
	if _, ok := input["table_name"]; !ok {
		if name := strings.TrimSpace(getString(input, "knowledge_base")); name != "" {
			input["table_name"] = name
		}
	}
	if _, ok := input["policy"]; !ok {
		input["policy"] = "OVERWRITE"
	}

	// Multi-level index expansion (doc/section/chunk) before vector write.
	enableMultilevel := true
	if v, ok := input["enable_multilevel_index"]; ok {
		if b, ok := v.(bool); ok {
			enableMultilevel = b
		}
	}
	if enableMultilevel {
		mlInput := map[string]interface{}{
			"documents":    input["documents"],
			"enable":       true,
			"section_size": input["section_size"],
		}
		mlPayload, _ := json.Marshal(mlInput)
		msg.Data = string(mlPayload)
		if _, err := (&MultiLevelIndex{}).Handle(ctx, wctx, msg); err != nil {
			return nil, fmt.Errorf("multilevel_index: %w", err)
		}
		// msg.Data now contains expanded documents; merge back into input
		var mlOut map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Data), &mlOut); err == nil {
			if docs, ok := mlOut["documents"]; ok {
				input["documents"] = docs
			}
		}
	}
	ensureKnowledgeIndexVersion(input)

	payload, _ := json.Marshal(input)
	msg.Data = string(payload)
	return (&DataVectorWrite{Factory: b.Factory, Logger: b.Logger}).Handle(ctx, wctx, msg)
}

func ensureKnowledgeIndexVersion(input map[string]interface{}) int64 {
	indexVersion := toInt64(input["index_version"], 0)
	if indexVersion <= 0 {
		indexVersion = time.Now().UnixMilli()
		input["index_version"] = indexVersion
	}
	if docs, ok := input["documents"].([]interface{}); ok {
		for _, raw := range docs {
			doc, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			meta := ensureMap(doc["metadata"])
			meta["index_version"] = indexVersion
			doc["metadata"] = meta
		}
	}
	return indexVersion
}

func (t *DataTransform) Handle(_ context.Context, _ moi.WorkItemContext, msg *mowl.MowlMessage) (*mowl.MowlMessage, error) {
	input, err := parseJSONMap(msg.Data)
	if err != nil {
		return nil, err
	}
	target := strings.ToLower(strings.TrimSpace(getString(input, "target_type")))
	if target == "" {
		target = "json"
	}
	value := firstNonNilValue(input["value"], input["json"], input["documents"], input["rows"], input["text"])
	out := map[string]interface{}{"target_type": target}
	switch target {
	case "text":
		out["text"] = stringifyTransformValue(value)
	case "documents":
		if docs, ok := input["documents"]; ok {
			out["documents"] = docs
		} else {
			out["documents"] = []map[string]interface{}{{"content": stringifyTransformValue(value), "type": "text"}}
		}
	case "rows", "table":
		out["columns"] = input["columns"]
		out["rows"] = input["rows"]
	default:
		out["json"] = value
	}
	payload, _ := json.Marshal(out)
	msg.Data = string(payload)
	return nil, nil
}

func stringifyTransformValue(value interface{}) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		b, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(b)
	}
}
