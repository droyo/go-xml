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

	"aqwari.net/xml/internal/gen"
	"aqwari.net/xml/xmltree"
	"aqwari.net/xml/xsd"
	"aqwari.net/xml/xsdgen"
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
	cases := findXSDTestCases()

	cfg.Option(xsdgen.DefaultOptions...)
	for _, dir := range cases {
		data, err := ioutil.ReadFile(glob(filepath.Join(dir, "*.xsd")))
		if err != nil {
			errorsEncountered = true
			log.Print(err)
			continue
		}
		tests, err := genXSDTests(*cfg, data, dir)
		if err != nil {
			errorsEncountered = true
			log.Print(dir, ":", err)
			continue
		} else {
			log.Printf("generated xsd tests for %s", dir)
		}
		if err := writeTestFiles(tests, dir); err != nil {
			errorsEncountered = true
			log.Print(dir, ":", err)
		}
	}

	if errorsEncountered {
		os.Exit(1)
	}
}

func writeTestFiles(file *ast.File, pkg string) error {
	testFilename := filepath.Join(pkg, pkg+"_test.go")

	src, err := gen.FormattedSource(file, testFilename)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(testFilename, src, 0666); err != nil {
		return err
	}

	// This is needed so 'go build' works and automated CI doesn't complain.
	buildFilename := filepath.Join(pkg, pkg+".go")
	return ioutil.WriteFile(buildFilename, []byte("package "+pkg+"\n"), 0666)
}

// Generates unit tests for xml marshal unmarshalling of
// schema-generated code. The unit test will do the
// following:
//
// - Unmarshal the sample data (dataFile) into a struct representing
//   the document described in the XML schema.
// - Marshal the resulting file back into an XML document.
// - Compare the two documents for equality.
func genXSDTests(cfg xsdgen.Config, data []byte, dir string) (*ast.File, error) {
	cfg.Option(xsdgen.PackageName(dir))
	code, err := cfg.GenCode(data)
	if err != nil {
		return nil, err
	}
	file, err := code.GenAST()
	if err != nil {
		return nil, err
	}

	// We look for top-level elements in the schema to determine what
	// the example document looks like.
	roots, err := xsd.Normalize(data)
	if err != nil {
		return nil, err
	}
	if len(roots) < 1 {
		return nil, fmt.Errorf("no schema in %s", dir)
	}
	root := roots[0]
	doc := topLevelElements(root)
	fields := make([]ast.Expr, 0, len(doc)*3)

	for _, elem := range doc {
		fields = append(fields,
			gen.Public(elem.Name.Local),
			ast.NewIdent(code.NameOf(elem.Type)),
			gen.String(fmt.Sprintf(`xml:"%s %s"`, elem.Name.Space, elem.Name.Local)))
	}
	expr, err := gen.ToString(gen.Struct(fields...))
	if err != nil {
		return nil, err
	}

	var params struct {
		DocStruct string
		Dir       string
	}
	params.DocStruct = expr
	params.Dir = dir
	fn, err := gen.Func("Test"+strings.Title(dir)).
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
				t.Fatal("{{.Dir}}: ", err)
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
		return nil, err
	}
	// Test goes at the top
	file.Decls = append([]ast.Decl{fn}, file.Decls...)
	return file, nil
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

// Looks for subdirectories containing pairs of (xml, xsd) files
// that should contain an xml document and the schema it conforms to,
// respectively. Returns slice of the directory names
func findXSDTestCases() []string {
	filenames, err := filepath.Glob("*/*.xsd")
	if err != nil {
		return nil
	}
	result := make([]string, 0, len(filenames))
	for _, xsdfile := range filenames {
		result = append(result, filepath.Base(filepath.Dir(xsdfile)))
	}
	return result
}
