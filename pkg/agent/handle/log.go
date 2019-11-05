package handle

import (
	"github.com/gin-gonic/gin"
	"github.com/gzlj/install-agent/pkg/common"
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

	if bytes, err = ioutil.ReadFile(common.LOGS_DIR + jobId + common.LOG_FILE_SUFFIX); err != nil {
		result = err.Error()
	}
	result = string(bytes)
	return
}
