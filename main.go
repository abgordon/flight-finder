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
	  - then return cheapest total (or 10 cheapest)
	  - it's not ideal because it's the cheapest sum, but not necesssarily the cheapest flight for each person. That's kind of a tricky problem
	  - need to get more clever
*/

func main() {

	ss, err := util.NewSkyScanner("./util/airports.json")
	if err != nil {
		fmt.Printf("err instantiating API client: %v\n", err)
		os.Exit(1)
	}

	// ss.PrettyPrint()

	travelers := map[string]*util.Traveler{
		"andrew": util.NewTraveler("andrew", "DEN-sky"),
	}

	outboundDate := "2020-01-01"
	inboundDate := "2020-01-05"

	for _, location := range ss.List() {
		fmt.Println()
		fmt.Println("initiating session for", location)

		for _, traveler := range travelers {
			for {
				sessionKey, err := ss.InitSession(outboundDate, inboundDate, traveler.LocationCode, location.PlaceID)
				if err != nil {
					// try again. this shouldn't happen
					fmt.Println("error initiating session:", err.Error())
					continue
				}

				bestPrice, err := ss.PollSession(sessionKey)
				if err != nil {
					// try again. fake error
					if strings.Contains(err.Error(), "Rate limit has been exceeded") {
						continue
					}
					fmt.Println("error polling session:", err.Error())
					break
				}

				fmt.Printf("best price: %f to %s\n", bestPrice.Price, bestPrice.Deeplink)

				time.Sleep(1 * time.Second)
				break
			}
		}
	}
}
