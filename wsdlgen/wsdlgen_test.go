package wsdlgen

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/CognitoIQ/go-xml/xsdgen"
)

type testLogger struct {
	*testing.T
}

func (t testLogger) Printf(format string, args ...interface{}) { t.Logf(format, args...) }

func testGen(t *testing.T, files ...string) {
	output_file, err := ioutil.TempFile("", "wsdlgen")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(output_file.Name())

	var cfg Config
	cfg.Option(DefaultOptions...)
	cfg.Option(LogOutput(testLogger{t}))
	cfg.xsdgen.Option(xsdgen.DefaultOptions...)
	cfg.xsdgen.Option(xsdgen.UseFieldNames())

	args := []string{"-vv", "-o", output_file.Name()}
	err = cfg.GenCLI(append(args, files...)...)
	if err != nil {
		t.Error(err)
		return
	}
	if data, err := ioutil.ReadFile(output_file.Name()); err != nil {
		t.Error(err)
	} else {
		t.Logf("\n%s\n", data)
	}
}

func TestNationalWeatherForecast(t *testing.T) {
	testGen(t, "../testdata/ndfdXML.wsdl")
}

func TestGlobalWeather(t *testing.T) {
	testGen(t, "../testdata/webservicex-globalweather-ws.wsdl")
}

func TestHello(t *testing.T) {
	testGen(t, "../testdata/hello.wsdl")
}

func TestElementWisePart(t *testing.T) {
	testGen(t, "testdata/ElementPart.wsdl")
}
