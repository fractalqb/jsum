package jsum

type Invalid struct {
	error
}

func (Invalid) Accepts(_ interface{}) bool { return false }

func (i Invalid) Example(v interface{}) Deducer { return i }

func (Invalid) Nullable() bool { return false }

func (Invalid) Hash(dh DedupHash) uint64 { return 0 }

func (Invalid) Equal(_ Deducer) bool { return false }

func (Invalid) Copies() []Deducer { return nil }

var invBase dedBase

func (Invalid) super() *dedBase { return &invBase }
