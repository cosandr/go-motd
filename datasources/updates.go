package datasources

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/cosandr/go-check-updates/types"
	"github.com/cosandr/go-motd/utils"
	"gopkg.in/yaml.v2"
)

// ConfUpdates extends ConfBase with a show toggle (same as warnOnly), path to file and how often to check
type ConfUpdates struct {
	ConfBase `yaml:",inline"`
	// Show packages that can be upgraded
	Show *bool `yaml:"show,omitempty"`
	// Path to go-check-updates cache file
	File string `yaml:"file"`
}

// Init sets default alignment and default cache file location
func (c *ConfUpdates) Init() {
	c.PadHeader = []int{0, 2}
	c.PadContent = []int{1, 0}
	c.File = "/tmp/go-check-updates.yaml"
}

// GetUpdates reads cached updates file and formats it
func GetUpdates(ret chan<- string, c *ConfUpdates) {
	header, content, _ := parseFile(c.File, *c.Show)
	// Pad header
	var p = utils.Pad{Delims: map[string]int{padL: c.PadHeader[0], padR: c.PadHeader[1]}, Content: header}
	header = p.Do()
	if len(content) == 0 {
		ret <- header
		return
	}
	// Pad container list
	p = utils.Pad{Delims: map[string]int{padL: c.PadContent[0], padR: c.PadContent[1]}, Content: content}
	content = p.Do()
	ret <- header + "\n" + content
}

func readUpdateCache(cacheFp string) (parsed types.YamlT, err error) {
	yamlFile, err := ioutil.ReadFile(cacheFp)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(yamlFile, &parsed)
	if err != nil {
		return
	}
	return
}

func timeStr(d time.Duration) string {
	if d.Hours() > 48 {
		return fmt.Sprintf("%.0f days", d.Hours()/24)
	}
	if d.Minutes() > 120 {
		return fmt.Sprintf("%.0f hours", d.Hours())
	}
	return fmt.Sprintf("%.0f minutes", d.Minutes())
}

func parseFile(cacheFp string, show bool) (header string, content string, err error) {
	parsed, err := readUpdateCache(cacheFp)
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Updates", padL, padR), utils.Warn("unavailable"))
		return
	}
	var timeElapsed = time.Since(parsed.Checked)
	header = fmt.Sprintf("%s: %d pending, checked %s ago\n", utils.Wrap("Updates", padL, padR), len(parsed.Updates), timeStr(timeElapsed))
	if !show {
		return
	}
	for _, u := range parsed.Updates {
		content += fmt.Sprintf("%s -> %s\n", utils.Wrap(u.Pkg, padL, padR), u.NewVer)
	}
	return
}
