# RAG Ingest Default Workflow

This document captures the default RAG ingest workflow using the moi-core
WorkItem-based DSL. The workflow provides 1:1 functional parity with the
workflow_be Haystack pipeline created via the catalog API.

## Mapping to workflow_be Haystack Pipeline

The workflow_be RAG ingest pipeline is created by the frontend via catalog API
with the following node chain:

```
RootNode → DocumentParseNode → ChunkNode → EmbedNode → WriteNode
```

Each node maps to Haystack components internally. The moi-core workflow
replaces these with equivalent WorkItems and adds explicit data lineage
tracking (asset register/link) that workflow_be handles externally via
`job_consumer.process_file` + `CRUDCatalogService.create_file`.

### Core Pipeline Mapping

| Step | workflow_be Node | Haystack Component | Workflow Node | moi-core WorkItem |
|------|-----------------|-------------------|---------------|-------------------|
| 1 | `RootNode` + `DocumentParseNode` | `FileTypeRouter` → converters (PDF/DOCX/PPTX/HTML/TXT/Excel) | `parse` | `moi:parser.convert.document.rich` |
| 2 | `ChunkNode` | `EnhancedDocumentSplitter` (enable_level_based_split=true) | `chunk` | `moi:parser.split.documents.length` |
| 3 | `EmbedNode` | `OpenAIDocumentEmbedder` | `embedding_generate` | `moi:embedding.generate` |
| 4 | `WriteNode` | `DocumentWriter` + `MOIDocumentStore` | `vector_write` | `moi:data.retrieval.vector.write` |

### Standard RAG Image Context

Parse V3 image-owned text (OCR, source figure caption, generated caption,
`figure_no`) is defined once in
[parse-v3-image-semantics.md](../workers/parse-v3-image-semantics.md).
API `documents.Content` / top-level `text`, ordinary chunks,
`document_visual`, and Writer ZIP each present that truth differently.
Do not treat an API `text` pass as a Writer or visual pass.

When the standard RAG workflow enables image indexing, `moi:document_visual.parse`
builds a `standard_rag_v1` visual manifest for page/object image rows. With
`require_visual_context=false`, each manifest `objects[]` entry keeps its
non-empty `ocr` (OCR), `figure_caption` (source figure caption), `caption`
(generated caption), `text` (local text), `figure_no`, and
`caption_block_uuid` values. Its local context has one canonical projection
order: OCR -> source figure caption -> generated caption -> local text. Values
are trimmed and deduplicated by exact case-sensitive equality; case folding and
semantic matching are not used. Page-level or document-level text is not copied
into visual object context, and `moi:document_visual.index.image` uses the same
local object context for `visual_object` image rows.

V3 figure-caption relations fail closed. A valid relation is a one-to-one pair
in the same source and on the same page. Source identity is the first non-empty
trimmed value from `raw_file_id`, `source_file_id`, then `file_id`. The pair has
unique, non-empty `block_uuid` values: the `IMAGE` has non-empty
`figure_caption` and `caption_block_uuid` and has no `caption_for`. Unrelated
open metadata such as an IMAGE `role=decorative` is preserved and does not
participate in the relation. The referenced `TEXT` or `LIST` has
`role=caption` plus a non-empty reciprocal `caption_for`. After trimming, both references must
identify each other and the caption content must equal `figure_caption`
case-sensitively. If either endpoint has `figure_no`, both must have the same
non-empty trimmed value. Once any V3
relation marker (`figure_caption`, `figure_no`, `caption_block_uuid`, or
`caption_for`) is present, missing, blank, non-string, dangling, one-way,
wrong-kind, wrong-role, cross-source, cross-page, text/number-mismatched,
duplicate-ID, or non-one-to-one data rejects the operation instead of being
partially projected. A legacy `role=caption` plus `contained_in` block with no
V3 relation marker remains a non-relation caption.

Both `moi:document_visual.index.text` and
`moi:document_visual.index.image` project non-empty `ocr_text`, `caption`,
`figure_caption`, `figure_no`, `caption_block_uuid`, and `source_block_id`
values into object-row index metadata.

### Runtime Retrieval Version Scope

Runtime RAG and visual retrieval must search the file versions authorized by the
knowledge base scope, not every historical vector row for the same `file_id`.
For semantic-model-backed knowledge bases, Catalog resolves file scope from
`knowledge_base_sources` and legacy semantic model file pointers:

- A governed source is searchable only when `status='succeeded'` and
  `effective_enabled=true`.
- Pending, running, failed, disabled, or expired governed sources are not added
  to RAG scope, and an existing governance row for the same file prevents legacy
  compatibility from bypassing that state.
- If an explicit legacy file pointer has no non-removed governance row, Catalog
  keeps it in runtime RAG scope and marks the source metadata with
  `governance_mode=legacy_compat`.
- If a searchable source has `segment_version_id` and `index_version`, runtime
  scope uses that exact `(file_id, index_version)` pair.
- If a searchable legacy-compatible source has no pointer, Catalog reads
  existing vector rows for that source file and builds a runtime-only
  `index_version` constraint. It chooses the smallest non-NULL `index_version`
  present for the file; if only NULL versions exist, retrieval uses
  `index_version IS NULL`.
- If the vector table has no `index_version` column, retrieval treats it as an
  old single-version table and keeps only the file scope.

`search_rag_chunks`, `find_rag_files`, and visual search consume the resolved
scope. They must not fall back to a bare `file_id IN (...)` for files that have a
version constraint, because that recalls all historical versions of the same
file. This runtime resolution is read-only; it does not update
`knowledge_base_sources` or backfill governance pointers.

### Data Lineage Tracking (moi-core additions)

| Step | workflow_be Equivalent | Workflow Node | moi-core WorkItem | State Usage |
|------|----------------------|---------------|-------------------|-------------|
| 5 | chunks stored in embedding_results table | `write_parsed_docset` | `moi:files.write_documents` | reads `state.embedded_documents` (saved at step 3); saves `parsed_file_id` |
| 6 | `CRUDCatalogService.create_file` | `register_raw_asset` | `moi:data.asset.register` | saves `raw_asset_id` |
| 7 | (no direct equivalent) | `register_parsed_asset` | `moi:data.asset.register` | reads `state.parsed_file_id`; saves `parsed_asset_id` |
| 8 | (no direct equivalent) | `link_parsed_from_raw` | `moi:data.asset.link` | reads `state.raw_asset_id`, `state.parsed_file_id` |
| 9 | (no direct equivalent) | `register_vector_asset` | `moi:data.asset.register` | saves `vector_asset_id` |
| 10 | (no direct equivalent) | `link_indexed_from_parsed` | `moi:data.asset.link` | reads `state.vector_asset_id`, `state.parsed_file_id` |

## State Usage

The workflow uses the [Workflow State](../guide/DSL.md#9-工作流-state跨节点共享状态) feature to pass
data between non-adjacent steps. The key challenge is that `vector_write` (step 4) consumes
the embedded documents from `.data`, so `write_parsed_docset` (step 5) cannot read them via
`.data`. Instead, `embedding_generate` saves them to `state.embedded_documents` first.

State flow:
- Step 3 (`embedding_generate`): saves `embedded_documents` → used by step 5
- Step 5 (`write_parsed_docset`): saves `parsed_file_id` → used by steps 7, 8, 10
- Step 6 (`register_raw_asset`): saves `raw_asset_id` → used by step 8
- Step 9 (`register_vector_asset`): saves `vector_asset_id` → used by step 10

## Notes

- In workflow_be, `RootNode` (FileTypeRouter) routes files by MIME type to
  specific converters. In moi-core, `moi:parser.convert.document.rich` handles
  MIME routing and conversion internally via backend parser endpoints.
- `ChunkNode` uses `EnhancedDocumentSplitter` with `enable_level_based_split=true`
  by default. The moi-core equivalent passes this as a template variable.
- workflow_be's `DocumentCleanerNode` (MoiDocumentCleaner) is an optional node
  not included in the default pipeline. It can be added to the moi-core workflow
  when a `moi:parser.clean.documents` WorkItem is implemented.
- Steps 5-10 (data lineage) are new in moi-core. In workflow_be, file registration
  is handled by `job_consumer.process_file` calling `CRUDCatalogService.create_file`
  after the Haystack pipeline completes. The moi-core workflow makes this explicit
  as composable WorkItem steps.
- The `embedding_dimension` default is 1024 (matching BAAI/bge-m3 output dimension).
- The workflow YAML is at `moi-core/workflows/rag-ingest-default-v1.yaml`.
