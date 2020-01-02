package zfs

import (
	"fmt"
	"bytes"
	"os/exec"
	"strconv"
	"strings"
	"github.com/cosandr/go-motd/colors"
)

const tebibyte float64 = 1099511627776
const gibibyte float64 = 1073741824

// zpool list -Hpo name,alloc,size,health
// tank    6277009096704   11991548690432   ONLINE
// Sizes are in bytes

// GetPoolStatus runs `zpool list -Ho name,alloc,size,health` and parses the output
func GetPoolStatus(warnUsage int, critUsage int, warnOnly bool) (header string, content string, err error) {
	var buf bytes.Buffer
	cmd := exec.Command("zpool", "list", "-Hpo", "name,alloc,size,health")
	cmd.Stdout = &buf
	err = cmd.Run()
	if err != nil {
		header = fmt.Sprintf("ZFS\t: %s\n", colors.Warn("unavailable"))
		return
	}
	var status rune = 'o'
	for _, pool := range strings.Split(buf.String(), "\n") {
		var tmp = strings.Split(pool, "\t")
		if len(tmp) < 3 { continue }
		usedBytes, _ := strconv.ParseFloat(tmp[1], 64)
		totalBytes, _ := strconv.ParseFloat(tmp[2], 64)
		var usedStr = fmt.Sprintf("%.2f GB", usedBytes/gibibyte)
		var totalStr = fmt.Sprintf("%.2f GB", totalBytes/gibibyte)
		if usedBytes > tebibyte {
			usedStr = fmt.Sprintf("%.2f TB", usedBytes/tebibyte)
		}
		if totalBytes > tebibyte {
			totalStr = fmt.Sprintf("%.2f TB", totalBytes/tebibyte)
		}
		usedPerc := int((usedBytes/totalBytes)*100)
		if tmp[3] != "ONLINE" {
			status = 'e'
			content += fmt.Sprintf("%s\t: %s, %s used out of %s\n", tmp[0], colors.Err(tmp[3]), usedStr, totalStr)
		} else if usedPerc < warnUsage && !warnOnly {
			content += fmt.Sprintf("%s\t: %s, %s used out of %s\n", tmp[0], colors.Good(tmp[3]), usedStr, totalStr)
		} else if usedPerc >= warnUsage && usedPerc < critUsage {
			if status != 'e' { status = 'w' }
			content += fmt.Sprintf("%s\t: %s, %s used out of %s\n", tmp[0], colors.Warn(tmp[3]), usedStr, totalStr)
		} else if usedPerc >= critUsage {
			status = 'e'
			content += fmt.Sprintf("%s\t: %s, %s used out of %s\n", tmp[0], colors.Err(tmp[3]), usedStr, totalStr)
		}
	}
	if status == 'o' {
		header = fmt.Sprintf("ZFS\t: %s\n", colors.Good("OK"))
	} else if status == 'w' {
		header = fmt.Sprintf("ZFS\t: %s\n", colors.Warn("Warning"))
	} else {
		header = fmt.Sprintf("ZFS\t: %s\n", colors.Err("Critical"))
	}
	return
}
