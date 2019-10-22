package handle

import (
	"github.com/gin-gonic/gin"
	"io/ioutil"
)

func QueryJobLog(c *gin.Context) {
	jobId := c.Query("jobId")
	c.String(200, QueryJobLogByid(jobId))
}

func QueryJobLogByid(jobId string) (result string) {
	var (
		bytes []byte
		err   error
	)

	if bytes, err = ioutil.ReadFile("/tmp/" + jobId + ".log"); err != nil {
		result = err.Error()
	}
	result = string(bytes)
	return
}
