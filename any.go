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

type Any struct{ dedBase }

func newAny(cfg *Config, count, nulln int) *Any {
	return &Any{dedBase{
		cfg:   cfg,
		Count: count,
		Null:  nulln,
	}}
}

func (*Any) JsonType() JsonType { return JsonAny }

func (*Any) Accepts(v any, jt JsumType) float64 { return 1 }

func (a *Any) Example(v any, _ JsumType, _ float64) Deducer {
	a.Count++
	if v == nil {
		a.Null++
	}
	return a
}

func (a *Any) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(JsonAny)
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], a)
	return res
}

func (a *Any) Equal(d Deducer) bool {
	b, ok := d.(*Any)
	if !ok {
		return false
	}
	return a.dedBase.Equal(&b.dedBase)
}

func (*Any) JSONSchema() any { return struct{}{} }

func (a *Any) super() *dedBase { return &a.dedBase }
