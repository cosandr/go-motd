package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
	"gopkg.in/yaml.v2"

	"github.com/cosandr/go-motd/datasources"
	"github.com/cosandr/go-motd/utils"
)

var (
	defaultCfgPath = "./config.yaml"
	defaultOrder   = []string{"sysinfo", "updates", "systemd", "docker", "podman", "disk", "cpu", "zfs", "btrfs"}
)

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
func makePrintOrder(c *datasources.Conf) (printOrder []string) {
	if args.Updates {
		return []string{"updates"}
	}
	if len(c.ColDef) > 0 {
		// Flatten 2-dim input
		for _, row := range c.ColDef {
			for _, k := range row {
				printOrder = append(printOrder, k)
			}
		}
	} else if len(c.ShowOrder) > 0 {
		printOrder = c.ShowOrder
	} else {
		// Use default order
		printOrder = make([]string, len(defaultOrder))
		copy(printOrder, defaultOrder)
	}
	return
}

var args struct {
	ConfigFile string `arg:"-c,--config,env:CONFIG_FILE" help:"Path to config yaml"`
	Debug      bool   `arg:"--debug,env:DEBUG" help:"Debug mode"`
	DumpConfig bool   `arg:"--dump-config" help:"Dump config and exit"`
	LogLevel   string `arg:"--log.level,env:LOG_LEVEL" default:"WARN" help:"Set log level"`
	NoColors   bool   `arg:"--no-colors,env:NO_COLORS" help:"Disable colors"`
	Quiet      bool   `arg:"-q,--quiet" help:"Don't log to console"`
	Updates    bool   `arg:"-u,--updates" help:"Show pending updates and exit"`
}

func setupLogging() {
	var logLevel log.Level
	var err error
	getLogLevels := func(level log.Level) []log.Level {
		ret := make([]log.Level, 0)
		for _, lvl := range log.AllLevels {
			if level >= lvl {
				ret = append(ret, lvl)
			}
		}
		return ret
	}

	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	log.SetOutput(ioutil.Discard)
	if args.Debug {
		logLevel = log.DebugLevel
	} else {
		logLevel, err = log.ParseLevel(args.LogLevel)
		if err != nil {
			logLevel = log.WarnLevel
			log.Warnf("Unknown log level %s, defaulting to WARN", args.LogLevel)
		}
	}
	log.SetLevel(logLevel)
	levels := getLogLevels(logLevel)
	if !args.Quiet {
		log.AddHook(&writer.Hook{
			Writer:    os.Stderr,
			LogLevels: levels,
		})
	}
}

func main() {
	args.ConfigFile = defaultCfgPath
	arg.MustParse(&args)

	setupLogging()

	var mainStart time.Time
	if args.Debug {
		mainStart = time.Now()
	}
	if args.NoColors {
		utils.NoColors = true
	}
	// Read config file
	c, err := datasources.NewConfFromFile(args.ConfigFile, args.Debug)
	if err != nil {
		log.Warn(err)
	}

	if args.DumpConfig {
		log.Info("Dumping config")
		if flag.NArg() > 0 {
			dumpConfig(&c, flag.Arg(0))
		} else {
			dumpConfig(&c, "")
		}
		return
	}

	if args.Updates {
		log.Debug("Show only updates")
		// Set show to true
		c.Updates.Show = &args.Updates
		c.Updates.PadHeader = []int{0, 0}
	}

	outOrder, outData := datasources.RunSources(makePrintOrder(&c), &c)
	outStr := make(map[string]string)
	// Wait and print results
	for _, k := range outOrder {
		v, ok := outData[k]
		if !ok {
			continue
		}
		outStr[k] = v.Header
		if v.Content != "" {
			outStr[k] += "\n" + v.Content
		}
		if v.Error != nil {
			log.Warnf("%s error: %v", k, v.Error)
		}
	}
	if len(c.ColDef) > 0 {
		log.Debug("Format as table")
		outBuf := &strings.Builder{}
		mapToTable(outStr, c.ColDef, outBuf, c.ColPad)
		fmt.Print(outBuf.String())
	} else {
		log.Debug("Print as is")
		for _, k := range outOrder {
			fmt.Println(outStr[k])
		}
	}
	// Show timing results
	if args.Debug {
		times := make(map[string]time.Duration)
		times["MAIN"] = time.Now().Sub(mainStart)
		outOrder = append(outOrder, "MAIN")
		for _, k := range outOrder {
			log.Debugf("%s ran in: %s", k, outData[k].Time.String())
		}
	}
}

func dumpConfig(c *datasources.Conf, writeFile string) {
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
