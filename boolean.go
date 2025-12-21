package jsum

type Boolean struct {
	dedBase
	tNo, fNo int
}

func (a *Boolean) Accepts(v any) bool { return JsonTypeOf(v) == JsonBoolean }

func (a *Boolean) Example(v any) Deducer {
	vjt := JsonTypeOf(v)
	switch vjt {
	case 0:
		a.null++
		return a
	case JsonBoolean:
		if v.(bool) {
			a.tNo++
		} else {
			a.fNo++
		}
		return a
	}
	return &Union{
		dedBase:  dedBase{cfg: a.cfg},
		Variants: []Deducer{a, Deduce(a.cfg, v)},
	}
}

func (a *Boolean) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(JsonBoolean)
	if a.cfg.DedupBool&DedupBoolFalse != 0 {
		if a.fNo > 0 {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	if a.cfg.DedupBool&DedupBoolTrue != 0 {
		if a.tNo > 0 {
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
	if (s.fNo > 0) != (b.fNo > 0) {
		return false
	}
	if (s.tNo > 0) != (b.tNo > 0) {
		return false
	}
	res := s.dedBase.Equal(&b.dedBase)
	return res
}

func (a *Boolean) JSONSchema() any {
	return jscmString{jscmType: jscmType{Type: "boolean"}}
}

func (s *Boolean) super() *dedBase { return &s.dedBase }
