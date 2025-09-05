package jsum

import "encoding/binary"

type Array struct {
	dedBase
	minLen, maxLen int
	elem           Deducer
}

func newArrJson(cfg *Config, a []any) *Array {
	res := &Array{
		dedBase: dedBase{
			cfg:  cfg,
			null: a == nil,
		},
		minLen: len(a),
		maxLen: len(a),
		elem:   NewUnknown(cfg),
	}
	for _, e := range a {
		res.elem = res.elem.Example(e)
	}
	return res
}

func (a *Array) Accepts(v any) bool {
	return JsonTypeOf(v) == JsonArray
}

func (a *Array) Example(v any) Deducer {
	vjt := JsonTypeOf(v)
	switch vjt {
	case 0:
		a.null = true
		return a
	case JsonArray:
		switch av := v.(type) {
		case []any:
			if l := len(av); l < a.minLen {
				a.minLen = l
			} else if l > a.maxLen {
				a.maxLen = l
			}
			for _, e := range av {
				a.elem = a.elem.Example(e)
			}
		}
		return a
	}
	return newAny(a.cfg, a.null)
}

func (a *Array) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(JsonArray)
	if a.maxLen == 0 {
		hash.WriteByte(0)
	} else {
		hash.WriteByte(1)
	}
	eh := a.elem.Hash(dh)
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
	if (a.minLen == 0) != (b.minLen == 0) {
		return false
	}
	return a.elem.Equal(b.elem)
}

func (a *Array) super() *dedBase { return &a.dedBase }
