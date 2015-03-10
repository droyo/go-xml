// Package xmltree converts XML documents as a tree of Go structs.
//
// The xmltree package provides routines for accessing an XML document
// as a tree, along with functionality to resolve namespace-prefixed
// strings at any point in the tree.
package xmltree // import "aqwari.net/xml/xmltree"

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

const recursionLimit = 3000

var errDeepXML = errors.New("xmltree: xml document too deeply nested")

// An Element represents a single element in an XML document. Elements
// may have zero or more children. The byte array used by the Content
// field is shared among all elements in the document, and should not
// be modified. An Element also captures xml namespace prefixes, so
// that arbitrary QNames in attribute values can be resolved.
type Element struct {
	xml.StartElement
	// The XML namespace scope at this element's location in the
	// document.
	Scope
	// The raw content contained within this element's start and
	// end tags. Uses the underlying byte array passed to Parse.
	Content []byte
	// Sub-elements contained within this element.
	Children []Element
}

// Attr gets the value of the first attribute whose name matches the
// space and local arguments. If space is the empty string, only
// attributes' local names are considered when looking for a match.
// If an attribute could not be found, the empty string is returned.
func (el *Element) Attr(space, local string) string {
	for _, v := range el.StartElement.Attr {
		if v.Name.Local != local {
			continue
		}
		if space == "" || space == v.Name.Space {
			return v.Value
		}
	}
	return ""
}

// The JoinScope method joins two Scopes together. When resolving
// prefixes using the returned scope, the prefix list in the argument
// Scope is searched before that of the receiver Scope.
func (outer *Scope) JoinScope(inner *Scope) *Scope {
	return &Scope{append(outer.ns, inner.ns...)}
}

// Unmarshal parses the XML encoding of the Element and stores the result
// in the value pointed to by v. Unmarshal follows the same rules as
// xml.Unmarshal, but only parses the portion of the XML document
// contained by the Element.
func (el *Element) Unmarshal(v interface{}) error {
	start := el.StartElement
	for _, ns := range el.ns {
		name := xml.Name{"", "xmlns"}
		if ns.Local != "" {
			name.Local += ":" + ns.Local
		}
		start.Attr = append(start.Attr, xml.Attr{name, ns.Space})
	}
	if start.Name.Space != "" {
		for i := len(el.ns) - 1; i >= 0; i-- {
			if el.ns[i].Space == start.Name.Space {
				start.Name.Space = ""
				start.Name.Local = el.ns[i].Local + ":" + start.Name.Local
				break
			}
		}
		if start.Name.Space != "" {
			return fmt.Errorf("Could not find namespace prefix for %q when decoding %s",
				start.Name.Space, start.Name.Local)
		}
	}

	var buf bytes.Buffer
	e := xml.NewEncoder(&buf)

	if err := e.EncodeToken(start); err != nil {
		return err
	}
	if err := e.Flush(); err != nil {
		return err
	}

	// BUG(droyo) The Unmarshal method unmarshals an XML fragment as it
	// was returned by the Parse method; further modifications to a tree of
	// Elements are ignored by the Unmarshal method.
	buf.Write(el.Content)
	if err := e.EncodeToken(xml.EndElement{start.Name}); err != nil {
		return err
	}
	if err := e.Flush(); err != nil {
		return err
	}
	return xml.Unmarshal(buf.Bytes(), v)
}

// A Scope represents the xml namespace scope at a given position in
// the document.
type Scope struct {
	ns []xml.Name
}

// Resolve translates an XML QName (namespace-prefixed string) to an
// xml.Name with a canonicalized namespace in its Space field.  This can
// be used when working with XSD documents, which put QNames in attribute
// values. If qname does not have a prefix, the default namespace is used.If
// a namespace prefix cannot be resolved, the returned value's Space field
// will be the unresolved prefix. Use the ResolveNS function to detect when
// a namespace prefix cannot be resolved.
func (scope *Scope) Resolve(qname string) xml.Name {
	name, _ := scope.ResolveNS(qname)
	return name
}

// The ResolveNS method is like Resolve, but returns false for its second
// return value if a namespace prefix cannot be resolved.
func (scope *Scope) ResolveNS(qname string) (xml.Name, bool) {
	var prefix, local string
	parts := strings.SplitN(qname, ":", 2)
	if len(parts) == 2 {
		prefix, local = parts[0], parts[1]
	} else {
		prefix, local = "", parts[0]
	}
	for i := len(scope.ns) - 1; i >= 0; i-- {
		if scope.ns[i].Local == prefix {
			return xml.Name{Space: scope.ns[i].Space, Local: local}, true
		}
	}
	return xml.Name{Space: prefix, Local: local}, false
}

// ResolveDefault is like Resolve, but allows for the default namespace to
// be overridden. The namespace of strings without a namespace prefix
// (known as an NCName in XML terminology) will be defaultns.
func (scope *Scope) ResolveDefault(qname, defaultns string) xml.Name {
	if defaultns == "" || strings.Contains(qname, ":") {
		return scope.Resolve(qname)
	}
	return xml.Name{defaultns, qname}
}

// Prefix is the inverse of Resolve. It uses the closest prefix
// defined for a namespace to create a string of the form
// prefix:local. If the namespace cannot be found, or is the
// default namespace, an unqualified name is returned.
func (scope *Scope) Prefix(name xml.Name) (qname string) {
	if name.Space == "" {
		return name.Local
	}
	for i := len(scope.ns) - 1; i >= 0; i-- {
		if scope.ns[i].Space == name.Space {
			if scope.ns[i].Local == "" {
				return name.Local
			}
			return scope.ns[i].Local + ":" + name.Local
		}
	}
	return name.Local
}

func (scope *Scope) pushNS(tag xml.StartElement) {
	var ns []xml.Name
	for _, attr := range tag.Attr {
		if attr.Name.Space == "xmlns" {
			ns = append(ns, xml.Name{attr.Value, attr.Name.Local})
		} else if attr.Name.Local == "xmlns" {
			ns = append(ns, xml.Name{attr.Value, ""})
		} else {
			continue
		}
	}
	if len(ns) > 0 {
		scope.ns = append(scope.ns, ns...)
		// Ensure that future additions to the scope create
		// a new backing array. This prevents the scope from
		// being clobbered during parsing.
		scope.ns = scope.ns[:len(scope.ns):len(scope.ns)]
	}
}

// Save some typing when scanning xml
type scanner struct {
	*xml.Decoder
	tok xml.Token
	err error
}

func (s *scanner) scan() bool {
	if s.err != nil {
		return false
	}
	s.tok, s.err = s.Token()
	return s.err == nil
}

// Parse builds a tree of Elements by reading an XML document.  The
// byte slice passed to Parse is expected to be a valid XML document
// with a single root element.
func Parse(doc []byte) (*Element, error) {
	d := xml.NewDecoder(bytes.NewReader(doc))
	scanner := scanner{Decoder: d}
	root := new(Element)

	for scanner.scan() {
		if start, ok := scanner.tok.(xml.StartElement); ok {
			root.StartElement = start
			break
		}
	}
	if scanner.err != nil {
		return nil, scanner.err
	}
	if err := root.parse(&scanner, doc, 0); err != nil {
		return nil, err
	}
	return root, nil
}

func (el *Element) parse(scanner *scanner, data []byte, depth int) error {
	if depth > recursionLimit {
		return errDeepXML
	}
	el.pushNS(el.StartElement)

	begin := scanner.InputOffset()
	end := begin
walk:
	for scanner.scan() {
		switch tok := scanner.tok.(type) {
		case xml.StartElement:
			child := Element{StartElement: tok.Copy(), Scope: el.Scope}
			if err := child.parse(scanner, data, depth+1); err != nil {
				return err
			}
			el.Children = append(el.Children, child)
		case xml.EndElement:
			if tok.Name != el.Name {
				return fmt.Errorf("Expecting </%s>, got </%s>", el.Prefix(el.Name), el.Prefix(tok.Name))
			}
			el.Content = data[int(begin):int(end)]
			break walk
		}
		end = scanner.InputOffset()
	}
	return scanner.err
}

// The walk method calls the walkFunc for each of the Element's children.
// If the WalkFunc returns a non-nil error, Walk will return it
// immediately.
func (el *Element) walk(fn walkFunc) error {
	for i := 0; i < len(el.Children); i++ {
		fn(&el.Children[i])
	}
	return nil
}

// SetAttr adds an XML attribute to an Element's existing Attributes.
// If the attribute already exists, it is replaced.
func (el *Element) SetAttr(space, local, value string) {
	for i, a := range el.StartElement.Attr {
		if a.Name.Local != local {
			continue
		}
		if space == "" || a.Name.Space == space {
			el.StartElement.Attr[i].Value = value
			return
		}
	}
	el.StartElement.Attr = append(el.StartElement.Attr, xml.Attr{
		Name:  xml.Name{space, local},
		Value: value,
	})
}

// walkFunc is the type of the function called for each of an Element's
// children.
type walkFunc func(*Element)

// SearchFunc traverses the Element tree in depth-first order and returns
// a slice of Elements for which the function fn returns true. Note that
// SearchFunc does not search the children of Elements that match the search;
// there is no parent-child relationship between the Elements returned in
// the result.
func (root *Element) SearchFunc(fn func(*Element) bool) []*Element {
	var results []*Element
	var search func(el *Element)

	search = func(el *Element) {
		if fn(el) {
			results = append(results, el)
		}
		el.walk(search)
	}
	root.walk(search)
	return results
}

// Search searches the Element tree for Elements with an xml tag
// matching the name and xml namespace. If space is the empty string,
// any namespace is matched.
func (root *Element) Search(space, local string) []*Element {
	return root.SearchFunc(func(el *Element) bool {
		if local != el.Name.Local {
			return false
		}
		return space == "" || space == el.Name.Space
	})
}
