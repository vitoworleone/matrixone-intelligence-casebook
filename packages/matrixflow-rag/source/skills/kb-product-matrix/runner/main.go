// Local-only Product SDK matrix for Knowledge Base.
// Not for git under .runtime/; skill copy is the durable entry.
//
// Cells (must all pass for MATRIX_PASSED):
//
//	M1 lifecycle: Create / Get / List / Update / Delete empty KB
//	M2 catalog_table: user table → CreateWithSources → list source
//	M3 structured_upload: connector LocalUpload CSV → structured CreateWithSources → settle
//	M4 local_file_doc: knowledge UploadLocalFile PDF → CreateWithSources → ingest succeeded
//	M5 catalog_file_lineage: volume PDF catalog_file → standard_rag doc prep + ingest + lineage + workflow completed
//	M6 a2a_rag: explore StreamText on M5 model → answer contains 80% + source_ref
//	M7 append_source: AppendSources catalog_file onto existing model
//	M8 entries: CreateEntry / ListEntries / DeleteEntry
//	M9 delete_source: DeleteSource on append source
//	M10 image_index_doc_prep: mixed PDF + ImageIndexEnabled → standard_rag_with_image_index settle + workflow completed
//	M11 a2a_rag_image_index: explore StreamText on M10 model → answer contains 80% + source_ref
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	sdk "github.com/matrixorigin/matrixflow/sdk/go-sdk"
)

const (
	question           = "文档说 MatrixOne 通过统一架构将数据组件和技术栈减少了多少？"
	answerNeed         = "80%"
	defaultCatalogName = "Default"
)

type checkResult struct {
	Cell   string `json:"cell"`
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail"`
}

type matrixReport struct {
	OK          bool           `json:"ok"`
	Status      string         `json:"status"`
	Suffix      string         `json:"suffix"`
	WorkspaceID string         `json:"workspace_id"`
	Endpoint    string         `json:"endpoint"`
	Frontend    string         `json:"frontend"`
	GeneratedAt string         `json:"generated_at"`
	Checks      []checkResult  `json:"checks"`
	Artifacts   map[string]any `json:"artifacts"`
	Note        string         `json:"note"`
}

func envOr(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "MATRIX FAILED: "+format+"\n", args...)
	os.Exit(1)
}

func must[T any](v T, err error) T {
	if err != nil {
		fail("%v", err)
	}
	return v
}

func findRoot() string {
	if v := envOr("WORKTREE_ROOT", ""); v != "" {
		return v
	}
	if v := envOr("MATRIXFLOW_ROOT", ""); v != "" {
		return v
	}
	// Prefer executable location: <root>/.runtime/kb-product-matrix-runner or skills/.../runner
	if ex, err := os.Executable(); err == nil {
		dir := filepath.Dir(ex)
		for i := 0; i < 10; i++ {
			if _, err := os.Stat(filepath.Join(dir, "sdk/go-sdk")); err == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		fail("getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "sdk/go-sdk")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return wd
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func resolveDefaultCatalogID(result *sdk.CatalogListResult) (int64, error) {
	if result == nil {
		return 0, fmt.Errorf("catalog list is empty")
	}
	var defaultID int64
	for _, item := range result.GetList() {
		if item == nil || item.GetName() != defaultCatalogName {
			continue
		}
		if item.GetId() <= 0 {
			return 0, fmt.Errorf("default catalog has invalid id %d", item.GetId())
		}
		if defaultID != 0 {
			return 0, fmt.Errorf("multiple %q catalogs: %d and %d", defaultCatalogName, defaultID, item.GetId())
		}
		defaultID = item.GetId()
	}
	if defaultID == 0 {
		return 0, fmt.Errorf("default catalog %q is not available", defaultCatalogName)
	}
	return defaultID, nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func main() {
	root := findRoot()
	suffix := envOr("ACCEPTANCE_SUFFIX", time.Now().Format("20060102-150405"))
	outDir := envOr("MATRIX_OUT", filepath.Join(root, "output/kb-product-matrix"))
	_ = os.MkdirAll(outDir, 0o755)

	aistudioPort := envOr("AISTUDIO_PORT", "19050")
	ucPort := envOr("UC_PORT", "19080")
	frontendURL := envOr("AISTUDIO_PUBLIC_URL", "http://localhost:18000")
	backendURL := "http://127.0.0.1:" + aistudioPort
	productEndpoint := backendURL + "/newmoi"
	ucBase := "http://127.0.0.1:" + ucPort
	email := envOr("SEED_EMAIL", "local-admin+kb_catalog_lineage_acceptance@matrixflow.local")
	password := envOr("SEED_PASSWORD", "Admin@1234")
	patProductBase := strings.TrimRight(frontendURL, "/") + "/newmoi"

	pdfPath := envOr("SAMPLE_PDF", filepath.Join(root, "optools/matrixflow/moi-connector/sample_files/MatrixOne_Introduction.pdf"))
	pdfImagePath := envOr("SAMPLE_PDF_IMAGE", filepath.Join(root, "optools/matrixflow/moi-connector/sample_files/MatrixOne 简介（图文混合）.pdf"))
	csvPath := envOr("SAMPLE_CSV", filepath.Join(root, "optools/matrixflow/moi-connector/agent_home_guide_sample_files/team_member_map.csv"))
	if !fileExists(pdfPath) {
		fail("missing SAMPLE_PDF: %s", pdfPath)
	}
	if !fileExists(pdfImagePath) {
		fail("missing SAMPLE_PDF_IMAGE: %s", pdfImagePath)
	}
	if !fileExists(csvPath) {
		fail("missing SAMPLE_CSV: %s", csvPath)
	}

	fmt.Fprintf(os.Stderr, "[matrix] root=%s endpoint=%s frontend=%s suffix=%s\n", root, productEndpoint, frontendURL, suffix)

	pat, revoke, err := issuePAT(root, ucBase, patProductBase, email, password)
	if err != nil {
		fail("issue PAT: %v", err)
	}
	defer func() {
		if rerr := revoke(); rerr != nil {
			fmt.Fprintf(os.Stderr, "[matrix] PAT revoke warn: %v\n", rerr)
		} else {
			fmt.Fprintf(os.Stderr, "[matrix] PAT revoked\n")
		}
	}()

	// Catalog names are localized. Pin the public Product SDK read surface to
	// en-US so the system-initialized default catalog has its stable display name.
	client := must(sdk.NewWithPersonalAccessToken(productEndpoint, pat,
		sdk.WithCustomHeader("Accept-Language", "en-US")))
	defer client.Close()
	ctx := context.Background()

	wsName := "kb-matrix-ws-" + suffix
	ws := must(client.EnsureWorkspace(ctx, wsName))
	fmt.Fprintf(os.Stderr, "[matrix] workspace id=%s name=%s\n", ws.ID(), wsName)

	knowledge := ws.Knowledge()
	report := matrixReport{
		OK:          true,
		Status:      "MATRIX_PASSED",
		Suffix:      suffix,
		WorkspaceID: ws.ID(),
		Endpoint:    productEndpoint,
		Frontend:    frontendURL,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Checks:      []checkResult{},
		Artifacts:   map[string]any{},
		Note:        "local Product SDK matrix; do not commit secrets or PAT",
	}
	check := func(cell, name string, pass bool, detail string) {
		report.Checks = append(report.Checks, checkResult{Cell: cell, Name: name, Pass: pass, Detail: detail})
		if pass {
			fmt.Fprintf(os.Stderr, "[matrix] PASS %s/%s — %s\n", cell, name, detail)
		} else {
			fmt.Fprintf(os.Stderr, "[matrix] FAIL %s/%s — %s\n", cell, name, detail)
			report.OK = false
			report.Status = "MATRIX_FAILED"
		}
	}
	// hard fail stops matrix early for broken auth/setup; soft check continues for product cells
	hard := func(cell, name string, err error, detail string) {
		if err == nil {
			check(cell, name, true, detail)
			return
		}
		check(cell, name, false, fmt.Sprintf("%s err=%v", detail, err))
		writeReport(outDir, report)
		fail("%s/%s: %v", cell, name, err)
	}

	// ---------- M1 lifecycle ----------
	emptyName := "kb-matrix-empty-" + suffix
	createdEmpty, err := knowledge.Create(ctx, emptyName)
	hard("M1", "create_empty", err, fmt.Sprintf("name=%s", emptyName))
	emptyID := ""
	if createdEmpty != nil {
		emptyID = strconv.FormatInt(createdEmpty.GetId(), 10)
		report.Artifacts["m1_model_id"] = emptyID
	}
	got, err := knowledge.Get(ctx, emptyID)
	hard("M1", "get", err, fmt.Sprintf("id=%s", emptyID))
	if got != nil && got.GetName() != emptyName {
		check("M1", "get_name", false, fmt.Sprintf("got=%q want=%q", got.GetName(), emptyName))
	} else {
		check("M1", "get_name", true, emptyName)
	}
	listed, err := knowledge.List(ctx)
	hard("M1", "list", err, "ok")
	foundList := false
	if listed != nil {
		for _, m := range listed.GetItems() {
			if m.GetId() == createdEmpty.GetId() {
				foundList = true
				break
			}
		}
	}
	check("M1", "list_contains", foundList, emptyID)
	// Empty creation must establish the Catalog database before returning success.
	catalogs, catalogList, err := ws.Catalogs(ctx)
	hard("M1", "list_catalogs", err, "ok")
	defaultCatalogID, defaultCatalogErr := resolveDefaultCatalogID(catalogList)
	hard("M1", "default_catalog", defaultCatalogErr, fmt.Sprintf("catalog_id=%d", defaultCatalogID))
	report.Artifacts["default_catalog_id"] = defaultCatalogID
	emptyDatabaseID := 0
	emptyDatabaseCatalogID := int64(0)
	emptyDatabasePath := ""
	for _, catalog := range catalogs {
		_, databases, listErr := catalog.Databases(ctx)
		if listErr != nil {
			hard("M1", "list_catalog_databases", listErr, catalog.Name())
		}
		for _, database := range databases.GetList() {
			if database.GetName() != emptyName {
				continue
			}
			emptyDatabaseID = int(database.GetId())
			emptyDatabaseCatalogID = int64(catalog.ID())
			emptyDatabasePath = catalog.Name() + "/" + database.GetName()
		}
	}
	check("M1", "empty_database_created", emptyDatabaseID > 0 && emptyDatabaseCatalogID == defaultCatalogID,
		fmt.Sprintf("database_id=%d catalog_id=%d default_catalog_id=%d path=%s", emptyDatabaseID, emptyDatabaseCatalogID, defaultCatalogID, emptyDatabasePath))
	report.Artifacts["m1_database_id"] = emptyDatabaseID
	report.Artifacts["m1_database_path"] = emptyDatabasePath
	// Name is immutable after create (aligned with catalog database identifier).
	// Update with the same name must succeed; a rename must be rejected as 400/ErrParamInvalid.
	_, err = knowledge.Update(ctx, emptyID, emptyName)
	hard("M1", "update_same_name", err, emptyName)
	renamed := emptyName + "-renamed"
	_, err = knowledge.Update(ctx, emptyID, renamed)
	if err == nil {
		check("M1", "update_rename_rejected", false, "rename unexpectedly succeeded")
	} else if !sdk.IsCode(err, "ErrParamInvalid") {
		check("M1", "update_rename_rejected", false, fmt.Sprintf("want ErrParamInvalid, got %v", err))
	} else {
		var apiErr *sdk.Error
		if errors.As(err, &apiErr) && apiErr.Status != 0 && apiErr.Status != http.StatusBadRequest {
			check("M1", "update_rename_rejected", false,
				fmt.Sprintf("want HTTP 400, got status=%d err=%v", apiErr.Status, err))
		} else {
			check("M1", "update_rename_rejected", true, err.Error())
		}
	}
	gotAfterUpdate, err := knowledge.Get(ctx, emptyID)
	hard("M1", "get_after_update", err, emptyID)
	if gotAfterUpdate != nil && gotAfterUpdate.GetName() != emptyName {
		check("M1", "name_unchanged_after_rename_attempt", false,
			fmt.Sprintf("got=%q want=%q", gotAfterUpdate.GetName(), emptyName))
	} else {
		check("M1", "name_unchanged_after_rename_attempt", true, emptyName)
	}
	_, err = knowledge.Delete(ctx, emptyID)
	hard("M1", "delete", err, emptyID)

	// ---------- shared catalog for M2/M5/M10 ----------
	cat := must(ws.PrepareCatalog(ctx, "kb_matrix_cat_"+suffix))
	db := must(cat.PrepareDatabase(ctx, "kb_matrix_db_"+suffix))
	vol := must(db.PrepareVolume(ctx, "kb_matrix_vol_"+suffix))
	report.Artifacts["catalog_id"] = cat.ID()
	report.Artifacts["database_id"] = db.ID()
	report.Artifacts["volume_id"] = vol.ID()

	// Built-in document prep templates must exist (standard_rag + image-index variant).
	tplList, tplErr := ws.Workflows().WorkflowTemplates(ctx)
	hard("M5", "list_workflow_templates", tplErr, "")
	hasStandardRAG := false
	hasImageIndexRAG := false
	if tplList != nil {
		for _, t := range tplList.GetTemplates() {
			switch t.GetTemplateKey() {
			case "standard_rag":
				hasStandardRAG = true
			case "standard_rag_with_image_index":
				hasImageIndexRAG = true
			}
		}
	}
	check("M5", "template_standard_rag_present", hasStandardRAG, "key=standard_rag")
	check("M10", "template_image_index_present", hasImageIndexRAG, "key=standard_rag_with_image_index")

	// ---------- M2 catalog_table ----------
	tableName := "kb_matrix_tbl_" + strings.ReplaceAll(suffix, "-", "_")
	table, createdTbl, err := db.CreateTable(ctx, tableName, []sdk.TableColumn{
		{Name: "id", Type: "INT"},
		{Name: "name", Type: "VARCHAR(64)"},
	})
	hard("M2", "create_user_table", err, tableName)
	tableID := int64(0)
	if table != nil {
		tableID = int64(table.ID())
	}
	if tableID == 0 && createdTbl != nil {
		tableID = createdTbl.GetId()
	}
	check("M2", "table_id", tableID > 0, fmt.Sprintf("table_id=%d", tableID))
	kbTableName := "kb-matrix-table-" + suffix
	createdTableKB, err := knowledge.CreateWithSources(ctx, kbTableName, []sdk.SemanticModelSourceInput{{
		SourceType: "catalog_table",
		TableID:    tableID,
	}})
	hard("M2", "create_kb_with_table", err, kbTableName)
	modelTable := createdTableKB.GetModel()
	domainTable := createdTableKB.GetDataDomain()
	if modelTable == nil || domainTable == nil {
		hard("M2", "model_domain", fmt.Errorf("empty model/domain"), "")
	}
	modelTableID := strconv.FormatInt(modelTable.GetId(), 10)
	report.Artifacts["m2_model_id"] = modelTableID
	check("M2", "domain_default_catalog",
		domainTable.GetCatalogId() == defaultCatalogID,
		fmt.Sprintf("domain.catalog_id=%d default_catalog_id=%d", domainTable.GetCatalogId(), defaultCatalogID))
	// KB name must equal the knowledge-base catalog physical database name.
	if modelTable.GetName() != kbTableName {
		check("M2", "kb_name", false, fmt.Sprintf("model.name=%q want=%q", modelTable.GetName(), kbTableName))
	} else {
		check("M2", "kb_name", true, kbTableName)
	}
	if domainTable.GetDatabaseId() > 0 {
		dbInfo, dbErr := client.Catalog(ws.ID()).DatabaseInfo(ctx, int(domainTable.GetDatabaseId()))
		if dbErr != nil {
			check("M2", "kb_db_physical_name", false, dbErr.Error())
		} else if dbInfo == nil || dbInfo.GetDatabase() == nil {
			check("M2", "kb_db_physical_name", false, "empty database info")
		} else {
			phys := dbInfo.GetDatabase().GetName()
			disp := dbInfo.GetDatabase().GetDisplayName()
			// Name must be the physical identifier; display_name may carry an optional projection.
			ok := phys == kbTableName
			check("M2", "kb_db_physical_name", ok,
				fmt.Sprintf("name=%q display_name=%q want_name=%q", phys, disp, kbTableName))
			report.Artifacts["m2_kb_database_name"] = phys
		}
	} else {
		check("M2", "kb_db_physical_name", false, "domain.database_id missing")
	}
	sourcesA, err := knowledge.ListSources(ctx, modelTableID)
	hard("M2", "list_sources", err, modelTableID)
	foundTable := false
	tableDisplay := ""
	for _, it := range sourcesA.GetItems() {
		st := it.GetSourceType()
		if st == "table" || st == "catalog_table" {
			foundTable = true
			tableDisplay = it.GetDisplayName()
		}
	}
	check("M2", "list_has_table_source", foundTable, fmt.Sprintf("display=%q n=%d", tableDisplay, len(sourcesA.GetItems())))

	// ---------- M3 structured_upload ----------
	// Catalog metadata/sync can race with background metadata_discovery on a brand-new
	// workspace and fail the structured import once. Retry once with a fresh upload/table.
	connectors := client.Connectors(ws.ID())
	var (
		modelStructID  string
		structSourceOK bool
		lastStatus     string
		lastErr        string
		structAttempts int
	)
	for attempt := 1; attempt <= 2; attempt++ {
		structAttempts = attempt
		csvFile, oerr := os.Open(csvPath)
		hard("M3", "open_csv", oerr, csvPath)
		tmpUpload, uerr := connectors.LocalUpload(ctx, sdk.MultipartFile{
			FieldName: "files",
			Filename:  filepath.Base(csvPath),
			Reader:    csvFile,
		})
		_ = csvFile.Close()
		hard("M3", "connector_local_upload", uerr, filepath.Base(csvPath))
		connFileIDs := tmpUpload.GetConnFileIds()
		sourceFileIDs := tmpUpload.GetFileIds()
		check("M3", "upload_ids", len(connFileIDs) > 0 && connFileIDs[0] != "" && len(sourceFileIDs) > 0 && sourceFileIDs[0] != "",
			fmt.Sprintf("attempt=%d conn_file_ids=%v file_ids=%v", attempt, connFileIDs, sourceFileIDs))
		connFileID, fileID := "", ""
		if len(connFileIDs) > 0 {
			connFileID = connFileIDs[0]
		}
		if len(sourceFileIDs) > 0 {
			fileID = sourceFileIDs[0]
		}
		// Unique table name per attempt to avoid collisions after a partial create.
		structTableName := "reg_struct_" + strings.ReplaceAll(suffix, "-", "_")
		if attempt > 1 {
			structTableName = structTableName + "_r" + strconv.Itoa(attempt)
		}
		tableConfigObj := map[string]any{
			"sheet_name":    filepath.Base(csvPath),
			"new_table":     true,
			"conn_file_ids": []string{connFileID},
			"isColumnName":  true,
			"columnNameRow": 1,
			"rowStart":      2,
			"conflict":      0,
			"csv": map[string]any{
				"separator": ",",
				"delimiter": "\"",
				"isEscape":  false,
			},
			"create_table": map[string]any{
				"name": structTableName,
				"tableColumn": []map[string]any{
					{"column": "成员", "dataType": "VARCHAR", "col_num_in_file": 1},
					{"column": "角色", "dataType": "VARCHAR", "col_num_in_file": 2},
					{"column": "负责事项", "dataType": "VARCHAR", "col_num_in_file": 3},
				},
			},
		}
		tableConfigBytes, _ := json.Marshal(tableConfigObj)
		kbStructName := "kb-matrix-struct-" + suffix
		if attempt > 1 {
			kbStructName = kbStructName + "-r" + strconv.Itoa(attempt)
			// Brief pause so concurrent metadata_discovery settles before re-import.
			time.Sleep(5 * time.Second)
		}
		createdStruct, cerr := knowledge.CreateWithSources(ctx, kbStructName, []sdk.SemanticModelSourceInput{{
			SourceType:  "local_file",
			FileName:    filepath.Base(csvPath),
			FileID:      fileID,
			UploadKind:  "structured",
			TableConfig: string(tableConfigBytes),
		}})
		hard("M3", "create_structured_kb", cerr, kbStructName)
		modelStruct := createdStruct.GetModel()
		domainStruct := createdStruct.GetDataDomain()
		if modelStruct == nil || domainStruct == nil {
			hard("M3", "model_domain", fmt.Errorf("empty"), "")
		}
		modelStructID = strconv.FormatInt(modelStruct.GetId(), 10)
		report.Artifacts["m3_model_id"] = modelStructID
		report.Artifacts["m3_struct_attempt"] = attempt
		check("M3", "domain_default_catalog", domainStruct.GetCatalogId() == defaultCatalogID,
			fmt.Sprintf("domain.catalog_id=%d default_catalog_id=%d", domainStruct.GetCatalogId(), defaultCatalogID))

		deadline := time.Now().Add(4 * time.Minute)
		structSourceOK = false
		lastStatus = ""
		lastErr = ""
		settledOK := false
		terminalFail := false
		for time.Now().Before(deadline) {
			_, _ = knowledge.ReconcileSourceJobs(ctx, modelStructID)
			jobs, jerr := knowledge.ListSourceJobs(ctx, modelStructID)
			if jerr != nil {
				hard("M3", "list_source_jobs", jerr, modelStructID)
			}
			allDone := true
			failed := false
			for _, j := range jobs.GetItems() {
				st := j.GetJobStatus()
				lastStatus = st
				if j.GetError() != "" {
					lastErr = j.GetError()
				}
				switch st {
				case "succeeded", "success", "completed", "skipped":
				case "failed":
					failed = true
				default:
					allDone = false
				}
			}
			sources, serr := knowledge.ListSources(ctx, modelStructID)
			if serr != nil {
				hard("M3", "list_sources_poll", serr, modelStructID)
			}
			sourceIngestFailed := false
			for _, it := range sources.GetItems() {
				if it.GetSourceType() == "table" || it.GetSourceType() == "catalog_table" {
					if it.GetIngestStatus() == "succeeded" || it.GetKbTableId() > 0 || it.GetSourceTableId() > 0 {
						structSourceOK = true
					}
					if strings.EqualFold(it.GetIngestStatus(), "failed") {
						sourceIngestFailed = true
						if it.GetError() != "" {
							lastErr = it.GetError()
						}
					}
					if it.GetDisplayName() != "" {
						lastStatus = "source:" + it.GetIngestStatus() + "/" + it.GetDisplayName()
					}
				}
			}
			if failed || sourceIngestFailed {
				terminalFail = true
				break
			}
			if structSourceOK || (allDone && len(jobs.GetItems()) > 0) {
				settledOK = true
				break
			}
			if !jobs.GetReconcileRequired() && jobs.GetTotal() == 0 && structSourceOK {
				settledOK = true
				break
			}
			time.Sleep(3 * time.Second)
		}
		if settledOK || structSourceOK {
			check("M3", "structured_settled", true,
				fmt.Sprintf("attempt=%d status=%s structSourceOK=%v", attempt, lastStatus, structSourceOK))
			break
		}
		if terminalFail && attempt < 2 {
			fmt.Fprintf(os.Stderr, "[matrix] M3 structured import failed attempt=%d status=%s err=%s; retrying once\n",
				attempt, lastStatus, truncate(lastErr, 200))
			continue
		}
		check("M3", "structured_settled", false,
			fmt.Sprintf("attempt=%d status=%s err=%s", attempt, lastStatus, truncate(lastErr, 300)))
		break
	}
	report.Artifacts["m3_struct_attempts"] = structAttempts
	sourcesB, err := knowledge.ListSources(ctx, modelStructID)
	hard("M3", "list_sources_final", err, modelStructID)
	check("M3", "has_source_row", len(sourcesB.GetItems()) > 0, fmt.Sprintf("n=%d", len(sourcesB.GetItems())))

	// ---------- M4 local_file_doc (unstructured local upload via knowledge API) ----------
	pdfBytes := must(os.ReadFile(pdfPath))
	localName := "MatrixOne_Local_" + suffix + ".pdf"
	upLocal, err := knowledge.UploadLocalFile(ctx, localName, bytes.NewReader(pdfBytes))
	hard("M4", "upload_local_file", err, localName)
	localFileID := ""
	if upLocal != nil {
		localFileID = upLocal.GetFileId()
	}
	check("M4", "local_file_id", localFileID != "", fmt.Sprintf("file_id=%s", localFileID))
	kbLocalName := "kb-matrix-local-" + suffix
	var createdLocal *sdk.SemanticModelWithSourcesResult
	if localFileID != "" {
		createdLocal, err = knowledge.CreateWithSources(ctx, kbLocalName, []sdk.SemanticModelSourceInput{{
			SourceType: "local_file",
			FileID:     localFileID,
			FileName:   localName,
		}})
		hard("M4", "create_kb_local_file", err, kbLocalName)
	} else {
		check("M4", "create_kb_local_file", false, "skip: no file_id from upload")
	}
	modelLocalID := ""
	if createdLocal != nil && createdLocal.GetModel() != nil {
		modelLocalID = strconv.FormatInt(createdLocal.GetModel().GetId(), 10)
		report.Artifacts["m4_model_id"] = modelLocalID
		report.Artifacts["m4_file_id"] = localFileID
		if domain := createdLocal.GetDataDomain(); domain != nil {
			check("M4", "domain_default_catalog", domain.GetCatalogId() == defaultCatalogID,
				fmt.Sprintf("domain.catalog_id=%d default_catalog_id=%d", domain.GetCatalogId(), defaultCatalogID))
		} else {
			check("M4", "domain_default_catalog", false, "empty data domain")
		}
		ingested := false
		dl := time.Now().Add(12 * time.Minute)
		for time.Now().Before(dl) {
			_, _ = knowledge.ReconcileSourceJobs(ctx, modelLocalID)
			jobs, _ := knowledge.ListSourceJobs(ctx, modelLocalID)
			srcs, serr := knowledge.ListSources(ctx, modelLocalID)
			if serr == nil {
				for _, it := range srcs.GetItems() {
					if it.GetIngestStatus() == "succeeded" {
						ingested = true
					}
					if strings.EqualFold(it.GetIngestStatus(), "failed") {
						check("M4", "ingest_not_failed", false, it.GetIngestStatus())
						goto m4done
					}
				}
			}
			if ingested {
				check("M4", "ingest_succeeded", true, modelLocalID)
				goto m4done
			}
			if jobs != nil && !jobs.GetReconcileRequired() && jobs.GetTotal() == 0 && srcs != nil && len(srcs.GetItems()) > 0 {
				// still waiting ingest field
			}
			time.Sleep(3 * time.Second)
		}
		check("M4", "ingest_succeeded", ingested, "timeout")
	m4done:
	}

	// ---------- M5 catalog_file lineage + workflow ----------
	fileName := "MatrixOne_Introduction_" + suffix + ".pdf"
	file := must(vol.UploadBytesHandle(ctx, fileName, pdfBytes))
	sourceFileID := file.ID()
	report.Artifacts["m5_source_file_id"] = sourceFileID
	fmt.Fprintf(os.Stderr, "[matrix] M5 uploaded source_file_id=%s\n", sourceFileID)

	baselineList := must(ws.Workflows().ListFileExecutions(ctx, sdk.WithFileID(sourceFileID)))
	baselineIDs := map[string]struct{}{}
	for _, ex := range baselineList.GetExecutions() {
		if id := ex.GetExecutionId(); id != "" {
			baselineIDs[id] = struct{}{}
		}
	}

	kbFileName := "kb-matrix-file-" + suffix
	createdFile := must(knowledge.CreateWithSources(ctx, kbFileName, []sdk.SemanticModelSourceInput{{
		SourceType: "catalog_file",
		FileID:     sourceFileID,
		VolumeID:   int64(vol.ID()),
	}}))
	modelFile := createdFile.GetModel()
	domainFile := createdFile.GetDataDomain()
	if modelFile == nil || domainFile == nil {
		hard("M5", "model_domain", fmt.Errorf("empty"), "")
	}
	modelFileID := strconv.FormatInt(modelFile.GetId(), 10)
	rawVolumeID := int64(0)
	processedVolumeID := int64(0)
	if domainFile != nil {
		rawVolumeID = domainFile.GetRawVolumeId()
		processedVolumeID = domainFile.GetProcessedVolumeId()
	}
	report.Artifacts["m5_model_id"] = modelFileID
	report.Artifacts["m5_raw_volume_id"] = rawVolumeID
	report.Artifacts["m5_processed_volume_id"] = processedVolumeID
	check("M5", "domain_default_catalog",
		domainFile.GetCatalogId() == defaultCatalogID,
		fmt.Sprintf("domain.catalog_id=%d default_catalog_id=%d", domainFile.GetCatalogId(), defaultCatalogID))

	// settle jobs (do not record per-poll list checks — only final settle)
	dl5 := time.Now().Add(15 * time.Minute)
	for {
		if time.Now().After(dl5) {
			hard("M5", "source_jobs_settle", fmt.Errorf("timeout"), modelFileID)
		}
		jobs, err := knowledge.ListSourceJobs(ctx, modelFileID)
		if err != nil {
			hard("M5", "list_source_jobs", err, modelFileID)
		}
		for _, it := range jobs.GetItems() {
			if strings.EqualFold(it.GetJobStatus(), "failed") {
				hard("M5", "job_not_failed", fmt.Errorf("job failed %s", it.GetJobId()), it.GetJobStatus())
			}
		}
		if jobs.GetReconcileRequired() {
			_, _ = knowledge.ReconcileSourceJobs(ctx, modelFileID)
			time.Sleep(2 * time.Second)
			continue
		}
		if jobs.GetTotal() != 0 {
			time.Sleep(2 * time.Second)
			continue
		}
		check("M5", "source_jobs_settled", true, "reconcile_required=false total=0")
		break
	}

	sources5 := must(knowledge.ListSources(ctx, modelFileID))
	var matched *sdk.SemanticModelSource
	for _, s := range sources5.GetItems() {
		if s.GetSourceFileId() == sourceFileID || s.GetKbFileId() == sourceFileID {
			matched = s
			break
		}
	}
	if matched == nil {
		hard("M5", "source_binding", fmt.Errorf("missing source_file_id=%s", sourceFileID), "")
	}
	st := matched.GetSourceType()
	check("M5", "source_type", st == "file" || st == "catalog_file", st)
	check("M5", "ingest_succeeded", matched.GetIngestStatus() == "succeeded", matched.GetIngestStatus())
	check("M5", "segment_version", matched.GetSegmentVersionId() != "", matched.GetSegmentVersionId())
	check("M5", "index_version", matched.GetIndexVersion() > 0, fmt.Sprintf("%d", matched.GetIndexVersion()))

	rawHasSource := false
	if rawVolumeID > 0 {
		rawFiles := must(ws.Catalog().Files(ctx,
			sdk.WithFileListFilter("volume_id", strconv.FormatInt(rawVolumeID, 10)),
			sdk.WithFileListPageSize(500),
		))
		for _, f := range rawFiles.GetList() {
			if f.GetId() == sourceFileID {
				rawHasSource = true
				break
			}
		}
	}
	check("M5", "raw_volume_no_source", !rawHasSource, fmt.Sprintf("raw_volume_id=%d", rawVolumeID))

	artifact := must(file.Artifact(ctx))
	check("M5", "has_artifact", artifact.GetHasArtifact(), fmt.Sprintf("parsed=%s", artifact.GetParsedFileId()))
	parsedFileID := artifact.GetParsedFileId()

	sourceRole := ""
	artifactRole := ""
	outputFileID := ""
	artifactRowID := ""
	scanRoles := func(list *sdk.CatalogFileListResult) {
		if list == nil {
			return
		}
		for _, row := range list.GetList() {
			if row.GetId() == sourceFileID {
				sourceRole = row.GetWorkflowRole()
			}
			if row.GetWorkflowRole() == "output" && (row.GetRefFileId() == sourceFileID || row.GetParsedFileId() == parsedFileID || row.GetId() == parsedFileID || strings.Contains(row.GetName(), suffix)) {
				artifactRole = "output"
				if artifactRowID == "" {
					artifactRowID = row.GetId()
				}
				if row.GetId() != parsedFileID && outputFileID == "" {
					outputFileID = row.GetId()
				}
			}
		}
	}
	scanRoles(must(vol.Files(ctx, sdk.WithFileListPageSize(500))))
	if processedVolumeID > 0 {
		scanRoles(must(ws.Catalog().Files(ctx,
			sdk.WithFileListFilter("volume_id", strconv.FormatInt(processedVolumeID, 10)),
			sdk.WithFileListPageSize(500),
		)))
	}
	check("M5", "source_role_not_output", sourceRole != "output", fmt.Sprintf("role=%q", sourceRole))

	hasSourceToOutput := false
	if dataAsset, err := file.DataAsset(ctx); err == nil && dataAsset != nil {
		for _, d := range dataAsset.GetDerivations() {
			if d == nil {
				continue
			}
			fid := d.GetFileId()
			if fid != "" && fid != sourceFileID {
				hasSourceToOutput = true
				if fid != parsedFileID && outputFileID == "" {
					outputFileID = fid
				}
			}
		}
	}
	lineage := must(file.LineageOverview(ctx))
	if topo := lineage.GetTopology(); topo != nil {
		tb, _ := json.Marshal(topo)
		ts := string(tb)
		if strings.Contains(ts, sourceFileID) && (strings.Contains(ts, "output") || strings.Contains(ts, "transformed_from")) {
			hasSourceToOutput = true
		}
		_ = os.WriteFile(filepath.Join(outDir, "debug-lineage-topology.json"), tb, 0o644)
	}
	for _, d := range artifact.GetDerivations() {
		if d == nil {
			continue
		}
		m := d.AsMap()
		b, _ := json.Marshal(m)
		s := string(b)
		if strings.Contains(s, "transformed_from") || strings.Contains(s, "output") {
			if fid, ok := m["file_id"].(string); ok && fid != "" && fid != sourceFileID {
				hasSourceToOutput = true
				if fid != parsedFileID && outputFileID == "" {
					outputFileID = fid
				}
			}
		}
	}
	if artifactRowID == "" {
		if outputFileID != "" {
			artifactRowID = outputFileID
		} else {
			artifactRowID = parsedFileID
		}
	}
	if artifactRole != "output" && processedVolumeID > 0 {
		for _, row := range must(ws.Catalog().Files(ctx,
			sdk.WithFileListFilter("volume_id", strconv.FormatInt(processedVolumeID, 10)),
			sdk.WithFileListPageSize(500),
		)).GetList() {
			if row.GetId() == artifactRowID || row.GetId() == parsedFileID || row.GetRefFileId() == sourceFileID {
				if row.GetWorkflowRole() == "output" {
					artifactRole = "output"
					if artifactRowID == "" {
						artifactRowID = row.GetId()
					}
				}
			}
		}
	}
	check("M5", "lineage_source_to_output", hasSourceToOutput, fmt.Sprintf("output=%s parsed=%s", outputFileID, parsedFileID))
	check("M5", "artifact_role_output", artifactRole == "output", fmt.Sprintf("role=%q row=%s", artifactRole, artifactRowID))
	report.Artifacts["m5_output_file_id"] = outputFileID
	report.Artifacts["m5_parsed_file_id"] = parsedFileID
	report.Artifacts["m5_artifact_row_id"] = artifactRowID

	currentList := must(ws.Workflows().ListFileExecutions(ctx, sdk.WithFileID(sourceFileID)))
	var newExecs []*sdk.WorkflowFileExecution
	for _, ex := range currentList.GetExecutions() {
		id := ex.GetExecutionId()
		if id == "" {
			continue
		}
		if _, ok := baselineIDs[id]; !ok {
			newExecs = append(newExecs, ex)
		}
	}
	if len(newExecs) != 1 {
		ids := make([]string, 0, len(newExecs))
		for _, e := range newExecs {
			ids = append(ids, e.GetExecutionId()+"/"+e.GetStatus())
		}
		check("M5", "exactly_one_new_execution", false, fmt.Sprintf("got %d: %v", len(newExecs), ids))
	} else {
		sel := newExecs[0]
		okExec := sel.GetStatus() == "completed" && sel.GetSchedulerVisible() && sel.GetCaseStartState() == "started"
		check("M5", "execution_completed", okExec,
			fmt.Sprintf("id=%s status=%s visible=%v start=%s", sel.GetExecutionId(), sel.GetStatus(), sel.GetSchedulerVisible(), sel.GetCaseStartState()))
		report.Artifacts["m5_execution_id"] = sel.GetExecutionId()
	}

	// ---------- M6 A2A RAG ----------
	sessionID, err := createExploreSessionID(ctx, productEndpoint, pat, ws.ID(), "kb-matrix-"+suffix)
	hard("M6", "create_session", err, "")
	report.Artifacts["m6_session_id"] = sessionID
	agent := must(ws.AgentByCode("explore"))
	modelIDInt, _ := strconv.ParseInt(modelFileID, 10, 64)
	stream, err := agent.StreamText(ctx, sdk.AgentTextMessage{
		RequestID: "matrix-req-" + suffix,
		MessageID: "matrix-msg-" + suffix,
		SessionID: sessionID,
		Text:      question,
		Metadata: map[string]any{
			"matrixflow_client":  "kb-product-matrix",
			"workspace_id":       ws.ID(),
			"session_id":         sessionID,
			"semantic_model_ids": []any{modelIDInt},
			"scope_metadata": map[string]any{
				"semantic_model_ids":   modelFileID,
				"semantic_model_names": kbFileName,
			},
			"scope": map[string]any{
				"workspace_id": ws.ID(),
				"session_id":   sessionID,
				"scope_metadata": map[string]any{
					"semantic_model_ids":   modelFileID,
					"semantic_model_names": kbFileName,
				},
			},
		},
	})
	hard("M6", "stream_text", err, question)
	answerText, taskID, finalState, sourceRefs, err := parseKnowledgeAnswerSSE(stream.Body, 10*time.Minute)
	_ = stream.Body.Close()
	hard("M6", "parse_sse", err, finalState)
	check("M6", "final_completed", finalState == "completed", finalState)
	check("M6", "answer_has_80", strings.Contains(answerText, answerNeed), truncate(answerText, 200))
	hit := false
	for _, ref := range sourceRefs {
		b, _ := json.Marshal(ref)
		if strings.Contains(string(b), sourceFileID) {
			hit = true
			break
		}
	}
	check("M6", "source_ref_hit", hit, fmt.Sprintf("task=%s refs=%d", taskID, len(sourceRefs)))
	report.Artifacts["m6_task_id"] = taskID
	report.Artifacts["m6_answer_excerpt"] = truncate(answerText, 300)

	// ---------- M7 append_source ----------
	file2Name := "MatrixOne_Append_" + suffix + ".pdf"
	file2 := must(vol.UploadBytesHandle(ctx, file2Name, pdfBytes))
	appendID := file2.ID()
	report.Artifacts["m7_append_file_id"] = appendID
	_, err = knowledge.AppendSources(ctx, modelFileID, []sdk.SemanticModelSourceInput{{
		SourceType: "catalog_file",
		FileID:     appendID,
		VolumeID:   int64(vol.ID()),
	}})
	hard("M7", "append_sources", err, appendID)
	// wait a bit for source row
	appendRowID := ""
	for i := 0; i < 30; i++ {
		srcs, _ := knowledge.ListSources(ctx, modelFileID)
		if srcs != nil {
			for _, it := range srcs.GetItems() {
				if it.GetSourceFileId() == appendID || it.GetKbFileId() == appendID {
					appendRowID = it.GetRowId()
					if appendRowID == "" {
						appendRowID = it.GetSourceId()
					}
					break
				}
			}
		}
		if appendRowID != "" {
			break
		}
		time.Sleep(2 * time.Second)
	}
	check("M7", "append_source_listed", appendRowID != "", fmt.Sprintf("row=%s file=%s", appendRowID, appendID))
	report.Artifacts["m7_append_row_id"] = appendRowID

	// ---------- M8 entries ----------
	entryInput := sdk.SemanticEntryInput{
		Kind: "glossary",
		Key:  "matrix_rows_" + strings.ReplaceAll(suffix, "-", "_"),
		Spec: map[string]any{"term": "row count", "definition": "number of rows"},
	}
	entry, err := knowledge.CreateEntry(ctx, modelFileID, entryInput)
	if err != nil {
		check("M8", "create_entry", false, err.Error())
	} else {
		check("M8", "create_entry", entry != nil && entry.GetId() > 0, fmt.Sprintf("id=%d", entry.GetId()))
		if entry != nil && entry.GetId() > 0 {
			eid := entry.GetId()
			eidStr := strconv.FormatInt(eid, 10)
			report.Artifacts["m8_entry_id"] = eid
			entries, lerr := knowledge.ListEntries(ctx, modelFileID)
			hard("M8", "list_entries", lerr, "")
			foundE := false
			if entries != nil {
				for _, e := range entries.GetItems() {
					if e.GetId() == eid {
						foundE = true
					}
				}
			}
			check("M8", "list_contains_entry", foundE, eidStr)
			_, derr := knowledge.DeleteEntry(ctx, modelFileID, eidStr)
			check("M8", "delete_entry", derr == nil, fmt.Sprintf("%v", derr))
		}
	}

	// ---------- M9 delete_source ----------
	if appendRowID != "" {
		_, err = knowledge.DeleteSource(ctx, modelFileID, appendRowID)
		check("M9", "delete_source", err == nil, fmt.Sprintf("row=%s err=%v", appendRowID, err))
		// confirm gone
		srcs, _ := knowledge.ListSources(ctx, modelFileID)
		still := false
		if srcs != nil {
			for _, it := range srcs.GetItems() {
				if it.GetRowId() == appendRowID || it.GetSourceId() == appendRowID {
					still = true
				}
			}
		}
		check("M9", "source_removed", !still, appendRowID)
	} else {
		check("M9", "delete_source", false, "no append row to delete")
	}

	// ---------- optional: existence check ----------
	ex, err := knowledge.CheckSourceExistence(ctx, modelFileID, []string{sourceFileID}, nil)
	if err != nil {
		check("M5", "check_existence", false, err.Error())
	} else {
		check("M5", "check_existence", ex != nil, "ok")
	}

	// ---------- M10 image-index document prep + workflow ----------
	// Product path: CreateWithSources(..., ImageIndexEnabled=true) selects
	// standard_rag_with_image_index ("Prepare Document Knowledge Base with Image Index").
	pdfImageBytes := must(os.ReadFile(pdfImagePath))
	imgFileName := "MatrixOne_ImageMix_" + suffix + ".pdf"
	imgFile := must(vol.UploadBytesHandle(ctx, imgFileName, pdfImageBytes))
	imgSourceFileID := imgFile.ID()
	report.Artifacts["m10_source_file_id"] = imgSourceFileID
	fmt.Fprintf(os.Stderr, "[matrix] M10 uploaded image-mix source_file_id=%s\n", imgSourceFileID)

	imgBaseline := must(ws.Workflows().ListFileExecutions(ctx, sdk.WithFileID(imgSourceFileID)))
	imgBaselineIDs := map[string]struct{}{}
	for _, ex := range imgBaseline.GetExecutions() {
		if id := ex.GetExecutionId(); id != "" {
			imgBaselineIDs[id] = struct{}{}
		}
	}

	kbImgName := "kb-matrix-img-" + suffix
	createdImg := must(knowledge.CreateWithSources(ctx, kbImgName, []sdk.SemanticModelSourceInput{{
		SourceType: "catalog_file",
		FileID:     imgSourceFileID,
		VolumeID:   int64(vol.ID()),
	}}, sdk.WithSemanticModelWithSourcesImageIndexEnabled(true)))
	modelImg := createdImg.GetModel()
	if modelImg == nil {
		hard("M10", "model", fmt.Errorf("empty"), "")
	}
	modelImgID := strconv.FormatInt(modelImg.GetId(), 10)
	report.Artifacts["m10_model_id"] = modelImgID
	check("M10", "create_with_image_index", true, modelImgID)
	if domainImg := createdImg.GetDataDomain(); domainImg != nil {
		check("M10", "domain_default_catalog", domainImg.GetCatalogId() == defaultCatalogID,
			fmt.Sprintf("domain.catalog_id=%d default_catalog_id=%d", domainImg.GetCatalogId(), defaultCatalogID))
	} else {
		check("M10", "domain_default_catalog", false, "empty data domain")
	}

	// Assert model files metadata carries image index config when present.
	gotImg, gerr := knowledge.Get(ctx, modelImgID)
	hard("M10", "get_model", gerr, modelImgID)
	imgIndexHint := false
	if gotImg != nil && gotImg.GetFiles() != nil {
		fb, _ := json.Marshal(gotImg.GetFiles().AsInterface())
		fs := string(fb)
		if strings.Contains(fs, "image_index") || strings.Contains(fs, "image_vector") ||
			strings.Contains(fs, "efficientnet") || strings.Contains(fs, "image_embedding") {
			imgIndexHint = true
		}
		report.Artifacts["m10_files_excerpt"] = truncate(fs, 400)
	}
	// Soft: some stacks only materialize image fields after settle; re-check after jobs.
	// Hard path: jobs settle + ingest + exactly one completed file execution.

	dl10 := time.Now().Add(20 * time.Minute)
	for {
		if time.Now().After(dl10) {
			hard("M10", "source_jobs_settle", fmt.Errorf("timeout"), modelImgID)
		}
		jobs, jerr := knowledge.ListSourceJobs(ctx, modelImgID)
		if jerr != nil {
			hard("M10", "list_source_jobs", jerr, modelImgID)
		}
		for _, it := range jobs.GetItems() {
			if strings.EqualFold(it.GetJobStatus(), "failed") {
				hard("M10", "job_not_failed", fmt.Errorf("job failed %s", it.GetJobId()), it.GetJobStatus())
			}
		}
		if jobs.GetReconcileRequired() {
			_, _ = knowledge.ReconcileSourceJobs(ctx, modelImgID)
			time.Sleep(2 * time.Second)
			continue
		}
		if jobs.GetTotal() != 0 {
			time.Sleep(3 * time.Second)
			continue
		}
		check("M10", "source_jobs_settled", true, "reconcile_required=false total=0")
		break
	}

	if !imgIndexHint {
		if got2, err2 := knowledge.Get(ctx, modelImgID); err2 == nil && got2 != nil && got2.GetFiles() != nil {
			fb, _ := json.Marshal(got2.GetFiles().AsInterface())
			fs := string(fb)
			if strings.Contains(fs, "image_index") || strings.Contains(fs, "image_vector") ||
				strings.Contains(fs, "efficientnet") || strings.Contains(fs, "image_embedding") {
				imgIndexHint = true
			}
			report.Artifacts["m10_files_excerpt"] = truncate(fs, 400)
		}
	}
	check("M10", "image_index_config_present", imgIndexHint, "files metadata mentions image index")

	sources10 := must(knowledge.ListSources(ctx, modelImgID))
	var matchedImg *sdk.SemanticModelSource
	for _, s := range sources10.GetItems() {
		if s.GetSourceFileId() == imgSourceFileID || s.GetKbFileId() == imgSourceFileID {
			matchedImg = s
			break
		}
	}
	if matchedImg == nil {
		hard("M10", "source_binding", fmt.Errorf("missing source_file_id=%s", imgSourceFileID), "")
	}
	check("M10", "ingest_succeeded", matchedImg.GetIngestStatus() == "succeeded", matchedImg.GetIngestStatus())
	check("M10", "segment_version", matchedImg.GetSegmentVersionId() != "", matchedImg.GetSegmentVersionId())
	check("M10", "index_version", matchedImg.GetIndexVersion() > 0, fmt.Sprintf("%d", matchedImg.GetIndexVersion()))

	// Bound workflow instance for image-index path should reference image template naming.
	wfList, wferr := ws.Workflows().List(ctx)
	hard("M10", "list_workflows", wferr, "")
	boundImgWF := false
	if wfList != nil {
		for _, w := range wfList.GetWorkflows() {
			name := w.GetName()
			// Auto-created KB workflows typically include model name or image-index template name.
			if strings.Contains(name, kbImgName) ||
				strings.Contains(strings.ToLower(name), "image") ||
				strings.Contains(name, "Image Index") {
				boundImgWF = true
				report.Artifacts["m10_workflow_id"] = w.GetId()
				report.Artifacts["m10_workflow_name"] = name
			}
		}
	}
	// Soft signal only if naming is stable; hard signal is completed file execution below.
	if boundImgWF {
		check("M10", "workflow_bound", true, fmt.Sprintf("%v", report.Artifacts["m10_workflow_name"]))
	}

	currentImgExecs := must(ws.Workflows().ListFileExecutions(ctx, sdk.WithFileID(imgSourceFileID)))
	var newImgExecs []*sdk.WorkflowFileExecution
	for _, ex := range currentImgExecs.GetExecutions() {
		id := ex.GetExecutionId()
		if id == "" {
			continue
		}
		if _, ok := imgBaselineIDs[id]; !ok {
			newImgExecs = append(newImgExecs, ex)
		}
	}
	if len(newImgExecs) != 1 {
		ids := make([]string, 0, len(newImgExecs))
		for _, e := range newImgExecs {
			ids = append(ids, e.GetExecutionId()+"/"+e.GetStatus())
		}
		check("M10", "exactly_one_new_execution", false, fmt.Sprintf("got %d: %v", len(newImgExecs), ids))
	} else {
		sel := newImgExecs[0]
		okExec := sel.GetStatus() == "completed" && sel.GetSchedulerVisible() && sel.GetCaseStartState() == "started"
		check("M10", "execution_completed", okExec,
			fmt.Sprintf("id=%s status=%s visible=%v start=%s workflow=%s",
				sel.GetExecutionId(), sel.GetStatus(), sel.GetSchedulerVisible(), sel.GetCaseStartState(), sel.GetWorkflowId()))
		report.Artifacts["m10_execution_id"] = sel.GetExecutionId()
		report.Artifacts["m10_execution_workflow_id"] = sel.GetWorkflowId()
	}

	// ---------- M11 A2A RAG on image-index KB ----------
	sessionImgID, err := createExploreSessionID(ctx, productEndpoint, pat, ws.ID(), "kb-matrix-img-"+suffix)
	hard("M11", "create_session", err, "")
	report.Artifacts["m11_session_id"] = sessionImgID
	agentImg := must(ws.AgentByCode("explore"))
	modelImgIDInt, _ := strconv.ParseInt(modelImgID, 10, 64)
	streamImg, err := agentImg.StreamText(ctx, sdk.AgentTextMessage{
		RequestID: "matrix-img-req-" + suffix,
		MessageID: "matrix-img-msg-" + suffix,
		SessionID: sessionImgID,
		Text:      question,
		Metadata: map[string]any{
			"matrixflow_client":  "kb-product-matrix",
			"workspace_id":       ws.ID(),
			"session_id":         sessionImgID,
			"semantic_model_ids": []any{modelImgIDInt},
			"scope_metadata": map[string]any{
				"semantic_model_ids":   modelImgID,
				"semantic_model_names": kbImgName,
			},
			"scope": map[string]any{
				"workspace_id": ws.ID(),
				"session_id":   sessionImgID,
				"scope_metadata": map[string]any{
					"semantic_model_ids":   modelImgID,
					"semantic_model_names": kbImgName,
				},
			},
		},
	})
	hard("M11", "stream_text", err, question)
	answerImg, taskImgID, finalImgState, sourceRefsImg, err := parseKnowledgeAnswerSSE(streamImg.Body, 10*time.Minute)
	_ = streamImg.Body.Close()
	hard("M11", "parse_sse", err, finalImgState)
	check("M11", "final_completed", finalImgState == "completed", finalImgState)
	check("M11", "answer_has_80", strings.Contains(answerImg, answerNeed), truncate(answerImg, 200))
	hitImg := false
	for _, ref := range sourceRefsImg {
		b, _ := json.Marshal(ref)
		if strings.Contains(string(b), imgSourceFileID) {
			hitImg = true
			break
		}
	}
	check("M11", "source_ref_hit", hitImg, fmt.Sprintf("task=%s refs=%d", taskImgID, len(sourceRefsImg)))
	report.Artifacts["m11_task_id"] = taskImgID
	report.Artifacts["m11_answer_excerpt"] = truncate(answerImg, 300)

	writeReport(outDir, report)
	summary := map[string]any{
		"status":       report.Status,
		"ok":           report.OK,
		"workspace_id": report.WorkspaceID,
		"suffix":       report.Suffix,
		"checks":       len(report.Checks),
		"passed":       countPass(report.Checks),
		"failed":       countFail(report.Checks),
		"artifacts":    report.Artifacts,
		"report":       filepath.Join(outDir, "matrix-report.json"),
	}
	fmt.Println(string(must(json.MarshalIndent(summary, "", "  "))))
	if !report.OK {
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "MATRIX PASSED (%d/%d checks)\n", countPass(report.Checks), len(report.Checks))
}

func writeReport(outDir string, report matrixReport) {
	path := filepath.Join(outDir, "matrix-report.json")
	b, _ := json.MarshalIndent(report, "", "  ")
	_ = os.WriteFile(path, b, 0o644)
	fmt.Fprintf(os.Stderr, "[matrix] wrote %s\n", path)
}

func countPass(cs []checkResult) int {
	n := 0
	for _, c := range cs {
		if c.Pass {
			n++
		}
	}
	return n
}

func countFail(cs []checkResult) int {
	return len(cs) - countPass(cs)
}

func createExploreSessionID(ctx context.Context, productEndpoint, pat, workspaceID, title string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"title":  title,
		"source": "moi",
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(productEndpoint, "/")+"/sessions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", pat)
	req.Header.Set("X-Workspace-ID", workspaceID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("create session HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))
	}
	var envelope struct {
		Code string `json:"code"`
		Msg  string `json:"msg"`
		Data *struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", fmt.Errorf("parse create session: %w body=%s", err, truncate(string(raw), 300))
	}
	if envelope.Code != "OK" || envelope.Data == nil || envelope.Data.ID <= 0 {
		return "", fmt.Errorf("create session failed code=%s msg=%s body=%s", envelope.Code, envelope.Msg, truncate(string(raw), 300))
	}
	return strconv.FormatInt(envelope.Data.ID, 10), nil
}

func parseKnowledgeAnswerSSE(r io.Reader, timeout time.Duration) (answer string, taskID string, finalState string, refs []map[string]any, err error) {
	done := make(chan struct{})
	var (
		ans        string
		tid        string
		state      string
		sourceRefs []map[string]any
		parseErr   error
	)
	go func() {
		defer close(done)
		sc := bufio.NewScanner(r)
		buf := make([]byte, 0, 1024*1024)
		sc.Buffer(buf, 16*1024*1024)
		var dataLines []string
		flush := func() {
			if len(dataLines) == 0 {
				return
			}
			payload := strings.Join(dataLines, "\n")
			dataLines = nil
			var envelope map[string]any
			if jerr := json.Unmarshal([]byte(payload), &envelope); jerr != nil {
				return
			}
			result, _ := envelope["result"].(map[string]any)
			if result == nil {
				return
			}
			kind, _ := result["kind"].(string)
			if t, ok := result["taskId"].(string); ok && t != "" && tid == "" {
				tid = t
			}
			switch kind {
			case "artifact-update":
				art, _ := result["artifact"].(map[string]any)
				if art == nil {
					return
				}
				meta, _ := art["metadata"].(map[string]any)
				mfType, _ := meta["matrixflow_type"].(string)
				if mfType != "knowledge.answer" {
					return
				}
				parts, _ := art["parts"].([]any)
				var texts []string
				for _, p := range parts {
					pm, _ := p.(map[string]any)
					if pm == nil {
						continue
					}
					if pm["kind"] == "text" {
						if tx, ok := pm["text"].(string); ok {
							texts = append(texts, tx)
						}
					}
				}
				if len(texts) > 0 {
					ans = strings.Join(texts, "")
				}
				if rawRefs, ok := meta["source_refs"].([]any); ok {
					sourceRefs = sourceRefs[:0]
					for _, rr := range rawRefs {
						if m, ok := rr.(map[string]any); ok {
							sourceRefs = append(sourceRefs, m)
						}
					}
				}
			case "status-update":
				final, _ := result["final"].(bool)
				st, _ := result["status"].(map[string]any)
				stState, _ := st["state"].(string)
				if final {
					state = stState
					if t, ok := result["taskId"].(string); ok && t != "" {
						if tid != "" && t != tid {
							parseErr = fmt.Errorf("final taskId=%s differs from answer taskId=%s", t, tid)
							return
						}
						tid = t
					}
				} else if stState == "failed" || stState == "canceled" || stState == "rejected" {
					state = stState
					parseErr = fmt.Errorf("terminal non-completed state=%s", stState)
				}
			}
		}
		for sc.Scan() {
			line := sc.Text()
			if line == "" {
				flush()
				if state != "" {
					return
				}
				continue
			}
			if strings.HasPrefix(line, "data:") {
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
		flush()
		if parseErr == nil {
			if err := sc.Err(); err != nil {
				parseErr = err
			} else if state == "" {
				parseErr = fmt.Errorf("SSE ended before final status (answer_len=%d task=%s)", len(ans), tid)
			}
		}
	}()
	select {
	case <-done:
		if parseErr != nil {
			return ans, tid, state, sourceRefs, parseErr
		}
		if state != "completed" {
			return ans, tid, state, sourceRefs, fmt.Errorf("final state=%q", state)
		}
		if tid == "" {
			return ans, tid, state, sourceRefs, fmt.Errorf("missing taskId")
		}
		return ans, tid, state, sourceRefs, nil
	case <-time.After(timeout):
		return ans, tid, state, sourceRefs, fmt.Errorf("SSE timeout after %s", timeout)
	}
}

func issuePAT(root, ucBase, productEndpoint, email, password string) (token string, revoke func() error, err error) {
	// Prefer skill runner dir, then .runtime matrix runner, then lineage runner.
	candidates := []string{
		filepath.Join(root, "skills/kb-product-matrix/runner"),
		filepath.Join(root, ".runtime/kb-product-matrix-runner"),
		filepath.Join(root, ".runtime/kb-catalog-lineage-acceptance-runner"),
	}
	var runnerDir, issuePath string
	for _, d := range candidates {
		p := filepath.Join(d, "issue_pat_once.py")
		if fileExists(p) {
			runnerDir = d
			issuePath = p
			break
		}
	}
	if issuePath == "" {
		return "", nil, fmt.Errorf("missing issue_pat_once.py under skill/.runtime runners")
	}
	py := envOr("PYTHON_BIN", "python3")
	if venvPy := filepath.Join(runnerDir, "venv/bin/python3"); fileExists(venvPy) {
		py = venvPy
	} else if venvPy := filepath.Join(root, ".runtime/kb-catalog-lineage-acceptance-runner/venv/bin/python3"); fileExists(venvPy) {
		py = venvPy
	} else if venvPy := filepath.Join(root, ".runtime/kb-catalog-lineage-acceptance-runner/venv/bin/python"); fileExists(venvPy) {
		py = venvPy
	}
	cmd := exec.Command(py, issuePath)
	cmd.Env = append(os.Environ(),
		"UC_PAT_PYDIR="+filepath.Join(root, "moi-backend/api-tester"),
		"PYTHONPATH="+filepath.Join(root, "moi-backend/api-tester"),
		"UC_BASE_URL="+ucBase,
		"PRODUCT_BASE_URL="+productEndpoint,
		"SEED_EMAIL="+email,
		"SEED_PASSWORD="+password,
		"PAT_TTL=7200",
	)
	out, err := cmd.Output()
	if err != nil {
		stderr := []byte{}
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = ee.Stderr
		}
		return "", nil, fmt.Errorf("issue_pat: %v stderr=%s stdout=%s", err, string(stderr), string(out))
	}
	var payload struct {
		Token     string `json:"token"`
		KeyID     string `json:"key_id"`
		KeyETag   string `json:"key_etag"`
		UCBaseURL string `json:"uc_base_url"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return "", nil, fmt.Errorf("parse pat payload: %w", err)
	}
	if payload.Token == "" || payload.KeyID == "" {
		return "", nil, fmt.Errorf("empty token/key from issue_pat")
	}
	revokePath := filepath.Join(runnerDir, "revoke_pat_once.py")
	revokeScript := `
import os, sys
sys.path.insert(0, os.environ["UC_PAT_PYDIR"])
from utils.uc_pat import revoke_uc_pat_by_id
revoke_uc_pat_by_id(
    uc_base_url=os.environ["UC_BASE_URL"],
    email=os.environ["SEED_EMAIL"],
    password=os.environ["SEED_PASSWORD"],
    key_id=os.environ["KEY_ID"],
    key_etag=os.environ["KEY_ETAG"],
    timeout_seconds=60,
)
print("revoked")
`
	if err := os.WriteFile(revokePath, []byte(revokeScript), 0o600); err != nil {
		return "", nil, err
	}
	revoke = func() error {
		c := exec.Command(py, revokePath)
		c.Env = append(os.Environ(),
			"UC_PAT_PYDIR="+filepath.Join(root, "moi-backend/api-tester"),
			"PYTHONPATH="+filepath.Join(root, "moi-backend/api-tester"),
			"UC_BASE_URL="+payload.UCBaseURL,
			"KEY_ID="+payload.KeyID,
			"KEY_ETAG="+payload.KeyETag,
			"SEED_EMAIL="+email,
			"SEED_PASSWORD="+password,
		)
		out, err := c.CombinedOutput()
		if err != nil {
			return fmt.Errorf("revoke: %v out=%s", err, string(out))
		}
		return nil
	}
	return payload.Token, revoke, nil
}
