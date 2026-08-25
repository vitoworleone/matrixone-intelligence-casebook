package session

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
)

func validationMessage(t *testing.T, err error) string {
	t.Helper()
	msg, ok := i18n.Message(context.Background(), err)
	if !ok {
		t.Fatalf("validation error is not localized: %v", err)
	}
	return msg
}

func TestValidateCreateSemanticEntryRequest_DefaultConstraint(t *testing.T) {
	err := ValidateCreateSemanticEntryRequest(CreateSemanticEntryRequest{
		Kind: KindDefaultConstraint,
		Key:  "currency_cny_default",
		Spec: json.RawMessage(`{
			"column": "currency",
			"operator": "=",
			"values": ["CNY"],
			"gate_severity": "required"
		}`),
	})
	if err != nil {
		t.Fatalf("ValidateCreateSemanticEntryRequest() error = %v", err)
	}
}

func TestValidateCreateSemanticEntryRequest_DefaultConstraintRejectsInvalidSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    json.RawMessage
		wantErr string
	}{
		{
			name:    "missing column",
			spec:    json.RawMessage(`{"values":["CNY"]}`),
			wantErr: "column is required",
		},
		{
			name:    "missing values",
			spec:    json.RawMessage(`{"column":"currency","values":[]}`),
			wantErr: "values must have at least one value",
		},
		{
			name:    "invalid operator",
			spec:    json.RawMessage(`{"column":"currency","operator":"LIKE","values":["CNY"]}`),
			wantErr: "operator must be one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateSemanticEntryRequest(CreateSemanticEntryRequest{
				Kind: KindDefaultConstraint,
				Key:  "bad_default",
				Spec: tt.spec,
			})
			if err == nil {
				t.Fatal("expected validation error")
			}
			msg := validationMessage(t, err)
			if !strings.Contains(msg, tt.wantErr) {
				t.Fatalf("message = %v, want substring %q", msg, tt.wantErr)
			}
		})
	}
}

func TestValidateCreateSemanticEntryRequest_DimensionTypeMismatchIncludesSpecField(t *testing.T) {
	err := ValidateCreateSemanticEntryRequest(CreateSemanticEntryRequest{
		Kind: KindDimension,
		Key:  "city",
		Spec: json.RawMessage(`{"column":1}`),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}

	message := validationMessage(t, err)
	if !strings.Contains(message, "spec.column") {
		t.Fatalf("message = %q, want spec.column", message)
	}
	if strings.Contains(message, "<no value>") {
		t.Fatalf("message must not render missing entry key: %q", message)
	}
}

func TestValidateCreateSemanticEntryRequest_SQLResultset(t *testing.T) {
	err := ValidateCreateSemanticEntryRequest(CreateSemanticEntryRequest{
		Kind: KindSQLResultset,
		Key:  "paid_orders",
		Spec: json.RawMessage(`{
				"sql": "SELECT id, amount FROM orders WHERE status = 'PAID'",
				"description": "Paid orders",
				"max_rows": 10000,
				"max_bytes": 2097152,
				"timeout_seconds": 10,
				"retrieval": {
					"enabled": true,
					"embedding_model": "BAAI/bge-m3"
				}
			}`),
	})
	if err != nil {
		t.Fatalf("ValidateCreateSemanticEntryRequest() error = %v", err)
	}
}

func TestValidateCreateSemanticEntryRequest_SQLResultsetRejectsInvalidSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    json.RawMessage
		wantErr string
	}{
		{
			name:    "missing sql",
			spec:    json.RawMessage(`{"description":"Orders"}`),
			wantErr: "sql is required",
		},
		{
			name:    "missing description",
			spec:    json.RawMessage(`{"sql":"SELECT * FROM orders"}`),
			wantErr: "description is required",
		},
		{
			name:    "timeout too large",
			spec:    json.RawMessage(`{"sql":"SELECT * FROM orders","description":"Orders","timeout_seconds":61}`),
			wantErr: "timeout_seconds must be between 0 and 60",
		},
		{
			name:    "expand sql missing declared param",
			spec:    json.RawMessage(`{"sql":"SELECT * FROM orders","description":"Orders","expand_sql":{"sql":"SELECT * FROM orders WHERE status = '{{status}}'","params":["city"]}}`),
			wantErr: "does not reference param",
		},
		{
			name:    "expand sql undeclared param",
			spec:    json.RawMessage(`{"sql":"SELECT * FROM orders","description":"Orders","expand_sql":{"sql":"SELECT * FROM orders WHERE status = '{{status}}' AND city = '{{city}}'","params":["status"]}}`),
			wantErr: "references undeclared param",
		},
		{
			name:    "retrieval enabled without embedding model",
			spec:    json.RawMessage(`{"sql":"SELECT * FROM orders","description":"Orders","retrieval":{"enabled":true}}`),
			wantErr: "retrieval.embedding_model is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateSemanticEntryRequest(CreateSemanticEntryRequest{
				Kind: KindSQLResultset,
				Key:  "bad_resultset",
				Spec: tt.spec,
			})
			if err == nil {
				t.Fatal("expected validation error")
			}
			msg := validationMessage(t, err)
			if !strings.Contains(msg, tt.wantErr) {
				t.Fatalf("message = %v, want substring %q", msg, tt.wantErr)
			}
		})
	}
}

func TestValidateSQLResultsetSpec_ResolveMode(t *testing.T) {
	base := func(mode string, expandSQL bool) CreateSemanticEntryRequest {
		spec := map[string]any{
			"sql":         "SELECT code FROM t",
			"description": "test resultset",
		}
		if mode != "" {
			spec["resolve_mode"] = mode
		}
		if expandSQL {
			spec["expand_sql"] = map[string]any{
				"sql":    "SELECT * FROM t WHERE code = {{code}}",
				"params": []string{"code"},
			}
		}
		raw, _ := json.Marshal(spec)
		return CreateSemanticEntryRequest{
			Kind:   "sql_resultset",
			Key:    "test_rs",
			Tables: []string{"t"},
			Spec:   raw,
		}
	}

	t.Run("empty resolve_mode is valid", func(t *testing.T) {
		if err := ValidateCreateSemanticEntryRequest(base("", false)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("semantic resolve_mode is valid", func(t *testing.T) {
		if err := ValidateCreateSemanticEntryRequest(base("semantic", false)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("passthrough resolve_mode is valid", func(t *testing.T) {
		if err := ValidateCreateSemanticEntryRequest(base("passthrough", false)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid resolve_mode is rejected", func(t *testing.T) {
		err := ValidateCreateSemanticEntryRequest(base("invalid_mode", false))
		if err == nil || !strings.Contains(validationMessage(t, err), "resolve_mode") {
			t.Fatalf("want resolve_mode error, got %v", err)
		}
	})

	t.Run("passthrough with expand_sql is rejected", func(t *testing.T) {
		err := ValidateCreateSemanticEntryRequest(base("passthrough", true))
		if err == nil {
			t.Fatal("want passthrough+expand_sql error, got nil")
		}
		msg := validationMessage(t, err)
		if !strings.Contains(msg, "passthrough") || !strings.Contains(msg, "expand_sql") {
			t.Fatalf("want passthrough+expand_sql error, got %v", err)
		}
	})
}

func TestValidateCreateSemanticEntryRequest_RejectsDisabledLegacyEntry(t *testing.T) {
	err := ValidateCreateSemanticEntryRequest(CreateSemanticEntryRequest{
		Kind:   KindLogicText,
		Key:    "disabled_legacy_rule",
		Tables: []string{"__disabled_legacy_obsolete_rule__"},
		Spec: json.RawMessage(`{
			"content": "disabled",
			"injection_stages": ["planner_policy"]
		}`),
	})
	if err == nil {
		t.Fatal("expected disabled legacy semantic entry to be rejected")
	}
	if !strings.Contains(validationMessage(t, err), "disabled legacy semantic entries") {
		t.Fatalf("unexpected error: %v", err)
	}
}
