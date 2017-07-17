# go-scrape
Go-Scrape is a GoLang web scraping framework. Go-Scrape provides a number of helpful methods to perform network requests, scrape web sites and process the scraped content

# usage example
```go
package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"goscrape"
	"net/http"
)

type BlogPostHandler struct{}

func (p *BlogPostHandler) Success(
	g *goscrape.GoScrape,
	o *goscrape.HttpOptions,
	doc *goquery.Document,
	req *http.Request,
	args ...interface{},
) {
	doc.Find("h3.title a").Each(func(i int, s *goquery.Selection) {
		fmt.Println(s.Text())
	})
}

func (p *BlogPostHandler) Fail(g *goscrape.GoScrape, url string) {
	fmt.Println("Failed")
}

type BlogIndexHandler struct{}

func (h *BlogIndexHandler) Success(
	g *goscrape.GoScrape,
	o *goscrape.HttpOptions,
	doc *goquery.Document,
	req *http.Request,
	args ...interface{},
) {
	doc.Find(".blogtitle a").Each(func(i int, s *goquery.Selection) {
		link := fmt.Sprint(req.URL.Scheme, "://", req.URL.Host, s.AttrOr("href", ""))
		g.AddTask(new(BlogPostHandler), link, o, 100)
	})
}

func (h *BlogIndexHandler) Fail(g *goscrape.GoScrape, url string) {
	fmt.Println("Failed")
}

func main() {
	scraper := goscrape.NewScraper()
	url := "https://blog.golang.org/index"
	httpOptions := goscrape.NewHttpOptions()
	scraper.AddTask(new(BlogIndexHandler), url, httpOptions, 10)
	scraper.Start(3)
}
```
