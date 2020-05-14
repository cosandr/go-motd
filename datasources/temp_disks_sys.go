package datasources

import (
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

// getFromSys gets temperature data from /sys, requires drivetemp kernel module (Linux 5.6+)
func getFromSys() (deviceList []diskEntry, err error) {
	// Model (SCSI): /sys/block/sda/device/model
	// Temp (SCSI):  /sys/block/sda/device/hwmon/hwmon3/temp1_input
	// Model (NVMe): /sys/block/nvme0n1/device/model
	// Temp (NVMe):  /sys/block/nvme0n1/device/device/hwmon/hwmon1/temp1_input
	blockDevices, err := ioutil.ReadDir("/sys/block")
	if err != nil {
		return
	}
	var hwmonBaseDir string
	var model string
	for _, file := range blockDevices {
		devicePath := filepath.Join("/sys/block/", file.Name(), "/device")
		if strings.HasPrefix(file.Name(), "sd") {
			hwmonBaseDir = filepath.Join(devicePath, "/hwmon")
		} else if strings.HasPrefix(file.Name(), "nvme") {
			hwmonBaseDir = filepath.Join(devicePath, "/device/hwmon")
		} else {
			continue
		}
		content, err := ioutil.ReadFile(filepath.Join(devicePath, "/model"))
		if err != nil {
			model = "N/A"
			// Suppress error
			err = nil
		} else {
			model = string(content)
		}
		hwmonFiles, err := filepath.Glob(hwmonBaseDir + "/hwmon*/temp*_*")
		temps := readHwmonFiles(hwmonFiles)
		deviceList = append(deviceList, diskEntry{
			block: file.Name(),
			model: model,
			temps: temps,
		})
	}
	return
}

// Adapted from SensorsTemperaturesWithContext in gopsutil
func readHwmonFiles(files []string) (temps []diskTemp) {
	for _, file := range files {
		filename := strings.Split(filepath.Base(file), "_")
		// Only read current temperature
		if filename[1] != "input" {
			continue
		}

		// Get the label of the temperature you are reading
		var label string
		c, _ := ioutil.ReadFile(filepath.Join(filepath.Dir(file), filename[0]+"_label"))
		if c != nil {
			label = strings.Join(strings.Split(strings.TrimSpace(strings.ToLower(string(c))), " "), "")
		}

		// Get the temperature reading
		current, err := ioutil.ReadFile(file)
		if err != nil {
			continue
		}

		temperature, err := strconv.ParseFloat(strings.TrimSpace(string(current)), 64)
		if err != nil {
			continue
		}
		temps = append(temps, diskTemp{
			name: label,
			temp: temperature / 1000.0,
		})
	}
	return
}
