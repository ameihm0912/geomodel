// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Brandon Myers bmyers@mozilla.com

package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type override struct {
	cidr      string
	city      string
	country   string
	latitude  float64
	longitude float64
}

func readOverrides(path string) (overrides []override, err error) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line_contents := scanner.Text()
		if strings.HasPrefix(line_contents, "#") {
			// The line is a comment, so we'll skip it
			continue
		}

		elements := strings.Split(line_contents, ",")
		elements_len := len(elements)
		if elements_len != 5 {
			return nil, fmt.Errorf("Line must have 5 comma separated elements: %v", line_contents)
		}

		cidr := elements[0]
		city := elements[1]
		country := elements[2]
		latitude, lat_err := strconv.ParseFloat(elements[3], 64)
		if lat_err != nil {
			fmt.Fprintf(os.Stderr, "Error converting latitude: %v\n", lat_err)
			return nil, lat_err
		}
		longitude, long_err := strconv.ParseFloat(elements[4], 64)
		if long_err != nil {
			fmt.Fprintf(os.Stderr, "Error converting longitude: %v\n", long_err)
			return nil, long_err
		}

		overrides = append(overrides, override{cidr, city, country, latitude, longitude})
	}

	return overrides, nil
}
