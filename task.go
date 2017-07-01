package goscrape


type Task struct {
	Handler Handler
	Url     string
	Options *HttpOptions
	Args    []interface{}
}
