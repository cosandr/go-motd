package temps

import (
	"fmt"
	"github.com/cosandr/go-motd/colors"
	mt "github.com/cosandr/go-motd/types"
	"github.com/shirou/gopsutil/host"
	"regexp"
	"sort"
)

const (
	padL = "$"
	padR = "%"
)

// GetCPUTemp returns CPU core temps using gopsutil
func GetCPUTemp(ret *string, c *mt.CommonWithWarn) {
	header, content, _ := cpuTempGopsutil(c.Warn, c.Crit, *c.FailedOnly)
	// Pad header
	var p = mt.Pad{Delims: map[string]int{padL: c.Header[0], padR: c.Header[1]}, Content: header}
	header = p.Do()
	if len(content) == 0 {
		*ret = header
		return
	}
	// Pad container list
	p = mt.Pad{Delims: map[string]int{padL: c.Content[0], padR: c.Content[1]}, Content: content}
	content = p.Do()
	*ret = header + "\n" + content
}

func cpuTempGopsutil(warnTemp int, critTemp int, warnOnly bool) (header string, content string, err error) {
	temps, err := host.SensorsTemperatures()
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", mt.Wrap("CPU temp", padL, padR), colors.Warn("unavailable"))
		return
	}
	reCore := regexp.MustCompile(`coretemp\_core(\d+)\_input`)
	var tempMap = make(map[string]int)
	var sortedCPUs []string
	for _, stat := range temps {
		m := reCore.FindStringSubmatch(stat.SensorKey)
		if len(m) > 1 {
			tempMap[m[1]] = int(stat.Temperature)
			sortedCPUs = append(sortedCPUs, m[1])
		}
	}
	sort.Strings(sortedCPUs)
	var warnCount int = 0
	var errCount int = 0
	for _, k := range sortedCPUs {
		var v = tempMap[k]
		var wrapped = mt.Wrap(fmt.Sprintf("Core %s", k), padL, padR)
		if v < warnTemp && !warnOnly {
			content += fmt.Sprintf("%s: %s\n", wrapped, colors.Good(v))
		} else if v >= warnTemp && v < critTemp {
			content += fmt.Sprintf("%s: %s\n", wrapped, colors.Warn(v))
			warnCount++
		} else if v >= critTemp {
			warnCount++
			errCount++
			content += fmt.Sprintf("%s: %s\n", wrapped, colors.Err(v))
		}
	}
	if warnCount == 0 {
		header = fmt.Sprintf("%s: %s\n", mt.Wrap("CPU temp", padL, padR), colors.Good("OK"))
	} else if errCount > 0 {
		header = fmt.Sprintf("%s: %s\n", mt.Wrap("CPU temp", padL, padR), colors.Err("Critical"))
	} else if warnCount > 0 {
		header = fmt.Sprintf("%s: %s\n", mt.Wrap("CPU temp", padL, padR), colors.Warn("Warning"))
	}
	return
}