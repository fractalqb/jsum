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

func (s *String) Hash(dh DedupHash) uint64 {
	hash := s.dedBase.startHash(JsonString)
	if s.cfg.DedupString&DedupStringEmpty != 0 {
		if s.stats[""] > 0 {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], s)
	return res
}

func (s *String) Equal(d Deducer) bool {
	b, ok := d.(*String)
	if !ok {
		return false
	}
	if !s.dedBase.Equal(&b.dedBase) {
		return false
	}
	if se, de := s.stats[""], b.stats[""]; (se > 0) != (de > 0) {
		return false
	}
	return true
}

func (s *String) super() *dedBase { return &s.dedBase }
