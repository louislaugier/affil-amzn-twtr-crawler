package deal

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/dnlo/struct2csv"
	"github.com/gocolly/colly"
)

var csvFile = "latest_products.csv"

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
									Savings struct {
										Percentage struct {
											Value int `json:"value"`
										} `json:"percentage"`
									} `json:"savings"`
									DealPrice struct {
										MoneyValueOrRange struct {
											Value struct {
												Amount string `json:"amount"`
											} `json:"value"`
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
	ID                 string
	Title              string
	MinPrice           float64
	MaxPrice           float64
	DiscountPercentage int
	NewPrice           float64
	URL                string
	Type               string
	TimeLeft           string
}

// GetDeals updates deal list and tweets new results
func GetDeals() {
	log.Println("Begin posting")
	c := colly.NewCollector()

	defer c.Visit("https://www.amazon.com/deals")

	c.OnRequest(func(r *colly.Request) {
		// fmt.Println("Visiting URL: ", r.URL)
	})

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
			d.DiscountPercentage = v.Entity.Details.Entity.Price.Details.Savings.Percentage.Value
			d.NewPrice, _ = strconv.ParseFloat(v.Entity.Details.Entity.Price.Details.DealPrice.MoneyValueOrRange.Value.Amount, 64)
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
		f, err := os.Open(csvFile)
		if err != nil {
			rows, _ = struct2csv.New().Marshal(deals)
			f, _ = os.Create(csvFile)
			defer f.Close()
			w := csv.NewWriter(f)
			w.WriteAll(rows)
		} else {
			rows, _ = csv.NewReader(f).ReadAll()
		}
		defer f.Close()

		prevLen := len(rows)
		for _, d := range deals {
			found := false
			for i, r := range rows {
				if i > 0 && d.ID == r[0] {
					found = true
				}
			}

			// if one deal is not in CSV and is available, tweet all its info
			if !found {
				c = colly.NewCollector()
				c.OnHTML("html", func(e *colly.HTMLElement) {
					if !strings.Contains(e.Text, "This deal is currently unavailable, but you can find more great deals on our Today’s Deals page.") {
						// new row for CSV
						rows = append(rows, []string{d.ID, d.Title, strconv.FormatFloat(d.MinPrice, 'f', -1, 64), strconv.FormatFloat(d.MaxPrice, 'f', -1, 64), strconv.Itoa(d.DiscountPercentage), strconv.FormatFloat(d.NewPrice, 'f', -1, 64), d.URL, d.Type, d.TimeLeft})

						// tweet info
						dealDiscount := strconv.Itoa(d.DiscountPercentage) + "% off! " + strconv.FormatFloat(d.NewPrice, 'f', -1, 64) + "$ only for "
						if d.DiscountPercentage == 0 || d.NewPrice == 0 || strings.Contains(d.Title, "%") {
							dealDiscount = ""
						}
						dealRange := " Deals going from $" + strconv.FormatFloat(d.MinPrice, 'f', -1, 64) + " to $" + strconv.FormatFloat(d.MaxPrice, 'f', -1, 64) + "."
						if d.MinPrice == 0 || d.MaxPrice == 0 {
							dealRange = ""
						}

						// tweet POST
						config := oauth1.NewConfig(os.Getenv("TWITTER_CONSUMER_KEY"), os.Getenv("TWITTER_CONSUMER_SECRET"))
						token := oauth1.NewToken(os.Getenv("TWITTER_ACCESS_TOKEN"), os.Getenv("TWITTER_ACCESS_SECRET"))
						httpClient := config.Client(oauth1.NoContext, token)
						resp, _ := httpClient.Post("https://api.twitter.com/2/tweets", "application/json", bytes.NewBuffer([]byte(`{"text": "`+dealDiscount+d.Title+`.`+dealRange+` Offer ends in `+d.TimeLeft+`. Deal type: `+d.Type+`. `+d.URL+`?tag=`+os.Getenv("AMAZON_AFFILIATE_TAG")+`"}`)))
						defer resp.Body.Close()
					}
				})
				c.Visit(d.URL + "?tag=" + os.Getenv("AMAZON_AFFILIATE_TAG"))
			}
		}

		// if new rows, recreate CSV
		if len(rows) > prevLen {
			os.Remove(csvFile)
			f, _ = os.Create(csvFile)
			defer f.Close()
			w := csv.NewWriter(f)
			w.WriteAll(rows)
		}
		log.Println("Done posting")
	})

}
