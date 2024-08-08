package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

func readCfgOld(path string) (c OldConf, err error) {
	c.Init()
	yamlFile, err := os.ReadFile(path)
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

func dumpCfgOld(c *OldConf, writeFile string) {
	d, err := yaml.Marshal(c)
	if err != nil {
		fmt.Printf("Config parse error: %v\n", err)
		return
	}
	if writeFile != "" {
		err = os.WriteFile(writeFile, d, 0644)
		if err != nil {
			fmt.Printf("Config dumped failed: %v\n", err)
			return
		}
		fmt.Printf("Config dumped to: %s\n", writeFile)
	} else {
		fmt.Printf("%s\n", string(d))
	}
}

func dumpCfg(c *Conf, writeFile string) {
	d, err := yaml.Marshal(c)
	if err != nil {
		fmt.Printf("Config parse error: %v\n", err)
		return
	}
	if writeFile != "" && writeFile != "--" {
		err = os.WriteFile(writeFile, d, 0644)
		if err != nil {
			fmt.Printf("Config dumped failed: %v\n", err)
			return
		}
		fmt.Printf("Config dumped to: %s\n", writeFile)
	} else {
		fmt.Printf("%s\n", string(d))
	}
}

func readCfg(path string) (c Conf, err error) {
	c.Init()
	yamlFile, err := os.ReadFile(path)
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

func convertCfg(oc *OldConf) (c Conf) {
	c.Init()
	// Global
	c.WarnOnly = oc.FailedOnly
	c.ShowOrder = oc.ShowOrder
	c.ColDef = oc.ColDef
	c.ColPad = oc.ColPad
	// BTRFS
	c.BTRFS.WarnOnly = oc.BTRFS.FailedOnly
	c.BTRFS.PadHeader = oc.BTRFS.Header
	c.BTRFS.PadContent = oc.BTRFS.Content
	c.BTRFS.Warn = oc.BTRFS.Warn
	c.BTRFS.Crit = oc.BTRFS.Crit
	// CPU
	c.CPU.WarnOnly = oc.CPU.FailedOnly
	c.CPU.PadHeader = oc.CPU.Header
	c.CPU.PadContent = oc.CPU.Content
	c.CPU.Warn = oc.CPU.Warn
	c.CPU.Crit = oc.CPU.Crit
	c.CPU.Exec = oc.CPU.Exec
	// Disk
	c.Disk.WarnOnly = oc.Disk.FailedOnly
	c.Disk.PadHeader = oc.Disk.Header
	c.Disk.PadContent = oc.Disk.Content
	c.Disk.Warn = oc.Disk.Warn
	c.Disk.Crit = oc.Disk.Crit
	c.Disk.Ignore = oc.Disk.Ignore
	c.Disk.Sys = oc.Disk.Sys
	// Docker
	c.Docker.WarnOnly = oc.Docker.FailedOnly
	c.Docker.PadHeader = oc.Docker.Header
	c.Docker.PadContent = oc.Docker.Content
	c.Docker.Ignore = oc.Docker.Ignore
	c.Docker.Exec = oc.Docker.Exec
	// Podman
	c.Podman.WarnOnly = oc.Podman.FailedOnly
	c.Podman.PadHeader = oc.Podman.Header
	c.Podman.PadContent = oc.Podman.Content
	c.Podman.Ignore = oc.Podman.Ignore
	c.Podman.Sudo = oc.Podman.Sudo
	c.Podman.IncludeSudo = oc.Podman.IncludeSudo
	// SysInfo
	c.SysInfo.WarnOnly = oc.SysInfo.FailedOnly
	c.SysInfo.PadHeader = oc.SysInfo.Header
	c.SysInfo.PadContent = oc.SysInfo.Content
	// Systemd
	c.Systemd.WarnOnly = oc.Systemd.FailedOnly
	c.Systemd.PadHeader = oc.Systemd.Header
	c.Systemd.PadContent = oc.Systemd.Content
	c.Systemd.Units = oc.Systemd.Units
	c.Systemd.HideExt = oc.Systemd.HideExt
	c.Systemd.InactiveOK = oc.Systemd.InactiveOK
	c.Systemd.ShowFailed = oc.Systemd.ShowFailed
	// Updates
	c.Updates.WarnOnly = oc.Podman.FailedOnly
	c.Updates.PadHeader = oc.Podman.Header
	c.Updates.PadContent = oc.Podman.Content
	c.Updates.Show = oc.Updates.Show
	c.Updates.File = oc.Updates.File
	// ZFS
	c.ZFS.WarnOnly = oc.ZFS.FailedOnly
	c.ZFS.PadHeader = oc.ZFS.Header
	c.ZFS.PadContent = oc.ZFS.Content
	c.ZFS.Warn = oc.ZFS.Warn
	c.ZFS.Crit = oc.ZFS.Crit
	return
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Path to old config is required")
		os.Exit(1)
	}
	oldPath := os.Args[1]
	if oldPath == "-h" || oldPath == "--help" {
		fmt.Printf(`Usage %s: old_config [new_config]
If new_config is '--', it will be written to stdout
If omitted, .old is appended to old_config and the new config written in its place
`, os.Args[0])
		os.Exit(0)
	}
	oldCfg, err := readCfgOld(oldPath)
	if err != nil {
		fmt.Printf("Config parse error: %v", err)
		os.Exit(1)
	}
	var newPath string
	// We have a destination as well
	if len(os.Args) > 2 {
		newPath = os.Args[2]
	} else {
		// Rename old config
		err = os.Rename(oldPath, oldPath+".old")
		if err != nil {
			fmt.Printf("Cannot backup old config: %v", err)
			os.Exit(1)
		}
		newPath = oldPath
	}
	//dumpCfgOld(&oldCfg, "old_config.yaml")
	newCfg := convertCfg(&oldCfg)
	dumpCfg(&newCfg, newPath)
}
