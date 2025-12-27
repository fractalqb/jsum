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
	"iter"
	"math"
	"slices"
	"sort"
)

type Object struct {
	dedBase
	Members map[string]Member `json:"members"`
}

type Member struct {
	Occurence int     `json:"occurence"`
	Ded       Deducer `json:"type"`
}

func newObjJson(cfg *Config, count, nulln int, m iter.Seq2[string, any]) *Object {
	res := &Object{
		dedBase: dedBase{cfg: cfg, Count: count, Null: nulln},
		Members: make(map[string]Member),
	}
	if m != nil {
		res.Count++
	}
	res.mergeMap(m)
	return res
}

func (*Object) JsonType() JsonType { return JsonObject }

func (o *Object) Accepts(v any, jt JsumType) float64 {
	switch jt.v {
	case jsonObjStrAny:
		acpt := o.acceptance(objStrAnySeq(v))
		return max(math.SmallestNonzeroFloat64, acpt)
	}
	return 0
}

func (o *Object) Example(v any, jt JsumType, acpt float64) Deducer {
	if jt.t == JsonNull {
		o.Count++
		o.Null++
		return o
	}
	switch jt.v {
	case jsonObjStrAny:
		if acpt < 0 {
			acpt = o.acceptance(objStrAnySeq(v))
		}
		if o.cfg.Union.MergeRejectMax == 0 || acpt > o.cfg.Union.MergeRejectMax {
			o.Count++
			o.mergeMap(objStrAnySeq(v))
			return o
		}
		u := newUnion(o)
		return u.Example(v, jt, UnknownAccept)
		// TODO case jsonObjRMap:
	}
	return newAny(o.cfg, o.Count+1, o.Null) // TODO Why not union?
}

func (o *Object) acceptance(m iter.Seq2[string, any]) float64 {
	common, vmiss := 0, 0
	for n, v := range m {
		vjt := JsonTypeOf(v)
		if m, ok := o.Members[n]; ok && m.Ded.JsonType() == vjt.JsonType() {
			common++
		} else {
			vmiss++
		}
	}
	total := len(o.Members) + vmiss
	if total == 0 {
		return 1
	}
	return float64(common) / float64(total)
}

func (o *Object) mergeMap(m iter.Seq2[string, any]) {
	for k, v := range m {
		if m, ok := o.Members[k]; ok {
			o.Members[k] = Member{
				Occurence: m.Occurence + 1,
				Ded:       m.Ded.Example(v, JsonTypeOf(v), UnknownAccept),
			}
		} else {
			o.Members[k] = Member{Occurence: 1, Ded: Deduce(o.cfg, v)}
		}
	}
}

func (o *Object) Hash(dh DedupHash) uint64 {
	type memhash struct {
		n string
		h uint64
	}
	mems := make([]memhash, 0, len(o.Members))
	for n, m := range o.Members {
		mh := m.Ded.Hash(dh)
		mems = append(mems, memhash{n, mh})
	}
	sort.Slice(mems, func(i, j int) bool { return mems[i].n < mems[j].n })
	hash := o.dedBase.startHash(JsonObject)
	for _, m := range mems {
		binary.Write(hash, hashEndian, m.h)
	}
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], o)
	return res
}

func (o *Object) Equal(d Deducer) bool {
	b, ok := d.(*Object)
	if !ok {
		return false
	}
	res := o.dedBase.Equal(&b.dedBase) && len(o.Members) == len(b.Members)
	if res {
		for i := range o.Members {
			if res = o.Members[i].Ded.Equal(b.Members[i].Ded); !res {
				break
			}
		}
	}
	return res
}

func (o *Object) JSONSchema() any {
	res := jscmObj{
		jscmType: jscmType{Type: "object"},
		Props:    make(map[string]any, len(o.Members)),
	}
	for n, t := range o.Members {
		ded := t.Ded
		res.Props[n] = ded.JSONSchema()
		if t.Occurence == o.Count {
			res.Required = append(res.Required, n)
		}
	}
	slices.Sort(res.Required)
	if o.Null > 0 {
		return []any{"null", res}
	}
	return res
}

func (o *Object) super() *dedBase { return &o.dedBase }

func objStrAnySeq(v any) iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		m := v.(map[string]any)
		for n, v := range m {
			if !yield(n, v) {
				return
			}
		}
	}
}
