package jsum

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
)

type SummaryConfig struct {
	Indent    string
	StringMax int
}

type Summary struct {
	w io.Writer
	SummaryConfig
}

func NewSummary(w io.Writer, cfg *SummaryConfig) *Summary {
	res := &Summary{w: w}
	if cfg != nil {
		res.SummaryConfig = *cfg
	}
	return res
}

func (s *Summary) Print(scm Deducer) error {
	return s.printIndet(scm, 0)
}

func (s *Summary) printIndet(scm Deducer, indent int) (err error) {
	switch ded := scm.(type) {
	case *String:
		err = s.str(ded, indent)
	case *Number:
		err = s.number(ded, indent)
	case *Object:
		err = s.object(ded, indent)
	case *Boolean:
		err = s.bool(ded, indent)
	case *Array:
		err = s.array(ded, indent)
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

func (s *Summary) str(n *String, indent int) error {
	minLen, maxLen := math.MaxInt, 0
	var strs []string
	for str := range n.stats {
		strs = append(strs, str)
		l := len(str)
		if l < minLen {
			minLen = l
		}
		if l > maxLen {
			maxLen = l
		}
	}
	f := "String len:%d...%d\n"
	if n.Nullable() {
		f = "[String] len:%d...%d\n"
	}
	s.indent(indent)
	fmt.Fprintf(s.w, f, minLen, maxLen)
	sort.Strings(strs)
	if len(strs) > s.StringMax {
		h := s.StringMax / 2
		t := s.StringMax - h - 1
		if h < 1 {
			h = 1
		}
		if t < 1 {
			t = 1
		}
		for _, str := range strs[:h] {
			s.indent(indent + 1)
			fmt.Fprintf(s.w, "%3d × \"%s\"\n", n.stats[str], str)
		}
		s.indent(indent + 1)
		fmt.Fprintf(s.w, "  … %d …\n", len(strs)-h-t)
		for _, str := range strs[len(strs)-t:] {
			s.indent(indent + 1)
			fmt.Fprintf(s.w, "%3d × \"%s\"\n", n.stats[str], str)
		}
	} else {
		for _, str := range strs {
			s.indent(indent + 1)
			fmt.Fprintf(s.w, "%3d × \"%s\"\n", n.stats[str], str)
		}
	}
	return nil
}

func (s *Summary) bool(n *Boolean, indent int) error {
	f := "Boolean true:%d / false:%d\n"
	if n.Nullable() {
		f = "[Boolean] true:%d / false:%d\n"
	}
	s.indent(indent)
	fmt.Fprintf(s.w, f, n.tNo, n.fNo)
	return nil
}

func (s *Summary) number(n *Number, indent int) error {
	s.indent(indent)
	var sum string
	if n.isFloat {
		if n.hadFrac {
			sum = fmt.Sprintf("Number %f–%f", n.min, n.max)
		} else {
			sum = fmt.Sprintf("Number %f–%f 0-fracs", n.min, n.max)
		}
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
	for a := range o.mbrs {
		nms = append(nms, a)
	}
	sort.Strings(nms)
	for i, a := range nms {
		s.indent(indent + 1)
		m := o.mbrs[a]
		if m.occurence < o.count {
			fmt.Fprintf(s.w, "#%-2d \"%s\" in %d / %d:\n", i+1, a, m.occurence, o.count)
		} else {
			fmt.Fprintf(s.w, "#%-2d \"%s\" mandatory (%d×):\n", i+1, a, m.occurence)
		}
		if err := s.printIndet(m.ded, indent+2); err != nil {
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
		fmt.Fprintf(s.w, "[Array] of %s:\n", lens)
	} else {
		fmt.Fprintf(s.w, "Array of %s:\n", lens)
	}
	return s.printIndet(a.elem, indent+1)
}

func (s *Summary) union(u *Union, indent int) error {
	s.indent(indent)
	fmt.Fprintf(s.w, "Union of %d types:\n", len(u.variants))
	for _, d := range u.variants {
		if err := s.printIndet(d, indent+1); err != nil {
			return err
		}
	}
	return nil
}

func intWidth(i int) int {
	switch {
	case i == 0:
		return 1
	case i < 0:
		return 1 + intWidth(-i)
	}
	l := math.Log10(float64(i))
	return int(math.Trunc(l)) + 1
}
