package xsdgen // import "aqwari.net/xml/xsdgen"

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"sort"
	"strings"

	"aqwari.net/xml/internal/gen"
	"aqwari.net/xml/xmltree"
	"aqwari.net/xml/xsd"
)

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
			Children: []xmltree.Element{*tree},
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

// Code is the intermediate representation used by the xsdgen
// package. It can be used to generate a Go source file, and to
// lookup identifiers and attributes for a given type.
type Code struct {
	cfg   *Config
	names map[xml.Name]string
	decls map[string]spec
	types map[xml.Name]xsd.Type
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

	switch b := c.cfg.flatten1(t, func(xsd.Type) {}).(type) {
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
		decls: make(map[string]spec),
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
		for _, t := range cfg.flatten(primary.Types) {
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
	name    string
	expr    ast.Expr
	private bool
	methods []*ast.FuncDecl
	xsdType xsd.Type
}

// Flatten out our tree of dependent types. If a type is marked as
// private by a user filter and not used as a struct field or embedded
// struct, it is ommitted from the output.
func (cfg *Config) flatten(types map[xml.Name]xsd.Type) []xsd.Type {
	var result []xsd.Type
	push := func(t xsd.Type) {
		result = append(result, t)
	}
	for _, t := range types {
		cfg.debugf("flattening type %T(%s)\n", t, xsd.XMLName(t).Local)
		if cfg.filterTypes != nil && cfg.filterTypes(t) {
			continue
		}
		if cfg.allowTypes != nil {
			if !cfg.allowTypes[xsd.XMLName(t)] {
				continue
			}
		}
		if t := cfg.flatten1(t, push); t != nil {
			result = append(result, t)
		}
	}
	// Remove duplicates
	seen := make(map[xml.Name]bool)
	var a []xsd.Type
	for _, v := range result {
		name := xsd.XMLName(v)
		if _, ok := seen[name]; !ok {
			seen[name] = true
			a = append(a, v)
		}
	}
	cfg.debugf("discovered %d types", len(a))
	return a
}

// To reduce the size of the Go source generated, all intermediate types
// are "squashed"; every type should be based on a Builtin or another
// type that the user wants included in the Go source.
func (cfg *Config) flatten1(t xsd.Type, push func(xsd.Type)) xsd.Type {
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
		cfg.debugf("%T(%s) -> %T(%s)", t, xsd.XMLName(t).Local,
			t.Base, xsd.XMLName(t.Base).Local)
		return t
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
					return cfg.flatten1(t.Base, push)
				}
			}
		}
		// We can flatten a struct field if its type does not
		// need additional methods for unmarshalling.
		for i, el := range t.Elements {
			el.Type = cfg.flatten1(el.Type, push)
			if b, ok := el.Type.(*xsd.SimpleType); ok {
				if !b.List && len(b.Union) == 0 {
					el.Type = xsd.Base(el.Type)
				}
			}
			t.Elements[i] = el
			push(el.Type)
			cfg.debugf("%T(%s): %v", t, xsd.XMLName(t).Local,
				xsd.XMLName(el.Type))
		}
		for i, attr := range t.Attributes {
			attr.Type = cfg.flatten1(attr.Type, push)
			if b, ok := attr.Type.(*xsd.SimpleType); ok {
				if !b.List && len(b.Union) == 0 {
					attr.Type = xsd.Base(attr.Type)
				}
			}
			t.Attributes[i] = attr
		}
		cfg.debugf("%T(%s) -> %T(%s)", t, xsd.XMLName(t).Local,
			t.Base, xsd.XMLName(t.Base).Local)
		return t
	case xsd.Builtin:
		// There are a few built-ins that do not map directly to Go types.
		// for these, we will declare them in the Go source.
		switch t {
		case xsd.ENTITIES, xsd.IDREFS, xsd.NMTOKENS:
			push(t)
		case xsd.Base64Binary, xsd.HexBinary:
			push(t)
		case xsd.Date, xsd.Time, xsd.DateTime:
			push(t)
		case xsd.GDay, xsd.GMonth, xsd.GMonthDay, xsd.GYear, xsd.GYearMonth:
			push(t)
		}
		return t
	}
	panic(fmt.Sprintf("unexpected %T", t))
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
		// Some built-ins, though built-in, require marshal/unmarshal methods
		// to be able to use them with the encoding/xml package.
		switch t {
		case xsd.Date, xsd.Time, xsd.DateTime, xsd.GDay, xsd.GMonth, xsd.GMonthDay, xsd.GYear, xsd.GYearMonth:
			s, err = cfg.genTimeSpec(t)
		case xsd.HexBinary, xsd.Base64Binary:
			s, err = cfg.genBinarySpec(t)
		case xsd.ENTITIES, xsd.IDREFS, xsd.NMTOKENS:
			s, err = cfg.genTokenListSpec(t)
		}
	default:
		cfg.logf("unexpected %T %s", t, xsd.XMLName(t).Local)
	}
	if err != nil || s == nil {
		return result, err
	}
	return append(result, s...), nil
}

func (cfg *Config) genComplexType(t *xsd.ComplexType) ([]spec, error) {
	var result []spec
	var fields []ast.Expr

	if t.Extends {
		base, err := cfg.expr(t.Base)
		if err != nil {
			return nil, fmt.Errorf("%s base type %s: %v",
				t.Name.Local, xsd.XMLName(t.Base).Local, err)
		}
		switch b := t.Base.(type) {
		case *xsd.SimpleType:
			cfg.debugf("complexType %[1]s extends simpleType %[2]s. Naming"+
				" the chardata struct field after %[2]s", t.Name.Local, b.Name.Local)
			fields = append(fields, base, base, gen.String(`xml:",chardata"`))
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
			fields = append(fields, ast.NewIdent(b.String()), base, gen.String(`xml:",chardata"`))
		case *xsd.ComplexType:
			// Use struct embedding when extending a complex type
			cfg.debugf("complexType %s extends %s, embedding struct",
				t.Name.Local, b.Name.Local)
			fields = append(fields, nil, base, nil)
		}
	} else {
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
	}

	attributes, elements := cfg.filterFields(t)
	cfg.debugf("complexType %s: generating struct fields for %d elements and %d attributes",
		xsd.XMLName(t).Local, len(elements), len(attributes))
	hasDefault := false
	for _, attr := range attributes {
		hasDefault = hasDefault || (attr.Default != "")
		tag := fmt.Sprintf(`xml:"%s,attr"`, attr.Name.Local)
		base, err := cfg.expr(attr.Type)
		if err != nil {
			return nil, fmt.Errorf("%s attribute %s: %v", t.Name.Local, attr.Name.Local, err)
		}
		fields = append(fields, ast.NewIdent(cfg.public(attr.Name)), base, gen.String(tag))
	}
	for _, el := range elements {
		hasDefault = hasDefault || (el.Default != "")
		tag := fmt.Sprintf(`xml:"%s %s"`, el.Name.Space, el.Name.Local)
		base, err := cfg.expr(el.Type)
		if err != nil {
			return nil, fmt.Errorf("%s element %s: %v", t.Name.Local, el.Name.Local, err)
		}
		name := ast.NewIdent(cfg.public(el.Name))
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
	}
	expr := gen.Struct(fields...)
	s := spec{
		name:    cfg.public(t.Name),
		expr:    expr,
		xsdType: t,
	}
	if hasDefault {
		unmarshal, marshal, err := cfg.genMarshalComplexType(t)
		if err != nil {
			//NOTE(droyo) may want to log this instead of stopping the generator
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

func (cfg *Config) genMarshalComplexType(t *xsd.ComplexType) (marshal, unmarshal *ast.FuncDecl, err error) {
	// TODO(droyo): this one is a lot of work
	return nil, nil, nil
}

func (cfg *Config) genSimpleType(t *xsd.SimpleType) ([]spec, error) {
	var result []spec
	if t.List {
		return cfg.genSimpleListSpec(t)
	}
	if len(t.Union) > 0 {
		// We don't support unions because the code that needs
		// to be generated to check which of the member types
		// the value would be is too complex. Need a use case
		// first.
		result = append(result, spec{
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
	result = append(result, spec{
		name:    cfg.public(t.Name),
		expr:    base,
		xsdType: t,
	})
	return result, nil
}

// Generate a type declaration for the built-in time values, along with
// marshal/unmarshal methods for them.
func (cfg *Config) genTimeSpec(t xsd.Builtin) ([]spec, error) {
	var timespec string
	cfg.debugf("generating Go source for time type %q", xsd.XMLName(t).Local)

	s := spec{
		expr:    ast.NewIdent("time.Time"),
		name:    builtinExpr(t).(*ast.Ident).Name,
		xsdType: t,
	}

	switch t {
	case xsd.GDay:
		timespec = "---02"
	case xsd.GMonth:
		timespec = "--01"
	case xsd.GMonthDay:
		timespec = "--01-02"
	case xsd.GYear:
		timespec = "2006"
	case xsd.GYearMonth:
		timespec = "2006-01"
	case xsd.Time:
		timespec = "15:04:05.999999999"
	case xsd.Date:
		timespec = "2006-01-02"
	case xsd.DateTime:
		timespec = "2006-01-02T15:04:05.999999999"
	}
	unmarshal, err := gen.Func("UnmarshalText").
		Receiver("t *"+s.name).
		Args("text []byte").
		Returns("error").
		Body(`
			return _unmarshalTime(text, (*time.Time)(t), %q)
		`, timespec).Decl()
	if err != nil {
		return nil, fmt.Errorf("could not generate unmarshal function for %s: %v", s.name, err)
	}
	marshal, err := gen.Func("MarshalText").
		Receiver("t *"+s.name).
		Returns("[]byte", "error").
		Body(`
			return []byte((*time.Time)(t).Format(%q)), nil
		`, timespec).Decl()
	if err != nil {
		return nil, fmt.Errorf("could not generate marshal function for %s: %v", s.name, err)
	}
	s.methods = append(s.methods, unmarshal, marshal)
	if helper := cfg.helper("_unmarshalTime"); helper != nil {
		s.methods = append(s.methods, helper)
	}
	return []spec{s}, nil
}

// Generate a type declaration for the built-in binary values, along with
// marshal/unmarshal methods for them.
func (cfg *Config) genBinarySpec(t xsd.Builtin) ([]spec, error) {
	cfg.debugf("generating Go source for binary type %q", xsd.XMLName(t).Local)
	s := spec{
		expr:    builtinExpr(t),
		name:    xsd.XMLName(t).Local,
		xsdType: t,
	}
	marshal := gen.Func("MarshalText").Receiver("b "+s.name).Returns("[]byte", "error")
	unmarshal := gen.Func("UnmarshalText").Receiver("b " + s.name).Args("text []byte").
		Returns("err error")

	switch t {
	case xsd.HexBinary:
		unmarshal.Body(`
			*b, err = hex.DecodeString(string(text))
			return err
		`)
		marshal.Body(`
			n := hex.EncodedLen([]byte(b))
			buf := make([]byte, n)
			hex.Encode(buf, []byte(b))
			return buf, nil
		`)
	case xsd.Base64Binary:
		unmarshal.Body(`
			*b, err = base64.StdEncoding.DecodeString(string(text))
			return err
		`)
		marshal.Body(`
			var buf bytes.Buffer
			enc := base64.NewEncoder(base64.StdEncoding, &buf)
			enc.Write([]byte(b))
			enc.Close()
			return buf.Bytes()
		`)
	}
	marshalFn, err := marshal.Decl()
	if err != nil {
		return nil, fmt.Errorf("MarshalText %s: %v", s.name, err)
	}
	unmarshalFn, err := unmarshal.Decl()
	if err != nil {
		return nil, fmt.Errorf("UnmarshalText %s: %v", s.name, err)
	}
	s.methods = append(s.methods, unmarshalFn, marshalFn)
	return []spec{s}, nil
}

// Generate a type declaration for the bult-in list values, along with
// marshal/unmarshal methods
func (cfg *Config) genTokenListSpec(t xsd.Builtin) ([]spec, error) {
	cfg.debugf("generating Go source for token list %q", xsd.XMLName(t).Local)
	s := spec{
		name:    strings.ToLower(t.String()),
		expr:    builtinExpr(t),
		xsdType: t,
	}
	marshal, err := gen.Func("MarshalText").
		Receiver("x *"+s.name).
		Returns("[]byte", "error").
		Body(`
			return []byte(strings.Join(x, " ")), nil
		`).Decl()

	if err != nil {
		return nil, fmt.Errorf("MarshalText %s: %v", s.name, err)
	}

	unmarshal, err := gen.Func("UnmarshalText").
		Receiver("x *" + s.name).
		Args("text []byte").
		Returns("error").
		Body(`
			*x = bytes.Fields(text)
			return nil
		`).Decl()

	if err != nil {
		return nil, fmt.Errorf("UnmarshalText %s: %v", s.name, err)
	}

	s.methods = append(s.methods, marshal, unmarshal)
	return []spec{s}, nil
}

// Generate a type declaration for a <list> type, along with marshal/unmarshal
// methods.
func (cfg *Config) genSimpleListSpec(t *xsd.SimpleType) ([]spec, error) {
	cfg.debugf("generating Go source for simple list %q", xsd.XMLName(t).Local)
	expr, err := cfg.expr(t.Base)
	if err != nil {
		return nil, err
	}
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
	case xsd.ID, xsd.NCName, xsd.NMTOKEN, xsd.Name, xsd.QName, xsd.ENTITY, xsd.AnyURI, xsd.Language, xsd.String, xsd.Token, xsd.XMLLang, xsd.XMLSpace, xsd.XMLBase, xsd.XMLId, xsd.Duration, xsd.NormalizedString:
		marshalFn = marshalFn.Body(`
			result := make([][]byte, 0, len(*x))
			for _, v := range *x {
				result = append(result, []byte(v))
			}
			return bytes.Join(result, []byte(" ")), nil
		`)
		unmarshalFn = marshalFn.Body(`
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
				result = append(result, []byte(strconv.FormatUInt(uint64(v), 10)))
			}
			return bytes.Join(result, []byte(" ")), nil
		`)
		unmarshalFn = unmarshalFn.Body(`
			for _, v := range strings.Fields(string(text)) {
				if i, err := strconv.ParseUInt(v, 10, 32); err != nil {
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
