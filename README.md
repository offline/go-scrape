# go-scrape
Go-Scrape is a GoLang web scraping framework. Go-Scrape provides a number of helpful methods to perform network requests, scrape web sites and process the scraped content

# usage example
```go
type SearchHandler struct{}

func (p *SearchHandler) Success(g *goscrape.GoScrape, o *goscrape.HttpOptions, doc *goquery.Document, args ...interface{}) {
	fmt.Println("Success")
}

func (p *SearchHandler) Fail(g *goscrape.GoScrape, url string) {
	fmt.Println("Failed")
}


func main() {
  scraper := goscrape.NewScraper()
  appURL := "https://play.google.com/store/search?q=games&c=apps&hl=en"
  httpOptions := goscrape.NewHttpOptions()
  scraper.AddTask(new(SearchHandler), appURL, httpOptions, 10)
  scraper.Start(3)
```
