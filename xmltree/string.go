package xmltree

import (
	"bytes"
	"text/template"
)

var xmlTmpl = template.Must(template.New("Marshal XML Elements").Parse(
	`{{define "Element"}}<{{.Name.Local}}{{range .StartElement.Attr}} {{.Name.Local}}="{{.Value}}"{{end}}>{{range .Children}}{{template "Element" .}}{{else}}{{printf "%s" .Content}}{{end}}</{{.Name.Local}}>{{end}}`))

type marshalError string

// String returns an Element rendered as an XML document.
func (el *Element) String() (doc string) {
	var buf bytes.Buffer
	if err := xmlTmpl.ExecuteTemplate(&buf, "Element", el); err != nil {
		return "nil (" + err.Error() + ")"
	}
	return buf.String()
}
