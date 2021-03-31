
[![Go Report Card](https://goreportcard.com/badge/github.com/cosandr/go-motd)](https://goreportcard.com/report/github.com/cosandr/go-motd) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/cosandr/go-motd/blob/master/LICENSE)

# Introduction

This project was inspired by [RIKRUS's](https://github.com/RIKRUS/MOTD) and [Hermann Björgvin's](https://github.com/HermannBjorgvin/motd) MOTD scripts.

I've decided to use Go because it is about 10x faster than a similar bash script and it makes for a great first project using the language. In my tests it typically runs in 10-20ms, a similar bash script takes 200-500ms.

The available information will depend on the user privileges, you will need to be able to run (without sudo) `systemctl status`, `docker ps` and `zpool status` for example.

Note that the BTRFS and ZFS space statistics are totals, that is to say, a RAID5 setup shows the used/total space across all drives. For example 3x4TB disks in RAIDZ1 show 10.91TB total, not the usable space which is about 7TB.

You can dump the default config by passing an invalid path as the `-c/--config` argument and using `--dump-config` at the same time.

Configuration changed on 2020-05-29, automatic conversion can be done with [migrate.go](./tools/migrate.go). TL;DR of changes:
- `global` section instead of being root level
- `header` and `content` are now `pad_header` and `pad_content`
- `failedOnly` is now `warnings_only`, I think this more clearly communicates what it does
- All keys changed from camelCase to snake_case, follows yaml standards better

## Example

### All OK

![go-motd-OK](https://user-images.githubusercontent.com/7095687/71813464-f5215580-3079-11ea-9f70-46f66c2557da.jpg)

### With some warnings

![go-motd-warn](https://user-images.githubusercontent.com/7095687/71813465-f5215580-3079-11ea-809d-05f661614679.jpg)

## Requirements

- Kernel 5.6+ (drivetemp module) or hddtemp daemon are required for disk temps
- `dockerMinAPI` in [docker.go](./datasources/docker.go) might need tweaking
- `zfs-utils` for zpool status
- [go-check-updates](https://github.com/cosandr/go-check-updates) for updates
- `lm_sensors` for CPU temperatures

## Installation

### Arch Linux

```sh
wget https://raw.githubusercontent.com/cosandr/go-motd/master/PKGBUILD
makepkg -si
```

`go-motd` will use the config file in `/etc/go-motd/config.yaml`.

### Generic

```sh
# Clone repository and cd to it
git clone https://github.com/cosandr/go-motd
cd go-motd
## Manual installation
go mod vendor
go build -a -ldflags "-X main.defaultCfgPath=/etc/go-motd/config.yaml"
# Generate default config
sudo ./go-motd --config /dev/null --dump-config > "default-config.yaml" 2> /dev/null
# Install binary
sudo install -m 755 go-motd /usr/bin/
## Using setup.sh
sudo ./setup.sh install
```

## Running

Two modes of operations, running directly or
as a daemon writing to a file at fixed intervals and triggered by SIGHUP.

### Direct run at login

Assuming it was installed as outlined above, just run the binary by adding `go-motd` in your shell rc file.

### Daemon mode

Recommended usage is running with systemd and pointing `go-motd` at `/etc/motd`.

```
[Unit]
Description=Go MOTD generator

[Service]
PIDFile=/run/go-motd.pid
ExecReload=/usr/bin/kill -s HUP $MAINPID
ExecStart=/usr/bin/go-motd --daemon --pid /run/go-motd.pid --output /etc/motd
```

If it's not showing up, you can add `[[ -s /etc/motd ]] && cat /etc/motd` to your shell rc file.

A refresh can be forced by issuing a SIGHUP to the process, either with `systemctl reload go-motd.service` or
`kill -HUP $(cat /run/go-motd.pid)`

## Configuration

### Global

- `warnings_only` will hide content unless there is a warning, per-module override available
- `show_order` list of enabled modules, they will be displayed in the same order. If not defined, the order in [defaultOrder](./motd.go#L18) will be used.
- `col_def` arrange module output in columns as defined by a 2-dimensional array, configuration for example pictures shown below. Note that this overrides `show_order`.

```yaml
col_def:
  - [sysinfo]
  - [updates]
  - [docker, podman]
  - [systemd]
  - [cpu, disk]
  - [zfs]
  - [btrfs]
```

- `col_pad` number of spaces between columns

### Generic options

All modules implement at least `warnings_only`, `pad_header` and `pad_content`.

- `warnings_only` overrides global setting for that module only
- `pad_header` is a 2-element array of integers, the first represents the number of spaces before the text, the second is spaces after the text, but before `:`
```
# pad_header: [0, 2]
Example  : OK
# pad_header: [2, 0]
  Example: OK
# pad_header: [1, 2]
 Example  : OK
```
- `pad_content` is the same but for details, the padding applies to all lines equally

### CPU temperatures

- `warn`/`crit` are temperatures to consider warning or critical level
- `use_exec` get CPU temperature by parsing `sensors -j` output

### Disk temperatures

- `warn`/`crit` are temperatures to consider warning or critical level
- `ignore` list of disks to ignore (uses names from /dev/)
- `use_sys` will get disk temperatures from `/sys/block` instead of the hddtemp daemon.
The drivetemp kernel module is required.

### Disk usage (BTRFS/ZFS)

- `warn`/`crit` percentage of disk space used before it is considered a warning or critical level, default is 70% and 90% respectively

### BTRFS

- `show_free` show free space instead of used
- `use_exec` get BTRFS filesystem info by parsing `btrfs filesystem usage --raw`
- `sudo` use `sudo btrfs` commands, required for accurate RAID56 data
- `btrfs_cmd` override btrfs command, useful for wrapper scripts. For example:

```
# visudo -f /etc/sudoers.d/btrfs-us
andrei ALL=(root) NOPASSWD: /usr/bin/btrfs-us
# cat /usr/bin/btrfs-us
#!/bin/sh

btrfs filesystem usage $@
```

### Docker

- `ignore` list of ignored container names
- `use_exec` get containers by parsing `docker` command output

### Podman

- `ignore` list of ignored container names
- `sudo` get root containers, you should be able to run `sudo podman` without a password
- `include_sudo` includes both root and rootless containers

### System information

No extra config

### Systemd

- `units` list of monitored units, must include file extension. This option must be set for the module to work.
- `hide_ext` hide the unit file extension when displaying their status
- `inactive_ok` consider inactive units with exit code 0 as being OK, if false they will be considered warnings
- `show_failed` display all failed units, similar to `systemctl --failed`

### Updates

- `show` displays the list of pending updates
- `short_names` use short names for time values (1h5m instead of 1 hour, 5 min)
- `address` listen address of go-check-updates, can be unix socket
- `every` request cache update if it is older than this duration
- `file` path to `go-check-updates` output json, setting this will not use the API at all

## Adding more modules

Basic datasources/example.go

```go
package datasources

import "github.com/cosandr/go-motd/utils"

// These must not occur in the output string itself, if they do, feel free to use your own constants
const (
  examplePadL = "^"  // Default is ^L^
  examplePadR = "&"  // Default is ^R^
)

// Optional, can use ConfBase or ConfBaseWarn
// Recommended to use a struct, even if it only contains one of the base configs
type ConfExample struct {
  ConfBase `yaml:",inline"`
  More bool `yaml:"more"`
}

// Init is mandatory
func (c *ConfExample) Init() {
    // Base init must be called
    c.ConfBase.Init()
    // Can change default padding here
    // Set right padding for header to 2 spaces
    c.PadHeader[1] = 2
    // Set other defaults
    c.More = true
    // Custom padding strings
    c.padL = "^"
    c.padR = "&"
}


func GetExample(ch chan<- SourceReturn, conf *Conf) {
	c := conf.Example
	// Optional, but recommended if you use WarnOnly
	// Check for warnOnly override
	if c.WarnOnly == nil {
		c.WarnOnly = &conf.WarnOnly
	}
	sr := NewSourceReturn(conf.debug)
	defer func() {
		ch <- sr.Return(&c.ConfBase)
	}()
    sr.Header, sr.Content, sr.Error = internalFunc(&c)
    return
}

func internalFunc(c *ConfExample) (header string, content string, err error) {
	// You should return a ModuleNotAvailable error if it is appropriate.
	// Remember to use c.padL/c.padR when preparing header and content
	header = fmt.Sprintf("%s: %s\n", utils.Wrap("Example", c.padL, c.padR), utils.Good("OK"))
	return
}
```

Update common_vars.go
```go
type Conf struct {
  // Add your type to the Conf struct
  Example ConfExample
}

// Update Init()
func (c *Conf) Init() {
  // Init must be called to avoid likely panic 
  // This is caused by uninitialized padding slices if they are not in the config file
  c.Example.Init()
}

// Add to a case to run your function in RunSources
case "example":
    go GetExample(ch, c)
```

Modify main.go (optional)

```go
package main

// Add your module to defaultOrder
var defaultOrder = []string{..., "example"}
```

You may also add an entry to `config.yaml`, this will override what you have set in `Init()`.
