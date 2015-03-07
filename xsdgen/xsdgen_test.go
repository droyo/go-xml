package xsdgen

import (
	"bytes"
	"go/format"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"testing"

	"aqwari.net/xml/xsd"
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
	var data [][]byte
	var buf bytes.Buffer
	const soapNS = "http://schemas.xmlsoap.org/soap/encoding/"

	for _, file := range glob("testdata/*") {
		r, err := ioutil.ReadFile(file)
		if err != nil {
			t.Error(err)
			continue
		}
		data = append(data, r)
	}
	schema, err := xsd.Parse(data...)
	if err != nil {
		t.Fatal(err)
	}
	var cfg Config
	cfg.Option(
		ErrorLog((*testLogger)(t), 2),
		IgnoreAttributes("offset", "id", "href"),
		ReplaceAllNames("^WS", ""),
		ReplaceAllNames(`ArrayOfsoapenc(.*)`, `${1}Array`),
		ReplaceAllNames(`ArrayOftns1WS(.*)`, `${1}Array`),
		HandleSOAPArrayType(),
		PackageName("ws"),
		SOAPArrayAsSlice(),
	)

	for _, s := range schema {
		if s.TargetNS == soapNS {
			continue
		}
		node, err := cfg.GenAST(s, schema...)
		if err != nil {
			t.Error(err)
			continue
		}
		fs := token.NewFileSet()
		if err := format.Node(&buf, fs, node); err != nil {
			t.Error(err)
			continue
		}
		t.Log(buf.String())
		buf.Reset()
	}
}
