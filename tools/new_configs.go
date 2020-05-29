package main

type Conf struct {
	ConfGlobal `yaml:"global"`
	BTRFS      ConfBaseWarn `yaml:"btrfs"`
	CPU        ConfTempCPU  `yaml:"cpu"`
	Disk       ConfTempDisk `yaml:"disk"`
	Docker     ConfDocker   `yaml:"docker"`
	Podman     ConfPodman   `yaml:"podman"`
	SysInfo    ConfBase     `yaml:"sysinfo"`
	Systemd    ConfSystemd  `yaml:"systemd"`
	Updates    ConfUpdates  `yaml:"updates"`
	ZFS        ConfBaseWarn `yaml:"zfs"`
}

// Init a config with sane default values
func (c *Conf) Init() {
	// Init slices
	c.BTRFS.ConfBase.Init()
	c.CPU.ConfBase.Init()
	c.Disk.ConfBase.Init()
	c.Docker.ConfBase.Init()
	c.Podman.ConfBase.Init()
	c.SysInfo.Init()
	c.Systemd.ConfBase.Init()
	c.Updates.ConfBase.Init()
	c.ZFS.ConfBase.Init()
	// Set some defaults
	c.BTRFS.Crit = 90
	c.BTRFS.Warn = 70
	c.ColPad = 4
	c.CPU.Crit = 90
	c.CPU.Warn = 70
	c.Disk.Crit = 50
	c.Disk.Warn = 40
	c.Systemd.ShowFailed = true
	c.Updates.File = "/tmp/go-check-updates.yaml"
	c.ZFS.Crit = 90
	c.ZFS.Warn = 70
}

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

type ConfBase struct {
	// Override global setting
	WarnOnly *bool `yaml:"warnings_only,omitempty"`
	// 2-element array defining padding for header (title)
	PadHeader []int `yaml:"pad_header,flow"`
	// 2-element array defining padding for content (details)
	PadContent []int `yaml:"pad_content,flow"`
}

// Init sets `PadHeader` and `PadContent` to [0, 0]
func (c *ConfBase) Init() {
	var defPad = []int{0, 0}
	c.PadHeader = defPad
	c.PadContent = defPad
}

// ConfBaseWarn extends ConfBase with warning and critical values
type ConfBaseWarn struct {
	ConfBase `yaml:",inline"`
	Warn     int `yaml:"warn"`
	Crit     int `yaml:"crit"`
}

type ConfTempCPU struct {
	ConfBaseWarn `yaml:",inline"`
	// Get CPU temperatures by parsing 'sensors -j' output
	Exec bool `yaml:"use_exec"`
}

type ConfTempDisk struct {
	ConfBaseWarn `yaml:",inline"`
	// List of disks to ignore, as they appear in /dev/
	Ignore []string `yaml:"ignore,omitempty"`
	// Read temperatures from /sys/ directly, requires drivetemp kernel module
	Sys bool `yaml:"use_sys"`
}

type ConfDocker struct {
	ConfBase `yaml:",inline"`
	// Interact directly with the docker CLI, much slower than API
	Exec bool `yaml:"use_exec"`
	// List of container names to ignore
	Ignore []string `yaml:"ignore,omitempty"`
}

type ConfPodman struct {
	ConfBase `yaml:",inline"`
	// Run podman using sudo, you should have NOPASSWD set for the podman command
	Sudo bool `yaml:"sudo"`
	// Run podman as both root and current user
	IncludeSudo bool `yaml:"include_sudo"`
	// List of container names to ignore
	Ignore []string `yaml:"ignore,omitempty"`
}

type ConfSystemd struct {
	ConfBase `yaml:",inline"`
	// List of units to track, including extension
	Units []string `yaml:"units,omitempty"`
	// Remove extension when displaying units
	HideExt bool `yaml:"hide_ext"`
	// Consider inactive units OK
	InactiveOK bool `yaml:"inactive_ok"`
	// Get all failed units (in addition manually defined units above)
	ShowFailed bool `yaml:"show_failed"`
}

type ConfUpdates struct {
	ConfBase `yaml:",inline"`
	// Show packages that can be upgraded
	Show *bool `yaml:"show,omitempty"`
	// Path to go-check-updates cache file
	File string `yaml:"file"`
}
