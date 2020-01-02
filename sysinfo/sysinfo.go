package sysinfo

import (
	"os"
	"os/exec"
	"fmt"
	"bytes"
	"github.com/cosandr/go-motd/colors"
	"bufio"
	"regexp"
	"strings"
	"io/ioutil"
	"strconv"
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

func getUptime(buf *bytes.Buffer) string {
	uptime, err := runCmd("uptime", "-p", buf)
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

// GetSysInfo prints various stats about the host Linux OS (kernel, distro, load and more)
func GetSysInfo() (content string) {
	var stdout bytes.Buffer
	// Fetch all the things
	var distro = getDistroName()
	var kernel, _ = runCmd("uname", "-sr", &stdout)
	kernel = strings.ReplaceAll(kernel, "\n", "")
	var uptime = getUptime(&stdout)
	var loadavg = getLoadAvg()
	var mem = getMemoryInfo()

	// Add to content
	content += fmt.Sprintf("Distro\t: %s\n", distro)
	content += fmt.Sprintf("Kernel\t: %s\n", kernel)
	content += fmt.Sprintf("Uptime\t: %s\n", uptime)
	content += fmt.Sprintf("Load\t: %s\n", loadavg)
	content += fmt.Sprintf("RAM\t: %s\n", mem)

	return
}
