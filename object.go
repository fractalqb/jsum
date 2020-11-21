package jsum

import (
	"encoding/binary"
	"sort"
)

type Object struct {
	dedBase
	mbrs  map[string]member
	count int
}

type member struct {
	occurence int
	ded       Deducer
}

func newObjJson(cfg *Config, m map[string]interface{}) *Object {
	res := &Object{
		dedBase: dedBase{cfg: cfg},
		mbrs:    make(map[string]member),
		count:   1,
	}
	for k, v := range m {
		res.mbrs[k] = member{occurence: 1, ded: Deduce(cfg, v)}
	}
	return res
}

func (o *Object) Accepts(v interface{}) bool {
	return JsonTypeOf(v) == JsonObject
}

func (o *Object) Example(v interface{}) Deducer {
	vjt := JsonTypeOf(v)
	switch vjt {
	case 0:
		o.null = true
		return o
	case JsonObject:
		o.count++
		switch vo := v.(type) {
		case map[string]interface{}:
			o.mergeMap(vo)
		}
		// TODO more Object types?
		return o
	}
	return newAny(o.cfg, o.null)
}

func (o *Object) mergeMap(m map[string]interface{}) {
	for k, v := range m {
		if m, ok := o.mbrs[k]; ok {
			o.mbrs[k] = member{occurence: m.occurence + 1, ded: m.ded.Example(v)}
		} else {
			o.mbrs[k] = member{occurence: 1, ded: Deduce(o.cfg, v)}
		}
	}
}

func (o *Object) Hash(dh DedupHash) uint64 {
	type memhash struct {
		n string
		h uint64
	}
	mems := make([]memhash, 0, len(o.mbrs))
	for n, m := range o.mbrs {
		mh := m.ded.Hash(dh)
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
	res := o.dedBase.Equal(&b.dedBase) && len(o.mbrs) == len(b.mbrs)
	if res {
		for i := range o.mbrs {
			if res = o.mbrs[i].ded.Equal(b.mbrs[i].ded); !res {
				break
			}
		}
	}
	return res
}

func (o *Object) super() *dedBase { return &o.dedBase }
