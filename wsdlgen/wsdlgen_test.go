package wsdlgen

import (
	"errors"
	"os"
	"path"
	"testing"

	"aqwari.net/xml/xsdgen"
)

type testLogger struct {
	*testing.T
}

func (t testLogger) Printf(format string, args ...interface{}) { t.Logf(format, args...) }

func testGen(t *testing.T, files ...string) {
	output_file, err := os.CreateTemp("", "wsdlgen")
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
	data, err := os.ReadFile(output_file.Name())
	if err != nil {
		t.Error(err)
	} else {
		compareToGolden(t, data)
	}
}

func compareToGolden(t *testing.T, data []byte) {
	goldenPath := path.Join("testdata/output/", t.Name()+".xml.golden")
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err := os.WriteFile(goldenPath, data, 0644)
			if err != nil {
				t.Errorf("could not create golden file, %v", err)
			}
		} else {
			t.Errorf("could not read golden file, %v", err)
		}
		return
	}
	expectedString := string(expected)
	actualString := string(data)
	if expectedString != actualString {
		t.Error("output does not match golden file")
	}
}

func TestNationalWeatherForecast(t *testing.T) {
	testGen(t, "../testdata/ndfdXML.wsdl")
}

func TestGlobalWeather(t *testing.T) {
	testGen(t, "../testdata/webservicex-globalweather-ws.wsdl")
}

func TestGlobalWeatherPortFilter(t *testing.T) {
	testGen(t, "-port", "GlobalWeatherSoap", "../testdata/webservicex-globalweather-ws.wsdl")
}

func TestHello(t *testing.T) {
	testGen(t, "../testdata/hello.wsdl")
}

func TestElementWisePart(t *testing.T) {
	testGen(t, "testdata/ElementPart.wsdl")
}
