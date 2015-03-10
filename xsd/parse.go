package xsd

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"aqwari.net/xml/xmltree"
)

// A Ref contains the canonical namespace of a schema document, and
// possibly a URI to retrieve the document from. It is not required
// for XML Schema documents to provide the location of schema that
// they import; it is expected that all well-known schema namespaces
// are available to the consumer of a schema beforehand.
type Ref struct {
	Namespace, Location string
}

// Imports reads an XML document containing one or more <schema>
// elements and returns a list of canonical XML name spaces that
// the schema imports or includes, along with a URL for the schema,
// if provided.
func Imports(data []byte) ([]Ref, error) {
	var result []Ref

	root, err := xmltree.Parse(data)
	if err != nil {
		return nil, err
	}

	for _, v := range root.Search(schemaNS, "import") {
		s := Ref{v.Attr("", "namespace"), v.Attr("", "schemaLocation")}
		result = append(result, s)
	}

	var schema []*xmltree.Element
	if (root.Name == xml.Name{schemaNS, "schema"}) {
		schema = []*xmltree.Element{root}
	} else {
		schema = root.Search(schemaNS, "schema")
	}

	for _, tree := range schema {
		ns := tree.Attr("", "targetNamespace")
		for _, v := range tree.Search(schemaNS, "include") {
			s := Ref{ns, v.Attr("", "schemaLocation")}
			result = append(result, s)
		}
	}

	return result, nil
}

// Parse reads XML documents containing one or more <schema>
// elements. The returned slice has one Schema for every <schema>
// element in the documents. Parse will not fetch schema used in
// <import> or <include> statements; use the Imports function to
// find any additional schema documents required for a schema.
func Parse(docs ...[]byte) ([]Schema, error) {
	var (
		result = make([]Schema, 0, len(docs))
		schema = make(map[string]*xmltree.Element, len(docs))
		parsed = make(map[string]Schema, len(docs))
		types  = make(map[xml.Name]Type)
	)

	for _, data := range docs {
		var add []*xmltree.Element
		root, err := xmltree.Parse(data)
		if err != nil {
			return nil, err
		}
		if (root.Name == xml.Name{schemaNS, "schema"}) {
			add = []*xmltree.Element{root}
		} else {
			add = root.Search(schemaNS, "schema")
		}
		for _, s := range add {
			ns := s.Attr("", "targetNamespace")
			if v, ok := schema[ns]; !ok {
				schema[ns] = s
			} else {
				s.Children = append(s.Children, v.Children...)
				schema[ns] = s
			}
		}
	}

	for tns, root := range schema {
		s := Schema{TargetNS: tns, Types: make(map[xml.Name]Type)}
		if err := s.parse(root, schema); err != nil {
			return nil, err
		}
		parsed[tns] = s
	}

	for _, s := range parsed {
		for _, t := range s.Types {
			types[XMLName(t)] = t
		}
	}
	for _, s := range parsed {
		if err := s.resolvePartialTypes(types); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

// When working with an xml tree structure, we naturally have some
// pretty deep function calls.  To save some typing, we use panic/recover
// to bubble the errors up. These panics are not exposed to the user.
type parseError struct {
	message string
	path    []*xmltree.Element
}

func (err parseError) Error() string {
	breadcrumbs := make([]string, 0, len(err.path))
	for i := len(err.path) - 1; i >= 0; i-- {
		piece := err.path[i].Name.Local
		if name := err.path[i].Attr("", "name"); name != "" {
			piece = fmt.Sprintf("%s(%s)", piece, name)
		}
		breadcrumbs = append(breadcrumbs, piece)
	}
	return "Error at " + strings.Join(breadcrumbs, ">") + ": " + err.message
}

func stop(msg string) {
	panic(parseError{message: msg})
}

func walk(root *xmltree.Element, fn func(*xmltree.Element)) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(parseError); ok {
				err.path = append(err.path, root)
				panic(err)
			} else {
				panic(r)
			}
		}
	}()
	for i := 0; i < len(root.Children); i++ {
		// We don't care about elements outside of the
		// XML schema namespace
		if root.Children[i].Name.Space != schemaNS {
			continue
		}
		fn(&root.Children[i])
	}
}

// defer catchParseError(&err)
func catchParseError(err *error) {
	if r := recover(); r != nil {
		*err = r.(parseError)
	}
}

// Search predicates for the xmltree.Element.Search method
type predicate func(el *xmltree.Element) bool

func and(fns ...func(el *xmltree.Element) bool) predicate {
	return func(el *xmltree.Element) bool {
		for _, f := range fns {
			if !f(el) {
				return false
			}
		}
		return true
	}
}

func or(fns ...func(el *xmltree.Element) bool) predicate {
	return func(el *xmltree.Element) bool {
		for _, f := range fns {
			if f(el) {
				return true
			}
		}
		return false
	}
}

func hasChild(fn predicate) predicate {
	return func(el *xmltree.Element) bool {
		for i := range el.Children {
			if fn(&el.Children[i]) {
				return true
			}
		}
		return false
	}
}

func isElem(space, local string) predicate {
	return func(el *xmltree.Element) bool {
		if el.Name.Local != local {
			return false
		}
		return space == "" || el.Name.Space == space
	}
}

func hasAttr(space, local string) predicate {
	return func(el *xmltree.Element) bool {
		return el.Attr(space, local) != ""
	}
}

func hasAttrValue(space, local, value string) predicate {
	return func(el *xmltree.Element) bool {
		return el.Attr(space, local) == value
	}
}

var (
	isType           = or(isElem(schemaNS, "complexType"), isElem(schemaNS, "simpleType"))
	isAnonymousType  = and(isType, hasAttrValue("", "name", ""))
	hasAnonymousType = hasChild(isAnonymousType)
)

func parseType(name xml.Name) Type {
	if name.Space != schemaNS {
		return linkedType(name)
	}
	builtin, err := ParseBuiltin(name)
	if err != nil {
		stop(err.Error())
	}
	return builtin
}

func anonTypeName(n int, ns string) xml.Name {
	return xml.Name{ns, fmt.Sprintf("_anon%d", n)}
}

func (s *Schema) parse(root *xmltree.Element, extra map[string]*xmltree.Element) (err error) {
	defer catchParseError(&err)

	// First pass: name all anonymous types with placeholder names.
	var (
		typeCounter int
		updateAttr  string
		accum       bool
	)
	for _, el := range root.SearchFunc(hasAnonymousType) {
		if el.Name.Space != schemaNS {
			continue
		}
		switch el.Name.Local {
		case "element", "attribute":
			updateAttr = "type"
			accum = false
		case "list":
			updateAttr = "itemType"
			accum = false
		case "restriction":
			updateAttr = "base"
			accum = false
		case "union":
			updateAttr = "memberTypes"
			accum = true
		default:
			return fmt.Errorf("Did not expect <%s> to have an anonymous type",
				el.Prefix(el.Name))
		}
		for _, t := range el.SearchFunc(isType) {
			typeCounter++
			name := anonTypeName(typeCounter, s.TargetNS)
			qname := el.Prefix(name)

			t.SetAttr("", "name", name.Local)
			t.SetAttr("", "_isAnonymous", "true")
			if accum {
				qname = el.Attr("", updateAttr) + " " + qname
			}
			el.SetAttr("", updateAttr, qname)
			if !accum {
				break
			}
		}
	}

	// Second pass: de-reference all group/element/attribute references.
	for _, el := range root.SearchFunc(hasAttr("", "ref")) {
		var found bool
		sameType := and(isElem(el.Name.Space, el.Name.Local), hasAttr("", "name"))
		if el.Name.Space != schemaNS {
			continue
		}
		ref := el.ResolveDefault(el.Attr("", "ref"), s.TargetNS)
		for ns, doc := range extra {
			for _, real := range doc.SearchFunc(sameType) {
				name := real.ResolveDefault(real.Attr("", "name"), ns)
				if name == ref {
					extraAttr := el.StartElement.Attr
					el.Content = real.Content
					el.StartElement = real.StartElement
					el.Children = real.Children
					el.Scope = *real.JoinScope(&el.Scope)
					// In XML Schema, it is valid to
					// reference another element and
					// at the same time add attributes
					// to it.	We handle this by merging
					// the attributes of elements and
					// their references.
					for _, attr := range extraAttr {
						if attr.Name.Local == "ref" {
							continue
						}
						el.SetAttr(attr.Name.Space, attr.Name.Local, attr.Value)
					}
					found = true
					break
				}
			}
		}
		if !found {
			return fmt.Errorf("Could not dereference %s %s", el.Name.Local, el.Attr("", "ref"))
		}
	}

	// Final pass: parse all type declarations.
	if s.TargetNS == "" {
		println(root.String())
	}
	for _, el := range root.Search(schemaNS, "complexType") {
		t := s.parseComplexType(el)
		s.Types[t.Name] = t
	}
	for _, el := range root.Search(schemaNS, "simpleType") {
		t := s.parseSimpleType(el)
		s.Types[t.Name] = t
	}

	return err
}

// http://www.w3.org/TR/2004/REC-xmlschema-1-20041028/structures.html#element-complexType
func (s *Schema) parseComplexType(root *xmltree.Element) *ComplexType {
	var t ComplexType
	var doc annotation
	t.Name = root.ResolveDefault(root.Attr("", "name"), s.TargetNS)
	t.Abstract = parseBool(root.Attr("", "abstract"))

	walk(root, func(el *xmltree.Element) {
		switch el.Name.Local {
		case "annotation":
			doc = doc.append(parseAnnotation(el))
		case "simpleContent":
			t.parseSimpleContent(s.TargetNS, el)
		case "complexContent":
			t.parseComplexContent(s.TargetNS, el)
		default:
			// a complex type defined without any simpleContent or
			// complexContent is interpreted as shorthand for complex
			// content that restricts anyType.
			children := root.Children
			root.Children = nil

			restrict := *root
			restrict.StartElement = xml.StartElement{
				Name: xml.Name{schemaNS, "restriction"},
			}
			restrict.SetAttr("", "base", restrict.Prefix(AnyType.Name()))
			restrict.Children = children

			content := *root
			content.StartElement = xml.StartElement{
				Name: xml.Name{schemaNS, "complexContent"},
			}
			content.Children = []xmltree.Element{restrict}
			root.Children = []xmltree.Element{content}

			t.parseComplexContent(s.TargetNS, &content)
		}
	})
	t.Doc += string(doc)
	return &t
}

// simpleContent indicates that the content model of the new type
// contains only character data and no elements
func (t *ComplexType) parseSimpleContent(ns string, root *xmltree.Element) {
	var doc annotation
	walk(root, func(el *xmltree.Element) {
		switch el.Name.Local {
		case "annotation":
			doc = doc.append(parseAnnotation(el))
		case "restriction":
			t.Base = parseType(el.Resolve(el.Attr("", "base")))
		case "extension":
			t.Base = parseType(el.Resolve(el.Attr("", "base")))
			t.Extends = true
			for _, v := range el.Search(schemaNS, "attribute") {
				t.Attributes = append(t.Attributes, parseAttribute(ns, v))
			}
		}
	})
	t.Doc += string(doc)
}

// The complexContent element signals that we intend to restrict or extend
// the content model of a complex type.
func (t *ComplexType) parseComplexContent(ns string, root *xmltree.Element) {
	var doc annotation

	// We set this special attribute in a pre-processing step.
	t.Anonymous = (root.Attr("", "_isAnonymous") == "true")
	walk(root, func(el *xmltree.Element) {
		switch el.Name.Local {
		case "extension":
			t.Extends = true
			fallthrough
		case "restriction":
			t.Base = parseType(el.Resolve(el.Attr("", "base")))
			for _, v := range el.Search(schemaNS, "any") {
				t.Elements = append(t.Elements, parseAnyElement(ns, v))
			}
			for _, v := range el.Search(schemaNS, "element") {
				t.Elements = append(t.Elements, parseElement(ns, v))
			}
			for _, v := range el.Search(schemaNS, "attribute") {
				t.Attributes = append(t.Attributes, parseAttribute(ns, v))
			}
		case "annotation":
			doc = doc.append(parseAnnotation(el))
		default:
			stop("unexpected element " + el.Name.Local)
		}
	})
	t.Doc += string(doc)
}

func parseInt(s string) int {
	switch s {
	case "":
		return 0
	case "unbounded":
		return -1
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		stop(err.Error())
	}
	return n
}

func parseBool(s string) bool {
	switch s {
	case "", "0", "false":
		return false
	case "1", "true":
		return true
	}
	stop("Invalid boolean value " + s)
	return false
}

func parsePlural(el *xmltree.Element) bool {
	if min := parseInt(el.Attr("", "minOccurs")); min > 1 {
		return true
	} else if max := parseInt(el.Attr("", "maxOccurs")); max < 0 || max > 1 {
		return true
	}
	return false
}

func parseAnyElement(ns string, el *xmltree.Element) Element {
	var base Type = AnyType
	typeattr := el.Attr("", "type")
	if typeattr != "" {
		base = parseType(el.Resolve(typeattr))
	}
	return Element{
		Plural:   parsePlural(el),
		Type:     base,
		Wildcard: true,
	}
}

func parseElement(ns string, el *xmltree.Element) Element {
	var doc annotation
	e := Element{
		Name:     el.ResolveDefault(el.Attr("", "name"), ns),
		Type:     parseType(el.Resolve(el.Attr("", "type"))),
		Default:  el.Attr("", "default"),
		Fixed:    el.Attr("", "fixed"),
		Abstract: parseBool(el.Attr("", "abstract")),
		Nillable: parseBool(el.Attr("", "nillable")),
		Optional: (el.Attr("", "use") == "optional"),
		Plural:   parsePlural(el),
		Scope:    el.Scope,
	}

	walk(el, func(el *xmltree.Element) {
		if el.Name.Local == "annotation" {
			doc = doc.append(parseAnnotation(el))
		}
	})
	e.Doc = string(doc)
	e.Attr = el.StartElement.Attr
	return e
}

func parseAttribute(ns string, el *xmltree.Element) Attribute {
	var a Attribute
	var doc annotation
	// Non-QName xml attributes explicitly do *not* have a namespace.
	if name := el.Attr("", "name"); strings.Contains(name, ":") {
		a.Name = el.Resolve(el.Attr("", "name"))
	} else {
		a.Name.Local = name
	}
	a.Type = parseType(el.Resolve(el.Attr("", "type")))
	a.Default = el.Attr("", "default")
	a.Fixed = el.Attr("", "fixed")
	a.Scope = el.Scope

	walk(el, func(el *xmltree.Element) {
		if el.Name.Local == "annotation" {
			doc = doc.append(parseAnnotation(el))
		}
	})
	a.Doc = string(doc)
	// Other attributes could be useful later. One such attribute is
	// wsdl:arrayType.
	a.Attr = el.StartElement.Attr
	return a
}

func (s *Schema) parseSimpleType(root *xmltree.Element) *SimpleType {
	var t SimpleType
	var doc annotation

	t.Name = root.ResolveDefault(root.Attr("", "name"), s.TargetNS)
	t.Anonymous = (root.Attr("", "_isAnonymous") == "true")
	walk(root, func(el *xmltree.Element) {
		switch el.Name.Local {
		case "restriction":
			t.Base = parseType(el.Resolve(el.Attr("", "base")))
			t.Restriction = parseSimpleRestriction(el)
		case "list":
			t.Base = parseType(el.Resolve(el.Attr("", "itemType")))
			t.List = true
		case "union":
			for _, name := range strings.Fields(el.Attr("", "memberTypes")) {
				type_ := parseType(el.Resolve(name))
				t.Union = append(t.Union, type_)
			}
		case "annotation":
			doc = doc.append(parseAnnotation(el))
		}
	})
	t.Doc = string(doc)
	return &t
}

func parseAnnotation(el *xmltree.Element) (doc annotation) {
	if err := el.Unmarshal(&doc); err != nil {
		stop(err.Error())
	}
	return doc
}

func parseSimpleRestriction(root *xmltree.Element) Restriction {
	var r Restriction
	var doc annotation
	// Most of the restrictions on a simpleType are suited for
	// validating input. This package is not a validator; we assume
	// that the server sends valid data, and that it will tell us if
	// our data is wrong. As such, most of the fields here are
	// ignored.
	walk(root, func(el *xmltree.Element) {
		switch el.Name.Local {
		case "enumeration":
			r.Enum = append(r.Enum, el.Attr("", "value"))
		case "minExclusive":
			// NOTE(droyo) min/max is also valid in XSD for
			// dateTime elements. Currently, such an XSD will
			// cause an error here.
			r.Min = parseInt(el.Attr("", "value"))
		case "minInclusive":
			r.Min = parseInt(el.Attr("", "value")) - 1
		case "maxExclusive":
			r.Max = parseInt(el.Attr("", "value"))
		case "maxInclusive":
			r.Max = parseInt(el.Attr("", "value")) + 1
		case "length":
			r.MaxLength = parseInt(el.Attr("", "value"))
		case "minLength":
			r.MinLength = parseInt(el.Attr("", "value"))
		case "pattern":
			// We don't fully implement XML Schema's pattern language, and
			// we don't want to stop a parse because of this. Instead, if we
			// cannot compile a regex, we'll add the error msg to the annotation
			// for this restriction.
			pat := el.Attr("", "value")
			if r.Pattern != nil {
				pat = r.Pattern.String() + "|" + pat
			}
			reg, err := parsePattern(pat)
			if err != nil {
				msg := fmt.Sprintf("This type must conform to the pattern %q, but the XSD library could not parse the regular expression. (%v)", pat, err)
				doc = doc.append(annotation(msg))
			}
			r.Pattern = reg
		case "whiteSpace":
			break // TODO(droyo)
		case "fractionDigits":
			r.Precision = parseInt(el.Attr("", "value"))
			if r.Precision < 0 {
				stop("Invalid fractionDigits value " + el.Attr("", "value"))
			}
		case "annotation":
			doc = doc.append(parseAnnotation(el))
		}
	})
	r.Doc = string(doc)
	return r
}

// XML Schema defines its own flavor of regular expressions here:
//
// http://www.w3.org/TR/xmlschema-0/#regexAppendix
//
// For now, they are similar enough to RE2 expressions that we will
// just try and compile them as RE2 expressions. If anyone wants to
// transpile XML Schema patterns to RE2 expressions, be my guest :)
func parsePattern(pat string) (*regexp.Regexp, error) {
	return regexp.Compile(pat)
}

// Resolve all linkedTypes in a schema, so that all types are based
// on a SimpleType, ComplexType, or a Builtin. Also resolve the types
// of all Attributes and Elements.
func (s *Schema) resolvePartialTypes(types map[xml.Name]Type) error {
	for name, t := range s.Types {
		var (
			ref linkedType
			ok  bool
		)
		switch t := t.(type) {
		case Builtin:
			continue
		case *ComplexType:
			if t.Base != nil {
				ref, ok = t.Base.(linkedType)
				if ok {
					base, ok := s.lookupType(ref, types)
					if !ok {
						return fmt.Errorf("complexType %s: could not find base type %s in namespace %s",
							name.Local, ref.Local, ref.Space)
					}
					t.Base = base
				}
			}

			for i, e := range t.Elements {
				ref, ok := e.Type.(linkedType)
				if !ok {
					continue
				}
				base, ok := s.lookupType(ref, types)
				if !ok {
					return fmt.Errorf("complexType %s: could not find type %q in namespace %s for element %s",
						name.Local, ref.Local, ref.Space, e.Name.Local)
				}
				e.Type = base
				t.Elements[i] = e
			}
			for i, a := range t.Attributes {
				ref, ok := a.Type.(linkedType)
				if !ok {
					continue
				}
				base, ok := s.lookupType(ref, types)
				if !ok {
					return fmt.Errorf("complexType %s: could not find type %s in namespace %s for attribute %s",
						name.Local, ref.Local, ref.Space, a.Name.Local)
				}
				a.Type = base
				t.Attributes[i] = a
			}
		case *SimpleType:
			if t.Base != nil {
				ref, ok = t.Base.(linkedType)
				if ok {
					base, ok := s.lookupType(ref, types)
					if !ok {
						return fmt.Errorf("simpleType %s: could not find base type %s in namespace %s",
							name.Local, ref.Local, ref.Space)
					}
					t.Base = base
				}
			}

			for i, u := range t.Union {
				ref, ok := u.(linkedType)
				if !ok {
					continue
				}
				real, ok := s.lookupType(ref, types)
				if !ok {
					return fmt.Errorf("simpleType %s: could not find union memberType %s in namespace %s",
						name.Local, ref.Local, ref.Space)
				}
				t.Union[i] = real
			}
		default:
			// This should never happen, the parse function should only add
			// *SimpleType or *ComplexType to the Types map.
			panic(fmt.Sprintf("Unexpected type %s (%T) in Schema.Types map", name.Local, t))
		}
	}
	return nil
}

func (s *Schema) lookupType(name linkedType, ext map[xml.Name]Type) (Type, bool) {
	if v, ok := s.Types[xml.Name(name)]; ok {
		return v, true
	}
	v, ok := ext[xml.Name(name)]
	return v, ok
}
