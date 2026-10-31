package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBaselineRoundTrip(t *testing.T) {
	written := &baseline{
		width:  map[string]int{"modules/api": 712, "modules/auth": 270},
		files:  map[string]int{"modules/auth/internal/model/model.go": 1372},
		review: map[string]int{"modules/api/internal/model/model.go": 2},
	}
	path := filepath.Join(t.TempDir(), "baseline.txt")
	if err := written.write(path); err != nil {
		t.Fatalf("write: %v", err)
	}

	read, err := loadBaseline(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if read.width["modules/api"] != 712 || read.width["modules/auth"] != 270 {
		t.Fatalf("width = %v, want the two written entries", read.width)
	}
	if read.files["modules/auth/internal/model/model.go"] != 1372 {
		t.Fatalf("files = %v, want the written entry", read.files)
	}
	if read.review["modules/api/internal/model/model.go"] != 2 {
		t.Fatalf("review = %v, want the written entry", read.review)
	}
}

func TestBaselineRejectsMalformedLines(t *testing.T) {
	tests := map[string]string{
		"unknown kind":  "depth modules/api 3",
		"missing count": "width modules/api",
		"count is text": "width modules/api many",
	}
	for name, line := range tests {
		t.Run(name, func(t *testing.T) {
			if err := newBaseline().readLine(line); err == nil {
				t.Fatalf("readLine(%q) accepted a malformed entry", line)
			}
		})
	}
}

func TestBaselineIgnoresCommentsAndBlanks(t *testing.T) {
	loaded := newBaseline()
	for _, line := range []string{"", "   ", "# a comment"} {
		if err := loaded.readLine(line); err != nil {
			t.Fatalf("readLine(%q): %v", line, err)
		}
	}
	if len(loaded.width)+len(loaded.files)+len(loaded.review) != 0 {
		t.Fatalf("loaded %v/%v/%v, want nothing",
			loaded.width, loaded.files, loaded.review)
	}
}

func TestBaselineFormatIsSorted(t *testing.T) {
	out := (&baseline{
		width:  map[string]int{"modules/worker": 1, "modules/billing": 2},
		files:  map[string]int{},
		review: map[string]int{},
	}).format()
	billing := strings.Index(out, "modules/billing")
	worker := strings.Index(out, "modules/worker")
	if billing < 0 || worker < 0 || billing > worker {
		t.Fatalf("entries are not sorted:\n%s", out)
	}
}
