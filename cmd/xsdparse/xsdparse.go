package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"aqwari.net/xml/xmltree"
	"aqwari.net/xml/xsd"
)

var (
	TargetNS = flag.String("ns", "", "Namespace of schea to print")
)

func main() {
	log.SetFlags(0)
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatalf("Usage: %s [-ns xmlns] file.xsd ...", os.Args[0])
	}

	docs := make([][]byte, 0, flag.NArg())

	for _, filename := range flag.Args() {
		if data, err := ioutil.ReadFile(filename); err != nil {
			log.Fatal(err)
		} else {
			docs = append(docs, data)
		}
	}

	filterSchema := make(map[string]struct{})
	for _, doc := range xsd.StandardSchema {
		root, err := xmltree.Parse(doc)
		if err != nil {
			// should never happen
			panic(err)
		}
		filterSchema[root.Attr("", "targetNamespace")] = struct{}{}
	}

	norm, err := xsd.Normalize(docs...)
	if err != nil {
		log.Fatal(err)
	}

	selected := make([]*xmltree.Element, 0, len(norm))
	for _, root := range norm {
		tns := root.Attr("", "targetNamespace")
		if *TargetNS != "" && *TargetNS == tns {
			selected = append(selected, root)
		} else if _, ok := filterSchema[tns]; !ok {
			selected = append(selected, root)
		}
	}

	for _, root := range selected {
		fmt.Printf("%s\n", xmltree.MarshalIndent(root, "", "  "))
	}
}
