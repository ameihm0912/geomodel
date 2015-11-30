// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

// Define the settings used to create the state index in ElasticSearch

type stateSettings struct {
	Mappings stateMappings   `json:"mappings"`
	Settings inStateSettings `json:"settings"`
}

type inStateSettings struct {
	Shards   int `json:"number_of_shards"`
	Replicas int `json:"number_of_replicas"`
}

type stateMappings struct {
	GMS stateGMS `json:"geomodel_state"`
}

type stateGMS struct {
	Properties stateProperties `json:"properties"`
}

type stateProperties struct {
	ObjectID       stateValueProperties `json:"object_id"`
	ObjectIDString stateValueProperties `json:"object_id_string"`
}

type stateValueProperties struct {
	Type  string `json:"type"`
	Index string `json:"index"`
}

// Return a stateSettings struct useful as an argument passed to
// elastigo CreateIndexWithSettings()
func getStateSettings() stateSettings {
	return stateSettings{
		Settings: inStateSettings{
			Shards:   2,
			Replicas: 1,
		},
		Mappings: stateMappings{
			GMS: stateGMS{
				Properties: stateProperties{
					ObjectID: stateValueProperties{
						Type:  "string",
						Index: "not_analyzed",
					},
					ObjectIDString: stateValueProperties{
						Type:  "string",
						Index: "not_analyzed",
					},
				},
			},
		},
	}
}
