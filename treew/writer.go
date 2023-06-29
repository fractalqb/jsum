package treew

import (
	"bytes"
	"fmt"
	"io"
)

type Writer struct {
	w     io.Writer
	p     Prefix
	next  bool
	wrn   int
	wrerr error
}

func NewWriter(w io.Writer, s *Style) *Writer {
	return &Writer{
		w: w,
		p: Prefix{Style: s},
	}
}

func (w *Writer) Write(p []byte) (n int, err error) {
	if err != nil {
		n, err = w.wrn, w.wrerr
		w.wrn, w.wrerr = 0, nil
		return n, err
	}
	l := len(p)
	if l == 0 {
		return 0, nil
	}
	endnl := p[l-1] == '\n'
	if endnl {
		p = p[:l-1]
	}
	lines := bytes.Split(p, linesep)
	for _, l := range lines {
		if w.next {
			if m, err := io.WriteString(w.w, w.p.Cont(nil)); err != nil {
				return n + m, err
			} else {
				n += m
			}
		} else {
			w.next = true
		}
		if m, err := w.w.Write(l); err != nil {
			return n + m, err
		} else {
			n += m
		}
		if m, err := fmt.Fprintln(w.w); err != nil {
			return n + m, err
		} else {
			n += m
		}
	}
	if endnl {
		if w.next {
			if m, err := io.WriteString(w.w, w.p.Cont(nil)); err != nil {
				return n + m, err
			} else {
				n += m
			}
		} else {
			w.next = true
		}
		if m, err := fmt.Fprintln(w.w); err != nil {
			return n + m, err
		} else {
			n += m
		}
	}
	return n, nil
}

func (w *Writer) Descend() *Writer {
	w.p.Descend()
	return w
}

func (w *Writer) Ascend(up int) *Writer {
	w.p.Ascend(up)
	return w
}

func (w *Writer) First(s *Style) *Writer {
	w.wrn, w.wrerr = io.WriteString(w.w, w.p.First(s))
	w.next = false
	return w
}

func (w *Writer) Next(s *Style) *Writer {
	w.wrn, w.wrerr = io.WriteString(w.w, w.p.Next(s))
	w.next = false
	return w
}

func (w *Writer) Last(s *Style) *Writer {
	w.wrn, w.wrerr = io.WriteString(w.w, w.p.Last(s))
	w.next = false
	return w
}

var linesep = []byte{'\n'}
