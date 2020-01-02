package systemd

import (
	"fmt"
	"github.com/coreos/go-systemd/dbus"
	"strconv"
	"sort"
	"github.com/cosandr/go-motd/colors"
)

// GetConn returns new dbus connection
func GetConn() (con *dbus.Conn) {
	con, err := dbus.New()
	if err != nil {
		panic(err)
	}
	return con
}

// CloseConn closes dbus connetion
func CloseConn(con *dbus.Conn) {
	con.Close()
}

// GetServiceStatus get service properties
func GetServiceStatus(con *dbus.Conn, units []string, failedOnly bool) (header string, content string, err error) {
	if len(units) == 0 {
		header = fmt.Sprintf("Systemd\t: %s\n", colors.Warn("unconfigured"))
		return
	}
	getProps := []string{"ActiveState", "Result", "ExecMainStatus", "LoadState"}
	sort.Strings(units)
	var errStr string = ""
	// Map of maps to hold properties
	var unitProps = map[string]map[string]string{}
	for _, u := range units {
		unitProps[u] = map[string]string{}
		// Get and store all properties
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
		if len(stat) == 0 { continue }
		// No such unit file
		if stat["LoadState"] != "loaded" {
			failedUnits[unit] = fmt.Sprintf("%s\t: %s\n", unit, colors.Err(stat["LoadState"]))
		} else {
			// Service running
			if stat["ActiveState"] == "active" {
				goodUnits[unit] = fmt.Sprintf("%s\t: %s\n", unit, colors.Good(stat["ActiveState"]))
			} else {
				// Not running but existed sucessfully
				if stat["ExecMainStatus"] == "0" {
					goodUnits[unit] = fmt.Sprintf("%s\t: %s\n", unit, colors.Good(stat["Result"]))
				// Not running and failed
				} else {
					failedUnits[unit] = fmt.Sprintf("%s\t: %s\n", unit, colors.Err(stat["ActiveState"]))
				}
			} 
		}
	}
	// Decide what header should be
	// Only print all services if requested
	if len(goodUnits) == 0 {
		header = fmt.Sprintf("Systemd\t: %s\n", colors.Err("critical"))
	} else if len(failedUnits) == 0 {
		header = fmt.Sprintf("Systemd\t: %s\n", colors.Good("OK"))
		if failedOnly { return }
	} else if len(failedUnits) < len(units) {
		header = fmt.Sprintf("Systemd\t: %s\n", colors.Warn("warning"))
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