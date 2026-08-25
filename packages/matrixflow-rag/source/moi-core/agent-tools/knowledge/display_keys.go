package knowledge

import agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"

const (
	DisplayKeyToolFindRAGFiles         = "label.agent_runtime.tool.find_rag_files"
	DisplayKeyToolSearchRAGChunks      = "label.agent_runtime.tool.search_rag_chunks"
	DisplayKeyToolSearchVisualImage    = "label.agent_runtime.tool.search_visual_image"
	DisplayKeyToolReadParsedMarkdown   = "label.agent_runtime.tool.read_parsed_markdown"
	DisplayKeyToolSearchParsedMarkdown = "label.agent_runtime.tool.search_parsed_markdown"
	DisplayKeyToolDescribeSchema       = "label.agent_runtime.tool.describe_schema"
	DisplayKeyToolQuerySQL             = "label.agent_runtime.tool.query_sql"
	DisplayKeyToolUpsertKnowledgeTable = "label.agent_runtime.tool.upsert_knowledge_table"
	DisplayKeyToolSelectFinalSources   = "label.agent_runtime.tool.select_final_sources"
	DisplayKeyToolSubmitFinalAnswer    = "label.agent_runtime.tool.submit_final_answer"

	DisplayKeyResultRAGChunks    = "label.agent_runtime.result.rag_chunks"
	DisplayKeyResultSQLResult    = "label.agent_runtime.result.sql_result"
	DisplayKeyResultVisualSearch = "label.agent_runtime.result.visual_search"
	DisplayKeyResultAnswer       = "label.agent_runtime.result.answer"

	DisplayKeyReasonKnowledgeDependencyUnavailable = "reason.explore.knowledge.dependency_unavailable"
	DisplayKeyReasonKnowledgeInvalidParams         = "reason.explore.knowledge.invalid_params"
	DisplayKeyReasonKnowledgePermissionDenied      = "reason.explore.knowledge.permission_denied"
	DisplayKeyReasonKnowledgeSQLExecutionFailed    = "reason.explore.knowledge.sql_execution_failed"
	DisplayKeyReasonKnowledgeRAGUnavailable        = "reason.explore.knowledge.rag_unavailable"
)

func DisplayKeySpecs() []agentruntimev2.DisplayKeySpec {
	return []agentruntimev2.DisplayKeySpec{
		{Key: DisplayKeyToolFindRAGFiles, DefaultText: "Find RAG files", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceTool},
		{Key: DisplayKeyToolSearchRAGChunks, DefaultText: "Search RAG chunks", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceTool},
		{Key: DisplayKeyToolSearchVisualImage, DefaultText: "Search visual images", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceTool},
		{Key: DisplayKeyToolReadParsedMarkdown, DefaultText: "Read parsed Markdown", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceTool},
		{Key: DisplayKeyToolSearchParsedMarkdown, DefaultText: "Search parsed Markdown", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceTool},
		{Key: DisplayKeyToolDescribeSchema, DefaultText: "Describe schema", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceTool},
		{Key: DisplayKeyToolQuerySQL, DefaultText: "Query SQL", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceTool},
		{Key: DisplayKeyToolUpsertKnowledgeTable, DefaultText: "Upsert knowledge table", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceTool},
		{Key: DisplayKeyToolSelectFinalSources, DefaultText: "Select final sources", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceTool},
		{Key: DisplayKeyToolSubmitFinalAnswer, DefaultText: "Submit final answer", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceTool},
		{Key: DisplayKeyResultRAGChunks, DefaultText: "RAG chunks", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceResult},
		{Key: DisplayKeyResultSQLResult, DefaultText: "SQL result", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceResult},
		{Key: DisplayKeyResultVisualSearch, DefaultText: "Visual search", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceResult},
		{Key: DisplayKeyResultAnswer, DefaultText: "Final answer", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceResult},
		{Key: DisplayKeyReasonKnowledgeDependencyUnavailable, DefaultText: "A required knowledge dependency is unavailable.", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceReason},
		{Key: DisplayKeyReasonKnowledgeInvalidParams, DefaultText: "The tool request is invalid.", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceReason},
		{Key: DisplayKeyReasonKnowledgePermissionDenied, DefaultText: "Permission denied for the requested knowledge resource.", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceReason},
		{Key: DisplayKeyReasonKnowledgeSQLExecutionFailed, DefaultText: "SQL execution failed.", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceReason},
		{Key: DisplayKeyReasonKnowledgeRAGUnavailable, DefaultText: "RAG retrieval is unavailable.", Owner: "agent-tools/knowledge", Surface: agentruntimev2.DisplaySurfaceReason},
	}
}

func ResultDisplayKeySpecs() []agentruntimev2.DisplayKeySpec {
	specs := DisplayKeySpecs()
	out := make([]agentruntimev2.DisplayKeySpec, 0, len(specs))
	for _, spec := range specs {
		if spec.Surface == agentruntimev2.DisplaySurfaceResult {
			out = append(out, spec)
		}
	}
	return out
}

func ToolDisplayKeySpecs() []agentruntimev2.DisplayKeySpec {
	specs := DisplayKeySpecs()
	out := make([]agentruntimev2.DisplayKeySpec, 0, len(specs))
	for _, spec := range specs {
		if spec.Surface == agentruntimev2.DisplaySurfaceTool {
			out = append(out, spec)
		}
	}
	return out
}
