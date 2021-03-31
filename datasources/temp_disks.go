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

// ConfTempDisk extends ConfBase with a list of devices to ignore
type ConfTempDisk struct {
	ConfBaseWarn `yaml:",inline"`
	// List of disks to ignore, as they appear in /dev/
	Ignore []string `yaml:"ignore,omitempty"`
	// Read temperatures from /sys/ directly, requires drivetemp kernel module
	Sys bool `yaml:"use_sys"`
}

// Init sets warning temperature to 40C and critical to 50C
func (c *ConfTempDisk) Init() {
	c.ConfBase.Init()
	c.Warn = 40
	c.Crit = 50
}

// GetDiskTemps returns disk temperatures using hddtemp daemon or drivetemp kernel driver
func GetDiskTemps(ch chan<- SourceReturn, conf *Conf) {
	c := conf.Disk
	// Check for *c.WarnOnly override
	if c.WarnOnly == nil {
		c.WarnOnly = &conf.WarnOnly
	}
	sr := NewSourceReturn(conf.debug)
	defer func() {
		ch <- sr.Return(&c.ConfBase)
	}()
	var diskEntries []diskEntry
	var err error
	if c.Sys {
		diskEntries, err = getFromSys()
	} else {
		diskEntries, err = getFromHddtemp()
	}
	if err != nil {
		err = &ModuleNotAvailable{"disk", err}
		sr.Header = fmt.Sprintf("%s: %s\n", utils.Wrap("Disk temp", c.padL, c.padR), utils.Warn("unavailable"))
	} else {
		sr.Header, sr.Content, sr.Error = formatDiskEntries(diskEntries, &c)
	}
	return
}

func formatDiskEntries(diskEntries []diskEntry, c *ConfTempDisk) (header string, content string, err error) {
	var numNotOK uint8
	var numTotal uint8
	// Make set of ignored devices
	var ignoreSet utils.StringSet
	ignoreSet = ignoreSet.FromList(c.Ignore)
	for _, entry := range diskEntries {
		if ignoreSet.Contains(entry.block) {
			continue
		}
		if len(entry.temps) == 0 {
			content += fmt.Sprintf("%s: %s\n", utils.Wrap(entry.block, c.padL, c.padR), utils.Err("--"))
			numNotOK++
			continue
		}
		for _, t := range entry.temps {
			temp := int(t.temp)
			diskName := entry.block
			if len(t.name) > 0 {
				diskName += fmt.Sprintf(" - %s", t.name)
			}
			if temp < c.Warn && !*c.WarnOnly {
				content += fmt.Sprintf("%s: %s\n", utils.Wrap(diskName, c.padL, c.padR), utils.Good(temp))
			} else if temp >= c.Warn && temp < c.Crit {
				content += fmt.Sprintf("%s: %s\n", utils.Wrap(diskName, c.padL, c.padR), utils.Warn(temp))
				numNotOK++
			} else if temp >= c.Crit {
				content += fmt.Sprintf("%s: %s\n", utils.Wrap(diskName, c.padL, c.padR), utils.Err(temp))
				numNotOK++
			}
			numTotal++
		}
	}
	if numNotOK == 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Disk temp", c.padL, c.padR), utils.Good("OK"))
	} else if numNotOK < numTotal {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Disk temp", c.padL, c.padR), utils.Warn("Warning"))
	} else {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Disk temp", c.padL, c.padR), utils.Err("Critical"))
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
