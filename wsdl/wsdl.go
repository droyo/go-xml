// Package wsdl parses Web Service Definition Language documents.
package wsdl

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"aqwari.net/xml/xmltree"
)

const (
	wsdlNS    = "http://schemas.xmlsoap.org/wsdl/"
	soapNS    = "http://schemas.xmlsoap.org/wsdl/soap/"
	soap12NS  = "http://schemas.xmlsoap.org/wsdl/soap12/"
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
	Doc      string
	Ports    []Port
	Message  map[xml.Name]Message
	TargetNS string
}

func (def *Definition) String() string {
	var buf bytes.Buffer
	fmt.Fprintln(&buf, "Namespace: ", def.TargetNS)
	for _, port := range def.Ports {
		fmt.Fprintf(&buf, "Port %s %s\n", port.Name, port.Address)
		for _, op := range port.Operations {
			input := def.Message[op.Input]
			output := def.Message[op.Output]
			fmt.Fprintf(&buf, "\t%s(%s) -> %s\n",
				op.Name.Local, &input, &output)
		}
	}
	return buf.String()
}

// A Message is a set of zero or more parameters for WSDL
// operations.
type Message struct {
	Name  xml.Name
	Parts []Part
}

func (m *Message) String() string {
	parts := make([]string, 0, len(m.Parts))
	for _, p := range m.Parts {
		parts = append(parts, fmt.Sprintf("%s", p.Type.Local))
	}
	return fmt.Sprintf("%s", strings.Join(parts, ", "))
}

// A Part describes the name and type of a parameter to pass
// to a WSDL endpoint.
type Part struct {
	Name string
	Type xml.Name
}

// An Operation describes an RPC call that can be made against
// the remote server. Its inputs, outputs and transport information
// are parsed from the WSDL definition. Many Operations can be
// defined for a single endpoint.
type Operation struct {
	Doc           string
	Name          xml.Name
	Input, Output xml.Name
}

// A Port describes a set of RPCs and the address to reach them.
type Port struct {
	Name, Address, Method string
	Operations            []Operation
}

// Collect multiple <documentation> children into a newline-separate string
func documentation(el *xmltree.Element) string {
	var docs [][]byte
	for _, doc := range el.Children {
		if doc.Name.Local != "documentation" {
			continue
		}
		docs = append(docs, doc.Content)
	}
	return string(bytes.Join(docs, []byte{'\n'}))
}

func parseMessages(targetNS string, root *xmltree.Element) map[xml.Name]Message {
	messages := make(map[xml.Name]Message)
	for _, msg := range root.Search(wsdlNS, "message") {
		var m Message
		m.Name = msg.ResolveDefault(msg.Attr("", "name"), targetNS)
		for _, part := range msg.Search(wsdlNS, "part") {
			p := Part{
				Name: part.Attr("", "name"),
				Type: part.Resolve(part.Attr("", "type")),
			}
			if len(p.Type.Local) == 0 {
				p.Type = part.Resolve(part.Attr("", "element"))
			}
			m.Parts = append(m.Parts, p)
		}
		messages[m.Name] = m
	}
	return messages
}

func parsePorts(targetNS string, root, svc *xmltree.Element) []Port {
	var ports []Port
	for _, port := range svc.Search(wsdlNS, "port") {
		var p Port
		p.Name = port.Attr("", "name")
		for _, addr := range port.Search(soapNS, "address") {
			p.Address = addr.Attr("", "location")
		}
		for _, addr := range port.Search(soap12NS, "address") {
			p.Address = addr.Attr("", "location")
		}
		p.Method = "POST"
		p.Method = parseMethod(targetNS, root, port)
		p.Operations = parseOperations(targetNS, root, port)
		ports = append(ports, p)
	}
	return ports
}

// The HTTP verb used for the set of operations bound to port, default POST
func parseMethod(targetNS string, root, port *xmltree.Element) string {
	binding := port.Resolve(port.Attr("", "binding"))
	predicate := func(el *xmltree.Element) bool {
		return el.Name.Space == wsdlNS &&
			el.Name.Local == "binding" &&
			el.ResolveDefault(el.Attr("", "name"), targetNS) == binding
	}
	for _, bind := range root.SearchFunc(predicate) {
		for _, httpbind := range bind.Search(httpNS, "binding") {
			verb := httpbind.Attr("", "verb")
			if len(verb) > 0 {
				return strings.ToUpper(verb)
			}
		}
	}
	return "POST"
}

func parseOperations(targetNS string, root, port *xmltree.Element) []Operation {
	var ops []Operation
	binding := port.Resolve(port.Attr("", "binding"))
	predicate := func(el *xmltree.Element) bool {
		return el.Name.Space == wsdlNS &&
			el.Name.Local == "binding" &&
			el.ResolveDefault(el.Attr("", "name"), targetNS) == binding
	}

	for _, bind := range root.SearchFunc(predicate) {
		portType := bind.Resolve(bind.Attr("", "type"))
		predicate := func(el *xmltree.Element) bool {
			return el.Name.Space == wsdlNS &&
				el.Name.Local == "portType" &&
				el.ResolveDefault(el.Attr("", "name"), targetNS) == portType
		}
		for _, pt := range root.SearchFunc(predicate) {
			for _, op := range pt.Search(wsdlNS, "operation") {
				var oper Operation
				oper.Doc = documentation(op)
				oper.Name = op.ResolveDefault(op.Attr("", "name"), targetNS)
				for _, input := range op.Search(wsdlNS, "input") {
					oper.Input = input.Resolve(input.Attr("", "message"))
				}
				for _, output := range op.Search(wsdlNS, "output") {
					oper.Output = output.Resolve(output.Attr("", "message"))
				}
				ops = append(ops, oper)
			}
		}
	}
	return ops
}

// Parse reads the first WSDL definition from data.
func Parse(data []byte) (*Definition, error) {
	var def Definition
	root, err := xmltree.Parse(data)
	if err != nil {
		return nil, err
	}

	def.TargetNS = root.Attr("", "targetNamespace")
	def.Message = parseMessages(def.TargetNS, root)

	for _, svc := range root.Search(wsdlNS, "service") {
		def.Doc = documentation(svc)
		def.Ports = parsePorts(def.TargetNS, root, svc)
	}
	return &def, nil
}
