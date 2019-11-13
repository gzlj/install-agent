package handle

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gzlj/install-agent/pkg/common"
	"io/ioutil"
	"os"
	"os/exec"
)

func QueryJobStatus(c *gin.Context) {
	var (
		status common.Status
		err    error
	)
	jobId := c.Query("jobId")

	status, err = queryJobStatus(jobId)
	if err != nil {
		status = common.Status{
			Code: -1,
			Err:  err.Error(),
			Id:   jobId,
		}
		c.JSON(200, status)
		return
	}
	c.JSON(200, status)
}

func ListJobStatuses(c *gin.Context) {
	var (
		statuses []common.Status
		//err    error
	)
	statuses, _ = listJobStatuses()
	//if err != nil {
	//
	//}

	c.JSON(200, statuses)
}

//func UpdateJobStatuse(c *gin.Context) {
//	common.G_OracleDb.UpdateJobStatus("5555", "haha")
//
//
//	c.JSON(200, nil)
//}

func queryJobStatus(jobId string) (status common.Status, err error) {
	var (
		statusFile string = common.STATUS_DIR + jobId + common.STATUS_FILE_SUFFIX
		bytes      []byte
	)
	if bytes, err = ioutil.ReadFile(statusFile); err != nil {
		return
	}
	json.Unmarshal(bytes, &status)

	return
}

func listJobStatuses() (statuses []common.Status, err error) {
	var (
		//statusFile string = common.STATUS_DIR + jobId + common.STATUS_FILE_SUFFIX
		bytes      []byte
		files 		[]os.FileInfo
		status common.Status
	)
	files, err =  ioutil.ReadDir(common.STATUS_DIR)
	if err != nil {
		return
	}

	for _, f :=range files {
		if bytes, err = ioutil.ReadFile(common.STATUS_DIR + f.Name()); err != nil {
			continue
		}
		json.Unmarshal(bytes, &status)
		statuses = append(statuses, status)
	}


	//if bytes, err = ioutil.ReadFile(statusFile); err != nil {
	//	return
	//}
	//json.Unmarshal(bytes, &status)
	//
	//statuses = append(statuses, status)

	return
}

func CreateStatusFile(jobId string) (err error) {
	var (
		statusFile string = common.STATUS_DIR + jobId + common.STATUS_FILE_SUFFIX
		status     common.Status
		bytes      []byte
	)

	f, err := os.OpenFile(statusFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	status = common.Status{
		Code:  200,
		Err:   "failed to start task.",
		Phase: "created",
		Id:    jobId,
	}
	if bytes, err = json.Marshal(status); err != nil {
		fmt.Println("upload job status fail.")
		return
	}
	f.WriteString(string(bytes))

	return
}

func UpdateStatusFile(status common.Status) (err error) {
	var (
		statusFile string = common.STATUS_DIR + "/" + status.Id + common.STATUS_FILE_SUFFIX
		bytes      []byte
	)

	f, err := os.OpenFile(statusFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	if bytes, err = json.Marshal(status); err != nil {
		fmt.Println("upload job status fail.")
		return
	}
	f.WriteString(string(bytes))
	return
}

func UpdateStatusInoracle(status common.Status) (err error) {
	//common.G_OracleDb.UpdateJobStatus("5555", "haha")
	common.G_OracleDb.UpdateJobStatus(status.Id, status.Phase)
	return
}


func UpdateFinalStatus(jobId string) (status common.Status) {
	var (
		logFile string = common.LOGS_DIR + jobId + common.LOG_FILE_SUFFIX
		cmdStr         = "grep failed=0 " + logFile
		err     error
	)
	cmd := exec.CommandContext(context.TODO(), "bash", "-c", cmdStr)
	//fmt.Println("grep cmd str: ", cmdStr)
	err = cmd.Run()
	if err != nil {
		//fmt.Println(err)
		status = common.Status{
			Code:  500,
			Err:   "Some error happened.Please check log file.",
			Phase: "exited",
			Id:    jobId,
		}
	} else {
		status = common.Status{
			Code:  200,
			Err:   "",
			Phase: "exited",
			Id:    jobId,
		}
	}

	UpdateStatusFile(status)
	return
}
