package xmltree

import (
	"encoding/xml"
	"testing"

	"github.com/kr/pretty"
)

var doc = []byte(`<?xml version="1.0" encoding="utf-8"?>
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

func TestParse(t *testing.T) {
	var buf struct {
		Data []byte `xml:",innerxml"`
	}
	el, err := Parse(doc)
	if err != nil {
		t.Fatal(err)
	}
	el.walk(func(el *Element) {
		el.walk(func(el *Element) {
			if err := el.Unmarshal(&buf); err != nil {
				t.Error(err)
			}
			t.Logf("%s", buf.Data)
		})
	})
	if err != nil {
		t.Error(err)
	}
}

func TestSearch(t *testing.T) {
	root, err := Parse(doc)
	if err != nil {
		t.Fatal(err)
	}

	result := root.Search("http://schemas.xmlsoap.org/wsdl/", "binding")
	if len(result) != 2 {
		t.Errorf("Expected Search(\"http://schemas.xmlsoap.org/wsdl/\", \"binding\") to return 2 results, got %d",
			len(result))
	}
}

func TestNSResolution(t *testing.T) {
	root, err := Parse(doc)
	if err != nil {
		t.Fatal(err)
	}

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
		t.Logf("NS stack is %# v", pretty.Formatter(defaultns.Scope))
	}
}

func TestString(t *testing.T) {
	root, err := Parse(doc)
	if err != nil {
		t.Fatal(err)
	}
	s := root.String()
	if len(s) < 5 {
		t.Error(s)
	}
	t.Log(s)
}
