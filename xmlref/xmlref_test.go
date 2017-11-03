package xmlref

//go:generate go run ../cmd/xsdgen/xsdgen.go -pkg xmlref -o types_test.go wsdl.xml

import (
	"bytes"
	"encoding/xml"
	"testing"

	"github.com/kr/pretty"
)

var uglyXML = []byte(`<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
   <soapenv:Body>
      <ns1:getLocationsResponse soapenv:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xmlns:ns1="http://ivr.gws">
         <getLocationsReturn href="#id0"/>
      </ns1:getLocationsResponse>
      <multiRef id="id0" soapenc:root="0" soapenv:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xsi:type="ns2:IVROutputLocationMap" xmlns:soapenc="http://schemas.xmlsoap.org/soap/encoding/" xmlns:ns2="http://location.model.ivr.gws">
         <anyMoreRec href="#id1"/>
         <errors xsi:type="soapenc:Array" xsi:nil="true"/>
         <markers href="#id2"/>
         <sessionID xsi:type="soapenc:string" xsi:nil="true"/>
      </multiRef>
      <multiRef id="id2" soapenc:root="0" soapenv:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" soapenc:arrayType="xsd:anyType[240]" xsi:type="soapenc:Array" xmlns:soapenc="http://schemas.xmlsoap.org/soap/encoding/">
         <multiRef href="#id3"/>
         <multiRef href="#id4"/>
      </multiRef>
      <multiRef id="id4" soapenc:root="0" soapenv:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xsi:type="ns11:IVRGoogleMapsMarker" xmlns:ns11="http://location.model.ivr.gws" xmlns:soapenc="http://schemas.xmlsoap.org/soap/encoding/">
         <id xsi:type="soapenc:string">location-2-id</id>
         <type xsi:type="soapenc:string">location-2-type</type>
         <title xsi:type="soapenc:string">Location 2</title>
         <address xsi:type="soapenc:string">Address of location 2</address>
         <information xsi:type="soapenc:string">Some informations for location 2</information>
         <locLat xsi:type="soapenc:string">2.22222</locLat>
         <locLng xsi:type="soapenc:string">2.22222</locLng>
         <showGPS href="#id269"/>
      </multiRef>
      <multiRef id="id3" soapenc:root="0" soapenv:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xsi:type="ns11:IVRGoogleMapsMarker" xmlns:ns11="http://location.model.ivr.gws" xmlns:soapenc="http://schemas.xmlsoap.org/soap/encoding/">
         <id xsi:type="soapenc:string">location-1-id</id>
         <type xsi:type="soapenc:string">location-1-type</type>
         <title xsi:type="soapenc:string">Location 1</title>
         <address xsi:type="soapenc:string">Address of location 1</address>
         <information xsi:type="soapenc:string">Some informations for location 1</information>
         <locLat xsi:type="soapenc:string">1.11111</locLat>
         <locLng xsi:type="soapenc:string">1.11111</locLng>
         <showGPS href="#id269"/>
      </multiRef>
      <multiRef id="id1" soapenc:root="0" soapenv:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xsi:type="xsd:boolean" xmlns:soapenc="http://schemas.xmlsoap.org/soap/encoding/">false</multiRef>
      <multiRef id="id269" soapenc:root="0" soapenv:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/" xsi:type="xsd:boolean" xmlns:soapenc="http://schemas.xmlsoap.org/soap/encoding/">true</multiRef>
   </soapenv:Body>
</soapenv:Envelope>`)

type Envelope struct {
	XMLName xml.Name
	Return  IVROutputLocationMap `xml:"Body>Response>getLocationsReturn"`
}

func TestDecoder(t *testing.T) {
	r := bytes.NewReader(uglyXML)
	d := xml.NewDecoder(r)
	env := new(Envelope)

	if err := d.Decode(env); err != nil {
		t.Fatal(err)
	}
	if len(env.Return.Markers) < 1 {
		t.Error("no data unmarshalled")
	}
	t.Logf("%s", pretty.Sprint(env))
}
