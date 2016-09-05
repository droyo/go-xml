package chemspell

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"testing"
)

func TestNDFDGen(t *testing.T) {
	client := NewClient()
	client.HTTPClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client.RequestHook = func(req *http.Request) {
		data, err := httputil.DumpRequest(req, true)
		if err != nil {
			panic(err)
		}
		t.Log(req.URL.String(), string(data))
		//req.URL = nil
	}
	client.ResponseHook = func(rsp *http.Response) {
		data, err := httputil.DumpResponse(rsp, true)
		if err != nil {
			panic(err)
		}
		t.Log(string(data))
	}

	s, err := client.GetSugList("foo", "All databases")
	t.Log(s, err)
	//err := client.Main([]string{"foo", "bar"})
	//t.Log(err)
}
