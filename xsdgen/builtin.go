package xsdgen

import (
	"go/ast"

	"aqwari.net/xml/xsd"
)

func builtinExpr(b xsd.Builtin) ast.Expr {
	if int(b) > len(builtinTbl) || b < 0 {
		return nil
	}
	return builtinTbl[b]
}

// Returns true if t is an xsd.Builtin that is not trivially mapped to a
// builtin Go type; it requires additional marshal/unmarshal methods.
func nonTrivialBuiltin(t xsd.Type) bool {
	b, ok := t.(xsd.Builtin)
	if !ok {
		return false
	}
	switch b {
	case xsd.Base64Binary, xsd.HexBinary,
		xsd.Date, xsd.Time, xsd.DateTime,
		xsd.GDay, xsd.GMonth, xsd.GMonthDay, xsd.GYear, xsd.GYearMonth:
		return true
	}
	return false
}

// The 45 built-in types of the XSD schema
var builtinTbl = []ast.Expr{
	xsd.AnyType:       &ast.Ident{Name: "string"},
	xsd.AnySimpleType: &ast.Ident{Name: "string"},
	xsd.ENTITIES:      &ast.ArrayType{Elt: &ast.Ident{Name: "string"}},
	xsd.ENTITY:        &ast.Ident{Name: "string"},
	xsd.ID:            &ast.Ident{Name: "string"},
	xsd.IDREF:         &ast.Ident{Name: "string"},
	xsd.IDREFS:        &ast.ArrayType{Elt: &ast.Ident{Name: "string"}},
	xsd.NCName:        &ast.Ident{Name: "string"},
	xsd.NMTOKEN:       &ast.Ident{Name: "string"},
	xsd.NMTOKENS:      &ast.ArrayType{Elt: &ast.Ident{Name: "string"}},
	xsd.NOTATION:      &ast.ArrayType{Elt: &ast.Ident{Name: "string"}},
	xsd.Name:          &ast.Ident{Name: "string"},
	xsd.QName:         &ast.Ident{Name: "xml.Name"},
	xsd.AnyURI:        &ast.Ident{Name: "string"},
	xsd.Base64Binary:  &ast.ArrayType{Elt: &ast.Ident{Name: "byte"}},
	xsd.Boolean:       &ast.Ident{Name: "bool"},
	xsd.Byte:          &ast.Ident{Name: "byte"},
	xsd.Date:          &ast.Ident{Name: "time.Time"},
	xsd.DateTime:      &ast.Ident{Name: "time.Time"},
	xsd.Decimal:       &ast.Ident{Name: "float64"},
	xsd.Double:        &ast.Ident{Name: "float64"},
	// the "duration" built-in is especially broken, so we
	// don't parse it at all.
	xsd.Duration:           &ast.Ident{Name: "string"},
	xsd.Float:              &ast.Ident{Name: "float32"},
	xsd.GDay:               &ast.Ident{Name: "time.Time"},
	xsd.GMonth:             &ast.Ident{Name: "time.Time"},
	xsd.GMonthDay:          &ast.Ident{Name: "time.Time"},
	xsd.GYear:              &ast.Ident{Name: "time.Time"},
	xsd.GYearMonth:         &ast.Ident{Name: "time.Time"},
	xsd.HexBinary:          &ast.ArrayType{Elt: &ast.Ident{Name: "byte"}},
	xsd.Int:                &ast.Ident{Name: "int"},
	xsd.Integer:            &ast.Ident{Name: "int"},
	xsd.Language:           &ast.Ident{Name: "string"},
	xsd.Long:               &ast.Ident{Name: "int64"},
	xsd.NegativeInteger:    &ast.Ident{Name: "int"},
	xsd.NonNegativeInteger: &ast.Ident{Name: "int"},
	xsd.NormalizedString:   &ast.Ident{Name: "string"},
	xsd.NonPositiveInteger: &ast.Ident{Name: "int"},
	xsd.PositiveInteger:    &ast.Ident{Name: "int"},
	xsd.Short:              &ast.Ident{Name: "int"},
	xsd.String:             &ast.Ident{Name: "string"},
	xsd.Time:               &ast.Ident{Name: "time.Time"},
	xsd.Token:              &ast.Ident{Name: "string"},
	xsd.UnsignedByte:       &ast.Ident{Name: "byte"},
	xsd.UnsignedInt:        &ast.Ident{Name: "uint"},
	xsd.UnsignedLong:       &ast.Ident{Name: "uint64"},
	xsd.UnsignedShort:      &ast.Ident{Name: "uint"},
}
