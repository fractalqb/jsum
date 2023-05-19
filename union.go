package jsum

import (
	"encoding/binary"
	"sort"
)

type Union struct {
	dedBase
	variants []Deducer
}

func (u *Union) Accepts(v interface{}) bool {
	for _, d := range u.variants {
		if d.Accepts(v) {
			return true
		}
	}
	return false
}

func (u *Union) Example(v interface{}) Deducer {
	for _, d := range u.variants {
		if d.Accepts(v) {
			return u
		}
	}
	// TODO When does Union switch to Any
	u.variants = append(u.variants, Deduce(u.cfg, v))
	return u
}

func (u *Union) Hash(dh DedupHash) uint64 {
	dhs := make([]uint64, 0, len(u.variants))
	for _, ed := range u.variants {
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
	res := u.dedBase.Equal(&b.dedBase) && len(u.variants) == len(b.variants)
	if res {
		for i := range b.variants {
			if res = u.variants[i].Equal(b.variants[i]); !res {
				break
			}
		}
	}
	return res
}

func (u *Union) super() *dedBase { return &u.dedBase }
