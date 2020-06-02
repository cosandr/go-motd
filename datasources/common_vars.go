package datasources

import (
	"fmt"
	"strings"
	"time"
)

const (
	padL string = "^L^"
	padR string = "^R^"
)

// ConfInterface defines the interface for config structs
type ConfInterface interface {
	Init()
}

// ConfBase is the common type for all modules
//
// Custom modules should respect these options
type ConfBase struct {
	// Override global setting
	WarnOnly *bool `yaml:"warnings_only,omitempty"`
	// 2-element array defining padding for header (title)
	PadHeader []int `yaml:"pad_header,flow"`
	// 2-element array defining padding for content (details)
	PadContent []int `yaml:"pad_content,flow"`
}

// Init sets `PadHeader` and `PadContent` to [0, 0]
func (c *ConfBase) Init() {
	c.PadHeader = []int{0, 0}
	c.PadContent = []int{1, 0}
}

// ConfBaseWarn extends ConfBase with warning and critical values
type ConfBaseWarn struct {
	ConfBase `yaml:",inline"`
	Warn     int `yaml:"warn"`
	Crit     int `yaml:"crit"`
}

// Init sets warning to 70 and critical to 90
func (c *ConfBaseWarn) Init() {
	c.ConfBase.Init()
	c.Warn = 70
	c.Crit = 90
}

// timeStr returns human friendly time durations
func timeStr(d time.Duration, precision int, short bool) string {
	times := map[string]int{
		"year":   int(3.154e7),
		"month":  int(2.628e6),
		"week":   604800,
		"day":    86400,
		"hour":   3600,
		"minute": 60,
		"second": 1,
	}
	shortNames := map[string]string{
		"year":   "yr",
		"month":  "mo",
		"week":   "w",
		"day":    "d",
		"hour":   "h",
		"minute": "m",
		"second": "s",
	}
	seconds := int(d.Seconds())
	if seconds < 1 {
		return "just now"
	}
	var ret string
	var tmp int
	for name, val := range times {
		if tmp >= precision {
			break
		}
		q := seconds / val
		r := seconds % val
		// We have <1 of this unit
		if q == 0 {
			continue
		}
		if short {
			ret += fmt.Sprintf("%d%s", q, shortNames[name])
		} else {
			if q == 1 {
				// We have one, don't add s
				ret += fmt.Sprintf("%d %s, ", q, name)
			} else {
				// More than one or zero, add s at the end
				ret += fmt.Sprintf("%d %ss, ", q, name)
			}
		}
		seconds = r
		tmp++
	}
	return strings.TrimSuffix(ret, ", ")
}
