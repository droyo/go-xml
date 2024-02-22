package xmltree_test

import (
	"encoding/xml"
	"log"
	"testing"

	"aqwari.net/xml/xmltree"
)

// Check for proper XML escape quoting inside attributes

func TestXMLParseAttribute(t *testing.T) {
	var err error

	type Module struct {
		XMLName xml.Name `xml:"module"`
		Type    string   `xml:"name,attr"`
	}

	xmlBytes := []byte(`<module name="foo"></module>`)

	// []byte -> Module object
	var moduleValue Module
	err = xml.Unmarshal(xmlBytes, &moduleValue)
	if err != nil {
		panic(err)
	}

	// Format Module as XML
	xmlOutBytes, outErr := xml.Marshal(moduleValue)
	if outErr != nil {
		panic(outErr)
	}

	{
		have := string(xmlOutBytes)
		want := "<module name=\"foo\"></module>"

		if have != want {
			t.Fatalf("!Match : want : have :\n-----\n%v\n-----\n%v\n-----", want, have)
		}
	}
}

// golang xml Unmarshal for an attribute

func TestXMLParseEscapedAttributeStd(t *testing.T) {
	var err error

	type Module struct {
		XMLName xml.Name `xml:"module"`
		Name    string   `xml:"name,attr"`
	}

	// &lt; is the same as &#60;
	// &gt; is the same as &#62;
	//
	// < -> &lt;
	// > -> &gt;

	xmlBytes := []byte(`<module name='&lt;'></module>`)

	// []byte -> Module object
	var moduleValue Module
	err = xml.Unmarshal(xmlBytes, &moduleValue)
	if err != nil {
		panic(err)
	}

	// Format Module as XML
	xmlOutBytes, outErr := xml.Marshal(moduleValue)
	if outErr != nil {
		panic(outErr)
	}

	// Note that golang default XML Marshal will format as "&lt;"

	{
		have := string(xmlOutBytes)
		want := `<module name="&lt;"></module>`

		if have != want {
			t.Fatalf("!Match : want : have :\n-----\n%v\n-----\n%v\n-----", want, have)
		}
	}
}

// Escaped characters inside (as chardata)

func TestXMLParseEscapedValueStd(t *testing.T) {
	var err error

	type Module struct {
		XMLName xml.Name `xml:"module"`
		Value   string   `xml:",chardata"`
	}

	xmlBytes := []byte(`<module>&lt;</module>`)

	// []byte -> Module object
	var moduleValue Module
	err = xml.Unmarshal(xmlBytes, &moduleValue)
	if err != nil {
		panic(err)
	}

	// Format Module as XML
	xmlOutBytes, outErr := xml.Marshal(moduleValue)
	if outErr != nil {
		panic(outErr)
	}

	// Note that golang default XML Marshal will format as "&lt;"

	{
		have := string(xmlOutBytes)
		want := `<module>&lt;</module>`

		if have != want {
			t.Fatalf("!Match : want : have :\n-----\n%v\n-----\n%v\n-----", want, have)
		}
	}
}

// Parse and then format with xmltree module

func TestXMLParseEscapedAttributeWithXMLTree(t *testing.T) {
	var err error

	type Module struct {
		XMLName xml.Name `xml:"module"`
		Name    string   `xml:"name,attr"`
	}

	xmlBytes := []byte(`<module name='&lt;'></module>`)

	// []byte -> Module object
	rootNode, err := xmltree.Parse(xmlBytes)
	if err != nil {
		log.Fatal(err)
	}

	xmlOutBytes := xmltree.MarshalIndent(rootNode, "", "  ")

	{
		have := string(xmlOutBytes)
		want := `<module name="&lt;" />` + "\n"

		if have != want {
			t.Fatalf("!Match : want : have :\n-----\n%v\n-----\n%v\n-----", want, have)
		}
	}
}

// Parse escaped value inside XML tags using xmltree module

func TestXMLParseEscapedValueXMLTree(t *testing.T) {
	var err error

	type Module struct {
		XMLName xml.Name `xml:"module"`
		Value   string   `xml:",chardata"`
	}

	xmlBytes := []byte(`<module>&lt;&gt;</module>`)

	// []byte -> Module object
	rootNode, err := xmltree.Parse(xmlBytes)
	if err != nil {
		log.Fatal(err)
	}

	xmlOutBytes := xmltree.MarshalIndent(rootNode, "", "  ")

	{
		have := string(xmlOutBytes)
		want := `<module>&lt;&gt;</module>` + "\n"

		if have != want {
			t.Fatalf("!Match : want : have :\n-----\n%v\n-----\n%v\n-----", want, have)
		}
	}
}

