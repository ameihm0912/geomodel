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

// Defines a general state reader/write interface
type stateService interface {
	writeObject(object) error
	readObject(string) (*object, error)
	doInit() error
}

var stateServ stateService = nil

type esStateService struct {
	stateDomain string
	stateIndex  string
}

func (e *esStateService) writeObject(o object) (err error) {
	conn := elastigo.NewConn()
	conn.Domain = e.stateDomain
	_, err = conn.Index(e.stateIndex, "geomodel_state", o.ObjectID, nil, o)
	if err != nil {
		return err
	}
	return nil
}

func (e *esStateService) readObject(objid string) (o *object, err error) {
	conn := elastigo.NewConn()
	conn.Domain = e.stateDomain

	template := `{
		"query": {
			"term": {
				"object_id": "%v"
			}
		}
	}`
	tempbuf := fmt.Sprintf(template, objid)
	res, err := conn.Search(e.stateIndex, "geomodel_state", nil, tempbuf)
	if err != nil {
		return o, err
	}
	if res.Hits.Len() == 0 {
		return nil, nil
	}
	if res.Hits.Len() > 1 {
		return nil, fmt.Errorf("consistency failure, more than one object matched")
	}
	o = &object{}
	err = json.Unmarshal(*res.Hits.Hits[0].Source, o)
	if err != nil {
		return nil, err
	}

	return o, nil
}

func (e *esStateService) doInit() (err error) {
	if cfg.ES.StateESHost == "" {
		return fmt.Errorf("no valid es state host defined in configuration")
	}
	if cfg.ES.StateIndex == "" {
		return fmt.Errorf("no valid es state index defined in configuration")
	}
	e.stateDomain = cfg.ES.StateESHost
	e.stateIndex = cfg.ES.StateIndex
	return e.stateIndexInit()
}

func (e *esStateService) stateIndexInit() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("stateIndexInit() -> %v", e)
		}
	}()

	conn := elastigo.NewConn()
	conn.Domain = e.stateDomain

	if cfg.deleteStateIndex {
		logf("removing any existing state index")
		_, err := conn.DeleteIndex(e.stateIndex)
		if err != nil && err != elastigo.RecordNotFound {
			panic(err)
		}
	}

	ret, err := conn.IndicesExists(e.stateIndex)
	if err != nil {
		panic(err)
	}
	if ret {
		logf("state index exists, skipping creation")
		return nil
	}
	logf("state index does not exist, creating")
	_, err = conn.CreateIndexWithSettings(e.stateIndex, getStateSettings())
	if err != nil {
		panic(err)
	}
	time.Sleep(time.Duration(2) * time.Second)

	return nil
}

// In-process state tracking, synchronized with global state stored
// in by the state service.
type procState struct {
	timeEndpoint time.Time // Time we've read up until
}

// Update state using information from object o
func (s *procState) fromObject(o object) (err error) {
	s.timeEndpoint = o.State.TimeEndpoint
	return nil
}

// Create a new state object based on current state information
func (s *procState) toObject() (o object, err error) {
	o.ObjectIDString = stateMagic
	o.ObjectID, err = getObjectID(o.ObjectIDString)
	if err != nil {
		panic(err)
	}
	o.Context = cfg.General.Context
	o.State.TimeEndpoint = s.timeEndpoint
	o.LastUpdated = time.Now().UTC()
	o.Timestamp = o.LastUpdated
	return o, nil
}

// Initialize a new state object
func (s *procState) newState() {
	s.timeEndpoint = time.Now().UTC()
	if cfg.initialOffset != 0 {
		s.timeEndpoint = s.timeEndpoint.Add(-1 * time.Duration(cfg.initialOffset) * time.Second)
	}
}

var state procState

func setStateService(ss stateService) error {
	if stateServ != nil {
		return fmt.Errorf("setStateService() -> state service already initialized")
	}
	stateServ = ss
	return stateServ.doInit()
}

func getStateService() stateService {
	return stateServ
}

func updateState(ss stateService) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("updateState() -> %v", e)
		}
	}()
	stateobjid, err := getObjectID(stateMagic)
	if err != nil {
		panic(err)
	}
	obj, err := ss.readObject(stateobjid)
	if err != nil {
		panic(err)
	}
	if obj == nil {
		logf("no state found, setting initial value")
		state.newState()
		return nil
	}
	err = state.fromObject(*obj)
	if err != nil {
		panic(err)
	}
	return nil
}

func saveState(ss stateService) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("saveState() -> %v", e)
		}
	}()
	sobj, err := state.toObject()
	if err != nil {
		panic(err)
	}
	err = ss.writeObject(sobj)
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
		notifyCh <- true
	}()
	logf("state manager started")

	for {
		logf("state processor analyzing interval")
		// Update our current state from ES
		err := updateState(getStateService())
		if err != nil {
			panic(err)
		}

		err = dispatchQueries()
		if err != nil {
			panic(err)
		}

		// Save our current state to ES
		err = saveState(getStateService())
		if err != nil {
			panic(err)
		}

		select {
		case <-time.After(time.Duration(cfg.Timer.State) * time.Second):
		case <-exitCh:
			return
		}
	}
}
