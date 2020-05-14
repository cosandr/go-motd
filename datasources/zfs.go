package datasources

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/cosandr/go-motd/utils"
)

// zpool list -Hpo name,alloc,size,health
// tank    6277009096704   11991548690432   ONLINE
// Sizes are in bytes

// GetZFS runs `zpool list -Ho name,alloc,size,health` and parses the output
func GetZFS(ret chan<- string, c *CommonWithWarnConf) {
	header, content, _ := getPoolStatus(c.Warn, c.Crit, *c.FailedOnly)
	// Pad header
	var p = utils.Pad{Delims: map[string]int{padL: c.Header[0], padR: c.Header[1]}, Content: header}
	header = p.Do()
	if len(content) == 0 {
		ret <- header
		return
	}
	// Pad container list
	p = utils.Pad{Delims: map[string]int{padL: c.Content[0], padR: c.Content[1]}, Content: content}
	content = p.Do()
	ret <- header + "\n" + content
}

func getPoolStatus(warnUsage int, critUsage int, warnOnly bool) (header string, content string, err error) {
	var buf bytes.Buffer
	cmd := exec.Command("zpool", "list", "-Hpo", "name,alloc,size,health")
	cmd.Stdout = &buf
	err = cmd.Run()
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("ZFS", padL, padR), utils.Warn("unavailable"))
		return
	}
	var status rune = 'o'
	for _, pool := range strings.Split(buf.String(), "\n") {
		var tmp = strings.Split(pool, "\t")
		if len(tmp) < 3 {
			continue
		}
		usedBytes, _ := strconv.ParseFloat(tmp[1], 64)
		totalBytes, _ := strconv.ParseFloat(tmp[2], 64)
		var usedStr = utils.FormatBytes(usedBytes)
		var totalStr = utils.FormatBytes(totalBytes)
		usedPerc := int((usedBytes / totalBytes) * 100)
		if tmp[3] != "ONLINE" {
			status = 'e'
			content += fmt.Sprintf("%s: %s, %s used out of %s\n", utils.Wrap(tmp[0], padL, padR), utils.Err(tmp[3]), usedStr, totalStr)
		} else if usedPerc < warnUsage && !warnOnly {
			content += fmt.Sprintf("%s: %s, %s used out of %s\n", utils.Wrap(tmp[0], padL, padR), utils.Good(tmp[3]), usedStr, totalStr)
		} else if usedPerc >= warnUsage && usedPerc < critUsage {
			if status != 'e' {
				status = 'w'
			}
			content += fmt.Sprintf("%s: %s, %s used out of %s\n", utils.Wrap(tmp[0], padL, padR), utils.Warn(tmp[3]), usedStr, totalStr)
		} else if usedPerc >= critUsage {
			status = 'e'
			content += fmt.Sprintf("%s: %s, %s used out of %s\n", utils.Wrap(tmp[0], padL, padR), utils.Err(tmp[3]), usedStr, totalStr)
		}
	}
	if status == 'o' {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("ZFS", padL, padR), utils.Good("OK"))
	} else if status == 'w' {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("ZFS", padL, padR), utils.Warn("Warning"))
	} else {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("ZFS", padL, padR), utils.Err("Critical"))
	}
	return
}
