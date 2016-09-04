package xsdgen_test

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"aqwari.net/xml/xsdgen"
)

func tmpfile() *os.File {
	f, err := ioutil.TempFile("", "xsdgen_test")
	if err != nil {
		panic(err)
	}
	return f
}

func xsdfile(s string) (filename string) {
	file := tmpfile()
	defer file.Close()
	fmt.Fprintf(file, `
		<schema xmlns="http://www.w3.org/2001/XMLSchema"
		        xmlns:tns="http://www.example.com/"
		        xmlns:xs="http://www.w3.org/2001/XMLSchema"
		        xmlns:soapenc="http://schemas.xmlsoap.org/soap/encoding/"
		        xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/"
		        targetNamespace="http://www.example.com/">
		  %s
		</schema>
	`, s)
	return file.Name()
}

func ExampleConfig_GenCLI() {
	var cfg xsdgen.Config
	cfg.Option(
		xsdgen.IgnoreAttributes("id", "href", "offset"),
		xsdgen.IgnoreElements("comment"),
		xsdgen.PackageName("webapi"),
		xsdgen.Replace("_", ""),
		xsdgen.HandleSOAPArrayType(),
		xsdgen.SOAPArrayAsSlice(),
	)
	if err := cfg.GenCLI("webapi.xsd", "deps/soap11.xsd"); err != nil {
		log.Fatal(err)
	}
}

func ExampleLogOutput() {
	var cfg xsdgen.Config
	cfg.Option(
		xsdgen.LogOutput(log.New(os.Stderr, "", 0)),
		xsdgen.LogLevel(2))
	if err := cfg.GenCLI("file.wsdl"); err != nil {
		log.Fatal(err)
	}
}

func ExampleIgnoreAttributes() {
	doc := xsdfile(`
	  <complexType name="ArrayOfString">
	    <any maxOccurs="unbounded" />
	    <attribute name="soapenc:arrayType" type="xs:string" />
	  </complexType>
	`)
	var cfg xsdgen.Config
	cfg.Option(xsdgen.IgnoreAttributes("arrayType"))

	out, err := cfg.GenSource(doc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", out)

	// Output: package ws
	//
	// type ArrayOfString struct {
	// 	Items []string `xml:",any"`
	// }
}

func ExampleIgnoreElements() {
	doc := xsdfile(`
	  <complexType name="Person">
	    <sequence>
	      <element name="name" type="xs:string" />
	      <element name="deceased" type="soapenc:boolean" />
	      <element name="private" type="xs:int" />
	    </sequence>
	  </complexType>
	`)
	var cfg xsdgen.Config
	cfg.Option(
		xsdgen.IgnoreElements("private"),
		xsdgen.IgnoreAttributes("id", "href"))

	out, err := cfg.GenSource(doc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", out)

	// Output: package ws
	//
	// type Person struct {
	// 	Name     string `xml:"http://www.example.com/ name"`
	// 	Deceased bool   `xml:"http://www.example.com/ deceased"`
	// }
}

func ExamplePackageName() {
	doc := xsdfile(`
	  <simpleType name="zipcode">
	    <restriction base="xs:string">
	      <length value="10" />
	    </restriction>
	  </simpleType>
	`)
	var cfg xsdgen.Config
	cfg.Option(xsdgen.PackageName("postal"))

	out, err := cfg.GenSource(doc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", out)

	// Output: package postal
	//
	// // May be no more than 10 items long
	// type Zipcode string
}

func ExampleReplace() {
	doc := xsdfile(`
	  <complexType name="ArrayOfString">
	    <any maxOccurs="unbounded" />
	    <attribute name="soapenc:arrayType" type="xs:string" />
	  </complexType>
	`)
	var cfg xsdgen.Config
	cfg.Option(xsdgen.Replace("ArrayOf(.*)", "${1}Array"))

	out, err := cfg.GenSource(doc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", out)

	// Output: package ws
	//
	// type StringArray struct {
	// 	ArrayType string   `xml:"arrayType,attr"`
	// 	Items     []string `xml:",any"`
	// }
}

func ExampleHandleSOAPArrayType() {
	doc := xsdfile(`
	  <complexType name="BoolArray">
	    <complexContent>
	      <restriction base="soapenc:Array">
	        <attribute ref="soapenc:arrayType" wsdl:arrayType="xs:boolean[]"/>
	      </restriction>
	    </complexContent>
	  </complexType>`)

	var cfg xsdgen.Config
	cfg.Option(xsdgen.HandleSOAPArrayType())

	out, err := cfg.GenSource(doc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", out)

	// Output: package ws
	//
	// type BoolArray struct {
	// 	Offset ArrayCoordinate `xml:"offset,attr"`
	// 	Id     string          `xml:"id,attr"`
	// 	Href   string          `xml:"href,attr"`
	// 	Items  []bool          `xml:",any"`
	// }
}

func ExampleSOAPArrayAsSlice() {
	doc := xsdfile(`
	  <complexType name="BoolArray">
	    <complexContent>
	      <restriction base="soapenc:Array">
	        <attribute ref="soapenc:arrayType" wsdl:arrayType="xs:boolean[]"/>
	      </restriction>
	    </complexContent>
	  </complexType>`)

	var cfg xsdgen.Config
	cfg.Option(
		xsdgen.HandleSOAPArrayType(),
		xsdgen.SOAPArrayAsSlice(),
		xsdgen.LogOutput(log.New(os.Stderr, "", 0)),
		xsdgen.LogLevel(3),
		xsdgen.IgnoreAttributes("offset", "id", "href"))

	out, err := cfg.GenSource(doc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", out)

	// Output: package ws
	//
	// import "encoding/xml"
	//
	// type BoolArray []bool
	//
	// func (a BoolArray) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	// 	var output struct {
	// 		ArrayType string `xml:"http://schemas.xmlsoap.org/wsdl/ arrayType,attr"`
	// 		Items     []bool `xml:" item"`
	// 	}
	// 	output.Items = []bool(a)
	// 	start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{"", "xmlns:ns1"}, Value: "http://www.w3.org/2001/XMLSchema"})
	// 	output.ArrayType = "ns1:boolean[]"
	// 	return e.EncodeElement(&output, start)
	// }
	// func (a *BoolArray) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	// 	var tok xml.Token
	// 	for tok, err = d.Token(); err == nil; tok, err = d.Token() {
	// 		if tok, ok := tok.(xml.StartElement); ok {
	// 			var item bool
	// 			if err = d.DecodeElement(&item, &tok); err == nil {
	// 				*a = append(*a, item)
	// 			}
	// 		}
	// 		if _, ok := tok.(xml.EndElement); ok {
	// 			break
	// 		}
	// 	}
	// 	return err
	// }
}

func ExampleUseFieldNames() {
	doc := xsdfile(`
	  <complexType name="library">
	    <sequence>
	      <element name="book" maxOccurs="unbounded">
	        <complexType>
	          <all>
	            <element name="title" type="xs:string" />
	            <element name="published" type="xs:date" />
	            <element name="author" type="xs:string" />
	          </all>
	        </complexType>
	      </element>
	    </sequence>
	  </complexType>`)

	var cfg xsdgen.Config
	cfg.Option(xsdgen.UseFieldNames())

	out, err := cfg.GenSource(doc)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", out)

	// Output: package ws
	//
	// import (
	// 	"bytes"
	// 	"time"
	// )
	//
	// type Book struct {
	// 	Title     string  `xml:"http://www.example.com/ title"`
	// 	Published xsdDate `xml:"http://www.example.com/ published"`
	// 	Author    string  `xml:"http://www.example.com/ author"`
	// }
	//
	// type Library struct {
	// 	Book      []Book  `xml:"http://www.example.com/ book"`
	// 	Title     string  `xml:"http://www.example.com/ title"`
	// 	Published xsdDate `xml:"http://www.example.com/ published"`
	// 	Author    string  `xml:"http://www.example.com/ author"`
	// }
	//
	// type xsdDate time.Time
	//
	// func (t *xsdDate) UnmarshalText(text []byte) error {
	// 	return _unmarshalTime(text, (*time.Time)(t), "2006-01-02")
	// }
	// func (t *xsdDate) MarshalText() ([]byte, error) {
	// 	return []byte((*time.Time)(t).Format("2006-01-02")), nil
	// }
	// func _unmarshalTime(text []byte, t *time.Time, format string) (err error) {
	// 	s := string(bytes.TrimSpace(text))
	// 	*t, err = time.Parse(format, s)
	// 	if _, ok := err.(*time.ParseError); ok {
	// 		*t, err = time.Parse(format+"Z07:00", s)
	// 	}
	// 	return err
	// }
}
