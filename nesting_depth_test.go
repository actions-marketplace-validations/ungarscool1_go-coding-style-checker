package main

import "testing"

func TestDeepFunctions(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		found bool
	}{
		{
			name: "four levels is the ceiling",
			body: "if a {\n for range b {\n switch {\n case a:\n if a {\n }\n }\n }\n}",
		},
		{
			name:  "five levels is over",
			body:  "if a {\n for range b {\n switch {\n case a:\n if a {\n if a {\n }\n }\n }\n }\n}",
			found: true,
		},
		{
			name: "else-if chain stays at the depth of the first if",
			body: "if a {\n} else if a {\n} else if a {\n} else if a {\n} else if a {\n}",
		},
		{
			name:  "closure bodies count",
			body:  "if a {\n for range b {\n go func() {\n if a {\n if a {\n }\n }\n }()\n }\n}",
			found: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := "package p\n\nfunc f(a bool, b []int) {\n" + tc.body + "\n}\n"
			got := deepFunctions(sourceFrom(t, "p.go", src))
			if tc.found != (len(got) == 1) {
				t.Fatalf("deepFunctions = %v, want found=%v", got, tc.found)
			}
		})
	}
}

func TestDeepFunctionsNamesTheMethod(t *testing.T) {
	src := "package p\n\ntype T struct{}\n\nfunc (t *T) M(a bool) {\n" +
		"if a {\n if a {\n if a {\n if a {\n if a {\n }\n }\n }\n }\n}\n}\n"
	got := deepFunctions(sourceFrom(t, "p.go", src))
	if len(got) != 1 {
		t.Fatalf("got %d findings, want 1", len(got))
	}
	if want := "T.M nests 5 levels"; got[0].message[:len(want)] != want {
		t.Fatalf("message = %q, want prefix %q", got[0].message, want)
	}
}
