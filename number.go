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
	"math"
)

type Number struct {
	dedBase
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	IsFloat bool    `json:"is-float"`
	HasFrac bool    `json:"has-frac"`
}

func newNum(cfg *Config, count, nulln int) *Number {
	res := &Number{dedBase: dedBase{cfg: cfg, Count: count, Null: nulln},
		Min: math.Inf(1),
		Max: math.Inf(-1),
	}
	return res
}

func (nr *Number) Accepts(jt JsonType) bool { return jt.t == jsonNumber }

func (nr *Number) Example(v any, jt JsonType) Deducer {
	switch jt.t {
	case jsonNull:
		nr.Count++
		nr.Null++
	case jsonNumber:
		nr.Count++
		x, isFloat := asNumber(v, jt.v)
		_, frac := math.Modf(x)
		nr.Min = min(nr.Min, x)
		nr.Max = max(nr.Max, x)
		nr.IsFloat = nr.IsFloat || isFloat
		nr.HasFrac = nr.HasFrac || frac != 0
	default:
		return newUnion(nr, Deduce(nr.cfg, v))
	}
	return nr
}

func (nr *Number) Hash(dh DedupHash) uint64 {
	hash := nr.dedBase.startHash(jsonNumber)
	if nr.cfg.DedupNumber&DedpuNumberIntFloat != 0 {
		if nr.IsFloat {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	if nr.cfg.DedupNumber&DedupNumberFrac != 0 {
		if nr.HasFrac {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	if nr.cfg.DedupNumber&DedupNumberMin != 0 {
		binary.Write(hash, hashEndian, nr.Min)
	}
	if nr.cfg.DedupNumber&DedupNumberMax != 0 {
		binary.Write(hash, hashEndian, nr.Max)
	}
	if nr.cfg.DedupNumber&DedupNumberNeg != 0 {
		if nr.Min < 0 {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	if nr.cfg.DedupNumber&DedupNumberPos != 0 {
		if nr.Max > 0 {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], nr)
	return res
}

func (nr *Number) Equal(d Deducer) bool {
	b, ok := d.(*Number)
	if !ok {
		return false
	}
	res := nr.dedBase.Equal(&b.dedBase)
	if res && nr.cfg.DedupNumber&DedpuNumberIntFloat != 0 {
		res = nr.IsFloat == b.IsFloat
	}
	if res && nr.cfg.DedupNumber&DedupNumberFrac != 0 {
		res = nr.HasFrac == b.HasFrac
	}
	if res && nr.cfg.DedupNumber&DedupNumberMin != 0 {
		res = nr.Min == b.Min
	}
	if res && nr.cfg.DedupNumber&DedupNumberMax != 0 {
		res = nr.Max == b.Max
	}
	if res && nr.cfg.DedupNumber&DedupNumberNeg != 0 {
		res = (nr.Min < 0) == (b.Min < 0)
	}
	if res && nr.cfg.DedupNumber&DedupNumberPos != 0 {
		res = (nr.Max > 0) == (b.Max > 0)
	}
	return res
}

func (nr *Number) JSONSchema() any {
	scm := jscmNumber{
		Min: new(float64),
		Max: new(float64),
	}
	if nr.IsFloat && nr.HasFrac {
		scm.Type = "number"
	} else {
		scm.Type = "integer"
	}
	*scm.Min = nr.Min
	*scm.Max = nr.Max
	if nr.Null > 0 {
		return []any{"null", scm}
	}
	return scm
}

func (nr *Number) super() *dedBase { return &nr.dedBase }

func asNumber(n any, v jsonVariant) (float64, bool) {
	switch v {
	case jsonNumFloat64:
		return n.(float64), true
	case jsonNumFloat32:
		return float64(n.(float32)), true
	case jsonNumInt:
		return float64(n.(int)), false
	case jsonNumUint:
		return float64(n.(uint)), false
	case jsonNumInt64:
		return float64(n.(int64)), false
	case jsonNumUint64:
		return float64(n.(uint64)), false
	case jsonNumInt32:
		return float64(n.(int32)), false
	case jsonNumUint32:
		return float64(n.(uint32)), false
	case jsonNumInt16:
		return float64(n.(int16)), false
	case jsonNumUint16:
		return float64(n.(uint16)), false
	case jsonNumInt8:
		return float64(n.(int8)), false
	case jsonNumUint8:
		return float64(n.(uint8)), false
	}
	return math.NaN(), true
}
