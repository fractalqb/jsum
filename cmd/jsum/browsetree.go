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

package main

import (
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"

	"git.fractalqb.de/fractalqb/jsum"
	"github.com/rivo/tview"
)

type lbFmtFunc func(string) string
type searchBuild = map[string][]*tview.TreeNode

func browseTree(scm jsum.Deducer, lff lbFmtFunc, srb searchBuild) (res *tview.TreeNode) {
	switch scm := scm.(type) {
	case *jsum.String:
		res = browseString(scm, lff, srb)
	case *jsum.Number:
		res = browseNumber(scm, lff)
	case *jsum.Object:
		res = browseObject(scm, lff, srb)
	case *jsum.Boolean:
		res = browseBool(scm, lff)
	case *jsum.Array:
		res = browseArray(scm, lff, srb)
	case *jsum.Union:
		res = browseUnion(scm, lff, srb)
	case *jsum.Any:
		res = browseAny(scm, lff)
	case *jsum.Unknown:
		res = browseUnknown(scm, lff)
	case jsum.Invalid:
		res = browseInvalid(scm, lff)
	default:
		res = tview.NewTreeNode(lff(fmt.Sprintf("Unsupported deducer: %T", scm)))
		res.SetSelectable(false)
	}
	return res
}

func browseString(scm *jsum.String, lff lbFmtFunc, srb searchBuild) (res *tview.TreeNode) {
	fldNode := stdFolder(lff(jsum.StringLabel(scm)))
	res = tview.NewTreeNode(fldNode.label(false))
	initRef(res, &fldNode, scm)
	var maxCount int
	for _, n := range scm.Stats {
		maxCount = max(maxCount, n)
	}
	strs := slices.Collect(maps.Keys(scm.Stats))
	sort.Strings(strs)
	if maxCount > 1 {
		width := len(strconv.Itoa(maxCount))
		form := fmt.Sprintf(" %%%dd × %%#v", width)
		for _, s := range strs {
			sn := tview.NewTreeNode(fmt.Sprintf(form, scm.Stats[s], s))
			res.AddChild(sn)
			srb[s] = append(srb[s], sn)
		}
	} else {
		for _, s := range strs {
			sn := tview.NewTreeNode(fmt.Sprintf(" %#v", s))
			res.AddChild(sn)
			srb[s] = append(srb[s], sn)
		}
	}
	res.SetExpanded(false)
	fldNode.fold(res)
	return res
}

func browseNumber(scm *jsum.Number, lff lbFmtFunc) (res *tview.TreeNode) {
	res = tview.NewTreeNode(" " + lff(jsum.NumberLabel(scm)))
	initRef(res, nil, scm)
	return res
}

func browseObject(scm *jsum.Object, lff lbFmtFunc, srb searchBuild) (res *tview.TreeNode) {
	fldNode := stdFolder(lff(jsum.ObjectLabel(scm)))
	res = tview.NewTreeNode(fldNode.label(true))
	initRef(res, &fldNode, scm)
	nms := slices.Collect(maps.Keys(scm.Members))
	slices.Sort(nms)
	var sb strings.Builder
	for _, a := range nms {
		m := scm.Members[a]
		if m.Occurence < scm.Count {
			fmt.Fprintf(&sb, "[::b]\"%s\"[::-] [blue::]optional[-::] (%d/%d %.0f%%)",
				a,
				m.Occurence,
				scm.Count,
				100*float64(m.Occurence)/float64(scm.Count),
			)
		} else {
			fmt.Fprintf(&sb, "[::bu]\"%s\"[::-] [orange::]mandatory[-::] (%d×)", a, m.Occurence)
		}
		sb.WriteByte(':')
		fldMember := folder{
			text:  sb.String(),
			open:  "┯ ",
			close: "━ ",
		}
		sb.Reset()
		nm := tview.NewTreeNode(fldMember.label(true))
		initRef(nm, &fldMember, a)
		nm.AddChild(browseTree(m.Ded, noFmt, srb))
		fldMember.fold(nm)
		res.AddChild(nm)
		srb[a] = append(srb[a], nm)
	}
	fldNode.fold(res)
	return res
}

func browseBool(scm *jsum.Boolean, lff lbFmtFunc) (res *tview.TreeNode) {
	res = tview.NewTreeNode(" " + lff(jsum.BoolLabel(scm)))
	initRef(res, nil, scm)
	return res
}

func browseArray(scm *jsum.Array, lff lbFmtFunc, srb searchBuild) (res *tview.TreeNode) {
	res = tview.NewTreeNode("┬ " + lff(jsum.ArrayLabel(scm)) + ":")
	initRef(res, nil, scm)
	res.AddChild(browseTree(scm.Elem, noFmt, srb))
	return res
}

func browseUnion(scm *jsum.Union, lff lbFmtFunc, srb searchBuild) (res *tview.TreeNode) {
	fldNode := stdFolder(lff(jsum.UnionLabel(scm)) + ":")
	res = tview.NewTreeNode(fldNode.label(true))
	initRef(res, nil, scm)
	for _, d := range scm.Variants {
		res.AddChild(browseTree(d, noFmt, srb))
	}
	fldNode.fold(res)
	return res
}

func browseAny(scm *jsum.Any, lff lbFmtFunc) (res *tview.TreeNode) {
	res = tview.NewTreeNode(" " + lff(jsum.AnyLabel(scm)))
	initRef(res, nil, scm)
	return res
}

func browseUnknown(scm *jsum.Unknown, lff lbFmtFunc) (res *tview.TreeNode) {
	res = tview.NewTreeNode(" " + lff(jsum.UnknownLabel(scm)))
	initRef(res, nil, scm)
	return res
}

func browseInvalid(scm jsum.Invalid, lff lbFmtFunc) (res *tview.TreeNode) {
	res = tview.NewTreeNode(" " + lff(jsum.InvalidLabel(scm)))
	initRef(res, nil, scm)
	return res
}

type ref struct {
	fld  *folder
	info any
}

func initRef(n *tview.TreeNode, f *folder, info any) {
	n.SetReference(ref{f, info})
}

func getFolder(n *tview.TreeNode) *folder {
	tmp := n.GetReference()
	if tmp == nil {
		return nil
	}
	r, ok := tmp.(ref)
	if ok {
		return r.fld
	}
	return nil
}

func getInfo(n *tview.TreeNode) any {
	tmp := n.GetReference()
	if tmp == nil {
		return nil
	}
	r, ok := tmp.(ref)
	if ok {
		return r.info
	}
	return nil
}
