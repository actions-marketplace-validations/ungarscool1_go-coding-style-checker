// Command go-coding-style-checker enforces the Go style budgets golangci-lint
// cannot express: line width with its exemptions, file length, nesting depth,
// and branch, parameter, and return counts.
//
// Depth is enforced outright. The other budgets may carry a checked-in backlog
// whose numbers only change through -update.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const defaultBaseline = "stylecheck-baseline.txt"

func main() {
	root := flag.String("root", ".", "repository root to scan")
	baselinePath := flag.String("baseline", "",
		"backlog file to enforce (default <root>/"+defaultBaseline+")")
	exclude := flag.String("exclude", "",
		"comma-separated directories to skip, relative to the root")
	update := flag.Bool("update", false, "rewrite the backlog from the current tree")
	flag.Parse()

	if *baselinePath == "" {
		*baselinePath = filepath.Join(*root, defaultBaseline)
	}
	err := run(*root, *baselinePath, excludedDirs(*exclude), *update, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "stylecheck:", err)
		os.Exit(1)
	}
}
