package datasources

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/v3/disk"
	log "github.com/sirupsen/logrus"

	"github.com/cosandr/go-motd/utils"
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
func GetBtrfs(ch chan<- SourceReturn, conf *Conf) {
	c := conf.BTRFS
	// Check for warnOnly override
	if c.WarnOnly == nil {
		c.WarnOnly = &conf.WarnOnly
	}
	sr := NewSourceReturn(conf.debug)
	defer func() {
		ch <- sr.Return(&c.ConfBase)
	}()
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
		sr.Header, sr.Content, sr.Error = getBtrfsStatusExec(cmd, &c)
		return
	}
	sr.Header, sr.Content, sr.Error = getBtrfsStatus(&c)
}

func getBtrfsStatusExec(cmd string, c *ConfBtrfs) (header string, content string, err error) {
	// Find all btrfs mounts
	parts, err := disk.Partitions(false)
	if err != nil {
		err = &ModuleNotAvailable{"btrfs", err}
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
			command := exec.Command(tmp[0], tmp[1:]...)
			log.Debugf("btrfs: exec: '%s'", command.String())
			var buf bytes.Buffer
			command.Stdout = &buf
			cErr := command.Run()
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
			if c.ShowFree {
				firstStr = utils.FormatBytes(freeBytes) + " free"
			} else {
				firstStr = utils.FormatBytes(totalBytes-freeBytes) + " used"
			}
			if usedPerc < c.Warn && !*c.WarnOnly {
				content += fmt.Sprintf("%s: %s out of %s\n", utils.Wrap(p.Mountpoint, c.padL, c.padR), firstStr, totalStr)
			} else if usedPerc >= c.Warn && usedPerc < c.Crit {
				if status != 'e' {
					status = 'w'
				}
				content += fmt.Sprintf("%s: %s out of %s\n", utils.Wrap(p.Mountpoint, c.padL, c.padR), firstStr, totalStr)
			} else if usedPerc >= c.Crit {
				status = 'e'
				content += fmt.Sprintf("%s: %s out of %s\n", utils.Wrap(p.Mountpoint, c.padL, c.padR), firstStr, totalStr)
			}
		}
	}
	if status == 'o' {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("BTRFS", c.padL, c.padR), utils.Good("OK"))
	} else if status == 'w' {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("BTRFS", c.padL, c.padR), utils.Warn("Warning"))
	} else {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("BTRFS", c.padL, c.padR), utils.Err("Critical"))
	}
	return
}

func getBtrfsStatus(c *ConfBtrfs) (header string, content string, err error) {
	matches, err := filepath.Glob("/sys/fs/btrfs/*-*")
	if err != nil {
		err = &ModuleNotAvailable{"btrfs", err}
		return
	}
	var status = 'o'
	for _, fs := range matches {
		// Get FS label
		var label string
		read, _ := os.ReadFile(filepath.Join(fs, "/label"))
		if read != nil {
			label = strings.TrimSpace(string(read))
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
			content += fmt.Sprintf("%s: %s\n", utils.Wrap(label, c.padL, c.padR), utils.Err("read error"))
			continue
		}
		totalStr = utils.FormatBytes(totalBytes)
		usedPerc = int((usedBytes / totalBytes) * 100)
		var firstStr string
		if c.ShowFree {
			firstStr = utils.FormatBytes(totalBytes-usedBytes) + " free"
		} else {
			firstStr = utils.FormatBytes(usedBytes) + " used"
		}
		if usedPerc < c.Warn && !*c.WarnOnly {
			content += fmt.Sprintf("%s: %s out of %s\n", utils.Wrap(label, c.padL, c.padR), firstStr, totalStr)
		} else if usedPerc >= c.Warn && usedPerc < c.Crit {
			if status != 'e' {
				status = 'w'
			}
			content += fmt.Sprintf("%s: %s out of %s\n", utils.Wrap(label, c.padL, c.padR), firstStr, totalStr)
		} else if usedPerc >= c.Crit {
			status = 'e'
			content += fmt.Sprintf("%s: %s out of %s\n", utils.Wrap(label, c.padL, c.padR), firstStr, totalStr)
		}
	}
	if status == 'o' {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("BTRFS", c.padL, c.padR), utils.Good("OK"))
	} else if status == 'w' {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("BTRFS", c.padL, c.padR), utils.Warn("Warning"))
	} else {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("BTRFS", c.padL, c.padR), utils.Err("Critical"))
	}
	return
}

func readFloatFile(file string) float64 {
	readBytes, err := os.ReadFile(file)
	if err != nil {
		return 0
	}
	parsedFloat, err := strconv.ParseFloat(strings.TrimSpace(string(readBytes)), 64)
	if err != nil {
		return 0
	}
	return parsedFloat
}
