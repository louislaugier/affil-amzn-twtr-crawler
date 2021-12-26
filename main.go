package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	colly "github.com/gocolly/colly/v2"
	"github.com/joho/godotenv"
)

type rawDealList struct {
	PrefetchedData struct {
		AapiGetDealsList []struct {
			Entities []struct {
				Resource struct {
					URL string `json:"url"`
				} `json:"resource"`
				Entity struct {
					Details struct {
						Entity struct {
							Price struct {
								Details struct {
									DealPrice struct {
										MoneyValueOrRange struct {
											Range struct {
												Min struct {
													Amount string `json:"amount"`
												} `json:"min"`
												Max struct {
													Amount string `json:"amount"`
												} `json:"max"`
											} `json:"range"`
										} `json:"moneyValueOrRange"`
									} `json:"dealPrice"`
								} `json:"details"`
							} `json:"price"`
							EndTime struct {
								Value time.Time `json:"value"`
							} `json:"endTime"`
							Type  string `json:"type"`
							Title string `json:"title"`
						} `json:"entity"`
					} `json:"details"`
				} `json:"entity"`
			} `json:"entities"`
		} `json:"aapiGetDealsList"`
	} `json:"prefetchedData"`
}

type deal struct {
	Title   string
	Price   float64
	URL     string
	Type    string
	EndDate time.Time
}

// &tag=
func getDeals(c *colly.Collector, page int) []deal {
	deals := []deal{}
	c.OnHTML("html", func(e *colly.HTMLElement) {
		// format products json string
		str := strings.Replace(strings.Replace(strings.Split(strings.Split(strings.Split(e.Text, "window.P.when('DealsWidgetsHorizonteAssets').execute(function (assets) {")[1], "});")[0], "assets.mountWidget('slot-15', ")[1], "\n", "", -1), ")            ", "", -1)

		// json string to map
		rawDeals := rawDealList{}
		json.Unmarshal([]byte(str), &rawDeals)

		fmt.Println(rawDeals.PrefetchedData.AapiGetDealsList[0].Entities[0])
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.Visit("https://www.amazon.com/gp/goldbox?ref_=nav_cs_gb&deals-widget=%257B%2522version%2522%253A1%252C%2522viewIndex%2522%253A" + strconv.Itoa(60*page-60) + "%252C%2522presetId%2522%253A%2522024BF8E73BAAA8C1AB5B6D205172D8CE%2522%252C%2522sorting%2522%253A%2522BY_CUSTOM_CRITERION%2522%257D")
	return deals
}

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
	c := colly.NewCollector()
	getDeals(c, 1)
	// dealsPage1 := getDeals(c, 1)
	// dealsPage2 := getDeals(c, 2)
	// fmt.Println(dealsPage1, dealsPage2)
}
