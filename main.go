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
	"time"
)

var defaultCfgPath string = "./config.yaml"// Needs to match switch case below
var defaultOrder = []string{"sysinfo", "updates", "systemd", "docker", "disk", "cpu", "zfs"}

// Conf is the global config struct, defines YAML file
type Conf struct {
	FailedOnly bool `yaml:"failedOnly"`
	ShowOrder []string `yaml:"showOrder"`
	CPU mt.CommonWithWarn
	Disk mt.CommonWithWarn
	Docker docker.Conf
	SysInfo mt.Common
	Systemd systemd.Conf
	Updates updates.Conf
	ZFS mt.CommonWithWarn
}

func readCfg(path string) (c Conf, err error) {
	c.Init()
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("Config file error: %v ", err)
		return
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		err = fmt.Errorf("Cannot parse %s: %v", path, err)
		return
	}
	return
}

func getDocker(ret chan<- string, c Conf, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	// Check for failedOnly override
	if c.Docker.FailedOnly == nil {
		c.Docker.FailedOnly = &c.FailedOnly
	}
	docker.Get(ret, &c.Docker)
	if timing { fmt.Printf("Docker in: %s\n", time.Since(start).String()) }
}

func getSysInfo(ret chan<- string, c Conf, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	sysinfo.Get(ret, &c.SysInfo)
	if timing { fmt.Printf("Sysinfo in: %s\n", time.Since(start).String()) }
}

func getSystemD(ret chan<- string, c Conf, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	// Check for failedOnly override
	if c.Systemd.FailedOnly == nil {
		c.Systemd.FailedOnly = &c.FailedOnly
	}
	systemd.Get(ret, &c.Systemd)
	if timing { fmt.Printf("Systemd in: %s\n", time.Since(start).String()) }
}

func getCPUTemp(ret chan<- string, c Conf, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	// Check for failedOnly override
	if c.CPU.FailedOnly == nil {
		c.CPU.FailedOnly = &c.FailedOnly
	}
	temps.GetCPUTemp(ret, &c.CPU)
	if timing { fmt.Printf("CPU temp in: %s\n", time.Since(start).String()) }
}

func getDiskTemp(ret chan<- string, c Conf, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	// Check for failedOnly override
	if c.Disk.FailedOnly == nil {
		c.Disk.FailedOnly = &c.FailedOnly
	}
	temps.GetDiskTemps(ret, &c.Disk)
	if timing { fmt.Printf("Disk temp in: %s\n", time.Since(start).String()) }
}

func getUpdates(ret chan<- string, c Conf, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	// Check for failedOnly override
	if c.Updates.Show == nil {
		c.Updates.Show = &c.FailedOnly
	}
	updates.Get(ret, &c.Updates)
	if timing { fmt.Printf("Updates in: %s\n", time.Since(start).String()) }
}

func getZFS(ret chan<- string, c Conf, timing bool) {
	var start time.Time
	if timing { start = time.Now() }
	// Check for failedOnly override
	if c.ZFS.FailedOnly == nil {
		c.ZFS.FailedOnly = &c.FailedOnly
	}
	zfs.Get(ret, &c.ZFS)
	if timing { fmt.Printf("ZFS in: %s\n", time.Since(start).String()) }
}

func debugDumpConfig(c *Conf) {
	d, err := yaml.Marshal(c)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("Config dump:\n%s\n\n", string(d))
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

	// Ideally same as c.ShowOrder, invalid module names are excluded
	var printOrder []string

	// Did we get a list of enabled modules?
	if len(c.ShowOrder) > 0 {
		// checkSet is initialized with all valid module names
		var checkSet mt.StringSet
		checkSet = checkSet.FromList(defaultOrder)
		for _, k := range c.ShowOrder {
			if checkSet.Contains(k) {
				printOrder = append(printOrder, k)
			} else {
				fmt.Printf("Unknown module %s\n", k)
			}
		}
	} else {
		printOrder = make([]string, len(defaultOrder))
		copy(printOrder, defaultOrder)
	}

	// Generate output string channels
	var outCh = make(map[string]chan string)
	for _, k := range printOrder {
		outCh[k] = make(chan string, 1)
	}

	for _, k := range printOrder {
		switch k {
		case "docker":
			go getDocker(outCh[k], c, timing)
		case "systemd":
			go getSystemD(outCh[k], c, timing)
		case "sysinfo":
			go getSysInfo(outCh[k], c, timing)
		case "cpu":
			go getCPUTemp(outCh[k], c, timing)
		case "disk":
			go getDiskTemp(outCh[k], c, timing)
		case "updates":
			go getUpdates(outCh[k], c, timing)
		case "zfs":
			go getZFS(outCh[k], c, timing)
		default:
			// Critical failure
			panic(fmt.Errorf("no case for %s", k))
		}
	}

	// Wait and print results
	for _, k := range printOrder {
		fmt.Println(<- outCh[k])
	}

	if timing { fmt.Printf("Main ran in: %s\n", time.Since(startMain).String()) }
	// debugDumpConfig(&c)
	// fmt.Printf("Struct dump:\n%#v\n\n", c)
}

// Init a config with sane default values
func (c *Conf) Init() {
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
}
