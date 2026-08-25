package agenttools

import "github.com/matrixflow/moi-core/agent-tools/knowledge"

func findRAGFilesDefinition() ToolDefinition {
	return toolDefinitionFromResource(ToolKindFindRAGFiles, rawSchema(knowledge.FindRAGFilesSchema), rawSchema(knowledgeFindRAGFilesOutputSchema))
}

func searchRAGChunksDefinition() ToolDefinition {
	return toolDefinitionFromResource(ToolKindSearchRAGChunks, rawSchema(knowledge.SearchRAGChunksSchema), rawSchema(knowledgeSearchRAGChunksOutputSchema))
}

func searchVisualImageDefinition() ToolDefinition {
	return toolDefinitionFromResource(ToolKindSearchVisualImage, rawSchema(knowledge.SearchVisualImageSchema), rawSchema(knowledgeSearchVisualImageOutputSchema))
}

func readParsedMarkdownDefinition() ToolDefinition {
	return toolDefinitionFromResource(ToolKindReadParsedMarkdown, rawSchema(knowledge.ReadParsedMarkdownSchema), rawSchema(knowledgeReadParsedMarkdownOutputSchema))
}

func searchParsedMarkdownDefinition() ToolDefinition {
	return toolDefinitionFromResource(ToolKindSearchParsedMarkdown, rawSchema(knowledge.SearchParsedMarkdownSchema), rawSchema(knowledgeSearchParsedMarkdownOutputSchema))
}

func describeSchemaDefinition() ToolDefinition {
	return toolDefinitionFromResource(ToolKindDescribeSchema, rawSchema(knowledge.DescribeSchemaSchema), rawSchema(knowledgeDescribeSchemaOutputSchema))
}

func querySQLDefinition() ToolDefinition {
	return toolDefinitionFromResource(ToolKindQuerySQL, rawSchema(knowledge.QuerySQLSchema), rawSchema(knowledgeQuerySQLOutputSchema))
}

func upsertKnowledgeTableDefinition() ToolDefinition {
	return toolDefinitionFromResource(ToolKindUpsertKnowledgeTable, rawSchema(knowledge.UpsertKnowledgeTableSchema), rawSchema(knowledgeUpsertKnowledgeTableOutputSchema))
}

func submitFinalAnswerDefinition() ToolDefinition {
	return toolDefinitionFromResource(ToolKindSubmitFinalAnswer, rawSchema(knowledge.SubmitFinalAnswerSchema), rawSchema(knowledgeSubmitFinalAnswerOutputSchema))
}

func selectFinalSourcesDefinition() ToolDefinition {
	return toolDefinitionFromResource(ToolKindSelectFinalSources, rawSchema(knowledge.SelectFinalSourcesSchema), rawSchema(knowledgeSelectFinalSourcesOutputSchema))
}
