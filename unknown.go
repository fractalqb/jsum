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

import "fmt"

type Unknown struct{ dedBase }

func NewUnknown(cfg *Config) *Unknown {
	return &Unknown{dedBase: dedBase{cfg: cfg}}
}

func (a *Unknown) JsonType() JsonType { return JsonUnknown }

func (a *Unknown) Accepts(any, JsumType) float64 { return 1 }

func (a *Unknown) Example(v any, jt JsumType, _ float64) Deducer {
	switch jt.t {
	case JsonNull:
		a.Count++
		a.Null++
		return a
	case JsonString:
		str := newString(a.cfg, a.Count, a.Null)
		return str.Example(v, jt, UnknownAccept)
	case JsonNumber:
		ded := newNum(a.cfg, a.Count, a.Null)
		return ded.Example(v, jt, UnknownAccept)
	case JsonBoolean:
		b := newBool(a.cfg, a.Count, a.Null)
		return b.Example(v, jt, UnknownAccept)
	case JsonObject:
		switch v := v.(type) {
		case map[string]any:
			ded := newObjJson(a.cfg, a.Count, a.Null)
			return ded.Example(v, jt, UnknownAccept)
		}
	case JsonArray:
		switch v := v.(type) {
		case []any:
			ded := newArrJson(a.cfg, a.Count, a.Null)
			return ded.Example(v, jt, UnknownAccept)
		}
	}
	return newInvalid(fmt.Errorf("cannot deduce type from: %T", v))
}

func (a *Unknown) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(JsonUnknown)
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], a)
	return res
}

func (a *Unknown) Equal(d Deducer) bool {
	b, ok := d.(*Unknown)
	if !ok {
		return false
	}
	return a.dedBase.Equal(&b.dedBase)
}

func (*Unknown) JSONSchema() any { return false }

func (a *Unknown) super() *dedBase { return &a.dedBase }
