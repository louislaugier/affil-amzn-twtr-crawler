package follower

import (
	"bytes"
	"encoding/json"
	"log"
	"os"

	"github.com/dghubble/oauth1"
)

type follower struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type followers struct {
	Data []follower `json:"data"`
	Meta struct {
		ResultCount int    `json:"result_count"`
		NextToken   string `json:"next_token"`
	} `json:"meta"`
}

// GetAmazonFollowerList gets Amazon's 50 latest Twitter followers & follows them
func GetAmazonFollowerList() {
	log.Println("Starting 2")
	// GET followers
	config := oauth1.NewConfig(os.Getenv("TWITTER_CONSUMER_KEY"), os.Getenv("TWITTER_CONSUMER_SECRET"))
	token := oauth1.NewToken(os.Getenv("TWITTER_ACCESS_TOKEN"), os.Getenv("TWITTER_ACCESS_SECRET"))
	httpClient := config.Client(oauth1.NoContext, token)
	resp, _ := httpClient.Get("https://api.twitter.com/2/users/20793816/followers")
	defer resp.Body.Close()
	followers := followers{}
	json.NewDecoder(resp.Body).Decode(&followers)

	// POST follow
	for k, v := range followers.Data {
		if k < 30 {
			httpClient.Post("https://api.twitter.com/2/users/"+os.Getenv("TWITTER_ID")+"/following", "application/json", bytes.NewBuffer([]byte(`{"target_user_id": "`+v.ID+`"}`)))
		}
	}
	log.Println("Done 2")
}
