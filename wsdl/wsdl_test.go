package wsdl

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

func glob(pat string) []string {
	s, err := filepath.Glob(pat)
	if err != nil {
		panic(err)
	}
	return s
}

func TestParse(t *testing.T) {
	for _, filename := range glob("testdata/*.wsdl") {
		data, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Error(err)
			continue
		}
		_, err = Parse(data)
		if err != nil {
			t.Errorf("parse %s: %s", filename, err)
		}
	}
}
