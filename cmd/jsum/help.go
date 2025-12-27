package main

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"unicode/utf8"

	"git.fractalqb.de/fractalqb/eloc/must"
	"github.com/rivo/tview"
)

//go:embed help
var help embed.FS

type helpView struct {
	tview.TextView
	txtRows, txtCols int
}

func helpViews() (res []*helpView) {
	hfs := must.RetCtx(help.ReadDir("help")).Msg("list help texsts")
	for _, hf := range hfs {
		res = append(res, newHelpView(filepath.Join("help", hf.Name())))
	}
	res = slices.Clip(res)
	for i, h := range res {
		h.SetBorder(true).SetTitle(fmt.Sprintf(" Help %d/%d (🠈 🠊 ESC) ", i+1, len(res)))
	}
	return res
}

func newHelpView(name string) *helpView {
	txt := must.RetCtx(fs.ReadFile(help, name)).Msg("help file")
	scn := bufio.NewScanner(bytes.NewReader(txt))
	res := &helpView{
		TextView: *tview.NewTextView(),
		txtRows:  2,
	}
	for scn.Scan() {
		res.txtRows++
		res.txtCols = max(res.txtCols, utf8.RuneCount(scn.Bytes()))
	}
	res.txtCols += 2
	res.SetSize(res.txtRows, res.txtCols).SetText(string(txt))
	return res
}
