package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/dnlo/struct2csv"
	colly "github.com/gocolly/colly/v2"
	"github.com/joho/godotenv"
)

// copy raw JSON structure from Amazon Deals page scrap
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

// formatted deal structure ready for tweet
type deal struct {
	ID       string
	Title    string
	MinPrice float64
	MaxPrice float64
	URL      string
	Type     string
	TimeLeft string
}

// get deal list
func getDeals(c *colly.Collector) {
	// on Amazon Deals page load
	c.OnHTML("html", func(e *colly.HTMLElement) {
		deals := []deal{}

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
			d.URL = "https://www.amazon.com/deal/" + v.Entity.ID

			switch v.Entity.Details.Entity.Type {
			case "DEAL_OF_THE_DAY":
				d.Type = "Deal Of The Day"
			case "LIGHTNING_DEAL":
				d.Type = "Lightning Deal"
			case "BEST_DEAL":
				d.Type = "Best Deal"
			}

			// compute remaining time for current deal
			tLeft := v.Entity.Details.Entity.EndTime.Value.Unix() - time.Now().Unix()
			days, hours, minutes, seconds := "", "", "", ""
			if tLeft >= 86400 {
				d := int(tLeft) / 86400
				days = strconv.Itoa(d) + " day(s) "
				tLeft -= int64(86400 * d)
			}
			if tLeft >= 3600 {
				h := int(tLeft) / 3600
				hours = strconv.Itoa(h) + " hour(s) "
				tLeft -= int64(3600 * h)
			}
			if tLeft >= 60 {
				m := int(tLeft) / 60
				minutes = strconv.Itoa(m) + " minute(s) "
				tLeft -= int64(60 * m)
			}
			if tLeft > 0 {
				seconds = strconv.Itoa(int(tLeft)) + " second(s)"
			}
			d.TimeLeft = days + hours + minutes + seconds

			deals = append(deals, d)
		}

		// if CSV file doesn't exist, create it and write the slice of deals to it
		rows := [][]string{}
		f, err := os.Open("latest_products.csv")
		if err != nil {
			rows, _ = struct2csv.New().Marshal(deals)
			f, _ = os.Create("latest_products.csv")
			defer f.Close()
			w := csv.NewWriter(f)
			w.WriteAll(rows)
		} else {
			rows, _ = csv.NewReader(f).ReadAll()
		}
		defer f.Close()

		// for each deal, if one is not in CSV, tweet all its info
		prevLen := len(rows)
		for _, d := range deals {
			found := false
			for i, r := range rows {
				if i > 0 && d.ID == r[0] {
					found = true
				}
			}
			if !found {
				rows = append(rows, []string{d.ID, d.Title, strconv.FormatFloat(d.MinPrice, 'f', -1, 64), strconv.FormatFloat(d.MaxPrice, 'f', -1, 64), d.URL, d.Type, d.TimeLeft})

				// tweet about the deal
				dealRange := ` Deals going from $` + strconv.FormatFloat(d.MinPrice, 'f', -1, 64) + ` to $` + strconv.FormatFloat(d.MaxPrice, 'f', -1, 64) + `.`
				if d.MinPrice == 0 || d.MaxPrice == 0 {
					dealRange = ""
				}
				config := oauth1.NewConfig(os.Getenv("TWITTER_CONSUMER_KEY"), os.Getenv("TWITTER_CONSUMER_SECRET"))
				token := oauth1.NewToken(os.Getenv("TWITTER_ACCESS_TOKEN"), os.Getenv("TWITTER_ACCESS_SECRET"))
				httpClient := config.Client(oauth1.NoContext, token)
				resp, _ := httpClient.Post("https://api.twitter.com/2/tweets", "application/json", bytes.NewBuffer([]byte(`{"text": "`+d.Title+`.`+dealRange+` Offer ends in `+d.TimeLeft+`. Deal type: `+d.Type+`. `+d.URL+`?tag=`+os.Getenv("AMAZON_AFFILIATE_TAG")+`"}`)))
				defer resp.Body.Close()
			}
		}

		// if new rows, recreate CSV
		if len(rows) > prevLen {
			os.Remove("latest_products.csv")
			f, _ = os.Create("latest_products.csv")
			defer f.Close()
			w := csv.NewWriter(f)
			w.WriteAll(rows)
		}
	})

	c.OnRequest(func(r *colly.Request) {
		// fmt.Println("Visiting URL: ", r.URL)
	})

	c.Visit("https://www.amazon.com/deals")
}

func main() {
	// if no Amazon affiliate tag env variable found, load .env file
	if os.Getenv("AMAZON_AFFILIATE_TAG") == "" {
		if err := godotenv.Load(); err != nil {
			panic(err)
		}
	}

	// create scraper and get latest deals
	c := colly.NewCollector()
	getDeals(c)

	// follow / unfollow bot
	// factorize & scale for other categories
}
