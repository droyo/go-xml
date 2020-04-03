package xsd

import (
	"fmt"
	"strings"

	"github.com/CognitoIQ/go-xml/xmltree"
)

// When working with an xml tree structure, we naturally have some
// pretty deep function calls.  To save some typing, we use panic/recover
// to bubble the errors up. These panics are not exposed to the user.
type parseError struct {
	message string
	path    []*xmltree.Element
}

func (err parseError) Error() string {
	breadcrumbs := make([]string, 0, len(err.path))
	for i := len(err.path) - 1; i >= 0; i-- {
		piece := err.path[i].Name.Local
		if name := err.path[i].Attr("", "name"); name != "" {
			piece = fmt.Sprintf("%s(%s)", piece, name)
		}
		breadcrumbs = append(breadcrumbs, piece)
	}
	return "Error at " + strings.Join(breadcrumbs, ">") + ": " + err.message
}

func stop(msg string) {
	panic(parseError{message: msg})
}

func walk(root *xmltree.Element, fn func(*xmltree.Element)) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(parseError); ok {
				err.path = append(err.path, root)
				panic(err)
			} else {
				panic(r)
			}
		}
	}()
	for i := 0; i < len(root.Children); i++ {
		// We don't care about elements outside of the
		// XML schema namespace
		if root.Children[i].Name.Space != schemaNS {
			continue
		}
		fn(&root.Children[i])
	}
}

// defer catchParseError(&err)
func catchParseError(err *error) {
	if r := recover(); r != nil {
		*err = r.(parseError)
	}
}
