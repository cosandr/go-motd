package datasources

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"

	"github.com/cosandr/go-motd/utils"
)

func cpuTempSensors(warnTemp int, critTemp int, warnOnly bool) (header string, content string, err error) {
	var buf bytes.Buffer
	cmd := exec.Command("sensors", "-j")
	cmd.Stdout = &buf
	err = cmd.Run()
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("CPU temp", padL, padR), utils.Warn("unavailable"))
		return
	}
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("CPU temp", padL, padR), utils.Warn("sensors parse failed"))
		return
	}

	reAllCores := regexp.MustCompile(`coretemp\S*`)
	reCore := regexp.MustCompile(`Core\s(\d+)`)
	reTemp := regexp.MustCompile(`temp\d+_input`)

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
		var wrapped = utils.Wrap(fmt.Sprintf("Core %d", k), padL, padR)
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
