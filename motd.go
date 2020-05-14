package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/cosandr/go-motd/datasources"
	"github.com/cosandr/go-motd/utils"
	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v2"
)

var (
	defaultCfgPath = "./config.yaml"
	defaultOrder   = []string{"sysinfo", "updates", "systemd", "docker", "disk", "cpu", "zfs", "btrfs"}
)

// Conf is the global config struct, defines YAML file
type Conf struct {
	FailedOnly bool       `yaml:"failedOnly"`
	ShowOrder  []string   `yaml:"showOrder"`
	ColDef     [][]string `yaml:"colDef"`
	ColPad     int        `yaml:"colPad"`
	CPU        datasources.CPUTempConf
	Disk       datasources.DiskConf
	Docker     datasources.DockerConf
	SysInfo    datasources.CommonConf
	Systemd    datasources.SystemdConf
	Updates    datasources.UpdatesConf
	ZFS        datasources.CommonWithWarnConf
	BTRFS      datasources.CommonWithWarnConf
}

// Init a config with sane default values
func (c *Conf) Init() {
	// Init slices
	c.CPU.CommonConf.Init()
	c.Disk.CommonConf.Init()
	c.Docker.CommonConf.Init()
	c.SysInfo.Init()
	c.Systemd.CommonConf.Init()
	c.Updates.CommonConf.Init()
	c.ZFS.CommonConf.Init()
	c.BTRFS.CommonConf.Init()
	// Set some defaults
	c.ColPad = 4
	c.Updates.File = "/tmp/go-check-updates.yaml"
	c.Updates.Check, _ = time.ParseDuration("24h")
	c.Disk.Warn = 40
	c.Disk.Crit = 50
	c.CPU.Warn = 70
	c.CPU.Crit = 90
	c.ZFS.Warn = 70
	c.ZFS.Crit = 90
	c.BTRFS.Warn = 70
	c.BTRFS.Crit = 90
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

func getDocker(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for failedOnly override
	if c.Docker.FailedOnly == nil {
		c.Docker.FailedOnly = &c.FailedOnly
	}
	datasources.GetDocker(ret, &c.Docker)
	endTime <- time.Now()
}

func getSysInfo(ret chan<- string, c Conf, endTime chan<- time.Time) {
	datasources.GetSysInfo(ret, &c.SysInfo)
	endTime <- time.Now()
}

func getSystemD(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for failedOnly override
	if c.Systemd.FailedOnly == nil {
		c.Systemd.FailedOnly = &c.FailedOnly
	}
	datasources.GetSystemd(ret, &c.Systemd)
	endTime <- time.Now()
}

func getCPUTemp(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for failedOnly override
	if c.CPU.FailedOnly == nil {
		c.CPU.FailedOnly = &c.FailedOnly
	}
	datasources.GetCPUTemp(ret, &c.CPU)
	endTime <- time.Now()
}

func getDiskTemp(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for failedOnly override
	if c.Disk.FailedOnly == nil {
		c.Disk.FailedOnly = &c.FailedOnly
	}
	datasources.GetDiskTemps(ret, &c.Disk)
	endTime <- time.Now()
}

func getUpdates(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for failedOnly override
	if c.Updates.Show == nil {
		c.Updates.Show = &c.FailedOnly
	}
	datasources.GetUpdates(ret, &c.Updates)
	endTime <- time.Now()
}

func getZFS(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for failedOnly override
	if c.ZFS.FailedOnly == nil {
		c.ZFS.FailedOnly = &c.FailedOnly
	}
	datasources.GetZFS(ret, &c.ZFS)
	endTime <- time.Now()
}

func getBtrfs(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for failedOnly override
	if c.BTRFS.FailedOnly == nil {
		c.BTRFS.FailedOnly = &c.FailedOnly
	}
	datasources.GetBtrfs(ret, &c.BTRFS)
	endTime <- time.Now()
}

func makeTable(buf *strings.Builder, padding int) (table *tablewriter.Table) {
	table = tablewriter.NewWriter(buf)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding(strings.Repeat(" ", padding))
	table.SetNoWhiteSpace(true)
	return
}

func mapToTable(inStr map[string]string, colDef [][]string, buf *strings.Builder, padding int) {
	table := makeTable(buf, padding)
	var tmp []string
	// Render a new table every row for compact output
	for _, row := range colDef {
		// Just write block to buffer if it is alone
		if len(row) == 1 {
			a, ok := inStr[row[0]]
			// Skip invalid modules
			if !ok {
				continue
			}
			fmt.Fprintln(buf, a)
			continue
		}
		tmp = nil
		for _, k := range row {
			a, ok := inStr[k]
			if !ok {
				continue
			}
			tmp = append(tmp, a)
		}
		table.Append(tmp)
		table.Render()
		// Remake table to avoid imbalanced output
		table = makeTable(buf, padding)
	}
}

// makePrintOrder flattens colDef (if present). If showOrder is defined as well, it is ignored.
// The list of modules in either of them must be defined in defaultOrder or they will be ignored.
// This function removes invalid modules from c.ColDef
func makePrintOrder(c *Conf) (printOrder []string) {
	var tmp []string
	var verifiedCols [][]string
	var checkSet utils.StringSet
	checkSet = checkSet.FromList(defaultOrder)
	if len(c.ColDef) > 0 {
		// Flatten 2-dim input
		for _, row := range c.ColDef {
			tmp = nil
			for _, k := range row {
				if checkSet.Contains(k) {
					printOrder = append(printOrder, k)
					tmp = append(tmp, k)
				} else {
					fmt.Printf("Unknown module %s\n", k)
				}
			}
			if tmp != nil {
				verifiedCols = append(verifiedCols, tmp)
			}
		}
		c.ColDef = verifiedCols
	} else if len(c.ShowOrder) > 0 {
		for _, k := range c.ShowOrder {
			if checkSet.Contains(k) {
				printOrder = append(printOrder, k)
			} else {
				fmt.Printf("Unknown module %s\n", k)
			}
		}
	} else {
		// No need to check if using defaults
		printOrder = make([]string, len(defaultOrder))
		copy(printOrder, defaultOrder)
	}
	return
}

func main() {
	var timing bool
	var forceUpdates bool
	var path string
	var startTimes map[string]time.Time
	// Parse arguments
	flag.StringVar(&path, "cfg", defaultCfgPath, "Path to config.yml file")
	flag.BoolVar(&timing, "timing", false, "Enable timing")
	flag.BoolVar(&forceUpdates, "updates", false, "Show list of pending updates, overrides config")
	flag.Parse()

	if timing {
		startTimes = make(map[string]time.Time)
		startTimes["MAIN"] = time.Now()
	}
	// Read config file
	c, err := readCfg(path)
	if err != nil {
		fmt.Println(err)
	}

	if forceUpdates {
		c.Updates.Show = &forceUpdates
	}

	// Flatten colDef and check for invalid module names
	var printOrder = makePrintOrder(&c)

	var endTimes = make(map[string]chan time.Time)
	endTimes["MAIN"] = make(chan time.Time, 1)
	// Generate output string channels
	var outCh = make(map[string]chan string)
	for _, k := range printOrder {
		outCh[k] = make(chan string, 1)
		if timing {
			endTimes[k] = make(chan time.Time, 1)
		}
	}

	for _, k := range printOrder {
		if timing {
			startTimes[k] = time.Now()
		}
		switch k {
		case "docker":
			go getDocker(outCh[k], c, endTimes[k])
		case "systemd":
			go getSystemD(outCh[k], c, endTimes[k])
		case "sysinfo":
			go getSysInfo(outCh[k], c, endTimes[k])
		case "cpu":
			go getCPUTemp(outCh[k], c, endTimes[k])
		case "disk":
			go getDiskTemp(outCh[k], c, endTimes[k])
		case "updates":
			go getUpdates(outCh[k], c, endTimes[k])
		case "zfs":
			go getZFS(outCh[k], c, endTimes[k])
		case "btrfs":
			go getBtrfs(outCh[k], c, endTimes[k])
		default:
			// Critical failure
			panic(fmt.Errorf("no case for %s", k))
		}
	}

	var outStr = make(map[string]string)
	// Wait and print results
	for _, k := range printOrder {
		outStr[k] = <-outCh[k]
	}
	if len(c.ColDef) > 0 {
		outBuf := &strings.Builder{}
		mapToTable(outStr, c.ColDef, outBuf, c.ColPad)
		fmt.Print(outBuf.String())
	} else {
		for _, k := range printOrder {
			fmt.Println(outStr[k])
		}
	}
	// Show timing results
	if timing {
		endTimes["MAIN"] <- time.Now()
		printOrder = append(printOrder, "MAIN")
		for _, k := range printOrder {
			fmt.Printf("%s ran in: %s\n", k, (<-endTimes[k]).Sub(startTimes[k]).String())
		}
	}
	// debugDumpConfig(&c)
	// fmt.Printf("Struct dump:\n%#v\n\n", c)
}

func debugDumpConfig(c *Conf) {
	d, err := yaml.Marshal(c)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
	fmt.Printf("Config dump:\n%s\n\n", string(d))
}
