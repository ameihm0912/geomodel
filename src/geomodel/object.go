// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/jvehent/gozdef"
	"math"
	"os"
	"time"
)

// Describes an object stored in the state index used by geomodel. This
// could represent state metadata associated with entities in a context,
// or it could be a global state object. We use the same structure for
// both.
type object struct {
	ObjectID        string          `json:"object_id"`
	ObjectIDString  string          `json:"object_id_string"`
	Context         string          `json:"context"`
	State           objectState     `json:"state,omitempty"`
	Results         []objectResult  `json:"results,omitempty"`
	Geocenter       objectGeocenter `json:"geocenter"`
	LastUpdated     time.Time       `json:"last_updated"`
	WeightDeviation float64         `json:"weight_deviation"`
	NumCenters      int             `json:"numcenters"`
	Timestamp       time.Time       `json:"utctimestamp"`
}

func (o *object) addEventResult(e eventResult) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("addEventResult() -> %v", e)
		}
	}()

	if !e.Valid {
		panic("invalid result")
	}

	newres := objectResult{}
	newres.SourcePlugin = e.Name
	newres.BranchID = uuid.New()
	newres.Timestamp = e.Timestamp
	newres.Collapsed = false
	newres.Escalated = false
	newres.Weight = 1
	newres.SourceIPV4 = e.SourceIPV4
	err = geoObjectResult(&newres)
	if err != nil {
		panic(err)
	}
	o.Results = append(o.Results, newres)

	return nil
}

func (o *object) newFromPrincipal(principal string) {
	var err error
	o.ObjectID, err = getObjectID(principal)
	if err != nil {
		panic(err)
	}
	o.ObjectIDString = principal
	o.Context = cfg.General.Context
}

func (o *object) pruneExpiredEvents() error {
	newres := make([]objectResult, 0)
	for _, x := range o.Results {
		dur, err := time.ParseDuration(cfg.Timer.ExpireEvents)
		if err != nil {
			return err
		}
		cutoff := time.Now().UTC().Add(-1 * dur)
		if x.Timestamp.Before(cutoff) {
			continue
		}
		newres = append(newres, x)
	}
	o.Results = newres
	return nil
}

func (o *object) calculateWeightDeviation() {
	var fset []float64

	for _, x := range o.Results {
		// Only take into account branches that have not been
		// collapsed
		if x.Collapsed {
			continue
		}
		fset = append(fset, x.Weight)
	}
	if len(fset) <= 1 {
		o.WeightDeviation = 0
		return
	}
	var t0 float64 = 0
	for _, x := range fset {
		t0 += x
	}
	mean := t0 / float64(len(fset))
	var fset2 []float64
	for _, x := range fset {
		fset2 = append(fset2, math.Pow(x-mean, 2))
	}
	t0 = 0
	for _, x := range fset2 {
		t0 += x
	}
	variance := t0 / float64(len(fset2))
	o.WeightDeviation = math.Sqrt(variance)
}

func (o *object) markEscalated(branchID string) {
	for i := range o.Results {
		if o.Results[i].BranchID == branchID || o.Results[i].CollapseBranch == branchID {
			o.Results[i].Escalated = true
		}
	}
}

func (o *object) createAlertDetails(branchID string) (ret alertDetails, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("createAlertDetails() -> %v", e)
		}
	}()

	for _, x := range o.Results {
		if x.Collapsed {
			continue
		}
		if x.BranchID != branchID {
			continue
		}
		ret.Locality = x.Locality
		ret.SourceIPV4 = x.SourceIPV4
		ret.Informer = x.SourcePlugin
		ret.Principal = o.ObjectIDString
		ret.WeightDeviation = o.WeightDeviation
		break
	}
	if ret.Locality == "" {
		panic(err)
	}
	return ret, nil
}

func (o *object) sendAlert(branchID string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("sendAlert() -> %v", e)
		}
	}()

	ad, err := o.createAlertDetails(branchID)
	if err != nil {
		panic(err)
	}
	hname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	ac := gozdef.ApiConf{Url: cfg.MozDef.MozDefURL}
	pub, err := gozdef.InitApi(ac)
	if err != nil {
		panic(err)
	}
	newev := gozdef.Event{}
	newev.Notice()
	newev.Timestamp = time.Now().UTC()
	newev.Category = "geomodelnotice"
	newev.ProcessName = os.Args[0]
	newev.ProcessID = float64(os.Getpid())
	newev.Hostname = hname
	newev.Source = "geomodel"
	newev.Tags = append(newev.Tags, "geomodel")
	newev.Details = ad
	newev.Summary = ad.makeSummary()

	err = pub.Send(newev)
	if err != nil {
		panic(err)
	}
	return nil
}

func (o *object) alertAnalyze() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("alertAnalyze() -> %v", e)
		}
	}()

	o.calculateWeightDeviation()
	for i := range o.Results {
		if o.Results[i].Collapsed {
			continue
		}
		if o.Results[i].Escalated {
			continue
		}
		logf("[NOTICE] new geocenter for %v (%v)", o.ObjectIDString, o.Results[i].Locality)
		o.markEscalated(o.Results[i].BranchID)
		if !cfg.noSendAlert {
			err := o.sendAlert(o.Results[i].BranchID)
			if err != nil {
				panic(err)
			}
		}
	}
	return nil
}

// Specific to global state tracking
type objectState struct {
	TimeEndpoint time.Time `json:"time_endpoint,omitempty"`
}

// Principal geocenter
type objectGeocenter struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Locality  string  `json:"locality,omitempty"`
	AvgDist   float64 `json:"avg_dist,omitempty"`
	Weight    float64 `json:"weight"`
}

// Single authentication result for a principal
type objectResult struct {
	SourcePlugin string  `json:"source_plugin"`
	BranchID     string  `json:"branch_id"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Locality     string  `json:"locality"`
	SourceIPV4   string  `json:"source_ipv4"`
	Weight       float64 `json:"weight"`
	Escalated    bool    `json:"escalated"`

	Timestamp time.Time `json:"timestamp"`

	Collapsed      bool   `json:"collapsed"`
	CollapseBranch string `json:"collapse_branch,omitempty"`
}

// Describes an individual alert
type alertDetails struct {
	Principal       string  `json:"principal"`
	Locality        string  `json:"locality"`
	WeightDeviation float64 `json:"weight_deviation"`
	SourceIPV4      string  `json:"source_ipv4"`
	Informer        string  `json:"informer"`
}

func (ad *alertDetails) makeSummary() string {
	ret := fmt.Sprintf("%v NEWLOCATION %v access from %v (%v)", ad.Principal,
		ad.Locality, ad.SourceIPV4, ad.Informer)
	ret += fmt.Sprintf(" [deviation:%v]", ad.WeightDeviation)
	return ret
}
