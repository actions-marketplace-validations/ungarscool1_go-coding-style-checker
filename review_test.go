package main

import (
	"strings"
	"testing"
)

type reviewCase struct {
	name string
	src  string
	want []string
}

func reviewCases() []reviewCase {
	return []reviewCase{
		{
			name: "at every ceiling",
			src: "package p\n\nfunc f(a, b, c, d, e, f, g bool) (int, int, int, int) {\n" +
				"if a {\n} else if b {\n} else if c {\n}\nreturn 0, 0, 0, 0\n}\n",
		},
		{
			name: "reports every exceeded budget",
			src: "package p\n\nfunc f(a, b, c, d, e, f, g, h bool) (r1, r2, r3, r4, r5 int) {\n" +
				"if a {\n} else if b {\n} else if c {\n} else if d {\n}\nreturn\n}\n",
			want: []string{
				"f has 8 parameters",
				"f returns 5 values",
				"f has 4 branches in an if/else-if chain",
			},
		},
		{
			name: "counts a terminal else as a branch",
			src: "package p\n\nfunc f(a, b, c bool) {\n" +
				"if a {\n} else if b {\n} else if c {\n} else {\n}\n}\n",
			want: []string{"f has 4 branches"},
		},
		{
			name: "does not double report an else-if chain",
			src: "package p\n\nfunc f(a, b, c, d, e bool) {\n" +
				"if a {\n} else if b {\n} else if c {\n} else if d {\n} else if e {\n}\n}\n",
			want: []string{"f has 5 branches"},
		},
		{
			name: "does not attribute closure branches to the enclosing function",
			src: "package p\n\nfunc f() {\n_ = func(a, b, c, d bool) {\n" +
				"if a {\n} else if b {\n} else if c {\n} else if d {\n}\n}\n}\n",
			want: []string{"function literal has 4 branches"},
		},
		{
			name: "counts unnamed grouped and literal parameters",
			src: "package p\n\nvar f = func(a, b, c, d, e, f, g, h int) {}\n\n" +
				"func g(int, int, int, int, int, int, int, int) {}\n",
			want: []string{
				"function literal has 8 parameters",
				"g has 8 parameters",
			},
		},
	}
}

func TestReviewFindings(t *testing.T) {
	for _, tc := range reviewCases() {
		t.Run(tc.name, func(t *testing.T) {
			var messages []string
			for _, finding := range reviewFindings(sourceFrom(t, "p.go", tc.src)) {
				messages = append(messages, finding.message)
			}
			got := strings.Join(messages, "\n")
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Fatalf("findings = %q, want %q", got, want)
				}
			}
			if len(tc.want) == 0 && got != "" {
				t.Fatalf("findings = %q, want none", got)
			}
			if (tc.name == "does not double report an else-if chain" ||
				tc.name == "does not attribute closure branches to the enclosing function") &&
				len(messages) != 1 {
				t.Fatalf("findings = %v, want one chain finding", messages)
			}
		})
	}
}
