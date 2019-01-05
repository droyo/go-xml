// Package testutils contains common utility functions for unit tests.
package testutil

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"
)

// FakeClient returns an HTTP client that replies to all requests to the given
// address with the provided body text.
func FakeClient(url string, body []byte) http.Client {
	return http.Client{
		Transport: mockRoundTrip{
			url:  url,
			body: body,
		},
	}
}

type mockRoundTrip struct {
	url  string
	body []byte
}

func (r mockRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	var rsp http.Response
	rsp.Header = make(http.Header)

	if req.URL.String() == r.url {
		rsp.StatusCode = 200
		rsp.Body = ioutil.NopCloser(bytes.NewReader(r.body))
	} else {
		rsp.StatusCode = 404
		rsp.Body = ioutil.NopCloser(strings.NewReader("404 not found"))
	}
	return &rsp, nil
}
