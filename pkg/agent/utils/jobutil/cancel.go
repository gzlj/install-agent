package jobutil

import (
	"fmt"
	"github.com/gzlj/install-agent/pkg/common"
)

func JobCanBeCancled(jobId string) bool {
	_, ok := common.G_JobCancleFuncs[jobId]
	if ok {
		return true
	}
	return false
}

func CancelJobById(id string) (err error){
	f, ok := common.G_JobCancleFuncs[id]
	if !ok {
		err = fmt.Errorf("Job for %v does not exist.", id)
		return
	}
	f()
	return
}