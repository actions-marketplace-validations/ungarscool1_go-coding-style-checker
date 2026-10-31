package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

const (
	// widthCeiling is measured with each tab advancing to a tabColumns stop.
	widthCeiling = 100
	tabColumns   = 4
)

// exemption is why a line past the ceiling does not have to change: a
// gofmt-aligned declaration, an unsplittable string literal, or an import path.
type exemption int

// A comment is not on this list, because prose wraps. A trailing comment on an
// aligned struct field is still exempt: the field claims the line first.
const (
	notExempt exemption = iota
	exemptImport
	exemptAlignedDecl
	exemptStringLiteral
)

// overWideLines reports the non-exempt lines past the ceiling. Test files are
// excluded.
func overWideLines(f *sourceFile) []finding {
	if f.isTest() {
		return nil
	}
	exempt := classifyExemptions(f)

	var out []finding
	for i, text := range f.lines {
		columns := visualWidth(text)
		if columns <= widthCeiling || exempt[i+1] != notExempt {
			continue
		}
		out = append(out, finding{
			path:    f.path,
			line:    i + 1,
			module:  f.module,
			message: fmt.Sprintf("%d columns, over the %d ceiling", columns, widthCeiling),
		})
	}
	return out
}

// classifyExemptions maps a line number to the reason it is exempt. The first
// claim on a line wins, so the passes run most specific first.
func classifyExemptions(f *sourceFile) map[int]exemption {
	exempt := map[int]exemption{}
	claim := func(from, to token.Pos, why exemption) {
		start := f.fset.Position(from).Line
		end := f.fset.Position(to).Line
		for line := start; line <= end; line++ {
			if _, taken := exempt[line]; !taken {
				exempt[line] = why
			}
		}
	}

	// A long signature is not exempt. Claiming signatures first keeps a later
	// pass from exempting one that happens to hold a struct type.
	for _, decl := range f.syntax.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		claim(fn.Pos(), signatureEnd(fn), notExempt)
	}
	claimStringLiterals(f, claim)
	for _, decl := range f.syntax.Decls {
		general, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		claimDeclaration(general, claim)
	}
	// A struct or interface body written inside a function still aligns the
	// way gofmt aligns a declared one.
	ast.Inspect(f.syntax, func(node ast.Node) bool {
		switch typ := node.(type) {
		case *ast.StructType:
			claim(typ.Pos(), typ.End(), exemptAlignedDecl)
		case *ast.InterfaceType:
			claim(typ.Pos(), typ.End(), exemptAlignedDecl)
		}
		return true
	})
	return exempt
}

// claimStringLiterals exempts a literal only when it is what makes the line
// long: delete it, and a line still over the ceiling had other reasons to wrap.
// The test is per literal because a line often carries several.
func claimStringLiterals(f *sourceFile, claim func(from, to token.Pos, why exemption)) {
	ast.Inspect(f.syntax, func(node ast.Node) bool {
		lit, ok := node.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		from := f.fset.Position(lit.Pos())
		to := f.fset.Position(lit.End())
		if from.Line != to.Line {
			claim(lit.Pos(), lit.End(), exemptStringLiteral)
			return true
		}
		if visualWidth(f.lines[from.Line-1])-(to.Column-from.Column) > widthCeiling {
			return true
		}
		claim(lit.Pos(), lit.End(), exemptStringLiteral)
		return true
	})
}

// claimDeclaration exempts import paths and gofmt-aligned declarations. Alignment
// only happens inside a parenthesised group, so a one-line `var x = ...` is not.
func claimDeclaration(decl *ast.GenDecl, claim func(from, to token.Pos, why exemption)) {
	if decl.Tok == token.IMPORT {
		claim(decl.Pos(), decl.End(), exemptImport)
		return
	}
	if !decl.Lparen.IsValid() {
		return
	}
	claim(decl.Pos(), decl.End(), exemptAlignedDecl)
}

func signatureEnd(fn *ast.FuncDecl) token.Pos {
	if fn.Body != nil {
		return fn.Body.Lbrace
	}
	return fn.Type.End()
}

func visualWidth(text string) int {
	columns := 0
	for _, r := range text {
		if r == '\t' {
			columns += tabColumns - columns%tabColumns
			continue
		}
		columns++
	}
	return columns
}
