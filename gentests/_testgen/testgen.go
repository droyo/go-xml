// testgen is a wrapper around xsdgen that generates unit
// for generated code.
package main

import (
	"encoding/xml"
	"fmt"
	"go/ast"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"aqwari.net/xml/internal/gen"
	"aqwari.net/xml/xmltree"
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
	cfg := new(xsdgen.Config)
	cases := findTestCases()

	cfg.Option(xsdgen.DefaultOptions...)
	for _, dir := range cases {
		data, err := ioutil.ReadFile(glob(filepath.Join(dir, "*.xsd")))
		if err != nil {
			log.Print(err)
			continue
		}
		tests, err := genTests(*cfg, data, dir)
		if err != nil {
			log.Print(dir, ":", err)
			continue
		} else {
			log.Printf("generated tests for %s", dir)
		}
		source, err := gen.FormattedSource(tests)
		if err != nil {
			log.Print(dir, ":", err)
			continue
		}
		filename := filepath.Join(dir, dir+"_test.go")
		if err := ioutil.WriteFile(filename, source, 0666); err != nil {
			log.Print(dir, ":", err)
		}

		// This is needed so 'go build' works and automated CI doesn't complain.
		err = ioutil.WriteFile(filepath.Join(dir, dir+".go"), []byte("package "+dir+"\n"), 0666)
		if err != nil {
			log.Print(err)
		}
	}
}

// Generates unit tests for xml marshal unmarshalling of
// schema-generated code. The unit test will do the
// following:
//
// - Unmarshal the sample data (dataFile) into a struct representing
//   the document described in the XML schema.
// - Marshal the resulting file back into an XML document.
// - Compare the two documents for equality.
func genTests(cfg xsdgen.Config, data []byte, dir string) (*ast.File, error) {
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
	root, err := xmltree.Parse(data)
	if err != nil {
		return nil, err
	}
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
func findTestCases() []string {
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
