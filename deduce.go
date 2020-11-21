package jsum

import (
	"encoding/binary"
	"fmt"
	"hash/maphash"
	"math"
	"reflect"
	"sort"
)

type JsonType int

const (
	JsonObject JsonType = iota + 1
	JsonArray
	JsonString
	JsonNumber
	JsonBool

	jsonUnknown
	jsonEnum
	jsonUnion
	jsonAny
)

var (
	hashEndian = binary.LittleEndian
	hashSeed   = maphash.MakeSeed()
)

func (jt JsonType) Scalar() bool {
	return jt >= JsonString && jt <= JsonBool
}

func JsonTypeOf(v interface{}) JsonType {
	switch v.(type) {
	case nil:
		return 0
	case string:
		return JsonString
	case int, uint, int64, uint64, int32, uint32, int16, uint16, int8, uint8:
		return JsonNumber
	case float32, float64:
		return JsonNumber
	case bool:
		return JsonBool
	case map[string]interface{}:
		return JsonObject
	case []interface{}:
		return JsonArray
	}
	rty := reflect.TypeOf(v)
	switch rty.Kind() {
	case reflect.Struct:
		return JsonObject
	case reflect.Slice:
		return JsonArray
	}
	return 0
}

type DedupHash map[uint64][]Deducer

type Deducer interface {
	Accepts(v interface{}) bool
	Example(v interface{}) Deducer
	Nullable() bool
	Hash(dh DedupHash) uint64
	Copies() []Deducer
	Equal(d Deducer) bool
	super() *dedBase
}

type dedBase struct {
	cfg    *Config
	null   bool
	orig   Deducer
	copies []Deducer
}

func (d *dedBase) Nullable() bool { return d.null }

func (d *dedBase) Copies() []Deducer { return d.copies }

func (d *dedBase) startHash(jt JsonType) *maphash.Hash {
	h := new(maphash.Hash)
	h.SetSeed(hashSeed)
	binary.Write(h, hashEndian, jt)
	if d.null {
		h.WriteByte(0)
	} else {
		h.WriteByte(1)
	}
	return h
}

func (lhs *dedBase) Equal(rhs *dedBase) bool {
	return lhs.null == rhs.null
}

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
	case JsonString, JsonBool:
		if a.cfg.testAsEnum(nil, v) {
			return &Enum{
				dedBase: dedBase{cfg: a.cfg},
				base:    &Scalar{dedBase: dedBase{cfg: a.cfg}, jt: vjt},
				lits:    map[interface{}]int{v: 1},
			}
		}
		return &Scalar{dedBase: dedBase{cfg: a.cfg}, jt: vjt}
	case JsonNumber:
		num := &Number{dedBase: dedBase{cfg: a.cfg, null: a.null}}
		x := num.updateFloat(v)
		num.min, num.max = x, x
		if a.cfg.testAsEnum(nil, v) {
			return &Enum{
				dedBase: dedBase{cfg: a.cfg},
				base:    num,
				lits:    map[interface{}]int{v: 1},
			}
		}
		return num
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
	return Invalid{fmt.Errorf("Cannot deduce JSON from: %T", v)}
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

func Deduce(cfg *Config, v interface{}) Deducer {
	tmp := Unknown{dedBase: dedBase{cfg: cfg}}
	return tmp.Example(v)
}

type Scalar struct {
	dedBase
	jt JsonType
}

func (a *Scalar) Accepts(v interface{}) bool {
	return JsonTypeOf(v) == a.jt
}

func (a *Scalar) Example(v interface{}) Deducer {
	vjt := JsonTypeOf(v)
	switch {
	case a.jt == vjt:
		return a
	case vjt == 0:
		a.null = true
		return a
	}
	return &Union{
		dedBase: dedBase{cfg: a.cfg},
		ds:      []Deducer{a, Deduce(a.cfg, v)},
	}
}

func (a *Scalar) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(a.jt)
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], a)
	return res
}

func (s *Scalar) Equal(d Deducer) bool {
	b, ok := d.(*Scalar)
	if !ok {
		return false
	}
	res := s.dedBase.Equal(&b.dedBase)
	return res && s.jt == b.jt
}

func (s *Scalar) super() *dedBase { return &s.dedBase }

type Array struct {
	dedBase
	minLen, maxLen int
	ed             Deducer
}

func newArrJson(cfg *Config, a []interface{}) *Array {
	res := &Array{
		dedBase: dedBase{
			cfg:  cfg,
			null: a == nil,
		},
		minLen: len(a),
		maxLen: len(a),
		ed:     NewUnknown(cfg),
	}
	for _, e := range a {
		res.ed = res.ed.Example(e)
	}
	return res
}

func (a *Array) Accepts(v interface{}) bool {
	return JsonTypeOf(v) == JsonArray
}

func (a *Array) Example(v interface{}) Deducer {
	vjt := JsonTypeOf(v)
	switch vjt {
	case 0:
		a.null = true
		return a
	case JsonArray:
		switch av := v.(type) {
		case []interface{}:
			if l := len(av); l < a.minLen {
				a.minLen = l
			} else if l > a.maxLen {
				a.maxLen = l
			}
			for _, e := range av {
				a.ed = a.ed.Example(e)
			}
		}
		return a
	}
	return newAny(a.cfg, a.null)
}

func (a *Array) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(JsonArray)
	binary.Write(hash, hashEndian, a.minLen)
	binary.Write(hash, hashEndian, a.maxLen)
	eh := a.ed.Hash(dh)
	binary.Write(hash, hashEndian, eh)
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], a)
	return res
}

func (a *Array) Equal(d Deducer) bool {
	b, ok := d.(*Array)
	if !ok {
		return false
	}
	res := a.dedBase.Equal(&b.dedBase)
	// TODO
	return res
}

func (a *Array) super() *dedBase { return &a.dedBase }

type Object struct {
	dedBase
	mbrs  map[string]member
	count int
}

type member struct {
	n int
	d Deducer
}

func newObjJson(cfg *Config, m map[string]interface{}) *Object {
	res := &Object{
		dedBase: dedBase{cfg: cfg},
		mbrs:    make(map[string]member),
		count:   1,
	}
	for k, v := range m {
		res.mbrs[k] = member{n: 1, d: Deduce(cfg, v)}
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
			o.mbrs[k] = member{n: m.n + 1, d: m.d.Example(v)}
		} else {
			o.mbrs[k] = member{n: 1, d: Deduce(o.cfg, v)}
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
		mh := m.d.Hash(dh)
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
	b, ok := d.(*Enum)
	if !ok {
		return false
	}
	res := o.dedBase.Equal(&b.dedBase)
	// TODO
	return res
}

func (o *Object) super() *dedBase { return &o.dedBase }

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
		ds: []Deducer{e, Deduce(e.cfg, v)},
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
	// TODO
	return res
}

func (e *Enum) super() *dedBase { return &e.dedBase }

type Union struct {
	dedBase
	ds []Deducer
}

func (u *Union) Accepts(v interface{}) bool {
	for _, d := range u.ds {
		if d.Accepts(v) {
			return true
		}
	}
	return false
}

func (u *Union) Example(v interface{}) Deducer {
	vjt := JsonTypeOf(v)
	for _, d := range u.ds {
		if d.Accepts(vjt) {
			return u
		}
	}
	// TODO When does Union switch to Any
	u.ds = append(u.ds, Deduce(u.cfg, v))
	return u
}

func (u *Union) Hash(dh DedupHash) uint64 {
	dhs := make([]uint64, 0, len(u.ds))
	for _, ed := range u.ds {
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
	res := u.dedBase.Equal(&b.dedBase)
	// TODO
	return res
}

func (u *Union) super() *dedBase { return &u.dedBase }

type Any struct{ dedBase }

func newAny(cfg *Config, nullable bool) *Any {
	return &Any{
		dedBase{
			cfg:  cfg,
			null: nullable,
		},
	}
}

func (_ *Any) Accepts(v interface{}) bool { return true }

func (a *Any) Example(v interface{}) Deducer { return a }

func (a *Any) Hash(dh DedupHash) uint64 {
	hash := a.dedBase.startHash(jsonAny)
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], a)
	return res
}

func (a *Any) Equal(d Deducer) bool {
	b, ok := d.(*Any)
	if !ok {
		return false
	}
	return a.dedBase.Equal(&b.dedBase)
}

func (a *Any) super() *dedBase { return &a.dedBase }

type Number struct {
	dedBase
	isFloat  bool
	min, max float64
}

func (nr *Number) Accepts(v interface{}) bool {
	return JsonTypeOf(v) == JsonNumber
}

func (nr *Number) Example(v interface{}) Deducer {
	jvt := JsonTypeOf(v)
	switch jvt {
	case 0:
		nr.null = true
	case JsonNumber:
		x := nr.updateFloat(v)
		if x < nr.min {
			nr.min = x
		} else if x > nr.max {
			nr.max = x
		}
	default:
		return &Union{
			dedBase: dedBase{cfg: nr.cfg},
			ds:      []Deducer{nr, Deduce(nr.cfg, v)},
		}
	}
	return nr
}

func (nr *Number) Hash(dh DedupHash) uint64 {
	hash := nr.dedBase.startHash(JsonNumber)
	if nr.cfg.NumberHash&NumberHashIntFloat != 0 {
		if nr.isFloat {
			hash.WriteByte(1)
		} else {
			hash.WriteByte(0)
		}
	}
	if nr.cfg.NumberHash&NumberHashMin != 0 {
		binary.Write(hash, hashEndian, nr.min)
	}
	if nr.cfg.NumberHash&NumberHashMax != 0 {
		binary.Write(hash, hashEndian, nr.max)
	}
	res := hash.Sum64()
	dh[res] = addNotEqual(dh[res], nr)
	return res
}

func (nr *Number) Equal(d Deducer) bool {
	b, ok := d.(*Number)
	if !ok {
		return false
	}
	res := nr.dedBase.Equal(&b.dedBase)
	if res && nr.cfg.NumberHash&NumberHashIntFloat != 0 {
		res = nr.isFloat == b.isFloat
	}
	if res && nr.cfg.NumberHash&NumberHashMin != 0 {
		res = nr.min == b.min
	}
	if res && nr.cfg.NumberHash&NumberHashMax != 0 {
		res = nr.max == b.max
	}
	return res
}

func (nr *Number) super() *dedBase { return &nr.dedBase }

func (nr *Number) updateFloat(v interface{}) float64 {
	x := asNumber(v)
	switch {
	case math.IsNaN(x):
		nr.isFloat = true
	case math.IsInf(x, 0):
		nr.isFloat = true
	default:
		_, f := math.Modf(x)
		nr.isFloat = nr.isFloat || f != 0
	}
	return x
}

type Invalid struct {
	error
}

func (_ Invalid) Accepts(_ interface{}) bool { return false }

func (i Invalid) Example(v interface{}) Deducer { return i }

func (_ Invalid) Nullable() bool { return false }

func (_ Invalid) setNullable(_ bool) {}

func (_ Invalid) Hash(dh DedupHash) uint64 { return 0 }

func (_ Invalid) Equal(_ Deducer) bool { return false }

func (_ Invalid) Copies() []Deducer { return nil }

var invBase dedBase

func (_ Invalid) super() *dedBase { return &invBase }

func asNumber(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case uint:
		return float64(n)
	case int64:
		return float64(n)
	case uint64:
		return float64(n)
	case int32:
		return float64(n)
	case uint32:
		return float64(n)
	case int16:
		return float64(n)
	case uint16:
		return float64(n)
	case int8:
		return float64(n)
	case uint8:
		return float64(n)
	}
	return math.NaN()
}

func addNotEqual(ds []Deducer, d Deducer) []Deducer {
	for _, e := range ds {
		if e.Equal(d) {
			d.super().orig = e
			e.super().copies = append(e.super().copies, d)
			return ds
		}
	}
	return append(ds, d)
}
