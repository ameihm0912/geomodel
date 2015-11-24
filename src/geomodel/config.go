// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"code.google.com/p/gcfg"
	"fmt"
	"time"
)

type Config struct {
	ES struct {
		StateESHost string // ElasticSearch host for state information
		EventESHost string // ElasticSearch host for event information
		EventIndex  string // Index containing events
		StateIndex  string // geomodel state index
	}

	Geo struct {
		CollapseMaximum int // Maximum allowable collapse for branch locality (km)
	}

	MozDef struct {
		MozDefURL string // URL for MozDef event publishing
	}

	General struct {
		Context string // Context name
		Plugins string // Plugin directory path
		MaxMind string // Path to MaxMind DB
	}

	Timer struct {
		State          int    // State interval in seconds
		MaxQueryWindow int    // Maximum query window in seconds
		Merge          int    // Merge interval in seconds
		ExpireEvents   string // time.Duration specifying how to prune events
	}

	// Not expected to be in the configuration file, but other options we
	// want to store as part of the configuration.
	deleteStateIndex bool // Remove state index on startup
	initialOffset    int  // If creating initial state, start from this far back (seconds)
}

var cfg Config

func (c *Config) validate() error {
	if c.ES.StateESHost == "" {
		return fmt.Errorf("es..stateeshost must be set")
	}
	if c.ES.EventESHost == "" {
		return fmt.Errorf("es..eventeshost must be set")
	}
	if c.ES.EventIndex == "" {
		return fmt.Errorf("es..eventindex must be set")
	}
	if c.ES.StateIndex == "" {
		return fmt.Errorf("es..stateindex must be set")
	}
	if c.General.Context == "" {
		return fmt.Errorf("general..context must be set")
	}
	if c.General.Plugins == "" {
		return fmt.Errorf("general..plugins must be set")
	}
	if c.General.MaxMind == "" {
		return fmt.Errorf("general..maxmind must be set")
	}
	if c.MozDef.MozDefURL == "" {
		return fmt.Errorf("mozdef..mozdefurl must be set")
	}
	if c.Timer.State < 10 {
		return fmt.Errorf("timer..state must be >= 10")
	}
	if c.Timer.Merge < 10 {
		return fmt.Errorf("timer..merge must be >= 10")
	}
	if c.Timer.MaxQueryWindow < 60 {
		return fmt.Errorf("timer..maxquerywindow must be >= 60")
	}
	if c.Timer.ExpireEvents == "" {
		return fmt.Errorf("timer..expireevents must be set")
	}
	_, err := time.ParseDuration(c.Timer.ExpireEvents)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) loadConfiguration(path string) error {
	err := gcfg.ReadFileInto(&cfg, path)
	if err != nil {
		return err
	}
	return c.validate()
}
