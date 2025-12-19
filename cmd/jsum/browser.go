package main

import (
	"fmt"
	"log"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"

	"git.fractalqb.de/fractalqb/jsum"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	// github.com/sahilm/fuzzy
)

const (
	pgTree                = "tree"
	pgHelp                = "help"
	helpWidth, helpHeight = 46, 12
)

type browser struct {
	data *tview.TreeNode
	tree *tview.TreeView
	help *tview.TextArea
	pags *tview.Pages
	path *tview.TextView
	stat *tview.TextView
}

func newBrowser(scm jsum.Deducer, samples int) *browser {
	data := browseTree(scm, func(s string) string {
		return fmt.Sprintf("%d × %s", samples, s)
	})
	b := &browser{
		data: data,
		tree: tview.NewTreeView().SetRoot(data).SetCurrentNode(data),
		help: tview.NewTextArea().SetSize(helpHeight, helpWidth).SetText(helpText, false),
		pags: tview.NewPages(),
		path: tview.NewTextView(),
		stat: tview.NewTextView().SetText("Press ? for help"),
	}

	b.tree.SetInputCapture(b.treeInput)
	b.tree.SetChangedFunc(func(node *tview.TreeNode) {
		b.path.SetText(nodePath(b.tree.GetPath(node)))
	})

	b.help.SetBorder(true).SetTitle(" Help (ESC to close) ")
	b.help.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		b.pags.SendToFront(pgTree)
		return nil
	})

	b.path.SetTextStyle(tcell.StyleDefault.Reverse(true).Bold(true))

	b.stat.SetTextStyle(tcell.StyleDefault.Reverse(true))

	b.pags.AddPage(pgHelp, modal(b.help, helpWidth, helpHeight), true, true).
		AddPage(pgTree, b.tree, true, true)
	return b
}

func (b *browser) run() {
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(b.path, 1, 0, false).
		AddItem(b.pags, 0, 1, true).
		AddItem(b.stat, 1, 0, false)
	err := tview.NewApplication().
		SetRoot(flex, true).
		SetFocus(b.tree).
		Run()
	if err != nil {
		log.Fatal(err)
	}
}

func (b *browser) treeInput(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
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
	case '?':
		b.pags.SendToFront(pgHelp)
	}
	return event
}

type lbFmtFunc func(string) string

func browseTree(scm jsum.Deducer, lff lbFmtFunc) (res *tview.TreeNode) {
	switch scm := scm.(type) {
	case *jsum.String:
		res = browseString(scm, lff)
	case *jsum.Number:
		res = browseNumber(scm, lff)
	case *jsum.Object:
		res = browseObject(scm, lff)
	case *jsum.Boolean:
		res = browseBool(scm, lff)
	case *jsum.Array:
		res = browseArray(scm, lff)
	case *jsum.Union:
		res = browseUnion(scm, lff)
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

func browseString(scm *jsum.String, lff lbFmtFunc) (res *tview.TreeNode) {
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
		}
	} else {
		for _, s := range strs {
			sn := tview.NewTreeNode(fmt.Sprintf(" %#v", s))
			res.AddChild(sn)
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

func browseObject(scm *jsum.Object, lff lbFmtFunc) (res *tview.TreeNode) {
	fldNode := stdFolder(lff(jsum.ObjectLabel(scm)))
	res = tview.NewTreeNode(fldNode.label(true))
	initRef(res, &fldNode, scm)
	nms := slices.Collect(maps.Keys(scm.Members))
	slices.Sort(nms)
	var sb strings.Builder
	for _, a := range nms {
		fmt.Fprintf(&sb, "[::b]\"%s\"[::-] ", a)
		m := scm.Members[a]
		if m.Occurence < scm.Count {
			fmt.Fprintf(&sb, "optional (%d/%d %.0f%%)",
				m.Occurence,
				scm.Count,
				100*float64(m.Occurence)/float64(scm.Count),
			)
		} else {
			fmt.Fprintf(&sb, "mandatory (%d×)", m.Occurence)
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
		nm.AddChild(browseTree(m.Ded, noFmt))
		fldMember.fold(nm)
		res.AddChild(nm)
	}
	fldNode.fold(res)
	return res
}

func browseBool(scm *jsum.Boolean, lff lbFmtFunc) (res *tview.TreeNode) {
	res = tview.NewTreeNode(" " + lff(jsum.BoolLabel(scm)))
	initRef(res, nil, scm)
	return res
}

func browseArray(scm *jsum.Array, lff lbFmtFunc) (res *tview.TreeNode) {
	res = tview.NewTreeNode("┬ " + lff(jsum.ArrayLabel(scm)) + ":")
	initRef(res, nil, scm)
	res.AddChild(browseTree(scm.Elem, noFmt))
	return res
}

func browseUnion(scm *jsum.Union, lff lbFmtFunc) (res *tview.TreeNode) {
	fldNode := stdFolder(lff(jsum.UnionLabel(scm)) + ":")
	res = tview.NewTreeNode(fldNode.label(true))
	initRef(res, nil, scm)
	for _, d := range scm.Variants {
		res.AddChild(browseTree(d, noFmt))
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

type folder struct {
	open, close, text string
}

func stdFolder(text string) folder {
	return folder{
		open:  "▼ ",
		close: "▶ ",
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
	if len(path) < 2 {
		return
	}
	parent := path[len(path)-2]
	for _, c := range parent.GetChildren() {
		if f := getFolder(c); f != nil {
			c.SetExpanded(exp)
			c.SetText(f.label(exp))
		}
	}
}

func modal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}

const helpText = `j, ↓, →          : Move down by one node
k, ↑, ←          : Move up by one node
g, home          : Move to the top
G, end           : Move to the bottom
J                : Move down one level
K                : Move up one level
Ctrk-F, page down: Move down by one page
Ctrl-B, page up  : Move up by one page
m / M            : Fold siblings / subtree
r / R            : Unfold siblings / subtree`
