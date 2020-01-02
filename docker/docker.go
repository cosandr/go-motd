package docker

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/cosandr/go-motd/colors"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type stringSet map[string]struct{}
var empty struct{}

func (s stringSet) Contains(v string) bool {
	_, ok := s[v]
	return ok
}

func (s stringSet) FromList(listIn []string) stringSet {
	p := make(stringSet)
	for _, val := range listIn {
		p[val] = empty
	}
	return p
}

// CheckContainers using Docker API
func CheckContainers(ignoreList []string, failedOnly bool) (header string, content string, err error) {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion("1.40"))
	if err != nil {
		header = fmt.Sprintf("Docker\t: %s\n", colors.Warn("unavailable"))
		return
	}

	allContainers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		header = fmt.Sprintf("Docker\t: %s\n", colors.Warn("unavailable"))
		return
	}
	// Make set of ignored containers
	var ignoreSet stringSet
	ignoreSet = ignoreSet.FromList(ignoreList)
	// Process output
	var goodCont = make(map[string]string)
	var failedCont = make(map[string]string)
	var sortedNames []string
	for _, container := range allContainers {
		var cleanName = strings.TrimPrefix(container.Names[0], "/")
		if ignoreSet.Contains(cleanName) { continue }
		if container.State != "running" {
			failedCont[cleanName] = container.State
		} else {
			goodCont[cleanName] = container.State
		}
		sortedNames = append(sortedNames, cleanName)
	}
	sort.Strings(sortedNames)

	// Decide what header should be
	if len(goodCont) == 0 {
		header = fmt.Sprintf("Docker\t: %s\n", colors.Err("critical"))
	} else if len(failedCont) == 0 {
		header = fmt.Sprintf("Docker\t: %s\n", colors.Good("OK"))
		if failedOnly { return }
	} else if len(failedCont) < len(allContainers) {
		header = fmt.Sprintf("Docker\t: %s\n", colors.Warn("warning"))
	}
	// Only print all containers if requested
	for _, c := range sortedNames {
		if val, ok := goodCont[c]; ok && !failedOnly {
			content += fmt.Sprintf("%s\t: %s\n", c, colors.Good(val))
		} else if val, ok := failedCont[c]; ok {
			content += fmt.Sprintf("%s\t: %s\n", c, colors.Err(val))
		}
	}
	return
}
