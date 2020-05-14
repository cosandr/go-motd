package datasources

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/cosandr/go-motd/utils"
)

// SystemdConf extends CommonConf with a list of units to monitor
type SystemdConf struct {
	CommonConf `yaml:",inline"`
	Units      []string `yaml:"units"`
	HideExt    bool     `yaml:"hideExt"`
}

// GetSystemd gets systemd unit status using dbus
func GetSystemd(ret chan<- string, c *SystemdConf) {
	header, content, _ := getServiceStatus(c.Units, *c.FailedOnly, c.HideExt)
	// Pad header
	var p = utils.Pad{Delims: map[string]int{padL: c.Header[0], padR: c.Header[1]}, Content: header}
	header = p.Do()
	if len(content) == 0 {
		ret <- header
		return
	}
	// Pad container list
	p = utils.Pad{Delims: map[string]int{padL: c.Content[0], padR: c.Content[1]}, Content: content}
	content = p.Do()
	ret <- header + "\n" + content
}

// getServiceStatus get service properties
func getServiceStatus(units []string, failedOnly bool, hideExt bool) (header string, content string, err error) {
	con, err := dbus.New()
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Systemd", padL, padR), utils.Err("DBus failed"))
		return
	}
	defer con.Close()
	if len(units) == 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Systemd", padL, padR), utils.Warn("unconfigured"))
		return
	}
	getProps := []string{"ActiveState", "Result", "ExecMainStatus", "LoadState"}
	sort.Strings(units)
	var errStr = ""
	// Map of maps to hold properties
	var unitProps = map[string]map[string]string{}
	for _, u := range units {
		unitProps[u] = map[string]string{}
		// GetSystemd and store all properties
		props, err := con.GetAllProperties(u)
		if err != nil {
			errStr += fmt.Sprintf("Failed to get properties for %s: %s\n", u, err)
			continue
		}
		for _, p := range getProps {
			if data, ok := props[p].(string); ok {
				unitProps[u][p] = data
			} else if data, ok := props[p].(int32); ok {
				unitProps[u][p] = strconv.Itoa(int(data))
			} else {
				errStr += fmt.Sprintf("Unrecognized type for %s\n", props[p])
			}
		}
	}
	// Maps to make checking easier later
	var failedUnits = map[string]string{}
	var goodUnits = map[string]string{}
	// Loop through units so it is alphabetical
	for _, unit := range units {
		var stat = unitProps[unit]
		// Skip if we have no stats
		if len(stat) == 0 {
			continue
		}
		wrapped := utils.Wrap(unit, padL, padR)
		if hideExt {
			// Remove all systemd extensions
			re := regexp.MustCompile(`(\.service|\.socket|\.device|\.mount|\.automount|\.swap|\.target|\.path|\.timer|\.slice|\.scope)`)
			wrapped = re.ReplaceAllString(wrapped, "")
		}
		// No such unit file
		if stat["LoadState"] != "loaded" {
			failedUnits[unit] = fmt.Sprintf("%s: %s\n", wrapped, utils.Err(stat["LoadState"]))
		} else {
			// Service running
			if stat["ActiveState"] == "active" {
				goodUnits[unit] = fmt.Sprintf("%s: %s\n", wrapped, utils.Good(stat["ActiveState"]))
			} else {
				// Not running but existed successfully
				if stat["ExecMainStatus"] == "0" {
					goodUnits[unit] = fmt.Sprintf("%s: %s\n", wrapped, utils.Good(stat["Result"]))
					// Not running and failed
				} else {
					failedUnits[unit] = fmt.Sprintf("%s: %s\n", wrapped, utils.Err(stat["ActiveState"]))
				}
			}
		}
	}
	// Decide what header should be
	// Only print all services if requested
	if len(goodUnits) == 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Systemd", padL, padR), utils.Err("critical"))
	} else if len(failedUnits) == 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Systemd", padL, padR), utils.Good("OK"))
		if failedOnly {
			return
		}
	} else if len(failedUnits) < len(units) {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Systemd", padL, padR), utils.Warn("warning"))
	}
	// Print all in order
	for _, unit := range units {
		if val, ok := goodUnits[unit]; ok && !failedOnly {
			content += val
		} else if val, ok := failedUnits[unit]; ok {
			content += val
		}
	}
	if len(errStr) > 0 {
		content += errStr
	}
	return
}
