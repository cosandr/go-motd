package datasources

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/cosandr/go-motd/utils"
	"github.com/shirou/gopsutil/v3/host"
	log "github.com/sirupsen/logrus"
)

// ConfTempCPU extends ConfBase with a list of containers to ignore
type ConfTempCPU struct {
	ConfBaseWarn `yaml:",inline"`
	// Get CPU temperatures by parsing 'sensors -j' output
	Exec bool `yaml:"use_exec"`
}

// Init sets up default alignment
func (c *ConfTempCPU) Init() {
	c.ConfBaseWarn.Init()
	c.PadHeader[1] = 1
}

// GetCPUTemp returns CPU core temps using gopsutil or parsing sensors output
func GetCPUTemp(ret chan<- string, c *ConfTempCPU) {
	var header string
	var content string
	var tempMap map[string]int
	var isZen bool
	var err error
	if c.Exec {
		tempMap, isZen, err = cpuTempSensors()
	} else {
		tempMap, isZen, err = cpuTempGopsutil()
	}
	if err != nil {
		log.Warnf("[cpu] temperature read error: %v", err)
	}
	if len(tempMap) == 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("CPU temp", padL, padR), utils.Warn("unavailable"))
	} else {
		header, content, _ = formatCPUTemps(tempMap, isZen, c.Warn, c.Crit, *c.WarnOnly)
	}
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

func formatCPUTemps(tempMap map[string]int, isZen bool, warnTemp int, critTemp int, warnOnly bool) (header string, content string, err error) {
	// Sort keys
	sortedNames := make([]string, len(tempMap))
	i := 0
	for k := range tempMap {
		sortedNames[i] = k
		i++
	}
	sort.Strings(sortedNames)
	var warnCount int
	var errCount int
	for _, k := range sortedNames {
		v := tempMap[k]
		var wrapped string
		if !isZen {
			wrapped = utils.Wrap(fmt.Sprintf("Core %s", k), padL, padR)
		} else {
			wrapped = utils.Wrap(k, padL, padR)
		}
		if v < warnTemp && !warnOnly {
			content += fmt.Sprintf("%s: %s\n", wrapped, utils.Good(v))
		} else if v >= warnTemp && v < critTemp {
			content += fmt.Sprintf("%s: %s\n", wrapped, utils.Warn(v))
			warnCount++
		} else if v >= critTemp {
			warnCount++
			errCount++
			content += fmt.Sprintf("%s: %s\n", wrapped, utils.Err(v))
		}
	}
	if warnCount == 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("CPU temp", padL, padR), utils.Good("OK"))
	} else if errCount > 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("CPU temp", padL, padR), utils.Err("Critical"))
	} else if warnCount > 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("CPU temp", padL, padR), utils.Warn("Warning"))
	}
	return
}

func cpuTempGopsutil() (tempMap map[string]int, isZen bool, err error) {
	temps, err := host.SensorsTemperatures()
	tempMap = make(map[string]int)
	addTemp := func(re *regexp.Regexp) {
		for _, stat := range temps {
			log.Debugf("[cpu] check %s", stat.SensorKey)
			m := re.FindStringSubmatch(stat.SensorKey)
			if len(m) > 1 {
				log.Debugf("[cpu] OK %s: %.0f", stat.SensorKey, stat.Temperature)
				tempMap[m[1]] = int(stat.Temperature)
			}
		}
	}
	addTemp(regexp.MustCompile(`coretemp_core(?:_)?(\d+)`))
	// Try k10temp if we didn't find anything
	if len(tempMap) == 0 {
		isZen = true
		log.Debug("[cpu] trying k10temp")
		addTemp(regexp.MustCompile(`k10temp_(\w+)`))
	}
	// Something's really wrong if we still have none
	if len(tempMap) == 0 {
		log.Warn("[cpu] could not find any CPU temperatures")
	} else {
		err = nil
	}
	return
}
