package worker

type Status struct {
	Err string `json:"err"`
	Code int64 `json:"code"`
}

var (
	G_done chan struct{} = make(chan struct{})
	G_undone chan string = make(chan string)
)
