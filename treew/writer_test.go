package treew

import (
	"fmt"
	"io"
	"os"
)

func ExampleWriter() {
	w := NewWriter(os.Stdout, nil)
	io.WriteString(w, "This is some sort of preamble")
	io.WriteString(w.Descend().First(nil), "Node 1")
	io.WriteString(w.Descend().Next(nil), `Node 1.1
This is more additional text
for node 1.1.`)
	io.WriteString(w.Last(nil), "Node 1.2")
	fmt.Fprint(w.Descend().Next(nil), "Node 1.2.1")
	fmt.Fprint(w, "By the way, this is what I want to add to 1.2.1!")
	fmt.Fprint(w.Last(nil), "Node 1.2.2")
	fmt.Fprint(w.Descend().Last(nil), "Node 1.2.2.1")
	fmt.Fprint(w.Ascend(3).Last(nil), "Node 3")
	fmt.Fprint(w.Ascend(1), "And finally a footer")
	// Output:
	// This is some sort of preamble
	// ┌─ Node 1
	// │  ├─ Node 1.1
	// │  │  This is more additional text
	// │  │  for node 1.1.
	// │  └─ Node 1.2
	// │     ├─ Node 1.2.1
	// │     │  By the way, this is what I want to add to 1.2.1!
	// │     └─ Node 1.2.2
	// │        └─ Node 1.2.2.1
	// └─ Node 3
	// And finally a footer
}
