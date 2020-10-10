package handle

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gzlj/install-agent/pkg/agent/module"
	"github.com/gzlj/install-agent/pkg/agent/utils/jobutil"
	"github.com/gzlj/install-agent/pkg/common"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func StartTask(c *gin.Context) {
	var dto module.InstallConfig
	var err error
	if err = c.ShouldBindJSON(&dto); err != nil {
		c.JSON(400, "requet body is not correct.")
		return
	}

	if len(dto.JobId) == 0 {
		c.JSON(200, common.BuildResponse(400, "必须指定jobId.", nil))
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
	time.Sleep(time.Second)
	c.JSON(200, common.BuildResponse(200, "Job is starting to run.", nil))
}

func preWashConfig(config *module.InstallConfig) {
	if len(config.ApiserverLbport) == 0 {
		config.ApiserverLbport = "16443"
	}
	if config.ControlPlaneEndpoint == "" {
		if config.JobType == "ha-master-bootstrap" || config.JobType == "worker-node-join" {
			config.ControlPlaneEndpoint = config.ApiserverLb + ":" + config.ApiserverLbport
		} else {
			config.ControlPlaneEndpoint = config.Master1Info.Ip + ":" + "6443"
		}
	}
	if config.PodSubnet == "" {
		config.PodSubnet = "10.244.0.0/16"
	}
	if config.ServiceSubnet == "" {
		config.ServiceSubnet = "10.96.0.0/12"
	}
	if config.Token == "" {
		config.Token = "b99a00.a144ef80536d4344"
	}
}

func startTask(config module.InstallConfig) {
	var (
		filesAndCmds common.TaskFilesAndCmds
		//hostsFile string
		//logFile string
		//coreCmdStr string
		err    error
		status common.Status
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
		lines                 []string
		f                     *os.File
		cancelCtx, cancelFunc = context.WithCancel(context.TODO())
	)

	//hostsFile = filesAndCmds.HostsFile
	//logFile = filesAndCmds.Logfile
	common.G_JobExcutingInfo[config.JobId] = "created"
	common.G_JobCancleFuncs[config.JobId] = cancelFunc

	defer delete(common.G_JobExcutingInfo, config.JobId)
	defer delete(common.G_JobCancleFuncs, config.JobId)
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
	case "destroy":
		lines = nodeDestroyHostsFileContent(config)
	default:

	}
	f.WriteString(strings.Join(lines, "\r\n"))
	// ssh cmd
	//sshCmdStr = common.WORKING_DIR + "ssh-public-key.sh " + password + " " + targetHost
	//sshCmd := exec.CommandContext(context.TODO(), "bash", "-c", sshCmdStr)
	//err = sshCmd.Run()
	//fmt.Println("filesAndCmds.SshCmdStr: ", filesAndCmds.SshCmdStr)
	//err = runAndWait(filesAndCmds.SshCmdStr)
	//runSshForSingleHost
	if config.JobType == "ha-master-bootstrap" {
		err = runSshForSingleHost(config.Master1Info.Ip, config.Master1Info.Password)
		if err != nil {
			status = common.Status{
				Code:    500,
				Err:     "failed to sent ssh public key to target host: ",
				JobType: config.JobType,
			}
			goto FINISH
		}
		err = runSshForSingleHost(config.Master2Info.Ip, config.Master2Info.Password)
		if err != nil {
			status = common.Status{
				Code:    500,
				Err:     "failed to sent ssh public key to target host: ",
				JobType: config.JobType,
			}
			goto FINISH
		}
		err = runSshForSingleHost(config.Master3Info.Ip, config.Master3Info.Password)
		if err != nil {
			status = common.Status{
				Code:    500,
				Err:     "failed to sent ssh public key to target host: ",
				JobType: config.JobType,
			}
			goto FINISH
		}
	}

	err = runSsh(config.CommonPassword, filesAndCmds)
	if err != nil {
		//fmt.Println("failed to sent ssh public key to target host: ", filesAndCmds.SshCmdStr)
		status = common.Status{
			Code:    500,
			Err:     "failed to sent ssh public key to target host: ",
			JobType: config.JobType,
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

	//fmt.Println("cmd.Start() of " + config.JobId + " is being excuted.")
	log.Println("job for cluster " + config.JobId + " is being excuted.")
	status = common.Status{
		Code:    200,
		Err:     "",
		Id:      config.JobId,
		Phase:   "Running",
		JobType: config.JobType,
	}
	common.G_JobExcutingInfo[config.JobId] = "Running"
	common.G_JobStatus[config.JobId] = &status
	UpdateStatusFile(status)
	//UpdateStatusInoracle(status)
	err = cmd.Start()

	if err != nil {
		status = common.Status{
			Code:    500,
			Err:     "failed to start task.",
			JobType: config.JobType,
		}
		fmt.Println("cmd.Start(): ", err)
		common.G_JobStatus[config.JobId] = &status
		UpdateStatusFile(status)
		goto FINISH

	}

	//new a goroutine to get process and write it to status file
	go jobutil.SyncJobProcessLoop(status.Id)

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

	if err == nil {
		if config.JobType == "single-master-bootstrap" || config.JobType == "ha-master-bootstrap" {
			err = runAndWait(filesAndCmds.KubeconfigCmdStr)
			if err != nil {
				log.Println("KubeconfigCmdStr: ", filesAndCmds.KubeconfigCmdStr)
				log.Println("cannot get kubeconfig file for cluster "+config.JobId+": ", err.Error())
				return
			}
		}
	}
	status = UpdateFinalStatus(&config, err)
	/*err = UpdateStatusInoracle(status)
	if err != nil {
		fmt.Println("oracle error: ", err)
	}*/
	//fmt.Println("install job of " + config.JobId + " is done.")
	log.Println("job for cluster " + config.JobId + "  is done.")

}

func runSsh(password string, filesAndCmds common.TaskFilesAndCmds) (err error) {
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

func runSshForSingleHost(host, password string) (err error) {
	cmdstr := common.WORKING_DIR + "ssh-public-key.sh " + "'" + password + "' " + host
	err = runAndWait(cmdstr)
	if err != nil {
		return
	}
	return
}

func singleMasterHostsFileContent(config module.InstallConfig) (lines []string) {
	lines = []string{
		"[master]",
		config.Master1Info.Ip + " ansible_ssh_port=" + strconv.Itoa(config.Master1Info.SshPort) + " hostname=" + config.Master1Info.Hostname,
		"[all:vars]",
		"master_ip=" + config.Master1Info.Ip,
		"pod_subnet=" + config.PodSubnet,
		"service_subnet=" + config.ServiceSubnet,
		"control_plane_endpoint=" + config.ControlPlaneEndpoint,
		"token=" + config.Token,
		"kubernetes_version=" + config.KubernetesVersion,
		"service_subnet=" + config.ServiceSubnet,
		"set_private_registry=" + strconv.FormatBool(config.SetUpPrivateRegistry),
	}
	if config.SetUpPrivateRegistry {
		lines = append(lines, "private_registry_port="+strconv.Itoa(config.PrivateRegistryPort))
		lines = append(lines, "lan_registry=" + config.Master1Info.Ip+":"+strconv.Itoa(config.PrivateRegistryPort))
	} else {
		lines = append(lines, "lan_registry=" + config.LanRegistry)
	}

	if len(config.ClusterName) > 0 {
		lines = append(lines, "cluster_name=" + config.ClusterName)
	}
	if len(config.DockerHome) > 0 {
		lines = append(lines, "docker_home" + config.DockerHome)
	}

	var insecureRegistries string
	for _, ir := range config.InsecureRegistries {
		insecureRegistries = insecureRegistries + ir + ","
	}

	if config.SetUpPrivateRegistry {
		lines = append(lines, "private_registry_port="+strconv.Itoa(config.PrivateRegistryPort))
		lines = append(lines, "lan_registry="+config.Master1Info.Ip + ":" + strconv.Itoa(config.PrivateRegistryPort))
		insecureRegistries = insecureRegistries + config.Master1Info.Ip + ":" + strconv.Itoa(config.PrivateRegistryPort) + ","
	} else {
		lines = append(lines, "lan_registry="+config.LanRegistry)
		insecureRegistries = insecureRegistries + config.LanRegistry + ","
	}
	insecureRegistries = strings.TrimRight(insecureRegistries, ",")
	lines = append(lines, "insecure_registries=" + insecureRegistries)
	lines = append(lines, "enable_appplan=" + strconv.FormatBool(! config.DisableAppPlan))

	if len(config.NodesToJoin) > 0 {
		lines = append(lines, "[node]")
		for _, node := range config.NodesToJoin {
			s := node.Ip + " ansible_ssh_port=" + strconv.Itoa(node.SshPort) + " hostname=" + node.Hostname
			lines = append(lines, s)
		}
	}

	return

}

func haMasterHostsFileContent(config module.InstallConfig) (lines []string) {

	master1Str := config.Master1Info.Ip + " ansible_ssh_port=" + strconv.Itoa(config.Master1Info.SshPort) + " hostname=" + config.Master1Info.Hostname
	master2Str := config.Master2Info.Ip + " ansible_ssh_port=" + strconv.Itoa(config.Master2Info.SshPort) + " hostname=" + config.Master2Info.Hostname
	master3Str := config.Master3Info.Ip + " ansible_ssh_port=" + strconv.Itoa(config.Master3Info.SshPort) + " hostname=" + config.Master3Info.Hostname
	master1Password := config.CommonPassword
	if config.Master1Info.Password != "" {
		master1Password = config.Master1Info.Password
	}

	lines = []string{

		"[master]",
		master1Str,
		master2Str,
		master3Str,

		"[master1]",
		master1Str,
		"[master1:vars]",
		"master_role=master1",
		"set_private_registry=" + strconv.FormatBool(config.SetUpPrivateRegistry),

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
		"service_subnet=" + config.ServiceSubnet,
		"control_plane_endpoint=" + config.ControlPlaneEndpoint,
		"apiserver_lb=" + config.ApiserverLb,
		"apiserver_lbport=" + config.ApiserverLbport,
		"token=" + config.Token,
		"master1_ip=" + config.Master1Info.Ip,
		"master2_ip=" + config.Master2Info.Ip,
		"master3_ip=" + config.Master3Info.Ip,
		"master1_password=" + master1Password,
	}

	if len(config.DockerHome) > 0 {
		lines = append(lines, "docker_home" + config.DockerHome)
	}

	var insecureRegistries string
	for _, ir := range config.InsecureRegistries {
		insecureRegistries = insecureRegistries + ir + ","
	}

	if config.SetUpPrivateRegistry {
		lines = append(lines, "private_registry_port="+strconv.Itoa(config.PrivateRegistryPort))
		lines = append(lines, "lan_registry="+config.Master1Info.Ip + ":" + strconv.Itoa(config.PrivateRegistryPort))
		insecureRegistries = insecureRegistries + config.Master1Info.Ip + ":" + strconv.Itoa(config.PrivateRegistryPort) + ","
	} else {
		lines = append(lines, "lan_registry="+config.LanRegistry)
		insecureRegistries = insecureRegistries + config.LanRegistry + ","
	}
	insecureRegistries = strings.TrimRight(insecureRegistries, ",")
	lines = append(lines, "insecure_registries=" + insecureRegistries)
	lines = append(lines, "enable_appplan=" + strconv.FormatBool(! config.DisableAppPlan))
	if len(config.ClusterName) > 0 {
		lines = append(lines, "cluster_name=" + config.ClusterName)
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

func nodeJoinHostsFileContent(config module.InstallConfig) (lines []string) {
	lines = []string{
		"[all:vars]",
		"lan_registry=" + config.LanRegistry,
		"kubernetes_version=" + config.KubernetesVersion,
		//"kube_rpm_version=" + config.KubeRpmVersion,
		"control_plane_endpoint=" + config.ControlPlaneEndpoint,
		//"token=" + config.Token,
	}
	if config.Token != "" {
		lines = append(lines, "token="+config.Token)
	}
	if len(config.DockerHome) > 0 {
		lines = append(lines, "docker_home" + config.DockerHome)
	}
	var insecureRegistries string
	for _, ir := range config.InsecureRegistries {
		insecureRegistries = insecureRegistries + ir + ","
	}
	insecureRegistries = insecureRegistries + config.LanRegistry + ","
	insecureRegistries = strings.TrimRight(insecureRegistries, ",")
	lines = append(lines, "insecure_registries=" + insecureRegistries)
	if len(config.NodesToJoin) > 0 {
		lines = append(lines, "[node]")
		for _, node := range config.NodesToJoin {
			s := node.Ip + " ansible_ssh_port=" + strconv.Itoa(node.SshPort) + " hostname=" + node.Hostname
			lines = append(lines, s)
		}
	}
	return
}

func nodeDestroyHostsFileContent(config module.InstallConfig) (lines []string) {
	lines = []string{}
	if len(config.NodesToDestroy) > 0 {
		lines = append(lines, "[destroynode]")
		for _, node := range config.NodesToDestroy {
			s := node.Ip + " ansible_ssh_port=" + strconv.Itoa(node.SshPort)
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

func constructFilesAndCmd(config module.InstallConfig) (filesAndCmds common.TaskFilesAndCmds, err error) {
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
		//ansible 192.168.30.67 -m fetch -a "src=/etc/kubernetes/admin.conf flat=yes dest=/tmp/192.168.30.67-admin.conf"
		//filesAndCmds.KubeconfigCmdStr = "ansible " + config.ApiserverLb + "m fetch -a 'src=/etc/kubernetes/admin.conf flat=yes dest=" + common.STATUS_DIR  + config.JobId + ".kubeconfig"

	case "worker-node-join":
		filesAndCmds.CoreCmdStr = "ansible-playbook " + common.WOKER_NODE_JOIN_YAML_FILE + " -i " + filesAndCmds.HostsFile
		//nodes :=""
		for _, node := range config.NodesToJoin {
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
	case "destroy":
		filesAndCmds.CoreCmdStr = "ansible-playbook " + common.WOKER_NODE_DESTROY_YAML_FILE + " -i " + filesAndCmds.HostsFile
		for _, node := range config.NodesToDestroy {
			targeHosts = append(targeHosts, node.Ip)
		}
	default:
		err = errors.New("Job type must be one of single-master-bootstrap, worker-node-join, ha-master-bootstrap, ha-master-join, destroy.")
	}
	if config.JobType == "ha-master-bootstrap" || config.JobType == "single-master-bootstrap" {
		filesAndCmds.KubeconfigCmdStr = "ansible " + config.Master1Info.Ip + " -m fetch -a 'src=/etc/kubernetes/admin.conf flat=yes dest=" + common.STATUS_DIR + config.JobId + ".kubeconfig'" + " -i " + filesAndCmds.HostsFile
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

/*func GetJobProcess(c *gin.Context) {

	count := getJobProcess()

	c.JSON(200, common.BuildResponse(200, "", count))
}*/

func getJobProcess(jobId string) int {

	// file local-192.168.26.200.log
	var (
		f   *os.File
		err error
	)
	//if bytes, err = ioutil.ReadFile(common.LOGS_DIR + jobId + common.LOG_FILE_SUFFIX); err != nil {
	fileLocation := common.LOGS_DIR + jobId + common.LOG_FILE_SUFFIX

	f, err = os.OpenFile(fileLocation, os.O_RDWR, 0666)
	if err != nil {
		return 0
	}
	return countLines(f)
	// count
}

func countLines(file *os.File) (count int) {

	input := bufio.NewScanner(file)
	for input.Scan() {
		line := input.Text()
		if strings.HasPrefix(line, "TASK ") {
			count++
		}
	}
	return
}

func CancelJob(c *gin.Context) {
	jobId := c.Query("jobId")
	if len(jobId) == 0 {
		c.JSON(400, "Must specify a jobId.")
		return
	}
	var err error
	err = jobutil.CancelJobById(jobId)
	if err != nil {
		c.JSON(500, common.BuildResponse(500, err.Error(), nil))
		return
	}
	c.JSON(200, common.BuildResponse(200, "Job cancled.", nil))
}


