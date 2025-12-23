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
	"bufio"
	"encoding/json"
	"errors"
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
	fTreeStyle = "draw"
	fStrMax    = 6
	fTypes     bool
	fArgs      string
	fOut       string
	fState     string
	fSchema    string
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
	fmt.Fprintln(w, `Generate a summary from example JSON or YAML files.

  Usage: jsum [flags] <JSON/YAML file>|'-'...

Without printing and schema generation, JSUM will launch an interactive browser
for the summary in the terminal.

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
	flag.StringVar(&fState, "state", fState,
		`Keep deduced schema in state file.
This can be used for incremental refinement or simply to browse without
analysing the examples again.`)
	flag.Parse()

	var (
		scm     = loadState(fState, &cfg)
		samples int
		err     error
	)

	switch {
	case fArgs == "-":
		scm, samples = readArgs(os.Stdin, scm)
	case fArgs != "":
		scm, samples = readArgsFile(fArgs, scm)
	}

	for _, arg := range flag.Args() {
		var n int
		if arg == "-" {
			dec := json.NewDecoder(os.Stdin)
			scm, n = read(dec, scm)
		} else if scm, n, err = readFile(arg, scm); err != nil {
			log.Fatal(err)
		}
		samples += n
	}

	log.Printf("read %d sample records", samples)
	if fState != "" && samples > 0 {
		writeState(fState, scm)
	}

	if fOut == "" && fSchema == "" {
		log.Print("no output, no schema generation – staring interactive browser")
		newBrowser(scm, samples).run()
	} else if fOut != "" {
		var w io.Writer = os.Stdout
		if fOut != "-" {
			log.Print("writing jsum summary to stdout")
			if tmp, err := os.Create(fOut); err != nil {
				log.Fatal(err)
			} else {
				defer tmp.Close()
				w = tmp
			}
			log.Println("writing jsum summary to", fOut)
		} else {
			log.Print("writing jsum summary to stdout")
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
				head := fmt.Sprintf("\nOccurs %d times:", len(def.Copies())+1)
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
		jt := jsum.JsonTypeOf(jv)
		if !jt.Valid() {
			log.Fatalf("no deduced type for %T", jv)
		}
		d = d.Example(jv, jt)
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

func loadState(name string, cfg *jsum.Config) jsum.Deducer {
	if name == "" {
		return jsum.NewUnknown(cfg)
	}
	log.Println("read state", name)
	f, err := os.Open(name)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return jsum.NewUnknown(cfg)
	case err != nil:
		log.Fatal(err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		log.Fatal(err)
	}
	var sio jsum.StateIO
	state, err := sio.ReadState(f, cfg, stat.Size())
	if err != nil {
		log.Fatal(err)
	}
	return state
}

func writeState(name string, scm jsum.Deducer) {
	log.Println("write state", name)
	f, err := os.Create(name + "~")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	var sio jsum.StateIO
	if err := sio.WriteState(f, scm); err != nil {
		log.Fatal(err)
	}
	log.Println("stat write dedup", sio.StrDup, "/", sio.StrCount)
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stat(name); err == nil {
		ext := filepath.Ext(name)
		base := filepath.Base(name[:len(name)-len(ext)])
		if f, err := os.CreateTemp(filepath.Dir(name), base+"-*"+ext); err != nil {
			log.Fatal(err)
		} else {
			base = f.Name()
			f.Close()
		}
		os.Rename(name, base)
	}
	os.Rename(name+"~", name)
}
