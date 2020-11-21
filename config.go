package jsum

import (
	"math"
)

type NumberHash uint

const (
	NumberHashIntFloat NumberHash = (1 << iota)
	NumberHashMin
	NumberHashMax
)

type Config struct {
	// Check if a value v shall be deduced as a—possibly existing—enum. If
	// AsEnum is nil no value will be deduced to be an enum.
	AsEnum func(e *Enum, v interface{}) bool

	NumberHash NumberHash
}

func (cfg *Config) testAsEnum(e *Enum, v interface{}) bool {
	if cfg.AsEnum == nil {
		return false
	}
	return cfg.AsEnum(e, v)
}

func KeepBoolEnum(e *Enum, v interface{}) bool {
	if e == nil {
		return true
	}
	other := !v.(bool)
	for l, n := range e.lits {
		if n > 0 && l == other {
			return false
		}
	}
	return true
}

type EnumTest func(*Enum, interface{}) bool

type KeepJsonEnum struct {
	Default bool
	PerType map[JsonType]EnumTest
}

func (kje KeepJsonEnum) Test(e *Enum, v interface{}) bool {
	if test := kje.PerType[JsonTypeOf(v)]; test != nil {
		return test(e, v)
	}
	return kje.Default
}

type KeepStringEnum int

func (kse KeepStringEnum) Test(e *Enum, v interface{}) bool {
	switch {
	case kse == 0:
		return false
	case e == nil:
		return true
	}
	return len(e.lits) < int(kse)
}

type KeepNumberEnum int

func (kne KeepNumberEnum) Test(e *Enum, v interface{}) bool {
	x := asNumber(v)
	switch {
	case math.IsNaN(x):
		return false
	case math.IsInf(x, 1):
		return false
	case math.IsInf(x, -1):
		return false
	}
	if _, f := math.Modf(x); f != 0 {
		return false
	}
	if e == nil {
		return true
	}
	return len(e.lits) < int(kne)
}
