package goscrape

type Task struct {
	Id       int
	Handler  Handler
	Url      string
	Options  *HttpOptions
	Retry    int8
	Priority int
	Args     []interface{}
}
