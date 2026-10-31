package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func measurementOf(width, files map[string]int) *measurement {
	return &measurement{width: width, files: files, wide: map[string][]finding{}}
}

func TestCompareAcceptsAnExactMatch(t *testing.T) {
	current := measurementOf(map[string]int{"modules/api": 3}, map[string]int{"a.go": 700})
	previous := &baseline{
		width: map[string]int{"modules/api": 3},
		files: map[string]int{"a.go": 700},
	}
	if got := compare(current, previous); len(got) != 0 {
		t.Fatalf("compare = %v, want no problems", got)
	}
}

func TestCompareRejectsRegressions(t *testing.T) {
	tests := []struct {
		name    string
		current *measurement
		want    string
	}{
		{
			name:    "more wide lines than the backlog allows",
			current: measurementOf(map[string]int{"modules/api": 4}, nil),
			want:    "backlog allows 3",
		},
		{
			name:    "a module with no backlog entry",
			current: measurementOf(map[string]int{"modules/core": 1}, nil),
			want:    "modules/core: 1 non-exempt lines",
		},
		{
			name:    "a new file over the ceiling",
			current: measurementOf(nil, map[string]int{"new.go": 900}),
			want:    "new.go: 900 lines, over the 600 ceiling",
		},
		{
			name:    "a backlogged file that grew",
			current: measurementOf(nil, map[string]int{"a.go": 800}),
			want:    "a.go: grew to 800 lines",
		},
	}

	previous := &baseline{
		width: map[string]int{"modules/api": 3},
		files: map[string]int{"a.go": 700},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := strings.Join(compare(tc.current, previous), "\n")
			if !strings.Contains(got, tc.want) {
				t.Fatalf("compare said:\n%s\nwant it to mention %q", got, tc.want)
			}
		})
	}
}

func TestCompareRejectsStaleEntries(t *testing.T) {
	previous := &baseline{
		width: map[string]int{"modules/api": 3},
		files: map[string]int{"a.go": 700},
	}
	current := measurementOf(map[string]int{"modules/api": 1}, map[string]int{})
	got := strings.Join(compare(current, previous), "\n")
	for _, want := range []string{"down to 1 wide lines", "a.go is under budget or gone"} {
		if !strings.Contains(got, want) {
			t.Fatalf("compare said:\n%s\nwant it to mention %q", got, want)
		}
	}
}

func TestCompareReportsDepthOutright(t *testing.T) {
	current := measurementOf(nil, nil)
	current.depth = []finding{{path: "a.go", line: 12, message: "f nests 5 levels deep"}}
	got := strings.Join(compare(current, &baseline{}), "\n")
	if !strings.Contains(got, "a.go:12: f nests 5 levels deep") {
		t.Fatalf("compare said:\n%s", got)
	}
}

func TestCompareTracksReviewBudgetDebt(t *testing.T) {
	current := measurementOf(nil, nil)
	current.review = []finding{{
		path:    "a.go",
		line:    12,
		message: "f has 8 parameters, over the 7 ceiling",
	}}
	previous := &baseline{review: map[string]int{"a.go": 1}}
	if got := compare(current, previous); len(got) != 0 {
		t.Fatalf("compare = %v, want no problems", got)
	}

	current.review = append(current.review, finding{path: "a.go", line: 20})
	got := strings.Join(compare(current, previous), "\n")
	if !strings.Contains(got, "a.go: 2 review-budget findings, backlog allows 1") {
		t.Fatalf("compare said:\n%s", got)
	}
}

func writeWideFile(t *testing.T, path string) {
	t.Helper()
	src := "package wide\n\nfunc F() int { return " + strings.Repeat("1+", 60) + "1 }\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestRunRecordsDebtInARepositoryWithoutAGoMod(t *testing.T) {
	root := t.TempDir()
	writeWideFile(t, filepath.Join(root, "wide.go"))
	backlog := filepath.Join(root, "backlog.txt")
	none := map[string]bool{}

	if err := run(root, backlog, none, false, io.Discard); err == nil {
		t.Fatal("run passed with undeclared debt and no backlog file")
	}
	if err := run(root, backlog, none, true, io.Discard); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := run(root, backlog, none, false, io.Discard); err != nil {
		t.Fatalf("run against the recorded backlog: %v", err)
	}
}

func TestRunPassesACleanTreeWithoutABacklogFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.txt")
	if err := run(t.TempDir(), missing, map[string]bool{}, false, io.Discard); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func TestRunSkipsExcludedDirectoriesByRootRelativePath(t *testing.T) {
	root := t.TempDir()
	writeWideFile(t, filepath.Join(root, "gen", "api", "wide.go"))
	writeWideFile(t, filepath.Join(root, "testdata", "wide.go"))
	backlog := filepath.Join(root, "backlog.txt")

	excluded := excludedDirs("gen/api/, other")
	if err := run(root, backlog, excluded, false, io.Discard); err != nil {
		t.Fatalf("run with gen/api excluded: %v", err)
	}
	if err := run(root, backlog, excludedDirs("api"), false, io.Discard); err == nil {
		t.Fatal("a bare directory name excluded gen/api; want root-relative matching")
	}
}
