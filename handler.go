package goscrape

import "github.com/PuerkitoBio/goquery"

type Handler interface {
	Success(g *GoScrape, o *HttpOptions, doc *goquery.Document, args ...interface{})
	Fail(g *GoScrape, url string)
}
