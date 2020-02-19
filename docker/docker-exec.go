package docker

import (
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/cosandr/go-motd/colors"
	mt "github.com/cosandr/go-motd/types"
)

// checkContainersExec returns container status using os/exec, ~5x slower than API
func checkContainersExec(ignoreList []string, failedOnly bool) (header string, content string, err error) {
	var stdout bytes.Buffer
	cmd := exec.Command("docker", "ps", "--format", `"{{.Names}} {{.Status}}"`, "-a")
	cmd.Stdout = &stdout
	err = cmd.Run()
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", mt.Wrap("Docker", padL, padR), colors.Warn("unavailable"))
		return
	}
	// Make set of ignored containers
	var ignoreSet mt.StringSet
	ignoreSet = ignoreSet.FromList(ignoreList)
	// Process output
	var goodCont = make(map[string]string)
	var failedCont = make(map[string]string)
	var sortedNames []string
	for _, c := range strings.Split(stdout.String(), "\n") {
		var tmp = strings.Split(c, " ")
		if len(tmp) < 2 {
			continue
		}
		var cleanName = strings.TrimPrefix(tmp[0], `"`)
		if ignoreSet.Contains(cleanName) {
			continue
		}
		if tmp[1] == "Up" {
			goodCont[cleanName] = tmp[1]
		} else {
			failedCont[cleanName] = tmp[1]
		}
		sortedNames = append(sortedNames, cleanName)
	}
	sort.Strings(sortedNames)

	// Decide what header should be
	if len(goodCont) == 0 {
		header = fmt.Sprintf("%s: %s\n", mt.Wrap("Docker", padL, padR), colors.Err("critical"))
	} else if len(failedCont) == 0 {
		header = fmt.Sprintf("%s: %s\n", mt.Wrap("Docker", padL, padR), colors.Good("OK"))
		if failedOnly {
			return
		}
	} else if len(failedCont) < len(sortedNames) {
		header = fmt.Sprintf("%s: %s\n", mt.Wrap("Docker", padL, padR), colors.Warn("warning"))
	}
	// Only print all containers if requested
	for _, c := range sortedNames {
		if val, ok := goodCont[c]; ok && !failedOnly {
			content += fmt.Sprintf("%s: %s\n", mt.Wrap(c, padL, padR), colors.Good(val))
		} else if val, ok := failedCont[c]; ok {
			content += fmt.Sprintf("%s: %s\n", mt.Wrap(c, padL, padR), colors.Err(val))
		}
	}
	return
}
