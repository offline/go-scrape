package goscrape

import "github.com/PuerkitoBio/goquery"

//import (
//	"reflect"
//)

type Handler interface {
	Success(g *GoScrape, o *HttpOptions, doc *goquery.Document, args ...interface{})
	Fail(g *GoScrape)
}

type Task struct {
	Handler Handler
	Url     string
	Options *HttpOptions
	Args    []interface{}
}

//func CreateTask(p Pager) *Task {
//	values := reflect.ValueOf(p)
//	url := values.Elem().FieldByName("Url").String()
//	options := values.Elem().FieldByName("Options").Interface().(*HttpOptions)
//	task := Task{Url: url, Options: options}
//	return &task
//}
