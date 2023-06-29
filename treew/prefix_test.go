package treew

import "fmt"

func exampleTree(pf *Prefix) {
	fmt.Println("This is some sort of preamble")
	fmt.Printf("%s%s\n", pf.Descend().First(nil), "Node 1")
	fmt.Printf("%s%s\n", pf.Descend().Next(nil), "Node 1.1")
	fmt.Printf("%s%s\n", pf.Cont(nil), "This is more additional text")
	fmt.Printf("%s%s\n", pf.Cont(nil), "for node 1.1.")
	fmt.Printf("%s%s\n", pf.Last(nil), "Node 1.2")
	fmt.Printf("%s%s\n", pf.Descend().Next(nil), "Node 1.2.1")
	fmt.Printf("%s%s\n", pf.Cont(nil), "By the way, this is what I want to add to 1.2.1!")
	fmt.Printf("%s%s\n", pf.Last(nil), "Node 1.2.2")
	fmt.Printf("%s%s\n", pf.Descend().Last(nil), "Node 1.2.1.1")
	fmt.Printf("%s%s\n", pf.Ascend(3).Last(nil), "Node 3")
	fmt.Printf("%s%s\n", pf.Ascend(1).Next(nil), "And finally a footer")
}

func ExamplePrefix() {
	var pf Prefix
	exampleTree(&pf)
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
	// │        └─ Node 1.2.1.1
	// └─ Node 3
	// And finally a footer
}

func ExamplePrefix_ascii() {
	pf := Prefix{Style: ASCIIStyle()}
	exampleTree(&pf)
	// Output:
	// This is some sort of preamble
	// ,-- Node 1
	// |   +-- Node 1.1
	// |   |   This is more additional text
	// |   |   for node 1.1.
	// |   `-- Node 1.2
	// |       +-- Node 1.2.1
	// |       |   By the way, this is what I want to add to 1.2.1!
	// |       `-- Node 1.2.2
	// |           `-- Node 1.2.1.1
	// `-- Node 3
	// And finally a footer
}

func ExamplePrefix_items() {
	pf := Prefix{Style: ItemStyle()}
	exampleTree(&pf)
	// Output:
	// This is some sort of preamble
	// - Node 1
	//   - Node 1.1
	//     This is more additional text
	//     for node 1.1.
	//   - Node 1.2
	//     - Node 1.2.1
	//       By the way, this is what I want to add to 1.2.1!
	//     - Node 1.2.2
	//       - Node 1.2.1.1
	// - Node 3
	// And finally a footer
}
