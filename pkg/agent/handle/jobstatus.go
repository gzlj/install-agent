package handle

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gzlj/install-agent/pkg/agent/module"
	"github.com/gzlj/install-agent/pkg/agent/utils/jobutil"
	"github.com/gzlj/install-agent/pkg/common"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

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
	if status.Code == 0 {
		memStatus, ok := common.G_JobStatus[jobId]
		if !ok {
			status = common.Status{
				Code: 500,
				Err:  "Cannot query job status.",
				Id:   jobId,
			}
			c.JSON(200, status)
			return
		}
		status = common.Status{
			Code: memStatus.Code,
			Err:  memStatus.Err,
			Id:   memStatus.Id,
			JobType: memStatus.JobType,
			Progress: memStatus.Progress,
			Phase: memStatus.Phase,
		}
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
		if len(bytes) == 0 {
			time.Sleep(time.Second * 1)
			if bytes, err = ioutil.ReadFile(statusFile); err != nil {
				return
			}
		}
	}
	if len(bytes) == 0 {
		time.Sleep(time.Second * 1)
		if bytes, err = ioutil.ReadFile(statusFile); err != nil {
			return
		}
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



	//get job process
	process := jobutil.GetJobProcess(status.Id)

	if status.Progress < 100 {
		switch status.JobType {
		case "single-master-bootstrap":
			status.Progress = process * 100 / common.SINGEL_MASTER_JOB_TASK_COUNT
		case "worker-node-join":
			status.Progress = process * 100 / common.WORKER_NODE_JOIN_JOB_TASK_COUNT
		case "ha-master-bootstrap":
			status.Progress = process * 100 / common.HA_MASTER_JOB_TASK_COUNT
		case "ha-master-join":
		default:
		}
	}
	if status.Progress > 100 {
		status.Progress = 100
		if status.Phase == "Running" {
			status.Progress = 99
		}
	}

	if bytes, err = json.Marshal(status); err != nil {
		fmt.Println("upload job status fail.")
		return
	}
	f, err := os.OpenFile(statusFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	//f, err := os.OpenFile(statusFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return
	}
	defer f.Close()
	//f2, err := os.OpenFile(statusFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	f.WriteString(string(bytes))
	//f2.Close()
	return
}

/*func UpdateStatusInoracle(status common.Status) (err error) {
	//common.G_OracleDb.UpdateJobStatus("5555", "haha")
	err = common.G_OracleDb.UpdateJobStatus(status.Id, status.Phase)
	return
}*/


func UpdateFinalStatus(config *module.InstallConfig, err error) (status common.Status) {
	/*var (
		logFile string = common.LOGS_DIR + jobId + common.LOG_FILE_SUFFIX
		cmdStr         = "grep failed=0 " + logFile
		err     error
	)
	cmd := exec.CommandContext(context.TODO(), "bash", "-c", cmdStr)
	//fmt.Println("grep cmd str: ", cmdStr)


	err = cmd.Run()*/
	if err != nil {
		//fmt.Println(err)

		switch err.(type) {
		case *exec.ExitError:
			status = common.Status{
				Code:  500,
				Err:   "Job is killed by user or Job is exited with non-zero code.",
				Phase: "failed",
				Id:    config.JobId,
				JobType: config.JobType,
				Progress: 99,
			}
		default:
			status = common.Status{
				Code:  500,
				Err:   "Some error happened.Please check log file.",
				Phase: "failed",
				Id:    config.JobId,
				JobType: config.JobType,
				Progress: 99,
			}
		}
	} else {
		status = common.Status{
			Code:  200,
			Err:   "",
			Phase: "exited",
			Id:    config.JobId,
			JobType: config.JobType,
			Progress: 100,
		}
	}
	delete(common.G_JobStatus, config.JobId)
	UpdateStatusFile(status)
	return
}



