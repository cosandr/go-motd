package datasources

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/cosandr/go-motd/utils"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

const (
	dockerMinAPI = "1.40"
)

// DockerConf extends CommonConf with a list of containers to ignore
type DockerConf struct {
	CommonConf `yaml:",inline"`
	Exec       bool     `yaml:"useExec"`
	Ignore     []string `yaml:"ignore"`
}

// GetDocker docker container status using the API
func GetDocker(ret chan<- string, c *DockerConf) {
	var header string
	var content string
	if c.Exec {
		header, content, _ = checkContainersExec(c.Ignore, *c.FailedOnly)
	} else {
		header, content, _ = checkContainers(c.Ignore, *c.FailedOnly)
	}
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

func checkContainers(ignoreList []string, failedOnly bool) (header string, content string, err error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion(dockerMinAPI))
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Docker", padL, padR), utils.Warn("unavailable"))
		return
	}

	allContainers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Docker", padL, padR), utils.Warn("unavailable"))
		return
	}
	// Make set of ignored containers
	var ignoreSet utils.StringSet
	ignoreSet = ignoreSet.FromList(ignoreList)
	// Process output
	var goodCont = make(map[string]string)
	var failedCont = make(map[string]string)
	var sortedNames []string
	for _, container := range allContainers {
		var cleanName = strings.TrimPrefix(container.Names[0], "/")
		if ignoreSet.Contains(cleanName) {
			continue
		}
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
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Docker", padL, padR), utils.Err("critical"))
	} else if len(failedCont) == 0 {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Docker", padL, padR), utils.Good("OK"))
		if failedOnly {
			return
		}
	} else if len(failedCont) < len(allContainers) {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Docker", padL, padR), utils.Warn("warning"))
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
