// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"encoding/json"
	"fmt"
	elastigo "github.com/mattbaird/elastigo/lib"
	"time"
)

var stateMagic = "GEOMODEL_STATE"

// In process state tracking, synchronized with global state stored
// in ES.
type procState struct {
	timeEndpoint time.Time // Time we've read up until
}

func (s *procState) fromObject(o object) (err error) {
	s.timeEndpoint = o.State.TimeEndpoint
	return nil
}

func (s *procState) toObject() (o object, err error) {
	o.ObjectIDString = stateMagic
	o.ObjectID = getObjectID(o.ObjectIDString)
	o.Context = cfg.General.Context
	o.State.TimeEndpoint = s.timeEndpoint
	return o, nil
}

func (s *procState) newState() {
	s.timeEndpoint = time.Now().UTC()
	if cfg.initialOffset != 0 {
		s.timeEndpoint = s.timeEndpoint.Add(-1 * time.Duration(cfg.initialOffset) * time.Second)
	}
}

var state procState

func updateState() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("updateState() -> %v", e)
		}
	}()
	stateobjid := getObjectID(stateMagic)

	conn := elastigo.NewConn()
	conn.Domain = cfg.ES.StateESHost

	template := `{
		"query": {
			"term": {
				"object_id": "%v"
			}
		}
	}`
	tempbuf := fmt.Sprintf(template, stateobjid)
	res, err := conn.Search(cfg.ES.StateIndex, "geomodel_state", nil, tempbuf)
	if err != nil {
		panic(err)
	}
	if res.Hits.Len() == 0 {
		logf("no state found, setting initial value")
		state.newState()
		return nil
	}
	if res.Hits.Len() > 1 {
		panic("> 1 state matched in index")
	}
	o := object{}
	err = json.Unmarshal(*res.Hits.Hits[0].Source, &o)
	if err != nil {
		panic(err)
	}
	err = state.fromObject(o)
	if err != nil {
		panic(err)
	}
	return nil
}

func saveState() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("saveState() -> %v", e)
		}
	}()
	conn := elastigo.NewConn()
	conn.Domain = cfg.ES.StateESHost

	sobj, err := state.toObject()
	if err != nil {
		panic(err)
	}
	_, err = conn.Index(cfg.ES.StateIndex, "geomodel_state", sobj.ObjectID, nil, sobj)
	if err != nil {
		panic(err)
	}

	return nil
}

// Prepare the state index for use by the state management process
func stateIndexInit() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("stateIndexInit() -> %v", e)
		}
	}()
	conn := elastigo.NewConn()
	conn.Domain = cfg.ES.StateESHost

	if cfg.deleteStateIndex {
		logf("removing any existing state index")
		_, err := conn.DeleteIndex(cfg.ES.StateIndex)
		if err != nil && err != elastigo.RecordNotFound {
			panic(err)
		}
	}

	ret, err := conn.IndicesExists(cfg.ES.StateIndex)
	if err != nil {
		panic(err)
	}
	if ret {
		logf("state index exists, skipping creation")
		return nil
	}
	logf("state index does not exist, creating")
	_, err = conn.CreateIndexWithSettings(cfg.ES.StateIndex, getStateSettings())
	if err != nil {
		panic(err)
	}

	return nil
}

func dispatchQueries() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("dispatchQueries() -> %v", e)
		}
	}()

	newEndpoint := time.Now().UTC()

	// Generate a query request for intervals from the last known endpoint
	// up until our new endpoint value
	sv := state.timeEndpoint
	for {
		if sv.Equal(newEndpoint) || sv.After(newEndpoint) {
			break
		}
		ev := sv.Add(time.Duration(cfg.Timer.MaxQueryWindow) * time.Second)
		if ev.After(newEndpoint) {
			ev = newEndpoint
		}

		dur := ev.Sub(sv)
		logf("dispatch query for %v -> %v (%v)", sv, ev, dur)
		newqr := queryRequest{
			startTime: sv,
			endTime:   ev,
		}
		queryRequestCh <- newqr

		sv = ev
	}

	// Save our last endpoint in the state
	state.timeEndpoint = newEndpoint

	return nil
}

func stateManager(exitCh chan bool, notifyCh chan bool) {
	defer func() {
		if e := recover(); e != nil {
			logf("stateManager() -> %v", e)
		}
		logf("state manager exiting")
	}()
	logf("state manager started")

	err := stateIndexInit()
	if err != nil {
		panic(err)
	}

	for {
		logf("state processor analyzing interval")
		// Update our current state from ES
		err = updateState()
		if err != nil {
			panic(err)
		}

		err = dispatchQueries()
		if err != nil {
			panic(err)
		}

		// Save our current state to ES
		err = saveState()
		if err != nil {
			panic(err)
		}

		select {
		case <-time.After(time.Duration(cfg.Timer.State) * time.Second):
		case <-exitCh:
			notifyCh <- true
			return
		}
	}
}
