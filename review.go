package main

import (
	"fmt"
	"go/ast"
)

const (
	branchCeiling = 3
	paramCeiling  = 7
	returnCeiling = 4
)

// reviewFindings enforces the budgets that would otherwise rely on review: chain
// branches, parameter count, and return count.
func reviewFindings(f *sourceFile) []finding {
	var out []finding
	for _, decl := range f.syntax.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		out = append(out, reviewFunction(f, funcName(fn), fn.Type, fn.Body)...)
	}
	ast.Inspect(f.syntax, func(node ast.Node) bool {
		lit, ok := node.(*ast.FuncLit)
		if !ok {
			return true
		}
		out = append(out, reviewFunction(f, "function literal", lit.Type, lit.Body)...)
		return true
	})
	return out
}

func reviewFunction(f *sourceFile, name string, typ *ast.FuncType, body *ast.BlockStmt) []finding {
	var out []finding
	line := f.fset.Position(typ.Pos()).Line
	if count := fieldCount(typ.Params); count > paramCeiling {
		out = append(out, reviewFinding(f, line, fmt.Sprintf(
			"%s has %d parameters, over the %d ceiling", name, count, paramCeiling,
		)))
	}
	if count := fieldCount(typ.Results); count > returnCeiling {
		out = append(out, reviewFinding(f, line, fmt.Sprintf(
			"%s returns %d values, over the %d ceiling", name, count, returnCeiling,
		)))
	}
	return append(out, branchFindings(f, name, body)...)
}

func fieldCount(fields *ast.FieldList) int {
	if fields == nil {
		return 0
	}
	count := 0
	for _, field := range fields.List {
		if len(field.Names) == 0 {
			count++
			continue
		}
		count += len(field.Names)
	}
	return count
}

func branchFindings(f *sourceFile, name string, body *ast.BlockStmt) []finding {
	elseIf := map[*ast.IfStmt]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		if stmt, ok := node.(*ast.IfStmt); ok {
			for next, ok := stmt.Else.(*ast.IfStmt); ok; next, ok = next.Else.(*ast.IfStmt) {
				elseIf[next] = true
			}
		}
		return true
	})

	var out []finding
	ast.Inspect(body, func(node ast.Node) bool {
		if _, ok := node.(*ast.FuncLit); ok {
			return false
		}
		stmt, ok := node.(*ast.IfStmt)
		if !ok || elseIf[stmt] {
			return true
		}
		if count := ifChainLength(stmt); count > branchCeiling {
			line := f.fset.Position(stmt.Pos()).Line
			out = append(out, reviewFinding(f, line, fmt.Sprintf(
				"%s has %d branches in an if/else-if chain, over the %d ceiling",
				name, count, branchCeiling,
			)))
		}
		return true
	})
	return out
}

func ifChainLength(stmt *ast.IfStmt) int {
	count := 1
	for stmt.Else != nil {
		count++
		next, ok := stmt.Else.(*ast.IfStmt)
		if !ok {
			return count
		}
		stmt = next
	}
	return count
}

func reviewFinding(f *sourceFile, line int, message string) finding {
	return finding{path: f.path, line: line, module: f.module, message: message}
}
