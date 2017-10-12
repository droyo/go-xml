package xmltree

import (
	"bytes"
	"encoding/xml"
	"io"
	"text/template"
)

// NOTE(droyo) As of go1.5.1, the encoding/xml package does not resolve
// prefixes in attribute names. Therefore we add .Name.Space verbatim
// instead of trying to resolve it. One consequence is this is that we cannot
// rename prefixes without some work.
var tagTmpl = template.Must(template.New("Marshal XML tags").Parse(
	`{{define "start" -}}
	<{{.Scope.Prefix .Name -}}
	{{range .StartElement.Attr}} {{$.Scope.Prefix .Name -}}="{{.Value}}"{{end -}}
	{{range .NS }} xmlns{{ if .Local }}:{{ .Local }}{{end}}="{{ .Space }}"{{end}}>
	{{- end}}
	
	{{define "end" -}}
	</{{.Prefix .Name}}>{{end}}`))

// Marshal produces the XML encoding of an Element as a self-contained
// document. The xmltree package may adjust the declarations of XML
// namespaces if the Element has been modified, or is part of a larger scope,
// such that the document produced by Marshal is a valid XML document.
//
// The return value of Marshal will use the utf-8 encoding regardless of
// the original encoding of the source document.
func Marshal(el *Element) []byte {
	var buf bytes.Buffer
	if err := Encode(&buf, el); err != nil {
		// bytes.Buffer.Write should never return an error
		panic(err)
	}
	return buf.Bytes()
}

// MarshalIndent is like Marshal, but adds line breaks for each
// successive element. Each line begins with prefix and is
// followed by zero or more copies of indent according to the
// nesting depth.
func MarshalIndent(el *Element, prefix, indent string) []byte {
	var buf bytes.Buffer
	enc := encoder{
		w:      &buf,
		prefix: prefix,
		indent: indent,
		pretty: true,
	}
	if err := enc.encode(el, nil, make(map[*Element]struct{})); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// Encode writes the XML encoding of the Element to w.
// Encode returns any errors encountered writing to w.
func Encode(w io.Writer, el *Element) error {
	enc := encoder{w: w}
	return enc.encode(el, nil, make(map[*Element]struct{}))
}

// String returns the XML encoding of an Element
// and its children as a string.
func (el *Element) String() string {
	return string(Marshal(el))
}

type encoder struct {
	w              io.Writer
	prefix, indent string
	pretty         bool
}

// This could be used to print a subset of an XML document, or a document
// that has been modified. In such an event, namespace declarations must
// be "pulled" in, so they can be resolved properly. This is trickier than
// just defining everything at the top level because there may be conflicts
// introduced by the modifications.
func (e *encoder) encode(el, parent *Element, visited map[*Element]struct{}) error {
	if len(visited) > recursionLimit {
		// We only return I/O errors
		return nil
	}
	if _, ok := visited[el]; ok {
		// We have a cycle. Leave a comment, but no error
		io.WriteString(e.w, "<!-- cycle detected -->")
		return nil
	} else {
		visited[el] = struct{}{}
		defer delete(visited, el)
	}
	if e.pretty {
		io.WriteString(e.w, e.prefix)
		if len(visited) > 0 {
			io.WriteString(e.w, "\n")
		}
		for i := 0; i < len(visited); i++ {
			io.WriteString(e.w, e.indent)
		}
	}
	scope := diffScope(parent, el)
	if err := e.encodeOpenTag(el, scope); err != nil {
		return err
	}
	if len(el.Children) == 0 && len(el.Content) > 0 {
		e.w.Write(el.Content)
	}
	for i := range el.Children {
		if err := e.encode(&el.Children[i], el, visited); err != nil {
			return err
		}
	}
	if e.pretty {
		io.WriteString(e.w, "\n")
		for i := 0; i < len(visited); i++ {
			io.WriteString(e.w, e.indent)
		}
	}
	if err := e.encodeCloseTag(el); err != nil {
		return err
	}
	return nil
}

// diffScope returns the Scope of the child element, minus any
// identical namespace declaration in the parent's scope.
func diffScope(parent, child *Element) Scope {
	if parent == nil { // root element
		return child.Scope
	}
	childScope := child.Scope
	parentScope := parent.Scope
	for len(parentScope.ns) > 0 && len(childScope.ns) > 0 {
		if childScope.ns[0] == parentScope.ns[0] {
			childScope.ns = childScope.ns[1:]
			parentScope.ns = parentScope.ns[1:]
		}
		break
	}
	return childScope
}

func (e *encoder) encodeOpenTag(el *Element, scope Scope) error {
	var tag = struct {
		*Element
		NS []xml.Name
	}{Element: el, NS: scope.ns}
	return tagTmpl.ExecuteTemplate(e.w, "start", tag)
}

func (e *encoder) encodeCloseTag(el *Element) error {
	return tagTmpl.ExecuteTemplate(e.w, "end", el)
}
