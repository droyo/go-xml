package main

import (
	"log"
	"os"

	"aqwari.net/xml/wsdlgen"
)

func main() {
	if err := wsdlgen.GenCLI(os.Args[1:]...); err != nil {
		log.Fatal(err)
	}
}
