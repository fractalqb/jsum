package jsum

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"

	"git.fractalqb.de/fractalqb/tetrta"
)

type SummaryConfig struct {
	TreeStyle *tetrta.TreeStyle
	StringMax int
}

type Summary struct {
	w    io.Writer
	tree tetrta.Tree
	SummaryConfig
}

func NewSummary(w io.Writer, cfg *SummaryConfig) *Summary {
	res := &Summary{w: w}
	if cfg != nil {
		res.SummaryConfig = *cfg
		res.tree.Style = cfg.TreeStyle
	}
	return res
}

func (s *Summary) Print(scm Deducer) error {
	return s.printIndet(scm, true)
}

func (s *Summary) printIndet(scm Deducer, last bool) (err error) {
	if last {
		io.WriteString(s.w, s.tree.Last(nil))
	} else {
		io.WriteString(s.w, s.tree.Next(nil))
	}
	switch ded := scm.(type) {
	case *String:
		err = s.str(ded)
	case *Number:
		err = s.number(ded)
	case *Object:
		err = s.object(ded)
	case *Boolean:
		err = s.bool(ded)
	case *Array:
		err = s.array(ded)
	case *Union:
		err = s.union(ded)
	case *Any:
		if ns := ded.Nulls(); ns > 0 {
			fmt.Fprintf(s.w, "[Any %d×null]\n", ns)
		} else {
			io.WriteString(s.w, "Any\n")
		}
	case *Unknown:
		if ns := ded.Nulls(); ns > 0 {
			fmt.Fprintf(s.w, "[??? %d×null]\n", ns)
		} else {
			io.WriteString(s.w, "???\n")
		}
	case Invalid:
		fmt.Fprintf(s.w, "<INVALID: %s>\n", ded)
	default:
		err = fmt.Errorf("unsupported deducer type %T", scm)
	}
	return err
}

func (s *Summary) str(n *String) error {
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
	if ns := n.Nulls(); ns > 0 {
		fmt.Fprintf(s.w, "[String %d×null]", ns)
	} else {
		fmt.Fprint(s.w, "String")
	}
	switch n.format {
	case DateTimeFormat:
		fmt.Fprintln(s.w, " format=date-time")
	default:
		if minLen == maxLen {
			fmt.Fprintf(s.w, " len:%d\n", minLen)
		} else {
			fmt.Fprintf(s.w, " len:%d..%d\n", minLen, maxLen)
		}
	}
	sort.Strings(strs)
	switch s.StringMax {
	case 0:
		return nil
	case 1:
		str := strs[0]
		s.tree.Descend()
		fmt.Fprintf(s.w, "%s%d × %q", s.tree.Last(nil), n.stats[str], str)
		if len(strs) > 1 {
			fmt.Fprintln(s.w, "…")
		} else {
			fmt.Fprintln(s.w)
		}
		s.tree.Ascend(1)
		return nil
	}
	s.tree.Descend()
	if len(strs) > s.StringMax {
		t := s.StringMax / 2
		h := s.StringMax - t
		var iw int
		for _, str := range strs[:h] {
			iw = maxIntWidth(iw, n.stats[str])
		}
		for _, str := range strs[len(strs)-t:] {
			iw = maxIntWidth(iw, n.stats[str])
		}
		form := fmt.Sprintf("%%s%%%dd x %%q\n", iw)
		for _, str := range strs[:h] {
			fmt.Fprintf(s.w, form, s.tree.Next(nil), n.stats[str], str)
		}
		fmt.Fprintf(s.w, "%s... %d ...\n",
			s.tree.Cont(nil),
			len(strs)-h-t,
		)
		strs = strs[len(strs)-t:]
		for i, str := range strs {
			var pf string
			if i == len(strs)-1 {
				pf = s.tree.Last(nil)
			} else {
				pf = s.tree.Next(nil)
			}
			fmt.Fprintf(s.w, form, pf, n.stats[str], str)
		}
	} else {
		var iw int
		for _, n := range n.stats {
			iw = maxIntWidth(iw, n)
		}
		form := fmt.Sprintf("%%s%%%dd x %%q\n", iw)
		for i, str := range strs {
			var pf string
			if i == len(strs)-1 {
				pf = s.tree.Last(nil)
			} else {
				pf = s.tree.Next(nil)
			}
			fmt.Fprintf(s.w, form, pf, n.stats[str], str)
		}
	}
	s.tree.Ascend(1)
	return nil
}

func (s *Summary) bool(n *Boolean) error {
	if ns := n.Nulls(); ns > 0 {
		fmt.Fprintf(s.w, "[Boolean %d×null] true:%d / false:%d\n", ns, n.tNo, n.fNo)
	} else {
		fmt.Fprintf(s.w, "Boolean true:%d / false:%d\n", n.tNo, n.fNo)
	}
	return nil
}

func (s *Summary) number(n *Number) error {
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
	if ns := n.Nulls(); ns > 0 {
		fmt.Fprintf(s.w, "[%s %d×null]\n", sum, ns)
	} else {
		fmt.Fprintf(s.w, "%s\n", sum)
	}
	return nil
}

func (s *Summary) object(o *Object) error {
	if ns := o.Nulls(); ns > 0 {
		fmt.Fprintf(s.w, "[Object %d×null] with %d members:\n", ns, len(o.mbrs))
	} else {
		fmt.Fprintf(s.w, "Object with %d members:\n", len(o.mbrs))
	}
	nms := make([]string, 0, len(o.mbrs))
	for a := range o.mbrs {
		nms = append(nms, a)
	}
	sort.Strings(nms)
	s.tree.Descend()
	for i, a := range nms {
		var pf string
		if i == len(nms)-1 {
			pf = s.tree.Last(nil)
		} else {
			pf = s.tree.Next(nil)
		}
		m := o.mbrs[a]
		fmt.Fprintf(s.w, "%s#%-2d \"%s\" ", pf, i+1, a)
		if m.occurence < o.count {
			fmt.Fprintf(s.w, "optional (%d/%d %.0f%%)",
				m.occurence,
				o.count,
				100*float64(m.occurence)/float64(o.count),
			)
		} else {
			fmt.Fprintf(s.w, "mandatory (%d×)", m.occurence)
		}
		fmt.Println(":")
		s.tree.Descend()
		if err := s.printIndet(m.ded, true); err != nil {
			return err
		}
		s.tree.Ascend(1)
	}
	s.tree.Ascend(1)
	return nil
}

func (s *Summary) array(a *Array) error {
	var lens string
	if a.minLen == a.maxLen {
		lens = strconv.Itoa(a.minLen)
	} else {
		lens = fmt.Sprintf("%d..%d", a.minLen, a.maxLen)
	}
	if ns := a.Nulls(); ns > 0 {
		fmt.Fprintf(s.w, "[Array %d×null] of %s:\n", ns, lens)
	} else {
		fmt.Fprintf(s.w, "Array of %s:\n", lens)
	}
	s.tree.Descend()
	defer s.tree.Ascend(1)
	return s.printIndet(a.elem, true)
}

func (s *Summary) union(u *Union) error {
	fmt.Fprintf(s.w, "Union of %d types:\n", len(u.variants))
	s.tree.Descend()
	for i, d := range u.variants {
		if err := s.printIndet(d, i == len(u.variants)-1); err != nil {
			return err
		}
	}
	s.tree.Ascend(1)
	return nil
}

func maxIntWidth(width int, i int) int {
	if w := intWidth(i); w > width {
		width = w
	}
	return width
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
