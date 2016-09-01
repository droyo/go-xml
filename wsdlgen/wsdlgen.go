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
	inputs, err := p.argList(op.Input, true)
	if err != nil {
		return err
	}
	outputs, err := p.argList(op.Output, false)
	if err != nil {
		return err
	}
	outputs = append(outputs, "error")
	fn := gen.Func(p.xsdgen.NameOf(op.Name)).
		Comment(op.Doc).
		Receiver("c *Client").
		Args(inputs...).
		Body("return nil").
		Returns(outputs...)
	if decl, err := fn.Decl(); err != nil {
		return err
	} else {
		p.file.Decls = append(p.file.Decls, decl)
	}
	return nil
}

func (p *printer) argList(message xml.Name, named bool) ([]string, error) {
	msg, ok := p.wsdl.Message[message]
	if !ok {
		return nil, fmt.Errorf("unknown message type %s", message.Local)
	}
	args := make([]string, 0, len(msg.Parts))
	for _, part := range msg.Parts {
		arg := p.code.NameOf(part.Type)
		if named {
			arg = part.Name + " " + arg
		}
		args = append(args, arg)
	}
	return args, nil
}

// To keep our output small (as possible), we only generate type
// declarations for the types that are named in the WSDL definition.
func (cfg *Config) registerXSDTypes(def *wsdl.Definition) {
	// Some schema may list messages that are not used by any
	// ports, so we have to be thorough.
	for _, port := range def.Ports {
		for _, op := range port.Operations {
			for _, name := range []xml.Name{op.Input, op.Output} {
				if msg, ok := def.Message[name]; !ok {
					cfg.logf("ERROR: No message def found for %s", name.Local)
				} else {
					for _, part := range msg.Parts {
						cfg.xsdgen.Option(xsdgen.AllowType(part.Type))
					}
				}
			}
		}
	}
}
