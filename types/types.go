package types

import (
	"text/tabwriter"
	"bytes"
	"strings"
	"fmt"
)

// Common is the common type for all modules
//
// Custom modules should respect these options
type Common struct {
	FailedOnly *bool `yaml:"failedOnly,omitempty"`
	Header []int `yaml:"header"`
	Content []int `yaml:"content"`
}

// Init sets `Header` and `Content` to [0, 0]
func (c *Common) Init() {
	var defPad = []int{0, 0}
	c.Content = defPad
	c.Header = defPad
}

// CommonWithWarn extends Common with warning and critical values
type CommonWithWarn struct {
	Common `yaml:",inline"`
	Warn int `yaml:"warn"`
	Crit int `yaml:"crit"`
}

// Pad holds the pad char and its number of spaces a map as well as the string itself
type Pad struct {
	Delims map[string]int
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

// Wrap `s` around start and end, returns <start>s<end>
func Wrap(s string, start string, end string) string {
	return fmt.Sprintf("%s%s%s", start, s, end)
}
