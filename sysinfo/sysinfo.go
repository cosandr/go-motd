package sysinfo

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/cosandr/go-motd/colors"
	mt "github.com/cosandr/go-motd/types"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	padL = "$"
	padR = "%"
)

// runCmd executes command and returns stdout as string
func runCmd(name string, args string, buf *bytes.Buffer) (string, error) {
	var retStr string
	cmd := exec.Command(name, args)
	cmd.Stdout = buf
	err := cmd.Run()
	if err != nil {
		retStr = colors.Warn("unavailable")
	} else {
		retStr = buf.String()
	}
	buf.Reset()
	return retStr, err
}

func getDistroName() (retStr string) {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		retStr = colors.Warn("unavailable")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Look for pretty name
	re := regexp.MustCompile(`PRETTY_NAME=(.*)`)
	for scanner.Scan() {
		m := re.FindSubmatch(scanner.Bytes())
		if len(m) > 1 {
			// Remove quotes
			retStr = strings.Replace(string(m[1]), `"`, "", 2)
			return
		}
	}
	if err := scanner.Err(); err != nil {
		retStr = colors.Warn("unavailable")
		return
	}
	return
}

func getUptime() string {
	var buf bytes.Buffer
	uptime, err := runCmd("uptime", "-p", &buf)
	if err != nil {
		return uptime
	}
	re := regexp.MustCompile(`(up\s|\n)`)
	return re.ReplaceAllString(uptime, "")
}

func getLoadAvg() string {
	loadavg, err := ioutil.ReadFile("/proc/loadavg")
	if err != nil {
		return colors.Warn("unavailable")
	}
	var loadArr = strings.Split(string(loadavg), " ")
	return fmt.Sprintf("%s [1m], %s [5m], %s [15m]", loadArr[0], loadArr[1], loadArr[2])
}

func getMemoryInfo() (retStr string) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		retStr = colors.Warn("unavailable")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Look for active and total
	var memActive float64 = 0
	var memTotal float64 = 0
	reActive := regexp.MustCompile(`Active:\s+(\d+)`)
	reTotal := regexp.MustCompile(`MemTotal:\s+(\d+)`)
	for scanner.Scan() {
		if memTotal != 0 && memActive != 0 { break }
		if memActive == 0 {
			// Look for active
			m := reActive.FindSubmatch(scanner.Bytes())
			if len(m) > 1 {
				// Store as int
				memActive, _ = strconv.ParseFloat(string(m[1]), 64)
			}
		}
		if memTotal == 0 {
			m := reTotal.FindSubmatch(scanner.Bytes())
			if len(m) > 1 {
				// Store as int
				memTotal, _ = strconv.ParseFloat(string(m[1]), 64)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		retStr = colors.Warn("unavailable")
		return
	}
	// Convert to GB, meminfo is in kB
	return fmt.Sprintf("%.2f GB active of %.2f GB", memActive/1e6, memTotal/1e6)
}

func getKernel() string {
	var buf bytes.Buffer
	var kernel, _ = runCmd("uname", "-sr", &buf)
	return strings.ReplaceAll(kernel, "\n", "")
}

// Get various stats about the host Linux OS (kernel, distro, load and more)
func Get(ret *string, c *mt.Common) {
	header := getSysInfo()
	// Pad header
	var p = mt.Pad{Delims: map[string]int{padL: c.Header[0], padR: c.Header[1]}, Content: header}
	header = p.Do()
	*ret = header
}

func getSysInfo() (header string) {
	// Fetch all the things
	var info = map[string]string{
		"Distro": getDistroName(),
		"Kernel": getKernel(),
		"Uptime": getUptime(),
		"Load": getLoadAvg(),
		"RAM": getMemoryInfo(),
	}
	for name, val := range info {
		header += fmt.Sprintf("%s: %s\n", mt.Wrap(name, padL, padR), val)
	}
	return
}
