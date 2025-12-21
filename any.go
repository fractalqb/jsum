package jsum

type Any struct{ dedBase }

func newAny(cfg *Config, nulln int) *Any {
	return &Any{
		dedBase{
			cfg:  cfg,
			null: nulln,
		},
	}
}

func (*Any) Accepts(v any) bool { return true }

func (a *Any) Example(v any) Deducer {
	if v == nil {
		a.null++
	}
	return a
}

func (a *Any) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(jsonAny)
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
