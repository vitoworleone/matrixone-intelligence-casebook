package service

import "github.com/matrixflow/moi-core/agent-tools/knowledge"

func NewRegistry(deps Deps) *knowledge.Registry {
	registry := &knowledge.Registry{
		FindRAGFiles:         NewFindRAGFiles(deps),
		SearchRAGChunks:      NewSearchRAGChunks(deps),
		ReadParsedMarkdown:   NewReadParsedMarkdown(deps),
		SearchParsedMarkdown: NewSearchParsedMarkdown(deps),
		SearchVisualImage:    NewSearchVisualImage(deps),
		DescribeSchema:       NewDescribeSchema(deps),
		QuerySQL:             NewQuerySQL(deps),
	}
	if deps.SchemaReader != nil && deps.SQLMutationExecutor != nil {
		registry.UpsertKnowledgeTable = NewUpsertKnowledgeTable(deps)
	}
	if deps.ReadParsedMarkdown != nil {
		registry.ReadParsedMarkdown = deps.ReadParsedMarkdown
	}
	if deps.SearchParsedMarkdown != nil {
		registry.SearchParsedMarkdown = deps.SearchParsedMarkdown
	}
	if deps.SearchVisualImage != nil {
		registry.SearchVisualImage = deps.SearchVisualImage
	}
	if deps.DescribeSchema != nil {
		registry.DescribeSchema = deps.DescribeSchema
	}
	if deps.QuerySQL != nil {
		registry.QuerySQL = deps.QuerySQL
	}
	return registry
}
