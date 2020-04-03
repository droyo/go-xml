// testgen is a wrapper around xsdgen that generates unit
// for generated code.
package main

import (
	"encoding/xml"
	"fmt"
	"go/ast"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/CognitoIQ/go-xml/internal/gen"
	"github.com/CognitoIQ/go-xml/xmltree"
	"github.com/CognitoIQ/go-xml/xsd"
	"github.com/CognitoIQ/go-xml/xsdgen"
)

func glob(pat string) string {
	f, err := filepath.Glob(pat)
	if err != nil {
		log.Fatal(err)
	}
	if len(f) < 1 {
		log.Fatal("no files match ", pat)
	}
	return f[0]
}

func main() {
	var errorsEncountered bool
	cfg := new(xsdgen.Config)

	xsdTestCases, err := findXSDTestCases()
	if err != nil {
		log.Fatal(err)
	}

	cfg.Option(xsdgen.DefaultOptions...)
	for _, testCase := range xsdTestCases {
		code, tests, err := genXSDTests(*cfg, testCase.doc, testCase.pkg)
		if err != nil {
			errorsEncountered = true
			log.Print(testCase.pkg)
			continue
		} else {
			log.Printf("generated xsd tests for %s", testCase.pkg)
		}
		if err := writeTestFiles(code, tests, testCase.pkg); err != nil {
			errorsEncountered = true
			log.Print(testCase.pkg, ":", err)
		}
	}

	if errorsEncountered {
		os.Exit(1)
	}
}

func writeTestFiles(code, tests *ast.File, pkg string) error {
	testFilename := filepath.Join(pkg, pkg+"_test.go")
	testSrc, err := gen.FormattedSource(tests, testFilename)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(testFilename, testSrc, 0666); err != nil {
		return err
	}

	codeFilename := filepath.Join(pkg, pkg+".go")
	codeSrc, err := gen.FormattedSource(code, codeFilename)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(codeFilename, codeSrc, 0666)
}

// Generates unit tests for xml marshal unmarshalling of
// schema-generated code. The unit test will do the
// following:
//
// - Unmarshal the sample data (dataFile) into a struct representing
//   the document described in the XML schema.
// - Marshal the resulting file back into an XML document.
// - Compare the two documents for equality.
//
// Returns type definitions and unit tests as separate files.
func genXSDTests(cfg xsdgen.Config, data []byte, pkg string) (code, tests *ast.File, err error) {
	cfg.Option(xsdgen.PackageName(pkg))
	main, err := cfg.GenCode(data)
	if err != nil {
		return nil, nil, err
	}
	code, err = main.GenAST()
	if err != nil {
		return nil, nil, err
	}

	tests = new(ast.File)
	tests.Name = ast.NewIdent(pkg)

	// We look for top-level elements in the schema to determine what
	// the example document looks like.
	roots, err := xsd.Normalize(data)
	if err != nil {
		return nil, nil, err
	}
	if len(roots) < 1 {
		return nil, nil, fmt.Errorf("no schema in %s", pkg)
	}
	root := roots[0]
	doc := topLevelElements(root)
	fields := make([]ast.Expr, 0, len(doc)*3)

	for _, elem := range doc {
		fields = append(fields,
			gen.Public(elem.Name.Local),
			ast.NewIdent(main.NameOf(elem.Type)),
			gen.String(fmt.Sprintf(`xml:"%s %s"`, elem.Name.Space, elem.Name.Local)))
	}
	expr, err := gen.ToString(gen.Struct(fields...))
	if err != nil {
		return nil, nil, err
	}

	var params struct {
		DocStruct string
		Pkg       string
	}
	params.DocStruct = expr
	params.Pkg = pkg
	fn, err := gen.Func("Test"+strings.Title(pkg)).
		Args("t *testing.T").
		BodyTmpl(`
			type Document {{.DocStruct}}
			var document Document
			samples, err := filepath.Glob(filepath.Join("*.xml"))
			if err != nil {
				t.Fatal(err)
			}
			if len(samples) != 1 {
				t.Fatal("expected one sample file, found ", samples)
			}
			
			input, err := ioutil.ReadFile(samples[0])
			if err != nil {
				t.Fatal(err)
			}
			input = append([]byte("<Document>\n"), input...)
			input = append(input, []byte("</Document>")...)
			if err := xml.Unmarshal(input, &document); err != nil {
				t.Fatal("unmarshal: ", err)
			}
			output, err := xml.Marshal(&document)
			if err != nil {
				t.Fatal("marshal: ", err)
			}
			
			inputTree, err := xmltree.Parse(input)
			if err != nil {
				t.Fatal("{{.Pkg}}: ", err)
			}
			
			outputTree, err := xmltree.Parse(output)
			if err != nil {
				t.Fatal("remarshal: ", err)
			}
			
			if !xmltree.Equal(inputTree, outputTree) {
				t.Errorf("got \n%s\n, wanted \n%s\n",
					xmltree.MarshalIndent(outputTree, "", "  "),
					xmltree.MarshalIndent(inputTree, "", "  "))
			}
			`, params).Decl()

	if err != nil {
		return nil, nil, err
	}
	tests.Decls = append(tests.Decls, fn)
	return code, tests, nil
}

type Element struct {
	Name, Type xml.Name
}

func topLevelElements(root *xmltree.Element) []Element {
	const schemaNS = "http://www.w3.org/2001/XMLSchema"

	result := make([]Element, 0)
	root = &xmltree.Element{Scope: root.Scope, Children: []xmltree.Element{*root}}
	for _, schema := range root.Search(schemaNS, "schema") {
		tns := schema.Attr("", "targetNamespace")
		for _, el := range schema.Children {
			if (el.Name == xml.Name{schemaNS, "element"}) {
				result = append(result, Element{
					Name: el.ResolveDefault(el.Attr("", "name"), tns),
					Type: el.Resolve(el.Attr("", "type")),
				})
			}
		}
	}
	return result
}

type testCase struct {
	pkg string
	doc []byte
}

// Looks for subdirectories containing pairs of (xml, xsd) files
// that should contain an xml document and the schema it conforms to,
// respectively. Returns slice of the directory names
func findXSDTestCases() ([]testCase, error) {
	filenames, err := filepath.Glob("*/*.xsd")
	if err != nil {
		return nil, err
	}
	result := make([]testCase, 0, len(filenames))
	for _, xsdfile := range filenames {
		if data, err := ioutil.ReadFile(xsdfile); err != nil {
			return nil, err
		} else {
			result = append(result, testCase{
				pkg: filepath.Base(filepath.Dir(xsdfile)),
				doc: data,
			})
		}
	}
	return result, nil
}
