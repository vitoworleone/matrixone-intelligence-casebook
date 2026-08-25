package commands

import (

	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	sdk "github.com/matrixorigin/matrixflow/sdk/go-sdk"
)

// ExecuteSemanticModelCommand runs product SDK KnowledgeService methods via moi-cli.
func ExecuteSemanticModelCommand(ctx *Context, args []string) error {
	if len(args) == 0 {
		ShowSemanticModelHelp(nil)
		return nil
	}
	sub := args[0]
	switch sub {
	case "list":
		return runKnowledgeServiceList(ctx, args[1:])
	case "list-tags":
		return runKnowledgeServiceListTags(ctx, args[1:])
	case "create":
		return runKnowledgeServiceCreate(ctx, args[1:])
	case "get":
		return runKnowledgeServiceGet(ctx, args[1:])
	case "update":
		return runKnowledgeServiceUpdate(ctx, args[1:])
	case "delete":
		return runKnowledgeServiceDelete(ctx, args[1:])
	case "create-with-sources":
		return runKnowledgeServiceCreateWithSources(ctx, args[1:])
	case "create-empty":
		return runKnowledgeServiceCreateEmpty(ctx, args[1:])
	case "upload-local-file":
		return runKnowledgeServiceUploadLocalFile(ctx, args[1:])
	case "upload-model-local-file":
		return runKnowledgeServiceUploadModelLocalFile(ctx, args[1:])
	case "preview-source-selection-counts":
		return runKnowledgeServicePreviewSourceSelectionCounts(ctx, args[1:])
	case "preview-model-source-selection-counts":
		return runKnowledgeServicePreviewModelSourceSelectionCounts(ctx, args[1:])
	case "list-sources":
		return runKnowledgeServiceListSources(ctx, args[1:])
	case "list-sources-page":
		return runKnowledgeServiceListSourcesPage(ctx, args[1:])
	case "check-source-existence":
		return runKnowledgeServiceCheckSourceExistence(ctx, args[1:])
	case "append-sources":
		return runKnowledgeServiceAppendSources(ctx, args[1:])
	case "backfill-legacy-sources":
		return runKnowledgeServiceBackfillLegacySources(ctx, args[1:])
	case "delete-source":
		return runKnowledgeServiceDeleteSource(ctx, args[1:])
	case "get-source-document":
		return runKnowledgeServiceGetSourceDocument(ctx, args[1:])
	case "update-source-governance":
		return runKnowledgeServiceUpdateSourceGovernance(ctx, args[1:])
	case "import-initial-segments":
		return runKnowledgeServiceImportInitialSegments(ctx, args[1:])
	case "update-segment":
		return runKnowledgeServiceUpdateSegment(ctx, args[1:])
	case "delete-segment":
		return runKnowledgeServiceDeleteSegment(ctx, args[1:])
	case "create-segment":
		return runKnowledgeServiceCreateSegment(ctx, args[1:])
	case "update-segment-enabled":
		return runKnowledgeServiceUpdateSegmentEnabled(ctx, args[1:])
	case "reembed-segments":
		return runKnowledgeServiceReembedSegments(ctx, args[1:])
	case "set-current-segment-version":
		return runKnowledgeServiceSetCurrentSegmentVersion(ctx, args[1:])
	case "list-source-jobs":
		return runKnowledgeServiceListSourceJobs(ctx, args[1:])
	case "reconcile-source-jobs":
		return runKnowledgeServiceReconcileSourceJobs(ctx, args[1:])
	case "list-entries":
		return runKnowledgeServiceListEntries(ctx, args[1:])
	case "create-entry":
		return runKnowledgeServiceCreateEntry(ctx, args[1:])
	case "update-entry":
		return runKnowledgeServiceUpdateEntry(ctx, args[1:])
	case "delete-entry":
		return runKnowledgeServiceDeleteEntry(ctx, args[1:])
	case "import":
		return runKnowledgeServiceImport(ctx, args[1:])
	case "export":
		return runKnowledgeServiceExport(ctx, args[1:])
	case "validate":
		return runKnowledgeServiceValidate(ctx, args[1:])
	default:
		return fmt.Errorf("unknown semantic-model subcommand: %s", sub)
	}
}

// ShowSemanticModelHelp prints help for semantic-model.
func ShowSemanticModelHelp(_ []string) {
	fmt.Fprintf(os.Stderr, `Usage: semantic-model <subcommand> [options]

Product SDK surface: KnowledgeService

Subcommands:
  list                             KnowledgeService.List
  list-tags                        KnowledgeService.ListTags
  create                           KnowledgeService.Create
  get                              KnowledgeService.Get
  update                           KnowledgeService.Update
  delete                           KnowledgeService.Delete
  create-with-sources              KnowledgeService.CreateWithSources
  create-empty                     KnowledgeService.CreateEmpty
  upload-local-file                KnowledgeService.UploadLocalFile
  upload-model-local-file          KnowledgeService.UploadModelLocalFile
  preview-source-selection-counts  KnowledgeService.PreviewSourceSelectionCounts
  preview-model-source-selection-counts KnowledgeService.PreviewModelSourceSelectionCounts
  list-sources                     KnowledgeService.ListSources
  list-sources-page                KnowledgeService.ListSourcesPage
  check-source-existence           KnowledgeService.CheckSourceExistence
  append-sources                   KnowledgeService.AppendSources
  backfill-legacy-sources          KnowledgeService.BackfillLegacySources
  delete-source                    KnowledgeService.DeleteSource
  get-source-document              KnowledgeService.GetSourceDocument
  update-source-governance         KnowledgeService.UpdateSourceGovernance
  import-initial-segments          KnowledgeService.ImportInitialSegments
  update-segment                   KnowledgeService.UpdateSegment
  delete-segment                   KnowledgeService.DeleteSegment
  create-segment                   KnowledgeService.CreateSegment
  update-segment-enabled           KnowledgeService.UpdateSegmentEnabled
  reembed-segments                 KnowledgeService.ReembedSegments
  set-current-segment-version      KnowledgeService.SetCurrentSegmentVersion
  list-source-jobs                 KnowledgeService.ListSourceJobs
  reconcile-source-jobs            KnowledgeService.ReconcileSourceJobs
  list-entries                     KnowledgeService.ListEntries
  create-entry                     KnowledgeService.CreateEntry
  update-entry                     KnowledgeService.UpdateEntry
  delete-entry                     KnowledgeService.DeleteEntry
  import                           KnowledgeService.Import
  export                           KnowledgeService.Export
  validate                         KnowledgeService.Validate

Complex arguments accept --json / --json-file.
Stream methods accept --output <path>.

`)
}

func runKnowledgeServiceList(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model list", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model list [options]\n\nProduct SDK: KnowledgeService.List\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).List(context.Background())
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceListTags(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model list-tags", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model list-tags [options]\n\nProduct SDK: KnowledgeService.ListTags\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).ListTags(context.Background())
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceCreate(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model create", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	name := fs.String("name", "", "knowledge base name and immutable Catalog database name")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model create [options]\n\nProduct SDK: KnowledgeService.Create\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).Create(context.Background(), *name)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceGet(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model get", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model get [options]\n\nProduct SDK: KnowledgeService.Get\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).Get(context.Background(), *modelID)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceUpdate(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model update", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	name := fs.String("name", "", "existing immutable knowledge base and Catalog database name (rename unsupported)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model update [options]\n\nProduct SDK: KnowledgeService.Update\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).Update(context.Background(), *modelID, *name)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceDelete(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model delete", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model delete [options]\n\nProduct SDK: KnowledgeService.Delete\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).Delete(context.Background(), *modelID)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceCreateWithSources(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model create-with-sources", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	name := fs.String("name", "", "knowledge base name and immutable Catalog database name")
	imageIndexEnabled := fs.Bool("image-index-enabled", false, "enable image vector index (image_index_enabled=true)")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model create-with-sources [options]\n\nProduct SDK: KnowledgeService.CreateWithSources\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var sources []sdk.SemanticModelSourceInput
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &sources); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	var opts []sdk.SemanticModelWithSourcesOption
	if *imageIndexEnabled {
		opts = append(opts, sdk.WithSemanticModelWithSourcesImageIndexEnabled(true))
	}
	result, err := ctx.Client.Knowledge(wid).CreateWithSources(context.Background(), *name, sources, opts...)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceCreateEmpty(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model create-empty", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	name := fs.String("name", "", "knowledge base name and immutable Catalog database name")
	description := fs.String("description", "", "knowledge base description")
	imageIndexEnabled := fs.Bool("image-index-enabled", false, "enable image vector index (image_index_enabled=true)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model create-empty [options]\n\nProduct SDK: KnowledgeService.CreateEmpty\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	var opts []sdk.SemanticModelEmptyOption
	if *description != "" {
		opts = append(opts, sdk.WithSemanticModelEmptyDescription(*description))
	}
	if *imageIndexEnabled {
		opts = append(opts, sdk.WithSemanticModelEmptyImageIndexEnabled(true))
	}
	result, err := ctx.Client.Knowledge(wid).CreateEmpty(context.Background(), *name, opts...)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceUploadLocalFile(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model upload-local-file", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	filename := fs.String("filename", "", "filename")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model upload-local-file [options]\n\nProduct SDK: KnowledgeService.UploadLocalFile\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var reader io.Reader = bytes.NewReader(nil)
	if len(rawBody) > 0 { reader = bytes.NewReader(rawBody) }
	result, err := ctx.Client.Knowledge(wid).UploadLocalFile(context.Background(), *filename, reader)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceUploadModelLocalFile(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model upload-model-local-file", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	filename := fs.String("filename", "", "filename")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model upload-model-local-file [options]\n\nProduct SDK: KnowledgeService.UploadModelLocalFile\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var reader io.Reader = bytes.NewReader(nil)
	if len(rawBody) > 0 { reader = bytes.NewReader(rawBody) }
	result, err := ctx.Client.Knowledge(wid).UploadModelLocalFile(context.Background(), *modelID, *filename, reader)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServicePreviewSourceSelectionCounts(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model preview-source-selection-counts", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model preview-source-selection-counts [options]\n\nProduct SDK: KnowledgeService.PreviewSourceSelectionCounts\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var selections []sdk.SemanticModelSourceSelectionInput
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &selections); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	result, err := ctx.Client.Knowledge(wid).PreviewSourceSelectionCounts(context.Background(), selections)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServicePreviewModelSourceSelectionCounts(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model preview-model-source-selection-counts", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model preview-model-source-selection-counts [options]\n\nProduct SDK: KnowledgeService.PreviewModelSourceSelectionCounts\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var selections []sdk.SemanticModelSourceSelectionInput
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &selections); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	result, err := ctx.Client.Knowledge(wid).PreviewModelSourceSelectionCounts(context.Background(), *modelID, selections)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceListSources(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model list-sources", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model list-sources [options]\n\nProduct SDK: KnowledgeService.ListSources\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).ListSources(context.Background(), *modelID)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceListSourcesPage(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model list-sources-page", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	page := fs.Int("page", 0, "page")
	pageSize := fs.Int("page-size", 0, "pageSize")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model list-sources-page [options]\n\nProduct SDK: KnowledgeService.ListSourcesPage\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).ListSourcesPage(context.Background(), *modelID, *page, *pageSize)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceCheckSourceExistence(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model check-source-existence", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	fileIDs := fs.String("file-i-ds", "", "fileIDs (comma-separated)")
	tableIDs := fs.String("table-i-ds", "", "tableIDs (comma-separated)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model check-source-existence [options]\n\nProduct SDK: KnowledgeService.CheckSourceExistence\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).CheckSourceExistence(context.Background(), *modelID, splitCSV(*fileIDs), parseInt64CSV(*tableIDs))
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceAppendSources(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model append-sources", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model append-sources [options]\n\nProduct SDK: KnowledgeService.AppendSources\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var sources []sdk.SemanticModelSourceInput
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &sources); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	result, err := ctx.Client.Knowledge(wid).AppendSources(context.Background(), *modelID, sources)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceBackfillLegacySources(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model backfill-legacy-sources", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model backfill-legacy-sources [options]\n\nProduct SDK: KnowledgeService.BackfillLegacySources\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).BackfillLegacySources(context.Background(), *modelID)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceDeleteSource(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model delete-source", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	sourceID := fs.String("source-id", "", "sourceID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model delete-source [options]\n\nProduct SDK: KnowledgeService.DeleteSource\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).DeleteSource(context.Background(), *modelID, *sourceID)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceGetSourceDocument(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model get-source-document", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	sourceID := fs.String("source-id", "", "sourceID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model get-source-document [options]\n\nProduct SDK: KnowledgeService.GetSourceDocument\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).GetSourceDocument(context.Background(), *modelID, *sourceID)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceUpdateSourceGovernance(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model update-source-governance", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	sourceID := fs.String("source-id", "", "sourceID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model update-source-governance [options]\n\nProduct SDK: KnowledgeService.UpdateSourceGovernance\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).UpdateSourceGovernance(context.Background(), *modelID, *sourceID)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceImportInitialSegments(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model import-initial-segments", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	sourceID := fs.String("source-id", "", "sourceID")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model import-initial-segments [options]\n\nProduct SDK: KnowledgeService.ImportInitialSegments\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var base sdk.SemanticModelSegmentBase
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &base); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	result, err := ctx.Client.Knowledge(wid).ImportInitialSegments(context.Background(), *modelID, *sourceID, base)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceUpdateSegment(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model update-segment", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	sourceID := fs.String("source-id", "", "sourceID")
	segmentID := fs.String("segment-id", "", "segmentID")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model update-segment [options]\n\nProduct SDK: KnowledgeService.UpdateSegment\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var input sdk.SemanticModelSegmentUpdateInput
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &input); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	result, err := ctx.Client.Knowledge(wid).UpdateSegment(context.Background(), *modelID, *sourceID, *segmentID, input)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceDeleteSegment(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model delete-segment", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	sourceID := fs.String("source-id", "", "sourceID")
	segmentID := fs.String("segment-id", "", "segmentID")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model delete-segment [options]\n\nProduct SDK: KnowledgeService.DeleteSegment\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var base sdk.SemanticModelSegmentBase
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &base); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	result, err := ctx.Client.Knowledge(wid).DeleteSegment(context.Background(), *modelID, *sourceID, *segmentID, base)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceCreateSegment(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model create-segment", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	sourceID := fs.String("source-id", "", "sourceID")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model create-segment [options]\n\nProduct SDK: KnowledgeService.CreateSegment\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var input sdk.SemanticModelSegmentCreateInput
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &input); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	result, err := ctx.Client.Knowledge(wid).CreateSegment(context.Background(), *modelID, *sourceID, input)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceUpdateSegmentEnabled(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model update-segment-enabled", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	sourceID := fs.String("source-id", "", "sourceID")
	segmentID := fs.String("segment-id", "", "segmentID")
	enabled := fs.Bool("enabled", false, "enabled")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model update-segment-enabled [options]\n\nProduct SDK: KnowledgeService.UpdateSegmentEnabled\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var base sdk.SemanticModelSegmentBase
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &base); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	result, err := ctx.Client.Knowledge(wid).UpdateSegmentEnabled(context.Background(), *modelID, *sourceID, *segmentID, base, *enabled)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceReembedSegments(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model reembed-segments", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	sourceID := fs.String("source-id", "", "sourceID")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model reembed-segments [options]\n\nProduct SDK: KnowledgeService.ReembedSegments\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var base sdk.SemanticModelSegmentBase
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &base); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	result, err := ctx.Client.Knowledge(wid).ReembedSegments(context.Background(), *modelID, *sourceID, base)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceSetCurrentSegmentVersion(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model set-current-segment-version", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	sourceID := fs.String("source-id", "", "sourceID")
	versionID := fs.String("version-id", "", "versionID")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model set-current-segment-version [options]\n\nProduct SDK: KnowledgeService.SetCurrentSegmentVersion\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var base sdk.SemanticModelSegmentBase
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &base); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	result, err := ctx.Client.Knowledge(wid).SetCurrentSegmentVersion(context.Background(), *modelID, *sourceID, *versionID, base)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceListSourceJobs(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model list-source-jobs", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model list-source-jobs [options]\n\nProduct SDK: KnowledgeService.ListSourceJobs\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).ListSourceJobs(context.Background(), *modelID)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceReconcileSourceJobs(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model reconcile-source-jobs", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model reconcile-source-jobs [options]\n\nProduct SDK: KnowledgeService.ReconcileSourceJobs\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).ReconcileSourceJobs(context.Background(), *modelID)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceListEntries(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model list-entries", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model list-entries [options]\n\nProduct SDK: KnowledgeService.ListEntries\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).ListEntries(context.Background(), *modelID)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceCreateEntry(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model create-entry", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model create-entry [options]\n\nProduct SDK: KnowledgeService.CreateEntry\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var entry sdk.SemanticEntryInput
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &entry); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	result, err := ctx.Client.Knowledge(wid).CreateEntry(context.Background(), *modelID, entry)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceUpdateEntry(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model update-entry", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	entryID := fs.String("entry-id", "", "entryID")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model update-entry [options]\n\nProduct SDK: KnowledgeService.UpdateEntry\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var entry sdk.SemanticEntryInput
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &entry); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	result, err := ctx.Client.Knowledge(wid).UpdateEntry(context.Background(), *modelID, *entryID, entry)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceDeleteEntry(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model delete-entry", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	entryID := fs.String("entry-id", "", "entryID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model delete-entry [options]\n\nProduct SDK: KnowledgeService.DeleteEntry\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).DeleteEntry(context.Background(), *modelID, *entryID)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceImport(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model import", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	bodyJSON := fs.String("json", "", "JSON for complex arguments")
	bodyFile := fs.String("json-file", "", "JSON file for complex arguments")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model import [options]\n\nProduct SDK: KnowledgeService.Import\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	rawBody, err := loadJSONBytes(*bodyJSON, *bodyFile)
	if err != nil { return err }
	var entries []sdk.SemanticEntryInput
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &entries); err != nil { return fmt.Errorf("parse --json: %w", err) }
	}
	result, err := ctx.Client.Knowledge(wid).Import(context.Background(), *modelID, entries)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceExport(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model export", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model export [options]\n\nProduct SDK: KnowledgeService.Export\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).Export(context.Background(), *modelID)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}

func runKnowledgeServiceValidate(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model validate", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.String("model-id", "", "modelID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: moi-cli semantic-model validate [options]\n\nProduct SDK: KnowledgeService.Validate\n\nOptions:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil { return err }
	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil { return err }
	result, err := ctx.Client.Knowledge(wid).Validate(context.Background(), *modelID)
	if err != nil { return err }
	return printJSONOrTable(ctx, result, func() error { return printJSON(result) })
}
