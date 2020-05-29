package main

import "time"

// OldConf is the global config struct, defines YAML file
type OldConf struct {
	FailedOnly bool       `yaml:"failedOnly"`
	ShowOrder  []string   `yaml:"showOrder"`
	ColDef     [][]string `yaml:"colDef"`
	ColPad     int        `yaml:"colPad"`
	BTRFS      OldCommonWithWarnConf
	CPU        OldCPUTempConf
	Disk       OldDiskConf
	Docker     OldDockerConf
	Podman     OldPodmanConf
	SysInfo    OldCommonConf
	Systemd    OldSystemdConf
	Updates    OldUpdatesConf
	ZFS        OldCommonWithWarnConf
}

// Init a config with sane default values
func (c *OldConf) Init() {
	// Init slices
	c.BTRFS.OldCommonConf.Init()
	c.CPU.OldCommonConf.Init()
	c.Disk.OldCommonConf.Init()
	c.Docker.OldCommonConf.Init()
	c.Podman.OldCommonConf.Init()
	c.SysInfo.Init()
	c.Systemd.OldCommonConf.Init()
	c.Updates.OldCommonConf.Init()
	c.ZFS.OldCommonConf.Init()
	// Set some defaults
	c.BTRFS.Crit = 90
	c.BTRFS.Warn = 70
	c.ColPad = 4
	c.CPU.Crit = 90
	c.CPU.Warn = 70
	c.Disk.Crit = 50
	c.Disk.Warn = 40
	c.Systemd.ShowFailed = true
	c.Updates.Check, _ = time.ParseDuration("24h")
	c.Updates.File = "/tmp/go-check-updates.yaml"
	c.ZFS.Crit = 90
	c.ZFS.Warn = 70
}

type OldCommonConf struct {
	FailedOnly *bool `yaml:"failedOnly,omitempty"`
	Header     []int `yaml:"header,flow"`
	Content    []int `yaml:"content,flow"`
}

// Init sets `Header` and `Content` to [0, 0]
func (c *OldCommonConf) Init() {
	var defPad = []int{0, 0}
	c.Content = defPad
	c.Header = defPad
}

// OldCommonWithWarnConf extends ConfBase with warning and critical values
type OldCommonWithWarnConf struct {
	OldCommonConf `yaml:",inline"`
	Warn          int `yaml:"warn"`
	Crit          int `yaml:"crit"`
}

type OldCPUTempConf struct {
	OldCommonWithWarnConf `yaml:",inline"`
	Exec                  bool `yaml:"useExec"`
}

type OldDiskConf struct {
	OldCommonWithWarnConf `yaml:",inline"`
	Ignore                []string `yaml:"ignore"`
	Sys                   bool     `yaml:"useSys"`
}

type OldDockerConf struct {
	OldCommonConf `yaml:",inline"`
	Exec          bool     `yaml:"useExec"`
	Ignore        []string `yaml:"ignore"`
}

type OldPodmanConf struct {
	OldCommonConf `yaml:",inline"`
	Sudo          bool     `yaml:"sudo"`
	IncludeSudo   bool     `yaml:"includeSudo"`
	Ignore        []string `yaml:"ignore"`
}

type OldSystemdConf struct {
	OldCommonConf `yaml:",inline"`
	Units         []string `yaml:"units"`
	HideExt       bool     `yaml:"hideExt"`
	InactiveOK    bool     `yaml:"inactiveOK"`
	ShowFailed    bool     `yaml:"showFailed"`
}

type OldUpdatesConf struct {
	OldCommonConf `yaml:",inline"`
	Show          *bool         `yaml:"show"`
	File          string        `yaml:"file"`
	Check         time.Duration `yaml:"check"`
}
