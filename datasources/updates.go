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
	log "github.com/sirupsen/logrus"

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
	c.ConfBase.Init()
	c.PadHeader = []int{0, 2}
	c.PadContent = []int{1, 0}
	c.Address = "/run/go-check-updates.sock"
	c.Every = "1h"
}

// GetUpdates reads cached updates file and formats it
func GetUpdates(ch chan<- SourceReturn, conf *Conf) {
	c := conf.Updates
	// Check for warnOnly override
	if c.Show == nil {
		c.Show = &conf.WarnOnly
	}
	sr := NewSourceReturn(conf.debug)
	defer func() {
		ch <- sr.Return(&c.ConfBase)
	}()
	if c.File != "" {
		sr.Header, sr.Content, sr.Error = getUpdatesFile(&c)
	} else {
		sr.Header, sr.Content, sr.Error = getUpdatesAPI(&c)
	}
}

// getUpdatesResponse connects to go-check-updates at addr and returns the result
func getUpdatesResponse(addr string, url string, padL string, padR string) (header string, result api.Response, err error) {
	var client http.Client
	var connType string
	log.Debugf("[updates] request URL: %s", url)
	if strings.HasPrefix(addr, "/") {
		connType = "unix"
		log.Debugf("[updates] unix connection to %s", addr)
	} else {
		connType = "tcp"
		log.Debugf("[updates] tcp connection to %s", addr)
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
		err = &ModuleNotAvailable{"updates", err}
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Updates", padL, padR), utils.Warn("unavailable"))
		return
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		err = &ModuleNotAvailable{"updates", err}
		header = fmt.Sprintf("%s: %s (%v)\n", utils.Wrap("Updates", padL, padR), utils.Warn("Cannot decode response"), err)
		return
	}
	log.Debugf("[updates] response:\n%s", utils.PrettyPrint(&result))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = &ModuleNotAvailable{"updates", fmt.Errorf("invalid response code %s", resp.Status)}
		header = fmt.Sprintf("%s: %s (%s)\n", utils.Wrap("Updates", padL, padR), utils.Warn("Invalid response"), resp.Status)
		return
	}
	return
}

// getUpdatesAPI gets currently cached updates and queues an update if needed
func getUpdatesAPI(c *ConfUpdates) (header string, content string, err error) {
	var r api.Response
	reqURL := "http://con/api?updates"
	if c.Every != "" {
		reqURL += fmt.Sprintf("&refresh&immediate&every=%s", c.Every)
	}
	header, r, err = getUpdatesResponse(c.Address, reqURL, c.padL, c.padR)
	if err != nil {
		return
	}
	if r.Error != "" {
		log.Warnf("[updates] response contains error %s", r.Error)
		content = fmt.Sprintf("%s\n", utils.Warn(r.Error))
	}
	if r.Queued != nil && *r.Queued {
		if r.Data == nil {
			header = fmt.Sprintf("%s: No data, refreshing\n", utils.Wrap("Updates", c.padL, c.padR))
		} else {
			header = fmt.Sprintf("%s: %d pending, refreshing\n", utils.Wrap("Updates", c.padL, c.padR), len(r.Data.Updates))
		}
	} else {
		if r.Data == nil {
			header = fmt.Sprintf("%s: No data\n", utils.Wrap("Updates", c.padL, c.padR))
			return
		}
		t, err := time.Parse(time.RFC3339, r.Data.Checked)
		if err != nil {
			log.Warnf("[updates] cannot parse timestamp %s: %v", r.Data.Checked, err)
			header = fmt.Sprintf("%s: %d pending, cannot parse timestamp\n", utils.Wrap("Updates", c.padL, c.padR), len(r.Data.Updates))
			return header, content, err
		}
		var timeElapsed = time.Since(t)
		header = fmt.Sprintf("%s: %d pending, checked %s ago\n",
			utils.Wrap("Updates", c.padL, c.padR), len(r.Data.Updates), timeStr(timeElapsed, 2, c.ShortNames))
	}
	if r.Data == nil || c.Show == nil || !*c.Show {
		return
	}
	content += fmt.Sprint(utils.Wrap(r.Data.String(), c.padL, c.padR))
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
		err = &ModuleNotAvailable{"updates", err}
		header = fmt.Sprintf("%s: %s\n", utils.Wrap("Updates", c.padL, c.padR), utils.Warn("unavailable"))
		return
	}
	t, _ := time.Parse(time.RFC3339, data.Checked)
	var timeElapsed = time.Since(t)
	header = fmt.Sprintf("%s: %d pending, checked %s ago\n",
		utils.Wrap("Updates", c.padL, c.padR), len(data.Updates), timeStr(timeElapsed, 2, c.ShortNames))
	if c.Show == nil || !*c.Show {
		return
	}
	content += fmt.Sprint(utils.Wrap(data.String(), c.padL, c.padR))
	return
}
