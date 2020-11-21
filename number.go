package jsum

import (
	"encoding/binary"
	"math"
)

type Number struct {
	dedBase
	isFloat  bool
	min, max float64
}

func (nr *Number) Accepts(v interface{}) bool {
	return JsonTypeOf(v) == JsonNumber
}

func (nr *Number) Example(v interface{}) Deducer {
	jvt := JsonTypeOf(v)
	switch jvt {
	case 0:
		nr.null = true
	case JsonNumber:
		x := nr.updateFloat(v)
		if x < nr.min {
			nr.min = x
		} else if x > nr.max {
			nr.max = x
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
	if nr.cfg.DupNumber&NumberDupIntFloat != 0 {
		if nr.isFloat {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	if nr.cfg.DupNumber&NumberDupMin != 0 {
		binary.Write(hash, hashEndian, nr.min)
	}
	if nr.cfg.DupNumber&NumberDupMax != 0 {
		binary.Write(hash, hashEndian, nr.max)
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
	if res && nr.cfg.DupNumber&NumberDupIntFloat != 0 {
		res = nr.isFloat == b.isFloat
	}
	if res && nr.cfg.DupNumber&NumberDupMin != 0 {
		res = nr.min == b.min
	}
	if res && nr.cfg.DupNumber&NumberDupMax != 0 {
		res = nr.max == b.max
	}
	return res
}

func (nr *Number) super() *dedBase { return &nr.dedBase }

func (nr *Number) updateFloat(v interface{}) float64 {
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

func asNumber(v interface{}) float64 {
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
