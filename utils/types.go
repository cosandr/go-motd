package utils

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
)

// Pad holds the pad char and its number of spaces a map as well as the string itself
type Pad struct {
	Delims  map[string]int
	Content string
}

// Do performs the padding
func (p *Pad) Do() string {
	var buf bytes.Buffer
	var w *tabwriter.Writer
	var withTabs = p.Content
	// Replace padchar with tabs
	for k, v := range p.Delims {
		w = tabwriter.NewWriter(&buf, 0, 0, v, ' ', 0)
		withTabs = strings.ReplaceAll(withTabs, k, "\t")
		fmt.Fprint(w, withTabs)
		w.Flush()
		withTabs = buf.String()
		buf.Reset()
	}
	return strings.TrimSuffix(withTabs, "\n")
}

// StringSet a set for strings, useful for keeping track of elements
type StringSet map[string]struct{}

// Contains returns true if `v` is in the set
func (s StringSet) Contains(v string) bool {
	_, ok := s[v]
	return ok
}

// FromList returns a `StringSet` with the input list's contents
func (s StringSet) FromList(listIn []string) StringSet {
	var empty struct{}
	p := make(StringSet)
	for _, val := range listIn {
		p[val] = empty
	}
	return p
}
