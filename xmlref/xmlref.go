// Package xmlref simplifies XML documents with references.
//
// The xmlref package allows for processing of multi-reference
// SOAP objects in XML documents as if they were single reference,
// as expected by the encoding/xml package. Documents such as
//
// 	<response href="#id0"/>
// 	<multiRef id="id0">
// 	  <anyMoreRec href="#id1"/>
// 	</multiRef>
// 	<multiRef id="id1">false</multiRef>
//
// when read through a Reader, will appear as
//
// 	<response>
// 	  <anyMoreRec>false</anyMoreRec>
// 	</response>
//
package xmlref

import (
	"io"
	"strings"
)

func parseURIRef(ref string) (id string) {
	return strings.TrimPrefix(ref, "#")
}

// A Reader wraps an existing io.ReadSeeker positioned at the start of an
// XML document. The XML document read through a Reader will by identical
// to the original document, with the exception that any multi-reference
// SOAP elements will be replaced with the single-reference elements they
// point to. The state of the underlying ReadSeeker should not be changed
// (by calls to its Read or Seek method) between calls to Read.
type Reader struct {
	start int64
	buf   []byte
	rd    io.ReadSeeker
}

// Read reads up to len(p) bytes from the underlying ReadSeeker into
// p. The source's Seek method may be called when a multi-reference
// element is encountered. Read returns the number of bytes read,
// and any I/O errors encountered.
func (r *Reader) Read(p []byte) (int, error) {
	panic("TODO")
}

// NewReader creates a new Reader that reads from r.  NewReader will read
// through the XML document in r to locate any multi-reference objects. Any
// I/O or XML parsing errors are returned.
func NewReader(r io.ReadSeeker) (*Reader, error) {
	panic("TODO")
}
