// testgen is a wrapper around xsdgen that generates unit
// for generated code.
package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"go/ast"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"aqwari.net/xml/internal/gen"
	"aqwari.net/xml/xmltree"
	"aqwari.net/xml/xsdgen"
)

var (
	output = flag.String("o", "gen_test.go", "test filename ending")
	pkg    = flag.String("pkg", "", "name of test's package")
)

func main() {
	cfg := new(xsdgen.Config)

	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("usage: testgen [-o outfile] dir")
	}
	if *pkg == "" {
		*pkg = os.Getenv("GOPACKAGE")
	}
	cases := findTestCases(flag.Arg(0))

	cfg.Option(xsdgen.DefaultOptions...)
	cfg.Option(
		xsdgen.PackageName(*pkg),
	)
	for _, stem := range cases {
		data, err := ioutil.ReadFile(stem + ".xsd")
		if err != nil {
			log.Print(err)
			continue
		}
		tests, err := genTests(cfg, data, stem+".xml")
		if err != nil {
			log.Print(stem, ":", err)
			continue
		}
		source, err := gen.FormattedSource(tests)
		if err != nil {
			log.Print(stem, ":", err)
			continue
		}
		filename := filepath.Base(stem) + "_" + *output
		if err := ioutil.WriteFile(filename, source, 0666); err != nil {
			log.Print(stem, ":", err)
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
func genTests(cfg *xsdgen.Config, data []byte, dataFile string) (*ast.File, error) {
	base := filepath.Base(dataFile)
	base = base[:len(base)-len(filepath.Ext(base))]

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
		DataFile  string
	}
	params.DocStruct = expr
	params.DataFile = dataFile
	fn, err := gen.Func("Test"+strings.Title(base)).
		Args("t *testing.T").
		BodyTmpl(`
			type Document {{.DocStruct}}
			var document Document
			input, err := ioutil.ReadFile("{{.DataFile}}")
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
				t.Fatal("{{.DataFile}}: ", err)
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
			break
		}
	}
	return result
}

// Looks for pairs of (xml, xsd) files in a directory, that
// should contain xml data and the schema that describes
// it, respectively. Returns slice of file names with extension
// removed.
func findTestCases(dir string) []string {
	filenames, err := filepath.Glob(filepath.Join(dir, "*.xml"))
	if err != nil {
		return nil
	}
	testCases := make([]string, 0, len(filenames))
	for _, xmlfile := range filenames {
		ext := filepath.Ext(xmlfile)
		if len(ext) == len(xmlfile) {
			continue
		}
		base := xmlfile[:len(xmlfile)-len(ext)]
		schemafile := base + ".xsd"
		if _, err := os.Stat(schemafile); err != nil {
			continue
		}
		testCases = append(testCases, base)
	}
	return testCases
}
