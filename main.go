package main

import (
	"flag"
	"fmt"
	"github.com/cosandr/go-motd/docker"
	"github.com/cosandr/go-motd/sysinfo"
	"github.com/cosandr/go-motd/systemd"
	"github.com/cosandr/go-motd/temps"
	mt "github.com/cosandr/go-motd/types"
	"github.com/cosandr/go-motd/updates"
	"github.com/cosandr/go-motd/zfs"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"sync"
	"time"
)

var defaultCfgPath string = "./config.yaml"

// Conf is the global config struct, defines YAML file
type Conf struct {
	FailedOnly bool `yaml:"failedOnly"`
	CPU mt.CommonWithWarn
	Disk mt.CommonWithWarn
	Docker docker.Conf
	SysInfo mt.Common
	Systemd systemd.Conf
	Updates updates.Conf
	ZFS mt.CommonWithWarn
}

func readCfg(path string) (*Conf, error) {
	var c *Conf
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		return NewConf(), fmt.Errorf("Config file error: %v ", err)
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		return NewConf(), fmt.Errorf("Cannot parse %s: %v", path, err)
	}
	return c, nil
}

func getSystemD(ret *string, c Conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	// Check for failedOnly override
	if c.Systemd.FailedOnly == nil {
		c.Systemd.FailedOnly = &c.FailedOnly
	}
	systemd.Get(ret, &c.Systemd)
	if timing { fmt.Printf("Systemd in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func getDocker(ret *string, c Conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	// Check for failedOnly override
	if c.Docker.FailedOnly == nil {
		c.Docker.FailedOnly = &c.FailedOnly
	}
	docker.Get(ret, &c.Docker)
	if timing { fmt.Printf("Docker in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func getSysInfo(ret *string, c Conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	sysinfo.Get(ret, &c.SysInfo)
	if timing { fmt.Printf("Sysinfo in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func getDiskTemp(ret *string, c Conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	// Check for failedOnly override
	if c.Disk.FailedOnly == nil {
		c.Disk.FailedOnly = &c.FailedOnly
	}
	temps.GetDiskTemps(ret, &c.Disk)
	if timing { fmt.Printf("Disk temp in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func getCPUTemp(ret *string, c Conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	// Check for failedOnly override
	if c.CPU.FailedOnly == nil {
		c.CPU.FailedOnly = &c.FailedOnly
	}
	temps.GetCPUTemp(ret, &c.CPU)
	if timing { fmt.Printf("CPU temp in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func getZFS(ret *string, c Conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	// Check for failedOnly override
	if c.ZFS.FailedOnly == nil {
		c.ZFS.FailedOnly = &c.FailedOnly
	}
	zfs.Get(ret, &c.ZFS)
	if timing { fmt.Printf("ZFS in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func getUpdates(ret *string, c Conf, wg *sync.WaitGroup, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	// Check for failedOnly override
	if c.Updates.Show == nil {
		c.Updates.Show = &c.FailedOnly
	}
	updates.Get(ret, &c.Updates)
	if timing { fmt.Printf("Updates in: %s\n", time.Since(start).String()) }
	wg.Done()
}

func debugDumpConfig(c *Conf) {
	d, err := yaml.Marshal(c)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("Config dump:\n%s\n\n", string(d))
}

func checkCfgLen(m string, c *mt.Common) error {
	if len(c.Header) < 2 {
		return fmt.Errorf("Header pad array in %s too short", m)
	}
	if len(c.Content) < 2 {
		return fmt.Errorf("Content pad array in %s too short", m)
	}
	return nil
}

func main() {
	var timing bool
	var path string
	var startMain time.Time
	// Parse arguments
	flag.StringVar(&path, "cfg", defaultCfgPath, "Path to config.yml file")
	flag.BoolVar(&timing, "timing", false, "Enable timing")
	flag.Parse()
	if timing { startMain = time.Now() }
	// Read config file
	c, err := readCfg(path)
	if err != nil {
		fmt.Println(err)
	}

	var wg sync.WaitGroup

	// System info
	var sysInfoStr string
	wg.Add(1)
	go getSysInfo(&sysInfoStr, *c, &wg, timing)

	// Updates status
	var updatesStr string
	wg.Add(1)
	go getUpdates(&updatesStr, *c, &wg, timing)

	// Systemd service status
	var sysdStr string
	wg.Add(1)
	go getSystemD(&sysdStr, *c, &wg, timing)

	// Docker containers
	var dockerStr string
	wg.Add(1)
	go getDocker(&dockerStr, *c, &wg, timing)

	// Disk temps
	var diskStr string
	wg.Add(1)
	go getDiskTemp(&diskStr, *c, &wg, timing)

	// CPU temps
	var cpuStr string
	wg.Add(1)
	go getCPUTemp(&cpuStr, *c, &wg, timing)

	// ZFS pool status
	var zfsStr string
	wg.Add(1)
	go getZFS(&zfsStr, *c, &wg, timing)

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
	// debugDumpConfig(&c)
	// fmt.Printf("Struct dump:\n%#v\n\n", c)
}

// NewConf returns a `Conf` with sane default values
func NewConf() *Conf {
	var c Conf = Conf{}
	// Init slices
	c.CPU.Common.Init()
	c.Disk.Common.Init()
	c.Docker.Common.Init()
	c.SysInfo.Init()
	c.Systemd.Common.Init()
	c.Updates.Common.Init()
	c.ZFS.Common.Init()
	// Set some defaults
	c.Updates.File = "/tmp/go-check-updates.yaml"
	c.Updates.Check, _ = time.ParseDuration("24h")
	c.Disk.Warn = 40
	c.Disk.Crit = 50
	c.CPU.Warn = 70
	c.CPU.Crit = 90
	c.ZFS.Warn = 70
	c.ZFS.Crit = 90
	return &c
}
