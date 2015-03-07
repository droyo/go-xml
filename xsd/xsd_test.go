package xsd

import (
	"io/ioutil"
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

func TestParse(t *testing.T) {
	for _, file := range glob("testdata/*") {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			t.Error(err)
			continue
		}
		t.Logf("Parse %s", file)

		schemaList, err := Parse(data)
		if err != nil {
			t.Error(err)
			continue
		}
		for _, s := range schemaList {
			t.Logf("Parsed namespace %q with %d types", s.TargetNS, len(s.Types))
		}
	}
}
