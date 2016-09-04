package wsdlgen

import "aqwari.net/xml/internal/gen"

// One of the goals of this package is that generated code
// has no external dependencies, only the Go standard
// library. That means we have to bundle any static
// "helper" functions along with the generated code. We
// are playing a balancing game here; the larger the static
// code base grows, the weaker the argument against external
// dependencies becomes.
var helpers string = `
	type Client struct {
		HTTPClient http.Client
		ResponseHook func(*http.Response)
		RequestHook func(*http.Request)
	}

	type soapEnvelope struct {
		XMLName struct{} ` + "`" + `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"` + "`" + `
		Header []byte ` + "`" + `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header"` + "`" + `
		Body struct {
			Message interface{}
			Fault *struct {
				String string ` + "`xml:\"faultstring,omitempty\"`" + `
				Code string ` + "`xml:\"faultcode,omitempty\"`" + `
				Detail string ` + "`xml:\"detail,omitempty\"`" + `
			} ` + "`xml:\"http://schemas.xmlsoap.org/soap/envelope/ Fault,omitempty\"`" + `
		}` + "`" + `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"` + "`" + `
	}

	func (c *Client) do(method, uri, action string, in, out interface{}) error {
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
		if c.RequestHook != nil {
			c.RequestHook(req)
		}
		rsp, err := c.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		defer rsp.Body.Close()

		if c.ResponseHook != nil {
			c.ResponseHook(rsp)
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
`

func (p *printer) addHelpers() {
	decls, err := gen.Declarations(helpers)
	if err != nil {
		// code does not change at runtime, so
		// this should never happen
		panic(err)
	}
	p.file.Decls = append(p.file.Decls, decls...)
}
