package jsum

type Scalar struct {
	dedBase
	jt JsonType
}

func (a *Scalar) Accepts(v interface{}) bool {
	return JsonTypeOf(v) == a.jt
}

func (a *Scalar) Example(v interface{}) Deducer {
	vjt := JsonTypeOf(v)
	switch {
	case a.jt == vjt:
		return a
	case vjt == 0:
		a.null = true
		return a
	}
	return &Union{
		dedBase:  dedBase{cfg: a.cfg},
		variants: []Deducer{a, Deduce(a.cfg, v)},
	}
}

func (a *Scalar) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(a.jt)
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], a)
	return res
}

func (s *Scalar) Equal(d Deducer) bool {
	b, ok := d.(*Scalar)
	if !ok {
		return false
	}
	res := s.dedBase.Equal(&b.dedBase)
	return res && s.jt == b.jt
}

func (s *Scalar) super() *dedBase { return &s.dedBase }
