// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
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

func mergeResults(principal string, res []eventResult) {
	logf("merging and updating for %v", principal)
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
		for k, v := range princemap {
			mergeResults(k, v)
		}
	}
}

func integrate(pr pluginResult) {
	for _, x := range pr.Results {
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
