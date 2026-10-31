package main

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func sourceFrom(t *testing.T, name, src string) *sourceFile {
	t.Helper()
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	return &sourceFile{path: name, lines: textLines(src), fset: fset, syntax: syntax}
}

func wideLineNumbers(t *testing.T, name, src string) []int {
	t.Helper()
	var lines []int
	for _, f := range overWideLines(sourceFrom(t, name, src)) {
		lines = append(lines, f.line)
	}
	return lines
}

func padTo(prefix string, columns int) string {
	return prefix + strings.Repeat("x", columns-len(prefix))
}

// exemptionCase is one snippet and the lines overWideLines should report in
// it. An empty want means every over-long line in the snippet is exempt.
type exemptionCase struct {
	name string
	src  string
	want []int
}

func exemptionCases() []exemptionCase {
	return []exemptionCase{
		{
			name: "import path is exempt",
			src: "package p\n\nimport " +
				`"` + padTo("example.com/", 120) + `"` + "\n",
		},
		{
			name: "aligned const block is exempt",
			src:  "package p\n\nconst (\n\ta = 1 // " + padTo("", 120) + "\n)\n",
		},
		{
			name: "one-line declaration is not exempt",
			src:  "package p\n\ntype " + padTo("fn", 110) + " func()\n",
			want: []int{3},
		},
		{
			name: "struct body is exempt wherever it appears",
			src:  "package p\n\ntype T struct {\n\tA " + padTo("i", 110) + "\n}\n",
		},
		{
			name: "sql literal is exempt",
			src:  "package p\n\nfunc q() string {\n\treturn \"" + padTo("SELECT ", 130) + "\"\n}\n",
		},
		{
			name: "short literal on a long line is not exempt",
			src: "package p\n\nfunc q(a, b, c string) string {\n\treturn a + b + c + " +
				strings.Repeat("a + ", 25) + "\"x\"\n}\n",
			want: []int{4},
		},
		{
			name: "trailing comment on an aligned field is exempt",
			src:  "package p\n\ntype T struct {\n\tA int // " + padTo("", 120) + "\n}\n",
		},
		{
			name: "a comment of its own is not exempt",
			src:  "package p\n\n// " + padTo("", 120) + "\nfunc f() {\n}\n",
			want: []int{3},
		},
		{
			name: "signature is not exempt",
			src:  "package p\n\nfunc " + padTo("f", 110) + "() {\n}\n",
			want: []int{3},
		},
		{
			name: "plain code is not exempt",
			src:  "package p\n\nfunc f() {\n\t" + padTo("y", 120) + " := 1\n\t_ = " + padTo("y", 120) + "\n}\n",
			want: []int{4, 5},
		},
	}
}

func TestOverWideLinesExemptions(t *testing.T) {
	for _, tc := range exemptionCases() {
		t.Run(tc.name, func(t *testing.T) {
			got := wideLineNumbers(t, "p.go", tc.src)
			if len(got) != len(tc.want) {
				t.Fatalf("got lines %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got lines %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestOverWideLinesSkipsTests(t *testing.T) {
	src := "package p\n\nfunc f() {\n\t_ = \"\" + " + padTo("y", 120) + "\n}\n"
	if got := wideLineNumbers(t, "p_test.go", src); got != nil {
		t.Fatalf("test file reported %v, want none", got)
	}
}

func TestVisualWidthCountsTabAsFour(t *testing.T) {
	if got := visualWidth("\t\tab"); got != 10 {
		t.Fatalf("visualWidth = %d, want 10", got)
	}
}
