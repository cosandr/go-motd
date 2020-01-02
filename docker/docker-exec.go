// +build test

package docker

import (
	"os/exec"
	"fmt"
	"bytes"
	"text/tabwriter"
	"sort"
	"strings"

	"github.com/cosandr/go-motd/colors"
)

// CheckContainersExec returns container status using os/exec, ~5x slower than API
func CheckContainersExec(buf *bytes.Buffer, containers []string, padDockerHeader int, padDockerContent int, failedOnly bool) {
	w := tabwriter.NewWriter(buf, 0, 0, padDockerHeader, ' ', 0)
	var stdout bytes.Buffer
	cmd := exec.Command("docker", "ps", "--format", `"{{.Names}} {{.Status}}"`, "-a")
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(w, "Docker\t: %s\n", colors.Warn("unavailable"))
		fmt.Println(err)
		w.Flush()
		return
	}
	var goodCont = make(map[string]string)
	var failedCont = make(map[string]string)
	var sortedNames []string
	for _, c := range strings.Split(stdout.String(), "\n") {
		var tmp = strings.Split(c, " ")
		if len(tmp) < 2 { continue }
		var cleanName = strings.TrimPrefix(tmp[0], `"`)
		if tmp[1] == "Up" {
			goodCont[cleanName] = tmp[1]
		} else {
			failedCont[cleanName] = tmp[1]
		}
		sortedNames = append(sortedNames, cleanName)
	}
	sort.Strings(sortedNames)

	// Decide what header should be
	// Only print all containers if requested
	if len(goodCont) == 0 {
		fmt.Fprintf(w, "Docker\t: %s\n", colors.Err("critical"))
		w.Flush()
	} else if len(failedCont) == 0 {
		fmt.Fprintf(w, "Docker\t: %s\n", colors.Good("OK"))
		w.Flush()
		if failedOnly { return }
	} else if len(failedCont) < len(sortedNames) {
		fmt.Fprintf(w, "Docker\t: %s\n", colors.Warn("warning"))
		w.Flush()
	}
	w = tabwriter.NewWriter(buf, 0, 0, padDockerContent, ' ', 0)
	// Print all in order
	for _, c := range sortedNames {
		if val, ok := goodCont[c]; ok && !failedOnly {
			fmt.Fprintf(w, "%s\t: %s\n", c, colors.Good(val))
		} else if val, ok := failedCont[c]; ok {
			fmt.Fprintf(w, "%s\t: %s\n", c, colors.Err(val))
		}
	}
	w.Flush()
}