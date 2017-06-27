package goscrape

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
)

const (
	Low    int = 0
	Medium int = 1
	High   int = 2
)

// Scraper initializer
func NewScraper() *GoScrape {
	ch := []chan *Task{make(chan *Task), make(chan *Task), make(chan *Task)}
	return &GoScrape{ch: ch}
}

type GoScrape struct {
	wg sync.WaitGroup
	ch []chan *Task
}

func (g *GoScrape) AddTask(h Handler, url string, o *HttpOptions, priority int, args ...interface{}) {
	g.wg.Add(1)
	go func() {
		task := Task{h, url, o, args}
		g.ch[priority] <- &task
	}()
}

func GetCookies(uri string, options *HttpOptions) *cookiejar.Jar {
	jar, _ := cookiejar.New(nil)
	if u, err := url.Parse(uri); err != nil {
		fmt.Errorf("URL has wrong format", uri)
		return jar
	} else {
		jar.SetCookies(u, options.Cookies())
	}
	return jar
}

func (g *GoScrape) PrepareReq(uri string, o *HttpOptions) (*http.Request, error) {

	var buf bytes.Buffer
	req, err := http.NewRequest(o.Method(), uri, &buf)
	if err != nil {
		fmt.Println("Error creating request", uri, "-", err)
		return nil, err
	}
	for key, value := range o.Headers() {
		req.Header.Add(key, value)
	}
	return req, nil
}

func (g *GoScrape) Scrape(t *Task) {
	defer g.wg.Done()

	jar := GetCookies(t.Url, t.Options)
	client := &http.Client{Jar: jar}

	req, err := g.PrepareReq(t.Url, t.Options)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error while downloading", t.Url, "-", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		//fmt.Println(resp.Cookies())
		cookies := resp.Cookies()
		if len(cookies) > 0 {
			t.Options.SetCookies(cookies)
		}

		doc, err := goquery.NewDocumentFromReader(resp.Body)

		if err != nil {
			fmt.Println("Error parsing body of", t.Url, "-", err)
			return
		}
		t.Handler.Success(g, t.Options, doc, t.Args...)
	} else {
		fmt.Println(resp.StatusCode)
	}
}

func (g *GoScrape) Worker() {
	for {
		select {
		case task := <-g.ch[High]:
			g.Scrape(task)
		case task := <-g.ch[Medium]:
			g.Scrape(task)
		case task := <-g.ch[Low]:
			g.Scrape(task)
		}
	}
}

func (g *GoScrape) Start(workers int) {
	fmt.Println("Started")
	for i := 0; i < workers; i++ {
		go g.Worker()
	}
	g.wg.Wait()
	close(g.ch[High])
	close(g.ch[Medium])
	close(g.ch[Low])
	fmt.Println("Finished")
}
