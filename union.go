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

func newUnion(d Deducer) *Union {
	return &Union{
		dedBase: dedBase{
			cfg:   d.super().cfg,
			Count: d.super().Count,
			Null:  d.super().Null,
		},
		Variants: []Deducer{d},
	}
}

func (*Union) JsonType() JsonType { return JsonUnion }

func (u *Union) Accepts(a any, jt JsumType) (res float64) {
	for _, d := range u.Variants {
		res = max(res, d.Accepts(a, jt))
	}
	return res
}

func (u *Union) Example(v any, jt JsumType, _ float64) Deducer {
	u.Count++
	if v == nil {
		u.Null++
		return u
	}
	tset := NewTypeSet(jt.JsonType())
	avar, amax := -1, 0.0
	for i, d := range u.Variants {
		tset.Add(d.JsonType())
		if da := d.Accepts(v, jt); da > amax {
			avar, amax = i, da
		}
	}
	if amax > u.cfg.Union.MergeRejectMax {
		u.Variants[avar] = u.Variants[avar].Example(v, jt, amax)
		return u
	}
	for _, comb := range u.cfg.Union.Combine {
		if comb&tset == tset {
			u.Variants = append(u.Variants, Deduce(u.cfg, v))
			return u
		}
	}
	return newAny(u.cfg, u.Count, u.Null)
}

func (u *Union) Hash(dh DedupHash) uint64 {
	dhs := make([]uint64, 0, len(u.Variants))
	for _, ed := range u.Variants {
		dhs = append(dhs, ed.Hash(dh))
	}
	sort.Slice(dhs, func(i, j int) bool { return dhs[i] < dhs[j] })
	hash := u.dedBase.startHash(JsonUnion)
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
