package session

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/matrixorigin/matrixflow/moi-backend/pkg/i18n"
)

var (
	sqlResultsetParamName  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	sqlResultsetParamToken = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)
)

// ValidateCreateSemanticEntryRequest validates a create/update entry request.
func ValidateCreateSemanticEntryRequest(req CreateSemanticEntryRequest) error {
	if strings.TrimSpace(req.Kind) == "" {
		return localizedError(i18n.KeySessionKindRequired, nil)
	}
	if !ValidSemanticKinds[req.Kind] {
		return invalidSemanticKindError(req.Kind)
	}
	if strings.TrimSpace(req.Key) == "" {
		return localizedError(i18n.KeySessionKeyRequired, nil)
	}
	if IsDisabledLegacySemanticEntryTables(req.Tables) {
		return localizedError(i18n.KeySessionDisabledLegacyEntries, nil)
	}
	if len(req.Spec) == 0 {
		return localizedError(i18n.KeySessionSpecRequired, nil)
	}
	return validateSpecByKind(req.Kind, req.Spec)
}

// IsDisabledLegacySemanticEntryTables reports whether an entry carries the
// legacy internal marker used by old exports to keep obsolete semantic rules out
// of retrieval scope. New create/update paths should reject these entries, and
// import paths should skip them.
func IsDisabledLegacySemanticEntryTables(tables []string) bool {
	for _, table := range tables {
		name := normalizeSemanticEntryTableName(table)
		if strings.HasPrefix(name, "__disabled_legacy_") && strings.HasSuffix(name, "__") {
			return true
		}
	}
	return false
}

func normalizeSemanticEntryTableName(raw string) string {
	name := strings.Trim(strings.TrimSpace(strings.ToLower(raw)), "`\"")
	if dot := strings.LastIndex(name, "."); dot >= 0 {
		name = name[dot+1:]
	}
	return name
}

func validateSpecByKind(kind string, spec json.RawMessage) error {
	switch kind {
	case KindDimension:
		return validateDimensionSpec(spec)
	case KindFact:
		return validateFactSpec(spec)
	case KindMetric:
		return validateMetricSpec(spec)
	case KindRelationship:
		return validateRelationshipSpec(spec)
	case KindColumnPreference:
		return validateColumnPreferenceSpec(spec)
	case KindNamedFilter:
		return validateNamedFilterSpec(spec)
	case KindDefaultConstraint:
		return validateDefaultConstraintSpec(spec)
	case KindVerifiedQuery:
		return validateVerifiedQuerySpec(spec)
	case KindGlossary:
		return validateGlossarySpec(spec)
	case KindLogicText:
		return validateLogicTextSpec(spec)
	case KindSQLResultset:
		return validateSQLResultsetSpec(spec)
	default:
		return invalidSemanticKindError(kind)
	}
}

func validateDimensionSpec(spec json.RawMessage) error {
	var s struct {
		Column string `json:"column"`
	}
	if err := json.Unmarshal(spec, &s); err != nil {
		return invalidSemanticSpecError(KindDimension, err)
	}
	if strings.TrimSpace(s.Column) == "" {
		return localizedError(i18n.KeySessionDimensionColumnRequired, nil)
	}
	return nil
}

func validateFactSpec(spec json.RawMessage) error {
	var s struct {
		Column string `json:"column"`
	}
	if err := json.Unmarshal(spec, &s); err != nil {
		return invalidSemanticSpecError(KindFact, err)
	}
	if strings.TrimSpace(s.Column) == "" {
		return localizedError(i18n.KeySessionFactColumnRequired, nil)
	}
	return nil
}

func validateMetricSpec(spec json.RawMessage) error {
	var s struct {
		Expr string `json:"expr"`
	}
	if err := json.Unmarshal(spec, &s); err != nil {
		return invalidSemanticSpecError(KindMetric, err)
	}
	if strings.TrimSpace(s.Expr) == "" {
		return localizedError(i18n.KeySessionMetricExprRequired, nil)
	}
	return nil
}

func validateRelationshipSpec(spec json.RawMessage) error {
	var s struct {
		LeftTable   string `json:"left_table"`
		RightTable  string `json:"right_table"`
		JoinColumns []struct {
			Left  string `json:"left"`
			Right string `json:"right"`
		} `json:"join_columns"`
	}
	if err := json.Unmarshal(spec, &s); err != nil {
		return invalidSemanticSpecError(KindRelationship, err)
	}
	if strings.TrimSpace(s.LeftTable) == "" {
		return localizedError(i18n.KeySessionRelationshipLeftRequired, nil)
	}
	if strings.TrimSpace(s.RightTable) == "" {
		return localizedError(i18n.KeySessionRelationshipRightRequired, nil)
	}
	if len(s.JoinColumns) == 0 {
		return localizedError(i18n.KeySessionRelationshipJoinRequired, nil)
	}
	for i, pair := range s.JoinColumns {
		if strings.TrimSpace(pair.Left) == "" || strings.TrimSpace(pair.Right) == "" {
			return localizedError(i18n.KeySessionRelationshipJoinPairRequired, map[string]any{"Index": i})
		}
	}
	return nil
}

func validateColumnPreferenceSpec(spec json.RawMessage) error {
	var s struct {
		Preferred  string `json:"preferred"`
		Deprecated string `json:"deprecated"`
	}
	if err := json.Unmarshal(spec, &s); err != nil {
		return invalidSemanticSpecError(KindColumnPreference, err)
	}
	if strings.TrimSpace(s.Preferred) == "" {
		return localizedError(i18n.KeySessionColumnPreferredRequired, nil)
	}
	if strings.TrimSpace(s.Deprecated) == "" {
		return localizedError(i18n.KeySessionColumnDeprecatedRequired, nil)
	}
	if strings.EqualFold(s.Preferred, s.Deprecated) {
		return localizedError(i18n.KeySessionColumnPreferredDeprecated, nil)
	}
	return nil
}

func validateNamedFilterSpec(spec json.RawMessage) error {
	var s struct {
		Expr string `json:"expr"`
	}
	if err := json.Unmarshal(spec, &s); err != nil {
		return invalidSemanticSpecError(KindNamedFilter, err)
	}
	if strings.TrimSpace(s.Expr) == "" {
		return localizedError(i18n.KeySessionNamedFilterExprRequired, nil)
	}
	return nil
}

func validateDefaultConstraintSpec(spec json.RawMessage) error {
	var s struct {
		Column   string   `json:"column"`
		Operator string   `json:"operator,omitempty"`
		Values   []string `json:"values"`
	}
	if err := json.Unmarshal(spec, &s); err != nil {
		return invalidSemanticSpecError(KindDefaultConstraint, err)
	}
	if strings.TrimSpace(s.Column) == "" {
		return localizedError(i18n.KeySessionDefaultConstraintColumnRequired, nil)
	}
	if countNonEmptyStrings(s.Values) == 0 {
		return localizedError(i18n.KeySessionDefaultConstraintValuesRequired, nil)
	}
	op := strings.TrimSpace(s.Operator)
	if op == "" {
		op = "="
	}
	switch strings.ToUpper(op) {
	case "=", "!=", "<>", "IN", "NOT IN":
		return nil
	default:
		return localizedError(i18n.KeySessionDefaultConstraintOperatorInvalid, nil)
	}
}

func validateVerifiedQuerySpec(spec json.RawMessage) error {
	var s struct {
		Question string `json:"question"`
		SQL      string `json:"sql"`
	}
	if err := json.Unmarshal(spec, &s); err != nil {
		return invalidSemanticSpecError(KindVerifiedQuery, err)
	}
	if strings.TrimSpace(s.Question) == "" {
		return localizedError(i18n.KeySessionVerifiedQuestionRequired, nil)
	}
	if strings.TrimSpace(s.SQL) == "" {
		return localizedError(i18n.KeySessionVerifiedSQLRequired, nil)
	}
	return nil
}

func countNonEmptyStrings(values []string) int {
	count := 0
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			count++
		}
	}
	return count
}

func validateGlossarySpec(spec json.RawMessage) error {
	var s struct {
		Term       string `json:"term"`
		Definition string `json:"definition"`
	}
	if err := json.Unmarshal(spec, &s); err != nil {
		return invalidSemanticSpecError(KindGlossary, err)
	}
	if strings.TrimSpace(s.Term) == "" {
		return localizedError(i18n.KeySessionGlossaryTermRequired, nil)
	}
	if strings.TrimSpace(s.Definition) == "" {
		return localizedError(i18n.KeySessionGlossaryDefinitionRequired, nil)
	}
	return nil
}

func validateLogicTextSpec(spec json.RawMessage) error {
	var s struct {
		Content         string   `json:"content"`
		InjectionStages []string `json:"injection_stages"`
	}
	if err := json.Unmarshal(spec, &s); err != nil {
		return invalidSemanticSpecError(KindLogicText, err)
	}
	if strings.TrimSpace(s.Content) == "" {
		return localizedError(i18n.KeySessionLogicTextContentRequired, nil)
	}
	if len(s.InjectionStages) == 0 {
		return localizedError(i18n.KeySessionLogicTextStagesRequired, nil)
	}
	for _, stage := range s.InjectionStages {
		if !ValidInjectionStages[stage] {
			return localizedError(i18n.KeySessionLogicTextInvalidStage, map[string]any{"Stage": stage})
		}
	}
	return nil
}

func validateSQLResultsetSpec(spec json.RawMessage) error {
	var s struct {
		SQL         string `json:"sql"`
		Description string `json:"description"`
		ResolveMode string `json:"resolve_mode,omitempty"`
		ExpandSQL   *struct {
			SQL    string   `json:"sql"`
			Params []string `json:"params"`
		} `json:"expand_sql"`
		Retrieval *struct {
			Enabled        bool   `json:"enabled"`
			EmbeddingModel string `json:"embedding_model"`
		} `json:"retrieval"`
		MaxRows        int `json:"max_rows,omitempty"`
		MaxBytes       int `json:"max_bytes,omitempty"`
		TimeoutSeconds int `json:"timeout_seconds,omitempty"`
	}
	if err := json.Unmarshal(spec, &s); err != nil {
		return invalidSemanticSpecError(KindSQLResultset, err)
	}
	if strings.TrimSpace(s.SQL) == "" {
		return localizedError(i18n.KeySessionSQLResultSQLRequired, nil)
	}
	if len([]byte(s.SQL)) > 16*1024 {
		return localizedError(i18n.KeySessionSQLResultSQLTooLarge, nil)
	}
	if strings.TrimSpace(s.Description) == "" {
		return localizedError(i18n.KeySessionSQLResultDescriptionRequired, nil)
	}
	if len([]rune(s.Description)) > 1000 {
		return localizedError(i18n.KeySessionSQLResultDescriptionTooLarge, nil)
	}
	if s.MaxRows < 0 {
		return localizedError(i18n.KeySessionSQLResultMaxRowsPositive, nil)
	}
	if s.MaxBytes < 0 {
		return localizedError(i18n.KeySessionSQLResultMaxBytesPositive, nil)
	}
	if s.TimeoutSeconds < 0 || s.TimeoutSeconds > 60 {
		return localizedError(i18n.KeySessionSQLResultTimeoutInvalid, nil)
	}
	switch s.ResolveMode {
	case "", "semantic", "passthrough":
	default:
		return localizedError(i18n.KeySessionSQLResultResolveModeInvalid, nil)
	}
	if s.ResolveMode == "passthrough" && s.ExpandSQL != nil {
		return localizedError(i18n.KeySessionSQLResultPassthroughExpandSQLConflict, nil)
	}
	if s.ExpandSQL != nil {
		if strings.TrimSpace(s.ExpandSQL.SQL) == "" {
			return localizedError(i18n.KeySessionExpandSQLRequired, nil)
		}
		if len([]byte(s.ExpandSQL.SQL)) > 16*1024 {
			return localizedError(i18n.KeySessionExpandSQLTooLarge, nil)
		}
		if len(s.ExpandSQL.Params) == 0 {
			return localizedError(i18n.KeySessionExpandParamsRequired, nil)
		}
		if len(s.ExpandSQL.Params) > 8 {
			return localizedError(i18n.KeySessionExpandParamsTooMany, nil)
		}
		seen := make(map[string]struct{}, len(s.ExpandSQL.Params))
		for _, param := range s.ExpandSQL.Params {
			name := strings.TrimSpace(param)
			if name == "" {
				return localizedError(i18n.KeySessionExpandParamEmptyName, nil)
			}
			if !sqlResultsetParamName.MatchString(name) {
				return localizedError(i18n.KeySessionExpandParamInvalid, map[string]any{"Param": name})
			}
			key := strings.ToLower(name)
			if _, ok := seen[key]; ok {
				return localizedError(i18n.KeySessionExpandParamDuplicated, map[string]any{"Param": name})
			}
			seen[key] = struct{}{}
			if !sqlResultsetExpandSQLReferencesParam(s.ExpandSQL.SQL, name) {
				return localizedError(i18n.KeySessionExpandSQLParamNotReferenced, map[string]any{"Param": name})
			}
		}
		for _, match := range sqlResultsetParamToken.FindAllStringSubmatch(s.ExpandSQL.SQL, -1) {
			if len(match) == 2 {
				if _, ok := seen[strings.ToLower(match[1])]; !ok {
					return localizedError(i18n.KeySessionExpandSQLParamUndeclared, map[string]any{"Param": match[1]})
				}
			}
		}
	}
	if s.Retrieval != nil && s.Retrieval.Enabled && strings.TrimSpace(s.Retrieval.EmbeddingModel) == "" {
		return localizedError(i18n.KeySessionSQLResultRetrievalEmbeddingModelRequired, nil)
	}
	return nil
}

func sqlResultsetExpandSQLReferencesParam(sqlText string, param string) bool {
	for _, match := range sqlResultsetParamToken.FindAllStringSubmatch(sqlText, -1) {
		if len(match) == 2 && strings.EqualFold(match[1], param) {
			return true
		}
	}
	return false
}
