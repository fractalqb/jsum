/*
A tool to analyse the structure of JSON from a set of example JSON values.
Copyright (C) 2025  Marcus Perlick

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package jsum

import (
	"bytes"
	"encoding/json"
	"math"
	"strings"
	"testing"

	"git.fractalqb.de/fractalqb/testerr"
)

var (
	testDedBase = dedBase{Count: 4711, Null: 1174}
	testCfg     Config
)

func TestStateIO_rdString(t *testing.T) {
	var (
		buf bytes.Buffer
		sio StateIO
	)
	testerr.Shall(sio.Write(&buf, &Union{Variants: []Deducer{
		&String{Stats: map[string]int{"foo": 1}},
		&String{Stats: map[string]int{"foo": 2}},
	}})).BeNil(t)
	testerr.Shall1(sio.Read(&buf, &testCfg)).BeNil(t)
}

func testDedEq(t *testing.T, l, r Deducer) bool {
	var lb, rb strings.Builder
	testerr.Shall(json.NewEncoder(&lb).Encode(l)).BeNil(t)
	ls := strings.TrimSpace(lb.String())
	t.Log(ls)
	testerr.Shall(json.NewEncoder(&rb).Encode(r)).BeNil(t)
	if rs := strings.TrimSpace(rb.String()); ls != rs {
		t.Errorf("%s =/= %s", ls, rs)
		return false
	}
	return true
}

func TestTestIO_writeRead(t *testing.T) {
	testDedWriteRead := func(t *testing.T, ded Deducer) {
		var (
			buf bytes.Buffer
			sio StateIO
		)
		testerr.Shall(sio.Write(&buf, ded)).BeNil(t)
		ede := testerr.Shall1(sio.Read(&buf, &testCfg)).BeNil(t)
		testDedEq(t, ede, ded)
	}

	t.Run("Unknown", func(t *testing.T) {
		testDedWriteRead(t, &Unknown{dedBase: testDedBase})
	})
	t.Run("Boolean", func(t *testing.T) {
		testDedWriteRead(t, &Boolean{dedBase: testDedBase,
			TrueNo:  4,
			FalseNo: 7,
		})
	})
	t.Run("Number", func(t *testing.T) {
		testDedWriteRead(t, &Number{dedBase: testDedBase,
			Min: -math.Pi, Max: math.E,
			IsFloat: true,
			HasFrac: true,
		})
	})
	t.Run("String", func(t *testing.T) {
		testDedWriteRead(t, &String{dedBase: testDedBase,
			Stats: map[string]int{
				"foo": 1,
				"bar": 2,
				"baz": 3,
			},
			Format: DateTimeFormat,
		})
	})
	t.Run("Object", func(t *testing.T) {
		testDedWriteRead(t, &Object{dedBase: testDedBase,
			Members: map[string]Member{
				"name": {
					Occurence: 111,
					Ded:       newString(&testCfg, 3, 1),
				},
				"id": {
					Occurence: 222,
					Ded:       &Number{dedBase: testDedBase, Min: -100, Max: 100},
				},
			},
		})
	})
	t.Run("Array", func(t *testing.T) {
		testDedWriteRead(t, &Array{dedBase: testDedBase,
			MinLen: 1,
			MaxLen: 1024,
			Elem:   newString(&testCfg, 3, 1),
		})
	})
	t.Run("Union", func(t *testing.T) {
		testDedWriteRead(t, &Union{dedBase: testDedBase,
			Variants: []Deducer{
				newString(&testCfg, 3, 1),
				&Number{dedBase: testDedBase, Min: -100, Max: 100},
			},
		})
	})
	t.Run("Any", func(t *testing.T) {
		testDedWriteRead(t, &Any{dedBase: testDedBase})
	})
}
