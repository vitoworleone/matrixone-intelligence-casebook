package service

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/matrixflow/moi-core/agent-tools/knowledge"
)

type stubDescribeSchemaReader struct {
	tables   []string
	columns  []TableColumns
	entries  []knowledge.SemanticEntry
	listErr  error
	colErr   error
	semErr   error
	requests [][]string
}

func (r *stubDescribeSchemaReader) ListTables(context.Context, knowledge.WorkspaceScope) ([]string, error) {
	return append([]string(nil), r.tables...), r.listErr
}

func (r *stubDescribeSchemaReader) ListColumns(_ context.Context, _ knowledge.WorkspaceScope, tableNames []string) ([]TableColumns, error) {
	r.requests = append(r.requests, append([]string(nil), tableNames...))
	return append([]TableColumns(nil), r.columns...), r.colErr
}

func (r *stubDescribeSchemaReader) ListSemanticEntries(context.Context, knowledge.WorkspaceScope) ([]knowledge.SemanticEntry, error) {
	return append([]knowledge.SemanticEntry(nil), r.entries...), r.semErr
}

func (r *stubDescribeSchemaReader) ReadSampleRows(context.Context, knowledge.WorkspaceScope, string, int) ([][]any, error) {
	return nil, nil
}

// MF-2: stored bare entry tables must still attach to qualified selected tables
// when the bare name uniquely maps to one selected table.
func TestSemanticEntriesByTableMatchesBareStoredNames(t *testing.T) {
	entries := []knowledge.SemanticEntry{{
		Kind:    "metric",
		KeyName: "gmv",
		Tables:  []string{"orders"},
	}}
	got := semanticEntriesByTable(entries, []string{"sales.orders", "support.tickets"})
	if len(got["sales.orders"]) != 1 || got["sales.orders"][0].KeyName != "gmv" {
		t.Fatalf("sales.orders entries = %#v, want metric gmv", got["sales.orders"])
	}
	if len(got["support.tickets"]) != 0 {
		t.Fatalf("support.tickets entries = %#v, want empty", got["support.tickets"])
	}
	if len(got["orders"]) != 1 {
		t.Fatalf("bare orders entries = %#v, want metric gmv retained", got["orders"])
	}
}

// Cross-db same bare name: unqualified entry must not attach to every match.
func TestSemanticEntriesByTableSkipsAmbiguousCrossDBBareNames(t *testing.T) {
	entries := []knowledge.SemanticEntry{{
		Kind:    "metric",
		KeyName: "gmv",
		Tables:  []string{"orders"},
	}}
	got := semanticEntriesByTable(entries, []string{"sales.orders", "support.orders"})
	if len(got["sales.orders"]) != 0 {
		t.Fatalf("sales.orders entries = %#v, want empty (ambiguous bare orders)", got["sales.orders"])
	}
	if len(got["support.orders"]) != 0 {
		t.Fatalf("support.orders entries = %#v, want empty (ambiguous bare orders)", got["support.orders"])
	}
	if len(got["orders"]) != 0 {
		t.Fatalf("bare orders entries = %#v, want empty when cross-db ambiguous", got["orders"])
	}
}

func TestSemanticEntriesByTableExactQualifiedMatchStillWorksAcrossSameBareName(t *testing.T) {
	// Entry already qualified to one database must still attach even when the
	// selected scope contains multiple same-bare tables.
	entries := []knowledge.SemanticEntry{{
		Kind:    "metric",
		KeyName: "sales_gmv",
		Tables:  []string{"sales.orders"},
	}}
	got := semanticEntriesByTable(entries, []string{"sales.orders", "support.orders"})
	if len(got["sales.orders"]) != 1 || got["sales.orders"][0].KeyName != "sales_gmv" {
		t.Fatalf("sales.orders entries = %#v, want sales_gmv", got["sales.orders"])
	}
	if len(got["support.orders"]) != 0 {
		t.Fatalf("support.orders entries = %#v, want empty", got["support.orders"])
	}
}

// MF-4: unique bare request must resolve to qualified identity for readers.
func TestResolveDescribeSchemaTablesRewritesBareToQualified(t *testing.T) {
	got, err := resolveDescribeSchemaTablesInScope(
		[]string{"orders"},
		[]string{"sales.orders", "support.tickets"},
		"",
	)
	if err != nil {
		t.Fatalf("resolveDescribeSchemaTablesInScope() error = %v", err)
	}
	if !reflect.DeepEqual(got, []string{"sales.orders"}) {
		t.Fatalf("resolved = %#v, want [sales.orders]", got)
	}
}

// MF-5: qualified table DDL must quote database and table separately.
func TestBuildCreateTableDDLQuotesQualifiedTable(t *testing.T) {
	ddl := buildCreateTableDDL("sales.orders", []knowledge.ColumnInfo{{Name: "id", Type: "INT"}}, "sales")
	if !strings.Contains(ddl, "CREATE TABLE `sales`.`orders`") {
		t.Fatalf("DDL = %q, want CREATE TABLE `sales`.`orders`", ddl)
	}
	if strings.Contains(ddl, "`sales.orders`") {
		t.Fatalf("DDL = %q, must not quote whole sales.orders as one identifier", ddl)
	}
}

func TestBuildCreateTableDDLQuotesQualifiedDottedTableName(t *testing.T) {
	// After resolveTables, dotted physical names are always database.table
	// (ffff_15.test.csv), so DDL must quote database and table separately.
	ddl := buildCreateTableDDL("ffff_15.test.csv", []knowledge.ColumnInfo{{Name: "c1", Type: "VARCHAR"}}, "ffff_15")
	if !strings.Contains(ddl, "CREATE TABLE `ffff_15`.`test.csv`") {
		t.Fatalf("DDL = %q, want CREATE TABLE `ffff_15`.`test.csv`", ddl)
	}
	if strings.Contains(ddl, "`test`.`csv`") || strings.Contains(ddl, "`ffff_15.test.csv`") {
		t.Fatalf("DDL = %q, must not mis-split or quote whole qualified name", ddl)
	}
}

// Without a known database, bare physical xxx.xxx must quote as one identifier.
func TestBuildCreateTableDDLQuotesBareDottedPhysicalNameAsOneIdentifier(t *testing.T) {
	ddl := buildCreateTableDDL("xxx.xxx", []knowledge.ColumnInfo{{Name: "c1", Type: "VARCHAR"}})
	if !strings.Contains(ddl, "CREATE TABLE `xxx.xxx`") {
		t.Fatalf("DDL = %q, want CREATE TABLE `xxx.xxx`", ddl)
	}
	if strings.Contains(ddl, "`xxx`.`xxx`") {
		t.Fatalf("DDL = %q, must not invent database from dots", ddl)
	}
}

func TestDescribeSchemaExecuteRewritesBareRequestAndKeepsSemanticEntries(t *testing.T) {
	reader := &stubDescribeSchemaReader{
		columns: []TableColumns{{
			TableName: "sales.orders",
			Columns:   []knowledge.ColumnInfo{{Name: "id", Type: "INT"}},
		}},
		entries: []knowledge.SemanticEntry{{
			Kind:    "metric",
			KeyName: "order_count",
			Tables:  []string{"orders"},
		}},
	}
	svc := NewDescribeSchema(Deps{SchemaReader: reader})
	resp, err := svc.Execute(context.Background(), knowledge.DescribeSchemaRequest{
		Scope: knowledge.WorkspaceScope{
			WorkspaceID: "ws_1",
			Tables:      []string{"sales.orders", "support.tickets"},
		},
		TableNames: []string{"orders"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(reader.requests) != 1 || !reflect.DeepEqual(reader.requests[0], []string{"sales.orders"}) {
		t.Fatalf("ListColumns requests = %#v, want [[sales.orders]]", reader.requests)
	}
	if len(resp.Tables) != 1 {
		t.Fatalf("tables = %#v, want 1", resp.Tables)
	}
	if resp.Tables[0].Name != "sales.orders" {
		t.Fatalf("table name = %q, want sales.orders", resp.Tables[0].Name)
	}
	if !strings.Contains(resp.Tables[0].DDL, "`sales`.`orders`") {
		t.Fatalf("DDL = %q, want quoted sales.orders parts", resp.Tables[0].DDL)
	}
	if len(resp.Tables[0].SemanticEntries) != 1 || resp.Tables[0].SemanticEntries[0].KeyName != "order_count" {
		t.Fatalf("semantic entries = %#v, want order_count", resp.Tables[0].SemanticEntries)
	}
}
