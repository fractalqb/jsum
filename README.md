# JSON Summary

A tool to analyse the structure of JSON from a set of example JSON values.

## Rationale

To understand the _meaning_ of a set of JSON values it is often not enough to
have e.g. a JSON schema. I wanted a tool to give me a more detailed summary
of what is contained in some large files with JSON Lines in it. E.g.:

- Are the values of a number member all integer?
- What is the range of numbers in a number value?
- Does a string value only contain a small restricted set of values, i.e. could
  it be some sort of enumerated value?
- …etc

The tools I found were focused on JSON Schema, so I rolled my own…
 
## Package

1. On top level this Go package is a library that provides all the things to
   find the JSON summary. Use it with
   
   `go get git.fractalqb.de/fractalqb/jsum`
   
2. In the `cmd/jsum` directory you find a simple command line tool to generate
   summaries. Install it with
   
   `go install git.fractalqb.de/fractalqb/jsum/cmd/jsum`

## Whats next
- JSUM is currently in a "enough to give me the summary" state, it has a
  reasonable software design, but it breaks with many software engineering
  practices, first of all: tests. – This should be improved

- In principle the algorithm would also work in a streaming mode. But the
  current implementation depends on complete JSON values to be read into a Go
  `interface{}`. Quite standard for Go! It does not really hurt for the files
  I'm currently working with, but… – This could be improved

- There are no tools to control the level of detail _after_ the analysis run.
  Detail level currently can only be adjusted _during_ the analysis run. I expect
  it to be possible to handle both with some “unified” concept. – This cloud be
  improved
  
- For more insight it (IMHO) would be very helpful to detect schema patters that
  occur in different places. I.e. automatically detect common types that are
  reused in the schema. – Already have some infrastructure for that but its WiP