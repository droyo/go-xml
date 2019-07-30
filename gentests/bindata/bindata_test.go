package bindata

import (
	"encoding/xml"
	"io/ioutil"
	"path/filepath"
	"testing"

	"aqwari.net/xml/xmltree"
)

func TestBindata(t *testing.T) {
	type Document struct {
		Bindata Bindata `xml:"tns bindata"`
	}
	var document Document
	samples, err := filepath.Glob(filepath.Join("*.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 1 {
		t.Fatal("expected one sample file, found ", samples)
	}
	input, err := ioutil.ReadFile(samples[0])
	if err != nil {
		t.Fatal(err)
	}
	input = append([]byte("<Document>\n"), input...)
	input = append(input, []byte("</Document>")...)
	if err := xml.Unmarshal(input, &document); err != nil {
		t.Fatal("unmarshal: ", err)
	}
	output, err := xml.Marshal(&document)
	if err != nil {
		t.Fatal("marshal: ", err)
	}
	inputTree, err := xmltree.Parse(input)
	if err != nil {
		t.Fatal("bindata: ", err)
	}
	outputTree, err := xmltree.Parse(output)
	if err != nil {
		t.Fatal("remarshal: ", err)
	}
	if !xmltree.Equal(inputTree, outputTree) {
		t.Errorf("got \n%s\n, wanted \n%s\n", xmltree.MarshalIndent(outputTree, "", "  "), xmltree.MarshalIndent(inputTree, "", "  "))
	}
}
