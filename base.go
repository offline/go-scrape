package goscrape

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
	"os"
	"path/filepath"
	"strconv"
	"io/ioutil"
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
	tid int
	mu sync.Mutex
	logdir string
}

func (g *GoScrape) Incr() int {
	g.mu.Lock()
	taskId := g.tid+1
	g.tid = taskId
	g.mu.Unlock()
	return taskId
}

func (g *GoScrape) StoreInfo(req *http.Request, resp *http.Response) (err error) {
	if g.logdir != "" {

		hfname := fmt.Sprint(strconv.Itoa(g.tid), ".headers")
		hpath := filepath.Join(g.logdir, hfname)
		hf, err := os.Create(hpath)
		if err != nil {
			Error.Println("Can't create file:", hfname)
			return err
		}
		defer hf.Close()
		err = req.Header.Write(hf)
		if err != nil {
			Error.Println("Can't write headers to:", hfname)
			return err
		}
		rfname := fmt.Sprint(strconv.Itoa(g.tid), ".response")
		rpath := filepath.Join(g.logdir, rfname)
		rf, err := os.Create(rpath)
		if err != nil {
			Error.Println("Can't create file:", rfname)
			return err
		}
		defer rf.Close()
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		err = resp.Write(rf)


		if err != nil {
			Error.Println("Can't write headers to:", rfname)
		}
		resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		//f.WriteString(req.Hea)

	}
	return
}

func (g *GoScrape) AddTask(h Handler, url string, o *HttpOptions, priority int, args ...interface{}) {
	g.wg.Add(1)
	go func() {
		task := Task{h, url, o, args}
		g.ch[priority] <- &task
	}()
}

func (g *GoScrape) SetLogDir(path string) {
	g.logdir = path
}

func GetCookies(uri string, options *HttpOptions) *cookiejar.Jar {
	jar, _ := cookiejar.New(nil)
	if u, err := url.Parse(uri); err != nil {
		fmt.Errorf("URL has wrong format", uri)
		return jar
	} else {
		cookies := options.Cookies()
		if len(cookies) > 0 {
			jar.SetCookies(u, cookies)
		}
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

	taskId := g.Incr()
	Info.Println("Received task #", taskId, t.Url)

	jar := GetCookies(t.Url, t.Options)
	client := &http.Client{Jar: jar}

	req, err := g.PrepareReq(t.Url, t.Options)

	resp, err := client.Do(req)
	if err != nil {
		Error.Println("Error while downloading", t.Url, "-", err)
		return
	}
	defer resp.Body.Close()
	if err := g.StoreInfo(req, resp); err != nil {
		Error.Println("Failed task #", taskId, "-", err)
	}

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
		Info.Println("Finished task #", taskId, t.Url)
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
