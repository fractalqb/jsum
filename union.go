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
	"encoding/binary"
	"slices"
	"sort"
)

type Union struct {
	dedBase
	Variants []Deducer `json:"variants"`
}

func newUnion(a, b Deducer) *Union {
	return &Union{
		dedBase: dedBase{
			cfg:   a.super().cfg,
			Count: a.super().Count + b.super().Count,
			Null:  a.super().Null + b.super().Null,
		},
		Variants: []Deducer{a, b},
	}
}

func (u *Union) Accepts(jt JsonType) bool {
	for _, d := range u.Variants {
		if d.Accepts(jt) {
			return true
		}
	}
	return false
}

func (u *Union) Example(v any, jt JsonType) Deducer {
	u.Count++
	if v == nil {
		u.Null++
		return u
	}
	for i, d := range u.Variants {
		if d.Accepts(jt) {
			u.Variants[i] = d.Example(v, jt)
			return u
		}
	}
	// TODO When does Union switch to Any
	u.Variants = append(u.Variants, Deduce(u.cfg, v))
	return u
}

func (u *Union) Hash(dh DedupHash) uint64 {
	dhs := make([]uint64, 0, len(u.Variants))
	for _, ed := range u.Variants {
		dhs = append(dhs, ed.Hash(dh))
	}
	sort.Slice(dhs, func(i, j int) bool { return dhs[i] < dhs[j] })
	hash := u.dedBase.startHash(jsonUnion)
	for _, h := range dhs {
		binary.Write(hash, hashEndian, h)
	}
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], u)
	return res
}

func (u *Union) Equal(d Deducer) bool {
	b, ok := d.(*Union)
	if !ok {
		return false
	}
	res := u.dedBase.Equal(&b.dedBase) && len(u.Variants) == len(b.Variants)
	if res {
		for i := range b.Variants {
			if res = u.Variants[i].Equal(b.Variants[i]); !res {
				break
			}
		}
	}
	return res
}

func (u *Union) JSONSchema() any {
	scm := jscmAnyOf{AnyOf: make([]any, len(u.Variants))}
	for i, v := range u.Variants {
		scm.AnyOf[i] = v.JSONSchema()
	}
	if u.Null > 0 && !slices.Contains(scm.AnyOf, "null") {
		scm.AnyOf = append(scm.AnyOf, "null")
	}
	return scm
}

func (u *Union) super() *dedBase { return &u.dedBase }
