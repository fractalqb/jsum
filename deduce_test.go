package jsum

var (
	_ Deducer = (*Unknown)(nil)
	_ Deducer = (*Object)(nil)
	_ Deducer = (*String)(nil)
	_ Deducer = (*Number)(nil)
	_ Deducer = (*Boolean)(nil)
	_ Deducer = (*String)(nil)
	_ Deducer = (*Union)(nil)
	_ Deducer = (*Any)(nil)
	_ Deducer = (*Invalid)(nil)
)
