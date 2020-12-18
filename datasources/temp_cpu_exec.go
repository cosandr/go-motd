package datasources

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"regexp"

	log "github.com/sirupsen/logrus"
)

func cpuTempSensors() (tempMap map[string]int, isZen bool, err error) {
	var buf bytes.Buffer
	cmd := exec.Command("sensors", "-j")
	cmd.Stdout = &buf
	err = cmd.Run()
	if err != nil {
		return
	}
	var result map[string]interface{}
	err = json.Unmarshal(buf.Bytes(), &result)
	if err != nil {
		log.Warnf("[cpu] sensors parse failed: %v", err)
		return
	}

	tempMap = make(map[string]int)
	reTemp := regexp.MustCompile(`temp\d+_input`)
	addTemp := func(reModule *regexp.Regexp, reName *regexp.Regexp) {
		for k, v := range result {
			// Intel: Found all core temps "coretemp-isa-0000"
			// AMD: Found k10temp "k10temp-pci-00c3"
			if m := reModule.FindStringIndex(k); len(m) > 0 {
				for core, temps := range v.(map[string]interface{}) {
					// Intel: Found core temps ("Core 0")
					// AMD: Found tctl, tdie or tccd
					if mc := reName.FindStringSubmatch(core); len(mc) > 0 {
						for kk, temp := range temps.(map[string]interface{}) {
							// Found correct temperature value ("temp2_input")
							if mt := reTemp.FindStringIndex(kk); len(mt) > 0 {
								tempMap[mc[1]] = int(temp.(float64))
							}
						}
					}
				}
				break
			}
		}
	}
	// Try Intel
	addTemp(regexp.MustCompile(`coretemp\S*`), regexp.MustCompile(`Core\s(\d+)`))

	// Try k10temp if we didn't find anything
	if len(tempMap) == 0 {
		isZen = true
		log.Debug("[cpu] trying k10temp")
		addTemp(regexp.MustCompile(`k10temp\S*`), regexp.MustCompile(`(?i)(tctl|tdie|tccd\d+)`))
	}
	// Something's really wrong if we still have none
	if len(tempMap) == 0 {
		log.Warn("[cpu] could not find any CPU temperatures")
	} else {
		err = nil
	}
	return
}
