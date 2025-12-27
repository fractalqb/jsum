/*
A tool to analyse the structure of JSON from a set of example JSON values.
Copyright (C) 2025  Marcus Perlick

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package jsum

import (
	"encoding/binary"
	"hash/maphash"
	"reflect"
	"time"
)

type (
	JsonType    uint16
	jsonVariant uint16
)
type JsumType struct {
	t JsonType
	v jsonVariant
}

func (t JsumType) Valid() bool        { return t.t > 0 }
func (t JsumType) JsonType() JsonType { return t.t }
func (t JsumType) Scalar() bool       { return t.t.scalar() }

const (
	JsonNull JsonType = iota + 1
	JsonObject
	JsonArray
	JsonString
	JsonNumber
	JsonBoolean

	JsonUnknown
	JsonUnion
	JsonAny

	jsonInvalid
)

func (jt JsonType) scalar() bool {
	return jt >= JsonString && jt <= JsonBoolean
}

type TypeSet uint32

const AllTypes = ^TypeSet(1 << (jsonInvalid - 1))

func NewTypeSet(ts ...JsonType) (set TypeSet) {
	for _, t := range ts {
		set.Add(t)
	}
	return set
}

func (ts *TypeSet) Add(t JsonType) {
	if t > 0 && t < jsonInvalid {
		*ts |= 1 << (t - 1)
	}
}

const (
	jsonStrString jsonVariant = iota
	jsonStrTime
	jsonNumInt
	jsonNumUint
	jsonNumInt64
	jsonNumUint64
	jsonNumInt32
	jsonNumUint32
	jsonNumInt16
	jsonNumUint16
	jsonNumInt8
	jsonNumUint8
	jsonNumFloat32
	jsonNumFloat64
	jsonObjRMap
	jsonObjStrAny
	jsonArrAny
	jsonArrRSlice
)

// JsonTypeOf detects: nil, string, number, bool, object, array
func JsonTypeOf(v any) JsumType {
	switch v.(type) {
	case nil:
		return JsumType{t: JsonNull}
	case string:
		return JsumType{t: JsonString, v: jsonStrString}
	case int:
		return JsumType{t: JsonNumber, v: jsonNumInt}
	case uint:
		return JsumType{t: JsonNumber, v: jsonNumUint}
	case int64:
		return JsumType{t: JsonNumber, v: jsonNumInt64}
	case uint64:
		return JsumType{t: JsonNumber, v: jsonNumUint64}
	case int32:
		return JsumType{t: JsonNumber, v: jsonNumInt32}
	case uint32:
		return JsumType{t: JsonNumber, v: jsonNumUint32}
	case int16:
		return JsumType{t: JsonNumber, v: jsonNumInt16}
	case uint16:
		return JsumType{t: JsonNumber, v: jsonNumUint16}
	case int8:
		return JsumType{t: JsonNumber, v: jsonNumInt8}
	case uint8:
		return JsumType{t: JsonNumber, v: jsonNumUint8}
	case float32:
		return JsumType{t: JsonNumber, v: jsonNumFloat32}
	case float64:
		return JsumType{t: JsonNumber, v: jsonNumFloat64}
	case bool:
		return JsumType{t: JsonBoolean}
	case time.Time:
		return JsumType{t: JsonString, v: jsonStrTime}
	case map[string]any:
		return JsumType{t: JsonObject, v: jsonObjStrAny}
	case []any:
		return JsumType{t: JsonArray, v: jsonArrAny}
	}
	rty := reflect.TypeOf(v)
	switch rty.Kind() {
	case reflect.Map:
		return JsumType{t: JsonObject, v: jsonObjRMap}
	case reflect.Slice:
		return JsumType{t: JsonArray, v: jsonArrRSlice}
	}
	return JsumType{}
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

const UnknownAccept = -1

type Deducer interface {
	JsonType() JsonType
	Accepts(v any, jt JsumType) float64
	Example(v any, jt JsumType, acpt float64) Deducer
	Nulls() int
	Hash(dh DedupHash) uint64
	Copies() []Deducer
	Equal(d Deducer) bool
	JSONSchema() any
	super() *dedBase
}

type dedBase struct {
	cfg    *Config
	Count  int `json:"count"`
	Null   int `json:"null,omitempty"`
	orig   Deducer
	copies []Deducer
}

func (d *dedBase) Nulls() int { return d.Null }

func (d *dedBase) Copies() []Deducer { return d.copies }

func (d *dedBase) startHash(jt JsonType) *maphash.Hash {
	h := new(maphash.Hash)
	h.SetSeed(hashSeed)
	binary.Write(h, hashEndian, int32(jt))
	if d.Null > 0 {
		h.WriteByte(0)
	} else {
		h.WriteByte(1)
	}
	return h
}

func (lhs *dedBase) Equal(rhs *dedBase) bool {
	return lhs.Null == rhs.Null
}

func Deduce(cfg *Config, v any) Deducer {
	tmp := *NewUnknown(cfg)
	return tmp.Example(v, JsonTypeOf(v), UnknownAccept)
}

var (
	hashEndian = binary.LittleEndian
	hashSeed   = maphash.MakeSeed()
)

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
