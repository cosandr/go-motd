# Introduction

This project was inspired by [RIKRUS's](https://github.com/RIKRUS/MOTD) and [Hermann Björgvin's](https://github.com/HermannBjorgvin/motd) MOTD scripts.

I've decided to use Go because it is about 10x faster than a similar bash script and it makes for a great first project using the language. In my tests it typically runs in 10-20ms, a similar bash script takes 200-500ms.

The available information will depend on the user privileges, you will need to be able to run (without sudo) `systemctl status`, `docker inspect` and `zpool status` for example.

## Example

Compact:

```sh
Distro    : Fedora 30 (Thirty)
Kernel    : Linux 5.3.16-200.fc30.x86_64
Uptime    : 1 week, 3 days, 7 hours, 20 minutes
Load      : 0.04 [1m], 0.16 [5m], 0.16 [15m]
RAM       : 7.39 GB active of 16.17 GB
Updates   : 6 pending, checked 88 minutes ago
Systemd   : OK
Docker    : OK
Disk temp : OK
CPU temp  : OK
ZFS       : OK
```

Showing everything (cropped):

```sh
Distro    : Fedora 30 (Thirty)
Kernel    : Linux 5.3.16-200.fc30.x86_64
Uptime    : 1 week, 3 days, 7 hours, 28 minutes
Load      : 0.10 [1m], 0.17 [5m], 0.17 [15m]
RAM       : 7.36 GB active of 16.17 GB
Updates   : 6 pending, checked 95 minutes ago
  -> pgdg-fedora-repo.noarch [42.0-6]
  -> pgdg-fedora-repo.noarch [42.0-6]
...
Systemd   : OK
check-nginx-modules.service : success
firewalld.service           : active
...
Docker    : OK
bitwarden     : running
cloudflare-ac : running
...
Disk temp : OK
/dev/sda  : 27
/dev/sdb  : 29
...
CPU temp  : OK
Core 0    : 43
Core 1    : 45
...
ZFS       : OK
tank      : ONLINE, 5.71 TB used out of 10.91 TB
```

## Installation

### Arch Linux

See [PKGBUILD](./PKGBUILD)

`go-motd` will use the config file in `/etc/go-motd/config.yaml`, you probably want to run it at shell login.

### Generic

Build and run the binary at shell login. It is likely necessary to provide a config path, by default it must be in the same directory as the binary.

Example line in `~/.zshrc`

```sh
~/go/bin/go-motd -cfg ~/go/src/github.com/cosandr/go-motd/config.yaml
```

## Requirements

- hddtemp daemon is required for disk temps (start `hddtemp.service`)
- Docker API version might need tweaking in [docker.go](./docker/docker.go) (change `dockerMinAPI`)
- `zfs-utils` for zpool status
- [go-check-updates](https://github.com/cosandr/go-check-updates) for updates

## Configuration

See structs in [main.go](./main.go) and provided [config.yaml](./config.yaml) for examples. `failedOnly` hides content if there are no problems, usually defined by `warn` and `crit` for that specific module.

Each module has a `header` and `content` array, the first value is the left padding (before the string starts) and the second is padding after the string but before some kind of delimiter (usually a semicolon). The padding for the header (usually `Module: STATUS`) is adjustable separately from its content. `failedOnly` can be set on a per-module basis, if present it will override the global option.

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

func Get(ret *string, c *Conf) {
  header, content, err := internalFunc(c.More)
  // Initialize Pad
  var p = mt.Pad{Delims: map[string]int{padL: c.Header[0], padR: c.Header[1]}, Content: header}
  // Do() replaces the keys of the `Pad.Delims` map with value amount of spaces
  // For example `"$": 3` will replace `$` with 3 spaces.
  header = p.Do()
  // Repeat for content, reassign p and run p.Do again
  *ret = header
}

func internalFunc(more bool) (header string, content string, err error) {}
```

Modify main.go

```go
import "demo/module"

// Add your type to the Conf struct
type Conf struct {
  ...
  Module module.Conf
}

// Create WaitGroup compatible method
func getModule(ret *string, c Conf, wg *sync.WaitGroup, timing bool) {
  // You may do default checking here, see getZFS as an example
  module.Get(ret, &c.Module)
  wg.Done()
}

// Call method from main
func main() {
  ...
  var moduleStr string
  wg.Add(1)
  go getModule(&moduleStr, *c, &wg, timing)
}

// Update NewConf()
func NewConf() *Conf {
  ...
  c.Module.Common.Init()
  // Set custom defaults
  c.Module.More = true
}
```

Finally add entry to `config.yaml`. It will likely crash if the config file and struct mismatch.

## Todo

- Arrange into columns
- Select active modules
- Log to file
- Dumb terminal option
- Do something if update cache is out of date
- Parse `sensors` output directly to remove gopsutil dependency
- Try channels instead of `WaitGroup`
