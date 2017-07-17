package goscrape

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/jmcvetta/randutil"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Scraper initializer
func NewScraper() *GoScrape {
	smap := make(map[int][]*Task)
	taskChan := make(chan *Task, 1000)
	done := make(chan bool)
	return &GoScrape{
		smap:      smap,
		proxylist: []Proxy{},
		taskChan:  taskChan,
		done:      done,
	}
}

type GoScrape struct {
	wg         sync.WaitGroup
	taskChan   chan *Task
	done       chan bool
	tid        int
	mu         sync.Mutex
	smap       map[int][]*Task
	logdir     string
	proxylist  []Proxy
	packetSize int
	weights    []randutil.Choice
	min        int
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

func (g *GoScrape) SetPacketSize(size int) {
	g.packetSize = size
}

func (g *GoScrape) SetLogDir(path string) {
	g.logdir = path
}

func (g *GoScrape) Incr() int {
	g.mu.Lock()
	taskId := g.tid + 1
	g.tid = taskId
	g.mu.Unlock()
	return taskId
}

func (g *GoScrape) GenWeightedRandom(nmap map[int]int) {
	g.weights = []randutil.Choice{}
	for i, _ := range nmap {
		g.weights = append(g.weights, randutil.Choice{i, i})
	}
}

func (g *GoScrape) Calculate() map[int]int {
	var total int
	var maxSize int
	// put non-empty sources and its length in nmap
	// sum all available tasks
	nmap := make(map[int]int)
	for i, v := range g.smap {
		if len(v) > 0 {
			nmap[i] = len(v)
			total += len(v)
		}
	}

	// choose a min from existing elements or packet size
	if g.packetSize > total {
		maxSize = total
	} else {
		maxSize = g.packetSize
	}

	// fill with a new weighted choice slice
	g.GenWeightedRandom(nmap)

	picks := make(map[int]int)
	for i := 0; i < maxSize; i++ {
		choice, _ := randutil.WeightedChoice(g.weights)
		weight := choice.Item.(int)

		picks[weight]++
		nmap[weight]--
		if nmap[weight] == 0 {
			delete(nmap, weight)
			g.GenWeightedRandom(nmap)
		}

	}
	return picks
}

func (g *GoScrape) onDone() {
	var left int = 1
	for range g.done {
		left--
		if left < g.min {
			g.mu.Lock()
			tasks := []*Task{}
			picks := g.Calculate()
			for weight, counter := range picks {
				tasks = append(tasks, g.smap[weight][:counter]...)
				g.smap[weight] = g.smap[weight][counter:]
			}
			g.mu.Unlock()
			left += len(tasks)
			for _, task := range tasks {
				g.taskChan <- task
			}
		}
	}
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

func (g *GoScrape) AddTask(h Handler, url string, o *HttpOptions, weight int, args ...interface{}) {
	g.wg.Add(1)
	taskId := g.Incr()

	task := Task{
		Handler: h,
		Url:     url,
		Options: o,
		Weight:  weight,
		Id:      taskId,
		Args:    args,
	}
	g.mu.Lock()

	if _, ok := g.smap[weight]; ok {
		g.smap[weight] = append(g.smap[weight], &task)
	} else {
		g.smap[weight] = []*Task{}
		g.smap[weight] = append(g.smap[weight], &task)

	}
	g.mu.Unlock()

}

func (g *GoScrape) RequeueTask(t *Task) {
	g.wg.Add(1)
	Info.Println("Requeued #", t.Id, t.Url)
	g.mu.Lock()
	g.smap[t.Weight] = append(g.smap[t.Weight], t)
	g.mu.Unlock()

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
	for task := range g.taskChan {
		g.Scrape(task)
		g.done <- true
	}
}

func (g *GoScrape) Stats() map[string]int {
	data := map[string]int{}
	g.mu.Lock()
	for p, i := range g.smap {
		key := fmt.Sprint(p, " priority length")
		data[key] = len(i)
	}
	g.mu.Unlock()
	return data
}

func (g *GoScrape) Start(workers int) {
	fmt.Println("Started")
	if g.packetSize == 0 {
		g.packetSize = workers * 10
	}
	g.taskChan = make(chan *Task, g.packetSize+workers)

	g.min = workers
	go g.onDone()
	for i := 0; i < workers; i++ {
		go g.Worker()
	}
	g.done <- true

	g.wg.Wait()
	fmt.Println("Finished")
}
