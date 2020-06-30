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

func QueryKubeconfig(c *gin.Context) {
	jobId := c.Query("jobId")
	content, _ := QueryKubeconfigFileByid(jobId)
	c.String(200, content)
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

func QueryKubeconfigFileByid(jobId string) (result string, err error) {
	if jobId == "" {
		result = ""
		return
	}
	var (
		bytes []byte
	)
	if bytes, err = ioutil.ReadFile(common.STATUS_DIR + jobId + common.KUBECONFIG_FILE_SUFFIX); err != nil {
		result = ""
		return
	}
	result = string(bytes)
	return
}
