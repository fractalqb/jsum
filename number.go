package jsum

import (
	"encoding/binary"
	"math"
)

type Number struct {
	dedBase
	min, max float64
	isFloat  bool
	hadFrac  bool
}

func (nr *Number) Accepts(v any) bool {
	return JsonTypeOf(v) == JsonNumber
}

func (nr *Number) Example(v any) Deducer {
	jvt := JsonTypeOf(v)
	switch jvt {
	case 0:
		nr.null++
	case JsonNumber:
		x := nr.updateFloat(v)
		if x < nr.min {
			nr.min = x
		} else if x > nr.max {
			nr.max = x
		}
		if !nr.hadFrac {
			_, exp := math.Frexp(x)
			nr.hadFrac = exp != 0
		}
	default:
		return &Union{
			dedBase:  dedBase{cfg: nr.cfg},
			variants: []Deducer{nr, Deduce(nr.cfg, v)},
		}
	}
	return nr
}

func (nr *Number) Hash(dh DedupHash) uint64 {
	hash := nr.dedBase.startHash(JsonNumber)
	if nr.cfg.DedupNumber&DedpuNumberIntFloat != 0 {
		if nr.isFloat {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	if nr.cfg.DedupNumber&DedupNumberFrac != 0 {
		if nr.hadFrac {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	if nr.cfg.DedupNumber&DedupNumberMin != 0 {
		binary.Write(hash, hashEndian, nr.min)
	}
	if nr.cfg.DedupNumber&DedupNumberMax != 0 {
		binary.Write(hash, hashEndian, nr.max)
	}
	if nr.cfg.DedupNumber&DedupNumberNeg != 0 {
		if nr.min < 0 {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	if nr.cfg.DedupNumber&DedupNumberPos != 0 {
		if nr.max > 0 {
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
		res = nr.isFloat == b.isFloat
	}
	if res && nr.cfg.DedupNumber&DedupNumberFrac != 0 {
		res = nr.hadFrac == b.hadFrac
	}
	if res && nr.cfg.DedupNumber&DedupNumberMin != 0 {
		res = nr.min == b.min
	}
	if res && nr.cfg.DedupNumber&DedupNumberMax != 0 {
		res = nr.max == b.max
	}
	if res && nr.cfg.DedupNumber&DedupNumberNeg != 0 {
		res = (nr.min < 0) == (b.min < 0)
	}
	if res && nr.cfg.DedupNumber&DedupNumberPos != 0 {
		res = (nr.max > 0) == (b.max > 0)
	}
	return res
}

func (nr *Number) super() *dedBase { return &nr.dedBase }

func (nr *Number) updateFloat(v any) float64 {
	x := asNumber(v)
	switch {
	case math.IsNaN(x):
		nr.isFloat = true
	case math.IsInf(x, 0):
		nr.isFloat = true
	default:
		_, f := math.Modf(x)
		nr.isFloat = nr.isFloat || f != 0
	}
	return x
}

func asNumber(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case uint:
		return float64(n)
	case int64:
		return float64(n)
	case uint64:
		return float64(n)
	case int32:
		return float64(n)
	case uint32:
		return float64(n)
	case int16:
		return float64(n)
	case uint16:
		return float64(n)
	case int8:
		return float64(n)
	case uint8:
		return float64(n)
	}
	return math.NaN()
}
