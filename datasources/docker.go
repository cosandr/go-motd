package datasources

import (
	"context"
	"fmt"
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
	var err error
	var cl containerList
	var header string
	var content string
	if c.Exec {
		cl, err = getContainersExec(false, false)
	} else {
		cl, err = getDockerContainers()
	}
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Docker", padL, padR), utils.Warn("unavailable"))
	} else {
		header, content, _ = cl.toHeaderContent(c.Ignore, *c.FailedOnly)
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

func getDockerContainers() (cl containerList, err error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithVersion(dockerMinAPI))
	if err != nil {
		return
	}

	allContainers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return
	}
	cl.Runtime = "Docker"
	cl.Root = true
	for _, container := range allContainers {
		cl.Containers = append(cl.Containers, containerStatus{
			Name:   strings.TrimPrefix(container.Names[0], "/"),
			Status: container.State,
		})
	}
	return
}
