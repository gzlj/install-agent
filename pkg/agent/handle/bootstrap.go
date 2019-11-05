package handle

import (
	"context"
	"errors"
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

	//fmt.Println(dto)
	//bootstrap a k8s
	//go bootstrapK8s(dto)
	go startTask(dto)
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

func startTask(config module.InstallConfig) {
	var (
		filesAndCmds common.TaskFilesAndCmds
		//hostsFile string
		//logFile string
		//coreCmdStr string
		err       error
		status    common.Status
		//sshCmdStr string
		//password  string = config.CommonPassword
		//t1            *time.Timer
		cmdStdoutPipe io.ReadCloser
		cmdStderrPipe io.ReadCloser
		cmd           *exec.Cmd

		kubernetesVersion              = config.KubernetesVersion
		lanRegistry                    = config.LanRegistry
		masterIp                       = config.MasterIp
		podSubnet                      = config.PodSubnet
		controlPlaneEndpoint           = config.ControlPlaneEndpoint
		kube_rpm_version               = config.KubeRpmVersion
		token                          = config.Token
		serviceSubset                          = config.ServiceSubnet
		//etcdChan             chan bool = make(chan bool)

		targetHost string
		lines []string
		f *os.File
		cancelCtx, _ = context.WithCancel(context.TODO())
	)


	//hostsFile = filesAndCmds.HostsFile
	//logFile = filesAndCmds.Logfile
	common.G_JobExcutingInfo[config.JobId]="created"
	defer delete(common.G_JobExcutingInfo, config.JobId)
	CreateStatusFile(config.JobId)
	filesAndCmds, err = constructFilesAndCmd(config)
	if err != nil {
		goto FINISH
	}


	f, err = os.OpenFile(filesAndCmds.HostsFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	defer f.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	// ensure hostsfile and logfile and target host
	switch config.JobType {
	case "single-master-bootstrap":
		lines := []string{
			"[master]",
			masterIp,
			"[master:vars]",
			"lan_registry=" + lanRegistry,
			"master_ip=" + masterIp,
			"pod_subnet=" + podSubnet,
			"control_plane_endpoint=" + controlPlaneEndpoint,
			"kube_rpm_version=" + kube_rpm_version,
			"token=" + token,
			"kubernetes_version=" + kubernetesVersion,
			"service_subnet=" + serviceSubset,
		}
		f.WriteString(strings.Join(lines, "\r\n"))
		targetHost = masterIp

	case "worker-node-join":

	case "ha-master-bootstrap":
		lines = haMasterHostsFileContent(config)
		f.WriteString(strings.Join(lines, "\r\n"))


	case "ha-master-join":

		lines = haMasterHostsFileContent(config)
		f.WriteString(strings.Join(lines, "\r\n"))

	default:

	}

	// ssh cmd
	//sshCmdStr = common.WORKING_DIR + "ssh-public-key.sh " + password + " " + targetHost
	//sshCmd := exec.CommandContext(context.TODO(), "bash", "-c", sshCmdStr)
	//err = sshCmd.Run()
	err = runAndWait(filesAndCmds.SshCmdStr)
	if err != nil {
		fmt.Println("failed to sent ssh public key to target host: ", targetHost)
		status = common.Status{
			Code: 500,
			Err:  "failed to sent ssh public key to target host: " + targetHost,
		}
		fmt.Println(err)
		goto FINISH
	}



	// core cmd
	//cancelFunc = cancelFunc
	cmd = exec.CommandContext(cancelCtx, "bash", "-c", filesAndCmds.CoreCmdStr)

	//registry installation task to etcd
	fmt.Println("etcdclient.G_register --:", G_register)
	cmdStdoutPipe, _ = cmd.StdoutPipe()
	cmdStderrPipe, _ = cmd.StderrPipe()
	go syncLog(cmdStdoutPipe, filesAndCmds.Logfile, false)
	go syncLog(cmdStderrPipe, filesAndCmds.Logfile, false)

	err = cmd.Start()
	fmt.Println("cmd.Start() of " + config.JobId + " is being excuted.")
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

	if err == nil && config.JobType == "ha-master-bootstrap" {
		cmd = exec.CommandContext(cancelCtx, "bash", "-c", filesAndCmds.OtherCmdStrs[0])
		cmdStdoutPipe, _ = cmd.StdoutPipe()
		cmdStderrPipe, _ = cmd.StderrPipe()
		go syncLog(cmdStdoutPipe, filesAndCmds.Logfile, true)
		go syncLog(cmdStderrPipe, filesAndCmds.Logfile, true)
		err = cmd.Start()
		err = cmd.Wait()
		fmt.Println(config.JobId+" other cmd.Wait(): ", err)
	}

FINISH:

	UploadLog(config.JobId)
	status = UpdateFinalStatus(config.JobId)
	UploadStatus(config.JobId, status)
	//etcdChan <- true
	fmt.Println("install job of " + config.JobId + "is done.")


}

func haMasterHostsFileContent(config module.InstallConfig) (lines []string){
	lines = []string{
		"[master1]",
		config.Master1Ip,

		"[master2]",
		config.Master2Ip,
		"[master2:vars]",
		"master_role=master2",

		"[master3]",
		config.Master3Ip,
		"[master3:vars]",
		"master_role=master3",

		"[all:vars]",
		"lan_registry=" + config.LanRegistry,
		"kubernetes_version=" + config.KubernetesVersion,
		"kube_rpm_version=" + config.KubeRpmVersion,
		"pod_subnet=" + config.PodSubnet,
		"control_plane_endpoint=" + config.ControlPlaneEndpoint,
		"apiserver_lb=" + config.ApiserverLb,
		"apiserver_lbport=" + config.ApiserverLbport,
		"token=" + config.Token,
		"master1_ip=" + config.Master1Ip,
		"master2_ip=" + config.Master2Ip,
		"master3_ip=" + config.Master3Ip,

	}
	return
}

func runAndWait(cmdStr string) (err error) {
	cmd := exec.CommandContext(context.TODO(), "bash", "-c", cmdStr)
	err = cmd.Run()
	return
}

func constructFilesAndCmd(config module.InstallConfig) (filesAndCmds common.TaskFilesAndCmds, err error){
	filesAndCmds.HostsFile = common.HOSTS_DIR + config.JobId + common.HOSTS_FILE_SUFFIX
	filesAndCmds.Logfile = common.LOGS_DIR + config.JobId + common.LOG_FILE_SUFFIX
	switch config.JobType {
	case "single-master-bootstrap":
		filesAndCmds.CoreCmdStr = "ansible-playbook " + common.SINGLE_MASTER_BOOTSTRAP_YAML_FILE + " -i " + filesAndCmds.HostsFile
		filesAndCmds.SshCmdStr = common.WORKING_DIR + "ssh-public-key.sh " + config.CommonPassword + " " + config.MasterIp
	case "worker-node-join":
		filesAndCmds.CoreCmdStr = "ansible-playbook " + common.WOKER_NODE_JOIN_YAML_FILE + " -i " + filesAndCmds.HostsFile
		nodes :=""
		for _, n := range config.NodesToJoin  {
			nodes = nodes + n + " "
		}
		filesAndCmds.SshCmdStr = common.WORKING_DIR + "ssh-public-key.sh " + config.CommonPassword + " " + nodes

	case "ha-master-bootstrap":
		filesAndCmds.CoreCmdStr = "ansible-playbook " + common.HA_MASTER_BOOTSTRAP_YAML_FILE + " -i " + filesAndCmds.HostsFile
		filesAndCmds.SshCmdStr = common.WORKING_DIR + "ssh-public-key.sh " + config.CommonPassword + " " + config.Master1Ip + " " + config.Master2Ip + " " + config.Master3Ip
		filesAndCmds.OtherCmdStrs = []string{
			"ansible-playbook " + common.HA_MASTER_JOIN_YAML_FILE + " -i " + filesAndCmds.HostsFile,
		}

	case "ha-master-join":
		filesAndCmds.CoreCmdStr = "ansible-playbook " + common.HA_MASTER_JOIN_YAML_FILE + " -i " + filesAndCmds.HostsFile
		filesAndCmds.SshCmdStr = common.WORKING_DIR + "ssh-public-key.sh " + config.CommonPassword + " " + config.Master2Ip + " " + config.Master3Ip
	default:
		err = errors.New("Job type must be one of single-master-bootstrap, worker-node-join, ha-master-bootstrap, ha-master-join.")
	}
	return
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
	defer delete(common.G_JobExcutingInfo, config.JobId)
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
	go syncLog(cmdStdoutPipe, logFile, false)
	go syncLog(cmdStderrPipe, logFile, false)

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
	fmt.Println("FINISH: code block ")
	UploadLog(config.JobId)
	status = UpdateFinalStatus(config.JobId)
	UploadStatus(config.JobId, status)
	//etcdChan <- true
	fmt.Println("install job of " + config.JobId + "is done.")
}

func bootstrapHaK8s(config module.InstallConfig) {

}

func syncLog(reader io.ReadCloser, file string, append bool) {
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
	var (
	f *os.File
	)
	if append {
		f, _ = os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	} else {
		f, _ = os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	}
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
