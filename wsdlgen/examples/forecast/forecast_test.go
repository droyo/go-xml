package forecast

import (
	"crypto/tls"
	"net/http"
	"net/http/httputil"
	"testing"
	"time"
)

func TestNDFDGen(t *testing.T) {
	client := NewClient()
	client.HTTPClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client.RequestHook = func(req *http.Request) *http.Request {
		data, err := httputil.DumpRequest(req, true)
		if err != nil {
			panic(err)
		}
		t.Log(string(data))

		// NOTE(droyo) The national digital forecast database is
		// a public, shared service provided for free. It is not
		// polite or appropriate to test against this endpoint as
		// part of our frequent unit tests. The proper thing to do is
		// to capture output from this service and use it to setup
		// a mock server. The following line can be removed to
		// obtain such output. Please be responsible.
		req.URL = nil
		return req
	}
	client.ResponseHook = func(rsp *http.Response) *http.Response {
		data, err := httputil.DumpResponse(rsp, true)
		if err != nil {
			panic(err)
		}
		t.Log(string(data))
		return rsp
	}

	s, _ := client.NDFDgen(NDFDgenRequest{
		EndTime:   time.Now(),
		StartTime: time.Now().Add(-time.Minute * 10),
		Unit:      "m",
		Product:   "time-series",
		Latitude:  42,
		Longitude: 71,
		WeatherParameters: WeatherParameters{
			Sky: true,
		},
	})
	t.Log(s)
}
