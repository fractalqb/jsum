package jsum

type String struct {
	dedBase
	stats map[string]int
}

func NewString(cfg *Config, null bool) *String {
	return &String{
		dedBase: dedBase{cfg: cfg, null: null},
		stats:   make(map[string]int),
	}
}

func (a *String) Accepts(v interface{}) bool { return JsonTypeOf(v) == JsonString }

func (a *String) Example(v interface{}) Deducer {
	vjt := JsonTypeOf(v)
	if vjt == JsonString {
		a.stats[v.(string)]++
		return a
	}
	return &Union{
		dedBase:  dedBase{cfg: a.cfg},
		variants: []Deducer{a, Deduce(a.cfg, v)},
	}
}

func (a *String) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(JsonString)
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], a)
	return res
}

func (s *String) Equal(d Deducer) bool {
	b, ok := d.(*String)
	if !ok {
		return false
	}
	res := s.dedBase.Equal(&b.dedBase)
	// TODO consider samples for equality (-> .Hash())
	return res
}

func (s *String) super() *dedBase { return &s.dedBase }
