package xsdgen

import (
	"io/ioutil"
	"os"
	"testing"
)

type testLogger testing.T

func (t *testLogger) Printf(format string, v ...interface{}) {
	t.Logf(format, v...)
}

func TestLibrarySchema(t *testing.T) {
	testGen(t, "http://dyomedea.com/ns/library", "testdata/library.xsd")
}
func TestPurchasOrderSchema(t *testing.T) {
	testGen(t, "http://www.example.com/PO1", "testdata/po1.xsd")
}
func TestUSTreasureSDN(t *testing.T) {
	testGen(t, "http://tempuri.org/sdnList.xsd", "testdata/sdn.xsd")
}
func TestSoap(t *testing.T) {
	testGen(t, "http://schemas.xmlsoap.org/soap/encoding/", "testdata/soap11.xsd")
}

func testGen(t *testing.T, ns string, files ...string) {
	file, err := ioutil.TempFile("", "xsdgen")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	var cfg Config
	cfg.Option(DefaultOptions...)
	cfg.Option(LogOutput((*testLogger)(t)))

	args := []string{"-v", "-o", file.Name(), "-ns", ns}
	err = cfg.GenCLI(append(args, files...)...)
	if err != nil {
		t.Error(err)
	}
	if data, err := ioutil.ReadFile(file.Name()); err != nil {
		t.Error(err)
	} else {
		t.Logf("\n%s\n", data)
	}
}
