# Flight Finder

I was planning a trip with a large group of friends and one of them had an interesting idea: instead of picking a city, why don't we just let a program sift `all of the cities` and pick the cheapest average flight to an arbitrary destination? The trip is about catching up anyway, and this path may lead to an interesting destination. We could even tally up the flight costs and all pay an equal portion, so the furthest person from the destination doesn't get shafted. This tool probably exists somewhere, but it's fun to play around with this stuff.

### Skyscanner sucks

Warning: Don't use the sky scanner API. I started playing around with it because it was available on rapid API, but it's _complete garbage_. Just for grins, because I know they won't respond to me, I sent a bitchy letter to their customer support. The text of this is in the `complaints.txt` file. The post-submission screen reads "we receive many communications from customers and are unable to respond to them all," natch. Oh yeah, and the [new account link]("https://www.partners.skyscanner.net/log-in/create-account") gives you a 404. Not included in my complaint is the false rate limiting errors, or failure to create session keys with no error. 

I wish I had seen the red flags earlier but it didn't take too much coding time to find them. I'm really baffled by this service. Why does it exist? Who built it? Who maintains it? It's all crazy to me. I can't believe _any_ API is this bad.

### TODO 12/21/2019:

- pick a new API!
- make it more configurable
