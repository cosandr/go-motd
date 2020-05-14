package datasources

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cosandr/go-motd/utils"
)

// GetBtrfs gets btrfs filesystem used and total space by reading files in /sys
func GetBtrfs(ret chan<- string, c *CommonWithWarnConf) {
	header, content, _ := getBtrfsStatus(c.Warn, c.Crit, *c.FailedOnly)
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

func getBtrfsStatus(warnUsage int, critUsage int, warnOnly bool) (header string, content string, err error) {
	matches, err := filepath.Glob("/sys/fs/btrfs/*-*")
	if err != nil {
		return
	}
	var status = 'o'
	for _, fs := range matches {
		// Get FS label
		var label string
		c, _ := ioutil.ReadFile(filepath.Join(fs, "/label"))
		if c != nil {
			label = strings.TrimSpace(string(c))
		} else {
			label = "Unlabelled"
		}
		var usedBytes float64
		var totalBytes float64
		var usedPerc int
		usedStr := "N/A"
		totalStr := "N/A"
		// Add data, metadata and system together
		usedFiles, _ := filepath.Glob(fs + "/allocation/*/bytes_used")
		for _, file := range usedFiles {
			usedBytes += readFloatFile(file)
		}
		// Add all device sizes
		deviceFiles, _ := filepath.Glob(fs + "/devices/*/size")
		for _, file := range deviceFiles {
			// Size is in sectors, on Linux a sector is always 512 bytes
			totalBytes += readFloatFile(file) * 512
		}

		if usedBytes <= 0 || totalBytes <= 0 {
			content += fmt.Sprintf("%s: %s\n", utils.Wrap(label, padL, padR), utils.Err("read error"))
			continue
		}
		usedStr = utils.FormatBytes(usedBytes)
		totalStr = utils.FormatBytes(totalBytes)
		usedPerc = int((usedBytes / totalBytes) * 100)
		if usedPerc < warnUsage && !warnOnly {
			content += fmt.Sprintf("%s: %s used out of %s\n", utils.Wrap(label, padL, padR), usedStr, totalStr)
		} else if usedPerc >= warnUsage && usedPerc < critUsage {
			if status != 'e' {
				status = 'w'
			}
			content += fmt.Sprintf("%s: %s used out of %s\n", utils.Wrap(label, padL, padR), usedStr, totalStr)
		} else if usedPerc >= critUsage {
			status = 'e'
			content += fmt.Sprintf("%s: %s used out of %s\n", utils.Wrap(label, padL, padR), usedStr, totalStr)
		}
	}
	if status == 'o' {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("BTRFS", padL, padR), utils.Good("OK"))
	} else if status == 'w' {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("BTRFS", padL, padR), utils.Warn("Warning"))
	} else {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("BTRFS", padL, padR), utils.Err("Critical"))
	}
	return
}

func readFloatFile(file string) float64 {
	readBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return 0
	}
	parsedFloat, err := strconv.ParseFloat(strings.TrimSpace(string(readBytes)), 64)
	if err != nil {
		return 0
	}
	return parsedFloat
}
