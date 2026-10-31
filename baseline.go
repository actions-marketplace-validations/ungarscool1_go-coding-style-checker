package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"
)

// baseline is the style debt a repository still carries. stylecheck fails when
// a number rises or goes stale, so the file cannot drift from the tree.
type baseline struct {
	width  map[string]int
	files  map[string]int
	review map[string]int
}

const (
	widthKind  = "width"
	fileKind   = "file"
	reviewKind = "review"
)

const baselineHeader = `# Style budget backlog.
#
#   width  <module> <lines>  non-exempt lines over 100 columns
#   file   <path> <lines>    non-test file over 600 lines
#   review <path> <findings> branch, parameter, or return budgets over their ceilings
#
# The check fails when a number rises and when one is stale, so any change to
# the debt shows up here. Regenerate this file by rerunning the check with -update.
#
# An entry is not a licence to add more. New code goes in under budget; the
# backlog exists so the cleanup can land module by module.
`

func newBaseline() *baseline {
	return &baseline{
		width:  map[string]int{},
		files:  map[string]int{},
		review: map[string]int{},
	}
}

// loadBaseline reads an empty backlog from a missing file, so a project with no
// debt needs no file at all.
func loadBaseline(path string) (*baseline, error) {
	text, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return newBaseline(), nil
	}
	if err != nil {
		return nil, err
	}
	loaded := newBaseline()
	for number, line := range strings.Split(string(text), "\n") {
		if err := loaded.readLine(line); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, number+1, err)
		}
	}
	return loaded, nil
}

func (b *baseline) readLine(line string) error {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return nil
	}
	fields := strings.Fields(trimmed)
	if len(fields) != 3 {
		return fmt.Errorf("want `<kind> <path> <count>`, got %q", trimmed)
	}
	count, err := strconv.Atoi(fields[2])
	if err != nil {
		return fmt.Errorf("count %q is not a number", fields[2])
	}
	switch fields[0] {
	case widthKind:
		b.width[fields[1]] = count
	case fileKind:
		b.files[fields[1]] = count
	case reviewKind:
		b.review[fields[1]] = count
	default:
		return fmt.Errorf("unknown kind %q", fields[0])
	}
	return nil
}

func (b *baseline) write(path string) error {
	return os.WriteFile(path, []byte(b.format()), 0o644)
}

func (b *baseline) format() string {
	var out strings.Builder
	out.WriteString(baselineHeader)
	writeSection(&out, widthKind, b.width)
	writeSection(&out, fileKind, b.files)
	writeSection(&out, reviewKind, b.review)
	return out.String()
}

func writeSection(out *strings.Builder, kind string, entries map[string]int) {
	if len(entries) == 0 {
		return
	}
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out.WriteString("\n")
	for _, key := range keys {
		fmt.Fprintf(out, "%s %s %d\n", kind, key, entries[key])
	}
}
