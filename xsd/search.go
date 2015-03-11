package xsd

import "aqwari.net/xml/xmltree"

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
