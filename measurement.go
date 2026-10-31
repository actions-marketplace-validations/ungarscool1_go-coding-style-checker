package main

// finding is one place in the tree that misses a budget.
type finding struct {
	path    string
	line    int
	module  string
	message string
}

// measurement is what the tree measures today, in the shape the backlog stores.
type measurement struct {
	width  map[string]int
	files  map[string]int
	depth  []finding
	review []finding
	wide   map[string][]finding
}

func measure(files []*sourceFile) *measurement {
	m := &measurement{
		width: map[string]int{},
		files: oversizeFiles(files),
		wide:  map[string][]finding{},
	}
	for _, f := range files {
		m.depth = append(m.depth, deepFunctions(f)...)
		m.review = append(m.review, reviewFindings(f)...)
		for _, wide := range overWideLines(f) {
			m.width[wide.module]++
			m.wide[wide.module] = append(m.wide[wide.module], wide)
		}
	}
	return m
}

func (m *measurement) baseline() *baseline {
	return &baseline{
		width:  m.width,
		files:  m.files,
		review: reviewCounts(m.review),
	}
}

func reviewCounts(findings []finding) map[string]int {
	counts := map[string]int{}
	for _, finding := range findings {
		counts[finding.path]++
	}
	return counts
}

func (m *measurement) wideTotal() int {
	total := 0
	for _, count := range m.width {
		total += count
	}
	return total
}
