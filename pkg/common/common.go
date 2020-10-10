package common

import "strings"
import "errors"

const (
	// 任务保存目录
	JOB_SAVE_DIR = "/job/"

	// 任务强杀目录
	JOB_KILLER_DIR = "/killer/"

	// 任务锁目录
	JOB_LOCK_DIR = "/cron/lock/"

	// 服务注册目录
	JOB_WORKER_DIR = "/node/"

	JOB_STATUS_DIR = "/status/"

	JOB_LOG_DIR = "/logs/"

	WORKING_DIR = "/etc/ansible/"

	LOGS_DIR = WORKING_DIR + "/logs/"

	STATUS_DIR = WORKING_DIR + "status/"

	HOSTS_DIR = WORKING_DIR + "target-hosts/"

	STATUS_FILE_SUFFIX = ".status"
	LOG_FILE_SUFFIX = ".log"
	KUBECONFIG_FILE_SUFFIX = ".kubeconfig"
	HOSTS_FILE_SUFFIX = ".hosts"

	//ha-master-boostrap.yaml
	HA_MASTER_BOOTSTRAP_YAML_FILE = WORKING_DIR + "ha-master-boostrap.yaml"
	HA_MASTER2_JOIN_YAML_FILE = WORKING_DIR + "ha-master2-join.yaml"
	HA_MASTER3_JOIN_YAML_FILE = WORKING_DIR + "ha-master3-join.yaml"
	HA_MASTER_JOIN_YAML_FILE = WORKING_DIR + "ha-master-join.yaml"


	//single-master-bootstrap.yaml
	SINGLE_MASTER_BOOTSTRAP_YAML_FILE = WORKING_DIR + "/single-master-bootstrap.yaml"
	//worker-node-join.yaml
	WOKER_NODE_JOIN_YAML_FILE = WORKING_DIR + "/worker-node-join.yaml"

	WOKER_NODE_DESTROY_YAML_FILE = WORKING_DIR + "/destroy-node.yaml"

	SINGEL_MASTER_JOB_TASK_COUNT = 45
	HA_MASTER_JOB_TASK_COUNT = 85
	WORKER_NODE_JOIN_JOB_TASK_COUNT = 40

	WORKER_NODE_DESTROY_JOB_TASK_COUNT = 3
)

var (
	ERR_NO_LOCAL_IP_FOUND = errors.New("没有找到网卡IP")

	UPLOAD_STATUS_FAIL = errors.New("upload job status fail.")
	UPLOAD_LOG_FAIL    = errors.New("upload job log fail.")
)

func ExtractKillerName(killerKey string) string {
	return strings.TrimPrefix(killerKey, JOB_KILLER_DIR)
}

type Status struct {
	Err   string `json:"err"`
	Code  int64  `json:"code"`
	Phase string `json:"phase"` //created running stopping stoped exiting exited failed
	Id    string `json:"id"`
	JobType string `json:"jobType"`
	Progress int `json:"progress"`  //100 means 100%
}

var (
	G_done   chan struct{} = make(chan struct{})
	G_undone chan string   = make(chan string)
)

type TaskFilesAndCmds struct {
	TargetHosts []string
	HostsFile string
	Logfile string
	//SshCmdStr string
	CoreCmdStr string
	//OtherCmdStrs []string
	KubeconfigCmdStr string
}
