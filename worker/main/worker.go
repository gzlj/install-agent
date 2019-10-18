package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gzlj/install-agent/worker"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var (
	EtcdHost string
	MongoHost string
	MasterIp string
	LanRegistry string
	DownloadUrl string
	JobId string
	JobType string	//"node", "sing_master", "ha_master"
	done chan struct{}
	undone chan string
)

// 解析命令行参数
func initArgs() {

	flag.StringVar(&EtcdHost, "etcd-host", "127.0.0.1:2379", "etcd db")
	flag.StringVar(&MongoHost, "mongo-host", "./worker.json", "mogon db")
	flag.StringVar(&MasterIp, "master-ip", "127.0.0.1", "k8s master ip")
	flag.StringVar(&LanRegistry, "lan-registry", "127.0.0.1:5000", "lan docker image registry")
	flag.StringVar(&JobId, "job-id", "", "job id")
	flag.StringVar(&DownloadUrl, "download-url", "", "download-url")
	flag.StringVar(&JobType, "job-type", "node", "master: install master; node: install node.")
	flag.Parse()


}

// 初始化线程数量
func initEnv() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func syncLog(reader io.ReadCloser) {
	fmt.Println("start syncLog()")
	f, _ := os.OpenFile("file.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	defer f.Close()
	buf := make([]byte, 1024, 1024)
	for {
		strNum, err := reader.Read(buf)

		if err != nil {
			//读到结尾
			if err == io.EOF || strings.Contains(err.Error(), "file already closed") {
				err = nil
			}
		}
		outputByte := buf[:strNum]
		f.WriteString(string(outputByte))
		// TODO:
		// post to master process log api
	}
	fmt.Println("stop syncLog()")
	worker.G_done <- struct{}{}
}

func main() {
	var (
		err error
		status worker.Status
		cmdStr string
		t1 *time.Timer
		cmdStdoutPipe io.ReadCloser
		cmdStderrPipe io.ReadCloser
		cmd *exec.Cmd

	)

	// 初始化命令行参数
	initArgs()

	// 初始化线程
	initEnv()

	//InitConfig
	//etcdHost, masterIp, lanRegistry, downloadUrl, jobId string
	if err = worker.InitConfig(EtcdHost,MasterIp,LanRegistry, DownloadUrl, JobId  ); err != nil {
		//close
		fmt.Println("error InitConfig()")
		os.Exit(-1)
	}


	// 服务注册
	if err = worker.InitRegister(); err != nil {
		//close
		fmt.Println("error InitRegister()")
		os.Exit(-1)
	}



	//craete download script cmd
	cancelCtx, cancelFunc := context.WithCancel(context.TODO())
	downloadCmdStr := "wget -O k8s-node-join.tar.gz http://" + DownloadUrl + " ; tar xf k8s-node-join.tar.gz"
	downloadCmd := exec.CommandContext(context.TODO(), "bash", "-c", downloadCmdStr)
	err = downloadCmd.Run()
	if err != nil {
		status =worker.Status{
			Code: 500,
			Err: "failed to download installation script.",
		}
		goto FINISH
	}

	//craete installation cmd
	if JobType == "node" {
		cmdStr = " ./k8s_single_installation_centos_with_caas_online/centos_node_k8s-1.14.2.sh --master-ip "+MasterIp+ " --lan-registry " + LanRegistry
	} else if JobType == "sing_master"  {
		cmdStr = " ./k8s_single_installation_centos_with_caas_online/centos_node_k8s-1.14.2.sh --master-ip "+MasterIp+ " --lan-registry " + LanRegistry
	} else {
		cmdStr = " ./k8s_single_installation_centos_with_caas_online/centos_single_master_k8s-1.14.2.sh --master-ip "+MasterIp+ " --lan-registry " + LanRegistry
	}


	fmt.Println("cmdStr: " + cmdStr)
	worker.G_config.Cancel = cancelFunc

	cmd = exec.CommandContext(cancelCtx, "bash", "-c", cmdStr)
	//cmd := exec.CommandContext(cancelCtx, cmdStr)
	cmdStdoutPipe, _ = cmd.StdoutPipe()
	cmdStderrPipe, _ = cmd.StderrPipe()

	go syncLog(cmdStdoutPipe)
	go syncLog(cmdStderrPipe)

	err = cmd.Start()
	fmt.Println("cmd.Start() zhi hou")
	if err != nil {
		status = worker.Status{
			Code: 500,
			Err: "failed to download installation script.",
		}
		fmt.Println(err)
		goto FINISH
	}

	t1 = time.NewTimer(time.Second * 3)
	// 正常退出
	for {
		select {
			case <- worker.G_done:
				status =worker.Status{
					Code: 200,
				}
				fmt.Println(status)
				goto FINISH
			case errMsg := <- worker.G_undone:
				status =worker.Status{
					Code: 500,
					Err: errMsg,
				}
				fmt.Println(status)
				goto FINISH
		case <- t1.C:
			fmt.Println("timer in main loop****************************************")
		}
	}

FINISH:

	worker.UploadStatus(JobId, status)
	fmt.Println("install job is done.")
	return



}
