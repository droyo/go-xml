package stockquote

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"testing"

	"aqwari.net/xml/internal/testutil"
	"aqwari.net/xml/xmltree"
)

func TestGetLastTradePrice(t *testing.T) {
	var (
		client Client
		input  TradePriceRequest
	)
	inputSample, err := ioutil.ReadFile("GetLastTradePrice-input.xml")
	if err != nil {
		t.Fatal(err)
	}
	if err := xml.Unmarshal(inputSample, &input); err != nil {
		t.Fatal(err)
	} else {
		t.Logf("sending %#v", input)
	}
	outputSample, err := ioutil.ReadFile("GetLastTradePrice-output.xml")
	if err != nil {
		t.Fatal(err)
	}
	want, err := xmltree.Parse(outputSample)
	if err != nil {
		t.Fatal(err)
	}
	envelope := []byte(fmt.Sprintf("<Envelope xmlns=%q><Body>%s\n</Body></Envelope>", "http://schemas.xmlsoap.org/soap/envelope/", outputSample))
	client.HTTPClient = testutil.FakeClient("http://example.com/stockquote", envelope)
	output, err := client.GetLastTradePrice(input)
	if err != nil {
		t.Fatal(err)
	}
	outputXML, err := xml.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	got, err := xmltree.Parse(outputXML)
	if err != nil {
		t.Fatal(err)
	}
	inner := want.Search(got.Name.Space, got.Name.Local)
	if len(inner) < 1 {
		t.Fatalf("got \n%s\n, want \n%s\n", got, want)
	} else {
		want = inner[0]
	}
	if !xmltree.Equal(got, want) {
		t.Errorf("got \n%s\n, want \n%s\n", got, want)
	} else {
		t.Logf("got %s", xmltree.MarshalIndent(got, "", "  "))
	}
}
