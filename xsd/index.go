package xsd

import (
	"encoding/xml"

	"aqwari.net/xml/xmltree"
)

type elementKey struct {
	Name, Type xml.Name
}

type schemaIndex struct {
	eltByID []*xmltree.Element
	// name indices can collide since different element
	// types can have the same node.
	idByName map[elementKey]int
}

func (idx *schemaIndex) ByName(name, typ xml.Name) (*xmltree.Element, bool) {
	if id, ok := idx.idByName[elementKey{name, typ}]; ok {
		return idx.eltByID[id], true
	}
	return nil, false
}

func (idx *schemaIndex) ElementID(name, typ xml.Name) (int, bool) {
	id, ok := idx.idByName[elementKey{name, typ}]
	return id, ok
}

func indexSchema(schema []*xmltree.Element) *schemaIndex {
	index := &schemaIndex{
		idByName: make(map[elementKey]int),
	}
	for _, root := range schema {
		tns := root.Attr("", "targetNamespace")
		for _, el := range root.Flatten() {
			id := len(index.eltByID)
			index.eltByID = append(index.eltByID, el)
			if name := el.Attr("", "name"); name != "" {
				xmlname := el.ResolveDefault(name, tns)
				index.idByName[elementKey{xmlname, el.Name}] = id
			}
		}
	}
	return index
}
