package jsum

type NumberDup uint

const (
	NumberDupIntFloat NumberDup = (1 << iota)
	NumberDupMin
	NumberDupMax
)

type Config struct {
	DupNumber NumberDup
}
