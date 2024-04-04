package jsum

import "time"

type Format int

const (
	DateTimeFormat = 1 + iota
)

type String struct {
	dedBase
	stats  map[string]int
	format Format
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
		switch v := v.(type) {
		case string:
			if fmt := stringFormat(v); fmt == 0 {
				a.format = 0
			} else if len(a.stats) == 0 {
				a.format = fmt
			} else if fmt != a.format {
				a.format = 0
			}
			a.stats[v]++
		case time.Time:
			if len(a.stats) == 0 {
				a.format = DateTimeFormat
			}
			s := v.Format(time.RFC3339)
			a.stats[s]++
		}
		return a
	}
	return &Union{
		dedBase:  dedBase{cfg: a.cfg},
		variants: []Deducer{a, Deduce(a.cfg, v)},
	}
}

func stringFormat(s string) Format {
	if _, err := time.Parse(time.RFC3339, s); err == nil {
		return DateTimeFormat
	}
	return 0
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
