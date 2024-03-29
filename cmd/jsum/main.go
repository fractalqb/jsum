package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"git.fractalqb.de/fractalqb/jsum"
	"git.fractalqb.de/fractalqb/jsum/treew"
)

var (
	cfg = jsum.Config{
		DedupBool:   jsum.DedupBoolFalse | jsum.DedupBoolTrue,
		DedupNumber: jsum.DedpuNumberIntFloat | jsum.DedupNumberNeg,
		DedupString: jsum.DedupStringEmpty,
	}
	fTreeStyle string
	fStrMax    = 12
	fTypes     bool
)

func read(rd io.Reader, d jsum.Deducer) (jsum.Deducer, int) {
	samples := 0
	dec := json.NewDecoder(rd)
	for {
		var jv interface{}
		err := dec.Decode(&jv)
		switch {
		case err == io.EOF:
			return d, samples
		case err != nil:
			log.Fatal(err)
		}
		d = d.Example(jv)
		if i, ok := d.(jsum.Invalid); ok {
			log.Fatal(i)
		}
		samples++
	}
}

func readFile(name string, d jsum.Deducer) (jsum.Deducer, int) {
	rd, _ := os.Open(name)
	defer rd.Close()
	log.Println("read file", name)
	return read(rd, d)
}

func main() {
	flag.StringVar(&fTreeStyle, "tree", "draw", "Select style for tree drawing from: ascii, draw, items")
	flag.IntVar(&fStrMax, "strings", fStrMax, "Max number of strings values to show per property")
	flag.BoolVar(&fTypes, "types", fTypes, "Find reused types (experimental)")
	flag.Parse()
	var scm jsum.Deducer = jsum.NewUnknown(&cfg)
	var samples, n int
	if len(flag.Args()) > 0 {
		for _, arg := range flag.Args() {
			scm, n = readFile(arg, scm)
			samples += n
		}
	} else {
		scm, samples = read(os.Stdin, scm)
	}
	var tstyle *treew.Style
	switch fTreeStyle {
	case "a", "ascii":
		tstyle = treew.ASCIIStyle()
	case "d", "draw":
		tstyle = treew.BoxDrawStyle()
	case "i", "items":
		tstyle = treew.ItemStyle()
	}
	head := fmt.Sprintf("Deduced from %d samples:", samples)
	fmt.Println(head)
	fmt.Println(strings.Repeat("=", len(head)))
	sum := jsum.NewSummary(os.Stdout, &jsum.SummaryConfig{
		TreeStyle: tstyle,
		StringMax: fStrMax,
	})
	if err := sum.Print(scm); err != nil {
		log.Fatal(err)
	}
	if fTypes {
		dedup := make(jsum.DedupHash)
		scm.Hash(dedup)
		tdefs := dedup.ReusedTypes()
		fmt.Printf("\nFound %d distinct types\n", len(tdefs))
		for _, def := range tdefs {
			head = fmt.Sprintf("\nOccurs %d times:", len(def.Copies())+1)
			fmt.Println(head)
			fmt.Println(strings.Repeat("-", len(head)-1))
			sum.Print(def)
		}
	}
}
