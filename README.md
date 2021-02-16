
[![Go Report Card](https://goreportcard.com/badge/github.com/cosandr/go-motd)](https://goreportcard.com/report/github.com/cosandr/go-motd) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/cosandr/go-motd/blob/master/LICENSE)

# Introduction

This project was inspired by [RIKRUS's](https://github.com/RIKRUS/MOTD) and [Hermann Björgvin's](https://github.com/HermannBjorgvin/motd) MOTD scripts.

I've decided to use Go because it is about 10x faster than a similar bash script and it makes for a great first project using the language. In my tests it typically runs in 10-20ms, a similar bash script takes 200-500ms.

The available information will depend on the user privileges, you will need to be able to run (without sudo) `systemctl status`, `docker ps` and `zpool status` for example.

Note that the BTRFS and ZFS space statistics are totals, that is to say, a RAID5 setup shows the used/total space across all drives. For example 3x4TB disks in RAIDZ1 show 10.91TB total, not the usable space which is about 7TB.

You can dump the default config by passing an invalid path as the `-cfg` argument and using `-dump-config` at the same time.

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

## Installation

### Arch Linux

```sh
wget https://raw.githubusercontent.com/cosandr/go-motd/master/PKGBUILD
makepkg -si
```

`go-motd` will use the config file in `/etc/go-motd/config.yaml`, you probably want to run it at shell login.

### Generic

Build and run the binary at shell login. It is likely necessary to provide a config path, by default it must be in the same directory as the binary.

Example line in `~/.zshrc`

```sh
~/go/bin/go-motd -cfg ~/go/src/github.com/cosandr/go-motd/config.yaml
```

## Requirements

- Kernel 5.6+ (drivetemp module) or hddtemp daemon are required for disk temps
- `dockerMinAPI` in [docker.go](./datasources/docker.go) might need tweaking
- `zfs-utils` for zpool status
- [go-check-updates](https://github.com/cosandr/go-check-updates) for updates
- `lm_sensors` for CPU temperatures

## Configuration

### Global

- `warnings_only` will hide content unless there is a warning, per-module override available
- `show_order` list of enabled modules, they will be displayed in the same order. If not defined, the order in [defaultOrder](./motd.go#L18) will be used.
- `col_def` arrange module ouput in columns as defined by a 2-dimensional array, configuration for example pictures shown below. Note that this overrides `show_order`.

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

Basic example.go

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
    // Highly recommended to call init on base
    c.ConfBase.Init()
    // Can change default padding here
    // Set right padding for header to 2 spaces
    c.PadHeader[1] = 2
    // Set other defaults
    c.More = true
}


func GetExample(ret chan<- string, c *ConfExample) {
  header, content, err := internalFunc(c.More)
  // Initialize Pad
  var p = utils.Pad{Delims: map[string]int{examplePadL: c.PadHeader[0], examplePadR: c.PadHeader[1]}, Content: header}
  // Do() replaces the keys of the `Pad.Delims` map with value amount of spaces
  // For example `"$": 3` will replace `$` with 3 spaces.
  header = p.Do()
  // Repeat for content, reassign p and run p.Do again
  ret <- header
}

func internalFunc(more bool) (header string, content string, err error) {}
```

Modify main.go

```go
package main

// Add your module to defaultOrder
var defaultOrder = []string{..., "example"}

type Conf struct {
  // Add your type to the Conf struct
  Example datasources.ConfExample
}

// Update Init()
func (c *Conf) Init() {
  // Init must be called to avoid likely panic 
  // This is caused by uninitialized padding slices if they are not in the config file
  c.Example.Init()
}

// Create Get method
func getExample(ret chan<- string, c Conf, endTime chan<- time.Time) {
  // You may do default checking here, see getZFS as an example
  datasources.GetExample(ret, &c.Module)
  endTime <- time.Now()
}

// Add to main
func main() {
  // Add a case for it
  for _, k := range printOrder {
    switch k {
    case "example":
      go getExample(outCh[k], c, endTimes[k])
    }
  }
}
```

You may also add an entry to `config.yaml`, this will override what you have set in `Init()`.

## Todo

- Log to file
- Dumb terminal option
- Do something if update cache is out of date
