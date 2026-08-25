package commands

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	moi "github.com/matrixflow/moi-core/go-sdk"
)

// ExecuteSemanticModelCommand runs the semantic-model command.
func ExecuteSemanticModelCommand(ctx *Context, args []string) error {
	if len(args) == 0 {
		ShowSemanticModelHelp(nil)
		return nil
	}

	switch args[0] {
	case "create":
		return runSemanticModelCreate(ctx, args[1:])
	case "list":
		return runSemanticModelList(ctx, args[1:])
	case "list-tags":
		return runSemanticModelListTags(ctx, args[1:])
	case "get":
		return runSemanticModelGet(ctx, args[1:])
	case "update":
		return runSemanticModelUpdate(ctx, args[1:])
	case "delete":
		return runSemanticModelDelete(ctx, args[1:])
	case "import":
		return runSemanticModelImport(ctx, args[1:])
	case "export":
		return runSemanticModelExport(ctx, args[1:])
	case "validate":
		return runSemanticModelValidate(ctx, args[1:])
	case "create-entry":
		return runSemanticEntryCreate(ctx, args[1:])
	case "list-entries":
		return runSemanticEntryList(ctx, args[1:])
	case "update-entry":
		return runSemanticEntryUpdate(ctx, args[1:])
	case "delete-entry":
		return runSemanticEntryDelete(ctx, args[1:])
	default:
		return fmt.Errorf("unknown semantic-model subcommand: %s", args[0])
	}
}

func semanticModelClient(ctx *Context, workspaceID string) *moi.SemanticModelService {
	return ctx.Client.SemanticModels(workspaceID)
}

func runSemanticModelCreate(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model create", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	name := fs.String("name", "", "Semantic model name")
	description := fs.String("description", "", "Semantic model description")
	tables := fs.String("tables", "", "Comma-separated table names")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: core-cli semantic-model create --name <name> --tables <t1,t2> [--description <desc>] [--workspace-id <id>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("--name is required")
	}
	tableList := splitCSV(*tables)
	if len(tableList) == 0 {
		return fmt.Errorf("--tables is required")
	}

	tablesJSON, err := json.Marshal(tableList)
	if err != nil {
		return fmt.Errorf("marshal tables: %w", err)
	}

	model, err := semanticModelClient(ctx, wid).Create(context.Background(), &moi.SemanticModelUpsertRequest{
		Name:        strings.TrimSpace(*name),
		Description: strings.TrimSpace(*description),
		Tables:      tablesJSON,
	})
	if err != nil {
		return err
	}
	return printSemanticModel(ctx, model)
}

func runSemanticModelList(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model list", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: core-cli semantic-model list [--workspace-id <id>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil {
		return err
	}

	pageToken := ""
	first := true
	for {
		opts := []moi.ListOption{moi.WithPageSize(int32(defaultPageSize))}
		if pageToken != "" {
			opts = append(opts, moi.WithPageToken(pageToken))
		}
		resp, err := semanticModelClient(ctx, wid).List(context.Background(), opts...)
		if err != nil {
			return err
		}

		if first {
			first = false
		} else if len(resp.Items) > 0 {
			fmt.Println()
		}
		if len(resp.Items) == 0 && pageToken == "" {
			fmt.Println("No semantic models found.")
			return nil
		}
		if err := printSemanticModelList(ctx, resp.Items); err != nil {
			return err
		}
		if resp.NextPageToken == "" {
			return nil
		}
		if !ctx.AskNextPage() {
			fmt.Fprintf(os.Stderr, "还有更多结果，未继续获取。\n")
			return nil
		}
		pageToken = resp.NextPageToken
	}
}

func runSemanticModelListTags(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model list-tags", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	pageSize := fs.Int("page-size", defaultPageSize, "Page size")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: core-cli semantic-model list-tags [--page-size <n>] [--workspace-id <id>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil {
		return err
	}
	if *pageSize <= 0 {
		return fmt.Errorf("--page-size must be greater than 0")
	}

	resp, err := semanticModelClient(ctx, wid).ListTags(context.Background(), moi.WithPageSize(int32(*pageSize)))
	if err != nil {
		return err
	}
	if len(resp.Items) == 0 {
		fmt.Println("No semantic model tags found.")
		return nil
	}
	return printSemanticModelTagList(ctx, resp.Items)
}

func runSemanticModelGet(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model get", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.Int64("model-id", 0, "Semantic model ID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: core-cli semantic-model get --model-id <id> [--workspace-id <id>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil {
		return err
	}
	if *modelID <= 0 {
		return fmt.Errorf("--model-id is required")
	}

	model, err := semanticModelClient(ctx, wid).Get(context.Background(), *modelID)
	if err != nil {
		return err
	}
	return printSemanticModel(ctx, model)
}

func runSemanticModelUpdate(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model update", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.Int64("model-id", 0, "Semantic model ID")
	name := fs.String("name", "", "Semantic model name")
	description := fs.String("description", "", "Semantic model description")
	tables := fs.String("tables", "", "Comma-separated table names")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: core-cli semantic-model update --model-id <id> --name <name> --tables <t1,t2> [--description <desc>] [--workspace-id <id>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil {
		return err
	}
	if *modelID <= 0 {
		return fmt.Errorf("--model-id is required")
	}
	if strings.TrimSpace(*name) == "" {
		return fmt.Errorf("--name is required")
	}
	tableList := splitCSV(*tables)
	if len(tableList) == 0 {
		return fmt.Errorf("--tables is required")
	}

	tablesJSON, err := json.Marshal(tableList)
	if err != nil {
		return fmt.Errorf("marshal tables: %w", err)
	}

	resp, err := semanticModelClient(ctx, wid).Update(context.Background(), *modelID, &moi.SemanticModelUpsertRequest{
		Name:        strings.TrimSpace(*name),
		Description: strings.TrimSpace(*description),
		Tables:      tablesJSON,
	})
	if err != nil {
		return err
	}

	if ctx.OutputJSON {
		return json.NewEncoder(os.Stdout).Encode(resp)
	}
	fmt.Printf("Updated semantic model %d\n", *modelID)
	return nil
}

func runSemanticModelDelete(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model delete", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.Int64("model-id", 0, "Semantic model ID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: core-cli semantic-model delete --model-id <id> [--workspace-id <id>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil {
		return err
	}
	if *modelID <= 0 {
		return fmt.Errorf("--model-id is required")
	}
	if err := semanticModelClient(ctx, wid).Delete(context.Background(), *modelID); err != nil {
		return err
	}
	fmt.Printf("Deleted semantic model %d\n", *modelID)
	return nil
}

func runSemanticModelImport(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model import", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	filePath := fs.String("file", "", "Import JSON file path")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: core-cli semantic-model import --file <import.json> [--workspace-id <id>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(*filePath) == "" {
		return fmt.Errorf("--file is required")
	}

	raw, err := os.ReadFile(*filePath)
	if err != nil {
		return fmt.Errorf("read import file: %w", err)
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return fmt.Errorf("import file is empty")
	}

	if strings.HasPrefix(trimmed, "[") {
		var reqs []moi.SemanticModelImportRequest
		if err := json.Unmarshal(raw, &reqs); err != nil {
			return fmt.Errorf("invalid import file json array: %w", err)
		}
		resp, err := semanticModelClient(ctx, wid).ImportBatch(context.Background(), reqs)
		if err != nil {
			return err
		}
		if ctx.OutputJSON {
			return json.NewEncoder(os.Stdout).Encode(resp)
		}
		fmt.Printf("Imported %d semantic models.\n", resp.Total)
		return nil
	}

	var req moi.SemanticModelImportRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return fmt.Errorf("invalid import file json object: %w", err)
	}
	model, err := semanticModelClient(ctx, wid).Import(context.Background(), &req)
	if err != nil {
		return err
	}
	return printSemanticModel(ctx, model)
}

func runSemanticModelExport(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model export", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.Int64("model-id", 0, "Semantic model ID")
	output := fs.String("output", "", "Output JSON file path (optional)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: core-cli semantic-model export --model-id <id> [--output <file>] [--workspace-id <id>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil {
		return err
	}
	if *modelID <= 0 {
		return fmt.Errorf("--model-id is required")
	}
	if ctx.PreflightOnly {
		return nil
	}

	resp, err := semanticModelClient(ctx, wid).Export(context.Background(), *modelID)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return err
	}
	if strings.TrimSpace(*output) != "" {
		if err := os.WriteFile(*output, append(data, '\n'), 0o644); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}
		if !ctx.OutputJSON {
			fmt.Printf("Exported semantic model %d to %s\n", *modelID, *output)
		}
		return nil
	}
	_, err = os.Stdout.Write(append(data, '\n'))
	return err
}

func runSemanticModelValidate(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model validate", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.Int64("model-id", 0, "Semantic model ID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: core-cli semantic-model validate --model-id <id> [--workspace-id <id>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil {
		return err
	}
	if *modelID <= 0 {
		return fmt.Errorf("--model-id is required")
	}

	resp, err := semanticModelClient(ctx, wid).Validate(context.Background(), *modelID)
	if err != nil {
		return err
	}
	if ctx.OutputJSON {
		return json.NewEncoder(os.Stdout).Encode(resp)
	}
	fmt.Printf("Semantic model %d valid: %t\n", *modelID, resp.Valid)
	return nil
}

func runSemanticEntryCreate(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model create-entry", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.Int64("model-id", 0, "Semantic model ID")
	kind := fs.String("kind", "", "Entry kind")
	key := fs.String("key", "", "Entry key")
	tables := fs.String("tables", "", "Comma-separated table names (optional)")
	specJSON := fs.String("spec-json", "", "Spec JSON content")
	specFile := fs.String("spec-file", "", "Spec JSON file path")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: core-cli semantic-model create-entry --model-id <id> --kind <kind> --key <key> (--spec-json <json> | --spec-file <file>) [--tables <t1,t2>] [--workspace-id <id>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil {
		return err
	}
	if *modelID <= 0 {
		return fmt.Errorf("--model-id is required")
	}
	spec, err := readSpecPayload(*specJSON, *specFile)
	if err != nil {
		return err
	}
	entry, err := semanticModelClient(ctx, wid).CreateEntry(context.Background(), *modelID, &moi.SemanticEntryUpsertRequest{
		Kind:   strings.TrimSpace(*kind),
		Key:    strings.TrimSpace(*key),
		Tables: splitCSV(*tables),
		Spec:   spec,
	})
	if err != nil {
		return err
	}
	return printSemanticEntry(ctx, entry)
}

func runSemanticEntryList(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model list-entries", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.Int64("model-id", 0, "Semantic model ID")
	kind := fs.String("kind", "", "Entry kind filter")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: core-cli semantic-model list-entries --model-id <id> [--kind <kind>] [--workspace-id <id>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil {
		return err
	}
	if *modelID <= 0 {
		return fmt.Errorf("--model-id is required")
	}

	pageToken := ""
	first := true
	for {
		opts := []moi.ListOption{moi.WithPageSize(int32(defaultPageSize))}
		if pageToken != "" {
			opts = append(opts, moi.WithPageToken(pageToken))
		}
		resp, err := semanticModelClient(ctx, wid).ListEntries(context.Background(), *modelID, strings.TrimSpace(*kind), opts...)
		if err != nil {
			return err
		}
		if first {
			first = false
		} else if len(resp.Items) > 0 {
			fmt.Println()
		}
		if len(resp.Items) == 0 && pageToken == "" {
			fmt.Println("No semantic entries found.")
			return nil
		}
		if err := printSemanticEntryList(ctx, resp.Items); err != nil {
			return err
		}
		if resp.NextPageToken == "" {
			return nil
		}
		if !ctx.AskNextPage() {
			fmt.Fprintf(os.Stderr, "还有更多结果，未继续获取。\n")
			return nil
		}
		pageToken = resp.NextPageToken
	}
}

func runSemanticEntryUpdate(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model update-entry", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.Int64("model-id", 0, "Semantic model ID")
	entryID := fs.Int64("entry-id", 0, "Semantic entry ID")
	kind := fs.String("kind", "", "Entry kind")
	key := fs.String("key", "", "Entry key")
	tables := fs.String("tables", "", "Comma-separated table names (optional)")
	specJSON := fs.String("spec-json", "", "Spec JSON content")
	specFile := fs.String("spec-file", "", "Spec JSON file path")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: core-cli semantic-model update-entry --model-id <id> --entry-id <id> --kind <kind> --key <key> (--spec-json <json> | --spec-file <file>) [--tables <t1,t2>] [--workspace-id <id>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil {
		return err
	}
	if *modelID <= 0 {
		return fmt.Errorf("--model-id is required")
	}
	if *entryID <= 0 {
		return fmt.Errorf("--entry-id is required")
	}
	spec, err := readSpecPayload(*specJSON, *specFile)
	if err != nil {
		return err
	}
	resp, err := semanticModelClient(ctx, wid).UpdateEntry(context.Background(), *modelID, *entryID, &moi.SemanticEntryUpsertRequest{
		Kind:   strings.TrimSpace(*kind),
		Key:    strings.TrimSpace(*key),
		Tables: splitCSV(*tables),
		Spec:   spec,
	})
	if err != nil {
		return err
	}

	if ctx.OutputJSON {
		return json.NewEncoder(os.Stdout).Encode(resp)
	}
	fmt.Printf("Updated semantic entry %d in model %d\n", *entryID, *modelID)
	return nil
}

func runSemanticEntryDelete(ctx *Context, args []string) error {
	fs := flag.NewFlagSet("semantic-model delete-entry", flag.ExitOnError)
	workspaceID := fs.String("workspace-id", "", "Workspace ID (default: from config)")
	modelID := fs.Int64("model-id", 0, "Semantic model ID")
	entryID := fs.Int64("entry-id", 0, "Semantic entry ID")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: core-cli semantic-model delete-entry --model-id <id> --entry-id <id> [--workspace-id <id>]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	wid, err := resolveWorkspaceID(ctx, *workspaceID)
	if err != nil {
		return err
	}
	if *modelID <= 0 {
		return fmt.Errorf("--model-id is required")
	}
	if *entryID <= 0 {
		return fmt.Errorf("--entry-id is required")
	}

	if err := semanticModelClient(ctx, wid).DeleteEntry(context.Background(), *modelID, *entryID); err != nil {
		return err
	}
	fmt.Printf("Deleted semantic entry %d from model %d\n", *entryID, *modelID)
	return nil
}

func readSpecPayload(specJSON, specFile string) (json.RawMessage, error) {
	hasJSON := strings.TrimSpace(specJSON) != ""
	hasFile := strings.TrimSpace(specFile) != ""
	if hasJSON == hasFile {
		return nil, fmt.Errorf("exactly one of --spec-json or --spec-file is required")
	}

	var raw []byte
	if hasJSON {
		raw = []byte(specJSON)
	} else {
		data, err := os.ReadFile(specFile)
		if err != nil {
			return nil, fmt.Errorf("read spec file: %w", err)
		}
		raw = data
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return nil, fmt.Errorf("spec must not be empty")
	}
	var v any
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil {
		return nil, fmt.Errorf("invalid spec json: %w", err)
	}
	return json.RawMessage(trimmed), nil
}

func printSemanticModel(ctx *Context, model *moi.SemanticModel) error {
	if ctx.OutputJSON {
		return json.NewEncoder(os.Stdout).Encode(model)
	}
	fmt.Printf("ID: %d\n", model.ID)
	fmt.Printf("Name: %s\n", model.Name)
	if model.Description != "" {
		fmt.Printf("Description: %s\n", model.Description)
	}
	fmt.Printf("Tables: %s\n", string(model.Tables))
	return nil
}

func printSemanticModelList(ctx *Context, items []*moi.SemanticModel) error {
	if ctx.OutputJSON {
		return json.NewEncoder(os.Stdout).Encode(items)
	}
	for _, item := range items {
		fmt.Printf("%d\t%s\t%s\n", item.ID, item.Name, string(item.Tables))
	}
	return nil
}

func printSemanticEntry(ctx *Context, entry *moi.SemanticEntry) error {
	if ctx.OutputJSON {
		return json.NewEncoder(os.Stdout).Encode(entry)
	}
	fmt.Printf("ID: %d\n", entry.ID)
	fmt.Printf("ModelID: %d\n", entry.ModelID)
	fmt.Printf("Kind: %s\n", entry.Kind)
	fmt.Printf("Key: %s\n", entry.Key)
	if len(entry.Tables) > 0 {
		fmt.Printf("Tables: %s\n", strings.Join(entry.Tables, ", "))
	}
	fmt.Printf("Spec: %s\n", string(entry.Spec))
	return nil
}

func printSemanticEntryList(ctx *Context, items []*moi.SemanticEntry) error {
	if ctx.OutputJSON {
		return json.NewEncoder(os.Stdout).Encode(items)
	}
	for _, item := range items {
		fmt.Printf("%d\t%s\t%s\n", item.ID, item.Kind, item.Key)
	}
	return nil
}

func printSemanticModelTagList(ctx *Context, items []moi.SemanticModelTagStat) error {
	if ctx.OutputJSON {
		return json.NewEncoder(os.Stdout).Encode(items)
	}
	for _, item := range items {
		fmt.Printf("%s\t%d\n", item.Tag, item.Count)
	}
	return nil
}

// ShowSemanticModelHelp prints help for the semantic-model command.
func ShowSemanticModelHelp(_ []string) {
	fmt.Fprintf(os.Stderr, `Usage: core-cli semantic-model <subcommand> [options]

Subcommands:
  create         Create semantic model (--name, --tables required)
  list           List semantic models
  list-tags      List aggregated semantic model tags
  get            Get semantic model by ID (--model-id required)
  update         Update semantic model (--model-id, --name, --tables required)
  delete         Delete semantic model (--model-id required)
  import         Import semantic model(s) from JSON file (--file required)
  export         Export semantic model and entries (--model-id required)
  validate       Validate semantic model (--model-id required)
  create-entry   Create semantic entry (--model-id, --kind, --key, spec required)
  list-entries   List semantic entries (--model-id required)
  update-entry   Update semantic entry (--model-id, --entry-id, --kind, --key, spec required)
  delete-entry   Delete semantic entry (--model-id, --entry-id required)

Examples:
  core-cli semantic-model create --name sales_model --tables orders,users --description "sales semantic"
  core-cli semantic-model list
  core-cli semantic-model list-tags
  core-cli semantic-model get --model-id 1
  core-cli semantic-model create-entry --model-id 1 --kind metric --key gmv \
    --spec-json '{"expr":"sum(order_amount)"}'
  core-cli semantic-model import --file ./import.json
  core-cli semantic-model export --model-id 1 --output ./semantic_model_1.json
  core-cli semantic-model validate --model-id 1

For subcommand-specific help: core-cli semantic-model <subcommand> --help

`)
}
