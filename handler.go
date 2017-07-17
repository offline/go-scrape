package goscrape

import (
	"github.com/PuerkitoBio/goquery"
	"net/http"
)

type Handler interface {
	Success(g *GoScrape, o *HttpOptions, doc *goquery.Document, req *http.Request, args ...interface{})
	Fail(g *GoScrape, url string)
}
