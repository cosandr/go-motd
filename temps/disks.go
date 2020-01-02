package temps

import (
	"fmt"
	"net"
	"bufio"
	"strings"
	"strconv"
	"github.com/cosandr/go-motd/colors"
)

// GetDiskTemps returns disk temperatures as reported by the hddtemp deamon
func GetDiskTemps(warnTemp int, critTemp int, failedOnly bool) (header string, content string, err error) {
	conn, err := net.Dial("tcp", "127.0.0.1:7634")
	defer conn.Close()
	if err != nil {
		header = "Disk temp\t: " + colors.Warn("unavailable")
		return
	}
	message, err := bufio.NewReader(conn).ReadString('\n')
	if len(message) == 0 {
		header = "Disk temp\t: " + colors.Err("failed")
		return
	}
	var numNotOK uint8 = 0
	var numTotal uint8 = 0
	for _, line := range strings.Split(message, "||") {
		line = strings.TrimPrefix(line, "|")
		var tmp = strings.Split(line, "|")
		temp, err := strconv.Atoi(tmp[2])
		if err != nil {
			content += fmt.Sprintf("%s\t: %s\n", tmp[0], colors.Err("--"))
			numNotOK++
		} else if temp < warnTemp && !failedOnly {
			content += fmt.Sprintf("%s\t: %s\n", tmp[0], colors.Good(temp))
		} else if temp >= warnTemp && temp < critTemp {
			content += fmt.Sprintf("%s\t: %s\n", tmp[0], colors.Warn(temp))
			numNotOK++
		} else if temp >= critTemp {
			content += fmt.Sprintf("%s\t: %s\n", tmp[0], colors.Err(temp))
			numNotOK++
		}
		numTotal++
	}
	if numNotOK == 0 {
		header = fmt.Sprintf("Disk temp\t: %s\n", colors.Good("OK"))
	} else if numNotOK < numTotal {
		header = fmt.Sprintf("Disk temp\t: %s\n", colors.Warn("Warning"))
	} else {
		header = fmt.Sprintf("Disk temp\t: %s\n", colors.Err("Critical"))
	}
	return
}