package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"git.fractalqb.de/fractalqb/jsum"
	"git.fractalqb.de/fractalqb/tetrta"
	"gopkg.in/yaml.v3"
)

var (
	cfg = jsum.Config{
		DedupBool:   jsum.DedupBoolFalse | jsum.DedupBoolTrue,
		DedupNumber: jsum.DedpuNumberIntFloat | jsum.DedupNumberNeg,
		DedupString: jsum.DedupStringEmpty,
	}
	fTreeStyle           = "draw"
	fStrMax              = 6
	fTypes               bool
	fArgs, fOut, fSchema string
)

const (
	envJsumTree    = "JSUM_TREE"
	envJsumStrings = "JSUM_STRINGS"
)

func init() {
	if v, ok := os.LookupEnv(envJsumTree); ok {
		fTreeStyle = v
	}
	if v, ok := os.LookupEnv(envJsumStrings); ok {
		if n, err := strconv.Atoi(v); err == nil {
			fStrMax = n
		}
	}
}

func usage() {
	w := flag.CommandLine.Output()
	fmt.Print("Generate a summary from example JSON or YAML files.\n\n")
	fmt.Fprintln(w, `Usage: jsum [flags] <JSON/YAML file>...
FLAGS:`)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.StringVar(&fTreeStyle, "tree", fTreeStyle,
		"Select style for tree printing from: ascii, draw, items; Env: "+envJsumTree+"\n")
	flag.IntVar(&fStrMax, "strings", fStrMax,
		"Max number of strings values to print per property; Env: "+envJsumStrings+"\n")
	flag.BoolVar(&fTypes, "types", fTypes,
		"Find reused types (experimental)")
	flag.StringVar(&fArgs, "a", fArgs,
		"Read args from file ('-' reads from stdin)")
	flag.StringVar(&fOut, "o", fOut,
		"Print summary to file ('-' writes to stdout)")
	flag.StringVar(&fSchema, "schema", fSchema,
		"Generate JSON Schema file")
	flag.Parse()

	var (
		scm        jsum.Deducer = jsum.NewUnknown(&cfg)
		samples, n int
		err        error
	)
	switch {
	case fArgs == "-":
		scm, samples = readArgs(os.Stdin, scm)
	case fArgs != "":
		scm, samples = readArgsFile(fArgs, scm)
	case len(flag.Args()) > 0:
		for _, arg := range flag.Args() {
			if scm, n, err = readFile(arg, scm); err != nil {
				log.Fatal(err)
			}
			samples += n
		}
	default:
		dec := json.NewDecoder(os.Stdin)
		scm, samples = read(dec, scm)
	}

	if fOut == "" && fSchema == "" {
		newBrowser(scm, samples).run()
	} else {
		var w io.Writer = os.Stdout
		if fOut != "" && fOut != "-" {
			if tmp, err := os.Create(fOut); err != nil {
				log.Fatal(err)
			} else {
				defer tmp.Close()
				w = tmp
			}
		}
		var tstyle *tetrta.TreeStyle
		switch fTreeStyle {
		case "a", "ascii":
			tstyle = tetrta.ASCIITree()
		case "d", "draw":
			tstyle = tetrta.BoxDrawTree()
		case "i", "items":
			tstyle = tetrta.ItemTree()
		}
		head := fmt.Sprintf("Deduced from %d samples:", samples)
		fmt.Println(head)
		fmt.Println(strings.Repeat("=", len(head)))
		sum := jsum.NewSummary(w, &jsum.SummaryConfig{
			TreeStyle: tstyle,
			StringMax: fStrMax,
		})

		if fSchema != "" {
			scm := scm.JSONSchema()
			f, err := os.Create(fSchema)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			enc := json.NewEncoder(f)
			enc.SetIndent("", "   ")
			if err := enc.Encode(scm); err != nil {
				log.Fatal(err)
			}
		}

		if err := sum.Print(scm); err != nil {
			log.Fatal(err)
		}
		if fTypes {
			dedup := make(jsum.DedupHash)
			scm.Hash(dedup)
			tdefs := dedup.ReusedTypes()
			fmt.Fprintf(w, "\nFound %d distinct types\n", len(tdefs))
			for _, def := range tdefs {
				head = fmt.Sprintf("\nOccurs %d times:", len(def.Copies())+1)
				fmt.Println(head)
				fmt.Println(strings.Repeat("-", len(head)-1))
				sum.Print(def)
			}
		}
	}
}

func readArgsFile(file string, scm jsum.Deducer) (jsum.Deducer, int) {
	r, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()
	return readArgs(r, scm)
}

func readArgs(r io.Reader, scm jsum.Deducer) (_ jsum.Deducer, samples int) {
	var (
		n   int
		err error
	)
	scn := bufio.NewScanner(r)
	for scn.Scan() {
		file := scn.Text()
		if scm, n, err = readFile(file, scm); err != nil {
			log.Fatal(err)
		}
		samples += n
	}
	return scm, samples
}

type decoder interface{ Decode(any) error }

func read(dec decoder, d jsum.Deducer) (jsum.Deducer, int) {
	samples := 0
	for {
		var jv any
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

func readFile(name string, d jsum.Deducer) (_ jsum.Deducer, n int, err error) {
	rd, err := os.Open(name)
	if err != nil {
		return nil, 0, err
	}
	defer rd.Close()
	log.Println("read file", name)
	switch filepath.Ext(name) {
	case ".yml", ".yaml":
		dec := yaml.NewDecoder(rd)
		d, n = read(dec, d)
		return d, n, nil
	}
	dec := json.NewDecoder(rd)
	d, n = read(dec, d)
	return d, n, nil
}
