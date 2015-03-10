package xsdgen

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func glob(dir ...string) []string {
	files, err := filepath.Glob(filepath.Join(dir...))
	if err != nil {
		panic("error in glob util function: " + err.Error())
	}
	return files
}

type testLogger testing.T

func (t *testLogger) Printf(format string, v ...interface{}) {
	t.Logf(format, v...)
}

func TestGen(t *testing.T) {
	var xmlns []string

	xsdFiles := glob("testdata/*.xsd")
	for _, name := range xsdFiles {
		if data, err := ioutil.ReadFile(name); err != nil {
			t.Error(err)
		} else {
			xmlns = append(xmlns, lookupTargetNS(data)...)
		}
	}

	file, err := ioutil.TempFile("", "xsdgen")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	var cfg Config
	cfg.Option(DefaultOptions...)

	for _, ns := range xmlns {
		args := []string{"-o", file.Name(), "-ns", ns}
		err := cfg.Generate(append(args, xsdFiles...)...)
		if err != nil {
			t.Error(err)
			continue
		}
		if data, err := ioutil.ReadFile(file.Name()); err != nil {
			t.Error(err)
		} else {
			t.Logf("\n%s\n", data)
		}
	}
}
