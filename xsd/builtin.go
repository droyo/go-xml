package xsd

import (
	"encoding/xml"
	"fmt"
	"unicode"
	"unicode/utf8"
)

//go:generate stringer -type=Builtin

// A built-in represents one of the built-in xml schema types, as
// defined in the W3C specification, "XML Schema Part 2: Datatypes".
//
// http://www.w3.org/TR/xmlschema-2/#built-in-datatypes
type Builtin int

func (Builtin) isType() {}

const (
	AnyType Builtin = iota
	ENTITIES
	ENTITY
	ID
	IDREF
	IDREFS
	NCName
	NMTOKEN
	NMTOKENS
	NOTATION
	Name
	QName
	AnyURI
	Base64Binary
	Boolean
	Byte
	Date
	DateTime
	Decimal
	Double
	Duration
	Float
	GDay
	GMonth
	GMonthDay // ISO 8601 format: --MM-DD
	GYear
	GYearMonth
	HexBinary
	Int
	Integer
	Language
	Long
	NegativeInteger
	NonNegativeInteger
	NonPositiveInteger
	NormalizedString
	PositiveInteger
	Short
	String
	Time
	Token
	UnsignedByte
	UnsignedInt
	UnsignedLong
	UnsignedShort
	XMLLang  // xml:lang
	XMLSpace // xml:space
	XMLBase  // xml:base
	XMLId    // xml:id
)

// Name returns the canonical name of the built-in type. All
// built-in types are in the standard XML schema namespace,
// http://www.w3.org/2001/XMLSchema, or the XML namespace,
// http://www.w3.org/2009/01/xml.xsd
func (b Builtin) Name() xml.Name {
	name := b.String()
	space := schemaNS
	switch b {
	case ENTITIES, ENTITY, ID, IDREF, IDREFS, NCName, NMTOKEN, NMTOKENS, NOTATION, QName, Name:
	case XMLLang, XMLSpace, XMLBase, XMLId:
		space = "http://www.w3.org/2009/01/xml.xsd"
		fallthrough
	default:
		r, sz := utf8.DecodeRuneInString(name)
		name = string(unicode.ToLower(r)) + name[sz:]
	}
	return xml.Name{space, name}
}

// ParseBuiltin looks up a Builtin by name. If qname
// does not name a built-in type, ParseBuiltin returns
// a non-nil error.
func ParseBuiltin(qname xml.Name) (Builtin, error) {
	for i := AnyType; i <= UnsignedShort; i++ {
		if i.Name() == qname {
			return i, nil
		}
	}
	return -1, fmt.Errorf("xsd:%s is not a built-in", qname.Local)
}
