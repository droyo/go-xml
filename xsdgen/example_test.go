package xsdgen_test

import (
	"log"
	"os"

	"aqwari.net/xml/xsdgen"
)

func ExampleConfig_Generate() {
	var cfg xsdgen.Config
	cfg.Option(
		xsdgen.IgnoreAttributes("id", "href", "offset"),
		xsdgen.IgnoreElements("comment"),
		xsdgen.PackageName("webapi"),
		xsdgen.ReplaceAllNames("_", ""),
		xsdgen.HandleSOAPArrayType(),
		xsdgen.SOAPArrayAsSlice(),
	)
	if err := cfg.Generate("webapi.xsd", "deps/soap11.xsd"); err != nil {
		log.Fatal(err)
	}
}

func ExampleErrorLog() {
	var cfg xsdgen.Config
	cfg.Option(xsdgen.ErrorLog(log.New(os.Stderr, "", 0), 1))
	if err := cfg.Generate("file.wsdl"); err != nil {
		log.Fatal(err)
	}
}

func ExampleReplaceAllNames() {
	var cfg xsdgen.Config
	cfg.Option(xsdgen.ReplaceAllNames("ArrayOf(.*)", "${1}Array"))
	if err := cfg.Generate("ws.wsdl"); err != nil {
		log.Fatal(err)
	}
}
