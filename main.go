package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/abgordon/flight-finder/util"
)

const (
	skyScanHost   = "skyscanner-skyscanner-flight-search-v1.p.rapidapi.com"
	skyScanAPIKey = "GJcM6vJ5FImshFOb5Gfv0Ccq4COSp15iGqnjsnsyF2kQrqfXDu"
)

/*
	plan:
	 - let's find all locations and save them to disk. Need a list of every town with an airport and save
	   it to disk. That will speed the initialization up
	 - then iterate over each departure airport, and each person, and find the cheapest flight to each
	 - Then sum them all
	  - then TTN-skyreturn cheapest total (or 10 cheapest)
	  - it's not ideal because it's the cheapest sum, but not necesssarily the cheapest flight for each person. That's kind of a tricky problem
	  - need to get more clever
*/

func main() {

	ss, err := util.NewSkyScanner("./util/airports-test-data.json")
	if err != nil {
		fmt.Printf("err instantiating API client: %v\n", err)
		os.Exit(1)
	}

	// ss.PrettyPrint()

	travelers := map[string]*util.Traveler{
		// "andrew": util.NewTraveler("andrew", "DEN-sky"),
		// "graham": util.NewTraveler("graham", "DEN-sky"),
		// "john": util.NewTraveler("john", "PIT-sky"),
		"kris": util.NewTraveler("kris", "PHL-sky"),
		// "aj": util.NewTraveler("aj", "ORD-sky"),
	}

	outboundDate := "2020-01-01"
	inboundDate := "2020-01-05"

	// rate limit thread
	requests := 0
	var rateLimitExceeded bool
	go func() {
		minuteTicker := time.NewTicker(60 * time.Second)
		for {
			select {
			case _ = <-minuteTicker.C:
				requests = 0
				rateLimitExceeded = false
			case _ = <-time.After(100 * time.Millisecond):
				if requests > 60 {
					rateLimitExceeded = true
				}
			}
		}
	}()

	var cheapest float64
	var tripCostTotal float64
	var cheapestTripKey string
	cheapest = 99999999.00 // arbitrary big number
	itineraries := map[string][]*util.PricingOption{}
	for _, location := range ss.List() {
		tripCostTotal = 0
		fmt.Println("initiating session for", location)

		for _, traveler := range travelers {
			var bestPrice *util.PricingOption
			// person already lives here
			if traveler.LocationCode == location.PlaceID {
				bestPrice = &util.PricingOption{
					Price:    0,
					Deeplink: "This person already lives here",
				}
				break
			}
			fmt.Println("searching flights for", traveler.Name)
			attempts := 0
			noLegsFound := 0
			for {
				if rateLimitExceeded {
					fmt.Println("rate limit exceeded, sleeping 1s")
					time.Sleep(1 * time.Second)
					continue
				}
				attempts++
				if attempts > 10 {
					fmt.Println("exceeded 10 attempts. Skipping this destination")
					break
				}
				sessionKey, err := ss.InitSession(outboundDate, inboundDate, traveler.LocationCode, location.PlaceID)
				requests++
				if err != nil {
					// try again. this shouldn't happen
					fmt.Println("error initiating session:", err.Error())
					continue
				}

				bestPrice, err = ss.PollSession(sessionKey, traveler.LocationCode, location.PlaceID)
				requests++
				if err != nil {
					// try again. fake error
					if strings.Contains(err.Error(), "Rate limit has been exceeded") {
						continue
					} else if strings.Contains(err.Error(), "no pricing option") {
						fmt.Println("no legs found, trying again....")
						noLegsFound++
						if noLegsFound > 5 {
							fmt.Println("no legs found limit exceeded; breaking")
							break
						}
					} else {
						fmt.Println("error polling session:", err.Error())
						break
					}
				}
				if bestPrice != nil {
					fmt.Printf("best price: %f to %s\n", bestPrice.Price, bestPrice.Deeplink)

					tripCostTotal += bestPrice.Price
					itineraries[location.PlaceID] = append(itineraries[location.PlaceID], bestPrice)
					break
				}
				// ease up on rate limiting for testing
				time.Sleep(1 * time.Second)
			}
		}

		// determine if it was cheaper and save the key
		fmt.Println("comparing:", tripCostTotal, cheapest)
		if tripCostTotal != 0 && tripCostTotal < cheapest {
			cheapestTripKey = location.PlaceID
			cheapest = tripCostTotal
		}
	}

	fmt.Printf("RESULTS: cheapest trip: [ %s ] cheapest cost: [ %f ] \n", cheapestTripKey, cheapest)
	for i, p := range itineraries[cheapestTripKey] {
		fmt.Printf("Itinerary %d: %+v\n", i, p)
	}
}
