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
	"sync"
	"time"
)

type integQueue struct {
	queue []eventResult
	sync.Mutex
}

var queue integQueue

func (i *integQueue) getResult() (eventResult, bool) {
	var ret eventResult
	i.Lock()
	defer i.Unlock()
	if len(i.queue) == 0 {
		return ret, false
	}
	ret = i.queue[0]
	i.queue = i.queue[1:]
	return ret, true
}

func (i *integQueue) addResult(n eventResult) {
	i.Lock()
	i.queue = append(i.queue, n)
	i.Unlock()
}

func mergeResults(principal string, res []eventResult) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("mergeResults() -> %v", e)
		}
	}()

	logf("merging and updating for %v", principal)
	o, err := getPrincipalState(principal)
	if err != nil {
		panic(err)
	}

	// Add new events to the object state
	for _, x := range res {
		err = o.addEventResult(x)
		if err != nil {
			panic(err)
		}
	}

	// Collapse locality branches based on proximity
	err = geoCollapse(&o)
	if err != nil {
		panic(err)
	}

	// Calculate a geocenter for the principal based on known
	// authentications
	o.Geocenter, err = geoFindGeocenter(o)
	if err != nil {
		panic(err)
	}

	// Generate any alert events
	err = o.alertAnalyze()
	if err != nil {
		panic(err)
	}

	// Update the lastupdated timestamp
	o.LastUpdated = time.Now().UTC()

	err = savePrincipalState(o)
	if err != nil {
		panic(err)
	}

	return nil
}

func savePrincipalState(o object) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("savePrincipalState() -> %v", e)
		}
	}()

	conn := elastigo.NewConn()
	conn.Domain = cfg.ES.StateESHost

	_, err = conn.Index(cfg.ES.StateIndex, "geomodel_state", o.ObjectID, nil, o)
	if err != nil {
		panic(err)
	}

	return nil
}

func getPrincipalState(principal string) (ret object, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getPrincipalState() -> %v", e)
		}
	}()

	objid := getObjectID(principal)

	conn := elastigo.NewConn()
	conn.Domain = cfg.ES.StateESHost

	template := `{
		"query": {
			"term": {
				"object_id": "%v"
			}
		}
	}`
	tempbuf := fmt.Sprintf(template, objid)
	res, err := conn.Search(cfg.ES.StateIndex, "geomodel_state", nil, tempbuf)
	if err != nil {
		panic(err)
	}
	if res.Hits.Len() == 0 {
		logf("no state found for %v, creating", principal)
		ret.newFromPrincipal(principal)
		return ret, nil
	}
	err = json.Unmarshal(*res.Hits.Hits[0].Source, &ret)
	if err != nil {
		panic(err)
	}

	return ret, nil
}

func integrationMerge(exitCh chan bool) {
	defer func() {
		if e := recover(); e != nil {
			logf("integrationMerge() -> %v", e)
		}
		logf("integration merge exiting")
	}()
	logf("integration merge started")

	for {
		select {
		case <-exitCh:
			return
		case <-time.After(time.Duration(cfg.Timer.Merge) * time.Second):
		}
		logf("integration merge process running")

		princemap := make(map[string][]eventResult)
		// Fetch whatever we have queued; for efficiency group results
		// for the same principal together, reducing the number of
		// requests needed later.
		for e, ok := queue.getResult(); ok; e, ok = queue.getResult() {
			ptr, ok := princemap[e.Principal]
			if !ok {
				princemap[e.Principal] = make([]eventResult, 0)
				ptr = princemap[e.Principal]
			}
			ptr = append(ptr, e)
			princemap[e.Principal] = ptr
		}
		failed := false
		var err error
		for k, v := range princemap {
			err = mergeResults(k, v)
			if err != nil {
				failed = true
				break
			}
		}
		if failed {
			logf("integration merge failed, %v", err)
		}
	}
}

func integrate(pr pluginResult) {
	for _, x := range pr.Results {
		if !x.Valid {
			logf("ignoring invalid result from plugin")
			continue
		}
		queue.addResult(x)
	}
}

func integrator(exitCh chan bool, notifyCh chan bool) {
	defer func() {
		if e := recover(); e != nil {
			logf("integrator() -> %v", e)
		}
		logf("integrator exiting")
	}()
	logf("integrator started")

	var iwg sync.WaitGroup
	mergeExit := make(chan bool, 1)
	iwg.Add(1)
	go func() {
		integrationMerge(mergeExit)
		iwg.Done()
	}()

	for {
		select {
		case p := <-pluginResultCh:
			integrate(p)
		case <-exitCh:
			mergeExit <- true
			iwg.Wait()
			notifyCh <- true
			return
		}
	}
}
