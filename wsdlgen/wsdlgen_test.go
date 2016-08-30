package wsdlgen

import (
	"io/ioutil"
	"os"
	"testing"
)

type testLogger struct {
	*testing.T
}

func (t testLogger) Printf(format string, args ...interface{}) { t.Logf(format, args...) }

func testGen(t *testing.T, files ...string) {
	file, err := ioutil.TempFile("", "wsdlgen")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	var cfg Config
	cfg.Option(DefaultOptions...)
	cfg.Option(LogOutput(testLogger{t}))

	args := []string{"-v", "-o", file.Name()}
	err = cfg.GenCLI(append(args, files...)...)
	if err != nil {
		t.Error(err)
		return
	}
	if data, err := ioutil.ReadFile(file.Name()); err != nil {
		t.Error(err)
	} else {
		t.Logf("\n%s\n", data)
	}
}

func TestNationalWeatherForecast(t *testing.T) {
	testGen(t, "../testdata/ndfdXML.wsdl")
}

func testGlobalWeather(t *testing.T) {
	testGen(t, "../testdata/webservicex-globalweather-ws.wsdl")
}
