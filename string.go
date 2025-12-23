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
	"math"
	"time"
	"unicode/utf8"
)

type Format int

const (
	DateTimeFormat = 1 + iota
)

type String struct {
	dedBase
	Stats  map[string]int
	Format Format `json:"format,omitempty"`
}

func newString(cfg *Config, count, nulln int) *String {
	return &String{
		dedBase: dedBase{cfg: cfg, Count: count, Null: nulln},
		Stats:   make(map[string]int),
	}
}

func (a *String) Accepts(jt JsonType) bool { return jt.t == jsonString }

func (a *String) Example(v any, jt JsonType) Deducer {
	switch jt.t {
	case jsonNull:
		a.Count++
		a.Null++
	case jsonString:
		a.Count++
		switch jt.v {
		case jsonStrString:
			v := v.(string)
			if fmt := stringFormat(v); fmt == 0 {
				a.Format = 0
			} else if len(a.Stats) == 0 {
				a.Format = fmt
			} else if fmt != a.Format {
				a.Format = 0
			}
			a.Stats[v]++
		case jsonStrTime:
			v := v.(time.Time)
			if len(a.Stats) == 0 {
				a.Format = DateTimeFormat
			}
			s := v.Format(time.RFC3339)
			a.Stats[s]++
		}
		return a
	}
	return newUnion(a, Deduce(a.cfg, v)) // TODO reuse jt
}

func stringFormat(s string) Format {
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return DateTimeFormat
	}
	return 0
}

func (s *String) Hash(dh DedupHash) uint64 {
	hash := s.dedBase.startHash(jsonString)
	if s.cfg.DedupString&DedupStringEmpty != 0 {
		if s.Stats[""] > 0 {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], s)
	return res
}

func (s *String) Equal(d Deducer) bool {
	b, ok := d.(*String)
	if !ok {
		return false
	}
	if !s.dedBase.Equal(&b.dedBase) {
		return false
	}
	if se, de := s.Stats[""], b.Stats[""]; (se > 0) != (de > 0) {
		return false
	}
	return true
}

func (a *String) JSONSchema() any {
	scm := jscmString{
		jscmType: jscmType{Type: "string"},
	}
	switch a.Format {
	case 0:
		mi, ma := math.MaxInt, 0
		for s := range a.Stats {
			n := utf8.RuneCountInString(s)
			mi = min(mi, n)
			ma = max(ma, n)
		}
		scm.MinLen = new(int)
		*scm.MinLen = mi
		scm.MaxLen = new(int)
		*scm.MaxLen = ma
	case DateTimeFormat:
		scm.Format = "date-time"
	}
	if a.Null > 0 {
		return []any{"null", scm}
	}
	return scm
}

func (s *String) super() *dedBase { return &s.dedBase }
