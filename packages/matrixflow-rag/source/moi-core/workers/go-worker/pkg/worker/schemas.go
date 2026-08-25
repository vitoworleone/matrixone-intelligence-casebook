package worker

import moi "github.com/matrixflow/moi-core/go-sdk"

func schemaAnyObject() *moi.SchemaBuilder {
	return moi.NewSchema().AdditionalProperties(true)
}

func schemaStringMap() *moi.SchemaBuilder {
	return moi.NewSchema().AdditionalPropertiesSchema(moi.StringSchema())
}

func schemaNullableObject() *moi.SchemaBuilder {
	return moi.AnySchema().
		AnyOf(schemaAnyObject(), moi.NewSchema().Type("null")).
		Example(map[string]interface{}{}).
		Range("JSON object or null")
}

func schemaNullableStringMap() *moi.SchemaBuilder {
	return moi.AnySchema().AnyOf(schemaStringMap(), moi.NewSchema().Type("null"))
}

func schemaNullableObjectArray() *moi.SchemaBuilder {
	return moi.AnySchema().AnyOf(moi.ArraySchema().Items(schemaAnyObject()), moi.NewSchema().Type("null"))
}

func schemaNullableInteger() *moi.SchemaBuilder {
	return moi.AnySchema().AnyOf(moi.IntegerSchema(), moi.NewSchema().Type("null"))
}

func schemaNullableAny() *moi.SchemaBuilder {
	return moi.AnySchema().AnyOf(moi.AnySchema(), moi.NewSchema().Type("null"))
}

func schemaDocumentVisualManifest() *moi.SchemaBuilder {
	return schemaNullableObject().
		Description("Document visual manifest with parsed pages, visual objects, extracted text, and image file references. Null means no manifest is available for this branch.").
		Example(map[string]interface{}{
			"schema_version": "document_visual_manifest.v1",
			"profile":        "industrial_drawing_v1",
			"pages": []interface{}{
				map[string]interface{}{"page_number": 1},
			},
		}).
		Range("Object following the document visual manifest contract, or null.")
}

func schemaDocumentVisualManifestArray() *moi.SchemaBuilder {
	return schemaNullableObjectArray().
		Description("Document visual manifests for multi-file or multi-source runs. Null means no manifests are available for this branch.").
		Example([]interface{}{
			map[string]interface{}{
				"schema_version": "document_visual_manifest.v1",
				"profile":        "industrial_drawing_v1",
				"pages": []interface{}{
					map[string]interface{}{"page_number": 1},
				},
			},
		}).
		Range("Array of document visual manifest objects, or null.")
}

func schemaDataAsset() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Registered data asset. Use asset_id in subsequent moi:data.asset.link calls to build lineage.").
		Property("id", moi.IntegerSchema().Description("Auto-increment database primary key.")).
		Property("asset_id", moi.StringSchema().Description("Unique asset identifier.")).
		Property("asset_type", moi.StringSchema().Description("Typed asset namespace.").Enum("file", "vector_index", "table")).
		Property("asset_ref", moi.StringSchema().Description("Type-local asset reference.")).
		Property("name", moi.StringSchema().Description("Human-readable asset name.")).
		Property("volume_id", moi.IntegerSchema().Description("Catalog volume ID, 0 if not volume-scoped.")).
		Property("source", moi.StringSchema().Description("Pipeline/workflow that created this asset.")).
		Property("meta", schemaAnyObject().Description("Arbitrary metadata attached to the asset.")).
		Property("created_at", moi.IntegerSchema()).
		Property("updated_at", moi.IntegerSchema()).
		AdditionalProperties(true)
}

func schemaDataDerivation() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("A derivation link between two data assets, representing one step in the data lineage chain.").
		Property("id", moi.IntegerSchema().Description("Auto-increment database primary key.")).
		Property("root_asset_id", moi.StringSchema().Description("Root/source file family asset ID.")).
		Property("source_asset_id", moi.StringSchema().Description("Source asset ID (upstream).")).
		Property("target_asset_id", moi.StringSchema().Description("Target asset ID (downstream).")).
		Property("kind", moi.StringSchema().Description("Derivation type: parsed_from, indexed_from, extracted_from, or transformed_from.")).
		Property("case_id", moi.StringSchema()).
		Property("producer_workitem_id", moi.StringSchema()).
		Property("recorded_by_workitem_id", moi.StringSchema()).
		Property("parallel_index", moi.IntegerSchema()).
		Property("logical_slot", moi.StringSchema()).
		Property("idempotency_key", moi.StringSchema()).
		Property("meta", schemaAnyObject().Description("Optional metadata for this derivation link.")).
		Property("created_at", moi.IntegerSchema()).
		Property("updated_at", moi.IntegerSchema()).
		AdditionalProperties(true)
}

func schemaParsedManifest() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("id", moi.IntegerSchema()).
		Property("root_asset_id", moi.StringSchema()).
		Property("source_file_id", moi.StringSchema()).
		Property("parsed_asset_id", moi.StringSchema()).
		Property("parsed_file_id", moi.StringSchema()).
		Property("manifest", schemaAnyObject()).
		Property("created_at", moi.IntegerSchema()).
		Property("updated_at", moi.IntegerSchema()).
		AdditionalProperties(true)
}

func schemaFileMetadata() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("id", moi.StringSchema()).
		Property("name", moi.StringSchema()).
		Property("volume_id", moi.IntegerSchema()).
		Property("parent_id", moi.StringSchema()).
		Property("type", moi.IntegerSchema()).
		Property("extension", moi.StringSchema()).
		Property("size", moi.IntegerSchema()).
		Property("hash", moi.StringSchema()).
		Property("path", moi.StringSchema()).
		Property("created_by", moi.StringSchema()).
		Property("updated_by", moi.StringSchema()).
		Property("created_at", moi.IntegerSchema()).
		Property("updated_at", moi.IntegerSchema()).
		AdditionalProperties(true)
}

func schemaSource() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Raw source descriptor produced by catalog/source readers and consumed by parsers. It identifies a file or inline content before it becomes normalized documents.").
		Property("name", moi.StringSchema().Description("Source display name or file name.")).
		Property("mime_type", moi.StringSchema().Description("Detected or declared MIME type used by parser routing.")).
		Property("content", moi.StringSchema().Description("Inline UTF-8 content for text-like sources. Binary files usually use file_id or content_base64 instead.")).
		Property("content_base64", moi.StringSchema().Description("Inline base64 bytes for binary sources when no file_id is available.")).
		Property("file_id", moi.StringSchema().Description("Catalog/workspace file ID. Parsers can fetch the file bytes from this ID.")).
		AdditionalProperties(true)
}

func schemaDocument() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Normalized document chunk or parsed document. Most document-processing nodes consume and produce arrays of this shape.").
		Property("id", moi.StringSchema().Description("Stable document or chunk identifier when available.")).
		Property("content", moi.StringSchema().Description("Main text content used by split, extraction, indexing, and save nodes.")).
		Property("type", moi.StringSchema().Description("Document kind, such as text, markdown, table, image_caption, or chunk.")).
		Property("metadata", schemaAnyObject().Description("Source and parser metadata, for example file_id, file_name, page_number, heading path, chunk index, or table metadata.")).
		Property("embedding", moi.ArraySchema().Items(moi.NumberSchema()).Description("Optional vector embedding attached by embedding/indexing nodes.")).
		AdditionalProperties(true)
}

func schemaSourcesInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("sources", moi.ArraySchema().Items(schemaSource()).
			Description("Source file descriptors bound from an upstream data-read or routing node.")).
		Property("options", schemaAnyObject().
			Description("Runtime options for the parser or converter node.")).
		Required("sources").
		AdditionalProperties(true)
}

func schemaImageParseInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("sources", moi.ArraySchema().Items(schemaSource()).
			Description("Source image descriptors bound from an upstream data-read or routing node.")).
		Property("options", moi.NewSchema().
			Description("Image parsing options.").
			Property("image_process_type", moi.AnySchema().
				AnyOf(
					moi.ArraySchema().Items(moi.StringSchema().Enum("ocr", "caption")),
					moi.StringSchema().Enum("ocr", "caption"),
				).
				Default([]string{"ocr", "caption"}).
				Range("array of ocr/caption values, or legacy scalar ocr/caption").
				Description("Image processing methods. OCR and caption both use the Parse V3 visual-language-model backend. A legacy scalar value is also accepted.")).
			Property("caption_language", moi.StringSchema().
				Enum("zh", "en").
				Default("zh").
				Description("Output language for image captions. Ignored when caption processing is disabled.")).
			Property("vlm_ocr_model", moi.StringSchema().
				Description("Optional visual LLM model used for both OCR and caption. When omitted, the deployment default is used.")).
			AdditionalProperties(true)).
		Required("sources").
		AdditionalProperties(true)
}

func schemaMediaParseUserInput(kind string) *moi.SchemaBuilder {
	fileField := "media_file"
	if kind == "audio" {
		fileField = "audio_file"
	}
	if kind == "video" {
		fileField = "video_file"
	}
	return moi.NewSchema().
		Description("Media parsing parameters. User-facing fields describe the selected media and parse options; runtime fields keep upstream source binding explicit.").
		Property(fileField, moi.StringSchema().Description("Media file selected from Catalog or produced by an upstream data-read node.")).
		Property("language", moi.StringSchema().Description("Spoken language for transcription. Use auto when the language is unknown.").Enum("auto", "zh", "en")).
		Property("denoise", moi.StringSchema().Description("Audio denoise strategy before transcription.").Enum("auto", "off", "ffmpeg_afftdn")).
		Property("min_silence_duration", moi.NumberSchema().Minimum(0.1).Maximum(2).Default(0.5).Description("Minimum silence duration used as a split point, in seconds.")).
		Property("max_segment_duration", moi.NumberSchema().Minimum(5).Maximum(60).Default(30).Description("Maximum duration of one segment, in seconds.")).
		Property("sources", moi.ArraySchema().Items(schemaSource()).Description("Runtime-bound upstream media sources. This field is hidden from the user form.")).
		Property("execution_context", schemaNullableObject().Description("Optional runtime execution context supplied by the workflow runtime.")).
		Property("options", schemaAnyObject().Description("Runtime parser options derived from user-facing fields.")).
		Required("sources").
		AdditionalProperties(true)
}

func schemaMediaParseUserOutput(kind string) *moi.SchemaBuilder {
	resultField := "media_result"
	if kind == "audio" {
		resultField = "audio_result"
	}
	if kind == "video" {
		resultField = "video_result"
	}
	return moi.NewSchema().
		Description("Media parsing result. User-facing fields summarize the transcription; runtime documents remain available for downstream document-processing nodes.").
		Property("transcript", moi.StringSchema().Description("Full transcription text extracted from the media file.")).
		Property("text", moi.StringSchema().Description("Plain transcript text for generic downstream text consumers.")).
		Property("segments", moi.ArraySchema().Items(moi.NewSchema().
			Property("text", moi.StringSchema().Description("Segment transcription text.")).
			Property("start_seconds", moi.NumberSchema().Description("Segment start time in seconds.")).
			Property("end_seconds", moi.NumberSchema().Description("Segment end time in seconds.")).
			Property("speaker", moi.StringSchema().Description("Speaker label when available.")).
			AdditionalProperties(false)).
			Description("Time-aligned transcription segments.")).
		Property("language", moi.StringSchema().Description("Detected or configured transcription language.")).
		Property("duration_seconds", moi.NumberSchema().Description("Media duration in seconds when available.")).
		Property(resultField, moi.StringSchema().Description("Identifier or display name of the parsed media result.")).
		Property("documents", moi.ArraySchema().Items(schemaDocument()).Description("Runtime normalized documents for downstream split, extraction, indexing, save, or custom document-processing nodes.")).
		Required("documents").
		AdditionalProperties(true)
}

func schemaCatalogResourceRef() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Catalog resource reference selected by UI or returned by resource resolver. It can point to a volume/folder, one file, multiple files, or a data asset.").
		Property("kind", moi.StringSchema().Description("Resource kind, such as volume, folder, file, files, table, or asset.")).
		Property("catalog_id", moi.IntegerSchema().Minimum(1).Description("Catalog numeric ID.")).
		Property("database_id", moi.IntegerSchema().Minimum(1).Description("Database numeric ID inside the catalog.")).
		Property("database_name", moi.StringSchema().Description("Database name when known from the selected catalog path.")).
		Property("volume_id", moi.IntegerSchema().Minimum(1).Description("Volume/folder numeric ID used as source or destination location.")).
		Property("volume_ids", moi.ArraySchema().Items(moi.IntegerSchema().Minimum(1)).Description("Multiple volume IDs for cross-volume expansion. Used with file_types for the by-type file scope contract; by-file selection is single-volume only.")).
		Property("file_id", moi.StringSchema().Description("Single file ID.")).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Multiple selected file IDs.")).
		Property("file_name", moi.StringSchema().Description("File name when known.")).
		Property("asset_id", moi.StringSchema().Description("Data asset ID when the reference comes from lineage/catalog assets.")).
		Property("raw_file_id", moi.StringSchema().Description("Original source file ID associated with an asset.")).
		Property("table_id", moi.IntegerSchema().Minimum(1).Description("Table numeric ID inside the catalog database.")).
		Property("table_name", moi.StringSchema().Description("Table name when the resource is a structured table.")).
		Property("path", moi.StringSchema().Description("Catalog path or folder path when available.")).
		Property("name", moi.StringSchema().Description("Display name for the selected resource.")).
		AdditionalProperties(true)
}

func schemaNullableCatalogResourceRef() *moi.SchemaBuilder {
	return moi.AnySchema().
		AnyOf(
			schemaCatalogResourceRef(),
			moi.NewSchema().Type("null").Description("No Catalog resource reference is bound."),
		).
		Example(nil).
		Range("Catalog resource reference object or null")
}

func catalogSourceValueAlternatives() []*moi.SchemaBuilder {
	return []*moi.SchemaBuilder{
		moi.NewSchema().Property("file_id", schemaCatalogSourceID()).Required("file_id"),
		moi.NewSchema().Property("file_ids", schemaCatalogSourceIDs()).Required("file_ids"),
		moi.NewSchema().Property("volume_id", moi.IntegerSchema().Minimum(1)).Required("volume_id"),
		moi.NewSchema().Property("volume_ids", moi.ArraySchema().Items(moi.IntegerSchema().Minimum(1)).MinItems(1)).Required("volume_ids"),
		moi.NewSchema().Property("asset_id", schemaCatalogSourceID()).Required("asset_id"),
		moi.NewSchema().Property("raw_file_id", schemaCatalogSourceID()).Required("raw_file_id"),
	}
}

func schemaCatalogSourceID() *moi.SchemaBuilder {
	return moi.StringSchema().MinLength(1).Pattern(`\S`)
}

func schemaCatalogSourceIDs() *moi.SchemaBuilder {
	return moi.ArraySchema().Items(schemaCatalogSourceID()).MinItems(1)
}

func catalogSourceSelectorAlternatives() []*moi.SchemaBuilder {
	return append([]*moi.SchemaBuilder{moi.NewSchema().Required("source_ref")}, catalogSourceValueAlternatives()...)
}

func schemaCatalogSourceRef() *moi.SchemaBuilder {
	return schemaCatalogResourceRef().
		AnyOf(catalogSourceValueAlternatives()...)
}

func schemaCatalogSourceReadInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Select files, folders, volumes, or data assets from Catalog as workflow input. Use this as the first node when the workflow starts from user-selected Catalog resources.").
		Property("source_ref", schemaCatalogResourceRef().Description("Primary user-selected Catalog resource. This is normally bound from the generated input form.")).
		Property("file_id", moi.StringSchema().Description("Direct single file ID alternative when source_ref is not used.")).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Direct multiple file IDs alternative when source_ref is not used.")).
		Property("volume_id", moi.IntegerSchema().Minimum(1).Description("Direct volume/folder ID alternative when source_ref is not used.")).
		Property("asset_id", moi.StringSchema().Description("Data asset ID to resolve into underlying files.")).
		Property("raw_file_id", moi.StringSchema().Description("Original raw file ID associated with a data asset.")).
		Property("file_types", moi.ArraySchema().Items(moi.StringSchema()).Description("Optional extension filters for volume expansion (e.g. PDF, DOCX). Applied only when expanding volumes; explicit file_id/file_ids selection is not filtered.")).
		Property("read_mode", moi.StringSchema().Description("Optional read strategy. Omit or set files for file/resource expansion. documents is rejected because file content must not be passed through workflow data.")).
		Property("limit", moi.IntegerSchema().Minimum(1).Description("Optional maximum resources/files to return.")).
		Property("page_size", moi.IntegerSchema().Minimum(1).Description("Optional page size for listing folder/volume files.")).
		AdditionalProperties(true)
}

func schemaCatalogSourceReadV2Input() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Select files, folders, volumes, or data assets from Catalog as workflow input. Use this version for new workflows. This v2 contract rejects missing or blank selectors before execution.").
		Property("source_ref", schemaCatalogSourceRef().Description("Primary user-selected Catalog resource. This is normally bound from the generated input form.")).
		Property("file_id", schemaCatalogSourceID().Description("Direct single file ID alternative when source_ref is not used.")).
		Property("file_ids", schemaCatalogSourceIDs().Description("Direct multiple file IDs alternative when source_ref is not used.")).
		Property("volume_id", moi.IntegerSchema().Minimum(1).Description("Direct volume/folder ID alternative when source_ref is not used.")).
		Property("asset_id", schemaCatalogSourceID().Description("Data asset ID to resolve into underlying files.")).
		Property("raw_file_id", schemaCatalogSourceID().Description("Original raw file ID associated with a data asset.")).
		Property("file_types", moi.ArraySchema().Items(moi.StringSchema()).Description("Optional extension filters for volume expansion (e.g. PDF, DOCX). Applied only when expanding volumes; explicit file_id/file_ids selection is not filtered.")).
		Property("read_mode", moi.StringSchema().Description("Optional read strategy. Omit or set files for file/resource expansion. documents is rejected because file content must not be passed through workflow data.")).
		Property("limit", moi.IntegerSchema().Minimum(1).Description("Optional maximum resources/files to return.")).
		Property("page_size", moi.IntegerSchema().Minimum(1).Description("Optional page size for listing folder/volume files.")).
		AnyOf(catalogSourceSelectorAlternatives()...).
		AdditionalProperties(true)
}

func schemaCatalogSourceV2File() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Canonical file descriptor emitted by moi:catalog.source.read.v2 regardless of whether the source was selected directly, through a volume, or through a data asset.").
		Property("id", schemaCatalogSourceID().Description("Stable logical file identifier; equal to file_id.")).
		Property("file_id", schemaCatalogSourceID().Description("Catalog file ID.")).
		Property("name", schemaCatalogSourceID().Description("File name used for workflow routing and parser selection.")).
		Property("file_name", schemaCatalogSourceID().Description("Catalog or volume file display name.")).
		Property("original_name", moi.StringSchema().Description("Original uploaded file name when Catalog metadata is available.")).
		Property("volume_id", moi.IntegerSchema().Description("Containing volume ID when selected through a volume or trigger.")).
		Property("path", moi.StringSchema().Description("Virtual path inside the containing volume when available.")).
		Property("size", moi.IntegerSchema().Description("File size in bytes when Catalog metadata is available.")).
		Required("id", "file_id", "name", "file_name").
		AdditionalProperties(false)
}

func schemaCatalogSourceReadOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Catalog read result. `sources` and `file_ids` identify selected files for downstream parser or import nodes; file content is not returned in workflow data.").
		Property("resource", schemaCatalogResourceRef().Description("Resolved source resource reference.")).
		Property("sources", moi.ArraySchema().Items(schemaSource()).Description("Raw source descriptors for parsers such as moi:document.parse.")).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Selected file IDs. Use for file-based nodes or imports that consume file_ids.")).
		Property("files", moi.ArraySchema().Items(schemaAnyObject()).Description("Resolved file metadata records.")).
		Property("count", moi.IntegerSchema().Description("Number of selected or resolved resources/files.")).
		AdditionalProperties(true)
}

func schemaCatalogSourceReadV2Output() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Catalog source v2 result. `sources`, `file_ids`, and canonical `files` identify selected files for downstream parser, import, or routing nodes; file content is not returned in workflow data.").
		Property("resource", schemaCatalogResourceRef().Description("Resolved source resource reference.")).
		Property("sources", moi.ArraySchema().Items(schemaSource()).Description("Raw source descriptors for parsers such as moi:document.parse.")).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Selected file IDs. Use for file-based nodes or imports that consume file_ids.")).
		Property("files", moi.ArraySchema().Items(schemaCatalogSourceV2File()).Description("Canonical selected-file metadata. Every selector path exposes file_id, name, and file_name with the same shape.")).
		Property("count", moi.IntegerSchema().Description("Number of selected or resolved resources/files.")).
		AdditionalProperties(true)
}

func schemaCatalogPDFPrepareInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Internal input for the Catalog PDF preparation stage.").
		Property("sources", moi.ArraySchema().Items(schemaSource()).MinItems(1).MaxItems(1).Description("Exactly one Catalog source from moi:catalog.source.read.")).
		Required("sources").
		AdditionalProperties(false)
}

func schemaCatalogPDFPrepareOutput() *moi.SchemaBuilder {
	fastRoute := moi.NewSchema().
		Property("route", moi.StringSchema().Enum("pdf_fast")).
		Property("documents", moi.ArraySchema().Items(schemaDocument()).MinItems(1)).
		Required("route", "documents")
	visualRoute := moi.NewSchema().
		Property("route", moi.StringSchema().Enum("pdf_visual")).
		Property("visual_pages", moi.ArraySchema().Items(moi.IntegerSchema().Minimum(1)).MinItems(1).UniqueItems(true)).
		Property("page_selector", moi.StringSchema().MinLength(1)).
		Required("route", "visual_pages", "page_selector")
	standardRoute := moi.NewSchema().
		Property("route", moi.StringSchema().Enum("standard")).
		Required("route")
	return moi.NewSchema().
		Description("Internal PDF route decision plus deterministic native documents.").
		Property("route", moi.StringSchema().Enum("standard", "pdf_fast", "pdf_visual")).
		Property("documents", moi.ArraySchema().Items(schemaDocument())).
		Property("visual_pages", moi.ArraySchema().Items(moi.IntegerSchema().Minimum(1)).UniqueItems(true)).
		Property("page_selector", moi.StringSchema()).
		Property("page_count", moi.IntegerSchema().Minimum(0)).
		Property("table_count", moi.IntegerSchema().Minimum(0)).
		Property("pages_with_table", moi.IntegerSchema().Minimum(0)).
		AnyOf(standardRoute, fastRoute, visualRoute).
		Required("route").
		AdditionalProperties(false)
}

func schemaCatalogPDFMergeInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Internal input that replaces selected native PDF pages with rich-parser documents.").
		Property("base_documents", moi.ArraySchema().Items(schemaDocument())).
		Property("visual_documents", moi.ArraySchema().Items(schemaDocument()).MinItems(1)).
		Property("visual_pages", moi.ArraySchema().Items(moi.IntegerSchema().Minimum(1)).MinItems(1).UniqueItems(true)).
		Required("base_documents", "visual_documents", "visual_pages").
		AdditionalProperties(false)
}

func schemaCatalogFilesManifestPlanInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Enumerate Catalog files into stable JSONL shard manifest files. The node writes only file references and ordinals to Catalog; it does not download source file content. Provide exactly one source group: source_ref, sources, or file_id/file_ids.").
		Property("source_ref", schemaCatalogResourceRef().Description("Catalog source selected by the user. Use a volume, folder, file, or file list to create a stable shard manifest.")).
		Property("target_ref", schemaCatalogResourceRef().Description("Optional Catalog volume/folder that receives generated JSONL manifest files.")).
		Property("sources", moi.ArraySchema().Items(schemaSource()).Description("Runtime-bound source descriptors. Each source must contain file_id.")).
		Property("file_id", moi.StringSchema().Description("Single Catalog file ID to include when no source_ref or sources are provided.")).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Catalog file IDs to include when no source_ref or sources are provided.")).
		Property("output_prefix", moi.StringSchema().MinLength(1).Description("Filename prefix for generated JSONL manifest files.")).
		Property("output_volume_id", moi.IntegerSchema().Minimum(1).Description("Runtime/API volume/folder ID alternative to target_ref for generated manifest files.")).
		Property("page_size", moi.IntegerSchema().Minimum(1).Description("Catalog volume listing page size. Required by the handler when source_ref points to a volume.")).
		Property("shard_total", moi.IntegerSchema().Minimum(1).Maximum(128).Description("Number of shard manifest files to create. This is also the intended parallel branch count for downstream processing.")).
		AnyOf(
			moi.NewSchema().Required("source_ref"),
			moi.NewSchema().Required("sources"),
			moi.NewSchema().Required("file_id"),
			moi.NewSchema().Required("file_ids"),
		).
		Required("output_prefix", "shard_total").
		AdditionalProperties(true)
}

func schemaCatalogFilesManifestPlanOutput() *moi.SchemaBuilder {
	manifestFile := moi.NewSchema().
		Property("shard_index", moi.IntegerSchema().Description("Zero-based shard index.")).
		Property("file_id", moi.StringSchema().Description("Catalog JSONL manifest file ID for this shard.")).
		Property("file_name", moi.StringSchema().Description("Catalog JSONL manifest file name for this shard.")).
		Property("count", moi.IntegerSchema().Description("Number of source file records written to this shard.")).
		Required("shard_index", "file_id", "file_name", "count").
		AdditionalProperties(false)
	return moi.NewSchema().
		Description("Stable Catalog file shard manifest output. Downstream parallel branches consume one manifest_files entry or one manifest_file_ids item per branch.").
		Property("manifest_file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Catalog JSONL manifest file IDs in shard_index order.")).
		Property("manifest_files", moi.ArraySchema().Items(manifestFile).Description("Per-shard manifest descriptors in shard_index order.")).
		Property("output_volume_id", moi.IntegerSchema().Description("Destination volume/folder ID for generated manifest files when configured.")).
		Property("shard_total", moi.IntegerSchema().Description("Total number of manifest shards created.")).
		Property("count", moi.IntegerSchema().Description("Total number of Catalog file records enumerated.")).
		Required("manifest_file_ids", "manifest_files", "shard_total", "count").
		AdditionalProperties(true)
}

func schemaCatalogListInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("List one page of Catalog entries in the current workspace. Use page_token from the previous output to fetch the next page.").
		Property("page_size", moi.IntegerSchema().Minimum(1).Description("Number of Catalog entries to fetch in this page.")).
		Property("page_token", moi.StringSchema().Description("Optional cursor from the previous output next_page_token. Omit for the first page.")).
		Property("retry_count", moi.StringSchema().Description("Optional loop counter from the previous output retry_count. Omit for the first page.")).
		Required("page_size").
		AdditionalProperties(false)
}

func schemaCatalogListOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("One page of Catalog entries plus the cursor for the next page. Empty next_page_token means there is no next page.").
		Property("items", moi.ArraySchema().Items(schemaAnyObject()).Description("Catalog entries returned in this page.")).
		Property("total", moi.IntegerSchema().Description("Total number of Catalog entries when reported by the backend.")).
		Property("next_page_token", moi.StringSchema().Description("Cursor for the next page. Empty means no next page.")).
		Property("count", moi.IntegerSchema().Description("Number of entries in items.")).
		Property("retry_count", moi.IntegerSchema().Description("Loop counter incremented by this page request.")).
		Required("items", "next_page_token", "retry_count").
		AdditionalProperties(false)
}

func schemaCatalogSinkWriteInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Save one upstream payload into Catalog. Choose exactly the payload field that matches the upstream output: documents, rows/columns, text, json, file_id, or file_ids. Use this as a final sink when the user wants a downloadable Catalog file.").
		Property("target_ref", schemaCatalogResourceRef().Description("Destination Catalog location selected by the user, usually a volume or folder. Normally bound from the generated input form.")).
		Property("volume_id", moi.IntegerSchema().Minimum(1).Description("Destination volume/folder ID alternative when target_ref is not used.")).
		Property("payload_type", moi.StringSchema().Description("Optional explicit payload type. Usually omitted because the worker infers it from the provided payload field.")).
		Property("format", moi.StringSchema().Description("Output format. When rows are saved as CSV with columns, the columns are emitted as the first header row before data rows.")).
		Property("trigger_context", schemaCatalogResourceRef().Description("Optional trigger/source context used to derive output location or name.")).
		Property("documents", moi.ArraySchema().Items(schemaDocument()).Description("Document array to save as document output or parse artifacts. Do not use this field for tabular rows; use rows/columns instead.")).
		Property("md_file_id", moi.StringSchema().Description("Markdown artifact file ID from document.parse. When present with documents, catalog.sink.write saves a parse ZIP.")).
		Property("layout_file_id", moi.StringSchema().Description("Layout JSON artifact file ID from document.parse. When present with documents, catalog.sink.write saves a parse ZIP.")).
		Property("images_file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Optional image artifact file IDs to include in parse ZIP output.")).
		Property("download_includes", moi.ArraySchema().Items(moi.StringSchema()).Description("Optional parse ZIP include filter: md, json, images, tables.")).
		Property("extraction_result", moi.AnySchema().Description("Optional structured extraction result to bundle into parse ZIP as {stem}_extract.json when documents are provided.").Example(map[string]interface{}{"title": "example"}).Range("Any valid JSON value: object, array, string, number, boolean, or null.")).
		Property("rows", moi.ArraySchema().Items(moi.ArraySchema().Items(moi.AnySchema())).Description("Tabular rows to save. With columns present, catalog.sink.write writes CSV and emits the columns as the first header row before data rows.")).
		Property("columns", moi.ArraySchema().Items(moi.StringSchema()).Description("Column names for rows. When rows are written as CSV, these columns are written as the first header row; downstream CSV import should skip that header row, for example moi:connector.s3-mo start_row=1.")).
		Property("text", moi.StringSchema().Description("Plain text or markdown string to write as a text-like file.")).
		Property("json", moi.AnySchema().Description("JSON payload to write when payload_type or format is json.").Example(map[string]interface{}{"items": []interface{}{map[string]interface{}{"id": "row-1"}}}).Range("Any valid JSON value: object, array, string, number, boolean, or null.")).
		Property("file_id", moi.StringSchema().Description("Existing single file ID to copy/register into the destination.")).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Existing file IDs to copy/register into the destination. Use this when upstream created files instead of raw content.")).
		Property("register_asset", moi.BooleanSchema().Description("Whether to register the saved file as a data asset.")).
		Property("asset_id", moi.StringSchema().Description("Optional existing asset ID to associate with the saved output.")).
		Property("name", moi.StringSchema().Description("Output file name. If omitted, the worker derives a name from upstream provenance or defaults.")).
		Property("output_file_name", moi.StringSchema().Description("Explicit output file name. This is used as the saved Catalog file name and takes precedence over name.")).
		Property("source", moi.StringSchema().Description("Optional logical source/pipeline name recorded with output metadata.")).
		Property("meta", schemaAnyObject().Description("Optional metadata attached to the saved output.")).
		AdditionalProperties(true)
}

func schemaCatalogSinkWriteOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Catalog save result. Downstream nodes that need produced files should consume file_id or file_ids.").
		Property("result_ref", schemaCatalogResourceRef().Description("Reference to the saved Catalog output.")).
		Property("file_id", moi.StringSchema().Description("Primary saved file ID.")).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("All saved or registered file IDs.")).
		Property("file_name", moi.StringSchema().Description("Primary saved file name.")).
		Property("volume_id", moi.IntegerSchema().Description("Destination volume/folder ID.")).
		Property("asset", schemaAnyObject().Description("Registered data asset metadata when register_asset is enabled.")).
		Property("count", moi.IntegerSchema().Description("Number of saved records/files, depending on payload type.")).
		Property("size", moi.IntegerSchema().Description("Saved file size in bytes when available.")).
		AdditionalProperties(true)
}

func schemaEmailArchiveETLInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Parse RFC822/MIME email archive files into five relational table payloads or shard-local Catalog CSV files. Provide exactly one source group: source_ref, sources, file_id/file_ids, or manifest_file_id/manifest_file_ids. Path index fields are applied to each source name split by '/' so maildir-style archives can expose owner, mailbox, and message name columns without hard-coded dataset rules.").
		Property("source_ref", schemaCatalogResourceRef().Description("Catalog source selected by the user. Use a volume, folder, file, or file list that contains RFC822/MIME email files.")).
		Property("target_ref", schemaCatalogResourceRef().Description("Optional Catalog volume/folder that receives generated CSV files when output_mode is catalog_files.")).
		Property("sources", moi.ArraySchema().Items(schemaSource()).Description("Runtime-bound source descriptors. Each source must contain content, content_base64, or file_id.")).
		Property("file_id", moi.StringSchema().Description("Single Catalog file ID to parse when no source_ref or sources are provided.")).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Catalog file IDs to parse when no source_ref or sources are provided.")).
		Property("manifest_file_id", moi.StringSchema().Description("Catalog JSONL manifest file ID produced by moi:catalog.files.manifest. Use this to process a stable shard in a parallel workflow.")).
		Property("manifest_file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Catalog JSONL manifest file IDs produced by moi:catalog.files.manifest. Each manifest record must contain file_id and ordinal.")).
		Property("original_message_marker_regex", moi.StringSchema().MinLength(1).Description("Regular expression that marks quoted/original message blocks in the email body. The worker does not hard-code a language-specific marker; provide the archive's marker pattern explicitly.")).
		Property("default_charset", moi.StringSchema().MinLength(1).Description("Charset label used when a text/plain MIME part does not declare a charset. Use a WHATWG/IANA label such as utf-8 or latin-1.")).
		Property("max_email_bytes", moi.IntegerSchema().Minimum(1).Description("Maximum bytes to read for one email file. The handler fails if any source email exceeds this limit.")).
		Property("owner_path_index", moi.IntegerSchema().Description("Path segment index for the owner/account column after splitting the source name by '/'. Negative indexes count from the end.")).
		Property("mailbox_start_index", moi.IntegerSchema().Description("First path segment index included in the mailbox column after splitting the source name by '/'. Negative indexes count from the end; the range ends before message_path_index.")).
		Property("message_path_index", moi.IntegerSchema().Description("Path segment index for the message_name column after splitting the source name by '/'. Use -1 for the last path segment.")).
		Property("output_mode", moi.StringSchema().Enum("rows", "catalog_files").Description("Output mode. Use rows only for small samples. Parallel workflow execution requires catalog_files so each shard writes CSV files and returns file IDs.")).
		Property("output_prefix", moi.StringSchema().Description("Filename prefix for catalog_files output. Required when output_mode is catalog_files.")).
		Property("output_volume_id", moi.IntegerSchema().Minimum(1).Description("Runtime/API volume/folder ID alternative to target_ref for generated CSV files.")).
		Property("page_size", moi.IntegerSchema().Minimum(1).Description("Catalog volume listing page size. Required by the handler when source_ref points to a volume.")).
		Property("shard_index", moi.IntegerSchema().Minimum(0).Description("Runtime shard index. Usually injected from workflow parallel runtime; provide together with shard_total only for direct API sharding.")).
		Property("shard_total", moi.IntegerSchema().Minimum(1).Description("Runtime shard count. Usually injected from workflow parallel runtime; provide together with shard_index only for direct API sharding.")).
		Required("original_message_marker_regex", "default_charset", "max_email_bytes", "owner_path_index", "mailbox_start_index", "message_path_index", "output_mode").
		AdditionalProperties(true)
}

func schemaEmailArchiveETLOutput() *moi.SchemaBuilder {
	tableRows := func() *moi.SchemaBuilder {
		return moi.ArraySchema().Items(moi.ArraySchema().Items(moi.AnySchema()))
	}
	tableColumns := func() *moi.SchemaBuilder {
		return moi.ArraySchema().Items(moi.StringSchema())
	}
	return moi.NewSchema().
		Description("Five table payloads extracted from an email archive. In rows mode, each *_columns field pairs with the matching *_rows field. In catalog_files mode, each shard writes CSV files and returns the corresponding file IDs.").
		Property("email_columns", tableColumns().Description("Columns for the email path/provenance table: id, owner, mailbox, message_name, source_file_id, source_name.")).
		Property("email_rows", tableRows().Description("Rows for the email path/provenance table.")).
		Property("email_info_columns", tableColumns().Description("Columns for parsed email headers and body: id, MessageId, Date, Subject, From, To, XFrom, XTo, Body.")).
		Property("email_info_rows", tableRows().Description("Rows for parsed email headers and body.")).
		Property("email_to_columns", tableColumns().Description("Columns for split To recipients: id, NthTo, To.")).
		Property("email_to_rows", tableRows().Description("Rows for split To recipients.")).
		Property("email_x_to_columns", tableColumns().Description("Columns for split X-To recipients: id, NthXTo, XTo.")).
		Property("email_x_to_rows", tableRows().Description("Rows for split X-To recipients.")).
		Property("email_original_columns", tableColumns().Description("Columns for quoted/original message headers: id, nth, subject, From, To, XFrom, XTo.")).
		Property("email_original_rows", tableRows().Description("Rows for quoted/original message headers.")).
		Property("source_file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Catalog file IDs parsed by this run, in input order when file IDs were used.")).
		Property("output_mode", moi.StringSchema().Enum("rows", "catalog_files").Description("Output mode used by this run.")).
		Property("email_file_id", moi.StringSchema().Description("Catalog CSV file ID for the email path/provenance table in catalog_files mode.")).
		Property("email_info_file_id", moi.StringSchema().Description("Catalog CSV file ID for parsed email headers and body in catalog_files mode.")).
		Property("email_to_file_id", moi.StringSchema().Description("Catalog CSV file ID for split To recipients in catalog_files mode.")).
		Property("email_x_to_file_id", moi.StringSchema().Description("Catalog CSV file ID for split X-To recipients in catalog_files mode.")).
		Property("email_original_file_id", moi.StringSchema().Description("Catalog CSV file ID for quoted/original message headers in catalog_files mode.")).
		Property("output_file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("All generated Catalog CSV file IDs in stable table order.")).
		Property("output_file_names", moi.ArraySchema().Items(moi.StringSchema()).Description("Generated Catalog CSV file names in stable table order.")).
		Property("output_volume_id", moi.IntegerSchema().Description("Destination volume/folder ID for generated CSV files when configured.")).
		Property("shard_index", moi.IntegerSchema().Description("Shard index processed by this run.")).
		Property("shard_total", moi.IntegerSchema().Description("Total shard count for this run.")).
		Property("count", moi.IntegerSchema().Description("Number of email files parsed.")).
		Required("email_columns", "email_info_columns", "email_to_columns", "email_x_to_columns", "email_original_columns", "output_mode", "count").
		AdditionalProperties(true)
}

func schemaDocumentParseInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Parse raw Catalog sources or file IDs into normalized documents. Use this after moi:catalog.source.read when downstream nodes need document content instead of raw file references. Provide at least one of sources, file_ids, file_id, or documents. For user-facing direct/run-form input, use file_ids/file_id so residual planning can render a Catalog file picker; sources and documents are upstream payload alternatives normally bound from catalog.source.read or a prior document-producing node.").
		Property("sources", moi.ArraySchema().Items(schemaSource()).Description("Raw source descriptors, typically saved from moi:catalog.source.read output .sources. Upstream-bound payload; not a standalone run-form entry residual.")).
		Property("documents", moi.ArraySchema().Items(schemaDocument()).Description("Pre-existing documents supplied by an upstream node when parsing is already complete. Upstream-bound payload; not a standalone run-form entry residual.")).
		Property("file_id", moi.StringSchema().Description("Single Catalog file ID for standalone/run-form entry when sources are not bound from upstream.")).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Catalog file IDs for standalone/run-form entry when sources are not bound from upstream.")).
		Property("execution_context", schemaNullableObject().Description("Optional runtime execution context supplied by the workflow runtime.")).
		Property("options", schemaDocumentParseOptions().Description("Parser behavior options. Prefer defaults unless the user explicitly asks for parser pipeline, OCR, table, or page-selection behavior.")).
		AnyOf(
			moi.NewSchema().Required("sources"),
			moi.NewSchema().Required("file_ids"),
			moi.NewSchema().Required("file_id"),
			moi.NewSchema().Required("documents"),
		).
		AdditionalProperties(true)
}

func schemaParseStageRuntimeInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Parse raw Catalog sources or file IDs with the v3 parser runtime into normalized documents. Provide at least one of sources, file_ids, file_id, or documents. For user-facing direct/run-form input, use file_ids/file_id so residual planning can render a Catalog file picker; sources and documents are upstream payload alternatives normally bound from catalog.source.read or a prior document-producing node.").
		Property("sources", moi.ArraySchema().Items(schemaSource()).Description("Raw source descriptors, typically saved from moi:catalog.source.read output .sources. Upstream-bound payload; not a standalone run-form entry residual.")).
		Property("documents", moi.ArraySchema().Items(schemaDocument()).Description("Pre-existing documents supplied by an upstream node when parsing is already complete. Upstream-bound payload; not a standalone run-form entry residual.")).
		Property("file_id", moi.StringSchema().Description("Single Catalog file ID for standalone/run-form entry when sources are not bound from upstream.")).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Catalog file IDs for standalone/run-form entry when sources are not bound from upstream.")).
		Property("execution_context", schemaNullableObject().Description("Optional runtime execution context supplied by the workflow runtime.")).
		Property("options", schemaParseStageRuntimeOptions().Description("V3 parser behavior options.")).
		AnyOf(
			moi.NewSchema().Required("sources"),
			moi.NewSchema().Required("file_ids"),
			moi.NewSchema().Required("file_id"),
			moi.NewSchema().Required("documents"),
		).
		AdditionalProperties(true)
}

func schemaDocumentParseOptions() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Document parsing options. Controls parser pipeline, VLM OCR, table handling, and content processing. Most options only take effect when workflow_parser or enable_parser_pipeline is true.").
		Property("page_selector", moi.StringSchema().
			Description("Optional page range selector, for example 1-3,5,8-10. Empty means all pages.").
			Example(pageSelectorExample).
			Pattern(pageSelectorPattern).
			Default("")).
		Property("vlm_ocr_model", moi.StringSchema().
			Description("Required VLM model name for OCR-based parsing and VLM-assisted parser features (e.g. qwen-vl-ocr). Used for images, scanned PDF pages, table HTML regeneration, title/header-footer detection, and related vision tasks. Must be selected from workspace OCR/VLM models; empty values are rejected.").
			Example("qwen-vl-ocr").
			MinLength(1)).
		Property("enable_parser_pipeline", moi.BooleanSchema().
			Description("Enable the parser pipeline with layout detection, table extraction, and rich document assembly. When false, use basic plain-text extraction. When true (default), enables downstream parser options such as table merge, formula repair, and VLM detection. Use for PDF/DOCX/PPTX documents that need structure-aware parsing.").
			Default(true)).
		Property("workflow_parser", moi.BooleanSchema().
			Description("Equivalent to enable_parser_pipeline. Enables the parser pipeline with layout detection, table extraction, and heading detection. Default false. Set this OR enable_parser_pipeline to true; both have the same effect.").
			Default(false)).
		Property("enable_cross_page_table_merge", moi.BooleanSchema().
			Description("Merge table fragments that span across adjacent PDF pages into a single table. Default false. Requires workflow_parser=true. Use when documents contain large tables that break across pages.").
			Default(false)).
		Property("cast_table_as_image", moi.BooleanSchema().
			Description("Treat detected table regions as images and use VLM for extraction instead of text-based parsing. Default false. Requires workflow_parser=true. Use for complex tables with merged cells, nested structures, or poor OCR quality.").
			Default(false)).
		Property("enable_table_html_regeneration", moi.BooleanSchema().
			Description("Regenerate HTML representation for table blocks during enrichment. Default false. Requires workflow_parser=true. Use when downstream consumers need structured HTML tables rather than plain text.").
			Default(false)).
		Property("enable_table_embedded_image_extraction", moi.BooleanSchema().
			Description("Attach MinerU-emitted image blocks that are geometrically contained in table blocks. Default true. Requires workflow_parser=true. Use when PDFs include images embedded inside table cells.").
			Default(true)).
		Property("unmerge_table_cells", moi.BooleanSchema().
			Description("Unmerge merged table cells into an explicit grid. Default false. Requires workflow_parser=true. Use for tables with merged header/footer cells that need to be normalized into regular rows/columns.").
			Default(false)).
		Property("enable_vlm_title_detection", moi.BooleanSchema().
			Description("Use VLM-assisted title/heading detection for better document structure. Default false. Requires workflow_parser=true. Use for documents where headings are styled (bold, larger font) rather than using explicit heading markup.").
			Default(false)).
		Property("enable_vlm_header_footer_detection", moi.BooleanSchema().
			Description("Use VLM-assisted header/footer detection to identify and filter out repeated page elements (page numbers, running headers). Default false. Requires workflow_parser=true. Use for multi-page documents where headers/footers pollute extracted text.").
			Default(false)).
		Property("enable_formula_repair", moi.BooleanSchema().
			Description("Repair broken math formula blocks during content enrichment. Default false. Requires workflow_parser=true. Use for academic papers, technical documents, or textbooks containing LaTeX/MathML formulas.").
			Default(false)).
		Property("enable_fragment_merge", moi.BooleanSchema().
			Description("Merge adjacent text fragments into coherent paragraphs before assembly. Default true. Requires workflow_parser=true. Disable only when fragment-level granularity is needed for downstream processing.").
			Default(true)).
		Property("enable_paddle_preprocess", moi.BooleanSchema().
			Description("Enable Paddle-based table-region detection as a preprocessing step for PDFs. Default false; MinerU layout handles default parsing when Paddle is not deployed. Requires workflow_parser=true.").
			Default(false)).
		Property("image_process_type", moi.ArraySchema().Items(moi.StringSchema().Enum("ocr", "caption")).
			Description("Image processing methods for the image parser (Kind=image). 'ocr' extracts text from images via OCR engine. 'caption' generates natural-language descriptions of image content via VLM. Can select both [\"ocr\",\"caption\"] for combined output. Default: not set (uses parser defaults). Independent of workflow_parser flag.")).
		Property("max_workers", moi.IntegerSchema().Minimum(1).
			Description("Upper bound for internal worker goroutines in the parser pipeline. Default 16. Requires workflow_parser=true. Increase for large batch processing; decrease to limit resource usage.").
			Default(16)).
		Property("parser_concurrency", moi.IntegerSchema().Minimum(1).
			Description("Max concurrent parser tasks across pipeline stages (intake, preprocess, layout, structure, enrich, assemble). Default 16. Requires workflow_parser=true. Similar to max_workers but controls pipeline-stage-level parallelism.").
			Default(16)).
		Required("vlm_ocr_model").
		AdditionalProperties(true)
}

func schemaParseStageRuntimeOptions() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("V3 parser options.").
		Property("parser_version", moi.StringSchema().Description("Parser runtime version pinned by this node.").Default("v3")).
		Property("vlm_ocr_model", moi.StringSchema().
			Description("Optional VLM model name for image OCR/caption and VLM-assisted stages (e.g. qwen-vl-ocr). Select it when an explicitly configured stage needs a workspace OCR/VLM model.").
			Example("qwen-vl-ocr").
			MinLength(1)).
		Property("parse_tier", moi.StringSchema().Description("Parser quality tier. For PDF, Word, and PowerPoint, Standard is the fixed standard-balanced-v1 pipeline (PDF or Office-to-PDF, then MinerU, with no semantic postprocessing); Native prefers OpenXML where available; Enhanced enables the managed enhancement preset. For spreadsheets, Native is OpenXML-only with no WPS/VLM, Standard runs only explicitly requested enrichments, and Enhanced enables chart text plus WPS visual rendering and image OCR/caption by default.").Enum("native", "standard", "enhanced").Default("standard")).
		Property("page_selector", moi.StringSchema().
			Description("Optional page range selector, for example 1-3,5,8-10. Empty means all pages.").
			Example(pageSelectorExample).
			Pattern(pageSelectorPattern).
			Default("")).
		Property("complex_table", moi.BooleanSchema().Description("Enable complex table parsing. The Enhanced managed preset supplies its default; the generic schema must not materialize it into Standard.")).
		Property("cross_page_table_merge", moi.BooleanSchema().Description("Merge table fragments across adjacent pages. The Enhanced managed preset supplies its default; the generic schema must not materialize it into Standard.")).
		Property("save_as_image", moi.BooleanSchema().Description("Save table or visual regions as images where supported.").Default(false)).
		Property("unmerge_cells", moi.BooleanSchema().Description("Split merged cells into individual cells for row-oriented analysis.").Default(false)).
		Property("table_mode", moi.StringSchema().Description("Table output mode.").Enum("html", "image").Default("html")).
		Property("image_caption", moi.BooleanSchema().Description("Generate captions for image blocks. The Enhanced managed preset supplies its default; the generic schema must not materialize it into Standard.")).
		Property("caption_language", moi.StringSchema().Description("Output language for generated image captions.").Enum("zh", "en").Default("zh")).
		Property("image_ocr", moi.BooleanSchema().Description("Run OCR for image blocks. The Enhanced managed preset supplies its default; the generic schema must not materialize it into Standard.")).
		Property("image_fragment_merge", moi.BooleanSchema().Description("Merge page-level visual fragments into complete images with VLM confirmation. Requires image_caption=true or image_ocr=true so the replacement image receives content enrichment.").Default(false)).
		Property("title_enrichment", moi.StringSchema().Description("Title enrichment strategy. The Enhanced managed preset supplies its default; the generic schema must not materialize it into Standard.").Enum("off", "vlm")).
		Property("pptx_title_sidechannel", moi.BooleanSchema().Description("Recover explicit PPTX title placeholders from the original OOXML package. The Enhanced managed preset supplies its default; the generic schema must not materialize it into Standard.")).
		Property("formula", moi.BooleanSchema().Description("Repair or enrich formula blocks. The Enhanced managed preset supplies its default; the generic schema must not materialize it into Standard.")).
		Property("code_format_preserve", moi.BooleanSchema().Description("Repair redundant outer Markdown fences on recognized code blocks. Indentation and blank lines remain unchanged; final safe fencing is always provided by the writer. The Enhanced managed preset supplies its default; the generic schema must not materialize it into Standard.")).
		Property("header_footer_clean", moi.BooleanSchema().Description("Detect repeated page-edge text across pages in Enhanced mode. Matches are excluded from plain text and retained as DISCARDED blocks with audit metadata. Source Markdown is cleaned only when a trusted block-to-source mapping is available; otherwise it is preserved fail-open. Explicit false disables both detection stages.")).
		Property("chart_caption", moi.BooleanSchema().Description("Render a spreadsheet chart's OpenXML data series into searchable text. The Enhanced managed preset supplies its default; Native always forces it off, and the generic schema must not materialize it into Standard.")).
		Property("reading_order", moi.StringSchema().Description("Reading order strategy.").Enum("index", "xy_cut").Default("index")).
		Property("docx_route", moi.StringSchema().Description("DOCX conversion route.").Enum("pdf", "openxml").Default("pdf")).
		Property("docx_openxml_strict", moi.BooleanSchema().Description("When a configured aligner fails for one document: false continues without bounding boxes, true returns the geometry error. Missing alignment capability always fails.").Default(false)).
		Property("pptx_route", moi.StringSchema().Description("PPTX conversion route.").Enum("pdf", "openxml").Default("pdf")).
		AdditionalProperties(true)
}

func schemaDocumentVisualParseOptions() *moi.SchemaBuilder {
	return schemaDocumentParseOptions().
		Property("enable_engineering_page_plan", moi.BooleanSchema().
			Description("Enable industrial_drawing_v1 engineering page-level VLM planning. Must be true together with enable_engineering_region_extract; invalid for other profiles.")).
		Property("enable_engineering_region_extract", moi.BooleanSchema().
			Description("Enable industrial_drawing_v1 high-resolution region extraction. Must be true together with enable_engineering_page_plan; invalid for other profiles.")).
		Property("engineering_vlm_model", moi.StringSchema().
			Description("VLM model used only for engineering drawing page plan and region extraction. Required when engineering enhancement is enabled.")).
		Property("engineering_vlm_reasoning_effort", moi.StringSchema().
			Description("Optional reasoning.effort passed to the engineering VLM calls. Empty means do not send reasoning; unsupported models or providers should fail explicitly.")).
		Property("engineering_region_padding_ratio", moi.NumberSchema().Minimum(0).
			Description("Padding ratio applied around engineering region OCR crops. Default 0.16.")).
		Property("engineering_region_padding_px", moi.NumberSchema().Minimum(0).
			Description("Minimum pixel padding applied around engineering region OCR crops. Default 192."))
}

func schemaDocumentVisualParseInput() *moi.SchemaBuilder {
	return schemaDocumentVisualParseInputWithOptions(schemaDocumentVisualParseOptions(), false)
}

func schemaDocumentVisualParseCodexInput() *moi.SchemaBuilder {
	codexOptions := schemaDocumentVisualParseOptions().
		Property("model", moi.StringSchema().
			Description("Chat model selected from workspace LLM model resources for this Codex parse run.")).
		Property("backend_id", moi.IntegerSchema().
			Description("Backend ID returned with the selected workspace LLM model resource. Required together with model.")).
		Property("model_reasoning_effort", moi.StringSchema().
			Description("Optional Codex model_reasoning_effort override for this parse run.")).
		Required("model", "backend_id")
	return schemaDocumentVisualParseInputWithOptions(codexOptions, true)
}

func schemaDocumentVisualParseInputWithOptions(visualOptions *moi.SchemaBuilder, requireOptions bool) *moi.SchemaBuilder {
	schema := moi.NewSchema().
		Property("enabled", moi.BooleanSchema().
			Description("When false, the work item records status=disabled and skips visual manifest construction.")).
		Property("sources", moi.ArraySchema().Items(schemaSource())).
		Property("file_id", moi.StringSchema()).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema())).
		Property("documents", moi.ArraySchema().Items(schemaDocument())).
		Property("layout", schemaAnyObject()).
		Property("profile", moi.StringSchema().Default("industrial_drawing_v1")).
		Property("vlm_model", moi.StringSchema()).
		Property("options", visualOptions).
		Property("manifest_file_name", moi.StringSchema()).
		Property("require_page_images", moi.BooleanSchema().Default(true)).
		Property("require_object_images", moi.BooleanSchema().Default(true)).
		Property("require_visual_context", moi.BooleanSchema().Default(true)).
		Property("write_manifest_file", moi.BooleanSchema().Default(true))
	if requireOptions {
		schema.Required("options")
	}
	return schema.AdditionalProperties(true)
}

func schemaDocumentVisualParseOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("schema_version", moi.StringSchema().
			Description("Document visual manifest schema version emitted by the parser.")).
		Property("profile", moi.StringSchema().
			Description("Parser profile used for drawing visual extraction.")).
		Property("manifest", schemaDocumentVisualManifest()).
		Property("manifests", schemaDocumentVisualManifestArray()).
		Property("manifest_file_id", moi.StringSchema().
			Description("Catalog file ID of the primary visual manifest JSON artifact.")).
		Property("manifest_file_ids", moi.ArraySchema().Items(moi.StringSchema()).
			Description("Catalog file IDs of all visual manifest JSON artifacts produced by this run.")).
		Property("derived_file_ids_by_source", schemaAnyObject().
			Description("Maps each original Catalog file to the page and visual-object images extracted from it. Pass this mapping to data.lineage.register so every image is linked only to its original file.")).
		Property("documents", moi.ArraySchema().Items(schemaDocument()).
			Description("Document records projected from the visual manifest for downstream text processing.")).
		Property("validation", schemaAnyObject().
			Description("Validation details for the parsed visual manifest.")).
		Property("status", moi.StringSchema().Enum("ready", "disabled").
			Description("Visual parse status for downstream index nodes.")).
		Required("schema_version", "profile", "manifest", "validation").
		AdditionalProperties(true)
}

func schemaDocumentVisualIndexTextInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("enabled", moi.BooleanSchema().
			Description("When false, the work item records status=disabled and skips text embedding/index writes.")).
		Property("manifest", schemaAnyObject()).
		Property("manifest_file_id", moi.StringSchema()).
		Property("text_vector_table", moi.StringSchema().
			Description("MatrixOne vector table used to store text embeddings produced from the visual manifest.")).
		Property("table_name", moi.StringSchema()).
		Property("embedding_model", moi.StringSchema().
			Description("Embedding model used to convert visual text content into vectors before writing the text vector table.")).
		Property("embedding_dimension", moi.IntegerSchema().Minimum(1)).
		Property("enable_multilevel_index", moi.BooleanSchema()).
		Property("section_size", moi.IntegerSchema().Minimum(1)).
		Property("policy", moi.StringSchema().Enum("FAIL", "SKIP", "OVERWRITE")).
		Property("file_id", moi.StringSchema()).
		Property("volume_id", moi.IntegerSchema()).
		Property("dataset_meta_table", moi.StringSchema()).
		// Keep embedding_model/text_vector_table required for residual form planning.
		// enabled=false is handled at runtime before manifest loading.
		Required("embedding_model", "text_vector_table").
		AdditionalProperties(true)
}

func schemaDocumentVisualIndexTextOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("written", moi.IntegerSchema().
			Description("Number of text vector rows written for the visual manifest.")).
		Property("documents_count", moi.IntegerSchema().
			Description("Number of document records generated from visual manifest text.")).
		Property("text_vector_table", moi.StringSchema().
			Description("MatrixOne vector table that received text embedding rows.")).
		Property("embedding_model", moi.StringSchema().
			Description("Text embedding model used for the written vector rows.")).
		Property("index_version", moi.IntegerSchema().
			Description("Batch marker recorded on text vector rows so later image indexing and cleanup can target the same manifest write.")).
		Property("manifest_file_id", moi.StringSchema().
			Description("Catalog file ID of the visual manifest used for text indexing.")).
		Property("documents", moi.ArraySchema().Items(schemaAnyObject()).
			Description("Text document records written or prepared for the vector index.")).
		Property("status", moi.StringSchema().
			Description("Indexing status. Set to disabled when enabled=false so the work item skips text embedding/index writes.")).
		Required("written", "documents_count", "text_vector_table", "embedding_model").
		AdditionalProperties(true)
}

func schemaDocumentVisualIndexImageInput() *moi.SchemaBuilder {
	disabledBranch := moi.NewSchema().
		Property("enabled", moi.BooleanSchema().Enum(false)).
		Required("enabled").
		AdditionalProperties(true)
	activeBranch := moi.NewSchema().
		Property("enabled", moi.BooleanSchema()).
		Property("image_embedding_dimension", moi.IntegerSchema().Minimum(1)).
		Property("embedding_dimension", moi.IntegerSchema().Minimum(1)).
		Property("distance_metric", moi.StringSchema().Enum("cosine")).
		Property("embedding_source", moi.StringSchema().Enum("real")).
		Not(disabledBranch).
		AdditionalProperties(true)
	return moi.NewSchema().
		Property("enabled", moi.BooleanSchema().
			Description("When false, the work item records status=disabled and skips image embedding/index writes.")).
		Property("manifest", schemaDocumentVisualManifest()).
		Property("manifests", schemaDocumentVisualManifestArray()).
		Property("manifest_file_id", moi.StringSchema()).
		Property("manifest_file_ids", moi.ArraySchema().Items(moi.StringSchema())).
		Property("image_vector_table", moi.StringSchema().
			Description("Image vector table bound to the image embedding model, dimension, preprocess version and distance metric.")).
		Property("table_name", moi.StringSchema()).
		Property("image_embedding_model", moi.StringSchema().
			Description("Image embedding model, e.g. efficientnet-b3.")).
		Property("embedding_model", moi.StringSchema()).
		Property("image_embedding_backend_id", moi.StringSchema().
			Description("Image embedding backend identifier. Numeric values are passed as backend_id when routing the embedding request.")).
		Property("embedding_backend_id", moi.StringSchema()).
		Property("image_embedding_dimension", moi.IntegerSchema()).
		Property("embedding_dimension", moi.IntegerSchema()).
		Property("preprocess_version", moi.StringSchema()).
		Property("distance_metric", moi.StringSchema()).
		Property("embedding_source", moi.StringSchema()).
		Property("index_version", moi.IntegerSchema().
			Description("Version generated by the text index branch; image rows use the same version so KB chunk versions can govern text and image indexes together.")).
		Property("scopes", moi.ArraySchema().Items(moi.StringSchema().Enum("page", "visual_object"))).
		Property("allow_empty", moi.BooleanSchema().
			Description("When true, missing page/object images produce status=no_indexable_images instead of failing.")).
		Property("policy", moi.StringSchema().Enum("FAIL", "SKIP", "OVERWRITE")).
		Property("volume_id", moi.IntegerSchema()).
		AnyOf(disabledBranch, activeBranch).
		AdditionalProperties(true)
}

func schemaDocumentVisualIndexImageOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("written", moi.IntegerSchema().
			Description("Number of image vector rows written.")).
		Property("page_rows", moi.IntegerSchema().
			Description("Number of page-level image rows written.")).
		Property("visual_object_rows", moi.IntegerSchema().
			Description("Number of visual-object image rows written.")).
		Property("documents_count", moi.IntegerSchema().
			Description("Number of visual manifest documents considered for image indexing.")).
		Property("image_vector_table", moi.StringSchema().
			Description("MatrixOne vector table that received image embedding rows.")).
		Property("embedding_model", moi.StringSchema().
			Description("Image embedding model used for the written vector rows.")).
		Property("embedding_dimension", moi.IntegerSchema().
			Description("Embedding vector dimension used by the image index.")).
		Property("embedding_backend_id", moi.StringSchema().
			Description("Backend identifier associated with the image embedding model.")).
		Property("preprocess_version", moi.StringSchema().
			Description("Image preprocessing version used before embedding.")).
		Property("distance_metric", moi.StringSchema().
			Description("Vector distance metric used by the image index.")).
		Property("embedding_source", moi.StringSchema().
			Description("Embedding source category recorded with the image index rows.")).
		Property("index_version", moi.IntegerSchema().
			Description("Batch marker copied to image vector rows so downstream retrieval can match images with the corresponding text index batch.")).
		Property("manifest_file_id", moi.StringSchema().
			Description("Catalog file ID of the visual manifest used for image indexing.")).
		Property("source_file_id", moi.StringSchema().
			Description("Primary source file ID that produced image index rows.")).
		Property("all_source_file_ids", moi.ArraySchema().Items(moi.StringSchema()).
			Description("All source file IDs considered by this image indexing run.")).
		Property("indexed_source_file_ids", moi.ArraySchema().Items(moi.StringSchema()).
			Description("Source file IDs that actually wrote image index rows.")).
		Property("source_file_ids", moi.ArraySchema().Items(moi.StringSchema()).
			Description("Deprecated compatibility alias for indexed_source_file_ids.")).
		Property("file_statuses", moi.ArraySchema().Items(schemaAnyObject()).
			Description("Per source file image index status, including no_indexable_images for files without indexable images.")).
		Property("status", moi.StringSchema().Enum("ready", "no_indexable_images", "disabled").
			Description("Image index status for downstream workflow checks.")).
		Required("written", "image_vector_table", "embedding_model", "embedding_dimension", "preprocess_version", "distance_metric", "embedding_source").
		AdditionalProperties(true)
}

func schemaKnowledgeIndexBuildInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Build or refresh a searchable knowledge/vector index from parsed or split documents. Use this as a final sink when the user wants later retrieval or Q&A over documents.").
		Property("documents", moi.ArraySchema().Items(schemaDocument()).Description("Documents to index, usually parsed documents or chunks from a split node.")).
		Property("knowledge_base", moi.StringSchema().
			Description("Compatibility alias for table_name when table_name is omitted.")).
		Property("table_name", moi.StringSchema().
			Description("Vector index name. Use widget=vector_index_select in input_form so the user can select an existing index or create a new one. When the same DSL also has a search node, hardcode the same name in both.").
			Example("idx_documents")).
		Property("embedding_model", moi.StringSchema().
			Description("Embedding model name for vectorization, e.g. BAAI/bge-m3.")).
		Property("embedding_dimension", moi.IntegerSchema().Minimum(1).
			Description("Vector dimension. Usually inferred from embedding model output.")).
		Property("policy", moi.StringSchema().
			Description("Write policy: OVERWRITE (default), SKIP, or FAIL.")).
		Property("file_id", moi.StringSchema().Description("Optional source/parsed file ID associated with the indexed documents.")).
		Property("volume_id", moi.IntegerSchema().
			Description("Optional Catalog volume ID for vector write scope.").
			Example(123)).
		Property("enable_multilevel_index", moi.BooleanSchema().
			Description("When enabled, expand indexed content into document, section, and chunk levels before vector write so retrieval can use both precise chunks and broader context.").
			Default(true)).
		Property("section_size", moi.IntegerSchema().Minimum(1).
			Description("When multi-level indexing is enabled, group this many consecutive chunks into one section-level index entry.").
			Default(defaultSectionSize)).
		Required("documents").
		AdditionalProperties(true)
}

func schemaKnowledgeIndexBuildOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Knowledge index write result.").
		Property("written", moi.IntegerSchema().Description("Number of document/index rows written.")).
		Property("index_version", moi.IntegerSchema().Description("Batch marker stored in vector row metadata for later retrieval, replacement, or cleanup of this write.")).
		Required("written").
		AdditionalProperties(true)
}

func schemaDataTransformInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("target_type", moi.StringSchema()).
		Property("value", moi.AnySchema().Description("Generic upstream value to transform into the requested target_type.").Example(map[string]interface{}{"text": "hello"}).Range("Any valid workflow value: object, array, string, number, boolean, or null.")).
		Property("json", moi.AnySchema().Description("JSON payload to transform when target_type is json, rows, table, documents, or text.").Example(map[string]interface{}{"records": []interface{}{map[string]interface{}{"id": "row-1"}}}).Range("Any valid JSON value: object, array, string, number, boolean, or null.")).
		Property("documents", moi.ArraySchema().Items(schemaDocument())).
		Property("rows", moi.ArraySchema().Items(moi.ArraySchema().Items(moi.AnySchema()))).
		Property("columns", moi.ArraySchema().Items(moi.StringSchema())).
		Property("text", moi.StringSchema()).
		AdditionalProperties(true)
}

func schemaDataTransformOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("target_type", moi.StringSchema()).
		Property("text", moi.StringSchema()).
		Property("json", schemaAnyObject()).
		Property("documents", moi.ArraySchema().Items(schemaDocument())).
		Property("rows", moi.ArraySchema().Items(moi.ArraySchema().Items(moi.AnySchema()))).
		Property("columns", moi.ArraySchema().Items(moi.StringSchema())).
		AdditionalProperties(true)
}

func schemaRouterOutput() *moi.SchemaBuilder {
	arr := moi.ArraySchema().Items(schemaSource())
	return moi.NewSchema().
		Property("document", arr).
		Property("image", moi.ArraySchema().Items(schemaSource())).
		Property("audio", moi.ArraySchema().Items(schemaSource())).
		Property("video", moi.ArraySchema().Items(schemaSource())).
		Required("document", "image", "audio", "video").
		AdditionalProperties(true)
}

func schemaDocumentsInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Input contract for nodes that consume normalized documents.").
		Property("documents", moi.ArraySchema().Items(schemaDocument()).Description("Document array from parse, split, extraction, or another document-producing node.")).
		Required("documents").
		AdditionalProperties(true)
}

func schemaDocumentsOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Output contract for nodes that produce normalized documents.").
		Property("documents", moi.ArraySchema().Items(schemaDocument()).Description("Document array to feed into split, extraction, indexing, save, or custom document-processing nodes.")).
		Property("text", moi.StringSchema().Description("Plain text projection of the produced documents when available.")).
		Required("documents").
		AdditionalProperties(true)
}

func schemaParseStageRuntimeOutput() *moi.SchemaBuilder {
	return schemaDocumentsOutput().
		Property("metadata", schemaAnyObject().Description("Optional single-source public parser metadata. Emitted for the fixed Standard PDF/Word/PowerPoint policy or actionable V3 option warnings; responses without either preserve the existing shape."))
}

func schemaMultiLevelIndexInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Expand chunk-level documents into doc/section/chunk three-level index entries. "+
			"Each chunk gets level=chunk; groups of chunks become level=section; the full file becomes level=doc. "+
			"Downstream embedding and vector write will index all three levels, enabling multi-granularity retrieval.").
		Property("documents", moi.ArraySchema().Items(schemaDocument()).
			Description("Chunk documents from a prior split step. Each must have metadata.file_id and metadata.file_name.")).
		Property("enable", moi.BooleanSchema().
			Description("Set to false to skip multi-level expansion and pass documents through unchanged.").
			Default(true)).
		Property("index_version", moi.IntegerSchema().
			Description("Version tag written into metadata.index_version on every entry. Defaults to current timestamp (ms). "+
				"Use a fixed value when re-indexing to allow old entries to be identified and cleaned up.").
			Default(0)).
		Property("section_size", moi.IntegerSchema().Minimum(1).
			Description("Maximum number of consecutive chunks grouped into one section entry.").
			Default(defaultSectionSize)).
		Property("max_doc_summary_chars", moi.IntegerSchema().Minimum(1).
			Description("Maximum characters for the doc-level summary entry (concatenated from chunk content).").
			Default(2000)).
		Required("documents").
		AdditionalProperties(true)
}

func schemaWebCrawlInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("urls", moi.ArraySchema().Items(moi.StringSchema())).
		Property("sources", moi.ArraySchema().Items(schemaSource())).
		Property("timeout_seconds", moi.IntegerSchema().Minimum(1)).
		Property("max_content_bytes", moi.IntegerSchema().Minimum(1)).
		Property("user_agent", moi.StringSchema()).
		Property("include_assets", moi.BooleanSchema()).
		Property("asset_types", moi.ArraySchema().Items(moi.StringSchema().Enum("pdf", "image"))).
		Property("max_asset_count", moi.IntegerSchema().Minimum(1)).
		Property("max_asset_bytes", moi.IntegerSchema().Minimum(1)).
		Property("pdf_service_url", moi.StringSchema()).
		Property("render_js", moi.BooleanSchema()).
		Property("render_wait_ms", moi.IntegerSchema().Minimum(200)).
		Property("ocr_service_url", moi.StringSchema().
			Description("OCR service endpoint URL for image-based text extraction from crawled pages.")).
		Property("follow_html_links", moi.BooleanSchema().
			Description("Follow links found in HTML content. Defaults to true for HTML inputs.")).
		Property("max_linked_pages", moi.IntegerSchema().Minimum(1).
			Description("Maximum number of linked pages to crawl. Default 8.").
			Default(8)).
		AdditionalProperties(true)
}

func schemaWebCrawlOutput() *moi.SchemaBuilder {
	failure := moi.NewSchema().
		Property("url", moi.StringSchema()).
		Property("reason", moi.StringSchema()).
		Required("url", "reason").
		AdditionalProperties(true)
	return moi.NewSchema().
		Property("documents", moi.ArraySchema().Items(schemaDocument())).
		Property("failed_urls", moi.ArraySchema().Items(failure)).
		Property("pdf_links", moi.ArraySchema().Items(moi.StringSchema())).
		Required("documents").
		AdditionalProperties(true)
}

func schemaCleanTextInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Clean plain text. Prefer upstream text; when the previous node only emits documents (for example moi:document.parse), pass documents and the worker projects content into text.").
		Property("text", moi.StringSchema().Description("Plain text to clean. Usually bound from upstream .text or projected from documents.")).
		Property("documents", moi.ArraySchema().Items(schemaDocument()).Description("Optional upstream documents used when text is empty. Contents are joined into text before cleaning.")).
		Property("mask_sensitive_info", moi.BooleanSchema().
			Description("Mask structured sensitive data patterns such as email addresses and phone-like numbers.")).
		Property("unicode_normalization", moi.BooleanSchema().
			Description("Normalize Unicode text to NFC before other cleaning steps.")).
		Property("traditional_chinese_to_simple", moi.BooleanSchema().
			Description("Convert Traditional Chinese text to Simplified Chinese before other cleaning filters.")).
		Property("remove_url", moi.BooleanSchema().
			Description("Remove http(s) and www URL text from the input.")).
		Property("remove_invisible_char", moi.BooleanSchema().
			Description("Remove control and format characters while preserving tabs and line breaks for whitespace normalization.")).
		Property("remove_html_labels", moi.BooleanSchema().
			Description("Remove HTML tag labels from the input text.")).
		Property("filter_special_char_ratio", moi.NumberSchema().Minimum(0).Maximum(1).
			Description("Drop the text when the ratio of non-letter/non-number characters exceeds this threshold. 0 disables the filter.")).
		Property("deduplication_ngram_ratio", moi.NumberSchema().Minimum(0).Maximum(1).
			Description("Drop the text when the repeated 3-gram ratio exceeds this threshold. 0 disables the filter.")).
		AnyOf(
			moi.NewSchema().Required("text"),
			moi.NewSchema().Required("documents"),
		).
		AdditionalProperties(true)
}

func schemaJSONRepairInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("text", moi.StringSchema()).
		Required("text").
		AdditionalProperties(true)
}

func schemaTextOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("text", moi.StringSchema().
			Description("Cleaned text after whitespace and newline normalization.")).
		Required("text").
		AdditionalProperties(true)
}

func schemaSplitLengthInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("text", moi.StringSchema()).
		Property("chunk_size", moi.IntegerSchema().Minimum(1).Default(defaultSplitChunkSize).Example(defaultSplitChunkSize)).
		Property("overlap", moi.IntegerSchema().Minimum(0).Default(defaultSplitOverlap).Example(defaultSplitOverlap)).
		Required("text").
		AdditionalProperties(true)
}

func schemaSplitLengthOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("chunks", moi.ArraySchema().Items(moi.StringSchema())).
		Required("chunks").
		AdditionalProperties(true)
}

func schemaSplitDocumentsLengthInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Split each input document by length while preserving metadata lineage and semantic atomicity. Tables, typed code, and Markdown containers with fenced code remain whole and may exceed chunk_size.").
		Property("documents", moi.ArraySchema().Items(schemaDocument()).Description("Documents to split, usually from moi:document.parse output .documents.")).
		Property("chunk_size", moi.IntegerSchema().Minimum(1).Default(defaultSplitChunkSize).Example(defaultSplitChunkSize).Description("Target maximum chunk length for splittable text. Atomic tables, typed code, and fenced-code Markdown containers may exceed it to preserve structure.")).
		Property("overlap", moi.IntegerSchema().Minimum(0).Default(defaultSplitOverlap).Example(defaultSplitOverlap).Description("Overlap length between adjacent chunks. Runtime form can expose this when needed; otherwise use default.")).
		Required("documents").
		AdditionalProperties(true)
}

func schemaSplitDocumentsOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Split result. The output is still a documents array, now chunked and ready for extraction, indexing, saving, or custom document processors.").
		Property("documents", moi.ArraySchema().Items(schemaDocument()).Description("Chunk documents preserving source metadata and chunk positions.")).
		Required("documents").
		AdditionalProperties(true)
}

func schemaJSONRepairOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("json", moi.StringSchema()).
		Property("valid", moi.BooleanSchema()).
		Required("json", "valid").
		AdditionalProperties(true)
}

func schemaDataLineageRegisterInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Register data assets and create derivation links in one step. "+
			"Covers the complete retrieval lineage: source file → parsed docset → vector index. "+
			"Requires parsed_file_id and one source field: source_file_ids, source_file_id, or legacy raw_file_id. vector_table is optional.").
		Property("source_file_id", moi.StringSchema().
			Description("Source file ID for single-root lineage.")).
		Property("source_file_ids", moi.ArraySchema().Items(moi.StringSchema()).
			Description("Source file IDs for multi-root lineage fan-out. The worker calls lineage registration once per source file.")).
		Property("allow_empty_source_file_ids", moi.BooleanSchema().
			Description("When true, empty source_file_ids returns an empty lineage result instead of failing.")).
		Property("source_file_name", moi.StringSchema().
			Description("Source file name for display in lineage UI.")).
		Property("raw_file_id", moi.StringSchema().
			Description("Legacy alias for source_file_id. Kept for already-saved workflows.")).
		Property("raw_file_name", moi.StringSchema().
			Description("Legacy alias for source_file_name. Kept for already-saved workflows.")).
		Property("parsed_file_id", moi.StringSchema().
			Description("Parsed docset file ID (from a prior moi:files.write_documents step).")).
		Property("derived_file_ids", moi.ArraySchema().Items(moi.StringSchema()).
			Description("Catalog file IDs for derived artifacts owned by this lineage, such as visual manifest, page images, object images, or extracted markdown.")).
		Property("derived_file_ids_by_source", schemaNullableObject().
			Description("Maps each original Catalog file to the artifacts produced from it. For a multi-source run, this prevents artifacts from one source being attached to another and takes precedence over derived_file_ids. Null means no source-specific derived artifacts were supplied.")).
		Property("vector_table", moi.StringSchema().
			Description("Vector index table name. If provided, registers a vector asset and links it to the parsed asset.")).
		Property("embedding_model", moi.StringSchema().
			Description("Embedding model used to create the vector index. Required when vector_table is provided.")).
		Property("semantic_model_ref_vector_table", moi.StringSchema().
			Description("Vector table reference used to find the semantic model to bind: text lineage appends the source file into the model's files.file_ids (defaults to vector_table when omitted), and the image branch updates image index metadata. Does not register a text vector asset.")).
		Property("image_vector_table", moi.StringSchema().
			Description("Image vector index table name. If provided, registers an image vector asset and links it to the parsed asset.")).
		Property("image_embedding_model", moi.StringSchema().
			Description("Image embedding model used by the image vector index. Required when image_vector_table is provided.")).
		Property("image_embedding_backend_id", moi.StringSchema().
			Description("Image embedding backend identifier recorded in image vector asset metadata.")).
		Property("image_embedding_dimension", moi.IntegerSchema().
			Description("Image embedding dimension recorded in image vector asset metadata. Required when image_vector_table is provided.")).
		Property("image_preprocess_version", moi.StringSchema().
			Description("Image preprocessing version recorded in image vector asset metadata. Required when image_vector_table is provided.")).
		Property("image_distance_metric", moi.StringSchema().
			Description("Image vector distance metric recorded in image vector asset metadata. Required when image_vector_table is provided.")).
		Property("image_index_file_statuses", moi.ArraySchema().Items(schemaAnyObject()).
			Description("Per-source image indexing results. When provided, every source file must have exactly one ready or no_indexable_images status.")).
		Property("output_file_id", moi.StringSchema().
			Description("Output file ID (e.g. extract result). If provided, registers an output asset and links it to the raw asset with transformed_from.")).
		Property("volume_id", moi.IntegerSchema().
			Description("Catalog volume ID. Pass 0 or omit for volume-less assets.").
			Default(0)).
		Property("source", moi.StringSchema().
			Description("Pipeline name recorded on each asset.").
			Default("retrieval-index-pipeline")).
		Required("parsed_file_id").
		AdditionalProperties(true)
}

func schemaDataLineageRegisterOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Registered asset IDs for the complete lineage chain.").
		Property("raw_asset_id", moi.StringSchema().Description("Registered first source file asset ID. Deprecated alias for source_asset_id.")).
		Property("source_asset_id", moi.StringSchema().Description("Registered first source file asset ID.")).
		Property("source_asset_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Registered source file asset IDs in input order.")).
		Property("parsed_asset_id", moi.StringSchema().Description("Registered parsed docset asset ID.")).
		Property("vector_asset_id", moi.StringSchema().Description("Registered vector index asset ID (empty if vector_table not provided).")).
		Property("image_vector_asset_id", moi.StringSchema().Description("Registered image vector index asset ID (empty if image_vector_table not provided).")).
		Property("output_asset_id", moi.StringSchema().Description("Registered output asset ID (empty if output_file_id not provided).")).
		AdditionalProperties(true)
}

func schemaDataAssetRegisterInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Register a data asset that tracks a raw file or derived artifact for lineage. "+
			"Typically used after producing a new file (e.g. parsed JSONL, vector index) to record it as a named asset. "+
			"The returned asset_id is used by moi:data.asset.link to build derivation chains.").
		Property("asset_type", moi.StringSchema().
			Description("Typed asset namespace. Use file for catalog file IDs and vector_index for vector tables.").
			Enum("file", "vector_index", "table")).
		Property("asset_ref", moi.StringSchema().
			Description("Type-local asset reference, such as a catalog file_id or tenant vector table name.")).
		Property("raw_file_id", moi.StringSchema().
			Description("Deprecated compatibility input. When asset_type/asset_ref are omitted, existing catalog files are treated as file assets and missing file ids are treated as vector_index refs.")).
		Property("name", moi.StringSchema().
			Description("Human-readable asset name shown in lineage UI. "+
				"Typically the original file name or '<file_name>.parsed.jsonl' for parsed assets.")).
		Property("volume_id", moi.IntegerSchema().
			Description("Catalog volume ID that owns this asset. Pass 0 or omit for volume-less assets like vector tables.").
			Default(0)).
		Property("source", moi.StringSchema().
			Description("Identifier of the pipeline or workflow that created this asset, e.g. 'retrieval-index-pipeline', 'cv-ingest'.").
			Example("retrieval-index-pipeline")).
		Property("asset_id", moi.StringSchema().
			Description("Optional: supply a pre-generated asset ID. If omitted the server assigns one automatically.")).
		Property("meta", schemaAnyObject().
			Description("Arbitrary display metadata attached to the asset. Do not put run/case/workitem provenance here.")).
		AdditionalProperties(true)
}

func schemaDataAssetLinkInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Create a derivation link between two data assets, recording how one artifact was produced from another. "+
			"Call this after registering both the source and derived assets via moi:data.asset.register. "+
			"Example chain: raw file → (parsed_from) → parsed JSONL → (indexed_from) → vector index.").
		Property("root_asset_id", moi.StringSchema().
			Description("The root/source file family asset ID.")).
		Property("source_asset_id", moi.StringSchema().
			Description("The asset_id of the SOURCE asset (upstream).")).
		Property("target_asset_id", moi.StringSchema().
			Description("The asset_id of the TARGET asset (downstream).")).
		Property("asset_id", moi.StringSchema().
			Description("Deprecated compatibility input. For parsed_from this is the source asset id; for indexed_from this is the target vector asset id.")).
		Property("file_id", moi.StringSchema().
			Description("Deprecated compatibility input. For parsed_from this is the parsed file id; for indexed_from this is the parsed file id used to find the root bridge.")).
		Property("kind", moi.StringSchema().
			Description("Derivation relationship type describing how the target was produced from the source.").
			Enum("parsed_from", "indexed_from", "extracted_from", "transformed_from")).
		Property("producer_workitem_id", moi.StringSchema().
			Description("The workitem that produced the target asset.")).
		Property("logical_slot", moi.StringSchema().
			Description("Stable logical output slot, such as parsed_docset or vector_index.")).
		Property("meta", schemaAnyObject().
			Description("Optional metadata for this derivation link (e.g. pipeline version, parameters used).")).
		Required("kind").
		AdditionalProperties(true)
}

func schemaDataDocMapInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("asset_id", moi.StringSchema()).
		Property("raw_file_id", moi.StringSchema()).
		Property("parsed_file_id", moi.StringSchema()).
		Property("manifest", schemaAnyObject()).
		Required("asset_id", "raw_file_id", "parsed_file_id").
		AdditionalProperties(true)
}

func schemaDataTableUpsertJSONInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Write one structured JSON/object row into a workspace table. Use this after LLM extraction or a custom operator that returns a JSON object or values map. For tabular rows arrays, prefer catalog.sink.write or SQL/import nodes.").
		Property("table_name", moi.StringSchema().MinLength(1).Pattern(sqlIdentifierPattern).Description("Destination table name in the workspace database. Required. If the table exists it is reused; if it does not exist it can be auto-created from json_schema or raw data.")).
		Property("database", moi.StringSchema().Description("Optional database name. Omit to use the workspace default database.")).
		Property("database_id", moi.IntegerSchema().Minimum(1).Description("Optional database numeric ID alternative.")).
		Property("raw_json", moi.AnySchema().
			Description("Raw JSON/object payload source for column extraction and merge. Bind this to an upstream extract/custom result when it returns a JSON object.").
			Example(map[string]interface{}{"order_id": "A001", "amount": 120.5}).
			Range("any valid JSON value (object/array/scalar)")).
		Property("json_schema", schemaAnyObject().
			Description("JSON Schema describing the expected row structure. When the table does not exist, this schema drives auto-creation: each property becomes a column with type inferred from the schema type (string→TEXT, integer→BIGINT, number→DOUBLE, boolean→BOOLEAN, array/object→JSON). Property names must be valid MatrixOne column names; invalid names fail explicitly instead of being skipped. Typically the same schema used by the upstream extract node.")).
		Property("values", schemaAnyObject().Description("Explicit column values map. These values are merged with raw_json and take precedence on key conflict.")).
		Property("aliases", schemaAnyObject().Description("Optional mapping from input JSON keys to destination column names.")).
		Property("column_whitelist", moi.ArraySchema().Items(moi.StringSchema()).Description("Optional list of columns allowed to be written; other keys are ignored.")).
		Property("update_exclude_columns", moi.ArraySchema().Items(moi.StringSchema()).Description("Columns excluded from update when an existing row is overwritten.")).
		Property("policy", moi.StringSchema().Enum("FAIL", "SKIP", "OVERWRITE").Description("Conflict policy when the target row/table exists.")).
		Required("table_name").
		AdditionalProperties(true)
}

func schemaDataTableUpsertJSONOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Structured table write result.").
		Property("database", moi.StringSchema().Description("Database that received the write.")).
		Property("table_name", moi.StringSchema().Description("Table that received the write.")).
		Property("affected_rows", moi.IntegerSchema().Description("Number of rows inserted or updated.")).
		Property("written_columns", moi.ArraySchema().Items(moi.StringSchema()).Description("Column names written from raw_json/values.")).
		Required("database", "table_name", "affected_rows").
		AdditionalProperties(true)
}

func schemaEmbeddingGenerateInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("documents", moi.ArraySchema().Items(schemaDocument()).
			Description("Document array whose content will be embedded; usually bound from parse, split, or extraction output.")).
		Property("model", moi.StringSchema().
			Description("Embedding model used to convert each document content field into a vector.")).
		Property("encoding_format", moi.StringSchema().
			Description("Embedding vector encoding format requested from the embedding backend.")).
		Required("documents").
		AdditionalProperties(true)
}

func schemaVectorWriteInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("documents", moi.ArraySchema().Items(schemaAnyObject())).
		Property("file_id", moi.AnySchema().
			Description("Optional file ID (string) associated with current documents.").
			Example("file_01JABCDXYZ").
			Range("string file ID or null")).
		Property("volume_id", moi.IntegerSchema().
			Description("Optional Catalog volume ID for vector write scope.").
			Example(123).
			Range("integer volume ID or null")).
		Property("embedding_model", moi.StringSchema().
			Description("Embedding model name for vectorization, e.g. BAAI/bge-m3.")).
		Property("embedding_dimension", moi.IntegerSchema().Minimum(1).
			Description("Vector dimension. Usually inferred from the embedding model output; only set when you need to override.")).
		Property("table_name", moi.StringSchema().
			Description("Vector index name. Use widget=vector_index_select in input_form so the user can select an existing index or create a new one.").
			Example("idx_documents")).
		Property("policy", moi.StringSchema().Enum("FAIL", "SKIP", "OVERWRITE").
			Description("Write policy when the table already exists: OVERWRITE (default), SKIP, or FAIL."))
}

func schemaVectorWriteOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("written", moi.IntegerSchema()).
		AdditionalProperties(true)
}

func schemaRunSQLInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Execute SQL against the workspace MatrixOne database. Use this for SQL queries, DDL, DML, aggregation, or table preparation. SELECT returns rows; write statements return affected_rows.").
		Property("sql", moi.StringSchema().MinLength(1).Description("SQL statement to execute. It can be a SELECT query or a mutating statement. Keep user-provided business SQL explicit in DSL.")).
		Property("database", moi.StringSchema().Description("Optional database name. Omit to use the workspace default database.")).
		Property("connection", schemaAnyObject().Description("Optional connection override. Normally omitted because the worker uses the workspace database connection.")).
		Property("params", schemaAnyObject().Description("Optional SQL bind parameters when the handler supports parameterized execution.")).
		Property("max_rows", moi.IntegerSchema().Minimum(0).Description("Maximum rows returned for SELECT queries. Omit or set 0 for no worker-side row limit.")).
		Property("timeout_seconds", moi.IntegerSchema().Minimum(0).Description("Optional execution timeout in seconds. Omit or set 0 to use the parent context without adding a timeout.")).
		Property("output_table", moi.StringSchema().Description("Optional output table name for materializing SELECT results. Use table or database.table.")).
		Property("write_mode", moi.StringSchema().Enum("create", "replace", "append").Description("Required when output_table is set. Supported values: create, replace, append.")).
		Property("materialization_strategy", moi.StringSchema().Enum("table").Description("Required when output_table is set. Only table is currently supported.")).
		Required("sql").
		AdditionalProperties(true)
}

func schemaDashboardRefreshInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Refresh a persisted Data Dashboard chart after reauthorizing its dashboard and current Effective Role.").
		Property("dashboard_id", moi.StringSchema().MinLength(1).Description("Persisted Data Dashboard ID supplied by the managed scheduler.")).
		Property("chart_id", moi.StringSchema().MinLength(1).Description("Persisted chart ID supplied by the managed scheduler.")).
		Required("dashboard_id", "chart_id").
		AdditionalProperties(false)
}

func schemaSQLProcessInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Execute one SQL statement against the workspace MatrixOne database. Required input is sql. Optional table_ref accepts a Catalog table reference (typically bound from moi:data.table.read) so the SQL node has a traceable Catalog-to-SQL data edge. When table_ref is bound, SQL may use the ${source_table} placeholder which is replaced with the quoted database.table FQN at runtime. SELECT/SHOW/WITH queries return rows; DML/DDL statements return affected_rows and execution status fields.").
		Property("sql", moi.StringSchema().MinLength(1).Example("SELECT now()").Description("SQL statement to execute. Put the complete business SQL here. When table_ref is bound, use ${source_table} as the Catalog-bound table reference instead of hard-coding physical names. Do not add materialization parameters; compose another SQL statement when a table write is needed.")).
		Property("table_ref", schemaNullableCatalogResourceRef().Description("Optional Catalog table reference consumed by this SQL node. Bind from an upstream moi:data.table.read table_ref output to establish a traceable Catalog-to-SQL data edge. Runtime resolves physical names from table_id. Unbound optional inputs are materialized as null by the workflow runtime.")).
		Required("sql").
		AdditionalProperties(false)
}

func schemaAPIRequestInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Send one HTTP request to an external API, webhook, or HTTP endpoint. Use this as the user-facing API node instead of catalog:http.request.").
		Property("method", moi.StringSchema().Description("HTTP method. Must be one of GET, POST, PUT, PATCH, DELETE, HEAD.").Enum("GET", "POST", "PUT", "PATCH", "DELETE", "HEAD")).
		Property("url", moi.StringSchema().MinLength(1).Description("Absolute HTTP or HTTPS request URL.")).
		Property("headers", schemaNullableStringMap().Description("Optional request headers object. Missing or null means no extra request headers. Every value must be a string, for example {\"Authorization\":\"Bearer token\"}.").
			Example(map[string]interface{}{"Content-Type": "application/json"}).
			Range("Object whose keys are HTTP header names and values are strings, or null.")).
		Property("body", schemaNullableAny().Description("Optional JSON request body. Missing or null means no request body. Objects and arrays are sent as JSON text.").
			Example(map[string]interface{}{"name": "moi"}).
			Range("Any valid JSON value. Omit or set null for methods that do not send a body.")).
		Property("timeout_seconds", schemaNullableInteger().Minimum(0).Maximum(300).Description("Optional request timeout in seconds. Missing, null, or 0 uses the node default timeout.").
			Example(30).
			Range("Integer from 0 to 300 seconds, or null. Missing, null, or 0 uses the node default timeout.")).
		Required("method", "url").
		AdditionalProperties(false)
}

func schemaProductSourceInput() *moi.SchemaBuilder {
	return moi.ObjectSchema().
		Description("Internal product-source payload assembled by Catalog. Resolves the image-baked MOI source tree without GitHub access.").
		Required("request_id", "workspace_id", "request").
		AdditionalProperties(false).
		Property("request_id", moi.StringSchema().MinLength(1)).
		Property("workspace_id", moi.StringSchema().MinLength(1)).
		Property("request", moi.ObjectSchema().
			Required("operation").
			AdditionalProperties(false).
			Property("operation", moi.StringSchema().Enum("ensure_product_source", "get_product_source").Description("Resolve the mounted product source and return a Codex workspace_ref.")).
			Property("expected_version", moi.StringSchema().Description("Optional exact product version that MANIFEST.version must match. Defaults to the running go-worker build version when that version is known.")))
}

func schemaGitHubRepoWorkspaceInput() *moi.SchemaBuilder {
	return moi.ObjectSchema().
		Description("Internal GitHub repository workspace payload assembled by Catalog. Ensures one shared clone per repository under the worker workspace root.").
		Property("request_id", moi.StringSchema().Description("Tool call / request id.")).
		Property("workspace_id", moi.StringSchema().Description("Workspace id.")).
		Property("credential", moi.ObjectSchema().
			Property("access_token", moi.StringSchema().Description("GitHub token injected by Catalog.")).
			Required("access_token"),
		).
		Property("request", moi.ObjectSchema().
			Property("operation", moi.StringSchema().Enum("ensure_repository", "create_worktree", "prepare_review_worktree", "remove_worktree").Description("Repository workspace operation. prepare_review_worktree creates a detached PR review worktree for the Catalog-resolved base/head snapshot.")).
			Property("owner", moi.StringSchema().Description("Repository owner.")).
			Property("repo", moi.StringSchema().Description("Repository name.")).
			Property("host", moi.StringSchema().Description("GitHub host. Defaults to github.com.")).
			Property("branch", moi.StringSchema().Description("Branch for ensure checkout or create_worktree.")).
			Property("base_ref", moi.StringSchema().Description("Base ref for create_worktree when creating a new branch. Defaults to HEAD.")).
			Property("worktree_name", moi.StringSchema().Description("Stable worktree name. Defaults from branch.")).
			Property("workspace_ref", moi.StringSchema().Description("Existing worktree workspace_ref for remove_worktree.")).
			Property("create_branch", moi.BooleanSchema().Description("When true (default), create_worktree creates/resets branch with -B.")).
			Property("force_remove", moi.BooleanSchema().Description("When true, remove_worktree uses git worktree remove --force.")).
			Property("number", moi.IntegerSchema().Minimum(1).Description("Pull request number for prepare_review_worktree.")).
			Property("base_sha", moi.StringSchema().Description("Exact base commit SHA resolved and injected by Catalog for prepare_review_worktree.")).
			Property("head_sha", moi.StringSchema().Description("Exact head commit SHA resolved and injected by Catalog for prepare_review_worktree.")).
			Required("owner", "repo"),
		).
		Required("workspace_id", "credential", "request")
}

func schemaGitHubRepoWriteInput() *moi.SchemaBuilder {
	return moi.ObjectSchema().
		Description("Internal GitHub channel write payload assembled by Catalog. Runtime callers must not pass credentials directly.").
		Property("request_id", moi.StringSchema().Description("Tool call / request id.")).
		Property("workspace_id", moi.StringSchema().Description("Workspace id.")).
		Property("credential", moi.ObjectSchema().
			Property("access_token", moi.StringSchema().Description("GitHub token injected by Catalog.")).
			Required("access_token"),
		).
		Property("request", moi.NewSchema().
			Description("GitHub write operation and its operation-specific fields. The registered worker validates the operation and fields.").
			Property("operation", moi.StringSchema()).
			Property("owner", moi.StringSchema()).
			Property("repo", moi.StringSchema()).
			Property("host", moi.StringSchema().Description("GitHub host bound by Catalog. Defaults to github.com.")).
			Property("title", moi.StringSchema()).
			Property("body", moi.StringSchema()).
			Property("assignees", moi.ArraySchema().Items(moi.StringSchema())).
			Property("labels", moi.ArraySchema().Items(moi.StringSchema())).
			Property("milestone", moi.IntegerSchema().Minimum(1)).
			Property("head", moi.StringSchema()).
			Property("base", moi.StringSchema()).
			Property("expected_base_sha", moi.StringSchema().Pattern("^[0-9a-fA-F]{40}$").Description("Optional server-owned PR base SHA that review and merge operations must still match immediately before mutation.")).
			Property("draft", moi.BooleanSchema()).
			Property("maintainer_can_modify", moi.BooleanSchema()).
			Required("operation").
			AdditionalProperties(true),
		).
		Required("workspace_id", "credential", "request")
}

func schemaGitHubRepoReadInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Internal GitHub channel read payload assembled by Catalog. Runtime callers must not pass credentials directly.").
		Property("request_id", moi.StringSchema()).
		Property("workspace_id", moi.StringSchema().MinLength(1)).
		Property("credential", moi.NewSchema().
			Property("access_token", moi.StringSchema().MinLength(1)).
			Required("access_token").
			AdditionalProperties(false)).
		Property("request", moi.NewSchema().
			Description("GitHub read operation and its operation-specific fields. list_stale_project_issues, list_stale_project_pull_requests, and list_untriaged_project_issues own internal GitHub Project pagination and stop as soon as limit matching items are found. list_untriaged_project_issues returns open Issues older than stale_for_hours whose existing Type, Priority, or Status Project field is empty or which has no assignee; when fewer than limit matches exist, it scans the complete Project so later items cannot starve. Use stale_for_hours and limit, not client pagination. Issue lists support since or updated_since only with state=all and sort=updated; Pull request lists have no since filter. The worker rejects issue pagination beyond page 3; do not use page scans.").
			Property("operation", moi.StringSchema().Description("GitHub read operation. list_stale_project_issues, list_stale_project_pull_requests, and list_untriaged_project_issues require stale_for_hours and limit and own all Project pagination. list_untriaged_project_issues returns at most 10 incomplete Issue triage candidates and scans until that limit is reached or the Project ends. Issue lists support since or updated_since only with state=all and sort=updated; Pull request lists have no since filter. Workflow-run created filters support recent/latest time windows.")).
			Property("owner", moi.StringSchema()).
			Property("repo", moi.StringSchema()).
			Property("host", moi.StringSchema().Description("GitHub host bound by Catalog. Defaults to github.com.")).
			Property("number", moi.IntegerSchema().Minimum(1)).
			Property("milestone", moi.StringSchema()).
			Property("assignee", moi.StringSchema()).
			Property("creator", moi.StringSchema()).
			Property("mentioned", moi.StringSchema()).
			Property("labels", moi.StringSchema()).
			Property("sort", moi.StringSchema().Description("Use sort=updated with state=all when filtering Issues by since or updated_since. Pull request lists have no since filter.")).
			Property("direction", moi.StringSchema()).
			Property("head", moi.StringSchema()).
			Property("base", moi.StringSchema()).
			Property("actor", moi.StringSchema()).
			Property("created", moi.StringSchema().Description("workflow-run created date filter. Use recent/latest time-window values.")).
			Property("head_sha", moi.StringSchema()).
			Property("exclude_pull_requests", moi.BooleanSchema()).
			Property("since", moi.StringSchema().Description("Issue update timestamp filter; only valid with state=all and sort=updated. Pull request lists have no since filter.")).
			Property("updated_since", moi.StringSchema().Description("Alias for since when listing Issues; only valid with state=all and sort=updated.")).
			Property("per_page", moi.IntegerSchema().Minimum(1).Maximum(100)).
			Property("page", moi.IntegerSchema().Minimum(1).Description("Worker rejects issue pagination beyond page 3; do not use page scans.")).
			Property("owner_type", moi.StringSchema().Enum("organization", "user")).
			Property("project_id", moi.StringSchema().Description("GitHub ProjectV2 node ID.")).
			Property("project_number", moi.IntegerSchema().Minimum(1)).
			Property("cursor", moi.StringSchema()).
			Property("stale_for_hours", moi.IntegerSchema().Minimum(1).Maximum(8760).Description("Required by list_stale_project_issues, list_stale_project_pull_requests, or list_untriaged_project_issues. The matching item's updated_at must be older than this many hours.")).
			Property("limit", moi.IntegerSchema().Minimum(1).Maximum(10).Description("Required by bounded Project list operations. Maximum matching items returned before the worker stops its internal Project traversal.")).
			Required("operation").
			AdditionalProperties(true)).
		Required("workspace_id", "credential", "request").
		AdditionalProperties(false)
}

func schemaCodexRunInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Internal Codex channel payload assembled by Catalog. Runtime callers must not pass credentials directly.").
		Property("request_id", moi.StringSchema()).
		Property("workspace_id", moi.StringSchema().MinLength(1)).
		Property("credential", moi.NewSchema().
			Property("api_key", moi.StringSchema().MinLength(1)).
			Property("base_url", moi.StringSchema().MinLength(1)).
			Property("model", moi.StringSchema().MinLength(1)).
			Property("reasoning_effort", moi.StringSchema().Enum("minimal", "low", "medium", "high", "xhigh", "max", "ultra")).
			Required("api_key", "base_url", "model", "reasoning_effort").
			AdditionalProperties(false)).
		Property("request", moi.NewSchema().
			Property("operation", moi.StringSchema().Enum("connection_test", "run")).
			Property("task", moi.StringSchema().MinLength(1).Description("Single Codex task.")).
			Property("parallel_tasks", moi.ArraySchema().Items(moi.NewSchema().
				Property("id", moi.StringSchema().MinLength(1).MaxLength(128)).
				Property("task", moi.StringSchema().MinLength(1).MaxLength(65536)).
				Property("output_schema", schemaAnyObject()).
				Required("id", "task", "output_schema").
				AdditionalProperties(false)).MinItems(1).MaxItems(8).Description("Independent read-only Codex tasks started concurrently without synthesis.")).
			Property("workspace_ref", moi.StringSchema().MinLength(1)).
			Property("image_file_ids", moi.ArraySchema().Items(moi.StringSchema().MinLength(1)).MinItems(1).MaxItems(8).UniqueItems(true).Description("Workspace-scoped image file IDs materialized by a trusted tool. run requires workspace_ref, image_file_ids, or both.")).
			Property("output_schema", schemaAnyObject().Description("Optional JSON schema for run results.")).
			Property("review_target", moi.NewSchema().
				Description("Server-owned pull request snapshot bound to the prepared review worktree.").
				Property("host", moi.StringSchema().MinLength(1)).
				Property("owner", moi.StringSchema().MinLength(1)).
				Property("repo", moi.StringSchema().MinLength(1)).
				Property("number", moi.IntegerSchema().Minimum(1)).
				Property("base_sha", moi.StringSchema().Pattern("^[0-9a-fA-F]{40}$")).
				Property("head_sha", moi.StringSchema().Pattern("^[0-9a-fA-F]{40}$")).
				Required("host", "owner", "repo", "number", "base_sha", "head_sha").
				AdditionalProperties(false)).
			Required("operation").
			AdditionalProperties(false)).
		Required("workspace_id", "credential", "request").
		AdditionalProperties(false)
}

func schemaCodexRunOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Result from one Codex request. A parallel request returns one result per independent task; credentials are never returned.").
		Property("ok", moi.BooleanSchema()).
		Property("provider", moi.StringSchema()).
		Property("operation", moi.StringSchema()).
		Property("model", moi.StringSchema()).
		Property("workspace_ref", moi.StringSchema()).
		Property("parallel_task_count", moi.IntegerSchema().Minimum(0)).
		Property("result", schemaNullableAny().
			Description("Final result emitted by Codex. Omitted for connection_test.").
			Example(map[string]any{"summary": "Repository analysis complete."})).
		Property("request_id", moi.StringSchema()).
		Required("ok", "provider", "operation").
		AdditionalProperties(false)
}

func schemaGrafanaReadInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Internal Grafana channel read payload assembled by Catalog. Runtime callers must not pass credentials directly. Recent/latest/time-window metric and log queries must use range operations with explicit start, end, and step instead of repeated instant queries.").
		Property("request_id", moi.StringSchema()).
		Property("workspace_id", moi.StringSchema().MinLength(1)).
		Property("credential", moi.NewSchema().
			Property("grafana_url", moi.StringSchema().MinLength(1)).
			Property("authorization", moi.StringSchema().MinLength(1)).
			Required("grafana_url", "authorization").
			AdditionalProperties(false)).
		Property("request", moi.NewSchema().
			Property("operation", moi.StringSchema().Enum("health", "list_datasources", "query_prometheus", "query_prometheus_range", "query_loki", "query_loki_range").Description("Grafana read operation. For recent/latest/time-window metric or log questions, use query_prometheus_range or query_loki_range with explicit start/end/step instead of repeating instant queries.")).
			Property("grafana_url", moi.StringSchema().Description("Optional exact Grafana base URL consistency check. A value different from the bound channel credential is rejected.")).
			Property("datasource_uid", moi.StringSchema().Description("Grafana datasource UID for Prometheus or Loki query operations.")).
			Property("query", moi.StringSchema().Description("PromQL or LogQL expression.")).
			Property("time", moi.StringSchema().Description("Optional absolute evaluation time for instant query_prometheus or query_loki. Do not repeat instant queries to scan a range; use start/end/step with a range operation.")).
			Property("start", moi.StringSchema().Description("Absolute range start time for query_prometheus_range or query_loki_range. Required for recent/latest/time-window queries.")).
			Property("end", moi.StringSchema().Description("Absolute range end time for query_prometheus_range or query_loki_range. Required for recent/latest/time-window queries.")).
			Property("step", moi.StringSchema().Description("Range query resolution step. Required for Prometheus range queries and optional for Loki range queries; choose a bounded step for the requested window.")).
			Property("limit", moi.StringSchema().Description("Optional Loki log entry limit. Defaults to 50; maximum 100. Increase only when more log lines are explicitly needed.")).
			Property("direction", moi.StringSchema().Enum("forward", "backward").Description("Optional Loki query direction.")).
			Required("operation").
			AdditionalProperties(false)).
		Required("workspace_id", "credential", "request").
		AdditionalProperties(false)
}

func schemaKubernetesReadInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Internal Kubernetes channel read payload assembled by Catalog. Runtime callers must not pass credentials directly. This tool has no since/time filter; for recent event or workload troubleshooting use namespace, name, label_selector, field_selector, and a small limit instead of cluster-wide pagination.").
		Property("request_id", moi.StringSchema()).
		Property("workspace_id", moi.StringSchema().MinLength(1)).
		Property("credential", moi.NewSchema().
			Property("api_server", moi.StringSchema().MinLength(1)).
			Property("auth_method", moi.StringSchema().Enum("service_account_token", "client_certificate").Description("Required Kubernetes authentication method in the internal Catalog-to-Worker credential contract.")).
			Property("certificate_authority_data", moi.StringSchema().Description("Optional base64 PEM trust roots for the Kubernetes API Server.")).
			Property("tls_server_name", moi.StringSchema().Description("Optional TLS ServerName override for Kubernetes API Server certificate validation.")).
			Property("bearer_token", moi.StringSchema().Description("Required only when auth_method is service_account_token.")).
			Property("client_certificate_data", moi.StringSchema().Description("Required with client_key_data only when auth_method is client_certificate.")).
			Property("client_key_data", moi.StringSchema().Description("Required with client_certificate_data only when auth_method is client_certificate.")).
			Required("api_server", "auth_method").
			AdditionalProperties(false)).
		Property("request", moi.NewSchema().
			Property("operation", moi.StringSchema().Enum("version", "list_namespaces", "list_nodes", "list_pods", "get_pod", "list_deployments", "list_services", "list_events").Description("Kubernetes read operation. This tool has no since/time filter; use scoped list queries with a bounded limit for recent troubleshooting.")).
			Property("namespace", moi.StringSchema().Description("Namespace for namespaced operations. Provide it when known to avoid broad cluster scans.")).
			Property("name", moi.StringSchema().Description("Resource name for get operations.")).
			Property("label_selector", moi.StringSchema().Description("Kubernetes labelSelector query parameter. Use selectors to narrow broad list queries when labels are known.")).
			Property("field_selector", moi.StringSchema().Description("Kubernetes fieldSelector query parameter. Use it to narrow events or specific resource names when applicable.")).
			Property("limit", moi.IntegerSchema().Minimum(1).Maximum(100).Description("Maximum resource count. Defaults to 50 for list operations; use a smaller bounded limit for recent event or workload troubleshooting. This tool has no since/time filter.")).
			Property("continue", moi.StringSchema().Description("Kubernetes continue token returned by a previous list response. Use only when has_next=true and more resources are explicitly needed; do not use continue to approximate a time window.")).
			Required("operation").
			AdditionalProperties(false)).
		Required("workspace_id", "credential", "request").
		AdditionalProperties(false)
}

func schemaFeishuMessageSendInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Internal Feishu channel message payload assembled by Catalog. Runtime callers must not pass credentials directly.").
		Property("request_id", moi.StringSchema()).
		Property("workspace_id", moi.StringSchema().MinLength(1)).
		Property("credential", moi.NewSchema().
			Property("app_id", moi.StringSchema().MinLength(1)).
			Property("app_secret", moi.StringSchema().MinLength(1)).
			Required("app_id", "app_secret").
			AdditionalProperties(false)).
		Property("message", moi.NewSchema().
			Property("receive_id_type", moi.StringSchema().Enum("open_id", "user_id", "union_id", "email", "chat_id")).
			Property("receive_id", moi.StringSchema().MinLength(1)).
			Property("msg_type", moi.StringSchema().Enum("text", "post", "interactive")).
			Property("content", moi.StringSchema()).
			Property("content_json", schemaAnyObject()).
			Required("receive_id").
			AdditionalProperties(false)).
		Required("workspace_id", "credential", "message").
		AdditionalProperties(false)
}

func schemaFeishuConnectionTestInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Internal Feishu connection-test payload assembled by Catalog. It only requests a tenant access token and must not send messages or change provider resources.").
		Property("request_id", moi.StringSchema()).
		Property("workspace_id", moi.StringSchema().MinLength(1)).
		Property("credential", moi.NewSchema().
			Property("app_id", moi.StringSchema().MinLength(1)).
			Property("app_secret", moi.StringSchema().MinLength(1)).
			Required("app_id", "app_secret").
			AdditionalProperties(false)).
		Required("workspace_id", "credential").
		AdditionalProperties(false)
}

func schemaSlackConnectionTestInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Internal Slack connection-test payload assembled by Catalog. It only calls auth.test and must not send messages or change provider resources.").
		Property("request_id", moi.StringSchema()).
		Property("workspace_id", moi.StringSchema().MinLength(1)).
		Property("credential", moi.NewSchema().
			Property("bot_token", moi.StringSchema().MinLength(1)).
			Required("bot_token").
			AdditionalProperties(false)).
		Required("workspace_id", "credential").
		AdditionalProperties(false)
}

func schemaSlackMessageSendInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Internal Slack channel message payload assembled by Catalog. Runtime callers must not pass credentials directly.").
		Property("request_id", moi.StringSchema()).
		Property("workspace_id", moi.StringSchema().MinLength(1)).
		Property("credential", moi.NewSchema().
			Property("bot_token", moi.StringSchema().MinLength(1)).
			Required("bot_token").
			AdditionalProperties(false)).
		Property("message", moi.NewSchema().
			Property("channel_id", moi.StringSchema().MinLength(1)).
			Property("text", moi.StringSchema()).
			Property("blocks", moi.ArraySchema().Items(schemaAnyObject())).
			Property("attachments", moi.ArraySchema().Items(schemaAnyObject())).
			Property("thread_ts", moi.StringSchema()).
			Required("channel_id").
			AdditionalProperties(false)).
		Required("workspace_id", "credential", "message").
		AdditionalProperties(false)
}

func schemaDataTableReadInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Read rows from a workspace MatrixOne table selected from the MOI catalog. Use this as a table source node when the user starts a workflow from an existing MOI table.").
		Property("table_ref", schemaCatalogResourceRef().Description("Catalog table selected by the user. The UI submits the selected table name and catalog/database/table IDs.")).
		Property("limit", moi.IntegerSchema().Minimum(0).Description("Maximum rows to read. Set 0 only when the workflow intentionally needs all rows.")).
		Property("timeout_seconds", moi.IntegerSchema().Minimum(0).Description("Optional execution timeout in seconds.")).
		Required("table_ref").
		AdditionalProperties(true)
}

func schemaDataTableReadOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Rows read from a workspace table. Downstream nodes should consume rows for SQL-independent processing or table sinks.").
		Property("table_ref", schemaCatalogResourceRef().Description("Catalog table reference used for the read.")).
		Property("table_name", moi.StringSchema().Description("Source table name.")).
		Property("database", moi.StringSchema().Description("Database used for the read.")).
		Property("rows", moi.ArraySchema().Items(schemaAnyObject()).Description("Rows read from the table, each row keyed by column name.")).
		Property("sql", moi.StringSchema().Description("SQL statement executed by the worker.")).
		Property("elapsed_ms", moi.IntegerSchema().Description("Execution time in milliseconds.")).
		Property("truncated", moi.BooleanSchema().Description("Whether rows were truncated by the limit.")).
		Required("table_name").
		AdditionalProperties(true)
}

func schemaLLMOutputGenerateInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Generate natural-language output with a workspace LLM model. Use for model inference over upstream text, questions, prompts, or instructions.").
		Property("model", moi.StringSchema().MinLength(1).Description("Workspace LLM model name selected by the user.")).
		Property("prompt", moi.StringSchema().MinLength(1).Description("Prompt sent as the user message.")).
		Property("system_prompt", moi.StringSchema().Description("Optional system prompt. Use only when the user explicitly specifies role or behavior constraints.")).
		Property("temperature", moi.NumberSchema().Minimum(0).Maximum(2).Description("Optional sampling temperature.")).
		Property("max_tokens", moi.IntegerSchema().Minimum(1).Description("Optional maximum generated tokens.")).
		Required("model", "prompt").
		AdditionalProperties(true)
}

func schemaLLMOutputGenerateOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("LLM inference output. Downstream nodes can consume text or result.").
		Property("text", moi.StringSchema().Description("Generated assistant text.")).
		Property("result", moi.StringSchema().Description("Alias of generated assistant text for generic downstream consumers.")).
		Property("model", moi.StringSchema().Description("Model used for generation.")).
		Required("text").
		AdditionalProperties(true)
}

func schemaWorkflowTriggerInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Trigger another published workflow version from the current workflow. Use this for product-level workflow chaining when the user explicitly asks to run an existing workflow.").
		Property("workflow_version_id", moi.StringSchema().MinLength(1).Description("Published workflow version ID to execute.")).
		Property("task_name", moi.StringSchema().MinLength(1).Description("Task name for the triggered execution.")).
		Property("data", schemaNullableAny().Description("Optional data JSON passed to the triggered workflow. Missing or null means no data payload.").
			Example(map[string]interface{}{"source": "upstream"}).
			Range("Any valid JSON value passed as the triggered workflow data payload, or null.")).
		Property("vars", schemaNullableAny().Description("Optional vars JSON passed to the triggered workflow. Missing or null means no vars payload.").
			Example(map[string]interface{}{"env": "prod"}).
			Range("Any valid JSON value passed as the triggered workflow vars payload, or null.")).
		Required("workflow_version_id", "task_name").
		AdditionalProperties(true)
}

func schemaWorkflowTriggerOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Triggered workflow execution identifiers.").
		Property("workflow_version_id", moi.StringSchema().Description("Workflow version that was triggered.")).
		Property("task_id", moi.StringSchema().Description("Mowl task ID created for the triggered workflow.")).
		Property("workflow_execution_id", moi.StringSchema().Description("Workflow execution ID created by Catalog/Mowl, when available.")).
		Property("status", moi.IntegerSchema().Description("Initial task status code returned by Mowl.")).
		Required("workflow_version_id", "task_id").
		AdditionalProperties(true)
}

func schemaRunSQLOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("SQL execution result. Bind rows to catalog/table sinks when the SQL is a SELECT.").
		Property("sql", moi.StringSchema().Description("Executed SQL statement.")).
		Property("rows", moi.ArraySchema().Items(schemaAnyObject()).Description("Result rows for SELECT queries; each row is an object keyed by column name.")).
		Property("affected_rows", moi.IntegerSchema().Description("Affected rows for write statements, or result count when reported by the backend.")).
		Property("elapsed_ms", moi.IntegerSchema().Description("Execution time in milliseconds.")).
		Property("output_table", moi.StringSchema().Description("Materialized output table when output_table was set.")).
		Property("truncated", moi.BooleanSchema().Description("Whether rows were truncated by max_rows or backend limits.")).
		Property("write_mode", moi.StringSchema().Description("Write mode used for materialization.")).
		Property("materialization_strategy", moi.StringSchema().Description("Materialization strategy used for output_table.")).
		Required("sql").
		AdditionalProperties(true)
}

func schemaSQLProcessOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Stable SQL execution result envelope. SELECT/SHOW/WITH queries put result records in rows. DML/DDL statements normally use affected_rows and do not produce business rows. When table_ref was bound, the resolved Catalog reference and physical names are echoed for lineage.").
		Property("sql", moi.StringSchema().Description("Executed SQL statement (with ${source_table} already expanded when used).")).
		Property("rows", moi.ArraySchema().Items(schemaAnyObject()).Description("Result rows for query statements; each row is an object keyed by column name.")).
		Property("affected_rows", moi.IntegerSchema().Description("Affected rows for DML/DDL statements when reported by the backend.")).
		Property("elapsed_ms", moi.IntegerSchema().Description("Execution time in milliseconds.")).
		Property("truncated", moi.BooleanSchema().Description("Whether returned rows were truncated by backend limits.")).
		Property("table_ref", schemaCatalogResourceRef().Description("Catalog table reference when table_ref input was bound.")).
		Property("database", moi.StringSchema().Description("Resolved database name when table_ref was bound.")).
		Property("table_name", moi.StringSchema().Description("Resolved table name when table_ref was bound.")).
		Property("source_table", moi.StringSchema().Description("Quoted database.table FQN substituted for ${source_table} when table_ref was bound.")).
		Required("sql").
		AdditionalProperties(false)
}

func schemaAPIRequestOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("HTTP response envelope returned by the API node. HTTP 4xx/5xx responses are represented by status_code and body; transport failures return a node error.").
		Property("status_code", moi.IntegerSchema().Description("HTTP response status code.")).
		Property("headers", schemaAnyObject().Description("HTTP response headers object. Each key is a response header name and each value is the merged header string reported by Go's http client.")).
		Property("body", moi.StringSchema().Description("Raw HTTP response body.")).
		Required("status_code", "headers", "body").
		AdditionalProperties(false)
}

func schemaChannelMessageSendOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Channel message send result. Secrets and access tokens are never returned.").
		Property("ok", moi.BooleanSchema()).
		Property("provider", moi.StringSchema()).
		Property("status_code", moi.IntegerSchema()).
		Property("response", schemaAnyObject()).
		Required("ok", "provider").
		AdditionalProperties(true)
}

func schemaChannelConnectionTestOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Channel connection-test result. Secrets and access tokens are never returned.").
		Property("ok", moi.BooleanSchema()).
		Property("provider", moi.StringSchema()).
		Property("request_id", moi.StringSchema()).
		Property("team_id", moi.StringSchema()).
		Required("ok", "provider").
		AdditionalProperties(true)
}

func schemaSQLPipelineInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("mode", moi.StringSchema().Enum("replace_by_clone").Description("Optional execution mode. Omit for legacy behavior; replace_by_clone builds a staging table and swaps the target with CREATE TABLE ... CLONE inside a transaction.")).
		Property("target_table", moi.StringSchema().Description("Target table for replace_by_clone mode, as table or database.table. Statements in that mode must write to {{target_table}}, which is replaced with the generated staging table.")).
		Property("pre_statements", moi.ArraySchema().Items(moi.StringSchema().MinLength(1)).Description("SQL statements executed outside transaction before main statements (e.g., DDL).")).
		Property("statements", moi.ArraySchema().Items(moi.StringSchema().MinLength(1)).Description("SQL statements executed inside a single transaction (e.g., TRUNCATE + INSERT...SELECT).")).
		Property("post_statements", moi.ArraySchema().Items(moi.StringSchema().MinLength(1)).Description("SQL statements executed outside transaction after commit (e.g., batch updates).")).
		Property("database", moi.StringSchema().Description("Target database name. Defaults to workspace database if omitted.")).
		AdditionalProperties(true)
}

func schemaSQLPipelineOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("results", moi.ArraySchema().Items(
			moi.NewSchema().
				Property("index", moi.IntegerSchema()).
				Property("affected_rows", moi.IntegerSchema()).
				Property("error", moi.StringSchema()).
				Property("elapsed_ms", moi.IntegerSchema()),
		)).
		Property("total_affected_rows", moi.IntegerSchema()).
		Property("statements_executed", moi.IntegerSchema()).
		Property("elapsed_ms", moi.IntegerSchema()).
		Property("mode", moi.StringSchema()).
		Property("target_table", moi.StringSchema()).
		Property("staging_table", moi.StringSchema()).
		AdditionalProperties(true)
}

func schemaFileRef() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("file_id", moi.StringSchema()).
		Property("file_name", moi.StringSchema()).
		Required("file_id").
		AdditionalProperties(true)
}

func schemaUnifiedExtractInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Extract structured JSON from text, documents, or file references with an LLM. Use this when the user asks to identify fields/entities/tables from document content. Use custom operators instead when deterministic code is enough.").
		Property("text", moi.StringSchema().Description("Single text input to extract from. Use when upstream output is plain text.")).
		Property("documents", moi.ArraySchema().Items(schemaDocument()).Description("Document array to extract from, typically from parse or split output.")).
		Property("files", moi.ArraySchema().Items(schemaFileRef()).Description("File references for direct file-based extraction when no parsed documents are available.")).
		Property("json_schema", schemaAnyObject().Description("JSON Schema describing the extraction output fields. Downstream table writes can reuse the same schema.")).
		Property("mode", moi.StringSchema().Enum("n_to_1", "n_to_n").Default("n_to_1").Description("n_to_1 aggregates all inputs into one extraction result; n_to_n returns one result per input document/file.")).
		Property("instruction", moi.StringSchema().Description("Natural-language extraction instruction, for example what fields to extract and normalization rules.")).
		Property("model", moi.StringSchema().Description("Chat model used for extraction. Normally bound from runtime/model selector.")).
		Property("page_selector", moi.StringSchema().Description("Optional page range selector, for example 1-3,5,8-10. Only meaningful when this node directly processes single-file inputs through files or page-aware documents; parse->extract workflows should put page_selector on moi:document.parse instead.")).
		Property("retry_count", moi.IntegerSchema().Minimum(0).Description("Retry attempts for LLM extraction failures.")).
		Property("max_concurrency", moi.IntegerSchema().Minimum(1).Description("Maximum concurrent extraction calls in n_to_n mode.")).
		Property("use_kit_extractor", moi.BooleanSchema().Description("Use the kit extractor implementation when enabled by deployment.")).
		Property("mock_result", moi.StringSchema().Description("Test-only single mocked extraction result.")).
		Property("mock_results", moi.ArraySchema().Items(moi.StringSchema()).Description("Test-only mocked extraction results.")).
		Property("output_zip", moi.BooleanSchema().Description("Whether to produce parse/extract artifact file IDs for ZIP output.")).
		AdditionalProperties(true)
}

func schemaUnifiedExtractOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Structured extraction result. `result` is used for n_to_1; `results` is used for n_to_n. Values are JSON strings unless the extractor implementation returns richer documents/artifacts.").
		Property("mode", moi.StringSchema().Description("Extraction mode used: n_to_1 or n_to_n.")).
		Property("result", moi.StringSchema().Description("Single extracted JSON result, usually for n_to_1. Downstream custom/table nodes may parse this JSON string.")).
		Property("results", moi.ArraySchema().Items(moi.StringSchema()).Description("Per-input extracted JSON results, usually for n_to_n.")).
		Property("sources_tracking", moi.NewSchema().AdditionalProperties(true).Description("Mapping from extracted values back to source documents/pages when available.")).
		Property("metrics", schemaAnyObject().Description("Extraction runtime metrics.")).
		Property("retry_count", moi.IntegerSchema().Description("Retry attempts used.")).
		Property("max_concurrency", moi.IntegerSchema().Description("Concurrency used.")).
		Property("document_metadata", moi.ArraySchema().Items(schemaAnyObject()).Description("Metadata for documents/files used during extraction.")).
		Property("md_file_id", moi.StringSchema().Description("Markdown artifact file ID when output_zip/artifact output is enabled.")).
		Property("layout_file_id", moi.StringSchema().Description("Layout JSON artifact file ID when output_zip/artifact output is enabled.")).
		Property("file_name", moi.StringSchema().Description("Source or artifact file name.")).
		Property("output_zip", moi.BooleanSchema().Description("Whether artifact ZIP output was requested.")).
		Property("documents", moi.ArraySchema().Items(schemaDocument()).Description("Optional documents passed through or produced by extraction.")).
		AnyOf(
			moi.NewSchema().Property("mode", moi.StringSchema().Enum("n_to_1")).Required("mode", "result"),
			moi.NewSchema().Property("mode", moi.StringSchema().Enum("n_to_n")).Required("mode", "results"),
		).
		AdditionalProperties(true)
}

func schemaFileReadTextInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("file_id", moi.StringSchema().MinLength(1)).
		Required("file_id").
		AdditionalProperties(true)
}

func schemaFileReadTextOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Disabled: file content must not be passed through workflow data.").
		AdditionalProperties(true)
}

func schemaFileMetadataGetInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("file_id", moi.StringSchema().MinLength(1)).
		Required("file_id").
		AdditionalProperties(true)
}

func schemaReadDocsInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("file_id", moi.StringSchema().MinLength(1)).
		Property("limit", moi.IntegerSchema().Minimum(1)).
		Required("file_id").
		AdditionalProperties(true)
}

func schemaReadDocsOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Disabled: file content must not be passed through workflow data.").
		AdditionalProperties(true)
}

func schemaWriteDocsInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Write a documents array as JSONL into workspace files. Use this when downstream nodes need file_id/file_ids for parsed or transformed documents instead of in-memory documents.").
		Property("documents", moi.ArraySchema().Items(schemaAnyObject()).Description("Documents to serialize as JSONL. Usually from parse, split, extraction, or a custom document-producing node.")).
		Property("output_file_name", moi.StringSchema().Description("Optional output file name. If omitted, the worker derives a default name.")).
		Required("documents").
		AdditionalProperties(true)
}

func schemaWriteDocsOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("File write result for document JSONL output.").
		Property("file_id", moi.StringSchema().Description("Primary created file ID.")).
		Property("file_name", moi.StringSchema().Description("Primary created file name.")).
		Property("count", moi.IntegerSchema().Description("Number of documents written.")).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Created file IDs. Usually one file, but kept as an array for batch compatibility.")).
		Property("file_names", moi.ArraySchema().Items(moi.StringSchema()).Description("Created file names.")).
		Required("file_id", "file_name", "count").
		AdditionalProperties(true)
}

func schemaCatalogResolveInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("name", moi.StringSchema().MinLength(1)).
		Required("name").
		AdditionalProperties(true)
}

func schemaCatalogResolveOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("catalog_id", moi.IntegerSchema()).
		Property("name", moi.StringSchema()).
		Required("catalog_id", "name").
		AdditionalProperties(true)
}

func schemaDatabaseResolveInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("catalog_id", moi.IntegerSchema().Minimum(1)).
		Property("name", moi.StringSchema().MinLength(1)).
		Required("catalog_id", "name").
		AdditionalProperties(true)
}

func schemaDatabaseResolveOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("database_id", moi.IntegerSchema()).
		Property("name", moi.StringSchema()).
		Required("database_id", "name").
		AdditionalProperties(true)
}

func schemaVolumeResolveInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("database_id", moi.IntegerSchema().Minimum(1)).
		Property("name", moi.StringSchema().MinLength(1)).
		Required("database_id", "name").
		AdditionalProperties(true)
}

func schemaVolumeResolveOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("volume_id", moi.IntegerSchema()).
		Property("name", moi.StringSchema()).
		Required("volume_id", "name").
		AdditionalProperties(true)
}

func schemaVolumeEnsureInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("database_id", moi.IntegerSchema().Minimum(1)).
		Property("name", moi.StringSchema().MinLength(1)).
		Property("comment", moi.StringSchema()).
		Property("parent_id", moi.IntegerSchema().Minimum(1)).
		Required("database_id", "name").
		AdditionalProperties(true)
}

func schemaVolumeEnsureOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("volume_id", moi.IntegerSchema()).
		Property("name", moi.StringSchema()).
		Property("created", moi.BooleanSchema()).
		Required("volume_id", "name", "created").
		AdditionalProperties(true)
}

func schemaVolumeFilesAddInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("volume_id", moi.IntegerSchema().Minimum(1)).
		Property("file_id", moi.StringSchema()).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema())).
		Required("volume_id").
		AdditionalProperties(true)
}

func schemaVolumeFilesAddOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("volume_id", moi.IntegerSchema()).
		Property("added", moi.IntegerSchema()).
		Required("volume_id", "added").
		AdditionalProperties(true)
}

func schemaVolumeFilesListInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("volume_id", moi.IntegerSchema().Minimum(1)).
		Property("page_size", moi.IntegerSchema().Minimum(1)).
		Required("volume_id").
		AdditionalProperties(true)
}

func schemaVolumeFilesListOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("items", moi.ArraySchema().Items(schemaAnyObject())).
		Property("total", moi.IntegerSchema()).
		Required("items", "total").
		AdditionalProperties(true)
}

func schemaVolumeFilesMoveInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("source_volume_id", moi.IntegerSchema().Minimum(1)).
		Property("target_volume_id", moi.IntegerSchema().Minimum(1)).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema())).
		Required("source_volume_id", "target_volume_id", "file_ids").
		AdditionalProperties(true)
}

func schemaVolumeFilesMoveOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("source_volume_id", moi.IntegerSchema()).
		Property("target_volume_id", moi.IntegerSchema()).
		Property("moved", moi.IntegerSchema()).
		Required("source_volume_id", "target_volume_id", "moved").
		AdditionalProperties(true)
}

func schemaVolumeFilesRemoveInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("volume_id", moi.IntegerSchema().Minimum(1)).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema())).
		Required("volume_id", "file_ids").
		AdditionalProperties(true)
}

func schemaVolumeFilesRemoveOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("volume_id", moi.IntegerSchema()).
		Property("removed", moi.IntegerSchema()).
		Required("volume_id", "removed").
		AdditionalProperties(true)
}

func schemaCDHExportS3Input() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("cdh_config_id", moi.IntegerSchema()).
		Property("database", moi.StringSchema()).
		Property("table", moi.StringSchema()).
		Required("cdh_config_id", "database", "table").
		AdditionalProperties(true)
}

func schemaCDHExportS3Output() *moi.SchemaBuilder {
	return moi.NewSchema().
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema())).
		Property("schema_file_id", moi.StringSchema()).
		Property("total_rows", moi.IntegerSchema()).
		Property("file_count", moi.IntegerSchema()).
		Required("file_ids", "schema_file_id", "total_rows", "file_count").
		AdditionalProperties(true)
}

func schemaS3ToMOImportInput() *moi.SchemaBuilder {
	columnMapEntry := moi.NewSchema().
		Property("table_column", moi.StringSchema().Description("Destination table column name.")).
		Property("col_num_in_file", moi.IntegerSchema().Description("1-based source file column number.")).
		Property("data_from", moi.IntegerSchema().Description("Optional source data position marker used by import backend.")).
		Required("table_column")

	return moi.NewSchema().
		Description("Import CSV, Parquet, ORC, or Excel files from Catalog/S3 into an existing MatrixOne table in the current workspace. For Excel imports, numeric MatrixOne target columns use raw cell values while other target columns retain formatted cell values; the destination table type controls precision and scale. The source files are identified by file_ids and interpreted under base_path; table columns must match the file data or be configured through column_mapping.").
		Property("base_path", moi.StringSchema().Description("Base Catalog/S3 path used by the import backend to resolve the selected file IDs, for example /data/imports.")).
		Property("table_name", moi.StringSchema().Description("Destination MatrixOne table name. The table must already exist and match the imported columns unless column_mapping is provided.")).
		Property("mo_database", moi.StringSchema().Description("Destination MatrixOne database name in the current workspace.")).
		Property("delimiter", moi.StringSchema().Description("CSV field delimiter. Leave empty to use the import backend default comma delimiter.")).
		Property("line_separator", moi.StringSchema().Description("CSV line separator. Leave empty to use the import backend default newline separator.")).
		Property("start_row", moi.IntegerSchema().Description("For CSV/Excel import, skip this many leading rows before loading data. Use start_row=1 when importing CSV produced by moi:catalog.sink.write from rows+columns, because the first row is the columns header.")).
		Property("sheet_name", moi.StringSchema().Description("Excel worksheet name to import. Leave empty to use the active worksheet.")).
		Property("overwrite", moi.BooleanSchema().Description("Whether to truncate the destination table before importing the selected files.")).
		Property("file_ids", moi.ArraySchema().Items(moi.StringSchema()).Description("Source data file IDs produced by Catalog read, catalog.sink.write, or another file-producing node. These files should contain CSV, Parquet, ORC, or Excel data.")).
		Property("conflict_policy", moi.IntegerSchema().Description("Backend-defined conflict policy.")).
		Property("column_mapping", moi.ArraySchema().Items(columnMapEntry).Description("Optional mapping from source file columns to destination table columns. Use this when file column order does not match the table.")).
		Property("quote_char", moi.StringSchema().Description("CSV quote character. Leave empty to use the import backend default double quote.")).
		Required("base_path", "table_name", "mo_database", "file_ids").
		AdditionalProperties(true)
}

func schemaS3ToMOImportOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("MatrixOne import result including imported row count, imported file count, destination table name, and compact runtime diagnostics for OOM/resource analysis.").
		Property("total_rows", moi.IntegerSchema().Description("Number of rows imported into the destination MatrixOne table.")).
		Property("file_count", moi.IntegerSchema().Description("Number of source files imported.")).
		Property("table_name", moi.StringSchema().Description("Destination MatrixOne table name.")).
		Property("data_lineage_analysis", schemaAnyObject().Description("Compact data-flow summary: source file count, schema file presence, file extensions, target database/table, column counts, conflict policy, overwrite flag, start row, and imported rows.")).
		Property("memory_usage_analysis", schemaAnyObject().Description("Go runtime memory summary captured during the import: start/end/peak allocation and system bytes, GC counters, and peak parsed rows/cells held in memory.")).
		Property("cache_usage_analysis", schemaAnyObject().Description("Cache/download-path observation. S3-to-MO currently streams go-sdk file downloads to temp files; go-worker does not expose fileservice cache counters.")).
		Property("disk_usage_analysis", schemaAnyObject().Description("Worker-visible local disk usage summary: temp storage scope, peak temp bytes/file count, downloaded bytes, XLS conversion temp directories, cleanup behavior, and disk risk observation.")).
		Property("temp_file_usage_analysis", schemaAnyObject().Description("Temporary local file usage summary: import temp directory base name, peak bytes/file count, downloaded bytes, XLS conversion temp directory count, and cleanup flag.")).
		Property("s3_usage_analysis", schemaAnyObject().Description("Fileservice/S3 usage summary: metadata lookup count, download count, downloaded bytes, and download duration.")).
		Property("timing_analysis", schemaAnyObject().Description("Wall-clock timing summary for total import, download, schema validation, parsing, and MatrixOne write stages.")).
		Property("oom_risk_analysis", moi.ArraySchema().Items(schemaAnyObject()).Description("Structured risk entries explaining observed memory, disk, S3, and MatrixOne transaction pressure points for this run.")).
		Required("total_rows", "file_count", "table_name").
		AdditionalProperties(true)
}

func schemaParserConfig() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Parser configuration flags controlling preprocess, layout, and enrichment behaviors.").
		Property("debug_enabled", moi.BooleanSchema().Description("Enable debug artifacts and tracing output.").Default(false).Example(false)).
		Property("max_workers", moi.IntegerSchema().Description("Upper bound for internal worker goroutines.").Minimum(1).Default(16).Example(16)).
		Property("pptx_normalize_before_pdf", moi.BooleanSchema().Description("Normalize PPTX before converting to PDF; valid only on the PDF conversion route.").Default(false).Example(false)).
		Property("enable_paddle_preprocess", moi.BooleanSchema().Description("Enable Paddle-based table-region preprocess for PDFs. Default false; MinerU layout is used by default.").Default(false).Example(false)).
		Property("save_table_image_file", moi.BooleanSchema().Description("Persist table crops as files for downstream table reasoning.").Default(false).Example(false)).
		Property("cast_table_as_image", moi.BooleanSchema().Description("Treat table blocks as image inputs for VLM enrichment.").Default(false).Example(false)).
		Property("cast_table_as_image_scope", moi.StringSchema().Description("Scope of table-to-image casting policy (engine-defined).").Example("all")).
		Property("enable_table_html_regeneration", moi.BooleanSchema().Description("Regenerate HTML for table blocks during enrichment.").Default(false).Example(false)).
		Property("enable_table_embedded_image_extraction", moi.BooleanSchema().Description("Extract images embedded in table cells.").Default(true).Example(true)).
		Property("enable_merged_table_split", moi.BooleanSchema().Description("Split merged table cells when needed.").Default(false).Example(false)).
		Property("enable_cross_page_table_merge", moi.BooleanSchema().Description("Merge table fragments across adjacent pages.").Default(false).Example(false)).
		Property("unmerge_table_cells", moi.BooleanSchema().Description("Attempt to unmerge merged table cells into explicit grid.").Default(false).Example(false)).
		Property("enable_table_inline_image_text", moi.BooleanSchema().Description("Use OCR/VLM text for inline table images.").Default(false).Example(false)).
		Property("enable_table_image_in_markdown", moi.BooleanSchema().Description("Emit table image links in markdown output.").Default(false).Example(false)).
		Property("flowchart_table_use_cells", moi.BooleanSchema().Description("Handle flowchart-like tables by cell-level reconstruction.").Default(false).Example(false)).
		Property("enable_vlm_title_detection", moi.BooleanSchema().Description("Enable VLM-assisted title detection.").Default(false).Example(false)).
		Property("enable_vlm_header_footer_detection", moi.BooleanSchema().Description("Enable VLM-assisted header/footer detection.").Default(false).Example(false)).
		Property("enable_formula_repair", moi.BooleanSchema().Description("Repair formula blocks during content enrichment.").Default(false).Example(false)).
		Property("enable_list_marker_repair", moi.BooleanSchema().Description("Repair broken list markers in OCR text.").Default(false).Example(false)).
		Property("enable_indent_detection", moi.BooleanSchema().Description("Infer indentation levels for list-like text blocks.").Default(false).Example(false)).
		Property("enable_fragment_merge", moi.BooleanSchema().Description("Merge adjacent text fragments before assembly.").Default(true).Example(true)).
		Property("enable_strikethrough_detection", moi.BooleanSchema().Description("Detect strikethrough content markers.").Default(false).Example(false)).
		Property("header_footer_min_occurrence", moi.IntegerSchema().Description("Minimum repeated occurrences to classify header/footer.").Minimum(1).Default(3).Example(3)).
		Property("header_footer_similarity_threshold", moi.NumberSchema().Description("Similarity threshold for header/footer grouping.").Minimum(0).Maximum(1).Default(0.8).Example(0.8)).
		Property("indent_spaces_per_level", moi.IntegerSchema().Description("Spaces represented by one indentation level.").Minimum(1).Default(2).Example(2)).
		Property("parser_concurrency", moi.IntegerSchema().Description("Max concurrent parser tasks in pipeline stages.").Minimum(1).Default(16).Example(16)).
		AdditionalProperties(true)
}

func schemaParserSource() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Input source descriptor for parser intake.").
		Property("file_id", moi.StringSchema().Description("Workspace file identifier to parse.").MinLength(1).Example("file_01JABCDXYZ")).
		Required("file_id").
		AdditionalProperties(true)
}

func schemaParserEnrichOutputItem() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Output of one enrichment branch (content/table/image).").
		Property("type", moi.StringSchema().Description("Enrichment branch type.").Enum("content", "table", "image").Example("content")).
		Property("result_blocks_file_id", moi.StringSchema().Description("File ID containing enriched blocks JSON.").Example("file_enriched_blocks_01")).
		Required("type", "result_blocks_file_id").
		AdditionalProperties(true)
}

func schemaParserIntakeInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Initial parser intake request.").
		Property("sources", moi.ArraySchema().Description("Input files to parse (first item is used).").Items(schemaParserSource()).Example([]interface{}{map[string]interface{}{"file_id": "file_01JABCDXYZ"}})).
		Property("options", schemaAnyObject().Description("Optional parser options merged into ParserConfig.").Example(map[string]interface{}{"parser_version": "v2"})).
		Required("sources").
		AdditionalProperties(true)
}

func schemaParserIntakeOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Resolved file/context metadata after intake stage.").
		Property("file_id", moi.StringSchema().Description("Original input file ID.").Example("file_01JABCDXYZ")).
		Property("file_type", moi.StringSchema().Description("Normalized file type used by parser.").Example("pdf")).
		Property("mime_type", moi.StringSchema().Description("Detected MIME type from source bytes.").Example("application/pdf")).
		Property("parser_version", moi.StringSchema().Description("Resolved parser version routed by strategy.").Example("v2")).
		Property("workspace_id", moi.StringSchema().Description("Workspace scope for downstream operations.").Example("ws_01HXYZ")).
		Property("config", schemaParserConfig().Description("Merged parser configuration for downstream stages.")).
		Property("options", schemaAnyObject().Description("Original parser options for downstream stages that need non-ParserConfig values such as vlm_ocr_model.")).
		Required("file_id", "file_type", "mime_type", "parser_version", "workspace_id", "config").
		AdditionalProperties(true)
}

func schemaParserConvertInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Convert stage input (Office to PDF when required).").
		Property("file_id", moi.StringSchema().Description("Input file ID to convert.").MinLength(1).Example("file_01JABCDXYZ")).
		Property("file_type", moi.StringSchema().Description("Input file type, e.g. docx/pptx/pdf.").MinLength(1).Example("docx")).
		Property("config", schemaParserConfig().Description("Optional parser config override.")).
		Required("file_id", "file_type").
		AdditionalProperties(true)
}

func schemaParserConvertOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Convert stage output.").
		Property("pdf_file_id", moi.StringSchema().Description("Output PDF file ID.").Example("file_pdf_01")).
		Property("file_type", moi.StringSchema().Description("Converted file type (always pdf).").Example("pdf")).
		Property("original_file_type", moi.StringSchema().Description("Original input type before conversion.").Example("docx")).
		Required("pdf_file_id", "file_type", "original_file_type").
		AdditionalProperties(true)
}

func schemaParserPreprocessInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Preprocess stage input (table detection + whitening).").
		Property("file_id", moi.StringSchema().Description("Direct PDF file ID path.").Example("file_pdf_01")).
		Property("pdf_file_id", moi.StringSchema().Description("PDF file ID from convert output.").Example("file_pdf_01")).
		Property("config", schemaParserConfig().Description("Optional parser config override.")).
		AdditionalProperties(true)
}

func schemaParserPreprocessOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Preprocess stage output.").
		Property("pdf_file_id", moi.StringSchema().Description("Whitened (or original) PDF file ID.").Example("file_pdf_preprocessed_01")).
		Property("table_regions_file_id", moi.StringSchema().Description("JSON file ID storing detected table regions.").Example("file_table_regions_01")).
		Property("placeholder_mapping_file_id", moi.StringSchema().Description("JSON file ID storing placeholder mappings.").Example("file_placeholder_map_01")).
		Property("degraded", moi.BooleanSchema().Description("Whether preprocess degraded due to upstream dependency failure.").Example(false)).
		Required("pdf_file_id", "table_regions_file_id", "placeholder_mapping_file_id", "degraded").
		AdditionalProperties(true)
}

func schemaParserLayoutInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Layout extraction stage input (MinerU).").
		Property("pdf_file_id", moi.StringSchema().Description("PDF file ID for layout extraction.").MinLength(1).Example("file_pdf_preprocessed_01")).
		Property("config", schemaParserConfig().Description("Optional parser config override.")).
		Required("pdf_file_id").
		AdditionalProperties(true)
}

func schemaParserLayoutOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Layout extraction stage output.").
		Property("layout_file_id", moi.StringSchema().Description("Layout JSON file ID.").Example("file_layout_json_01")).
		Property("md_file_id", moi.StringSchema().Description("Markdown file ID produced from layout.").Example("file_layout_md_01")).
		Property("full_text", moi.StringSchema().Description("Flattened full text extracted from layout.").Example("Section 1 ...")).
		Required("layout_file_id", "md_file_id", "full_text").
		AdditionalProperties(true)
}

func schemaParserStructureInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Structure stage input (block creation + cleanup + split).").
		Property("layout_file_id", moi.StringSchema().Description("Layout JSON file ID from layout stage.").MinLength(1).Example("file_layout_json_01")).
		Property("table_regions_file_id", moi.StringSchema().Description("Optional table regions JSON file ID.").Example("file_table_regions_01")).
		Property("placeholder_mapping_file_id", moi.StringSchema().Description("Optional placeholder mapping JSON file ID.").Example("file_placeholder_map_01")).
		Property("md_file_id", moi.StringSchema().Description("Markdown file ID passthrough for assembly stage.").Example("file_layout_md_01")).
		Property("config", schemaParserConfig().Description("Optional parser config override.")).
		Required("layout_file_id").
		AdditionalProperties(true)
}

func schemaParserStructureOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Structure stage output split by block family.").
		Property("content_blocks_file_id", moi.StringSchema().Description("TEXT/TITLE/LIST/CODE/EQUATION blocks file ID.").Example("file_blocks_content_01")).
		Property("table_blocks_file_id", moi.StringSchema().Description("TABLE blocks file ID.").Example("file_blocks_table_01")).
		Property("image_blocks_file_id", moi.StringSchema().Description("IMAGE blocks file ID.").Example("file_blocks_image_01")).
		Property("other_blocks_file_id", moi.StringSchema().Description("HEADER/FOOTER/DISCARDED blocks file ID.").Example("file_blocks_other_01")).
		Property("layout_file_id", moi.StringSchema().Description("Layout JSON file ID passthrough.").Example("file_layout_json_01")).
		Property("md_file_id", moi.StringSchema().Description("Markdown file ID passthrough.").Example("file_layout_md_01")).
		Required("content_blocks_file_id", "table_blocks_file_id", "image_blocks_file_id", "other_blocks_file_id", "layout_file_id", "md_file_id").
		AdditionalProperties(true)
}

func schemaParserEnrichInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Shared enrich stage input for content/table/image branches.").
		Property("input_blocks_file_id", moi.StringSchema().Description("Block JSON file ID to enrich.").MinLength(1).Example("file_blocks_content_01")).
		Property("config", schemaParserConfig().Description("Optional parser config override.")).
		Property("options", schemaAnyObject().Description("Original parser options. The image branch uses options.vlm_ocr_model to OCR IMAGE blocks with the selected VLM OCR model.")).
		Required("input_blocks_file_id").
		AdditionalProperties(true)
}

func schemaParserEnrichOutput() *moi.SchemaBuilder {
	return schemaParserEnrichOutputItem()
}

func schemaParserAssembleInput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Assembly stage input.").
		Property("enrich_results", moi.ArraySchema().Description("Outputs from enrich branches (content/table/image).").Items(schemaParserEnrichOutputItem()).Example([]interface{}{map[string]interface{}{"type": "content", "result_blocks_file_id": "file_blocks_content_enriched_01"}})).
		Property("other_blocks_file_id", moi.StringSchema().Description("Optional OTHER block file ID.").Example("file_blocks_other_01")).
		Property("layout_file_id", moi.StringSchema().Description("Layout JSON file ID passthrough.").Example("file_layout_json_01")).
		Property("md_file_id", moi.StringSchema().Description("Markdown file ID passthrough.").Example("file_layout_md_01")).
		Required("enrich_results", "layout_file_id", "md_file_id").
		AdditionalProperties(true)
}

func schemaParserAssembleOutput() *moi.SchemaBuilder {
	return moi.NewSchema().
		Description("Final parser output after block assembly.").
		Property("documents", moi.ArraySchema().Description("Assembled documents for downstream retrieval/LLM steps.").Items(schemaDocument())).
		Property("layout_file_id", moi.StringSchema().Description("Layout JSON file ID passthrough.").Example("file_layout_json_01")).
		Property("md_file_id", moi.StringSchema().Description("Markdown file ID passthrough.").Example("file_layout_md_01")).
		Property("plain_text", moi.StringSchema().Description("Flattened plain text built from assembled documents.").Example("Section 1 ...")).
		Required("documents", "layout_file_id", "md_file_id", "plain_text").
		AdditionalProperties(true)
}
