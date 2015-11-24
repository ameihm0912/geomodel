// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"
)

var logch chan string
var logActive bool
var queryRequestCh chan queryRequest
var pluginResultCh chan pluginResult

func startRoutines() {
	var wg sync.WaitGroup

	queryRequestCh = make(chan queryRequest, 128)
	pluginResultCh = make(chan pluginResult, 128)

	exitNotifyCh := make(chan bool, 12)
	stateExitCh := make(chan bool, 1)
	queryExitCh := make(chan bool, 1)
	integExitCh := make(chan bool, 1)

	go func() {
		<-exitNotifyCh
		stateExitCh <- true
		queryExitCh <- true
		integExitCh <- true
	}()

	// Install signal handler
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	go func() {
		for _ = range sigch {
			logf("caught signal, attempting to exit")
			exitNotifyCh <- true
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		stateManager(stateExitCh, exitNotifyCh)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		queryHandler(queryExitCh, exitNotifyCh)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		integrator(integExitCh, exitNotifyCh)
	}()
	wg.Wait()
}

func logf(s string, args ...interface{}) {
	if !logActive {
		return
	}
	buf := fmt.Sprintf(s, args...)
	tstr := time.Now().Format("2006-01-02 15:04:05")
	logbuf := fmt.Sprintf("[%v] %v", tstr, buf)
	logch <- logbuf
}

func logger() {
	for s := range logch {
		fmt.Fprintf(os.Stdout, "%v\n", s)
	}
}

func main() {
	var delIndex = flag.Bool("D", false, "delete and recreate state index on startup")
	var confPath = flag.String("f", "etc/geomodel.conf", "configuration path")
	var nsAlert = flag.Bool("n", false, "dont send alerts to mozdef")
	var initOff = flag.Int("o", 0, "initial state offset in seconds")
	var eventIdx = flag.String("I", "", "override event index name from config file")
	flag.Parse()

	err := cfg.loadConfiguration(*confPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading configuration: %v\n", err)
		os.Exit(2)
	}
	cfg.deleteStateIndex = *delIndex
	cfg.initialOffset = *initOff
	cfg.noSendAlert = *nsAlert
	if *eventIdx != "" {
		cfg.ES.EventIndex = *eventIdx
	}

	// Initialize the logging routine
	var wg sync.WaitGroup
	logch = make(chan string, 32)
	logActive = true
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger()
	}()

	setStateService(&esStateService{})

	err = maxmindInit()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error initializing maxmind: %v\n", err)
		os.Exit(2)
	}

	err = loadPlugins()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading plugins: %v\n", err)
		os.Exit(2)
	}

	// Start the other primary routines
	startRoutines()
	logf("routines exited, waiting for logger to finish")
	close(logch)
	wg.Wait()
	fmt.Fprintf(os.Stdout, "exiting\n")
	os.Exit(0)
}
