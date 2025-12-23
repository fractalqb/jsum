/*
A tool to analyse the structure of JSON from a set of example JSON values.
Copyright (C) 2025  Marcus Perlick

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

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

func numsLabel(b *dedBase) string {
	if b.Null > 0 {
		p := int(math.Round(100 * float64(b.Null) / float64(b.Count)))
		if p == 0 {
			return fmt.Sprintf("[null:%d/%d]", b.Null, b.Count)
		}
		return fmt.Sprintf("[null:%d/%d %d%%]", b.Null, b.Count, p)
	}
	return fmt.Sprintf("[%d×]", b.Count)
}

func AnyLabel(ded *Any) string { return "Any " + numsLabel(&ded.dedBase) }

func UnknownLabel(ded *Unknown) string { return "??? " + numsLabel(&ded.dedBase) }

func InvalidLabel(n Invalid) string {
	return fmt.Sprintf("<INVALID: %s>", n.error.Error())
}

func StringLabel(ded *String) string {
	var sb strings.Builder
	minLen, maxLen := math.MaxInt, 0
	for str := range ded.Stats {
		l := len(str)
		if l < minLen {
			minLen = l
		}
		if l > maxLen {
			maxLen = l
		}
	}
	sb.WriteString("String")
	switch ded.Format {
	case DateTimeFormat:
		fmt.Fprint(&sb, " format=date-time")
	default:
		if minLen == maxLen {
			fmt.Fprintf(&sb, " len:%d", minLen)
		} else {
			fmt.Fprintf(&sb, " len:%d..%d", minLen, maxLen)
		}
	}
	fmt.Fprintf(&sb, " distinct:%d %s", len(ded.Stats), numsLabel(&ded.dedBase))
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

func BoolLabel(ded *Boolean) string {
	return fmt.Sprintf("Boolean true:%d / false:%d %s",
		ded.TrueNo,
		ded.FalseNo,
		numsLabel(&ded.dedBase),
	)
}

func (s *Summary) bool(n *Boolean) error {
	fmt.Fprintln(s.w, BoolLabel(n))
	return nil
}

func NumberLabel(ded *Number) string {
	var sum string
	if ded.IsFloat {
		if ded.Min == ded.Max {
			if ded.HasFrac {
				sum = fmt.Sprintf("Number = %f ", ded.Min)
			} else {
				sum = fmt.Sprintf("Number = %f 0-fracs ", ded.Min)
			}
		} else {
			if ded.HasFrac {
				sum = fmt.Sprintf("Number %f–%f ", ded.Min, ded.Max)
			} else {
				sum = fmt.Sprintf("Number %f–%f 0-fracs ", ded.Min, ded.Max)
			}
		}
	} else {
		mi, ma := int64(ded.Min), int64(ded.Max)
		if mi == ma {
			sum = fmt.Sprintf("Integer = %d ", mi)
		} else {
			sum = fmt.Sprintf("Integer %d–%d ", mi, ma)
		}
	}
	return sum + numsLabel(&ded.dedBase)
}

func (s *Summary) number(n *Number) error {
	fmt.Fprintln(s.w, NumberLabel(n))
	return nil
}

func ObjectLabel(ded *Object) string {
	return fmt.Sprintf("Object with %d members %s",
		len(ded.Members),
		numsLabel(&ded.dedBase),
	)
}

func (s *Summary) object(o *Object) error {
	fmt.Fprintf(s.w, "%s\n", ObjectLabel(o))
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
		fmt.Fprintln(s.w, ":")
		s.tree.Descend()
		if err := s.printIndet(m.Ded, true); err != nil {
			return err
		}
		s.tree.Ascend(1)
	}
	s.tree.Ascend(1)
	return nil
}

func ArrayLabel(ded *Array) string {
	var lens string
	if ded.MinLen == ded.MaxLen {
		lens = strconv.Itoa(ded.MinLen)
	} else {
		lens = fmt.Sprintf("%d..%d", ded.MinLen, ded.MaxLen)
	}
	return fmt.Sprintf("Array of %s %s", lens, numsLabel(&ded.dedBase))
}

func (s *Summary) array(a *Array) error {
	fmt.Fprintf(s.w, "%s:\n", ArrayLabel(a))
	s.tree.Descend()
	defer s.tree.Ascend(1)
	return s.printIndet(a.Elem, true)
}

func UnionLabel(u *Union) string {
	return fmt.Sprintf("Union of %d types %s",
		len(u.Variants),
		numsLabel(&u.dedBase),
	)
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
