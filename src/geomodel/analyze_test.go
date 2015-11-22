// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

type testTable []testPhase
type testPhase []testEvent

type testEvent struct {
	p         string
	srcip     string
	timestamp string
	n         int
}

// Tests invalidation of internal addresses
var testtab0 = testTable{
	{
		{"user@host.com", "10.0.0.1", "", 1},
		{"user@host.com", "172.16.0.1", "", 1},
		{"user@host.com", "192.168.100.1", "", 1},
		{"user@host.com", "0.0.0.0", "", 1},
		{"user@host.com", "1.2.3.4", "", 1},
	},
}

// Ensure weight threshold is zero based on no deviation in model
var testtab1 = testTable{
	{
		{"user@host.com", "1.2.3.4", "", 4},
	},
	{
		{"user@host.com", "1.2.3.4", "", 4},
	},
}

var testtab2 = testTable{
	{
		{"user@host.com", "63.245.214.133", "", 20},
		{"user@host.com", "118.163.10.187", "", 1},
	},
}

var testtab3 = testTable{
	{
		{"user@host.com", "63.245.214.133", "", 15},
		{"user@host.com", "118.163.10.187", "", 15},
	},
}

var testtab4 = testTable{
	{
		{"user@host.com", "63.245.214.133", "", 15},
	},
	{
		{"user@host.com", "63.245.214.133", "", 15},
		{"user@host.com", "118.163.10.187", "", 5},
	},
}

type simpleStateService struct {
	store map[string]object
}

func (s *simpleStateService) readObject(objid string) (o *object, err error) {
	r, ok := s.store[objid]
	if !ok {
		return nil, nil
	}
	return &r, nil
}

func (s *simpleStateService) writeObject(o object) (err error) {
	s.store[o.ObjectID] = o
	return nil
}

func (s *simpleStateService) doInit() (err error) {
	s.store = make(map[string]object)
	return nil
}

func (s *simpleStateService) getStore() map[string]object {
	return s.store
}

func makePhaseResults(tp testPhase) (pluginResult, error) {
	var ret pluginResult

	ret.Results = make([]eventResult, 0)
	for i := range tp {
		for j := 1; j <= tp[i].n; j++ {
			newres := eventResult{}
			newres.Timestamp = time.Now().UTC()
			newres.Principal = tp[i].p
			newres.SourceIPV4 = tp[i].srcip
			newres.Name = "test"
			newres.Valid = true
			err := newres.validate()
			if err != nil {
				return ret, err
			}
			ret.Results = append(ret.Results, newres)
		}
	}

	return ret, nil
}

func runTestPhase(tp testPhase) error {
	ev, err := makePhaseResults(tp)
	if err != nil {
		return err
	}
	integrate(ev)
	err = integrationMergeQueue()
	if err != nil {
		return err
	}
	return nil
}

func runTestTable(t testTable) error {
	fmt.Println("running a test table...")
	for i := range t {
		err := runTestPhase(t[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func testGenericInit() error {
	cfg.General.MaxMind = os.Getenv("TESTMMF")
	cfg.Geo.CollapseMaximum = 500
	cfg.Geo.DeviationMinimum = 5
	err := maxmindInit()
	if err != nil {
		return err
	}
	setStateService(&simpleStateService{})
	return nil
}

func TestAnalyzeTab0(t *testing.T) {
	defer func() {
		if e := recover(); e != nil {
			t.Fatalf("TestAnalyzeTab0 -> %v", e)
		}
	}()

	err := testGenericInit()
	if err != nil {
		panic(err)
	}
	getStateService().(*simpleStateService).doInit()
	err = runTestTable(testtab0)
	if err != nil {
		panic(err)
	}
	s := getStateService().(*simpleStateService).getStore()
	cnt := len(s)
	if cnt != 1 {
		panic("more than one entry in state")
	}
	for _, v := range s {
		// Since most of the results should have been invalidated, we
		// should only have one stored
		if len(v.Results) != 1 {
			panic("more than one result in state entry")
		}
		if v.Results[0].SourceIPV4 != "1.2.3.4" {
			panic("result we had was not the correct one")
		}
	}
}

func TestAnalyzeTab1(t *testing.T) {
	defer func() {
		if e := recover(); e != nil {
			t.Fatalf("TestAnalyzeTab1 -> %v", e)
		}
	}()

	err := testGenericInit()
	if err != nil {
		panic(err)
	}
	getStateService().(*simpleStateService).doInit()
	err = runTestTable(testtab1)
	if err != nil {
		panic(err)
	}
	s := getStateService().(*simpleStateService).getStore()
	cnt := len(s)
	if cnt != 1 {
		panic("more than one entry in state")
	}
	for _, v := range s {
		if v.WeightThresh != 0 {
			panic("weight threshold was not zero")
		}
	}
}

func TestAnalyzeTab2(t *testing.T) {
	defer func() {
		if e := recover(); e != nil {
			t.Fatalf("TestAnalyzeTab2 -> %v", e)
		}
	}()

	err := testGenericInit()
	if err != nil {
		panic(err)
	}
	getStateService().(*simpleStateService).doInit()
	err = runTestTable(testtab2)
	if err != nil {
		panic(err)
	}
	s := getStateService().(*simpleStateService).getStore()
	cnt := len(s)
	if cnt != 1 {
		panic("more than one entry in state")
	}
	for _, v := range s {
		if v.WeightThresh == 0 {
			panic("weight threshold was zero")
		}
		if !v.AlertAnalyzed {
			panic("alertanalyzed was false")
		}
	}
}

func TestAnalyzeTab3(t *testing.T) {
	defer func() {
		if e := recover(); e != nil {
			t.Fatalf("TestAnalyzeTab3 -> %v", e)
		}
	}()

	err := testGenericInit()
	if err != nil {
		panic(err)
	}
	getStateService().(*simpleStateService).doInit()
	err = runTestTable(testtab3)
	if err != nil {
		panic(err)
	}
	s := getStateService().(*simpleStateService).getStore()
	cnt := len(s)
	if cnt != 1 {
		panic("more than one entry in state")
	}
	for _, v := range s {
		if v.AlertAnalyzed {
			panic("alertanalyzed was true")
		}
	}
}

func TestAnalyzeTab4(t *testing.T) {
	defer func() {
		if e := recover(); e != nil {
			t.Fatalf("TestAnalyzeTab4 -> %v", e)
		}
	}()

	err := testGenericInit()
	if err != nil {
		panic(err)
	}
	getStateService().(*simpleStateService).doInit()
	err = runTestTable(testtab4)
	if err != nil {
		panic(err)
	}
	s := getStateService().(*simpleStateService).getStore()
	cnt := len(s)
	if cnt != 1 {
		panic("more than one entry in state")
	}
	for _, v := range s {
		if !v.AlertAnalyzed {
			panic("alertanalyzed was false")
		}
	}
}
