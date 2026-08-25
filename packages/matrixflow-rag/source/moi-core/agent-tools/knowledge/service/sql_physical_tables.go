package service

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	_ "github.com/pingcap/tidb/pkg/parser/test_driver"
)

var mysqlParserPool = sync.Pool{
	New: func() any {
		return parser.New()
	},
}

func parseSingleMySQLStatement(sql string) (ast.StmtNode, error) {
	stmtNodes, err := parseMySQLStatements(sql)
	if err != nil {
		return nil, err
	}
	if len(stmtNodes) != 1 {
		return nil, fmt.Errorf("expected exactly one statement, got %d", len(stmtNodes))
	}
	return stmtNodes[0], nil
}

func parseMySQLStatements(sql string) ([]ast.StmtNode, error) {
	p := mysqlParserPool.Get().(*parser.Parser)
	defer mysqlParserPool.Put(p)
	p.Reset()

	stmtNodes, _, err := p.ParseSQL(strings.TrimSpace(sql))
	if err != nil {
		return nil, err
	}
	return stmtNodes, nil
}

// isPureScalarSelectSQL reports whether sql is a single SELECT (or UNION of
// SELECTs) that never references a FROM clause. These are valid MySQL scalar
// expressions such as SELECT NOW() / SELECT 1 and must not be blocked by the
// physical-table policy used for scoped table queries.
func isPureScalarSelectSQL(sql string) bool {
	stmt, err := parseSingleMySQLStatement(querySQLValidationBody(sql))
	if err != nil || stmt == nil {
		return false
	}
	return isPureScalarSelectNode(stmt)
}

func isPureScalarSelectNode(node ast.Node) bool {
	switch s := node.(type) {
	case *ast.SelectStmt:
		if s.From != nil {
			return false
		}
		// WITH ... SELECT without FROM still materializes CTE names; reject if
		// the WITH clause is present so physical-table policy stays strict for
		// non-trivial scaffolding.
		if s.With != nil {
			return false
		}
		return true
	case *ast.SetOprStmt:
		if s.With != nil {
			return false
		}
		if s.SelectList == nil {
			return false
		}
		return isPureScalarSelectNode(s.SelectList)
	case *ast.SetOprSelectList:
		if s.With != nil {
			return false
		}
		if len(s.Selects) == 0 {
			return false
		}
		for _, sel := range s.Selects {
			if !isPureScalarSelectNode(sel) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// ExtractPhysicalTableNames returns physical table references from SQL in
// first-seen order. CTE names and derived-table aliases are intentionally
// omitted so downstream provenance points at user data tables, not
// intermediate query scaffolding.
func ExtractPhysicalTableNames(sql string) []string {
	refs := collectPhysicalTableRefs(sql)
	if len(refs) == 0 {
		return nil
	}
	out := make([]string, 0, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		name := ref.display()
		if strings.TrimSpace(name) == "" {
			continue
		}
		key := ref.key()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}

type physicalTableRef struct {
	schema string
	name   string
}

func (r physicalTableRef) nameKey() string {
	return normalizeSQLIdentName(r.name)
}

func (r physicalTableRef) schemaKey() string {
	return normalizeSQLIdentName(r.schema)
}

func (r physicalTableRef) key() string {
	name := r.nameKey()
	if schema := r.schemaKey(); schema != "" {
		return schema + "." + name
	}
	return name
}

func (r physicalTableRef) display() string {
	name := strings.TrimSpace(r.name)
	if schema := strings.TrimSpace(r.schema); schema != "" {
		return schema + "." + name
	}
	return name
}

func normalizeSQLIdentName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "`\"")
	return strings.ToLower(s)
}

func collectPhysicalTableRefs(sql string) []physicalTableRef {
	stmt, err := parseSingleMySQLStatement(sql)
	if err != nil || stmt == nil {
		return nil
	}
	v := &physicalTableVisitor{cteNames: make(map[string]int)}
	stmt.Accept(v)
	return v.tables
}

type physicalTableVisitor struct {
	cteNames      map[string]int
	tables        []physicalTableRef
	frames        []cteScopeFrame
	withFrames    []withTraversalFrame
	cteBodyFrames []cteBodyFrame
}

type cteScopeFrame struct {
	names []string
}

type withTraversalFrame struct {
	names     []string
	recursive bool
	index     int
}

type cteBodyFrame struct {
	removed []string
}

func (v *physicalTableVisitor) Enter(n ast.Node) (ast.Node, bool) {
	switch x := n.(type) {
	case *ast.SelectStmt:
		v.pushCTENames(cteNamesFromWith(x.With))
	case *ast.SetOprStmt:
		v.pushCTENames(cteNamesFromWith(x.With))
	case *ast.SetOprSelectList:
		v.pushCTENames(cteNamesFromWith(x.With))
	case *ast.WithClause:
		v.withFrames = append(v.withFrames, withTraversalFrame{
			names:     cteNamesFromWith(x),
			recursive: x.IsRecursive,
		})
	case *ast.CommonTableExpression:
		v.enterCTEBody(x)
	case *ast.TableName:
		if x.IsAlias {
			return n, false
		}
		ref := physicalTableRef{schema: x.Schema.O, name: x.Name.O}
		if ref.schemaKey() == "" && v.isCTEName(ref.nameKey()) {
			return n, false
		}
		v.tables = append(v.tables, ref)
	}
	return n, false
}

func (v *physicalTableVisitor) Leave(n ast.Node) (ast.Node, bool) {
	switch n.(type) {
	case *ast.SelectStmt, *ast.SetOprStmt, *ast.SetOprSelectList:
		v.popCTENames()
	case *ast.WithClause:
		if len(v.withFrames) > 0 {
			v.withFrames = v.withFrames[:len(v.withFrames)-1]
		}
	case *ast.CommonTableExpression:
		v.leaveCTEBody()
	}
	return n, true
}

func (v *physicalTableVisitor) pushCTENames(names []string) {
	v.frames = append(v.frames, cteScopeFrame{names: names})
	for _, name := range names {
		if name == "" {
			continue
		}
		v.cteNames[name]++
	}
}

func (v *physicalTableVisitor) popCTENames() {
	if len(v.frames) == 0 {
		return
	}
	frame := v.frames[len(v.frames)-1]
	v.frames = v.frames[:len(v.frames)-1]
	for _, name := range frame.names {
		if v.cteNames[name] <= 1 {
			delete(v.cteNames, name)
			continue
		}
		v.cteNames[name]--
	}
}

func (v *physicalTableVisitor) enterCTEBody(cte *ast.CommonTableExpression) {
	if len(v.withFrames) == 0 {
		v.cteBodyFrames = append(v.cteBodyFrames, cteBodyFrame{})
		return
	}
	frame := &v.withFrames[len(v.withFrames)-1]
	visible := make(map[string]struct{}, len(frame.names))
	for i := 0; i < frame.index && i < len(frame.names); i++ {
		visible[frame.names[i]] = struct{}{}
	}
	current := normalizeSQLIdentName(cte.Name.O)
	if frame.recursive || cte.IsRecursive {
		visible[current] = struct{}{}
	}

	removed := make([]string, 0, len(frame.names))
	seen := make(map[string]struct{}, len(frame.names))
	for _, name := range frame.names {
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		if _, ok := visible[name]; ok {
			continue
		}
		if v.cteNames[name] == 0 {
			continue
		}
		v.cteNames[name]--
		if v.cteNames[name] == 0 {
			delete(v.cteNames, name)
		}
		removed = append(removed, name)
	}
	v.cteBodyFrames = append(v.cteBodyFrames, cteBodyFrame{removed: removed})
}

func (v *physicalTableVisitor) leaveCTEBody() {
	if len(v.cteBodyFrames) == 0 {
		return
	}
	frame := v.cteBodyFrames[len(v.cteBodyFrames)-1]
	v.cteBodyFrames = v.cteBodyFrames[:len(v.cteBodyFrames)-1]
	for _, name := range frame.removed {
		v.cteNames[name]++
	}
	if len(v.withFrames) > 0 {
		v.withFrames[len(v.withFrames)-1].index++
	}
}

func (v *physicalTableVisitor) isCTEName(name string) bool {
	_, ok := v.cteNames[name]
	return ok
}

func cteNamesFromWith(with *ast.WithClause) []string {
	if with == nil || len(with.CTEs) == 0 {
		return nil
	}
	names := make([]string, 0, len(with.CTEs))
	for _, cte := range with.CTEs {
		if cte == nil {
			continue
		}
		if name := normalizeSQLIdentName(cte.Name.O); name != "" {
			names = append(names, name)
		}
	}
	return names
}
