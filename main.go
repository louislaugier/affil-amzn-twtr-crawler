package main

import (
	"fmt"
	"strings"

	colly "github.com/gocolly/colly/v2"
)

func checkAccountLogged(c *colly.Collector) bool {
	logged := false
	c.OnHTML("a[data-nav-ref]", func(e *colly.HTMLElement) {
		if !strings.Contains(e.Text, "Sign in") {
			// browser logged in
			logged = true
		}
	})
	c.Visit("https://amazon.com/deals")
	return logged
}

func main() {
	c := colly.NewCollector()
	isAccountLogged := checkAccountLogged(c)
	fmt.Println(isAccountLogged)
}

// Find and visit all links
// c.OnHTML("a[href]", func(e *colly.HTMLElement) {
// 	e.Request.Visit(e.Attr("href"))
// })
// On link click
// c.OnRequest(func(r *colly.Request) {
// 	fmt.Println("Visiting", r.URL)
// })
