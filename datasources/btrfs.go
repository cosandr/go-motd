package datasources

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/cosandr/go-motd/utils"
	"github.com/shirou/gopsutil/v3/disk"
	log "github.com/sirupsen/logrus"
)

// ConfBtrfs is the configuration for btrfs data
type ConfBtrfs struct {
	ConfBaseWarn `yaml:",inline"`
	// Show free space instead of used space
	ShowFree bool `yaml:"show_free"`
	// Parse btrfs command output
	Exec bool `yaml:"use_exec"`
	// Run btrfs using sudo
	Sudo bool `yaml:"sudo"`
	// Override btrfs command, example `btrfs-us --raw`
	Command string `yaml:"btrfs_cmd"`
}

// Init sets up default alignment
func (c *ConfBtrfs) Init() {
	c.ConfBaseWarn.Init()
	c.PadHeader[1] = 4
}

// GetBtrfs gets btrfs filesystem used and total space by reading files in /sys
func GetBtrfs(ret chan<- string, c *ConfBtrfs) {
	var header string
	var content string
	if c.Exec {
		// Check if we are root
		runningUser, err := user.Current()
		if err == nil && runningUser.Uid == "0" {
			// Do not run sudo as root, there's no point
			c.Sudo = false
		}
		var cmd string
		if c.Command == "" {
			cmd = "btrfs filesystem usage --raw"
		} else {
			cmd = c.Command
		}
		if c.Sudo {
			cmd = "sudo " + cmd
		}
		header, content, _ = getBtrfsStatusExec(cmd, c.Warn, c.Crit, *c.WarnOnly, c.ShowFree)
	} else {
		header, content, _ = getBtrfsStatus(c.Warn, c.Crit, *c.WarnOnly, c.ShowFree)
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

func getBtrfsStatusExec(cmd string, warnUsage int, critUsage int, warnOnly bool, showFree bool) (header string, content string, err error) {
	// Find all btrfs mounts
	parts, err := disk.Partitions(false)
	if err != nil {
		return
	}
	checked := make(map[string]struct{})
	var empty struct{}
	var status = 'o'
	// Group 1: total device size in bytes
	reSize := regexp.MustCompile(`(?im)^\s+device\s+size:\s+(\d+)`)
	// Group 1: estimated free space in bytes
	reFree := regexp.MustCompile(`(?im)^\s+free\s+(?:\(estimated\))?:\s+(\d+)`)
	// Group 1: data ratio
	reRatio := regexp.MustCompile(`(?im)^\s+data\s+ratio:\s+(\S+)`)
	args := strings.Split(cmd, " ")
	for _, p := range parts {
		if p.Fstype != "btrfs" {
			continue
		}
		if _, ok := checked[p.Device]; !ok {
			checked[p.Device] = empty
			log.Debugf("btrfs: device %s mounted at %s", p.Device, p.Mountpoint)
			tmp := append(args, p.Mountpoint)
			c := exec.Command(tmp[0], tmp[1:]...)
			log.Debugf("btrfs: exec: '%s'", c.String())
			var buf bytes.Buffer
			c.Stdout = &buf
			cErr := c.Run()
			if cErr != nil {
				log.Warnf("btrfs: cannot get usage for %s: %v", p.Mountpoint, cErr)
				continue
			}
			stdout := buf.String()
			log.Debugf("btrfs: output\n%s", stdout)
			mSize := reSize.FindStringSubmatch(stdout)
			if len(mSize) != 2 {
				log.Debugf("btrfs: mSize: %v", mSize)
				log.Warnf("btrfs: cannot read size for %s", p.Mountpoint)
				continue
			}
			mFree := reFree.FindStringSubmatch(stdout)
			if len(mFree) != 2 {
				log.Warnf("btrfs: cannot read free space for %s", p.Mountpoint)
				continue
			}
			mRatio := reRatio.FindStringSubmatch(stdout)
			if len(mRatio) != 2 {
				log.Warnf("btrfs: cannot read data ratio for %s", p.Mountpoint)
				continue
			}
			log.Debugf("btrfs: %s: size %s, free %s, ratio %s",
				p.Mountpoint, mSize[1], mFree[1], mRatio[1])
			var totalBytes float64
			var freeBytes float64
			var dataRatio float64
			totalBytes, pErr := strconv.ParseFloat(mSize[1], 64)
			if pErr != nil {
				log.Warnf("btrfs: cannot parse size for %s: %v", p.Mountpoint, pErr)
				continue
			}
			freeBytes, pErr = strconv.ParseFloat(mFree[1], 64)
			if pErr != nil {
				log.Warnf("btrfs: cannot parse free space for %s: %v", p.Mountpoint, pErr)
				continue
			}
			dataRatio, pErr = strconv.ParseFloat(mRatio[1], 64)
			if pErr != nil {
				log.Warnf("btrfs: cannot parse data ratio for %s: %v", p.Mountpoint, pErr)
				continue
			}
			// Correct totalBytes by diving it by data ratio
			// this is only accurate when data >> metadata
			totalBytes = totalBytes / dataRatio
			usedPerc := int((1 - (freeBytes / totalBytes)) * 100)
			totalStr := utils.FormatBytes(totalBytes)
			var firstStr string
			if showFree {
				firstStr = utils.FormatBytes(freeBytes) + " free"
			} else {
				firstStr = utils.FormatBytes(totalBytes-freeBytes) + " used"
			}
			if usedPerc < warnUsage && !warnOnly {
				content += fmt.Sprintf("%s: %s out of %s\n", utils.Wrap(p.Mountpoint, padL, padR), firstStr, totalStr)
			} else if usedPerc >= warnUsage && usedPerc < critUsage {
				if status != 'e' {
					status = 'w'
				}
				content += fmt.Sprintf("%s: %s out of %s\n", utils.Wrap(p.Mountpoint, padL, padR), firstStr, totalStr)
			} else if usedPerc >= critUsage {
				status = 'e'
				content += fmt.Sprintf("%s: %s out of %s\n", utils.Wrap(p.Mountpoint, padL, padR), firstStr, totalStr)
			}
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

func getBtrfsStatus(warnUsage int, critUsage int, warnOnly bool, showFree bool) (header string, content string, err error) {
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
		totalStr = utils.FormatBytes(totalBytes)
		usedPerc = int((usedBytes / totalBytes) * 100)
		var firstStr string
		if showFree {
			firstStr = utils.FormatBytes(totalBytes-usedBytes) + " free"
		} else {
			firstStr = utils.FormatBytes(usedBytes) + " used"
		}
		if usedPerc < warnUsage && !warnOnly {
			content += fmt.Sprintf("%s: %s out of %s\n", utils.Wrap(label, padL, padR), firstStr, totalStr)
		} else if usedPerc >= warnUsage && usedPerc < critUsage {
			if status != 'e' {
				status = 'w'
			}
			content += fmt.Sprintf("%s: %s out of %s\n", utils.Wrap(label, padL, padR), firstStr, totalStr)
		} else if usedPerc >= critUsage {
			status = 'e'
			content += fmt.Sprintf("%s: %s out of %s\n", utils.Wrap(label, padL, padR), firstStr, totalStr)
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
