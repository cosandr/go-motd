# Installation

Run compiled binary at shell login (run from `~/.zlogin` for example)

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

## Configuration

See structs in [main.go](./main.go) and provided [config.yaml](./config.yaml) for examples. `failedOnly` hides content if there are no problems, usually defined by `warn` and `crit` for that specific module.

## Requirements

- hddtemp daemon is required for disk temps (start `hddtemp.service`)
- Docker API version might need tweaking in [docker.go](./docker/docker.go) (change `client.WithVersion`)

## Todo

- Arrange into columns
- Implement padding on the left
- Select active modules
- Log to file
- Dumb terminal option
- Do something if update cache is out of date
