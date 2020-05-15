package datasources

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cosandr/go-motd/utils"
)

type containerStatus struct {
	Name   string
	Status string
}

type containerList struct {
	Runtime    string
	Root       bool
	Containers []containerStatus
}

func (cl *containerList) toHeaderContent(ignoreList []string, failedOnly bool) (header string, content string, err error) {
	// Make set of ignored containers
	var ignoreSet utils.StringSet
	ignoreSet = ignoreSet.FromList(ignoreList)
	// Process output
	var goodCont = make(map[string]string)
	var failedCont = make(map[string]string)
	var sortedNames []string
	for _, c := range cl.Containers {
		if ignoreSet.Contains(c.Name) {
			continue
		}
		status := strings.ToLower(c.Status)
		if status == "up" || status == "created" || status == "running" {
			goodCont[c.Name] = status
		} else {
			failedCont[c.Name] = status
		}
		sortedNames = append(sortedNames, c.Name)
	}
	sort.Strings(sortedNames)

	// Decide what header should be
	if len(goodCont) == 0 && len(sortedNames) > 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap(cl.Runtime, padL, padR), utils.Err("critical"))
	} else if len(failedCont) == 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap(cl.Runtime, padL, padR), utils.Good("OK"))
		if failedOnly {
			return
		}
	} else if len(failedCont) < len(sortedNames) {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap(cl.Runtime, padL, padR), utils.Warn("warning"))
	}
	// Only print all containers if requested
	for _, c := range sortedNames {
		if val, ok := goodCont[c]; ok && !failedOnly {
			content += fmt.Sprintf("%s: %s\n", utils.Wrap(c, padL, padR), utils.Good(val))
		} else if val, ok := failedCont[c]; ok {
			content += fmt.Sprintf("%s: %s\n", utils.Wrap(c, padL, padR), utils.Err(val))
		}
	}
	return
}
