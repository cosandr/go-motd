package temps

import (
	"fmt"
	"github.com/shirou/gopsutil/host"
	"regexp"
	"sort"

	"github.com/cosandr/go-motd/colors"
)

// GetCPUTemp returns CPU core temps using gopsutil
func GetCPUTemp(warnTemp int, critTemp int, warnOnly bool) (header string, content string, err error) {
	temps, err := host.SensorsTemperatures()
	if err != nil {
		header = "CPU temp\t: " + colors.Warn("unavailable")
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
		if v < warnTemp && !warnOnly {
			content += fmt.Sprintf("Core %s\t: %s\n", k, colors.Good(v))
		} else if v >= warnTemp && v < critTemp {
			content += fmt.Sprintf("Core %s\t: %s\n", k, colors.Warn(v))
			warnCount++
		} else if v >= critTemp {
			warnCount++
			errCount++
			content += fmt.Sprintf("Core %s\t: %s\n", k, colors.Err(v))
		}
	}
	if warnCount == 0 {
		header = fmt.Sprintf("CPU temp\t: %s\n", colors.Good("OK"))
	} else if errCount > 0 {
		header = fmt.Sprintf("CPU temp\t: %s\n", colors.Err("Critical"))
	} else if warnCount > 0 {
		header = fmt.Sprintf("CPU temp\t: %s\n", colors.Warn("Warning"))
	}
	return
}