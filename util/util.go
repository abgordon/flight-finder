package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"time"
)

type Traveler struct {
	Name         string `json:"name"`
	LocationCode string `json:"location_code"`
	PriceOptions map[int]*PricingOption
}

func NewTraveler(name, location string) *Traveler {
	return &Traveler{
		Name:         name,
		LocationCode: location,
	}
}

type PollResponse struct {
	ValidationErrs *validationErrs `json:"ValidationErrors`
	// omitted: Query, status, legs, etc
	Itineraries []*Itinerary `json:"Itineraries"`
}

type validationErrs struct {
	Message string `json:"Message"`
}

type Itinerary struct {
	PricingOptions []*PricingOption `json:"PricingOptions`
}

type PricingOption struct {
	Price    float64 `json:"Price"`
	Deeplink string  `json:"DeeplinkUrl"`
}

type LocationWrapper struct {
	Places []Location `json:"Places"`
}
type Location struct {
	PlaceID     string `json:"PlaceId"`
	PlaceName   string `json:"PlaceName"`
	CountryID   string `json:"CountryId"`
	RegionID    string `json:"RegionId"`
	CityID      string `json:"CityId"`
	CountryName string `json:"CountryName"`
}

// get locations. dont really need this anymore with NewSkyscanner
func Locations() (*LocationWrapper, error) {
	airports, err := ioutil.ReadFile("./util/airports.json")
	if err != nil {
		return nil, err
	}

	allAirports := &LocationWrapper{
		Places: []Location{},
	}

	err = json.Unmarshal(airports, &allAirports)
	if err != nil {
		return nil, err
	}

	return allAirports, nil
}

// non-go way of reading file line by line, to output json
// make api calls from semantic location strings and get an airport location json back
func GetLocationsJSON(ss SkyScanner) (*LocationWrapper, error) {
	airports, err := ioutil.ReadFile("./util/airports")
	if err != nil {
		return nil, err
	}

	allAirports := &LocationWrapper{
		Places: []Location{},
	}

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt)
	go func() {
		select {
		case s := <-sigChan:
			fmt.Printf("caught signal: %v, cleaning up\n", s)
			b, err := json.Marshal(allAirports)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				os.Exit(1)
			}

			err = ioutil.WriteFile("airports.json", b, 0644)
			if err != nil {
				fmt.Println("err writing to file:", err.Error())
				os.Exit(1)
			}

			fmt.Println("successfully cleaned up, exiting")
			os.Exit(0)
		}
	}()

	airportsString := strings.Split(string(airports), "\n")
	for _, s := range airportsString {
		if s == "" {
			continue
		}

		l, err := ss.GetLocation(s)
		if err != nil {
			fmt.Printf("[ ERROR ] could not find location %s: %v\n", s, err)
			continue
		}

		// fucking rate limiting bullshit
		time.Sleep(1250 * time.Millisecond)

		allAirports.Places = append(allAirports.Places, l...)
	}

	return allAirports, nil
}

func InitLocations(ss SkyScanner) (*LocationWrapper, error) {
	locations, err := GetLocationsJSON(ss)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(1)
	}

	b, err := json.Marshal(locations)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		os.Exit(1)
	}

	err = ioutil.WriteFile("airports.json", b, 0644)
	if err != nil {
		fmt.Println("err writing to file:", err.Error())
		os.Exit(1)
	}

	return nil, nil
}
