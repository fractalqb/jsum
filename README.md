This project has been migrated to [Codeberg](https://codeberg.org/fractalqb/jsum).

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
   
   `go install git.fractalqb.de/fractalqb/jsum/cmd/jsum@latest`

## Whats next
- JSUM is currently in a "enough to give me insights" state. It is programmed in
  a rather casual manner and probably will remain like that. It did the job… 

- In principle the algorithm would also work in a streaming mode. But the
  current implementation depends on complete JSON values to be read into a Go
  `any`. Quite standard for Go! It does not really hurt for the files
  I'm currently working with, but… – This could be improved

- There are no tools to control the level of detail. – This cloud be improved
  
- For more insight it (IMHO) would be very helpful to detect schema patters that
  occur in different places. I.e. automatically detect common types that are
  reused in the schema. – Already have some infrastructure for that but its WiP
