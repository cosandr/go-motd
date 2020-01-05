package updates

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"time"

	"github.com/cosandr/go-check-updates/types"
	"github.com/cosandr/go-motd/colors"
	mt "github.com/cosandr/go-motd/types"
)

const (
	padL = "$"
	padR = "%"
)

// Conf extends Common with a show toggle (same as failedOnly), path to file and how often to check
type Conf struct {
	mt.Common `yaml:",inline"`
	Show *bool `yaml:"show"`
	File string `yaml:"file"`
	Check time.Duration `yaml:"check"`
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

// Get reads cached updates file and formats it
func Get(ret chan<- string, c *Conf) {
	header, content, _ := parseFile(c.File, c.Check, *c.Show)
	// Pad header
	var p = mt.Pad{Delims: map[string]int{padL: c.Common.Header[0], padR: c.Common.Header[1]}, Content: header}
	header = p.Do()
	if len(content) == 0 {
		ret <- header
		return
	}
	// Pad container list
	p = mt.Pad{Delims: map[string]int{padL: c.Common.Content[0], padR: c.Common.Content[1]}, Content: content}
	content = p.Do()
	ret <- header + "\n" + content
}

func parseFile(cacheFp string, checkDur time.Duration, show bool) (header string, content string, err error) {
	parsed, err := readUpdateCache(cacheFp)
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", mt.Wrap("Updates", padL, padR), colors.Warn("unavailable"))
		return
	}
	var timeElapsed = time.Since(parsed.Checked)
	// TODO: Run go-check-updates and return
	if timeElapsed > checkDur {
		// Run in bg
		// header = fmt.Sprintf("%s: %s\n", mt.Wrap("Updates", padL, padR), colors.Warn("checking"))
		// return
	}
	header = fmt.Sprintf("%s: %d pending, checked %s ago\n", mt.Wrap("Updates", padL, padR), len(parsed.Updates), timeStr(timeElapsed))
	if !show {
		return
	}
	for _, u := range parsed.Updates {
		content += fmt.Sprintf("%s -> %s\n", mt.Wrap(u.Pkg, padL, padR), u.NewVer)
	}
	return
}
