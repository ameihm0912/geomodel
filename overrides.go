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

func readOverrides(path string) (overrides []override) {
	file, err := os.Open(path)
	defer file.Close()
	if err != nil {
		fmt.Printf("error opening file: %v\n", err)
		return overrides
	}

	reader := bufio.NewReader(file)

	var line string
	for {
		line, err = reader.ReadString('\n')
		line_contents := strings.TrimSpace(line)
		if strings.HasPrefix(line_contents, "#") {
			// The line is a comment, so we'll skip it
			continue
		}
		if line_contents == "" {
			break
		}

		elements := strings.Split(line_contents, ",")
		cidr := elements[0]
		city := elements[1]
		country := elements[2]
		latitude, lat_err := strconv.ParseFloat(elements[3], 64)
		if lat_err != nil {
			fmt.Printf("Error converting latitude: %v\n", err)
			continue
		}
		longitude, long_err := strconv.ParseFloat(elements[4], 64)
		if long_err != nil {
			fmt.Printf("Error converting longitude: %v\n", err)
			continue
		}

		overrides = append(overrides, override{cidr, city, country, latitude, longitude})

		if err != nil {
			break
		}
	}

	return overrides
}
