package temps

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"

	"github.com/cosandr/go-motd/colors"
	mt "github.com/cosandr/go-motd/types"
)

// GetCPUTempSensors returns CPU core temps by parsing sensors command output
func GetCPUTempSensors(ret chan<- string, c *mt.CommonWithWarn) {
	header, content, _ := cpuTempSensors(c.Warn, c.Crit, *c.FailedOnly)
	// Pad header
	var p = mt.Pad{Delims: map[string]int{padL: c.Header[0], padR: c.Header[1]}, Content: header}
	header = p.Do()
	if len(content) == 0 {
		ret <- header
		return
	}
	// Pad container list
	p = mt.Pad{Delims: map[string]int{padL: c.Content[0], padR: c.Content[1]}, Content: content}
	content = p.Do()
	ret <- header + "\n" + content
}

func cpuTempSensors(warnTemp int, critTemp int, warnOnly bool) (header string, content string, err error) {
	var buf bytes.Buffer
	cmd := exec.Command("sensors", "-j")
	cmd.Stdout = &buf
	err = cmd.Run()
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", mt.Wrap("CPU temp", padL, padR), colors.Warn("unavailable"))
		return
	}
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", mt.Wrap("CPU temp", padL, padR), colors.Warn("sensors parse failed"))
		return
	}

	reAllCores := regexp.MustCompile(`coretemp\S*`)
	reCore := regexp.MustCompile(`Core\s(\d+)`)
	reTemp := regexp.MustCompile(`temp\d+\_input`)

	var tempMap = make(map[int]int)
	var sortedCPUs []int

	for k, v := range result {
		// Found all core temps ("coretemp-isa-0000")
		if m := reAllCores.FindStringIndex(k); len(m) > 0 {
			for core, temps := range v.(map[string]interface{}) {
				// Found core temps ("Core 0")
				if mc := reCore.FindStringSubmatch(core); len(mc) > 0 {
					for kk, temp := range temps.(map[string]interface{}) {
						// Found correct temperature value ("temp2_input")
						if mt := reTemp.FindStringIndex(kk); len(mt) > 0 {
							coreNum, _ := strconv.Atoi(mc[1])
							tempMap[coreNum] = int(temp.(float64))
							sortedCPUs = append(sortedCPUs, coreNum)
						}
					}
				}
			}
			break
		}
	}
	sort.Ints(sortedCPUs)
	var warnCount int
	var errCount int
	for _, k := range sortedCPUs {
		var v = tempMap[k]
		var wrapped = mt.Wrap(fmt.Sprintf("Core %d", k), padL, padR)
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
