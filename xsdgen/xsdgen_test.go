package xsdgen

import (
	"io/ioutil"
	"os"
	"regexp"
	"testing"
)

type testLogger testing.T

func grep(pattern, data string) bool {
	matched, err := regexp.MatchString(pattern, data)
	if err != nil {
		panic(err)
	}
	return matched
}

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
func TestSimpleStruct(t *testing.T) {
	testGen(t, "http://example.org/ns", "testdata/simple-struct.xsd")
}
func testGen(t *testing.T, ns string, files ...string) string {
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
	data, err := ioutil.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestMixedType(t *testing.T) {
	data := testGen(t, "http://example.org", "testdata/mixed-complex.xsd")
	if !grep(`PositiveNumber[^}]*,chardata`, data) {
		t.Errorf("type decl for PositiveNumber did not contain chardata, got \n%s", data)
	} else {
		t.Logf("got \n%s", data)
	}
}
