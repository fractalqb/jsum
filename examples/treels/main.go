package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"git.fractalqb.de/fractalqb/jsum/treew"
)

func main() {
	for _, d := range os.Args[1:] {
		lsdir(d)
	}
}

func lsdir(dir string) (err error) {
	depth := 0
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	tpf := treew.Prefix{}
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, _ error) error {
		path, _ = filepath.Rel(dir, path)
		pws := strings.Split(path, string(filepath.Separator))
		if l := len(pws); l > depth {
			tpf.Descend()
			depth = l
		} else if l < depth {
			tpf.Ascend(depth - l)
			depth = l
		}
		i, err := d.Info()
		if err != nil {
			return err
		}
		fmt.Fprintf(tw, "%s%v\t%d\t%s\t\n",
			tpf.Next(nil),
			d.Name(),
			i.Size(),
			i.Mode(),
		)
		return nil
	})
	tw.Flush()
	return err
}
