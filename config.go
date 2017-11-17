// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"fmt"
	gcfg "gopkg.in/gcfg.v1"
	"time"
)

type config struct {
	ES struct {
		StateESHost string // ElasticSearch host for state information
		EventESHost string // ElasticSearch host for event information
		EventIndex  string // Index containing events
		StateIndex  string // geomodel state index
	}

	Geo struct {
		CollapseMaximum  int    // Maximum allowable collapse for branch locality (km)
		MovementWindow   string // time.Duration for movement heuristic
		MovementDistance int    // Distance for movement heuristic (km)
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
		Offset         string // time.Duration specifying standoff for query window
	}

	// Not expected to be in the configuration file, but other options we
	// want to store as part of the configuration.
	deleteStateIndex bool // Remove state index on startup
	initialOffset    int  // If creating initial state, start from this far back (seconds)
	noSendAlert      bool // Don't send alerts to MozDef
	overrides        []Override // Keeps track of custom ip -> city,country overrides
}

var cfg config

func (c *config) validate() error {
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
	if c.Timer.Offset != "" {
		_, err := time.ParseDuration(c.Timer.Offset)
		if err != nil {
			return err
		}
	}
	_, err := time.ParseDuration(c.Timer.ExpireEvents)
	if err != nil {
		return err
	}
	_, err = time.ParseDuration(c.Geo.MovementWindow)
	if err != nil {
		return err
	}
	if c.Geo.MovementDistance < 500 {
		return fmt.Errorf("geo..movementdistance must be >= 500")
	}
	return nil
}

func (c *config) loadConfiguration(path string) error {
	err := gcfg.ReadFileInto(&cfg, path)
	if err != nil {
		return err
	}
	return c.validate()
}
