
[![Go Report Card](https://goreportcard.com/badge/github.com/cosandr/go-motd)](https://goreportcard.com/report/github.com/cosandr/go-motd) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/cosandr/go-motd/blob/master/LICENSE)

# Introduction

This project was inspired by [RIKRUS's](https://github.com/RIKRUS/MOTD) and [Hermann Björgvin's](https://github.com/HermannBjorgvin/motd) MOTD scripts.

I've decided to use Go because it is about 10x faster than a similar bash script and it makes for a great first project using the language. In my tests it typically runs in 10-20ms, a similar bash script takes 200-500ms.

The available information will depend on the user privileges, you will need to be able to run (without sudo) `systemctl status`, `docker inspect` and `zpool status` for example.

Note that the BTRFS and ZFS space statistics are totals, that is to say, a RAID5 setup shows the used/total space across all drives. For example 3x4TB disks in RAIDZ1 show 10.91TB total, not the usable space which is about 7TB.

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
- [dockerMinAPI](./datasources/docker.go#L15) might need tweaking
- `zfs-utils` for zpool status
- [go-check-updates](https://github.com/cosandr/go-check-updates) for updates
- `lm_sensors` for CPU temperatures

## Configuration

### Global

- `failedOnly` will hide content unless there is a warning, per-module override available
- `showOrder` list of enabled modules, they will be displayed in the same order. If not defined, the order in [defaultOrder](./main.go#L18) will be used.
- `colDef` arrange module ouput in columns as defined by a 2-dimensional array, configuration for example pictures shown below. Note that this overrides `showOrder`.

```yaml
colDef:
  - [sysinfo]
  - [updates]
  - [docker, systemd]
  - [cpu, disk]
  - [zfs]
  - [btrfs]
```

- `colPad` number of spaces between columns

### Generic options

All modules implement at least `header`/`content`.

- `header`/`content` arrays define padding, first element is padding to the left (of the module name) and second to the right, before the semicolon (useful for aligning vertically)
- `warn`/`crit` unit depends on the module, for CPU/Disk temperatures it is degrees celsius, for ZFS pools it is % used

### Updates

- `show` displays the list of pending updates
- `file` path to `go-check-updates` output yaml
- `check` refresh updates if the file hasn't been updated for this long (not implemented)

### Disk temperatures

- `useSys` will get disk temperatures from `/sys/block` instead of the hddtemp daemon.
The drivetemp kernel module is required.

### Docker

- `ignore` list of ignored container names

### Systemd

- `units` list of monitored units, must include file extension. This option must be set for the module to work.
- `hideExt` hide the unit file extension when displaying their status

## Adding more modules

Basic example.go

```go
package datasources

import "github.com/cosandr/go-motd/utils"

// These must not occur in the output string itself, if they do, feel free to use your own constants
const (
  examplePadL = "^"  // Default is $
  examplePadR = "&"  // Default is %
)

// Optional, can use CommonConf or CommonWithWarnConf
type ExampleConf struct {
  CommonConf `yaml:",inline"`
  More bool `yaml:"more"`
}

func GetExample(ret chan<- string, c *ExampleConf) {
  header, content, err := internalFunc(c.More)
  // Initialize Pad
  var p = utils.Pad{Delims: map[string]int{examplePadL: c.Header[0], examplePadR: c.Header[1]}, Content: header}
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
  Example datasources.ExampleConf
}

// Update Init()
func (c *Conf) Init() {
  // Init should be called, but adding defaults is optional
  c.Example.CommonConf.Init()
  // Set custom defaults
  c.Example.More = true
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
