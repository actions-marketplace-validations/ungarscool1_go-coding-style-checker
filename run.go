package main

import (
	"errors"
	"fmt"
	"io"
	"sort"
)

// examplesPerModule caps how many offending lines a regression prints. The
// count is the finding; the examples are there to start from.
const examplesPerModule = 5

// updateHint closes a failed run. A number that moves is not forbidden — a
// number that moves without saying so is, which is why the fix is a baseline
// diff a reviewer can read rather than a flag that silences the check.
const updateHint = "if the change was deliberate, record it from the repository root:\n" +
	"    go run github.com/ungarscool1/go-coding-style-checker@latest -update"

func run(
	root, baselinePath string,
	excluded map[string]bool,
	update bool,
	out io.Writer,
) error {
	files, err := survey(root, excluded)
	if err != nil {
		return err
	}
	report := reporter{out: out}
	current := measure(files)
	summarize(report, current)

	if update {
		if err := current.baseline().write(baselinePath); err != nil {
			return err
		}
		report.printf("\nwrote %s\n", baselinePath)
		return nil
	}

	previous, err := loadBaseline(baselinePath)
	if err != nil {
		return err
	}
	problems := compare(current, previous)
	if len(problems) == 0 {
		report.println("\nstylecheck: ok")
		return nil
	}
	for _, problem := range problems {
		report.println(problem)
	}
	// Depth findings are printed first and have no backlog entry behind them,
	// so the remedy only applies when a backlogged budget failed.
	if len(problems) > len(current.depth) {
		report.println("\n" + updateHint)
	}
	return errors.New("style budgets regressed")
}

func summarize(report reporter, current *measurement) {
	report.printf("nesting depth over %d: %d function(s)\n",
		depthCeiling, len(current.depth))
	report.printf("review budgets over their ceilings: %d finding(s)\n",
		len(current.review))
	report.printf("non-exempt lines over %d columns: %d in %d module(s)\n",
		widthCeiling, current.wideTotal(), len(current.width))
	report.printf("non-test files over %d lines: %d\n",
		fileLineCeiling, len(current.files))
}

func compare(current *measurement, previous *baseline) []string {
	var problems []string
	for _, deep := range current.depth {
		problems = append(problems, fmt.Sprintf(
			"%s:%d: %s", deep.path, deep.line, deep.message))
	}
	problems = append(problems, compareReview(current, previous)...)
	problems = append(problems, compareWidth(current, previous)...)
	problems = append(problems, compareFiles(current, previous)...)
	return problems
}

func compareReview(current *measurement, previous *baseline) []string {
	var problems []string
	currentCounts := reviewCounts(current.review)
	for _, path := range union(currentCounts, previous.review) {
		now, before := currentCounts[path], previous.review[path]
		switch {
		case now > before:
			problems = append(problems, fmt.Sprintf(
				"%s: %d review-budget findings, backlog allows %d",
				path, now, before))
			problems = append(problems, examples(reviewFindingsAt(current.review, path))...)
		case now < before:
			problems = append(problems, staleEntry(fmt.Sprintf(
				"%s is down to %d review-budget findings from %d", path, now, before)))
		}
	}
	return problems
}

func reviewFindingsAt(findings []finding, path string) []finding {
	var atPath []finding
	for _, finding := range findings {
		if finding.path == path {
			atPath = append(atPath, finding)
		}
	}
	return atPath
}

func compareWidth(current *measurement, previous *baseline) []string {
	var problems []string
	for _, module := range union(current.width, previous.width) {
		now, before := current.width[module], previous.width[module]
		switch {
		case now > before:
			problems = append(problems, fmt.Sprintf(
				"%s: %d non-exempt lines over %d columns, backlog allows %d",
				module, now, widthCeiling, before))
			problems = append(problems, examples(current.wide[module])...)
		case now < before:
			problems = append(problems, staleEntry(fmt.Sprintf(
				"%s is down to %d wide lines from %d", module, now, before)))
		}
	}
	return problems
}

func compareFiles(current *measurement, previous *baseline) []string {
	var problems []string
	for _, path := range union(current.files, previous.files) {
		now, before := current.files[path], previous.files[path]
		switch {
		case before == 0:
			problems = append(problems, fmt.Sprintf(
				"%s: %d lines, over the %d ceiling", path, now, fileLineCeiling))
		case now == 0:
			problems = append(problems, staleEntry(path+" is under budget or gone"))
		case now > before:
			problems = append(problems, fmt.Sprintf(
				"%s: grew to %d lines, backlog allows %d", path, now, before))
		}
	}
	return problems
}

func examples(findings []finding) []string {
	if len(findings) > examplesPerModule {
		findings = findings[:examplesPerModule]
	}
	out := make([]string, 0, len(findings))
	for _, f := range findings {
		out = append(out, fmt.Sprintf("    %s:%d: %s", f.path, f.line, f.message))
	}
	return out
}

// staleEntry is a failure on purpose: an entry that outlives its debt is how a
// backlog stops being a backlog.
func staleEntry(what string) string {
	return "stale backlog entry: " + what + " — rerun with -update"
}

func union(left, right map[string]int) []string {
	seen := map[string]bool{}
	keys := make([]string, 0, len(left)+len(right))
	for _, side := range []map[string]int{left, right} {
		for key := range side {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	sort.Strings(keys)
	return keys
}
