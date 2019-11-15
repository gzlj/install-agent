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
	"strconv"
	"strings"
)

func StartTask(c *gin.Context) {
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
	preWashConfig(&dto)
	//fmt.Println(dto)
	//bootstrap a k8s
	//go bootstrapK8s(dto)
	go startTask(dto)
	c.JSON(200, common.BuildResponse(200, "Job is starting to run.", nil))
}

func preWashConfig(config *module.InstallConfig) {
	if config.ControlPlaneEndpoint == "" {
		if config.IsHa || config.JobType == "worker-node-join" {
			config.ControlPlaneEndpoint = config.ApiserverLb + ":" + config.ApiserverLbport
		} else {
			config.ControlPlaneEndpoint = config.Master1Info.Ip + ":" + "6443"
		}
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

		//kubernetesVersion              = config.KubernetesVersion
		//lanRegistry                    = config.LanRegistry
		//masterIp                       = config.MasterIp
		//podSubnet                      = config.PodSubnet
		//controlPlaneEndpoint           = config.ControlPlaneEndpoint
		//kube_rpm_version               = config.KubeRpmVersion
		//token                          = config.Token
		//serviceSubset                          = config.ServiceSubnet
		//etcdChan             chan bool = make(chan bool)

		//targetHost string
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
		/*lines = []string{
			"[master]",
			masterIp,
			"[master:vars]",
			//"lan_registry=" + lanRegistry,
			"master_ip=" + masterIp,
			"pod_subnet=" + podSubnet,
			"control_plane_endpoint=" + controlPlaneEndpoint,
			"kube_rpm_version=" + kube_rpm_version,
			"token=" + token,
			"kubernetes_version=" + kubernetesVersion,
			"service_subnet=" + serviceSubset,
			"set_private_registry="+ strconv.FormatBool(config.SetUpPrivateRegistry),

		}
		if config.SetUpPrivateRegistry {
			lines = append(lines, "private_registry_port=" + strconv.Itoa(config.PrivateRegistryPort))
			lines = append(lines, "lan_registry=" + masterIp + ":" + strconv.Itoa(config.PrivateRegistryPort))
		} else {
			lines = append(lines, "lan_registry=" + lanRegistry)
		}*/

		lines = singleMasterHostsFileContent(config)

		//targetHost = masterIp

	case "worker-node-join":
		lines = nodeJoinHostsFileContent(config)

	case "ha-master-bootstrap":
		lines = haMasterHostsFileContent(config)
		//f.WriteString(strings.Join(lines, "\r\n"))


	case "ha-master-join":

		lines = haMasterHostsFileContent(config)
		//f.WriteString(strings.Join(lines, "\r\n"))

	default:

	}
	f.WriteString(strings.Join(lines, "\r\n"))
	// ssh cmd
	//sshCmdStr = common.WORKING_DIR + "ssh-public-key.sh " + password + " " + targetHost
	//sshCmd := exec.CommandContext(context.TODO(), "bash", "-c", sshCmdStr)
	//err = sshCmd.Run()
	//fmt.Println("filesAndCmds.SshCmdStr: ", filesAndCmds.SshCmdStr)
	//err = runAndWait(filesAndCmds.SshCmdStr)
	err = runSsh(config.CommonPassword, filesAndCmds)
	if err != nil {
		//fmt.Println("failed to sent ssh public key to target host: ", filesAndCmds.SshCmdStr)
		status = common.Status{
			Code: 500,
			Err:  "failed to sent ssh public key to target host: ",
		}
		//fmt.Println(err)
		goto FINISH
	}



	// core cmd
	cmd = exec.CommandContext(cancelCtx, "bash", "-c", filesAndCmds.CoreCmdStr)
	cmdStdoutPipe, _ = cmd.StdoutPipe()
	cmdStderrPipe, _ = cmd.StderrPipe()
	go syncLog(cmdStdoutPipe, filesAndCmds.Logfile, false)
	go syncLog(cmdStderrPipe, filesAndCmds.Logfile, false)


	fmt.Println("cmd.Start() of " + config.JobId + " is being excuted.")
	status = common.Status{
		Code:  200,
		Err:   "",
		Id:    config.JobId,
		Phase: "Running",
	}
	common.G_JobExcutingInfo[config.JobId]="Running"
	UpdateStatusFile(status)
	UpdateStatusInoracle(status)
	err = cmd.Start()

	if err != nil {
		status = common.Status{
			Code: 500,
			Err:  "failed to start task.",
		}
		fmt.Println("cmd.Start(): ",err)
		UpdateStatusFile(status)
		goto FINISH

	}
	err = cmd.Wait()
	//fmt.Println(config.JobId+" cmd.Wait(): ", err)

	//if err == nil && config.JobType == "ha-master-bootstrap" {
	//	cmd = exec.CommandContext(cancelCtx, "bash", "-c", filesAndCmds.OtherCmdStrs[0])
	//	cmdStdoutPipe, _ = cmd.StdoutPipe()
	//	cmdStderrPipe, _ = cmd.StderrPipe()
	//	go syncLog(cmdStdoutPipe, filesAndCmds.Logfile, true)
	//	go syncLog(cmdStderrPipe, filesAndCmds.Logfile, true)
	//	err = cmd.Start()
	//	err = cmd.Wait()
	//	fmt.Println(config.JobId+" other cmd.Wait(): ", err)
	//}

FINISH:
	status = UpdateFinalStatus(config.JobId, err)
	err = UpdateStatusInoracle(status)
	if err != nil {
		fmt.Println("oracle error: ", err)
	}
	fmt.Println("install job of " + config.JobId + " is done.")


}

func runSsh(password string, filesAndCmds common.TaskFilesAndCmds) (err error){
	for _, host := range filesAndCmds.TargetHosts {
		cmdstr := common.WORKING_DIR + "ssh-public-key.sh " + "'" + password + "' " + host
		//fmt.Println("ssh cmdstr: ", cmdstr)
		err = runAndWait(cmdstr)
		if err != nil {
			return
		}
	}
	return
}

func singleMasterHostsFileContent(config module.InstallConfig) (lines []string){
	lines = []string{
		"[master]",
		config.Master1Info.Ip + " ansible_ssh_port=" + strconv.Itoa(config.Master1Info.SshPort) + " hostname=" + config.Master1Info.Hostname,
		"[all:vars]",
		"master_ip=" + config.Master1Info.Ip,
		"pod_subnet=" + config.PodSubnet,
		"control_plane_endpoint=" + config.ControlPlaneEndpoint,
		"token=" + config.Token,
		"kubernetes_version=" + config.KubernetesVersion,
		"service_subnet=" + config.ServiceSubnet,
		"set_private_registry="+ strconv.FormatBool(config.SetUpPrivateRegistry),
	}
	if config.SetUpPrivateRegistry {
		lines = append(lines, "private_registry_port=" + strconv.Itoa(config.PrivateRegistryPort))
		lines = append(lines, "lan_registry=" + config.Master1Info.Ip + ":" + strconv.Itoa(config.PrivateRegistryPort))
	} else {
		lines = append(lines, "lan_registry=" + config.LanRegistry)
	}
	if len(config.NodesToJoin) > 0 {
		lines = append(lines, "[node]")
		for _, node := range config.NodesToJoin {
			s := node.Ip + " ansible_ssh_port=" + strconv.Itoa(node.SshPort) + " hostname=" + node.Hostname
			lines = append(lines, s)
		}
	}

	return

}

func haMasterHostsFileContent(config module.InstallConfig) (lines []string){

	master1Str := config.Master1Info.Ip + " ansible_ssh_port=" + strconv.Itoa(config.Master1Info.SshPort) + " hostname=" + config.Master1Info.Hostname
	master2Str := config.Master2Info.Ip + " ansible_ssh_port=" + strconv.Itoa(config.Master2Info.SshPort) + " hostname=" + config.Master2Info.Hostname
	master3Str := config.Master3Info.Ip + " ansible_ssh_port=" + strconv.Itoa(config.Master3Info.SshPort) + " hostname=" + config.Master3Info.Hostname


	lines = []string{

		"[master]",
		master1Str,
		master2Str,
		master3Str,

		"[master1]",
		master1Str,
		"[master1:vars]",
		"master_role=master1",
		"set_private_registry="+ strconv.FormatBool(config.SetUpPrivateRegistry),

		//"[master2]",
		//master2Str,
		//"[master2:vars]",
		//"master_role=master2",
		//
		//"[master3]",
		//master3Str,
		//"[master3:vars]",
		//"master_role=master3",
		"[master23]",
		master2Str + " master_role=master2",
		master3Str + " master_role=master3",

		"[all:vars]",
		"kubernetes_version=" + config.KubernetesVersion,
		"kube_rpm_version=" + config.KubeRpmVersion,
		"pod_subnet=" + config.PodSubnet,
		"control_plane_endpoint=" + config.ControlPlaneEndpoint,
		"apiserver_lb=" + config.ApiserverLb,
		"apiserver_lbport=" + config.ApiserverLbport,
		"token=" + config.Token,
		"master1_ip=" + config.Master1Info.Ip,
		"master2_ip=" + config.Master2Info.Ip,
		"master3_ip=" + config.Master3Info.Ip,
		"master1_password=" + config.CommonPassword,
	}

	if config.SetUpPrivateRegistry {
		lines = append(lines, "private_registry_port=" + strconv.Itoa(config.PrivateRegistryPort))
		lines = append(lines, "lan_registry=" + config.Master1Info.Ip + ":" + strconv.Itoa(config.PrivateRegistryPort))
	} else {
		lines = append(lines, "lan_registry=" + config.LanRegistry)
	}

	if len(config.NodesToJoin) > 0 {
		lines = append(lines, "[node]")
		for _, node := range config.NodesToJoin {
			s := node.Ip + " ansible_ssh_port=" + strconv.Itoa(node.SshPort) + " hostname=" + node.Hostname
			lines = append(lines, s)
		}
	}
	return
}

func nodeJoinHostsFileContent(config module.InstallConfig) (lines []string){
	lines = []string{
		"[all:vars]",
		"lan_registry=" + config.LanRegistry,
		"kubernetes_version=" + config.KubernetesVersion,
		//"kube_rpm_version=" + config.KubeRpmVersion,
		"control_plane_endpoint=" + config.ControlPlaneEndpoint,
		//"token=" + config.Token,
	}
	if config.Token != "" {
		lines = append(lines, "token=" + config.Token)
	}
	if len(config.NodesToJoin) > 0 {
		lines = append(lines, "[node]")
		for _, node := range config.NodesToJoin {
			s := node.Ip + " ansible_ssh_port=" + strconv.Itoa(node.SshPort) + " hostname=" + node.Hostname
			lines = append(lines, s)
		}
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
	var targeHosts []string
	if config.Master1Info.Ip != "" {
		targeHosts = append(targeHosts, config.Master1Info.Ip)
	}
	if config.Master2Info.Ip != "" {
		targeHosts = append(targeHosts, config.Master2Info.Ip)
	}
	if config.Master3Info.Ip != "" {
		targeHosts = append(targeHosts, config.Master3Info.Ip)
	}

	//fmt.Println("config module.InstallConfig", config)
	//fmt.Println("var sshHostsStr: ", sshHostsStr)
	switch config.JobType {
	case "single-master-bootstrap":
		filesAndCmds.CoreCmdStr = "ansible-playbook " + common.SINGLE_MASTER_BOOTSTRAP_YAML_FILE + " -i " + filesAndCmds.HostsFile
		if len(config.NodesToJoin) > 0 {
			for _, node := range config.NodesToJoin {
				targeHosts = append(targeHosts, node.Ip)
			}
		}

	case "worker-node-join":
		filesAndCmds.CoreCmdStr = "ansible-playbook " + common.WOKER_NODE_JOIN_YAML_FILE + " -i " + filesAndCmds.HostsFile
		//nodes :=""
		for _, node := range config.NodesToJoin  {
			targeHosts = append(targeHosts, node.Ip)
		}

	case "ha-master-bootstrap":
		filesAndCmds.CoreCmdStr = "ansible-playbook " + common.HA_MASTER_BOOTSTRAP_YAML_FILE + " -i " + filesAndCmds.HostsFile
		if len(config.NodesToJoin) > 0 {
			for _, node := range config.NodesToJoin {
				targeHosts = append(targeHosts, node.Ip)
			}
		}

	case "ha-master-join":
		filesAndCmds.CoreCmdStr = "ansible-playbook " + common.HA_MASTER_JOIN_YAML_FILE + " -i " + filesAndCmds.HostsFile
	default:
		err = errors.New("Job type must be one of single-master-bootstrap, worker-node-join, ha-master-bootstrap, ha-master-join.")
	}
	filesAndCmds.TargetHosts = targeHosts
	return
}

/*
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
		//etcdChan             chan bool = make(chan bool)


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
	//fmt.Println("etcdclient.G_register --:", G_register)

	//go G_register.KeepTaskOnline(config.JobId, etcdChan)

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
	//UploadLog(config.JobId)
	status = UpdateFinalStatus(config.JobId)
	//UploadStatus(config.JobId, status)
	//etcdChan <- true
	fmt.Println("install job of " + config.JobId + "is done.")
}

func bootstrapHaK8s(config module.InstallConfig) {

}*/

func syncLog(reader io.ReadCloser, file string, append bool) {
	//fmt.Println("start syncLog()")
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
			//读到结尾
			if err == io.EOF || strings.Contains(err.Error(), "file already closed") {
				//err = nil
				break
			}
		}
		outputByte := buf[:strNum]
		f.WriteString(string(outputByte))
	}
}
