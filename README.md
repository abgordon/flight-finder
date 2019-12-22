# Flight Finder

I was planning a trip with a large group of friends and one of them had an interesting idea: instead of picking a city, why don't we just let a program sift `all of the cities` and pick the cheapest average flight to an arbitrary destination? The trip is about catching up anyway, and this path may lead to an interesting destination. We could even tally up the flight costs and all pay an equal portion, so the furthest person from the destination doesn't get shafted. This tool probably exists somewhere, but it's fun to play around with this stuff.

This run loop is a wretched kludge, but it's for a reason - the SkyScanner API is super unreliable:

- sometimes a session key fails for no reason
- sometimes it doesn't return any results for a route for no reason
- it's rate limited, but it doesn't tell you if the request is failing because of the rate limit (whyyyy)
- it's rate limited, but I don't know what the limits are because I can't find them

Thus most of the horrendous looking collections of ifs and sleeps and weird logs are designed to harden this code a bit against that.

### TODO 12/21/2019:

- make it more configuration friendly
- less kludgy code
- write results to file
- most expensive flight feature
- dockerize it, put it on aws, build a frontend, write a server that outputs results to communicate with front....
