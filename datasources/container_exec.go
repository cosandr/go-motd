package datasources

import (
	"bytes"
	"os/exec"
	"strings"
)

// getContainersExec returns container status using os/exec, ~5x slower than API
func getContainersExec(podman bool, sudo bool) (cl containerList, err error) {
	var stdout bytes.Buffer
	cl.Root = sudo
	if podman {
		cl.Runtime = "Podman"
	} else {
		cl.Runtime = "Docker"
	}
	var cmd *exec.Cmd
	if sudo {
		cmd = exec.Command("sudo", strings.ToLower(cl.Runtime), "ps", "--format", `"{{.Names}} {{.Status}}"`, "-a")
	} else {
		cmd = exec.Command(strings.ToLower(cl.Runtime), "ps", "--format", `"{{.Names}} {{.Status}}"`, "-a")
	}
	cmd.Stdout = &stdout
	err = cmd.Run()
	if err != nil {
		return
	}
	for _, c := range strings.Split(stdout.String(), "\n") {
		var tmp = strings.Split(strings.Trim(c, `"`), " ")
		if len(tmp) < 2 {
			continue
		}
		// tmp[0] - container name
		// tmp[1] - container status (up/created/exited)
		cl.Containers = append(cl.Containers, containerStatus{
			Name:   tmp[0],
			Status: tmp[1],
		})
	}
	return
}
