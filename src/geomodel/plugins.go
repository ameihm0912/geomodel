// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	elastigo "github.com/mattbaird/elastigo/lib"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
)

type pluginRequest struct {
	Events []*json.RawMessage `json:"events"`
}

type pluginResult struct {
	Results []eventResult `json:"results"`
}

type eventResult struct {
	Principal  string `json:"principal"`
	SourceIPV4 string `json:"source_ipv4"`
	Valid      bool   `json:"valid"`
}

type plugin struct {
	name        string
	path        string
	searchTerms []pluginTerm
}

func (p *plugin) runPlugin(input []byte) (err error) {
	var output bytes.Buffer
	cmd := exec.Command(p.path)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = &output
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error running %v: %v", p.path, err)
	}
	var res pluginResult
	err = json.Unmarshal(output.Bytes(), &res)
	if err != nil {
		return fmt.Errorf("error parsing output from %v: %v", p.path, err)
	}
	pluginResultCh <- res
	return nil
}

type pluginTerm struct {
	key   string
	value string
}

var pluginList []plugin

// Given an event query response from ES, return a byte slice suitable to be
// passed to a plugin
func pluginRequestDataFromES(r elastigo.SearchResult) (ret []byte, err error) {
	pr := pluginRequest{}
	for _, x := range r.Hits.Hits {
		pr.Events = append(pr.Events, x.Source)
	}
	ret, err = json.Marshal(pr)
	if err != nil {
		return ret, err
	}
	return ret, nil
}

func pluginFromFile(ppath string) (plugin, error) {
	np := plugin{}
	np.path = ppath

	fd, err := os.Open(ppath)
	if err != nil {
		return np, err
	}
	defer fd.Close()

	scnr := bufio.NewScanner(fd)
	for scnr.Scan() {
		buf := scnr.Text()
		args := strings.Split(buf, " ")
		if len(args) < 3 {
			continue
		}
		if args[0] != "#" {
			continue
		}
		if args[1] == "@@" && len(args) >= 3 {
			np.name = args[2]
		} else if args[1] == "@T" && len(args) >= 4 {
			nterm := pluginTerm{}
			nterm.key = args[2]
			nterm.value = args[3]
			np.searchTerms = append(np.searchTerms, nterm)
		}
	}
	err = scnr.Err()
	if err != nil {
		return np, err
	}

	return np, nil
}

func loadPlugins() error {
	dirents, err := ioutil.ReadDir(cfg.General.Plugins)
	if err != nil {
		return err
	}
	for _, x := range dirents {
		if !strings.HasSuffix(x.Name(), ".py") {
			continue
		}
		fname := path.Join(cfg.General.Plugins, x.Name())
		newplugin, err := pluginFromFile(fname)
		if err != nil {
			return err
		}
		pluginList = append(pluginList, newplugin)
		logf("added plugin %v (%v terms)", newplugin.name, len(newplugin.searchTerms))
	}
	return nil
}
