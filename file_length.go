package main

// fileLineCeiling applies to non-test files only: a 3000-line table-driven
// service_test.go is fine.
const fileLineCeiling = 600

// oversizeFiles reports the non-test files past the ceiling, keyed by path,
// with the line count each one carries today.
func oversizeFiles(files []*sourceFile) map[string]int {
	over := map[string]int{}
	for _, f := range files {
		if f.isTest() || len(f.lines) <= fileLineCeiling {
			continue
		}
		over[f.path] = len(f.lines)
	}
	return over
}
