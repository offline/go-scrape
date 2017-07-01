package goscrape

import (
	"net/http"
	"strings"
	"net/url"
	"log"
)

type HttpOptions struct {
	cookies []*http.Cookie
	method  string
	headers map[string]string
}

func NewHttpOptions() *HttpOptions {
	var cookies = []*http.Cookie{}
	headers := make(map[string]string)
	return &HttpOptions{
		cookies: cookies,
		method:  "GET",
		headers: headers,
	}
}

func (o *HttpOptions) SetMethod(name string) {
	o.method = name
}

func (o *HttpOptions) Method() string {
	return o.method
}

func (o *HttpOptions) AddHeader(name string, value string) {
	o.headers[strings.ToLower(name)] = value
}

func (o *HttpOptions) Headers() map[string]string {
	return o.headers
}

func (o *HttpOptions) SetCookies(cookies []*http.Cookie) {
	o.cookies = cookies
}

func (o *HttpOptions) AddCookie(name, value, uri string) {

	parsedUrl, err := url.Parse(uri)
	if err != nil {
		log.Println("Wrong url", uri)
	}

	domain := parsedUrl.Host

	if strings.HasPrefix(domain, "localhost") {
		domain = ".localhost"
	}

	cookie := &http.Cookie{
		Name:   name,
		Value:  value,
		Path:   "/",
		Domain: domain,
	}

	o.cookies = append(o.cookies, cookie)
}

func (o *HttpOptions) Cookies() []*http.Cookie {
	return o.cookies
}
