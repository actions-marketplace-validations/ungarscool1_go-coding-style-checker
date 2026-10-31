package main

import (
	"fmt"
	"go/ast"
)

// depthCeiling caps nesting inside a function body. Unlike the other budgets
// this one has no backlog, so any violation fails immediately.
const depthCeiling = 4

// deepFunctions reports functions nested past the ceiling. Test files count.
func deepFunctions(f *sourceFile) []finding {
	var out []finding
	for _, decl := range f.syntax.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		m := meter{}
		m.block(fn.Body, 0)
		if m.deepest <= depthCeiling {
			continue
		}
		out = append(out, finding{
			path:   f.path,
			line:   f.fset.Position(fn.Pos()).Line,
			module: f.module,
			message: fmt.Sprintf(
				"%s nests %d levels deep (line %d), over the %d ceiling",
				funcName(fn), m.deepest, f.fset.Position(m.at.Pos()).Line, depthCeiling,
			),
		})
	}
	return out
}

// meter counts nesting from level 0 at the body, adding one per block-introducing
// statement. An `else if` chain stays at the level of its first `if`.
type meter struct {
	deepest int
	at      ast.Node
}

func (m *meter) mark(level int, node ast.Node) {
	if level > m.deepest {
		m.deepest = level
		m.at = node
	}
}

func (m *meter) block(body *ast.BlockStmt, level int) {
	if body == nil {
		return
	}
	for _, stmt := range body.List {
		m.stmt(stmt, level)
	}
}

func (m *meter) stmt(stmt ast.Stmt, level int) {
	switch typed := stmt.(type) {
	case *ast.IfStmt:
		m.mark(level+1, typed)
		m.block(typed.Body, level+1)
		m.elseBranch(typed.Else, level)
	case *ast.ForStmt:
		m.nested(typed, typed.Body, level)
	case *ast.RangeStmt:
		m.nested(typed, typed.Body, level)
	case *ast.BlockStmt:
		m.nested(typed, typed, level)
	case *ast.SwitchStmt:
		m.clauses(typed, typed.Body, level)
	case *ast.TypeSwitchStmt:
		m.clauses(typed, typed.Body, level)
	case *ast.SelectStmt:
		m.clauses(typed, typed.Body, level)
	case *ast.LabeledStmt:
		m.stmt(typed.Stmt, level)
	}
	m.funcLits(stmt, level)
}

func (m *meter) nested(node ast.Node, body *ast.BlockStmt, level int) {
	m.mark(level+1, node)
	m.block(body, level+1)
}

func (m *meter) elseBranch(stmt ast.Stmt, level int) {
	switch typed := stmt.(type) {
	case *ast.IfStmt:
		m.mark(level+1, typed)
		m.block(typed.Body, level+1)
		m.elseBranch(typed.Else, level)
	case *ast.BlockStmt:
		m.block(typed, level+1)
	}
}

func (m *meter) clauses(node ast.Node, body *ast.BlockStmt, level int) {
	m.mark(level+1, node)
	if body == nil {
		return
	}
	for _, clause := range body.List {
		for _, stmt := range clauseBody(clause) {
			m.stmt(stmt, level+1)
		}
	}
}

func clauseBody(clause ast.Stmt) []ast.Stmt {
	switch typed := clause.(type) {
	case *ast.CaseClause:
		return typed.Body
	case *ast.CommClause:
		return typed.Body
	}
	return nil
}

// funcLits meters closures hiding in expressions. A literal's body starts one
// level below the statement that holds it, so a callback inside a loop inside
// an `if` is as deep as it looks.
func (m *meter) funcLits(stmt ast.Stmt, level int) {
	ast.Inspect(stmt, func(node ast.Node) bool {
		lit, ok := node.(*ast.FuncLit)
		if !ok {
			return true
		}
		m.mark(level+1, lit)
		m.block(lit.Body, level+1)
		return false
	})
}

func funcName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	return receiverName(fn.Recv.List[0].Type) + "." + fn.Name.Name
}

func receiverName(expr ast.Expr) string {
	switch typed := expr.(type) {
	case *ast.StarExpr:
		return receiverName(typed.X)
	case *ast.Ident:
		return typed.Name
	case *ast.IndexExpr:
		return receiverName(typed.X)
	}
	return "?"
}
