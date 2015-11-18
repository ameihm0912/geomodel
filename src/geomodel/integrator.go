// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

func integrate(pr pluginResult) {
}

func integrator(exitCh chan bool, notifyCh chan bool) {
	defer func() {
		if e := recover(); e != nil {
			logf("integrator() -> %v", e)
		}
		logf("integrator exiting")
	}()
	logf("integrator started")

	for {
		select {
		case p := <-pluginResultCh:
			integrate(p)
		case <-exitCh:
			notifyCh <- true
			return
		}
	}
}
