package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/cosandr/go-motd/datasources"
	"github.com/cosandr/go-motd/utils"
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	defaultCfgPath = "./config.yaml"
	defaultOrder   = []string{"sysinfo", "updates", "systemd", "docker", "podman", "disk", "cpu", "zfs", "btrfs"}
)

// ConfGlobal is the config struct for global settings
type ConfGlobal struct {
	// Hide fields which are deemed to be OK
	WarnOnly bool `yaml:"warnings_only"`
	// Order in which to display data sources
	ShowOrder []string `yaml:"show_order,flow,omitempty"`
	// Define how data sources are displayed
	ColDef [][]string `yaml:"col_def,flow,omitempty"`
	// Padding between columns when using col_def
	ColPad int `yaml:"col_pad"`
}

// Conf is the combined config struct, defines YAML file
type Conf struct {
	ConfGlobal `yaml:"global"`
	BTRFS      datasources.ConfBtrfs    `yaml:"btrfs"`
	CPU        datasources.ConfTempCPU  `yaml:"cpu"`
	Disk       datasources.ConfTempDisk `yaml:"disk"`
	Docker     datasources.ConfDocker   `yaml:"docker"`
	Podman     datasources.ConfPodman   `yaml:"podman"`
	SysInfo    datasources.ConfSysInfo  `yaml:"sysinfo"`
	Systemd    datasources.ConfSystemd  `yaml:"systemd"`
	Updates    datasources.ConfUpdates  `yaml:"updates"`
	ZFS        datasources.ConfZFS      `yaml:"zfs"`
}

// Init a config with sane default values
func (c *Conf) Init() {
	// Set global defaults
	c.WarnOnly = true
	c.ColPad = 4
	// Init data source configs
	c.BTRFS.Init()
	c.CPU.Init()
	c.Disk.Init()
	c.Docker.Init()
	c.Podman.Init()
	c.SysInfo.Init()
	c.Systemd.Init()
	c.Updates.Init()
	c.ZFS.Init()
}

func readCfg(path string) (c Conf, err error) {
	c.Init()
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("config file error: %v ", err)
		return
	}
	err = yaml.Unmarshal(yamlFile, &c)
	if err != nil {
		err = fmt.Errorf("cannot parse %s: %v", path, err)
		return
	}
	return
}

func getBtrfs(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for warnOnly override
	if c.BTRFS.WarnOnly == nil {
		c.BTRFS.WarnOnly = &c.WarnOnly
	}
	datasources.GetBtrfs(ret, &c.BTRFS)
	endTime <- time.Now()
}

func getCPUTemp(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for warnOnly override
	if c.CPU.WarnOnly == nil {
		c.CPU.WarnOnly = &c.WarnOnly
	}
	datasources.GetCPUTemp(ret, &c.CPU)
	endTime <- time.Now()
}

func getDiskTemp(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for warnOnly override
	if c.Disk.WarnOnly == nil {
		c.Disk.WarnOnly = &c.WarnOnly
	}
	datasources.GetDiskTemps(ret, &c.Disk)
	endTime <- time.Now()
}

func getDocker(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for warnOnly override
	if c.Docker.WarnOnly == nil {
		c.Docker.WarnOnly = &c.WarnOnly
	}
	datasources.GetDocker(ret, &c.Docker)
	endTime <- time.Now()
}

func getPodman(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for warnOnly override
	if c.Podman.WarnOnly == nil {
		c.Podman.WarnOnly = &c.WarnOnly
	}
	datasources.GetPodman(ret, &c.Podman)
	endTime <- time.Now()
}

func getSysInfo(ret chan<- string, c Conf, endTime chan<- time.Time) {
	datasources.GetSysInfo(ret, &c.SysInfo)
	endTime <- time.Now()
}

func getSystemD(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for warnOnly override
	if c.Systemd.WarnOnly == nil {
		c.Systemd.WarnOnly = &c.WarnOnly
	}
	datasources.GetSystemd(ret, &c.Systemd)
	endTime <- time.Now()
}

func getUpdates(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for warnOnly override
	if c.Updates.Show == nil {
		c.Updates.Show = &c.WarnOnly
	}
	datasources.GetUpdates(ret, &c.Updates)
	endTime <- time.Now()
}

func getZFS(ret chan<- string, c Conf, endTime chan<- time.Time) {
	// Check for warnOnly override
	if c.ZFS.WarnOnly == nil {
		c.ZFS.WarnOnly = &c.WarnOnly
	}
	datasources.GetZFS(ret, &c.ZFS)
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
					log.Warnf("Unknown module %s", k)
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
				log.Warnf("Unknown module %s", k)
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
	var flagUpdates bool
	var flagDumpCfg bool
	var flagCfg string
	// Parse arguments
	flag.StringVar(&flagCfg, "cfg", defaultCfgPath, "Path to yaml config file")
	flag.BoolVar(&utils.DebugMode, "debug", false, "Debug mode")
	flag.BoolVar(&flagUpdates, "updates", false, "Show list of pending updates only")
	flag.BoolVar(&flagDumpCfg, "dump-config", false, "Dump config to stdout or provided filepath")
	flag.Parse()

	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	log.SetOutput(os.Stderr)
	if utils.DebugMode {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}

	var startTimes map[string]time.Time
	if utils.DebugMode {
		startTimes = make(map[string]time.Time)
		startTimes["MAIN"] = time.Now()
	}
	// Read config file
	c, err := readCfg(flagCfg)
	if err != nil {
		log.Warn(err)
	}

	if flagDumpCfg {
		log.Info("Dumping config")
		if flag.NArg() > 0 {
			dumpConfig(&c, flag.Arg(0))
		} else {
			dumpConfig(&c, "")
		}
		return
	}

	var printOrder []string

	if flagUpdates {
		log.Debug("Show only updates")
		// Set show to true
		c.Updates.Show = &flagUpdates
		c.Updates.PadHeader = []int{0, 0}
		// Only show updates
		printOrder = []string{"updates"}
	} else {
		// Flatten colDef and check for invalid module names
		printOrder = makePrintOrder(&c)
	}

	var endTimes = make(map[string]chan time.Time)
	endTimes["MAIN"] = make(chan time.Time, 1)
	// Generate output string channels
	var outCh = make(map[string]chan string)
	for _, k := range printOrder {
		outCh[k] = make(chan string, 1)
		if utils.DebugMode {
			endTimes[k] = make(chan time.Time, 1)
		}
	}

	log.Debug("Start data collection goroutines")
	for _, k := range printOrder {
		if utils.DebugMode {
			startTimes[k] = time.Now()
		}
		switch k {
		case "btrfs":
			go getBtrfs(outCh[k], c, endTimes[k])
		case "cpu":
			go getCPUTemp(outCh[k], c, endTimes[k])
		case "disk":
			go getDiskTemp(outCh[k], c, endTimes[k])
		case "docker":
			go getDocker(outCh[k], c, endTimes[k])
		case "podman":
			go getPodman(outCh[k], c, endTimes[k])
		case "sysinfo":
			go getSysInfo(outCh[k], c, endTimes[k])
		case "systemd":
			go getSystemD(outCh[k], c, endTimes[k])
		case "updates":
			go getUpdates(outCh[k], c, endTimes[k])
		case "zfs":
			go getZFS(outCh[k], c, endTimes[k])
		default:
			// Critical failure
			log.Panicf("no case for %s", k)
		}
	}

	var outStr = make(map[string]string)
	// Wait and print results
	log.Debug("Wait for goroutines")
	for _, k := range printOrder {
		outStr[k] = <-outCh[k]
	}
	if len(c.ColDef) > 0 {
		log.Debug("Format as table")
		outBuf := &strings.Builder{}
		mapToTable(outStr, c.ColDef, outBuf, c.ColPad)
		fmt.Print(outBuf.String())
	} else {
		log.Debug("Print as is")
		for _, k := range printOrder {
			fmt.Println(outStr[k])
		}
	}
	// Show timing results
	if utils.DebugMode {
		endTimes["MAIN"] <- time.Now()
		printOrder = append(printOrder, "MAIN")
		for _, k := range printOrder {
			log.Debugf("%s ran in: %s\n", k, (<-endTimes[k]).Sub(startTimes[k]).String())
		}
	}
}

func dumpConfig(c *Conf, writeFile string) {
	d, err := yaml.Marshal(c)
	if err != nil {
		log.Errorf("Config parse error: %v", err)
		return
	}
	if writeFile != "" {
		err = ioutil.WriteFile(writeFile, d, 0644)
		if err != nil {
			log.Errorf("Config dumped failed: %v", err)
			return
		}
		log.Infof("Config dumped to: %s", writeFile)
	} else {
		fmt.Printf("%s\n", string(d))
	}
	return
}
