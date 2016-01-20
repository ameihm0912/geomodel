// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"fmt"
	elastigo "github.com/mattbaird/elastigo/lib"
	"time"
)

type queryRequest struct {
	startTime time.Time
	endTime   time.Time
}

func queryUsingPlugin(p plugin, req queryRequest) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("queryUsingPlugin() -> %v", e)
		}
	}()

	template := `{
		"size": 10000,
		"query": {
			"bool": {
				"must": [
				%v
				]
			}
		},
		"filter": {
			"range": {
				"utctimestamp": {
					"from": "%v",
					"to": "%v"
				}
			}
		}
	}`

	// Add plugins search terms to the query
	mult := false
	temp := ""
	if len(p.searchTerms) == 0 {
		panic("plugin has no search terms defined")
	}
	for _, x := range p.searchTerms {
		if mult {
			temp += ","
		}
		termtemplate := `{
			"term": {
				"%v": "%v"
			}
		}`
		termbuf := fmt.Sprintf(termtemplate, x.key, x.value)
		temp += termbuf
		mult = true
	}
	querybuf := fmt.Sprintf(template, temp, req.startTime.Format(time.RFC3339), req.endTime.Format(time.RFC3339))
	conn := elastigo.NewConn()
	defer conn.Close()
	conn.Domain = cfg.ES.EventESHost
	res, err := conn.Search(cfg.ES.EventIndex, "", nil, querybuf)
	if err != nil {
		panic(err)
	}
	logf("plugin %v returned %v hits", p.name, res.Hits.Len())

	if res.Hits.Len() == 0 {
		return nil
	}

	pluginInput, err := pluginRequestDataFromES(res)
	if err != nil {
		panic(err)
	}
	err = p.runPlugin(pluginInput)
	if err != nil {
		panic(err)
	}

	return nil
}

func handleQueryRequest(q queryRequest) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("handleQueryRequest() -> %v", e)
		}
	}()
	logf("handling new query request")

	// Execute a query for each registered plugin
	for _, x := range pluginList {
		err := queryUsingPlugin(x, q)
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func queryHandler(exitCh chan bool, notifyCh chan bool) {
	defer func() {
		if e := recover(); e != nil {
			logf("queryHandler() -> %v", e)
		}
		logf("query handler exiting")
		notifyCh <- true
	}()
	logf("query handler started")

	for {
		select {
		case qr := <-queryRequestCh:
			err := handleQueryRequest(qr)
			if err != nil {
				panic(err)
			}
		case <-exitCh:
			return
		}
	}
}
