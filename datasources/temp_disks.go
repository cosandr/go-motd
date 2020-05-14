package datasources

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/cosandr/go-motd/utils"
)

type diskEntry struct {
	block string
	temps []diskTemp
	model string
}

type diskTemp struct {
	name string
	temp float64
}

// DiskConf extends CommonConf with a list of devices to ignore
type DiskConf struct {
	CommonWithWarnConf `yaml:",inline"`
	Ignore             []string `yaml:"ignore"`
	Sys                bool     `yaml:"useSys"`
}

// GetDiskTemps returns disk temperatures using hddtemp daemon or drivetemp kernel driver
func GetDiskTemps(ret chan<- string, c *DiskConf) {
	var diskEntries []diskEntry
	var err error
	if c.Sys {
		diskEntries, err = getFromSys()
	} else {
		diskEntries, err = getFromHddtemp()
	}
	var header string
	var content string
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Disk temp", padL, padR), utils.Warn("unavailable"))
	} else {
		header, content, _ = formatDiskEntries(diskEntries, c.Ignore, c.Warn, c.Crit, *c.FailedOnly)
	}
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

func formatDiskEntries(diskEntries []diskEntry, ignoreList []string, warnTemp int, critTemp int, failedOnly bool) (header string, content string, err error) {
	var numNotOK uint8
	var numTotal uint8
	// Make set of ignored devices
	var ignoreSet utils.StringSet
	ignoreSet = ignoreSet.FromList(ignoreList)
	for _, entry := range diskEntries {
		if ignoreSet.Contains(entry.block) {
			continue
		}
		if len(entry.temps) == 0 {
			content += fmt.Sprintf("%s: %s\n", utils.Wrap(entry.block, padL, padR), utils.Err("--"))
			numNotOK++
			continue
		}
		for _, t := range entry.temps {
			temp := int(t.temp)
			diskName := entry.block
			if len(t.name) > 0 {
				diskName += fmt.Sprintf(" - %s", t.name)
			}
			if temp < warnTemp && !failedOnly {
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

func getFromHddtemp() (deviceList []diskEntry, err error) {
	conn, err := net.Dial("tcp", "127.0.0.1:7634")
	if err != nil {
		return
	}
	defer conn.Close()
	message, err := bufio.NewReader(conn).ReadString('\n')
	if len(message) == 0 {
		err = fmt.Errorf("no response from hddtemp")
		return
	}
	// Remove EOF error
	err = nil
	var temps []diskTemp
	for _, line := range strings.Split(message, "||") {
		line = strings.TrimPrefix(line, "|")
		tmp := strings.Split(line, "|")
		if tmp[2] == "NA" {
			temps = nil
		} else {
			temp, _ := strconv.ParseFloat(tmp[2], 64)
			temps = []diskTemp{{name: "sensor", temp: temp}}
		}
		block := strings.TrimPrefix(tmp[0], "/dev/")
		deviceList = append(deviceList, diskEntry{
			block: block,
			model: tmp[1],
			temps: temps,
		})
	}
	return
}
