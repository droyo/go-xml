package xmltree

import (
	"encoding/xml"
	"testing"
)

var exampleDoc = []byte(`<?xml version="1.0" encoding="utf-8"?>
<wsdl:definitions xmlns:soap="http://schemas.xmlsoap.org/wsdl/soap/" xmlns:tm="http://microsoft.com/wsdl/mime/textMatching/" xmlns:soapenc="http://schemas.xmlsoap.org/soap/encoding/" xmlns:mime="http://schemas.xmlsoap.org/wsdl/mime/" xmlns:tns="http://www.sci-grupo.com.mx/" xmlns:s="http://www.w3.org/2001/XMLSchema" xmlns:soap12="http://schemas.xmlsoap.org/wsdl/soap12/" xmlns:http="http://schemas.xmlsoap.org/wsdl/http/" targetNamespace="http://www.sci-grupo.com.mx/" xmlns:wsdl="http://schemas.xmlsoap.org/wsdl/" xmlns="http://defaultns.net/">
  <wsdl:types>
    <s:schema elementFormDefault="qualified" targetNamespace="http://www.sci-grupo.com.mx/">
      <s:element name="RecibeCFD">
        <s:complexType>
          <s:sequence>
            <s:element minOccurs="0" maxOccurs="1" name="XMLCFD" type="s:string" />
          </s:sequence>
        </s:complexType>
      </s:element>
      <s:element name="RecibeCFDResponse">
        <s:complexType>
          <s:sequence>
            <s:element minOccurs="0" maxOccurs="1" name="RecibeCFDResult" type="s:string" />
          </s:sequence>
        </s:complexType>
      </s:element>
    </s:schema>
  </wsdl:types>
  <wsdl:message name="RecibeCFDSoapIn">
    <wsdl:part name="parameters" element="tns:RecibeCFD" />
  </wsdl:message>
  <wsdl:message name="RecibeCFDSoapOut">
    <wsdl:part name="parameters" element="tns:RecibeCFDResponse" />
  </wsdl:message>
  <wsdl:portType name="wseDocReciboSoap">
    <wsdl:operation name="RecibeCFD">
      <wsdl:input message="tns:RecibeCFDSoapIn" />
      <wsdl:output message="tns:RecibeCFDSoapOut" />
    </wsdl:operation>
  </wsdl:portType>
  <wsdl:binding name="wseDocReciboSoap" type="tns:wseDocReciboSoap" xmlns="http://custom2/">
    <soap:binding transport="http://schemas.xmlsoap.org/soap/http" />
    <wsdl:operation name="RecibeCFD">
      <soap:operation soapAction="http://www.sci-grupo.com.mx/RecibeCFD" style="document" />
      <wsdl:input>
        <soap:body use="literal" />
      </wsdl:input>
      <wsdl:output>
        <soap:body use="literal" />
      </wsdl:output>
    </wsdl:operation>
  </wsdl:binding>
  <wsdl:binding name="wseDocReciboSoap12" type="tns:wseDocReciboSoap" xmlns="http://custom/">
    <soap12:binding transport="http://schemas.xmlsoap.org/soap/http" />
    <wsdl:operation name="RecibeCFD">
      <soap12:operation soapAction="http://www.sci-grupo.com.mx/RecibeCFD" style="document" />
      <wsdl:input>
        <soap12:body use="literal" />
      </wsdl:input>
      <wsdl:output>
        <soap12:body use="literal" />
      </wsdl:output>
    </wsdl:operation>
  </wsdl:binding>
  <wsdl:service name="wseDocRecibo">
    <wsdl:port name="wseDocReciboSoap" binding="tns:wseDocReciboSoap">
      <soap:address location="http://www2.soriana.com/integracion/recibecfd/wseDocRecibo.asmx" />
    </wsdl:port>
    <wsdl:port name="wseDocReciboSoap12" binding="tns:wseDocReciboSoap12">
      <soap12:address location="http://www2.soriana.com/integracion/recibecfd/wseDocRecibo.asmx" />
    </wsdl:port>
  </wsdl:service>
</wsdl:definitions>`)

func parseDoc(t *testing.T, document []byte) *Element {
	root, err := Parse(document)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestParse(t *testing.T) {
	var buf struct {
		Data []byte `xml:",innerxml"`
	}
	el := parseDoc(t, exampleDoc)
	el.walk(func(el *Element) {
		el.walk(func(el *Element) {
			if err := el.Unmarshal(&buf); err != nil {
				t.Error(err)
			}
			t.Logf("%s", buf.Data)
		})
	})
}

func TestSearch(t *testing.T) {
	root := parseDoc(t, exampleDoc)

	result := root.Search("http://schemas.xmlsoap.org/wsdl/", "binding")
	if len(result) != 2 {
		t.Errorf("Expected Search(\"http://schemas.xmlsoap.org/wsdl/\", \"binding\") to return 2 results, got %d",
			len(result))
	}
}

func TestNSResolution(t *testing.T) {
	root := parseDoc(t, exampleDoc)

	for _, el := range root.Search("http://schemas.xmlsoap.org/wsdl/", "definitions") {
		for _, prefix := range []string{"soap", "wsdl", "s", "soap12"} {
			if name, ok := el.ResolveNS(prefix + ":foo"); !ok {
				t.Errorf("Failed to resolve %s: prefix at <%s>", prefix, el.Name.Local)
			} else {
				t.Logf("Resoved prefix %s to %q at <%s name=%q>", prefix, name.Space,
					el.Name.Local, el.Attr("", "name"))
			}
		}
	}

	defaultns := root.SearchFunc(func(el *Element) bool {
		if (el.Name != xml.Name{"http://schemas.xmlsoap.org/wsdl/", "binding"}) {
			return false
		}
		return el.Attr("", "name") == "wseDocReciboSoap12"
	})[0]

	name := defaultns.Resolve("foo")
	if name.Space != "http://custom/" {
		t.Errorf("Resolve default namespace at <%s name=%q>: wanted %q, got %q",
			defaultns.Prefix(defaultns.Name), defaultns.Attr("", "name"), defaultns.Attr("", "xmlns"), name.Space)
		t.Logf("NS stack is %# v", defaultns.Scope)
	}
}

func TestString(t *testing.T) {
	root := parseDoc(t, exampleDoc)
	s := root.String()
	if len(s) < 5 {
		t.Error(s)
	}
	parseDoc(t, []byte(s))
	t.Log(s)
}

func TestSubstring(t *testing.T) {
	root := parseDoc(t, exampleDoc)
	for _, el := range root.Search("http://www.w3.org/2001/XMLSchema", "complexType") {
		s := el.String()
		parseDoc(t, []byte(s))
		break
	}
}

func TestModification(t *testing.T) {
	from := []byte(`<ul><li>1</li><em>bad</em><li>2</li></ul>`)
	to := `<ul><li>1</li><li>2</li></ul>`
	root := parseDoc(t, from)
	// Remove any non-<li> children from all <ul> elements
	// in the document.
	valid := make([]Element, 0, len(root.Children))
	for _, p := range root.Search("", "li") {
		t.Logf("%#v", *p)
		valid = append(valid, *p)
	}
	root.Children = valid
	if s := root.String(); s != to {
		t.Errorf("%s -> %s, expected %s", from, s, to)
	}
}

func TestStringPreserveNS(t *testing.T) {
	root := parseDoc(t, exampleDoc)
	var doc []byte
	var descent = 4
	for _, el := range root.SearchFunc(func(*Element) bool { return true }) {
		descent--
		if descent <= 0 {
			doc = Marshal(el)
			break
		}
	}
	root = parseDoc(t, doc)
	t.Logf("%s", doc)
	if len(root.Search("http://www.w3.org/2001/XMLSchema", "sequence")) == 0 {
		t.Errorf("Could not find <s:sequence> in %s", root.String())
	}
}
