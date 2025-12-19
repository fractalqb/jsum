package jsum

import (
	"fmt"
	"io"
	"maps"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

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
		fmt.Fprintln(s.w, AnyLabel(ded))
	case *Unknown:
		fmt.Fprintln(s.w, UnknownLabel(ded))
	case Invalid:
		fmt.Fprintln(s.w, InvalidLabel(ded))
	default:
		err = fmt.Errorf("unsupported deducer type %T", scm)
	}
	return err
}

func AnyLabel(n *Any) string {
	if ns := n.Nulls(); ns > 0 {
		return fmt.Sprintf("[Any %d×null]", ns)
	}
	return "Any"
}

func UnknownLabel(n *Unknown) string {
	if ns := n.Nulls(); ns > 0 {
		return fmt.Sprintf("[??? %d×null]", ns)
	}
	return "???"
}

func InvalidLabel(n Invalid) string {
	return fmt.Sprintf("<INVALID: %s>\n", n.error.Error())
}

func StringLabel(n *String) string {
	var sb strings.Builder
	minLen, maxLen := math.MaxInt, 0
	for str := range n.Stats {
		l := len(str)
		if l < minLen {
			minLen = l
		}
		if l > maxLen {
			maxLen = l
		}
	}
	if ns := n.Nulls(); ns > 0 {
		fmt.Fprintf(&sb, "[String %d×null]", ns)
	} else {
		fmt.Fprint(&sb, "String")
	}
	switch n.format {
	case DateTimeFormat:
		fmt.Fprint(&sb, " format=date-time")
	default:
		if minLen == maxLen {
			fmt.Fprintf(&sb, " len:%d", minLen)
		} else {
			fmt.Fprintf(&sb, " len:%d..%d", minLen, maxLen)
		}
	}
	fmt.Fprintf(&sb, " distinct:%d", len(n.Stats))
	return sb.String()
}

func (s *Summary) str(n *String) error {
	fmt.Fprintln(s.w, StringLabel(n))
	strs := slices.Collect(maps.Keys(n.Stats))
	sort.Strings(strs)
	switch s.StringMax {
	case 0:
		return nil
	case 1:
		str := strs[0]
		s.tree.Descend()
		fmt.Fprintf(s.w, "%s%d × %q", s.tree.Last(nil), n.Stats[str], str)
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
			iw = maxIntWidth(iw, n.Stats[str])
		}
		for _, str := range strs[len(strs)-t:] {
			iw = maxIntWidth(iw, n.Stats[str])
		}
		form := fmt.Sprintf("%%s%%%dd x %%q\n", iw)
		for _, str := range strs[:h] {
			fmt.Fprintf(s.w, form, s.tree.Next(nil), n.Stats[str], str)
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
			fmt.Fprintf(s.w, form, pf, n.Stats[str], str)
		}
	} else {
		var iw int
		for _, n := range n.Stats {
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
			fmt.Fprintf(s.w, form, pf, n.Stats[str], str)
		}
	}
	s.tree.Ascend(1)
	return nil
}

func BoolLabel(n *Boolean) string {
	if ns := n.Nulls(); ns > 0 {
		return fmt.Sprintf("[Boolean %d×null] true:%d / false:%d\n", ns, n.tNo, n.fNo)
	}
	return fmt.Sprintf("Boolean true:%d / false:%d\n", n.tNo, n.fNo)
}

func (s *Summary) bool(n *Boolean) error {
	fmt.Fprintln(s.w, BoolLabel(n))
	return nil
}

func NumberLabel(n *Number) string {
	var sum string
	if n.isFloat {
		if n.min == n.max {
			if n.hadFrac {
				sum = fmt.Sprintf("Number = %f", n.min)
			} else {
				sum = fmt.Sprintf("Number = %f 0-fracs", n.min)
			}
		} else {
			if n.hadFrac {
				sum = fmt.Sprintf("Number %f–%f", n.min, n.max)
			} else {
				sum = fmt.Sprintf("Number %f–%f 0-fracs", n.min, n.max)
			}
		}
	} else {
		mi, ma := int64(n.min), int64(n.max)
		if mi == ma {
			sum = fmt.Sprintf("Integer = %d", mi)
		} else {
			sum = fmt.Sprintf("Integer %d–%d", mi, ma)
		}
	}
	if ns := n.Nulls(); ns > 0 {
		return fmt.Sprintf("[%s %d×null]", sum, ns)
	}
	return sum
}

func (s *Summary) number(n *Number) error {
	fmt.Fprintln(s.w, NumberLabel(n))
	return nil
}

func ObjectLabel(o *Object) string {
	if ns := o.Nulls(); ns > 0 {
		return fmt.Sprintf("[Object %d×null] with %d members", ns, len(o.Members))
	}
	return fmt.Sprintf("Object with %d members", len(o.Members))
}

func (s *Summary) object(o *Object) error {
	fmt.Fprintf(s.w, "%s:\n", ObjectLabel(o))
	nms := make([]string, 0, len(o.Members))
	for a := range o.Members {
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
		m := o.Members[a]
		fmt.Fprintf(s.w, "%s#%-2d \"%s\" ", pf, i+1, a)
		if m.Occurence < o.Count {
			fmt.Fprintf(s.w, "optional (%d/%d %.0f%%)",
				m.Occurence,
				o.Count,
				100*float64(m.Occurence)/float64(o.Count),
			)
		} else {
			fmt.Fprintf(s.w, "mandatory (%d×)", m.Occurence)
		}
		fmt.Println(":")
		s.tree.Descend()
		if err := s.printIndet(m.Ded, true); err != nil {
			return err
		}
		s.tree.Ascend(1)
	}
	s.tree.Ascend(1)
	return nil
}

func ArrayLabel(a *Array) string {
	var lens string
	if a.minLen == a.maxLen {
		lens = strconv.Itoa(a.minLen)
	} else {
		lens = fmt.Sprintf("%d..%d", a.minLen, a.maxLen)
	}
	if ns := a.Nulls(); ns > 0 {
		return fmt.Sprintf("[Array %d×null] of %s", ns, lens)
	}
	return fmt.Sprintf("Array of %s", lens)
}

func (s *Summary) array(a *Array) error {
	fmt.Fprintf(s.w, "%s:\n", ArrayLabel(a))
	s.tree.Descend()
	defer s.tree.Ascend(1)
	return s.printIndet(a.Elem, true)
}

func UnionLabel(u *Union) string {
	return fmt.Sprintf("Union of %d types", len(u.Variants))
}

func (s *Summary) union(u *Union) error {
	fmt.Fprintf(s.w, "%s:\n", UnionLabel(u))
	s.tree.Descend()
	for i, d := range u.Variants {
		if err := s.printIndet(d, i == len(u.Variants)-1); err != nil {
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
