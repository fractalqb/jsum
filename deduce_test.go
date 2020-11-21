package jsum

var (
	_ Deducer = (*Unknown)(nil)
	_ Deducer = (*Object)(nil)
	_ Deducer = (*Array)(nil)
	_ Deducer = (*Scalar)(nil)
	_ Deducer = (*Enum)(nil)
	_ Deducer = (*Union)(nil)
	_ Deducer = (*Any)(nil)
	_ Deducer = (*Number)(nil)
	_ Deducer = (*Invalid)(nil)
)
