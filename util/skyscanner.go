package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SkyScanner is rate limited to 50 requests per minute
type SkyScanner interface {
	PrettyPrint()
	List() []Location
	GetLocation(location string) ([]Location, error)
	InitSession(outboundDate, inboundDate, departureAirport string, destinationAirport string) (string, error)
	PollSession(sessionKey, departureAirport, destinationAirport, placeName string) (*PricingOption, error)
	CreateView() (string, error)
	InitSessionCommercial(utid, outboundDate, inboundDate, departureAirport string, destinationAirport string) (string, error)
}

type skyScanner struct {
	client   *http.Client
	airports []Location
}

func NewSkyScanner(jsonLocation string) (SkyScanner, error) {
	airportJSON, err := ioutil.ReadFile(jsonLocation)
	if err != nil {
		return nil, err
	}

	locations := &LocationWrapper{}
	err = json.Unmarshal(airportJSON, &locations)
	if err != nil {
		return nil, err
	}

	return &skyScanner{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		airports: locations.Places,
	}, nil
}

func (s *skyScanner) InitSessionCommercial(utid, outboundDate, inboundDate, departureAirport string, destinationAirport string) (string, error) {
	sessionURI := "https://www.skyscanner.de/conductor/v1/fps3/search/?geo_schema=skyscanner&carrier_schema=skyscanner&response_include=query;deeplink;segment;stats;fqs;pqs"

	req := newSessionRequest(sessionURI)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("err on view creation request: %s", err.Error())
	}

	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)
	fmt.Println("body:", string(body))

	return "", nil
}

func (s *skyScanner) CreateView() (string, error) {
	viewURL := "https://www.skyscanner.de/transport/flights/nyca/wasa/191216/191223/?adults=1&children=0&adultsv2=1&childrenv2=&infants=0&cabinclass=economy&rtn=1&preferdirects=false&outboundaltsenabled=false&inboundaltsenabled=false&ref=home#/"
	req := newAuthedMethod(http.MethodGet, viewURL, &bytes.Buffer{})

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("err on view creation request: %s", err.Error())
	}

	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)
	fmt.Println("body:", string(body))
	// p := &PollResponse{}
	// err = json.Unmarshal(body, &p)
	// if err != nil {
	// 	return nil, fmt.Errorf("error unmarshaling poll response: %s", err.Error())
	// }

	return "", nil
}

// accept date as 2020-01-01
func (s *skyScanner) InitSession(outboundDate, inboundDate, departureAirport string, destinationAirport string) (string, error) {
	pollURL := "https://skyscanner-skyscanner-flight-search-v1.p.rapidapi.com/apiservices/pricing/v1.0"

	payload := strings.NewReader(fmt.Sprintf("inboundDate=%s&cabinClass=economy&children=0&infants=0&country=US&currency=USD&locale=en-US&originPlace=%s&destinationPlace=%s&outboundDate=%s&adults=1", inboundDate, departureAirport, destinationAirport, outboundDate))

	req := newAuthedMethodFromReader(http.MethodPost, pollURL, payload)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("err on request: %s", err.Error())
	}

	defer res.Body.Close()

	locationURL := res.Header.Get("location")
	urlParsed, err := url.Parse(locationURL)
	if err != nil {
		return "", fmt.Errorf("err parsing url: %s", err.Error())
	}

	locationURLSpl := strings.Split(urlParsed.Path, "/")
	sessionKey := locationURLSpl[len(locationURLSpl)-1]

	if sessionKey == "" {
		return "", fmt.Errorf("no session key was created; exiting")
	}

	return sessionKey, nil

}

// PollSession can sort by price, a src airport, and an _array_ of dst airports
// with this, we can sift through a large result set in-memory with 1 http call
func (s *skyScanner) PollSession(sessionKey, departureAirport, destinationAirport, placeName string) (*PricingOption, error) {

	pollUrl := fmt.Sprintf("https://skyscanner-skyscanner-flight-search-v1.p.rapidapi.com/apiservices/pricing/uk2/v1.0/%s?sortType=price&sortOrder=asc&originAirports=%s&destinationAirports=%s&pageIndex=0&pageSize=10", sessionKey, departureAirport, destinationAirport)
	fmt.Println("pollurl:", pollUrl)
	initReq := newAuthedMethod(http.MethodGet, pollUrl, &bytes.Buffer{})
	res, err := s.client.Do(initReq)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	p := &PollResponse{}
	err = json.Unmarshal(body, &p)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling poll response: %s", err.Error())
	}

	if p.ValidationErrs != nil && p.ValidationErrs.Message != "" {
		return nil, fmt.Errorf("poll response saw validation err: %s", err.Error())
	}

	if len(p.Itineraries) > 0 {
		itin := p.Itineraries[0]
		if len(itin.PricingOptions) > 0 {
			bestPrice := itin.PricingOptions[0]
			bestPrice.Location = placeName
			bestPrice.SrcAirport = departureAirport
			bestPrice.DstAirport = destinationAirport
			return bestPrice, nil
		}
	}

	return nil, fmt.Errorf("no pricing option was found for this leg")
}

// GetLocation get airport codes for use in polling from a semantic string, like "Denver" || "Washington, DC"
func (s *skyScanner) GetLocation(location string) ([]Location, error) {
	fmt.Printf("finding skyscanner locations for city: %s\n", location)
	baseURL := "https://skyscanner-skyscanner-flight-search-v1.p.rapidapi.com/apiservices/autosuggest/v1.0/UK/GBP/en-GB/?query="
	req := newAuthedMethod(http.MethodGet, fmt.Sprintf("%s%s", baseURL, location), &bytes.Buffer{})

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// dumb way to see under the response cover
	fmt.Println("b:", string(b))

	locations := &LocationWrapper{
		Places: []Location{},
	}

	err = json.Unmarshal(b, locations)
	if err != nil {
		return nil, err
	}

	fmt.Println("returning:", locations.Places)
	return locations.Places, nil
}

func (s *skyScanner) PrettyPrint() {
	for _, l := range s.airports {
		fmt.Printf("[ %s ] ID [ %s ] CountryID [ %s ] RegionID [ %s ] CityID [ %s ] CountryName [ %s ] \n", l.PlaceName, l.PlaceID, l.CountryID, l.RegionID, l.CityID, l.CountryName)
	}
}

func (s *skyScanner) List() []Location {
	return s.airports
}

func newSessionRequest(url string) *http.Request {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("accept", "application/json")
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("origin", "https://www.skyscanner.net")
	req.Header.Set("pragma", "no-cache")
	req.Header.Set("referer", "https://www.skyscanner.net/transport/flights/nyca/wasa/191217/191224/?adults=1&children=0&adultsv2=1&childrenv2=&infants=0&cabinclass=economy&rtn=1&preferdirects=false&outboundaltsenabled=false&inboundaltsenabled=false&ref=home")
	req.Header.Set("user-agent", "Mozilla/5.0 (X11; U; Linux i686) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.94 Safari/537.36 OPR/46.0.2137.58")
	req.Header.Set("x-skyscanner-channelid", "website")
	req.Header.Set("x-skyscanner-devicedetection-ismobile", "false")
	req.Header.Set("x-skyscanner-devicedetection-istablet", "false")
	req.Header.Set("x-skyscanner-traveller-context", "dac2aaf8-723d-4d2c-bcd9-cfca01a33b73")
	req.Header.Set("x-skyscanner-utid", "dac2aaf8-723d-4d2c-bcd9-cfca01a33b73")
	req.Header.Set("x-skyscanner-viewid", "368174fc-cedd-445f-8151-c7cc77b7b763")

	return req
}

// auto-apply auth headers
func newAuthedMethod(method, url string, data *bytes.Buffer) *http.Request {

	req, _ := http.NewRequest(method, url, data)
	req.Header.Set("x-rapidapi-host", "skyscanner-skyscanner-flight-search-v1.p.rapidapi.com")
	req.Header.Set("x-rapidapi-key", "GJcM6vJ5FImshFOb5Gfv0Ccq4COSp15iGqnjsnsyF2kQrqfXDu")

	if method == http.MethodPost {
		req.Header.Set("content-type", "application/x-www-form-urlencoded")
	}
	return req
}

// auto-apply auth headers
func newAuthedMethodFromReader(method, url string, data *strings.Reader) *http.Request {

	req, _ := http.NewRequest(method, url, data)
	req.Header.Set("x-rapidapi-host", "skyscanner-skyscanner-flight-search-v1.p.rapidapi.com")
	req.Header.Set("x-rapidapi-key", "GJcM6vJ5FImshFOb5Gfv0Ccq4COSp15iGqnjsnsyF2kQrqfXDu")

	if method == http.MethodPost {
		req.Header.Set("content-type", "application/x-www-form-urlencoded")
	}
	return req

}

/* delete probly

func (s *skyScanner) InitSessionSave(departureAirport string, destinationAirports []string) (string, error) {

	// skyscanner accepts data as multipart form; add src-dst airports here
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	params := map[string]string{
		"src-airports": "",
		"dst-airports": departureAirport,
	}

	for key, val := range params {
		err := writer.WriteField(key, val)
		if err != nil {
			return "", err
		}
	}

	// dont forget this or the end-of-form bytes will not be written, making it invalid
	err := writer.Close()
	if err != nil {
		return "", err
	}

	baseURL := "https://skyscanner-skyscanner-flight-search-v1.p.rapidapi.com/apiservices/pricing/v1.0"
	initReq := newAuthedMethod(http.MethodPost, baseURL, &bytes.Buffer{})

	resp, err := s.client.Do(initReq)
	if err != nil {
		return "", err
	}

	return resp.Header.Get("location"), nil
}
*/
