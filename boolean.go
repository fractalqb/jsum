package jsum

type Boolean struct {
	dedBase
	tNo, fNo int
}

func (a *Boolean) Accepts(v interface{}) bool { return JsonTypeOf(v) == JsonBoolean }

func (a *Boolean) Example(v interface{}) Deducer {
	vjt := JsonTypeOf(v)
	if vjt == JsonBoolean {
		if v.(bool) {
			a.tNo++
		} else {
			a.fNo++
		}
		return a
	}
	return &Union{
		dedBase:  dedBase{cfg: a.cfg},
		variants: []Deducer{a, Deduce(a.cfg, v)},
	}
}

func (a *Boolean) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(JsonBoolean)
	var nos byte
	if a.fNo > 0 {
		nos = 1
	}
	if a.tNo > 0 {
		nos |= 2
	}
	hash.WriteByte(nos)
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

func (s *Boolean) super() *dedBase { return &s.dedBase }
