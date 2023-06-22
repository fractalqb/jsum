package jsum

import "fmt"

type Unknown struct{ dedBase }

func NewUnknown(cfg *Config) *Unknown {
	return &Unknown{dedBase: dedBase{cfg: cfg}}
}

func (a *Unknown) Accepts(v interface{}) bool { return true }

func (a *Unknown) Example(v interface{}) Deducer {
	vjt := JsonTypeOf(v)
	switch vjt {
	case 0:
		a.null = true
		return a
	case JsonString:
		str := NewString(a.cfg, a.null)
		return str.Example(v)
	case JsonNumber:
		num := &Number{dedBase: dedBase{cfg: a.cfg, null: a.null}}
		x := num.updateFloat(v)
		num.min, num.max = x, x
		return num
	case JsonBoolean:
		return &Boolean{dedBase: dedBase{cfg: a.cfg}}
	case JsonObject:
		switch o := v.(type) {
		case map[string]interface{}:
			return newObjJson(a.cfg, o)
		}
	case JsonArray:
		switch av := v.(type) {
		case []interface{}:
			return newArrJson(a.cfg, av)
		}
	}
	return Invalid{fmt.Errorf("cannot deduce JSON from: %T", v)}
}

func (a *Unknown) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(jsonUnknown)
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], a)
	return res
}

func (a *Unknown) Equal(d Deducer) bool {
	b, ok := d.(*Unknown)
	if !ok {
		return false
	}
	return a.dedBase.Equal(&b.dedBase)
}

func (a *Unknown) super() *dedBase { return &a.dedBase }
