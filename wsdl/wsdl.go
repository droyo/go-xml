// Package wsdl parses Web Service Definition Language documents.
//
// The wsdl package implements a parser for wsdl documents.
package wsdl

import (
	"encoding/xml"

	"aqwari.net/xml/xmltree"
	"aqwari.net/xml/xsd"
)

const (
	wsdlNS    = "http://schemas.xmlsoap.org/wsdl/"
	soapNS    = "http://schemas.xmlsoap.org/wsdl/soap/"
	httpNS    = "http://schemas.xmlsoap.org/wsdl/http/"
	mimeNS    = "http://schemas.xmlsoap.org/wsdl/mime/"
	soapencNS = "http://schemas.xmlsoap.org/soap/encoding/"
	soapenvNS = "http://schemas.xmlsoap.org/soap/envelope/"
	xsiNS     = "http://www.w3.org/2000/10/XMLSchema-instance"
	xsdNS     = "http://www.w3.org/2000/10/XMLSchema"
)

// A Definition contains all information necessary to generate Go code
// from a wsdl document.
type Definition struct {
	Doc        string
	Types      []xsd.Schema
	Operations []Operation
}

// An Operation describes an RPC call that can be made against
// the remote server. Its inputs, outputs and transport information
// are parsed from the WSDL definition. Many Operations can be
// defined for a single endpoint.
type Operation struct {
	Name            xml.Name
	Inputs, Outputs []xml.Name
}

// Parse reads the first WSDL definition from data.
func Parse(data []byte) (*Definition, error) {
	var def Definition
	types, err := xsd.Parse(data)
	if err != nil {
		return nil, err
	}

	def.Types = types
	root, err := xmltree.Parse(data)
	if err != nil {
		return nil, err
	}

	for _, el := range root.Search(wsdlNS, "message") {
	}
	for _, el := range root.Search(wsdlNS, "portType") {
	}
	for _, el := range root.Search(wsdlNS, "binding") {
	}
	for _, el := range root.Search(wsdlNS, "service") {
	}
	return &def, nil
}
