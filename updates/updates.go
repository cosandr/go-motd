package updates

import (
	"fmt"
	"time"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"github.com/cosandr/go-motd/colors"
	"github.com/cosandr/go-check-updates/types"
)

func readUpdateCache(cacheFp string) (parsed types.YamlT, err error) {
	yamlFile, err := ioutil.ReadFile(cacheFp)
	if err != nil {
		return
	}
	err = yaml.Unmarshal(yamlFile, &parsed)
	if err != nil {
		return
	}
	return
}

func timeStr(d time.Duration) string {
	if d.Hours() > 48 {
		return fmt.Sprintf("%.0f days", d.Hours()/24)
	}
	if d.Minutes() > 120 {
		return fmt.Sprintf("%.0f hours", d.Hours())
	}
	return fmt.Sprintf("%.0f minutes", d.Minutes())
}

// Get reads cached updates file and formats it
func Get(cacheFp string, checkDur time.Duration) (header string, content string, err error) {
	parsed, err := readUpdateCache(cacheFp)
	// TODO: Run go-check-updates and return
	if err != nil {
		header = fmt.Sprintf("Updates\t: %s\n", colors.Warn("unavailable"))
		return
	}
	var timeElapsed = time.Since(parsed.Checked)
	// TODO
	if timeElapsed > checkDur {
		// Run in bg
		// header = fmt.Sprintf("Updates\t: %s\n", colors.Warn("checking"))
		// return
	}
	for _, u := range parsed.Updates {
		content += fmt.Sprintf("\t-> %s [%s]\n", u.Pkg, u.NewVer)
	}
	header = fmt.Sprintf("Updates\t: %d pending, checked %s ago\n", len(parsed.Updates), timeStr(timeElapsed))
	return
}
