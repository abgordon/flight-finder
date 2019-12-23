package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"sort"
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
	Price      float64 `json:"Price"`
	Deeplink   string  `json:"DeeplinkUrl"`
	Location   string  `json:"Location"`
	SrcAirport string  `json:"SrcAirport`
	DstAirport string  `json:"DstAirport`
}

type Trips [][]*PricingOption

// go is so goddamn stupid sometimes
func IterTripsAndPrint(trips [][]*PricingOption) {
	for _, trip := range trips {
		fmt.Printf("\nLOCATION: %s\nTOTAL PRICE: $%f\n\n", trip[0].Location, SumPricingOptList(trip))
	}
}

type PricingOptionList []*PricingOption

func SumPricingOptList(p []*PricingOption) float64 {
	var sum float64
	for _, flight := range p {
		sum += flight.Price
	}
	return sum
}

// sort option for pricing option list so we can output results nicely
func (t Trips) Len() int {
	return len(t)
}
func (t Trips) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
func (t Trips) Less(i, j int) bool {
	return SumPricingOptList(t[i]) < SumPricingOptList(t[j])
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

func FilterJSON(locations []Location, filters ...string) *LocationWrapper {
	res := &LocationWrapper{
		Places: []Location{},
	}

	for _, l := range locations {
		for _, f := range filters {
			if l.CountryName == f {
				res.Places = append(res.Places, l)
			}
		}
	}

	return res
}

func WriteResultsToFile(travelers map[string]*Traveler, itineraries map[string][]*PricingOption) {

	// sort and write EVERY time bc this thing takes forever, and a write is cheap
	viableTrips := Trips{}
	nonViableTrips := Trips{}
	for _, p := range itineraries {
		if len(p) == len(travelers) {
			viableTrips = append(viableTrips, p)
		} else {
			nonViableTrips = append(nonViableTrips, p)
		}
	}

	sort.Sort(viableTrips)
	sort.Sort(nonViableTrips)

	bytesViableTrips, err := json.Marshal(viableTrips)
	if err != nil {
		fmt.Println("error marshaling json:", err.Error())
	}

	bytesNonViableTrips, err := json.Marshal(nonViableTrips)
	if err != nil {
		fmt.Println("error marshaling json:", err.Error())
	}

	err = ioutil.WriteFile("./results-viable.json", bytesViableTrips, 0644)
	if err != nil {
		fmt.Println("error writing to file:", err.Error())
	}

	err = ioutil.WriteFile("./results-non-viable.json", bytesNonViableTrips, 0644)
	if err != nil {
		fmt.Println("error writing to file:", err.Error())
	}
}

// pretty print the results
func OutputResults() {
	viableTripsJSON, err := ioutil.ReadFile("../results-viable.json")
	if err != nil {
		fmt.Println("error reading file:", err.Error())
		return
	}

	nonViableTripsJSON, err := ioutil.ReadFile("../results-non-viable.json")
	if err != nil {
		fmt.Println("error reading file:", err.Error())
		return
	}

	viable := [][]*PricingOption{}
	nonViable := [][]*PricingOption{}

	err = json.Unmarshal(viableTripsJSON, &viable)
	if err != nil {
		fmt.Println("error unmarshaling:", err.Error())
		return
	}

	err = json.Unmarshal(nonViableTripsJSON, &nonViable)
	if err != nil {
		fmt.Println("error unmarshaling:", err.Error())
		return
	}

	fmt.Printf("\n\n====================================\n===== VIABLE TRIPS\n====================================\n\n")
	IterTripsAndPrint(viable)

	fmt.Printf("\n\n====================================\n===== NON VIABLE TRIPS\n====================================\n\n")
	IterTripsAndPrint(nonViable)

}
