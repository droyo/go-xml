[![GoDoc](https://godoc.org/aqwari.net/xml?status.svg)](https://godoc.org/aqwari.net/xml) [![Build Status](https://travis-ci.org/droyo/go-xml.svg?branch=master)](https://travis-ci.org/droyo/go-xml)

This repository contains a collection of Go packages for working
with XML, with the ultimate goal of enabling code generation based
on XML documents.

- The `xmltree` package converts xml documents to a tree data
  structure, and provides convenient methods for manipulating and
  searching through that tree.
- The `xsd` package implements a parser for XML Schema. It takes
  some liberties from the specification, and would need some work for
  use as a validator, but it handles type inheritance and XML namespaces
  in a relatively sane way.
- The `xsdgen` package provides a customizable code generator that
  generates Go type declarations and marshal/unmarshal methods for
  an XML Schema.
- The `xsdgen` command generates Go code with default settings and
  is suitable for use with `go generate`.

This code is still very rough around the edges, but I have succesfully
used it to generate type declarations for some pretty complex XML
schema from an Apache Axis application. There are github issues
opened for missing functionality.
