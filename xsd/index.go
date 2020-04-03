package xsd

import (
	"encoding/xml"

	"github.com/CognitoIQ/go-xml/xmltree"
)

type elementKey struct {
	Name, Type xml.Name
	Depth int
}

type schemaIndex struct {
	eltByID []*xmltree.Element
	// name indices can collide since different element
	// types can have the same node.
	idByName map[elementKey]int
}

func (idx *schemaIndex) ByName(name xml.Name, typ xml.Name, depth int) (*xmltree.Element, bool) {
	if id, ok := idx.idByName[elementKey{name, typ, depth}]; ok {
		return idx.eltByID[id], true
	}
	return nil, false
}

func (idx *schemaIndex) ElementID(name xml.Name, typ xml.Name, depth int) (int, bool) {
	id, ok := idx.idByName[elementKey{name, typ, depth}]
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
				index.idByName[elementKey{xmlname, el.Name,  el.Depth}] = id
			}
		}
	}
	return index
}
