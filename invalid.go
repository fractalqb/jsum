package jsum

type Invalid struct {
	error
}

func (_ Invalid) Accepts(_ interface{}) bool { return false }

func (i Invalid) Example(v interface{}) Deducer { return i }

func (_ Invalid) Nullable() bool { return false }

func (_ Invalid) setNullable(_ bool) {}

func (_ Invalid) Hash(dh DedupHash) uint64 { return 0 }

func (_ Invalid) Equal(_ Deducer) bool { return false }

func (_ Invalid) Copies() []Deducer { return nil }

var invBase dedBase

func (_ Invalid) super() *dedBase { return &invBase }
