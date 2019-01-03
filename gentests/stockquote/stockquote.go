// Package stockquote
//
// My first service
package stockquote

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

type TradePrice struct {
	Price float32 `xml:"http://example.com/stockquote.xsd price"`
}

type TradePriceRequest struct {
	TickerSymbol string `xml:"http://example.com/stockquote.xsd tickerSymbol"`
}
type Client struct {
	HTTPClient   http.Client
	ResponseHook func(*http.Response)
	RequestHook  func(*http.Request)
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
func (c *Client) GetLastTradePrice(body TradePriceRequest) (TradePrice, error) {
	var input struct {
		XMLName struct{} `xml:"http://example.com/stockquote.wsdl GetLastTradePrice"`
		Args    struct {
			Body TradePriceRequest `xml:"http://example.com/stockquote.wsdl body"`
		} `xml:"http://example.com/stockquote.wsdl GetLastTradePriceInput"`
	}
	input.Args.Body = TradePriceRequest(body)
	var output struct {
		XMLName struct{} `xml:"http://example.com/stockquote.wsdl GetLastTradePrice"`
		Args    struct {
			Body TradePrice `xml:"http://example.com/stockquote.wsdl body"`
		} `xml:"http://example.com/stockquote.wsdl GetLastTradePriceOutput"`
	}
	err := c.do("POST", "http://example.com/stockquote", "http://example.com/GetLastTradePrice", &input, &output)
	return TradePrice(output.Args.Body), err
}
