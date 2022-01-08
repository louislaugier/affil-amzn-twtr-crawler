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

func main() {
	// if no Amazon affiliate tag env variable found, load .env file
	if os.Getenv("AMAZON_AFFILIATE_TAG") == "" {
		if err := godotenv.Load(); err != nil {
			panic(err)
		}
	}

	deal.GetDeals(time.Time{})

	refresh(3*time.Minute, deal.GetDeals)

	// follow bot
	follower.GetAmazonFollowerList(time.Time{})

	refresh(15*time.Minute, follower.GetAmazonFollowerList)
}
