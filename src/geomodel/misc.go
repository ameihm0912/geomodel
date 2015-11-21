// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor:
// - Aaron Meihm ameihm@mozilla.com

package main

import (
	"crypto/sha256"
	"fmt"
)

// Given an identifier for an object in the state index, produce the id
// value that will be used for the object.
func getObjectID(n string) (ret string, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("getObjectID() -> %v", e)
		}
	}()
	if n == "" {
		panic("zero length object id")
	}
	h := sha256.New()
	idstr := fmt.Sprintf("id-%v-%v", cfg.General.Context, n)
	h.Write([]byte(idstr))
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
