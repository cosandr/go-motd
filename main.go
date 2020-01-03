package main

import (
	"github.com/cosandr/go-motd/systemd"
	"github.com/cosandr/go-motd/sysinfo"
	"github.com/cosandr/go-motd/docker"
	"github.com/cosandr/go-motd/temps"
	"github.com/cosandr/go-motd/zfs"
	"github.com/cosandr/go-motd/updates"
	"time"
	"fmt"
	"flag"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"bytes"
	"strings"
	"text/tabwriter"
	"sync"
)

var defaultCfgPath string = "./config.yaml"

type conf struct {
	FailedOnly bool `yaml:"failedOnly"`
	Updates UpdatesType
	SysInfo CommonT
	Systemd SystemdType
	Docker DockerType
	Disk PadWithWarn
	CPU PadWithWarn
	ZFS PadWithWarn
}

// UpdatesType extends CommonT with updates specific settings
type UpdatesType struct {
	CommonT `yaml:",inline"`
	Show bool `yaml:"show"`
	File string `yaml:"file"`
	Check time.Duration `yaml:"check"`
}

// SystemdType extends CommonT with a list of units
type SystemdType struct {
	CommonT `yaml:",inline"`
	Units []string `yaml:"units"`
}

// DockerType extends CommonT with a list of containers
type DockerType struct {
	CommonT `yaml:",inline"`
	Ignore []string `yaml:"ignore"`
}

// PadWithWarn extends CommonT with warning/critical values
type PadWithWarn struct {
	CommonT `yaml:",inline"`
	Warn int `yaml:"warn"`
	Crit int `yaml:"crit"`
}

// CommonT is the common type for all modules
type CommonT struct {
	FailedOnly bool `yaml:"failedOnly"`
	HeadR int `yaml:"headerRight"`
	HeadL int `yaml:"headerLeft"`
	ContR int `yaml:"contentRight"`
	ContL int `yaml:"contentLeft"`
}

func readCfg(c *conf, path *string) (*conf, error) {
	yamlFile, err := ioutil.ReadFile(*path)
	if err != nil {
		return defaultCfg(c)
		// return nil, fmt.Errorf("Config file error: %v ", err)
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return defaultCfg(c)
		// return nil, fmt.Errorf("Cannot parse %s: %v", *path, err)
	}
	return c, nil
}

// defaultCfg generates an empty config struct
func defaultCfg(c *conf) (*conf, error) {
	var data = `
updates:
  file: /tmp/go-updates.yaml
  check: 24h
disk:
  warn: 40
  crit: 50
cpu:
  warn: 70
  crit: 90
zfs:
  warn: 70
  crit: 90
`
	yaml.Unmarshal([]byte(data), c)
	return c, nil
}

func getSystemD(ret *string, c conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	var buf bytes.Buffer
	systemdConn := systemd.GetConn()
	defer systemd.CloseConn(systemdConn)
	systemdHeader, systemdContent, _ := systemd.GetServiceStatus(systemdConn, c.Systemd.Units, c.FailedOnly)
	// Pad header
	w := tabwriter.NewWriter(&buf, 0, 0, c.Systemd.HeadR, ' ', 0)
	fmt.Fprint(w, systemdHeader)
	w.Flush()
	// Pad services
	w = tabwriter.NewWriter(&buf, 0, 0, c.Systemd.ContR, ' ', 0)
	fmt.Fprint(w, systemdContent)
	w.Flush()
	*ret = strings.TrimSuffix(buf.String(), "\n")
	if timing { fmt.Printf("Systemd in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func getDocker(ret *string, c conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	var buf bytes.Buffer
	header, content, _ := docker.CheckContainers(c.Docker.Ignore, c.FailedOnly)
	// Pad header
	w := tabwriter.NewWriter(&buf, 0, 0, c.Docker.HeadR, ' ', 0)
	fmt.Fprint(w, header)
	w.Flush()
	// Pad containers
	w = tabwriter.NewWriter(&buf, 0, 0, c.Docker.ContR, ' ', 0)
	fmt.Fprint(w, content)
	w.Flush()
	*ret = strings.TrimSuffix(buf.String(), "\n")
	if timing { fmt.Printf("Docker in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func getSysInfo(ret *string, c conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	var buf bytes.Buffer
	var content = sysinfo.GetSysInfo()
	w := tabwriter.NewWriter(&buf, 0, 0, c.SysInfo.HeadR, ' ', 0)
	fmt.Fprint(w, content)
	w.Flush()
	*ret = strings.TrimSuffix(buf.String(), "\n")
	if timing { fmt.Printf("Sysinfo in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func getDiskTemp(ret *string, c conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	var buf bytes.Buffer
	header, content, _ := temps.GetDiskTemps(c.Disk.Warn, c.Disk.Crit, c.FailedOnly)
	// Pad header
	w := tabwriter.NewWriter(&buf, 0, 0, c.Disk.HeadR, ' ', 0)
	fmt.Fprint(w, header)
	w.Flush()
	// Pad containers
	w = tabwriter.NewWriter(&buf, 0, 0, c.Disk.ContR, ' ', 0)
	fmt.Fprint(w, content)
	w.Flush()
	*ret = strings.TrimSuffix(buf.String(), "\n")
	if timing { fmt.Printf("Disk temp in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func getCPUTemp(ret *string, c conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	var buf bytes.Buffer
	header, content, _ := temps.GetCPUTemp(c.CPU.Warn, c.CPU.Crit, c.FailedOnly)
	// Pad header
	w := tabwriter.NewWriter(&buf, 0, 0, c.CPU.HeadR, ' ', 0)
	fmt.Fprint(w, header)
	w.Flush()
	// Pad containers
	w = tabwriter.NewWriter(&buf, 0, 0, c.CPU.ContR, ' ', 0)
	fmt.Fprint(w, content)
	w.Flush()
	*ret = strings.TrimSuffix(buf.String(), "\n")
	if timing { fmt.Printf("CPU temp in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func getZFS(ret *string, c conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	var buf bytes.Buffer
	header, content, _ := zfs.GetPoolStatus(c.ZFS.Warn, c.ZFS.Crit, c.FailedOnly)
	// Pad header
	w := tabwriter.NewWriter(&buf, 0, 0, c.ZFS.HeadR, ' ', 0)
	fmt.Fprint(w, header)
	w.Flush()
	// Pad containers
	w = tabwriter.NewWriter(&buf, 0, 0, c.ZFS.ContR, ' ', 0)
	fmt.Fprint(w, content)
	w.Flush()
	*ret = strings.TrimSuffix(buf.String(), "\n")
	if timing { fmt.Printf("ZFS in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func getUpdates(ret *string, c conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	var buf bytes.Buffer
	header, content, _ := updates.Get(c.Updates.File, c.Updates.Check)
	// Pad header
	w := tabwriter.NewWriter(&buf, 0, 0, c.Updates.HeadR, ' ', 0)
	fmt.Fprint(w, header)
	w.Flush()
	if c.Updates.Show {
		// Pad content
		w = tabwriter.NewWriter(&buf, 0, 0, c.Updates.ContL, ' ', 0)
		fmt.Fprint(w, content)
		w.Flush()
	}	
	*ret = strings.TrimSuffix(buf.String(), "\n")
	if timing { fmt.Printf("Updates in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func main() {
	var timing bool
	var startMain time.Time
	// Parse arguments
	var path = flag.String("cfg", defaultCfgPath, "Path to config.yml file")
	flag.BoolVar(&timing, "timing", false, "Enable timing")
	flag.Parse()
	if timing { startMain = time.Now() }
	// Read config file
	var c conf
	_, err := readCfg(&c, path)
	if err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	// System info
	var sysInfoStr string
	wg.Add(1)
	go getSysInfo(&sysInfoStr, c, &wg, timing)

	// Updates status
	var updatesStr string
	wg.Add(1)
	go getUpdates(&updatesStr, c, &wg, timing)

	// Systemd service status
	var sysdStr string
	wg.Add(1)
	go getSystemD(&sysdStr, c, &wg, timing)

	// Docker containers
	var dockerStr string
	wg.Add(1)
	go getDocker(&dockerStr, c, &wg, timing)

	// Disk temps
	var diskStr string
	wg.Add(1)
	go getDiskTemp(&diskStr, c, &wg, timing)

	// CPU temps
	var cpuStr string
	wg.Add(1)
	go getCPUTemp(&cpuStr, c, &wg, timing)

	// ZFS pool status
	var zfsStr string
	wg.Add(1)
	go getZFS(&zfsStr, c, &wg, timing)

	wg.Wait()
	// Print results
	fmt.Println(sysInfoStr)
	fmt.Println(updatesStr)
	fmt.Println(sysdStr)
	fmt.Println(dockerStr)
	fmt.Println(diskStr)
	fmt.Println(cpuStr)
	fmt.Println(zfsStr)

	if timing { fmt.Printf("Main ran in: %s\n", time.Since(startMain).String()) }

}
