package jsum

type Invalid struct {
	error
}

func (Invalid) Accepts(_ any) bool { return false }

func (i Invalid) Example(v any) Deducer { return i }

func (Invalid) Nulls() int { return -1 }

func (Invalid) Hash(dh DedupHash) uint64 { return 0 }

func (Invalid) Equal(_ Deducer) bool { return false }

func (Invalid) Copies() []Deducer { return nil }

var invBase dedBase

func (Invalid) super() *dedBase { return &invBase }
