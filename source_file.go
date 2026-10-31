package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// sourceFile is one parsed Go file, kept with the text the checks measure and
// with the module that owns it.
type sourceFile struct {
	path   string
	module string
	lines  []string
	fset   *token.FileSet
	syntax *ast.File
}

func (f *sourceFile) isTest() bool {
	return strings.HasSuffix(f.path, "_test.go")
}

// skippedDir reports trees that hold no first-party Go, plus the directories the
// caller excluded. testdata fixtures may not even parse.
func skippedDir(rel string, excluded map[string]bool) bool {
	switch filepath.Base(rel) {
	case ".git", "node_modules", "testdata", "vendor":
		return true
	}
	return excluded[rel]
}

// excludedDirs parses a comma- or whitespace-separated list of root-relative
// directories.
func excludedDirs(list string) map[string]bool {
	excluded := map[string]bool{}
	separator := func(r rune) bool { return r == ',' || unicode.IsSpace(r) }
	for _, dir := range strings.FieldsFunc(list, separator) {
		excluded[filepath.ToSlash(filepath.Clean(dir))] = true
	}
	return excluded
}

func survey(root string, excluded map[string]bool) ([]*sourceFile, error) {
	modules := map[string]string{}
	var files []*sourceFile

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && skippedDir(relative(root, path), excluded) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		parsed, err := parseFile(root, path, modules)
		if err != nil {
			return err
		}
		files = append(files, parsed)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func parseFile(root, path string, modules map[string]string) (*sourceFile, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	syntax, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return &sourceFile{
		path:   relative(root, path),
		module: moduleOf(root, filepath.Dir(path), modules),
		lines:  textLines(string(src)),
		fset:   fset,
		syntax: syntax,
	}, nil
}

// textLines drops the empty element a trailing newline produces, so len() is
// the line count `wc -l` reports.
func textLines(src string) []string {
	lines := strings.Split(src, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// moduleOf reports the nearest ancestor directory holding a go.mod. Files outside
// every module belong to the root, so each backlog entry keeps a name.
func moduleOf(root, dir string, cache map[string]string) string {
	if known, ok := cache[dir]; ok {
		return known
	}
	owner := "."
	switch _, err := os.Stat(filepath.Join(dir, "go.mod")); {
	case err == nil:
		owner = relative(root, dir)
	case dir != filepath.Clean(root):
		owner = moduleOf(root, filepath.Dir(dir), cache)
	}
	cache[dir] = owner
	return owner
}

func relative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
