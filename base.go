package goscrape

import (
	"bytes"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
	"log"
	"bufio"
	"strings"
	"math/rand"
	"errors"
)

const (
	Low    int = 0
	Medium int = 1
	High   int = 2
)

// Scraper initializer
func NewScraper() *GoScrape {
	ch := []chan *Task{make(chan *Task), make(chan *Task), make(chan *Task)}
	return &GoScrape{ch: ch, proxylist: []Proxy{}}
}

type GoScrape struct {
	wg     sync.WaitGroup
	ch     []chan *Task
	tid    int
	mu     sync.Mutex
	logdir string
	proxylist []Proxy
}

func (g *GoScrape) SetProxyFile(path string, proxytype string) error {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal("Proxyfile invalid")
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		host := parts[0]
		port, _ := strconv.Atoi(parts[1])
		proxy := Proxy{Scheme: proxytype, Host: host, Port: port}
		g.proxylist = append(g.proxylist, proxy)
	}

	return err
}

func (g *GoScrape) Incr() int {
	g.mu.Lock()
	taskId := g.tid + 1
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
	taskId := g.Incr()
	go func() {
		task := Task{Handler: h, Url: url, Options: o, Priority: priority, Id: taskId, Args: args}
		g.ch[priority] <- &task
	}()
}

func (g *GoScrape) RequeueTask(t *Task) {
	g.wg.Add(1)
	Info.Println("Requeued #", t.Id, t.Url)
	go func() {
		g.ch[t.Priority] <- t
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

func (g *GoScrape) GetRandomProxy() (Proxy, error) {
	if len(g.proxylist) == 0 {
		return Proxy{}, errors.New("no proxy")
	}
	rand.Seed(time.Now().UnixNano())
	n := rand.Int() % len(g.proxylist)
	return g.proxylist[n], nil
}

func (g *GoScrape) SetupClient(uri string, o *HttpOptions) (*http.Client, *http.Request, error) {
	var buf bytes.Buffer
	req, err := http.NewRequest(o.Method(), uri, &buf)
	if err != nil {
		fmt.Println("Error creating request", uri, "-", err)
		return nil, nil, err
	}
	for key, value := range o.Headers() {
		req.Header.Add(key, value)
	}

	jar := GetCookies(uri, o)

	proxy, err := g.GetRandomProxy()
	if err != nil {
		client := &http.Client{Jar: jar}
		return client, req, nil
	}
	tr := &http.Transport{
		Proxy: func(r *http.Request) (*url.URL, error) {
			rawUrl := fmt.Sprintf("%s://%s:%d", proxy.Scheme, proxy.Host, proxy.Port)
			proxyUrl, err := url.Parse(rawUrl)

			return proxyUrl, err
		},
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: true,
	}
	client := &http.Client{Transport: tr, Jar: jar}
	return client, req, nil

}

func (g *GoScrape) Scrape(t *Task) {
	defer g.wg.Done()

	Info.Println("Received task #", t.Id, t.Url)

	client, req, err := g.SetupClient(t.Url, t.Options)

	resp, err := client.Do(req)
	if err != nil {
		Error.Println("Error while downloading", t.Url, "-", err)
		if t.Retry < 5 {
			t.Retry++
			g.RequeueTask(t)
		}
		return
	}
	defer resp.Body.Close()
	if err := g.StoreInfo(req, resp); err != nil {
		Error.Println("Failed task #", t.Id, "-", err)
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
		Info.Println("Finished task #", t.Id, t.Url)
	} else {
		t.Handler.Fail(g, t.Url)
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
