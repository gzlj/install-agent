package jobutil

import (
	"bufio"
	"encoding/json"
	"github.com/gzlj/install-agent/pkg/common"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

func GetJobProcess(jobId string) int{
	var (
		f *os.File
		err error
	)
	fileLocation := common.LOGS_DIR + jobId + common.LOG_FILE_SUFFIX
	f, err = os.OpenFile(fileLocation, os.O_RDONLY, 0666)
	if err != nil {
		return 0
	}
	return CountStr(f, "TASK ")
}

func CountStr(file *os.File, str string) (count int){
	input := bufio.NewScanner(file)
	for input.Scan() {
		line := input.Text()
		if strings.HasPrefix(line, str) {
			count++
		}
	}
	return
}

func SyncJobProcessLoop(jobId string) {

	timer1 := time.NewTimer(15 * time.Second)
	timer2 := time.NewTimer(30 * time.Second)
	for {
		select {
		case <-timer1.C:
			if JobIsExited(jobId) {
				return
			}
			updateJobProcessInMemory(jobId)
			timer1.Reset(15 * time.Second)
		case <-timer2.C:
			//get status from memory
			status, ok := common.G_JobStatus[jobId]
			if !ok {
				return
			}
			UpdateJobProcess(status)
			timer2.Reset(20 * time.Second)
		}
	}

}

func JobIsExited(jobId string) bool {
	status, err := QueryJobStatus(jobId)
	if err != nil {
		return false
	}
	if status.Phase == "exited" {
		return true
	}
	return false
}

func QueryJobStatus(jobId string) (status common.Status, err error) {
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

func UpdateJobProcess(status *common.Status) (err error) {

	if status.Phase == "Running" && status.Progress >= 100 {
		return
	}

	var (
		statusFile string = common.STATUS_DIR + "/" + status.Id + common.STATUS_FILE_SUFFIX
		bytes      []byte
	)
	var f *os.File
	f, err = os.OpenFile(statusFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	//f, err := os.OpenFile(statusFile, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return
	}
	defer f.Close()
	if bytes, err = json.Marshal(status); err != nil {
		log.Println("update job status fail.")
		return
	}
	f.WriteString(string(bytes))
	/*bytes, err = ioutil.ReadAll(f)
	if err != nil {
		return
	}
	//var status common.Status
	err = json.Unmarshal(bytes, &status)
	if err != nil {
		return
	}

	//get job process
	process := GetJobProcess(status.Id)

	switch status.JobType {
	case "single-master-bootstrap":
		status.Progress = process * 100 / common.SINGEL_MASTER_JOB_TASK_COUNT
	case "worker-node-join":
		status.Progress = process * 100 / common.WORKER_NODE_JOIN_JOB_TASK_COUNT
	case "ha-master-bootstrap":
		status.Progress = process * 100 / common.HA_MASTER_JOB_TASK_COUNT
	case "ha-master-join":
	default:
		return
	}*/

	/*if status.JobType == "single-master-bootstrap" {
	}

	status.Progress = process * 100 / common.SINGEL_MASTER_JOB_TASK_COUNT*/
/*
	if bytes, err = json.Marshal(status); err != nil {
		log.Println("update job status fail.")
		return
	}
	f2, err := os.OpenFile(statusFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return
	}
	f2.Close()
	f2.WriteString(string(bytes))
*/
	return
}

func updateJobProcessInMemory(jobId string) (err error) {

	status, ok := common.G_JobStatus[jobId]
	if !ok {
		return
	}
	//update process
	process := GetJobProcess(status.Id)

	switch status.JobType {
	case "single-master-bootstrap":
		status.Progress = process * 100 / common.SINGEL_MASTER_JOB_TASK_COUNT
	case "worker-node-join":
		status.Progress = process * 100 / common.WORKER_NODE_JOIN_JOB_TASK_COUNT
	case "ha-master-bootstrap":
		status.Progress = process * 100 / common.HA_MASTER_JOB_TASK_COUNT
	case "ha-master-join":
	case "destroy":
		status.Progress = process * 100 / common.WORKER_NODE_DESTROY_JOB_TASK_COUNT
	default:
	}
	if status.Progress > 100 {
		status.Progress = 99
	}
	return
}