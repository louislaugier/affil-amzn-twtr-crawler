package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dnlo/struct2csv"
	colly "github.com/gocolly/colly/v2"
	"github.com/joho/godotenv"
)

type rawDealList struct {
	PrefetchedData struct {
		AapiGetDealsList []struct {
			Entities []struct {
				Entity struct {
					ID      string `json:"id"`
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
	ID       string
	Title    string
	MinPrice float64
	MaxPrice float64
	URL      string
	Type     string
	TimeLeft string
}

func getDeals(c *colly.Collector) {
	deals := []deal{}

	c.OnHTML("html", func(e *colly.HTMLElement) {
		// format products raw HTML into JSON string
		str := strings.Replace(strings.Replace(strings.Split(strings.Split(strings.Split(e.Text, "window.P.when('DealsWidgetsHorizonteAssets').execute(function (assets) {")[1], "});")[0], "assets.mountWidget('slot-15', ")[1], "\n", "", -1), ")            ", "", -1)

		// JSON string to struct
		rawDeals := rawDealList{}
		json.Unmarshal([]byte(str), &rawDeals)

		// parse raw deals into typed structs
		for _, v := range rawDeals.PrefetchedData.AapiGetDealsList[0].Entities {
			d := deal{}

			d.ID = v.Entity.ID
			d.Title = v.Entity.Details.Entity.Title
			d.MinPrice, _ = strconv.ParseFloat(v.Entity.Details.Entity.Price.Details.DealPrice.MoneyValueOrRange.Range.Min.Amount, 64)
			d.MaxPrice, _ = strconv.ParseFloat(v.Entity.Details.Entity.Price.Details.DealPrice.MoneyValueOrRange.Range.Max.Amount, 64)
			d.URL = "https://www.amazon.com/deal/" + v.Entity.ID + "?tag=" + os.Getenv("AMAZON_AFFILIATE_TAG")

			switch v.Entity.Details.Entity.Type {
			case "DEAL_OF_THE_DAY":
				d.Type = "Deal Of The Day"
			case "LIGHTNING_DEAL":
				d.Type = "Lightning Deal"
			case "BEST_DEAL":
				d.Type = "Best Deal"
			}

			tLeft := v.Entity.Details.Entity.EndTime.Value.Unix() - time.Now().Unix()
			days, hours, minutes, seconds := "", "", "", ""
			if tLeft >= 86400 {
				d := int(tLeft) / 86400
				days = strconv.Itoa(d) + " days "
				tLeft -= int64(86400 * d)
			}
			if tLeft >= 3600 {
				h := int(tLeft) / 3600
				hours = strconv.Itoa(h) + " hours "
				tLeft -= int64(3600 * h)
			}
			if tLeft >= 60 {
				m := int(tLeft) / 60
				minutes = strconv.Itoa(m) + " minutes "
				tLeft -= int64(60 * m)
			}
			if tLeft > 0 {
				seconds = strconv.Itoa(int(tLeft)) + " seconds "
			}
			d.TimeLeft = days + hours + minutes + seconds

			deals = append(deals, d)
		}

		// if csv file doesn't exist, create it and write slice of deals to it
		rows := [][]string{}
		f, err := os.Open("products.csv")
		defer f.Close()
		if err != nil {
			rows, _ = struct2csv.New().Marshal(deals)
			f, _ = os.Create("products.csv")
			defer f.Close()
			w := csv.NewWriter(f)
			w.WriteAll(rows)
		} else {
			rows, _ = csv.NewReader(f).ReadAll()
		}

		// for each deal, if one is not in CSV (check ID), add it to csv and tweet all its info
		for _, d := range deals {
			found := false
			for i, r := range rows {
				if i > 0 && d.ID == r[0] {
					found = true
				}
			}
			if !found {
				w := csv.NewWriter(f)
				w.Write([]string{d.ID, d.Title, strconv.FormatFloat(d.MinPrice, 'f', -1, 64), strconv.FormatFloat(d.MaxPrice, 'f', -1, 64), d.URL, d.Type, d.TimeLeft})
				// tweet
				fmt.Println("- Needs add to csv: ", d.Title)
			}
		}

		// https://developer.twitter.com/en/portal/products
	})

	c.OnRequest(func(r *colly.Request) {
		// fmt.Println("Visiting URL: ", r.URL)
	})

	c.Visit("https://www.amazon.com/deals")
}

func main() {
	// load .env
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	// routine to check every 15 mins
	c := colly.NewCollector()
	getDeals(c)
	// follow/unfollow bot
}
