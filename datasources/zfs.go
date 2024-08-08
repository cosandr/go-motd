package datasources

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/cosandr/go-motd/utils"
)

type ConfZFS struct {
	ConfBaseWarn `yaml:",inline"`
}

// Init sets up default alignment
func (c *ConfZFS) Init() {
	c.ConfBaseWarn.Init()
	c.PadHeader[1] = 6
}

// zpool list -Hpo name,alloc,size,health
// tank    6277009096704   11991548690432   ONLINE
// Sizes are in bytes

// GetZFS runs `zpool list -Ho name,alloc,size,health` and parses the output
func GetZFS(ch chan<- SourceReturn, conf *Conf) {
	c := conf.ZFS
	// Check for *c.WarnOnly override
	if c.WarnOnly == nil {
		c.WarnOnly = &conf.WarnOnly
	}
	sr := NewSourceReturn(conf.debug)
	defer func() {
		ch <- sr.Return(&c.ConfBase)
	}()
	sr.Header, sr.Content, sr.Error = getPoolStatus(&c)
}

func getPoolStatus(c *ConfZFS) (header string, content string, err error) {
	var buf bytes.Buffer
	cmd := exec.Command("zpool", "list", "-Hpo", "name,alloc,size,health")
	cmd.Stdout = &buf
	err = cmd.Run()
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("ZFS", c.padL, c.padR), utils.Warn("unavailable"))
		err = &ModuleNotAvailable{"zfs", err}
		return
	}
	var status = 'o'
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
			content += fmt.Sprintf("%s: %s, %s used out of %s\n", utils.Wrap(tmp[0], c.padL, c.padR), utils.Err(tmp[3]), usedStr, totalStr)
		} else if usedPerc < c.Warn && !*c.WarnOnly {
			content += fmt.Sprintf("%s: %s, %s used out of %s\n", utils.Wrap(tmp[0], c.padL, c.padR), utils.Good(tmp[3]), usedStr, totalStr)
		} else if usedPerc >= c.Warn && usedPerc < c.Crit {
			if status != 'e' {
				status = 'w'
			}
			content += fmt.Sprintf("%s: %s, %s used out of %s\n", utils.Wrap(tmp[0], c.padL, c.padR), utils.Warn(tmp[3]), usedStr, totalStr)
		} else if usedPerc >= c.Crit {
			status = 'e'
			content += fmt.Sprintf("%s: %s, %s used out of %s\n", utils.Wrap(tmp[0], c.padL, c.padR), utils.Err(tmp[3]), usedStr, totalStr)
		}
	}
	if status == 'o' {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("ZFS", c.padL, c.padR), utils.Good("OK"))
	} else if status == 'w' {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("ZFS", c.padL, c.padR), utils.Warn("Warning"))
	} else {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("ZFS", c.padL, c.padR), utils.Err("Critical"))
	}
	return
}
