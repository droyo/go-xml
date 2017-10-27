package xsd

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"aqwari.net/xml/xmltree"
)

type blob map[string]interface{}

// produces sorted keys
func keys(m map[string]blob) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (b blob) keys() []string {
	keys := make([]string, 0, len(b))
	for k := range b {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type test struct {
	actual   Schema
	expected map[string]blob
}

func (tt *test) Test(t *testing.T) {
	for _, typeName := range keys(tt.expected) {
		expected := tt.expected[typeName]
		xmlName := xml.Name{"tns", typeName}
		xsdType, ok := tt.actual.Types[xmlName]

		if !ok {
			t.Errorf("Type %q not found in Parsed schema", typeName)
			continue
		}

		// Let encoding/json do the reflection for us
		actual := unmarshal(t, marshal(t, xsdType))

		for _, field := range expected.keys() {
			want := expected[field]
			if got, ok := actual[field]; !ok {
				t.Errorf("expected %s field %q not in result",
					typeName, field)
			} else {
				testCompare(t, []string{field}, got, want)
			}
		}
	}
}

func rangeMap(m map[string]interface{}, fn func(string)) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fn(k)
	}
}

func testCompare(t *testing.T, prefix []string, got, want interface{}) bool {
	const maxDepth = 1000
	if len(prefix) > maxDepth {
		panic("max depth for type comparison exceeded")
	}
	path := strings.Join(prefix, ".")

	switch got := got.(type) {
	case []interface{}:
		w, ok := want.([]interface{})
		if !ok {
			t.Errorf("%s: got %T, want %T", path, got, w)
			return false
		}
		if len(got) != len(w) {
			t.Errorf("%s: got [%d], wanted [%d]", path, len(got), len(w))
			return false
		}
		for i := range got {
			if !testCompare(t, append(prefix, strconv.Itoa(i)), got[i], w[i]) {
				return false
			}
		}
		return true
	case map[string]interface{}:
		w, ok := want.(map[string]interface{})
		if !ok {
			t.Errorf("%s: got %T, want %T", path, got, want)
			return false
		}
		match := true
		rangeMap(w, func(key string) {
			if _, ok := got[key]; !ok {
				t.Errorf("%s: no key %s", path, key)
				keys := make([]string, 0, len(got))
				rangeMap(got, func(k string) {
					keys = append(keys, k)
				})
				t.Logf("have keys %s", strings.Join(keys, ", "))
				match = false
			} else if match {
				match = testCompare(t, append(prefix, key), got[key], w[key])
			}
		})
		return match
	default:
		switch want.(type) {
		case []interface{}, map[string]interface{}:
			t.Errorf("%s: got %T, want %T", path, got, want)
			return false
		}
	}
	if got != want {
		t.Errorf("%s: got %#v, wanted %#v", path, got, want)
	}
	return true
}

func marshal(t *testing.T, v interface{}) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func unmarshal(t *testing.T, data []byte) blob {
	var result blob
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func parseFragment(t *testing.T, filename string) (Schema, *xmltree.Element) {
	const tmpl = `<schema targetNamespace="tns" ` +
		`xmlns="http://www.w3.org/2001/XMLSchema" xmlns:tns="tns">%s</schema>`
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	doc := []byte(fmt.Sprintf(tmpl, data))
	doctrees, err := Normalize(doc)
	if err != nil {
		t.Fatalf("Failed to load schema %q: %v", filename, err)
	}

	schema, err := Parse(doc)
	if err != nil {
		t.Fatalf("Failed to Parse schema %q: %v", filename, err)
	}

	for _, s := range schema {
		if s.TargetNS == "tns" {
			for _, t := range doctrees {
				if t.Attr("", "targetNamespace") == "tns" {
					return s, t
				}
			}
		}
	}

	panic("Target schema not found")
}

func parseAnswer(t *testing.T, filename string) map[string]blob {
	result := make(map[string]blob)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to parse %q: %v", filename, err)
	}

	return result
}

func TestCases(t *testing.T) {
	names, err := filepath.Glob("testdata/*.xsd")
	if err != nil {
		t.Fatal(err)
	}

	for _, filename := range names {
		base := filename[:len(filename)-len(".xsd")]
		schema, doc := parseFragment(t, base+".xsd")
		answer := parseAnswer(t, base+".json")

		testCase := test{schema, answer}
		if !t.Run(filepath.Base(base), testCase.Test) {
			t.Logf("subtest in %s.json failed", base)
			t.Logf("normalized XSD:\n%s",
				xmltree.MarshalIndent(doc, "", "  "))
		}
	}
}
