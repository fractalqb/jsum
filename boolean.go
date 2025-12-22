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

type Boolean struct {
	dedBase
	TrueNo  int `json:"true"`
	FalseNo int `json:"false"`
}

func newBool(cfg *Config, count, nulln int) *Boolean {
	return &Boolean{dedBase: dedBase{cfg: cfg, Count: count, Null: nulln}}
}

func (a *Boolean) Accepts(v any) bool { return JsonTypeOf(v) == JsonBoolean }

func (a *Boolean) Example(v any) Deducer {
	vjt := JsonTypeOf(v)
	switch vjt {
	case 0:
		a.Count++
		a.Null++
		return a
	case JsonBoolean:
		a.Count++
		if v.(bool) {
			a.TrueNo++
		} else {
			a.FalseNo++
		}
		return a
	}
	return newUnion(a, Deduce(a.cfg, v))
}

func (a *Boolean) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(JsonBoolean)
	if a.cfg.DedupBool&DedupBoolFalse != 0 {
		if a.FalseNo > 0 {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	if a.cfg.DedupBool&DedupBoolTrue != 0 {
		if a.TrueNo > 0 {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], a)
	return res
}

func (s *Boolean) Equal(d Deducer) bool {
	b, ok := d.(*Boolean)
	if !ok {
		return false
	}
	if (s.FalseNo > 0) != (b.FalseNo > 0) {
		return false
	}
	if (s.TrueNo > 0) != (b.TrueNo > 0) {
		return false
	}
	res := s.dedBase.Equal(&b.dedBase)
	return res
}

func (a *Boolean) JSONSchema() any {
	res := jscmString{jscmType: jscmType{Type: "boolean"}}
	if a.Null > 0 {
		return []any{"null", res}
	}
	return res
}

func (s *Boolean) super() *dedBase { return &s.dedBase }
