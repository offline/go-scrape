package goscrape

import (
	"net/http"
	"strings"
)

type HttpOptions struct {
	cookies []*http.Cookie
	method  string
	headers map[string]string
}

func NewHttpOptions() *HttpOptions {
	cookies := make([]*http.Cookie, 1)
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

func (o *HttpOptions) Cookies() []*http.Cookie {
	return o.cookies
}
