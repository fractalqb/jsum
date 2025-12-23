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
	jsonType    uint16
	jsonVariant uint16
)
type JsonType struct {
	t jsonType
	v jsonVariant
}

func (t JsonType) Valid() bool  { return t.t > 0 }
func (t JsonType) Scalar() bool { return t.t.scalar() }

const (
	jsonNull jsonType = iota + 1
	jsonObject
	jsonArray
	jsonString
	jsonNumber
	jsonBoolean

	jsonUnknown
	jsonUnion
	jsonAny
)

func (jt jsonType) scalar() bool {
	return jt >= jsonString && jt <= jsonBoolean
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
func JsonTypeOf(v any) JsonType {
	switch v.(type) {
	case nil:
		return JsonType{t: jsonNull}
	case string:
		return JsonType{t: jsonString, v: jsonStrString}
	case int:
		return JsonType{t: jsonNumber, v: jsonNumInt}
	case uint:
		return JsonType{t: jsonNumber, v: jsonNumUint}
	case int64:
		return JsonType{t: jsonNumber, v: jsonNumInt64}
	case uint64:
		return JsonType{t: jsonNumber, v: jsonNumUint64}
	case int32:
		return JsonType{t: jsonNumber, v: jsonNumInt32}
	case uint32:
		return JsonType{t: jsonNumber, v: jsonNumUint32}
	case int16:
		return JsonType{t: jsonNumber, v: jsonNumInt16}
	case uint16:
		return JsonType{t: jsonNumber, v: jsonNumUint16}
	case int8:
		return JsonType{t: jsonNumber, v: jsonNumInt8}
	case uint8:
		return JsonType{t: jsonNumber, v: jsonNumUint8}
	case float32:
		return JsonType{t: jsonNumber, v: jsonNumFloat32}
	case float64:
		return JsonType{t: jsonNumber, v: jsonNumFloat64}
	case bool:
		return JsonType{t: jsonBoolean}
	case time.Time:
		return JsonType{t: jsonString, v: jsonStrTime}
	case map[string]any:
		return JsonType{t: jsonObject, v: jsonObjStrAny}
	case []any:
		return JsonType{t: jsonArray, v: jsonArrAny}
	}
	rty := reflect.TypeOf(v)
	switch rty.Kind() {
	case reflect.Map:
		return JsonType{t: jsonObject, v: jsonObjRMap}
	case reflect.Slice:
		return JsonType{t: jsonArray, v: jsonArrRSlice}
	}
	return JsonType{}
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
	Accepts(jt JsonType) bool
	Example(v any, jt JsonType) Deducer
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

func (d *dedBase) startHash(jt jsonType) *maphash.Hash {
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
	return tmp.Example(v, JsonTypeOf(v))
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
