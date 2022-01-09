package main

import (
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/louislaugier/affil-amzn-twtr-crawler/deal"
	"github.com/louislaugier/affil-amzn-twtr-crawler/follower"
)

func refresh(d time.Duration, refresh func(time.Time)) {
	for tick := range time.Tick(d) {
		refresh(tick)
	}
}

func schedule(f func(), delay time.Duration) {
	f()
	select {
	case <-time.After(delay):
		schedule(f, delay)
	}
}

func main() {
	// if no Amazon affiliate tag env variable found, load .env file
	if os.Getenv("AMAZON_AFFILIATE_TAG") == "" {
		if err := godotenv.Load(); err != nil {
			panic(err)
		}
	}

	go schedule(deal.GetDeals, 1*time.Minute)
	schedule(follower.GetAmazonFollowerList, 15*time.Minute)
}
