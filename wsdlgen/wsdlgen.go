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
	"aqwari.net/xml/xmltree"
	"aqwari.net/xml/xsd"
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

	SOAPAction string

	// Name of the method to call
	MsgName xml.Name

	// if we're returning individual values, these slices
	// are in an order matching the input/output slices.
	InputName, OutputName xml.Name
	InputFields           []field
	OutputFields          []field

	// If not "", inputs come in a wrapper struct
	InputType string

	// If not "", we return values in a wrapper struct
	ReturnType   string
	ReturnFields []field
}

// struct members. Need to export the fields for our template
type field struct {
	Name, Type string
	XMLName    xml.Name

	// If this is a wrapper struct for >InputThreshold arguments,
	// PublicType holds the type that we want to expose to the
	// user. For example, if the web service expects an xsdDate
	// to be sent to it, PublicType will be time.Time and a conversion
	// will take place before sending the request to the server.
	PublicType string

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
	cfg.debugf("procuring top-level element types")
	elementTypes, err := cfg.scanElementTypes(docs)
	if err != nil {
		return nil, err
	}
	cfg.verbosef("building xsd type whitelist from WSDL")
	cfg.registerXSDTypes(def, elementTypes)

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
	params, err := p.opArgs(port.Address, port.Method, op, input, output)
	if err != nil {
		return err
	}

	if params.InputType != "" {
		decls, err := gen.Snippets(params, `
			type {{.InputType}} struct {
			{{ range .InputFields -}}
				{{.Name}} {{.PublicType}}
			{{ end -}}
			}`,
		)
		if err != nil {
			return err
		}
		p.file.Decls = append(p.file.Decls, decls...)
	}
	if params.ReturnType != "" {
		decls, err := gen.Snippets(params, `
			type {{.ReturnType}} struct {
			{{ range .ReturnFields -}}
				{{.Name}} {{.Type}}
			{{ end -}}
			}`,
		)
		if err != nil {
			return err
		}
		p.file.Decls = append(p.file.Decls, decls...)
	}
	fn := gen.Func(p.xsdgen.NameOf(op.Name)).
		Comment(op.Doc).
		Receiver("c *Client").
		Args(params.input...).
		BodyTmpl(`
			var input struct {
				XMLName struct{} `+"`"+`xml:"{{.MsgName.Space}} {{.MsgName.Local}}"`+"`"+`
				Args struct {
					{{ range .InputFields -}}
					{{.Name}} {{.Type}} `+"`"+`xml:"{{.XMLName.Space}} {{.XMLName.Local}}"`+"`"+`
					{{ end -}}
				}`+"`xml:\"{{.InputName.Space}} {{.InputName.Local}}\"`"+`
			}
			
			{{- range .InputFields }}
			input.Args.{{.Name}} = {{.Type}}({{.InputArg}})
			{{ end }}
			
			var output struct {
				XMLName struct{} `+"`"+`xml:"{{.MsgName.Space}} {{.MsgName.Local}}"`+"`"+`
				Args struct {
					{{ range .OutputFields -}}
					{{.Name}} {{.Type}} `+"`"+`xml:"{{.XMLName.Space}} {{.XMLName.Local}}"`+"`"+`
					{{ end -}}
				}`+"`xml:\"{{.OutputName.Space}} {{.OutputName.Local}}\"`"+`
			}
			
			err := c.do({{.Method|printf "%q"}}, {{.Address|printf "%q"}}, {{.SOAPAction|printf "%q"}}, &input, &output)
			
			{{ if .OutputFields -}}
			return {{ range .OutputFields }}{{.Type}}(output.Args.{{.Name}}), {{ end }} err
			{{- else if .ReturnType -}}
			var result {{ .ReturnType }}
			{{ range .ReturnFields -}}
			result.{{.Name}} = {{.Type}}(output.Args.{{.InputArg}})
			{{ end -}}
			return result, err
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

// Generates mapping of top-level element names to the names
// of their types.
func (cfg *Config) scanElementTypes(docs [][]byte) (map[xml.Name]xml.Name, error) {
	const schemaNS = "http://www.w3.org/2001/XMLSchema"
	result := make(map[xml.Name]xml.Name)

	trees, err := xsd.Normalize(docs...)
	if err != nil {
		return nil, err
	}
	for _, root := range trees {
		container := xmltree.Element{Children: []xmltree.Element{*root}}
		for _, schema := range container.Search(schemaNS, "schema") {
			tns := schema.Attr("", "targetNamespace")
			for _, el := range schema.Children {
				if el.Name != (xml.Name{schemaNS, "element"}) {
					continue
				}
				xmlname := el.ResolveDefault(el.Attr("", "name"), tns)
				xmltype := el.Resolve(el.Attr("", "type"))

				if xmlname.Local != "" && xmltype.Local != "" {
					result[xmlname] = xmltype
				}
			}
		}
	}
	return result, nil
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

func (p *printer) getPartType(part wsdl.Part) (string, error) {
	if part.Type.Local != "" {
		return p.code.NameOf(part.Type), nil
	}
	if part.Element.Local != "" {
		doc, ok := p.code.DocType(part.Element.Space)
		if !ok {
			return "", fmt.Errorf("part %s: could not lookup element %v",
				part.Name, part.Element)
		}
		for _, el := range doc.Elements {
			if el.Name == part.Element {
				return p.code.NameOf(xsd.XMLName(el.Type)), nil
			}
		}
	}
	return "", fmt.Errorf("part %s has no element or type", part.Name)
}

func (p *printer) opArgs(addr, method string, op wsdl.Operation, input, output wsdl.Message) (opArgs, error) {
	var args opArgs
	args.Address = addr
	args.Method = method
	args.SOAPAction = op.SOAPAction
	args.MsgName = op.Name
	args.InputName = input.Name
	for _, part := range input.Parts {
		typ, err := p.getPartType(part)
		if err != nil {
			return args, err
		}
		inputType := exposeType(typ)
		vname := gen.Sanitize(part.Name)
		if vname == typ {
			vname += "_"
		}
		args.input = append(args.input, vname+" "+inputType)
		args.InputFields = append(args.InputFields, field{
			Name:       strings.Title(part.Name),
			Type:       typ,
			PublicType: exposeType(typ),
			XMLName:    xml.Name{p.wsdl.TargetNS, part.Name},
			InputArg:   vname,
		})
	}
	if len(args.input) > p.maxArgs {
		args.InputType = strings.Title(args.InputName.Local)
		args.input = []string{"v " + args.InputName.Local}
		for i, v := range input.Parts {
			args.InputFields[i].InputArg = "v." + strings.Title(v.Name)
		}
	}
	args.OutputName = output.Name
	for _, part := range output.Parts {
		typ, err := p.getPartType(part)
		if err != nil {
			return args, err
		}
		outputType := exposeType(typ)
		args.output = append(args.output, outputType)
		args.OutputFields = append(args.OutputFields, field{
			Name:    strings.Title(part.Name),
			Type:    typ,
			XMLName: xml.Name{p.wsdl.TargetNS, part.Name},
		})
	}
	if len(args.output) > p.maxReturns {
		args.ReturnType = strings.Title(args.OutputName.Local)
		args.ReturnFields = make([]field, len(args.OutputFields))
		for i, v := range args.OutputFields {
			args.ReturnFields[i] = field{
				Name:     v.Name,
				Type:     exposeType(v.Type),
				InputArg: v.Name,
			}
		}
		args.output = []string{args.ReturnType}
	}
	// NOTE(droyo) if we decide to name our return values,
	// we have to change this too.
	args.output = append(args.output, "error")

	return args, nil
}

// To keep our output small (as possible), we only generate type
// declarations for the types that are named in the WSDL definition.
func (cfg *Config) registerXSDTypes(def *wsdl.Definition, elementTypes map[xml.Name]xml.Name) {
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
						var typeName xml.Name
						if part.Type.Space != "" {
							typeName = part.Type
						}
						if part.Element.Space != "" {
							if t, ok := elementTypes[part.Element]; !ok {
								cfg.verbosef("could not determine type for part %v", part)
								typeName = part.Element
							} else {
								typeName = t
							}
							xmlns[part.Element.Space] = struct{}{}
						}
						xmlns[typeName.Space] = struct{}{}
						cfg.xsdgen.Option(xsdgen.AllowType(typeName))
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
