// Package chemspell accesses the NLM ChemSpell web service.
//
package chemspell

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

type ArrayOfxsdstring []string

func (a ArrayOfxsdstring) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	var output struct {
		ArrayType string   `xml:"http://schemas.xmlsoap.org/wsdl/ arrayType,attr"`
		Items     []string `xml:" item"`
	}
	output.Items = []string(a)
	start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{"", "xmlns:ns1"}, Value: "http://www.w3.org/2001/XMLSchema"})
	output.ArrayType = "ns1:string[]"
	return e.EncodeElement(&output, start)
}
func (a *ArrayOfxsdstring) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	var tok xml.Token
	for tok, err = d.Token(); err == nil; tok, err = d.Token() {
		if tok, ok := tok.(xml.StartElement); ok {
			var item string
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

type Client struct {
	HTTPClient   *http.Client
	ResponseHook func(*http.Response) *http.Response
	RequestHook  func(*http.Request) *http.Request
}
type soapEnvelope struct {
	XMLName struct{} `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Header  []byte   `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header"`
	Body    struct {
		Message interface{}
		Fault   *struct {
			String string `xml:"faultstring,omitempty"`
			Code   string `xml:"faultcode,omitempty"`
			Detail string `xml:"detail,omitempty"`
		} `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault,omitempty"`
	} `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
}

func (c *Client) do(ctx context.Context, method, uri, action string, in, out interface{}) error {
	var body io.Reader
	var envelope soapEnvelope
	if method == "POST" || method == "PUT" {
		var buf bytes.Buffer
		envelope.Body.Message = in
		enc := xml.NewEncoder(&buf)
		if err := enc.Encode(envelope); err != nil {
			return err
		}
		if err := enc.Flush(); err != nil {
			return err
		}
		body = &buf
	}
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return err
	}
	req.Header.Set("SOAPAction", action)
	req = req.WithContext(ctx)
	if c.RequestHook != nil {
		req = c.RequestHook(req)
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	rsp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if c.ResponseHook != nil {
		rsp = c.ResponseHook(rsp)
	}
	dec := xml.NewDecoder(rsp.Body)
	envelope.Body.Message = out
	if err := dec.Decode(&envelope); err != nil {
		return err
	}
	if envelope.Body.Fault != nil {
		return fmt.Errorf("%s: %s", envelope.Body.Fault.Code, envelope.Body.Fault.String)
	}
	return nil
}
func (c *Client) GetSugList(ctx context.Context, name string, src string) (string, error) {
	var input struct {
		XMLName struct{} `xml:"http://chemspell.nlm.nih.gov/axis/SpellAid.jws/axis/SpellAid.jws getSugList"`
		Args    struct {
			Name string `xml:"http://chemspell.nlm.nih.gov/axis/SpellAid.jws/axis/SpellAid.jws name"`
			Src  string `xml:"http://chemspell.nlm.nih.gov/axis/SpellAid.jws/axis/SpellAid.jws src"`
		} `xml:"http://chemspell.nlm.nih.gov/axis/SpellAid.jws/axis/SpellAid.jws getSugListRequest"`
	}
	input.Args.Name = string(name)
	input.Args.Src = string(src)
	var output struct {
		XMLName struct{} `xml:"http://chemspell.nlm.nih.gov/axis/SpellAid.jws/axis/SpellAid.jws getSugList"`
		Args    struct {
			Return string `xml:"http://chemspell.nlm.nih.gov/axis/SpellAid.jws/axis/SpellAid.jws return"`
		} `xml:"http://chemspell.nlm.nih.gov/axis/SpellAid.jws/axis/SpellAid.jws getSugListResponse"`
	}
	err := c.do(ctx, "POST", "http://chemspell.nlm.nih.gov/axis/SpellAid.jws", "", &input, &output)
	return string(output.Args.Return), err
}
func (c *Client) Main(ctx context.Context, args ArrayOfxsdstring) error {
	var input struct {
		XMLName struct{} `xml:"http://chemspell.nlm.nih.gov/axis/SpellAid.jws/axis/SpellAid.jws main"`
		Args    struct {
			Args ArrayOfxsdstring `xml:"http://chemspell.nlm.nih.gov/axis/SpellAid.jws/axis/SpellAid.jws args"`
		} `xml:"http://chemspell.nlm.nih.gov/axis/SpellAid.jws/axis/SpellAid.jws mainRequest"`
	}
	input.Args.Args = ArrayOfxsdstring(args)
	var output struct {
		XMLName struct{} `xml:"http://chemspell.nlm.nih.gov/axis/SpellAid.jws/axis/SpellAid.jws main"`
		Args    struct{} `xml:"http://chemspell.nlm.nih.gov/axis/SpellAid.jws/axis/SpellAid.jws mainResponse"`
	}
	err := c.do(ctx, "POST", "http://chemspell.nlm.nih.gov/axis/SpellAid.jws", "", &input, &output)
	return err
}
