// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"math"
	"time"
)

// Describes an object stored in the state index used by geomodel. This
// could represent state metadata associated with entities in a context,
// or it could be a global state object. We use the same structure for
// both.
type object struct {
	ObjectID       string          `json:"object_id"`
	ObjectIDString string          `json:"object_id_string"`
	Context        string          `json:"context"`
	State          objectState     `json:"state,omitempty"`
	Results        []objectResult  `json:"results,omitempty"`
	Geocenter      objectGeocenter `json:"geocenter"`
	LastUpdated    time.Time       `json:"last_updated"`
	WeightThresh   float64         `json:"weight_threshold"`
	AlertAnalyzed  bool            `json:"alert_analyzed"`

	MLocations []location `json:"locations_model"`
	RLocations []location `json:"locations_review"`
}

func (o *object) addEventResult(e eventResult) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("addEventResult() -> %v", e)
		}
	}()

	if !e.Valid {
		panic("invalid result")
	}

	newres := objectResult{}
	newres.SourcePlugin = e.Name
	newres.BranchID = uuid.New()
	newres.Timestamp = e.Timestamp
	newres.Collapsed = false
	newres.SourceIPV4 = e.SourceIPV4
	err = geoObjectResult(&newres)
	if err != nil {
		panic(err)
	}
	o.Results = append(o.Results, newres)

	return nil
}

func (o *object) newFromPrincipal(principal string) {
	var err error
	o.ObjectID, err = getObjectID(principal)
	if err != nil {
		panic(err)
	}
	o.ObjectIDString = principal
	o.Context = cfg.General.Context
}

func (o *object) updateLocations() {
	o.MLocations = o.MLocations[:0]
	o.RLocations = o.RLocations[:0]
	// Sort based on the mean of the data set
	cnt := 0
	var s float64 = 0
	for _, x := range o.Results {
		if x.Collapsed {
			continue
		}
		s += x.Weight
		cnt++

	}
	m := s / float64(cnt)
	for _, x := range o.Results {
		if x.Collapsed {
			continue
		}
		nloc := location{
			Locality: x.Locality,
			Weight:   x.Weight,
		}
		if x.Weight < m {
			o.RLocations = append(o.RLocations, nloc)
		} else {
			o.MLocations = append(o.MLocations, nloc)
		}
	}
}

func (o *object) weightThresholdDeviation() {
	var fset []float64

	for _, x := range o.Results {
		// Only take into account branches that have not been
		// collapsed
		if x.Collapsed {
			continue
		}
		fset = append(fset, x.Weight)
	}
	if len(fset) <= 1 {
		o.WeightThresh = 0
		return
	}
	var t0 float64 = 0
	for _, x := range fset {
		t0 += x
	}
	mean := t0 / float64(len(fset))
	var fset2 []float64
	for _, x := range fset {
		fset2 = append(fset2, math.Pow(x-mean, 2))
	}
	t0 = 0
	for _, x := range fset {
		t0 += x
	}
	variance := t0 / float64(len(fset2))
	o.WeightThresh = math.Sqrt(variance)
}

func (o *object) alertAnalyze() error {
	o.AlertAnalyzed = false
	o.weightThresholdDeviation()
	o.updateLocations()
	if o.WeightThresh < float64(cfg.Geo.DeviationMinimum) {
		logf("skipping alertAnalyze() on %v, %v below deviation min", o.ObjectIDString, o.WeightThresh)
		return nil
	}
	o.AlertAnalyzed = true
	logf("suspect deviation %v for %v", o.WeightThresh, o.ObjectIDString)
	return nil
}

type location struct {
	Weight   float64 `json:"weight"`
	Locality string  `json:"locality"`
}

// Specific to global state tracking
type objectState struct {
	TimeEndpoint time.Time `json:"time_endpoint,omitempty"`
}

// Principal geocenter
type objectGeocenter struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Locality  string  `json:"locality,omitempty"`
	AvgDist   float64 `json:"avg_dist,omitempty"`
	Weight    float64 `json:"weight"`
}

// Single authentication result for a principal
type objectResult struct {
	SourcePlugin string  `json:"source_plugin"`
	BranchID     string  `json:"branch_id"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Locality     string  `json:"locality"`
	SourceIPV4   string  `json:"source_ipv4"`
	Weight       float64 `json:"weight"`

	Timestamp time.Time `json:"timestamp"`

	Collapsed      bool   `json:"collapsed"`
	CollapseBranch string `json:"collapse_branch,omitempty"`
}
