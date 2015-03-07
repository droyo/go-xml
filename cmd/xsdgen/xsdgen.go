package main

import (
	"log"
	"os"

	"aqwari.net/xml/xsdgen"
)

func main() {
	log.SetFlags(0)
	var cfg xsdgen.Config
	cfg.Option(xsdgen.DefaultOptions...)
	cfg.Option(xsdgen.ErrorLog(log.New(os.Stderr, "", 0), 1))

	if err := cfg.Generate(os.Args[1:]...); err != nil {
		log.Fatal(err)
	}
}
