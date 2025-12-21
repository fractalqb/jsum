package jsum

import (
	"encoding/binary"
	"sort"
)

type Union struct {
	dedBase
	Variants []Deducer
}

func (u *Union) Accepts(v any) bool {
	for _, d := range u.Variants {
		if d.Accepts(v) {
			return true
		}
	}
	return false
}

func (u *Union) Example(v any) Deducer {
	for _, d := range u.Variants {
		if d.Accepts(v) {
			return u
		}
	}
	// TODO When does Union switch to Any
	u.Variants = append(u.Variants, Deduce(u.cfg, v))
	return u
}

func (u *Union) Hash(dh DedupHash) uint64 {
	dhs := make([]uint64, 0, len(u.Variants))
	for _, ed := range u.Variants {
		dhs = append(dhs, ed.Hash(dh))
	}
	sort.Slice(dhs, func(i, j int) bool { return dhs[i] < dhs[j] })
	hash := u.dedBase.startHash(jsonUnion)
	for _, h := range dhs {
		binary.Write(hash, hashEndian, h)
	}
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], u)
	return res
}

func (u *Union) Equal(d Deducer) bool {
	b, ok := d.(*Union)
	if !ok {
		return false
	}
	res := u.dedBase.Equal(&b.dedBase) && len(u.Variants) == len(b.Variants)
	if res {
		for i := range b.Variants {
			if res = u.Variants[i].Equal(b.Variants[i]); !res {
				break
			}
		}
	}
	return res
}

func (u *Union) JSONSchema() any {
	scm := jscmOneOf{OneOf: make([]any, len(u.Variants))}
	for i, v := range u.Variants {
		scm.OneOf[i] = v.JSONSchema()
	}
	return scm
}

func (u *Union) super() *dedBase { return &u.dedBase }
