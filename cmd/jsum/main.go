package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"git.fractalqb.de/fractalqb/jsum"
)

var (
	keepEnum = jsum.KeepJsonEnum{
		Default: true,
		PerType: map[jsum.JsonType]jsum.EnumTest{
			jsum.JsonBool:   jsum.KeepBoolEnum,
			jsum.JsonString: jsum.KeepStringEnum(50).Test,
			jsum.JsonNumber: jsum.KeepNumberEnum(20).Test,
		},
	}
	cfg = jsum.Config{
		AsEnum:     keepEnum.Test,
		NumberHash: jsum.NumberHashIntFloat,
	}
	indentStr = ". "
	fEnums    bool
)

func read(rd io.Reader, d jsum.Deducer) (jsum.Deducer, int) {
	samples := 0
	dec := json.NewDecoder(rd)
	for {
		var jv interface{}
		err := dec.Decode(&jv)
		if err == io.EOF {
			return d, samples
		} else if err != nil {
			log.Fatal(err)
		}
		if err != nil {
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
	flag.StringVar(&indentStr, "indent", indentStr, "Indentation string")
	flag.BoolVar(&fEnums, "enums", false, "Enable deduction of enums")
	flag.BoolVar(&keepEnum.Default, "keep-enum-default", true, "Keep enum by default")
	flag.Parse()
	if !fEnums {
		cfg.AsEnum = nil
	}
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
	fmt.Printf("Deduced from %d samples:\n", samples)
	sum := jsum.NewSummary(os.Stdout)
	sum.Indent = indentStr
	if err := sum.Print(scm); err != nil {
		log.Fatal(err)
	}
	dedup := make(jsum.DedupHash)
	scm.Hash(dedup)
	fmt.Printf("Found %d distinct types\n", len(dedup))
	// for _, scms := range dedup {
	// 	fmt.Printf("- %d\n", len(scms))
	// }
}
