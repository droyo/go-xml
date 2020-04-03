package main

import (
	"log"
	"os"

	"github.com/CognitoIQ/go-xml/wsdlgen"
)

func main() {
	if err := wsdlgen.GenCLI(os.Args[1:]...); err != nil {
		log.Fatal(err)
	}
}
