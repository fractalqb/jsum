package jsum

import (
	"encoding/binary"
	"hash/maphash"
	"reflect"
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

func Deduce(cfg *Config, v interface{}) Deducer {
	tmp := Unknown{dedBase: dedBase{cfg: cfg}}
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
