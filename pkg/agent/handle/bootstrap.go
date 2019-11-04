package handle

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gzlj/install-agent/pkg/agent/module"
	"github.com/gzlj/install-agent/pkg/common"
	"io"
	"os"
	"os/exec"
	"strings"
)

func BootstrapK8s(c *gin.Context) {
	var dto module.InstallConfig
	var err error
	if err = c.ShouldBindJSON(&dto); err != nil {
		c.JSON(400, "requet body is not correct.")
		return
	}

	_, ok := common.G_JobExcutingInfo[dto.JobId]
	if ok {
		c.JSON(200, common.BuildResponse(400, "Job is already Running.Please wait to complete.", nil))
		return
	}

	fmt.Println(dto)
	//bootstrap a k8s
	go bootstrapK8s(dto)

	c.JSON(200, common.BuildResponse(200, "Job is starting to run.", nil))
}

func bootstrapK8s(config module.InstallConfig) {
	// check job is running or not

	if config.IsHa {
		bootstrapHaK8s(config)
	} else {
		bootstrapSingleK8s(config)
	}
}

func bootstrapSingleK8s(config module.InstallConfig) {
	var (
		err       error
		status    common.Status
		cmdStr    string
		sshCmdStr string
		password  string = config.CommonPassword
		//t1            *time.Timer
		cmdStdoutPipe io.ReadCloser
		cmdStderrPipe io.ReadCloser
		cmd           *exec.Cmd

		kubernetesVersion              = config.KubernetesVersion
		LanRegistry                    = config.LanRegistry
		masterIp                       = config.MasterIp
		podSubnet                      = config.PodSubnet
		controlPlaneEndpoint           = config.ControlPlaneEndpoint
		kube_rpm_version               = config.KubeRpmVersion
		token                          = config.Token
		serviceSubset                          = config.ServiceSubnet
		etcdChan             chan bool = make(chan bool)
	)

	//create job status file
	common.G_JobExcutingInfo[config.JobId]="created"
	CreateStatusFile(config.JobId)

	//create ansible hosts file
	var hostsFile string = common.HOSTS_DIR + config.JobId + common.HOSTS_FILE_SUFFIX
	var logFile string = common.LOGS_DIR + config.JobId + common.LOG_FILE_SUFFIX
	cancelCtx, cancelFunc := context.WithCancel(context.TODO())

	f, err := os.OpenFile(hostsFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	defer f.Close()
	if err != nil {
		fmt.Println(err)
	}

	if ! config.IsHa {
		lines := []string{
			"[master]",
			masterIp,
			"[master:vars]",
			"lan_registry=" + LanRegistry,
			"master_ip=" + masterIp,
			"pod_subnet=" + podSubnet,
			"control_plane_endpoint=" + controlPlaneEndpoint,
			"kube_rpm_version=" + kube_rpm_version,
			"token=" + token,
			"kubernetes_version=" + kubernetesVersion,
			"service_subnet=" + serviceSubset,
		}
		f.WriteString(strings.Join(lines, "\r\n"))
	} else {

	}

	fmt.Println(hostsFile + " is set up.")

	//ensure ssh public key is sent to target host
	//ssh-public-key.sh 5743138 192.168.25.200 192.168.25.202
	sshCmdStr = common.WORKING_DIR + "ssh-public-key.sh " + password + " " + masterIp
	sshCmd := exec.CommandContext(context.TODO(), "bash", "-c", sshCmdStr)
	err = sshCmd.Run()
	if err != nil {

		fmt.Println("failed to sent ssh public key to target host: ", masterIp)
		status = common.Status{
			Code: 500,
			Err:  "failed to sent ssh public key to target host: " + masterIp,
		}
		fmt.Println(err)
		goto FINISH
	}

	// core cmd
	cmdStr = "ansible-playbook " + common.SINGLE_MASTER_BOOTSTRAP_YAML_FILE + " -i " + hostsFile
	password = config.CommonPassword

	fmt.Println(cmdStr)

	cancelFunc = cancelFunc
	cmd = exec.CommandContext(cancelCtx, "bash", "-c", cmdStr)

	//registry installation task to etcd
	fmt.Println("etcdclient.G_register --:", G_register)

	go G_register.KeepTaskOnline(config.JobId, etcdChan)

	cmdStdoutPipe, _ = cmd.StdoutPipe()
	cmdStderrPipe, _ = cmd.StderrPipe()
	go syncLog(cmdStdoutPipe, logFile)
	go syncLog(cmdStderrPipe, logFile)

	err = cmd.Start()
	fmt.Println("cmd.Start() of " + config.JobId + " is being excuted.")
	//job status change to running
	status = common.Status{
		Code:  200,
		Err:   "",
		Id:    config.JobId,
		Phase: "Running",
	}
	common.G_JobExcutingInfo[config.JobId]="Running"
	UpdateStatusFile(status)

	if err != nil {
		status = common.Status{
			Code: 500,
			Err:  "failed to start task.",
		}
		fmt.Println(err)
		UpdateStatusFile(status)
		goto FINISH

	}
	err = cmd.Wait()
	fmt.Println(config.JobId+" cmd.Wait(): ", err)

FINISH:

	UploadLog(config.JobId)
	status = UpdateFinalStatus(config.JobId)
	UploadStatus(config.JobId, status)
	etcdChan <- true
	fmt.Println("install job of " + config.JobId + "is done.")
	delete(common.G_JobExcutingInfo, config.JobId)

}

func bootstrapHaK8s(config module.InstallConfig) {

}

func syncLog(reader io.ReadCloser, file string) {
	fmt.Println("start syncLog()")
	/*	scanner := bufio.NewScanner(reader)
		for scanner.Scan() { // 命令在执行的过程中, 实时地获取其输出
			data, err := simplifiedchinese.GB18030.NewDecoder().Bytes(scanner.Bytes()) // 防止乱码
			if err != nil {
				fmt.Println("transfer error with bytes:", scanner.Bytes())
				continue
			}

			fmt.Printf("%s\n", string(data))
		}*/

	f, _ := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	defer f.Close()
	buf := make([]byte, 1024, 1024)
	for {
		strNum, err := reader.Read(buf)

		if err != nil {
			fmt.Println(err)

			//读到结尾
			if err == io.EOF || strings.Contains(err.Error(), "file already closed") {
				//err = nil
				break
			}
		}
		outputByte := buf[:strNum]
		f.WriteString(string(outputByte))
		// TODO:
		// post to master process log api
	}

	fmt.Println("stop syncLog()")
	//worker.G_done <- struct{}{}
}
