# Parse V3 Image Semantic Surfaces

This is the canonical policy for image-owned text after Parse V3 assemble.
Implementation lives in `extractImageSemantics` / `project`
(`moi-core/workers/go-worker/pkg/workitems/image_semantic_projection.go`).
Issue #14769 tracks the contract; #13765 tracks Writer source-Markdown
annotation partial matches (D-12). Do not treat an API `text` pass as a
Writer or `document_visual` pass.

The producer (`image_understanding`) writes OCR and generated caption into
IMAGE metadata. Assemble does not copy those fields into `Document.Content`.
That matches the old `workflow_be` *external* API (split `image_ocr` /
`image_caption`, then drop `content`), not the old internal Haystack packed
string. Downstream surfaces derive from the same structured fields; they do
not all emit the same blob.

## Fields

| Field | Owner | Meaning |
|---|---|---|
| `ocr` (`ocr_text`, `OCR`) | generated | Characters read from the image |
| `figure_caption` | source | MinerU/PDFium typed figure caption, materialized as its own TEXT/LIST block |
| `caption` (`Caption`) | generated | VLM description |
| `figure_no` | source | Parsed from a typed figure caption (`图 9-2` / `Figure 9-2`). Never inferred from generated `caption` |
| `caption_block_uuid` / `caption_for` | source | Reciprocal relation markers. Any present marker fail-closes if the pair is incomplete |

Retrieval projection order is always:

**OCR → source `figure_caption` → generated `caption`**

Values are trimmed. Dedup is exact and case-sensitive. The first occurrence
wins. A source figure caption that is represented by an earlier identical
value still counts as consumed, so its reciprocal TEXT/LIST block is not
emitted again.

There is no image-specific length budget. Ordinary chunks inherit
`moi:parser.split.documents.length` `chunk_size` (default 800 runes) and
truncate the projected string after the OCR-first join.

## Surface policy

`Content` below means the assembled IMAGE `documents[].content`. It stays
empty unless a later stage wrote authoritative local text into the block.

| Input | API `documents.Content` | API top-level `text` | Ordinary chunk (empty Content) | `document_visual` | Writer `.md` (no asset) | Writer `.md` (asset) | Writer `_parse.json` |
|---|---|---|---|---|---|---|---|
| OCR-only | `""` | empty Content → OCR stays out | OCR; truncate at `chunk_size` | `objects[].ocr` + context in the same order | OCR body | `![]()` + `**OCR**` | `ocr` set |
| caption-only | `""` | empty Content → caption stays out | generated caption | `objects[].caption` + context | caption body | `![]()` + `**Caption**` | `caption` set |
| OCR + caption | `""` | empty Content → both stay out | `OCR\n\nfigure_caption\n\ncaption`, then truncate | structured fields + same-order context | same join | `![]()` + labeled OCR / plain figure caption / labeled Caption | both fields |
| all empty | `""` | nothing | `[image]` | empty semantic fields | empty body | `![]()` only | no ocr/caption |
| typed `figure_no` | still `""`; number stays in metadata | includes the reciprocal TEXT caption block, not OCR or generated caption | does not splice `figure_no` into Content | `objects[].figure_no` plus the reciprocal pair | source `figure_caption` may appear as body | source figure caption is unlabeled body | relation metadata preserved |

API `text` is `projectDocumentsPlainText` over assembled documents. It skips
only `non_body` furniture (headers, footers, discarded blocks) and joins every
remaining document's non-empty `Content`, regardless of type. Assembled
IMAGE/TABLE `Content` is empty, so OCR/caption metadata do not appear in
`text`. If a later stage writes authoritative IMAGE `Content`, that text is
included. This is not `plainTextBlock`, which is an assemble-stage type filter
and is not the public `text` path.

Ordinary chunks are retrieval text. An empty IMAGE `Content` projects
metadata; a non-empty IMAGE `Content` is authoritative and is not replaced.
A non-image document with `ocr` / `caption` metadata must not project.

`document_visual` keeps structured fields and builds object context with
`\n` (not `\n\n`) in the same order, then local text.

Writer without an image asset joins the retrieval projection into Markdown.
Writer with an asset uses empty-alt `![](./images/...)` plus labeled `**OCR**`
/ `**Caption**` sections; a source figure caption stays unlabeled. See
`go-worker-component-mapping.md` for the asset-backed layout.

## Out of scope

- Writer source-Markdown annotation partial match (warn-and-succeed) is #13765.
- Filling every surface with one long string is not the contract.
- Filling IMAGE `Content`, or changing `projectDocumentsPlainText` so public
  `text` carries image metadata, is a compatibility change and needs its own
  PR note.
