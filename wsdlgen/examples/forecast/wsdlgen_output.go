// Package forecast access the National Digital Forecast Database.
//
// The service has 12 exposed functions, NDFDgen, NDFDgenLatLonList, NDFDgenByDay, NDFDgenByDayLatLonList,
// LatLonListSubgrid, LatLonListLine, LatLonListZipCode, CornerPoints, LatLonListSquare, GmlLatLonList, GmlTimeSeries, and LatLonListCityNames.
// For the NDFDgen function, the client needs to provide a latitude and longitude pair and the product type. The Unit will default
// to U.S. Standard (english) unless Metric is chosen by client. The client also needs to provide the start and end time (Local)
// of the period that it wants data for (if shorter than the 7 days is wanted).  For the time-series product, the client needs to
// provide an array of boolean values corresponding to which NDFD values are desired.
// For the NDFDgenByDay function, the client needs to provide a latitude and longitude pair, the date (Local) it wants to start
// retrieving data for and the number of days worth of data.  The Unit will default to U.S. Standard (english) unless Metric is
// chosen by client. The client also needs to provide the format that is desired.
// For the multi point versions, NDFDgenLatLonList and NDFDgenByDayLatLonList a space delimited list of latitude and longitude
// pairs are substituted for the single latitude and longitude input.  Each latitude and longitude
// pair is composed of a latitude and longitude delimited by a comma.
// For the LatLonListSubgrid, the user provides a comma delimited latitude and longitude pair for the lower left and for
// the upper right corners of a rectangular subgrid.  The function can also take a integer
// resolution to reduce the number of grid points returned. The service then returns a list of
// latitude and longitude pairs for all the grid points contained in the subgrid.
// weather values should appear in the time series product.
// For the LatLonListLine, The inputs are the same as the function NDFDgen except the latitude and longitude pair is
// replaced by two latitude and longitude pairs, one for each end point a line. The two points are delimited with a space.
// The service then returns data for all the NDFD points on the line formed by the two points.
// For the LatLonListZipCode function, the input is the same as the NDFDgen function except the latitude and longitude values
// are relaced by a zip code for the 50 United States and Puerto Rico.
// For the LatLonListSquare function, the input is the same as the NDFDgen function except the latitude and longitude values
// are relaced by a zip code for the 50 United States and Puerto Rico.
// For the CornerPoints function, the service requires a valid NDFD grid name.  The function returns a
// list of four latitude and longitude pairs, one for each corner of the NDFD grid.  The function
// also returns the minimum resolution required to return the entire grid below the maximum points
// threshold.
// For the GmlLatLonList function, the service requires a list of latitude and longitude pairs, the time (UTC) the user
// wants data for, the GML feature type and the array of boolean values corresponding to which NDFD values are desired.
// For the GmlTimeSeries function, the service requires a list of latitude and longitude pairs, the start and end time (UTC) the user
// wants data for, a comparison type (IsEqual, Between, GreaterThan, GreaterThan, GreaterThanEqualTo, LessThan, and
// LessThanEqualTo), the GML feature type and The input variable "propertyName" contains a comma delimited string of NDFD element to
// indicate which weather parameters are being requested.
// For the LatLonListCityNames function, the services requires a detail level that that ranges from 1 to 4.  Level 1 generally represents
// large main cities.  Level 2 represents progressively smaller cities or large cities that are close to another even larger city.  Levels
// 3 and 4 are part one and two of a list of cities that help increase the areal coverage of the cities dataset.  This functions
// returns a list of latitude and longitude values along with a seperate list of city name for those point.
package forecast

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"time"
)

// May be one of IsEqual, Between, GreaterThan, GreaterThanEqualTo, LessThan, LessThanEqualTo
type CompType string

// May be one of 1, 2, 3, 4, 12, 34, 1234
type DisplayLevel int

// May be one of Forecast_Gml2Point, Forecast_Gml2AllWx, Forecast_GmlsfPoint, Forecast_GmlObs, NdfdMultiPointCoverage, Ndfd_KmlPoint
type FeatureType string

// May be one of 24 hourly, 12 hourly
type Format string

// Must match the pattern [a-zA-Z'\-]*( ?[a-zA-Z'\-]*)*,[A-Z][A-Z](\|[a-zA-Z'\-]*( ?[a-zA-Z'\-]*)*,[A-Z][A-Z])*
type ListCityNames string

// Must match the pattern [\-]?\d{1,2}\.\d+,[\-]?\d{1,3}\.\d+( [\-]?\d{1,2}\.\d+,[\-]?\d{1,3}\.\d+)*
type ListLatLon string

// May be one of time-series, glance
type Product string

// May be one of conus, nhemi, alaska, guam, hawaii, puertori, npacocn
type Sector string

// May be one of e, m
type Unit string

type WeatherParameters struct {
	Maxt         bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd maxt"`
	Mint         bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd mint"`
	Temp         bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd temp"`
	Dew          bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd dew"`
	Pop12        bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd pop12"`
	Qpf          bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd qpf"`
	Sky          bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd sky"`
	Snow         bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd snow"`
	Wspd         bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd wspd"`
	Wdir         bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd wdir"`
	Wx           bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd wx"`
	Waveh        bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd waveh"`
	Icons        bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd icons"`
	Rh           bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd rh"`
	Appt         bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd appt"`
	Incw34       bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd incw34"`
	Incw50       bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd incw50"`
	Incw64       bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd incw64"`
	Cumw34       bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd cumw34"`
	Cumw50       bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd cumw50"`
	Cumw64       bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd cumw64"`
	Critfireo    bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd critfireo"`
	Dryfireo     bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd dryfireo"`
	Conhazo      bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd conhazo"`
	Ptornado     bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd ptornado"`
	Phail        bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd phail"`
	Ptstmwinds   bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd ptstmwinds"`
	Pxtornado    bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd pxtornado"`
	Pxhail       bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd pxhail"`
	Pxtstmwinds  bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd pxtstmwinds"`
	Ptotsvrtstm  bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd ptotsvrtstm"`
	Pxtotsvrtstm bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd pxtotsvrtstm"`
	Tmpabv14d    bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd tmpabv14d"`
	Tmpblw14d    bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd tmpblw14d"`
	Tmpabv30d    bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd tmpabv30d"`
	Tmpblw30d    bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd tmpblw30d"`
	Tmpabv90d    bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd tmpabv90d"`
	Tmpblw90d    bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd tmpblw90d"`
	Prcpabv14d   bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd prcpabv14d"`
	Prcpblw14d   bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd prcpblw14d"`
	Prcpabv30d   bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd prcpabv30d"`
	Prcpblw30d   bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd prcpblw30d"`
	Prcpabv90d   bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd prcpabv90d"`
	Prcpblw90d   bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd prcpblw90d"`
	Precipar     bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd precipa_r"`
	Skyr         bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd sky_r"`
	Tdr          bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd td_r"`
	Tempr        bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd temp_r"`
	Wdirr        bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd wdir_r"`
	Wspdr        bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd wspd_r"`
	Wwa          bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd wwa"`
	Wgust        bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd wgust"`
	Iceaccum     bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd iceaccum"`
	Maxrh        bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd maxrh"`
	Minrh        bool `xml:"http://graphical.weather.gov/xml/DWMLgen/schema/DWML.xsd minrh"`
}

// Must match the pattern \d{5}(\-\d{4})?( \d{5}(\-\d{4})?)*
type ZipCodeList string

type xsdDate time.Time

func (t *xsdDate) UnmarshalText(text []byte) error {
	return _unmarshalTime(text, (*time.Time)(t), "2006-01-02")
}
func (t *xsdDate) MarshalText() ([]byte, error) {
	return []byte((*time.Time)(t).Format("2006-01-02")), nil
}

type xsdDateTime time.Time

func (t *xsdDateTime) UnmarshalText(text []byte) error {
	return _unmarshalTime(text, (*time.Time)(t), "2006-01-02T15:04:05.999999999")
}
func (t *xsdDateTime) MarshalText() ([]byte, error) {
	return []byte((*time.Time)(t).Format("2006-01-02T15:04:05.999999999")), nil
}
func _unmarshalTime(text []byte, t *time.Time, format string) (err error) {
	s := string(bytes.TrimSpace(text))
	*t, err = time.Parse(format, s)
	if _, ok := err.(*time.ParseError); ok {
		*t, err = time.Parse(format+"Z07:00", s)
	}
	return err
}

type Client struct {
	HTTPClient   http.Client
	ResponseHook func(*http.Response)
	RequestHook  func(*http.Request)
}
type soapEnvelope struct {
	XMLName struct{}                      `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Header  []byte                        `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header"`
	Body    struct{ Message interface{} } `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
}

func (c *Client) do(method, uri string, in, out interface{}) error {
	var body io.Reader
	var envelope soapEnvelope
	if method == "POST" || method == "PUT" {
		var buf bytes.Buffer
		envelope.Body.Message = in
		enc := xml.NewEncoder(&buf)
		if err := enc.Encode(envelope); err != nil {
			return err
		}
		if err := enc.Flush(); err != nil {
			return err
		}
		body = &buf
	}
	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return err
	}
	if c.RequestHook != nil {
		c.RequestHook(req)
	}
	rsp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if c.ResponseHook != nil {
		c.ResponseHook(rsp)
	}
	dec := xml.NewDecoder(rsp.Body)
	envelope.Body.Message = out
	return dec.Decode(&envelope)
}

type NDFDgenRequest struct {
	Latitude          float64
	Longitude         float64
	Product           Product
	StartTime         time.Time
	EndTime           time.Time
	Unit              Unit
	WeatherParameters WeatherParameters
}

// Returns National Weather Service digital weather forecast data
func (c *Client) NDFDgen(v NDFDgenRequest) (string, error) {
	var input struct {
		XMLName           struct{}          `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl NDFDgenRequest"`
		Latitude          float64           `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl latitude"`
		Longitude         float64           `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl longitude"`
		Product           Product           `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl product"`
		StartTime         xsdDateTime       `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl startTime"`
		EndTime           xsdDateTime       `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl endTime"`
		Unit              Unit              `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl Unit"`
		WeatherParameters WeatherParameters `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl weatherParameters"`
	}
	input.Latitude = float64(v.Latitude)
	input.Longitude = float64(v.Longitude)
	input.Product = Product(v.Product)
	input.StartTime = xsdDateTime(v.StartTime)
	input.EndTime = xsdDateTime(v.EndTime)
	input.Unit = Unit(v.Unit)
	input.WeatherParameters = WeatherParameters(v.WeatherParameters)
	var output struct {
		XMLName struct{} `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl NDFDgenResponse"`
		DwmlOut string   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl dwmlOut"`
	}
	err := c.do("POST", "http://graphical.weather.gov/xml/SOAP_server/ndfdXMLserver.php", &input, &output)
	return string(output.DwmlOut), err
}

type NDFDgenByDayRequest struct {
	Latitude  float64
	Longitude float64
	StartDate time.Time
	NumDays   int
	Unit      Unit
	Format    Format
}

// Returns National Weather Service digital weather forecast data summarized over either 24- or 12-hourly periods
func (c *Client) NDFDgenByDay(v NDFDgenByDayRequest) (string, error) {
	var input struct {
		XMLName   struct{} `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl NDFDgenByDayRequest"`
		Latitude  float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl latitude"`
		Longitude float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl longitude"`
		StartDate xsdDate  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl startDate"`
		NumDays   int      `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl numDays"`
		Unit      Unit     `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl Unit"`
		Format    Format   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl format"`
	}
	input.Latitude = float64(v.Latitude)
	input.Longitude = float64(v.Longitude)
	input.StartDate = xsdDate(v.StartDate)
	input.NumDays = int(v.NumDays)
	input.Unit = Unit(v.Unit)
	input.Format = Format(v.Format)
	var output struct {
		XMLName      struct{} `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl NDFDgenByDayResponse"`
		DwmlByDayOut string   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl dwmlByDayOut"`
	}
	err := c.do("POST", "http://graphical.weather.gov/xml/SOAP_server/ndfdXMLserver.php", &input, &output)
	return string(output.DwmlByDayOut), err
}

type NDFDgenLatLonListRequest struct {
	ListLatLon        ListLatLon
	Product           Product
	StartTime         time.Time
	EndTime           time.Time
	Unit              Unit
	WeatherParameters WeatherParameters
}

// Returns National Weather Service digital weather forecast data
func (c *Client) NDFDgenLatLonList(v NDFDgenLatLonListRequest) (string, error) {
	var input struct {
		XMLName           struct{}          `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl NDFDgenLatLonListRequest"`
		ListLatLon        ListLatLon        `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl listLatLon"`
		Product           Product           `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl product"`
		StartTime         xsdDateTime       `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl startTime"`
		EndTime           xsdDateTime       `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl endTime"`
		Unit              Unit              `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl Unit"`
		WeatherParameters WeatherParameters `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl weatherParameters"`
	}
	input.ListLatLon = ListLatLon(v.ListLatLon)
	input.Product = Product(v.Product)
	input.StartTime = xsdDateTime(v.StartTime)
	input.EndTime = xsdDateTime(v.EndTime)
	input.Unit = Unit(v.Unit)
	input.WeatherParameters = WeatherParameters(v.WeatherParameters)
	var output struct {
		XMLName struct{} `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl NDFDgenLatLonListResponse"`
		DwmlOut string   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl dwmlOut"`
	}
	err := c.do("POST", "http://graphical.weather.gov/xml/SOAP_server/ndfdXMLserver.php", &input, &output)
	return string(output.DwmlOut), err
}

type NDFDgenByDayLatLonListRequest struct {
	ListLatLon ListLatLon
	StartDate  time.Time
	NumDays    int
	Unit       Unit
	Format     Format
}

// Returns National Weather Service digital weather forecast data summarized over either 24- or 12-hourly periods
func (c *Client) NDFDgenByDayLatLonList(v NDFDgenByDayLatLonListRequest) (string, error) {
	var input struct {
		XMLName    struct{}   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl NDFDgenByDayLatLonListRequest"`
		ListLatLon ListLatLon `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl listLatLon"`
		StartDate  xsdDate    `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl startDate"`
		NumDays    int        `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl numDays"`
		Unit       Unit       `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl Unit"`
		Format     Format     `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl format"`
	}
	input.ListLatLon = ListLatLon(v.ListLatLon)
	input.StartDate = xsdDate(v.StartDate)
	input.NumDays = int(v.NumDays)
	input.Unit = Unit(v.Unit)
	input.Format = Format(v.Format)
	var output struct {
		XMLName      struct{} `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl NDFDgenByDayLatLonListResponse"`
		DwmlByDayOut string   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl dwmlByDayOut"`
	}
	err := c.do("POST", "http://graphical.weather.gov/xml/SOAP_server/ndfdXMLserver.php", &input, &output)
	return string(output.DwmlByDayOut), err
}

type GmlLatLonListRequest struct {
	ListLatLon        ListLatLon
	RequestedTime     time.Time
	FeatureType       FeatureType
	WeatherParameters WeatherParameters
}

// Returns National Weather Service digital weather forecast data encoded in GML for a single time
func (c *Client) GmlLatLonList(v GmlLatLonListRequest) (string, error) {
	var input struct {
		XMLName           struct{}          `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl GmlLatLonListRequest"`
		ListLatLon        ListLatLon        `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl listLatLon"`
		RequestedTime     xsdDateTime       `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl requestedTime"`
		FeatureType       FeatureType       `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl featureType"`
		WeatherParameters WeatherParameters `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl weatherParameters"`
	}
	input.ListLatLon = ListLatLon(v.ListLatLon)
	input.RequestedTime = xsdDateTime(v.RequestedTime)
	input.FeatureType = FeatureType(v.FeatureType)
	input.WeatherParameters = WeatherParameters(v.WeatherParameters)
	var output struct {
		XMLName  struct{} `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl GmlLatLonListResponse"`
		DwGmlOut string   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl dwGmlOut"`
	}
	err := c.do("POST", "http://graphical.weather.gov/xml/SOAP_server/ndfdXMLserver.php", &input, &output)
	return string(output.DwGmlOut), err
}

type GmlTimeSeriesRequest struct {
	ListLatLon   ListLatLon
	StartTime    time.Time
	EndTime      time.Time
	CompType     CompType
	FeatureType  FeatureType
	PropertyName string
}

// Returns National Weather Service digital weather forecast data encoded in GML for a time period
func (c *Client) GmlTimeSeries(v GmlTimeSeriesRequest) (string, error) {
	var input struct {
		XMLName      struct{}    `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl GmlTimeSeriesRequest"`
		ListLatLon   ListLatLon  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl listLatLon"`
		StartTime    xsdDateTime `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl startTime"`
		EndTime      xsdDateTime `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl endTime"`
		CompType     CompType    `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl compType"`
		FeatureType  FeatureType `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl featureType"`
		PropertyName string      `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl propertyName"`
	}
	input.ListLatLon = ListLatLon(v.ListLatLon)
	input.StartTime = xsdDateTime(v.StartTime)
	input.EndTime = xsdDateTime(v.EndTime)
	input.CompType = CompType(v.CompType)
	input.FeatureType = FeatureType(v.FeatureType)
	input.PropertyName = string(v.PropertyName)
	var output struct {
		XMLName  struct{} `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl GmlTimeSeriesResponse"`
		DwGmlOut string   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl dwGmlOut"`
	}
	err := c.do("POST", "http://graphical.weather.gov/xml/SOAP_server/ndfdXMLserver.php", &input, &output)
	return string(output.DwGmlOut), err
}

type LatLonListSubgridRequest struct {
	LowerLeftLatitude   float64
	LowerLeftLongitude  float64
	UpperRightLatitude  float64
	UpperRightLongitude float64
	Resolution          float64
}

// Returns a list of latitude and longitude pairs in a rectangular subgrid defined by the lower left and upper right points
func (c *Client) LatLonListSubgrid(v LatLonListSubgridRequest) (ListLatLon, error) {
	var input struct {
		XMLName             struct{} `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl LatLonListSubgridRequest"`
		LowerLeftLatitude   float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl lowerLeftLatitude"`
		LowerLeftLongitude  float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl lowerLeftLongitude"`
		UpperRightLatitude  float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl upperRightLatitude"`
		UpperRightLongitude float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl upperRightLongitude"`
		Resolution          float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl resolution"`
	}
	input.LowerLeftLatitude = float64(v.LowerLeftLatitude)
	input.LowerLeftLongitude = float64(v.LowerLeftLongitude)
	input.UpperRightLatitude = float64(v.UpperRightLatitude)
	input.UpperRightLongitude = float64(v.UpperRightLongitude)
	input.Resolution = float64(v.Resolution)
	var output struct {
		XMLName       struct{}   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl LatLonListSubgridResponse"`
		ListLatLonOut ListLatLon `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl listLatLonOut"`
	}
	err := c.do("POST", "http://graphical.weather.gov/xml/SOAP_server/ndfdXMLserver.php", &input, &output)
	return ListLatLon(output.ListLatLonOut), err
}

type LatLonListLineRequest struct {
	EndPoint1Lat float64
	EndPoint1Lon float64
	EndPoint2Lat float64
	EndPoint2Lon float64
}

// Returns a list of latitude and longitude pairs along a line defined by the latitude and longitude of the 2 endpoints
func (c *Client) LatLonListLine(v LatLonListLineRequest) (ListLatLon, error) {
	var input struct {
		XMLName      struct{} `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl LatLonListLineRequest"`
		EndPoint1Lat float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl endPoint1Lat"`
		EndPoint1Lon float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl endPoint1Lon"`
		EndPoint2Lat float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl endPoint2Lat"`
		EndPoint2Lon float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl endPoint2Lon"`
	}
	input.EndPoint1Lat = float64(v.EndPoint1Lat)
	input.EndPoint1Lon = float64(v.EndPoint1Lon)
	input.EndPoint2Lat = float64(v.EndPoint2Lat)
	input.EndPoint2Lon = float64(v.EndPoint2Lon)
	var output struct {
		XMLName       struct{}   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl LatLonListLineResponse"`
		ListLatLonOut ListLatLon `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl listLatLonOut"`
	}
	err := c.do("POST", "http://graphical.weather.gov/xml/SOAP_server/ndfdXMLserver.php", &input, &output)
	return ListLatLon(output.ListLatLonOut), err
}

// Returns a list of latitude and longitude pairs with each pair corresponding to an input zip code.
func (c *Client) LatLonListZipCode(zipCodeList ZipCodeList) (ListLatLon, error) {
	var input struct {
		XMLName     struct{}    `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl LatLonListZipCodeRequest"`
		ZipCodeList ZipCodeList `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl zipCodeList"`
	}
	input.ZipCodeList = ZipCodeList(zipCodeList)
	var output struct {
		XMLName       struct{}   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl LatLonListZipCodeResponse"`
		ListLatLonOut ListLatLon `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl listLatLonOut"`
	}
	err := c.do("POST", "http://graphical.weather.gov/xml/SOAP_server/ndfdXMLserver.php", &input, &output)
	return ListLatLon(output.ListLatLonOut), err
}

type LatLonListSquareRequest struct {
	CenterPointLat float64
	CenterPointLon float64
	DistanceLat    float64
	DistanceLon    float64
	Resolution     float64
}

// Returns a list of latitude and longitude pairs in a rectangle defined by a central point and distance from that point in the latitudinal and longitudinal directions
func (c *Client) LatLonListSquare(v LatLonListSquareRequest) (ListLatLon, error) {
	var input struct {
		XMLName        struct{} `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl LatLonListSquareRequest"`
		CenterPointLat float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl centerPointLat"`
		CenterPointLon float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl centerPointLon"`
		DistanceLat    float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl distanceLat"`
		DistanceLon    float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl distanceLon"`
		Resolution     float64  `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl resolution"`
	}
	input.CenterPointLat = float64(v.CenterPointLat)
	input.CenterPointLon = float64(v.CenterPointLon)
	input.DistanceLat = float64(v.DistanceLat)
	input.DistanceLon = float64(v.DistanceLon)
	input.Resolution = float64(v.Resolution)
	var output struct {
		XMLName       struct{}   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl LatLonListSquareResponse"`
		ListLatLonOut ListLatLon `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl listLatLonOut"`
	}
	err := c.do("POST", "http://graphical.weather.gov/xml/SOAP_server/ndfdXMLserver.php", &input, &output)
	return ListLatLon(output.ListLatLonOut), err
}

// Returns four latitude and longitude pairs for corners of an NDFD grid and the minimum resolution that will return the entire grid
func (c *Client) CornerPoints(sector Sector) (ListLatLon, error) {
	var input struct {
		XMLName struct{} `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl CornerPointsRequest"`
		Sector  Sector   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl sector"`
	}
	input.Sector = Sector(sector)
	var output struct {
		XMLName       struct{}   `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl CornerPointsResponse"`
		ListLatLonOut ListLatLon `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl listLatLonOut"`
	}
	err := c.do("POST", "http://graphical.weather.gov/xml/SOAP_server/ndfdXMLserver.php", &input, &output)
	return ListLatLon(output.ListLatLonOut), err
}

// Returns a list of latitude and longitude pairs paired with the city names they correspond to
func (c *Client) LatLonListCityNames(displayLevel DisplayLevel) (ListCityNames, error) {
	var input struct {
		XMLName      struct{}     `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl LatLonListCityNamesRequest"`
		DisplayLevel DisplayLevel `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl displayLevel"`
	}
	input.DisplayLevel = DisplayLevel(displayLevel)
	var output struct {
		XMLName          struct{}      `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl LatLonListCityNamesResponse"`
		ListCityNamesOut ListCityNames `xml:"http://graphical.weather.gov/xml/DWMLgen/wsdl/ndfdXML.wsdl listCityNamesOut"`
	}
	err := c.do("POST", "http://graphical.weather.gov/xml/SOAP_server/ndfdXMLserver.php", &input, &output)
	return ListCityNames(output.ListCityNamesOut), err
}
