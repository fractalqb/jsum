package jsum

import "encoding/binary"

type Enum struct {
	dedBase
	base Deducer
	lits map[interface{}]int
}

func (e *Enum) Accepts(v interface{}) bool {
	return e.base.Accepts(v)
}

func (e *Enum) Example(v interface{}) Deducer {
	if e.base.Accepts(v) {
		if !e.cfg.testAsEnum(e, v) {
			if e.Nullable() {
				e.base.super().null = true
			}
			return e.base
		}
		if _, ok := e.lits[v]; ok {
			e.lits[v]++
		} else {
			e.lits[v] = 1
		}
		e.base.Example(v)
		return e
	}
	res := &Union{
		dedBase: dedBase{
			cfg:  e.cfg,
			null: e.null,
		},
		variants: []Deducer{e, Deduce(e.cfg, v)},
	}
	e.null = false
	return res
	//return newAny(e.cfg)
}

func (e *Enum) Hash(dh DedupHash) uint64 {
	hash := e.dedBase.startHash(jsonEnum)
	eh := e.base.Hash(dh)
	binary.Write(hash, hashEndian, eh)
	// TODO take enum literals into account
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], e)
	return res
}

func (e *Enum) Equal(d Deducer) bool {
	b, ok := d.(*Enum)
	if !ok {
		return false
	}
	res := e.dedBase.Equal(&b.dedBase)
	// TODO take enum literals into account
	return res
}

func (e *Enum) super() *dedBase { return &e.dedBase }
