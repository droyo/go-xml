package xsdgen // import "aqwari.net/xml/xsdgen"

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"sort"
	"strconv"
	"strings"

	"aqwari.net/xml/internal/dependency"
	"aqwari.net/xml/internal/gen"
	"aqwari.net/xml/xmltree"
	"aqwari.net/xml/xsd"
)

type orderedStringMap interface {
	keys() []string
}

func rangeMap(m orderedStringMap, fn func(string)) {
	keys := m.keys()
	sort.Strings(keys)
	for _, key := range keys {
		fn(key)
	}
}

type errorList []error

func (l errorList) Error() string {
	var buf bytes.Buffer
	for _, err := range l {
		io.WriteString(&buf, err.Error()+"\n")
	}
	return buf.String()
}

func lookupTargetNS(data ...[]byte) []string {
	var result []string
	for _, doc := range data {
		tree, err := xmltree.Parse(doc)
		if err != nil {
			continue
		}
		outer := xmltree.Element{
			Children: []*xmltree.Element{tree},
		}
		elts := outer.Search("http://www.w3.org/2001/XMLSchema", "schema")
		for _, el := range elts {
			ns := el.Attr("", "targetNamespace")
			if ns != "" {
				result = append(result, ns)
			}
		}
	}
	return result
}

type specListing map[string]spec

func (m specListing) keys() (result []string) {
	for k := range m {
		result = append(result, k)
	}
	return result
}

// Code is the intermediate representation used by the xsdgen
// package. It can be used to generate a Go source file, and to
// lookup identifiers and attributes for a given type.
type Code struct {
	cfg   *Config
	names map[xml.Name]string
	decls specListing
	types map[xml.Name]xsd.Type
}

// DocType retrieves the complexType for the provided target
// namespace.
func (c *Code) DocType(targetNS string) (*xsd.ComplexType, bool) {
	key := xml.Name{targetNS, "_self"}
	doc, ok := c.types[key].(*xsd.ComplexType)
	return doc, ok
}

// NameOf returns the Go identifier associated with the canonical
// XML type.
func (c *Code) NameOf(name xml.Name) string {
	c.cfg.debugf("looking up Go name for %v", name)
	if id, ok := c.names[name]; ok {
		c.cfg.debugf("%v -> %s", name, id)
		return id
	}

	if b, err := xsd.ParseBuiltin(name); err == nil {
		s, err := gen.ToString(builtinExpr(b))
		if err != nil {
			return "ERROR" + name.Local
		}
		c.names[name] = s
		return s
	}

	t, ok := c.types[name]
	if !ok {
		return "NOTFOUND" + name.Local
	}

	switch b := c.cfg.flatten1(t, func(xsd.Type) {}, 0).(type) {
	case xsd.Builtin:
		return c.NameOf(b.Name())
	}

	specs, err := c.cfg.genTypeSpec(t)
	if err != nil {
		c.cfg.logf("error generating type for %s: %s", name.Local, err)
		return "ERROR" + name.Local
	}

	// memoize them for later
	for _, s := range specs {
		c.names[xsd.XMLName(s.xsdType)] = s.name
	}
	if s, ok := c.names[name]; ok {
		return s
	}
	return "NOTFOUND" + name.Local
}

func (cfg *Config) gen(primaries, deps []xsd.Schema) (*Code, error) {
	var errList errorList

	code := &Code{
		cfg:   cfg,
		names: make(map[xml.Name]string),
		decls: make(specListing),
	}

	all := make(map[xml.Name]xsd.Type)
	for _, primary := range primaries {
		for k, v := range primary.Types {
			all[k] = v
		}
	}
	for _, dep := range deps {
		for k, v := range dep.Types {
			all[k] = v
		}
	}

	code.types = all
	if cfg.preprocessType != nil {
		cfg.debugf("running user-defined pre-processing functions")
		for i, primary := range primaries {
			// So user code has visibility into all types
			prev := primary.Types
			primary.Types = all

			for name, t := range prev {
				if t := cfg.preprocessType(primary, t); t != nil {
					prev[name] = t
				}
			}

			// So we don't pull in type expressions for
			// unused dependencies
			primaries[i].Types = prev
		}
	}

	for _, primary := range primaries {
		cfg.debugf("flattening type hierarchy for schema %q", primary.TargetNS)
		types := cfg.flatten(primary.Types)
		types = cfg.expandComplexTypes(types)
		for _, t := range types {
			specs, err := cfg.genTypeSpec(t)
			if err != nil {
				errList = append(errList, fmt.Errorf("gen type %q: %v",
					xsd.XMLName(t).Local, err))
			} else {
				for _, s := range specs {
					code.names[xsd.XMLName(s.xsdType)] = s.name
					code.decls[s.name] = s
				}
			}
		}
	}

	if len(errList) > 0 {
		return nil, errList
	}

	if cfg.postprocessType != nil {
		cfg.debugf("running user-defined post-processing functions")
		for name, s := range code.decls {
			code.decls[name] = cfg.postprocessType(s)
		}
	}

	for t, s := range code.decls {
		cfg.debugf("processing dependencies for type %v", t)
		for _, dep := range s.helperTypes {
			if h, ok := cfg.helperTypes[dep]; ok {
				code.decls[h.name] = h
				delete(cfg.helperTypes, dep)
			}
		}
	}
	rangeMap(code.decls, func(t string) {
		s := code.decls[t]
		for _, dep := range s.helperFuncs {
			if h, ok := cfg.helperFuncs[dep]; ok {
				cfg.debugf("adding helper function %v for type %v", dep, t)
				s.methods = append(s.methods, h)
				code.decls[t] = s
				delete(cfg.helperFuncs, dep)
			}
		}
	})
	return code, nil
}

// GenAST generates a Go abstract syntax tree with
// the type declarations contained in the xml schema document.
func (code *Code) GenAST() (*ast.File, error) {
	var file ast.File

	keys := make([]string, 0, len(code.decls))
	for name := range code.decls {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		info := code.decls[name]
		typeDecl := &ast.GenDecl{
			Doc: gen.CommentGroup(info.doc),
			Tok: token.TYPE,
			Specs: []ast.Spec{
				&ast.TypeSpec{
					Name: ast.NewIdent(name),
					Type: info.expr,
				},
			},
		}
		file.Decls = append(file.Decls, typeDecl)
		for _, f := range info.methods {
			file.Decls = append(file.Decls, f)
		}
	}
	pkgname := code.cfg.pkgname
	if pkgname == "" {
		pkgname = "ws"
	}
	file.Name = ast.NewIdent(pkgname)
	return &file, nil
}

type spec struct {
	name, doc   string
	expr        ast.Expr
	private     bool
	methods     []*ast.FuncDecl
	xsdType     xsd.Type
	helperTypes []xml.Name
	helperFuncs []string
}

// Simplifies complex types derived from other complex types by merging
// parent fields into the derived type.
func (cfg *Config) expandComplexTypes(types []xsd.Type) []xsd.Type {
	index := make(map[xml.Name]int)
	graph := new(dependency.Graph)

	for i, v := range types {
		index[xsd.XMLName(v)] = i
	}
	alltypes := types
	for _, v := range types {
		for base := xsd.Base(v); base != nil; base = xsd.Base(base) {
			xmlname := xsd.XMLName(base)
			if _, ok := index[xmlname]; !ok {
				index[xmlname] = len(alltypes)
				alltypes = append(alltypes, base)
			}
		}
	}

	for i, v := range alltypes {
		c, ok := v.(*xsd.ComplexType)
		if !ok {
			continue
		}
		if b, ok := c.Base.(*xsd.ComplexType); ok {
			if _, ok := index[b.Name]; !ok {
				// should never happen
				panic(fmt.Errorf("missing base type for %v.", c.Name))
			}
			graph.Add(i, index[b.Name])
		}
	}

	graph.Flatten(func(i int) {
		c := alltypes[i].(*xsd.ComplexType)
		b, ok := c.Base.(*xsd.ComplexType)
		if !ok {
			return
		}

		cfg.debugf("complexType %s: expanding base %s fields",
			c.Name.Local, b.Name.Local)

		shadowedElements := make(map[xml.Name]struct{})
		shadowedAttributes := make(map[xml.Name]struct{})

		for _, el := range c.Elements {
			shadowedElements[el.Name] = struct{}{}
		}
		for _, attr := range c.Attributes {
			shadowedAttributes[attr.Name] = struct{}{}
		}

		elements := []*xsd.Element{}
		for _, el := range b.Elements {
			if _, ok := shadowedElements[el.Name]; !ok {
				elements = append(elements, el)
			} else {
				cfg.debugf("complexType %s: extended element %s is overrided",
					c.Name.Local, el.Name.Local)
			}
		}
		c.Elements = append(elements, c.Elements...)
		for _, attr := range b.Attributes {
			if _, ok := shadowedAttributes[attr.Name]; !ok {
				c.Attributes = append(c.Attributes, attr)
			} else {
				cfg.debugf("complexType %s: extended attribute %s is overrided",
					c.Name.Local, attr.Name.Local)
			}
		}

		for base := c.Base; base != nil; base = xsd.Base(base) {
			if _, ok := base.(*xsd.ComplexType); !ok {
				break
			}
		}
		c.Extends = false

		alltypes[i] = c
	})
	return types
}

// To reduce the size of the Go source generated, all intermediate types
// are "squashed"; every type should be based on a Builtin or another
// type that the user wants included in the Go source. In affect, what we
// want to do is take the linked list:
//
// 	t1 -> t2 -> t3 -> builtin
//
// And produce a set of tuples:
//
// 	t1 -> builtin, t2 -> builtin, t3 -> builtin
//
// This is a heuristic that tends to generate better-looking Go code.
func (cfg *Config) flatten(types map[xml.Name]xsd.Type) []xsd.Type {
	var result []xsd.Type
	push := func(t xsd.Type) {
		result = append(result, t)
	}
	for _, t := range types {
		if xsd.XMLName(t).Local == "_self" {
			continue
		}
		cfg.debugf("flattening type %T(%s)\n", t, xsd.XMLName(t).Local)
		if cfg.filterTypes != nil && cfg.filterTypes(t) {
			continue
		}
		if cfg.allowTypes != nil {
			if !cfg.allowTypes[xsd.XMLName(t)] {
				continue
			}
		}
		if t := cfg.flatten1(t, push, 0); t != nil {
			push(t)
		}
	}
	return dedup(result)
}

func dedup(types []xsd.Type) (unique []xsd.Type) {
	seen := make(map[xml.Name]bool)
	for _, v := range types {
		if name := xsd.XMLName(v); !seen[name] {
			seen[name] = true
			unique = append(unique, v)
		}
	}
	return unique
}

func (cfg *Config) flatten1(t xsd.Type, push func(xsd.Type), depth int) xsd.Type {
	const maxDepth = 1000
	if depth > maxDepth {
		return t
	}
	switch t := t.(type) {
	case *xsd.SimpleType:
		var (
			chain         []xsd.Type
			base, builtin xsd.Type
			ok            bool
		)
		// TODO: handle list/union types
		for base = xsd.Base(t); base != nil; base = xsd.Base(base) {
			if builtin, ok = base.(xsd.Builtin); ok {
				break
			}
			chain = append(chain, base)
		}
		for _, v := range chain {
			if v, ok := v.(*xsd.SimpleType); ok {
				v.Base = builtin
				push(v)
			}
		}
		t.Base = builtin

		if t.Base != nil {
			cfg.debugf("%T(%s) -> %T(%s)", t, xsd.XMLName(t).Local,
				t.Base, xsd.XMLName(t.Base).Local)
		}

		// NOTE(droyo) if a simpleType does not impose any(many?) restrictions
		// on a builtin type, and does not require its own XML marshal/unmarshal
		// methods, it does not really add any value, as we can just use the builtin
		// type (which maps directly to a Go type). So a simpleType must prove it
		// is useful enough for its own Go type. Our threshold for "useful enough"
		// is pretty low; if we can attach a godoc comment to it describing how it
		// should be used, that's good enough.
		if t.List || len(t.Union) > 0 {
			return t
		}
		if nonTrivialBuiltin(t.Base) {
			return t
		}
		if len(t.Restriction.Enum) > 0 {
			t.Doc = "May be one of " + strings.Join(t.Restriction.Enum, ", ")
			return t
		}
		if t.Restriction.Pattern != nil {
			t.Doc = "Must match the pattern " + t.Restriction.Pattern.String()
			return t
		}
		if t.Restriction.MaxLength != 0 {
			t.Doc = "May be no more than " + strconv.Itoa(t.Restriction.MaxLength) + " items long"
			return t
		}
		if t.Restriction.MinLength != 0 {
			t.Doc = "Must be at least " + strconv.Itoa(t.Restriction.MinLength) + " items long"
			return t
		}
		return t.Base
	case *xsd.ComplexType:
		// We can "unpack" a struct if it is extending a simple
		// or built-in type and we are ignoring all of its attributes.
		switch t.Base.(type) {
		case xsd.Builtin, *xsd.SimpleType:
			if b, ok := t.Base.(xsd.Builtin); ok && b == xsd.AnyType {
				break
			}
			attributes, _ := cfg.filterFields(t)
			if len(attributes) == 0 {
				cfg.debugf("complexType %s extends simpleType %s, but extra attributes are filtered. unpacking.",
					t.Name.Local, xsd.XMLName(t.Base))
				switch b := t.Base.(type) {
				case xsd.Builtin:
					return b
				case *xsd.SimpleType:
					return cfg.flatten1(t.Base, push, depth+1)
				}
			}
		}
		// We can flatten a struct field if its type does not
		// need additional methods for unmarshalling.
		for i, el := range t.Elements {
			el.Type = cfg.flatten1(el.Type, push, depth+1)
			t.Elements[i] = el
			push(el.Type)
			cfg.debugf("element %s %T(%s): %v", el.Name.Local, t,
				xsd.XMLName(t).Local, xsd.XMLName(el.Type))
		}
		for i, attr := range t.Attributes {
			attr.Type = cfg.flatten1(attr.Type, push, depth+1)
			t.Attributes[i] = attr
			push(attr.Type)
			cfg.debugf("attribute %s %T(%s): %v", attr.Name.Local, t,
				xsd.XMLName(t).Local, xsd.XMLName(attr.Type))
		}

		t.Base = cfg.flatten1(t.Base, push, depth+1)

		// We expand all complexTypes to merge all of the fields from
		// their ancestors, so generated code has no dependencies
		// on ancestor types. The only exception is when the type
		// has a mixed content model; we need to pull in the type
		// of its chardata.
		if t.Mixed {
			push(t.Base)
		}

		cfg.debugf("%T(%s) -> %T(%s)", t, xsd.XMLName(t).Local,
			t.Base, xsd.XMLName(t.Base).Local)

		return t
	case xsd.Builtin:
		return t
	}
	panic(fmt.Errorf("unexpected %T(%s %s)", t, xsd.XMLName(t).Space, xsd.XMLName(t).Local))
}

func (cfg *Config) genTypeSpec(t xsd.Type) (result []spec, err error) {
	var s []spec
	cfg.debugf("generating type spec for %q", xsd.XMLName(t).Local)

	switch t := t.(type) {
	case *xsd.SimpleType:
		s, err = cfg.genSimpleType(t)
	case *xsd.ComplexType:
		s, err = cfg.genComplexType(t)
	case xsd.Builtin:
		// pass
	default:
		cfg.logf("unexpected %T %s", t, xsd.XMLName(t).Local)
	}
	if err != nil || s == nil {
		return result, err
	}
	return append(result, s...), nil
}

type fieldOverride struct {
	FieldName        string
	FromType, ToType string
	DefaultValue     string
	Type             xsd.Type
	Tag              string
}

type nameGenerator struct {
	cfg   *Config
	taken map[string]struct{}
}

func (gen *nameGenerator) unique(name string) ast.Expr {
	const maxDupNames = 1000
	if _, ok := gen.taken[name]; ok {
		for i := 0; i < maxDupNames; i++ {
			unique := name + strconv.Itoa(i)
			if _, ok := gen.taken[unique]; !ok {
				gen.taken[unique] = struct{}{}
				return ast.NewIdent(unique)
			}
		}
	} else {
		gen.taken[name] = struct{}{}
	}
	return ast.NewIdent(name)
}

func (gen *nameGenerator) attribute(base xml.Name) ast.Expr {
	name := gen.cfg.public(base)
	if _, ok := gen.taken[name]; !ok {
		gen.taken[name] = struct{}{}
		return ast.NewIdent(name)
	}
	return gen.unique(name + "Attr")
}

func (gen *nameGenerator) element(base xml.Name) ast.Expr {
	name := gen.cfg.public(base)
	if _, ok := gen.taken[name]; !ok {
		gen.taken[name] = struct{}{}
		return ast.NewIdent(name)
	}
	return gen.unique(name)
}

func (cfg *Config) genComplexType(t *xsd.ComplexType) ([]spec, error) {
	var result []spec
	var fields []ast.Expr
	var overrides []fieldOverride
	var helperTypes []xml.Name

	namegen := nameGenerator{cfg, make(map[string]struct{})}

	if t.Mixed {
		// For complex types with mixed content models, we must drill
		// down to the base simple or builtin type to determine the
		// ",chardata" struct field.
		base := xsd.Base(t)
		for xsd.Base(base) != nil {
			if _, ok := base.(*xsd.SimpleType); ok {
				break
			}
			base = xsd.Base(base)
		}
		expr, err := cfg.expr(base)
		if err != nil {
			return nil, fmt.Errorf("%s base type %s: %v",
				t.Name.Local, xsd.XMLName(t.Base).Local, err)
		}
		switch b := base.(type) {
		case *xsd.SimpleType:
			cfg.debugf("complexType %[1]s extends simpleType %[2]s. Naming"+
				" the chardata struct field after %[2]s", t.Name.Local, b.Name.Local)
			fields = append(fields, expr, expr, gen.String(`xml:",chardata"`))
		case xsd.Builtin:
			if b == xsd.AnyType {
				// extending anyType doesn't really make sense, but
				// we can just ignore it.
				cfg.debugf("complexType %s: don't know how to extend anyType, ignoring",
					t.Name.Local)
				break
			}
			// Name the field after the xsd type name.
			cfg.debugf("complexType %[1]s extends %[2]s, naming chardata struct field %[2]s",
				t.Name.Local, b)
			name := "Value"
			tag := `xml:",chardata"`
			if nonTrivialBuiltin(b) {
				h, ok := cfg.helperTypes[xsd.XMLName(b)]
				if !ok {
					return nil, fmt.Errorf("missing helper type for %v", b)
				}
				helperTypes = append(helperTypes, xsd.XMLName(h.xsdType))
				overrides = append(overrides, fieldOverride{
					FieldName: name,
					FromType:  cfg.exprString(b),
					Tag:       tag,
					ToType:    h.name,
					Type:      b,
				})
			}
			fields = append(fields, namegen.unique(name), expr, gen.String(tag))
		default:
			panic(fmt.Errorf("%s does not derive from a builtin type", t.Name.Local))
		}
	}

	// When restricting a complex type, all attributes are "inherited" from
	// the base type (but not elements!). In addition, any <xs:any> elements,
	// while not explicitly inherited, do not disappear.
	switch b := t.Base.(type) {
	case *xsd.ComplexType:
		t.Attributes = mergeAttributes(t, b)
		hasWildcard := false
		for _, el := range t.Elements {
			if el.Wildcard {
				hasWildcard = true
				break
			}
		}
		if hasWildcard {
			break
		}
		for _, el := range b.Elements {
			if el.Wildcard {
				t.Elements = append(t.Elements, el)
				break
			}
		}
	}

	attributes, elements := cfg.filterFields(t)
	cfg.debugf("complexType %s: generating struct fields for %d elements and %d attributes",
		xsd.XMLName(t).Local, len(elements), len(attributes))

	for _, el := range elements {
		options := ""
		if el.Nillable || el.Optional {
			options = ",omitempty"
		}
		tag := fmt.Sprintf(`xml:"%s %s%s"`, el.Name.Space, el.Name.Local, options)
		base, err := cfg.expr(el.Type)
		if err != nil {
			return nil, fmt.Errorf("%s element %s: %v", t.Name.Local, el.Name.Local, err)
		}
		name := namegen.element(el.Name)
		if el.Wildcard {
			tag = `xml:",any"`
			if el.Plural {
				name = ast.NewIdent("Items")
			} else {
				name = ast.NewIdent("Item")
			}
			if b, ok := el.Type.(xsd.Builtin); ok && b == xsd.AnyType {
				cfg.debugf("complexType %s: defaulting wildcard element to []string", t.Name.Local)
				base = builtinExpr(xsd.String)
			}
		}
		if el.Plural {
			base = &ast.ArrayType{Elt: base}
		}
		fields = append(fields, name, base, gen.String(tag))
		if el.Default != "" || nonTrivialBuiltin(el.Type) {
			typeName := cfg.exprString(el.Type)
			if nonTrivialBuiltin(el.Type) {
				h, ok := cfg.helperTypes[xsd.XMLName(el.Type)]
				if !ok {
					return nil, fmt.Errorf("no helper type for type %v element %v", t.Name, el.Name)
				}
				helperTypes = append(helperTypes, xsd.XMLName(h.xsdType))
				typeName = h.name
			}
			overrides = append(overrides, fieldOverride{
				DefaultValue: el.Default,
				FieldName:    name.(*ast.Ident).Name,
				FromType:     cfg.exprString(el.Type),
				Tag:          tag,
				ToType:       typeName,
				Type:         el.Type,
			})
		}
	}
	for _, attr := range attributes {
		options := ""
		if attr.Optional {
			options = ",omitempty"
		}
		qualified := false
		for _, attrAttr := range attr.Attr {
			if attrAttr.Name.Space == "" && attrAttr.Name.Local == "form" && attrAttr.Value == "qualified" {
				qualified = true
			}
		}
		var tag string
		if qualified {
			tag = fmt.Sprintf(`xml:"%s %s,attr%s"`, attr.Name.Space, attr.Name.Local, options)
		} else {
			tag = fmt.Sprintf(`xml:"%s,attr%s"`, attr.Name.Local, options)
		}
		base, err := cfg.expr(attr.Type)
		if err != nil {
			return nil, fmt.Errorf("%s attribute %s: %v", t.Name.Local, attr.Name.Local, err)
		}
		cfg.debugf("adding %s attribute %s as %v", t.Name.Local, attr.Name.Local, base)
		name := namegen.attribute(attr.Name)
		fields = append(fields, name, base, gen.String(tag))
		if attr.Default != "" || nonTrivialBuiltin(attr.Type) {
			typeName := cfg.exprString(attr.Type)
			if nonTrivialBuiltin(attr.Type) {
				h, ok := cfg.helperTypes[xsd.XMLName(attr.Type)]
				if !ok {
					return nil, fmt.Errorf("no helper type for type %v attribute %v", t.Name, attr.Name)
				}
				typeName = h.name
				helperTypes = append(helperTypes, xsd.XMLName(attr.Type))
			}
			overrides = append(overrides, fieldOverride{
				DefaultValue: attr.Default,
				FieldName:    name.(*ast.Ident).Name,
				FromType:     cfg.exprString(attr.Type),
				Tag:          tag,
				ToType:       typeName,
				Type:         attr.Type,
			})
		}
	}
	expr := gen.Struct(fields...)
	s := spec{
		doc:         t.Doc,
		name:        cfg.public(t.Name),
		expr:        expr,
		xsdType:     t,
		helperTypes: helperTypes,
	}
	if len(overrides) > 0 {
		unmarshal, marshal, err := cfg.genComplexTypeMethods(t, overrides)
		if err != nil {
			return result, err
		} else {
			if unmarshal != nil {
				s.methods = append(s.methods, unmarshal)
			}
			if marshal != nil {
				s.methods = append(s.methods, marshal)
			}
		}
	}
	result = append(result, s)
	return result, nil
}

func (cfg *Config) genComplexTypeMethods(t *xsd.ComplexType, overrides []fieldOverride) (marshal, unmarshal *ast.FuncDecl, err error) {
	var data struct {
		Overrides []fieldOverride
		Type      string
	}
	data.Overrides = overrides
	data.Type = cfg.public(t.Name)

	unmarshal, err = gen.Func("UnmarshalXML").
		Receiver("t *"+data.Type).
		Args("d *xml.Decoder", "start xml.StartElement").
		Returns("error").
		BodyTmpl(`
			type T {{.Type}}
			var overlay struct{
				*T
				{{range .Overrides}}
				{{.FieldName}} *{{.ToType}} `+"`{{.Tag}}`"+`
				{{end}}
			}
			overlay.T = (*T)(t)
			{{range .Overrides}}
			overlay.{{.FieldName}} = (*{{.ToType}})(&overlay.T.{{.FieldName}})
			{{if .DefaultValue -}}
			// overlay.{{.FieldName}} = {{.DefaultValue}}
			{{end -}}
			{{end}}

			return d.DecodeElement(&overlay, &start)
		`, data).Decl()
	if err != nil {
		return nil, nil, err
	}

	// We don't set defaults in MarshalXML; there's no way to distinguish
	// an intentional zero value from "no value", and the consumer of the
	// XML should know what the default is from the XSD.
	nonDefaultOverrides := make([]fieldOverride, 0, len(overrides))
	for _, v := range overrides {
		if nonTrivialBuiltin(v.Type) {
			nonDefaultOverrides = append(nonDefaultOverrides, v)
		}
	}
	if len(nonDefaultOverrides) == 0 {
		return nil, unmarshal, nil
	}

	data.Overrides = nonDefaultOverrides
	marshal, err = gen.Func("MarshalXML").
		Receiver("t *"+data.Type).
		Args("e *xml.Encoder", "start xml.StartElement").
		Returns("error").
		BodyTmpl(`
			type T {{.Type}}
			var layout struct{
				*T
				{{- range .Overrides}}
				{{.FieldName}} *{{.ToType}}`+"`{{.Tag}}`"+`
				{{end -}}
			}
			layout.T = (*T)(t)
			{{- range .Overrides}}
			layout.{{.FieldName}} = (*{{.ToType}})(&layout.T.{{.FieldName}})
			{{end -}}

			return e.EncodeElement(layout, start)
		`, data).Decl()
	return marshal, unmarshal, err
}

func (cfg *Config) genSimpleType(t *xsd.SimpleType) ([]spec, error) {
	var result []spec
	if t.List {
		return cfg.genSimpleListSpec(t)
	}
	if len(t.Union) > 0 {
		// We don't support unions because the code that needs
		// to be generated to check which of the member types
		// the value would be too complex. Need a use case
		// first.
		result = append(result, spec{
			doc:     t.Doc,
			name:    cfg.public(t.Name),
			expr:    builtinExpr(xsd.String),
			xsdType: t,
		})
		return result, nil
	}
	base, err := cfg.expr(t.Base)
	if err != nil {
		return nil, fmt.Errorf("simpleType %s: base type %s: %v",
			t.Name.Local, xsd.XMLName(t.Base).Local, err)
	}

	spec, err := cfg.addSpecMethods(spec{
		doc:     t.Doc,
		name:    cfg.public(t.Name),
		expr:    base,
		xsdType: t,
	})
	if err != nil {
		return result, err
	}
	return append(result, spec), nil
}

// Attach Marshal/Unmarshal methods to a simple type, if necessary.
func (cfg *Config) addSpecMethods(s spec) (spec, error) {
	t, ok := s.xsdType.(*xsd.SimpleType)
	if !ok || !nonTrivialBuiltin(t.Base) {
		return s, nil
	}

	helper, ok := cfg.helperTypes[xsd.XMLName(t.Base)]
	if !ok {
		return s, fmt.Errorf("no helper type for %v", t.Base)
	}

	s.helperTypes = append(s.helperTypes,
		xsd.XMLName(helper.xsdType))

	s.methods = append(s.methods, gen.Func("UnmarshalText").
		Receiver("t *"+s.name).
		Args("text []byte").
		Returns("error").
		Body(`return (*%s)(t).UnmarshalText(text)`, helper.name).
		MustDecl())

	s.methods = append(s.methods, gen.Func("MarshalText").
		Receiver("t "+s.name).
		Returns("[]byte", "error").
		Body(`return %s(t).MarshalText()`, helper.name).
		MustDecl())

	return s, nil
}

// Generate a type declaration for a <list> type, along with marshal/unmarshal
// methods.
func (cfg *Config) genSimpleListSpec(t *xsd.SimpleType) ([]spec, error) {
	cfg.debugf("generating Go source for simple list %q", xsd.XMLName(t).Local)
	expr, err := cfg.expr(t.Base)
	if err != nil {
		return nil, err
	}
	expr = &ast.ArrayType{Elt: expr}
	s := spec{
		name:    cfg.public(t.Name),
		expr:    expr,
		xsdType: t,
	}
	marshalFn := gen.Func("MarshalText").
		Receiver("x *"+s.name).
		Returns("[]byte", "error")
	unmarshalFn := gen.Func("UnmarshalText").
		Receiver("x *" + s.name).
		Args("text []byte").
		Returns("error")

	base := t.Base
	for xsd.Base(base) != nil {
		base = xsd.Base(base)
	}

	switch base.(xsd.Builtin) {
	case xsd.ID, xsd.NCName, xsd.NMTOKEN, xsd.Name, xsd.QName, xsd.ENTITY, xsd.AnyURI, xsd.Language, xsd.String, xsd.Token, xsd.XMLLang, xsd.XMLSpace, xsd.XMLBase, xsd.XMLId, xsd.Duration, xsd.NormalizedString, xsd.AnySimpleType:
		marshalFn = marshalFn.Body(`
			result := make([][]byte, 0, len(*x))
			for _, v := range *x {
				result = append(result, []byte(v))
			}
			return bytes.Join(result, []byte(" ")), nil
		`)
		unmarshalFn = unmarshalFn.Body(`
			for _, v := range bytes.Fields(text) {
				*x = append(*x, string(v))
			}
			return nil
		`)
	case xsd.Date, xsd.DateTime, xsd.GDay, xsd.GMonth, xsd.GMonthDay, xsd.GYear, xsd.GYearMonth, xsd.Time:
		marshalFn = marshalFn.Body(`
			result := make([][]byte, 0, len(*x))
			for _, v := range *x {
				if b, err := v.MarshalText(); err != nil {
					return result, err
				} else {
					result = append(result, b)
				}
			}
			return bytes.Join(result, []byte(" "))
		`)
		unmarshalFn = unmarshalFn.Body(`
			for _, v := range bytes.Fields(text) {
				var t %s
				if err := t.UnmarshalText(v); err != nil {
					return err
				}
				*x = append(*x, t)
			}
		`, builtinExpr(base.(xsd.Builtin)).(*ast.Ident).Name)
	case xsd.Long:
		marshalFn = marshalFn.Body(`
			result := make([][]byte, 0, len(*x))
			for _, v := range *x {
				result = append(result, []byte(strconv.FormatInt(v, 10)))
			}
			return bytes.Join(result, []byte(" ")), nil
		`)
		unmarshalFn = unmarshalFn.Body(`
			for _, v := range strings.Fields(string(text)) {
				if i, err := strconv.ParseInt(v, 10, 64); err != nil {
					return err
				} else {
					*x = append(*x, i)
				}
			}
			return nil
		`)
	case xsd.Decimal, xsd.Double:
		marshalFn = marshalFn.Body(`
			result := make([][]byte, 0, len(*x))
			for _, v := range *x {
				s := strconv.FormatFloat(v, 'g', -1, 64)
				result = append(result, []byte(s))
			}
			return bytes.Join(result, []byte(" ")), nil
		`)
		unmarshalFn = unmarshalFn.Body(`
			for _, v := range strings.Fields(string(text)) {
				if f, err := strconv.ParseFloat(v, 64); err != nil {
					return err
				} else {
					*x = append(*x, f)
				}
			}
			return nil
		`)
	case xsd.Int, xsd.Integer, xsd.NegativeInteger, xsd.NonNegativeInteger, xsd.NonPositiveInteger, xsd.Short:
		marshalFn = marshalFn.Body(`
			result := make([][]byte, 0, len(*x))
			for _, v := range *x {
				result = append(result, []byte(strconv.Itoa(v)))
			}
			return bytes.Join(result, []byte(" ")), nil
		`)
		unmarshalFn = unmarshalFn.Body(`
			for _, v := range strings.Fields(string(text)) {
				if i, err := strconv.Atoi(v); err != nil {
					return err
				} else {
					*x = append(*x, i)
				}
			}
			return nil
		`)
	case xsd.UnsignedInt, xsd.UnsignedShort:
		marshalFn = marshalFn.Body(`
			result := make([][]byte, 0, len(*x))
			for _, v := range *x {
				result = append(result, []byte(strconv.FormatUint(uint64(v), 10)))
			}
			return bytes.Join(result, []byte(" ")), nil
		`)
		unmarshalFn = unmarshalFn.Body(`
			for _, v := range strings.Fields(string(text)) {
				if i, err := strconv.ParseUint(v, 10, 32); err != nil {
					return err
				} else {
					*x = append(*x, uint(i))
				}
			}
			return nil
		`)
	case xsd.UnsignedLong:
		marshalFn = marshalFn.Body(`
			result := make([][]byte, 0, len(*x))
			for _, v := range *x {
				result = append(result, []byte(strconv.FormatUInt(v, 10)))
			}
			return bytes.Join(result, []byte(" ")), nil
		`)
		unmarshalFn = unmarshalFn.Body(`
			for _, v := range strings.Fields(string(text)) {
				if i, err := strconv.ParseInt(v, 10, 64); err != nil {
					return err
				} else {
					*x = append(*x, i)
				}
			}
			return nil
		`)
	case xsd.Byte, xsd.UnsignedByte:
		marshalFn = marshalFn.Body(`
			return []byte(*x), nil
		`)
		unmarshalFn = unmarshalFn.Body(`
			*x = %s(text)
			return nil
		`, s.name)
	case xsd.Boolean:
		marshalFn = marshalFn.Body(`
			result := make([][]byte
			for _, b := range *x {
				if b {
					result = append(result, []byte("1"))
				} else {
					result = append(result, []byte("0"))
				}
			}
			return bytes.Join(result, []byte(" ")), nil
		`)
		unmarshalFn = unmarshalFn.Body(`
			for _, v := range bytes.Fields(text) {
				switch string(v) {
				case "1", "true":
					*x = append(*x, true)
				default:
					*x = append(*x, false)
				}
			}
			return nil
		`)
	default:
		return nil, fmt.Errorf("don't know how to marshal/unmarshal <list> of %s", base)
	}

	marshal, err := marshalFn.Decl()
	if err != nil {
		return nil, fmt.Errorf("MarshalText %s: %v", s.name, err)
	}

	unmarshal, err := unmarshalFn.Decl()
	if err != nil {
		return nil, fmt.Errorf("UnmarshalText %s: %v", s.name, err)
	}

	s.methods = append(s.methods, marshal, unmarshal)
	return []spec{s}, nil
}

// O(n²) is OK since you'll never see more than ~40 attributes...
// right?
func mergeAttributes(src, base *xsd.ComplexType) []xsd.Attribute {
Loop:
	for _, baseattr := range base.Attributes {
		for _, srcattr := range src.Attributes {
			if srcattr.Name == baseattr.Name {
				continue Loop
			}
		}
		src.Attributes = append(src.Attributes, baseattr)
	}
	return src.Attributes
}
