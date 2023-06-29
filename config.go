package jsum

type DedupBool uint

const (
	DedupBoolTrue DedupBool = 1 << iota
	DedupBoolFalse
)

type DedupNumber uint

const (
	DedpuNumberIntFloat DedupNumber = 1 << iota
	DedupNumberFrac
	DedupNumberMin
	DedupNumberMax
	DedupNumberNeg
	DedupNumberPos
)

type DedupString uint

const (
	DedupStringEmpty DedupString = 1 << iota
)

type Config struct {
	DedupBool   DedupBool
	DedupNumber DedupNumber
	DedupString DedupString
}
