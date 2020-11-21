package jsum

import (
	"fmt"
	"io"
	"sort"
	"strconv"
)

type Summary struct {
	w      io.Writer
	Indent string
}

func NewSummary(w io.Writer) *Summary {
	return &Summary{w: w}
}

func (s *Summary) Print(scm Deducer) error {
	return s.printIndet(scm, 0)
}

func (s *Summary) printIndet(scm Deducer, indent int) (err error) {
	switch ded := scm.(type) {
	case *Scalar:
		f := "%s"
		if ded.Nullable() {
			f = "[%s]"
		}
		switch ded.jt {
		case JsonString:
			s.indent(indent)
			fmt.Fprintf(s.w, f, "String\n")
		case JsonBool:
			s.indent(indent)
			fmt.Fprintf(s.w, f, "Boolean\n")
		default:
			err = fmt.Errorf("illegal scalar %d", ded.jt)
			fmt.Fprintf(s.w, "<%s>\n", err)
		}
	case *Number:
		err = s.number(ded, indent)
	case *Object:
		err = s.object(ded, indent)
	case *Array:
		err = s.array(ded, indent)
	case *Enum:
		err = s.enum(ded, indent)
	case *Union:
		err = s.union(ded, indent)
	case *Any:
		s.indent(indent)
		if ded.Nullable() {
			io.WriteString(s.w, "Any\n")
		} else {
			io.WriteString(s.w, "[Any]\n")
		}
	case *Unknown:
		s.indent(indent)
		if ded.Nullable() {
			io.WriteString(s.w, "???\n")
		} else {
			io.WriteString(s.w, "[???]\n")
		}
	default:
		err = fmt.Errorf("unsupported deducer type %T", scm)
	}
	return err
}

func (s *Summary) indent(i int) {
	for i > 0 {
		io.WriteString(s.w, s.Indent)
		i--
	}
}

func (s *Summary) number(n *Number, indent int) error {
	s.indent(indent)
	var sum string
	if n.isFloat {
		sum = fmt.Sprintf("Number %f–%f", n.min, n.max)
	} else {
		sum = fmt.Sprintf("Integer %d–%d", int64(n.min), int64(n.max))
	}
	if n.Nullable() {
		fmt.Fprintf(s.w, "[%s]\n", sum)
	} else {
		fmt.Fprintf(s.w, "%s\n", sum)
	}
	return nil
}

func (s *Summary) object(o *Object, indent int) error {
	s.indent(indent)
	fmt.Fprintf(s.w, "Object with %d members:\n", len(o.mbrs))
	nms := make([]string, 0, len(o.mbrs))
	for a, _ := range o.mbrs {
		nms = append(nms, a)
	}
	sort.Strings(nms)
	for i, a := range nms {
		s.indent(indent + 1)
		m := o.mbrs[a]
		if m.n < o.count {
			fmt.Fprintf(s.w, "#%-2d \"%s\" %d/%d:\n", i+1, a, m.n, o.count)
		} else {
			fmt.Fprintf(s.w, "#%-2d \"%s\" × %d:\n", i+1, a, m.n)
		}
		if err := s.printIndet(m.d, indent+2); err != nil {
			return err
		}
	}
	return nil
}

func (s *Summary) array(a *Array, indent int) error {
	s.indent(indent)
	var lens string
	if a.minLen == a.maxLen {
		lens = strconv.Itoa(a.minLen)
	} else {
		lens = fmt.Sprintf("%d…%d", a.minLen, a.maxLen)
	}
	if a.Nullable() {
		fmt.Fprintf(s.w, "[Array %s] of:\n", lens)
	} else {
		fmt.Fprintf(s.w, "Array %s of:\n", lens)
	}
	return s.printIndet(a.ed, indent+1)
}

func (s *Summary) enum(e *Enum, indent int) error {
	if len(e.lits) == 1 {
		return s.printIndet(e.base, indent)
	}
	s.indent(indent)
	fmt.Fprintf(s.w, "Enum with %d literals of:\n", len(e.lits))
	if err := s.printIndet(e.base, indent+1); err != nil {
		return err
	}
	for v, n := range e.lits {
		s.indent(indent + 1)
		fmt.Fprintf(s.w, "%d ×\t{%+v}\n", n, v)
	}
	return nil
}

func (s *Summary) union(u *Union, indent int) error {
	s.indent(indent)
	fmt.Fprintf(s.w, "Union of %d types:\n", len(u.ds))
	for _, d := range u.ds {
		if err := s.printIndet(d, indent+1); err != nil {
			return err
		}
	}
	return nil
}
