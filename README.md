# go-coding-style-checker

Enforces the Go style budgets that `golangci-lint` cannot express. Each budget
is a number, so a human or an LLM gets a pass or a fail rather than an opinion.

## Usage

### GitHub Action

```yaml
- uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
  with:
    persist-credentials: false
- uses: ungarscool1/go-coding-style-checker@main # pin to a commit SHA
  with:
    exclude: web,internal/generated
```

| Input | Default | Meaning |
| --- | --- | --- |
| `root` | `.` | Repository root to scan |
| `baseline` | `<root>/stylecheck-baseline.txt` | Backlog file to enforce |
| `exclude` | | Comma-separated directories to skip, relative to `root` |
| `go-version` | `stable` | Go used to build and parse. Empty uses the Go already on `PATH` |

A non-empty `go-version` runs `actions/setup-go`, which also changes the Go seen
by later steps of the job. If your job has already set up Go and needs that
version afterwards, pass `go-version: ''`.

### Command line

```sh
go run github.com/ungarscool1/go-coding-style-checker@latest
go run github.com/ungarscool1/go-coding-style-checker@latest -update  # regenerate the backlog
```

| Flag | Default | Meaning |
| --- | --- | --- |
| `-root` | `.` | Repository root to scan |
| `-baseline` | `<root>/stylecheck-baseline.txt` | Backlog file to enforce |
| `-exclude` | | Comma-separated directories to skip, relative to the root |
| `-update` | `false` | Rewrite the backlog from the current tree |

`.git`, `node_modules`, `testdata`, and `vendor` directories are always skipped.

## What it checks

| Budget | Ceiling | Why not a linter |
| --- | --- | --- |
| Nesting depth | 4 | `nestif` scores complexity, not depth. |
| Line width | 100 columns, tab = 4 | `lll` cannot express the exemptions below. |
| File lines (non-test) | 600 | Nothing measures a file. |
| `if`/`else if`/`else` chain branches | 3 | No linter counts one chain as one construct. |
| Function parameters | 7 | Needs a hard ceiling, not a score. |
| Return values | 4 | Needs a hard ceiling, not a score. |

The function body is level 0 for nesting, and each block-introducing statement
adds one. An `else if` chain stays at the level of its first `if`.

### Line width exemptions

Test files are not measured for width or file length. Everywhere else, a line
over 100 columns is exempt only if it is one of these:

- an import path
- part of a `gofmt`-aligned declaration: a parenthesised `const`, `var`, or
  `type` group, or a struct or interface body
- a string literal that cannot be split, such as an SQL statement

A literal earns the exemption only by being what makes the line long: if
deleting it still leaves the line over 100 columns, the line is reported.

Function signatures and comments are not exempt. A trailing comment on a
`gofmt`-aligned struct field is, because the field claims the line first.

## The backlog

`stylecheck-baseline.txt` records debt that predates enforcement: per module for
line width, and per file for file length and the other budgets. Nesting depth
has no backlog. A project with no debt does not need the file.

The check fails when a number rises and when one is *stale*, and the fix in both
directions is to regenerate with `-update`. A number may go up, but not quietly:
the baseline diff says which file grew and by how much.

An entry is where the next cleanup starts, not permission to add more.
