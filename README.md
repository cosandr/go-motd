
[![Go Report Card](https://goreportcard.com/badge/github.com/cosandr/go-motd)](https://goreportcard.com/report/github.com/cosandr/go-motd) [![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/cosandr/go-motd/blob/master/LICENSE)

# Introduction

This project was inspired by [RIKRUS's](https://github.com/RIKRUS/MOTD) and [Hermann Björgvin's](https://github.com/HermannBjorgvin/motd) MOTD scripts.

I've decided to use Go because it is about 10x faster than a similar bash script and it makes for a great first project using the language. In my tests it typically runs in 10-20ms, a similar bash script takes 200-500ms.

The available information will depend on the user privileges, you will need to be able to run (without sudo) `systemctl status`, `docker inspect` and `zpool status` for example.

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

- hddtemp daemon is required for disk temps (start `hddtemp.service`)
- [dockerMinAPI](./docker/docker.go#L16) might need tweaking
- `zfs-utils` for zpool status
- [go-check-updates](https://github.com/cosandr/go-check-updates) for updates
- `lm_sensors` for CPU temperatures

## Configuration

### Global

- `failedOnly` will hide content unless there is a warning, per-module override available
- `showOrder` list of enabled modules, they will be displayed in the same order. If not defined, the order in [defaultOrder](./main.go#L22) will be used.
- `colDef` arrange module ouput in columns as defined by a 2-dimensional array, configuration for example pictures shown below. Note that this overrides `showOrder`.

```yaml
colDef:
  - [sysinfo]
  - [updates]
  - [docker, systemd]
  - [cpu, disk]
  - [zfs]
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

### Docker

- `ignore` list of ignored container names

### Systemd

- `units` list of monitored units, must include file extension. This option must be set for the module to work.
- `hideExt` hide the unit file extension when displaying their status

## Adding more modules

Basic module.go

```go
package module

import (
  mt "github.com/cosandr/go-motd/types"
)

// Choice is arbitrary, but they must not be the same or appear in the content itself
const (
  padL = "$"
  padR = "%"
)

// Optional, can use mt.Common or mt.CommonWithWarn
type Conf struct {
  mt.Common `yaml:",inline"`
  More bool `yaml:"more"`
}

func Get(ret chan<- string, c *Conf) {
  header, content, err := internalFunc(c.More)
  // Initialize Pad
  var p = mt.Pad{Delims: map[string]int{padL: c.Header[0], padR: c.Header[1]}, Content: header}
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
import "demo/module"

// Add your module to defaultOrder
var defaultOrder = []string{..."module"}

// Add your type to the Conf struct
type Conf struct {
  ...
  Module module.Conf
}

// Update Init()
func (c *Conf) Init() {
  ...
  c.Module.Common.Init()
  // Set custom defaults
  c.Module.More = true
}

// Create Get method
func getModule(ret chan<- string, c Conf, endTime chan<- time.Time) {
  // You may do default checking here, see getZFS as an example
  module.Get(ret, &c.Module)
  endTime <- time.Now()
}

// Add to main
func main() {
  ...
  // Add a case for it
  for _, k := range printOrder {
    switch k {
      ...
    case "module":
      go getModule(outCh[k], c, endTimes[k])
    }
  }
}
```

You may also add an entry to `config.yaml`, this will override what you have set in `Init()`.

## Todo

- Log to file
- Dumb terminal option
- Do something if update cache is out of date
