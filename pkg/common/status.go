package common

import "context"

var (

	G_JobExcutingInfo map[string]string = make(map[string]string, 10)
	G_JobCancleFuncs map[string]context.CancelFunc = make(map[string]context.CancelFunc, 10)
	G_JobStatus map[string]*Status = make(map[string]*Status, 10)
)
