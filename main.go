package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/louislaugier/affil-amzn-twtr-crawler/deal"
)

func schedule() {
	client := &http.Client{}
	client.Get(os.Getenv("APP_URL"))
	mins := time.Now().Minute()
	if mins%3 == 0 {
		deal.GetDeals()
	}
	// if mins%16 == 0 || mins == 0 {
	// 	follower.GetAmazonFollowerList()
	// }
	time.Sleep(52 * time.Second)
	schedule()
}

func main() {
	// if no Amazon affiliate tag env variable found, load .env file
	if os.Getenv("AMAZON_AFFILIATE_TAG") == "" {
		if err := godotenv.Load(); err != nil {
			panic(err)
		}
	}

	go schedule()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Refreshing")
	})
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}
