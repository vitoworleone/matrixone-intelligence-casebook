package knowledge

// Registry bundles the knowledge tools exposed through agent-tools.
type Registry struct {
	FindRAGFiles         FindRAGFiles
	SearchRAGChunks      SearchRAGChunks
	SearchVisualImage    SearchVisualImage
	ReadParsedMarkdown   ReadParsedMarkdown
	SearchParsedMarkdown SearchParsedMarkdown
	DescribeSchema       DescribeSchema
	QuerySQL             QuerySQL
	UpsertKnowledgeTable UpsertKnowledgeTable
}

func Names() []string {
	return []string{
		ToolNameFindRAGFiles,
		ToolNameSearchRAGChunks,
		ToolNameSearchVisualImage,
		ToolNameReadParsedMarkdown,
		ToolNameSearchParsedMarkdown,
		ToolNameDescribeSchema,
		ToolNameQuerySQL,
		ToolNameUpsertKnowledgeTable,
		ToolNameSelectFinalSources,
		ToolNameSubmitFinalAnswer,
	}
}
