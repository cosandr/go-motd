package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/cosandr/go-check-updates/api"
	"github.com/cosandr/go-motd/utils"
)

// ConfUpdates extends ConfBase with a show toggle (same as warnOnly), path to file and how often to check
type ConfUpdates struct {
	ConfBase `yaml:",inline"`
	// Show packages that can be upgraded
	Show *bool `yaml:"show,omitempty"`
	// Listen address of go-check-updates, absolute path indicates unix socket, otherwise <addr>:<port>
	Address string `yaml:"address"`
	// File will read the cache file directly
	File string `yaml:"file"`
	// Every defines how often the cache will be asked to update itself
	Every string `yaml:"every"`
	// ShortNames uses short names for time durations (1h5m instead of 1 hour, 5 min)
	ShortNames bool `yaml:"short_names"`
}

// Init sets default alignment and default socket file
func (c *ConfUpdates) Init() {
	c.PadHeader = []int{0, 2}
	c.PadContent = []int{1, 0}
	c.Address = "/run/go-check-updates.sock"
	c.Every = "1h"
}

// GetUpdates reads cached updates file and formats it
func GetUpdates(ret chan<- string, c *ConfUpdates) {
	var header string
	var content string
	if c.File != "" {
		header, content, _ = getUpdatesFile(c)
	} else {
		header, content, _ = getUpdatesAPI(c)
	}
	// Pad header
	var p = utils.Pad{Delims: map[string]int{padL: c.PadHeader[0], padR: c.PadHeader[1]}, Content: header}
	header = p.Do()
	if len(content) == 0 {
		ret <- header
		return
	}
	// Pad container list
	p = utils.Pad{Delims: map[string]int{padL: c.PadContent[0], padR: c.PadContent[1]}, Content: content}
	content = p.Do()
	ret <- header + "\n" + content
}

// getUpdatesResponse connects to go-check-updates at addr and returns the result
func getUpdatesResponse(addr string, url string) (header string, result api.Response, err error) {
	var client http.Client
	var connType string
	if strings.HasPrefix(addr, "/") {
		connType = "unix"
	} else {
		connType = "tcp"
	}
	client = http.Client{
		Timeout: 1 * time.Second,
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial(connType, addr)
			},
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Updates", padL, padR), utils.Warn("unavailable"))
		return
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		header = fmt.Sprintf("%s: %s (%v)\n", utils.Wrap("Updates", padL, padR), utils.Warn("Cannot decode response"), err)
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		header = fmt.Sprintf("%s: %s (%s)\n", utils.Wrap("Updates", padL, padR), utils.Warn("Invalid response"), resp.Status)
		return
	}
	return
}

// getUpdatesAPI gets currently cached updates and queues an update if needed
func getUpdatesAPI(c *ConfUpdates) (header string, content string, err error) {
	var r api.Response
	reqUrl := "http://con/api?updates"
	if c.Every != "" {
		reqUrl += fmt.Sprintf("&refresh&immediate&every=%s", c.Every)
	}
	header, r, err = getUpdatesResponse(c.Address, reqUrl)
	if err != nil {
		return
	}
	if r.Error != "" {
		content = fmt.Sprintf("%s\n", utils.Warn(r.Error))
	}
	if r.Queued != nil && *r.Queued == true {
		header = fmt.Sprintf("%s: %d pending, refreshing\n", utils.Wrap("Updates", padL, padR), len(r.Data.Updates))
	} else {
		t, err := time.Parse(time.RFC3339, r.Data.Checked)
		if err != nil {
			header = fmt.Sprintf("%s: %d pending, cannot parse timestamp\n", utils.Wrap("Updates", padL, padR), len(r.Data.Updates))
		}
		var timeElapsed = time.Since(t)
		header = fmt.Sprintf("%s: %d pending, checked %s ago\n",
			utils.Wrap("Updates", padL, padR), len(r.Data.Updates), timeStr(timeElapsed, 2, c.ShortNames))
	}
	if c.Show == nil || *c.Show == false {
		return
	}
	for _, u := range r.Data.Updates {
		content += fmt.Sprintf("%s -> %s\n", utils.Wrap(u.Pkg, padL, padR), u.NewVer)
	}
	return
}

// readUpdatesCache reads the cache file from the given path
func readUpdatesCache(cacheFp string) (parsed api.File, err error) {
	fb, err := ioutil.ReadFile(cacheFp)
	if err != nil {
		return
	}
	err = json.Unmarshal(fb, &parsed)
	return
}

func getUpdatesFile(c *ConfUpdates) (header string, content string, err error) {
	data, err := readUpdatesCache(c.File)
	if err != nil {
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Updates", padL, padR), utils.Warn("unavailable"))
		content = fmt.Sprint(err)
		return
	}
	t, _ := time.Parse(time.RFC3339, data.Checked)
	var timeElapsed = time.Since(t)
	header = fmt.Sprintf("%s: %d pending, checked %s ago\n",
		utils.Wrap("Updates", padL, padR), len(data.Updates), timeStr(timeElapsed, 2, c.ShortNames))
	if c.Show == nil || *c.Show == false {
		return
	}
	for _, u := range data.Updates {
		content += fmt.Sprintf("%s -> %s\n", utils.Wrap(u.Pkg, padL, padR), u.NewVer)
	}
	return
}
