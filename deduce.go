package jsum

import (
	"encoding/binary"
	"hash/maphash"
	"reflect"
	"time"
)

type JsonType int

const (
	JsonObject JsonType = iota + 1
	JsonArray
	JsonString
	JsonNumber
	JsonBoolean

	jsonUnknown
	jsonUnion
	jsonAny
)

var (
	hashEndian = binary.LittleEndian
	hashSeed   = maphash.MakeSeed()
)

func (jt JsonType) Scalar() bool {
	return jt >= JsonString && jt <= JsonBoolean
}

func JsonTypeOf(v any) JsonType {
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
		return JsonBoolean
	case time.Time:
		return JsonString
	case map[string]any:
		return JsonObject
	case []any:
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

func (dh DedupHash) ReusedTypes() (res []Deducer) {
	for _, d := range dh {
		for _, t := range d {
			if len(t.super().copies) > 0 {
				res = append(res, t)
			}
		}
	}
	return res
}

type Deducer interface {
	Accepts(v any) bool
	Example(v any) Deducer
	Nulls() int
	Hash(dh DedupHash) uint64
	Copies() []Deducer
	Equal(d Deducer) bool
	JSONSchema() any
	super() *dedBase
}

type dedBase struct {
	cfg    *Config
	null   int
	orig   Deducer
	copies []Deducer
}

func (d *dedBase) Nulls() int { return d.null }

func (d *dedBase) Copies() []Deducer { return d.copies }

func (d *dedBase) startHash(jt JsonType) *maphash.Hash {
	h := new(maphash.Hash)
	h.SetSeed(hashSeed)
	binary.Write(h, hashEndian, int32(jt))
	if d.null > 0 {
		h.WriteByte(0)
	} else {
		h.WriteByte(1)
	}
	return h
}

func (lhs *dedBase) Equal(rhs *dedBase) bool {
	return lhs.null == rhs.null
}

func Deduce(cfg *Config, v any) Deducer {
	tmp := *NewUnknown(cfg)
	return tmp.Example(v)
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
