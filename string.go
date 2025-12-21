package jsum

import (
	"math"
	"time"
	"unicode/utf8"
)

type Format int

const (
	DateTimeFormat = 1 + iota
)

type String struct {
	dedBase
	Stats  map[string]int
	format Format
}

func NewString(cfg *Config, nulln int) *String {
	return &String{
		dedBase: dedBase{cfg: cfg, null: nulln},
		Stats:   make(map[string]int),
	}
}

func (a *String) Accepts(v any) bool { return JsonTypeOf(v) == JsonString }

func (a *String) Example(v any) Deducer {
	vjt := JsonTypeOf(v)
	if vjt == JsonString {
		switch v := v.(type) {
		case string:
			if fmt := stringFormat(v); fmt == 0 {
				a.format = 0
			} else if len(a.Stats) == 0 {
				a.format = fmt
			} else if fmt != a.format {
				a.format = 0
			}
			a.Stats[v]++
		case time.Time:
			if len(a.Stats) == 0 {
				a.format = DateTimeFormat
			}
			s := v.Format(time.RFC3339)
			a.Stats[s]++
		}
		return a
	}
	return &Union{
		dedBase:  dedBase{cfg: a.cfg},
		Variants: []Deducer{a, Deduce(a.cfg, v)},
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
		if s.Stats[""] > 0 {
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
	if se, de := s.Stats[""], b.Stats[""]; (se > 0) != (de > 0) {
		return false
	}
	return true
}

func (a *String) JSONSchema() any {
	scm := jscmString{
		jscmType: jscmType{Type: "string"},
	}
	switch a.format {
	case 0:
		mi, ma := math.MaxInt, 0
		for s := range a.Stats {
			n := utf8.RuneCountInString(s)
			mi = min(mi, n)
			ma = max(ma, n)
		}
		scm.MinLen = new(int)
		*scm.MinLen = mi
		scm.MaxLen = new(int)
		*scm.MaxLen = ma
	case DateTimeFormat:
		scm.Format = "date-time"
	}
	return scm
}

func (s *String) super() *dedBase { return &s.dedBase }
