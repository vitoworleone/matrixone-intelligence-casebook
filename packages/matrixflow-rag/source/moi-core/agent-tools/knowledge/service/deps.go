// Package service implements the knowledge tool protocol declared in
// agent-tools/knowledge.
package service

import "github.com/matrixflow/moi-core/agent-tools/knowledge"

type Deps struct {
	SQLExecutor                    knowledge.SQLExecutor
	SQLMutationExecutor            knowledge.SQLMutationExecutor
	QuerySQLHooks                  QuerySQLHooks
	SchemaReader                   SchemaReader
	DescribeSchemaSource           DescribeSchemaSource
	DescribeSchemaCache            DescribeSchemaCache
	DescribeSchemaAccessExecutor   knowledge.SQLExecutor
	DescribeSchemaSemanticProvider DescribeSchemaSemanticProvider
	DescribeSchemaSemanticState    DescribeSchemaSemanticState
	DescribeSchemaHooks            DescribeSchemaHooks
	VisualSearchBackend            VisualSearchBackend
	ParsedMarkdownBackend          ParsedMarkdownBackend
	DescribeSchema                 knowledge.DescribeSchema
	QuerySQL                       knowledge.QuerySQL
	SearchVisualImage              knowledge.SearchVisualImage
	ReadParsedMarkdown             knowledge.ReadParsedMarkdown
	SearchParsedMarkdown           knowledge.SearchParsedMarkdown
	Embedder                       knowledge.EmbeddingService
	DefaultRetrieverConfig         knowledge.RetrieverConfig
}
