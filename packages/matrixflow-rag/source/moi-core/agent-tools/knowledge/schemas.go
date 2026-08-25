package knowledge

import "encoding/json"

const FindRAGFilesDescription = "Search the scoped knowledge base for relevant source files/documents when the user's request can narrow retrieval to source-level constraints, such as a named file/document/source, period, title, document type, or other source identifier. Do not use this merely because the question is broad. If the request cannot narrow to source files, skip this tool and call search_rag_chunks without file_ids for whole-scope retrieval. If this returns no files, do not treat that alone as absence of evidence; call search_rag_chunks without file_ids unless the user only asked for file inventory."
const SearchRAGChunksDescription = "Search the scoped knowledge base for relevant text chunks/passages and return primary evidence for answering. Use this for document content, facts, entities, requirements, summaries, comparisons, and follow-up questions. Call it directly without file_ids when the request cannot be narrowed to specific source files/documents; that performs whole-scope knowledge-base retrieval. Choose max_hits from the task size and recall needs; for list-building, comparison, or ranking requests, request enough candidate hits and evidence groups to cover the requested population plus review margin. Some chunks may include markdown_file_id; that is only a drill-down pointer for read_parsed_markdown/search_parsed_markdown when the returned chunks are insufficient. Do not ignore or replace these chunks just because markdown_file_id is present. When results are empty or insufficient, retry with broader or alternative keywords or larger limits before concluding evidence is unavailable."
const SearchVisualImageDescription = "Search the scoped visual document image index and return matched drawing/page/object image backlinks. Use this for text-to-drawing, visual evidence lookup, screenshot search, similar drawings, or text+visual-input hybrid search. Choose parameters from the current user message inputs: text-only requests pass only query_text and omit query_visual; requests with only a visual input attachment pass only query_visual; requests with both text and a visual input attachment pass both query_text and query_visual for hybrid search. query_visual refers to a current-message visual input attachment that can be embedded visually; ordinary document/file attachments that are not visual inputs should be handled through document/RAG tools instead. If query_visual is provided but no current-message visual input exists, the tool returns an explicit error unless query_text is also provided for text-only search. When the current message includes visual inputs, omitted query_visual defaults to the first current-message visual input and omitted ranking_profile defaults to visual_object_first so hybrid searches keep that visual input as primary evidence. Use ranking_profile=visual_object_first only when matching an engineering drawing or part image by internal visual objects. Use ranking_profile=visual_text_region_first when matching a technical requirements, notes, table, or title-block region by visible text or table structure. Do not use search_rag_chunks as a substitute for visual image evidence. If no visual results are returned, do not create a visual_hit source in select_final_sources."
const ReadParsedMarkdownDescription = "Read a bounded page from parsed markdown by markdown_file_id returned by search_rag_chunks. Use this only to inspect more surrounding document text after chunk search has identified a relevant source. Keep reads bounded and advance cursor only when additional context is needed."
const SearchParsedMarkdownDescription = "Search literal text inside a parsed markdown document by markdown_file_id returned by search_rag_chunks. Use this after chunk search when the evidence source is known and exact terms or nearby context need targeted lookup."
const DescribeSchemaDescription = "Describe the selected structured tables for NL2SQL questions. Use this before writing SQL when the selected knowledge base contains database tables or semantic models. The server injects workspace, database, selected tables, and semantic model scope."
const QuerySQLDescription = "Execute a read-only SELECT or WITH SQL statement against the selected structured table scope and return result rows. SQL must be valid MySQL 8 syntax: reserved words used as identifiers or aliases must be backtick-quoted (for example AS `current_time`) or renamed to a non-reserved alias such as now_value. Prefer table queries over selected tables; pure scalar SELECT without FROM (for example SELECT NOW() AS now_value) is allowed. Use this to answer NL2SQL questions after inspecting describe_schema. Do not use this for document/RAG retrieval. When the SQL intentionally follows semantic entries returned by describe_schema, pass their exact keys in semantic_claims so trace/UI can show which semantics were used."
const SelectFinalSourcesDescription = "Select the evidence sources that will be used for the final user-facing answer. Call this after retrieval tools finish and before writing the final answer text. For document text evidence, cite rag_chunk with chunk_id or chunk_ids copied from search_rag_chunks. For visual image evidence, cite visual_hit with object_id, image_file_id, or page_image_file_id copied from search_visual_image. When a retrieval result includes semantic_model_id, copy it with the evidence identifier so identical IDs from different semantic models remain unambiguous. For NL2SQL evidence, cite sql_result and copy the exact artifact_id returned by query_sql. Empty search results are not evidence: if no citable evidence exists after a successful retrieval, pass an empty sources array. If this runtime exposes no citable evidence retrieval tool, pass an empty sources array."
const SelectFinalSourcesSchema = `{
  "type":"object",
  "properties":{
    "sources":{"type":"array","description":"Evidence sources that will support the upcoming final answer. Use an empty array only after a RAG, visual, or SQL retrieval completed with no citable evidence, or when this runtime exposes no citable evidence retrieval tool.","items":{"type":"object","properties":{
      "type":{"type":"string","enum":["rag_chunk","visual_hit","sql_result"],"description":"Source kind."},
      "semantic_model_id":{"type":"integer","minimum":1,"description":"Trusted semantic model owner copied exactly from the selected retrieval result. Required to disambiguate identical evidence IDs returned by different semantic models."},
      "artifact_id":{"type":"string","description":"Exact SQL artifact_id returned by query_sql."},
      "chunk_id":{"type":"string","description":"RAG chunk id when type is rag_chunk."},
      "chunk_ids":{"type":"array","items":{"type":"string"},"description":"RAG chunk ids when citing several chunks."},
      "object_id":{"type":"string","description":"Visual object id from search_visual_image."},
      "image_file_id":{"type":"string","description":"Matched visual image file id from search_visual_image."},
      "page_image_file_id":{"type":"string","description":"Matched page image file id from search_visual_image."}
    },"required":["type"],"additionalProperties":false}}
  },
  "required":["sources"],
  "additionalProperties":false
}`
const SubmitFinalAnswerDescription = "Submit the final user-facing answer with only the evidence identifiers actually used in that answer. Use this as the final knowledge-agent step after SQL/RAG/visual evidence is inspected. For document text evidence, cite rag_chunk with chunk_id or chunk_ids copied from search_rag_chunks. For visual image evidence, cite visual_hit with object_id, image_file_id, or page_image_file_id copied from search_visual_image. When a retrieval result includes semantic_model_id, copy it with the evidence identifier so identical IDs from different semantic models remain unambiguous. For NL2SQL evidence, cite sql_result and copy the exact artifact_id returned by query_sql; the tool resolves database table sources deterministically. Do not include every tool result; include only evidence that supports final-answer claims. Empty search results are not evidence: if the answer says no relevant result was found, pass an empty sources array instead of placeholder rag_chunk or visual_hit sources. The answer field must be natural language for end users: never paste internal chunk locators such as file_uuid:version:chunk:start:end, raw chunk_id strings, or evidence artifact ids into the answer body; put those only in sources."
const SubmitFinalAnswerRepairPrompt = "The previous run produced final assistant text but did not call the required final-answer submission tool. Convert the latest assistant final answer into exactly one final-answer submission tool call now.\n\nUse the latest assistant final answer as the user-facing answer. Preserve the user's language. Cite only evidence identifiers already present in the preceding tool results or prefetched visual search context, copying IDs and reference fields exactly from those results. For prefetched search_visual_image results, cite visual_hit with object_id, image_file_id, or page_image_file_id from the JSON result and copy semantic_model_id when present. Empty search results are not evidence; if no cited evidence exists, use sources:[]. Do not add new claims, do not call any other tool, and do not answer with plain assistant text."
const SelectFinalSourcesRepairPrompt = "The previous run produced or attempted a final answer before selecting sources. Call select_final_sources exactly once now.\n\nSelect only evidence identifiers already present in the preceding tool results and copy semantic_model_id when it is present on a cited RAG or visual result. Empty search results are not evidence; if no citable evidence exists after retrieval, use sources:[]. Do not write the final answer in this step and do not call any other tool."
const SubmitFinalAnswerEvidenceRepairPrompt = "The previous run produced user-facing assistant text before calling any required evidence tool. Inspect the available runtime evidence now before sources are selected or the final answer is written.\n\nCall exactly one available evidence tool first. Use focused arguments derived from the user's latest request and the current turn context. If the request is ambiguous, still search the available scoped resources for the likely relevant terms before asking for clarification. Do not call select_final_sources in this repair step."

const FindRAGFilesSchema = `{
  "type":"object",
  "properties":{
    "query":{"type":"string","description":"Natural-language query containing source-level constraints for source files/documents in the selected knowledge scope."},
    "top_k":{"type":"integer","minimum":1,"description":"Maximum number of candidate files."}
  },
  "required":["query"],
  "additionalProperties":false
}`

const SearchRAGChunksSchema = `{
  "type":"object",
  "properties":{
    "keywords":{"type":"array","items":{"type":"string"},"description":"Evidence keywords/fragments to recall. Pass focused fragments such as entity, role, requirement, company, period, document title, table name, and metric names. The tool uses exactly these items for full-text and vector recall; avoid passing the complete user question as one item when focused fragments are available."},
    "volume_id":{"type":"string","description":"Optional volume boundary. Usually omit it because the request scope is injected by the server."},
    "file_ids":{"type":"array","items":{"type":"string"},"description":"Optional explicit file_id scope from find_rag_files. Omit this field for whole-scope knowledge-base retrieval."},
    "max_hits":{"type":"integer","minimum":1,"description":"Required candidate hits per keyword per route before expansion. Choose this from the answer size and recall needs; the server uses it directly and applies no hidden hard cap."},
    "max_rows":{"type":"integer","minimum":1,"description":"Optional maximum logical evidence groups returned after table/section expansion. One merged table or document section counts as one result even when it contains multiple expanded chunks. Omit it to return all recalled groups."},
    "before":{"type":"integer","minimum":0,"description":"Neighboring parent_index values before an expanded body range. Omit for 0."},
    "after":{"type":"integer","minimum":0,"description":"Neighboring parent_index values after an expanded body range. Omit for 0."}
  },
  "required":["keywords","max_hits"],
  "additionalProperties":false
}`

const SearchVisualImageSchema = `{
  "type":"object",
  "properties":{
    "query_visual":{"type":"integer","minimum":1,"description":"1-based index of the current user message visual input attachment parts. Omit for text-only visual search. Use for visual-input-only search, or together with query_text for hybrid text+visual search. Do not use this for ordinary document/file attachments that are not visual inputs. If no current-message visual input exists, providing this field without query_text is an error."},
    "query_text":{"type":"string","description":"Text query for text-to-drawing search. Use alone for text-only visual search, or together with query_visual for hybrid text+visual search. This field can be used without query_visual."},
    "ranking_profile":{"type":"string","enum":["visual_object_first","visual_text_region_first"],"description":"Optional ranking profile. Use visual_object_first only when matching an engineering drawing or part image by internal visual objects. Use visual_text_region_first when matching a technical requirements, notes, table, or title-block region by visible text or table structure. Omit for normal document/page visual search."},
    "top_k":{"type":"integer","minimum":1,"description":"Maximum number of visual matches to return. Use enough results to satisfy the requested Top K."}
  },
  "additionalProperties":false
}`

const ReadParsedMarkdownSchema = `{
  "type":"object",
  "properties":{
    "markdown_file_id":{"type":"string","description":"markdown_file_id returned by search_rag_chunks."},
    "cursor":{"type":"integer","minimum":0,"description":"Character cursor. Omit or pass 0 for the first page."},
    "limit_chars":{"type":"integer","minimum":1,"description":"Maximum characters to return."}
  },
  "required":["markdown_file_id"],
  "additionalProperties":false
}`

const SearchParsedMarkdownSchema = `{
  "type":"object",
  "properties":{
    "markdown_file_id":{"type":"string","description":"markdown_file_id returned by search_rag_chunks."},
    "query":{"type":"string","description":"Literal text query to search inside the parsed markdown file."},
    "max_matches":{"type":"integer","minimum":1,"description":"Maximum number of matches."},
    "context_chars":{"type":"integer","minimum":0,"description":"Characters of context around each match."}
  },
  "required":["markdown_file_id","query"],
  "additionalProperties":false
}`

const DescribeSchemaSchema = `{
  "type":"object",
  "properties":{
    "table_names":{"type":"array","items":{"type":"string"},"description":"Optional subset of selected table names to describe. Omit or pass [] to describe every selected table."},
    "include_samples":{"type":"integer","minimum":0,"description":"Number of sample rows per table. Omit for the server default."},
    "max_ddl_chars":{"type":"integer","minimum":0,"description":"Maximum DDL characters per table. Omit for the server default; 0 returns full DDL."}
  },
  "additionalProperties":false
}`

// QuerySQLSchema is the JSON Schema for query_sql tool input. Backticks cannot
// appear inside a Go raw string, so the reserved-alias example uses an explicit
// "quote with backticks" wording instead of embedding `current_time`.
const QuerySQLSchema = `{
  "type":"object",
  "properties":{
    "sql":{"type":"string","description":"A single read-only MySQL 8 SELECT or WITH statement. Prefer queries over selected tables as database.table. Pure scalar SELECT without FROM is allowed (for example SELECT NOW() AS now_value). Reserved-word aliases must be quoted with backticks or renamed to a non-reserved alias such as now_value."},
    "max_rows":{"type":"integer","minimum":1,"description":"Maximum rows returned to the model. Omit for the server default."},
    "semantic_claims":{"type":"array","items":{"type":"string"},"description":"Exact semantic keys from describe_schema that this SQL intentionally honors. Omit or pass [] when no semantic entry was used."}
  },
  "required":["sql"],
  "additionalProperties":false
}`

const UpsertKnowledgeTableSchema = `{
  "type":"object",
  "properties":{
    "table_name":{"type":"string","description":"Exact table name from the currently bound knowledge base."},
    "key":{"type":"object","minProperties":1,"description":"One or more existing key columns and their values. The server uses these to make the update idempotent."},
    "values":{"type":"object","minProperties":1,"description":"Confirmed values to create or update. Do not include unverified inferences."},
    "records":{"type":"array","minItems":1,"maxItems":100,"description":"A bounded atomic batch. Every record must have the same key and value column sets. Use this instead of key and values when several confirmed facts belong to the same table.","items":{"type":"object","properties":{"key":{"type":"object","minProperties":1},"values":{"type":"object","minProperties":1}},"required":["key","values"],"additionalProperties":false}}
  },
  "required":["table_name"],
  "oneOf":[
    {"required":["key","values"]},
    {"required":["records"]}
  ],
  "additionalProperties":false
}`

const SubmitFinalAnswerSchema = `{
  "type":"object",
  "properties":{
    "answer":{"type":"string","description":"Final user-facing answer in Markdown, in the user's language."},
    "sources":{"type":"array","description":"Only concrete evidence sources actually used by this final answer. Use an empty array when the answer only states that searches returned no relevant evidence; never include placeholder empty rag_chunk or visual_hit sources.","items":{"type":"object","properties":{
      "type":{"type":"string","description":"Source kind. Use rag_chunk for RAG text evidence, visual_hit for image-search evidence, or sql_result for NL2SQL evidence. sql_table is accepted only when no sql_result artifact exists."},
      "semantic_model_id":{"type":"integer","minimum":1,"description":"Trusted semantic model owner copied exactly from the selected retrieval result. Required to disambiguate identical evidence IDs returned by different semantic models."},
      "artifact_id":{"type":"string","description":"Exact SQL artifact_id returned by query_sql. Use only when type is sql_result; do not use artifact_id for rag_chunk."},
      "chunk_id":{"type":"string","description":"RAG chunk id when type is rag_chunk."},
      "chunk_ids":{"type":"array","items":{"type":"string"},"description":"RAG chunk ids when citing several chunks."},
      "object_id":{"type":"string","description":"Visual object id copied from search_visual_image when type is visual_hit."},
      "image_file_id":{"type":"string","description":"Matched visual object image file id copied from search_visual_image when type is visual_hit."},
      "page_image_file_id":{"type":"string","description":"Matched page image file id copied from search_visual_image when type is visual_hit."},
      "database":{"type":"string","description":"Database/schema name only when type is sql_table."},
      "table":{"type":"string","description":"Table name only when type is sql_table."},
      "label":{"type":"string","description":"Short display label only when ids are unavailable."}
    },"required":["type"],"additionalProperties":false}}
  },
  "required":["answer","sources"],
  "additionalProperties":false
}`

func schema(raw string) json.RawMessage {
	return json.RawMessage(raw)
}
