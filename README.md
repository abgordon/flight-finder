# Flight Finder

I was planning a trip with a large group of friends and one of them had an interesting idea: instead of picking a city, why don't we just let a program sift `all of the cities` and pick the cheapest average flight to an arbitrary destination? The trip is about catching up anyway, and this path may lead to an interesting destination. We could even tally up the flight costs and all pay an equal portion, so the furthest person from the destination doesn't get shafted. This tool probably exists somewhere, but it's fun to play around with this stuff.

This run loop is a wretched kludge, but it's for a reason - the SkyScanner API is super unreliable:

- sometimes a session key fails for no reason
- sometimes it doesn't return any results for a route for no reason
- it's rate limited, but it doesn't tell you if the request is failing because of the rate limit (whyyyy)
- it's rate limited, but I don't know what the limits are because I can't find them

Thus most of the horrendous looking collections of ifs and sleeps and weird logs are designed to harden this code a bit against that.

### Reverse engineering the API

The public API is _absolute bogus_. It would be nice to be able to return things directly from the private API, and dodge captcha requests somehow.

On a request via the www.skyscanner.com website, the network panel reveals this request:

`https://www.skyscanner.com/transport/flights/dena/iad/200119/200126/?adults=1&children=0&adultsv2=1&childrenv2=&infants=0&cabinclass=economy&rtn=1&preferdirects=false&outboundaltsenabled=false&inboundaltsenabled=false&ref=home`

With the following headers:

```
:authority: www.skyscanner.com
:method: GET
:path: /transport/flights/dena/iad/200119/200126/?adults=1&children=0&adultsv2=1&childrenv2=&infants=0&cabinclass=economy&rtn=1&preferdirects=false&outboundaltsenabled=false&inboundaltsenabled=false&ref=home
:scheme: https
accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9
accept-encoding: gzip, deflate, br
accept-language: en-US,en;q=0.9
cache-control: no-cache
cookie: <couple thousand bytes of gobbledygook>
pragma: no-cache
referer: https://www.skyscanner.com/?previousCultureSource=GEO_LOCATION&redirectedFrom=www.skyscanner.net
sec-fetch-mode: navigate
sec-fetch-site: same-origin
sec-fetch-user: ?1
upgrade-insecure-requests: 1
user-agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.88 Safari/537.36
```

And returns the bytes of an HTML document that has the following info, among other things, something called a `viewid`:

```
"utid": "e508f0bb-33f2-4eaa-8fac-ee0d6a732b10",
"viewId": "79e58113-943a-4f48-8602-f9e3c6485e91",
```

It seems that this can be used to make subsequent requests against the private API. This effort is in-progress.

### TODO 12/21/2019:

- make it more configuration friendly
- less kludgy code
- write results to file
- most expensive flight feature
- actually, sort everything by price, and put it in one big list as the output
- dockerize it, put it on aws, build a frontend, write a server that outputs results to communicate with front....
