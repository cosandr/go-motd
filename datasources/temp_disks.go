package datasources

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/cosandr/go-motd/utils"
)

// DiskConf extends CommonConf with a list of devices to ignore
type DiskConf struct {
	CommonWithWarnConf `yaml:",inline"`
	Ignore             []string `yaml:"ignore"`
}

// GetDiskTemps returns disk temperatures as reported by the hddtemp deamon
func GetDiskTemps(ret chan<- string, c *DiskConf) {
	header, content, _ := getFromHddtemp(c.Ignore, c.Warn, c.Crit, *c.FailedOnly)
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

func getFromHddtemp(ignoreList []string, warnTemp int, critTemp int, failedOnly bool) (header string, content string, err error) {
	conn, err := net.Dial("tcp", "127.0.0.1:7634")
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Disk temp", padL, padR), utils.Warn("unavailable"))
		return
	}
	defer conn.Close()
	message, err := bufio.NewReader(conn).ReadString('\n')
	if len(message) == 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Disk temp", padL, padR), utils.Err("failed"))
		return
	}
	var numNotOK uint8
	var numTotal uint8
	// Make set of ignored devices
	var ignoreSet utils.StringSet
	ignoreSet = ignoreSet.FromList(ignoreList)
	for _, line := range strings.Split(message, "||") {
		line = strings.TrimPrefix(line, "|")
		var tmp = strings.Split(line, "|")
		temp, err := strconv.Atoi(tmp[2])
		var diskName = strings.TrimPrefix(tmp[0], "/dev/")
		if ignoreSet.Contains(diskName) {
			continue
		}
		if err != nil {
			content += fmt.Sprintf("%s: %s\n", utils.Wrap(diskName, padL, padR), utils.Err("--"))
			numNotOK++
		} else if temp < warnTemp && !failedOnly {
			content += fmt.Sprintf("%s: %s\n", utils.Wrap(diskName, padL, padR), utils.Good(temp))
		} else if temp >= warnTemp && temp < critTemp {
			content += fmt.Sprintf("%s: %s\n", utils.Wrap(diskName, padL, padR), utils.Warn(temp))
			numNotOK++
		} else if temp >= critTemp {
			content += fmt.Sprintf("%s: %s\n", utils.Wrap(diskName, padL, padR), utils.Err(temp))
			numNotOK++
		}
		numTotal++
	}
	if numNotOK == 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Disk temp", padL, padR), utils.Good("OK"))
	} else if numNotOK < numTotal {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Disk temp", padL, padR), utils.Warn("Warning"))
	} else {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Disk temp", padL, padR), utils.Err("Critical"))
	}
	return
}
