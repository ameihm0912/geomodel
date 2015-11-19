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
	newres.BranchID = uuid.New()
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
	o.ObjectID = getObjectID(principal)
	o.ObjectIDString = principal
	o.Context = cfg.General.Context
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
	BranchID   string  `json:"branch_id"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Locality   string  `json:"locality"`
	SourceIPV4 string  `json:"source_ipv4"`
	Weight     float64 `json:"weight"`

	Collapsed      bool   `json:"collapsed"`
	CollapseBranch string `json:"collapse_branch,omitempty"`
}
