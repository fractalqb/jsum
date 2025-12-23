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
	"log"
	"slices"
	"strings"
	"unicode/utf8"

	"git.fractalqb.de/fractalqb/jsum"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"
)

const (
	pgTree            = "tree"
	pgHelp            = "help-%d"
	pgStat            = "stat"
	pgSearchTerm      = "search-term"
	pgSearchMatch     = "search-match"
	slsWidth, slsRows = 38, 12
	statDefault       = "Press ? for help"
)

type searchTerm struct {
	txt   string
	nodes []*tview.TreeNode
}

type searchSource []searchTerm

func (src searchSource) Len() int            { return len(src) }
func (src searchSource) String(i int) string { return src[i].txt }

type browser struct {
	app *tview.Application

	data      *tview.TreeNode
	tree      *tview.TreeView
	path      *tview.TextView
	stat      *tview.TextView
	srchTerm  *tview.TextArea
	srchMatch *tview.Table
	help      []*helpView
	pgsMain   *tview.Pages
	pgsFoot   *tview.Pages

	searchSrc searchSource
	search    struct {
		matches    []*searchTerm
		matchNodes int
		match      *searchTerm
		matchNode  int
		term       string
	}
}

func newBrowser(scm jsum.Deducer, samples int) *browser {
	srb := make(searchBuild)
	data := browseTree(scm, noFmt, srb)
	b := &browser{
		data:      data,
		tree:      tview.NewTreeView().SetRoot(data).SetCurrentNode(data),
		path:      tview.NewTextView(),
		stat:      tview.NewTextView().SetText(statDefault).SetDynamicColors(true),
		srchTerm:  tview.NewTextArea(),
		srchMatch: tview.NewTable().SetSelectable(true, false),
		help:      helpViews(),
		pgsMain:   tview.NewPages(),
		pgsFoot:   tview.NewPages(),

		searchSrc: make(searchSource, 0, len(srb)),
	}

	b.tree.SetInputCapture(b.treeInput)
	b.tree.SetChangedFunc(func(node *tview.TreeNode) {
		b.path.SetText(nodePath(b.tree.GetPath(node)))
	})

	b.path.SetTextStyle(tcell.StyleDefault.Reverse(true).Bold(true))

	b.stat.SetTextStyle(tcell.StyleDefault.Reverse(true))

	b.srchMatch.
		SetSelectionChangedFunc(b.matchSelected).
		SetInputCapture(b.searchMatchInput).
		SetBorder(true).
		SetTitle(" Search matches ")

	for i, help := range b.help {
		help.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyRight:
				b.pgsMain.SendToBack(fmt.Sprintf(pgHelp, i))
				j := (i + 1) % len(b.help)
				b.pgsMain.SendToFront(fmt.Sprintf(pgHelp, j))
			case tcell.KeyLeft:
				b.pgsMain.SendToBack(fmt.Sprintf(pgHelp, i))
				j := i - 1
				if j < 0 {
					j = len(b.help) - 1
				}
				b.pgsMain.SendToFront(fmt.Sprintf(pgHelp, j))
			default:
				b.pgsMain.SendToFront(pgTree)
			}
			return nil
		})
		b.pgsMain.AddPage(
			fmt.Sprintf(pgHelp, i),
			modal(help, "c", help.txtCols, help.txtRows),
			true, true,
		)
	}

	b.pgsMain.
		AddPage(pgSearchMatch, modal(b.srchMatch, "R", slsWidth, slsRows+2), true, true).
		AddPage(pgTree, b.tree, true, true)

	b.srchTerm.
		SetTextStyle(tcell.StyleDefault.Reverse(true)).
		SetChangedFunc(b.searchChange).
		SetInputCapture(b.searchTermInput)

	b.pgsFoot.
		AddPage(pgSearchTerm, b.srchTerm, true, true).
		AddPage(pgStat, b.stat, true, true)

	for s, ns := range srb {
		b.searchSrc = append(b.searchSrc, searchTerm{s, ns})
	}
	return b
}

func (b *browser) run() {
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(b.path, 1, 0, false).
		AddItem(b.pgsMain, 0, 1, true).
		AddItem(b.pgsFoot, 1, 0, false)
	b.app = tview.NewApplication().
		SetRoot(flex, true).
		SetFocus(b.tree)
	err := b.app.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func (b *browser) treeInput(evt *tcell.EventKey) *tcell.EventKey {
	switch evt.Key() {
	case tcell.KeyLeft:
		if evt.Modifiers()&tcell.ModShift == 0 {
			return evt
		}
		p := b.tree.GetPath(b.tree.GetCurrentNode())
		if l := len(p); l > 1 {
			b.tree.SetCurrentNode(p[l-2])
			b.path.SetText(nodePath(p[:l-1]))
		}
		return nil
	}
	switch evt.Rune() {
	case 'r':
		siblSetExpand(b.tree, true)
		return nil
	case 'm':
		siblSetExpand(b.tree, false)
		return nil
	case 'R':
		treeSetExpand(b.tree.GetCurrentNode(), true)
		return nil
	case 'M':
		treeSetExpand(b.tree.GetCurrentNode(), false)
		return nil
	case 's':
		b.srchTerm.SetLabel("Search:")
		if txt := b.srchTerm.GetText(); txt != "" {
			b.srchTerm.Select(0, len(txt))
		}
		b.pgsFoot.SendToFront(pgSearchTerm)
		b.srchMatch.Select(0, 0)
		b.pgsMain.SendToFront(pgSearchMatch)
		b.app.SetFocus(b.srchTerm)
	case '?':
		b.pgsMain.SendToFront(fmt.Sprintf(pgHelp, 0))
	}
	return evt
}

func (b *browser) searchTermInput(evt *tcell.EventKey) *tcell.EventKey {
	switch evt.Key() {
	case tcell.KeyEnter:
		b.srchMatch.Select(0, 0)
		b.app.SetFocus(b.srchMatch)
		b.pgsFoot.SendToFront(pgStat)
		return nil
	case tcell.KeyESC:
		b.stat.SetText(statDefault)
		b.pgsMain.SendToFront(pgTree)
		b.pgsFoot.SendToFront(pgStat)
		b.app.SetFocus(b.tree)
		return nil
	}
	return evt
}

func (b *browser) searchMatchInput(evt *tcell.EventKey) *tcell.EventKey {
	prevNode := func() {
		if b.search.match != nil {
			if b.search.matchNode--; b.search.matchNode < 0 {
				b.search.matchNode = len(b.search.match.nodes) - 1
			}
			b.visitNode(b.search.match.nodes[b.search.matchNode])
			b.srchStat()
		}
	}
	nextNode := func() {
		if b.search.match != nil {
			if b.search.matchNode++; b.search.matchNode >= len(b.search.match.nodes) {
				b.search.matchNode = 0
			}
			b.visitNode(b.search.match.nodes[b.search.matchNode])
			b.srchStat()
		}
	}
	switch evt.Key() {
	case tcell.KeyESC:
		b.app.SetFocus(b.srchTerm)
		b.pgsFoot.SendToFront(pgSearchTerm)
		return nil
	case tcell.KeyLeft:
		if evt.Modifiers()&tcell.ModShift == 0 {
			return evt
		}
		prevNode()
		return nil
	case tcell.KeyRight:
		if evt.Modifiers()&tcell.ModShift == 0 {
			return evt
		}
		nextNode()
		return nil
	}
	switch evt.Rune() {
	case 'H':
		prevNode()
		return nil
	case 'L':
		nextNode()
		return nil
	}
	return evt
}

func (b *browser) matchSelected(row, column int) {
	if row >= len(b.search.matches) {
		return
	}
	b.search.match = b.search.matches[row]
	b.search.matchNode = 0
	if ref := b.srchMatch.GetCell(row, 1).GetReference(); ref != nil {
		b.search.term = ref.(string)
	}
	b.visitNode(b.search.match.nodes[0])
	b.srchStat()

	b.srchMatch.SetTitle(fmt.Sprintf(" %d/%d terms • %d matches ",
		row+1,
		len(b.search.matches),
		b.search.matchNodes,
	))
}

func (b *browser) srchStat() {
	if len(b.search.match.nodes) > 1 {
		b.stat.SetText(fmt.Sprintf("%d/%d: [::b]%s[::-] (select: S-🠈/H, S-🠊/L)",
			b.search.matchNode+1,
			len(b.search.match.nodes),
			b.search.term,
		))
	} else {
		b.stat.SetText(fmt.Sprintf("1/1: [::b]%s[::-]", b.search.term))
	}
}

func (b *browser) visitNode(n *tview.TreeNode) {
	path := b.tree.GetPath(n)
	for _, n := range path {
		if !n.IsExpanded() {
			if f := getFolder(n); f != nil {
				n.SetText(f.label(true))
			}
			n.SetExpanded(true)
		}
	}
	b.tree.SetCurrentNode(n)
}

func (b *browser) searchChange() {
	pat := b.srchTerm.GetText()
	if utf8.RuneCountInString(pat) < 1 {
		return
	}
	matches := fuzzy.FindFrom(b.srchTerm.GetText(), b.searchSrc)
	b.srchMatch.Clear()
	b.search.matches = b.search.matches[:0]
	b.search.matchNodes = 0
	for i, m := range matches {
		match := &b.searchSrc[m.Index]
		b.search.matches = append(b.search.matches, match)
		b.srchMatch.SetCell(i, 0, tview.NewTableCell(
			fmt.Sprintf("%d×", len(match.nodes)),
		).SetAlign(tview.AlignRight))
		b.srchMatch.SetCell(i, 1, tview.NewTableCell(matchStr(&m)).SetReference(m.Str))
		b.search.matchNodes += len(match.nodes)
	}
	b.srchMatch.Select(0, 0)
}

func matchStr(m *fuzzy.Match) string {
	hi := false
	var sb strings.Builder
	for i, r := range ([]rune)(m.Str) {
		if slices.Contains(m.MatchedIndexes, i) {
			if !hi {
				sb.WriteString("[::b]")
				hi = true
			}
		} else if hi {
			sb.WriteString("[::-]")
			hi = false
		}
		sb.WriteRune(r)
	}
	if hi {
		sb.WriteString("[::-]")
	}
	return sb.String()
}

type folder struct {
	open, close, text string
}

func stdFolder(text string) folder {
	return folder{
		open:  "[lightgreen]▼[-] ",
		close: "[lightgreen]▶[-] ",
		text:  text,
	}
}

func (f *folder) label(open bool) string {
	if open {
		return f.open + f.text
	}
	return f.close + f.text
}

func (f *folder) fold(n *tview.TreeNode) {
	n.SetSelectable(true)
	n.SetSelectedFunc(func() {
		n.SetExpanded(!n.IsExpanded())
		n.SetText(f.label(n.IsExpanded()))
	})
}

func noFmt(s string) string { return s }

func nodePath(p []*tview.TreeNode) string {
	var sb strings.Builder
	sb.WriteByte('$')
	for _, n := range p {
		info := getInfo(n)
		if info == nil {
			continue
		}
		switch ref := info.(type) {
		case string:
			fmt.Fprintf(&sb, ".%s", ref)
		case *jsum.Array:
			sb.WriteString("[*]")
		}
	}
	return sb.String()
}

func treeSetExpand(n *tview.TreeNode, exp bool) {
	if f := getFolder(n); f != nil {
		n.SetExpanded(exp)
		n.SetText(f.label(exp))
	}
	for _, c := range n.GetChildren() {
		treeSetExpand(c, exp)
	}
}

func siblSetExpand(b *tview.TreeView, exp bool) {
	path := b.GetPath(b.GetCurrentNode())
	l := len(path)
	switch l {
	case 0:
		return
	case 1:
		if f := getFolder(path[0]); f != nil {
			path[0].SetExpanded(exp)
			path[0].SetText(f.label(exp))
		}
	default:
		parent := path[len(path)-2]
		for _, c := range parent.GetChildren() {
			if f := getFolder(c); f != nil {
				c.SetExpanded(exp)
				c.SetText(f.label(exp))
			}
		}
	}
}

func modal(p tview.Primitive, a string, width, height int) tview.Primitive {
	switch a {
	case "R":
		return tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(p, width, 1, true)
	default:
		return tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(p, height, 1, true).
				AddItem(nil, 0, 1, false),
				width, 1, true,
			).AddItem(nil, 0, 1, false)
	}
}
