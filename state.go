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
	"slices"
	"strconv"
	"strings"

	"git.fractalqb.de/fractalqb/eloc"
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
	buf              []byte
	strs             map[string]int64
	sids             map[int64]string
	StrCount, StrDup int
}

func (sio *StateIO) Write(w io.Writer, ded Deducer) error {
	if _, err := fmt.Fprintf(w, "JSUM%d\n", StateVersion); err != nil {
		return eloc.At(err)
	}
	if sio.strs == nil {
		sio.strs = make(map[string]int64)
	} else {
		clear(sio.strs)
	}
	sio.StrCount, sio.StrDup = 0, 0
	return sio.wrDed(w, ded)
}

func (sio *StateIO) Read(r io.Reader, cfg *Config) (Deducer, error) {
	br := bufio.NewReader(r)
	if err := sio.rdHeader(br); err != nil {
		return nil, err
	}
	if sio.sids == nil {
		sio.sids = make(map[int64]string)
	} else {
		clear(sio.sids)
	}
	sio.StrCount, sio.StrDup = 0, 0
	return sio.rdDed(br, cfg)
}

func (sio *StateIO) wrDed(w io.Writer, ded Deducer) error {
	switch ded := ded.(type) {
	case *String:
		return sio.wrDedStr(w, ded)
	case *Number:
		return sio.wrDedNum(w, ded)
	case *Boolean:
		return sio.wrDedBool(w, ded)
	case *Object:
		return sio.wrDedObj(w, ded)
	case *Array:
		return sio.wrDedArray(w, ded)
	case *Union:
		return sio.wrDedUnion(w, ded)
	case *Any:
		return sio.wrDedAny(w, ded)
	case *Unknown:
		return sio.wrDedUnk(w, ded)
	case *Invalid:
		return eloc.Errorf("invalid deducer: %w", ded.error)
	}
	return eloc.Errorf("unsupported deducer: %T", ded)
}

func (sio *StateIO) rdDed(r *bufio.Reader, cfg *Config) (Deducer, error) {
	tid, err := r.ReadByte()
	if err != nil {
		return nil, eloc.At(err)
	}
	switch tid {
	case tidString:
		return sio.rdDedStr(r, cfg)
	case tidNumber:
		return sio.rdDedNum(r, cfg)
	case tidBool:
		return sio.rdDedBool(r, cfg)
	case tidObject:
		return sio.rdDedObj(r, cfg)
	case tidArray:
		return sio.rdDedArray(r, cfg)
	case tidUnion:
		return sio.rdDedUnion(r, cfg)
	case tidAny:
		return sio.rdDedAny(r, cfg)
	case tidUnknown:
		return sio.rdDedUnk(r, cfg)
	}
	return nil, eloc.Errorf("illegal deducer type id: %d", tid)
}

func (sio *StateIO) wrBase(tid byte, ded *dedBase) {
	sio.buf = append(sio.buf[:0], tid)
	sio.buf = binary.AppendUvarint(sio.buf, uint64(ded.Count))
	sio.buf = binary.AppendUvarint(sio.buf, uint64(ded.Null))
}

func (sio *StateIO) rdBase(r *bufio.Reader, ded *dedBase) error {
	u, err := binary.ReadUvarint(r)
	if err != nil {
		return eloc.Errorf("base null: %w", err)
	}
	ded.Count = int(u)
	if u, err = binary.ReadUvarint(r); err != nil {
		return eloc.Errorf("base null: %w", err)
	}
	ded.Null = int(u)
	return nil
}

func (sio *StateIO) wrDedStr(w io.Writer, ded *String) error {
	sio.wrBase(tidString, &ded.dedBase)
	sio.buf = binary.AppendUvarint(sio.buf, uint64(len(ded.Stats)))
	if _, err := w.Write(sio.buf); err != nil {
		return eloc.At(err)
	}
	for s, n := range ded.Stats {
		if err := sio.wrString(w, s); err != nil {
			return err
		}
		sio.buf = binary.AppendUvarint(sio.buf[:0], uint64(n))
		if _, err := w.Write(sio.buf); err != nil {
			return eloc.At(err)
		}
	}
	sio.buf = binary.AppendVarint(sio.buf[:0], int64(ded.Format))
	_, err := w.Write(sio.buf)
	return eloc.At(err)
}

func (sio *StateIO) rdDedStr(r *bufio.Reader, cfg *Config) (*String, error) {
	ded := &String{dedBase: dedBase{cfg: cfg}}
	if err := sio.rdBase(r, &ded.dedBase); err != nil {
		return nil, err
	}
	nstats, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, eloc.Errorf("string stats len: %w", err)
	}
	ded.Stats = make(map[string]int, nstats)
	for i := range nstats {
		s, err := sio.rdString(r)
		if err != nil {
			return nil, fmt.Errorf("string stat %d string: %w", i, err)
		}
		n, err := binary.ReadUvarint(r)
		if err != nil {
			return nil, eloc.Errorf("string stat %d count: %w", i, err)
		}
		ded.Stats[s] = int(n)
	}
	form, err := binary.ReadVarint(r)
	if err != nil {
		return nil, eloc.Errorf("string format: %w", err)
	}
	ded.Format = Format(form)
	return ded, nil
}

func (sio *StateIO) wrDedNum(w io.Writer, ded *Number) (err error) {
	sio.wrBase(tidNumber, &ded.dedBase)
	if sio.buf, err = binary.Append(sio.buf, ndn, ded.Min); err != nil {
		return eloc.At(err)
	}
	if sio.buf, err = binary.Append(sio.buf, ndn, ded.Max); err != nil {
		return eloc.At(err)
	}
	var flags byte
	if ded.IsFloat {
		flags |= 1
	}
	if ded.HasFrac {
		flags |= 2
	}
	sio.buf = append(sio.buf, flags)
	_, err = w.Write(sio.buf)
	return eloc.At(err)
}

func (sio *StateIO) rdDedNum(r *bufio.Reader, cfg *Config) (*Number, error) {
	ded := &Number{dedBase: dedBase{cfg: cfg}}
	if err := sio.rdBase(r, &ded.dedBase); err != nil {
		return nil, err
	}
	if err := binary.Read(r, ndn, &ded.Min); err != nil {
		return nil, eloc.At(err)
	}
	if err := binary.Read(r, ndn, &ded.Max); err != nil {
		return nil, eloc.At(err)
	}
	flags, err := r.ReadByte()
	if err != nil {
		return nil, eloc.At(err)
	}
	ded.IsFloat = flags&1 != 0
	ded.HasFrac = flags&2 != 0
	return ded, nil
}

func (sio *StateIO) wrDedBool(w io.Writer, ded *Boolean) (err error) {
	sio.wrBase(tidBool, &ded.dedBase)
	sio.buf = binary.AppendUvarint(sio.buf, uint64(ded.TrueNo))
	sio.buf = binary.AppendUvarint(sio.buf, uint64(ded.FalseNo))
	_, err = w.Write(sio.buf)
	return eloc.At(err)
}

func (sio *StateIO) rdDedBool(r *bufio.Reader, cfg *Config) (ded *Boolean, err error) {
	ded = &Boolean{dedBase: dedBase{cfg: cfg}}
	if err = sio.rdBase(r, &ded.dedBase); err != nil {
		return nil, err
	}
	var tmp uint64
	if tmp, err = binary.ReadUvarint(r); err != nil {
		return nil, eloc.At(err)
	}
	ded.TrueNo = int(tmp)
	if tmp, err = binary.ReadUvarint(r); err != nil {
		return nil, eloc.At(err)
	}
	ded.FalseNo = int(tmp)
	return ded, nil
}

func (sio *StateIO) wrDedObj(w io.Writer, ded *Object) (err error) {
	sio.wrBase(tidObject, &ded.dedBase)
	sio.buf = binary.AppendUvarint(sio.buf, uint64(len(ded.Members)))
	if _, err = w.Write(sio.buf); err != nil {
		return eloc.At(err)
	}
	for n, m := range ded.Members {
		if err = sio.wrString(w, n); err != nil {
			return err
		}
		sio.buf = binary.AppendUvarint(sio.buf[:0], uint64(m.Occurence))
		if _, err = w.Write(sio.buf); err != nil {
			return eloc.At(err)
		}
		if err = sio.wrDed(w, m.Ded); err != nil {
			return eloc.At(err)
		}
	}
	return nil
}

func (sio *StateIO) rdDedObj(r *bufio.Reader, cfg *Config) (_ *Object, err error) {
	ded := &Object{dedBase: dedBase{cfg: cfg}}
	if err = sio.rdBase(r, &ded.dedBase); err != nil {
		return nil, err
	}
	var mno uint64
	if mno, err = binary.ReadUvarint(r); err != nil {
		return nil, eloc.At(err)
	}
	ded.Members = make(map[string]Member, mno)
	for range mno {
		n, err := sio.rdString(r)
		if err != nil {
			return nil, err
		}
		occ, err := binary.ReadUvarint(r)
		if err != nil {
			return nil, eloc.At(err)
		}
		mded, err := sio.rdDed(r, cfg)
		if err != nil {
			return nil, err
		}
		ded.Members[n] = Member{Occurence: int(occ), Ded: mded}
	}
	return ded, nil
}

func (sio *StateIO) wrDedArray(w io.Writer, ded *Array) (err error) {
	sio.wrBase(tidArray, &ded.dedBase)
	sio.buf = binary.AppendVarint(sio.buf, int64(ded.MinLen))
	sio.buf = binary.AppendVarint(sio.buf, int64(ded.MaxLen))
	if _, err = w.Write(sio.buf); err != nil {
		return eloc.At(err)
	}
	return sio.wrDed(w, ded.Elem)
}

func (sio *StateIO) rdDedArray(r *bufio.Reader, cfg *Config) (_ *Array, err error) {
	ded := &Array{dedBase: dedBase{cfg: cfg}}
	if err := sio.rdBase(r, &ded.dedBase); err != nil {
		return nil, err
	}
	var tmp int64
	if tmp, err = binary.ReadVarint(r); err != nil {
		return nil, eloc.At(err)
	}
	ded.MinLen = int(tmp)
	if tmp, err = binary.ReadVarint(r); err != nil {
		return nil, eloc.At(err)
	}
	ded.MaxLen = int(tmp)
	if eded, err := sio.rdDed(r, cfg); err != nil {
		return nil, err
	} else {
		ded.Elem = eded
	}
	return ded, nil
}

func (sio *StateIO) wrDedUnion(w io.Writer, ded *Union) (err error) {
	sio.wrBase(tidUnion, &ded.dedBase)
	sio.buf = binary.AppendUvarint(sio.buf, uint64(len(ded.Variants)))
	if _, err = w.Write(sio.buf); err != nil {
		return eloc.At(err)
	}
	for _, v := range ded.Variants {
		if err := sio.wrDed(w, v); err != nil {
			return err
		}
	}
	return nil
}

func (sio *StateIO) rdDedUnion(r *bufio.Reader, cfg *Config) (_ *Union, err error) {
	ded := &Union{dedBase: dedBase{cfg: cfg}}
	if err := sio.rdBase(r, &ded.dedBase); err != nil {
		return nil, err
	}
	var varNo uint64
	if varNo, err = binary.ReadUvarint(r); err != nil {
		return nil, eloc.At(err)
	}
	ded.Variants = make([]Deducer, varNo)
	for i := range varNo {
		vded, err := sio.rdDed(r, cfg)
		if err != nil {
			return nil, err
		}
		ded.Variants[i] = vded
	}
	return ded, nil
}

func (sio *StateIO) wrDedAny(w io.Writer, ded *Any) (err error) {
	sio.wrBase(tidAny, &ded.dedBase)
	_, err = w.Write(sio.buf)
	return eloc.At(err)
}

func (sio *StateIO) rdDedAny(r *bufio.Reader, cfg *Config) (*Any, error) {
	ded := &Any{dedBase: dedBase{cfg: cfg}}
	if err := sio.rdBase(r, &ded.dedBase); err != nil {
		return nil, err
	}
	return ded, nil
}

func (sio *StateIO) wrDedUnk(w io.Writer, ded *Unknown) (err error) {
	sio.wrBase(tidUnknown, &ded.dedBase)
	_, err = w.Write(sio.buf)
	return eloc.At(err)
}

func (sio *StateIO) rdDedUnk(r *bufio.Reader, cfg *Config) (*Unknown, error) {
	ded := &Unknown{dedBase: dedBase{cfg: cfg}}
	if err := sio.rdBase(r, &ded.dedBase); err != nil {
		return nil, err
	}
	return ded, nil
}

func (sio *StateIO) rdHeader(r *bufio.Reader) error {
	line, err := r.ReadString('\n')
	if err != nil {
		return eloc.At(err)
	}
	line = line[:len(line)-1]
	if !strings.HasPrefix(line, "JSUM") {
		return eloc.New("not a JSUM state file")
	}
	if v, err := strconv.Atoi(line[4:]); err != nil {
		return eloc.Errorf("JSUM header version: %w", err)
	} else if v != StateVersion {
		return eloc.Errorf("unsupported state version %d", v)
	}
	return nil
}

func (sio *StateIO) wrString(w io.Writer, s string) error {
	sio.StrCount++
	if id := sio.strs[s]; id > 0 {
		sio.StrDup++
		sio.buf = binary.AppendVarint(sio.buf[:0], id)
		_, err := w.Write(sio.buf)
		return eloc.At(err)
	}
	id := int64(len(sio.strs) + 1)
	sio.strs[s] = id
	sio.buf = binary.AppendVarint(sio.buf[:0], -id)
	sio.buf = binary.AppendUvarint(sio.buf, uint64(len(s)))
	sio.buf = append(sio.buf, s...)
	_, err := w.Write(sio.buf)
	return eloc.At(err)
}

func (sio *StateIO) rdString(r *bufio.Reader) (string, error) {
	sio.StrCount++
	id, err := binary.ReadVarint(r)
	if err != nil {
		return "", eloc.At(err)
	}
	switch {
	case id > 0:
		s, ok := sio.sids[id]
		if !ok {
			return "", eloc.Errorf("unknown string id %d", id)
		}
		sio.StrDup++
		return s, nil
	case id < 0:
		l, err := binary.ReadUvarint(r)
		if err != nil {
			return "", eloc.At(err)
		}
		// FIXME restrict size
		if bl := len(sio.buf); bl < int(l) {
			sio.buf = slices.Grow(sio.buf, int(l)-bl)
		}
		sio.buf = sio.buf[:int(l)]
		if _, err := io.ReadFull(r, sio.buf); err != nil {
			return "", eloc.At(err)
		}
		s := string(sio.buf)
		sio.sids[-id] = s
		return s, nil
	}
	return "", eloc.New("invalid string id")
}

var ndn = binary.BigEndian
