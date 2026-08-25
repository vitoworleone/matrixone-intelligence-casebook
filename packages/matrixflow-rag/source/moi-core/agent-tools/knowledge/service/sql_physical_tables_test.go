package service

import (
	"reflect"
	"testing"
)

func TestExtractPhysicalTableNamesExcludesCTENames(t *testing.T) {
	got := ExtractPhysicalTableNames(`
WITH recent_orders AS (
  SELECT * FROM orders WHERE paid_at IS NOT NULL
)
SELECT c.region, COUNT(*)
FROM recent_orders ro
JOIN customers c ON c.customer_id = ro.customer_id
GROUP BY c.region`)
	want := []string{"orders", "customers"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tables = %v, want %v", got, want)
	}
}

func TestExtractPhysicalTableNamesKeepsQuotedDotTableNameAsName(t *testing.T) {
	got := collectPhysicalTableRefs("SELECT * FROM `test.csv`")
	if len(got) != 1 {
		t.Fatalf("refs = %#v, want one table", got)
	}
	if got[0].schema != "" || got[0].name != "test.csv" {
		t.Fatalf("ref = %#v, want schema empty and name test.csv", got[0])
	}
	names := ExtractPhysicalTableNames("SELECT * FROM `test.csv`")
	if !reflect.DeepEqual(names, []string{"test.csv"}) {
		t.Fatalf("names = %#v, want test.csv", names)
	}
}

func TestExtractPhysicalTableNamesKeepsSchemaQualifiedIdentity(t *testing.T) {
	got := collectPhysicalTableRefs("SELECT * FROM other.orders")
	if len(got) != 1 {
		t.Fatalf("refs = %#v, want one table", got)
	}
	if got[0].schema != "other" || got[0].name != "orders" {
		t.Fatalf("ref = %#v, want other.orders", got[0])
	}
}
