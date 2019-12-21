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
	PollSession(sessionKey string) (*PricingOption, error)
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
func (s *skyScanner) PollSession(sessionKey string) (*PricingOption, error) {
	pollUrl := fmt.Sprintf("https://skyscanner-skyscanner-flight-search-v1.p.rapidapi.com/apiservices/pricing/uk2/v1.0/%s", sessionKey)
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
			for _, price := range itin.PricingOptions {
				fmt.Println("cost:", price.Price)
			}
			bestPrice := itin.PricingOptions[0]
			fmt.Println("returning:", bestPrice.Price)
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
