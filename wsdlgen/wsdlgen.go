// Package wsdlgen generates Go source code from wsdl documents.
//
// The wsdlgen package generates Go source for calling the various
// methods defined in a WSDL (Web Service Definition Language) document.
// The generated Go source is self-contained, with no dependencies on
// non-standard packages.
//
// Code generation for the wsdlgen package can be configured by using
// the provided Option functions.
package wsdlgen

import (
	"encoding/xml"
	"errors"
	"fmt"
	"go/ast"
	"io/ioutil"
	"strings"

	"aqwari.net/xml/internal/gen"
	"aqwari.net/xml/wsdl"
	"aqwari.net/xml/xsdgen"
)

// Types conforming to the Logger interface can receive information about
// the code generation process.
type Logger interface {
	Printf(format string, v ...interface{})
}

type printer struct {
	*Config
	code *xsdgen.Code
	wsdl *wsdl.Definition
	file *ast.File
}

// Provides aspects about an RPC call to the template for the function
// bodies.
type opArgs struct {
	// formatted with appropriate variable names
	input, output []string

	// URL to send request to
	Address string

	// POST or GET
	Method string

	// Name of the method to call
	MsgName xml.Name

	// if we're returning individual values, these slices
	// are in an order matching the input/output slices.
	InputName, OutputName xml.Name
	InputFields           []field
	OutputFields          []field
}

// struct members. Need to export the fields for our template
type field struct {
	Name, Type string
	XMLName    xml.Name

	// This refers to the name of the value to assign to this field
	// in the argument list. Empty for return values.
	InputArg string
}

// GenAST creates a Go source file containing type and method declarations
// that can be used to access the service described in the provided set of wsdl
// files.
func (cfg *Config) GenAST(files ...string) (*ast.File, error) {
	if len(files) == 0 {
		return nil, errors.New("must provide at least one file name")
	}
	if cfg.pkgName == "" {
		cfg.pkgName = "ws"
	}
	if cfg.pkgHeader == "" {
		cfg.pkgHeader = fmt.Sprintf("Package %s", cfg.pkgName)
	}
	docs := make([][]byte, 0, len(files))
	for _, filename := range files {
		if data, err := ioutil.ReadFile(filename); err != nil {
			return nil, err
		} else {
			cfg.debugf("read %s", filename)
			docs = append(docs, data)
		}
	}

	cfg.debugf("parsing WSDL file %s", files[0])
	def, err := wsdl.Parse(docs[0])
	if err != nil {
		return nil, err
	}
	cfg.verbosef("building xsd type whitelist from WSDL")
	cfg.registerXSDTypes(def)

	cfg.verbosef("generating type declarations from xml schema")
	code, err := cfg.xsdgen.GenCode(docs...)
	if err != nil {
		return nil, err
	}

	cfg.verbosef("generating function definitions from WSDL")
	return cfg.genAST(def, code)
}

func (cfg *Config) genAST(def *wsdl.Definition, code *xsdgen.Code) (*ast.File, error) {
	file, err := code.GenAST()
	if err != nil {
		return nil, err
	}
	file.Name = ast.NewIdent(cfg.pkgName)
	file = gen.PackageDoc(file, cfg.pkgHeader, "\n", def.Doc)
	p := &printer{
		Config: cfg,
		wsdl:   def,
		file:   file,
		code:   code,
	}
	return p.genAST()
}

func (p *printer) genAST() (*ast.File, error) {
	p.addHelpers()
	for _, port := range p.wsdl.Ports {
		if err := p.port(port); err != nil {
			return nil, err
		}
	}
	return p.file, nil
}

func (p *printer) port(port wsdl.Port) error {
	for _, operation := range port.Operations {
		if err := p.operation(port, operation); err != nil {
			return err
		}
	}
	return nil
}

func (p *printer) operation(port wsdl.Port, op wsdl.Operation) error {
	input, ok := p.wsdl.Message[op.Input]
	if !ok {
		return fmt.Errorf("unknown input message type %s", op.Input.Local)
	}
	output, ok := p.wsdl.Message[op.Output]
	if !ok {
		return fmt.Errorf("unknown output message type %s", op.Output.Local)
	}
	params, err := p.opArgs(port.Address, port.Method, input, output)
	if err != nil {
		return err
	}

	fn := gen.Func(p.xsdgen.NameOf(op.Name)).
		Comment(op.Doc).
		Receiver("c *Client").
		Args(params.input...).
		BodyTmpl(`
			var input struct {
				XMLName struct{} `+"`"+`xml:"{{.InputName.Space}} {{.InputName.Local}}"`+"`"+`
				{{ range .InputFields -}}
				{{.Name}} {{.Type}} `+"`"+`xml:"{{.XMLName.Space}} {{.XMLName.Local}}"`+"`"+`
				{{ end -}}
			}
			
			{{- range .InputFields }}
			input.{{.Name}} = {{.Type}}({{.InputArg}})
			{{ end }}
			
			var output struct {
				XMLName struct{} `+"`"+`xml:"{{.OutputName.Space}} {{.OutputName.Local}}"`+"`"+`
				{{ range .OutputFields -}}
				{{.Name}} {{.Type}} `+"`"+`xml:"{{.XMLName.Space}} {{.XMLName.Local}}"`+"`"+`
				{{ end -}}
			}
			
			err := c.do({{.Method|printf "%q"}}, {{.Address|printf "%q"}}, &input, &output)
			
			{{ if .OutputFields -}}
			return {{ range .OutputFields }}output.{{.Name}}, {{ end }} err
			{{- else -}}
			return err
			{{- end -}}
		`, params).
		Returns(params.output...)
	if decl, err := fn.Decl(); err != nil {
		return err
	} else {
		p.file.Decls = append(p.file.Decls, decl)
	}
	return nil
}

// The xsdgen package generates private types for some builtin
// types. These types should be hidden from the user and converted
// on the fly.
func exposeType(typ string) string {
	switch typ {
	case "xsdDate", "xsdTime", "xsdDateTime", "gDay",
		"gMonth", "gMonthDay", "gYear", "gYearMonth":
		return "time.Time"
	case "hexBinary", "base64Binary":
		return "[]byte"
	case "idrefs", "nmtokens", "notation", "entities":
		return "[]string"
	}
	return typ
}

func (p *printer) opArgs(addr, method string, input, output wsdl.Message) (opArgs, error) {
	var args opArgs
	args.Address = addr
	args.Method = method
	args.InputName = input.Name
	for _, part := range input.Parts {
		typ := p.code.NameOf(part.Type)
		inputType := exposeType(typ)
		vname := gen.Sanitize(part.Name)
		args.input = append(args.input, vname+" "+inputType)
		args.InputFields = append(args.InputFields, field{
			Name:     strings.Title(part.Name),
			Type:     typ,
			XMLName:  xml.Name{p.wsdl.TargetNS, part.Name},
			InputArg: vname,
		})
	}
	args.OutputName = output.Name
	for _, part := range output.Parts {
		typ := p.code.NameOf(part.Type)
		outputType := exposeType(typ)
		args.output = append(args.output, outputType)
		args.OutputFields = append(args.OutputFields, field{
			Name:    strings.Title(part.Name),
			Type:    typ,
			XMLName: xml.Name{p.wsdl.TargetNS, part.Name},
		})
	}
	// NOTE(droyo) if we decide to name our return values,
	// we have to change this too.
	args.output = append(args.output, "error")

	return args, nil
}

// To keep our output small (as possible), we only generate type
// declarations for the types that are named in the WSDL definition.
func (cfg *Config) registerXSDTypes(def *wsdl.Definition) {
	xmlns := make(map[string]struct{})
	// Some schema may list messages that are not used by any
	// ports, so we have to be thorough.
	for _, port := range def.Ports {
		for _, op := range port.Operations {
			for _, name := range []xml.Name{op.Input, op.Output} {
				if msg, ok := def.Message[name]; !ok {
					cfg.logf("ERROR: No message def found for %s", name.Local)
				} else {
					for _, part := range msg.Parts {
						xmlns[part.Type.Space] = struct{}{}
						cfg.xsdgen.Option(xsdgen.AllowType(part.Type))
					}
				}
			}
		}
	}
	namespaces := make([]string, 0, len(xmlns))
	for ns := range xmlns {
		namespaces = append(namespaces, ns)
	}
	cfg.xsdgen.Option(xsdgen.Namespaces(namespaces...))
}
