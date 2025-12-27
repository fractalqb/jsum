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
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"strings"

	"git.fractalqb.de/fractalqb/eloc"
	"git.fractalqb.de/fractalqb/eloc/must"
)

const StateVersion = 0

const (
	tidInvalid byte = iota
	tidUnknown
	tidBool
	tidNumber
	tidString
	tidObject
	tidArray
	tidUnion
	tidAny
)

type StateIO struct {
	buf  []byte
	strs map[string]int64
	sids map[int64]string
	wr   io.Writer
	rd   restCountReader
	cfg  *Config

	StrCount, StrDup int
}

func (sio *StateIO) WriteState(w io.Writer, ded Deducer) (err error) {
	must.RecoverAs(&err, "write jsum state")
	must.RetCtx(fmt.Fprintf(w, "JSUM%d\n", StateVersion)).Msg("header")
	if sio.strs == nil {
		sio.strs = make(map[string]int64)
	} else {
		clear(sio.strs)
	}
	sio.StrCount, sio.StrDup = 0, 0
	sio.wr = w
	defer func() { sio.wr = nil }()
	sio.wrDed(ded)
	return nil
}

func (sio *StateIO) ReadState(r io.Reader, cfg *Config, size int64) (_ Deducer, err error) {
	must.RecoverAs(&err, "read jsum state")
	sio.rd = restCountReader{bufio.NewReader(r), size}
	sio.cfg = cfg
	defer func() {
		sio.rd.r = nil
		sio.cfg = nil
	}()
	sio.rdHeader()
	if sio.sids == nil {
		sio.sids = make(map[int64]string)
	} else {
		clear(sio.sids)
	}
	sio.StrCount, sio.StrDup = 0, 0
	return sio.rdDed(), nil
}

func (sio *StateIO) wrDed(ded Deducer) {
	switch ded := ded.(type) {
	case *String:
		sio.wrDedStr(ded)
	case *Number:
		sio.wrDedNum(ded)
	case *Boolean:
		sio.wrDedBool(ded)
	case *Object:
		sio.wrDedObj(ded)
	case *Array:
		sio.wrDedArray(ded)
	case *Union:
		sio.wrDedUnion(ded)
	case *Any:
		sio.wrDedAny(ded)
	case *Unknown:
		sio.wrDedUnk(ded)
	case Invalid:
		eloc.Errorf("invalid deducer: %w", ded.error)
	default:
		panic(eloc.Errorf("unsupported deducer: %T", ded))
	}
}

func (sio *StateIO) rdDed() Deducer {
	tid := must.RetCtx(sio.rd.ReadByte()).Msg("deducer type id")
	switch tid {
	case tidString:
		return sio.rdDedStr()
	case tidNumber:
		return sio.rdDedNum()
	case tidBool:
		return sio.rdDedBool()
	case tidObject:
		return sio.rdDedObj()
	case tidArray:
		return sio.rdDedArray()
	case tidUnion:
		return sio.rdDedUnion()
	case tidAny:
		return sio.rdDedAny()
	case tidUnknown:
		return sio.rdDedUnk()
	}
	return newInvalid(eloc.Errorf("illegal deducer type id: %d", tid))
}

func (sio *StateIO) wrBase(tid byte, ded *dedBase) {
	sio.buf = append(sio.buf[:0], tid)
	sio.buf = binary.AppendUvarint(sio.buf, uint64(ded.Count))
	sio.buf = binary.AppendUvarint(sio.buf, uint64(ded.Null))
}

func (sio *StateIO) rdBase(ded *dedBase) {
	u := must.RetCtx(binary.ReadUvarint(&sio.rd)).Msg("base deducer count")
	ded.Count = int(u)
	u = must.RetCtx(binary.ReadUvarint(&sio.rd)).Msg("base deducer null")
	ded.Null = int(u)
}

func (sio *StateIO) wrDedStr(ded *String) {
	sio.wrBase(tidString, &ded.dedBase)
	sio.buf = binary.AppendUvarint(sio.buf, uint64(len(ded.Stats)))
	must.RetCtx(sio.wr.Write(sio.buf)).Msg("string stats len")
	for s, n := range ded.Stats {
		sio.wrString(s)
		sio.buf = binary.AppendUvarint(sio.buf[:0], uint64(n))
		must.RetCtx(sio.wr.Write(sio.buf)).Msg("string stats for %s", s)
	}
	sio.buf = binary.AppendVarint(sio.buf[:0], int64(ded.Format))
	must.RetCtx(sio.wr.Write(sio.buf)).Msg("string format")
}

func (sio *StateIO) rdDedStr() *String {
	ded := &String{dedBase: dedBase{cfg: sio.cfg}}
	sio.rdBase(&ded.dedBase)
	nstats := must.RetCtx(binary.ReadUvarint(&sio.rd)).Msg("string stats len")
	sio.rd.checkU(statMinStrLen*nstats, "string stats len") // TODO factor N *varNo?
	ded.Stats = make(map[string]int, nstats)
	for i := range nstats {
		s := sio.rdString()
		n := must.RetCtx(binary.ReadUvarint(&sio.rd)).Msg("string stat %d", i)
		ded.Stats[s] = int(n)
	}
	form := must.RetCtx(binary.ReadVarint(&sio.rd)).Msg("string format")
	ded.Format = Format(form)
	return ded
}

func (sio *StateIO) wrDedNum(ded *Number) {
	sio.wrBase(tidNumber, &ded.dedBase)
	sio.buf = must.RetCtx(binary.Append(sio.buf, ndn, ded.Min)).
		Msg("number deducer min")
	sio.buf = must.RetCtx(binary.Append(sio.buf, ndn, ded.Max)).
		Msg("number deducer max")
	var flags byte
	if ded.IsFloat {
		flags |= 1
	}
	if ded.HasFrac {
		flags |= 2
	}
	sio.buf = append(sio.buf, flags)
	must.RetCtx(sio.wr.Write(sio.buf)).Msg("number deducer")
}

func (sio *StateIO) rdDedNum() *Number {
	ded := &Number{dedBase: dedBase{cfg: sio.cfg}}
	sio.rdBase(&ded.dedBase)
	must.DoCtx(binary.Read(&sio.rd, ndn, &ded.Min), "number deducer min")
	must.DoCtx(binary.Read(&sio.rd, ndn, &ded.Max), "number deducer max")
	flags := must.RetCtx(sio.rd.ReadByte()).Msg("number deducer flags")
	ded.IsFloat = flags&1 != 0
	ded.HasFrac = flags&2 != 0
	return ded
}

func (sio *StateIO) wrDedBool(ded *Boolean) {
	sio.wrBase(tidBool, &ded.dedBase)
	sio.buf = binary.AppendUvarint(sio.buf, uint64(ded.TrueNo))
	sio.buf = binary.AppendUvarint(sio.buf, uint64(ded.FalseNo))
	must.RetCtx(sio.wr.Write(sio.buf)).Msg("bool deducer")
}

func (sio *StateIO) rdDedBool() *Boolean {
	ded := &Boolean{dedBase: dedBase{cfg: sio.cfg}}
	sio.rdBase(&ded.dedBase)
	tmp := must.RetCtx(binary.ReadUvarint(&sio.rd)).Msg("bool true count")
	ded.TrueNo = int(tmp)
	tmp = must.RetCtx(binary.ReadUvarint(&sio.rd)).Msg("bool false count")
	ded.FalseNo = int(tmp)
	return ded
}

func (sio *StateIO) wrDedObj(ded *Object) {
	sio.wrBase(tidObject, &ded.dedBase)
	sio.buf = binary.AppendUvarint(sio.buf, uint64(len(ded.Members)))
	must.RetCtx(sio.wr.Write(sio.buf)).Msg("object member count")
	for n, m := range ded.Members {
		sio.wrMbr(n, m)
	}
}

func (sio *StateIO) wrMbr(n string, m Member) {
	sio.wrString(n)
	sio.buf = binary.AppendUvarint(sio.buf[:0], uint64(m.Occurence))
	must.RetCtx(sio.wr.Write(sio.buf)).Msg("object member accurence")
	sio.wrDed(m.Ded)
}

func (sio *StateIO) rdDedObj() *Object {
	ded := &Object{dedBase: dedBase{cfg: sio.cfg}}
	sio.rdBase(&ded.dedBase)
	mno := must.RetCtx(binary.ReadUvarint(&sio.rd)).
		Msg("object member count")
	sio.rd.checkU(statMinMbrSz*mno, "object member count") // TODO factor N *varNo?
	ded.Members = make(map[string]Member, mno)
	for range mno {
		n := sio.rdString()
		occ := must.RetCtx(binary.ReadUvarint(&sio.rd)).
			Msg("object member occurence")
		mded := sio.rdDed()
		ded.Members[n] = Member{Occurence: int(occ), Ded: mded}
	}
	return ded
}

func (sio *StateIO) wrDedArray(ded *Array) {
	sio.wrBase(tidArray, &ded.dedBase)
	sio.buf = binary.AppendVarint(sio.buf, int64(ded.MinLen))
	sio.buf = binary.AppendVarint(sio.buf, int64(ded.MaxLen))
	must.RetCtx(sio.wr.Write(sio.buf)).Msg("array min and max len")
	sio.wrDed(ded.Elem)
}

func (sio *StateIO) rdDedArray() *Array {
	ded := &Array{dedBase: dedBase{cfg: sio.cfg}}
	sio.rdBase(&ded.dedBase)
	tmp := must.RetCtx(binary.ReadVarint(&sio.rd)).Msg("read array min len")
	ded.MinLen = int(tmp)
	tmp = must.RetCtx(binary.ReadVarint(&sio.rd)).Msg("read array max len")
	ded.MaxLen = int(tmp)
	ded.Elem = sio.rdDed()
	return ded
}

func (sio *StateIO) wrDedUnion(ded *Union) {
	sio.wrBase(tidUnion, &ded.dedBase)
	sio.buf = binary.AppendUvarint(sio.buf, uint64(len(ded.Variants)))
	must.RetCtx(sio.wr.Write(sio.buf)).Msg("union variants len")
	for _, v := range ded.Variants {
		sio.wrDed(v)
	}
}

func (sio *StateIO) rdDedUnion() *Union {
	ded := &Union{dedBase: dedBase{cfg: sio.cfg}}
	sio.rdBase(&ded.dedBase)
	varNo := must.RetCtx(binary.ReadUvarint(&sio.rd)).Msg("union variant count")
	sio.rd.checkU(statMinVarSz*varNo, "union variant count") // TODO factor N *varNo?
	ded.Variants = make([]Deducer, varNo)
	for i := range varNo {
		ded.Variants[i] = sio.rdDed()
	}
	return ded
}

func (sio *StateIO) wrDedAny(ded *Any) {
	sio.wrBase(tidAny, &ded.dedBase)
	must.RetCtx(sio.wr.Write(sio.buf)).Msg("any")
}

func (sio *StateIO) rdDedAny() *Any {
	ded := &Any{dedBase: dedBase{cfg: sio.cfg}}
	sio.rdBase(&ded.dedBase)
	return ded
}

func (sio *StateIO) wrDedUnk(ded *Unknown) {
	sio.wrBase(tidUnknown, &ded.dedBase)
	must.RetCtx(sio.wr.Write(sio.buf)).Msg("unknown")
}

func (sio *StateIO) rdDedUnk() *Unknown {
	ded := &Unknown{dedBase: dedBase{cfg: sio.cfg}}
	sio.rdBase(&ded.dedBase)
	return ded
}

func (sio *StateIO) rdHeader() {
	line := must.RetCtx(sio.rd.ReadString('\n')).Msg("header line")
	line = line[:len(line)-1]
	if !strings.HasPrefix(line, "JSUM") {
		panic(eloc.New("not a JSUM state file"))
	}
	v := must.RetCtx(strconv.Atoi(line[4:])).Msg("header version")
	if v != StateVersion {
		panic(eloc.Errorf("unsupported state version %d", v))
	}
}

func (sio *StateIO) wrString(s string) {
	sio.StrCount++
	if id := sio.strs[s]; id > 0 {
		sio.StrDup++
		sio.buf = binary.AppendVarint(sio.buf[:0], id)
		must.RetCtx(sio.wr.Write(sio.buf)).Msg("string id")
	} else {
		id := int64(len(sio.strs) + 1)
		sio.strs[s] = id
		sio.buf = binary.AppendVarint(sio.buf[:0], -id)
		sio.buf = binary.AppendUvarint(sio.buf, uint64(len(s)))
		sio.buf = append(sio.buf, s...)
		must.RetCtx(sio.wr.Write(sio.buf)).Msg("string decl")
	}
}

func (sio *StateIO) rdString() string {
	sio.StrCount++
	id := must.RetCtx(binary.ReadVarint(&sio.rd)).Msg("read string ID")
	switch {
	case id > 0:
		s, ok := sio.sids[id]
		if !ok {
			panic(eloc.Errorf("unknown string id %d", id))
		}
		sio.StrDup++
		return s
	case id < 0:
		l := must.RetCtx(binary.ReadUvarint(&sio.rd)).Msg("read string len")
		sio.rd.checkU(l, "string len")
		if bl := len(sio.buf); bl < int(l) {
			sio.buf = slices.Grow(sio.buf, int(l)-bl)
		}
		sio.buf = sio.buf[:int(l)]
		must.RetCtx(io.ReadFull(&sio.rd, sio.buf)).Msg("read string text")
		s := string(sio.buf)
		sio.sids[-id] = s
		return s
	}
	panic(eloc.New("invalid string id"))
}

var ndn = binary.BigEndian

type restCountReader struct {
	r    *bufio.Reader
	rest int64
}

func (rc *restCountReader) Read(p []byte) (n int, err error) {
	n, err = rc.r.Read(p)
	rc.rest -= int64(n)
	return
}

func (rc *restCountReader) ReadByte() (b byte, err error) {
	b, err = rc.r.ReadByte()
	if err == nil {
		rc.rest--
	}
	return
}

func (rc *restCountReader) ReadString(delim byte) (s string, err error) {
	s, err = rc.r.ReadString(delim)
	rc.rest -= int64(len(s))
	return
}

const (
	statMinStrLen = 1
	statMinMbrSz  = 5
	statMinVarSz  = 3
)

func (rc *restCountReader) checkU(s uint64, f string, a ...any) {
	if s > math.MaxInt64 {
		panic(fmt.Errorf(
			"size %d exceeds int64 range \\"+f,
			append([]any{s}, a...),
		))
	}
	rc.checkI(int64(s), f, a...)
}

// checkI verifies that s bytes are available according to the reader's
// remaining count. If the remaining count is initialized with negative value
// the remaining size is considered unbounded and the check is skipped. If s
// is larger than the available rc.rest, checkI panics with a formatted error
// message.
func (rc *restCountReader) checkI(s int64, f string, a ...any) {
	if rc.rest < 0 {
		return
	}
	if s > rc.rest {
		panic(fmt.Errorf(
			"size %d exceeds rest %d \\"+f,
			append([]any{s, rc.rest}, a...),
		))
	}
}
