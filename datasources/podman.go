package datasources

import (
	"fmt"
	"os/user"

	"github.com/cosandr/go-motd/utils"
)

// PodmanConf extends CommonConf with a list of containers to ignore
type PodmanConf struct {
	CommonConf  `yaml:",inline"`
	Sudo        bool     `yaml:"sudo"`
	IncludeSudo bool     `yaml:"includeSudo"`
	Ignore      []string `yaml:"ignore"`
}

// GetPodman podman container status by parsing cli output
func GetPodman(ret chan<- string, c *PodmanConf) {
	var header string
	var content string
	// Check if we are root
	runningUser, err := user.Current()
	if err == nil && runningUser.Uid == "0" {
		// Do not run sudo as root, there's no point
		c.IncludeSudo = false
		c.Sudo = false
	}
	if !c.IncludeSudo {
		cl, err := getContainersExec(true, c.Sudo)
		if err != nil {
			header = fmt.Sprintf("%s: %s\n", utils.Wrap("Podman", padL, padR), utils.Warn("unavailable"))
		} else {
			header, content, _ = cl.toHeaderContent(c.Ignore, *c.FailedOnly)
		}
	} else {
		clUser, errUser := getContainersExec(true, false)
		clRoot, errRoot := getContainersExec(true, true)
		// Combine lists for now
		cl := containerList{Runtime: "Podman", Root: true}
		// Add # in front of root containers
		for _, c := range clRoot.Containers {
			cl.Containers = append(cl.Containers, containerStatus{
				Name:   "# " + c.Name,
				Status: c.Status,
			})
		}
		// Add $ in front of user containers
		for _, c := range clUser.Containers {
			cl.Containers = append(cl.Containers, containerStatus{
				Name:   "$ " + c.Name,
				Status: c.Status,
			})
		}
		if len(cl.Containers) == 0 && (errUser != nil || errRoot != nil) {
			header = fmt.Sprintf("%s: %s\n", utils.Wrap("Podman", padL, padR), utils.Warn("unavailable"))
		} else {
			header, content, _ = cl.toHeaderContent(c.Ignore, *c.FailedOnly)
		}
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
