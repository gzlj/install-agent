package worker

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

	// 保存任务事件
	JOB_EVENT_SAVE = 1

	// 删除任务事件
	JOB_EVENT_DELETE = 2

	// 强杀任务事件
	JOB_EVENT_KILL = 3


)

var (
	ERR_NO_LOCAL_IP_FOUND = errors.New("没有找到网卡IP")

	UPLOAD_STATUS_FAIL = errors.New("upload job status fail.")
)

func ExtractKillerName(killerKey string) (string) {
	return strings.TrimPrefix(killerKey, JOB_KILLER_DIR)
}
