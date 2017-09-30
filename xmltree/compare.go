package xmltree

import (
	"bytes"
	"encoding/xml"
	"sort"
)

// Equal returns true if two xmltree.Elements are equal, ignoring
// differences in white space, sub-element order, and namespace prefixes.
func Equal(a, b *Element) bool {
	return equal(a, b, 0)
}

type byName []Element

func (l byName) Len() int { return len(l) }
func (l byName) Less(i, j int) bool {
	return l[i].Name.Space+l[i].Name.Local < l[j].Name.Space+l[j].Name.Local
}
func (l byName) Swap(i, j int) { l[i], l[j] = l[j], l[i] }

func equal(a, b *Element, depth int) bool {
	const maxDepth = 1000
	if depth > maxDepth {
		return false
	}
	if !equalElement(a, b) {
		return false
	}
	if len(a.Children) != len(b.Children) {
		return false
	}
	if len(a.Children) == 0 {
		return bytes.Equal(bytes.TrimSpace(a.Content), bytes.TrimSpace(b.Content))
	}
	sort.Sort(byName(a.Children))
	sort.Sort(byName(b.Children))
	for i := range a.Children {
		if !equal(&a.Children[i], &b.Children[i], depth+1) {
			return false
		}
	}
	return true
}

func equalElement(a, b *Element) bool {
	if a.Name != b.Name {
		return false
	}
	attrs := make(map[xml.Name]string)
	for _, a := range a.StartElement.Attr {
		if a.Name.Space == "xmlns" || a.Name.Local == "xmlns" {
			continue
		}
		attrs[a.Name] = a.Value
	}

	for _, a := range b.StartElement.Attr {
		if a.Name.Space == "xmlns" || a.Name.Local == "xmlns" {
			continue
		}
		if v, ok := attrs[a.Name]; !ok || v != a.Value {
			return false
		}
	}
	return true
}
