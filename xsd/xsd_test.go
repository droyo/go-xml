package xsd

import (
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

func glob(dir ...string) []string {
	files, err := filepath.Glob(filepath.Join(dir...))
	if err != nil {
		panic("error in glob util function: " + err.Error())
	}
	return files
}

func parseFile(t *testing.T, name string) []Schema {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		panic(err)
	}
	s, err := Parse(data)
	if err != nil {
		t.Error(err)
		t.Logf("%#v", s[0].Types)
		return nil
	}
	return s
}

func TestParse(t *testing.T) {
	for _, file := range glob("testdata/*") {
		for _, s := range parseFile(t, file) {
			t.Logf("Parsed namespace %q with %d types", s.TargetNS, len(s.Types))
		}
	}
}

func TestMixedContentModel(t *testing.T) {
	const tns = "http://example.net"
	for _, schema := range parseFile(t, "testdata/mixed.xsd") {
		if schema.TargetNS != tns {
			continue
		}
		for name, typ := range schema.Types {
			if name.Space != tns {
				continue
			}
			c, ok := typ.(*ComplexType)
			if !ok {
				t.Logf("Skipping %T %s", typ, name.Local)
				continue
			}
			if strings.HasPrefix(name.Local, "NOTMIXED") {
				if c.Mixed {
					t.Errorf("got %s Mixed=true, wanted false",
						name.Local)
				}
			} else {
				if !c.Mixed {
					t.Errorf("got %s Mixed=false, wanted true",
						name.Local)
				}
			}
		}
	}
}
