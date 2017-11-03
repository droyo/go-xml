package xmlref

import "encoding/xml"

type ArrayOftns2MessageError []MessageError

func (a ArrayOftns2MessageError) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	var output struct {
		ArrayType string         `xml:"http://schemas.xmlsoap.org/wsdl/ arrayType,attr"`
		Items     []MessageError `xml:" item"`
	}
	output.Items = []MessageError(a)
	start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{"", "xmlns:ns1"}, Value: "http://soap.entity.commons.gws"})
	output.ArrayType = "ns1:MessageError[]"
	return e.EncodeElement(&output, start)
}
func (a *ArrayOftns2MessageError) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	var tok xml.Token
	for tok, err = d.Token(); err == nil; tok, err = d.Token() {
		if tok, ok := tok.(xml.StartElement); ok {
			var item MessageError
			if err = d.DecodeElement(&item, &tok); err == nil {
				*a = append(*a, item)
			}
		}
		if _, ok := tok.(xml.EndElement); ok {
			break
		}
	}
	return err
}

type ArrayOftns6IVRGoogleMapsMarker []IVRGoogleMapsMarker

func (a ArrayOftns6IVRGoogleMapsMarker) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	var output struct {
		ArrayType string                `xml:"http://schemas.xmlsoap.org/wsdl/ arrayType,attr"`
		Items     []IVRGoogleMapsMarker `xml:" item"`
	}
	output.Items = []IVRGoogleMapsMarker(a)
	start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{"", "xmlns:ns1"}, Value: "http://location.model.ivr.gws"})
	output.ArrayType = "ns1:IVRGoogleMapsMarker[]"
	return e.EncodeElement(&output, start)
}
func (a *ArrayOftns6IVRGoogleMapsMarker) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	var tok xml.Token
	for tok, err = d.Token(); err == nil; tok, err = d.Token() {
		if tok, ok := tok.(xml.StartElement); ok {
			var item IVRGoogleMapsMarker
			if err = d.DecodeElement(&item, &tok); err == nil {
				*a = append(*a, item)
			}
		}
		if _, ok := tok.(xml.EndElement); ok {
			break
		}
	}
	return err
}

type GenericSoapRequest struct {
	LanguageId string `xml:"http://soap.entity.commons.gws languageId,omitempty"`
	MaxRecords int    `xml:"http://soap.entity.commons.gws maxRecords"`
}

type GenericSoapResponse struct {
	AnyMoreRec bool                    `xml:"http://soap.entity.commons.gws anyMoreRec"`
	Errors     ArrayOftns2MessageError `xml:"http://soap.entity.commons.gws errors,omitempty"`
}

type IVRGoogleMapsMarker struct {
	Id          string `xml:"http://location.model.ivr.gws id,omitempty"`
	Type        string `xml:"http://location.model.ivr.gws type,omitempty"`
	Title       string `xml:"http://location.model.ivr.gws title,omitempty"`
	Address     string `xml:"http://location.model.ivr.gws address,omitempty"`
	Information string `xml:"http://location.model.ivr.gws information,omitempty"`
	LocLat      string `xml:"http://location.model.ivr.gws locLat,omitempty"`
	LocLng      string `xml:"http://location.model.ivr.gws locLng,omitempty"`
	ShowGPS     bool   `xml:"http://location.model.ivr.gws showGPS"`
}

type IVRInput struct {
	SessionID  string `xml:"http://model.ivr.gws sessionID,omitempty"`
	LanguageId string `xml:"http://soap.entity.commons.gws languageId,omitempty"`
	MaxRecords int    `xml:"http://soap.entity.commons.gws maxRecords"`
}

type IVROutput struct {
	SessionID  string                  `xml:"http://model.ivr.gws sessionID,omitempty"`
	AnyMoreRec bool                    `xml:"http://soap.entity.commons.gws anyMoreRec"`
	Errors     ArrayOftns2MessageError `xml:"http://soap.entity.commons.gws errors,omitempty"`
}

type IVROutputLocationMap struct {
	Markers    ArrayOftns6IVRGoogleMapsMarker `xml:"http://location.model.ivr.gws markers,omitempty"`
	SessionID  string                         `xml:"http://model.ivr.gws sessionID,omitempty"`
	AnyMoreRec bool                           `xml:"http://soap.entity.commons.gws anyMoreRec"`
	Errors     ArrayOftns2MessageError        `xml:"http://soap.entity.commons.gws errors,omitempty"`
}

type MessageError struct {
	Code        string `xml:"http://soap.entity.commons.gws code,omitempty"`
	Description string `xml:"http://soap.entity.commons.gws description,omitempty"`
	Reference   string `xml:"http://soap.entity.commons.gws reference,omitempty"`
	Severity    string `xml:"http://soap.entity.commons.gws severity,omitempty"`
}

type Vector []MessageError

func (a Vector) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	var output struct {
		ArrayType string         `xml:"http://schemas.xmlsoap.org/wsdl/ arrayType,attr"`
		Items     []MessageError `xml:" item"`
	}
	output.Items = []MessageError(a)
	start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{"", "xmlns:ns1"}, Value: "http://soap.entity.commons.gws"})
	output.ArrayType = "ns1:MessageError[]"
	return e.EncodeElement(&output, start)
}
func (a *Vector) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	var tok xml.Token
	for tok, err = d.Token(); err == nil; tok, err = d.Token() {
		if tok, ok := tok.(xml.StartElement); ok {
			var item MessageError
			if err = d.DecodeElement(&item, &tok); err == nil {
				*a = append(*a, item)
			}
		}
		if _, ok := tok.(xml.EndElement); ok {
			break
		}
	}
	return err
}
