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
	"sort"
	"strings"
	"time"
)

type genericAlert interface {
	makeSummary() (string, error)
}

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

func (o *object) upgradeState() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("upgradeState() -> %v", e)
		}
	}()

	// Update any object results that use the old locality format
	for i := range o.Results {
		if o.Results[i].OldLocality == "" {
			continue
		}
		sv := strings.Split(o.Results[i].OldLocality, ",")
		// We should have 2 values here
		if len(sv) != 2 {
			panic("unable to upgrade old format locality")
		}
		o.Results[i].Locality.City = strings.Trim(sv[0], " ")
		o.Results[i].Locality.Country = strings.Trim(sv[1], " ")
		o.Results[i].OldLocality = ""
	}

	return nil
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
	var newres []objectResult
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
	var t0 float64
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

func (o *object) createAlertDetailsMovement(objlist []objectResult) (ret alertDetailsMovement, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("createAlertDetailsMovement() -> %v", e)
		}
	}()

	// This alert should have at least two localities in objlist, otherwise
	// it should not trigger
	if len(objlist) < 2 {
		panic("objlist length does not make sense")
	}

	ret.Localities = objlist
	ret.Principal = o.ObjectIDString

	return ret, nil
}

func (o *object) createAlertDetailsBranch(branchID string) (ret alertDetailsBranch, err error) {
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
		ret.Locality.City = x.Locality.City
		ret.Locality.Country = x.Locality.Country
		ret.Latitude = x.Latitude
		ret.Longitude = x.Longitude
		ret.SourceIPV4 = x.SourceIPV4
		ret.Informer = x.SourcePlugin
		ret.Principal = o.ObjectIDString
		ret.WeightDeviation = o.WeightDeviation
		ret.Timestamp = x.Timestamp
		break
	}
	if ret.Locality.City == "" || ret.Locality.Country == "" {
		panic("unable to create alert with no locality information")
	}
	return ret, nil
}

func (o *object) sendMovementAlert(objlist []objectResult) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("sendMovementAlert() -> %v", e)
		}
	}()

	ad, err := o.createAlertDetailsMovement(objlist)
	if err != nil {
		panic(err)
	}
	err = sendAlert(&ad)
	if err != nil {
		panic(err)
	}
	return nil
}

func (o *object) sendBranchAlert(branchID string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("sendAlert() -> %v", e)
		}
	}()

	ad, err := o.createAlertDetailsBranch(branchID)
	if err != nil {
		panic(err)
	}
	err = ad.addPreviousEvent(o, branchID)
	if err != nil {
		panic(err)
	}
	err = ad.calculateSeverity()
	if err != nil {
		panic(err)
	}
	err = sendAlert(&ad)
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
		lval, err := o.Results[i].Locality.assemble()
		if err != nil {
			panic(err)
		}
		logf("[NOTICE] new geocenter for %v (%v)", o.ObjectIDString, lval)
		o.markEscalated(o.Results[i].BranchID)
		if !cfg.noSendAlert {
			err := o.sendBranchAlert(o.Results[i].BranchID)
			if err != nil {
				panic(err)
			}
		}
	}

	// Now that new gencenters have been handled, apply a heuristic on the entire
	// state to create any additional alerts required. Given a window of time, get
	// a list of all authentication events that have occurred. If we see events
	// occuring within that window, where the distance is unreasonable given the
	// window, also create an alert for this.
	//
	// The distance and time frame are sourced from the configuration file.
	_, err = o.analyzeUsageWithinWindow()
	if err != nil {
		panic(err)
	}
	return nil
}

// Apply movement heuristic to results stored in object; returns true if we would have
// generated an alert, or false otherwise
func (o *object) analyzeUsageWithinWindow() (ret []objectResult, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("analyzeUsageWithinWindow() -> %v", e)
		}
	}()

	dur, err := time.ParseDuration(cfg.Geo.MovementWindow)
	if err != nil {
		panic(err)
	}
	cutoff := time.Now().UTC().Add(-1 * dur)

	resl := make([]objectResult, 0)

	// Build a slice of all the results we want to consider
	for _, x := range o.Results {
		if x.Timestamp.Before(cutoff) {
			continue
		}
		resl = append(resl, x)
	}

	// Filter this list down further to the latest event in each geocenter within
	// the window
	geocenters := make(map[string]objectResult)
	for _, x := range resl {
		var bid string
		if x.Collapsed {
			bid = x.CollapseBranch
		} else {
			bid = x.BranchID
		}

		compval, ok := geocenters[bid]
		if !ok {
			geocenters[bid] = x
			continue
		}
		if x.Timestamp.After(compval.Timestamp) {
			geocenters[bid] = x
		}
	}

	// Compare the distances between each of our candidate results, if any
	// exceed the configuration movement distance create an alert for this.
	largest := 0.0
	for k1, v1 := range geocenters {
		for k2, v2 := range geocenters {
			if k2 == k1 {
				continue
			}
			dv := kmBetweenTwoPoints(v1.Latitude, v1.Longitude,
				v2.Latitude, v2.Longitude)
			if dv > largest {
				largest = dv
			}
		}
	}

	// If the largest value is less than the movement distance, we are done
	// here
	if largest < float64(cfg.Geo.MovementDistance) {
		return ret, nil
	}

	// Build the slice of geocenters we want to include in the alert
	alertlist := make([]objectResult, 0)
	for _, v := range geocenters {
		alertlist = append(alertlist, v)
	}

	if !cfg.noSendAlert {
		err = o.sendMovementAlert(alertlist)
		if err != nil {
			panic(err)
		}
	}

	return alertlist, nil
}

// Specific to global state tracking
type objectState struct {
	TimeEndpoint time.Time `json:"time_endpoint,omitempty"`
}

// Locality
type Locality struct {
	City    string `json:"city"`
	Country string `json:"country"`
}

func (l *Locality) assemble() (string, error) {
	if l.City == "" || l.Country == "" {
		return "", fmt.Errorf("unable to assemble locality with empty values")
	}
	return l.City + ", " + l.Country, nil
}

// Principal geocenter
type objectGeocenter struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`

	Locality Locality `json:"locality_details"`

	AvgDist float64 `json:"avg_dist,omitempty"`
	Weight  float64 `json:"weight"`

	// Compatibility with older state documents
	OldLocality string `json:"locality,omitempty"`
}

// Single authentication result for a principal
type objectResult struct {
	SourcePlugin string   `json:"source_plugin"`
	BranchID     string   `json:"branch_id"`
	Latitude     float64  `json:"latitude"`
	Longitude    float64  `json:"longitude"`
	Locality     Locality `json:"locality_details"`
	SourceIPV4   string   `json:"source_ipv4"`
	Weight       float64  `json:"weight"`
	Escalated    bool     `json:"escalated"`

	Timestamp time.Time `json:"timestamp"`

	Collapsed      bool   `json:"collapsed"`
	CollapseBranch string `json:"collapse_branch,omitempty"`

	// Compatibility with older state documents
	OldLocality string `json:"locality,omitempty"`
}

// Describes an individual alert for a movement hit
type alertDetailsMovement struct {
	Principal  string         `json:"principal"`
	Localities []objectResult `json:"localities"`
}

func (ad *alertDetailsMovement) makeSummary() (string, error) {
	ret := fmt.Sprintf("%v MOVEMENT window violation (", ad.Principal)
	iv := len(ad.Localities)
	if iv > 3 {
		iv = 3
	}
	for i := 0; i < iv; i++ {
		if i > 0 {
			ret += ","
		}
		lval, err := ad.Localities[i].Locality.assemble()
		if err != nil {
			return "", err
		}
		ret += "[" + lval + "]"
	}
	ret += ")"
	return ret, nil
}

// Describes an individual alert for a branch
type alertDetailsBranch struct {
	Principal       string    `json:"principal"`
	Locality        Locality  `json:"locality_details"`
	Latitude        float64   `json:"latitude"`
	Longitude       float64   `json:"longitude"`
	Timestamp       time.Time `json:"event_time"`
	WeightDeviation float64   `json:"weight_deviation"`
	SourceIPV4      string    `json:"source_ipv4"`
	Informer        string    `json:"informer"`
	Severity        int       `json:"severity"`

	PrevLocality  Locality  `json:"prev_locality_details"`
	PrevLatitude  float64   `json:"prev_latitude"`
	PrevLongitude float64   `json:"prev_longitude"`
	PrevTimestamp time.Time `json:"prev_timestamp"`
	PrevDistance  float64   `json:"prev_distance"`
}

func (ad *alertDetailsBranch) makeSummary() (string, error) {
	lval, err := ad.Locality.assemble()
	if err != nil {
		return "", err
	}
	ret := fmt.Sprintf("%v NEWLOCATION %v access from %v (%v)", ad.Principal,
		lval, ad.SourceIPV4, ad.Informer)
	ret += fmt.Sprintf(" [deviation:%v]", ad.WeightDeviation)
	if ad.PrevLocality.Country != "" && ad.PrevLocality.City != "" {
		dur := ad.Timestamp.Sub(ad.PrevTimestamp)
		hs := dur.Hours()
		var sstr string
		if hs > 1 {
			sstr = fmt.Sprintf("approx %.2f hours before", dur.Hours())
		} else {
			sstr = "within hour before"
		}
		lval2, err := ad.PrevLocality.assemble()
		if err != nil {
			return "", err
		}
		ret += fmt.Sprintf(" last activity was from %v (%.0f km away) %v", lval2,
			ad.PrevDistance, sstr)
	} else {
		ret += ", no previous locations stored in window"
	}
	return ret, nil
}

func (ad *alertDetailsBranch) calculateSeverity() error {
	// Default to a severity value of 1, we will adjust up based on the
	// outcome of this function.
	ad.Severity = 1

	// If the previous country is a different country from this new alert,
	// increase the severity.
	if ad.PrevLocality.Country != "" {
		if ad.PrevLocality.Country != ad.Locality.Country {
			ad.Severity++
		}
	}
	return nil
}

// Locate the event in this object that is unrelated to the alert event,
// and is closest to it based on the timestamp
func (ad *alertDetailsBranch) addPreviousEvent(o *object, branchID string) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("addPreviousEvent() -> %v", e)
		}
	}()

	var res *objectResult
	var latest time.Time
	for i := range o.Results {
		if o.Results[i].BranchID == branchID {
			continue
		} else if o.Results[i].CollapseBranch == branchID {
			continue
		}
		if latest.Before(o.Results[i].Timestamp) {
			res = &o.Results[i]
		}
	}
	if res == nil {
		return nil
	}

	ad.PrevLocality = res.Locality
	ad.PrevLatitude = res.Latitude
	ad.PrevLongitude = res.Longitude
	ad.PrevTimestamp = res.Timestamp
	ad.PrevDistance = kmBetweenTwoPoints(ad.Latitude, ad.Longitude,
		ad.PrevLatitude, ad.PrevLongitude)
	return nil
}

func sendAlert(d genericAlert) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("sendAlert() -> %v", e)
		}
	}()

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
	newev.Details = d
	newev.Summary, err = d.makeSummary()
	if err != nil {
		panic(err)
	}

	err = pub.Send(newev)
	if err != nil {
		panic(err)
	}

	return nil
}
