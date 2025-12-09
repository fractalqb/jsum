package jsum

import "testing"

func TestUnknow_toBool(t *testing.T) {
	var cfg Config
	var u Deducer = NewUnknown(&cfg)
	u = u.Example(true)
	if b, ok := u.(*Boolean); !ok {
		t.Fatalf("deduced not bool but %T", u)
	} else if b.tNo != 1 {
		t.Errorf("true count %d not 1", b.tNo)
	}
	u = u.Example(true)
	if b, ok := u.(*Boolean); !ok {
		t.Fatalf("deduced not bool but %T", u)
	} else if b.tNo != 2 {
		t.Errorf("true count %d not 2", b.tNo)
	}
}
