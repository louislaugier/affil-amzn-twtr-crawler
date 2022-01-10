package follower

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
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

// GetAmazonFollowerList gets Amazon's 10 latest Twitter followers & follows them + unfollows 10 oldest followers
func GetAmazonFollowerList() {
	log.Println("Begin following")

	// GET followers
	config := oauth1.NewConfig(os.Getenv("TWITTER_CONSUMER_KEY"), os.Getenv("TWITTER_CONSUMER_SECRET"))
	token := oauth1.NewToken(os.Getenv("TWITTER_ACCESS_TOKEN"), os.Getenv("TWITTER_ACCESS_SECRET"))
	httpClient := config.Client(oauth1.NoContext, token)

	resp, _ := httpClient.Get("https://api.twitter.com/2/users/20793816/followers")
	defer resp.Body.Close()
	following := followers{}
	json.NewDecoder(resp.Body).Decode(&following)

	// POST follow
	for k, v := range following.Data {
		if k < 10 {
			httpClient.Post("https://api.twitter.com/2/users/"+os.Getenv("TWITTER_ID")+"/following", "application/json", bytes.NewBuffer([]byte(`{"target_user_id": "`+v.ID+`"}`)))
		}
	}
	log.Println("Done following")

	log.Println("Begin unfollowing")
	// DELETE follow
	resp2, _ := httpClient.Get("https://api.twitter.com/2/users/" + os.Getenv("TWITTER_ID") + "/followers?max_results=1000")
	defer resp2.Body.Close()
	followers := followers{}
	json.NewDecoder(resp2.Body).Decode(&followers)
	// reverse slice
	for i, j := 0, len(followers.Data)-1; i < j; i, j = i+1, j-1 {
		followers.Data[i], followers.Data[j] = followers.Data[j], followers.Data[i]
	}
	for k, v := range followers.Data {
		if k < 10 {
			http.NewRequest("DELETE", "https://api.twitter.com/2/users/"+os.Getenv("TWITTER_ID")+"/following/"+v.ID, nil)
		}
	}
	log.Println("Done unfollowing")
}
