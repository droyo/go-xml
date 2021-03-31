package xsd

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"aqwari.net/xml/internal/dependency"
	"aqwari.net/xml/xmltree"
)

func hasCycle(root *xmltree.Element, visited map[*xmltree.Element]struct{}) bool {
	if visited == nil {
		visited = make(map[*xmltree.Element]struct{})
	}
	visited[root] = struct{}{}
	for i := range root.Children {
		el := &root.Children[i]
		if _, ok := visited[el]; ok {
			return true
		}
		visited[el] = struct{}{}
		if hasCycle(el, visited) {
			return true
		}
	}
	delete(visited, root)
	return false
}

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

// Normalize reads XML schema documents and returns xml trees
// for each schema with the following properties:
//
// * various XSD shorthand, such as omitting <complexContent>,
//   are expanded into their canonical forms.
// * all links are dereferenced by merging the linked element.
// * all types have names. For anonymous types, unique (per
//   namespace) names of the form "_anon1", "_anon2", etc are
//   generated, and the attribute "_isAnonymous" is set to
//   "true".
//
// Because one document may contain more than one schema, the
// number of trees returned by Normalize may not equal the
// number of arguments.
func Normalize(docs ...[]byte) ([]*xmltree.Element, error) {
	docs = append(docs, StandardSchema...)
	result := make([]*xmltree.Element, 0, len(docs))

	for _, data := range docs {
		root, err := xmltree.Parse(data)
		if err != nil {
			return nil, err
		}
		if (root.Name == xml.Name{schemaNS, "schema"}) {
			result = append(result, root)
		} else {
			result = append(result, root.Search(schemaNS, "schema")...)
		}
	}
	for _, root := range result {
		attributeDefaultType(root)
		elementDefaultType(root)
		copyEltNamesToAnonTypes(root)
	}
	typeCounter := 0
	for _, root := range result {
		if err := nameAnonymousTypes(root, &typeCounter); err != nil {
			return nil, err
		}
		if err := setChoicesOptional(root); err != nil {
			return nil, err
		}
	}
	for _, root := range result {
		expandComplexShorthand(root)
	}
	if err := flattenRef(result); err != nil {
		return nil, err
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
		parsed = make(map[string]Schema, len(docs))
		types  = make(map[xml.Name]Type)
	)

	schema, err := Normalize(docs...)
	if err != nil {
		return nil, err
	}

	for _, root := range schema {
		tns := root.Attr("", "targetNamespace")
		s := Schema{TargetNS: tns, Types: make(map[xml.Name]Type)}
		if err := s.parse(root); err != nil {
			return nil, err
		}
		if existingSchema, found := parsed[tns]; found {
			err = existingSchema.MergeTypes(&s)
			if err != nil {
				return nil, fmt.Errorf("error merging types into existing schema: %v", err)
			}
		} else {
			parsed[tns] = s
		}
	}

	for _, s := range parsed {
		for _, t := range s.Types {
			types[XMLName(t)] = t
		}
	}

	for _, root := range schema {
		s := parsed[root.Attr("", "targetNamespace")]
		if err := s.resolvePartialTypes(types); err != nil {
			return nil, err
		}
		err := s.addElementTypeAliases(root, types)
		if err != nil {
			return nil, err
		}
		s.propagateMixedAttr()
		result = append(result, s)
	}
	result = append(result, builtinSchema)
	return result, nil
}

func parseType(name xml.Name) Type {
	builtin, err := ParseBuiltin(name)
	if err != nil {
		return linkedType(name)
	}
	return builtin
}

func anonTypeName(n int, ns string) xml.Name {
	return xml.Name{ns, fmt.Sprintf("_anon%d", n)}
}

/* Convert
<element name="foo">
  <complexType>
  ...
  </complexType>
</element>

to

<element name="foo" type="foo">
  <complexType name="foo">
  ...
  </complexType>
</element>
*/
func copyEltNamesToAnonTypes(root *xmltree.Element) {
	used := make(map[xml.Name]struct{})
	tns := root.Attr("", "targetNamespace")

	namedTypes := and(isType, hasAttr("", "name"))
	for _, el := range root.SearchFunc(namedTypes) {
		used[el.ResolveDefault(el.Attr("", "name"), tns)] = struct{}{}
	}

	eltWithAnonType := and(
		or(isElem(schemaNS, "element"), isElem(schemaNS, "attribute")),
		hasAttr("", "name"),
		hasAnonymousType)

	for _, el := range root.SearchFunc(eltWithAnonType) {
		// Make sure we can use this element's name
		xmlname := el.ResolveDefault(el.Attr("", "name"), tns)
		if _, ok := used[xmlname]; ok {
			continue
		}
		used[xmlname] = struct{}{}
		for i, t := range el.Children {
			if !isAnonymousType(&t) {
				continue
			}
			t.SetAttr("", "name", el.Attr("", "name"))
			el.SetAttr("", "type", el.Prefix(xmlname))

			el.Children = append(el.Children[:i], el.Children[i+1:]...)
			el.Content = nil
			root.Children = append(root.Children, t)
			break
		}
	}
}

// Inside a <xs:choice>, set all children to optional
// If child is a <xs:sequence> set its children to optional
func setChoicesOptional(root *xmltree.Element) error {

	for _, el := range root.SearchFunc(isElem(schemaNS, "choice")) {
		for i := 0; i < len(el.Children); i++ {
			t := el.Children[i]

			if t.Name.Space == schemaNS && t.Name.Local == "sequence" {
				for j := 0; j < len(t.Children); j++ {
					t2 := t.Children[j]
					t2.SetAttr("", "minOccurs", "0")
					t.Children[j] = t2
				}
			} else {
				t.SetAttr("", "minOccurs", "0")
			}

			el.Children[i] = t
		}
	}

	return nil
}

/*
Convert

  <xs:complexType name="foo" base="xs:anyType"/>
    <xs:sequence>
      <xs:element name="a">
        <xs:simpleType base="xs:int">
          ...
        </xs:simpleType>
      </xs:element>
    </xs:sequence>
  </xs:complexType>

to

  <xs:complexType name="foo" base="xs:anyType"/>
    <xs:sequence>
      <xs:element name="a">
        <xs:simpleType name="_anon1" _isAnonymous="true" base="xs:int">
          ...
        </xs:simpleType>
      </xs:element>
    </xs:sequence>
  </xs:complexType>
*/
func nameAnonymousTypes(root *xmltree.Element, typeCounter *int) error {
	var (
		updateAttr string
		accum      bool
	)
	targetNS := root.Attr("", "targetNamespace")
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
		for i := 0; i < len(el.Children); i++ {
			t := el.Children[i]

			if !isAnonymousType(&t) {
				continue
			}
			*typeCounter++

			name := anonTypeName(*typeCounter, targetNS)
			qname := el.Prefix(name)

			t.SetAttr("", "name", name.Local)
			t.SetAttr("", "_isAnonymous", "true")
			if accum {
				qname = el.Attr("", updateAttr) + " " + qname
			}
			el.SetAttr("", updateAttr, qname)
			el.Children = append(el.Children[:i], el.Children[i+1:]...)
			el.Content = nil
			root.Children = append(root.Children, t)
			if !accum {
				break
			}
		}
	}
	return nil
}

/*

Dereference all ref= links within a document.

  <attribute name="id" type="xsd:ID" />
  <complexType name="MyType">
    <attribute ref="tns:id" />
  </complexType>

becomes

  <complexType name="MyType">
    <attribute name="id" type="xsd:ID" />
  </complexType>

*/
func flattenRef(schema []*xmltree.Element) error {
	var (
		depends = new(dependency.Graph)
		index   = indexSchema(schema)
	)
	for id, el := range index.eltByID {
		if el.Attr("", "ref") == "" {
			continue
		}
		name := el.Resolve(el.Attr("", "ref"))
		if dep, ok := index.ElementID(name, el.Name); !ok {
			return fmt.Errorf("could not find ref %s in %s",
				el.Attr("", "ref"), el)
		} else {
			depends.Add(id, dep)
		}
	}
	depends.Flatten(func(id int) {
		el := index.eltByID[id]
		if el.Attr("", "ref") == "" {
			return
		}
		ref := el.Resolve(el.Attr("", "ref"))
		real, ok := index.ByName(ref, el.Name)
		if !ok {
			panic("bug building dep tree; missing " + el.Attr("", "ref"))
		}
		*el = *deref(el, real)
	})
	for ns, doc := range schema {
		unpackGroups(doc)
		if hasCycle(doc, nil) {
			return fmt.Errorf("cycle detected after flattening references "+
				"in schema %d:\n%s", ns, xmltree.MarshalIndent(doc, "", "  "))
		}
	}
	return nil
}

// Flatten a reference to an XML element, returning the full XML
// object.
func deref(ref, real *xmltree.Element) *xmltree.Element {
	el := new(xmltree.Element)
	el.Scope = ref.Scope
	el.Name = real.Name
	el.StartElement.Attr = append([]xml.Attr{}, real.StartElement.Attr...)
	el.Content = append([]byte{}, real.Content...)
	el.Children = append([]xmltree.Element{}, real.Children...)

	// Some attributes can contain a qname, and must be converted to use the
	// xmlns prefixes in ref's scope.
	hasQName := map[xml.Name]bool{
		xml.Name{"", "type"}: true,
	}
	for i, attr := range el.StartElement.Attr {
		if hasQName[attr.Name] {
			xmlname := real.Resolve(attr.Value)
			attr.Value = ref.Prefix(xmlname)
			el.StartElement.Attr[i] = attr
		}
	}
	// If there are child elements, rather than checking all children
	// for xmlns conversion, we can just import the xmlns prefixes
	if len(el.Children) > 0 {
		el.Scope = *real.JoinScope(&ref.Scope)
	}

	// Attributes added to the reference overwrite attributes in the
	// referenced element.
	for _, attr := range ref.StartElement.Attr {
		if (attr.Name != xml.Name{"", "ref"}) {
			el.SetAttr(attr.Name.Space, attr.Name.Local, attr.Value)
		}
	}

	return el
}

// After dereferencing groups and attributeGroups, we need to
// unpack them within their parent elements.
func unpackGroups(doc *xmltree.Element) {
	isGroup := or(isElem(schemaNS, "group"), isElem(schemaNS, "attributeGroup"))
	hasGroups := hasChild(isGroup)

	for _, el := range doc.SearchFunc(hasGroups) {
		children := make([]xmltree.Element, 0, len(el.Children))
		for _, c := range el.Children {
			if isGroup(&c) {
				children = append(children, c.Children...)
			} else {
				children = append(children, c)
			}
		}
		el.Children = children
	}
}

// a complex type defined without any simpleContent or
// complexContent is interpreted as shorthand for complex
// content that restricts anyType.
func expandComplexShorthand(root *xmltree.Element) {
	isComplexType := isElem(schemaNS, "complexType")

Loop:
	for _, el := range root.SearchFunc(isComplexType) {
		newChildren := make([]xmltree.Element, 0, len(el.Children))
		restrict := xmltree.Element{
			Scope:    el.Scope,
			Children: make([]xmltree.Element, 0, len(el.Children)),
		}

		for _, child := range el.Children {
			if child.Name.Space != schemaNS {
				newChildren = append(newChildren, child)
				continue
			}
			switch child.Name.Local {
			case "annotation":
				newChildren = append(newChildren, child)
				continue
			case "simpleContent", "complexContent":
				continue Loop
			}
			restrict.Children = append(restrict.Children, child)
		}
		restrict.Name.Space = schemaNS
		restrict.Name.Local = "restriction"
		restrict.SetAttr("", "base", restrict.Prefix(AnyType.Name()))

		content := xmltree.Element{
			Scope:    el.Scope,
			Children: []xmltree.Element{restrict},
		}
		content.Name.Space = schemaNS
		content.Name.Local = "complexContent"

		el.Content = nil
		el.Children = append(newChildren, content)
	}
}

func (s *Schema) addElementTypeAliases(root *xmltree.Element, types map[xml.Name]Type) error {
	for _, el := range root.Children {
		if (el.Name != xml.Name{schemaNS, "element"}) {
			continue
		}
		name := el.ResolveDefault(el.Attr("", "name"), s.TargetNS)
		ref := el.Resolve(el.Attr("", "type"))
		if ref.Local == "" || name.Local == "" {
			continue
		}
		if _, ok := s.Types[name]; !ok {
			if t, ok := s.lookupType(linkedType(ref), types); !ok {
				return fmt.Errorf("could not lookup type %s for element %s",
					el.Prefix(ref), el.Prefix(name))
			} else {
				s.Types[name] = t
			}
		}
	}
	return nil
}

// Propagate the "mixed" attribute of a type appropriately to
// all types derived from it.  For the propagation rules, see
// https://www.w3.org/TR/xmlschema-1/#coss-ct. That Definition is written
// for a computer, so I've translated the relevant portion into plain
// English. My translation may be incorrect; check the reference if you
// think so. The rules are as follows:
//
// 	- When extending a complex type, the derived type *must* be mixed iff
//    the base type is mixed.
// 	- When restricting a complex type, the derived type *may* be mixed iff
//    the base type is mixed.
// 	- The builtin "xs:anyType" is mixed.
//
// This package extends the concept of "Mixed" to apply to complex types
// with simpleContent. This is done because Mixed is used as an indicator
// that the user should care about the chardata content in a type.
func (s *Schema) propagateMixedAttr() {
	for _, t := range s.Types {
		propagateMixedAttr(t, Base(t), 0)
	}
}

func propagateMixedAttr(t, b Type, depth int) {
	const maxDepth = 1000
	if b == nil || depth > maxDepth {
		return
	}
	// Mixed attr needs to "bubble up" from the bottom, so we
	// recurse to do this backwards.
	propagateMixedAttr(b, Base(b), depth+1)

	c, ok := t.(*ComplexType)
	if !ok || c.Mixed {
		return
	}
	switch b := b.(type) {
	case Builtin:
		if b == AnyType {
			c.Mixed = c.Mixed || c.Extends
		}
	case *ComplexType:
		if c.Extends {
			c.Mixed = b.Mixed
		}
	case *SimpleType:
		c.Mixed = true
	default:
		panic(fmt.Sprintf("unexpected %T", b))
	}
}

// 3.2.2 XML Representation of Attribute Declaration Schema Components
//
// Specifies that attributes without a type default to anySimpleType.
//
// http://www.w3.org/TR/xmlschema-1/#cAttribute_Declarations
func attributeDefaultType(root *xmltree.Element) {
	var (
		isAttr    = isElem(schemaNS, "attribute")
		hasNoType = hasAttrValue("", "type", "")
		anyType   = xml.Name{Space: schemaNS, Local: "anySimpleType"}
	)
	for _, el := range root.SearchFunc(and(isAttr, hasNoType)) {
		el.SetAttr("", "type", el.Prefix(anyType))
	}
}

// 3.3.2 XML Representation of Element Declaration Schema Components
//
// Elements types default to anyType
//
// https://www.w3.org/TR/xmlschema-1/#Element_Declaration_details
func elementDefaultType(root *xmltree.Element) {
	var (
		isElement = isElem(schemaNS, "element")
		hasNoType = hasAttrValue("", "type", "")
		anyType   = xml.Name{Space: schemaNS, Local: "anyType"}
	)
	for _, el := range root.SearchFunc(and(isElement, hasNoType)) {
		el.SetAttr("", "type", el.Prefix(anyType))
	}
}

func (s *Schema) parse(root *xmltree.Element) error {
	return s.parseTypes(root)
}

func (s *Schema) parseTypes(root *xmltree.Element) (err error) {
	defer catchParseError(&err)
	tns := root.Attr("", "targetNamespace")

	for _, el := range root.Search(schemaNS, "complexType") {
		t := s.parseComplexType(el)
		s.Types[t.Name] = t
	}
	for _, el := range root.Search(schemaNS, "simpleType") {
		t := s.parseSimpleType(el)
		s.Types[t.Name] = t
	}
	s.Types[xml.Name{tns, "_self"}] = s.parseSelfType(root)
	return err
}

func (s *Schema) parseSelfType(root *xmltree.Element) *ComplexType {
	self := *root
	self.Content = nil
	self.Children = nil
	for _, el := range root.Children {
		if (el.Name == xml.Name{schemaNS, "element"}) {
			self.Children = append(self.Children, el)
		}
	}
	newdoc := self
	self.Name.Local = "complexType"
	self.SetAttr("", "name", "_self")
	newdoc.Children = []xmltree.Element{self}
	expandComplexShorthand(&newdoc)
	return s.parseComplexType(&newdoc.Children[0])
}

// http://www.w3.org/TR/2004/REC-xmlschema-1-20041028/structures.html#element-complexType
func (s *Schema) parseComplexType(root *xmltree.Element) *ComplexType {
	var t ComplexType
	var doc annotation
	t.Name = root.ResolveDefault(root.Attr("", "name"), s.TargetNS)
	t.Abstract = parseBool(root.Attr("", "abstract"))
	t.Mixed = parseBool(root.Attr("", "mixed"))

	// We set this special attribute in a pre-processing step.
	t.Anonymous = (root.Attr("", "_isAnonymous") == "true")

	walk(root, func(el *xmltree.Element) {
		switch el.Name.Local {
		case "annotation":
			doc = doc.append(parseAnnotation(el))
		case "simpleContent":
			t.parseSimpleContent(s.TargetNS, el)
		case "complexContent":
			t.parseComplexContent(s.TargetNS, el)
		default:
			stop("unexpected element " + el.Name.Local)
		}
	})
	t.Doc += string(doc)
	return &t
}

// simpleContent indicates that the content model of the new type
// contains only character data and no elements
func (t *ComplexType) parseSimpleContent(ns string, root *xmltree.Element) {
	var doc annotation

	t.Mixed = true
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
	if mixed := root.Attr("", "mixed"); mixed != "" {
		t.Mixed = parseBool(mixed)
	}
	walk(root, func(el *xmltree.Element) {
		switch el.Name.Local {
		case "extension":
			t.Extends = true
			fallthrough
		case "restriction":
			t.Base = parseType(el.Resolve(el.Attr("", "base")))

			for _, v := range el.Search(schemaNS, "any") {
				t.Elements = append(t.Elements, parseAnyElement(ns, v))
				break
			}

			usedElt := make(map[xml.Name]int)
			for _, v := range el.Search(schemaNS, "element") {
				elt := parseElement(ns, v)
				if existing, ok := usedElt[elt.Name]; !ok {
					usedElt[elt.Name] = len(t.Elements)
					t.Elements = append(t.Elements, elt)
				} else {
					t.Elements[existing] = joinElem(t.Elements[existing], elt)
				}
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

func joinElem(a, b Element) Element {
	if a.Doc != "" {
		a.Doc += "\n"
	}
	a.Doc += b.Doc
	a.Abstract = a.Abstract && b.Abstract
	a.Plural = a.Plural || b.Plural
	a.Optional = a.Optional || b.Optional
	a.Nillable = a.Nillable || b.Nillable

	if a.Default != b.Default {
		a.Default = ""
	}

	return a
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
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

// https://www.w3.org/TR/xmlschema-2/#decimal
func parseDecimal(s string) float64 {
	s = strings.TrimSpace(s)
	switch s {
	case "":
		return 0
	}
	n, err := strconv.ParseFloat(s, 64)
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
		Abstract: parseBool(el.Attr("", "abstract")),
		Nillable: parseBool(el.Attr("", "nillable")),
		Plural:   parsePlural(el),
		Scope:    el.Scope,
	}
	if el.Attr("", "type") == "" {
		e.Type = AnyType
	}
	if x := el.Attr("", "minOccurs"); x != "" && parseInt(x) == 0 {
		e.Optional = true
	} else if e.Default != "" {
		e.Optional = true
	}
	walk(el, func(el *xmltree.Element) {
		if el.Name.Local == "annotation" {
			doc = doc.append(parseAnnotation(el))
		}
	})
	t, ok := e.Type.(linkedType)
	if ok {
		e.Name.Space = t.Space
	}
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
	a.Name.Space = ns
	a.Type = parseType(el.Resolve(el.Attr("", "type")))
	a.Default = el.Attr("", "default")
	a.Scope = el.Scope
	a.Optional = el.Attr("", "use") != "required"

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
			t.Restriction = parseSimpleRestriction(el, t.Base)
		case "list":
			t.Base = parseType(el.Resolve(el.Attr("", "itemType")))
			t.List = true
		case "union":
			for _, name := range strings.Fields(el.Attr("", "memberTypes")) {
				type_ := parseType(el.Resolve(name))
				t.Union = append(t.Union, type_)
				t.Base = AnySimpleType
			}
		case "annotation":
			doc = doc.append(parseAnnotation(el))
		}
	})
	t.Doc = string(doc)
	return &t
}

func parseAnnotation(el *xmltree.Element) (doc annotation) {
	if err := xmltree.Unmarshal(el, &doc); err != nil {
		stop(err.Error())
	}
	return doc
}

func parseSimpleRestriction(root *xmltree.Element, base Type) Restriction {
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
		case "minExclusive", "minInclusive":
			r.MinDate, r.Min = parseMinMaxRestriction(el, base)
		case "maxExclusive", "maxInclusive":
			r.MaxDate, r.Max = parseMinMaxRestriction(el, base)
		case "length":
			r.Length = parseInt(el.Attr("", "value"))
		case "maxLength":
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
		case "totalDigits":
			r.TotalDigits = parseInt(el.Attr("", "value"))
		}
	})
	r.Doc = string(doc)
	return r
}

// For minInclusive, minExclusive, maxInclusive and maxExclusive restrictions
func parseMinMaxRestriction(el *xmltree.Element, base Type) (time.Time, float64) {
	var date time.Time
	var decimal float64
	if v, ok := base.(Builtin); ok {
		var format string
		if v == Date {
			format = "2006-01-02"
		} else if v == DateTime {
			format = time.RFC3339
		}

		if format != "" {
			d, err := time.Parse(format, el.Attr("", "value"))
			if err != nil {
				stop(err.Error())
			}
			date = d
		} else {
			decimal = parseDecimal(el.Attr("", "value"))
		}
	} else if _, ok := base.(linkedType); ok {
		// The base of the linked type is unknown, try to parse dates and decimals
		d, err := time.Parse("2006-01-02", el.Attr("", "value"))
		if err != nil {
			d, err := time.Parse(time.RFC3339, el.Attr("", "value"))
			if err != nil {
				decimal = parseDecimal(el.Attr("", "value"))
			}
			date = d
		}
		date = d
	} else {
		decimal = parseDecimal(el.Attr("", "value"))
	}
	return date, decimal
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
	if b, err := ParseBuiltin(xml.Name(name)); err == nil {
		return b, true
	}
	if v, ok := ext[xml.Name(name)]; ok {
		return v, true
	}
	v, ok := s.Types[xml.Name(name)]
	return v, ok
}
