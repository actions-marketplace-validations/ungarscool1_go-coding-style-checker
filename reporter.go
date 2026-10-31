package main

import (
	"fmt"
	"io"
)

// reporter is where stylecheck's own output goes. Nothing downstream can act
// on a failed write to stdout, so the error is dropped once, here, instead of
// at every call site.
type reporter struct {
	out io.Writer
}

func (r reporter) printf(format string, args ...any) {
	_, _ = fmt.Fprintf(r.out, format, args...)
}

func (r reporter) println(line string) {
	_, _ = fmt.Fprintln(r.out, line)
}
