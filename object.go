package jsum

import (
	"encoding/binary"
	"sort"
)

type Object struct {
	dedBase
	Members map[string]Member
	Count   int
}

type Member struct {
	Occurence int
	Ded       Deducer
}

func newObjJson(cfg *Config, m map[string]any) *Object {
	res := &Object{
		dedBase: dedBase{cfg: cfg},
		Members: make(map[string]Member),
		Count:   1,
	}
	for k, v := range m {
		res.Members[k] = Member{Occurence: 1, Ded: Deduce(cfg, v)}
	}
	return res
}

func (o *Object) Accepts(v any) bool {
	return JsonTypeOf(v) == JsonObject
}

func (o *Object) Example(v any) Deducer {
	vjt := JsonTypeOf(v)
	switch vjt {
	case 0:
		o.null++
		return o
	case JsonObject:
		o.Count++
		switch vo := v.(type) {
		case map[string]any:
			o.mergeMap(vo)
		}
		// TODO more Object types?
		return o
	}
	return newAny(o.cfg, o.null)
}

func (o *Object) mergeMap(m map[string]any) {
	for k, v := range m {
		if m, ok := o.Members[k]; ok {
			o.Members[k] = Member{Occurence: m.Occurence + 1, Ded: m.Ded.Example(v)}
		} else {
			o.Members[k] = Member{Occurence: 1, Ded: Deduce(o.cfg, v)}
		}
	}
}

func (o *Object) Hash(dh DedupHash) uint64 {
	type memhash struct {
		n string
		h uint64
	}
	mems := make([]memhash, 0, len(o.Members))
	for n, m := range o.Members {
		mh := m.Ded.Hash(dh)
		mems = append(mems, memhash{n, mh})
	}
	sort.Slice(mems, func(i, j int) bool { return mems[i].n < mems[j].n })
	hash := o.dedBase.startHash(JsonObject)
	for _, m := range mems {
		binary.Write(hash, hashEndian, m.h)
	}
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], o)
	return res
}

func (o *Object) Equal(d Deducer) bool {
	b, ok := d.(*Object)
	if !ok {
		return false
	}
	res := o.dedBase.Equal(&b.dedBase) && len(o.Members) == len(b.Members)
	if res {
		for i := range o.Members {
			if res = o.Members[i].Ded.Equal(b.Members[i].Ded); !res {
				break
			}
		}
	}
	return res
}

func (o *Object) super() *dedBase { return &o.dedBase }
