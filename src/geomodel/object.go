// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"time"
)

// Describes an object stored in the state index used by geomodel. This
// could represent state metadata associated with entities in a context,
// or it could be a global state object. We use the same structure for
// both.
type object struct {
	ObjectID       string         `json:"object_id"`
	ObjectIDString string         `json:"object_id_string"`
	Context        string         `json:"context"`
	State          objectState    `json:"state,omitempty"`
	Results        []objectResult `json:"results,omitempty"`
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

// Single authentication result for a principal
type objectResult struct {
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Locality   string  `json:"locality"`
	SourceIPV4 string  `json:"source_ipv4"`
}
