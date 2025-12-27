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
)

type Array struct {
	dedBase
	MinLen int     `json:"min-len"`
	MaxLen int     `json:"max-len"`
	Elem   Deducer `json:"elements"`
}

func newArrJson(cfg *Config, count, nulln int) *Array {
	res := &Array{
		dedBase: dedBase{cfg: cfg, Count: count, Null: nulln},
		MinLen:  -1,
		MaxLen:  -1,
		Elem:    NewUnknown(cfg),
	}
	return res
}

func (*Array) JsonType() JsonType { return JsonArray }

func (a *Array) Accepts(_ any, jt JsumType) float64 {
	if jt.t == JsonArray {
		return 1
	}
	return 0
}

func (a *Array) Example(v any, jt JsumType, _ float64) Deducer {
	if jt.t == JsonNull {
		a.Count++
		a.Null++
		return a
	}
	switch jt.v {
	case jsonArrAny:
		a.Count++
		v := v.([]any)
		l := len(v)
		if a.MinLen < 0 {
			a.MinLen, a.MaxLen = l, l
		} else if l < a.MinLen {
			a.MinLen = l
		} else {
			a.MaxLen = max(a.MaxLen, l)
		}
		for _, e := range v {
			a.Elem = a.Elem.Example(e, JsonTypeOf(e), UnknownAccept)
		}
		return a
		// TODO case jsonArrRSlice:
	}
	return newAny(a.cfg, a.Count+1, a.Null) // TODO Why not union?
}

func (a *Array) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(JsonArray)
	if a.MaxLen == 0 {
		hash.WriteByte(0)
	} else {
		hash.WriteByte(1)
	}
	eh := a.Elem.Hash(dh)
	binary.Write(hash, hashEndian, eh)
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], a)
	return res
}

func (a *Array) Equal(d Deducer) bool {
	b, ok := d.(*Array)
	if !ok {
		return false
	}
	if !a.dedBase.Equal(&b.dedBase) {
		return false
	}
	if (a.MinLen == 0) != (b.MinLen == 0) {
		return false
	}
	return a.Elem.Equal(b.Elem)
}

func (a *Array) JSONSchema() any {
	res := jscmArray{
		jscmType: jscmType{Type: "array"},
		Items:    a.Elem.JSONSchema(),
		MinItems: a.MinLen,
		MaxItems: a.MaxLen,
	}
	if a.Null > 0 {
		return []any{"null", res}
	}
	return res
}

func (a *Array) super() *dedBase { return &a.dedBase }
