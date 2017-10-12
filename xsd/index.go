package xsd

import (
	"encoding/xml"

	"aqwari.net/xml/internal/ordered"
	"aqwari.net/xml/xmltree"
)

type elementKey struct {
	Name, Type xml.Name
}

type schemaIndex struct {
	eltByID map[int]*xmltree.Element
	// name indices can collide since different element
	// types can have the same node.
	idByName map[elementKey]int
}

func (idx *schemaIndex) ByName(name, typ xml.Name) (*xmltree.Element, bool) {
	if id, ok := idx.idByName[elementKey{name, typ}]; ok {
		if el, ok := idx.eltByID[id]; ok {
			return el, true
		}
		panic("bug building schema index; name map does not match ID map")
	}
	return nil, false
}

func (idx *schemaIndex) ElementID(name, typ xml.Name) (int, bool) {
	id, ok := idx.idByName[elementKey{name, typ}]
	return id, ok
}

func indexSchema(schema map[string]*xmltree.Element) *schemaIndex {
	counter := 0
	index := &schemaIndex{
		eltByID:  make(map[int]*xmltree.Element),
		idByName: make(map[elementKey]int),
	}
	ordered.RangeStrings(schema, func(targetNS string) {
		for _, el := range schema[targetNS].Flatten() {
			index.eltByID[counter] = el
			if name := el.Attr("", "name"); name != "" {
				xmlname := el.ResolveDefault(name, targetNS)
				index.idByName[elementKey{xmlname, el.Name}] = counter
			}
			counter++
		}
	})
	return index
}
