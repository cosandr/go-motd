package datasources

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/cosandr/go-motd/utils"
)

// ConfSystemd extends ConfBase with a list of units to monitor
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

// Init sets ShowFailed to true
func (c *ConfSystemd) Init() {
	c.ConfBase.Init()
	c.PadHeader[1] = 2
	c.ShowFailed = true
}

type systemdUnit struct {
	Name           string
	ActiveState    string
	Result         string
	ExecMainStatus string
	LoadState      string
}

// IsEmpty returns true if we have no information about unit state
func (s *systemdUnit) IsEmpty() bool {
	return s.ActiveState == "" && s.Result == "" && s.ExecMainStatus == "" && s.LoadState == ""
}

// GetProperties gets this unit's properties from DBus
func (s *systemdUnit) GetProperties(con *dbus.Conn) (err error) {
	// Do nothing if we already have everything
	if s.ActiveState != "" && s.Result != "" && s.ExecMainStatus != "" && s.LoadState != "" {
		return
	}
	props, err := con.GetUnitProperties(s.Name)
	if err != nil {
		return
	}
	if s.ActiveState == "" {
		if data, ok := props["ActiveState"].(string); ok {
			s.ActiveState = data
		}
	}
	if s.Result == "" {
		if data, ok := props["Result"].(string); ok {
			s.Result = data
		}
	}
	if s.ExecMainStatus == "" {
		if data, ok := props["ExecMainStatus"].(int32); ok {
			s.ExecMainStatus = strconv.Itoa(int(data))
		}
	}
	if s.LoadState == "" {
		if data, ok := props["LoadState"].(string); ok {
			s.LoadState = data
		}
	}
	return
}

// GetSystemd gets systemd unit status using dbus
func GetSystemd(ch chan<- SourceReturn, conf *Conf) {
	c := conf.Systemd
	// Check for *c.WarnOnly override
	if c.WarnOnly == nil {
		c.WarnOnly = &conf.WarnOnly
	}
	sr := NewSourceReturn(conf.debug)
	defer func() {
		ch <- sr.Return(&c.ConfBase)
	}()
	sr.Header, sr.Content, sr.Error = getServiceStatus(&c)
	return
}

// getServiceStatus get service properties
func getServiceStatus(c *ConfSystemd) (header string, content string, err error) {
	con, err := dbus.New()
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Systemd", c.padL, c.padR), utils.Err("DBus failed"))
		return
	}
	defer con.Close()
	// No units to check and didn't request to show failed
	if len(c.Units) == 0 && !c.ShowFailed {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Systemd", c.padL, c.padR), utils.Warn("unconfigured"))
		return
	}
	units := make([]systemdUnit, 0)
	if c.ShowFailed {
		// Get all failed
		listFailed, _ := con.ListUnitsFiltered([]string{"failed"})
		if len(listFailed) > 0 {
			for _, u := range listFailed {
				units = append(units, systemdUnit{
					Name:        u.Name,
					ActiveState: u.ActiveState,
					LoadState:   u.LoadState,
				})
			}
		}
	}
	if len(c.Units) > 0 {
		for _, name := range c.Units {
			units = append(units, systemdUnit{
				Name: name,
			})
		}
	}
	var errStr = ""
	// Get missing properties
	for i := range units {
		err = units[i].GetProperties(con)
		if err != nil {
			errStr += fmt.Sprintf("Failed to get properties for %s: %s\n", units[i].Name, err)
			err = nil
		}
	}
	// Map of maps to hold properties
	sort.Slice(units, func(i, j int) bool {
		return units[i].Name < units[j].Name
	})
	// Maps to make checking easier later
	var failedUnits = map[string]string{}
	var goodUnits = map[string]string{}
	// Loop through units so it is alphabetical
	for _, u := range units {
		// Skip if we have no stats
		if u.IsEmpty() {
			continue
		}
		wrapped := utils.Wrap(u.Name, c.padL, c.padR)
		if c.HideExt {
			// Remove all systemd extensions
			re := regexp.MustCompile(`(\.service|\.socket|\.device|\.mount|\.automount|\.swap|\.target|\.path|\.timer|\.slice|\.scope)`)
			wrapped = re.ReplaceAllString(wrapped, "")
		}
		// No such unit file
		if u.LoadState != "loaded" {
			failedUnits[u.Name] = fmt.Sprintf("%s: %s\n", wrapped, utils.Err(u.LoadState))
		} else {
			// Service running
			if u.ActiveState == "active" {
				goodUnits[u.Name] = fmt.Sprintf("%s: %s\n", wrapped, utils.Good(u.ActiveState))
			} else {
				// Not running but existed successfully
				if u.ExecMainStatus == "0" {
					if c.InactiveOK {
						goodUnits[u.Name] = fmt.Sprintf("%s: %s\n", wrapped, utils.Good(u.Result))
					} else {
						failedUnits[u.Name] = fmt.Sprintf("%s: %s\n", wrapped, utils.Warn(u.ActiveState))
					}
					// Not running and failed
				} else {
					failedUnits[u.Name] = fmt.Sprintf("%s: %s\n", wrapped, utils.Err(u.ActiveState))
				}
			}
		}
	}
	// Decide what header should be
	// Only print all services if requested
	if len(goodUnits) == 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Systemd", c.padL, c.padR), utils.Err("critical"))
	} else if len(failedUnits) == 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Systemd", c.padL, c.padR), utils.Good("OK"))
		if *c.WarnOnly {
			return
		}
	} else if len(failedUnits) < len(units) {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Systemd", c.padL, c.padR), utils.Warn("warning"))
	}
	// Print all in order
	for _, u := range units {
		if val, ok := goodUnits[u.Name]; ok && !*c.WarnOnly {
			content += val
		} else if val, ok := failedUnits[u.Name]; ok {
			content += val
		}
	}
	if len(errStr) > 0 {
		content += errStr
	}
	return
}
