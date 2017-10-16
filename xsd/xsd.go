// Package xsd parses type declarations in XML Schema documents.
//
// The xsd package implements a parser for a subset of the XML Schema
// standard. This package is intended for use in code-generation programs for
// client libraries, and as such, does not validate XML Schema documents,
// nor does it provide sufficient information to validate the documents
// described by a schema. Notably, the xsd package does not preserve
// information about element or attribute groups. Instead, all groups
// are de-referenced before parsing is done, and all nested sequences of
// elements are flattened.
//
// The xsd package respects XML name spaces in schema documents, and can
// parse schema documents that import or include other schema documents.
package xsd // import "aqwari.net/xml/xsd"

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"

	"aqwari.net/xml/xmltree"
)

const (
	schemaNS         = "http://www.w3.org/2001/XMLSchema"
	schemaInstanceNS = "http://www.w3.org/2001/XMLSchema-instance"
)

// Types in XML Schema Documents are derived from one of the built-in types
// defined by the standard, by restricting or extending the range of values
// a type may contain. A Type may be one of *SimpleType, *ComplexType,
// or Builtin.
type Type interface {
	// just for compile-time type checking
	isType()
}

// An Element describes an XML element that may appear as part of a complex
// type. Elements may have restrictions about the number of times they may
// appear and they values they may contain. The xsd package converts this
// low-level information into boolean flags where appropriate.
//
// http://www.w3.org/TR/2004/REC-xmlschema-1-20041028/structures.html#element-element
type Element struct {
	// Annotations for this element
	Doc string
	// The canonical name of this element
	Name xml.Name
	// True if this element can have any name. See
	// http://www.w3.org/TR/2004/REC-xmlschema-1-20041028/structures.html#element-any
	Wildcard bool
	// Type of this element.
	Type Type
	// An abstract type does not appear in the xml document, but
	// is "implemented" by other types in its substitution group.
	Abstract bool
	// True if maxOccurs > 1 or maxOccurs == "unbounded"
	Plural bool
	// True if the element is optional.
	Optional bool
	// If true, this element will be declared as a pointer.
	Nillable bool
	// Default overrides the zero value of this element.
	Default string
	// Any additional attributes provided in the <xs:element> element.
	Attr []xml.Attr
	// Used for resolving prefixed strings in extra attribute values.
	xmltree.Scope
}

// An Attribute describes the key=value pairs that may appear within the
// opening tag of an element. Only complex types may contain attributes.
// the Type of an Attribute can only be a Builtin or SimpleType.
//
// http://www.w3.org/TR/2004/REC-xmlschema-1-20041028/structures.html#element-attribute
type Attribute struct {
	// The canonical name of this attribute. It is uncommon for attributes
	// to have a name space.
	Name xml.Name
	// Annotation provided for this attribute by the schema author.
	Doc string
	// The type of the attribute value. Must be a simple or built-in Type.
	Type Type
	// True if this attribute has a <list> simpleType
	Plural bool
	// Default overrides the zero value of this element.
	Default string
	// True if the attribute is not required
	Optional bool
	// Any additional attributes provided in the <xs:attribute> element.
	Attr []xml.Attr
	// Used for resolving qnames in additional attributes.
	xmltree.Scope
}

// A Schema is the decoded form of an XSD <schema> element. It contains
// a collection of all types declared in the schema. Top-level elements
// are not recorded in a Schema.
type Schema struct {
	// The Target namespace of the schema. All types defined in this
	// schema will be in this name space.
	TargetNS string `xml:"targetNamespace,attr"`
	// Types defined in this schema declaration
	Types map[xml.Name]Type
	// Any annotations declared at the top-level of the schema, separated
	// by new lines.
	Doc string
}

// FindType looks for a type by its canonical name. In addition to the types
// declared in a Schema, FindType will also search through the types that
// a Schema's top-level types are derived from. FindType will return nil if
// a type could not be found with the given name.
func (s *Schema) FindType(name xml.Name) Type {
	for _, t := range s.Types {
		if t := findType(t, name); t != nil {
			return t
		}
	}
	return nil
}

func findType(t Type, name xml.Name) Type {
	if XMLName(t) == name {
		return t
	}
	if b := Base(t); b != nil {
		return findType(b, name)
	}
	return nil
}

// An XSD type can reference other types when deriving new types or
// describing elements. These types don't have to appear in-order; a type
// may be declared before its dependencies.  To handle this, we define a
// "stub" Type, which we can resolve in a second pass.
type linkedType xml.Name

func (linkedType) isType() {}

// A ComplexType describes an XML element that may contain attributes
// and elements in its content. Complex types are derived by extending
// or restricting another type. The xsd package records the elements and
// attributes that may occur in an xml element conforming to the type.
// A ComplexType is part of a linked list, through its Base field, that is
// guaranteed to end in the Builtin value AnyType.
//
// http://www.w3.org/TR/2004/REC-xmlschema-1-20041028/structures.html#element-complexType
type ComplexType struct {
	// Annotations provided by the schema author.
	Doc string
	// The canonical name of this type.
	Name xml.Name
	// The type this type is derived from.
	Base Type
	// True if this is an anonymous type
	Anonymous bool
	// XML elements that this type may contain in its content.
	Elements []Element
	// Possible attributes for the element's opening tag.
	Attributes []Attribute
	// An abstract type does not appear in the xml document, but
	// is "implemented" by other types in its substitution group.
	Abstract bool
	// If true, this type is an extension to Base.  Otherwise,
	// this type is derived by restricting the set of elements and
	// attributes allowed in Base.
	Extends bool
	// If true, this type is allowed to contain character data that is
	// not part of any sub-element.
	Mixed bool
}

func (*ComplexType) isType() {}

// A SimpleType describes an XML element that does not contain elements
// or attributes. SimpleTypes are suitable for use as attribute values.
// A SimpleType can be an "atomic" type (int, string, etc), or a list of
// atomic types, separated by white space. In addition, a SimpleType may
// be declared as a union; or one of a set of SimpleTypes. A SimpleType
// is guaranteed to be part of a linked list, through its Base field,
// that ends in a Builtin value.
//
// http://www.w3.org/TR/2004/REC-xmlschema-2-20041028/datatypes.html#element-simpleType
//
// http://www.w3.org/TR/2004/REC-xmlschema-2-20041028/datatypes.html#element-union
//
// http://www.w3.org/TR/2004/REC-xmlschema-2-20041028/datatypes.html#element-list
type SimpleType struct {
	// True if this is an anonymous type
	Anonymous bool
	// True if this type is a whitespace-delimited list, with
	// items of type Base.
	List bool
	// A simpleType may be described as a union: one of many
	// possible simpleTypes.
	Union []Type
	// Restrictions on this type's values
	Restriction Restriction
	// The canonical name of this type
	Name xml.Name
	// Any annotations for this type, as provided by the schema
	// author.
	Doc string
	// The type this type is derived from. This is guaranteed to be
	// part of a linked list that always ends in a Builtin type.
	Base Type
}

func (*SimpleType) isType() {}

// A SimpleType can be derived from a built-in or SimpleType by
// restricting the set of values it may contain. The xsd package only
// records restrictions that are useful for generating client libraries,
// and not for validating documents.
//
// http://www.w3.org/TR/2004/REC-xmlschema-2-20041028/datatypes.html#element-restriction
type Restriction struct {
	// The max digits to the right of the decimal point for
	// floating-point values.
	Precision int
	// If len(Enum) > 0, the type must be one of the values contained
	// in Enum.
	Enum []string
	// The minimum and maximum (exclusive) value of this type, if
	// numeric
	Min, Max float64
	// Maximum and minimum length (in characters) of this type
	MinLength, MaxLength int
	// Regular expression that values of this type must match
	Pattern *regexp.Regexp
	// Any annotations for the restriction, if present.
	Doc string
}

type annotation string

func (a annotation) append(extra annotation) annotation {
	if a != "" {
		a += "\n\n"
	}
	return a + extra
}

// An <xs:annotation> element may contain zero or more <xs:documentation>
// children.  The xsd package joins the content of these children, separated
// with blank lines.
func (doc *annotation) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	buf := make([][]byte, 1)
	var (
		tok xml.Token
		err error
	)

Loop:
	for {
		tok, err = d.Token()
		if err != nil {
			break
		}

		switch tok := tok.(type) {
		case xml.EndElement:
			break Loop
		case xml.StartElement:
			if (tok.Name != xml.Name{schemaNS, "documentation"}) {
				if err := d.Skip(); err != nil {
					return err
				}
			}
			var frag []byte
			if err := d.DecodeElement(&frag, &tok); err != nil {
				return err
			}
			buf = append(buf, bytes.TrimSpace(frag))
		}
	}
	*doc = annotation(bytes.TrimSpace(bytes.Join(buf, []byte("\n\n"))))
	return err
}

// XMLName returns the canonical xml name of a Type.
func XMLName(t Type) xml.Name {
	switch t := t.(type) {
	case *SimpleType:
		return t.Name
	case *ComplexType:
		return t.Name
	case Builtin:
		return t.Name()
	case linkedType:
		return xml.Name(t)
	}
	panic(fmt.Sprintf("xsd: unexpected xsd.Type %[1]T %[1]v passed to XMLName", t))
	return xml.Name{}
}

// Base returns the base type that a Type is derived from.
// If the value is of type Builtin, Base will return nil.
func Base(t Type) Type {
	switch t := t.(type) {
	case *ComplexType:
		return t.Base
	case *SimpleType:
		return t.Base
	case Builtin:
		return nil
	case linkedType:
		return nil
	}
	panic(fmt.Sprintf("xsd: unexpected xsd.Type %[1]T %[1]v passed to Base", t))
}

// The xsd package bundles a number of well-known schemas.
// These schemas are always added to the list of available schema
// when parsing an XML schema using the Parse function.
var StandardSchema = [][]byte{
	soapenc11xsd, // http://schemas.xmlsoap.org/soap/encoding/
	xmlnsxsd,     // http://www.w3.org/XML/1998/namespace
	wsdl2003xsd,  // http://schemas.xmlsoap.org/wsdl/
	xlinkxsd,     // http://www.w3.org/1999/xlink
}
