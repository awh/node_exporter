// Copyright 2016 Adam Harrison
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !nohwmon

package collector

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
)

type hwmonCollector struct {
	temp *prometheus.Desc
}

func init() {
	Factories["hwmon"] = NewHwmonCollector
}

// Takes a prometheus registry and returns a new Collector exposing
// sensor data from the kernel hwmon subsystem.
func NewHwmonCollector() (Collector, error) {
	return &hwmonCollector{
		temp: prometheus.NewDesc(
			prometheus.BuildFQName(Namespace, "hwmon", "temp"),
			"Hardware temperatures.",
			[]string{"monitorType", "monitor", "sensor", "label"}, nil,
		),
	}, nil
}

// Expose sensor data from the kernel hwmon subsystem.
func (c *hwmonCollector) Update(ch chan<- prometheus.Metric) error {
	fis, err := ioutil.ReadDir("/sys/class/hwmon")
	if err != nil {
		return err
	}

	for _, fi := range fis {
		var monitor int
		_, err := fmt.Sscanf(fi.Name(), "hwmon%d", &monitor)
		if err != nil {
			continue
		}

		err = c.scrapeHWMon(ch, monitor, path.Join("/sys/class/hwmon", fi.Name()))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *hwmonCollector) scrapeHWMon(ch chan<- prometheus.Metric, monitor int, hwmonDir string) error {
	nameBytes, err := ioutil.ReadFile(path.Join(hwmonDir, "name"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	fis, err := ioutil.ReadDir(hwmonDir)
	if err != nil {
		return err
	}

	for _, fi := range fis {
		var sensor int
		_, err := fmt.Sscanf(fi.Name(), "temp%d_input", &sensor)
		if err != nil {
			continue
		}

		inputBytes, err := ioutil.ReadFile(path.Join(hwmonDir, fi.Name()))
		if err != nil {
			return err
		}

		labelBytes, err := ioutil.ReadFile(path.Join(hwmonDir, fmt.Sprintf("temp%d_label", sensor)))
		if err != nil {
			return err
		}

		var milliC int
		_, err = fmt.Sscanf(string(inputBytes), "%d", &milliC)
		if err != nil {
			return err
		}

		ch <- prometheus.MustNewConstMetric(c.temp,
			prometheus.GaugeValue,
			float64(milliC)/1000.0,
			strings.TrimSpace(string(nameBytes)),
			strconv.Itoa(monitor),
			strconv.Itoa(sensor),
			strings.TrimSpace(string(labelBytes)))
	}

	return nil
}
